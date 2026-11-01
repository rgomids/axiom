package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/stageexecution"
	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
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
)

func (s lifecycleService) executionStage(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	// A human disposition never records without both its subject and explicit
	// human authority; only the built-in minimal Workflow can override the
	// Project selection. Checked before any read or write.
	if in.Operation == "execution-run" && (in.ApproveGate != "" || in.ReviewStage != "") != in.AuthorizeHuman || in.Operation == "reconcile" && (in.Abandon != "") != in.AuthorizeHuman {
		return s.stageFailure(completion.DeniedAuthority, "human_authority_required", "A human disposition needs its exact subject and --authorize-human", "Repeat with both --approve-gate/--abandon and --authorize-human after real human review", nil)
	}
	if in.AuthorizeHuman && (os.Getenv("AXIOM_ATTEMPT_ID") != "" || os.Getenv("AXIOM_EXECUTION_ID") != "") {
		return s.stageFailure(completion.DeniedAuthority, "human_authority_required", "A Runtime child cannot record human approval", "Run this operation from the human operator context", nil)
	}
	if in.ReviewStage != "" && (in.ResultDigest == "" || in.ReviewEvidence == "" || in.ApproveGate != "") || in.ReviewStage == "" && (in.ResultDigest != "" || in.ReviewEvidence != "") {
		return s.stageFailure(completion.ValidationFailure, "invalid_review", "Review requires an exact stage, result digest and retained Evidence", "Inspect status and supply --review-stage --result-digest --evidence --authorize-human", nil)
	}
	if (in.Operation == "cancel" || in.Operation == "reconcile" && in.Abandon == "") && !in.AuthorizeLocal {
		return s.stageFailure(completion.DeniedAuthority, "authority_required", "This local Execution mutation needs explicit authority", "Repeat with --authorize-local", nil)
	}
	if in.Workflow != "" && in.Workflow != "minimal" {
		return s.stageFailure(completion.ValidationFailure, "invalid_selection", "Only the built-in minimal Workflow overrides the Project selection", "Use --workflow minimal or select the Project Workflow with project workflow select", nil)
	}
	operation := projectapp.AdmitExecutionAdvance
	if in.Operation == "status" {
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
	case "execution-run":
		ctx, stop := interruptibleRun(ctx)
		defer stop()
		return c.run(ctx, in)
	case "status":
		x, err := store.Load(ctx, resolved.Project.ID, in.Execution)
		if err != nil {
			return c.failure(err, nil)
		}
		return c.outcome("Execution status", x, nil, true)
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
		return c.outcome("Execution reconciled without signalling or replay", x, facts, true)
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
		if len(selections) != 0 || len(in.Inputs) != 0 || in.Workflow != "" || in.Repository != "" || in.WorkItem != "" {
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
		return c.outcome("Execution progressed", x, nil, false)
	}
	if in.ApproveGate != "" || in.ReviewStage != "" {
		return c.failure(errSelectionSyntax, nil)
	}
	provider, resource, externalID, _ := cli.ParseWorkItemSelector(in.WorkItem)
	target, failed := c.s.workflows.ResolveLinked(ctx, workflowTarget(cli.WorkflowInput{Project: c.project.ID, Repository: in.Repository, Provider: provider, ProviderRepository: resource, ExternalID: externalID}))
	if failed.Category != "" {
		return c.s.stageFailure(completionStatus(failed.Status), failed.Category, "Linked Work Item could not be resolved", "Link the existing Work Item to this Project Repository with work-item select; nothing was written", nil)
	}
	request := stageexecution.StartRequest{ProjectID: c.project.ID, WorkItem: stageexecution.LinkedWorkItem{Provider: target.WorkItem.Provider, ID: workItemToken(target.WorkItem), RepositoryKey: target.Repository.Key}, Selections: selections}
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
	return c.outcome("Execution started", x, nil, false)
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
	return c.outcome("Retry attempt recorded", x, nil, false)
}

func (c executionComposition) engine(launcher stageexecution.Launcher) stageexecution.Engine {
	return stageexecution.Engine{Store: c.store, Runner: executiongraph.OSProcessRunner{}, Launcher: launcher, Resolve: c.resolve, NewID: allocateStageID}
}

// launcher binds the machine-local workspace, private output root and Axiom
// path. Outputs live under the state root, never inside the repository.
func (c executionComposition) launcher(executionID, repositoryKey string) (stageexecution.RuntimeLauncher, error) {
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
	// Created through the state root so no path component can redirect it
	// outside; Build still refuses any symlinked component before spawning.
	relative := filepath.Join("executions", "v2-attempt-outputs", c.project.ID, executionID)
	state, err := os.OpenRoot(c.s.stateRoot)
	if err != nil {
		return stageexecution.RuntimeLauncher{}, err
	}
	defer state.Close()
	if err := state.MkdirAll(relative, 0o700); err != nil {
		return stageexecution.RuntimeLauncher{}, err
	}
	inbox := filepath.Join(c.s.stateRoot, relative)
	artifacts, err := local.NewArtifactStore(c.s.stateRoot)
	if err != nil {
		return stageexecution.RuntimeLauncher{}, err
	}
	return stageexecution.RuntimeLauncher{Workspace: workspace, InboxRoot: inbox, AxiomPath: axiom, Environment: os.Environ(), ValidateOutput: local.NewWorkflowReferenceValidator(artifacts).ValidateExecutionOutput}, nil
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
// new selection of one authorized retry. The current Project must still bind
// the Execution's repository with the same identity.
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
	bound := ""
	for _, r := range x.Repositories {
		if r.Key == x.WorkItem.RepositoryKey {
			bound = r.Identity
		}
	}
	current := ""
	for _, r := range c.project.Repositories {
		if r.Key == x.WorkItem.RepositoryKey {
			current = repositoryIdentity(r)
		}
	}
	if current == "" || current != bound {
		return workflow.Invocation{}, workflow.AttemptAuthority{}, errRepositoryBinding
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
	return invocation, workflow.AttemptAuthority{RepositoryKey: x.WorkItem.RepositoryKey, Effects: effects, Actor: executionActor("user:")}, nil
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
		for _, o := range s.Outputs {
			if o.Required {
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
			agent := workflow.AgentSpec{ID: a.ID, OutputScope: true, DependsOn: append([]string(nil), a.DependsOn...), Outputs: append([]string(nil), a.Outputs...), TimeoutSeconds: int64(a.TimeoutSeconds), MaximumAttempts: a.MaximumAttempts, Instructions: strings.TrimSpace(s.Instructions + "\n\n" + a.Responsibilities), Runtimes: append([]string(nil), a.RuntimeConstraints...)}
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
		stages = append(stages, spec)
		previous = s.ID
	}
	if _, err := workflow.NewExecution(workflow.NewExecutionInput{ExecutionID: "00000000-0000-0000-0000-000000000001", ProjectID: "validation", WorkItem: workflow.WorkItemLink{Provider: "local", ID: "validation", RepositoryKey: "repo"}, Repositories: []workflow.RepositoryBinding{{Key: "repo", Identity: "validation"}}, WorkflowID: d.WorkflowID, Revision: revision, Stages: stages, Now: time.Unix(1, 0)}); err != nil {
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
	ExecutionID      string                         `json:"executionId"`
	ProjectID        string                         `json:"projectId"`
	WorkItem         workflow.WorkItemLink          `json:"workItem"`
	WorkflowID       string                         `json:"workflowId"`
	WorkflowRevision string                         `json:"workflowRevision"`
	WorkflowDigest   string                         `json:"workflowDigest"`
	Revision         uint64                         `json:"revision"`
	RunnerGeneration uint64                         `json:"runnerGeneration"`
	Status           workflow.RunStatus             `json:"status"`
	Cancel           *workflow.CancelIntent         `json:"cancel,omitempty"`
	Stages           []executionStageView           `json:"stages"`
	Reconciliation   []stageexecution.ReconcileFact `json:"reconciliation,omitempty"`
	NextAction       string                         `json:"nextAction"`
}

func executionPayload(x workflow.Execution, facts []stageexecution.ReconcileFact) executionView {
	v := executionView{ExecutionID: x.ExecutionID, ProjectID: x.ProjectID, WorkItem: x.WorkItem, WorkflowID: x.Workflow.ID, WorkflowRevision: x.Workflow.Revision, WorkflowDigest: x.Workflow.Digest,
		Revision: x.Revision, RunnerGeneration: x.RunnerGeneration, Status: x.Status, Cancel: x.Cancel, Stages: []executionStageView{}, Reconciliation: facts, NextAction: executionNext(x)}
	for _, s := range x.Stages {
		stage := executionStageView{StageID: s.StageID, Status: s.Status, Gate: s.Gate, BeforeApproval: s.BeforeApproval, Review: s.Review, Tasks: []executionTaskView{}}
		stage.ResultDigest, _ = x.StageResultDigest(s.StageID)
		for _, t := range s.Tasks {
			task := executionTaskView{AgentID: t.AgentID, Attempts: []executionAttemptView{}}
			for _, a := range t.Attempts {
				view := executionAttemptView{WorkspaceDisposition: a.WorkspaceDisposition, AttemptID: a.AttemptID, Number: a.Number, RunnerGeneration: a.RunnerGeneration, Status: a.Status, Runtime: a.Invocation.Runtime, Model: a.Invocation.Model, Effort: a.Invocation.Effort,
					Profile: a.Invocation.Profile, ProfileDigest: a.Invocation.ProfileDigest, InstructionsDigest: a.Invocation.InstructionsDigest, Provenance: a.Invocation.Provenance,
					RepositoryKey: a.Authority.RepositoryKey, AuthorizedEffects: append([]string{}, a.Authority.Effects...), ExitCode: a.ExitCode, Diagnostic: a.Diagnostic, Effects: a.Effects, Artifacts: a.Artifacts, Resolution: a.Resolution}
				if a.Process != nil {
					view.Process = &executionProcessView{PID: a.Process.PID, StartIdentity: a.Process.StartIdentity}
				}
				task.Attempts = append(task.Attempts, view)
			}
			stage.Tasks = append(stage.Tasks, task)
		}
		v.Stages = append(v.Stages, stage)
	}
	return v
}

// executionNext is the smallest safe next action for the persisted facts.
// Technical completion is never human acceptance.
func executionNext(x workflow.Execution) string {
	selector := " --execution " + x.ExecutionID
	switch x.Status {
	case workflow.RunCompleted:
		return "Inspect the recorded artifacts; technical completion is not human acceptance"
	case workflow.RunAwaitingGate:
		for _, s := range x.Stages {
			if s.Status == workflow.StageAwaitingGate {
				for _, spec := range x.Workflow.Stages {
					if spec.ID == s.StageID && spec.ReviewRequired && s.Review == nil {
						if digest, err := x.StageResultDigest(s.StageID); err == nil {
							return "After real human review: axiom workflow run" + selector + " --review-stage " + s.StageID + " --result-digest " + digest + " --evidence <retained-evidence-id> --authorize-human --authorize-external"
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

func (c executionComposition) outcome(message string, x workflow.Execution, facts []stageexecution.ReconcileFact, read bool) cli.WorkflowStageResponse {
	payload := executionPayload(x, facts)
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
		{errSelectionSyntax, completion.ValidationFailure, "invalid_selection", "Use --select [<stage-id>/]<agent-id>.<runtime|model|effort|instructions|profile>=<value> naming a Workflow agent"},
		{errSelectionsFrozen, completion.ValidationFailure, "selections_frozen", "Start selections are immutable; change them only with stage retry --new-selection --select ..."},
		{runtimeapplication.ErrInvocationConflict, completion.ValidationFailure, "invocation_selection_conflict", "Give each agent field one value per level; nothing was written"},
		{runtimeapplication.ErrInvocationMissing, completion.ValidationFailure, "invocation_selection_missing", "Select runtime, model and effort (or runtime-default) explicitly, or an existing Profile; nothing is inferred"},
		{runtimeapplication.ErrRuntimeNotPermitted, completion.DeniedAuthority, "runtime_not_permitted", "Select a Runtime allowed by the Workflow agent and local bindings"},
		{runtimeapplication.ErrRuntimeNotConfigured, completion.ValidationFailure, "runtime_not_configured", "Create and enable the machine-local Runtime binding (axiom runtime binding create) with its executable on PATH"},
		{errWorkflowUnmappable, completion.ValidationFailure, "workflow_unmappable", "Select a Workflow with supported identifiers and controls; configured validators need exact implementation before dispatch"},
		{errRepositoryBinding, completion.DeniedAuthority, "repository_binding_changed", "The Project no longer binds this Execution's repository identity; start a new Execution"},
		{errExecutionProjectMissing, completion.ValidationFailure, "execution_not_found", "Use the exact Execution identity returned by workflow run"},
		{stageexecution.ErrWorkItemLinkMismatch, completion.ValidationFailure, "work_item_link_mismatch", "Link the Work Item to this Project Repository; nothing was written"},
		{workflow.ErrLegacyExecution, completion.ValidationFailure, "execution_legacy_format_refused", "Old Executions cannot resume after the breaking cutover; start a new Execution with workflow run"},
		{workflow.ErrRunnerHeld, completion.RetryableFailure, "execution_runner_active", "Another runner holds this Execution; inspect stage status and wait or cancel"},
		{workflow.ErrStaleRunner, completion.DeniedAuthority, "execution_stale_runner", "A newer runner generation owns this Execution; inspect stage status"},
		{workflow.ErrUnknownOutstanding, completion.Failure, "recovery_required", "Effects are unknown and never replayed; reconcile and record a human disposition first"},
		{workflow.ErrCancelled, completion.Interrupted, "execution_cancelled", "The Execution is cancelled; start a new Execution"},
		{workflow.ErrUnauthorizedEffect, completion.DeniedAuthority, "authority_denied", "Keep the attempt within the Workflow agent effect and Runtime restrictions"},
		{workflow.ErrWorkspaceDispositionRequired, completion.DeniedAuthority, "workspace_disposition_required", "Inspect and preserve the prior workspace explicitly with --disposition preserve; no automatic reset is supported"},
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
		if !ok || key == "" || refs[key] != "" {
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
		return workflow.ErrTransition
	}
	store, err := local.NewArtifactStore(c.s.stateRoot)
	if err != nil {
		return err
	}
	a, err := store.Read(ctx, in.ReviewEvidence)
	if err != nil || a.ExecutionID != x.ExecutionID || a.Category != "evidence" || a.Retention != "evidence" || a.Outcome != "success" {
		return workflow.ErrTransition
	}
	for kind, value := range map[string]string{"workflow-stage": in.ReviewStage, "workflow-definition": x.Workflow.Digest, "workflow-result": digest, "workflow-repository": x.WorkItem.RepositoryKey} {
		count := 0
		for _, r := range a.References {
			if r.Kind == kind {
				if r.Value != value {
					return workflow.ErrTransition
				}
				count++
			}
		}
		if count != 1 {
			return workflow.ErrTransition
		}
	}
	evidence := workflow.ArtifactRef{Name: "human-review", RetainedID: a.ID, RepositoryKey: x.WorkItem.RepositoryKey, Digest: "sha256:" + hex.EncodeToString(a.Digest[:]), Size: int64(a.ContentBytes)}
	_, err = c.store.UpdateFenced(ctx, x.ProjectID, x.ExecutionID, nil, func(current *workflow.Execution) error {
		return current.ReviewStage(in.ReviewStage, executionActor("human:"), digest, evidence, time.Now())
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
