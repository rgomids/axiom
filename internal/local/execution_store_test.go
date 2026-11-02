package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// DMH-02: a runner fence is bound to the state root that issued it. Two
// private roots holding independently valid same-generation leases for the
// same logical Execution must never let B's lease commit A's record, whether or
// not A's own runner lease is held. Refusal writes nothing.
func TestExecutionStoreRejectsForeignStateRootLease(t *testing.T) {
	ctx := context.Background()
	a, rootA := newTestExecutionStore(t)
	b, _ := newTestExecutionStore(t)
	for _, s := range []ExecutionStore{a, b} {
		if err := s.Create(ctx, storeTestExecution(t)); err != nil {
			t.Fatal(err)
		}
	}
	record := filepath.Join(rootA, "executions", "v2", "proj", storeTestExecutionID+".json")
	attempt := func(e *workflow.Execution, gen uint64) error {
		return e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "foreign", Generation: gen,
			Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: []string{}, Actor: "user:local"}, Now: storeT0})
	}
	refuses := func(name string, foreign *RunLease) {
		t.Helper()
		before, err := os.ReadFile(record)
		if err != nil {
			t.Fatal(err)
		}
		current, _ := a.Load(ctx, "proj", storeTestExecutionID)
		next := current
		next.Stages = cloneExecutionStages(current.Stages)
		if err := attempt(&next, foreign.Epoch); err != nil {
			t.Fatal(err)
		}
		if err := a.Save(ctx, next, current.Revision, foreign); !errors.Is(err, workflow.ErrStaleRunner) {
			t.Fatalf("%s: foreign-root lease admitted by Save: %v", name, err)
		}
		if _, err := a.Update(ctx, "proj", storeTestExecutionID, foreign, func(e *workflow.Execution) error { return attempt(e, foreign.Epoch) }); !errors.Is(err, workflow.ErrStaleRunner) {
			t.Fatalf("%s: foreign-root lease admitted by Update: %v", name, err)
		}
		after, err := os.ReadFile(record)
		if err != nil || string(before) != string(after) {
			t.Fatalf("%s: refused foreign lease mutated A: %v", name, err)
		}
	}
	// A's runner holds its own lease at generation 1; B's lease has the same epoch.
	leaseA := claim(t, a)
	leaseB := claim(t, b)
	if leaseA.Epoch != leaseB.Epoch {
		t.Fatalf("fixture needs equal generations: %d vs %d", leaseA.Epoch, leaseB.Epoch)
	}
	refuses("A lease held", leaseB)
	// A released: B's still-valid lease must not take over A either.
	leaseA.Close()
	refuses("A lease released", leaseB)
	leaseB.Close()
	// A deleted/never-issued foreign root lease from a third root is refused too.
	foreignStore, err := NewRunLeaseStore(privateTestRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := foreignStore.Acquire(ctx, "proj", storeTestExecutionID, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer foreign.Close()
	current, _ := a.Load(ctx, "proj", storeTestExecutionID)
	if foreign.Epoch != current.RunnerGeneration+1 {
		t.Fatalf("third-root epoch %d", foreign.Epoch)
	}
	if _, err := a.Update(ctx, "proj", storeTestExecutionID, foreign, func(e *workflow.Execution) error { return e.ClaimRunner(foreign.Epoch, storeT0) }); !errors.Is(err, workflow.ErrStaleRunner) {
		t.Fatalf("third-root lease claimed A: %v", err)
	}
	// Positive: A's own lease commits.
	own := claim(t, a)
	defer own.Close()
	if _, err := a.Update(ctx, "proj", storeTestExecutionID, own, func(e *workflow.Execution) error { return attempt(e, own.Epoch) }); err != nil {
		t.Fatalf("own-root lease refused: %v", err)
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

// An interrupted canonical publication must be visible to, and finalized by,
// the existing reviewed maintenance recovery; otherwise the Execution stays
// unreadable with no supported repair.
func TestExecutionStoreInterruptedPublicationIsRecoverable(t *testing.T) {
	ctx := context.Background()
	s, root := newTestExecutionStore(t)
	if err := s.Create(ctx, storeTestExecution(t)); err != nil {
		t.Fatal(err)
	}
	current, _ := s.Load(ctx, "proj", storeTestExecutionID)
	next := current
	_ = next.RequestCancel("user:local", storeT0)
	s.hooks.fault = func(stage FaultStage) error {
		if stage == FaultF6 {
			return ErrSimulatedInterruption
		}
		return nil
	}
	if err := s.Save(ctx, next, current.Revision, nil); err == nil {
		t.Fatal("interruption ignored")
	}
	s.hooks.fault = nil
	if _, err := s.Load(ctx, "proj", storeTestExecutionID); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("interrupted publication: %v", err)
	}
	report, err := InspectRecovery(ctx, RecoveryRoots{State: root})
	if err != nil {
		t.Fatal(err)
	}
	var plan *RecoveryPlan
	for i, p := range report.Plans {
		if p.Directory == "executions/v2/proj" {
			plan = &report.Plans[i]
		}
	}
	if plan == nil {
		t.Fatalf("canonical Execution protocol absent from recovery: %+v", report)
	}
	authority, err := AuthorizeRecovery(*plan, plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRecovery(ctx, RecoveryRoots{State: root}, *plan, authority); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(ctx, "proj", storeTestExecutionID)
	if err != nil {
		t.Fatalf("recovered Execution unreadable: %v", err)
	}
	if got.Revision != current.Revision && got.Revision != next.Revision {
		t.Fatalf("recovery invented revision %d", got.Revision)
	}
}

func storeInvocation() workflow.Invocation {
	return workflow.Invocation{Runtime: "stub", Executable: "stub-bin", Model: "m", Effort: "runtime-default", Instructions: "i", InstructionsDigest: workflow.DigestText([]byte("i")),
		Provenance: []workflow.FieldProvenance{{Field: "runtime", Source: "local_default"}, {Field: "model", Source: "local_default"}, {Field: "effort", Source: "local_default"}, {Field: "instructions", Source: "workflow_agent"}}}
}

// TestExecutionStoreRefusedCompletionReturnsPersistedState is the IF-02
// probe (PR #366 review 5732338943): CompleteAttempt writes result fields
// before refusing mismatched command results, but Update discards that
// working copy and returns the persisted aggregate unchanged.
func TestExecutionStoreRefusedCompletionReturnsPersistedState(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestExecutionStore(t)
	if err := s.Create(ctx, storeTestExecution(t)); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	defer lease.Close()
	running, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Epoch, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local"}, Now: storeT0}); err != nil {
			return err
		}
		return e.MarkStarted("a1", lease.Epoch, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0)
	})
	if err != nil {
		t.Fatal(err)
	}
	persisted, _ := workflow.EncodeExecution(running)
	refused := workflow.AttemptResult{ExitCode: 0, Diagnostic: "not persisted", Commands: []workflow.CommandResult{{ID: "unexpected", Status: "completed"}}, Now: storeT0.Add(time.Minute)}
	// The refusal happens after the result fields were written to the working
	// copy, so the store's discard is what keeps them out of the result.
	working, _ := workflow.DecodeExecution(persisted)
	if err := working.CompleteAttempt("a1", lease.Epoch, refused); !errors.Is(err, workflow.ErrTransition) {
		t.Fatalf("working copy: %v", err)
	}
	if a, _, _ := working.FindAttempt("a1"); a.EndedAt == nil || a.Diagnostic != "not persisted" {
		t.Fatal("probe did not reach the command-result refusal")
	}
	returned, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		return e.CompleteAttempt("a1", lease.Epoch, refused)
	})
	if !errors.Is(err, workflow.ErrTransition) {
		t.Fatalf("mismatched command results: %v", err)
	}
	reloaded, err := s.Load(ctx, "proj", storeTestExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	for name, x := range map[string]workflow.Execution{"returned": returned, "reloaded": reloaded} {
		wire, _ := workflow.EncodeExecution(x)
		if string(wire) != string(persisted) {
			t.Fatalf("%s aggregate shows non-persisted completion values", name)
		}
	}
}

