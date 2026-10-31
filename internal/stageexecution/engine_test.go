//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

const engineExecutionID = "00000000-0000-4000-8000-0000000000e1"

type fakeRunner struct {
	spawns atomic.Int32
	run    func(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult
}

func (f *fakeRunner) RunObserved(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	f.spawns.Add(1)
	return f.run(ctx, inv, started)
}

func succeed(output string) func(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	return func(_ context.Context, _ executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		if err := started(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
			return executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true, Stopped: true}
		}
		return executiongraph.ProcessResult{ExitCode: 0, Output: []byte(output)}
	}
}

// stubLauncher declares an output only when the process printed "OUTPUT <name>".
type stubLauncher struct {
	argv     []string
	buildErr error
}

func (s stubLauncher) Build(context.Context, LaunchRequest) (executiongraph.Invocation, error) {
	if s.buildErr != nil {
		return executiongraph.Invocation{}, s.buildErr
	}
	argv := s.argv
	if argv == nil {
		argv = []string{"/bin/true"}
	}
	return executiongraph.Invocation{RuntimeID: "stub", Argv: argv, OutputMax: 4096}, nil
}

func (s stubLauncher) Collect(_ context.Context, r LaunchRequest, p executiongraph.ProcessResult) (Outcome, error) {
	var out Outcome
	for _, line := range strings.Split(string(p.Output), "\n") {
		if name, ok := strings.CutPrefix(line, "OUTPUT "); ok {
			out.Artifacts = append(out.Artifacts, workflow.ArtifactRef{Name: name, RepositoryKey: r.Attempt.Authority.RepositoryKey, Digest: workflow.DigestText(p.Output), Size: int64(len(p.Output))})
		}
		if target, ok := strings.CutPrefix(line, "EFFECT "); ok {
			out.Effects = append(out.Effects, workflow.EffectFact{Kind: "workspace_write", Target: target, State: workflow.EffectConfirmed})
		}
	}
	return out, nil
}

func stubInvocation() workflow.Invocation {
	return workflow.Invocation{Runtime: "stub", Executable: "stub-bin", Model: "stub-model", Effort: "runtime-default", Instructions: "work", InstructionsDigest: workflow.DigestText([]byte("work")),
		Provenance: []workflow.FieldProvenance{{Field: "runtime", Source: "local_default"}, {Field: "model", Source: "local_default"}, {Field: "effort", Source: "local_default"}, {Field: "instructions", Source: "workflow_agent"}}}
}

type engineFixture struct {
	engine Engine
	runner *fakeRunner
	root   string
}

