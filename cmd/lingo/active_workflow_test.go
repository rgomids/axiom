package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

func workflow303Run(t *testing.T, env bootstrapEnv, category string, args ...string) authoringEvent {
	t.Helper()
	var output bytes.Buffer
	code := cli.Run(context.Background(), append([]string{"project", "workflow"}, append(args, "--project", "workflows")...), env.service, currentProvenance(), &output)
	var got authoringEvent
	if err := json.Unmarshal(output.Bytes(), &got); err != nil || got.Category != category {
		t.Fatalf("%v: code=%d %s (%v), want %s", args, code, output.String(), err, category)
	}
	return got
}
func workflow303Authority(preview authoringEvent, args []string) []string {
	return append(append([]string{}, args...), "--expected-revision", preview.Workflow.ProjectRevision, "--preview-digest", preview.Workflow.PreviewDigest, "--authorize-local")
}
func workflow303Ref(op string, ref workflowdefinition.Ref) []string {
	return []string{op, "--workflow", ref.WorkflowID, "--revision", strconv.Itoa(ref.Revision), "--digest", ref.Digest, "--source", ref.Source}
}
func workflow303Env(t *testing.T) bootstrapEnv {
	env := newBootstrapEnv(t)
	repo := writeTree(t, filepath.Join(env.workspace, "main"), map[string]string{"README.md": "untouched"})
	configureProject(t, env.service, "workflows", "Workflows", "main="+repo, "none")
	return env
}
func active303Read(t *testing.T, env bootstrapEnv, status string) {
	t.Helper()
	before := snapshot303Trees(t, env.root, env.state, env.workspace)
	for _, args := range [][]string{{"project", "show", "--selector", "workflows"}, {"project", "list"}} {
		var output bytes.Buffer
		if code := cli.Run(context.Background(), args, env.service, currentProvenance(), &output); code != cli.ExitSuccess {
			t.Fatalf("%v: %d %s", args, code, output.String())
		}
		var event struct {
			Project  *cli.ProjectView      `json:"project"`
			Projects []cli.ProjectListView `json:"projects"`
		}
		if err := json.Unmarshal(output.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		var active *projectapp.ActiveWorkflow
		if event.Project != nil {
			active = event.Project.ActiveWorkflow
		} else {
			for _, p := range event.Projects {
				if p.Slug == "workflows" {
					active = p.ActiveWorkflow
				}
			}
		}
		if active == nil || active.Status != status {
			t.Fatalf("%v: %+v", args, active)
		}
		if status == "unresolvable" && (active.Name != "" || active.Category != "recovery_required" || active.WorkflowID == "") {
			t.Fatalf("unresolved ref: %+v", active)
		}
		t.Logf("R10 JSON %s", output.String())
		output.Reset()
		if code := cli.RunInteractive(context.Background(), args, env.service, currentProvenance(), nil, &output, nil); code != cli.ExitSuccess || !strings.Contains(output.String(), "activeWorkflow") || !strings.Contains(output.String(), status) {
			t.Fatalf("human: %d %s", code, output.String())
		}
		t.Logf("R10 human %s", output.String())
	}
	if !bytes.Equal(before, snapshot303Trees(t, env.root, env.state, env.workspace)) {
		t.Fatal("read changed tree")
	}
}
func active303AssertRef(t *testing.T, env bootstrapEnv, ref workflowdefinition.Ref) {
	t.Helper()
	var output bytes.Buffer
	code := cli.Run(context.Background(), []string{"project", "show", "--selector", "workflows"}, env.service, currentProvenance(), &output)
	var event struct {
		Project cli.ProjectView `json:"project"`
	}
	if err := json.Unmarshal(output.Bytes(), &event); err != nil || code != cli.ExitSuccess {
		t.Fatalf("show: %d %s", code, output.String())
	}
	got := event.Project.ActiveWorkflow
	if got == nil || got.Status != "selected" || got.WorkflowID != ref.WorkflowID || got.Revision != ref.Revision || got.Digest != ref.Digest || got.Source != ref.Source || got.Name == "" {
		t.Fatalf("selection: %+v, want %+v", got, ref)
	}
}

func TestActiveWorkflowShowListSelectedNoneAndNoWrites(t *testing.T) {
	env := workflow303Env(t)
	active303Read(t, env, "none")
	builtin := workflowdefinition.Builtin().Ref("builtin")
	args := workflow303Ref("select", builtin)
	preview := workflow303Run(t, env, "previewed", args...)
	active303Read(t, env, "none")
	workflow303Run(t, env, "applied", workflow303Authority(preview, args)...)
	active303Read(t, env, "selected")
	active303AssertRef(t, env, builtin)
	create := []string{"create", "--from-default", "--workflow", "custom"}
	preview = workflow303Run(t, env, "previewed", create...)
	workflow303Run(t, env, "applied", workflow303Authority(preview, create)...)
	customRef := *preview.Workflow.Reference
	active303AssertRef(t, env, builtin)
	args = workflow303Ref("select", customRef)
	preview = workflow303Run(t, env, "previewed", args...)
	active303AssertRef(t, env, builtin)
	workflow303Run(t, env, "applied", workflow303Authority(preview, args)...)
	active303Read(t, env, "selected")
	active303AssertRef(t, env, customRef)
}
func TestActiveWorkflowUnresolvableCatalogAndMixedListing(t *testing.T) {
	for _, kind := range []string{"missing", "unreadable", "invalid"} {
		t.Run(kind, func(t *testing.T) {
			env := workflow303Env(t)
			create := []string{"create", "--from-default", "--workflow", "custom"}
			preview := workflow303Run(t, env, "previewed", create...)
			workflow303Run(t, env, "applied", workflow303Authority(preview, create)...)
			args := workflow303Ref("select", *preview.Workflow.Reference)
			preview = workflow303Run(t, env, "previewed", args...)
			workflow303Run(t, env, "applied", workflow303Authority(preview, args)...)
			repo := filepath.Join(env.workspace, "main")
			configureProject(t, env.service, "another", "Another", "main="+repo, "none")
			path := filepath.Join(env.root, "workflows", "workflows", "custom", "1.json")
			switch kind {
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "unreadable":
				if err := os.Chmod(path, 0); err != nil {
					t.Fatal(err)
				}
			case "invalid":
				if err := os.WriteFile(path, []byte(`{"unknown":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			active303Read(t, env, "unresolvable")
			var listing bytes.Buffer
			if code := cli.Run(context.Background(), []string{"project", "list"}, env.service, currentProvenance(), &listing); code != cli.ExitSuccess {
				t.Fatalf("list %d", code)
			}
			var got struct {
				Projects []cli.ProjectListView `json:"projects"`
			}
			if err := json.Unmarshal(listing.Bytes(), &got); err != nil || len(got.Projects) != 2 || got.Projects[0].Slug != "another" || got.Projects[0].ActiveWorkflow.Status != "none" || got.Projects[1].Slug != "workflows" {
				t.Fatalf("mixed isolation: %s (%v)", listing.String(), err)
			}
		})
	}
}
func TestWorkflowScratchDraftAuthorityJourney303(t *testing.T) {
	env := workflow303Env(t)
	draft := filepath.Join(t.TempDir(), "workflow.json")
	d := workflowdefinition.Builtin().Definition
	d.WorkflowID = "scratch"
	d.Name = "Scratch workflow"
	write := func() {
		wire, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(draft, wire, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	workflow303Run(t, env, "validated", "validate", "--file", draft)
	create := []string{"create", "--file", draft}
	preview := workflow303Run(t, env, "previewed", create...)
	refused := func(args []string) {
		before := snapshot303Trees(t, env.root, env.state, env.workspace, filepath.Dir(draft))
		workflow303Run(t, env, "stale_authority", args...)
		if !bytes.Equal(before, snapshot303Trees(t, env.root, env.state, env.workspace, filepath.Dir(draft))) {
			t.Fatal("refusal wrote state")
		}
	}
	refused(append(append([]string{}, create...), "--expected-revision", preview.Workflow.ProjectRevision, "--preview-digest", preview.Workflow.PreviewDigest))
	d.Name = "Changed draft"
	write()
	refused(workflow303Authority(preview, create))
	d.Name = "Scratch workflow"
	write()
	workflow303Run(t, env, "applied", workflow303Authority(preview, create)...)
	workflow303Run(t, env, "inspected", workflow303Ref("show", *preview.Workflow.Reference)...)
	active303Read(t, env, "none")
	selectArgs := workflow303Ref("select", *preview.Workflow.Reference)
	selected := workflow303Run(t, env, "previewed", selectArgs...)
	active303Read(t, env, "none")
	refused(append(append([]string{}, selectArgs...), "--expected-revision", selected.Workflow.ProjectRevision, "--preview-digest", selected.Workflow.PreviewDigest))
	workflow303Run(t, env, "applied", workflow303Authority(selected, selectArgs)...)
	active303Read(t, env, "selected")
}

// Snapshot unreadable regular files by metadata as well as readable content.
// This covers refusal/read proofs without changing their permissions to inspect.
func snapshot303Trees(t *testing.T, roots ...string) []byte {
	t.Helper()
	var out bytes.Buffer
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			fmt.Fprintf(&out, "%s %s %d %d %d\n", root, relative, info.Mode(), info.Size(), info.ModTime().UnixNano())
			if info.Mode().IsRegular() {
				data, err := os.ReadFile(path)
				if err != nil {
					if os.IsPermission(err) {
						out.WriteString("permission denied\n")
						return nil
					}
					return err
				}
				out.Write(data)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out.Bytes()
}
