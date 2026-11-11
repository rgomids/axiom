package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
	"github.com/rgomids/axiom/internal/workitem"
)

// Canonical Execution operations (ADR-0037, Spec 015 S1). One run action
// resolves the linked Work Item, the selected (or minimal default) Workflow
// and explicit per-agent selections, creates the format-4 Execution and drives
// the one fenced engine. There is no Stage Plan, Validator Policy, preview
// digest, expected revision, admission catalogue or Provider write.

var (
	errSelectionSyntax         = errors.New("invalid_selection")
	errSelectionsFrozen        = errors.New("selections_frozen")
	errWorkflowUnmappable      = errors.New("workflow_unmappable")
	errRepositoryBinding       = errors.New("repository_binding_changed")
	errExecutionProjectMissing = errors.New("execution_not_found")
	errProjectAdmission        = errors.New("project_admission_revoked")
	errReviewEvidence          = errors.New("review_evidence_invalid")
	errResultDigest            = errors.New("result_digest_mismatch")
	errReviewState             = fmt.Errorf("review state: %w", workflow.ErrTransition)
)

func (s lifecycleService) executionStage(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	// A human disposition never records without both its subject and explicit
	// human authority; only the built-in minimal Workflow can override the
	// Project selection. Checked before any read or write.
	if in.Operation == "reject" && (!in.AuthorizeHuman || !in.AuthorizeLocal || in.ReviewStage == "" || in.ReworkSource != "") {
		return s.stageFailure(completion.DeniedAuthority, "human_authority_required", "Rejection needs exact result and human/local authority", "Supply --stage --result-digest, --evidence or --note, --authorize-human --authorize-local", nil)
	}
	if in.Operation == "execution-run" && (in.ApproveGate != "" || in.ReviewStage != "") != in.AuthorizeHuman || in.Operation == "reconcile" && (in.Abandon != "") != in.AuthorizeHuman {
		return s.stageFailure(completion.DeniedAuthority, "human_authority_required", "A human disposition needs its exact subject and --authorize-human", "Repeat with both --approve-gate/--abandon and --authorize-human after real human review", nil)
	}
	if in.AuthorizeHuman && (os.Getenv("AXIOM_ATTEMPT_ID") != "" || os.Getenv("AXIOM_EXECUTION_ID") != "") {
		return s.stageFailure(completion.DeniedAuthority, "human_authority_required", "A Runtime child cannot record human approval", "Run this operation from the human operator context", nil)
	}
	if in.ReviewStage != "" && (in.ResultDigest == "" || (in.ReviewEvidence == "") == (in.ReviewNote == "") || in.ApproveGate != "") || in.ReviewStage == "" && (in.ResultDigest != "" || in.ReviewEvidence != "" || in.ReviewNote != "") {
		return s.stageFailure(completion.ValidationFailure, "invalid_review", "Review requires an exact stage, result digest and exactly one of retained Evidence or a human note", "Inspect status and supply --review-stage (or stage reject --stage) --result-digest with either --evidence <id> or --note <text>, and --authorize-human", nil)
	}
	if (in.Operation == "cancel" || in.Operation == "reconcile" && in.Abandon == "") && !in.AuthorizeLocal {
		return s.stageFailure(completion.DeniedAuthority, "authority_required", "This local Execution mutation needs explicit authority", "Repeat with --authorize-local", nil)
	}
	if in.Workflow != "" && in.Workflow != "minimal" {
		return s.stageFailure(completion.ValidationFailure, "invalid_selection", "Only the built-in minimal Workflow overrides the Project selection", "Use --workflow minimal or select the Project Workflow with project workflow select", nil)
	}
	operation := projectapp.AdmitExecutionAdvance
	if in.Operation == "status" || in.Operation == "diagnose" {
		operation = projectapp.AdmitExecutionStatus
	}
	selector := in.Project
	if selector == "" {
		id, failure := s.EffectiveProject(ctx, "")
		if id == "" {
			return cli.WorkflowStageResponse{Result: failure}
		}
		selector = id
	}
	if blocked := s.gate(ctx, selector, operation); blocked != nil {
		return cli.WorkflowStageResponse{Result: *blocked}
	}
	resolved := s.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return s.stageFailure(completion.ValidationFailure, resolved.Category, "Project could not be resolved", "Inspect project context", nil)
	}
	store, err := local.NewExecutionStore(s.stateRoot)
	if err != nil {
		return s.stageFailure(completion.ValidationFailure, "invalid_state_root", "Execution state root is unsafe", "Use an absolute, private LINGO_STATE_ROOT", nil)
	}
	c := executionComposition{s: s, store: store, project: resolved.Project, authorizeCommands: in.AuthorizeCommands}
	switch in.Operation {
	case "diagnose":
		return c.diagnose(ctx, in)
	case "execution-run":
		ctx, stop := interruptibleRun(ctx)
		defer stop()
		return c.run(ctx, in)
	case "reject":
		x, err := store.Load(ctx, resolved.Project.ID, in.Execution)
		if err != nil {
			return c.failure(err, nil)
		}
		if err = c.review(ctx, x, in); err != nil {
			return c.failure(err, &x)
		}
		x, err = store.Load(ctx, x.ProjectID, x.ExecutionID)
		if err != nil {
			return c.failure(err, nil)
		}
		return c.outcome(ctx, "Human rejection recorded; source facts preserved", x, nil, true)
	case "status":
		x, err := store.Load(ctx, resolved.Project.ID, in.Execution)
		if err != nil {
			return c.failure(err, nil)
		}
		return c.outcome(ctx, "Execution status", x, nil, true)
	case "cancel":
		x, err := c.engine(nil).Cancel(ctx, resolved.Project.ID, in.Execution, executionActor("user:"))
		if err != nil {
			return c.failure(err, nil)
		}
		return s.stageSuccess("cancel_requested", "Cancel intent recorded; the runner reports the observed stop", []string{"execution:" + x.ExecutionID}, "Inspect stage status for requested versus observed stop", executionPayload(x, nil))
	case "retry":
		ctx, stop := interruptibleRun(ctx)
		defer stop()
		return c.retry(ctx, in)
	case "reconcile":
		if in.Abandon != "" {
			x, facts, err := c.engine(nil).Abandon(ctx, resolved.Project.ID, in.Execution, in.Abandon, executionActor("human:"))
			if errors.Is(err, stageexecution.ErrOrphanLiveness) {
				return s.stageFailure(completion.Failure, "stop_orphan_then_abandon", "A recorded process of this Execution is alive or could not be observed; nothing was written or signalled", "Stop the process reported in reconciliation outside Axiom (confirm its PID and start identity), then repeat reconcile --abandon with --authorize-human", executionPayload(x, facts))
			}
			if err != nil {
				return c.failure(err, nil)
			}
			return s.stageSuccess("attempt_abandoned", "Unknown attempt abandoned by human disposition; its effects stay uncertain and were not replayed", []string{"execution:" + x.ExecutionID, "attempt:" + in.Abandon}, executionNext(x), executionPayload(x, nil))
		}
		launcher, _, err := c.existingLauncher(ctx, in.Execution)
		if err != nil {
			return c.failure(err, nil)
		}
		x, facts, err := c.engine(launcher).Reconcile(ctx, resolved.Project.ID, in.Execution)
		if err != nil {
			return c.failure(err, nil)
		}
		return c.outcome(ctx, "Execution reconciled without signalling or replay", x, facts, true)
	}
	return s.stageFailure(completion.ValidationFailure, "invalid_command", "Unsupported Execution operation", "Run axiom workflow --help", nil)
}

