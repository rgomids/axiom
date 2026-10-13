package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

var _ cli.WorkflowStageService = lifecycleService{}

// stageStores are the protected machine-local stores used by #276 stage
// operations. They are opened per request; nothing is cached across calls.
type stageStores struct {
	graphs     local.GraphStore
	leases     local.RunLeaseStore
	executions local.WorkflowStore
	policies   local.ValidatorPolicyStore
	profiles   local.RuntimeProfileStore
}

func (s lifecycleService) stageStores() (stageStores, error) {
	var out stageStores
	var err error
	if out.graphs, err = local.NewGraphStore(s.stateRoot); err != nil {
		return out, err
	}
	if out.leases, err = local.NewRunLeaseStore(s.stateRoot); err != nil {
		return out, err
	}
	if out.executions, err = local.NewWorkflowStore(s.stateRoot); err != nil {
		return out, err
	}
	if out.policies, err = local.NewValidatorPolicyStore(s.stateRoot); err != nil {
		return out, err
	}
	out.profiles, err = local.NewRuntimeProfileStore(s.stateRoot)
	return out, err
}

// stageTarget is one exact Execution resolved through the authoritative
// workflow resolver; stage commands never infer selectors.
type stageTarget struct {
	state      workflow.State
	request    stageexecution.StatusRequest
	ref        workflowcompiler.ExecutionRef
	repository workflow.Repository
}

func (s lifecycleService) WorkflowStage(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	if ctx.Err() != nil {
		return s.stageFailure(completion.Interrupted, "workflow_cancelled", "Stage operation was cancelled", "Inspect stage status before repeating any mutation", nil)
	}
	stores, err := s.stageStores()
	if err != nil {
		return s.stageFailure(completion.ValidationFailure, "invalid_state_root", "Stage state root is unsafe", "Use an absolute, private LINGO_STATE_ROOT", nil)
	}
	if in.Operation == "coordination-publish" {
		return s.stagePublish(ctx, in)
	}
	operation := projectapp.AdmitExecutionAdvance
	if in.Operation == "status" || strings.HasPrefix(in.Operation, "coordination-") {
		operation = projectapp.AdmitExecutionStatus
	}
	target, failure := s.resolveStageTarget(ctx, in, operation)
	if failure != nil {
		return *failure
	}
	switch in.Operation {
	case "status":
		return s.stageStatus(ctx, stores, target)
	case "coordination-list", "coordination-show":
		return s.stageCoordinationRead(ctx, stores, target, in)
	case "cancel":
		return s.stageCancel(ctx, stores, target, in)
	case "reconcile":
		return s.stageReconcile(ctx, stores, target, in)
	case "integrate":
		return s.stageIntegrate(ctx, stores, target, in)
	case "run":
		return s.stageRun(ctx, stores, target, in)
	case "retry":
		return s.stageRetry(ctx, stores, target, in)
	}
	return s.stageFailure(completion.ValidationFailure, "invalid_command", "Unsupported stage operation", "Run axiom workflow stage --help", nil)
}

