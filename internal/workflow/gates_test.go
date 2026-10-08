package workflow

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestAutomaticGateUsesCanonicalTransition(t *testing.T) {
	service, store, provider := newS4Service(t)
	target := s4Target()
	service.Start(context.Background(), target)
	advanced := service.AdvanceAutomatic(context.Background(), target, 1)
	if advanced.Status != Succeeded || advanced.State.Stage != Specification || advanced.State.Revision != 2 || len(advanced.State.Transitions) != 1 || len(provider.effects) != 0 {
		t.Fatalf("advance = %#v", advanced)
	}
	if !ValidState(store.state) {
		t.Fatal("automatic transition produced invalid history")
	}
	before := cloneState(store.state)
	for _, revision := range []uint64{1, 2} {
		denied := service.AdvanceAutomatic(context.Background(), target, revision)
		if denied.Status != Denied || !reflect.DeepEqual(before, store.state) {
			t.Fatalf("unexpected advance: %#v", denied)
		}
	}
}

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

func TestAutomaticGateStopsAtConditionsAndInterruption(t *testing.T) {
	for _, kind := range []LifecycleFactKind{FactBlocked, FactNeedsDecision, FactNeedsApproval} {
		t.Run(string(kind), func(t *testing.T) {
			service, store, provider := newS4Service(t)
			service.references = acceptingLifecycleReferences{}
			target := s4Target()
			service.Start(context.Background(), target)
			service.RecordLifecycleFact(context.Background(), target, LifecycleFactInput{ExpectedRevision: 1, Kind: kind, Active: true, Reference: Reference{Kind: "evidence", ID: "condition", Digest: testDigest}}, true)
			before := cloneState(store.state)
			action := NextGateAction(before)
			if action == nil || action.Fact != kind || action.Active {
				t.Fatalf("condition action = %#v", action)
			}
			for _, automatic := range []bool{true, false} {
				var result Result
				if automatic {
					result = service.AdvanceAutomatic(context.Background(), target, before.Revision)
				} else {
					result = service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: before.Revision, Stage: Intake, Outcome: OutcomePassed})
				}
				if result.Status != Denied || !reflect.DeepEqual(before, store.state) || len(provider.effects) != 0 {
					t.Fatalf("denial = %#v", result)
				}
			}
		})
	}
	service, store, _ := newS4Service(t)
	target := s4Target()
	service.Start(context.Background(), target)
	failed := service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: 1, Stage: Intake, Outcome: OutcomeFailed})
	if failed.State.Stage != Intake || NextGateAction(failed.State).Operation != "resume" {
		t.Fatalf("failure = %#v", failed)
	}
	before := cloneState(store.state)
	if result := service.AdvanceAutomatic(context.Background(), target, before.Revision); result.Status != Denied || !reflect.DeepEqual(before, store.state) {
		t.Fatalf("automatic resume = %#v", result)
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

func TestAutomaticGateCancellationAndSaveFailure(t *testing.T) {
	service, store, _ := newS4Service(t)
	target := s4Target()
	service.Start(context.Background(), target)
	before := cloneState(store.state)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := service.AdvanceAutomatic(ctx, target, 1); result.Status != Interrupted || !reflect.DeepEqual(before, store.state) {
		t.Fatalf("cancel = %#v", result)
	}
	store.failSaveAt = 1
	if result := service.AdvanceAutomatic(context.Background(), target, 1); result.Status == Succeeded || !reflect.DeepEqual(before, store.state) {
		t.Fatalf("save failure = %#v", result)
	}
	corrupt := before
	corrupt.Stage = Stage("unknown")
	if NextGateAction(corrupt) != nil {
		t.Fatal("corrupt state exposes mutation action")
	}
}
