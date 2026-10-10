package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const ConfiguredFormatVersion = 2
const ConfiguredWorkflowVersion = "configured-sequential-v1"

func sameStartContract(left, right State) bool {
	if left.Binding == nil || right.Binding == nil {
		return left.Binding == nil && right.Binding == nil
	}
	return left.Binding.Definition == right.Binding.Definition && left.Binding.ObservationDigest == right.Binding.ObservationDigest && left.Binding.RuntimePreview == right.Binding.RuntimePreview
}

func digestBytes(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }

type BindingView struct {
	ProjectRevision   string                 `json:"projectRevision"`
	LocalRevision     string                 `json:"localRevision"`
	ProjectID         string                 `json:"projectId"`
	Definition        workflowdefinition.Ref `json:"definition"`
	SchemaVersion     int                    `json:"schemaVersion"`
	SnapshotRef       string                 `json:"snapshotRef"`
	ObservationDigest string                 `json:"observationDigest"`
	RuntimePreview    string                 `json:"runtimePreview"`
	BoundAt           time.Time              `json:"boundAt"`
}

func InspectBinding(state State) *BindingView {
	if state.Binding == nil {
		return nil
	}
	b := state.Binding
	return &BindingView{ProjectID: b.ProjectID, Definition: b.Definition, SchemaVersion: b.SchemaVersion, SnapshotRef: b.SnapshotRef, ObservationDigest: b.ObservationDigest, RuntimePreview: b.RuntimePreview, BoundAt: b.BoundAt, ProjectRevision: b.ProjectRevision, LocalRevision: b.LocalRevision}
}

// WorkflowBinding retains the complete validated contract in the same protected
// publication as its Execution. SnapshotRef addresses this record, not mutable
// Project storage; a single publication avoids a half-published dependency.
type WorkflowBinding struct {
	ProjectRevision   string                        `json:"projectRevision"`
	LocalRevision     string                        `json:"localRevision"`
	Contexts          map[string]ContextObservation `json:"contexts"`
	ProjectID         string                        `json:"projectId"`
	Definition        workflowdefinition.Ref        `json:"definition"`
	SchemaVersion     int                           `json:"schemaVersion"`
	SnapshotRef       string                        `json:"snapshotRef"`
	Snapshot          json.RawMessage               `json:"snapshot"`
	ObservationDigest string                        `json:"observationDigest"`
	RuntimePreview    string                        `json:"runtimePreview"`
	BoundAt           time.Time                     `json:"boundAt"`
}

type DefinitionObservation struct {
	ProjectRevision, LocalRevision string
	Contexts                       map[string]ContextObservation
	ProjectID                      string
	Document                       workflowdefinition.Document
	Source                         string
	Digest                         string
}

type ContextObservation struct {
	Source  string `json:"source"`
	Digest  string `json:"digest"`
	Content []byte `json:"content"`
}
type DefinitionResolver interface {
	ResolveDefinition(context.Context, string) (DefinitionObservation, string)
}

func validDefinitionObservation(value DefinitionObservation) bool {
	doc, issues := workflowdefinition.Decode(value.Document.Canonical)
	if len(issues) != 0 || doc.Ref(value.Source) != value.Document.Ref(value.Source) || !reflect.DeepEqual(doc.Definition, value.Document.Definition) || !workflowdefinition.ValidDigest(value.Digest) || !workflowdefinition.ValidDigest(value.ProjectRevision) || !workflowdefinition.ValidDigest(value.LocalRevision) {
		return false
	}
	bytes := 0
	if len(value.Contexts) > 64 {
		return false
	}
	for key, c := range value.Contexts {
		bytes += len(c.Content)
		if !workflowdefinition.ValidKey(key) || c.Source != key || digestBytes(c.Content) != c.Digest {
			return false
		}
	}
	if bytes > workflowdefinition.MaxBytes {
		return false
	}
	for _, stage := range doc.Definition.Stages {
		for _, input := range stage.Inputs {
			if input.Kind == "project-context" && input.Required {
				if _, ok := value.Contexts[input.Source]; !ok {
					return false
				}
			}
		}
	}
	return true
}

