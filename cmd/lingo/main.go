package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/githubissues"
	"github.com/rgomids/axiom/internal/install"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/runtimeapplication"
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
	if handled, code := installReleaseCommand(os.Args[1:], os.Stdout, os.Stderr); handled {
		os.Exit(code)
	}
	source := currentProvenance()
	if handled, code := cli.InspectSkill(os.Args[1:], source, os.Stdout); handled {
		os.Exit(code)
	}
	if format, ok := versionFormat(os.Args[1:]); ok {
		os.Exit(writeVersion(os.Stdout, format, source))
	}
	if len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		os.Exit(cli.Help(os.Stdout))
	}
	os.Exit(cli.RunInteractive(context.Background(), os.Args[1:], composeWithProvenance(source), source, os.Stdin, os.Stdout, os.Stderr))
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

// selfBuild is this binary's release identity exactly as injected at build
// time. The installer compares it with the verified candidate's metadata; it
// never grants ownership of anything.
func selfBuild() install.Build {
	release, err := strconv.ParseBool(buildRelease)
	return install.Build{Release: err == nil && release, Version: buildVersion, Revision: buildRevision, SourceState: buildSourceState}
}

// archiveRoot is the Axiom-owned machine-local preservation namespace for the
// RecognizedPOC transition: AXIOM_ARCHIVE_ROOT when set to an absolute path,
// otherwise "archive" beside the installation receipt directory, which is
// Axiom-owned state outside every Project, State and Skills root. A relative
// override disables the transition (empty) instead of guessing.
func archiveRoot(receiptDir string) string {
	if override, set := os.LookupEnv("AXIOM_ARCHIVE_ROOT"); set {
		if !filepath.IsAbs(override) {
			return ""
		}
		return filepath.Clean(override)
	}
	if !filepath.IsAbs(receiptDir) {
		return ""
	}
	return filepath.Join(filepath.Dir(filepath.Clean(receiptDir)), "archive")
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
	github, githubErr := githubissues.New(os.Getenv("AXIOM_GH_BIN"))
	var capability workitem.Capability
	var legacy workitem.LegacyProjection
	if githubErr == nil {
		capability = github
		legacy = github
	}
	workItemService := workitem.New(workItemResolver{installation: installation, portable: store}, capability, legacy, workItems, source)
	workflows, err := local.NewWorkflowStore(state)
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	artifacts, err := local.NewArtifactStore(state)
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	references := local.NewWorkflowReferenceValidator(artifacts)
	operational, err := local.NewOperationalStore(state)
	if err != nil {
		return cli.NewUnavailableService(source)
	}
	workflowService := workflow.New(workflowResolver{installation}, workflowWorkItems{workItemService}, workflows, github, references, source, nil, nil)
	return lifecycleService{lifecycle: projectapp.NewLifecycle(store, manifest.Codec{}, local.IdentityAllocator{}), portable: store, installation: installation, operational: operational, projectCatalog: projectapp.NewProjectCatalog(installation, installation), codex: codex, workItems: workItemService, workflows: workflowService, projectsRoot: root, stateRoot: state, skillsRoot: codexSkillsRoot(), runtimes: discoverRuntimeRoots(), provenance: source}
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
	if runtime.GOOS == "windows" {
		return local.WindowsStateRoot(home, os.Getenv("LOCALAPPDATA"))
	}
	return local.NativeStateRoot(runtime.GOOS, home, os.Getenv("XDG_STATE_HOME"))
}

type lifecycleService struct {
	// RuntimePolicySource allows composition tests to inject fresh snapshots.
	RuntimePolicySource    runtimeapplication.Source
	lifecycle              projectapp.Lifecycle
	portable               local.PortableStore
	installation           local.InstallationStore
	operational            local.OperationalStore
	projectCatalog         projectapp.ProjectCatalog
	codex                  codexruntime.Service
	workItems              workitem.Service
	workflows              workflow.Service
	projectsRoot           string
	stateRoot              string
	skillsRoot             string
	runtimes               runtimeRoots
	provenance             provenance.Value
	beforeLocalPublication func()
}

type workItemResolver struct {
	installation local.InstallationStore
	portable     local.PortableStore
}

type workflowResolver struct{ installation local.InstallationStore }
type workflowWorkItems struct{ service workitem.Service }

func (r workItemResolver) Resolve(ctx context.Context, selector string) (workitem.Project, string) {
	resolved := r.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return workitem.Project{}, resolved.Category
	}
	// Source confinement (#147): read only the portable source the protected
	// record names, never <projects-root>/<slug>, and accept it only when it is
	// exactly the configuration that installation validated.
	portable, err := r.portable.InspectRecordedSource(ctx, resolved.Project.Source, resolved.Project.Slug)
	switch {
	case ctx.Err() != nil:
		return workitem.Project{}, "cancelled"
	case errors.Is(err, local.ErrRecoveryRequired):
		return workitem.Project{}, "recovery_required"
	case err == nil && !portable.Exists:
		return workitem.Project{}, "project_source_unavailable"
	case err != nil:
		return workitem.Project{}, "invalid_project_capability_state"
	}
	state := portable.Snapshot.Project().State()
	if state.ID != resolved.Project.ID || state.Slug != resolved.Project.Slug || portable.Snapshot.Revision() != resolved.Project.PortableRevision {
		return workitem.Project{}, "invalid_project_capability_state"
	}
	// Same capability mapping algorithm as readiness; GitHub is the only
	// implemented Work Item Provider in this build.
	capability := projectapp.ResolveCapability(state, projectapp.WorkItemCapability, projectapp.SupportedProviders)
	if capability.Readiness != projectapp.CapabilityReady {
		return workitem.Project{}, "work_item_capability_unavailable"
	}
	project := workitem.Project{ID: resolved.Project.ID, Provider: capability.Provider, Repositories: make([]workitem.Repository, 0, len(resolved.Project.Repositories))}
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

