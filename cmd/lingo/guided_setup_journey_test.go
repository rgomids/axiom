package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
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

// workflowNeeds derives Profile requirements only from a definition read
// through the public `project workflow show` result: the union of Agent
// capabilities, each Agent role/complexity, and the Runtime constraints.
func workflowNeeds(t *testing.T, shown map[string]any) (capabilities, complexities, roles, runtimes []string) {
	t.Helper()
	seen := map[string]bool{}
	add := func(list *[]string, kind, v string) {
		if v != "" && !seen[kind+v] {
			seen[kind+v] = true
			*list = append(*list, v)
		}
	}
	stages, _ := field(shown, "workflowAuthoring", "definition", "stages").([]any)
	for _, s := range stages {
		agents, _ := s.(map[string]any)["agents"].([]any)
		for _, a := range agents {
			agent := a.(map[string]any)
			for _, c := range agent["capabilities"].([]any) {
				add(&capabilities, "c", c.(string))
			}
			for _, r := range agent["runtimeConstraints"].([]any) {
				add(&runtimes, "r", r.(string))
			}
			add(&complexities, "x", text(agent, "complexity"))
			add(&roles, "p", text(agent, "role")+"/"+text(agent, "complexity"))
		}
	}
	sort.Strings(capabilities)
	if len(capabilities) == 0 || len(roles) == 0 {
		t.Fatal("definition exposes no Agent requirements", shown)
	}
	return
}

