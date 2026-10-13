package stageexecution

import (
	"context"
	"reflect"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/graphapplication"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflow"
)

type SingleAttemptLoader interface {
	LoadSingleAttempt(context.Context, workflow.SingleDispatch, executiongraph.DispatchAuthority) (AttemptMaterial, error)
}

// SingleCompletion drains before using the shared classifier. The sequential
// supervisor must atomically persist this classification and qualifying record.
type SingleCompletion func(context.Context, executiongraph.ProcessResult, bool) (executiongraph.AttemptStatus, executiongraph.ClassificationReason, string, error)
type SingleInvocationSupervisor interface {
	StartSingleInvocation(context.Context, workflow.SingleDispatch, executiongraph.DispatchAuthority, executiongraph.Invocation, SingleCompletion) (executiongraph.ProcessCorrelation, error)
}

// CompiledSingleStarter adapts the sequential seam to the same T21/T19/T20/T23
// implementation used by graph supervision. It allocates no graph or child.
type CompiledSingleStarter struct {
	Loader     SingleAttemptLoader
	Checker    graphapplication.StageBindingChecker
	Sources    graphapplication.EffortSourceInspector
	Auth       graphapplication.SubscriptionAuthenticator
	Recorder   runtimeadapter.TaskPacketRecorder
	Supervisor SingleInvocationSupervisor
}

func (s CompiledSingleStarter) StartSingle(ctx context.Context, single workflow.SingleDispatch, a executiongraph.DispatchAuthority) (executiongraph.ProcessCorrelation, error) {
	if s.Loader == nil || s.Supervisor == nil || s.Recorder == nil || single.ResolvedProfile == nil {
		return executiongraph.ProcessCorrelation{}, ErrPreview
	}
	m, e := s.Loader.LoadSingleAttempt(ctx, single, a)
	if e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	frozen := *single.ResolvedProfile
	inputDigest, e := canonicalDigest(m.Input)
	if e != nil || inputDigest != single.InputDigest {
		return executiongraph.ProcessCorrelation{}, ErrStalePreview
	}
	if m.Input.ResolvedProfile == nil || *m.Input.ResolvedProfile != frozen || m.Input.StageID != string(single.State.Stage) || m.Input.ProjectID != single.State.ProjectID || m.Input.RepositoryKey != single.State.RepositoryKey || m.Invocation.Workspace != single.Workspace || m.Invocation.Inbox != single.Inbox || m.Packet.AttemptID != single.AttemptID || m.Packet.Inbox != single.Inbox || m.Packet.AxiomDigest != single.AxiomDigest || m.Packet.AxiomPath != m.Invocation.AxiomPath || m.Context.ParentID != single.State.ExecutionID || m.Context.ChildID != single.State.ExecutionID || m.Context.AttemptID != single.AttemptID || m.Context.GraphRevision != single.State.Revision || m.Context.Workspace != single.Workspace || m.Context.Inbox != single.Inbox || !coordination.ValidChildContext(m.Context) || len(a.Children) != 1 || !reflect.DeepEqual(m.Input.Effects, a.Children[0].Effects) {
		return executiongraph.ProcessCorrelation{}, ErrPreview
	}
	m.Invocation.RepositoryWrite = false
	m.Invocation.Process = false
	for _, effect := range m.Input.Effects {
		if effect.Kind == "repository-write" {
			m.Invocation.RepositoryWrite = true
		}
		if effect.Kind == "process" {
			m.Invocation.Process = true
		}
	}
	inv, _, e := graphapplication.PrepareFrozenStageInvocation(ctx, m.Invocation, frozen, m.Input.RuntimePreview, s.Checker, s.Sources, s.Auth)
	if e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	inv, _, e = graphapplication.PrepareCompiledTaskPacket(ctx, inv, m.Input, m.Packet, s.Recorder)
	if e != nil {
		return executiongraph.ProcessCorrelation{}, e
	}
	complete := func(ctx context.Context, process executiongraph.ProcessResult, cancelled bool) (executiongraph.AttemptStatus, executiongraph.ClassificationReason, string, error) {
		records, e := m.Ingestor.Drain(ctx, m.Context, true)
		if e != nil {
			return "", "", "", e
		}
		status, reason, ref := executiongraph.ClassifyAttempt(process, cancelled, executiongraph.AttemptCorrelation{ParentID: single.State.ExecutionID, ChildID: single.State.ExecutionID, AttemptID: single.AttemptID, GraphRevision: single.State.Revision}, records)
		return status, reason, ref, nil
	}
	return s.Supervisor.StartSingleInvocation(ctx, single, a, inv, complete)
}
