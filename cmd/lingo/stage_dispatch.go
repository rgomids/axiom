package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/stagepublication"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type dispatchObservation struct {
	s        lifecycleService
	stores   stageStores
	target   stageTarget
	input    cli.WorkflowStageInput
	policy   stagePolicy
	compiler workflowcompiler.Compiler
	root     string
}

func (o dispatchObservation) Observe(ctx context.Context, ref workflowcompiler.ExecutionRef, stage string) (stageexecution.PreviewObservation, error) {
	if ref != o.target.ref || stage != string(o.target.state.Stage) {
		return stageexecution.PreviewObservation{}, stageexecution.ErrStalePreview
	}
	provider, resource, id, ok := cli.ParseWorkItemSelector(o.input.WorkItem)
	if !ok {
		return stageexecution.PreviewObservation{}, stageexecution.ErrPreview
	}
	target := workflowTarget(cli.WorkflowInput{Project: o.target.state.ProjectID, Repository: o.target.state.RepositoryKey, Provider: provider, ProviderRepository: resource, ExternalID: id, Execution: ref.ExecutionID})
	planning, loaded := o.s.workflows.StagePlanningContext(ctx, target, ref.Revision, stage)
	if loaded.Status != workflow.Succeeded {
		return stageexecution.PreviewObservation{}, stageexecution.ErrStalePreview
	}
	contract, err := o.s.stageContract(ctx, o.stores, loaded.State)
	if err != nil {
		return stageexecution.PreviewObservation{}, err
	}
	document, digest, err := readPlanDocument(o.input.PlanFile)
	if err != nil {
		return stageexecution.PreviewObservation{}, err
	}
	if _, err = o.stores.policies.Load(ctx, o.input.ValidatorPolicy); err != nil {
		return stageexecution.PreviewObservation{}, stageexecution.ErrValidatorPolicy
	}
	r := workflowcompiler.StagePlanRequest{Execution: ref, PlanDocumentDigest: digest, Compile: workflowcompiler.Request{ProjectID: planning.ProjectID, RepositoryKey: planning.RepositoryKey, Workflow: planning.Binding.Definition, Definition: planning.Document, StageID: stage, Plan: document.Plan, ParentAuthority: document.AuthorityCeiling}}
	for _, b := range planning.Inputs {
		r.Compile.InputBindings = append(r.Compile.InputBindings, workflowcompiler.InputBinding{InputID: b.InputID, Reference: b.Reference, Digest: b.Digest})
	}
	for _, g := range planning.Gates {
		r.Gates = append(r.Gates, workflowcompiler.GateRef{Kind: g.Kind, Timing: g.Timing, Satisfied: g.Satisfied})
	}
	plan, err := o.compiler.PlanStageV2(ctx, r)
	if err != nil {
		return stageexecution.PreviewObservation{}, err
	}
	binary, err := os.Executable()
	if err != nil {
		return stageexecution.PreviewObservation{}, err
	}
	axiomDigest, err := runtimeadapter.ExecutableIdentity(binary)
	if err != nil {
		return stageexecution.PreviewObservation{}, err
	}
	out := stageexecution.PreviewObservation{Contract: contract, Request: r, Reviewed: plan, ValidatorPolicyDigest: o.input.ValidatorPolicy, AxiomDigest: axiomDigest, Workspaces: map[string]string{}, Inboxes: map[string]string{}}
	policyDigests := []string{}
	for _, a := range plan.Compilation.Agents {
		policyDigests = append(policyDigests, a.Input.RuntimePreview.ProjectDigest)
		key := a.Input.NodeKey
		out.Workspaces[key] = filepath.Join(o.root, "worktrees", digestText(ref.ExecutionID, stage, key)[:24])
		// Preview cannot allocate an attempt; the disclosed parent is its
		// eventual private per-attempt inbox namespace.
		out.Inboxes[key] = filepath.Join(o.root, "inboxes", digestText(ref.ExecutionID, stage, key)[:24])
	}
	out.PolicyDigest = digestText(policyDigests...)
	return out, nil
}

