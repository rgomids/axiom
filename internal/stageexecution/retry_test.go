package stageexecution

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
)

// scriptedRunner returns a fixed synthetic process result; nothing executes.
type scriptedRunner struct {
	calls  atomic.Int32
	result executiongraph.ProcessResult
}

func (r *scriptedRunner) RunObserved(_ context.Context, _ executiongraph.Invocation, started func(*executiongraph.ProcessCorrelation) error) executiongraph.ProcessResult {
	count := r.calls.Add(1)
	if e := started(&executiongraph.ProcessCorrelation{PID: int64(count), StartIdentity: "fake-start", GroupIdentity: "fake-group"}); e != nil {
		return executiongraph.ProcessResult{OutcomeUnknown: true}
	}
	return r.result
}

type fakeRetryStore struct {
	*fakeSupervisorStore
	receipts map[string]executiongraph.RetryAuthority
	commits  int
}

func (f *fakeRetryStore) LoadRetry(_ context.Context, _ SupervisorRequest, ref string) (executiongraph.RetryAuthority, error) {
	a, ok := f.receipts[ref]
	if !ok {
		return a, ErrSupervisorAuthority
	}
	return a, nil
}
func (f *fakeRetryStore) StageSnapshot(context.Context, StatusRequest) (Contract, error) {
	c := f.contract
	g := cloneSupervisorGraph(f.g)
	c.Graph = &executiongraph.GraphDocument{Successor: &g}
	return c, nil
}
func (f *fakeRetryStore) CommitRetry(_ context.Context, g executiongraph.SuccessorGraph, a executiongraph.RetryAuthority) (string, error) {
	f.commits++
	if g.StorageRevision != f.g.StorageRevision {
		return "", ErrStalePreview
	}
	ref, e := executiongraph.RetryAuthorityDigest(a)
	if e != nil {
		return "", e
	}
	f.receipts[ref] = a
	return ref, nil
}

type fakeDisposer struct {
	dispositions []string
	err          error
}

func (f *fakeDisposer) Dispose(_ context.Context, _ executiongraph.SuccessorGraph, _ executiongraph.SuccessorChild, d string) error {
	f.dispositions = append(f.dispositions, d)
	return f.err
}

func TestRetryDispositionFailureRequiresRecovery(t *testing.T) {
	h := retryFixture(t, "unknown", 3)
	appendRetrySafeFact(t, h)
	blocker := errors.New("synthetic authority fencing unavailable")
	h.disposer.err = blocker
	p, err := h.service.Preview(context.Background(), h.request)
	if err != nil {
		t.Fatal(err)
	}
	result, err := h.service.Apply(context.Background(), RetryApply{Preview: p, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "operator", Disposition: executiongraph.DispositionReset})
	if !errors.Is(err, blocker) || result.Status != "recovery_required" || result.Recovery == "" || result.AuthorityRef == "" {
		t.Fatal(result, err)
	}
	if h.store.commits != 1 || len(h.starter.refs) != 0 || len(h.store.g.Children[0].Attempts) != 1 {
		t.Fatal("refusal started or changed an attempt")
	}
	if _, ok := h.store.receipts[result.AuthorityRef]; !ok {
		t.Fatal("committed receipt lost")
	}
}

func TestRetryPreserveDisposition(t *testing.T) {
	h := retryFixture(t, "unknown", 3)
	appendRetrySafeFact(t, h)
	p, err := h.service.Preview(context.Background(), h.request)
	if err != nil {
		t.Fatal(err)
	}
	result, err := h.service.Apply(context.Background(), RetryApply{Preview: p, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "operator", Disposition: executiongraph.DispositionPreserve})
	if err != nil || result.Status != "success" || len(h.disposer.dispositions) != 1 || h.disposer.dispositions[0] != executiongraph.DispositionPreserve {
		t.Fatal(result, err)
	}
}

type fakeReset struct {
	children []string
	before   func()
	err      error
}

