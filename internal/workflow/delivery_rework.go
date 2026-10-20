package workflow

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflowdefinition"
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

type ReworkStore interface {
	ApplyDeliveryRework(context.Context, State, ReworkPreview) (State, error)
}

type reworkAdmission struct{ state *State }

func (m *reworkAdmission) Create(_ context.Context, s State) error {
	if m.state != nil {
		return ErrConflict
	}
	m.state = &s
	return nil
}
func (m *reworkAdmission) Load(context.Context, string, string, WorkItem) (State, error) {
	if m.state == nil {
		return State{}, ErrNotFound
	}
	return *m.state, nil
}
func (*reworkAdmission) Save(context.Context, State) error { return ErrConflict }

type retainedDefinition struct{ observation DefinitionObservation }

func (r retainedDefinition) ResolveDefinition(context.Context, string) (DefinitionObservation, string) {
	return r.observation, ""
}

func (s Service) PreviewDeliveryRework(ctx context.Context, original State, d DeliveryDecision, input ReworkInput, target Target) (ReworkPreview, error) {
	p := ReworkPreview{FormatVersion: 1, Kind: "delivery_rework", OriginalPacket: d.Packet, DecisionDigest: d.Digest(), Input: input}
	if d.Outcome != "reject" || !deliveryPacketCurrent(original, d.Packet) || original.DeliveryDecision == nil || !SameDeliveryDecision(*original.DeliveryDecision, d) || !validOpaqueID(input.NewExecutionID) || input.NewExecutionID == original.ExecutionID || !validText(input.Scope) || input.Plan.Kind != "plan" || !validReference(input.Plan) || !validReference(input.Authority) || input.Authority == d.Authority || len(input.Inputs) > 16 || s.references == nil {
		return p, ErrDelivery
	}
	if _, err := EncodeDeliveryDecision(d); err != nil {
		return p, err
	}
	resolved, failed := s.ValidateTarget(ctx, target)
	if failed.Category != "" && failed.Status != Succeeded {
		return p, ErrDelivery
	}
	if resolved.ProjectID != original.ProjectID || resolved.Repository.Key != original.RepositoryKey || resolved.WorkItem != original.WorkItem {
		return p, ErrDelivery
	}
	for _, r := range append([]Reference{input.Plan, input.Authority}, input.Inputs...) {
		if err := s.references.Validate(ctx, original.ExecutionID, resolved.Repository.Path, r); err != nil {
			return p, err
		}
	}
	for _, r := range input.Inputs {
		found := false
		for _, stage := range d.Packet.Stages {
			for _, artifact := range stage.Artifacts {
				found = found || artifact == r
			}
		}
		if !found {
			return p, ErrDelivery
		}
	}
	admission := &reworkAdmission{}
	s.store = admission
	s.allocateID = func() (string, error) { return input.NewExecutionID, nil }
	s.now = func() time.Time { return original.UpdatedAt.Add(time.Nanosecond) }
	switch input.RevisionChoice {
	case "retained":
		b := original.Binding
		if b == nil {
			return p, ErrDelivery
		}
		doc, issues := workflowdefinition.Decode(b.Snapshot)
		if len(issues) != 0 {
			return p, ErrDelivery
		}
		s.definitions = retainedDefinition{DefinitionObservation{ProjectID: original.ProjectID, Document: doc, Source: b.Definition.Source, Digest: b.ObservationDigest, ProjectRevision: b.ProjectRevision, LocalRevision: b.LocalRevision, Contexts: b.Contexts}}
	case "selected":
		if s.definitions == nil {
			return p, ErrDelivery
		}
	default:
		return p, ErrDelivery
	}
	target.ExecutionID = ""
	target.RuntimeID = input.RuntimeID
	target.RuntimePreview = input.RuntimePreview
	target.FrozenProfile = input.FrozenProfile
	target.ReviewedTarget = nil
	started := s.Start(ctx, target)
	if started.Status != Succeeded || admission.state == nil {
		return p, ErrDelivery
	}
	p.NewState = started.State
	if _, err := EncodeReworkPreview(p); err != nil {
		return p, err
	}
	return p, nil
}
func ApplyDeliveryRework(ctx context.Context, store ReworkStore, original State, p ReworkPreview, preview string, local, execution bool) (State, error) {
	if _, err := EncodeReworkPreview(p); err != nil || !local || !execution || preview != p.Digest() || !deliveryPacketCurrent(original, p.OriginalPacket) || original.DeliveryDecision == nil || original.DeliveryDecision.Outcome != "reject" || original.DeliveryDecision.Digest() != p.DecisionDigest {
		return State{}, ErrDelivery
	}
	return store.ApplyDeliveryRework(ctx, original, p)
}
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