// interruptibleRun binds the one foreground engine (ADR-0037) to the
// operator's interrupt. The agent runs in its own process group, so a terminal
// Ctrl-C, SIGTERM or hang-up reaches only this runner: cancelling the run
// context makes OSProcessRunner stop and confirm its own verified group, and
// the engine commits the observed stop before exiting.
func interruptibleRun(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

type executionComposition struct {
	s       lifecycleService
	store   local.ExecutionStore
	project local.ResolvedProject
	// retrySelections are explicit overrides of one authorized new selection.
	retrySelections   []workflow.Selection
	retryRef          *workflow.TaskRef
	ordinaryRetry     bool
	authorizeCommands bool
}

func (c executionComposition) run(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	if !in.AuthorizeExternal {
		return c.s.stageFailure(completion.DeniedAuthority, "authority_required", "Spawning the selected Runtimes needs explicit authority", "Repeat with --authorize-external to allow vendor inference within the Workflow effect bounds", nil)
	}
	selections, err := parseSelections(in.Selections)
	if err != nil {
		return c.failure(err, nil)
	}
	if in.Execution != "" {
		if len(selections) != 0 || len(in.Inputs) != 0 || in.Workflow != "" || in.Repository != "" || in.WorkItem != "" || in.ReworkSource != "" {
			return c.failure(errSelectionsFrozen, nil)
		}
		launcher, x, err := c.existingLauncher(ctx, in.Execution)
		if err != nil {
			return c.failure(err, nil)
		}
		if in.ReviewStage != "" {
			if err := c.review(ctx, x, in); err != nil {
				return c.failure(err, &x)
			}
		}
		if in.ApproveGate != "" {
			if _, err := c.engine(nil).ApproveGate(ctx, c.project.ID, in.Execution, in.ApproveGate, executionActor("human:")); err != nil {
				return c.failure(err, &x)
			}
		}
		x, err = c.engine(launcher).Run(ctx, c.project.ID, in.Execution, stageexecution.RunOptions{})
		if err != nil {
			return c.failure(err, &x)
		}
		return c.outcome(ctx, "Execution progressed", x, nil, false)
	}
	if in.ApproveGate != "" || in.ReviewStage != "" {
		return c.failure(errSelectionSyntax, nil)
	}
	provider, resource, externalID, _ := cli.ParseWorkItemSelector(in.WorkItem)
	target, failed := c.s.workflows.ResolveLinked(ctx, workflowTarget(cli.WorkflowInput{Project: c.project.ID, Repository: in.Repository, Provider: provider, ProviderRepository: resource, ExternalID: externalID}))
	if failed.Category != "" {
		return c.s.stageFailure(completionStatus(failed.Status), failed.Category, "Linked Work Item could not be resolved", "Link the existing Work Item to this Project Repository with work-item select; nothing was written", nil)
	}
	request := stageexecution.StartRequest{ProjectID: c.project.ID, WorkItem: stageexecution.LinkedWorkItem{Provider: target.WorkItem.Provider, ID: workItemToken(target.WorkItem), RepositoryKey: target.Repository.Key}, Selections: selections, EnrollIntegration: true}
	for _, r := range c.project.Repositories {
		identity := repositoryIdentity(r)
		request.Repositories = append(request.Repositories, workflow.RepositoryBinding{Key: r.Key, Identity: identity})
		if r.Key == target.Repository.Key {
			request.WorkItem.RepositoryIdentity = identity
		}
	}
	if in.Workflow == "" {
		request.WorkflowID, request.WorkflowRevision, request.Stages, err = c.selectedWorkflow(ctx)
		if err != nil {
			return c.failure(err, nil)
		}
	}
	if err = freezeCommands(request.Stages, in.AuthorizeCommands); err != nil {
		return c.failure(err, nil)
	}
	if err = c.freezeInputs(ctx, request.Stages, in.Inputs, target.Repository.Key); err != nil {
		return c.failure(err, nil)
	}
	if failed := c.freezeWorkItem(ctx, request.Stages, target); failed.Category != "" {
		return c.s.stageFailure(failed.Status, failed.Category, "Work Item content could not be read", "Check Provider access to the linked Work Item; nothing was written", nil)
	}
	if in.ReworkSource != "" {
		source, loadErr := c.store.Load(ctx, c.project.ID, in.ReworkSource)
		if loadErr != nil {
			return c.failure(loadErr, nil)
		}
		link, linkErr := source.RejectedSource()
		if linkErr != nil {
			return c.failure(linkErr, nil)
		}
		request.ReworkOf = &link
	}
	if request.ExecutionID, err = allocateStageID(); err != nil {
		return c.failure(err, nil)
	}
	launcher, err := c.launcher(request.ExecutionID, target.Repository.Key)
	if err != nil {
		return c.failure(err, nil)
	}
	x, err := c.engine(launcher).Start(ctx, c.store, request)
	if err != nil {
		if x.ExecutionID == "" {
			return c.failure(err, nil)
		}
		return c.failure(err, &x)
	}
	return c.outcome(ctx, "Execution started", x, nil, false)
}

func (c executionComposition) retry(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	if !in.AuthorizeExternal {
		return c.s.stageFailure(completion.DeniedAuthority, "authority_required", "A retry spawns a Runtime process and needs explicit authority", "Repeat with --authorize-external", nil)
	}
	selections, err := parseSelections(in.Selections)
	if err != nil {
		return c.failure(err, nil)
	}
	if len(selections) != 0 && !in.NewSelection {
		// Selections without explicit new-selection authority are ambiguous:
		// an ordinary retry always reuses the frozen invocation.
		return c.failure(errSelectionsFrozen, nil)
	}
	ref := workflow.TaskRef{StageID: in.Stage, AgentID: in.Agent}
	launcher, _, err := c.existingLauncher(ctx, in.Execution)
	if err != nil {
		return c.failure(err, nil)
	}
	c.retrySelections, c.retryRef = selections, &ref
	c.ordinaryRetry = !in.NewSelection
	x, err := c.engine(launcher).Run(ctx, c.project.ID, in.Execution, stageexecution.RunOptions{Retry: &ref, NewSelection: in.NewSelection, WorkspaceDisposition: in.WorkspaceDisposition})
	if err != nil {
		return c.failure(err, &x)
	}
	return c.outcome(ctx, "Retry attempt recorded", x, nil, false)
}

func (c executionComposition) engine(launcher stageexecution.Launcher) stageexecution.Engine {
	return stageexecution.Engine{Store: c.store, Runner: executiongraph.OSProcessRunner{}, Launcher: launcher, Resolve: c.resolve, Admit: c.admit, NewID: allocateStageID, Workspaces: c.workspaceGit}
}

func executionWorktreeRoot(repository, executionID, key string) string {
	return filepath.Join(filepath.Dir(repository), ".axiom-worktrees-"+executionID+"-"+key)
}

func (c executionComposition) workspaceGit(ctx context.Context, x workflow.Execution, key string) (stageexecution.WorkspaceGit, error) {
	if err := c.admit(ctx, x); err != nil {
		return nil, err
	}
	for _, r := range c.project.Repositories {
		if r.Key == key {
			return gitworkspace.NewIntegrator(ctx, r.Path, executionWorktreeRoot(r.Path, x.ExecutionID, key))
		}
	}
	return nil, errRepositoryBinding
}

// admit re-reads the actual Project operational state and Repository binding
// before each dispatch; the composition's Project snapshot is never treated as
// current. The Project must still be admitted for Execution work and bind the
// Execution's repository with the same identity at the same local path the
// launcher froze, so a rebind can never silently retarget the workspace.
func (c executionComposition) admit(ctx context.Context, x workflow.Execution) error {
	if blocked := c.s.gate(ctx, c.project.ID, projectapp.AdmitExecutionAdvance); blocked != nil {
		return errProjectAdmission
	}
	resolved := c.s.installation.Resolve(ctx, c.project.ID)
	if resolved.Status != local.ResolutionFound || resolved.Project.ID != x.ProjectID {
		return errProjectAdmission
	}
	keys := []string{x.WorkItem.RepositoryKey}
	for _, stage := range x.Workflow.Stages {
		for _, a := range stage.Agents {
			if a.Repository != "" && !slices.Contains(keys, a.Repository) {
				keys = append(keys, a.Repository)
			}
		}
	}
	for _, key := range keys {
		if !c.repositoryAdmitted(x, resolved.Project.Repositories, key) {
			return errRepositoryBinding
		}
	}
	return nil
}

func (c executionComposition) repositoryAdmitted(x workflow.Execution, current []local.ResolvedRepository, key string) bool {
	bound, frozen := "", ""
	for _, r := range x.Repositories {
		if r.Key == key {
			bound = r.Identity
		}
	}
	for _, r := range c.project.Repositories {
		if r.Key == key {
			frozen = r.Path
		}
	}
	for _, r := range current {
		if r.Key == key && bound != "" && repositoryIdentity(r) == bound && frozen != "" && r.Path == frozen {
			return true
		}
	}
	return false
}

// launcher binds the machine-local workspace, private output root and Axiom
// path. Outputs live under the state root, never inside the repository.
func (c executionComposition) plannedLauncher(executionID, repositoryKey string) (stageexecution.RuntimeLauncher, error) {
	if !workflow.ValidExecutionID(executionID) {
		return stageexecution.RuntimeLauncher{}, errExecutionProjectMissing
	}
	workspace := ""
	for _, r := range c.project.Repositories {
		if r.Key == repositoryKey {
			workspace = r.Path
		}
	}
	if workspace == "" {
		return stageexecution.RuntimeLauncher{}, errRepositoryBinding
	}
	axiom, err := os.Executable()
	if err != nil {
		return stageexecution.RuntimeLauncher{}, err
	}
	// os.Executable is unresolved on darwin (Homebrew links, /var temp dirs);
	// Build refuses any symlinked component, so pass the resolved binary.
	axiom, err = filepath.EvalSymlinks(axiom)
	if err != nil {
		return stageexecution.RuntimeLauncher{}, err
	}
	workspaces := map[string]string{}
	worktreeRoots := map[string]string{}
	for _, r := range c.project.Repositories {
		workspaces[r.Key] = r.Path
		worktreeRoots[r.Key] = executionWorktreeRoot(r.Path, executionID, r.Key)
	}
	return stageexecution.RuntimeLauncher{Workspace: workspace, Workspaces: workspaces, WorktreeRoots: worktreeRoots, InboxRoot: filepath.Join(c.s.stateRoot, "executions", "v2-attempt-outputs", c.project.ID, executionID), AxiomPath: axiom, Environment: os.Environ()}, nil
}

func (c executionComposition) launcher(executionID, repositoryKey string) (stageexecution.RuntimeLauncher, error) {
	launcher, err := c.plannedLauncher(executionID, repositoryKey)
	if err != nil {
		return launcher, err
	}
	relative := filepath.Join("executions", "v2-attempt-outputs", c.project.ID, executionID)
	state, err := os.OpenRoot(c.s.stateRoot)
	if err != nil {
		return launcher, err
	}
	defer state.Close()
	if err = state.MkdirAll(relative, 0700); err != nil {
		return launcher, err
	}
	artifacts, err := local.NewArtifactStore(c.s.stateRoot)
	if err != nil {
		return launcher, err
	}
	validator := local.NewWorkflowReferenceValidator(artifacts)
	launcher.ValidateOutput = validator.ValidateExecutionOutput
	launcher.ReadInput = validator.ReadExecutionArtifact
	launcher.RetainOutput = func(ctx context.Context, subject workflow.OutputSubject, content []byte) (workflow.ArtifactRef, error) {
		return validator.RetainExecutionOutput(ctx, subject, content, c.s.provenance)
	}
	return launcher, nil
}

// existingLauncher binds the launcher to a persisted Execution's repository.
func (c executionComposition) existingLauncher(ctx context.Context, executionID string) (stageexecution.RuntimeLauncher, workflow.Execution, error) {
	if !workflow.ValidExecutionID(executionID) {
		return stageexecution.RuntimeLauncher{}, workflow.Execution{}, errExecutionProjectMissing
	}
	x, err := c.store.Load(ctx, c.project.ID, executionID)
	if err != nil {
		return stageexecution.RuntimeLauncher{}, x, err
	}
	launcher, err := c.launcher(executionID, x.WorkItem.RepositoryKey)
	return launcher, x, err
}

func repositoryIdentity(r local.ResolvedRepository) string {
	if r.CanonicalIdentity != "" && len(r.CanonicalIdentity) <= 512 {
		return r.CanonicalIdentity
	}
	return "project-repository:" + r.Key
}

func workItemToken(item workflow.WorkItem) string {
	return strings.ReplaceAll(item.Resource, "/", ":") + ":" + item.ExternalID
}

// resolve freezes one attempt's invocation and current authority. Run-level
// overrides are the Execution's immutable start selections, or the explicit
// new selection of one authorized retry. The reloaded Project must still be
// admitted and bind the Execution's repository with the same identity.
func (c executionComposition) resolve(ctx context.Context, x workflow.Execution, ref workflow.TaskRef) (workflow.Invocation, workflow.AttemptAuthority, error) {
	var agent workflow.AgentSpec
	found := false
	for _, stage := range x.Workflow.Stages {
		for _, a := range stage.Agents {
			if stage.ID == ref.StageID && a.ID == ref.AgentID {
				agent, found = a, true
			}
		}
	}
	if !found {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, workflow.ErrTransition
	}
	if err := c.admit(ctx, x); err != nil {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, err
	}
	config, err := c.runtimeConfiguration(ctx)
	if err != nil {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, err
	}
	selections := x.Selections
	if c.retryRef != nil && *c.retryRef == ref && len(c.retrySelections) != 0 {
		// The authorized new selection replaces only the fields it names; the
		// remaining start selections stay explicit run-level inputs.
		selections = nil
		for _, s := range x.Selections {
			replaced := false
			for _, n := range c.retrySelections {
				replaced = replaced || n.AgentID == s.AgentID && n.Field == s.Field && (s.StageID == "" || n.StageID == "" || n.StageID == s.StageID)
			}
			if !replaced {
				selections = append(selections, s)
			}
		}
		selections = append(selections, c.retrySelections...)
	}
	request := runtimeapplication.InvocationRequest{PermittedRuntimes: agent.Runtimes, Executables: map[string]string{}}
	var frozen *workflow.Invocation
	if c.ordinaryRetry && c.retryRef != nil && *c.retryRef == ref {
		for _, stage := range x.Stages {
			if stage.StageID == ref.StageID {
				for _, task := range stage.Tasks {
					if task.AgentID == ref.AgentID && len(task.Attempts) != 0 {
						previous := task.Attempts[len(task.Attempts)-1].Invocation
						frozen = &previous
					}
				}
			}
		}
	}
	if frozen != nil {
		for field, value := range map[string]string{runtimeapplication.FieldRuntime: frozen.Runtime, runtimeapplication.FieldModel: frozen.Model, runtimeapplication.FieldEffort: frozen.Effort, runtimeapplication.FieldInstructions: frozen.Instructions} {
			request.RunOverride.Assignments = append(request.RunOverride.Assignments, runtimeapplication.Assignment{Field: field, Value: value})
		}
	} else {
		if request.RunOverride, err = selectionLayer(selections, ref, config); err != nil {
			return workflow.Invocation{}, workflow.AttemptAuthority{}, err
		}
		request.LocalDefault, err = localInvocationDefaults(config)
		if err != nil {
			return workflow.Invocation{}, workflow.AttemptAuthority{}, err
		}
		request.WorkflowAgent = runtimeapplication.InvocationLayer{}
		for field, value := range map[string]string{runtimeapplication.FieldRuntime: agent.Runtime, runtimeapplication.FieldModel: agent.Model, runtimeapplication.FieldEffort: agent.Effort, runtimeapplication.FieldInstructions: agent.Instructions} {
			if value != "" {
				request.WorkflowAgent.Assignments = append(request.WorkflowAgent.Assignments, runtimeapplication.Assignment{Field: field, Value: value})
			}
		}
		if agent.Profile != "" {
			profile, err := profileInput(config, agent.Profile)
			if err != nil {
				return workflow.Invocation{}, workflow.AttemptAuthority{}, err
			}
			request.WorkflowAgent.Profile = &profile
		}

	}
	if config != nil {
		for _, r := range config.Runtimes {
			if !r.Enabled || r.Adapter != r.ID || (r.ID != "codex" && r.ID != "claude") || c.s.runtimes.lookPath == nil {
				continue
			}
			if path, err := c.s.runtimes.lookPath(r.ID); err == nil && filepath.IsAbs(path) {
				request.Executables[r.ID] = path
			}
		}
	}
	invocation, err := runtimeapplication.ResolveInvocation(request)
	if err != nil {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, err
	}
	if frozen != nil {
		if invocation.Runtime != frozen.Runtime || invocation.Executable != frozen.Executable {
			return workflow.Invocation{}, workflow.AttemptAuthority{}, runtimeapplication.ErrRuntimeNotConfigured
		}
		invocation = *frozen
	}
	if invocation.Profile != "" && config != nil {
		allowed := false
		for _, r := range config.Runtimes {
			allowed = allowed || r.ID == invocation.Runtime && slices.Contains(r.AllowlistedProfileIDs, invocation.Profile)
		}
		if !allowed {
			return workflow.Invocation{}, workflow.AttemptAuthority{}, runtimeapplication.ErrRuntimeNotPermitted
		}
	}
	effects := append([]string{}, agent.Effects...)
	if slices.Contains(effects, "validation_command") && !c.authorizeCommands {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, workflow.ErrUnauthorizedEffect
	}
	repository := x.WorkItem.RepositoryKey
	if agent.Repository != "" {
		repository = agent.Repository
	}
	return invocation, workflow.AttemptAuthority{RepositoryKey: repository, Effects: effects, Actor: executionActor("user:")}, nil
}

// runtimeConfiguration reads the machine-local Runtime bindings and optional
// Profiles. Absent configuration resolves no executable; it is not a
// catalogue requirement and is never generated.
func (c executionComposition) runtimeConfiguration(ctx context.Context) (*runtimeprofile.ConfigurationV4, error) {
	profiles, err := local.NewRuntimeProfileStore(c.s.stateRoot)
	if err != nil {
		return nil, err
	}
	snapshot, err := profiles.Inspect(ctx)
	if err != nil {
		return nil, runtimeapplication.ErrRuntimeNotConfigured
	}
	return snapshot.V4, nil
}

// localInvocationDefaults is explicitly configured machine-local input, never a
// capability preference or inferred selection. The bounded JSON array retains
// duplicate assignments so the resolver can reject conflicting sources.
// Example: LINGO_INVOCATION_DEFAULTS='["runtime=codex","model=my-model",
// "effort=runtime-default"]'. Profiles remain optional.
func localInvocationDefaults(config *runtimeprofile.ConfigurationV4) (runtimeapplication.InvocationLayer, error) {
	raw, present := os.LookupEnv("LINGO_INVOCATION_DEFAULTS")
	if !present {
		return runtimeapplication.InvocationLayer{}, nil
	}
	var values []string
	if strictjson.Decode([]byte(raw), &values, 64*1024) != nil || len(values) > 16 || values == nil {
		return runtimeapplication.InvocationLayer{}, runtimeapplication.ErrInvocationConflict
	}
	var selections []workflow.Selection
	for _, value := range values {
		field, value, ok := strings.Cut(value, "=")
		if !ok || !slices.Contains(workflow.SelectionFields, field) {
			return runtimeapplication.InvocationLayer{}, runtimeapplication.ErrInvocationConflict
		}
		selections = append(selections, workflow.Selection{AgentID: "default", Field: field, Value: value})
	}
	return selectionLayer(selections, workflow.TaskRef{AgentID: "default"}, config)
}

func profileInput(config *runtimeprofile.ConfigurationV4, id string) (runtimeapplication.ProfileInput, error) {
	if config == nil {
		return runtimeapplication.ProfileInput{}, runtimeapplication.ErrInvocationMissing
	}
	for _, p := range config.ModelProfiles {
		if p.ID != id {
			continue
		}
		wire, err := json.Marshal(p)
		if err != nil {
			return runtimeapplication.ProfileInput{}, err
		}
		sum := sha256.Sum256(wire)
		effort := p.Effort.Value
		if p.Effort.Mode == runtimeprofile.EffortRuntimeDefault {
			effort = "runtime-default"
		}
		return runtimeapplication.ProfileInput{ID: p.ID, Digest: "sha256:" + hex.EncodeToString(sum[:]), Runtime: p.RuntimeID, Model: p.Model, Effort: effort}, nil
	}
	return runtimeapplication.ProfileInput{}, runtimeapplication.ErrInvocationMissing
}

func selectionLayer(selections []workflow.Selection, ref workflow.TaskRef, config *runtimeprofile.ConfigurationV4) (runtimeapplication.InvocationLayer, error) {
	var layer runtimeapplication.InvocationLayer
	for _, s := range selections {
		if s.AgentID != ref.AgentID || s.StageID != "" && s.StageID != ref.StageID {
			continue
		}
		if s.Field != "profile" {
			layer.Assignments = append(layer.Assignments, runtimeapplication.Assignment{Field: s.Field, Value: s.Value})
			continue
		}
		if layer.Profile != nil && layer.Profile.ID != s.Value {
			return layer, runtimeapplication.ErrInvocationConflict
		}
		profile, err := profileInput(config, s.Value)
		if err != nil {
			return layer, err
		}
		layer.Profile = &profile
	}
	return layer, nil
}

// parseSelections accepts `[<stage-id>/]<agent-id>.<field>=<value>`.
func parseSelections(raw []string) ([]workflow.Selection, error) {
	var out []workflow.Selection
	for _, v := range raw {
		key, value, _ := strings.Cut(v, "=")
		target, field, ok := strings.Cut(key, ".")
		stage, agent, scoped := strings.Cut(target, "/")
		if !scoped {
			stage, agent = "", target
		}
		if !ok || !slices.Contains(workflow.SelectionFields, field) || !workflow.ValidToken(agent) || scoped && !workflow.ValidToken(stage) {
			return nil, errSelectionSyntax
		}
		for _, existing := range out {
			if existing.StageID == stage && existing.AgentID == agent && existing.Field == field {
				return nil, runtimeapplication.ErrInvocationConflict
			}
		}
		out = append(out, workflow.Selection{StageID: stage, AgentID: agent, Field: field, Value: value})
	}
	return out, nil
}

// selectedWorkflow maps the selected definition only when its controls have an
// exact implementation. Unsupported validators refuse before
// creation; substituting a later human gate would weaken the configured check.
func (c executionComposition) selectedWorkflow(ctx context.Context) (string, string, []workflow.StageSpec, error) {
	observation, category := (workflowDefinitions{installation: c.s.installation, portable: c.s.portable}).ResolveDefinition(ctx, c.project.ID)
	if category == "workflow_selection_required" {
		return "", "", nil, nil
	}
	if category != "" {
		return "", "", nil, errors.New(category)
	}
	id, revision, stages, err := mapWorkflowDefinition(observation.Document.Definition)
	if err != nil {
		return "", "", nil, err
	}
	for i := range stages {
		for j := range stages[i].FrozenInputs {
			input := &stages[i].FrozenInputs[j]
			if input.Kind != "project-context" {
				continue
			}
			context, ok := observation.Contexts[input.Source]
			if !ok {
				if input.Required {
					return "", "", nil, errWorkflowUnmappable
				}
				continue
			}
			if len(context.Content) > workflow.MaxInstructionBytes {
				return "", "", nil, errWorkflowUnmappable
			}
			input.Content = string(context.Content)
			input.Digest = "sha256:" + context.Digest
		}
	}
	return id, revision, stages, nil
}

func mapWorkflowDefinition(d workflowdefinition.Definition) (string, string, []workflow.StageSpec, error) {
	document, issues := workflowdefinition.Encode(d)
	if len(issues) != 0 {
		return "", "", nil, errWorkflowUnmappable
	}
	revision := "r" + strconv.Itoa(d.Revision)
	if !workflow.ValidToken(d.WorkflowID) {
		return "", "", nil, errWorkflowUnmappable
	}
	var stages []workflow.StageSpec
	previous := ""
	for _, s := range d.Stages {
		if s.Concurrency != 1 {
			return "", "", nil, errWorkflowUnmappable
		}
		for _, validator := range s.Validators {
			if validator.PolicyRef != "builtin-sdd-v1" || (validator.Kind != "artifact-schema" && validator.Kind != "evidence-check" && validator.Kind != "human-review" && validator.Kind != "command") {
				return "", "", nil, errWorkflowUnmappable
			}
		}
		spec := workflow.StageSpec{ID: s.ID, TimeoutSeconds: workflow.DefaultStageTimeout}
		for _, validator := range s.Validators {
			if validator.Kind == "human-review" {
				spec.ReviewRequired = true
			}
			if validator.Kind == "command" {
				spec.Commands = append(spec.Commands, workflow.CommandCheck{ID: validator.ID, Argv: append([]string(nil), validator.Argv...), TimeoutSeconds: validator.TimeoutSeconds, Effects: append([]string(nil), validator.EffectCeilings...)})
			}
		}
		for _, input := range s.Inputs {
			switch input.Kind {
			case "work-item":
				if input.Source != "target" {
					return "", "", nil, errWorkflowUnmappable
				}
				spec.WorkItemInputs = append(spec.WorkItemInputs, input.ID)
			case "project-context":

				spec.FrozenInputs = append(spec.FrozenInputs, workflow.FrozenInput{Name: input.ID, Kind: input.Kind, Source: input.Source, Required: input.Required})
			case "artifact":
				spec.FrozenInputs = append(spec.FrozenInputs, workflow.FrozenInput{Name: input.ID, Kind: input.Kind, Source: input.Source, Required: input.Required})
			case "stage-output":
				parts := strings.Split(input.Source, "/")
				if len(parts) != 2 {
					return "", "", nil, errWorkflowUnmappable
				}
				spec.StageOutputInputs = append(spec.StageOutputInputs, workflow.StageOutputInput{Name: input.ID, StageID: parts[0], Output: parts[1], Required: input.Required})
			default:
				// Unimplemented context/artifact resolution must not disappear just
				// because a stage has no validators.
				return "", "", nil, errWorkflowUnmappable
			}
		}

		for _, gate := range s.HumanGates {
			switch gate.Timing {
			case "before":
				spec.BeforeGate = true
			case "after":
				spec.HumanGate = true
			default:
				return "", "", nil, errWorkflowUnmappable
			}
		}

		if previous != "" {
			spec.DependsOn = []string{previous}
		}
		// Every completion criterion requires its referenced output to be present
		// (as the historical configured executor did), even when the output
		// declaration alone is optional. Unreferenced optional outputs stay optional.
		criterionOutputs := map[string]bool{}
		for _, criterion := range s.CompletionCriteria {
			criterionOutputs[criterion.OutputRef] = true
		}
		for _, o := range s.Outputs {
			if o.Required || criterionOutputs[o.ID] {
				spec.Outputs = append(spec.Outputs, o.ID)
			} else {
				spec.OptionalOutputs = append(spec.OptionalOutputs, o.ID)
			}
			if len(s.Validators) != 0 {
				check := workflow.OutputCheck{Name: o.ID, Kind: o.Kind}
				for _, criterion := range s.CompletionCriteria {
					for _, validator := range s.Validators {
						if criterion.OutputRef == o.ID && criterion.ValidatorRef == validator.ID && validator.Kind == "evidence-check" {
							check.Evidence = true
						}
					}
				}
				spec.OutputChecks = append(spec.OutputChecks, check)
				spec.DefinitionDigest = "sha256:" + document.Digest
			}
		}
		for _, a := range s.Agents {
			agent := workflow.AgentSpec{ID: a.ID, Repository: a.Repository, OutputScope: true, DependsOn: append([]string(nil), a.DependsOn...), Outputs: append([]string(nil), a.Outputs...), TimeoutSeconds: int64(a.TimeoutSeconds), MaximumAttempts: a.MaximumAttempts, Instructions: strings.TrimSpace(s.Instructions + "\n\n" + a.Responsibilities), Runtimes: append([]string(nil), a.RuntimeConstraints...)}
			agent.Inputs = append([]string(nil), a.Inputs...)
			// policy-default named the retired Project Runtime policy; the
			// canonical path uses explicit selections instead of a guess.
			if a.ProfileRef != "" && a.ProfileRef != "policy-default" {
				agent.Profile = a.ProfileRef
			}
			switch a.Effort.Mode {
			case runtimeprofile.EffortRuntimeDefault:
				agent.Effort = "runtime-default"
			case runtimeprofile.EffortExplicit:
				agent.Effort = a.Effort.Value
			}
			// Schema-2 explicit Workflow-agent values (never combined with the
			// historical schema-1 effort above, which schema 2 refuses).
			if a.Invocation != nil {
				agent.Runtime, agent.Model = a.Invocation.Runtime, a.Invocation.Model
				if a.Invocation.Effort != "" {
					agent.Effort = a.Invocation.Effort
				}
			}
			if slices.Contains(a.EffectCeilings, "repository-write") {
				agent.Effects = []string{"workspace_write"}
			}
			if a.TimeoutSeconds > 0 && int64(a.TimeoutSeconds) > spec.TimeoutSeconds {
				spec.TimeoutSeconds = int64(a.TimeoutSeconds)
			}
			for _, command := range spec.Commands {
				for _, effect := range command.Effects {
					if effect == "network" || effect == "workspace_write" && !slices.Contains(a.EffectCeilings, "repository-write") || effect == "validation_command" && !slices.Contains(a.EffectCeilings, "process") {
						return "", "", nil, errWorkflowUnmappable
					}
					if !slices.Contains(agent.Effects, effect) {
						agent.Effects = append(agent.Effects, effect)
					}
				}
			}
			agent.Outputs = nil
			for _, output := range append(append([]string(nil), spec.Outputs...), spec.OptionalOutputs...) {
				if slices.Contains(a.Outputs, output) {
					agent.Outputs = append(agent.Outputs, output)
				}
			}
			spec.Agents = append(spec.Agents, agent)
		}
		// A criterion output that no agent produces can never be satisfied.
		for output := range criterionOutputs {
			produced := false
			for _, agent := range spec.Agents {
				produced = produced || slices.Contains(agent.Outputs, output)
			}
			if !produced {
				return "", "", nil, errWorkflowUnmappable
			}
		}
		stages = append(stages, spec)
		previous = s.ID
	}
	// Agent Repository keys are checked against the real bindings at start.
	repositories := []workflow.RepositoryBinding{{Key: "repo", Identity: "validation"}}
	for _, s := range stages {
		for _, a := range s.Agents {
			if a.Repository != "" && !slices.ContainsFunc(repositories, func(r workflow.RepositoryBinding) bool { return r.Key == a.Repository }) {
				repositories = append(repositories, workflow.RepositoryBinding{Key: a.Repository, Identity: "validation-" + a.Repository})
			}
		}
	}
	if _, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: "00000000-0000-0000-0000-000000000001", ProjectID: "validation", WorkItem: workflow.WorkItemLink{Provider: "local", ID: "validation", RepositoryKey: "repo"}, Repositories: repositories, WorkflowID: d.WorkflowID, Revision: revision, Stages: stages, Now: time.Unix(1, 0)}); err != nil {
		return "", "", nil, errWorkflowUnmappable
	}
	return d.WorkflowID, revision, stages, nil
}

