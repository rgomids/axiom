package local

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/projectapp"
)

type InstallationStore struct {
	root              string
	beforePublication func()
	afterPublication  func()
}
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

func NewInstallationStore(path string) (InstallationStore, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) == string(filepath.Separator) {
		return InstallationStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(path)
	if err != nil {
		return InstallationStore{}, err
	}
	return InstallationStore{root: canonical}, nil
}

func (s InstallationStore) Install(ctx context.Context, source string) InstallationResult {
	snapshot, result := portableSnapshot(ctx, source)
	if result.Status == InstallationFailed {
		return result
	}
	source = filepath.Clean(source)
	record, issues := NewRecord(RecordState{ProjectID: snapshot.Project().State().ID, ObservedSlug: snapshot.Project().State().Slug, SourceLocation: source, PortableRevision: snapshot.Revision(), ArtifactDigests: snapshot.Digests()})
	if len(issues) != 0 {
		return failedInstallation("invalid_local_state")
	}
	wire, issues := EncodeRecord(record)
	if len(issues) != 0 {
		return failedInstallation("invalid_local_state")
	}
	return s.withIDLock(true, func(projects *os.Root) InstallationResult {
		target, err := privateChild(projects, snapshot.Project().State().ID)
		if err != nil {
			return failedInstallation("storage_failure")
		}
		defer target.Close()
		if category := installationDirectoryIssue(target); category != "" {
			return failedInstallation(category)
		}
		prior, err := readPrivateFile(target, "installation.json")
		if err == nil {
			if _, _, issues := DecodeObservedRecord(prior, true); len(issues) != 0 {
				return failedInstallation("invalid_existing_local_state")
			}
			if bytes.Equal(prior, wire) {
				return InstallationResult{Status: InstallationUnchanged, Category: "already_installed"}
			}
			return InstallationResult{Status: InstallationConflict, Category: "explicit_replacement_required"}
		}
		if !os.IsNotExist(err) {
			if errors.Is(err, ErrUnsafe) {
				return failedInstallation("invalid_existing_local_state")
			}
			return failedInstallation("storage_failure")
		}
		again, result := portableSnapshot(ctx, source)
		if result.Status == InstallationFailed || again.Revision() != snapshot.Revision() {
			return failedInstallation("source_changed")
		}
		if err := ctx.Err(); err != nil {
			return failedInstallation("cancelled")
		}
		temporary, err := temporaryName(".lingo-install-")
		if err != nil {
			return failedInstallation("storage_failure")
		}
		published := false
		defer func() {
			if !published {
				_ = target.Remove(temporary)
			}
		}()
		if err := writePrivateFile(target, temporary, wire); err != nil {
			return failedInstallation("storage_failure")
		}
		if err := ctx.Err(); err != nil {
			return failedInstallation("cancelled")
		}
		attempt, err := markAttempt(target, ".lingo-attempt-install-")
		if err != nil {
			return failedInstallation("recovery_required")
		}
		if err := ctx.Err(); err != nil {
			if clearAttempt(target, attempt) != nil {
				return failedInstallation("recovery_required")
			}
			return failedInstallation("cancelled")
		}
		if s.beforePublication != nil {
			s.beforePublication()
		}
		if err := ctx.Err(); err != nil {
			if clearAttempt(target, attempt) != nil {
				return failedInstallation("recovery_required")
			}
			return failedInstallation("cancelled")
		}
		if err := renameNoReplace(target, temporary, "installation.json"); err != nil {
			if clearAttempt(target, attempt) != nil {
				return failedInstallation("recovery_required")
			}
			if os.IsExist(err) {
				return InstallationResult{Status: InstallationConflict, Category: "explicit_replacement_required"}
			}
			return failedInstallation("storage_failure")
		}
		published = true
		if s.afterPublication != nil {
			s.afterPublication()
		}
		if err := syncRoot(target); err != nil {
			return failedInstallation("recovery_required")
		}
		if err := syncRoot(projects); err != nil {
			return failedInstallation("recovery_required")
		}
		if err := clearAttempt(target, attempt); err != nil {
			return failedInstallation("recovery_required")
		}
		return InstallationResult{Status: InstallationApplied, Category: "installed"}
	})
}

