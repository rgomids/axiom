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
	"github.com/rgomids/axiom/internal/provenance"
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
	Resolve(context.Context, ResolveInput) Result
	Show(context.Context, ResolveInput) Result
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
}

type InitInput struct {
	Slug string
	Name string
}

type ProjectInput struct{ Slug string }

type UpdateInput struct {
	Slug string
	Name string
}
type InstallInput struct{ Source string }
type ResolveInput struct{ Selector string }
type RepositoryInput struct{ Key, Path string }
type ConfigureInput struct {
	Slug, Name   string
	Repositories []RepositoryInput
}
type WorkItemInput struct {
	Project, Repository, Title, Body, Message string
	Number                                    int
	AuthorizeExternal                         bool
}
type WorkflowInput struct {
	Project, Repository, Gate, Outcome, Reference string
	Number                                        int
	AuthorizeExternal                             bool
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
	Status     Status
	Category   string
	Project    *ProjectView
	WorkItem   *WorkItemView
	Workflow   *WorkflowView
	Completion *completion.Result
}

type RepositoryView struct {
	Key  string `json:"key"`
	Path string `json:"path"`
}
type ProjectView struct {
	ID           string           `json:"id"`
	Slug         string           `json:"slug"`
	Source       string           `json:"source"`
	Repositories []RepositoryView `json:"repositories"`
}
type WorkItemView struct {
	ProjectID     string `json:"projectId"`
	RepositoryKey string `json:"repositoryKey"`
	Repository    string `json:"repository"`
	Number        int    `json:"number"`
	URL           string `json:"url"`
	State         string `json:"state"`
}
type WorkflowStepView struct {
	Gate      string `json:"gate"`
	Status    string `json:"status"`
	Reference string `json:"reference,omitempty"`
	Digest    string `json:"digest,omitempty"`
}
type WorkflowView struct {
	Status         string             `json:"status"`
	CurrentGate    string             `json:"currentGate,omitempty"`
	RepositoryKey  string             `json:"repositoryKey"`
	RepositoryPath string             `json:"repositoryPath"`
	WorkItem       int                `json:"workItem"`
	Steps          []WorkflowStepView `json:"steps"`
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
	mode, args := parseOutputMode(args)
	if service == nil {
		return emit(stdout, mode, event{Operation: "unknown", Status: Failed, Category: "application_unavailable"})
	}
	if len(args) == 2 && args[0] == "project" && args[1] == "configure" && stdin != nil {
		input, ok := promptConfiguration(stdin, stderr)
		if !ok {
			return emit(stdout, mode, event{Operation: configureAction, Status: Failed, Category: "missing_required_input"})
		}
		response := service.Configure(ctx, input)
		return emit(stdout, mode, eventFrom(configureAction, response))
	}
	operation, input, result := request(args, service)
	if result != nil {
		if operation == validateAction || operation == showAction {
			return emitParserFailure(stdout, mode, operation, *result, source)
		}
		return emit(stdout, mode, event{Operation: operation, Status: Failed, Category: *result})
	}
	response := dispatch(ctx, operation, input, service)
	return emitResponse(stdout, mode, operation, response)
}

func emitParserFailure(writer io.Writer, mode outputMode, operation action, issue string, source provenance.Value) int {
	message, next := parserFailureText(operation, issue)
	statement, err := provenance.NewText(message, provenance.AxiomAuthored)
	if err != nil {
		return ExitFailure
	}
	nextAction, err := provenance.NewText(next, provenance.AxiomAuthored)
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
	if operation == validateAction && issue == "missing_required_input" {
		return "Project slug is required", "Provide a Project slug and retry validation"
	}
	if operation == showAction && issue == "missing_required_input" {
		return "Project selector is required", "Provide a Project UUID or slug and retry inspection"
	}
	if operation == validateAction {
		return "Project validation input is invalid", "Review supported validation flags and retry"
	}
	return "Project inspection input is invalid", "Review supported inspection flags and retry"
}

func emitResponse(writer io.Writer, mode outputMode, operation action, response Result) int {
	if response.Completion != nil {
		return emitCompletion(writer, mode, *response.Completion)
	}
	return emit(writer, mode, eventFrom(operation, response))
}

type action string

const (
	initAction             action = "init"
	validateAction         action = "validate"
	reopenAction           action = "reopen"
	updateAction           action = "update"
	installAction          action = "install"
	resolveAction          action = "resolve"
	showAction             action = "show"
	configureAction        action = "configure"
	workItemCreateAction   action = "work_item_create"
	workItemSelectAction   action = "work_item_select"
	workItemShowAction     action = "work_item_show"
	workItemCommentAction  action = "work_item_comment"
	workItemCompleteAction action = "work_item_complete"
	workflowStartAction    action = "workflow_start"
	workflowAdvanceAction  action = "workflow_advance"
	workflowResumeAction   action = "workflow_resume"
	workflowStatusAction   action = "workflow_status"
	workflowEvidenceAction action = "workflow_evidence"
	codexInstallAction     action = "runtime_codex_install"
	codexStatusAction      action = "runtime_codex_status"
)

