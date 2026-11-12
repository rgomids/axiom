//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/executiongraph"
)

// CR-S1-07 final: stub agents must not outlive a dead test binary. The
// observer re-executes this test binary as a victim that runs a foreground
// runner on a hanging stub agent, then signals, fails or times the victim out.
// From outside the victim it proves the runner and every process of the stub's
// own group are gone. SIGKILL and -test.timeout run no cleanup hook, so the
// proof never relies on one; an emergency stop by the observer is a FAIL.

const (
	leakVictimMode   = "LINGO_TEST_LEAK_VICTIM"
	leakVictimReport = "LINGO_TEST_LEAK_REPORT"
	leakVictimBinary = "LINGO_TEST_LEAK_BINARY"
	leakSettleBound  = 10 * time.Second
)

type leakReport struct{ Runner, Stub int64 }

func TestLeakVictimHelper(t *testing.T) {
	mode := os.Getenv(leakVictimMode)
	if mode == "" {
		t.Skip("victim process of TestKilledTestBinaryLeavesNoOwnedProcess")
	}
	f := newExecutionRunFixture(t)
	runner, stdout, _, pid := startHangingRunner(t, os.Getenv(leakVictimBinary), f)
	wire, err := json.Marshal(leakReport{Runner: int64(runner.Process.Pid), Stub: pid})
	if err != nil {
		t.Fatal(err)
	}
	report := os.Getenv(leakVictimReport)
	if err := os.WriteFile(report+".tmp", wire, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(report+".tmp", report); err != nil {
		t.Fatal(err)
	}
	// Proceed only once the observer has recorded the live family.
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(report + ".observed"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("observer never recorded the family")
		}
	}
	switch mode {
	case "complete":
		if err := runner.Process.Signal(syscall.SIGTERM); err != nil {
			t.Fatal(err)
		}
		var exit *exec.ExitError
		if err := runner.Wait(); !errors.As(err, &exit) || exit.ExitCode() != cli.ExitCancelled {
			t.Fatalf("runner exit: %v %s", err, stdout.String())
		}
	case "fail":
		// The path that leaked in the signal test: a failure before the stub
		// cleanup exists. Only the runner cleanup (SIGKILL) runs.
		t.Fatal("forced failure before any stub cleanup is registered")
	default:
		time.Sleep(60 * time.Second)
		t.Fatal("victim was not stopped")
	}
}

type ownedProcess struct {
	role     string
	identity executiongraph.ProcessCorrelation
}

func TestKilledTestBinaryLeavesNoOwnedProcess(t *testing.T) {
	binary := buildLingoBinary(t)
	cases := []struct {
		name, mode, timeout string
		signal              syscall.Signal
		wantExit            int
	}{
		{name: "SIGINT", mode: "kill", signal: syscall.SIGINT, wantExit: -1},
		{name: "SIGTERM", mode: "kill", signal: syscall.SIGTERM, wantExit: -1},
		{name: "SIGHUP", mode: "kill", signal: syscall.SIGHUP, wantExit: -1},
		{name: "SIGKILL", mode: "kill", signal: syscall.SIGKILL, wantExit: -1},
		{name: "complete", mode: "complete", wantExit: 0},
		{name: "forced_failure", mode: "fail", wantExit: 1},
		{name: "test_timeout", mode: "kill", timeout: "12s", wantExit: 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			report := filepath.Join(dir, "report.json")
			timeout := c.timeout
			if timeout == "" {
				timeout = "90s"
			}
			victim := leakVictimCommand(os.Args[0], "-test.run=^TestLeakVictimHelper$", "-test.count=1", "-test.timeout="+timeout)
			victim.Env = append(os.Environ(), leakVictimMode+"="+c.mode, leakVictimReport+"="+report, leakVictimBinary+"="+binary, "TMPDIR="+dir)
			output := &limitedOutput{max: 16 << 10}
			victim.Stdout, victim.Stderr = output, output
			// The victim's group is owned by this test: its leader is an
			// unreaped child, so the group ID cannot be reused while it runs.
			victim.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
			if err := victim.Start(); err != nil {
				t.Fatal(err)
			}
			exited := make(chan error, 1)
			go func() { exited <- victim.Wait() }()
			stopVictim := func() {
				select {
				case err := <-exited:
					exited <- err
				default:
					_ = syscall.Kill(-victim.Process.Pid, syscall.SIGKILL)
				}
			}
			t.Cleanup(stopVictim)
			rep, err := awaitLeakReport(report, exited)
			if err != nil {
				t.Fatalf("victim never reported its runner: %v\n%s", err, output)
			}
			family := observeOwnedFamily(t, int64(victim.Process.Pid), rep)
			t.Logf("observed alive before %s: %s", c.name, describeFamily(family))
			if err := os.WriteFile(report+".observed", nil, 0o600); err != nil {
				emergencyStop(family, rep.Stub)
				t.Fatal(err)
			}
			if c.signal != 0 {
				if err := syscall.Kill(victim.Process.Pid, c.signal); err != nil {
					t.Fatal(err)
				}
			}
			// Leave time for the independent no-survivor check before the test deadline.
			exitDeadline := time.Now().Add(2 * time.Minute)
			if deadline, ok := t.Deadline(); ok && deadline.Add(-leakSettleBound).Before(exitDeadline) {
				exitDeadline = deadline.Add(-leakSettleBound)
			}
			var exitErr error
			select {
			case exitErr = <-exited:
			case <-time.After(time.Until(exitDeadline)):
				identity, observeErr := executiongraph.ObserveProcessCorrelation(int64(victim.Process.Pid))
				stopVictim()
				t.Fatalf("victim wait exceeded test budget (alive=%t, observation error=%v); owned processes: %s", identity != nil, observeErr, strings.Join(leftoverOwned(family, rep.Stub), ", "))
			}
			if got := victimExit(exitErr); got != c.wantExit {
				t.Fatalf("victim exit %d (%v), want %d\n%s", got, exitErr, c.wantExit, output)
			}
			start := time.Now()
			left := leftoverOwned(family, rep.Stub)
			for ; len(left) != 0 && time.Since(start) < leakSettleBound; left = leftoverOwned(family, rep.Stub) {
				time.Sleep(50 * time.Millisecond)
			}
			if len(left) != 0 {
				emergencyStop(family, rep.Stub)
				t.Fatalf("owned processes survived the victim for %s: %s (observer emergency stop performed; this is a leak)", leakSettleBound, strings.Join(left, ", "))
			}
			t.Logf("victim exit %d; no owned process left after %s", victimExit(exitErr), time.Since(start).Round(time.Millisecond))
		})
	}
}

