package local

import (
	"context"
	"github.com/rgomids/axiom/internal/workflow"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/validatorpolicy"
)

// A record is supported only at the location its store addresses it by. These
// cases keep a decodable payload and change only its location.

const foreignProjectUUID = "123e4567-e89b-42d3-a456-426614174999"

func TestDispatchAuthorityInventoryBinding(t *testing.T) {
	g, a := localDispatch(t)
	wire, err := executiongraph.EncodeDispatchAuthority(a)
	if err != nil {
		t.Fatal(err)
	}
	name := g.DispatchAuthorityRef + ".json"
	if kind := classifyDispatchAuthority(a.ProjectID, name, wire, a.Scope); kind != InventoryDispatchAuthority {
		t.Fatalf("canonical authority classified %s", kind)
	}
	for _, test := range []struct {
		project, name, scope string
	}{
		{"project-2", name, a.Scope},
		{a.ProjectID, strings.Repeat("f", 64) + ".json", a.Scope},
		{a.ProjectID, name, "single-first-attempt"},
	} {
		if kind := classifyDispatchAuthority(test.project, test.name, wire, test.scope); kind.Supported() {
			t.Fatalf("foreign authority location classified %s", kind)
		}
	}
	if kind := classifyDispatchAuthority(a.ProjectID, name, append([]byte(" "), wire...), a.Scope); kind.Supported() {
		t.Fatalf("changed exact wire classified %s", kind)
	}
}

