//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

// S2.3 deterministic matrix. Every repository is a fresh temporary local Git
// repository with no remote; agent attempts are in-process stubs that edit
// and commit in their own isolated worktree; only post-merge checks run as
// real processes through the observed process boundary.

const (
	integrationTarget   = "refs/heads/axiom/integration"
	integrationIdentity = "github.com/example/app"
	secondExecutionID   = "00000000-0000-4000-8000-0000000000e2"
)

var gitTestEnv = []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C",
	"GIT_AUTHOR_NAME=fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid"}

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitE(dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func gitE(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH")}, gitTestEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// edit is one agent's work in its own worktree; nil means the agent fails.
type edit func(t *testing.T, dir string)

func commitFile(name, content string) edit {
	return func(t *testing.T, dir string) {
		write(t, filepath.Join(dir, name), content)
		gitT(t, dir, "add", "-A")
		gitT(t, dir, "commit", "-q", "-m", "agent edits "+name)
	}
}

type integrationFixture struct {
	t                 *testing.T
	repo, root, base  string
	git               *gitworkspace.Integrator
	engine            Engine
	store             local.ExecutionStore
	edits             map[string]edit
	barrier           *sync.WaitGroup
	active, maxActive atomic.Int32
	checks            []workflow.CommandCheck
	gate              bool
	lastCheck         atomic.Pointer[executiongraph.ProcessCorrelation]
}

// gitRunner observes real check processes and runs agent stubs in-process.
type gitRunner struct{ f *integrationFixture }

func (r gitRunner) RunObserved(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	if inv.Argv[0] == commandGateShell {
		return (executiongraph.OSProcessRunner{}).RunObserved(ctx, inv, func(p *executiongraph.ProcessCorrelation) error {
			r.f.lastCheck.Store(p)
			return started(p)
		})
	}
	f, agent := r.f, inv.Argv[1]
	if err := started(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
		return executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true, Stopped: true}
	}
	now := f.active.Add(1)
	for old := f.maxActive.Load(); now > old && !f.maxActive.CompareAndSwap(old, now); old = f.maxActive.Load() {
	}
	defer f.active.Add(-1)
	if f.barrier != nil {
		f.barrier.Done()
		done := make(chan struct{})
		go func() { f.barrier.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			return executiongraph.ProcessResult{ExitCode: 1}
		}
	}
	work := f.edits[agent]
	if work == nil {
		return executiongraph.ProcessResult{ExitCode: 1}
	}
	work(f.t, filepath.Join(f.root, "agent-"+agent))
	return executiongraph.ProcessResult{ExitCode: 0}
}

// gitLauncher maps an attempt to its agent stub and records the stage
// checks of the attempt itself as observed fake processes.
type gitLauncher struct{}

func (gitLauncher) Build(_ context.Context, r LaunchRequest) (executiongraph.Invocation, error) {
	return executiongraph.Invocation{RuntimeID: "stub", Argv: []string{"agent", r.Ref.AgentID}, OutputMax: 4096}, nil
}

func (gitLauncher) Collect(_ context.Context, r LaunchRequest, _ executiongraph.ProcessResult) (Outcome, error) {
	var out Outcome
	for _, c := range r.Stage.Commands {
		if err := r.PrepareCommand(c.ID); err != nil {
			return out, err
		}
		if err := r.StartCommand(c.ID, workflow.ProcessIdentity{PID: 778, StartIdentity: "boot:1", GroupIdentity: "778"}); err != nil {
			return out, err
		}
		record, err := r.FinishCommand(c.ID, 0)
		if err != nil {
			return out, err
		}
		out.Commands = append(out.Commands, record)
	}
	return out, nil
}

// checkCommand uses the symlink-free shell path: check executables, like
// stage commands, must not traverse a symlink (/bin is one on some hosts).
func checkCommand(script string, timeout int64) []workflow.CommandCheck {
	shell, err := filepath.EvalSymlinks("/bin/sh")
	if err != nil {
		shell = "/bin/sh"
	}
	return []workflow.CommandCheck{{ID: "post-merge", Argv: []string{shell, "-c", script}, TimeoutSeconds: timeout, Effects: []string{"validation_command"}}}
}

// realTempDir resolves t.TempDir: launcher and Git roots refuse symlinked
// components, and darwin temp dirs live under the /var symlink.
func realTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func newIntegrationFixture(t *testing.T, edits map[string]edit) *integrationFixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	if _, err := gitE(t.TempDir(), "merge-tree", "-h"); err != nil && !strings.Contains(err.Error(), "write-tree") {
		t.Skip("git merge-tree --write-tree unavailable")
	}
	tmp := realTempDir(t)
	repo := filepath.Join(tmp, "repo")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	gitT(t, repo, "init", "-q", "-b", "main")
	for _, name := range []string{"a.txt", "b.txt", "shared.txt"} {
		write(t, filepath.Join(repo, name), name+" base\n")
	}
	gitT(t, repo, "add", "-A")
	gitT(t, repo, "commit", "-q", "-m", "base")
	base := gitT(t, repo, "rev-parse", "HEAD")
	gitT(t, repo, "branch", "axiom/integration", base)
	// Unrelated user state that must survive every scenario untouched.
	gitT(t, repo, "branch", "user/keep", base)
	gitT(t, repo, "worktree", "add", "-q", filepath.Join(tmp, "user-worktree"), "user/keep")
	write(t, filepath.Join(tmp, "user-worktree", "user-dirty.txt"), "user work in progress\n")
	root := filepath.Join(tmp, "worktrees")
	g, err := gitworkspace.NewIntegrator(context.Background(), repo, root)
	if err != nil {
		t.Fatal(err)
	}
	storeRoot := filepath.Join(tmp, "state")
	if err := os.Mkdir(storeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := local.NewExecutionStore(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	f := &integrationFixture{t: t, repo: repo, root: root, base: base, git: g, store: store, edits: edits, checks: checkCommand("exit 0", 10)}
	var n atomic.Int32
	f.engine = Engine{
		Store: store, Runner: gitRunner{f}, Launcher: gitLauncher{},
		Resolve: func(_ context.Context, x workflow.Execution, ref workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
			return stubInvocation(), workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"workspace_write", "validation_command"}, Actor: "user:local"}, nil
		},
		NewID:      func() (string, error) { return fmt.Sprintf("att-%d", n.Add(1)), nil },
		CancelPoll: 10 * time.Millisecond,
	}
	t.Cleanup(func() { assertNoChildProcesses(t) })
	return f
}

