package runtimeapplication

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func setupV2(t *testing.T) (*sourceFake, observerFake) {
	t.Helper()
	s, o := setup(t, "codex")
	s.snapshot.Configuration.FormatVersion = runtimeprofile.FormatVersionV2
	s.snapshot.Configuration.Runtimes[0].CredentialReference = ""
	s.snapshot.Configuration.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: "explicit", Value: "high"}
	obs := o["codex"]
	obs.ExecutableDigest = strings.Repeat("a", 64)
	p := s.snapshot.Configuration.ModelProfiles[0]
	obs.ProfileProofs = []runtimeprofile.ProfileProof{{RuntimeID: p.RuntimeID, Model: p.Model, Effort: p.Effort, ExecutableDigest: obs.ExecutableDigest, RuntimeVersion: obs.Version, InvocationMode: "noninteractive", ObservationRevision: obs.Revision, Reference: "injected-proof", EffortSourceDigest: strings.Repeat("b", 64), Authentication: runtimeprofile.CapabilityProven, ModelAvailability: runtimeprofile.CapabilityProven, EffortSupport: runtimeprofile.CapabilityProven, NoOverride: runtimeprofile.CapabilityProven, CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"go": runtimeprofile.CapabilityProven}}}
	o["codex"] = obs
	return s, o
}
func TestV2PreviewFullSnapshotAndExactCheck(t *testing.T) {
	s, o := setupV2(t)
	service := New(s)
	p, err := service.Preview(context.Background(), projectID, request())
	if err != nil || p.Choice.ResolvedProfile == nil {
		t.Fatal("injected preview refused", err)
	}
	r := p.Choice.ResolvedProfile
	if r.ConfigurationDigest != p.ConfigurationDigest || r.ObservationDigest != p.ObservationDigest || r.Effort.Value != "high" {
		t.Fatal("tuple did not bind exact preview")
	}
	binding, err := service.Check(context.Background(), p)
	if err != nil || binding.CredentialReference != "" {
		t.Fatal("v2 check failed", err)
	}
	binding.Choice.ResolvedProfile.Model = "mutated"
	if p.Choice.ResolvedProfile.Model == "mutated" {
		t.Fatal("binding aliased reviewed snapshot")
	}
	obs := o["codex"]
	obs.ProfileProofs[0].EffortSourceDigest = strings.Repeat("c", 64)
	o["codex"] = obs
	if _, err := service.Check(context.Background(), p); !errors.Is(err, ErrStale) {
		t.Fatal("source drift did not stale reviewed tuple")
	}
}
func TestV2AvailabilitySeparatesUnknownFacts(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		change     func(*runtimeprofile.Observation)
	}{{"proven", "", func(*runtimeprofile.Observation) {}}, {"no proof", "authentication_unproven", func(o *runtimeprofile.Observation) { o.ProfileProofs = nil }}, {"auth unknown", "authentication_unproven", func(o *runtimeprofile.Observation) {
		o.ProfileProofs[0].Authentication = runtimeprofile.CapabilityDeclared
	}}, {"model unknown", "model_unproven", func(o *runtimeprofile.Observation) {
		o.ProfileProofs[0].ModelAvailability = runtimeprofile.CapabilityDeclared
	}}, {"effort unknown", "effort_unproven", func(o *runtimeprofile.Observation) {
		o.ProfileProofs[0].EffortSupport = runtimeprofile.CapabilityDeclared
	}}, {"sources unknown", "effort_configuration_unproven", func(o *runtimeprofile.Observation) { o.ProfileProofs[0].NoOverride = runtimeprofile.CapabilityDeclared }}, {"stale", "stale_observation", func(o *runtimeprofile.Observation) { o.Revision++ }}} {
		t.Run(tc.name, func(t *testing.T) {
			s, o := setupV2(t)
			obs := o["codex"]
			tc.change(&obs)
			o["codex"] = obs
			if got := New(s).Availability(context.Background(), projectID); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
