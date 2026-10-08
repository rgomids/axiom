package projectapp

import (
	"context"
	"errors"
	"sort"

	"github.com/rgomids/axiom/internal/project"
)

const maxProjectListTextBytes = 1 << 20

var ErrProjectDefinitionUnavailable = errors.New("project definition unavailable")

// InstalledProject is the protected local-state identity used for discovery.
// Source is consumed only by the adapter that reads portable display metadata.
type InstalledProject struct {
	ID, Slug, Source string
}

type InstalledProjectReader interface {
	ListInstalled(context.Context) ([]InstalledProject, error)
}

type InstalledProjectDefinitionReader interface {
	ReadInstalledProject(context.Context, InstalledProject) (project.Project, error)
}

type ProjectListStatus string

const (
	ProjectListSucceeded ProjectListStatus = "succeeded"
	ProjectListCancelled ProjectListStatus = "cancelled"
	ProjectListFailed    ProjectListStatus = "failed"
)

type ProjectSummary struct {
	ID, Slug, Name string
	// Status is the machine-local discoverable status (Issue #230). It is empty
	// only when the catalog has no operational-state reader.
	Status ProjectStatus
}

// ProjectListOptions selects the listing. The default excludes archived
// Projects; invalid and recovery_required Projects are never hidden.
type ProjectListOptions struct{ IncludeArchived bool }

type ProjectListResult struct {
	Status   ProjectListStatus
	Category string
	Projects []ProjectSummary
}

// ProjectCatalog lists configured Projects from protected local state. Portable
// definitions contribute display names only and never define membership.
type ProjectCatalog struct {
	installed   InstalledProjectReader
	definitions InstalledProjectDefinitionReader
	operational OperationalStore
}

// WithOperationalState returns a catalog that also reports each Project's
// machine-local status and hides archived Projects unless asked.
func (c ProjectCatalog) WithOperationalState(store OperationalStore) ProjectCatalog {
	c.operational = store
	return c
}

func NewProjectCatalog(installed InstalledProjectReader, definitions InstalledProjectDefinitionReader) ProjectCatalog {
	return ProjectCatalog{installed: installed, definitions: definitions}
}

// List lists Projects with the default options (archived excluded).
func (c ProjectCatalog) List(ctx context.Context) ProjectListResult {
	return c.ListProjects(ctx, ProjectListOptions{})
}

func (c ProjectCatalog) ListProjects(ctx context.Context, options ProjectListOptions) ProjectListResult {
	if c.installed == nil || c.definitions == nil {
		return failedProjectList("application_unavailable")
	}
	if err := ctx.Err(); err != nil {
		return cancelledProjectList()
	}
	records, err := c.installed.ListInstalled(ctx)
	if err != nil {
		return projectListError(ctx, err)
	}
	projects := make([]ProjectSummary, 0, len(records))
	textBytes := 0
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return cancelledProjectList()
		}
		status := ProjectStatus("")
		if c.operational != nil {
			state, category := InspectProjectState(ctx, c.operational, record.ID)
			if category == OperationalCancelled {
				return cancelledProjectList()
			}
			if category != "" {
				return failedProjectList("storage_failure")
			}
			status = state.Status
			if status == ProjectArchived && !options.IncludeArchived {
				continue
			}
		}
		definition, err := c.definitions.ReadInstalledProject(ctx, record)
		if errors.Is(err, ErrProjectDefinitionUnavailable) {
			textBytes += len(record.ID) + len(record.Slug)
			if textBytes > maxProjectListTextBytes {
				return failedProjectList("invalid_project_state")
			}
			projects = append(projects, ProjectSummary{ID: record.ID, Slug: record.Slug, Status: status})
			continue
		}
		if err != nil {
			if errors.Is(err, ErrUnsafe) {
				return failedProjectList("invalid_project_state")
			}
			return projectListError(ctx, err)
		}
		state := definition.State()
		if state.ID != record.ID || state.Slug != record.Slug {
			return failedProjectList("invalid_project_state")
		}
		textBytes += len(record.ID) + len(record.Slug) + len(state.Name)
		if textBytes > maxProjectListTextBytes {
			return failedProjectList("invalid_project_state")
		}
		projects = append(projects, ProjectSummary{ID: record.ID, Slug: record.Slug, Name: state.Name, Status: status})
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Slug != projects[j].Slug {
			return projects[i].Slug < projects[j].Slug
		}
		return projects[i].ID < projects[j].ID
	})
	return ProjectListResult{Status: ProjectListSucceeded, Category: "projects_listed", Projects: projects}
}

func projectListError(ctx context.Context, err error) ProjectListResult {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return cancelledProjectList()
	}
	if errors.Is(err, ErrRecoveryRequired) {
		return failedProjectList("recovery_required")
	}
	if errors.Is(err, ErrUnsafe) {
		return failedProjectList("invalid_existing_local_state")
	}
	return failedProjectList("storage_failure")
}

func failedProjectList(category string) ProjectListResult {
	return ProjectListResult{Status: ProjectListFailed, Category: category}
}

func cancelledProjectList() ProjectListResult {
	return ProjectListResult{Status: ProjectListCancelled, Category: "cancelled"}
}
