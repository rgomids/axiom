package local

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// ReadEffortDeclarationInput reads one operator-supplied EffortDeclaration
// file (Specification 007 BI-3.1) exactly once: a regular file reached through
// no symlink other than the trusted root-owned system aliases, opened without
// following its leaf and bounded by the configuration size limit. The host
// path is input only; it is never persisted or reported.
func ReadEffortDeclarationInput(path string) ([]byte, error) {
	if path == "" || strings.ContainsRune(path, 0) {
		return nil, ErrUnsafe
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, ErrUnsafe
	}
	canonical, err := trustedCanonical(absolute)
	if err != nil {
		return nil, ErrUnsafe
	}
	current := volumeRoot(canonical)
	for _, part := range pathComponents(canonical) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil, ErrUnsafe
		}
	}
	// Special files are refused before open, so a FIFO cannot block the read.
	if leaf, err := os.Lstat(canonical); err != nil || !leaf.Mode().IsRegular() {
		return nil, ErrUnsafe
	}
	root, err := os.OpenRoot(filepath.Dir(canonical))
	if err != nil {
		return nil, ErrUnsafe
	}
	defer root.Close()
	name := filepath.Base(canonical)
	file, err := root.OpenFile(name, os.O_RDONLY|noFollow, 0)
	if err != nil {
		return nil, ErrUnsafe
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > runtimeprofile.MaxConfigurationBytes {
		return nil, ErrUnsafe
	}
	visible, err := root.Lstat(name)
	if err != nil || !visible.Mode().IsRegular() || !os.SameFile(info, visible) {
		return nil, ErrUnsafe
	}
	wire, err := io.ReadAll(io.LimitReader(file, runtimeprofile.MaxConfigurationBytes+1))
	if err != nil || len(wire) > runtimeprofile.MaxConfigurationBytes {
		return nil, ErrUnsafe
	}
	return wire, nil
}
