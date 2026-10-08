package local

import (
	"os"
	"path/filepath"
)

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
	case "windows":
		return filepath.Join(home, ".axiom", "windows", "state"), nil
	default:
		return "", ErrUnsafe
	}
}

// WindowsStateRoot keeps existing legacy state in place. Fresh installations
// use private profile storage, independently of AppData ancestor permissions.
func WindowsStateRoot(home, localAppData string) (string, error) {
	if !filepath.IsAbs(home) || (localAppData != "" && !filepath.IsAbs(localAppData)) {
		return "", ErrUnsafe
	}
	if localAppData == "" {
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	modern := filepath.Join(home, ".axiom", "windows", "state")
	if _, err := os.Lstat(modern); err == nil {
		return modern, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	legacy := filepath.Join(localAppData, "Axiom", "state")
	if _, err := os.Lstat(legacy); err == nil {
		return legacy, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return modern, nil
}
