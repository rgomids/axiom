//go:build unix

package runtimeadapter

import (
	"fmt"
	"os"
	"syscall"
)

func effortObjectIdentity(file *os.File) (string, error) {
	info, err := file.Stat()
	if err != nil {
		return "", ErrEffortConfigurationUnsafe
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", ErrEffortConfigurationUnsafe
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}

func effortCanonicalTarget(path string) (string, error) { return path, nil }

// effortAncestorSafe mirrors the ADR-0005 container rule used by the local
// filesystem boundary: owned by root or the current user, and without group or
// other write unless the sticky bit restricts replacement of existing entries.
func effortAncestorSafe(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 && int(stat.Uid) != os.Geteuid() {
		return false
	}
	return info.Mode().Perm()&0o022 == 0 || info.Mode()&os.ModeSticky != 0
}

// effortRootSafe is stricter for an anchor root itself: another principal must
// not be able to create an absent declared source there, so sticky is no excuse.
func effortRootSafe(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && (stat.Uid == 0 || int(stat.Uid) == os.Geteuid()) && info.Mode().Perm()&0o022 == 0
}
