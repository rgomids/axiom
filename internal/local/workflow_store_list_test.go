package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/workflow"
)

func listFixture(t *testing.T) (WorkflowStore, string, workflow.State) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewWorkflowStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return store, root, validExecution()
}

func TestExecutionStoreListEnumeratesDeterministically(t *testing.T) {
	store, _, base := listFixture(t)
	ctx := context.Background()
	if states, err := store.List(ctx, base.ProjectID); err != nil || states == nil || len(states) != 0 {
		t.Fatalf("missing directory = %v, %v", states, err)
	}
	for index, key := range []string{"main", "api", "web"} {
		state := base
		state.RepositoryKey = key
		state.ExecutionID = "018f4a44-7c31-7dd4-9d00-11111111111" + string(rune('1'+index))
		if err := store.Create(ctx, state); err != nil {
			t.Fatal(err)
		}
	}
	states, err := store.List(ctx, base.ProjectID)
	if err != nil || len(states) != 3 {
		t.Fatalf("states = %d, %v", len(states), err)
	}
	for _, state := range states {
		if state.StorageRevision == ([32]byte{}) || state.ProjectID != base.ProjectID {
			t.Fatalf("state = %#v", state)
		}
	}
	// Another Project's records are never mixed in.
	other := base
	other.ProjectID = "223e4567-e89b-42d3-a456-426614174000"
	if err := store.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	if states, err := store.List(ctx, base.ProjectID); err != nil || len(states) != 3 {
		t.Fatalf("states = %d, %v", len(states), err)
	}
	if _, err := store.List(ctx, "not-a-project"); err == nil {
		t.Fatal("invalid project identity accepted")
	}
}

func TestExecutionStoreListFailsClosed(t *testing.T) {
	for name, damage := range map[string]func(t *testing.T, dir string, state workflow.State){
		"corrupt record": func(t *testing.T, dir string, _ workflow.State) {
			if err := os.WriteFile(filepath.Join(dir, "garbage.json"), []byte("{not-json}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"corrupt published record": func(t *testing.T, dir string, state workflow.State) {
			if err := os.WriteFile(filepath.Join(dir, executionName(state.RepositoryKey, state.WorkItem)), []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"interrupted protocol state": func(t *testing.T, dir string, _ workflow.State) {
			if err := os.WriteFile(filepath.Join(dir, ".axiom-recovery-x"), []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"foreign project record": func(t *testing.T, dir string, state workflow.State) {
			foreign := state
			foreign.ProjectID = "223e4567-e89b-42d3-a456-426614174000"
			foreign.RepositoryKey = "other"
			wire, err := encodeExecution(foreign)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, executionName(foreign.RepositoryKey, foreign.WorkItem)), wire, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"misaddressed record": func(t *testing.T, dir string, state workflow.State) {
			wire, err := encodeExecution(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "0000.json"), wire, 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			store, root, state := listFixture(t)
			if err := store.Create(context.Background(), state); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "executions", "v1", state.ProjectID)
			damage(t, dir, state)
			states, err := store.List(context.Background(), state.ProjectID)
			if err == nil || states != nil {
				t.Fatalf("damaged listing accepted: %v, %v", states, err)
			}
			if name == "interrupted protocol state" && !errors.Is(err, workflow.ErrRecoveryRequired) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
