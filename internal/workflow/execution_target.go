package workflow

import (
	"context"
	"regexp"
	"strconv"
)

// ResolvedTarget is an ephemeral application snapshot. Execution v1 remains
// the canonical persisted identity; Repository.Path is never presented to users.
type ResolvedTarget struct {
	DefinitionObservation string
	ProjectID             string
	Repository            Repository
	WorkItem              WorkItem
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
	resolved := ResolvedTarget{ProjectID: project.ID, Repository: repository, WorkItem: item}
	if s.definitions != nil {
		observation, category := s.definitions.ResolveDefinition(ctx, target.ProjectSelector)
		if category != "" {
			return ResolvedTarget{}, result(ValidationFailed, category, State{})
		}
		if observation.ProjectID != project.ID || !validDefinitionObservation(observation) {
			return ResolvedTarget{}, result(Failed, "recovery_required", State{})
		}
		resolved.DefinitionObservation = observation.Digest
	}
	return resolved, Result{}
}

// linearIdentifierPattern accepts Linear issue identifiers (TEAM-N).
var linearIdentifierPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,9}-[1-9][0-9]{0,9}$`)

func validWorkItemSelector(value string) bool {
	number, err := strconv.Atoi(value)
	if err == nil && number > 0 && strconv.Itoa(number) == value {
		return true
	}
	return ValidLinearWorkItemSelector(value)
}

// ValidLinearWorkItemSelector reports whether value is a Linear issue identifier (TEAM-N).
func ValidLinearWorkItemSelector(value string) bool {
	return linearIdentifierPattern.MatchString(value)
}
