package gitworkspace_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
)

var errFenceLost = errors.New("fence lost")

func dirtyWorkspace(t *testing.T, workspace string) {
	t.Helper()
	writeFile(t, filepath.Join(workspace, "a.txt"), "attempt one\n")
	writeFile(t, filepath.Join(workspace, "README.md"), "modified\n")
	writeFile(t, filepath.Join(workspace, "nested", "deep", "untracked.txt"), "untracked\n")
	writeFile(t, filepath.Join(workspace, ".gitignore"), "ignored.log\n")
	writeFile(t, filepath.Join(workspace, "ignored.log"), "ignored\n")
}

func countingGuard(failAt int) (func(context.Context) error, *int) {
	calls := 0
	return func(context.Context) error {
		calls++
		if calls == failAt {
			return errFenceLost
		}
		return nil
	}, &calls
}

func TestResetWorkspaceRestoresBaseUnderGuard(t *testing.T) {
	fixture := newGitFixture(t, graphOptions{})
	fixture.prepare(t)
	child, sibling := fixture.child("a"), fixture.child("b")
	dirtyWorkspace(t, child.Envelope.Workspace)
	writeFile(t, filepath.Join(sibling.Envelope.Workspace, "b.txt"), "sibling\n")
	outside := filepath.Join(filepath.Dir(fixture.root), "outside")
	writeFile(t, filepath.Join(outside, "kept.txt"), "outside\n")
	if err := os.Symlink(outside, filepath.Join(child.Envelope.Workspace, "link")); err != nil {
		t.Fatal(err)
	}
	shared := fixture.sharedState(t)
	guard, calls := countingGuard(0)
	if err := fixture.manager.ResetWorkspace(context.Background(), child, guard); err != nil {
		t.Fatal(err)
	}
	if *calls != 3 {
		t.Fatalf("guard calls=%d", *calls)
	}
	if err := fixture.manager.ValidateWorkspace(context.Background(), child); err != nil {
		t.Fatalf("reset left changes: %v", err)
	}
	if got := readFile(t, filepath.Join(child.Envelope.Workspace, "README.md")); got != "base\n" {
		t.Fatalf("tracked file not restored: %q", got)
	}
	for _, name := range []string{"a.txt", "nested", "ignored.log", ".gitignore", "link"} {
		if _, err := os.Lstat(filepath.Join(child.Envelope.Workspace, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s survived reset: %v", name, err)
		}
	}
	if got := readFile(t, filepath.Join(outside, "kept.txt")); got != "outside\n" {
		t.Fatal("reset followed a symlink out of the worktree")
	}
	if got := readFile(t, filepath.Join(sibling.Envelope.Workspace, "b.txt")); got != "sibling\n" {
		t.Fatal("reset touched a sibling worktree")
	}
	if fixture.sharedState(t) != shared {
		t.Fatal("reset touched the shared checkout")
	}
	again, _ := countingGuard(0)
	if err := fixture.manager.ResetWorkspace(context.Background(), child, again); err != nil {
		t.Fatalf("reset of a clean worktree is not repeatable: %v", err)
	}
}

func TestResetWorkspaceFencingFailures(t *testing.T) {
	t.Run("missing guard", func(t *testing.T) {
		fixture := newGitFixture(t, graphOptions{})
		fixture.prepare(t)
		child := fixture.child("a")
		dirtyWorkspace(t, child.Envelope.Workspace)
		if err := fixture.manager.ResetWorkspace(context.Background(), child, nil); !errors.Is(err, gitworkspace.ErrRetryResetFencingUnavailable) {
			t.Fatal(err)
		}
		if got := readFile(t, filepath.Join(child.Envelope.Workspace, "a.txt")); got != "attempt one\n" {
			t.Fatal("refused reset changed the worktree")
		}
		var none *gitworkspace.Manager
		if err := none.ResetWorkspace(context.Background(), child, func(context.Context) error { return nil }); !errors.Is(err, gitworkspace.ErrRetryResetFencingUnavailable) {
			t.Fatal(err)
		}
	})
	t.Run("fence lost before the first effect", func(t *testing.T) {
		fixture := newGitFixture(t, graphOptions{})
		fixture.prepare(t)
		child := fixture.child("a")
		dirtyWorkspace(t, child.Envelope.Workspace)
		guard, calls := countingGuard(1)
		if err := fixture.manager.ResetWorkspace(context.Background(), child, guard); !errors.Is(err, errFenceLost) || *calls != 1 {
			t.Fatalf("err=%v calls=%d", err, *calls)
		}
		if readFile(t, filepath.Join(child.Envelope.Workspace, "README.md")) != "modified\n" || readFile(t, filepath.Join(child.Envelope.Workspace, "a.txt")) != "attempt one\n" {
			t.Fatal("effect ran without its fence")
		}
	})
	t.Run("fence lost between effects then recovered", func(t *testing.T) {
		fixture := newGitFixture(t, graphOptions{})
		fixture.prepare(t)
		child := fixture.child("a")
		dirtyWorkspace(t, child.Envelope.Workspace)
		guard, calls := countingGuard(2)
		if err := fixture.manager.ResetWorkspace(context.Background(), child, guard); !errors.Is(err, errFenceLost) || *calls != 2 {
			t.Fatalf("err=%v calls=%d", err, *calls)
		}
		if readFile(t, filepath.Join(child.Envelope.Workspace, "README.md")) != "base\n" || readFile(t, filepath.Join(child.Envelope.Workspace, "a.txt")) != "attempt one\n" {
			t.Fatal("partial reset did not stop at the lost fence")
		}
		if err := fixture.manager.ValidateWorkspace(context.Background(), child); !errors.Is(err, gitworkspace.ErrForeignMutation) {
			t.Fatalf("partial reset reported clean: %v", err)
		}
		recovered, _ := countingGuard(0)
		if err := fixture.manager.ResetWorkspace(context.Background(), child, recovered); err != nil {
			t.Fatal(err)
		}
		if err := fixture.manager.ValidateWorkspace(context.Background(), child); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fence lost after the last effect", func(t *testing.T) {
		fixture := newGitFixture(t, graphOptions{})
		fixture.prepare(t)
		child := fixture.child("a")
		dirtyWorkspace(t, child.Envelope.Workspace)
		guard, _ := countingGuard(3)
		if err := fixture.manager.ResetWorkspace(context.Background(), child, guard); !errors.Is(err, errFenceLost) {
			t.Fatal(err)
		}
	})
}

func TestResetWorkspaceRefusesForeignOrEscapedWorktrees(t *testing.T) {
	cases := map[string]func(t *testing.T, f *gitFixture, child executiongraph.ChildExecution) error{
		"foreign head": func(t *testing.T, f *gitFixture, child executiongraph.ChildExecution) error {
			git(t, child.Envelope.Workspace, "add", "a.txt")
			git(t, child.Envelope.Workspace, "-c", "user.name=Axiom Test", "-c", "user.email=axiom@example.invalid", "commit", "-m", "foreign")
			return gitworkspace.ErrWorkspaceInvalid
		},
		"tampered owner record": func(t *testing.T, f *gitFixture, child executiongraph.ChildExecution) error {
			owner := filepath.Join(f.root, ".axiom-workspace-owners", child.ExecutionID+".json")
			wire, err := os.ReadFile(owner)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(owner, append(wire[:len(wire)-2], []byte(",\"extra\":1}\n")...), 0o600); err != nil {
				t.Fatal(err)
			}
			return gitworkspace.ErrWorkspaceInvalid
		},
		"workspace replaced by symlink": func(t *testing.T, f *gitFixture, child executiongraph.ChildExecution) error {
			moved := filepath.Join(filepath.Dir(f.root), "moved")
			if err := os.Rename(child.Envelope.Workspace, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(moved, child.Envelope.Workspace); err != nil {
				t.Fatal(err)
			}
			return gitworkspace.ErrWorkspaceInvalid
		},
		"executable filter configuration": func(t *testing.T, f *gitFixture, child executiongraph.ChildExecution) error {
			git(t, f.repository, "config", "filter.sentinel.smudge", "/usr/bin/true")
			return gitworkspace.ErrExecutableGitConfiguration
		},
	}
	for name, tamper := range cases {
		t.Run(name, func(t *testing.T) {
			fixture := newGitFixture(t, graphOptions{})
			fixture.prepare(t)
			child := fixture.child("a")
			dirtyWorkspace(t, child.Envelope.Workspace)
			want := tamper(t, fixture, child)
			resolved, err := filepath.EvalSymlinks(child.Envelope.Workspace)
			if err != nil {
				t.Fatal(err)
			}
			guard, calls := countingGuard(0)
			if err := fixture.manager.ResetWorkspace(context.Background(), child, guard); !errors.Is(err, want) || *calls != 1 {
				t.Fatalf("err=%v calls=%d", err, *calls)
			}
			if readFile(t, filepath.Join(resolved, "README.md")) != "modified\n" {
				t.Fatal("refused reset changed the worktree")
			}
			if _, err := os.Lstat(filepath.Join(resolved, "nested", "deep", "untracked.txt")); err != nil {
				t.Fatal("refused reset removed untracked content")
			}
		})
	}
}
