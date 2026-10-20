package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/stagepublication"
	"github.com/rgomids/axiom/internal/workflow"
)

// loadStageSingle returns the single-stage ledger of the Execution's current
// stage, or nil when that stage has none. Corruption is never treated as absent.
func loadStageSingle(ctx context.Context, singles local.SingleStageStore, state workflow.State) (*executiongraph.SingleStageRun, error) {
	digest, err := executiongraph.SingleStageScopeDigest(stagepublication.SingleScope(state))
	if err != nil {
		return nil, nil
	}
	run, err := singles.Load(ctx, state.ProjectID, digest)
	if errors.Is(err, local.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func singleRequest(run executiongraph.SingleStageRun) stageexecution.SingleRequest {
	return stageexecution.SingleRequest{ProjectID: run.Scope.ProjectID, ExecutionID: run.Scope.ExecutionID, ScopeDigest: run.ScopeDigest, AuthorityRef: run.Authorities[len(run.Authorities)-1]}
}

// localSingleStore is the protected SingleStore of one resolved target. Its
// contract reloads Execution3 and Runtime configuration on every call.
type localSingleStore struct {
	s      lifecycleService
	stores stageStores
	target stageTarget
	lease  *local.RunLease
}

func (l *localSingleStore) LoadSingle(ctx context.Context, r stageexecution.SingleRequest) (executiongraph.SingleStageRun, executiongraph.SingleStageAuthority, error) {
	run, err := l.stores.singles.Load(ctx, r.ProjectID, r.ScopeDigest)
	if err != nil {
		return run, executiongraph.SingleStageAuthority{}, err
	}
	if run.Scope.ExecutionID != r.ExecutionID {
		return run, executiongraph.SingleStageAuthority{}, stageexecution.ErrSupervisorAuthority
	}
	a, err := l.stores.singles.LoadAuthority(ctx, run, r.AuthorityRef)
	return run, a, err
}
func (l *localSingleStore) SingleContract(ctx context.Context, r stageexecution.SingleRequest) (stageexecution.Contract, error) {
	if r.ProjectID != l.target.state.ProjectID || r.ExecutionID != l.target.state.ExecutionID {
		return stageexecution.Contract{}, stageexecution.ErrAdmission
	}
	state, err := l.stores.executions.Load(ctx, l.target.state.ProjectID, l.target.state.RepositoryKey, l.target.state.WorkItem)
	if err != nil {
		return stageexecution.Contract{}, err
	}
	return l.s.stageContract(ctx, l.stores, state)
}
func (l *localSingleStore) AcquireSingle(ctx context.Context, run executiongraph.SingleStageRun) (stageexecution.SupervisorLease, error) {
	lease, err := l.stores.leases.Acquire(ctx, run.Scope.ProjectID, run.Scope.ExecutionID, run.RunEpoch)
	if err != nil {
		return nil, err
	}
	l.lease = lease
	return localSupervisorLease{lease}, nil
}
func (l *localSingleStore) SaveSingle(ctx context.Context, run executiongraph.SingleStageRun, lease stageexecution.SupervisorLease) (executiongraph.SingleStageRun, error) {
	held, ok := lease.(localSupervisorLease)
	if !ok {
		return run, stageexecution.ErrSupervisorRecovery
	}
	return l.stores.singles.Save(ctx, run, held.lease)
}

// localSingleLedger is the lifecycle port: cancel, reconcile and retry
// receipts, each through the store's own fenced write.
type localSingleLedger struct {
	*localSingleStore
	observer singleObserver
}

func (l localSingleLedger) SingleSnapshot(ctx context.Context) (stageexecution.Contract, executiongraph.SingleStageRun, error) {
	run, err := l.stores.singles.Load(ctx, l.target.single.Scope.ProjectID, l.target.single.ScopeDigest)
	if err != nil {
		return stageexecution.Contract{}, run, err
	}
	c, err := l.s.stageContract(ctx, l.stores, l.target.state)
	return c, run, err
}
func (l localSingleLedger) RequestSingleCancel(ctx context.Context, run executiongraph.SingleStageRun, r executiongraph.CancelRequest) (executiongraph.SingleStageRun, error) {
	return l.stores.singles.RequestCancel(ctx, run, r)
}
func (l localSingleLedger) AppendSingleReconciliation(ctx context.Context, run executiongraph.SingleStageRun, f executiongraph.ReconciliationFact) (executiongraph.SingleStageRun, error) {
	return l.stores.singles.AppendReconciliation(ctx, run, f, l.observer.ObserveSingle)
}
func (l localSingleLedger) CommitSingleRetry(ctx context.Context, next executiongraph.SingleStageRun, a executiongraph.SingleStageAuthority, state workflow.State) error {
	lease, err := l.stores.leases.Acquire(ctx, next.Scope.ProjectID, next.Scope.ExecutionID, next.RunEpoch)
	if err != nil {
		return err
	}
	defer lease.Close()
	return l.stores.singles.CommitAuthority(ctx, next, a, state, lease)
}

// singleObserver gathers the same bounded digest-only observations as the
// graph observer, keyed by the Execution as both parent and Runtime identity.
type singleObserver struct {
	stateRoot string
	leases    local.RunLeaseStore
}

func (o singleObserver) ObserveSingle(ctx context.Context, run executiongraph.SingleStageRun, attemptID string) (executiongraph.ReconciliationObservations, error) {
	id := run.Scope.ExecutionID
	return executiongraph.GatherSingleStageObservations(run, attemptID, executiongraph.ReconciliationSources{
		LeaseProbe:   func() (bool, error) { return o.leases.Probe(ctx, run.Scope.ProjectID, id) },
		ProcessProbe: executiongraph.ObserveRecordedProcess,
		Worktree:     func() (string, error) { return worktreeDigest(ctx, run.Binding.Workspace) },
		CanonicalRecords: func() (string, error) {
			return canonicalStreamDigest(ctx, o.stateRoot, run.Scope.ProjectID, id, id)
		},
		QuarantinedInbox: func() (string, error) { return inboxDigest(attemptInbox(run.Binding.Workspace, attemptID)), nil },
	})
}

func canonicalStreamDigest(ctx context.Context, stateRoot, projectID, parentID, childID string) (string, error) {
	store, err := local.NewCoordinationStore(stateRoot, projectID)
	if err != nil {
		return "", err
	}
	stream, found, err := store.ReadStream(ctx, parentID, childID)
	if err != nil {
		return "", err
	}
	if !found {
		return digestText("absent"), nil
	}
	wire, err := json.Marshal(stream)
	return digestText(string(wire)), err
}

// singleChild projects the ledger binding onto the workspace manager's
// in-memory child shape. Nothing is persisted as a graph.
func singleChild(run executiongraph.SingleStageRun) executiongraph.ChildExecution {
	id := run.Scope.ExecutionID
	b := run.Binding
	return executiongraph.ChildExecution{ExecutionID: id, ParentID: id, GraphRevision: b.WorkflowRevision, NodeKey: run.Scope.StageID, Envelope: executiongraph.ChildEnvelope{Workspace: b.Workspace, AllowedEffects: b.Effects, Controls: b.Controls, Resolution: b.Resolution, Scope: executiongraph.Scope{ProjectID: run.Scope.ProjectID, RepositoryKey: run.Scope.RepositoryKey}}}
}

func singleManager(ctx context.Context, repository string, run executiongraph.SingleStageRun) (*gitworkspace.Manager, executiongraph.ChildExecution, error) {
	head, err := gitHead(ctx, repository)
	if err != nil {
		return nil, executiongraph.ChildExecution{}, err
	}
	child := singleChild(run)
	m, err := gitworkspace.NewSingleManager(ctx, repository, filepath.Dir(filepath.Dir(run.Binding.Workspace)), head, child)
	return m, child, err
}

func prepareStageDirectories(root, workspace string) error {
	anchor, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer anchor.Close()
	for _, name := range []string{"worktrees", "inboxes", filepath.Join("inboxes", filepath.Base(workspace))} {
		if err = privateInboxDirectory(anchor, name); err != nil {
			return err
		}
	}
	return nil
}

func (w stageWorkspaces) prepareSingle(ctx context.Context, run executiongraph.SingleStageRun) error {
	if err := prepareStageDirectories(filepath.Dir(filepath.Dir(run.Binding.Workspace)), run.Binding.Workspace); err != nil {
		return err
	}
	m, child, err := singleManager(ctx, w.repository, run)
	if err != nil {
		return err
	}
	return m.EnsureWorkspace(ctx, child)
}

type singleDisposer struct{ repository string }

func (d singleDisposer) DisposeSingle(ctx context.Context, run executiongraph.SingleStageRun, a executiongraph.SingleStageAuthority) error {
	if a.Disposition != executiongraph.DispositionPreserve && a.Disposition != executiongraph.DispositionReset {
		return stageexecution.ErrDispositionRequired
	}
	m, child, err := singleManager(ctx, d.repository, run)
	if err != nil {
		return err
	}
	return m.InspectRetryWorkspace(ctx, child)
}

func (d singleDisposer) ResetSingle(ctx context.Context, run executiongraph.SingleStageRun, guard func(context.Context) error) error {
	m, child, err := singleManager(ctx, d.repository, run)
	if err != nil {
		return err
	}
	return m.ResetWorkspace(ctx, child, guard)
}

func (s lifecycleService) singleLifecycle(stores stageStores, t stageTarget, dispatch stageexecution.PreviewService) stageexecution.SingleLifecycle {
	store := &localSingleStore{s: s, stores: stores, target: t}
	observer := singleObserver{stateRoot: s.stateRoot, leases: stores.leases}
	return stageexecution.SingleLifecycle{Ledger: localSingleLedger{localSingleStore: store, observer: observer}, Observer: observer, Dispatch: dispatch, Disposer: singleDisposer{t.repository.Path}, Store: store, Detached: stageexecution.SameBinaryDetacher{}, AllocateID: allocateStageID}
}

func (s lifecycleService) singleStatus(ctx context.Context, stores stageStores, t stageTarget) cli.WorkflowStageResponse {
	result, err := s.singleLifecycle(stores, t, stageexecution.PreviewService{}).Status(ctx, localStatusObserver{leases: stores.leases})
	if err != nil {
		return s.stageError(err, nil)
	}
	return s.stageSuccess("stage_status", "Persisted single-stage state", []string{"execution:" + t.state.ExecutionID, "single-stage-run:" + t.single.ScopeDigest}, "Status renders persisted fields only; it never claims a stop or success", result)
}

func (s lifecycleService) singleCancel(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	service := s.singleLifecycle(stores, t, stageexecution.PreviewService{})
	preview, err := service.CancelPreview(ctx)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Cancellation preview", "Repeat with --preview-digest and --authorize-local to publish the request", preview)
	}
	request, err := service.Cancel(ctx, in.PreviewDigest, in.AuthorizeLocal, stageActor())
	if err != nil {
		return s.stageError(err, nil)
	}
	return s.stageSuccess("cancellation_requested", "Cancellation requested (not an observed stop)", []string{"cancel-request:" + request.RequestID}, "Inspect stage status; an attempt is cancelled only after a confirmed stop", map[string]any{"request": request})
}

func (s lifecycleService) singleReconcile(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	service := s.singleLifecycle(stores, t, stageexecution.PreviewService{})
	preview, err := service.ReconcilePreview(ctx, in.Attempt)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Reconciliation preview", "Repeat with --preview-digest and --authorize-local to append the fact; status never changes", preview)
	}
	fact, err := service.Reconcile(ctx, in.Attempt, in.PreviewDigest, in.AuthorizeLocal, stageActor())
	if err != nil {
		return s.stageError(err, nil)
	}
	return s.stageSuccess("reconciliation_recorded", "Reconciliation fact appended (status unchanged)", []string{"reconciliation-fact:" + fact.FactID}, "A retry still needs its own authorized stage retry", map[string]any{"fact": fact})
}

