package local

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

type RuntimeProfileStore struct {
	root  string
	hooks publicationHooks
}

func NewRuntimeProfileStore(root string) (RuntimeProfileStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return RuntimeProfileStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return RuntimeProfileStore{}, err
	}
	return RuntimeProfileStore{root: canonical}, nil
}

func (s RuntimeProfileStore) Create(ctx context.Context, cfg runtimeprofile.Configuration) error {
	if cfg.Revision != 1 || cfg.FormatVersion != runtimeprofile.FormatVersion {
		return runtimeprofile.ErrInvalidConfiguration
	}
	return s.write(ctx, cfg, true)
}

func (s RuntimeProfileStore) Save(ctx context.Context, cfg runtimeprofile.Configuration) error {
	if cfg.FormatVersion != runtimeprofile.FormatVersion {
		return runtimeprofile.ErrInvalidConfiguration
	}
	return s.write(ctx, cfg, false)
}

func (s RuntimeProfileStore) write(ctx context.Context, cfg runtimeprofile.Configuration, create bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	wire, err := runtimeprofile.Encode(cfg)
	if err != nil {
		return err
	}
	root, profiles, version, err := s.open(create)
	if err != nil {
		return err
	}
	defer root.Close()
	defer profiles.Close()
	defer version.Close()
	locks, err := lockRoots(true, root, profiles, version)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	var expected []byte
	if !create {
		expected, err = readPublishedFile(version, "configuration.json")
		if err != nil {
			return err
		}
		current, err := runtimeprofile.Decode(expected)
		if err != nil {
			return ErrUnsafe
		}
		if current.FormatVersion != runtimeprofile.FormatVersion || cfg.FormatVersion != current.FormatVersion || cfg.Revision != current.Revision+1 {
			return ErrConflict
		}
	}
	facts, err := s.runtimeProfileFacts()
	if err != nil {
		return err
	}
	hooks := s.hooks
	before := hooks.beforeCommit
	hooks.beforeCommit = func() error {
		if before != nil {
			if err := before(); err != nil {
				return err
			}
		}
		observed, err := s.runtimeProfileFacts()
		if err != nil || observed != facts {
			return ErrConflict
		}
		return nil
	}
	if err := publishFile(ctx, version, "configuration.json", expected, wire, create, hooks); err != nil {
		return err
	}
	for _, item := range []struct {
		root *os.Root
		path string
	}{{root, s.root}, {profiles, filepath.Join(s.root, "runtime-profiles")}, {version, filepath.Join(s.root, "runtime-profiles", "v1")}} {
		if err := stillAtPath(item.root, item.path); err != nil {
			return publicationFailure(FaultF6, true, ErrRecoveryRequired)
		}
	}
	return nil
}

func (s RuntimeProfileStore) Load(ctx context.Context) (runtimeprofile.Configuration, error) {
	if err := ctx.Err(); err != nil {
		return runtimeprofile.Configuration{}, err
	}
	root, profiles, version, err := s.open(false)
	if err != nil {
		return runtimeprofile.Configuration{}, err
	}
	defer root.Close()
	defer profiles.Close()
	defer version.Close()
	locks, err := lockRoots(false, root, profiles, version)
	if err != nil {
		return runtimeprofile.Configuration{}, err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(version, "configuration.json")
	if err != nil {
		return runtimeprofile.Configuration{}, err
	}
	return runtimeprofile.Decode(wire)
}

func (s RuntimeProfileStore) open(create bool) (*os.Root, *os.Root, *os.Root, error) {
	openRoot := existingPrivateRoot
	openChild := existingPrivateChild
	if create {
		openRoot = privateRoot
		openChild = privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, nil, err
	}
	profiles, err := openChild(root, "runtime-profiles")
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	version, err := openChild(profiles, "v1")
	if err != nil {
		profiles.Close()
		root.Close()
		return nil, nil, nil, err
	}
	return root, profiles, version, nil
}
