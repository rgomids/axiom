//go:build unix

package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
)

type ownerInfo struct {
	os.FileInfo
	uid  uint32
	mode os.FileMode
}

func (i ownerInfo) Sys() any          { return &syscall.Stat_t{Uid: i.uid} }
func (i ownerInfo) Mode() os.FileMode { return i.mode }

func TestAncestorOwnershipRefusal(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	info, err := root.Stat(".")
	if err != nil {
		t.Fatal(err)
	}
	foreign := uint32(65534)
	if uint32(os.Geteuid()) == foreign {
		foreign--
	}
	for _, tc := range []struct {
		name          string
		uid           uint32
		mode          os.FileMode
		unsafe, owner bool
	}{
		{"root", 0, 0755, false, false}, {"user", uint32(os.Geteuid()), 0700, false, false},
		{"foreign", foreign, 0755, true, true}, {"foreign-sticky", foreign, os.ModeSticky | 0777, true, true},
		{"writable-root", 0, 0777, true, false}, {"sticky-root", 0, os.ModeSticky | 0777, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkAncestor(root, ownerInfo{info, tc.uid, tc.mode})
			var refusal *AncestorOwnerError
			if errors.Is(err, ErrUnsafe) != tc.unsafe || errors.As(err, &refusal) != tc.owner {
				t.Fatalf("err = %v", err)
			}
			if refusal != nil && (refusal.PathClass != "ancestor" || refusal.UID != foreign) {
				t.Fatalf("refusal = %+v", refusal)
			}
		})
	}
	// Injection reaches the actual rooted walker and store boundaries. No chown
	// or sandbox is needed; create must refuse before creating its first component.
	path := filepath.Join(t.TempDir(), "canary-private", "state")
	checker := func(root *os.Root, info os.FileInfo) error {
		return checkAncestor(root, ownerInfo{info, foreign, 0755})
	}
	_, _, err = anchoredRootChecked(path, true, checker)
	var refusal *AncestorOwnerError
	if !errors.As(err, &refusal) || refusal.PathClass != "filesystem_root" {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("mutation: %v", err)
	}
	store := InstallationStore{root: path, openExistingRoot: func(path string) (*os.Root, error) { r, _, e := anchoredRootChecked(path, false, checker); return r, e }}
	observation, category := store.Inspect(context.Background(), "synthetic-id")
	if category != AncestorOwnerCategory(err) || observation.Exists {
		t.Fatalf("Inspect = %+v, %s", observation, category)
	}
	resolved := store.Select(context.Background(), "synthetic")
	if resolved.Category != category {
		t.Fatalf("Select = %+v", resolved)
	}
	if _, err := store.ListInstalled(context.Background()); !errors.As(err, &refusal) {
		t.Fatalf("ListInstalled = %v", err)
	}
	if storageCategory(ErrUnsafe, "invalid_existing_local_state") != "invalid_existing_local_state" {
		t.Fatal("generic unsafe misclassified")
	}
}

// The installation state is readable; only the independent portable source sits
// under a foreign-owned ancestor. Listing must report that refusal, not corruption.
func TestProjectListPortableSourceAncestorOwnerRefusal(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	store, err := NewInstallationStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	source := privateDirectory(t, "canary-private-source")
	id := "123e4567-e89b-42d3-a456-426614174000"
	writeResolutionRecord(t, stateRoot, recordForResolution(id, "sample", source, nil))
	foreign := uint32(65534)
	if uint32(os.Geteuid()) == foreign {
		foreign--
	}
	store.openSourceRoot = func(path string) (*os.Root, error) {
		r, _, e := anchoredRootChecked(path, false, func(root *os.Root, info os.FileInfo) error {
			return checkAncestor(root, ownerInfo{info, foreign, 0755})
		})
		return r, e
	}
	before := directorySnapshot(t, stateRoot)
	result := projectapp.NewProjectCatalog(store, store).List(context.Background())
	parsed, ok := ParseAncestorOwnerCategory(result.Category)
	if result.Status != projectapp.ProjectListFailed || !ok || parsed.UID != foreign || strings.Contains(result.Category, "canary") {
		t.Fatalf("list = %+v", result)
	}
	if after := directorySnapshot(t, stateRoot); !reflect.DeepEqual(after, before) {
		t.Fatalf("refusal mutated state\nbefore=%v\nafter=%v", before, after)
	}
	// Genuinely unsafe sources keep the existing classification.
	store.openSourceRoot = func(string) (*os.Root, error) { return nil, ErrUnsafe }
	if result := projectapp.NewProjectCatalog(store, store).List(context.Background()); result.Category != "invalid_project_state" {
		t.Fatalf("generic unsafe = %+v", result)
	}
}

func TestAncestorOwnerCategoryBounded(t *testing.T) {
	for _, category := range []string{"untrusted_ancestor_owner:/private/canary:uid=1:uid_mapping=mapped", "untrusted_ancestor_owner:ancestor:uid=-1:uid_mapping=unknown", "untrusted_ancestor_owner:ancestor:uid=1:uid_mapping=anything", "invalid_existing_local_state"} {
		if _, ok := ParseAncestorOwnerCategory(category); ok {
			t.Fatalf("accepted %s", category)
		}
	}
	err := &AncestorOwnerError{PathClass: "ancestor", UID: 65534, UIDMapping: "unknown"}
	parsed, ok := ParseAncestorOwnerCategory(AncestorOwnerCategory(err))
	if !ok || parsed != *err {
		t.Fatalf("round trip: %+v", parsed)
	}
}
