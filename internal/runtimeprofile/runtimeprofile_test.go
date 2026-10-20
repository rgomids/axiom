package runtimeprofile

import (
	"errors"
	"reflect"
	"testing"
)

func TestConfigurationClosedSchemaAndSecretReferenceSafety(t *testing.T) {
	cfg := testConfiguration()
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
	first, err := Digest(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Runtimes[0].CredentialReference = "env:AXIOM_TEST_SECRET"
	second, err := Digest(cfg)
	if err != nil || first == second {
		t.Fatalf("digest must bind reference only: first=%s second=%s err=%v", first, second, err)
	}
	invalid := []Configuration{
		{},
		{FormatVersion: 2, Revision: 1, Runtimes: cfg.Runtimes, ModelProfiles: cfg.ModelProfiles},
		{FormatVersion: 1, Revision: 1, Runtimes: []Runtime{{ID: "other", Adapter: "other", Enabled: true, AllowlistedProfileIDs: []string{"p"}}}, ModelProfiles: []ModelProfile{{ID: "p", RuntimeID: "other", Model: "m", Capabilities: []string{"go"}}}},
	}
	for index, candidate := range invalid {
		if err := Validate(candidate); !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatalf("invalid[%d] err=%v", index, err)
		}
	}
}

func testConfiguration() Configuration {
	return Configuration{
		FormatVersion: 1, Revision: 7,
		Runtimes: []Runtime{
			{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-high"}, CredentialReference: "keychain:codex"},
			{ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"claude-low"}},
		},
		ModelProfiles: []ModelProfile{
			{ID: "codex-high", RuntimeID: "codex", Model: "local-codex-model", Capabilities: []string{"go", "repository-write"}, Complexities: []string{"high"}},
			{ID: "claude-low", RuntimeID: "claude", Model: "local-claude-model", Capabilities: []string{"go"}, Complexities: []string{"low"}},
		},
		Preferences: []Preference{{Role: "implementation", Complexity: "high", ModelProfileID: "codex-high"}},
	}
}

func TestDigestDoesNotMutateCallerPreferences(t *testing.T) {
	cfg := testConfiguration()
	cfg.Preferences = []Preference{
		{Role: "z-review", Complexity: "high", ModelProfileID: cfg.ModelProfiles[0].ID},
		{Role: "a-implementation", Complexity: "high", ModelProfileID: cfg.ModelProfiles[0].ID},
	}
	before := append([]Preference(nil), cfg.Preferences...)
	if _, err := Digest(cfg); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, cfg.Preferences) {
		t.Fatal("read-only digest changed preferences")
	}
}