func (w workflowWorkItems) Load(ctx context.Context, project, repository, provider, resource, selector string) (workflow.WorkItem, error) {
	if provider != "" && provider != "github" {
		return workflow.WorkItem{}, workitem.ErrNotFound
	}
	result := w.service.Show(ctx, workitem.Target{ProjectSelector: project, RepositoryKey: repository, ProviderResource: resource}, selector)
	if result.Status != workitem.Succeeded {
		return workflow.WorkItem{}, workitem.ErrNotFound
	}
	return workflow.WorkItem{Provider: result.Link.Provider, Resource: result.Link.Resource, ExternalID: result.Link.ExternalID, URL: result.Link.URL, State: result.Link.State}, nil
}

func (s lifecycleService) RuntimeCodexInstall(ctx context.Context) cli.Result {
	return runtimeResult(s.codex.Install(ctx))
}
func (s lifecycleService) RuntimeCodexStatus(ctx context.Context) cli.Result {
	return runtimeStatus(s.codex.Inspect(ctx), "Codex", "codex", s.provenance)
}

func runtimeStatus(result codexruntime.Result, label, id string, source provenance.Value) cli.Result {
	if result.Status == codexruntime.Ready {
		response := canonicalCompletion(completion.Facts{Completed: true}, "Lingo and "+label+" skills are compatible", []string{"skill-set:" + result.SkillSetVersion}, "Run project configure with explicit Project inputs", source)
		response.Runtime = runtimeView(result)
		return response
	}
	if result.Status == codexruntime.Incompatible || result.Status == codexruntime.Missing || result.Status == codexruntime.Partial {
		response := canonicalCompletion(completion.Facts{ValidationFailed: true}, label+" skill compatibility is not ready", nil, "Run runtime "+id+" install, then project configure", source)
		response.Runtime = runtimeView(result)
		return response
	}
	response := canonicalCompletion(completion.Facts{Failed: true}, label+" compatibility inspection failed", nil, "Inspect the configured "+label+" skill root", source)
	response.Runtime = runtimeView(result)
	return response
}

func runtimeResult(result codexruntime.Result) cli.Result {
	status := cli.Failed
	if result.Status == codexruntime.Applied || result.Status == codexruntime.Unchanged || result.Status == codexruntime.Ready {
		status = cli.Succeeded
	}
	return cli.Result{Status: status, Category: result.Category, Runtime: runtimeView(result)}
}

func runtimeView(result codexruntime.Result) *cli.RuntimeView {
	view := &cli.RuntimeView{SkillSetVersion: result.SkillSetVersion, BinaryCompatibility: result.BinaryCompatibility, Skills: make([]cli.RuntimeSkillView, 0, len(result.Skills)), Receipt: result.Receipt}
	for _, skill := range result.Skills {
		view.Skills = append(view.Skills, cli.RuntimeSkillView{Name: skill.Name, SHA256: skill.Digest, State: skill.State})
	}
	for _, conflict := range result.Conflicts {
		view.Conflicts = append(view.Conflicts, cli.RuntimeConflictView{Artifact: conflict.Artifact, State: conflict.State, SHA256: conflict.Digest})
	}
	return view
}

