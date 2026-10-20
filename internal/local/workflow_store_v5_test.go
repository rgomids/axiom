package local

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
)

func TestWorkflowStoreExecution5RoundTripAndHistoricalIsolation(t *testing.T) {
	historical, err := os.ReadFile("../workflowcompiler/testdata/issue306/execution-v3.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := decodeExecution(historical)
	if err != nil || state.FormatVersion != workflow.ProfileConfiguredFormatVersion {
		t.Fatal("historical Execution 3 unreadable", err)
	}
	if encoded, err := encodeExecutionSnapshot(state); err != nil || !bytes.Equal(encoded, historical) {
		t.Fatal("historical Execution 3 bytes changed", err)
	}
	d := strings.Repeat("c", 64)
	frozen := runtimeprofile.ResolvedProfileV4{ProfileID: state.Binding.ResolvedProfile.ProfileID, RuntimeID: state.RuntimeID, Model: state.Binding.ResolvedProfile.Model, Effort: state.Binding.ResolvedProfile.Effort, ProfileDigest: d, ConfigurationRevision: 9, ConfigurationDigest: d, ExecutableDigest: d, InvocationMode: "noninteractive", ObservationRevision: 9, ObservationDigest: d, EffortSourceDigest: d}
	state.FormatVersion = workflow.OperationalConfiguredFormatVersion
	state.Binding.ResolvedProfile = nil
	state.Binding.FrozenProfile = &frozen
	state.StorageRevision = [32]byte{}
	wire, err := encodeExecutionSnapshot(state)
	if err != nil {
		t.Fatal("Execution 5 refused", err)
	}
	for _, retired := range []string{"capabilityProofReference", "runtimeVersion"} {
		if bytes.Contains(wire, []byte(retired)) {
			t.Fatalf("retired field %s", retired)
		}
	}
	store, _ := NewWorkflowStore(privateTestRoot(t))
	if err := store.Create(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
	if err != nil || loaded.FormatVersion != workflow.OperationalConfiguredFormatVersion || loaded.Binding.FrozenProfile == nil || *loaded.Binding.FrozenProfile != frozen {
		t.Fatal("Execution 5 not preserved", err)
	}
	downgraded := bytes.Replace(wire, []byte(`"formatVersion":5`), []byte(`"formatVersion":3`), 1)
	if _, err := decodeExecution(downgraded); err == nil {
		t.Fatal("proof-free tuple accepted as Execution 3")
	}
	proof := bytes.Replace(wire, []byte(`"frozenProfile":{`), []byte(`"frozenProfile":{"capabilityProofReference":"retired",`), 1)
	if _, err := decodeExecution(proof); err == nil {
		t.Fatal("retired proof field accepted in Execution 5")
	}
}
