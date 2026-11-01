package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/provenance"
)

func TestS4StartConvergesAndConflictingScopeFails(t *testing.T) {
	service, store, _ := newS4Service(t)
	target := s4Target()

	first := service.Start(context.Background(), target)
	second := service.Start(context.Background(), target)
	if first.Category != "execution_started" || second.Category != "execution_already_started" {
		t.Fatalf("start results = %#v %#v", first, second)
	}
	if first.State.ExecutionID == "" || second.State.ExecutionID != first.State.ExecutionID || first.State.Revision != 1 {
		t.Fatalf("unstable execution lineage = %#v %#v", first.State, second.State)
	}

	store.state.RuntimeID = "other-runtime"
	conflict := service.Start(context.Background(), target)
	if conflict.Category != "execution_scope_conflict" {
		t.Fatalf("conflicting start = %#v", conflict)
	}
}

func TestS4SequentialTransitionRequiresExactRevisionAndReplayConverges(t *testing.T) {
	service, _, _ := newS4Service(t)
	target := s4Target()
	started := service.Start(context.Background(), target)

	skipped := service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: 1, Stage: Specification, Outcome: OutcomePassed})
	if skipped.Category != "workflow_stage_conflict" {
		t.Fatalf("skipped transition = %#v", skipped)
	}
	advanced := service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: 1, Stage: Intake, Outcome: OutcomePassed})
	if advanced.Category != "workflow_advanced" || advanced.State.Stage != Specification || advanced.State.Revision != 2 {
		t.Fatalf("advance = %#v", advanced)
	}
	replayed := service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: 1, Stage: Intake, Outcome: OutcomePassed})
	if replayed.Category != "workflow_transition_replayed" || replayed.State.Revision != 2 {
		t.Fatalf("replay = %#v", replayed)
	}
	stale := service.Transition(context.Background(), target, TransitionInput{ExpectedRevision: started.State.Revision, Stage: Intake, Outcome: OutcomeFailed})
	if stale.Category != "stale_execution_revision" {
		t.Fatalf("stale transition = %#v", stale)
	}
}

func TestS4CancelledOperationHasNoEffect(t *testing.T) {
	service, store, provider := newS4Service(t)
	target := s4Target()
	service.Start(context.Background(), target)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	transitioned := service.Transition(ctx, target, TransitionInput{ExpectedRevision: 1, Stage: Intake, Outcome: OutcomePassed})
	if transitioned.Status != Interrupted || store.state.Revision != 1 || len(provider.effects) != 0 {
		t.Fatalf("cancelled result = %#v state=%#v effects=%v", transitioned, store.state, provider.effects)
	}
}

func TestS4ClosedStateRejectsIncoherentHistoryAndDuplicateProjection(t *testing.T) {
	_, store, _ := newS4Service(t)
	state := State{ExecutionID: "018f4a44-7c31-7dd4-9d00-111111111111", FormatVersion: FormatVersion, WorkflowVersion: WorkflowVersion, ProjectID: "123e4567-e89b-42d3-a456-426614174000", RepositoryKey: "main", WorkItem: WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "7", URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}, RuntimeID: "codex", Stage: Specification, Revision: 2, Status: ExecutionActive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Provenance: identity(newS4Source(t))}
	state.Transitions = []Transition{{Revision: 2, From: Intake, To: Plan, Outcome: OutcomePassed, RequestDigest: digest("request"), CommittedAt: state.UpdatedAt, Provenance: state.Provenance}}
	if ValidState(state) {
		t.Fatal("skipped stage accepted")
	}
	state.Transitions[0].To = Specification
	record := ProjectionRecord{ExecutionRevision: 2, Key: projectionKey(state.ExecutionID, 2)}
	state.Projections = []ProjectionRecord{record, record}
	if ValidState(state) {
		t.Fatal("duplicate projection revision accepted")
	}
	_ = store
}

func newS4Service(t *testing.T) (Service, *s4Store, *s4Projection) {
	t.Helper()
	source := newS4Source(t)
	store := &s4Store{}
	provider := &s4Projection{observation: s4ProjectionObservation()}
	service := New(
		s4Resolver{},
		s4WorkItems{},
		store,
		nil,
		source,
		func() (string, error) { return "018f4a44-7c31-7dd4-9d00-111111111111", nil },
		func() time.Time { return time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC) },
	)
	return service, store, provider
}

func newS4Source(t *testing.T) provenance.Value {
	t.Helper()
	source, err := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func s4Target() Target {
	return Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: "7", RuntimeID: "codex"}
}

func s4ProjectionObservation() ProjectionObservation {
	return ProjectionObservation{
		Provider:         "github",
		Resource:         "owner/repo",
		IssueExternalID:  "7",
		IssueURL:         "https://github.com/owner/repo/issues/7",
		IssueState:       "OPEN",
		RepositoryLabels: []string{"axiom:stage:intake"},
		IssueLabels:      []string{"axiom:stage:intake"},
	}
}

type s4Resolver struct{}

func (s4Resolver) Resolve(context.Context, string) (Project, string) {
	return Project{ID: "123e4567-e89b-42d3-a456-426614174000", Repositories: []Repository{{Key: "main", Path: "/tmp/repository"}}}, ""
}

type s4WorkItems struct{}

func (s4WorkItems) Load(context.Context, string, string, string, string, string) (WorkItem, error) {
	return WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: "7", URL: "https://github.com/owner/repo/issues/7", State: "OPEN"}, nil
}

type s4Store struct {
	state             State
	created           bool
	saves, failSaveAt int
}

func (s *s4Store) Create(_ context.Context, state State) error {
	if s.created {
		return ErrConflict
	}
	s.created = true
	s.state = cloneState(state)
	return nil
}

func (s *s4Store) Load(context.Context, string, string, WorkItem) (State, error) {
	if !s.created {
		return State{}, ErrNotFound
	}
	return cloneState(s.state), nil
}

func (s *s4Store) Save(_ context.Context, state State) error {
	s.saves++
	if s.failSaveAt != 0 && s.saves == s.failSaveAt {
		return errors.New("injected save failure")
	}
	if state.StorageRevision != s.state.StorageRevision {
		return ErrConflict
	}
	state.StorageRevision[0]++
	s.state = cloneState(state)
	return nil
}

// Inert observation fixture for retained pure projection calculations.
type s4Projection struct {
	observation ProjectionObservation
	effects     []ProjectionEffect
}