// start prepares one isolated worktree/branch per agent at base, then runs
// the agents of one Execution through the shared engine.
func (f *integrationFixture) start(executionID string, agents ...string) workflow.Execution {
	t := f.t
	t.Helper()
	var specs []workflow.AgentSpec
	for _, a := range agents {
		if _, err := f.git.PrepareAgentWorktree(context.Background(), "agent-"+a, "refs/heads/axiom/agent-"+a, f.base); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, workflow.AgentSpec{ID: a, Effects: []string{"workspace_write", "validation_command"}})
	}
	x, err := workflow.NewExecution(workflow.NewExecutionInput{
		ExecutionID: executionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"},
		Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: integrationIdentity}}, WorkflowID: "parallel", Revision: "r1", Now: time.Now(),
		Stages: []workflow.StageSpec{{ID: "work", Agents: specs, Commands: f.checks, HumanGate: f.gate, TimeoutSeconds: 30}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.Create(context.Background(), x); err != nil {
		t.Fatal(err)
	}
	// Failing agents are part of some scenarios; the stored state is the fact.
	if _, err := f.engine.Run(context.Background(), "proj", executionID, RunOptions{}); err != nil {
		t.Logf("run %s: %v", executionID, err)
	}
	return f.load(executionID)
}

func attemptFor(t *testing.T, x workflow.Execution, agent string) workflow.Attempt {
	t.Helper()
	for _, s := range x.Stages {
		for _, task := range s.Tasks {
			if task.AgentID == agent && len(task.Attempts) != 0 {
				return task.Attempts[len(task.Attempts)-1]
			}
		}
	}
	t.Fatalf("no attempt for %s", agent)
	return workflow.Attempt{}
}

func (f *integrationFixture) request(x workflow.Execution, agent, id, expected string) IntegrationRequest {
	return IntegrationRequest{
		IntegrationID: id, SourceAttemptID: attemptFor(f.t, x, agent).AttemptID,
		RepositoryKey: "app", RepositoryIdentity: integrationIdentity,
		Worktree: "agent-" + agent, SourceBranch: "refs/heads/axiom/agent-" + agent,
		BaseSHA: f.base, TargetRef: integrationTarget, ExpectedTargetHead: expected,
	}
}

func (f *integrationFixture) integrate(executionID string, git IntegrationGit, req IntegrationRequest) (workflow.Execution, error) {
	return f.engine.Integrate(context.Background(), "proj", executionID, git, req)
}

func (f *integrationFixture) target() string {
	return gitT(f.t, f.repo, "rev-parse", integrationTarget)
}

func (f *integrationFixture) load(id string) workflow.Execution {
	x, err := f.store.Load(context.Background(), "proj", id)
	if err != nil {
		f.t.Fatal(err)
	}
	return x
}

// snapshot is every ref and worktree; scenarios may add, never remove.
type gitSnapshot struct{ refs, worktrees map[string]string }

func (f *integrationFixture) snapshot() gitSnapshot {
	s := gitSnapshot{refs: map[string]string{}, worktrees: map[string]string{}}
	for _, line := range strings.Split(gitT(f.t, f.repo, "for-each-ref", "--format=%(refname) %(objectname)"), "\n") {
		if name, sha, ok := strings.Cut(line, " "); ok {
			s.refs[name] = sha
		}
	}
	for _, line := range strings.Split(gitT(f.t, f.repo, "worktree", "list", "--porcelain"), "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			s.worktrees[path] = path
		}
	}
	return s
}

// assertPreserved checks that nothing was removed or moved except the
// principal's target, that no remote exists and no remote ref was created,
// and that the unrelated dirty user worktree is untouched.
func (f *integrationFixture) assertPreserved(before gitSnapshot, targetMayMove bool) {
	t := f.t
	t.Helper()
	after := f.snapshot()
	for name, sha := range before.refs {
		if after.refs[name] == "" {
			t.Fatalf("ref %s was removed", name)
		}
		if after.refs[name] != sha && !(targetMayMove && name == integrationTarget) {
			t.Fatalf("ref %s moved %s -> %s", name, sha, after.refs[name])
		}
	}
	for path := range before.worktrees {
		if after.worktrees[path] == "" {
			t.Fatalf("worktree %s was removed", path)
		}
	}
	for name := range after.refs {
		if strings.HasPrefix(name, "refs/remotes/") {
			t.Fatalf("remote ref %s created", name)
		}
	}
	if remotes := gitT(t, f.repo, "remote"); remotes != "" {
		t.Fatalf("remote configured: %q", remotes)
	}
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(f.repo), "user-worktree", "user-dirty.txt")); err != nil || string(b) != "user work in progress\n" {
		t.Fatalf("user worktree changed: %q %v", b, err)
	}
}

func integrationOf(t *testing.T, x workflow.Execution, id string) workflow.Integration {
	t.Helper()
	for _, i := range x.Integrations {
		if i.IntegrationID == id {
			return i
		}
	}
	t.Fatalf("no integration %s", id)
	return workflow.Integration{}
}

