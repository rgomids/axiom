package workflow

import (
	"context"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
)

type bindingPreferences map[string]string

func (p bindingPreferences) ReadProjectContext(_ context.Context, session string) (string, error) {
	return p[session], nil
}
func (p bindingPreferences) WriteProjectContext(_ context.Context, session, id string) error {
	p[session] = id
	return nil
}

type bindingResolver struct{}

func (bindingResolver) ResolveContextProject(_ context.Context, selector string) (string, string) {
	return selector, ""
}
func (bindingResolver) Resolve(_ context.Context, selector string) (Project, string) {
	return Project{ID: selector, Repositories: []Repository{{Key: "main", Path: "/tmp/repository"}}}, ""
}

func TestExecutionStartCapturesEffectiveProjectBeforeContextChanges(t *testing.T) {
	ctx := context.Background()
	idA, idB := "123e4567-e89b-42d3-a456-426614174000", "123e4567-e89b-42d3-a456-426614174001"
	preferences := bindingPreferences{"": idA}
	selection := projectapp.ProjectContext{Preferences: preferences, Resolver: bindingResolver{}}
	view, category := selection.Effective(ctx, "")
	if category != "" {
		t.Fatal(category)
	}
	service, store, _ := newS4Service(t)
	service.resolver = bindingResolver{}
	target := s4Target()
	target.ProjectSelector = view.Effective
	started := service.Start(ctx, target)
	if started.Category != "execution_started" || started.State.ProjectID != idA {
		t.Fatal(started)
	}
	before := store.state
	if selection.Set(ctx, "", idB) != "" || selection.Set(ctx, "one", idB) != "" {
		t.Fatal("context change failed")
	}
	view, category = selection.Effective(projectapp.WithProjectSession(ctx, "one"), "")
	if category != "" || view.Effective != idB {
		t.Fatal(view, category)
	}
	if !reflect.DeepEqual(before, store.state) || started.State.ProjectID != idA {
		t.Fatal("Execution or provenance changed")
	}
	retained := service.Status(ctx, target)
	if retained.State.ProjectID != idA || retained.State.ExecutionID != started.State.ExecutionID || !reflect.DeepEqual(retained.State.Provenance, started.State.Provenance) {
		t.Fatal(retained)
	}
}
