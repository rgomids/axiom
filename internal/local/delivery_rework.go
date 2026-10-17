package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type activeExecution struct {
	FormatVersion int                 `json:"formatVersion"`
	Kind          string              `json:"kind"`
	ReworkOf      workflow.ReworkLink `json:"reworkOf"`
	State         json.RawMessage     `json:"state"`
}

func retainedExecutionName(id string) string {
	return hexDigest([]byte("retained-execution:"+id)) + ".json"
}
func reworkName(id string) string { return hexDigest([]byte("delivery-rework:"+id)) + ".json" }
func decodeExecution(wire []byte) (workflow.State, error) {
	var header struct {
		FormatVersion int `json:"formatVersion"`
	}
	if json.Unmarshal(wire, &header) != nil {
		return workflow.State{}, ErrUnsafe
	}
	if header.FormatVersion != 4 {
		return decodeExecutionSnapshot(wire)
	}
	var envelope activeExecution
	if strictjson.Decode(wire, &envelope, MaxExecutionBytes) != nil || envelope.Kind != "execution_link" || !validUUID(envelope.ReworkOf.ExecutionID) || !workflowdefinition.ValidDigest(envelope.ReworkOf.DecisionDigest) || !workflowdefinition.ValidDigest(envelope.ReworkOf.PreviewDigest) {
		return workflow.State{}, ErrUnsafe
	}
	state, err := decodeExecutionSnapshot(envelope.State)
	if err != nil || state.ExecutionID == envelope.ReworkOf.ExecutionID {
		return state, ErrUnsafe
	}
	state.ReworkOf = &envelope.ReworkOf
	return state, nil
}
func encodeExecution(state workflow.State) ([]byte, error) {
	wire, err := encodeExecutionSnapshot(state)
	if err != nil || state.ReworkOf == nil {
		return wire, err
	}
	envelope := activeExecution{FormatVersion: 4, Kind: "execution_link", ReworkOf: *state.ReworkOf, State: wire}
	wire, err = json.Marshal(envelope)
	if err != nil || len(wire)+1 > MaxExecutionBytes {
		return nil, ErrUnsafe
	}
	wire = append(wire, '\n')
	if _, err = decodeExecution(wire); err != nil {
		return nil, err
	}
	return wire, nil
}
func (s WorkflowStore) ApplyDeliveryRework(ctx context.Context, original workflow.State, p workflow.ReworkPreview) (workflow.State, error) {
	proposal, err := workflow.EncodeReworkPreview(p)
	if err != nil {
		return workflow.State{}, err
	}
	if p.OriginalPacket.ExecutionID != original.ExecutionID || p.OriginalPacket.ProjectID != original.ProjectID || p.OriginalPacket.RepositoryKey != original.RepositoryKey || p.OriginalPacket.WorkItem != original.WorkItem || p.OriginalPacket.ExecutionRevision != original.Revision {
		return workflow.State{}, ErrUnsafe
	}
	root, executions, version, project, err := s.openProject(original.ProjectID, false)
	if err != nil {
		return workflow.State{}, err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, executions, version, project)
	if err != nil {
		return workflow.State{}, err
	}
	defer closeFiles(locks)
	name := executionName(original.RepositoryKey, original.WorkItem)
	canonical, err := readExecutionFile(project, name)
	if err != nil {
		return workflow.State{}, err
	}
	current, err := decodeExecution(canonical)
	if err != nil {
		return current, err
	}
	if current.ReworkOf != nil && current.ReworkOf.PreviewDigest == p.Digest() && current.ExecutionID == p.NewState.ExecutionID {
		current.StorageRevision = sha256.Sum256(canonical)
		return attachDeliveryDecision(project, current)
	}
	if sha256.Sum256(canonical) != original.StorageRevision || current.ExecutionID != original.ExecutionID || p.OriginalPacket.LedgerDigest != hexDigestRevision(original.StorageRevision) {
		return workflow.State{}, ErrConflict
	}
	decision, err := readDeliveryDecision(project, original.ProjectID, original.ExecutionID)
	if err != nil || decision.Digest() != p.DecisionDigest || decision.Outcome != "reject" {
		return workflow.State{}, ErrUnsafe
	}
	if err = s.retainDeliveryFile(ctx, project, retainedExecutionName(original.ExecutionID), canonical); err != nil {
		return workflow.State{}, err
	}
	if err = s.retainDeliveryFile(ctx, project, reworkName(original.ExecutionID), proposal); err != nil {
		return workflow.State{}, err
	}
	next := p.NewState
	next.ReworkOf = &workflow.ReworkLink{ExecutionID: original.ExecutionID, DecisionDigest: p.DecisionDigest, PreviewDigest: p.Digest()}
	wire, err := encodeExecution(next)
	if err != nil {
		return workflow.State{}, err
	}
	hooks := s.hooks
	hooks.contentLimit = MaxExecutionBytes
	if err = publishFile(ctx, project, name, canonical, wire, false, hooks); err != nil {
		return workflow.State{}, err
	}
	confirmed, err := readExecutionFile(project, name)
	if err != nil || !bytes.Equal(wire, confirmed) {
		return workflow.State{}, ErrRecoveryRequired
	}
	next.StorageRevision = sha256.Sum256(confirmed)
	return next, nil
}
func (s WorkflowStore) retainDeliveryFile(ctx context.Context, project *os.Root, name string, wire []byte) error {
	prior, err := readExecutionFile(project, name)
	if err == nil {
		if bytes.Equal(prior, wire) {
			return nil
		}
		return ErrConflict
	}
	if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, ErrNotFound) {
		return err
	}
	hooks := s.hooks
	hooks.contentLimit = MaxExecutionBytes
	if err = publishFile(ctx, project, name, nil, wire, true, hooks); err != nil {
		return err
	}
	confirmed, err := readExecutionFile(project, name)
	if err != nil || !bytes.Equal(wire, confirmed) {
		return ErrRecoveryRequired
	}
	return nil
}
func (s WorkflowStore) LoadExecutionID(ctx context.Context, projectID, repository string, item workflow.WorkItem, id string) (workflow.State, error) {
	if !validUUID(id) || !validExecutionAddress(projectID, repository, item) {
		return workflow.State{}, workflow.ErrNotFound
	}
	if err := ctx.Err(); err != nil {
		return workflow.State{}, err
	}
	root, executions, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return workflow.State{}, workflowStoreError(err)
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, executions, version, project)
	if err != nil {
		return workflow.State{}, err
	}
	defer closeFiles(locks)
	wire, err := readExecutionFile(project, executionName(repository, item))
	if err != nil {
		return workflow.State{}, workflowStoreError(err)
	}
	current, err := decodeExecution(wire)
	if err != nil {
		return current, err
	}
	if current.ExecutionID != id {
		wire, err = readExecutionFile(project, retainedExecutionName(id))
		if err != nil {
			return workflow.State{}, workflowStoreError(err)
		}
		current, err = decodeExecution(wire)
		if err != nil {
			return current, err
		}
	}
	if current.ProjectID != projectID || current.RepositoryKey != repository || current.WorkItem != item || current.ExecutionID != id {
		return current, ErrUnsafe
	}
	current.StorageRevision = sha256.Sum256(wire)
	return attachDeliveryDecision(project, current)
}
func classifyDeliveryRework(projectID, file string, wire []byte) InventoryKind {
	p, err := workflow.DecodeReworkPreview(wire)
	if err != nil {
		return ""
	}
	if p.OriginalPacket.ProjectID != projectID || reworkName(p.OriginalPacket.ExecutionID) != file {
		return InventoryMalformed
	}
	return InventoryDeliveryRework
}
func classifyRetainedExecution(projectID, file string, wire []byte) InventoryKind {
	s, err := decodeExecution(wire)
	if err != nil {
		return ""
	}
	if s.ProjectID == projectID && retainedExecutionName(s.ExecutionID) == file {
		return InventoryRetainedExecution
	}
	return ""
}

func (s WorkflowStore) LoadDeliveryRework(ctx context.Context, projectID, originalID string) (workflow.ReworkPreview, error) {
	if !validUUID(projectID) || !validUUID(originalID) {
		return workflow.ReworkPreview{}, ErrUnsafe
	}
	if err := ctx.Err(); err != nil {
		return workflow.ReworkPreview{}, err
	}
	root, executions, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return workflow.ReworkPreview{}, workflowStoreError(err)
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, executions, version, project)
	if err != nil {
		return workflow.ReworkPreview{}, err
	}
	defer closeFiles(locks)
	wire, err := readExecutionFile(project, reworkName(originalID))
	if err != nil {
		return workflow.ReworkPreview{}, workflowStoreError(err)
	}
	p, err := workflow.DecodeReworkPreview(wire)
	if err != nil || p.OriginalPacket.ProjectID != projectID || p.OriginalPacket.ExecutionID != originalID {
		return p, ErrUnsafe
	}
	return p, nil
}