func (s lifecycleService) resolveStageTarget(ctx context.Context, in cli.WorkflowStageInput, operation projectapp.AdmissionOperation) (stageTarget, *cli.WorkflowStageResponse) {
	if in.Project == "" {
		id, failure := s.EffectiveProject(ctx, "")
		if id == "" {
			return stageTarget{}, &cli.WorkflowStageResponse{Result: failure}
		}
		in.Project = id
	}
	if blocked := s.gate(ctx, in.Project, operation); blocked != nil {
		return stageTarget{}, &cli.WorkflowStageResponse{Result: *blocked}
	}
	provider, resource, externalID, ok := cli.ParseWorkItemSelector(in.WorkItem)
	if !ok || in.Execution == "" {
		response := s.stageFailure(completion.ValidationFailure, "invalid_execution_input", "Stage input is invalid", "Provide exact selectors returned by Lingo", nil)
		return stageTarget{}, &response
	}
	input := cli.WorkflowInput{Project: in.Project, Repository: in.Repository, Provider: provider, ProviderRepository: resource, ExternalID: externalID}
	resolved, loaded := s.workflows.ValidateTarget(ctx, workflowTarget(input))
	if loaded.Category != "" && loaded.Status != workflow.Succeeded {
		response := s.stageFailure(loaded.Status, loaded.Category, "Stage target could not be resolved", "Inspect the Project, Repository and Work Item selectors", nil)
		return stageTarget{}, &response
	}
	input.Execution = in.Execution
	status := s.workflows.Status(ctx, workflowTarget(input))
	if status.Status != workflow.Succeeded {
		response := s.stageFailure(status.Status, status.Category, "Execution could not be loaded", "Inspect workflow status with the exact selectors", nil)
		return stageTarget{}, &response
	}
	state := status.State
	t := stageTarget{state: state, request: stageexecution.StatusRequest{ProjectID: state.ProjectID, ParentID: state.ExecutionID}, ref: workflowcompiler.ExecutionRef{ExecutionID: state.ExecutionID, Revision: state.Revision}, repository: resolved.Repository}
	if graphs, err := local.NewGraphStore(s.stateRoot); err == nil {
		parent, err := graphs.FindStageGraph(ctx, state.ProjectID, state.ExecutionID, string(state.Stage))
		if err == nil {
			t.request.ParentID = parent
		} else if !errors.Is(err, local.ErrNotFound) && !errors.Is(err, os.ErrNotExist) {
			response := s.stageError(err, nil)
			return stageTarget{}, &response
		}
	}
	if in.ExpectedRevision != 0 && in.ExpectedRevision != state.Revision {
		response := s.stageFailure(completion.DeniedAuthority, "stale_execution_revision", "Execution revision changed", "Inspect workflow status and request a fresh preview", nil)
		return stageTarget{}, &response
	}
	return t, nil
}

// stageContract assembles reader-validated snapshots for admission. It never
// migrates, resolves another Profile or treats declaration as proof.
func (s lifecycleService) stageContract(ctx context.Context, stores stageStores, state workflow.State) (stageexecution.Contract, error) {
	c := stageexecution.Contract{Execution: state, PolicyAllowed: true}
	if state.Binding != nil {
		doc, issues := workflowdefinition.Decode(state.Binding.Snapshot)
		if len(issues) != 0 {
			return c, stageexecution.ErrAdmission
		}
		c.Definition = doc
		// The Workflow was explicitly selected when this Execution was bound.
		c.WorkflowSelected = state.Binding.SchemaVersion == 2
	}
	cfg, err := stores.profiles.Load(ctx)
	if err != nil {
		return c, stageexecution.ErrProfileEffortUnconfigured
	}
	c.Configuration = cfg
	if state.Binding != nil && state.Binding.ResolvedProfile != nil {
		frozen := state.Binding.ResolvedProfile
		// A removed, disallowed or disabled frozen Profile blocks; nothing
		// falls back to another Profile, model, Runtime or effort.
		for _, runtime := range cfg.Runtimes {
			if runtime.ID != frozen.RuntimeID || !runtime.Enabled || !slices.Contains(runtime.AllowlistedProfileIDs, frozen.ProfileID) {
				continue
			}
			for _, profile := range cfg.ModelProfiles {
				if profile.ID == frozen.ProfileID && profile.RuntimeID == frozen.RuntimeID {
					c.ProfileAvailable = true
				}
			}
		}
	}
	return c, nil
}

