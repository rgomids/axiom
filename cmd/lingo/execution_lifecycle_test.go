package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
)

// Issue #230 I230-T07 black-box Evidence: Execution discovery is local,
// read-only inspection that fails closed, and no cancel verb exists.

func seedExecution(t *testing.T, env admissionEnv, key, externalID, executionID string) workflow.State {
	t.Helper()
	store, err := local.NewWorkflowStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	state := workflow.State{ExecutionID: executionID, FormatVersion: workflow.FormatVersion, WorkflowVersion: workflow.WorkflowVersion, ProjectID: env.projectID, RepositoryKey: key, WorkItem: workflow.WorkItem{Provider: "github", Resource: "owner/repo", ExternalID: externalID, URL: "https://github.com/owner/repo/issues/" + externalID, State: "OPEN"}, RuntimeID: "codex", Stage: workflow.Intake, Revision: 1, Status: workflow.ExecutionActive, CreatedAt: now, UpdatedAt: now, Provenance: workflow.Identity{Product: "Axiom", Version: "development", Revision: "abc123", SourceState: "clean"}}
	if err := store.Create(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	return state
}

type listedEvent struct {
	Status     string `json:"status"`
	Executions []struct {
		ExecutionID   string `json:"executionId"`
		RepositoryKey string `json:"repositoryKey"`
		WorkItem      struct {
			ExternalID string `json:"externalId"`
		} `json:"workItem"`
		Status      string `json:"status"`
		CurrentGate string `json:"currentGate"`
		Revision    uint64 `json:"revision"`
	} `json:"executions"`
	Admission *projectapp.AdmissionDecision `json:"admission"`
}

func runList(t *testing.T, env admissionEnv, extra ...string) (int, listedEvent) {
	t.Helper()
	var output bytes.Buffer
	args := append([]string{"workflow", "list", "--project", "guarded"}, extra...)
	code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output)
	var event listedEvent
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("output %q: %v", output.String(), err)
	}
	return code, event
}

func TestWorkflowListEnumeratesDeterministicallyAndFilters(t *testing.T) {
	env := newAdmissionEnv(t)
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	code, event := runList(t, env)
	if code != cli.ExitSuccess || event.Status != string(completion.Success) || event.Executions == nil || len(event.Executions) != 0 {
		t.Fatalf("empty = %d %+v", code, event)
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("empty listing changed state")
	}
	seedExecution(t, env, "main", "9", "018f4a44-7c31-7dd4-9d00-000000000003")
	seedExecution(t, env, "main", "10", "018f4a44-7c31-7dd4-9d00-000000000002")
	seedExecution(t, env, "main", "2", "018f4a44-7c31-7dd4-9d00-000000000001")
	seeded := snapshotTrees(t, env.root, env.state, env.workspace)
	code, event = runList(t, env)
	if code != cli.ExitSuccess || len(event.Executions) != 3 {
		t.Fatalf("many = %d %+v", code, event)
	}
	// Work item order is lexical by external id, then execution id.
	for index, want := range []string{"10", "2", "9"} {
		got := event.Executions[index]
		if got.WorkItem.ExternalID != want || got.Status != "active" || got.CurrentGate != "intake" || got.Revision != 1 || got.RepositoryKey != "main" {
			t.Fatalf("row %d = %+v", index, got)
		}
	}
	_, again := runList(t, env, "--repository", "main")
	if len(again.Executions) != 3 {
		t.Fatalf("filtered = %+v", again)
	}
	// A preserved record of a detached (unattached) Repository key is not
	// listed until the key is re-attached (F-03), as in work-item list.
	seedExecution(t, env, "api", "1", "018f4a44-7c31-7dd4-9d00-000000000004")
	seeded = snapshotTrees(t, env.root, env.state, env.workspace)
	if _, all := runList(t, env); len(all.Executions) != 3 || all.Executions[0].RepositoryKey != "main" {
		t.Fatalf("detached history listed = %+v", all.Executions)
	}
	service := env.service.(lifecycleService)
	unknown := service.WorkflowList(context.Background(), cli.ExecutionListInput{Project: "guarded", Repository: "nope"})
	if unknown.Listed || unknown.Result.Category != "repository_not_configured" {
		t.Fatalf("unknown repository = %+v", unknown)
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(seeded, after) {
		t.Fatal("listing changed state")
	}
	if env.providerCalled() {
		t.Fatal("listing invoked the Provider")
	}
	// Strict parser: unknown, duplicate, missing and positional input fail.
	for _, args := range [][]string{{"workflow", "list"}, {"workflow", "list", "--project", "guarded", "--bogus", "x"}, {"workflow", "list", "--project", "guarded", "--project", "guarded"}, {"workflow", "list", "--project", "guarded", "extra"}, {"workflow", "list", "--project", "guarded", "--repository="}} {
		var output bytes.Buffer
		if code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output); code != cli.ExitFailure || bytes.Contains(output.Bytes(), []byte("bogus")) {
			t.Fatalf("%v accepted: %d %s", args, code, output.String())
		}
	}
}

