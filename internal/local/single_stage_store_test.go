package local

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflow"
)

func localSingleStage(t *testing.T, state workflow.State) (executiongraph.SingleStageRun, executiongraph.SingleStageAuthority) {
	t.Helper()
	profile := &runtimeprofile.ResolvedProfileV4{ProfileID: "codex-high", RuntimeID: "codex", Model: "gpt-synthetic", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, ProfileDigest: strings.Repeat("1", 64), ConfigurationRevision: 1, ConfigurationDigest: strings.Repeat("2", 64), ExecutableDigest: strings.Repeat("3", 64), InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: strings.Repeat("4", 64), EffortSourceDigest: strings.Repeat("5", 64)}
	item := state.WorkItem
	scope := executiongraph.SingleStageScope{ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, WorkItem: item.Provider + ":" + item.Resource + "#" + item.ExternalID, ExecutionID: state.ExecutionID, StageID: string(state.Stage), StageOrdinal: state.StageOrdinal}
	digest, err := executiongraph.SingleStageScopeDigest(scope)
	if err != nil {
		t.Fatal(err)
	}
	created := state.UpdatedAt.Add(time.Second).UTC()
	binding := executiongraph.SingleStageBinding{ExecutionGeneration: hexDigestRevision(state.StorageRevision), WorkflowRevision: state.Revision, PreviewDigest: strings.Repeat("7", 64), ProposalDigest: strings.Repeat("8", 64), InputDigest: strings.Repeat("9", 64), AxiomDigest: strings.Repeat("a", 64), Workspace: "/synthetic/worktrees/one", Inbox: "/synthetic/inboxes/one", Effects: []executiongraph.Effect{{Kind: "repository-write", Target: "src"}}, Controls: executiongraph.ExecutionControls{Timeout: time.Hour, MaximumAttempts: 3}, Resolution: executiongraph.Resolution{FrozenProfile: profile, RuntimeID: "codex", ModelProfileID: "codex-high", ConfigurationRevision: 1, ObservationRevision: 1}}
	a := executiongraph.SingleStageAuthority{FormatVersion: 2, Scope: executiongraph.SingleStageAuthorityScope, ProjectID: scope.ProjectID, ExecutionID: scope.ExecutionID, StageID: scope.StageID, StageOrdinal: scope.StageOrdinal, ScopeDigest: digest, ExecutionGeneration: binding.ExecutionGeneration, WorkflowRevision: binding.WorkflowRevision, PreviewDigest: binding.PreviewDigest, DispatchDigest: binding.PreviewDigest, ProposalDigest: binding.ProposalDigest, AttemptNumber: 1, Effects: binding.Effects, Disposition: executiongraph.DispositionNotRequired, ActorKind: "human-operator", Actor: "synthetic-operator", RecordedAt: created}
	ref, err := executiongraph.SingleStageAuthorityDigest(a)
	if err != nil {
		t.Fatal(err)
	}
	return executiongraph.SingleStageRun{Kind: executiongraph.SingleStageRunKind, FormatVersion: executiongraph.OperationalSingleStageRunVersion, ScopeDigest: digest, Scope: scope, Binding: binding, CreatedAt: created, Authorities: []string{ref}, Attempts: []executiongraph.SuccessorAttempt{}}, a
}

type singleStageFixture struct {
	state  string
	store  SingleStageStore
	leases RunLeaseStore
	ledger workflow.State
	run    executiongraph.SingleStageRun
	auth   executiongraph.SingleStageAuthority
}

func newSingleStageFixture(t *testing.T) singleStageFixture {
	t.Helper()
	state := privateTestRoot(t)
	ledgers, err := NewWorkflowStore(state)
	if err != nil {
		t.Fatal(err)
	}
	execution := createLocalOperationalExecution(t, ledgers)
	store, err := NewSingleStageStore(state)
	if err != nil {
		t.Fatal(err)
	}
	leases, err := NewRunLeaseStore(state)
	if err != nil {
		t.Fatal(err)
	}
	run, auth := localSingleStage(t, execution)
	return singleStageFixture{state: state, store: store, leases: leases, ledger: execution, run: run, auth: auth}
}

