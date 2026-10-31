package main

import (
	"context"
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
