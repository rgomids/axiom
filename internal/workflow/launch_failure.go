package workflow

import (
	"slices"
	"time"
)

// LaunchFailure contains only closed, redacted facts, never an underlying error.
type LaunchFailure struct {
	Reason string `json:"reason"`
	Phase  string `json:"phase"`
	Source string `json:"source,omitempty"`
}

func (f LaunchFailure) Valid() bool {
	return slices.Contains([]string{"configuration_invalid", "selector_conflict", "configuration_unsafe", "path_unsafe", "permission_denied", "identity_unsafe", "executable_unavailable", "builder_failed"}, f.Reason) &&
		slices.Contains([]string{"invocation_build", "source_resolution", "selector_inspection", "spawn", "runtime"}, f.Phase) &&
		slices.Contains([]string{"", "user", "project", "project-local", "managed", "managed-drop-in", "argv", "environment"}, f.Source)
}

func (f LaunchFailure) NextAction() string {
	switch f.Reason {
	case "configuration_invalid":
		return "Correct recognized Runtime configuration syntax; retry explicitly."
	case "selector_conflict":
		return "Remove competing Runtime selectors; retry explicitly with the frozen selection."
	case "configuration_unsafe":
		return "Inspect recognized Runtime configuration sources and their safety; retry explicitly."
	case "path_unsafe", "permission_denied", "identity_unsafe", "executable_unavailable":
		return "Inspect configured executable and local path safety; retry explicitly."
	default:
		return "Inspect local invocation inputs; retry explicitly."
	}
}

func (e *Execution) MarkLaunchFailed(id string, generation uint64, failure LaunchFailure, now time.Time) error {
	if !failure.Valid() {
		return ErrTransition
	}
	a, err := e.fencedAttempt(id, generation, AttemptPrepared)
	if err != nil {
		return err
	}
	a.LaunchFailure = &failure
	return e.MarkSpawnFailed(id, generation, "launch_refused: "+failure.Reason, now)
}
