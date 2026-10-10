package workflowcompiler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const (
	StagePlanFormatVersion    = 1
	PlanDocumentFormatVersion = 1
	// MaxPlanDocumentBytes bounds the operator-supplied approved stage Plan.
	MaxPlanDocumentBytes = 256 << 10
)

var ErrInvalidPlanDocument = errors.New("invalid stage plan document")

// PlanDocument is the bounded approved work Plan for one stage: the existing
// graph ApprovedPlan plus the explicit parent authority ceiling it may use.
// Effect targets and Repository paths come only from this reviewed document.
type PlanDocument struct {
	FormatVersion    int                         `json:"formatVersion"`
	Plan             executiongraph.ApprovedPlan `json:"plan"`
	AuthorityCeiling []executiongraph.Effect     `json:"authorityCeiling"`
}

// DecodePlanDocument strictly decodes one Plan document and returns the
// digest of its canonical encoding, so formatting alone never changes it.
func DecodePlanDocument(wire []byte) (PlanDocument, string, error) {
	var document PlanDocument
	if workflowdefinition.StrictJSONBounded(wire, &document, MaxPlanDocumentBytes) != nil || document.FormatVersion != PlanDocumentFormatVersion {
		return PlanDocument{}, "", ErrInvalidPlanDocument
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return PlanDocument{}, "", ErrInvalidPlanDocument
	}
	return document, digestOf(canonical), nil
}

type ExecutionRef struct {
	ExecutionID string `json:"executionId"`
	Revision    uint64 `json:"revision"`
}

type PlanRef struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
	Digest   string `json:"digest"`
}

type GraphProposalRef struct {
	PlanRevision string `json:"planRevision"`
	Digest       string `json:"digest"`
}

// GateRef is read-only criteria. Planning reports whether the explicit human
// fact exists; it never records, satisfies or infers one.
type GateRef struct {
	Kind      string `json:"kind"`
	Timing    string `json:"timing"`
	Satisfied bool   `json:"satisfied"`
}

// Resolution is one agent's resolved Runtime/Profile selection. Effective
// effort is "default" (no override) or the exact capability-proven value.
type Resolution struct {
	AgentID               string                    `json:"agentId"`
	RuntimeID             string                    `json:"runtimeId"`
	ModelProfileID        string                    `json:"modelProfileId"`
	ConfigurationRevision uint64                    `json:"configurationRevision"`
	ObservationRevision   uint64                    `json:"observationRevision"`
	RequestedEffort       workflowdefinition.Effort `json:"requestedEffort"`
	EffectiveEffort       string                    `json:"effectiveEffort"`
	CapabilityEvidenceRef string                    `json:"capabilityEvidenceRef"`
}

type StagePlanRequest struct {
	Execution          ExecutionRef
	Gates              []GateRef
	PlanDocumentDigest string
	Compile            Request
}

// StagePlan is the canonical, inspectable, non-dispatching stage proposal.
// Compilation retains the complete compiler result; Digest binds both to the
// exact Execution revision and Plan document.
type StagePlan struct {
	FormatVersion      int                    `json:"formatVersion"`
	Execution          ExecutionRef           `json:"executionRef"`
	ProjectID          string                 `json:"projectId"`
	RepositoryKey      string                 `json:"repositoryKey"`
	Workflow           workflowdefinition.Ref `json:"workflowRef"`
	StageID            string                 `json:"stageId"`
	ExecutionKind      string                 `json:"executionKind"`
	Plan               PlanRef                `json:"planRef"`
	PlanDocumentDigest string                 `json:"planDocumentDigest"`
	StageInput         StageInputReference    `json:"stageInputRef"`
	GraphProposal      *GraphProposalRef      `json:"graphProposalRef,omitempty"`
	Resolutions        []Resolution           `json:"resolutions"`
	ValidatorRefs      []string               `json:"validatorRefs"`
	GateRefs           []GateRef              `json:"gateRefs"`
	Blockers           []string               `json:"blockers"`
	Compilation        Result                 `json:"compilation"`
	Digest             string                 `json:"digest"`
}

