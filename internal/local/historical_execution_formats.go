package local

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/rgomids/axiom/internal/executiongraph"
)

// Historical format classifiers serve inert inventory and lease addressing only.
// No legacy execution store, writer, dispatcher or resume path remains here.
func graphProjectID(graph executiongraph.Graph) string {
	return graph.Children[0].Envelope.Scope.ProjectID
}

// graphStoredAt reports whether a decoded graph is the record GraphStore
// addresses at graphs/v1/<projectID>/<name>: its Project owns the directory
// and its parent identity names the file.

func graphStoredAt(graph executiongraph.Graph, projectID, name string) bool {
	return validGraphAddress(projectID, graph.Parent.ExecutionID) && graphProjectID(graph) == projectID && graphName(graph.Parent.ExecutionID) == name
}

func graphName(parentID string) string {
	digest := sha256.Sum256([]byte(parentID))
	return hex.EncodeToString(digest[:]) + ".json"
}

func validGraphAddress(projectID, parentID string) bool {
	return validGraphToken(projectID) && len(parentID) == 36
}

func validGraphToken(value string) bool {
	if value == "" || len(value) > 512 {
		return false
	}
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':') {
			return false
		}
	}
	return true
}

func singleStageName(scopeDigest string) string { return scopeDigest + ".json" }

func classifySingleStage(projectID, file string, wire []byte) InventoryKind {
	if a, err := executiongraph.DecodeSingleStageAuthority(wire); err == nil {
		if a.ProjectID != projectID || hexDigest(wire)+".json" != file {
			return InventoryMalformed
		}
		return InventorySingleStageAuthority
	} else if errors.Is(err, executiongraph.ErrUnsupportedSingleStageFormat) {
		return versionedFailureWithin(wire, executiongraph.SingleStageAuthorityVersion)
	}
	if kind := classifyDispatchInputs(projectID, file, wire); kind != "" {
		return kind
	}
	r, err := executiongraph.DecodeSingleStageRun(wire)
	if errors.Is(err, executiongraph.ErrUnsupportedSingleStageFormat) {
		return versionedFailureWithin(wire, executiongraph.SingleStageRunVersion)
	}
	if err != nil {
		return InventoryMalformed
	}
	if r.Scope.ProjectID != projectID || singleStageName(r.ScopeDigest) != file {
		return InventoryMalformed
	}
	return InventorySingleStageRun
}

func classifyDispatchInputs(projectID, file string, wire []byte) InventoryKind {
	d, err := executiongraph.DecodeDispatchInputs(wire)
	if err != nil {
		return ""
	}
	if d.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryDispatchInputs
}

func classifyRetryAuthority(projectID, file string, wire []byte) InventoryKind {
	a, err := executiongraph.DecodeRetryAuthority(wire)
	if err != nil {
		return ""
	}
	if a.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryRetryAuthority
}

func classifyIntegrationReceipt(projectID, file string, wire []byte) InventoryKind {
	a, err := executiongraph.DecodeIntegrationReceipt(wire)
	if err != nil {
		return ""
	}
	if a.ProjectID != projectID || hexDigest(wire)+".json" != file {
		return InventoryMalformed
	}
	return InventoryIntegrationAuthority
}

func hexDigestRevision(revision [32]byte) string { return fmt.Sprintf("%x", revision) }
