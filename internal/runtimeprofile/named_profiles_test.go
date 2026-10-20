package runtimeprofile

import (
	"reflect"
	"testing"
)

func TestMultipleNamedProfilesRemainSelectable(t *testing.T) {
	cfg, err := DecodeV4(readV4Fixture(t, "configuration-v4.json"))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"review-go", "build-tools", "docs-only", "integration-go"}
	for _, name := range names {
		for i := range cfg.Runtimes {
			if cfg.Runtimes[i].ID == "codex" {
				cfg.Runtimes[i].AllowlistedProfileIDs = append(cfg.Runtimes[i].AllowlistedProfileIDs, name)
			}
		}
		cfg.ModelProfiles = append(cfg.ModelProfiles, ModelProfile{ID: name, RuntimeID: "codex", Model: "synthetic-codex-model", Effort: Effort{Mode: EffortRuntimeDefault, Value: "default"}, Capabilities: []string{"go"}, Complexities: []string{"high"}})
	}
	consumer, err := NewConsumerV4(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		first, err := consumer.ObservedProfile("codex", name, testOperationalObservation("codex"))
		if err != nil || first.ProfileID != name {
			t.Fatalf("%s: %+v %v", name, first, err)
		}
		again, err := consumer.ObservedProfile("codex", name, testOperationalObservation("codex"))
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatal("unstable profile selection", err)
		}
	}
}