func executionActor(prefix string) string {
	user := stageActor()
	if !workflow.ValidToken(prefix + user) {
		user = "local-operator"
	}
	return prefix + user
}

func completionStatus(s workflow.Status) completion.Status {
	switch s {
	case workflow.ValidationFailed:
		return completion.ValidationFailure
	case workflow.Interrupted:
		return completion.Interrupted
	}
	return completion.Failure
}

// --- presentation ----------------------------------------------------------

type executionAttemptView struct {
	Workspace            *executionWorkspaceView    `json:"workspace,omitempty"`
	WorkspaceUncertain   bool                       `json:"workspaceUncertain,omitempty"`
	LaunchFailure        *workflow.LaunchFailure    `json:"launchFailure,omitempty"`
	NextAction           string                     `json:"nextAction,omitempty"`
	WorkspaceDisposition string                     `json:"workspaceDisposition,omitempty"`
	AttemptID            string                     `json:"attemptId"`
	Number               int                        `json:"number"`
	RunnerGeneration     uint64                     `json:"runnerGeneration"`
	Status               workflow.AttemptStatus     `json:"status"`
	Runtime              string                     `json:"runtime"`
	Model                string                     `json:"model"`
	Effort               string                     `json:"effort"`
	Profile              string                     `json:"profile,omitempty"`
	ProfileDigest        string                     `json:"profileDigest,omitempty"`
	InstructionsDigest   string                     `json:"instructionsDigest"`
	Provenance           []workflow.FieldProvenance `json:"provenance"`
	RepositoryKey        string                     `json:"repositoryKey"`
	AuthorizedEffects    []string                   `json:"authorizedEffects"`
	Process              *executionProcessView      `json:"process,omitempty"`
	ExitCode             *int                       `json:"exitCode,omitempty"`
	Diagnostic           string                     `json:"diagnostic,omitempty"`
	Effects              []workflow.EffectFact      `json:"effects,omitempty"`
	Artifacts            []workflow.ArtifactRef     `json:"artifacts,omitempty"`
	Resolution           *workflow.Disposition      `json:"resolution,omitempty"`
}

