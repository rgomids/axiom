package workflow

import "context"

// GateAction describes the next supported local operation. It is derived from
// canonical state, never persisted as a second workflow or Runtime decision.
type GateAction struct {
	Operation         string            `json:"operation"`
	Gate              Stage             `json:"gate,omitempty"`
	Fact              LifecycleFactKind `json:"fact,omitempty"`
	Automatic         bool              `json:"automatic"`
	ReferenceRequired bool              `json:"referenceRequired"`
	AuthorityRequired bool              `json:"authorityRequired"`
	Active            bool              `json:"active"`
}

// NextGateAction exposes authority boundaries before any mutation. Technical
// stages still require the caller's observed result; a digest proves artifact
// integrity, not that implementation, review, or clarification succeeded.
func NextGateAction(state State) *GateAction {
	lifecycle, err := DeriveLifecycle(state)
	if err != nil {
		return nil
	}
	if state.Status == ExecutionInterrupted {
		return &GateAction{Operation: "resume"}
	}
	for _, condition := range []struct {
		active bool
		kind   LifecycleFactKind
	}{
		{lifecycle.Conditions.Blocked, FactBlocked}, {lifecycle.Conditions.NeedsDecision, FactNeedsDecision}, {lifecycle.Conditions.NeedsApproval, FactNeedsApproval},
	} {
		if condition.active {
			return &GateAction{Operation: "fact", Fact: condition.kind, ReferenceRequired: true, AuthorityRequired: true}
		}
	}
	facts, _, _ := lifecycleFacts(state)
	if state.Status == ExecutionCompleted {
		if !facts[FactHumanAcceptance] {
			return &GateAction{Operation: "fact", Fact: FactHumanAcceptance, Active: true, ReferenceRequired: true, AuthorityRequired: true}
		}
		return nil
	}
	if state.Binding != nil {
		stage := CurrentStage(state)
		required := configuredPhaseGate(*stage)
		if required != "" && !facts[required] {
			return &GateAction{Operation: "fact", Fact: required, Active: true, ReferenceRequired: true, AuthorityRequired: true}
		}
		return &GateAction{Operation: "advance", Gate: state.Stage, ReferenceRequired: true}
	}
	var required LifecycleFactKind
	switch state.Stage {
	case Plan:
		required = FactPlanningAuthority
	case Implementation:
		required = FactImplementationAuthority
	case Review:
		required = FactReviewStarted
	}
	if required != "" && !facts[required] {
		return &GateAction{Operation: "fact", Fact: required, Active: true, ReferenceRequired: true, AuthorityRequired: true}
	}
	return &GateAction{Operation: "advance", Gate: state.Stage, Automatic: state.Stage == Intake}
}

// AdvanceAutomatic only evaluates Intake: resolving the exact linked Work Item
// and valid Execution scope already supplies its deterministic prerequisites.
// All other gates need actual work/results or explicit authority. One invocation
// commits at most one transition through the same revisioned path as the CLI.
func (s Service) AdvanceAutomatic(ctx context.Context, target Target, expectedRevision uint64) Result {
	current := s.Status(ctx, target)
	if current.Status != Succeeded {
		return current
	}
	state := current.State
	if state.Revision != expectedRevision {
		return result(Denied, "stale_execution_revision", state)
	}
	action := NextGateAction(state)
	if action == nil || !action.Automatic {
		return result(Denied, "workflow_action_required", state)
	}
	return s.Transition(ctx, target, TransitionInput{ExpectedRevision: expectedRevision, Stage: Intake, Outcome: OutcomePassed})
}
