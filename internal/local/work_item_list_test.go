package local

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workitem"
)

func TestWorkItemStoreListEnumeratesLinksWithRevisionsAndSkipsCreateAttempts(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	store, err := NewWorkItemStore(state)
	if err != nil {
		t.Fatal(err)
	}
	projectID := testWorkItemLink().ProjectID
	if links, err := store.List(context.Background(), projectID); err != nil || links == nil || len(links) != 0 {
		t.Fatalf("missing Project list = %#v, %v", links, err)
	}
	first := testWorkItemLink()
	second := first
	second.Resource, second.URL = "other/repo", "https://github.com/other/repo/issues/7"
	for _, link := range []workitem.Link{first, second} {
		if err := store.Save(context.Background(), link); err != nil {
			t.Fatal(err)
		}
	}
	projectRoot := filepath.Join(state, "work-items", projectID)
	legacy := []byte("{\"formatVersion\":1,\"projectId\":\"" + projectID + "\",\"repositoryKey\":\"docs\",\"providerRepository\":\"owner/repo\",\"number\":3,\"url\":\"https://github.com/owner/repo/issues/3\",\"state\":\"CLOSED\"}\n")
	if err := os.WriteFile(filepath.Join(projectRoot, "docs-3.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	attempt := workitem.CreateAttempt{Target: workitem.DraftTarget{ProjectID: projectID, RepositoryKey: "main", Provider: "github", Resource: "owner/repo"}, Correlation: strings.Repeat("a", 64), PreviewDigest: strings.Repeat("b", 64), State: workitem.CreateAttemptPending}
	if err := store.SaveCreateAttempt(context.Background(), attempt); err != nil {
		t.Fatal(err)
	}
	links, err := store.List(context.Background(), projectID)
	if err != nil || len(links) != 3 {
		t.Fatalf("list = %#v, %v", links, err)
	}
	seen := map[string]workitem.Link{}
	for _, link := range links {
		seen[link.RepositoryKey+":"+link.Resource+"#"+link.ExternalID] = link
	}
	if seen["docs:owner/repo#3"].State != "CLOSED" || seen["docs:owner/repo#3"].Revision != sha256.Sum256(legacy) || seen["main:other/repo#7"].URL != second.URL || seen["main:owner/repo#7"].Revision == ([32]byte{}) {
		t.Fatalf("listed links = %#v", seen)
	}
	// Listed revisions are the exact CAS revisions Load returns.
	loaded, err := store.Load(context.Background(), projectID, "main", "github", "owner/repo", "7")
	if err != nil || loaded.Revision != seen["main:owner/repo#7"].Revision {
		t.Fatalf("revision mismatch: %#v %v", loaded, err)
	}
}

func TestWorkItemStoreListFailsClosedOnCorruptForeignOrInterruptedState(t *testing.T) {
	projectID := testWorkItemLink().ProjectID
	otherProject := "223e4567-e89b-42d3-a456-426614174000"
	foreign := []byte("{\"formatVersion\":1,\"projectId\":\"" + otherProject + "\",\"repositoryKey\":\"main\",\"providerRepository\":\"owner/repo\",\"number\":9,\"url\":\"https://github.com/owner/repo/issues/9\",\"state\":\"OPEN\"}\n")
	misnamed := []byte("{\"formatVersion\":1,\"projectId\":\"" + projectID + "\",\"repositoryKey\":\"main\",\"providerRepository\":\"owner/repo\",\"number\":9,\"url\":\"https://github.com/owner/repo/issues/9\",\"state\":\"OPEN\"}\n")
	for name, test := range map[string]struct {
		file string
		wire []byte
		want error
	}{
		"corrupt":           {"main-8.json", []byte("{not json"), nil},
		"future format":     {"main-8.json", []byte(`{"formatVersion":2}` + "\n"), nil},
		"foreign project":   {"main-9.json", foreign, ErrUnsafe},
		"misnamed link":     {"main-10.json", misnamed, ErrUnsafe},
		"unknown entry":     {"notes.txt", []byte("x"), ErrUnsafe},
		"hidden entry":      {".hidden.json", []byte("x"), ErrUnsafe},
		"interrupted stage": {".axiom-stage-file-0011", []byte("x"), workitem.ErrRecoveryRequired},
		"recovery marker":   {".axiom-recovery-0011", []byte("x"), workitem.ErrRecoveryRequired},
	} {
		t.Run(name, func(t *testing.T) {
			state := filepath.Join(t.TempDir(), "state")
			store, err := NewWorkItemStore(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Save(context.Background(), testWorkItemLink()); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(state, "work-items", projectID, test.file), test.wire, 0o600); err != nil {
				t.Fatal(err)
			}
			links, err := store.List(context.Background(), projectID)
			if err == nil || links != nil || test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("list = %#v, %v", links, err)
			}
		})
	}
	store, err := NewWorkItemStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(context.Background(), "../escape"); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("unsafe Project ID = %v", err)
	}
}
