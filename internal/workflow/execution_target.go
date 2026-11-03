package workflow

import (
	"context"
	"strconv"
)

// ResolvedTarget is an ephemeral linked-input snapshot; Repository.Path is never presented to users.
type ResolvedTarget struct {
	ProjectID  string
	Repository Repository
	WorkItem   WorkItem
}

// ResolveLinked validates the Project, Repository and existing Work Item link
// without requiring a Workflow selection: the
// canonical run uses the minimal default when none is selected. It reads no
// Provider and writes nothing.
func (s Service) ResolveLinked(ctx context.Context, target Target) (ResolvedTarget, Result) {
	if ctx.Err() != nil {
		return ResolvedTarget{}, result(Interrupted, "workflow_cancelled", State{})
	}
	if target.ExecutionID != "" {
		return ResolvedTarget{}, result(ValidationFailed, "execution_selector_conflict", State{})
	}
	project, repository, item, failed := s.resolve(ctx, target)
	if failed.Category != "" {
		return ResolvedTarget{}, failed
	}
	return ResolvedTarget{ProjectID: project.ID, Repository: repository, WorkItem: item}, Result{}
}

func validWorkItemSelector(value string) bool {
	number, err := strconv.Atoi(value)
	return err == nil && number > 0 && strconv.Itoa(number) == value
}
