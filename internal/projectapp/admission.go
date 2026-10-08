package projectapp

import (
	"context"
	"sort"
)

// Issue #230 central operational admission (I230-T05). This file owns the
// only operation classification and the only archive/disable admission
// decision; CLI and Runtime surfaces consume it through the same composition
// gate that applies #231 readiness. Admission answers "may this machine
// evolve this Project, or use this Integration, now". Readiness answers "can
// the Project satisfy the operation". Authority answers "may this exact
// effect happen". Admission reads machine-local state only, performs no
// Provider or Transport call and never grants authority.

type AdmissionClass string

const (
	AdmissionInspection     AdmissionClass = "inspection"
	AdmissionAdministrative AdmissionClass = "administrative"
	AdmissionEvolution      AdmissionClass = "evolution"
)

// AdmissionOperation is the canonical "<resource>.<operation>" name of one
// lifecycle operation, including its preview step: a preview exists only to
// authorize that operation, so it is admitted or denied with it.
type AdmissionOperation string

const (
	AdmitProjectList       AdmissionOperation = "project.list"
	AdmitProjectShow       AdmissionOperation = "project.show"
	AdmitProjectResolve    AdmissionOperation = "project.resolve"
	AdmitProjectValidate   AdmissionOperation = "project.validate"
	AdmitProjectConfigure  AdmissionOperation = "project.configure"
	AdmitProjectEdit       AdmissionOperation = "project.edit"
	AdmitProjectArchive    AdmissionOperation = "project.archive"
	AdmitProjectReactivate AdmissionOperation = "project.reactivate"

	AdmitRepositoryList   AdmissionOperation = "repository.list"
	AdmitRepositoryShow   AdmissionOperation = "repository.show"
	AdmitRepositoryAttach AdmissionOperation = "repository.attach"
	AdmitRepositoryUpdate AdmissionOperation = "repository.update"
	AdmitRepositoryDetach AdmissionOperation = "repository.detach"

	AdmitIntegrationList     AdmissionOperation = "integration.list"
	AdmitIntegrationShow     AdmissionOperation = "integration.show"
	AdmitIntegrationValidate AdmissionOperation = "integration.validate"
	AdmitIntegrationUpdate   AdmissionOperation = "integration.update"
	AdmitIntegrationDisable  AdmissionOperation = "integration.disable"
	AdmitIntegrationEnable   AdmissionOperation = "integration.enable"
	AdmitIntegrationRemove   AdmissionOperation = "integration.remove"

	AdmitWorkItemList     AdmissionOperation = "work-item.list"
	AdmitWorkItemShow     AdmissionOperation = "work-item.show"
	AdmitWorkItemCreate   AdmissionOperation = "work-item.create"
	AdmitWorkItemSelect   AdmissionOperation = "work-item.select"
	AdmitWorkItemUpdate   AdmissionOperation = "work-item.update"
	AdmitWorkItemComment  AdmissionOperation = "work-item.comment"
	AdmitWorkItemClose    AdmissionOperation = "work-item.close"
	AdmitWorkItemReopen   AdmissionOperation = "work-item.reopen"
	AdmitWorkItemComplete AdmissionOperation = "work-item.complete"

	AdmitExecutionList      AdmissionOperation = "execution.list"
	AdmitExecutionStatus    AdmissionOperation = "execution.status"
	AdmitExecutionEvidence  AdmissionOperation = "execution.evidence"
	AdmitExecutionStart     AdmissionOperation = "execution.start"
	AdmitExecutionAdvance   AdmissionOperation = "execution.advance"
	AdmitExecutionFact      AdmissionOperation = "execution.fact"
	AdmitExecutionResume    AdmissionOperation = "execution.resume"
	AdmitExecutionReconcile AdmissionOperation = "execution.reconcile"
)

// AdmissionRule classifies one operation. Capability names the Project
// capability whose Integration the operation uses for Provider/Transport
// effects, reads included; it is empty for local-only operations. Readiness
// is the #231 requirement set enforced by the same gate, when any.
type AdmissionRule struct {
	Class      AdmissionClass
	Capability string
	Readiness  ReadinessOperation
}

