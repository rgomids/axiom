package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// Scripted turns, not native conversation or proof of semantic LLM routing.
// Each entry fixture hands the same discovered operation to the real CLI.
func TestOperatorSyntheticDualEntryProjectWorkflowJourney(t *testing.T) {
	traces := map[string][]string{}
	for _, entry := range []string{"natural-intent-script", "explicit-operation-script"} {
		t.Run(entry, func(t *testing.T) {
			env := workflow303Env(t)
			other := writeTree(t, filepath.Join(env.workspace, "other"), map[string]string{"README.md": "synthetic"})
			configureProject(t, env.service, "other", "Other", "main="+other, "none")
			trace := []string{}
			invoke := func(args []string, success bool, read bool) []byte {
				t.Helper()
				before := snapshot303Trees(t, env.root, env.state, env.workspace)
				var out bytes.Buffer
				code := cli.Run(context.Background(), args, env.service, currentProvenance(), &out)
				if (code == cli.ExitSuccess) != success {
					t.Fatalf("%v: %d %s", args, code, &out)
				}
				if read && !bytes.Equal(before, snapshot303Trees(t, env.root, env.state, env.workspace)) {
					t.Fatalf("read/refusal wrote: %v", args)
				}
				trace = append(trace, strings.Join(args[:2], " "))
				return out.Bytes()
			}
			invoke([]string{"skill", "inspect", "axiom-project"}, true, true)
			listed := invoke([]string{"project", "list"}, true, true)
			var list struct {
				Projects []cli.ProjectListView `json:"projects"`
			}
			if json.Unmarshal(listed, &list) != nil || len(list.Projects) != 2 {
				t.Fatal("missing multiple Projects")
			}
			// Ambiguous pronoun is scripted as a question, with no CLI mutation.
			baseline := snapshot303Trees(t, env.root, env.state, env.workspace)
			trace = append(trace, "ask:Project UUID")
			var projectID string
			for _, p := range list.Projects {
				if p.Slug == "workflows" {
					projectID = p.ID
				}
			}
			show := func() cli.ProjectView {
				t.Helper()
				raw := invoke([]string{"project", "show", "--selector", projectID}, true, true)
				var event struct {
					Project cli.ProjectView `json:"project"`
				}
				if json.Unmarshal(raw, &event) != nil || event.Project.ID != projectID {
					t.Fatal("read-back identity mismatch")
				}
				return event.Project
			}
			observed := show()
			if len(observed.Repositories) != 1 || observed.Repositories[0].Key != "main" || !bytes.Equal(baseline, snapshot303Trees(t, env.root, env.state, env.workspace)) {
				t.Fatal("discovery changed associations")
			}
			// Add main collides with an observed key; do not invoke valid update/upsert.
			trace = append(trace, "collision:main; ask:alternative key")
			if !bytes.Equal(baseline, snapshot303Trees(t, env.root, env.state, env.workspace)) {
				t.Fatal("collision changed state")
			}
			docs := writeTree(t, filepath.Join(env.workspace, "docs"), map[string]string{"README.md": "human-chosen docs"})
			edit := []string{"project", "configure", "--project", projectID, "--repository", "docs=" + docs}
			var preview editEvent
			if json.Unmarshal(invoke(edit, true, true), &preview) != nil {
				t.Fatal("preview")
			}
			// A new intent/different draft cannot use authority over the old preview.
			changed := append(append([]string{}, edit...), "--name", "Changed without reviewed preview")
			changed = append(changed, "--project-id", preview.Edit.ProjectID, "--preview-digest", preview.Edit.Digest, "--authorize-local")
			invoke(changed, false, true)
			invoke(append(append([]string{}, edit...), "--project-id", preview.Edit.ProjectID, "--preview-digest", preview.Edit.Digest, "--authorize-local"), true, false)
			observed = show()
			if len(observed.Repositories) != 2 {
				t.Fatal("apply not confirmed by canonical read-back")
			}
			active303Read(t, env, "none")
			builtin := workflowdefinition.Builtin().Ref("builtin")
			selection := workflow303Ref("select", builtin)
			selected := workflow303Run(t, env, "previewed", selection...)
			workflow303Run(t, env, "applied", workflow303Authority(selected, selection)...)
			active303AssertRef(t, env, builtin)
			definition := workflow303Run(t, env, "inspected", workflow303Ref("show", builtin)...)
			beforeBuiltin, _ := json.Marshal(definition.Workflow.Definition)
			draft := filepath.Join(t.TempDir(), "draft.json")
			wire, _ := json.Marshal(definition.Workflow.Definition)
			if err := os.WriteFile(draft, wire, 0600); err != nil {
				t.Fatal(err)
			}
			// Built-in direct edit refuses, without touching canonical state.
			before := snapshot303Trees(t, env.root, env.state, env.workspace)
			workflow303Run(t, env, "prior_revision_required", "edit", "--workflow", builtin.WorkflowID, "--prior-revision", strconv.Itoa(builtin.Revision), "--prior-digest", builtin.Digest, "--file", draft)
			if !bytes.Equal(before, snapshot303Trees(t, env.root, env.state, env.workspace)) {
				t.Fatal("built-in refusal wrote")
			}
			copyArgs := []string{"create", "--from-default", "--workflow", "operator-copy"}
			copyPreview := workflow303Run(t, env, "previewed", copyArgs...)
			workflow303Run(t, env, "applied", workflow303Authority(copyPreview, copyArgs)...)
			active303AssertRef(t, env, builtin)
			// Adding the same key never invents a suffix or changes selection.
			before = snapshot303Trees(t, env.root, env.state, env.workspace)
			trace = append(trace, "collision:operator-copy; no create invocation; ask:key")
			if !bytes.Equal(before, snapshot303Trees(t, env.root, env.state, env.workspace)) {
				t.Fatal("duplicate key wrote")
			}
			current := workflow303Run(t, env, "inspected", workflow303Ref("show", builtin)...)
			afterBuiltin, _ := json.Marshal(current.Workflow.Definition)
			if !bytes.Equal(beforeBuiltin, afterBuiltin) {
				t.Fatal("built-in changed")
			}
			copyRef := *copyPreview.Workflow.Reference
			readCopy := workflow303Run(t, env, "inspected", workflow303Ref("show", copyRef)...)
			if *readCopy.Workflow.Reference != copyRef {
				t.Fatal("copy read-back diverged")
			}
			// Host without graphics: a scripted auxiliary list uses the same observed
			// definition; no second operation is executed to obtain presentation.
			fallback := []string{}
			for _, stage := range current.Workflow.Definition.Stages {
				fallback = append(fallback, stage.ID)
			}
			if len(fallback) != len(current.Workflow.Definition.Stages) || fallback[0] != current.Workflow.Definition.Stages[0].ID {
				t.Fatal("fallback invented order")
			}
			var human bytes.Buffer
			invoke([]string{"skill", "inspect", "axiom-workflow"}, true, true)
			// Real CLI refusal remains visible in both canonical views.
			if code := cli.RunInteractive(context.Background(), []string{"project", "show", "--selector", "missing"}, env.service, currentProvenance(), nil, &human, io.Discard); code == 0 || !strings.Contains(human.String(), "Validation failed") {
				t.Fatal("human refusal hidden")
			}
			trace = append(trace, "workflow:copy separate from selection; default preserved")
			traces[entry] = trace
			t.Logf("classification=synthetic entry=%s turns=%d negative_effects=0", entry, len(trace))
		})
	}
	if !reflect.DeepEqual(traces["natural-intent-script"], traces["explicit-operation-script"]) {
		t.Fatalf("entry traces diverged: %+v", traces)
	}
}

