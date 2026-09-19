package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type cliEvent struct {
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Category  string `json:"category"`
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
	run := func(wantCode int, wantStatus, wantCategory string, args ...string) {
		t.Helper()
		command := exec.Command(binary, args...)
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
	}
	run(0, "success", "codex_configured", "runtime", "codex", "install")
	run(0, "success", "codex_ready", "runtime", "codex", "status")
	installed, err := filepath.Glob(filepath.Join(skills, "axiom-*", "SKILL.md"))
	if err != nil || len(installed) != 5 {
		t.Fatalf("installed Codex skills = %v, %v", installed, err)
	}
	run(1, "error", "missing_required_input", "project", "init", "--slug", "sample")
	run(0, "success", "applied", "project", "init", "--slug", "sample", "--name", "Sample")
	run(0, "success", "already_initialized", "project", "init", "--slug", "sample", "--name", "Sample")
	run(1, "error", "explicit_update_required", "project", "init", "--slug", "sample", "--name", "Other")
	run(0, "success", "valid", "project", "validate", "--slug", "sample")
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
	if err != nil || len(records) != 1 {
		t.Fatalf("local records = %v, %v", records, err)
	}
	run(0, "success", "applied", "project", "update", "--slug", "sample", "--name", "Changed")
	run(1, "error", "local_state_revalidation_required", "project", "reopen", "--slug", "sample")
	run(1, "error", "project_not_found", "project", "validate", "--slug", "missing")
}
