package main

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

var (
	buildVersion     = provenance.Development
	buildRevision    = provenance.Unavailable
	buildSourceState = string(provenance.Unknown)
	buildRelease     = "false"
)

func main() {
	source := currentProvenance()
	if format, ok := versionFormat(os.Args[1:]); ok {
		os.Exit(writeVersion(os.Stdout, format, source))
	}
	if len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		os.Exit(cli.Help(os.Stdout))
	}
	os.Exit(cli.RunInteractive(context.Background(), os.Args[1:], composeWithProvenance(source), os.Stdin, os.Stdout, os.Stderr))
}

func versionFormat(args []string) (cli.CompletionFormat, bool) {
	if len(args) == 1 && args[0] == "version" {
		return cli.CompletionHuman, true
	}
	if len(args) == 2 && args[1] == "version" && args[0] == "--json" {
		return cli.CompletionJSON, true
	}
	if len(args) == 2 && args[1] == "version" && args[0] == "--human" {
		return cli.CompletionHuman, true
	}
	return "", false
}

func writeVersion(output *os.File, format cli.CompletionFormat, source provenance.Value) int {
	result, ok := newCompletion(completion.Facts{Completed: true}, "Axiom build information", nil, "", source)
	if !ok {
		return cli.ExitFailure
	}
	return cli.WriteCompletion(output, format, *result.Completion)
}

func currentProvenance() provenance.Value {
	release, err := strconv.ParseBool(buildRelease)
	if err != nil {
		return unknownProvenance()
	}
	settings := make([]provenance.Setting, 0, 2)
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" || setting.Key == "vcs.modified" {
				settings = append(settings, provenance.Setting{Key: setting.Key, Value: setting.Value})
			}
		}
	}
	value, err := provenance.FromBuild(provenance.Build{Release: release, Version: buildVersion, Revision: buildRevision, SourceState: provenance.SourceState(buildSourceState)}, settings)
	if err != nil {
		return unknownProvenance()
	}
	return value
}

func unknownProvenance() provenance.Value {
	value, _ := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: provenance.Unavailable, SourceState: provenance.Unknown}, nil)
	return value
}

func compose() cli.Service {
	return composeWithProvenance(currentProvenance())
}

func composeWithProvenance(source provenance.Value) cli.Service {
	root, err := projectsRoot()
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	state, err := stateRoot()
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	if overlap, err := local.RootsOverlap(root, state); err != nil || overlap {
		return cli.NewUnavailableService(source)
	}
	store, err := local.NewPortableStore(root)
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	installation, err := local.NewInstallationStore(state)
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	codex, err := codexruntime.New(codexSkillsRoot())
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	workItems, err := local.NewWorkItemStore(state)
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	github, _ := workitem.NewGitHubAdapter(os.Getenv("AXIOM_GIT_BIN"), os.Getenv("AXIOM_GH_BIN"))
	workItemService := workitem.New(workItemResolver{installation}, github, github, workItems)
	workflows, err := local.NewWorkflowStore(state)
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	workflowService := workflow.New(workflowResolver{installation}, workflowWorkItems{workItemService}, workflows)
	return lifecycleService{lifecycle: projectapp.NewLifecycle(store, manifest.Codec{}, local.IdentityAllocator{}), installation: installation, codex: codex, workItems: workItemService, workflows: workflowService, projectsRoot: root, provenance: source}
}

func codexSkillsRoot() string {
	if override := os.Getenv("AXIOM_CODEX_SKILLS_ROOT"); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agents", "skills")
}

func projectsRoot() (string, error) {
	if override := os.Getenv("LINGO_PROJECTS_ROOT"); override != "" {
		if !filepath.IsAbs(override) {
			return "", projectapp.ErrUnsafe
		}
		return filepath.Clean(override), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".axiom", "projects"), nil
}
func stateRoot() (string, error) {
	if override, set := os.LookupEnv("LINGO_STATE_ROOT"); set {
		if !filepath.IsAbs(override) || filepath.Clean(override) == string(filepath.Separator) {
			return "", projectapp.ErrUnsafe
		}
		return filepath.Clean(override), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return local.NativeStateRoot(runtime.GOOS, home, os.Getenv("XDG_STATE_HOME"))
}

type lifecycleService struct {
	lifecycle    projectapp.Lifecycle
	installation local.InstallationStore
	codex        codexruntime.Service
	workItems    workitem.Service
	workflows    workflow.Service
	projectsRoot string
	provenance   provenance.Value
}

type workItemResolver struct{ installation local.InstallationStore }

type workflowResolver struct{ installation local.InstallationStore }
type workflowWorkItems struct{ service workitem.Service }

func (r workItemResolver) Resolve(ctx context.Context, selector string) (workitem.Project, string) {
	resolved := r.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return workitem.Project{}, resolved.Category
	}
	project := workitem.Project{ID: resolved.Project.ID, Repositories: make([]workitem.Repository, 0, len(resolved.Project.Repositories))}
	for _, repository := range resolved.Project.Repositories {
		project.Repositories = append(project.Repositories, workitem.Repository{Key: repository.Key, Path: repository.Path})
	}
	return project, ""
}

