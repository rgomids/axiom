//go:build unix

package local

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// DirectoryIdentity observes one explicit local directory without following a
// symlink at the selected path. The value binds setup authority and later local
// resolution to the same filesystem object; it is not portable Project intent.
// It hashes the clean path, inode and the object's birth time: st_dev is not
// stable across reboots or volume remounts (APFS on macOS), while path+inode
// alone cannot tell apart objects of two volumes mounted at the same path.
func DirectoryIdentity(path string) (string, error) {
	clean, stat, err := lstatIdentity(path, isDirectory)
	if err != nil {
		return "", err
	}
	return currentIdentity("fs:", clean, stat)
}

// MatchesDirectoryIdentity reports whether the directory at path is the object
// a recorded identity names. Values recorded with st_dev still match while the
// device number is unchanged; any other stale value needs a rebind.
func MatchesDirectoryIdentity(path, recorded string) bool {
	clean, stat, err := lstatIdentity(path, isDirectory)
	return err == nil && matchesIdentity("fs:", clean, stat, recorded)
}

func isDirectory(info os.FileInfo) bool { return info.IsDir() }

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

func currentIdentity(prefix, clean string, stat *syscall.Stat_t) (string, error) {
	birth, err := objectBirth(clean, stat)
	if err != nil {
		return "", err
	}
	return digestIdentity(prefix, clean, strconv.FormatUint(uint64(stat.Ino), 10), birth), nil
}

func legacyIdentity(prefix, clean string, stat *syscall.Stat_t) string {
	return digestIdentity(prefix, clean, strconv.FormatUint(uint64(stat.Dev), 10), strconv.FormatUint(uint64(stat.Ino), 10))
}

func digestIdentity(prefix string, facts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(facts, "\x00")))
	return prefix + hex.EncodeToString(digest[:])
}

func birthFact(sec, nsec int64) string {
	return "btime:" + strconv.FormatInt(sec, 10) + "." + strconv.FormatInt(nsec, 10)
}

func matchesIdentity(prefix, clean string, stat *syscall.Stat_t, recorded string) bool {
	current, err := currentIdentity(prefix, clean, stat)
	return (err == nil && recorded == current) || recorded == legacyIdentity(prefix, clean, stat)
}
