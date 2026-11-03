package local

import (
	"context"

	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func (s WorkflowStore) LoadDeliveryPacket(ctx context.Context, projectID, ref string) (workflow.DeliveryPacket, error) {
	if !validUUID(projectID) || !workflowdefinition.ValidDigest(ref) {
		return workflow.DeliveryPacket{}, ErrUnsafe
	}
	if err := ctx.Err(); err != nil {
		return workflow.DeliveryPacket{}, err
	}
	root, executions, version, project, err := s.openProject(projectID)
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
