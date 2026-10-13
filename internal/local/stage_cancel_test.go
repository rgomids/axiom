package local

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
)

func TestRequestCancelPublishesOnlyControlMessage(t *testing.T) {
	ctx := context.Background()
	root := privateTestRoot(t)
	s, err := NewGraphStore(root)
	if err != nil {
		t.Fatal(err)
	}
	g := localSuccessor(t, "running")
	projectID := g.Children[0].Envelope.Scope.ProjectID
	if err = s.CreateSuccessor(ctx, g); err != nil {
		t.Fatal(err)
	}
	g, err = s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	request := executiongraph.CancelRequest{RequestID: "00000000-0000-4000-8000-000000000777", ParentID: g.Parent.ExecutionID, GraphRevision: g.Parent.GraphRevision, ActorKind: "human-operator", Actor: "operator", RequestedAt: g.Parent.CreatedAt.Add(time.Second), AuthorityRef: strings.Repeat("e", 64)}
	// A live run lease does not block the control message; graph CAS does.
	leases, err := NewRunLeaseStore(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := leases.Acquire(ctx, projectID, g.Parent.ExecutionID, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	next, err := s.RequestCancel(ctx, g, request)
	if err != nil || next.CancelRequest == nil || *next.CancelRequest != request {
		t.Fatal(next.CancelRequest, err)
	}
	loaded, err := s.LoadSuccessor(ctx, projectID, g.Parent.ExecutionID)
	if err != nil || loaded.CancelRequest == nil || loaded.StorageRevision != next.StorageRevision || executiongraph.ValidateSuccessorUpdate(g, loaded) != nil {
		t.Fatal("cancel changed more than the control message", err)
	}
	if loaded.Children[0].Attempts[0].Status != executiongraph.AttemptRunning {
		t.Fatal("cancel rewrote attempt status")
	}
	// Stale snapshot: CAS refuses, persisted request is unchanged.
	other := request
	other.RequestID = "00000000-0000-4000-8000-000000000778"
	if _, err = s.RequestCancel(ctx, g, other); !errors.Is(err, ErrConflict) {
		t.Fatal("stale cancel accepted", err)
	}
	// Repeating the same request on the current snapshot is idempotent.
	again, err := s.RequestCancel(ctx, loaded, request)
	if err != nil || again.StorageRevision != loaded.StorageRevision {
		t.Fatal("repeated cancel rewrote graph", err)
	}
	if _, err = s.RequestCancel(ctx, loaded, other); !errors.Is(err, ErrConflict) {
		t.Fatal("second distinct request replaced the first", err)
	}
	forged := request
	forged.AuthorityRef = loaded.DispatchAuthorityRef
	if _, err = s.RequestCancel(ctx, g, forged); !errors.Is(err, executiongraph.ErrInvalidGraph) {
		t.Fatal("dispatch authority reused as cancel authority", err)
	}
}
