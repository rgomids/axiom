package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
)

// Issue #230 (I230-T03) composition of Project archive/reactivate and Project
// discovery. Archive and reactivate are machine-local: they select the exact
// installed Project through protected local state (never the working
// directory), pass the central admission gate and delegate preview, no-op,
// authority and CAS to projectapp.ApplyOperational. They never touch portable
// intent, the installation record, Repositories or the S7 archive namespace.

var _ cli.ProjectLifecycleService = lifecycleService{}

func (s lifecycleService) ProjectArchive(ctx context.Context, input cli.ProjectLifecycleInput) cli.Result {
	return s.projectTransition(ctx, input, projectapp.AdmitProjectArchive, projectapp.ArchiveProject, "Project archive")
}

func (s lifecycleService) ProjectReactivate(ctx context.Context, input cli.ProjectLifecycleInput) cli.Result {
	return s.projectTransition(ctx, input, projectapp.AdmitProjectReactivate, projectapp.ReactivateProject, "Project reactivation")
}

func (s lifecycleService) projectTransition(ctx context.Context, input cli.ProjectLifecycleInput, admission projectapp.AdmissionOperation, operation projectapp.OperationalOperation, noun string) cli.Result {
	selected := s.installation.Select(ctx, input.Project)
	if selected.Status != local.ResolutionFound {
		return s.operationalSelectionFailure(selected.Category)
	}
	if denied := s.gate(ctx, input.Project, admission); denied != nil {
		return *denied
	}
	return s.applyOperational(ctx, projectapp.OperationalRequest{ProjectID: selected.Project.ID, Operation: operation}, input.PreviewDigest, input.AuthorizeLocal, noun)
}

func (s lifecycleService) operationalSelectionFailure(category string) cli.Result {
	var result cli.Result
	switch category {
	case "project_not_found":
		category = projectapp.OperationalNotInstalled
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project is not installed on this machine", nil, "Select an installed Project by UUID or slug", s.provenance)
	case "project_ambiguous":
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project selector is ambiguous", nil, "Provide the exact Project UUID", s.provenance)
	case "invalid_project_selector":
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project selector is invalid", nil, "Provide a valid Project UUID or slug", s.provenance)
	case "invalid_existing_local_state":
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Project state is invalid", nil, "Inspect preserved local state; it is never overwritten", s.provenance)
	case "recovery_required":
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Project state requires recovery", nil, "Run recovery inspect and apply the reviewed plan, then retry", s.provenance)
	case "cancelled":
		result = canonicalCompletion(completion.Facts{WasInterrupted: true}, "Project selection was cancelled", nil, "Retry the request", s.provenance)
	default:
		category = projectapp.OperationalStorageFailure
		result = canonicalCompletion(completion.Facts{Failed: true}, "Project selection failed", nil, "Inspect local storage before retrying", s.provenance)
	}
	result.Category = category
	return result
}

// ProjectListFiltered is project list with explicit options.
func (s lifecycleService) ProjectListFiltered(ctx context.Context, input cli.ProjectListInput) cli.Result {
	return s.listProjects(ctx, projectapp.ProjectListOptions{IncludeArchived: input.IncludeArchived})
}

func (s lifecycleService) listProjects(ctx context.Context, options projectapp.ProjectListOptions) cli.Result {
	result := s.projectCatalog.ListProjects(ctx, options)
	switch result.Status {
	case projectapp.ProjectListSucceeded:
		message := "Configured Projects listed"
		if len(result.Projects) == 0 {
			message = "No configured Projects"
		}
		response := canonicalCompletion(completion.Facts{Completed: true}, message, nil, "", s.provenance)
		response.Projects = make([]cli.ProjectListView, 0, len(result.Projects))
		for _, configured := range result.Projects {
			response.Projects = append(response.Projects, cli.ProjectListView{ID: configured.ID, Slug: configured.Slug, Name: configured.Name, Status: string(configured.Status)})
		}
		return response
	case projectapp.ProjectListCancelled:
		return canonicalCompletion(completion.Facts{WasInterrupted: true}, "Project listing was interrupted", nil, "Retry Project listing", s.provenance)
	default:
		if result.Category == "invalid_existing_local_state" || result.Category == "invalid_project_state" {
			return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Configured Project state is invalid", nil, "Repair protected Project state before retrying listing", s.provenance)
		}
		if result.Category == "recovery_required" {
			return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Configured Project state requires recovery", nil, "Review preserved local recovery state before retrying listing", s.provenance)
		}
		return canonicalCompletion(completion.Facts{Failed: true}, "Project listing failed", nil, "Inspect local storage and application availability before retrying", s.provenance)
	}
}

