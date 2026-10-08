package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T04 black-box Evidence: the Integration group inspects,
// validates, locally disables/enables and previews the portable removal of
// Integrations through the real stores. Disable/enable is machine-local
// eligibility only; remove is portable declaration removal only; neither
// touches credentials, Runtime configuration or the Provider.

type integrationEvent struct {
	Status       string                         `json:"status"`
	Result       string                         `json:"result"`
	Category     string                         `json:"category"`
	Next         string                         `json:"next"`
	Integrations *projectapp.IntegrationReport  `json:"integrations"`
	Operational  *projectapp.OperationalPreview `json:"operational"`
	Edit         *projectapp.EditPreview        `json:"edit"`
	Admission    *projectapp.AdmissionDecision  `json:"admission"`
}

func runIntegrationCLI(t *testing.T, service cli.Service, wantCode int, args ...string) (integrationEvent, string) {
	t.Helper()
	var output bytes.Buffer
	code := cli.Run(context.Background(), append([]string{"integration"}, args...), service, currentProvenance(), &output)
	var event integrationEvent
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("%v: %v: %s", args, err, output.String())
	}
	if code != wantCode {
		t.Fatalf("%v: exit=%d want=%d output=%s", args, code, wantCode, output.String())
	}
	return event, output.String()
}

// zeroWrites runs one operation and proves portable, local and Repository
// state are byte-identical afterwards.
func zeroWrites(t *testing.T, env admissionEnv, run func()) {
	t.Helper()
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	run()
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("operation wrote state")
	}
	if env.providerCalled() {
		t.Fatal("operation reached the Provider")
	}
}

// stateExcludingOperational is every state file except the machine-local
// operational record of the Project.
func stateExcludingOperational(t *testing.T, state string) []byte {
	t.Helper()
	var out bytes.Buffer
	err := filepath.Walk(state, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Base(path) == "operational.json" {
			return nil
		}
		relative, _ := filepath.Rel(state, path)
		out.WriteString(relative + ":" + info.Mode().String())
		if info.Mode().IsRegular() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out.Write(content)
		}
		out.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func listed(report *projectapp.IntegrationReport) []string {
	keys := []string{}
	for _, view := range report.Integrations {
		keys = append(keys, view.Key+":"+view.Local)
	}
	return keys
}

func TestIntegrationInspectionOnConfiguredUnconfiguredAndV1Projects(t *testing.T) {
	env := newAdmissionEnv(t)
	repository := writeTree(t, filepath.Join(env.workspace, "bare"), map[string]string{"README.md": "x"})
	configureProject(t, env.service, "bare", "Bare", "main="+repository, "")
	// A schema v1 Project (project init) installed from outside the root.
	sourceRoot := filepath.Join(t.TempDir(), "elsewhere")
	t.Setenv("LINGO_PROJECTS_ROOT", sourceRoot)
	runCLI(t, compose(), []string{"project", "init", "--slug", "legacy", "--name", "Legacy"}, cli.ExitSuccess, "applied")
	t.Setenv("LINGO_PROJECTS_ROOT", env.root)
	runCLI(t, env.service, []string{"project", "install", "--source", filepath.Join(sourceRoot, "legacy")}, cli.ExitSuccess, "installed")

	zeroWrites(t, env, func() {
		configured, wire := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", "guarded")
		if configured.Status != "success" || configured.Integrations == nil || configured.Integrations.ProjectID != env.projectID || !reflect.DeepEqual(listed(configured.Integrations), []string{"work-items:enabled"}) {
			t.Fatalf("configured list = %+v", configured)
		}
		view := configured.Integrations.Integrations[0]
		if view.Provider != "github" || view.ProviderRef != "work-items" || !reflect.DeepEqual(view.Capabilities, []string{"work-item"}) {
			t.Fatalf("view = %+v", view)
		}
		for _, machine := range []string{env.root, env.state, env.workspace} {
			if strings.Contains(wire, machine) {
				t.Fatalf("output leaks machine path %s: %s", machine, wire)
			}
		}
		byID, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", env.projectID)
		if !reflect.DeepEqual(byID.Integrations, configured.Integrations) {
			t.Fatal("slug and UUID selection differ")
		}
		shown, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "show", "--project", "guarded", "--integration", "work-items")
		if shown.Result != "Integration resolved" || len(shown.Integrations.Integrations) != 1 {
			t.Fatalf("show = %+v", shown)
		}
		missing, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "show", "--project", "guarded", "--integration", "nope")
		if missing.Category != projectapp.IntegrationNotFound || missing.Status != string(completion.ValidationFailure) {
			t.Fatalf("unknown show = %+v", missing)
		}
		validated, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "validate", "--project", "guarded")
		if validated.Integrations.Validation == nil || validated.Integrations.Validation.Status != projectapp.IntegrationValid || len(validated.Integrations.Validation.Findings) != 0 {
			t.Fatalf("validate = %+v", validated.Integrations)
		}
		filtered, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "validate", "--project", "guarded", "--integration", "work-items")
		if len(filtered.Integrations.Integrations) != 1 {
			t.Fatalf("filtered = %+v", filtered.Integrations)
		}
		if unknown, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "validate", "--project", "guarded", "--integration", "nope"); unknown.Category != projectapp.IntegrationNotFound {
			t.Fatalf("filtered unknown = %+v", unknown)
		}

		for _, slug := range []string{"bare", "legacy"} {
			empty, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", slug)
			if empty.Result != "Project declares no Integrations" || len(empty.Integrations.Integrations) != 0 || empty.Integrations.Integrations == nil {
				t.Fatalf("%s list = %+v", slug, empty)
			}
			check, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "validate", "--project", slug)
			if check.Integrations.Validation.Status != projectapp.IntegrationValid || len(check.Integrations.Validation.Findings) != 1 || check.Integrations.Validation.Findings[0].Code != projectapp.FindingIntegrationsNotDeclared {
				t.Fatalf("%s validate = %+v", slug, check.Integrations.Validation)
			}
			if absent, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "show", "--project", slug, "--integration", "work-items"); absent.Category != projectapp.IntegrationNotFound {
				t.Fatalf("%s show = %+v", slug, absent)
			}
		}
		if unknownProject, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "list", "--project", "nonexistent"); unknownProject.Category != "project_not_found" {
			t.Fatalf("unknown Project = %+v", unknownProject)
		}
	})

	// Human rendering carries the same facts and no machine path.
	var output bytes.Buffer
	code := cli.RunInteractive(context.Background(), []string{"--human", "integration", "list", "--project", "guarded"}, env.service, currentProvenance(), nil, &output, &bytes.Buffer{})
	if code != cli.ExitSuccess || !strings.Contains(output.String(), "integration: work-items local=enabled provider=github") || strings.Contains(output.String(), env.state) || strings.Contains(output.String(), env.root) {
		t.Fatalf("human output (%d): %s", code, &output)
	}
}