func parents(t *testing.T, repo, commit string) []string {
	return strings.Fields(gitT(t, repo, "rev-list", "--parents", "-n", "1", commit))[1:]
}

func TestIntegrateParallelWorktreesSequentiallyWithCheckEvidence(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "from a\n"), "b": commitFile("b-new.txt", "from b\n")})
	f.checks = checkCommand("test -f a.txt && test -f b.txt && test -f shared.txt", 10)
	f.barrier = &sync.WaitGroup{}
	f.barrier.Add(2)
	x := f.start(engineExecutionID, "a", "b")
	if f.maxActive.Load() != 2 {
		t.Fatalf("agents did not edit concurrently: max active %d", f.maxActive.Load())
	}
	before := f.snapshot()
	candA, candB := gitT(t, f.repo, "rev-parse", "axiom/agent-a"), gitT(t, f.repo, "rev-parse", "axiom/agent-b")
	if candA == candB || parents(t, f.repo, candA)[0] != f.base || parents(t, f.repo, candB)[0] != f.base {
		t.Fatal("agent branches do not carry distinct work from the same base")
	}

	x, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base))
	if err != nil {
		t.Fatal(err)
	}
	a := integrationOf(t, x, "int-a")
	if a.Status != workflow.IntegrationIntegrated || a.CandidateSHA != candA || a.BaseSHA != f.base || a.ExpectedTargetHead != f.base || a.FinalSHA != f.target() || len(a.Checks) != 1 || a.Checks[0].Status != workflow.CheckPassed {
		t.Fatalf("first integration record: %+v", a)
	}
	x, err = f.integrate(engineExecutionID, f.git, f.request(x, "b", "int-b", a.FinalSHA))
	if err != nil {
		t.Fatal(err)
	}
	b := integrationOf(t, x, "int-b")
	if b.Status != workflow.IntegrationIntegrated || b.CandidateSHA != candB || b.ExpectedTargetHead != a.FinalSHA || b.FinalSHA != f.target() || b.ObservedTargetHead != b.FinalSHA {
		t.Fatalf("second integration record: %+v", b)
	}
	if p := parents(t, f.repo, b.FinalSHA); len(p) != 2 || p[0] != a.FinalSHA || p[1] != candB {
		t.Fatalf("final lineage %v", p)
	}
	if p := parents(t, f.repo, a.FinalSHA); len(p) != 2 || p[0] != f.base || p[1] != candA {
		t.Fatalf("first lineage %v", p)
	}
	if gitT(t, f.repo, "rev-parse", "main") != f.base || gitT(t, f.repo, "rev-parse", "axiom/agent-a") != candA || gitT(t, f.repo, "rev-parse", "axiom/agent-b") != candB {
		t.Fatal("base or agent branches moved")
	}
	for _, name := range []string{"a-new.txt", "b-new.txt"} {
		gitT(t, f.repo, "cat-file", "-e", b.FinalSHA+":"+name)
	}
	for _, id := range []string{"int-a", "int-b"} {
		if _, err := os.Stat(filepath.Join(f.root, "integration-"+id)); err != nil {
			t.Fatalf("check worktree %s not retained: %v", id, err)
		}
	}
	f.assertPreserved(before, true)
	// Replaying an integrated candidate is refused, never a second effect.
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a2", b.FinalSHA)); !errors.Is(err, ErrIntegrationRefused) || f.target() != b.FinalSHA {
		t.Fatalf("replayed candidate: %v", err)
	}
}

func TestIntegrateSameFileConflictIsExplicit(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("shared.txt", "a wins\n"), "b": commitFile("shared.txt", "b wins\n")})
	x := f.start(engineExecutionID, "a", "b")
	x, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base))
	if err != nil {
		t.Fatal(err)
	}
	head := f.target()
	before := f.snapshot()
	x, err = f.integrate(engineExecutionID, f.git, f.request(x, "b", "int-b", head))
	if !errors.Is(err, ErrNotIntegrated) {
		t.Fatalf("conflict: %v", err)
	}
	b := integrationOf(t, x, "int-b")
	if b.Status != workflow.IntegrationNotIntegrated || b.Diagnostic != "merge_conflict" || b.MergeSHA != "" || b.FinalSHA != "" || b.ObservedTargetHead != head || f.target() != head {
		t.Fatalf("conflict record: %+v", b)
	}
	if _, err := os.Stat(filepath.Join(f.root, "integration-int-b")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("conflict produced a check worktree")
	}
	f.assertPreserved(before, false)
}

func TestIntegrateStaleBaseAndDirectTargetDriftRefuse(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n"), "b": commitFile("b-new.txt", "b\n")})
	x := f.start(engineExecutionID, "a", "b")
	x, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base))
	if err != nil {
		t.Fatal(err)
	}
	accepted := f.target()
	before := f.snapshot()
	revision := f.load(engineExecutionID).Integrations
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "b", "int-b", f.base)); !errors.Is(err, ErrIntegrationRefused) || !errors.Is(err, workflow.ErrIntegrationStale) {
		t.Fatalf("stale expected head: %v", err)
	}
	f.assertPreserved(before, false)
	// A child shares the repository refs and could move the target itself;
	// the principal observes the drift and neither adopts nor records it.
	gitT(t, f.repo, "update-ref", integrationTarget, gitT(t, f.repo, "rev-parse", "axiom/agent-b"), accepted)
	drifted := f.target()
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "b", "int-b", accepted)); !errors.Is(err, workflow.ErrIntegrationStale) {
		t.Fatalf("drifted target: %v", err)
	}
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "b", "int-b", drifted)); !errors.Is(err, workflow.ErrIntegrationStale) {
		t.Fatalf("drifted head adopted against the recorded lineage: %v", err)
	}
	if got := f.load(engineExecutionID); len(got.Integrations) != len(revision) || got.AcceptedTargetHead("app", integrationTarget) != accepted || f.target() != drifted {
		t.Fatal("drift changed recorded integration state")
	}
}

