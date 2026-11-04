package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
)

// testRuntime gives an in-process service one observable Runtime: a stub
// executable that is never run and a verified Axiom skill integration.
func testRuntime(t *testing.T, service lifecycleService, runtimeID string) (lifecycleService, string) {
	t.Helper()
	executable := filepath.Join(t.TempDir(), runtimeID)
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 99\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	skills := filepath.Join(t.TempDir(), "skills")
	integration, err := codexruntime.New(skills)
	if runtimeID == "claude" {
		integration, err = codexruntime.NewClaude(skills)
	}
	if err != nil {
		t.Fatal(err)
	}
	if result := integration.Install(context.Background()); result.Status != codexruntime.Applied {
		t.Fatalf("install %s skills: %+v", runtimeID, result)
	}
	service.runtimes.lookPath = func(name string) (string, error) {
		if name == runtimeID {
			return executable, nil
		}
		return "", exec.ErrNotFound
	}
	if runtimeID == "claude" {
		service.runtimes.claudeSkillsRoot, service.runtimes.claudeSkillsError = skills, nil
	} else {
		service.codex = integration
	}
	return service, executable
}
