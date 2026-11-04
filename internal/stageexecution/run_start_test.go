//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

const startExecutionID = "00000000-0000-4000-8000-0000000000f1"

func startFixture(t *testing.T, output string) (*engineFixture, local.ExecutionStore) {
	t.Helper()
	// Reuse the engine fixture's wiring, but start from an empty store.
	f := newEngineFixture(t, twoStages(false), succeed(output))
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := local.NewExecutionStore(root)
	if err != nil {
		t.Fatal(err)
	}
	f.engine.Store, f.root = store, root
	return f, store
}

func linkedRequest() StartRequest {
	return StartRequest{
		ExecutionID: startExecutionID, ProjectID: "proj",
		Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/app"}},
		WorkItem:     LinkedWorkItem{Provider: "linear", ID: "AXM-42", RepositoryKey: "app", RepositoryIdentity: "github.com/example/app"},
	}
}

func TestStartRunsMinimalDefaultWorkflowInOneAction(t *testing.T) {
	f, store := startFixture(t, "OUTPUT result\n")
	x, err := f.engine.Start(context.Background(), store, linkedRequest())
	if err != nil {
		t.Fatal(err)
	}
	if x.Status != workflow.RunCompleted || x.Workflow.ID != workflow.DefaultWorkflowID || len(x.Stages) != 1 || f.runner.spawns.Load() != 1 {
		t.Fatalf("default run: %s %s %d", x.Status, x.Workflow.ID, f.runner.spawns.Load())
	}
	entries, _ := os.ReadDir(f.root)
	for _, entry := range entries {
		if name := entry.Name(); name != "executions" && name != "run-leases" {
			t.Fatalf("one run action created %q (Plan/Policy/Graph/Single/Provider state)", name)
		}
	}
}

func TestStartRefusesMismatchedWorkItemLinkBeforeAnyState(t *testing.T) {
	f, store := startFixture(t, "OUTPUT result\n")
	r := linkedRequest()
	r.WorkItem.RepositoryIdentity = "github.com/example/other"
	if _, err := f.engine.Start(context.Background(), store, r); !errors.Is(err, ErrWorkItemLinkMismatch) {
		t.Fatalf("mismatched link: %v", err)
	}
	r = linkedRequest()
	r.WorkItem.RepositoryKey = "unbound"
	if _, err := f.engine.Start(context.Background(), store, r); !errors.Is(err, ErrWorkItemLinkMismatch) {
		t.Fatalf("unbound repository key: %v", err)
	}
	if entries, _ := os.ReadDir(f.root); len(entries) != 0 || f.runner.spawns.Load() != 0 {
		t.Fatal("refused start left state or spawned")
	}
}

func TestStartProgressesTwoSelectedStagesAutomatically(t *testing.T) {
	f, store := startFixture(t, "OUTPUT result\n")
	r := linkedRequest()
	r.WorkflowID, r.WorkflowRevision = "custom", "r7"
	r.Stages = []workflow.StageSpec{
		{ID: "build", Agents: []workflow.AgentSpec{{ID: "builder"}}, Outputs: []string{"result"}, TimeoutSeconds: 5},
		{ID: "check", DependsOn: []string{"build"}, Agents: []workflow.AgentSpec{{ID: "checker"}}, Outputs: []string{"result"}, TimeoutSeconds: 5},
	}
	x, err := f.engine.Start(context.Background(), store, r)
	if err != nil || x.Status != workflow.RunCompleted || f.runner.spawns.Load() != 2 {
		t.Fatalf("two stages: %v %s %d", err, x.Status, f.runner.spawns.Load())
	}
	if _, err := f.engine.Start(context.Background(), store, r); !errors.Is(err, local.ErrConflict) {
		t.Fatalf("second start reused the Execution identity: %v", err)
	}
}
