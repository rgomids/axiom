//go:build linux || darwin

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
)

func TestCanonicalReadinessIgnoresLegacyLedgersAndGrantsNoEffects(t *testing.T) {
	f := newExecutionRunFixture(t)
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	service := f.env.service.(lifecycleService)
	// Deliberately undecodable historical bytes cannot become current truth.
	for _, family := range []string{"graphs/v1", "single-stage-runs/v1", "executions/v1", "coordination/v1"} {
		path := filepath.Join(f.env.state, family, "legacy.json")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("not a current Execution"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := snapshotTrees(t, f.env.state)
	in := cli.WorkflowStageInput{Operation: "delivery-packet", Project: "external", Execution: done.Stage.ExecutionID}
	got := service.WorkflowStage(context.Background(), in)
	payload, ok := got.Payload.(map[string]any)
	if !ok || got.Result.Category != "execution_readiness" || payload["ready"] != true || payload["deliveryAuthorized"] != false {
		t.Fatalf("readiness %+v", got)
	}
	in.AuthorizeLocal = true
	if got := service.WorkflowStage(context.Background(), in); got.Result.Category != "unsupported_execution_operation" {
		t.Fatalf("unexpected mutation %+v", got)
	}
	in = cli.WorkflowStageInput{Operation: "coordination-list", Project: "external", Execution: done.Stage.ExecutionID}
	if got := service.WorkflowStage(context.Background(), in); got.Result.Category != "execution_messages" {
		t.Fatalf("messages %+v", got)
	}
	if after := snapshotTrees(t, f.env.state); !bytes.Equal(before, after) {
		t.Fatal("inspection changed state")
	}
}
