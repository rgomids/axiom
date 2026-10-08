package workflow

import (
	"context"
	"strconv"
)

// ResolvedTarget is an ephemeral application snapshot. Execution v1 remains
// the canonical persisted identity; Repository.Path is never presented to users.
type ResolvedTarget struct {
	ProjectID  string
	Repository Repository
	WorkItem   WorkItem
}

// ValidateTarget performs the same authoritative resolution used by Start,
// without allocating an Execution, observing a Runtime, or mutating state.
func (s Service) ValidateTarget(ctx context.Context, target Target) (ResolvedTarget, Result) {
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
