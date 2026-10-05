package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/workitem"
)

func historicalWorkItem(t *testing.T) (WorkItemStore, workitem.Link, InventoryEntry) {
	t.Helper()
	store, err := NewWorkItemStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	link := testWorkItemLink()
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(store.root, "work-items", link.ProjectID)
	name := legacyWorkItemName(link.RepositoryKey, link.ExternalID)
	if err := os.Rename(filepath.Join(parent, workItemName(link.RepositoryKey, link.Provider, link.Resource, link.ExternalID)), filepath.Join(parent, name)); err != nil {
		t.Fatal(err)
	}
	link, err = store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Provider, link.Resource, link.ExternalID)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := os.ReadFile(filepath.Join(parent, name))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(wire)
	entry := InventoryEntry{Relative: "work-items/" + link.ProjectID + "/" + name, Kind: InventoryWorkItem, Digest: hex.EncodeToString(digest[:]), Bytes: int64(len(wire))}
	return store, link, entry
}

// The barrier is after the retirement read and immediately before deletion.
// A concurrent real WorkItemStore writer must conflict; after lock release a
// fresh successful writer's generation remains available to the normal reader.
func TestHistoricalRetirementCoordinatesConcurrentWorkItemWriter(t *testing.T) {
	store, link, entry := historicalWorkItem(t)
	ready, proceed := make(chan struct{}), make(chan struct{})
	retired := make(chan error, 1)
	go func() {
		retired <- retireHistoricalStateObject(context.Background(), store.root, entry, publicationHooks{beforeCommit: func() error {
			close(ready)
			<-proceed
			return nil
		}})
	}()
	select {
	case <-ready:
	case err := <-retired:
		t.Fatalf("retirement failed before barrier: %v", err)
	}
	link.State = "CLOSED"
	writerErr := store.Save(context.Background(), link)
	close(proceed)
	retirementErr := <-retired
	if !errors.Is(writerErr, ErrConflict) {
		t.Fatalf("writer during retirement = %v, want conflict", writerErr)
	}
	if retirementErr != nil {
		t.Fatalf("retirement = %v", retirementErr)
	}
	link.Revision = [32]byte{}
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatalf("fresh writer after retirement = %v", err)
	}
	got, err := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Provider, link.Resource, link.ExternalID)
	if err != nil || got.State != "CLOSED" {
		t.Fatalf("confirmed writer generation disappeared: %+v, %v", got, err)
	}
}

func TestHistoricalRetirementCannotRemoveConfirmedWorkItemRevision(t *testing.T) {
	store, link, entry := historicalWorkItem(t)
	link.State = "CLOSED"
	if err := store.Save(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	if err := RetireHistoricalStateObject(context.Background(), store.root, entry); !errors.Is(err, ErrConflict) {
		t.Fatalf("retirement of stale revision = %v", err)
	}
	got, err := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Provider, link.Resource, link.ExternalID)
	if err != nil || got.State != "CLOSED" {
		t.Fatalf("confirmed generation lost: %+v, %v", got, err)
	}
}

func TestHistoricalRetirementConflictsWithWorkItemPublicationInProgress(t *testing.T) {
	store, link, entry := historicalWorkItem(t)
	ready, proceed := make(chan struct{}), make(chan struct{})
	store.hooks.beforeCommit = func() error {
		close(ready)
		<-proceed
		return nil
	}
	link.State = "CLOSED"
	written := make(chan error, 1)
	go func() { written <- store.Save(context.Background(), link) }()
	select {
	case <-ready:
	case err := <-written:
		t.Fatalf("writer failed before barrier: %v", err)
	}
	retirementErr := RetireHistoricalStateObject(context.Background(), store.root, entry)
	close(proceed)
	writerErr := <-written
	if !errors.Is(retirementErr, ErrConflict) || writerErr != nil {
		t.Fatalf("retirement = %v, writer = %v", retirementErr, writerErr)
	}
	got, err := store.Load(context.Background(), link.ProjectID, link.RepositoryKey, link.Provider, link.Resource, link.ExternalID)
	if err != nil || got.State != "CLOSED" {
		t.Fatalf("confirmed concurrent generation lost: %+v, %v", got, err)
	}
}

func TestHistoricalRetirementRejectsLeafAndParentReplacement(t *testing.T) {
	for _, replace := range []string{"leaf", "parent"} {
		t.Run(replace, func(t *testing.T) {
			store, _, entry := historicalWorkItem(t)
			path := filepath.Join(store.root, filepath.FromSlash(entry.Relative))
			wire, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			err = retireHistoricalStateObject(context.Background(), store.root, entry, publicationHooks{beforeCommit: func() error {
				if replace == "leaf" {
					if err := os.Rename(path, path+".prior"); err != nil {
						return err
					}
					return os.WriteFile(path, wire, 0o600)
				}
				parent := filepath.Dir(path)
				if err := os.Rename(parent, parent+".prior"); err != nil {
					return err
				}
				if err := os.Mkdir(parent, 0o700); err != nil {
					return err
				}
				return os.WriteFile(path, wire, 0o600)
			}})
			if err == nil {
				t.Fatal("retirement accepted replaced generation")
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != string(wire) {
				t.Fatalf("replacement changed: %q, %v", got, err)
			}
		})
	}
}
