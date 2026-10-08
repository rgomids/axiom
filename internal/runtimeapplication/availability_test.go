package runtimeapplication

import (
	"context"
	"errors"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Availability is the Issue #231 readiness projection of the same
// portable/local intersection Preview uses; it never chooses a Runtime.
func TestAvailabilityProjection(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(*sourceFake, observerFake)
		want   string
	}{
		"available":      {func(*sourceFake, observerFake) {}, ""},
		"source failure": {func(s *sourceFake, _ observerFake) { s.err = errors.New("private") }, "policy_unavailable"},
		"not installed": {func(_ *sourceFake, o observerFake) {
			o["claude"] = runtimeprofile.Observation{RuntimeID: "claude", Adapter: "claude"}
		}, "runtime_unavailable"},
		"disabled":       {func(s *sourceFake, _ observerFake) { s.snapshot.Configuration.Runtimes[0].Enabled = false }, "runtime_unavailable"},
		"model mismatch": {func(s *sourceFake, _ observerFake) { s.snapshot.Configuration.ModelProfiles[0].Model = "other" }, "no_allowed_match"},
		"no observer":    {func(s *sourceFake, _ observerFake) { s.snapshot.Observer = nil }, "inventory_unavailable"},
		"unconfigured": {func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.RuntimePreferences = project.Unconfigured[[]project.RuntimePreference]()
			s.snapshot.Project, _ = project.New(state)
		}, "policy_unconfigured"},
		"v3 retains v2 semantics": {func(s *sourceFake, _ observerFake) {
			state := s.snapshot.Project.State()
			state.SchemaVersion = 3
			s.snapshot.Project, _ = project.New(state)
		}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			source, observer := setup(t, "claude")
			tc.change(source, observer)
			if got := New(source).Availability(context.Background(), projectID); got != tc.want {
				t.Fatalf("availability %q, want %q", got, tc.want)
			}
		})
	}
	if New(nil).Availability(context.Background(), projectID) != "inventory_unavailable" || New(nil).Availability(context.Background(), "../x") != "invalid_request" {
		t.Fatal("fail-closed inputs")
	}
}
