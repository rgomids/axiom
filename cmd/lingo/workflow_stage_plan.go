package main

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

var _ cli.WorkflowStagePlanService = lifecycleService{}

// policyCapabilities confirms that at least one supported Runtime, narrowed
// explicitly, can serve a graph capability request under the same Project
// policy. The compiler has already resolved each agent exactly under its own
// Runtime constraints; this check selects nothing and never widens them.
type policyCapabilities struct {
	policy    runtimeapplication.Service
	projectID string
}

func (p policyCapabilities) ValidateCapability(ctx context.Context, request executiongraph.CapabilityRequest) error {
	err := runtimeapplication.ErrBlocked
	for _, runtimeID := range []string{"claude", "codex"} {
		if _, err = p.policy.Preview(ctx, p.projectID, runtimeapplication.Request{Role: request.Role, Complexity: request.Complexity, Capabilities: request.Capabilities, RuntimeID: runtimeID}); err == nil {
			return nil
		}
	}
	return err
}

// WorkflowStagePlan composes the retained Execution binding (#274), the
// approved Plan document, current Project Runtime policy and the existing
// stage compiler (#297). It is read-only: no Execution, attempt, artifact,
// Runtime process or Provider effect is created.
func (s lifecycleService) WorkflowStagePlan(ctx context.Context, input cli.WorkflowStagePlanInput) cli.WorkflowStagePlanResponse {
	if ctx.Err() != nil {
		return s.stagePlanFailure(completion.Interrupted, "workflow_cancelled", nil, nil)
	}
	if input.Project == "" {
		id, failure := s.EffectiveProject(ctx, "")
		if id == "" {
			return cli.WorkflowStagePlanResponse{Result: failure}
		}
		input.Project = id
	}
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionStatus); blocked != nil {
		return cli.WorkflowStagePlanResponse{Result: *blocked}
	}
	provider, resource, externalID, ok := cli.ParseWorkItemSelector(input.WorkItem)
	if !ok || input.Execution == "" || input.ExpectedRevision == 0 || input.Stage == "" || input.PlanFile == "" {
		return s.stagePlanFailure(completion.ValidationFailure, "invalid_execution_input", nil, nil)
	}
	target := workflowTarget(cli.WorkflowInput{Project: input.Project, Repository: input.Repository, Provider: provider, ProviderRepository: resource, ExternalID: externalID, Execution: input.Execution})
	planning, loaded := s.workflows.StagePlanningContext(ctx, target, input.ExpectedRevision, input.Stage)
	if loaded.Status != workflow.Succeeded {
		var conditions []string
		for _, id := range planning.MissingInputs {
			conditions = append(conditions, "missing_input:"+id)
		}
		response := s.stagePlanFailure(loaded.Status, loaded.Category, conditions, nil)
		if loaded.State.ExecutionID != "" {
			response.Execution = &workflowcompiler.ExecutionRef{ExecutionID: loaded.State.ExecutionID, Revision: loaded.State.Revision}
		}
		return response
	}
	execution := workflowcompiler.ExecutionRef{ExecutionID: planning.ExecutionID, Revision: planning.ExecutionRevision}
	document, documentDigest, err := readPlanDocument(input.PlanFile)
	if err != nil {
		return s.stagePlanFailure(completion.ValidationFailure, "invalid_stage_plan", nil, &execution)
	}
	source := s.RuntimePolicySource
	if source == nil {
		source = runtimePolicySource{installation: s.installation, portable: s.portable, stateRoot: s.stateRoot, runtimes: s.runtimeObservationSources()}
	}
	policy := runtimeapplication.New(source)
	bindings := make([]workflowcompiler.InputBinding, 0, len(planning.Inputs))
	for _, binding := range planning.Inputs {
		bindings = append(bindings, workflowcompiler.InputBinding{InputID: binding.InputID, Reference: binding.Reference, Digest: binding.Digest})
	}
	gates := make([]workflowcompiler.GateRef, 0, len(planning.Gates))
	for _, gate := range planning.Gates {
		gates = append(gates, workflowcompiler.GateRef{Kind: gate.Kind, Timing: gate.Timing, Satisfied: gate.Satisfied})
	}
	request := workflowcompiler.StagePlanRequest{Execution: execution, Gates: gates, PlanDocumentDigest: documentDigest, Compile: workflowcompiler.Request{ProjectID: planning.ProjectID, RepositoryKey: planning.RepositoryKey, Workflow: planning.Binding.Definition, Definition: planning.Document, StageID: planning.Stage.ID, Plan: document.Plan, InputBindings: bindings, ParentAuthority: document.AuthorityCeiling}}
	plan, err := workflowcompiler.New(policy, policyCapabilities{policy: policy, projectID: planning.ProjectID}).PlanStage(ctx, request)
	if ctx.Err() != nil {
		return s.stagePlanFailure(completion.Interrupted, "workflow_cancelled", nil, &execution)
	}
	if err != nil {
		failure := workflowcompiler.ClassifyFailure(err)
		status := completion.ValidationFailure
		if failure.AuthorityDenied {
			status = completion.DeniedAuthority
		}
		return s.stagePlanFailure(status, failure.Category, nil, &execution)
	}
	// Runtime observation is read-only but may take time. A concurrent
	// Execution change makes this proposal stale instead of silently current;
	// policy/observation changes are bound by the digest of what was observed.
	recheck, reloaded := s.workflows.StagePlanningContext(ctx, target, input.ExpectedRevision, input.Stage)
	if reloaded.Status != workflow.Succeeded || recheck.ExecutionRevision != planning.ExecutionRevision || recheck.Binding.Definition != planning.Binding.Definition || recheck.Binding.ObservationDigest != planning.Binding.ObservationDigest {
		return s.stagePlanFailure(completion.DeniedAuthority, "stale_execution_revision", nil, &execution)
	}
	next := "Review the proposal; it grants no authority and is not dispatchable in this release (#276). Any Execution, Plan, Project policy or Runtime change requires a fresh plan"
	if len(plan.Blockers) != 0 {
		next = "Resolve the listed blockers, recording required human gates only through workflow fact; then request a fresh plan. The proposal grants no authority"
	}
	response := cli.WorkflowStagePlanResponse{Result: canonicalCompletion(completion.Facts{Completed: true}, "Stage plan proposal ready", []string{"stage-plan:" + plan.Digest}, next, s.provenance), Planned: true, Execution: &execution, Workflow: &plan.Workflow, StageID: plan.StageID, Plan: &plan}
	response.Result.Category = "stage_plan_ready"
	return response
}