var admissionRules = map[AdmissionOperation]AdmissionRule{
	AdmitProjectList:       {Class: AdmissionInspection},
	AdmitProjectShow:       {Class: AdmissionInspection},
	AdmitProjectResolve:    {Class: AdmissionInspection},
	AdmitProjectValidate:   {Class: AdmissionInspection},
	AdmitProjectConfigure:  {Class: AdmissionAdministrative},
	AdmitProjectEdit:       {Class: AdmissionAdministrative},
	AdmitProjectArchive:    {Class: AdmissionAdministrative},
	AdmitProjectReactivate: {Class: AdmissionAdministrative},

	AdmitRepositoryList:   {Class: AdmissionInspection},
	AdmitRepositoryShow:   {Class: AdmissionInspection},
	AdmitRepositoryAttach: {Class: AdmissionAdministrative},
	AdmitRepositoryUpdate: {Class: AdmissionAdministrative},
	AdmitRepositoryDetach: {Class: AdmissionAdministrative},

	AdmitIntegrationList:     {Class: AdmissionInspection},
	AdmitIntegrationShow:     {Class: AdmissionInspection},
	AdmitIntegrationValidate: {Class: AdmissionInspection},
	AdmitIntegrationUpdate:   {Class: AdmissionAdministrative},
	AdmitIntegrationDisable:  {Class: AdmissionAdministrative},
	AdmitIntegrationEnable:   {Class: AdmissionAdministrative},
	AdmitIntegrationRemove:   {Class: AdmissionAdministrative},

	AdmitWorkItemList:     {Class: AdmissionInspection},
	AdmitWorkItemShow:     {Class: AdmissionInspection, Readiness: OperationWorkItem},
	AdmitWorkItemCreate:   {Class: AdmissionEvolution, Capability: WorkItemCapability, Readiness: OperationWorkItem},
	AdmitWorkItemSelect:   {Class: AdmissionEvolution, Capability: WorkItemCapability, Readiness: OperationWorkItem},
	AdmitWorkItemUpdate:   {Class: AdmissionEvolution, Capability: WorkItemCapability, Readiness: OperationWorkItem},
	AdmitWorkItemComment:  {Class: AdmissionEvolution, Capability: WorkItemCapability, Readiness: OperationWorkItem},
	AdmitWorkItemClose:    {Class: AdmissionEvolution, Capability: WorkItemCapability, Readiness: OperationWorkItem},
	AdmitWorkItemReopen:   {Class: AdmissionEvolution, Capability: WorkItemCapability, Readiness: OperationWorkItem},
	AdmitWorkItemComplete: {Class: AdmissionEvolution, Capability: WorkItemCapability, Readiness: OperationWorkItem},

	AdmitExecutionList:     {Class: AdmissionInspection},
	AdmitExecutionStatus:   {Class: AdmissionInspection},
	AdmitExecutionEvidence: {Class: AdmissionInspection},
	// Start enforces the shared and Work Item readiness requirements; its
	// Runtime requirement is the #140 reviewed Runtime preview.
	AdmitExecutionStart:     {Class: AdmissionEvolution, Readiness: OperationWorkItem},
	AdmitExecutionAdvance:   {Class: AdmissionEvolution},
	AdmitExecutionFact:      {Class: AdmissionEvolution},
	AdmitExecutionResume:    {Class: AdmissionEvolution, Readiness: OperationExecution},
	AdmitExecutionReconcile: {Class: AdmissionEvolution, Capability: WorkItemCapability},
}

// AdmissionRuleFor returns the classification of one operation.
func AdmissionRuleFor(operation AdmissionOperation) (AdmissionRule, bool) {
	rule, ok := admissionRules[operation]
	return rule, ok
}

// AdmissionOperations lists every classified operation in canonical order.
func AdmissionOperations() []AdmissionOperation {
	operations := make([]AdmissionOperation, 0, len(admissionRules))
	for operation := range admissionRules {
		operations = append(operations, operation)
	}
	sort.Slice(operations, func(i, j int) bool { return operations[i] < operations[j] })
	return operations
}