func TestInventoryRejectsRecordsAwayFromTheirCanonicalLocation(t *testing.T) {
	cases := []struct {
		name string
		kind InventoryKind
		// relocate moves or copies the single record of kind and returns the
		// state-relative path that must now fail closed.
		relocate func(t *testing.T, state, relative string) string
	}{
		{"decision renamed", InventoryDeliveryDecision, renameTo(strings.Repeat("f", 64) + ".json")},
		{"rework renamed", InventoryDeliveryRework, renameTo(strings.Repeat("f", 64) + ".json")},
		{"retained renamed", InventoryRetainedExecution, renameTo(strings.Repeat("f", 64) + ".json")},
		{"authority renamed to another digest", InventoryDispatchAuthority, renameTo(strings.Repeat("f", 64) + ".json")},
		{"authority moved under another project", InventoryDispatchAuthority, moveToProject("project-2")},
		{"graph authority moved to executions", InventoryDispatchAuthority, moveAuthorityFamily},
		{"lease epoch renamed", InventoryRunLease, renameTo("foreign.json")},
		{"lease moved to another graph", InventoryRunLease, moveLeaseGraph},
		{"policy renamed to another digest", InventoryValidatorPolicy, renameTo(strings.Repeat("f", 64) + ".json")},
		{"graph renamed to another digest", InventoryGraph, renameTo(strings.Repeat("a", 64) + ".json")},
		{"graph moved under another project", InventoryGraph, moveToProject("project-2")},
		{"coordination stream renamed to another digest", InventoryCoordination, renameTo(strings.Repeat("b", 64) + ".json")},
		{"coordination stream moved under another project", InventoryCoordination, moveToProject("project-2")},
		{"create attempt renamed to another digest", InventoryCreateAttempt, renameTo(".axiom-create-" + strings.Repeat("c", 64) + ".json")},
		{"create attempt moved under another project", InventoryCreateAttempt, moveToProject(foreignProjectUUID)},
		{"create attempt stored under a work item name", InventoryCreateAttempt, renameTo("main-attempt.json")},
		{"work item stored under a create attempt name", InventoryWorkItem, renameTo(".axiom-create-" + strings.Repeat("d", 64) + ".json")},
		{"operational state moved under another project", InventoryOperational, moveToProject(foreignProjectUUID)},
		{"single-stage ledger renamed to another digest", InventorySingleStageRun, renameTo(strings.Repeat("e", 64) + ".json")},
		{"single-stage ledger moved under another project", InventorySingleStageRun, moveToProject("project-2")},
		{"single-stage authority renamed to another digest", InventorySingleStageAuthority, renameTo(strings.Repeat("e", 64) + ".json")},
		{"single-stage authority moved under another project", InventorySingleStageAuthority, moveToProject("project-2")},
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
		case InventoryDeliveryDecision:
			d, err := workflow.DecodeDeliveryDecision(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, _ := NewWorkflowStore(state)
			if _, err = store.LoadDeliveryDecision(ctx, parts[2], d.Packet.ExecutionID); err != nil {
				t.Fatal(err)
			}
		case InventoryDeliveryRework:
			p, err := workflow.DecodeReworkPreview(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, _ := NewWorkflowStore(state)
			if _, err = store.LoadDeliveryRework(ctx, parts[2], p.OriginalPacket.ExecutionID); err != nil {
				t.Fatal(err)
			}
		case InventoryRetainedExecution:
			execution, err := decodeExecution(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, _ := NewWorkflowStore(state)
			if _, err = store.LoadExecutionID(ctx, parts[2], execution.RepositoryKey, execution.WorkItem, execution.ExecutionID); err != nil {
				t.Fatal(err)
			}
		case InventoryDeliveryPublication:
			r, err := workflow.DecodeDeliveryPublication(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, _ := NewWorkflowStore(state)
			if _, err = store.LoadDeliveryPublication(ctx, parts[2], r.Preview.Digest()); err != nil {
				t.Fatal(err)
			}
		case InventoryDeliveryPacket:
			store, _ := NewWorkflowStore(state)
			if _, err := store.LoadDeliveryPacket(ctx, parts[2], strings.TrimSuffix(parts[3], ".json")); err != nil {
				t.Fatal(err)
			}
		case InventoryDispatchAuthority:
			a, err := executiongraph.DecodeDispatchAuthority(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewGraphStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.LoadDispatchAuthority(ctx, a.ProjectID, a.ParentID, a.PreviewDigest); err != nil {
				t.Errorf("supported authority %s does not load canonically: %v", entry.Relative, err)
			}
		case InventoryRunLease:
			record, err := decodeRunLease(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewRunLeaseStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if held, err := store.Probe(ctx, record.ProjectID, record.ParentID); err != nil || held {
				t.Errorf("supported lease %s does not probe canonically: held=%v err=%v", entry.Relative, held, err)
			}
		case InventoryValidatorPolicy:
			p, err := validatorpolicy.Decode(wire())
			if err != nil {
				t.Fatal(err)
			}
			id, err := validatorpolicy.Digest(p)
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewValidatorPolicyStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(ctx, id); err != nil {
				t.Errorf("supported policy %s does not load: %v", entry.Relative, err)
			}
		case InventoryGraph:
			decoded, err := executiongraph.DecodeGraphDocument(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewGraphStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Successor != nil {
				_, err = store.LoadSuccessor(ctx, decoded.Successor.Children[0].Envelope.Scope.ProjectID, decoded.Successor.Parent.ExecutionID)
			} else {
				_, err = store.Load(ctx, graphProjectID(*decoded.Historical), decoded.Historical.Parent.ExecutionID)
			}
			if err != nil {
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
			if snapshot, err := store.Inspect(ctx); err != nil || snapshot.State != "active-v3" && snapshot.State != "retired" {
				t.Errorf("supported runtime profile %s does not load canonically: %s %v", entry.Relative, snapshot.State, err)
			}
		case InventoryOperational:
			projectID, decoded, err := DecodeOperational(wire())
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewOperationalStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if observation, err := store.InspectOperational(ctx, projectID); err != nil || !observation.State.Equal(decoded) {
				t.Errorf("supported operational state %s does not load canonically: %+v %v", entry.Relative, observation, err)
			}
		default:
			continue
		}
		checked[entry.Kind] = true
	}
	for _, kind := range []InventoryKind{InventoryDispatchAuthority, InventoryRunLease, InventoryValidatorPolicy, InventoryGraph, InventoryCoordination, InventoryCreateAttempt, InventoryRuntimeProfile, InventoryOperational} {
		if !checked[kind] {
			t.Errorf("no supported %s record was loaded through its store", kind)
		}
	}
}

func moveAuthorityFamily(t *testing.T, state, relative string) string {
	t.Helper()
	parts := strings.Split(relative, "/")
	parts[0] = "executions"
	target := strings.Join(parts, "/")
	if err := os.MkdirAll(filepath.Join(state, filepath.FromSlash(filepath.Dir(target))), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(state, filepath.FromSlash(relative)), filepath.Join(state, filepath.FromSlash(target))); err != nil {
		t.Fatal(err)
	}
	return target
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
	if len(found) == 0 || len(found) > 1 && kind != InventoryRunLease {
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

func moveLeaseGraph(t *testing.T, state, relative string) string {
	t.Helper()
	parts := strings.Split(relative, "/")
	prior := filepath.Join(state, filepath.FromSlash(strings.Join(parts[:len(parts)-1], "/")))
	parts[len(parts)-2] = strings.Repeat("f", 64) + ".json"
	target := filepath.Join(state, filepath.FromSlash(strings.Join(parts[:len(parts)-1], "/")))
	if err := os.Rename(prior, target); err != nil {
		t.Fatal(err)
	}
	return strings.Join(parts, "/")
}
