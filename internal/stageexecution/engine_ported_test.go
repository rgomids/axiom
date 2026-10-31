//go:build linux

package stageexecution

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// Invariants ported from the retired graph/single reconcile suites
// (ErrReconcileOrphanAlive, ErrReconcileLeaseHeld) onto the one engine.

func TestEngineReconcileReportsLiveOrphanWithoutSignalling(t *testing.T) {
	child := exec.Command("/bin/sh", "-c", "while :; do sleep 1; done")
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Kill(-child.Process.Pid, syscall.SIGKILL); _ = child.Wait() })
	identity, err := executiongraph.ObserveProcessCorrelation(int64(child.Process.Pid))
	if err != nil || identity == nil {
		t.Fatalf("observe child: %v", err)
	}
	f := newEngineFixture(t, twoStages(false), succeed(""))
	ctx := context.Background()
	lease, _, err := f.engine.Claim(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.engine.Store.UpdateFenced(ctx, "proj", engineExecutionID, lease, func(x *workflow.Execution) error {
		if err := x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "orphan", Generation: lease.Generation(), Invocation: stubInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: time.Now()}); err != nil {
			return err
		}
		return x.MarkStarted("orphan", lease.Generation(), workflow.ProcessIdentity{PID: identity.PID, StartIdentity: identity.StartIdentity, GroupIdentity: identity.GroupIdentity}, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	x, facts, err := f.engine.Reconcile(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts[0].Observation != "process_alive_effects_unknown" || x.Status != workflow.RunRecoveryRequired || f.runner.spawns.Load() != 0 {
		t.Fatalf("live orphan: %+v %s spawns=%d", facts, x.Status, f.runner.spawns.Load())
	}
	if err := syscall.Kill(child.Process.Pid, 0); err != nil {
		t.Fatalf("reconcile signalled the live orphan: %v", err)
	}
	// A later run never respawns the unknown attempt.
	if x, err = f.run(t); err != nil || x.Status != workflow.RunRecoveryRequired || f.runner.spawns.Load() != 0 {
		t.Fatalf("unknown attempt replayed: %v %s", err, x.Status)
	}
}

func TestEngineReconcileRefusedWhileRunnerHeld(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	ctx := context.Background()
	lease, before, err := f.engine.Claim(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, _, err := f.engine.Reconcile(ctx, "proj", engineExecutionID); !errors.Is(err, workflow.ErrRunnerHeld) {
		t.Fatalf("reconcile under a held lease: %v", err)
	}
	after, err := f.engine.Store.Load(ctx, "proj", engineExecutionID)
	if err != nil || after.Revision != before.Revision {
		t.Fatalf("refused reconcile mutated state: %v", err)
	}
}

func TestEngineRetryStopsAtAttemptLimit(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	if _, err := f.run(t); err != nil {
		t.Fatal(err)
	}
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}
	for i := 1; i < workflow.MaxTaskAttempts; i++ {
		if _, err := f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, WorkspaceDisposition: "preserve"}); err != nil {
			t.Fatalf("retry %d: %v", i, err)
		}
	}
	if _, err := f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, WorkspaceDisposition: "preserve"}); !errors.Is(err, workflow.ErrTransition) || f.runner.spawns.Load() != workflow.MaxTaskAttempts {
		t.Fatalf("attempt limit exceeded: %v spawns=%d", err, f.runner.spawns.Load())
	}
}

func TestEngineWritableRetryRequiresExplicitPreservation(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	if _, err := f.run(t); err != nil {
		t.Fatal(err)
	}
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}
	for _, disposition := range []string{"", "reset"} {
		_, err := f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, WorkspaceDisposition: disposition})
		if err == nil || f.runner.spawns.Load() != 1 {
			t.Fatalf("unsafe retry: %q %v", disposition, err)
		}
	}
	x, err := f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, WorkspaceDisposition: "preserve"})
	if err != nil || f.runner.spawns.Load() != 2 {
		t.Fatalf("preserve: %v", err)
	}
	a := x.Stages[0].Tasks[0].Attempts[1]
	if a.WorkspaceDisposition != "preserve" {
		t.Fatal("lost explicit disposition")
	}
}
