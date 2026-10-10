package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const authoredPolicyManifest = `schemaVersion: 2
project:
  id: 123e4567-e89b-42d3-a456-426614174000
  slug: authored
  name: Authored
repositories:
  - key: main
providers:
  - key: work-items
    id: github
integrations:
  - key: work-items
    providerRef: work-items
    capabilities:
      - work-item
runtimes:
  - id: claude
  - id: codex
modelProfiles:
  - key: careful
    runtimeRef: claude
    model: approved-model
  - key: worker
    runtimeRef: codex
    model: approved-model
runtimePreferences:
  - role: implementation
    complexity: high
    modelProfileRef: worker
`

// An operator-authored Runtime/Profile policy reaches production only through
// install with an exact binding for every declared Repository.
func TestProjectInstallBindsEveryDeclaredRepository(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	source := filepath.Join(env.elsewhere, "authored")
	repository := filepath.Join(t.TempDir(), "repository")
	for _, directory := range []string{source, repository} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "axiom.yaml"), []byte(authoredPolicyManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	before := snapshotTrees(t, env.root, env.elsewhere)
	for _, bindings := range [][]string{
		nil,
		{"--repository", "main=relative"},
		{"--repository", "other=" + repository},
		{"--repository", "main=" + filepath.Join(t.TempDir(), "missing")},
		{"--repository", "main=" + repository, "--repository", "main=" + repository},
		{"--repository", "main=" + repository, "--repository", "other=" + repository},
	} {
		var output bytes.Buffer
		args := append([]string{"--json", "project", "install", "--source", source}, bindings...)
		if code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output); code == cli.ExitSuccess || strings.Contains(output.String(), repository) {
			t.Fatalf("%v: code=%d output=%s", bindings, code, &output)
		}
	}
	if after := snapshotTrees(t, env.root, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("rejected install changed portable state")
	}
	if records, _ := filepath.Glob(filepath.Join(env.state, "projects", "*", "installation.json")); len(records) != 0 {
		t.Fatalf("rejected install recorded local state: %v", records)
	}
	runCLI(t, env.service, []string{"project", "install", "--source", source, "--repository", "main=" + repository}, cli.ExitSuccess, "installed")
	runCanonicalCLI(t, env.service, []string{"project", "show", "--selector", "authored"}, cli.ExitSuccess, "success", "Project resolved")
	runCLI(t, env.service, []string{"project", "install", "--source", source, "--repository", "main=" + repository}, cli.ExitSuccess, "already_installed")
}

// The installed authored policy resolves through Lingo's own observation and
// the reviewed digest is the only way from preview to the next step.
func TestAuthoredPolicyPreviewThenStartBothRuntimes(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		t.Run(runtimeID, func(t *testing.T) {
			env := newWorkItemSourceEnvironment(t)
			source := filepath.Join(env.elsewhere, "authored")
			repository := filepath.Join(t.TempDir(), "repository")
			for _, directory := range []string{source, repository} {
				if err := os.Mkdir(directory, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(source, "axiom.yaml"), []byte(authoredPolicyManifest), 0o600); err != nil {
				t.Fatal(err)
			}
			runCLI(t, env.service, []string{"project", "install", "--source", source, "--repository", "main=" + repository}, cli.ExitSuccess, "installed")
			selectBuiltinForTest(t, env.state, "123e4567-e89b-42d3-a456-426614174000")
			store, err := local.NewRuntimeProfileStore(env.state)
			if err != nil {
				t.Fatal(err)
			}
			profile := map[string]string{"codex": "worker", "claude": "careful"}[runtimeID]
			cfg := runtimeprofile.Configuration{FormatVersion: 1, Revision: 1,
				Runtimes:      []runtimeprofile.Runtime{{ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"careful"}}, {ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"worker"}}},
				ModelProfiles: []runtimeprofile.ModelProfile{{ID: "careful", RuntimeID: "claude", Model: "approved-model", Capabilities: []string{runtimeadapter.IntegrationCapability}}, {ID: "worker", RuntimeID: "codex", Model: "approved-model", Capabilities: []string{runtimeadapter.IntegrationCapability}}}}
			if err := store.Create(context.Background(), cfg); err != nil {
				t.Fatal(err)
			}
			// Only one Runtime is observable; the other declared one is absent.
			service, _ := testRuntime(t, env.service.(lifecycleService), runtimeID)
			saveTargetLink(t, env.state, "123e4567-e89b-42d3-a456-426614174000", "7", "owner/repo")
			input := cli.WorkflowInput{Project: "authored", Repository: "main", Number: 7, Role: "review", Complexity: "low", Capabilities: []string{runtimeadapter.IntegrationCapability}}
			preview := service.WorkflowStart(context.Background(), input)
			if preview.RuntimeResolution == nil || preview.RuntimeResolution.Choice == nil || preview.RuntimeResolution.Choice.RuntimeID != runtimeID || preview.RuntimeResolution.Choice.ModelProfileID != profile || preview.Workflow != nil || preview.PreviewDigest == "" {
				t.Fatalf("preview=%+v", preview.RuntimeResolution)
			}
			input.RuntimePreview = preview.PreviewDigest
			started := service.WorkflowStart(context.Background(), input)
			// The linked target and reviewed policy produce the selected Runtime.
			if started.RuntimeResolution != nil || started.Workflow == nil || started.Workflow.RuntimeID != runtimeID {
				t.Fatalf("reviewed start still blocked by policy: %+v", started.RuntimeResolution)
			}
		})
	}
}