type requestInput struct {
	slug                                      string
	name                                      string
	source                                    string
	selector                                  string
	repositories                              repositoryFlags
	project, repository, title, body, message string
	gate, outcome, reference                  string
	number                                    int
	authorizeExternal                         bool
}

func request(args []string, service Service) (action, requestInput, *string) {
	if service == nil {
		return "unknown", requestInput{}, category("application_unavailable")
	}
	if len(args) == 3 && args[0] == "runtime" && args[1] == "codex" {
		operation := action("runtime_codex_" + args[2])
		if operation == codexInstallAction || operation == codexStatusAction {
			return operation, requestInput{}, nil
		}
		return "unknown", requestInput{}, category("invalid_command")
	}
	if len(args) >= 2 && args[0] == "work-item" {
		operation := action("work_item_" + args[1])
		if !knownWorkItem(operation) {
			return "unknown", requestInput{}, category("invalid_command")
		}
		values, ok := workItemFlags(operation, args[2:])
		if !ok {
			return operation, requestInput{}, category("invalid_input")
		}
		if values.project == "" || values.repository == "" || operation == workItemCreateAction && values.title == "" || operation != workItemCreateAction && values.number <= 0 || operation == workItemCommentAction && values.message == "" {
			return operation, values, category("missing_required_input")
		}
		return operation, values, nil
	}
	if len(args) >= 2 && args[0] == "workflow" {
		operation := action("workflow_" + args[1])
		if !knownWorkflow(operation) {
			return "unknown", requestInput{}, category("invalid_command")
		}
		values, ok := workflowFlags(operation, args[2:])
		if !ok {
			return operation, requestInput{}, category("invalid_input")
		}
		if values.project == "" || values.repository == "" || values.number <= 0 || operation == workflowAdvanceAction && (values.gate == "" || values.outcome == "") {
			return operation, values, category("missing_required_input")
		}
		return operation, values, nil
	}
	if len(args) < 2 || args[0] != "project" {
		return "unknown", requestInput{}, category("invalid_command")
	}
	operation := action(args[1])
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
	if (operation == resolveAction || operation == showAction) && values.selector == "" {
		return operation, values, category("missing_required_input")
	}
	if operation == configureAction && (values.slug == "" || values.name == "" || len(values.repositories) == 0) {
		return operation, values, category("missing_required_input")
	}
	if operation != initAction && operation != installAction && operation != resolveAction && operation != showAction && values.slug == "" {
		return operation, values, category("missing_required_input")
	}
	if operation == updateAction && values.name == "" {
		return operation, values, category("missing_required_input")
	}
	return operation, values, nil
}

func flags(operation action, args []string) (requestInput, bool) {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var values requestInput
	set.StringVar(&values.slug, "slug", "", "")
	if operation == initAction || operation == updateAction {
		set.StringVar(&values.name, "name", "", "")
	}
	if operation == installAction {
		set.StringVar(&values.source, "source", "", "")
	}
	if operation == resolveAction || operation == showAction {
		set.StringVar(&values.selector, "selector", "", "")
	}
	if operation == configureAction {
		set.StringVar(&values.name, "name", "", "")
		set.Var(&values.repositories, "repository", "")
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return requestInput{}, false
	}
	return values, true
}

func workItemFlags(operation action, args []string) (requestInput, bool) {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var values requestInput
	set.StringVar(&values.project, "project", "", "")
	set.StringVar(&values.repository, "repository", "", "")
	set.BoolVar(&values.authorizeExternal, "authorize-external", false, "")
	if operation == workItemCreateAction {
		set.StringVar(&values.title, "title", "", "")
		set.StringVar(&values.body, "body", "", "")
	} else {
		set.IntVar(&values.number, "number", 0, "")
	}
	if operation == workItemCommentAction {
		set.StringVar(&values.message, "message", "", "")
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return requestInput{}, false
	}
	return values, true
}

func knownWorkItem(operation action) bool {
	return operation == workItemCreateAction || operation == workItemSelectAction || operation == workItemShowAction || operation == workItemCommentAction || operation == workItemCompleteAction
}

func workflowFlags(operation action, args []string) (requestInput, bool) {
	set := flag.NewFlagSet(string(operation), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var values requestInput
	set.StringVar(&values.project, "project", "", "")
	set.StringVar(&values.repository, "repository", "", "")
	set.IntVar(&values.number, "number", 0, "")
	set.BoolVar(&values.authorizeExternal, "authorize-external", false, "")
	if operation == workflowAdvanceAction {
		set.StringVar(&values.gate, "gate", "", "")
		set.StringVar(&values.outcome, "outcome", "", "")
		set.StringVar(&values.reference, "reference", "", "")
	}
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return requestInput{}, false
	}
	return values, true
}

func knownWorkflow(operation action) bool {
	return operation == workflowStartAction || operation == workflowAdvanceAction || operation == workflowResumeAction || operation == workflowStatusAction || operation == workflowEvidenceAction
}

