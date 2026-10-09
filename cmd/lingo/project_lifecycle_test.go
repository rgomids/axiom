package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T03 black-box Evidence: Project archive/reactivate are
// reviewed, machine-local and reversible, and Project discovery (list, show,
// validate) reports the local status without hiding or failing on it.

type lifecycleRepositoryView struct {
	Key          string `json:"key"`
	Path         string `json:"path"`
	Availability string `json:"availability"`
}

type lifecycleState struct {
	Status      string                      `json:"status"`
	Operational *projectapp.OperationalView `json:"operational"`
}

type lifecycleProjectView struct {
	ID           string                    `json:"id"`
	Slug         string                    `json:"slug"`
	Repositories []lifecycleRepositoryView `json:"repositories"`
	State        *lifecycleState           `json:"state"`
}

func (v *lifecycleProjectView) Repository(key string) lifecycleRepositoryView {
	for _, repository := range v.Repositories {
		if repository.Key == key {
			return repository
		}
	}
	return lifecycleRepositoryView{}
}

type lifecycleListEntry struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type lifecycleEvent struct {
	Status      string                         `json:"status"`
	Result      string                         `json:"result"`
	Next        string                         `json:"next"`
	Details     string                         `json:"details"`
	Category    string                         `json:"category"`
	Project     *lifecycleProjectView          `json:"project"`
	Projects    []lifecycleListEntry           `json:"projects"`
	Operational *projectapp.OperationalPreview `json:"operational"`
	Readiness   *projectapp.ReadinessReport    `json:"readiness"`
	State       *lifecycleState                `json:"state"`
}

func runLifecycle(t *testing.T, service cli.Service, wantCode int, args ...string) lifecycleEvent {
	t.Helper()
	var output bytes.Buffer
	if code := cli.Run(context.Background(), args, service, currentProvenance(), &output); code != wantCode {
		t.Fatalf("%v: exit %d, want %d; output=%s", args, code, wantCode, output.String())
	}
	var event lifecycleEvent
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("%v: %v: %s", args, err, output.String())
	}
	return event
}

func showProjectEvent(t *testing.T, service cli.Service, selector string) lifecycleEvent {
	t.Helper()
	event := runLifecycle(t, service, cli.ExitSuccess, "project", "show", "--selector", selector)
	if event.Project == nil || event.Project.State == nil {
		t.Fatalf("show %s = %+v", selector, event)
	}
	return event
}

