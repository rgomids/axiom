//go:build linux || darwin

package stageexecution

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
)

func oneAgent(maximum int) []workflow.StageSpec {
	return []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder", Effects: []string{"workspace_write"}, MaximumAttempts: maximum}}, Outputs: []string{"result"}, TimeoutSeconds: 5}}
}

func lastAttempt(x workflow.Execution) workflow.Attempt {
	a := x.Stages[0].Tasks[0].Attempts
	return a[len(a)-1]
}

// A durable charged reservation precedes the single Runner call.
func TestI369ReservationPrecedesRunner(t *testing.T) {
	var f *engineFixture
	var seen string
	f = newEngineFixturePolicy(t, oneAgent(1), func(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		x, err := f.engine.Store.Load(ctx, "proj", engineExecutionID)
		if err == nil {
			seen = lastAttempt(x).Accounting
		}
		// Crash after reservation, before the start callback: outcome unknown.
		return executiongraph.ProcessResult{ExitCode: -1, OutcomeUnknown: true}
	}, workflow.SpawnBoundaryPolicy)
	x, err := f.run(t)
	if err != nil {
		t.Fatal(err)
	}
	a := lastAttempt(x)
	if seen != workflow.AccountingCharged || f.runner.spawns.Load() != 1 || a.Status != workflow.AttemptUnknown || a.Accounting != workflow.AccountingCharged || a.NoSpawn {
		t.Fatalf("reservation %q spawns %d attempt %+v", seen, f.runner.spawns.Load(), a)
	}
}

// A lost fence before reservation permits no Runner call.
func TestI369LostFenceBeforeReservationPermitsNoCall(t *testing.T) {
	f := newEngineFixturePolicy(t, oneAgent(1), succeed("OUTPUT result\n"), workflow.SpawnBoundaryPolicy)
	f.engine.Admit = func(ctx context.Context, _ workflow.Execution) error {
		_, err := f.engine.Store.UpdateFenced(ctx, "proj", engineExecutionID, nil, func(x *workflow.Execution) error {
			x.RunnerGeneration++
			return nil
		})
		return err
	}
	if _, err := f.run(t); err == nil || f.runner.spawns.Load() != 0 {
		t.Fatalf("stale fence reached the Runner: %v %d", err, f.runner.spawns.Load())
	}
}

// Only the engine-owned launcher can prove a deterministic no-spawn. Any other
// launcher failure stays charged, even when it carries a typed reason.
func TestI369ForeignLauncherFailureStaysCharged(t *testing.T) {
	f := newEngineFixturePolicy(t, oneAgent(2), succeed(""), workflow.SpawnBoundaryPolicy)
	_, typed := refuseLaunch(context.Background(), runtimeadapter.ErrEffortOverrideConflict, "selector_inspection", "")
	f.engine.Launcher = stubLauncher{buildErr: typed}
	x, err := f.run(t)
	if err != nil {
		t.Fatal(err)
	}
	if a := lastAttempt(x); a.Accounting != workflow.AccountingCharged || a.NoSpawn || a.LaunchFailure == nil || f.runner.spawns.Load() != 0 {
		t.Fatalf("foreign failure credited: %+v", a)
	}
}

// Two deterministic refusals per task lifetime, each needing explicit retry
// with preserve disposition; the third preparation is denied without a record.
func TestI369LifetimeRefusalsThroughEngine(t *testing.T) {
	f := newEngineFixturePolicy(t, oneAgent(1), succeed("OUTPUT result\n"), workflow.SpawnBoundaryPolicy)
	l, r := runtimeLauncherFixture(t, "codex", "runtime-default")
	path := filepath.Join(l.Workspace, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("model_reasoning_effort = \"private-canary\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	f.engine.Launcher = l
	resolve := f.engine.Resolve
	f.engine.Resolve = func(ctx context.Context, x workflow.Execution, ref workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		inv, authority, err := resolve(ctx, x, ref)
		frozen := r.Attempt.Invocation
		frozen.Provenance = inv.Provenance
		return frozen, authority, err
	}
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}
	x, err := f.run(t)
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 2; n++ {
		a := lastAttempt(x)
		if a.Accounting != workflow.AccountingRefusal || !a.NoSpawn || a.LaunchFailure == nil || a.LaunchFailure.Reason != "selector_conflict" || a.Process != nil {
			t.Fatalf("refusal %d: %+v", n, a)
		}
		if n == 2 && a.ParentAttemptID != x.Stages[0].Tasks[0].Attempts[0].AttemptID {
			t.Fatal("retry lineage lost")
		}
		if n == 1 {
			// No automatic retry; an explicit retry needs preserve disposition.
			if _, err = f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref}); err == nil {
				t.Fatal("retry without disposition accepted")
			}
			if x, err = f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, WorkspaceDisposition: "preserve"}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err = f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, WorkspaceDisposition: "preserve"}); err == nil {
		t.Fatal("third refusal slot granted")
	}
	after, err := f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
	if err != nil || len(after.Stages[0].Tasks[0].Attempts) != 2 || f.runner.spawns.Load() != 0 {
		t.Fatalf("exhausted allowance appended or spawned: %v", err)
	}
	if charged, refusals := workflow.AttemptCounts(after.Stages[0].Tasks[0].Attempts); charged != 0 || refusals != 2 {
		t.Fatalf("counts charged=%d refusals=%d", charged, refusals)
	}
}
