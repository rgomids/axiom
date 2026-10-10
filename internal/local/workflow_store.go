package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const MaxExecutionBytes = 4 << 20

func readExecutionFile(root *os.Root, name string) ([]byte, error) {
	pending, err := protocolStatePresent(root)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrRecoveryRequired
	}
	return readPrivateFileBounded(root, name, MaxExecutionBytes)
}

type WorkflowStore struct {
	root  string
	hooks publicationHooks
}

type executionDTO struct {
	Binding         *workflow.WorkflowBinding   `json:"binding,omitempty"`
	StageOrdinal    int                         `json:"stageOrdinal,omitempty"`
	FormatVersion   int                         `json:"formatVersion"`
	ExecutionID     string                      `json:"executionId"`
	WorkflowVersion string                      `json:"workflowVersion"`
	ProjectID       string                      `json:"projectId"`
	RepositoryKey   string                      `json:"repositoryKey"`
	WorkItem        workflow.WorkItem           `json:"workItem"`
	RuntimeID       string                      `json:"runtimeId"`
	Stage           workflow.Stage              `json:"stage"`
	Revision        uint64                      `json:"revision"`
	Status          workflow.ExecutionStatus    `json:"status"`
	Transitions     []workflow.Transition       `json:"transitions"`
	CreatedAt       string                      `json:"createdAt"`
	UpdatedAt       string                      `json:"updatedAt"`
	Provenance      workflow.Identity           `json:"provenance"`
	Terminal        *workflow.Terminal          `json:"terminal,omitempty"`
	Projections     []workflow.ProjectionRecord `json:"projections"`
}

func NewWorkflowStore(root string) (WorkflowStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) == string(filepath.Separator) {
		return WorkflowStore{}, ErrUnsafe
	}
	canonical, err := trustedCanonical(root)
	if err != nil {
		return WorkflowStore{}, err
	}
	return WorkflowStore{root: canonical}, nil
}

func (s WorkflowStore) Create(ctx context.Context, state workflow.State) error {
	return s.write(ctx, state, true)
}

func (s WorkflowStore) Save(ctx context.Context, state workflow.State) error {
	return s.write(ctx, state, false)
}