// A valid record and current revision do not authorize rewriting provenance.
func TestExecutionStoreRejectsHistoryRewrite(t *testing.T) {
	for _, mode := range []string{"definition", "invocation", "authority", "identity", "terminal-result", "remove-attempt"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			s, _ := newTestExecutionStore(t)
			original := storeTestExecution(t)
			if err := s.Create(ctx, original); err != nil {
				t.Fatal(err)
			}
			lease := claim(t, s)
			defer lease.Close()
			completed, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
				if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Epoch, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local"}, Now: storeT0}); err != nil {
					return err
				}
				if err := e.MarkStarted("a1", lease.Epoch, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0); err != nil {
					return err
				}
				return e.CompleteAttempt("a1", lease.Epoch, workflow.AttemptResult{ExitCode: 0, Now: storeT0})
			})
			if err != nil {
				t.Fatal(err)
			}
			before, _ := workflow.EncodeExecution(completed)
			next, _ := workflow.DecodeExecution(before)
			a := &next.Stages[0].Tasks[0].Attempts[0]
			switch mode {
			case "definition":
				replacement := original.Workflow.Stages
				replacement[0].Agents[0].Instructions = "replacement"
				fresh, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: storeTestExecutionID, ProjectID: "proj", WorkItem: original.WorkItem, Repositories: original.Repositories, WorkflowID: "minimal", Revision: "r1", Now: storeT0, Stages: replacement})
				if err != nil {
					t.Fatal(err)
				}
				next.Workflow = fresh.Workflow
			case "invocation":
				a.Invocation.Model = "changed-model"
			case "authority":
				a.Authority.Actor = "user:other"
			case "identity":
				a.AttemptID = "rewritten"
			case "terminal-result":
				a.Diagnostic = "rewritten observation"
			case "remove-attempt":
				next.Stages[0].Tasks[0].Attempts = nil
				next.Stages[0].Status = workflow.StagePending
				next.Status = workflow.RunReady
			}
			next.Revision++
			if err := next.Validate(); err != nil {
				t.Fatalf("fixture must be internally valid: %v", err)
			}
			if err := s.Save(ctx, next, completed.Revision, lease); !errors.Is(err, ErrConflict) {
				t.Fatalf("history rewrite %s accepted: %v", mode, err)
			}
			got, err := s.Load(ctx, "proj", storeTestExecutionID)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := workflow.EncodeExecution(got)
			if string(before) != string(after) {
				t.Fatal("refused update changed bytes")
			}
		})
	}
}

