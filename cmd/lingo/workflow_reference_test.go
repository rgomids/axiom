package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// stateFingerprint fingerprints every entry below root (path, mode, bytes or link
// target) so a refusal can be proven side-effect free.
func stateFingerprint(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable directory is part of the fingerprint, not a failure.
			fmt.Fprintf(h, "%s|unreadable\x00", path)
			return fs.SkipDir
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		fmt.Fprintf(h, "%s|%v|", relative, info.Mode())
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, _ := os.Readlink(path)
			fmt.Fprintf(h, "%s", target)
		case info.Mode().IsRegular():
			wire, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			h.Write(wire)
		}
		h.Write([]byte{0})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// TestWorkflowRemoveWithCanonicalExecutions is the IF-01 regression (PR #366
// review 5732338943): canonical Executions under executions/v2 and their
// attempt outputs must not make every retirement fail as an unknown
// inventory, while the strict ExecutionStore.List scan stays the reference
// authority and any doubt still refuses without effects.
func TestWorkflowRemoveWithCanonicalExecutions(t *testing.T) {
	env := newBootstrapEnv(t)
	repository := writeTree(t, filepath.Join(env.workspace, "main"), map[string]string{"README.md": "untouched"})
	projectID := configureProject(t, env.service, "refs", "Refs", "main="+repository, "none").ProjectID
	run := func(category string, args ...string) authoringEvent {
		t.Helper()
		var out bytes.Buffer
		code := cli.Run(context.Background(), append([]string{"project", "workflow"}, append(args, "--project", "refs")...), env.service, currentProvenance(), &out)
		var event authoringEvent
		if json.Unmarshal(out.Bytes(), &event) != nil || event.Category != category {
			t.Fatalf("%v => %s (exit %d); want %s", args, out.String(), code, category)
		}
		success := category == "previewed" || category == "applied"
		if (code == cli.ExitSuccess) != success {
			t.Fatalf("wrong exit %d for %s: %s", code, category, out.String())
		}
		t.Logf("if01-evidence %v => category=%s exit=%d", args[:1], event.Category, code)
		return event
	}
	authorize := func(preview authoringEvent, args []string) []string {
		return append(append([]string{}, args...), "--expected-revision", preview.Workflow.ProjectRevision, "--preview-digest", preview.Workflow.PreviewDigest, "--authorize-local")
	}
	remove := func(ref workflowdefinition.Ref) []string {
		return []string{"remove", "--workflow", ref.WorkflowID, "--revision", strconv.Itoa(ref.Revision), "--digest", ref.Digest, "--source", ref.Source}
	}
	create := func(id string) workflowdefinition.Ref {
		args := []string{"create", "--from-default", "--workflow", id}
		preview := run("previewed", args...)
		run("applied", authorize(preview, args)...)
		return *preview.Workflow.Reference
	}
	used, unused, spare := create("custom"), create("other"), create("spare")

	// (iv) Blank state: no Execution namespace exists; retirement previews.
	if _, err := os.Stat(filepath.Join(env.state, "executions")); !os.IsNotExist(err) {
		t.Fatalf("fixture already has executions: %v", err)
	}
	run("previewed", remove(spare)...)

	// Real canonical Executions on disk through the owning store, plus the
	// attempt-output namespace a launched attempt creates.
	store, err := local.NewExecutionStore(env.state)
	if err != nil {
		t.Fatal(err)
	}
	execution := func(n int, workflowID string, revision int) {
		t.Helper()
		x, err := workflow.NewExecution(workflow.NewExecutionInput{
			ExecutionID: fmt.Sprintf("00000000-0000-4000-8000-%012d", n), ProjectID: projectID,
			WorkItem:     workflow.WorkItemLink{Provider: "local", ID: "IF-01", RepositoryKey: "main"},
			Repositories: []workflow.RepositoryBinding{{Key: "main", Identity: "project-repository:main"}},
			WorkflowID:   workflowID, Revision: "r" + strconv.Itoa(revision), Now: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
			Stages: []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder"}}, TimeoutSeconds: 60}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Create(context.Background(), x); err != nil {
			t.Fatal(err)
		}
	}
	// (v) Many Executions; the only one referencing used/r1 sorts last.
	for n := 1; n <= 24; n++ {
		execution(n, used.WorkflowID, 9)
	}
	execution(99, used.WorkflowID, used.Revision)
	outputs := filepath.Join(env.state, "executions", "v2-attempt-outputs", projectID, "00000000-0000-4000-8000-000000000099")
	if err := os.MkdirAll(outputs, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputs, "result.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory, err := local.InspectStateInventory(context.Background(), env.state)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range inventory.Entries {
		t.Logf("if01-inventory %s %s", entry.Relative, entry.Kind)
	}

	// (ii) A referenced revision refuses with revision_in_use, effect free,
	// both at preview and when authorization is presented.
	before := stateFingerprint(t, env.state) + stateFingerprint(t, env.root)
	run("revision_in_use", remove(used)...)
	stale := run("previewed", remove(spare)...)
	run("revision_in_use", authorize(stale, remove(used))...)
	if after := stateFingerprint(t, env.state) + stateFingerprint(t, env.root); after != before {
		t.Fatal("referenced-revision refusal changed state")
	}

	// (i) An unreferenced revision is retired despite canonical Executions.
	preview := run("previewed", remove(unused)...)
	run("applied", authorize(preview, remove(unused))...)

	// (iii) Any doubt about the canonical namespace refuses with the bounded
	// reference_inventory_unknown category and no effect.
	projectDir := filepath.Join(env.state, "executions", "v2", projectID)
	record := filepath.Join(projectDir, "00000000-0000-4000-8000-000000000099.json")
	refuse := func(name string, damage, restore func()) {
		t.Helper()
		damage()
		before := stateFingerprint(t, env.state) + stateFingerprint(t, env.root)
		run("reference_inventory_unknown", remove(spare)...)
		if after := stateFingerprint(t, env.state) + stateFingerprint(t, env.root); after != before {
			t.Fatalf("%s: refusal changed state", name)
		}
		restore()
		t.Logf("if01-evidence fail-closed %s", name)
	}
	write := func(path string, wire []byte) func() {
		return func() {
			if err := os.WriteFile(path, wire, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	unlink := func(path string) func() {
		return func() {
			if err := os.RemoveAll(path); err != nil {
				t.Fatal(err)
			}
		}
	}
	original, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	refuse("corrupt referenced record", write(record, []byte(`{"formatVersion":4,`)), write(record, original))
	refuse("newer record format", write(record, []byte(`{"formatVersion":99}`)), write(record, original))
	refuse("unknown entry in project namespace", write(filepath.Join(projectDir, "junk.txt"), []byte("x")), unlink(filepath.Join(projectDir, "junk.txt")))
	refuse("pending protocol state", write(filepath.Join(projectDir, ".axiom-stage-x"), []byte("x")), unlink(filepath.Join(projectDir, ".axiom-stage-x")))
	refuse("unsupported newer namespace", func() { os.MkdirAll(filepath.Join(env.state, "executions", "v3"), 0o700) }, unlink(filepath.Join(env.state, "executions", "v3")))
	refuse("unknown executions sibling", func() { os.MkdirAll(filepath.Join(env.state, "executions", "v2-other"), 0o700) }, unlink(filepath.Join(env.state, "executions", "v2-other")))
	refuse("unreadable project namespace", func() { os.Chmod(projectDir, 0o000) }, func() { os.Chmod(projectDir, 0o700) })
	v2 := filepath.Join(env.state, "executions", "v2")
	// Outside the state root, so only the symlink itself can cause refusal.
	moved := filepath.Join(t.TempDir(), "moved")
	refuse("symlinked v2 namespace", func() {
		if err := os.Rename(v2, moved); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(moved, v2); err != nil {
			t.Fatal(err)
		}
	}, func() {
		os.Remove(v2)
		if err := os.Rename(moved, v2); err != nil {
			t.Fatal(err)
		}
	})
	outputsRoot := filepath.Join(env.state, "executions", "v2-attempt-outputs")
	refuse("symlinked attempt-output namespace", func() {
		if err := os.Rename(outputsRoot, moved); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(moved, outputsRoot); err != nil {
			t.Fatal(err)
		}
	}, func() {
		os.Remove(outputsRoot)
		if err := os.Rename(moved, outputsRoot); err != nil {
			t.Fatal(err)
		}
	})
	// Restored state is valid again; the referenced revision is still held.
	run("previewed", remove(spare)...)
	run("revision_in_use", remove(used)...)
}