func TestIntegrationDisableEnableLifecycleIsLocalAndBlocksOnlyProviderUse(t *testing.T) {
	env := newAdmissionEnv(t)
	service := env.service.(lifecycleService)
	operational := filepath.Join(env.state, "projects", env.projectID, "operational.json")
	args := func(verb string, extra ...string) []string {
		return append([]string{verb, "--project", "guarded", "--integration", "work-items"}, extra...)
	}
	rootsBefore := snapshotTrees(t, env.root, env.workspace)
	stateBefore := stateExcludingOperational(t, env.state)

	var preview integrationEvent
	zeroWrites(t, env, func() {
		preview, _ = runIntegrationCLI(t, env.service, cli.ExitSuccess, args("disable")...)
	})
	if preview.Category != projectapp.OperationalPreviewReady || preview.Operational == nil || preview.Operational.Digest == "" || preview.Operational.Boundary.Portable != "unchanged" || preview.Operational.Boundary.Provider != "none" || preview.Operational.Boundary.Credentials != "unchanged" {
		t.Fatalf("preview = %+v", preview)
	}
	if got := effectList(preview.Operational.Effects); !reflect.DeepEqual(got, []string{"local:disable_integration:work-items"}) {
		t.Fatalf("effects = %v", got)
	}

	// A stale or foreign digest never authorizes and writes nothing.
	stale := strings.Repeat("ab", 32)
	zeroWrites(t, env, func() {
		denied, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, args("disable", "--preview-digest", stale, "--authorize-local")...)
		if denied.Category != projectapp.OperationalAuthorityDenied {
			t.Fatalf("stale digest = %+v", denied)
		}
		unauthorized, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, args("disable", "--preview-digest", preview.Operational.Digest)...)
		if unauthorized.Category != projectapp.OperationalAuthorityDenied {
			t.Fatalf("digest without authority = %+v", unauthorized)
		}
	})

	applied, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, args("disable", "--preview-digest", preview.Operational.Digest, "--authorize-local")...)
	if applied.Category != projectapp.OperationalApplied || applied.Operational.Result.DisabledIntegrations[0] != "work-items" {
		t.Fatalf("applied = %+v", applied)
	}
	if _, err := os.Stat(operational); err != nil {
		t.Fatalf("operational record absent: %v", err)
	}
	// Only the machine-local operational record changed.
	if !bytes.Equal(rootsBefore, snapshotTrees(t, env.root, env.workspace)) || !bytes.Equal(stateBefore, stateExcludingOperational(t, env.state)) {
		t.Fatal("disable changed portable, Repository, credential or non-operational state")
	}

	// Disabled stays listable, showable and validatable.
	zeroWrites(t, env, func() {
		list, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", "guarded")
		if !reflect.DeepEqual(listed(list.Integrations), []string{"work-items:disabled"}) {
			t.Fatalf("list = %+v", list.Integrations)
		}
		show, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "show", "--project", "guarded", "--integration", "work-items")
		validate, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "validate", "--project", "guarded")
		if show.Integrations.Integrations[0].Local != "disabled" || validate.Integrations.Validation.Status != projectapp.IntegrationValid || len(validate.Integrations.Validation.Findings) != 1 || validate.Integrations.Validation.Findings[0].Code != projectapp.FindingIntegrationDisabledLocally {
			t.Fatalf("show=%+v validate=%+v", show.Integrations, validate.Integrations.Validation)
		}
		// The Provider is never reached: Work Item creation is denied first.
		item := cli.WorkItemInput{Project: "guarded", Repository: "main", ProviderRepository: "owner/repo", Type: "feature", Beneficiary: "user", Value: "x"}
		if denied := service.WorkItemCreate(context.Background(), item); denied.Category != projectapp.AdmissionIntegrationDisabled || denied.Admission == nil || denied.Admission.Integration != "work-items" {
			t.Fatalf("create while disabled = %+v", denied)
		}
	})

	// Disabling again is a deterministic no-op without authority.
	zeroWrites(t, env, func() {
		again, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, args("disable", "--preview-digest", stale, "--authorize-local")...)
		if again.Category != projectapp.IntegrationAlreadyDisabled || len(again.Operational.Effects) != 0 {
			t.Fatalf("no-op = %+v", again)
		}
	})

	// Enable restores admission.
	enablePreview, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, args("enable")...)
	if enablePreview.Category != projectapp.OperationalPreviewReady || effectList(enablePreview.Operational.Effects)[0] != "local:enable_integration:work-items" {
		t.Fatalf("enable preview = %+v", enablePreview)
	}
	enabled, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, args("enable", "--preview-digest", enablePreview.Operational.Digest, "--authorize-local")...)
	if enabled.Category != projectapp.OperationalApplied || len(enabled.Operational.Result.DisabledIntegrations) != 0 {
		t.Fatalf("enabled = %+v", enabled)
	}
	if after := service.WorkItemCreate(context.Background(), cli.WorkItemInput{Project: "guarded", Repository: "main", ProviderRepository: "owner/repo"}); after.Category == projectapp.AdmissionIntegrationDisabled || after.Admission != nil && after.Admission.Code == projectapp.AdmissionIntegrationDisabled {
		t.Fatalf("enable did not restore admission: %+v", after)
	}
	noop, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, args("enable", "--preview-digest", stale, "--authorize-local")...)
	if noop.Category != projectapp.IntegrationAlreadyEnabled {
		t.Fatalf("enable no-op = %+v", noop)
	}
	if !bytes.Equal(rootsBefore, snapshotTrees(t, env.root, env.workspace)) || !bytes.Equal(stateBefore, stateExcludingOperational(t, env.state)) {
		t.Fatal("enable changed portable, Repository, credential or non-operational state")
	}
}

