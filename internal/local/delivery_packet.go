package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func (s WorkflowStore) RecordDeliveryPacket(ctx context.Context, state workflow.State, packet workflow.DeliveryPacket) (string, error) {
	wire, err := workflow.EncodeDeliveryPacket(packet)
	if err != nil {
		return "", err
	}
	binding, _ := json.Marshal(state.Binding)
	if state.Binding == nil || packet.BindingDigest != hexDigest(binding) || packet.Definition != state.Binding.Definition {
		return "", ErrUnsafe
	}
	if packet.ExecutionID != state.ExecutionID || packet.ProjectID != state.ProjectID || packet.RepositoryKey != state.RepositoryKey || packet.WorkItem != state.WorkItem || packet.LedgerDigest != hexDigestRevision(state.StorageRevision) || packet.ExecutionRevision != state.Revision {
		return "", ErrUnsafe
	}
	root, executions, version, project, err := s.openProject(state.ProjectID, false)
	if err != nil {
		return "", err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, executions, version, project)
	if err != nil {
		return "", err
	}
	defer closeFiles(locks)
	canonical, err := readExecutionFile(project, executionName(state.RepositoryKey, state.WorkItem))
	if err != nil {
		return "", err
	}
	if sha256.Sum256(canonical) != state.StorageRevision {
		return "", ErrConflict
	}
	ref := hexDigest(wire)
	if prior, e := readPrivateFileBounded(project, ref+".json", workflow.MaxDeliveryBytes); e == nil {
		if !bytes.Equal(prior, wire) {
			return "", ErrUnsafe
		}
		return ref, nil
	} else if !errors.Is(e, os.ErrNotExist) && !errors.Is(e, ErrNotFound) {
		return "", e
	}
	hooks := s.hooks
	hooks.contentLimit = workflow.MaxDeliveryBytes
	if err = publishFile(ctx, project, ref+".json", nil, wire, true, hooks); err != nil {
		return "", err
	}
	confirmed, err := readPrivateFileBounded(project, ref+".json", workflow.MaxDeliveryBytes)
	if err != nil || !bytes.Equal(confirmed, wire) {
		return "", &PublicationError{Stage: FaultF8, Committed: true, Err: ErrRecoveryRequired}
	}
	return ref, nil
}

func (s WorkflowStore) LoadDeliveryPacket(ctx context.Context, projectID, ref string) (workflow.DeliveryPacket, error) {
	if !validUUID(projectID) || !workflowdefinition.ValidDigest(ref) {
		return workflow.DeliveryPacket{}, ErrUnsafe
	}
	if err := ctx.Err(); err != nil {
		return workflow.DeliveryPacket{}, err
	}
	root, executions, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return workflow.DeliveryPacket{}, err
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, executions, version, project)
	if err != nil {
		return workflow.DeliveryPacket{}, err
	}
	defer closeFiles(locks)
	wire, err := readExecutionFile(project, ref+".json")
	if err != nil {
		return workflow.DeliveryPacket{}, err
	}
	p, err := workflow.DecodeDeliveryPacket(wire)
	if err != nil || p.ProjectID != projectID || hexDigest(wire) != ref {
		return workflow.DeliveryPacket{}, ErrUnsafe
	}
	return p, nil
}

func classifyDeliveryPacket(projectID, file string, wire []byte) InventoryKind {
	p, err := workflow.DecodeDeliveryPacket(wire)
	if err != nil {
		return ""
	}
	if p.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryDeliveryPacket
}