func (f *fakeReset) ResetChild(ctx context.Context, _ executiongraph.SuccessorGraph, id string, guard func(context.Context) error) error {
	if f.before != nil {
		f.before()
	}
	if err := guard(ctx); err != nil {
		return err
	}
	f.children = append(f.children, id)
	if f.err != nil {
		return f.err
	}
	return guard(ctx)
}

func TestRetryResetRunsOnlyInsideTheFencedSupervisor(t *testing.T) {
	prepare := func(t *testing.T) (retryHarness, string) {
		h := retryFixture(t, "unknown", 3)
		appendRetrySafeFact(t, h)
		p, err := h.service.Preview(context.Background(), h.request)
		if err != nil {
			t.Fatal(err)
		}
		result, err := h.service.Apply(context.Background(), RetryApply{Preview: p, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "operator", Disposition: executiongraph.DispositionReset})
		if err != nil || result.Status != "success" {
			t.Fatal(result, err)
		}
		return h, result.AuthorityRef
	}
	t.Run("missing port", func(t *testing.T) {
		h, ref := prepare(t)
		h.sup.Reset = nil
		calls := h.runner.calls.Load()
		if err := h.sup.RunRetry(context.Background(), h.supReq, ref); !errors.Is(err, gitworkspace.ErrRetryResetFencingUnavailable) {
			t.Fatal(err)
		}
		if h.runner.calls.Load() != calls || len(h.store.g.Children[0].Attempts) != 1 {
			t.Fatal("unfenced reset started an attempt")
		}
	})
	t.Run("ledger changed under the fence", func(t *testing.T) {
		h, ref := prepare(t)
		reset := &fakeReset{}
		reset.before = func() { h.store.g.StorageRevision[0] ^= 1 }
		h.sup.Reset = reset
		calls := h.runner.calls.Load()
		if err := h.sup.RunRetry(context.Background(), h.supReq, ref); err == nil || len(reset.children) != 0 || h.runner.calls.Load() != calls {
			t.Fatalf("err=%v reset=%v", err, reset.children)
		}
	})
	t.Run("effect failure", func(t *testing.T) {
		h, ref := prepare(t)
		blocker := errors.New("synthetic reset failure")
		h.sup.Reset = &fakeReset{err: blocker}
		calls := h.runner.calls.Load()
		if err := h.sup.RunRetry(context.Background(), h.supReq, ref); !errors.Is(err, blocker) || h.runner.calls.Load() != calls || len(h.store.g.Children[0].Attempts) != 1 {
			t.Fatal(err)
		}
		reset := &fakeReset{}
		h.sup.Reset = reset
		if err := h.sup.RunRetry(context.Background(), h.supReq, ref); err != nil || len(reset.children) != 1 || len(h.store.g.Children[0].Attempts) != 2 {
			t.Fatalf("pending receipt did not recover: %v", err)
		}
	})
}

type fakeRetryStarter struct{ refs []string }

func (f *fakeRetryStarter) StartRetry(_ context.Context, _ SupervisorRequest, ref string) (executiongraph.ProcessCorrelation, error) {
	f.refs = append(f.refs, ref)
	return executiongraph.ProcessCorrelation{PID: 77, StartIdentity: "fake-retry", GroupIdentity: "fake-group"}, nil
}

type retryHarness struct {
	service  RetryService
	request  RetryRequest
	sup      Supervisor
	supReq   SupervisorRequest
	store    *fakeRetryStore
	disposer *fakeDisposer
	starter  *fakeRetryStarter
	source   *fakePreviewSource
	prep     *fakeSupervisorPreparation
	runner   *scriptedRunner
}