func TestIntegrateDirtyCandidateAndCheckedOutDestinationRefuse(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n"), "b": commitFile("b-new.txt", "b\n")})
	x := f.start(engineExecutionID, "a", "b")
	before := f.snapshot()
	agentA := filepath.Join(f.root, "agent-a")
	write(t, filepath.Join(agentA, "a.txt"), "uncommitted tracked change\n")
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base)); !errors.Is(err, gitworkspace.ErrForeignMutation) {
		t.Fatalf("dirty tracked candidate: %v", err)
	}
	gitT(t, agentA, "checkout", "--", "a.txt")
	write(t, filepath.Join(agentA, "untracked.txt"), "untracked\n")
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base)); !errors.Is(err, gitworkspace.ErrForeignMutation) {
		t.Fatalf("untracked candidate: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(agentA, "untracked.txt")); string(b) != "untracked\n" {
		t.Fatal("dirty agent state was not preserved")
	}
	// A destination checked out anywhere (here with uncommitted work) is
	// never moved underneath that working tree.
	dest := filepath.Join(filepath.Dir(f.repo), "dest")
	gitT(t, f.repo, "worktree", "add", "-q", dest, "axiom/integration")
	write(t, filepath.Join(dest, "dest-dirty.txt"), "destination work\n")
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "b", "int-b", f.base)); !errors.Is(err, workflow.ErrIntegrationTargetCheckedOut) {
		t.Fatalf("checked-out destination: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "dest-dirty.txt")); string(b) != "destination work\n" {
		t.Fatal("destination work was not preserved")
	}
	if got := f.load(engineExecutionID); len(got.Integrations) != 0 || f.target() != f.base {
		t.Fatal("refusals recorded or moved anything")
	}
	f.assertPreserved(before, false)
}

func TestIntegrateRepositoryBindingAndUnauthorizedSources(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n"), "b": nil})
	x := f.start(engineExecutionID, "a", "b")
	gitT(t, filepath.Join(f.root, "agent-b"), "commit", "-q", "--allow-empty", "-m", "b leaves a commit despite failing")
	before := f.snapshot()
	wrong := f.request(x, "a", "int-a", f.base)
	wrong.RepositoryIdentity = "github.com/example/other"
	if _, err := f.integrate(engineExecutionID, f.git, wrong); !errors.Is(err, ErrIntegrationRefused) || !errors.Is(err, workflow.ErrTransition) {
		t.Fatalf("changed repository identity: %v", err)
	}
	// A failed agent attempt has no validated result to integrate.
	failed := f.request(x, "b", "int-b", f.base)
	if _, err := f.integrate(engineExecutionID, f.git, failed); !errors.Is(err, workflow.ErrTransition) {
		t.Fatalf("failed attempt integrated: %v", err)
	}
	// A worktree of another repository placed under the authorized root is
	// refused by common-directory binding.
	other := filepath.Join(filepath.Dir(f.repo), "other")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	gitT(t, other, "init", "-q", "-b", "main")
	gitT(t, other, "commit", "-q", "--allow-empty", "-m", "other base")
	gitT(t, other, "worktree", "add", "-q", "-b", "axiom/agent-x", filepath.Join(f.root, "agent-x"))
	foreign := f.request(x, "a", "int-x", f.base)
	foreign.Worktree, foreign.SourceBranch = "agent-x", "refs/heads/axiom/agent-x"
	if _, err := f.integrate(engineExecutionID, f.git, foreign); !errors.Is(err, gitworkspace.ErrWorkspaceInvalid) {
		t.Fatalf("foreign repository worktree: %v", err)
	}
	// The refusals poisoned nothing: the authorized source still integrates.
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base)); err != nil {
		t.Fatal(err)
	}
	if got := f.load(engineExecutionID); len(got.Integrations) != 1 {
		t.Fatalf("refusals recorded state: %d", len(got.Integrations))
	}
	f.assertPreserved(before, true)
}

func TestIntegrateInvalidCandidateAndEscapedWorktree(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": func(*testing.T, string) {}, "b": commitFile("b-new.txt", "b\n")})
	x := f.start(engineExecutionID, "a", "b")
	before := f.snapshot()
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base)); !errors.Is(err, gitworkspace.ErrNoResult) {
		t.Fatalf("branch without a candidate commit: %v", err)
	}
	cases := map[string]func(*IntegrationRequest){
		"invalid base":         func(r *IntegrationRequest) { r.BaseSHA = "zzzz" },
		"abbreviated base":     func(r *IntegrationRequest) { r.BaseSHA = f.base[:12] },
		"unrelated base":       func(r *IntegrationRequest) { r.BaseSHA = strings.Repeat("1", 40) },
		"escaping worktree":    func(r *IntegrationRequest) { r.Worktree = "../repo" },
		"absolute worktree":    func(r *IntegrationRequest) { r.Worktree = f.repo },
		"branch mismatch":      func(r *IntegrationRequest) { r.SourceBranch = "refs/heads/axiom/agent-a" },
		"invalid source ref":   func(r *IntegrationRequest) { r.SourceBranch = "refs/heads/axiom/../main" },
		"invalid target ref":   func(r *IntegrationRequest) { r.TargetRef = "refs/heads/main.lock" },
		"missing target ref":   func(r *IntegrationRequest) { r.TargetRef = "refs/heads/axiom/absent" },
		"symlinked worktree":   func(r *IntegrationRequest) { r.Worktree = "agent-link" },
		"missing attempt":      func(r *IntegrationRequest) { r.SourceAttemptID = "att-404" },
		"unbound repository":   func(r *IntegrationRequest) { r.RepositoryKey = "lib" },
		"invalid expected sha": func(r *IntegrationRequest) { r.ExpectedTargetHead = "" },
	}
	if err := os.Symlink(filepath.Join(f.root, "agent-b"), filepath.Join(f.root, "agent-link")); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range cases {
		req := f.request(x, "b", "int-b", f.base)
		mutate(&req)
		if _, err := f.integrate(engineExecutionID, f.git, req); !errors.Is(err, ErrIntegrationRefused) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if got := f.load(engineExecutionID); len(got.Integrations) != 0 || f.target() != f.base {
		t.Fatal("invalid requests recorded or moved anything")
	}
	f.assertPreserved(before, false)
}

