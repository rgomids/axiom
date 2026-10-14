package runtimeadapter

import (
	"context"
	"path/filepath"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// ObserveEffortApplicability independently observes the Runtime identity an
// authored EffortDeclaration must match (Specification 007 BI-3.2 item 5),
// through the same passive seams as BindingSource: LookPath, canonical
// ExecutableIdentity and the version observer keyed by that identity, with the
// executable re-read for drift. It runs nothing itself and proves no
// capability; any absent or drifted identity is effort_configuration_unproven.
func ObserveEffortApplicability(ctx context.Context, lookPath func(string) (string, error), version func(context.Context, string, string) (string, error), runtimeID string) (runtimeprofile.EffortApplicability, error) {
	var none runtimeprofile.EffortApplicability
	if err := ctx.Err(); err != nil {
		return none, err
	}
	if lookPath == nil || version == nil || runtimeID != "codex" && runtimeID != "claude" {
		return none, ErrEffortConfigurationUnproven
	}
	executable, err := lookPath(runtimeID)
	if err != nil || !filepath.IsAbs(executable) {
		return none, ErrEffortConfigurationUnproven
	}
	digest, err := ExecutableIdentity(executable)
	if err != nil {
		return none, ErrEffortConfigurationUnproven
	}
	observed, err := version(ctx, runtimeID, digest)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return none, ctxErr
	}
	if err != nil || !runtimeprofile.SafeRuntimeVersion(observed) {
		return none, ErrEffortConfigurationUnproven
	}
	if after, err := ExecutableIdentity(executable); err != nil || after != digest {
		return none, ErrEffortConfigurationUnproven
	}
	return runtimeprofile.EffortApplicability{RuntimeID: runtimeID, RuntimeVersion: observed, ExecutableDigest: digest, InvocationMode: "noninteractive"}, nil
}
