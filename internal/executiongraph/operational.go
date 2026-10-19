package executiongraph

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/strictjson"
	"reflect"
	"strings"
	"time"
)

const OperationalGraphVersion = 4
const OperationalSingleStageRunVersion = 2

type OperationalResolution struct {
	ResolvedProfile       runtimeprofile.ResolvedProfileV4 `json:"resolvedProfile"`
	RuntimeID             string                           `json:"runtimeId"`
	ModelProfileID        string                           `json:"modelProfileId"`
	ConfigurationRevision uint64                           `json:"configurationRevision"`
	ObservationRevision   uint64                           `json:"observationRevision"`
}

type OperationalChildEnvelope struct {
	Capability         CapabilityRequest     `json:"capability"`
	Inputs             []string              `json:"inputs"`
	Outputs            []string              `json:"outputs"`
	Scope              Scope                 `json:"scope"`
	Workspace          string                `json:"workspace"`
	Dependencies       []string              `json:"dependencies"`
	AllowedEffects     []Effect              `json:"allowedEffects"`
	AuthorityReference string                `json:"authorityReference"`
	Controls           ExecutionControls     `json:"controls"`
	ValidationOwner    bool                  `json:"validationOwner"`
	IntegrationOwner   bool                  `json:"integrationOwner"`
	Optional           bool                  `json:"optional"`
	Resolution         OperationalResolution `json:"resolution"`
}

type OperationalGraph struct {
	FormatVersion        int                `json:"formatVersion"`
	Parent               ParentExecution    `json:"parent"`
	Children             []OperationalChild `json:"children"`
	ClassificationPolicy string             `json:"classificationPolicy"`
	DispatchAuthorityRef string             `json:"dispatchAuthorityRef"`
	StageConcurrency     uint32             `json:"stageConcurrency"`
	CancelRequest        *CancelRequest     `json:"cancelRequest,omitempty"`
	StorageRevision      [sha256.Size]byte  `json:"-"`
}

type OperationalChild struct {
	ExecutionID   string                   `json:"executionId"`
	ParentID      string                   `json:"parentId"`
	GraphRevision uint64                   `json:"graphRevision"`
	NodeKey       string                   `json:"nodeKey"`
	Envelope      OperationalChildEnvelope `json:"envelope"`
	Attempts      []OperationalAttempt     `json:"attempts"`
}

type OperationalSingleStageBinding struct {
	ExecutionGeneration string                `json:"executionGeneration"`
	WorkflowRevision    uint64                `json:"workflowRevision"`
	PreviewDigest       string                `json:"previewDigest"`
	ProposalDigest      string                `json:"proposalDigest"`
	InputDigest         string                `json:"inputDigest"`
	AxiomDigest         string                `json:"axiomDigest"`
	Workspace           string                `json:"workspace"`
	Inbox               string                `json:"inbox"`
	Effects             []Effect              `json:"effects"`
	Controls            ExecutionControls     `json:"controls"`
	Resolution          OperationalResolution `json:"resolution"`
}

type OperationalSingleStageRun struct {
	Kind            string                        `json:"kind"`
	FormatVersion   int                           `json:"formatVersion"`
	ScopeDigest     string                        `json:"scopeDigest"`
	Scope           SingleStageScope              `json:"scope"`
	Binding         OperationalSingleStageBinding `json:"binding"`
	CreatedAt       time.Time                     `json:"createdAt"`
	RunEpoch        uint64                        `json:"runEpoch"`
	Authorities     []string                      `json:"authorityRefs"`
	Attempts        []OperationalAttempt          `json:"attempts"`
	CancelRequest   *CancelRequest                `json:"cancelRequest,omitempty"`
	StorageRevision [sha256.Size]byte             `json:"-"`
}

type OperationalAttempt struct {
	SuccessorAttempt
	Diagnostic *AttemptDiagnostic `json:"diagnostic,omitempty"`
}

