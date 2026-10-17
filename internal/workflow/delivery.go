package workflow

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"

	"github.com/rgomids/axiom/internal/detailartifact"
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

type DeliveryEvidence interface {
	Reference(context.Context, State, Reference) error
	Artifact(context.Context, State, workflowdefinition.Stage, workflowdefinition.Output, Reference) (detailartifact.Artifact, error)
	Stage(context.Context, State, int) (DeliveryStage, error)
}

func ComposeDeliveryPacket(ctx context.Context, state State, evidence DeliveryEvidence) (DeliveryPacket, error) {
	doc, ok := bindingDocument(state)
	if !ok || !ValidState(state) || evidence == nil || state.StorageRevision == ([32]byte{}) {
		return DeliveryPacket{}, ErrDelivery
	}
	p := DeliveryPacket{FormatVersion: 1, Kind: "delivery_packet", ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, WorkItem: state.WorkItem, ExecutionID: state.ExecutionID, ExecutionRevision: state.Revision, LedgerDigest: fmtDigest(state.StorageRevision), BindingDigest: digest(state.Binding), Definition: state.Binding.Definition, Stages: []DeliveryStage{}, Blockers: []string{}, Provenance: state.Provenance}
	block := func(stage, reason string) { p.Blockers = append(p.Blockers, stage+":"+reason) }
	_, conditions, valid := configuredFacts(state)
	if !valid {
		return DeliveryPacket{}, ErrDelivery
	}
	if conditions.Blocked || conditions.NeedsDecision || conditions.NeedsApproval {
		block("execution", "human_gate_pending")
	}
	for ordinal, stage := range doc.Definition.Stages {
		item := DeliveryStage{Ordinal: ordinal, StageID: stage.ID, Artifacts: []Reference{}, Reviews: []DeliveryReview{}}
		for _, event := range state.Transitions {
			if event.Ledger != nil && event.Ledger.Ordinal == ordinal {
				ledger := *event.Ledger
				item.Ledger = &ledger
			}
		}
		observed, err := evidence.Stage(ctx, state, ordinal)
		if err != nil {
			return DeliveryPacket{}, err
		}
		item.Children = observed.Children
		item.SingleAttempt = observed.SingleAttempt
		item.Blocker = observed.Blocker
		if observed.Blocker != "" {
			block(stage.ID, observed.Blocker)
		}
		if observed.Outcome != nil && observed.Outcome.Rollup.Status != "success" {
			block(stage.ID, "integration_failed")
		}
		item.ResultTree, item.DiffDigest = observed.ResultTree, observed.DiffDigest
		item.OutcomeRef, item.Outcome, item.GraphDigest, item.ReceiptRef, item.SingleDigest = observed.OutcomeRef, observed.Outcome, observed.GraphDigest, observed.ReceiptRef, observed.SingleDigest
		if len(stage.Agents) == 1 {
			if item.SingleAttempt == nil || !validDigest(item.SingleDigest) {
				block(stage.ID, "single_stage_execution_missing")
			} else if !validOpaqueID(item.SingleAttempt.AttemptID) || item.SingleAttempt.Status != executiongraph.AttemptSucceeded || item.SingleAttempt.ClassificationReason != executiongraph.ResultRecorded || !validDigest(item.SingleAttempt.QualifyingRecordRef) {
				block(stage.ID, "single_stage_incomplete")
			}
		}
		if item.Ledger == nil {
			block(stage.ID, "checkpoint_missing")
			p.Stages = append(p.Stages, item)
			continue
		}
		if !reflect.DeepEqual(item.Ledger.Validators, expectedValidators(stage, item.Ledger.Result)) {
			return DeliveryPacket{}, ErrDelivery
		}
		if len(stage.Agents) > 1 && item.Outcome == nil {
			block(stage.ID, "integration_missing")
		}
		for _, output := range stage.Outputs {
			ref, exists := item.Ledger.Result.Outputs[output.ID]
			if !exists {
				if output.Required {
					block(stage.ID, "output_missing:"+output.ID)
				}
				continue
			}
			artifact, err := evidence.Artifact(ctx, state, stage, output, ref)
			if err != nil {
				return DeliveryPacket{}, err
			}
			if !artifact.Valid() || artifact.ExecutionID != state.ExecutionID || fmtDigest(artifact.Digest) != ref.Digest || artifact.Category != output.Kind || artifact.Outcome != "success" {
				return DeliveryPacket{}, ErrDelivery
			}
			item.Artifacts = append(item.Artifacts, ref)
			if stage.Phase == "review" && output.Required {
				var review DeliveryReview
				if strictjson.Decode(artifact.Markdown, &review, MaxDeliveryBytes) != nil || !validDeliveryReview(review, state, stage, item.Ledger.Result) {
					block(stage.ID, "review_unverified:"+output.ID)
					continue
				}
				for _, finding := range review.Findings {
					if evidence.Reference(ctx, state, finding.Evidence) != nil {
						return DeliveryPacket{}, ErrDelivery
					}
				}
				for _, waiver := range review.Waivers {
					if evidence.Reference(ctx, state, waiver.Authority) != nil {
						return DeliveryPacket{}, ErrDelivery
					}
				}
				item.Reviews = append(item.Reviews, review)
				for _, finding := range review.Findings {
					if !finding.Resolved && (finding.Severity == "blocker" || finding.Severity == "major") && !deliveryWaived(state, *item.Ledger, review, finding) {
						block(stage.ID, "finding:"+finding.ID)
					}
				}
				if review.Outcome != "pass" {
					block(stage.ID, "review_failed")
				}
			}
		}
		p.Stages = append(p.Stages, item)
	}
	if state.Status != ExecutionCompleted {
		block("execution", "not_completed")
	}
	sort.Strings(p.Blockers)
	p.Ready = len(p.Blockers) == 0
	p.Next = "preview delivery publication"
	if !p.Ready {
		p.Next = "resolve packet blockers through the retained workflow gates; run or reconcile required stage attempts with matching authority and coordination Evidence"
	}
	if _, err := EncodeDeliveryPacket(p); err != nil {
		return DeliveryPacket{}, err
	}
	return p, nil
}

