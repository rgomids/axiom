package local

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/workflow"
)

func localReworkFixture(t *testing.T, store WorkflowStore, state workflow.State) (workflow.State, workflow.ReworkPreview) {
	t.Helper()
	ctx := context.Background()
	packet := localDeliveryPacket(state)
	ref, err := store.RecordDeliveryPacket(ctx, state, packet)
	if err != nil {
		t.Fatal(err)
	}
	d := workflow.DeliveryDecision{FormatVersion: 1, Kind: "delivery_decision", Packet: packet, PacketDigest: ref, Outcome: "reject", ActorKind: "human-operator", Actor: "fixture-human", Authority: workflow.Reference{Kind: "decision", ID: "human", Digest: strings.Repeat("e", 64)}, Reason: "repair required output", Evidence: []workflow.Reference{}}
	if err = store.RecordDeliveryDecision(ctx, state, d); err != nil {
		t.Fatal(err)
	}
	state, err = store.Load(ctx, state.ProjectID, state.RepositoryKey, state.WorkItem)
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
	p := workflow.ReworkPreview{FormatVersion: 1, Kind: "delivery_rework", OriginalPacket: packet, DecisionDigest: d.Digest(), Input: workflow.ReworkInput{NewExecutionID: next.ExecutionID, RevisionChoice: "retained", ResolvedProfile: next.Binding.ResolvedProfile, RuntimeID: next.RuntimeID, RuntimePreview: next.Binding.RuntimePreview, Scope: d.Reason, Plan: workflow.Reference{Kind: "plan", ID: "corrective-plan", Digest: strings.Repeat("a", 64)}, Authority: workflow.Reference{Kind: "decision", ID: "new-authority", Digest: strings.Repeat("b", 64)}, Inputs: []workflow.Reference{}}, NewState: next}
	if _, err = workflow.EncodeReworkPreview(p); err != nil {
		t.Fatal(err)
	}
	return state, p
}
func TestDeliveryReworkRetentionReplayDowngradeAndCAS(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	original, p := localReworkFixture(t, store, createLocalV3Execution(t, store))
	before, _ := os.ReadFile(executionPath(t, root, original))
	if _, err := workflow.ApplyDeliveryRework(ctx, store, original, p, p.Digest(), true, false); err == nil {
		t.Fatal("missing new authority")
	}
	next, err := workflow.ApplyDeliveryRework(ctx, store, original, p, p.Digest(), true, true)
	if err != nil {
		t.Fatal(err)
	}
	historical, err := store.LoadExecutionID(ctx, original.ProjectID, original.RepositoryKey, original.WorkItem, original.ExecutionID)
	if err != nil || historical.DeliveryDecision == nil || historical.ExecutionID != original.ExecutionID {
		t.Fatal(historical, err)
	}
	retained, _ := os.ReadFile(filepath.Join(root, "executions", "v1", original.ProjectID, retainedExecutionName(original.ExecutionID)))
	if !bytes.Equal(before, retained) {
		t.Fatal("historical bytes changed")
	}
	active, _ := os.ReadFile(executionPath(t, root, next))
	if _, err = decodeExecutionSnapshot(active); err == nil {
		t.Fatal("old writer accepted active envelope")
	}
	if _, err = store.ApplyDeliveryRework(ctx, original, p); err != nil {
		t.Fatal("replay failed", err)
	}
	changed := p
	changed.Input.Scope = "different"
	if _, err = store.ApplyDeliveryRework(ctx, original, changed); err == nil {
		t.Fatal("conflicting rework accepted")
	}
	states, err := store.List(ctx, original.ProjectID)
	if err != nil || len(states) != 2 {
		t.Fatal("duplicate or lost execution", len(states), err)
	}
	next.Projections = []workflow.ProjectionRecord{}
	if err = store.Save(ctx, next); err != nil {
		t.Fatal("active writer failed", err)
	}
	if err = store.Save(ctx, original); err == nil {
		t.Fatal("historical writer accepted")
	}
}
func TestDeliveryReworkPublicationFaultRecoveryNeverCreatesThirdExecution(t *testing.T) {
	for _, boundary := range []int{1, 2, 3} {
		for _, stage := range []FaultStage{FaultF3, FaultF6, FaultF8} {
			t.Run(string(stage)+string(rune('0'+boundary)), func(t *testing.T) {
				ctx := context.Background()
				root := privateTestRoot(t)
				store, _ := NewWorkflowStore(root)
				original, p := localReworkFixture(t, store, createLocalV3Execution(t, store))
				count := 0
				store.hooks.fault = func(s FaultStage) error {
					if s == stage {
						count++
						if count == boundary {
							return ErrSimulatedInterruption
						}
					}
					return nil
				}
				if _, err := store.ApplyDeliveryRework(ctx, original, p); err == nil {
					t.Fatal("fault ignored")
				}
				store.hooks = publicationHooks{}
				if _, err := store.Load(ctx, original.ProjectID, original.RepositoryKey, original.WorkItem); !errors.Is(err, workflow.ErrRecoveryRequired) {
					t.Fatal("interrupted write admitted", err)
				}
				roots := RecoveryRoots{State: root}
				report, err := InspectRecovery(ctx, roots)
				if err != nil || len(report.Plans) != 1 {
					t.Fatal(report, err)
				}
				plan := report.Plans[0]
				authority, err := AuthorizeRecovery(plan, plan.Digest)
				if err != nil {
					t.Fatal(plan, err)
				}
				if _, err = ApplyRecovery(ctx, roots, plan, authority); err != nil {
					t.Fatal(err)
				}
				if _, err = store.ApplyDeliveryRework(ctx, original, p); err != nil {
					t.Fatal("replay after recovery", err)
				}
				states, err := store.List(ctx, original.ProjectID)
				if err != nil || len(states) != 2 {
					t.Fatal(len(states), err)
				}
			})
		}
	}
}
func TestDeliveryReworkConcurrentApplyHasOneActiveIdentity(t *testing.T) {
	ctx := context.Background()
	store, _ := NewWorkflowStore(privateTestRoot(t))
	original, p := localReworkFixture(t, store, createLocalV3Execution(t, store))
	var wait sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() { defer wait.Done(); _, err := store.ApplyDeliveryRework(ctx, original, p); errs <- err }()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	list, err := store.List(ctx, original.ProjectID)
	if err != nil || len(list) != 2 {
		t.Fatal(list, err)
	}
}
func TestDeliveryDecisionConflictingReplayAndStrictCodec(t *testing.T) {
	ctx := context.Background()
	store, _ := NewWorkflowStore(privateTestRoot(t))
	original, _ := localReworkFixture(t, store, createLocalV3Execution(t, store))
	d := *original.DeliveryDecision
	if err := store.RecordDeliveryDecision(ctx, original, d); err != nil {
		t.Fatal(err)
	}
	d.Reason = "contradiction"
	if err := store.RecordDeliveryDecision(ctx, original, d); err == nil {
		t.Fatal("conflict accepted")
	}
	wire, _ := json.Marshal(d)
	if _, err := workflow.DecodeDeliveryDecision(append(wire, []byte(`{}`)...)); err == nil {
		t.Fatal("trailing content")
	}
}

func TestDeliveryReworkLineageTamperFailsClosed(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	original, p := localReworkFixture(t, store, createLocalV3Execution(t, store))
	if _, err := store.ApplyDeliveryRework(ctx, original, p); err != nil {
		t.Fatal(err)
	}
	p.Input.Scope = "tampered lineage"
	wire, err := workflow.EncodeReworkPreview(p)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, "executions", "v1", original.ProjectID, reworkName(original.ExecutionID))
	if err = os.WriteFile(name, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Load(ctx, original.ProjectID, original.RepositoryKey, original.WorkItem); err == nil {
		t.Fatal("tampered lineage admitted")
	}
}
