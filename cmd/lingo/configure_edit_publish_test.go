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
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T03 Evidence (Issue #132 I132-T02 behavior): authorized
// Project EDIT publication over the real portable/local stores, stale-safe
// authority, and ADR-0007 cross-store recovery.

func (env editEnvironment) configureWith(t *testing.T, service cli.Service, args ...string) (int, editEvent, string) {
	t.Helper()
	var output bytes.Buffer
	code := cli.Run(context.Background(), append([]string{"project", "configure"}, args...), service, currentProvenance(), &output)
	var event editEvent
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("%v: %v: %s", args, err, output.String())
	}
	return code, event, output.String()
}

// replayArgs returns the exact replay tuple of a reviewed preview.
func replayArgs(preview projectapp.EditPreview, args ...string) []string {
	return append(append([]string(nil), args...), "--project-id", preview.ProjectID, "--preview-digest", preview.Digest, "--authorize-local")
}

// publish previews (zero-write) and replays the exact reviewed edit.
func (env editEnvironment) publish(t *testing.T, args ...string) (projectapp.EditPreview, editEvent) {
	t.Helper()
	preview, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", args...)
	if want := "--project-id " + preview.Edit.ProjectID + " --preview-digest " + preview.Edit.Digest + " --authorize-local"; !strings.Contains(preview.Next, want) {
		t.Fatalf("preview next action does not name the exact replay: %q", preview.Next)
	}
	code, event, output := env.configureWith(t, env.service, replayArgs(preview.Edit, args...)...)
	if code != cli.ExitSuccess || event.Status != "success" || event.Result != "Project edit published" {
		t.Fatalf("%v: code=%d output=%s", args, code, output)
	}
	env.assertNoEditState(t)
	return preview.Edit, event
}

func (env editEnvironment) installationPath() string {
	return filepath.Join(env.state, "projects", env.projectID, "installation.json")
}

func (env editEnvironment) record(t *testing.T) local.RecordState {
	t.Helper()
	wire, err := os.ReadFile(env.installationPath())
	if err != nil {
		t.Fatal(err)
	}
	record, issues := local.DecodeRecord(wire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return record.State()
}

func (env editEnvironment) manifestPath() string {
	return filepath.Join(env.root, "sample", "axiom.yaml")
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func (env editEnvironment) editStates(t *testing.T) []string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(env.state, "projects", env.projectID, ".axiom-edit-*"))
	if err != nil {
		t.Fatal(err)
	}
	return names
}

func (env editEnvironment) assertNoEditState(t *testing.T) {
	t.Helper()
	if names := env.editStates(t); len(names) != 0 {
		t.Fatalf("edit recovery state remains: %v", names)
	}
}

