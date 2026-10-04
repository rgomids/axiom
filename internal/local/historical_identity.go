package local

import (
	"encoding/json"
	"path"
	"strconv"

	"github.com/rgomids/axiom/internal/workitem"
)

// POCWorkItemReference is the identity the frozen POC workflow actually
// persisted. Provider/resource were not stored; callers must prove a unique
// corresponding validated link rather than infer those missing fields.
type POCWorkItemReference struct {
	ProjectID, RepositoryKey, ExternalID string
}

// DecodePOCWorkItemReference validates the historical payload and its exact
// store location before exposing deterministic identity facts.
func DecodePOCWorkItemReference(relative string, wire []byte) (POCWorkItemReference, error) {
	if !validPOCWorkflow(wire) {
		return POCWorkItemReference{}, ErrUnsafe
	}
	var dto pocWorkflowDTO
	if err := json.Unmarshal(wire, &dto); err != nil {
		return POCWorkItemReference{}, err
	}
	number := strconv.Itoa(dto.WorkItem)
	if relative != path.Join("workflows", dto.ProjectID, legacyWorkItemName(dto.RepositoryKey, number)) {
		return POCWorkItemReference{}, ErrUnsafe
	}
	return POCWorkItemReference{dto.ProjectID, dto.RepositoryKey, number}, nil
}

// DecodeInventoryWorkItem validates a link using the canonical store decoder,
// bound to either of the two locations the store supports.
func DecodeInventoryWorkItem(relative string, wire []byte) (workitem.Link, error) {
	link, err := decodeWorkItem(wire)
	if err != nil {
		return workitem.Link{}, err
	}
	prefix := path.Join("work-items", link.ProjectID)
	if relative != path.Join(prefix, workItemName(link.RepositoryKey, link.Provider, link.Resource, link.ExternalID)) &&
		relative != path.Join(prefix, legacyWorkItemName(link.RepositoryKey, link.ExternalID)) {
		return workitem.Link{}, ErrUnsafe
	}
	return link, nil
}
