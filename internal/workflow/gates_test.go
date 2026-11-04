package workflow

import (
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