// CR-356-01: a machine with no Project completes the journey through public
// operations only. Identity is published first (partial bootstrap), so the
// explicitly chosen Workflow can be inspected and selected; Profiles are then
// derived from its Agents and the portable policy reconciles both.
func TestGuidedSetupGovsmFromScratchWithCodexAndClaude(t *testing.T) {
	e := newSetupEnv(t, "codex", "claude")
	repoA := writeTree(t, filepath.Join(e.base, "work", "govsm-api"), map[string]string{"README.md": "api"})
	repoB := writeTree(t, filepath.Join(e.base, "work", "govsm-web"), map[string]string{"README.md": "web"})

	// Step 1: discovery is passive and writes nothing; no Project exists.
	if listed := e.run("inspected", "runtime", "binding", "list"); text(listed, "runtimeAuthoring", "state") != "fresh" {
		t.Fatal(listed)
	}
	if projects, _ := e.run("", "project", "list")["projects"].([]any); len(projects) != 0 {
		t.Fatal("environment is not empty", projects)
	}
	if _, err := os.Stat(e.state); !os.IsNotExist(err) {
		t.Fatal("discovery wrote state")
	}

	// The original v0.22.0 command names the missing field, in JSON and human
	// output alike, and never points at authentication.
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
	// A Profile without its binding is binding_missing, not model evidence.
	missing := e.run("binding_missing", "runtime", "profile", "create", "--runtime", "claude", "--profile", "claude-default", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "read")
	if text(missing, "runtimeAuthoring", "modelSupport") != "unconfirmed" || diagnosticFields(missing)[0] != "binding_missing:--runtime" {
		t.Fatal(missing)
	}
	if _, err := os.Stat(e.state); !os.IsNotExist(err) {
		t.Fatal("refusal wrote state")
	}

	// Step 2: identity, Repositories, Provider and context, without policy.
	e.configureProject("project", "configure", "--slug", "govsm", "--name", "GovSM",
		"--repository", "api="+repoA, "--repository-remote", "api=none",
		"--repository", "web="+repoB, "--repository-remote", "web=none",
		"--work-item-provider", "github",
		"--technology", "language=go", "--business-context", "Synthetic governance service",
		"--glossary", "sm=SM:service management")
	shown := e.run("", "project", "show", "--selector", "govsm")
	if got := mustJSON(t, shown["project"]); !strings.Contains(got, `"key":"api"`) || !strings.Contains(got, `"key":"web"`) {
		t.Fatal("repository read-back", got)
	}
	if !strings.Contains(mustJSON(t, e.run("", "project", "validate", "--project", "govsm")), "runtime_policy_unavailable") {
		t.Fatal("bootstrap Project claimed Runtime readiness")
	}

	// Step 3: the operator chooses one listed definition explicitly; the
	// latest built-in revision stands in for that choice. Its requirements
	// are read through the public show result, never from internal packages.
	// Inspection is read-only; selection waits for step 6.
	listed, _ := field(e.run("", "project", "workflow", "list", "--project", "govsm"), "workflowAuthoring", "revisions").([]any)
	var chosen map[string]any
	for _, r := range listed {
		ref := r.(map[string]any)
		if text(ref, "source") == "builtin" && text(ref, "workflowId") == "default-sdd" && (chosen == nil || ref["revision"].(float64) > chosen["revision"].(float64)) {
			chosen = ref
		}
	}
	if chosen == nil {
		t.Fatal("default-sdd not offered", listed)
	}
	revision := strconv.Itoa(int(chosen["revision"].(float64)))
	ref := []string{"--project", "govsm", "--workflow", text(chosen, "workflowId"), "--revision", revision, "--digest", text(chosen, "digest"), "--source", text(chosen, "source")}
	capabilities, complexities, roles, runtimes := workflowNeeds(t, e.run("", append([]string{"project", "workflow", "show"}, ref...)...))
	if strings.Join(runtimes, ",") != "codex,claude" {
		t.Fatal("definition does not admit both Runtimes", runtimes)
	}
	// Step 4: bindings and Profiles carry the derived capabilities.
	derived := []string{"--capabilities", strings.Join(capabilities, ","), "--complexities", strings.Join(complexities, ",")}
	fixed := append(append([]string{}, original...), derived...)
	candidate := e.run("candidate_valid", append([]string{"runtime", "binding", "validate-candidate"}, fixed[3:]...)...)
	if !strings.Contains(text(candidate, "next"), "not vendor certification") {
		t.Fatal(candidate)
	}
	e.authorizeRuntime(fixed...)
	e.authorizeRuntime(append([]string{"runtime", "binding", "create", "--runtime", "claude", "--profile", "claude-default", "--model", "operator-claude-model", "--effort-mode", "runtime-default", "--effort", "default"}, derived...)...)
	profiles := e.run("inspected", "runtime", "profile", "list")
	if got := mustJSON(t, field(profiles, "runtimeAuthoring", "profiles")); !strings.Contains(got, `"id":"codex-default"`) || !strings.Contains(got, `"model":"gpt-6-luna"`) || !strings.Contains(got, `"id":"claude-default"`) || text(profiles, "runtimeAuthoring", "modelSupport") != "unconfirmed" {
		t.Fatal("read-back", got)
	}
	e.run("binding_exists", fixed...)

	// Step 5: portable policy reconciles the Project with the local Profiles;
	// one preference per Agent role/complexity resolves policy-default.
	preferences := []string{}
	for _, rc := range roles {
		pair := strings.SplitN(rc, "/", 2)
		preferences = append(preferences, `{"role":"`+pair[0]+`","complexity":"`+pair[1]+`","modelProfileRef":"codex-default"}`)
	}
	draft := `{"runtimes":{"mode":"replace","values":[{"id":"codex"},{"id":"claude"}]},"modelProfiles":{"mode":"replace","values":[{"key":"codex-default","runtimeRef":"codex","model":"gpt-6-luna"},{"key":"claude-default","runtimeRef":"claude","model":"operator-claude-model"}]},"runtimePreferences":{"mode":"replace","values":[` + strings.Join(preferences, ",") + `]}}`
	applyPolicy(t, e, "govsm", draft)

	// Step 6: selection last; policy editing is refused once a Workflow is
	// selected, so the explicit choice from step 3 is applied now.
	selectArgs := append([]string{"project", "workflow", "select"}, ref...)
	preview := e.run("", selectArgs...)
	if _, code := e.exec(true, append(selectArgs, "--expected-revision", text(preview, "workflowAuthoring", "projectRevision"), "--preview-digest", strings.Repeat("0", 64), "--authorize-local")...); code == 0 {
		t.Fatal("stale workflow preview applied")
	}
	e.run("", append(selectArgs, "--expected-revision", text(preview, "workflowAuthoring", "projectRevision"), "--preview-digest", text(preview, "workflowAuthoring", "previewDigest"), "--authorize-local")...)
	if got := mustJSON(t, field(e.run("", "project", "show", "--selector", "govsm"), "project")); !strings.Contains(got, text(chosen, "digest")) {
		t.Fatal("selection read-back", got)
	}

	// Step 7: final validation keeps structure, readiness and per-operation
	// results separate; authentication is not observed by this journey.
	validated := e.run("", "project", "validate", "--project", "govsm")
	readiness := mustJSON(t, validated["readiness"])
	if text(validated, "readiness", "structure") != "valid" || text(validated, "readiness", "runtime", "status") != "ready" ||
		!strings.Contains(readiness, `{"blockers":[],"operation":"execution","status":"ready"}`) || !strings.Contains(readiness, `{"blockers":[],"operation":"work-item","status":"ready"}`) {
		t.Fatal(readiness)
	}
}

