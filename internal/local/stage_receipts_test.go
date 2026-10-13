package local

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
)

func localDispatchInputs(g executiongraph.SuccessorGraph, executionID, stageID string) executiongraph.DispatchInputs {
	preview, _ := json.Marshal(map[string]any{"stagePlan": map[string]any{"executionRef": map[string]any{"executionId": executionID, "revision": g.Parent.GraphRevision}, "stageId": stageID}, "digest": "synthetic"})
	return executiongraph.DispatchInputs{FormatVersion: 1, ProjectID: g.Children[0].Envelope.Scope.ProjectID, ParentID: g.Parent.ExecutionID, PreviewDigest: g.DispatchAuthorityRef, Preview: preview}
}

func TestDispatchInputsLocateCommittedStageGraph(t *testing.T) {
	ctx := context.Background()
	root, graphs, _, g := committedLocalDispatch(t)
	projectID := g.Children[0].Envelope.Scope.ProjectID
	if _, err := graphs.FindStageGraph(ctx, projectID, "00000000-0000-4000-8000-0000000005e1", "implementation"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("graph found before its inputs record: %v", err)
	}
	inputs := localDispatchInputs(g, "00000000-0000-4000-8000-0000000005e1", "implementation")
	ref, err := graphs.CommitDispatchInputs(ctx, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := graphs.CommitDispatchInputs(ctx, inputs); err != nil || again != ref {
		t.Fatalf("repeat = %s %v", again, err)
	}
	parent, err := graphs.FindStageGraph(ctx, projectID, "00000000-0000-4000-8000-0000000005e1", "implementation")
	if err != nil || parent != g.Parent.ExecutionID {
		t.Fatalf("parent = %s %v", parent, err)
	}
	if _, err := graphs.FindStageGraph(ctx, projectID, "00000000-0000-4000-8000-0000000005e1", "review"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other stage resolved: %v", err)
	}
	loaded, err := graphs.LoadDispatchInputs(ctx, projectID, g.Parent.ExecutionID, inputs.PreviewDigest)
	if err != nil || string(loaded.Preview) != string(inputs.Preview) {
		t.Fatalf("inputs readback = %v", err)
	}
	// An orphan inputs record whose graph was never committed is inert.
	orphan := inputs
	orphan.ParentID = "00000000-0000-4000-8000-0000000000ee"
	if _, err := graphs.CommitDispatchInputs(ctx, orphan); err != nil {
		t.Fatal(err)
	}
	if parent, err := graphs.FindStageGraph(ctx, projectID, "00000000-0000-4000-8000-0000000005e1", "implementation"); err != nil || parent != g.Parent.ExecutionID {
		t.Fatalf("orphan changed resolution: %s %v", parent, err)
	}
	inventory, err := InspectStateInventory(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, entry := range inventory.Entries {
		if entry.Kind == InventoryDispatchInputs {
			found++
		}
		if !entry.Kind.Supported() {
			t.Errorf("%s classified %s", entry.Relative, entry.Kind)
		}
	}
	if found != 2 {
		t.Fatalf("dispatch inputs inventoried %d times", found)
	}
	if kind := classifyDispatchInputs(projectID, "0000000000000000000000000000000000000000000000000000000000000000.json", []byte(`{"formatVersion":1}`)); kind != "" {
		t.Fatalf("non-inputs bytes classified %s", kind)
	}
}
