package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPortableStoreRejectsSymlinkProjectDirectory(t *testing.T) {
	root := t.TempDir()
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
	root := t.TempDir()
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
