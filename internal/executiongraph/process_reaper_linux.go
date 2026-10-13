//go:build linux

package executiongraph

import (
	"sync"
	"syscall"
)

const prSetChildSubreaper = 36

var subreaper struct {
	once sync.Once
	err  error
}

// enableChildSubreaper makes orphaned attempt descendants reparent to this
// process, so a double-forked escapee stays observable at stop confirmation.
func enableChildSubreaper() error {
	subreaper.once.Do(func() {
		if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prSetChildSubreaper, 1, 0); errno != 0 {
			subreaper.err = errno
		}
	})
	return subreaper.err
}
