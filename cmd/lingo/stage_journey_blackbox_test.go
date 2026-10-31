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
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const stageJourneyDecision = "Explicit synthetic human gate decision\n"

type stageJourney struct {
	env                            workItemSourceEnvironment
	axiom, bin, home               string
	runtimeID                      string
	workspace, repository, overlay string
	digests                        map[string]string
	managed                        map[string]string
	environment                    []string
}

type stageJourneyEvent struct {
	Diagnostic    *cli.Diagnostic `json:"diagnostic"`
	Status        string          `json:"status"`
	Category      string          `json:"category"`
	PreviewDigest string          `json:"previewDigest"`
	References    []string        `json:"references"`
	Stage         json.RawMessage `json:"stage"`
	Workflow      json.RawMessage `json:"workflow"`
	Payload       json.RawMessage `json:"payload"`
}

func newStageJourney(t *testing.T, concurrency ...int) *stageJourney {
	return newRuntimeStageJourney(t, "codex", concurrency...)
}

func newRuntimeStageJourney(t *testing.T, runtimeID string, concurrency ...int) *stageJourney {
	t.Helper()
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", temp)
	j := &stageJourney{runtimeID: runtimeID, env: newWorkItemSourceEnvironment(t), bin: t.TempDir(), home: t.TempDir(), digests: map[string]string{}, managed: map[string]string{}}
	t.Cleanup(func() {
		for _, name := range []string{"capability-evidence", "capability-probes"} {
			if _, err := os.Lstat(filepath.Join(j.env.state, name)); !os.IsNotExist(err) {
				t.Errorf("certification state created: %s %v", name, err)
			}
		}
		if _, err := os.Lstat(filepath.Join(j.home, "fake-gh-invocations.log")); !os.IsNotExist(err) {
			t.Errorf("unexpected Provider invocation: %v", err)
		}
		if t.Failed() {
			for _, name := range []string{"fake-stage-invocations.jsonl", "fake-stage-diagnostics.log"} {
				wire, _ := os.ReadFile(filepath.Join(j.home, name))
				t.Logf("%s: %s", name, wire)
			}
		}
	})
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(git, filepath.Join(j.bin, "git")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(j.bin, "gh"), []byte("#!/bin/sh\nprintf 'unexpected Provider invocation\\n' >> \"$HOME/fake-gh-invocations.log\"\nexit 97\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"codex", "claude"} {
		j.managed[id] = filepath.Join(j.home, "managed-"+id)
		if err := os.Mkdir(j.managed[id], 0o700); err != nil {
			t.Fatal(err)
		}
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	for _, runtimeID := range []string{"codex", "claude"} {
		path := filepath.Join(j.bin, runtimeID)
		if err := os.WriteFile(path, append(append([]byte(nil), image...), []byte("\nsynthetic-"+runtimeID+"\n")...), 0o700); err != nil {
			t.Fatal(err)
		}
		if j.digests[runtimeID], err = runtimeadapter.ExecutableIdentity(path); err != nil {
			t.Fatal(err)
		}
	}
	j.overlay = j.writeOverlay(t)
	build := t.TempDir()
	j.axiom = filepath.Join(build, "axiom")
	if output, err := exec.Command("go", "build", "-overlay", j.overlay, "-o", j.axiom, ".").CombinedOutput(); err != nil {
		t.Fatalf("build synthetic test binary: %v %s", err, output)
	}
	installLinkedTarget(t, j.env)
	resolved := j.env.installation.Resolve(context.Background(), "external")
	j.repository = resolved.Project.Repositories[0].Path
	j.initRepository(t)
	installStagePlanPolicy(t, j.env.state, workItemSourceProjectID)
	j.workspace = filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(j.workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	j.environment = []string{"HOME=" + j.home, "PATH=" + j.bin, "LINGO_PROJECTS_ROOT=" + j.env.root, "LINGO_STATE_ROOT=" + j.env.state, "AXIOM_CODEX_SKILLS_ROOT=" + filepath.Join(j.home, ".codex", "skills"), "AXIOM_GH_BIN=" + filepath.Join(t.TempDir(), "gh-never-executed"), "LANG=C"}
	j.selectWorkflow(t, concurrency...)
	return j
}

func (j *stageJourney) selectWorkflow(t *testing.T, concurrency ...int) {
	t.Helper()
	wire, err := os.ReadFile("../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	if err != nil {
		t.Fatal(err)
	}
	legacy, issues := workflowdefinition.Decode(wire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	definition := legacy.Definition
	definition.SchemaVersion = 2
	for i := range definition.Stages {
		stage := &definition.Stages[i]
		if stage.ID == "implementation" && len(concurrency) == 1 {
			stage.Concurrency = concurrency[0]
		}
		for k := range stage.Agents {
			a := &stage.Agents[k]
			a.Effort = workflowdefinition.Effort{}
			if stage.ID == "intake" {
				a.RuntimeConstraints = []string{j.runtimeID}
			}
			a.ProfileRef = "profile-codex"
			if len(a.RuntimeConstraints) == 1 && a.RuntimeConstraints[0] == "claude" {
				a.ProfileRef = "profile-claude"
			}
			if !slices.Contains(a.EffectCeilings, "repository-write") {
				a.EffectCeilings = append(a.EffectCeilings, "repository-write")
				a.Capabilities = append(a.Capabilities, "repository-write")
			}
		}
	}
	current, issues := workflowdefinition.Encode(definition)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	file := filepath.Join(t.TempDir(), "journey-schema2.json")
	if err = os.WriteFile(file, current.Canonical, 0o600); err != nil {
		t.Fatal(err)
	}
	author := j.env.service.(lifecycleService)
	author.projectsRoot = j.env.elsewhere
	if author.portable, err = local.NewPortableStore(j.env.elsewhere); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	apply := func(input cli.ProjectWorkflowInput) cli.Result {
		t.Helper()
		preview := author.ProjectWorkflow(ctx, input)
		if preview.WorkflowAuthoring == nil || preview.WorkflowAuthoring.PreviewDigest == "" {
			t.Fatalf("authoring %s", preview.Category)
		}
		input.PreviewDigest, input.ExpectedRevision, input.AuthorizeLocal = preview.WorkflowAuthoring.PreviewDigest, preview.WorkflowAuthoring.ProjectRevision, true
		out := author.ProjectWorkflow(ctx, input)
		if out.Category != "applied" {
			t.Fatalf("authoring apply %s", out.Category)
		}
		return out
	}
	created := apply(cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "create", Project: "external"}, File: file})
	apply(cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "select", Project: "external", Ref: *created.WorkflowAuthoring.Reference}})
}

