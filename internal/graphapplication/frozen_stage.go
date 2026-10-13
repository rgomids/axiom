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

// StageBindingChecker is implemented by runtimeapplication.Service with the
// capabilityproof.Observer. Check reloads local configuration and exact model
// proof on each attempt, including removal, tombstones and disabled Runtimes.
type StageBindingChecker interface {
	Check(context.Context, runtimeapplication.Preview) (runtimeapplication.Binding, error)
}

// PrepareFrozenStageInvocation is the per-attempt #276 composition. The caller
// supplies the persisted tuple and its reviewed preview; neither is replaced by
// the current selection. No process is started here.
func PrepareFrozenStageInvocation(ctx context.Context, request runtimeadapter.StageInvocationRequest, frozen runtimeprofile.ResolvedProfile, preview runtimeapplication.Preview, checker StageBindingChecker, sources EffortSourceInspector, auth SubscriptionAuthenticator) (executiongraph.Invocation, runtimeadapter.AuthReport, error) {
	report := runtimeadapter.AuthReport{}
	if err := ctx.Err(); err != nil {
		return executiongraph.Invocation{}, report, err
	}
	if runtimeprofile.ValidateResolvedProfile(frozen) != nil || checker == nil || preview.Choice == nil || preview.Choice.ResolvedProfile == nil || *preview.Choice.ResolvedProfile != frozen {
		return executiongraph.Invocation{}, report, runtimeapplication.ErrStale
	}
	binding, err := checker.Check(ctx, preview)
	if err != nil {
		return executiongraph.Invocation{}, report, err
	}
	if binding.Choice.ResolvedProfile == nil || *binding.Choice.ResolvedProfile != frozen {
		return executiongraph.Invocation{}, report, runtimeapplication.ErrStale
	}
	if binding.CredentialReference != "" || request.Profile.CredentialReference != "" || auth == nil {
		return executiongraph.Invocation{}, report, executiongraph.ErrAuthenticationBlocked
	}
	inv, err := runtimeadapter.BuildFrozenStageInvocation(request, frozen)
	if err != nil {
		return executiongraph.Invocation{}, report, err
	}
	if sources == nil {
		return executiongraph.Invocation{}, report, runtimeadapter.ErrEffortConfigurationUnproven
	}
	digest, err := sources.CheckEffortSources(ctx, frozen, inv)
	if err != nil {
		return executiongraph.Invocation{}, report, stageEffortError(err)
	}
	if digest != frozen.EffortSourceDigest {
		return executiongraph.Invocation{}, report, runtimeapplication.ErrStale
	}
	report = auth.Check(ctx, runtimeadapter.AuthTarget{RuntimeID: frozen.RuntimeID, Executable: inv.Argv[0], ExpectedDigest: frozen.ExecutableDigest, WorkingDirectory: inv.CWD, Environment: executiongraph.EffectiveInvocationEnvironment(inv)})
	if report.RuntimeID != frozen.RuntimeID || !report.DispatchAllowed() || report.Version != frozen.RuntimeVersion || report.ExecutableDigest != frozen.ExecutableDigest {
		return executiongraph.Invocation{}, report, fmt.Errorf("%w: subscription_binding_unproven", executiongraph.ErrAuthenticationBlocked)
	}
	// Authentication may observe changed local state; the last source check still
	// compares against the frozen digest, never approves a replacement.
	digest, err = sources.CheckEffortSources(ctx, frozen, inv)
	if err != nil {
		return executiongraph.Invocation{}, report, stageEffortError(err)
	}
	if digest != frozen.EffortSourceDigest {
		return executiongraph.Invocation{}, report, runtimeapplication.ErrStale
	}
	return inv, report, ctx.Err()
}

// Inspectors expose safe categories only; opaque source failures cannot leak raw
// configuration, paths or environment through dispatch errors.
func stageEffortError(err error) error {
	if errors.Is(err, runtimeadapter.ErrEffortOverrideConflict) || errors.Is(err, ErrEffortOverrideConflict) {
		return runtimeadapter.ErrEffortOverrideConflict
	}
	return runtimeadapter.ErrEffortConfigurationUnproven
}
