package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// Specification 013: the complete guided Project setup journey reproduced for
// the synthetic Project govsm with Codex and Claude, through the production
// binary, isolated roots and synthetic executables that are never invoked. No
// step edits a persisted file, authenticates or runs inference.

type setupEnv struct {
	t                                   *testing.T
	binary, base, state, projects, home string
}

func newSetupEnv(t *testing.T, installed ...string) setupEnv {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "axiom")
	if b, e := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); e != nil {
		t.Fatalf("build: %v %s", e, b)
	}
	base := t.TempDir()
	bin := filepath.Join(base, "bin")
	if e := os.Mkdir(bin, 0o700); e != nil {
		t.Fatal(e)
	}
	for _, id := range installed {
		// Any invocation would fail the journey loudly.
		if e := os.WriteFile(filepath.Join(bin, id), []byte("#!/bin/sh\necho invoked >&2\nexit 97\n"), 0o700); e != nil {
			t.Fatal(e)
		}
	}
	return setupEnv{t, binary, base, filepath.Join(base, "state"), filepath.Join(base, "projects"), filepath.Join(base, "home")}
}

func (e setupEnv) exec(json bool, args ...string) ([]byte, int) {
	if json {
		args = append([]string{"--json"}, args...)
	}
	cmd := exec.Command(e.binary, args...)
	cmd.Env = []string{"PATH=" + filepath.Join(e.base, "bin"), "HOME=" + e.home, "LINGO_STATE_ROOT=" + e.state, "LINGO_PROJECTS_ROOT=" + e.projects, "AXIOM_CODEX_SKILLS_ROOT=" + filepath.Join(e.base, "skills")}
	out, err := cmd.Output()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	}
	if bytes.Contains(out, []byte("invoked")) {
		e.t.Fatal("vendor executable invoked")
	}
	return out, code
}

func (e setupEnv) run(category string, args ...string) map[string]any {
	e.t.Helper()
	out, _ := e.exec(true, args...)
	var event map[string]any
	if err := json.Unmarshal(out, &event); err != nil {
		e.t.Fatalf("%v: %v %s", args, err, out)
	}
	if category != "" && text(event, "category") != category {
		e.t.Fatalf("%v: want %s got %s", args, category, out)
	}
	return event
}

func (e setupEnv) authorizeRuntime(args ...string) map[string]any {
	e.t.Helper()
	preview := e.run("preview", args...)
	auth := append(append([]string{}, args...), "--expected-revision", text(preview, "runtimeAuthoring", "expectedRevision"), "--preview-digest", text(preview, "runtimeAuthoring", "previewDigest"), "--authorize-local")
	applied := e.run("applied", auth...)
	// Exact authority is single-use: a replay is refused without effects.
	e.run("stale_preview", auth...)
	return applied
}

func (e setupEnv) configureProject(args ...string) map[string]any {
	e.t.Helper()
	preview := e.run("", args...)
	mode := "setup"
	if field(preview, "edit") != nil {
		mode = "edit"
	}
	digest, id := text(preview, mode, "digest"), text(preview, mode, "projectId")
	if digest == "" || id == "" {
		e.t.Fatalf("%v: no reviewable preview %v", args, preview)
	}
	// A wrong digest is refused and publishes nothing.
	before := treeDigest(e.t, e.projects)
	if _, code := e.exec(true, append(args, "--project-id", id, "--preview-digest", strings.Repeat("0", 64), "--authorize-local")...); code == 0 || treeDigest(e.t, e.projects) != before {
		e.t.Fatal("stale project preview applied")
	}
	return e.run("", append(args, "--project-id", id, "--preview-digest", digest, "--authorize-local")...)
}

func treeDigest(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			wire, _ := os.ReadFile(p)
			b.WriteString(p + "\x00" + string(wire) + "\x00")
		}
		return nil
	})
	return b.String()
}

func diagnosticFields(event map[string]any) []string {
	out := []string{}
	list, _ := field(event, "runtimeAuthoring", "diagnostics").([]any)
	for _, d := range list {
		m, _ := d.(map[string]any)
		out = append(out, text(m, "code")+":"+text(m, "field"))
	}
	return out
}

