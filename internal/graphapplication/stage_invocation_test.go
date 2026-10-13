package graphapplication

import (
	"context"
	"errors"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"slices"
	"testing"
)

type stageAuthFake struct {
	target  runtimeadapter.AuthTarget
	status  runtimeadapter.AuthStatus
	version string
}

func (f *stageAuthFake) Check(_ context.Context, target runtimeadapter.AuthTarget) runtimeadapter.AuthReport {
	f.target = target
	return runtimeadapter.AuthReport{RuntimeID: target.RuntimeID, Status: f.status, Version: f.version, ExecutableDigest: target.ExpectedDigest, EvidenceKind: runtimeadapter.EvidenceFake}
}
func TestPrepareStageMandatorySubscription(t *testing.T) {
	skipPOSIXStagePaths(t)
	request := runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: "codex", Executable: "/bin/codex", Model: "model"}, Version: "0.159.1", Workspace: "/work", Inbox: "/inbox", AxiomPath: "/bin/axiom", ParentEnvironment: []string{"HOME=/home/example", "PATH=/bin", "GH_TOKEN=synthetic"}}
	for _, status := range []runtimeadapter.AuthStatus{runtimeadapter.AuthUnavailable, runtimeadapter.AuthUnproven, runtimeadapter.AuthIncompatible, runtimeadapter.AuthUnsupported} {
		f := &stageAuthFake{status: status, version: request.Version}
		if _, _, err := PrepareStageInvocation(context.Background(), request, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", f); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) {
			t.Fatal(err)
		}
	}
	if _, report, err := PrepareStageInvocation(context.Background(), request, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || report.Reason != "subscription_auth_required" {
		t.Fatalf("%+v %v", report, err)
	}
	f := &stageAuthFake{status: runtimeadapter.AuthSubscriptionObserved, version: request.Version}
	inv, _, err := PrepareStageInvocation(context.Background(), request, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", f)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.target.Environment, executiongraph.EffectiveInvocationEnvironment(inv)) || slices.Contains(f.target.Environment, "GH_TOKEN=synthetic") {
		t.Fatal("preflight differs")
	}
	t.Setenv("HOME", "changed")
	request.ParentEnvironment[0] = "HOME=changed"
	if !slices.Equal(f.target.Environment, executiongraph.EffectiveInvocationEnvironment(inv)) {
		t.Fatal("environment not frozen")
	}
	request.Profile.CredentialReference = "reference"
	_, report, err := PrepareStageInvocation(context.Background(), request, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", f)
	if !errors.Is(err, executiongraph.ErrAuthenticationBlocked) || report.Reason != "credential_reference_configured" {
		t.Fatalf("%+v %v", report, err)
	}
	request.Profile.CredentialReference = ""
	f.version = "unknown"
	if _, _, err := PrepareStageInvocation(context.Background(), request, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", f); !errors.Is(err, executiongraph.ErrAuthenticationBlocked) {
		t.Fatal("unknown version", err)
	}
}
