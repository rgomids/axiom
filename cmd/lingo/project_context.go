package main

import (
	"context"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

func (s lifecycleService) projectContext() projectapp.ProjectContext {
	return projectapp.ProjectContext{Preferences: s.installation, Resolver: contextProjectResolver{s}}
}

type contextProjectResolver struct{ service lifecycleService }

func (r contextProjectResolver) ResolveContextProject(ctx context.Context, selector string) (string, string) {
	resolved := r.service.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return "", resolved.Category
	}
	portable, err := r.service.portable.InspectRecordedSource(ctx, resolved.Project.Source, resolved.Project.Slug)
	if err != nil || !portable.Exists {
		return "", "project_source_unavailable"
	}
	state := portable.Snapshot.Project().State()
	if state.ID != resolved.Project.ID || state.Slug != resolved.Project.Slug || portable.Snapshot.Revision() != resolved.Project.PortableRevision {
		return "", "stale_project_context"
	}
	return resolved.Project.ID, ""
}

func (s lifecycleService) EffectiveProject(ctx context.Context, selector string) (string, cli.Result) {
	view, category := s.projectContext().Effective(ctx, selector)
	return view.Effective, s.contextResult(view, category)
}

func (s lifecycleService) contextResult(view projectapp.EffectiveContext, category string) cli.Result {
	message, next := "Project context resolved", ""
	facts := completion.Facts{Completed: true}
	if category != "" {
		view.Issue = category
		facts = completion.Facts{ValidationFailed: true}
		message, next = "Project context could not be resolved: "+category, "Inspect project context; select a configured Project or clear the stale preference"
	}
	refs := []string{}
	if view.Effective != "" {
		refs = append(refs, "project:"+view.Effective)
	}
	result := canonicalCompletion(facts, message, refs, next, s.provenance)
	result.Category, result.Context = category, &view
	return result
}

func (s lifecycleService) ProjectContext(ctx context.Context, input cli.ProjectContextInput) cli.Result {
	contextService := s.projectContext()
	if input.Action != "show" {
		if !input.AuthorizeLocal {
			return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project context mutation requires local authority", nil, "Review the target and pass --authorize-local", s.provenance)
		}
		session := ""
		if strings.HasPrefix(input.Action, "session-") {
			session = projectapp.ProjectSession(ctx)
			if session == "" {
				return s.contextResult(projectapp.EffectiveContext{}, "project_session_required")
			}
		}
		category := ""
		if strings.HasSuffix(input.Action, "-set") {
			category = contextService.Set(ctx, session, input.Selector)
		} else {
			category = contextService.Clear(ctx, session)
		}
		if category != "" {
			return s.contextResult(projectapp.EffectiveContext{}, category)
		}
		// Successful clear may intentionally leave no effective Project. Report
		// the mutation as success without claiming resolution or acceptance.
		view, category := contextService.Inspect(ctx, "")
		view.Issue = category
		next := ""
		if category != "" && category != "project_context_unresolved" {
			next = "Inspect and repair the remaining Project context"
		}
		result := canonicalCompletion(completion.Facts{Completed: true}, "Project context preference updated", nil, next, s.provenance)
		result.Category, result.Context = "project_context_updated", &view
		return result
	}
	view, category := contextService.Inspect(ctx, input.Selector)
	return s.contextResult(view, category)
}