// PlanStage compiles the requested stage through the existing compiler and
// wraps it in the canonical StagePlan. It has no dispatch, persistence or
// effect port: a returned plan is inspection data, never authority.
func (c Compiler) PlanStage(ctx context.Context, request StagePlanRequest) (StagePlan, error) {
	if request.Execution.ExecutionID == "" || request.Execution.Revision == 0 || !workflowdefinition.ValidDigest(request.PlanDocumentDigest) {
		return StagePlan{}, ErrInvalidRequest
	}
	compiled, err := c.Compile(ctx, request.Compile)
	if err != nil {
		return StagePlan{}, err
	}
	stage, _ := validateRequest(request.Compile)
	plan := StagePlan{FormatVersion: StagePlanFormatVersion, Execution: request.Execution, ProjectID: request.Compile.ProjectID, RepositoryKey: request.Compile.RepositoryKey, Workflow: compiled.Workflow, StageID: compiled.StageID, ExecutionKind: compiled.ExecutionKind, Plan: PlanRef{ID: compiled.PlanRevision, Revision: compiled.PlanRevision, Digest: compiled.PlanDigest}, PlanDocumentDigest: request.PlanDocumentDigest, Resolutions: make([]Resolution, 0, len(compiled.Agents)), ValidatorRefs: []string{}, GateRefs: []GateRef{}, Blockers: []string{}, Compilation: compiled}
	if compiled.Proposal != nil {
		plan.GraphProposal = &GraphProposalRef{PlanRevision: compiled.Proposal.PlanRevision, Digest: compiled.Proposal.Digest}
	}
	inputRefs := make([]StageInputReference, 0, len(compiled.Agents))
	for _, agent := range compiled.Agents {
		preview := agent.Input.RuntimePreview
		effective := "default"
		if agent.Input.RequestedEffort.Mode == "explicit" {
			effective = agent.Input.RequestedEffort.Value
		}
		plan.Resolutions = append(plan.Resolutions, Resolution{AgentID: agent.AgentID, RuntimeID: agent.Choice.RuntimeID, ModelProfileID: agent.Choice.ModelProfileID, ConfigurationRevision: agent.Choice.ConfigurationRevision, ObservationRevision: agent.Choice.ObservationRevision, RequestedEffort: agent.Input.RequestedEffort, EffectiveEffort: effective, CapabilityEvidenceRef: "runtime-observation:" + agent.Choice.RuntimeID + ":" + preview.ObservationDigest})
		inputRefs = append(inputRefs, agent.InputReference)
	}
	sort.Slice(plan.Resolutions, func(i, j int) bool { return plan.Resolutions[i].AgentID < plan.Resolutions[j].AgentID })
	sort.Slice(inputRefs, func(i, j int) bool { return inputRefs[i].ID < inputRefs[j].ID })
	wire, err := json.Marshal(inputRefs)
	if err != nil {
		return StagePlan{}, ErrInvalidRequest
	}
	setDigest := digestOf(wire)
	plan.StageInput = StageInputReference{ID: "stage-input-set-" + setDigest, Digest: setDigest}
	for _, validator := range stage.Validators {
		plan.ValidatorRefs = append(plan.ValidatorRefs, validator.ID)
	}
	sort.Strings(plan.ValidatorRefs)
	for _, gate := range request.Gates {
		plan.GateRefs = append(plan.GateRefs, gate)
		if gate.Timing == "before" && !gate.Satisfied {
			plan.Blockers = append(plan.Blockers, "human_gate_pending:"+gate.Kind)
		}
	}
	sort.Strings(plan.Blockers)
	plan.Digest = ""
	wire, err = json.Marshal(plan)
	if err != nil || len(wire) > executiongraph.MaxGraphBytes {
		return StagePlan{}, ErrInvalidRequest
	}
	plan.Digest = digestOf(wire)
	return plan, nil
}

// Failure is the fixed, presentation-safe classification of a refused plan.
type Failure struct {
	Category        string
	AuthorityDenied bool
}

// ClassifyFailure maps compiler and graph errors to the Specification 007
// refusal categories. Unknown errors stay a generic invalid plan.
func ClassifyFailure(err error) Failure {
	switch {
	case errors.Is(err, ErrUnsupportedEffort):
		return Failure{Category: "unsupported_effort"}
	case errors.Is(err, ErrRuntimeBlocked), errors.Is(err, executiongraph.ErrUnresolvable):
		return Failure{Category: "runtime_unresolvable"}
	case errors.Is(err, ErrAuthorityExceeded), errors.Is(err, executiongraph.ErrAuthoritySubset):
		return Failure{Category: "authority_denied", AuthorityDenied: true}
	case errors.Is(err, executiongraph.ErrCycle), errors.Is(err, executiongraph.ErrUnsafeOverlap), errors.Is(err, executiongraph.ErrMissingOwner), errors.Is(err, executiongraph.ErrInvalidGraph):
		return Failure{Category: "invalid_stage_topology"}
	}
	return Failure{Category: "invalid_stage_plan"}
}

func digestOf(wire []byte) string {
	hash := sha256.Sum256(wire)
	return hex.EncodeToString(hash[:])
}
