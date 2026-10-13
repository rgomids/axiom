package stageexecution

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// fakeReconcileObserver returns synthetic digests only; no process is probed.
type fakeReconcileObserver struct {
	held, alive, unknown bool
	worktree             string
	calls                int
}

func (f *fakeReconcileObserver) ObserveReconciliation(_ context.Context, g executiongraph.SuccessorGraph, child, attempt string) (executiongraph.ReconciliationObservations, error) {
	f.calls++
	o := executiongraph.ReconciliationObservations{ParentID: g.Parent.ExecutionID, ChildID: child, GraphRevision: g.Parent.GraphRevision, AttemptID: attempt, LeaseHeld: f.held, ProcessAlive: f.alive, ProcessUnknown: f.unknown, WorktreeDigest: strings.Repeat("1", 64), CanonicalRecordsDigest: strings.Repeat("2", 64), QuarantinedInboxDigest: strings.Repeat("3", 64)}
	if f.worktree != "" {
		o.WorktreeDigest = f.worktree
	}
	for _, c := range g.Children {
		for _, a := range c.Attempts {
			if a.AttemptID == attempt {
				o.LeaseEpoch = a.LeaseEpoch
			}
		}
	}
	return o, nil
}

type fakeReconcileStore struct {
	*fakeLifecycleStore
	observer *fakeReconcileObserver
	appends  int
}

// AppendReconciliation mirrors the local store: re-observe, then append.
func (f *fakeReconcileStore) AppendReconciliation(ctx context.Context, g executiongraph.SuccessorGraph, fact executiongraph.ReconciliationFact) (executiongraph.SuccessorGraph, error) {
	f.appends++
	if g.StorageRevision != f.sup.g.StorageRevision {
		return g, ErrStalePreview
	}
	o, e := f.observer.ObserveReconciliation(ctx, g, fact.ChildID, fact.AttemptID)
	if e != nil {
		return g, e
	}
	next, e := executiongraph.AppendReconciliationFact(g, fact, o)
	if e != nil {
		return g, e
	}
	wire, e := executiongraph.EncodeSuccessorGraph(next)
	if e != nil {
		return g, e
	}
	next.StorageRevision = sha256.Sum256(wire)
	f.sup.g = cloneSupervisorGraph(next)
	return next, nil
}

func reconcileFixture(t *testing.T) (ReconcileService, ReconcileRequest, *fakeReconcileStore, *fakeReconcileObserver) {
	t.Helper()
	_, _, sup := interruptedSupervisor(t)
	observer := &fakeReconcileObserver{}
	store := &fakeReconcileStore{fakeLifecycleStore: &fakeLifecycleStore{sup: sup}, observer: observer}
	r := ReconcileRequest{Target: lifecycleTarget(sup)}
	for _, c := range sup.g.Children {
		for _, a := range c.Attempts {
			if a.Status == executiongraph.AttemptRunning {
				r.ChildID, r.AttemptID = c.ExecutionID, a.AttemptID
			}
		}
	}
	s := ReconcileService{Store: store, Observer: observer, AllocateID: func() (string, error) { return "00000000-0000-4000-8000-000000000555", nil }, Now: func() time.Time { return time.Unix(40, 0).UTC() }}
	return s, r, store, observer
}