func (j *stageJourney) writeOverlay(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	managedOriginal, err := filepath.Abs("../../internal/runtimeadapter/authpreflight.go")
	if err != nil {
		t.Fatal(err)
	}
	managedSource, err := os.ReadFile(managedOriginal)
	if err != nil {
		t.Fatal(err)
	}
	for original, replacement := range map[string]string{
		"/private/etc/codex/config.toml":                                filepath.Join(j.managed["codex"], "config.toml"),
		"/private/etc/codex/managed_config.toml":                        filepath.Join(j.managed["codex"], "managed_config.toml"),
		"/etc/codex/config.toml":                                        filepath.Join(j.managed["codex"], "config.toml"),
		"/etc/codex/managed_config.toml":                                filepath.Join(j.managed["codex"], "managed_config.toml"),
		"/etc/claude-code/managed-settings.json":                        filepath.Join(j.managed["claude"], "managed-settings.json"),
		"/Library/Application Support/ClaudeCode/managed-settings.json": filepath.Join(j.managed["claude"], "managed-settings.json"),
	} {
		quoted := strconv.Quote(original)
		if !bytes.Contains(managedSource, []byte(quoted)) {
			t.Fatalf("managed location absent from fixture overlay: %s", original)
		}
		managedSource = bytes.ReplaceAll(managedSource, []byte(quoted), []byte(strconv.Quote(replacement)))
	}
	managedPath := filepath.Join(dir, "authpreflight.go")
	if err := os.WriteFile(managedPath, managedSource, 0o600); err != nil {
		t.Fatal(err)
	}
	overlay, err := json.Marshal(map[string]map[string]string{"Replace": {managedOriginal: managedPath}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(path, overlay, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func (j *stageJourney) initRepository(t *testing.T) {
	t.Helper()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"}, {"-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "-q", "--allow-empty", "-m", "base"}} {
		for name, content := range map[string]string{"README.md": "synthetic\n", "decision.md": stageJourneyDecision} {
			if err := os.WriteFile(filepath.Join(j.repository, name), []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		command := exec.Command("git", args...)
		command.Dir = j.repository
		command.Env = []string{"HOME=" + j.home, "PATH=/usr/bin:/bin", "GIT_CONFIG_NOSYSTEM=1"}
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, output)
		}
	}
}

func (j *stageJourney) startExecution(t *testing.T, workItem string) *cli.WorkflowView {
	t.Helper()
	role := "engineer"
	if j.runtimeID == "claude" {
		role = "reviewer"
	}
	args := []string{"workflow", "start", "--project", "external", "--repository", "main", "--work-item", workItem, "--role", role, "--complexity", "medium", "--capabilities", "read", "--runtime", j.runtimeID, "--authorize-local"}
	preview := j.run(t, 0, args...)
	if preview.PreviewDigest == "" {
		t.Fatalf("start preview: %+v", preview)
	}
	started := j.run(t, 0, append(args, "--runtime-preview", preview.PreviewDigest)...)
	var view cli.WorkflowView
	if err := json.Unmarshal(started.Workflow, &view); err != nil || view.Binding == nil {
		t.Fatalf("start: %+v %v", started, err)
	}
	return &view
}

func (j *stageJourney) invoke(t *testing.T, binary string, args ...string) (int, stageJourneyEvent, []byte) {
	t.Helper()
	command := exec.Command(binary, append([]string{"--json"}, args...)...)
	command.Env = j.environment
	command.Dir = j.home
	output, err := command.Output()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	var event stageJourneyEvent
	if err := json.Unmarshal(bytes.TrimSpace(output), &event); err != nil {
		t.Fatalf("%v: invalid JSON %q", args, output)
	}
	if d := event.Diagnostic; d != nil {
		if code == 0 || d.SchemaVersion != 1 || len(d.Options) > 3 || d.Category != event.Category {
			t.Fatalf("invalid installed diagnostic for %v", args[:2])
		}
		for _, option := range d.Options {
			if option.Availability == "unavailable" && option.Operation != "" {
				t.Fatal("unavailable action became command")
			}
		}
	}
	return code, event, output
}

func (j *stageJourney) run(t *testing.T, want int, args ...string) stageJourneyEvent {
	t.Helper()
	code, event, output := j.invoke(t, j.axiom, args...)
	if code != want {
		t.Fatalf("%v: code=%d %s", args, code, output)
	}
	return event
}

func stageSelector(view *cli.WorkflowView) []string {
	return []string{"--project", "external", "--repository", "main", "--work-item", "github:" + view.WorkItem.Resource + "#" + view.WorkItem.ExternalID, "--execution", view.ExecutionID}
}

type workflowEvent struct {
	Workflow *cli.WorkflowView `json:"workflow"`
}

func (j *stageJourney) deliveryOutput(t *testing.T, view *cli.WorkflowView) string {
	t.Helper()
	store, err := local.NewArtifactStore(j.env.state)
	if err != nil {
		t.Fatal(err)
	}
	result := workflow.StageResult{Inputs: view.StageInputs, Outputs: map[string]workflow.Reference{}}
	inputs, _ := json.Marshal(view.StageInputs)
	sum := sha256.Sum256(inputs)
	for _, output := range view.StageContract.Outputs {
		content := []byte("Validated synthetic delivery output\n")
		if view.StageContract.Phase == "review" {
			content, err = json.Marshal(workflow.DeliveryReview{FormatVersion: 1, ExecutionID: view.ExecutionID, DefinitionDigest: view.Binding.Definition.Digest, StageID: view.CurrentGate, InputsDigest: hex.EncodeToString(sum[:]), Outcome: "pass", Findings: []workflow.DeliveryFinding{}, Waivers: []workflow.DeliveryWaiver{}})
			if err != nil {
				t.Fatal(err)
			}
		}
		artifact, err := store.Create(context.Background(), detailartifact.Draft{ExecutionID: view.ExecutionID, Category: output.Kind, Outcome: "success", Retention: detailartifact.Evidence, Markdown: content, References: []detailartifact.Reference{{Kind: "workflow-stage", Value: view.CurrentGate}, {Kind: "workflow-definition", Value: view.Binding.Definition.Digest}, {Kind: "workflow-output", Value: output.ID}}, Provenance: currentProvenance()})
		if err != nil {
			t.Fatal(err)
		}
		result.Outputs[output.ID] = workflow.Reference{Kind: "artifact", ID: artifact.ID, Digest: hex.EncodeToString(artifact.Digest[:])}
	}
	file := filepath.Join(t.TempDir(), "delivery-result.json")
	wire, _ := json.Marshal(result)
	if err = os.WriteFile(file, wire, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

// Exercise the protected run/authority stores and canonical coordination through
// the installed CLI and fake Runtime, rather than fabricating delivery evidence.
func TestDeliveryCLIValidatedCheckpointsWithoutRuns(t *testing.T) {
	j := newStageJourney(t)
	saveTargetLink(t, j.env.state, workItemSourceProjectID, "15", "owner/repo")
	view := j.startExecution(t, "github:owner/repo#15")
	selector := stageSelector(view)
	sum := sha256.Sum256([]byte(stageJourneyDecision))
	for view.Status != "completed" {
		if action := view.GateAction; action != nil && action.Operation == "fact" {
			kind := "evidence"
			if action.Fact == workflow.FactPlanningAuthority {
				kind = "specification"
			}
			if action.Fact == workflow.FactImplementationAuthority {
				kind = "plan"
			}
			code, _, wire := j.invoke(t, j.axiom, append(append([]string{"workflow", "fact"}, selector...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--fact", strings.ReplaceAll(string(action.Fact), "_", "-"), "--active", "--actor", "fixture-human", "--reference", kind+":decision.md:"+hex.EncodeToString(sum[:]), "--authorize-local")...)
			var event workflowEvent
			if code != 0 || json.Unmarshal(wire, &event) != nil || event.Workflow == nil {
				t.Fatalf("fact: %s", wire)
			}
			view = event.Workflow
		}
		file := j.deliveryOutput(t, view)
		code, _, wire := j.invoke(t, j.axiom, append(append([]string{"workflow", "advance"}, selector...), "--expected-revision", strconv.FormatUint(view.Revision, 10), "--gate", view.CurrentGate, "--outcome", "pass", "--stage-result", file)...)
		var event workflowEvent
		if code != 0 || json.Unmarshal(wire, &event) != nil || event.Workflow == nil {
			t.Fatalf("advance: %s", wire)
		}
		view = event.Workflow
	}
	event := j.run(t, 0, append([]string{"workflow", "delivery", "packet"}, selector...)...)
	var packet struct {
		Packet workflow.DeliveryPacket `json:"packet"`
	}
	if err := json.Unmarshal(event.Stage, &packet); err != nil || packet.Packet.Ready {
		t.Fatalf("packet: %s %v", event.Stage, err)
	}
	for _, stage := range packet.Packet.Stages {
		if stage.Ledger == nil || len(stage.Artifacts) == 0 || stage.Blocker != "single_stage_run_missing" {
			t.Fatalf("stage: %+v", stage)
		}
	}
	request := filepath.Join(t.TempDir(), "local.json")
	requestWire, err := json.Marshal(workflow.DeliveryTarget{Kind: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(request, requestWire, 0600); err != nil {
		t.Fatal(err)
	}
	preview := j.run(t, 0, append(append([]string{"workflow", "delivery", "preview"}, selector...), "--request", request)...)
	var value struct {
		Preview workflow.DeliveryPreview `json:"preview"`
	}
	if err := json.Unmarshal(preview.Stage, &value); err != nil {
		t.Fatal(err)
	}
	j.run(t, 1, append(append([]string{"workflow", "delivery", "apply"}, selector...), "--request", request, "--preview-digest", value.Preview.Digest(), "--authorize-local")...)
	store, err := local.NewWorkflowStore(j.env.state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadDeliveryPublication(context.Background(), packet.Packet.ProjectID, value.Preview.Digest()); err == nil {
		t.Fatal("publication created without attempts")
	}
	if wire, err := os.ReadFile(filepath.Join(j.home, "fake-stage-invocations.jsonl")); err == nil && len(bytes.TrimSpace(wire)) != 0 {
		t.Fatal("unexpected Runtime call")
	}
}
