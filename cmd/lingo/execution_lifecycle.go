package main

import (
	"context"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
)

var _ cli.ExecutionListService = lifecycleService{}

const executionTimeFormat = "2006-01-02T15:04:05.999999999Z07:00"

// WorkflowList is a read-only canonical namespace observation. Legacy ledgers
// are never read, converted or resumed by discovery.
func (s lifecycleService) WorkflowList(ctx context.Context, input cli.ExecutionListInput) cli.ExecutionListResponse {
	selector := input.Project
	if selector == "" {
		var failure cli.Result
		selector, failure = s.EffectiveProject(ctx, "")
		if selector == "" {
			return cli.ExecutionListResponse{Result: failure}
		}
	}
	if blocked := s.gate(ctx, selector, projectapp.AdmitExecutionList); blocked != nil {
		return cli.ExecutionListResponse{Result: *blocked}
	}
	resolved := s.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return cli.ExecutionListResponse{Result: s.stageFailure(completion.ValidationFailure, resolved.Category, "Project could not be resolved", "Inspect project context", nil).Result}
	}
	attached := map[string]bool{}
	for _, r := range resolved.Project.Repositories {
		attached[r.Key] = true
	}
	if input.Repository != "" && !attached[input.Repository] {
		return cli.ExecutionListResponse{Result: s.stageFailure(completion.ValidationFailure, "repository_not_configured", "Repository is not configured", "Select an attached Repository key", nil).Result}
	}
	store, err := local.NewExecutionStore(s.stateRoot)
	if err != nil {
		return cli.ExecutionListResponse{Result: s.stageFailure(completion.Failure, "invalid_state_root", "Execution state root is unsafe", "Inspect local state", nil).Result}
	}
	c := executionComposition{s: s, store: store, project: resolved.Project}
	xs, err := store.List(ctx, resolved.Project.ID)
	if err != nil {
		return cli.ExecutionListResponse{Result: c.failure(err, nil).Result}
	}
	views := make([]cli.ExecutionSummaryView, 0, len(xs))
	for _, x := range xs {
		if !attached[x.WorkItem.RepositoryKey] || input.Repository != "" && input.Repository != x.WorkItem.RepositoryKey {
			continue
		}
		resource, id := executionWorkItemIdentity(x.WorkItem)
		stage := ""
		for _, current := range x.Stages {
			if current.Status != workflow.StageCompleted {
				stage = current.StageID
				break
			}
		}
		views = append(views, cli.ExecutionSummaryView{ExecutionID: x.ExecutionID, RepositoryKey: x.WorkItem.RepositoryKey, WorkItem: cli.ExecutionWorkItemRef{Provider: x.WorkItem.Provider, Resource: resource, ExternalID: id}, Status: string(x.Status), CurrentGate: stage, Revision: x.Revision, CreatedAt: x.CreatedAt.UTC().Format(executionTimeFormat), UpdatedAt: x.UpdatedAt.UTC().Format(executionTimeFormat)})
	}
	sort.Slice(views, func(i, j int) bool {
		a, b := views[i], views[j]
		if a.RepositoryKey != b.RepositoryKey {
			return a.RepositoryKey < b.RepositoryKey
		}
		if a.WorkItem.ExternalID != b.WorkItem.ExternalID {
			return a.WorkItem.ExternalID < b.WorkItem.ExternalID
		}
		return a.ExecutionID < b.ExecutionID
	})
	result := canonicalCompletion(completion.Facts{Completed: true}, "Canonical Executions listed", nil, "", s.provenance)
	result.Category = "executions_listed"
	return cli.ExecutionListResponse{Result: result, Executions: views, Listed: true}
}

func executionWorkItemIdentity(link workflow.WorkItemLink) (string, string) {
	if link.Provider == "github" {
		resource, id, ok := strings.Cut(link.ID, "#")
		if ok {
			return resource, id
		}
	}
	return "", link.ID
}

func (s lifecycleService) canonicalWorkflowInspection(ctx context.Context, input cli.WorkflowInput, evidence bool) cli.Result {
	response := s.executionStage(ctx, cli.WorkflowStageInput{Operation: "status", Project: input.Project, Execution: input.Execution})
	if response.Payload == nil {
		return response.Result
	}
	view, ok := response.Payload.(executionView)
	if !ok {
		return response.Result
	}
	if input.Repository != "" && input.Repository != view.WorkItem.RepositoryKey {
		return s.stageFailure(completion.ValidationFailure, "execution_selector_conflict", "Execution does not match Repository selector", "Inspect the exact Execution identity", nil).Result
	}
	resource, id := executionWorkItemIdentity(view.WorkItem)
	if input.Provider != "" && input.Provider != view.WorkItem.Provider || input.ProviderRepository != "" && input.ProviderRepository != resource || input.ExternalID != "" && input.ExternalID != id {
		return s.stageFailure(completion.ValidationFailure, "execution_selector_conflict", "Execution does not match Work Item selector", "Inspect the exact Execution identity", nil).Result
	}
	current := ""
	for _, stage := range view.Stages {
		if stage.Status != workflow.StageCompleted {
			current = stage.StageID
			break
		}
	}
	result := response.Result
	result.Workflow = &cli.WorkflowView{ExecutionKind: "canonical", ExecutionID: view.ExecutionID, WorkflowVersion: view.WorkflowRevision, Status: string(view.Status), CurrentGate: current, Revision: view.Revision, RepositoryKey: view.WorkItem.RepositoryKey, WorkItem: cli.WorkItemView{ProjectID: view.ProjectID, RepositoryKey: view.WorkItem.RepositoryKey, Provider: view.WorkItem.Provider, Resource: resource, ExternalID: id}}
	// The bounded canonical view includes immutable attempt outcomes, artifacts
	// and reconciliation facts; it never exposes invocation text or local paths.
	result.Maintenance = view
	if evidence {
		result.Category = "workflow_evidence_ready"
	}
	return result
}
