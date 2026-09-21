package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

func TestWorkItemStoreReportsMissingForFirstSelection(t *testing.T) {
	store, err := NewWorkItemStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Load(context.Background(), "123e4567-e89b-42d3-a456-426614174000", "main", 7)
	if !errors.Is(err, workitem.ErrNotFound) {
		t.Fatalf("missing load = %v", err)
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
	if err := store.Save(context.Background(), stale); !errors.Is(err, ErrConflict) || !errors.Is(err, workitem.ErrConflict) {
		t.Fatalf("stale update = %v", err)
	}
}

func TestPublishFileCleanupFailuresRequireRecoveryBeforeCommit(t *testing.T) {
	for _, failure := range []string{"marker_remove", "staging_remove", "cleanup_sync"} {
		t.Run(failure, func(t *testing.T) {
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
			store.hooks.fault = func(stage FaultStage) error {
				if stage == FaultF3 {
					return context.Canceled
				}
				return nil
			}
			if failure == "marker_remove" || failure == "staging_remove" {
				store.hooks.remove = func(root *os.Root, name string) error {
					prefix := ".axiom-recovery-"
					if failure == "staging_remove" {
						prefix = ".axiom-stage-file-"
					}
					if strings.HasPrefix(name, prefix) {
						return syscall.EIO
					}
					return root.Remove(name)
				}
			}
			if failure == "cleanup_sync" {
				syncCalls := 0
				store.hooks.sync = func(root *os.Root) error {
					syncCalls++
					if syncCalls == 2 {
						return syscall.EIO
					}
					return syncRoot(root)
				}
			}

			err = store.Save(context.Background(), loaded)
			var publication *PublicationError
			if !errors.As(err, &publication) || publication.Committed || !errors.Is(err, ErrRecoveryRequired) || !errors.Is(err, context.Canceled) {
				t.Fatalf("cleanup failure = %v", err)
			}
			if _, readErr := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Number); !errors.Is(readErr, ErrRecoveryRequired) {
				t.Fatalf("reader after cleanup failure = %v", readErr)
			}
		})
	}
}