func (r workflowResolver) Resolve(ctx context.Context, selector string) (workflow.Project, string) {
	resolved := r.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return workflow.Project{}, resolved.Category
	}
	project := workflow.Project{ID: resolved.Project.ID, Repositories: make([]workflow.Repository, 0, len(resolved.Project.Repositories))}
	for _, repository := range resolved.Project.Repositories {
		project.Repositories = append(project.Repositories, workflow.Repository{Key: repository.Key, Path: repository.Path})
	}
	return project, ""
}

func (w workflowWorkItems) Available(ctx context.Context, project, repository string, number int) bool {
	result := w.service.Show(ctx, workitem.Target{ProjectSelector: project, RepositoryKey: repository}, number)
	return result.Status == workitem.Succeeded && result.Link.State == "OPEN"
}

func (w workflowWorkItems) Complete(ctx context.Context, project, repository string, number int, authorized bool) workflow.WorkItemCompletion {
	result := w.service.Complete(ctx, workitem.Target{ProjectSelector: project, RepositoryKey: repository}, number, authorized)
	return workflow.WorkItemCompletion{
		Category: result.Category,
		WorkItem: workflow.WorkItem{
			ProjectID:          result.Link.ProjectID,
			RepositoryKey:      result.Link.RepositoryKey,
			ProviderRepository: result.Link.ProviderRepository,
			Number:             result.Link.Number,
			URL:                result.Link.URL,
			State:              result.Link.State,
		},
	}
}

func (s lifecycleService) RuntimeCodexInstall(ctx context.Context) cli.Result {
	return runtimeResult(s.codex.Install(ctx))
}
func (s lifecycleService) RuntimeCodexStatus(ctx context.Context) cli.Result {
	return runtimeResult(s.codex.Inspect(ctx))
}

func runtimeResult(result codexruntime.Result) cli.Result {
	status := cli.Failed
	if result.Status == codexruntime.Applied || result.Status == codexruntime.Unchanged || result.Status == codexruntime.Ready {
		status = cli.Succeeded
	}
	return cli.Result{Status: status, Category: result.Category}
}

func (s lifecycleService) Init(ctx context.Context, input cli.InitInput) cli.Result {
	return cliResult(s.lifecycle.Init(ctx, projectapp.InitRequest{Slug: input.Slug, Name: input.Name}))
}
func (s lifecycleService) Validate(ctx context.Context, input cli.ProjectInput) cli.Result {
	result := s.lifecycle.Validate(ctx, projectapp.ProjectRequest{Slug: input.Slug})
	if result.Status == projectapp.LifecycleApplied || result.Status == projectapp.LifecycleUnchanged {
		return canonicalCompletion(completion.Facts{Completed: true}, "Project is valid", nil, "", s.provenance)
	}
	if result.Status == projectapp.LifecycleCancelled {
		return canonicalCompletion(completion.Facts{WasInterrupted: true}, "Project validation was interrupted", nil, "Retry Project validation", s.provenance)
	}
	if result.Category == "storage_failure" || result.Category == "application_unavailable" {
		return canonicalCompletion(completion.Facts{Failed: true}, "Project validation failed", nil, "", s.provenance)
	}
	return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project state is invalid", nil, "Correct Project state and retry validation", s.provenance)
}
func (s lifecycleService) Reopen(ctx context.Context, input cli.ProjectInput) cli.Result {
	result := cliResult(s.lifecycle.Reopen(ctx, projectapp.ProjectRequest{Slug: input.Slug}))
	if result.Status != cli.Succeeded {
		return result
	}
	localResult := s.installation.Reopen(ctx, filepath.Join(s.projectsRoot, input.Slug))
	if localResult.Status == local.InstallationFailed || localResult.Status == local.InstallationConflict {
		return cli.Result{Status: cli.Failed, Category: localResult.Category}
	}
	return cli.Result{Status: cli.Succeeded, Category: localResult.Category}
}
func (s lifecycleService) Update(ctx context.Context, input cli.UpdateInput) cli.Result {
	return cliResult(s.lifecycle.Update(ctx, projectapp.UpdateRequest{Slug: input.Slug, Name: input.Name}))
}
func (s lifecycleService) Install(ctx context.Context, input cli.InstallInput) cli.Result {
	result := s.installation.Install(ctx, input.Source)
	status := cli.Failed
	if result.Status == local.InstallationApplied || result.Status == local.InstallationUnchanged {
		status = cli.Succeeded
	}
	return cli.Result{Status: status, Category: result.Category}
}
func (s lifecycleService) Resolve(ctx context.Context, input cli.ResolveInput) cli.Result {
	result := s.installation.Resolve(ctx, input.Selector)
	status := cli.Failed
	if result.Status == local.ResolutionFound {
		status = cli.Succeeded
	}
	response := cli.Result{Status: status, Category: result.Category}
	if result.Status == local.ResolutionFound {
		response.Project = projectView(result.Project)
	}
	return response
}

