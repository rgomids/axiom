package stageexecution

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// fakeLifecycleStore exposes the supervisor fake's persisted graph to status
// and cancel. It owns no process and performs no Runtime or Provider effect.
type fakeLifecycleStore struct {
	sup     *fakeSupervisorStore
	cancels int
	fail    bool
}

func (f *fakeLifecycleStore) StageSnapshot(context.Context, StatusRequest) (Contract, error) {
	c := f.sup.contract
	g := cloneSupervisorGraph(f.sup.g)
	c.Graph = &executiongraph.GraphDocument{Successor: &g}
	return c, nil
}
func (f *fakeLifecycleStore) RequestCancel(_ context.Context, g executiongraph.SuccessorGraph, r executiongraph.CancelRequest) (executiongraph.SuccessorGraph, error) {
	f.cancels++
	if f.fail {
		return g, errInjected
	}
	if g.StorageRevision != f.sup.g.StorageRevision {
		return g, ErrStalePreview
	}
	next := cloneSupervisorGraph(f.sup.g)
	next.CancelRequest = &r
	if executiongraph.ValidateSuccessorUpdate(f.sup.g, next) != nil {
		return g, executiongraph.ErrInvalidGraph
	}
	wire, e := executiongraph.EncodeSuccessorGraph(next)
	if e != nil {
		return g, e
	}
	next.StorageRevision = sha256.Sum256(wire)
	f.sup.g = cloneSupervisorGraph(next)
	return next, nil
}

type fakeStatusObserver struct {
	held     bool
	leaseErr error
	alive    bool
	calls    int
}

func (f *fakeStatusObserver) LeaseHeld(context.Context, StatusRequest) (bool, error) {
	return f.held, f.leaseErr
}
func (f *fakeStatusObserver) Process(_ context.Context, p executiongraph.ProcessCorrelation) (*executiongraph.ProcessCorrelation, error) {
	f.calls++
	if !f.alive {
		return nil, nil
	}
	return &p, nil
}

func lifecycleTarget(store *fakeSupervisorStore) StatusRequest {
	return StatusRequest{ProjectID: store.g.Children[0].Envelope.Scope.ProjectID, ParentID: store.g.Parent.ExecutionID}
}

// interruptedSupervisor leaves persisted running attempts, as a killed
// supervisor would. No real process exists behind the recorded identity.
func interruptedSupervisor(t *testing.T) (Supervisor, SupervisorRequest, *fakeSupervisorStore) {
	t.Helper()
	s, r, store, _, runner := supervisorFixture(t)
	runner.barrier = make(chan struct{})
	store.failSave = 3
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := s.Run(ctx, r); e == nil {
		t.Fatal("interruption ignored")
	}
	store.failSave = 0
	running := false
	for _, c := range store.g.Children {
		for _, a := range c.Attempts {
			running = running || a.Status == executiongraph.AttemptRunning
		}
	}
	if !running {
		t.Fatal("fixture has no persisted running attempt")
	}
	return s, r, store
}

