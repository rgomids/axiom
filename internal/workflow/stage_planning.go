package workflow

import (
	"context"
	"encoding/json"

	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// StageInputBinding is one digest-bound logical stage input taken from the
// retained Execution: its pinned binding contexts, its validated stage ledger,
// or its canonical Work Item identity. It never carries content bodies.
type StageInputBinding struct {
	InputID   string `json:"inputId"`
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
}

// StageGate reports one pinned human gate and whether its explicit human fact
// is already recorded. Planning only reads it; it never satisfies a gate.
type StageGate struct {
	Kind      string `json:"kind"`
	Timing    string `json:"timing"`
	Satisfied bool   `json:"satisfied"`
}

// StagePlanningContext is the read-only view of one configured Execution that
// stage planning needs: the exact pinned definition and bound stage inputs.
type StagePlanningContext struct {
	ExecutionID       string
	ExecutionRevision uint64
	ProjectID         string
	RepositoryKey     string
	Binding           BindingView
	Document          workflowdefinition.Document
	Stage             workflowdefinition.Stage
	StageOrdinal      int
	Inputs            []StageInputBinding
	MissingInputs     []string
	Gates             []StageGate
}

// StagePlanningContext loads the exact Execution, verifies its expected
// revision and retained binding, and binds the requested stage's inputs. It
// never saves state, resolves the current Project selection, or reads a
// substitute definition: the retained snapshot is the only source.
func (s Service) StagePlanningContext(ctx context.Context, target Target, expectedRevision uint64, stageID string) (StagePlanningContext, Result) {
	if err := ctx.Err(); err != nil {
		return StagePlanningContext{}, result(Interrupted, "workflow_cancelled", State{})
	}
	if target.ExecutionID == "" || expectedRevision == 0 || !workflowdefinition.ValidKey(stageID) {
		return StagePlanningContext{}, result(ValidationFailed, "invalid_execution_input", State{})
	}
	state, failed := s.load(ctx, target)
	if failed.Category != "" {
		return StagePlanningContext{}, failed
	}
	if _, err := DeriveLifecycle(state); err != nil {
		return StagePlanningContext{}, result(Failed, "recovery_required", state)
	}
	if state.Binding == nil {
		// Historical format-1 Executions have no pinned definition to compile;
		// no binding or default workflow is invented for them.
		return StagePlanningContext{}, result(ValidationFailed, "configured_binding_required", state)
	}
	doc, ok := bindingDocument(state)
	if !ok {
		return StagePlanningContext{}, result(Failed, "recovery_required", state)
	}
	if state.Revision != expectedRevision {
		return StagePlanningContext{}, result(Denied, "stale_execution_revision", state)
	}
	ordinal := -1
	for index, candidate := range doc.Definition.Stages {
		if candidate.ID == stageID {
			ordinal = index
		}
	}
	if ordinal < 0 {
		return StagePlanningContext{}, result(ValidationFailed, "stage_not_found", state)
	}
	if state.Status == ExecutionCompleted || state.Terminal != nil || ordinal < state.StageOrdinal {
		return StagePlanningContext{}, result(ValidationFailed, "stage_not_plannable", state)
	}
	facts, _, valid := configuredFacts(state)
	if !valid {
		return StagePlanningContext{}, result(Failed, "recovery_required", state)
	}
	stage := doc.Definition.Stages[ordinal]
	planning := StagePlanningContext{ExecutionID: state.ExecutionID, ExecutionRevision: state.Revision, ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, Binding: *InspectBinding(state), Document: doc, Stage: stage, StageOrdinal: ordinal, Inputs: []StageInputBinding{}, MissingInputs: []string{}, Gates: []StageGate{}}
	for _, input := range stage.Inputs {
		binding, bound := stageInputBinding(state, input)
		if bound {
			planning.Inputs = append(planning.Inputs, binding)
		} else if input.Required {
			planning.MissingInputs = append(planning.MissingInputs, input.ID)
		}
	}
	for _, gate := range stage.HumanGates {
		// Facts are recorded only for the current stage's gate; a later stage
		// reusing a gate kind still needs its own explicit human decision.
		satisfied := ordinal == state.StageOrdinal && facts[LifecycleFactKind(gate.Kind)]
		planning.Gates = append(planning.Gates, StageGate{Kind: gate.Kind, Timing: gate.Timing, Satisfied: satisfied})
	}
	if len(planning.MissingInputs) != 0 {
		return planning, result(ValidationFailed, "stage_prerequisite_missing", state)
	}
	return planning, result(Succeeded, "stage_planning_context_ready", state)
}

// stageInputBinding uses only retained, already validated facts. Artifact
// inputs have no authoritative pre-execution source and stay unbound.
func stageInputBinding(state State, input workflowdefinition.Input) (StageInputBinding, bool) {
	binding := StageInputBinding{InputID: input.ID, Kind: input.Kind, Reference: input.Source}
	switch input.Kind {
	case "project-context":
		c, ok := state.Binding.Contexts[input.Source]
		binding.Digest = c.Digest
		return binding, ok && validDigest(c.Digest)
	case "stage-output":
		ref, ok := priorOutput(state, input.Source)
		binding.Digest = ref.Digest
		return binding, ok && validDigest(ref.Digest)
	case "work-item":
		wire, err := json.Marshal(struct {
			Provider   string `json:"provider"`
			Resource   string `json:"resource"`
			ExternalID string `json:"externalId"`
		}{state.WorkItem.Provider, state.WorkItem.Resource, state.WorkItem.ExternalID})
		if err != nil {
			return StageInputBinding{}, false
		}
		binding.Digest = digestBytes(wire)
		return binding, true
	}
	return StageInputBinding{}, false
}