func (s lifecycleService) dispatchPreview(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) (stageexecution.PreviewService, error) {
	root := in.WorkspaceRoot
	if root == "" && t.single != nil {
		root = filepath.Dir(filepath.Dir(t.single.Binding.Workspace))
	}
	if root == "" {
		doc, err := stageGraph(ctx, stores, t.request)
		if err != nil || doc == nil || doc.Successor == nil {
			return stageexecution.PreviewService{}, stageexecution.ErrPreview
		}
		root, err = stageWorkspaceRoot(*doc.Successor)
		if err != nil {
			return stageexecution.PreviewService{}, err
		}
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return stageexecution.PreviewService{}, stageexecution.ErrPreview
	}
	for _, other := range []string{s.stateRoot, t.repository.Path} {
		if overlap, err := local.RootsOverlap(root, other); err != nil || overlap {
			return stageexecution.PreviewService{}, stageexecution.ErrPreview
		}
	}
	policy, err := s.stageRuntimePolicy(ctx, t.repository.Path, in.AuthorizeLocal)
	if err != nil {
		return stageexecution.PreviewService{}, err
	}
	compiler := workflowcompiler.New(policy, stagePolicyCapabilities{policy: policy, projectID: t.state.ProjectID})
	return stageexecution.PreviewService{Source: dispatchObservation{s: s, stores: stores, target: t, input: in, policy: policy, compiler: compiler, root: root}, Compiler: compiler}, nil
}

type stagePolicy struct {
	s        lifecycleService
	runtimes runtimeObservationSources
}

func (p stagePolicy) service() runtimeapplication.Service {
	if p.s.RuntimePolicySource != nil {
		return runtimeapplication.New(p.s.RuntimePolicySource)
	}
	return runtimeapplication.New(runtimePolicySource{installation: p.s.installation, portable: p.s.portable, stateRoot: p.s.stateRoot, runtimes: p.runtimes})
}
func (p stagePolicy) Preview(ctx context.Context, id string, r runtimeapplication.Request) (runtimeapplication.Preview, error) {
	return p.service().Preview(ctx, id, r)
}
func (p stagePolicy) Check(ctx context.Context, preview runtimeapplication.Preview) (runtimeapplication.Binding, error) {
	return p.service().Check(ctx, preview)
}

func (s lifecycleService) stageRuntimePolicy(ctx context.Context, repository string, consented bool) (stagePolicy, error) {
	if ctx.Err() != nil {
		return stagePolicy{}, ctx.Err()
	}
	return stagePolicy{s: s, runtimes: s.runtimeObservationSources().scoped(repository, consented)}, nil
}

type retainedPublisher struct {
	stagepublication.LocalPublisher
	preview stageexecution.DispatchPreview
}

func (p *retainedPublisher) Build(ctx context.Context, preview stageexecution.DispatchPreview, actor string, now time.Time) (stageexecution.Publication, error) {
	p.preview = preview
	return p.LocalPublisher.Build(ctx, preview, actor, now)
}
func (p *retainedPublisher) Commit(ctx context.Context, v stageexecution.Publication) error {
	wire, err := json.Marshal(p.preview)
	if err != nil {
		return err
	}
	if v.Single != nil {
		run := v.Single.Run
		if _, err = p.Singles.CommitDispatchInputs(ctx, executiongraph.DispatchInputs{FormatVersion: executiongraph.OperationalDispatchInputsVersion, ProjectID: run.Scope.ProjectID, ParentID: run.Scope.ExecutionID, PreviewDigest: run.Binding.PreviewDigest, Preview: wire}); err != nil {
			return err
		}
		return p.LocalPublisher.Commit(ctx, v)
	}
	if _, err = p.Graphs.CommitDispatchInputs(ctx, executiongraph.DispatchInputs{FormatVersion: executiongraph.OperationalDispatchInputsVersion, ProjectID: v.Authority.ProjectID, ParentID: v.Authority.ParentID, PreviewDigest: v.Authority.PreviewDigest, Preview: wire}); err != nil {
		return err
	}
	return p.LocalPublisher.Commit(ctx, v)
}

