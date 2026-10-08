package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

// Issue #230 I230-T05 black-box Evidence: the central admission guard denies
// operational evolution on an archived Project and any use of a locally
// disabled Integration before readiness, Provider calls or local writes,
// while inspection and administration stay available.

type admissionEnv struct {
	bootstrapEnv
	calls     string
	projectID string
}

func newAdmissionEnv(t *testing.T) admissionEnv {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Provider spy is a POSIX script")
	}
	env := admissionEnv{bootstrapEnv: newBootstrapEnv(t)}
	env.calls = filepath.Join(t.TempDir(), "gh-calls")
	gh := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\necho called >> '"+env.calls+"'\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_GH_BIN", gh)
	env.service = compose()
	repository := writeTree(t, filepath.Join(env.workspace, "main"), map[string]string{"README.md": "x"})
	env.projectID = configureProject(t, env.service, "guarded", "Guarded", "main="+repository, "github").ProjectID
	return env
}

func (env admissionEnv) apply(t *testing.T, operation projectapp.OperationalOperation, integration string) {
	t.Helper()
	store, err := local.NewOperationalStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	request := projectapp.OperationalRequest{ProjectID: env.projectID, Operation: operation, Integration: integration}
	preview := projectapp.ApplyOperational(context.Background(), store, request, "", false)
	if preview.Preview == nil {
		t.Fatalf("preview = %+v", preview)
	}
	if result := projectapp.ApplyOperational(context.Background(), store, request, preview.Preview.Digest, true); result.Status != projectapp.OperationalCommitted {
		t.Fatalf("apply %s = %+v", operation, result)
	}
}

func (env admissionEnv) providerCalled() bool {
	_, err := os.Stat(env.calls)
	return !os.IsNotExist(err)
}

// evolution runs every operational-evolution entrypoint, each in its preview
// and in its authorized form.
func evolution(service lifecycleService) map[string]cli.Result {
	ctx := context.Background()
	item := cli.WorkItemInput{Project: "guarded", Repository: "main", ProviderRepository: "owner/repo", Number: 1, Message: "x", Intent: "x", Problem: "x", DesiredOutcome: "x", Context: "x", Scope: "x", Constraints: "x", NonGoals: "x", Acceptance: "x"}
	authorized := item
	authorized.PreviewDigest, authorized.AuthorizeExternal, authorized.AuthorizeLocal = "any", true, true
	flow := cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 1, ExpectedRevision: 1, Role: "implementation", Complexity: "high", Capabilities: []string{"axiom-skills"}, Gate: "planning", Outcome: "pass", Reference: "evidence:x", Fact: "blocked", Active: true}
	authorizedFlow := flow
	authorizedFlow.RuntimePreview, authorizedFlow.PreviewDigest, authorizedFlow.AuthorizeExternal, authorizedFlow.AuthorizeLocal = "any", "any", true, true
	return map[string]cli.Result{
		"create preview":    service.WorkItemCreate(ctx, item),
		"create":            service.WorkItemCreate(ctx, authorized),
		"select preview":    service.WorkItemSelect(ctx, item),
		"select":            service.WorkItemSelect(ctx, authorized),
		"comment":           service.WorkItemComment(ctx, authorized),
		"complete":          service.WorkItemComplete(ctx, authorized),
		"start preview":     service.WorkflowStart(ctx, flow),
		"start":             service.WorkflowStart(ctx, authorizedFlow),
		"advance":           service.WorkflowAdvance(ctx, flow),
		"fact":              service.WorkflowFact(ctx, authorizedFlow),
		"resume":            service.WorkflowResume(ctx, flow),
		"reconcile preview": service.WorkflowReconcile(ctx, flow),
		"reconcile":         service.WorkflowReconcile(ctx, authorizedFlow),
	}
}

func TestArchivedProjectDeniesEvolutionBeforeAnyEffect(t *testing.T) {
	env := newAdmissionEnv(t)
	env.apply(t, projectapp.ArchiveProject, "")
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	for name, result := range evolution(env.service.(lifecycleService)) {
		if result.Completion == nil || result.Completion.Status() != completion.ValidationFailure || result.Category != projectapp.AdmissionProjectArchived {
			t.Fatalf("%s: %+v", name, result)
		}
		if result.Admission == nil || result.Admission.Allowed || result.Admission.Class != projectapp.AdmissionEvolution || result.Admission.ProjectID != env.projectID || result.Admission.GrantsAuthority || result.Preflight != nil {
			t.Fatalf("%s admission = %+v", name, result.Admission)
		}
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("denied evolution changed portable, local or Repository state")
	}
	if env.providerCalled() {
		t.Fatal("denied evolution invoked the Provider")
	}
	// Inspection is admitted: these fail for their own reasons, never archive.
	service := env.service.(lifecycleService)
	for name, result := range map[string]cli.Result{
		"work-item show":    service.WorkItemShow(context.Background(), cli.WorkItemInput{Project: "guarded", Repository: "main", Number: 1}),
		"workflow status":   service.WorkflowStatus(context.Background(), cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 1}),
		"workflow evidence": service.WorkflowEvidence(context.Background(), cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 1}),
	} {
		if result.Category == projectapp.AdmissionProjectArchived || result.Admission != nil {
			t.Fatalf("%s denied by archive: %+v", name, result)
		}
	}
	// Reactivation restores admission; the next denial is the operation's own.
	env.apply(t, projectapp.ReactivateProject, "")
	if result := service.WorkflowAdvance(context.Background(), cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 1, ExpectedRevision: 1, Gate: "planning", Outcome: "pass", Reference: "evidence:x"}); result.Admission != nil || result.Category == projectapp.AdmissionProjectArchived {
		t.Fatalf("reactivated Project still denied: %+v", result)
	}
}

