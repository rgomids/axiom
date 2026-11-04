package local

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

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

// WorkflowStore reads inert legacy state; canonical writes use ExecutionStore.
type WorkflowStore struct {
	root string
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

func (s WorkflowStore) openProject(projectID string) (*os.Root, *os.Root, *os.Root, *os.Root, error) {
	root, err := existingPrivateRoot(s.root)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	executions, err := existingPrivateChild(root, "executions")
	if err != nil {
		root.Close()
		return nil, nil, nil, nil, err
	}
	version, err := existingPrivateChild(executions, "v1")
	if err != nil {
		executions.Close()
		root.Close()
		return nil, nil, nil, nil, err
	}
	projectRoot, err := existingPrivateChild(version, projectID)
	if err != nil {
		version.Close()
		executions.Close()
		root.Close()
		return nil, nil, nil, nil, err
	}
	return root, executions, version, projectRoot, nil
}

func encodeExecutionSnapshot(state workflow.State) ([]byte, error) {
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

func decodeExecutionSnapshot(wire []byte) (workflow.State, error) {
	if len(wire) == 0 || len(wire) > MaxExecutionBytes {
		return workflow.State{}, ErrUnsafe
	}
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(wire), MaxExecutionBytes+1))
	decoder.DisallowUnknownFields()
	var dto executionDTO
	if err := decoder.Decode(&dto); err != nil {
		return workflow.State{}, ErrUnsafe
	}
	if (dto.FormatVersion == 2 || dto.FormatVersion == 3 || dto.FormatVersion == workflow.OperationalConfiguredFormatVersion) && workflowdefinition.StrictJSONBounded(wire, &dto, MaxExecutionBytes) != nil {
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
