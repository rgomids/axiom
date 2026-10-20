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
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
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

type changingRuntimePolicySource struct {
	snapshot runtimeapplication.Snapshot
	loads    int
}

func (s *changingRuntimePolicySource) Load(context.Context, string) (runtimeapplication.Snapshot, error) {
	s.loads++
	snapshot := s.snapshot
	if s.loads > 1 {
		cfg := snapshot.Consumer.Configuration()
		cfg.Revision++
		snapshot.Consumer, _ = runtimeprofile.NewConsumerV4(cfg)
	}
	return snapshot, nil
}

func TestWorkflowStartRequiresReviewedFreshPolicy(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service, executable := testRuntime(t, env.service.(lifecycleService), "codex")
	service = currentProfileStartFixture(t, service, env.state)
	input := cli.WorkflowInput{Project: workItemSourceProjectID, Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{"axiom-skills"}}
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	preview := service.WorkflowStart(context.Background(), input)
	if preview.RuntimeResolution == nil || preview.RuntimeResolution.Choice == nil || preview.Workflow != nil {
		t.Fatalf("preview=%+v", preview)
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("preview created an execution")
	}
	input.RuntimePreview = strings.Repeat("a", 64)
	mismatch := service.WorkflowStart(context.Background(), input)
	if mismatch.RuntimeResolution.Blocker.Code != "stale_preview" {
		t.Fatalf("mismatch=%+v", mismatch)
	}
	input.RuntimePreview = preview.PreviewDigest
	originalSource := service.RuntimePolicySource
	snapshot, err := originalSource.Load(context.Background(), input.Project)
	if err != nil {
		t.Fatal(err)
	}
	changing := &changingRuntimePolicySource{snapshot: snapshot}
	service.RuntimePolicySource = changing
	stale := service.WorkflowStart(context.Background(), input)
	if stale.RuntimeResolution.Blocker.Code != "stale_preview" || changing.loads != 2 {
		t.Fatalf("stale=%+v loads=%d", stale, changing.loads)
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("stale preview created an execution")
	}
	service.RuntimePolicySource = originalSource
	// The reviewed digest names one executable; replacing or removing it after
	// review blocks before any Execution exists.
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n# replaced\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if replaced := service.WorkflowStart(context.Background(), input); replaced.RuntimeResolution == nil || replaced.RuntimeResolution.Blocker == nil || replaced.RuntimeResolution.Blocker.Code != "stale_preview" || replaced.Workflow != nil {
		t.Fatalf("replaced=%+v", replaced)
	}
	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	if removed := service.WorkflowStart(context.Background(), input); removed.RuntimeResolution == nil || removed.RuntimeResolution.Blocker == nil || removed.Workflow != nil {
		t.Fatalf("removed=%+v", removed)
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("drifted Runtime created an execution")
	}
	input.Runtime = "claude"
	denied := service.WorkflowStart(context.Background(), input)
	if denied.RuntimeResolution.Choice != nil || denied.RuntimeResolution.Blocker == nil {
		t.Fatal("explicit Runtime widened policy")
	}
	input.Role = ""
	input.Runtime = ""
	invalid := service.WorkflowStart(context.Background(), input)
	if invalid.RuntimeResolution.Blocker.Code != "invalid_request" {
		t.Fatal("missing policy requirements used fallback")
	}
}

// A Project that allows only Claude never starts on an installed Codex, with or
// without an explicit Runtime, and nothing selects Codex by default.
func TestWorkflowStartNeverFallsBackToCodex(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "claude")
	service, _ := testRuntime(t, env.service.(lifecycleService), "codex")
	service = currentProfileStartFixture(t, service, env.state)
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	for _, runtimeID := range []string{"", "codex", "claude"} {
		input := cli.WorkflowInput{Project: workItemSourceProjectID, Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{"axiom-skills"}, Runtime: runtimeID}
		result := service.WorkflowStart(context.Background(), input)
		if result.RuntimeResolution == nil || result.RuntimeResolution.Choice != nil || result.RuntimeResolution.Blocker == nil || result.Workflow != nil {
			t.Fatalf("runtime=%q result=%+v", runtimeID, result)
		}
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("blocked start wrote state")
	}
}