func TestIntegrationDisableEnableRejectUndeclaredKeysAndClearStaleOnes(t *testing.T) {
	env := newAdmissionEnv(t)
	service := env.service.(lifecycleService)
	zeroWrites(t, env, func() {
		for _, verb := range []string{"disable", "enable"} {
			missing, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, verb, "--project", "guarded", "--integration", "ghost")
			if missing.Category != projectapp.IntegrationNotFound {
				t.Fatalf("%s undeclared = %+v", verb, missing)
			}
		}
		if invalid, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "disable", "--project", "guarded", "--integration", "Bad Key"); invalid.Category != projectapp.OperationalInvalidInput {
			t.Fatalf("invalid key = %+v", invalid)
		}
	})

	// A key disabled locally and no longer declared is stale: inert, listed
	// separately, never blocking, and cleared only by a local enable.
	env.apply(t, projectapp.DisableIntegration, "ghost")
	zeroWrites(t, env, func() {
		list, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", "guarded")
		if !reflect.DeepEqual(list.Integrations.StaleDisabled, []string{"ghost"}) || !reflect.DeepEqual(listed(list.Integrations), []string{"work-items:enabled"}) {
			t.Fatalf("list = %+v", list.Integrations)
		}
		if absent, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "show", "--project", "guarded", "--integration", "ghost"); absent.Category != projectapp.IntegrationNotFound {
			t.Fatalf("stale show = %+v", absent)
		}
		validate, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "validate", "--project", "guarded")
		if validate.Integrations.Validation.Status != projectapp.IntegrationValid || len(validate.Integrations.Validation.Findings) != 1 || validate.Integrations.Validation.Findings[0].Code != projectapp.FindingIntegrationStaleDisabled {
			t.Fatalf("validate = %+v", validate.Integrations.Validation)
		}
		if blocked := service.WorkItemCreate(context.Background(), cli.WorkItemInput{Project: "guarded", Repository: "main", ProviderRepository: "owner/repo"}); blocked.Category == projectapp.AdmissionIntegrationDisabled {
			t.Fatalf("stale entry blocked a different Integration: %+v", blocked)
		}
	})
	preview, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "enable", "--project", "guarded", "--integration", "ghost")
	if preview.Category != projectapp.OperationalPreviewReady {
		t.Fatalf("stale enable preview = %+v", preview)
	}
	if cleared, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "enable", "--project", "guarded", "--integration", "ghost", "--preview-digest", preview.Operational.Digest, "--authorize-local"); cleared.Category != projectapp.OperationalApplied {
		t.Fatalf("stale enable = %+v", cleared)
	}
	if list, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", "guarded"); len(list.Integrations.StaleDisabled) != 0 {
		t.Fatalf("stale entry survived: %+v", list.Integrations)
	}
}

