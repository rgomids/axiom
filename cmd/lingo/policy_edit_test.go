package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

type policyEvent struct {
	Status    string                      `json:"status"`
	Result    string                      `json:"result"`
	Policy    projectapp.PolicyEditReport `json:"policyEdit"`
	Readiness *projectapp.ReadinessReport `json:"readiness"`
}

const policyCLIDraft = `{"runtimes":{"mode":"replace","values":[{"id":"codex"}]},"modelProfiles":{"mode":"replace","values":[{"key":"worker","runtimeRef":"codex","model":"synthetic-model"}]},"runtimePreferences":{"mode":"replace","values":[{"role":"implementation","complexity":"high","modelProfileRef":"worker"}]}}`

func runPolicyCLI(t *testing.T, env editEnvironment, want int, args ...string) (policyEvent, string) {
	t.Helper()
	var output bytes.Buffer
	code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output)
	var event policyEvent
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err, output.String())
	}
	if code != want {
		t.Fatal(code, event, output.String())
	}
	for _, sentinel := range []string{env.root, env.state, env.api, env.web, "portableManifest", "SENTINEL-POLICY-SECRET"} {
		if strings.Contains(output.String(), sentinel) {
			t.Fatal("unsafe output", sentinel)
		}
	}
	return event, output.String()
}

