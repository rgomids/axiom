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

// Contract contains reader-validated snapshots: the inspected local record
// state and its canonical versioned configuration. Admission never migrates
// them, resolves another Profile, acquires proof or grants effect authority.
// Current Profile availability is derived from ConsumerV3, never supplied.
type Contract struct {
	Definition       workflowdefinition.Document
	Execution        workflow.State
	Graph            *executiongraph.GraphDocument
	Record           runtimeprofile.RecordStateV3
	Configuration    runtimeprofile.VersionedConfiguration
	WorkflowSelected bool
	PolicyAllowed    bool
}
type Admission struct{ Next string }

// Admit is the Stage admission boundary. Since BLD-001B.3c-2b.3 it is AdmitV3:
// only canonical format 3 is current configuration. A graph is optional for
// planning, publication and genuine single stages; any supplied graph has
// already passed the one canonical versioned codec.
func Admit(operation Operation, c Contract) (Admission, error) {
	admission, err := AdmitV3(operation, c)
	return admission.Admission, err
}

// admitExecution applies the configuration-independent operation, graph,
// historical inspection, Workflow and frozen binding checks. done reports a
// final decision; otherwise the caller checks current Profile configuration.
func admitExecution(operation Operation, c Contract) (Admission, bool, error) {
	inspection := operation == StageStatus || operation == CoordinationRead
	switch operation {
	case StagePlan, StageRun, StageRetry, StageResume, StageCancel, StageReconcile, StageIntegrate, StageStatus, CoordinationRead, CapabilityProbe:
	default:
		return Admission{}, true, ErrAdmission
	}
	if c.Graph != nil {
		if _, err := executiongraph.EncodeGraphDocument(*c.Graph); err != nil {
			return Admission{}, true, executiongraph.ErrUnsupportedGraphFormat
		}
		if !inspection && c.Graph.Successor == nil {
			return Admission{}, true, executiongraph.ErrUnsupportedGraphFormat
		}
	}
	if inspection {
		if c.Execution.FormatVersion != workflow.ProfileConfiguredFormatVersion {
			return Admission{Next: ErrLegacyExecution.Error()}, true, nil
		}
		if c.Graph != nil && c.Graph.Historical != nil {
			return Admission{Next: executiongraph.ErrUnsupportedGraphFormat.Error()}, true, nil
		}
		return Admission{}, true, nil
	}
	if operation != CapabilityProbe {
		if operation != StagePlan && c.Execution.FormatVersion != workflow.ProfileConfiguredFormatVersion {
			return Admission{}, true, ErrLegacyExecution
		}
		if c.Definition.Definition.SchemaVersion == 1 || !c.WorkflowSelected {
			return Admission{}, true, workflowcompiler.ErrWorkflowMigrationRequired
		}
		if c.Definition.Definition.SchemaVersion != 2 {
			return Admission{}, true, ErrAdmission
		}
		if operation != StagePlan && (c.Execution.Binding == nil || c.Execution.Binding.SchemaVersion != 2 || c.Execution.Binding.ResolvedProfile == nil || runtimeprofile.ValidateResolvedProfile(*c.Execution.Binding.ResolvedProfile) != nil) {
			return Admission{}, true, ErrLegacyExecution
		}
	}
	return Admission{}, false, nil
}

// ContractV3 is the format-3 admission input (Specification 007 AXM-22
// BLD-001B.3c-2b.2), now the only Contract.
type ContractV3 = Contract

// AdmissionV3 reports the current intent of the frozen Profile: identity is
// DigestV3 and the unchanged ProfileDigest. It is not capability proof.
type AdmissionV3 struct {
	Admission
	Profile *runtimeprofile.ProfileIntentV3
}

// AdmitV3 is the separately invoked format-3 admission evaluator. Execution,
// graph and historical inspection checks are Admit's own; only the current
// configuration check reads ConsumerV3. Missing, retired and historical
// format-1/2 records are profile_effort_unconfigured; an invalid record is
// invalid_stage_contract.
func AdmitV3(operation Operation, c ContractV3) (AdmissionV3, error) {
	if admission, done, err := admitExecution(operation, c); done {
		return AdmissionV3{Admission: admission}, err
	}
	var consumer runtimeprofile.ConsumerV3
	var err error
	if c.Record == runtimeprofile.RecordActiveV3 {
		consumer, err = runtimeprofile.ConsumeVersioned(c.Configuration)
	} else {
		consumer, err = runtimeprofile.NewConsumerV3(c.Record, c.Configuration.V3)
	}
	switch {
	case errors.Is(err, runtimeprofile.ErrConfigurationMissingV3), errors.Is(err, runtimeprofile.ErrConfigurationRetiredV3), errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat):
		return AdmissionV3{}, ErrProfileEffortUnconfigured
	case err != nil:
		return AdmissionV3{}, ErrAdmission
	}
	// A removed, disallowed or disabled frozen Profile blocks; nothing falls
	// back to another Profile, model, Runtime or effort.
	binding := c.Execution.Binding
	if binding == nil || binding.ResolvedProfile == nil {
		return AdmissionV3{}, ErrProfileEffortUnconfigured
	}
	intent, err := consumer.ProfileIntent(binding.ResolvedProfile.RuntimeID, binding.ResolvedProfile.ProfileID)
	if err != nil {
		return AdmissionV3{}, ErrProfileEffortUnconfigured
	}
	if !c.PolicyAllowed {
		return AdmissionV3{}, ErrPolicyUpdateRequired
	}
	return AdmissionV3{Profile: &intent}, nil
}
