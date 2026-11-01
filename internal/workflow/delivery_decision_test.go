package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type decisionMemory struct{ value *DeliveryDecision }

func (m *decisionMemory) LoadDeliveryDecision(context.Context, string, string) (DeliveryDecision, error) {
	if m.value == nil {
		return DeliveryDecision{}, ErrNotFound
	}
	return *m.value, nil
}
func (m *decisionMemory) RecordDeliveryDecision(_ context.Context, _ State, d DeliveryDecision) error {
	if m.value != nil && !SameDeliveryDecision(*m.value, d) {
		return ErrConflict
	}
	copy := d
	m.value = &copy
	return nil
}
func decisionFixture(t *testing.T, outcome string) (State, DeliveryDecision) {
	t.Helper()
	state, publisher, _, _, preview := publicationFixture(t)
	r, err := publisher.Apply(context.Background(), state, preview, publicationAuthority(preview))
	if err != nil {
		t.Fatal(err)
	}
	d, err := PrepareDeliveryDecision(context.Background(), state, preview.Packet, &r, outcome, "maintainer", "correct the bounded scope", Reference{Kind: "decision", ID: "human-authority", Digest: strings.Repeat("e", 64)}, []Reference{}, configuredReferences{}, "/fixture")
	if err != nil {
		t.Fatal(err)
	}
	return state, d
}
func TestDeliveryDecisionHumanAuthorityExactAcceptanceAndConflict(t *testing.T) {
	state, d := decisionFixture(t, "accept")
	ctx := context.Background()
	store := &decisionMemory{}
	for _, scenario := range []string{"human", "local", "digest", "actor", "stale", "publication"} {
		copy, s := d, state
		human, local, ref := true, true, d.Digest()
		switch scenario {
		case "human":
			human = false
		case "local":
			local = false
		case "digest":
			ref = strings.Repeat("f", 64)
		case "actor":
			copy.ActorKind = "agent"
			ref = copy.Digest()
		case "stale":
			s.StorageRevision[0] ^= 1
		case "publication":
			copy.Publication = nil
			ref = copy.Digest()
		}
		if _, err := ApplyDeliveryDecision(ctx, store, s, copy, ref, local, human); err == nil || store.value != nil {
			t.Fatal("decision bypass", scenario)
		}
	}
	if _, err := ApplyDeliveryDecision(ctx, store, state, d, d.Digest(), true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyDeliveryDecision(ctx, store, state, d, d.Digest(), true, true); err != nil {
		t.Fatal("replay", err)
	}
	conflict := d
	conflict.Outcome = "reject"
	if _, err := ApplyDeliveryDecision(ctx, store, state, conflict, conflict.Digest(), true, true); err == nil {
		t.Fatal("conflicting decision")
	}
	state.DeliveryDecision = &d
	status, accepted := DeliveryDisposition(state)
	lifecycle, err := DeriveLifecycle(state)
	if err != nil || !accepted || status != "accepted" || lifecycle.Stage != LifecycleAccepted {
		t.Fatal(status, lifecycle, err)
	}
	d.Outcome = "reject"
	status, accepted = DeliveryDisposition(state)
	lifecycle, err = DeriveLifecycle(state)
	if err != nil || accepted || status != "rejected" || lifecycle.Stage == LifecycleAccepted {
		t.Fatal(status, lifecycle, err)
	}
}
func TestDeliveryReworkExplicitRevisionIndependentGatesAndInputs(t *testing.T) {
	original, d := decisionFixture(t, "reject")
	original.DeliveryDecision = &d
	s, _, definitions, _ := newConfigured(t)
	input := ReworkInput{NewExecutionID: "00000000-0000-4000-8000-000000000099", RevisionChoice: "retained", RuntimeID: original.RuntimeID, RuntimePreview: strings.Repeat("f", 64), Scope: "repair reviewed finding", Plan: Reference{Kind: "plan", ID: "corrective-plan", Digest: strings.Repeat("e", 64)}, Authority: Reference{Kind: "decision", ID: "new-execution-authority", Digest: strings.Repeat("f", 64)}, Inputs: []Reference{}}
	r2 := definitions.document.Definition
	r2.Revision = 2
	r2.Stages[0].Instructions = "new selected revision"
	doc, issues := workflowdefinition.Encode(r2)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	definitions.document = doc
	for _, choice := range []string{"retained", "selected"} {
		input.RevisionChoice = choice
		p, err := s.PreviewDeliveryRework(context.Background(), original, d, input, configuredTarget())
		if err != nil {
			t.Fatal(choice, err)
		}
		expected := 1
		if choice == "selected" {
			expected = 2
		}
		if p.NewState.Binding.Definition.Revision != expected || len(p.NewState.Transitions) != 0 || p.NewState.StageOrdinal != 0 || p.NewState.Revision != 1 || p.NewState.ExecutionID == original.ExecutionID {
			t.Fatal("invalid fresh binding", p.NewState)
		}
	}
	input.RevisionChoice = ""
	if _, err := s.PreviewDeliveryRework(context.Background(), original, d, input, configuredTarget()); err == nil {
		t.Fatal("implicit revision")
	}
	input.RevisionChoice = "retained"
	input.Inputs = []Reference{{Kind: "artifact", ID: "tampered", Digest: strings.Repeat("f", 64)}}
	if _, err := s.PreviewDeliveryRework(context.Background(), original, d, input, configuredTarget()); err == nil {
		t.Fatal("unbound prior input")
	}
	input.Inputs = nil
	input.Authority = d.Authority
	if _, err := s.PreviewDeliveryRework(context.Background(), original, d, input, configuredTarget()); err == nil {
		t.Fatal("inherited authority")
	}
}