func (s lifecycleService) Show(ctx context.Context, input cli.ResolveInput) cli.Result {
	result := s.installation.Resolve(ctx, input.Selector)
	if result.Status != local.ResolutionFound {
		if result.Category == "cancelled" {
			return canonicalCompletion(completion.Facts{WasInterrupted: true}, "Project inspection was interrupted", nil, "Retry Project inspection", s.provenance)
		}
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project could not be resolved", nil, "Provide an explicit Project UUID or slug", s.provenance)
	}
	references := []string{"project:" + result.Project.ID}
	for _, repository := range result.Project.Repositories {
		references = append(references, "repository:"+repository.Key)
	}
	return canonicalCompletion(completion.Facts{Completed: true}, "Project resolved", references, "", s.provenance)
}

func projectView(project local.ResolvedProject) *cli.ProjectView {
	view := &cli.ProjectView{ID: project.ID, Slug: project.Slug, Source: project.Source, Repositories: make([]cli.RepositoryView, 0, len(project.Repositories))}
	for _, repository := range project.Repositories {
		view.Repositories = append(view.Repositories, cli.RepositoryView{Key: repository.Key, Path: repository.Path})
	}
	return view
}
func (s lifecycleService) Configure(ctx context.Context, input cli.ConfigureInput) cli.Result {
	keys := make([]string, 0, len(input.Repositories))
	bindings := make([]projectapp.RepositoryBinding, 0, len(input.Repositories))
	for _, repository := range input.Repositories {
		info, err := os.Lstat(repository.Path)
		if err != nil || !filepath.IsAbs(repository.Path) || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return cli.Result{Status: cli.Failed, Category: "repository_unavailable"}
		}
		keys = append(keys, repository.Key)
		bindings = append(bindings, projectapp.RepositoryBinding{
			RepositoryKey: repository.Key,
			ExplicitPath:  filepath.Clean(repository.Path),
			Observation:   projectapp.Observation{Availability: projectapp.Unverified, Basis: projectapp.NotChecked, ObservedAt: time.Time{}},
		})
	}
	portable := s.lifecycle.Configure(ctx, projectapp.ConfigureRequest{Slug: input.Slug, Name: input.Name, RepositoryKeys: keys})
	if portable.Status != projectapp.LifecycleApplied && portable.Status != projectapp.LifecycleUnchanged {
		return cliResult(portable)
	}
	localResult := s.installation.InstallWithBindings(ctx, filepath.Join(s.projectsRoot, input.Slug), bindings)
	if localResult.Status != local.InstallationApplied && localResult.Status != local.InstallationUnchanged {
		category := "local_configuration_failed"
		if portable.Status == projectapp.LifecycleApplied {
			category = "portable_committed_local_failed"
		}
		return cli.Result{Status: cli.Failed, Category: category}
	}
	category := "project_configured"
	if portable.Status == projectapp.LifecycleUnchanged && localResult.Status == local.InstallationUnchanged {
		category = "project_already_configured"
	}
	return cli.Result{Status: cli.Succeeded, Category: category}
}

func (s lifecycleService) WorkItemCreate(ctx context.Context, input cli.WorkItemInput) cli.Result {
	return workItemResult(s.workItems.Create(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository}, input.Title, input.Body, input.AuthorizeExternal))
}
func (s lifecycleService) WorkItemSelect(ctx context.Context, input cli.WorkItemInput) cli.Result {
	return workItemResult(s.workItems.Select(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository}, input.Number))
}
func (s lifecycleService) WorkItemShow(ctx context.Context, input cli.WorkItemInput) cli.Result {
	return workItemResult(s.workItems.Show(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository}, input.Number))
}
func (s lifecycleService) WorkItemComment(ctx context.Context, input cli.WorkItemInput) cli.Result {
	return workItemResult(s.workItems.Comment(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository}, input.Number, input.Message, input.AuthorizeExternal))
}
func (s lifecycleService) WorkItemComplete(ctx context.Context, input cli.WorkItemInput) cli.Result {
	return workItemResult(s.workItems.Complete(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository}, input.Number, input.AuthorizeExternal))
}