func TestDisabledIntegrationDeniesOnlyItsProviderUse(t *testing.T) {
	env := newAdmissionEnv(t)
	env.apply(t, projectapp.DisableIntegration, "work-items")
	before := snapshotTrees(t, env.root, env.state, env.workspace)
	results := evolution(env.service.(lifecycleService))
	for _, name := range []string{"create preview", "create", "select preview", "select", "comment", "complete", "reconcile preview", "reconcile"} {
		result := results[name]
		if result.Category != projectapp.AdmissionIntegrationDisabled || result.Admission == nil || result.Admission.Integration != "work-items" || result.Preflight != nil {
			t.Fatalf("%s: %+v", name, result)
		}
	}
	// Execution transitions use no Provider through the Integration.
	for _, name := range []string{"start preview", "start", "advance", "fact", "resume"} {
		if result := results[name]; result.Admission != nil || result.Category == projectapp.AdmissionIntegrationDisabled {
			t.Fatalf("%s denied by a disabled Integration it does not use: %+v", name, result)
		}
	}
	if env.providerCalled() {
		t.Fatal("disabled Integration reached the Provider")
	}
	if after := snapshotTrees(t, env.root, env.state, env.workspace); !bytes.Equal(before, after) {
		t.Fatal("denied Provider use changed state")
	}
	portable, err := os.ReadFile(filepath.Join(env.root, "guarded", "axiom.yaml"))
	if err != nil || !bytes.Contains(portable, []byte("work-items")) {
		t.Fatalf("disable changed the portable declaration: %s %v", portable, err)
	}
}

func TestCorruptOperationalStateFailsClosedForEvolutionOnly(t *testing.T) {
	env := newAdmissionEnv(t)
	record := filepath.Join(env.state, "projects", env.projectID, "operational.json")
	if err := os.WriteFile(record, []byte(`{"formatVersion":9}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := env.service.(lifecycleService)
	result := service.WorkflowAdvance(context.Background(), cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 1, ExpectedRevision: 1, Gate: "planning", Outcome: "pass", Reference: "evidence:x"})
	if result.Category != projectapp.OperationalStateInvalid || result.Admission == nil {
		t.Fatalf("corrupt state admitted evolution: %+v", result)
	}
	if status := service.WorkflowStatus(context.Background(), cli.WorkflowInput{Project: "guarded", Repository: "main", Number: 1}); status.Admission != nil {
		t.Fatalf("inspection consulted corrupt operational state: %+v", status)
	}
	if env.providerCalled() {
		t.Fatal("corrupt state reached the Provider")
	}
	wire, _ := os.ReadFile(record)
	if string(wire) != `{"formatVersion":9}`+"\n" {
		t.Fatal("corrupt record was rewritten")
	}
}

func TestAdmissionDenialIsTheSameThroughTheCLIEntrypoint(t *testing.T) {
	env := newAdmissionEnv(t)
	env.apply(t, projectapp.ArchiveProject, "")
	var output bytes.Buffer
	args := []string{"work-item", "comment", "--project", "guarded", "--repository", "main", "--number", "1", "--message", "x", "--authorize-external"}
	if code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output); code != cli.ExitFailure {
		t.Fatalf("exit %d output=%s", code, output.String())
	}
	var event struct {
		Status    string                        `json:"status"`
		Admission *projectapp.AdmissionDecision `json:"admission"`
	}
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	direct := env.service.(lifecycleService).WorkItemComment(context.Background(), cli.WorkItemInput{Project: "guarded", Repository: "main", Number: 1, Message: "x", AuthorizeExternal: true})
	if event.Status != string(completion.ValidationFailure) || event.Admission == nil || *event.Admission != *direct.Admission {
		t.Fatalf("CLI %+v != service %+v", event.Admission, direct.Admission)
	}
	if env.providerCalled() {
		t.Fatal("CLI denial invoked the Provider")
	}
}

// Positive control for the Provider spy: an admitted, authorized Provider
// operation does reach the stub, so every zero-call assertion above can fail.
func TestProviderSpyRecordsAdmittedProviderCalls(t *testing.T) {
	env := newAdmissionEnv(t)
	env.service.(lifecycleService).WorkItemSelect(context.Background(), cli.WorkItemInput{Project: "guarded", Repository: "main", ProviderRepository: "owner/repo", Number: 1})
	if !env.providerCalled() {
		t.Fatal("admitted Work Item select never reached the Provider spy")
	}
}
