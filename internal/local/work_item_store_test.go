package local

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/workitem"
)

func TestWorkItemStoreRoundTrip(t *testing.T) {
	store, err := NewWorkItemStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	link := workitem.Link{ProjectID: "123e4567-e89b-42d3-a456-426614174000", RepositoryKey: "main", ProviderRepository: "owner/repo", Number: 7, URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Number)
	if err != nil || got != link {
		t.Fatalf("round trip = %#v, %v", got, err)
	}
	link.State = "CLOSED"
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	got, err = store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Number)
	if err != nil || got.State != "CLOSED" {
		t.Fatalf("updated state = %#v, %v", got, err)
	}
}
