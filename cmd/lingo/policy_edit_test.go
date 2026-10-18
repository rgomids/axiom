package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
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

func TestExecutablePolicyPreviewApplyAndReadBack(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "axiom")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatal(err, string(output))
	}
	env := newEditEnvironment(t)
	run := func(want int, args ...string) policyEvent {
		t.Helper()
		command := exec.Command(binary, append([]string{"--json"}, args...)...)
		command.Dir = t.TempDir()
		command.Env = append(os.Environ(), "LINGO_PROJECTS_ROOT="+env.root, "LINGO_STATE_ROOT="+env.state)
		var output, stderr bytes.Buffer
		command.Stdout, command.Stderr = &output, &stderr
		err := command.Run()
		code := 0
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		if code != want || stderr.Len() != 0 {
			t.Fatal(code, output.String(), stderr.String())
		}
		for _, sentinel := range []string{env.root, env.state, env.api, env.web, "SENTINEL-POLICY-SECRET"} {
			if bytes.Contains(output.Bytes(), []byte(sentinel)) {
				t.Fatal("executable output leaked data")
			}
		}
		var event policyEvent
		if err := json.Unmarshal(output.Bytes(), &event); err != nil {
			t.Fatal(err, output.String())
		}
		return event
	}
	before := snapshotTrees(t, env.root, env.state)
	args := []string{"project", "configure", "--project", env.projectID, "--policy-draft", policyCLIDraft}
	preview := run(cli.ExitSuccess, args...)
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("executable preview wrote state")
	}
	run(cli.ExitFailure, append(append([]string{}, args...), "--authorize-local")...)
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("partial authority wrote state")
	}
	authorized := append(append([]string{}, args...), "--project-id", env.projectID, "--preview-digest", preview.Policy.Preview.Digest, "--authorize-local")
	applied := run(cli.ExitSuccess, authorized...)
	if applied.Policy.Outcome != projectapp.EditPublished || applied.Policy.ReadBack.Status != "confirmed" {
		t.Fatal(applied)
	}
	validated := run(cli.ExitSuccess, "project", "validate", "--project", env.projectID)
	if !reflect.DeepEqual(validated.Readiness.Operations, applied.Policy.ReadBack.Observed.Readiness) {
		t.Fatal("executable read-back differs from validate")
	}
	after := snapshotTrees(t, env.root, env.state)
	run(cli.ExitFailure, authorized...)
	if !bytes.Equal(after, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("stale executable replay wrote state")
	}
	t.Logf("synthetic project=%s schema=3 before_portable=%s before_local=%s preview=%s observed_portable=%s observed_local=%s publication=%s read_back=%s writes=manifest,installation", env.projectID, preview.Policy.Preview.PortableRevision, preview.Policy.Preview.LocalRevision, preview.Policy.Preview.Digest, applied.Policy.ReadBack.Observed.PortableRevision, applied.Policy.ReadBack.Observed.LocalRevision, applied.Policy.Publication, applied.Policy.ReadBack.Status)
}

func TestPolicyCLIReadinessDoesNotAcquireRuntimeProof(t *testing.T) {
	env := newEditEnvironment(t)
	service := env.service.(lifecycleService)
	service.runtimes.lookPath = func(string) (string, error) {
		t.Fatal("policy readiness attempted executable discovery")
		return "", os.ErrPermission
	}
	service.RuntimePolicySource = forbiddenPolicySource{test: t}
	env.service = service
	args := []string{"project", "configure", "--project", env.projectID, "--policy-draft", policyCLIDraft}
	preview, _ := runPolicyCLI(t, env, cli.ExitSuccess, args...)
	runPolicyCLI(t, env, cli.ExitSuccess, append(args, "--project-id", env.projectID, "--preview-digest", preview.Policy.Preview.Digest, "--authorize-local")...)
	runPolicyCLI(t, env, cli.ExitSuccess, "project", "validate", "--project", env.projectID)
}

type forbiddenPolicySource struct{ test *testing.T }

func (s forbiddenPolicySource) Load(context.Context, string) (runtimeapplication.Snapshot, error) {
	s.test.Fatal("policy readiness attempted live Runtime policy observation")
	return runtimeapplication.Snapshot{}, os.ErrPermission
}
