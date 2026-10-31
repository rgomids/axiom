//go:build linux || darwin

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// stubRuntimeScript is a local stand-in for the codex executable. It uses only
// shell builtins, reads the frozen task packet on stdin and writes the declared
// output into its private attempt directory. Instructions select failure modes;
// it never contacts a vendor and has no credentials.
const stubRuntimeScript = `#!/bin/sh
IFS= read -r packet
dir=${packet#*\"OutputDirectory\":\"}
dir=${dir%%\"*}
stage=${packet#*\"StageID\":\"}
stage=${stage%%\"*}
case "$packet" in
*mode:fail*) exit 3 ;;
*mode:missing*) exit 0 ;;
*mode:hang*) while :; do :; done ;;
esac
printf 'result of %s\n' "$stage" > "$dir/result"
exit 0
`

type executionRunEvent struct {
	Status   string        `json:"status"`
	Category string        `json:"category"`
	Next     string        `json:"next"`
	Stage    executionView `json:"stage"`
	Raw      string        `json:"-"`
}

type executionRunFixture struct {
	env workItemSourceEnvironment
}

func newExecutionRunFixture(t *testing.T) *executionRunFixture {
	t.Helper()
	temp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(temp, "bin")
	home := filepath.Join(temp, "home")
	for _, dir := range []string{bin, home} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte(stubRuntimeScript), 0o700); err != nil {
		t.Fatal(err)
	}
	// Host Runtime configuration and credentials are never consulted.
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	os.Unsetenv("CODEX_HOME")
	t.Setenv("PATH", bin)
	t.Setenv("TMPDIR", temp)
	f := &executionRunFixture{env: newWorkItemSourceEnvironment(t)}
	installLinkedTarget(t, f.env)
	publishRuntimeProfileV3(t, f.env.state, singleRuntimeV3("codex", "profile-codex", "stub-model", []string{"read"}, []string{"medium"}))
	return f
}

func (f *executionRunFixture) run(t *testing.T, want int, args ...string) executionRunEvent {
	t.Helper()
	var out bytes.Buffer
	code := cli.Run(context.Background(), args, f.env.service, currentProvenance(), &out)
	var event executionRunEvent
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &event); err != nil {
		t.Fatalf("%v: invalid JSON %q", args, out.String())
	}
	event.Raw = out.String()
	if code != want {
		t.Fatalf("%v: exit %d want %d: %s", args, code, want, out.String())
	}
	return event
}

func (f *executionRunFixture) executions(t *testing.T) []string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(f.env.state, "executions", "v2", "*", "*", "*.json"))
	matches2, _ := filepath.Glob(filepath.Join(f.env.state, "executions", "v2", "*", "*.json"))
	return append(matches, matches2...)
}

var minimalSelections = []string{"--workflow", "minimal", "--select", "agent.runtime=codex", "--select", "agent.model=stub-model", "--select", "agent.effort=runtime-default"}

