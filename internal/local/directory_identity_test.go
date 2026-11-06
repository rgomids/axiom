//go:build unix

package local

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A reboot or remount can change st_dev for the same directory (APFS on
// macOS). The identity must not depend on it, values recorded by older
// binaries (path+dev+inode) still match while the device is unchanged, and an
// object of another volume at the same path with the same inode (CR-376-F1)
// must not match.
func TestDirectoryIdentityIgnoresDeviceNumberButNotVolume(t *testing.T) {
	dir := t.TempDir()
	recorded, err := DirectoryIdentity(dir)
	if err != nil {
		t.Fatal(err)
	}
	clean, stat, err := lstatIdentity(dir, isDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := DirectoryIdentity(dir); err != nil || again != recorded || !MatchesDirectoryIdentity(dir, recorded) {
		t.Fatal("identity of an unchanged directory is not stable")
	}
	if !MatchesDirectoryIdentity(dir, legacyIdentity("fs:", clean, stat)) {
		t.Fatal("legacy identity with unchanged device does not match")
	}
	remounted := *stat
	remounted.Dev++
	if MatchesDirectoryIdentity(dir, legacyIdentity("fs:", clean, &remounted)) {
		t.Fatal("legacy identity from another device matched")
	}
	ino := strconv.FormatUint(uint64(stat.Ino), 10)
	if MatchesDirectoryIdentity(dir, digestIdentity("fs:", clean, ino)) {
		t.Fatal("path+inode alone matched; another volume could reuse the binding")
	}
	if MatchesDirectoryIdentity(dir, digestIdentity("fs:", clean, ino, birthFact(1, 2))) {
		t.Fatal("an object with the same path and inode but another birth matched")
	}
	if birth, err := objectBirth(clean, &remounted); err != nil {
		t.Fatal(err)
	} else if strings.HasPrefix(birth, "dev:") {
		t.Skip("filesystem reports no birth time; st_dev stands in")
	} else if digestIdentity("fs:", clean, ino, birth) != recorded {
		t.Fatal("identity changed with st_dev")
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
	if MatchesDirectoryIdentity(dir, recorded) {
		t.Fatal("replaced directory matched the old identity")
	}
}

func TestFileIdentityIgnoresDeviceNumberButNotVolume(t *testing.T) {
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
	if !MatchesFileIdentity(path, recorded) || !MatchesFileIdentity(path, legacyIdentity("file:", clean, stat)) {
		t.Fatal("file identity rejects its current or legacy value")
	}
	if MatchesFileIdentity(path, digestIdentity("file:", clean, strconv.FormatUint(uint64(stat.Ino), 10))) {
		t.Fatal("path+inode alone matched a file identity")
	}
}