// retryFixture runs first attempts to the named outcome with fakes, then
// prepares a retry service over the persisted graph.
func retryFixture(t *testing.T, outcome string, maximum uint32) retryHarness {
	t.Helper()
	run, runReq, _, source := runFixture(t, false)
	s, r, sup, prep, _ := supervisorFixture(t)
	for i := range sup.g.Children {
		sup.g.Children[i].Envelope.Controls.MaximumAttempts = maximum
	}
	wire, e := executiongraph.EncodeSuccessorGraph(sup.g)
	if e != nil {
		t.Fatal(e)
	}
	sup.g.StorageRevision = sha256.Sum256(wire)
	runner := &scriptedRunner{result: executiongraph.ProcessResult{Output: []byte("fake"), Stopped: true}}
	switch outcome {
	case "failed":
		prep.results = false
	case "blocked":
		prep.kind = "blocker"
	case "unknown":
		runner.result = executiongraph.ProcessResult{EffectAmbiguous: true, Stopped: true}
	case "succeeded":
	}
	s.Runner = runner
	store := &fakeRetryStore{fakeSupervisorStore: sup, receipts: map[string]executiongraph.RetryAuthority{}}
	s.Store = store
	if e = s.Run(context.Background(), r); e != nil {
		t.Fatal(e)
	}
	prep.results, prep.kind = true, ""
	runner.result = executiongraph.ProcessResult{Output: []byte("fake"), Stopped: true}
	child := sup.g.Children[0]
	if len(child.Attempts) != 1 || outcome != "succeeded" && string(child.Attempts[0].Status) != outcome {
		t.Fatal("fixture outcome", child.Attempts)
	}
	disposer, starter := &fakeDisposer{}, &fakeRetryStarter{}
	service := RetryService{Store: store, Dispatch: run.Preview, Disposer: disposer, Starter: starter, Now: func() time.Time { return time.Unix(50, 0).UTC() }, Authority: func(context.Context, StatusRequest) (SupervisorRequest, error) { return r, nil }}
	request := RetryRequest{Target: lifecycleTarget(sup), Execution: runReq.Execution, StageID: runReq.StageID, ChildID: child.ExecutionID}
	s.Now = func() time.Time { return time.Unix(60, 0).UTC() }
	return retryHarness{service, request, s, r, store, disposer, starter, source, prep, runner}
}

func appendRetrySafeFact(t *testing.T, h retryHarness) {
	t.Helper()
	g := h.store.g
	a := g.Children[0].Attempts[0]
	o := executiongraph.ReconciliationObservations{ParentID: g.Parent.ExecutionID, ChildID: g.Children[0].ExecutionID, GraphRevision: g.Parent.GraphRevision, AttemptID: a.AttemptID, LeaseEpoch: a.LeaseEpoch, WorktreeDigest: strings.Repeat("1", 64), CanonicalRecordsDigest: strings.Repeat("2", 64), QuarantinedInboxDigest: strings.Repeat("3", 64)}
	d, e := o.Digest()
	if e != nil {
		t.Fatal(e)
	}
	f := executiongraph.ReconciliationFact{FactID: "00000000-0000-4000-8000-000000000555", ParentID: o.ParentID, ChildID: o.ChildID, GraphRevision: o.GraphRevision, AttemptID: o.AttemptID, ObservationsDigest: d, Classification: "retry_safe", AuthorityRef: strings.Repeat("d", 64), ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(40, 0).UTC()}
	next, e := executiongraph.AppendReconciliationFact(g, f, o)
	if e != nil {
		t.Fatal(e)
	}
	wire, e := executiongraph.EncodeSuccessorGraph(next)
	if e != nil {
		t.Fatal(e)
	}
	next.StorageRevision = sha256.Sum256(wire)
	h.store.g = next
}