func TestIntegrationDisableIsIndependentPerStateRoot(t *testing.T) {
	first := newAdmissionEnv(t)
	second := newAdmissionEnv(t)
	first.apply(t, projectapp.DisableIntegration, "work-items")
	if list, _ := runIntegrationCLI(t, first.service, cli.ExitSuccess, "list", "--project", "guarded"); !reflect.DeepEqual(listed(list.Integrations), []string{"work-items:disabled"}) {
		t.Fatalf("first = %+v", list.Integrations)
	}
	if list, _ := runIntegrationCLI(t, second.service, cli.ExitSuccess, "list", "--project", "guarded"); !reflect.DeepEqual(listed(list.Integrations), []string{"work-items:enabled"}) {
		t.Fatalf("second root saw the first root's disable: %+v", list.Integrations)
	}
	if first.projectID == second.projectID {
		t.Fatal("independent Projects share an identity")
	}
}

func TestIntegrationAdministrationRemainsAvailableOnArchivedProject(t *testing.T) {
	env := newAdmissionEnv(t)
	env.apply(t, projectapp.ArchiveProject, "")
	zeroWrites(t, env, func() {
		for _, args := range [][]string{
			{"list", "--project", "guarded"},
			{"show", "--project", "guarded", "--integration", "work-items"},
			{"validate", "--project", "guarded"},
			{"disable", "--project", "guarded", "--integration", "work-items"},
			{"enable", "--project", "guarded", "--integration", "work-items"},
			{"remove", "--project", "guarded", "--integration", "work-items"},
		} {
			event, output := runIntegrationCLI(t, env.service, cli.ExitSuccess, args...)
			if event.Admission != nil || event.Category == projectapp.AdmissionProjectArchived {
				t.Fatalf("%v denied by archive: %s", args, output)
			}
		}
	})
	preview, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "disable", "--project", "guarded", "--integration", "work-items")
	if applied, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "disable", "--project", "guarded", "--integration", "work-items", "--preview-digest", preview.Operational.Digest, "--authorize-local"); applied.Category != projectapp.OperationalApplied || applied.Operational.Result.ProjectStatus != projectapp.ProjectArchived {
		t.Fatalf("archived disable = %+v", applied)
	}
}

