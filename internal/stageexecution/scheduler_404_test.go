//go:build linux || darwin

package stageexecution

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
)

// occupancy404 observes durable facts and worker lifetimes independently of the
// admission implementation. A terminal worker stays occupied at its barrier.
type occupancy404 struct {
	ExecutionStore
	mu        sync.Mutex
	workers   map[string]bool
	peak      int
	overflow  bool
	saturated chan workflow.Execution
}

func (s *occupancy404) observe(x workflow.Execution) {
	occupied := map[string]bool{}
	for id := range s.workers {
		occupied[id] = true
	}
	for _, stage := range x.Stages {
		if stage.StageID != "work" {
			continue
		}
		for _, task := range stage.Tasks {
			for _, a := range task.Attempts {
				if a.Status == workflow.AttemptPrepared || a.Status == workflow.AttemptRunning {
					occupied[a.AttemptID] = true
				}
			}
		}
	}
	if len(occupied) > s.peak {
		s.peak = len(occupied)
	}
	s.overflow = s.overflow || len(occupied) > 2
}

func (s *occupancy404) UpdateFenced(ctx context.Context, project, execution string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	return s.ExecutionStore.UpdateFenced(ctx, project, execution, fence, func(x *workflow.Execution) error {
		err := mutate(x)
		s.mu.Lock()
		s.observe(*x)
		s.mu.Unlock()
		if errors.Is(err, workflow.ErrStageCapacity) {
			snapshot, copyErr := executionSnapshot(*x)
			if copyErr != nil {
				return copyErr
			}
			select {
			case s.saturated <- snapshot:
			default:
			}
		}
		return err
	})
}