func TestStatusRendersPersistedStateAcrossStatuses(t *testing.T) {
	cases := map[string]func(*testing.T) *fakeSupervisorStore{
		"not-started": func(t *testing.T) *fakeSupervisorStore { _, _, store, _, _ := supervisorFixture(t); return store },
		"succeeded": func(t *testing.T) *fakeSupervisorStore {
			s, r, store, _, _ := supervisorFixture(t)
			if e := s.Run(context.Background(), r); e != nil {
				t.Fatal(e)
			}
			return store
		},
		"running": func(t *testing.T) *fakeSupervisorStore { _, _, store := interruptedSupervisor(t); return store },
		"cancelled": func(t *testing.T) *fakeSupervisorStore {
			s, r, store, p, runner := supervisorFixture(t)
			runner.waitCancel = true
			p.onPrepare = func() {
				if p.prepares == 2 {
					store.g.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000999", ParentID: r.ParentID, GraphRevision: 1, ActorKind: "human-operator", Actor: "fake-operator", RequestedAt: time.Unix(20, 0).UTC(), AuthorityRef: strings.Repeat("c", 64)}
				}
			}
			if e := s.Run(context.Background(), r); e != nil {
				t.Fatal(e)
			}
			return store
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			store := build(t)
			source := &fakeLifecycleStore{sup: store}
			target := lifecycleTarget(store)
			before, e := StatusService{Source: source}.Read(context.Background(), target)
			if e != nil {
				t.Fatal(e)
			}
			// Restart: the same persisted bytes decoded by a fresh process.
			wire, e := executiongraph.EncodeSuccessorGraph(store.g)
			if e != nil {
				t.Fatal(e)
			}
			reloaded, e := executiongraph.DecodeSuccessorGraph(wire)
			if e != nil {
				t.Fatal(e)
			}
			reloaded.StorageRevision = store.g.StorageRevision
			store.g = reloaded
			after, e := StatusService{Source: source}.Read(context.Background(), target)
			if e != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("status changed across restart", e)
			}
			found := name == "not-started"
			for _, c := range after.Graph.Successor.Children {
				for _, a := range c.Attempts {
					found = found || string(a.Status) == name
					if a.Status != executiongraph.AttemptSucceeded && string(a.Status) != name {
						t.Fatal("status not persisted truth", a.Status)
					}
					if a.Status == executiongraph.AttemptRunning && !after.Health[a.AttemptID].LivenessUnknown {
						t.Fatal("liveness invented without observer")
					}
				}
			}
			if !found {
				t.Fatal("status missing", name)
			}
		})
	}
}

func TestStatusObservationIsDiagnosticOnly(t *testing.T) {
	for _, kind := range []string{"orphan", "stale", "lease-held", "lease-error"} {
		t.Run(kind, func(t *testing.T) {
			_, _, store := interruptedSupervisor(t)
			persisted := cloneSupervisorGraph(store.g)
			o := &fakeStatusObserver{}
			want := ""
			switch kind {
			case "orphan":
				o.alive, want = true, "stop_orphan_then_reconcile"
			case "stale":
				want = "reconcile_required"
			case "lease-held":
				o.held, want = true, "attempt_running"
			case "lease-error":
				o.leaseErr, want = errInjected, "attempt_running"
			}
			result, e := StatusService{Source: &fakeLifecycleStore{sup: store}, Observer: o}.Read(context.Background(), lifecycleTarget(store))
			if e != nil || !reflect.DeepEqual(store.g, persisted) {
				t.Fatal("status mutated state", e)
			}
			for _, c := range result.Graph.Successor.Children {
				for _, a := range c.Attempts {
					if a.Status != executiongraph.AttemptRunning {
						continue
					}
					if result.Next[c.ExecutionID] != want {
						t.Fatal(kind, result.Next)
					}
					h := result.Health[a.AttemptID]
					if kind == "lease-error" && !h.LivenessUnknown || kind == "lease-held" && (h.Stale || o.calls != 0) {
						t.Fatal(kind, h)
					}
				}
			}
		})
	}
}

func TestStatusInspectsLegacyAndRefusesForeignTarget(t *testing.T) {
	_, _, store, _, _ := supervisorFixture(t)
	store.contract.Execution.FormatVersion = 2
	source := &fakeLifecycleStore{sup: store}
	result, e := StatusService{Source: source}.Read(context.Background(), lifecycleTarget(store))
	if e != nil || result.Next[store.g.Parent.ExecutionID] != ErrLegacyExecution.Error() {
		t.Fatal("legacy status not inspectable", result.Next, e)
	}
	store.contract.Execution.FormatVersion = 5
	foreign := lifecycleTarget(store)
	foreign.ProjectID = "foreign-project"
	if _, e = (StatusService{Source: source}).Read(context.Background(), foreign); !errors.Is(e, ErrAdmission) {
		t.Fatal(e)
	}
}

