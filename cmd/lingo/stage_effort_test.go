package main

import (
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/stageexecution"
	"path/filepath"
	"testing"
)

func TestProductionEffortInspectorComposition(t *testing.T) {
	base := t.TempDir()
	children := []stageexecution.ChildRights{
		{Workspace: filepath.Join(base, "worktrees", "a"), Effects: []executiongraph.Effect{{Kind: "repository-write"}}},
		{Workspace: filepath.Join(base, "worktrees", "b"), Effects: []executiongraph.Effect{{Kind: "process"}}},
	}
	i, err := productionEffortInspector(base, base, children)
	if err != nil {
		t.Fatal(err)
	}
	if !i.Consented || i.Configuration == nil || i.Repository != base || len(i.Workspaces) != 1 || i.Workspaces[0] != children[0].Workspace {
		t.Fatalf("inspector %+v", i)
	}
	if _, err := productionEffortInspector("relative", base, children); err == nil {
		t.Fatal("unsafe state root")
	}
}