func (s lifecycleService) Init(ctx context.Context, input cli.InitInput) cli.Result {
	return cliResult(s.lifecycle.Init(ctx, projectapp.InitRequest{Slug: input.Slug, Name: input.Name}))
}
func (s lifecycleService) Validate(ctx context.Context, input cli.ProjectInput) cli.Result {
	result := s.lifecycle.Validate(ctx, projectapp.ProjectRequest{Slug: input.Slug})
	if result.Status == projectapp.LifecycleApplied || result.Status == projectapp.LifecycleUnchanged {
		// Structural validity is not readiness: report what each supported
		// operation can do on this machine (Issue #231). Read-only.
		response := canonicalCompletion(completion.Facts{Completed: true}, "Project is valid", nil, "", s.provenance)
		report := s.readiness().Evaluate(ctx, input.Slug)
		response.Readiness = &report
		return response
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

// Install records an operator-authored manifest. A manifest that declares
// Repositories installs only with an exact binding for each declared key.
func (s lifecycleService) Install(ctx context.Context, input cli.InstallInput) cli.Result {
	bindings := make([]projectapp.RepositoryBinding, 0, len(input.Repositories))
	seen := map[string]bool{}
	for _, repository := range input.Repositories {
		if !filepath.IsAbs(repository.Path) || seen[repository.Key] {
			return cli.Result{Status: cli.Failed, Category: "invalid_repository_bindings"}
		}
		seen[repository.Key] = true
		path := filepath.Clean(repository.Path)
		identity, err := local.DirectoryIdentity(path)
		if err != nil {
			return cli.Result{Status: cli.Failed, Category: "invalid_repository_bindings"}
		}
		bindings = append(bindings, projectapp.RepositoryBinding{RepositoryKey: repository.Key, ExplicitPath: path, CanonicalIdentity: identity, Observation: projectapp.Observation{Availability: projectapp.Unverified, Basis: projectapp.NotChecked}})
	}
	result := s.installation.InstallWithBindings(ctx, input.Source, bindings)
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
		return projectShowFailure(result.Category, s.provenance)
	}
	references := []string{"project:" + result.Project.ID}
	for _, repository := range result.Project.Repositories {
		references = append(references, "repository:"+repository.Key)
	}
	response := canonicalCompletion(completion.Facts{Completed: true}, "Project resolved", references, "", s.provenance)
	response.Project = projectView(result.Project)
	return response
}

func (s lifecycleService) List(ctx context.Context) cli.Result {
	result := s.projectCatalog.List(ctx)
	switch result.Status {
	case projectapp.ProjectListSucceeded:
		message := "Configured Projects listed"
		if len(result.Projects) == 0 {
			message = "No configured Projects"
		}
		response := canonicalCompletion(completion.Facts{Completed: true}, message, nil, "", s.provenance)
		response.Projects = make([]cli.ProjectListView, 0, len(result.Projects))
		for _, configured := range result.Projects {
			response.Projects = append(response.Projects, cli.ProjectListView{ID: configured.ID, Slug: configured.Slug, Name: configured.Name})
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

func projectShowFailure(category string, source provenance.Value) cli.Result {
	switch category {
	case "invalid_project_selector":
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project selector is invalid", nil, "Provide a valid Project UUID or slug", source)
	case "project_not_found":
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project was not found", nil, "Provide an existing Project UUID or slug", source)
	case "project_ambiguous":
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project selector is ambiguous", nil, "Provide the exact Project UUID", source)
	case "invalid_existing_local_state":
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Project state is invalid", nil, "Repair or reconfigure local Project state before retrying inspection", source)
	case "recovery_required":
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Project state requires recovery", nil, "Review preserved local recovery state before retrying inspection", source)
	case "project_source_unavailable":
		return canonicalCompletion(completion.Facts{RetrySafeFailure: true}, "Project source is unavailable", nil, "Restore the configured Project source and retry inspection", source)
	case "repository_unavailable":
		return canonicalCompletion(completion.Facts{RetrySafeFailure: true}, "Project repository is unavailable", nil, "Restore the configured repository binding and retry inspection", source)
	case "cancelled":
		return canonicalCompletion(completion.Facts{WasInterrupted: true}, "Project inspection was interrupted", nil, "Retry Project inspection", source)
	default:
		return canonicalCompletion(completion.Facts{Failed: true}, "Project inspection failed", nil, "Inspect local storage and application availability before retrying", source)
	}
}

func projectView(project local.ResolvedProject) *cli.ProjectView {
	view := &cli.ProjectView{ID: project.ID, Slug: project.Slug, Source: project.Source, Repositories: make([]cli.RepositoryView, 0, len(project.Repositories))}
	for _, repository := range project.Repositories {
		view.Repositories = append(view.Repositories, cli.RepositoryView{Key: repository.Key, Path: repository.Path})
	}
	return view
}
func (s lifecycleService) Configure(ctx context.Context, input cli.ConfigureInput) cli.Result {
	if input.Project != "" {
		return s.configureEdit(ctx, input)
	}
	if input.RemoveWorkItemProvider || len(input.RemoveRepositories) != 0 {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project setup input is invalid", nil, "Removal applies only to an existing Project selected with --project", s.provenance)
	}
	repositories := make([]projectapp.SetupRepository, 0, len(input.Repositories))
	for _, repository := range input.Repositories {
		path := cleanPath(repository.Path)
		identity, err := local.DirectoryIdentity(path)
		if err != nil {
			return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project setup input is invalid", nil, "Provide an existing absolute non-link Repository path", s.provenance)
		}
		// Discovery reads only this explicit location: no CWD, no parent walk,
		// no Git process, no network.
		repositories = append(repositories, projectapp.SetupRepository{Key: repository.Key, Path: path, Revision: identity, Discovery: discoverRepository(path)})
	}
	portableObservation, err := s.portable.Inspect(ctx, input.Slug)
	if err != nil && !errors.Is(err, projectapp.ErrNotFound) {
		return canonicalCompletion(completion.Facts{Failed: true}, "Project setup inspection failed", nil, "Inspect portable Project state before retrying", s.provenance)
	}
	// CREATE never reuses, adopts, or edits an existing Project: a configured
	// slug is a terminal collision even when every supplied value is equivalent.
	if portableObservation.Exists {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project slug is already configured", nil, "Choose a different slug, or select the existing Project with --project to preview an edit", s.provenance)
	}
	projectID := input.ProjectID
	if projectID == "" {
		allocated, issues := (local.IdentityAllocator{}).NewID()
		if len(issues) != 0 {
			return canonicalCompletion(completion.Facts{Failed: true}, "Project identity allocation failed", nil, "Retry Project setup", s.provenance)
		}
		projectID = allocated
	}
	localObservation, category := s.installation.Inspect(ctx, projectID)
	if category != "" {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Project state is not safe to configure", nil, "Inspect preserved local state before retrying", s.provenance)
	}
	// A supplied --project-id that is already installed names another Project;
	// CREATE never reuses or adopts an existing identity.
	if localObservation.Exists {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project identity is already configured", nil, "Omit --project-id to allocate a new identity, or select the existing Project with --project to preview an edit", s.provenance)
	}
	observation := projectapp.SetupObservation{
		PortableDestination: filepath.Join(s.projectsRoot, input.Slug),
		LocalDestination:    filepath.Join(s.stateRoot, "projects", projectID),
		PortableRevision:    portableObservation.Revision,
		LocalRevision:       localObservation.Revision,
	}
	setupInput := projectapp.SetupInput{ProjectID: projectID, Slug: input.Slug, Name: input.Name, Repositories: repositories, WorkItemProvider: input.WorkItemProvider,
		RepositoryRemotes: input.RepositoryRemotes, RuntimeCandidates: runtimeCandidates(ctx, s.stateRoot), Runtimes: input.Runtimes, ModelProfiles: input.ModelProfiles,
		RemoveTechnology: input.RemoveTechnology, Documentation: documentationInputs(input), BusinessContext: input.BusinessContext, ContextSources: input.ContextSources}
	for _, preference := range input.RuntimePreferences {
		setupInput.RuntimePreferences = append(setupInput.RuntimePreferences, projectapp.SetupPreference{Role: preference.Role, Complexity: preference.Complexity, ModelProfile: preference.ModelProfile})
	}
	for _, fact := range input.Technology {
		setupInput.Technology = append(setupInput.Technology, project.TechnologyFact{Key: fact.Key, Value: fact.Value})
	}
	for _, entry := range input.Glossary {
		setupInput.Glossary = append(setupInput.Glossary, project.GlossaryEntry{Key: entry.Key, Term: entry.Term, Definition: entry.Definition})
	}
	proposal, issues := projectapp.PrepareSetup(manifest.Codec{}, setupInput, observation)
	if len(issues) != 0 {
		message, next := setupIssueText(issues)
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, message, nil, next, s.provenance)
	}
	desiredSnapshot, snapshotIssues := projectapp.ReadSnapshot(manifest.Codec{}, proposal.Manifest(), nil)
	if len(snapshotIssues) != 0 {
		return canonicalCompletion(completion.Facts{Failed: true}, "Project setup proposal could not be encoded", nil, "Review application availability before retrying", s.provenance)
	}
	desiredRecord, recordIssues := local.NewRecord(local.RecordState{ProjectID: projectID, ObservedSlug: input.Slug, SourceLocation: filepath.Join(s.projectsRoot, input.Slug), PortableRevision: desiredSnapshot.Revision(), ArtifactDigests: desiredSnapshot.Digests(), Repositories: proposal.Bindings(), Documentation: proposal.DocumentationBindings()})
	if len(recordIssues) != 0 {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Local Project proposal is invalid", nil, "Correct Repository bindings and retry", s.provenance)
	}
	if _, encodeIssues := local.EncodeRecord(desiredRecord); len(encodeIssues) != 0 {
		return canonicalCompletion(completion.Facts{Failed: true}, "Local Project proposal could not be encoded", nil, "Review application availability before retrying", s.provenance)
	}
	preview := proposal.Preview()
	if !input.AuthorizeLocal {
		next := "Review preview, then repeat with --project-id, --preview-digest, and --authorize-local"
		if len(preview.Blockers) != 0 {
			next = "Resolve the reported bootstrap blockers (for example --repository-remote <key>=<locator|none>) and preview again"
		} else if len(preview.Effects) == 0 {
			next = "No publication is required"
		}
		result := canonicalCompletion(completion.Facts{Completed: true}, "Project setup preview ready", []string{"project:" + projectID}, next, s.provenance)
		result.Setup = &preview
		return result
	}
	// Bootstrap blockers are unresolved intent: no authority can publish them.
	if !proposal.Publishable() {
		result := canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project bootstrap has unresolved blockers", nil, "Resolve the reported bootstrap blockers and preview again", s.provenance)
		result.Setup = &preview
		return result
	}
	if !proposal.MatchesDigest(input.PreviewDigest) {
		result := canonicalCompletion(completion.Facts{AuthorityDenied: true}, "Project setup authority is missing or stale", nil, "Review the current preview and authorize its exact digest", s.provenance)
		result.Setup = &preview
		return result
	}
	portableResult := s.lifecycle.PublishConfigured(ctx, proposal.Project())
	if portableResult.Status != projectapp.LifecycleApplied && portableResult.Status != projectapp.LifecycleUnchanged {
		facts := completion.Facts{Failed: true}
		if portableResult.Status == projectapp.LifecycleConflict {
			facts = completion.Facts{AuthorityDenied: true}
		}
		result := canonicalCompletion(facts, "Portable Project publication failed", nil, "Review current state and prepare a fresh preview", s.provenance)
		result.Setup = &preview
		return result
	}
	if s.beforeLocalPublication != nil {
		s.beforeLocalPublication()
	}
	localResult := s.installation.InstallWithContext(ctx, filepath.Join(s.projectsRoot, input.Slug), proposal.Bindings(), proposal.DocumentationBindings())
	if localResult.Status != local.InstallationApplied && localResult.Status != local.InstallationUnchanged {
		facts := completion.Facts{Failed: true}
		references := []string(nil)
		message := "Local Project publication failed"
		if portableResult.Status == projectapp.LifecycleApplied {
			facts = completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}
			references = []string{"project:" + projectID, "portable:" + observation.PortableDestination}
			message = "Portable Project published; local bindings did not complete"
		}
		result := canonicalCompletion(facts, message, references, "Inspect local state and resume with a fresh preview", s.provenance)
		result.Setup = &preview
		return result
	}
	message := "Project setup published"
	if portableResult.Status == projectapp.LifecycleUnchanged && localResult.Status == local.InstallationUnchanged {
		message = "Project setup already matches the authorized proposal"
	}
	next := "Project resolves by UUID or slug; Work Item capability is ready"
	if preview.Capability.Readiness != projectapp.CapabilityReady {
		next = "Project is valid; configure a supported Work Item capability before starting that journey"
	}
	result := canonicalCompletion(completion.Facts{Completed: true}, message, []string{"project:" + projectID}, next, s.provenance)
	result.Setup = &preview
	return result
}

// configureEdit exposes only the zero-write EDIT preview. No EDIT publication
// or replay path exists, so replay/authority inputs fail before any read.
func (s lifecycleService) configureEdit(ctx context.Context, input cli.ConfigureInput) cli.Result {
	if input.Slug != "" {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project edit input is invalid", nil, "Remove --slug; Project rename is not supported", s.provenance)
	}
	if input.ProjectID != "" || input.PreviewDigest != "" || input.AuthorizeLocal {
		return canonicalCompletion(completion.Facts{ValidationFailed: true}, "Project edit publication is not available", nil, "Remove --project-id, --preview-digest, and --authorize-local; edit only previews", s.provenance)
	}
	intent := projectapp.EditIntent{
		Selector:               input.Project,
		Name:                   projectapp.OptionalText{Supplied: input.NameSupplied, Value: input.Name},
		WorkItemProvider:       projectapp.OptionalText{Supplied: input.WorkItemProviderSupplied, Value: input.WorkItemProvider},
		RemoveWorkItemProvider: input.RemoveWorkItemProvider,
		RepositoryRemovals:     append([]string(nil), input.RemoveRepositories...),
	}
	for _, repository := range input.Repositories {
		intent.RepositoryUpserts = append(intent.RepositoryUpserts, projectapp.RepositoryUpsert{Key: repository.Key, Path: repository.Path})
	}
	ports := projectapp.EditPorts{
		Source:    editSource{installation: s.installation, portable: s.portable, stateRoot: s.stateRoot},
		Checkouts: local.DirectoryObserver{}, Manifest: manifest.Codec{}, Local: local.RecordCodec{},
	}
	proposal, failure := projectapp.PreviewEdit(ctx, ports, intent)
	if failure != projectapp.EditOK {
		return editFailure(failure, s.provenance)
	}
	preview := proposal.Preview()
	next := "Review the complete preview; edit publication is not available in this build"
	if len(preview.Effects) == 0 {
		next = "No change is required"
	}
	result := canonicalCompletion(completion.Facts{Completed: true}, "Project edit preview ready", []string{"project:" + preview.ProjectID}, next, s.provenance)
	result.Edit = &preview
	return result
}

func editFailure(failure projectapp.EditFailure, source provenance.Value) cli.Result {
	facts := completion.Facts{ValidationFailed: true}
	message, next := "Project edit input is invalid", "Correct conflicting or invalid edit operations and retry"
	switch failure {
	case projectapp.EditProjectNotFound:
		message, next = "Project was not found", "Provide the UUID or slug of an existing Project; edit never creates a Project"
	case projectapp.EditProjectAmbiguous:
		message, next = "Project selector is ambiguous", "Select the Project by its UUID"
	case projectapp.EditUnknownRepository:
		message, next = "Repository to remove is not configured", "Remove only configured Repository keys"
	case projectapp.EditRepositoryUnavailable:
		message, next = "Repository path is unavailable", "Provide an existing absolute non-link Repository path"
	case projectapp.EditCandidateInvalid:
		message, next = "Resulting Project configuration is invalid", "Adjust the edit so the complete Project remains valid"
	case projectapp.EditStateUnsafe:
		facts, message, next = completion.Facts{Failed: true}, "Selected Project state is not safe to edit", "Inspect preserved portable and local state before retrying"
	case projectapp.EditRecoveryRequired:
		facts, message, next = completion.Facts{Failed: true}, "Selected Project state requires recovery", "Inspect and recover preserved state before editing"
	case projectapp.EditCancelled:
		facts, message, next = completion.Facts{WasInterrupted: true}, "Project edit preview was cancelled", "Retry the edit preview"
	case projectapp.EditUnavailable:
		facts, message, next = completion.Facts{Failed: true}, "Project edit is unavailable", "Review application availability before retrying"
	}
	return canonicalCompletion(facts, message, nil, next, source)
}

// editSource selects exactly one installed Project and loads its protected
// local record plus the recorded portable source it names. It does not check
// binding availability so a broken binding can be repaired or removed.
type editSource struct {
	installation local.InstallationStore
	portable     local.PortableStore
	stateRoot    string
}

func (e editSource) SelectForEdit(ctx context.Context, selector string) (projectapp.EditSelection, projectapp.EditFailure) {
	selected := e.installation.Select(ctx, selector)
	if selected.Status != local.ResolutionFound {
		return projectapp.EditSelection{}, editSelectionFailure(selected.Category)
	}
	observation, category := e.installation.Inspect(ctx, selected.Project.ID)
	if category != "" {
		return projectapp.EditSelection{}, editSelectionFailure(category)
	}
	if !observation.Exists {
		return projectapp.EditSelection{}, projectapp.EditStateUnsafe
	}
	state := observation.Record.State()
	// Source confinement: read only the exact portable source the protected
	// record names, through the same safe loading rules used to install it.
	portable, err := e.portable.InspectRecordedSource(ctx, state.SourceLocation, state.ObservedSlug)
	switch {
	case ctx.Err() != nil:
		return projectapp.EditSelection{}, projectapp.EditCancelled
	case errors.Is(err, projectapp.ErrRecoveryRequired):
		return projectapp.EditSelection{}, projectapp.EditRecoveryRequired
	case err != nil || !portable.Exists:
		return projectapp.EditSelection{}, projectapp.EditStateUnsafe
	}
	return projectapp.EditSelection{
		Portable: portable.Snapshot, Local: local.ApplicationRecord(observation.Record), LocalWire: observation.Wire,
		PortableDestination: state.SourceLocation, LocalDestination: filepath.Join(e.stateRoot, "projects", state.ProjectID),
		PortableRevision: portable.Revision, LocalRevision: observation.Revision,
	}, projectapp.EditOK
}

func editSelectionFailure(category string) projectapp.EditFailure {
	switch category {
	case "project_not_found":
		return projectapp.EditProjectNotFound
	case "project_ambiguous":
		return projectapp.EditProjectAmbiguous
	case "invalid_project_selector":
		return projectapp.EditInvalidIntent
	case "cancelled":
		return projectapp.EditCancelled
	case "recovery_required":
		return projectapp.EditRecoveryRequired
	}
	return projectapp.EditStateUnsafe
}

func (s lifecycleService) WorkItemCreate(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemCreate); blocked != nil {
		return *blocked
	}
	draft := workitem.DraftInput{
		Type: workitem.Type(input.Type), Beneficiary: workitem.SectionInput{Supplied: input.Beneficiary}, Value: workitem.SectionInput{Supplied: input.Value}, Classification: input.Classification,
		Target: workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, ProviderResource: input.ProviderRepository},
		Intent: input.Intent, Problem: workitem.SectionInput{Supplied: input.Problem, Elaborated: input.ElaboratedSections["problem"]}, DesiredOutcome: workitem.SectionInput{Supplied: input.DesiredOutcome, Elaborated: input.ElaboratedSections["desired_outcome"]},
		Context: workitem.SectionInput{Supplied: input.Context, Elaborated: input.ElaboratedSections["context"]}, Scope: workitem.SectionInput{Supplied: input.Scope, Elaborated: input.ElaboratedSections["scope"]}, Constraints: workitem.SectionInput{Supplied: input.Constraints, Elaborated: input.ElaboratedSections["constraints"]},
		NonGoals: workitem.SectionInput{Supplied: input.NonGoals, Elaborated: input.ElaboratedSections["non_goals"]}, Acceptance: workitem.SectionInput{Supplied: input.Acceptance, Elaborated: input.ElaboratedSections["acceptance_expectations"]}, Cancelled: input.Cancelled,
	}
	if !input.AuthorizeExternal && input.PreviewDigest == "" {
		return workItemResult(s.workItems.Prepare(ctx, draft), s.provenance)
	}
	return workItemResult(s.workItems.Create(ctx, draft, input.PreviewDigest, input.AuthorizeExternal), s.provenance)
}
func (s lifecycleService) WorkItemSelect(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemSelect); blocked != nil {
		return *blocked
	}
	if input.Provider != "" && input.Provider != "github" {
		return workItemResult(workitem.Result{Status: completion.ValidationFailure, Category: "invalid_work_item_input"}, s.provenance)
	}
	target := workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, ProviderResource: input.ProviderRepository}
	if !input.AuthorizeLocal && input.PreviewDigest == "" {
		return workItemResult(s.workItems.PreviewSelect(ctx, target, workItemExternalID(input)), s.provenance)
	}
	return workItemResult(s.workItems.Select(ctx, target, workItemExternalID(input), input.PreviewDigest, input.AuthorizeLocal), s.provenance)
}
func (s lifecycleService) WorkItemShow(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemShow); blocked != nil {
		return *blocked
	}
	if input.Provider != "" && input.Provider != "github" {
		return workItemResult(workitem.Result{Status: completion.ValidationFailure, Category: "invalid_work_item_input"}, s.provenance)
	}
	return workItemResult(s.workItems.Show(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, ProviderResource: input.ProviderRepository}, workItemExternalID(input)), s.provenance)
}
func (s lifecycleService) WorkItemComment(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemComment); blocked != nil {
		return *blocked
	}
	if input.Provider != "" && input.Provider != "github" {
		return workItemResult(workitem.Result{Status: completion.ValidationFailure, Category: "invalid_work_item_input"}, s.provenance)
	}
	return workItemResult(s.workItems.Comment(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, ProviderResource: input.ProviderRepository}, workItemExternalID(input), input.Message, input.AuthorizeExternal), s.provenance)
}
func (s lifecycleService) WorkItemComplete(ctx context.Context, input cli.WorkItemInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitWorkItemComplete); blocked != nil {
		return *blocked
	}
	if input.Provider != "" && input.Provider != "github" {
		return workItemResult(workitem.Result{Status: completion.ValidationFailure, Category: "invalid_work_item_input"}, s.provenance)
	}
	return workItemResult(s.workItems.Complete(ctx, workitem.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, ProviderResource: input.ProviderRepository}, workItemExternalID(input), input.AuthorizeExternal), s.provenance)
}