func ValidateOperationalResolution(r OperationalResolution) error {
	p := r.ResolvedProfile
	if runtimeprofile.ValidateResolvedProfileV4(p) != nil || r.RuntimeID != p.RuntimeID || r.ModelProfileID != p.ProfileID || r.ConfigurationRevision != p.ConfigurationRevision || r.ObservationRevision != p.ObservationRevision {
		return ErrInvalidGraph
	}
	return nil
}

func ValidOperationalProfileControls(p runtimeprofile.ResolvedProfileV4, controls ExecutionControls, capability CapabilityRequest) bool {
	if runtimeprofile.ValidateResolvedProfileV4(p) != nil {
		return false
	}
	expected := ""
	if p.Effort.Mode == runtimeprofile.EffortExplicit {
		expected = p.Effort.Value
	}
	if controls.ReasoningEffort != expected {
		return false
	}
	found := false
	for _, value := range capability.Capabilities {
		if strings.HasPrefix(value, "reasoning-effort-") {
			if expected == "" || value != "reasoning-effort-"+expected || found {
				return false
			}
			found = true
		}
	}
	return expected == "" || found
}

func (e OperationalChildEnvelope) topology() ChildEnvelope {
	return ChildEnvelope{Capability: e.Capability, Inputs: e.Inputs, Outputs: e.Outputs, Scope: e.Scope, Workspace: e.Workspace, Dependencies: e.Dependencies, AllowedEffects: e.AllowedEffects, AuthorityReference: e.AuthorityReference, Controls: e.Controls, ValidationOwner: e.ValidationOwner, IntegrationOwner: e.IntegrationOwner, Optional: e.Optional, Resolution: Resolution{RuntimeID: e.Resolution.RuntimeID, ModelProfileID: e.Resolution.ModelProfileID, ConfigurationRevision: e.Resolution.ConfigurationRevision, ObservationRevision: e.Resolution.ObservationRevision}}
}

func ValidateOperationalGraph(g OperationalGraph) error {
	if g.FormatVersion != OperationalGraphVersion || len(g.Children) == 0 || len(g.Children) > maxNodes || g.ClassificationPolicy != ClassificationPolicyV1 || !successorDigest(g.DispatchAuthorityRef) || g.StageConcurrency == 0 || g.StageConcurrency > MaxStageConcurrency || !utc(g.Parent.CreatedAt) {
		return ErrInvalidGraph
	}
	topology := Graph{Parent: g.Parent}
	for _, c := range g.Children {
		if ValidateOperationalResolution(c.Envelope.Resolution) != nil || !ValidOperationalProfileControls(c.Envelope.Resolution.ResolvedProfile, c.Envelope.Controls, c.Envelope.Capability) {
			return ErrInvalidGraph
		}
		topology.Children = append(topology.Children, ChildExecution{ExecutionID: c.ExecutionID, ParentID: c.ParentID, GraphRevision: c.GraphRevision, NodeKey: c.NodeKey, Envelope: c.Envelope.topology()})
	}
	if !validGraphTopology(topology) {
		return ErrInvalidGraph
	}
	if r := g.CancelRequest; r != nil {
		if !validOpaqueID(r.RequestID) || r.ParentID != g.Parent.ExecutionID || r.GraphRevision != g.Parent.GraphRevision || r.ActorKind != "human-operator" || !boundedIdentity(r.Actor) || !utc(r.RequestedAt) || r.RequestedAt.Before(g.Parent.CreatedAt) || !successorDigest(r.AuthorityRef) {
			return ErrInvalidGraph
		}
	}
	ids := map[string]bool{g.Parent.ExecutionID: true}
	for _, c := range g.Children {
		ids[c.ExecutionID] = true
	}
	facts := map[string]bool{}
	for _, c := range g.Children {
		attempts, err := operationalAttemptSequence(c.Attempts)
		if err != nil {
			return err
		}
		if err := validateAttemptSequence(attempts, attemptScope{created: g.Parent.CreatedAt, cancel: g.CancelRequest, parentID: g.Parent.ExecutionID, childID: c.ExecutionID, revision: g.Parent.GraphRevision, maximum: c.Envelope.Controls.MaximumAttempts, integration: c.Envelope.IntegrationOwner, authority: map[string]bool{g.DispatchAuthorityRef: true}}, ids, facts); err != nil {
			return err
		}
	}
	for id := range facts {
		if ids[id] {
			return ErrInvalidGraph
		}
	}
	if g.CancelRequest != nil && (ids[g.CancelRequest.RequestID] || facts[g.CancelRequest.RequestID]) {
		return ErrInvalidGraph
	}
	return nil
}