func (s InstallationStore) Reopen(ctx context.Context, source string) InstallationResult {
	snapshot, result := portableSnapshot(ctx, source)
	if result.Status == InstallationFailed {
		return result
	}
	source = filepath.Clean(source)
	return s.withIDLock(false, func(projects *os.Root) InstallationResult {
		target, err := existingPrivateChild(projects, snapshot.Project().State().ID)
		if errors.Is(err, ErrNotFound) {
			return InstallationResult{Status: InstallationUnchanged, Category: "reopened_without_local_state"}
		}
		if err != nil {
			return failedInstallation("storage_failure")
		}
		defer target.Close()
		if category := installationDirectoryIssue(target); category != "" {
			return failedInstallation(category)
		}
		wire, err := readPrivateFile(target, "installation.json")
		if os.IsNotExist(err) {
			return InstallationResult{Status: InstallationUnchanged, Category: "reopened_without_local_state"}
		}
		if err != nil {
			return failedInstallation("invalid_existing_local_state")
		}
		record, _, issues := DecodeObservedRecord(wire, true)
		if len(issues) != 0 {
			return failedInstallation("invalid_existing_local_state")
		}
		state := record.State()
		if state.SourceLocation != source || state.PortableRevision != snapshot.Revision() {
			return InstallationResult{Status: InstallationConflict, Category: "local_state_revalidation_required"}
		}
		return InstallationResult{Status: InstallationUnchanged, Category: "reopened_with_local_state"}
	})
}

func (s InstallationStore) withIDLock(create bool, action func(*os.Root) InstallationResult) InstallationResult {
	open := existingPrivateRoot
	if create {
		open = privateRoot
	}
	root, err := open(s.root)
	if errors.Is(err, ErrNotFound) && !create {
		return InstallationResult{Status: InstallationUnchanged, Category: "reopened_without_local_state"}
	}
	if err != nil {
		return failedInstallation("storage_failure")
	}
	defer root.Close()
	lock, err := lockDirectory(root, create)
	if err != nil {
		return failedInstallation("storage_failure")
	}
	defer lock.Close()
	var projects *os.Root
	if create {
		projects, err = privateChild(root, "projects")
	} else {
		projects, err = existingPrivateChild(root, "projects")
	}
	if errors.Is(err, ErrNotFound) && !create {
		return InstallationResult{Status: InstallationUnchanged, Category: "reopened_without_local_state"}
	}
	if err != nil {
		return failedInstallation("storage_failure")
	}
	defer projects.Close()
	return action(projects)
}

func portableSnapshot(ctx context.Context, source string) (projectapp.ArtifactSnapshot, InstallationResult) {
	if err := ctx.Err(); err != nil {
		return projectapp.ArtifactSnapshot{}, failedInstallation("cancelled")
	}
	if !filepath.IsAbs(source) {
		return projectapp.ArtifactSnapshot{}, failedInstallation("invalid_input")
	}
	if _, err := os.Lstat(source); os.IsNotExist(err) {
		return projectapp.ArtifactSnapshot{}, failedInstallation("project_not_found")
	} else if err != nil {
		return projectapp.ArtifactSnapshot{}, failedInstallation("unsafe_source")
	}
	root, err := existingPrivateRoot(source)
	if err != nil {
		return projectapp.ArtifactSnapshot{}, failedInstallation("unsafe_source")
	}
	defer root.Close()
	file, err := root.Open(".")
	if err != nil {
		return projectapp.ArtifactSnapshot{}, failedInstallation("unsafe_source")
	}
	entries, err := file.Readdirnames(-1)
	file.Close()
	if err != nil || len(entries) != 1 || entries[0] != manifestName {
		return projectapp.ArtifactSnapshot{}, failedInstallation("unsafe_source")
	}
	input, err := readPrivateFile(root, manifestName)
	if err != nil {
		return projectapp.ArtifactSnapshot{}, failedInstallation("unsafe_source")
	}
	snapshot, issues := projectapp.ReadSnapshot(manifest.Codec{}, input, nil)
	if len(issues) != 0 {
		return projectapp.ArtifactSnapshot{}, failedInstallation("invalid_project")
	}
	return snapshot, InstallationResult{Status: InstallationApplied}
}

func failedInstallation(category string) InstallationResult {
	return InstallationResult{Status: InstallationFailed, Category: category}
}

func installationDirectoryIssue(root *os.Root) string {
	file, err := root.Open(".")
	if err != nil {
		return "storage_failure"
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return "storage_failure"
	}
	for _, name := range names {
		if strings.HasPrefix(name, ".lingo-install-") || strings.HasPrefix(name, ".lingo-attempt-install-") {
			return "recovery_required"
		}
		if name != "installation.json" {
			return "invalid_existing_local_state"
		}
	}
	return ""
}