func workItemExternalID(input cli.WorkItemInput) string {
	if input.ExternalID != "" {
		return input.ExternalID
	}
	return strconv.Itoa(input.Number)
}

func workflowTarget(input cli.WorkflowInput) workflow.Target {
	externalID := input.ExternalID
	if externalID == "" {
		externalID = strconv.Itoa(input.Number)
	}
	return workflow.Target{ProjectSelector: input.Project, RepositoryKey: input.Repository, WorkItemProvider: input.Provider, WorkItemResource: input.ProviderRepository, WorkItem: externalID, ExecutionID: input.Execution, RuntimeID: input.Runtime}
}

func (s lifecycleService) WorkflowStart(ctx context.Context, input cli.WorkflowInput) cli.Result {
	// Start enforces the shared and Work Item requirements here; its Runtime
	// requirement is the #140 request-specific projection (reviewed preview
	// plus fresh Check) below, so Runtimes are not observed twice.
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionStart); blocked != nil {
		return *blocked
	}
	policyInput := cli.RuntimeProfilePreviewInput{Project: input.Project, Role: input.Role, Complexity: input.Complexity, Capabilities: input.Capabilities, Runtime: input.Runtime}
	preview, policy, err := s.runtimePolicyPreview(ctx, policyInput)
	if err != nil {
		return s.runtimeResolutionResult(preview, false)
	}
	if input.RuntimePreview == "" {
		return s.runtimeResolutionResult(preview, true)
	}
	if input.RuntimePreview != preview.Digest() {
		return s.runtimePolicyFailure("stale_preview")
	}
	binding, err := policy.Check(ctx, preview)
	if err != nil {
		return s.runtimePolicyFailure("stale_preview")
	}
	if input.Runtime != "" && input.Runtime != binding.Choice.RuntimeID {
		return s.runtimePolicyFailure("runtime_mismatch")
	}
	input.Runtime = binding.Choice.RuntimeID
	return workflowResult(s.workflows.Start(ctx, workflowTarget(input)), s.provenance)
}
func (s lifecycleService) WorkflowAdvance(ctx context.Context, input cli.WorkflowInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionAdvance); blocked != nil {
		return *blocked
	}
	references, ok := workflowReferences(input.Reference)
	if !ok {
		return workflowResult(workflow.Result{Status: workflow.ValidationFailed, Category: "invalid_workflow_reference"}, s.provenance)
	}
	return workflowResult(s.workflows.Transition(ctx, workflowTarget(input), workflow.TransitionInput{ExpectedRevision: input.ExpectedRevision, Stage: workflow.Stage(input.Gate), Outcome: workflow.Outcome(input.Outcome), References: references, Next: input.Next}), s.provenance)
}
func (s lifecycleService) WorkflowFact(ctx context.Context, input cli.WorkflowInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionFact); blocked != nil {
		return *blocked
	}
	references, ok := workflowReferences(input.Reference)
	if !ok || len(references) != 1 {
		return workflowResult(workflow.Result{Status: workflow.ValidationFailed, Category: "invalid_workflow_reference"}, s.provenance)
	}
	kinds := map[string]workflow.LifecycleFactKind{
		"planning-authority":       workflow.FactPlanningAuthority,
		"implementation-authority": workflow.FactImplementationAuthority,
		"review-started":           workflow.FactReviewStarted,
		"human-acceptance":         workflow.FactHumanAcceptance,
		"blocked":                  workflow.FactBlocked,
		"needs-decision":           workflow.FactNeedsDecision,
		"needs-approval":           workflow.FactNeedsApproval,
	}
	kind := kinds[input.Fact]
	if kind == "" {
		return workflowResult(workflow.Result{Status: workflow.ValidationFailed, Category: "invalid_lifecycle_fact"}, s.provenance)
	}
	return workflowResult(s.workflows.RecordLifecycleFact(ctx, workflowTarget(input), workflow.LifecycleFactInput{ExpectedRevision: input.ExpectedRevision, Kind: kind, Active: input.Active, Reference: references[0]}, input.AuthorizeLocal), s.provenance)
}
func (s lifecycleService) WorkflowResume(ctx context.Context, input cli.WorkflowInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionResume); blocked != nil {
		return *blocked
	}
	return workflowResult(s.workflows.Resume(ctx, workflowTarget(input), input.ExpectedRevision), s.provenance)
}
func (s lifecycleService) WorkflowStatus(ctx context.Context, input cli.WorkflowInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionStatus); blocked != nil {
		return *blocked
	}
	return workflowResult(s.workflows.Status(ctx, workflowTarget(input)), s.provenance)
}
func (s lifecycleService) WorkflowEvidence(ctx context.Context, input cli.WorkflowInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionEvidence); blocked != nil {
		return *blocked
	}
	result := s.workflows.Status(ctx, workflowTarget(input))
	if result.Status == workflow.Succeeded {
		result.Category = "workflow_evidence_ready"
	}
	return workflowResult(result, s.provenance)
}

