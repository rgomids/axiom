//go:build unix

package local

import (
	"os"
	"path/filepath"
	"testing"
)

// A reboot or remount can change st_dev for the same directory (APFS on
// macOS). The identity must not depend on it, and values recorded by older
// binaries (path+dev+inode) still match while the device is unchanged.
func TestDirectoryIdentityIgnoresDeviceNumber(t *testing.T) {
	dir := t.TempDir()
	recorded, err := DirectoryIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	clean, stat, err := lstatIdentity(dir, func(info os.FileInfo) bool { return info.IsDir() })
	if err != nil {
		t.Fatal(err)
	}
	remounted := *stat
	remounted.Dev++
	if identityValue("fs:", clean, &remounted, false) != recorded {
		t.Fatal("identity changed with st_dev")
	}
	if !MatchesDirectoryIdentity(dir, recorded) {
		t.Fatal("current identity does not match")
	}
	if !MatchesDirectoryIdentity(dir, identityValue("fs:", clean, stat, true)) {
		t.Fatal("legacy identity with unchanged device does not match")
	}
	if MatchesDirectoryIdentity(dir, identityValue("fs:", clean, &remounted, true)) {
		t.Fatal("legacy identity from another device matched")
	}

	// A replaced directory at the same path is a different object.
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	placeholder := filepath.Join(filepath.Dir(dir), "keep-inode-busy")
	if err := os.Mkdir(placeholder, 0o700); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(placeholder)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if replaced, _ := DirectoryIdentity(dir); replaced == recorded && MatchesDirectoryIdentity(dir, recorded) {
		t.Skip("filesystem reused the inode; replacement not observable here")
	}
	if MatchesDirectoryIdentity(dir, recorded) {
		t.Fatal("replaced directory matched the old identity")
	}
}

func TestFileIdentityIgnoresDeviceNumber(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	recorded, err := FileIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	clean, stat, err := lstatIdentity(path, isRegular)
	if err != nil {
		t.Fatal(err)
	}
	remounted := *stat
	remounted.Dev++
	if identityValue("file:", clean, &remounted, false) != recorded || !MatchesFileIdentity(path, recorded) || !MatchesFileIdentity(path, identityValue("file:", clean, stat, true)) {
		t.Fatal("file identity depends on st_dev or rejects a legacy value")
	}
}