func TestExecutionStoreUnknownResolutionPreservesObservations(t *testing.T) {
	ctx := context.Background()
	s, _ := newTestExecutionStore(t)
	if err := s.Create(ctx, storeTestExecution(t)); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	defer lease.Close()
	_, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Epoch, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local"}, Now: storeT0}); err != nil {
			return err
		}
		return e.MarkUnknown("a1", lease.Epoch, "lost terminal observation", storeT0)
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := s.Update(ctx, "proj", storeTestExecutionID, nil, func(e *workflow.Execution) error { return e.ResolveUnknown("a1", "human:maintainer", storeT0) })
	if err != nil {
		t.Fatal(err)
	}
	a := resolved.Stages[0].Tasks[0].Attempts[0]
	if a.Status != workflow.AttemptUnknown || a.Diagnostic != "lost terminal observation" || a.Resolution == nil {
		t.Fatal("resolution rewrote unknown history")
	}
}

// Accepted messages are attempt-owned facts; a current writer cannot drop or
// rewrite them, even with a valid record and the expected revision.
func TestExecutionStoreRejectsMessageHistoryRewrite(t *testing.T) {
	for _, mode := range []string{"drop", "rewrite"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			s, _ := newTestExecutionStore(t)
			original, err := workflow.NewExecution(workflow.NewExecutionInput{
				ExecutionID: storeTestExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"},
				Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/app"}}, WorkflowID: "pair", Revision: "r1", Now: storeT0,
				Stages: []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder"}, {ID: "reviewer"}}, TimeoutSeconds: 60}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Create(ctx, original); err != nil {
				t.Fatal(err)
			}
			lease := claim(t, s)
			defer lease.Close()
			current, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
				if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Epoch, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Actor: "user:local"}, Now: storeT0}); err != nil {
					return err
				}
				if err := e.MarkStarted("a1", lease.Epoch, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0); err != nil {
					return err
				}
				return e.AcceptMessage(workflow.IngressContext{ExecutionID: storeTestExecutionID, AttemptID: "a1", AgentID: "builder", Runtime: "stub", Generation: lease.Epoch}, workflow.MessageCandidate{ID: "m1", Recipient: "reviewer", Kind: workflow.MessageQuestion, Content: "original"}, storeT0)
			})
			if err != nil {
				t.Fatal(err)
			}
			wire, _ := workflow.EncodeExecution(current)
			next, _ := workflow.DecodeExecution(wire)
			if mode == "drop" {
				next.Messages = nil
			} else {
				next.Messages[0].Content = "rewritten"
			}
			next.Revision++
			if err := next.Validate(); err != nil {
				t.Fatalf("fixture must be internally valid: %v", err)
			}
			if err := s.Save(ctx, next, current.Revision, lease); !errors.Is(err, ErrConflict) {
				t.Fatalf("message history %s accepted: %v", mode, err)
			}
		})
	}
}