// resumeFixture binds the committed supervisor authority to the fake preview
// so a stage run re-invocation can be checked against fresh state.
func resumeFixture(t *testing.T) (RunService, RunRequest, Supervisor, *fakeSupervisorStore, *fakeRunPorts, *fakePreviewSource) {
	t.Helper()
	service, r, ports, source := runFixture(t, false)
	s, req, store, _, _ := supervisorFixture(t)
	store.a.PreviewDigest = r.PreviewDigest
	digest, e := executiongraph.DispatchAuthorityDigest(store.a)
	if e != nil {
		t.Fatal(e)
	}
	store.g.DispatchAuthorityRef = digest
	for i := range store.g.Children {
		store.g.Children[i].Envelope.AuthorityReference = digest
	}
	wire, e := executiongraph.EncodeSuccessorGraph(store.g)
	if e != nil {
		t.Fatal(e)
	}
	store.g.StorageRevision = sha256.Sum256(wire)
	req.PreviewDigest, req.AuthorityRef = r.PreviewDigest, digest
	service.ResumeStore = store
	r.Committed = &req
	return service, r, s, store, ports, source
}

func TestResumeAfterSupervisorKillStartsOnlyRemainingFirstAttempts(t *testing.T) {
	service, r, s, store, ports, _ := resumeFixture(t)
	s.Store = store
	store.g.StageConcurrency = 1
	wire, e := executiongraph.EncodeSuccessorGraph(store.g)
	if e != nil {
		t.Fatal(e)
	}
	store.g.StorageRevision = sha256.Sum256(wire)
	// Kill after the first child's terminal save, before the second starts.
	ctx, kill := context.WithCancel(context.Background())
	store.onSave = func(n int) {
		if n == 3 {
			kill()
		}
	}
	if e = s.Run(ctx, *r.Committed); e == nil {
		t.Fatal("kill ignored")
	}
	store.onSave = nil
	started := 0
	for _, c := range store.g.Children {
		started += len(c.Attempts)
	}
	if started != 1 {
		t.Fatal("fixture did not leave remaining first attempts", started)
	}
	persisted := cloneSupervisorGraph(store.g)
	result, e := service.Apply(context.Background(), r)
	if e != nil || result.Status != "success" || result.Process == nil || !reflect.DeepEqual(ports.events, []string{"workspaces", "start"}) {
		t.Fatal(result, e, ports.events)
	}
	if !reflect.DeepEqual(store.g, persisted) {
		t.Fatal("resume mutated persisted state before the supervisor")
	}
	runner := s.Runner.(*fakeSupervisorRunner)
	calls := runner.calls.Load()
	if e = s.Run(context.Background(), *r.Committed); e != nil || runner.calls.Load() != calls+1 {
		t.Fatal("resumed supervisor redispatched or skipped", e)
	}
}

