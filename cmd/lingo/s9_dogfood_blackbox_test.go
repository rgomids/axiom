package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGitHubStateVariable switches the test binary into a stateful fake `gh`
// so the executable exercises the real GitHub adapter without a network.
const fakeGitHubStateVariable = "AXIOM_TEST_FAKE_GH_STATE"

func TestMain(m *testing.M) {
	if path := os.Getenv(fakeGitHubStateVariable); path != "" {
		os.Exit(runFakeGitHub(path, os.Args[1:], os.Stdin, os.Stdout))
	}
	os.Exit(m.Run())
}

type fakeGitHub struct {
	RepositoryLabels []string `json:"repositoryLabels"`
	IssueLabels      []string `json:"issueLabels"`
	Comments         []string `json:"comments"`
	IssueCreated     bool     `json:"issueCreated"`
	Mutations        []string `json:"mutations"`
}

func runFakeGitHub(path string, args []string, stdin io.Reader, stdout io.Writer) int {
	wire, err := os.ReadFile(path)
	if err != nil {
		return 1
	}
	var state fakeGitHub
	if err := json.Unmarshal(wire, &state); err != nil {
		return 1
	}
	method, endpoint := "GET", ""
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "api", "--include":
		case "--method":
			index++
			method = args[index]
		case "--input", "-f":
			index++
		default:
			if endpoint == "" {
				endpoint = args[index]
			}
		}
	}
	body, _ := io.ReadAll(stdin)
	const issue = "repos/owner/repo/issues/7"
	names := func(values []string) []map[string]string {
		result := make([]map[string]string, 0, len(values))
		for _, value := range values {
			result = append(result, map[string]string{"name": value})
		}
		return result
	}
	var response any
	switch {
	case method == "GET" && endpoint == "search/issues":
		response = map[string]any{"total_count": 0, "items": []any{}}
	case method == "POST" && endpoint == "repos/owner/repo/issues":
		var request struct{ Labels []string }
		if json.Unmarshal(body, &request) != nil {
			return 1
		}
		state.IssueLabels = append([]string(nil), request.Labels...)
		state.IssueCreated = true
		state.Mutations = append(state.Mutations, "create_issue")
		response = map[string]any{"number": 7, "html_url": "https://github.com/owner/repo/issues/7", "state": "open", "labels": names(state.IssueLabels)}
	case method == "GET" && strings.HasPrefix(endpoint, "repos/owner/repo/labels?per_page=100"):
		response = names(state.RepositoryLabels)
	case !state.IssueCreated:
		return 1
	case method == "POST" && endpoint == "repos/owner/repo/labels":
		var label struct{ Name string }
		if json.Unmarshal(body, &label) != nil || label.Name == "" {
			return 1
		}
		state.RepositoryLabels = append(state.RepositoryLabels, label.Name)
		state.Mutations = append(state.Mutations, "create_label:"+label.Name)
		response = map[string]string{"name": label.Name}
	case method == "GET" && endpoint == issue:
		response = map[string]any{"number": 7, "html_url": "https://github.com/owner/repo/issues/7", "state": "open", "labels": names(state.IssueLabels)}
	case method == "POST" && endpoint == issue+"/labels":
		var request struct{ Labels []string }
		if json.Unmarshal(body, &request) != nil || len(request.Labels) != 1 {
			return 1
		}
		state.IssueLabels = append(state.IssueLabels, request.Labels[0])
		state.Mutations = append(state.Mutations, "add_label:"+request.Labels[0])
		response = names(state.IssueLabels)
	case method == "DELETE" && strings.HasPrefix(endpoint, issue+"/labels/"):
		name := strings.TrimPrefix(endpoint, issue+"/labels/")
		kept := []string{}
		for _, label := range state.IssueLabels {
			if label != name {
				kept = append(kept, label)
			}
		}
		if len(kept) == len(state.IssueLabels) {
			return 1
		}
		state.IssueLabels = kept
		state.Mutations = append(state.Mutations, "remove_label:"+name)
		response = names(state.IssueLabels)
	case method == "GET" && endpoint == issue+"/comments?per_page=100":
		comments := make([]map[string]string, 0, len(state.Comments))
		for _, comment := range state.Comments {
			comments = append(comments, map[string]string{"body": comment})
		}
		response = comments
	case method == "POST" && endpoint == issue+"/comments":
		var comment struct{ Body string }
		if json.Unmarshal(body, &comment) != nil || comment.Body == "" {
			return 1
		}
		state.Comments = append(state.Comments, comment.Body)
		state.Mutations = append(state.Mutations, "comment")
		response = map[string]int{"id": len(state.Comments)}
	default:
		return 1
	}
	updated, err := json.Marshal(state)
	if err != nil || os.WriteFile(path, updated, 0o600) != nil {
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(response); err != nil {
		return 1
	}
	return 0
}

