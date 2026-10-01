package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #132 I132-T01 Evidence: EDIT is a zero-write preview over the real
// portable/local stores; CREATE against a configured slug fails closed.

type editEnvironment struct {
	root, state   string
	api, web      string
	service       cli.Service
	projectID     string
	createPreview projectapp.SetupPreview
}

func newEditEnvironment(t *testing.T) editEnvironment {
	t.Helper()
	env := editEnvironment{
		root:  filepath.Join(t.TempDir(), "projects"),
		state: filepath.Join(t.TempDir(), "state"),
		api:   filepath.Join(t.TempDir(), "api"),
		web:   filepath.Join(t.TempDir(), "web"),
	}
	for _, path := range []string{env.api, env.web} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("LINGO_PROJECTS_ROOT", env.root)
	t.Setenv("LINGO_STATE_ROOT", env.state)
	env.service = compose()
	env.createPreview = configureProjectWith(t, env.service, "sample", "Sample", "github", "api="+env.api, "web="+env.web)
	env.projectID = env.createPreview.ProjectID
	return env
}

func configureProjectWith(t *testing.T, service cli.Service, slug, name, provider string, repositories ...string) projectapp.SetupPreview {
	t.Helper()
	args := []string{"project", "configure", "--slug", slug, "--name", name, "--work-item-provider", provider}
	for _, repository := range repositories {
		args = append(args, "--repository", repository)
	}
	var output bytes.Buffer
	if code := cli.Run(context.Background(), args, service, currentProvenance(), &output); code != cli.ExitSuccess {
		t.Fatalf("create preview: %s", output.String())
	}
	var event struct {
		Setup projectapp.SetupPreview `json:"setup"`
	}
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	authorized := append(args, "--project-id", event.Setup.ProjectID, "--preview-digest", event.Setup.Digest, "--authorize-local")
	if code := cli.Run(context.Background(), authorized, service, currentProvenance(), &output); code != cli.ExitSuccess {
		t.Fatalf("create publish: %s", output.String())
	}
	return event.Setup
}

type editEvent struct {
	Status     string                 `json:"status"`
	Result     string                 `json:"result"`
	Next       string                 `json:"next"`
	References []string               `json:"references"`
	Edit       projectapp.EditPreview `json:"edit"`
}

// runEdit executes one EDIT request and proves that portable and local roots
// are byte-identical before and after it.
func (env editEnvironment) runEdit(t *testing.T, wantCode int, wantStatus, wantResult string, args ...string) (editEvent, string) {
	t.Helper()
	before := snapshotTrees(t, env.root, env.state)
	var output bytes.Buffer
	code := cli.Run(context.Background(), append([]string{"project", "configure"}, args...), env.service, currentProvenance(), &output)
	if after := snapshotTrees(t, env.root, env.state); !bytes.Equal(before, after) {
		t.Fatalf("%v wrote state\nbefore=%s\nafter=%s", args, before, after)
	}
	var event editEvent
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("%v: %v: %s", args, err, output.String())
	}
	if code != wantCode || event.Status != wantStatus || event.Result != wantResult {
		t.Fatalf("%v: code=%d event=%+v output=%s", args, code, event, output.String())
	}
	return event, output.String()
}

func effectList(effects []projectapp.EditEffect) []string {
	result := []string{}
	for _, effect := range effects {
		value := effect.Scope + ":" + effect.Code
		if effect.Key != "" {
			value += ":" + effect.Key
		}
		result = append(result, value)
	}
	return result
}

