//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestI404RestartPendingPreservesCapacityAndSnapshot(t *testing.T) {
	for _, ceiling := range []int{0, 1, 2} {
		t.Run(map[int]string{0: "absent", 1: "explicit1", 2: "explicit2"}[ceiling], func(t *testing.T) {
			wire, err := os.ReadFile("../workflow/testdata/i404/dag.json")
			if err != nil {
				t.Fatal(err)
			}
			x, err := workflow.DecodeExecution(wire)
			if err != nil {
				t.Fatal(err)
			}
			if ceiling != 0 {
				x.Workflow.Stages[0].Concurrency = &ceiling
				x, err = workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: x.ExecutionID, ProjectID: x.ProjectID, WorkItem: x.WorkItem, Repositories: x.Repositories, WorkflowID: x.Workflow.ID, Revision: x.Workflow.Revision, Stages: x.Workflow.Stages, Now: x.CreatedAt})
				if err != nil {
					t.Fatal(err)
				}
			}
			frozen := x.Workflow
			f, store := startFixture(t, "")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := store.Create(ctx, x); err != nil {
				t.Fatal(err)
			}
			restarted, err := local.NewExecutionStore(f.root)
			if err != nil {
				t.Fatal(err)
			}
			f.engine.Store = restarted
			started := make(chan string, 3)
			release := make(chan struct{})
			var active, maximum atomic.Int32
			f.runner.run = func(ctx context.Context, inv executiongraph.Invocation, commit func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
				if err := commit(&executiongraph.ProcessCorrelation{PID: 404, StartIdentity: "synthetic", GroupIdentity: "404"}); err != nil {
					return executiongraph.ProcessResult{OutcomeUnknown: true}
				}
				n := active.Add(1)
				defer active.Add(-1)
				for old := maximum.Load(); n > old && !maximum.CompareAndSwap(old, n); old = maximum.Load() {
				}
				id := inv.Argv[len(inv.Argv)-1]
				started <- id
				if id != "join" {
					select {
					case <-release:
					case <-ctx.Done():
						return executiongraph.ProcessResult{Stopped: true}
					}
				}
				return executiongraph.ProcessResult{}
			}
			f.engine.Launcher = taskLauncher{}
			type result struct {
				x   workflow.Execution
				err error
			}
			done := make(chan result, 1)
			go func() {
				got, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{})
				done <- result{got, err}
			}()
			if id := awaitTask(t, started); id != "a" && id != "b" {
				t.Fatal("join started before validated roots", id)
			}
			if ceiling == 2 {
				if id := awaitTask(t, started); id != "a" && id != "b" {
					t.Fatal("join started while roots held", id)
				}
			}
			close(release)
			got := <-done
			wantPeak := int32(1)
			if ceiling == 2 {
				wantPeak = 2
			}
			if got.err != nil || got.x.Status != workflow.RunCompleted || !reflect.DeepEqual(frozen, got.x.Workflow) || maximum.Load() != wantPeak || f.runner.spawns.Load() != 3 {
				t.Fatalf("restart changed dispatch/snapshot: err=%v status=%s peak=%d spawns=%d", got.err, got.x.Status, maximum.Load(), f.runner.spawns.Load())
			}
		})
	}
}

