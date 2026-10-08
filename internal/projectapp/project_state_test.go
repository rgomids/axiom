package projectapp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 (I230-T03) Evidence: discoverable status and listing filter.

type stateStore struct {
	states map[string]projectapp.OperationalState
	errors map[string]error
}

func (s stateStore) InspectOperational(_ context.Context, id string) (projectapp.OperationalObservation, error) {
	if err := s.errors[id]; err != nil {
		return projectapp.OperationalObservation{}, err
	}
	if state, ok := s.states[id]; ok {
		return projectapp.OperationalObservation{Exists: true, Revision: "r-" + id, State: state}, nil
	}
	return projectapp.OperationalObservation{Revision: projectapp.OperationalRevisionAbsent, State: projectapp.DefaultOperationalState()}, nil
}

func (stateStore) CommitOperational(context.Context, string, string, projectapp.OperationalState) error {
	panic("discovery must never write")
}

func TestInspectProjectStateNeverReportsUnreadableStateAsActive(t *testing.T) {
	archived := projectapp.OperationalState{ProjectStatus: projectapp.ProjectArchived, DisabledIntegrations: []string{"work-items"}}
	store := stateStore{
		states: map[string]projectapp.OperationalState{"123e4567-e89b-42d3-a456-426614174001": archived},
		errors: map[string]error{
			"123e4567-e89b-42d3-a456-426614174002": projectapp.ErrUnsafe,
			"123e4567-e89b-42d3-a456-426614174003": projectapp.ErrRecoveryRequired,
			"123e4567-e89b-42d3-a456-426614174004": projectapp.ErrNotFound,
			"123e4567-e89b-42d3-a456-426614174005": errors.New("disk"),
		},
	}
	for id, want := range map[string]projectapp.ProjectStatus{
		"123e4567-e89b-42d3-a456-426614174000": projectapp.ProjectActive,
		"123e4567-e89b-42d3-a456-426614174001": projectapp.ProjectArchived,
		"123e4567-e89b-42d3-a456-426614174002": projectapp.ProjectInvalid,
		"123e4567-e89b-42d3-a456-426614174003": projectapp.ProjectRecoveryRequired,
		"123e4567-e89b-42d3-a456-426614174004": projectapp.ProjectInvalid,
	} {
		state, category := projectapp.InspectProjectState(context.Background(), store, id)
		if category != "" || state.Status != want {
			t.Fatalf("%s = %+v %q, want %s", id, state, category, want)
		}
		if (want == projectapp.ProjectActive || want == projectapp.ProjectArchived) != (state.Operational != nil) {
			t.Fatalf("%s operational = %+v", id, state.Operational)
		}
	}
	if state, category := projectapp.InspectProjectState(context.Background(), store, "123e4567-e89b-42d3-a456-426614174005"); category != projectapp.OperationalStorageFailure || state.Status != "" {
		t.Fatalf("storage failure = %+v %q", state, category)
	}
	if _, category := projectapp.InspectProjectState(context.Background(), nil, "123e4567-e89b-42d3-a456-426614174000"); category == "" {
		t.Fatal("missing store must fail, not default to active")
	}
}

func TestProjectCatalogFiltersArchivedAndKeepsDeterministicOrder(t *testing.T) {
	ids := []string{"123e4567-e89b-42d3-a456-426614174010", "123e4567-e89b-42d3-a456-426614174011", "123e4567-e89b-42d3-a456-426614174012", "123e4567-e89b-42d3-a456-426614174013", "123e4567-e89b-42d3-a456-426614174014"}
	definitions := definitionReader{projects: map[string]project.Project{}, errors: map[string]error{}}
	records := []projectapp.InstalledProject{}
	for index, slug := range []string{"echo", "alpha", "delta", "bravo", "charlie"} {
		definitions.projects[ids[index]] = configuredProject(t, ids[index], slug, slug)
		records = append(records, projectapp.InstalledProject{ID: ids[index], Slug: slug})
	}
	store := stateStore{
		states: map[string]projectapp.OperationalState{ids[1]: {ProjectStatus: projectapp.ProjectArchived, DisabledIntegrations: []string{}}},
		errors: map[string]error{ids[2]: projectapp.ErrUnsafe, ids[3]: projectapp.ErrRecoveryRequired},
	}
	catalog := projectapp.NewProjectCatalog(installedReader{records: records}, definitions).WithOperationalState(store)
	summarize := func(result projectapp.ProjectListResult) []string {
		t.Helper()
		if result.Status != projectapp.ProjectListSucceeded {
			t.Fatalf("list = %+v", result)
		}
		values := []string{}
		for _, summary := range result.Projects {
			values = append(values, summary.Slug+":"+string(summary.Status))
		}
		return values
	}
	equal := func(got, want []string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for index := range got {
			if got[index] != want[index] {
				t.Fatalf("got %v, want %v", got, want)
			}
		}
	}
	// Only archived is hidden by default; invalid and recovery never are.
	equal(summarize(catalog.List(context.Background())), []string{"bravo:recovery_required", "charlie:active", "delta:invalid", "echo:active"})
	equal(summarize(catalog.ListProjects(context.Background(), projectapp.ProjectListOptions{})), []string{"bravo:recovery_required", "charlie:active", "delta:invalid", "echo:active"})
	equal(summarize(catalog.ListProjects(context.Background(), projectapp.ProjectListOptions{IncludeArchived: true})), []string{"alpha:archived", "bravo:recovery_required", "charlie:active", "delta:invalid", "echo:active"})
}