func operationalAttemptSequence(attempts []OperationalAttempt) ([]SuccessorAttempt, error) {
	result := make([]SuccessorAttempt, len(attempts))
	for i, a := range attempts {
		if validateOperationalDiagnostic(a) != nil {
			return nil, ErrInvalidGraph
		}
		result[i] = a.SuccessorAttempt
	}
	return result, nil
}

func ValidateOperationalSingleStageRun(r OperationalSingleStageRun) error {
	digest, err := SingleStageScopeDigest(r.Scope)
	b := r.Binding
	if r.Kind != SingleStageRunKind || r.FormatVersion != OperationalSingleStageRunVersion || err != nil || digest != r.ScopeDigest || !utc(r.CreatedAt) || ValidateOperationalResolution(b.Resolution) != nil {
		return ErrInvalidGraph
	}
	if !successorDigest(b.ExecutionGeneration) || b.WorkflowRevision == 0 || !successorDigest(b.PreviewDigest) || !successorDigest(b.ProposalDigest) || !successorDigest(b.InputDigest) || !successorDigest(b.AxiomDigest) || !validTarget(b.Workspace) || !validTarget(b.Inbox) || b.Workspace == b.Inbox || !validDispatchEffects(b.Effects) {
		return ErrInvalidGraph
	}
	if b.Controls.MaximumAttempts < 1 || b.Controls.MaximumAttempts > MaxSingleStageAttempts || b.Controls.Timeout <= 0 || b.Controls.Timeout > MaxSingleStageAttemptTime {
		return ErrInvalidGraph
	}
	expected := ""
	if b.Resolution.ResolvedProfile.Effort.Mode == runtimeprofile.EffortExplicit {
		expected = b.Resolution.ResolvedProfile.Effort.Value
	}
	if b.Controls.ReasoningEffort != expected || len(r.Authorities) == 0 || len(r.Authorities) > MaxSingleStageAttempts || len(r.Attempts) > len(r.Authorities) || len(r.Authorities) > len(r.Attempts)+1 {
		return ErrInvalidGraph
	}
	refs := map[string]bool{}
	for _, ref := range r.Authorities {
		if !successorDigest(ref) || refs[ref] {
			return ErrInvalidGraph
		}
		refs[ref] = true
	}
	if c := r.CancelRequest; c != nil {
		if !validOpaqueID(c.RequestID) || c.ParentID != r.Scope.ExecutionID || c.GraphRevision != b.WorkflowRevision || c.ActorKind != "human-operator" || !boundedIdentity(c.Actor) || !utc(c.RequestedAt) || c.RequestedAt.Before(r.CreatedAt) || !successorDigest(c.AuthorityRef) || refs[c.AuthorityRef] {
			return ErrInvalidGraph
		}
	}
	attempts, err := operationalAttemptSequence(r.Attempts)
	if err != nil {
		return err
	}
	ids := map[string]bool{r.Scope.ExecutionID: true}
	facts := map[string]bool{}
	scope := attemptScope{created: r.CreatedAt, cancel: r.CancelRequest, parentID: r.Scope.ExecutionID, childID: r.Scope.ExecutionID, revision: b.WorkflowRevision, maximum: b.Controls.MaximumAttempts, authority: refs}
	if err := validateAttemptSequence(attempts, scope, ids, facts); err != nil {
		return err
	}
	for _, a := range attempts {
		if a.LeaseEpoch > r.RunEpoch {
			return ErrInvalidGraph
		}
	}
	if r.CancelRequest != nil && (ids[r.CancelRequest.RequestID] || facts[r.CancelRequest.RequestID]) {
		return ErrInvalidGraph
	}
	return nil
}

