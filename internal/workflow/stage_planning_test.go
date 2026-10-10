package workflow

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// Stage planning reads the retained binding only: it never saves, never
// consults the current Project selection and binds inputs from validated facts.
func TestStagePlanningContextReadsRetainedBindingWithoutMutation(t *testing.T) {
	s, store, definitions, _ := newConfigured(t)
	ctx := context.Background()
	target := configuredTarget()
	started := s.Start(ctx, target)
	if started.Status != Succeeded {
		t.Fatal(started.Category)
	}
	target.ExecutionID = started.State.ExecutionID
	calls, saves, before := definitions.calls, store.saves, cloneState(store.state)
	definitions.category = "workflow_selection_required"
	planning, ready := s.StagePlanningContext(ctx, target, 1, "intake")
	if ready.Status != Succeeded || planning.ExecutionID != started.State.ExecutionID || planning.ExecutionRevision != 1 || planning.Stage.ID != "intake" || planning.Binding.Definition != started.State.Binding.Definition {
		t.Fatalf("planning context %s: %+v", ready.Category, planning)
	}
	if len(planning.Inputs) != 1 || planning.Inputs[0].Kind != "work-item" || !validDigest(planning.Inputs[0].Digest) || len(planning.MissingInputs) != 0 {
		t.Fatalf("work item input not bound: %+v", planning.Inputs)
	}
	again, _ := s.StagePlanningContext(ctx, target, 1, "intake")
	if !reflect.DeepEqual(again.Inputs, planning.Inputs) {
		t.Fatal("input binding is not deterministic")
	}
	if definitions.calls != calls || store.saves != saves || !equivalentPersistedState(store.state, before) {
		t.Fatal("planning read current selection or mutated the Execution")
	}

	for _, tc := range []struct {
		revision uint64
		stage    string
		status   Status
		category string
	}{
		{2, "intake", Denied, "stale_execution_revision"},
		{1, "missing-stage", ValidationFailed, "stage_not_found"},
		{1, "specification", ValidationFailed, "stage_prerequisite_missing"},
		{0, "intake", ValidationFailed, "invalid_execution_input"},
		{1, "Not A Key", ValidationFailed, "invalid_execution_input"},
	} {
		got, refused := s.StagePlanningContext(ctx, target, tc.revision, tc.stage)
		if refused.Status != tc.status || refused.Category != tc.category {
			t.Fatalf("%d/%s: %s %s", tc.revision, tc.stage, refused.Status, refused.Category)
		}
		if tc.category == "stage_prerequisite_missing" && !reflect.DeepEqual(got.MissingInputs, []string{"source"}) {
			t.Fatalf("missing inputs not reported: %+v", got.MissingInputs)
		}
	}
	unscoped := target
	unscoped.ExecutionID = ""
	if _, refused := s.StagePlanningContext(ctx, unscoped, 1, "intake"); refused.Category != "invalid_execution_input" {
		t.Fatal("planning accepted an Execution without its exact identity")
	}

	state := advanceConfigured(t, s, started.State)
	if _, refused := s.StagePlanningContext(ctx, target, state.Revision, "intake"); refused.Category != "stage_not_plannable" {
		t.Fatalf("completed stage planned: %s", refused.Category)
	}
	next, ready := s.StagePlanningContext(ctx, target, state.Revision, "specification")
	output, _ := priorOutput(state, "intake/result")
	if ready.Status != Succeeded || len(next.Inputs) != 1 || next.Inputs[0].Reference != "intake/result" || next.Inputs[0].Digest != output.Digest {
		t.Fatalf("stage output not bound from the validated ledger: %s %+v", ready.Category, next.Inputs)
	}
	for state.Stage != "plan" {
		state = advanceConfigured(t, s, state)
	}
	gated, ready := s.StagePlanningContext(ctx, target, state.Revision, "plan")
	if ready.Status != Succeeded || !reflect.DeepEqual(gated.Gates, []StageGate{{Kind: "planning_authority", Timing: "before", Satisfied: false}}) {
		t.Fatalf("pending human gate not reported: %s %+v", ready.Category, gated.Gates)
	}
}

// A historical Execution has no pinned definition; none is invented for it.
func TestStagePlanningContextRefusesLegacyExecution(t *testing.T) {
	s, store, _ := newS4Service(t)
	ctx := context.Background()
	started := s.Start(ctx, s4Target())
	if started.Status != Succeeded || started.State.Binding != nil {
		t.Fatalf("legacy start %s", started.Category)
	}
	target := s4Target()
	target.ExecutionID = started.State.ExecutionID
	saves := store.saves
	if _, refused := s.StagePlanningContext(ctx, target, started.State.Revision, "intake"); refused.Status != ValidationFailed || refused.Category != "configured_binding_required" || store.saves != saves {
		t.Fatalf("legacy Execution planned: %s", refused.Category)
	}
}

// A recorded fact is reported for the current stage's gate. Definitions reject
// a repeated gate kind (duplicate_gate), and planning still scopes satisfaction
// to the current stage so no earlier fact can satisfy a later stage's gate.
func TestStagePlanningReportsRecordedCurrentGate(t *testing.T) {
	s, _, _, _ := newConfigured(t)
	ctx := context.Background()
	target := configuredTarget()
	state := s.Start(ctx, target).State
	target.ExecutionID = state.ExecutionID
	for state.Stage != "plan" {
		state = advanceConfigured(t, s, state)
	}
	fact := s.RecordLifecycleFact(ctx, configuredTarget(), LifecycleFactInput{ExpectedRevision: state.Revision, Kind: FactPlanningAuthority, Active: true, Actor: "maintainer", Reference: Reference{Kind: "specification", ID: "decision", Digest: strings.Repeat("b", 64)}}, true)
	if fact.Status != Succeeded {
		t.Fatal(fact.Category)
	}
	current, ready := s.StagePlanningContext(ctx, target, fact.State.Revision, "plan")
	if ready.Status != Succeeded || len(current.Gates) != 1 || !current.Gates[0].Satisfied {
		t.Fatalf("recorded current-stage fact not reported: %s %+v", ready.Category, current.Gates)
	}
}
