package stageexecution

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

type fakeSingleLedger struct {
	store    *fakeSingleStore
	retries  []executiongraph.SingleStageAuthority
	disposed []string
	failure  error
}

func (f *fakeSingleLedger) SingleSnapshot(context.Context) (Contract, executiongraph.SingleStageRun, error) {
	return f.store.contract, executiongraph.CloneSingleStageRun(f.store.run), nil
}
func (f *fakeSingleLedger) RequestSingleCancel(_ context.Context, r executiongraph.SingleStageRun, c executiongraph.CancelRequest) (executiongraph.SingleStageRun, error) {
	next := executiongraph.CloneSingleStageRun(r)
	next.CancelRequest = &c
	if err := executiongraph.ValidateSingleStageUpdate(f.store.run, next); err != nil {
		return r, err
	}
	next.StorageRevision = sha256.Sum256([]byte("cancel"))
	f.store.run = next
	return next, nil
}
func (f *fakeSingleLedger) AppendSingleReconciliation(_ context.Context, r executiongraph.SingleStageRun, fact executiongraph.ReconciliationFact) (executiongraph.SingleStageRun, error) {
	next := executiongraph.CloneSingleStageRun(r)
	last := &next.Attempts[len(next.Attempts)-1]
	last.ReconciliationFacts = append(append([]executiongraph.ReconciliationFact(nil), last.ReconciliationFacts...), fact)
	if err := executiongraph.ValidateSingleStageUpdate(f.store.run, next); err != nil {
		return r, err
	}
	next.StorageRevision = sha256.Sum256([]byte("fact"))
	f.store.run = next
	return next, nil
}
func (f *fakeSingleLedger) CommitSingleRetry(_ context.Context, next executiongraph.SingleStageRun, a executiongraph.SingleStageAuthority, _ workflow.State) error {
	if f.failure != nil {
		return f.failure
	}
	if err := executiongraph.CheckSingleStageRetryAuthority(f.store.run, a); err != nil {
		return err
	}
	ref, _ := executiongraph.SingleStageAuthorityDigest(a)
	next.StorageRevision = sha256.Sum256([]byte("retry"))
	f.store.run = next
	f.store.authorities[ref] = a
	f.retries = append(f.retries, a)
	return nil
}
func (f *fakeSingleLedger) DisposeSingle(_ context.Context, _ executiongraph.SingleStageRun, a executiongraph.SingleStageAuthority) error {
	f.disposed = append(f.disposed, a.Disposition)
	return nil
}

type fakeSingleObserver struct{ alive, held bool }

func (o fakeSingleObserver) ObserveSingle(_ context.Context, r executiongraph.SingleStageRun, attempt string) (executiongraph.ReconciliationObservations, error) {
	return executiongraph.GatherSingleStageObservations(r, attempt, executiongraph.ReconciliationSources{
		LeaseProbe: func() (bool, error) { return o.held, nil },
		ProcessProbe: func(p executiongraph.ProcessCorrelation) (*executiongraph.ProcessCorrelation, error) {
			if o.alive {
				return &p, nil
			}
			return nil, nil
		},
		Worktree:         func() (string, error) { return strings.Repeat("1", 64), nil },
		CanonicalRecords: func() (string, error) { return strings.Repeat("2", 64), nil },
		QuarantinedInbox: func() (string, error) { return strings.Repeat("3", 64), nil },
	})
}

func withFinishedAttempt(store *fakeSingleStore, status executiongraph.AttemptStatus, reason executiongraph.ClassificationReason, exit int) {
	start := store.run.CreatedAt.Add(time.Second)
	finish := start.Add(time.Second)
	store.run.RunEpoch = 1
	store.run.Attempts = []executiongraph.SuccessorAttempt{{AttemptID: "00000000-0000-4000-8000-000000000201", Number: 1, Status: status, ClassificationReason: reason, ExitCode: exit, StartedAt: &start, FinishedAt: &finish, LeaseEpoch: 1, PacketDigest: strings.Repeat("c", 64), Process: &executiongraph.ProcessCorrelation{PID: 3, StartIdentity: "s", GroupIdentity: "g"}, ReconciliationFacts: []executiongraph.ReconciliationFact{}}}
	store.run.StorageRevision = sha256.Sum256([]byte(fmt.Sprint(status)))
}

