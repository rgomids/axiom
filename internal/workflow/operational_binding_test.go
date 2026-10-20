package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func frozenBindingFixture(runtimeID string) runtimeprofile.ResolvedProfileV4 {
	d := strings.Repeat("b", 64)
	return runtimeprofile.ResolvedProfileV4{ProfileID: "worker", RuntimeID: runtimeID, Model: "opaque-model", Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}, ProfileDigest: d, ConfigurationRevision: 5, ConfigurationDigest: d, ExecutableDigest: d, InvocationMode: "noninteractive", ObservationRevision: 5, ObservationDigest: d, EffortSourceDigest: d}
}

func operationalState(t *testing.T) State {
	t.Helper()
	s, _, d, _ := newConfigured(t)
	d.document, _ = workflowdefinition.BuiltinRevision(2)
	target := configuredTarget()
	frozen := frozenBindingFixture("codex")
	target.FrozenProfile = &frozen
	return s.Start(context.Background(), target).State
}

func TestExecution5ProofFreeBindingCodec(t *testing.T) {
	state := operationalState(t)
	if !ValidState(state) {
		t.Fatal("proof-free Execution 5 refused")
	}
	wire, err := json.Marshal(state.Binding)
	if err != nil {
		t.Fatal(err)
	}
	for _, retired := range []string{"capabilityProofReference", "runtimeVersion", `"resolvedProfile"`} {
		if bytes.Contains(wire, []byte(retired)) {
			t.Fatalf("retired field %s", retired)
		}
	}
	view := InspectBinding(state)
	view.FrozenProfile.Model = "caller-mutation"
	if state.Binding.FrozenProfile.Model == "caller-mutation" {
		t.Fatal("binding view aliases frozen tuple")
	}
	for name, mutate := range map[string]func(*State){
		"historical generation": func(s *State) { s.FormatVersion = ProfileConfiguredFormatVersion },
		"mixed tuple": func(s *State) {
			profile := historicalProfileFixture()
			s.Binding.ResolvedProfile = &profile
		},
		"missing tuple":  func(s *State) { s.Binding.FrozenProfile = nil },
		"wrong runtime":  func(s *State) { s.Binding.FrozenProfile.RuntimeID = "claude" },
		"invalid digest": func(s *State) { s.Binding.FrozenProfile.EffortSourceDigest = "" },
		"schema":         func(s *State) { s.Binding.SchemaVersion = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			changed := operationalState(t)
			mutate(&changed)
			if ValidState(changed) {
				t.Fatal("accepted mismatched Execution 5")
			}
		})
	}
	historical := operationalState(t)
	profile := historicalProfileFixture()
	historical.FormatVersion = ProfileConfiguredFormatVersion
	historical.Binding.FrozenProfile = nil
	historical.Binding.ResolvedProfile = &profile
	if !ValidState(historical) {
		t.Fatal("historical Execution 3 no longer readable")
	}
}
