package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #231 read-only readiness adapters. None of them writes, follows a
// link, reads document content or consults the caller's working directory.

// ReadinessProjects loads one installed Project by exact UUID/slug selector.
// Selection does not check binding availability: readiness reports each
// Repository instead of failing the whole Project.
type ReadinessProjects struct {
	Installation InstallationStore
	Portable     PortableStore
}

func (r ReadinessProjects) LoadReadiness(ctx context.Context, selector string) (projectapp.ReadinessSubject, string) {
	selected := r.Installation.Select(ctx, selector)
	if selected.Status != ResolutionFound {
		switch selected.Category {
		case "project_not_found", "invalid_project_selector":
			return projectapp.ReadinessSubject{}, projectapp.BlockerProjectNotInstalled
		case "recovery_required":
			return projectapp.ReadinessSubject{}, projectapp.BlockerRecoveryRequired
		}
		return projectapp.ReadinessSubject{}, projectapp.BlockerProjectStateInvalid
	}
	resolved := selected.Project
	portable, err := r.Portable.InspectRecordedSource(ctx, resolved.Source, resolved.Slug)
	switch {
	case errors.Is(err, ErrRecoveryRequired):
		return projectapp.ReadinessSubject{}, projectapp.BlockerRecoveryRequired
	case err != nil:
		return projectapp.ReadinessSubject{}, projectapp.BlockerProjectStateInvalid
	case !portable.Exists:
		return projectapp.ReadinessSubject{}, projectapp.BlockerProjectSourceUnavailable
	}
	state := portable.Snapshot.Project().State()
	if state.ID != resolved.ID || state.Slug != resolved.Slug {
		return projectapp.ReadinessSubject{}, projectapp.BlockerProjectStateInvalid
	}
	// An edited source no longer matches what installation validated.
	if portable.Snapshot.Revision() != resolved.PortableRevision {
		return projectapp.ReadinessSubject{}, projectapp.BlockerInstallationStale
	}
	return projectapp.ReadinessSubject{Project: portable.Snapshot.Project(), Local: resolved.Local}, ""
}

// RepositoryProbe applies the same availability rule as Resolve.
type RepositoryProbe struct{}

func (RepositoryProbe) RepositoryAvailable(ctx context.Context, binding projectapp.RepositoryBinding) bool {
	if ctx.Err() != nil || !availableDirectory(binding.ExplicitPath) {
		return false
	}
	if binding.CanonicalIdentity == "" {
		return true
	}
	identity, err := DirectoryIdentity(binding.ExplicitPath)
	return err == nil && identity == binding.CanonicalIdentity
}

// DocumentationResolver classifies portable documentation sources through
// their declared Repository binding or machine-local file binding.
type DocumentationResolver struct{}

func (DocumentationResolver) ResolveDocumentation(ctx context.Context, request projectapp.DocumentationRequest) projectapp.DocumentationStatus {
	if ctx.Err() != nil {
		return projectapp.DocumentationMissing
	}
	switch request.Source.Kind {
	case project.RepositorySource:
		path, ok := request.Source.Path.Value()
		if request.RepositoryPath == "" {
			return projectapp.DocumentationRepositoryUnavailable
		}
		if !ok || !project.RepositoryRelativePath(path) {
			return projectapp.DocumentationUnsafe
		}
		return resolveContained(request.RepositoryPath, path)
	case project.LocalFileSource:
		if request.Binding == nil {
			return projectapp.DocumentationUnbound
		}
		return resolveLocalFile(*request.Binding)
	}
	return projectapp.DocumentationUnsafe
}

// resolveContained walks every component with lstat: a link or special file
// anywhere below the Repository root is unsafe, never followed.
func resolveContained(root, relative string) projectapp.DocumentationStatus {
	current := filepath.Clean(root)
	parts := strings.Split(relative, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return projectapp.DocumentationMissing
		}
		if err != nil {
			return projectapp.DocumentationUnsafe
		}
		mode := info.Mode()
		last := i == len(parts)-1
		if mode&os.ModeSymlink != 0 || !mode.IsDir() && !(last && mode.IsRegular()) {
			if !last && mode.IsRegular() {
				return projectapp.DocumentationMissing
			}
			return projectapp.DocumentationUnsafe
		}
	}
	return projectapp.DocumentationAvailable
}

func resolveLocalFile(binding projectapp.DocumentationBinding) projectapp.DocumentationStatus {
	info, err := os.Lstat(binding.ExplicitPath)
	switch {
	case os.IsNotExist(err):
		return projectapp.DocumentationMissing
	case err != nil || !info.Mode().IsRegular():
		return projectapp.DocumentationUnsafe
	}
	identity, err := FileIdentity(binding.ExplicitPath)
	if err != nil {
		return projectapp.DocumentationUnsafe
	}
	if identity != binding.CanonicalIdentity {
		return projectapp.DocumentationStale
	}
	return projectapp.DocumentationAvailable
}

// ObserveDocumentationFile creates a binding for an explicit absolute
// regular file at bootstrap. Content is never read.
func ObserveDocumentationFile(sourceKey, path string, observedAt func() projectapp.Observation) (projectapp.DocumentationBinding, bool) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return projectapp.DocumentationBinding{}, false
	}
	identity, err := FileIdentity(clean)
	if err != nil {
		return projectapp.DocumentationBinding{}, false
	}
	return projectapp.DocumentationBinding{SourceKey: sourceKey, ExplicitPath: clean, CanonicalIdentity: identity, Observation: observedAt()}, true
}
