package local

import "github.com/rgomids/axiom/internal/executiongraph"

// classifyStageOutcome recognizes inert legacy bytes for cutover inventory only.
func classifyStageOutcome(projectID, file string, wire []byte) InventoryKind {
	o, err := executiongraph.DecodeStageOutcome(wire)
	if err != nil {
		return ""
	}
	if o.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryStageOutcome
}
