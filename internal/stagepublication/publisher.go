// Package stagepublication composes stage execution with protected local stores.
package stagepublication

import (
	"context"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
)

// SinglePublicationBuilder is the sequential executor's ledger-generation seam.
// It must retain the bound stage and return a valid CAS candidate plus its exact
// Execution3 codec digest; it cannot create a graph or resolve another Profile.
type SinglePublicationBuilder interface {
	BuildSingle(context.Context, stageexecution.DispatchPreview, string, time.Time) (workflow.SingleDispatch, error)
}

type LocalPublisher struct {
	Graphs     local.GraphStore
	Executions local.WorkflowStore
	Leases     local.RunLeaseStore
	Single     SinglePublicationBuilder
	AllocateID func() (string, error)
}

func (p LocalPublisher) Build(ctx context.Context, preview stageexecution.DispatchPreview, actor string, now time.Time) (stageexecution.Publication, error) {
	if p.AllocateID == nil {
		return stageexecution.Publication{}, stageexecution.ErrPreview
	}
	a := executiongraph.DispatchAuthority{FormatVersion: 1, ProjectID: preview.Plan.ProjectID, PreviewDigest: preview.Digest, ActorKind: "human-operator", Actor: actor, RecordedAt: now}
	if preview.Publication == nil {
		if p.Single == nil {
			return stageexecution.Publication{}, stageexecution.ErrPreview
		}
		attempt, e := p.AllocateID()
		if e != nil {
			return stageexecution.Publication{}, e
		}
		single, e := p.Single.BuildSingle(ctx, preview, attempt, now)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		if single.State.ExecutionID != preview.Plan.Execution.ExecutionID || single.ResolvedProfile == nil || preview.Plan.Compilation.Agents[0].Input.ResolvedProfile == nil || *single.ResolvedProfile != *preview.Plan.Compilation.Agents[0].Input.ResolvedProfile || single.AttemptID != attempt || single.Workspace != preview.Children[0].Workspace || single.Inbox != preview.Children[0].Inbox || single.AxiomDigest != preview.AxiomDigest {
			return stageexecution.Publication{}, stageexecution.ErrPreview
		}
		digest, e := local.ExecutionDigest(single.State)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		a.Scope = "single-first-attempt"
		a.ParentID = single.State.ExecutionID
		a.Revision = single.State.Revision
		a.ProposalDigest = preview.Plan.Digest
		a.SingleStageID = string(single.State.Stage)
		a.SingleStageOrdinal = single.State.StageOrdinal
		a.SingleAttemptID = attempt
		a.LedgerDigest = digest
		a.Children = []executiongraph.DispatchChildAuthority{{ChildID: a.ParentID, Effects: preview.Children[0].Effects}}
		if executiongraph.ValidateDispatchAuthority(a) != nil {
			return stageexecution.Publication{}, stageexecution.ErrPreview
		}
		return stageexecution.Publication{Single: &single, Authority: a}, nil
	}
	request := *preview.Publication
	request.AuthorityReferences = map[string]string{}
	for key := range request.ChildAuthorities {
		request.AuthorityReferences[key] = "pending-dispatch-authority"
	}
	base, e := executiongraph.NewGraphService(nil, p.AllocateID, func() time.Time { return now }).Build(request)
	if e != nil {
		return stageexecution.Publication{}, e
	}
	a.Scope = "graph-first-attempts"
	a.ParentID = base.Parent.ExecutionID
	a.Revision = base.Parent.GraphRevision
	a.ProposalDigest = base.Parent.ProposalDigest
	for _, c := range base.Children {
		a.Children = append(a.Children, executiongraph.DispatchChildAuthority{ChildID: c.ExecutionID, Effects: c.Envelope.AllowedEffects})
	}
	ref, e := executiongraph.DispatchAuthorityDigest(a)
	if e != nil {
		return stageexecution.Publication{}, e
	}
	g := executiongraph.SuccessorGraph{FormatVersion: 3, Parent: base.Parent, ClassificationPolicy: executiongraph.ClassificationPolicyV1, DispatchAuthorityRef: ref, StageConcurrency: uint32(preview.Plan.Compilation.Concurrency)}
	for _, c := range base.Children {
		c.Envelope.AuthorityReference = ref
		g.Children = append(g.Children, executiongraph.SuccessorChild{ExecutionID: c.ExecutionID, ParentID: c.ParentID, GraphRevision: c.GraphRevision, NodeKey: c.NodeKey, Envelope: c.Envelope})
	}
	if executiongraph.ValidateGraphDispatchAuthority(g, a, preview.Digest) != nil {
		return stageexecution.Publication{}, stageexecution.ErrPreview
	}
	return stageexecution.Publication{Graph: &g, Authority: a}, nil
}
func (p LocalPublisher) Commit(ctx context.Context, v stageexecution.Publication) error {
	if v.Graph != nil && v.Single == nil {
		lease, e := p.Leases.Acquire(ctx, v.Authority.ProjectID, v.Authority.ParentID, 0)
		if e != nil {
			return e
		}
		defer lease.Close()
		return p.Graphs.CommitDispatch(ctx, *v.Graph, v.Authority, v.Authority.PreviewDigest, lease)
	}
	if v.Single != nil && v.Graph == nil {
		return p.Executions.CommitSingleDispatch(ctx, v.Single.State, v.Authority, v.Authority.PreviewDigest)
	}
	return stageexecution.ErrPreview
}
func (p LocalPublisher) ReadBack(ctx context.Context, v stageexecution.Publication) (stageexecution.Publication, error) {
	if v.Graph != nil && v.Single == nil {
		g, e := p.Graphs.LoadSuccessor(ctx, v.Authority.ProjectID, v.Authority.ParentID)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		a, e := p.Graphs.LoadDispatchAuthority(ctx, v.Authority.ProjectID, v.Authority.ParentID, v.Authority.PreviewDigest)
		return stageexecution.Publication{Graph: &g, Authority: a}, e
	}
	if v.Single != nil && v.Graph == nil {
		single := *v.Single
		state, e := p.Executions.Load(ctx, single.State.ProjectID, single.State.RepositoryKey, single.State.WorkItem)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		single.State = state
		ref, e := executiongraph.DispatchAuthorityDigest(v.Authority)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		a, e := p.Executions.LoadSingleDispatchAuthority(ctx, state, ref, v.Authority.PreviewDigest, single.AttemptID)
		return stageexecution.Publication{Single: &single, Authority: a}, e
	}
	return stageexecution.Publication{}, stageexecution.ErrPreview
}
