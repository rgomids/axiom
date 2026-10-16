//go:build unix

package executiongraph

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// startedGroups pins direct attempt leaders by kernel start identity. Their
// descendants never inherit that trust by joining a registered process group.
var startedGroups = struct {
	sync.Mutex
	leaders map[int]string
}{leaders: map[int]string{}}

type posixProcessGroup struct {
	pid            int
	escaped        bool
	observationErr error
	orphans        map[int]string
	identity       *ProcessCorrelation
}

var errProcessGroupIdentity = errors.New("process group identity unavailable or changed")

func newProcessContainment() (processContainment, error) { return &posixProcessGroup{}, nil }
func (g *posixProcessGroup) start(cmd *exec.Cmd) error {
	// Orphaned descendants must reparent to this process to stay observable.
	if err := enableChildSubreaper(); err != nil {
		return err
	}
	baseline, err := observeAdoptedProcesses(true, 0)
	if err != nil {
		return err
	}
	g.orphans = baseline
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	g.pid = cmd.Process.Pid
	identity, identityErr := ObserveProcessCorrelation(int64(g.pid))
	g.identity = identity
	if identityErr != nil {
		g.observationErr = identityErr
	}
	startedGroups.Lock()
	startedGroups.leaders[g.pid] = ""
	if identity != nil {
		startedGroups.leaders[g.pid] = identity.StartIdentity
	}
	startedGroups.Unlock()
	return nil
}
func (g *posixProcessGroup) stop() error {
	if g.pid <= 1 {
		return ErrInvalidGraph
	}
	if err := g.sameLeader(); err != nil {
		return err
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
	if err := g.sameLeader(); err != nil {
		return false, err
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

func (g *posixProcessGroup) sameLeader() error {
	if g.identity == nil {
		return errProcessGroupIdentity
	}
	current, err := ObserveProcessCorrelation(int64(g.pid))
	if err != nil {
		return err
	}
	if current != nil && *current != *g.identity {
		return errProcessGroupIdentity
	}
	return nil
}
func (g *posixProcessGroup) close() {
	startedGroups.Lock()
	delete(startedGroups.leaders, g.pid)
	startedGroups.Unlock()
}

type processObservation struct{ pid, parent, group int }

func (g *posixProcessGroup) observeEscapes() {
	adopted, err := observeAdoptedProcesses(false, g.pid)
	if err != nil {
		g.observationErr = err
		return
	}
	for pid, identity := range adopted {
		if g.orphans[pid] != identity {
			g.escaped = true
		}
	}
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
	// An orphan can join the supervisor's or another attempt's group. Group
	// membership cannot attribute it; only known attempt leaders are excluded.
	self := os.Getpid()
	startedGroups.Lock()
	for _, p := range processes {
		if p.parent != self || p.group == g.pid {
			continue
		}
		if start := startedGroups.leaders[p.pid]; start != "" {
			current, err := ObserveProcessCorrelation(int64(p.pid))
			if err != nil {
				g.observationErr = err
			} else if current != nil && current.StartIdentity == start {
				continue
			}
		}
		descendants[p.pid] = true
	}
	startedGroups.Unlock()
	for _, p := range processes {
		if descendants[p.pid] && p.group != g.pid {
			g.escaped = true
		}
	}
}
