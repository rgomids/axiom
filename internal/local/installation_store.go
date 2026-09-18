package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
)

type InstallationStore struct{ root string }
type InstallationStatus string

const (
	InstallationApplied   InstallationStatus = "applied"
	InstallationUnchanged InstallationStatus = "unchanged"
	InstallationFailed    InstallationStatus = "failed"
	InstallationConflict  InstallationStatus = "conflict"
)

type InstallationResult struct {
	Status   InstallationStatus
	Category string
}

func NewInstallationStore(root string) (InstallationStore, error) {
	if !filepath.IsAbs(root) {
		return InstallationStore{}, ErrUnsafe
	}
	root = filepath.Clean(root)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return InstallationStore{}, err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return InstallationStore{}, ErrUnsafe
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return InstallationStore{}, err
	}
	return InstallationStore{root: root}, nil
}

func (s InstallationStore) Install(ctx context.Context, source string) InstallationResult {
	if err := ctx.Err(); err != nil {
		return failedInstallation("cancelled")
	}
	if !filepath.IsAbs(source) {
		return failedInstallation("invalid_input")
	}
	source = filepath.Clean(source)
	if err := onlyManifest(source); err != nil {
		return installationError(err)
	}
	bytes, err := readRegular(filepath.Join(source, manifestName))
	if err != nil {
		return installationError(err)
	}
	snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, bytes, nil)
	if len(issues) != 0 {
		return failedInstallation("invalid_project")
	}
	record, recordIssues := NewRecord(RecordState{ProjectID: snapshot.Project().State().ID, ObservedSlug: snapshot.Project().State().Slug, SourceLocation: source, PortableRevision: snapshot.Revision(), ArtifactDigests: snapshot.Digests()})
	if len(recordIssues) != 0 {
		return failedInstallation("invalid_local_state")
	}
	wire, recordIssues := EncodeRecord(record)
	if len(recordIssues) != 0 {
		return failedInstallation("invalid_local_state")
	}
	target := filepath.Join(s.root, "projects", snapshot.Project().State().ID)
	if err := os.MkdirAll(target, 0o700); err != nil {
		return failedInstallation("storage_failure")
	}
	path := filepath.Join(target, "installation.json")
	if prior, err := os.ReadFile(path); err == nil {
		existing, _, issues := DecodeObservedRecord(prior, true)
		if len(issues) != 0 {
			return failedInstallation("invalid_existing_local_state")
		}
		state := existing.State()
		if state.SourceLocation == source && state.PortableRevision == snapshot.Revision() {
			return InstallationResult{Status: InstallationUnchanged, Category: "already_installed"}
		}
		return InstallationResult{Status: InstallationConflict, Category: "explicit_replacement_required"}
	} else if !os.IsNotExist(err) {
		return failedInstallation("storage_failure")
	}
	if err := writeDurable(path, wire); err != nil {
		return failedInstallation("storage_failure")
	}
	return InstallationResult{Status: InstallationApplied, Category: "installed"}
}

func installationError(err error) InstallationResult {
	if errors.Is(err, ErrNotFound) {
		return failedInstallation("project_not_found")
	}
	if errors.Is(err, ErrUnsafe) {
		return failedInstallation("unsafe_source")
	}
	return failedInstallation("storage_failure")
}
func failedInstallation(category string) InstallationResult {
	return InstallationResult{Status: InstallationFailed, Category: category}
}
