package compatibility

import (
	"github.com/rgomids/axiom/internal/local"
)

// historicalRetirementSet resolves policy membership from validated persisted
// identities. A POC reference has no provider/resource: exactly one full link
// must match its Project/repository/number, otherwise retirement is ambiguous.
// The same rule runs on active bytes before preservation and archived bytes on
// resume, so confirmed retirement never erases the facts needed for recovery.
func historicalRetirementSet(objects []Object, read func(Object) ([]byte, error)) ([]Object, error) {
	links := map[local.POCWorkItemReference][]Object{}
	workflows := map[Object]local.POCWorkItemReference{}
	for _, object := range objects {
		if object.Kind != local.InventoryWorkItem && object.Kind != local.InventoryPOCWorkflow {
			continue
		}
		if object.Category != CategoryState {
			return nil, ErrPreservationSource
		}
		wire, err := read(object)
		if err != nil {
			return nil, err
		}
		if object.Kind == local.InventoryWorkItem {
			link, err := local.DecodeInventoryWorkItem(object.Relative, wire)
			if err != nil {
				return nil, ErrPreservationSource
			}
			identity := local.POCWorkItemReference{ProjectID: link.ProjectID, RepositoryKey: link.RepositoryKey, ExternalID: link.ExternalID}
			links[identity] = append(links[identity], object)
			continue
		}
		reference, err := local.DecodePOCWorkItemReference(object.Relative, wire)
		if err != nil {
			return nil, ErrPreservationSource
		}
		workflows[object] = reference
	}
	if len(workflows) == 0 {
		return nil, ErrPreservationSource
	}
	selected := map[Object]bool{}
	for workflow, reference := range workflows {
		matches := links[reference]
		if len(matches) != 1 {
			return nil, ErrPreservationSource
		}
		selected[workflow], selected[matches[0]] = true, true
	}
	retired := []Object{}
	for _, object := range objects {
		if selected[object] {
			retired = append(retired, object)
		}
	}
	return retired, nil
}

func archiveObjectReader(archive string) func(Object) ([]byte, error) {
	return func(object Object) ([]byte, error) {
		wire, err := local.ReadOwnedFile(archive, preservationObjects+"/"+object.Digest, maxTransferFile)
		if err != nil || digestHex(wire) != object.Digest || int64(len(wire)) != object.Bytes {
			return nil, ErrPreservationConflict
		}
		return wire, nil
	}
}