func (s lifecycleService) singleRetry(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	dispatch, err := s.dispatchPreview(ctx, stores, t, in)
	if err != nil {
		return s.stageError(err, nil)
	}
	service := s.singleLifecycle(stores, t, dispatch)
	preview, err := service.RetryPreview(ctx, t.ref, in.Stage)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Stage retry preview", "Review attempt budget and workspace disposition; repeat with both authority flags", preview)
	}
	result, err := service.Retry(ctx, t.ref, in.Stage, in.PreviewDigest, in.AuthorizeLocal, in.AuthorizeExternal, stageActor(), in.Disposition)
	if err != nil {
		return s.stageDispatchError(err, result.Status, result)
	}
	return s.stageSuccess("stage_retry_dispatched", "Committed retry authority and started its supervisor", []string{"retry-authority:" + result.AuthorityRef}, "Inspect stage status; retry start does not imply success", result)
}

// retainedSinglePreview verifies the retained preview bytes against the
// ledger's bound digest before the detached process consumes task content.
func retainedSinglePreview(ctx context.Context, singles local.SingleStageStore, run executiongraph.SingleStageRun) (stageexecution.DispatchPreview, error) {
	d, err := singles.LoadDispatchInputs(ctx, run.Scope.ProjectID, run.Scope.ExecutionID, run.Binding.PreviewDigest)
	if err != nil {
		return stageexecution.DispatchPreview{}, err
	}
	return verifiedPreview(d.Preview, run.Binding.PreviewDigest)
}

