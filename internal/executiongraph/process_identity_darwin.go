//go:build darwin

package executiongraph

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
)

func ObserveProcessCorrelation(pid int64) (*ProcessCorrelation, error) {
	if pid <= 0 {
		return nil, ErrInvalidGraph
	}
	processes, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}
	for _, p := range processes {
		if int64(p.Proc.P_pid) == pid && p.Proc.P_stat != 5 {
			return &ProcessCorrelation{PID: pid, StartIdentity: fmt.Sprintf("%d:%d", p.Proc.P_starttime.Sec, p.Proc.P_starttime.Usec), GroupIdentity: fmt.Sprintf("%d", p.Eproc.Pgid)}, nil
		}
	}
	return nil, nil
}
func ObserveRecordedProcess(recorded ProcessCorrelation) (*ProcessCorrelation, error) {
	current, err := ObserveProcessCorrelation(recorded.PID)
	if err != nil {
		return nil, err
	}
	if current != nil && *current == recorded {
		return current, nil
	}
	processes, err := observeProcesses()
	if err != nil {
		return nil, err
	}
	for _, p := range processes {
		if fmt.Sprintf("%d", p.group) == recorded.GroupIdentity {
			return nil, errors.New("recorded group alive without matching leader identity")
		}
	}
	return current, nil
}
