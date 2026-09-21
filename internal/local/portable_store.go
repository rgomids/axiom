package local

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

const manifestName = "axiom.yaml"

var (
	ErrNotFound         = projectapp.ErrNotFound
	ErrConflict         = projectapp.ErrConflict
	ErrUnsafe           = projectapp.ErrUnsafe
	ErrRecoveryRequired = projectapp.ErrRecoveryRequired
)

// PortableStore supports one validated manifest per Project. Operations stay
// anchored to private directory objects and coordinate across processes.
type PortableStore struct {
	root                    string
	beforeCreatePublication func()
	afterCreatePublication  func()
	beforeUpdatePublication func()
	afterUpdatePublication  func()
	syncDirectory           func(*os.Root) error
	writeFile               func(*os.Root, string, []byte) error
	removeAttempt           func(*os.Root, string) error
}

func NewPortableStore(path string) (PortableStore, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) == string(filepath.Separator) {
		return PortableStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(path)
	if err != nil {
		return PortableStore{}, err
	}
	return PortableStore{root: canonical}, nil
}

func (s PortableStore) Read(ctx context.Context, slug string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result []byte
	err := s.withLock(slug, false, false, func(root *os.Root) error {
		projectRoot, err := openManifestProject(root, slug)
		if err != nil {
			return err
		}
		defer projectRoot.Close()
		result, err = readPrivateFile(projectRoot, manifestName)
		if os.IsNotExist(err) {
			return ErrUnsafe
		}
		return err
	})
	return result, err
}

// Create publishes a complete private directory with no replacement.
// Directory rename is the commit point.
func (s PortableStore) Create(ctx context.Context, slug string, manifest []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.withLock(slug, true, true, func(root *os.Root) error {
		if _, err := root.Lstat(slug); err == nil {
			return ErrConflict
		} else if !os.IsNotExist(err) {
			return err
		}
		stageName, err := temporaryName(".lingo-stage-" + slug + "-")
		if err != nil {
			return err
		}
		stage, err := privateChild(root, stageName)
		if err != nil {
			return err
		}
		defer stage.Close()
		published := false
		defer func() {
			if !published {
				_ = stage.Remove(manifestName)
				_ = root.Remove(stageName)
			}
		}()
		if err := s.write(stage, manifestName, manifest); err != nil {
			return err
		}
		if err := s.sync(stage); err != nil {
			return err
		}
		if s.beforeCreatePublication != nil {
			s.beforeCreatePublication()
		}
		if err := verifyPreparedFile(stage, manifestName, manifest); err != nil {
			return err
		}
		if err := stillAtPath(root, s.root); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		attempt, err := markAttempt(root, ".lingo-attempt-"+slug+"-")
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			if cleanup := s.clear(root, attempt); cleanup != nil {
				return cleanup
			}
			return err
		}
		if err := renameNoReplace(root, stageName, slug); err != nil {
			if cleanup := s.clear(root, attempt); cleanup != nil {
				return cleanup
			}
			if os.IsExist(err) {
				return ErrConflict
			}
			return fmt.Errorf("publish project: %w", err)
		}
		published = true
		if s.afterCreatePublication != nil {
			s.afterCreatePublication()
		}
		if err := s.sync(root); err != nil {
			return fmt.Errorf("project committed; durability unverified: %w", ErrRecoveryRequired)
		}
		return s.clear(root, attempt)
	})
}

// Update compares exact observed bytes under a process lock and replaces
// only the supported manifest. Readers never accept a partial manifest.
func (s PortableStore) Update(ctx context.Context, slug string, expected, manifest []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.withLock(slug, false, true, func(root *os.Root) error {
		projectRoot, err := openManifestProject(root, slug)
		if err != nil {
			return err
		}
		defer projectRoot.Close()
		current, err := readPrivateFile(projectRoot, manifestName)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, expected) {
			return ErrConflict
		}
		temporary, err := temporaryName(".lingo-manifest-")
		if err != nil {
			return err
		}
		defer projectRoot.Remove(temporary)
		if err := s.write(projectRoot, temporary, manifest); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		attempt, err := markAttempt(projectRoot, ".lingo-attempt-update-")
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			if cleanup := s.clear(projectRoot, attempt); cleanup != nil {
				return cleanup
			}
			return err
		}
		if s.beforeUpdatePublication != nil {
			s.beforeUpdatePublication()
		}
		if err := verifyPreparedFile(projectRoot, temporary, manifest); err != nil {
			if s.clear(projectRoot, attempt) != nil {
				return ErrRecoveryRequired
			}
			return err
		}
		if err := stillAtPath(root, s.root); err != nil {
			if s.clear(projectRoot, attempt) != nil {
				return ErrRecoveryRequired
			}
			return err
		}
		if err := stillAtPath(projectRoot, filepath.Join(s.root, slug)); err != nil {
			if s.clear(projectRoot, attempt) != nil {
				return ErrRecoveryRequired
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			if cleanup := s.clear(projectRoot, attempt); cleanup != nil {
				return cleanup
			}
			return err
		}
		if err := projectRoot.Rename(temporary, manifestName); err != nil {
			if cleanup := s.clear(projectRoot, attempt); cleanup != nil {
				return cleanup
			}
			return err
		}
		if s.afterUpdatePublication != nil {
			s.afterUpdatePublication()
		}
		if err := s.sync(projectRoot); err != nil {
			return fmt.Errorf("project committed; durability unverified: %w", ErrRecoveryRequired)
		}
		return s.clear(projectRoot, attempt)
	})
}

func (s PortableStore) sync(root *os.Root) error {
	if s.syncDirectory != nil {
		return s.syncDirectory(root)
	}
	return syncRoot(root)
}

func (s PortableStore) write(root *os.Root, name string, content []byte) error {
	if s.writeFile != nil {
		return s.writeFile(root, name, content)
	}
	return writePrivateFile(root, name, content)
}

func (s PortableStore) clear(root *os.Root, name string) error {
	if s.removeAttempt != nil {
		return s.removeAttempt(root, name)
	}
	return clearAttempt(root, name)
}

func (s PortableStore) withLock(slug string, create, exclusive bool, action func(*os.Root) error) error {
	if !project.ValidSlug(slug) {
		return ErrUnsafe
	}
	open := existingPrivateRoot
	if create {
		open = privateRoot
	}
	root, err := open(s.root)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := stillAtPath(root, s.root); err != nil {
		return err
	}
	lock, err := lockDirectory(root, exclusive)
	if err != nil {
		return err
	}
	defer lock.Close()
	entries, err := root.Open(".")
	if err != nil {
		return err
	}
	names, err := entries.Readdirnames(-1)
	entries.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if strings.HasPrefix(name, ".lingo-stage-"+slug+"-") || strings.HasPrefix(name, ".lingo-attempt-"+slug+"-") {
			return ErrRecoveryRequired
		}
	}
	return action(root)
}

func openManifestProject(root *os.Root, slug string) (*os.Root, error) {
	projectRoot, err := existingPrivateChild(root, slug)
	if err != nil {
		return nil, err
	}
	file, err := projectRoot.Open(".")
	if err != nil {
		projectRoot.Close()
		return nil, err
	}
	entries, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		projectRoot.Close()
		return nil, err
	}
	for _, name := range entries {
		if strings.HasPrefix(name, ".lingo-manifest-") || strings.HasPrefix(name, ".lingo-attempt-update-") {
			projectRoot.Close()
			return nil, ErrRecoveryRequired
		}
	}
	if len(entries) != 1 || entries[0] != manifestName {
		projectRoot.Close()
		return nil, ErrUnsafe
	}
	return projectRoot, nil
}
