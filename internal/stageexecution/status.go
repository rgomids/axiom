package stageexecution

import (
	"context"

	"github.com/rgomids/axiom/internal/executiongraph"
)

type StatusRequest struct{ ProjectID, ParentID string }

// StatusSource reads state independently of dispatch receipts and mutable
// Runtime configuration. Inspection must remain possible when dispatch is denied.
type StatusSource interface {
	StageSnapshot(context.Context, StatusRequest) (Contract, error)
}

// StatusObservation is optional, read-only and non-blocking. Its diagnostics
// never alter the persisted outcome, qualify records or grant authority.
type StatusObservation interface {
	LeaseHeld(context.Context, StatusRequest) (bool, error)
	Process(context.Context, executiongraph.ProcessCorrelation) (*executiongraph.ProcessCorrelation, error)
}

type StatusResult struct {
	Graph    executiongraph.GraphDocument
	Blockers map[string]string
	Health   map[string]executiongraph.AttemptHealth
	Next     map[string]string
}

type StatusService struct {
	Source   StatusSource
	Observer StatusObservation
}

func (s StatusService) Read(ctx context.Context, r StatusRequest) (StatusResult, error) {
	result := StatusResult{Blockers: map[string]string{}, Health: map[string]executiongraph.AttemptHealth{}, Next: map[string]string{}}
	if s.Source == nil {
		return result, ErrPreview
	}
	c, err := s.Source.StageSnapshot(ctx, r)
	if err != nil {
		return result, err
	}
	admission, err := Admit(StageStatus, c)
	if err != nil {
		return result, err
	}
	if admission.Next != "" {
		result.Next[r.ParentID] = admission.Next
	}
	if c.Graph == nil {
		return result, nil
	}
	wire, err := executiongraph.EncodeGraphDocument(*c.Graph)
	if err != nil {
		return result, err
	}
	result.Graph, err = executiongraph.DecodeGraphDocument(wire)
	if err != nil {
		return result, err
	}
	if c.Graph.Successor == nil {
		return result, nil
	}
	g := result.Graph.Successor
	g.StorageRevision = c.Graph.Successor.StorageRevision
	if g.Parent.ExecutionID != r.ParentID || g.Children[0].Envelope.Scope.ProjectID != r.ProjectID {
		return StatusResult{}, ErrAdmission
	}
	_, result.Blockers, err = executiongraph.ReadySuccessorChildren(*g, nil)
	if err != nil {
		return result, err
	}
	var held bool
	var leaseErr error
	if s.Observer != nil {
		held, leaseErr = s.Observer.LeaseHeld(ctx, r)
	}
	for _, child := range g.Children {
		if reason := result.Blockers[child.ExecutionID]; reason != "" {
			result.Next[child.ExecutionID] = reason
		}
		for _, at := range child.Attempts {
			if at.Status != executiongraph.AttemptRunning {
				continue
			}
			if s.Observer == nil || leaseErr != nil {
				result.Health[at.AttemptID] = executiongraph.AttemptHealth{LivenessUnknown: true}
				continue
			}
			var observed *executiongraph.ProcessCorrelation
			var observationErr error
			if !held && at.Process != nil {
				observed, observationErr = s.Observer.Process(ctx, *at.Process)
			}
			h := executiongraph.ObserveAttemptHealth(at, held, observed, observationErr)
			result.Health[at.AttemptID] = h
			if h.Stale {
				result.Next[child.ExecutionID] = "reconcile_required"
			}
			if h.Orphan {
				result.Next[child.ExecutionID] = "stop_orphan_then_reconcile"
			}
		}
	}
	return result, nil
}
