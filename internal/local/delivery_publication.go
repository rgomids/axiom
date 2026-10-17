package local

import (
	"bytes"
	"context"
	"crypto/sha256"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func publicationName(ref string) string {
	return hexDigest([]byte("delivery-publication:"+ref)) + ".json"
}

func (s WorkflowStore) LoadDeliveryPublication(ctx context.Context, projectID, ref string) (workflow.DeliveryPublication, error) {
	if !validUUID(projectID) || !workflowdefinition.ValidDigest(ref) {
		return workflow.DeliveryPublication{}, ErrUnsafe
	}
	if err := ctx.Err(); err != nil {
		return workflow.DeliveryPublication{}, err
	}
	root, executions, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return workflow.DeliveryPublication{}, workflowStoreError(err)
	}
	defer root.Close()
	defer executions.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, executions, version, project)
	if err != nil {
		return workflow.DeliveryPublication{}, err
	}
	defer closeFiles(locks)
	wire, err := readExecutionFile(project, publicationName(ref))
	if err != nil {
		return workflow.DeliveryPublication{}, workflowStoreError(err)
	}
	r, err := workflow.DecodeDeliveryPublication(wire)
	if err != nil || r.Preview.Digest() != ref || r.Preview.Packet.ProjectID != projectID {
		return r, ErrUnsafe
	}
	return r, nil
}

func (s WorkflowStore) SaveDeliveryPublication(ctx context.Context, state workflow.State, r workflow.DeliveryPublication, prior *workflow.DeliveryPublication) error {
	wire, err := workflow.EncodeDeliveryPublication(r)
	if err != nil {
		return err
	}
	if r.Preview.Packet.ExecutionID != state.ExecutionID || r.Preview.Packet.ProjectID != state.ProjectID || r.Preview.Packet.LedgerDigest != hexDigestRevision(state.StorageRevision) {
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
	var expected []byte
	if prior != nil {
		if prior.Preview.Digest() != r.Preview.Digest() {
			return ErrUnsafe
		}
		expected, err = workflow.EncodeDeliveryPublication(*prior)
		if err != nil {
			return err
		}
	}
	hooks := s.hooks
	hooks.contentLimit = workflow.MaxDeliveryBytes
	name := publicationName(r.Preview.Digest())
	if err = publishFile(ctx, project, name, expected, wire, prior == nil, hooks); err != nil {
		return err
	}
	confirmed, err := readPrivateFileBounded(project, name, workflow.MaxDeliveryBytes)
	if err != nil || !bytes.Equal(confirmed, wire) {
		return ErrRecoveryRequired
	}
	return nil
}

func classifyDeliveryPublication(projectID, file string, wire []byte) InventoryKind {
	r, err := workflow.DecodeDeliveryPublication(wire)
	if err != nil {
		return ""
	}
	if r.Preview.Packet.ProjectID != projectID || publicationName(r.Preview.Digest()) != file {
		return InventoryMalformed
	}
	return InventoryDeliveryPublication
}