func (s lifecycleService) WorkflowReconcile(ctx context.Context, input cli.WorkflowInput) cli.Result {
	if blocked := s.gate(ctx, input.Project, projectapp.AdmitExecutionReconcile); blocked != nil {
		return *blocked
	}
	if !input.AuthorizeExternal {
		return workflowResult(s.workflows.PrepareProjection(ctx, workflowTarget(input), input.ExpectedRevision), s.provenance)
	}
	return workflowResult(s.workflows.Project(ctx, workflowTarget(input), input.ExpectedRevision, input.PreviewDigest, true), s.provenance)
}

func workflowReferences(value string) ([]workflow.Reference, bool) {
	if value == "" {
		return nil, true
	}
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return nil, false
	}
	return []workflow.Reference{{Kind: parts[0], ID: parts[1], Digest: parts[2]}}, true
}

func workflowResult(result workflow.Result, source provenance.Value) cli.Result {
	facts := factsForCompletionStatus(result.Status)
	response := canonicalCompletion(facts, workflowResultText(result), workflowResultReferences(result), workflowResultNext(result), source)
	response.Category = result.Category
	response.Projection = result.Preview
	if result.State.ProjectID != "" {
		view := &cli.WorkflowView{ExecutionID: result.State.ExecutionID, WorkflowVersion: result.State.WorkflowVersion, Status: string(result.State.Status), CurrentGate: string(result.State.Stage), Revision: result.State.Revision, RepositoryKey: result.State.RepositoryKey, RuntimeID: result.State.RuntimeID, WorkItem: cli.WorkItemView{ProjectID: result.State.ProjectID, RepositoryKey: result.State.RepositoryKey, Provider: result.State.WorkItem.Provider, Resource: result.State.WorkItem.Resource, ExternalID: result.State.WorkItem.ExternalID, URL: result.State.WorkItem.URL, State: result.State.WorkItem.State}, Transitions: make([]cli.WorkflowStepView, 0, len(result.State.Transitions))}
		if lifecycle, err := workflow.DeriveLifecycle(result.State); err == nil {
			view.LifecycleStage = string(lifecycle.Stage)
			view.Blocked = lifecycle.Conditions.Blocked
			view.NeedsDecision = lifecycle.Conditions.NeedsDecision
			view.NeedsApproval = lifecycle.Conditions.NeedsApproval
		}
		for _, step := range result.State.Transitions {
			view.Transitions = append(view.Transitions, cli.WorkflowStepView{Revision: step.Revision, From: string(step.From), To: string(step.To), Outcome: string(step.Outcome), CommittedAt: step.CommittedAt.Format("2006-01-02T15:04:05.999999999Z07:00")})
		}
		response.Workflow = view
	}
	return response
}

