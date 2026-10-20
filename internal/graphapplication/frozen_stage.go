package graphapplication

import (
	"context"
	"errors"
	"fmt"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// SubscriptionAuthenticator is the machine-local CLI subscription
// authentication preflight (runtimeadapter.AuthPreflight).
type SubscriptionAuthenticator interface {
	Check(context.Context, runtimeadapter.AuthTarget) runtimeadapter.AuthReport
}

// EffortSourceInspector re-reads the effort sources of the exact invocation.
// It returns a safe canonical source digest, never source content/paths.
// Unknown/unreadable sources must return an error.
type EffortSourceInspector interface {
	CheckEffortSources(context.Context, runtimeprofile.ResolvedProfileV4, executiongraph.Invocation) (string, error)
}

// StageBindingChecker is implemented by runtimeapplication.Service over the
// operational observer. Check reloads local configuration, the executable and
// effort sources on each attempt, including removal, tombstones and disabled
// Runtimes.
type StageBindingChecker interface {
	Check(context.Context, runtimeapplication.Preview) (runtimeapplication.Binding, error)
}

// PrepareFrozenStageInvocation is the per-attempt composition. The caller
// supplies the persisted tuple and its reviewed preview; neither is replaced by
// the current selection. Authentication is observed on the exact invocation;
// an optional version is never compared. No process is started here.
func PrepareFrozenStageInvocation(ctx context.Context, request runtimeadapter.StageInvocationRequest, frozen runtimeprofile.ResolvedProfileV4, preview runtimeapplication.Preview, checker StageBindingChecker, sources EffortSourceInspector, auth SubscriptionAuthenticator) (executiongraph.Invocation, runtimeadapter.AuthReport, error) {
	report := runtimeadapter.AuthReport{}
	if err := ctx.Err(); err != nil {
		return executiongraph.Invocation{}, report, err
	}
	if runtimeprofile.ValidateResolvedProfileV4(frozen) != nil || checker == nil || preview.Choice == nil || preview.Choice.FrozenProfile == nil || *preview.Choice.FrozenProfile != frozen {
		return executiongraph.Invocation{}, report, runtimeapplication.ErrStale
	}
	binding, err := checker.Check(ctx, preview)
	if err != nil {
		return executiongraph.Invocation{}, report, err
	}
	if binding.Choice.FrozenProfile == nil || *binding.Choice.FrozenProfile != frozen {
		return executiongraph.Invocation{}, report, runtimeapplication.ErrStale
	}
	if binding.CredentialReference != "" || request.Profile.CredentialReference != "" || auth == nil {
		return executiongraph.Invocation{}, report, executiongraph.ErrAuthenticationBlocked
	}
	inv, err := runtimeadapter.BuildFrozenStageInvocation(request, frozen)
	if err != nil {
		if errors.Is(err, runtimeadapter.ErrInvalidAdapterConfiguration) {
			return executiongraph.Invocation{}, report, runtimeapplication.ErrStale
		}
		return executiongraph.Invocation{}, report, stageEffortError(err)
	}
	if sources == nil {
		return executiongraph.Invocation{}, report, runtimeadapter.ErrEffortConfigurationUnsafe
	}
	if err = checkFrozenSources(ctx, sources, frozen, inv); err != nil {
		return executiongraph.Invocation{}, report, err
	}
	report = auth.Check(ctx, runtimeadapter.AuthTarget{RuntimeID: frozen.RuntimeID, Executable: inv.Argv[0], ExpectedDigest: frozen.ExecutableDigest, WorkingDirectory: inv.CWD, Environment: executiongraph.EffectiveInvocationEnvironment(inv)})
	if report.Reason == "executable_identity_changed" {
		return executiongraph.Invocation{}, report, fmt.Errorf("%w: %w", executiongraph.ErrAuthenticationBlocked, runtimeapplication.ErrStale)
	}
	if report.RuntimeID != frozen.RuntimeID || !report.DispatchAllowed() || report.ExecutableDigest != frozen.ExecutableDigest {
		return executiongraph.Invocation{}, report, fmt.Errorf("%w: %s", executiongraph.ErrAuthenticationBlocked, report.Reason)
	}
	// Authentication may observe changed local state; the last source check still
	// compares against the frozen digest, never approves a replacement.
	if err = checkFrozenSources(ctx, sources, frozen, inv); err != nil {
		return executiongraph.Invocation{}, report, err
	}
	return inv, report, ctx.Err()
}

func checkFrozenSources(ctx context.Context, sources EffortSourceInspector, frozen runtimeprofile.ResolvedProfileV4, inv executiongraph.Invocation) error {
	digest, err := sources.CheckEffortSources(ctx, frozen, inv)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return stageEffortError(err)
	}
	if digest != frozen.EffortSourceDigest {
		return runtimeapplication.ErrStale
	}
	return nil
}

// Inspectors expose safe categories only; opaque source failures cannot leak raw
// configuration, paths or environment through dispatch errors.
func stageEffortError(err error) error {
	switch {
	case errors.Is(err, runtimeadapter.ErrEffortConfigurationInvalid):
		return runtimeadapter.ErrEffortConfigurationInvalid
	case errors.Is(err, runtimeadapter.ErrEffortOverrideConflict):
		return runtimeadapter.ErrEffortOverrideConflict
	case errors.Is(err, runtimeadapter.ErrExecutableUnavailable):
		return runtimeadapter.ErrExecutableUnavailable
	}
	return runtimeadapter.ErrEffortConfigurationUnsafe
}