func workflowTarget(input cli.WorkflowInput) workflow.Target {
	return workflow.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, WorkItem: input.Number}
}

func (s lifecycleService) WorkflowStart(ctx context.Context, input cli.WorkflowInput) cli.Result {
	return workflowResult(s.workflows.Start(ctx, workflowTarget(input)))
}
func (s lifecycleService) WorkflowAdvance(ctx context.Context, input cli.WorkflowInput) cli.Result {
	return workflowResult(s.workflows.Advance(ctx, workflowTarget(input), input.Gate, input.Outcome, input.Reference, input.AuthorizeExternal))
}
func (s lifecycleService) WorkflowResume(ctx context.Context, input cli.WorkflowInput) cli.Result {
	return workflowResult(s.workflows.Resume(ctx, workflowTarget(input)))
}
func (s lifecycleService) WorkflowStatus(ctx context.Context, input cli.WorkflowInput) cli.Result {
	return workflowResult(s.workflows.Status(ctx, workflowTarget(input)))
}
func (s lifecycleService) WorkflowEvidence(ctx context.Context, input cli.WorkflowInput) cli.Result {
	result := s.workflows.Status(ctx, workflowTarget(input))
	if result.Status == workflow.Succeeded {
		result.Category = "workflow_evidence_ready"
	}
	return workflowResult(result)
}

func workflowResult(result workflow.Result) cli.Result {
	status := cli.Failed
	if result.Status == workflow.Succeeded {
		status = cli.Succeeded
	}
	response := cli.Result{Status: status, Category: result.Category}
	if result.State.ProjectID != "" {
		view := &cli.WorkflowView{Status: result.State.Status, RepositoryKey: result.State.RepositoryKey, RepositoryPath: result.State.RepositoryPath, WorkItem: result.State.WorkItem, Steps: make([]cli.WorkflowStepView, 0, len(result.State.Steps))}
		if result.State.Current < len(result.State.Steps) {
			view.CurrentGate = result.State.Steps[result.State.Current].Gate
		}
		for _, step := range result.State.Steps {
			digest := ""
			if step.Digest != ([32]byte{}) {
				digest = hex.EncodeToString(step.Digest[:])
			}
			view.Steps = append(view.Steps, cli.WorkflowStepView{Gate: step.Gate, Status: step.Status, Reference: step.Reference, Digest: digest})
		}
		response.Workflow = view
	}
	if result.WorkItem != nil {
		response.WorkItem = &cli.WorkItemView{ProjectID: result.WorkItem.ProjectID, RepositoryKey: result.WorkItem.RepositoryKey, Repository: result.WorkItem.ProviderRepository, Number: result.WorkItem.Number, URL: result.WorkItem.URL, State: result.WorkItem.State}
	}
	return response
}

func workItemResult(result workitem.Result) cli.Result {
	status := cli.Failed
	if result.Status == workitem.Succeeded {
		status = cli.Succeeded
	}
	response := cli.Result{Status: status, Category: result.Category}
	if result.Link.Number > 0 {
		response.WorkItem = &cli.WorkItemView{ProjectID: result.Link.ProjectID, RepositoryKey: result.Link.RepositoryKey, Repository: result.Link.ProviderRepository, Number: result.Link.Number, URL: result.Link.URL, State: result.Link.State}
	}
	return response
}

func cliResult(result projectapp.LifecycleResult) cli.Result {
	status := cli.Failed
	if result.Status == projectapp.LifecycleApplied || result.Status == projectapp.LifecycleUnchanged {
		status = cli.Succeeded
	}
	if result.Status == projectapp.LifecycleCancelled {
		status = cli.Cancelled
	}
	return cli.Result{Status: status, Category: result.Category}
}

func canonicalCompletion(facts completion.Facts, message string, references []string, next string, source provenance.Value) cli.Result {
	result, ok := newCompletion(facts, message, references, next, source)
	if !ok {
		return cli.Result{Status: cli.Failed, Category: "application_unavailable"}
	}
	return result
}

func newCompletion(facts completion.Facts, message string, references []string, next string, source provenance.Value) (cli.Result, bool) {
	statement, err := provenance.NewText(message, provenance.AxiomAuthored)
	if err != nil {
		return cli.Result{}, false
	}
	var nextAction provenance.Text
	if next != "" {
		nextAction, err = provenance.NewText(next, provenance.AxiomAuthored)
		if err != nil {
			return cli.Result{}, false
		}
	}
	canonical, err := completion.New(facts, statement, references, nextAction, "", source)
	if err != nil {
		return cli.Result{}, false
	}
	return cli.Result{Completion: &canonical}, true
}
