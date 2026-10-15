//go:build linux || darwin

package executiongraph

import (
	"context"
	"os"
	"strconv"
	"syscall"
	"testing"
)

func TestOSProcessRunnerOrphanedEscapeIsUnknown(t *testing.T) {
	pidfile := t.TempDir() + "/pid"
	result := (OSProcessRunner{}).Run(context.Background(), Invocation{
		Argv: []string{os.Args[0], "-test.run=^TestPOSIXProcessContainment$"},
		Env:  append(os.Environ(), "AXIOM_GROUP_HELPER=parent", "AXIOM_GROUP_ESCAPE=1", "AXIOM_GROUP_ORPHAN=1", "AXIOM_GROUP_PIDFILE="+pidfile),
		CWD:  t.TempDir(), OutputMax: 1024,
	})
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
		t.Fatalf("escapee absent: %+v, %v", descendant, err)
	}
	t.Cleanup(func() {
		current, err := ObserveProcessCorrelation(pid)
		if err == nil && current != nil && *current == *descendant {
			_ = syscall.Kill(int(pid), syscall.SIGKILL)
		}
	})
	if result.ExitCode != 0 || result.Stopped || !result.OutcomeUnknown || !result.EffectAmbiguous {
		t.Fatalf("escaped descendant misclassified: %+v", result)
	}
	status, reason, _ := ClassifyAttempt(result, true, AttemptCorrelation{}, nil)
	if status != AttemptUnknown || reason != ProcessOutcomeUnknown {
		t.Fatalf("escape classified as %s/%s", status, reason)
	}
}
