//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

type taskLauncher struct{ stubLauncher }

func (l taskLauncher) Build(ctx context.Context, r LaunchRequest) (executiongraph.Invocation, error) {
	i, err := l.stubLauncher.Build(ctx, r)
	i.Argv = append(i.Argv, r.Ref.AgentID)
	return i, err
}

type observedWriter struct {
	ExecutionStore
	active, maximum atomic.Int32
	stale           atomic.Bool
}

func (s *observedWriter) UpdateFenced(ctx context.Context, project, execution string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	if fence != nil && s.stale.Load() {
		return workflow.Execution{}, workflow.ErrStaleRunner
	}
	n := s.active.Add(1)
	defer s.active.Add(-1)
	for old := s.maximum.Load(); n > old && !s.maximum.CompareAndSwap(old, n); old = s.maximum.Load() {
	}
	return s.ExecutionStore.UpdateFenced(ctx, project, execution, fence, mutate)
}

func parallelStages(gate bool) []workflow.StageSpec {
	return []workflow.StageSpec{
		{ID: "work", Agents: []workflow.AgentSpec{{ID: "a"}, {ID: "b"}, {ID: "c", DependsOn: []string{"a"}}}, HumanGate: gate, TimeoutSeconds: 5},
		{ID: "next", DependsOn: []string{"work"}, Agents: []workflow.AgentSpec{{ID: "next"}}, TimeoutSeconds: 5},
	}
}

func awaitTask(t *testing.T, started <-chan string) string {
	t.Helper()
	select {
	case id := <-started:
		return id
	case <-time.After(3 * time.Second):
		t.Fatal("ready tasks did not start concurrently")
		return ""
	}
}

func TestEngineParallelReadyTasksStreamAndCommitWithoutLoss(t *testing.T) {
	started := make(chan string, 4)
	releaseA, releaseB := make(chan struct{}), make(chan struct{})
	// A barrier forces both independent tasks to be running at once. A's
	// validated completion releases C while B is still running, proving that
	// scheduling does not wait for a whole batch.
	f := newEngineFixture(t, parallelStages(false), func(ctx context.Context, inv executiongraph.Invocation, commit func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		id := inv.Argv[len(inv.Argv)-1]
		if err := commit(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
			return executiongraph.ProcessResult{OutcomeUnknown: true}
		}
		started <- id
		var release <-chan struct{}
		switch id {
		case "a":
			release = releaseA
		case "b":
			release = releaseB
		default:
			return executiongraph.ProcessResult{ExitCode: 0}
		}
		select {
		case <-release:
			return executiongraph.ProcessResult{ExitCode: 0}
		case <-ctx.Done():
			return executiongraph.ProcessResult{Stopped: true}
		}
	})
	f.engine.Launcher = taskLauncher{}
	writer := &observedWriter{ExecutionStore: f.engine.Store}
	f.engine.Store = writer
	type result struct {
		x   workflow.Execution
		err error
	}
	done := make(chan result, 1)
	go func() { x, err := f.run(t); done <- result{x, err} }()
	seen := map[string]bool{awaitTask(t, started): true, awaitTask(t, started): true}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("dependencies did not block c/next: %v", seen)
	}
	close(releaseA)
	if id := awaitTask(t, started); id != "c" {
		t.Fatalf("expected c while b is still running, got %s", id)
	}
	close(releaseB)
	if id := awaitTask(t, started); id != "next" {
		t.Fatalf("expected dependent stage, got %s", id)
	}
	r := <-done
	if r.err != nil || r.x.Status != workflow.RunCompleted || writer.maximum.Load() != 1 || f.runner.spawns.Load() != 4 {
		t.Fatalf("parallel result: %v %s writer-concurrency=%d spawns=%d", r.err, r.x.Status, writer.maximum.Load(), f.runner.spawns.Load())
	}
	for _, stage := range r.x.Stages {
		for _, task := range stage.Tasks {
			if len(task.Attempts) != 1 || task.Attempts[0].Status != workflow.AttemptSucceeded || task.Attempts[0].Process == nil {
				t.Fatalf("independent completion lost: %+v", task)
			}
		}
	}
	stored, err := f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
	if err != nil || stored.Revision != r.x.Revision || stored.Status != workflow.RunCompleted {
		t.Fatalf("returned snapshot is not canonical final state: %v %+v", err, stored)
	}
}