func (s WorkflowStore) write(ctx context.Context, state workflow.State, create bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	wire, err := encodeExecution(state)
	if err != nil {
		return workflowStoreError(err)
	}
	root, executions, version, projectRoot, err := s.openProject(state.ProjectID, true)
	if err != nil {
		return workflowStoreError(err)
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(true, root, executions, version, projectRoot)
	if err != nil {
		return workflowStoreError(err)
	}
	defer closeFiles(locks)
	name := executionName(state.RepositoryKey, state.WorkItem)
	var expected []byte
	if !create {
		expected, err = readExecutionFile(projectRoot, name)
		if err != nil {
			return workflowStoreError(err)
		}
		if state.StorageRevision == ([sha256.Size]byte{}) || sha256.Sum256(expected) != state.StorageRevision {
			return workflowStoreError(ErrConflict)
		}
		prior, e := decodeExecution(expected)
		if e != nil || !sameExecutionBinding(prior, state) {
			return workflow.ErrRecoveryRequired
		}
	}
	hooks := s.hooks
	hooks.contentLimit = MaxExecutionBytes
	return workflowStoreError(publishFile(ctx, projectRoot, name, expected, wire, create, hooks))
}

func (s WorkflowStore) Load(ctx context.Context, projectID, repositoryKey string, item workflow.WorkItem) (workflow.State, error) {
	if err := ctx.Err(); err != nil {
		return workflow.State{}, err
	}
	if !validExecutionAddress(projectID, repositoryKey, item) {
		return workflow.State{}, workflowStoreError(ErrUnsafe)
	}
	root, executions, version, projectRoot, err := s.openProject(projectID, false)
	if err != nil {
		return workflow.State{}, workflowStoreError(err)
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(false, root, executions, version, projectRoot)
	if err != nil {
		return workflow.State{}, workflowStoreError(err)
	}
	defer closeFiles(locks)
	wire, err := readExecutionFile(projectRoot, executionName(repositoryKey, item))
	if err != nil {
		return workflow.State{}, workflowStoreError(err)
	}
	state, err := decodeExecution(wire)
	if err != nil {
		return workflow.State{}, workflowStoreError(err)
	}
	if state.ProjectID != projectID || state.RepositoryKey != repositoryKey || state.WorkItem != item {
		return workflow.State{}, workflowStoreError(ErrUnsafe)
	}
	state.StorageRevision = sha256.Sum256(wire)
	return state, nil
}

// List enumerates every Execution record of one Project under the same shared
// lock as Load. Every entry is strictly decoded and must be addressed exactly
// by its own identity; a protocol state, undecodable, foreign or misaddressed
// record fails the whole listing closed. A Project with no records is empty.
func (s WorkflowStore) List(ctx context.Context, projectID string) ([]workflow.State, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(project.ValidateIdentity(projectID, "execution")) != 0 {
		return nil, workflowStoreError(ErrUnsafe)
	}
	root, executions, version, projectRoot, err := s.openProject(projectID, false)
	if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
		return []workflow.State{}, nil
	}
	if err != nil {
		return nil, workflowStoreError(err)
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(false, root, executions, version, projectRoot)
	if err != nil {
		return nil, workflowStoreError(err)
	}
	defer closeFiles(locks)
	if pending, err := protocolStatePresent(projectRoot); err != nil {
		return nil, workflowStoreError(err)
	} else if pending {
		return nil, workflowStoreError(ErrRecoveryRequired)
	}
	names, err := childNames(projectRoot)
	if err != nil {
		return nil, workflowStoreError(err)
	}
	states := make([]workflow.State, 0, len(names))
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		wire, err := readExecutionFile(projectRoot, name)
		if err != nil {
			return nil, workflowStoreError(err)
		}
		state, err := decodeExecution(wire)
		if err != nil || state.ProjectID != projectID || executionName(state.RepositoryKey, state.WorkItem) != name {
			return nil, workflowStoreError(ErrUnsafe)
		}
		state.StorageRevision = sha256.Sum256(wire)
		states = append(states, state)
	}
	return states, nil
}

func (s WorkflowStore) openProject(projectID string, create bool) (*os.Root, *os.Root, *os.Root, *os.Root, error) {
	openRoot := existingPrivateRoot
	openChild := existingPrivateChild
	if create {
		openRoot = privateRoot
		openChild = privateChild
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	executions, err := openChild(root, "executions")
	if err != nil {
		root.Close()
		return nil, nil, nil, nil, err
	}
	version, err := openChild(executions, "v1")
	if err != nil {
		executions.Close()
		root.Close()
		return nil, nil, nil, nil, err
	}
	projectRoot, err := openChild(version, projectID)
	if err != nil {
		version.Close()
		executions.Close()
		root.Close()
		return nil, nil, nil, nil, err
	}
	return root, executions, version, projectRoot, nil
}

func encodeExecution(state workflow.State) ([]byte, error) {
	if !workflow.ValidState(state) {
		return nil, ErrUnsafe
	}
	dto := executionDTO{
		Binding: state.Binding, StageOrdinal: state.StageOrdinal,
		FormatVersion: state.FormatVersion, ExecutionID: state.ExecutionID, WorkflowVersion: state.WorkflowVersion,
		ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, WorkItem: state.WorkItem,
		RuntimeID: state.RuntimeID, Stage: state.Stage, Revision: state.Revision, Status: state.Status,
		Transitions: state.Transitions, CreatedAt: state.CreatedAt.UTC().Format(timeFormat), UpdatedAt: state.UpdatedAt.UTC().Format(timeFormat),
		Provenance: state.Provenance, Terminal: state.Terminal, Projections: state.Projections,
	}
	var buffer bytes.Buffer
	if state.FormatVersion == 1 {
		wire, err := json.Marshal(dto)
		if err != nil || len(wire)+1 > MaxRecordBytes {
			return nil, ErrUnsafe
		}
		return append(wire, '\n'), nil
	}
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(dto)
	wire := bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'})
	if err != nil || len(wire)+1 > MaxExecutionBytes || state.FormatVersion == 1 && len(wire)+1 > MaxRecordBytes {
		return nil, ErrUnsafe
	}
	wire = append(wire, '\n')
	if _, err := decodeExecution(wire); err != nil {
		return nil, err
	}
	return wire, nil
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func decodeExecution(wire []byte) (workflow.State, error) {
	if len(wire) == 0 || len(wire) > MaxExecutionBytes {
		return workflow.State{}, ErrUnsafe
	}
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(wire), MaxExecutionBytes+1))
	decoder.DisallowUnknownFields()
	var dto executionDTO
	if err := decoder.Decode(&dto); err != nil {
		return workflow.State{}, ErrUnsafe
	}
	if dto.FormatVersion == 2 && workflowdefinition.StrictJSONBounded(wire, &dto, MaxExecutionBytes) != nil {
		return workflow.State{}, ErrUnsafe
	}
	if dto.FormatVersion == 1 && len(wire) > MaxRecordBytes {
		return workflow.State{}, ErrUnsafe
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return workflow.State{}, ErrUnsafe
	}
	created, err := parseExecutionTime(dto.CreatedAt)
	if err != nil {
		return workflow.State{}, ErrUnsafe
	}
	updated, err := parseExecutionTime(dto.UpdatedAt)
	if err != nil {
		return workflow.State{}, ErrUnsafe
	}
	state := workflow.State{
		Binding: dto.Binding, StageOrdinal: dto.StageOrdinal,
		ExecutionID: dto.ExecutionID, FormatVersion: dto.FormatVersion, WorkflowVersion: dto.WorkflowVersion,
		ProjectID: dto.ProjectID, RepositoryKey: dto.RepositoryKey, WorkItem: dto.WorkItem, RuntimeID: dto.RuntimeID,
		Stage: dto.Stage, Revision: dto.Revision, Status: dto.Status, Transitions: dto.Transitions,
		CreatedAt: created, UpdatedAt: updated, Provenance: dto.Provenance, Terminal: dto.Terminal, Projections: dto.Projections,
	}
	if !workflow.ValidState(state) {
		return workflow.State{}, ErrUnsafe
	}
	return state, nil
}

func sameExecutionBinding(left, right workflow.State) bool {
	a, _ := json.Marshal(left.Binding)
	b, _ := json.Marshal(right.Binding)
	if !bytes.Equal(a, b) || left.ExecutionID != right.ExecutionID || left.FormatVersion != right.FormatVersion || left.WorkflowVersion != right.WorkflowVersion || left.ProjectID != right.ProjectID || left.RepositoryKey != right.RepositoryKey || left.WorkItem != right.WorkItem || left.RuntimeID != right.RuntimeID {
		return false
	}
	if left.Binding != nil {
		if !left.CreatedAt.Equal(right.CreatedAt) || left.Provenance != right.Provenance || len(right.Transitions) < len(left.Transitions) {
			return false
		}
		for i, event := range left.Transitions {
			if !reflect.DeepEqual(event, right.Transitions[i]) {
				return false
			}
		}
	}
	return true
}

func parseExecutionTime(value string) (time.Time, error) {
	return time.Parse(timeFormat, value)
}

func validExecutionAddress(projectID, repositoryKey string, item workflow.WorkItem) bool {
	if len(project.ValidateIdentity(projectID, "execution")) != 0 || !project.ValidSlug(repositoryKey) {
		return false
	}
	probe := workflow.State{
		ExecutionID: "00000000-0000-4000-8000-000000000000", FormatVersion: workflow.FormatVersion,
		WorkflowVersion: workflow.WorkflowVersion, ProjectID: projectID, RepositoryKey: repositoryKey,
		WorkItem: item, RuntimeID: "probe", Stage: workflow.Intake, Revision: 1, Status: workflow.ExecutionActive,
		CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(1, 0).UTC(),
		Provenance: workflow.Identity{Product: "Axiom", Version: "development", Revision: "unavailable", SourceState: "unknown"},
	}
	return workflow.ValidState(probe)
}

func executionName(repositoryKey string, item workflow.WorkItem) string {
	value := sha256.Sum256([]byte(repositoryKey + "\x00" + item.Provider + "\x00" + item.Resource + "\x00" + item.ExternalID))
	return hex.EncodeToString(value[:]) + ".json"
}

func workflowStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrRecoveryRequired):
		return errors.Join(err, workflow.ErrRecoveryRequired)
	case errors.Is(err, ErrConflict):
		return errors.Join(err, workflow.ErrConflict)
	case errors.Is(err, ErrNotFound), os.IsNotExist(err):
		return errors.Join(err, workflow.ErrNotFound)
	default:
		return errors.Join(err, workflow.ErrRecoveryRequired)
	}
}