// StageValidator is a domain-owned adapter, never an executable from a
// definition or an agent's claimed pass flag. Unknown policies fail closed.
type StageValidator interface {
	ValidateStageOutput(context.Context, State, string, workflowdefinition.Stage, workflowdefinition.Output, workflowdefinition.Validator, Reference) error
}
type StageResult struct {
	Inputs  map[string]Reference `json:"inputs"`
	Outputs map[string]Reference `json:"outputs"`
}
type ValidatorResult struct {
	CriterionID string    `json:"criterionId"`
	ValidatorID string    `json:"validatorId"`
	PolicyRef   string    `json:"policyRef"`
	Output      Reference `json:"output"`
}
type StageLedger struct {
	Ordinal    int               `json:"ordinal"`
	StageID    string            `json:"stageId"`
	Result     StageResult       `json:"result"`
	Validators []ValidatorResult `json:"validators"`
}

func (s Service) WithDefinitions(definitions DefinitionResolver, validators StageValidator) Service {
	s.definitions, s.validators = definitions, validators
	return s
}

func bindingDocument(state State) (workflowdefinition.Document, bool) {
	b := state.Binding
	if b != nil && (!workflowdefinition.ValidDigest(b.ProjectRevision) || !workflowdefinition.ValidDigest(b.LocalRevision)) {
		return workflowdefinition.Document{}, false
	}
	if b == nil || state.FormatVersion != ConfiguredFormatVersion || state.WorkflowVersion != ConfiguredWorkflowVersion || b.ProjectID != state.ProjectID || !b.Definition.Valid() || b.SchemaVersion != 1 || b.SnapshotRef != "execution:"+state.ExecutionID+"#/binding/snapshot" || !workflowdefinition.ValidDigest(b.ObservationDigest) || !workflowdefinition.ValidDigest(b.RuntimePreview) || !b.BoundAt.Equal(state.CreatedAt) {
		return workflowdefinition.Document{}, false
	}
	doc, issues := workflowdefinition.Decode(b.Snapshot)
	if len(b.Contexts) > 64 {
		return workflowdefinition.Document{}, false
	}
	contextBytes := 0
	for key, c := range b.Contexts {
		contextBytes += len(c.Content)
		if !workflowdefinition.ValidKey(key) || c.Source != key || digestBytes(c.Content) != c.Digest {
			return workflowdefinition.Document{}, false
		}
	}
	if contextBytes > workflowdefinition.MaxBytes {
		return workflowdefinition.Document{}, false
	}
	for _, stage := range doc.Definition.Stages {
		for _, input := range stage.Inputs {
			if input.Kind == "project-context" && input.Required {
				if _, ok := b.Contexts[input.Source]; !ok {
					return workflowdefinition.Document{}, false
				}
			}
		}
	}
	return doc, len(issues) == 0 && doc.Ref(b.Definition.Source) == b.Definition && string(doc.Canonical) == string(b.Snapshot)
}

func CurrentStage(state State) *workflowdefinition.Stage {
	doc, ok := bindingDocument(state)
	if !ok || state.StageOrdinal < 0 || state.StageOrdinal >= len(doc.Definition.Stages) {
		return nil
	}
	stage := doc.Definition.Stages[state.StageOrdinal]
	return &stage
}

func StageInputs(state State) map[string]Reference {
	stage := CurrentStage(state)
	if stage == nil {
		return nil
	}
	refs := map[string]Reference{}
	for _, input := range stage.Inputs {
		switch input.Kind {
		case "stage-output":
			if ref, ok := priorOutput(state, input.Source); ok {
				refs[input.ID] = ref
			}
		case "project-context":
			if c, ok := state.Binding.Contexts[input.Source]; ok {
				refs[input.ID] = Reference{Kind: "context", ID: input.Source, Digest: c.Digest}
			}
		}
	}
	return refs
}

