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

// Specification 007 AXM-22 D5/D8: v2-only local paths refuse format 3
// explicitly, never read it as v2, never publish it and never mutate state.
func TestRuntimeProfileV2OnlyPathsRefuseV3(t *testing.T) {
	ctx := context.Background()
	v3 := validV3Wire(t)
	store := authoringStore(t)
	fresh, err := store.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(ctx, fresh.Token, v3); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("fresh v3 publication: %v", err)
	}
	if after, _ := store.Inspect(ctx); after.State != "fresh" {
		t.Fatal("refused publication mutated state")
	}
	v2 := authoringWire(t, 1)
	active, err := store.Publish(ctx, fresh.Token, v2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(ctx, active.Token, v3); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("v3 publication over v2: %v", err)
	}
	path := filepath.Join(store.root, "runtime-profiles", "v1", "configuration.json")
	if current, _ := os.ReadFile(path); !bytes.Equal(current, v2) {
		t.Fatal("historical v2 bytes changed by refused publication")
	}
	if snapshot, err := store.Inspect(ctx); err != nil || snapshot.State != "active-v2" || snapshot.Configuration.FormatVersion != runtimeprofile.FormatVersionV2 {
		t.Fatal("v2 record migrated", err)
	}

	// A format-3 record present on disk is preserved and refused by readers.
	if err := os.WriteFile(path, v3, 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Inspect(ctx)
	if !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) || !errors.Is(err, ErrUnsafe) || snapshot.State == "active-v1" || snapshot.State == "active-v2" || snapshot.Configuration.FormatVersion != 0 {
		t.Fatalf("v2-only snapshot reader accepted v3: %s %v", snapshot.State, err)
	}
	if _, err := store.Load(ctx); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("v2-only load accepted v3: %v", err)
	}
	if err := store.Save(ctx, authoringConfig(2)); err == nil {
		t.Fatal("legacy save replaced a v3 record")
	}
	if kind := kindOf(t, store.root, "runtime-profiles/v1/configuration.json"); kind != InventoryNewer {
		t.Fatalf("inventory classification %s", kind)
	}
	if current, _ := os.ReadFile(path); !bytes.Equal(current, v3) {
		t.Fatal("refusal mutated the v3 record")
	}
}