func listSlugs(event lifecycleEvent) []string {
	slugs := []string{}
	for _, entry := range event.Projects {
		slugs = append(slugs, entry.Slug+":"+entry.Status)
	}
	return slugs
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestProjectArchiveAndReactivateAreReviewedLocalAndReversible(t *testing.T) {
	env := newAdmissionEnv(t)
	archiveNamespace := filepath.Join(t.TempDir(), "axiom-archive")
	t.Setenv("AXIOM_ARCHIVE_ROOT", archiveNamespace)
	projectDirectory := filepath.Join(env.state, "projects", env.projectID)
	installation := readBytes(t, filepath.Join(projectDirectory, "installation.json"))
	baseline := snapshotTrees(t, env.root, env.workspace)
	initialState := snapshotTrees(t, env.state)

	for _, selector := range []string{"guarded", env.projectID} {
		preview := runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", selector)
		if preview.Category != projectapp.OperationalPreviewReady || preview.Operational == nil || preview.Operational.Digest == "" {
			t.Fatalf("preview = %+v", preview)
		}
		operational := preview.Operational
		if operational.Operation != projectapp.ArchiveProject || operational.ProjectID != env.projectID || operational.Current.ProjectStatus != projectapp.ProjectActive || operational.Result.ProjectStatus != projectapp.ProjectArchived {
			t.Fatalf("preview = %+v", operational)
		}
		if len(operational.Effects) != 1 || operational.Effects[0].Scope != projectapp.LocalScope || operational.Effects[0].Code != "archive_project" || operational.Boundary.Portable != "unchanged" || operational.Boundary.Provider != "none" || operational.Boundary.Credentials != "unchanged" {
			t.Fatalf("preview effects = %+v", operational)
		}
		if !bytes.Equal(initialState, snapshotTrees(t, env.state)) {
			t.Fatal("preview wrote local state")
		}
	}
	digest := runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", "guarded").Operational.Digest

	// Exact digest authority: stale digest, digest without authority and
	// authority without a digest all deny with zero writes.
	for name, args := range map[string][]string{
		"stale digest":   {"--preview-digest", "stale", "--authorize-local"},
		"digest only":    {"--preview-digest", digest},
		"authority only": {"--authorize-local"},
	} {
		denied := runLifecycle(t, env.service, cli.ExitFailure, append([]string{"project", "archive", "--project", "guarded"}, args...)...)
		if denied.Status != "denied_authority" || denied.Category != projectapp.OperationalAuthorityDenied || denied.Operational == nil || denied.Operational.Digest != digest {
			t.Fatalf("%s: %+v", name, denied)
		}
		if !bytes.Equal(initialState, snapshotTrees(t, env.state)) {
			t.Fatalf("%s wrote local state", name)
		}
	}

	applied := runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", env.projectID, "--preview-digest", digest, "--authorize-local")
	if applied.Category != projectapp.OperationalApplied || applied.Operational == nil || applied.Operational.Result.ProjectStatus != projectapp.ProjectArchived {
		t.Fatalf("applied = %+v", applied)
	}
	// Archive is local operational state only (C230-01, F-07).
	if !bytes.Equal(baseline, snapshotTrees(t, env.root, env.workspace)) {
		t.Fatal("archive changed portable configuration or a Repository working copy")
	}
	if !bytes.Equal(installation, readBytes(t, filepath.Join(projectDirectory, "installation.json"))) {
		t.Fatal("archive changed installation.json")
	}
	entries, err := os.ReadDir(projectDirectory)
	if err != nil || len(entries) != 2 || entries[0].Name() != "installation.json" || entries[1].Name() != "operational.json" {
		t.Fatalf("installation directory = %v, %v", entries, err)
	}
	if _, err := os.Stat(archiveNamespace); !os.IsNotExist(err) {
		t.Fatalf("archive wrote the S7 preservation namespace: %v", err)
	}
	archivedState := snapshotTrees(t, env.state)

	// Equivalent request: deterministic no-op, no authority, no write; a replay
	// of the consumed digest is the same no-op.
	for _, args := range [][]string{{"project", "archive", "--project", "guarded"}, {"project", "archive", "--project", "guarded", "--preview-digest", digest, "--authorize-local"}} {
		noop := runLifecycle(t, env.service, cli.ExitSuccess, args...)
		if noop.Category != projectapp.ProjectAlreadyArchived || noop.Operational == nil || len(noop.Operational.Effects) != 0 {
			t.Fatalf("archive no-op = %+v", noop)
		}
	}
	if !bytes.Equal(archivedState, snapshotTrees(t, env.state)) {
		t.Fatal("no-op archive wrote local state")
	}

	// Archive blocks evolution, not inspection or administration.
	shown := showProjectEvent(t, env.service, "guarded")
	if shown.Project.State.Status != "archived" || shown.Project.State.Operational == nil || shown.Project.State.Operational.ProjectStatus != projectapp.ProjectArchived || len(shown.Project.State.Operational.DisabledIntegrations) != 0 {
		t.Fatalf("archived show = %+v", shown.Project)
	}
	runLifecycle(t, env.service, cli.ExitSuccess, "project", "resolve", "--selector", "guarded")
	validated := runLifecycle(t, env.service, cli.ExitSuccess, "project", "validate", "--project", "guarded")
	if validated.Readiness == nil || validated.Readiness.Structure != "valid" || validated.State == nil || validated.State.Status != "archived" {
		t.Fatalf("archived validate = %+v", validated)
	}
	edit := runLifecycle(t, env.service, cli.ExitSuccess, "project", "configure", "--project", "guarded", "--name", "Renamed")
	if edit.Result != "Project edit preview ready" {
		t.Fatalf("archived configure edit preview = %+v", edit)
	}
	if !bytes.Equal(archivedState, snapshotTrees(t, env.state)) || !bytes.Equal(baseline, snapshotTrees(t, env.root, env.workspace)) {
		t.Fatal("inspection while archived wrote state")
	}
	if listed := runLifecycle(t, env.service, cli.ExitSuccess, "project", "list"); len(listed.Projects) != 0 {
		t.Fatalf("default list hides nothing: %v", listSlugs(listed))
	}
	if listed := runLifecycle(t, env.service, cli.ExitSuccess, "project", "list", "--include-archived"); !reflect.DeepEqual(listSlugs(listed), []string{"guarded:archived"}) {
		t.Fatalf("include-archived list = %v", listSlugs(listed))
	}

	// Reactivation is the exact reviewed inverse.
	preview := runLifecycle(t, env.service, cli.ExitSuccess, "project", "reactivate", "--project", "guarded")
	if preview.Category != projectapp.OperationalPreviewReady || preview.Operational.Operation != projectapp.ReactivateProject || preview.Operational.Effects[0].Code != "reactivate_project" {
		t.Fatalf("reactivate preview = %+v", preview)
	}
	if denied := runLifecycle(t, env.service, cli.ExitFailure, "project", "reactivate", "--project", "guarded", "--preview-digest", digest, "--authorize-local"); denied.Category != projectapp.OperationalAuthorityDenied {
		t.Fatalf("archive digest authorized reactivation: %+v", denied)
	}
	if !bytes.Equal(archivedState, snapshotTrees(t, env.state)) {
		t.Fatal("denied reactivation wrote local state")
	}
	reactivated := runLifecycle(t, env.service, cli.ExitSuccess, "project", "reactivate", "--project", "guarded", "--preview-digest", preview.Operational.Digest, "--authorize-local")
	if reactivated.Category != projectapp.OperationalApplied || reactivated.Operational.Result.ProjectStatus != projectapp.ProjectActive {
		t.Fatalf("reactivated = %+v", reactivated)
	}
	if shown := showProjectEvent(t, env.service, env.projectID); shown.Project.State.Status != "active" {
		t.Fatalf("reactivated show = %+v", shown.Project.State)
	}
	if listed := runLifecycle(t, env.service, cli.ExitSuccess, "project", "list"); !reflect.DeepEqual(listSlugs(listed), []string{"guarded:active"}) {
		t.Fatalf("reactivated list = %v", listSlugs(listed))
	}
	activeState := snapshotTrees(t, env.state)
	if noop := runLifecycle(t, env.service, cli.ExitSuccess, "project", "reactivate", "--project", "guarded"); noop.Category != projectapp.ProjectAlreadyActive || len(noop.Operational.Effects) != 0 {
		t.Fatalf("reactivate no-op = %+v", noop)
	}
	if !bytes.Equal(activeState, snapshotTrees(t, env.state)) {
		t.Fatal("no-op reactivation wrote local state")
	}
	if !bytes.Equal(baseline, snapshotTrees(t, env.root, env.workspace)) || !bytes.Equal(installation, readBytes(t, filepath.Join(projectDirectory, "installation.json"))) {
		t.Fatal("archive/reactivate changed portable configuration, installation.json or a Repository")
	}
	if _, err := os.Stat(archiveNamespace); !os.IsNotExist(err) {
		t.Fatalf("reactivate wrote the S7 preservation namespace: %v", err)
	}
	if env.providerCalled() {
		t.Fatal("archive lifecycle invoked the Provider")
	}
}

func TestProjectArchiveAndReactivateRequireAnInstalledProject(t *testing.T) {
	env := newAdmissionEnv(t)
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	for _, verb := range []string{"archive", "reactivate"} {
		for _, selector := range []string{"missing", "123e4567-e89b-42d3-a456-426614174999"} {
			event := runLifecycle(t, env.service, cli.ExitFailure, "project", verb, "--project", selector, "--preview-digest", "any", "--authorize-local")
			if event.Status != "validation_failure" || event.Category != projectapp.OperationalNotInstalled || event.Result != "Project is not installed on this machine" {
				t.Fatalf("%s %s = %+v", verb, selector, event)
			}
		}
	}
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.workspace)) {
		t.Fatal("not-installed lifecycle request wrote state")
	}
	// The caller's working directory never selects a Project.
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.Chdir(env.workspace); err != nil {
		t.Fatal(err)
	}
	if event := runLifecycle(t, env.service, cli.ExitFailure, "project", "archive", "--project", "main"); event.Category != projectapp.OperationalNotInstalled {
		t.Fatalf("CWD-derived selection = %+v", event)
	}
}