func configuredPhaseGate(stage workflowdefinition.Stage) LifecycleFactKind {
	for _, gate := range stage.HumanGates {
		if gate.Timing == "before" {
			return LifecycleFactKind(gate.Kind)
		}
	}
	return ""
}

func configuredFactAllowed(state State, kind LifecycleFactKind) bool {
	stage := CurrentStage(state)
	if stage == nil {
		return false
	}
	switch kind {
	case FactStageReview:
		return state.Status == ExecutionActive
	case FactBlocked, FactNeedsDecision, FactNeedsApproval:
		return true
	case FactHumanAcceptance:
		return state.Status == ExecutionCompleted && stage.Phase == "completion"
	default:
		return configuredPhaseGate(*stage) == kind
	}
}

func configuredFacts(state State) (map[LifecycleFactKind]bool, LifecycleConditions, bool) {
	facts := map[LifecycleFactKind]bool{}
	conditions := LifecycleConditions{}
	for _, event := range state.Transitions {
		if event.Fact == nil {
			continue
		}
		f := event.Fact
		if f.ScopeDigest != executionScopeDigest(state) || !validReference(f.Reference) {
			return nil, conditions, false
		}
		if !validText(f.Actor) {
			return nil, conditions, false
		}
		switch f.Kind {
		case FactStageReview:
			if !f.Active || !validDigest(f.ResultDigest) {
				return nil, conditions, false
			}
		case FactBlocked:
			conditions.Blocked = f.Active
		case FactNeedsDecision:
			conditions.NeedsDecision = f.Active
		case FactNeedsApproval:
			conditions.NeedsApproval = f.Active
		case FactPlanningAuthority, FactImplementationAuthority, FactReviewStarted, FactHumanAcceptance:
			if !f.Active || facts[f.Kind] {
				return nil, conditions, false
			}
			facts[f.Kind] = true
		default:
			return nil, conditions, false
		}
	}
	return facts, conditions, true
}

func configuredLifecycle(state State) (LifecycleProjection, error) {
	facts, conditions, ok := configuredFacts(state)
	stage := CurrentStage(state)
	if !ok || stage == nil {
		return LifecycleProjection{}, ErrRecoveryRequired
	}
	var value WorkItemLifecycleStage
	switch stage.Phase {
	case "intake":
		value = LifecycleIntake
	case "specification":
		value = LifecycleSpecifying
	case "planning":
		value = LifecycleSpecified
		if facts[FactPlanningAuthority] {
			value = LifecyclePlanning
		}
	case "implementation":
		value = LifecyclePlanned
		if facts[FactImplementationAuthority] {
			value = LifecycleImplementing
		}
	case "review":
		value = LifecycleImplemented
		if facts[FactReviewStarted] {
			value = LifecycleReviewing
		}
	case "completion":
		value = LifecycleReviewing
		if state.Status == ExecutionCompleted {
			value = LifecycleReviewed
			if facts[FactHumanAcceptance] {
				value = LifecycleAccepted
			}
		}
	default:
		return LifecycleProjection{}, ErrRecoveryRequired
	}
	return LifecycleProjection{Stage: value, Conditions: conditions}, nil
}

