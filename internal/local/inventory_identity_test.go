package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
)

// A record is supported only at the location its store addresses it by. These
// cases keep a decodable payload and change only its location.

const foreignProjectUUID = "123e4567-e89b-42d3-a456-426614174999"

func TestInventoryRejectsRecordsAwayFromTheirCanonicalLocation(t *testing.T) {
	cases := []struct {
		name string
		kind InventoryKind
		// relocate moves or copies the single record of kind and returns the
		// state-relative path that must now fail closed.
		relocate func(t *testing.T, state, relative string) string
	}{
		{"graph renamed to another digest", InventoryGraph, renameTo(strings.Repeat("a", 64) + ".json")},
		{"graph moved under another project", InventoryGraph, moveToProject("project-2")},
		{"coordination stream renamed to another digest", InventoryCoordination, renameTo(strings.Repeat("b", 64) + ".json")},
		{"coordination stream moved under another project", InventoryCoordination, moveToProject("project-2")},
		{"create attempt renamed to another digest", InventoryCreateAttempt, renameTo(".axiom-create-" + strings.Repeat("c", 64) + ".json")},
		{"create attempt moved under another project", InventoryCreateAttempt, moveToProject(foreignProjectUUID)},
		{"create attempt stored under a work item name", InventoryCreateAttempt, renameTo("main-attempt.json")},
		{"work item stored under a create attempt name", InventoryWorkItem, renameTo(".axiom-create-" + strings.Repeat("d", 64) + ".json")},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, state := writeEveryV1Kind(t)
			relative := onlyEntryOfKind(t, state, test.kind)
			moved := test.relocate(t, state, relative)
			inventory, err := InspectStateInventory(context.Background(), state)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, entry := range inventory.Entries {
				if entry.Relative == moved {
					found = true
					if entry.Kind.Supported() {
						t.Fatalf("relocated %s classified %s, want fail closed", moved, entry.Kind)
					}
				}
			}
			if !found {
				t.Fatalf("relocated %s not inventoried: %+v", moved, inventory.Entries)
			}
		})
	}
}

// A coordination stream carries no Project, so its Project is proven through
// the graph that owns its parent and child.
func TestInventoryRejectsCoordinationWithoutItsOwningGraph(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, state, graph string)
	}{
		{"graph absent", func(t *testing.T, state, graph string) {
			if err := os.Remove(filepath.Join(state, filepath.FromSlash(graph))); err != nil {
				t.Fatal(err)
			}
		}},
		{"graph not canonical", func(t *testing.T, state, graph string) {
			renameTo(strings.Repeat("e", 64)+".json")(t, state, graph)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, state := writeEveryV1Kind(t)
			stream := onlyEntryOfKind(t, state, InventoryCoordination)
			test.mutate(t, state, onlyEntryOfKind(t, state, InventoryGraph))
			if kind := kindOf(t, state, stream); kind.Supported() {
				t.Fatalf("coordination stream %s classified %s without its owning graph", stream, kind)
			}
		})
	}
}

// Refutation pass: every record the inventory supports must load through its
// canonical store API, addressed only by the identity in its own payload.
func TestEverySupportedRecordLoadsThroughItsCanonicalStore(t *testing.T) {
	_, state := writeEveryV1Kind(t)
	inventory, err := InspectStateInventory(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	checked := map[InventoryKind]bool{}
	for _, entry := range inventory.Entries {
		parts := strings.Split(entry.Relative, "/")
		wire := func() []byte {
			data, err := os.ReadFile(filepath.Join(state, filepath.FromSlash(entry.Relative)))
			if err != nil {
				t.Fatal(err)
			}
			return data
		}
		switch entry.Kind {
		case InventoryGraph:
			decoded, err := executiongraph.DecodeGraph(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewGraphStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(ctx, graphProjectID(decoded), decoded.Parent.ExecutionID); err != nil {
				t.Errorf("supported graph %s does not load canonically: %v", entry.Relative, err)
			}
		case InventoryCoordination:
			stream, err := coordination.DecodeStream(wire())
			if err != nil {
				t.Fatal(err)
			}
			// The stream's Project is the directory the inventory bound to
			// its owning graph.
			store, err := NewCoordinationStore(state, parts[2])
			if err != nil {
				t.Fatal(err)
			}
			if _, ok, err := store.Latest(ctx, stream.ParentID, stream.ChildID); err != nil || !ok {
				t.Errorf("supported coordination stream %s does not load canonically: ok=%v err=%v", entry.Relative, ok, err)
			}
		case InventoryCreateAttempt:
			attempt, err := decodeCreateAttempt(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewWorkItemStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.LoadCreateAttempt(ctx, attempt.Target); err != nil {
				t.Errorf("supported create attempt %s does not load canonically: %v", entry.Relative, err)
			}
		case InventoryRuntimeProfile:
			store, err := NewRuntimeProfileStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(ctx); err != nil {
				t.Errorf("supported runtime profile %s does not load canonically: %v", entry.Relative, err)
			}
		default:
			continue
		}
		checked[entry.Kind] = true
	}
	for _, kind := range []InventoryKind{InventoryGraph, InventoryCoordination, InventoryCreateAttempt, InventoryRuntimeProfile} {
		if !checked[kind] {
			t.Errorf("no supported %s record was loaded through its store", kind)
		}
	}
}

func renameTo(name string) func(*testing.T, string, string) string {
	return func(t *testing.T, state, relative string) string {
		t.Helper()
		target := filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(relative)), name))
		if err := os.Rename(filepath.Join(state, filepath.FromSlash(relative)), filepath.Join(state, filepath.FromSlash(target))); err != nil {
			t.Fatal(err)
		}
		return target
	}
}

func moveToProject(projectID string) func(*testing.T, string, string) string {
	return func(t *testing.T, state, relative string) string {
		t.Helper()
		parts := strings.Split(relative, "/")
		parts[len(parts)-2] = projectID
		target := strings.Join(parts, "/")
		directory := filepath.Join(state, filepath.FromSlash(strings.Join(parts[:len(parts)-1], "/")))
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(state, filepath.FromSlash(relative)), filepath.Join(state, filepath.FromSlash(target))); err != nil {
			t.Fatal(err)
		}
		return target
	}
}

func onlyEntryOfKind(t *testing.T, state string, kind InventoryKind) string {
	t.Helper()
	inventory, err := InspectStateInventory(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	found := []string{}
	for _, entry := range inventory.Entries {
		if entry.Kind == kind {
			found = append(found, entry.Relative)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one %s entry, got %v", kind, found)
	}
	return found[0]
}

func kindOf(t *testing.T, state, relative string) InventoryKind {
	t.Helper()
	inventory, err := InspectStateInventory(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range inventory.Entries {
		if entry.Relative == relative {
			return entry.Kind
		}
	}
	t.Fatalf("%s not inventoried", relative)
	return ""
}
