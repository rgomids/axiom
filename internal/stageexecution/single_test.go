package stageexecution

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

type fakeSingleLease struct{ epoch uint64 }

func (l fakeSingleLease) Epoch() uint64   { return l.epoch }
func (l fakeSingleLease) Validate() error { return nil }
func (l fakeSingleLease) Close() error    { return nil }

type fakeSingleStore struct {
	mu          sync.Mutex
	run         executiongraph.SingleStageRun
	authorities map[string]executiongraph.SingleStageAuthority
	contract    Contract
	saves       []executiongraph.SingleStageRun
	epoch       uint64
	onSave      func(executiongraph.SingleStageRun)
}

func (f *fakeSingleStore) LoadSingle(_ context.Context, r SingleRequest) (executiongraph.SingleStageRun, executiongraph.SingleStageAuthority, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return executiongraph.CloneSingleStageRun(f.run), f.authorities[r.AuthorityRef], nil
}
func (f *fakeSingleStore) SingleContract(context.Context, SingleRequest) (Contract, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.contract, nil
}
func (f *fakeSingleStore) AcquireSingle(context.Context, executiongraph.SingleStageRun) (SupervisorLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.epoch++
	return fakeSingleLease{f.epoch}, nil
}
func (f *fakeSingleStore) SaveSingle(_ context.Context, r executiongraph.SingleStageRun, l SupervisorLease) (executiongraph.SingleStageRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.StorageRevision != f.run.StorageRevision {
		return r, errors.New("cas conflict")
	}
	r.RunEpoch = l.Epoch()
	if err := executiongraph.ValidateSingleStageUpdate(f.run, r); err != nil {
		return r, err
	}
	r.StorageRevision = sha256.Sum256([]byte(fmt.Sprint(len(f.saves), r.Attempts)))
	f.run = r
	f.saves = append(f.saves, executiongraph.CloneSingleStageRun(r))
	if f.onSave != nil {
		f.onSave(r)
	}
	return r, nil
}
func (f *fakeSingleStore) cancel(t *testing.T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	next := executiongraph.CloneSingleStageRun(f.run)
	next.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000077", ParentID: next.Scope.ExecutionID, GraphRevision: next.Binding.WorkflowRevision, ActorKind: "human-operator", Actor: "op", RequestedAt: next.CreatedAt.Add(time.Second), AuthorityRef: strings.Repeat("e", 64)}
	if err := executiongraph.ValidateSingleStageUpdate(f.run, next); err != nil {
		t.Error(err)
		return
	}
	next.StorageRevision = sha256.Sum256([]byte("cancelled"))
	f.run = next
}

func singleFixture(t *testing.T) (*fakeSingleStore, SingleRequest) {
	t.Helper()
	return singleFixtureWithPreview(t, strings.Repeat("7", 64))
}

func singleFixtureWithPreview(t *testing.T, preview string) (*fakeSingleStore, SingleRequest) {
	t.Helper()
	c := currentContract(t)
	c.Graph = nil
	p := c.Execution.Binding.ResolvedProfile
	c.Execution.ExecutionID, c.Execution.ProjectID, c.Execution.RepositoryKey, c.Execution.Stage, c.Execution.Revision = "00000000-0000-4000-8000-000000000001", "project", "main", "intake", 2
	c.Execution.StorageRevision = sha256.Sum256([]byte("execution3"))
	scope := executiongraph.SingleStageScope{ProjectID: "project", RepositoryKey: "main", WorkItem: "github:owner/repo#7", ExecutionID: c.Execution.ExecutionID, StageID: "intake"}
	digest, err := executiongraph.SingleStageScopeDigest(scope)
	if err != nil {
		t.Fatal(err)
	}
	created := time.Unix(1000, 0).UTC()
	generation := fmt.Sprintf("%x", c.Execution.StorageRevision)
	binding := executiongraph.SingleStageBinding{ExecutionGeneration: generation, WorkflowRevision: 2, PreviewDigest: preview, ProposalDigest: strings.Repeat("8", 64), InputDigest: strings.Repeat("9", 64), AxiomDigest: strings.Repeat("a", 64), Workspace: "/synthetic/worktree", Inbox: "/synthetic/inbox", Effects: []executiongraph.Effect{}, Controls: executiongraph.ExecutionControls{Timeout: 5 * time.Second, MaximumAttempts: 3}, Resolution: executiongraph.Resolution{ResolvedProfile: p, RuntimeID: p.RuntimeID, ModelProfileID: p.ProfileID, ConfigurationRevision: p.ConfigurationRevision, ObservationRevision: p.ObservationRevision}}
	a := executiongraph.SingleStageAuthority{FormatVersion: 2, Scope: executiongraph.SingleStageAuthorityScope, ProjectID: "project", ExecutionID: scope.ExecutionID, StageID: "intake", ScopeDigest: digest, ExecutionGeneration: generation, WorkflowRevision: 2, PreviewDigest: binding.PreviewDigest, DispatchDigest: binding.PreviewDigest, ProposalDigest: binding.ProposalDigest, AttemptNumber: 1, Effects: []executiongraph.Effect{}, Disposition: executiongraph.DispositionNotRequired, ActorKind: "human-operator", Actor: "op", RecordedAt: created}
	ref, err := executiongraph.SingleStageAuthorityDigest(a)
	if err != nil {
		t.Fatal(err)
	}
	run := executiongraph.SingleStageRun{Kind: executiongraph.SingleStageRunKind, FormatVersion: 1, ScopeDigest: digest, Scope: scope, Binding: binding, CreatedAt: created, Authorities: []string{ref}, Attempts: []executiongraph.SuccessorAttempt{}, StorageRevision: sha256.Sum256([]byte("ledger"))}
	if err := executiongraph.ValidateSingleStageRun(run); err != nil {
		t.Fatal(err)
	}
	store := &fakeSingleStore{run: run, authorities: map[string]executiongraph.SingleStageAuthority{ref: a}, contract: c}
	return store, SingleRequest{ProjectID: "project", ExecutionID: scope.ExecutionID, ScopeDigest: digest, AuthorityRef: ref}
}

