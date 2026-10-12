package workflow

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func profileBindingFixture() runtimeprofile.ResolvedProfile {
	d := strings.Repeat("a", 64)
	return runtimeprofile.ResolvedProfile{ProfileID: "worker", RuntimeID: "codex", Model: "opaque-model", Effort: runtimeprofile.Effort{Mode: "explicit", Value: "opaque-high"}, ProfileDigest: d, ConfigurationRevision: 5, ConfigurationDigest: d, ExecutableDigest: d, RuntimeVersion: "test-1", InvocationMode: "noninteractive", ObservationRevision: 7, ObservationDigest: d, EffortSourceDigest: d, CapabilityProofReference: "injected-proof"}
}
func TestExecution3FreezesWorkflowAndProfileAcrossRestart(t *testing.T) {
	s, store, definitions, _ := newConfigured(t)
	definitions.document, _ = workflowdefinition.BuiltinRevision(2)
	target := configuredTarget()
	profile := profileBindingFixture()
	target.ResolvedProfile = &profile
	started := s.Start(context.Background(), target)
	if started.Status != Succeeded || started.State.FormatVersion != 3 || started.State.WorkflowVersion != "configured-sequential-v2" || started.State.Binding.SchemaVersion != 2 || started.State.Binding.ResolvedProfile == nil {
		t.Fatalf("start3: %+v", started)
	}
	frozen := *started.State.Binding.ResolvedProfile
	originalWorkflow := append([]byte(nil), started.State.Binding.Snapshot...)
	// Local edits and changed portable selection cannot replace the existing binding.
	profile.Effort.Value = "opaque-low"
	profile.ConfigurationRevision++
	next := definitions.document.Definition
	next.Revision = 3
	next.Stages[0].Instructions = "new selection"
	doc, issues := workflowdefinition.Encode(next)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	definitions.document = doc
	wire, err := json.Marshal(store.state)
	if err != nil {
		t.Fatal(err)
	}
	var restartedState State
	if err = json.Unmarshal(wire, &restartedState); err != nil || !ValidState(restartedState) {
		t.Fatal("restart decode", err)
	}
	if !reflect.DeepEqual(frozen, *restartedState.Binding.ResolvedProfile) || string(originalWorkflow) != string(restartedState.Binding.Snapshot) {
		t.Fatal("restart rebound from mutable state")
	}
	restart := s
	restart.store = &s4Store{state: restartedState, created: true}
	calls := definitions.calls
	definitions.category = "workflow_selection_required"
	inspected := restart.Status(context.Background(), target)
	if inspected.Status != Succeeded || definitions.calls != calls || !reflect.DeepEqual(frozen, *inspected.State.Binding.ResolvedProfile) || CurrentStage(inspected.State).Instructions == "new selection" {
		t.Fatalf("historical inspection changed binding: %+v", inspected)
	}
	// A returned read projection must not expose the store's mutable snapshot.
	inspected.State.Binding.ResolvedProfile.Effort.Value = "caller-mutation"
	loaded, err := restart.store.Load(context.Background(), restartedState.ProjectID, restartedState.RepositoryKey, restartedState.WorkItem)
	if err != nil || loaded.Binding.ResolvedProfile.Effort.Value != frozen.Effort.Value {
		t.Fatal("read result aliases frozen store tuple")
	}
}
func TestExecution3RejectsMissingUnknownAndMixedSnapshot(t *testing.T) {
	for _, name := range []string{"missing", "unknown-source", "wrong-runtime"} {
		t.Run(name, func(t *testing.T) {
			s, store, d, _ := newConfigured(t)
			d.document, _ = workflowdefinition.BuiltinRevision(2)
			target := configuredTarget()
			profile := profileBindingFixture()
			target.ResolvedProfile = &profile
			switch name {
			case "missing":
				target.ResolvedProfile = nil
			case "unknown-source":
				profile.EffortSourceDigest = ""
			case "wrong-runtime":
				profile.RuntimeID = "claude"
			}
			out := s.Start(context.Background(), target)
			if out.Status == Succeeded || store.created {
				t.Fatalf("accepted incomplete snapshot %+v", out)
			}
		})
	}
	s, _, d, _ := newConfigured(t)
	d.document, _ = workflowdefinition.BuiltinRevision(2)
	target := configuredTarget()
	profile := profileBindingFixture()
	target.ResolvedProfile = &profile
	state := s.Start(context.Background(), target).State
	for _, name := range []string{"format", "schema", "runtime", "proof"} {
		t.Run(name, func(t *testing.T) {
			wire, _ := json.Marshal(state)
			var changed State
			_ = json.Unmarshal(wire, &changed)
			switch name {
			case "format":
				changed.FormatVersion = 2
				changed.WorkflowVersion = ConfiguredWorkflowVersion
			case "schema":
				changed.Binding.SchemaVersion = 1
			case "runtime":
				changed.Binding.ResolvedProfile.RuntimeID = "claude"
			case "proof":
				changed.Binding.ResolvedProfile.CapabilityProofReference = ""
			}
			if ValidState(changed) {
				t.Fatal("accepted mismatched versioned snapshot")
			}
		})
	}
}
func TestExecution2HistoricalRoundTripKeepsBindingBytes(t *testing.T) {
	s, _, _, _ := newConfigured(t)
	state := s.Start(context.Background(), configuredTarget()).State
	before := append([]byte(nil), state.Binding.Snapshot...)
	wire, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	if err = json.Unmarshal(wire, &restored); err != nil || !ValidState(restored) {
		t.Fatal("legacy decode", err)
	}
	second, err := json.Marshal(restored)
	if err != nil || string(second) != string(wire) || string(restored.Binding.Snapshot) != string(before) || restored.Binding.ResolvedProfile != nil {
		t.Fatal("legacy bytes or ownership changed")
	}
}