func TestResumeRefusals(t *testing.T) {
	for _, kind := range []string{"stale-attempt", "orphan-attempt", "policy-drift", "consumed-scope", "cancel-requested", "no-external", "foreign-preview", "legacy"} {
		t.Run(kind, func(t *testing.T) {
			service, r, s, store, ports, source := resumeFixture(t)
			want := error(ErrSupervisorAuthority)
			switch kind {
			case "stale-attempt", "orphan-attempt":
				// Running persisted attempts need reconcile whether or not the
				// recorded process still lives; resume never observes liveness.
				_, _, interrupted := interruptedSupervisor(t)
				store.g.Children = interrupted.g.Children
				for i := range store.g.Children {
					store.g.Children[i].Envelope.AuthorityReference = store.g.DispatchAuthorityRef
				}
				wire, e := executiongraph.EncodeSuccessorGraph(store.g)
				if e != nil {
					t.Fatal(e)
				}
				store.g.StorageRevision = sha256.Sum256(wire)
				want = ErrSupervisorRecovery
			case "policy-drift":
				source.o.PolicyDigest = strings.Repeat("f", 64)
				want = ErrStalePreview
			case "consumed-scope":
				s.Store = store
				if e := s.Run(context.Background(), *r.Committed); e != nil {
					t.Fatal(e)
				}
			case "cancel-requested":
				store.g.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000999", ParentID: store.g.Parent.ExecutionID, GraphRevision: 1, ActorKind: "human-operator", Actor: "fake-operator", RequestedAt: time.Unix(20, 0).UTC(), AuthorityRef: strings.Repeat("c", 64)}
			case "no-external":
				r.AuthorizeExternal = false
				want = ErrDispatchAuthority
			case "foreign-preview":
				r.PreviewDigest = strings.Repeat("f", 64)
			case "legacy":
				store.contract.Execution.FormatVersion = 2
				want = ErrLegacyExecution
			}
			saves := store.saves
			result, e := service.Apply(context.Background(), r)
			if !errors.Is(e, want) || result.Status == "success" || result.Process != nil {
				t.Fatal(kind, result, e)
			}
			for _, ev := range ports.events {
				if ev == "start" {
					t.Fatal("supervisor started", kind)
				}
			}
			if store.saves != saves {
				t.Fatal("resume wrote state")
			}
		})
	}
}

func TestCancelBeforeDispatchPreventsAttempts(t *testing.T) {
	s, r, store, p, runner := supervisorFixture(t)
	lifecycle := &fakeLifecycleStore{sup: store}
	c := CancelService{Store: lifecycle, AllocateID: func() (string, error) { return "00000000-0000-4000-8000-000000000777", nil }, Now: func() time.Time { return time.Unix(30, 0).UTC() }}
	preview, e := c.Preview(context.Background(), lifecycleTarget(store))
	if e != nil {
		t.Fatal(e)
	}
	result, e := c.Apply(context.Background(), CancelApply{Preview: preview, AuthorizeLocal: true, Actor: "operator"})
	if e != nil || result.Category != "cancellation_requested" || result.Request == nil || store.g.CancelRequest == nil || *store.g.CancelRequest != *result.Request {
		t.Fatal(result, e)
	}
	if e = s.Run(context.Background(), r); e != nil || runner.calls.Load() != 0 || p.prepares != 0 {
		t.Fatal("dispatch after cancel", e)
	}
	for _, child := range store.g.Children {
		if len(child.Attempts) != 0 {
			t.Fatal("attempt created after cancel")
		}
	}
}