// Stable admission denial categories.
const (
	AdmissionProjectArchived       = "project_archived"
	AdmissionIntegrationDisabled   = "integration_disabled"
	AdmissionOperationUnclassified = "operation_unclassified"
)

// AdmissionDecision is the canonical, bounded admission outcome. Deferred
// carries a failure that a later stage of the same gate reports before any
// effect: the operation's own exact selection, or its readiness requirement.
type AdmissionDecision struct {
	Operation       AdmissionOperation `json:"operation"`
	Class           AdmissionClass     `json:"class,omitempty"`
	Allowed         bool               `json:"allowed"`
	Code            string             `json:"code,omitempty"`
	ProjectID       string             `json:"projectId,omitempty"`
	Integration     string             `json:"integration,omitempty"`
	Deferred        string             `json:"-"`
	GrantsAuthority bool               `json:"grantsAuthority"`
}

// AdmissionSelector resolves an exact UUID/slug selector to one installed
// Project ID through the same protected selection every operation uses. It
// reads no portable configuration.
type AdmissionSelector interface {
	SelectProjectID(context.Context, string) (string, string)
}

// AdmissionGuard composes the read-only observations admission needs.
type AdmissionGuard struct {
	Selection   AdmissionSelector
	Projects    ReadinessProjects
	Operational OperationalStore
	Catalog     ProviderCatalog
}

// Admit returns the decision for one operation on one selected Project.
// Inspection and administrative maintenance stay available while archived
// or disabled; only an operation that uses a disabled Integration's
// Provider/Transport is denied for that reason. Operational evolution is
// denied while archived. Unknown, unsafe or interrupted operational state
// fails closed for every operation that consults it.
func (g AdmissionGuard) Admit(ctx context.Context, selector string, operation AdmissionOperation) AdmissionDecision {
	rule, ok := admissionRules[operation]
	if !ok {
		return AdmissionDecision{Operation: operation, Code: AdmissionOperationUnclassified}
	}
	decision := AdmissionDecision{Operation: operation, Class: rule.Class}
	if rule.Class == AdmissionAdministrative || rule.Class == AdmissionInspection && rule.Capability == "" {
		decision.Allowed = true
		return decision
	}
	if g.Selection == nil || g.Operational == nil || rule.Capability != "" && g.Projects == nil {
		decision.Code = OperationalStorageFailure
		return decision
	}
	projectID, category := g.Selection.SelectProjectID(ctx, selector)
	if category != "" {
		decision.Allowed, decision.Deferred = true, category
		return decision
	}
	decision.ProjectID = projectID
	observation, category := InspectOperational(ctx, g.Operational, projectID)
	if category != "" {
		decision.Code = category
		return decision
	}
	if rule.Class == AdmissionEvolution && observation.State.Archived() {
		decision.Code = AdmissionProjectArchived
		return decision
	}
	if rule.Capability != "" {
		// The Integration actually providing the capability is the one whose
		// local eligibility applies. When it cannot be attributed, no Provider
		// call is admitted: an operation with a readiness requirement reports
		// the same code through that gate (#231 preflight); one without it is
		// denied here.
		unattributed := func(code string) AdmissionDecision {
			if rule.Readiness != "" {
				decision.Allowed, decision.Deferred = true, code
			} else {
				decision.Code = code
			}
			return decision
		}
		subject, code := g.Projects.LoadReadiness(ctx, selector)
		if code != "" {
			return unattributed(code)
		}
		state := subject.Project.State()
		if state.ID != projectID {
			return unattributed(BlockerProjectStateInvalid)
		}
		resolution := ResolveCapability(state, rule.Capability, g.Catalog)
		switch resolution.Readiness {
		case CapabilityReady:
		case CapabilityAmbiguous:
			return unattributed(BlockerCapabilityAmbiguous)
		case CapabilityUnsupported:
			return unattributed(BlockerProviderUnsupported)
		default:
			return unattributed(BlockerCapabilityMissing)
		}
		decision.Integration = resolution.Integration
		if observation.State.IntegrationDisabled(resolution.Integration) {
			decision.Code = AdmissionIntegrationDisabled
			return decision
		}
	}
	decision.Allowed = true
	return decision
}
