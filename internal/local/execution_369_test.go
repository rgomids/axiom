package local

import (
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func TestI369FailureHistoryAndClone(t *testing.T) {
	s, _ := newTestExecutionStore(t)
	x := storeTestExecution(t)
	if err := s.Create(context.Background(), x); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	defer lease.Close()
	x, err := s.UpdateFenced(context.Background(), x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error {
		if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Generation(), Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local", Effects: []string{}}, Now: storeT0}); err != nil {
			return err
		}
		return e.MarkLaunchFailed("a1", lease.Generation(), workflow.LaunchFailure{Reason: "configuration_invalid", Phase: "selector_inspection", Source: "user"}, storeT0)
	})
	if err != nil {
		t.Fatal(err)
	}
	clone := cloneExecutionStages(x.Stages)
	clone[0].Tasks[0].Attempts[0].LaunchFailure.Reason = "builder_failed"
	if x.Stages[0].Tasks[0].Attempts[0].LaunchFailure.Reason != "configuration_invalid" {
		t.Fatal("snapshot alias")
	}
	_, err = s.UpdateFenced(context.Background(), x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error {
		e.Stages[0].Tasks[0].Attempts[0].LaunchFailure.Reason = "builder_failed"
		return nil
	})
	if err == nil {
		t.Fatal("history rewrite accepted")
	}
}

func TestI369AccountingHistory(t *testing.T) {
	s, _ := newTestExecutionStore(t)
	x := storeTestExecution(t)
	x.AttemptPolicy = workflow.SpawnBoundaryPolicy
	if err := s.Create(context.Background(), x); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	defer lease.Close()
	x, err := s.UpdateFenced(context.Background(), x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error {
		return e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Generation(), Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local", Effects: []string{}}, Now: storeT0})
	})
	if err != nil {
		t.Fatal(err)
	}
	x, err = s.UpdateFenced(context.Background(), x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error { return e.ReserveLaunch("a1", lease.Generation(), storeT0) })
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateFenced(context.Background(), x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error {
		e.Stages[0].Tasks[0].Attempts[0].Accounting = workflow.AccountingPending
		return nil
	})
	if err == nil {
		t.Fatal("charge refunded")
	}
	_, err = s.UpdateFenced(context.Background(), x.ProjectID, x.ExecutionID, lease, func(e *workflow.Execution) error {
		e.AttemptPolicy = ""
		e.Stages[0].Tasks[0].Attempts[0].Accounting = ""
		return nil
	})
	if err == nil {
		t.Fatal("policy stripped")
	}
}
