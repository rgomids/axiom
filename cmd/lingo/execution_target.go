package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
)

// workflowStartInput accepts only existing exact selector forms. It also runs
// for direct application calls, which do not pass through CLI parsing.
func workflowStartInput(input cli.WorkflowInput) (cli.WorkflowInput, bool) {
	if input.Execution != "" || input.Number < 0 {
		return input, false
	}
	if input.WorkItem != "" {
		provider, resource, id, ok := cli.ParseWorkItemSelector(input.WorkItem)
		if !ok || input.Number != 0 || input.Provider != "" && input.Provider != provider || input.ProviderRepository != "" && input.ProviderRepository != resource || input.ExternalID != "" && input.ExternalID != id {
			return input, false
		}
		input.Provider, input.ProviderRepository, input.ExternalID = provider, resource, id
	} else if input.ExternalID != "" {
		number, err := strconv.Atoi(input.ExternalID)
		if err != nil || number <= 0 || strconv.Itoa(number) != input.ExternalID || input.Number != 0 {
			return input, false
		}
	} else if input.Number <= 0 {
		return input, false
	}
	return input, true
}

// This domain-separated envelope binds only workflow start. The independent
// Runtime/Profile preview API and its policy digest remain unchanged.
func workflowStartDigest(target workflow.ResolvedTarget, source, policyDigest string) string {
	wire, _ := json.Marshal(struct {
		Protocol                      string
		Target                        workflow.ResolvedTarget
		ProjectSource, RuntimePreview string
	}{"axiom-workflow-start-v1", target, source, policyDigest})
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}

func executionTargetView(target workflow.ResolvedTarget, source string) *cli.ExecutionTargetView {
	return &cli.ExecutionTargetView{ProjectID: target.ProjectID, ProjectSource: source, RepositoryKey: target.Repository.Key, Provider: target.WorkItem.Provider, Resource: target.WorkItem.Resource, ExternalID: target.WorkItem.ExternalID, WorkItem: target.WorkItem.Provider + ":" + target.WorkItem.Resource + "#" + target.WorkItem.ExternalID}
}

func (s lifecycleService) WorkflowStart(ctx context.Context, input cli.WorkflowInput) cli.Result {
	if ctx.Err() != nil {
		return workflowResult(workflow.Result{Status: workflow.Interrupted, Category: "workflow_cancelled"}, s.provenance)
	}
	original := cli.WorkflowProjectSelector(ctx, input.Project)
	selection, category := s.projectContext().Effective(ctx, original)
	if category != "" {
		return s.contextResult(selection, category)
	}
	input, ok := workflowStartInput(input)
	if !ok {
		return workflowResult(workflow.Result{Status: workflow.ValidationFailed, Category: "invalid_execution_input"}, s.provenance)
	}
	input.Project = selection.Effective
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionStart); blocked != nil {
		return *blocked
	}
	target, failed := s.workflows.ValidateTarget(ctx, workflowTarget(input))
	if failed.Category != "" {
		return workflowResult(failed, s.provenance)
	}
	policyInput := cli.RuntimeProfilePreviewInput{Project: input.Project, Role: input.Role, Complexity: input.Complexity, Capabilities: input.Capabilities, Runtime: input.Runtime}
	preview, policy, err := s.runtimePolicyPreview(ctx, policyInput)
	if err != nil {
		return s.runtimeResolutionResult(preview, false)
	}
	reviewed := workflowStartDigest(target, selection.Source, preview.Digest())
	if input.RuntimePreview == "" {
		response := s.runtimeResolutionResult(preview, true)
		response.ExecutionTarget = executionTargetView(target, selection.Source)
		response.PreviewDigest = reviewed
		return response
	}
	if input.RuntimePreview != reviewed {
		return s.runtimePolicyFailure("stale_preview")
	}
	binding, err := policy.Check(ctx, preview)
	if err != nil {
		return s.runtimePolicyFailure("stale_preview")
	}
	if input.Runtime != "" && input.Runtime != binding.Choice.RuntimeID {
		return s.runtimePolicyFailure("runtime_mismatch")
	}
	// Policy observation is read-only but may take time. Check the winning Project
	// again and let Start revalidate linkage/repository against the reviewed snapshot.
	fresh, category := s.projectContext().Effective(ctx, original)
	if category != "" || fresh.Effective != selection.Effective || fresh.Source != selection.Source {
		return s.runtimePolicyFailure("stale_preview")
	}
	// Recheck operational admission after policy observation, before effects.
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionStart); blocked != nil {
		return *blocked
	}
	input.Runtime = binding.Choice.RuntimeID
	startTarget := workflowTarget(input)
	startTarget.ReviewedTarget = &target
	startTarget.RuntimePreview = preview.Digest()
	result := s.workflows.Start(ctx, startTarget)
	if result.Category == "stale_execution_target" {
		return s.runtimePolicyFailure("stale_preview")
	}
	return workflowResult(result, s.provenance)
}
