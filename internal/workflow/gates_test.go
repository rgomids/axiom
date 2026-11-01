package workflow

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestNextActionGateAndAuthorityMatrix(t *testing.T) {
	state := lifecycleTestState()
	for _, stage := range Stages() {
		if state.Stage != stage {
			t.Fatalf("stage = %s, want %s", state.Stage, stage)
		}
		var fact LifecycleFactKind
		switch stage {
		case Plan:
			fact = FactPlanningAuthority
		case Implementation:
			fact = FactImplementationAuthority
		case Review:
			fact = FactReviewStarted
		}
		if fact != "" {
			action := NextGateAction(state)
			if action == nil || action.Operation != "fact" || action.Fact != fact || !action.Active || !action.AuthorityRequired || !action.ReferenceRequired {
				t.Fatalf("authority at %s = %#v", stage, action)
			}
			addLifecycleFact(&state, fact, true, map[LifecycleFactKind]string{FactPlanningAuthority: "specification", FactImplementationAuthority: "plan", FactReviewStarted: "evidence"}[fact])
		}
		action := NextGateAction(state)
		if action == nil || action.Operation != "advance" || action.Gate != stage || action.Automatic != (stage == Intake) {
			t.Fatalf("action at %s = %#v", stage, action)
		}
		if stage != Completion {
			advanceLifecycleState(&state)
		}
	}
	completeLifecycleState(&state)
	accept := NextGateAction(state)
	if accept == nil || accept.Fact != FactHumanAcceptance || !accept.AuthorityRequired {
		t.Fatalf("acceptance = %#v", accept)
	}
	addLifecycleFact(&state, FactHumanAcceptance, true, "evidence")
	if NextGateAction(state) != nil {
		t.Fatal("accepted execution has another action")
	}
}

func TestExplicitMissingAuthorityAndInvalidReferenceDoNotAdvance(t *testing.T) {
	service, store, _ := newS4Service(t)
	target := s4Target()
	service.Start(context.Background(), target)
	for _, stage := range []Stage{Intake, Specification, Clarification} {
		service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: store.state.Revision, Stage: stage, Outcome: OutcomePassed})
	}
	before := cloneState(store.state)
	denied := service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: before.Revision, Stage: Plan, Outcome: OutcomePassed})
	if denied.Category != "workflow_authority_required" || !reflect.DeepEqual(before, store.state) {
		t.Fatalf("authority denial = %#v", denied)
	}
	service.references = rejectingGateReferences{}
	invalid := service.RecordLifecycleFact(context.Background(), target, LifecycleFactInput{ExpectedRevision: before.Revision, Kind: FactPlanningAuthority, Active: true, Reference: Reference{Kind: "specification", ID: "missing", Digest: testDigest}}, true)
	if invalid.Status != ValidationFailed || !reflect.DeepEqual(before, store.state) {
		t.Fatalf("invalid Evidence = %#v", invalid)
	}
}

type rejectingGateReferences struct{}

func (rejectingGateReferences) Validate(context.Context, string, string, Reference) error {
	return errors.New("unavailable")
}
