package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], compose(), os.Stdout))
}

func compose() cli.Service {
	root, err := projectsRoot()
	if err != nil {
		return cli.UnavailableService{}
	}
	store, err := local.NewPortableStore(root)
	if err != nil {
		return cli.UnavailableService{}
	}
	state, err := stateRoot()
	if err != nil {
		return cli.UnavailableService{}
	}
	installation, err := local.NewInstallationStore(state)
	if err != nil {
		return cli.UnavailableService{}
	}
	return lifecycleService{projectapp.NewLifecycle(store, manifest.Codec{}, local.IdentityAllocator{}), installation, root}
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
	if override := os.Getenv("LINGO_STATE_ROOT"); override != "" {
		if !filepath.IsAbs(override) {
			return "", projectapp.ErrUnsafe
		}
		return filepath.Clean(override), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".axiom", "state"), nil
}

type lifecycleService struct {
	lifecycle    projectapp.Lifecycle
	installation local.InstallationStore
	projectsRoot string
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
