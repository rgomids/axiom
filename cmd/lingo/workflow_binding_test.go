package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func selectBuiltinWorkflowExecutable(t *testing.T, binary string, environment []string, project string) {
	t.Helper()
	ref := workflowdefinition.Builtin().Ref("builtin")
	args := []string{"--json", "project", "workflow", "select", "--project", project, "--workflow", ref.WorkflowID, "--revision", strconv.Itoa(ref.Revision), "--digest", ref.Digest, "--source", ref.Source}
	run := func(category string, args []string) authoringEvent {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = environment
		wire, err := command.CombinedOutput()
		var event authoringEvent
		if err != nil || json.Unmarshal(wire, &event) != nil || event.Category != category {
			t.Fatalf("workflow selection: %v %s", err, wire)
		}
		return event
	}
	preview := run("previewed", args)
	run("applied", append(args, "--expected-revision", preview.Workflow.ProjectRevision, "--preview-digest", preview.Workflow.PreviewDigest, "--authorize-local"))
}

func publishConfiguredOutput(t *testing.T, stateRoot string, view *cli.WorkflowView) string {
	t.Helper()
	store, err := local.NewArtifactStore(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	value := workflow.StageResult{Inputs: view.StageInputs, Outputs: map[string]workflow.Reference{}}
	for _, output := range view.StageContract.Outputs {
		a, err := store.Create(context.Background(), detailartifact.Draft{ExecutionID: view.ExecutionID, Category: output.Kind, Outcome: "success", Retention: detailartifact.Evidence, Markdown: []byte("# Bounded output\n"), References: []detailartifact.Reference{{Kind: "workflow-stage", Value: view.StageContract.ID}, {Kind: "workflow-definition", Value: view.Binding.Definition.Digest}, {Kind: "workflow-output", Value: output.ID}}, Provenance: currentProvenance()})
		if err != nil {
			t.Fatal(err)
		}
		value.Outputs[output.ID] = workflow.Reference{Kind: "artifact", ID: a.ID, Digest: hex.EncodeToString(a.Digest[:])}
	}
	file := filepath.Join(t.TempDir(), "stage-result.json")
	wire, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, wire, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

// The executable uses real protected stores and authored revisions; Runtime
// stubs are only observed, never invoked. No Provider/network effect is needed.
func TestConfiguredExecutableRevisionIsolationAndAuthority(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	installLinkedTarget(t, env)
	installTestRuntimePolicy(t, env.state, workItemSourceProjectID, "codex")
	binary := filepath.Join(t.TempDir(), testExecutableName("axiom"))
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, output)
	}
	runtimeDir := t.TempDir()
	runtimePath := filepath.Join(runtimeDir, testExecutableName("codex"))
	if err := os.WriteFile(runtimePath, []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
		t.Fatal(err)
	}
	isolatedHome := t.TempDir()
	environment := append(os.Environ(), "PATH="+runtimeDir, "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(t.TempDir(), "skills"), "USERPROFILE="+isolatedHome, "HOME="+isolatedHome)
	type event struct {
		Status        string            `json:"status"`
		Next          string            `json:"next"`
		Workflow      *cli.WorkflowView `json:"workflow"`
		PreviewDigest string            `json:"previewDigest"`
	}
	run := func(code int, args ...string) event {
		t.Helper()
		command := exec.Command(binary, append([]string{"--json"}, args...)...)
		command.Env = environment
		wire, err := command.Output()
		actual := 0
		if exit, ok := err.(*exec.ExitError); ok {
			actual = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		var value event
		if json.Unmarshal(bytes.TrimSpace(wire), &value) != nil || actual != code {
			t.Fatalf("%v code=%d: %s", args, actual, wire)
		}
		return value
	}
	run(0, "runtime", "codex", "install")
	selector := []string{"--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7"}
	start := append([]string{"workflow", "start"}, selector...)
	start = append(start, "--role", "implementation", "--complexity", "high", "--capabilities", "axiom-skills")
	preview := run(0, start...)
	a := run(0, append(start, "--runtime-preview", preview.PreviewDigest)...).Workflow
	if a == nil || a.Binding == nil || a.Binding.Definition.Revision != 1 {
		t.Fatal("missing R1 binding")
	}
	execution := append(append([]string{}, selector...), "--execution", a.ExecutionID)
	// Publish custom R1 and R2 and select R2 through the supported Project surface.
	service := env.service.(lifecycleService)
	request := cli.ProjectWorkflowInput{}
	request.Project = "external"
	request.WorkflowRequest.Project = "external"
	// The installed source is outside the author's configured root, so use the
	// same root that owns its recorded companion directory for these operations.
	service.projectsRoot = env.elsewhere
	portable, err := local.NewPortableStore(env.elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	service.portable = portable
	apply := func(input cli.ProjectWorkflowInput) cli.Result {
		t.Helper()
		p := service.ProjectWorkflow(context.Background(), input)
		if p.WorkflowAuthoring == nil || p.WorkflowAuthoring.PreviewDigest == "" {
			t.Fatalf("authoring %s", p.Category)
		}
		input.PreviewDigest = p.WorkflowAuthoring.PreviewDigest
		input.ExpectedRevision = p.WorkflowAuthoring.ProjectRevision
		input.AuthorizeLocal = true
		out := service.ProjectWorkflow(context.Background(), input)
		if out.Category != "applied" && out.Category != "unchanged" {
			t.Fatalf("authoring apply %s", out.Category)
		}
		return out
	}
	request.Operation = "create"
	request.WorkflowID = "custom"
	request.FromDefault = true
	created := apply(request)
	inspected := service.ProjectWorkflow(context.Background(), cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "show", Project: "external", Ref: *created.WorkflowAuthoring.Reference}})
	if inspected.WorkflowAuthoring == nil || inspected.WorkflowAuthoring.Definition == nil {
		t.Fatal("missing custom revision")
	}
	definition := *inspected.WorkflowAuthoring.Definition
	definition.Revision = 2
	definition.Stages[0].Instructions = "R2 execution instructions"
	wire, _ := json.Marshal(definition)
	file := filepath.Join(t.TempDir(), "definition.json")
	if err = os.WriteFile(file, wire, 0600); err != nil {
		t.Fatal(err)
	}
	request.Operation = "edit"
	request.FromDefault = false
	request.File = file
	request.Prior = *created.WorkflowAuthoring.Reference
	edited := apply(request)
	request = cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "select", Project: "external", Ref: *edited.WorkflowAuthoring.Reference}}
	apply(request)
	// A retains R1, while B's stale preview cannot authorize a changed selection.
	status := run(0, append([]string{"workflow", "status"}, execution...)...).Workflow
	if status.Binding.Definition != a.Binding.Definition || status.StageContract.Instructions != a.StageContract.Instructions {
		t.Fatal("A drifted to current Project defaults")
	}
	bSelector := []string{"--project", "external", "--repository", "main", "--work-item", "github:owner/repo#8"}
	bStart := append([]string{"workflow", "start"}, bSelector...)
	bStart = append(bStart, "--role", "implementation", "--complexity", "high", "--capabilities", "axiom-skills")
	bPreview := run(0, bStart...)
	b := run(0, append(bStart, "--runtime-preview", bPreview.PreviewDigest)...).Workflow
	if b.Binding.Definition.Revision != 2 || b.Binding.Definition.WorkflowID != "custom" {
		t.Fatal("B did not bind R2")
	}
	run(1, append(append([]string{"workflow", "advance"}, execution...), "--expected-revision", "1", "--gate", a.CurrentGate, "--outcome", "pass")...)
	value := publishConfiguredOutput(t, env.state, a)
	advanced := run(0, append(append([]string{"workflow", "advance"}, execution...), "--expected-revision", "1", "--gate", a.CurrentGate, "--outcome", "pass", "--stage-result", value)...).Workflow
	if advanced.Revision != 2 || len(advanced.StageLedger) != 1 {
		t.Fatal("missing persisted validation ledger")
	}
	stopped := run(2, append(append([]string{"workflow", "advance"}, execution...), "--expected-revision", "2", "--gate", advanced.CurrentGate, "--outcome", "fail")...).Workflow
	resumed := run(0, append(append([]string{"workflow", "resume"}, execution...), "--expected-revision", strconv.FormatUint(stopped.Revision, 10))...).Workflow
	if resumed.Binding.Definition != a.Binding.Definition {
		t.Fatal("resume rebound R2")
	}
	for i := 0; i < 2; i++ {
		file := publishConfiguredOutput(t, env.state, resumed)
		resumed = run(0, append(append([]string{"workflow", "advance"}, execution...), "--expected-revision", strconv.FormatUint(resumed.Revision, 10), "--gate", resumed.CurrentGate, "--outcome", "pass", "--stage-result", file)...).Workflow
	}
	if resumed.CurrentGate != "plan" {
		t.Fatal("wrong planning frontier")
	}
	file = publishConfiguredOutput(t, env.state, resumed)
	if denied := run(1, append(append([]string{"workflow", "advance"}, execution...), "--expected-revision", strconv.FormatUint(resumed.Revision, 10), "--gate", "plan", "--outcome", "pass", "--stage-result", file)...); denied.Status != "denied_authority" {
		t.Fatalf("human gate bypass: %s", denied.Status)
	}
	if denied := run(1, append(append([]string{"workflow", "fact"}, execution...), "--expected-revision", strconv.FormatUint(resumed.Revision, 10), "--fact", "planning-authority", "--active", "--actor", "agent", "--reference", "specification:missing:"+a.Binding.Definition.Digest)...); denied.Status != "denied_authority" {
		t.Fatalf("agent fact granted authority: %s", denied.Status)
	}
	evidence := run(0, append([]string{"workflow", "evidence"}, execution...)...).Workflow
	if evidence.Revision != resumed.Revision || evidence.Binding.Definition != a.Binding.Definition || len(evidence.StageLedger) != 3 || len(evidence.Blockers) == 0 {
		t.Fatal("authority denial changed canonical Evidence")
	}
	resolved := env.installation.Resolve(context.Background(), "external")
	decision := []byte("Explicit synthetic human gate decision\n")
	if err := os.WriteFile(filepath.Join(resolved.Project.Repositories[0].Path, "decision.md"), decision, 0600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(decision)
	decisionDigest := hex.EncodeToString(hash[:])
	for count := 0; resumed.Status != "completed" && count < 16; count++ {
		if action := resumed.GateAction; action != nil && action.Operation == "fact" {
			kind := map[workflow.LifecycleFactKind]string{workflow.FactPlanningAuthority: "specification", workflow.FactImplementationAuthority: "plan", workflow.FactReviewStarted: "evidence"}[action.Fact]
			resumed = run(0, append(append([]string{"workflow", "fact"}, execution...), "--expected-revision", strconv.FormatUint(resumed.Revision, 10), "--fact", strings.ReplaceAll(string(action.Fact), "_", "-"), "--active", "--actor", "maintainer", "--reference", kind+":decision.md:"+decisionDigest, "--authorize-local")...).Workflow
		}
		file := publishConfiguredOutput(t, env.state, resumed)
		resumed = run(0, append(append([]string{"workflow", "advance"}, execution...), "--expected-revision", strconv.FormatUint(resumed.Revision, 10), "--gate", resumed.CurrentGate, "--outcome", "pass", "--stage-result", file)...).Workflow
	}
	if resumed.Status != "completed" || resumed.LifecycleStage != "reviewed" || resumed.GateAction != nil || len(resumed.GateCommand) != 0 || len(resumed.Blockers) != 1 || resumed.Blockers[0] != "delivery_packet_required" {
		t.Fatalf("unsupported acceptance advertised: %+v", resumed)
	}
	before := snapshotTrees(t, env.state)
	denied := run(1, append(append([]string{"workflow", "fact"}, execution...), "--expected-revision", strconv.FormatUint(resumed.Revision, 10), "--fact", "human-acceptance", "--active", "--actor", "maintainer", "--reference", "evidence:decision.md:"+decisionDigest, "--authorize-local")...)
	if denied.Status != "denied_authority" || denied.Workflow.LifecycleStage != "reviewed" || !bytes.Equal(before, snapshotTrees(t, env.state)) || !strings.Contains(denied.Next, "#277") {
		t.Fatal("unrelated valid Evidence granted acceptance")
	}
	finalEvidence := run(0, append([]string{"workflow", "evidence"}, execution...)...).Workflow
	if finalEvidence.Revision != resumed.Revision || finalEvidence.LifecycleStage != "reviewed" || len(finalEvidence.StageLedger) != resumed.StageOrdinal+1 {
		t.Fatal("denied acceptance rewrote final Evidence")
	}
}