func applyPolicy(t *testing.T, e setupEnv, project, draft string) map[string]any {
	t.Helper()
	policy := []string{"project", "configure", "--project", project, "--policy-draft", draft}
	preview := e.run("", policy...)
	digest := text(preview, "policyEdit", "preview", "digest")
	id := text(preview, "policyEdit", "preview", "projectId")
	if digest == "" || id == "" || field(preview, "policyEdit", "preview", "workflow", "compatible") != true {
		t.Fatal("no compatible policy preview", mustJSON(t, preview))
	}
	before := treeDigest(t, e.projects)
	if _, code := e.exec(true, append(policy, "--project-id", id, "--preview-digest", strings.Repeat("0", 64), "--authorize-local")...); code == 0 || treeDigest(t, e.projects) != before {
		t.Fatal("stale policy preview applied")
	}
	applied := e.run("", append(policy, "--project-id", id, "--preview-digest", digest, "--authorize-local")...)
	if text(applied, "policyEdit", "readBack", "status") != "confirmed" {
		t.Fatal("policy read-back", applied["policyEdit"])
	}
	return applied
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
	// Reuse: an existing local Profile is listed, not recreated.
	if got := mustJSON(t, field(e.run("inspected", "runtime", "profile", "list"), "runtimeAuthoring", "profiles")); !strings.Contains(got, `"id":"worker"`) {
		t.Fatal(got)
	}
	e.run("profile_exists", "runtime", "profile", "create", "--runtime", "codex", "--profile", "worker", "--model", "operator-model", "--effort-mode", "explicit", "--effort", "low", "--capabilities", "read")
	applyPolicy(t, e, "partial", `{"runtimes":{"mode":"replace","values":[{"id":"codex"}]},"modelProfiles":{"mode":"replace","values":[{"key":"worker","runtimeRef":"codex","model":"operator-model"}]}}`)
	// Without a selected Workflow, readiness stays metadata-only: Runtime
	// resolution is explicitly unobserved, never ready.
	if got := e.run("", "project", "validate", "--project", "partial"); strings.Contains(mustJSON(t, got), "runtime_policy_unavailable") || text(got, "readiness", "runtime", "detail") != "inventory_unavailable" {
		t.Fatal("readiness after policy", mustJSON(t, got["readiness"]))
	}
	listed, _ := field(e.run("", "project", "workflow", "list", "--project", "partial"), "workflowAuthoring", "revisions").([]any)
	last := listed[len(listed)-1].(map[string]any)
	selectArgs := []string{"project", "workflow", "select", "--project", "partial", "--workflow", text(last, "workflowId"), "--revision", strconv.Itoa(int(last["revision"].(float64))), "--digest", text(last, "digest"), "--source", text(last, "source")}
	preview := e.run("", selectArgs...)
	e.run("", append(selectArgs, "--expected-revision", text(preview, "workflowAuthoring", "projectRevision"), "--preview-digest", text(preview, "workflowAuthoring", "previewDigest"), "--authorize-local")...)
	// Selection makes Runtime readiness observable. Execution remains blocked
	// only by the Provider gap, which is reported with its own cause; the
	// policy is no longer editable by G-1.
	after := e.run("", "project", "validate", "--project", "partial")
	if got := mustJSON(t, after["readiness"]); text(after, "readiness", "runtime", "status") != "ready" || !strings.Contains(got, `{"blockers":[{"code":"capability_mapping_missing","subject":"work-item"}],"operation":"execution","status":"blocked"}`) || strings.Contains(got, "runtime_") {
		t.Fatal("readiness after selection", got)
	}
	if !strings.Contains(mustJSON(t, e.run("", "project", "configure", "--project", "partial", "--policy-draft", "{}")), "requires schema 2 or 3") {
		t.Fatal("policy edit after selection not refused as documented")
	}
}

// CR-356-02 through the production parser: JSON and human output name
// --enabled, and the refusal changes no state.
func TestProfileEnabledRefusalNamesFieldInJSONAndHuman(t *testing.T) {
	e := newSetupEnv(t, "codex")
	e.authorizeRuntime("runtime", "binding", "create", "--runtime", "codex", "--profile", "base", "--model", "m", "--effort-mode", "explicit", "--effort", "low", "--capabilities", "read")
	before := treeDigest(t, e.state)
	for _, op := range []string{"create", "edit", "validate-candidate"} {
		profile := "p"
		if op == "edit" {
			profile = "base"
		}
		args := []string{"runtime", "profile", op, "--runtime", "codex", "--profile", profile, "--model", "m", "--effort-mode", "explicit", "--effort", "low", "--capabilities", "read", "--enabled", "true"}
		refused := e.run("invalid_input", args...)
		if got := diagnosticFields(refused); len(got) != 1 || got[0] != "field_invalid:--enabled" || !strings.Contains(text(refused, "next"), "Remove --enabled") {
			t.Fatal(op, got, text(refused, "next"))
		}
		human, code := e.exec(false, args...)
		if code == 0 || !bytes.Contains(human, []byte("field: --enabled")) || !bytes.Contains(human, []byte("Remove --enabled")) {
			t.Fatalf("%s human: %s", op, human)
		}
	}
	if treeDigest(t, e.state) != before {
		t.Fatal("refusal changed state")
	}
}
