package gitworkspace_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/workflow"
)

func commitAll(t *testing.T, directory, message string) string {
	t.Helper()
	git(t, directory, "add", "-A")
	git(t, directory, "-c", "user.name=Axiom Test", "-c", "user.email=axiom@example.invalid", "commit", "-m", message)
	return git(t, directory, "rev-parse", "HEAD")
}

// divergedRepository returns a repository whose branches x and y both edit
// shared.txt, so a three-way content merge consults any merge driver.
func divergedRepository(t *testing.T, directory string) (string, string, string) {
	t.Helper()
	initRepository(t, directory)
	writeFile(t, filepath.Join(directory, ".gitattributes"), "*.txt merge=sentinel\n")
	writeFile(t, filepath.Join(directory, "shared.txt"), "one\ntwo\nthree\nfour\nfive\n")
	base := commitAll(t, directory, "attributes")
	git(t, directory, "checkout", "-q", "-b", "x")
	writeFile(t, filepath.Join(directory, "shared.txt"), "ONE\ntwo\nthree\nfour\nfive\n")
	x := commitAll(t, directory, "x")
	git(t, directory, "checkout", "-q", "-b", "y", base)
	writeFile(t, filepath.Join(directory, "shared.txt"), "one\ntwo\nthree\nfour\nFIVE\n")
	y := commitAll(t, directory, "y")
	return base, x, y
}

func TestIntegratorRejectsMergeDriverConfiguredAfterBinding(t *testing.T) {
	root := canonicalTempDir(t)
	controlSentinel := filepath.Join(root, "control-sentinel")
	control := filepath.Join(root, "control")
	_, cx, cy := divergedRepository(t, control)
	git(t, control, "config", "merge.sentinel.driver", filterCommand(controlSentinel)+"; exit 1")
	_, _ = gitAllowFailure(control, "merge-tree", "--write-tree", cx, cy)
	assertSentinel(t, controlSentinel)

	repository := filepath.Join(root, "repository")
	_, x, y := divergedRepository(t, repository)
	g, err := gitworkspace.NewIntegrator(context.Background(), repository, filepath.Join(root, "worktrees"))
	if err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(root, "sentinel")
	git(t, repository, "config", "merge.sentinel.driver", filterCommand(sentinel)+"; exit 1")
	if _, err := g.Merge(context.Background(), x, y, "probe"); !errors.Is(err, gitworkspace.ErrExecutableGitConfiguration) {
		t.Fatalf("merge driver not rejected: %v", err)
	}
	assertNoSentinel(t, sentinel)
	if _, err := gitworkspace.NewIntegrator(context.Background(), repository, filepath.Join(root, "worktrees2")); !errors.Is(err, gitworkspace.ErrExecutableGitConfiguration) {
		t.Fatalf("binding accepted a merge driver: %v", err)
	}
}

