package local

import (
	"context"
	"encoding/json"
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

func seedHistoricalPacket(t *testing.T, store WorkflowStore, packet workflow.DeliveryPacket) string {
	t.Helper()
	wire := mustHistoricalWire(workflow.EncodeDeliveryPacket(packet))
	historicalFixtureDigestWire(t, store.root, "executions", packet.ProjectID, wire)
	return hexDigest(wire)
}

func TestHistoricalDeliveryPacketReadbackTamperAndInventory(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	state := createLocalV3Execution(t, store)
	before, _ := os.ReadFile(executionPath(t, root, state))
	ref := seedHistoricalPacket(t, store, localDeliveryPacket(state))
	if _, err := store.LoadDeliveryPacket(ctx, state.ProjectID, ref); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(executionPath(t, root, state))
	if string(before) != string(after) {
		t.Fatal("reader mutated ledger")
	}
	path := filepath.Join("executions", "v1", state.ProjectID, ref+".json")
	if got := kindOf(t, root, filepath.ToSlash(path)); got != InventoryDeliveryPacket {
		t.Fatal(got)
	}
	if err := os.WriteFile(filepath.Join(root, path), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadDeliveryPacket(ctx, state.ProjectID, ref); err == nil {
		t.Fatal("tamper accepted")
	}
}

func TestDeliveryPacketReferencesRemainRetained(t *testing.T) {
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	state := createLocalV3Execution(t, store)
	packet := localDeliveryPacket(state)
	packet.Stages = []workflow.DeliveryStage{{Ordinal: 0, StageID: "intake", Artifacts: []workflow.Reference{{Kind: "artifact", ID: "00000000-0000-4000-8000-000000000008", Digest: strings.Repeat("e", 64)}}}}
	digest := seedHistoricalPacket(t, store, packet)
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
