package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// publishRuntimeProfileV3 initializes canonical v3 local configuration through
// the only writer, the exact-token Publish (Specification 007 PM-1/PM-11).
func publishRuntimeProfileV3(t *testing.T, stateRoot string, cfg runtimeprofile.ConfigurationV3) local.RuntimeProfileSnapshot {
	t.Helper()
	store, err := local.NewRuntimeProfileStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := store.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Revision = fresh.Revision + 1
	wire, err := runtimeprofile.EncodeV3(cfg)
	if err != nil {
		t.Fatal(err)
	}
	active, err := store.Publish(context.Background(), fresh.Token, wire)
	if err != nil || active.State != "active-v3" {
		t.Fatalf("publish canonical v3: %+v %v", active, err)
	}
	return active
}

// singleRuntimeV3 is one enabled Runtime allowlisting one runtime-default
// Profile; it carries no declaration and claims no capability.
func singleRuntimeV3(runtimeID, profileID, model string, capabilities, complexities []string) runtimeprofile.ConfigurationV3 {
	return runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 1,
		Runtimes:      []runtimeprofile.RuntimeV3{{ID: runtimeID, Adapter: runtimeID, Enabled: true, AllowlistedProfileIDs: []string{profileID}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: profileID, RuntimeID: runtimeID, Model: model, Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, Capabilities: capabilities, Complexities: complexities}},
	}
}

// writeLegacyRuntimeProfile places pre-stable format-1/2 bytes exactly as an
// earlier release left them; no current writer produces them.
func writeLegacyRuntimeProfile(t *testing.T, stateRoot string, cfg runtimeprofile.Configuration) []byte {
	t.Helper()
	wire, err := runtimeprofile.Encode(cfg)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(stateRoot, "runtime-profiles", "v1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{stateRoot, filepath.Join(stateRoot, "runtime-profiles"), dir} {
		if err := os.Chmod(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "configuration.json"), wire, 0o600); err != nil {
		t.Fatal(err)
	}
	return wire
}