func singleLifecycle(t *testing.T) (SingleLifecycle, *fakeSingleLedger, *fakeDetacher) {
	t.Helper()
	service, _, compiler, _ := previewFixture(t, true)
	reviewed, err := service.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := singleFixtureWithPreview(t, reviewed.Digest)
	store.run.Binding.ProposalDigest = reviewed.Plan.Digest
	store.run.Binding.Resolution.ResolvedProfile = reviewed.Plan.Compilation.Agents[0].Input.ResolvedProfile
	ledger := &fakeSingleLedger{store: store}
	d := &fakeDetacher{}
	n := 0
	return SingleLifecycle{Ledger: ledger, Observer: fakeSingleObserver{}, Dispatch: service, Disposer: ledger, Store: store, Detached: d, AllocateID: func() (string, error) { n++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", 300+n), nil }, Now: func() time.Time { return time.Unix(5000, 0) }}, ledger, d
}

func TestSingleCancelPublishesOnlyTheControlMessage(t *testing.T) {
	s, ledger, _ := singleLifecycle(t)
	p, err := s.CancelPreview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cancel(context.Background(), p.Digest, false, "op"); !errors.Is(err, ErrDispatchAuthority) {
		t.Fatalf("cancel without authority = %v", err)
	}
	request, err := s.Cancel(context.Background(), p.Digest, true, "op")
	if err != nil || request == nil || ledger.store.run.CancelRequest == nil || len(ledger.store.run.Attempts) != 0 {
		t.Fatal(request, err)
	}
	if _, err := s.Cancel(context.Background(), p.Digest, true, "op"); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("stale cancel preview = %v", err)
	}
	again, _ := s.CancelPreview(context.Background())
	repeated, err := s.Cancel(context.Background(), again.Digest, true, "op")
	if err != nil || *repeated != *request {
		t.Fatal("repeated cancel created another request", err)
	}
}

func TestSingleReconcileAppendsFactsWithoutStatusChange(t *testing.T) {
	s, ledger, _ := singleLifecycle(t)
	withFinishedAttempt(ledger.store, executiongraph.AttemptUnknown, executiongraph.ProcessOutcomeUnknown, 0)
	id := ledger.store.run.Attempts[0].AttemptID
	s.Observer = fakeSingleObserver{alive: true}
	p, err := s.ReconcilePreview(context.Background(), id)
	if err != nil || p.Classification != "recovery_required" || p.Next != ErrReconcileOrphanAlive.Error() {
		t.Fatal(p, err)
	}
	if _, err := s.Reconcile(context.Background(), id, p.Digest, true, "op"); !errors.Is(err, ErrReconcileOrphanAlive) {
		t.Fatalf("live orphan = %v", err)
	}
	s.Observer = fakeSingleObserver{}
	p, err = s.ReconcilePreview(context.Background(), id)
	if err != nil || p.Classification != "retry_safe" {
		t.Fatal(p, err)
	}
	fact, err := s.Reconcile(context.Background(), id, p.Digest, true, "op")
	if err != nil || fact == nil || ledger.store.run.Attempts[0].Status != executiongraph.AttemptUnknown {
		t.Fatal(fact, err)
	}
	withFinishedAttempt(ledger.store, executiongraph.AttemptFailed, executiongraph.ProcessFailed, 7)
	if _, err := s.ReconcilePreview(context.Background(), id); !errors.Is(err, ErrReconcileIneligible) {
		t.Fatalf("terminal attempt reconciled: %v", err)
	}
}

func TestSingleRetryNeedsDispositionAuthorityAndBudget(t *testing.T) {
	s, ledger, d := singleLifecycle(t)
	ledger.store.run.Binding.Effects = []executiongraph.Effect{{Kind: "repository-write", Target: "src"}}
	for ref, a := range ledger.store.authorities {
		a.Effects = ledger.store.run.Binding.Effects
		delete(ledger.store.authorities, ref)
		newRef, _ := executiongraph.SingleStageAuthorityDigest(a)
		ledger.store.authorities[newRef] = a
		ledger.store.run.Authorities = []string{newRef}
	}
	withFinishedAttempt(ledger.store, executiongraph.AttemptFailed, executiongraph.ProcessFailed, 7)
	ref, stage := previewRef(t)
	p, err := s.RetryPreview(context.Background(), ref, stage)
	if err != nil || !p.Disposition || p.AttemptNumber != 2 || p.Remaining != 2 {
		t.Fatal(p, err)
	}
	if _, err := s.Retry(context.Background(), ref, stage, p.Digest, true, false, "op", executiongraph.DispositionPreserve); !errors.Is(err, ErrDispatchAuthority) {
		t.Fatalf("retry without external authority = %v", err)
	}
	if _, err := s.Retry(context.Background(), ref, stage, p.Digest, true, true, "op", ""); !errors.Is(err, ErrDispositionRequired) {
		t.Fatalf("missing disposition = %v", err)
	}
	result, err := s.Retry(context.Background(), ref, stage, p.Digest, true, true, "op", executiongraph.DispositionPreserve)
	if err != nil || result.Status != "success" || len(ledger.retries) != 1 || ledger.retries[0].AttemptNumber != 2 || len(ledger.disposed) != 1 || d.calls != 1 {
		t.Fatal(result, err, ledger.disposed)
	}
	if _, err := s.RetryPreview(context.Background(), ref, stage); !errors.Is(err, ErrRetryIneligible) {
		t.Fatalf("second retry while one is pending = %v", err)
	}
	ledger.store.run.Binding.Controls.MaximumAttempts = 1
	ledger.store.run.Authorities = ledger.store.run.Authorities[:1]
	if _, err := s.RetryPreview(context.Background(), ref, stage); !errors.Is(err, ErrRetryExhausted) {
		t.Fatalf("exhausted = %v", err)
	}
}

func TestSingleRetryRefusesUnreconciledUnknownAndStalePreview(t *testing.T) {
	s, ledger, _ := singleLifecycle(t)
	withFinishedAttempt(ledger.store, executiongraph.AttemptUnknown, executiongraph.ProcessOutcomeUnknown, 0)
	ref, stage := previewRef(t)
	if _, err := s.RetryPreview(context.Background(), ref, stage); !errors.Is(err, ErrRetryIneligible) {
		t.Fatalf("unreconciled unknown = %v", err)
	}
	withFinishedAttempt(ledger.store, executiongraph.AttemptFailed, executiongraph.ProcessFailed, 7)
	ledger.store.run.Binding.ProposalDigest = strings.Repeat("f", 64)
	if _, err := s.RetryPreview(context.Background(), ref, stage); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("changed compiled plan = %v", err)
	}
}

func previewRef(t *testing.T) (workflowcompiler.ExecutionRef, string) {
	t.Helper()
	_, _, compiler, _ := previewFixture(t, true)
	return compiler.plan.Execution, compiler.plan.StageID
}
