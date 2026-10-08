package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 composition of the central admission guard. gate is the single
// pre-effect gate of every lifecycle operation: #230 admission first, then
// the #231 readiness requirement the same operation declares. No admission
// rule lives here; the classification and decision belong to projectapp.

func (s lifecycleService) admission() projectapp.AdmissionGuard {
	projects := local.ReadinessProjects{Installation: s.installation, Portable: s.portable}
	return projectapp.AdmissionGuard{Selection: projects, Projects: projects, Operational: s.operational, Catalog: projectapp.SupportedProviders}
}

func (s lifecycleService) gate(ctx context.Context, selector string, operation projectapp.AdmissionOperation) *cli.Result {
	decision := s.admission().Admit(ctx, selector, operation)
	if !decision.Allowed {
		result := s.admissionDenied(decision)
		return &result
	}
	if rule, _ := projectapp.AdmissionRuleFor(operation); rule.Readiness != "" {
		return s.preflight(ctx, selector, rule.Readiness)
	}
	return nil
}

func (s lifecycleService) admissionDenied(decision projectapp.AdmissionDecision) cli.Result {
	message, next := "Operation is not admitted for this Project", "Inspect the Project with project show and project validate, then retry"
	switch decision.Code {
	case projectapp.AdmissionProjectArchived:
		message, next = "Project is archived on this machine; operational work is blocked", "Reactivate the Project with project reactivate before operational work; inspection and administration stay available"
	case projectapp.AdmissionIntegrationDisabled:
		message, next = "Integration is disabled on this machine; it cannot be used", "Enable the Integration with integration enable before using it; inspection and administration stay available"
	case projectapp.OperationalStateInvalid:
		message, next = "Local Project operational state is invalid", "Inspect preserved local state; it is never treated as absent"
	case projectapp.OperationalRecoveryRequired:
		message, next = "Local Project operational state requires recovery", "Run recovery inspect and apply the reviewed recovery plan, then retry"
	case projectapp.BlockerCapabilityMissing, projectapp.BlockerCapabilityAmbiguous, projectapp.BlockerProviderUnsupported:
		message, next = "No eligible Integration provides the required capability", "Run project validate and configure exactly one supported Integration for the capability"
	}
	references := []string(nil)
	if decision.ProjectID != "" {
		references = []string{"project:" + decision.ProjectID}
	}
	result := canonicalCompletion(completion.Facts{ValidationFailed: true}, message, references, next, s.provenance)
	result.Category = decision.Code
	result.Admission = &decision
	return result
}
