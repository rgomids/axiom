// Package workflow implements the bounded sequential Axiom POC workflow.
package workflow

import (
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var Gates = []string{"specification", "clarification", "plan", "tasks", "implementation", "review", "evidence", "reconciliation", "completion"}

type Repository struct{ Key, Path string }
type Project struct {
	ID           string
	Repositories []Repository
}
type Resolver interface {
	Resolve(context.Context, string) (Project, string)
}

type WorkItems interface {
	Available(context.Context, string, string, int) bool
	Complete(context.Context, string, string, int, bool) string
}

type Step struct {
	Gate, Status, Reference string
	Digest                  [32]byte
}

type State struct {
	ProjectID, RepositoryKey, RepositoryPath string
	WorkItem                                 int
	Status                                   string
	Current                                  int
	Steps                                    []Step
}

type Store interface {
	Create(context.Context, State) error
	Load(context.Context, string, string, int) (State, error)
	Save(context.Context, State) error
}

type Status string

const (
	Succeeded Status = "success"
	Failed    Status = "failed"
)

type Result struct {
	Status   Status
	Category string
	State    State
}

type Target struct {
	ProjectSelector, RepositoryKey string
	WorkItem                       int
}

type Service struct {
	resolver  Resolver
	workItems WorkItems
	store     Store
}

func New(resolver Resolver, workItems WorkItems, store Store) Service {
	return Service{resolver: resolver, workItems: workItems, store: store}
}

func (s Service) Start(ctx context.Context, target Target) Result {
	project, repository, result := s.resolve(ctx, target)
	if result.Status == Failed {
		return result
	}
	if !s.workItems.Available(ctx, target.ProjectSelector, target.RepositoryKey, target.WorkItem) {
		return failure("work_item_not_linked")
	}
	steps := make([]Step, len(Gates))
	for index, gate := range Gates {
		steps[index] = Step{Gate: gate, Status: "pending"}
	}
	state := State{ProjectID: project.ID, RepositoryKey: target.RepositoryKey, RepositoryPath: repository.Path, WorkItem: target.WorkItem, Status: "active", Steps: steps}
	if err := s.store.Create(ctx, state); err != nil {
		if existing, loadErr := s.store.Load(ctx, project.ID, target.RepositoryKey, target.WorkItem); loadErr == nil {
			if existing.RepositoryPath != repository.Path {
				return failure("workflow_repository_changed")
			}
			return Result{Status: Succeeded, Category: "workflow_already_started", State: existing}
		}
		return failure("workflow_write_failed")
	}
	return Result{Status: Succeeded, Category: "workflow_started", State: state}
}

func (s Service) Advance(ctx context.Context, target Target, gate, outcome, reference string, authorizeExternal bool) Result {
	project, repository, result := s.resolve(ctx, target)
	if result.Status == Failed {
		return result
	}
	state, err := s.store.Load(ctx, project.ID, target.RepositoryKey, target.WorkItem)
	if err != nil {
		return failure("workflow_not_found")
	}
	if state.Status == "completed" {
		return failure("workflow_already_completed")
	}
	if state.Status == "interrupted" {
		return failure("workflow_resume_required")
	}
	if state.RepositoryPath != repository.Path || state.Current >= len(state.Steps) || state.Steps[state.Current].Gate != gate {
		return failure("wrong_workflow_gate")
	}
	if outcome != "pass" && outcome != "fail" {
		return failure("invalid_workflow_result")
	}
	if gate == "completion" {
		return s.complete(ctx, target, state, outcome, authorizeExternal)
	}
	digest, ok := artifactDigest(repository.Path, reference)
	if !ok {
		return failure("workflow_artifact_unavailable")
	}
	step := &state.Steps[state.Current]
	step.Reference = reference
	step.Digest = digest
	if outcome == "fail" {
		step.Status = "failed"
		state.Status = "interrupted"
		if err := s.store.Save(ctx, state); err != nil {
			return failure("workflow_write_failed")
		}
		return Result{Status: Failed, Category: "workflow_interrupted", State: state}
	}
	step.Status = "passed"
	state.Current++
	state.Status = "active"
	if err := s.store.Save(ctx, state); err != nil {
		return failure("workflow_write_failed")
	}
	return Result{Status: Succeeded, Category: "workflow_advanced", State: state}
}

func (s Service) Resume(ctx context.Context, target Target) Result {
	state, result := s.load(ctx, target)
	if result.Status == Failed {
		return result
	}
	if state.Status != "interrupted" {
		return Result{Status: Succeeded, Category: "workflow_not_interrupted", State: state}
	}
	state.Status = "active"
	state.Steps[state.Current].Status = "pending"
	if err := s.store.Save(ctx, state); err != nil {
		return failure("workflow_write_failed")
	}
	return Result{Status: Succeeded, Category: "workflow_resumed", State: state}
}

func (s Service) Status(ctx context.Context, target Target) Result {
	state, result := s.load(ctx, target)
	if result.Status == Failed {
		return result
	}
	return Result{Status: Succeeded, Category: "workflow_" + state.Status, State: state}
}

func (s Service) load(ctx context.Context, target Target) (State, Result) {
	project, repository, result := s.resolve(ctx, target)
	if result.Status == Failed {
		return State{}, result
	}
	state, err := s.store.Load(ctx, project.ID, target.RepositoryKey, target.WorkItem)
	if err != nil {
		return State{}, failure("workflow_not_found")
	}
	if state.RepositoryPath != repository.Path {
		return State{}, failure("workflow_repository_changed")
	}
	return state, Result{}
}

func (s Service) complete(ctx context.Context, target Target, state State, outcome string, authorized bool) Result {
	if outcome != "pass" {
		return failure("completion_requires_pass")
	}
	if category := s.workItems.Complete(ctx, target.ProjectSelector, target.RepositoryKey, target.WorkItem, authorized); category != "work_item_completed" {
		return failure(category)
	}
	state.Steps[state.Current].Status = "passed"
	state.Current++
	state.Status = "completed"
	if err := s.store.Save(ctx, state); err != nil {
		return Result{Status: Failed, Category: "work_item_completed_workflow_write_failed", State: state}
	}
	return Result{Status: Succeeded, Category: "workflow_completed", State: state}
}

func (s Service) resolve(ctx context.Context, target Target) (Project, Repository, Result) {
	if s.resolver == nil || s.workItems == nil || s.store == nil || target.ProjectSelector == "" || target.RepositoryKey == "" || target.WorkItem <= 0 {
		return Project{}, Repository{}, failure("invalid_workflow_input")
	}
	project, category := s.resolver.Resolve(ctx, target.ProjectSelector)
	if category != "" {
		return Project{}, Repository{}, failure(category)
	}
	for _, repository := range project.Repositories {
		if repository.Key == target.RepositoryKey {
			return project, repository, Result{}
		}
	}
	return Project{}, Repository{}, failure("repository_not_configured")
}

func artifactDigest(root, reference string) ([32]byte, bool) {
	var zero [32]byte
	if reference == "" || filepath.IsAbs(reference) || strings.ContainsRune(reference, '\\') {
		return zero, false
	}
	clean := filepath.Clean(reference)
	if clean != reference || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return zero, false
	}
	anchored, err := os.OpenRoot(root)
	if err != nil {
		return zero, false
	}
	defer anchored.Close()
	file, err := anchored.Open(clean)
	if err != nil {
		return zero, false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1024*1024 {
		return zero, false
	}
	content, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil || len(content) > 1024*1024 {
		return zero, false
	}
	return sha256.Sum256(content), true
}

func failure(category string) Result { return Result{Status: Failed, Category: category} }
