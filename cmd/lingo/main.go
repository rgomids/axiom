package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workitem"
)

var (
	buildVersion = "devel"
	buildCommit  = "unknown"
	buildSource  = "https://github.com/rgomids/axiom"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		os.Exit(writeVersion(os.Stdout))
	}
	os.Exit(cli.RunInteractive(context.Background(), os.Args[1:], compose(), os.Stdin, os.Stdout, os.Stderr))
}

type versionInfo struct {
	Product string `json:"product"`
	Binary  string `json:"binary"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Source  string `json:"source"`
}

func writeVersion(output *os.File) int {
	info := versionInfo{Product: "Axiom", Binary: "lingo", Version: buildVersion, Commit: buildCommit, Source: buildSource}
	if err := json.NewEncoder(output).Encode(info); err != nil {
		return cli.ExitFailure
	}
	return cli.ExitSuccess
}

func compose() cli.Service {
	root, err := projectsRoot()
	if err != nil {
		return cli.UnavailableService{}
	}
	state, err := stateRoot()
	if err != nil {
		return cli.UnavailableService{}
	}
	if overlap, err := local.RootsOverlap(root, state); err != nil || overlap {
		return cli.UnavailableService{}
	}
	store, err := local.NewPortableStore(root)
	if err != nil {
		return cli.UnavailableService{}
	}
	installation, err := local.NewInstallationStore(state)
	if err != nil {
		return cli.UnavailableService{}
	}
	codex, err := codexruntime.New(codexSkillsRoot())
	if err != nil {
		return cli.UnavailableService{}
	}
	workItems, err := local.NewWorkItemStore(state)
	if err != nil {
		return cli.UnavailableService{}
	}
	github, _ := workitem.NewGitHubAdapter(os.Getenv("AXIOM_GIT_BIN"), os.Getenv("AXIOM_GH_BIN"))
	workItemService := workitem.New(workItemResolver{installation}, github, github, workItems)
	return lifecycleService{projectapp.NewLifecycle(store, manifest.Codec{}, local.IdentityAllocator{}), installation, codex, workItemService, root}
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
	projectsRoot string
}

type workItemResolver struct{ installation local.InstallationStore }

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
	return cliResult(s.lifecycle.Validate(ctx, projectapp.ProjectRequest{Slug: input.Slug}))
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
	return cli.Result{Status: status, Category: result.Category}
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

func workItemResult(result workitem.Result) cli.Result {
	status := cli.Failed
	if result.Status == workitem.Succeeded {
		status = cli.Succeeded
	}
	return cli.Result{Status: status, Category: result.Category}
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
