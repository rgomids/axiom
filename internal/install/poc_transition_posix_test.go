//go:build !windows

package install

import (
	"os"
	"syscall"
	"testing"
)

func sameDevice(t *testing.T, first, second string) bool {
	t.Helper()
	a, errA := os.Stat(first)
	b, errB := os.Stat(second)
	if errA != nil || errB != nil {
		return true
	}
	return a.Sys().(*syscall.Stat_t).Dev == b.Sys().(*syscall.Stat_t).Dev
}
