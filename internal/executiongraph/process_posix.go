//go:build unix

package executiongraph

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// startedGroups are the process groups this process created for attempts.
// A reparented orphan in any other group cannot be attributed to an attempt
// and is therefore treated as an escape by every stop confirmation.
var startedGroups = struct {
	sync.Mutex
	leaders map[int]bool
}{leaders: map[int]bool{}}

type posixProcessGroup struct {
	pid            int
	escaped        bool
	observationErr error
}

func newProcessContainment() (processContainment, error) { return &posixProcessGroup{}, nil }
func (g *posixProcessGroup) start(cmd *exec.Cmd) error {
	// Orphaned descendants must reparent to this process to stay observable.
	if err := enableChildSubreaper(); err != nil {
		return err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	g.pid = cmd.Process.Pid
	startedGroups.Lock()
	startedGroups.leaders[g.pid] = true
	startedGroups.Unlock()
	return nil
}
func (g *posixProcessGroup) stop() error {
	if g.pid <= 1 {
		return ErrInvalidGraph
	}
	g.observeEscapes()
	err := syscall.Kill(-g.pid, syscall.SIGKILL)
	// Darwin reports EPERM for a group whose members are all zombies; gone()
	// confirms through process observation instead of trusting the signal.
	if errors.Is(err, syscall.ESRCH) || errors.Is(err, syscall.EPERM) {
		return nil
	}
	return err
}
func (g *posixProcessGroup) gone() (bool, error) {
	if g.pid <= 1 {
		return false, ErrInvalidGraph
	}
	err := syscall.Kill(-g.pid, 0)
	if errors.Is(err, syscall.EPERM) {
		err = nil // zombie-only group on Darwin: decide by observation below
	}
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
	// Intermediates may exit between the stop snapshot and the kill, orphaning
	// an escapee; observe again before any confirmation.
	g.observeEscapes()
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
func (g *posixProcessGroup) close() {
	startedGroups.Lock()
	delete(startedGroups.leaders, g.pid)
	startedGroups.Unlock()
}

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
	// With the child subreaper, an orphaned descendant is reparented here. One
	// outside this process's own group and every started attempt group cannot be
	// attributed, so it is never mistaken for a confirmed stop.
	self, selfGroup := os.Getpid(), syscall.Getpgrp()
	startedGroups.Lock()
	for _, p := range processes {
		if p.parent == self && p.group != selfGroup && !startedGroups.leaders[p.group] {
			descendants[p.pid] = true
		}
	}
	startedGroups.Unlock()
	for _, p := range processes {
		if descendants[p.pid] && p.group != g.pid {
			g.escaped = true
		}
	}
}
