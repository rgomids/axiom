package local

import "golang.org/x/sys/unix"

func probeFilesystemReadOnly(stat unix.Statfs_t) bool { return stat.Flags&unix.ST_RDONLY != 0 }