// showProject is read-only inspection. It uses the exact selection without
// per-Repository validation and reports each Repository's availability by the
// readiness rule, so one broken binding never hides the Project. It stays
// available while the Project is archived.
func (s lifecycleService) showProject(ctx context.Context, selector string) cli.Result {
	selected := s.installation.SelectForInspection(ctx, selector)
	if selected.Status != local.ResolutionFound {
		return projectShowFailure(selected.Category, s.provenance)
	}
	state, category := projectapp.InspectProjectState(ctx, s.operational, selected.Project.ID)
	if category != "" {
		return projectShowFailure(category, s.provenance)
	}
	references := []string{"project:" + selected.Project.ID}
	view := projectView(selected.Project)
	probe := local.RepositoryProbe{}
	unavailable := false
	for index, repository := range selected.Project.Repositories {
		references = append(references, "repository:"+repository.Key)
		view.Repositories[index].Availability = "available"
		if !probe.RepositoryAvailable(ctx, projectapp.RepositoryBinding{RepositoryKey: repository.Key, ExplicitPath: repository.Path, CanonicalIdentity: repository.CanonicalIdentity}) {
			view.Repositories[index].Availability = "unavailable"
			unavailable = true
		}
	}
	view.State = projectStateView(state)
	next := ""
	switch {
	case state.Status == projectapp.ProjectInvalid || state.Status == projectapp.ProjectRecoveryRequired:
		next = "Inspect preserved local operational state; it is never treated as active"
	case unavailable:
		next = "Restore the unavailable Repository binding, or repair it with project configure --project"
	}
	response := canonicalCompletion(completion.Facts{Completed: true}, "Project resolved", references, next, s.provenance)
	response.Project = view
	return response
}

func projectStateView(state projectapp.ProjectLocalState) *cli.ProjectStateView {
	view := &cli.ProjectStateView{Status: string(state.Status)}
	if state.Operational != nil {
		operational := projectapp.ViewOperational(*state.Operational)
		view.Operational = &operational
	}
	return view
}

// ProjectValidateInstalled validates the recorded source of one installed
// Project selected by UUID or slug: the canonical #231 readiness report, which
// reads only the source the protected installation record names, plus the
// machine-local operational status. It is read-only and admitted while archived.
func (s lifecycleService) ProjectValidateInstalled(ctx context.Context, input cli.ProjectLifecycleInput) cli.Result {
	if ctx.Err() != nil {
		return canonicalCompletion(completion.Facts{WasInterrupted: true}, "Project validation was interrupted", nil, "Retry Project validation", s.provenance)
	}
	report := s.readiness().Evaluate(ctx, input.Project)
	var state *cli.ProjectStateView
	var references []string
	if selected := s.installation.Select(ctx, input.Project); selected.Status == local.ResolutionFound {
		references = []string{"project:" + selected.Project.ID}
		if observed, category := projectapp.InspectProjectState(ctx, s.operational, selected.Project.ID); category == "" {
			state = projectStateView(observed)
		}
	}
	var response cli.Result
	if report.Structure == "valid" {
		next := ""
		if state != nil && (state.Status == string(projectapp.ProjectInvalid) || state.Status == string(projectapp.ProjectRecoveryRequired)) {
			next = "Inspect preserved local operational state; it is never treated as active"
		}
		response = canonicalCompletion(completion.Facts{Completed: true}, "Project is valid", references, next, s.provenance)
	} else {
		message, next := "Project state is invalid", "Correct Project state and retry validation"
		code := ""
		if len(report.Blockers) != 0 {
			code = report.Blockers[0].Code
		}
		switch code {
		case projectapp.BlockerProjectNotInstalled:
			message, next = "Project is not installed on this machine", "Select an installed Project by UUID or slug"
		case projectapp.BlockerProjectSourceUnavailable:
			message, next = "Project source is unavailable", "Restore the recorded Project source and retry validation"
		case projectapp.BlockerRecoveryRequired:
			message, next = "Project state requires recovery", "Run recovery inspect and apply the reviewed plan, then retry"
		case projectapp.BlockerInstallationStale:
			message, next = "Installation record does not match the recorded Project source", "Reinstall the edited Project source, or republish it through project configure --project"
		}
		response = canonicalCompletion(completion.Facts{ValidationFailed: true}, message, references, next, s.provenance)
	}
	response.Readiness = &report
	response.ProjectState = state
	return response
}

// explicitEditRequired refuses the legacy by-slug update of an installed
// Project: it would write portable state and leave the installation record
// stale. The category is also carried as the completion detail so it is on the
// wire.
func (s lifecycleService) explicitEditRequired() cli.Result {
	result, ok := completionWithDetails(completion.Facts{ValidationFailed: true}, "Installed Project cannot be updated by slug", "Use project configure --project <slug> --name <name> to preview an explicit edit", projectapp.ExplicitEditRequired, s.provenance)
	if !ok {
		return cli.Result{Status: cli.Failed, Category: projectapp.ExplicitEditRequired}
	}
	result.Category = projectapp.ExplicitEditRequired
	return result
}

func completionWithDetails(facts completion.Facts, message, next, details string, source provenance.Value) (cli.Result, bool) {
	statement, err := provenance.NewText(message, provenance.AxiomAuthored)
	if err != nil {
		return cli.Result{}, false
	}
	nextAction, err := provenance.NewText(next, provenance.AxiomAuthored)
	if err != nil {
		return cli.Result{}, false
	}
	canonical, err := completion.New(facts, statement, nil, nextAction, details, source)
	if err != nil {
		return cli.Result{}, false
	}
	return cli.Result{Completion: &canonical}, true
}
