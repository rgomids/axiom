//go:build !windows

package local

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Canonicalize only the test fixture: macOS TMPDIR commonly begins with /var,
// which is intentionally refused by the installation API.
func installTestRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(privateTestRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCreateInstallDirectoryPreservesUnsafeExistingMode(t *testing.T) {
	path := filepath.Join(installTestRoot(t), "state")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	chmodOrFatal(t, path, 0o755)
	if directory, err := CreateOwnedDirectory(path); !errors.Is(err, ErrUnsafe) {
		directory.Close()
		t.Fatalf("unsafe existing root accepted: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("root repaired: %v, %v", info, err)
	}
}

func TestCheckInstallDirectoryRefusesUserOwnedSymlink(t *testing.T) {
	base := installTestRoot(t)
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{link, filepath.Join(link, "missing", "state")} {
		for _, private := range []bool{true, false} {
			if err := CheckInstallDirectory(path, private); !errors.Is(err, ErrUnsafe) {
				t.Fatalf("symlink %q accepted: %v", path, err)
			}
		}
	}
}

func TestCheckInstallDirectoryRefusesRootOwnedSymlink(t *testing.T) {
	var link string
	if os.Geteuid() == 0 {
		base := installTestRoot(t)
		link = filepath.Join(base, "root-link")
		if err := os.Symlink(base, link); err != nil {
			t.Fatal(err)
		}
	} else {
		// macOS exposes system-owned aliases; Linux hosts may expose /var/run.
		// Inspect ownership rather than assume an alias exists on every host.
		for _, path := range []string{"/tmp", "/var", "/etc", "/var/run"} {
			info, err := os.Lstat(path)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				continue
			}
			if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid == 0 {
				link = path
				break
			}
		}
		if link == "" {
			t.Skip("root-owned symlink unavailable without elevated privileges")
		}
	}
	for _, private := range []bool{true, false} {
		if err := CheckInstallDirectory(filepath.Join(link, "axiom-test-missing-root"), private); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("root-owned symlink accepted: %v", err)
		}
	}
}

func TestCreateInstallDirectoryRefusesUnsafeAncestorBeforeMissingLeaf(t *testing.T) {
	ancestor := filepath.Join(installTestRoot(t), "shared")
	if err := os.Mkdir(ancestor, 0o700); err != nil {
		t.Fatal(err)
	}
	chmodOrFatal(t, ancestor, 0o777)
	path := filepath.Join(ancestor, "missing", "state")
	for _, private := range []bool{true, false} {
		if err := CheckInstallDirectory(path, private); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("unsafe ancestor accepted: %v", err)
		}
	}
	if directory, err := CreateOwnedDirectory(path); !errors.Is(err, ErrUnsafe) {
		directory.Close()
		t.Fatalf("unsafe ancestor created: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(ancestor, "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing component created: %v", err)
	}
}

func TestCheckInstallPublicationDirectoryAccepts0755(t *testing.T) {
	path := filepath.Join(installTestRoot(t), "bin")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	chmodOrFatal(t, path, 0o755)
	if err := CheckInstallDirectory(path, false); err != nil {
		t.Fatal(err)
	}
	if err := CheckInstallDirectory(path, true); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("private root accepted 0755: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("publication mode changed: %v, %v", info, err)
	}
}

func TestInstallPublishNewConflictPreservesBothFiles(t *testing.T) {
	path := filepath.Join(installTestRoot(t), "state")
	directory, err := CreateOwnedDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	if err := os.WriteFile(filepath.Join(path, "target"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage, err := directory.Stage(".install-test-", []byte("candidate"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.PublishNew(stage, "target"); !os.IsExist(err) {
		t.Fatalf("conflict = %v", err)
	}
	for name, expected := range map[string]string{"target": "original", stage: "candidate"} {
		wire, err := os.ReadFile(filepath.Join(path, name))
		if err != nil || string(wire) != expected {
			t.Fatalf("%s changed: %q, %v", name, wire, err)
		}
	}
}

func TestInstallPublishNewRefusesControlledRootReplacement(t *testing.T) {
	base := installTestRoot(t)
	path := filepath.Join(base, "state")
	directory, err := CreateOwnedDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	stage, err := directory.Stage(".install-test-", []byte("candidate"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(base, "original")
	if err := os.Rename(path, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := directory.PublishNew(stage, "target"); !errors.Is(err, ErrReplaced) {
		t.Fatalf("replaced root accepted: %v", err)
	}
	for _, root := range []string{path, original} {
		if _, err := os.Lstat(filepath.Join(root, "target")); !os.IsNotExist(err) {
			t.Fatalf("target published in %s: %v", root, err)
		}
	}
	wire, err := os.ReadFile(filepath.Join(original, stage))
	if err != nil || string(wire) != "candidate" {
		t.Fatalf("stage changed: %q, %v", wire, err)
	}
}
