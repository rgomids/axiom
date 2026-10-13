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
		return "", ErrEffortConfigurationUnproven
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", ErrEffortConfigurationUnproven
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}

func effortCanonicalTarget(path string) (string, error) { return path, nil }
