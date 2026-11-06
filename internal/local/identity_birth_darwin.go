//go:build darwin

package local

import "syscall"

// objectBirth is the creation time APFS/HFS+ keep with the object itself, so
// it survives reboot and remount, unlike st_dev.
func objectBirth(_ string, stat *syscall.Stat_t) (string, error) {
	return birthFact(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec), nil
}
