package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/workflow"
)

const storeTestExecutionID = "00000000-0000-4000-8000-000000000042"

var storeT0 = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

func storeTestExecution(t *testing.T) workflow.Execution {
	t.Helper()
	e, err := workflow.NewExecution(workflow.NewExecutionInput{
		ExecutionID: storeTestExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"},
		Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/app"}},
		WorkflowID:   "minimal", Revision: "r1", Now: storeT0,
		Stages: []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder"}}, TimeoutSeconds: 60}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func newTestExecutionStore(t *testing.T) (ExecutionStore, string) {
	t.Helper()
	root := privateTestRoot(t)
	s, err := NewExecutionStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return s, root
}

func claim(t *testing.T, s ExecutionStore) *RunLease {
	t.Helper()
	ctx := context.Background()
	lease, err := s.AcquireRunner(ctx, "proj", storeTestExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		return e.ClaimRunner(lease.Epoch, storeT0)
	}); err != nil {
		t.Fatal(err)
	}
	return lease
}

func TestExecutionStoreCreateLoadAndNamespace(t *testing.T) {
	s, root := newTestExecutionStore(t)
	ctx := context.Background()
	e := storeTestExecution(t)
	if err := s.Create(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, e); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate create: %v", err)
	}
	got, err := s.Load(ctx, "proj", storeTestExecutionID)
	if err != nil || got.Revision != 1 || got.Workflow.Digest != e.Workflow.Digest {
		t.Fatalf("load: %v %+v", err, got)
	}
	path := filepath.Join(root, "executions", "v2", "proj", storeTestExecutionID+".json")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("record not private at %s: %v", path, err)
	}
	for _, legacy := range []string{"graphs", "single-stage-runs"} {
		if _, err := os.Stat(filepath.Join(root, legacy)); !os.IsNotExist(err) {
			t.Fatalf("new run created legacy %s state", legacy)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "executions", "v1")); !os.IsNotExist(err) {
		t.Fatal("new run created an executions/v1 ledger")
	}
	if _, err := s.Load(ctx, "../proj", storeTestExecutionID); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("path escape: %v", err)
	}
	if _, err := s.Load(ctx, "proj", "../../00000000-0000-4000-8000-0000000"); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("id escape: %v", err)
	}
}

func TestExecutionStoreRejectsSymlinkedNamespace(t *testing.T) {
	s, root := newTestExecutionStore(t)
	outside := privateTestRoot(t)
	if err := os.Mkdir(filepath.Join(root, "executions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "executions", "v2")); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(context.Background(), storeTestExecution(t)); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("symlinked executions/v2 accepted: %v", err)
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("write escaped through symlink")
	}
}

func TestExecutionStoreRefusesLegacyBytesInNewNamespace(t *testing.T) {
	s, root := newTestExecutionStore(t)
	ctx := context.Background()
	if err := s.Create(ctx, storeTestExecution(t)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "executions", "v2", "proj", storeTestExecutionID+".json")
	if err := os.WriteFile(path, []byte(`{"formatVersion":5,"executionId":"`+storeTestExecutionID+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(ctx, "proj", storeTestExecutionID); !errors.Is(err, workflow.ErrLegacyExecution) {
		t.Fatalf("old resume not refused: %v", err)
	}
	if _, err := s.AcquireRunner(ctx, "proj", storeTestExecutionID); !errors.Is(err, workflow.ErrLegacyExecution) {
		t.Fatalf("runner acquired over legacy bytes: %v", err)
	}
}

func TestExecutionStoreRevisionCASAndSingleRunner(t *testing.T) {
	s, _ := newTestExecutionStore(t)
	ctx := context.Background()
	if err := s.Create(ctx, storeTestExecution(t)); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	defer lease.Close()
	if _, err := s.AcquireRunner(ctx, "proj", storeTestExecutionID); !errors.Is(err, ErrConflict) {
		t.Fatalf("second runner acquired: %v", err)
	}
	current, _ := s.Load(ctx, "proj", storeTestExecutionID)
	stale := current
	if err := current.RequestCancel("user:local", storeT0); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, current, current.Revision-1, nil); err != nil {
		t.Fatal(err)
	}
	stale.Stages = cloneExecutionStages(stale.Stages)
	if err := stale.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Epoch,
		Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: storeT0}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, stale, stale.Revision-1, lease); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale writer overwrote cancellation: %v", err)
	}
	got, _ := s.Load(ctx, "proj", storeTestExecutionID)
	if got.Cancel == nil || len(got.Stages[0].Tasks[0].Attempts) != 0 {
		t.Fatal("losing CAS changed state")
	}
}

