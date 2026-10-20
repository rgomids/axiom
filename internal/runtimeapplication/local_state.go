package runtimeapplication

import "errors"

// LocalStateError communicates safe canonical state, never paths or source text.
type LocalStateError struct{ State string }

func (e LocalStateError) Error() string { return "local runtime configuration unavailable" }
func localStateCategory(err error) string {
	var state LocalStateError
	if errors.As(err, &state) {
		switch state.State {
		case "fresh":
			return "configuration_missing"
		case "retired":
			return "configuration_retired"
		case "pending":
			return "configuration_recovery_required"
		case "unsafe":
			return "configuration_unsafe"
		case "active-v1", "active-v2", "active-v3":
			return "unsupported_configuration_format"
		}
	}
	return "policy_unavailable"
}
