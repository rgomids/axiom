//go:build linux || darwin

package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestCanonicalLifecycleDiscoveryAndEvidence(t *testing.T) {
	f := newExecutionRunFixture(t)
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	service := f.env.service.(lifecycleService)
	before := snapshotTrees(t, f.env.state)
	listed := service.WorkflowList(context.Background(), cli.ExecutionListInput{Project: "external"})
	if !listed.Listed || len(listed.Executions) != 1 || listed.Executions[0].ExecutionID != done.Stage.ExecutionID || listed.Executions[0].Status != string(workflow.RunCompleted) {
		t.Fatalf("list %+v", listed)
	}
	for _, evidence := range []bool{false, true} {
		input := cli.WorkflowInput{Project: "external", Execution: done.Stage.ExecutionID}
		var got cli.Result
		if evidence {
			got = service.WorkflowEvidence(context.Background(), input)
		} else {
			got = service.WorkflowStatus(context.Background(), input)
		}
		if got.Workflow == nil || got.Workflow.ExecutionKind != "canonical" || got.Workflow.Revision != done.Stage.Revision || got.Maintenance == nil {
			t.Fatalf("inspection %+v", got)
		}
	}
	bad := service.WorkflowStatus(context.Background(), cli.WorkflowInput{Project: "external", Execution: done.Stage.ExecutionID, Repository: "another"})
	if bad.Category != "execution_selector_conflict" {
		t.Fatalf("wrong repository %+v", bad)
	}
	if after := snapshotTrees(t, f.env.state); !bytes.Equal(before, after) {
		t.Fatal("inspection changed bytes")
	}
	for _, result := range []cli.Result{service.WorkflowStart(context.Background(), cli.WorkflowInput{}), service.WorkflowAdvance(context.Background(), cli.WorkflowInput{}), service.WorkflowFact(context.Background(), cli.WorkflowInput{}), service.WorkflowResume(context.Background(), cli.WorkflowInput{}), service.WorkflowReconcile(context.Background(), cli.WorkflowInput{})} {
		if result.Category != "workflow_legacy_operation_retired" {
			t.Fatalf("legacy writer active %+v", result)
		}
	}
}
