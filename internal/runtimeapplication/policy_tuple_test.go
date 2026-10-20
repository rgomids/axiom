package runtimeapplication

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func TestPreviewFreezesProofFreeTupleAndExactCheck(t *testing.T) {
	s, o := setup(t, "codex")
	service := New(s)
	p, err := service.Preview(context.Background(), projectID, request())
	if err != nil || p.Choice.FrozenProfile == nil || p.Choice.ResolvedProfile != nil {
		t.Fatal("preview refused", err)
	}
	r := p.Choice.FrozenProfile
	digest, _ := runtimeprofile.DigestV4(s.config)
	if r.ConfigurationDigest != p.ConfigurationDigest || r.ConfigurationDigest != digest || r.ObservationDigest != p.ObservationDigest || r.Effort.Value != "high" || r.EffortSourceDigest != strings.Repeat("b", 64) || runtimeprofile.ValidateResolvedProfileV4(*r) != nil {
		t.Fatalf("tuple did not bind exact preview: %+v", r)
	}
	binding, err := service.Check(context.Background(), p)
	if err != nil || binding.CredentialReference != "" {
		t.Fatal("check failed", err)
	}
	binding.Choice.FrozenProfile.Model = "mutated"
	if p.Choice.FrozenProfile.Model == "mutated" {
		t.Fatal("binding aliased reviewed snapshot")
	}
	o.edit("codex", func(v *runtimeprofile.OperationalObservation) { v.EffortSourceDigest = strings.Repeat("c", 64) })
	if _, err := service.Check(context.Background(), p); !errors.Is(err, ErrStale) {
		t.Fatal("source drift did not stale reviewed tuple")
	}
}

func TestAvailabilityObservesExecutableOnly(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		change     func(*observerFake)
	}{
		{"executable present", "", func(*observerFake) {}},
		{"effort source refused", "", func(o *observerFake) { o.failure = runtimeprofile.ErrEffortOverrideConflict }},
		{"executable absent", "runtime_unavailable", func(o *observerFake) { delete(o.runtimes, "codex") }},
		{"stale", "stale_observation", func(o *observerFake) {
			o.edit("codex", func(v *runtimeprofile.OperationalObservation) { v.Revision++ })
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, o := setup(t, "codex")
			tc.change(o)
			if got := New(s).Availability(context.Background(), projectID); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
