package local

import (
	"bytes"
	"context"
	"errors"
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

func TestWorkflowServiceStartReturnsExistingWorkflowWithRealStore(t *testing.T) {
	repository := t.TempDir()
	store, err := NewWorkflowStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.New(
		workflowStoreResolver{repository: repository},
		workflowStoreWorkItems{},
		store,
	)
	target := workflow.Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: 7}

	first := service.Start(context.Background(), target)
	if first.Status != workflow.Succeeded || first.Category != "workflow_started" {
		t.Fatalf("first start = %#v", first)
	}
	second := service.Start(context.Background(), target)
	if second.Status != workflow.Succeeded || second.Category != "workflow_already_started" {
		t.Fatalf("second start = %#v", second)
	}
}

func TestWorkflowStoreCreateCollisionNeverReplacesCanonicalState(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*workflow.State)
	}{
		{name: "identical", change: func(*workflow.State) {}},
		{name: "different", change: func(state *workflow.State) { state.RepositoryPath = t.TempDir() }},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, err := NewWorkflowStore(filepath.Join(t.TempDir(), "state"))
			if err != nil {
				t.Fatal(err)
			}
			state := validWorkflow(t.TempDir())
			if err := store.Create(context.Background(), state); err != nil {
				t.Fatal(err)
			}
			attempt := state
			test.change(&attempt)

			if err := store.Create(context.Background(), attempt); !errors.Is(err, ErrConflict) {
				t.Fatalf("create collision = %v", err)
			}
			loaded, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
			if err != nil || loaded.Status != state.Status || loaded.RepositoryPath != state.RepositoryPath {
				t.Fatalf("canonical state = %#v, %v", loaded, err)
			}
		})
	}
}

func TestWorkflowStoreRequiresExpectedRevision(t *testing.T) {
	store, err := NewWorkflowStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	state := validWorkflow(t.TempDir())
	if err := store.Create(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
	if err != nil || loaded.Revision == ([32]byte{}) {
		t.Fatalf("missing revision: %#v %v", loaded, err)
	}
	stale := loaded
	stale.Steps = append([]workflow.Step(nil), loaded.Steps...)
	loaded.Status = "interrupted"
	loaded.Steps[0].Status = "failed"
	loaded.Steps[0].Reference = "spec.md"
	loaded.Steps[0].Digest[0] = 1
	if err := store.Save(context.Background(), loaded); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), stale); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale save = %v", err)
	}
}

func TestWorkflowStoreFailsClosedAcrossF0F8(t *testing.T) {
	for index, stage := range []FaultStage{FaultF0, FaultF1, FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			store, err := NewWorkflowStore(filepath.Join(t.TempDir(), "state"))
			if err != nil {
				t.Fatal(err)
			}
			state := validWorkflow(t.TempDir())
			if err := store.Create(context.Background(), state); err != nil {
				t.Fatal(err)
			}
			loaded, err := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
			if err != nil {
				t.Fatal(err)
			}
			loaded.Status = "interrupted"
			loaded.Steps[0].Status = "failed"
			loaded.Steps[0].Reference = "spec.md"
			loaded.Steps[0].Digest[0] = 1
			store.hooks.fault = func(current FaultStage) error {
				if current == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			err = store.Save(context.Background(), loaded)
			var publication *PublicationError
			if !errors.As(err, &publication) || publication.Committed != (index >= 6) {
				t.Fatalf("fault outcome: %v", err)
			}
			_, readErr := store.Load(context.Background(), state.ProjectID, state.RepositoryKey, state.WorkItem)
			if stage == FaultF0 || stage == FaultF1 {
				if readErr != nil {
					t.Fatalf("%s lost prior authority: %v", stage, readErr)
				}
				return
			}
			if !errors.Is(readErr, ErrRecoveryRequired) {
				t.Fatalf("stage %s reader = %v", stage, readErr)
			}
		})
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

type workflowStoreResolver struct{ repository string }

func (r workflowStoreResolver) Resolve(context.Context, string) (workflow.Project, string) {
	return workflow.Project{
		ID: "123e4567-e89b-42d3-a456-426614174000",
		Repositories: []workflow.Repository{{
			Key:  "main",
			Path: r.repository,
		}},
	}, ""
}

type workflowStoreWorkItems struct{}

func (workflowStoreWorkItems) Available(context.Context, string, string, int) bool { return true }

func (workflowStoreWorkItems) Complete(context.Context, string, string, int, bool) workflow.WorkItemCompletion {
	return workflow.WorkItemCompletion{}
}