// S23-CR-03 fixtures use real store transitions with a current lease/revision.
func storeIntegratedExecution(t *testing.T) (ExecutionStore, *RunLease, workflow.Execution) {
	t.Helper()
	ctx := context.Background()
	s, _ := newTestExecutionStore(t)
	effects := []string{"workspace_write", "validation_command"}
	x, err := workflow.NewExecution(workflow.NewExecutionInput{
		ExecutionID: storeTestExecutionID, ProjectID: "proj", WorkItem: workflow.WorkItemLink{Provider: "linear", ID: "AXM-42", RepositoryKey: "app"},
		Repositories: []workflow.RepositoryBinding{{Key: "app", Identity: "github.com/example/app"}}, WorkflowID: "minimal", Revision: "r1", Now: storeT0,
		Stages: []workflow.StageSpec{{ID: "work", Agents: []workflow.AgentSpec{{ID: "builder", Effects: effects}}, TimeoutSeconds: 60,
			Commands: []workflow.CommandCheck{{ID: "check", Argv: []string{"/bin/true"}, TimeoutSeconds: 10, Effects: []string{"validation_command"}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Create(ctx, x); err != nil {
		t.Fatal(err)
	}
	lease := claim(t, s)
	t.Cleanup(func() { lease.Close() })
	x, err = s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		if err := e.PrepareAttempt(workflow.PrepareInput{Ref: workflow.TaskRef{StageID: "work", AgentID: "builder"}, AttemptID: "a1", Generation: lease.Epoch, Invocation: storeInvocation(), Authority: workflow.AttemptAuthority{RepositoryKey: "app", Effects: effects, Actor: "user:local"}, Now: storeT0}); err != nil {
			return err
		}
		if err := e.MarkStarted("a1", lease.Epoch, workflow.ProcessIdentity{PID: 100, StartIdentity: "s", GroupIdentity: "100"}, storeT0); err != nil {
			return err
		}
		if err := e.PrepareCommand("a1", lease.Epoch, "check", storeT0); err != nil {
			return err
		}
		if err := e.StartCommand("a1", lease.Epoch, "check", workflow.ProcessIdentity{PID: 101, StartIdentity: "s", GroupIdentity: "101"}, storeT0); err != nil {
			return err
		}
		if err := e.FinishCommand("a1", lease.Epoch, "check", 0, storeT0); err != nil {
			return err
		}
		a, _, _ := e.FindAttempt("a1")
		return e.CompleteAttempt("a1", lease.Epoch, workflow.AttemptResult{Commands: append([]workflow.CommandResult(nil), a.Commands...), Now: storeT0})
	})
	if err != nil {
		t.Fatal(err)
	}
	for n, id := range []string{"int-one", "int-two"} {
		head, candidate, merge := strings.Repeat("b", 40), strings.Repeat("c", 40), strings.Repeat("d", 40)
		if n == 1 {
			head, candidate, merge = merge, strings.Repeat("e", 40), strings.Repeat("f", 40)
		}
		_, err = s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
			return e.PrepareIntegration(workflow.IntegrationInput{IntegrationID: id, Generation: lease.Epoch, SourceAttemptID: "a1", RepositoryKey: "app", RepositoryIdentity: "github.com/example/app", SourceBranch: "refs/heads/agent", Worktree: "agent", BaseSHA: strings.Repeat("b", 40), TargetRef: "refs/heads/integration", ExpectedTargetHead: head, CandidateSHA: candidate, Now: storeT0})
		})
		if err != nil {
			t.Fatal(err)
		}
		x, err = s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
			return e.FinishIntegration(id, lease.Epoch, workflow.IntegrationResult{Status: workflow.IntegrationIntegrated, MergeSHA: merge, FinalSHA: merge, ObservedTargetHead: merge, Checks: []workflow.IntegrationCheck{{ID: "check", Status: workflow.CheckPassed}}, Now: storeT0})
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return s, lease, x
}

func TestExecutionStoreRejectsIntegrationHistoryRewrite(t *testing.T) {
	for _, mode := range []string{"delete", "target", "final-sha", "diagnostic", "checks", "reorder", "identity", "terminal-append"} {
		t.Run(mode, func(t *testing.T) {
			s, lease, current := storeIntegratedExecution(t)
			before, _ := workflow.EncodeExecution(current)
			next, _ := workflow.DecodeExecution(before)
			i := &next.Integrations[0]
			switch mode {
			case "delete":
				next.Integrations = nil
			case "target":
				i.TargetRef = "refs/heads/other"
			case "final-sha":
				i.FinalSHA = strings.Repeat("a", 40)
				i.MergeSHA = i.FinalSHA
				i.ObservedTargetHead = i.FinalSHA
			case "diagnostic":
				i.Diagnostic = "rewritten"
			case "checks":
				i.Checks = nil
				i.Status = workflow.IntegrationNotIntegrated
				i.FinalSHA = ""
				i.Diagnostic = "rewritten"
			case "reorder":
				next.Integrations[0], next.Integrations[1] = next.Integrations[1], next.Integrations[0]
			case "identity":
				i.IntegrationID = "other-id"
			case "terminal-append":
				appended := *i
				appended.IntegrationID = "unprepared-terminal"
				next.Integrations = append(next.Integrations, appended)
			}
			next.Revision++
			if err := next.Validate(); err != nil {
				t.Fatalf("fixture must be valid: %v", err)
			}
			if err := s.Save(context.Background(), next, current.Revision, lease); !errors.Is(err, ErrConflict) {
				t.Fatalf("accepted %s rewrite: %v", mode, err)
			}
			got, err := s.Load(context.Background(), "proj", storeTestExecutionID)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := workflow.EncodeExecution(got)
			if string(before) != string(after) {
				t.Fatal("refused save changed history/revision")
			}
		})
	}
}

func TestExecutionStoreUpdateIntegrationSnapshotIsolation(t *testing.T) {
	for _, mode := range []string{"integration", "checks", "ended-at"} {
		t.Run(mode, func(t *testing.T) {
			s, lease, current := storeIntegratedExecution(t)
			before, _ := workflow.EncodeExecution(current)
			refusal := errors.New("callback refused")
			got, err := s.Update(context.Background(), "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
				switch mode {
				case "integration":
					e.Integrations[0].Diagnostic = "mutated"
				case "checks":
					e.Integrations[0].Checks[0].Status = workflow.CheckFailed
				case "ended-at":
					*e.Integrations[0].EndedAt = storeT0.Add(time.Hour)
				}
				return refusal
			})
			if !errors.Is(err, refusal) {
				t.Fatal(err)
			}
			after, _ := workflow.EncodeExecution(got)
			if string(before) != string(after) {
				t.Fatal("callback mutation aliased prior snapshot")
			}
		})
	}
}

func TestExecutionStoreIntegrationPreparedAndHumanDisposition(t *testing.T) {
	s, lease, _ := storeIntegratedExecution(t)
	ctx := context.Background()
	prepared, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		return e.PrepareIntegration(workflow.IntegrationInput{IntegrationID: "third", Generation: lease.Epoch, SourceAttemptID: "a1", RepositoryKey: "app", RepositoryIdentity: "github.com/example/app", SourceBranch: "refs/heads/agent", Worktree: "agent", BaseSHA: strings.Repeat("b", 40), TargetRef: "refs/heads/integration", ExpectedTargetHead: strings.Repeat("f", 40), CandidateSHA: strings.Repeat("a", 40), Now: storeT0})
	})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := workflow.EncodeExecution(prepared)
	next, _ := workflow.DecodeExecution(before)
	next.Integrations[2].TargetRef = "refs/heads/rewritten"
	next.Revision++
	if err := s.Save(ctx, next, prepared.Revision, lease); !errors.Is(err, ErrConflict) {
		t.Fatalf("prepared intent rewrite: %v", err)
	}
	_, err = s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		return e.MarkIntegrationUnknown("third", lease.Epoch, "lost observation", storeT0)
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := s.Update(ctx, "proj", storeTestExecutionID, nil, func(e *workflow.Execution) error {
		return e.ResolveIntegration("third", "human:maintainer", storeT0)
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Integrations[2].Status != workflow.IntegrationUnknown || resolved.Integrations[2].Diagnostic != "lost observation" || resolved.Integrations[2].Resolution == nil {
		t.Fatal("human disposition lost observations")
	}
	before, _ = workflow.EncodeExecution(resolved)
	refusal := errors.New("callback refused")
	got, err := s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		e.Integrations[2].Resolution.Actor = "human:other"
		return refusal
	})
	if !errors.Is(err, refusal) {
		t.Fatal(err)
	}
	after, _ := workflow.EncodeExecution(got)
	if string(before) != string(after) {
		t.Fatal("resolution pointer aliased prior snapshot")
	}
	_, err = s.Update(ctx, "proj", storeTestExecutionID, lease, func(e *workflow.Execution) error {
		e.Integrations[2].Resolution.Actor = "human:other"
		e.Revision++
		return nil
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("terminal resolution rewrite accepted: %v", err)
	}
	got, err = s.Load(ctx, "proj", storeTestExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	after, _ = workflow.EncodeExecution(got)
	if string(before) != string(after) {
		t.Fatal("refused update changed current history/revision")
	}
}
