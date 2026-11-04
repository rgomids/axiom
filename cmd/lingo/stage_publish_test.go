package main

import (
	"context"
	"strings"
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
	if !strings.Contains(result.Result.Completion.Next().String(), "Publication is retired") || !strings.Contains(result.Result.Completion.Next().String(), "private engine ingress") || !strings.Contains(result.Result.Completion.Next().String(), "workflow coordination list") {
		t.Fatal(result.Result.Completion.Next().String())
	}
}
