package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

func TestComposedCLICompletesMinimalPortableLifecycle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()

	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	runCLI(t, service, []string{"project", "validate", "--slug", "sample"}, cli.ExitSuccess, "valid")
	runCLI(t, service, []string{"project", "reopen", "--slug", "sample"}, cli.ExitSuccess, "reopened_without_local_state")
	beforeInstall, err := os.ReadFile(filepath.Join(root, "sample", "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "install", "--source", filepath.Join(root, "sample")}, cli.ExitSuccess, "installed")
	runCLI(t, service, []string{"project", "reopen", "--slug", "sample"}, cli.ExitSuccess, "reopened_with_local_state")
	afterInstall, err := os.ReadFile(filepath.Join(root, "sample", "axiom.yaml"))
	if err != nil || string(beforeInstall) != string(afterInstall) {
		t.Fatalf("install changed portable manifest: %v", err)
	}
	records, err := filepath.Glob(filepath.Join(state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 1 {
		t.Fatalf("installation record paths = %v, %v", records, err)
	}
	runCLI(t, service, []string{"project", "update", "--slug", "sample", "--name", "Changed"}, cli.ExitSuccess, "applied")

	manifest, err := os.ReadFile(filepath.Join(root, "sample", "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "name: Changed") {
		t.Fatalf("updated manifest does not contain new name: %q", manifest)
	}
}

func TestComposedCLIRejectsRelativeRoot(t *testing.T) {
	t.Setenv("LINGO_PROJECTS_ROOT", "relative")
	var output bytes.Buffer
	if code := cli.Run(context.Background(), []string{"project", "init", "--slug", "sample", "--name", "Sample"}, compose(), &output); code != cli.ExitFailure {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(output.String(), "application_unavailable") {
		t.Fatalf("unexpected error event: %q", output.String())
	}
}

func TestConfigurePublishesPortableKeysAndLocalPathsThenResolves(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "configure", "--slug", "configured", "--name", "Configured", "--repository", "main=" + repository}, cli.ExitSuccess, "project_configured")
	runCLI(t, service, []string{"project", "configure", "--slug", "configured", "--name", "Configured", "--repository", "main=" + repository}, cli.ExitSuccess, "project_already_configured")
	runCLI(t, service, []string{"project", "resolve", "--selector", "configured"}, cli.ExitSuccess, "project_resolved")
	manifestBytes, err := os.ReadFile(filepath.Join(root, "configured", "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	manifestText := string(manifestBytes)
	if !strings.Contains(manifestText, "key: main") || strings.Contains(manifestText, repository) {
		t.Fatalf("portable/local boundary violated: %s", manifestText)
	}
	records, err := filepath.Glob(filepath.Join(state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %v, %v", records, err)
	}
	recordBytes, err := os.ReadFile(records[0])
	if err != nil || !bytes.Contains(recordBytes, []byte(repository)) {
		t.Fatalf("local repository binding absent: %s, %v", recordBytes, err)
	}
}

func TestConfigureInvalidRepositoriesPublishNothing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "configure", "--slug", "configured", "--name", "Configured", "--repository", "main=" + repository, "--repository", "main=" + repository}, cli.ExitFailure, "invalid_input")
	if _, err := os.Stat(filepath.Join(root, "configured")); !os.IsNotExist(err) {
		t.Fatalf("invalid configuration published portable state: %v", err)
	}
}

func TestConfigureReportsCommittedPortableStateWhenLocalPublicationFails(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	if err := os.Symlink(t.TempDir(), state); err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "configure", "--slug", "configured", "--name", "Configured", "--repository", "main=" + repository}, cli.ExitFailure, "portable_committed_local_failed")
	if _, err := os.Stat(filepath.Join(root, "configured", "axiom.yaml")); err != nil {
		t.Fatalf("committed portable state not reported truthfully: %v", err)
	}
}

func TestVersionReportsAxiomSourceMetadata(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "version")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if code := writeVersion(file); code != cli.ExitSuccess {
		t.Fatalf("version exit code = %d", code)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"product":"Axiom"`, `"binary":"lingo"`, `"version":"devel"`, `"commit":"unknown"`, `"dirty":false`, `"dirtyKnown":false`, `"source":"https://github.com/rgomids/axiom"`} {
		if !bytes.Contains(data, []byte(expected)) {
			t.Fatalf("version output missing %q: %s", expected, data)
		}
	}
}

func TestWorkItemFailureRendersCommittedExternalState(t *testing.T) {
	result := workItemResult(workitem.Result{
		Status:   workitem.Failed,
		Category: "provider_committed_local_failed",
		Link: workitem.Link{
			ProjectID:          "123e4567-e89b-42d3-a456-426614174000",
			RepositoryKey:      "main",
			ProviderRepository: "owner/repo",
			Number:             7,
			URL:                "https://github.com/owner/repo/issues/7",
			State:              "CLOSED",
		},
	})
	if result.WorkItem == nil || result.WorkItem.ProjectID == "" || result.WorkItem.RepositoryKey != "main" || result.WorkItem.Repository != "owner/repo" || result.WorkItem.State != "CLOSED" {
		t.Fatalf("result = %#v", result)
	}
	payload, err := json.Marshal(result.WorkItem)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"projectId":"123e4567-e89b-42d3-a456-426614174000"`, `"repositoryKey":"main"`, `"repository":"owner/repo"`, `"number":7`, `"url":"https://github.com/owner/repo/issues/7"`, `"state":"CLOSED"`} {
		if !bytes.Contains(payload, []byte(expected)) {
			t.Fatalf("work item payload missing %q: %s", expected, payload)
		}
	}
}

