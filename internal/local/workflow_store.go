package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/workflow"
)

type WorkflowStore struct {
	root  string
	hooks publicationHooks
}

type workflowDTO struct {
	FormatVersion  int               `json:"formatVersion"`
	ProjectID      string            `json:"projectId"`
	RepositoryKey  string            `json:"repositoryKey"`
	RepositoryPath string            `json:"repositoryPath"`
	WorkItem       int               `json:"workItem"`
	Status         string            `json:"status"`
	Current        int               `json:"current"`
	Steps          []workflowStepDTO `json:"steps"`
}
type workflowStepDTO struct {
	Gate      string `json:"gate"`
	Status    string `json:"status"`
	Reference string `json:"reference,omitempty"`
	Digest    string `json:"digest,omitempty"`
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
	wire, err := encodeWorkflow(state)
	if err != nil {
		return ErrUnsafe
	}
	root, workflows, projectRoot, err := s.openProject(state.ProjectID, true)
	if err != nil {
		return err
	}
	defer root.Close()
	defer workflows.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(true, root, workflows, projectRoot)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	name := workflowName(state.RepositoryKey, state.WorkItem)
	var expected []byte
	if !create {
		expected, err = readPublishedFile(projectRoot, name)
		if err != nil {
			return err
		}
		if state.Revision == ([32]byte{}) || sha256.Sum256(expected) != state.Revision {
			return ErrConflict
		}
	}
	err = publishFile(ctx, projectRoot, name, expected, wire, create, s.hooks)
	return workflowStoreError(err)
}

func (s WorkflowStore) Load(ctx context.Context, projectID, repositoryKey string, workItem int) (workflow.State, error) {
	if err := ctx.Err(); err != nil {
		return workflow.State{}, err
	}
	if !validWorkflowAddress(projectID, repositoryKey, workItem) {
		return workflow.State{}, ErrUnsafe
	}
	root, workflows, projectRoot, err := s.openProject(projectID, false)
	if err != nil {
		return workflow.State{}, err
	}
	defer root.Close()
	defer workflows.Close()
	defer projectRoot.Close()
	locks, err := lockRoots(false, root, workflows, projectRoot)
	if err != nil {
		return workflow.State{}, err
	}
	defer closeFiles(locks)
	wire, err := readPublishedFile(projectRoot, workflowName(repositoryKey, workItem))
	if err != nil {
		return workflow.State{}, workflowStoreError(err)
	}
	state, err := decodeWorkflow(wire)
	if err == nil {
		state.Revision = sha256.Sum256(wire)
	}
	return state, err
}

func workflowStoreError(err error) error {
	if errors.Is(err, ErrRecoveryRequired) {
		return errors.Join(err, workflow.ErrRecoveryRequired)
	}
	return err
}

func (s WorkflowStore) openProject(projectID string, create bool) (*os.Root, *os.Root, *os.Root, error) {
	openRoot := existingPrivateRoot
	if create {
		openRoot = privateRoot
	}
	root, err := openRoot(s.root)
	if err != nil {
		return nil, nil, nil, err
	}
	openChild := existingPrivateChild
	if create {
		openChild = privateChild
	}
	workflows, err := openChild(root, "workflows")
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	projectRoot, err := openChild(workflows, projectID)
	if err != nil {
		workflows.Close()
		root.Close()
		return nil, nil, nil, err
	}
	return root, workflows, projectRoot, nil
}

func encodeWorkflow(state workflow.State) ([]byte, error) {
	if !validWorkflowState(state) {
		return nil, ErrUnsafe
	}
	dto := workflowDTO{FormatVersion: 1, ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, RepositoryPath: state.RepositoryPath, WorkItem: state.WorkItem, Status: state.Status, Current: state.Current, Steps: make([]workflowStepDTO, len(state.Steps))}
	for index, step := range state.Steps {
		digest := ""
		if step.Digest != ([32]byte{}) {
			digest = hex.EncodeToString(step.Digest[:])
		}
		dto.Steps[index] = workflowStepDTO{step.Gate, step.Status, step.Reference, digest}
	}
	wire, err := json.Marshal(dto)
	return append(wire, '\n'), err
}

func decodeWorkflow(wire []byte) (workflow.State, error) {
	decoder := json.NewDecoder(io.LimitReader(bytes.NewReader(wire), MaxRecordBytes+1))
	decoder.DisallowUnknownFields()
	var dto workflowDTO
	if err := decoder.Decode(&dto); err != nil {
		return workflow.State{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || dto.FormatVersion != 1 {
		return workflow.State{}, ErrUnsafe
	}
	state := workflow.State{ProjectID: dto.ProjectID, RepositoryKey: dto.RepositoryKey, RepositoryPath: dto.RepositoryPath, WorkItem: dto.WorkItem, Status: dto.Status, Current: dto.Current, Steps: make([]workflow.Step, len(dto.Steps))}
	for index, step := range dto.Steps {
		var digest [32]byte
		if step.Digest != "" {
			decoded, err := hex.DecodeString(step.Digest)
			if err != nil || len(decoded) != len(digest) {
				return workflow.State{}, ErrUnsafe
			}
			copy(digest[:], decoded)
		}
		state.Steps[index] = workflow.Step{Gate: step.Gate, Status: step.Status, Reference: step.Reference, Digest: digest}
	}
	if !validWorkflowState(state) {
		return workflow.State{}, ErrUnsafe
	}
	return state, nil
}

func validWorkflowState(state workflow.State) bool {
	if !validWorkflowAddress(state.ProjectID, state.RepositoryKey, state.WorkItem) || !filepath.IsAbs(state.RepositoryPath) || len(state.Steps) != len(workflow.Gates) || state.Current < 0 || state.Current > len(state.Steps) {
		return false
	}
	if state.Status != "active" && state.Status != "interrupted" && state.Status != "completed" {
		return false
	}
	for index, step := range state.Steps {
		if step.Gate != workflow.Gates[index] || step.Status != "pending" && step.Status != "passed" && step.Status != "failed" {
			return false
		}
		if step.Gate == "completion" {
			if step.Reference != "" || step.Digest != ([32]byte{}) || step.Status == "failed" {
				return false
			}
		} else if step.Status == "pending" {
			if (step.Reference == "") != (step.Digest == ([32]byte{})) {
				return false
			}
		} else if step.Reference == "" || step.Digest == ([32]byte{}) {
			return false
		}
		if index < state.Current && step.Status != "passed" {
			return false
		}
		if index > state.Current && step.Status != "pending" {
			return false
		}
	}
	if state.Status == "completed" {
		return state.Current == len(state.Steps)
	}
	if state.Current == len(state.Steps) {
		return false
	}
	if state.Status == "active" {
		return state.Steps[state.Current].Status == "pending"
	}
	return state.Steps[state.Current].Status == "failed"
}

func validWorkflowAddress(projectID, repositoryKey string, workItem int) bool {
	return len(project.ValidateIdentity(projectID, "workflow")) == 0 && project.ValidSlug(repositoryKey) && workItem > 0
}

func workflowName(repositoryKey string, workItem int) string {
	return fmt.Sprintf("%s-%d.json", repositoryKey, workItem)
}