// leakVictimCommand starts name with SIGHUP and SIGINT at their default
// disposition. #389: an inherited SIG_IGN (nohup, detached runner) survives
// exec and Go keeps it for those two signals, so the victim would never die of
// the signal under test. env resets them in the child and then execs, keeping
// the PID, so the observer never changes its own signal handling.
func leakVictimCommand(name string, args ...string) *exec.Cmd {
	return exec.Command("env", append([]string{"--default-signal=HUP,INT", name}, args...)...)
}

// TestLeakVictimLaunchKeepsObserverSignals proves the launch resets an
// inherited ignore in the child only: the observer's handlers and kernel
// dispositions are unchanged after a successful and a failed Start.
func TestLeakVictimLaunchKeepsObserverSignals(t *testing.T) {
	handled := make(chan os.Signal, 1)
	signal.Notify(handled, syscall.SIGHUP, syscall.SIGINT)
	defer signal.Stop(handled)
	before := signalDispositions(t)

	failed := leakVictimCommand("true")
	failed.Dir = filepath.Join(t.TempDir(), "missing")
	if err := failed.Start(); err == nil {
		_ = failed.Wait()
		t.Fatal("Start in a missing directory succeeded")
	}
	if err := leakVictimCommand("true").Run(); err != nil {
		t.Fatal(err)
	}
	// sh installs the ignore the child inherits, as nohup does for SIGHUP.
	ignoreThen := func(cmd ...string) string {
		out, err := exec.Command("sh", append([]string{"-c", `trap "" HUP INT; exec "$@"`, "sh"}, cmd...)...).Output()
		if err != nil {
			t.Fatalf("%v: %v", cmd, err)
		}
		return strings.TrimSpace(string(out))
	}
	const hupInt = 1<<(syscall.SIGHUP-1) | 1<<(syscall.SIGINT-1)
	for _, c := range []struct {
		cmd  []string
		want uint64
	}{
		{cmd: []string{"grep", "^SigIgn:", "/proc/self/status"}, want: hupInt},
		{cmd: leakVictimCommand("grep", "^SigIgn:", "/proc/self/status").Args, want: 0},
	} {
		ignored, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(ignoreThen(c.cmd...), "SigIgn:")), 16, 64)
		if err != nil || ignored&hupInt != c.want {
			t.Fatalf("%v: SigIgn %x (%v), want HUP/INT bits %x", c.cmd, ignored, err, c.want)
		}
	}

	if after := signalDispositions(t); after != before {
		t.Fatalf("observer dispositions changed: %s -> %s", before, after)
	}
	for _, sig := range []syscall.Signal{syscall.SIGHUP, syscall.SIGINT} {
		if err := syscall.Kill(os.Getpid(), sig); err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-handled:
			if got != sig {
				t.Fatalf("handler got %v, want %v", got, sig)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("observer handler no longer receives %v", sig)
		}
	}
}

// signalDispositions returns the kernel's ignored and caught signal masks.
func signalDispositions(t *testing.T) string {
	t.Helper()
	wire, err := os.ReadFile("/proc/self/status")
	if err != nil {
		t.Fatal(err)
	}
	var masks []string
	for _, line := range strings.Split(string(wire), "\n") {
		if strings.HasPrefix(line, "SigIgn:") || strings.HasPrefix(line, "SigCgt:") {
			masks = append(masks, strings.Join(strings.Fields(line), " "))
		}
	}
	return strings.Join(masks, "; ")
}

