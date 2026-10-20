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

func operationalWire(t *testing.T, revision uint64) []byte {
	t.Helper()
	return authoringWire(t, revision)
}
func activeOperationalStore(t *testing.T) (RuntimeProfileStore, RuntimeProfileSnapshot, string) {
	t.Helper()
	ctx := context.Background()
	store := authoringStore(t)
	fresh, err := store.Inspect(ctx)
	if err != nil || fresh.State != "fresh" {
		t.Fatal(fresh, err)
	}
	active, err := store.Publish(ctx, fresh.Token, authoringWire(t, 1))
	if err != nil || active.State != "active-v4" || active.V4 == nil {
		t.Fatal(active, err)
	}
	return store, active, filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject)
}
func TestRuntimeProfileOperationalPublication(t *testing.T) {
	ctx := context.Background()
	store, active, path := activeOperationalStore(t)
	if current, _ := os.ReadFile(path); !bytes.Equal(current, operationalWire(t, 1)) {
		t.Fatal("initialized bytes differ from the canonical candidate")
	}
	if _, err := store.Publish(ctx, active.Token, legacyWire(t, 2, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatalf("legacy publication replaced v4: %v", err)
	}
	next := operationalWire(t, 2)
	published, err := store.Publish(ctx, active.Token, next)
	if err != nil || published.State != "active-v4" || published.Revision != 2 || published.V4 == nil {
		t.Fatalf("%+v %v", published, err)
	}
	if current, _ := os.ReadFile(path); !bytes.Equal(current, next) {
		t.Fatal("published bytes differ from the canonical candidate")
	}
	versioned, err := store.Inspect(ctx)
	if err != nil || versioned.Token != published.Token || versioned.V4.Revision != 2 {
		t.Fatalf("%+v %v", versioned, err)
	}

}
func TestRuntimeProfileOperationalPublicationRefusals(t *testing.T) {
	ctx := context.Background()
	t.Run("revision-and-canonical", func(t *testing.T) {
		store := authoringStore(t)
		fresh, _ := store.Inspect(ctx)
		indented := append([]byte(" "), operationalWire(t, 1)...)
		for name, wire := range map[string][]byte{"skipped": operationalWire(t, 2), "non-canonical": indented} {
			if _, err := store.Publish(ctx, fresh.Token, wire); err == nil {
				t.Fatalf("%s accepted", name)
			}
		}
		if after, _ := store.Inspect(ctx); after.State != "fresh" {
			t.Fatal("refusal mutated fresh state")
		}
	})
	t.Run("downgrade", func(t *testing.T) {
		store, active, path := activeOperationalStore(t)
		digest, _ := runtimeprofile.Digest(legacyConfig(2, 1))
		tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
		if _, err := store.Publish(ctx, active.Token, legacyWire(t, 2, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("v2 over v4: %v", err)
		}
		if _, err := store.Publish(ctx, active.Token, tomb); !errors.Is(err, ErrConflict) {
			t.Fatalf("historical-digest tombstone over v4: %v", err)
		}
		if current, _ := os.ReadFile(path); !bytes.Equal(current, operationalWire(t, 1)) {
			t.Fatal("downgrade refusal mutated v4")
		}
	})
	t.Run("v2", func(t *testing.T) {
		store := authoringStore(t)
		writeLegacyRecord(t, store.root, legacyWire(t, 2, 1))
		v2, err := store.Inspect(ctx)
		if err != nil || v2.State != "active-v2" {
			t.Fatalf("%+v %v", v2, err)
		}
		if _, err := store.Publish(ctx, v2.Token, operationalWire(t, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("v2 → v4: %v", err)
		}
		if current, _ := os.ReadFile(filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject)); !bytes.Equal(current, legacyWire(t, 2, 1)) {
			t.Fatal("refusal mutated v2")
		}
	})
	t.Run("v1", func(t *testing.T) {
		store := authoringStore(t)
		writeLegacyRecord(t, store.root, legacyWire(t, 1, 1))
		v1, err := store.Inspect(ctx)
		if err != nil || v1.State != "active-v1" {
			t.Fatalf("%+v %v", v1, err)
		}
		if _, err := store.Publish(ctx, v1.Token, operationalWire(t, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			t.Fatalf("v1 → v4: %v", err)
		}
	})
	t.Run("retired", func(t *testing.T) {
		store := authoringStore(t)
		fresh, _ := store.Inspect(ctx)
		active, err := store.Publish(ctx, fresh.Token, authoringWire(t, 1))
		if err != nil {
			t.Fatal(err)
		}
		digest, _ := runtimeprofile.DigestV4(authoringConfig(1))
		tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 2, PriorConfigurationDigest: digest})
		retired, err := store.Publish(ctx, active.Token, tomb)
		if err != nil {
			t.Fatal(err)
		}
		if next, err := store.Publish(ctx, retired.Token, operationalWire(t, 3)); err != nil || next.State != "active-v4" || next.Revision != 3 {
			t.Fatalf("retired → v4: %+v %v", next, err)
		}
	})
}
func TestRuntimeProfileOperationalPublicationRecovery(t *testing.T) {
	ctx := context.Background()
	for _, stage := range []FaultStage{FaultF3, FaultF6} {
		t.Run(string(stage), func(t *testing.T) {
			store, active, _ := activeOperationalStore(t)
			v4 := operationalWire(t, 2)
			store.hooks.fault = func(at FaultStage) error {
				if at == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			if _, err := store.Publish(ctx, active.Token, v4); err == nil {
				t.Fatal("interrupted publication reported success")
			}
			store.hooks = publicationHooks{}
			pending, err := store.Inspect(ctx)
			if !errors.Is(err, ErrRecoveryRequired) || pending.State != "pending" {
				t.Fatalf("%+v %v", pending, err)
			}
			if _, err := store.Publish(ctx, active.Token, v4); err == nil {
				t.Fatal("publication over pending state")
			}
			plan, found, err := store.InspectRuntimeRecovery(ctx)
			if err != nil || !found {
				t.Fatalf("%+v %v", plan, err)
			}
			authority, err := AuthorizeRecovery(plan, plan.Digest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.ApplyRuntimeRecovery(ctx, plan, authority); err != nil {
				t.Fatal(err)
			}
			recovered, err := store.Inspect(ctx)
			if err != nil || recovered.State != "active-v4" {
				t.Fatalf("%+v %v", recovered, err)
			}
			current, _ := os.ReadFile(filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject))
			if recovered.Revision == 1 {
				if stage == FaultF6 {
					t.Fatal("committed publication rolled back")
				}
				if !bytes.Equal(current, operationalWire(t, 1)) {
					t.Fatal("rollback did not restore the exact source bytes")
				}
				return
			}
			if !bytes.Equal(current, v4) {
				t.Fatal("recovered v4 bytes differ")
			}
			if _, err := store.Publish(ctx, active.Token, v4); err == nil {
				t.Fatal("pre-recovery preview token reused")
			}
		})
	}
}