func TestWorkflowListFailsClosedOnDamagedState(t *testing.T) {
	for name, damage := range map[string]func(t *testing.T, dir string, state workflow.State){
		"corrupt record": func(t *testing.T, dir string, _ workflow.State) {
			if err := os.WriteFile(filepath.Join(dir, "garbage.json"), []byte("{not-json}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"interrupted protocol": func(t *testing.T, dir string, _ workflow.State) {
			if err := os.WriteFile(filepath.Join(dir, ".axiom-recovery-x"), []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			env := newAdmissionEnv(t)
			state := seedExecution(t, env, "main", "7", "018f4a44-7c31-7dd4-9d00-000000000001")
			damage(t, filepath.Join(env.state, "executions", "v1", env.projectID), state)
			before := snapshotTrees(t, env.root, env.state, env.workspace)
			code, event := runList(t, env)
			if code != cli.ExitFailure || event.Executions != nil {
				t.Fatalf("damaged listing = %d %+v", code, event)
			}
			if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
				t.Fatal("failed listing changed state")
			}
		})
	}
}

func TestWorkflowListIsInspectionUnderArchiveAndDisabledIntegration(t *testing.T) {
	env := newAdmissionEnv(t)
	seedExecution(t, env, "main", "7", "018f4a44-7c31-7dd4-9d00-000000000001")
	env.apply(t, projectapp.DisableIntegration, "work-items")
	if code, event := runList(t, env); code != cli.ExitSuccess || len(event.Executions) != 1 || event.Admission != nil {
		t.Fatalf("disabled Integration list = %d %+v", code, event)
	}
	env.apply(t, projectapp.EnableIntegration, "work-items")
	env.apply(t, projectapp.ArchiveProject, "")
	code, event := runList(t, env)
	if code != cli.ExitSuccess || len(event.Executions) != 1 || event.Admission != nil {
		t.Fatalf("archived list = %d %+v", code, event)
	}
	service := env.service.(lifecycleService)
	for name, result := range map[string]cli.Result{
		"status":   service.WorkflowStatus(context.Background(), cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 7}),
		"evidence": service.WorkflowEvidence(context.Background(), cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 7}),
	} {
		if result.Admission != nil || result.Category == projectapp.AdmissionProjectArchived {
			t.Fatalf("%s denied while archived: %+v", name, result)
		}
	}
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	for _, name := range []string{"start preview", "start", "advance", "fact", "resume", "reconcile preview", "reconcile"} {
		result := evolution(service)[name]
		if result.Category != projectapp.AdmissionProjectArchived || result.Admission == nil {
			t.Fatalf("%s not denied while archived: %+v", name, result)
		}
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("denied evolution changed state")
	}
	if env.providerCalled() {
		t.Fatal("Provider invoked")
	}
}

func TestWorkflowCancelStaysInvalidCommandWithZeroEffects(t *testing.T) {
	env := newAdmissionEnv(t)
	state := seedExecution(t, env, "main", "7", "018f4a44-7c31-7dd4-9d00-000000000001")
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	record := filepath.Join(env.state, "executions", "v1", env.projectID)
	entries, err := os.ReadDir(record)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, %v", entries, err)
	}
	wire, _ := os.ReadFile(filepath.Join(record, entries[0].Name()))
	for _, args := range [][]string{
		{"workflow", "cancel", "--project", "guarded", "--repository", "main", "--number", "7", "--execution", state.ExecutionID},
		{"workflow", "cancel"},
	} {
		var output bytes.Buffer
		code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output)
		var event struct {
			Status   string `json:"status"`
			Category string `json:"category"`
		}
		if err := json.Unmarshal(output.Bytes(), &event); err != nil || code != cli.ExitFailure || event.Category != "invalid_command" {
			t.Fatalf("%v = %d %s", args, code, output.String())
		}
	}
	after, _ := os.ReadFile(filepath.Join(record, entries[0].Name()))
	if !bytes.Equal(wire, after) || !bytes.Equal(before, snapshotTrees(t, env.root, env.state, env.workspace)) {
		t.Fatal("cancel changed state")
	}
	if env.providerCalled() {
		t.Fatal("cancel invoked the Provider")
	}
}