func TestWorkflowFailureRendersCommittedExternalState(t *testing.T) {
	result := workflowResult(workflow.Result{
		Status:   workflow.Failed,
		Category: "provider_committed_local_failed",
		WorkItem: &workflow.WorkItem{
			ProjectID:          "123e4567-e89b-42d3-a456-426614174000",
			RepositoryKey:      "main",
			ProviderRepository: "owner/repo",
			Number:             7,
			URL:                "https://github.com/owner/repo/issues/7",
			State:              "CLOSED",
		},
	})
	if result.WorkItem == nil || result.WorkItem.ProjectID == "" || result.WorkItem.RepositoryKey != "main" || result.WorkItem.Repository != "owner/repo" || result.WorkItem.State != "CLOSED" {
		t.Fatalf("result = %#v", result)
	}
}

func TestStateRootOverride(t *testing.T) {
	t.Setenv("LINGO_STATE_ROOT", "/tmp/lingo-state/../lingo-state")
	got, err := stateRoot()
	if err != nil || got != "/tmp/lingo-state" {
		t.Fatalf("stateRoot override = %q, %v", got, err)
	}
}

func TestStateRootRejectsInvalidOverride(t *testing.T) {
	for _, value := range []string{"", "relative/state", "/"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("LINGO_STATE_ROOT", value)
			if root, err := stateRoot(); err == nil || root != "" {
				t.Fatalf("stateRoot accepted invalid override: %q, %v", root, err)
			}
		})
	}
}

func TestCompositionRejectsOverlappingRootsBeforeCreation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", filepath.Join(root, "state"))
	var output bytes.Buffer
	if code := cli.Run(context.Background(), []string{"project", "init", "--slug", "sample", "--name", "Sample"}, compose(), &output); code != cli.ExitFailure {
		t.Fatalf("overlapping roots accepted: code=%d", code)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("portable root created before root validation: %v", err)
	}
}

func TestCompositionRejectsSymlinkAliasBeforeCreation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", filepath.Join(alias, "state"))
	var output bytes.Buffer
	if code := cli.Run(context.Background(), []string{"project", "init", "--slug", "sample", "--name", "Sample"}, compose(), &output); code != cli.ExitFailure {
		t.Fatalf("alias accepted: %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "state")); !os.IsNotExist(err) {
		t.Fatalf("state created through alias: %v", err)
	}
}

func TestInstallRejectsUnsafeSourceWithoutChangingPermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	source := filepath.Join(root, "sample")
	if err := os.Chmod(source, 0o755); err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitFailure, "unsafe_source")
	info, err := os.Stat(source)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("install mutated source mode: %v, %v", info, err)
	}
}