func fmtDigest(v [32]byte) string { return hex.EncodeToString(v[:]) }

func validDeliveryReview(r DeliveryReview, s State, stage workflowdefinition.Stage, result StageResult) bool {
	if r.FormatVersion != 1 || r.ExecutionID != s.ExecutionID || r.DefinitionDigest != s.Binding.Definition.Digest || r.StageID != stage.ID || r.InputsDigest != digest(result.Inputs) || (r.Outcome != "pass" && r.Outcome != "fail") || r.Findings == nil || r.Waivers == nil || len(r.Findings) > 128 || len(r.Waivers) > 128 {
		return false
	}
	seen := map[string]bool{}
	for _, f := range r.Findings {
		if !validText(f.ID) || seen[f.ID] || !validReference(f.Evidence) {
			return false
		}
		seen[f.ID] = true
		switch f.Severity {
		case "blocker", "major", "minor", "note":
		default:
			return false
		}
	}
	for _, w := range r.Waivers {
		if !validText(w.Actor) || !validText(w.Reason) || !validReference(w.Authority) || !validDigest(w.FindingDigest) || w.ActorKind != "human-operator" || w.ExecutionID != r.ExecutionID || w.DefinitionDigest != r.DefinitionDigest || w.InputsDigest != r.InputsDigest {
			return false
		}
	}
	return true
}

func deliveryWaived(state State, ledger StageLedger, r DeliveryReview, f DeliveryFinding) bool {
	for _, w := range r.Waivers {
		if w.FindingDigest == digest(f) {
			for _, event := range state.Transitions {
				if event.From == Stage(ledger.StageID) && event.Fact != nil && event.Fact.Kind == FactStageReview && event.Fact.Actor == w.Actor && event.Fact.Reference == w.Authority && event.Fact.ResultDigest == digest(ledger.Result) {
					return true
				}
			}
		}
	}
	return false
}

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
