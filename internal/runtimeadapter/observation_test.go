package runtimeadapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, invalid := range []string{"codex", filepath.Join(root, "missing"), root} {
		if _, err := ExecutableIdentity(invalid); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
}
