// Package cli is Lingo's presentation boundary for Project lifecycle actions.
package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"io"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

const (
	ExitSuccess   = 0
	ExitFailure   = 1
	ExitCancelled = 2
)

// Service is deliberately operation-shaped. Presentation can request a lifecycle
// action but cannot issue a generic command or a raw filesystem mutation.
type Service interface {
	Init(context.Context, InitInput) Result
	Validate(context.Context, ProjectInput) Result
	Reopen(context.Context, ProjectInput) Result
	Update(context.Context, UpdateInput) Result
	Install(context.Context, InstallInput) Result
	RuntimeCodexInstall(context.Context) Result
	RuntimeCodexStatus(context.Context) Result
	RuntimeClaudeInstall(context.Context) Result
	RuntimeClaudeStatus(context.Context) Result
	FirstRun(context.Context) Result
	Resolve(context.Context, ResolveInput) Result
	Show(context.Context, ResolveInput) Result
	List(context.Context) Result
	Configure(context.Context, ConfigureInput) Result
	WorkItemCreate(context.Context, WorkItemInput) Result
	WorkItemSelect(context.Context, WorkItemInput) Result
	WorkItemShow(context.Context, WorkItemInput) Result
	WorkItemComment(context.Context, WorkItemInput) Result
	WorkItemComplete(context.Context, WorkItemInput) Result
	WorkflowStart(context.Context, WorkflowInput) Result
	WorkflowAdvance(context.Context, WorkflowInput) Result
	WorkflowResume(context.Context, WorkflowInput) Result
	WorkflowStatus(context.Context, WorkflowInput) Result
	WorkflowEvidence(context.Context, WorkflowInput) Result
	WorkflowReconcile(context.Context, WorkflowInput) Result
}

// RuntimeAuthService is optional: the read-only CLI subscription
// authentication preflight of one Runtime (#272).
type RuntimeAuthService interface {
	RuntimeAuth(context.Context, string) Result
}

// RuntimeProfileService is optional so existing presentation services and mocks
// need not implement runtime profile validation.
type RuntimeProfileService interface {
	RuntimeProfileValidate(context.Context) Result
}

const runtimeProfileValidateAction action = "runtime_profile_validate"

type InitInput struct {
	Slug string
	Name string
}

type ProjectInput struct{ Slug string }

type UpdateInput struct {
	Slug string
	Name string
}
type InstallInput struct {
	Source       string
	Repositories []RepositoryInput
}
type ResolveInput struct{ Selector string }
type RepositoryInput struct{ Key, Path string }
type ConfigureInput struct {
	ProjectID, Slug, Name string
	Repositories          []RepositoryInput
	WorkItemProvider      string
	PreviewDigest         string
	AuthorizeLocal        bool
	// Project is the EDIT selector; its presence selects EDIT and its absence
	// keeps CREATE. Supplied flags retain omission so application preserves
	// existing values; presentation never merges Project state.
	Project                  string
	NameSupplied             bool
	WorkItemProviderSupplied bool
	RemoveWorkItemProvider   bool
	RemoveRepositories       []string
	// Issue #231 CREATE bootstrap intent; syntax is split here, every rule
	// (ambiguity, local candidates, bounds) belongs to the application.
	RepositoryRemotes  map[string]string
	Runtimes           []string
	ModelProfiles      []string
	RuntimePreferences []RuntimePreferenceInput
	Technology         []KeyValueInput
	RemoveTechnology   []string
	Documentation      []DocumentationInput
	BusinessContext    string
	ContextSources     []string
	Glossary           []GlossaryInput
}
type RuntimePreferenceInput struct{ Role, Complexity, ModelProfile string }
type KeyValueInput struct{ Key, Value string }
type DocumentationInput struct{ Key, Kind, Repository, Path string }
type GlossaryInput struct{ Key, Term, Definition string }
type WorkItemInput struct {
	Type, Beneficiary, Value                 string
	Classification                           []string
	ElaboratedSections                       map[string]string
	Project, Repository, WorkItem            string
	Provider, ProviderRepository, ExternalID string
	Intent, Problem, DesiredOutcome, Context string
	Scope, Constraints, NonGoals, Acceptance string
	Message, PreviewDigest                   string
	Title                                    string
	Number                                   int
	AuthorizeExternal, AuthorizeLocal        bool
	Cancelled                                bool
}
type WorkflowInput struct {
	Automatic                                                   bool
	Project, Repository, WorkItem, Provider, ProviderRepository string
	ExternalID, Execution, Gate, Outcome, Reference, Next       string
	Fact                                                        string
	Runtime                                                     string
	Role, Complexity, RuntimePreview                            string
	Capabilities                                                []string
	Number                                                      int
	ExpectedRevision                                            uint64
	PreviewDigest                                               string
	AuthorizeExternal, AuthorizeLocal                           bool
	Active                                                      bool
}

type LifecycleFactService interface {
	WorkflowFact(context.Context, WorkflowInput) Result
}

type Status string

const (
	Succeeded Status = "success"
	Failed    Status = "error"
	Cancelled Status = "cancelled"
)

// Result only permits fixed, presentation-safe categories. Application adapters
// must not return raw paths, input values, parser output, or operating-system
// error strings for this surface.
type Result struct {
	Status            Status
	Category          string
	Project           *ProjectView
	Projects          []ProjectListView
	WorkItem          *WorkItemView
	Workflow          *WorkflowView
	Completion        *completion.Result
	Context           *projectapp.EffectiveContext
	Setup             *projectapp.SetupPreview
	Edit              *projectapp.EditPreview
	RuntimeResolution *runtimeapplication.Preview
	ExecutionTarget   *ExecutionTargetView
	// RuntimeAuth is the sanitized authentication preflight report.
	RuntimeAuth *runtimeadapter.AuthReport
	// Readiness is the canonical Project readiness report (project validate);
	// Preflight is the operation projection that blocked an effect.
	Readiness *projectapp.ReadinessReport
	Preflight *projectapp.OperationReadiness
	// Admission is the #230 central admission decision that denied an
	// operation before its readiness and effects.
	Admission *projectapp.AdmissionDecision
	// Operational is the #230 machine-local operational-state preview of a
	// Project archive/reactivate or Integration disable/enable request.
	Operational *projectapp.OperationalPreview
	// Integrations is the #230 Integration inventory, show or static
	// validation report (integration list|show|validate).
	Integrations *projectapp.IntegrationReport
	// ProjectState is the machine-local operational status reported beside a
	// selector-based Project validation.
	ProjectState  *ProjectStateView
	PreviewDigest string
	Runtime       *RuntimeView
	Bootstrap     *BootstrapView
	Draft         *workitem.DraftPreview
	Selection     *workitem.SelectionPreview
	Questions     []workitem.Question
	// WorkItemChange is the reviewed Work Item lifecycle preview (#230);
	// WorkItems is the local-link enumeration of work-item list.
	WorkItemChange *workitem.ChangePreview
	WorkItems      []WorkItemView
	Projection     *workflow.ProjectionPreview
	// Maintenance is a bounded, content-free view for compatibility,
	// cleanup, recovery, and upgrade previews and results.
	Maintenance       any
	WorkflowAuthoring *projectapp.WorkflowReport
}

type RuntimeSkillView struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	State  string `json:"state"`
}

// RuntimeConflictView names one preserved artifact, relative to the Runtime
// skill root, that blocks convergence.
type RuntimeConflictView struct {
	Artifact string `json:"artifact"`
	State    string `json:"state"`
	SHA256   string `json:"sha256,omitempty"`
}
type RuntimeView struct {
	SkillSetVersion     string                `json:"skillSetVersion"`
	BinaryCompatibility string                `json:"binaryCompatibility"`
	Skills              []RuntimeSkillView    `json:"skills"`
	Receipt             string                `json:"receipt,omitempty"`
	Conflicts           []RuntimeConflictView `json:"conflicts,omitempty"`
}

// RuntimeBootstrapView is one supported Runtime in a first-run report.
type RuntimeBootstrapView struct {
	Runtime                        string                `json:"runtime"`
	Executable                     string                `json:"executable"`
	Present                        bool                  `json:"present"`
	ConfigurationWithoutExecutable bool                  `json:"configurationWithoutExecutable"`
	State                          string                `json:"state"`
	Reason                         string                `json:"reason"`
	SkillSetVersion                string                `json:"skillSetVersion,omitempty"`
	Skills                         []RuntimeSkillView    `json:"skills,omitempty"`
	Receipt                        string                `json:"receipt,omitempty"`
	Conflicts                      []RuntimeConflictView `json:"conflicts,omitempty"`
}

// BootstrapView reports every supported Runtime after first run.
type BootstrapView struct {
	Detected int                    `json:"detected"`
	Failed   int                    `json:"failed"`
	Runtimes []RuntimeBootstrapView `json:"runtimes"`
}

