package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func TestI369FailureProjection(t *testing.T) {
	f := workflow.LaunchFailure{Reason: "selector_conflict", Phase: "selector_inspection", Source: "user"}
	x := workflow.Execution{Stages: []workflow.StageRun{{StageID: "work", Tasks: []workflow.TaskRun{{AgentID: "builder", Attempts: []workflow.Attempt{{AttemptID: "a1", LaunchFailure: &f}}}}}}}
	v := executionPayload(x, nil)
	a := v.Stages[0].Tasks[0].Attempts[0]
	if a.LaunchFailure == nil || *a.LaunchFailure != f || a.NextAction == "" {
		t.Fatal("typed status lost")
	}
	wire, err := json.Marshal(v)
	if err != nil || !strings.Contains(string(wire), `"phase":"selector_inspection"`) {
		t.Fatal("typed JSON lost", err)
	}
}
