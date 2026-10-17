package local

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func localDeliveryPacket(state workflow.State) workflow.DeliveryPacket {
	binding, _ := json.Marshal(state.Binding)
	return workflow.DeliveryPacket{FormatVersion: 1, Kind: "delivery_packet", ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, WorkItem: state.WorkItem, ExecutionID: state.ExecutionID, ExecutionRevision: state.Revision, LedgerDigest: hexDigestRevision(state.StorageRevision), BindingDigest: hexDigest(binding), Definition: state.Binding.Definition, Stages: []workflow.DeliveryStage{}, Blockers: []string{"execution:not_completed"}, Ready: false, Next: "resolve packet blockers through the retained workflow gates", Provenance: state.Provenance}
}

func TestDeliveryPacketStoreCASReadbackTamperAndInventory(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	state := createLocalV3Execution(t, store)
	before, _ := os.ReadFile(executionPath(t, root, state))
	packet := localDeliveryPacket(state)
	ref, err := store.RecordDeliveryPacket(ctx, state, packet)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.RecordDeliveryPacket(ctx, state, packet)
	if err != nil || again != ref {
		t.Fatal("not idempotent", err)
	}
	if _, err := store.LoadDeliveryPacket(ctx, state.ProjectID, ref); err != nil {
		t.Fatal(err)
	}
	if list, err := store.List(ctx, state.ProjectID); err != nil || len(list) != 1 {
		t.Fatal("packet breaks execution list", err)
	}
	after, _ := os.ReadFile(executionPath(t, root, state))
	if string(before) != string(after) {
		t.Fatal("ledger mutated")
	}
	if got := kindOf(t, root, filepath.ToSlash(filepath.Join("executions", "v1", state.ProjectID, ref+".json"))); got != InventoryDeliveryPacket {
		t.Fatal(got)
	}
	stale := state
	stale.StorageRevision[0] ^= 1
	packet = localDeliveryPacket(stale)
	if _, err := store.RecordDeliveryPacket(ctx, stale, packet); !errors.Is(err, ErrConflict) {
		t.Fatal("stale generation accepted", err)
	}
	path := filepath.Join(root, "executions", "v1", state.ProjectID, ref+".json")
	wire, _ := os.ReadFile(path)
	wire = append(wire, []byte("{}")...)
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadDeliveryPacket(ctx, state.ProjectID, ref); err == nil {
		t.Fatal("tamper accepted")
	}
}

func TestDeliveryPacketPublicationFailureIsProtected(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	state := createLocalV3Execution(t, store)
	store.hooks.beforeCommit = func() error { return errors.New("injected publication failure") }
	if _, err := store.RecordDeliveryPacket(ctx, state, localDeliveryPacket(state)); err == nil {
		t.Fatal("publication fault ignored")
	}
}

func TestDeliveryPacketReferencesRemainRetained(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	state := createLocalV3Execution(t, store)
	packet := localDeliveryPacket(state)
	packet.Stages = []workflow.DeliveryStage{{Ordinal: 0, StageID: "intake", Artifacts: []workflow.Reference{{Kind: "artifact", ID: "00000000-0000-4000-8000-000000000008", Digest: strings.Repeat("e", 64)}}}}
	digest, err := store.RecordDeliveryPacket(ctx, state, packet)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := existingPrivateRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	refs, err := scanArtifactReferences(handle, nil)
	if err != nil || !slices.Contains(refs[packet.Stages[0].Artifacts[0].ID], "delivery-packet:"+digest) {
		t.Fatalf("packet references lost: %+v %v", refs, err)
	}
}