func TestIntegrationRemoveWorkItemsPreviewIsCompleteAndZeroWrite(t *testing.T) {
	env := newAdmissionEnv(t)
	env.apply(t, projectapp.DisableIntegration, "work-items")
	var preview integrationEvent
	zeroWrites(t, env, func() {
		preview, _ = runIntegrationCLI(t, env.service, cli.ExitSuccess, "remove", "--project", "guarded", "--integration", "work-items")
		byID, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "remove", "--project", env.projectID, "--integration", "work-items")
		if !reflect.DeepEqual(preview.Edit, byID.Edit) {
			t.Fatal("slug and UUID previews differ")
		}
	})
	if preview.Result != "Project edit preview ready" || preview.Edit == nil || preview.Edit.ProjectID != env.projectID || preview.Edit.Digest == "" {
		t.Fatalf("preview = %+v", preview)
	}
	if got := effectList(preview.Edit.Effects); !reflect.DeepEqual(got, []string{"portable:update_portable_project", "portable:remove_portable_integration:work-items", "portable:remove_portable_provider:work-items", "local:update_local_record"}) {
		t.Fatalf("effects = %v", got)
	}
	if preview.Edit.Capability.Readiness != projectapp.CapabilityMissing || strings.Contains(preview.Edit.PortableManifest, "work-items") {
		t.Fatalf("candidate still declares work-items: %+v", preview.Edit)
	}
	// Remove writes no operational state: the local disable is untouched.
	if list, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", "guarded"); !reflect.DeepEqual(listed(list.Integrations), []string{"work-items:disabled"}) {
		t.Fatalf("remove preview changed local eligibility: %+v", list.Integrations)
	}

	// Publication belongs to the Project edit engine: replay inputs are refused
	// by the same seam and nothing is written.
	zeroWrites(t, env, func() {
		replay, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "remove", "--project", "guarded", "--integration", "work-items", "--project-id", env.projectID, "--preview-digest", preview.Edit.Digest, "--authorize-local")
		if replay.Result != "Project edit publication is not available" {
			t.Fatalf("replay = %+v", replay)
		}
		unknown, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "remove", "--project", "guarded", "--integration", "ghost")
		if unknown.Category != projectapp.IntegrationNotFound || unknown.Status != string(completion.ValidationFailure) {
			t.Fatalf("unknown remove = %+v", unknown)
		}
		invalid := env.service.(lifecycleService).IntegrationRemove(context.Background(), cli.IntegrationInput{Project: "guarded", Integration: "Bad Key"})
		if invalid.Completion == nil || invalid.Completion.Result().String() != "Project edit input is invalid" {
			t.Fatalf("invalid key = %+v", invalid)
		}
		if missing, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "remove", "--project", "nonexistent", "--integration", "work-items"); missing.Result != "Project was not found" {
			t.Fatalf("missing Project = %+v", missing)
		}
	})
}

const multiIntegrationManifest = `schemaVersion: 2
project:
  id: 123e4567-e89b-42d3-a456-426614174111
  slug: multi
  name: Multi
repositories:
  - key: main
providers:
  - key: chat
    id: slack
  - key: work-items
    id: github
integrations:
  - key: chat
    providerRef: chat
    capabilities:
      - chat
    transport:
      id: mcp
      reference: private-chat-server
    credentialRef: chat-token
  - key: work-items
    providerRef: work-items
    capabilities:
      - work-item
credentialReferences:
  - key: chat-token
    sourceHint: keychain
businessContext:
  text: Unrelated portable context
`

func newMultiIntegrationEnv(t *testing.T) admissionEnv {
	t.Helper()
	env := newAdmissionEnv(t)
	source := filepath.Join(t.TempDir(), "multi")
	repository := writeTree(t, filepath.Join(env.workspace, "multi-main"), map[string]string{"README.md": "x"})
	writeTree(t, source, map[string]string{"axiom.yaml": multiIntegrationManifest})
	runCLI(t, env.service, []string{"project", "install", "--source", source, "--repository", "main=" + repository}, cli.ExitSuccess, "installed")
	return env
}

