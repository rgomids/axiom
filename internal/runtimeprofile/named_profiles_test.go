package runtimeprofile

import (
	"context"
	"reflect"
	"testing"
)

// Operator names are references, not a fixed set of three quality tiers.
func TestMultipleNamedProfilesPerRuntimeResolveDeterministically(t *testing.T) {
	cfg := testConfiguration()
	names := []string{"review-go", "build-tools", "docs-only", "integration-go"}
	for _, name := range names {
		cfg.Runtimes[0].AllowlistedProfileIDs = append(cfg.Runtimes[0].AllowlistedProfileIDs, name)
		cfg.ModelProfiles = append(cfg.ModelProfiles, ModelProfile{ID: name, RuntimeID: "codex", Model: "local-codex-model", Capabilities: []string{"go"}, Complexities: []string{"high"}})
	}
	observer := &fakeObserver{observations: map[string]Observation{"codex": testObservation("codex", "codex", true), "claude": testObservation("claude", "claude", true)}}
	resolver := NewResolver(observer)
	for _, name := range names {
		req := Request{ConfigurationRevision: 7, Role: "implementation", Complexity: "high", Capabilities: []string{"go"}, ModelProfileID: name}
		first, err := resolver.Resolve(context.Background(), cfg, req)
		if err != nil || first.Choice == nil || first.Choice.ModelProfileID != name {
			t.Fatalf("profile %s: %+v %v", name, first, err)
		}
		again, err := resolver.Resolve(context.Background(), cfg, req)
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatalf("profile %s not deterministic", name)
		}
	}
}
