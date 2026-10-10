package main

import (
	"context"
	"path/filepath"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type activeWorkflowReader struct{ s lifecycleService }

func (r activeWorkflowReader) ReadActiveWorkflow(ctx context.Context, selector string) projectapp.ActiveWorkflow {
	resolved := r.s.installation.Select(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return projectapp.ResolveActiveWorkflow(project.State{}, workflowdefinition.Catalog{}, resolved.Category)
	}
	installed := projectapp.InstalledProject{ID: resolved.Project.ID, Slug: resolved.Project.Slug, Source: resolved.Project.Source}
	store, err := local.NewPortableStore(filepath.Dir(installed.Source))
	if err != nil || filepath.Base(installed.Source) != installed.Slug {
		return projectapp.ResolveActiveWorkflow(project.State{}, workflowdefinition.Catalog{}, "unsupported_portable_source")
	}
	snapshot, catalog, err := store.InspectWorkflows(ctx, installed.Slug)
	state := snapshot.Project().State()
	if err != nil || state.ID != installed.ID {
		return projectapp.ResolveActiveWorkflow(state, catalog, "recovery_required")
	}
	state = snapshot.Project().State()
	if snapshot.Revision() != resolved.Project.PortableRevision {
		return projectapp.ResolveActiveWorkflow(state, catalog, "project_configuration_drift")
	}
	return projectapp.ResolveActiveWorkflow(state, catalog, "")
}

// Catalog damage does not discard a strictly decoded portable Project. Layout
// or manifest failures still use the historical metadata reader's categories.
func (r activeWorkflowReader) ReadInstalledProject(ctx context.Context, installed projectapp.InstalledProject) (project.Project, error) {
	store, err := local.NewPortableStore(filepath.Dir(installed.Source))
	if err == nil && filepath.Base(installed.Source) == installed.Slug {
		snapshot, _, _ := store.InspectWorkflows(ctx, installed.Slug)
		state := snapshot.Project().State()
		if state.ID == installed.ID && state.Slug == installed.Slug {
			return snapshot.Project(), nil
		}
	}
	return r.s.installation.ReadInstalledProject(ctx, installed)
}