// Portable workspace facts omit machine-local paths and repository identity.
type executionWorkspaceView struct {
	BaseSHA           string `json:"baseSha"`
	SourceBranch      string `json:"sourceBranch"`
	Worktree          string `json:"worktree"`
	TargetRef         string `json:"targetRef"`
	InitialTargetHead string `json:"initialTargetHead"`
}

type executionProcessView struct {
	PID           int64  `json:"pid"`
	StartIdentity string `json:"startIdentity"`
}

type executionTaskView struct {
	AgentID  string                 `json:"agentId"`
	Attempts []executionAttemptView `json:"attempts"`
}

type executionStageView struct {
	StageID        string                 `json:"stageId"`
	Status         workflow.StageStatus   `json:"status"`
	Gate           *workflow.Disposition  `json:"gate,omitempty"`
	BeforeApproval *workflow.Disposition  `json:"beforeApproval,omitempty"`
	Review         *workflow.ResultReview `json:"review,omitempty"`
	ResultDigest   string                 `json:"resultDigest,omitempty"`
	Tasks          []executionTaskView    `json:"tasks"`
}

// executionView is the safe projection: no executable path, instruction
// text, repository identity or child output.
type executionView struct {
	Rejection          *workflow.ResultRejection      `json:"rejection,omitempty"`
	ReworkOf           *workflow.RejectedSource       `json:"reworkOf,omitempty"`
	Deliveries         []workflow.ExecutionDelivery   `json:"deliveries,omitempty"`
	RepositoryOutcomes []workflow.RepositoryOutcome   `json:"repositoryOutcomes"`
	ExecutionID        string                         `json:"executionId"`
	ProjectID          string                         `json:"projectId"`
	WorkItem           workflow.WorkItemLink          `json:"workItem"`
	WorkflowID         string                         `json:"workflowId"`
	WorkflowRevision   string                         `json:"workflowRevision"`
	WorkflowDigest     string                         `json:"workflowDigest"`
	Revision           uint64                         `json:"revision"`
	RunnerGeneration   uint64                         `json:"runnerGeneration"`
	Status             workflow.RunStatus             `json:"status"`
	Cancel             *workflow.CancelIntent         `json:"cancel,omitempty"`
	Stages             []executionStageView           `json:"stages"`
	Reconciliation     []stageexecution.ReconcileFact `json:"reconciliation,omitempty"`
	NextAction         string                         `json:"nextAction"`
}

