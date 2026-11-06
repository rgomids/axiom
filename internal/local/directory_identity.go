//go:build unix

package local

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// DirectoryIdentity observes one explicit local directory without following a
// symlink at the selected path. The value binds setup authority and later local
// resolution to the same filesystem object; it is not portable Project intent.
// It hashes the clean path and inode only: st_dev is not stable across reboots
// or volume remounts (APFS on macOS), so including it made every binding
// spuriously "repository_unavailable" after a restart.
func DirectoryIdentity(path string) (string, error) {
	clean, stat, err := lstatIdentity(path, func(info os.FileInfo) bool { return info.IsDir() })
	if err != nil {
		return "", err
	}
	return identityValue("fs:", clean, stat, false), nil
}

// MatchesDirectoryIdentity reports whether the directory at path is the object
// a recorded identity names. Values recorded before st_dev was dropped still
// match while the device number is unchanged; a stale one needs a rebind.
func MatchesDirectoryIdentity(path, recorded string) bool {
	clean, stat, err := lstatIdentity(path, func(info os.FileInfo) bool { return info.IsDir() })
	return err == nil && matchesIdentity("fs:", clean, stat, recorded)
}

func lstatIdentity(path string, kind func(os.FileInfo) bool) (string, *syscall.Stat_t, error) {
	if !filepath.IsAbs(path) {
		return "", nil, errors.New("path is not absolute")
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err != nil || !kind(info) || info.Mode()&os.ModeSymlink != 0 {
		return "", nil, errors.New("path is unavailable")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", nil, errors.New("identity is unavailable")
	}
	return clean, stat, nil
}

func identityValue(prefix, clean string, stat *syscall.Stat_t, legacy bool) string {
	facts := clean + "\x00"
	if legacy {
		facts += strconv.FormatUint(uint64(stat.Dev), 10) + "\x00"
	}
	facts += strconv.FormatUint(uint64(stat.Ino), 10)
	digest := sha256.Sum256([]byte(facts))
	return prefix + hex.EncodeToString(digest[:])
}

func matchesIdentity(prefix, clean string, stat *syscall.Stat_t, recorded string) bool {
	return recorded == identityValue(prefix, clean, stat, false) || recorded == identityValue(prefix, clean, stat, true)
}
