package local

import (
	"github.com/rgomids/axiom/internal/workflow"
	"strings"
	"testing"
	"time"
)

func localReworkFixture(t *testing.T, store WorkflowStore, state workflow.State) (workflow.State, workflow.ReworkPreview) {
	t.Helper()
	packet := localDeliveryPacket(state)
	ref := seedHistoricalPacket(t, store, packet)
	var err error
	d := workflow.DeliveryDecision{FormatVersion: 1, Kind: "delivery_decision", Packet: packet, PacketDigest: ref, Outcome: "reject", ActorKind: "human-operator", Actor: "fixture-human", Authority: workflow.Reference{Kind: "decision", ID: "human", Digest: strings.Repeat("e", 64)}, Reason: "repair required output", Evidence: []workflow.Reference{}}
	historicalFixtureWire(t, store.root, "executions", state.ProjectID, decisionName(state.ExecutionID), mustHistoricalWire(workflow.EncodeDeliveryDecision(d)))
	state, err = readHistoricalExecutionFixture(store, state)
	if err != nil {
		t.Fatal(err)
	}
	next := state
	binding := *state.Binding
	next.Binding = &binding
	next.DeliveryDecision = nil
	next.ExecutionID = "00000000-0000-4000-8000-000000000099"
	next.StorageRevision = [32]byte{}
	next.ReworkOf = nil
	next.CreatedAt = state.UpdatedAt.Add(time.Nanosecond)
	next.UpdatedAt = next.CreatedAt
	next.Binding.BoundAt = next.CreatedAt
	next.Binding.SnapshotRef = "execution:" + next.ExecutionID + "#/binding/snapshot"
	p := workflow.ReworkPreview{FormatVersion: 1, Kind: "delivery_rework", OriginalPacket: packet, DecisionDigest: d.Digest(), Input: workflow.ReworkInput{NewExecutionID: next.ExecutionID, RevisionChoice: "retained", ResolvedProfile: next.Binding.ResolvedProfile, FrozenProfile: next.Binding.FrozenProfile, RuntimeID: next.RuntimeID, RuntimePreview: next.Binding.RuntimePreview, Scope: d.Reason, Plan: workflow.Reference{Kind: "plan", ID: "corrective-plan", Digest: strings.Repeat("a", 64)}, Authority: workflow.Reference{Kind: "decision", ID: "new-authority", Digest: strings.Repeat("b", 64)}, Inputs: []workflow.Reference{}}, NewState: next}
	if _, err = workflow.EncodeReworkPreview(p); err != nil {
		t.Fatal(err)
	}
	return state, p
}