func TestInstallRejectsSymlinkRecordWithoutWritingOutside(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	source := filepath.Join(root, "sample")
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitSuccess, "installed")
	records, err := filepath.Glob(filepath.Join(state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 1 {
		t.Fatalf("records: %v, %v", records, err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(records[0]); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, records[0]); err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitFailure, "invalid_existing_local_state")
	if data, err := os.ReadFile(outside); err != nil || string(data) != "keep" {
		t.Fatalf("outside changed: %q, %v", data, err)
	}
}

func TestInstallRejectsHardLinkedRecordWithoutWritingOutside(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	source := filepath.Join(root, "sample")
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitSuccess, "installed")
	records, err := filepath.Glob(filepath.Join(state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 1 {
		t.Fatalf("records: %v, %v", records, err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(records[0]); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(outside, records[0]); err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitFailure, "invalid_existing_local_state")
	if data, err := os.ReadFile(outside); err != nil || string(data) != "keep" {
		t.Fatalf("outside changed: %q, %v", data, err)
	}
}

func TestInstallPreservesUnknownLocalArtifact(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	source := filepath.Join(root, "sample")
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitSuccess, "installed")
	records, err := filepath.Glob(filepath.Join(state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 1 {
		t.Fatalf("records: %v, %v", records, err)
	}
	before, err := os.ReadFile(records[0])
	if err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(filepath.Dir(records[0]), "unknown")
	if err := os.WriteFile(unknown, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitFailure, "invalid_existing_local_state")
	runCLI(t, service, []string{"project", "reopen", "--slug", "sample"}, cli.ExitFailure, "invalid_existing_local_state")
	if after, err := os.ReadFile(records[0]); err != nil || !bytes.Equal(before, after) {
		t.Fatalf("record changed: %v", err)
	}
	if data, err := os.ReadFile(unknown); err != nil || string(data) != "keep" {
		t.Fatalf("unknown artifact changed: %q, %v", data, err)
	}
}

func TestValidateDoesNotCreateRootsOrLockFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "validate", "--slug", "missing"}, cli.ExitFailure, "project_not_found")
	for _, path := range []string{root, state} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("validate created %q: %v", path, err)
		}
	}
	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	before, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "validate", "--slug", "sample"}, cli.ExitSuccess, "valid")
	after, err := os.ReadDir(root)
	if err != nil || len(after) != len(before) {
		t.Fatalf("validate changed root entries: %v, %v", after, err)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("validate created local state: %v", err)
	}
}

func TestInterruptedInstallRequiresRecoveryAndPreservesPortableBytes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "projects")
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("LINGO_PROJECTS_ROOT", root)
	t.Setenv("LINGO_STATE_ROOT", state)
	service := compose()
	runCLI(t, service, []string{"project", "init", "--slug", "sample", "--name", "Sample"}, cli.ExitSuccess, "applied")
	source := filepath.Join(root, "sample")
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitSuccess, "installed")
	manifest, err := os.ReadFile(filepath.Join(source, "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	records, err := filepath.Glob(filepath.Join(state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %v, %v", records, err)
	}
	stage := filepath.Join(filepath.Dir(records[0]), ".lingo-install-interrupted")
	if err := os.WriteFile(stage, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	runCLI(t, service, []string{"project", "reopen", "--slug", "sample"}, cli.ExitFailure, "recovery_required")
	runCLI(t, service, []string{"project", "install", "--source", source}, cli.ExitFailure, "recovery_required")
	if current, err := os.ReadFile(filepath.Join(source, "axiom.yaml")); err != nil || !bytes.Equal(current, manifest) {
		t.Fatalf("portable bytes changed: %v", err)
	}
	if current, err := os.ReadFile(stage); err != nil || string(current) != "partial" {
		t.Fatalf("unknown attempt removed: %q, %v", current, err)
	}
}

func runCLI(t *testing.T, service cli.Service, args []string, wantCode int, wantCategory string) {
	t.Helper()
	var output bytes.Buffer
	if code := cli.Run(context.Background(), args, service, &output); code != wantCode {
		t.Fatalf("%v: exit code = %d, output=%q", args, code, output.String())
	}
	if !strings.Contains(output.String(), `"category":"`+wantCategory+`"`) {
		t.Fatalf("%v: category absent from %q", args, output.String())
	}
}
