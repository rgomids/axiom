package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func TestRuntimeProfileStorePreservesRevisionAndCompletePriorGeneration(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewRuntimeProfileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := store.Inspect(ctx)
	first, err := store.Publish(ctx, fresh.Token, authoringWire(t, 1))
	if err != nil {
		t.Fatal(err)
	}
	store.hooks.fault = func(stage FaultStage) error {
		if stage == FaultF2 {
			return ErrSimulatedInterruption
		}
		return nil
	}
	if _, err := store.Publish(ctx, first.Token, authoringWire(t, 2)); err == nil {
		t.Fatal("faulted update succeeded")
	}
	store.hooks = publicationHooks{}
	if _, err := store.Inspect(ctx); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("interrupted reader err=%v", err)
	}
	wire, err := os.ReadFile(filepath.Join(root, "runtime-profiles", "v1", "configuration.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := runtimeprofile.DecodeVersioned(wire)
	if err != nil || loaded.V4 == nil || loaded.V4.Revision != 1 {
		t.Fatalf("canonical=%+v err=%v", loaded, err)
	}
	root = filepath.Join(t.TempDir(), "state")
	store, err = NewRuntimeProfileStore(root)
	if err != nil {
		t.Fatalf("fresh store err=%v", err)
	}
	fresh, _ = store.Inspect(ctx)
	first, err = store.Publish(ctx, fresh.Token, authoringWire(t, 1))
	if err != nil {
		t.Fatalf("fresh create err=%v", err)
	}
	if _, err := store.Publish(ctx, first.Token, authoringWire(t, 3)); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale revision err=%v", err)
	}
}

func runtimeConfiguration(revision uint64) runtimeprofile.Configuration {
	return runtimeprofile.Configuration{FormatVersion: 1, Revision: revision,
		Runtimes:      []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-default"}}, {ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"claude-default"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-default", RuntimeID: "codex", Model: "local-codex", Capabilities: []string{"go"}}, {ID: "claude-default", RuntimeID: "claude", Model: "local-claude", Capabilities: []string{"go"}}},
	}
}