func TestEditPreviewBySlugAndUUIDIsCompleteAndZeroWrite(t *testing.T) {
	env := newEditEnvironment(t)
	bySlug, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed")
	byID, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", env.projectID, "--name", "Renamed")
	if !reflect.DeepEqual(bySlug.Edit, byID.Edit) {
		t.Fatalf("slug and UUID produced different previews:\n%+v\n%+v", bySlug.Edit, byID.Edit)
	}
	preview := bySlug.Edit
	if preview.Mode != "edit" || preview.ProjectID != env.projectID || preview.Slug != "sample" || preview.Name != "Renamed" || preview.Capability.Provider != "github" {
		t.Fatalf("preview = %+v", preview)
	}
	if preview.PortableDestination != filepath.Join(env.root, "sample") || preview.LocalDestination != filepath.Join(env.state, "projects", env.projectID) {
		t.Fatalf("destinations = %s %s", preview.PortableDestination, preview.LocalDestination)
	}
	portable, err := local.NewPortableStore(env.root)
	if err != nil {
		t.Fatal(err)
	}
	observed, err := portable.Inspect(context.Background(), "sample")
	if err != nil || preview.PortableRevision != observed.Revision {
		t.Fatalf("portable revision %s != observed %s (%v)", preview.PortableRevision, observed.Revision, err)
	}
	for _, expected := range []string{"name: Renamed", "slug: sample", "key: api", "key: web", "work-items"} {
		if !strings.Contains(preview.PortableManifest, expected) {
			t.Fatalf("portable result missing %q:\n%s", expected, preview.PortableManifest)
		}
	}
	if strings.Contains(preview.PortableManifest, env.api) {
		t.Fatal("portable candidate carries a machine-local path")
	}
	if got := effectList(preview.Effects); !reflect.DeepEqual(got, []string{"portable:update_portable_project", "local:update_local_record"}) {
		t.Fatalf("effects = %v", got)
	}
	for _, repository := range preview.Repositories {
		if repository.Change != projectapp.RepositoryPreserved || repository.LocalPath != "" {
			t.Fatalf("omitted repository exposed or changed: %+v", repository)
		}
	}
	noop, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample")
	if len(noop.Edit.Effects) != 0 || noop.Next != "No change is required" || noop.Edit.Name != "Sample" {
		t.Fatalf("selector-only edit = %+v", noop)
	}
}

func TestEditPreviewRepositoryAndProviderSemanticsOverRealStores(t *testing.T) {
	env := newEditEnvironment(t)
	docs := filepath.Join(t.TempDir(), "docs")
	moved := filepath.Join(t.TempDir(), "web-moved")
	for _, path := range []string{docs, moved} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	event, output := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready",
		"--project", "sample", "--remove-repository", "api", "--repository", "web="+moved, "--repository", "docs="+docs, "--remove-work-item-provider")
	want := []string{
		"portable:update_portable_project", "portable:remove_portable_repository:api", "portable:add_portable_repository:docs",
		"local:update_local_record", "local:remove_local_binding:api", "local:add_local_binding:docs", "local:update_local_binding:web",
	}
	if got := effectList(event.Edit.Effects); !reflect.DeepEqual(got, want) {
		t.Fatalf("effects = %v", got)
	}
	if event.Edit.Capability.Provider != "" || event.Edit.Capability.Readiness != projectapp.CapabilityMissing || strings.Contains(event.Edit.PortableManifest, "work-items") {
		t.Fatalf("provider removal not reflected: %+v", event.Edit.Capability)
	}
	if strings.Contains(event.Edit.PortableManifest, "key: api") || !strings.Contains(event.Edit.PortableManifest, "key: docs") {
		t.Fatalf("portable associations wrong:\n%s", event.Edit.PortableManifest)
	}
	if !strings.Contains(output, moved) || !strings.Contains(output, docs) {
		t.Fatal("changed local bindings are not reviewable")
	}

	env.runEdit(t, cli.ExitFailure, "validation_failure", "Repository to remove is not configured", "--project", "sample", "--remove-repository", "unknown")
	env.runEdit(t, cli.ExitFailure, "validation_failure", "Repository path is unavailable", "--project", "sample", "--repository", "docs="+filepath.Join(t.TempDir(), "absent"))
	for _, conflict := range [][]string{
		{"--project", "sample", "--repository", "api=" + env.api, "--remove-repository", "api"},
		{"--project", "sample", "--work-item-provider", "linear", "--remove-work-item-provider"},
		{"--project", "sample", "--slug", "renamed"},
	} {
		env.runEdit(t, cli.ExitFailure, "validation_failure", "Explicit selector input is invalid", conflict...)
	}
	linear, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--work-item-provider", "linear")
	if linear.Edit.Capability.Provider != "linear" || linear.Edit.Capability.Readiness != projectapp.CapabilityUnsupported || linear.Edit.Name != "Sample" {
		t.Fatalf("provider set = %+v", linear.Edit)
	}
}