func workflowResultText(result workflow.Result) string {
	if result.Category == "workflow_cancelled" {
		return "Execution workflow operation was cancelled"
	}
	if result.Status == workflow.Succeeded {
		return "Execution workflow operation completed"
	}
	if result.Status == workflow.Partial {
		return "Provider effect confirmed but projection bookkeeping is incomplete"
	}
	if result.Status == workflow.Retryable {
		return "Provider projection requires bounded reconciliation"
	}
	if result.Status == workflow.Denied {
		return "Execution authority is stale or incomplete"
	}
	if result.Status == workflow.Interrupted {
		return "Execution remains at the current workflow stage"
	}
	return "Execution workflow operation did not complete"
}
func workflowResultReferences(result workflow.Result) []string {
	refs := []string{}
	if result.State.ExecutionID != "" {
		refs = append(refs, "execution:"+result.State.ExecutionID)
	}
	if result.State.WorkItem.URL != "" && result.Status == workflow.Partial {
		refs = append(refs, "provider:"+result.State.WorkItem.URL)
	}
	return refs
}
func workflowResultNext(result workflow.Result) string {
	if result.Category == "workflow_cancelled" {
		return "Read current Execution status before retrying"
	}
	if result.Status == workflow.Partial || result.Status == workflow.Retryable {
		return "Re-read Provider state and reconcile the same projection key before retrying mutation"
	}
	if result.Status == workflow.Denied {
		return "Read current Execution status and prepare fresh exact authority"
	}
	if result.Status == workflow.Interrupted {
		return "Resume from the exact committed Execution revision"
	}
	return ""
}