func validConfiguredState(state State) bool {
	doc, ok := bindingDocument(state)
	if !ok || !validOpaqueID(state.ExecutionID) || !validText(state.ProjectID) || !validText(state.RepositoryKey) || !validWorkItem(state.WorkItem) || !SupportedRuntime(state.RuntimeID) || !validIdentity(state.Provenance) || state.CreatedAt.IsZero() || !validExecutionStatus(state.Status) || state.Revision != uint64(len(state.Transitions))+1 || len(state.Transitions) > 512 || len(state.Projections) > 512 {
		return false
	}
	ordinal, status, at := 0, ExecutionActive, state.CreatedAt
	prefix := state
	prefix.Transitions = nil
	prefix.Status = status
	prefix.StageOrdinal = ordinal
	prefix.Stage = Stage(doc.Definition.Stages[0].ID)
	for i, event := range state.Transitions {
		stage := doc.Definition.Stages[ordinal]
		if event.Revision != uint64(i+2) || event.From != Stage(stage.ID) || event.RequestDigest == "" || !validDigest(event.RequestDigest) || !validIdentity(event.Provenance) || event.CommittedAt.Before(at) || !validOptionalText(event.Next) || len(event.References) > maxReferences {
			return false
		}
		for _, ref := range event.References {
			if !validReference(ref) {
				return false
			}
		}
		next := ordinal
		switch event.Outcome {
		case OutcomeFact:
			if event.Fact == nil || event.Ledger != nil || !configuredFactAllowed(prefix, event.Fact.Kind) {
				return false
			}
		case OutcomeFailed:
			if status != ExecutionActive || event.Fact != nil || event.Ledger != nil {
				return false
			}
			status = ExecutionInterrupted
		case OutcomeResumed:
			if status != ExecutionInterrupted || event.Fact != nil || event.Ledger != nil {
				return false
			}
			status = ExecutionActive
		case OutcomePassed:
			if status != ExecutionActive || event.Fact != nil || event.Ledger == nil || event.Ledger.Ordinal != ordinal || event.Ledger.StageID != stage.ID {
				return false
			}
			facts, cond, valid := configuredFacts(prefix)
			gate := configuredPhaseGate(stage)
			if !valid || cond.Blocked || cond.NeedsDecision || cond.NeedsApproval || gate != "" && !facts[gate] || !validStageResult(prefix, stage, event.Ledger.Result) || !reflect.DeepEqual(event.Ledger.Validators, expectedValidators(stage, event.Ledger.Result)) {
				return false
			}
			if ordinal == len(doc.Definition.Stages)-1 {
				// The final checkpoint preserves pending human acceptance.
				status = ExecutionCompleted
			} else {
				next++
			}
			for _, validator := range stage.Validators {
				required := false
				for _, criterion := range stage.CompletionCriteria {
					required = required || criterion.ValidatorRef == validator.ID
				}
				if required && validator.Kind == "human-review" {
					found := false
					for _, prior := range prefix.Transitions {
						if prior.From == event.From && prior.Fact != nil && prior.Fact.Kind == FactStageReview && prior.Fact.ResultDigest == digest(event.Ledger.Result) {
							found = true
						}
					}
					if !found {
						return false
					}
				}
			}
		default:
			return false
		}
		if event.Outcome != OutcomeFact && event.Fact != nil || event.To != Stage(doc.Definition.Stages[next].ID) {
			return false
		}
		ordinal, at = next, event.CommittedAt
		prefix.StageOrdinal, prefix.Stage, prefix.Status = ordinal, event.To, status
		prefix.Transitions = state.Transitions[:i+1]
	}
	if state.StageOrdinal != ordinal || state.Stage != Stage(doc.Definition.Stages[ordinal].ID) || state.Status != status || !state.UpdatedAt.Equal(at) {
		return false
	}
	if _, _, ok := configuredFacts(state); !ok || !validLifecycleReferences(state) {
		return false
	}
	if status == ExecutionCompleted {
		if state.Terminal == nil || state.Terminal.Status != completion.Success || !reflect.DeepEqual(state.Terminal.ConfirmedEffects, []string{"local_execution_transition"}) {
			return false
		}
	} else if state.Terminal != nil {
		return false
	}
	// Projection cannot establish or advance local state. Configured records use
	// the same bounded effects and exact revision-based projection protocol.
	seen := map[uint64]bool{}
	for _, p := range state.Projections {
		if p.ExecutionRevision < 2 || p.ExecutionRevision > state.Revision || seen[p.ExecutionRevision] || p.Key != projectionKey(state.ExecutionID, p.ExecutionRevision) || p.Complete != effectsConfirmed(p.Intended, p.Confirmed) || len(p.Intended) > 16 || len(p.Confirmed) > len(p.Intended) || hasDuplicateEffects(p.Intended) || hasDuplicateEffects(p.Confirmed) {
			return false
		}
		if !validProjectionRecord(state, p) {
			return false
		}
		seen[p.ExecutionRevision] = true
		for _, e := range append(cloneEffects(p.Intended), p.Confirmed...) {
			if !validEffect(e) || e.Kind != PostTransitionComment && !ValidDesiredProjectionLabel(e.Value) {
				return false
			}
		}
	}
	return true
}

