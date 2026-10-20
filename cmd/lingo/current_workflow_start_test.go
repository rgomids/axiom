package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
)

// Positive admission consumes explicitly injected model/auth/effort/source proof;
// the real protected stores and executable digest remain part of the path.
// The fake executable is only observed, never invoked as a model.
func TestCurrentWorkflowStartFreezesReviewedV2ProfileWithoutModelRun(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	service = currentProfileStartFixture(t, service, env.state)
	input := cli.WorkflowInput{Project: "external", Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{"axiom-skills"}}
	preview := service.WorkflowStart(context.Background(), input)
	if preview.Completion == nil || preview.Completion.Status() != "success" || preview.Workflow != nil || preview.PreviewDigest == "" {
		t.Fatalf("current preview %+v", preview)
	}
	input.RuntimePreview = preview.PreviewDigest
	started := service.WorkflowStart(context.Background(), input)
	if started.Completion == nil || started.Completion.Status() != "success" || started.Workflow == nil || started.Workflow.Binding == nil || started.Workflow.Binding.SchemaVersion != 2 || started.Workflow.Binding.FrozenProfile == nil {
		t.Fatalf("current admission %+v", started)
	}
	profile := started.Workflow.Binding.FrozenProfile
	if runtimeprofile.ValidateResolvedProfileV4(*profile) != nil || profile.ProfileID != "worker" || profile.Effort.Mode != "runtime-default" || profile.Effort.Value != "default" || profile.EffortSourceDigest == "" {
		t.Fatalf("lost frozen intent %+v", profile)
	}
	store, err := local.NewWorkflowStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	item := started.Workflow.WorkItem
	saved, err := store.Load(context.Background(), workItemSourceProjectID, "main", workflow.WorkItem{Provider: item.Provider, Resource: item.Resource, ExternalID: item.ExternalID, URL: item.URL, State: item.State})
	if err != nil || saved.FormatVersion != workflow.OperationalConfiguredFormatVersion || saved.Binding == nil || saved.Binding.FrozenProfile == nil || *saved.Binding.FrozenProfile != *profile {
		t.Fatal("protected publication lost snapshot", err)
	}
	// Preserve the closed Plan parser assertion at current admission, where it is reachable.
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	invalid := filepath.Join(t.TempDir(), "invalid-plan.json")
	if err := os.WriteFile(invalid, []byte(`{"formatVersion":1,"plan":{},"authorityCeiling":[],"grant":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	planned := service.WorkflowStagePlan(context.Background(), cli.WorkflowStagePlanInput{Project: "external", Repository: "main", WorkItem: "github:owner/repo#7", Execution: started.Workflow.ExecutionID, ExpectedRevision: 1, Stage: "intake", PlanFile: invalid})
	if planned.Result.Category != "invalid_stage_plan" || planned.Plan != nil {
		t.Fatalf("unknown Plan field accepted %+v", planned)
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.elsewhere)) {
		t.Fatal("refused stage planning mutated state")
	}
	input.Execution = started.Workflow.ExecutionID
	input.RuntimePreview = ""
	input.Role = ""
	input.Complexity = ""
	input.Capabilities = nil
	inspected := service.WorkflowStatus(context.Background(), input)
	if inspected.Workflow == nil || inspected.Workflow.Binding == nil || inspected.Workflow.Binding.FrozenProfile == nil || *inspected.Workflow.Binding.FrozenProfile != *profile {
		t.Fatal("read-back changed frozen tuple")
	}
}