func newEngineFixture(t *testing.T, stages []workflow.StageSpec, run func(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult) *engineFixture {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := local.NewExecutionStore(root)
	if err != nil {
		t.Fatal(err)
	}
	x, err := workflow.NewExecution(workflow.NewExecutionInput{
		ExecutionID: engineExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"},
		Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/app"}}, WorkflowID: "minimal", Revision: "r1", Now: time.Now(), Stages: stages,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), x); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{run: run}
	var n atomic.Int32
	engine := Engine{
		Store: store, Runner: runner, Launcher: stubLauncher{},
		Resolve: func(_ context.Context, x workflow.Execution, ref workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
			effects := []string{}
			for _, s := range x.Workflow.Stages {
				for _, a := range s.Agents {
					if s.ID == ref.StageID && a.ID == ref.AgentID && a.Effects != nil {
						effects = a.Effects
					}
				}
			}
			return stubInvocation(), workflow.AttemptAuthority{RepositoryKey: "app", Effects: effects, Actor: "user:local"}, nil
		},
		NewID:      func() (string, error) { return fmt.Sprintf("att-%d", n.Add(1)), nil },
		CancelPoll: 10 * time.Millisecond,
	}
	return &engineFixture{engine: engine, runner: runner, root: root}
}

func twoStages(gate bool) []workflow.StageSpec {
	return []workflow.StageSpec{
		{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder", Effects: []string{"workspace_write"}}}, Outputs: []string{"result"}, HumanGate: gate, TimeoutSeconds: 5},
		{ID: "review", DependsOn: []string{"work"}, Agents: []workflow.AgentSpec{{ID: "reviewer"}}, TimeoutSeconds: 5},
	}
}

func (f *engineFixture) run(t *testing.T) (workflow.Execution, error) {
	t.Helper()
	return f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{})
}

func TestEngineProgressesTwoStagesWithoutPlanPolicyProfileOrLegacyStores(t *testing.T) {
	// Only the builder is authorized for workspace_write; the reviewer
	// reports no effect (an unauthorized report would become unknown).
	outputs := []string{"OUTPUT result\nEFFECT app\n", "reviewed\n"}
	var call atomic.Int32
	f := newEngineFixture(t, twoStages(false), func(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		return succeed(outputs[call.Add(1)-1])(ctx, inv, started)
	})
	x, err := f.run(t)
	if err != nil {
		t.Fatal(err)
	}
	if x.Status != workflow.RunCompleted || f.runner.spawns.Load() != 2 {
		t.Fatalf("status %s spawns %d", x.Status, f.runner.spawns.Load())
	}
	a, _, _ := x.FindAttempt("att-1")
	if a.Process == nil || a.Process.PID != 777 || len(a.Effects) != 1 || a.Invocation.Profile != "" {
		t.Fatalf("attempt facts %+v", a)
	}
	entries, _ := os.ReadDir(f.root)
	for _, entry := range entries {
		if name := entry.Name(); name != "executions" && name != "run-leases" {
			t.Fatalf("new run created legacy state %q", name)
		}
	}
	if _, err := os.Stat(filepath.Join(f.root, "executions", "v1")); !os.IsNotExist(err) {
		t.Fatal("executions/v1 ledger created")
	}
}

func TestEngineStopsSuccessorsOnMissingOutputGateAndSpawnFailure(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed("no declared output\n"))
	x, err := f.run(t)
	if err != nil || x.Status != workflow.RunFailed || f.runner.spawns.Load() != 1 {
		t.Fatalf("missing output: %v %s %d", err, x.Status, f.runner.spawns.Load())
	}

	g := newEngineFixture(t, twoStages(true), succeed("OUTPUT result\n"))
	x, err = g.run(t)
	if err != nil || x.Status != workflow.RunAwaitingGate || g.runner.spawns.Load() != 1 {
		t.Fatalf("gate: %v %s %d", err, x.Status, g.runner.spawns.Load())
	}
	if _, err := g.engine.Store.UpdateFenced(context.Background(), "proj", engineExecutionID, nil, func(x *workflow.Execution) error {
		return x.ApproveGate("work", "human:maintainer", time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	if x, err = g.run(t); err != nil || x.Status != workflow.RunCompleted {
		t.Fatalf("after human gate: %v %s", err, x.Status)
	}

	s := newEngineFixture(t, twoStages(false), func(context.Context, executiongraph.Invocation, func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		return executiongraph.ProcessResult{ExitCode: -1, SpawnFailed: true, FailureCode: "executable_unavailable"}
	})
	x, err = s.run(t)
	a, _, _ := x.FindAttempt("att-1")
	if err != nil || x.Status != workflow.RunFailed || !strings.Contains(a.Diagnostic, "executable_unavailable") || s.runner.spawns.Load() != 1 {
		t.Fatalf("spawn failure must be an observed failure without fallback: %v %s %+v", err, x.Status, a)
	}

	b := newEngineFixture(t, twoStages(false), succeed(""))
	b.engine.Launcher = stubLauncher{buildErr: errors.New("effort_override_conflict")}
	x, err = b.run(t)
	if err != nil || x.Status != workflow.RunFailed || b.runner.spawns.Load() != 0 {
		t.Fatalf("selector refusal spawned: %v %s %d", err, x.Status, b.runner.spawns.Load())
	}
}

func TestEngineUnauthorizedEffectReportBecomesUnknownAndStops(t *testing.T) {
	stages := []workflow.StageSpec{
		{ID: "work", Agents: []workflow.AgentSpec{{ID: "reader"}}, TimeoutSeconds: 5},
		{ID: "next", DependsOn: []string{"work"}, Agents: []workflow.AgentSpec{{ID: "later"}}, TimeoutSeconds: 5},
	}
	f := newEngineFixture(t, stages, succeed("EFFECT app\n"))
	x, err := f.run(t)
	a, _, _ := x.FindAttempt("att-1")
	if err != nil || a.Status != workflow.AttemptUnknown || a.Diagnostic != "unauthorized_effect_reported" || x.Status != workflow.RunRecoveryRequired || f.runner.spawns.Load() != 1 {
		t.Fatalf("unauthorized effect: %v %s %q %s %d", err, a.Status, a.Diagnostic, x.Status, f.runner.spawns.Load())
	}
}

func TestEngineResolveRefusalRecordsNothing(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	f.engine.Resolve = func(context.Context, workflow.Execution, workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, errors.New("invocation_selection_missing: effort")
	}
	before, _ := f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
	if _, err := f.run(t); err == nil {
		t.Fatal("missing selection dispatched")
	}
	after, _ := f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
	if f.runner.spawns.Load() != 0 || len(after.Stages[0].Tasks[0].Attempts) != 0 || after.Revision != before.Revision+1 {
		t.Fatalf("refusal recorded an attempt: %+v", after.Stages[0])
	}
}

func TestEngineCrashWindowsBecomeUnknownAndNeverRespawn(t *testing.T) {
	for _, started := range []bool{false, true} {
		f := newEngineFixture(t, twoStages(false), succeed("OUTPUT result\n"))
		ctx := context.Background()
		// A first runner prepares (and possibly publishes the spawn) and dies.
		lease, _, err := f.engine.Claim(ctx, "proj", engineExecutionID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.engine.Store.UpdateFenced(ctx, "proj", engineExecutionID, lease, func(x *workflow.Execution) error {
			if err := x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "crashed", Generation: lease.Generation(), Invocation: stubInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: time.Now()}); err != nil {
				return err
			}
			if started {
				return x.MarkStarted("crashed", lease.Generation(), workflow.ProcessIdentity{PID: 999999, StartIdentity: "gone", GroupIdentity: "999999"}, time.Now())
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		lease.Close()
		x, err := f.run(t)
		if err != nil {
			t.Fatal(err)
		}
		a, _, _ := x.FindAttempt("crashed")
		if x.Status != workflow.RunRecoveryRequired || a.Status != workflow.AttemptUnknown || f.runner.spawns.Load() != 0 {
			t.Fatalf("started=%t: status %s attempt %s spawns %d", started, x.Status, a.Status, f.runner.spawns.Load())
		}
		if _, err := f.engine.Run(ctx, "proj", engineExecutionID, RunOptions{Retry: &workflow.TaskRef{StageID: "work", AgentID: "builder"}}); !errors.Is(err, workflow.ErrUnknownOutstanding) {
			t.Fatalf("blind retry over unknown effect: %v", err)
		}
		if f.runner.spawns.Load() != 0 {
			t.Fatal("unknown attempt respawned")
		}
	}
}

func TestEngineRefusesSecondRunner(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed("OUTPUT result\n"))
	lease, _, err := f.engine.Claim(context.Background(), "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err := f.run(t); !errors.Is(err, ErrRunnerActive) || f.runner.spawns.Load() != 0 {
		t.Fatalf("second runner: %v spawns %d", err, f.runner.spawns.Load())
	}
}

func TestEngineCancelBeforeSpawn(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed("OUTPUT result\n"))
	// The intent is persisted before the runner reaches its pre-spawn check.
	if _, err := f.engine.Cancel(context.Background(), "proj", engineExecutionID, "user:local"); err != nil {
		t.Fatal(err)
	}
	x, err := f.run(t)
	if err != nil || x.Status != workflow.RunCancelled || f.runner.spawns.Load() != 0 {
		t.Fatalf("cancel before spawn: %v %s %d", err, x.Status, f.runner.spawns.Load())
	}
}

func osEngine(t *testing.T, script string, timeout int64) *engineFixture {
	t.Helper()
	stages := []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder"}}, TimeoutSeconds: timeout}}
	f := newEngineFixture(t, stages, nil)
	f.engine.Runner = executiongraph.OSProcessRunner{}
	f.engine.Launcher = stubLauncher{argv: []string{"/bin/sh", "-c", script}}
	return f
}

func TestEngineCancelDuringRunStopsOwnedProcessGroup(t *testing.T) {
	f := osEngine(t, "sleep 30 & wait", 20)
	done := make(chan struct{})
	var x workflow.Execution
	var err error
	go func() { x, err = f.run(t); close(done) }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		current, _ := f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
		if a, _, ok := current.FindAttempt("att-1"); ok && a.Status == workflow.AttemptRunning {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("attempt never started")
		}
		time.Sleep(10 * time.Millisecond)
	}
	start := time.Now()
	if _, err := f.engine.Cancel(context.Background(), "proj", engineExecutionID, "user:local"); err != nil {
		t.Fatal(err)
	}
	<-done
	if err != nil {
		t.Fatal(err)
	}
	a, _, _ := x.FindAttempt("att-1")
	if a.Status != workflow.AttemptCancelled || a.Diagnostic != "group_stop_confirmed" || time.Since(start) > 8*time.Second {
		t.Fatalf("cancel during run: %s %q after %s", a.Status, a.Diagnostic, time.Since(start))
	}
	if observeRecorded(a.Process) != "process_gone_effects_unknown" {
		t.Fatal("owned process still observable after confirmed stop")
	}
}

func TestEngineTimeoutStopsProcessAndFails(t *testing.T) {
	f := osEngine(t, "sleep 30", 1)
	x, err := f.run(t)
	if err != nil {
		t.Fatal(err)
	}
	a, _, _ := x.FindAttempt("att-1")
	if a.Status != workflow.AttemptFailed || a.Diagnostic != "attempt_timeout" || x.Status != workflow.RunFailed {
		t.Fatalf("timeout: %s %q %s", a.Status, a.Diagnostic, x.Status)
	}
}

func TestEngineReconcileNeverSignalsAReusedPID(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("recorded group identity format is boot:pgid on Linux only")
	}
	f := newEngineFixture(t, twoStages(false), succeed(""))
	ctx := context.Background()
	lease, _, err := f.engine.Claim(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	// The recorded identity names this live test process with a different
	// start identity: a reused PID. Reconcile must report, never signal.
	self := int64(os.Getpid())
	if _, err := f.engine.Store.UpdateFenced(ctx, "proj", engineExecutionID, lease, func(x *workflow.Execution) error {
		if err := x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "old", Generation: lease.Generation(), Invocation: stubInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: time.Now()}); err != nil {
			return err
		}
		return x.MarkStarted("old", lease.Generation(), workflow.ProcessIdentity{PID: self, StartIdentity: "earlier-boot:1", GroupIdentity: "earlier-boot:1"}, time.Now())
	}); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	x, facts, err := f.engine.Reconcile(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts[0].Observation != "pid_reused_not_signalled" || x.Status != workflow.RunRecoveryRequired {
		t.Fatalf("reconcile: %+v %s", facts, x.Status)
	}
}

func TestEngineIngestRejectsMessagesForNonRunningAttempts(t *testing.T) {
	f := newEngineFixture(t, []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "codex"}, {ID: "claude"}}, TimeoutSeconds: 5}}, succeed(""))
	_, err := f.engine.Ingest(context.Background(), "proj", workflow.IngressContext{ExecutionID: engineExecutionID, AttemptID: "att-1", AgentID: "codex", Generation: 1}, workflow.MessageCandidate{ID: "m", Recipient: "claude", Kind: workflow.MessageQuestion})
	if !errors.Is(err, workflow.ErrMessageRejected) {
		t.Fatalf("message for a non-existent attempt: %v", err)
	}
}
