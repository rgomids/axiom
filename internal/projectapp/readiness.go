package projectapp

import (
	"context"
	"sort"

	"github.com/rgomids/axiom/internal/project"
)

// Issue #231 canonical Project readiness. This file owns the only readiness
// algorithm: project validate, Work Item preflight and Execution preflight all
// consume ProjectReadiness.Evaluate or a projection of its report. Evaluation is
// read-only, performs no Provider call or Runtime dispatch, and grants no
// authority: readiness answers "can this Project satisfy the operation";
// authority independently answers "may this effect happen now".

type ReadinessOperation string

const (
	OperationWorkItem  ReadinessOperation = "work-item"
	OperationExecution ReadinessOperation = "execution"
)

// Stable blocker and warning codes (project-bootstrap-v3.md §4).
const (
	BlockerProjectNotInstalled      = "project_not_installed"
	BlockerProjectSourceUnavailable = "project_source_unavailable"
	BlockerProjectStateInvalid      = "project_state_invalid"
	BlockerInstallationStale        = "installation_stale"
	BlockerRecoveryRequired         = "recovery_required"
	BlockerRepositoryBindingMissing = "repository_binding_missing"
	BlockerRepositoryUnavailable    = "repository_unavailable"
	BlockerCapabilityMissing        = "capability_mapping_missing"
	BlockerCapabilityAmbiguous      = "capability_mapping_ambiguous"
	BlockerProviderUnsupported      = "provider_unsupported"
	BlockerRuntimePolicyUnavailable = "runtime_policy_unavailable"
	BlockerRuntimeResolution        = "runtime_resolution_blocked"

	// No credential-binding writer exists yet and the implemented Provider
	// (GitHub through gh) authenticates ambiently, so an unbound declared
	// credential reference is reported, never enforced (pre-#231 behavior).
	WarningCredentialUnbound        = "credential_binding_missing"
	WarningDocumentationUnbound     = "documentation_binding_missing"
	WarningDocumentationUnavailable = "documentation_unavailable"
	WarningDocumentationStale       = "documentation_stale"
	WarningDocumentationUnsafe      = "documentation_unsafe"
	WarningTechnologyAbsent         = "technology_context_absent"
	WarningBusinessContextAbsent    = "business_context_absent"
)

type DocumentationStatus string

const (
	DocumentationAvailable             DocumentationStatus = "available"
	DocumentationMissing               DocumentationStatus = "missing"
	DocumentationUnbound               DocumentationStatus = "unbound"
	DocumentationStale                 DocumentationStatus = "stale"
	DocumentationUnsafe                DocumentationStatus = "unsafe"
	DocumentationRepositoryUnavailable DocumentationStatus = "repository_unavailable"
)

type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

type RepositoryReadiness struct {
	Key          string `json:"key"`
	Binding      string `json:"binding"`
	Availability string `json:"availability"`
	Remote       string `json:"remote"`
}

type CapabilityReadinessView struct {
	CapabilityResolution
	Credential string `json:"credential"`
}

type RuntimeReadiness struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type DocumentationReadiness struct {
	Key    string              `json:"key"`
	Kind   string              `json:"kind"`
	Status DocumentationStatus `json:"status"`
}

type ContextReadiness struct {
	TechnologyFacts      int  `json:"technologyFacts"`
	DocumentationSources int  `json:"documentationSources"`
	SourceRefs           int  `json:"sourceRefs"`
	GlossaryEntries      int  `json:"glossaryEntries"`
	Documents            int  `json:"documents"`
	Policies             int  `json:"policies"`
	BusinessText         bool `json:"businessText"`
}

// AuthorityExpectations is constant disclosure: readiness never authorizes.
type AuthorityExpectations struct {
	GrantsAuthority bool   `json:"grantsAuthority"`
	LocalEffects    string `json:"localEffects"`
	ProviderEffects string `json:"providerEffects"`
}

var readinessAuthority = AuthorityExpectations{LocalEffects: "explicit_local_authorization", ProviderEffects: "explicit_external_authorization"}

type OperationReadiness struct {
	Operation ReadinessOperation `json:"operation"`
	Status    string             `json:"status"`
	Blockers  []Finding          `json:"blockers"`
}

