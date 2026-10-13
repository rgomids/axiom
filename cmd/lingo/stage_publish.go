package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
)

func attemptInbox(workspace, attempt string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(workspace)), "inboxes", filepath.Base(workspace), attempt)
}

// Correlation is a selector, never authority. The protected graph supplies the
// running attempt, epoch and locations; publish has no flags for those fields.
type persistedChildResolver struct {
	graphs                          local.GraphStore
	leases                          local.RunLeaseStore
	project, parent, child, attempt string
}

func (r persistedChildResolver) ResolveChild(ctx context.Context, cwd string) (coordination.ChildContext, error) {
	held, err := r.leases.Probe(ctx, r.project, r.parent)
	if err != nil || !held {
		return coordination.ChildContext{}, coordination.ErrChildContext
	}
	g, err := r.graphs.LoadSuccessor(ctx, r.project, r.parent)
	if err != nil {
		return coordination.ChildContext{}, coordination.ErrChildContext
	}
	for _, c := range g.Children {
		if c.ExecutionID != r.child || len(c.Attempts) == 0 {
			continue
		}
		a := c.Attempts[len(c.Attempts)-1]
		if a.AttemptID != r.attempt || a.Status != executiongraph.AttemptRunning {
			break
		}
		binding := coordination.ChildContext{Workspace: c.Envelope.Workspace, Inbox: attemptInbox(c.Envelope.Workspace, a.AttemptID), ParentID: r.parent, ChildID: r.child, AttemptID: r.attempt, GraphRevision: g.Parent.GraphRevision, Epoch: a.LeaseEpoch, Running: true}
		// Read-only adapters use the inbox as their CWD. Both paths are exact
		// persisted locations; unrelated descendants are deliberately refused.
		if cwd != binding.Workspace && cwd != binding.Inbox {
			break
		}
		return binding, nil
	}
	return coordination.ChildContext{}, coordination.ErrChildContext
}

type boundChildResolver struct{ context coordination.ChildContext }

func (r boundChildResolver) ResolveChild(ctx context.Context, workspace string) (coordination.ChildContext, error) {
	if ctx.Err() != nil {
		return coordination.ChildContext{}, ctx.Err()
	}
	if r.context.Workspace != workspace {
		return coordination.ChildContext{}, coordination.ErrChildContext
	}
	return r.context, nil
}

func (s lifecycleService) stagePublish(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	graphs, err := local.NewGraphStore(s.stateRoot)
	if err != nil {
		return s.stageError(err, nil)
	}
	leases, err := local.NewRunLeaseStore(s.stateRoot)
	if err != nil {
		return s.stageError(err, nil)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return s.stageError(coordination.ErrChildContext, nil)
	}
	resolver := persistedChildResolver{graphs: graphs, leases: leases, project: os.Getenv("AXIOM_CORRELATION_ID"), parent: os.Getenv("AXIOM_EXECUTION_ID"), child: os.Getenv("AXIOM_GRAPH_ID"), attempt: os.Getenv("AXIOM_ATTEMPT_ID")}
	c, err := resolver.ResolveChild(ctx, cwd)
	if err != nil {
		return s.stageError(err, nil)
	}
	inbox, err := local.NewAttemptInbox(c)
	if err != nil {
		return s.stageError(err, nil)
	}
	fields := make([]coordination.Field, 0, len(in.Fields))
	for _, field := range in.Fields {
		name, value, ok := strings.Cut(field, "=")
		if !ok || name == "" {
			return s.stageError(coordination.ErrInvalidRecord, nil)
		}
		fields = append(fields, coordination.Field{Name: name, Value: value})
	}
	// Append checks the inbox's original binding, including epoch; no canonical
	// record is published until the supervisor's fenced ingestion accepts it.
	publisher := coordination.InboxPublisher{Resolver: boundChildResolver{c}, Inbox: inbox, AllocateID: allocateStageID}
	candidate, err := publisher.Publish(ctx, c.Workspace, coordination.Kind(in.Kind), fields, nil, coordination.Provenance{Product: "axiom", Version: "development", Revision: "unavailable", SourceState: "child-inbox"})
	if err != nil {
		return s.stageError(err, nil)
	}
	return s.stageSuccess("coordination_candidate_published", "Coordination candidate appended to the attempt inbox", []string{"coordination-candidate:" + candidate.Record.RecordID}, "The supervisor validates and ingests this candidate into the canonical stream", candidate)
}