func (s lifecycleService) stagePlanFailure(status completion.Status, category string, conditions []string, execution *workflowcompiler.ExecutionRef) cli.WorkflowStagePlanResponse {
	message, next := "Stage plan refused: "+category, "Correct the reported input or state and request a fresh plan; never widen authority or substitute a Runtime or Profile"
	switch category {
	case "stale_execution_revision":
		next = "Inspect workflow status and request a fresh plan with the current Execution revision"
	case "stage_prerequisite_missing":
		next = "Complete the predecessor stage outputs listed in conditions, then request a fresh plan"
	case "configured_binding_required":
		next = "Historical Executions have no pinned workflow revision; start a new configured Execution to plan stages"
	case "recovery_required":
		next = "Inspect the Execution through recovery inspect; no stage plan is available until its binding is recovered"
	case "runtime_unresolvable", "unsupported_effort":
		next = "Correct the Project Runtime policy or local Runtime/Profile observations, then request a fresh plan"
	case "workflow_cancelled":
		message, next = "Stage planning was cancelled", "Retry the read-only plan request"
	}
	response := cli.WorkflowStagePlanResponse{Result: canonicalCompletion(factsForCompletionStatus(status), message, nil, next, s.provenance), Planned: true, Conditions: conditions, Execution: execution}
	response.Result.Category = category
	return response
}

var errPlanFile = errors.New("stage plan file is not a regular file")

// readPlanDocument accepts only a regular, non-symlink file, checked before
// opening so a FIFO or device cannot block or stream unbounded input.
func readPlanDocument(path string) (workflowcompiler.PlanDocument, string, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return workflowcompiler.PlanDocument{}, "", errPlanFile
	}
	f, err := os.Open(path)
	if err != nil {
		return workflowcompiler.PlanDocument{}, "", err
	}
	defer f.Close()
	if opened, err := f.Stat(); err != nil || !os.SameFile(before, opened) {
		return workflowcompiler.PlanDocument{}, "", errPlanFile
	}
	wire, err := io.ReadAll(io.LimitReader(f, workflowcompiler.MaxPlanDocumentBytes+1))
	if err != nil {
		return workflowcompiler.PlanDocument{}, "", err
	}
	return workflowcompiler.DecodePlanDocument(wire)
}