func known(operation action) bool {
	return operation == initAction || operation == validateAction || operation == reopenAction || operation == updateAction || operation == installAction || operation == resolveAction || operation == showAction || operation == configureAction
}

func dispatch(ctx context.Context, operation action, input requestInput, service Service) Result {
	if err := ctx.Err(); err != nil {
		if operation == validateAction {
			return service.Validate(ctx, ProjectInput{Slug: input.slug})
		}
		if operation == showAction {
			return service.Show(ctx, ResolveInput{Selector: input.selector})
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
		return service.Install(ctx, InstallInput{Source: input.source})
	case resolveAction:
		return service.Resolve(ctx, ResolveInput{Selector: input.selector})
	case showAction:
		return service.Show(ctx, ResolveInput{Selector: input.selector})
	case configureAction:
		repositories, ok := parseRepositories(input.repositories)
		if !ok {
			return Result{Status: Failed, Category: "invalid_input"}
		}
		return service.Configure(ctx, ConfigureInput{Slug: input.slug, Name: input.name, Repositories: repositories})
	case workItemCreateAction, workItemSelectAction, workItemShowAction, workItemCommentAction, workItemCompleteAction:
		value := WorkItemInput{Project: input.project, Repository: input.repository, Title: input.title, Body: input.body, Message: input.message, Number: input.number, AuthorizeExternal: input.authorizeExternal}
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
	case workflowStartAction, workflowAdvanceAction, workflowResumeAction, workflowStatusAction, workflowEvidenceAction:
		value := WorkflowInput{Project: input.project, Repository: input.repository, Number: input.number, Gate: input.gate, Outcome: input.outcome, Reference: input.reference, AuthorizeExternal: input.authorizeExternal}
		switch operation {
		case workflowStartAction:
			return service.WorkflowStart(ctx, value)
		case workflowAdvanceAction:
			return service.WorkflowAdvance(ctx, value)
		case workflowResumeAction:
			return service.WorkflowResume(ctx, value)
		case workflowStatusAction:
			return service.WorkflowStatus(ctx, value)
		default:
			return service.WorkflowEvidence(ctx, value)
		}
	case codexInstallAction:
		return service.RuntimeCodexInstall(ctx)
	case codexStatusAction:
		return service.RuntimeCodexStatus(ctx)
	}
	return Result{Status: Failed, Category: "invalid_command"}
}

type event struct {
	Operation action        `json:"operation"`
	Status    Status        `json:"status"`
	Category  string        `json:"category"`
	Project   *ProjectView  `json:"project,omitempty"`
	WorkItem  *WorkItemView `json:"workItem,omitempty"`
	Workflow  *WorkflowView `json:"workflow,omitempty"`
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
	return event{Operation: operation, Status: result.Status, Category: result.Category, Project: result.Project, WorkItem: result.WorkItem, Workflow: result.Workflow}
}

func emit(writer io.Writer, mode outputMode, value event) int {
	if writer == nil {
		return ExitFailure
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
	if value.Project != nil {
		_, _ = io.WriteString(writer, "project "+value.Project.Slug+" ["+value.Project.ID+"]\n")
		for _, repository := range value.Project.Repositories {
			_, _ = io.WriteString(writer, "repository "+repository.Key+" "+strconv.Quote(repository.Path)+"\n")
		}
	}
	if value.WorkItem != nil {
		_, _ = io.WriteString(writer, "work-item "+value.WorkItem.URL+" ["+value.WorkItem.State+"] project="+value.WorkItem.ProjectID+" repository-key="+value.WorkItem.RepositoryKey+" provider-repository="+value.WorkItem.Repository+"\n")
	}
	if value.Workflow != nil {
		_, _ = io.WriteString(writer, "workflow "+value.Workflow.Status+" current="+value.Workflow.CurrentGate+" repository="+strconv.Quote(value.Workflow.RepositoryPath)+"\n")
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

func promptConfiguration(input io.Reader, prompts io.Writer) (ConfigureInput, bool) {
	scanner := bufio.NewScanner(input)
	read := func(prompt string) (string, bool) {
		_, _ = io.WriteString(prompts, prompt)
		if !scanner.Scan() {
			return "", false
		}
		value := strings.TrimSpace(scanner.Text())
		return value, value != ""
	}
	slug, ok := read("Project slug: ")
	if !ok {
		return ConfigureInput{}, false
	}
	name, ok := read("Project name: ")
	if !ok {
		return ConfigureInput{}, false
	}
	key, ok := read("Repository key: ")
	if !ok {
		return ConfigureInput{}, false
	}
	path, ok := read("Repository path: ")
	if !ok {
		return ConfigureInput{}, false
	}
	return ConfigureInput{Slug: slug, Name: name, Repositories: []RepositoryInput{{Key: key, Path: path}}}, true
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
func (UnavailableService) Resolve(context.Context, ResolveInput) Result { return unavailable() }
func (s UnavailableService) Show(context.Context, ResolveInput) Result {
	return s.canonicalUnavailable("Project inspection unavailable")
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