// ReadinessReport is bounded and deterministic. It carries validated logical
// keys only: no local path, credential value, document content or raw error.
type ReadinessReport struct {
	ProjectID     string                    `json:"projectId,omitempty"`
	SchemaVersion int                       `json:"schemaVersion,omitempty"`
	Structure     string                    `json:"structure"`
	Effective     string                    `json:"effective"`
	Repositories  []RepositoryReadiness     `json:"repositories"`
	Capabilities  []CapabilityReadinessView `json:"capabilities"`
	Runtime       RuntimeReadiness          `json:"runtime"`
	Documentation []DocumentationReadiness  `json:"documentation"`
	Context       ContextReadiness          `json:"context"`
	Authority     AuthorityExpectations     `json:"authority"`
	Operations    []OperationReadiness      `json:"operations"`
	Blockers      []Finding                 `json:"blockers"`
	Warnings      []Finding                 `json:"warnings"`
}

// Operation returns the requirement-specific projection used before effects.
func (r ReadinessReport) Operation(operation ReadinessOperation) OperationReadiness {
	for _, current := range r.Operations {
		if current.Operation == operation {
			return current
		}
	}
	return OperationReadiness{Operation: operation, Status: "blocked", Blockers: []Finding{{Code: BlockerProjectStateInvalid}}}
}

func (o OperationReadiness) Ready() bool { return o.Status == "ready" }

// ReadinessSubject is one consistent observation of an installed Project.
type ReadinessSubject struct {
	Project project.Project
	Local   LocalRecordState
}

// ReadinessProjects loads the installed Project for an explicit UUID/slug
// selector only. It returns a stable structural blocker code on failure.
type ReadinessProjects interface {
	LoadReadiness(context.Context, string) (ReadinessSubject, string)
}

// RepositoryProbe observes one bound Repository location without following links.
type RepositoryProbe interface {
	RepositoryAvailable(context.Context, RepositoryBinding) bool
}

// DocumentationRequest names one portable source and its machine-local
// resolution inputs. RepositoryPath is empty when the Repository is unusable.
type DocumentationRequest struct {
	Source         project.DocumentationSource
	RepositoryPath string
	Binding        *DocumentationBinding
}

type DocumentationResolver interface {
	ResolveDocumentation(context.Context, DocumentationRequest) DocumentationStatus
}

// RuntimeProbe is the #140 availability projection (runtimeapplication).
type RuntimeProbe interface {
	Availability(context.Context, string) string
}

type ProjectReadiness struct {
	Projects      ReadinessProjects
	Repositories  RepositoryProbe
	Documentation DocumentationResolver
	Runtime       RuntimeProbe
	Catalog       ProviderCatalog
}

// RuntimeNotEvaluated marks a report whose Runtime dimension was outside the
// requested operation's requirement set.
const RuntimeNotEvaluated = "not_evaluated"

// Evaluate gathers read-only observations and applies EvaluateReadiness.
func (r ProjectReadiness) Evaluate(ctx context.Context, selector string) ReadinessReport {
	return r.evaluate(ctx, selector, true)
}

// EvaluateOperation is the requirement-specific projection used before
// effects: it observes only what the operation requires (the Work Item set
// does not observe Runtimes) and returns that operation's readiness.
func (r ProjectReadiness) EvaluateOperation(ctx context.Context, selector string, operation ReadinessOperation) (ReadinessReport, OperationReadiness) {
	report := r.evaluate(ctx, selector, operation == OperationExecution)
	return report, report.Operation(operation)
}

func (r ProjectReadiness) evaluate(ctx context.Context, selector string, runtime bool) ReadinessReport {
	if r.Projects == nil || r.Repositories == nil || r.Documentation == nil || r.Runtime == nil {
		return blockedReport(BlockerProjectStateInvalid)
	}
	subject, code := r.Projects.LoadReadiness(ctx, selector)
	if code != "" {
		return blockedReport(code)
	}
	state := subject.Project.State()
	input := ReadinessInput{Project: subject.Project, Local: subject.Local, Repositories: map[string]bool{}, Documentation: map[string]DocumentationStatus{}, Catalog: r.Catalog}
	paths := map[string]string{}
	for _, binding := range subject.Local.Repositories {
		available := r.Repositories.RepositoryAvailable(ctx, binding)
		input.Repositories[binding.RepositoryKey] = available
		if available {
			paths[binding.RepositoryKey] = binding.ExplicitPath
		}
	}
	bindings := map[string]*DocumentationBinding{}
	for i := range subject.Local.Documentation {
		bindings[subject.Local.Documentation[i].SourceKey] = &subject.Local.Documentation[i]
	}
	sources, _ := state.DocumentationSources.Value()
	for _, source := range sources {
		request := DocumentationRequest{Source: source, Binding: bindings[source.Key]}
		if ref, ok := source.RepositoryRef.Value(); ok {
			request.RepositoryPath = paths[ref]
		}
		input.Documentation[source.Key] = r.Documentation.ResolveDocumentation(ctx, request)
	}
	input.RuntimeCode = RuntimeNotEvaluated
	if runtime {
		// Missing portable intent needs no machine observation.
		input.RuntimeCode = "policy_unconfigured"
		if project.RuntimePolicyDeclared(state) {
			input.RuntimeCode = r.Runtime.Availability(ctx, state.ID)
		}
	}
	return EvaluateReadiness(input)
}

