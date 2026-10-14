package runtimeadapter

import (
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// BuildFrozenStageInvocation derives effort exclusively from the approved tuple.
// Fresh proof, local configuration and effort-source checks belong to the
// dispatch composition; this builder never resolves or substitutes a Profile.
func BuildFrozenStageInvocation(request StageInvocationRequest, frozen runtimeprofile.ResolvedProfile) (executiongraph.Invocation, error) {
	if runtimeprofile.ValidateResolvedProfile(frozen) != nil || request.Profile.RuntimeID != frozen.RuntimeID || request.Profile.ModelProfileID != frozen.ProfileID || request.Profile.Model != frozen.Model || request.Version != frozen.RuntimeVersion {
		return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
	}
	inv, err := BuildStageInvocation(request)
	if err != nil {
		return executiongraph.Invocation{}, err
	}
	return withStageEffort(inv, frozen.Effort)
}

// withStageEffort is the adapter's single encoding of Model Profile effort
// intent onto a stage invocation (ADR-0024).
func withStageEffort(inv executiongraph.Invocation, effort runtimeprofile.Effort) (executiongraph.Invocation, error) {
	if effort.Mode == runtimeprofile.EffortExplicit {
		switch inv.RuntimeID {
		case "codex":
			inv.Argv = append(inv.Argv, "--config", "model_reasoning_effort="+effort.Value)
		case "claude":
			inv = executiongraph.WithExactEnvironment(inv, append(inv.Env, "CLAUDE_CODE_EFFORT_LEVEL="+effort.Value))
		default:
			return executiongraph.Invocation{}, ErrInvalidAdapterConfiguration
		}
	}
	return inv, nil
}