func TestCancelWhileRunningRequiresConfirmedStop(t *testing.T) {
	for _, survives := range []bool{false, true} {
		t.Run(map[bool]string{false: "confirmed-stop", true: "descendant-survives"}[survives], func(t *testing.T) {
			s, r, store, p, runner := supervisorFixture(t)
			runner.waitCancel = true
			if survives {
				runner.waitCancel = false
				runner.barrier = make(chan struct{})
				s.Runner = survivingRunner{runner}
			}
			c := CancelService{Store: &fakeLifecycleStore{sup: store}, AllocateID: func() (string, error) { return "00000000-0000-4000-8000-000000000777", nil }, Now: func() time.Time { return time.Unix(20, 0).UTC() }}
			p.onPrepare = func() {
				if p.prepares != 2 {
					return
				}
				preview, e := c.Preview(context.Background(), lifecycleTarget(store))
				if e == nil {
					_, e = c.Apply(context.Background(), CancelApply{Preview: preview, AuthorizeLocal: true, Actor: "operator"})
				}
				if e != nil {
					t.Error(e)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if e := s.Run(ctx, r); e != nil {
				t.Log("run:", e)
			}
			seen := 0
			for _, child := range store.g.Children {
				for _, a := range child.Attempts {
					seen++
					if a.Status == executiongraph.AttemptCancelled && (survives || a.ClassificationReason != executiongraph.CancelConfirmed) {
						t.Fatal("unobserved stop reported cancelled", a)
					}
					if survives && a.Status != executiongraph.AttemptUnknown && a.Status != executiongraph.AttemptRunning {
						t.Fatal("surviving descendant not unknown", a)
					}
					if !survives && a.Status != executiongraph.AttemptCancelled {
						t.Fatal("confirmed stop not cancelled", a)
					}
				}
			}
			if seen == 0 {
				t.Fatal("no attempt observed")
			}
		})
	}
}

// survivingRunner reports that the group stop could not be confirmed.
type survivingRunner struct{ inner *fakeSupervisorRunner }

func (r survivingRunner) RunObserved(ctx context.Context, inv executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	count := r.inner.calls.Add(1)
	if e := started(&executiongraph.ProcessCorrelation{PID: int64(count), StartIdentity: "fake-start", GroupIdentity: "fake-group"}); e != nil {
		return executiongraph.ProcessResult{OutcomeUnknown: true}
	}
	<-ctx.Done()
	return executiongraph.ProcessResult{OutcomeUnknown: true, Stopped: false}
}

func TestCancelRefusalsAndIdempotence(t *testing.T) {
	t.Run("supervisor-absent", func(t *testing.T) {
		_, _, store := interruptedSupervisor(t)
		lifecycle := &fakeLifecycleStore{sup: store}
		c := CancelService{Store: lifecycle, AllocateID: func() (string, error) { return "00000000-0000-4000-8000-000000000777", nil }}
		preview, e := c.Preview(context.Background(), lifecycleTarget(store))
		if e != nil {
			t.Fatal(e)
		}
		result, e := c.Apply(context.Background(), CancelApply{Preview: preview, AuthorizeLocal: true, Actor: "operator"})
		if e != nil || result.Category != "cancellation_requested" {
			t.Fatal(result, e)
		}
		// Only the request is published; nobody observed a stop.
		for _, child := range store.g.Children {
			for _, a := range child.Attempts {
				if a.Status == executiongraph.AttemptCancelled {
					t.Fatal("cancel invented a stop")
				}
			}
		}
		// Repeated cancel returns the same request without another write.
		preview, e = c.Preview(context.Background(), lifecycleTarget(store))
		if e != nil {
			t.Fatal(e)
		}
		again, e := c.Apply(context.Background(), CancelApply{Preview: preview, AuthorizeLocal: true, Actor: "operator"})
		if e != nil || again.Request == nil || *again.Request != *result.Request || lifecycle.cancels != 1 {
			t.Fatal("repeated cancel rewrote request", again, e, lifecycle.cancels)
		}
	})
	for _, kind := range []string{"no-authority", "no-actor", "stale-preview", "legacy", "write-failure"} {
		t.Run(kind, func(t *testing.T) {
			_, _, store, _, _ := supervisorFixture(t)
			lifecycle := &fakeLifecycleStore{sup: store}
			c := CancelService{Store: lifecycle, AllocateID: func() (string, error) { return "00000000-0000-4000-8000-000000000777", nil }}
			preview, e := c.Preview(context.Background(), lifecycleTarget(store))
			if e != nil {
				t.Fatal(e)
			}
			apply := CancelApply{Preview: preview, AuthorizeLocal: true, Actor: "operator"}
			switch kind {
			case "no-authority":
				apply.AuthorizeLocal = false
			case "no-actor":
				apply.Actor = ""
			case "stale-preview":
				store.g.StorageRevision[0] ^= 1
			case "legacy":
				store.contract.Execution.FormatVersion = 2
			case "write-failure":
				lifecycle.fail = true
			}
			result, e := c.Apply(context.Background(), apply)
			if e == nil || result.Category == "cancellation_requested" || store.g.CancelRequest != nil {
				t.Fatal(kind, result, e)
			}
		})
	}
}