// stageGraph loads the stage document of an Execution. Historical Graph1/2
// are returned for read-only inspection only; mutations require successor.
func stageGraph(ctx context.Context, stores stageStores, r stageexecution.StatusRequest) (*executiongraph.GraphDocument, error) {
	g, err := stores.graphs.LoadSuccessor(ctx, r.ProjectID, r.ParentID)
	if err == nil {
		return &executiongraph.GraphDocument{Successor: &g}, nil
	}
	historical, herr := stores.graphs.Load(ctx, r.ProjectID, r.ParentID)
	if herr == nil {
		return &executiongraph.GraphDocument{Historical: &historical}, nil
	}
	if errors.Is(err, local.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, err
}

// stageState is the protected StatusSource/CancelStore/ReconcileStore over
// local stores, bound to one resolved target.
type stageState struct {
	s        lifecycleService
	stores   stageStores
	target   stageTarget
	observer local.ReconciliationObserver
}

func (st stageState) StageSnapshot(ctx context.Context, r stageexecution.StatusRequest) (stageexecution.Contract, error) {
	if r != st.target.request {
		return stageexecution.Contract{}, stageexecution.ErrAdmission
	}
	c, err := st.s.stageContract(ctx, st.stores, st.target.state)
	if err != nil {
		return c, err
	}
	c.Graph, err = stageGraph(ctx, st.stores, r)
	if err == nil && c.Graph == nil {
		err = errStageGraphMissing
	}
	return c, err
}
func (st stageState) RequestCancel(ctx context.Context, g executiongraph.SuccessorGraph, r executiongraph.CancelRequest) (executiongraph.SuccessorGraph, error) {
	return st.stores.graphs.RequestCancel(ctx, g, r)
}
func (st stageState) AppendReconciliation(ctx context.Context, g executiongraph.SuccessorGraph, f executiongraph.ReconciliationFact) (executiongraph.SuccessorGraph, error) {
	if st.observer == nil {
		return g, stageexecution.ErrPreview
	}
	return st.stores.graphs.AppendReconciliation(ctx, g, f, st.observer)
}

var errStageGraphMissing = errors.New("stage_not_dispatched")

// canonicalRecords projects the accepted canonical streams of every child.
func canonicalRecords(ctx context.Context, stateRoot string, g executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error) {
	store, err := local.NewCoordinationStore(stateRoot, g.Children[0].Envelope.Scope.ProjectID)
	if err != nil {
		return nil, err
	}
	reader := coordination.Reader{Store: store}
	records := []executiongraph.AttemptRecord{}
	for _, c := range g.Children {
		list, err := reader.List(ctx, coordination.Filter{ParentID: g.Parent.ExecutionID, ChildID: c.ExecutionID})
		if err != nil {
			return nil, err
		}
		for _, record := range list {
			records = append(records, coordination.ClassifierRecord(record))
		}
	}
	return records, nil
}

// --- result mapping -------------------------------------------------------

func (s lifecycleService) stageFailure(status completion.Status, category, message, next string, payload any) cli.WorkflowStageResponse {
	if category == "" {
		category = string(status)
	}
	result := canonicalCompletion(factsForCompletionStatus(status), message+": "+category, nil, next, s.provenance)
	result.Category = category
	return cli.WorkflowStageResponse{Result: result, Payload: payload}
}

func (s lifecycleService) stageSuccess(category, message string, references []string, next string, payload any) cli.WorkflowStageResponse {
	result := canonicalCompletion(completion.Facts{Completed: true}, message, references, next, s.provenance)
	result.Category = category
	return cli.WorkflowStageResponse{Result: result, Payload: payload}
}

// stageError maps application errors to canonical status and category. It
// never upgrades an uncertain outcome.
func (s lifecycleService) stageError(err error, payload any) cli.WorkflowStageResponse {
	type rule struct {
		target   error
		status   completion.Status
		category string
		next     string
	}
	rules := []rule{
		{stageexecution.ErrDispatchAuthority, completion.DeniedAuthority, "authority_required", "Review the preview, then repeat with the exact --preview-digest and the operation's authority flags"},
		{stageexecution.ErrStalePreview, completion.DeniedAuthority, "stale_preview", "Request a fresh preview; never reuse a digest after a change"},
		{executiongraph.ErrUnsupportedGraphFormat, completion.Failure, "unsupported_graph_format", "Historical graphs are read-only to stage operations"},
		{stageexecution.ErrLegacyExecution, completion.ValidationFailure, "legacy_execution_contract", "Start a new Execution under a migrated schema-2 Workflow"},
		{workflowcompiler.ErrWorkflowMigrationRequired, completion.ValidationFailure, "workflow_migration_required", "Migrate and select a schema-2 Workflow (#306)"},
		{stageexecution.ErrProfileEffortUnconfigured, completion.ValidationFailure, "profile_effort_unconfigured", "Migrate local configuration to Profile-owned effort (#306)"},
		{stageexecution.ErrPolicyUpdateRequired, completion.ValidationFailure, "project_policy_update_required", "Update the Project Runtime policy"},
		{capabilityproof.ErrBlocked, completion.ValidationFailure, "capability_unknown", "Run an authorized capability probe: axiom runtime <codex|claude> capability probe"},
		{stageexecution.ErrReconcileLeaseHeld, completion.ValidationFailure, "reconcile_lease_held", "Wait for the supervisor to finish or cancel the stage"},
		{stageexecution.ErrReconcileOrphanAlive, completion.ValidationFailure, "stop_orphan_then_reconcile", "Stop the orphaned process group, then reconcile"},
		{stageexecution.ErrReconcileIneligible, completion.ValidationFailure, "reconcile_not_required", "Only stale or unknown latest attempts are reconciled"},
		{stageexecution.ErrRetryIneligible, completion.ValidationFailure, "retry_not_eligible", "Reconcile unknown attempts first; skipped and succeeded children are never retried"},
		{stageexecution.ErrRetryExhausted, completion.ValidationFailure, "retry_attempts_exhausted", "The child's maximum attempts are used"},
		{stageexecution.ErrDispositionRequired, completion.ValidationFailure, "workspace_disposition_required", "Choose --disposition preserve or reset explicitly"},
		{stageexecution.ErrIntegrationRecorded, completion.ValidationFailure, "integration_already_recorded", "Inspect stage status; nothing remains to integrate"},
		{stageexecution.ErrIntegrationReconcile, completion.Failure, "recovery_required", "Reconcile the integration attempt; its effect is never repeated blindly"},
		{stageexecution.ErrValidatorPolicy, completion.ValidationFailure, "validator_policy_unavailable", "Reference a reviewed validator policy of this Project by digest"},
		{stageexecution.ErrProjectionPartial, completion.Partial, "confirmed_effect_reconciliation_required", "Repair the stage outcome projection with stage integrate"},
		{executiongraph.ErrIntegrationFailed, completion.Failure, "integration_failed", "Inspect validator results; a new integration needs its own authority"},
		{executiongraph.ErrIntegrationBlocked, completion.ValidationFailure, "integration_blocked", "Every required child must be succeeded/result_recorded without conflicts"},
		{stageexecution.ErrSupervisorRecovery, completion.Failure, "recovery_required", "Inspect stage status and reconcile before any new dispatch"},
		{stageexecution.ErrSupervisorAuthority, completion.DeniedAuthority, "supervisor_authority_required", "Only committed, exact-bound authority can be resumed"},
		{stageexecution.ErrDetachmentUnsupported, completion.Failure, "supervisor_detachment_unsupported", "This platform cannot detach the supervisor; nothing was started"},
		{coordination.ErrChildContext, completion.DeniedAuthority, "child_context_required", "Publish only from a running stage child's own task context"},
		{coordination.ErrRecordNotFound, completion.ValidationFailure, "record_not_found", "List the child's canonical records first"},
		{coordination.ErrInvalidWait, completion.ValidationFailure, "invalid_wait", "Use a positive --wait no longer than 30s"},
		{errStageGraphMissing, completion.ValidationFailure, "stage_not_dispatched", "Run stage run for this Execution first"},
		{local.ErrConflict, completion.RetryableFailure, "temporarily_unavailable", "A concurrent stage operation holds the lease; retry the read first"},
		{local.ErrRecoveryRequired, completion.Failure, "recovery_required", "Run recovery inspect and apply the reviewed plan"},
		{executiongraph.ErrInvalidGraph, completion.Failure, "recovery_required", "Inspect stage status and recovery state"},
	}
	for _, r := range rules {
		if errors.Is(err, r.target) {
			return s.stageFailure(r.status, r.category, "Stage operation refused", r.next, payload)
		}
	}
	return s.stageFailure(completion.Failure, "stage_operation_failed", "Stage operation failed", "Inspect stage status; never repeat an authorized mutation blindly", payload)
}

// --- status ---------------------------------------------------------------

type stageStatusPayload struct {
	Graph    executiongraph.GraphDocument            `json:"graph"`
	Blockers map[string]string                       `json:"blockers"`
	Health   map[string]executiongraph.AttemptHealth `json:"health"`
	Next     map[string]string                       `json:"next"`
	Outcomes map[string]executiongraph.StageOutcome  `json:"outcomes,omitempty"`
}

type localStatusObserver struct {
	leases local.RunLeaseStore
}

func (o localStatusObserver) LeaseHeld(ctx context.Context, r stageexecution.StatusRequest) (bool, error) {
	return o.leases.Probe(ctx, r.ProjectID, r.ParentID)
}
func (o localStatusObserver) Process(_ context.Context, p executiongraph.ProcessCorrelation) (*executiongraph.ProcessCorrelation, error) {
	return executiongraph.ObserveRecordedProcess(p)
}

func (s lifecycleService) stageStatus(ctx context.Context, stores stageStores, t stageTarget) cli.WorkflowStageResponse {
	state := stageState{s: s, stores: stores, target: t}
	result, err := stageexecution.StatusService{Source: state, Observer: localStatusObserver{leases: stores.leases}}.Read(ctx, t.request)
	if err != nil {
		return s.stageError(err, nil)
	}
	payload := stageStatusPayload{Graph: result.Graph, Blockers: result.Blockers, Health: result.Health, Next: result.Next}
	if outcomes, err := stores.executions.LoadStageOutcomes(ctx, t.request.ProjectID, t.request.ParentID); err == nil && len(outcomes) > 0 {
		payload.Outcomes = outcomes
	}
	return s.stageSuccess("stage_status", "Persisted stage state", []string{"execution:" + t.state.ExecutionID}, "Status renders persisted fields only; it never claims a stop or success", payload)
}

// --- coordination read ----------------------------------------------------

func (s lifecycleService) stageCoordinationRead(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	store, err := local.NewCoordinationStore(s.stateRoot, t.request.ProjectID)
	if err != nil {
		return s.stageError(err, nil)
	}
	reader := coordination.Reader{Store: store}
	children := []string{in.Child}
	if in.Child == "" {
		doc, err := stageGraph(ctx, stores, t.request)
		if err != nil || doc == nil || doc.Successor == nil {
			return s.stageError(errors.Join(errStageGraphMissing, err), nil)
		}
		children = nil
		for _, c := range doc.Successor.Children {
			children = append(children, c.ExecutionID)
		}
	}
	if in.Operation == "coordination-show" {
		record, err := reader.Show(ctx, coordination.Filter{ParentID: t.request.ParentID, ChildID: in.Child}, in.Record)
		if err != nil {
			return s.stageError(err, nil)
		}
		return s.stageSuccess("coordination_record", "Canonical coordination record", []string{"coordination-record:" + record.Digest}, "Records grant no authority", map[string]any{"record": record})
	}
	records := []coordination.Record{}
	for _, child := range children {
		filter := coordination.Filter{ParentID: t.request.ParentID, ChildID: child, AttemptID: in.Attempt, Kind: coordination.Kind(in.Kind)}
		var list []coordination.Record
		if in.Wait > 0 && in.Child != "" {
			waitCtx, cancel := context.WithTimeout(ctx, in.Wait+time.Second)
			list, err = reader.Wait(waitCtx, filter, in.After, in.Wait)
			cancel()
			if errors.Is(err, context.DeadlineExceeded) {
				list, err = []coordination.Record{}, nil
			}
		} else {
			list, err = reader.List(ctx, filter)
		}
		if err != nil {
			return s.stageError(err, nil)
		}
		for _, r := range list {
			if r.Revision > in.After {
				records = append(records, r)
			}
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].ChildID != records[j].ChildID {
			return records[i].ChildID < records[j].ChildID
		}
		return records[i].Revision < records[j].Revision
	})
	return s.stageSuccess("coordination_records", fmt.Sprintf("%d canonical coordination records", len(records)), nil, "Records are read-only and grant no authority", map[string]any{"records": records})
}

