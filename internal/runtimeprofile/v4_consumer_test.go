package runtimeprofile

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testOperationalObservation(runtimeID string) OperationalObservation {
	return OperationalObservation{RuntimeID: runtimeID, Adapter: runtimeID, Installed: true, Available: true, ExecutableDigest: strings.Repeat("a", 64), InvocationMode: "noninteractive", Revision: 7, EffortSourceDigest: strings.Repeat("b", 64), ObservedAt: time.Unix(1, 0)}
}

func TestV4ConsumerRequiresExactAllowedIdentityWithoutCertification(t *testing.T) {
	cfg, err := DecodeV4(readV4Fixture(t, "configuration-v4.json"))
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := NewConsumerV4(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, runtimeID := range []string{"codex", "claude"} {
		o := testOperationalObservation(runtimeID)
		resolved, err := consumer.ObservedProfile(runtimeID, runtimeID+"-default", o)
		if err != nil || resolved.ConfigurationDigest != consumer.Digest() || resolved.ConfigurationRevision != consumer.Revision() || resolved.Model != "synthetic-"+runtimeID+"-model" || resolved.Effort != (Effort{Mode: EffortRuntimeDefault, Value: "default"}) {
			t.Fatalf("exact observation: %+v %v", resolved, err)
		}
		o.ObservedAt = o.ObservedAt.Add(time.Hour)
		later, err := consumer.ObservedProfile(runtimeID, runtimeID+"-default", o)
		if err != nil || !reflect.DeepEqual(resolved, later) {
			t.Fatal("collection time changed binding", err)
		}
		for name, change := range map[string]func(*OperationalObservation){
			"stale-revision":             func(o *OperationalObservation) { o.Revision++ },
			"missing-executable":         func(o *OperationalObservation) { o.Installed = false },
			"unavailable":                func(o *OperationalObservation) { o.Available = false },
			"invalid-identity":           func(o *OperationalObservation) { o.ExecutableDigest = "invalid" },
			"missing-effort-observation": func(o *OperationalObservation) { o.EffortSourceDigest = "" },
			"different-runtime":          func(o *OperationalObservation) { o.RuntimeID = "other" },
			"different-adapter":          func(o *OperationalObservation) { o.Adapter = "other" },
			"missing-time":               func(o *OperationalObservation) { o.ObservedAt = time.Time{} },
			"different-mode":             func(o *OperationalObservation) { o.InvocationMode = "interactive" },
		} {
			t.Run(runtimeID+"/"+name, func(t *testing.T) {
				o := testOperationalObservation(runtimeID)
				change(&o)
				if _, err := consumer.ObservedProfile(runtimeID, runtimeID+"-default", o); !errors.Is(err, ErrNoMatch) {
					t.Fatal("unsafe observation accepted", err)
				}
			})
		}
		for name, change := range map[string]func(*OperationalObservation){
			"executable-drift": func(o *OperationalObservation) { o.ExecutableDigest = strings.Repeat("c", 64) },
			"source-drift":     func(o *OperationalObservation) { o.EffortSourceDigest = strings.Repeat("d", 64) },
		} {
			drifted := testOperationalObservation(runtimeID)
			change(&drifted)
			fresh, err := consumer.ObservedProfile(runtimeID, runtimeID+"-default", drifted)
			if err != nil || fresh == resolved || fresh.ObservationDigest == resolved.ObservationDigest {
				t.Fatal("fresh identity/source did not change tuple", name, err)
			}
		}
	}
	if _, err := consumer.AllowedProfile("codex", "claude-default"); !errors.Is(err, ErrProfileRuntimeMismatch) {
		t.Fatal("cross-Runtime selection accepted", err)
	}
	if _, err := consumer.AllowedProfile("codex", "missing"); !errors.Is(err, ErrProfileMissing) {
		t.Fatal("missing Profile substituted", err)
	}
	if _, err := consumer.EnabledRuntime("other"); !errors.Is(err, ErrRuntimeAbsent) {
		t.Fatal("absent Runtime substituted", err)
	}
}

func TestV4ConsumerCopiesConfigurationAndPreservesDisabledUnlistedProfiles(t *testing.T) {
	cfg, _ := DecodeV4(readV4Fixture(t, "configuration-v4.json"))
	cfg.ModelProfiles = append(cfg.ModelProfiles, ModelProfile{ID: "unlisted", RuntimeID: "codex", Model: "synthetic-model", Effort: Effort{Mode: EffortExplicit, Value: "future-effort"}, Capabilities: []string{"read"}})
	consumer, err := NewConsumerV4(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Runtimes[0].AllowlistedProfileIDs[0] = "mutated"
	copy := consumer.Configuration()
	copy.ModelProfiles[0].Capabilities[0] = "mutated"
	runtime, err := consumer.EnabledRuntime("codex")
	if err != nil {
		t.Fatal(err)
	}
	runtime.AllowlistedProfileIDs[0] = "mutated"
	profile, err := consumer.AllowedProfile("codex", "codex-default")
	if err != nil {
		t.Fatal("consumer aliased input/output", err)
	}
	profile.Capabilities[0] = "mutated"
	profile, err = consumer.AllowedProfile("codex", "codex-default")
	if err != nil || profile.Capabilities[0] != "read" {
		t.Fatal("Profile aliases configuration", err)
	}
	if _, err := consumer.AllowedProfile("codex", "unlisted"); !errors.Is(err, ErrProfileNotAllowlisted) {
		t.Fatal("unlisted Profile accepted", err)
	}
	cfg = consumer.Configuration()
	for i := range cfg.Runtimes {
		cfg.Runtimes[i].Enabled = false
	}
	disabled, _ := NewConsumerV4(cfg)
	if _, err := disabled.ObservedProfile("codex", "codex-default", testOperationalObservation("codex")); !errors.Is(err, ErrRuntimeDisabled) {
		t.Fatal("disabled Runtime admitted", err)
	}
}