func (productionSupervisorLoader) LoadSingleSupervisor(ctx context.Context, r stageexecution.SingleRequest) (stageexecution.SingleSupervisor, error) {
	s, ok := composeWithProvenance(currentProvenance()).(lifecycleService)
	if !ok {
		return stageexecution.SingleSupervisor{}, stageexecution.ErrAdmission
	}
	stores, err := s.stageStores()
	if err != nil {
		return stageexecution.SingleSupervisor{}, err
	}
	run, err := stores.singles.Load(ctx, r.ProjectID, r.ScopeDigest)
	if err != nil || run.Scope.ExecutionID != r.ExecutionID {
		return stageexecution.SingleSupervisor{}, stageexecution.ErrSupervisorAuthority
	}
	preview, err := retainedSinglePreview(ctx, stores.singles, run)
	if err != nil || len(preview.Children) != 1 || len(preview.Plan.Compilation.Agents) != 1 {
		return stageexecution.SingleSupervisor{}, stageexecution.ErrSupervisorAuthority
	}
	input := cli.WorkflowStageInput{Project: run.Scope.ProjectID, Repository: run.Scope.RepositoryKey, WorkItem: run.Scope.WorkItem, Execution: run.Scope.ExecutionID, ExpectedRevision: run.Binding.WorkflowRevision}
	t, failure := s.resolveStageTarget(ctx, input, projectapp.AdmitExecutionAdvance)
	if failure != nil || t.single == nil || t.single.ScopeDigest != run.ScopeDigest {
		return stageexecution.SingleSupervisor{}, stageexecution.ErrAdmission
	}
	sources, err := productionEffortInspector(s.stateRoot, t.repository.Path, preview.Children)
	if err != nil {
		return stageexecution.SingleSupervisor{}, err
	}
	policy, err := s.stageRuntimePolicy(ctx, t.repository.Path, true)
	if err != nil {
		return stageexecution.SingleSupervisor{}, err
	}
	auth, err := runtimeadapter.NewAuthPreflight(runtimeadapter.OSStatusRunner{}, runtimeadapter.DefaultManagedConfiguration())
	if err != nil {
		return stageexecution.SingleSupervisor{}, err
	}
	store := &localSingleStore{s: s, stores: stores, target: t}
	preparation := &stageexecution.CompiledSinglePreparation{Loader: singleAttemptLoader{s: s, stores: stores, target: t, store: store, preview: preview}, Checker: policy, Auth: auth, Sources: sources}
	return stageexecution.SingleSupervisor{Store: store, Preparation: singleContextPreparation{next: preparation, stateRoot: s.stateRoot, projectsRoot: s.projectsRoot, skillsRoot: s.skillsRoot}, Runner: executiongraph.OSProcessRunner{}, Reset: singleDisposer{t.repository.Path}, AllocateID: allocateStageID}, nil
}

