package workflow

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func TestExplicitNewSelectionRetryKeepsPriorAttemptImmutable(t *testing.T) {
	e := twoStageExecution(t, false)
	if err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: "a0", Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0, NewSelection: true}); !errors.Is(err, ErrTransition) {
		t.Fatalf("new selection without retry accepted: %v", err)
	}
	run(t, &e, TaskRef{"work", "builder"}, "a1")
	if err := e.CompleteAttempt("a1", 1, AttemptResult{ExitCode: 1, Now: t0}); err != nil {
		t.Fatal(err)
	}
	first := e.Stages[0].Tasks[0].Attempts[0].Invocation
	changed := testInvocation("do")
	changed.Model, changed.Effort = "other-model", "high"
	changed.Provenance = []FieldProvenance{{"runtime", "local_default", ""}, {"model", "run_override", ""}, {"effort", "run_override", ""}, {"instructions", "workflow_agent", ""}}
	if err := e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: "a2", Generation: 1, Invocation: changed, Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0, Retry: true, NewSelection: true}); err != nil {
		t.Fatalf("explicit new selection refused: %v", err)
	}
	attempts := e.Stages[0].Tasks[0].Attempts
	if !reflect.DeepEqual(attempts[0].Invocation, first) || attempts[1].Invocation.Model != "other-model" || attempts[1].Number != 2 {
		t.Fatalf("new selection rebound the prior attempt: %#v", attempts)
	}
}

func TestAgentRuntimeRestrictionRefusesOtherRuntime(t *testing.T) {
	restricted := func(runtimes ...string) (Execution, error) {
		return NewExecution(NewExecutionInput{
			ExecutionID: testExecutionID, ProjectID: "proj", WorkItem: WorkItemLink{"linear", "AXM-42", "app"},
			Repositories: []RepositoryBinding{{"app", "github.com/example/app"}}, WorkflowID: "minimal", Revision: "r1", Now: t0,
			Stages: []StageSpec{{ID: "work", Agents: []AgentSpec{{ID: "builder", Runtimes: runtimes}}, TimeoutSeconds: 60}},
		})
	}
	e, err := restricted("claude")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ClaimRunner(1, t0); err != nil {
		t.Fatal(err)
	}
	before, _ := EncodeExecution(e)
	err = e.PrepareAttempt(PrepareInput{Ref: TaskRef{"work", "builder"}, AttemptID: "a1", Generation: 1, Invocation: testInvocation("do"), Authority: AttemptAuthority{"app", []string{}, "user:local"}, Now: t0})
	if !errors.Is(err, ErrUnauthorizedEffect) {
		t.Fatalf("restricted Runtime accepted: %v", err)
	}
	if after, _ := EncodeExecution(e); !bytes.Equal(before, after) {
		t.Fatal("refusal mutated the aggregate")
	}
	if _, err := restricted("claude", "claude"); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("duplicate restriction accepted: %v", err)
	}
}