func priorOutput(state State, source string) (Reference, bool) {
	parts := strings.Split(source, "/")
	if len(parts) != 2 {
		return Reference{}, false
	}
	for _, event := range state.Transitions {
		if event.Ledger != nil && event.Ledger.StageID == parts[0] {
			ref, ok := event.Ledger.Result.Outputs[parts[1]]
			return ref, ok
		}
	}
	return Reference{}, false
}
func validStageResult(state State, stage workflowdefinition.Stage, value StageResult) bool {
	if len(value.Inputs) > len(stage.Inputs) || len(value.Outputs) > len(stage.Outputs) {
		return false
	}
	inputs, outputs := map[string]bool{}, map[string]bool{}
	for _, in := range stage.Inputs {
		inputs[in.ID] = true
		ref, exists := value.Inputs[in.ID]
		if in.Required && in.Kind != "work-item" && !exists {
			return false
		}
		if exists && !validReference(ref) && !(ref.Kind == "context" && validText(ref.ID) && validDigest(ref.Digest)) {
			return false
		}
		if in.Kind == "work-item" && exists || in.Kind != "project-context" && exists && ref.Kind == "context" {
			return false
		}
		if in.Kind == "project-context" && (in.Required || exists) {
			c, ok := state.Binding.Contexts[in.Source]
			if !ok || ref != (Reference{Kind: "context", ID: in.Source, Digest: c.Digest}) {
				return false
			}
		}
		if in.Kind == "artifact" && exists && (ref.ID != in.Source || ref.Kind != "artifact") {
			return false
		}
		if in.Kind == "stage-output" && (in.Required || exists) {
			previous, ok := priorOutput(state, in.Source)
			if !ok || ref != previous {
				return false
			}
		}
	}
	for _, out := range stage.Outputs {
		outputs[out.ID] = true
		ref, exists := value.Outputs[out.ID]
		if out.Required && !exists || exists && !validReference(ref) {
			return false
		}
		if exists && ref.Kind != "artifact" {
			return false
		}
	}
	for id := range value.Inputs {
		if !inputs[id] {
			return false
		}
	}
	for id := range value.Outputs {
		if !outputs[id] {
			return false
		}
	}
	for _, c := range stage.CompletionCriteria {
		if _, ok := value.Outputs[c.OutputRef]; !ok {
			return false
		}
	}
	return true
}
func expectedValidators(stage workflowdefinition.Stage, value StageResult) []ValidatorResult {
	results := make([]ValidatorResult, 0, len(stage.CompletionCriteria))
	for _, c := range stage.CompletionCriteria {
		for _, v := range stage.Validators {
			if v.ID == c.ValidatorRef {
				results = append(results, ValidatorResult{c.ID, v.ID, v.PolicyRef, value.Outputs[c.OutputRef]})
			}
		}
	}
	return results
}

