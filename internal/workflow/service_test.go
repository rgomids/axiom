package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCompletionPreservesExternalStateWhenWorkItemPersistenceFails(t *testing.T) {
	repository := t.TempDir()
	store := &memoryStore{}
	items := &fakeWorkItems{
		exists: true,
		completion: WorkItemCompletion{
			Category: "provider_committed_local_failed",
			WorkItem: WorkItem{
				ProjectID:          "123e4567-e89b-42d3-a456-426614174000",
				RepositoryKey:      "main",
				ProviderRepository: "owner/repo",
				Number:             7,
				URL:                "https://github.com/owner/repo/issues/7",
				State:              "CLOSED",
			},
		},
	}
	service := New(fakeResolver{repository}, items, store)
	target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: 7}
	advanceToCompletion(t, service, target, repository)

	result := service.Advance(context.Background(), target, "completion", "pass", "", true)
	if result.Category != "provider_committed_local_failed" || result.WorkItem == nil || *result.WorkItem != confirmedClosedWorkItem() {
		t.Fatalf("completion result = %#v", result)
	}
}

func TestCompletionPreservesExternalStateWhenWorkflowPersistenceFails(t *testing.T) {
	repository := t.TempDir()
	store := &memoryStore{}
	items := &fakeWorkItems{exists: true}
	service := New(fakeResolver{repository}, items, store)
	target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: 7}
	advanceToCompletion(t, service, target, repository)
	store.failSave = true

	result := service.Advance(context.Background(), target, "completion", "pass", "", true)
	if result.Category != "work_item_completed_workflow_write_failed" || result.WorkItem == nil || *result.WorkItem != confirmedClosedWorkItem() {
		t.Fatalf("completion result = %#v", result)
	}
}

func TestWorkflowStoreRecoveryRequiredRemainsDistinct(t *testing.T) {
	repository := t.TempDir()
	store := &memoryStore{}
	service := New(fakeResolver{repository}, &fakeWorkItems{exists: true}, store)
	target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: 7}
	assertCategory(t, service.Start(context.Background(), target), Succeeded, "workflow_started")
	store.recoveryLoad = true
	assertCategory(t, service.Status(context.Background(), target), Failed, "recovery_required")
}

func TestWorkflowRunsSequentiallyInterruptsResumesAndCompletes(t *testing.T) {
	repository := t.TempDir()
	store := &memoryStore{}
	items := &fakeWorkItems{exists: true}
	service := New(fakeResolver{repository}, items, store)
	target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: 7}

	assertCategory(t, service.Start(context.Background(), target), Succeeded, "workflow_started")
	assertCategory(t, service.Advance(context.Background(), target, "plan", "pass", "plan.md", false), Failed, "wrong_workflow_gate")
	for _, gate := range Gates[:4] {
		writeArtifact(t, repository, gate+".md")
		assertCategory(t, service.Advance(context.Background(), target, gate, "pass", gate+".md", false), Succeeded, "workflow_advanced")
	}
	writeArtifact(t, repository, "implementation.md")
	assertCategory(t, service.Advance(context.Background(), target, "implementation", "fail", "implementation.md", false), Failed, "workflow_interrupted")
	assertCategory(t, service.Status(context.Background(), target), Succeeded, "workflow_interrupted")
	assertCategory(t, service.Advance(context.Background(), target, "implementation", "pass", "implementation.md", false), Failed, "workflow_resume_required")
	assertCategory(t, service.Resume(context.Background(), target), Succeeded, "workflow_resumed")
	for _, gate := range Gates[4:8] {
		writeArtifact(t, repository, gate+".md")
		assertCategory(t, service.Advance(context.Background(), target, gate, "pass", gate+".md", false), Succeeded, "workflow_advanced")
	}
	assertCategory(t, service.Advance(context.Background(), target, "completion", "pass", "", false), Failed, "external_mutation_denied")
	assertCategory(t, service.Advance(context.Background(), target, "completion", "pass", "", true), Succeeded, "workflow_completed")
	if !items.completed {
		t.Fatal("work item was not completed")
	}
}

