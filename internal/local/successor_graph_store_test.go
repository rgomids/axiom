package local

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

type reconciliationObserverFunc func(context.Context, executiongraph.SuccessorGraph, string, string) (executiongraph.ReconciliationObservations, error)

func (f reconciliationObserverFunc) ObserveReconciliation(ctx context.Context, g executiongraph.SuccessorGraph, child, attempt string) (executiongraph.ReconciliationObservations, error) {
	return f(ctx, g, child, attempt)
}

func localSuccessor(t *testing.T, name string) executiongraph.SuccessorGraph {
	t.Helper()
	wire, err := os.ReadFile("../executiongraph/testdata/successor/valid/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	g, err := executiongraph.DecodeSuccessorGraph(wire)
	if err != nil {
		t.Fatal(err)
	}
	g.Children[0].Envelope.Controls.MaximumAttempts = 3
	return g
}

func localReconciliation(g executiongraph.SuccessorGraph) (executiongraph.ReconciliationFact, executiongraph.ReconciliationObservations) {
	a := g.Children[0].Attempts[0]
	o := executiongraph.ReconciliationObservations{ParentID: g.Parent.ExecutionID, ChildID: g.Children[0].ExecutionID, GraphRevision: g.Parent.GraphRevision, AttemptID: a.AttemptID, LeaseEpoch: a.LeaseEpoch, WorktreeDigest: strings.Repeat("1", 64), CanonicalRecordsDigest: strings.Repeat("2", 64), QuarantinedInboxDigest: strings.Repeat("3", 64)}
	d, _ := o.Digest()
	f := executiongraph.ReconciliationFact{FactID: "00000000-0000-4000-8000-000000000030", ParentID: o.ParentID, ChildID: o.ChildID, GraphRevision: o.GraphRevision, AttemptID: o.AttemptID, ObservationsDigest: d, Classification: "retry_safe", AuthorityRef: strings.Repeat("d", 64), ActorKind: "human-operator", Actor: "operator", RecordedAt: time.Unix(13, 0).UTC()}
	return f, o
}

func TestSuccessorReconciliationStore(t *testing.T) {
	for _, name := range []string{"running", "process_outcome_unknown"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			root := privateTestRoot(t)
			s, err := NewGraphStore(root)
			if err != nil {
				t.Fatal(err)
			}
			g := localSuccessor(t, name)
			projectID := g.Children[0].Envelope.Scope.ProjectID
			if err := s.CreateSuccessor(ctx, g); err != nil {
				t.Fatal(err)
			}
			g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
			if err != nil {
				t.Fatal(err)
			}
			f, o := localReconciliation(g)
			observer := reconciliationObserverFunc(func(context.Context, executiongraph.SuccessorGraph, string, string) (executiongraph.ReconciliationObservations, error) {
				return o, nil
			})
			leases, err := NewRunLeaseStore(root)
			if err != nil {
				t.Fatal(err)
			}
			l, err := leases.Acquire(ctx, projectID, g.Parent.ExecutionID, 1)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.AppendReconciliation(ctx, g, f, observer); !errors.Is(err, ErrConflict) {
				t.Fatal("live lease admitted fact", err)
			}
			l.Close()
			next, err := s.AppendReconciliation(ctx, g, f, observer)
			if err != nil {
				t.Fatal(err)
			}
			if next.Children[0].Attempts[0].Status != g.Children[0].Attempts[0].Status {
				t.Fatal("status changed")
			}
			if _, err := s.AppendReconciliation(ctx, g, f, observer); !errors.Is(err, ErrConflict) {
				t.Fatal("stale CAS accepted", err)
			}
			restarted, err := NewGraphStore(root)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := restarted.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
			if err != nil || !reflect.DeepEqual(next, loaded) {
				t.Fatal("restart", err)
			}
			inv, err := InspectStateInventory(ctx, root)
			if err != nil {
				t.Fatal(err)
			}
			kinds := map[InventoryKind]bool{}
			for _, e := range inv.Entries {
				if !e.Kind.Supported() {
					t.Fatalf("%s: %s", e.Relative, e.Kind)
				}
				kinds[e.Kind] = true
			}
			if !kinds[InventoryGraph] || !kinds[InventoryRunLease] {
				t.Fatal(kinds)
			}
			f.FactID = "00000000-0000-4000-8000-000000000031"
			f.Classification = "recovery_required"
			contradictory, err := s.AppendReconciliation(ctx, next, f, observer)
			if err != nil {
				t.Fatal(err)
			}
			if _, conflict := executiongraph.ReconciliationDisposition(contradictory.Children[0].Attempts[0]); !conflict {
				t.Fatal("conflict hidden")
			}
		})
	}
}

