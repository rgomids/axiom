package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T03 Evidence: stale/concurrent authority, Repository
// detach semantics (F-03), archived administration, and ADR-0007 cross-store
// recovery observed by new processes.

func TestEditReplayPreservesDocumentationBindings(t *testing.T) {
	env := editEnvironment{root: filepath.Join(t.TempDir(), "projects"), state: filepath.Join(t.TempDir(), "state"), api: filepath.Join(t.TempDir(), "api")}
	if err := os.Mkdir(env.api, 0o700); err != nil {
		t.Fatal(err)
	}
	notes := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(notes, []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINGO_PROJECTS_ROOT", env.root)
	t.Setenv("LINGO_STATE_ROOT", env.state)
	env.service = compose()
	create := []string{"--slug", "sample", "--name", "Sample", "--work-item-provider", "none", "--repository", "api=" + env.api, "--documentation", "notes=local-file:" + notes}
	var output bytes.Buffer
	if code := cli.Run(context.Background(), append([]string{"project", "configure"}, create...), env.service, currentProvenance(), &output); code != cli.ExitSuccess {
		t.Fatalf("create preview: %s", output.String())
	}
	var setup struct {
		Setup projectapp.SetupPreview `json:"setup"`
	}
	if err := json.Unmarshal(output.Bytes(), &setup); err != nil || setup.Setup.Digest == "" {
		t.Fatalf("create preview: %s", output.String())
	}
	output.Reset()
	if code := cli.Run(context.Background(), append([]string{"project", "configure"}, append(create, "--project-id", setup.Setup.ProjectID, "--preview-digest", setup.Setup.Digest, "--authorize-local")...), env.service, currentProvenance(), &output); code != cli.ExitSuccess {
		t.Fatalf("create: %s", output.String())
	}
	env.projectID = setup.Setup.ProjectID
	before := env.record(t)
	if len(before.Documentation) != 1 {
		t.Fatalf("fixture has no documentation binding: %+v", before.Documentation)
	}
	env.publish(t, "--project", "sample", "--name", "Renamed")
	if after := env.record(t); !jsonEqual(t, after.Documentation, before.Documentation) || !jsonEqual(t, after.Repositories, before.Repositories) {
		t.Fatal("edit publication changed documentation or repository bindings")
	}
}

func TestEditReplayNoOpStaleAndDriftFailWithZeroWrites(t *testing.T) {
	env := newEditEnvironment(t)
	noop, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample")
	env.runEdit(t, cli.ExitSuccess, "success", "Project edit already matches the authorized preview", replayArgs(noop.Edit, "--project", "sample")...)

	renamed, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed")
	stale := renamed.Edit
	stale.Digest = strings.Repeat("0", 64)
	env.runEdit(t, cli.ExitFailure, "denied_authority", "Project edit authority is missing or stale", replayArgs(stale, "--project", "sample", "--name", "Renamed")...)
	// The digest of one intent never authorizes another.
	env.runEdit(t, cli.ExitFailure, "denied_authority", "Project edit authority is missing or stale", replayArgs(renamed.Edit, "--project", "sample", "--name", "Other")...)
	drift := renamed.Edit
	drift.ProjectID = "123e4567-e89b-42d3-a456-426614174999"
	env.runEdit(t, cli.ExitFailure, "denied_authority", "Project edit authority is missing or stale", replayArgs(drift, "--project", "sample", "--name", "Renamed")...)

	// Local drift between preview and replay changes the fresh digest.
	env.writeRecord(t, enrichLocalMetadata)
	env.runEdit(t, cli.ExitFailure, "denied_authority", "Project edit authority is missing or stale", replayArgs(renamed.Edit, "--project", "sample", "--name", "Renamed")...)

	// Selector drift: the reviewed ID no longer names the selected Project.
	refreshed, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed")
	second := configureProjectWith(t, env.service, "second", "Second", "none", "api="+env.api)
	env.runEdit(t, cli.ExitFailure, "denied_authority", "Project edit authority is missing or stale", replayArgs(refreshed.Edit, "--project", second.ProjectID, "--name", "Renamed")...)

	// Portable drift between preview and replay leaves the selection
	// incoherent: zero writes, no publication.
	// (The legacy by-slug update now refuses installed Projects, so the drift
	// is an external edit of the portable manifest.)
	changed := strings.Replace(string(readBytes(t, env.manifestPath())), "name: Sample", "name: Changed", 1)
	if err := os.WriteFile(env.manifestPath(), []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	env.runEdit(t, cli.ExitFailure, "failure", "Selected Project state is not safe to edit", replayArgs(refreshed.Edit, "--project", "sample", "--name", "Renamed")...)
}

func TestEditPublicationRefusesConcurrentWritersWithoutOverwrite(t *testing.T) {
	env := newEditEnvironment(t)
	preview, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed")
	priorLocal := readBytes(t, env.installationPath())
	concurrent := strings.Replace(string(readBytes(t, env.manifestPath())), "name: Sample", "name: Concurrent", 1)
	operational, err := local.NewOperationalStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	service := env.withFault(func(stage local.EditStage) error {
		if stage != local.EditStageRecorded {
			return nil
		}
		// A lock-respecting local writer is excluded while publication runs.
		if err := operational.CommitOperational(context.Background(), env.projectID, projectapp.OperationalRevisionAbsent, projectapp.OperationalState{ProjectStatus: projectapp.ProjectArchived, DisabledIntegrations: []string{}}); !errors.Is(err, local.ErrConflict) {
			t.Errorf("concurrent local writer was not excluded: %v", err)
		}
		// A portable writer wins the race before the portable CAS.
		return os.WriteFile(env.manifestPath(), []byte(concurrent), 0o600)
	})
	code, event, output := env.configureWith(t, service, replayArgs(preview.Edit, "--project", "sample", "--name", "Renamed")...)
	if code != cli.ExitFailure || event.Status != "denied_authority" || event.Result != "Project state changed before publication" {
		t.Fatalf("concurrent writer: code=%d %s", code, output)
	}
	if string(readBytes(t, env.manifestPath())) != concurrent || !bytes.Equal(readBytes(t, env.installationPath()), priorLocal) {
		t.Fatal("conflicting publication overwrote the concurrent writer or local state")
	}
	env.assertNoEditState(t)
}

func TestEditPublicationFailsClosedForSourceOutsideProjectsRoot(t *testing.T) {
	env := newEditEnvironment(t)
	env.installExternal(t)
	preview, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "external", "--name", "Renamed")
	env.runEdit(t, cli.ExitFailure, "validation_failure", "Project source is outside the Lingo projects root; edit publication is not supported", replayArgs(preview.Edit, "--project", "external", "--name", "Renamed")...)
}

func TestRepositoryDetachPreservesWorkingCopyAndDependentHistory(t *testing.T) {
	env := newEditEnvironment(t)
	if err := os.WriteFile(filepath.Join(env.api, "SENTINEL"), []byte("working copy"), 0o600); err != nil {
		t.Fatal(err)
	}
	history := []string{
		filepath.Join(env.state, "work-items", env.projectID, "api-sentinel.json"),
		filepath.Join(env.state, "executions", "v1", env.projectID, "sentinel.json"),
	}
	for _, path := range history {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("history"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	historyRoots := []string{filepath.Join(env.state, "work-items"), filepath.Join(env.state, "executions")}
	workingCopy, historyBefore := snapshotTrees(t, env.api), snapshotTrees(t, historyRoots...)
	preview, _ := env.publish(t, "--project", "sample", "--remove-repository", "api")
	found := false
	for _, effect := range preview.Effects {
		found = found || effect.Scope == projectapp.LocalScope && effect.Code == projectapp.PreserveRepositoryHistory && effect.Key == "api"
	}
	if !found {
		t.Fatalf("detach preview does not disclose preserved history: %v", effectList(preview.Effects))
	}
	if !bytes.Equal(workingCopy, snapshotTrees(t, env.api)) {
		t.Fatal("detach changed the working copy")
	}
	if !bytes.Equal(historyBefore, snapshotTrees(t, historyRoots...)) {
		t.Fatal("detach changed dependent local history")
	}
	if _, ok := bindingPaths(env.record(t))["api"]; ok {
		t.Fatal("detach kept the local binding")
	}
	if strings.Contains(string(readBytes(t, env.manifestPath())), "key: api") {
		t.Fatal("detach kept the portable association")
	}
	env.runEdit(t, cli.ExitFailure, "validation_failure", "Repository to remove is not configured", "--project", "sample", "--remove-repository", "api")
	// Re-attaching the same key restores reachability.
	env.publish(t, "--project", "sample", "--repository", "api="+env.api)
	if bindingPaths(env.record(t))["api"] != env.api {
		t.Fatal("re-attach did not restore the binding")
	}
}

func TestEditAllowedWhileArchived(t *testing.T) {
	env := newAdmissionEnv(t)
	env.apply(t, projectapp.ArchiveProject, "")
	args := []string{"project", "configure", "--project", "guarded", "--name", "Renamed"}
	var output bytes.Buffer
	cli.Run(context.Background(), args, env.service, currentProvenance(), &output)
	var preview editEvent
	if err := json.Unmarshal(output.Bytes(), &preview); err != nil || preview.Result != "Project edit preview ready" {
		t.Fatalf("archived preview: %s", output.String())
	}
	output.Reset()
	code := cli.Run(context.Background(), replayArgs(preview.Edit, args...), env.service, currentProvenance(), &output)
	if code != cli.ExitSuccess || !strings.Contains(output.String(), `"result":"Project edit published"`) {
		t.Fatalf("archived publish: %s", output.String())
	}
}

func TestEditPartialPublicationRequiresRecoveryAndFinalizes(t *testing.T) {
	env := newEditEnvironment(t)
	edit := []string{"--project", "sample", "--name", "Renamed", "--remove-repository", "web"}
	preview, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", edit...)
	priorLocal := readBytes(t, env.installationPath())
	service := env.withFault(func(stage local.EditStage) error {
		if stage == local.EditStagePortableCommitted {
			return errors.New("injected local publication failure")
		}
		return nil
	})
	code, event, output := env.configureWith(t, service, replayArgs(preview.Edit, edit...)...)
	if code == cli.ExitSuccess || event.Status != "partial" || event.Result != "Portable Project edit published; local publication did not complete" || !strings.Contains(event.Next, "recovery inspect") {
		t.Fatalf("partial: code=%d %s", code, output)
	}
	if !containsText(event.References, "portable:"+preview.Edit.PortableDestination) {
		t.Fatalf("partial result does not name the confirmed portable effect: %v", event.References)
	}
	if !strings.Contains(string(readBytes(t, env.manifestPath())), "name: Renamed") || !bytes.Equal(readBytes(t, env.installationPath()), priorLocal) {
		t.Fatal("partial state is not portable-new / local-prior")
	}
	if len(env.editStates(t)) != 1 {
		t.Fatal("partial publication left no owned recovery state")
	}
	reopened := compose() // a new composition observes only durable state
	before := snapshotTrees(t, env.root, env.state)
	runCanonicalCLI(t, reopened, []string{"project", "show", "--selector", "sample"}, cli.ExitFailure, "validation_failure", "Local Project state requires recovery")
	runCanonicalCLI(t, reopened, []string{"project", "configure", "--project", "sample", "--name", "Other"}, cli.ExitFailure, "failure", "Selected Project state requires recovery")
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("fail-closed readers wrote state")
	}
	plan := recoveryPlanFor(t, reopened, "project_edit")
	if plan.Action != local.FinalizeCommitted {
		t.Fatalf("plan = %+v", plan)
	}
	runCanonicalCLI(t, reopened, []string{"recovery", "apply", "--preview-digest", plan.Digest, "--authorize-local"}, cli.ExitSuccess, "success", "Recovery applied: finalize_committed")
	env.assertNoEditState(t)
	runCanonicalCLI(t, reopened, []string{"project", "show", "--selector", "sample"}, cli.ExitSuccess, "success", "Project resolved")
	again, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", edit[:4]...)
	if len(again.Edit.Effects) != 0 {
		t.Fatalf("finalized state differs from the authorized candidate: %v", effectList(again.Edit.Effects))
	}
	if _, ok := bindingPaths(env.record(t))["web"]; ok {
		t.Fatal("finalized local record kept the removed binding")
	}
}

func TestEditPreCommitInterruptionRestoresPrior(t *testing.T) {
	env := newEditEnvironment(t)
	preview, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed")
	priorPortable, priorLocal := readBytes(t, env.manifestPath()), readBytes(t, env.installationPath())
	service := env.withFault(func(stage local.EditStage) error {
		if stage == local.EditStageRecorded {
			return local.ErrSimulatedInterruption
		}
		return nil
	})
	code, event, output := env.configureWith(t, service, replayArgs(preview.Edit, "--project", "sample", "--name", "Renamed")...)
	if code == cli.ExitSuccess || event.Result != "Project edit requires recovery" {
		t.Fatalf("interrupted: %s", output)
	}
	if !bytes.Equal(readBytes(t, env.manifestPath()), priorPortable) || !bytes.Equal(readBytes(t, env.installationPath()), priorLocal) || len(env.editStates(t)) != 1 {
		t.Fatal("pre-commit interruption changed state or lost its recovery state")
	}
	reopened := compose()
	env.runEdit(t, cli.ExitFailure, "failure", "Selected Project state requires recovery", "--project", "sample", "--name", "Renamed")
	plan := recoveryPlanFor(t, reopened, "project_edit")
	if plan.Action != local.RestorePrior {
		t.Fatalf("plan = %+v", plan)
	}
	runCanonicalCLI(t, reopened, []string{"recovery", "apply", "--preview-digest", plan.Digest, "--authorize-local"}, cli.ExitSuccess, "success", "Recovery applied: restore_prior")
	env.assertNoEditState(t)
	if !bytes.Equal(readBytes(t, env.manifestPath()), priorPortable) || !bytes.Equal(readBytes(t, env.installationPath()), priorLocal) {
		t.Fatal("restore_prior changed a generation")
	}
	env.publish(t, "--project", "sample", "--name", "Renamed")
}

func TestEditContradictoryCrossStoreStateIsPreservedForReview(t *testing.T) {
	env := newEditEnvironment(t)
	preview, _ := env.runEdit(t, cli.ExitSuccess, "success", "Project edit preview ready", "--project", "sample", "--name", "Renamed")
	service := env.withFault(func(stage local.EditStage) error {
		if stage == local.EditStagePortableCommitted {
			// An out-of-band writer replaced the local record with a third
			// generation the recovery state does not identify.
			env.writeRecord(t, enrichLocalMetadata)
			return errors.New("injected local failure")
		}
		return nil
	})
	if code, event, output := env.configureWith(t, service, replayArgs(preview.Edit, "--project", "sample", "--name", "Renamed")...); code == cli.ExitSuccess || event.Status != "partial" {
		t.Fatalf("contradictory: %s", output)
	}
	reopened := compose()
	plan := recoveryPlanFor(t, reopened, "project_edit")
	if plan.Action != local.PreservedReview {
		t.Fatalf("plan = %+v", plan)
	}
	before := snapshotTrees(t, env.root, env.state)
	runCanonicalCLI(t, reopened, []string{"recovery", "apply", "--preview-digest", plan.Digest, "--authorize-local"}, cli.ExitFailure, "validation_failure", "Recovery plan requires operator review")
	if !bytes.Equal(before, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("preserved state was changed")
	}
}

func containsText(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func recoveryPlanFor(t *testing.T, service cli.Service, protocol string) local.RecoveryPlan {
	t.Helper()
	var output bytes.Buffer
	if code := cli.Run(context.Background(), []string{"recovery", "inspect"}, service, currentProvenance(), &output); code != cli.ExitSuccess {
		t.Fatalf("recovery inspect: %s", output.String())
	}
	var event struct {
		Maintenance struct {
			Plans []local.RecoveryPlan `json:"plans"`
		} `json:"maintenance"`
	}
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if len(event.Maintenance.Plans) != 1 || event.Maintenance.Plans[0].Protocol != protocol {
		t.Fatalf("recovery plans = %s", output.String())
	}
	return event.Maintenance.Plans[0]
}

// Executable black-box: the built binary previews and replays an EDIT, and
// new processes observe and recover a portable-committed/local-failed edit.
func TestExecutableEditReplayAndCrossStoreRecovery(t *testing.T) {
	binary := filepath.Join(t.TempDir(), testExecutableName("lingo"))
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v: %s", err, output)
	}
	env := newEditEnvironment(t)
	environment := append(os.Environ(), "LINGO_PROJECTS_ROOT="+env.root, "LINGO_STATE_ROOT="+env.state, "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(t.TempDir(), "skills"))
	run := func(args ...string) (int, editEvent, string) {
		t.Helper()
		command := exec.Command(binary, append([]string{"--json"}, args...)...)
		command.Env, command.Dir = environment, t.TempDir()
		var stdout bytes.Buffer
		command.Stdout = &stdout
		err := command.Run()
		code := 0
		if exitError, ok := err.(*exec.ExitError); ok {
			code = exitError.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		var event editEvent
		if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &event); err != nil {
			t.Fatalf("%v: %v: %s", args, err, stdout.String())
		}
		return code, event, stdout.String()
	}
	edit := []string{"project", "configure", "--project", "sample", "--name", "Renamed", "--remove-repository", "web"}
	code, preview, output := run(edit...)
	if code != 0 || preview.Result != "Project edit preview ready" || preview.Edit.Digest == "" {
		t.Fatalf("preview: %s", output)
	}
	if code, partial, output := run(append(edit, "--project-id", preview.Edit.ProjectID, "--authorize-local")...); code == 0 || partial.Result != "Project edit authority is incomplete" {
		t.Fatalf("partial replay tuple: %s", output)
	}
	if code, published, output := run(replayArgs(preview.Edit, edit...)...); code != 0 || published.Result != "Project edit published" {
		t.Fatalf("replay: %s", output)
	}
	if code, shown, output := run("project", "show", "--selector", "sample"); code != 0 || shown.Result != "Project resolved" || strings.Contains(output, `"key":"web"`) {
		t.Fatalf("show after publish: %s", output)
	}

	// Portable committed, local failed (in-process fault), then new processes.
	rename := []string{"project", "configure", "--project", "sample", "--name", "Again"}
	_, second, _ := run(rename...)
	service := env.withFault(func(stage local.EditStage) error {
		if stage == local.EditStagePortableCommitted {
			return errors.New("injected local publication failure")
		}
		return nil
	})
	if code, event, output := env.configureWith(t, service, replayArgs(second.Edit, rename[2:]...)...); code == 0 || event.Status != "partial" {
		t.Fatalf("partial: %s", output)
	}
	if code, shown, output := run("project", "show", "--selector", "sample"); code == 0 || shown.Result != "Local Project state requires recovery" {
		t.Fatalf("show during recovery: %s", output)
	}
	if code, blocked, output := run(rename...); code == 0 || blocked.Result != "Selected Project state requires recovery" {
		t.Fatalf("edit preview during recovery: %s", output)
	}
	_, _, inspected := run("recovery", "inspect")
	var view struct {
		Maintenance struct {
			Plans []local.RecoveryPlan `json:"plans"`
		} `json:"maintenance"`
	}
	if err := json.Unmarshal([]byte(inspected), &view); err != nil || len(view.Maintenance.Plans) != 1 || view.Maintenance.Plans[0].Action != local.FinalizeCommitted {
		t.Fatalf("recovery inspect: %s", inspected)
	}
	if code, applied, output := run("recovery", "apply", "--preview-digest", view.Maintenance.Plans[0].Digest, "--authorize-local"); code != 0 || applied.Result != "Recovery applied: finalize_committed" {
		t.Fatalf("recovery apply: %s", output)
	}
	if code, after, output := run(rename...); code != 0 || len(after.Edit.Effects) != 0 {
		t.Fatalf("preview after recovery: %s", output)
	}
}
