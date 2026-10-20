package stageexecution

import (
	"errors"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

var (
	ErrLegacyExecution           = errors.New("legacy_execution_contract")
	ErrProfileEffortUnconfigured = errors.New("profile_effort_unconfigured")
	ErrPolicyUpdateRequired      = errors.New("project_policy_update_required")
	ErrAdmission                 = errors.New("invalid_stage_contract")
)

type Operation string

const (
	StagePlan        Operation = "stage.plan"
	StageRun         Operation = "stage.run"
	StageRetry       Operation = "stage.retry"
	StageResume      Operation = "stage.resume"
	StageCancel      Operation = "stage.cancel"
	StageReconcile   Operation = "stage.reconcile"
	StageIntegrate   Operation = "stage.integrate"
	StageStatus      Operation = "stage.status"
	CoordinationRead Operation = "coordination.read"
)

// Contract contains reader-validated snapshots: the inspected local record
// state (as local.RuntimeProfileStore.Inspect reports it) and, for an
// active-v4 record, its canonical configuration. Admission never migrates
// them, resolves another Profile or grants effect authority. Current Profile
// availability is derived from the configuration, never supplied.
type Contract struct {
	Definition       workflowdefinition.Document
	Execution        workflow.State
	Graph            *executiongraph.GraphDocument
	Record           string
	Configuration    *runtimeprofile.ConfigurationV4
	WorkflowSelected bool
	PolicyAllowed    bool
}

// Admission reports the current intent of the frozen Profile. It is not a
// claim of vendor support.
type Admission struct {
	Next    string
	Profile *runtimeprofile.ProfileIntent
}

// Admit is the Stage admission boundary. Only canonical configuration 4 and
// a proof-free Execution 5 binding admit planning or dispatch (AXM-36 D2).
// A historical Execution remains inspectable, cancellable and reconcilable
// without dispatch rights. A graph is optional for planning, publication and
// genuine single stages; any supplied graph has already passed the one
// canonical versioned codec.
func Admit(operation Operation, c Contract) (Admission, error) {
	switch operation {
	case StagePlan, StageRun, StageRetry, StageResume, StageCancel, StageReconcile, StageIntegrate, StageStatus, CoordinationRead:
	default:
		return Admission{}, ErrAdmission
	}
	if c.Graph != nil {
		if _, err := executiongraph.EncodeGraphDocument(*c.Graph); err != nil {
			return Admission{}, executiongraph.ErrUnsupportedGraphFormat
		}
	}
	current := c.Execution.FormatVersion == workflow.OperationalConfiguredFormatVersion
	historical := c.Execution.FormatVersion == workflow.ProfileConfiguredFormatVersion
	if operation == StageStatus || operation == CoordinationRead {
		if !current {
			return Admission{Next: ErrLegacyExecution.Error()}, nil
		}
		if c.Graph != nil && c.Graph.Historical != nil {
			return Admission{Next: executiongraph.ErrUnsupportedGraphFormat.Error()}, nil
		}
		return Admission{}, nil
	}
	if c.Graph != nil && c.Graph.Successor == nil {
		return Admission{}, executiongraph.ErrUnsupportedGraphFormat
	}
	if historical && (operation == StageCancel || operation == StageReconcile) {
		return Admission{Next: ErrLegacyExecution.Error()}, nil
	}
	if operation != StagePlan && !current {
		return Admission{}, ErrLegacyExecution
	}
	if c.Graph != nil && c.Graph.Successor.FormatVersion != executiongraph.OperationalGraphVersion {
		return Admission{}, executiongraph.ErrUnsupportedGraphFormat
	}
	if c.Definition.Definition.SchemaVersion == 1 || !c.WorkflowSelected {
		return Admission{}, workflowcompiler.ErrWorkflowMigrationRequired
	}
	if c.Definition.Definition.SchemaVersion != 2 {
		return Admission{}, ErrAdmission
	}
	binding := c.Execution.Binding
	if operation != StagePlan && (binding == nil || binding.SchemaVersion != 2 || binding.FrozenProfile == nil || runtimeprofile.ValidateResolvedProfileV4(*binding.FrozenProfile) != nil) {
		return Admission{}, ErrLegacyExecution
	}
	switch c.Record {
	case "active-v4":
	case "fresh", "retired", "active-v1", "active-v2", "active-v3":
		return Admission{}, ErrProfileEffortUnconfigured
	default:
		return Admission{}, ErrAdmission
	}
	if c.Configuration == nil {
		return Admission{}, ErrAdmission
	}
	consumer, err := runtimeprofile.NewConsumerV4(*c.Configuration)
	if err != nil {
		return Admission{}, ErrAdmission
	}
	// A removed, disallowed or disabled frozen Profile blocks; nothing falls
	// back to another Profile, model, Runtime or effort.
	if binding == nil || binding.FrozenProfile == nil {
		return Admission{}, ErrProfileEffortUnconfigured
	}
	intent, err := consumer.ProfileIntent(binding.FrozenProfile.RuntimeID, binding.FrozenProfile.ProfileID)
	if err != nil {
		return Admission{}, ErrProfileEffortUnconfigured
	}
	if !c.PolicyAllowed {
		return Admission{}, ErrPolicyUpdateRequired
	}
	return Admission{Profile: &intent}, nil
}