func awaitLeakReport(path string, exited chan error) (leakReport, error) {
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		if wire, err := os.ReadFile(path); err == nil {
			var rep leakReport
			if err := json.Unmarshal(wire, &rep); err != nil || rep.Runner <= 1 || rep.Stub <= 1 {
				return rep, fmt.Errorf("invalid report %q", wire)
			}
			return rep, nil
		}
		select {
		case err := <-exited:
			exited <- err
			return leakReport{}, fmt.Errorf("victim exited early: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
	}
	return leakReport{}, errors.New("no report")
}

// observeOwnedFamily proves the chain victim → runner → stub group leader, so
// the observer only ever tracks processes this test started.
func observeOwnedFamily(t *testing.T, victim int64, rep leakReport) []ownedProcess {
	t.Helper()
	procs, err := scanProcStat()
	if err != nil {
		t.Fatal(err)
	}
	byPID := map[int64]procStat{}
	for _, p := range procs {
		byPID[p.pid] = p
	}
	runner, stub := byPID[rep.Runner], byPID[rep.Stub]
	if runner.ppid != victim || stub.ppid != rep.Runner || stub.pgid != rep.Stub {
		t.Fatalf("ownership chain not proven: runner=%+v stub=%+v victim=%d", runner, stub, victim)
	}
	var family []ownedProcess
	add := func(role string, pid int64) {
		identity, err := executiongraph.ObserveProcessCorrelation(pid)
		if err != nil || identity == nil {
			t.Fatalf("%s %d not alive before the experiment: %v", role, pid, err)
		}
		family = append(family, ownedProcess{role: role, identity: *identity})
	}
	add("runner", rep.Runner)
	add("stub", rep.Stub)
	for _, p := range procs {
		if p.pgid == rep.Stub && p.pid != rep.Stub {
			if identity, err := executiongraph.ObserveProcessCorrelation(p.pid); err == nil && identity != nil {
				family = append(family, ownedProcess{role: "stub-group", identity: *identity})
			}
		}
	}
	return family
}

// leftoverOwned reports recorded identities still alive and any live member
// of the stub's own group (descendants that kept the group).
func leftoverOwned(family []ownedProcess, stubGroup int64) []string {
	var left []string
	for _, p := range family {
		if current, err := executiongraph.ObserveProcessCorrelation(p.identity.PID); err != nil || (current != nil && *current == p.identity) {
			left = append(left, fmt.Sprintf("%s pid=%d", p.role, p.identity.PID))
		}
	}
	procs, err := scanProcStat()
	if err != nil {
		return append(left, "process scan failed: "+err.Error())
	}
	for _, p := range procs {
		if p.pgid == stubGroup && p.state != "Z" {
			left = append(left, fmt.Sprintf("stub-group member pid=%d", p.pid))
		}
	}
	return left
}

// emergencyStop signals only exact recorded identities, re-observed
// immediately before each signal, and the stub group only through its
// still-identical leader.
func emergencyStop(family []ownedProcess, stubGroup int64) {
	for _, p := range family {
		current, err := executiongraph.ObserveProcessCorrelation(p.identity.PID)
		if err != nil || current == nil || *current != p.identity {
			continue
		}
		if p.identity.PID == stubGroup {
			_ = syscall.Kill(-int(stubGroup), syscall.SIGKILL)
		}
		_ = syscall.Kill(int(p.identity.PID), syscall.SIGKILL)
	}
}

func describeFamily(family []ownedProcess) string {
	parts := make([]string, 0, len(family))
	for _, p := range family {
		parts = append(parts, fmt.Sprintf("%s pid=%d start=%s", p.role, p.identity.PID, p.identity.StartIdentity[strings.LastIndex(p.identity.StartIdentity, ":")+1:]))
	}
	return strings.Join(parts, "; ")
}

func victimExit(err error) int {
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.ExitCode()
	default:
		return -2
	}
}

type procStat struct {
	pid, ppid, pgid int64
	state           string
}

func scanProcStat() ([]procStat, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var procs []procStat
	for _, e := range entries {
		pid, err := strconv.ParseInt(e.Name(), 10, 64)
		if err != nil {
			continue
		}
		wire, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		stat := string(wire)
		fields := strings.Fields(stat[strings.LastIndex(stat, ")")+1:])
		if len(fields) < 3 {
			continue
		}
		ppid, _ := strconv.ParseInt(fields[1], 10, 64)
		pgid, _ := strconv.ParseInt(fields[2], 10, 64)
		procs = append(procs, procStat{pid: pid, ppid: ppid, pgid: pgid, state: fields[0]})
	}
	return procs, nil
}

type limitedOutput struct {
	buf bytes.Buffer
	max int
}

func (l *limitedOutput) Write(p []byte) (int, error) {
	if room := l.max - l.buf.Len(); room > 0 {
		l.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (l *limitedOutput) String() string { return l.buf.String() }