func TestIntegratorPreservesUserBranchesAndBindsExactRepository(t *testing.T) {
	root := canonicalTempDir(t)
	repository := initRepository(t, filepath.Join(root, "repository"))
	git(t, repository.repository, "branch", "user/keep")
	g, err := gitworkspace.NewIntegrator(context.Background(), repository.repository, filepath.Join(root, "worktrees"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.PrepareAgentWorktree(context.Background(), "agent-a", "refs/heads/user/keep", repository.revision); !errors.Is(err, gitworkspace.ErrWorkspaceCollision) {
		t.Fatalf("existing user branch reused: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "worktrees", "agent-a")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refused preparation left a worktree")
	}
	observed, err := g.PrepareAgentWorktree(context.Background(), "agent-a", "refs/heads/axiom/agent-a", repository.revision)
	if err != nil || observed.HeadSHA != repository.revision {
		t.Fatalf("prepare: %+v %v", observed, err)
	}
	if _, err := g.PrepareAgentWorktree(context.Background(), "agent-a", "refs/heads/axiom/agent-b", repository.revision); !errors.Is(err, gitworkspace.ErrWorkspaceCollision) {
		t.Fatalf("existing worktree path reused: %v", err)
	}
	for _, name := range []string{"../escape", ".hidden", "a/b", ""} {
		if _, err := g.PrepareAgentWorktree(context.Background(), name, "refs/heads/axiom/x", repository.revision); err == nil {
			t.Fatalf("worktree name %q accepted", name)
		}
	}
	if got := git(t, repository.repository, "rev-parse", "user/keep"); got != repository.revision {
		t.Fatal("user branch moved")
	}
	for name, bind := range map[string][2]string{
		"root inside repository": {repository.repository, filepath.Join(repository.repository, "worktrees")},
		"repository inside root": {repository.repository, root},
		"subdirectory":           {filepath.Join(repository.repository, "sub"), filepath.Join(root, "w3")},
	} {
		if name == "subdirectory" {
			writeFile(t, filepath.Join(repository.repository, "sub", "f"), "x\n")
		}
		if _, err := gitworkspace.NewIntegrator(context.Background(), bind[0], bind[1]); !errors.Is(err, gitworkspace.ErrInvalidConfiguration) {
			t.Fatalf("%s accepted: %v", name, err)
		}
	}
}

func TestObserveIntegrationTwoRepositoriesAndStaleOrForeignFacts(t *testing.T) {
	ctx := context.Background()
	root := canonicalTempDir(t)
	var repos []string
	var observations []workflow.Integration
	for _, key := range []string{"app", "lib"} {
		f := initRepository(t, filepath.Join(root, key))
		repo := f.repository
		writeFile(t, filepath.Join(repo, "result.txt"), key)
		candidate := commitAll(t, repo, key)
		git(t, repo, "branch", "agent", candidate)
		git(t, repo, "branch", "target", candidate)
		i := workflow.Integration{RepositoryKey: key, SourceBranch: "refs/heads/agent", TargetRef: "refs/heads/target", BaseSHA: f.revision, ExpectedTargetHead: f.revision, CandidateSHA: candidate, FinalSHA: candidate, MergeSHA: candidate, ObservedTargetHead: candidate, Status: workflow.IntegrationIntegrated}
		if err := gitworkspace.ObserveIntegration(ctx, repo, i, true); err != nil {
			t.Fatal(err)
		}
		repos = append(repos, repo)
		observations = append(observations, i)
	}
	if observations[0].CandidateSHA == observations[1].CandidateSHA {
		t.Fatal("fixture results not distinct")
	}
	if gitworkspace.ObserveIntegration(ctx, repos[0], observations[1], true) == nil {
		t.Fatal("foreign repository result accepted")
	}
	for _, mode := range []string{"ref", "source-ref", "source", "base", "sha", "path"} {
		t.Run(mode, func(t *testing.T) {
			i := observations[0]
			repo := repos[0]
			switch mode {
			case "ref":
				i.TargetRef = "refs/heads/missing"
			case "source-ref":
				i.SourceBranch = "refs/heads/missing"
			case "source":
				i.CandidateSHA = observations[1].CandidateSHA
			case "base":
				i.BaseSHA = observations[1].CandidateSHA
			case "sha":
				i.FinalSHA = i.BaseSHA
			case "path":
				repo = filepath.Join(repo, "..", "lib")
			}
			if gitworkspace.ObserveIntegration(ctx, repo, i, true) == nil {
				t.Fatal("wrong target facts ready")
			}
		})
	}
	git(t, repos[0], "update-ref", observations[0].TargetRef, observations[0].BaseSHA)
	if gitworkspace.ObserveIntegration(ctx, repos[0], observations[0], true) == nil {
		t.Fatal("stale target ready")
	}
	if err := gitworkspace.ObserveIntegration(ctx, repos[0], observations[0], false); err != nil {
		t.Fatalf("historical target must not require current tip: %v", err)
	}
}

func TestIntegrationWorkspacePlanningAndTargetCollision(t *testing.T) {
	ctx := context.Background()
	root := canonicalTempDir(t)
	f := initRepository(t, filepath.Join(root, "repository"))
	worktrees := filepath.Join(root, "worktrees")
	g, err := gitworkspace.NewIntegrator(ctx, f.repository, worktrees)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(worktrees); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("planning created workspace root")
	}
	base, err := g.ObserveBase(ctx)
	if err != nil || base != f.revision {
		t.Fatalf("base=%s err=%v", base, err)
	}
	writeFile(t, filepath.Join(f.repository, "uncommitted"), "retain me")
	if _, err := g.ObserveBase(ctx); !errors.Is(err, gitworkspace.ErrForeignMutation) {
		t.Fatalf("dirty base accepted: %v", err)
	}
	commitAll(t, f.repository, "next base")
	next, err := g.ObserveBase(ctx)
	if err != nil || next == base {
		t.Fatalf("changed base not observed: %s %v", next, err)
	}
	target := "refs/heads/axiom/private-target"
	if err := g.PrepareTarget(ctx, target, base); err != nil {
		t.Fatal(err)
	}
	if err := g.PrepareTarget(ctx, target, next); !errors.Is(err, gitworkspace.ErrWorkspaceCollision) {
		t.Fatalf("target adopted: %v", err)
	}
	if got := git(t, f.repository, "rev-parse", target); got != base {
		t.Fatalf("collision moved target: %s", got)
	}
	path, err := g.MaterializeWorkspace(ctx, "attempt-a", "refs/heads/axiom/attempt-a", base)
	if err != nil || path != filepath.Join(worktrees, "attempt-a") {
		t.Fatalf("materialize %s %v", path, err)
	}
	if _, err := g.MaterializeWorkspace(ctx, "attempt-a", "refs/heads/axiom/attempt-a", base); !errors.Is(err, gitworkspace.ErrWorkspaceCollision) {
		t.Fatalf("workspace adopted: %v", err)
	}
}
