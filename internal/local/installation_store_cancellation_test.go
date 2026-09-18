package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallationCancellationBeforePublicationLeavesNoRecord(t *testing.T) {
	source := privateTestRoot(t)
	manifest := []byte("schemaVersion: 1\nproject:\n  id: 123e4567-e89b-42d3-a456-426614174000\n  slug: sample\n  name: Sample\n")
	if err := os.WriteFile(filepath.Join(source, manifestName), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	state := privateTestRoot(t)
	store, err := NewInstallationStore(state)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	store.beforePublication = cancel
	if got := store.Install(ctx, source); got.Status != InstallationFailed || got.Category != "cancelled" {
		t.Fatalf("cancelled install = %+v", got)
	}
	reopened := store.Reopen(context.Background(), source)
	if reopened.Category != "reopened_without_local_state" {
		t.Fatalf("reopen after cancelled install = %+v", reopened)
	}
	record := filepath.Join(state, "projects", "123e4567-e89b-42d3-a456-426614174000", "installation.json")
	if _, err := os.Lstat(record); !os.IsNotExist(err) {
		t.Fatalf("record after cancelled install = %v", err)
	}
}
