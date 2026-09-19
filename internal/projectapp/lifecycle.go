package projectapp

import (
	"context"
	"errors"

	"github.com/rgomids/axiom/internal/project"
)

var (
	ErrNotFound         = errors.New("project not found")
	ErrConflict         = errors.New("project conflict")
	ErrUnsafe           = errors.New("unsafe project storage")
	ErrRecoveryRequired = errors.New("project recovery required")
)

// PortableLifecycleStore is consumer-owned. It has no path, document-body, or
// generic write method; the adapter controls storage from a validated slug.
type PortableLifecycleStore interface {
	Read(context.Context, string) ([]byte, error)
	Create(context.Context, string, []byte) error
	Update(context.Context, string, []byte, []byte) error
}

type Lifecycle struct {
	store PortableLifecycleStore
	codec ManifestCodec
	ids   IdentityAllocator
}

func NewLifecycle(store PortableLifecycleStore, codec ManifestCodec, ids IdentityAllocator) Lifecycle {
	return Lifecycle{store: store, codec: codec, ids: ids}
}

type InitRequest struct{ Slug, Name string }
type ConfigureRequest struct {
	Slug, Name     string
	RepositoryKeys []string
}
type ProjectRequest struct{ Slug string }
type UpdateRequest struct{ Slug, Name string }

type LifecycleStatus string

const (
	LifecycleApplied   LifecycleStatus = "applied"
	LifecycleUnchanged LifecycleStatus = "unchanged"
	LifecycleFailed    LifecycleStatus = "failed"
	LifecycleConflict  LifecycleStatus = "conflict"
	LifecycleCancelled LifecycleStatus = "cancelled"
)

type LifecycleResult struct {
	Status   LifecycleStatus
	Category string
}

func (l Lifecycle) Init(ctx context.Context, request InitRequest) LifecycleResult {
	if l.invalid() {
		return failed("application_unavailable")
	}
	if !project.ValidSlug(request.Slug) || request.Name == "" {
		return failed("invalid_input")
	}
	existing, err := l.store.Read(ctx, request.Slug)
	if err == nil {
		snapshot, issues := ReadSnapshot(l.codec, existing, nil)
		if len(issues) != 0 {
			return failed("invalid_existing_project")
		}
		if snapshot.Project().State().Name == request.Name && minimal(snapshot.Project()) {
			return LifecycleResult{Status: LifecycleUnchanged, Category: "already_initialized"}
		}
		return LifecycleResult{Status: LifecycleConflict, Category: "explicit_update_required"}
	}
	if !errors.Is(err, ErrNotFound) {
		return storageResult(err)
	}
	id, issues := l.ids.NewID()
	if len(issues) != 0 {
		return failed("identity_allocation_failed")
	}
	p, domainIssues := project.New(project.State{SchemaVersion: 1, ID: id, Slug: request.Slug, Name: request.Name})
	if len(domainIssues) != 0 {
		return failed("invalid_input")
	}
	manifest, issues := l.codec.Encode(p)
	if len(issues) != 0 {
		return failed("encoding_failed")
	}
	if _, issues := ReadSnapshot(l.codec, manifest, nil); len(issues) != 0 {
		return failed("encoding_failed")
	}
	return result(l.store.Create(ctx, request.Slug, manifest))
}

// Configure creates one complete portable Project including repository keys.
// Machine-local paths remain outside this request and portable domain state.
func (l Lifecycle) Configure(ctx context.Context, request ConfigureRequest) LifecycleResult {
	if l.invalid() {
		return failed("application_unavailable")
	}
	if !project.ValidSlug(request.Slug) || request.Name == "" || len(request.RepositoryKeys) == 0 {
		return failed("invalid_input")
	}
	repositories := make([]project.Repository, len(request.RepositoryKeys))
	for index, key := range request.RepositoryKeys {
		repositories[index] = project.Repository{Key: key}
	}
	existing, err := l.store.Read(ctx, request.Slug)
	if err == nil {
		snapshot, issues := ReadSnapshot(l.codec, existing, nil)
		if len(issues) != 0 {
			return failed("invalid_existing_project")
		}
		state := snapshot.Project().State()
		desired, domainIssues := project.New(project.State{SchemaVersion: 1, ID: state.ID, Slug: request.Slug, Name: request.Name, Repositories: project.Configured(repositories)})
		if len(domainIssues) != 0 {
			return failed("invalid_input")
		}
		if snapshot.Project().Equivalent(desired) {
			return LifecycleResult{Status: LifecycleUnchanged, Category: "already_configured"}
		}
		return LifecycleResult{Status: LifecycleConflict, Category: "explicit_update_required"}
	}
	if !errors.Is(err, ErrNotFound) {
		return storageResult(err)
	}
	id, issues := l.ids.NewID()
	if len(issues) != 0 {
		return failed("identity_allocation_failed")
	}
	p, domainIssues := project.New(project.State{SchemaVersion: 1, ID: id, Slug: request.Slug, Name: request.Name, Repositories: project.Configured(repositories)})
	if len(domainIssues) != 0 {
		return failed("invalid_input")
	}
	manifest, issues := l.codec.Encode(p)
	if len(issues) != 0 {
		return failed("encoding_failed")
	}
	if _, issues := ReadSnapshot(l.codec, manifest, nil); len(issues) != 0 {
		return failed("encoding_failed")
	}
	return result(l.store.Create(ctx, request.Slug, manifest))
}