type RepositoryView struct {
	Key  string `json:"key"`
	Path string `json:"path"`
	// Availability is reported by project show only: available or unavailable.
	Availability string `json:"availability,omitempty"`
}
type ProjectView struct {
	ID           string           `json:"id"`
	Slug         string           `json:"slug"`
	Source       string           `json:"source"`
	Repositories []RepositoryView `json:"repositories"`
	// State is the machine-local operational status (project show only).
	State *ProjectStateView `json:"state,omitempty"`
}
type ProjectListView struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	// Status is active, archived, invalid or recovery_required.
	Status string `json:"status,omitempty"`
}
type WorkItemView struct {
	ProjectID     string `json:"projectId"`
	RepositoryKey string `json:"repositoryKey"`
	Provider      string `json:"provider"`
	Resource      string `json:"resource"`
	ExternalID    string `json:"externalId"`
	URL           string `json:"url"`
	State         string `json:"state"`
}
type WorkflowStepView struct {
	Revision    uint64 `json:"revision"`
	From        string `json:"from"`
	To          string `json:"to"`
	Outcome     string `json:"outcome"`
	CommittedAt string `json:"committedAt"`
}
type WorkflowView struct {
	GateAction      *workflow.GateAction `json:"gateAction,omitempty"`
	GateCommand     []string             `json:"gateCommand,omitempty"`
	ExecutionID     string               `json:"executionId"`
	WorkflowVersion string               `json:"workflowVersion"`
	Status          string               `json:"status"`
	CurrentGate     string               `json:"currentGate"`
	LifecycleStage  string               `json:"lifecycleStage,omitempty"`
	Blocked         bool                 `json:"blocked,omitempty"`
	NeedsDecision   bool                 `json:"needsDecision,omitempty"`
	NeedsApproval   bool                 `json:"needsApproval,omitempty"`
	Revision        uint64               `json:"revision"`
	RepositoryKey   string               `json:"repositoryKey"`
	WorkItem        WorkItemView         `json:"workItem"`
	RuntimeID       string               `json:"runtimeId"`
	Transitions     []WorkflowStepView   `json:"transitions"`
}

// Run parses one CLI action, delegates it, and emits one safe structured event.
// It never turns a parser error into user-visible text because parser text may
// contain rejected input.
func Run(ctx context.Context, args []string, service Service, source provenance.Value, stdout io.Writer) int {
	structured := append([]string{"--json"}, args...)
	return RunInteractive(ctx, structured, service, source, nil, stdout, io.Discard)
}

// RunInteractive adds the bounded prompt path used by `project configure` and
// selects human or machine-readable presentation without changing application behavior.
func RunInteractive(ctx context.Context, args []string, service Service, source provenance.Value, stdin io.Reader, stdout, stderr io.Writer) int {
	if handled, code := HandleHelp(args, source, stdout); handled {
		return code
	}
	stdout = withHelpGuidance(stdout, args)
	if handled, code := InspectSkill(args, source, stdout); handled {
		return code
	}
	mode, args := parseOutputMode(args)
	var sessionOK bool
	ctx, args, sessionOK = projectSessionArgs(ctx, args)
	if !sessionOK {
		return emitParserFailure(stdout, mode, "project_context", "invalid_input", source)
	}
	if service == nil {
		return emit(stdout, mode, event{Operation: "unknown", Status: Failed, Category: "application_unavailable"})
	}
	if contextual, ok := service.(ProjectContextService); ok {
		if len(args) >= 2 && args[0] == "project" && args[1] == "context" {
			response := runProjectContext(ctx, args[2:], contextual)
			if response.Completion == nil && response.Category == "invalid_input" {
				return emitParserFailure(stdout, mode, "project_context", "invalid_input", source)
			}
			return emitResponse(stdout, mode, "project_context", response)
		}
		var failure *Result
		ctx, args, failure = effectiveProjectArgs(ctx, args, contextual)
		if failure != nil {
			return emitResponse(stdout, mode, "project_context", *failure)
		}
	}
	if commandMatches(args, runtimeProfilePreviewAction) {
		input, ok := runtimePreviewFlags(args[3:])
		if !ok {
			return emitParserFailure(stdout, mode, runtimeProfilePreviewAction, "invalid_input", source)
		}
		profiles, ok := service.(RuntimeProfilePreviewService)
		if !ok {
			return emit(stdout, mode, event{Operation: runtimeProfilePreviewAction, Status: Failed, Category: "application_unavailable"})
		}
		return emitResponse(stdout, mode, runtimeProfilePreviewAction, profiles.RuntimeProfilePreview(ctx, input))
	}
	if commandMatches(args, codexAuthAction) || commandMatches(args, claudeAuthAction) {
		operation, runtimeID := codexAuthAction, "codex"
		if commandMatches(args, claudeAuthAction) {
			operation, runtimeID = claudeAuthAction, "claude"
		}
		if len(args) != 3 {
			return emitParserFailure(stdout, mode, operation, "invalid_input", source)
		}
		auth, ok := service.(RuntimeAuthService)
		if !ok {
			return emit(stdout, mode, event{Operation: operation, Status: Failed, Category: "application_unavailable"})
		}
		return emitResponse(stdout, mode, operation, auth.RuntimeAuth(ctx, runtimeID))
	}
	if commandMatches(args, runtimeProfileValidateAction) {
		if len(args) != 3 {
			return emitParserFailure(stdout, mode, runtimeProfileValidateAction, "invalid_input", source)
		}
		profiles, ok := service.(RuntimeProfileService)
		if !ok {
			return emit(stdout, mode, event{Operation: runtimeProfileValidateAction, Status: Failed, Category: "application_unavailable"})
		}
		return emitResponse(stdout, mode, runtimeProfileValidateAction, profiles.RuntimeProfileValidate(ctx))
	}
	if handled, code := runWorkflowList(ctx, args, service, source, mode, stdout); handled {
		return code
	}
	if operation, rest, ok := maintenanceAction(args); ok {
		return runMaintenance(ctx, mode, operation, rest, service, source, stdout)
	}
	if operation, rest, ok := integrationAction(args); ok {
		return runIntegration(ctx, mode, operation, rest, service, source, stdout)
	}
	if handled, code := runProjectWorkflow(ctx, mode, args, service, source, stdout); handled {
		return code
	}
	if handled, code := runProjectLifecycle(ctx, mode, args, service, source, stdout); handled {
		return code
	}
	if len(args) >= 2 && args[0] == "project" && args[1] == "configure" && stdin != nil {
		values, ok := flags(configureAction, args[2:])
		if !ok {
			return emitParserFailure(stdout, mode, configureAction, "invalid_input", source)
		}
		// Guided EDIT asks nothing: omission already means preservation.
		if !values.projectSupplied && configureRequestIssue(values) != "invalid_input" && (values.slug == "" || values.name == "" || len(values.repositories) == 0 || !flagSupplied(args[2:], "--work-item-provider")) {
			return runInteractiveConfiguration(ctx, mode, args[2:], service, stdin, stdout, stderr)
		}
	}
	if len(args) >= 2 && args[0] == "work-item" && args[1] == "create" && stdin != nil {
		values, ok := workItemFlags(workItemCreateAction, args[2:])
		if !ok {
			return emitParserFailure(stdout, mode, workItemCreateAction, "invalid_input", source)
		}
		if !completeWorkItemCreate(values) {
			return runInteractiveWorkItemCreate(ctx, mode, values, service, stdin, stdout, stderr)
		}
	}
	if len(args) >= 2 && args[0] == "work-item" && args[1] != "create" && stdin != nil {
		operation := action("work_item_" + args[1])
		values, ok := workItemFlags(operation, args[2:])
		if knownWorkItem(operation) && operation != workItemListAction && ok && values.number == 0 && (values.project == "" || values.repository == "" || values.workItem == "") {
			return runInteractiveSelectors(ctx, mode, operation, values, service, source, stdin, stdout, stderr)
		}
	}
	if len(args) >= 2 && args[0] == "workflow" && stdin != nil {
		operation := action("workflow_" + args[1])
		values, ok := workflowFlags(operation, args[2:])
		needsExecution := operation != workflowStartAction
		if knownWorkflow(operation) && ok && values.number == 0 && (values.project == "" || values.repository == "" || values.workItem == "" || needsExecution && values.execution == "") {
			return runInteractiveSelectors(ctx, mode, operation, values, service, source, stdin, stdout, stderr)
		}
	}
	operation, input, result := request(args, service)
	if result != nil {
		if operation == validateAction || operation == showAction || selectorAction(operation) || *result == "invalid_input" && (operation == configureAction || operation == resolveAction) || *result == "incomplete_edit_authority" {
			return emitParserFailure(stdout, mode, operation, *result, source)
		}
		return emit(stdout, mode, event{Operation: operation, Status: Failed, Category: *result})
	}
	response := dispatch(ctx, operation, input, service)
	return emitResponse(stdout, mode, operation, response)
}

func flagSupplied(args []string, name string) bool {
	for _, value := range args {
		if value == name || strings.HasPrefix(value, name+"=") {
			return true
		}
	}
	return false
}

