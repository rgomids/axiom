package main

import (
	"context"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/stageexecution"
	"os"
)

type stageSupervisorLoader interface {
	LoadSupervisor(context.Context, stageexecution.SupervisorRequest) (stageexecution.Supervisor, error)
	LoadSingleSupervisor(context.Context, stageexecution.SingleRequest) (stageexecution.SingleSupervisor, error)
}

func (s LocalSupervisorStore) LoadRetry(ctx context.Context, r stageexecution.SupervisorRequest, ref string) (executiongraph.RetryAuthority, error) {
	return s.Graphs.LoadRetry(ctx, r.ProjectID, r.ParentID, ref)
}

// The current lease is captured by the same store that owns CAS persistence.
// Preparation never acquires a second lease and ingestion uses this exact one.
type activeStageStore struct {
	LocalSupervisorStore
	lease *local.RunLease
}

func (s *activeStageStore) Acquire(ctx context.Context, g executiongraph.SuccessorGraph) (stageexecution.SupervisorLease, error) {
	l, err := s.LocalSupervisorStore.Acquire(ctx, g)
	if err == nil {
		s.lease = l.(localSupervisorLease).lease
	}
	return l, err
}

type productionSupervisorLoader struct{}

func (productionSupervisorLoader) LoadSupervisor(ctx context.Context, r stageexecution.SupervisorRequest) (stageexecution.Supervisor, error) {
	s, ok := composeWithProvenance(currentProvenance()).(lifecycleService)
	if !ok {
		return stageexecution.Supervisor{}, stageexecution.ErrAdmission
	}
	stores, err := s.stageStores()
	if err != nil {
		return stageexecution.Supervisor{}, err
	}
	preview, err := retainedPreview(ctx, stores.graphs, r)
	if err != nil {
		return stageexecution.Supervisor{}, err
	}
	states, err := stores.executions.List(ctx, r.ProjectID)
	if err != nil {
		return stageexecution.Supervisor{}, err
	}
	var t stageTarget
	for _, state := range states {
		if state.ExecutionID != preview.Plan.Execution.ExecutionID {
			continue
		}
		item := state.WorkItem
		input := cli.WorkflowStageInput{Project: state.ProjectID, Repository: state.RepositoryKey, WorkItem: item.Provider + ":" + item.Resource + "#" + item.ExternalID, Execution: state.ExecutionID, ExpectedRevision: preview.Plan.Execution.Revision}
		var failure *cli.WorkflowStageResponse
		t, failure = s.resolveStageTarget(ctx, input, projectapp.AdmitExecutionAdvance)
		if failure != nil || t.request.ParentID != r.ParentID {
			return stageexecution.Supervisor{}, stageexecution.ErrAdmission
		}
		break
	}
	if t.state.ExecutionID == "" {
		return stageexecution.Supervisor{}, stageexecution.ErrAdmission
	}
	// The committed dispatch authority is the inspection consent boundary;
	// every child is a retained, digest-verified preview child.
	if len(preview.Children) == 0 {
		return stageexecution.Supervisor{}, stageexecution.ErrAdmission
	}
	sources, err := productionEffortInspector(s.stateRoot, t.repository.Path, preview.Children)
	if err != nil {
		return stageexecution.Supervisor{}, err
	}
	policy, err := s.stageRuntimePolicy(ctx, t.repository.Path, true)
	if err != nil {
		return stageexecution.Supervisor{}, err
	}
	store := &activeStageStore{LocalSupervisorStore: s.supervisorStore(stores, t)}
	loader := stageAttemptLoader{s: s, stores: stores, target: t, store: store, preview: preview}
	auth, err := runtimeadapter.NewAuthPreflight(runtimeadapter.OSStatusRunner{}, runtimeadapter.DefaultManagedConfiguration())
	if err != nil {
		return stageexecution.Supervisor{}, err
	}
	preparation := &stageexecution.CompiledSupervisorPreparation{Loader: loader, Checker: policy, Auth: auth, Sources: sources}
	return stageexecution.Supervisor{Store: store, Preparation: stageContextPreparation{next: preparation, stateRoot: s.stateRoot, projectsRoot: s.projectsRoot, skillsRoot: s.skillsRoot}, Runner: executiongraph.OSProcessRunner{}, Reset: stageDisposer{t.repository.Path}, AllocateID: allocateStageID}, nil
}

// Only installation-owned roots are added after the adapter constructs its
// clean environment. Correlation and vendor settings still belong to the
// frozen adapter; ambient credentials and arbitrary AXIOM_* are never copied.
type stageContextPreparation struct {
	next                                stageexecution.SupervisorPreparation
	stateRoot, projectsRoot, skillsRoot string
}

func (p stageContextPreparation) Prepare(ctx context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, id string, epoch uint64) (executiongraph.Invocation, error) {
	inv, err := p.next.Prepare(ctx, g, c, id, epoch)
	if err != nil {
		return inv, err
	}
	env := executiongraph.EffectiveInvocationEnvironment(inv)
	for _, entry := range []string{"LINGO_STATE_ROOT=" + p.stateRoot, "LINGO_PROJECTS_ROOT=" + p.projectsRoot, "AXIOM_CODEX_SKILLS_ROOT=" + p.skillsRoot} {
		env = append(env, entry)
	}
	return executiongraph.WithExactEnvironment(inv, env), nil
}
func (p stageContextPreparation) Drain(ctx context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, a executiongraph.SuccessorAttempt, final bool) ([]executiongraph.AttemptRecord, error) {
	return p.next.Drain(ctx, g, c, a, final)
}

