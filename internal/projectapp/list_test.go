package projectapp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

type installedReader struct {
	records []projectapp.InstalledProject
	err     error
}

func (r installedReader) ListInstalled(context.Context) ([]projectapp.InstalledProject, error) {
	return append([]projectapp.InstalledProject(nil), r.records...), r.err
}

type definitionReader struct {
	projects map[string]project.Project
	errors   map[string]error
}

func (r definitionReader) ReadInstalledProject(_ context.Context, installed projectapp.InstalledProject) (project.Project, error) {
	return r.projects[installed.ID], r.errors[installed.ID]
}

func TestProjectCatalogListsEmptyAndSortedConfiguredProjects(t *testing.T) {
	definitions := definitionReader{projects: map[string]project.Project{}, errors: map[string]error{}}
	empty := projectapp.NewProjectCatalog(installedReader{}, definitions).List(context.Background())
	if empty.Status != projectapp.ProjectListSucceeded || empty.Projects == nil || len(empty.Projects) != 0 {
		t.Fatalf("empty list = %#v", empty)
	}

	firstID := "123e4567-e89b-42d3-a456-426614174000"
	secondID := "123e4567-e89b-42d3-a456-426614174001"
	definitions.projects[firstID] = configuredProject(t, firstID, "zeta", "Zeta")
	definitions.projects[secondID] = configuredProject(t, secondID, "alpha", "Alpha")
	result := projectapp.NewProjectCatalog(installedReader{records: []projectapp.InstalledProject{
		{ID: firstID, Slug: "zeta", Source: "/untrusted/zeta"},
		{ID: secondID, Slug: "alpha", Source: "/untrusted/alpha"},
	}}, definitions).List(context.Background())
	if result.Status != projectapp.ProjectListSucceeded || len(result.Projects) != 2 || result.Projects[0].Slug != "alpha" || result.Projects[1].Name != "Zeta" {
		t.Fatalf("sorted list = %#v", result)
	}
}

func TestProjectCatalogUsesIDAsSlugTieBreaker(t *testing.T) {
	firstID := "123e4567-e89b-42d3-a456-426614174000"
	secondID := "123e4567-e89b-42d3-a456-426614174001"
	definitions := definitionReader{projects: map[string]project.Project{
		firstID:  configuredProject(t, firstID, "shared", "First"),
		secondID: configuredProject(t, secondID, "shared", "Second"),
	}, errors: map[string]error{}}
	result := projectapp.NewProjectCatalog(installedReader{records: []projectapp.InstalledProject{
		{ID: secondID, Slug: "shared"},
		{ID: firstID, Slug: "shared"},
	}}, definitions).List(context.Background())
	if result.Status != projectapp.ProjectListSucceeded || len(result.Projects) != 2 || result.Projects[0].ID != firstID || result.Projects[1].ID != secondID {
		t.Fatalf("tie-break list = %#v", result)
	}
}

func TestProjectCatalogPreservesConfiguredProjectWhenDefinitionUnavailable(t *testing.T) {
	id := "123e4567-e89b-42d3-a456-426614174000"
	result := projectapp.NewProjectCatalog(
		installedReader{records: []projectapp.InstalledProject{{ID: id, Slug: "alpha", Source: "/missing"}}},
		definitionReader{projects: map[string]project.Project{}, errors: map[string]error{id: projectapp.ErrProjectDefinitionUnavailable}},
	).List(context.Background())
	if result.Status != projectapp.ProjectListSucceeded || len(result.Projects) != 1 || result.Projects[0] != (projectapp.ProjectSummary{ID: id, Slug: "alpha"}) {
		t.Fatalf("unavailable definition = %#v", result)
	}
}

func TestProjectCatalogFailsClosedWithoutPartialResults(t *testing.T) {
	id := "123e4567-e89b-42d3-a456-426614174000"
	result := projectapp.NewProjectCatalog(
		installedReader{records: []projectapp.InstalledProject{{ID: id, Slug: "alpha"}}},
		definitionReader{projects: map[string]project.Project{}, errors: map[string]error{id: projectapp.ErrUnsafe}},
	).List(context.Background())
	if result.Status != projectapp.ProjectListFailed || result.Category != "invalid_project_state" || result.Projects != nil {
		t.Fatalf("invalid definition = %#v", result)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	result = projectapp.NewProjectCatalog(installedReader{err: errors.New("ignored")}, definitionReader{}).List(cancelled)
	if result.Status != projectapp.ProjectListCancelled || result.Category != "cancelled" {
		t.Fatalf("cancelled list = %#v", result)
	}
}

func configuredProject(t *testing.T, id, slug, name string) project.Project {
	t.Helper()
	value, issues := project.New(project.State{SchemaVersion: 1, ID: id, Slug: slug, Name: name})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return value
}
