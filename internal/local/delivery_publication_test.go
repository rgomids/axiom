package local

import (
	"context"
	"github.com/rgomids/axiom/internal/workflow"
	"os"
	"path/filepath"
	"testing"
)

func TestHistoricalDeliveryPublicationReadbackAndTamper(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	state := createLocalV3Execution(t, store)
	packet := localDeliveryPacket(state)
	ref := seedHistoricalPacket(t, store, packet)
	r := workflow.DeliveryPublication{FormatVersion: 1, Kind: "delivery_publication", Preview: workflow.DeliveryPreview{Packet: packet, PacketDigest: ref, Target: workflow.DeliveryTarget{Kind: "local"}}, Status: "failure", Push: "not_requested", PR: "not_requested", Next: "inspect"}
	name := publicationName(r.Preview.Digest())
	historicalFixtureWire(t, root, "executions", state.ProjectID, name, mustHistoricalWire(workflow.EncodeDeliveryPublication(r)))
	got, err := store.LoadDeliveryPublication(ctx, state.ProjectID, r.Preview.Digest())
	if err != nil || got.Status != "failure" {
		t.Fatal(got, err)
	}
	path := filepath.Join("executions", "v1", state.ProjectID, name)
	if got := kindOf(t, root, filepath.ToSlash(path)); got != InventoryDeliveryPublication {
		t.Fatal(got)
	}
	if err = os.WriteFile(filepath.Join(root, path), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.LoadDeliveryPublication(ctx, state.ProjectID, r.Preview.Digest()); err == nil {
		t.Fatal("tamper accepted")
	}
}