// --- cancel ---------------------------------------------------------------

func allocateStageID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func stageActor() string {
	if user := os.Getenv("USER"); user != "" && !strings.ContainsAny(user, "\x00\r\n") && len(user) <= 128 {
		return user
	}
	return "local-operator"
}

func (s lifecycleService) stageCancel(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	service := stageexecution.CancelService{Store: stageState{s: s, stores: stores, target: t}, AllocateID: allocateStageID}
	preview, err := service.Preview(ctx, t.request)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Cancellation preview", "Repeat with --preview-digest and --authorize-local to publish the request", preview)
	}
	reviewed := preview
	reviewed.Digest = in.PreviewDigest
	result, err := service.Apply(ctx, stageexecution.CancelApply{Preview: reviewed, AuthorizeLocal: in.AuthorizeLocal, Actor: stageActor()})
	if err != nil {
		return s.stageError(err, result)
	}
	return s.stageSuccess(result.Category, "Cancellation requested (not an observed stop)", []string{"cancel-request:" + result.Request.RequestID}, "Inspect stage status; an attempt is cancelled only after a confirmed stop", result)
}

func (s lifecycleService) stagePreview(digest, message, next string, payload any) cli.WorkflowStageResponse {
	return s.stageSuccess("stage_preview_ready", message, []string{"stage-preview:" + digest}, next, map[string]any{"previewDigest": digest, "preview": payload})
}

