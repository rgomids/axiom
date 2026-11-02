//go:build linux || darwin

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

// countingRunner counts real process dispatches and runs a hook after each.
type countingRunner struct {
	base   stageexecution.ObservedRunner
	spawns int
	after  func(int)
}

func (r *countingRunner) RunObserved(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	r.spawns++
	result := r.base.RunObserved(ctx, inv, started)
	if r.after != nil {
		r.after(r.spawns)
	}
	return result
}

// buildHookRuntimeLauncher runs a hook inside Build of the given stage, after
// Resolve has passed and before the spawn.
type buildHookRuntimeLauncher struct {
	stageexecution.RuntimeLauncher
	stage string
	hook  func()
}

func (l buildHookRuntimeLauncher) Build(ctx context.Context, r stageexecution.LaunchRequest) (executiongraph.Invocation, error) {
	inv, err := l.RuntimeLauncher.Build(ctx, r)
	if r.Ref.StageID == l.stage {
		l.hook()
	}
	return inv, err
}

func operational(t *testing.T, root string, operation projectapp.OperationalOperation) {
	t.Helper()
	ctx := context.Background()
	store, err := local.NewOperationalStore(root)
	if err != nil {
		t.Fatal(err)
	}
	request := projectapp.OperationalRequest{ProjectID: workItemSourceProjectID, Operation: operation}
	preview := projectapp.ApplyOperational(ctx, store, request, "", false)
	if preview.Preview == nil {
		t.Fatalf("%s preview = %+v", operation, preview)
	}
	if result := projectapp.ApplyOperational(ctx, store, request, preview.Preview.Digest, true); result.Status != projectapp.OperationalCommitted {
		t.Fatalf("%s = %+v", operation, result)
	}
}

// pendingTwoStageExecution creates a two-stage Execution held at a before gate
// on its first stage (nothing spawned) and records the human approval, so a
// test-owned engine run dispatches both stages.
func pendingTwoStageExecution(t *testing.T, f *executionRunFixture) (executionComposition, string) {
	t.Helper()
	builtin, _ := workflowdefinition.BuiltinRevision(2)
	d := builtin.Definition
	d.WorkflowID, d.Name, d.Revision = "admission-recheck", "Admission recheck", 1
	d.Stages = d.Stages[:2]
	for i := range d.Stages {
		s := &d.Stages[i]
		s.Checkpoint = true
		s.Validators, s.CompletionCriteria, s.HumanGates = []workflowdefinition.Validator{}, []workflowdefinition.Criterion{}, []workflowdefinition.Gate{}
	}
	return pendingExecutionDefinition(t, f, d)
}

func pendingExecutionDefinition(t *testing.T, f *executionRunFixture, d workflowdefinition.Definition) (executionComposition, string) {
	t.Helper()
	d.Stages[0].Phase, d.Stages[1].Phase = "planning", "implementation"
	d.Stages[0].HumanGates = []workflowdefinition.Gate{{Kind: "planning_authority", Timing: "before"}}
	f.installWorkflowDefinition(t, d)
	t.Setenv("LINGO_INVOCATION_DEFAULTS", `["runtime=codex","model=stub-model","effort=runtime-default"]`)
	first := f.run(t, cli.ExitSuccess, "workflow", "run", "--project", "external", "--repository", "main", "--work-item", "github:owner/repo#7", "--authorize-external")
	if first.Stage.Status != workflow.RunAwaitingGate || len(first.Stage.Stages[0].Tasks[0].Attempts) != 0 {
		t.Fatal(first.Raw)
	}
	service := f.env.service.(lifecycleService)
	ctx := context.Background()
	resolved := service.installation.Resolve(ctx, "external")
	if resolved.Status != local.ResolutionFound {
		t.Fatal(resolved)
	}
	store, err := local.NewExecutionStore(f.env.state)
	if err != nil {
		t.Fatal(err)
	}
	c := executionComposition{s: service, store: store, project: resolved.Project}
	if _, err := c.engine(nil).ApproveGate(ctx, resolved.Project.ID, first.Stage.ExecutionID, d.Stages[0].ID, "human:test"); err != nil {
		t.Fatal(err)
	}
	return c, first.Stage.ExecutionID
}

func attemptsIn(x workflow.Execution, stage int) int {
	return len(x.Stages[stage].Tasks[0].Attempts)
}

// refusal is the bounded public category of a run error ("" when it ran).
func refusal(c executionComposition, err error, x workflow.Execution) string {
	if err == nil {
		return ""
	}
	return c.failure(err, &x).Result.Category
}

