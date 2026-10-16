package local

import "golang.org/x/sys/unix"

func probeFilesystemReadOnly(stat unix.Statfs_t) bool { return stat.Flags&unix.MNT_RDONLY != 0 }
