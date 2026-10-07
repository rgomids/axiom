package runtimeadapter

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func writeStub(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n# "+content+"\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestExecutableIdentityBindsPathAndContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "codex")
	writeStub(t, path, "original")
	original, err := ExecutableIdentity(path)
	if err != nil || len(original) != 64 || strings.Contains(original, root) {
		t.Fatalf("identity=%q err=%v", original, err)
	}
	if again, err := ExecutableIdentity(path); err != nil || again != original {
		t.Fatal("identity is not deterministic")
	}
	other := filepath.Join(t.TempDir(), "codex")
	writeStub(t, other, "original")
	if identity, err := ExecutableIdentity(other); err != nil || identity == original {
		t.Fatal("same basename at another path shares an identity")
	}
	writeStub(t, path, "replaced")
	if identity, err := ExecutableIdentity(path); err != nil || identity == original {
		t.Fatal("replaced content kept its identity")
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(t.TempDir(), "codex")
		if err := os.Symlink(path, link); err != nil {
			t.Fatal(err)
		}
		target, _ := ExecutableIdentity(path)
		if identity, err := ExecutableIdentity(link); err != nil || identity != target {
			t.Fatal("a link does not resolve to its target executable")
		}
		plain := filepath.Join(t.TempDir(), "codex")
		if err := os.WriteFile(plain, []byte("not executable"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ExecutableIdentity(plain); err == nil {
			t.Fatal("non-executable file accepted")
		}
	}
	for _, invalid := range []string{"codex", filepath.Join(root, "missing"), root} {
		if _, err := ExecutableIdentity(invalid); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
}

type readyIntegration bool

func (r readyIntegration) IntegrationReady(context.Context) bool { return bool(r) }

func TestExecutableObserverProvesOnlyWhatLingoVerifies(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), id)
			writeStub(t, path, "observed")
			identity, err := ExecutableIdentity(path)
			if err != nil {
				t.Fatal(err)
			}
			observedAt := time.Unix(10, 0)
			observer, err := NewExecutableObserver(3, observedAt, []ObservedRuntime{{ID: id, Executable: path, Integration: readyIntegration(true)}})
			if err != nil {
				t.Fatal(err)
			}
			observation, err := observer.Observe(context.Background(), id)
			if err != nil || !observation.Installed || !observation.Available || observation.ExecutableDigest != identity || observation.Revision != 3 || observation.Version != "" {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
			if len(observation.CapabilityStatus) != 1 || observation.CapabilityStatus[IntegrationCapability] != runtimeprofile.CapabilityProven {
				t.Fatalf("capabilities=%v", observation.CapabilityStatus)
			}
			unverified, _ := NewExecutableObserver(3, observedAt, []ObservedRuntime{{ID: id, Executable: path, Integration: readyIntegration(false)}})
			if observation, _ := unverified.Observe(context.Background(), id); len(observation.CapabilityStatus) != 0 || !observation.Installed {
				t.Fatalf("unverified integration proved %v", observation.CapabilityStatus)
			}
			for _, absent := range []string{"", filepath.Join(t.TempDir(), id)} {
				missing, _ := NewExecutableObserver(3, observedAt, []ObservedRuntime{{ID: id, Executable: absent, Integration: readyIntegration(true)}})
				if observation, err := missing.Observe(context.Background(), id); err != nil || observation.Installed || observation.Available || observation.ExecutableDigest != "" || len(observation.CapabilityStatus) != 0 {
					t.Fatalf("absent executable observed as %+v err=%v", observation, err)
				}
			}
			if _, err := observer.Observe(context.Background(), "other"); err == nil {
				t.Fatal("unconfigured Runtime observed")
			}
		})
	}
	for _, invalid := range [][]ObservedRuntime{{{ID: "other"}}, {{ID: "codex", Executable: "codex"}}, {{ID: "codex"}, {ID: "codex"}}} {
		if _, err := NewExecutableObserver(1, time.Unix(1, 0), invalid); err == nil {
			t.Fatalf("accepted %+v", invalid)
		}
	}
	if _, err := NewExecutableObserver(0, time.Unix(1, 0), nil); err == nil {
		t.Fatal("accepted zero revision")
	}
}