// --- reconcile ------------------------------------------------------------

// localReconcileObserver gathers bounded digest-only observations. It never
// classifies records as success and never signals a process.
type localReconcileObserver struct {
	stateRoot string
	leases    local.RunLeaseStore
}

func digestText(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprintf(h, "%d:%s;", len(p), p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (o localReconcileObserver) ObserveReconciliation(ctx context.Context, g executiongraph.SuccessorGraph, childID, attemptID string) (executiongraph.ReconciliationObservations, error) {
	var workspace string
	for _, c := range g.Children {
		if c.ExecutionID == childID {
			workspace = c.Envelope.Workspace
		}
	}
	return executiongraph.GatherReconciliationObservations(g, childID, attemptID, executiongraph.ReconciliationSources{
		LeaseProbe: func() (bool, error) {
			return o.leases.Probe(ctx, g.Children[0].Envelope.Scope.ProjectID, g.Parent.ExecutionID)
		},
		ProcessProbe: executiongraph.ObserveRecordedProcess,
		Worktree:     func() (string, error) { return worktreeDigest(ctx, workspace) },
		CanonicalRecords: func() (string, error) {
			store, err := local.NewCoordinationStore(o.stateRoot, g.Children[0].Envelope.Scope.ProjectID)
			if err != nil {
				return "", err
			}
			stream, found, err := store.ReadStream(ctx, g.Parent.ExecutionID, childID)
			if err != nil {
				return "", err
			}
			if !found {
				return digestText("absent"), nil
			}
			wire, err := json.Marshal(stream)
			return digestText(string(wire)), err
		},
		QuarantinedInbox: func() (string, error) { return inboxDigest(attemptInbox(workspace, attemptID)), nil },
	})
}

// worktreeDigest binds HEAD and porcelain status; an absent worktree is a
// stable observation, never an error that could hide a change.
func worktreeDigest(ctx context.Context, workspace string) (string, error) {
	if workspace == "" {
		return digestText("no-workspace"), nil
	}
	if _, err := os.Lstat(workspace); errors.Is(err, os.ErrNotExist) {
		return digestText("absent"), nil
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}
	run := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, git, append([]string{"-C", workspace}, args...)...)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0"}
		out, err := cmd.Output()
		return string(out), err
	}
	head, err := run("rev-parse", "HEAD")
	if err != nil {
		return digestText("unreadable"), nil
	}
	status, err := run("status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return digestText("unreadable", head), nil
	}
	return digestText(head, status), nil
}