func TestProjectLifecycleFailsClosedOnCorruptOrInterruptedOperationalState(t *testing.T) {
	env := newAdmissionEnv(t)
	projectDirectory := filepath.Join(env.state, "projects", env.projectID)
	operationalPath := filepath.Join(projectDirectory, "operational.json")
	if err := os.WriteFile(operationalPath, []byte("{not canonical"), 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt := snapshotTrees(t, env.root, env.state, env.workspace)
	for _, verb := range []string{"archive", "reactivate"} {
		for _, args := range [][]string{{}, {"--preview-digest", "any", "--authorize-local"}} {
			event := runLifecycle(t, env.service, cli.ExitFailure, append([]string{"project", verb, "--project", "guarded"}, args...)...)
			if event.Category != projectapp.OperationalStateInvalid || event.Status != "failure" {
				t.Fatalf("%s %v = %+v", verb, args, event)
			}
		}
	}
	if !bytes.Equal(corrupt, snapshotTrees(t, env.root, env.state, env.workspace)) {
		t.Fatal("corrupt operational state was overwritten or normalized")
	}
	// Inspection stays available and reports invalid, never active.
	if shown := showProjectEvent(t, env.service, "guarded"); shown.Project.State.Status != "invalid" || shown.Project.State.Operational != nil {
		t.Fatalf("show corrupt = %+v", shown.Project.State)
	}
	validated := runLifecycle(t, env.service, cli.ExitSuccess, "project", "validate", "--project", "guarded")
	if validated.State == nil || validated.State.Status != "invalid" || validated.Readiness.Structure != "valid" {
		t.Fatalf("validate corrupt = %+v", validated)
	}
	for _, args := range [][]string{{"project", "list"}, {"project", "list", "--include-archived"}} {
		if listed := runLifecycle(t, env.service, cli.ExitSuccess, args...); !reflect.DeepEqual(listSlugs(listed), []string{"guarded:invalid"}) {
			t.Fatalf("%v with corrupt state = %v", args, listSlugs(listed))
		}
	}

	// An interrupted publication in the Project directory requires recovery.
	if err := os.Remove(operationalPath); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(projectDirectory, ".axiom-stage-interrupted")
	if err := os.WriteFile(stage, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	interrupted := snapshotTrees(t, env.root, env.state, env.workspace)
	for _, args := range [][]string{{"project", "archive", "--project", "guarded"}, {"project", "reactivate", "--project", "guarded", "--preview-digest", "any", "--authorize-local"}} {
		if event := runLifecycle(t, env.service, cli.ExitFailure, args...); event.Category != "recovery_required" || event.Status != "validation_failure" {
			t.Fatalf("%v = %+v", args, event)
		}
	}
	if event := runLifecycle(t, env.service, cli.ExitFailure, "project", "list"); event.Result != "Configured Project state requires recovery" {
		t.Fatalf("list with interrupted state = %+v", event)
	}
	if event := runLifecycle(t, env.service, cli.ExitFailure, "project", "show", "--selector", "guarded"); event.Result != "Local Project state requires recovery" {
		t.Fatalf("show with interrupted state = %+v", event)
	}
	if !bytes.Equal(interrupted, snapshotTrees(t, env.root, env.state, env.workspace)) {
		t.Fatal("interrupted state was changed")
	}
}

func TestProjectArchiveIsIndependentPerMachineStateRoot(t *testing.T) {
	env := newAdmissionEnv(t)
	repository := filepath.Join(env.workspace, "main")
	otherState := filepath.Join(t.TempDir(), "other-state")
	t.Setenv("LINGO_STATE_ROOT", otherState)
	other := compose()
	runCLI(t, other, []string{"project", "install", "--source", filepath.Join(env.root, "guarded"), "--repository", "main=" + repository}, cli.ExitSuccess, "installed")
	// Same portable Project, two machines: each installation has its own identity.
	if listed := runLifecycle(t, other, cli.ExitSuccess, "project", "list"); len(listed.Projects) != 1 || listed.Projects[0].Status != "active" {
		t.Fatalf("other machine list = %v", listSlugs(listed))
	}
	portable := snapshotTrees(t, env.root)
	otherBefore := snapshotTrees(t, otherState)

	preview := runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", "guarded")
	runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", "guarded", "--preview-digest", preview.Operational.Digest, "--authorize-local")
	if shown := showProjectEvent(t, env.service, "guarded"); shown.Project.State.Status != "archived" {
		t.Fatalf("archived machine show = %+v", shown.Project.State)
	}
	if shown := showProjectEvent(t, other, "guarded"); shown.Project.State.Status != "active" {
		t.Fatalf("other machine saw the archive: %+v", shown.Project.State)
	}
	if listed := runLifecycle(t, other, cli.ExitSuccess, "project", "list"); !reflect.DeepEqual(listSlugs(listed), []string{"guarded:active"}) {
		t.Fatalf("other machine list = %v", listSlugs(listed))
	}
	if !bytes.Equal(portable, snapshotTrees(t, env.root)) || !bytes.Equal(otherBefore, snapshotTrees(t, otherState)) {
		t.Fatal("archive crossed the machine boundary")
	}
}

func TestProjectListFiltersArchivedAndOrdersDeterministically(t *testing.T) {
	env := newAdmissionEnv(t)
	for _, slug := range []string{"zulu", "alpha"} {
		repository := writeTree(t, filepath.Join(env.workspace, slug), map[string]string{"README.md": slug})
		configureProject(t, env.service, slug, strings.ToUpper(slug[:1])+slug[1:], "main="+repository, "github")
	}
	want := []string{"alpha:active", "guarded:active", "zulu:active"}
	if listed := runLifecycle(t, env.service, cli.ExitSuccess, "project", "list"); !reflect.DeepEqual(listSlugs(listed), want) {
		t.Fatalf("list = %v", listSlugs(listed))
	}
	env.apply(t, projectapp.ArchiveProject, "")
	if listed := runLifecycle(t, env.service, cli.ExitSuccess, "project", "list"); !reflect.DeepEqual(listSlugs(listed), []string{"alpha:active", "zulu:active"}) {
		t.Fatalf("default list = %v", listSlugs(listed))
	}
	listed := runLifecycle(t, env.service, cli.ExitSuccess, "project", "list", "--include-archived")
	if !reflect.DeepEqual(listSlugs(listed), []string{"alpha:active", "guarded:archived", "zulu:active"}) || listed.Projects[1].Name != "Guarded" {
		t.Fatalf("include-archived list = %v", listSlugs(listed))
	}
	// A malformed record is invalid, never active, and never hidden by default.
	var zuluID string
	for _, entry := range listed.Projects {
		if entry.Slug == "zulu" {
			zuluID = entry.ID
		}
	}
	if err := os.WriteFile(filepath.Join(env.state, "projects", zuluID, "operational.json"), []byte(`{"formatVersion":2}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"project", "list"}, {"project", "list", "--include-archived"}} {
		got := listSlugs(runLifecycle(t, env.service, cli.ExitSuccess, args...))
		if got[len(got)-1] != "zulu:invalid" || strings.Contains(strings.Join(got, ","), "zulu:active") {
			t.Fatalf("%v = %v", args, got)
		}
	}
	var human bytes.Buffer
	if code := cli.RunInteractive(context.Background(), []string{"--human", "project", "list", "--include-archived"}, env.service, currentProvenance(), nil, &human, io.Discard); code != cli.ExitSuccess {
		t.Fatalf("human list exit %d", code)
	}
	for _, expected := range []string{"id `" + env.projectID + "` · slug `guarded` · name `Guarded` · status `archived`", "status `invalid`", "status `active`"} {
		if !strings.Contains(human.String(), expected) {
			t.Fatalf("%q absent from %s", expected, human.String())
		}
	}
	if code := cli.Run(context.Background(), []string{"project", "list", "--bogus"}, env.service, currentProvenance(), io.Discard); code != cli.ExitFailure {
		t.Fatalf("unknown list flag exit %d", code)
	}
}

func TestProjectShowReportsUnavailableBindingsWithoutFailingAndKeepsResolveStrict(t *testing.T) {
	env := newAdmissionEnv(t)
	repository := filepath.Join(env.workspace, "main")
	writeTree(t, repository, map[string]string{"sentinel.txt": "working copy"})
	shown := showProjectEvent(t, env.service, "guarded")
	if got := shown.Project.Repository("main"); got.Availability != "available" || got.Path != repository || shown.Project.State.Status != "active" || shown.Project.State.Operational == nil || len(shown.Project.State.Operational.DisabledIntegrations) != 0 {
		t.Fatalf("available show = %+v", shown.Project)
	}
	moved := repository + "-moved"
	if err := os.Rename(repository, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	shown = showProjectEvent(t, env.service, env.projectID)
	if got := shown.Project.Repository("main"); got.Availability != "unavailable" || got.Path != repository || shown.Result != "Project resolved" || !strings.Contains(shown.Next, "unavailable Repository binding") {
		t.Fatalf("unavailable show = %+v", shown)
	}
	runCLI(t, env.service, []string{"project", "resolve", "--selector", "guarded"}, cli.ExitFailure, "repository_unavailable")
	// Administration does not depend on Repository availability.
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	preview := runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", "guarded")
	runLifecycle(t, env.service, cli.ExitSuccess, "project", "archive", "--project", "guarded", "--preview-digest", preview.Operational.Digest, "--authorize-local")
	if shown := showProjectEvent(t, env.service, "guarded"); shown.Project.State.Status != "archived" || shown.Project.Repository("main").Availability != "unavailable" {
		t.Fatalf("archived unavailable show = %+v", shown.Project)
	}
	if bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.workspace)) {
		t.Fatal("archive did not publish local state")
	}
	if wire := readBytes(t, filepath.Join(moved, "sentinel.txt")); string(wire) != "working copy" {
		t.Fatalf("Repository working copy changed: %q", wire)
	}
	// Source unavailability keeps its established inspection failure.
	if err := os.RemoveAll(filepath.Join(env.root, "guarded")); err != nil {
		t.Fatal(err)
	}
	runCanonicalCLI(t, env.service, []string{"project", "show", "--selector", "guarded"}, cli.ExitFailure, "retryable_failure", "Project source is unavailable")
}

func TestProjectShowHumanRendersStateAndAvailability(t *testing.T) {
	env := newAdmissionEnv(t)
	env.apply(t, projectapp.ArchiveProject, "")
	var human bytes.Buffer
	if code := cli.RunInteractive(context.Background(), []string{"--human", "project", "show", "--selector", "guarded"}, env.service, currentProvenance(), nil, &human, io.Discard); code != cli.ExitSuccess {
		t.Fatalf("exit %d: %s", code, human.String())
	}
	for _, expected := range []string{"- **status:** `archived`", "- **disabledIntegrations:** _none_", "availability `available`"} {
		if !strings.Contains(human.String(), expected) {
			t.Fatalf("%q absent from %s", expected, human.String())
		}
	}
	var archive bytes.Buffer
	if code := cli.RunInteractive(context.Background(), []string{"--human", "project", "archive", "--project", "guarded"}, env.service, currentProvenance(), nil, &archive, io.Discard); code != cli.ExitSuccess {
		t.Fatalf("archive exit %d: %s", code, archive.String())
	}
	for _, expected := range []string{"- **category:** `project_already_archived`", "- **operation:** `archive`", "- **boundary:** portable `unchanged` · provider `none` · credentials `unchanged`"} {
		if !strings.Contains(archive.String(), expected) {
			t.Fatalf("%q absent from %s", expected, archive.String())
		}
	}
}

func TestProjectValidateBySelectorReadsTheRecordedSource(t *testing.T) {
	env := newEditEnvironment(t)
	roots := []string{env.root, env.state}
	before := snapshotTrees(t, roots...)
	for _, selector := range []string{"sample", env.projectID} {
		event := runLifecycle(t, env.service, cli.ExitSuccess, "project", "validate", "--project", selector)
		if event.Result != "Project is valid" || event.Readiness == nil || event.Readiness.Structure != "valid" || event.Readiness.ProjectID != env.projectID || event.State == nil || event.State.Status != "active" {
			t.Fatalf("validate %s = %+v", selector, event)
		}
	}
	if !bytes.Equal(before, snapshotTrees(t, roots...)) {
		t.Fatal("validate wrote state")
	}

	// A source outside the projects root is validated where the protected
	// record names it; the POC slug path reads only the projects root.
	source := env.installExternal(t)
	roots = append(roots, env.extra...)
	if event := runLifecycle(t, env.service, cli.ExitSuccess, "project", "validate", "--project", "external"); event.Readiness.Structure != "valid" {
		t.Fatalf("external validate = %+v", event)
	}
	runLifecycle(t, env.service, cli.ExitFailure, "project", "validate", "--slug", "external")
	rewriteManifest(t, source, func(wire string) string { return strings.Replace(wire, "name: External", "name: Edited", 1) })
	before = snapshotTrees(t, roots...)
	stale := runLifecycle(t, env.service, cli.ExitFailure, "project", "validate", "--project", "external")
	if stale.Status != "validation_failure" || stale.Readiness == nil || stale.Readiness.Structure != "unavailable" || len(stale.Readiness.Blockers) != 1 || stale.Readiness.Blockers[0].Code != projectapp.BlockerInstallationStale || stale.State == nil || stale.State.Status != "active" {
		t.Fatalf("stale validate = %+v", stale)
	}
	if !bytes.Equal(before, snapshotTrees(t, roots...)) {
		t.Fatal("failed validate wrote state")
	}

	missing := runLifecycle(t, env.service, cli.ExitFailure, "project", "validate", "--project", "nope")
	if missing.Result != "Project is not installed on this machine" || missing.State != nil {
		t.Fatalf("not installed validate = %+v", missing)
	}
	for _, args := range [][]string{
		{"project", "validate", "--slug", "sample", "--project", "sample"},
		{"project", "validate", "--project", "sample", "--project", "sample"},
		{"project", "validate", "--project", "sample", "--bogus"},
	} {
		if event := runLifecycle(t, env.service, cli.ExitFailure, args...); event.Status != "validation_failure" || event.Readiness != nil {
			t.Fatalf("%v = %+v", args, event)
		}
	}
	// The POC path is unchanged.
	runCanonicalCLI(t, env.service, []string{"project", "validate", "--slug", "sample"}, cli.ExitSuccess, "success", "Project is valid")
}

func TestLegacyUpdateStalenessIsConfirmedAndRefusedForInstalledProjects(t *testing.T) {
	env := newEditEnvironment(t)
	roots := []string{env.root, env.state}
	before := snapshotTrees(t, roots...)
	refused := runLifecycle(t, env.service, cli.ExitFailure, "project", "update", "--slug", "sample", "--name", "Renamed")
	if refused.Status != "validation_failure" || refused.Details != projectapp.ExplicitEditRequired || !strings.Contains(refused.Next, "project configure --project <slug> --name <name>") {
		t.Fatalf("refused update = %+v", refused)
	}
	if !bytes.Equal(before, snapshotTrees(t, roots...)) {
		t.Fatal("refused update wrote state")
	}
	if result := env.service.(lifecycleService).Update(context.Background(), cli.UpdateInput{Slug: "sample", Name: "Renamed"}); result.Category != projectapp.ExplicitEditRequired {
		t.Fatalf("service category = %q", result.Category)
	}
	if event := runLifecycle(t, env.service, cli.ExitSuccess, "project", "validate", "--project", "sample"); event.Readiness.Structure != "valid" {
		t.Fatalf("refused update left the Project stale: %+v", event.Readiness)
	}
	// Confirmed: the portable-only write the update performed leaves the
	// installation record stale.
	if result := env.service.(lifecycleService).lifecycle.Update(context.Background(), projectapp.UpdateRequest{Slug: "sample", Name: "Renamed"}); result.Status != projectapp.LifecycleApplied {
		t.Fatalf("direct update = %+v", result)
	}
	stale := runLifecycle(t, env.service, cli.ExitFailure, "project", "validate", "--project", "sample")
	if stale.Readiness == nil || len(stale.Readiness.Blockers) != 1 || stale.Readiness.Blockers[0].Code != projectapp.BlockerInstallationStale {
		t.Fatalf("legacy update did not stale the installation: %+v", stale.Readiness)
	}
}

func TestLegacyUpdateStillServesProjectsWithoutInstallation(t *testing.T) {
	env := newBootstrapEnv(t)
	runCLI(t, env.service, []string{"project", "init", "--slug", "poc", "--name", "Poc"}, cli.ExitSuccess, "applied")
	runCLI(t, env.service, []string{"project", "update", "--slug", "poc", "--name", "Poc Two"}, cli.ExitSuccess, "applied")
	if !strings.Contains(string(readBytes(t, filepath.Join(env.root, "poc", "axiom.yaml"))), "name: Poc Two") {
		t.Fatal("uninstalled update did not apply")
	}
}