func emitParserFailure(writer io.Writer, mode outputMode, operation action, issue string, source provenance.Value) int {
	message, next := parserFailureText(operation, issue)
	statement, err := provenance.NewText(message, provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	nextAction, err := provenance.NewText(helpNext(writer, next), provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	result, err := completion.NewValidationFailure(statement, nextAction, source)
	if err != nil {
		return ExitFailure
	}
	return emitCompletion(writer, mode, result)
}

func parserFailureText(operation action, issue string) (string, string) {
	if issue == "invalid_command" {
		return "Command is invalid", "Choose a documented command"
	}
	if operation == skillInspectAction {
		return "Skill inspection input is invalid", "Run skill inspect with one exact embedded skill name and no workflow arguments"
	}
	if operation == runtimeProfilePreviewAction {
		return "Runtime profile preview input is invalid", "Provide explicit Project, role, complexity and capabilities"
	}
	if operation == runtimeProfileValidateAction {
		return "Runtime profile validation input is invalid", "Run runtime profile validate without flags or arguments"
	}
	if operation == codexAuthAction || operation == claudeAuthAction {
		return "Runtime authentication preflight input is invalid", "Run runtime codex auth or runtime claude auth without flags or arguments"
	}
	if issue == "incomplete_edit_authority" {
		return "Project edit authority is incomplete", "Supply --project-id, --preview-digest, and --authorize-local together from the reviewed preview, or omit all three to preview"
	}
	if issue == "invalid_input" && (operation == showAction || operation == resolveAction || operation == configureAction) {
		return "Explicit selector input is invalid", "Remove unknown, duplicate, or conflicting inputs and retry"
	}
	if operation == validateAction && issue == "missing_required_input" {
		return "Project slug is required", "Provide a Project slug and retry validation"
	}
	if operation == showAction && issue == "missing_required_input" {
		return "Project selector is required", "Provide a Project UUID or slug and retry inspection"
	}
	if operation == validateAction {
		return "Project validation input is invalid", "Review supported validation flags and retry"
	}
	if selectorAction(operation) {
		if issue == "missing_required_input" {
			return "Explicit selectors are incomplete", "Provide only the missing Project, Repository, Work Item, or Execution selector"
		}
		return "Explicit selector input is invalid", "Remove unknown, duplicate, or conflicting inputs and retry"
	}
	return "Project inspection input is invalid", "Review supported inspection flags and retry"
}

func selectorAction(operation action) bool {
	return knownWorkItem(operation) || knownWorkflow(operation)
}

func emitResponse(writer io.Writer, mode outputMode, operation action, response Result) int {
	if response.Completion != nil && response.Context != nil {
		return emitContextCompletion(writer, mode, *response.Completion, *response.Context)
	}
	if response.Completion != nil {
		if response.RuntimeResolution != nil {
			return emitRuntimeResolutionCompletion(writer, mode, *response.Completion, response)
		}
		if response.RuntimeAuth != nil {
			return emitRuntimeAuthCompletion(writer, mode, *response.Completion, *response.RuntimeAuth)
		}
		if response.Operational != nil {
			return emitOperationalCompletion(writer, mode, *response.Completion, response)
		}
		if response.Readiness != nil || response.Preflight != nil || response.Admission != nil {
			return emitReadinessCompletion(writer, mode, *response.Completion, response)
		}
		if response.Project != nil {
			return emitProjectCompletion(writer, mode, *response.Completion, *response.Project)
		}
		if response.Projects != nil {
			return emitProjectListCompletion(writer, mode, *response.Completion, response.Projects)
		}
		if response.Setup != nil {
			return emitSetupCompletion(writer, mode, *response.Completion, *response.Setup)
		}
		if response.Edit != nil {
			return emitEditCompletion(writer, mode, *response.Completion, *response.Edit)
		}
		if response.Runtime != nil {
			return emitRuntimeCompletion(writer, mode, *response.Completion, *response.Runtime)
		}
		if response.Bootstrap != nil {
			return emitBootstrapCompletion(writer, mode, *response.Completion, *response.Bootstrap)
		}
		if response.WorkItemChange != nil || response.WorkItems != nil {
			return emitWorkItemLifecycleCompletion(writer, mode, *response.Completion, response)
		}
		if response.Draft != nil || response.Selection != nil || response.WorkItem != nil || len(response.Questions) != 0 {
			return emitWorkItemCompletion(writer, mode, *response.Completion, response)
		}
		if response.Workflow != nil || response.Projection != nil {
			return emitWorkflowCompletion(writer, mode, *response.Completion, response)
		}
		return emitCompletion(writer, mode, *response.Completion)
	}
	return emit(writer, mode, eventFrom(operation, response))
}

type action string

const (
	initAction              action = "init"
	validateAction          action = "validate"
	reopenAction            action = "reopen"
	updateAction            action = "update"
	installAction           action = "install"
	resolveAction           action = "resolve"
	showAction              action = "show"
	listAction              action = "list"
	configureAction         action = "configure"
	workItemCreateAction    action = "work_item_create"
	workItemSelectAction    action = "work_item_select"
	workItemShowAction      action = "work_item_show"
	workItemCommentAction   action = "work_item_comment"
	workItemCompleteAction  action = "work_item_complete"
	workItemListAction      action = "work_item_list"
	workItemUpdateAction    action = "work_item_update"
	workItemCloseAction     action = "work_item_close"
	workItemReopenAction    action = "work_item_reopen"
	workflowStartAction     action = "workflow_start"
	workflowAdvanceAction   action = "workflow_advance"
	workflowFactAction      action = "workflow_fact"
	workflowResumeAction    action = "workflow_resume"
	workflowStatusAction    action = "workflow_status"
	workflowEvidenceAction  action = "workflow_evidence"
	workflowReconcileAction action = "workflow_reconcile"
	codexInstallAction      action = "runtime_codex_install"
	codexStatusAction       action = "runtime_codex_status"
	claudeInstallAction     action = "runtime_claude_install"
	claudeStatusAction      action = "runtime_claude_status"
	codexAuthAction         action = "runtime_codex_auth"
	claudeAuthAction        action = "runtime_claude_auth"
	firstRunAction          action = "first_run"
)

type requestInput struct {
	automatic                                bool
	itemType, beneficiary, value             string
	classification                           repositoryFlags
	elaboratedSections                       elaboratedSectionFlags
	slug                                     string
	name                                     string
	projectID                                string
	source                                   string
	selector                                 string
	repositories                             repositoryFlags
	removeRepositories                       repositoryFlags
	repositoryRemotes, runtimes              repositoryFlags
	modelProfiles, runtimePreferences        repositoryFlags
	technology, removeTechnology             repositoryFlags
	documentation, contextSources, glossary  repositoryFlags
	businessContext                          string
	bootstrapSupplied                        bool
	workItemProvider                         string
	removeWorkItemProvider                   bool
	projectSupplied, slugSupplied            bool
	nameSupplied, providerSupplied           bool
	replaySupplied                           bool
	previewDigest                            string
	project, repository, workItem, execution string
	providerRepository                       string
	intent, problem, desiredOutcome, context string
	scope, constraints, nonGoals, acceptance string
	message, title                           string
	gate, outcome, reference, next, fact     string
	runtime                                  string
	role, complexity, capabilities           string
	runtimePreview                           string
	number                                   int
	expectedRevision                         uint64
	authorizeExternal                        bool
	authorizeLocal                           bool
	active                                   bool
}

func request(args []string, service Service) (action, requestInput, *string) {
	if service == nil {
		return "unknown", requestInput{}, category("application_unavailable")
	}
	command, _, _, validCommand := resolveCommand(args)
	if !validCommand {
		return "unknown", requestInput{}, category("invalid_command")
	}
	if len(args) == 1 && command.operation == firstRunAction {
		return firstRunAction, requestInput{}, nil
	}
	if len(args) == 3 && args[0] == "runtime" && (args[1] == "codex" || args[1] == "claude") {
		operation := command.operation
		if operation == codexInstallAction || operation == codexStatusAction || operation == claudeInstallAction || operation == claudeStatusAction {
			return operation, requestInput{}, nil
		}
		return "unknown", requestInput{}, category("invalid_command")
	}
	if len(args) >= 2 && args[0] == "work-item" {
		operation := command.operation
		if !knownWorkItem(operation) {
			return "unknown", requestInput{}, category("invalid_command")
		}
		values, ok := workItemFlags(operation, args[2:])
		if !ok {
			return operation, requestInput{}, category("invalid_input")
		}
		if issue := selectorRequestIssue(operation, values); issue != "" {
			return operation, values, category(issue)
		}
		return operation, values, nil
	}
	if len(args) >= 2 && args[0] == "workflow" {
		operation := command.operation
		if !knownWorkflow(operation) {
			return "unknown", requestInput{}, category("invalid_command")
		}
		values, ok := workflowFlags(operation, args[2:])
		if !ok {
			return operation, requestInput{}, category("invalid_input")
		}
		if issue := selectorRequestIssue(operation, values); issue != "" {
			return operation, values, category(issue)
		}
		return operation, values, nil
	}
	if len(args) < 2 || args[0] != "project" {
		return "unknown", requestInput{}, category("invalid_command")
	}
	operation := command.operation
	if !known(operation) {
		return "unknown", requestInput{}, category("invalid_command")
	}
	values, ok := flags(operation, args[2:])
	if !ok {
		return operation, requestInput{}, category("invalid_input")
	}
	if operation == initAction && (values.slug == "" || values.name == "") {
		return operation, values, category("missing_required_input")
	}
	if operation == installAction && values.source == "" {
		return operation, values, category("missing_required_input")
	}
	if (operation == resolveAction || operation == showAction) && missingRequiredInputs(operation, values) {
		return operation, values, category("missing_required_input")
	}
	if operation == configureAction {
		if issue := configureRequestIssue(values); issue != "" {
			return operation, values, category(issue)
		}
		return operation, values, nil
	}
	if operation != initAction && operation != installAction && operation != resolveAction && operation != showAction && operation != listAction && values.slug == "" {
		return operation, values, category("missing_required_input")
	}
	if operation == updateAction && values.name == "" {
		return operation, values, category("missing_required_input")
	}
	return operation, values, nil
}

func selectorRequestIssue(operation action, values requestInput) string {
	return selectorRequestIssueWithPresence(operation, values, true)
}

func selectorRequestIssueWithPresence(operation action, values requestInput, requireInputs bool) string {
	if values.workItem != "" && (values.number != 0 || values.providerRepository != "") {
		return "invalid_input"
	}
	if requireInputs && missingRequiredInputs(operation, values, "project", "repository") {
		return "missing_required_input"
	}
	if knownWorkItem(operation) {
		if operation == workItemListAction {
			return ""
		}
		if operation == workItemCreateAction {
			if values.workItem != "" {
				return "invalid_input"
			}
			if requireInputs && missingRequiredInputs(operation, values, "provider-repository") {
				return "missing_required_input"
			}
			return ""
		}
		if requireInputs && missingRequiredInputs(operation, values, "number") {
			return "missing_required_input"
		}
		if values.workItem != "" {
			if _, _, _, ok := parseWorkItemSelector(values.workItem); !ok {
				return "invalid_input"
			}
		}
		if requireInputs && missingRequiredInputs(operation, values, "provider-repository") {
			return "missing_required_input"
		}
		if requireInputs && missingRequiredInputs(operation, values, "message") {
			return "missing_required_input"
		}
		return ""
	}
	if requireInputs && missingRequiredInputs(operation, values, "number") {
		return "missing_required_input"
	}
	if requireInputs && operation == workflowStartAction && missingRequiredInputs(operation, values, "role", "complexity", "capabilities") {
		return "missing_required_input"
	}
	if values.workItem != "" {
		if _, _, _, ok := parseWorkItemSelector(values.workItem); !ok {
			return "invalid_input"
		}
		if operation == workflowStartAction && values.execution != "" {
			return "invalid_input"
		}
		if requireInputs && missingRequiredInputs(operation, values, "execution") {
			return "missing_required_input"
		}
	}
	if requireInputs && missingRequiredInputs(operation, values, "expected-revision", "gate", "outcome") {
		return "missing_required_input"
	}
	if operation == workflowAdvanceAction && values.automatic && (values.gate != "" || values.outcome != "" || values.reference != "" || values.next != "") {
		return "invalid_input"
	}
	if requireInputs && missingRequiredInputs(operation, values, "fact", "reference") {
		return "missing_required_input"
	}
	return ""
}

func projectFlagSet(operation action, values *requestInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	if operation != listAction {
		set.StringVar(&values.slug, "slug", "", "Project slug (CREATE only for configure); `<slug>`.")
	}
	if operation == initAction || operation == updateAction {
		set.StringVar(&values.name, "name", "", "Project display name; `<text>`.")
	}
	if operation == installAction {
		set.StringVar(&values.source, "source", "", "Project manifest source; `<path>`.")
		set.Var(&values.repositories, "repository", "Bind each Repository the manifest declares; `<key>=<absolute-path>`.")
	}
	if operation == resolveAction || operation == showAction {
		set.StringVar(&values.selector, "selector", "", "Configured Project identity; `<uuid-or-slug>`.")
	}
	if operation == configureAction {
		set.StringVar(&values.projectID, "project-id", "", "Project UUID; `<uuid>`. Optional for creation; for an edit, the reviewed Project ID required with --preview-digest and --authorize-local.")
		set.StringVar(&values.name, "name", "", "Project display name; `<text>`.")
		set.Var(&values.repositories, "repository", "Add or update Repository; `<key>=<absolute-path>`. Conflicts with removing the same key.")
		set.StringVar(&values.workItemProvider, "work-item-provider", "", "Work Item provider; `<provider-id>`. CREATE also accepts none; edit uses --remove-work-item-provider.")
		set.StringVar(&values.previewDigest, "preview-digest", "", "Exact reviewed preview digest; `<digest>`. Required for authorized publication.")
		set.BoolVar(&values.authorizeLocal, "authorize-local", false, "Explicit authority for the exact local effect; boolean. Never inferred by discovery.")
		set.StringVar(&values.project, "project", "", "Configured Project identity; `<uuid-or-slug>`. For configure selects preview-only edit mode.")
		set.BoolVar(&values.removeWorkItemProvider, "remove-work-item-provider", false, "Remove the existing provider in edit mode; conflicts with --work-item-provider.")
		set.Var(&values.removeRepositories, "remove-repository", "Remove a Repository in edit mode; `<key>`. Conflicts with adding the same key.")
		set.Var(&values.repositoryRemotes, "repository-remote", "CREATE: explicit remote identity; `<key>=<locator>` or `<key>=none`. Required when discovered remotes are ambiguous.")
		set.Var(&values.runtimes, "runtime", "CREATE: allowed Runtime; `<runtime-id>`. Must be locally configured; never defaulted.")
		set.Var(&values.modelProfiles, "model-profile", "CREATE: allowed Model Profile; `<profile-key>`. Copied from local configuration of an allowed Runtime.")
		set.Var(&values.runtimePreferences, "runtime-preference", "CREATE: Runtime preference; `<role>/<complexity>=<profile-key>`.")
		set.Var(&values.technology, "technology", "CREATE: add or replace a technology fact; `<key>=<value>`.")
		set.Var(&values.removeTechnology, "remove-technology", "CREATE: drop a detected technology fact; `<key>`.")
		set.Var(&values.documentation, "documentation", "CREATE: documentation source; `<key>=repository:<repository-key>/<relative-path>` or `<key>=local-file:<absolute-path>`.")
		set.StringVar(&values.businessContext, "business-context", "", "CREATE: bounded business context; `<text>`.")
		set.Var(&values.contextSources, "context-source", "CREATE: business context documentation reference; `<documentation-key>`.")
		set.Var(&values.glossary, "glossary", "CREATE: glossary entry; `<key>=<term>:<definition>`.")
	}
	return set
}

func flags(operation action, args []string) (requestInput, bool) {
	var values requestInput
	set := projectFlagSet(operation, &values)
	if invalidFlagSyntax(set, args, repeatableFlags(set)) {
		return requestInput{}, false
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return requestInput{}, false
	}
	set.Visit(func(current *flag.Flag) {
		switch current.Name {
		case "project":
			values.projectSupplied = true
		case "slug":
			values.slugSupplied = true
		case "name":
			values.nameSupplied = true
		case "work-item-provider":
			values.providerSupplied = true
		case "project-id", "preview-digest", "authorize-local":
			values.replaySupplied = true
		case "repository-remote", "runtime", "model-profile", "runtime-preference", "technology", "remove-technology", "documentation", "business-context", "context-source", "glossary":
			values.bootstrapSupplied = true
		}
	})
	return values, true
}

// configureRequestIssue validates presence and conflicting operations only.
// Value validation and every merge rule belong to the application layer.
func configureRequestIssue(values requestInput) string {
	if !values.projectSupplied {
		if values.removeWorkItemProvider || len(values.removeRepositories) != 0 {
			return "invalid_input"
		}
		if _, ok := createConfigureInput(values); !ok {
			return "invalid_input"
		}
		if missingRequiredInputs(configureAction, values) {
			return "missing_required_input"
		}
		return ""
	}
	// EDIT replay is the complete reviewed tuple; a partial tuple fails here,
	// before any selector resolution or state read (I230-T03).
	if values.replaySupplied && (values.projectID == "" || values.previewDigest == "" || !values.authorizeLocal) {
		return "incomplete_edit_authority"
	}
	// Bootstrap intent is CREATE-only; post-create lifecycle belongs to #230.
	if values.bootstrapSupplied {
		return "invalid_input"
	}
	// EDIT: rename is out of scope, the CREATE-only `none` alias is not a
	// removal spelling, and set/remove of one target cannot be combined.
	if values.project == "" || values.slugSupplied || values.providerSupplied && (values.removeWorkItemProvider || values.workItemProvider == "none") {
		return "invalid_input"
	}
	upserts, ok := parseRepositories(values.repositories)
	if !ok {
		return "invalid_input"
	}
	keys := map[string]bool{}
	for _, upsert := range upserts {
		keys[upsert.Key] = true
	}
	removed := map[string]bool{}
	for _, key := range values.removeRepositories {
		if key == "" || keys[key] || removed[key] {
			return "invalid_input"
		}
		removed[key] = true
	}
	return ""
}

func workItemFlagSet(operation action, values *requestInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&values.project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
	if operation == workItemListAction {
		set.StringVar(&values.repository, "repository", "", "Optional Project-scoped Repository filter; `<key>`.")
		return set
	}
	set.StringVar(&values.repository, "repository", "", "Project-scoped Repository selector; `<key>`.")
	set.StringVar(&values.providerRepository, "provider-repository", "", "Explicit provider target; `<owner/repository>`. Conflicts with --work-item.")
	set.StringVar(&values.workItem, "work-item", "", "Exact existing Work Item; `github:<owner>/<repository>#<number>`. Alternative to --number; rejected by create.")
	set.StringVar(&values.previewDigest, "preview-digest", "", "Exact reviewed preview digest; `<digest>`. Required for authorized publication.")
	set.BoolVar(&values.authorizeExternal, "authorize-external", false, "Explicit authority for the exact reviewed Provider effect; boolean.")
	set.BoolVar(&values.authorizeLocal, "authorize-local", false, "Explicit authority for the exact local effect; boolean. Never inferred by discovery.")
	if operation == workItemUpdateAction {
		workItemUpdateFlags(set, values)
	}
	if operation == workItemCreateAction {
		set.Var(&values.elaboratedSections, "elaborated-section", "Axiom-authored section; `<name>=<content>`. Names: problem, desired_outcome, context, scope, constraints, non_goals, acceptance_expectations. Conflicts with a verbatim value for that section.")
		set.StringVar(&values.intent, "intent", "", "Original user intent; `<text>`. May supply the initial problem for guided elaboration.")
		set.StringVar(&values.itemType, "type", "", "Work Item delivery type; `<story|bug|task>`. Domain requires a type before publication.")
		set.StringVar(&values.beneficiary, "beneficiary", "", "Who benefits from a story; `<text>`. Domain requires this for story publication.")
		set.StringVar(&values.value, "value", "", "Concrete story benefit; `<text>`. Domain requires this for story publication.")
		set.Var(&values.classification, "classification", "Explicit provider classification; `<label>`. Validated against the provider catalog.")
		set.StringVar(&values.problem, "problem", "", "Verbatim problem section; `<text>`. Content gaps are gathered before publication.")
		set.StringVar(&values.desiredOutcome, "desired-outcome", "", "Verbatim desired outcome; `<text>`. Content gaps are gathered before publication.")
		set.StringVar(&values.context, "context", "", "Verbatim relevant context; `<text>`. Content gaps are gathered before publication.")
		set.StringVar(&values.scope, "scope", "", "Verbatim bounded scope; `<text>`. Content gaps are gathered before publication.")
		set.StringVar(&values.constraints, "constraints", "", "Verbatim constraints; `<text>`. Content gaps are gathered before publication.")
		set.StringVar(&values.nonGoals, "non-goals", "", "Verbatim exclusions; `<text>`. Content gaps are gathered before publication.")
		set.StringVar(&values.acceptance, "acceptance", "", "Verbatim acceptance expectations; `<text>`. Content gaps are gathered before publication.")
	} else {
		set.IntVar(&values.number, "number", 0, "Legacy Work Item number; `<positive-integer>`. Alternative to --work-item.")
	}
	if operation == workItemCommentAction {
		set.StringVar(&values.message, "message", "", "Provider comment; `<text>`.")
	}
	return set
}

func workItemFlags(operation action, args []string) (requestInput, bool) {
	var values requestInput
	set := workItemFlagSet(operation, &values)
	if invalidFlagSyntax(set, args, repeatableFlags(set)) {
		return requestInput{}, false
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 || values.workItem != "" && (values.providerRepository != "" || values.number != 0) {
		return requestInput{}, false
	}
	return values, true
}

func knownWorkItem(operation action) bool {
	return operationInGroup(operation, "work-item")
}

func workflowFlagSet(operation action, values *requestInput) *flag.FlagSet {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.StringVar(&values.project, "project", "", "Configured Project identity; `<uuid-or-slug>`.")
	set.StringVar(&values.repository, "repository", "", "Project-scoped Repository selector; `<key>`.")
	set.StringVar(&values.workItem, "work-item", "", "Exact existing Work Item; `github:<owner>/<repository>#<number>`. Alternative to --number; rejected by create.")
	set.StringVar(&values.execution, "execution", "", "Exact Execution identity; `<execution-id>`. Required with --work-item except start; start rejects that combination.")
	set.IntVar(&values.number, "number", 0, "Legacy Work Item number; `<positive-integer>`. Alternative to --work-item.")
	if operation == workflowStartAction {
		set.StringVar(&values.runtime, "runtime", "", "Optional Runtime constraint; `<codex|claude>`. Portable policy resolves omitted Runtime.")
		runtimeRequestFlags(set, &values.role, &values.complexity, &values.capabilities)
		set.StringVar(&values.runtimePreview, "runtime-preview", "", "Exact reviewed Runtime resolution digest; `<digest>`. Omission previews without starting.")
	}
	if operation == workflowAdvanceAction || operation == workflowFactAction || operation == workflowResumeAction || operation == workflowReconcileAction {
		set.Uint64Var(&values.expectedRevision, "expected-revision", 0, "Exact current Execution revision; `<positive-integer>`.")
	}
	if operation == workflowReconcileAction {
		set.StringVar(&values.previewDigest, "preview-digest", "", "Exact reviewed preview digest; `<digest>`. Required for authorized publication.")
		set.BoolVar(&values.authorizeExternal, "authorize-external", false, "Explicit authority for the exact reviewed Provider effect; boolean.")
	}
	if operation == workflowAdvanceAction {
		set.BoolVar(&values.automatic, "automatic", false, "Evaluate deterministic Intake only; boolean. Cannot combine with --gate, --outcome, --reference or --next.")
		set.StringVar(&values.gate, "gate", "", "Target workflow gate; `<gate>`. Validated against the current Execution.")
		set.StringVar(&values.outcome, "outcome", "", "Workflow outcome; `<outcome>`. Validated against the gate.")
		set.StringVar(&values.reference, "reference", "", "Evidence reference; `<reference>`. Validated by the workflow.")
		set.StringVar(&values.next, "next", "", "Next action; `<text>`.")
	}
	if operation == workflowFactAction {
		set.StringVar(&values.fact, "fact", "", "Workflow fact; `<fact>`. Validated by the workflow.")
		set.BoolVar(&values.active, "active", false, "Whether the fact is active; boolean.")
		set.StringVar(&values.reference, "reference", "", "Evidence reference; `<reference>`. Validated by the workflow.")
		set.BoolVar(&values.authorizeLocal, "authorize-local", false, "Explicit authority for the exact local effect; boolean. Never inferred by discovery.")
	}
	return set
}

func workflowFlags(operation action, args []string) (requestInput, bool) {
	var values requestInput
	set := workflowFlagSet(operation, &values)
	if invalidFlagSyntax(set, args, nil) {
		return requestInput{}, false
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 || values.workItem != "" && values.number != 0 {
		return requestInput{}, false
	}
	if values.automatic {
		for _, name := range []string{"--gate", "--outcome", "--reference", "--next"} {
			if flagSupplied(args, name) {
				return requestInput{}, false
			}
		}
	}
	if operation == workflowStartAction {
		if flagSupplied(args, "--runtime") && !workflow.SupportedRuntime(values.runtime) {
			return requestInput{}, false
		}
		if values.runtimePreview != "" && !validPreviewDigest(values.runtimePreview) {
			return requestInput{}, false
		}
	}
	return values, true
}

func invalidFlagSyntax(set *flag.FlagSet, args []string, repeatable map[string]bool) bool {
	seen := make(map[string]bool)
	for index := 0; index < len(args); index++ {
		value := args[index]
		// Go flag accepts single-hyphen aliases and silently overwrites duplicates.
		// Reject unsupported syntax before handing any arguments to it.
		if value == "--" || !strings.HasPrefix(value, "--") {
			return true
		}
		name := strings.TrimPrefix(value, "--")
		hasValue := false
		if separator := strings.IndexByte(name, '='); separator >= 0 {
			name = name[:separator]
			hasValue = true
		}
		current := set.Lookup(name)
		if current == nil {
			return true
		}
		if !repeatable[name] && seen[name] {
			return true
		}
		seen[name] = true
		boolean, isBoolean := current.Value.(interface{ IsBoolFlag() bool })
		if hasValue || isBoolean && boolean.IsBoolFlag() {
			continue
		}
		if index+1 >= len(args) {
			return true
		}
		index++
	}
	return false
}

func parseWorkItemSelector(value string) (string, string, string, bool) {
	provider, target, found := strings.Cut(value, ":")
	resource, externalID, numbered := strings.Cut(target, "#")
	if !found || !numbered || provider != "github" || !validProviderResource(resource) {
		return "", "", "", false
	}
	number, err := strconv.Atoi(externalID)
	if err != nil || number <= 0 || strconv.Itoa(number) != externalID {
		return "", "", "", false
	}
	return provider, resource, externalID, true
}

func validProviderResource(value string) bool {
	owner, repository, found := strings.Cut(value, "/")
	if !found || owner == "" || repository == "" || strings.Contains(repository, "/") || owner == "." || owner == ".." || repository == "." || repository == ".." {
		return false
	}
	for _, part := range []string{owner, repository} {
		for _, character := range part {
			if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune("_.-", character) {
				continue
			}
			return false
		}
	}
	return true
}

func knownWorkflow(operation action) bool {
	return operation != workflowListAction && operationInGroup(operation, "workflow")
}

func known(operation action) bool {
	return operation != projectArchiveAction && operation != projectReactivateAction && operationInGroup(operation, "project")
}

func dispatch(ctx context.Context, operation action, input requestInput, service Service) Result {
	if err := ctx.Err(); err != nil {
		if operation == validateAction {
			return service.Validate(ctx, ProjectInput{Slug: input.slug})
		}
		if operation == showAction {
			return service.Show(ctx, ResolveInput{Selector: input.selector})
		}
		if operation == listAction {
			return service.List(ctx)
		}
		return Result{Status: Cancelled, Category: "cancelled"}
	}
	switch operation {
	case initAction:
		return service.Init(ctx, InitInput{Slug: input.slug, Name: input.name})
	case validateAction:
		return service.Validate(ctx, ProjectInput{Slug: input.slug})
	case reopenAction:
		return service.Reopen(ctx, ProjectInput{Slug: input.slug})
	case updateAction:
		return service.Update(ctx, UpdateInput{Slug: input.slug, Name: input.name})
	case installAction:
		repositories, ok := parseRepositories(input.repositories)
		if !ok {
			return Result{Status: Failed, Category: "invalid_input"}
		}
		return service.Install(ctx, InstallInput{Source: input.source, Repositories: repositories})
	case resolveAction:
		return service.Resolve(ctx, ResolveInput{Selector: input.selector})
	case showAction:
		return service.Show(ctx, ResolveInput{Selector: input.selector})
	case listAction:
		return service.List(ctx)
	case configureAction:
		if !input.projectSupplied {
			configuration, ok := createConfigureInput(input)
			if !ok {
				return Result{Status: Failed, Category: "invalid_input"}
			}
			return service.Configure(ctx, configuration)
		}
		repositories, ok := parseRepositories(input.repositories)
		if !ok {
			return Result{Status: Failed, Category: "invalid_input"}
		}
		return service.Configure(ctx, ConfigureInput{
			ProjectID: input.projectID, Project: input.project, Name: input.name, NameSupplied: input.nameSupplied,
			WorkItemProvider: input.workItemProvider, WorkItemProviderSupplied: input.providerSupplied, RemoveWorkItemProvider: input.removeWorkItemProvider,
			Repositories: repositories, RemoveRepositories: append([]string(nil), input.removeRepositories...),
			PreviewDigest: input.previewDigest, AuthorizeLocal: input.authorizeLocal,
		})
	case workItemCreateAction, workItemSelectAction, workItemShowAction, workItemCommentAction, workItemCompleteAction, workItemListAction, workItemUpdateAction, workItemCloseAction, workItemReopenAction:
		provider, resource, externalID, ok := parseWorkItemSelector(input.workItem)
		if input.workItem != "" && !ok {
			return Result{Status: Failed, Category: "invalid_input"}
		}
		value := WorkItemInput{ElaboratedSections: input.elaboratedSections, Project: input.project, Repository: input.repository, WorkItem: input.workItem, Provider: provider, ProviderRepository: resource, ExternalID: externalID, Intent: input.intent, Problem: input.problem, DesiredOutcome: input.desiredOutcome, Context: input.context, Scope: input.scope, Constraints: input.constraints, NonGoals: input.nonGoals, Acceptance: input.acceptance, Message: input.message, PreviewDigest: input.previewDigest, Number: input.number, AuthorizeExternal: input.authorizeExternal, AuthorizeLocal: input.authorizeLocal}
		value.Type, value.Beneficiary, value.Value = input.itemType, input.beneficiary, input.value
		value.Classification = append([]string(nil), input.classification...)
		value.Title = input.title
		if input.workItem == "" {
			value.ProviderRepository = input.providerRepository
			value.ExternalID = strconv.Itoa(input.number)
		}
		if workItemLifecycleAction(operation) {
			return dispatchWorkItemLifecycle(ctx, operation, value, service)
		}
		switch operation {
		case workItemCreateAction:
			return service.WorkItemCreate(ctx, value)
		case workItemSelectAction:
			return service.WorkItemSelect(ctx, value)
		case workItemShowAction:
			return service.WorkItemShow(ctx, value)
		case workItemCommentAction:
			return service.WorkItemComment(ctx, value)
		default:
			return service.WorkItemComplete(ctx, value)
		}
	case workflowStartAction, workflowAdvanceAction, workflowFactAction, workflowResumeAction, workflowStatusAction, workflowEvidenceAction, workflowReconcileAction:
		provider, resource, externalID, ok := parseWorkItemSelector(input.workItem)
		if input.workItem != "" && !ok {
			return Result{Status: Failed, Category: "invalid_input"}
		}
		value := WorkflowInput{Project: input.project, Repository: input.repository, WorkItem: input.workItem, Provider: provider, ProviderRepository: resource, ExternalID: externalID, Execution: input.execution, Number: input.number, Gate: input.gate, Outcome: input.outcome, Reference: input.reference, Next: input.next, Fact: input.fact, Active: input.active, ExpectedRevision: input.expectedRevision, PreviewDigest: input.previewDigest, AuthorizeExternal: input.authorizeExternal, AuthorizeLocal: input.authorizeLocal, Runtime: input.runtime, Role: input.role, Complexity: input.complexity, Capabilities: strings.Split(input.capabilities, ","), RuntimePreview: input.runtimePreview}
		value.Automatic = input.automatic
		switch operation {
		case workflowStartAction:
			return service.WorkflowStart(ctx, value)
		case workflowAdvanceAction:
			return service.WorkflowAdvance(ctx, value)
		case workflowFactAction:
			factService, ok := service.(LifecycleFactService)
			if !ok {
				return Result{Status: Failed, Category: "application_unavailable"}
			}
			return factService.WorkflowFact(ctx, value)
		case workflowResumeAction:
			return service.WorkflowResume(ctx, value)
		case workflowStatusAction:
			return service.WorkflowStatus(ctx, value)
		case workflowReconcileAction:
			return service.WorkflowReconcile(ctx, value)
		default:
			return service.WorkflowEvidence(ctx, value)
		}
	case codexInstallAction:
		return service.RuntimeCodexInstall(ctx)
	case codexStatusAction:
		return service.RuntimeCodexStatus(ctx)
	case claudeInstallAction:
		return service.RuntimeClaudeInstall(ctx)
	case claudeStatusAction:
		return service.RuntimeClaudeStatus(ctx)
	case firstRunAction:
		return service.FirstRun(ctx)
	}
	return Result{Status: Failed, Category: "invalid_command"}
}

type event struct {
	NextAction        string                      `json:"nextAction,omitempty"`
	Operation         action                      `json:"operation"`
	Status            Status                      `json:"status"`
	Category          string                      `json:"category"`
	Project           *ProjectView                `json:"project,omitempty"`
	WorkItem          *WorkItemView               `json:"workItem,omitempty"`
	Workflow          *WorkflowView               `json:"workflow,omitempty"`
	Setup             *projectapp.SetupPreview    `json:"setup,omitempty"`
	Runtime           *RuntimeView                `json:"runtime,omitempty"`
	RuntimeResolution *runtimeapplication.Preview `json:"runtimeResolution,omitempty"`
	PreviewDigest     string                      `json:"previewDigest,omitempty"`
	Projection        *workflow.ProjectionPreview `json:"projection,omitempty"`
}

type outputMode string

const (
	humanOutput outputMode = "human"
	jsonOutput  outputMode = "json"
)

func parseOutputMode(args []string) (outputMode, []string) {
	if len(args) > 0 && args[0] == "--json" {
		return jsonOutput, args[1:]
	}
	if len(args) > 0 && args[0] == "--human" {
		return humanOutput, args[1:]
	}
	return humanOutput, args
}

func eventFrom(operation action, result Result) event {
	return event{Operation: operation, Status: result.Status, Category: result.Category, Project: result.Project, WorkItem: result.WorkItem, Workflow: result.Workflow, Setup: result.Setup, Runtime: result.Runtime, RuntimeResolution: result.RuntimeResolution, PreviewDigest: result.PreviewDigest, Projection: result.Projection}
}

func emit(writer io.Writer, mode outputMode, value event) int {
	if writer == nil {
		return ExitFailure
	}
	if value.Status == Failed && (value.Category == "invalid_command" || value.Category == "invalid_input" || value.Category == "missing_required_input") {
		value.NextAction = helpNext(writer, "Review supported inputs")
	}
	if mode == jsonOutput {
		_ = json.NewEncoder(writer).Encode(value)
	} else {
		emitHuman(writer, value)
	}
	if value.Status == Succeeded {
		return ExitSuccess
	}
	if value.Status == Cancelled {
		return ExitCancelled
	}
	return ExitFailure
}

func emitHuman(writer io.Writer, value event) {
	_, _ = io.WriteString(writer, string(value.Status)+": "+value.Category+" ("+string(value.Operation)+")\n")
	if value.NextAction != "" {
		_, _ = io.WriteString(writer, "next: "+value.NextAction+"\n")
	}
	if value.Project != nil {
		_, _ = io.WriteString(writer, "project "+value.Project.Slug+" ["+value.Project.ID+"]\n")
		for _, repository := range value.Project.Repositories {
			_, _ = io.WriteString(writer, "repository "+repository.Key+" "+strconv.Quote(repository.Path)+"\n")
		}
	}
	if value.WorkItem != nil {
		_, _ = io.WriteString(writer, "work-item "+value.WorkItem.URL+" ["+value.WorkItem.State+"] project="+value.WorkItem.ProjectID+" repository-key="+value.WorkItem.RepositoryKey+" provider="+value.WorkItem.Provider+" resource="+value.WorkItem.Resource+" external-id="+value.WorkItem.ExternalID+"\n")
	}
	if value.Workflow != nil {
		_, _ = io.WriteString(writer, "execution "+value.Workflow.ExecutionID+" "+value.Workflow.Status+" current="+value.Workflow.CurrentGate+" revision="+strconv.FormatUint(value.Workflow.Revision, 10)+"\n")
	}
	if value.Runtime != nil {
		emitRuntimeHuman(writer, *value.Runtime)
	}
}

func category(value string) *string { return &value }

type repositoryFlags []string

func (r *repositoryFlags) String() string { return strings.Join(*r, ",") }
func (r *repositoryFlags) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func parseRepositories(values []string) ([]RepositoryInput, bool) {
	result := make([]RepositoryInput, 0, len(values))
	for _, value := range values {
		key, path, ok := strings.Cut(value, "=")
		if !ok || key == "" || path == "" {
			return nil, false
		}
		result = append(result, RepositoryInput{Key: key, Path: path})
	}
	return result, true
}

func completeWorkItemCreate(values requestInput) bool {
	if values.project == "" || values.repository == "" || values.providerRepository == "" {
		return false
	}
	for name, supplied := range map[string]string{
		"problem": values.problem, "desired_outcome": values.desiredOutcome,
		"context": values.context, "scope": values.scope, "constraints": values.constraints,
		"non_goals": values.nonGoals, "acceptance_expectations": values.acceptance,
	} {
		if name == "problem" && values.intent != "" {
			continue
		}
		if supplied == "" && values.elaboratedSections[name] == "" {
			return false
		}
	}
	return true
}

func runInteractiveSelectors(ctx context.Context, mode outputMode, operation action, values requestInput, service Service, source provenance.Value, input io.Reader, stdout, prompts io.Writer) int {
	scanner := bufio.NewScanner(input)
	fields := []struct {
		value  *string
		prompt string
	}{
		{&values.project, "Project UUID or slug: "},
		{&values.repository, "Project repository key: "},
		{&values.workItem, "Work Item selector (github:owner/repository#number): "},
	}
	if knownWorkflow(operation) && operation != workflowStartAction {
		fields = append(fields, struct {
			value  *string
			prompt string
		}{&values.execution, "Execution ID: "})
	}
	for _, field := range fields {
		if *field.value != "" {
			continue
		}
		value, ok := readPromptLine(scanner, prompts, field.prompt, true)
		if !ok {
			return emitParserFailure(stdout, mode, operation, "missing_required_input", source)
		}
		*field.value = value
	}
	if issue := selectorRequestIssue(operation, values); issue != "" {
		return emitParserFailure(stdout, mode, operation, issue, source)
	}
	return emitResponse(stdout, mode, operation, dispatch(ctx, operation, values, service))
}

func runInteractiveWorkItemCreate(ctx context.Context, mode outputMode, values requestInput, service Service, input io.Reader, stdout, prompts io.Writer) int {
	scanner := bufio.NewScanner(input)
	fields := []struct {
		value  *string
		prompt string
	}{
		{&values.project, "Project UUID or slug: "},
		{&values.repository, "Project repository key: "},
		{&values.providerRepository, "GitHub repository owner/name: "},
	}
	for _, field := range fields {
		if *field.value != "" {
			continue
		}
		value, ok := readPromptLine(scanner, prompts, field.prompt, true)
		if !ok {
			return emitResponse(stdout, mode, workItemCreateAction, service.WorkItemCreate(ctx, WorkItemInput{Project: values.project, Repository: values.repository, ProviderRepository: values.providerRepository, Cancelled: true}))
		}
		*field.value = value
	}
	if values.intent == "" && values.problem == "" && values.elaboratedSections["problem"] == "" {
		value, ok := readPromptLine(scanner, prompts, "Intent or problem: ", true)
		if !ok {
			return emitResponse(stdout, mode, workItemCreateAction, service.WorkItemCreate(ctx, WorkItemInput{Project: values.project, Repository: values.repository, ProviderRepository: values.providerRepository, Cancelled: true}))
		}
		values.intent = value
	}
	if values.itemType == "" {
		value, ok := readPromptLine(scanner, prompts, "Work Item type (story, bug, task) [task]: ", false)
		if !ok {
			return emitResponse(stdout, mode, workItemCreateAction, service.WorkItemCreate(ctx, workItemInput(values, true)))
		}
		values.itemType = value
		if values.itemType == "" {
			values.itemType = "task"
		}
	}
	sections := []struct {
		value        *string
		name, prompt string
	}{
		{&values.desiredOutcome, "desired_outcome", "What should work differently when this is solved? "},
		{&values.context, "context", "Where or when does this happen? Include relevant background (or say none): "},
		{&values.scope, "scope", "What should this change cover? "},
		{&values.constraints, "constraints", "What limits or existing behavior must we preserve (or say none)? "},
		{&values.nonGoals, "non_goals", "What should we explicitly leave out (or say none)? "},
		{&values.acceptance, "acceptance_expectations", "How could we check that the problem is solved? "},
	}
	if values.itemType == "story" {
		sections = append(sections, struct {
			value        *string
			name, prompt string
		}{&values.beneficiary, "beneficiary", "Who benefits from this story? "}, struct {
			value        *string
			name, prompt string
		}{&values.value, "value", "Concrete user/product benefit beyond implementation: "})
	}
	for _, field := range sections {
		if *field.value != "" || values.elaboratedSections[field.name] != "" {
			continue
		}
		value, ok := readPromptLine(scanner, prompts, field.prompt, true)
		if !ok {
			return emitResponse(stdout, mode, workItemCreateAction, service.WorkItemCreate(ctx, workItemInput(values, true)))
		}
		*field.value = value
	}
	preview := service.WorkItemCreate(ctx, workItemInput(values, false))
	if preview.Draft == nil || preview.Completion == nil || preview.Completion.Status() != completion.Success {
		return emitResponse(stdout, mode, workItemCreateAction, preview)
	}
	if prompts != nil {
		wire, err := marshalWorkItemValue(preview.Draft, true)
		if err != nil {
			return ExitFailure
		}
		_, _ = prompts.Write(wire)
	}
	answer, ok := readPromptLine(scanner, prompts, "Publish this create-attempt fence, create this exact GitHub Issue, and publish its local link? [yes/no]: ", false)
	if !ok || answer != "yes" {
		cancelled := workItemInput(values, true)
		return emitResponse(stdout, mode, workItemCreateAction, service.WorkItemCreate(ctx, cancelled))
	}
	authorized := workItemInput(values, false)
	authorized.PreviewDigest = preview.Draft.Digest
	authorized.AuthorizeExternal = true
	return emitResponse(stdout, mode, workItemCreateAction, service.WorkItemCreate(ctx, authorized))
}

func workItemInput(values requestInput, cancelled bool) WorkItemInput {
	return WorkItemInput{
		Type: values.itemType, Beneficiary: values.beneficiary, Value: values.value, Classification: append([]string(nil), values.classification...),
		Project: values.project, Repository: values.repository, ProviderRepository: values.providerRepository,
		ElaboratedSections: values.elaboratedSections,
		Intent:             values.intent, Problem: values.problem, DesiredOutcome: values.desiredOutcome, Context: values.context,
		Scope: values.scope, Constraints: values.constraints, NonGoals: values.nonGoals, Acceptance: values.acceptance,
		PreviewDigest: values.previewDigest, AuthorizeExternal: values.authorizeExternal, Cancelled: cancelled,
	}
}

func runInteractiveConfiguration(ctx context.Context, mode outputMode, args []string, service Service, input io.Reader, stdout, prompts io.Writer) int {
	values, ok := flags(configureAction, args)
	if !ok {
		return emit(stdout, mode, event{Operation: configureAction, Status: Failed, Category: "invalid_input"})
	}
	initial, ok := createConfigureInput(values)
	if !ok {
		return emit(stdout, mode, event{Operation: configureAction, Status: Failed, Category: "invalid_input"})
	}
	// An explicit `none` was supplied; the prompt asks only for missing intent.
	if values.workItemProvider == "none" {
		initial.WorkItemProvider = "none"
	}
	scanner := bufio.NewScanner(input)
	configuration, ok := promptConfiguration(scanner, prompts, initial)
	if !ok {
		return emit(stdout, mode, event{Operation: configureAction, Status: Failed, Category: "missing_required_input"})
	}
	if configuration.AuthorizeLocal {
		return emitResponse(stdout, mode, configureAction, service.Configure(ctx, configuration))
	}
	preview := service.Configure(ctx, configuration)
	// Guided bootstrap asks only for unresolved intent: an ambiguous remote is
	// resolved by an explicit operator choice, then the proposal is rebuilt.
	if preview.Setup != nil && chooseAmbiguousRemotes(scanner, prompts, *preview.Setup, &configuration) {
		preview = service.Configure(ctx, configuration)
	}
	if preview.Setup == nil || preview.Completion == nil || preview.Completion.Status() != completion.Success {
		return emitResponse(stdout, mode, configureAction, preview)
	}
	if prompts != nil {
		wire, _ := json.MarshalIndent(preview.Setup, "", "  ")
		_, _ = prompts.Write(append(wire, '\n'))
	}
	answer, ok := readPromptLine(scanner, prompts, "Publish this exact proposal? [yes/no]: ", false)
	if !ok || answer != "yes" {
		configuration.ProjectID = preview.Setup.ProjectID
		configuration.PreviewDigest = ""
		configuration.AuthorizeLocal = true
		return emitResponse(stdout, mode, configureAction, service.Configure(ctx, configuration))
	}
	configuration.ProjectID = preview.Setup.ProjectID
	configuration.PreviewDigest = preview.Setup.Digest
	configuration.AuthorizeLocal = true
	return emitResponse(stdout, mode, configureAction, service.Configure(ctx, configuration))
}

func chooseAmbiguousRemotes(scanner *bufio.Scanner, prompts io.Writer, setup projectapp.SetupPreview, configuration *ConfigureInput) bool {
	chosen := false
	for _, blocker := range setup.Blockers {
		if blocker.Code != projectapp.BlockerRemoteAmbiguous {
			continue
		}
		for _, repository := range setup.Repositories {
			if repository.Key != blocker.Subject || repository.Remote == nil {
				continue
			}
			for index, candidate := range repository.Remote.Candidates {
				if prompts != nil {
					_, _ = io.WriteString(prompts, strconv.Itoa(index+1)+") "+candidate.Locator+"\n")
				}
			}
			answer, ok := readPromptLine(scanner, prompts, "Remote for "+repository.Key+" (number, locator, or none): ", true)
			if !ok {
				return chosen
			}
			if number, err := strconv.Atoi(answer); err == nil && number >= 1 && number <= len(repository.Remote.Candidates) {
				answer = repository.Remote.Candidates[number-1].Locator
			}
			if configuration.RepositoryRemotes == nil {
				configuration.RepositoryRemotes = map[string]string{}
			}
			configuration.RepositoryRemotes[repository.Key] = answer
			chosen = true
		}
	}
	return chosen
}

func promptConfiguration(scanner *bufio.Scanner, prompts io.Writer, current ConfigureInput) (ConfigureInput, bool) {
	var ok bool
	if current.Slug == "" {
		current.Slug, ok = readPromptLine(scanner, prompts, "Project slug: ", true)
		if !ok {
			return ConfigureInput{}, false
		}
	}
	if current.Name == "" {
		current.Name, ok = readPromptLine(scanner, prompts, "Project name: ", true)
		if !ok {
			return ConfigureInput{}, false
		}
	}
	if len(current.Repositories) == 0 {
		for {
			value, read := readPromptLine(scanner, prompts, "Repository key=absolute-path (blank to finish): ", false)
			if !read {
				return ConfigureInput{}, false
			}
			if value == "" {
				break
			}
			repositories, valid := parseBootstrapRepositories([]string{value})
			if !valid {
				return ConfigureInput{}, false
			}
			current.Repositories = append(current.Repositories, repositories[0])
		}
		if len(current.Repositories) == 0 {
			return ConfigureInput{}, false
		}
	}
	if current.WorkItemProvider == "" {
		current.WorkItemProvider, ok = readPromptLine(scanner, prompts, "Work Item provider (github or none): ", true)
		if !ok {
			return ConfigureInput{}, false
		}
		if current.WorkItemProvider == "none" {
			current.WorkItemProvider = ""
		}
	}
	return current, true
}

func readPromptLine(scanner *bufio.Scanner, output io.Writer, prompt string, required bool) (string, bool) {
	if output != nil {
		_, _ = io.WriteString(output, prompt)
	}
	if !scanner.Scan() {
		return "", false
	}
	value := strings.TrimSpace(scanner.Text())
	return value, !required || value != ""
}

// UnavailableService makes the executable fail closed until its composition root
// receives the authorized application use cases in subsequent POC delivery work.
type UnavailableService struct{ source provenance.Value }

func NewUnavailableService(source provenance.Value) UnavailableService {
	return UnavailableService{source: source}
}

func (UnavailableService) Init(context.Context, InitInput) Result {
	return unavailable()
}
func (s UnavailableService) Validate(context.Context, ProjectInput) Result {
	return s.canonicalUnavailable("Project validation unavailable")
}
func (UnavailableService) Reopen(context.Context, ProjectInput) Result {
	return unavailable()
}
func (UnavailableService) Update(context.Context, UpdateInput) Result {
	return unavailable()
}
func (UnavailableService) Install(context.Context, InstallInput) Result { return unavailable() }
func (UnavailableService) RuntimeCodexInstall(context.Context) Result   { return unavailable() }
func (UnavailableService) RuntimeCodexStatus(context.Context) Result    { return unavailable() }
func (UnavailableService) RuntimeClaudeInstall(context.Context) Result  { return unavailable() }
func (UnavailableService) RuntimeClaudeStatus(context.Context) Result   { return unavailable() }
func (UnavailableService) FirstRun(context.Context) Result              { return unavailable() }
func (UnavailableService) Resolve(context.Context, ResolveInput) Result { return unavailable() }
func (s UnavailableService) Show(context.Context, ResolveInput) Result {
	return s.canonicalUnavailable("Project inspection unavailable")
}
func (s UnavailableService) List(context.Context) Result {
	return s.canonicalUnavailable("Project listing unavailable")
}
func (UnavailableService) Configure(context.Context, ConfigureInput) Result     { return unavailable() }
func (UnavailableService) WorkItemCreate(context.Context, WorkItemInput) Result { return unavailable() }
func (UnavailableService) WorkItemSelect(context.Context, WorkItemInput) Result { return unavailable() }
func (UnavailableService) WorkItemShow(context.Context, WorkItemInput) Result   { return unavailable() }
func (UnavailableService) WorkItemComment(context.Context, WorkItemInput) Result {
	return unavailable()
}
func (UnavailableService) WorkItemComplete(context.Context, WorkItemInput) Result {
	return unavailable()
}
func (UnavailableService) WorkflowStart(context.Context, WorkflowInput) Result { return unavailable() }
func (UnavailableService) WorkflowAdvance(context.Context, WorkflowInput) Result {
	return unavailable()
}
func (UnavailableService) WorkflowResume(context.Context, WorkflowInput) Result { return unavailable() }
func (UnavailableService) WorkflowStatus(context.Context, WorkflowInput) Result { return unavailable() }
func (UnavailableService) WorkflowEvidence(context.Context, WorkflowInput) Result {
	return unavailable()
}
func (UnavailableService) WorkflowReconcile(context.Context, WorkflowInput) Result {
	return unavailable()
}
func unavailable() Result { return Result{Status: Failed, Category: "application_unavailable"} }

func (s UnavailableService) canonicalUnavailable(message string) Result {
	if !s.source.Valid() {
		return unavailable()
	}
	statement, err := provenance.NewText(message, provenance.AxiomAuthored)
	if err != nil {
		return unavailable()
	}
	result, err := completion.New(completion.Facts{Failed: true}, statement, nil, provenance.Text{}, "", s.source)
	if err != nil {
		return unavailable()
	}
	return Result{Completion: &result}
}