func TestIntegrationInventoryNamesTransportKindAndCredentialReferenceOnly(t *testing.T) {
	env := newMultiIntegrationEnv(t)
	zeroWrites(t, env, func() {
		list, wire := runIntegrationCLI(t, env.service, cli.ExitSuccess, "list", "--project", "multi")
		if !reflect.DeepEqual(listed(list.Integrations), []string{"chat:enabled", "work-items:enabled"}) {
			t.Fatalf("list = %+v", list.Integrations)
		}
		chat := list.Integrations.Integrations[0]
		if chat.Provider != "slack" || chat.Transport != "mcp" || chat.CredentialRef != "chat-token" || !reflect.DeepEqual(chat.Capabilities, []string{"chat"}) {
			t.Fatalf("chat = %+v", chat)
		}
		for _, secret := range []string{"private-chat-server", "keychain", env.state, env.root, env.workspace} {
			if strings.Contains(wire, secret) {
				t.Fatalf("output leaks %q: %s", secret, wire)
			}
		}
		validated, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, "validate", "--project", "multi")
		if validated.Status != string(completion.ValidationFailure) || validated.Integrations.Validation.Status != projectapp.IntegrationInvalid {
			t.Fatalf("validate = %+v", validated)
		}
		codes := []string{}
		for _, finding := range validated.Integrations.Validation.Findings {
			codes = append(codes, finding.Severity+":"+finding.Code+":"+finding.Integration+":"+finding.Capability)
		}
		if !reflect.DeepEqual(codes, []string{"error:provider_unsupported:chat:chat"}) {
			t.Fatalf("findings = %v", codes)
		}
		if clean, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "validate", "--project", "multi", "--integration", "work-items"); clean.Integrations.Validation.Status != projectapp.IntegrationValid {
			t.Fatalf("work-items validate = %+v", clean.Integrations.Validation)
		}
	})

	// Disabling the non-canonical Integration is local and does not block
	// Work Items, which use a different Integration.
	service := env.service.(lifecycleService)
	preview, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "disable", "--project", "multi", "--integration", "chat")
	runIntegrationCLI(t, env.service, cli.ExitSuccess, "disable", "--project", "multi", "--integration", "chat", "--preview-digest", preview.Operational.Digest, "--authorize-local")
	if create := service.WorkItemCreate(context.Background(), cli.WorkItemInput{Project: "multi", Repository: "main", ProviderRepository: "owner/repo"}); create.Category == projectapp.AdmissionIntegrationDisabled {
		t.Fatalf("disabled chat blocked work-items: %+v", create)
	}
}

func TestIntegrationRemoveNonCanonicalKeepsProviderAndCredentialDeclarations(t *testing.T) {
	env := newMultiIntegrationEnv(t)
	var preview integrationEvent
	zeroWrites(t, env, func() {
		preview, _ = runIntegrationCLI(t, env.service, cli.ExitSuccess, "remove", "--project", "multi", "--integration", "chat")
	})
	if got := effectList(preview.Edit.Effects); !reflect.DeepEqual(got, []string{"portable:update_portable_project", "portable:remove_portable_integration:chat", "local:update_local_record"}) {
		t.Fatalf("effects = %v", got)
	}
	manifest := preview.Edit.PortableManifest
	for _, kept := range []string{"key: chat\n    id: slack", "key: chat-token", "Unrelated portable context", "key: work-items"} {
		if !strings.Contains(manifest, kept) {
			t.Fatalf("%q dropped from candidate:\n%s", kept, manifest)
		}
	}
	if strings.Contains(manifest, "providerRef: chat") || strings.Contains(manifest, "private-chat-server") {
		t.Fatalf("removed Integration survived:\n%s", manifest)
	}
	if preview.Edit.Capability.Readiness != projectapp.CapabilityReady {
		t.Fatalf("work-items capability = %+v", preview.Edit.Capability)
	}
}

func TestIntegrationFailsClosedOnCorruptOperationalState(t *testing.T) {
	env := newAdmissionEnv(t)
	record := filepath.Join(env.state, "projects", env.projectID, "operational.json")
	if err := os.WriteFile(record, []byte(`{"formatVersion":9}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	for _, args := range [][]string{
		{"list", "--project", "guarded"},
		{"disable", "--project", "guarded", "--integration", "work-items"},
		{"enable", "--project", "guarded", "--integration", "work-items"},
	} {
		event, _ := runIntegrationCLI(t, env.service, cli.ExitFailure, args...)
		if event.Category != projectapp.OperationalStateInvalid {
			t.Fatalf("%v: %+v", args, event)
		}
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("corrupt operational state was rewritten")
	}
	// Portable removal does not consult operational state.
	if event, _ := runIntegrationCLI(t, env.service, cli.ExitSuccess, "remove", "--project", "guarded", "--integration", "work-items"); event.Edit == nil {
		t.Fatalf("remove preview = %+v", event)
	}
}
