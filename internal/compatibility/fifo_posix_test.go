//go:build unix

package compatibility

import "syscall"

func makeFIFO(path string) error { return syscall.Mkfifo(path, 0o600) }