func TestSuccessorStoreFencesLowerEpochAndStatusRewrite(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	g := localSuccessor(t, "running")
	if err = s.CreateSuccessor(ctx, g); err != nil {
		t.Fatal(err)
	}
	projectID := g.Children[0].Envelope.Scope.ProjectID
	g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	leases, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := leases.Acquire(ctx, projectID, g.Parent.ExecutionID, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	finish := time.Unix(12, 0).UTC()
	a := &g.Children[0].Attempts[0]
	a.Status = executiongraph.AttemptUnknown
	a.ClassificationReason = executiongraph.ProcessOutcomeUnknown
	a.FinishedAt = &finish
	if _, err = s.SaveSuccessor(ctx, g, l); !errors.Is(err, ErrConflict) {
		t.Fatal("lower epoch terminal write admitted", err)
	}
	f, o := localReconciliation(g)
	o.GraphRevision++
	f.GraphRevision = o.GraphRevision
	f.ObservationsDigest, _ = o.Digest()
	observer := reconciliationObserverFunc(func(context.Context, executiongraph.SuccessorGraph, string, string) (executiongraph.ReconciliationObservations, error) {
		return o, nil
	})
	l.Close()
	if _, err = s.AppendReconciliation(ctx, g, f, observer); err == nil {
		t.Fatal("foreign revision admitted")
	}
}

func TestSuccessorStoreCurrentEpochTerminalSave(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	g := localSuccessor(t, "running")
	projectID := g.Children[0].Envelope.Scope.ProjectID
	leases, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := leases.Acquire(ctx, projectID, g.Parent.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if g.Children[0].Attempts[0].LeaseEpoch != l.Epoch {
		t.Fatal("fixture epoch mismatch")
	}
	if err := s.CreateSuccessor(ctx, g); err != nil {
		t.Fatal(err)
	}
	g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	finished := time.Unix(12, 0).UTC()
	a := &g.Children[0].Attempts[0]
	process := *a.Process
	a.Status = executiongraph.AttemptUnknown
	a.ClassificationReason = executiongraph.ProcessOutcomeUnknown
	a.FinishedAt = &finished
	a.AmbiguousEffect = true
	saved, err := s.SaveSuccessor(ctx, g, l)
	if err != nil {
		t.Fatal(err)
	}
	readBack, err := s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil || !reflect.DeepEqual(saved, readBack) || *readBack.Children[0].Attempts[0].Process != process {
		t.Fatal("terminal read-back", err)
	}
	rewritten := saved
	rewritten.Children = append([]executiongraph.SuccessorChild(nil), saved.Children...)
	rewritten.Children[0].Attempts = append([]executiongraph.SuccessorAttempt(nil), saved.Children[0].Attempts...)
	changed := &rewritten.Children[0].Attempts[0]
	changed.Status = executiongraph.AttemptFailed
	changed.AmbiguousEffect = false
	changed.ClassificationReason = executiongraph.ResultMissing
	if _, err := s.SaveSuccessor(ctx, rewritten, l); err == nil {
		t.Fatal("terminal status rewrite admitted")
	}
}

func TestSuccessorReconciliationPublicationRecovery(t *testing.T) {
	for _, stage := range []FaultStage{FaultF3, FaultF5, FaultF6} {
		t.Run(string(stage), func(t *testing.T) {
			ctx := context.Background()
			root := privateTestRoot(t)
			s, err := NewGraphStore(root)
			if err != nil {
				t.Fatal(err)
			}
			g := localSuccessor(t, "process_outcome_unknown")
			projectID := g.Children[0].Envelope.Scope.ProjectID
			if err := s.CreateSuccessor(ctx, g); err != nil {
				t.Fatal(err)
			}
			g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
			if err != nil {
				t.Fatal(err)
			}
			f, o := localReconciliation(g)
			observer := reconciliationObserverFunc(func(context.Context, executiongraph.SuccessorGraph, string, string) (executiongraph.ReconciliationObservations, error) {
				return o, nil
			})
			s.hooks.fault = func(at FaultStage) error {
				if at == stage {
					return ErrSimulatedInterruption
				}
				return nil
			}
			if _, err := s.AppendReconciliation(ctx, g, f, observer); err == nil {
				t.Fatal("interruption ignored")
			}
			if _, err := s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatal("interrupted graph readable", err)
			}
			report, err := InspectRecovery(ctx, RecoveryRoots{State: root})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, p := range report.Plans {
				if p.Directory == "graphs/v1/"+projectID {
					found = true
					want := RestorePrior
					if stage == FaultF6 {
						want = FinalizeCommitted
					}
					if p.Action != want {
						t.Fatal(p.Action, want)
					}
				}
			}
			if !found {
				t.Fatalf("graph protocol absent: %+v", report)
			}
		})
	}
}

func TestSuccessorStoreRechecksLeaseAtCommit(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	g := localSuccessor(t, "running")
	projectID := g.Children[0].Envelope.Scope.ProjectID
	leases, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := leases.Acquire(ctx, projectID, g.Parent.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := s.CreateSuccessor(ctx, g); err != nil {
		t.Fatal(err)
	}
	g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	s.hooks.beforeCommit = func() error {
		return lease.root.WriteFile("epoch.json", []byte(`{"formatVersion":1,"projectId":"`+projectID+`","parentId":"`+g.Parent.ExecutionID+`","epoch":2}`), 0o600)
	}
	finished := time.Unix(12, 0).UTC()
	g.Children[0].Attempts[0].FinishedAt = &finished
	g.Children[0].Attempts[0].Status = executiongraph.AttemptUnknown
	g.Children[0].Attempts[0].ClassificationReason = executiongraph.ProcessOutcomeUnknown
	if _, err := s.SaveSuccessor(ctx, g, lease); !errors.Is(err, ErrConflict) {
		t.Fatal("lease drift committed", err)
	}
	loaded, err := s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil || loaded.Children[0].Attempts[0].Status != executiongraph.AttemptRunning {
		t.Fatal("prior graph lost", err)
	}
}

func TestSuccessorStoreRejectsForeignStateRootLease(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	g := localSuccessor(t, "running")
	projectID := g.Children[0].Envelope.Scope.ProjectID
	if err := s.CreateSuccessor(ctx, g); err != nil {
		t.Fatal(err)
	}
	g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := NewRunLeaseStore(privateTestRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	lease, err := foreign.Acquire(ctx, projectID, g.Parent.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err := s.SaveSuccessor(ctx, g, lease); !errors.Is(err, ErrConflict) {
		t.Fatal("foreign root lease admitted", err)
	}
}

func TestSuccessorSavePinsLeaseThroughCommit(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	g := localSuccessor(t, "running")
	projectID := g.Children[0].Envelope.Scope.ProjectID
	leases, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := leases.Acquire(ctx, projectID, g.Parent.ExecutionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := s.CreateSuccessor(ctx, g); err != nil {
		t.Fatal(err)
	}
	g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	closeStarted := make(chan struct{})
	closed := make(chan error, 1)
	s.hooks.afterStage = func() {
		go func() { close(closeStarted); closed <- lease.Close() }()
		<-closeStarted
		select {
		case err := <-closed:
			t.Errorf("lease closed during save: %v", err)
		default:
		}
	}
	finished := time.Unix(12, 0).UTC()
	a := &g.Children[0].Attempts[0]
	a.Status = executiongraph.AttemptUnknown
	a.ClassificationReason = executiongraph.ProcessOutcomeUnknown
	a.FinishedAt = &finished
	if _, err := s.SaveSuccessor(ctx, g, lease); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("lease close did not finish after commit")
	}
}