// TestExecutableClaudeRuntimeAndFirstProjectionOfCreatedWorkItem reproduces
// both Major findings of the S9 dogfood against v0.1.2-rc.1: an Execution
// conducted by Claude was persisted as codex, and `work-item create` →
// `workflow start` → first transition → `workflow reconcile` required
// recovery because the new Issue carried no axiom:stage:* marker.
// Both concrete Runtimes start only through the protocol the installed
// axiom-work-item-run skill teaches, on the Runtime Lingo itself observes.
func TestExecutableRuntimeAndFirstProjectionOfCreatedWorkItem(t *testing.T) {
	for _, runtimeID := range []string{"claude", "codex"} {
		t.Run(runtimeID, func(t *testing.T) { executableRuntimeJourney(t, runtimeID) })
	}
}

func executableRuntimeJourney(t *testing.T, runtimeID string) {
	other := map[string]string{"claude": "codex", "codex": "claude"}[runtimeID]
	binary := filepath.Join(t.TempDir(), testExecutableName("axiom"))
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build executable: %v: %s", err, output)
	}
	fakeGH, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(t.TempDir(), "github.json")
	writeFake := func(state fakeGitHub) {
		t.Helper()
		wire, err := json.Marshal(state)
		if err != nil || os.WriteFile(statePath, wire, 0o600) != nil {
			t.Fatalf("write fake GitHub state: %v", err)
		}
	}
	readFake := func() fakeGitHub {
		t.Helper()
		wire, err := os.ReadFile(statePath)
		var state fakeGitHub
		if err != nil || json.Unmarshal(wire, &state) != nil {
			t.Fatalf("read fake GitHub state: %v", err)
		}
		return state
	}
	writeFake(fakeGitHub{RepositoryLabels: []string{"bug"}})

	portable := filepath.Join(t.TempDir(), "portable")
	state := filepath.Join(t.TempDir(), "state")
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	runtimeBin := t.TempDir()
	environment := append(os.Environ(), "HOME="+t.TempDir(), "PATH="+runtimeBin, "CLAUDE_CONFIG_DIR=", "LINGO_PROJECTS_ROOT="+portable, "LINGO_STATE_ROOT="+state, "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(t.TempDir(), "skills"), "AXIOM_GH_BIN="+fakeGH, fakeGitHubStateVariable+"="+statePath)
	type event struct {
		canonicalEvent
		Workflow *struct {
			ExecutionID, CurrentGate, RuntimeID string
			Revision                            uint64
		} `json:"workflow"`
	}
	run := func(wantCode int, wantStatus string, args ...string) event {
		t.Helper()
		command := exec.Command(binary, append([]string{"--json"}, args...)...)
		command.Env = environment
		command.Dir = t.TempDir()
		output, err := command.Output()
		code := 0
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		var decoded event
		if err := json.Unmarshal(bytes.TrimSpace(output), &decoded); err != nil {
			t.Fatalf("%v: invalid JSON %q: %v", args, output, err)
		}
		if code != wantCode || decoded.Status != wantStatus {
			t.Fatalf("%v: code=%d output=%s", args, code, output)
		}
		return decoded
	}

	setup := run(0, "success", "project", "configure", "--slug", "configured", "--name", "Configured", "--repository", "main="+repository, "--work-item-provider", "github")
	run(0, "success", "project", "configure", "--project-id", setup.Setup.ProjectID, "--slug", "configured", "--name", "Configured", "--repository", "main="+repository, "--work-item-provider", "github", "--preview-digest", setup.Setup.Digest, "--authorize-local")
	draftArgs := []string{"work-item", "create", "--project", "configured", "--repository", "main", "--provider-repository", "owner/repo", "--intent", "Dogfood regression", "--desired-outcome", "First projection converges", "--context", "S9 dogfood", "--scope", "Bounded change", "--constraints", "Fail closed on drift", "--non-goals", "No release", "--acceptance", "Tests pass"}
	draft := run(0, "success", draftArgs...)
	// Classify the same delivery as a bug, review its concrete provider labels,
	// and prove that preview performed no mutation.
	draftArgs = append(draftArgs, "--type", "bug")
	draft = run(0, "success", draftArgs...)
	if draft.Draft == nil || draft.Draft.Draft.Type != "bug" || len(draft.Draft.ProviderDocument.Metadata.Labels) != 1 || draft.Draft.ProviderDocument.Metadata.Labels[0] != "bug" || len(readFake().Mutations) != 0 {
		t.Fatalf("classification preview=%+v state=%+v", draft.Draft, readFake())
	}
	created := run(0, "success", append(draftArgs, "--preview-digest", draft.Draft.Digest, "--authorize-external")...)
	if created.WorkItem == nil || created.WorkItem.ExternalID != "7" {
		t.Fatalf("created Work Item = %+v", created.WorkItem)
	}
	if applied := readFake(); len(applied.IssueLabels) != 1 || applied.IssueLabels[0] != "bug" || len(applied.Mutations) != 1 {
		t.Fatalf("creation classification=%+v", applied)
	}
	// A human adds a foreign label after creation; still no axiom:stage:*.
	github := readFake()
	github.IssueLabels = []string{"triage"}
	writeFake(github)

	selector := []string{"--project", "configured", "--repository", "main", "--work-item", "github:owner/repo#7"}
	executions := filepath.Join(state, "executions", "v1", setup.Setup.ProjectID, "*.json")
	for _, invalid := range [][]string{{"--runtime", "gemini"}, {"--runtime", ""}, {"--runtime", "Claude"}, {"--runtime", "claude", "--runtime", "codex"}} {
		run(1, "validation_failure", append(append([]string{"workflow", "start"}, selector...), invalid...)...)
	}
	if records, _ := filepath.Glob(executions); len(records) != 0 {
		t.Fatalf("invalid Runtime selector created Executions: %v", records)
	}

	installTestRuntimePolicy(t, state, setup.Setup.ProjectID, runtimeID)
	protocol := workItemRunStartProtocol(t)
	values := map[string]string{"role": "implementation", "complexity": "high", "capabilities": "axiom-skills", "runtime": runtimeID}
	previewArgs := expandProtocol(t, protocol[0], selector, values)
	// Lingo observes the Runtime itself: without its executable and verified
	// Axiom skill integration the reviewed policy cannot resolve.
	run(1, "validation_failure", previewArgs...)
	executed := filepath.Join(t.TempDir(), runtimeID+"-executed")
	if err := os.WriteFile(filepath.Join(runtimeBin, testExecutableName(runtimeID)), []byte("#!/bin/sh\ntouch "+executed+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	run(1, "validation_failure", previewArgs...)
	run(0, "success", "runtime", runtimeID, "install")
	// The other Runtime is neither allowed by the Project nor observed here.
	values["runtime"] = other
	if denied := run(1, "validation_failure", expandProtocol(t, protocol[0], selector, values)...); denied.RuntimeResolution == nil || denied.RuntimeResolution.Choice != nil {
		t.Fatalf("other Runtime resolved: %+v", denied.RuntimeResolution)
	}
	values["runtime"] = runtimeID
	runtimePreview := run(0, "success", previewArgs...)
	if runtimePreview.RuntimeResolution == nil || runtimePreview.RuntimeResolution.Choice == nil || runtimePreview.RuntimeResolution.Choice.RuntimeID != runtimeID || runtimePreview.RuntimeResolution.Choice.ExecutableDigest == "" || runtimePreview.PreviewDigest == "" || runtimePreview.Workflow != nil {
		t.Fatalf("preview = %+v", runtimePreview)
	}
	if records, _ := filepath.Glob(executions); len(records) != 0 {
		t.Fatalf("preview created Executions: %v", records)
	}
	values["previewDigest"] = strings.Repeat("0", 64)
	if stale := run(1, "validation_failure", expandProtocol(t, protocol[1], selector, values)...); stale.RuntimeResolution == nil || stale.RuntimeResolution.Blocker == nil || stale.RuntimeResolution.Blocker.Code != "stale_preview" {
		t.Fatalf("unreviewed digest = %+v", stale.RuntimeResolution)
	}
	values["previewDigest"] = runtimePreview.PreviewDigest
	started := run(0, "success", expandProtocol(t, protocol[1], selector, values)...)
	if started.Workflow == nil || started.Workflow.RuntimeID != runtimePreview.RuntimeResolution.Choice.RuntimeID || started.Workflow.Revision != 1 {
		t.Fatalf("started = %+v", started.Workflow)
	}
	values["executionId"] = started.Workflow.ExecutionID
	if status := run(0, "success", expandProtocol(t, protocol[2], selector, values)...); status.Workflow == nil || status.Workflow.ExecutionID != started.Workflow.ExecutionID || status.Workflow.RuntimeID != runtimeID {
		t.Fatalf("protocol status = %+v", status.Workflow)
	}
	if _, err := os.Lstat(executed); !os.IsNotExist(err) {
		t.Fatal("workflow start executed the Runtime")
	}
	execution := append(append([]string{}, selector...), "--execution", started.Workflow.ExecutionID)
	records, err := filepath.Glob(executions)
	if err != nil || len(records) != 1 {
		t.Fatalf("Execution records = %v, %v", records, err)
	}
	persisted, err := os.ReadFile(records[0])
	if err != nil || !strings.Contains(string(persisted), `"runtimeId":"`+runtimeID+`"`) {
		t.Fatalf("persisted Execution does not carry %s: %s %v", runtimeID, persisted, err)
	}
	// The Runtime of an existing Execution cannot be switched or re-selected.
	run(1, "validation_failure", append(append([]string{"workflow", "start"}, selector...), "--runtime", other)...)
	run(1, "validation_failure", append([]string{"workflow", "start"}, selector...)...)
	run(1, "validation_failure", append(append([]string{"workflow", "status"}, execution...), "--runtime", other)...)
	if unchanged, _ := os.ReadFile(records[0]); !bytes.Equal(unchanged, persisted) {
		t.Fatal("Runtime conflict changed the persisted Execution")
	}
	if status := run(0, "success", append([]string{"workflow", "status"}, execution...)...); status.Workflow.RuntimeID != runtimeID {
		t.Fatalf("status Runtime = %q", status.Workflow.RuntimeID)
	}

	advanced := run(0, "success", append(append([]string{"workflow", "advance"}, execution...), "--expected-revision", "1", "--gate", "intake", "--outcome", "pass")...)
	if advanced.Workflow.RuntimeID != runtimeID || advanced.Workflow.Revision != 2 {
		t.Fatalf("advanced = %+v", advanced.Workflow)
	}
	reconcile := append(append([]string{"workflow", "reconcile"}, execution...), "--expected-revision", "2")
	preview := run(0, "success", reconcile...)
	if preview.Projection == nil || preview.Workflow.RuntimeID != runtimeID {
		t.Fatalf("preview = %+v", preview)
	}
	var kinds []string
	for _, effect := range preview.Projection.Effects {
		kinds = append(kinds, effect.Kind+":"+strings.SplitN(effect.Value, "\n", 2)[0])
	}
	want := []string{"create_stage_label:axiom:stage:specifying", "add_stage_label:axiom:stage:specifying", "post_transition_comment:<!-- axiom:workflow-projection:" + preview.Projection.ProjectionKey + " -->"}
	if strings.Join(kinds, "|") != strings.Join(want, "|") {
		t.Fatalf("preview effects = %v want %v", kinds, want)
	}
	if mutations := readFake().Mutations; len(mutations) != 1 {
		t.Fatalf("preview mutated GitHub: %v", mutations)
	}

	authorized := append(append([]string{}, reconcile...), "--preview-digest", preview.Projection.Digest, "--authorize-external")
	applied := run(0, "success", authorized...)
	if applied.Workflow.RuntimeID != runtimeID {
		t.Fatalf("applied = %+v", applied.Workflow)
	}
	github = readFake()
	if strings.Join(github.IssueLabels, ",") != "triage,axiom:stage:specifying" || len(github.Comments) != 1 || strings.Join(github.Mutations, ",") != "create_issue,create_label:axiom:stage:specifying,add_label:axiom:stage:specifying,comment" {
		t.Fatalf("GitHub after projection = %+v", github)
	}

	run(0, "success", authorized...)
	if replay := run(0, "success", reconcile...); replay.Projection == nil || len(replay.Projection.Effects) != 0 {
		t.Fatalf("replay preview = %+v", replay.Projection)
	}
	if after := readFake(); len(after.Mutations) != len(github.Mutations) || len(after.Comments) != 1 {
		t.Fatalf("replay duplicated effects: %+v", after)
	}

	// Losing the established marker later is drift, never a new bootstrap.
	github.IssueLabels = []string{"triage"}
	writeFake(github)
	run(1, "failure", reconcile...)
	if after := readFake(); len(after.Mutations) != len(github.Mutations) {
		t.Fatalf("drift produced effects: %+v", after)
	}
}