func TestI404SchedulerTerminalCommitRetainsSlotUntilWorkerFinished(t *testing.T) {
	capacity := 2
	stages := []workflow.StageSpec{{ID: "work", Concurrency: &capacity, TimeoutSeconds: 30, Agents: []workflow.AgentSpec{
		{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "join", DependsOn: []string{"a", "b", "c"}},
	}}}
	started := make(chan string, 4)
	releaseA, releaseB, releaseC := make(chan struct{}), make(chan struct{}), make(chan struct{})
	terminalA, finishA := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	f := newEngineFixture(t, stages, func(ctx context.Context, inv executiongraph.Invocation, commit func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		id := inv.Argv[len(inv.Argv)-1]
		if err := commit(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:" + id, GroupIdentity: "777"}); err != nil {
			return executiongraph.ProcessResult{OutcomeUnknown: true}
		}
		started <- id
		var release <-chan struct{}
		switch id {
		case "a":
			release = releaseA
		case "b":
			release = releaseB
		case "c":
			release = releaseC
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
	observer := &occupancy404{ExecutionStore: f.engine.Store, workers: map[string]bool{}, saturated: make(chan workflow.Execution, 8)}
	f.engine.Store = observer
	lease, current, err := f.engine.Claim(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	type result struct {
		x   workflow.Execution
		err error
	}
	done := make(chan result, 1)
	go func() {
		x, err := f.engine.runReadyWithWorker(ctx, lease, current, RunOptions{}, func(worker Engine, request LaunchRequest) error {
			observer.mu.Lock()
			observer.workers[request.Attempt.AttemptID] = true
			observer.observe(request.Execution)
			observer.mu.Unlock()
			_, err := worker.runAttempt(ctx, lease, request)
			if request.Ref.AgentID == "a" {
				close(terminalA)
				select {
				case <-finishA:
				case <-ctx.Done():
				}
			}
			observer.mu.Lock()
			delete(observer.workers, request.Attempt.AttemptID)
			observer.mu.Unlock()
			return err
		})
		done <- result{x, err}
	}()
	seen := map[string]bool{awaitTask(t, started): true, awaitTask(t, started): true}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("roots not overlapped: %v", seen)
	}
	x, err := f.engine.Store.Load(ctx, "proj", engineExecutionID)
	if err != nil || len(x.Stages[0].Tasks[2].Attempts) != 0 || len(x.Stages[0].Tasks[3].Attempts) != 0 {
		t.Fatalf("third/join admitted at capacity: %v %+v", err, x.Stages)
	}
	close(releaseA)
	select {
	case <-terminalA:
	case <-ctx.Done():
		t.Fatal("A terminal barrier not reached")
	}
	// This is a principal-side fenced admission checkpoint, while A is held
	// after its real terminal commit and B is still at the runner barrier.
	select {
	case x = <-observer.saturated:
		if x.Stages[0].Tasks[0].Attempts[0].Status != workflow.AttemptSucceeded || x.Stages[0].Tasks[1].Attempts[0].Status != workflow.AttemptRunning || len(x.Stages[0].Tasks[2].Attempts) != 0 {
			t.Fatalf("wrong terminal window: %+v", x.Stages)
		}
	case <-ctx.Done():
		t.Fatal("principal did not recheck capacity during terminal window")
	}
	close(finishA)
	if id := awaitTask(t, started); id != "c" {
		t.Fatalf("genuine returned slot did not replenish third root: %s", id)
	}
	x, err = f.engine.Store.Load(ctx, "proj", engineExecutionID)
	if err != nil || x.Stages[0].Tasks[1].Attempts[0].Status != workflow.AttemptRunning || len(x.Stages[0].Tasks[3].Attempts) != 0 {
		t.Fatalf("peer/join controls lost: %v %+v", err, x.Stages)
	}
	close(releaseC)
	close(releaseB)
	if id := awaitTask(t, started); id != "join" {
		t.Fatalf("join missing: %s", id)
	}
	select {
	case r := <-done:
		if r.err != nil || r.x.Status != workflow.RunCompleted || f.runner.spawns.Load() != 4 {
			t.Fatalf("final: %s %v starts=%d", r.x.Status, r.err, f.runner.spawns.Load())
		}
	case <-ctx.Done():
		t.Fatal("workers did not drain")
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if observer.peak != 2 || observer.overflow {
		t.Fatalf("peak union=%d overflow=%v", observer.peak, observer.overflow)
	}
}

// corruptSnapshot404 changes only the returned read-only worker snapshot,
// after the real store has durably prepared the attempt.
type corruptSnapshot404 struct {
	ExecutionStore
	injected bool
}

func (s *corruptSnapshot404) UpdateFenced(ctx context.Context, project, execution string, fence workflow.RunnerFence, mutate func(*workflow.Execution) error) (workflow.Execution, error) {
	x, err := s.ExecutionStore.UpdateFenced(ctx, project, execution, fence, mutate)
	if err == nil && !s.injected {
		for _, stage := range x.Stages {
			for _, task := range stage.Tasks {
				for _, a := range task.Attempts {
					if a.Status == workflow.AttemptPrepared {
						s.injected = true
						x.Workflow.Digest = "corrupt-return-only"
						return x, nil
					}
				}
			}
		}
	}
	return x, err
}
func TestI404SchedulerSnapshotFailureKeepsDurableOccupancyAndNeverSpawns(t *testing.T) {
	capacity := 2
	stages := []workflow.StageSpec{{ID: "work", Concurrency: &capacity, TimeoutSeconds: 30, Agents: []workflow.AgentSpec{{ID: "a"}, {ID: "b"}, {ID: "c"}}}}
	f := newEngineFixture(t, stages, succeed(""))
	f.engine.Store = &corruptSnapshot404{ExecutionStore: f.engine.Store}
	if _, err := f.run(t); err == nil || f.runner.spawns.Load() != 0 {
		t.Fatalf("snapshot failure spawned: %v starts=%d", err, f.runner.spawns.Load())
	}
	x, err := f.engine.Store.Load(context.Background(), "proj", engineExecutionID)
	if err != nil || x.Stages[0].Tasks[0].Attempts[0].Status != workflow.AttemptPrepared || len(x.Stages[0].Tasks[1].Attempts) != 0 {
		t.Fatalf("durable prepared fact lost: %v %+v", err, x.Stages)
	}
	x, err = f.run(t)
	if err != nil || x.Status != workflow.RunRecoveryRequired || x.Stages[0].Tasks[0].Attempts[0].Status != workflow.AttemptUnknown || f.runner.spawns.Load() != 0 {
		t.Fatalf("snapshot failure replayed: %s %v starts=%d", x.Status, err, f.runner.spawns.Load())
	}
}

type materializeFailure404 struct {
	IntegrationGit
	calls int
	base  string
}

func (s *materializeFailure404) ObserveBase(context.Context) (string, error)         { return s.base, nil }
func (s *materializeFailure404) PrepareTarget(context.Context, string, string) error { return nil }
func (s *materializeFailure404) ObserveTarget(context.Context, string) (string, error) {
	return s.base, nil
}
func (s *materializeFailure404) MaterializeWorkspace(context.Context, string, string, string) (string, error) {
	s.calls++
	return "", errors.New("injected workspace materialization failure")
}
func TestI404SchedulerMaterializationFailureIsUnknownWithoutSpawnOrReplay(t *testing.T) {
	f, store := startFixture(t, "")
	capacity := 2
	request := linkedRequest()
	request.WorkflowID, request.WorkflowRevision, request.EnrollIntegration = "workspace", "r1", true
	effects := []string{"workspace_write", "validation_command"}
	request.Stages = []workflow.StageSpec{{ID: "work", Concurrency: &capacity, TimeoutSeconds: 30, Commands: checkCommand("exit 0", 10), Agents: []workflow.AgentSpec{{ID: "a", Effects: effects}, {ID: "b", Effects: effects}, {ID: "c", Effects: effects}}}}
	git := &materializeFailure404{base: "0123456789012345678901234567890123456789"}
	f.engine.Workspaces = func(context.Context, workflow.Execution, string) (WorkspaceGit, error) { return git, nil }
	if _, err := f.engine.Start(context.Background(), store, request); err == nil {
		t.Fatal("materialization fault not returned")
	}
	x, err := store.Load(context.Background(), "proj", startExecutionID)
	if err != nil || x.Status != workflow.RunRecoveryRequired || x.Stages[0].Tasks[0].Attempts[0].Status != workflow.AttemptUnknown || len(x.Stages[0].Tasks[1].Attempts) != 0 || f.runner.spawns.Load() != 0 || git.calls != 1 {
		t.Fatalf("untruthful materialization outcome: %v %+v starts=%d calls=%d", err, x.Stages, f.runner.spawns.Load(), git.calls)
	}
	x, err = f.engine.Run(context.Background(), "proj", startExecutionID, RunOptions{})
	if err != nil || x.Status != workflow.RunRecoveryRequired || f.runner.spawns.Load() != 0 || git.calls != 1 {
		t.Fatalf("materialization replay: %s %v starts=%d calls=%d", x.Status, err, f.runner.spawns.Load(), git.calls)
	}
}

type buildBarrier404 struct {
	taskLauncher
	entered chan struct{}
	release <-chan struct{}
}

func (s buildBarrier404) Build(ctx context.Context, request LaunchRequest) (executiongraph.Invocation, error) {
	if request.Ref.AgentID == "a" {
		close(s.entered)
		select {
		case <-s.release:
			return executiongraph.Invocation{}, errors.New("injected build failure")
		case <-ctx.Done():
			return executiongraph.Invocation{}, ctx.Err()
		}
	}
	return s.taskLauncher.Build(ctx, request)
}
func TestI404SchedulerBuildFailureStopsReplenishmentAndDrainsPeer(t *testing.T) {
	capacity := 2
	stages := []workflow.StageSpec{{ID: "work", Concurrency: &capacity, TimeoutSeconds: 30, Agents: []workflow.AgentSpec{{ID: "a"}, {ID: "b"}, {ID: "c"}}}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	started := make(chan string, 3)
	releaseBuild, releasePeer, finishA := make(chan struct{}), make(chan struct{}), make(chan struct{})
	buildEntered, terminalA := make(chan struct{}), make(chan struct{})
	f := newEngineFixture(t, stages, func(ctx context.Context, inv executiongraph.Invocation, commit func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
		if err := commit(&executiongraph.ProcessCorrelation{PID: 777, StartIdentity: "boot:1", GroupIdentity: "777"}); err != nil {
			return executiongraph.ProcessResult{OutcomeUnknown: true}
		}
		started <- inv.Argv[len(inv.Argv)-1]
		select {
		case <-releasePeer:
			return executiongraph.ProcessResult{ExitCode: 0}
		case <-ctx.Done():
			return executiongraph.ProcessResult{Stopped: true}
		}
	})
	f.engine.Launcher = buildBarrier404{entered: buildEntered, release: releaseBuild}
	lease, current, err := f.engine.Claim(ctx, "proj", engineExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	type result struct {
		x   workflow.Execution
		err error
	}
	done := make(chan result, 1)
	go func() {
		x, err := f.engine.runReadyWithWorker(ctx, lease, current, RunOptions{}, func(worker Engine, request LaunchRequest) error {
			_, err := worker.runAttempt(ctx, lease, request)
			if request.Ref.AgentID == "a" {
				close(terminalA)
				select {
				case <-finishA:
				case <-ctx.Done():
				}
			}
			return err
		})
		done <- result{x, err}
	}()
	select {
	case <-buildEntered:
	case <-ctx.Done():
		t.Fatal("build barrier missing")
	}
	if id := awaitTask(t, started); id != "b" {
		t.Fatalf("peer not admitted with failed build held: %s", id)
	}
	close(releaseBuild)
	select {
	case <-terminalA:
	case <-ctx.Done():
		t.Fatal("failure terminal barrier missing")
	}
	x, err := f.engine.Store.Load(ctx, "proj", engineExecutionID)
	if err != nil || x.Stages[0].Tasks[0].Attempts[0].Status != workflow.AttemptFailed || x.Stages[0].Tasks[1].Attempts[0].Status != workflow.AttemptRunning || len(x.Stages[0].Tasks[2].Attempts) != 0 {
		t.Fatalf("failure bypassed peer/drain controls: %v %+v", err, x.Stages)
	}
	close(finishA)
	close(releasePeer)
	select {
	case r := <-done:
		if r.err != nil || r.x.Status != workflow.RunFailed || r.x.Stages[0].Tasks[1].Attempts[0].Status != workflow.AttemptSucceeded || len(r.x.Stages[0].Tasks[2].Attempts) != 0 || f.runner.spawns.Load() != 1 {
			t.Fatalf("failure did not drain peer: %s %v starts=%d", r.x.Status, r.err, f.runner.spawns.Load())
		}
	case <-ctx.Done():
		t.Fatal("failed build stranded peer")
	}
}
