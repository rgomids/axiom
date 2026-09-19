package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/projectapp"
)

func TestResolveProjectBySlugAndIDOutsideConfiguredPaths(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	service, err := NewInstallationStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	projectSource := privateDirectory(t, "portable")
	firstRepository := privateDirectory(t, "repository-one")
	secondRepository := privateDirectory(t, "repository-two")
	id := "123e4567-e89b-42d3-a456-426614174000"
	writeResolutionRecord(t, stateRoot, recordForResolution(id, "sample", projectSource, []projectapp.RepositoryBinding{
		binding("api", firstRepository),
		binding("web", secondRepository),
	}))

	unrelated := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(unrelated); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	for _, selector := range []string{"sample", id} {
		result := service.Resolve(context.Background(), selector)
		if result.Status != ResolutionFound || result.Project.ID != id || len(result.Project.Repositories) != 2 {
			t.Fatalf("resolve %q = %#v", selector, result)
		}
	}
}

func TestResolveProjectFailsForAmbiguousSlug(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	service, err := NewInstallationStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"123e4567-e89b-42d3-a456-426614174000", "123e4567-e89b-42d3-a456-426614174001"} {
		writeResolutionRecord(t, stateRoot, recordForResolution(id, "duplicate", privateDirectory(t, id), nil))
	}
	result := service.Resolve(context.Background(), "duplicate")
	if result.Status != ResolutionFailed || result.Category != "project_ambiguous" {
		t.Fatalf("ambiguous resolution = %#v", result)
	}
}

func TestResolveProjectReportsMovedRepository(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	service, err := NewInstallationStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	repository := privateDirectory(t, "repository")
	writeResolutionRecord(t, stateRoot, recordForResolution("123e4567-e89b-42d3-a456-426614174000", "sample", privateDirectory(t, "portable"), []projectapp.RepositoryBinding{binding("main", repository)}))
	if err := os.Remove(repository); err != nil {
		t.Fatal(err)
	}
	result := service.Resolve(context.Background(), "sample")
	if result.Status != ResolutionFailed || result.Category != "repository_unavailable" {
		t.Fatalf("moved repository resolution = %#v", result)
	}
}

func recordForResolution(id, slug, source string, repositories []projectapp.RepositoryBinding) Record {
	digest := [32]byte{1}
	record, issues := NewRecord(RecordState{
		ProjectID:        id,
		ObservedSlug:     slug,
		SourceLocation:   source,
		PortableRevision: projectapp.RecordedPortableRevision(digest),
		ArtifactDigests:  []projectapp.ArtifactDigest{{Name: "axiom.yaml", Digest: digest}},
		Repositories:     repositories,
	})
	if len(issues) != 0 {
		panic(issues)
	}
	return record
}

func binding(key, path string) projectapp.RepositoryBinding {
	return projectapp.RepositoryBinding{RepositoryKey: key, ExplicitPath: path, Observation: projectapp.Observation{Availability: projectapp.Available, Basis: projectapp.PresentMetadata, ObservedAt: time.Unix(1, 0).UTC()}}
}

func privateDirectory(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeResolutionRecord(t *testing.T, root string, record Record) {
	t.Helper()
	wire, issues := EncodeRecord(record)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	path := filepath.Join(root, "projects", record.State().ProjectID)
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "installation.json"), wire, 0o600); err != nil {
		t.Fatal(err)
	}
}
