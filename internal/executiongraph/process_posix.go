//go:build unix

package executiongraph

import (
	"errors"
	"os/exec"
	"syscall"
)

// posixProcessGroup supervises the attempt leader and its own POSIX process
// group. Descendants that deliberately leave the group and lose ancestry to the
// leader (setsid plus double fork) are outside this boundary; an escape that is
// still observable through the leader's ancestry vetoes confirmation.
type posixProcessGroup struct {
	pid            int
	escaped        bool
	observationErr error
	identity       *ProcessCorrelation
	exitedAtStart  bool
}

var errProcessGroupIdentity = errors.New("process group identity unavailable or changed")

func newProcessContainment() (processContainment, error) { return &posixProcessGroup{}, nil }
func (g *posixProcessGroup) start(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	g.pid = cmd.Process.Pid
	g.identity, g.observationErr = ObserveProcessCorrelation(int64(g.pid))
	// An unreaped leader that already exited is unobservable but still holds
	// its PID; any later live process at that PID is a reuse and is refused.
	g.exitedAtStart = g.identity == nil && g.observationErr == nil
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
	// confirms through group observation instead of trusting the signal.
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
	if g.observationErr != nil {
		return false, g.observationErr
	}
	if g.escaped {
		return false, nil
	}
	err := syscall.Kill(-g.pid, 0)
	if errors.Is(err, syscall.ESRCH) {
		return true, nil
	}
	if err != nil && !errors.Is(err, syscall.EPERM) {
		return false, err
	}
	processes, err := observeProcesses()
	if err != nil {
		return false, err
	}
	for _, p := range processes {
		if p.group == g.pid {
			return false, nil
		}
	}
	return true, nil
}

func (g *posixProcessGroup) sameLeader() error {
	if g.identity == nil && !g.exitedAtStart {
		return errProcessGroupIdentity
	}
	current, err := ObserveProcessCorrelation(int64(g.pid))
	if err != nil {
		return err
	}
	if current != nil && (g.identity == nil || *current != *g.identity) {
		return errProcessGroupIdentity
	}
	return nil
}

type processObservation struct{ pid, parent, group int }

// observeEscapes follows only ancestry proven from the live leader; it never
// attributes unrelated host processes to the attempt.
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
