package runtimeapplication

import (
	"errors"
	"fmt"
	"slices"

	"github.com/rgomids/axiom/internal/workflow"
)

// Invocation resolution for new Execution attempts (ADR-0037 §4, Spec 015
// F5). Per field: explicit run override > explicit Workflow agent value >
// explicitly configured machine-local default. A Profile at a level supplies
// defaults for that level only; explicit fields at the same level replace
// them. Nothing is ranked, inferred from complexity or substituted.

var (
	ErrInvocationMissing    = errors.New("invocation_selection_missing")
	ErrInvocationConflict   = errors.New("invocation_selection_conflict")
	ErrRuntimeNotPermitted  = errors.New("runtime_not_permitted")
	ErrRuntimeNotConfigured = errors.New("runtime_not_configured")
)

// Invocation fields in their canonical order.
const (
	FieldRuntime      = "runtime"
	FieldModel        = "model"
	FieldEffort       = "effort"
	FieldInstructions = "instructions"
)

var invocationFieldOrder = []string{FieldRuntime, FieldModel, FieldEffort, FieldInstructions}

// ProfileInput is an optional reusable tuple. Digest identifies the exact
// Profile bytes the values came from.
type ProfileInput struct {
	ID, Digest                           string
	Runtime, Model, Effort, Instructions string
}

func (p ProfileInput) value(field string) string {
	return map[string]string{FieldRuntime: p.Runtime, FieldModel: p.Model, FieldEffort: p.Effort, FieldInstructions: p.Instructions}[field]
}

// Assignment is one explicit field value from a source at a level (a flag, a
// Workflow agent member or a local default entry).
type Assignment struct{ Field, Value string }

// InvocationLayer is one precedence level.
type InvocationLayer struct {
	Profile     *ProfileInput
	Assignments []Assignment
}

type InvocationRequest struct {
	RunOverride   InvocationLayer
	WorkflowAgent InvocationLayer
	LocalDefault  InvocationLayer
	// Executables maps configured Runtime identities to their machine-local
	// executable reference; an unconfigured Runtime refuses.
	Executables map[string]string
	// PermittedRuntimes is the explicit user restriction; empty means none.
	PermittedRuntimes []string
}

type layerSource struct {
	name  string
	layer InvocationLayer
}

// explicit returns the single explicit value of field at one level.
// Distinct values at the same level conflict.
func (l InvocationLayer) explicit(field string) (string, bool, error) {
	value, found := "", false
	for _, a := range l.Assignments {
		if !slices.Contains(invocationFieldOrder, a.Field) {
			return "", false, fmt.Errorf("%w: unknown field %q", ErrInvocationConflict, a.Field)
		}
		if a.Field != field {
			continue
		}
		if found && a.Value != value {
			return "", false, fmt.Errorf("%w: %s has different values at the same level", ErrInvocationConflict, field)
		}
		value, found = a.Value, true
	}
	return value, found, nil
}

// ResolveInvocation freezes the effective tuple and its provenance. The
// result is the attempt's sole Runtime/model/effort/instructions authority.
func ResolveInvocation(r InvocationRequest) (workflow.Invocation, error) {
	levels := []layerSource{{"run_override", r.RunOverride}, {"workflow_agent", r.WorkflowAgent}, {"local_default", r.LocalDefault}}
	values := map[string]string{}
	var provenance []workflow.FieldProvenance
	var profile *ProfileInput
	for _, field := range invocationFieldOrder {
		resolved := false
		for _, level := range levels {
			value, ok, err := level.layer.explicit(field)
			if err != nil {
				return workflow.Invocation{}, err
			}
			source := workflow.FieldProvenance{Field: field, Source: level.name}
			if !ok && level.layer.Profile != nil && level.layer.Profile.value(field) != "" {
				value, ok = level.layer.Profile.value(field), true
				source.Profile = level.layer.Profile.ID
				if profile == nil {
					profile = level.layer.Profile
				}
			}
			if !ok {
				continue
			}
			if value == "" {
				return workflow.Invocation{}, fmt.Errorf("%w: %s is empty at %s", ErrInvocationMissing, field, level.name)
			}
			values[field] = value
			provenance = append(provenance, source)
			resolved = true
			break
		}
		if !resolved {
			// Missing effort is never an implicit runtime-default.
			return workflow.Invocation{}, fmt.Errorf("%w: %s; select it explicitly for the run, the Workflow agent or a local default", ErrInvocationMissing, field)
		}
	}
	for _, level := range levels {
		if p := level.layer.Profile; p != nil && (!workflow.ValidToken(p.ID) || p.Digest == "") {
			return workflow.Invocation{}, fmt.Errorf("%w: profile reference at %s lacks identity", ErrInvocationConflict, level.name)
		}
	}
	runtime := values[FieldRuntime]
	if len(r.PermittedRuntimes) != 0 && !slices.Contains(r.PermittedRuntimes, runtime) {
		return workflow.Invocation{}, fmt.Errorf("%w: %s", ErrRuntimeNotPermitted, runtime)
	}
	executable := r.Executables[runtime]
	if executable == "" {
		return workflow.Invocation{}, fmt.Errorf("%w: %s", ErrRuntimeNotConfigured, runtime)
	}
	inv := workflow.Invocation{
		Runtime: runtime, Executable: executable, Model: values[FieldModel], Effort: values[FieldEffort],
		Instructions: values[FieldInstructions], InstructionsDigest: workflow.DigestText([]byte(values[FieldInstructions])),
		Provenance: provenance,
	}
	if profile != nil {
		inv.Profile, inv.ProfileDigest = profile.ID, profile.Digest
	}
	if !workflow.ValidToken(inv.Runtime) || !workflow.ValidToken(inv.Effort) || len(inv.Model) > 256 || len(inv.Instructions) > workflow.MaxInstructionBytes {
		return workflow.Invocation{}, fmt.Errorf("%w: unsafe selector value", ErrInvocationConflict)
	}
	return inv, nil
}