type singleContextPreparation struct {
	next                                stageexecution.SinglePreparation
	stateRoot, projectsRoot, skillsRoot string
}

func (p singleContextPreparation) PrepareSingle(ctx context.Context, run executiongraph.SingleStageRun, id string, epoch uint64) (executiongraph.Invocation, error) {
	inv, err := p.next.PrepareSingle(ctx, run, id, epoch)
	if err != nil {
		return inv, err
	}
	env := append(executiongraph.EffectiveInvocationEnvironment(inv), "LINGO_STATE_ROOT="+p.stateRoot, "LINGO_PROJECTS_ROOT="+p.projectsRoot, "AXIOM_CODEX_SKILLS_ROOT="+p.skillsRoot)
	return executiongraph.WithExactEnvironment(inv, env), nil
}
func (p singleContextPreparation) DrainSingle(ctx context.Context, run executiongraph.SingleStageRun, a executiongraph.SuccessorAttempt, final bool) ([]executiongraph.AttemptRecord, error) {
	return p.next.DrainSingle(ctx, run, a, final)
}

type singleAttemptLoader struct {
	s       lifecycleService
	stores  stageStores
	target  stageTarget
	store   *localSingleStore
	preview stageexecution.DispatchPreview
}

func (l singleAttemptLoader) LoadSingleAttempt(ctx context.Context, run executiongraph.SingleStageRun, id string, epoch uint64) (stageexecution.AttemptMaterial, error) {
	var material stageexecution.AttemptMaterial
	if l.store.lease == nil || l.store.lease.Epoch != epoch {
		return material, stageexecution.ErrSupervisorAuthority
	}
	m, child, err := singleManager(ctx, l.target.repository.Path, run)
	if err != nil {
		return material, err
	}
	if err = m.InspectRetryWorkspace(ctx, child); err != nil {
		return material, err
	}
	material.Input = l.preview.Plan.Compilation.Agents[0].Input
	if material.Input.FrozenProfile == nil {
		return material, stageexecution.ErrAdmission
	}
	frozen := *material.Input.FrozenProfile
	executable, err := l.s.runtimes.lookPath(frozen.RuntimeID)
	if err != nil {
		return material, runtimeadapter.ErrExecutableUnavailable
	}
	axiom, err := os.Executable()
	if err != nil {
		return material, err
	}
	execution := run.Scope.ExecutionID
	cctx := coordination.ChildContext{Workspace: run.Binding.Workspace, Inbox: attemptInbox(run.Binding.Workspace, id), ParentID: execution, ChildID: execution, AttemptID: id, GraphRevision: run.Binding.WorkflowRevision, Epoch: epoch, Running: true}
	inbox, err := local.NewAttemptInbox(cctx)
	if err != nil {
		return material, err
	}
	if err = inbox.Create(ctx, cctx); err != nil {
		return material, err
	}
	canonical, err := local.NewCoordinationStore(l.s.stateRoot, run.Scope.ProjectID)
	if err != nil {
		return material, err
	}
	fence := local.CoordinationFence{Lease: l.store.lease, Current: func(ctx context.Context) (coordination.ChildContext, error) {
		fresh, err := l.stores.singles.Load(ctx, run.Scope.ProjectID, run.ScopeDigest)
		if err == nil && len(fresh.Attempts) > 0 {
			at := fresh.Attempts[len(fresh.Attempts)-1]
			if at.AttemptID == id && at.LeaseEpoch == epoch && at.Status == executiongraph.AttemptRunning {
				return cctx, nil
			}
		}
		return coordination.ChildContext{}, coordination.ErrEpochFenced
	}}
	material.Context = cctx
	material.Ingestor = coordination.Ingestor{Service: coordination.New(canonical, allocateStageID, nil), Reader: canonical, Inbox: inbox, Fence: fence, MaximumAttempts: run.Binding.Controls.MaximumAttempts, ReferenceChildren: []string{execution}}
	material.Packet = runtimeadapter.PacketContext{AttemptID: id, Inbox: cctx.Inbox, AxiomPath: axiom, AxiomDigest: run.Binding.AxiomDigest}
	material.Invocation = runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: frozen.RuntimeID, ModelProfileID: frozen.ProfileID, Executable: executable, Model: frozen.Model}, Workspace: cctx.Workspace, Inbox: cctx.Inbox, AxiomPath: axiom, ParentEnvironment: os.Environ(), Correlation: runtimeadapter.AttemptCorrelation{ProjectID: run.Scope.ProjectID, ParentID: execution, ChildID: execution, AttemptID: id}}
	return material, nil
}
