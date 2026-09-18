package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func privateTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPortableStoreRejectsSymlinkProjectDirectory(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "sample")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("symlink read error = %v", err)
	}
}

func TestPortableStoreRejectsExtraArtifactsBeforeUpdate(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "sample")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, manifestName), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "unknown"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(context.Background(), "sample", []byte("old"), []byte("new")); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("update error = %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(target, manifestName)); err != nil || string(body) != "old" {
		t.Fatalf("unsafe update changed manifest: %q, %v", body, err)
	}
}

func TestPortableStoreRejectsUserSymlinkAncestor(t *testing.T) {
	outside := t.TempDir()
	link := filepath.Join(t.TempDir(), "redirect")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPortableStore(filepath.Join(link, "projects")); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("symlink ancestor accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "projects")); !os.IsNotExist(err) {
		t.Fatalf("outside tree changed: %v", err)
	}
}

func TestPortableStoreRejectsHardLinkedManifest(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sample"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(outside, filepath.Join(root, "sample", manifestName)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("hard-linked manifest accepted: %v", err)
	}
}

func TestPortableStoreReportsInterruptedUpdate(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(root, "sample")
	if err := os.Mkdir(projectPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, manifestName), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, ".lingo-manifest-interrupted"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("interrupted update = %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(projectPath, manifestName)); err != nil || string(data) != "old" {
		t.Fatalf("authoritative manifest changed: %q, %v", data, err)
	}
}

func TestPortableStoreReportsInterruptedCreate(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".lingo-stage-sample-interrupted"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("interrupted create = %v", err)
	}
}

func TestPortableUpdatePermissionFailurePreservesOldBytes(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission denial requires an unprivileged account")
	}
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), "sample", []byte("old")); err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(root, "sample")
	if err := os.Chmod(projectPath, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(projectPath, 0o700)
	if err := store.Update(context.Background(), "sample", []byte("old"), []byte("new")); err == nil {
		t.Fatal("read-only project directory accepted update")
	}
	if data, err := os.ReadFile(filepath.Join(projectPath, manifestName)); err != nil || string(data) != "old" {
		t.Fatalf("prior bytes changed: %q, %v", data, err)
	}
}

func TestPortableUpdateCancellationBeforePublicationPreservesOldBytes(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), "sample", []byte("old")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	store.beforeUpdatePublication = cancel
	if err := store.Update(ctx, "sample", []byte("old"), []byte("new")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled update = %v", err)
	}
	data, err := store.Read(context.Background(), "sample")
	if err != nil || string(data) != "old" {
		t.Fatalf("authoritative manifest after cancellation = %q, %v", data, err)
	}
}
