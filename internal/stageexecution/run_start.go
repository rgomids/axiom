package stageexecution

import (
	"context"
	"errors"
	"fmt"

	"github.com/rgomids/axiom/internal/workflow"
)

// ErrWorkItemLinkMismatch refuses dispatch when the stored Work Item link
// names another repository identity than the Project binding.
var ErrWorkItemLinkMismatch = errors.New("work_item_link_mismatch")

// LinkedWorkItem is the already-linked traceability identity as stored
// locally. Provider projection is optional and never consulted here.
type LinkedWorkItem struct {
	Provider, ID, RepositoryKey, RepositoryIdentity string
}

// StartRequest is the one run action's resolved input: linked context, the
// selected Workflow (or the minimal default) and nothing else. There is no
// Stage Plan, Validator Policy, preview digest, expected revision or Profile.
type StartRequest struct {
	ExecutionID  string
	ProjectID    string
	Repositories []workflow.RepositoryBinding
	WorkItem     LinkedWorkItem
	// Workflow is empty for the minimal default.
	WorkflowID, WorkflowRevision string
	Stages                       []workflow.StageSpec
}

// creator is the create half of the Execution store.
type creator interface {
	Create(context.Context, workflow.Execution) error
}

// Start validates the linked Work Item against the Project repository
// binding, creates the Execution and progresses eligible work in the same
// action. A Provider being unavailable cannot block it: no Provider is read
// or written.
func (e Engine) Start(ctx context.Context, store creator, r StartRequest) (workflow.Execution, error) {
	identity := ""
	for _, repo := range r.Repositories {
		if repo.Key == r.WorkItem.RepositoryKey {
			identity = repo.Identity
		}
	}
	if identity == "" || r.WorkItem.RepositoryIdentity != identity {
		return workflow.Execution{}, fmt.Errorf("%w: Work Item %s is linked to %q, the Project binds %q to %q", ErrWorkItemLinkMismatch, r.WorkItem.ID, r.WorkItem.RepositoryIdentity, r.WorkItem.RepositoryKey, identity)
	}
	id, rev, stages := r.WorkflowID, r.WorkflowRevision, r.Stages
	if len(stages) == 0 {
		if id != "" || rev != "" {
			return workflow.Execution{}, fmt.Errorf("%w: selected Workflow has no stages", workflow.ErrExecutionInvalid)
		}
		id, rev, stages = workflow.DefaultWorkflowID, workflow.DefaultWorkflowRevision, workflow.DefaultWorkflowStages()
	}
	x, err := workflow.NewExecution(workflow.NewExecutionInput{
		ExecutionID: r.ExecutionID, ProjectID: r.ProjectID,
		WorkItem:     workflow.WorkItemLink{Provider: r.WorkItem.Provider, ID: r.WorkItem.ID, RepositoryKey: r.WorkItem.RepositoryKey},
		Repositories: r.Repositories, WorkflowID: id, Revision: rev, Stages: stages, Now: e.now(),
	})
	if err != nil {
		return workflow.Execution{}, err
	}
	if err := store.Create(ctx, x); err != nil {
		return workflow.Execution{}, err
	}
	return e.Run(ctx, r.ProjectID, r.ExecutionID, RunOptions{})
}