func TestOperatorSyntheticDivergentReadBackStopsWithoutReapply(t *testing.T) {
	env := newEditEnvironment(t)
	intended := writeTree(t, filepath.Join(t.TempDir(), "intended"), map[string]string{"README.md": "synthetic"})
	concurrent := writeTree(t, filepath.Join(t.TempDir(), "concurrent"), map[string]string{"README.md": "synthetic"})
	env.publish(t, "--project", env.projectID, "--repository", "docs="+intended)
	// Another explicitly scripted authorized caller changes the same association
	// before our read-back. This is a canonical mutation, not a direct store edit.
	env.publish(t, "--project", env.projectID, "--repository", "docs="+concurrent)
	baseline := snapshotTrees(t, env.root, env.state)
	var out bytes.Buffer
	if code := cli.Run(context.Background(), []string{"project", "show", "--selector", env.projectID}, env.service, currentProvenance(), &out); code != 0 {
		t.Fatal("read-back failed")
	}
	var event struct {
		Project cli.ProjectView `json:"project"`
	}
	if json.Unmarshal(out.Bytes(), &event) != nil {
		t.Fatal("invalid read-back")
	}
	var actual string
	for _, repo := range event.Project.Repositories {
		if repo.Key == "docs" {
			actual = repo.Path
		}
	}
	if actual != concurrent || actual == intended {
		t.Fatal("canonical read-back did not reveal drift")
	}
	// Programmed operator stops here; it never reapplies the prior approved edit.
	if !bytes.Equal(baseline, snapshotTrees(t, env.root, env.state)) {
		t.Fatal("divergent read-back replayed an effect")
	}
	t.Log("classification=synthetic verification=divergent next=handoff reapply_calls=0")
}
