package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T04 composition of the Integration lifecycle. Every method
// passes the central admission gate first; inventory, validation and key
// resolution belong to projectapp, local transitions to ApplyOperational and
// portable removal to the single Project edit seam. Nothing here calls a
// Provider or Transport, resolves a credential, or touches Runtime/MCP
// configuration (C230-02).

var _ cli.IntegrationService = lifecycleService{}

func (s lifecycleService) integrations() projectapp.IntegrationInspector {
	projects := local.ReadinessProjects{Installation: s.installation, Portable: s.portable}
	return projectapp.IntegrationInspector{Selection: projects, Projects: projects, Operational: s.operational, Catalog: projectapp.SupportedProviders}
}

func (s lifecycleService) IntegrationList(ctx context.Context, input cli.IntegrationInput) cli.Result {
	if denied := s.gate(ctx, input.Project, projectapp.AdmitIntegrationList); denied != nil {
		return *denied
	}
	report, category := s.integrations().List(ctx, input.Project)
	if category != "" {
		return s.integrationFailure(category)
	}
	message := "Project Integrations listed"
	if len(report.Integrations) == 0 {
		message = "Project declares no Integrations"
	}
	return s.integrationReport(report, completion.Facts{Completed: true}, message, "")
}

func (s lifecycleService) IntegrationShow(ctx context.Context, input cli.IntegrationInput) cli.Result {
	if denied := s.gate(ctx, input.Project, projectapp.AdmitIntegrationShow); denied != nil {
		return *denied
	}
	report, category := s.integrations().Show(ctx, input.Project, input.Integration)
	if category != "" {
		return s.integrationFailure(category)
	}
	return s.integrationReport(report, completion.Facts{Completed: true}, "Integration resolved", "")
}

func (s lifecycleService) IntegrationValidate(ctx context.Context, input cli.IntegrationInput) cli.Result {
	if denied := s.gate(ctx, input.Project, projectapp.AdmitIntegrationValidate); denied != nil {
		return *denied
	}
	report, category := s.integrations().Validate(ctx, input.Project, input.Integration)
	if category != "" {
		return s.integrationFailure(category)
	}
	if report.Validation.Status != projectapp.IntegrationValid {
		return s.integrationReport(report, completion.Facts{ValidationFailed: true}, "Integration declarations are invalid", "Correct the reported Integration declarations; validation made no Provider call")
	}
	return s.integrationReport(report, completion.Facts{Completed: true}, "Integration declarations are valid", "")
}

func (s lifecycleService) IntegrationDisable(ctx context.Context, input cli.IntegrationInput) cli.Result {
	return s.integrationTransition(ctx, input, projectapp.AdmitIntegrationDisable, projectapp.DisableIntegration, "Integration disable")
}

func (s lifecycleService) IntegrationEnable(ctx context.Context, input cli.IntegrationInput) cli.Result {
	return s.integrationTransition(ctx, input, projectapp.AdmitIntegrationEnable, projectapp.EnableIntegration, "Integration enable")
}

func (s lifecycleService) integrationTransition(ctx context.Context, input cli.IntegrationInput, admission projectapp.AdmissionOperation, operation projectapp.OperationalOperation, noun string) cli.Result {
	if denied := s.gate(ctx, input.Project, admission); denied != nil {
		return *denied
	}
	projectID, category := s.integrations().Target(ctx, input.Project, operation, input.Integration)
	if category != "" {
		return s.integrationFailure(category)
	}
	request := projectapp.OperationalRequest{ProjectID: projectID, Operation: operation, Integration: input.Integration}
	return s.applyOperational(ctx, request, input.PreviewDigest, input.AuthorizeLocal, noun)
}

// IntegrationRemove is the portable declaration removal. It is a Project edit:
// preview and replay are owned by projectEdit, which never writes operational
// state, credentials, Runtime/MCP configuration or Provider resources.
func (s lifecycleService) IntegrationRemove(ctx context.Context, input cli.IntegrationInput) cli.Result {
	if denied := s.gate(ctx, input.Project, projectapp.AdmitIntegrationRemove); denied != nil {
		return *denied
	}
	intent := projectapp.EditIntent{Selector: input.Project, IntegrationRemovals: []string{input.Integration}}
	return s.projectEdit(ctx, intent, editReplay{ProjectID: input.ProjectID, PreviewDigest: input.PreviewDigest, AuthorizeLocal: input.AuthorizeLocal})
}

func (s lifecycleService) integrationReport(report projectapp.IntegrationReport, facts completion.Facts, message, next string) cli.Result {
	references := []string{"project:" + report.ProjectID}
	for _, view := range report.Integrations {
		references = append(references, "integration:"+view.Key)
	}
	result := canonicalCompletion(facts, message, references, next, s.provenance)
	result.Integrations = &report
	return result
}

// integrationFailure renders every refusal of the read-only Integration
// operations with its stable category and no rejected input.
func (s lifecycleService) integrationFailure(category string) cli.Result {
	var result cli.Result
	switch category {
	case projectapp.IntegrationNotFound:
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Integration is not declared by this Project", nil, "Run integration list to see the declared Integration keys", s.provenance)
	case projectapp.OperationalInvalidInput:
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Integration input is invalid", nil, "Provide an exact Project selector and a portable Integration key", s.provenance)
	case projectapp.BlockerProjectNotInstalled:
		result = projectShowFailure("project_not_found", s.provenance)
	case projectapp.BlockerProjectSourceUnavailable:
		result = projectShowFailure("project_source_unavailable", s.provenance)
	case projectapp.BlockerRecoveryRequired:
		result = projectShowFailure("recovery_required", s.provenance)
	case projectapp.BlockerProjectStateInvalid, projectapp.BlockerInstallationStale:
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project state is stale or invalid", nil, "Run project validate, then reinstall or repair the Project before retrying", s.provenance)
	case projectapp.OperationalStateInvalid:
		result = canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Project operational state is invalid", nil, "Inspect preserved local state; it is never treated as absent", s.provenance)
	default:
		result = projectShowFailure(category, s.provenance)
	}
	result.Category = category
	return result
}
