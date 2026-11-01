package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/codexruntime"
)

func TestRuntimePolicyPreviewReadsRecordedSourceAndMachineState(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	source := env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	flags := installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "claude")
	service, executable := testRuntime(t, env.service.(lifecycleService), "claude")
	service = withOperationalObservations(service, env.state)
	input := cli.RuntimeProfilePreviewInput{Project: "external", Role: "implementation", Complexity: "high", Capabilities: []string{"axiom-skills"}}
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	preview, policy, err := service.runtimePolicyPreview(context.Background(), input)
	if err != nil || preview.Choice == nil || preview.Choice.RuntimeID != "claude" || preview.Choice.ExecutableDigest == "" || preview.Choice.FrozenProfile == nil {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	response := service.RuntimeProfilePreview(context.Background(), input)
	var output bytes.Buffer
	if code := cli.RunInteractive(context.Background(), append([]string{"--json", "runtime", "profile", "preview", "--project", "external"}, flags...), service, currentProvenance(), nil, &output, nil); code != 0 {
		t.Fatalf("output=%s", &output)
	}
	if response.PreviewDigest != preview.Digest() || strings.Contains(output.String(), "private-test-reference") || strings.Contains(output.String(), env.state) || strings.Contains(output.String(), executable) || strings.Contains(output.String(), filepath.Dir(executable)) {
		t.Fatalf("response=%+v output=%s", response, &output)
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("preview wrote state")
	}
	// Every Check observes the machine again: a replaced or removed executable
	// no longer matches the reviewed identity.
	if _, err := policy.Check(context.Background(), preview); err != nil {
		t.Fatalf("unchanged Runtime rejected: %v", err)
	}
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n# replaced\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := policy.Check(context.Background(), preview); err == nil {
		t.Fatal("accepted replaced executable")
	}
	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	if _, err := policy.Check(context.Background(), preview); err == nil {
		t.Fatal("accepted removed executable")
	}
	// A changed portable source cannot bypass the installed revision boundary.
	manifestPath := filepath.Join(source, "axiom.yaml")
	wire, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	wire = []byte(strings.Replace(string(wire), "name: External", "name: Changed", 1))
	if err := os.WriteFile(manifestPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	blocked := service.RuntimeProfilePreview(context.Background(), input)
	if blocked.RuntimeResolution.Blocker == nil || blocked.RuntimeResolution.Blocker.Code != "policy_unavailable" {
		t.Fatalf("blocked=%+v", blocked.RuntimeResolution)
	}
}

func TestRuntimePolicyObservesConsentIdentityAndIntent(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			env := newWorkItemSourceEnvironment(t)
			env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
			installTestRuntimePolicy(t, env.state, workItemSourceProjectID, runtimeID)
			base := env.service.(lifecycleService)
			production, executable := testRuntime(t, base, runtimeID)
			input := cli.RuntimeProfilePreviewInput{Project: "external", Role: "implementation", Complexity: "high", Capabilities: []string{"axiom-skills"}}
			blockedWith := func(name, code string, service lifecycleService, input cli.RuntimeProfilePreviewInput) {
				t.Helper()
				preview, _, err := service.runtimePolicyPreview(context.Background(), input)
				if err == nil || preview.Choice != nil || preview.Blocker == nil || preview.Blocker.Code != code {
					t.Fatalf("%s resolved: %+v", name, preview)
				}
			}
			blockedWith("inspection consent missing", "inspection_consent_required", production, input)
			refusal := production.RuntimeProfilePreview(context.Background(), input)
			if !strings.Contains(refusal.Completion.Next().String(), "workflow start") {
				t.Fatal("inspection refusal lacks a scoped recovery command")
			}
			service := withOperationalObservations(production, env.state)
			if preview, _, err := service.runtimePolicyPreview(context.Background(), input); err != nil || preview.Choice == nil {
				t.Fatalf("verified Runtime blocked: %+v %v", preview, err)
			}
			unproven := input
			unproven.Capabilities = []string{"code"}
			blockedWith("unprovable capability", "no_allowed_match", service, unproven)
			absent := production
			absent.runtimes.lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
			blockedWith("absent executable", "executable_unavailable", withOperationalObservations(absent, env.state), input)
			unverified, _ := testRuntime(t, base, runtimeID)
			unverified.runtimes.lookPath = service.runtimes.lookPath
			skills, err := codexruntime.New(filepath.Join(t.TempDir(), "missing"))
			if err != nil {
				t.Fatal(err)
			}
			unverified.codex, unverified.runtimes.claudeSkillsRoot = skills, filepath.Join(t.TempDir(), "missing")
			if preview, _, err := withOperationalObservations(unverified, env.state).runtimePolicyPreview(context.Background(), input); err != nil || preview.Choice == nil {
				t.Fatal("skill installation became certification gate", err)
			}
			if err := os.Chmod(executable, 0o600); err != nil {
				t.Fatal(err)
			}
			blockedWith("non-executable file", "executable_unavailable", service, input)
		})
	}
}
