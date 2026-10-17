package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func TestDeliveryPublicationStoreFenceCASReadbackAndTamper(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	store, _ := NewWorkflowStore(root)
	state := createLocalV3Execution(t, store)
	packet := localDeliveryPacket(state)
	ref, err := store.RecordDeliveryPacket(ctx, state, packet)
	if err != nil {
		t.Fatal(err)
	}
	r := workflow.DeliveryPublication{FormatVersion: 1, Kind: "delivery_publication", Preview: workflow.DeliveryPreview{Packet: packet, PacketDigest: ref, Target: workflow.DeliveryTarget{Kind: "local"}}, Status: "failure", Push: "not_requested", PR: "not_requested", Next: "inspect"}
	if err = store.SaveDeliveryPublication(ctx, state, r, nil); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadDeliveryPublication(ctx, state.ProjectID, r.Preview.Digest())
	if err != nil || got.Status != "failure" {
		t.Fatal(got, err)
	}
	if err = store.SaveDeliveryPublication(ctx, state, r, nil); err == nil {
		t.Fatal("duplicate fence accepted")
	}
	next := r
	next.Next = "reconcile"
	if err = store.SaveDeliveryPublication(ctx, state, next, &r); err != nil {
		t.Fatal(err)
	}
	if err = store.SaveDeliveryPublication(ctx, state, next, &r); err == nil {
		t.Fatal("stale CAS accepted")
	}
	if list, err := store.List(ctx, state.ProjectID); err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}
	path := filepath.Join(root, "executions", "v1", state.ProjectID, publicationName(r.Preview.Digest()))
	if err = os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.LoadDeliveryPublication(ctx, state.ProjectID, r.Preview.Digest()); err == nil {
		t.Fatal("tamper accepted")
	}
}
func TestDeliveryPublicationStoreFailurePrecedesEffect(t *testing.T) {
	ctx := context.Background()
	store, _ := NewWorkflowStore(privateTestRoot(t))
	state := createLocalV3Execution(t, store)
	p := localDeliveryPacket(state)
	ref, _ := workflow.DeliveryPacketDigest(p)
	r := workflow.DeliveryPublication{FormatVersion: 1, Kind: "delivery_publication", Preview: workflow.DeliveryPreview{Packet: p, PacketDigest: ref, Target: workflow.DeliveryTarget{Kind: "local"}}, Status: "failure", Push: "not_requested", PR: "not_requested", Next: "inspect"}
	store.hooks.beforeCommit = func() error { return errors.New("synthetic fault") }
	if err := store.SaveDeliveryPublication(ctx, state, r, nil); err == nil {
		t.Fatal("fault ignored")
	}
}