func EncodeOperationalGraph(g OperationalGraph) ([]byte, error) {
	if ValidateOperationalGraph(g) != nil {
		return nil, ErrInvalidGraph
	}
	wire, err := json.Marshal(g)
	if err != nil || len(wire)+1 > MaxGraphBytes {
		return nil, ErrInvalidGraph
	}
	return append(wire, '\n'), nil
}

func DecodeOperationalGraph(wire []byte) (OperationalGraph, error) {
	var g OperationalGraph
	if strictjson.Decode(wire, &g, MaxGraphBytes) != nil || ValidateOperationalGraph(g) != nil {
		return OperationalGraph{}, ErrInvalidGraph
	}
	g.StorageRevision = sha256.Sum256(wire)
	return g, nil
}

func EncodeOperationalSingleStageRun(r OperationalSingleStageRun) ([]byte, error) {
	if ValidateOperationalSingleStageRun(r) != nil {
		return nil, ErrInvalidGraph
	}
	wire, err := json.Marshal(r)
	if err != nil || len(wire)+1 > MaxSingleStageRunBytes {
		return nil, ErrInvalidGraph
	}
	return append(wire, '\n'), nil
}

func DecodeOperationalSingleStageRun(wire []byte) (OperationalSingleStageRun, error) {
	var r OperationalSingleStageRun
	if strictjson.Decode(wire, &r, MaxSingleStageRunBytes) != nil || ValidateOperationalSingleStageRun(r) != nil {
		return OperationalSingleStageRun{}, ErrInvalidGraph
	}
	r.StorageRevision = sha256.Sum256(wire)
	return r, nil
}

func ValidateOperationalGraphUpdate(old, next OperationalGraph) error {
	if ValidateOperationalGraph(old) != nil || ValidateOperationalGraph(next) != nil {
		return ErrInvalidGraph
	}
	a, b := old, next
	a.Children, b.Children = nil, nil
	a.CancelRequest, b.CancelRequest = nil, nil
	a.StorageRevision, b.StorageRevision = [32]byte{}, [32]byte{}
	if !reflect.DeepEqual(a, b) || len(old.Children) != len(next.Children) || old.CancelRequest != nil && !reflect.DeepEqual(old.CancelRequest, next.CancelRequest) {
		return ErrInvalidGraph
	}
	for i, child := range old.Children {
		other := next.Children[i]
		prior, current := child.Attempts, other.Attempts
		child.Attempts, other.Attempts = nil, nil
		if !reflect.DeepEqual(child, other) || validateOperationalAttemptPrefix(prior, current) != nil {
			return ErrInvalidGraph
		}
	}
	return nil
}

func ValidateOperationalSingleStageUpdate(old, next OperationalSingleStageRun) error {
	if ValidateOperationalSingleStageRun(old) != nil || ValidateOperationalSingleStageRun(next) != nil {
		return ErrInvalidGraph
	}
	if old.ScopeDigest != next.ScopeDigest || !reflect.DeepEqual(old.Scope, next.Scope) || !reflect.DeepEqual(old.Binding, next.Binding) || !old.CreatedAt.Equal(next.CreatedAt) || next.RunEpoch < old.RunEpoch {
		return ErrInvalidGraph
	}
	if len(next.Authorities) < len(old.Authorities) || !reflect.DeepEqual(old.Authorities, next.Authorities[:len(old.Authorities)]) || old.CancelRequest != nil && !reflect.DeepEqual(old.CancelRequest, next.CancelRequest) {
		return ErrInvalidGraph
	}
	return validateOperationalAttemptPrefix(old.Attempts, next.Attempts)
}

func validateOperationalAttemptPrefix(old, next []OperationalAttempt) error {
	if len(next) < len(old) {
		return ErrInvalidGraph
	}
	for i, a := range old {
		b := next[i]
		if (a.Status != AttemptRunning || b.Status == AttemptRunning) && !reflect.DeepEqual(a.Diagnostic, b.Diagnostic) {
			return ErrInvalidGraph
		}
	}
	a, err := operationalAttemptSequence(old)
	if err != nil {
		return err
	}
	b, err := operationalAttemptSequence(next)
	if err != nil {
		return err
	}
	return validateAttemptPrefix(a, b)
}
