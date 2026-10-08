package workflow

import (
	"context"
	"testing"
)

// Issue #230 I230-T07: Execution discovery at the application boundary.

type listingStore struct {
	*s4Store
	states []State
	err    error
}

func (s *listingStore) List(context.Context, string) ([]State, error) {
	return s.states, s.err
}

func TestListDiscoversExecutionsAndFailsClosed(t *testing.T) {
	service, store, _ := newS4Service(t)
	if started := service.Start(context.Background(), s4Target()); started.Status != Succeeded {
		t.Fatalf("start = %+v", started)
	}
	// A store that cannot enumerate is never treated as empty.
	if result := service.List(context.Background(), Target{ProjectSelector: "sample"}); result.Status != ValidationFailed || result.Category != "invalid_execution_input" {
		t.Fatalf("non-lister = %+v", result)
	}
	lister := &listingStore{s4Store: store, states: []State{cloneState(store.state)}}
	service.store = lister
	listed := service.List(context.Background(), Target{ProjectSelector: "sample", RepositoryKey: "main"})
	if listed.Status != Succeeded || listed.Category != "executions_listed" || len(listed.Executions) != 1 || listed.Executions[0].ExecutionID != store.state.ExecutionID || listed.Executions[0].Stage != Intake {
		t.Fatalf("listed = %+v", listed)
	}
	for name, target := range map[string]Target{
		"execution identity": {ProjectSelector: "sample", ExecutionID: store.state.ExecutionID},
		"work item":          {ProjectSelector: "sample", WorkItem: "7"},
		"missing project":    {},
	} {
		if result := service.List(context.Background(), target); result.Category != "invalid_execution_input" {
			t.Fatalf("%s = %+v", name, result)
		}
	}
	if result := service.List(context.Background(), Target{ProjectSelector: "sample", RepositoryKey: "other"}); result.Category != "repository_not_configured" {
		t.Fatalf("unknown repository = %+v", result)
	}
	foreign := cloneState(store.state)
	foreign.ProjectID = "223e4567-e89b-42d3-a456-426614174000"
	lister.states = []State{foreign}
	if result := service.List(context.Background(), Target{ProjectSelector: "sample"}); result.Status != Failed || result.Category != "recovery_required" || result.Executions != nil {
		t.Fatalf("foreign record = %+v", result)
	}
	lister.states, lister.err = nil, ErrRecoveryRequired
	if result := service.List(context.Background(), Target{ProjectSelector: "sample"}); result.Status == Succeeded || result.Executions != nil {
		t.Fatalf("store recovery = %+v", result)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := service.List(ctx, Target{ProjectSelector: "sample"}); result.Status != Interrupted {
		t.Fatalf("cancelled = %+v", result)
	}
}
