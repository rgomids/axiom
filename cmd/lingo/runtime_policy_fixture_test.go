package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
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
	cfg := runtimeprofile.Configuration{FormatVersion: 1, Revision: 1, Runtimes: []runtimeprofile.Runtime{{ID: runtimeID, Adapter: runtimeID, Enabled: true, AllowlistedProfileIDs: []string{"worker"}, CredentialReference: "private-test-reference"}}, ModelProfiles: []runtimeprofile.ModelProfile{{ID: "worker", RuntimeID: runtimeID, Model: "approved-model", Capabilities: []string{"code"}, Complexities: []string{"high"}}}}
	if err := store.Create(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	observations := []runtimeprofile.Observation{{RuntimeID: runtimeID, Adapter: runtimeID, Installed: true, Available: true, Version: "1.0", Revision: 1, ObservedAt: time.Unix(1, 0).UTC(), CapabilityStatus: map[string]runtimeprofile.CapabilityStatus{"code": runtimeprofile.CapabilityProven}}}
	wire, err = json.Marshal(observations)
	if err != nil {
		t.Fatal(err)
	}
	observationPath := filepath.Join(t.TempDir(), "observations.json")
	if err := os.WriteFile(observationPath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	return []string{"--role", "implementation", "--complexity", "high", "--capabilities", "code", "--observations", observationPath}
}
