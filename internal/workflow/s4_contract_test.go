package workflow

import "testing"

func TestS4ClosedStateRejectsIncoherentHistoryAndDuplicateProjection(t *testing.T) {
	state := lifecycleTestState()
	state.Stage = Specification
	state.Revision = 2
	state.Transitions = []Transition{{Revision: 2, From: Intake, To: Plan, Outcome: OutcomePassed, RequestDigest: digest("request"), CommittedAt: state.UpdatedAt, Provenance: state.Provenance}}
	if ValidState(state) {
		t.Fatal("skipped stage accepted")
	}
	state.Transitions[0].To = Specification
	record := ProjectionRecord{ExecutionRevision: 2, Key: projectionKey(state.ExecutionID, 2)}
	state.Projections = []ProjectionRecord{record, record}
	if ValidState(state) {
		t.Fatal("duplicate projection revision accepted")
	}
}
