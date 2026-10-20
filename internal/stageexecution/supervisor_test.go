package stageexecution

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

var errInjected = errors.New("injected interruption")

type fakeSupervisorLease struct {
	closed  bool
	invalid bool
}

func (*fakeSupervisorLease) Epoch() uint64 { return 1 }
func (l *fakeSupervisorLease) Validate() error {
	if l.invalid {
		return errInjected
	}
	return nil
}
func (l *fakeSupervisorLease) Close() error { l.closed = true; return nil }

type fakeSupervisorStore struct {
	contract    Contract
	g           executiongraph.SuccessorGraph
	a           executiongraph.DispatchAuthority
	lease       fakeSupervisorLease
	held        bool
	missing     bool
	saves       int
	failSave    int
	failRecords bool
	history     []executiongraph.SuccessorGraph
	onSave      func(int)
	records     []executiongraph.AttemptRecord
}

func (f *fakeSupervisorStore) Load(context.Context, SupervisorRequest) (executiongraph.SuccessorGraph, executiongraph.DispatchAuthority, error) {
	if f.missing {
		return f.g, f.a, ErrSupervisorAuthority
	}
	return cloneSupervisorGraph(f.g), f.a, nil
}
func (f *fakeSupervisorStore) Acquire(context.Context, executiongraph.SuccessorGraph) (SupervisorLease, error) {
	if f.held {
		return nil, errInjected
	}
	return &f.lease, nil
}
func (f *fakeSupervisorStore) Save(ctx context.Context, g executiongraph.SuccessorGraph, l SupervisorLease) (executiongraph.SuccessorGraph, error) {
	if e := ctx.Err(); e != nil {
		return g, e
	}
	f.saves++
	if f.onSave != nil {
		f.onSave(f.saves)
	}
	if e := ctx.Err(); e != nil {
		return g, e
	}
	if f.saves == f.failSave {
		return g, errInjected
	}
	if e := l.Validate(); e != nil {
		return g, e
	}
	if executiongraph.ValidateSuccessorUpdate(f.g, g) != nil || executiongraph.ValidateLeaseEpoch(g, l.Epoch()) != nil {
		return g, executiongraph.ErrInvalidGraph
	}
	wire, e := executiongraph.EncodeSuccessorGraph(g)
	if e != nil {
		return g, e
	}
	g.StorageRevision = sha256.Sum256(wire)
	f.g = cloneSupervisorGraph(g)
	f.history = append(f.history, cloneSupervisorGraph(g))
	return g, nil
}
func (f *fakeSupervisorStore) Records(context.Context, executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error) {
	if f.failRecords {
		return nil, errInjected
	}
	return append([]executiongraph.AttemptRecord(nil), f.records...), nil
}

type fakeSupervisorPreparation struct {
	store        *fakeSupervisorStore
	prepares     int
	drains       int
	finals       int
	failPrepare  int
	prepareError error
	failDrain    bool
	failFinal    bool
	results      bool
	kind         string
	onPrepare    func()
}

func (f *fakeSupervisorPreparation) Prepare(_ context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, id string, epoch uint64) (executiongraph.Invocation, error) {
	f.prepares++
	if f.onPrepare != nil {
		f.onPrepare()
	}
	if f.prepares == f.failPrepare {
		if f.prepareError != nil {
			return executiongraph.Invocation{}, f.prepareError
		}
		return executiongraph.Invocation{}, errInjected
	}
	if epoch != 1 || c.Envelope.Resolution.FrozenProfile == nil {
		return executiongraph.Invocation{}, ErrAdmission
	}
	return executiongraph.WithExactEnvironment(executiongraph.Invocation{RuntimeID: "codex", Argv: []string{"fake-runtime"}, Stdin: []byte(id), OutputMax: 64}, []string{"AXIOM_FAKE=1"}), nil
}
func (f *fakeSupervisorPreparation) Drain(_ context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, a executiongraph.SuccessorAttempt, final bool) ([]executiongraph.AttemptRecord, error) {
	f.drains++
	if f.failDrain {
		return nil, errInjected
	}
	if !final {
		return nil, nil
	}
	f.finals++
	if f.failFinal {
		return nil, errInjected
	}
	if f.results {
		kind := "result"
		if f.kind != "" {
			kind = f.kind
		}
		r := executiongraph.AttemptRecord{ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID, AttemptID: a.AttemptID, GraphRevision: g.Parent.GraphRevision, Revision: uint64(f.finals), Digest: fmt.Sprintf("%064x", f.finals), Kind: kind, Outcome: "succeeded", Accepted: true}
		f.store.records = append(f.store.records, r)
		return []executiongraph.AttemptRecord{r}, nil
	}
	return nil, nil
}

