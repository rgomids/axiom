// Package stagepublication composes stage execution with protected local stores.
package stagepublication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
)

type LocalPublisher struct {
	Graphs     local.GraphStore
	Executions local.WorkflowStore
	Singles    local.SingleStageStore
	Leases     local.RunLeaseStore
	// Execution is the resolved target. Its protected state is reloaded at
	// Build, Commit and ReadBack; this value only addresses it.
	Execution  workflow.State
	AllocateID func() (string, error)
}

func (p LocalPublisher) Build(ctx context.Context, preview stageexecution.DispatchPreview, actor string, now time.Time) (stageexecution.Publication, error) {
	if p.AllocateID == nil {
		return stageexecution.Publication{}, stageexecution.ErrPreview
	}
	a := executiongraph.DispatchAuthority{FormatVersion: 1, ProjectID: preview.Plan.ProjectID, PreviewDigest: preview.Digest, ActorKind: "human-operator", Actor: actor, RecordedAt: now}
	if preview.Publication == nil {
		state, e := p.Executions.Load(ctx, p.Execution.ProjectID, p.Execution.RepositoryKey, p.Execution.WorkItem)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		single, e := BuildSingle(preview, state, actor, now)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		return stageexecution.Publication{Single: &single}, nil
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
	g := executiongraph.SuccessorGraph{FormatVersion: executiongraph.OperationalGraphVersion, Parent: base.Parent, ClassificationPolicy: executiongraph.ClassificationPolicyV1, DispatchAuthorityRef: ref, StageConcurrency: uint32(preview.Plan.Compilation.Concurrency)}
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
		run := v.Single.Run
		lease, e := p.Leases.Acquire(ctx, run.Scope.ProjectID, run.Scope.ExecutionID, run.RunEpoch)
		if e != nil {
			return e
		}
		defer lease.Close()
		return p.Singles.CommitAuthority(ctx, run, v.Single.Authority, v.Single.Execution, lease)
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
		run, e := p.Singles.Load(ctx, v.Single.Run.Scope.ProjectID, v.Single.Run.ScopeDigest)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		a, e := p.Singles.LoadAuthority(ctx, run, run.Authorities[len(run.Authorities)-1])
		if e != nil {
			return stageexecution.Publication{}, e
		}
		state, e := p.Executions.Load(ctx, v.Single.Execution.ProjectID, v.Single.Execution.RepositoryKey, v.Single.Execution.WorkItem)
		if e != nil {
			return stageexecution.Publication{}, e
		}
		return stageexecution.Publication{Single: &stageexecution.SinglePublication{Run: run, Authority: a, Execution: state}}, nil
	}
	return stageexecution.Publication{}, stageexecution.ErrPreview
}

// SingleScope is the canonical single-stage identity of an Execution's current
// stage; the Work Item is its exact selector, never a display form.
func SingleScope(state workflow.State) executiongraph.SingleStageScope {
	item := state.WorkItem
	return executiongraph.SingleStageScope{ProjectID: state.ProjectID, RepositoryKey: state.RepositoryKey, WorkItem: item.Provider + ":" + item.Resource + "#" + item.ExternalID, ExecutionID: state.ExecutionID, StageID: string(state.Stage), StageOrdinal: state.StageOrdinal}
}

// BuildSingle derives the first-attempt ledger and receipt from the reviewed
// single preview and the protected Execution 5 generation. It is pure.
func BuildSingle(preview stageexecution.DispatchPreview, state workflow.State, actor string, now time.Time) (stageexecution.SinglePublication, error) {
	plan := preview.Plan
	if preview.Publication != nil || plan.ExecutionKind != "single" || len(plan.Compilation.Agents) != 1 || len(preview.Children) != 1 || state.StorageRevision == ([32]byte{}) {
		return stageexecution.SinglePublication{}, stageexecution.ErrPreview
	}
	input := plan.Compilation.Agents[0].Input
	child := preview.Children[0]
	if plan.Execution.ExecutionID != state.ExecutionID || plan.Execution.Revision != state.Revision || plan.StageID != string(state.Stage) || plan.ProjectID != state.ProjectID || input.RepositoryKey != state.RepositoryKey || input.FrozenProfile == nil || state.FormatVersion != workflow.OperationalConfiguredFormatVersion {
		return stageexecution.SinglePublication{}, stageexecution.ErrStalePreview
	}
	wire, e := json.Marshal(input)
	if e != nil {
		return stageexecution.SinglePublication{}, e
	}
	sum := sha256.Sum256(wire)
	scope := SingleScope(state)
	digest, e := executiongraph.SingleStageScopeDigest(scope)
	if e != nil {
		return stageexecution.SinglePublication{}, e
	}
	frozen := input.FrozenProfile
	effects := append([]executiongraph.Effect{}, child.Effects...)
	generation := hex.EncodeToString(state.StorageRevision[:])
	binding := executiongraph.SingleStageBinding{ExecutionGeneration: generation, WorkflowRevision: state.Revision, PreviewDigest: preview.Digest, ProposalDigest: plan.Digest, InputDigest: hex.EncodeToString(sum[:]), AxiomDigest: preview.AxiomDigest, Workspace: child.Workspace, Inbox: child.Inbox, Effects: effects, Controls: child.Controls, Resolution: executiongraph.Resolution{FrozenProfile: frozen, RuntimeID: frozen.RuntimeID, ModelProfileID: frozen.ProfileID, ConfigurationRevision: frozen.ConfigurationRevision, ObservationRevision: frozen.ObservationRevision}}
	a := executiongraph.SingleStageAuthority{FormatVersion: executiongraph.SingleStageAuthorityVersion, Scope: executiongraph.SingleStageAuthorityScope, ProjectID: scope.ProjectID, ExecutionID: scope.ExecutionID, StageID: scope.StageID, StageOrdinal: scope.StageOrdinal, ScopeDigest: digest, ExecutionGeneration: generation, WorkflowRevision: state.Revision, PreviewDigest: preview.Digest, DispatchDigest: preview.Digest, ProposalDigest: plan.Digest, AttemptNumber: 1, Effects: effects, Disposition: executiongraph.DispositionNotRequired, ActorKind: "human-operator", Actor: actor, RecordedAt: now}
	ref, e := executiongraph.SingleStageAuthorityDigest(a)
	if e != nil {
		return stageexecution.SinglePublication{}, stageexecution.ErrPreview
	}
	run := executiongraph.SingleStageRun{Kind: executiongraph.SingleStageRunKind, FormatVersion: executiongraph.OperationalSingleStageRunVersion, ScopeDigest: digest, Scope: scope, Binding: binding, CreatedAt: now, Authorities: []string{ref}, Attempts: []executiongraph.SuccessorAttempt{}}
	if executiongraph.ValidateSingleStageBindingAuthority(run, a) != nil {
		return stageexecution.SinglePublication{}, stageexecution.ErrPreview
	}
	return stageexecution.SinglePublication{Run: run, Authority: a, Execution: state}, nil
}