func executionPayload(x workflow.Execution, facts []stageexecution.ReconcileFact) executionView {
	v := executionView{Rejection: x.Rejection, ReworkOf: x.ReworkOf, ExecutionID: x.ExecutionID, ProjectID: x.ProjectID, WorkItem: x.WorkItem, WorkflowID: x.Workflow.ID, WorkflowRevision: x.Workflow.Revision, WorkflowDigest: x.Workflow.Digest,
		Revision: x.Revision, RunnerGeneration: x.RunnerGeneration, Status: x.Status, Cancel: x.Cancel, Stages: []executionStageView{}, Reconciliation: facts, NextAction: executionNext(x)}
	for _, s := range x.Stages {
		stage := executionStageView{StageID: s.StageID, Status: s.Status, Gate: s.Gate, BeforeApproval: s.BeforeApproval, Review: s.Review, Tasks: []executionTaskView{}}
		stage.ResultDigest, _ = x.StageResultDigest(s.StageID)
		for _, t := range s.Tasks {
			task := executionTaskView{AgentID: t.AgentID, Attempts: []executionAttemptView{}}
			for _, a := range t.Attempts {
				view := executionAttemptView{WorkspaceDisposition: a.WorkspaceDisposition, AttemptID: a.AttemptID, Number: a.Number, RunnerGeneration: a.RunnerGeneration, Status: a.Status, Runtime: a.Invocation.Runtime, Model: a.Invocation.Model, Effort: a.Invocation.Effort,
					Profile: a.Invocation.Profile, ProfileDigest: a.Invocation.ProfileDigest, InstructionsDigest: a.Invocation.InstructionsDigest, Provenance: a.Invocation.Provenance,
					RepositoryKey: a.Authority.RepositoryKey, AuthorizedEffects: append([]string{}, a.Authority.Effects...), ExitCode: a.ExitCode, Diagnostic: a.Diagnostic, LaunchFailure: a.LaunchFailure, Effects: a.Effects, Artifacts: a.Artifacts, Resolution: a.Resolution}
				if a.LaunchFailure != nil {
					view.NextAction = a.LaunchFailure.NextAction()
				}
				if w := a.Workspace; w != nil {
					view.Workspace = &executionWorkspaceView{w.BaseSHA, w.SourceBranch, w.Worktree, w.TargetRef, w.InitialTargetHead}
					view.WorkspaceUncertain = a.WorkspaceUncertain
				}
				if a.Process != nil {
					view.Process = &executionProcessView{PID: a.Process.PID, StartIdentity: a.Process.StartIdentity}
				}
				task.Attempts = append(task.Attempts, view)
			}
			stage.Tasks = append(stage.Tasks, task)
		}
		v.Stages = append(v.Stages, stage)
	}
	v.RepositoryOutcomes = x.RepositoryOutcomes()
	v.Deliveries = x.Deliveries

	return v
}

