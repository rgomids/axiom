package local

import (
	"os"
	"path/filepath"
)

// RuntimeProfileStore owns the canonical local Runtime/Profile record. Its only
// reader is Inspect and its only writer is Publish (runtime_profile_authoring.go);
// no format-1/2 configuration writer exists (Specification 007 PM-11).
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
