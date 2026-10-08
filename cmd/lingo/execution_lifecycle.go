package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
)

var _ cli.ExecutionListService = lifecycleService{}

const executionTimeFormat = "2006-01-02T15:04:05.999999999Z07:00"

// WorkflowList discovers Executions of one Project without a known identity.
// It is classified inspection: it is admitted for an archived Project, never
// consults a Provider and writes nothing. Sequential workflow Executions have
// no cancellation contract, so no cancel verb exists (issue #230, F-02).
func (s lifecycleService) WorkflowList(ctx context.Context, input cli.ExecutionListInput) cli.ExecutionListResponse {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionList); blocked != nil {
		return cli.ExecutionListResponse{Result: *blocked}
	}
	listed := s.workflows.List(ctx, workflow.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository})
	if listed.Status != workflow.Succeeded {
		failure := workflowResult(listed, s.provenance)
		return cli.ExecutionListResponse{Result: failure}
	}
	views := make([]cli.ExecutionSummaryView, 0, len(listed.Executions))
	for _, item := range listed.Executions {
		views = append(views, cli.ExecutionSummaryView{
			ExecutionID: item.ExecutionID, RepositoryKey: item.RepositoryKey,
			WorkItem: cli.ExecutionWorkItemRef{Provider: item.WorkItem.Provider, Resource: item.WorkItem.Resource, ExternalID: item.WorkItem.ExternalID},
			Status:   string(item.Status), CurrentGate: string(item.Stage), LifecycleStage: string(item.LifecycleStage), Revision: item.Revision,
			CreatedAt: item.CreatedAt.UTC().Format(executionTimeFormat), UpdatedAt: item.UpdatedAt.UTC().Format(executionTimeFormat),
		})
	}
	result := canonicalCompletion(completion.Facts{Completed: true}, "Executions listed for the Project", nil, "", s.provenance)
	result.Category = listed.Category
	return cli.ExecutionListResponse{Result: result, Executions: views, Listed: true}
}