// writeRecord replaces the local record with a valid enriched record, the way
// an out-of-band machine-local writer would.
func (env editEnvironment) writeRecord(t *testing.T, mutate func(*local.RecordState)) {
	t.Helper()
	state := env.record(t)
	mutate(&state)
	record, issues := local.NewRecord(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, issues := local.EncodeRecord(record)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := os.WriteFile(env.installationPath(), wire, 0o600); err != nil {
		t.Fatal(err)
	}
}

func enrichLocalMetadata(state *local.RecordState) {
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	state.Credentials = []projectapp.CredentialBinding{{ReferenceKey: "chat-token", SourceKind: "keychain", ItemReference: "axiom/preserved-credential-item"}}
	state.Runtime = projectapp.RuntimeBinding{RuntimeID: "codex", ExplicitPath: "/opt/preserved-runtime-path", Observation: projectapp.Observation{Availability: projectapp.Available, Basis: projectapp.PresentMetadata, ObservedAt: at}}
	state.Attempt = projectapp.AttemptMetadata{Correlation: "preserved-attempt-correlation", At: at}
}

func (env editEnvironment) withFault(fault func(local.EditStage) error) cli.Service {
	service := env.service.(lifecycleService)
	service.editFault = fault
	return service
}

func bindingPaths(state local.RecordState) map[string]string {
	paths := map[string]string{}
	for _, binding := range state.Repositories {
		paths[binding.RepositoryKey] = binding.ExplicitPath
	}
	return paths
}

func TestEditReplayPublishesNameProviderAndRepositoriesPreservingEverythingElse(t *testing.T) {
	env := newEditEnvironment(t)
	env.writeRecord(t, enrichLocalMetadata)
	priorRecord := env.record(t)
	priorPortable, issues := manifest.Decode(readBytes(t, env.manifestPath()))
	if len(issues) != 0 {
		t.Fatal(issues)
	}

	env.publish(t, "--project", "sample", "--name", "Renamed")
	published, issues := manifest.Decode(readBytes(t, env.manifestPath()))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	expected := priorPortable.State()
	expected.Name = "Renamed"
	want, issues := project.New(expected)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if published.State().Name != "Renamed" || !want.Equivalent(published) {
		t.Fatalf("name publication changed other portable fields:\n%s", readBytes(t, env.manifestPath()))
	}
	after := env.record(t)
	if !jsonEqual(t, after.Repositories, priorRecord.Repositories) || !jsonEqual(t, after.Credentials, priorRecord.Credentials) || !jsonEqual(t, after.Runtime, priorRecord.Runtime) ||
		!jsonEqual(t, after.Attempt, priorRecord.Attempt) || !jsonEqual(t, after.Documentation, priorRecord.Documentation) || after.SourceLocation != priorRecord.SourceLocation {
		t.Fatal("name publication changed unrelated local state")
	}
	if after.PortableRevision == priorRecord.PortableRevision {
		t.Fatal("local record does not bind the published portable revision")
	}
	runCanonicalCLI(t, env.service, []string{"project", "show", "--selector", "sample"}, cli.ExitSuccess, "success", "Project resolved")
	if noop, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed"); len(noop.Edit.Effects) != 0 {
		t.Fatalf("published state is not coherent: %v", noop.Edit.Effects)
	}

	env.publish(t, "--project", env.projectID, "--work-item-provider", "linear")
	if !strings.Contains(string(readBytes(t, env.manifestPath())), "linear") {
		t.Fatal("provider set was not published")
	}
	env.publish(t, "--project", "sample", "--remove-work-item-provider")
	if strings.Contains(string(readBytes(t, env.manifestPath())), "work-items") {
		t.Fatal("provider removal was not published")
	}

	docs := filepath.Join(t.TempDir(), "docs")
	moved := filepath.Join(t.TempDir(), "web-moved")
	for _, path := range []string{docs, moved} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	env.publish(t, "--project", "sample", "--repository", "docs="+docs)
	env.publish(t, "--project", "sample", "--repository", "web="+moved)
	env.publish(t, "--project", "sample", "--remove-repository", "api")
	final := env.record(t)
	if paths := bindingPaths(final); len(paths) != 2 || paths["docs"] != docs || paths["web"] != moved {
		t.Fatalf("bindings = %v", paths)
	}
	wire := string(readBytes(t, env.manifestPath()))
	if strings.Contains(wire, "key: api") || !strings.Contains(wire, "key: docs") || !strings.Contains(wire, "key: web") || strings.Contains(wire, docs) {
		t.Fatalf("portable associations:\n%s", wire)
	}
	if !jsonEqual(t, final.Credentials, priorRecord.Credentials) || !jsonEqual(t, final.Runtime, priorRecord.Runtime) || !jsonEqual(t, final.Attempt, priorRecord.Attempt) {
		t.Fatal("repository publications changed unrelated local metadata")
	}
	runCanonicalCLI(t, env.service, []string{"project", "show", "--selector", env.projectID}, cli.ExitSuccess, "success", "Project resolved")
}

func jsonEqual(t *testing.T, a, b any) bool {
	t.Helper()
	left, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(left, right)
}
