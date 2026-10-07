package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Fixtures explicitly authorize a portable policy and its matching installed
// snapshot; configuration alone must never regain the old implicit fallback.
func installTestRuntimePolicy(t *testing.T, stateRoot, id, runtimeID string) []string {
	t.Helper()
	recordPath := filepath.Join(stateRoot, "projects", id, "installation.json")
	wire, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	record, issues := local.DecodeRecord(wire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	recordState := record.State()
	source := filepath.Join(recordState.SourceLocation, "axiom.yaml")
	wire, err = os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	value, manifestIssues := manifest.Decode(wire)
	if len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	state := value.State()
	state.Runtime = project.Configured(project.Runtime{ID: runtimeID})
	state.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "worker", RuntimeRef: project.Configured(runtimeID), Model: project.Configured("approved-model")}})
	value, manifestIssues = project.New(state)
	if len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	wire, manifestIssues = manifest.Encode(value)
	if len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	if err := os.WriteFile(source, wire, 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, readIssues := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(readIssues) != 0 {
		t.Fatal(readIssues)
	}
	recordState.PortableRevision = snapshot.Revision()
	recordState.ArtifactDigests = snapshot.Digests()
	record, issues = local.NewRecord(recordState)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, issues = local.EncodeRecord(record)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := os.WriteFile(recordPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := local.NewRuntimeProfileStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	cfg := runtimeprofile.Configuration{FormatVersion: 1, Revision: 1, Runtimes: []runtimeprofile.Runtime{{ID: runtimeID, Adapter: runtimeID, Enabled: true, AllowlistedProfileIDs: []string{"worker"}, CredentialReference: "private-test-reference"}}, ModelProfiles: []runtimeprofile.ModelProfile{{ID: "worker", RuntimeID: runtimeID, Model: "approved-model", Capabilities: []string{runtimeadapter.IntegrationCapability}, Complexities: []string{"high"}}}}
	if err := store.Create(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	// Observations are never supplied: Lingo reads the executable and the
	// Axiom skill integration that the caller's environment exposes.
	return []string{"--role", "implementation", "--complexity", "high", "--capabilities", runtimeadapter.IntegrationCapability}
}

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
