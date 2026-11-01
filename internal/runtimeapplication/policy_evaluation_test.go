package runtimeapplication

import (
	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
	"testing"
)

func TestConsumerFromRecordStates(t *testing.T) {
	valid := fx.Current(fx.CodexOnly())
	invalid := fx.Current(fx.CodexOnly())
	invalid.ModelProfiles[0].Effort = runtimeprofile.Effort{}
	for _, tc := range []struct {
		state string
		cfg   *runtimeprofile.ConfigurationV4
		want  string
	}{
		{"active-v4", &valid, ""},
		{"fresh", nil, "configuration_missing"},
		{"retired", nil, "configuration_retired"},
		{"active-v4", &invalid, "invalid_configuration"},
		{"active-v4", nil, "invalid_configuration"},
		{"active-v3", nil, "unsupported_configuration_format"},
		{"active-v1", nil, "unsupported_configuration_format"},
		{"active-v2", nil, "unsupported_configuration_format"},
		{"pending", nil, "configuration_recovery_required"},
		{"unsafe", nil, "configuration_unsafe"},
		{"", nil, "configuration_unsafe"},
	} {
		consumer, code := ConsumerFromRecord(tc.state, tc.cfg)
		if code != tc.want || (code == "") != (consumer.Digest() != "") {
			t.Fatalf("%s: got %q want %q", tc.state, code, tc.want)
		}
	}
}

func TestReadinessCandidates(t *testing.T) {
	cfg := fx.WithRuntimeDisabled(fx.MultiRuntime(), fx.ClaudeRuntimeID)
	before := fx.Encode(t, cfg)
	current := fx.Current(cfg)
	candidates, code := ReadinessCandidates("active-v4", &current)
	if code != "" || len(candidates) != 1 || candidates[0].RuntimeID != fx.CodexRuntimeID || len(candidates[0].Profiles) != 2 {
		t.Fatalf("disabled Runtime or unlisted Profile disclosed: %+v %s", candidates, code)
	}
	for _, intent := range candidates[0].Profiles {
		if intent.ProfileID == fx.CodexUnlistedProfileID || intent.Model != "synthetic-codex-model" || !runtimeprofile.ValidEffort(intent.Effort) || intent.ConfigurationDigest != fx.CurrentDigest(t, cfg) {
			t.Fatalf("candidate lacks exact intent: %+v", intent)
		}
	}
	if string(fx.Encode(t, cfg)) != string(before) {
		t.Fatal("projection mutated its input")
	}
	multi := fx.Current(fx.MultiRuntime())
	all, code := ReadinessCandidates("active-v4", &multi)
	if code != "" || len(all) != 2 {
		t.Fatalf("enabled Runtimes not all disclosed: %+v", all)
	}
	invalid := fx.Current(fx.CodexOnly())
	invalid.Runtimes[0].AllowlistedProfileIDs = append(invalid.Runtimes[0].AllowlistedProfileIDs, fx.UnknownProfileID)
	for state, want := range map[string]string{"fresh": "configuration_missing", "retired": "configuration_retired", "active-v2": "unsupported_configuration_format", "active-v3": "unsupported_configuration_format"} {
		if got, code := ReadinessCandidates(state, nil); got != nil || code != want {
			t.Fatalf("%s: got %q want %q", state, code, want)
		}
	}
	if got, code := ReadinessCandidates("active-v4", &invalid); got != nil || code != "invalid_configuration" {
		t.Fatalf("invalid configuration disclosed: %q", code)
	}
}
