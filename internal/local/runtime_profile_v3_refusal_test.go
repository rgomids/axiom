package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// validV3Wire is the canonical synthetic fixture owned by runtimeprofile; it
// must be a valid format-3 record so refusals below are version refusals.
func validV3Wire(t *testing.T) []byte {
	t.Helper()
	wire, err := os.ReadFile(filepath.Join("..", "runtimeprofile", "testdata", "configuration-v3.json"))
	if err != nil {
		t.Fatal(err)
	}
	if v, err := runtimeprofile.DecodeVersioned(wire); err != nil || v.FormatVersion != runtimeprofile.FormatVersionV3 {
		t.Fatal("fixture is not valid format 3", err)
	}
	return wire
}

// Specification 007 AXM-22 PM-11 (B1): the single reader recognizes canonical
// v3, keeps a future format unsupported and never reinterprets it, and
// refusals never mutate state.
func TestRuntimeProfileReaderRecognizesV3AndRefusesFuture(t *testing.T) {
	ctx := context.Background()
	v3 := validV3Wire(t)
	store := authoringStore(t)
	fresh, err := store.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// PM-1 admits v3 over fresh state only as the exact revision-1 candidate;
	// this revision-7 fixture is not, so nothing is written.
	if _, err := store.Publish(ctx, fresh.Token, v3); !errors.Is(err, ErrConflict) {
		t.Fatalf("fresh v3 publication: %v", err)
	}
	if after, _ := store.Inspect(ctx); after.State != "fresh" {
		t.Fatal("refused publication mutated state")
	}
	writeLegacyRecord(t, store.root, v3)
	path := filepath.Join(store.root, "runtime-profiles", "v1", "configuration.json")
	// Every canonical v3 record is published after an authorization epoch
	// advance (PM-6); without that record the state is unsafe, not active.
	if snapshot, err := store.Inspect(ctx); !errors.Is(err, ErrUnsafe) || snapshot.V3 != nil {
		t.Fatalf("v3 without authorization epoch: %+v %v", snapshot, err)
	}
	writeEpochRecord(t, store.root, 1)
	snapshot, err := store.Inspect(ctx)
	if err != nil || snapshot.State != "active-v3" || snapshot.V3 == nil || snapshot.Legacy.FormatVersion != 0 {
		t.Fatalf("v3 record not recognized: %+v %v", snapshot, err)
	}
	if kind := kindOf(t, store.root, "runtime-profiles/v1/configuration.json"); kind != InventoryRuntimeProfile {
		t.Fatalf("inventory classification %s", kind)
	}
	future := bytes.Replace(v3, []byte(`"formatVersion":3`), []byte(`"formatVersion":4`), 1)
	if err := os.WriteFile(path, future, 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err = store.Inspect(ctx)
	if !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) || !errors.Is(err, ErrUnsafe) || snapshot.V3 != nil {
		t.Fatalf("future format accepted: %s %v", snapshot.State, err)
	}
	if kind := kindOf(t, store.root, "runtime-profiles/v1/configuration.json"); kind != InventoryNewer {
		t.Fatalf("future inventory classification %s", kind)
	}
	if current, _ := os.ReadFile(path); !bytes.Equal(current, future) {
		t.Fatal("refusal mutated the future record")
	}
}
