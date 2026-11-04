package local

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Retired stream bytes remain inert and untouched, even when syntactically valid.
func TestRetiredCoordinationWithoutGraphRefusesAndNeverRewrites(t *testing.T) {
	root := privateTestRoot(t)
	matches, err := filepath.Glob("../compatibility/testdata/stable-v1/snapshots/v0.8.1-issue233/state/coordination/v1/project-1/*.json")
	if err != nil || len(matches) != 1 {
		t.Fatalf("fixture: %v %v", matches, err)
	}
	wire, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "coordination", "v1", "project-1", filepath.Base(matches[0]))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	inv, err := InspectStateInventory(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Entries) != 1 {
		t.Fatalf("historical stream disappeared from inventory: %+v", inv.Entries)
	}
	for _, e := range inv.Entries {
		if e.Kind.Supported() {
			t.Fatalf("retired stream supported: %+v", e)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, wire) {
		t.Fatalf("historical bytes changed: %v", err)
	}
}