type fakeSinglePreparation struct {
	prepared []string
	drains   int
	records  []executiongraph.AttemptRecord
	fail     error
}

func (f *fakeSinglePreparation) PrepareSingle(_ context.Context, _ executiongraph.SingleStageRun, id string, _ uint64) (executiongraph.Invocation, error) {
	f.prepared = append(f.prepared, id)
	return executiongraph.Invocation{Stdin: []byte("packet")}, f.fail
}
func (f *fakeSinglePreparation) DrainSingle(_ context.Context, _ executiongraph.SingleStageRun, _ executiongraph.SuccessorAttempt, final bool) ([]executiongraph.AttemptRecord, error) {
	if final {
		f.drains++
	}
	return f.records, nil
}

type fakeSingleRunner struct {
	result  executiongraph.ProcessResult
	block   bool
	started chan struct{}
	calls   int
}

func (f *fakeSingleRunner) RunObserved(ctx context.Context, _ executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	f.calls++
	if err := started(&executiongraph.ProcessCorrelation{PID: 42, StartIdentity: "s", GroupIdentity: "g"}); err != nil {
		return executiongraph.ProcessResult{Stopped: true, OutcomeUnknown: true}
	}
	if f.started != nil {
		close(f.started)
	}
	if f.block {
		<-ctx.Done()
		return executiongraph.ProcessResult{Stopped: true, ExitCode: -1}
	}
	return f.result
}