func TestProjectCatalogOperationalFailuresFailTheListing(t *testing.T) {
	id := "123e4567-e89b-42d3-a456-426614174000"
	definitions := definitionReader{projects: map[string]project.Project{id: configuredProject(t, id, "alpha", "Alpha")}, errors: map[string]error{}}
	records := installedReader{records: []projectapp.InstalledProject{{ID: id, Slug: "alpha"}}}
	failing := projectapp.NewProjectCatalog(records, definitions).WithOperationalState(stateStore{errors: map[string]error{id: errors.New("disk")}}).List(context.Background())
	if failing.Status != projectapp.ProjectListFailed || failing.Category != "storage_failure" || failing.Projects != nil {
		t.Fatalf("storage failure = %+v", failing)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if cancelled := projectapp.NewProjectCatalog(records, definitions).WithOperationalState(stateStore{}).List(ctx); cancelled.Status != projectapp.ProjectListCancelled {
		t.Fatalf("cancelled = %+v", cancelled)
	}
	// A catalog without a reader keeps its established behavior and claims no status.
	plain := projectapp.NewProjectCatalog(records, definitions).List(context.Background())
	if plain.Status != projectapp.ProjectListSucceeded || len(plain.Projects) != 1 || plain.Projects[0].Status != "" {
		t.Fatalf("catalog without reader = %+v", plain)
	}
}

type updateSelection struct {
	id, category string
}

func (s updateSelection) SelectProjectID(context.Context, string) (string, string) {
	return s.id, s.category
}

type fixedIDs struct{}

func (fixedIDs) NewID() (string, []projectapp.Issue) {
	return "123e4567-e89b-42d3-a456-426614174000", nil
}

type recordingPortable struct{ writes int }

func (p *recordingPortable) Read(context.Context, string) ([]byte, error) {
	return nil, projectapp.ErrNotFound
}
func (p *recordingPortable) Create(context.Context, string, []byte) error { p.writes++; return nil }
func (p *recordingPortable) Update(context.Context, string, []byte, []byte) error {
	p.writes++
	return nil
}

func TestUpdateUninstalledRefusesInstalledAndUnknownStateWithoutWrites(t *testing.T) {
	store := &recordingPortable{}
	lifecycle := projectapp.NewLifecycle(store, manifest.Codec{}, fixedIDs{})
	request := projectapp.UpdateRequest{Slug: "sample", Name: "Changed"}
	for name, test := range map[string]struct {
		selection updateSelection
		status    projectapp.LifecycleStatus
		category  string
	}{
		"installed":        {updateSelection{id: "123e4567-e89b-42d3-a456-426614174000"}, projectapp.LifecycleFailed, projectapp.ExplicitEditRequired},
		"ambiguous":        {updateSelection{category: "project_ambiguous"}, projectapp.LifecycleFailed, projectapp.ExplicitEditRequired},
		"cancelled":        {updateSelection{category: "cancelled"}, projectapp.LifecycleCancelled, "cancelled"},
		"unsafe state":     {updateSelection{category: "invalid_existing_local_state"}, projectapp.LifecycleFailed, "invalid_existing_local_state"},
		"recovery pending": {updateSelection{category: "recovery_required"}, projectapp.LifecycleFailed, "recovery_required"},
	} {
		if got := lifecycle.UpdateUninstalled(context.Background(), test.selection, request); got.Status != test.status || got.Category != test.category {
			t.Fatalf("%s = %+v", name, got)
		}
	}
	if got := lifecycle.UpdateUninstalled(context.Background(), updateSelection{category: "project_not_found"}, projectapp.UpdateRequest{Slug: "Bad Slug", Name: "x"}); got.Category != "invalid_input" {
		t.Fatalf("invalid slug = %+v", got)
	}
	if got := lifecycle.UpdateUninstalled(context.Background(), nil, request); got.Status != projectapp.LifecycleFailed {
		t.Fatalf("missing selection = %+v", got)
	}
	if store.writes != 0 {
		t.Fatalf("refused updates wrote %d times", store.writes)
	}
}
