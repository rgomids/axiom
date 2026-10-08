package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Issue #230 I230-T08 bounded end-to-end maintenance journey (AC-51, AC-53,
// AC-54, AC-55, AC-56, AC-57, AC-58, AC-61). Every step goes through the
// executable CLI parser and the same service the Runtime skills invoke; no
// step edits an Axiom persisted file. The only setup outside the journey is
// the machine-local Runtime Profile configuration and the Provider stub.

type journey struct {
	t        *testing.T
	service  cli.Service
	provider string
}

func (j journey) run(wantCode int, args ...string) map[string]any {
	j.t.Helper()
	var output bytes.Buffer
	code := cli.Run(context.Background(), args, j.service, currentProvenance(), &output)
	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil || code != wantCode {
		j.t.Fatalf("%v: code=%d err=%v output=%s", args, code, err, output.String())
	}
	return event
}

// field walks a decoded JSON object by keys.
func field(event map[string]any, path ...string) any {
	var current any = event
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[key]
	}
	return current
}

func text(event map[string]any, path ...string) string {
	value, _ := field(event, path...).(string)
	return value
}

func (j journey) patches() string {
	wire, _ := os.ReadFile(filepath.Join(j.provider, "patches"))
	return string(wire)
}

func TestResourceLifecycleJourneyWithoutPersistedFileEditing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Provider stub is a POSIX script")
	}
	env := newBootstrapEnv(t)
	providerDir := t.TempDir()
	gh := filepath.Join(providerDir, "gh")
	script := `#!/bin/sh
dir="$AXIOM_TEST_PROVIDER"
state=$(cat "$dir/state" 2>/dev/null || echo open)
case "$*" in
  *PATCH*repos/acme/core/issues/7*)
    input=$(cat)
    printf '%s\n' "$input" >> "$dir/patches"
    case "$input" in
      *'"state":"closed"'*) state=closed ;;
      *'"state":"open"'*) state=open ;;
    esac
    echo "$state" > "$dir/state" ;;
  *repos/acme/core/issues/7*) ;;
  *) exit 1 ;;
esac
printf '{"number":7,"html_url":"https://github.com/acme/core/issues/7","state":"%s","title":"Current title","body":"Current body"}\n' "$state"
`
	if err := os.WriteFile(gh, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_GH_BIN", gh)
	t.Setenv("AXIOM_TEST_PROVIDER", providerDir)
	profiles, err := local.NewRuntimeProfileStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	cfg := runtimeprofile.Configuration{FormatVersion: 1, Revision: 1, Runtimes: []runtimeprofile.Runtime{{ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"worker"}, CredentialReference: "private-test-reference"}}, ModelProfiles: []runtimeprofile.ModelProfile{{ID: "worker", RuntimeID: "claude", Model: "approved-model", Capabilities: []string{"axiom-skills"}, Complexities: []string{"high"}}}}
	if err := profiles.Create(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	service, _ := testRuntime(t, compose().(lifecycleService), "claude")
	j := journey{t: t, service: service, provider: providerDir}
	core := writeTree(t, filepath.Join(env.workspace, "core"), map[string]string{"README.md": "core"})
	docs := writeTree(t, filepath.Join(env.workspace, "docs"), map[string]string{"SENTINEL": "working copy content"})

	// Project: configure (create), discover, inspect, validate.
	create := []string{"project", "configure", "--slug", "journey", "--name", "Journey", "--repository", "core=" + core, "--repository-remote", "core=none", "--work-item-provider", "github",
		"--runtime", "claude", "--model-profile", "worker", "--runtime-preference", "implementation/high=worker"}
	setup := j.run(cli.ExitSuccess, create...)
	projectID := text(setup, "setup", "projectId")
	j.run(cli.ExitSuccess, append(create, "--project-id", projectID, "--preview-digest", text(setup, "setup", "digest"), "--authorize-local")...)
	listed := j.run(cli.ExitSuccess, "project", "list")
	if !strings.Contains(mustJSON(t, listed["projects"]), `"slug":"journey"`) || !strings.Contains(mustJSON(t, listed["projects"]), `"status":"active"`) {
		t.Fatalf("list = %v", listed["projects"])
	}
	j.run(cli.ExitSuccess, "project", "show", "--selector", "journey")
	validated := j.run(cli.ExitSuccess, "project", "validate", "--project", "journey")
	if text(validated, "readiness", "effective") != "ready" {
		t.Fatalf("validate = %v", validated["readiness"])
	}

	// Project edit and Repository association: attach, then detach.
	rename := []string{"project", "configure", "--project", "journey", "--name", "Journey Renamed", "--repository", "docs=" + docs}
	preview := j.run(cli.ExitSuccess, rename...)
	j.run(cli.ExitSuccess, append(rename, "--project-id", projectID, "--preview-digest", text(preview, "edit", "digest"), "--authorize-local")...)
	shown := j.run(cli.ExitSuccess, "project", "show", "--selector", "journey")
	if !strings.Contains(mustJSON(t, shown["project"]), `"key":"docs"`) {
		t.Fatalf("attached Repository not shown: %v", shown["project"])
	}
	detach := []string{"project", "configure", "--project", "journey", "--remove-repository", "docs"}
	preview = j.run(cli.ExitSuccess, detach...)
	if !strings.Contains(mustJSON(t, preview["edit"]), "preserve_repository_history") {
		t.Fatalf("detach preview hides preserved history: %v", preview["edit"])
	}
	j.run(cli.ExitSuccess, append(detach, "--project-id", projectID, "--preview-digest", text(preview, "edit", "digest"), "--authorize-local")...)
	if sentinel, err := os.ReadFile(filepath.Join(docs, "SENTINEL")); err != nil || string(sentinel) != "working copy content" {
		t.Fatalf("detach touched the working copy: %q %v", sentinel, err)
	}
	manifest, err := os.ReadFile(filepath.Join(env.root, "journey", "axiom.yaml"))
	if err != nil || !strings.Contains(string(manifest), "name: Journey Renamed") || strings.Contains(string(manifest), "docs") || strings.Contains(string(manifest), env.workspace) {
		t.Fatalf("portable intent after edit = %s %v", manifest, err)
	}

	// Integration: inspect and validate.
	integrations := j.run(cli.ExitSuccess, "integration", "list", "--project", "journey")
	if !strings.Contains(mustJSON(t, integrations["integrations"]), `"key":"work-items"`) {
		t.Fatalf("integration list = %v", integrations["integrations"])
	}
	j.run(cli.ExitSuccess, "integration", "validate", "--project", "journey")

	// Work Item: select, discover, update.
	selector := []string{"--project", "journey", "--repository", "core", "--work-item", "github:acme/core#7"}
	selection := j.run(cli.ExitSuccess, append([]string{"work-item", "select"}, selector...)...)
	j.run(cli.ExitSuccess, append(append([]string{"work-item", "select"}, selector...), "--preview-digest", text(selection, "selection", "digest"), "--authorize-local")...)
	if items := j.run(cli.ExitSuccess, "work-item", "list", "--project", "journey"); !strings.Contains(mustJSON(t, items["workItems"]), `"externalId":"7"`) {
		t.Fatalf("work-item list = %v", items["workItems"])
	}
	update := append(append([]string{"work-item", "update"}, selector...), "--title", "Clearer title")
	change := j.run(cli.ExitSuccess, update...)
	j.run(cli.ExitSuccess, append(update, "--preview-digest", text(change, "change", "digest"), "--authorize-external")...)

	// Execution: reviewed start, then discovery without a known identity.
	start := append(append([]string{"workflow", "start"}, selector...), "--role", "implementation", "--complexity", "high", "--capabilities", "axiom-skills", "--runtime", "claude")
	resolution := j.run(cli.ExitSuccess, start...)
	j.run(cli.ExitSuccess, append(start, "--runtime-preview", text(resolution, "previewDigest"))...)
	executions := j.run(cli.ExitSuccess, "workflow", "list", "--project", "journey")
	listedExecutions, _ := executions["executions"].([]any)
	if len(listedExecutions) != 1 {
		t.Fatalf("workflow list = %v", executions["executions"])
	}
	executionID := text(listedExecutions[0].(map[string]any), "executionId")
	j.run(cli.ExitSuccess, append(append([]string{"workflow", "status"}, selector...), "--execution", executionID)...)

	// Archive blocks evolution, not inspection or administration.
	archive := []string{"project", "archive", "--project", "journey"}
	archivePreview := j.run(cli.ExitSuccess, archive...)
	j.run(cli.ExitSuccess, append(archive, "--preview-digest", text(archivePreview, "operational", "digest"), "--authorize-local")...)
	if hidden := j.run(cli.ExitSuccess, "project", "list"); strings.Contains(mustJSON(t, hidden["projects"]), "journey") {
		t.Fatal("archived Project listed by default")
	}
	if all := j.run(cli.ExitSuccess, "project", "list", "--include-archived"); !strings.Contains(mustJSON(t, all["projects"]), `"status":"archived"`) {
		t.Fatalf("include-archived = %v", all["projects"])
	}
	patches := j.patches()
	for _, denied := range [][]string{
		update,
		append(append([]string{"workflow", "advance"}, selector...), "--execution", executionID, "--expected-revision", "1", "--gate", "specification", "--outcome", "pass", "--reference", "evidence:spec.md:abc"),
		append([]string{"work-item", "close"}, selector...),
	} {
		if event := j.run(cli.ExitFailure, denied...); text(event, "admission", "code") != "project_archived" {
			t.Fatalf("%v while archived = %v", denied, event)
		}
	}
	j.run(cli.ExitSuccess, "workflow", "list", "--project", "journey")
	j.run(cli.ExitSuccess, "integration", "list", "--project", "journey")
	j.run(cli.ExitSuccess, "project", "validate", "--project", "journey")
	reactivate := []string{"project", "reactivate", "--project", "journey"}
	reactivatePreview := j.run(cli.ExitSuccess, reactivate...)
	j.run(cli.ExitSuccess, append(reactivate, "--preview-digest", text(reactivatePreview, "operational", "digest"), "--authorize-local")...)

	// A locally disabled Integration cannot be used; enabling restores it.
	disable := []string{"integration", "disable", "--project", "journey", "--integration", "work-items"}
	disablePreview := j.run(cli.ExitSuccess, disable...)
	j.run(cli.ExitSuccess, append(disable, "--preview-digest", text(disablePreview, "operational", "digest"), "--authorize-local")...)
	closeArgs := append([]string{"work-item", "close"}, selector...)
	if event := j.run(cli.ExitFailure, closeArgs...); text(event, "admission", "code") != "integration_disabled" {
		t.Fatalf("close with disabled Integration = %v", event)
	}
	if j.patches() != patches {
		t.Fatal("denied operations reached the Provider")
	}
	if !strings.Contains(string(mustRead(t, filepath.Join(env.root, "journey", "axiom.yaml"))), "work-items") {
		t.Fatal("disable changed the portable declaration")
	}
	enable := []string{"integration", "enable", "--project", "journey", "--integration", "work-items"}
	enablePreview := j.run(cli.ExitSuccess, enable...)
	j.run(cli.ExitSuccess, append(enable, "--preview-digest", text(enablePreview, "operational", "digest"), "--authorize-local")...)
	closePreview := j.run(cli.ExitSuccess, closeArgs...)
	j.run(cli.ExitSuccess, append(closeArgs, "--preview-digest", text(closePreview, "change", "digest"), "--authorize-external")...)
	if items := j.run(cli.ExitSuccess, "work-item", "list", "--project", "journey"); !strings.Contains(mustJSON(t, items["workItems"]), `"state":"CLOSED"`) {
		t.Fatalf("closed Work Item = %v", items["workItems"])
	}
	if want := `{"title":"Clearer title"}` + "\n" + `{"state":"closed"}` + "\n"; j.patches() != want {
		t.Fatalf("Provider mutations = %q", j.patches())
	}
	// Machine-local operational state never leaked into portable intent.
	if portable := string(mustRead(t, filepath.Join(env.root, "journey", "axiom.yaml"))); strings.Contains(portable, "archived") || strings.Contains(portable, "disabled") {
		t.Fatalf("operational state in portable intent: %s", portable)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	wire, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(wire)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}