func TestRetryEligibleStatusesCreateNewAttempt(t *testing.T) {
	for _, outcome := range []string{"failed", "blocked", "unknown"} {
		t.Run(outcome, func(t *testing.T) {
			h := retryFixture(t, outcome, 3)
			disposition := ""
			if outcome == "unknown" {
				appendRetrySafeFact(t, h)
				disposition = executiongraph.DispositionReset
			}
			p, e := h.service.Preview(context.Background(), h.request)
			if e != nil {
				t.Fatal(e)
			}
			if p.AttemptNumber != 2 || p.Remaining != 2 || p.DispositionRequired != (outcome == "unknown") {
				t.Fatal(p)
			}
			result, e := h.service.Apply(context.Background(), RetryApply{Preview: p, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "operator", Disposition: disposition})
			if e != nil || result.Status != "success" || result.Authority == nil || len(h.starter.refs) != 1 || h.starter.refs[0] != result.AuthorityRef {
				t.Fatal(result, e)
			}
			if outcome == "unknown" && (len(h.disposer.dispositions) != 1 || h.disposer.dispositions[0] != executiongraph.DispositionReset) {
				t.Fatal("disposition not checked", h.disposer.dispositions)
			}
			reset := &fakeReset{}
			h.sup.Reset = reset
			if outcome != "unknown" && len(h.disposer.dispositions) != 0 {
				t.Fatal("implicit workspace effect")
			}
			// Apply itself never creates the attempt; the supervisor does.
			if len(h.store.g.Children[0].Attempts) != 1 {
				t.Fatal("apply created attempt")
			}
			calls := h.runner.calls.Load()
			if e = h.sup.RunRetry(context.Background(), h.supReq, result.AuthorityRef); e != nil {
				t.Fatal(e)
			}
			attempts := h.store.g.Children[0].Attempts
			if len(attempts) != 2 || attempts[1].Number != 2 || attempts[1].Status != executiongraph.AttemptSucceeded || h.runner.calls.Load() != calls+1 {
				t.Fatal("retry attempt", attempts)
			}
			if attempts[0].Status != executiongraph.AttemptStatus(outcome) {
				t.Fatal("prior attempt rewritten")
			}
			if wantReset := outcome == "unknown"; wantReset != (len(reset.children) == 1) || wantReset && reset.children[0] != h.store.g.Children[0].ExecutionID {
				t.Fatal("reset", reset.children)
			}
			// The receipt is consumed by the attempt it authorized.
			if e = h.sup.RunRetry(context.Background(), h.supReq, result.AuthorityRef); e == nil || h.runner.calls.Load() != calls+1 {
				t.Fatal("retry receipt reused", e)
			}
		})
	}
}