func TestI404RestartHistoricalMultiActiveBecomesUnknownWithoutReplay(t *testing.T) {
	f, store := startFixture(t, "")
	wire, err := os.ReadFile("../workflow/testdata/i404/multi-active.json")
	if err != nil {
		t.Fatal(err)
	}
	x, err := workflow.DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	frozen := x.Workflow
	dir := filepath.Join(f.root, "executions", "v2", x.ProjectID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, x.ExecutionID+".json"), wire, 0600); err != nil {
		t.Fatal(err)
	}
	restarted, err := local.NewExecutionStore(f.root)
	if err != nil {
		t.Fatal(err)
	}
	f.engine.Store = restarted
	ctx := context.Background()
	got, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{})
	if err != nil || got.Status != workflow.RunRecoveryRequired || !reflect.DeepEqual(frozen, got.Workflow) || f.runner.spawns.Load() != 0 {
		t.Fatalf("historical restart replayed or lost snapshot: %v %s spawns=%d", err, got.Status, f.runner.spawns.Load())
	}
	for _, id := range []string{"attempt-a", "attempt-b"} {
		a, _, ok := got.FindAttempt(id)
		if !ok || a.Status != workflow.AttemptUnknown || a.Resolution != nil {
			t.Fatalf("historical active %s not unknown", id)
		}
	}
	ref := workflow.TaskRef{StageID: "work", AgentID: "a"}
	if _, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{Retry: &ref}); !errors.Is(err, workflow.ErrUnknownOutstanding) || f.runner.spawns.Load() != 0 {
		t.Fatal("unknown effects allowed blind retry", err)
	}
	loaded, err := store.Load(ctx, x.ProjectID, x.ExecutionID)
	if err != nil || !reflect.DeepEqual(loaded.Workflow, frozen) || loaded.Workflow.Stages[0].Concurrency != nil {
		t.Fatal("recovery materialized historical capacity", err)
	}
}

// endedOwnedProcess starts and reaps a process group owned by this test and
// returns its real recorded identity, so liveness is observed, never assumed.
func endedOwnedProcess(t *testing.T) *workflow.ProcessIdentity {
	t.Helper()
	cmd := exec.Command("/bin/cat")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p, err := executiongraph.ObserveProcessCorrelation(int64(cmd.Process.Pid))
	stdin.Close()
	if werr := cmd.Wait(); err != nil || p == nil || werr != nil {
		t.Fatalf("owned process identity: %v %v %+v", err, werr, p)
	}
	return &workflow.ProcessIdentity{PID: p.PID, StartIdentity: p.StartIdentity, GroupIdentity: p.GroupIdentity}
}

