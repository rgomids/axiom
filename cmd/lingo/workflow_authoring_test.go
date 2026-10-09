package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type authoringEvent struct {
	Category string                    `json:"category"`
	Status   string                    `json:"status"`
	Workflow projectapp.WorkflowReport `json:"workflowAuthoring"`
}

func TestExecutableWorkflowDefaultCustomEditSelectAndRetire(t *testing.T) {
	env := newBootstrapEnv(t)
	repository := writeTree(t, filepath.Join(env.workspace, "main"), map[string]string{"README.md": "untouched"})
	configureProject(t, env.service, "workflows", "Workflows", "main="+repository, "none")
	binary := filepath.Join(t.TempDir(), testExecutableName("axiom"))
	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("build %v %s", e, output)
	}
	run := func(category string, args ...string) authoringEvent {
		t.Helper()
		command := exec.Command(binary, append([]string{"--json", "project", "workflow"}, append(args, "--project", "workflows")...)...)
		output, e := command.CombinedOutput()
		var event authoringEvent
		if json.Unmarshal(output, &event) != nil || event.Category != category {
			t.Fatalf("%v => %s %v; want %s", args, output, e, category)
		}
		success := category == "listed" || category == "inspected" || category == "previewed" || category == "applied" || category == "unchanged" || category == "validated"
		if (e == nil) != success {
			t.Fatalf("wrong exit for %s: %v %s", category, e, output)
		}
		return event
	}
	authorize := func(preview authoringEvent, args []string) []string {
		return append(append([]string{}, args...), "--expected-revision", preview.Workflow.ProjectRevision, "--preview-digest", preview.Workflow.PreviewDigest, "--authorize-local")
	}
	refArgs := func(op string, ref workflowdefinition.Ref) []string {
		return []string{op, "--workflow", ref.WorkflowID, "--revision", strconv.Itoa(ref.Revision), "--digest", ref.Digest, "--source", ref.Source}
	}
	list := run("listed", "list")
	if list.Workflow.Selection != nil || len(list.Workflow.Revisions) != 1 {
		t.Fatal("fresh Project implicitly selected a workflow")
	}
	builtin := list.Workflow.Revisions[0]
	selectDefault := refArgs("select", builtin)
	preview := run("previewed", selectDefault...)
	stale := authorize(preview, selectDefault)
	stale[len(stale)-2] = strings.Repeat("a", 64)
	run("stale_authority", stale...)
	run("applied", authorize(preview, selectDefault)...)
	run("unchanged", authorize(preview, selectDefault)...)
	if selected := run("listed", "list").Workflow.Selection; selected == nil || *selected != builtin {
		t.Fatal("default not selected")
	}
	create := []string{"create", "--from-default", "--workflow", "custom"}
	preview = run("previewed", create...)
	run("applied", authorize(preview, create)...)
	r1 := *preview.Workflow.Reference
	selection := refArgs("select", r1)
	preview = run("previewed", selection...)
	run("applied", authorize(preview, selection)...)
	r1Document := run("inspected", refArgs("show", r1)...).Workflow.Definition
	if r1Document == nil {
		t.Fatal("custom definition missing")
	}
	customWire, err := os.ReadFile("../../docs/specifications/007-configurable-workflows/examples/custom-r1.json")
	if err != nil {
		t.Fatal(err)
	}
	custom, issues := workflowdefinition.Decode(customWire)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	next := custom.Definition
	next.WorkflowID = r1.WorkflowID
	next.Revision = 2
	next.Stages[0].Instructions += " Preserve reviewed context."
	file := filepath.Join(t.TempDir(), "revision.json")
	wire, _ := json.Marshal(next)
	if e := os.WriteFile(file, wire, 0o600); e != nil {
		t.Fatal(e)
	}
	run("validated", "validate", "--file", file)
	edit := []string{"edit", "--workflow", r1.WorkflowID, "--prior-revision", "1", "--prior-digest", r1.Digest, "--file", file}
	preview = run("previewed", edit...)
	r2 := *preview.Workflow.Reference
	run("applied", authorize(preview, edit)...)
	run("unchanged", authorize(preview, edit)...)
	inspected := run("inspected", refArgs("show", r2)...).Workflow.Definition
	expectedWire, _ := json.Marshal(next)
	inspectedWire, _ := json.Marshal(inspected)
	if string(expectedWire) != string(inspectedWire) {
		t.Fatal("custom stage/agent fields changed during authoring")
	}
	if selected := run("listed", "list").Workflow.Selection; selected == nil || *selected != r1 {
		t.Fatal("edit changed selection")
	}
	selection = refArgs("select", r2)
	oldPreview := run("previewed", selection...)
	preview = run("previewed", selectDefault...)
	run("applied", authorize(preview, selectDefault)...)
	run("stale_authority", authorize(oldPreview, selection)...)
	preview = run("previewed", selection...)
	run("applied", authorize(preview, selection)...)
	next.Revision = 3
	next.Stages[0].Agents[0].DependsOn = []string{next.Stages[0].Agents[0].ID}
	wire, _ = json.Marshal(next)
	os.WriteFile(file, wire, 0o600)
	run("invalid_definition", "create", "--file", file)
	if selected := run("listed", "list").Workflow.Selection; selected == nil || *selected != r2 {
		t.Fatal("invalid edit altered valid selection")
	}
	run("revision_in_use", refArgs("remove", r2)...)
	preview = run("previewed", refArgs("remove", r1)...)
	run("applied", authorize(preview, refArgs("remove", r1))...)
	run("inspected", refArgs("show", r1)...)
	run("workflow_not_found", refArgs("select", r1)...)
	before := readBytes(t, filepath.Join(repository, "README.md"))
	if string(before) != "untouched" {
		t.Fatal("workflow authoring changed Repository")
	}
	command := exec.Command(binary, "--json", "workflow", "list", "--project", "workflows")
	output, e := command.CombinedOutput()
	if e != nil || strings.Contains(string(output), "workflowAuthoring") {
		t.Fatalf("legacy list semantics changed: %s %v", output, e)
	}
	// An unclassified future Execution/reference format blocks retirement.
	preview = run("previewed", selectDefault...)
	run("applied", authorize(preview, selectDefault)...)
	os.WriteFile(filepath.Join(env.state, "unknown-reference.json"), []byte(`{}`), 0o600)
	run("reference_inventory_unknown", refArgs("remove", r2)...)
}
