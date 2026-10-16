//go:build linux || darwin

package executiongraph

import (
	"context"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestOSProcessRunnerDecoupledOrphanOutsideGuarantee(t *testing.T) {
	pidfile := t.TempDir() + "/pid"
	result := (OSProcessRunner{}).Run(context.Background(), Invocation{
		Argv: []string{os.Args[0], "-test.run=^TestPOSIXProcessContainment$"},
		Env:  append(os.Environ(), "AXIOM_GROUP_HELPER=parent", "AXIOM_GROUP_ESCAPE=1", "AXIOM_GROUP_ORPHAN=1", "AXIOM_GROUP_PIDFILE="+pidfile),
		CWD:  t.TempDir(), OutputMax: 1024,
	})
	pid := observedHelper(t, pidfile)
	if result.ExitCode != 0 || result.Stopped || result.OutcomeUnknown || result.EffectAmbiguous {
		t.Fatalf("decoupled orphan changed the group result: %+v", result)
	}
	if syscall.Kill(pid, 0) != nil {
		t.Fatal("process outside the attempt group was signaled")
	}
	status, reason, _ := ClassifyAttempt(result, false, AttemptCorrelation{}, nil)
	if status != AttemptFailed || reason != ResultMissing {
		t.Fatalf("exit 0 without a recorded result classified as %s/%s", status, reason)
	}
}

func TestOSProcessRunnerCancelWithKnownEscapeIsUnknown(t *testing.T) {
	pidfile := t.TempDir() + "/pid"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(pidfile); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		cancel()
	}()
	result := (OSProcessRunner{}).Run(ctx, Invocation{
		Argv: []string{os.Args[0], "-test.run=^TestPOSIXProcessContainment$"},
		Env:  append(os.Environ(), "AXIOM_GROUP_HELPER=parent", "AXIOM_GROUP_ESCAPE=1", "AXIOM_GROUP_PIDFILE="+pidfile),
		CWD:  t.TempDir(), OutputMax: 1024,
	})
	pid := observedHelper(t, pidfile)
	if result.Stopped || !result.OutcomeUnknown {
		t.Fatalf("known escape confirmed as stopped: %+v", result)
	}
	if syscall.Kill(pid, 0) != nil {
		t.Fatal("escaped process outside the group was signaled")
	}
	status, reason, _ := ClassifyAttempt(result, true, AttemptCorrelation{}, nil)
	if status != AttemptUnknown || reason != ProcessOutcomeUnknown {
		t.Fatalf("known escape classified as %s/%s", status, reason)
	}
}

func observedHelper(t *testing.T, pidfile string) int {
	t.Helper()
	data, err := os.ReadFile(pidfile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	descendant, err := ObserveProcessCorrelation(pid)
	if err != nil || descendant == nil {
		t.Fatalf("helper absent: %+v, %v", descendant, err)
	}
	t.Cleanup(func() {
		current, err := ObserveProcessCorrelation(pid)
		if err == nil && current != nil && *current == *descendant {
			_ = syscall.Kill(int(pid), syscall.SIGKILL)
		}
	})
	return int(pid)
}
