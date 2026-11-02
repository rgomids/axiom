package gitworkspace_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/gitworkspace"
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
