package local

import (
	"context"
	"crypto/sha256"
	"os"
	"reflect"

	"github.com/rgomids/axiom/internal/executiongraph"
)

func (s GraphStore) CreateSuccessor(ctx context.Context, g executiongraph.SuccessorGraph) error {
	wire, err := executiongraph.EncodeSuccessorGraph(g)
	if err != nil {
		return err
	}
	root, graphs, version, project, err := s.openProject(g.Children[0].Envelope.Scope.ProjectID, true)
	if err != nil {
		return err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(true, root, graphs, version, project)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxGraphBytes
	return publishFile(ctx, project, graphName(g.Parent.ExecutionID), nil, wire, true, hooks)
}

func (s GraphStore) LoadSuccessor(ctx context.Context, projectID, parentID string) (executiongraph.SuccessorGraph, error) {
	if err := ctx.Err(); err != nil {
		return executiongraph.SuccessorGraph{}, err
	}
	if !validGraphAddress(projectID, parentID) {
		return executiongraph.SuccessorGraph{}, ErrUnsafe
	}
	root, graphs, version, project, err := s.openProject(projectID, false)
	if err != nil {
		return executiongraph.SuccessorGraph{}, err
	}
	defer root.Close()
	defer graphs.Close()
	defer version.Close()
	defer project.Close()
	locks, err := lockRoots(false, root, graphs, version, project)
	if err != nil {
		return executiongraph.SuccessorGraph{}, err
	}
	defer closeFiles(locks)
	wire, err := readSuccessorPublishedFile(project, graphName(parentID))
	if err != nil {
		return executiongraph.SuccessorGraph{}, err
	}
	g, err := executiongraph.DecodeSuccessorGraph(wire)
	if err != nil || g.Parent.ExecutionID != parentID || g.Children[0].Envelope.Scope.ProjectID != projectID {
		return executiongraph.SuccessorGraph{}, ErrUnsafe
	}
	return g, nil
}

// SaveSuccessor requires the run lease before taking any graph CAS lock. New
// attempts and terminal writes must use the held epoch; historical attempts
// and their process correlation remain immutable under the codec validator.
func (s GraphStore) SaveSuccessor(ctx context.Context, g executiongraph.SuccessorGraph, lease *RunLease) (executiongraph.SuccessorGraph, error) {
	return s.saveSuccessor(ctx, g, lease, false)
}

func (s GraphStore) saveSuccessor(ctx context.Context, g executiongraph.SuccessorGraph, lease *RunLease, reconciliation bool) (executiongraph.SuccessorGraph, error) {
	if err := ctx.Err(); err != nil {
		return g, err
	}
	if lease == nil {
		return g, ErrConflict
	}
	lease.mu.Lock()
	defer lease.mu.Unlock()
	if err := lease.validateLocked(); err != nil {
		return g, err
	}
	if lease.stateRoot != s.root || executiongraph.ValidateLeaseEpoch(g, lease.Epoch) != nil || lease.projectID != g.Children[0].Envelope.Scope.ProjectID || lease.parentID != g.Parent.ExecutionID {
		return g, ErrConflict
	}
	wire, err := executiongraph.EncodeSuccessorGraph(g)
	if err != nil || g.StorageRevision == ([32]byte{}) {
		return g, executiongraph.ErrInvalidGraph
	}
	root, graphs, version, project, err := s.openProject(lease.projectID, false)
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
	expected, err := readSuccessorPublishedFile(project, graphName(lease.parentID))
	if err != nil {
		return g, err
	}
	if sha256.Sum256(expected) != g.StorageRevision {
		return g, ErrConflict
	}
	old, err := executiongraph.DecodeSuccessorGraph(expected)
	if err != nil || executiongraph.ValidateSuccessorUpdate(old, g) != nil {
		return g, executiongraph.ErrInvalidGraph
	}
	for i, c := range g.Children {
		prior := old.Children[i]
		if reconciliation && len(c.Attempts) != len(prior.Attempts) {
			return g, ErrConflict
		}
		for j, a := range c.Attempts {
			if j >= len(prior.Attempts) {
				if a.LeaseEpoch != lease.Epoch || len(a.ReconciliationFacts) != 0 {
					return g, ErrConflict
				}
				continue
			}
			before, after := prior.Attempts[j], a
			before.ReconciliationFacts, after.ReconciliationFacts = nil, nil
			if !reflect.DeepEqual(before, after) && (reconciliation || a.LeaseEpoch != lease.Epoch) {
				return g, ErrConflict
			}
			if !reconciliation && !reflect.DeepEqual(prior.Attempts[j].ReconciliationFacts, a.ReconciliationFacts) {
				return g, ErrConflict
			}
		}
	}
	if err := lease.validateLocked(); err != nil {
		return g, err
	}
	hooks := s.hooks
	hooks.contentLimit = executiongraph.MaxGraphBytes
	beforeCommit := hooks.beforeCommit
	hooks.beforeCommit = func() error {
		if beforeCommit != nil {
			if err := beforeCommit(); err != nil {
				return err
			}
		}
		return lease.validateLocked()
	}
	if err := publishFile(ctx, project, graphName(lease.parentID), expected, wire, false, hooks); err != nil {
		return g, err
	}
	g.StorageRevision = sha256.Sum256(wire)
	return g, nil
}

// ReconciliationObserver rechecks bounded preview observations at apply. It is
// called only after acquiring the run lease, with no graph/coordination locks
// held. Implementations gather worktree, canonical and quarantined projections;
// they cannot classify records as success or grant reconciliation authority.
type ReconciliationObserver interface {
	ObserveReconciliation(context.Context, executiongraph.SuccessorGraph, string, string) (executiongraph.ReconciliationObservations, error)
}

func (s GraphStore) AppendReconciliation(ctx context.Context, g executiongraph.SuccessorGraph, fact executiongraph.ReconciliationFact, observer ReconciliationObserver) (executiongraph.SuccessorGraph, error) {
	if observer == nil || executiongraph.ValidateSuccessorGraph(g) != nil {
		return g, executiongraph.ErrInvalidGraph
	}
	leases, err := NewRunLeaseStore(s.root)
	if err != nil {
		return g, err
	}
	var epoch uint64
	for _, c := range g.Children {
		for _, a := range c.Attempts {
			if a.LeaseEpoch > epoch {
				epoch = a.LeaseEpoch
			}
		}
	}
	// Holding a live lease rejects apply. Once acquired, no supervisor can start
	// between the observation check and the fact CAS publication.
	lease, err := leases.Acquire(ctx, g.Children[0].Envelope.Scope.ProjectID, g.Parent.ExecutionID, epoch)
	if err != nil {
		return g, err
	}
	defer lease.Close()
	observations, err := observer.ObserveReconciliation(ctx, g, fact.ChildID, fact.AttemptID)
	if err != nil {
		return g, err
	}
	// Acquiring the exclusive lease proves there was no live supervisor.
	// A raw status probe now sees our own reconciliation lease as held; bind
	// this component to the acquisition evidence rather than that self-probe.
	observations.LeaseHeld = false
	next, err := executiongraph.AppendReconciliationFact(g, fact, observations)
	if err != nil {
		return g, err
	}
	return s.saveSuccessor(ctx, next, lease, true)
}

func readSuccessorPublishedFile(root *os.Root, name string) ([]byte, error) {
	pending, err := protocolStatePresent(root)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrRecoveryRequired
	}
	return readPrivateFileBounded(root, name, executiongraph.MaxGraphBytes)
}