// ReadinessInput is fully observed input; EvaluateReadiness is pure.
type ReadinessInput struct {
	Project       project.Project
	Local         LocalRecordState
	Repositories  map[string]bool
	Documentation map[string]DocumentationStatus
	RuntimeCode   string
	Catalog       ProviderCatalog
}

func blockedReport(code string) ReadinessReport {
	report := emptyReport()
	report.Structure, report.Effective = "invalid", "blocked"
	if code == BlockerProjectNotInstalled || code == BlockerProjectSourceUnavailable || code == BlockerRecoveryRequired || code == BlockerInstallationStale {
		report.Structure = "unavailable"
	}
	report.Blockers = []Finding{{Code: code}}
	for _, operation := range []ReadinessOperation{OperationWorkItem, OperationExecution} {
		report.Operations = append(report.Operations, OperationReadiness{Operation: operation, Status: "blocked", Blockers: []Finding{{Code: code}}})
	}
	return report
}

func emptyReport() ReadinessReport {
	return ReadinessReport{Repositories: []RepositoryReadiness{}, Capabilities: []CapabilityReadinessView{}, Documentation: []DocumentationReadiness{}, Authority: readinessAuthority, Operations: []OperationReadiness{}, Blockers: []Finding{}, Warnings: []Finding{}, Runtime: RuntimeReadiness{Status: "blocked"}}
}

