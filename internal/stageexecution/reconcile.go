package stageexecution

import (
	"context"
	"errors"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

var (
	ErrReconcileLeaseHeld   = errors.New("reconcile_lease_held")
	ErrReconcileOrphanAlive = errors.New("stop_orphan_then_reconcile")
	ErrReconcileIneligible  = errors.New("reconcile_not_required")
)

// ReconcileObserver gathers bounded, digest-only observations. The same
// implementation is rechecked by the store under the run lease at apply; it
// never classifies records as success or grants retry authority.
type ReconcileObserver interface {
	ObserveReconciliation(context.Context, executiongraph.SuccessorGraph, string, string) (executiongraph.ReconciliationObservations, error)
}

// ReconcileStore appends one fact with graph CAS after acquiring the run
// lease and re-observing. It cannot change attempt status or other fields.
type ReconcileStore interface {
	StatusSource
	AppendReconciliation(context.Context, executiongraph.SuccessorGraph, executiongraph.ReconciliationFact) (executiongraph.SuccessorGraph, error)
}

type ReconcileRequest struct {
	Target             StatusRequest
	ChildID, AttemptID string
}
type ReconcilePreview struct {
	Request         ReconcileRequest
	Revision        uint64
	StorageRevision [32]byte
	Status          executiongraph.AttemptStatus
	Observations    executiongraph.ReconciliationObservations
	// ObservationsDigest binds the fact; Classification is derived, never chosen.
	ObservationsDigest string
	Classification     string
	// Diagnostics are display-only: quarantined inbox content never qualifies
	// a result, and stop guidance is not an authority to signal processes.
	Diagnostics []string
	Next        string
	Digest      string
}
type ReconcileApply struct {
	Preview        ReconcilePreview
	AuthorizeLocal bool
	Actor          string
}
type ReconcileResult struct {
	Status string
	Fact   *executiongraph.ReconciliationFact
	Next   string
}
type ReconcileService struct {
	Store      ReconcileStore
	Observer   ReconcileObserver
	AllocateID func() (string, error)
	Now        func() time.Time
}

func (s ReconcileService) snapshot(ctx context.Context, r ReconcileRequest) (executiongraph.SuccessorGraph, executiongraph.SuccessorAttempt, error) {
	if s.Store == nil || s.Observer == nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorAttempt{}, ErrPreview
	}
	c, err := s.Store.StageSnapshot(ctx, r.Target)
	if err != nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorAttempt{}, err
	}
	if _, err = Admit(StageReconcile, c); err != nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorAttempt{}, err
	}
	if c.Graph == nil || c.Graph.Successor == nil {
		return executiongraph.SuccessorGraph{}, executiongraph.SuccessorAttempt{}, executiongraph.ErrUnsupportedGraphFormat
	}
	g := *c.Graph.Successor
	if g.Parent.ExecutionID != r.Target.ParentID || g.Children[0].Envelope.Scope.ProjectID != r.Target.ProjectID {
		return g, executiongraph.SuccessorAttempt{}, ErrAdmission
	}
	for _, child := range g.Children {
		if child.ExecutionID != r.ChildID {
			continue
		}
		for i, a := range child.Attempts {
			if a.AttemptID != r.AttemptID {
				continue
			}
			// Only the latest stale or unknown attempt needs reconciliation;
			// terminal classified outcomes are never reinterpreted.
			if i != len(child.Attempts)-1 || a.Status != executiongraph.AttemptRunning && a.Status != executiongraph.AttemptUnknown && !a.AmbiguousEffect {
				return g, a, ErrReconcileIneligible
			}
			return g, a, nil
		}
	}
	return g, executiongraph.SuccessorAttempt{}, ErrAdmission
}