func workItemResult(result workitem.Result, source provenance.Value) cli.Result {
	message, next := workItemResultText(result)
	facts := factsForCompletionStatus(result.Status)
	references := []string{}
	if result.Link.ExternalID != "" {
		references = append(references, "provider:"+result.Link.Provider+":"+result.Link.Resource+"#"+result.Link.ExternalID)
	}
	response := canonicalCompletion(facts, message, references, next, source)
	response.Category = result.Category
	response.Draft = result.Draft
	response.Selection = result.Selection
	response.Questions = result.Questions
	if result.Link.ExternalID != "" {
		response.WorkItem = &cli.WorkItemView{ProjectID: result.Link.ProjectID, RepositoryKey: result.Link.RepositoryKey, Provider: result.Link.Provider, Resource: result.Link.Resource, ExternalID: result.Link.ExternalID, URL: result.Link.URL, State: result.Link.State}
	}
	return response
}

func factsForCompletionStatus(status completion.Status) completion.Facts {
	switch status {
	case completion.Success:
		return completion.Facts{Completed: true}
	case completion.ValidationFailure:
		return completion.Facts{ValidationFailed: true}
	case completion.DeniedAuthority:
		return completion.Facts{AuthorityDenied: true}
	case completion.Partial:
		return completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}
	case completion.Interrupted:
		return completion.Facts{WasInterrupted: true}
	case completion.RetryableFailure:
		return completion.Facts{RetrySafeFailure: true}
	default:
		return completion.Facts{Failed: true}
	}
}

