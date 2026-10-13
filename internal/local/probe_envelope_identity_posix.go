//go:build !windows

package local

import (
	"fmt"
	"os"
	"syscall"
)

func envelopeIdentity(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrUnsafeEnvelope
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}