// executionNext is the smallest safe next action for the persisted facts.
// Technical completion is never human acceptance.
func executionNext(x workflow.Execution) string {
	if x.Rejection != nil {
		return "Rejected source is immutable. Start a new workflow run --rework-of " + x.ExecutionID + " with the same linked Work Item/Repositories and fresh selection/authority; no permissions transfer"
	}
	selector := " --execution " + x.ExecutionID
	for _, stage := range x.Stages {
		for _, task := range stage.Tasks {
			for _, a := range task.Attempts {
				if a.WorkspaceUncertain {
					return "The retained workspace/integration outcome is uncertain; inspect stage status and branches/worktrees without replay. No public integration resolution is available"
				}
			}
		}
	}
	for _, i := range x.Integrations {
		if x.RequiresIntegration(i.SourceAttemptID) && i.Status != workflow.IntegrationIntegrated {
			return "Inspect retained integration " + i.IntegrationID + " and its branches/worktrees; no automatic retry, reset or public integration resolution is available"
		}
	}
	for k, d := range x.Deliveries {
		if d.Status == "unknown" || d.Status == "prepared" {
			return fmt.Sprintf("Inspect exact delivery target: axiom workflow delivery advance%s --inspect %d --grant %s --authorize-local --authorize-human with its original --authorize-push or --authorize-pr; no replay or acceptance", selector, k, d.Grant.Digest)
		}
	}
	switch x.Status {
	case workflow.RunCompleted:
		return "Inspect the recorded artifacts; technical completion is not human acceptance"
	case workflow.RunAwaitingGate:
		for _, s := range x.Stages {
			if s.Status == workflow.StageAwaitingGate {
				for _, spec := range x.Workflow.Stages {
					if spec.ID == s.StageID && spec.ReviewRequired && s.Review == nil {
						if digest, err := x.StageResultDigest(s.StageID); err == nil {
							return "After real human review: axiom workflow run" + selector + " --review-stage " + s.StageID + " --result-digest " + digest + " --note <review-note> --authorize-human --authorize-external"
						}
					}
				}
				return "After real human review: axiom workflow run" + selector + " --approve-gate " + s.StageID + " --authorize-human --authorize-external"
			}
		}
	case workflow.RunRecoveryRequired:
		return "Effects are unknown and are never replayed: run axiom workflow stage reconcile" + selector + " --authorize-local, stop any process it reports alive, then axiom workflow stage reconcile" + selector + " --abandon <attempt-id> --authorize-human before any explicit retry"
	case workflow.RunFailed:
		return "Inspect the attempt diagnostic; then axiom workflow stage retry" + selector + " --stage <stage-id> --agent <agent-id> --authorize-external"
	case workflow.RunCancelled:
		return "The Execution is cancelled; start a new Execution with axiom workflow run"
	case workflow.RunRunning:
		return "A runner is active: inspect axiom workflow stage status" + selector + " or request axiom workflow stage cancel" + selector
	}
	return "axiom workflow run" + selector + " --authorize-external"
}