func (f singleStageFixture) commit(t *testing.T) executiongraph.SingleStageRun {
	t.Helper()
	lease, err := f.leases.Acquire(context.Background(), f.run.Scope.ProjectID, f.run.Scope.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := f.store.CommitAuthority(context.Background(), f.run, f.auth, f.ledger, lease); err != nil {
		t.Fatal(err)
	}
	r, err := f.store.Load(context.Background(), f.run.Scope.ProjectID, f.run.ScopeDigest)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func runAttempt(t *testing.T, f singleStageFixture, r executiongraph.SingleStageRun, status executiongraph.AttemptStatus, reason executiongraph.ClassificationReason, exit int) executiongraph.SingleStageRun {
	t.Helper()
	lease, err := f.leases.Acquire(context.Background(), r.Scope.ProjectID, r.Scope.ExecutionID, r.RunEpoch)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	start := r.CreatedAt.Add(time.Duration(len(r.Attempts)+1) * time.Minute)
	running := executiongraph.SuccessorAttempt{AttemptID: fmt.Sprintf("00000000-0000-4000-8000-%012d", 100+len(r.Attempts)), Number: uint32(len(r.Attempts) + 1), Status: executiongraph.AttemptRunning, StartedAt: &start, LeaseEpoch: lease.Epoch, PacketDigest: strings.Repeat("c", 64), Process: &executiongraph.ProcessCorrelation{PID: 10, StartIdentity: "s", GroupIdentity: "g"}, ReconciliationFacts: []executiongraph.ReconciliationFact{}}
	next := executiongraph.CloneSingleStageRun(r)
	next.Attempts = append(next.Attempts, running)
	saved, err := f.store.Save(context.Background(), next, lease)
	if err != nil {
		t.Fatal(err)
	}
	finish := start.Add(time.Second)
	done := executiongraph.CloneSingleStageRun(saved)
	at := running
	at.Status, at.ClassificationReason, at.FinishedAt, at.ExitCode = status, reason, &finish, exit
	done.Attempts[len(done.Attempts)-1] = at
	saved, err = f.store.Save(context.Background(), done, lease)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

func TestSingleStageCommitReadBackAndInventory(t *testing.T) {
	f := newSingleStageFixture(t)
	r := f.commit(t)
	a, err := f.store.LoadAuthority(context.Background(), r, r.Authorities[0])
	if got, _ := executiongraph.SingleStageAuthorityDigest(a); err != nil || got != r.Authorities[0] {
		t.Fatalf("authority = %+v %v", a, err)
	}
	if n, err := executiongraph.CheckSingleStageDispatch(r, a); err != nil || n != 1 {
		t.Fatalf("dispatch check = %d %v", n, err)
	}
	stored, err := (WorkflowStore{root: f.store.root}).Load(context.Background(), f.ledger.ProjectID, f.ledger.RepositoryKey, f.ledger.WorkItem)
	if err != nil || stored.StorageRevision != f.ledger.StorageRevision {
		t.Fatal("single-stage commit wrote Execution3")
	}
	inventory, err := InspectStateInventory(context.Background(), f.state)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[InventoryKind]int{}
	for _, e := range inventory.Entries {
		kinds[e.Kind]++
	}
	if kinds[InventorySingleStageRun] != 1 || kinds[InventorySingleStageAuthority] != 1 || kinds[InventoryUnknown] != 0 || kinds[InventoryMalformed] != 0 {
		t.Fatalf("inventory kinds = %v", kinds)
	}
	lease, err := f.leases.Acquire(context.Background(), r.Scope.ProjectID, r.Scope.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := f.store.CommitAuthority(context.Background(), f.run, f.auth, f.ledger, lease); !errors.Is(err, ErrConflict) {
		t.Fatalf("second first-attempt commit = %v", err)
	}
}

func TestSingleStageCommitRefusesDriftAndForeignLease(t *testing.T) {
	f := newSingleStageFixture(t)
	ctx := context.Background()
	if err := f.store.CommitAuthority(ctx, f.run, f.auth, f.ledger, nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("no lease = %v", err)
	}
	foreign, err := f.leases.Acquire(ctx, f.run.Scope.ProjectID, "00000000-0000-4000-8000-0000000000ff", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.CommitAuthority(ctx, f.run, f.auth, f.ledger, foreign); !errors.Is(err, ErrConflict) {
		t.Fatalf("foreign lease = %v", err)
	}
	foreign.Close()
	lease, err := f.leases.Acquire(ctx, f.run.Scope.ProjectID, f.run.Scope.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	stale := f.ledger
	stale.StorageRevision = sha256.Sum256([]byte("other generation"))
	if err := f.store.CommitAuthority(ctx, f.run, f.auth, stale, lease); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale Execution3 generation = %v", err)
	}
	if _, err := f.store.Load(ctx, f.run.Scope.ProjectID, f.run.ScopeDigest); err == nil {
		t.Fatal("refused commit published a ledger")
	}
}

func TestSingleStageCommitFaultWindows(t *testing.T) {
	for _, fault := range []FaultStage{FaultF0, FaultF1, FaultF2, FaultF3, FaultF4, FaultF5, FaultF6, FaultF7, FaultF8} {
		t.Run(string(fault), func(t *testing.T) {
			f := newSingleStageFixture(t)
			f.store.hooks.fault = func(stage FaultStage) error {
				if stage == fault {
					return ErrSimulatedInterruption
				}
				return nil
			}
			lease, err := f.leases.Acquire(context.Background(), f.run.Scope.ProjectID, f.run.Scope.ExecutionID, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer lease.Close()
			err = f.store.CommitAuthority(context.Background(), f.run, f.auth, f.ledger, lease)
			var pe *PublicationError
			if !errors.As(err, &pe) {
				t.Fatalf("fault not reported: %v", err)
			}
			if pe.Committed != (fault == FaultF6 || fault == FaultF7 || fault == FaultF8) {
				t.Fatalf("commit truth %+v", pe)
			}
			f.store.hooks.fault = nil
			r, err := f.store.Load(context.Background(), f.run.Scope.ProjectID, f.run.ScopeDigest)
			if err == nil {
				if _, err := f.store.LoadAuthority(context.Background(), r, r.Authorities[0]); err != nil {
					t.Fatalf("readable ledger without its receipt: %v", err)
				}
				if !pe.Committed {
					t.Fatal("uncommitted publication became readable")
				}
				return
			}
			if !errors.Is(err, ErrRecoveryRequired) && !errors.Is(err, ErrNotFound) && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("interrupted publication = %v", err)
			}
		})
	}
}

func TestSingleStageSaveEpochFencingAndCancel(t *testing.T) {
	f := newSingleStageFixture(t)
	ctx := context.Background()
	r := f.commit(t)
	old, err := f.leases.Acquire(ctx, r.Scope.ProjectID, r.Scope.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	oldEpoch := old.Epoch
	old.Close()
	failed := runAttempt(t, f, r, executiongraph.AttemptFailed, executiongraph.ProcessFailed, 7)
	if failed.RunEpoch <= oldEpoch || failed.Attempts[0].LeaseEpoch != failed.RunEpoch {
		t.Fatalf("epoch not advanced: %+v", failed)
	}
	if _, err := f.leases.Acquire(ctx, r.Scope.ProjectID, r.Scope.ExecutionID, failed.RunEpoch+5); !errors.Is(err, ErrConflict) {
		t.Fatalf("lease skipped the persisted epoch: %v", err)
	}
	lease, err := f.leases.Acquire(ctx, r.Scope.ProjectID, r.Scope.ExecutionID, failed.RunEpoch)
	if err != nil {
		t.Fatal(err)
	}
	rewrite := executiongraph.CloneSingleStageRun(failed)
	rewrite.Attempts[0].ExitCode = 9
	if _, err := f.store.Save(ctx, rewrite, lease); err == nil {
		t.Fatal("older-epoch terminal attempt rewritten")
	}
	lease.Close()
	if _, err := f.store.Save(ctx, failed, lease); err == nil {
		t.Fatal("closed lease saved")
	}
	request := executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000020", ParentID: r.Scope.ExecutionID, GraphRevision: r.Binding.WorkflowRevision, ActorKind: "human-operator", Actor: "op", RequestedAt: failed.CreatedAt.Add(time.Hour), AuthorityRef: strings.Repeat("f", 64)}
	cancelled, err := f.store.RequestCancel(ctx, failed, request)
	if err != nil || cancelled.CancelRequest == nil {
		t.Fatalf("cancel = %v", err)
	}
	if _, err := f.store.RequestCancel(ctx, failed, request); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale cancel CAS = %v", err)
	}
	again, err := f.store.RequestCancel(ctx, cancelled, request)
	if err != nil || again.StorageRevision != cancelled.StorageRevision {
		t.Fatal("repeated cancel rewrote the ledger")
	}
	if ok, _ := executiongraph.SingleStageRetryEligible(cancelled); ok {
		t.Fatal("cancelled stage retry eligible")
	}
}

func TestSingleStageRetryConsumesReviewedGeneration(t *testing.T) {
	f := newSingleStageFixture(t)
	ctx := context.Background()
	r := runAttempt(t, f, f.commit(t), executiongraph.AttemptFailed, executiongraph.ProcessFailed, 7)
	retry := f.auth
	retry.AttemptNumber, retry.PriorAttemptID, retry.LedgerGeneration, retry.Disposition, retry.RecordedAt = 2, r.Attempts[0].AttemptID, hexDigestRevision(r.StorageRevision), executiongraph.DispositionPreserve, r.CreatedAt.Add(time.Hour)
	ref, err := executiongraph.SingleStageAuthorityDigest(retry)
	if err != nil {
		t.Fatal(err)
	}
	next := executiongraph.CloneSingleStageRun(r)
	next.Authorities = append(next.Authorities, ref)
	lease, err := f.leases.Acquire(ctx, r.Scope.ProjectID, r.Scope.ExecutionID, r.RunEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.CommitAuthority(ctx, next, retry, f.ledger, lease); err != nil {
		t.Fatal(err)
	}
	if err := f.store.CommitAuthority(ctx, next, retry, f.ledger, lease); !errors.Is(err, ErrConflict) {
		t.Fatalf("consumed retry generation reused: %v", err)
	}
	lease.Close()
	pending, err := f.store.Load(ctx, r.Scope.ProjectID, r.ScopeDigest)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := f.store.LoadAuthority(ctx, pending, ref)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := executiongraph.CheckSingleStageDispatch(pending, loaded); err != nil || n != 2 {
		t.Fatalf("retry dispatch = %d %v", n, err)
	}
	if _, err := f.store.LoadAuthority(ctx, pending, strings.Repeat("0", 64)); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("unlisted receipt = %v", err)
	}
}

func TestSingleStageReconciliationUnderLease(t *testing.T) {
	f := newSingleStageFixture(t)
	ctx := context.Background()
	r := runAttempt(t, f, f.commit(t), executiongraph.AttemptUnknown, executiongraph.ProcessOutcomeUnknown, 0)
	observe := func(_ context.Context, run executiongraph.SingleStageRun, attempt string) (executiongraph.ReconciliationObservations, error) {
		return executiongraph.GatherSingleStageObservations(run, attempt, executiongraph.ReconciliationSources{
			LeaseProbe:       func() (bool, error) { return true, nil },
			ProcessProbe:     func(executiongraph.ProcessCorrelation) (*executiongraph.ProcessCorrelation, error) { return nil, nil },
			Worktree:         func() (string, error) { return strings.Repeat("1", 64), nil },
			CanonicalRecords: func() (string, error) { return strings.Repeat("2", 64), nil },
			QuarantinedInbox: func() (string, error) { return strings.Repeat("3", 64), nil },
		})
	}
	o, err := observe(ctx, r, r.Attempts[0].AttemptID)
	if err != nil {
		t.Fatal(err)
	}
	o.LeaseHeld = false
	digest, _ := o.Digest()
	fact := executiongraph.ReconciliationFact{FactID: "00000000-0000-4000-8000-000000000030", ParentID: r.Scope.ExecutionID, ChildID: r.Scope.ExecutionID, GraphRevision: r.Binding.WorkflowRevision, AttemptID: r.Attempts[0].AttemptID, ObservationsDigest: digest, Classification: "retry_safe", AuthorityRef: strings.Repeat("4", 64), ActorKind: "human-operator", Actor: "op", RecordedAt: r.CreatedAt.Add(time.Hour)}
	held, err := f.leases.Acquire(ctx, r.Scope.ProjectID, r.Scope.ExecutionID, r.RunEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.AppendReconciliation(ctx, r, fact, observe); err == nil {
		t.Fatal("reconciliation while the supervisor lease is held")
	}
	held.Close()
	next, err := f.store.AppendReconciliation(ctx, r, fact, observe)
	if err != nil {
		t.Fatal(err)
	}
	if next.Attempts[0].Status != executiongraph.AttemptUnknown || len(next.Attempts[0].ReconciliationFacts) != 1 || next.RunEpoch <= r.RunEpoch {
		t.Fatalf("reconciliation result = %+v", next.Attempts[0])
	}
}
