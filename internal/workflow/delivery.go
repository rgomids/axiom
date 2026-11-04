package workflow

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const MaxDeliveryBytes = 1 << 20

var ErrDelivery = errors.New("delivery_evidence_invalid")

type DeliveryFinding struct {
	ID       string    `json:"id"`
	Severity string    `json:"severity"`
	Evidence Reference `json:"evidence"`
	Resolved bool      `json:"resolved"`
}

type DeliveryWaiver struct {
	FindingDigest    string    `json:"findingDigest"`
	ExecutionID      string    `json:"executionId"`
	DefinitionDigest string    `json:"definitionDigest"`
	InputsDigest     string    `json:"inputsDigest"`
	ActorKind        string    `json:"actorKind"`
	Actor            string    `json:"actor"`
	Authority        Reference `json:"authority"`
	Reason           string    `json:"reason"`
}

type DeliveryReview struct {
	FormatVersion    int               `json:"formatVersion"`
	ExecutionID      string            `json:"executionId"`
	DefinitionDigest string            `json:"definitionDigest"`
	StageID          string            `json:"stageId"`
	InputsDigest     string            `json:"inputsDigest"`
	Outcome          string            `json:"outcome"`
	Findings         []DeliveryFinding `json:"findings"`
	Waivers          []DeliveryWaiver  `json:"waivers"`
}

type DeliveryChild struct {
	ChildID  string                            `json:"childId"`
	Attempts []executiongraph.SuccessorAttempt `json:"attempts"`
}

type DeliveryStage struct {
	Children      []DeliveryChild                  `json:"children,omitempty"`
	SingleAttempt *executiongraph.SuccessorAttempt `json:"singleAttempt,omitempty"`
	Blocker       string                           `json:"blocker,omitempty"`
	ResultTree    string                           `json:"resultTree,omitempty"`
	DiffDigest    string                           `json:"diffDigest,omitempty"`
	Ordinal       int                              `json:"ordinal"`
	StageID       string                           `json:"stageId"`
	Ledger        *StageLedger                     `json:"ledger,omitempty"`
	OutcomeRef    string                           `json:"outcomeRef,omitempty"`
	Outcome       *executiongraph.StageOutcome     `json:"outcome,omitempty"`
	GraphDigest   string                           `json:"graphDigest,omitempty"`
	ReceiptRef    string                           `json:"receiptRef,omitempty"`
	SingleDigest  string                           `json:"singleDigest,omitempty"`
	Artifacts     []Reference                      `json:"artifacts"`
	Reviews       []DeliveryReview                 `json:"reviews"`
}

type DeliveryPacket struct {
	FormatVersion     int                    `json:"formatVersion"`
	Kind              string                 `json:"kind"`
	ProjectID         string                 `json:"projectId"`
	RepositoryKey     string                 `json:"repositoryKey"`
	WorkItem          WorkItem               `json:"workItem"`
	ExecutionID       string                 `json:"executionId"`
	ExecutionRevision uint64                 `json:"executionRevision"`
	LedgerDigest      string                 `json:"ledgerDigest"`
	BindingDigest     string                 `json:"bindingDigest"`
	Definition        workflowdefinition.Ref `json:"definition"`
	Stages            []DeliveryStage        `json:"stages"`
	Blockers          []string               `json:"blockers"`
	Ready             bool                   `json:"ready"`
	Next              string                 `json:"next"`
	Provenance        Identity               `json:"provenance"`
}

func fmtDigest(v [32]byte) string { return hex.EncodeToString(v[:]) }

func EncodeDeliveryPacket(p DeliveryPacket) ([]byte, error) {
	if p.FormatVersion != 1 || p.Kind != "delivery_packet" || !validText(p.ProjectID) || !validText(p.RepositoryKey) || !validWorkItem(p.WorkItem) || !validOpaqueID(p.ExecutionID) || p.ExecutionRevision == 0 || !validDigest(p.LedgerDigest) || !validDigest(p.BindingDigest) || !p.Definition.Valid() || !validIdentity(p.Provenance) || len(p.Stages) > 64 || p.Stages == nil || p.Blockers == nil || len(p.Blockers) > 512 || p.Ready != (len(p.Blockers) == 0) || !validText(p.Next) {
		return nil, ErrDelivery
	}
	for i, s := range p.Stages {
		if s.Ordinal != i || !workflowdefinition.ValidKey(s.StageID) || len(s.Artifacts) > 64 || len(s.Reviews) > 64 {
			return nil, ErrDelivery
		}
	}
	for i, b := range p.Blockers {
		if !validText(b) || i > 0 && p.Blockers[i-1] > b {
			return nil, ErrDelivery
		}
	}
	wire, err := json.Marshal(p)
	if err != nil || len(wire)+1 > MaxDeliveryBytes {
		return nil, ErrDelivery
	}
	return append(wire, '\n'), nil
}

func DecodeDeliveryPacket(wire []byte) (DeliveryPacket, error) {
	var p DeliveryPacket
	if strictjson.Decode(wire, &p, MaxDeliveryBytes) != nil {
		return p, ErrDelivery
	}
	_, err := EncodeDeliveryPacket(p)
	return p, err
}
func DeliveryPacketDigest(p DeliveryPacket) (string, error) {
	wire, err := EncodeDeliveryPacket(p)
	if err != nil {
		return "", err
	}
	return digestBytes(wire), nil
}