func newSingleSupervisor(store SingleStore, prep SinglePreparation, runner SupervisorRunner) SingleSupervisor {
	n := 0
	return SingleSupervisor{Store: store, Preparation: prep, Runner: runner, AllocateID: func() (string, error) { n++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", 500+n), nil }, Now: func() time.Time { return time.Unix(2000, 0).UTC() }, PollInterval: 5 * time.Millisecond}
}

func TestLaunchSingleVerifiesThePendingReceipt(t *testing.T) {
	store, r := singleFixture(t)
	d := &fakeDetacher{}
	if _, err := LaunchSingle(context.Background(), r, store, d); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.args, []string{"__stage-supervisor", "single", r.ProjectID, r.ExecutionID, r.ScopeDigest, r.AuthorityRef}) {
		t.Fatal(d.args)
	}
	forged := r
	forged.AuthorityRef = strings.Repeat("0", 64)
	if _, err := LaunchSingle(context.Background(), forged, store, d); !errors.Is(err, ErrSupervisorAuthority) {
		t.Fatalf("forged receipt = %v", err)
	}
	store.contract.Execution.Revision = 3
	if _, err := LaunchSingle(context.Background(), r, store, d); !errors.Is(err, ErrStalePreview) {
		t.Fatalf("advanced Execution3 = %v", err)
	}
	if d.calls != 1 {
		t.Fatal("refused launch spawned a supervisor")
	}
}

func TestSingleSupervisorPersistsOneClassifiedAttempt(t *testing.T) {
	for name, tc := range map[string]struct {
		result executiongraph.ProcessResult
		status executiongraph.AttemptStatus
		reason executiongraph.ClassificationReason
	}{
		"exit zero without result": {executiongraph.ProcessResult{Stopped: true}, executiongraph.AttemptFailed, executiongraph.ResultMissing},
		"exit seven":               {executiongraph.ProcessResult{Stopped: true, ExitCode: 7}, executiongraph.AttemptFailed, executiongraph.ProcessFailed},
		"ambiguous":                {executiongraph.ProcessResult{EffectAmbiguous: true}, executiongraph.AttemptUnknown, executiongraph.ProcessOutcomeUnknown},
	} {
		t.Run(name, func(t *testing.T) {
			store, r := singleFixture(t)
			prep := &fakeSinglePreparation{}
			runner := &fakeSingleRunner{result: tc.result}
			if err := newSingleSupervisor(store, prep, runner).Run(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			if len(store.saves) != 2 || store.saves[0].Attempts[0].Status != executiongraph.AttemptRunning || store.saves[0].Attempts[0].Process == nil {
				t.Fatalf("running attempt not persisted before completion: %+v", store.saves)
			}
			at := store.run.Attempts[0]
			if at.Status != tc.status || at.ClassificationReason != tc.reason || at.LeaseEpoch != store.run.RunEpoch || prep.drains != 1 {
				t.Fatalf("attempt = %+v", at)
			}
			if err := newSingleSupervisor(store, prep, runner).Run(context.Background(), r); !errors.Is(err, ErrSupervisorAuthority) || runner.calls != 1 {
				t.Fatalf("consumed receipt reused: %v calls=%d", err, runner.calls)
			}
		})
	}
}

func TestSingleSupervisorCancellationAndDrift(t *testing.T) {
	store, r := singleFixture(t)
	runner := &fakeSingleRunner{block: true, started: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- newSingleSupervisor(store, &fakeSinglePreparation{}, runner).Run(context.Background(), r) }()
	<-runner.started
	store.cancel(t)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	at := store.run.Attempts[0]
	if at.Status != executiongraph.AttemptCancelled || at.ClassificationReason != executiongraph.CancelConfirmed || !at.CancellationSeen {
		t.Fatalf("cancelled attempt = %+v", at)
	}

	store, r = singleFixture(t)
	store.cancel(t)
	runner = &fakeSingleRunner{}
	if err := newSingleSupervisor(store, &fakeSinglePreparation{}, runner).Run(context.Background(), r); err == nil || runner.calls != 0 {
		t.Fatal("dispatch after a cancellation request")
	}

	store, r = singleFixture(t)
	prep := &fakeSinglePreparation{}
	runner = &fakeSingleRunner{}
	store.contract.Execution.StorageRevision = sha256.Sum256([]byte("advanced"))
	if err := newSingleSupervisor(store, prep, runner).Run(context.Background(), r); !errors.Is(err, ErrStalePreview) || len(prep.prepared) != 0 || runner.calls != 0 {
		t.Fatalf("Execution3 drift = %v", err)
	}

	store, r = singleFixture(t)
	prep = &fakeSinglePreparation{fail: errInjected}
	if err := newSingleSupervisor(store, prep, runner).Run(context.Background(), r); !errors.Is(err, errInjected) || runner.calls != 0 || len(store.saves) != 0 {
		t.Fatalf("preparation failure = %v", err)
	}
}

func TestResumeSingleStartsOnlyThePendingReceipt(t *testing.T) {
	service, _, compiler, _ := previewFixture(t, true)
	reviewed, err := service.Preview(context.Background(), compiler.plan.Execution, compiler.plan.StageID)
	if err != nil {
		t.Fatal(err)
	}
	store, r := singleFixtureWithPreview(t, reviewed.Digest)
	ports := &fakeRunPorts{}
	s := RunService{SingleStore: store, Preview: service, Workspaces: ports, Starter: ports}
	req := RunRequest{CommittedSingle: &r, Execution: compiler.plan.Execution, StageID: compiler.plan.StageID, PreviewDigest: reviewed.Digest, ExpectedRevision: compiler.plan.Execution.Revision, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "op"}
	result, err := s.Apply(context.Background(), req)
	if err != nil || result.Status != "success" || !reflect.DeepEqual(ports.events, []string{"workspaces", "start"}) {
		t.Fatal(result, err, ports.events)
	}
	noExternal := req
	noExternal.AuthorizeExternal = false
	if _, err := s.Apply(context.Background(), noExternal); !errors.Is(err, ErrDispatchAuthority) {
		t.Fatalf("resume without external authority = %v", err)
	}
	start := store.run.CreatedAt.Add(time.Second)
	running := executiongraph.CloneSingleStageRun(store.run)
	running.RunEpoch = 1
	running.Attempts = []executiongraph.SuccessorAttempt{{AttemptID: "00000000-0000-4000-8000-000000000099", Number: 1, Status: executiongraph.AttemptRunning, StartedAt: &start, LeaseEpoch: 1, PacketDigest: strings.Repeat("c", 64), Process: &executiongraph.ProcessCorrelation{PID: 1, StartIdentity: "s", GroupIdentity: "g"}, ReconciliationFacts: []executiongraph.ReconciliationFact{}}}
	store.run = running
	ports.events = nil
	if result, err := s.Apply(context.Background(), req); !errors.Is(err, ErrSupervisorRecovery) || result.Recovery != "reconcile_required" || len(ports.events) != 0 {
		t.Fatalf("resume over a running attempt = %+v %v", result, err)
	}
}
