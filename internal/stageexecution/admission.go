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
	CapabilityProbe  Operation = "capability.probe"
)

// Contract contains reader-validated snapshots. Admission never migrates them,
// resolves another Profile, acquires proof or grants effect authority.
type Contract struct {
	Definition       workflowdefinition.Document
	Execution        workflow.State
	Configuration    runtimeprofile.Configuration
	Graph            *executiongraph.GraphDocument
	WorkflowSelected bool
	ProfileAvailable bool
	PolicyAllowed    bool
}
type Admission struct{ Next string }

func Admit(operation Operation, c Contract) (Admission, error) {
	inspection := operation == StageStatus || operation == CoordinationRead
	switch operation {
	case StagePlan, StageRun, StageRetry, StageResume, StageCancel, StageReconcile, StageIntegrate, StageStatus, CoordinationRead, CapabilityProbe:
	default:
		return Admission{}, ErrAdmission
	}
	if c.Graph != nil {
		if _, err := executiongraph.EncodeGraphDocument(*c.Graph); err != nil {
			return Admission{}, executiongraph.ErrUnsupportedGraphFormat
		}
		if !inspection && c.Graph.Successor == nil {
			return Admission{}, executiongraph.ErrUnsupportedGraphFormat
		}
	}
	if inspection {
		if c.Execution.FormatVersion != workflow.ProfileConfiguredFormatVersion {
			return Admission{Next: ErrLegacyExecution.Error()}, nil
		}
		if c.Graph != nil && c.Graph.Historical != nil {
			return Admission{Next: executiongraph.ErrUnsupportedGraphFormat.Error()}, nil
		}
		return Admission{}, nil
	}
	if operation != CapabilityProbe {
		if operation != StagePlan && c.Execution.FormatVersion != workflow.ProfileConfiguredFormatVersion {
			return Admission{}, ErrLegacyExecution
		}
		if c.Definition.Definition.SchemaVersion == 1 || !c.WorkflowSelected {
			return Admission{}, workflowcompiler.ErrWorkflowMigrationRequired
		}
		if c.Definition.Definition.SchemaVersion != 2 {
			return Admission{}, ErrAdmission
		}
		if operation != StagePlan && (c.Execution.Binding == nil || c.Execution.Binding.SchemaVersion != 2 || c.Execution.Binding.ResolvedProfile == nil || runtimeprofile.ValidateResolvedProfile(*c.Execution.Binding.ResolvedProfile) != nil) {
			return Admission{}, ErrLegacyExecution
		}
	}
	if c.Configuration.FormatVersion == runtimeprofile.FormatVersion || !c.ProfileAvailable {
		return Admission{}, ErrProfileEffortUnconfigured
	}
	if runtimeprofile.Validate(c.Configuration) != nil || c.Configuration.FormatVersion != runtimeprofile.FormatVersionV2 {
		return Admission{}, ErrAdmission
	}
	if !c.PolicyAllowed {
		return Admission{}, ErrPolicyUpdateRequired
	}
	// A graph is optional for planning, publication and genuine single stages.
	// Any supplied graph has already passed the one canonical versioned codec.
	return Admission{}, nil
}