// inboxDigest is diagnostics only: names and sizes, never content semantics.
func inboxDigest(path string) string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return digestText("absent")
	}
	parts := []string{}
	for _, e := range entries {
		info, err := e.Info()
		if err == nil {
			parts = append(parts, fmt.Sprintf("%s:%d", e.Name(), info.Size()))
		}
	}
	sort.Strings(parts)
	return digestText(parts...)
}

func (s lifecycleService) stageReconcile(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	observer := localReconcileObserver{stateRoot: s.stateRoot, leases: stores.leases}
	service := stageexecution.ReconcileService{Store: stageState{s: s, stores: stores, target: t, observer: observer}, Observer: observer, AllocateID: allocateStageID}
	request := stageexecution.ReconcileRequest{Target: t.request, ChildID: in.Child, AttemptID: in.Attempt}
	preview, err := service.Preview(ctx, request)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Reconciliation preview", "Repeat with --preview-digest and --authorize-local to append the fact; status never changes", preview)
	}
	reviewed := preview
	reviewed.Digest = in.PreviewDigest
	result, err := service.Apply(ctx, stageexecution.ReconcileApply{Preview: reviewed, AuthorizeLocal: in.AuthorizeLocal, Actor: stageActor()})
	if err != nil {
		return s.stageError(err, result)
	}
	return s.stageSuccess("reconciliation_recorded", "Reconciliation fact appended (status unchanged)", []string{"reconciliation-fact:" + result.Fact.FactID}, "A retry still needs its own authorized stage retry", result)
}

