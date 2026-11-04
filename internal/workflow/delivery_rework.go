package workflow

import (
	"encoding/json"
	"reflect"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/strictjson"
)

type ReworkLink struct {
	ExecutionID    string `json:"executionId"`
	DecisionDigest string `json:"decisionDigest"`
	PreviewDigest  string `json:"previewDigest"`
}
type ReworkInput struct {
	NewExecutionID  string                            `json:"newExecutionId"`
	RevisionChoice  string                            `json:"revisionChoice"`
	RuntimeID       string                            `json:"runtimeId"`
	RuntimePreview  string                            `json:"runtimePreview"`
	ResolvedProfile *runtimeprofile.ResolvedProfile   `json:"resolvedProfile,omitempty"`
	FrozenProfile   *runtimeprofile.ResolvedProfileV4 `json:"frozenProfile,omitempty"`
	Scope           string                            `json:"scope"`
	Plan            Reference                         `json:"plan"`
	Inputs          []Reference                       `json:"inputs"`
	Authority       Reference                         `json:"authority"`
}
type ReworkPreview struct {
	FormatVersion  int            `json:"formatVersion"`
	Kind           string         `json:"kind"`
	OriginalPacket DeliveryPacket `json:"originalPacket"`
	DecisionDigest string         `json:"decisionDigest"`
	Input          ReworkInput    `json:"input"`
	NewState       State          `json:"newState"`
}

func (p ReworkPreview) Digest() string { return digest(p) }

// Legacy preview codec remains read-only for historical inventory and retention.
// New rework uses Execution.ReworkOf and the shared Engine/ExecutionStore.
func EncodeReworkPreview(p ReworkPreview) ([]byte, error) {
	n, o := p.NewState, p.OriginalPacket
	if _, err := EncodeDeliveryPacket(o); err != nil || p.FormatVersion != 1 || p.Kind != "delivery_rework" || !validDigest(p.DecisionDigest) || !ValidState(n) || n.Binding == nil || n.RuntimeID != p.Input.RuntimeID || n.Binding.RuntimePreview != p.Input.RuntimePreview || !reflect.DeepEqual(n.Binding.ResolvedProfile, p.Input.ResolvedProfile) || !reflect.DeepEqual(n.Binding.FrozenProfile, p.Input.FrozenProfile) || n.ExecutionID != p.Input.NewExecutionID || n.ExecutionID == o.ExecutionID || n.ProjectID != o.ProjectID || n.RepositoryKey != o.RepositoryKey || n.WorkItem != o.WorkItem || n.Status != ExecutionActive || n.Revision != 1 || n.StageOrdinal != 0 || len(n.Transitions) != 0 || len(n.Projections) != 0 || n.DeliveryDecision != nil || n.ReworkOf != nil || !validText(p.Input.Scope) || !validReference(p.Input.Plan) || !validReference(p.Input.Authority) || len(p.Input.Inputs) > 16 || (p.Input.RevisionChoice != "retained" && p.Input.RevisionChoice != "selected") {
		return nil, ErrDelivery
	}
	if p.Input.RevisionChoice == "retained" && n.Binding.Definition != o.Definition {
		return nil, ErrDelivery
	}
	wire, err := json.Marshal(p)
	if err != nil || len(wire)+1 > MaxDeliveryBytes {
		return nil, ErrDelivery
	}
	return append(wire, '\n'), nil
}
func DecodeReworkPreview(wire []byte) (ReworkPreview, error) {
	var p ReworkPreview
	if strictjson.Decode(wire, &p, MaxDeliveryBytes) != nil {
		return p, ErrDelivery
	}
	_, err := EncodeReworkPreview(p)
	return p, err
}
