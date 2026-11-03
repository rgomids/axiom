//go:build unix

package gitworkspace

import (
	"golang.org/x/sys/unix"
	"os"
	"syscall"
)

func privateDirectory(name string) bool {
	info, err := os.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return false
	}
	directory, err := os.OpenFile(name, os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return false
	}
	defer directory.Close()
	opened, err := directory.Stat()
	if err != nil || !os.SameFile(info, opened) || !ownedByUser(opened) {
		return false
	}
	return checkPrivateACL(directory) == nil
}

func ownedByUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == effectiveUID()
}
