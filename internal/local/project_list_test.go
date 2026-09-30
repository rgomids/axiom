package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
)

func TestListInstalledUsesOnlyProtectedRecordsWithoutAvailabilityChecks(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	service, err := NewInstallationStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	missingSource := filepath.Join(t.TempDir(), "missing-source")
	missingRepository := filepath.Join(t.TempDir(), "missing-repository")
	id := "123e4567-e89b-42d3-a456-426614174000"
	writeResolutionRecord(t, stateRoot, recordForResolution(id, "sample", missingSource, []projectapp.RepositoryBinding{binding("main", missingRepository)}))

	before := directorySnapshot(t, stateRoot)
	unrelated := t.TempDir()
	prior, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(unrelated); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prior) })

	projects, err := service.ListInstalled(context.Background())
	if err != nil || len(projects) != 1 || projects[0].ID != id || projects[0].Slug != "sample" {
		t.Fatalf("installed projects = %#v, %v", projects, err)
	}
	if after := directorySnapshot(t, stateRoot); !reflect.DeepEqual(after, before) {
		t.Fatalf("listing mutated state\nbefore=%v\nafter=%v", before, after)
	}
}

func TestListInstalledReturnsExplicitEmptyState(t *testing.T) {
	service, err := NewInstallationStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	projects, err := service.ListInstalled(context.Background())
	if err != nil || projects == nil || len(projects) != 0 {
		t.Fatalf("empty installed projects = %#v, %v", projects, err)
	}
}

func TestReadInstalledProjectDistinguishesUnavailableFromUnsafeSource(t *testing.T) {
	service, err := NewInstallationStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	installed := projectapp.InstalledProject{ID: "123e4567-e89b-42d3-a456-426614174000", Slug: "sample", Source: filepath.Join(t.TempDir(), "missing")}
	if _, err := service.ReadInstalledProject(context.Background(), installed); !errors.Is(err, projectapp.ErrProjectDefinitionUnavailable) {
		t.Fatalf("missing source error = %v", err)
	}

	outside := privateDirectory(t, "outside-source")
	installed.Source = filepath.Join(t.TempDir(), "linked-source")
	if err := os.Symlink(outside, installed.Source); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReadInstalledProject(context.Background(), installed); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("unsafe source error = %v", err)
	}
}

func TestListInstalledFailsClosedForMalformedAndAmbiguousState(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		stateRoot := filepath.Join(t.TempDir(), "state")
		service, _ := NewInstallationStore(stateRoot)
		path := filepath.Join(stateRoot, "projects", "123e4567-e89b-42d3-a456-426614174000")
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "installation.json"), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if projects, err := service.ListInstalled(context.Background()); err == nil || projects != nil {
			t.Fatalf("malformed list = %#v, %v", projects, err)
		}
	})

	t.Run("symlinked record", func(t *testing.T) {
		stateRoot := filepath.Join(t.TempDir(), "state")
		service, _ := NewInstallationStore(stateRoot)
		id := "123e4567-e89b-42d3-a456-426614174000"
		directory := filepath.Join(stateRoot, "projects", id)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(t.TempDir(), "installation.json")
		if err := os.WriteFile(outside, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(directory, "installation.json")); err != nil {
			t.Fatal(err)
		}
		if projects, err := service.ListInstalled(context.Background()); err == nil || projects != nil {
			t.Fatalf("unsafe list = %#v, %v", projects, err)
		}
		if wire, err := os.ReadFile(outside); err != nil || string(wire) != "{}\n" {
			t.Fatalf("unsafe target changed: %q, %v", wire, err)
		}
	})

	t.Run("duplicate slug", func(t *testing.T) {
		stateRoot := filepath.Join(t.TempDir(), "state")
		service, _ := NewInstallationStore(stateRoot)
		for _, id := range []string{"123e4567-e89b-42d3-a456-426614174000", "123e4567-e89b-42d3-a456-426614174001"} {
			writeResolutionRecord(t, stateRoot, recordForResolution(id, "duplicate", filepath.Join(t.TempDir(), "missing"), nil))
		}
		if projects, err := service.ListInstalled(context.Background()); err == nil || projects != nil {
			t.Fatalf("ambiguous list = %#v, %v", projects, err)
		}
	})
}

func directorySnapshot(t *testing.T, root string) map[string]os.FileMode {
	t.Helper()
	result := map[string]os.FileMode{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			result[path] = info.Mode()
		}
		return nil
	})
	return result
}
