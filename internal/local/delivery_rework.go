package local

import (
	"encoding/json"
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
