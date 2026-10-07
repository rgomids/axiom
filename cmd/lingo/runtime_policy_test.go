package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

func TestRuntimePolicyPreviewReadsRecordedSourceAndFreshInventory(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	source := env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	flags := installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "claude")
	path := flags[len(flags)-1]
	service := env.service.(lifecycleService)
	input := cli.RuntimeProfilePreviewInput{Project: "external", Role: "implementation", Complexity: "high", Capabilities: []string{"code"}, Observations: path}
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	preview, policy, err := service.runtimePolicyPreview(context.Background(), input)
	if err != nil || preview.Choice == nil || preview.Choice.RuntimeID != "claude" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	response := service.RuntimeProfilePreview(context.Background(), input)
	var output bytes.Buffer
	if code := cli.RunInteractive(context.Background(), append([]string{"--json", "runtime", "profile", "preview", "--project", "external"}, flags...), service, currentProvenance(), nil, &output, nil); code != 0 {
		t.Fatalf("output=%s", &output)
	}
	if response.PreviewDigest != preview.Digest() || strings.Contains(output.String(), "private-test-reference") || strings.Contains(output.String(), env.state) || strings.Contains(output.String(), path) {
		t.Fatalf("response=%+v output=%s", response, &output)
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("preview wrote state")
	}
	// Inventory is read on every Load, so old observations cannot authorize Start.
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var observations []runtimeprofile.Observation
	if err := json.Unmarshal(wire, &observations); err != nil {
		t.Fatal(err)
	}
	observations[0].Revision++
	wire, err = json.Marshal(observations)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := policy.Check(context.Background(), preview); err == nil {
		t.Fatal("accepted stale inventory")
	}
	// A changed portable source cannot bypass the installed revision boundary.
	manifestPath := filepath.Join(source, "axiom.yaml")
	wire, err = os.ReadFile(manifestPath)
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

func TestRuntimeObservationInventoryRejectsUntrustedData(t *testing.T) {
	observation := runtimeprofile.Observation{RuntimeID: "codex", Adapter: "codex", Installed: true, Available: true, Revision: 1, ObservedAt: time.Unix(1, 0).UTC(), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"code": runtimeprofile.CapabilityProven}}
	valid, err := json.Marshal([]runtimeprofile.Observation{observation})
	if err != nil {
		t.Fatal(err)
	}
	for name, wire := range map[string][]byte{"valid": valid, "unknown": []byte(strings.Replace(string(valid), `"installed":true`, `"installed":true,"private-field":"hidden"`, 1)), "unicode-fold": []byte(strings.Replace(string(valid), `"installed":true`, `"installed":false,"inſtalled":true`, 1)),
		"noncanonical-case":  []byte(strings.Replace(string(valid), `"installed":true`, `"Installed":true`, 1)),
		"unicode-capability": []byte(strings.Replace(string(valid), `"code":"proven"`, `"ſcode":"proven"`, 1)),
		"duplicate":          []byte(strings.Replace(string(valid), `"available":true`, `"available":false,"available":true`, 1)), "case-duplicate": []byte(strings.Replace(string(valid), `"available":true`, `"Available":false,"available":true`, 1)), "duplicate-capability": []byte(strings.Replace(string(valid), `"code":"proven"`, `"code":"declared","code":"proven"`, 1)), "trailing": append(append([]byte{}, valid...), valid...), "null": []byte("null"), "oversized": bytes.Repeat([]byte(" "), runtimeprofile.MaxConfigurationBytes+1)} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "inventory.json")
			if err := os.WriteFile(path, wire, 0600); err != nil {
				t.Fatal(err)
			}
			_, err := readRuntimeObservations(path)
			if (err == nil) != (name == "valid") {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if _, err := readRuntimeObservations("relative.json"); err == nil {
		t.Fatal("accepted relative path")
	}
	if _, err := readRuntimeObservations(t.TempDir()); err == nil {
		t.Fatal("accepted directory")
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
		snapshot.Configuration.Revision++
	}
	return snapshot, nil
}

func TestWorkflowStartRequiresReviewedFreshPolicy(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	flags := installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	service := env.service.(lifecycleService)
	input := cli.WorkflowInput{Project: workItemSourceProjectID, Repository: "main", Number: 7, Role: "implementation", Complexity: "high", Capabilities: []string{"code"}, Observations: flags[len(flags)-1]}
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
	snapshot, err := (runtimePolicySource{installation: service.installation, portable: service.portable, stateRoot: env.state, observations: input.Observations}).Load(context.Background(), input.Project)
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
	service.RuntimePolicySource = nil
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
