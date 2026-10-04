//go:build !windows

package local

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InspectInstallFile reads existing owned content before the installer's
// separate hard-link refusal. Inspection is read-only and anchored.
func (d AnchoredDirectory) InspectInstallFile(name string, limit int) ([]byte, error) {
	if !safeEntryName(name) || limit <= 0 || d.StillAtPath() != nil {
		return nil, ErrUnsafe
	}
	file, err := d.root.OpenFile(name, os.O_RDONLY|noFollow, 0)
	if err != nil {
		return nil, ErrUnsafe
	}
	defer file.Close()
	info, err := file.Stat()
	visible, visibleErr := d.root.Lstat(name)
	if err != nil || visibleErr != nil || !info.Mode().IsRegular() || !visible.Mode().IsRegular() || !os.SameFile(info, visible) || !ownedByUser(info) || forbiddenPermissions(info, 0o077) {
		return nil, ErrUnsafe
	}
	if err := checkPrivateACL(file); err != nil {
		return nil, err
	}
	wire, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil || len(wire) > limit {
		return nil, ErrUnsafe
	}
	return wire, nil
}

// CheckInstallDirectory inspects existing components without creating or
// repairing them. Unlike general state roots, installation roots refuse all
// symlinks, including system-owned aliases.
func CheckInstallDirectory(path string, private bool) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) == "/" {
		return ErrUnsafe
	}
	// Inspect lexical prefixes before cleaning; link/.. must not erase a link.
	lexical := ""
	for _, part := range strings.Split(path, "/") {
		lexical += part + "/"
		if info, err := os.Lstat(strings.TrimSuffix(lexical, "/")); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafe
		}
	}
	current := "/"
	parts := pathComponents(filepath.Clean(path))
	for i, part := range parts {
		container, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil || !container.IsDir() || !ancestorSafe(container) {
			return ErrUnsafe
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafe
		}
		if i == len(parts)-1 {
			if !ownedByUser(info) || info.Mode().Perm()&0o022 != 0 || private && (info.Mode().Perm() != 0o700 || runtime.GOOS == "linux" && info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0) {
				return ErrUnsafe
			}
			file, err := os.Open(current)
			if err != nil {
				return ErrUnsafe
			}
			err = checkPrivateACL(file)
			file.Close()
			return err
		}
	}
	return ErrUnsafe
}

// CreateOwnedDirectory creates missing private components without chmod-repair.
func CreateOwnedDirectory(path string) (AnchoredDirectory, error) {
	if err := CheckInstallDirectory(path, true); err != nil {
		return AnchoredDirectory{}, err
	}
	canonical := filepath.Clean(path)
	root, chain, err := anchoredRoot(canonical, true)
	if err != nil {
		return AnchoredDirectory{}, err
	}
	root.Close()
	directory, err := OpenOwnedDirectory(canonical)
	if err != nil {
		return AnchoredDirectory{}, err
	}
	directory.anchor = anchor{path: canonical, chain: chain}
	if err := directory.StillAtPath(); err != nil {
		directory.Close()
		return AnchoredDirectory{}, err
	}
	return directory, nil
}

// PublishNew commits through the anchored directory without replacement.
func (d AnchoredDirectory) PublishNew(staged, destination string) error {
	if !safeEntryName(staged) || !safeEntryName(destination) || strings.ContainsAny(staged+destination, "\r\n") {
		return ErrUnsafe
	}
	if err := d.StillAtPath(); err != nil {
		return err
	}
	return renameNoReplace(d.root, staged, destination)
}