// EvaluateReadiness derives per-operation readiness from observed input.
func EvaluateReadiness(in ReadinessInput) ReadinessReport {
	if !in.Project.Equivalent(in.Project) {
		return blockedReport(BlockerProjectStateInvalid)
	}
	state := in.Project.State()
	report := emptyReport()
	report.ProjectID, report.SchemaVersion, report.Structure = state.ID, state.SchemaVersion, "valid"
	var shared, workItem, execution []Finding

	bound := map[string]bool{}
	for _, binding := range in.Local.Repositories {
		bound[binding.RepositoryKey] = true
	}
	repositories, _ := state.Repositories.Value()
	for _, repository := range repositories {
		view := RepositoryReadiness{Key: repository.Key, Binding: "bound", Availability: "available", Remote: "local_only"}
		if repository.Remote.Form() == project.Present {
			view.Remote = "declared"
		}
		switch {
		case !bound[repository.Key]:
			view.Binding, view.Availability = "missing", "unavailable"
			shared = append(shared, Finding{Code: BlockerRepositoryBindingMissing, Subject: repository.Key})
		case !in.Repositories[repository.Key]:
			view.Availability = "unavailable"
			shared = append(shared, Finding{Code: BlockerRepositoryUnavailable, Subject: repository.Key})
		}
		report.Repositories = append(report.Repositories, view)
	}

	capability := ResolveCapability(state, WorkItemCapability, in.Catalog)
	view := CapabilityReadinessView{CapabilityResolution: capability, Credential: "not_required"}
	switch capability.Readiness {
	case CapabilityMissing:
		workItem = append(workItem, Finding{Code: BlockerCapabilityMissing, Subject: WorkItemCapability})
	case CapabilityAmbiguous:
		workItem = append(workItem, Finding{Code: BlockerCapabilityAmbiguous, Subject: WorkItemCapability})
	case CapabilityUnsupported:
		workItem = append(workItem, Finding{Code: BlockerProviderUnsupported, Subject: WorkItemCapability})
	}
	if capability.CredentialRef != "" {
		view.Credential = "unbound"
		for _, credential := range in.Local.Credentials {
			if credential.ReferenceKey == capability.CredentialRef && credential.SourceKind != "" {
				view.Credential = "bound"
			}
		}
		if view.Credential == "unbound" {
			report.Warnings = append(report.Warnings, Finding{Code: WarningCredentialUnbound, Subject: capability.CredentialRef})
		}
	}
	report.Capabilities = append(report.Capabilities, view)

	runtimeCode := in.RuntimeCode
	if runtimeCode == "" && !project.RuntimePolicyDeclared(state) {
		runtimeCode = "policy_unconfigured"
	}
	switch runtimeCode {
	case "":
		report.Runtime = RuntimeReadiness{Status: "ready"}
	case RuntimeNotEvaluated:
		report.Runtime = RuntimeReadiness{Status: RuntimeNotEvaluated}
		execution = append(execution, Finding{Code: BlockerRuntimeResolution, Detail: RuntimeNotEvaluated})
	case "policy_unconfigured":
		report.Runtime = RuntimeReadiness{Status: "unavailable", Detail: runtimeCode}
		execution = append(execution, Finding{Code: BlockerRuntimePolicyUnavailable, Detail: runtimeCode})
	default:
		report.Runtime = RuntimeReadiness{Status: "blocked", Detail: runtimeCode}
		execution = append(execution, Finding{Code: BlockerRuntimeResolution, Detail: runtimeCode})
	}

	sources, _ := state.DocumentationSources.Value()
	for _, source := range sources {
		status, ok := in.Documentation[source.Key]
		if !ok {
			status = DocumentationMissing
		}
		report.Documentation = append(report.Documentation, DocumentationReadiness{Key: source.Key, Kind: source.Kind, Status: status})
		if code := documentationWarning(status); code != "" {
			report.Warnings = append(report.Warnings, Finding{Code: code, Subject: source.Key})
		}
	}
	report.Context = contextReadiness(state)
	if facts, ok := state.TechnologyContext.Value(); !ok || len(facts) == 0 {
		report.Warnings = append(report.Warnings, Finding{Code: WarningTechnologyAbsent})
	}
	if context, ok := state.BusinessContext.Value(); !ok || context.Text.Form() != project.Present {
		report.Warnings = append(report.Warnings, Finding{Code: WarningBusinessContextAbsent})
	}

	// Every current Execution is scoped to a linked Work Item, so Execution
	// requires the Work Item set plus the Runtime set.
	workItemBlockers := sortedFindings(append(append([]Finding{}, shared...), workItem...))
	executionBlockers := sortedFindings(append(append(append([]Finding{}, shared...), workItem...), execution...))
	report.Operations = []OperationReadiness{operationReadiness(OperationWorkItem, workItemBlockers), operationReadiness(OperationExecution, executionBlockers)}
	report.Blockers = sortedFindings(append(append(append([]Finding{}, shared...), workItem...), execution...))
	report.Warnings = sortedFindings(report.Warnings)
	ready := 0
	for _, operation := range report.Operations {
		if operation.Ready() {
			ready++
		}
	}
	switch {
	case ready == len(report.Operations):
		report.Effective = "ready"
	case ready > 0:
		report.Effective = "partial"
	default:
		report.Effective = "blocked"
	}
	return report
}

func operationReadiness(operation ReadinessOperation, blockers []Finding) OperationReadiness {
	status := "ready"
	if len(blockers) != 0 {
		status = "blocked"
	}
	return OperationReadiness{Operation: operation, Status: status, Blockers: blockers}
}

func documentationWarning(status DocumentationStatus) string {
	switch status {
	case DocumentationAvailable:
		return ""
	case DocumentationUnbound:
		return WarningDocumentationUnbound
	case DocumentationStale:
		return WarningDocumentationStale
	case DocumentationUnsafe:
		return WarningDocumentationUnsafe
	}
	return WarningDocumentationUnavailable
}

func contextReadiness(state project.State) ContextReadiness {
	facts, _ := state.TechnologyContext.Value()
	sources, _ := state.DocumentationSources.Value()
	policies, _ := state.Policies.Value()
	result := ContextReadiness{TechnologyFacts: len(facts), DocumentationSources: len(sources), Policies: len(policies)}
	if context, ok := state.BusinessContext.Value(); ok {
		refs, _ := context.SourceRefs.Value()
		glossary, _ := context.Glossary.Value()
		documents, _ := context.Documents.Value()
		result.SourceRefs, result.GlossaryEntries, result.Documents = len(refs), len(glossary), len(documents)
		result.BusinessText = context.Text.Form() == project.Present
	}
	return result
}

func sortedFindings(findings []Finding) []Finding {
	seen := map[Finding]bool{}
	result := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if !seen[finding] {
			seen[finding] = true
			result = append(result, finding)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Code != result[j].Code {
			return result[i].Code < result[j].Code
		}
		return result[i].Subject < result[j].Subject
	})
	return result
}