func (c executionComposition) outcome(ctx context.Context, message string, x workflow.Execution, facts []stageexecution.ReconcileFact, read bool) cli.WorkflowStageResponse {
	payload := executionPayload(x, facts)
	payload.RepositoryOutcomes = c.repositoryOutcomes(ctx, x)
	refs := []string{"execution:" + x.ExecutionID}
	if read {
		return c.s.stageSuccess("execution_"+string(x.Status), message, refs, payload.NextAction, payload)
	}
	switch x.Status {
	case workflow.RunCompleted, workflow.RunAwaitingGate, workflow.RunReady:
		return c.s.stageSuccess("execution_"+string(x.Status), message, refs, payload.NextAction, payload)
	case workflow.RunCancelled:
		return c.s.stageFailure(completion.Interrupted, "execution_cancelled", message, payload.NextAction, payload)
	case workflow.RunRecoveryRequired:
		return c.s.stageFailure(completion.Failure, "recovery_required", message, payload.NextAction, payload)
	}
	return c.s.stageFailure(completion.Failure, "execution_"+string(x.Status), message, payload.NextAction, payload)
}

// failure maps refusals to bounded categories; error text from Runtimes,
// paths or child output never reaches the response.
func (c executionComposition) failure(err error, x *workflow.Execution) cli.WorkflowStageResponse {
	var payload any
	if x != nil && x.ExecutionID != "" {
		payload = executionPayload(*x, nil)
	}
	type rule struct {
		target   error
		status   completion.Status
		category string
		next     string
	}
	rules := []rule{
		{stageexecution.ErrIntegrationUnknown, completion.Failure, "recovery_required", "Integration effects are unknown; inspect retained state and the exact target without replay"},
		{stageexecution.ErrNotIntegrated, completion.Failure, "not_integrated", "Inspect retained integration checks/conflict and branches/worktrees; no automatic retry or reset is available"},
		{workflow.ErrIntegrationChecksUnavailable, completion.ValidationFailure, "integration_checks_required", "Configure post-merge stage commands and authorize them with --authorize-commands before starting enrolled writers"},
		{workflow.ErrIntegrationStale, completion.Failure, "integration_base_changed", "The clean base or integration target changed; preserve all branches/worktrees and inspect before starting a new Execution"},
		{gitworkspace.ErrWorkspaceCollision, completion.Failure, "workspace_collision", "A reserved branch or worktree already exists; it was not adopted or overwritten. Inspect retained state"},
		{stageexecution.ErrAfterGateIntegrationDependency, completion.Failure, "integration_refused", "This Stage's after human gate would wait on a same-Stage dependent of an integrated writer; move the review/acceptance gate to a separate later Stage that runs after integration. Nothing was spawned or written to Git"},
		{stageexecution.ErrIntegrationRefused, completion.Failure, "integration_refused", "The recorded workspace/result could not be safely integrated; preserve it and inspect stage status"},
		{errSelectionSyntax, completion.ValidationFailure, "invalid_selection", "Use --select [<stage-id>/]<agent-id>.<runtime|model|effort|instructions|profile>=<value> naming a Workflow agent"},
		{errSelectionsFrozen, completion.ValidationFailure, "selections_frozen", "Start selections are immutable; change them only with stage retry --new-selection --select ..."},
		{runtimeapplication.ErrInvocationConflict, completion.ValidationFailure, "invocation_selection_conflict", "Give each agent field one value per level; nothing was written"},
		{runtimeapplication.ErrInvocationMissing, completion.ValidationFailure, "invocation_selection_missing", "Select runtime, model and effort (or runtime-default) explicitly, or an existing Profile; nothing is inferred"},
		{runtimeapplication.ErrRuntimeNotPermitted, completion.DeniedAuthority, "runtime_not_permitted", "Select a Runtime allowed by the Workflow agent and local bindings"},
		{runtimeapplication.ErrRuntimeNotConfigured, completion.ValidationFailure, "runtime_not_configured", "Create and enable the machine-local Runtime binding (axiom runtime binding create) with its executable on PATH"},
		{errWorkflowUnmappable, completion.ValidationFailure, "workflow_unmappable", "Select a Workflow with supported identifiers and controls; configured validators need exact implementation before dispatch"},
		{errRepositoryBinding, completion.DeniedAuthority, "repository_binding_changed", "The Project no longer binds this Execution's repository identity; start a new Execution"},
		{errProjectAdmission, completion.DeniedAuthority, "project_admission_revoked", "The Project is no longer admitted for Execution work (archived, removed or invalid); nothing further was spawned. Inspect the Project and continue only after it is admitted again"},
		{errExecutionProjectMissing, completion.ValidationFailure, "execution_not_found", "Use the exact Execution identity returned by workflow run"},
		{stageexecution.ErrWorkItemLinkMismatch, completion.ValidationFailure, "work_item_link_mismatch", "Link the Work Item to this Project Repository; nothing was written"},
		{workflow.ErrLegacyExecution, completion.ValidationFailure, "execution_legacy_format_refused", "Old Executions cannot resume after the breaking cutover; start a new Execution with workflow run"},
		{workflow.ErrRunnerHeld, completion.RetryableFailure, "execution_runner_active", "Another runner holds this Execution; inspect stage status and wait or cancel"},
		{workflow.ErrStaleRunner, completion.DeniedAuthority, "execution_stale_runner", "A newer runner generation owns this Execution; inspect stage status"},
		{workflow.ErrUnknownOutstanding, completion.Failure, "recovery_required", "Effects are unknown and never replayed; reconcile and record a human disposition first"},
		{workflow.ErrCancelled, completion.Interrupted, "execution_cancelled", "The Execution is cancelled; start a new Execution"},
		{workflow.ErrUnauthorizedEffect, completion.DeniedAuthority, "authority_denied", "Keep the attempt within the Workflow agent effect and Runtime restrictions"},
		{workflow.ErrWorkspaceDispositionRequired, completion.DeniedAuthority, "workspace_disposition_required", "Inspect and preserve the prior workspace explicitly with --disposition preserve; no automatic reset is supported"},
		{errResultDigest, completion.ValidationFailure, "result_digest_mismatch", "The stage has no completed result with this digest; copy --stage and --result-digest exactly from stage status"},
		{errReviewEvidence, completion.ValidationFailure, "review_evidence_invalid", "Evidence must be a retained success evidence artifact of this Execution bound once to the exact stage, definition digest, result digest and repository; or pass --note <text> (no secret-like content) and Axiom retains it"},
		{errReviewState, completion.ValidationFailure, "transition_refused", "Evidence and digest are valid, but the Execution state does not admit this disposition (already rejected or reviewed, cancelled, recovery required, an active attempt, a prepared integration/delivery, or the stage requires no review); inspect stage status"},
		{workflow.ErrTransition, completion.ValidationFailure, "transition_refused", "Inspect stage status for the eligible next action"},
		{local.ErrConflict, completion.RetryableFailure, "execution_revision_conflict", "A concurrent writer changed the Execution; inspect status before repeating"},
		{local.ErrNotFound, completion.ValidationFailure, "execution_not_found", "Use the exact Execution identity returned by workflow run"},
		{os.ErrNotExist, completion.ValidationFailure, "execution_not_found", "Use the exact Execution identity returned by workflow run"},
		{workflow.ErrExecutionInvalid, completion.Failure, "recovery_required", "The Execution record failed validation; nothing was changed"},
		{context.Canceled, completion.Interrupted, "workflow_cancelled", "Inspect stage status before repeating any mutation"},
	}
	for _, r := range rules {
		if errors.Is(err, r.target) {
			return c.s.stageFailure(r.status, r.category, "Execution operation refused", r.next, payload)
		}
	}
	for _, category := range []string{"recovery_required", "project_configuration_drift", "unsupported_portable_source", "stage_prerequisite_missing"} {
		if err.Error() == category {
			return c.s.stageFailure(completion.Failure, category, "Selected Workflow could not be read", "Inspect project workflow selection", payload)
		}
	}
	return c.s.stageFailure(completion.Failure, "execution_operation_failed", "Execution operation failed", "Inspect stage status; never repeat an authorized mutation blindly", payload)
}

