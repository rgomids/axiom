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

// DMH-07: the Work Item token written by a canonical run decodes back to the
// exact Provider/resource/ID, so the original selector inspects its Execution
// through list/status/evidence. Different or malformed identity fails closed.
func TestCanonicalLifecycleWorkItemSelectorRoundTrip(t *testing.T) {
	f := newExecutionRunFixture(t)
	done := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	if done.Stage.WorkItem.ID != "owner:repo:7" {
		t.Fatalf("canonical token changed: %+v", done.Stage.WorkItem)
	}
	service := f.env.service.(lifecycleService)
	listed := service.WorkflowList(context.Background(), cli.ExecutionListInput{Project: "external"})
	if len(listed.Executions) != 1 || listed.Executions[0].WorkItem != (cli.ExecutionWorkItemRef{Provider: "github", Resource: "owner/repo", ExternalID: "7"}) {
		t.Fatalf("list misprojects identity: %+v", listed.Executions)
	}
	for _, command := range []string{"status", "evidence"} {
		args := []string{"workflow", command, "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--execution", done.Stage.ExecutionID}
		var out bytes.Buffer
		if code := cli.Run(context.Background(), args, f.env.service, currentProvenance(), &out); code != cli.ExitSuccess || bytes.Contains(out.Bytes(), []byte("execution_selector_conflict")) {
			t.Fatalf("%s with original selector: %d %s", command, code, out.String())
		}
		if !bytes.Contains(out.Bytes(), []byte(`"resource":"owner/repo"`)) || !bytes.Contains(out.Bytes(), []byte(`"externalId":"7"`)) {
			t.Fatalf("%s lost exact identity: %s", command, out.String())
		}
		for _, other := range []string{"github:owner/repo#8", "github:owner/other#7", "github:other/repo#7"} {
			args[7] = other
			out.Reset()
			if code := cli.Run(context.Background(), args, f.env.service, currentProvenance(), &out); code == cli.ExitSuccess || !bytes.Contains(out.Bytes(), []byte("execution_selector_conflict")) {
				t.Fatalf("%s accepted different identity %s: %s", command, other, out.String())
			}
		}
	}
	for _, tc := range []struct {
		link             workflow.WorkItemLink
		resource, wantID string
	}{
		{workflow.WorkItemLink{Provider: "github", ID: "owner:repo:7"}, "owner/repo", "7"},
		{workflow.WorkItemLink{Provider: "github", ID: "owner.x:re-po_1:12"}, "owner.x/re-po_1", "12"},
		// Malformed or legacy forms are never guessed into a GitHub identity.
		{workflow.WorkItemLink{Provider: "github", ID: "owner:repo"}, "", "owner:repo"},
		{workflow.WorkItemLink{Provider: "github", ID: "owner:repo:7:8"}, "", "owner:repo:7:8"},
		{workflow.WorkItemLink{Provider: "github", ID: "owner:repo:07"}, "", "owner:repo:07"},
		{workflow.WorkItemLink{Provider: "github", ID: "owner::7"}, "", "owner::7"},
		{workflow.WorkItemLink{Provider: "github", ID: "..:repo:7"}, "", "..:repo:7"},
		{workflow.WorkItemLink{Provider: "linear", ID: "AXM:42:1"}, "", "AXM:42:1"},
	} {
		if resource, id := executionWorkItemIdentity(tc.link); resource != tc.resource || id != tc.wantID {
			t.Fatalf("%+v decoded to %q %q", tc.link, resource, id)
		}
	}
}
