//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

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
