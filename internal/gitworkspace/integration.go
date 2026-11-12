package gitworkspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rgomids/axiom/internal/workflow"
)

// S2.3 principal-controlled integration of isolated agent branches. This is
// the shared hardened Git boundary (executable configuration veto, rooted
// no-symlink worktree paths, no cleanup), bound to one repository and one
// principal-authorized worktree root. It never pushes, fetches, deletes a ref/branch/worktree or resets a
// working tree; the only ref it moves is the principal's integration target,
// and only by compare-and-swap.
var (
	ErrTargetCheckedOut = workflow.ErrIntegrationTargetCheckedOut
	ErrTargetMoved      = errors.New("Git integration target moved")
	ErrCandidateInvalid = errors.New("Git integration candidate invalid")
)

const (
	principalName  = "axiom-principal"
	principalEmail = "axiom-principal@localhost.invalid"
)

// Integrator binds the exact repository and worktree root a principal is
// authorized to use.
type Integrator struct {
	m *Manager
	// mu serializes target observation and advance within this process; the
	// update-ref compare-and-swap serializes across processes.
	mu sync.Mutex
}

// WorktreeObservation is one isolated agent worktree on its own branch.
type WorktreeObservation struct {
	Path, Branch, BaseSHA, HeadSHA string
}

func NewIntegrator(ctx context.Context, repository, root string) (*Integrator, error) {
	m, err := openIntegrationRepository(ctx, repository)
	if err != nil {
		return nil, err
	}
	root, err = plannedIntegrationRoot(root)
	if err != nil || samePath(root, m.repository) || isWithin(m.repository, root) || isWithin(root, m.repository) {
		return nil, ErrInvalidConfiguration
	}
	m.root = root
	return &Integrator{m: m}, nil
}

func (g *Integrator) worktreePath(name string, mustExist bool) (string, error) {
	if !workflow.ValidWorktreeName(name) {
		return "", ErrWorkspaceInvalid
	}
	return g.m.confinedWorkspace(filepath.Join(g.m.root, name), mustExist)
}

func (g *Integrator) refExists(ctx context.Context, ref string) bool {
	_, err := g.m.runGit(ctx, g.m.repository, nil, nil, "show-ref", "--verify", "--quiet", ref)
	return err == nil
}

// PrepareAgentWorktree is the recommended isolation for agents editing one
// Repository: a new worktree on a new branch at base. It refuses an existing
// path or branch rather than reusing or overwriting user state.
func (g *Integrator) PrepareAgentWorktree(ctx context.Context, name, branch, base string) (WorktreeObservation, error) {
	if _, err := cleanAuthorizedRoot(g.m.root); err != nil {
		return WorktreeObservation{}, err
	}
	if !workflow.ValidBranchRef(branch) || !workflow.ValidObjectID(base) {
		return WorktreeObservation{}, ErrWorkspaceInvalid
	}
	path, err := g.worktreePath(name, false)
	if err != nil {
		return WorktreeObservation{}, err
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) || g.refExists(ctx, branch) {
		return WorktreeObservation{}, ErrWorkspaceCollision
	}
	if _, err := g.m.revision(ctx, g.m.repository, base); err != nil {
		return WorktreeObservation{}, ErrCandidateInvalid
	}
	if _, err := g.m.gitOutput(ctx, g.m.repository, nil, nil, "worktree", "add", "-b", strings.TrimPrefix(branch, "refs/heads/"), path, base); err != nil {
		return WorktreeObservation{}, fmt.Errorf("%w: %v", ErrWorkspaceCollision, err)
	}
	return WorktreeObservation{Path: path, Branch: branch, BaseSHA: base, HeadSHA: base}, nil
}

