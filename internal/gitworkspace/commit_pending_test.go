package gitworkspace_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/gitworkspace"
)

// A sandboxed writer that cannot commit leaves its changes in the worktree;
// the principal records them as one commit on exactly the agent branch.
func TestIntegratorCommitPendingRecordsUncommittedWriterChanges(t *testing.T) {
	ctx := context.Background()
	root := canonicalTempDir(t)
	repository := initRepository(t, filepath.Join(root, "repository"))
	boundary, err := gitworkspace.NewIntegrator(ctx, repository.repository, filepath.Join(root, "workspaces"))
	if err != nil {
		t.Fatal(err)
	}
	observation, err := boundary.PrepareAgentWorktree(ctx, "agent", "refs/heads/agent", repository.revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := boundary.ObserveCandidate(ctx, "agent", "refs/heads/agent", repository.revision); !errors.Is(err, gitworkspace.ErrNoResult) {
		t.Fatalf("clean worktree before work: %v", err)
	}
	before := sharedCheckoutState(t, repository.repository)
	if err := boundary.CapturePending(ctx, "agent", "refs/heads/agent", repository.revision, "record", func(string) error { return nil }, func(string) error { return nil }, func() error { return nil }); err != nil {
		t.Fatalf("clean worktree is a no-op: %v", err)
	}
	if head := git(t, repository.repository, "rev-parse", "refs/heads/agent"); head != repository.revision {
		t.Fatal("no-op moved the branch")
	}
	writeFile(t, filepath.Join(observation.Path, "new.txt"), "new\n")
	if _, err := boundary.ObserveCandidate(ctx, "agent", "refs/heads/agent", repository.revision); !errors.Is(err, gitworkspace.ErrForeignMutation) {
		t.Fatalf("dirty worktree must not observe: %v", err)
	}
	if err := boundary.CapturePending(ctx, "agent", "refs/heads/agent", repository.revision, "record", func(string) error { return nil }, func(string) error { return nil }, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	candidate, err := boundary.ObserveCandidate(ctx, "agent", "refs/heads/agent", repository.revision)
	if err != nil {
		t.Fatal(err)
	}
	if parent := git(t, repository.repository, "rev-parse", candidate+"^"); parent != repository.revision {
		t.Fatalf("parent %s", parent)
	}
	if files := git(t, repository.repository, "show", "--name-only", "--format=%an", candidate); !strings.Contains(files, "axiom-principal") || !strings.Contains(files, "new.txt") {
		t.Fatalf("commit %q", files)
	}
	if err := boundary.CapturePending(ctx, "agent", "refs/heads/agent", repository.revision, "record", func(string) error { return nil }, func(string) error { return nil }, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if again := git(t, repository.repository, "rev-parse", "refs/heads/agent"); again != candidate {
		t.Fatal("rerun was not idempotent")
	}
	if after := sharedCheckoutState(t, repository.repository); after != before {
		t.Fatal("shared checkout changed")
	}
}

func TestIntegratorCommitPendingRefusesExecutableFilterAndForeignBranch(t *testing.T) {
	ctx := context.Background()
	root := canonicalTempDir(t)
	repository := initRepository(t, filepath.Join(root, "repository"))
	boundary, err := gitworkspace.NewIntegrator(ctx, repository.repository, filepath.Join(root, "workspaces"))
	if err != nil {
		t.Fatal(err)
	}
	observation, err := boundary.PrepareAgentWorktree(ctx, "agent", "refs/heads/agent", repository.revision)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(observation.Path, "a.txt"), "a\n")
	if err := boundary.CapturePending(ctx, "agent", "refs/heads/other", repository.revision, "record", func(string) error { return nil }, func(string) error { return nil }, func() error { return nil }); !errors.Is(err, gitworkspace.ErrWorkspaceInvalid) {
		t.Fatalf("foreign branch: %v", err)
	}
	sentinel := filepath.Join(root, "sentinel")
	common := git(t, repository.repository, "rev-parse", "--path-format=absolute", "--git-common-dir")
	writeFile(t, filepath.Join(common, "info", "attributes"), "*.txt filter=sentinel\n")
	git(t, observation.Path, "config", "filter.sentinel.clean", filterCommand(sentinel))
	if err := boundary.CapturePending(ctx, "agent", "refs/heads/agent", repository.revision, "record", func(string) error { return nil }, func(string) error { return nil }, func() error { return nil }); !errors.Is(err, gitworkspace.ErrExecutableGitConfiguration) {
		t.Fatalf("filter: %v", err)
	}
	assertNoSentinel(t, sentinel)
	if head := git(t, repository.repository, "rev-parse", "refs/heads/agent"); head != repository.revision {
		t.Fatal("refused capture moved the branch")
	}
}

// A writer that committed itself keeps the old contract: leftovers are never
// adopted and the candidate is refused.
func TestIntegratorCommitPendingLeavesSelfCommittedWriterUntouched(t *testing.T) {
	ctx := context.Background()
	root := canonicalTempDir(t)
	repository := initRepository(t, filepath.Join(root, "repository"))
	boundary, err := gitworkspace.NewIntegrator(ctx, repository.repository, filepath.Join(root, "workspaces"))
	if err != nil {
		t.Fatal(err)
	}
	observation, err := boundary.PrepareAgentWorktree(ctx, "agent", "refs/heads/agent", repository.revision)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(observation.Path, "a.txt"), "a\n")
	commitAll(t, observation.Path, "candidate")
	committed := git(t, repository.repository, "rev-parse", "refs/heads/agent")
	writeFile(t, filepath.Join(observation.Path, "late.txt"), "late\n")
	if err := boundary.CapturePending(ctx, "agent", "refs/heads/agent", repository.revision, "record", func(string) error { return nil }, func(string) error { return nil }, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if head := git(t, repository.repository, "rev-parse", "refs/heads/agent"); head != committed {
		t.Fatal("leftover adopted")
	}
	if _, err := boundary.ObserveCandidate(ctx, "agent", "refs/heads/agent", repository.revision); !errors.Is(err, gitworkspace.ErrForeignMutation) {
		t.Fatalf("leftover must refuse: %v", err)
	}
}
