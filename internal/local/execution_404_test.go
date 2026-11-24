package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func i404StoreExecution(t *testing.T, limit *int) workflow.Execution {
	t.Helper()
	wire, err := os.ReadFile("../workflow/testdata/i404/dag.json")
	if err != nil {
		t.Fatal(err)
	}
	x, err := workflow.DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	x.Workflow.Stages[0].Concurrency = limit
	fresh, err := workflow.NewExecution(workflow.NewExecutionInput{
		ExecutionID: x.ExecutionID, ProjectID: x.ProjectID, WorkItem: x.WorkItem,
		Repositories: x.Repositories, WorkflowID: x.Workflow.ID, Revision: x.Workflow.Revision,
		Stages: x.Workflow.Stages, Now: x.CreatedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func TestI404StoreFrozenWorkflowProgressRestartAndCAS(t *testing.T) {
	for _, name := range []string{"absent", "explicit1", "explicit2"} {
		t.Run(name, func(t *testing.T) {
			var limit *int
			if name != "absent" {
				value := 1
				if name == "explicit2" {
					value = 2
				}
				limit = &value
			}
			x := i404StoreExecution(t, limit)
			frozen := x.Workflow
			s, root := newTestExecutionStore(t)
			ctx := context.Background()
			if err := s.Create(ctx, x); err != nil {
				t.Fatal(err)
			}
			lease, err := s.AcquireRunner(ctx, x.ProjectID, x.ExecutionID)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.Close()
			mutate := func(fn func(*workflow.Execution) error) workflow.Execution {
				t.Helper()
				next, err := s.UpdateFenced(ctx, x.ProjectID, x.ExecutionID, lease, fn)
				if err != nil {
					t.Fatal(err)
				}
				return next
			}
			x = mutate(func(e *workflow.Execution) error { return e.ClaimRunner(lease.Generation(), e.CreatedAt) })
			stale := x
			x = mutate(func(e *workflow.Execution) error {
				return e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "a"}, AttemptID: "a1", Generation: lease.Generation(), Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: e.CreatedAt})
			})
			x = mutate(func(e *workflow.Execution) error {
				return e.MarkStarted("a1", lease.Generation(), workflow.ProcessIdentity{PID: 404, StartIdentity: "synthetic", GroupIdentity: "404"}, e.CreatedAt)
			})
			x = mutate(func(e *workflow.Execution) error {
				return e.CompleteAttempt("a1", lease.Generation(), workflow.AttemptResult{Now: e.CreatedAt})
			})
			if !reflect.DeepEqual(frozen, x.Workflow) {
				t.Fatal("fenced progress changed frozen Workflow")
			}
			if err := stale.RequestCancel("user:local", stale.CreatedAt); err != nil {
				t.Fatal(err)
			}
			if err := s.Save(ctx, stale, stale.Revision-1, lease); !errors.Is(err, ErrConflict) {
				t.Fatal("stale exact-byte CAS accepted", err)
			}
			lease.Close()
			restarted, err := NewExecutionStore(root)
			if err != nil {
				t.Fatal(err)
			}
			got, err := restarted.Load(ctx, x.ProjectID, x.ExecutionID)
			if err != nil || !reflect.DeepEqual(frozen, got.Workflow) || got.Status != workflow.RunRunning || got.Stages[0].Tasks[0].Attempts[0].Status != workflow.AttemptSucceeded {
				t.Fatalf("restart lost snapshot/progress: %v %+v", err, got)
			}
			if _, err := restarted.UpdateFenced(ctx, x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error { return e.RequestCancel("user:local", e.CreatedAt) }); !errors.Is(err, workflow.ErrStaleRunner) {
				t.Fatal("released fence accepted", err)
			}
		})
	}
}

func TestI404StoreRejectsRehashedCapacityRewrite(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed", true: "removed"}[remove], func(t *testing.T) {
			two := 2
			x := i404StoreExecution(t, &two)
			s, _ := newTestExecutionStore(t)
			ctx := context.Background()
			if err := s.Create(ctx, x); err != nil {
				t.Fatal(err)
			}
			one := 1
			var replacement *int
			if !remove {
				replacement = &one
			}
			forged := i404StoreExecution(t, replacement)
			if err := forged.Validate(); err != nil {
				t.Fatal("rehashed candidate is malformed", err)
			}
			before, _ := workflow.EncodeExecution(x)
			_, err := s.UpdateFenced(ctx, x.ProjectID, x.ExecutionID, nil, func(e *workflow.Execution) error {
				e.Workflow = forged.Workflow
				return e.RequestCancel("user:local", e.CreatedAt)
			})
			if !errors.Is(err, ErrConflict) {
				t.Fatal("frozen capacity rewrite accepted", err)
			}
			got, err := s.Load(ctx, x.ProjectID, x.ExecutionID)
			after, encodeErr := workflow.EncodeExecution(got)
			if err != nil || encodeErr != nil || !bytes.Equal(before, after) {
				t.Fatal("refused rewrite changed canonical bytes", err, encodeErr)
			}
		})
	}
}

func TestI404HistoricalMultiActiveLoadsAtCanonicalLocation(t *testing.T) {
	s, root := newTestExecutionStore(t)
	wire, err := os.ReadFile("../workflow/testdata/i404/multi-active.json")
	if err != nil {
		t.Fatal(err)
	}
	x, err := workflow.DecodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "executions", "v2", x.ProjectID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, x.ExecutionID+".json")
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(context.Background(), x.ProjectID, x.ExecutionID)
	if err != nil || got.Workflow.Stages[0].Concurrency != nil || len(got.Stages[0].Tasks[0].Attempts) != 1 || len(got.Stages[0].Tasks[1].Attempts) != 1 {
		t.Fatal("historical multi-active rejected/rewritten", err)
	}
	if kind := kindOf(t, root, "executions/v2"); kind != InventoryNewer || !ExecutionReferenceNamespace(root, InventoryEntry{Relative: "executions/v2", Kind: kind}) {
		t.Fatal("canonical namespace lost deliberate inventory recognition", kind)
	}
	listed, err := s.List(context.Background(), x.ProjectID)
	if err != nil || len(listed) != 1 || !reflect.DeepEqual(listed[0].Workflow, x.Workflow) {
		t.Fatal("canonical list lost historical record", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(wire, after) {
		t.Fatal("read rewrote historical bytes", err)
	}
}
