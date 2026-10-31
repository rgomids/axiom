package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowcompiler"
)

var _ cli.WorkflowStageService = lifecycleService{}

// stageStores are the protected machine-local stores used by #276 stage
// operations. They are opened per request; nothing is cached across calls.
type stageStores struct {
	graphs     local.GraphStore
	leases     local.RunLeaseStore
	executions local.WorkflowStore
	profiles   local.RuntimeProfileStore
	singles    local.SingleStageStore
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
	if out.singles, err = local.NewSingleStageStore(s.stateRoot); err != nil {
		return out, err
	}
	out.profiles, err = local.NewRuntimeProfileStore(s.stateRoot)
	return out, err
}

// stageTarget is one exact Execution resolved through the authoritative
// workflow resolver; stage commands never infer selectors.
type stageTarget struct {
	state      workflow.State
	request    stageScope
	ref        workflowcompiler.ExecutionRef
	repository workflow.Repository
	single     *executiongraph.SingleStageRun
}

func (s lifecycleService) WorkflowStage(ctx context.Context, in cli.WorkflowStageInput) (out cli.WorkflowStageResponse) {
	var observed *stageTarget
	defer func() {
		if out.Result.Diagnostic != nil {
			d := out.Result.Diagnostic
			switch in.Operation {
			case "execution-run":
				d.Operation = "workflow run"
			case "status", "cancel", "retry", "reconcile":
				d.Operation = "workflow stage " + in.Operation
			case "coordination-list", "coordination-show", "coordination-publish":
				d.Operation = "workflow coordination " + strings.TrimPrefix(in.Operation, "coordination-")
			default:
				// Delivery belongs to AXM-9; do not enrich its shared mapper
				// calls or invent an operation for an unsupported input.
				out.Result.Diagnostic = nil
				return
			}
			if observed != nil {
				bindStageDiagnostic(d, *observed)
			}
		}
	}()
	if ctx.Err() != nil {
		return s.stageFailure(completion.Interrupted, "workflow_cancelled", "Stage operation was cancelled", "Inspect stage status before repeating any mutation", nil)
	}
	switch in.Operation {
	case "execution-run", "status", "cancel", "retry", "reconcile":
		return s.executionStage(ctx, in)
	}
	stores, err := s.stageStores()
	if err != nil {
		return s.stageFailure(completion.ValidationFailure, "invalid_state_root", "Stage state root is unsafe", "Use an absolute, private LINGO_STATE_ROOT", nil)
	}
	if in.Operation == "coordination-publish" {
		return s.stagePublish(ctx, in)
	}
	operation := projectapp.AdmitExecutionAdvance
	if in.Operation == "status" || in.Operation == "delivery-packet" && !in.AuthorizeLocal || strings.HasPrefix(in.Operation, "coordination-") {
		operation = projectapp.AdmitExecutionStatus
	}
	target, failure := s.resolveStageTarget(ctx, in, operation)
	if failure != nil {
		return *failure
	}
	observed = &target
	if in.Operation == "delivery-decision" {
		return s.deliveryDecision(ctx, stores, target, in)
	}
	if in.Operation == "delivery-rework" {
		return s.deliveryRework(ctx, stores, target, in)
	}
	if strings.HasPrefix(in.Operation, "delivery-") && in.Operation != "delivery-packet" {
		return s.deliveryPublication(ctx, stores, target, in)
	}
	if in.Operation == "delivery-packet" {
		return s.deliveryPacket(ctx, stores, target, in)
	}
	switch in.Operation {
	case "coordination-list", "coordination-show":
		return s.stageCoordinationRead(ctx, stores, target, in)
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
	t := stageTarget{state: state, request: stageScope{ProjectID: state.ProjectID, ParentID: state.ExecutionID}, ref: workflowcompiler.ExecutionRef{ExecutionID: state.ExecutionID, Revision: state.Revision}, repository: resolved.Repository}
	if graphs, err := local.NewGraphStore(s.stateRoot); err == nil {
		parent, err := graphs.FindStageGraph(ctx, state.ProjectID, state.ExecutionID, string(state.Stage))
		if err == nil {
			t.request.ParentID = parent
		} else if !errors.Is(err, local.ErrNotFound) && !errors.Is(err, os.ErrNotExist) {
			response := s.stageError(err, nil)
			return stageTarget{}, &response
		}
	}
	if t.request.ParentID == state.ExecutionID {
		singles, err := local.NewSingleStageStore(s.stateRoot)
		if err == nil {
			t.single, err = loadStageSingle(ctx, singles, state)
		}
		if err != nil {
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

// stageScope addresses a pre-cutover stage document (Graph parent or the
// Execution itself) for the read-only delivery and coordination paths.
type stageScope struct{ ProjectID, ParentID string }

// stageGraph loads the stage document of an Execution. Historical Graph1/2
// are returned for read-only inspection only; no writer remains.
func stageGraph(ctx context.Context, stores stageStores, r stageScope) (*executiongraph.GraphDocument, error) {
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
	result := canonicalCompletion(factsForCompletionStatus(status), message+": "+category, stageConfirmedReferences(status, payload), next, s.provenance)
	result.Category = category
	result.Diagnostic = stageDiagnostic(category, status)
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
		{executiongraph.ErrUnsupportedGraphFormat, completion.Failure, "unsupported_graph_format", "Historical graphs are read-only"},
		{coordination.ErrChildContext, completion.DeniedAuthority, "child_context_required", "Publish only from a running stage child's own task context"},
		{coordination.ErrRecordNotFound, completion.ValidationFailure, "record_not_found", "List the child's canonical records first"},
		{coordination.ErrInvalidWait, completion.ValidationFailure, "invalid_wait", "Use a positive --wait no longer than 30s"},
		{errStageGraphMissing, completion.ValidationFailure, "stage_not_dispatched", "No pre-cutover stage record exists; new work uses axiom workflow run"},
		{executiongraph.ErrUnsupportedSingleStageFormat, completion.Failure, "unsupported_single_stage_format", "This release cannot read the single-stage ledger; nothing was changed"},
		{local.ErrConflict, completion.RetryableFailure, "temporarily_unavailable", "A concurrent operation holds the lease; retry the read first"},
		{local.ErrRecoveryRequired, completion.Failure, "recovery_required", "Run recovery inspect and apply the reviewed plan"},
		{executiongraph.ErrInvalidGraph, completion.Failure, "recovery_required", "Inspect recovery state"},
	}
	for _, r := range rules {
		if errors.Is(err, r.target) {
			return s.stageFailure(r.status, r.category, "Stage operation refused", r.next, payload)
		}
	}
	return s.stageFailure(completion.Failure, "stage_operation_failed", "Stage operation failed", "Inspect status; never repeat an authorized mutation blindly", payload)
}

// --- coordination read ----------------------------------------------------

func (s lifecycleService) stageCoordinationRead(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	store, err := local.NewCoordinationStore(s.stateRoot, t.request.ProjectID)
	if err != nil {
		return s.stageError(err, nil)
	}
	reader := coordination.Reader{Store: store}
	children := []string{in.Child}
	if in.Child == "" && t.single != nil {
		children = []string{t.state.ExecutionID}
	} else if in.Child == "" {
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

func (s lifecycleService) stagePreview(digest, message, next string, payload any) cli.WorkflowStageResponse {
	return s.stageSuccess("stage_preview_ready", message, []string{"stage-preview:" + digest}, next, map[string]any{"previewDigest": digest, "preview": payload})
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

func integrationProjection(projection executiongraph.Graph) (executiongraph.ChildExecution, error) {
	for _, c := range projection.Children {
		if c.ExecutionID == projection.Parent.IntegrationChild {
			return c, nil
		}
	}
	return executiongraph.ChildExecution{}, executiongraph.ErrInvalidGraph
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