func TestGuidedSetupGovsmFromScratchWithCodexAndClaude(t *testing.T) {
	e := newSetupEnv(t, "codex", "claude")
	repoA := writeTree(t, filepath.Join(e.base, "work", "govsm-api"), map[string]string{"README.md": "api"})
	repoB := writeTree(t, filepath.Join(e.base, "work", "govsm-web"), map[string]string{"README.md": "web"})

	// Step 0: discovery is passive and writes nothing.
	if listed := e.run("inspected", "runtime", "binding", "list"); text(listed, "runtimeAuthoring", "state") != "fresh" {
		t.Fatal(listed)
	}
	if _, err := os.Stat(e.state); !os.IsNotExist(err) {
		t.Fatal("discovery wrote state")
	}

	// Step 1: the original v0.22.0 command names the missing field, in JSON
	// and human output alike, and never points at authentication.
	original := []string{"runtime", "binding", "create", "--runtime", "codex", "--profile", "codex-default", "--model", "gpt-6-luna", "--effort-mode", "explicit", "--effort", "low"}
	refused := e.run("candidate_invalid", original...)
	if got := diagnosticFields(refused); len(got) != 1 || got[0] != "field_required:--capabilities" {
		t.Fatal(got)
	}
	if !strings.Contains(text(refused, "next"), "--capabilities <comma-list>") || !strings.Contains(text(refused, "next"), "authentication was not evaluated") {
		t.Fatal(text(refused, "next"))
	}
	human, code := e.exec(false, original...)
	if code == 0 || !bytes.Contains(human, []byte("field_required")) || !bytes.Contains(human, []byte("--capabilities <comma-list>")) {
		t.Fatalf("human output lacks parity: %s", human)
	}
	for _, out := range [][]byte{human, []byte(mustJSON(t, refused))} {
		for _, forbidden := range []string{"OPENAI_API_KEY", "ANTHROPIC_BASE_URL", "CLAUDE_CODE"} {
			if bytes.Contains(out, []byte(forbidden)) {
				t.Fatal("auth remediation for a declaration refusal", forbidden)
			}
		}
	}
	if _, err := os.Stat(e.state); !os.IsNotExist(err) {
		t.Fatal("refusal wrote state")
	}

	// A Profile without its binding is binding_missing, not model evidence.
	missing := e.run("binding_missing", "runtime", "profile", "create", "--runtime", "claude", "--profile", "claude-default", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "read")
	if text(missing, "runtimeAuthoring", "modelSupport") != "unconfirmed" || diagnosticFields(missing)[0] != "binding_missing:--runtime" {
		t.Fatal(missing)
	}

	// Supplying the field is enough: validate-candidate, preview, exact apply.
	fixed := append(append([]string{}, original...), "--capabilities", "read,repository-write")
	candidate := e.run("candidate_valid", append([]string{"runtime", "binding", "validate-candidate"}, fixed[3:]...)...)
	if !strings.Contains(text(candidate, "next"), "not vendor certification") {
		t.Fatal(candidate)
	}
	e.authorizeRuntime(fixed...)
	e.authorizeRuntime("runtime", "binding", "create", "--runtime", "claude", "--profile", "claude-default", "--model", "operator-claude-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "read,repository-write")
	profiles := e.run("inspected", "runtime", "profile", "list")
	if got := mustJSON(t, field(profiles, "runtimeAuthoring", "profiles")); !strings.Contains(got, `"id":"codex-default"`) || !strings.Contains(got, `"model":"gpt-6-luna"`) || !strings.Contains(got, `"id":"claude-default"`) || text(profiles, "runtimeAuthoring", "modelSupport") != "unconfirmed" {
		t.Fatal("read-back", got)
	}
	// Reuse is detected, never duplicated.
	e.run("binding_exists", fixed...)

	// Steps 2-4: identity, Repositories, Provider, context and portable policy.
	create := []string{"project", "configure", "--slug", "govsm", "--name", "GovSM",
		"--repository", "api=" + repoA, "--repository-remote", "api=none",
		"--repository", "web=" + repoB, "--repository-remote", "web=none",
		"--work-item-provider", "github",
		"--technology", "language=go", "--business-context", "Synthetic governance service",
		"--glossary", "sm=SM:service management",
		"--runtime", "codex", "--runtime", "claude", "--model-profile", "codex-default", "--model-profile", "claude-default",
		"--runtime-preference", "implementation/high=codex-default"}
	e.configureProject(create...)
	shown := e.run("", "project", "show", "--selector", "govsm")
	if got := mustJSON(t, shown["project"]); !strings.Contains(got, `"key":"api"`) || !strings.Contains(got, `"key":"web"`) {
		t.Fatal("repository read-back", got)
	}

	// Step 5: the built-in default is offered explicitly, then selected under
	// its own authority; a stale digest is refused first.
	listed := e.run("", "project", "workflow", "list", "--project", "govsm")
	builtin, _ := workflowdefinition.BuiltinRevision(2)
	ref := builtin.Ref("builtin")
	if !strings.Contains(mustJSON(t, listed), ref.WorkflowID) {
		t.Fatal("default workflow not offered", listed)
	}
	selectArgs := []string{"project", "workflow", "select", "--project", "govsm", "--workflow", ref.WorkflowID, "--revision", "2", "--digest", ref.Digest, "--source", ref.Source}
	preview := e.run("", selectArgs...)
	if _, code := e.exec(true, append(selectArgs, "--expected-revision", text(preview, "workflowAuthoring", "projectRevision"), "--preview-digest", strings.Repeat("0", 64), "--authorize-local")...); code == 0 {
		t.Fatal("stale workflow preview applied")
	}
	e.run("", append(selectArgs, "--expected-revision", text(preview, "workflowAuthoring", "projectRevision"), "--preview-digest", text(preview, "workflowAuthoring", "previewDigest"), "--authorize-local")...)
	if got := mustJSON(t, field(e.run("", "project", "show", "--selector", "govsm"), "project")); !strings.Contains(got, ref.WorkflowID) {
		t.Fatal("selection read-back", got)
	}

	// Step 6: final validation keeps structure, readiness and per-operation
	// results separate; authentication is not observed by this journey.
	validated := e.run("", "project", "validate", "--project", "govsm")
	readiness := mustJSON(t, validated["readiness"])
	if text(validated, "readiness", "structure") != "valid" || text(validated, "readiness", "runtime", "status") != "ready" ||
		!strings.Contains(readiness, `{"blockers":[],"operation":"execution","status":"ready"}`) || !strings.Contains(readiness, `{"blockers":[],"operation":"work-item","status":"ready"}`) {
		t.Fatal(readiness)
	}
	if strings.Contains(mustJSON(t, validated), "runtime_policy_unavailable") {
		t.Fatal("policy reported missing after configuration", validated["readiness"])
	}
}

func TestGuidedSetupCompletesExistingIncompleteProject(t *testing.T) {
	e := newSetupEnv(t, "codex")
	repo := writeTree(t, filepath.Join(e.base, "work", "core"), map[string]string{"README.md": "core"})
	// An existing Project published before any Runtime/Profile existed.
	e.configureProject("project", "configure", "--slug", "partial", "--name", "Partial", "--repository", "core="+repo, "--repository-remote", "core=none", "--work-item-provider", "none")
	before := e.run("", "project", "validate", "--project", "partial")
	if !strings.Contains(mustJSON(t, before["readiness"]), "runtime_policy_unavailable") {
		t.Fatal("missing policy not reported", before["readiness"])
	}
	// An absent Runtime is an environment gap, not an authentication one.
	absent := e.run("runtime_absent", "runtime", "binding", "create", "--runtime", "claude", "--profile", "claude-default", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "read")
	if got := diagnosticFields(absent); len(got) != 1 || got[0] != "runtime_absent:--runtime" || !strings.Contains(mustJSON(t, absent), "not authentication") {
		t.Fatal(got, absent)
	}
	// Local prerequisite first, then the portable policy through G-1.
	e.authorizeRuntime("runtime", "binding", "create", "--runtime", "codex", "--profile", "worker", "--model", "operator-model", "--effort-mode", "explicit", "--effort", "low", "--capabilities", "read,repository-write")
	draft := `{"runtimes":{"mode":"replace","values":[{"id":"codex"}]},"modelProfiles":{"mode":"replace","values":[{"key":"worker","runtimeRef":"codex","model":"operator-model"}]}}`
	policy := []string{"project", "configure", "--project", "partial", "--policy-draft", draft}
	preview := e.run("", policy...)
	id, digest := text(preview, "policyEdit", "preview", "projectId"), text(preview, "policyEdit", "preview", "digest")
	if digest == "" {
		t.Fatal("no policy preview", preview)
	}
	if id == "" {
		id = text(e.run("", "project", "show", "--selector", "partial"), "project", "id")
	}
	// Reuse: an existing local Profile is listed, not recreated.
	if got := mustJSON(t, field(e.run("inspected", "runtime", "profile", "list"), "runtimeAuthoring", "profiles")); !strings.Contains(got, `"id":"worker"`) {
		t.Fatal(got)
	}
	e.run("profile_exists", "runtime", "profile", "create", "--runtime", "codex", "--profile", "worker", "--model", "operator-model", "--effort-mode", "explicit", "--effort", "low", "--capabilities", "read")
	applied := e.run("", append(policy, "--project-id", id, "--preview-digest", digest, "--authorize-local")...)
	if text(applied, "policyEdit", "readBack", "status") != "confirmed" {
		t.Fatal("policy read-back", applied["policyEdit"])
	}
	after := e.run("", "project", "validate", "--project", "partial")
	// The policy gap is resolved. A schema-3 Project keeps metadata-only
	// readiness: Runtime resolution stays explicitly unobserved, never ready,
	// and the Work Item gap stays a separate blocker.
	if got := mustJSON(t, after["readiness"]); strings.Contains(got, "runtime_policy_unavailable") || text(after, "readiness", "runtime", "detail") != "inventory_unavailable" || text(after, "readiness", "effective") == "ready" || !strings.Contains(got, "capability_mapping_missing") {
		t.Fatal("readiness after policy", got)
	}
}
