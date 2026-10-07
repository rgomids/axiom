package local

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"

	"github.com/rgomids/axiom/internal/windowsfs"
)

// FileIdentity mirrors DirectoryIdentity for one regular, non-reparse file.
func FileIdentity(path string) (string, error) {
	clean, err := windowsfs.Canonical(path)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(clean); err != nil || !info.Mode().IsRegular() {
		return "", ErrUnsafe
	}
	f, err := os.Open(clean)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", ErrUnsafe
	}
	id, err := windowsfs.Identity(f)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(strings.ToLower(clean) + "\x00" + id))
	return "file:" + hex.EncodeToString(digest[:]), nil
}