// worktreeAt verifies that path is a registered worktree of this repository
// (same common directory) and returns its listed branch ("" when detached)
// and HEAD.
func (g *Integrator) worktreeAt(ctx context.Context, path string) (string, string, error) {
	if err := g.m.rejectExecutableConfiguration(ctx, path); err != nil {
		return "", "", err
	}
	top, err := g.m.gitPath(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil || !samePath(top, path) {
		return "", "", ErrWorkspaceInvalid
	}
	common, err := g.m.gitPath(ctx, path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil || !samePath(common, g.m.commonDirectory) {
		return "", "", ErrWorkspaceInvalid
	}
	for _, w := range g.worktrees(ctx) {
		if samePath(w.path, path) {
			return w.branch, w.head, nil
		}
	}
	return "", "", ErrWorkspaceInvalid
}

type listedWorktree struct{ path, head, branch string }

func (g *Integrator) worktrees(ctx context.Context) []listedWorktree {
	output, err := g.m.gitOutput(ctx, g.m.repository, nil, nil, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil
	}
	var out []listedWorktree
	for _, field := range splitNUL(output) {
		switch {
		case strings.HasPrefix(field, "worktree "):
			out = append(out, listedWorktree{path: strings.TrimPrefix(field, "worktree ")})
		case len(out) != 0 && strings.HasPrefix(field, "HEAD "):
			out[len(out)-1].head = strings.TrimPrefix(field, "HEAD ")
		case len(out) != 0 && strings.HasPrefix(field, "branch "):
			out[len(out)-1].branch = strings.TrimPrefix(field, "branch ")
		}
	}
	return out
}

// ObserveCandidate returns the committed candidate of an agent worktree: on
// exactly its branch, descending from base, with no tracked or untracked
// change outside that commit.
func (g *Integrator) ObserveCandidate(ctx context.Context, name, branch, base string) (string, error) {
	if !workflow.ValidBranchRef(branch) || !workflow.ValidObjectID(base) {
		return "", ErrCandidateInvalid
	}
	path, err := g.worktreePath(name, true)
	if err != nil {
		return "", err
	}
	listedBranch, head, err := g.worktreeAt(ctx, path)
	if err != nil {
		return "", err
	}
	if listedBranch != branch {
		return "", ErrWorkspaceInvalid
	}
	changed, err := g.m.changedPaths(ctx, path)
	if err != nil || len(changed) != 0 {
		return "", ErrForeignMutation
	}
	tip, err := g.m.revision(ctx, g.m.repository, branch)
	if err != nil || tip != head {
		return "", ErrCandidateInvalid
	}
	if _, err := g.m.revision(ctx, g.m.repository, base); err != nil || tip == base {
		return "", ErrNoResult
	}
	if _, err := g.m.runGit(ctx, g.m.repository, nil, nil, "merge-base", "--is-ancestor", base, tip); err != nil {
		return "", ErrCandidateInvalid
	}
	return tip, nil
}

// ObserveTarget returns the integration target's head. A target checked out
// in any worktree is refused: moving it would desynchronize, and could
// overwrite, that working tree's state.
func (g *Integrator) ObserveTarget(ctx context.Context, ref string) (string, error) {
	if !workflow.ValidBranchRef(ref) {
		return "", ErrWorkspaceInvalid
	}
	for _, w := range g.worktrees(ctx) {
		if w.branch == ref {
			return "", ErrTargetCheckedOut
		}
	}
	return g.m.revision(ctx, g.m.repository, ref)
}

// Merge computes the merge of candidate into head without touching any ref,
// index or working tree, and returns the intermediate merge commit. A conflict
// is reported, never resolved.
func (g *Integrator) Merge(ctx context.Context, head, candidate, message string) (string, error) {
	if !workflow.ValidObjectID(head) || !workflow.ValidObjectID(candidate) || strings.ContainsAny(message, "\x00") {
		return "", ErrCandidateInvalid
	}
	if err := g.m.rejectExecutableConfiguration(ctx, g.m.repository); err != nil {
		return "", err
	}
	output, err := g.m.runGit(ctx, g.m.repository, nil, nil, "merge-tree", "--write-tree", "--no-messages", head, candidate)
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return "", fmt.Errorf("%w: %w", ErrIntegrationConflict, workflow.ErrIntegrationConflict)
		}
		return "", err
	}
	tree := strings.ToLower(strings.TrimSpace(strings.SplitN(string(output), "\n", 2)[0]))
	if !workflow.ValidObjectID(tree) {
		return "", ErrWorkspaceInvalid
	}
	identity := []string{"GIT_AUTHOR_NAME=" + principalName, "GIT_AUTHOR_EMAIL=" + principalEmail, "GIT_COMMITTER_NAME=" + principalName, "GIT_COMMITTER_EMAIL=" + principalEmail}
	return g.m.gitObjectWithEnv(ctx, g.m.repository, identity, "commit-tree", tree, "-p", head, "-p", candidate, "-m", message)
}

