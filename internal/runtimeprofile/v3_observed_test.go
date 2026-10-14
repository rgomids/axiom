package runtimeprofile_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	fx "github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

// observedCodex is an injected, current observation of the synthetic Codex
// executable; proof is attached only when proven is set.
func observedCodex(c runtimeprofile.ConfigurationV3, proven bool) runtimeprofile.Observation {
	o := runtimeprofile.Observation{RuntimeID: fx.CodexRuntimeID, Adapter: fx.CodexRuntimeID, Installed: true, Available: true, Version: fx.CodexRuntimeVersion, ExecutableDigest: fx.CodexExecutableDigest, Revision: c.Revision, ObservedAt: time.Unix(1, 0), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{fx.SyntheticCapability: runtimeprofile.CapabilityProven}}
	if proven {
		p := c.ModelProfiles[0]
		o.ProfileProofs = []runtimeprofile.ProfileProof{{RuntimeID: fx.CodexRuntimeID, Model: p.Model, Effort: p.Effort, ExecutableDigest: o.ExecutableDigest, RuntimeVersion: o.Version, InvocationMode: fx.InvocationMode, ObservationRevision: o.Revision, Reference: "injected-proof", EffortSourceDigest: strings.Repeat("e", 64), Authentication: runtimeprofile.CapabilityProven, ModelAvailability: runtimeprofile.CapabilityProven, EffortSupport: runtimeprofile.CapabilityProven, NoOverride: runtimeprofile.CapabilityProven, CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{fx.SyntheticCapability: runtimeprofile.CapabilityProven}}}
	}
	return o
}

// BLD-001B.3c-2b.3: the observed choice reuses the existing proof rule bound
// to DigestV3. An applicable EffortDeclaration is never capability proof, and
// stale, unavailable or unproven observations fail closed.
func TestConsumerV3ObservedChoiceAppliesExistingProofGates(t *testing.T) {
	cfg := fx.CodexOnly()
	consumer := fx.Consumer(t, cfg)
	profile := cfg.ModelProfiles[0].ID
	if _, err := consumer.EffortDeclaration(fx.CodexApplicability()); err != nil {
		t.Fatal("fixture lost its applicable declaration", err)
	}
	if _, code := consumer.ObservedChoice(fx.CodexRuntimeID, profile, observedCodex(cfg, false), []string{fx.SyntheticCapability}); code != "effort_unproven" {
		t.Fatalf("declaration promoted to proof: %s", code)
	}
	if code := consumer.ProfileReadiness(fx.CodexRuntimeID, profile, observedCodex(cfg, false)); code != "authentication_unproven" {
		t.Fatalf("readiness without proof: %s", code)
	}
	choice, code := consumer.ObservedChoice(fx.CodexRuntimeID, profile, observedCodex(cfg, true), []string{fx.SyntheticCapability})
	if code != "" || choice.ResolvedProfile == nil || choice.ResolvedProfile.ConfigurationDigest != fx.Digest(t, cfg) || choice.ResolvedProfile.ConfigurationRevision != cfg.Revision || choice.RuntimeVersion != fx.CodexRuntimeVersion {
		t.Fatalf("proven choice %+v %s", choice, code)
	}
	if pd, _ := runtimeprofile.ProfileDigest(cfg.ModelProfiles[0]); choice.ResolvedProfile.ProfileDigest != pd {
		t.Fatal("ProfileDigest changed")
	}
	if code := consumer.ProfileReadiness(fx.CodexRuntimeID, profile, observedCodex(cfg, true)); code != "" {
		t.Fatalf("proven readiness: %s", code)
	}
	for name, tc := range map[string]struct {
		change func(*runtimeprofile.Observation)
		caps   []string
		want   string
	}{
		"stale revision":  {func(o *runtimeprofile.Observation) { o.Revision++ }, []string{fx.SyntheticCapability}, "no_allowed_match"},
		"unavailable":     {func(o *runtimeprofile.Observation) { o.Available = false }, []string{fx.SyntheticCapability}, "no_allowed_match"},
		"unknown version": {func(o *runtimeprofile.Observation) { o.Version = "" }, []string{fx.SyntheticCapability}, "effort_unproven"},
		"declared capability": {func(o *runtimeprofile.Observation) {
			o.CapabilityStatus[fx.SyntheticCapability] = runtimeprofile.CapabilityDeclared
		}, []string{fx.SyntheticCapability}, "capability_unproven"},
		"drifted executable": {func(o *runtimeprofile.Observation) { o.ExecutableDigest = strings.Repeat("9", 64) }, nil, "effort_unproven"},
	} {
		o := observedCodex(cfg, true)
		tc.change(&o)
		if _, code := consumer.ObservedChoice(fx.CodexRuntimeID, profile, o, tc.caps); code != tc.want {
			t.Errorf("%s: %s want %s", name, code, tc.want)
		}
	}
	// No implicit Runtime or Profile: another or a disabled Runtime never resolves.
	if _, code := consumer.ObservedChoice(fx.ClaudeRuntimeID, profile, observedCodex(cfg, true), nil); code != "no_allowed_match" {
		t.Fatalf("other Runtime: %s", code)
	}
	disabled := fx.Consumer(t, fx.WithRuntimeDisabled(cfg, fx.CodexRuntimeID))
	if _, code := disabled.ObservedChoice(fx.CodexRuntimeID, profile, observedCodex(cfg, true), nil); code != "no_allowed_match" {
		t.Fatalf("disabled Runtime: %s", code)
	}
}
