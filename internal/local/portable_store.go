package local

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

const manifestName = "axiom.yaml"

var (
	ErrNotFound = projectapp.ErrNotFound
	ErrConflict = projectapp.ErrConflict
	ErrUnsafe   = projectapp.ErrUnsafe
)

// PortableStore is a deliberately small filesystem adapter for the POC's
// one-artifact portable Project. It accepts a validated slug, never a raw path.
// Context/policy documents are rejected rather than silently overwritten; their
// complete-artifact protocol is a later POC increment.
type PortableStore struct{ root string }

func NewPortableStore(root string) (PortableStore, error) {
	if !filepath.IsAbs(root) {
		return PortableStore{}, ErrUnsafe
	}
	clean := filepath.Clean(root)
	if err := os.MkdirAll(clean, 0o700); err != nil {
		return PortableStore{}, fmt.Errorf("prepare portable root: %w", err)
	}
	info, err := os.Lstat(clean)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return PortableStore{}, ErrUnsafe
	}
	if err := os.Chmod(clean, 0o700); err != nil {
		return PortableStore{}, fmt.Errorf("protect portable root: %w", err)
	}
	locks := filepath.Join(clean, ".lingo-locks")
	if err := os.Mkdir(locks, 0o700); err != nil && !os.IsExist(err) {
		return PortableStore{}, fmt.Errorf("prepare portable locks: %w", err)
	}
	info, err = os.Lstat(locks)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return PortableStore{}, ErrUnsafe
	}
	return PortableStore{root: clean}, nil
}

// Read returns a complete manifest byte slice or a typed missing result. It
// rejects extra artifacts because this minimal adapter cannot safely preserve a
// multi-artifact Project during update.
func (s PortableStore) Read(ctx context.Context, slug string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result []byte
	err := s.withLock(slug, func(target string) error {
		if err := onlyManifest(target); err != nil {
			return err
		}
		input, err := readRegular(filepath.Join(target, manifestName))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		result = input
		return nil
	})
	return result, err
}

// Create publishes a complete one-file Project directory. The staging directory
// is private; directory rename is the POC create commit point.
func (s PortableStore) Create(ctx context.Context, slug string, manifest []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.withLock(slug, func(target string) error {
		if _, err := os.Lstat(target); err == nil {
			return ErrConflict
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect portable target: %w", err)
		}
		stage, err := os.MkdirTemp(s.root, ".lingo-stage-")
		if err != nil {
			return fmt.Errorf("create private stage: %w", err)
		}
		defer os.RemoveAll(stage)
		if err := os.Chmod(stage, 0o700); err != nil {
			return fmt.Errorf("protect private stage: %w", err)
		}
		if err := writeDurable(filepath.Join(stage, manifestName), manifest); err != nil {
			return err
		}
		if err := syncDirectory(stage); err != nil {
			return err
		}
		if err := os.Rename(stage, target); err != nil {
			return fmt.Errorf("publish portable project: %w", err)
		}
		return syncDirectory(s.root)
	})
}

// Update atomically replaces the sole supported artifact after comparing its
// exact observed bytes under the slug lock. A stale writer reports ErrConflict.
func (s PortableStore) Update(ctx context.Context, slug string, expected, manifest []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.withLock(slug, func(target string) error {
		if err := onlyManifest(target); err != nil {
			return err
		}
		current, err := readRegular(filepath.Join(target, manifestName))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !bytes.Equal(current, expected) {
			return ErrConflict
		}
		file, err := os.CreateTemp(target, ".lingo-manifest-")
		if err != nil {
			return fmt.Errorf("create replacement manifest: %w", err)
		}
		temporary := file.Name()
		defer os.Remove(temporary)
		if err := file.Chmod(0o600); err != nil {
			file.Close()
			return fmt.Errorf("protect replacement manifest: %w", err)
		}
		if _, err := file.Write(manifest); err != nil {
			file.Close()
			return fmt.Errorf("write replacement manifest: %w", err)
		}
		if err := file.Sync(); err != nil {
			file.Close()
			return fmt.Errorf("sync replacement manifest: %w", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close replacement manifest: %w", err)
		}
		if err := os.Rename(temporary, filepath.Join(target, manifestName)); err != nil {
			return fmt.Errorf("publish replacement manifest: %w", err)
		}
		return syncDirectory(target)
	})
}

func (s PortableStore) withLock(slug string, action func(string) error) error {
	if !project.ValidSlug(slug) {
		return ErrUnsafe
	}
	lock := filepath.Join(s.root, ".lingo-locks", slug+".lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		if os.IsExist(err) {
			return ErrConflict
		}
		return fmt.Errorf("acquire portable lock: %w", err)
	}
	defer os.Remove(lock)
	return action(filepath.Join(s.root, slug))
}

func onlyManifest(target string) error {
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ErrUnsafe
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return fmt.Errorf("read portable project: %w", err)
	}
	if len(entries) != 1 || entries[0].Name() != manifestName {
		return ErrUnsafe
	}
	return nil
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrUnsafe
	}
	return os.ReadFile(path)
}

func writeDurable(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create portable manifest: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write portable manifest: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync portable manifest: %w", err)
	}
	return file.Close()
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open portable directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync portable directory: %w", err)
	}
	return nil
}
