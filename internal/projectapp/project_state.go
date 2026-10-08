package projectapp

import "context"

// Issue #230 (I230-T03) discoverable machine-local Project status. A Project
// is discoverable as active or archived only from a valid operational record;
// a malformed, unknown-version, unsafe or interrupted record is reported as
// invalid or recovery_required and is never collapsed into active.
const (
	ProjectInvalid          ProjectStatus = "invalid"
	ProjectRecoveryRequired ProjectStatus = "recovery_required"
)

// ProjectLocalState is the status of one installed Project on this machine.
// Operational is set only when the record was read and is valid.
type ProjectLocalState struct {
	Status      ProjectStatus
	Operational *OperationalState
}

// InspectProjectState reads the operational record and maps it to the
// discoverable status. A non-empty category reports a failure the caller must
// surface instead of guessing a status (cancellation, storage failure).
func InspectProjectState(ctx context.Context, store OperationalStore, projectID string) (ProjectLocalState, string) {
	observation, category := InspectOperational(ctx, store, projectID)
	switch category {
	case "":
		state := observation.State.clone()
		return ProjectLocalState{Status: state.ProjectStatus, Operational: &state}, ""
	case OperationalStateInvalid, OperationalNotInstalled:
		return ProjectLocalState{Status: ProjectInvalid}, ""
	case OperationalRecoveryRequired:
		return ProjectLocalState{Status: ProjectRecoveryRequired}, ""
	}
	return ProjectLocalState{}, category
}
