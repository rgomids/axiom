package main

import (
	"bytes"
	"context"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"strings"
	"testing"
)

func TestRuntimePolicyPreviewRetiredWithoutObservationOrWrites(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	service := env.service.(lifecycleService)
	service.runtimes.lookPath = func(string) (string, error) { t.Fatal("retired preview observed executable"); return "", nil }
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	var out bytes.Buffer
	args := []string{"--json", "runtime", "profile", "preview", "--project", "external", "--role", "implementation", "--complexity", "high", "--capabilities", "axiom-skills"}
	if code := cli.RunInteractive(context.Background(), args, service, currentProvenance(), nil, &out, nil); code != cli.ExitFailure {
		t.Fatalf("retired preview exit %d: %s", code, &out)
	}
	if !strings.Contains(out.String(), "Predictive Runtime Profile preview is retired") || !strings.Contains(out.String(), "workflow run") || strings.Contains(out.String(), env.state) {
		t.Fatalf("retirement diagnostic: %s", &out)
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("retired preview wrote state")
	}
	// No Project resolution or local configuration is required to explain retirement.
	if got := (lifecycleService{provenance: currentProvenance()}).RuntimeProfilePreview(context.Background(), cli.RuntimeProfilePreviewInput{}); got.Category != "runtime_policy_preview_retired" || got.RuntimeResolution != nil {
		t.Fatalf("unexpected retired response: %+v", got)
	}
}

func TestRuntimeAuthDiagnosticDoesNotClaimDispatchAdmission(t *testing.T) {
	for _, status := range []runtimeadapter.AuthStatus{runtimeadapter.AuthSubscriptionObserved, runtimeadapter.AuthUnproven, runtimeadapter.AuthUnavailable} {
		result := runtimeAuthResult(runtimeadapter.AuthReport{RuntimeID: "codex", Status: status, Revalidation: runtimeadapter.RevalidateBeforeDispatch}, "Codex", currentProvenance())
		if result.RuntimeAuth == nil || result.RuntimeAuth.Revalidation != "diagnostic_only" {
			t.Fatalf("auth claims dispatch admission: %+v", result.RuntimeAuth)
		}
		if next := result.Completion.Next().String(); strings.Contains(next, "repeats this preflight") || strings.Contains(next, "dispatch stays blocked") {
			t.Fatalf("auth claims admission: %s", next)
		}
	}
}
