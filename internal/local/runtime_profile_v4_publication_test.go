package local

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

func v4Wire(t *testing.T, revision uint64) ([]byte, runtimeprofile.ConfigurationV4) {
	t.Helper()
	cfg := authoringConfig(revision)
	next := runtimeprofile.ConfigurationV4{FormatVersion: runtimeprofile.FormatVersionV4, Revision: revision, ModelProfiles: cfg.ModelProfiles, Preferences: cfg.Preferences}
	for _, r := range cfg.Runtimes {
		next.Runtimes = append(next.Runtimes, runtimeprofile.RuntimeV4{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs})
	}
	wire, err := runtimeprofile.EncodeV4(next)
	if err != nil {
		t.Fatal(err)
	}
	return wire, next
}

func TestRuntimeProfileV4PublicationAndExplicitRetirement(t *testing.T) {
	ctx := context.Background()
	store := authoringStore(t)
	fresh, err := store.Inspect(ctx)
	if err != nil || fresh.State != "fresh" {
		t.Fatal(fresh, err)
	}
	first, _ := v4Wire(t, 1)
	active, err := store.Publish(ctx, fresh.Token, first)
	if err != nil || active.State != "active-v4" || active.V4 == nil || active.V3 != nil || active.Revision != 1 {
		t.Fatalf("%+v %v", active, err)
	}
	if _, err := store.Publish(ctx, active.Token, v3Wire(t, 2)); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatal("format 3 replaced format 4", err)
	}
	if _, err := store.Publish(ctx, active.Token, first); err == nil {
		t.Fatal("non-monotonic revision accepted")
	}
	second, cfg := v4Wire(t, 2)
	active, err = store.Publish(ctx, active.Token, second)
	if err != nil || active.State != "active-v4" || active.Revision != 2 {
		t.Fatal(active, err)
	}
	digest, _ := runtimeprofile.DigestV4(cfg)
	tombstone, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: 3, PriorConfigurationDigest: digest})
	retired, err := store.Publish(ctx, active.Token, tombstone)
	if err != nil || retired.State != "retired" || retired.Revision != 3 {
		t.Fatal(retired, err)
	}
	again, _ := v4Wire(t, 4)
	if restarted, err := store.Publish(ctx, retired.Token, again); err != nil || restarted.State != "active-v4" {
		t.Fatal("retired record cannot restart at format 4", err)
	}

	historical, v3, _ := activeV3Store(t)
	upgraded, _ := v4Wire(t, 2)
	if _, err := historical.Publish(ctx, v3.Token, upgraded); !errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
		t.Fatal("active format 3 converted implicitly", err)
	}
}

func v3Wire(t *testing.T, revision uint64) []byte {
	t.Helper()
	cfg := runtimeprofiletest.WithRevision(runtimeprofiletest.MultiRuntime(), revision)
	return runtimeprofiletest.Encode(t, cfg)
}
func activeV3Store(t *testing.T) (RuntimeProfileStore, RuntimeProfileSnapshot, string) {
	t.Helper()
	store := authoringStore(t)
	writeLegacyRecord(t, store.root, v3Wire(t, 1))
	writeEpochRecord(t, store.root, 1)
	snapshot, err := store.Inspect(context.Background())
	if err != nil || snapshot.State != "active-v3" {
		t.Fatal(snapshot, err)
	}
	return store, snapshot, filepath.Join(store.root, "runtime-profiles", "v1", runtimeProfileObject)
}