func TestIntegrateFailedPostMergeCheckStaysNotIntegrated(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n"), "b": commitFile("b-new.txt", "b\n")})
	// Each change passes alone; together they break the semantic check.
	f.checks = checkCommand("! { test -f a-new.txt && test -f b-new.txt; }", 10)
	x := f.start(engineExecutionID, "a", "b")
	x, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base))
	if err != nil {
		t.Fatal(err)
	}
	head := f.target()
	before := f.snapshot()
	x, err = f.integrate(engineExecutionID, f.git, f.request(x, "b", "int-b", head))
	if !errors.Is(err, ErrNotIntegrated) {
		t.Fatalf("failed check: %v", err)
	}
	b := integrationOf(t, x, "int-b")
	if b.Status != workflow.IntegrationNotIntegrated || b.Diagnostic != "post_merge_check_failed" || b.FinalSHA != "" || b.MergeSHA == "" || len(b.Checks) != 1 || b.Checks[0].Status != workflow.CheckFailed || b.Checks[0].ExitCode != 1 {
		t.Fatalf("failed check record: %+v", b)
	}
	if f.target() != head || strings.Contains(b.Diagnostic, "rollback") {
		t.Fatal("failed check moved the target or claimed a rollback")
	}
	// The intermediate merge stays inspectable in the retained check
	// worktree; it is not reachable from the target.
	if gitT(t, f.repo, "cat-file", "-t", b.MergeSHA) != "commit" || gitT(t, filepath.Join(f.root, "integration-int-b"), "rev-parse", "HEAD") != b.MergeSHA {
		t.Fatal("intermediate merge result not preserved")
	}
	if _, err := gitE(f.repo, "merge-base", "--is-ancestor", b.MergeSHA, integrationTarget); err == nil {
		t.Fatal("failed merge reachable from the target")
	}
	if x.Status == workflow.RunRecoveryRequired {
		t.Fatal("a known failed check is not an unknown effect")
	}
	f.assertPreserved(before, false)
}

// barrierGit holds every integration at Advance until all have arrived, so
// the shared destination is contended for real.
type barrierGit struct {
	IntegrationGit
	arrive *sync.WaitGroup
}

func (b barrierGit) Advance(ctx context.Context, ref, expected, merge string) (string, error) {
	b.arrive.Done()
	b.arrive.Wait()
	return b.IntegrationGit.Advance(ctx, ref, expected, merge)
}

func TestIntegrateConcurrentRaceOnSharedDestination(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n"), "b": commitFile("b-new.txt", "b\n")})
	x1 := f.start(engineExecutionID, "a")
	x2 := f.start(secondExecutionID, "b")
	before := f.snapshot()
	// Two principals (two Executions, two Integrator instances) race for one
	// destination; only the target compare-and-swap decides.
	second, err := gitworkspace.NewIntegrator(context.Background(), f.repo, f.root)
	if err != nil {
		t.Fatal(err)
	}
	arrive := &sync.WaitGroup{}
	arrive.Add(2)
	type result struct {
		x   workflow.Execution
		err error
		id  string
	}
	results := make(chan result, 2)
	go func() {
		x, err := f.integrate(engineExecutionID, barrierGit{f.git, arrive}, f.request(x1, "a", "int-a", f.base))
		results <- result{x, err, "int-a"}
	}()
	go func() {
		x, err := f.integrate(secondExecutionID, barrierGit{second, arrive}, f.request(x2, "b", "int-b", f.base))
		results <- result{x, err, "int-b"}
	}()
	var won, lost []result
	for range 2 {
		r := <-results
		if r.err == nil {
			won = append(won, r)
		} else {
			lost = append(lost, r)
		}
	}
	if len(won) != 1 || len(lost) != 1 || !errors.Is(lost[0].err, ErrNotIntegrated) {
		t.Fatalf("race outcome won=%d lost=%v", len(won), lost)
	}
	winner, loser := integrationOf(t, won[0].x, won[0].id), integrationOf(t, lost[0].x, lost[0].id)
	if f.target() != winner.FinalSHA || loser.Diagnostic != "target_moved_concurrently" || loser.FinalSHA != "" || loser.ObservedTargetHead != winner.FinalSHA {
		t.Fatalf("race records winner=%+v loser=%+v", winner, loser)
	}
	if _, err := gitE(f.repo, "merge-base", "--is-ancestor", loser.MergeSHA, integrationTarget); err == nil {
		t.Fatal("loser's merge reachable from target")
	}
	f.assertPreserved(before, true)
}

// faultGit injects Git boundary outcomes the real adapter cannot produce on
// demand.
type faultGit struct {
	IntegrationGit
	advances     atomic.Int32
	advance      func() (string, error)
	beforeAdvice func()
}

func (g *faultGit) ObserveCheckWorktree(ctx context.Context, path, merge string) error {
	if g.beforeAdvice != nil {
		g.beforeAdvice()
	}
	return g.IntegrationGit.ObserveCheckWorktree(ctx, path, merge)
}

func (g *faultGit) Advance(ctx context.Context, ref, expected, merge string) (string, error) {
	g.advances.Add(1)
	if g.advance != nil {
		return g.advance()
	}
	return g.IntegrationGit.Advance(ctx, ref, expected, merge)
}

