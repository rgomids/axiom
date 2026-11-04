package local

import (
	"context"

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
	root, executions, version, project, err := s.openProject(projectID)
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
