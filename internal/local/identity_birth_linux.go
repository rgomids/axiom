//go:build linux

package local

import (
	"errors"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// objectBirth reads the object's birth time through statx without following a
// link. The inode must still be the one lstat observed.
func objectBirth(clean string, stat *syscall.Stat_t) (string, error) {
	var x unix.Statx_t
	err := unix.Statx(unix.AT_FDCWD, clean, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_INO|unix.STATX_BTIME, &x)
	if err == nil && x.Ino != stat.Ino {
		return "", errors.New("identity is unavailable")
	}
	if err != nil || x.Mask&unix.STATX_BTIME == 0 {
		// shortcut: without statx birth time (old kernel, NFS) the device number stands in and a remount needs a rebind; upgrade if such hosts matter.
		return "dev:" + strconv.FormatUint(stat.Dev, 10), nil
	}
	return birthFact(x.Btime.Sec, int64(x.Btime.Nsec)), nil
}