func TestIntegrateUnknownGitOutcomeIsNeverRetried(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n"), "b": commitFile("b-new.txt", "b\n")})
	x := f.start(engineExecutionID, "a", "b")
	before := f.snapshot()
	g := &faultGit{IntegrationGit: f.git, advance: func() (string, error) { return "", errors.New("git update-ref killed; ref unreadable") }}
	x, err := f.integrate(engineExecutionID, g, f.request(x, "a", "int-a", f.base))
	if !errors.Is(err, ErrIntegrationUnknown) {
		t.Fatalf("unknown outcome: %v", err)
	}
	a := integrationOf(t, x, "int-a")
	if a.Status != workflow.IntegrationUnknown || a.Diagnostic != "advance_outcome_unknown" || a.FinalSHA != "" || x.Status != workflow.RunRecoveryRequired {
		t.Fatalf("unknown record: %+v status %s", a, x.Status)
	}
	if _, err := f.integrate(engineExecutionID, g, f.request(x, "b", "int-b", f.base)); !errors.Is(err, workflow.ErrUnknownOutstanding) {
		t.Fatalf("integration after unknown outcome: %v", err)
	}
	if g.advances.Load() != 1 || f.target() != f.base {
		t.Fatalf("unknown effect retried: %d advances", g.advances.Load())
	}
	// An advance that reports failure while the target shows our merge is
	// ambiguous and is not declared integrated either.
	f.assertPreserved(before, false)

	g2 := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n")})
	x2 := g2.start(engineExecutionID, "a")
	ambiguous := &faultGit{IntegrationGit: ambiguousAdvance{g2.git}}
	x2, err = g2.integrate(engineExecutionID, ambiguous, g2.request(x2, "a", "int-a", g2.base))
	if i := integrationOf(t, x2, "int-a"); !errors.Is(err, ErrIntegrationUnknown) || i.Diagnostic != "advance_outcome_ambiguous" || i.FinalSHA != "" || ambiguous.advances.Load() != 1 {
		t.Fatalf("ambiguous advance: %v %+v", err, i)
	}
}

// ambiguousAdvance performs the real compare-and-swap and then reports an
// error, as a process killed after the ref write would.
type ambiguousAdvance struct{ *gitworkspace.Integrator }

func (a ambiguousAdvance) Advance(ctx context.Context, ref, expected, merge string) (string, error) {
	observed, err := a.Integrator.Advance(ctx, ref, expected, merge)
	if err != nil {
		return observed, err
	}
	return observed, errors.New("lost response after update-ref")
}

// lostFinishStore drops the outcome commit, as a runner crash after the Git
// effect would.
type lostFinishStore struct {
	local.ExecutionStore
	drop atomic.Bool
}

func (s *lostFinishStore) UpdateFenced(ctx context.Context, projectID, executionID string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	if current, err := s.ExecutionStore.Load(ctx, projectID, executionID); err == nil && s.drop.Load() {
		for _, i := range current.Integrations {
			if i.Status == workflow.IntegrationPrepared {
				return current, errors.New("runner crashed before recording the outcome")
			}
		}
	}
	return s.ExecutionStore.UpdateFenced(ctx, projectID, executionID, fence, mutate)
}

func TestIntegrateLostOutcomeBecomesUnknownForNextRunner(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n")})
	x := f.start(engineExecutionID, "a")
	store := &lostFinishStore{ExecutionStore: f.store}
	store.drop.Store(true)
	f.engine.Store = store
	if _, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base)); err == nil {
		t.Fatal("lost outcome reported success")
	}
	moved := f.target()
	if moved == f.base {
		t.Fatal("fixture: the Git effect should have happened before the lost commit")
	}
	if i := integrationOf(t, f.load(engineExecutionID), "int-a"); i.Status != workflow.IntegrationPrepared {
		t.Fatalf("intent not retained: %s", i.Status)
	}
	store.drop.Store(false)
	// The next runner cannot correlate the effect: unknown, never re-run.
	_, _, err := f.engine.Reconcile(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	after := f.load(engineExecutionID)
	if i := integrationOf(t, after, "int-a"); i.Status != workflow.IntegrationUnknown || i.Diagnostic != "runner_lost_before_integration_outcome" || after.Status != workflow.RunRecoveryRequired {
		t.Fatalf("orphaned integration: %+v", i)
	}
	if f.target() != moved {
		t.Fatal("reconcile moved the target")
	}
}

func TestIntegrateHumanGateAndCancelProhibitEffect(t *testing.T) {
	gated := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n")})
	gated.gate = true
	x := gated.start(engineExecutionID, "a")
	if _, err := gated.integrate(engineExecutionID, gated.git, gated.request(x, "a", "int-a", gated.base)); !errors.Is(err, workflow.ErrTransition) || gated.target() != gated.base {
		t.Fatalf("pending human gate: %v", err)
	}

	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n")})
	x = f.start(engineExecutionID, "a")
	before := f.snapshot()
	g := &faultGit{IntegrationGit: f.git, beforeAdvice: func() {
		if _, err := f.engine.Cancel(context.Background(), "proj", engineExecutionID, "human:operator"); err != nil {
			t.Error(err)
		}
	}}
	x, err := f.integrate(engineExecutionID, g, f.request(x, "a", "int-a", f.base))
	if !errors.Is(err, ErrNotIntegrated) || integrationOf(t, x, "int-a").Diagnostic != "cancelled_before_effect" || g.advances.Load() != 0 || f.target() != f.base {
		t.Fatalf("cancel before effect: %v", err)
	}
	f.assertPreserved(before, false)
}

