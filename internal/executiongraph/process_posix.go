//go:build !windows

package executiongraph

import (
	"errors"
	"os/exec"
	"syscall"
)

type posixProcessGroup struct {
	pid            int
	escaped        bool
	observationErr error
}

func newProcessContainment() (processContainment, error) { return &posixProcessGroup{}, nil }
func (g *posixProcessGroup) start(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	g.pid = cmd.Process.Pid
	return nil
}
func (g *posixProcessGroup) stop() error {
	g.observeEscapes()
	err := syscall.Kill(-g.pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
func (g *posixProcessGroup) gone() (bool, error) {
	err := syscall.Kill(-g.pid, 0)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return false, err
	}
	// Group stop is observable; arbitrary daemonization between snapshots is
	// not. This is advisory group containment, not a security sandbox.
	if g.observationErr != nil {
		return false, g.observationErr
	}
	if g.escaped {
		return false, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return true, nil
	}
	processes, observeErr := observeProcesses()
	if observeErr != nil {
		return false, observeErr
	}
	for _, p := range processes {
		if p.group == g.pid {
			return false, nil
		}
	}
	return true, nil
}
func (*posixProcessGroup) close() {}

type processObservation struct{ pid, parent, group int }

func (g *posixProcessGroup) observeEscapes() {
	processes, err := observeProcesses()
	if err != nil {
		g.observationErr = err
		return
	}
	descendants := map[int]bool{g.pid: true}
	for changed := true; changed; {
		changed = false
		for _, p := range processes {
			if descendants[p.parent] && !descendants[p.pid] {
				descendants[p.pid] = true
				changed = true
			}
		}
	}
	for _, p := range processes {
		if descendants[p.pid] && p.group != g.pid {
			g.escaped = true
		}
	}
}
