package stageexecution

import (
	"context"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// CancelStore publishes only the control message with graph CAS, even while
// the supervisor holds its lease. It cannot change attempts or signal processes.
type CancelStore interface {
	StatusSource
	RequestCancel(context.Context, executiongraph.SuccessorGraph, executiongraph.CancelRequest) (executiongraph.SuccessorGraph, error)
}
type CancelPreview struct {
	Target          StatusRequest
	Revision        uint64
	StorageRevision [32]byte
	Digest          string
}
type CancelApply struct {
	Preview        CancelPreview
	AuthorizeLocal bool
	Actor          string
}
type CancelResult struct {
	Status, Category string
	Request          *executiongraph.CancelRequest
}
type CancelService struct {
	Store      CancelStore
	AllocateID func() (string, error)
	Now        func() time.Time
}

func (s CancelService) snapshot(ctx context.Context, r StatusRequest) (executiongraph.SuccessorGraph, error) {
	if s.Store == nil {
		return executiongraph.SuccessorGraph{}, ErrPreview
	}
	c, err := s.Store.StageSnapshot(ctx, r)
	if err != nil {
		return executiongraph.SuccessorGraph{}, err
	}
	if _, err = Admit(StageCancel, c); err != nil {
		return executiongraph.SuccessorGraph{}, err
	}
	if c.Graph == nil || c.Graph.Successor == nil {
		return executiongraph.SuccessorGraph{}, executiongraph.ErrUnsupportedGraphFormat
	}
	g := *c.Graph.Successor
	if g.Parent.ExecutionID != r.ParentID || g.Children[0].Envelope.Scope.ProjectID != r.ProjectID {
		return g, ErrAdmission
	}
	return g, nil
}
func cancelPreview(r StatusRequest, g executiongraph.SuccessorGraph) (CancelPreview, error) {
	p := CancelPreview{Target: r, Revision: g.Parent.GraphRevision, StorageRevision: g.StorageRevision}
	// Bind persisted bytes too: an in-memory source cannot hide state changes
	// behind a caller-asserted storage revision.
	digest, err := canonicalDigest(struct {
		Operation Operation
		Preview   CancelPreview
		Graph     executiongraph.SuccessorGraph
	}{StageCancel, p, g})
	p.Digest = digest
	return p, err
}
func (s CancelService) Preview(ctx context.Context, r StatusRequest) (CancelPreview, error) {
	g, err := s.snapshot(ctx, r)
	if err != nil {
		return CancelPreview{}, err
	}
	return cancelPreview(r, g)
}
func (s CancelService) Apply(ctx context.Context, r CancelApply) (CancelResult, error) {
	result := CancelResult{Status: "denied_authority"}
	if !r.AuthorizeLocal || r.Actor == "" {
		return result, ErrDispatchAuthority
	}
	g, err := s.snapshot(ctx, r.Preview.Target)
	if err != nil {
		return result, err
	}
	fresh, err := cancelPreview(r.Preview.Target, g)
	if err != nil || fresh != r.Preview {
		return result, ErrStalePreview
	}
	if g.CancelRequest != nil {
		request := *g.CancelRequest
		return CancelResult{Status: "success", Category: "cancellation_requested", Request: &request}, nil
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
	request := executiongraph.CancelRequest{RequestID: id, ParentID: g.Parent.ExecutionID, GraphRevision: g.Parent.GraphRevision, ActorKind: "human-operator", Actor: r.Actor, RequestedAt: now().UTC()}
	request.AuthorityRef, err = canonicalDigest(struct {
		Operation Operation
		Preview   string
		Request   executiongraph.CancelRequest
	}{StageCancel, fresh.Digest, request})
	if err != nil {
		return result, err
	}
	candidate := cloneSupervisorGraph(g)
	candidate.CancelRequest = &request
	if executiongraph.ValidateSuccessorUpdate(g, candidate) != nil || request.AuthorityRef == g.DispatchAuthorityRef {
		return result, executiongraph.ErrInvalidGraph
	}
	result.Status = "failure"
	saved, err := s.Store.RequestCancel(ctx, g, request)
	if err != nil {
		return result, err
	}
	if saved.CancelRequest == nil || *saved.CancelRequest != request {
		result.Status = "recovery_required"
		return result, ErrSupervisorRecovery
	}
	// Only publication is observed here. An absent supervisor or surviving
	// descendant cannot be mistaken for a cancelled attempt.
	result.Status, result.Category, result.Request = "success", "cancellation_requested", &request
	return result, nil
}
