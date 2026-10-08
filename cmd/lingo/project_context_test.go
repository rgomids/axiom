package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
)

func TestProjectContextCLIAndRuntimeSessions(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	source := env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	ctx := context.Background()
	run := func(want int, args ...string) map[string]any {
		t.Helper()
		var out bytes.Buffer
		code := cli.Run(ctx, args, env.service, currentProvenance(), &out)
		if code != want {
			t.Fatalf("args=%v code=%d output=%s", args, code, &out)
		}
		var result map[string]any
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), env.state) || strings.Contains(out.String(), source) {
			t.Fatal("local path disclosed")
		}
		return result
	}
	before, err := os.ReadFile(filepath.Join(source, "axiom.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	run(1, "project", "context", "default-set", "--selector", "external")
	if invalid := run(1, "project", "context", "show", "--unknown"); invalid["status"] != "validation_failure" || invalid["provenance"] == nil {
		t.Fatal(invalid)
	}
	run(0, "project", "context", "default-set", "--selector", "external", "--authorize-local")
	for _, prefix := range [][]string{nil, {"--session", "codex-one"}, {"--session", "claude-two"}} {
		result := run(0, append(prefix, "project", "show")...)
		if result["project"].(map[string]any)["id"] != workItemSourceProjectID {
			t.Fatal(result)
		}
	}
	run(0, "--session", "codex-one", "project", "context", "session-set", "--selector", "external", "--authorize-local")
	view := run(0, "--session", "codex-one", "project", "context", "show")["context"].(map[string]any)
	if view["source"] != "session" || view["default"] != workItemSourceProjectID {
		t.Fatal(view)
	}
	view = run(0, "--session", "claude-two", "project", "context", "show")["context"].(map[string]any)
	if view["source"] != "default" || view["session"] != nil {
		t.Fatal(view)
	}
	run(1, "--session", "codex-one", "project", "show", "--selector", "unknown")
	run(1, "project", "show", "--selector", "")
	run(1, "project", "context", "show", "--selector", "")
	run(1, "--session", "../unsafe", "project", "context", "show")
	run(1, "--session", "one", "--session", "two", "project", "context", "show")
	run(0, "--session", "codex-one", "project", "context", "session-end", "--authorize-local")
	view = run(0, "--session", "codex-one", "project", "context", "show")["context"].(map[string]any)
	if view["source"] != "default" {
		t.Fatal(view)
	}
	// Capability/preflight consumes the resolved Project before a Work Item
	// operation; no Provider effect occurs for an invalid Repository.
	run(1, "work-item", "show", "--repository", "missing", "--number", "7")
	run(0, "project", "context", "default-clear", "--authorize-local")
	run(1, "project", "context", "show")
	run(1, "project", "show")
	after, _ := os.ReadFile(filepath.Join(source, "axiom.yaml"))
	if !bytes.Equal(before, after) {
		t.Fatal("preferences changed portable configuration")
	}
}

func TestProjectContextFailsOnStalePortableRevision(t *testing.T) {
	env := newWorkItemSourceEnvironment(t)
	source := env.installOutsideRoot(t, portableManifest(t, workItemSourceProjectID, "external", "External", true))
	service := env.service.(lifecycleService)
	result := service.ProjectContext(context.Background(), cli.ProjectContextInput{Action: "default-set", Selector: "external", AuthorizeLocal: true})
	if result.Completion.Status() != "success" {
		t.Fatal(result)
	}
	path := filepath.Join(source, "axiom.yaml")
	wire, _ := os.ReadFile(path)
	if err := os.WriteFile(path, bytes.Replace(wire, []byte("name: External"), []byte("name: Changed"), 1), 0600); err != nil {
		t.Fatal(err)
	}
	if id, result := service.EffectiveProject(context.Background(), ""); id != "" || result.Category != "stale_project_context" {
		t.Fatal(id, result)
	}
	before := snapshotTrees(t, env.root, env.state, env.elsewhere)
	if result := service.WorkItemCreate(context.Background(), cli.WorkItemInput{Repository: "main"}); result.Category != "stale_project_context" {
		t.Fatal(result)
	}
	if result := service.WorkflowStart(context.Background(), cli.WorkflowInput{Repository: "main"}); result.Category != "stale_project_context" {
		t.Fatal(result)
	}
	if after := snapshotTrees(t, env.root, env.state, env.elsewhere); !bytes.Equal(before, after) {
		t.Fatal("stale context caused an effect")
	}
}
