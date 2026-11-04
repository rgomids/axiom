package stageexecution

import (
	"context"
	"errors"
	"os"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
)

// Causes remain in memory for Is/As; Error never formats the cause.
type launchError struct {
	failure workflow.LaunchFailure
	cause   error
}

func (e *launchError) Error() string        { return "launch_refused: " + e.failure.Reason }
func (e *launchError) Unwrap() error        { return e.cause }
func (e *launchError) Is(target error) bool { return target == ErrRuntimeLaunch }

func launchFailure(err error, phase string) workflow.LaunchFailure {
	var typed *launchError
	if errors.As(err, &typed) {
		return typed.failure
	}
	reason := "builder_failed"
	switch {
	case errors.Is(err, runtimeprofile.ErrConfigurationInvalid):
		reason = "configuration_invalid"
	case errors.Is(err, runtimeadapter.ErrEffortOverrideConflict):
		reason = "selector_conflict"
	case errors.Is(err, os.ErrPermission):
		reason = "permission_denied"
	case errors.Is(err, runtimeadapter.ErrExecutableUnavailable):
		reason = "executable_unavailable"
	case errors.Is(err, runtimeadapter.ErrEffortConfigurationUnsafe):
		reason = "configuration_unsafe"
	}
	failure := workflow.LaunchFailure{Reason: reason, Phase: phase}
	var source *runtimeadapter.SourceError
	if errors.As(err, &source) {
		failure.Source = source.Source
		if !failure.Valid() {
			failure.Source = ""
		}
	}
	return failure
}
func refuseLaunch(ctx context.Context, err error, phase, reason string) (executiongraph.Invocation, error) {
	if ctx.Err() != nil {
		return executiongraph.Invocation{}, ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return executiongraph.Invocation{}, err
	}
	failure := launchFailure(err, phase)
	if reason != "" {
		failure.Reason = reason
	}
	return executiongraph.Invocation{}, &launchError{failure: failure, cause: err}
}