func TestReconcileStaleAndUnknownAppendsFactsOnly(t *testing.T) {
	for _, kind := range []string{"stale", "liveness-unknown"} {
		t.Run(kind, func(t *testing.T) {
			s, r, store, observer := reconcileFixture(t)
			observer.unknown = kind == "liveness-unknown"
			before := cloneSupervisorGraph(store.sup.g)
			p, e := s.Preview(context.Background(), r)
			if e != nil {
				t.Fatal(e)
			}
			want := map[string]string{"stale": "retry_safe", "liveness-unknown": "recovery_required"}[kind]
			if p.Classification != want || len(p.Diagnostics) != 1 || !strings.HasPrefix(p.Diagnostics[0], "quarantined-inbox:") {
				t.Fatal(p)
			}
			result, e := s.Apply(context.Background(), ReconcileApply{Preview: p, AuthorizeLocal: true, Actor: "operator"})
			if e != nil || result.Status != "success" || result.Fact == nil || result.Fact.ObservationsDigest != p.ObservationsDigest || result.Fact.Classification != want {
				t.Fatal(result, e)
			}
			if executiongraph.ValidateSuccessorUpdate(before, store.sup.g) != nil {
				t.Fatal("fact changed frozen state")
			}
			for i, c := range store.sup.g.Children {
				for j, a := range c.Attempts {
					if a.Status != before.Children[i].Attempts[j].Status {
						t.Fatal("reconcile changed status")
					}
				}
			}
			// A retry_safe fact enables only an explicit retry, never success.
			eligible, e := executiongraph.SuccessorRetryEligible(store.sup.g, r.ChildID, false)
			if e != nil || eligible {
				t.Fatal("fact authorized implicit retry")
			}
		})
	}
	t.Run("unknown-status", func(t *testing.T) {
		s, _, store, _ := reconcileFixture(t)
		raw, e := os.ReadFile("../executiongraph/testdata/successor/valid/process_outcome_unknown.json")
		if e != nil {
			t.Fatal(e)
		}
		unknown, e := executiongraph.DecodeSuccessorGraph(raw)
		if e != nil {
			t.Fatal(e)
		}
		wire, e := executiongraph.EncodeSuccessorGraph(unknown)
		if e != nil {
			t.Fatal(e)
		}
		unknown.StorageRevision = sha256.Sum256(wire)
		store.sup.g = unknown
		r := ReconcileRequest{Target: lifecycleTarget(store.sup), ChildID: unknown.Children[0].ExecutionID, AttemptID: unknown.Children[0].Attempts[0].AttemptID}
		p, e := s.Preview(context.Background(), r)
		if e != nil || p.Status != executiongraph.AttemptUnknown {
			t.Fatal(p, e)
		}
		if _, e = s.Apply(context.Background(), ReconcileApply{Preview: p, AuthorizeLocal: true, Actor: "operator"}); e != nil {
			t.Fatal(e)
		}
		if store.sup.g.Children[0].Attempts[0].Status != executiongraph.AttemptUnknown || len(store.sup.g.Children[0].Attempts[0].ReconciliationFacts) != 1 {
			t.Fatal("unknown upgraded")
		}
	})
}

func TestReconcileRefusals(t *testing.T) {
	for _, kind := range []string{"live-lease", "orphan-alive", "stale-preview", "observation-drift", "no-authority", "terminal-attempt", "foreign-attempt"} {
		t.Run(kind, func(t *testing.T) {
			s, r, store, observer := reconcileFixture(t)
			var want error
			switch kind {
			case "live-lease":
				observer.held, want = true, ErrReconcileLeaseHeld
			case "orphan-alive":
				observer.alive, want = true, ErrReconcileOrphanAlive
			case "terminal-attempt":
				for _, c := range store.sup.g.Children {
					for _, a := range c.Attempts {
						if a.Status == executiongraph.AttemptSucceeded {
							r.ChildID, r.AttemptID = c.ExecutionID, a.AttemptID
						}
					}
				}
				if _, e := s.Preview(context.Background(), r); !errors.Is(e, ErrReconcileIneligible) {
					t.Fatal("terminal attempt reconciled", e)
				}
				return
			case "foreign-attempt":
				r.AttemptID = "00000000-0000-4000-8000-000000000444"
				if _, e := s.Preview(context.Background(), r); !errors.Is(e, ErrAdmission) {
					t.Fatal(e)
				}
				return
			}
			p, e := s.Preview(context.Background(), r)
			if e != nil {
				t.Fatal(e)
			}
			if kind == "orphan-alive" && p.Next != "stop_orphan_then_reconcile" {
				t.Fatal("missing stop guidance", p.Next)
			}
			apply := ReconcileApply{Preview: p, AuthorizeLocal: true, Actor: "operator"}
			switch kind {
			case "stale-preview":
				store.sup.g.StorageRevision[0] ^= 1
				want = ErrStalePreview
			case "observation-drift":
				observer.worktree, want = strings.Repeat("9", 64), ErrStalePreview
			case "no-authority":
				apply.AuthorizeLocal, want = false, ErrDispatchAuthority
			}
			result, e := s.Apply(context.Background(), apply)
			if !errors.Is(e, want) || result.Fact != nil || store.appends != 0 {
				t.Fatal(kind, result, e)
			}
			for _, c := range store.sup.g.Children {
				for _, a := range c.Attempts {
					if len(a.ReconciliationFacts) != 0 {
						t.Fatal("fact recorded", kind)
					}
				}
			}
		})
	}
}
