package main

import (
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"os"
	"path/filepath"
	"testing"
)

func installStagePlanPolicy(t *testing.T, stateRoot, id string) {
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
	if wire, err = os.ReadFile(source); err != nil {
		t.Fatal(err)
	}
	value, manifestIssues := manifest.Decode(wire)
	if len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	state := value.State()
	if state.SchemaVersion < 2 {
		t.Fatal("fixture requires a Runtime allowlist schema")
	}
	state.Runtimes = project.Configured([]project.Runtime{{ID: "codex"}, {ID: "claude"}})
	state.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "profile-codex", RuntimeRef: project.Configured("codex"), Model: project.Configured("local-codex-model")}, {Key: "profile-claude", RuntimeRef: project.Configured("claude"), Model: project.Configured("local-claude-model")}})
	state.RuntimePreferences = project.Configured([]project.RuntimePreference{{Role: "engineer", Complexity: "medium", ModelProfileRef: "profile-codex"}, {Role: "integrator", Complexity: "medium", ModelProfileRef: "profile-codex"}})
	state.BusinessContext = project.Configured(project.BusinessContext{Text: project.Configured("Synthetic bounded business context")})
	if value, manifestIssues = project.New(state); len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	if wire, manifestIssues = manifest.Encode(value); len(manifestIssues) != 0 {
		t.Fatal(manifestIssues)
	}
	if err := os.WriteFile(source, wire, 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, readIssues := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(readIssues) != 0 {
		t.Fatal(readIssues)
	}
	recordState.PortableRevision, recordState.ArtifactDigests = snapshot.Revision(), snapshot.Digests()
	if record, issues = local.NewRecord(recordState); len(issues) != 0 {
		t.Fatal(issues)
	}
	if wire, issues = local.EncodeRecord(record); len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := os.WriteFile(recordPath, wire, 0o600); err != nil {
		t.Fatal(err)
	}
	capabilities := []string{"axiom-skills", "read", "repository-write"}
	cfg := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 1}
	for _, runtimeID := range []string{"codex", "claude"} {
		cfg.Runtimes = append(cfg.Runtimes, runtimeprofile.RuntimeV3{ID: runtimeID, Adapter: runtimeID, Enabled: true, AllowlistedProfileIDs: []string{"profile-" + runtimeID}})
		cfg.ModelProfiles = append(cfg.ModelProfiles, runtimeprofile.ModelProfile{ID: "profile-" + runtimeID, RuntimeID: runtimeID, Model: "local-" + runtimeID + "-model", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, Capabilities: capabilities, Complexities: []string{"medium", "high"}})
	}
	publishRuntimeProfileV3(t, stateRoot, cfg)
}