func (l Lifecycle) Validate(ctx context.Context, request ProjectRequest) LifecycleResult {
	if l.invalid() {
		return failed("application_unavailable")
	}
	if !project.ValidSlug(request.Slug) {
		return failed("invalid_input")
	}
	manifest, err := l.store.Read(ctx, request.Slug)
	if err != nil {
		return storageResult(err)
	}
	if _, issues := ReadSnapshot(l.codec, manifest, nil); len(issues) != 0 {
		return failed("invalid_project")
	}
	return LifecycleResult{Status: LifecycleApplied, Category: "valid"}
}

func (l Lifecycle) Reopen(ctx context.Context, request ProjectRequest) LifecycleResult {
	result := l.Validate(ctx, request)
	if result.Status == LifecycleApplied {
		result.Category = "reopened"
	}
	return result
}

func (l Lifecycle) Update(ctx context.Context, request UpdateRequest) LifecycleResult {
	if l.invalid() {
		return failed("application_unavailable")
	}
	if !project.ValidSlug(request.Slug) || request.Name == "" {
		return failed("invalid_input")
	}
	manifest, err := l.store.Read(ctx, request.Slug)
	if err != nil {
		return storageResult(err)
	}
	snapshot, issues := ReadSnapshot(l.codec, manifest, nil)
	if len(issues) != 0 {
		return failed("invalid_project")
	}
	proposed, domainIssues := snapshot.Project().Propose(project.Intent{Name: project.Set(request.Name)})
	if len(domainIssues) != 0 {
		return failed("invalid_input")
	}
	encoded, issues := l.codec.Encode(proposed)
	if len(issues) != 0 {
		return failed("encoding_failed")
	}
	if _, issues := ReadSnapshot(l.codec, encoded, nil); len(issues) != 0 {
		return failed("encoding_failed")
	}
	if snapshot.Project().Equivalent(proposed) {
		return LifecycleResult{Status: LifecycleUnchanged, Category: "no_change"}
	}
	return result(l.store.Update(ctx, request.Slug, manifest, encoded))
}

func (l Lifecycle) invalid() bool { return l.store == nil || l.codec == nil || l.ids == nil }

func minimal(p project.Project) bool {
	s := p.State()
	return s.Repositories.Form() == project.Absent && s.Runtime.Form() == project.Absent && s.Providers.Form() == project.Absent && s.Integrations.Form() == project.Absent && s.ModelProfiles.Form() == project.Absent && s.BusinessContext.Form() == project.Absent && s.CredentialReferences.Form() == project.Absent && s.Policies.Form() == project.Absent
}

func result(err error) LifecycleResult {
	if err == nil {
		return LifecycleResult{Status: LifecycleApplied, Category: "applied"}
	}
	return storageResult(err)
}

func storageResult(err error) LifecycleResult {
	switch {
	case errors.Is(err, ErrNotFound):
		return failed("project_not_found")
	case errors.Is(err, ErrConflict):
		return LifecycleResult{Status: LifecycleConflict, Category: "conflict"}
	case errors.Is(err, context.Canceled):
		return LifecycleResult{Status: LifecycleCancelled, Category: "cancelled"}
	case errors.Is(err, ErrRecoveryRequired):
		return failed("recovery_required")
	default:
		return failed("storage_failure")
	}
}

func failed(category string) LifecycleResult {
	return LifecycleResult{Status: LifecycleFailed, Category: category}
}
