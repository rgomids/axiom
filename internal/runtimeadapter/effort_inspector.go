package runtimeadapter

import (
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// EffortSourceRoots are the trusted current local roots of the closed
// configuration anchors (Specification 007 AXM-22 BI-2a D6). They are host
// observations supplied by the caller, never read from persisted configuration.
type EffortSourceRoots struct {
	User, Project, Managed string
}

// trustedEffortAnchor applies trustedEffortRoot, except that a missing
// managed root is an absence fact (Specification 007 PM-13): its nearest
// existing ancestor must be a safe anchor-grade directory no other principal
// can create entries in, every existing ancestor above it must be safe, and
// its absent binding is revalidated like a present one. Nothing is created.
// User and project roots must exist.
func trustedEffortAnchor(anchor, root string) (string, error) {
	if anchor != runtimeprofile.EffortAnchorManaged {
		return trustedEffortRoot(root)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		return trustedEffortRoot(root)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return "", ErrEffortConfigurationUnsafe
	}
	binding, present, err := effortPathBinding(root)
	if err != nil || present {
		return "", ErrEffortConfigurationUnsafe
	}
	nearest := true
	for current := filepath.Dir(root); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		switch {
		case os.IsNotExist(err) && nearest:
		case err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0:
			return "", ErrEffortConfigurationUnsafe
		case nearest && !effortRootSafe(info), !nearest && !effortAncestorSafe(info):
			return "", ErrEffortConfigurationUnsafe
		default:
			nearest = false
		}
		if current == filepath.Dir(current) {
			if nearest {
				return "", ErrEffortConfigurationUnsafe
			}
			return "absent\x00" + binding, nil
		}
	}
}

// trustedEffortRoot requires an existing, canonical, non-symlinked anchor
// directory whose path and identity bind through the PE-05a boundary, owned
// by the current user or root without group/other write, below ancestors that
// no other principal can replace. Nothing is created or repaired.
func trustedEffortRoot(root string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return "", ErrEffortConfigurationUnsafe
	}
	binding, present, err := effortPathBinding(root)
	if err != nil || !present {
		return "", ErrEffortConfigurationUnsafe
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || !effortRootSafe(info) {
		return "", ErrEffortConfigurationUnsafe
	}
	for current := filepath.Dir(root); ; current = filepath.Dir(current) {
		ancestor, err := os.Lstat(current)
		if err != nil || !ancestor.IsDir() || !effortAncestorSafe(ancestor) {
			return "", ErrEffortConfigurationUnsafe
		}
		if current == filepath.Dir(current) {
			return binding, nil
		}
	}
}
