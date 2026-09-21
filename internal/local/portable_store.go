package local

import (
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
		published := false
		stageOpen := true
		defer func() {
			if !published {
				if stageOpen {
					_ = stage.Remove(manifestName)
					_ = stage.Close()
				}
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
		hooks := s.publicationHooks()
		markerName, err := writeProtocolMarker(root, slug, stageName, false, [32]byte{}, digestBytes(manifest), hooks)
		if err != nil {
			return publicationFailure(FaultF3, false, err)
		}
		marker, err := readProtocolMarker(root, markerName)
		if err != nil {
			return publicationFailure(FaultF3, false, err)
		}
		removeStage := func() error {
			if stageOpen {
				if err := stage.Remove(manifestName); err != nil {
					return err
				}
				if err := stage.Close(); err != nil {
					return err
				}
				stageOpen = false
			}
			return hooks.removeName(root, stageName)
		}
		cleanup := func() error { return cleanupPreCommit(root, markerName, removeStage, hooks) }
		if err := ctx.Err(); err != nil {
			return cleanupFailure(FaultF3, err, cleanup())
		}
		if err := updateProtocolStage(root, markerName, marker, FaultF5, hooks); err != nil {
			return publicationFailure(FaultF4, false, ErrRecoveryRequired)
		}
		if err := renameNoReplace(root, stageName, slug); err != nil {
			if os.IsExist(err) {
				return cleanupFailure(FaultF5, ErrConflict, cleanup())
			}
			return cleanupFailure(FaultF5, fmt.Errorf("publish project: %w", err), cleanup())
		}
		published = true
		if err := stage.Close(); err != nil {
			return publicationFailure(FaultF6, true, ErrRecoveryRequired)
		}
		stageOpen = false
		if s.afterCreatePublication != nil {
			s.afterCreatePublication()
		}
		if err := updateProtocolStage(root, markerName, marker, FaultF6, hooks); err != nil {
			return publicationFailure(FaultF6, true, ErrRecoveryRequired)
		}
		if err := hooks.syncRoot(root); err != nil {
			return publicationFailure(FaultF6, true, ErrRecoveryRequired)
		}
		if err := updateProtocolStage(root, markerName, marker, FaultF7, hooks); err != nil {
			return publicationFailure(FaultF7, true, ErrRecoveryRequired)
		}
		if err := updateProtocolStage(root, markerName, marker, FaultF8, hooks); err != nil {
			return publicationFailure(FaultF8, true, ErrRecoveryRequired)
		}
		if err := hooks.removeName(root, markerName); err != nil {
			return publicationFailure(FaultF8, true, ErrRecoveryRequired)
		}
		if err := hooks.syncRoot(root); err != nil {
			return publicationFailure(FaultF8, true, ErrRecoveryRequired)
		}
		return nil
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
		hooks := s.publicationHooks()
		hooks.afterStage = s.beforeUpdatePublication
		hooks.beforeCommit = func() error {
			if err := stillAtPath(root, s.root); err != nil {
				return err
			}
			return stillAtPath(projectRoot, filepath.Join(s.root, slug))
		}
		hooks.afterCommit = s.afterUpdatePublication
		return publishFile(ctx, projectRoot, manifestName, expected, manifest, false, hooks)
	})
}

func (s PortableStore) publicationHooks() publicationHooks {
	return publicationHooks{remove: s.removeAttempt, sync: s.syncDirectory, write: s.writeFile, stagePrefix: ".lingo-manifest-"}
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
	if pending, err := protocolStatePresent(root); err != nil || pending {
		if err != nil {
			return err
		}
		return ErrRecoveryRequired
	}
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
	if pending, err := protocolStatePresent(projectRoot); err != nil || pending {
		projectRoot.Close()
		if err != nil {
			return nil, err
		}
		return nil, ErrRecoveryRequired
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
