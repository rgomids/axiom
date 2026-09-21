package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutableGuidedProjectConfiguration(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "lingo")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v: %s", err, output)
	}
	portable := filepath.Join(t.TempDir(), "portable")
	state := filepath.Join(t.TempDir(), "state")
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "--json", "project", "configure")
	command.Env = append(os.Environ(), "LINGO_PROJECTS_ROOT="+portable, "LINGO_STATE_ROOT="+state)
	command.Stdin = strings.NewReader("guided\nGuided Project\nmain\n" + repository + "\n")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("guided configure: %v: stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	var event cliEvent
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &event); err != nil || event.Category != "project_configured" {
		t.Fatalf("guided event = %+v, %v; output=%s", event, err, stdout.String())
	}
	if !strings.Contains(stderr.String(), "Repository path") {
		t.Fatalf("guided prompts missing: %q", stderr.String())
	}
}

func TestExecutableVersionHumanJSONAndBuildProvenance(t *testing.T) {
	releaseBinary := filepath.Join(t.TempDir(), "lingo-release")
	build := exec.Command("go", "build", "-ldflags", "-X main.buildVersion=1.2.3 -X main.buildRevision=abc123def456 -X main.buildSourceState=clean -X main.buildRelease=true", "-o", releaseBinary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build release executable: %v: %s", err, output)
	}
	human, err := exec.Command(releaseBinary, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("human version: %v: %s", err, human)
	}
	for _, expected := range []string{"status: success", "result: Axiom build information", "provenance: Axiom 1.2.3 revision=abc123def456 source=clean"} {
		if !bytes.Contains(human, []byte(expected)) {
			t.Fatalf("human version missing %q: %s", expected, human)
		}
	}
	structured, err := exec.Command(releaseBinary, "--json", "version").CombinedOutput()
	if err != nil {
		t.Fatalf("JSON version: %v: %s", err, structured)
	}
	var release canonicalEvent
	if err := json.Unmarshal(bytes.TrimSpace(structured), &release); err != nil {
		t.Fatalf("JSON version: %v: %s", err, structured)
	}
	if release.Status != "success" || release.Result != "Axiom build information" || release.Provenance.Product != "Axiom" || release.Provenance.Version != "1.2.3" || release.Provenance.Revision != "abc123def456" || release.Provenance.SourceState != "clean" {
		t.Fatalf("release provenance = %+v", release)
	}

	dirtyBinary := filepath.Join(t.TempDir(), "lingo-dirty")
	build = exec.Command("go", "build", "-ldflags", "-X main.buildVersion=development -X main.buildRevision=def456abc123 -X main.buildSourceState=dirty -X main.buildRelease=false", "-o", dirtyBinary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build dirty executable: %v: %s", err, output)
	}
	dirtyOutput, err := exec.Command(dirtyBinary, "--json", "version").CombinedOutput()
	if err != nil {
		t.Fatalf("dirty version: %v: %s", err, dirtyOutput)
	}
	var dirty canonicalEvent
	if err := json.Unmarshal(bytes.TrimSpace(dirtyOutput), &dirty); err != nil {
		t.Fatalf("dirty JSON version: %v: %s", err, dirtyOutput)
	}
	if dirty.Provenance.Version != "development" || dirty.Provenance.Revision != "def456abc123" || dirty.Provenance.SourceState != "dirty" {
		t.Fatalf("dirty provenance = %+v", dirty.Provenance)
	}
}

type cliEvent struct {
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Category  string `json:"category"`
	Project   *struct {
		Slug         string `json:"slug"`
		Repositories []struct {
			Key, Path string
		} `json:"repositories"`
	} `json:"project"`
	Workflow *struct {
		Status, CurrentGate, RepositoryPath string
	} `json:"workflow"`
	WorkItem *struct {
		URL, State string
		Number     int
	} `json:"workItem"`
}

type canonicalEvent struct {
	Status     string   `json:"status"`
	Result     string   `json:"result"`
	References []string `json:"references"`
	Provenance struct {
		Product, Version, Revision, SourceState string
	} `json:"provenance"`
}