func TestRetryRefusals(t *testing.T) {
	cases := []struct {
		name, outcome string
		maximum       uint32
		want          error
		change        func(*testing.T, *retryHarness, *RetryApply)
	}{
		{"never-dispatched", "succeeded", 3, ErrRetryIneligible, func(_ *testing.T, h *retryHarness, _ *RetryApply) {
			h.request.ChildID = h.store.g.Children[1].ExecutionID
		}},
		{"succeeded", "succeeded", 3, ErrRetryIneligible, nil},
		{"exhausted", "failed", 1, ErrRetryExhausted, nil},
		{"unknown-unreconciled", "unknown", 3, ErrRetryIneligible, nil},
		{"stale-capability", "failed", 3, ErrStalePreview, func(_ *testing.T, h *retryHarness, _ *RetryApply) {
			c := h.service.Dispatch.Capabilities.(*fakeCapabilities)
			for k, v := range c.views {
				v.EvidenceRef = "changed"
				c.views[k] = v
			}
		}},
		{"legacy-execution", "failed", 3, ErrLegacyExecution, func(_ *testing.T, h *retryHarness, _ *RetryApply) {
			h.store.contract.Execution.FormatVersion = 2
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := retryFixture(t, tc.outcome, tc.maximum)
			if tc.change != nil {
				tc.change(t, &h, nil)
			}
			if _, e := h.service.Preview(context.Background(), h.request); !errors.Is(e, tc.want) {
				t.Fatal(tc.name, e)
			}
			if h.store.commits != 0 || len(h.starter.refs) != 0 {
				t.Fatal("retry effects")
			}
		})
	}
	applyCases := []struct {
		name, outcome string
		want          error
		change        func(*testing.T, *retryHarness, *RetryApply)
	}{
		{"missing-disposition", "unknown", ErrDispositionRequired, nil},
		{"invented-disposition", "unknown", ErrDispositionRequired, func(_ *testing.T, _ *retryHarness, a *RetryApply) { a.Disposition = "wipe" }},
		{"disposition-without-effects", "failed", ErrDispositionRequired, func(_ *testing.T, _ *retryHarness, a *RetryApply) { a.Disposition = executiongraph.DispositionReset }},
		{"no-external", "failed", ErrDispatchAuthority, func(_ *testing.T, _ *retryHarness, a *RetryApply) { a.AuthorizeExternal = false }},
		{"no-local", "failed", ErrDispatchAuthority, func(_ *testing.T, _ *retryHarness, a *RetryApply) { a.AuthorizeLocal = false }},
		{"policy-drift", "failed", ErrStalePreview, func(_ *testing.T, h *retryHarness, _ *RetryApply) { h.source.o.PolicyDigest = strings.Repeat("f", 64) }},
		{"graph-drift", "failed", ErrStalePreview, func(_ *testing.T, h *retryHarness, _ *RetryApply) { h.store.g.StorageRevision[0] ^= 1 }},
		{"cancel-requested", "failed", ErrRetryIneligible, func(_ *testing.T, h *retryHarness, _ *RetryApply) {
			h.store.g.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000999", ParentID: h.store.g.Parent.ExecutionID, GraphRevision: 1, ActorKind: "human-operator", Actor: "fake-operator", RequestedAt: time.Unix(45, 0).UTC(), AuthorityRef: strings.Repeat("c", 64)}
		}},
	}
	for _, tc := range applyCases {
		t.Run(tc.name, func(t *testing.T) {
			h := retryFixture(t, tc.outcome, 3)
			if tc.outcome == "unknown" {
				appendRetrySafeFact(t, h)
			}
			p, e := h.service.Preview(context.Background(), h.request)
			if e != nil {
				t.Fatal(e)
			}
			apply := RetryApply{Preview: p, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "operator"}
			if tc.change != nil {
				tc.change(t, &h, &apply)
			}
			result, e := h.service.Apply(context.Background(), apply)
			if !errors.Is(e, tc.want) || result.Status == "success" {
				t.Fatal(tc.name, result, e)
			}
			if h.store.commits != 0 || len(h.starter.refs) != 0 || len(h.disposer.dispositions) != 0 {
				t.Fatal("retry effects after refusal")
			}
		})
	}
}

func TestRetrySupervisorRefusesForeignReceipts(t *testing.T) {
	h := retryFixture(t, "failed", 3)
	p, e := h.service.Preview(context.Background(), h.request)
	if e != nil {
		t.Fatal(e)
	}
	result, e := h.service.Apply(context.Background(), RetryApply{Preview: p, AuthorizeLocal: true, AuthorizeExternal: true, Actor: "operator"})
	if e != nil {
		t.Fatal(e)
	}
	calls := h.runner.calls.Load()
	// First-attempt authority digest is not a retry receipt.
	if e = h.sup.RunRetry(context.Background(), h.supReq, h.supReq.AuthorityRef); e == nil {
		t.Fatal("first-attempt authority used for retry")
	}
	// A tampered receipt no longer matches its content address.
	forged := *result.Authority
	forged.AttemptNumber = 3
	h.store.receipts[result.AuthorityRef] = forged
	if e = h.sup.RunRetry(context.Background(), h.supReq, result.AuthorityRef); e == nil {
		t.Fatal("forged receipt accepted")
	}
	// A write after apply (here: cancellation) consumes the receipt.
	h.store.receipts[result.AuthorityRef] = *result.Authority
	h.store.g.CancelRequest = &executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000999", ParentID: h.store.g.Parent.ExecutionID, GraphRevision: 1, ActorKind: "human-operator", Actor: "fake-operator", RequestedAt: time.Unix(55, 0).UTC(), AuthorityRef: strings.Repeat("c", 64)}
	if e = h.sup.RunRetry(context.Background(), h.supReq, result.AuthorityRef); e == nil || h.runner.calls.Load() != calls {
		t.Fatal("retry after cancel", e)
	}
}
