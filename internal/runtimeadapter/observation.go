package runtimeadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const maxExecutableBytes = 1 << 30

var ErrExecutableUnavailable = runtimeprofile.ErrExecutableUnavailable

// ExecutableIdentity is a presentation-safe identity of the regular executable
// file that path currently resolves to. It binds the resolved path and the
// exact content, so removal, replacement or another target fails or changes
// it. It never executes the file and never exposes the path.
func ExecutableIdentity(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", ErrExecutableUnavailable
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", ErrExecutableUnavailable
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() || !executableMode(info) || info.Size() > maxExecutableBytes {
		return "", ErrExecutableUnavailable
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", ErrExecutableUnavailable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return "", ErrExecutableUnavailable
	}
	content := sha256.New()
	read, err := io.Copy(content, io.LimitReader(file, maxExecutableBytes+1))
	if err != nil || read != opened.Size() || read > maxExecutableBytes {
		return "", ErrExecutableUnavailable
	}
	identity := sha256.New()
	identity.Write([]byte("axiom-runtime-executable/v1\x00" + resolved + "\x00"))
	identity.Write(content.Sum(nil))
	return hex.EncodeToString(identity.Sum(nil)), nil
}

func executableMode(info os.FileInfo) bool {
	return info.Mode().Perm()&0o111 != 0
}