type fakeSupervisorRunner struct {
	calls      atomic.Int32
	active     atomic.Int32
	max        atomic.Int32
	barrier    chan struct{}
	once       sync.Once
	waitCancel bool
	started    chan struct{}
}

func (f *fakeSupervisorRunner) RunObserved(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	count := f.calls.Add(1)
	if e := started(&executiongraph.ProcessCorrelation{PID: int64(count), StartIdentity: "fake-start", GroupIdentity: "fake-group"}); e != nil {
		return executiongraph.ProcessResult{OutcomeUnknown: true}
	}
	active := f.active.Add(1)
	defer f.active.Add(-1)
	for {
		old := f.max.Load()
		if active <= old || f.max.CompareAndSwap(old, active) {
			break
		}
	}
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}
	if f.barrier != nil {
		if active == 2 {
			f.once.Do(func() { close(f.barrier) })
		}
		select {
		case <-f.barrier:
		case <-ctx.Done():
			return executiongraph.ProcessResult{OutcomeUnknown: true, Stopped: true}
		}
	}
	if f.waitCancel {
		<-ctx.Done()
		return executiongraph.ProcessResult{OutcomeUnknown: true, Stopped: true}
	}
	return executiongraph.ProcessResult{Output: []byte("fake-output"), Stopped: true}
}
func supervisorFixture(t *testing.T) (Supervisor, SupervisorRequest, *fakeSupervisorStore, *fakeSupervisorPreparation, *fakeSupervisorRunner) {
	t.Helper()
	g := graphFixture(t)
	g.StageConcurrency = 2
	a := executiongraph.DispatchAuthority{FormatVersion: 1, ProjectID: g.Children[0].Envelope.Scope.ProjectID, PreviewDigest: strings.Repeat("a", 64), Scope: "graph-first-attempts", ParentID: g.Parent.ExecutionID, Revision: g.Parent.GraphRevision, ProposalDigest: g.Parent.ProposalDigest, ActorKind: "human-operator", Actor: "fake-operator", RecordedAt: g.Parent.CreatedAt}
	for _, c := range g.Children {
		a.Children = append(a.Children, executiongraph.DispatchChildAuthority{ChildID: c.ExecutionID, Effects: c.Envelope.AllowedEffects})
	}
	digest, e := executiongraph.DispatchAuthorityDigest(a)
	if e != nil {
		t.Fatal(e)
	}
	g.DispatchAuthorityRef = digest
	for i := range g.Children {
		g.Children[i].Envelope.AuthorityReference = digest
	}
	wire, e := executiongraph.EncodeSuccessorGraph(g)
	if e != nil {
		t.Fatal(e)
	}
	g.StorageRevision = sha256.Sum256(wire)
	store := &fakeSupervisorStore{g: g, a: a, contract: currentContract(t)}
	prep := &fakeSupervisorPreparation{store: store, results: true}
	runner := &fakeSupervisorRunner{}
	ids := 0
	s := Supervisor{Store: store, Preparation: prep, Runner: runner, AllocateID: func() (string, error) { ids++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", 100+ids), nil }, Now: func() time.Time { return time.Unix(20, 0).UTC() }, PollInterval: time.Millisecond}
	return s, SupervisorRequest{ProjectID: a.ProjectID, ParentID: a.ParentID, PreviewDigest: a.PreviewDigest, AuthorityRef: digest}, store, prep, runner
}
func TestSupervisorConcurrentFirstAttemptsIntegrationPending(t *testing.T) {
	s, r, store, prep, runner := supervisorFixture(t)
	runner.barrier = make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := s.Run(ctx, r); e != nil {
		t.Fatal(e)
	}
	if runner.calls.Load() != 2 || runner.max.Load() != 2 || prep.finals != 2 || !store.lease.closed {
		t.Fatal("not a bounded concurrent run")
	}
	if len(store.g.Children[1].Attempts) != 0 {
		t.Fatal("integration dispatched")
	}
	for _, i := range []int{0, 2} {
		at := store.g.Children[i].Attempts[0]
		if at.Status != executiongraph.AttemptSucceeded || at.ClassificationReason != executiongraph.ResultRecorded || at.Process == nil || at.PacketDigest == "" {
			t.Fatal(at)
		}
	}
	if store.saves != 4 {
		t.Fatal("expected separate initial and terminal saves", store.saves)
	}
	if e := s.Run(ctx, r); e != nil || runner.calls.Load() != 2 {
		t.Fatal("restart redispatched", e)
	}
}
func TestSupervisorAuthorityRefusedBeforeSpawn(t *testing.T) {
	for _, kind := range []string{"missing", "forged-reference", "foreign-graph", "wrong-preview", "wrong-scope"} {
		t.Run(kind, func(t *testing.T) {
			s, r, store, prep, runner := supervisorFixture(t)
			switch kind {
			case "missing":
				store.missing = true
			case "forged-reference":
				r.AuthorityRef = strings.Repeat("b", 64)
			case "foreign-graph":
				r.ParentID = "00000000-0000-4000-8000-000000000099"
			case "wrong-preview":
				r.PreviewDigest = strings.Repeat("b", 64)
			case "wrong-scope":
				store.a.Scope = "retry"
			}
			if e := s.Run(context.Background(), r); e == nil || runner.calls.Load() != 0 || prep.prepares != 0 || store.saves != 0 {
				t.Fatal("authority bypass", e)
			}
		})
	}
}
func TestSupervisorFailureInjectionPhases(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*Supervisor, *fakeSupervisorStore, *fakeSupervisorPreparation, *fakeSupervisorRunner)
	}{
		{"lease-held", func(_ *Supervisor, f *fakeSupervisorStore, _ *fakeSupervisorPreparation, _ *fakeSupervisorRunner) {
			f.held = true
		}},
		{"record-load", func(_ *Supervisor, f *fakeSupervisorStore, _ *fakeSupervisorPreparation, _ *fakeSupervisorRunner) {
			f.failRecords = true
		}},
		{"prepare-before-spawn", func(_ *Supervisor, _ *fakeSupervisorStore, p *fakeSupervisorPreparation, _ *fakeSupervisorRunner) {
			p.failPrepare = 1
		}},
		{"initial-save", func(_ *Supervisor, f *fakeSupervisorStore, _ *fakeSupervisorPreparation, _ *fakeSupervisorRunner) {
			f.failSave = 1
		}},
		{"second-initial-save", func(_ *Supervisor, f *fakeSupervisorStore, _ *fakeSupervisorPreparation, r *fakeSupervisorRunner) {
			f.failSave = 2
			r.waitCancel = true
		}},
		{"live-ingestion", func(_ *Supervisor, _ *fakeSupervisorStore, p *fakeSupervisorPreparation, r *fakeSupervisorRunner) {
			p.failDrain = true
			r.waitCancel = true
		}},
		{"final-drain", func(_ *Supervisor, _ *fakeSupervisorStore, p *fakeSupervisorPreparation, _ *fakeSupervisorRunner) {
			p.failFinal = true
		}},
		{"terminal-save", func(_ *Supervisor, f *fakeSupervisorStore, _ *fakeSupervisorPreparation, r *fakeSupervisorRunner) {
			f.failSave = 3
			r.barrier = make(chan struct{})
		}},
		{"drift-between-children", func(_ *Supervisor, _ *fakeSupervisorStore, p *fakeSupervisorPreparation, r *fakeSupervisorRunner) {
			p.failPrepare = 2
			r.waitCancel = true
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, r, store, p, runner := supervisorFixture(t)
			tc.setup(&s, store, p, runner)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if e := s.Run(ctx, r); e == nil {
				t.Fatal("injection ignored")
			}
			if !store.held && !store.lease.closed {
				t.Fatal("lease leaked")
			}
			for _, c := range store.g.Children {
				for _, a := range c.Attempts {
					if a.Status != executiongraph.AttemptRunning && a.ClassificationReason == "" {
						t.Fatal("partial terminal save")
					}
				}
			}
			if e := executiongraph.ValidateSuccessorGraph(store.g); e != nil {
				t.Fatal("invalid persisted truth", e)
			}
			for _, c := range store.g.Children {
				if len(c.Attempts) > 0 && c.Attempts[0].Status == executiongraph.AttemptRunning {
					store.failSave = 0
					p.failFinal = false
					calls := runner.calls.Load()
					if e := s.Run(ctx, r); !errors.Is(e, ErrSupervisorRecovery) || runner.calls.Load() != calls {
						t.Fatal("stale work redispatched", e)
					}
					break
				}
			}
		})
	}
}
func TestSupervisorCancellationInFlight(t *testing.T) {
	s, r, store, p, runner := supervisorFixture(t)
	runner.waitCancel = true
	// The fake owns all mutation on the supervisor goroutine: the first drain
	// poll installs the already-authorized cancellation snapshot on next load.
	p.onPrepare = func() {
		if p.prepares == 2 {
			store.g.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000999", ParentID: r.ParentID, GraphRevision: 1, ActorKind: "human-operator", Actor: "fake-operator", RequestedAt: time.Unix(20, 0).UTC(), AuthorityRef: strings.Repeat("c", 64)}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := s.Run(ctx, r); e != nil {
		t.Fatal(e)
	}
	for _, c := range store.g.Children {
		for _, a := range c.Attempts {
			if a.Status != executiongraph.AttemptCancelled || a.ClassificationReason != executiongraph.CancelConfirmed {
				t.Fatal(a)
			}
		}
	}
}

type fakeDetacher struct {
	calls int
	err   error
	args  []string
}

func (f *fakeDetacher) Start(_ context.Context, args []string) (executiongraph.ProcessCorrelation, error) {
	f.calls++
	f.args = append([]string(nil), args...)
	return executiongraph.ProcessCorrelation{PID: 9, StartIdentity: "fake", GroupIdentity: "fake"}, f.err
}
func TestDetachedLaunchUsesOnlyCommittedScope(t *testing.T) {
	_, r, store, _, _ := supervisorFixture(t)
	d := &fakeDetacher{}
	// No initiating context is inherited into Run; the detacher receives only
	// immutable identity arguments and no authority/configuration flags.
	got, e := Launch(context.Background(), r, store, d)
	if e != nil || got.PID != 9 || len(d.args) != 5 || d.args[0] != "__stage-supervisor" {
		t.Fatal(got, e)
	}
	d.err = ErrDetachmentUnsupported
	if _, e = Launch(context.Background(), r, store, d); !errors.Is(e, ErrDetachmentUnsupported) {
		t.Fatal(e)
	}
	calls := d.calls
	store.missing = true
	if _, e = Launch(context.Background(), r, store, d); e == nil || d.calls != calls {
		t.Fatal("uncommitted spawn")
	}
}

func TestSupervisorInterruptedAtSaveBoundariesLeavesPersistedTruth(t *testing.T) {
	for _, phase := range []int{1, 2, 3, 4} {
		t.Run(fmt.Sprintf("save-%d", phase), func(t *testing.T) {
			s, r, store, _, runner := supervisorFixture(t)
			runner.barrier = make(chan struct{})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store.onSave = func(n int) {
				if n == phase {
					cancel()
				}
			}
			if e := s.Run(ctx, r); e == nil {
				t.Fatal("interruption ignored")
			}
			if e := executiongraph.ValidateSuccessorGraph(store.g); e != nil {
				t.Fatal("partial truth", e)
			}
			if !store.lease.closed {
				t.Fatal("lease leaked")
			}
			for _, c := range store.g.Children {
				for _, a := range c.Attempts {
					if a.Status == executiongraph.AttemptRunning {
						calls := runner.calls.Load()
						store.onSave = nil
						if e := s.Run(context.Background(), r); !errors.Is(e, ErrSupervisorRecovery) || runner.calls.Load() != calls {
							t.Fatal("restart reinterpreted interrupted work", e)
						}
						return
					}
				}
			}
		})
	}
}
func TestSupervisorInitiatingContextDoesNotOwnRun(t *testing.T) {
	s, r, store, _, runner := supervisorFixture(t)
	initiator, kill := context.WithCancel(context.Background())
	detacher := &fakeDetacher{}
	if _, e := Launch(initiator, r, store, detacher); e != nil {
		t.Fatal(e)
	}
	kill()
	if e := s.Run(context.Background(), r); e != nil || runner.calls.Load() != 2 {
		t.Fatal("run tied to dead initiating context", e)
	}
}
func TestSupervisorCancellationBeforeDispatchDoesNotCreateAttempts(t *testing.T) {
	s, r, store, p, runner := supervisorFixture(t)
	store.g.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000999", ParentID: r.ParentID, GraphRevision: 1, ActorKind: "human-operator", Actor: "fake-operator", RequestedAt: time.Unix(20, 0).UTC(), AuthorityRef: strings.Repeat("c", 64)}
	if e := s.Run(context.Background(), r); e != nil || runner.calls.Load() != 0 || p.prepares != 0 {
		t.Fatal("cancel ignored", e)
	}
}

func (f *fakeSupervisorStore) Contract(context.Context, SupervisorRequest) (Contract, error) {
	return f.contract, nil
}

func TestSupervisorVersionGuardRunsBeforeLaunchAndDispatch(t *testing.T) {
	for _, kind := range []string{"legacy-execution", "local-v1", "unselected-workflow"} {
		t.Run(kind, func(t *testing.T) {
			s, r, store, prep, runner := supervisorFixture(t)
			switch kind {
			case "legacy-execution":
				store.contract.Execution.FormatVersion = 2
			case "local-v1":
				store.contract.Configuration.FormatVersion = 1
			case "unselected-workflow":
				store.contract.WorkflowSelected = false
			}
			detacher := &fakeDetacher{}
			if _, e := Launch(context.Background(), r, store, detacher); e == nil || detacher.calls != 0 {
				t.Fatal("legacy launch", e)
			}
			if e := s.Run(context.Background(), r); e == nil || runner.calls.Load() != 0 || prep.prepares != 0 {
				t.Fatal("legacy dispatch", e)
			}
		})
	}
}
