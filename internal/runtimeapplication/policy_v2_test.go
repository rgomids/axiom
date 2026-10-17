package runtimeapplication

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

// setupV2 is the single-Runtime fixture whose injected proof binds the exact
// v3 Profile intent; the tests keep their historical names.
func setupV2(t *testing.T) (*sourceFake, observerFake) {
	t.Helper()
	return setup(t, "codex")
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
	if err != nil || binding.CredentialReference != "" || r.ConfigurationDigest != runtimeprofiletest.Digest(t, s.config) {
		t.Fatal("v3 check failed", err)
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