func (s ReconcileService) preview(ctx context.Context, r ReconcileRequest) (ReconcilePreview, executiongraph.SuccessorGraph, error) {
	g, a, err := s.snapshot(ctx, r)
	if err != nil {
		return ReconcilePreview{}, g, err
	}
	o, err := s.Observer.ObserveReconciliation(ctx, g, r.ChildID, r.AttemptID)
	if err != nil {
		return ReconcilePreview{}, g, err
	}
	if o.ParentID != g.Parent.ExecutionID || o.ChildID != r.ChildID || o.AttemptID != r.AttemptID || o.GraphRevision != g.Parent.GraphRevision || o.LeaseEpoch != a.LeaseEpoch {
		return ReconcilePreview{}, g, ErrStalePreview
	}
	digest, err := o.Digest()
	if err != nil {
		return ReconcilePreview{}, g, err
	}
	p := ReconcilePreview{Request: r, Revision: g.Parent.GraphRevision, StorageRevision: g.StorageRevision, Status: a.Status, Observations: o, ObservationsDigest: digest, Classification: "recovery_required", Diagnostics: []string{"quarantined-inbox:" + o.QuarantinedInboxDigest}}
	switch {
	case o.LeaseHeld:
		p.Next = ErrReconcileLeaseHeld.Error()
	case o.ProcessAlive:
		p.Next = ErrReconcileOrphanAlive.Error()
	case o.ProcessUnknown:
		p.Next = "process_liveness_unknown"
	default:
		p.Classification = "retry_safe"
		p.Next = "explicit_retry_required"
	}
	p.Digest, err = canonicalDigest(struct {
		Operation Operation
		Preview   ReconcilePreview
		Graph     executiongraph.SuccessorGraph
	}{StageReconcile, p, g})
	return p, g, err
}

func (s ReconcileService) Preview(ctx context.Context, r ReconcileRequest) (ReconcilePreview, error) {
	p, _, err := s.preview(ctx, r)
	return p, err
}

// Apply records one append-only fact. It never upgrades status, never makes a
// result qualifying and never authorizes a retry, integration or stop.
func (s ReconcileService) Apply(ctx context.Context, r ReconcileApply) (ReconcileResult, error) {
	result := ReconcileResult{Status: "denied_authority"}
	if !r.AuthorizeLocal || r.Actor == "" {
		return result, ErrDispatchAuthority
	}
	fresh, g, err := s.preview(ctx, r.Preview.Request)
	if err != nil {
		return result, err
	}
	if fresh.Digest != r.Preview.Digest {
		return result, ErrStalePreview
	}
	if fresh.Observations.LeaseHeld {
		result.Next = fresh.Next
		return result, ErrReconcileLeaseHeld
	}
	if fresh.Observations.ProcessAlive {
		result.Next = fresh.Next
		return result, ErrReconcileOrphanAlive
	}
	if s.AllocateID == nil {
		return result, ErrPreview
	}
	id, err := s.AllocateID()
	if err != nil {
		return result, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	f := executiongraph.ReconciliationFact{FactID: id, ParentID: g.Parent.ExecutionID, ChildID: fresh.Request.ChildID, GraphRevision: g.Parent.GraphRevision, AttemptID: fresh.Request.AttemptID, ObservationsDigest: fresh.ObservationsDigest, Classification: fresh.Classification, ActorKind: "human-operator", Actor: r.Actor, RecordedAt: now().UTC()}
	f.AuthorityRef, err = canonicalDigest(struct {
		Operation Operation
		Preview   string
		Fact      executiongraph.ReconciliationFact
	}{StageReconcile, fresh.Digest, f})
	if err != nil {
		return result, err
	}
	// Validate locally before any write; the store repeats it under the lease.
	if _, err = executiongraph.AppendReconciliationFact(g, f, fresh.Observations); err != nil {
		return result, err
	}
	result.Status = "failure"
	saved, err := s.Store.AppendReconciliation(ctx, g, f)
	if err != nil {
		return result, err
	}
	if !factPersisted(saved, f) {
		result.Status = "recovery_required"
		return result, ErrSupervisorRecovery
	}
	result.Status, result.Fact, result.Next = "success", &f, fresh.Next
	return result, nil
}

func factPersisted(g executiongraph.SuccessorGraph, f executiongraph.ReconciliationFact) bool {
	for _, c := range g.Children {
		for _, a := range c.Attempts {
			for _, got := range a.ReconciliationFacts {
				same := got.RecordedAt.Equal(f.RecordedAt)
				got.RecordedAt = f.RecordedAt
				if same && got == f {
					return true
				}
			}
		}
	}
	return false
}
