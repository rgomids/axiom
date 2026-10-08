package local

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
)

func TestProjectContextLocalPersistenceAndExecutionBinding(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	store, _ := NewInstallationStore(root)
	idA, idB := "123e4567-e89b-42d3-a456-426614174000", "123e4567-e89b-42d3-a456-426614174001"
	for _, p := range []struct{ id, slug string }{{idA, "alpha"}, {idB, "beta"}} {
		writeResolutionRecord(t, root, recordForResolution(p.id, p.slug, privateDirectory(t, p.slug), nil))
	}
	service := projectapp.ProjectContext{Preferences: store, Resolver: store}
	if category := service.Set(ctx, "", "alpha"); category != "" {
		t.Fatal(category)
	}
	if category := service.Set(ctx, "session-one", "beta"); category != "" {
		t.Fatal(category)
	}
	fresh, _ := NewInstallationStore(root)
	if id, err := fresh.ReadProjectContext(ctx, ""); id != idA || err != nil {
		t.Fatal(id, err)
	}
	if id, err := fresh.ReadProjectContext(ctx, "session-two"); id != "" || err != nil {
		t.Fatal(id, err)
	}
	view, category := service.Inspect(projectapp.WithProjectSession(ctx, "session-one"), "")
	if category != "" || view.Effective != idB || view.Default != idA || view.Session != idB {
		t.Fatal(view, category)
	}

	// Existing Execution storage is ID-addressed; context updates cannot rewrite
	// any identity, Runtime provenance, transitions or Evidence on that record.
	executions, _ := NewWorkflowStore(root)
	state := validExecution()
	state.ProjectID = idA
	if err := executions.Create(ctx, state); err != nil {
		t.Fatal(err)
	}
	before, err := executions.Load(ctx, idA, state.RepositoryKey, state.WorkItem)
	if err != nil {
		t.Fatal(err)
	}
	if category := service.Set(ctx, "", "beta"); category != "" {
		t.Fatal(category)
	}
	if category := service.Clear(ctx, "session-one"); category != "" {
		t.Fatal(category)
	}
	after, err := executions.Load(ctx, idA, state.RepositoryKey, state.WorkItem)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("Execution/provenance changed", err)
	}
	if err := os.RemoveAll(filepath.Join(root, "projects", idB)); err != nil {
		t.Fatal(err)
	}
	if _, category := service.Effective(ctx, ""); category != "project_not_found" {
		t.Fatal("deleted default fell back", category)
	}
	if explicit, category := service.Effective(ctx, "alpha"); category != "" || explicit.Effective != idA {
		t.Fatal(explicit, category)
	}
}

func TestProjectContextStrictStateAndReadOnlyAbsence(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	store, _ := NewInstallationStore(root)
	if id, err := store.ReadProjectContext(context.Background(), ""); id != "" || err != nil {
		t.Fatal(id, err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("read created state")
	}
	for _, wire := range []string{
		`{"formatVersion":1,"projectId":""}`, // the only valid empty preference
		`{"formatVersion":2,"projectId":""}`,
		`{"formatVersion":1,"projectId":"slug"}`,
		`{"formatVersion":1,"projectId":null}`,
		`{"formatVersion":1,"projectId":"","projectId":""}`,
		`{"formatVersion":1,"projectId":"","extra":true}`,
		`{"formatVersion":1,"ProjectId":""}`,
	} {
		_, err := decodeProjectContext([]byte(wire))
		if (err == nil) != (wire == `{"formatVersion":1,"projectId":""}`) {
			t.Fatalf("decode=%s err=%v", wire, err)
		}
	}
}

func TestProjectContextCorruptionPreservedAndRecoveryRefused(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	store, err := NewInstallationStore(root)
	if err != nil {
		t.Fatal(err)
	}
	id := "123e4567-e89b-42d3-a456-426614174000"
	if err := store.WriteProjectContext(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "project-context", "v1", "default.json")
	wire := []byte(`{"formatVersion":99,"projectId":""}`)
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadProjectContext(ctx, ""); err == nil {
		t.Fatal("newer state accepted")
	}
	if err := store.WriteProjectContext(ctx, "", ""); err == nil {
		t.Fatal("foreign state overwritten")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(wire) {
		t.Fatal("state changed")
	}
	marker := filepath.Join(root, "project-context", "v1", ".axiom-recovery-context")
	if err := os.WriteFile(marker, []byte("pending"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadProjectContext(ctx, "fresh-session"); err == nil {
		t.Fatal("pending recovery treated as absent session")
	}
}