func TestIntegrateCheckTimeoutStopsOwnedProcess(t *testing.T) {
	f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a-new.txt", "a\n")})
	f.checks = checkCommand("sleep 30", 1)
	x := f.start(engineExecutionID, "a")
	started := time.Now()
	x, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base))
	if !errors.Is(err, ErrNotIntegrated) || time.Since(started) > 20*time.Second {
		t.Fatalf("timed out check: %v after %s", err, time.Since(started))
	}
	a := integrationOf(t, x, "int-a")
	if a.Diagnostic != "post_merge_check_terminated" || a.Checks[0].Status != workflow.CheckTerminated || f.target() != f.base {
		t.Fatalf("timeout record: %+v", a)
	}
	p := f.lastCheck.Load()
	if p == nil {
		t.Fatal("check identity not observed")
	}
	if current, err := executiongraph.ObserveRecordedProcess(*p); err != nil || current != nil && current.StartIdentity == p.StartIdentity {
		t.Fatalf("timed-out check process survives: %+v %v", current, err)
	}
}

// S23-CR-01: an unconfirmed check effect must block another integration.
func TestIntegrateUnknownCheckBlocksReplay(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(fmt.Sprint(ambiguous), func(t *testing.T) {
			f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a.txt", "changed\n")})
			x := f.start(engineExecutionID, "a")
			r := &fakeRunner{run: func(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
				return executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true, EffectAmbiguous: ambiguous}
			}}
			f.engine.Runner = r
			g := &faultGit{IntegrationGit: f.git}
			got, err := f.integrate(engineExecutionID, g, f.request(x, "a", "unknown-check", f.base))
			firstErr := err
			before := f.load(engineExecutionID)
			_, err = f.integrate(engineExecutionID, g, f.request(x, "a", "second-check", f.base))
			if !errors.Is(firstErr, ErrIntegrationUnknown) || got.Status != workflow.RunRecoveryRequired || err == nil || r.spawns.Load() != 1 || g.advances.Load() != 0 || f.target() != f.base {
				t.Fatalf("unknown check replay: first status=%s first err=%v second err=%v calls=%d", got.Status, firstErr, err, r.spawns.Load())
			}
			after := f.load(engineExecutionID)
			if len(after.Integrations) != 1 || after.Integrations[0].Status != workflow.IntegrationUnknown || after.Integrations[0].Diagnostic != before.Integrations[0].Diagnostic || after.Integrations[0].Checks[0].Status != "unknown" || after.Integrations[0].Checks[0].ExitCode != -1 {
				t.Fatalf("lost unknown evidence: %+v", after.Integrations)
			}
		})
	}
}

func TestIntegrationCheckObservationSemantics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result executiongraph.ProcessResult
		want   string
	}{
		{"unknown", executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true}, "unknown"},
		{"ambiguous-zero-exit", executiongraph.ProcessResult{EffectAmbiguous: true}, "unknown"},
		{"unknown-even-if-stopped", executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true, Stopped: true}, "unknown"},
		{"confirmed-timeout", executiongraph.ProcessResult{ExitCode: -1, TimedOut: true, Stopped: true}, workflow.CheckTerminated},
		{"spawn-refusal", executiongraph.ProcessResult{ExitCode: -1, SpawnFailed: true}, workflow.CheckTerminated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Engine{Runner: &fakeRunner{run: func(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
				return tc.result
			}}}
			got := e.runIntegrationCheck(context.Background(), realTempDir(t), checkCommand("exit 0", 10)[0], func() error { return nil })
			if got.Status != tc.want {
				t.Fatalf("got %+v, want %s", got, tc.want)
			}
		})
	}
}

// S23-CR-02: lease-free persisted cancellation stops this scope and no later check.
func TestIntegratePersistedCancelChecks(t *testing.T) {
	for _, mode := range []string{"active-group", "unconfirmed-stop", "between-checks", "start-race"} {
		t.Run(mode, func(t *testing.T) {
			f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a.txt", "changed\n")})
			marker := filepath.Join(t.TempDir(), "running")
			f.checks = checkCommand("echo ready > '"+marker+"'; sleep 30", 10)
			second := checkCommand("exit 0", 10)[0]
			second.ID = "second"
			f.checks = append(f.checks, second)
			x := f.start(engineExecutionID, "a")
			cancelIntent := func() {
				if _, err := f.engine.Cancel(context.Background(), "proj", engineExecutionID, "human:maintainer"); err != nil {
					t.Error(err)
				}
			}
			entered := make(chan struct{})
			r := &fakeRunner{run: func(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
				if mode == "between-checks" {
					cancelIntent()
					return executiongraph.ProcessResult{}
				}
				if mode == "unconfirmed-stop" {
					close(entered)
					<-ctx.Done()
					return executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true, EffectAmbiguous: true}
				}
				return (executiongraph.OSProcessRunner{}).RunObserved(ctx, inv, func(p *executiongraph.ProcessCorrelation) error {
					f.lastCheck.Store(p)
					if mode == "start-race" {
						cancelIntent()
					}
					return started(p)
				})
			}}
			f.engine.Runner = r
			g := &faultGit{IntegrationGit: f.git}
			ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			type outcome struct {
				x   workflow.Execution
				err error
			}
			done := make(chan outcome, 1)
			go func() {
				got, err := f.engine.Integrate(ctx, "proj", engineExecutionID, g, f.request(x, "a", "cancel-check", f.base))
				done <- outcome{got, err}
			}()
			if mode == "active-group" {
				deadline := time.After(3 * time.Second)
				for {
					if _, err := os.Stat(marker); err == nil {
						break
					}
					select {
					case <-deadline:
						t.Fatal("check never reached barrier")
					case <-time.After(10 * time.Millisecond):
					}
				}
				cancelIntent()
			} else if mode == "unconfirmed-stop" {
				select {
				case <-entered:
				case <-time.After(3 * time.Second):
					t.Fatal("check never started")
				}
				cancelIntent()
			}
			var got outcome
			select {
			case got = <-done:
			case <-time.After(2 * time.Second):
				stop()
				<-done // join the runner even when the regression fails
				t.Fatal("persisted cancel did not stop active check")
			}
			wantErr := ErrNotIntegrated
			if mode == "unconfirmed-stop" || mode == "start-race" {
				wantErr = ErrIntegrationUnknown
			}
			if !errors.Is(got.err, wantErr) || r.spawns.Load() != 1 || g.advances.Load() != 0 || f.target() != f.base {
				t.Fatalf("cancel: calls=%d status=%s err=%v", r.spawns.Load(), got.x.Status, got.err)
			}
			if wantErr == ErrIntegrationUnknown && got.x.Status != workflow.RunRecoveryRequired {
				t.Fatalf("uncertain stop lost: %+v", got.x.Integrations)
			}
			if mode == "start-race" {
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatalf("command escaped start gate: %v", err)
				}
			}
			if p := f.lastCheck.Load(); p != nil {
				if current, err := executiongraph.ObserveRecordedProcess(*p); err != nil || current != nil && current.StartIdentity == p.StartIdentity {
					t.Fatalf("owned process survives: %+v %v", current, err)
				}
			}
		})
	}
}

