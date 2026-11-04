package gitworkspace_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/gitworkspace"
)

// filterCommand is executed by Git through its own shell when a driver is
// live. It only creates the sentinel, so its presence proves execution.
func filterCommand(sentinel string) string {
	return "touch '" + sentinel + "'"
}

func assertNoSentinel(t *testing.T, sentinel string) {
	t.Helper()
	if _, err := os.Lstat(sentinel); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("filter helper executed: sentinel stat err=%v", err)
	}
}

func assertSentinel(t *testing.T, sentinel string) {
	t.Helper()
	if _, err := os.Lstat(sentinel); err != nil {
		t.Fatalf("control filter was not live: %v", err)
	}
}

// filteredRepository commits a tracked file routed through the "sentinel"
// filter by attribute and configures the given driver variable.
func filteredRepository(t *testing.T, directory, variable, command string) repositoryFixture {
	t.Helper()
	repository := initRepository(t, directory)
	writeFile(t, filepath.Join(directory, ".gitattributes"), "*.txt filter=sentinel\n")
	git(t, directory, "add", ".gitattributes")
	git(t, directory, "-c", "user.name=Axiom Test", "-c", "user.email=axiom@example.invalid", "commit", "-m", "attributes")
	git(t, directory, "config", "filter.sentinel."+variable, command)
	repository.revision = git(t, directory, "rev-parse", "HEAD")
	repository.tree = git(t, directory, "rev-parse", "HEAD^{tree}")
	return repository
}

// sharedCheckoutState observes the shared checkout without Git commands that
// could run a live filter: HEAD, tree, raw index entries, untracked names and
// the bytes of every tracked file read directly from disk.
func sharedCheckoutState(t *testing.T, repository string) string {
	t.Helper()
	state := git(t, repository, "rev-parse", "HEAD") + git(t, repository, "rev-parse", "HEAD^{tree}") + git(t, repository, "ls-files", "--stage") + git(t, repository, "ls-files", "--others")
	for _, name := range strings.Split(git(t, repository, "ls-files"), "\n") {
		content, err := os.ReadFile(filepath.Join(repository, name))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(content)
		state += name + ":" + hex.EncodeToString(digest[:]) + "\n"
	}
	return state
}

func TestIntegratorRejectsExecutableFilterConfigurationBeforeCheckout(t *testing.T) {
	for _, variable := range []string{"clean", "smudge", "process"} {
		t.Run(variable, func(t *testing.T) {
			root := canonicalTempDir(t)
			controlSentinel := filepath.Join(root, "control-sentinel")
			control := filteredRepository(t, filepath.Join(root, "control"), variable, filterCommand(controlSentinel))
			if variable == "clean" {
				writeFile(t, filepath.Join(control.repository, "shared.txt"), "changed\n")
				_, _ = gitAllowFailure(control.repository, "add", "shared.txt")
			} else {
				_, _ = gitAllowFailure(control.repository, "worktree", "add", "--detach", filepath.Join(root, "control-worktree"), control.revision)
			}
			assertSentinel(t, controlSentinel)
			sentinel := filepath.Join(root, "sentinel")
			repository := filteredRepository(t, filepath.Join(root, "repository"), variable, filterCommand(sentinel))
			before := sharedCheckoutState(t, repository.repository)
			_, err := gitworkspace.NewIntegrator(context.Background(), repository.repository, filepath.Join(root, "workspaces"))
			if !errors.Is(err, gitworkspace.ErrExecutableGitConfiguration) {
				t.Fatal(err)
			}
			assertNoSentinel(t, sentinel)
			if after := sharedCheckoutState(t, repository.repository); after != before {
				t.Fatal("shared checkout changed")
			}
		})
	}
}

func TestIntegratorRejectsFilterConfigurationIntroducedAfterPreparation(t *testing.T) {
	for _, kind := range []string{"repository", "worktree", "external-diff", "disabled"} {
		t.Run(kind, func(t *testing.T) {
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
			before := sharedCheckoutState(t, repository.repository)
			sentinel := filepath.Join(root, "sentinel")
			switch kind {
			case "repository":
				common := git(t, repository.repository, "rev-parse", "--path-format=absolute", "--git-common-dir")
				writeFile(t, filepath.Join(common, "info", "attributes"), "*.txt filter=sentinel\n")
				git(t, observation.Path, "config", "filter.sentinel.clean", filterCommand(sentinel))
			case "worktree":
				git(t, observation.Path, "config", "extensions.worktreeConfig", "true")
				git(t, observation.Path, "config", "--worktree", "filter.sentinel.process", filterCommand(sentinel))
			case "external-diff":
				git(t, repository.repository, "config", "diff.external", "/usr/bin/true")
			case "disabled":
				git(t, repository.repository, "config", "filter.lfs.smudge", "")
			}
			_, err = boundary.ObserveCandidate(ctx, "agent", "refs/heads/agent", repository.revision)
			if kind == "disabled" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, gitworkspace.ErrExecutableGitConfiguration) {
				t.Fatal(err)
			}
			assertNoSentinel(t, sentinel)
			if after := sharedCheckoutState(t, repository.repository); after != before {
				t.Fatal("shared checkout changed")
			}
		})
	}
}

func TestIntegratorRequiresPrivateWorkspaceRoot(t *testing.T) {
	for _, mode := range []os.FileMode{0700, 0755, 0770, 0701} {
		t.Run(mode.String(), func(t *testing.T) {
			root := canonicalTempDir(t)
			repository := initRepository(t, filepath.Join(root, "repository"))
			workspaces := filepath.Join(root, "workspaces")
			if err := os.Mkdir(workspaces, mode); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(workspaces, mode); err != nil {
				t.Fatal(err)
			}
			_, err := gitworkspace.NewIntegrator(context.Background(), repository.repository, workspaces)
			if mode == 0700 {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, gitworkspace.ErrInvalidConfiguration) {
				t.Fatal(err)
			}
		})
	}
	t.Run("foreign ownership", func(t *testing.T) {
		gitworkspace.SetEffectiveUID(t, os.Geteuid()+1)
		root := canonicalTempDir(t)
		repository := initRepository(t, filepath.Join(root, "repository"))
		_, err := gitworkspace.NewIntegrator(context.Background(), repository.repository, filepath.Join(root, "workspaces"))
		if !errors.Is(err, gitworkspace.ErrInvalidConfiguration) {
			t.Fatal(err)
		}
	})
	t.Run("symlink root", func(t *testing.T) {
		root := canonicalTempDir(t)
		repository := initRepository(t, filepath.Join(root, "repository"))
		target, workspaces := filepath.Join(root, "target"), filepath.Join(root, "workspaces")
		if err := os.Mkdir(target, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, workspaces); err != nil {
			t.Fatal(err)
		}
		_, err := gitworkspace.NewIntegrator(context.Background(), repository.repository, workspaces)
		if !errors.Is(err, gitworkspace.ErrInvalidConfiguration) {
			t.Fatal(err)
		}
	})
}