// DMH-01: archiving the Project after stage one ran revokes operational
// authority for stage two. The reloaded admission refuses before any attempt
// is recorded; zero processes spawn for stage two. Once the Project is admitted
// again and unchanged, the same Execution dispatches stage two.
func TestExecutionDispatchRechecksProjectAdmissionBetweenStages(t *testing.T) {
	f := newExecutionRunFixture(t)
	c, id := pendingTwoStageExecution(t, f)
	ctx := context.Background()
	launcher, _, err := c.existingLauncher(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	e := c.engine(launcher)
	runner := &countingRunner{base: e.Runner, after: func(n int) {
		if n == 1 {
			operational(t, f.env.state, projectapp.ArchiveProject)
		}
	}}
	e.Runner = runner
	x, err := e.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
	if got := refusal(c, err, x); err == nil || got != "project_admission_revoked" {
		t.Fatalf("archived Project reached stage two: err=%v category=%q status=%s spawns=%d", err, got, x.Status, runner.spawns)
	}
	stored, _ := c.store.Load(ctx, c.project.ID, id)
	if runner.spawns != 1 || stored.Stages[0].Status != workflow.StageCompleted || attemptsIn(stored, 1) != 0 {
		t.Fatalf("spawns=%d stage one=%s stage two attempts=%d", runner.spawns, stored.Stages[0].Status, attemptsIn(stored, 1))
	}
	operational(t, f.env.state, projectapp.ReactivateProject)
	again := c.engine(launcher)
	resumed := &countingRunner{base: again.Runner}
	again.Runner = resumed
	x, err = again.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
	if err != nil || x.Status != workflow.RunCompleted || resumed.spawns != 1 || attemptsIn(x, 1) != 1 {
		t.Fatalf("admitted unchanged Project did not dispatch stage two: %v %s spawns=%d", err, x.Status, resumed.spawns)
	}
}

// DMH-01 check-to-spawn window: revocation after Resolve passed (here during
// Build) is caught by the pre-spawn recheck. The prepared attempt records that
// nothing ran and no process starts.
func TestExecutionDispatchRechecksProjectAdmissionBeforeSpawn(t *testing.T) {
	f := newExecutionRunFixture(t)
	c, id := pendingTwoStageExecution(t, f)
	ctx := context.Background()
	base, x, err := c.existingLauncher(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	second := x.Workflow.Stages[1].ID
	e := c.engine(buildHookRuntimeLauncher{RuntimeLauncher: base, stage: second, hook: func() { operational(t, f.env.state, projectapp.ArchiveProject) }})
	runner := &countingRunner{base: e.Runner}
	e.Runner = runner
	x, err = e.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
	if got := refusal(c, err, x); err == nil || got != "project_admission_revoked" {
		t.Fatalf("revocation in check-to-spawn window: err=%v category=%q status=%s spawns=%d", err, got, x.Status, runner.spawns)
	}
	stored, _ := c.store.Load(ctx, c.project.ID, id)
	a := stored.Stages[1].Tasks[0].Attempts
	if runner.spawns != 1 || len(a) != 1 || a[0].Status != workflow.AttemptFailed || a[0].Diagnostic != "admission_revoked_before_spawn" || a[0].Process != nil {
		t.Fatalf("spawns=%d attempts=%+v", runner.spawns, a)
	}
}

// DMH-01 rebind: if the Project now binds the repository at another local
// path than the one the launcher froze, dispatch refuses instead of silently
// retargeting; nothing is recorded or spawned.
func TestExecutionDispatchRefusesRebindRetarget(t *testing.T) {
	f := newExecutionRunFixture(t)
	c, id := pendingTwoStageExecution(t, f)
	ctx := context.Background()
	launcher, _, err := c.existingLauncher(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	stale := c
	stale.project.Repositories = append([]local.ResolvedRepository(nil), c.project.Repositories...)
	for i := range stale.project.Repositories {
		stale.project.Repositories[i].Path += "-previous-binding"
	}
	e := stale.engine(launcher)
	runner := &countingRunner{base: e.Runner}
	e.Runner = runner
	before, _ := c.store.Load(ctx, c.project.ID, id)
	x, err := e.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
	if got := refusal(c, err, x); got != "repository_binding_changed" {
		t.Fatalf("rebind retargeted: err=%v category=%q status=%s spawns=%d", err, got, x.Status, runner.spawns)
	}
	after, _ := c.store.Load(ctx, c.project.ID, id)
	if runner.spawns != 0 || attemptsIn(after, 0) != 0 || after.Revision != before.Revision+1 {
		t.Fatalf("refusal recorded or spawned: spawns=%d attempts=%d", runner.spawns, attemptsIn(after, 0))
	}
}

// Rebind the actual installation after stage one. The frozen composition and
// launcher must never follow the replacement workspace for stage two.
func TestExecutionDispatchRefusesPersistedRebindBetweenStages(t *testing.T) {
	f := newExecutionRunFixture(t)
	c, id := pendingTwoStageExecution(t, f)
	ctx := context.Background()
	launcher, _, err := c.existingLauncher(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	e := c.engine(launcher)
	runner := &countingRunner{base: e.Runner, after: func(n int) {
		if n != 1 {
			return
		}
		observed, category := c.s.installation.Inspect(ctx, c.project.ID)
		if category != "" {
			t.Fatal(category)
		}
		state := observed.Record.State()
		// A fixture-owned valid installation rebind, using the canonical codec.
		state.Repositories[0].ExplicitPath = t.TempDir()
		identity, err := local.DirectoryIdentity(state.Repositories[0].ExplicitPath)
		if err != nil {
			t.Fatal(err)
		}
		state.Repositories[0].CanonicalIdentity = identity
		record, issues := local.NewRecord(state)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		wire, issues := local.EncodeRecord(record)
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		if err := os.WriteFile(filepath.Join(f.env.state, "projects", c.project.ID, "installation.json"), wire, 0600); err != nil {
			t.Fatal(err)
		}
	}}
	e.Runner = runner
	x, err := e.Run(ctx, c.project.ID, id, stageexecution.RunOptions{})
	if err == nil || refusal(c, err, x) != "repository_binding_changed" {
		t.Fatalf("rebind allowed: %v %s", err, refusal(c, err, x))
	}
	stored, err := c.store.Load(ctx, c.project.ID, id)
	if err != nil || runner.spawns != 1 || stored.Stages[0].Status != workflow.StageCompleted || attemptsIn(stored, 1) != 0 {
		t.Fatalf("next stage spawned: %v spawns=%d", err, runner.spawns)
	}
}
