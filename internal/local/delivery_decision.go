package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/workflow"
)

func decisionName(id string) string { return hexDigest([]byte("delivery-decision:"+id)) + ".json" }
func readDeliveryDecision(project *os.Root, projectID, executionID string) (workflow.DeliveryDecision, error) {
	wire, err := readExecutionFile(project, decisionName(executionID))
	if err != nil {
		return workflow.DeliveryDecision{}, err
	}
	d, err := workflow.DecodeDeliveryDecision(wire)
	if err != nil || d.Packet.ProjectID != projectID || d.Packet.ExecutionID != executionID {
		return d, ErrUnsafe
	}
	return d, nil
}
func (s WorkflowStore) LoadDeliveryDecision(ctx context.Context, projectID, executionID string) (workflow.DeliveryDecision, error) {
	if !validUUID(projectID) || !validUUID(executionID) {
		return workflow.DeliveryDecision{}, ErrUnsafe
	}
	if err := ctx.Err(); err != nil {
		return workflow.DeliveryDecision{}, err
	}
	root, executions, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return workflow.DeliveryDecision{}, workflowStoreError(err)
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, executions, version, project)
	if err != nil {
		return workflow.DeliveryDecision{}, err
	}
	defer closeFiles(locks)
	d, err := readDeliveryDecision(project, projectID, executionID)
	return d, workflowStoreError(err)
}
func (s WorkflowStore) RecordDeliveryDecision(ctx context.Context, state workflow.State, d workflow.DeliveryDecision) error {
	wire, err := workflow.EncodeDeliveryDecision(d)
	if err != nil {
		return err
	}
	if d.Packet.ExecutionID != state.ExecutionID || d.Packet.ProjectID != state.ProjectID || d.Packet.LedgerDigest != hexDigestRevision(state.StorageRevision) {
		return ErrUnsafe
	}
	root, executions, version, project, err := s.openProject(state.ProjectID, false)
	if err != nil {
		return err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, executions, version, project)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	canonical, err := readExecutionFile(project, executionName(state.RepositoryKey, state.WorkItem))
	if err != nil {
		return err
	}
	if sha256.Sum256(canonical) != state.StorageRevision {
		return ErrConflict
	}
	if d.Publication != nil {
		observed, err := readExecutionFile(project, publicationName(d.Publication.Preview.Digest()))
		if err != nil {
			return err
		}
		expected, _ := workflow.EncodeDeliveryPublication(*d.Publication)
		if !bytes.Equal(observed, expected) {
			return ErrConflict
		}
	}
	name := decisionName(state.ExecutionID)
	if prior, err := readExecutionFile(project, name); err == nil {
		if bytes.Equal(prior, wire) {
			return nil
		}
		return ErrConflict
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, ErrNotFound) {
		return err
	}
	packet, err := readExecutionFile(project, d.PacketDigest+".json")
	if err != nil {
		return err
	}
	if hexDigest(packet) != d.PacketDigest {
		return ErrUnsafe
	}
	hooks := s.hooks
	hooks.contentLimit = workflow.MaxDeliveryBytes
	if err = publishFile(ctx, project, name, nil, wire, true, hooks); err != nil {
		return err
	}
	confirmed, err := readExecutionFile(project, name)
	if err != nil || !bytes.Equal(wire, confirmed) {
		return ErrRecoveryRequired
	}
	return nil
}
func attachDeliveryDecision(project *os.Root, state workflow.State) (workflow.State, error) {
	if state.ReworkOf != nil {
		wire, err := readExecutionFile(project, reworkName(state.ReworkOf.ExecutionID))
		if err != nil {
			return state, err
		}
		p, err := workflow.DecodeReworkPreview(wire)
		if err != nil || p.Digest() != state.ReworkOf.PreviewDigest || p.DecisionDigest != state.ReworkOf.DecisionDigest || !sameExecutionBinding(p.NewState, state) {
			return state, ErrUnsafe
		}
		original, err := readExecutionFile(project, retainedExecutionName(state.ReworkOf.ExecutionID))
		if err != nil || hexDigest(original) != p.OriginalPacket.LedgerDigest {
			return state, ErrUnsafe
		}
		decision, err := readDeliveryDecision(project, state.ProjectID, state.ReworkOf.ExecutionID)
		if err != nil || decision.Digest() != p.DecisionDigest || decision.Outcome != "reject" {
			return state, ErrUnsafe
		}
	}

	d, err := readDeliveryDecision(project, state.ProjectID, state.ExecutionID)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, ErrNotFound) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	state.DeliveryDecision = &d
	if disposition, _ := workflow.DeliveryDisposition(state); disposition == "recovery_required" {
		return state, ErrUnsafe
	}
	return state, nil
}
func classifyDeliveryDecision(projectID, file string, wire []byte) InventoryKind {
	d, err := workflow.DecodeDeliveryDecision(wire)
	if err != nil {
		return ""
	}
	if d.Packet.ProjectID != projectID || decisionName(d.Packet.ExecutionID) != file {
		return InventoryMalformed
	}
	return InventoryDeliveryDecision
}