func TestExecutableMinimalLifecycleAndFailurePaths(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "lingo")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v: %s", err, output)
	}
	portable := filepath.Join(t.TempDir(), "portable")
	state := filepath.Join(t.TempDir(), "state")
	skills := filepath.Join(t.TempDir(), "skills")
	environment := append(os.Environ(), "LINGO_PROJECTS_ROOT="+portable, "LINGO_STATE_ROOT="+state, "AXIOM_CODEX_SKILLS_ROOT="+skills)
	run := func(wantCode int, wantStatus, wantCategory string, args ...string) cliEvent {
		t.Helper()
		command := exec.Command(binary, append([]string{"--json"}, args...)...)
		command.Env = environment
		output, err := command.CombinedOutput()
		code := 0
		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("%v: process error: %v", args, err)
			}
			code = exit.ExitCode()
		}
		var event cliEvent
		if err := json.Unmarshal(bytes.TrimSpace(output), &event); err != nil {
			t.Fatalf("%v: invalid JSON event %q: %v", args, output, err)
		}
		if code != wantCode || event.Status != wantStatus || event.Category != wantCategory {
			t.Fatalf("%v: code=%d event=%+v", args, code, event)
		}
		return event
	}
	runCanonical := func(wantCode int, wantStatus, wantResult string, args ...string) canonicalEvent {
		t.Helper()
		command := exec.Command(binary, append([]string{"--json"}, args...)...)
		command.Env = environment
		output, err := command.CombinedOutput()
		code := 0
		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("%v: process error: %v", args, err)
			}
			code = exit.ExitCode()
		}
		var event canonicalEvent
		if err := json.Unmarshal(bytes.TrimSpace(output), &event); err != nil {
			t.Fatalf("%v: invalid canonical JSON %q: %v", args, output, err)
		}
		if code != wantCode || event.Status != wantStatus || event.Result != wantResult || event.Provenance.Product != "Axiom" {
			t.Fatalf("%v: code=%d event=%+v", args, code, event)
		}
		return event
	}
	run(0, "success", "codex_configured", "runtime", "codex", "install")
	run(0, "success", "codex_ready", "runtime", "codex", "status")
	installed, err := filepath.Glob(filepath.Join(skills, "axiom-*", "SKILL.md"))
	if err != nil || len(installed) != 5 {
		t.Fatalf("installed Codex skills = %v, %v", installed, err)
	}
	repository := filepath.Join(t.TempDir(), "configured-repository")
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	gitBinary := filepath.Join(t.TempDir(), "git")
	ghBinary := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(gitBinary, []byte("#!/bin/sh\nprintf '%s\\n' 'git@github.com:owner/repo.git'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ghScript := `#!/bin/sh
if [ "$1" = issue ] && [ "$2" = create ]; then printf '%s\n' 'https://github.com/owner/repo/issues/7'; exit 0; fi
if [ "$1" = issue ] && [ "$2" = view ]; then printf '%s\n' '{"Number":7,"URL":"https://github.com/owner/repo/issues/7","State":"CLOSED"}'; exit 0; fi
if [ "$1" = issue ] && { [ "$2" = comment ] || [ "$2" = close ]; }; then printf '%s\n' ok; exit 0; fi
exit 1
`
	if err := os.WriteFile(ghBinary, []byte(ghScript), 0o700); err != nil {
		t.Fatal(err)
	}
	environment = append(environment, "AXIOM_GIT_BIN="+gitBinary, "AXIOM_GH_BIN="+ghBinary)
	run(0, "success", "project_configured", "project", "configure", "--slug", "configured", "--name", "Configured", "--repository", "main="+repository)
	run(0, "success", "project_resolved", "project", "resolve", "--selector", "configured")
	shown := runCanonical(0, "success", "Project resolved", "project", "show", "--selector", "configured")
	if len(shown.References) != 2 || !strings.HasPrefix(shown.References[0], "project:") || shown.References[1] != "repository:main" {
		t.Fatalf("project references = %+v", shown.References)
	}
	run(1, "error", "external_mutation_denied", "work-item", "create", "--project", "configured", "--repository", "main", "--title", "POC")
	created := run(0, "success", "work_item_linked", "work-item", "create", "--project", "configured", "--repository", "main", "--title", "POC", "--authorize-external")
	if created.WorkItem == nil || created.WorkItem.Number != 7 || created.WorkItem.State != "OPEN" {
		t.Fatalf("work item payload = %+v", created.WorkItem)
	}
	run(0, "success", "work_item_loaded", "work-item", "show", "--project", "configured", "--repository", "main", "--number", "7")
	run(0, "success", "work_item_commented", "work-item", "comment", "--project", "configured", "--repository", "main", "--number", "7", "--message", "Evidence", "--authorize-external")
	run(0, "success", "workflow_started", "workflow", "start", "--project", "configured", "--repository", "main", "--number", "7")
	for _, gate := range []string{"specification", "clarification", "plan", "tasks"} {
		if err := os.WriteFile(filepath.Join(repository, gate+".md"), []byte(gate), 0o600); err != nil {
			t.Fatal(err)
		}
		run(0, "success", "workflow_advanced", "workflow", "advance", "--project", "configured", "--repository", "main", "--number", "7", "--gate", gate, "--outcome", "pass", "--reference", gate+".md")
	}
	if err := os.WriteFile(filepath.Join(repository, "implementation.md"), []byte("implementation"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(1, "error", "workflow_interrupted", "workflow", "advance", "--project", "configured", "--repository", "main", "--number", "7", "--gate", "implementation", "--outcome", "fail", "--reference", "implementation.md")
	interrupted := run(0, "success", "workflow_interrupted", "workflow", "status", "--project", "configured", "--repository", "main", "--number", "7")
	if interrupted.Workflow == nil || interrupted.Workflow.Status != "interrupted" || interrupted.Workflow.CurrentGate != "implementation" || interrupted.Workflow.RepositoryPath != repository {
		t.Fatalf("workflow payload = %+v", interrupted.Workflow)
	}
	run(0, "success", "workflow_resumed", "workflow", "resume", "--project", "configured", "--repository", "main", "--number", "7")
	for _, gate := range []string{"implementation", "review", "evidence", "reconciliation"} {
		if err := os.WriteFile(filepath.Join(repository, gate+".md"), []byte(gate), 0o600); err != nil {
			t.Fatal(err)
		}
		run(0, "success", "workflow_advanced", "workflow", "advance", "--project", "configured", "--repository", "main", "--number", "7", "--gate", gate, "--outcome", "pass", "--reference", gate+".md")
	}
	run(0, "success", "workflow_evidence_ready", "workflow", "evidence", "--project", "configured", "--repository", "main", "--number", "7")
	run(1, "error", "external_mutation_denied", "workflow", "advance", "--project", "configured", "--repository", "main", "--number", "7", "--gate", "completion", "--outcome", "pass")
	run(0, "success", "workflow_completed", "workflow", "advance", "--project", "configured", "--repository", "main", "--number", "7", "--gate", "completion", "--outcome", "pass", "--authorize-external")
	run(1, "error", "missing_required_input", "project", "init", "--slug", "sample")
	run(0, "success", "applied", "project", "init", "--slug", "sample", "--name", "Sample")
	run(0, "success", "already_initialized", "project", "init", "--slug", "sample", "--name", "Sample")
	run(1, "error", "explicit_update_required", "project", "init", "--slug", "sample", "--name", "Other")
	runCanonical(0, "success", "Project is valid", "project", "validate", "--slug", "sample")
	run(0, "success", "reopened_without_local_state", "project", "reopen", "--slug", "sample")
	source := filepath.Join(portable, "sample")
	before, err := os.ReadFile(filepath.Join(source, "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	run(0, "success", "installed", "project", "install", "--source", source)
	run(0, "success", "already_installed", "project", "install", "--source", source)
	run(0, "success", "reopened_with_local_state", "project", "reopen", "--slug", "sample")
	after, err := os.ReadFile(filepath.Join(source, "axiom.yaml"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("install changed portable bytes: %v", err)
	}
	records, err := filepath.Glob(filepath.Join(state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 2 {
		t.Fatalf("local records = %v, %v", records, err)
	}
	run(0, "success", "applied", "project", "update", "--slug", "sample", "--name", "Changed")
	run(1, "error", "local_state_revalidation_required", "project", "reopen", "--slug", "sample")
	runCanonical(1, "validation_failure", "Project state is invalid", "project", "validate", "--slug", "missing")
}
