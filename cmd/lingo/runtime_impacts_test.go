package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Fixture publication uses the actual portable/local codecs. It changes only
// temporary test files and reconciles the installation's exact source revision.
func installRuntimeImpactPolicy(t *testing.T, stateRoot, id string) {
	t.Helper()
	recordPath := filepath.Join(stateRoot, "projects", id, "installation.json")
	record, issues := local.DecodeRecord(readBytes(t, recordPath))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	state := record.State()
	source := filepath.Join(state.SourceLocation, "axiom.yaml")
	value, portableIssues := manifest.Decode(readBytes(t, source))
	if len(portableIssues) != 0 {
		t.Fatal(portableIssues)
	}
	portable := value.State()
	portable.Runtimes = project.Configured([]project.Runtime{{ID: "codex"}})
	portable.ModelProfiles = project.Configured([]project.ModelProfile{{Key: "shared", RuntimeRef: project.Configured("codex"), Model: project.Configured("shared-model")}})
	value, portableIssues = project.New(portable)
	if len(portableIssues) != 0 {
		t.Fatal(portableIssues)
	}
	wire, portableIssues := manifest.Encode(value)
	if len(portableIssues) != 0 {
		t.Fatal(portableIssues)
	}
	if err := os.WriteFile(source, wire, 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, snapshotIssues := projectapp.ReadSnapshot(manifest.Codec{}, wire, nil)
	if len(snapshotIssues) != 0 {
		t.Fatal(snapshotIssues)
	}
	state.PortableRevision, state.ArtifactDigests = snapshot.Revision(), snapshot.Digests()
	record, issues = local.NewRecord(state)
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
}
func runtimeImpactEnvironment(t *testing.T) (editEnvironment, lifecycleService, string) {
	t.Helper()
	env := newEditEnvironment(t)
	second := configureProjectWith(t, env.service, "second", "Second", "github", "api="+env.api)
	for _, id := range []string{env.projectID, second.ProjectID} {
		installRuntimeImpactPolicy(t, env.state, id)
	}
	service, ok := env.service.(lifecycleService)
	if !ok {
		t.Fatal("unexpected service")
	}
	return env, service, second.ProjectID
}
func runtimeImpactConfiguration() runtimeprofile.Configuration {
	return runtimeprofile.Configuration{FormatVersion: 2, Revision: 1,
		Runtimes:      []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"shared", "survivor"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "shared", RuntimeID: "codex", Model: "shared-model", Capabilities: []string{"read"}, Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}}, {ID: "survivor", RuntimeID: "codex", Model: "survivor-model", Capabilities: []string{"read"}, Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}}},
	}
}
func TestRuntimeImpactsSharedProjectsPreservedAfterRemoval(t *testing.T) {
	env, service, secondID := runtimeImpactEnvironment(t)
	ctx := context.Background()
	preview := runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", secondID)
	if preview.Operational == nil {
		t.Fatal("missing archive preview")
	}
	runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", secondID, "--preview-digest", preview.Operational.Digest, "--authorize-local")
	store, err := local.NewRuntimeProfileStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := store.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := runtimeprofile.Encode(runtimeImpactConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(ctx, fresh.Token, wire); err != nil {
		t.Fatal(err)
	}
	before := snapshotTrees(t, env.root, filepath.Join(env.state, "projects"))
	app := runtimeapplication.Authoring{Store: runtimeAuthoringStore{store}, Impacts: service.runtimeImpacts}
	req := runtimeapplication.AuthoringRequest{Kind: "profile", Operation: "remove", ProfileID: "shared"}
	report, err := app.Run(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.KnownImpacts) != 2 {
		t.Fatalf("known impacts %v", report.KnownImpacts)
	}
	seen := map[string]bool{}
	for _, impact := range report.KnownImpacts {
		if impact.ProfileID != "shared" {
			t.Fatal("wrong reference")
		}
		seen[impact.ProjectID] = true
	}
	if !seen[env.projectID] || !seen[secondID] {
		t.Fatal("archived installed Project omitted")
	}
	if !strings.Contains(report.ImpactInventory, "outside catalog excluded") {
		t.Fatal("unbound inventory limit undisclosed")
	}
	req.ExpectedRevision, req.PreviewDigest, req.AuthorizeLocal = report.ExpectedRevision, report.PreviewDigest, true
	applied, err := app.Run(ctx, req)
	if err != nil || applied.Category != "applied" {
		t.Fatalf("apply category %s error %v", applied.Category, err)
	}
	if after := snapshotTrees(t, env.root, filepath.Join(env.state, "projects")); !bytes.Equal(before, after) {
		t.Fatal("local Profile removal modified Project bytes/records/status")
	}
	// Resolution checks the real Project/local intersection before any executable
	// observation. Neither Project is patched to substitute the surviving Profile.
	source := runtimePolicySource{installation: service.installation, portable: service.portable, stateRoot: env.state, runtimes: runtimeObservationSources{lookPath: func(string) (string, error) { return "", exec.ErrNotFound }}}
	policy := runtimeapplication.New(source)
	for _, id := range []string{env.projectID, secondID} {
		if code := policy.CurrentAvailability(ctx, id); code != "no_allowed_match" {
			t.Fatalf("removed profile for %s blocker %s", id, code)
		}
	}
	output, err := json.Marshal(applied)
	if err != nil {
		t.Fatal(err)
	}
	for _, hidden := range []string{env.root, env.state, env.api, env.web, "installation.json", "axiom.yaml"} {
		if strings.Contains(string(output), hidden) {
			t.Fatal("impact report leaked source/installation path")
		}
	}
}
func TestRuntimeImpactsInaccessibleAndUnboundInventory(t *testing.T) {
	for _, scenario := range []string{"inaccessible-source", "unbound-repository", "unknown-catalog-record"} {
		t.Run(scenario, func(t *testing.T) {
			env, service, _ := runtimeImpactEnvironment(t)
			switch scenario {
			case "inaccessible-source":
				if err := os.Rename(filepath.Join(env.root, "second"), filepath.Join(env.root, "second-inaccessible")); err != nil {
					t.Fatal(err)
				}
			case "unbound-repository":
				if err := os.Rename(env.api, env.api+"-unavailable"); err != nil {
					t.Fatal(err)
				}
			case "unknown-catalog-record":
				if err := os.Mkdir(filepath.Join(env.state, "projects", "unknown-private-canary"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			before := snapshotTrees(t, env.root, env.state)
			impacts, classification := service.runtimeImpacts(context.Background())
			if scenario == "unbound-repository" {
				if len(impacts) != 2 || !strings.Contains(classification, "outside catalog excluded") {
					t.Fatalf("bounded installed references %v %s", impacts, classification)
				}
			} else if !strings.HasPrefix(classification, "incomplete:") {
				t.Fatalf("unavailable inventory claimed complete: %s", classification)
			}
			output, _ := json.Marshal(struct {
				Impacts   []runtimeapplication.LocalImpact
				Inventory string
			}{impacts, classification})
			for _, hidden := range []string{env.root, env.state, env.api, "private-canary"} {
				if strings.Contains(string(output), hidden) {
					t.Fatal("inventory diagnostic leaked private source")
				}
			}
			if after := snapshotTrees(t, env.root, env.state); !bytes.Equal(before, after) {
				t.Fatal("impact inspection wrote files")
			}
		})
	}
}

func TestRuntimeImpactsCurrentReadinessAndMissingConfigurationRouting(t *testing.T) {
	env, service, _ := runtimeImpactEnvironment(t)
	ctx := context.Background()
	lookups := 0
	service.runtimes.lookPath = func(string) (string, error) { lookups++; return "", exec.ErrNotFound }
	before := snapshotTrees(t, env.root, env.state)
	if code := (runtimeReadiness{service}).Availability(ctx, env.projectID); code != "configuration_missing" {
		t.Fatalf("missing local configuration blocker %s", code)
	}
	if lookups != 0 {
		t.Fatal("missing local configuration observed vendor executable")
	}
	if after := snapshotTrees(t, env.root, env.state); !bytes.Equal(before, after) {
		t.Fatal("missing configuration readiness wrote files")
	}
	var output bytes.Buffer
	if code := cli.Run(ctx, []string{"runtime", "profile", "validate"}, env.service, currentProvenance(), &output); code != cli.ExitFailure {
		t.Fatalf("missing validate exit %d", code)
	}
	if !strings.Contains(output.String(), "axiom-runtime") {
		t.Fatalf("missing configuration did not route to axiom-runtime: %s", output.String())
	}
	for _, path := range []string{env.root, env.state} {
		if strings.Contains(output.String(), path) {
			t.Fatal("missing configuration exposed source path")
		}
	}
	store, err := local.NewRuntimeProfileStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	legacy := runtimeImpactConfiguration()
	legacy.FormatVersion = 1
	for index := range legacy.ModelProfiles {
		legacy.ModelProfiles[index].Effort = runtimeprofile.Effort{}
	}
	if err := store.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	before = snapshotTrees(t, env.root, env.state)
	if code := (runtimeReadiness{service}).Availability(ctx, env.projectID); code != "profile_effort_unconfigured" {
		t.Fatalf("historical local1 admitted as current readiness: %s", code)
	}
	if after := snapshotTrees(t, env.root, env.state); !bytes.Equal(before, after) {
		t.Fatal("local1 readiness implicitly migrated local or Project state")
	}
}
