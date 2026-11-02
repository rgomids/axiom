//go:build linux

package stageexecution

import (
	"context"
	"errors"
	"os"
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
	child := exec.Command("/bin/sh", "-c", boundedHang)
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

// liveGroup starts a /bin/sh leader in its own process group and returns its
// recorded identity and a function that kills and reaps the whole group.
func liveGroup(t *testing.T) (workflow.ProcessIdentity, func()) {
	t.Helper()
	child := exec.Command("/bin/sh", "-c", boundedHang)
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	reaped := false
	kill := func() {
		if !reaped {
			reaped = true
			_ = syscall.Kill(-child.Process.Pid, syscall.SIGKILL)
			_ = child.Wait()
		}
	}
	t.Cleanup(kill)
	identity, err := executiongraph.ObserveProcessCorrelation(int64(child.Process.Pid))
	if err != nil || identity == nil {
		t.Fatalf("observe child: %v", err)
	}
	return workflow.ProcessIdentity{PID: identity.PID, StartIdentity: identity.StartIdentity, GroupIdentity: identity.GroupIdentity}, kill
}

// orphanUnknown records an attempt started by a runner that then died, and
// reconciles it to unknown, as after SIGKILL/OOM of the runner.
func orphanUnknown(t *testing.T, f *engineFixture, process workflow.ProcessIdentity) workflow.Execution {
	t.Helper()
	ctx := context.Background()
	lease, _, err := f.engine.Claim(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.engine.Store.UpdateFenced(ctx, "proj", engineExecutionID, lease, func(x *workflow.Execution) error {
		if err := x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "orphan", Generation: lease.Generation(), Invocation: stubInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: time.Now()}); err != nil {
			return err
		}
		return x.MarkStarted("orphan", lease.Generation(), process, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	x, _, err := f.engine.Reconcile(ctx, "proj", engineExecutionID)
	if err != nil || x.Status != workflow.RunRecoveryRequired {
		t.Fatalf("reconcile: %v %s", err, x.Status)
	}
	return x
}

func hasObservation(facts []ReconcileFact, want string) bool {
	for _, f := range facts {
		if f.Observation == want {
			return true
		}
	}
	return false
}

// IR-01: abandonment (and therefore retry) is refused while the orphan lives;
// after its group is verifiably gone exactly one explicit retry spawns.
func TestEngineAbandonRefusesLiveOrphanThenAllowsOneRetry(t *testing.T) {
	process, kill := liveGroup(t)
	f := newEngineFixture(t, twoStages(false), succeed(""))
	ctx := context.Background()
	reconciled := orphanUnknown(t, f, process)
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}

	x, facts, err := f.engine.Abandon(ctx, "proj", engineExecutionID, "orphan", "human:operator")
	if !errors.Is(err, ErrOrphanLiveness) || !hasObservation(facts, "process_alive_effects_unknown") {
		t.Fatalf("abandon beside a live orphan: %v %+v", err, facts)
	}
	after, err := f.engine.Store.Load(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if a, _, _ := after.FindAttempt("orphan"); after.Revision != reconciled.Revision || a.Resolution != nil || x.Revision != reconciled.Revision {
		t.Fatalf("refused abandon wrote state: revision %d->%d resolution %+v", reconciled.Revision, after.Revision, a.Resolution)
	}
	if err := syscall.Kill(int(process.PID), 0); err != nil {
		t.Fatalf("abandon signalled the orphan: %v", err)
	}
	if _, err := f.engine.Run(ctx, "proj", engineExecutionID, RunOptions{Retry: &ref}); !errors.Is(err, workflow.ErrUnknownOutstanding) || f.runner.spawns.Load() != 0 {
		t.Fatalf("retry beside a live orphan: %v spawns=%d", err, f.runner.spawns.Load())
	}

	kill()
	x, facts, err = f.engine.Abandon(ctx, "proj", engineExecutionID, "orphan", "human:operator")
	if err != nil || !hasObservation(facts, "process_gone_effects_unknown") {
		t.Fatalf("abandon after a confirmed stop: %v %+v", err, facts)
	}
	if a, _, _ := x.FindAttempt("orphan"); a.Resolution == nil || a.Status != workflow.AttemptUnknown {
		t.Fatalf("abandon: %+v", a)
	}
	if _, err := f.engine.Run(ctx, "proj", engineExecutionID, RunOptions{Retry: &ref}); err != nil || f.runner.spawns.Load() != 1 {
		t.Fatalf("retry after abandon: %v spawns=%d", err, f.runner.spawns.Load())
	}
}

func TestEngineAbandonRefusesWhileRunnerHeld(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	ctx := context.Background()
	reconciled := orphanUnknown(t, f, workflow.ProcessIdentity{PID: 999999, StartIdentity: "gone", GroupIdentity: "gone"})
	lease, err := f.engine.Store.AcquireFence(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, _, err := f.engine.Abandon(ctx, "proj", engineExecutionID, "orphan", "human:operator"); !errors.Is(err, workflow.ErrRunnerHeld) {
		t.Fatalf("abandon under a held runner lease: %v", err)
	}
	if after, _ := f.engine.Store.Load(ctx, "proj", engineExecutionID); after.Revision != reconciled.Revision {
		t.Fatal("refused abandon wrote state")
	}
}

func TestEngineAbandonRefusesFailedObservation(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	reconciled := orphanUnknown(t, f, workflow.ProcessIdentity{PID: 999999, StartIdentity: "gone", GroupIdentity: "gone"})
	original := observeRecordedProcess
	observeRecordedProcess = func(executiongraph.ProcessCorrelation) (*executiongraph.ProcessCorrelation, error) {
		return nil, errors.New("proc unreadable")
	}
	t.Cleanup(func() { observeRecordedProcess = original })
	_, facts, err := f.engine.Abandon(context.Background(), "proj", engineExecutionID, "orphan", "human:operator")
	if !errors.Is(err, ErrOrphanLiveness) || !hasObservation(facts, "observation_failed") {
		t.Fatalf("abandon with unobservable liveness: %v %+v", err, facts)
	}
	if after, _ := f.engine.Store.Load(context.Background(), "proj", engineExecutionID); after.Revision != reconciled.Revision {
		t.Fatal("refused abandon wrote state")
	}
}

func TestEngineAbandonReportsReusedPIDWithoutSignalling(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	// The recorded identity names this live test process with another start
	// identity: a reused PID is safe to abandon, reported and never touched.
	orphanUnknown(t, f, workflow.ProcessIdentity{PID: int64(os.Getpid()), StartIdentity: "earlier-boot:1", GroupIdentity: "earlier-boot:1"})
	x, facts, err := f.engine.Abandon(context.Background(), "proj", engineExecutionID, "orphan", "human:operator")
	if err != nil || !hasObservation(facts, "pid_reused_not_signalled") {
		t.Fatalf("reused PID: %v %+v", err, facts)
	}
	if a, _, _ := x.FindAttempt("orphan"); a.Resolution == nil {
		t.Fatal("reused PID abandon not recorded")
	}
}

func TestEngineAbandonRefusesLiveUnfinishedCommand(t *testing.T) {
	command, _ := liveGroup(t)
	check := workflow.CommandCheck{ID: "check", Argv: []string{"/bin/true"}, TimeoutSeconds: 5, Effects: []string{"validation_command"}}
	stages := []workflow.StageSpec{{ID: "work", Commands: []workflow.CommandCheck{check}, Agents: []workflow.AgentSpec{{ID: "builder", Effects: []string{"validation_command"}}}, TimeoutSeconds: 5}}
	f := newEngineFixture(t, stages, succeed(""))
	ctx := context.Background()
	lease, _, err := f.engine.Claim(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	gen := lease.Generation()
	// The agent leader's PID was reused (safe on its own), but its validation
	// command survived the runner.
	if _, err := f.engine.Store.UpdateFenced(ctx, "proj", engineExecutionID, lease, func(x *workflow.Execution) error {
		if err := x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "orphan", Generation: gen, Invocation: stubInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{"validation_command"}, Actor: "user:local"}, Now: time.Now()}); err != nil {
			return err
		}
		if err := x.MarkStarted("orphan", gen, workflow.ProcessIdentity{PID: int64(os.Getpid()), StartIdentity: "earlier-boot:1", GroupIdentity: "earlier-boot:1"}, time.Now()); err != nil {
			return err
		}
		if err := x.PrepareCommand("orphan", gen, "check", time.Now()); err != nil {
			return err
		}
		return x.StartCommand("orphan", gen, "check", command, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	if _, _, err := f.engine.Reconcile(ctx, "proj", engineExecutionID); err != nil {
		t.Fatal(err)
	}
	_, facts, err := f.engine.Abandon(ctx, "proj", engineExecutionID, "orphan", "human:operator")
	if !errors.Is(err, ErrOrphanLiveness) || !hasObservation(facts, "pid_reused_not_signalled") || !hasObservation(facts, "command:check:process_alive_effects_unknown") {
		t.Fatalf("abandon beside a live command: %v %+v", err, facts)
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
