package workflow

import (
	"context"
	"strings"
	"testing"
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
