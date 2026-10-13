package graphapplication

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
)

// PrepareStageInvocation is the internal #276 dispatch/probe seam. It cannot
// opt out of subscription authentication and never resolves credentials.
// It is intentionally not wired to historical public graph commands.
func PrepareStageInvocation(ctx context.Context, request runtimeadapter.StageInvocationRequest, digest string, auth SubscriptionAuthenticator) (executiongraph.Invocation, runtimeadapter.AuthReport, error) {
	report := runtimeadapter.AuthReport{RuntimeID: request.Profile.RuntimeID, Status: runtimeadapter.AuthUnproven, Reason: "subscription_auth_required", Method: "unknown", Version: "unknown", EvidenceKind: runtimeadapter.EvidenceFake, Usability: runtimeadapter.UsabilityUnproven, Revalidation: runtimeadapter.RevalidateBeforeDispatch}
	if request.Profile.CredentialReference != "" {
		report.Status, report.Reason = runtimeadapter.AuthIncompatible, "credential_reference_configured"
		return executiongraph.Invocation{}, report, fmt.Errorf("%w: %s", executiongraph.ErrAuthenticationBlocked, report.Reason)
	}
	digestBytes, digestErr := hex.DecodeString(digest)
	if auth == nil || digestErr != nil || len(digestBytes) != 32 {
		return executiongraph.Invocation{}, report, fmt.Errorf("%w: %s", executiongraph.ErrAuthenticationBlocked, report.Reason)
	}
	invocation, err := runtimeadapter.BuildStageInvocation(request)
	if err != nil {
		return executiongraph.Invocation{}, report, err
	}
	report = auth.Check(ctx, runtimeadapter.AuthTarget{RuntimeID: invocation.RuntimeID, Executable: invocation.Argv[0], ExpectedDigest: digest, WorkingDirectory: invocation.CWD, Environment: executiongraph.EffectiveInvocationEnvironment(invocation)})
	if report.RuntimeID != request.Profile.RuntimeID || !report.DispatchAllowed() || report.Version != request.Version || report.ExecutableDigest != digest {
		return executiongraph.Invocation{}, report, fmt.Errorf("%w: subscription_binding_unproven", executiongraph.ErrAuthenticationBlocked)
	}
	return invocation, report, nil
}