// Derived from the frozen multi-active record (which is never rewritten): its
// synthetic identities stay unobservable, so only these real ended identities
// let the canonical Abandon qualify liveness before an explicit retry.
func TestI404RestartHistoricalResolvedWithQualifiedLivenessPermitsExplicitRetry(t *testing.T) {
	const corpus = "../workflow/testdata/i404/multi-active.json"
	wire, err := os.ReadFile(corpus)
	if err != nil {
		t.Fatal(err)
	}
	historical, err := workflow.DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	x, err := workflow.DecodeExecution(wire) // independent copy for the derived case
	if err != nil {
		t.Fatal(err)
	}
	old := map[string]workflow.Attempt{}
	for _, id := range []string{"attempt-a", "attempt-b"} {
		a, _, _ := x.FindAttempt(id)
		a.Process = endedOwnedProcess(t)
		old[id] = *a
	}
	derived, err := workflow.EncodeExecution(x)
	if err != nil {
		t.Fatal(err)
	}
	f, store := startFixture(t, "")
	dir := filepath.Join(f.root, "executions", "v2", x.ProjectID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, x.ExecutionID+".json"), derived, 0600); err != nil {
		t.Fatal(err)
	}
	frozen := historical.Workflow
	var spawned []string
	var active, peak atomic.Int32
	f.runner.run = func(ctx context.Context, inv executiongraph.Invocation, commit func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		if err := commit(&executiongraph.ProcessCorrelation{PID: 404, StartIdentity: "synthetic", GroupIdentity: "404"}); err != nil {
			return executiongraph.ProcessResult{OutcomeUnknown: true}
		}
		n := active.Add(1)
		defer active.Add(-1)
		for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
		}
		spawned = append(spawned, inv.Argv[len(inv.Argv)-1])
		return executiongraph.ProcessResult{}
	}
	f.engine.Launcher = taskLauncher{}
	ctx := context.Background()
	check := func(step string, got workflow.Execution) {
		t.Helper()
		if !reflect.DeepEqual(got.Workflow, frozen) || got.Workflow.Digest != frozen.Digest || got.Workflow.Stages[0].Concurrency != nil || len(got.Integrations) != 0 || len(got.Deliveries) != 0 {
			t.Fatalf("%s changed the frozen Workflow or recorded an effect", step)
		}
		for id, before := range old {
			a, _, ok := got.FindAttempt(id)
			if !ok || a.Status != workflow.AttemptUnknown || a.RunnerGeneration != 1 || !reflect.DeepEqual(a.Invocation, before.Invocation) || !reflect.DeepEqual(a.Process, before.Process) {
				t.Fatalf("%s lost uncertain history of %s: %+v", step, id, a)
			}
		}
	}

	claimed, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{})
	if err != nil || claimed.Status != workflow.RunRecoveryRequired || f.runner.spawns.Load() != 0 {
		t.Fatalf("claim: %v %s spawns=%d", err, claimed.Status, f.runner.spawns.Load())
	}
	check("claim", claimed)
	a := workflow.TaskRef{StageID: "work", AgentID: "a"}
	b := workflow.TaskRef{StageID: "work", AgentID: "b"}
	if _, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{Retry: &a}); !errors.Is(err, workflow.ErrUnknownOutstanding) {
		t.Fatal("blind retry before resolution", err)
	}
	resolvedA, facts, err := f.engine.Abandon(ctx, x.ProjectID, x.ExecutionID, "attempt-a", "human:operator")
	if err != nil || len(facts) != 2 {
		t.Fatalf("qualified abandonment refused: %v %+v", err, facts)
	}
	for _, fact := range facts {
		if fact.Observation != "process_gone_effects_unknown" && fact.Observation != "pid_reused_not_signalled" {
			t.Fatalf("unqualified liveness fact: %+v", fact)
		}
	}
	check("abandon a", resolvedA)
	if _, _, err := f.engine.Abandon(ctx, x.ProjectID, x.ExecutionID, "attempt-a", "human:operator"); !errors.Is(err, workflow.ErrTransition) {
		t.Fatal("abandonment recorded twice", err)
	}
	// One unresolved peer still forbids every retry.
	if _, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{Retry: &a}); !errors.Is(err, workflow.ErrUnknownOutstanding) || f.runner.spawns.Load() != 0 {
		t.Fatal("retry beside an unresolved unknown", err)
	}
	if _, _, err := f.engine.Abandon(ctx, x.ProjectID, x.ExecutionID, "attempt-b", "human:operator"); err != nil {
		t.Fatal(err)
	}
	retried, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{Retry: &a})
	if err != nil || f.runner.spawns.Load() != 1 || !reflect.DeepEqual(spawned, []string{"a"}) {
		t.Fatalf("explicit retry: %v spawned=%v", err, spawned)
	}
	check("retry a", retried)
	task := retried.Stages[0].Tasks[0]
	fresh := task.Attempts[len(task.Attempts)-1]
	if len(task.Attempts) != 2 || fresh.AttemptID == "attempt-a" || fresh.Number != 2 || fresh.Status != workflow.AttemptSucceeded || fresh.RunnerGeneration <= 1 || !reflect.DeepEqual(fresh.Invocation, old["attempt-a"].Invocation) {
		t.Fatalf("retry did not create one new frozen attempt: %+v", task.Attempts)
	}
	// Absence still narrows admission to one at fallback, without materializing it.
	probe, err := workflow.DecodeExecution(must(workflow.EncodeExecution(retried)))
	if err != nil {
		t.Fatal(err)
	}
	if err := probe.PrepareAttempt(workflow.PrepareInput{Ref: b, AttemptID: "probe", Generation: probe.RunnerGeneration, Invocation: old["attempt-b"].Invocation, Authority: old["attempt-b"].Authority, Now: time.Now(), Retry: true, Inflight: []workflow.InflightAttempt{{Ref: a, AttemptID: fresh.AttemptID}}}); !errors.Is(err, workflow.ErrStageCapacity) {
		t.Fatal("historical absence admitted beyond fallback 1", err)
	}
	final, err := f.engine.Run(ctx, x.ProjectID, x.ExecutionID, RunOptions{Retry: &b})
	if err != nil || final.Status != workflow.RunCompleted || !reflect.DeepEqual(spawned, []string{"a", "b", "join"}) || peak.Load() != 1 {
		t.Fatalf("resolved run: %v %s spawned=%v peak=%d", err, final.Status, spawned, peak.Load())
	}
	check("retry b", final)
	if after, err := os.ReadFile(corpus); err != nil || !reflect.DeepEqual(after, wire) {
		t.Fatal("frozen corpus changed", err)
	}
	if loaded, err := store.Load(ctx, x.ProjectID, x.ExecutionID); err != nil || !reflect.DeepEqual(loaded, final) {
		t.Fatal("reload differs", err)
	}
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