func TestPolicyCLIJourneyGuidedParityReadbackAndValidate(t *testing.T) {
	env := newEditEnvironment(t)
	args := []string{"project", "configure", "--project", "sample", "--policy-draft", policyCLIDraft}
	before := snapshotTrees(t, env.root, env.state)
	preview, _ := runPolicyCLI(t, env, cli.ExitSuccess, args...)
	if preview.Policy.Outcome != projectapp.EditPreviewed || preview.Policy.Preview.Runtime[0].Code != "configuration_missing" {
		t.Fatal(preview)
	}
	var guided bytes.Buffer
	if code := cli.RunInteractive(context.Background(), append([]string{"--json"}, args...), env.service, currentProvenance(), strings.NewReader("no\n"), &guided, io.Discard); code != cli.ExitSuccess {
		t.Fatal(guided.String())
	}
	var event policyEvent
	if err := json.Unmarshal(guided.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(event.Policy, preview.Policy) || !bytes.Equal(before, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("guided preview differs or wrote state")
	}
	authorized := append(append([]string{}, args...), "--project-id", preview.Policy.Preview.ProjectID, "--preview-digest", preview.Policy.Preview.Digest, "--authorize-local")
	applied, _ := runPolicyCLI(t, env, cli.ExitSuccess, authorized...)
	if applied.Policy.Outcome != projectapp.EditPublished || applied.Policy.ReadBack == nil || applied.Policy.ReadBack.Status != "confirmed" {
		t.Fatal(applied)
	}
	validated, _ := runPolicyCLI(t, env, cli.ExitSuccess, "project", "validate", "--project", env.projectID)
	if validated.Readiness == nil || validated.Readiness.Runtime.Detail != applied.Policy.ReadBack.Observed.Runtime[0].Code || !reflect.DeepEqual(validated.Readiness.Operations, applied.Policy.ReadBack.Observed.Readiness) {
		t.Fatal("validate/read-back parity failed", validated)
	}
	read, _ := runPolicyCLI(t, env, cli.ExitSuccess, "project", "configure", "--project", env.projectID, "--policy-draft", "{}")
	if len(read.Policy.Preview.Diff) != 0 || len(read.Policy.Preview.Effects) != 0 || len(read.Policy.Preview.Policy.ModelProfiles.Values) != 1 {
		t.Fatal(read)
	}
	bytesBefore := snapshotTrees(t, env.root, env.state)
	runPolicyCLI(t, env, cli.ExitFailure, authorized...)
	if !bytes.Equal(bytesBefore, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("stale replay wrote state")
	}
	noop := []string{"project", "configure", "--project", env.projectID, "--policy-draft", "{}", "--project-id", env.projectID, "--preview-digest", read.Policy.Preview.Digest, "--authorize-local"}
	unchanged, _ := runPolicyCLI(t, env, cli.ExitSuccess, noop...)
	if unchanged.Policy.Outcome != projectapp.EditUnchanged || !bytes.Equal(bytesBefore, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("no-op changed state", unchanged)
	}
}

func TestPolicyCLIRejectsUnsafeMixedOrIncompleteInputs(t *testing.T) {
	env := newEditEnvironment(t)
	before := snapshotTrees(t, env.root, env.state)
	for _, suffix := range [][]string{
		{"--policy-draft", `{"runtime":"SENTINEL-POLICY-SECRET"}`},
		{"--policy-draft", `{"runtimes":{"mode":"replace","values":[{"id":"/private/SENTINEL-POLICY-SECRET"}]}}`},
		{"--policy-draft", "{"},
		{"--policy-draft", strings.Repeat(" ", projectapp.MaxPolicyDraftBytes+1)},
		{"--policy-draft", "{}", "--name", "Changed"},
		{"--policy-draft", "{}", "--authorize-local"},
		{"--policy-draft", "{}", "--policy-draft", "{}"},
	} {
		runPolicyCLI(t, env, cli.ExitFailure, append([]string{"project", "configure", "--project", "sample"}, suffix...)...)
		if !bytes.Equal(before, snapshotTrees(t, env.root, env.state)) {
			t.Fatal("refusal wrote state")
		}
	}
	runPolicyCLI(t, env, cli.ExitFailure, "project", "configure", "--policy-draft", "{}")
}

func TestPolicyCLIRefusesTamperedSchemasWithoutAdoption(t *testing.T) {
	for _, version := range []string{"1", "2", "4", "99"} {
		t.Run(version, func(t *testing.T) {
			env := newEditEnvironment(t)
			path := filepath.Join(env.root, "sample", "axiom.yaml")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			original = bytes.Replace(original, []byte("schemaVersion: 3"), []byte("schemaVersion: "+version), 1)
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			if version == "2" {
				runPolicyCLI(t, env, cli.ExitFailure, "project", "configure", "--project", "sample", "--policy-draft", policyCLIDraft)
			} else {
				runPolicyCLI(t, env, cli.ExitFailure, "project", "configure", "--project", "sample", "--policy-draft", "{}")
			}
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(original, current) {
				t.Fatal("refused schema changed", err)
			}
		})
	}
}

func TestPolicyCLIV2InstalledJourney(t *testing.T) {
	env := newEditEnvironment(t)
	store, err := local.NewPortableStore(env.root)
	if err != nil {
		t.Fatal(err)
	}
	wire := []byte("schemaVersion: 2\nproject:\n  id: 123e4567-e89b-42d3-a456-426614174000\n  slug: portable-two\n  name: Portable two\n")
	if err := store.Create(context.Background(), "portable-two", wire); err != nil {
		t.Fatal(err)
	}
	installation, err := local.NewInstallationStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	if result := installation.Install(context.Background(), filepath.Join(env.root, "portable-two")); result.Status != local.InstallationApplied {
		t.Fatal(result)
	}
	args := []string{"project", "configure", "--project", "portable-two", "--policy-draft", policyCLIDraft}
	preview, _ := runPolicyCLI(t, env, cli.ExitSuccess, args...)
	applied, _ := runPolicyCLI(t, env, cli.ExitSuccess, append(args, "--project-id", preview.Policy.Preview.ProjectID, "--preview-digest", preview.Policy.Preview.Digest, "--authorize-local")...)
	if applied.Policy.Outcome != projectapp.EditPublished || applied.Policy.ReadBack.Status != "confirmed" || applied.Policy.ReadBack.Observed.SchemaVersion != 2 {
		t.Fatal(applied)
	}
}
