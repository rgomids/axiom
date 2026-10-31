//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func TestEngineExplicitNewSelectionFreezesFreshTupleAsNewAttempt(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	x, err := f.run(t)
	if err != nil || x.Status != workflow.RunFailed {
		t.Fatalf("initial attempt: %v %s", err, x.Status)
	}
	original := x.Stages[0].Tasks[0].Attempts[0].Invocation
	resolve := f.engine.Resolve
	f.engine.Resolve = func(ctx context.Context, x workflow.Execution, ref workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		inv, authority, err := resolve(ctx, x, ref)
		inv.Runtime, inv.Model = "other-runtime", "explicit-model"
		inv.Provenance[0].Source, inv.Provenance[1].Source = "run_override", "run_override"
		return inv, authority, err
	}
	f.runner.run = succeed("OUTPUT result\n")
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}
	x, err = f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, NewSelection: true, WorkspaceDisposition: "preserve"})
	if err != nil || x.Status != workflow.RunCompleted {
		t.Fatalf("new selection retry: %v %s", err, x.Status)
	}
	attempts := x.Stages[0].Tasks[0].Attempts
	if len(attempts) != 2 || !reflect.DeepEqual(attempts[0].Invocation, original) || attempts[1].Invocation.Runtime != "other-runtime" || attempts[1].Invocation.Provenance[1].Source != "run_override" {
		t.Fatalf("new selection was not a new attempt with new provenance: %#v", attempts)
	}
}

func TestEngineNewSelectionNeverRetargetsRepository(t *testing.T) {
	f := newEngineFixture(t, twoStages(false), succeed(""))
	if _, err := f.run(t); err != nil {
		t.Fatal(err)
	}
	resolve := f.engine.Resolve
	f.engine.Resolve = func(ctx context.Context, x workflow.Execution, ref workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
		inv, authority, err := resolve(ctx, x, ref)
		authority.RepositoryKey = "other"
		return inv, authority, err
	}
	ref := workflow.TaskRef{StageID: "work", AgentID: "builder"}
	if _, err := f.engine.Run(context.Background(), "proj", engineExecutionID, RunOptions{Retry: &ref, NewSelection: true, WorkspaceDisposition: "preserve"}); !errors.Is(err, workflow.ErrTransition) || f.runner.spawns.Load() != 1 {
		t.Fatalf("repository retargeted: %v spawns=%d", err, f.runner.spawns.Load())
	}
}

func TestEngineGateApprovalAndUnknownAbandonNeedHumanActor(t *testing.T) {
	f := newEngineFixture(t, twoStages(true), succeed("OUTPUT result\n"))
	x, err := f.run(t)
	if err != nil || x.Status != workflow.RunAwaitingGate || f.runner.spawns.Load() != 1 {
		t.Fatalf("gate did not stop progression: %v %s", err, x.Status)
	}
	if _, err := f.engine.ApproveGate(context.Background(), "proj", engineExecutionID, "work", "agent:codex"); !errors.Is(err, workflow.ErrTransition) {
		t.Fatalf("non-human gate approval accepted: %v", err)
	}
	if _, err := f.engine.ApproveGate(context.Background(), "proj", engineExecutionID, "work", "human:operator"); err != nil {
		t.Fatal(err)
	}
	if x, err = f.run(t); err != nil || x.Status != workflow.RunCompleted || f.runner.spawns.Load() != 2 {
		t.Fatalf("approved gate did not progress: %v %s", err, x.Status)
	}
	if _, err := f.engine.Abandon(context.Background(), "proj", engineExecutionID, x.Stages[0].Tasks[0].Attempts[0].AttemptID, "human:operator"); !errors.Is(err, workflow.ErrTransition) {
		t.Fatalf("abandoned a non-unknown attempt: %v", err)
	}
}