type stageWorkspaces struct{ repository string }

func (w stageWorkspaces) Prepare(ctx context.Context, p stageexecution.Publication) error {
	if p.Single != nil && p.Graph == nil {
		return w.prepareSingle(ctx, p.Single.Run)
	}
	if p.Graph == nil {
		return stageexecution.ErrPreview
	}
	root, err := stageWorkspaceRoot(*p.Graph)
	if err != nil {
		return err
	}
	anchor, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer anchor.Close()
	for _, name := range []string{"worktrees", "inboxes"} {
		if err = privateInboxDirectory(anchor, name); err != nil {
			return err
		}
	}
	for _, child := range p.Graph.Children {
		if err = privateInboxDirectory(anchor, filepath.Join("inboxes", filepath.Base(child.Envelope.Workspace))); err != nil {
			return err
		}
	}
	m, projection, err := (gitIntegration{repository: w.repository}).manager(ctx, *p.Graph)
	if err != nil {
		return err
	}
	for _, child := range projection.Children {
		if err := m.EnsureWorkspace(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

func privateInboxDirectory(root *os.Root, name string) error {
	if err := root.Mkdir(name, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := root.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return local.ErrUnsafe
	}
	return nil
}

func (s lifecycleService) supervisorStore(stores stageStores, t stageTarget) LocalSupervisorStore {
	return LocalSupervisorStore{Graphs: stores.graphs, Leases: stores.leases, ReadRecords: func(ctx context.Context, g executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error) {
		return canonicalRecords(ctx, s.stateRoot, g)
	}, LoadContract: func(ctx context.Context, r stageexecution.SupervisorRequest) (stageexecution.Contract, error) {
		if r.ProjectID != t.request.ProjectID {
			return stageexecution.Contract{}, stageexecution.ErrAdmission
		}
		preview, err := retainedPreview(ctx, stores.graphs, r)
		if err != nil || preview.Plan.Execution != t.ref || preview.Plan.StageID != string(t.state.Stage) {
			return stageexecution.Contract{}, stageexecution.ErrAdmission
		}
		state, err := stores.executions.Load(ctx, t.state.ProjectID, t.state.RepositoryKey, t.state.WorkItem)
		if err != nil || state.ExecutionID != t.ref.ExecutionID || state.Revision != t.ref.Revision {
			return stageexecution.Contract{}, stageexecution.ErrStalePreview
		}
		return s.stageContract(ctx, stores, state)
	}}
}

func (s lifecycleService) stageRun(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	previewService, err := s.dispatchPreview(ctx, stores, t, in)
	if err != nil {
		return s.stageError(err, nil)
	}
	preview, err := previewService.Preview(ctx, t.ref, in.Stage)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Stage dispatch preview", "Review child rights; repeat with exact digest, revision and both authority flags", preview)
	}
	store := s.supervisorStore(stores, t)
	single := &localSingleStore{s: s, stores: stores, target: t}
	service := stageexecution.RunService{ResumeStore: store, SingleStore: single, Preview: previewService, Publisher: &retainedPublisher{LocalPublisher: stagepublication.LocalPublisher{Graphs: stores.graphs, Executions: stores.executions, Singles: stores.singles, Leases: stores.leases, Execution: t.state, AllocateID: allocateStageID}}, Workspaces: stageWorkspaces{t.repository.Path}, Starter: stageexecution.CommittedStarter{Store: store, Single: single, Detached: stageexecution.SameBinaryDetacher{}}}
	r := stageexecution.RunRequest{Execution: t.ref, StageID: in.Stage, PreviewDigest: in.PreviewDigest, ExpectedRevision: in.ExpectedRevision, AuthorizeLocal: in.AuthorizeLocal, AuthorizeExternal: in.AuthorizeExternal, Actor: stageActor()}
	if t.single != nil {
		request := singleRequest(*t.single)
		r.CommittedSingle = &request
	}
	if doc, err := stageGraph(ctx, stores, t.request); err == nil && doc != nil && doc.Successor != nil {
		committed, err := committedRequest(ctx, s.stateRoot, stores.graphs, *doc.Successor)
		if err != nil {
			return s.stageError(err, nil)
		}
		r.Committed = &committed
	}
	result, err := service.Apply(ctx, r)
	if err != nil {
		return s.stageDispatchError(err, result.Status, result)
	}
	reference := "dispatch-authority:" + result.Publication.Authority.PreviewDigest
	if result.Publication.Single != nil {
		reference = "single-stage-authority:" + singleRequest(result.Publication.Single.Run).AuthorityRef
	}
	return s.stageSuccess("stage_dispatched", "Committed stage authority and started the supervisor", []string{reference}, "Inspect stage status; supervisor start does not imply child success", result)
}

func committedRequest(ctx context.Context, stateRoot string, graphs local.GraphStore, g executiongraph.SuccessorGraph) (stageexecution.SupervisorRequest, error) {
	// The authority digest is immutable and protected; its preview is loaded
	// from the graph's receipt, never taken from a new command's flags.
	project := g.Children[0].Envelope.Scope.ProjectID
	root, err := os.OpenRoot(stateRoot)
	if err != nil {
		return stageexecution.SupervisorRequest{}, err
	}
	defer root.Close()
	wire, err := root.ReadFile(filepath.Join("graphs", "v1", project, g.DispatchAuthorityRef+".json"))
	if err != nil {
		return stageexecution.SupervisorRequest{}, err
	}
	a, err := executiongraph.DecodeDispatchAuthority(wire)
	if err != nil {
		return stageexecution.SupervisorRequest{}, err
	}
	a, err = graphs.LoadDispatchAuthority(ctx, project, g.Parent.ExecutionID, a.PreviewDigest)
	return stageexecution.SupervisorRequest{ProjectID: a.ProjectID, ParentID: a.ParentID, PreviewDigest: a.PreviewDigest, AuthorityRef: g.DispatchAuthorityRef}, err
}

type stageRetryStore struct{ stageState }

func (st stageRetryStore) CommitRetry(ctx context.Context, g executiongraph.SuccessorGraph, a executiongraph.RetryAuthority) (string, error) {
	return st.stores.graphs.CommitRetry(ctx, g, a)
}

type stageDisposer struct{ repository string }

func (d stageDisposer) Dispose(ctx context.Context, g executiongraph.SuccessorGraph, c executiongraph.SuccessorChild, disposition string) error {
	if disposition != executiongraph.DispositionPreserve && disposition != executiongraph.DispositionReset {
		return stageexecution.ErrDispositionRequired
	}
	m, child, err := d.child(ctx, g, c.ExecutionID)
	if err != nil {
		return err
	}
	return m.InspectRetryWorkspace(ctx, child)
}

func (d stageDisposer) child(ctx context.Context, g executiongraph.SuccessorGraph, id string) (*gitworkspace.Manager, executiongraph.ChildExecution, error) {
	m, projection, err := (gitIntegration{d.repository}).manager(ctx, g)
	if err != nil {
		return nil, executiongraph.ChildExecution{}, err
	}
	for _, child := range projection.Children {
		if child.ExecutionID == id {
			return m, child, nil
		}
	}
	return nil, executiongraph.ChildExecution{}, gitworkspace.ErrInvalidConfiguration
}

func (d stageDisposer) ResetChild(ctx context.Context, g executiongraph.SuccessorGraph, id string, guard func(context.Context) error) error {
	m, child, err := d.child(ctx, g, id)
	if err != nil {
		return err
	}
	return m.ResetWorkspace(ctx, child, guard)
}
func (s lifecycleService) stageRetry(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	dispatch, err := s.dispatchPreview(ctx, stores, t, in)
	if err != nil {
		return s.stageError(err, nil)
	}
	store := s.supervisorStore(stores, t)
	service := stageexecution.RetryService{Store: stageRetryStore{stageState{s: s, stores: stores, target: t}}, Dispatch: dispatch, Disposer: stageDisposer{t.repository.Path}, Starter: stageexecution.CommittedStarter{Store: store, Detached: stageexecution.SameBinaryDetacher{}}, Authority: func(ctx context.Context, r stageexecution.StatusRequest) (stageexecution.SupervisorRequest, error) {
		g, err := stores.graphs.LoadSuccessor(ctx, r.ProjectID, r.ParentID)
		if err != nil {
			return stageexecution.SupervisorRequest{}, err
		}
		return committedRequest(ctx, s.stateRoot, stores.graphs, g)
	}}
	preview, err := service.Preview(ctx, stageexecution.RetryRequest{Target: t.request, Execution: t.ref, StageID: in.Stage, ChildID: in.Child})
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Stage retry preview", "Review attempt budget and workspace disposition; repeat with both authority flags", preview)
	}
	preview.Digest = in.PreviewDigest
	result, err := service.Apply(ctx, stageexecution.RetryApply{Preview: preview, AuthorizeLocal: in.AuthorizeLocal, AuthorizeExternal: in.AuthorizeExternal, Actor: stageActor(), Disposition: in.Disposition})
	if err != nil {
		return s.stageDispatchError(err, result.Status, result)
	}
	return s.stageSuccess("stage_retry_dispatched", "Committed retry authority and started its supervisor", []string{"retry-authority:" + result.AuthorityRef}, "Inspect stage status; retry start does not imply success", result)
}
func (s lifecycleService) stageDispatchError(err error, status string, payload any) cli.WorkflowStageResponse {
	if status == "partial" {
		return s.stageFailure(completion.Partial, "confirmed_effect_reconciliation_required", "Stage publication committed incompletely", "Inspect persisted authority before resuming", payload)
	}
	if status == "recovery_required" {
		return s.stageFailure(completion.Failure, "recovery_required", "Stage authority committed; dispatch needs recovery", "Inspect stage status before any new dispatch", payload)
	}
	return s.stageError(err, payload)
}

// retainedPreview verifies the digest from canonical preview bytes before the
// detached process consumes task content. Receipt validation follows separately.
func retainedPreview(ctx context.Context, graphs local.GraphStore, r stageexecution.SupervisorRequest) (stageexecution.DispatchPreview, error) {
	d, err := graphs.LoadDispatchInputs(ctx, r.ProjectID, r.ParentID, r.PreviewDigest)
	if err != nil {
		return stageexecution.DispatchPreview{}, err
	}
	return verifiedPreview(d.Preview, r.PreviewDigest)
}

func verifiedPreview(wire []byte, expected string) (stageexecution.DispatchPreview, error) {
	var p stageexecution.DispatchPreview
	if workflowdefinition.StrictJSONBounded(wire, &p, executiongraph.MaxDispatchInputsBytes) != nil || p.Digest != expected {
		return p, stageexecution.ErrSupervisorAuthority
	}
	digest := p.Digest
	p.Digest = ""
	wire, err := json.Marshal(p)
	p.Digest = digest
	sum := sha256.Sum256(wire)
	if err != nil || hex.EncodeToString(sum[:]) != digest {
		return p, stageexecution.ErrSupervisorAuthority
	}
	return p, nil
}

type stagePolicyCapabilities struct {
	policy    stagePolicy
	projectID string
}

func (p stagePolicyCapabilities) ValidateCapability(ctx context.Context, r executiongraph.CapabilityRequest) error {
	var err error
	for _, runtime := range []string{"codex", "claude"} {
		_, err = p.policy.Preview(ctx, p.projectID, runtimeapplication.Request{Role: r.Role, Complexity: r.Complexity, Capabilities: r.Capabilities, RuntimeID: runtime})
		if err == nil {
			return nil
		}
	}
	return err
}