type stageAttemptLoader struct {
	s       lifecycleService
	stores  stageStores
	target  stageTarget
	store   *activeStageStore
	preview stageexecution.DispatchPreview
}

func (l stageAttemptLoader) LoadAttempt(ctx context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, id string, epoch uint64) (stageexecution.AttemptMaterial, error) {
	if l.store.lease == nil || l.store.lease.Epoch != epoch {
		return stageexecution.AttemptMaterial{}, stageexecution.ErrSupervisorAuthority
	}
	m, projection, err := (gitIntegration{l.target.repository.Path}).manager(ctx, g)
	if err != nil {
		return stageexecution.AttemptMaterial{}, err
	}
	for _, child := range projection.Children {
		if child.ExecutionID == c.ExecutionID {
			err = m.InspectRetryWorkspace(ctx, child)
			break
		}
	}
	if err != nil {
		return stageexecution.AttemptMaterial{}, err
	}
	var material stageexecution.AttemptMaterial
	for _, a := range l.preview.Plan.Compilation.Agents {
		if a.Input.NodeKey == c.NodeKey {
			material.Input = a.Input
			break
		}
	}
	if material.Input.FrozenProfile == nil {
		return material, stageexecution.ErrAdmission
	}
	frozen := *material.Input.FrozenProfile
	executable, err := l.s.runtimes.lookPath(frozen.RuntimeID)
	if err != nil {
		return material, err
	}
	axiom, err := os.Executable()
	if err != nil {
		return material, err
	}
	cctx := coordination.ChildContext{Workspace: c.Envelope.Workspace, Inbox: attemptInbox(c.Envelope.Workspace, id), ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID, AttemptID: id, GraphRevision: g.Parent.GraphRevision, Epoch: epoch, Running: true}
	inbox, err := local.NewAttemptInbox(cctx)
	if err != nil {
		return material, err
	}
	if err = inbox.Create(ctx, cctx); err != nil {
		return material, err
	}
	canonical, err := local.NewCoordinationStore(l.s.stateRoot, l.target.state.ProjectID)
	if err != nil {
		return material, err
	}
	fence := local.CoordinationFence{Lease: l.store.lease, Current: func(ctx context.Context) (coordination.ChildContext, error) {
		fresh, err := l.stores.graphs.LoadSuccessor(ctx, l.target.state.ProjectID, g.Parent.ExecutionID)
		if err != nil {
			return coordination.ChildContext{}, err
		}
		for _, child := range fresh.Children {
			if child.ExecutionID == c.ExecutionID && len(child.Attempts) > 0 {
				at := child.Attempts[len(child.Attempts)-1]
				if at.AttemptID == id && at.LeaseEpoch == epoch && at.Status == executiongraph.AttemptRunning {
					return cctx, nil
				}
			}
		}
		return coordination.ChildContext{}, coordination.ErrEpochFenced
	}}
	material.Context = cctx
	material.Ingestor = coordination.Ingestor{Service: coordination.New(canonical, allocateStageID, nil), Reader: canonical, Inbox: inbox, Fence: fence, MaximumAttempts: c.Envelope.Controls.MaximumAttempts}
	for _, child := range g.Children {
		material.Ingestor.ReferenceChildren = append(material.Ingestor.ReferenceChildren, child.ExecutionID)
	}
	material.Packet = runtimeadapter.PacketContext{AttemptID: id, Inbox: cctx.Inbox, AxiomPath: axiom, AxiomDigest: l.preview.AxiomDigest}
	material.Invocation = runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: frozen.RuntimeID, ModelProfileID: frozen.ProfileID, Executable: executable, Model: frozen.Model}, Workspace: cctx.Workspace, Inbox: cctx.Inbox, AxiomPath: axiom, ParentEnvironment: os.Environ(), Correlation: runtimeadapter.AttemptCorrelation{ProjectID: l.target.state.ProjectID, ParentID: cctx.ParentID, ChildID: cctx.ChildID, AttemptID: id}}
	return material, nil
}

// The stage-run composition owns the loader. A shell invocation cannot supply
// a runner, paths, configuration or authority. Absent composition fails closed.
func stageSupervisorCommand(ctx context.Context, args []string, loader stageSupervisorLoader) (bool, int) {
	if len(args) == 0 || args[0] != "__stage-supervisor" {
		return false, 0
	}
	if loader == nil {
		return true, 1
	}
	if len(args) == 6 && args[1] == "single" {
		r := stageexecution.SingleRequest{ProjectID: args[2], ExecutionID: args[3], ScopeDigest: args[4], AuthorityRef: args[5]}
		s, e := loader.LoadSingleSupervisor(ctx, r)
		if e != nil || s.Run(ctx, r) != nil {
			return true, 1
		}
		return true, 0
	}
	if len(args) != 5 && len(args) != 6 {
		return true, 1
	}
	r := stageexecution.SupervisorRequest{ProjectID: args[1], ParentID: args[2], PreviewDigest: args[3], AuthorityRef: args[4]}
	s, e := loader.LoadSupervisor(ctx, r)
	if e != nil {
		return true, 1
	}
	// A sixth argument names a committed retry receipt; it never widens the
	// first-attempt scope and is verified again under the run lease.
	if len(args) == 6 {
		e = s.RunRetry(ctx, r, args[5])
	} else {
		e = s.Run(ctx, r)
	}
	if e != nil {
		return true, 1
	}
	return true, 0
}