// --- integrate ------------------------------------------------------------

// localIntegrateStore composes the protected graph, lease and receipt stores.
type localIntegrateStore struct {
	stageState
	supervisor LocalSupervisorStore
}

func (st localIntegrateStore) Records(ctx context.Context, g executiongraph.SuccessorGraph) ([]executiongraph.AttemptRecord, error) {
	return canonicalRecords(ctx, st.s.stateRoot, g)
}
func (st localIntegrateStore) Acquire(ctx context.Context, g executiongraph.SuccessorGraph) (stageexecution.SupervisorLease, error) {
	return st.supervisor.Acquire(ctx, g)
}
func (st localIntegrateStore) Save(ctx context.Context, g executiongraph.SuccessorGraph, l stageexecution.SupervisorLease) (executiongraph.SuccessorGraph, error) {
	return st.supervisor.Save(ctx, g, l)
}
func (st localIntegrateStore) CommitIntegrationReceipt(ctx context.Context, g executiongraph.SuccessorGraph, a executiongraph.IntegrationReceipt) (string, error) {
	return st.stores.graphs.CommitIntegrationReceipt(ctx, g, a)
}
func (st localIntegrateStore) LoadIntegrationReceipt(ctx context.Context, r stageexecution.StatusRequest, ref string) (executiongraph.IntegrationReceipt, error) {
	return st.stores.graphs.LoadIntegrationReceipt(ctx, r.ProjectID, r.ParentID, ref)
}

// gitIntegration projects the successor graph onto the existing digest-bound
// Git workspace manager. The repository HEAD must still equal the dispatch
// base: any foreign change refuses integration.
type gitIntegration struct{ repository string }

func stageWorkspaceRoot(g executiongraph.SuccessorGraph) (string, error) {
	root := ""
	for _, c := range g.Children {
		r := filepath.Dir(filepath.Dir(c.Envelope.Workspace))
		if root == "" {
			root = r
		}
		if r != root || !filepath.IsAbs(r) {
			return "", gitworkspace.ErrInvalidConfiguration
		}
	}
	return root, nil
}

func (gi gitIntegration) manager(ctx context.Context, g executiongraph.SuccessorGraph) (*gitworkspace.Manager, executiongraph.Graph, error) {
	projection, err := executiongraph.WorkspaceProjection(g)
	if err != nil {
		return nil, projection, err
	}
	root, err := stageWorkspaceRoot(g)
	if err != nil {
		return nil, projection, err
	}
	head, err := gitHead(ctx, gi.repository)
	if err != nil {
		return nil, projection, err
	}
	m, err := gitworkspace.NewManager(ctx, gi.repository, root, head, projection)
	return m, projection, err
}