func minimalRun(extra ...string) []string {
	args := append([]string{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7"}, minimalSelections...)
	return append(args, extra...)
}

func latestAttempt(t *testing.T, v executionView, stage string) executionAttemptView {
	t.Helper()
	for _, s := range v.Stages {
		if s.StageID == stage && len(s.Tasks) != 0 && len(s.Tasks[0].Attempts) != 0 {
			return s.Tasks[0].Attempts[len(s.Tasks[0].Attempts)-1]
		}
	}
	t.Fatalf("no attempt in stage %s: %+v", stage, v)
	return executionAttemptView{}
}

func TestPublicWorkflowRunMinimalDefaultCompletesWithoutPlanPolicyOrProfile(t *testing.T) {
	f := newExecutionRunFixture(t)
	got := f.run(t, cli.ExitSuccess, minimalRun("--authorize-external")...)
	if got.Category != "execution_completed" || got.Stage.Status != workflow.RunCompleted || got.Stage.WorkflowID != workflow.DefaultWorkflowID {
		t.Fatalf("run: %s", got.Raw)
	}
	a := latestAttempt(t, got.Stage, workflow.DefaultStageID)
	if a.Status != workflow.AttemptSucceeded || a.Runtime != "codex" || a.Model != "stub-model" || a.Effort != "runtime-default" || a.Profile != "" || len(a.Artifacts) != 1 || a.Artifacts[0].Name != "result" || a.Process == nil {
		t.Fatalf("attempt: %+v", a)
	}
	for _, p := range a.Provenance {
		if p.Field != "instructions" && p.Source != "run_override" || p.Field == "instructions" && p.Source != "workflow_agent" {
			t.Fatalf("provenance: %+v", a.Provenance)
		}
	}
	// No executable path, instruction text or Stage Plan / Policy state.
	for _, forbidden := range []string{filepath.Join(f.env.state), "Complete the linked Work Item", "previewDigest"} {
		if strings.Contains(got.Raw, forbidden) {
			t.Fatalf("unsafe projection contains %q", forbidden)
		}
	}
	for _, retired := range []string{"graphs", "single-stage-runs", "validator-policies"} {
		if _, err := os.Lstat(filepath.Join(f.env.state, retired)); !os.IsNotExist(err) {
			t.Fatalf("retired store %s written: %v", retired, err)
		}
	}
	status := f.run(t, cli.ExitSuccess, "workflow", "stage", "status", "--project", "external", "--execution", got.Stage.ExecutionID)
	if status.Stage.Revision != got.Stage.Revision || status.Stage.Status != workflow.RunCompleted || !strings.Contains(status.Stage.NextAction, "not human acceptance") {
		t.Fatalf("status: %s", status.Raw)
	}
}

func TestPublicWorkflowRunRefusesBeforeAnyWrite(t *testing.T) {
	f := newExecutionRunFixture(t)
	cases := []struct {
		name     string
		args     []string
		category string
	}{
		{"no external authority", minimalRun(), "authority_required"},
		{"wrong repository link", append([]string{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/other#7", "--authorize-external"}, minimalSelections...), "work_item_not_linked"},
		{"ambiguous same-level selection", minimalRun("--select", "work/agent.model=other", "--authorize-external"), "invocation_selection_conflict"},
		{"missing effort", []string{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--workflow", "minimal", "--select", "agent.runtime=codex", "--select", "agent.model=m", "--authorize-external"}, "invocation_selection_missing"},
		{"unknown agent", minimalRun("--select", "ghost.model=m", "--authorize-external"), "recovery_required"},
		{"unconfigured runtime", []string{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--workflow", "minimal", "--select", "agent.runtime=claude", "--select", "agent.model=m", "--select", "agent.effort=runtime-default", "--authorize-external"}, "runtime_not_configured"},
	}
	for _, c := range cases {
		got := f.run(t, cli.ExitFailure, c.args...)
		if got.Category != c.category {
			t.Fatalf("%s: %s", c.name, got.Raw)
		}
		if n := f.executions(t); len(n) != 0 {
			t.Fatalf("%s wrote an Execution: %v", c.name, n)
		}
	}
	for _, args := range [][]string{minimalRun("--approve-gate", "work", "--authorize-external"), minimalRun("--authorize-human", "--authorize-external")} {
		if got := f.run(t, cli.ExitFailure, args...); got.Category != "human_authority_required" || len(f.executions(t)) != 0 {
			t.Fatalf("unpaired human authority: %s", got.Raw)
		}
	}
	for _, args := range [][]string{
		{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--workflow", "custom", "--authorize-external"},
		minimalRun("--select", "runtime", "--authorize-external"),
	} {
		if got := f.run(t, cli.ExitFailure, args...); got.Category != "invalid_selection" || len(f.executions(t)) != 0 {
			t.Fatalf("invalid selection accepted: %s", got.Raw)
		}
	}
}

func TestPublicWorkflowRunFailureMissingOutputRetryAndNewSelection(t *testing.T) {
	f := newExecutionRunFixture(t)
	failed := f.run(t, cli.ExitFailure, minimalRun("--select", "agent.instructions=mode:fail", "--authorize-external")...)
	a := latestAttempt(t, failed.Stage, "work")
	if failed.Category != "execution_failed" || a.Status != workflow.AttemptFailed || !strings.HasPrefix(a.Diagnostic, "runtime_exit_nonzero") || len(a.Artifacts) != 0 {
		t.Fatalf("failure: %s", failed.Raw)
	}
	id := failed.Stage.ExecutionID
	retry := []string{"workflow", "stage", "retry", "--project", "external", "--execution", id, "--stage", "work", "--agent", "agent", "--disposition", "preserve", "--authorize-external"}
	// An ordinary retry reuses the frozen invocation (still mode:fail).
	again := f.run(t, cli.ExitFailure, retry...)
	if b := latestAttempt(t, again.Stage, "work"); b.Number != 2 || b.InstructionsDigest != a.InstructionsDigest || b.Status != workflow.AttemptFailed {
		t.Fatalf("ordinary retry changed the frozen invocation: %+v", b)
	}
	// Selections without explicit new-selection authority are refused.
	if got := f.run(t, cli.ExitFailure, append(retry, "--select", "agent.instructions=mode:missing")...); got.Category != "selections_frozen" {
		t.Fatalf("implicit reselection: %s", got.Raw)
	}
	missing := f.run(t, cli.ExitFailure, append(retry, "--new-selection", "--select", "agent.instructions=mode:missing")...)
	c := latestAttempt(t, missing.Stage, "work")
	if c.Number != 3 || c.InstructionsDigest == a.InstructionsDigest || c.Status != workflow.AttemptFailed || !strings.HasPrefix(c.Diagnostic, "missing_output") {
		t.Fatalf("exit zero without declared output accepted: %+v", c)
	}
	done := f.run(t, cli.ExitSuccess, append(retry, "--new-selection", "--select", "agent.instructions=complete it")...)
	if done.Stage.Status != workflow.RunCompleted || latestAttempt(t, done.Stage, "work").Number != 4 {
		t.Fatalf("explicit new selection: %s", done.Raw)
	}
	// Prior attempts keep their frozen digests.
	attempts := done.Stage.Stages[0].Tasks[0].Attempts
	if attempts[0].InstructionsDigest != a.InstructionsDigest || attempts[2].InstructionsDigest != c.InstructionsDigest {
		t.Fatal("new selection rebound an earlier attempt")
	}
	if got := f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--execution", id, "--select", "agent.model=x", "--authorize-external"); got.Category != "selections_frozen" {
		t.Fatalf("continuation changed start selections: %s", got.Raw)
	}
}

func TestPublicWorkflowRunRunnerFenceAndUnknownEffectsNeverReplay(t *testing.T) {
	f := newExecutionRunFixture(t)
	failed := f.run(t, cli.ExitFailure, minimalRun("--select", "agent.instructions=mode:fail", "--authorize-external")...)
	id := failed.Stage.ExecutionID
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// A held runner lease (live or stale owner) wins; nothing is spawned.
	lease, err := store.AcquireFence(ctx, workItemSourceProjectID, id)
	if err != nil {
		t.Fatal(err)
	}
	retry := []string{"workflow", "stage", "retry", "--project", "external", "--execution", id, "--stage", "work", "--agent", "agent", "--disposition", "preserve", "--authorize-external"}
	if got := f.run(t, cli.ExitFailure, retry...); got.Category != "execution_runner_active" {
		t.Fatalf("second runner: %s", got.Raw)
	}
	// Simulate a runner that prepared an attempt and crashed before any
	// terminal evidence.
	gen := lease.Generation()
	frozen := failed.Stage.Stages[0].Tasks[0].Attempts[0]
	if _, err := store.UpdateFenced(ctx, workItemSourceProjectID, id, lease, func(x *workflow.Execution) error {
		if err := x.ClaimRunner(gen, time.Now()); err != nil {
			return err
		}
		last := x.Stages[0].Tasks[0].Attempts[0]
		return x.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "agent"}, AttemptID: "crashed-attempt", Generation: gen, Invocation: last.Invocation, Authority: last.Authority, Now: time.Now(), Retry: true, WorkspaceDisposition: "preserve"})
	}); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	recovered := f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--execution", id, "--authorize-external")
	crashed := latestAttempt(t, recovered.Stage, "work")
	if recovered.Category != "recovery_required" || crashed.AttemptID != "crashed-attempt" || crashed.Status != workflow.AttemptUnknown || crashed.Diagnostic != "runner_lost_before_terminal_evidence" {
		t.Fatalf("lost runner: %s", recovered.Raw)
	}
	if got := f.run(t, cli.ExitFailure, retry...); got.Category != "recovery_required" || len(got.Stage.Stages[0].Tasks[0].Attempts) != 2 {
		t.Fatalf("unknown effect replayed: %s", got.Raw)
	}
	if got := f.run(t, cli.ExitFailure, "workflow", "stage", "reconcile", "--project", "external", "--execution", id); got.Category != "authority_required" {
		t.Fatalf("reconcile without local authority: %s", got.Raw)
	}
	reconciled := f.run(t, cli.ExitSuccess, "workflow", "stage", "reconcile", "--project", "external", "--execution", id, "--authorize-local")
	if reconciled.Stage.Status != workflow.RunRecoveryRequired || len(reconciled.Stage.Stages[0].Tasks[0].Attempts) != 2 {
		t.Fatalf("reconcile changed facts: %s", reconciled.Raw)
	}
	if got := f.run(t, cli.ExitFailure, "workflow", "stage", "reconcile", "--project", "external", "--execution", id, "--abandon", "crashed-attempt"); got.Category != "human_authority_required" {
		t.Fatalf("abandon without human authority: %s", got.Raw)
	}
	abandoned := f.run(t, cli.ExitSuccess, "workflow", "stage", "reconcile", "--project", "external", "--execution", id, "--abandon", "crashed-attempt", "--authorize-human")
	if a := latestAttempt(t, abandoned.Stage, "work"); a.Status != workflow.AttemptUnknown || a.Resolution == nil || !strings.HasPrefix(a.Resolution.Actor, "human:") {
		t.Fatalf("abandon: %s", abandoned.Raw)
	}
	again := f.run(t, cli.ExitFailure, retry...)
	if a := latestAttempt(t, again.Stage, "work"); a.Number != 3 || a.InstructionsDigest != frozen.InstructionsDigest || a.RunnerGeneration <= gen {
		t.Fatalf("retry after disposition: %+v", a)
	}
}

func TestPublicWorkflowRunCancelDuringSpawnStopsOwnedProcess(t *testing.T) {
	f := newExecutionRunFixture(t)
	var wg sync.WaitGroup
	var result executionRunEvent
	wg.Add(1)
	go func() {
		defer wg.Done()
		var out bytes.Buffer
		cli.Run(context.Background(), minimalRun("--select", "agent.instructions=mode:hang", "--authorize-external"), f.env.service, currentProvenance(), &out)
		_ = json.Unmarshal(bytes.TrimSpace(out.Bytes()), &result)
		result.Raw = out.String()
	}()
	id := ""
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if files := f.executions(t); len(files) == 1 {
			id = strings.TrimSuffix(filepath.Base(files[0]), ".json")
			var out bytes.Buffer
			cli.Run(context.Background(), []string{"workflow", "stage", "status", "--project", "external", "--execution", id}, f.env.service, currentProvenance(), &out)
			if strings.Contains(out.String(), `"status":"running"`) && strings.Contains(out.String(), `"pid":`) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if id == "" {
		t.Fatal("no running Execution observed")
	}
	f.run(t, cli.ExitSuccess, "workflow", "stage", "cancel", "--project", "external", "--execution", id, "--authorize-local")
	wg.Wait()
	a := latestAttempt(t, result.Stage, "work")
	if result.Category != "execution_cancelled" || a.Status != workflow.AttemptCancelled || a.Diagnostic != "group_stop_confirmed" {
		t.Fatalf("cancel during run: %s", result.Raw)
	}
	if got := f.run(t, cli.ExitCancelled, "workflow", "stage", "retry", "--project", "external", "--execution", id, "--stage", "work", "--agent", "agent", "--authorize-external"); got.Category != "execution_cancelled" {
		t.Fatalf("retry after cancel: %s", got.Raw)
	}
}

func TestPublicWorkflowRunSelectedWorkflowStopsAtRealGate(t *testing.T) {
	f := newExecutionRunFixture(t)
	f.selectTwoStageWorkflow(t, true)
	args := []string{"workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--select", "owner.runtime=codex", "--select", "owner.model=stub-model", "--select", "owner.effort=runtime-default", "--authorize-external"}
	first := f.run(t, cli.ExitSuccess, args...)
	if first.Stage.Status != workflow.RunAwaitingGate || first.Stage.Stages[0].Status != workflow.StageCompleted || first.Stage.Stages[1].Status != workflow.StageAwaitingGate {
		t.Fatalf("gate: %s", first.Raw)
	}
	second := f.run(t, cli.ExitSuccess, "workflow", "run", "--project", "external", "--execution", first.Stage.ExecutionID, "--approve-gate", "specification", "--authorize-human", "--authorize-external")
	if second.Stage.Status != workflow.RunCompleted || second.Stage.Stages[1].Gate == nil || !strings.HasPrefix(second.Stage.Stages[1].Gate.Actor, "human:") {
		t.Fatalf("approval: %s", second.Raw)
	}
}

func TestRetiredStageCommandsReturnReplacementGuidance(t *testing.T) {
	f := newExecutionRunFixture(t)
	for _, name := range []string{"plan", "run", "integrate"} {
		var out bytes.Buffer
		code := cli.Run(context.Background(), []string{"workflow", "stage", name, "--project", "external"}, f.env.service, currentProvenance(), &out)
		if code == cli.ExitSuccess || !strings.Contains(out.String(), `"category":"command_retired"`) || !strings.Contains(out.String(), "axiom workflow run") {
			t.Fatalf("%s: %d %s", name, code, out.String())
		}
	}
	var help bytes.Buffer
	if _, code := cli.HandleHelp([]string{"workflow", "run", "--help"}, currentProvenance(), &help); code != cli.ExitSuccess {
		t.Fatalf("help: %s", help.String())
	}
	for _, flag := range []string{"--select", "--authorize-external", "--approve-gate", "--workflow"} {
		if !strings.Contains(help.String(), flag) {
			t.Fatalf("help lacks %s: %s", flag, help.String())
		}
	}
	for _, retired := range []string{"--plan", "--validator-policy", "--preview-digest", "--expected-revision"} {
		if strings.Contains(help.String(), retired) {
			t.Fatalf("run help advertises %s", retired)
		}
	}
}

func (f *executionRunFixture) selectTwoStageWorkflow(t *testing.T, gate bool) {
	t.Helper()
	builtin, _ := workflowdefinition.BuiltinRevision(2)
	d := builtin.Definition
	d.WorkflowID, d.Name, d.Revision = "two-stage-stub", "Two stages with local stub", 1
	d.Stages = d.Stages[:2]
	for i := range d.Stages {
		s := &d.Stages[i]
		s.Validators, s.HumanGates, s.CompletionCriteria = []workflowdefinition.Validator{}, []workflowdefinition.Gate{}, []workflowdefinition.Criterion{}
		s.Checkpoint = true
		if gate && i == 1 {
			s.Phase = "completion"
			s.HumanGates = []workflowdefinition.Gate{{Kind: "human_acceptance", Timing: "after"}}
		}
	}
	doc, issues := workflowdefinition.Encode(d)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	file := filepath.Join(t.TempDir(), "workflow.json")
	if err := os.WriteFile(file, doc.Canonical, 0o600); err != nil {
		t.Fatal(err)
	}
	author := f.env.service.(lifecycleService)
	author.projectsRoot = f.env.elsewhere
	var err error
	author.portable, err = local.NewPortableStore(f.env.elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(in cli.ProjectWorkflowInput) cli.Result {
		t.Helper()
		preview := author.ProjectWorkflow(context.Background(), in)
		if preview.WorkflowAuthoring == nil {
			t.Fatalf("preview: %+v", preview)
		}
		in.PreviewDigest, in.ExpectedRevision, in.AuthorizeLocal = preview.WorkflowAuthoring.PreviewDigest, preview.WorkflowAuthoring.ProjectRevision, true
		out := author.ProjectWorkflow(context.Background(), in)
		if out.Category != "applied" {
			t.Fatalf("apply: %+v", out)
		}
		return out
	}
	created := apply(cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "create", Project: "external"}, File: file})
	apply(cli.ProjectWorkflowInput{WorkflowRequest: projectapp.WorkflowRequest{Operation: "select", Project: "external", Ref: *created.WorkflowAuthoring.Reference}})
}

func TestPublicWorkflowRunTwoStagesProgressWithoutManualIntervention(t *testing.T) {
	f := newExecutionRunFixture(t)
	f.selectTwoStageWorkflow(t, false)
	got := f.run(t, cli.ExitSuccess, "workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--select", "owner.runtime=codex", "--select", "owner.model=stub-model", "--select", "owner.effort=runtime-default", "--authorize-external")
	if got.Stage.Status != workflow.RunCompleted || len(got.Stage.Stages) != 2 {
		t.Fatalf("two-stage run: %s", got.Raw)
	}
	for _, s := range got.Stage.Stages {
		if s.Status != workflow.StageCompleted || len(s.Tasks[0].Attempts) != 1 {
			t.Fatalf("stage: %+v", s)
		}
	}
}

func TestPublicWorkflowRunRefusesUnsupportedWorkflowControls(t *testing.T) {
	f := newExecutionRunFixture(t)
	got := f.run(t, cli.ExitFailure, "workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--select", "owner.runtime=codex", "--select", "owner.model=stub-model", "--select", "owner.effort=runtime-default", "--authorize-external")
	if got.Category != "workflow_unmappable" || len(f.executions(t)) != 0 {
		t.Fatalf("configured validators silently skipped: %s", got.Raw)
	}
}