func TestWorkflowRejectsMissingAndEscapingEvidence(t *testing.T) {
	repository := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repository, "linked.md")); err != nil {
		t.Fatal(err)
	}
	service := New(fakeResolver{repository}, &fakeWorkItems{exists: true}, &memoryStore{})
	target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: 7}
	assertCategory(t, service.Start(context.Background(), target), Succeeded, "workflow_started")
	for _, reference := range []string{"missing.md", "../outside.md", "linked.md"} {
		assertCategory(t, service.Advance(context.Background(), target, "specification", "pass", reference, false), Failed, "workflow_artifact_unavailable")
	}
}

func TestWorkflowRejectsChangedRepositoryBinding(t *testing.T) {
	resolver := &mutableResolver{repository: t.TempDir()}
	service := New(resolver, &fakeWorkItems{exists: true}, &memoryStore{})
	target := Target{ProjectSelector: "sample", RepositoryKey: "main", WorkItem: 7}
	assertCategory(t, service.Start(context.Background(), target), Succeeded, "workflow_started")
	resolver.repository = t.TempDir()
	assertCategory(t, service.Status(context.Background(), target), Failed, "workflow_repository_changed")
	assertCategory(t, service.Start(context.Background(), target), Failed, "workflow_repository_changed")
}

type fakeResolver struct{ repository string }

func (r fakeResolver) Resolve(context.Context, string) (Project, string) {
	return Project{ID: "123e4567-e89b-42d3-a456-426614174000", Repositories: []Repository{{Key: "main", Path: r.repository}}}, ""
}

type mutableResolver struct{ repository string }

func (r *mutableResolver) Resolve(context.Context, string) (Project, string) {
	return Project{ID: "123e4567-e89b-42d3-a456-426614174000", Repositories: []Repository{{Key: "main", Path: r.repository}}}, ""
}

type fakeWorkItems struct {
	exists, completed bool
	completion        WorkItemCompletion
}

func (f *fakeWorkItems) Available(context.Context, string, string, int) bool { return f.exists }
func (f *fakeWorkItems) Complete(_ context.Context, _, _ string, _ int, authorized bool) WorkItemCompletion {
	if !authorized {
		return WorkItemCompletion{Category: "external_mutation_denied"}
	}
	f.completed = true
	if f.completion.Category != "" {
		return f.completion
	}
	return WorkItemCompletion{
		Category: "work_item_completed",
		WorkItem: confirmedClosedWorkItem(),
	}
}

func confirmedClosedWorkItem() WorkItem {
	return WorkItem{
		ProjectID:          "123e4567-e89b-42d3-a456-426614174000",
		RepositoryKey:      "main",
		ProviderRepository: "owner/repo",
		Number:             7,
		URL:                "https://github.com/owner/repo/issues/7",
		State:              "CLOSED",
	}
}

type memoryStore struct {
	state        State
	created      bool
	failSave     bool
	recoveryLoad bool
}

func (s *memoryStore) Create(_ context.Context, state State) error {
	if s.created {
		return os.ErrExist
	}
	s.created = true
	s.state = state
	return nil
}
func (s *memoryStore) Load(context.Context, string, string, int) (State, error) {
	if s.recoveryLoad {
		return State{}, ErrRecoveryRequired
	}
	if !s.created {
		return State{}, os.ErrNotExist
	}
	return s.state, nil
}
func (s *memoryStore) Save(_ context.Context, state State) error {
	if s.failSave {
		return errors.New("write failed")
	}
	s.state = state
	return nil
}

func advanceToCompletion(t *testing.T, service Service, target Target, repository string) {
	t.Helper()
	assertCategory(t, service.Start(context.Background(), target), Succeeded, "workflow_started")
	for _, gate := range Gates[:8] {
		writeArtifact(t, repository, gate+".md")
		assertCategory(t, service.Advance(context.Background(), target, gate, "pass", gate+".md", false), Succeeded, "workflow_advanced")
	}
}

func writeArtifact(t *testing.T, root, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertCategory(t *testing.T, result Result, status Status, category string) {
	t.Helper()
	if result.Status != status || result.Category != category {
		t.Fatalf("result = %#v", result)
	}
}
