package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
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

func TestPortableConflictingWritersHaveOneWinner(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), "sample", []byte("old")); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, next := range []string{"first", "second"} {
		workers.Add(1)
		go func(next string) {
			defer workers.Done()
			<-start
			results <- store.Update(context.Background(), "sample", []byte("old"), []byte(next))
		}(next)
	}
	close(start)
	workers.Wait()
	close(results)
	success, conflict := 0, 0
	for result := range results {
		switch {
		case result == nil:
			success++
		case errors.Is(result, ErrConflict):
			conflict++
		default:
			t.Fatalf("unexpected writer result: %v", result)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("writer results: success=%d conflict=%d", success, conflict)
	}
	actual, err := store.Read(context.Background(), "sample")
	if err != nil || string(actual) != "first" && string(actual) != "second" {
		t.Fatalf("winner bytes = %q, %v", actual, err)
	}
}

func TestPortableCreateRejectsAncestorReplacementBeforePublication(t *testing.T) {
	parent := privateTestRoot(t)
	root := filepath.Join(parent, "projects")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.beforeCreatePublication = func() {
		if err := os.Rename(root, filepath.Join(parent, "moved")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, root); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Create(context.Background(), "sample", []byte("manifest")); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("ancestor replacement = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(outside, "sample")); !os.IsNotExist(err) {
		t.Fatalf("outside target changed: %v", err)
	}
}

func TestPortableUpdateRejectsProjectRenameBeforePublication(t *testing.T) {
	root := privateTestRoot(t)
	store, err := NewPortableStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), "sample", []byte("old")); err != nil {
		t.Fatal(err)
	}
	outside := privateTestRoot(t)
	store.beforeUpdatePublication = func() {
		if err := os.Rename(filepath.Join(root, "sample"), filepath.Join(root, "moved")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(root, "sample")); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Update(context.Background(), "sample", []byte("old"), []byte("new")); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("project replacement = %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(root, "moved", manifestName)); err != nil || string(body) != "old" {
		t.Fatalf("prior bytes = %q, %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(outside, manifestName)); !os.IsNotExist(err) {
		t.Fatalf("outside target changed: %v", err)
	}
}

func TestPortablePostPublicationSyncFailureRequiresRecovery(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			root := privateTestRoot(t)
			store, err := NewPortableStore(root)
			if err != nil {
				t.Fatal(err)
			}
			if operation == "update" {
				if err := store.Create(context.Background(), "sample", []byte("old")); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			store.syncDirectory = func(directory *os.Root) error {
				calls++
				if operation == "update" || calls == 2 {
					return syscall.EIO
				}
				return syncRoot(directory)
			}
			if operation == "create" {
				err = store.Create(context.Background(), "sample", []byte("new"))
			} else {
				err = store.Update(context.Background(), "sample", []byte("old"), []byte("new"))
			}
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("sync failure = %v", err)
			}
			if _, err := store.Read(context.Background(), "sample"); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("read after uncertain durability = %v", err)
			}
			actual, err := os.ReadFile(filepath.Join(root, "sample", manifestName))
			if err != nil || string(actual) != "new" {
				t.Fatalf("published bytes = %q, %v", actual, err)
			}
		})
	}
}
