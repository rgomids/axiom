//go:build !windows

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

// FileIdentity observes one explicit regular file without following a link at
// the selected path and without reading its content. It binds a portable
// local-file documentation source to the same filesystem object (Issue #231);
// a replaced or relocated file yields a different identity.
func FileIdentity(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("file path is not absolute")
	}
	clean := filepath.Clean(path)
	info, err := os.Lstat(clean)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("file is unavailable")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", errors.New("file identity is unavailable")
	}
	facts := clean + "\x00" + strconv.FormatUint(uint64(stat.Dev), 10) + "\x00" + strconv.FormatUint(uint64(stat.Ino), 10)
	digest := sha256.Sum256([]byte(facts))
	return "file:" + hex.EncodeToString(digest[:]), nil
}