// freezeInputs resolves only explicit input sources and records their provenance
// in the immutable Workflow snapshot before any Execution or process exists.
func (c executionComposition) freezeInputs(ctx context.Context, stages []workflow.StageSpec, bindings []string, repository string) error {
	refs := map[string]string{}
	used := map[string]bool{}
	for _, binding := range bindings {
		key, id, ok := strings.Cut(binding, "=")
		if _, duplicate := refs[key]; !ok || key == "" || id == "" || duplicate {
			return errSelectionSyntax
		}
		refs[key] = id
	}
	artifacts, err := local.NewArtifactStore(c.s.stateRoot)
	if err != nil {
		return err
	}
	for i := range stages {
		for j := range stages[i].FrozenInputs {
			input := &stages[i].FrozenInputs[j]
			if input.Kind == "project-context" {
				if input.Source != "project" {
					if input.Required && input.Content == "" {
						return errWorkflowUnmappable
					}
					continue
				}
				revision, valid := c.project.PortableRevision.Digest()
				if !valid {
					return errWorkflowUnmappable
				}
				p := &workflow.ProjectInput{ID: c.project.ID, Slug: c.project.Slug, PortableRevision: "sha256:" + hex.EncodeToString(revision[:])}
				for _, r := range c.project.Repositories {
					p.Repositories = append(p.Repositories, workflow.RepositoryBinding{Key: r.Key, Identity: repositoryIdentity(r)})
				}
				input.Project = p
				continue
			}
			id := refs[input.Source]
			if id == "" {
				if input.Required {
					return errWorkflowUnmappable
				}
				continue
			}
			used[input.Source] = true
			a, err := artifacts.Read(ctx, id)
			if err != nil || a.Outcome != "success" {
				return errWorkflowUnmappable
			}
			count := 0
			for _, ref := range a.References {
				if ref.Kind == "workflow-repository" {
					if ref.Value != repository {
						return errRepositoryBinding
					}
					count++
				}
			}
			if count != 1 {
				return errWorkflowUnmappable
			}
			provenance := &workflow.InputProvenance{ExecutionID: a.ExecutionID}
			for kind, target := range map[string]*string{"workflow-stage": &provenance.StageID, "workflow-attempt": &provenance.AttemptID} {
				count := 0
				for _, ref := range a.References {
					if ref.Kind == kind {
						*target = ref.Value
						count++
					}
				}
				if count != 1 {
					return errWorkflowUnmappable
				}
			}
			if !provenance.Valid() {
				return errWorkflowUnmappable
			}
			input.Provenance = provenance
			input.Artifact = &workflow.ArtifactRef{Name: input.Name, RetainedID: a.ID, RepositoryKey: repository, Digest: "sha256:" + hex.EncodeToString(a.Digest[:]), Size: int64(a.ContentBytes)}
		}
	}
	for key := range refs {
		if !used[key] {
			return errSelectionSyntax
		}
	}
	return nil
}

func (c executionComposition) review(ctx context.Context, x workflow.Execution, in cli.WorkflowStageInput) error {
	digest, err := x.StageResultDigest(in.ReviewStage)
	if err != nil || digest != in.ResultDigest {
		return errResultDigest
	}
	transition := func(current *workflow.Execution, evidence workflow.ArtifactRef) error {
		var err error
		if in.Operation == "reject" {
			err = current.RejectResult(in.ReviewStage, executionActor("human:"), digest, evidence, time.Now())
		} else {
			err = current.ReviewStage(in.ReviewStage, executionActor("human:"), digest, evidence, time.Now())
		}
		if errors.Is(err, workflow.ErrTransition) {
			return errReviewState
		}
		return err
	}
	store, err := local.NewArtifactStore(c.s.stateRoot)
	if err != nil {
		return err
	}
	references := map[string]string{"workflow-stage": in.ReviewStage, "workflow-definition": x.Workflow.Digest, "workflow-result": digest, "workflow-repository": x.WorkItem.RepositoryKey}
	var a detailartifact.Artifact
	if in.ReviewNote != "" {
		note := []byte(in.ReviewNote)
		if !detailartifact.SafeMarkdown(note) {
			return errReviewEvidence
		}
		// Dry-run the disposition on a private copy so a refused state retains
		// no orphan Evidence; the fenced update below re-checks authoritatively.
		wire, err := json.Marshal(x)
		if err != nil {
			return err
		}
		var probe workflow.Execution
		if err = json.Unmarshal(wire, &probe); err != nil {
			return err
		}
		if err = transition(&probe, workflow.ArtifactRef{Name: "human-review", RetainedID: "pending", RepositoryKey: x.WorkItem.RepositoryKey, Digest: workflow.DigestText(note), Size: int64(len(note))}); err != nil {
			return err
		}
		draft := detailartifact.Draft{ExecutionID: x.ExecutionID, Category: "evidence", Outcome: "success", Retention: detailartifact.Evidence, Markdown: note, Provenance: c.s.provenance}
		for _, kind := range []string{"workflow-stage", "workflow-definition", "workflow-result", "workflow-repository"} {
			draft.References = append(draft.References, detailartifact.Reference{Kind: kind, Value: references[kind]})
		}
		if a, err = store.Create(ctx, draft); err != nil {
			return err
		}
	} else {
		a, err = store.Read(ctx, in.ReviewEvidence)
		if err != nil || a.ExecutionID != x.ExecutionID || a.Category != "evidence" || a.Retention != "evidence" || a.Outcome != "success" {
			return errReviewEvidence
		}
		for kind, value := range references {
			count := 0
			for _, r := range a.References {
				if r.Kind == kind {
					if r.Value != value {
						return errReviewEvidence
					}
					count++
				}
			}
			if count != 1 {
				return errReviewEvidence
			}
		}
	}
	evidence := workflow.ArtifactRef{Name: "human-review", RetainedID: a.ID, RepositoryKey: x.WorkItem.RepositoryKey, Digest: "sha256:" + hex.EncodeToString(a.Digest[:]), Size: int64(a.ContentBytes)}
	_, err = c.store.UpdateFenced(ctx, x.ProjectID, x.ExecutionID, nil, func(current *workflow.Execution) error {
		return transition(current, evidence)
	})
	return err
}

// Freeze command executable resolution once; ordinary retries never pick a
// different executable from a changed PATH. The explicit argv stays in state.
// The frozen path is the symlink-free target (python3, alternatives, merged
// /usr are commonly links), checked with the launcher's own containment rule
// before any Execution exists, so a command that could never start is refused
// here instead of becoming an unknown effect after the agent ran.
func freezeCommands(stages []workflow.StageSpec, authorized bool) error {
	for i := range stages {
		for j := range stages[i].Commands {
			command := &stages[i].Commands[j]
			if !authorized {
				return workflow.ErrUnauthorizedEffect
			}
			if len(command.Argv) == 0 {
				return errWorkflowUnmappable
			}
			path, err := exec.LookPath(command.Argv[0])
			if err != nil {
				return errWorkflowUnmappable
			}
			path, err = filepath.Abs(path)
			if err != nil {
				return errWorkflowUnmappable
			}
			path, err = filepath.EvalSymlinks(path)
			if err != nil || !stageexecution.CleanRuntimePath(path) {
				return errWorkflowUnmappable
			}
			command.Argv[0] = path
		}
	}
	return nil
}

// freezeWorkItem reads the linked Work Item once at start and freezes its title
// and body, with a digest, into every stage input that names the target. Stage
// children have no Provider access, so without it they only see an identifier.
func (c executionComposition) freezeWorkItem(ctx context.Context, stages []workflow.StageSpec, target workflow.ResolvedTarget) workitem.Result {
	needed := false
	for _, stage := range stages {
		needed = needed || len(stage.WorkItemInputs) != 0
	}
	if !needed {
		return workitem.Result{}
	}
	document, failed := c.s.workItems.ReadDocument(ctx, workitem.Target{ProjectSelector: c.project.ID, RepositoryKey: target.Repository.Key, ProviderResource: target.WorkItem.Resource}, target.WorkItem.ExternalID)
	if failed.Category != "" {
		return failed
	}
	content := "# " + document.Title + "\n\n" + document.Body
	if len(content) > workflow.MaxInstructionBytes {
		return workitem.Result{Status: completion.ValidationFailure, Category: "work_item_too_large"}
	}
	for i := range stages {
		for _, name := range stages[i].WorkItemInputs {
			stages[i].FrozenInputs = append(stages[i].FrozenInputs, workflow.FrozenInput{Name: name, Kind: "work-item", Source: "target", Required: true, Content: content, Digest: workflow.DigestText([]byte(content))})
		}
		stages[i].WorkItemInputs = nil
	}
	return workitem.Result{}
}
