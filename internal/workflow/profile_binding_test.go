package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func historicalProfileFixture() runtimeprofile.ResolvedProfile {
	d := strings.Repeat("a", 64)
	return runtimeprofile.ResolvedProfile{ProfileID: "worker", RuntimeID: "codex", Model: "opaque-model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "opaque-high"}, ProfileDigest: d, ConfigurationRevision: 5, ConfigurationDigest: d, ExecutableDigest: d, RuntimeVersion: "test-1", InvocationMode: "noninteractive", ObservationRevision: 7, ObservationDigest: d, EffortSourceDigest: d, CapabilityProofReference: "injected-proof"}
}

func TestHistoricalExecutionBindingRoundTripAndValidation(t *testing.T) {
	state := operationalState(t)
	for _, name := range []string{"format", "historical", "schema", "runtime", "source", "missing"} {
		t.Run(name, func(t *testing.T) {
			changed := cloneState(state)
			switch name {
			case "format":
				changed.FormatVersion = 2
				changed.WorkflowVersion = ConfiguredWorkflowVersion
			case "historical":
				changed.FormatVersion = ProfileConfiguredFormatVersion
			case "schema":
				changed.Binding.SchemaVersion = 1
			case "runtime":
				changed.Binding.FrozenProfile.RuntimeID = "claude"
			case "source":
				changed.Binding.FrozenProfile.EffortSourceDigest = ""
			case "missing":
				changed.Binding.FrozenProfile = nil
			}
			if ValidState(changed) {
				t.Fatal("invalid historical snapshot accepted")
			}
		})
	}
	doc := workflowdefinition.Builtin()
	state.FormatVersion = ConfiguredFormatVersion
	state.WorkflowVersion = ConfiguredWorkflowVersion
	state.Binding.FrozenProfile = nil
	state.Binding.SchemaVersion = 1
	state.Binding.Definition = doc.Ref("project")
	state.Binding.Snapshot = doc.Canonical
	state.Binding.Contexts = nil
	before := append([]byte(nil), state.Binding.Snapshot...)
	wire, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	if err = json.Unmarshal(wire, &restored); err != nil || !ValidState(restored) {
		t.Fatal("historical decode", err)
	}
	second, err := json.Marshal(restored)
	if err != nil || string(second) != string(wire) || string(before) != string(restored.Binding.Snapshot) {
		t.Fatal("historical bytes changed")
	}
}
