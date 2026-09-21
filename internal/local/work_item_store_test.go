package local

import (
	"context"
	"errors"
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
	if err != nil || got.ProjectID != link.ProjectID || got.State != link.State || got.Revision == ([32]byte{}) {
		t.Fatalf("round trip = %#v, %v", got, err)
	}
	link.Revision = got.Revision
	link.State = "CLOSED"
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	got, err = store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Number)
	if err != nil || got.State != "CLOSED" {
		t.Fatalf("updated state = %#v, %v", got, err)
	}
}

func TestWorkItemStoreRequiresExpectedRevisionAndFailsClosedAcrossF0F8(t *testing.T) {
	for index, stage := range []FaultStage{FaultF0, FaultF1, FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(stage), func(t *testing.T) {
			store, err := NewWorkItemStore(filepath.Join(t.TempDir(), "state"))
			if err != nil {
				t.Fatal(err)
			}
			link := workitem.Link{ProjectID: "123e4567-e89b-42d3-a456-426614174000", RepositoryKey: "main", ProviderRepository: "owner/repo", Number: 7, URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}
			if err := store.Save(context.Background(), link); err != nil {
				t.Fatal(err)
			}
			loaded, err := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Number)
			if err != nil {
				t.Fatal(err)
			}
			loaded.State = "CLOSED"
			store.hooks.fault = func(current FaultStage) error {
				if current == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			err = store.Save(context.Background(), loaded)
			var publication *PublicationError
			if !errors.As(err, &publication) || publication.Committed != (index >= 6) {
				t.Fatalf("fault outcome: %v", err)
			}
			_, readErr := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Number)
			if stage == FaultF0 || stage == FaultF1 {
				if readErr != nil {
					t.Fatalf("F0 lost prior authority: %v", readErr)
				}
				return
			}
			if !errors.Is(readErr, ErrRecoveryRequired) {
				t.Fatalf("stage %s reader = %v", stage, readErr)
			}
		})
	}
}

func TestWorkItemStoreRejectsStaleRevision(t *testing.T) {
	store, err := NewWorkItemStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	link := workitem.Link{ProjectID: "123e4567-e89b-42d3-a456-426614174000", RepositoryKey: "main", ProviderRepository: "owner/repo", Number: 7, URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	first, _ := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Number)
	stale := first
	first.State = "CLOSED"
	if err := store.Save(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), stale); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update = %v", err)
	}
}
