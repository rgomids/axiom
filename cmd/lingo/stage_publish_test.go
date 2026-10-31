package main

import (
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
)

func TestPublishRefusesAbsentRunningContext(t *testing.T) {
	root := t.TempDir()
	s := lifecycleService{stateRoot: root, provenance: unknownProvenance()}
	for _, key := range []string{"AXIOM_CORRELATION_ID", "AXIOM_EXECUTION_ID", "AXIOM_GRAPH_ID", "AXIOM_ATTEMPT_ID"} {
		t.Setenv(key, "")
	}
	result := s.stagePublish(context.Background(), cli.WorkflowStageInput{Kind: "progress", Fields: []string{"summary=synthetic"}})
	if result.Result.Category != "child_context_required" {
		t.Fatal(result.Result.Category)
	}
}
