package runtimeadapter

import (
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

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