// PrepareCheckWorktree checks the merge result out, detached, in a new
// principal worktree where post-merge checks run. It is retained afterwards.
func (g *Integrator) PrepareCheckWorktree(ctx context.Context, name, merge string) (string, error) {
	if _, err := cleanAuthorizedRoot(g.m.root); err != nil {
		return "", err
	}
	if !workflow.ValidObjectID(merge) {
		return "", ErrCandidateInvalid
	}
	path, err := g.worktreePath(name, false)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return "", ErrWorkspaceCollision
	}
	if _, err := g.m.gitOutput(ctx, g.m.repository, nil, nil, "worktree", "add", "--detach", path, merge); err != nil {
		return "", fmt.Errorf("%w: %v", ErrWorkspaceCollision, err)
	}
	return path, g.ObserveCheckWorktree(ctx, path, merge)
}

// ObserveCheckWorktree verifies that the checks left the merge result exactly
// as checked out: detached at merge with no tracked or untracked change.
func (g *Integrator) ObserveCheckWorktree(ctx context.Context, path, merge string) error {
	path, err := g.m.confinedWorkspace(path, true)
	if err != nil {
		return err
	}
	branch, head, err := g.worktreeAt(ctx, path)
	if err != nil || branch != "" || head != merge {
		return ErrForeignMutation
	}
	changed, err := g.m.changedPaths(ctx, path)
	if err != nil || len(changed) != 0 {
		return ErrForeignMutation
	}
	return nil
}

// Advance moves the target from expected to merge by compare-and-swap and
// reports the head observed afterwards. An empty observation with an error
// means the effect cannot be established.
func (g *Integrator) Advance(ctx context.Context, ref, expected, merge string) (string, error) {
	if !workflow.ValidBranchRef(ref) || !workflow.ValidObjectID(expected) || !workflow.ValidObjectID(merge) {
		return "", ErrWorkspaceInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, err := g.ObserveTarget(ctx, ref); errors.Is(err, ErrTargetCheckedOut) {
		return "", err
	}
	_, advanceErr := g.m.runGit(ctx, g.m.repository, nil, nil, "update-ref", "-m", "axiom principal integration", ref, merge, expected)
	observed, observeErr := g.m.revision(context.WithoutCancel(ctx), g.m.repository, ref)
	if observeErr != nil {
		return "", errors.Join(advanceErr, observeErr)
	}
	if advanceErr != nil {
		return observed, fmt.Errorf("%w: %v", ErrTargetMoved, advanceErr)
	}
	return observed, nil
}

// openIntegrationRepository opens the existing hardened local Git boundary;
// read-only observations need no worktree root or workspace mutation.
func openIntegrationRepository(ctx context.Context, repository string) (*Manager, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	git, err = filepath.Abs(git)
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	repository, err = cleanExistingDirectory(repository)
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	m := &Manager{git: git, repository: repository}
	if err := m.rejectExecutableConfiguration(ctx, repository); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfiguration, err)
	}
	top, err := m.gitPath(ctx, repository, "rev-parse", "--show-toplevel")
	if err != nil || !samePath(top, repository) {
		return nil, ErrInvalidConfiguration
	}
	m.commonDirectory, err = m.gitPath(ctx, repository, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	return m, nil
}

