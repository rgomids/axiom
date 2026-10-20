package runtimeprofile_test

import (
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

func syntheticV3Fixtures() map[string]runtimeprofile.ConfigurationV3 {
	return map[string]runtimeprofile.ConfigurationV3{
		"codex only":            fx.CodexOnly(),
		"claude only":           fx.ClaudeOnly(),
		"multiple runtimes":     fx.MultiRuntime(),
		"nonempty preferences":  fx.WithPreferences(fx.MultiRuntime()),
		"revision changed":      fx.WithRevision(fx.MultiRuntime(), 7),
		"disabled runtime":      fx.WithRuntimeDisabled(fx.MultiRuntime(), fx.ClaudeRuntimeID),
		"no applicable effort":  fx.WithoutDeclarations(fx.MultiRuntime(), fx.CodexRuntimeID),
		"all declarations gone": fx.WithoutDeclarations(fx.WithoutDeclarations(fx.MultiRuntime(), fx.CodexRuntimeID), fx.ClaudeRuntimeID),
	}
}

func TestV3FixturesAreCanonicalAndRoundTrip(t *testing.T) {
	for name, cfg := range syntheticV3Fixtures() {
		wire := fx.Encode(t, cfg)
		versioned, err := runtimeprofile.DecodeVersioned(wire)
		if err != nil || versioned.FormatVersion != runtimeprofile.FormatVersionV3 || versioned.Configuration != nil {
			t.Fatalf("%s: decode %v", name, err)
		}
		if again := fx.Encode(t, *versioned.V3); string(again) != string(wire) {
			t.Fatalf("%s: historical decode changed canonical bytes", name)
		}
		if versioned.V3.Revision != cfg.Revision {
			t.Fatalf("%s: revision %d", name, versioned.V3.Revision)
		}
	}
	// Builders are deterministic and never share mutable state.
	a, b := fx.MultiRuntime(), fx.MultiRuntime()
	a.Runtimes[0].EffortDeclarations[0].Sources[0].Path = "changed/config.toml"
	a.ModelProfiles[0].Capabilities[0] = "changed"
	if !reflect.DeepEqual(b, fx.MultiRuntime()) {
		t.Fatal("fixture builders share state")
	}
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