func gitHead(ctx context.Context, repository string) (string, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, git, "-C", repository, "rev-parse", "HEAD")
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0"}
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func (gi gitIntegration) Observe(ctx context.Context, g executiongraph.SuccessorGraph, waivers map[string]string) (executiongraph.IntegrationObservation, []executiongraph.ChildResult, error) {
	m, _, err := gi.manager(ctx, g)
	if err != nil {
		return executiongraph.IntegrationObservation{}, nil, err
	}
	observation, results, _, err := m.ObserveIntegration(ctx, waivers)
	return observation, results, err
}
func integrationProjection(projection executiongraph.Graph) (executiongraph.ChildExecution, error) {
	for _, c := range projection.Children {
		if c.ExecutionID == projection.Parent.IntegrationChild {
			return c, nil
		}
	}
	return executiongraph.ChildExecution{}, executiongraph.ErrInvalidGraph
}
func (gi gitIntegration) Apply(ctx context.Context, g executiongraph.SuccessorGraph, preview executiongraph.IntegrationPreview) (executiongraph.AppliedIntegration, error) {
	m, projection, err := gi.manager(ctx, g)
	if err != nil {
		return executiongraph.AppliedIntegration{}, err
	}
	integration, err := integrationProjection(projection)
	if err != nil {
		return executiongraph.AppliedIntegration{}, err
	}
	return m.Apply(ctx, integration, preview)
}
func (gi gitIntegration) Validate(ctx context.Context, g executiongraph.SuccessorGraph, tree string, commands []gitworkspace.ValidationCommand) ([]executiongraph.ValidationResult, error) {
	m, projection, err := gi.manager(ctx, g)
	if err != nil {
		return nil, err
	}
	integration, err := integrationProjection(projection)
	if err != nil {
		return nil, err
	}
	validator, err := gitworkspace.NewCombinedValidator(m, commands)
	if err != nil {
		return nil, err
	}
	return validator.ValidateCombined(ctx, integration, tree)
}

func (s lifecycleService) stageIntegrate(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	state := stageState{s: s, stores: stores, target: t}
	service := stageexecution.IntegrateService{
		Store:      localIntegrateStore{stageState: state, supervisor: LocalSupervisorStore{Graphs: stores.graphs, Leases: stores.leases}},
		Workspace:  gitIntegration{repository: t.repository.Path},
		Validators: stageexecution.ValidatorPolicyService{Store: stores.policies},
		Ledger:     stores.executions,
		AllocateID: allocateStageID,
		Self:       stageexecution.CurrentProcess,
	}
	request := stageexecution.IntegrateRequest{Target: t.request, Execution: t.ref, StageID: in.Stage, ValidatorPolicyDigest: in.ValidatorPolicy, OptionalWaivers: in.Waivers}
	preview, err := service.Preview(ctx, request)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" {
		return s.stagePreview(preview.Digest, "Integration preview", "Repeat with --preview-digest and --authorize-local to apply the local integration", preview)
	}
	reviewed := preview
	reviewed.Digest = in.PreviewDigest
	result, err := service.Apply(ctx, stageexecution.IntegrateApply{Preview: reviewed, AuthorizeLocal: in.AuthorizeLocal, Actor: stageActor()})
	if err != nil {
		return s.stageError(err, result)
	}
	refs := []string{"integration-receipt:" + result.ReceiptRef}
	if result.OutcomeRef != "" {
		refs = append(refs, "stage-outcome:"+result.OutcomeRef)
	}
	return s.stageSuccess("stage_integrated", "Stage integrated locally; the Work Item is not accepted", refs, "Advance the Execution through its own lifecycle; no push, PR or merge happened", result)
}

// profileFrozen reports the frozen tuple of a bound Execution.
func profileFrozen(state workflow.State) *runtimeprofile.ResolvedProfile {
	if state.Binding == nil {
		return nil
	}
	return state.Binding.ResolvedProfile
}
