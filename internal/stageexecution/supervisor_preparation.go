package stageexecution

import (
	"context"
	"reflect"

	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/graphapplication"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

// AttemptMaterial is supplied by the trusted stage publication loader. Inputs
// remain bound to the frozen child; no raw CLI/task data can select a Runtime.
type AttemptMaterial struct {
	Input      workflowcompiler.ExecutionInput
	Invocation runtimeadapter.StageInvocationRequest
	Packet     runtimeadapter.PacketContext
	Ingestor   coordination.Ingestor
	Context    coordination.ChildContext
}
type AttemptMaterialLoader interface {
	LoadAttempt(context.Context, executiongraph.SuccessorGraph, executiongraph.SuccessorChild, string, uint64) (AttemptMaterial, error)
}

// CompiledSupervisorPreparation composes T21, T19, T20 and T23. The loader
// owns workspace validation and inbox creation under the held lease. This
// component never rereads mutable Workflow intent or resolves a replacement.
type CompiledSupervisorPreparation struct {
	Loader   AttemptMaterialLoader
	Checker  graphapplication.StageBindingChecker
	Sources  graphapplication.EffortSourceInspector
	Auth     graphapplication.SubscriptionAuthenticator
	attempts map[string]AttemptMaterial
}

func (p *CompiledSupervisorPreparation) Prepare(ctx context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, id string, epoch uint64) (executiongraph.Invocation, error) {
	if p == nil || p.Loader == nil || c.Envelope.Resolution.FrozenProfile == nil {
		return executiongraph.Invocation{}, ErrAdmission
	}
	m, e := p.Loader.LoadAttempt(ctx, g, c, id, epoch)
	if e != nil {
		return executiongraph.Invocation{}, e
	}
	frozen := *c.Envelope.Resolution.FrozenProfile
	if !executiongraph.ValidFrozenProfileControls(&frozen, c.Envelope.Controls, c.Envelope.Capability) {
		return executiongraph.Invocation{}, ErrAdmission
	}
	m.Invocation.Process = false
	m.Invocation.RepositoryWrite = false
	for _, effect := range c.Envelope.AllowedEffects {
		switch effect.Kind {
		case "process":
			m.Invocation.Process = true
		case "repository-write":
			m.Invocation.RepositoryWrite = true
		}
	}
	if m.Input.FrozenProfile == nil || *m.Input.FrozenProfile != frozen || m.Input.ProjectID != c.Envelope.Scope.ProjectID || m.Input.RepositoryKey != c.Envelope.Scope.RepositoryKey || m.Input.NodeKey != c.NodeKey || !reflect.DeepEqual(m.Input.Scope, c.Envelope.Scope) || !reflect.DeepEqual(m.Input.Effects, c.Envelope.AllowedEffects) || m.Input.Controls != c.Envelope.Controls || m.Invocation.Workspace != c.Envelope.Workspace || m.Context.Workspace != c.Envelope.Workspace || m.Context.Inbox != m.Invocation.Inbox || m.Packet.Inbox != m.Context.Inbox || m.Packet.AxiomPath != m.Invocation.AxiomPath || m.Context.ParentID != g.Parent.ExecutionID || m.Context.GraphRevision != g.Parent.GraphRevision || m.Context.ChildID != c.ExecutionID || m.Context.AttemptID != id || m.Packet.AttemptID != id || m.Context.Epoch != epoch || !coordination.ValidChildContext(m.Context) {
		return executiongraph.Invocation{}, ErrAdmission
	}
	inv, _, e := graphapplication.PrepareFrozenStageInvocation(ctx, m.Invocation, frozen, m.Input.RuntimePreview, p.Checker, p.Sources, p.Auth)
	if e != nil {
		return executiongraph.Invocation{}, e
	}
	inv, _, e = graphapplication.AttachCompiledTaskPacket(inv, m.Input, m.Packet)
	if e != nil {
		return executiongraph.Invocation{}, e
	}
	if p.attempts == nil {
		p.attempts = map[string]AttemptMaterial{}
	}
	if _, exists := p.attempts[id]; exists {
		return executiongraph.Invocation{}, ErrSupervisorRecovery
	}
	p.attempts[id] = m
	return inv, nil
}
func (p *CompiledSupervisorPreparation) Drain(ctx context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, a executiongraph.SuccessorAttempt, final bool) ([]executiongraph.AttemptRecord, error) {
	m, ok := p.attempts[a.AttemptID]
	if !ok || m.Context.ParentID != g.Parent.ExecutionID || m.Context.ChildID != c.ExecutionID || m.Context.Epoch != a.LeaseEpoch {
		return nil, coordination.ErrChildContext
	}
	records, e := m.Ingestor.Drain(ctx, m.Context, final)
	if e == nil && final {
		delete(p.attempts, a.AttemptID)
	}
	return records, e
}
