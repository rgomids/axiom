//go:build windows

package stageexecution

import (
	"fmt"
	"github.com/rgomids/axiom/internal/executiongraph"
	"golang.org/x/sys/windows"
	"os/exec"
	"syscall"
)

func configureDetached(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200}
	return nil
}

// Supervisor correlation is a detached session, not a child containment job.
func observeDetached(pid int64) (*executiongraph.ProcessCorrelation, error) {
	if pid <= 0 || pid > int64(^uint32(0)) {
		return nil, ErrDetachmentUnsupported
	}
	h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if e != nil {
		return nil, e
	}
	defer windows.CloseHandle(h)
	var creation, exit, kernel, user windows.Filetime
	if e = windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); e != nil {
		return nil, e
	}
	if exit.HighDateTime != 0 || exit.LowDateTime != 0 {
		return nil, ErrDetachmentUnsupported
	}
	id := fmt.Sprintf("%08x%08x", creation.HighDateTime, creation.LowDateTime)
	return &executiongraph.ProcessCorrelation{PID: pid, StartIdentity: id, GroupIdentity: fmt.Sprintf("detached:%d:%s", pid, id)}, nil
}