// ObserveIntegration checks immutable Git objects/ancestry in the bound local
// repository. Only the latest accepted record for a target requires the ref
// still at FinalSHA. Historical source refs may have advanced or been removed;
// their recorded commits must remain verifiable. It performs no ref mutation.
func ObserveIntegration(ctx context.Context, repository string, i workflow.Integration, current bool) error {
	if i.Status != workflow.IntegrationIntegrated || !workflow.ValidBranchRef(i.SourceBranch) || !workflow.ValidBranchRef(i.TargetRef) || i.FinalSHA != i.MergeSHA || i.ObservedTargetHead != i.FinalSHA {
		return ErrCandidateInvalid
	}
	for _, sha := range []string{i.BaseSHA, i.ExpectedTargetHead, i.CandidateSHA, i.FinalSHA} {
		if !workflow.ValidObjectID(sha) {
			return ErrCandidateInvalid
		}
	}
	m, err := openIntegrationRepository(ctx, repository)
	if err != nil {
		return err
	}
	for _, sha := range []string{i.BaseSHA, i.ExpectedTargetHead, i.CandidateSHA, i.FinalSHA} {
		observed, err := m.revision(ctx, m.repository, sha)
		if err != nil || observed != sha {
			return ErrCandidateInvalid
		}
	}
	for _, pair := range [][2]string{{i.BaseSHA, i.CandidateSHA}, {i.CandidateSHA, i.FinalSHA}, {i.ExpectedTargetHead, i.FinalSHA}} {
		if _, err := m.runGit(ctx, m.repository, nil, nil, "merge-base", "--is-ancestor", pair[0], pair[1]); err != nil {
			return ErrCandidateInvalid
		}
	}
	if current {
		source, err := m.revision(ctx, m.repository, i.SourceBranch)
		if err != nil || source != i.CandidateSHA {
			return ErrCandidateInvalid
		}
		observed, err := m.revision(ctx, m.repository, i.TargetRef)
		if err != nil || observed != i.FinalSHA {
			return ErrTargetMoved
		}
	}
	return nil
}

// plannedIntegrationRoot validates a private sibling root without creating it.
// All materialization remains behind the principal's persisted intent.
func plannedIntegrationRoot(root string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", ErrInvalidConfiguration
	}
	for current := root; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !ownedByUser(info) {
				return "", ErrInvalidConfiguration
			}
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil || !samePath(resolved, current) {
				return "", ErrInvalidConfiguration
			}
			if current == root && !privateDirectory(root) {
				return "", ErrInvalidConfiguration
			}
			return root, nil
		}
		if !errors.Is(err, os.ErrNotExist) || current == filepath.Dir(current) {
			return "", ErrInvalidConfiguration
		}
	}
}

// ObserveBase refuses a dirty operator workspace and returns its current HEAD.
func (g *Integrator) ObserveBase(ctx context.Context) (string, error) {
	if err := g.m.rejectExecutableConfiguration(ctx, g.m.repository); err != nil {
		return "", err
	}
	changed, err := g.m.changedPaths(ctx, g.m.repository)
	if err != nil || len(changed) != 0 {
		return "", ErrForeignMutation
	}
	return g.m.revision(ctx, g.m.repository, "HEAD")
}

// PrepareTarget creates a new principal ref with Git's absent-value CAS. An
// existing ref is a collision even when it happens to point at the same base.
func (g *Integrator) PrepareTarget(ctx context.Context, ref, base string) error {
	if !workflow.ValidBranchRef(ref) || !workflow.ValidObjectID(base) {
		return ErrWorkspaceInvalid
	}
	if _, err := g.m.runGit(ctx, g.m.repository, nil, nil, "update-ref", ref, base, strings.Repeat("0", len(base))); err != nil {
		return fmt.Errorf("%w: %v", ErrWorkspaceCollision, err)
	}
	return nil
}

func (g *Integrator) MaterializeWorkspace(ctx context.Context, name, branch, base string) (string, error) {
	observation, err := g.PrepareAgentWorktree(ctx, name, branch, base)
	return observation.Path, err
}