func TestExecutionStoreReleasedOrSupersededLeaseCannotCommit(t *testing.T) {
	s, _ := newTestExecutionStore(t)
	ctx := context.Background()
	if err := s.Create(ctx, storeTestExecution(t)); err != nil {
		t.Fatal(err)
	}
	old := claim(t, s)
	old.Close()
	next := claim(t, s)
	defer next.Close()
	if next.Epoch <= old.Epoch {
		t.Fatalf("generation not monotonic: %d then %d", old.Epoch, next.Epoch)
	}
	_, err := s.Update(ctx, "proj", storeTestExecutionID, old, func(e *workflow.Execution) error {
		e.RunnerGeneration = old.Epoch
		return nil
	})
	if err == nil {
		t.Fatal("released stale runner committed")
	}
	_, err = s.Update(ctx, "proj", storeTestExecutionID, nil, func(e *workflow.Execution) error {
		return e.ClaimRunner(e.RunnerGeneration+1, storeT0)
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("lease-less generation change accepted: %v", err)
	}
}

func TestExecutionStoreConcurrentWritersSerializeWithoutLoss(t *testing.T) {
	s, _ := newTestExecutionStore(t)
	ctx := context.Background()
	e, err := workflow.NewExecution(workflow.NewExecutionInput{
		ExecutionID: storeTestExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"},
		Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "id"}}, WorkflowID: "gates", Revision: "r1", Now: storeT0,
		Stages: []workflow.StageSpec{
			{ID: "a", Agents: []workflow.AgentSpec{{ID: "x"}}, HumanGate: true, TimeoutSeconds: 1},
			{ID: "b", Agents: []workflow.AgentSpec{{ID: "y"}}, HumanGate: true, TimeoutSeconds: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, e); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	for _, stage := range []string{"a", "b"} {
		ref := workflow.TaskRef{StageID: stage, AgentID: map[string]string{"a": "x", "b": "y"}[stage]}
		if _, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
			if err := e.PrepareAttempt(workflow.PrepareInput{Ref: ref, AttemptID: "att-" + stage, Generation: lease.Epoch, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: storeT0}); err != nil {
				return err
			}
			if err := e.MarkStarted("att-"+stage, lease.Epoch, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0); err != nil {
				return err
			}
			return e.CompleteAttempt("att-"+stage, lease.Epoch, workflow.AttemptResult{Now: storeT0})
		}); err != nil {
			t.Fatal(err)
		}
	}
	lease.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, stage := range []string{"a", "b"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Update(ctx, "proj", storeTestExecutionID, nil, func(e *workflow.Execution) error {
				return e.ApproveGate(stage, "human:maintainer", storeT0)
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, _ := s.Load(ctx, "proj", storeTestExecutionID)
	if got.Stages[0].Gate == nil || got.Stages[1].Gate == nil || got.Status != workflow.RunCompleted {
		t.Fatalf("concurrent update lost: %+v", got.Stages)
	}
}

func TestExecutionStoreCrashWindowsNeverInventState(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		stage     FaultStage
		interrupt bool
		want      string
	}{
		{FaultF2, false, "prior"},
		{FaultF2, true, "recovery"},
		{FaultF4, true, "recovery"},
		{FaultF6, false, "recovery"},
	} {
		s, _ := newTestExecutionStore(t)
		if err := s.Create(ctx, storeTestExecution(t)); err != nil {
			t.Fatal(err)
		}
		current, _ := s.Load(ctx, "proj", storeTestExecutionID)
		next := current
		_ = next.RequestCancel("user:local", storeT0)
		fault := errors.New("injected")
		if tc.interrupt {
			fault = ErrSimulatedInterruption
		}
		s.hooks.fault = func(stage FaultStage) error {
			if stage == tc.stage {
				return fault
			}
			return nil
		}
		if err := s.Save(ctx, next, current.Revision, nil); err == nil {
			t.Fatalf("%s: fault not reported", tc.stage)
		}
		s.hooks.fault = nil
		got, err := s.Load(ctx, "proj", storeTestExecutionID)
		switch tc.want {
		case "prior":
			if err != nil || got.Cancel != nil {
				t.Fatalf("%s: pre-commit failure changed state: %v", tc.stage, err)
			}
		case "recovery":
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("%s interrupt=%t: got %v, want recovery_required", tc.stage, tc.interrupt, err)
			}
		}
	}
}

func storeInvocation() workflow.Invocation {
	return workflow.Invocation{Runtime: "stub", Executable: "stub-bin", Model: "m", Effort: "runtime-default", Instructions: "i", InstructionsDigest: workflow.DigestText([]byte("i")),
		Provenance: []workflow.FieldProvenance{{Field: "runtime", Source: "local_default"}, {Field: "model", Source: "local_default"}, {Field: "effort", Source: "local_default"}, {Field: "instructions", Source: "workflow_agent"}}}
}
