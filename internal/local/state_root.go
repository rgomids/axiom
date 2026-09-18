package local

import "path/filepath"

// NativeStateRoot resolves the approved platform directory without filesystem
// effects. XDG_STATE_HOME is used only when it is an absolute Linux path.
func NativeStateRoot(goos, home, xdgStateHome string) (string, error) {
	if !filepath.IsAbs(home) {
		return "", ErrUnsafe
	}
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Lingo"), nil
	case "linux":
		if filepath.IsAbs(xdgStateHome) {
			return filepath.Join(xdgStateHome, "lingo"), nil
		}
		return filepath.Join(home, ".local", "state", "lingo"), nil
	default:
		return "", ErrUnsafe
	}
}
