package local

import (
	"context"
	"crypto/sha256"

	"github.com/rgomids/axiom/internal/executiongraph"
)

// RequestCancel is the sole write allowed without owning the run lease: an
// append-only control message for its current owner. Graph CAS still serializes
// it with attempt saves. No attempt, epoch or dispatch authority can change.
func (s GraphStore) RequestCancel(ctx context.Context, g executiongraph.SuccessorGraph, request executiongraph.CancelRequest) (executiongraph.SuccessorGraph, error) {
	if err := ctx.Err(); err != nil {
		return g, err
	}
	if executiongraph.ValidateSuccessorGraph(g) != nil || g.StorageRevision == ([32]byte{}) || request.AuthorityRef == g.DispatchAuthorityRef {
		return g, executiongraph.ErrInvalidGraph
	}
	next := g
	if g.CancelRequest != nil {
		if *g.CancelRequest != request {
			return g, ErrConflict
		}
	} else {
		next.CancelRequest = &request
	}
	if executiongraph.ValidateSuccessorUpdate(g, next) != nil {
		return g, executiongraph.ErrInvalidGraph
	}
	wire, err := executiongraph.EncodeSuccessorGraph(next)
	if err != nil {
		return g, err
	}
	root, graphs, version, project, err := s.openProject(g.Children[0].Envelope.Scope.ProjectID, false)
	if err != nil {
		return g, err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, graphs, version, project)
	if err != nil {
		return g, err
	}
	defer closeFiles(locks)
	expected, err := readSuccessorPublishedFile(project, graphName(g.Parent.ExecutionID))
	if err != nil {
		return g, err
	}
	if sha256.Sum256(expected) != g.StorageRevision {
		return g, ErrConflict
	}
	old, err := executiongraph.DecodeSuccessorGraph(expected)
	if err != nil || executiongraph.ValidateSuccessorUpdate(old, next) != nil {
		return g, executiongraph.ErrInvalidGraph
	}
	priorWire, err := executiongraph.EncodeSuccessorGraph(g)
	if err != nil || sha256.Sum256(priorWire) != g.StorageRevision {
		return g, ErrConflict
	}
	if old.CancelRequest != nil {
		return old, nil
	}
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxGraphBytes
	if err = publishFile(ctx, project, graphName(g.Parent.ExecutionID), expected, wire, false, hooks); err != nil {
		return g, err
	}
	next.StorageRevision = sha256.Sum256(wire)
	return next, nil
}
