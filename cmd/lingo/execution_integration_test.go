//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// This executable is a fake vendor: it consumes the real public task packet,
// commits only in the assigned temporary workspace and never contacts a service.
const integrationRuntimeStub = `#!/bin/sh
set -eu
IFS= read -r packet || [ -n "$packet" ]
dir=${packet#*\"OutputDirectory\":\"}; dir=${dir%%\"*}
root=${packet#*\"RepositoryRoot\":\"}; root=${root%%\"*}
agent=${packet#*\"AgentID\":\"}; agent=${agent%%\"*}
cd "$root"
case "$agent" in
a|b)
  shared=${root%/*}
  printf ready > "$shared/$agent-ready"
  n=0
  case "$packet" in *mode:dependent*) printf ready > "$shared/b-ready" ;; esac
  while [ ! -f "$shared/a-ready" ] || [ ! -f "$shared/b-ready" ]; do
    n=$((n+1)); [ "$n" -lt 200 ] || exit 8; /bin/sleep 0.05
  done
  file=$agent.txt
  case "$packet" in *mode:conflict*) file=shared.txt ;; esac
  printf '%s\n' "$agent" > "$file"
  /usr/bin/git -c user.name=fixture -c user.email=fixture@localhost.invalid add "$file"
  /usr/bin/git -c user.name=fixture -c user.email=fixture@localhost.invalid commit -q -m "fixture $agent"
  ;;
observer)
  [ "$(cat a.txt)" = a ] && [ "$(cat b.txt)" = b ] || exit 9
  ;;
esac
printf 'Validated fixture result\n' > "$dir/result"
`

