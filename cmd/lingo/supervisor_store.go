package main

import (
	"context"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/stageexecution"
)

// LocalSupervisorStore reuses protected publication and the lease-before-CAS
// lock order. Records is supplied by stage composition, never reconstructed by
// rendering or inferred from child stdout.
type LocalSupervisorStore struct {
	LoadContract func(context.Context, stageexecution.SupervisorRequest) (stageexecution.Contract, error)
	Graphs       local.GraphStore
	Leases       local.RunLeaseStore
	ReadRecords  func(context.Context, executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error)
}

var _ stageexecution.SupervisorStore = LocalSupervisorStore{}
var _ stageexecution.SupervisorLease = localSupervisorLease{}

type localSupervisorLease struct{ lease *local.RunLease }

func (l localSupervisorLease) Epoch() uint64   { return l.lease.Epoch }
func (l localSupervisorLease) Validate() error { return l.lease.Validate() }
func (l localSupervisorLease) Close() error    { return l.lease.Close() }
func (s LocalSupervisorStore) Load(ctx context.Context, r stageexecution.SupervisorRequest) (executiongraph.SuccessorGraph, executiongraph.DispatchAuthority, error) {
	g, e := s.Graphs.LoadSuccessor(ctx, r.ProjectID, r.ParentID)
	if e != nil {
		return g, executiongraph.DispatchAuthority{}, e
	}
	a, e := s.Graphs.LoadDispatchAuthority(ctx, r.ProjectID, r.ParentID, r.PreviewDigest)
	return g, a, e
}
func (s LocalSupervisorStore) Acquire(ctx context.Context, g executiongraph.SuccessorGraph) (stageexecution.SupervisorLease, error) {
	var epoch uint64
	for _, c := range g.Children {
		for _, a := range c.Attempts {
			if a.LeaseEpoch > epoch {
				epoch = a.LeaseEpoch
			}
		}
	}
	l, e := s.Leases.Acquire(ctx, g.Children[0].Envelope.Scope.ProjectID, g.Parent.ExecutionID, epoch)
	if e != nil {
		return nil, e
	}
	return localSupervisorLease{l}, nil
}
func (s LocalSupervisorStore) Save(ctx context.Context, g executiongraph.SuccessorGraph, l stageexecution.SupervisorLease) (executiongraph.SuccessorGraph, error) {
	lease, ok := l.(localSupervisorLease)
	if !ok {
		return g, stageexecution.ErrSupervisorRecovery
	}
	return s.Graphs.SaveSuccessor(ctx, g, lease.lease)
}
func (s LocalSupervisorStore) Records(ctx context.Context, g executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error) {
	if s.ReadRecords == nil {
		return nil, stageexecution.ErrSupervisorRecovery
	}
	return s.ReadRecords(ctx, g)
}

func (s LocalSupervisorStore) Contract(ctx context.Context, r stageexecution.SupervisorRequest) (stageexecution.Contract, error) {
	if s.LoadContract == nil {
		return stageexecution.Contract{}, stageexecution.ErrAdmission
	}
	return s.LoadContract(ctx, r)
}