func TestEditSelectorFailuresNeverFallBackToCreate(t *testing.T) {
	env := newEditEnvironment(t)
	env.runEdit(t, cli.ExitFailure, "validation_failure", "Project was not found", "--project", "missing", "--name", "Missing", "--repository", "api="+env.api)
	if _, err := os.Stat(filepath.Join(env.root, "missing")); !os.IsNotExist(err) {
		t.Fatalf("unknown EDIT selector created a Project: %v", err)
	}
	env.runEdit(t, cli.ExitFailure, "validation_failure", "Project identity does not match the selector", "--project", "sample", "--project-id", "123e4567-e89b-42d3-a456-426614174999")

	// A second installed record observing the same slug is representable and
	// must fail as ambiguous rather than choosing either Project.
	other := "123e4567-e89b-42d3-a456-426614174001"
	record, issues := local.NewRecord(local.RecordState{
		ProjectID: other, ObservedSlug: "sample", SourceLocation: filepath.Join(env.root, "sample"),
		PortableRevision: projectapp.RecordedPortableRevision([32]byte{1}),
		ArtifactDigests:  []projectapp.ArtifactDigest{{Name: "axiom.yaml", Digest: [32]byte{1}}},
	})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	wire, issues := local.EncodeRecord(record)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := os.MkdirAll(filepath.Join(env.state, "projects", other), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(env.state, "projects", other, "installation.json"), wire, 0o600); err != nil {
		t.Fatal(err)
	}
	env.runEdit(t, cli.ExitFailure, "validation_failure", "Project selector is ambiguous", "--project", "sample", "--name", "Renamed")
	uuid, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", env.projectID, "--name", "Renamed")
	if uuid.Edit.ProjectID != env.projectID {
		t.Fatal("UUID did not disambiguate the selection")
	}
}

func TestEditPreviewRepairsBrokenBindingWithoutAvailabilityGate(t *testing.T) {
	env := newEditEnvironment(t)
	if err := os.Remove(env.web); err != nil {
		t.Fatal(err)
	}
	runCanonicalCLI(t, env.service, []string{"project", "show", "--selector", "sample"}, cli.ExitFailure, "retryable_failure", "Project repository is unavailable")
	removed, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--remove-repository", "web")
	if got := effectList(removed.Edit.Effects); !reflect.DeepEqual(got, []string{"portable:update_portable_project", "portable:remove_portable_repository:web", "local:update_local_record", "local:remove_local_binding:web"}) {
		t.Fatalf("broken binding removal effects = %v", got)
	}
	replacement := filepath.Join(t.TempDir(), "web-replacement")
	if err := os.Mkdir(replacement, 0o700); err != nil {
		t.Fatal(err)
	}
	repaired, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--repository", "web="+replacement)
	if got := effectList(repaired.Edit.Effects); !reflect.DeepEqual(got, []string{"local:update_local_record", "local:update_local_binding:web"}) {
		t.Fatalf("broken binding repair effects = %v", got)
	}
}

