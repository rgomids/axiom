package runtimeadapter

import (
	"context"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// RuntimeDefaultEffort is the explicit choice to send no effort selector and
// let the Runtime apply its own default; Axiom asserts no concrete level.
const RuntimeDefaultEffort = "runtime-default"

// CheckFrozenSelectors is the one effective-selector check for new Execution
// attempts (ADR-0037 §4/§5). It reuses the bounded effort-source reader to
// refuse any inherited argv, environment or consented configuration selector
// that would silently override the frozen Runtime/model/effort, with redacted
// diagnostics. It never probes authentication or predicts vendor support;
// those failures are observed when the authorized attempt runs.
func CheckFrozenSelectors(ctx context.Context, runtimeID, model, effort string, roots EffortSourceRoots, inv executiongraph.Invocation) error {
	if ctx.Err() != nil || len(inv.Argv) == 0 || (runtimeID != "codex" && runtimeID != "claude") {
		return ErrEffortConfigurationUnsafe
	}
	digest, err := ExecutableIdentity(inv.Argv[0])
	if err != nil {
		return ErrExecutableUnavailable
	}
	frozen := runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}
	if effort != RuntimeDefaultEffort {
		frozen = runtimeprofile.Effort{Mode: runtimeprofile.EffortExplicit, Value: effort}
	}
	if !runtimeprofile.SafeIdentifier(frozen.Value) {
		return ErrEffortConfigurationUnsafe
	}
	mode := "exec"
	if runtimeID == "claude" {
		mode = "print"
	}
	_, err = observeEffortFacts(ctx, effortIntent{RuntimeID: runtimeID, InvocationMode: mode, ExecutableDigest: digest, Model: model, Effort: frozen}, fixedEffortRules(runtimeID, roots), inv)
	return err
}
