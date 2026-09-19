package local

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func TestWorkflowStoreRoundTripsStrictPrivateState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewWorkflowStore(root)
	if err != nil {
		t.Fatal(err)
	}
	state := validWorkflow(t.TempDir())
	if err := store.Create(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
	if err != nil || loaded.RepositoryPath != state.RepositoryPath || loaded.Steps[0].Gate != "specification" {
		t.Fatalf("loaded = %#v, %v", loaded, err)
	}
	record := filepath.Join(root, "workflows", state.ProjectID, "main-7.json")
	info, err := os.Stat(record)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("record mode = %v, %v", info, err)
	}
}

func TestWorkflowStoreRejectsUnknownFieldsAndMissingSave(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewWorkflowStore(root)
	if err != nil {
		t.Fatal(err)
	}
	state := validWorkflow(t.TempDir())
	if err := store.Save(context.Background(), state); err == nil {
		t.Fatal("save created missing workflow")
	}
	if err := store.Create(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(root, "workflows", state.ProjectID, "main-7.json")
	wire, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	wire = bytes.Replace(wire, []byte(`"status":`), []byte(`"unknown":true,"status":`), 1)
	if err := os.WriteFile(record, wire, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func validWorkflow(repository string) workflow.State {
	steps := make([]workflow.Step, len(workflow.Gates))
	for index, gate := range workflow.Gates {
		steps[index] = workflow.Step{Gate: gate, Status: "pending"}
	}
	return workflow.State{ProjectID: "123e4567-e89b-42d3-a456-426614174000", RepositoryKey: "main", RepositoryPath: repository, WorkItem: 7, Status: "active", Steps: steps}
}
