package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 machine-local Project operational state (I230-T02). One strict
// versioned record lives beside installation.json, which is never extended.
// A missing record means the Project is active with no locally disabled
// Integration; a malformed, unknown-version, unsafe or interrupted record is
// never collapsed into that default.

const (
	OperationalFormatVersion  = 1
	operationalRecordName     = "operational.json"
	maxOperationalRecordBytes = 64 << 10
)

type operationalDTO struct {
	FormatVersion        int      `json:"formatVersion"`
	ProjectID            string   `json:"projectId"`
	ProjectStatus        string   `json:"projectStatus"`
	DisabledIntegrations []string `json:"disabledIntegrations"`
}

// EncodeOperational emits the closed canonical DTO and verifies it with the
// complete decoder before returning bytes.
func EncodeOperational(projectID string, state projectapp.OperationalState) ([]byte, error) {
	if len(project.ValidateIdentity(projectID, "operational")) != 0 || !projectapp.ValidOperationalState(state) {
		return nil, ErrUnsafe
	}
	wire, err := json.Marshal(operationalDTO{FormatVersion: OperationalFormatVersion, ProjectID: projectID, ProjectStatus: string(state.ProjectStatus), DisabledIntegrations: state.DisabledIntegrations})
	if err != nil {
		return nil, ErrUnsafe
	}
	wire = append(wire, '\n')
	if _, _, err := DecodeOperational(wire); err != nil {
		return nil, err
	}
	return wire, nil
}

// DecodeOperational accepts exactly format 1 in its canonical encoding:
// duplicate, unknown or missing fields, null values, other versions,
// contradictory sets and noncanonical bytes all fail closed.
func DecodeOperational(wire []byte) (string, projectapp.OperationalState, error) {
	if len(wire) == 0 || len(wire) > maxOperationalRecordBytes {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	tree, issues := parseRecord(wire)
	if len(issues) != 0 {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	object, ok := tree.(map[string]any)
	if !ok {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	if version, ok := object["formatVersion"].(json.Number); !ok || string(version) != "1" {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	if issues := checkShape(object, reflect.TypeFor[operationalDTO](), "operational"); len(issues) != 0 {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	var dto operationalDTO
	if err := json.Unmarshal(wire, &dto); err != nil {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	state := projectapp.OperationalState{ProjectStatus: projectapp.ProjectStatus(dto.ProjectStatus), DisabledIntegrations: dto.DisabledIntegrations}
	if len(project.ValidateIdentity(dto.ProjectID, "operational")) != 0 || !projectapp.ValidOperationalState(state) {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	canonical, err := json.Marshal(dto)
	if err != nil || !bytes.Equal(append(canonical, '\n'), wire) {
		return "", projectapp.OperationalState{}, ErrUnsafe
	}
	return dto.ProjectID, state, nil
}

// OperationalStore persists operational.json inside the existing ID-addressed
// installation directory using the ADR-0007 file publication protocol, so an
// interrupted write is classified by recovery inspect/apply.
type OperationalStore struct {
	root  string
	hooks publicationHooks
}

func NewOperationalStore(root string) (OperationalStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return OperationalStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return OperationalStore{}, err
	}
	return OperationalStore{root: canonical}, nil
}

func (s OperationalStore) InspectOperational(ctx context.Context, projectID string) (projectapp.OperationalObservation, error) {
	if err := ctx.Err(); err != nil {
		return projectapp.OperationalObservation{}, err
	}
	root, projects, target, err := s.open(projectID)
	if err != nil {
		return projectapp.OperationalObservation{}, err
	}
	defer root.Close()
	defer projects.Close()
	defer target.Close()
	locks, err := lockRoots(false, root, projects, target)
	if err != nil {
		return projectapp.OperationalObservation{}, err
	}
	defer closeFiles(locks)
	observation, _, err := observeOperational(target, projectID)
	return observation, err
}

// CommitOperational replaces the complete record only while its exact current
// revision equals expected. It never creates the installation directory: a
// Project that is not installed has no operational state to change.
func (s OperationalStore) CommitOperational(ctx context.Context, projectID, expected string, next projectapp.OperationalState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	wire, err := EncodeOperational(projectID, next)
	if err != nil {
		return err
	}
	root, projects, target, err := s.open(projectID)
	if err != nil {
		return err
	}
	defer root.Close()
	defer projects.Close()
	defer target.Close()
	locks, err := lockRoots(true, root, projects, target)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	observation, current, err := observeOperational(target, projectID)
	if err != nil {
		return err
	}
	if observation.Revision != expected {
		return ErrConflict
	}
	if bytes.Equal(current, wire) {
		return nil
	}
	hooks := s.hooks
	testBeforeCommit := hooks.beforeCommit
	// Commit only into the directory still visible at the canonical address;
	// a replaced or relinked chain fails before the commit point.
	hooks.beforeCommit = func() error {
		if testBeforeCommit != nil {
			if err := testBeforeCommit(); err != nil {
				return err
			}
		}
		for _, check := range []struct {
			root *os.Root
			path string
		}{
			{root, s.root},
			{projects, filepath.Join(s.root, "projects")},
			{target, filepath.Join(s.root, "projects", projectID)},
		} {
			if err := stillAtPath(check.root, check.path); err != nil {
				return err
			}
		}
		return nil
	}
	return publishFile(ctx, target, operationalRecordName, current, wire, !observation.Exists, hooks)
}

func (s OperationalStore) open(projectID string) (*os.Root, *os.Root, *os.Root, error) {
	if len(project.ValidateIdentity(projectID, "operational")) != 0 {
		return nil, nil, nil, ErrUnsafe
	}
	root, err := existingPrivateRoot(s.root)
	if err != nil {
		return nil, nil, nil, err
	}
	projects, err := existingPrivateChild(root, "projects")
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	target, err := existingPrivateChild(projects, projectID)
	if err != nil {
		projects.Close()
		root.Close()
		return nil, nil, nil, err
	}
	return root, projects, target, nil
}

// observeOperational requires a coherent installation directory with a
// readable installation record before it interprets operational state.
func observeOperational(target *os.Root, projectID string) (projectapp.OperationalObservation, []byte, error) {
	switch installationDirectoryIssue(target) {
	case "":
	case "recovery_required":
		return projectapp.OperationalObservation{}, nil, ErrRecoveryRequired
	default:
		return projectapp.OperationalObservation{}, nil, ErrUnsafe
	}
	if _, err := readPrivateFile(target, "installation.json"); err != nil {
		if os.IsNotExist(err) {
			return projectapp.OperationalObservation{}, nil, ErrNotFound
		}
		return projectapp.OperationalObservation{}, nil, ErrUnsafe
	}
	wire, err := readPublishedFile(target, operationalRecordName)
	if os.IsNotExist(err) {
		return projectapp.OperationalObservation{Revision: projectapp.OperationalRevisionAbsent, State: projectapp.DefaultOperationalState()}, nil, nil
	}
	if err != nil {
		return projectapp.OperationalObservation{}, nil, err
	}
	id, state, err := DecodeOperational(wire)
	if err != nil || id != projectID {
		return projectapp.OperationalObservation{}, nil, ErrUnsafe
	}
	digest := sha256.Sum256(wire)
	return projectapp.OperationalObservation{Exists: true, Revision: hex.EncodeToString(digest[:]), State: state}, wire, nil
}