func integrationGit(t *testing.T, repository string, args ...string) string {
	t.Helper()
	cmd := exec.Command("/usr/bin/git", append([]string{"-C", repository, "-c", "user.name=fixture", "-c", "user.email=fixture@localhost.invalid", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func integrationDefinition(mode string, checks bool) workflowdefinition.Definition {
	builtin, _ := workflowdefinition.BuiltinRevision(2)
	d := builtin.Definition
	d.WorkflowID, d.Name, d.Revision = "same-repository", "Same repository fixture", 1
	d.Stages = d.Stages[:1]
	s := &d.Stages[0]
	s.ID = "work"
	s.Checkpoint, s.Phase = true, "implementation"
	s.Mode = "dag"
	s.Instructions = "Complete the fixture " + mode
	s.HumanGates, s.Validators, s.CompletionCriteria = []workflowdefinition.Gate{}, []workflowdefinition.Validator{}, []workflowdefinition.Criterion{}
	a := s.Agents[0]
	a.ID, a.Repository, a.Outputs = "a", "main", []string{"result"}
	a.EffectCeilings = []string{"repository-write", "process"}
	a.IntegrationOwner = false
	a.DependsOn = []string{}
	b := a
	b.ID = "b"
	if mode == "mode:dependent" {
		b.DependsOn = []string{"a"}
	}
	observer := a
	observer.ID, observer.EffectCeilings, observer.DependsOn = "observer", []string{"process"}, []string{"a", "b"}
	observer.IntegrationOwner = true
	s.Agents = []workflowdefinition.Agent{a, b, observer}
	if mode == "mode:gate" {
		s.Phase = "completion"
		s.HumanGates = []workflowdefinition.Gate{{Kind: "human_acceptance", Timing: "after"}}
	}
	if mode == "mode:held" {
		s.HumanGates = []workflowdefinition.Gate{{Kind: "implementation_authority", Timing: "before"}}
	}
	if checks {
		command := "exit 0"
		if mode == "mode:check-failure" {
			command = `[ "$(cat a.txt)" != a ] || [ "$(cat b.txt)" != b ]`
		}
		s.Validators = []workflowdefinition.Validator{{ID: "post-merge", Kind: "command", PolicyRef: "builtin-sdd-v1", Argv: []string{"/bin/sh", "-c", command}, TimeoutSeconds: 10, EffectCeilings: []string{"validation_command"}}}
	}
	return d
}

func newPublicIntegrationFixture(t *testing.T, mode string, checks bool) (*executionRunFixture, string) {
	t.Helper()
	f, paths := newExecutionRunFixtureRepositories(t, "main")
	bin := os.Getenv("PATH")
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte(integrationRuntimeStub), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	repo := paths["main"]
	integrationGit(t, repo, "init", "-q", "-b", "main")
	for _, name := range []string{"a.txt", "b.txt", "shared.txt"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte("base\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	integrationGit(t, repo, "add", ".")
	integrationGit(t, repo, "commit", "-q", "-m", "fixture base")
	f.installWorkflowDefinition(t, integrationDefinition(mode, checks))
	t.Setenv("LINGO_INVOCATION_DEFAULTS", `["runtime=codex","model=stub-model","effort=runtime-default"]`)
	return f, repo
}

func publicIntegrationArgs() []string {
	return []string{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--authorize-external", "--authorize-commands"}
}

func loadPublicIntegration(t *testing.T, f *executionRunFixture, id string) workflow.Execution {
	t.Helper()
	s, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	x, err := s.Load(context.Background(), workItemSourceProjectID, id)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestPublicWorkflowRunSameRepositoryIntegration(t *testing.T) {
	f, repo := newPublicIntegrationFixture(t, "", true)
	base := integrationGit(t, repo, "rev-parse", "HEAD")
	got := f.run(t, cli.ExitSuccess, publicIntegrationArgs()...)
	x := loadPublicIntegration(t, f, got.Stage.ExecutionID)
	if x.Status != workflow.RunCompleted || len(x.Integrations) != 2 {
		t.Fatalf("%s integrations=%+v", got.Raw, x.Integrations)
	}
	head := base
	for _, i := range x.Integrations {
		if i.Status != workflow.IntegrationIntegrated || i.ExpectedTargetHead != head || len(i.Checks) != 1 || i.Checks[0].Status != workflow.CheckPassed {
			t.Fatalf("integration: %+v", i)
		}
		if parents := integrationGit(t, repo, "show", "-s", "--format=%P", i.FinalSHA); parents != head+" "+i.CandidateSHA {
			t.Fatalf("parents %s", parents)
		}
		head = i.FinalSHA
	}
	if integrationGit(t, repo, "rev-parse", x.Integrations[1].TargetRef) != head || integrationGit(t, repo, "rev-parse", "HEAD") != base {
		t.Fatal("target or operator HEAD changed unexpectedly")
	}
	if len(x.Stages[0].Tasks[2].Attempts) != 1 || x.Stages[0].Tasks[2].Attempts[0].Status != workflow.AttemptSucceeded {
		t.Fatal("dependent did not observe integrated contents")
	}
	if integrationGit(t, repo, "remote") != "" || integrationGit(t, repo, "status", "--porcelain") != "" {
		t.Fatal("operator workspace modified")
	}
	before := integrationGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)")
	f.run(t, cli.ExitSuccess, "workflow", "run", "--project", "external", "--execution", x.ExecutionID, "--authorize-external", "--authorize-commands")
	if after := loadPublicIntegration(t, f, x.ExecutionID); len(after.Integrations) != 2 || integrationGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)") != before {
		t.Fatal("completed integration replayed")
	}
}

func TestPublicWorkflowRunSameRepositoryDependentWriterAndHumanGate(t *testing.T) {
	for _, mode := range []string{"dependent", "gate"} {
		t.Run(mode, func(t *testing.T) {
			f, repo := newPublicIntegrationFixture(t, "mode:"+mode, true)
			want := cli.ExitSuccess
			if mode == "gate" {
				want = cli.ExitFailure
			}
			got := f.run(t, want, publicIntegrationArgs()...)
			x := loadPublicIntegration(t, f, got.Stage.ExecutionID)
			if mode == "gate" {
				for _, task := range x.Stages[0].Tasks {
					if len(task.Attempts) != 0 {
						t.Fatal("after gate changed timing or spawned partial work")
					}
				}
				if got.Category != "integration_refused" || len(x.Integrations) != 0 || x.Stages[0].Gate != nil || strings.Contains(integrationGit(t, repo, "for-each-ref", "--format=%(refname)"), "axiom/") {
					t.Fatal(got.Raw)
				}
				return
			}
			if x.Status != workflow.RunCompleted || len(x.Enrollments) != 1 || len(x.Integrations) != 2 || len(x.Stages[0].Tasks[2].Attempts) != 1 {
				t.Fatalf("dependent writer enrollment/completion: %s", got.Raw)
			}
			b := x.Stages[0].Tasks[1].Attempts[0]
			if b.Workspace.BaseSHA != x.Integrations[0].FinalSHA {
				t.Fatal("dependent writer did not start from integrated result")
			}
		})
	}
}

func TestPublicWorkflowRunSameRepositoryStopsOnConflictOrFailedCheck(t *testing.T) {
	for _, mode := range []string{"mode:conflict", "mode:check-failure"} {
		t.Run(mode, func(t *testing.T) {
			f, repo := newPublicIntegrationFixture(t, mode, true)
			base := integrationGit(t, repo, "rev-parse", "HEAD")
			got := f.run(t, cli.ExitFailure, publicIntegrationArgs()...)
			x := loadPublicIntegration(t, f, got.Stage.ExecutionID)
			if len(x.Integrations) != 2 || x.Integrations[0].Status != workflow.IntegrationIntegrated || x.Integrations[1].Status != workflow.IntegrationNotIntegrated || len(x.Stages[0].Tasks[2].Attempts) != 0 {
				t.Fatalf("%s %+v", got.Raw, x.Integrations)
			}
			want := "merge_conflict"
			if mode == "mode:check-failure" {
				want = "post_merge_check_failed"
			}
			if x.Integrations[1].Diagnostic != want || integrationGit(t, repo, "rev-parse", x.Integrations[0].TargetRef) != x.Integrations[0].FinalSHA || integrationGit(t, repo, "rev-parse", "HEAD") != base {
				t.Fatalf("unsafe failure: %+v", x.Integrations)
			}
			f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--execution", x.ExecutionID, "--authorize-external", "--authorize-commands")
			if after := loadPublicIntegration(t, f, x.ExecutionID); len(after.Integrations) != 2 || len(after.Stages[0].Tasks[2].Attempts) != 0 {
				t.Fatal("failure automatically replayed")
			}
		})
	}
}

func TestPublicWorkflowRunSameRepositoryRefusesMissingChecksAuthorityAndDirtyBase(t *testing.T) {
	for _, mode := range []string{"checks", "authority", "dirty"} {
		t.Run(mode, func(t *testing.T) {
			f, repo := newPublicIntegrationFixture(t, "", mode != "checks")
			if mode == "dirty" {
				if err := os.WriteFile(filepath.Join(repo, "untracked"), []byte("preserve me"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := integrationGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)")
			args := publicIntegrationArgs()
			if mode == "authority" {
				args = args[:len(args)-1]
			}
			got := f.run(t, cli.ExitFailure, args...)
			if got.Stage.ExecutionID != "" {
				x := loadPublicIntegration(t, f, got.Stage.ExecutionID)
				for _, task := range x.Stages[0].Tasks {
					if len(task.Attempts) != 0 {
						t.Fatalf("refusal spawned/prepared: %s", got.Raw)
					}
				}
			}
			if integrationGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)") != before {
				t.Fatal("refusal changed refs")
			}
		})
	}
}

// Faults wrap the ports of the actual CLI composition, retaining its store,
// Git boundary, RuntimeLauncher, resolver and admission checks.
type integrationFaultStore struct {
	stageexecution.ExecutionStore
	mode         string
	fired        bool
	afterPrepare func(workflow.Execution)
}

func (s *integrationFaultStore) UpdateFenced(ctx context.Context, project, execution string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	prepared := false
	x, err := s.ExecutionStore.UpdateFenced(ctx, project, execution, fence, func(x *workflow.Execution) error {
		oldAttempts, oldIntegrations := 0, len(x.Integrations)
		for _, stage := range x.Stages {
			for _, task := range stage.Tasks {
				oldAttempts += len(task.Attempts)
			}
		}
		if err := mutate(x); err != nil {
			return err
		}
		newAttempts := 0
		for _, stage := range x.Stages {
			for _, task := range stage.Tasks {
				newAttempts += len(task.Attempts)
			}
		}
		prepared = newAttempts > oldAttempts
		if !s.fired {
			fault := false
			switch s.mode {
			case "lost-result":
				for _, stage := range x.Stages {
					for _, task := range stage.Tasks {
						for _, a := range task.Attempts {
							fault = fault || a.Status == workflow.AttemptSucceeded
						}
					}
				}
			case "lost-integration-debt":
				for _, stage := range x.Stages {
					for _, task := range stage.Tasks {
						for _, a := range task.Attempts {
							if a.Status == workflow.AttemptSucceeded {
								s.fired = true
							}
						}
					}
				}
			case "lost-integration-outcome":
				for _, i := range x.Integrations {
					fault = fault || i.Status == workflow.IntegrationIntegrated
				}
			}
			if fault {
				s.fired = true
				return errors.New("synthetic lost terminal commit")
			}
			if s.mode == "lost-integration-intent" && len(x.Integrations) > oldIntegrations {
				s.fired = true
			}
		}
		return nil
	})
	if err == nil && prepared && s.afterPrepare != nil {
		s.afterPrepare(x)
		s.afterPrepare = nil
	}
	if err == nil && !s.fired && s.mode == "lost-workspace-intent" && prepared {
		s.fired = true
		return x, errors.New("synthetic lost workspace intent response")
	}
	if err == nil && s.fired && s.mode == "lost-integration-intent" {
		return x, errors.New("synthetic lost integration intent response")
	}
	if err == nil && s.fired && s.mode == "lost-integration-debt" {
		return x, errors.New("synthetic lost result response before integration")
	}
	return x, err
}

type integrationFaultGit struct {
	stageexecution.WorkspaceGit
	mode     string
	advances *int
	cancel   func()
}

func (g integrationFaultGit) Advance(ctx context.Context, ref, expected, merge string) (string, error) {
	*g.advances++
	head, err := g.WorkspaceGit.Advance(ctx, ref, expected, merge)
	if g.mode == "lost-advance" {
		return "", errors.New("synthetic lost ref outcome")
	}
	return head, err
}

func (g integrationFaultGit) Merge(ctx context.Context, head, candidate, message string) (string, error) {
	merge, err := g.WorkspaceGit.Merge(ctx, head, candidate, message)
	if g.cancel != nil {
		g.cancel()
	}
	return merge, err
}

func publicIntegrationHeldAtGate(t *testing.T) (*executionRunFixture, string, executionComposition, string) {
	t.Helper()
	f, repo := newPublicIntegrationFixture(t, "mode:held", true)
	got := f.run(t, cli.ExitSuccess, publicIntegrationArgs()...)
	if got.Stage.Status != workflow.RunAwaitingGate {
		t.Fatal(got.Raw)
	}
	s := f.env.service.(lifecycleService)
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	resolved := s.installation.Resolve(context.Background(), "external")
	c := executionComposition{s: s, store: store, project: resolved.Project, authorizeCommands: true}
	return f, repo, c, got.Stage.ExecutionID
}

func TestPublicCompositionSameRepositoryFaultsNeverReplay(t *testing.T) {
	for _, mode := range []string{"lost-workspace-intent", "lost-result", "lost-integration-debt", "lost-integration-intent", "lost-integration-outcome", "lost-advance"} {
		t.Run(mode, func(t *testing.T) {
			f, repo, c, id := publicIntegrationHeldAtGate(t)
			ctx := context.Background()
			if _, err := c.engine(nil).ApproveGate(ctx, c.project.ID, id, "work", "human:fixture"); err != nil {
				t.Fatal(err)
			}
			launcher, _, err := c.existingLauncher(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			e := c.engine(launcher)
			faults := &integrationFaultStore{ExecutionStore: e.Store, mode: mode}
			e.Store = faults
			advances := 0
			factory := e.Workspaces
			e.Workspaces = func(ctx context.Context, x workflow.Execution, key string) (stageexecution.WorkspaceGit, error) {
				g, err := factory(ctx, x, key)
				return integrationFaultGit{WorkspaceGit: g, mode: mode, advances: &advances}, err
			}
			_, _ = e.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
			beforeRefs := integrationGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)")
			before := loadPublicIntegration(t, f, id)
			count := 0
			for _, task := range before.Stages[0].Tasks {
				count += len(task.Attempts)
			}
			got := f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--execution", id, "--authorize-external", "--authorize-commands")
			after := loadPublicIntegration(t, f, id)
			if got.Category != "recovery_required" || after.Status != workflow.RunRecoveryRequired || len(after.Stages[0].Tasks[2].Attempts) != 0 {
				t.Fatalf("%s %+v", got.Raw, after.Integrations)
			}
			afterCount := 0
			for _, task := range after.Stages[0].Tasks {
				afterCount += len(task.Attempts)
			}
			if count != afterCount || beforeRefs != integrationGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)") || advances > 1 {
				t.Fatal("uncertain effect replayed")
			}
			if mode != "lost-advance" && !faults.fired {
				t.Fatal("fault was not reached")
			}
			if (mode == "lost-advance" || mode == "lost-integration-outcome") && advances != 1 {
				t.Fatal("ref-outcome fault was not reached exactly once")
			}
		})
	}
}

func TestPublicCompositionSameRepositoryChangedBaseCollisionAndCancel(t *testing.T) {
	for _, mode := range []string{"changed-base", "branch-collision", "worktree-collision", "cancel", "admission"} {
		t.Run(mode, func(t *testing.T) {
			f, repo, c, id := publicIntegrationHeldAtGate(t)
			ctx := context.Background()
			if _, err := c.engine(nil).ApproveGate(ctx, c.project.ID, id, "work", "human:fixture"); err != nil {
				t.Fatal(err)
			}
			launcher, _, err := c.existingLauncher(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			e := c.engine(launcher)
			advances := 0
			if mode != "cancel" && mode != "admission" {
				e.Store = &integrationFaultStore{ExecutionStore: e.Store, afterPrepare: func(x workflow.Execution) {
					w := x.Stages[0].Tasks[0].Attempts[0].Workspace
					switch mode {
					case "changed-base":
						integrationGit(t, repo, "commit", "--allow-empty", "-q", "-m", "external base drift")
					case "branch-collision":
						integrationGit(t, repo, "branch", strings.TrimPrefix(w.SourceBranch, "refs/heads/"), w.BaseSHA)
					case "worktree-collision":
						path := filepath.Join(executionWorktreeRoot(repo, id, "main"), w.Worktree)
						if err := os.MkdirAll(path, 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(path, "preserve"), []byte("collision"), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}}
			}
			factory := e.Workspaces
			e.Workspaces = func(ctx context.Context, x workflow.Execution, key string) (stageexecution.WorkspaceGit, error) {
				g, err := factory(ctx, x, key)
				wrapped := integrationFaultGit{WorkspaceGit: g, advances: &advances}
				if mode == "cancel" {
					wrapped.cancel = func() {
						if _, err := c.engine(nil).Cancel(ctx, c.project.ID, id, "user:fixture"); err != nil {
							t.Error(err)
						}
					}
				}
				if mode == "admission" {
					wrapped.cancel = func() { operational(t, f.env.state, projectapp.ArchiveProject) }
				}
				return wrapped, err
			}
			_, _ = e.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
			x := loadPublicIntegration(t, f, id)
			if mode == "branch-collision" || mode == "worktree-collision" {
				w := x.Stages[0].Tasks[0].Attempts[0].Workspace
				if mode == "branch-collision" && integrationGit(t, repo, "rev-parse", w.SourceBranch) != w.BaseSHA {
					t.Fatal("colliding branch was adopted or overwritten")
				}
				if mode == "worktree-collision" {
					content, err := os.ReadFile(filepath.Join(executionWorktreeRoot(repo, id, "main"), w.Worktree, "preserve"))
					if err != nil || string(content) != "collision" {
						t.Fatal("colliding worktree was overwritten")
					}
					if strings.Contains(integrationGit(t, repo, "worktree", "list", "--porcelain"), w.Worktree) {
						t.Fatal("colliding worktree was adopted")
					}
				}
			}
			if advances != 0 || len(x.Stages[0].Tasks[2].Attempts) != 0 {
				t.Fatalf("unsafe effect after %s: %+v", mode, x.Integrations)
			}
			if mode == "cancel" && (x.Cancel == nil || len(x.Integrations) != 1 || x.Integrations[0].Status != workflow.IntegrationNotIntegrated) {
				t.Fatalf("cancel outcome %+v", x.Integrations)
			}
			if mode == "admission" && (x.Cancel != nil || len(x.Integrations) != 1 || x.Integrations[0].Status != workflow.IntegrationNotIntegrated || x.Integrations[0].Diagnostic != "admission_revoked_before_check" || len(x.Integrations[0].Checks) != 0) {
				t.Fatalf("revocation outcome %+v", x.Integrations)
			}
		})
	}
}
