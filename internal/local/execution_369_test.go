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
