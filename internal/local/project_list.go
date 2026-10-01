package local

import (
	"context"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

const maxProjectListMetadataBytes = 1 << 20

// ListInstalled enumerates only ID-addressed records in protected local state.
// It deliberately does not validate source or repository availability.
func (s InstallationStore) ListInstalled(ctx context.Context) ([]projectapp.InstalledProject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := existingPrivateRoot(s.root)
	if errors.Is(err, ErrNotFound) {
		return []projectapp.InstalledProject{}, nil
	}
	if err != nil {
		return nil, ErrUnsafe
	}
	defer root.Close()
	projects, err := existingPrivateChild(root, "projects")
	if errors.Is(err, ErrNotFound) {
		return []projectapp.InstalledProject{}, nil
	}
	if err != nil {
		return nil, ErrUnsafe
	}
	defer projects.Close()
	names, err := childNames(projects)
	if err != nil {
		return nil, err
	}
	result := make([]projectapp.InstalledProject, 0, len(names))
	metadataBytes := 0
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resolved, category := readResolvedProject(projects, name)
		if category != "" {
			if category == "recovery_required" {
				return nil, ErrRecoveryRequired
			}
			return nil, ErrUnsafe
		}
		metadataBytes += len(resolved.ID) + len(resolved.Slug) + len(resolved.Source)
		if metadataBytes > maxProjectListMetadataBytes {
			return nil, ErrUnsafe
		}
		result = append(result, projectapp.InstalledProject{ID: resolved.ID, Slug: resolved.Slug, Source: resolved.Source})
	}
	return result, nil
}

// ReadInstalledProject reads display metadata from the record's portable source.
// Missing or inaccessible sources are availability facts, not loss of installation.
func (s InstallationStore) ReadInstalledProject(ctx context.Context, installed projectapp.InstalledProject) (project.Project, error) {
	if err := ctx.Err(); err != nil {
		return project.Project{}, err
	}
	if _, err := os.Lstat(installed.Source); err != nil {
		return project.Project{}, projectapp.ErrProjectDefinitionUnavailable
	}
	root, err := existingPrivateRoot(installed.Source)
	if err != nil {
		if errors.Is(err, ErrUnsafe) {
			return project.Project{}, ErrUnsafe
		}
		return project.Project{}, projectapp.ErrProjectDefinitionUnavailable
	}
	defer root.Close()
	entries, err := readDirectoryNamesBounded(root, 1)
	if err != nil || len(entries) != 1 || entries[0] != manifestName {
		return project.Project{}, ErrUnsafe
	}
	wire, err := readPrivateFile(root, manifestName)
	if err != nil {
		if errors.Is(err, ErrUnsafe) {
			return project.Project{}, ErrUnsafe
		}
		return project.Project{}, projectapp.ErrProjectDefinitionUnavailable
	}
	snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(issues) != 0 {
		return project.Project{}, ErrUnsafe
	}
	return snapshot.Project(), nil
}