type admissionSpyGit struct {
	IntegrationGit
	advances int
}

func (g *admissionSpyGit) Advance(ctx context.Context, ref, expected, merge string) (string, error) {
	g.advances++
	return g.IntegrationGit.Advance(ctx, ref, expected, merge)
}

func TestIntegrateRechecksAdmissionBeforeChecksAndAdvance(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		refuseAt int
		recorded bool
	}{
		{"integration", 1, false}, {"check", 2, true}, {"check-start", 4, true}, {"advance", 5, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newIntegrationFixture(t, map[string]edit{"a": commitFile("a.txt", "changed\n")})
			x := f.start(engineExecutionID, "a")
			denied := errors.New("repository admission revoked")
			calls := 0
			f.engine.Admit = func(context.Context, workflow.Execution) error {
				calls++
				if calls >= scenario.refuseAt {
					return denied
				}
				return nil
			}
			if scenario.name == "check-start" {
				f.engine.Runner = &fakeRunner{run: func(_ context.Context, _ executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
					if err := started(&executiongraph.ProcessCorrelation{PID: 779, StartIdentity: "boot:1", GroupIdentity: "779"}); err == nil {
						t.Error("start admission was not revoked")
					}
					return executiongraph.ProcessResult{OutcomeUnknown: true, ExitCode: 125}
				}}
			}
			spy := &admissionSpyGit{IntegrationGit: f.git}
			x, err := f.integrate(engineExecutionID, spy, f.request(x, "a", "admission", f.base))
			if err == nil || spy.advances != 0 || f.target() != f.base {
				t.Fatalf("unauthorized effect: err=%v advances=%d calls=%d", err, spy.advances, calls)
			}
			if scenario.recorded {
				wantStatus := workflow.IntegrationNotIntegrated
				if scenario.name == "check-start" {
					wantStatus = workflow.IntegrationUnknown
				}
				if len(x.Integrations) != 1 || x.Integrations[0].Status != wantStatus {
					t.Fatalf("missing refusal outcome: %+v", x.Integrations)
				}
				wantDiagnostic := "admission_revoked_before_check"
				if scenario.name == "advance" {
					wantDiagnostic = "admission_revoked_before_effect"
				}
				if scenario.name == "check-start" {
					wantDiagnostic += "_outcome_unknown"
				}
				if x.Integrations[0].Diagnostic != wantDiagnostic {
					t.Fatalf("mislabelled refusal: %s want %s", x.Integrations[0].Diagnostic, wantDiagnostic)
				}
				if scenario.name == "check" && len(x.Integrations[0].Checks) != 0 {
					t.Fatal("revoked command ran")
				}
				if scenario.name == "advance" && (len(x.Integrations[0].Checks) != 1 || x.Integrations[0].Checks[0].Status != workflow.CheckPassed) {
					t.Fatalf("advance check missing: %+v", x.Integrations)
				}
			} else if len(x.Integrations) != 0 {
				t.Fatal("pre-intent refusal recorded an integration")
			}
		})
	}
}

func writeFileOnly(name, content string) edit {
	return func(t *testing.T, dir string) { write(t, filepath.Join(dir, name), content) }
}

// A sandboxed writer (Codex keeps .git read-only) leaves its work uncommitted;
// the principal records it as the candidate, and post-merge checks get the
// allowlisted environment (HOME) toolchains need.
func TestIntegrateUncommittedSandboxedWriterWithHomeAwareCheck(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newIntegrationFixture(t, map[string]edit{"a": writeFileOnly("a-new.txt", "from a\n"), "b": commitFile("b-new.txt", "from b\n")})
	f.checks = checkCommand(`test -n "$HOME" && test -d "$HOME" && test -f a-new.txt`, 10)
	x := f.start(engineExecutionID, "a", "b")
	x, err := f.integrate(engineExecutionID, f.git, f.request(x, "a", "int-a", f.base))
	if err != nil {
		t.Fatal(err)
	}
	a := integrationOf(t, x, "int-a")
	if a.Status != workflow.IntegrationIntegrated || len(a.Checks) != 1 || a.Checks[0].Status != workflow.CheckPassed {
		t.Fatalf("uncommitted writer integration: %+v", a)
	}
	if parents(t, f.repo, a.CandidateSHA)[0] != f.base || gitT(t, f.repo, "log", "-1", "--format=%an", a.CandidateSHA) != "axiom-principal" {
		t.Fatal("candidate is not one principal commit on base")
	}
	if status := gitT(t, filepath.Join(f.root, "agent-a"), "status", "--porcelain"); status != "" {
		t.Fatalf("writer worktree left dirty: %q", status)
	}
}