func TestEditPreviewHidesUnrelatedLocalMetadataButBindsIt(t *testing.T) {
	env := newEditEnvironment(t)
	path := filepath.Join(env.state, "projects", env.projectID, "installation.json")
	base, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--remove-repository", "api")

	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	record, issues := local.DecodeRecord(wire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	state := record.State()
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	state.Credentials = []projectapp.CredentialBinding{{ReferenceKey: "chat-token", SourceKind: "keychain", ItemReference: "axiom/hidden-credential-item"}}
	state.Runtime = projectapp.RuntimeBinding{RuntimeID: "codex", ExplicitPath: "/opt/hidden-runtime-path", Observation: projectapp.Observation{Availability: projectapp.Available, Basis: projectapp.PresentMetadata, ObservedAt: at}}
	state.Attempt = projectapp.AttemptMetadata{Correlation: "hidden-attempt-correlation", At: at}
	enriched, issues := local.NewRecord(state)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	enrichedWire, issues := local.EncodeRecord(enriched)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	if err := os.WriteFile(path, enrichedWire, 0o600); err != nil {
		t.Fatal(err)
	}

	event, output := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--remove-repository", "api")
	for _, hidden := range []string{"hidden-credential-item", "hidden-runtime-path", "hidden-attempt-correlation", env.web, "observedAt", "present_metadata"} {
		if strings.Contains(output, hidden) {
			t.Fatalf("preview exposed unrelated local metadata %q: %s", hidden, output)
		}
	}
	if event.Edit.Digest == base.Edit.Digest || event.Edit.LocalRevision == base.Edit.LocalRevision {
		t.Fatal("digest/observation did not bind hidden local metadata")
	}
	if !reflect.DeepEqual(event.Edit.Effects, base.Edit.Effects) || event.Edit.PortableManifest != base.Edit.PortableManifest {
		t.Fatal("hidden metadata changed visible candidate")
	}
}

func TestEditAuthorityFlagsFailClosedWithoutWrites(t *testing.T) {
	env := newEditEnvironment(t)
	preview, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed")
	event, _ := env.runEdit(t, cli.ExitFailure, "validation_failure", "Project edit publication is not available",
		"--project", "sample", "--name", "Renamed", "--project-id", env.projectID, "--preview-digest", preview.Edit.Digest, "--authorize-local")
	if event.Edit.Digest != preview.Edit.Digest {
		t.Fatal("authorized request did not rebuild the same preview")
	}
}

func TestEditFailsClosedOnStalePortableLocalRelationship(t *testing.T) {
	env := newEditEnvironment(t)
	// The legacy update changes only the portable manifest, leaving the local
	// record's recorded portable revision stale.
	runCLI(t, env.service, []string{"project", "update", "--slug", "sample", "--name", "Changed"}, cli.ExitSuccess, "applied")
	env.runEdit(t, cli.ExitFailure, "failure", "Selected Project state is not safe to edit", "--project", "sample", "--name", "Renamed")
}

func TestCreateCollisionFailsWithoutReuseOrEditFallback(t *testing.T) {
	env := newEditEnvironment(t)
	for name, args := range map[string][]string{
		"equivalent values": {"--slug", "sample", "--name", "Sample", "--work-item-provider", "github", "--repository", "api=" + env.api, "--repository", "web=" + env.web},
		"different values":  {"--slug", "sample", "--name", "Other", "--work-item-provider", "none", "--repository", "api=" + env.api},
		"existing identity": {"--slug", "sample", "--name", "Sample", "--work-item-provider", "github", "--repository", "api=" + env.api, "--repository", "web=" + env.web, "--project-id", env.projectID},
		"authorized replay": {"--slug", "sample", "--name", "Sample", "--work-item-provider", "github", "--repository", "api=" + env.api, "--repository", "web=" + env.web, "--project-id", env.projectID, "--preview-digest", env.createPreview.Digest, "--authorize-local"},
	} {
		t.Run(name, func(t *testing.T) {
			event, output := env.runEdit(t, cli.ExitFailure, "validation_failure", "Project slug is already configured", args...)
			if strings.Contains(output, `"edit"`) || strings.Contains(output, `"setup"`) || strings.Contains(output, env.projectID) || len(event.References) != 0 {
				t.Fatalf("collision reused or exposed existing Project state: %s", output)
			}
		})
	}
	records, err := filepath.Glob(filepath.Join(env.state, "projects", "*", "installation.json"))
	if err != nil || len(records) != 1 {
		t.Fatalf("collision allocated another identity: %v %v", records, err)
	}
}