// Reachability boundary for retry saturation: only Run accepts a retry, and
// its Claim turns every older active attempt unknown before admission, so a
// saturating peer refuses the retry as unknown outstanding instead of parking it.
// After a failure no new root is eligible, so only a historical multi-active
// record (derived here, corpus unchanged) can hold the slot beside a failure.
func TestI404RetryFacingSaturatingPeersIsRecoveryNotCapacity(t *testing.T) {
	wire, err := os.ReadFile("../workflow/testdata/i404/multi-active.json")
	if err != nil {
		t.Fatal(err)
	}
	x, err := workflow.DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	step := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	step(x.CompleteAttempt("attempt-a", 1, workflow.AttemptResult{ExitCode: 1, Now: x.UpdatedAt}))
	ref := workflow.TaskRef{StageID: "work", AgentID: "a"}
	last, _, _ := x.FindAttempt("attempt-a")
	// The domain parks this retry as saturated while b holds the fallback slot.
	probe, err := workflow.DecodeExecution(must(workflow.EncodeExecution(x)))
	step(err)
	if !errors.Is(probe.PrepareAttempt(workflow.PrepareInput{Ref: ref, AttemptID: "probe", Generation: 1, Invocation: last.Invocation, Authority: last.Authority, Now: x.UpdatedAt, Retry: true}), workflow.ErrStageCapacity) {
		t.Fatal("domain saturation not observed")
	}
	f, _ := startFixture(t, "")
	dir := filepath.Join(f.root, "executions", "v2", x.ProjectID)
	step(os.MkdirAll(dir, 0700))
	step(os.WriteFile(filepath.Join(dir, x.ExecutionID+".json"), must(workflow.EncodeExecution(x)), 0600))
	got, err := f.engine.Run(context.Background(), x.ProjectID, x.ExecutionID, RunOptions{Retry: &ref})
	if !errors.Is(err, workflow.ErrUnknownOutstanding) || errors.Is(err, workflow.ErrStageCapacity) || f.runner.spawns.Load() != 0 {
		t.Fatalf("retry beside saturating peers: %v spawns=%d", err, f.runner.spawns.Load())
	}
	if got.Status != workflow.RunRecoveryRequired || got.RunnerGeneration <= 1 || got.Workflow.Stages[0].Concurrency != nil {
		t.Fatalf("status %s generation %d", got.Status, got.RunnerGeneration)
	}
	if b, _, _ := got.FindAttempt("attempt-b"); b.Status != workflow.AttemptUnknown || b.Resolution != nil {
		t.Fatalf("saturating peer not claimed unknown: %s", b.Status)
	}
	if a, _, _ := got.FindAttempt("attempt-a"); a.Status != workflow.AttemptFailed || len(got.Stages[0].Tasks[0].Attempts) != 1 {
		t.Fatal("retry consumed or replayed attempt-a")
	}
}
