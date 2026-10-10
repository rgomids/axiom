package local

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/provenance"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func TestConfiguredOutputValidatorRequiresRegisteredPolicyAndCorrelation(t *testing.T) {
	ctx := context.Background()
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	state := validConfiguredExecution()
	state.ExecutionID = "123e4567-e89b-42d3-a456-426614174000"
	state.Binding.SnapshotRef = "execution:" + state.ExecutionID + "#/binding/snapshot"
	stage := *workflow.CurrentStage(state)
	output := stage.Outputs[0]
	source, _ := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
	artifact, err := store.Create(ctx, detailartifact.Draft{ExecutionID: state.ExecutionID, Category: output.Kind, Outcome: "success", Retention: detailartifact.Evidence, Markdown: []byte("# Output\n"), Provenance: source, References: []detailartifact.Reference{{Kind: "workflow-stage", Value: stage.ID}, {Kind: "workflow-definition", Value: state.Binding.Definition.Digest}, {Kind: "workflow-output", Value: output.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	ref := workflow.Reference{Kind: "artifact", ID: artifact.ID, Digest: hex.EncodeToString(artifact.Digest[:])}
	policy := workflowdefinition.Validator{Kind: "artifact-schema", PolicyRef: "builtin-sdd-v1"}
	validator := NewWorkflowReferenceValidator(store)
	if err := validator.ValidateStageOutput(ctx, state, t.TempDir(), stage, output, policy, ref); err != nil {
		t.Fatal(err)
	}
	unknown := policy
	unknown.PolicyRef = "uninstalled-policy"
	if validator.ValidateStageOutput(ctx, state, t.TempDir(), stage, output, unknown, ref) == nil {
		t.Fatal("unregistered policy accepted")
	}
	foreignStage := stage
	foreignStage.ID = "foreign-stage"
	if validator.ValidateStageOutput(ctx, state, t.TempDir(), foreignStage, output, policy, ref) == nil {
		t.Fatal("foreign stage accepted")
	}
	foreignOutput := output
	foreignOutput.ID = "foreign-output"
	if validator.ValidateStageOutput(ctx, state, t.TempDir(), stage, foreignOutput, policy, ref) == nil {
		t.Fatal("foreign output accepted")
	}
	wrongCategory := output
	wrongCategory.Kind = "decision"
	if output.Kind == "decision" {
		wrongCategory.Kind = "evidence"
	}
	if validator.ValidateStageOutput(ctx, state, t.TempDir(), stage, wrongCategory, policy, ref) == nil {
		t.Fatal("wrong typed category accepted")
	}
}

func validConfiguredExecution() workflow.State {
	state := validExecution()
	doc := workflowdefinition.Builtin()
	state.FormatVersion = 2
	state.WorkflowVersion = workflow.ConfiguredWorkflowVersion
	state.Binding = &workflow.WorkflowBinding{ProjectID: state.ProjectID, Definition: doc.Ref("builtin"), SchemaVersion: 1, SnapshotRef: "execution:" + state.ExecutionID + "#/binding/snapshot", Snapshot: doc.Canonical, ObservationDigest: strings.Repeat("a", 64), RuntimePreview: strings.Repeat("b", 64), BoundAt: state.CreatedAt, Contexts: map[string]workflow.ContextObservation{}}
	state.Binding.ProjectRevision, state.Binding.LocalRevision = strings.Repeat("c", 64), strings.Repeat("d", 64)
	return state
}

func TestConfiguredExecutionPersistedCorpus(t *testing.T) {
	wire, err := os.ReadFile(filepath.Join("testdata", "configured-v2", "initial.json"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := decodeExecution(wire)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeExecution(state)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("configured corpus changed: %v", err)
	}
	// Legacy format 1 stays a separate codec without invented bindings.
	legacy := validExecution()
	old, err := encodeExecution(legacy)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := decodeExecution(old)
	if err != nil || loaded.Binding != nil || loaded.FormatVersion != 1 {
		t.Fatal("legacy format changed")
	}
}

func TestConfiguredStoreRoundTripCorruptionAndImmutableBinding(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewWorkflowStore(root)
	if err != nil {
		t.Fatal(err)
	}
	state := validConfiguredExecution()
	ctx := context.Background()
	if err = store.Create(ctx, state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(ctx, state.ProjectID, state.RepositoryKey, state.WorkItem)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Binding.Definition != state.Binding.Definition || string(loaded.Binding.Snapshot) != string(state.Binding.Snapshot) {
		t.Fatal("binding changed")
	}
	inventory, err := InspectStateInventory(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	supported := false
	for _, entry := range inventory.Entries {
		if entry.Kind == InventoryExecution {
			supported = true
		}
	}
	if !supported {
		t.Fatalf("configured writer not inventoried: %+v", inventory)
	}
	r2 := workflowdefinition.Builtin().Definition
	r2.Revision = 2
	doc, issues := workflowdefinition.Encode(r2)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	loaded.Binding.Definition = doc.Ref("builtin")
	loaded.Binding.Snapshot = doc.Canonical
	if err = store.Save(ctx, loaded); !errors.Is(err, workflow.ErrRecoveryRequired) {
		t.Fatalf("binding was reassigned: %v", err)
	}
	path := filepath.Join(root, "executions", "v1", state.ProjectID, executionName(state.RepositoryKey, state.WorkItem))
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func([]byte) []byte{
		"missing": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"snapshot":{`), []byte(`"snapshot":null,"unknown":{`), 1)
		},
		"digest": func(b []byte) []byte {
			return bytes.Replace(b, []byte(state.Binding.Definition.Digest), bytes.Repeat([]byte{'0'}, 64), 1)
		},
		"duplicate": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"formatVersion":2`), []byte(`"formatVersion":2,"formatVersion":2`), 1)
		},
		"future": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"formatVersion":2`), []byte(`"formatVersion":3`), 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			corrupt := mutate(wire)
			if err := os.WriteFile(path, corrupt, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(ctx, state.ProjectID, state.RepositoryKey, state.WorkItem); !errors.Is(err, workflow.ErrRecoveryRequired) {
				t.Fatalf("corruption not refused: %v", err)
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(after, corrupt) {
				t.Fatal("corruption repaired implicitly")
			}
		})
	}
}

func TestConfiguredStoreConcurrentStartConverges(t *testing.T) {
	store, err := NewWorkflowStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	state := validConfiguredExecution()
	ctx := context.Background()
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- store.Create(ctx, state) }()
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, workflow.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestConfiguredPublicationFaultsNeverExposePartialBinding(t *testing.T) {
	for _, stage := range []FaultStage{FaultF0, FaultF1, FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			store, err := NewWorkflowStore(filepath.Join(t.TempDir(), "state"))
			if err != nil {
				t.Fatal(err)
			}
			store.hooks.fault = func(at FaultStage) error {
				if at == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			state := validConfiguredExecution()
			_ = store.Create(context.Background(), state)
			loaded, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
			if err == nil && (loaded.Binding == nil || !workflow.ValidState(loaded) || loaded.Binding.Definition != state.Binding.Definition) {
				t.Fatal("partial binding admitted")
			}
		})
	}
}