func workItemResultText(result workitem.Result) (string, string) {
	switch result.Category {
	case "work_item_draft_ready":
		return "Work Item draft ready for review", "Repeat create with this preview digest and explicit external authority"
	case "draft_incomplete":
		return "Work Item intent is incomplete", "Provide answers only for the listed missing fields"
	case "invalid_work_item_type":
		return "Work Item type is invalid", "Choose story, bug, or task and review a new draft"
	case "story_value_requires_story":
		return "Story value requires the story type", "Choose story or omit the story-specific fields"
	case "provider_classification_unsupported":
		return "Provider classification is unsupported", "Select only existing provider classifications and review a new draft"
	case "provider_classification_failed":
		return "Provider classification could not be resolved", "Inspect provider availability before reviewing a new draft"
	case "provider_metadata_incomplete":
		return "Work Item created but provider classification was not applied", "Inspect the existing Issue and provider permissions; do not create another Issue"
	case "provider_metadata_unverified":
		return "Work Item linked but provider classification remains unverified", "Review the original draft and existing Issue; do not create another Issue"
	case "draft_cancelled", "create_cancelled":
		return "Work Item operation cancelled", "Resume with the same explicit inputs when ready"
	case "work_item_selection_ready":
		return "GitHub Work Item selection ready for review", "Repeat select with this preview digest and explicit local authority"
	case "external_authority_denied", "local_authority_denied", "external_mutation_denied":
		return "Work Item authority denied", "Review the exact preview and grant only the required authority"
	case "work_item_linked", "work_item_already_linked":
		return "GitHub Work Item linked", "Inspect the local Work Item link before starting later workflow work"
	case "work_item_loaded":
		return "Work Item link loaded", "Use only separately authorized later operations"
	case "work_item_commented":
		return "Historical Work Item comment completed", "Treat this as POC behavior until the later Slice replaces it"
	case "work_item_completed":
		return "Historical Work Item completion completed", "Treat this as POC behavior until the later Slice replaces it"
	case "provider_create_ambiguous":
		return "GitHub create result is ambiguous", "Repeat the same reviewed draft to reconcile only; Axiom will not create again while the durable attempt is pending"
	case "provider_rate_limited", "provider_unavailable":
		return "GitHub capability is temporarily unavailable", "Retry the same operation after provider recovery"
	case "local_create_attempt_recovery_required", "local_create_attempt_conflict", "local_create_attempt_failed":
		return "Create-attempt state is not safely writable", "Inspect protected local state; no GitHub create was issued by this execution"
	case "work_item_ambiguous":
		return "Work Item identity is ambiguous", "Specify and reconcile the exact provider resource before continuing"
	case "provider_confirmed_local_failed", "provider_confirmed_local_conflict", "provider_confirmed_local_recovery_required", "provider_confirmed_local_read_failed", "provider_confirmed_create_attempt_recovery_required", "local_link_committed_recovery_required":
		return "GitHub effect confirmed but local linkage is incomplete", "Preserve the Issue reference and reconcile local linkage before retrying create"
	default:
		return "Work Item operation did not complete", "Review bounded validation details and retry safely"
	}
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