func (s Service) transitionConfigured(ctx context.Context, state State, repository Repository, input TransitionInput) Result {
	if input.ExpectedRevision != state.Revision {
		if replayed(state, input.ExpectedRevision, digest(input)) {
			return result(Succeeded, "workflow_transition_replayed", state)
		}
		return result(Denied, "stale_execution_revision", state)
	}
	if input.Stage != state.Stage || input.Outcome != OutcomePassed && input.Outcome != OutcomeFailed || state.Status != ExecutionActive || !validOptionalText(input.Next) || len(input.References) > maxReferences {
		return result(ValidationFailed, "invalid_workflow_transition", state)
	}
	stage := CurrentStage(state)
	action := NextGateAction(state)
	if input.Outcome == OutcomePassed && action != nil && action.Operation == "fact" {
		return result(Denied, "workflow_authority_required", state)
	}
	var ledger *StageLedger
	if input.Outcome == OutcomePassed {
		if input.StageResult == nil || !validStageResult(state, *stage, *input.StageResult) {
			return result(ValidationFailed, "stage_prerequisite_missing", state)
		}
		for _, refs := range []map[string]Reference{input.StageResult.Inputs, input.StageResult.Outputs} {
			for _, ref := range refs {
				if ref.Kind == "context" {
					continue
				}
				if s.references == nil || s.references.Validate(ctx, state.ExecutionID, repository.Path, ref) != nil {
					return result(ValidationFailed, "stage_prerequisite_missing", state)
				}
			}
		}
		// Every declared output is a typed, correlated retained artifact, even
		// when no additional completion criterion references that output.
		for _, output := range stage.Outputs {
			if ref, exists := input.StageResult.Outputs[output.ID]; exists {
				if s.validators == nil || s.validators.ValidateStageOutput(ctx, state, repository.Path, *stage, output, workflowdefinition.Validator{Kind: "artifact-schema", PolicyRef: "builtin-sdd-v1"}, ref) != nil {
					return result(ValidationFailed, "stage_validator_failed", state)
				}
			}
		}
		for _, c := range stage.CompletionCriteria {
			for _, v := range stage.Validators {
				if c.ValidatorRef == v.ID {
					if v.Kind == "human-review" {
						found := false
						for _, event := range state.Transitions {
							if event.From == state.Stage && event.Fact != nil && event.Fact.Kind == FactStageReview && event.Fact.ResultDigest == digest(input.StageResult) && s.references != nil && s.references.Validate(ctx, state.ExecutionID, repository.Path, event.Fact.Reference) == nil {
								found = true
							}
						}
						if !found {
							return result(Denied, "stage_human_review_required", state)
						}
					}
					var output workflowdefinition.Output
					for _, candidate := range stage.Outputs {
						if candidate.ID == c.OutputRef {
							output = candidate
						}
					}
					if s.validators == nil || s.validators.ValidateStageOutput(ctx, state, repository.Path, *stage, output, v, input.StageResult.Outputs[c.OutputRef]) != nil {
						return result(ValidationFailed, "stage_validator_failed", state)
					}
				}
			}
		}
		ledger = &StageLedger{state.StageOrdinal, stage.ID, *input.StageResult, expectedValidators(*stage, *input.StageResult)}
	}
	for _, ref := range input.References {
		if !validReference(ref) || s.references == nil || s.references.Validate(ctx, state.ExecutionID, repository.Path, ref) != nil {
			return result(ValidationFailed, "workflow_reference_unavailable", state)
		}
	}
	previous := cloneState(state)
	next := state.Stage
	category, status := "workflow_interrupted", Interrupted
	state.Status = ExecutionInterrupted
	if input.Outcome == OutcomePassed {
		category, status = "workflow_advanced", Succeeded
		doc, _ := bindingDocument(state)
		if state.StageOrdinal == len(doc.Definition.Stages)-1 {
			state.Status = ExecutionCompleted
			category = "workflow_completed"
			state.Terminal = &Terminal{Status: completion.Success, ConfirmedEffects: []string{"local_execution_transition"}}
		} else {
			state.StageOrdinal++
			next = Stage(doc.Definition.Stages[state.StageOrdinal].ID)
			state.Status = ExecutionActive
		}
	}
	event := transitionFor(state, input, next, s.now().UTC(), s.source)
	event.Ledger = ledger
	state.Revision++
	state.Stage = next
	state.UpdatedAt = event.CommittedAt
	state.Transitions = append(state.Transitions, event)
	if !ValidState(state) {
		return result(ValidationFailed, "invalid_execution_state", previous)
	}
	return s.save(ctx, state, status, category)
}
