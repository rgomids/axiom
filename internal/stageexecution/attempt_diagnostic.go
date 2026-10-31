package stageexecution

import (
	"context"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// PreSpawnDiagnostic reports a typed operational refusal with no child effect.
func PreSpawnDiagnostic(err error, operation string) *executiongraph.AttemptDiagnostic {
	if errors.Is(err, runtimeapplication.ErrStale) {
		return nil
	}
	code := ""
	switch {
	case errors.Is(err, executiongraph.ErrAuthenticationBlocked):
		code = "authentication_unavailable"
	case errors.Is(err, runtimeadapter.ErrExecutableUnavailable):
		code = "executable_unavailable"
	case errors.Is(err, runtimeadapter.ErrEffortOverrideConflict):
		code = "effort_override_conflict"
	case errors.Is(err, runtimeprofile.ErrConfigurationInvalid):
		code = "configuration_invalid"
	case errors.Is(err, runtimeadapter.ErrEffortConfigurationUnsafe):
		code = "configuration_unsafe"
	case errors.Is(err, os.ErrPermission):
		code = "permission_denied"
	case errors.Is(err, context.DeadlineExceeded):
		code = "attempt_timeout"
	case errors.Is(err, context.Canceled):
		code = "attempt_cancelled"
	}
	var blocker runtimeprofile.Blocker
	if errors.As(err, &blocker) {
		switch blocker.Code {
		case "executable_unavailable", "configuration_unsafe", "configuration_invalid", "effort_override_conflict":
			code = blocker.Code
		}
	}
	if code == "" {
		return nil
	}
	d := &executiongraph.AttemptDiagnostic{Operation: operation, EvidenceRefs: []string{}, EffectCertainty: "none"}
	d.Code, d.Cause, d.CauseConfidence = code, code, "confirmed"
	d.NextAction, d.RequiredAuthority = "inspect-configuration-and-preview", "configuration"
	if code == "authentication_unavailable" {
		d.NextAction = "restore-auth-and-preview"
	}
	if code == "executable_unavailable" {
		d.NextAction, d.RequiredAuthority = "inspect-executable-and-preview", "inspection"
	}
	if executiongraph.ValidateAttemptDiagnostic(*d) != nil {
		return nil
	}
	return d
}
