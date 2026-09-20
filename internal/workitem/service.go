// Package workitem implements the bounded provider-neutral Work Item use cases
// required by the E2E POC.
package workitem

import "context"

type Repository struct{ Key, Path string }
type Project struct {
	ID           string
	Repositories []Repository
}

type Resolver interface {
	Resolve(context.Context, string) (Project, string)
}

type RepositoryLocator interface {
	GitHubRepository(context.Context, string) (string, error)
}

type External struct {
	Number int
	URL    string
	State  string
}

type Provider interface {
	Create(context.Context, string, string, string) (External, error)
	Read(context.Context, string, int) (External, error)
	Comment(context.Context, string, int, string) error
	Close(context.Context, string, int) (External, error)
}

type Link struct {
	ProjectID, RepositoryKey, ProviderRepository string
	Number                                       int
	URL, State                                   string
}

type Store interface {
	Save(context.Context, Link) error
	Load(context.Context, string, string, int) (Link, error)
}

type Status string

const (
	Succeeded Status = "success"
	Failed    Status = "failed"
)

type Result struct {
	Status   Status
	Category string
	Link     Link
}

type Service struct {
	resolver Resolver
	locator  RepositoryLocator
	provider Provider
	store    Store
}

func New(resolver Resolver, locator RepositoryLocator, provider Provider, store Store) Service {
	return Service{resolver: resolver, locator: locator, provider: provider, store: store}
}

type Target struct {
	ProjectSelector, RepositoryKey string
}

func (s Service) Create(ctx context.Context, target Target, title, body string, authorized bool) Result {
	if !authorized {
		return failure("external_mutation_denied")
	}
	project, repository, result := s.resolve(ctx, target)
	if result.Status == Failed {
		return result
	}
	external, err := s.provider.Create(ctx, repository, title, body)
	if err != nil {
		return failure("github_mutation_failed")
	}
	result = s.persist(ctx, project, target.RepositoryKey, repository, external)
	if result.Status == Failed {
		result.Category = "provider_committed_local_failed"
	}
	return result
}

func (s Service) Select(ctx context.Context, target Target, number int) Result {
	project, repository, result := s.resolve(ctx, target)
	if result.Status == Failed {
		return result
	}
	external, err := s.provider.Read(ctx, repository, number)
	if err != nil {
		return failure("github_read_failed")
	}
	return s.persist(ctx, project, target.RepositoryKey, repository, external)
}

func (s Service) Show(ctx context.Context, target Target, number int) Result {
	project, _, result := s.resolve(ctx, target)
	if result.Status == Failed {
		return result
	}
	link, err := s.store.Load(ctx, project.ID, target.RepositoryKey, number)
	if err != nil {
		return failure("work_item_not_found")
	}
	return Result{Status: Succeeded, Category: "work_item_loaded", Link: link}
}

func (s Service) Comment(ctx context.Context, target Target, number int, message string, authorized bool) Result {
	if !authorized {
		return failure("external_mutation_denied")
	}
	shown := s.Show(ctx, target, number)
	if shown.Status == Failed {
		return shown
	}
	if err := s.provider.Comment(ctx, shown.Link.ProviderRepository, number, message); err != nil {
		return failure("github_mutation_failed")
	}
	shown.Category = "work_item_commented"
	return shown
}

func (s Service) Complete(ctx context.Context, target Target, number int, authorized bool) Result {
	if !authorized {
		return failure("external_mutation_denied")
	}
	shown := s.Show(ctx, target, number)
	if shown.Status == Failed {
		return shown
	}
	external, err := s.provider.Close(ctx, shown.Link.ProviderRepository, number)
	if err != nil {
		return failure("github_mutation_failed")
	}
	result := s.persist(ctx, Project{ID: shown.Link.ProjectID}, shown.Link.RepositoryKey, shown.Link.ProviderRepository, external)
	if result.Status == Failed {
		result.Category = "provider_committed_local_failed"
		return result
	}
	result.Category = "work_item_completed"
	return result
}

func (s Service) resolve(ctx context.Context, target Target) (Project, string, Result) {
	if s.resolver == nil || s.locator == nil || s.provider == nil || s.store == nil || target.ProjectSelector == "" || target.RepositoryKey == "" {
		return Project{}, "", failure("invalid_work_item_input")
	}
	project, category := s.resolver.Resolve(ctx, target.ProjectSelector)
	if category != "" {
		return Project{}, "", failure(category)
	}
	for _, repository := range project.Repositories {
		if repository.Key != target.RepositoryKey {
			continue
		}
		identity, err := s.locator.GitHubRepository(ctx, repository.Path)
		if err != nil {
			return Project{}, "", failure("github_repository_unavailable")
		}
		return project, identity, Result{}
	}
	return Project{}, "", failure("repository_not_configured")
}

func (s Service) persist(ctx context.Context, project Project, key, repository string, external External) Result {
	if !validExternal(repository, external.Number, external) {
		return failure("invalid_github_response")
	}
	link := Link{ProjectID: project.ID, RepositoryKey: key, ProviderRepository: repository, Number: external.Number, URL: external.URL, State: external.State}
	if err := s.store.Save(ctx, link); err != nil {
		return Result{Status: Failed, Category: "local_work_item_write_failed", Link: link}
	}
	return Result{Status: Succeeded, Category: "work_item_linked", Link: link}
}

func failure(category string) Result { return Result{Status: Failed, Category: category} }
