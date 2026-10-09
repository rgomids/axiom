package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
	"github.com/rgomids/axiom/internal/runtimeadapter"
)

func authProvenance(t *testing.T) provenance.Value {
	t.Helper()
	source, err := provenance.FromBuild(provenance.Build{Version: provenance.Development, Revision: "abc123", SourceState: provenance.Clean}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

// Only an observed subscription login completes; every other outcome is a
// validation failure that keeps dispatch blocked.
func TestRuntimeAuthResultCompletesOnlyObservedSubscription(t *testing.T) {
	source := authProvenance(t)
	for status, want := range map[runtimeadapter.AuthStatus]completion.Status{
		runtimeadapter.AuthSubscriptionObserved: completion.Success,
		runtimeadapter.AuthUnproven:             completion.ValidationFailure,
		runtimeadapter.AuthIncompatible:         completion.ValidationFailure,
		runtimeadapter.AuthUnavailable:          completion.ValidationFailure,
		runtimeadapter.AuthUnsupported:          completion.ValidationFailure,
	} {
		result := runtimeAuthResult(runtimeadapter.AuthReport{RuntimeID: "codex", Status: status}, "Codex", source)
		if result.Completion == nil || result.Completion.Status() != want || result.RuntimeAuth == nil || result.RuntimeAuth.Status != status {
			t.Fatalf("%s: result=%+v", status, result)
		}
	}
}

func TestRuntimeAuthWithoutExecutableIsUnavailable(t *testing.T) {
	service := lifecycleService{provenance: authProvenance(t), runtimes: runtimeRoots{lookPath: func(string) (string, error) { return "", errors.New("absent") }}}
	result := service.RuntimeAuth(context.Background(), "claude")
	if result.RuntimeAuth == nil || result.RuntimeAuth.Status != runtimeadapter.AuthUnavailable || result.RuntimeAuth.EvidenceKind != runtimeadapter.EvidenceLocalObservation {
		t.Fatalf("result=%+v", result.RuntimeAuth)
	}
}

// A controlled fake executable on the lookup path stands in for the vendor
// CLI; it never authenticates or runs inference.
func TestRuntimeAuthObservesTheLookedUpExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	for _, name := range []string{"OPENAI_API_KEY", "OPENAI_BASE_URL", "CODEX_API_KEY", "AZURE_OPENAI_API_KEY"} {
		t.Setenv(name, "") // registers restoration
		os.Unsetenv(name)
	}
	executable := filepath.Join(t.TempDir(), "codex")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 0.159.1'; exit 0; fi\necho 'Logged in using ChatGPT' >&2\n"
	if err := os.WriteFile(executable, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	service := lifecycleService{provenance: authProvenance(t), runtimes: runtimeRoots{lookPath: func(string) (string, error) { return executable, nil }}}
	result := service.RuntimeAuth(context.Background(), "codex")
	if result.RuntimeAuth == nil || result.RuntimeAuth.Status != runtimeadapter.AuthSubscriptionObserved || result.RuntimeAuth.Version != "0.159.1" || result.Completion.Status() != completion.Success {
		t.Fatalf("report=%+v", result.RuntimeAuth)
	}
	t.Setenv("CODEX_API_KEY", "sk-synthetic")
	if result := service.RuntimeAuth(context.Background(), "codex"); result.RuntimeAuth.Status != runtimeadapter.AuthIncompatible {
		t.Fatalf("inherited override report=%+v", result.RuntimeAuth)
	}
}