func TestEngineParallelStopsDependentWork(t *testing.T) {
	for _, mode := range []string{"gate", "cancel", "interrupt", "unknown", "missing-output", "stale"} {
		t.Run(mode, func(t *testing.T) {
			started, release := make(chan string, 2), make(chan struct{})
			stages := parallelStages(mode == "gate")
			stages[0].Agents = stages[0].Agents[:2]
			if mode == "missing-output" {
				stages[0].Outputs = []string{"result"}
				stages[0].Agents[0].OutputScope = true
				stages[0].Agents[0].Outputs = []string{"result"}
				stages[0].Agents[1].OutputScope = true
			}
			f := newEngineFixture(t, stages, func(ctx context.Context, inv executiongraph.Invocation, commit func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
				if err := commit(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
					return executiongraph.ProcessResult{OutcomeUnknown: true}
				}
				started <- inv.Argv[len(inv.Argv)-1]
				select {
				case <-release:
				case <-ctx.Done():
				}
				if mode == "unknown" {
					return executiongraph.ProcessResult{OutcomeUnknown: true}
				}
				if mode == "cancel" || mode == "interrupt" {
					return executiongraph.ProcessResult{Stopped: true}
				}
				return executiongraph.ProcessResult{ExitCode: 0}
			})
			f.engine.Launcher = taskLauncher{}
			writer := &observedWriter{ExecutionStore: f.engine.Store}
			f.engine.Store = writer
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			type result struct {
				x   workflow.Execution
				err error
			}
			done := make(chan result, 1)
			go func() { x, err := f.engine.Run(ctx, "proj", engineExecutionID, RunOptions{}); done <- result{x, err} }()
			awaitTask(t, started)
			awaitTask(t, started)
			if mode == "cancel" {
				if _, err := f.engine.Cancel(context.Background(), "proj", engineExecutionID, "human:test"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "interrupt" {
				cancel()
			}
			if mode == "stale" {
				writer.stale.Store(true)
			}
			close(release)
			var r result
			select {
			case r = <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("principal stranded worker commits")
			}
			if f.runner.spawns.Load() != 2 || len(r.x.Stages[1].Tasks[0].Attempts) != 0 {
				t.Fatalf("%s dispatched dependent work: spawns=%d %+v", mode, f.runner.spawns.Load(), r.x)
			}
			want := map[string]workflow.RunStatus{"gate": workflow.RunAwaitingGate, "cancel": workflow.RunCancelled, "interrupt": workflow.RunCancelled, "unknown": workflow.RunRecoveryRequired, "missing-output": workflow.RunFailed}
			if mode == "stale" {
				if !errors.Is(r.err, workflow.ErrStaleRunner) {
					t.Fatalf("stale writer: %v", r.err)
				}
			} else if r.x.Status != want[mode] || mode != "interrupt" && r.err != nil || mode == "interrupt" && !errors.Is(r.err, context.Canceled) {
				t.Fatalf("%s: %s %v", mode, r.x.Status, r.err)
			}
			if mode != "stale" {
				for _, task := range r.x.Stages[0].Tasks {
					if a := task.Attempts[0]; a.Status == workflow.AttemptRunning || a.Status == workflow.AttemptPrepared {
						t.Fatalf("lost terminal observation: %+v", a)
					}
				}
			}
		})
	}
}

func TestEngineParallelDependencyDefinitionsRefuseBeforeSpawn(t *testing.T) {
	for _, mode := range []string{"missing-agent", "cycle-agent", "missing-stage", "cycle-stage"} {
		t.Run(mode, func(t *testing.T) {
			stages := parallelStages(false)
			switch mode {
			case "missing-agent":
				stages[0].Agents[0].DependsOn = []string{"missing"}
			case "cycle-agent":
				stages[0].Agents[0].DependsOn = []string{"b"}
				stages[0].Agents[1].DependsOn = []string{"a"}
			case "missing-stage":
				stages[0].DependsOn = []string{"missing"}
			case "cycle-stage":
				stages[0].DependsOn = []string{"next"}
			}
			_, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: engineExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"}, Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "example/app"}}, WorkflowID: "parallel", Revision: "r1", Stages: stages, Now: time.Now()})
			if err == nil {
				t.Fatal("invalid dependency admitted by canonical constructor")
			}
		})
	}
}
