// Package workflowcompiler compiles one immutable workflow stage and its
// bounded approved work plan into inspectable, non-dispatching proposals.
package workflowcompiler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/portableconfig"
	"github.com/rgomids/axiom/internal/runtimeapplication"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

var (
	ErrInvalidRequest    = errors.New("invalid stage compilation request")
	ErrRuntimeBlocked    = errors.New("stage runtime resolution blocked")
	ErrUnsupportedEffort = errors.New("requested reasoning effort is unsupported")
)

const maxAgents = 32

type RuntimeResolver interface {
	Preview(context.Context, string, runtimeapplication.Request) (runtimeapplication.Preview, error)
}

type InputBinding struct {
	InputID   string `json:"inputId"`
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
}

// Request contains immutable definition data, a bounded approved plan, and
// digest-bound references only. It has no process runner or dispatch port.
type Request struct {
	ProjectID       string
	RepositoryKey   string
	Workflow        workflowdefinition.Ref
	Definition      workflowdefinition.Document
	StageID         string
	Plan            executiongraph.ApprovedPlan
	InputBindings   []InputBinding
	ParentAuthority []executiongraph.Effect
}

type ExecutionInput struct {
	ProjectID         string                           `json:"projectId"`
	RepositoryKey     string                           `json:"repositoryKey"`
	AgentID           string                           `json:"agentId"`
	NodeKey           string                           `json:"nodeKey"`
	PlanRevision      string                           `json:"planRevision"`
	PlanDigest        string                           `json:"planDigest"`
	Role              string                           `json:"role"`
	Responsibilities  string                           `json:"responsibilities"`
	RuntimePreview    runtimeapplication.Preview       `json:"runtimePreview"`
	RequestedEffort   workflowdefinition.Effort        `json:"requestedEffort"`
	Controls          executiongraph.ExecutionControls `json:"controls"`
	Scope             executiongraph.Scope             `json:"scope"`
	Effects           []executiongraph.Effect          `json:"effects"`
	Workflow          workflowdefinition.Ref           `json:"workflow"`
	StageID           string                           `json:"stageId"`
	Purpose           string                           `json:"purpose"`
	Instructions      string                           `json:"instructions"`
	Inputs            []InputBinding                   `json:"inputs"`
	ExpectedOutputs   []workflowdefinition.Output      `json:"expectedOutputs"`
	Criteria          []workflowdefinition.Criterion   `json:"completionCriteria"`
	Validators        []workflowdefinition.Validator   `json:"validators"`
	HumanGates        []workflowdefinition.Gate        `json:"humanGates"`
	FailurePolicy     workflowdefinition.FailurePolicy `json:"failurePolicy"`
	Dependencies      []string                         `json:"dependencyOutputs"`
	DependencyOutputs []DependencyOutput               `json:"dependencyOutputReferences"`
}

type DependencyOutput struct {
	AgentID                string   `json:"agentId"`
	Outputs                []string `json:"outputIds"`
	RequireValidatedDigest bool     `json:"requireValidatedDigest"`
}

type ValidationResult struct {
	Check  string `json:"check"`
	Status string `json:"status"`
}

type ResolvedAgent struct {
	AgentID          string                           `json:"agentId"`
	Role             string                           `json:"role"`
	Responsibilities string                           `json:"responsibilities"`
	Choice           runtimeprofile.Choice            `json:"runtimeProfile"`
	Capability       executiongraph.CapabilityRequest `json:"capability"`
	Controls         executiongraph.ExecutionControls `json:"controls"`
	Scope            executiongraph.Scope             `json:"scope"`
	Effects          []executiongraph.Effect          `json:"effects"`
	Input            ExecutionInput                   `json:"executionInput"`
	InputReference   StageInputReference              `json:"stageInputRef"`
}

// StageInputReference identifies canonical input bytes to publish through the
// existing artifact store before envelope admission. Compilation writes nothing.
type StageInputReference struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
}

type Result struct {
	FormatVersion    int                      `json:"formatVersion"`
	Workflow         workflowdefinition.Ref   `json:"workflow"`
	StageID          string                   `json:"stageId"`
	ExecutionKind    string                   `json:"executionKind"`
	Mode             string                   `json:"mode"`
	Concurrency      int                      `json:"concurrency"`
	Ordering         [][]string               `json:"executionOrdering"`
	Authority        []executiongraph.Effect  `json:"requiredAuthority"`
	AuthorityCeiling []executiongraph.Effect  `json:"authorityCeiling"`
	Validation       []ValidationResult       `json:"validation"`
	Proposal         *executiongraph.Proposal `json:"graphProposal,omitempty"`
	Agents           []ResolvedAgent          `json:"agents"`
	PlanRevision     string                   `json:"planRevision"`
	PlanDigest       string                   `json:"planDigest"`
	Digest           string                   `json:"digest"`
}

type Compiler struct {
	runtimes     RuntimeResolver
	capabilities executiongraph.CapabilityValidator
}

func New(runtimes RuntimeResolver, capabilities executiongraph.CapabilityValidator) Compiler {
	return Compiler{runtimes: runtimes, capabilities: capabilities}
}

func (c Compiler) Compile(ctx context.Context, request Request) (Result, error) {
	stage, err := validateRequest(request)
	if err != nil {
		return Result{}, err
	}
	if c.runtimes == nil || c.capabilities == nil {
		return Result{}, ErrRuntimeBlocked
	}
	agents := make([]ResolvedAgent, 0, len(stage.Agents))
	workByID := make(map[string]executiongraph.WorkUnit, len(request.Plan.Work))
	for _, work := range request.Plan.Work {
		workByID[work.Key] = work
	}
	units := make([]executiongraph.WorkUnit, 0, len(stage.Agents))
	for _, agent := range stage.Agents {
		work := workByID[agent.ID]
		resolved, err := c.resolveAgent(ctx, request, stage, agent)
		if err != nil {
			return Result{}, err
		}
		if len(agents) > 0 {
			prior := agents[0].Input.RuntimePreview
			if prior.ProjectDigest != resolved.ProjectDigest || prior.ConfigurationDigest != resolved.ConfigurationDigest || prior.ConfigurationRevision != resolved.ConfigurationRevision {
				return Result{}, fmt.Errorf("%w: policy changed during compilation", ErrRuntimeBlocked)
			}
		}
		for _, priorAgent := range agents {
			prior := priorAgent.Input.RuntimePreview
			if prior.Choice.RuntimeID != resolved.Choice.RuntimeID {
				continue
			}
			if prior.Choice.ObservationRevision != resolved.Choice.ObservationRevision || prior.Choice.RuntimeVersion != resolved.Choice.RuntimeVersion || prior.Choice.ExecutableDigest != resolved.Choice.ExecutableDigest || prior.Request.RuntimeID == resolved.Request.RuntimeID && prior.ObservationDigest != resolved.ObservationDigest {
				return Result{}, fmt.Errorf("%w: Runtime changed during compilation", ErrRuntimeBlocked)
			}
		}
		unit := compiledUnit(request, stage, agent, work)
		input := executionInput(request, stage, agent)
		input.ProjectID, input.RepositoryKey = request.ProjectID, request.RepositoryKey
		input.AgentID, input.NodeKey = agent.ID, unit.Key
		input.PlanRevision, input.PlanDigest = request.Plan.PlanRevision, request.Plan.PlanDigest
		input.Role, input.Responsibilities = agent.Role, agent.Responsibilities
		input.RuntimePreview, input.RequestedEffort = resolved, agent.Effort
		input.Controls, input.Scope, input.Effects = unit.Controls, unit.Scope, unit.Effects
		inputReference, err := stageInputReference(input)
		if err != nil {
			return Result{}, err
		}
		unit.Inputs = sortedUnique(append(unit.Inputs, inputReference.ID))
		agents = append(agents, ResolvedAgent{AgentID: agent.ID, Role: agent.Role, Responsibilities: agent.Responsibilities, Choice: *resolved.Choice, Capability: unit.Capability, Controls: unit.Controls, Scope: unit.Scope, Effects: unit.Effects, Input: input, InputReference: inputReference})
		units = append(units, unit)
	}
	var requiredEffects []executiongraph.Effect
	for _, unit := range units {
		for _, effect := range unit.Effects {
			if !containsEffect(requiredEffects, effect) {
				requiredEffects = append(requiredEffects, effect)
			}
		}
	}
	result := Result{FormatVersion: 1, Workflow: request.Workflow, StageID: stage.ID, ExecutionKind: "graph", Mode: stage.Mode, Concurrency: stage.Concurrency, Authority: sortedEffects(requiredEffects), AuthorityCeiling: sortedEffects(request.ParentAuthority), Validation: []ValidationResult{{Check: "workflow-revision", Status: "passed"}, {Check: "stage-contract", Status: "passed"}, {Check: "bounded-plan", Status: "passed"}, {Check: "input-references", Status: "passed"}, {Check: "runtime-profile-resolution", Status: "passed"}, {Check: "effort-policy", Status: "passed"}, {Check: "authority-ceiling", Status: "passed"}, {Check: "topology-and-concurrency", Status: "passed"}}, Agents: agents, PlanRevision: request.Plan.PlanRevision, PlanDigest: request.Plan.PlanDigest}
	if len(stage.Agents) == 1 {
		result.ExecutionKind = "single"
		result.Ordering = [][]string{{units[0].Key}}
		plan := request.Plan
		plan.MaximumNodes = 1
		plan.Work = units
		if _, err := executiongraph.NewPlanner(c.capabilities).Propose(ctx, plan); err != nil {
			return Result{}, fmt.Errorf("validate single-agent plan: %w", err)
		}
	} else {
		applyConcurrencyLimit(units, stage, stage.Concurrency)
		plan := request.Plan
		plan.MaximumNodes = len(units)
		plan.Work = units
		proposal, err := executiongraph.NewPlanner(c.capabilities).Propose(ctx, plan)
		if err != nil {
			return Result{}, err
		}
		result.Proposal = &proposal
		result.Ordering = executionLayers(units, stage.Concurrency)
	}
	result.Digest, err = digest(result)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func validateRequest(request Request) (workflowdefinition.Stage, error) {
	if request.ProjectID == "" || request.RepositoryKey == "" || !request.Workflow.Valid() || request.Workflow != request.Definition.Ref(request.Workflow.Source) || request.Definition.Digest == "" || !request.Plan.Approved || request.Plan.PlanRevision == "" || !workflowdefinition.ValidDigest(request.Plan.PlanDigest) || request.Plan.MaximumNodes < 1 || request.Plan.MaximumNodes > maxAgents || len(request.Plan.Work) == 0 || len(request.Plan.Work) > maxAgents {
		return workflowdefinition.Stage{}, ErrInvalidRequest
	}
	var stage workflowdefinition.Stage
	for _, candidate := range request.Definition.Definition.Stages {
		if candidate.ID == request.StageID {
			stage = candidate
			break
		}
	}
	if stage.ID == "" || len(stage.Agents) == 0 || len(stage.Agents) > maxAgents || stage.Concurrency < 1 || stage.Concurrency > len(stage.Agents) || !request.Plan.Approved || request.Plan.MaximumNodes > maxAgents || len(request.Plan.Work) != len(stage.Agents) || len(request.Plan.Work) > request.Plan.MaximumNodes {
		return workflowdefinition.Stage{}, ErrInvalidRequest
	}
	canonical, diagnostics := workflowdefinition.Encode(request.Definition.Definition)
	if len(diagnostics) != 0 || canonical.Digest != request.Definition.Digest || string(canonical.Canonical) != string(request.Definition.Canonical) {
		return workflowdefinition.Stage{}, ErrInvalidRequest
	}
	if request.Workflow.Source == "builtin" {
		builtin := workflowdefinition.Builtin()
		if request.Workflow != builtin.Ref("builtin") || canonical.Digest != builtin.Digest {
			return workflowdefinition.Stage{}, ErrInvalidRequest
		}
	}
	if err := validateBindings(stage, request.InputBindings); err != nil {
		return workflowdefinition.Stage{}, err
	}
	work := map[string]executiongraph.WorkUnit{}
	for _, unit := range request.Plan.Work {
		if unit.Key == "" || !validEffects(unit.Effects, request.ParentAuthority) || unit.Controls.Timeout <= 0 || unit.Controls.MaximumAttempts == 0 || unit.Scope.ProjectID != request.ProjectID || unit.Scope.RepositoryKey != request.RepositoryKey {
			return workflowdefinition.Stage{}, ErrInvalidRequest
		}
		if _, exists := work[unit.Key]; exists {
			return workflowdefinition.Stage{}, ErrInvalidRequest
		}
		work[unit.Key] = unit
	}
	for _, agent := range stage.Agents {
		unit, exists := work[agent.ID]
		if !exists || !workMatchesAgent(unit, agent) || unit.Controls.Timeout > time.Duration(agent.TimeoutSeconds)*time.Second || unit.Controls.MaximumAttempts > uint32(agent.MaximumAttempts) || unit.Controls.ReasoningEffort != "" || unit.Optional || !withinEffectCeilings(unit.Effects, agent.EffectCeilings) || !withinScope(unit.Effects, unit.Scope.Paths) {
			return workflowdefinition.Stage{}, ErrInvalidRequest
		}
	}
	return stage, nil
}

func validateBindings(stage workflowdefinition.Stage, bindings []InputBinding) error {
	byID := map[string]InputBinding{}
	for _, binding := range bindings {
		if binding.InputID == "" || binding.Reference == "" || !workflowdefinition.ValidDigest(binding.Digest) {
			return ErrInvalidRequest
		}
		if _, exists := byID[binding.InputID]; exists {
			return ErrInvalidRequest
		}
		byID[binding.InputID] = binding
	}
	for _, input := range stage.Inputs {
		binding, exists := byID[input.ID]
		if input.Required && !exists || exists && binding.Reference != input.Source {
			return ErrInvalidRequest
		}
		if exists && input.Kind != "stage-output" && input.Kind != "work-item" && input.Kind != "project-context" && input.Kind != "artifact" {
			return ErrInvalidRequest
		}
	}
	for id := range byID {
		found := false
		for _, input := range stage.Inputs {
			found = found || input.ID == id
		}
		if !found {
			return ErrInvalidRequest
		}
	}
	return nil
}

func (c Compiler) resolveAgent(ctx context.Context, request Request, stage workflowdefinition.Stage, agent workflowdefinition.Agent) (runtimeapplication.Preview, error) {
	constraints := append([]string(nil), agent.RuntimeConstraints...)
	sort.Strings(constraints)
	runtimeIDs := constraints
	if len(runtimeIDs) == 0 {
		runtimeIDs = []string{""}
	}
	var matches []runtimeapplication.Preview
	for _, runtimeID := range runtimeIDs {
		selection := runtimeapplication.Request{Role: agent.Role, Complexity: agent.Complexity, Capabilities: sortedUnique(append([]string(nil), agent.Capabilities...)), RuntimeID: runtimeID, ModelProfileID: profileID(agent.ProfileRef)}
		preview, err := c.runtimes.Preview(ctx, request.ProjectID, selection)
		if err == nil && previewMatches(preview, request.ProjectID, selection) && (len(constraints) == 0 || contains(constraints, preview.Choice.RuntimeID)) {
			matches = append(matches, preview)
		}
	}
	if len(matches) != 1 {
		return runtimeapplication.Preview{}, fmt.Errorf("%w: %s", ErrRuntimeBlocked, agent.ID)
	}
	if agent.Effort.Mode == "explicit" {
		choice := matches[0].Choice
		if choice.RuntimeVersion == "" {
			return runtimeapplication.Preview{}, ErrUnsupportedEffort
		}
		capabilities := append(append([]string(nil), agent.Capabilities...), "reasoning-effort-"+agent.Effort.Value)
		selection := runtimeapplication.Request{Role: agent.Role, Complexity: agent.Complexity, Capabilities: sortedUnique(capabilities), RuntimeID: choice.RuntimeID, ModelProfileID: choice.ModelProfileID}
		preview, err := c.runtimes.Preview(ctx, request.ProjectID, selection)
		if err != nil || !previewMatches(preview, request.ProjectID, selection) || preview.ConfigurationDigest != matches[0].ConfigurationDigest || preview.ObservationDigest != matches[0].ObservationDigest || preview.ProjectDigest != matches[0].ProjectDigest || preview.Choice.RuntimeVersion != choice.RuntimeVersion || preview.Choice.ExecutableDigest != choice.ExecutableDigest {
			return runtimeapplication.Preview{}, ErrUnsupportedEffort
		}
		matches[0] = preview
	}
	return matches[0], nil
}

func compiledUnit(request Request, stage workflowdefinition.Stage, agent workflowdefinition.Agent, work executiongraph.WorkUnit) executiongraph.WorkUnit {
	inputs := make([]string, 0, len(agent.Inputs)+len(agent.DependsOn)*2)
	for _, id := range agent.Inputs {
		inputs = append(inputs, "input-"+id)
	}
	for _, dependency := range agent.DependsOn {
		for _, prior := range stage.Agents {
			if prior.ID == dependency {
				for _, output := range prior.Outputs {
					inputs = append(inputs, "output-"+stage.ID+"-"+dependency+"-"+output)
				}
			}
		}
	}
	outputs := make([]string, 0, len(agent.Outputs))
	for _, output := range agent.Outputs {
		outputs = append(outputs, "output-"+stage.ID+"-"+agent.ID+"-"+output)
	}
	controls := work.Controls
	if int(controls.Timeout.Seconds()) > agent.TimeoutSeconds {
		controls.Timeout = time.Duration(agent.TimeoutSeconds) * time.Second
	}
	if controls.MaximumAttempts > uint32(agent.MaximumAttempts) {
		controls.MaximumAttempts = uint32(agent.MaximumAttempts)
	}
	if agent.Effort.Mode == "explicit" {
		controls.ReasoningEffort = agent.Effort.Value
	}
	capabilities := append([]string(nil), agent.Capabilities...)
	if agent.Effort.Mode == "explicit" {
		capabilities = append(capabilities, "reasoning-effort-"+agent.Effort.Value)
	}
	dependencies := make([]string, 0, len(agent.DependsOn))
	for _, dependency := range agent.DependsOn {
		dependencies = append(dependencies, stage.ID+":"+dependency)
	}
	scope := work.Scope
	scope.Paths = sortedUnique(append([]string(nil), scope.Paths...))
	return executiongraph.WorkUnit{Key: stage.ID + ":" + agent.ID, Capability: executiongraph.CapabilityRequest{Role: agent.Role, Complexity: agent.Complexity, Capabilities: sortedUnique(capabilities)}, Dependencies: dependencies, Inputs: sortedUnique(inputs), Outputs: sortedUnique(outputs), Scope: scope, Effects: sortedEffects(work.Effects), ValidationOwner: agent.ValidationOwner, IntegrationOwner: agent.IntegrationOwner, Controls: controls}
}

func applyConcurrencyLimit(units []executiongraph.WorkUnit, stage workflowdefinition.Stage, concurrency int) {
	if stage.Mode == "sequential" || concurrency >= len(units) {
		return
	}
	ids := topologicalIDs(stage.Agents)
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		for _, agent := range stage.Agents {
			if agent.ID == id && !agent.IntegrationOwner {
				filtered = append(filtered, stage.ID+":"+id)
			}
		}
	}
	ids = filtered
	for index := concurrency; index < len(ids); index++ {
		for unit := range units {
			if units[unit].Key == ids[index] && !contains(units[unit].Dependencies, ids[index-concurrency]) {
				units[unit].Dependencies = append(units[unit].Dependencies, ids[index-concurrency])
				units[unit].Dependencies = sortedUnique(units[unit].Dependencies)
			}
		}
	}
}

func executionLayers(units []executiongraph.WorkUnit, concurrency int) [][]string {
	remaining := map[string]bool{}
	for _, unit := range units {
		remaining[unit.Key] = true
	}
	var layers [][]string
	for len(remaining) > 0 {
		var ready []string
		for _, unit := range units {
			if !remaining[unit.Key] {
				continue
			}
			ok := true
			for _, dependency := range unit.Dependencies {
				if remaining[dependency] {
					ok = false
					break
				}
			}
			if ok {
				ready = append(ready, unit.Key)
			}
		}
		if len(ready) == 0 {
			return nil
		}
		sort.Strings(ready)
		if len(ready) > concurrency {
			ready = ready[:concurrency]
		}
		layers = append(layers, ready)
		for _, key := range ready {
			delete(remaining, key)
		}
	}
	return layers
}

func executionInput(request Request, stage workflowdefinition.Stage, agent workflowdefinition.Agent) ExecutionInput {
	byID := map[string]InputBinding{}
	for _, binding := range request.InputBindings {
		byID[binding.InputID] = binding
	}
	inputs := make([]InputBinding, 0, len(agent.Inputs))
	for _, id := range agent.Inputs {
		if binding, ok := byID[id]; ok {
			inputs = append(inputs, binding)
		}
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].InputID < inputs[j].InputID })
	outputs := make([]workflowdefinition.Output, 0, len(agent.Outputs))
	for _, agentOutput := range agent.Outputs {
		for _, output := range stage.Outputs {
			if output.ID == agentOutput {
				outputs = append(outputs, output)
				break
			}
		}
	}
	criteria := append([]workflowdefinition.Criterion(nil), stage.CompletionCriteria...)
	validators := append([]workflowdefinition.Validator(nil), stage.Validators...)
	gates := append([]workflowdefinition.Gate(nil), stage.HumanGates...)
	dependencies := make([]string, 0, len(agent.DependsOn))
	dependencyOutputs := make([]DependencyOutput, 0, len(agent.DependsOn))
	for _, dependency := range agent.DependsOn {
		dependencies = append(dependencies, "agent-"+dependency)
		for _, prior := range stage.Agents {
			if prior.ID == dependency {
				dependencyOutputs = append(dependencyOutputs, DependencyOutput{AgentID: dependency, Outputs: append([]string(nil), prior.Outputs...), RequireValidatedDigest: true})
			}
		}
	}
	sort.Strings(dependencies)
	sort.Slice(dependencyOutputs, func(i, j int) bool { return dependencyOutputs[i].AgentID < dependencyOutputs[j].AgentID })
	return ExecutionInput{Workflow: request.Workflow, StageID: stage.ID, Purpose: stage.Purpose, Instructions: stage.Instructions, Inputs: inputs, ExpectedOutputs: outputs, Criteria: criteria, Validators: validators, HumanGates: gates, FailurePolicy: stage.FailurePolicy, Dependencies: dependencies, DependencyOutputs: dependencyOutputs}
}

func digest(result Result) (string, error) {
	result.Digest = ""
	wire, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	if len(wire) > executiongraph.MaxGraphBytes {
		return "", ErrInvalidRequest
	}
	hash := sha256.Sum256(wire)
	return hex.EncodeToString(hash[:]), nil
}

func stageInputReference(input ExecutionInput) (StageInputReference, error) {
	wire, err := json.Marshal(input)
	if err != nil || len(wire) > executiongraph.MaxGraphBytes {
		return StageInputReference{}, ErrInvalidRequest
	}
	hash := sha256.Sum256(wire)
	digest := hex.EncodeToString(hash[:])
	return StageInputReference{ID: "stage-input-" + digest, Digest: digest}, nil
}

func containsEffect(effects []executiongraph.Effect, expected executiongraph.Effect) bool {
	for _, effect := range effects {
		if effect == expected {
			return true
		}
	}
	return false
}

func previewMatches(preview runtimeapplication.Preview, projectID string, request runtimeapplication.Request) bool {
	choice := preview.Choice
	if choice == nil || preview.Blocker != nil || preview.ProjectID != projectID || preview.Request.Role != request.Role || preview.Request.Complexity != request.Complexity || preview.Request.RuntimeID != request.RuntimeID || preview.Request.ModelProfileID != request.ModelProfileID || !sameStrings(preview.Request.Capabilities, request.Capabilities) || request.RuntimeID != "" && choice.RuntimeID != request.RuntimeID || request.ModelProfileID != "" && choice.ModelProfileID != request.ModelProfileID || !sameStrings(choice.Capabilities, request.Capabilities) || !workflowdefinition.ValidDigest(preview.ProjectDigest) || !workflowdefinition.ValidDigest(preview.ConfigurationDigest) || !workflowdefinition.ValidDigest(preview.ObservationDigest) || !workflowdefinition.ValidDigest(choice.ExecutableDigest) || choice.ObservationRevision == 0 {
		return false
	}
	return choice.RuntimeID == choice.Adapter && choice.RuntimeID != "" && choice.ModelProfileID != "" && choice.Model != "" && choice.ConfigurationRevision > 0 && choice.ConfigurationRevision == preview.ConfigurationRevision
}

func workMatchesAgent(work executiongraph.WorkUnit, agent workflowdefinition.Agent) bool {
	if work.Capability.Role != "" || work.Capability.Complexity != "" || len(work.Capability.Capabilities) > 0 {
		if work.Capability.Role != agent.Role || work.Capability.Complexity != agent.Complexity || !sameStrings(work.Capability.Capabilities, agent.Capabilities) {
			return false
		}
	}
	if len(work.Dependencies) > 0 && !sameStrings(work.Dependencies, agent.DependsOn) || len(work.Inputs) > 0 && !sameStrings(work.Inputs, agent.Inputs) || len(work.Outputs) > 0 && !sameStrings(work.Outputs, agent.Outputs) {
		return false
	}
	return (!work.ValidationOwner || agent.ValidationOwner) && (!work.IntegrationOwner || agent.IntegrationOwner)
}

func sameStrings(left, right []string) bool {
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validEffects(effects, authority []executiongraph.Effect) bool {
	allowed := map[string]bool{}
	for _, effect := range authority {
		if portableconfig.ContainsSecretBearingValue(effect.Target) {
			return false
		}
		allowed[effect.Kind+"\x00"+effect.Target] = true
	}
	for _, effect := range effects {
		if !allowed[effect.Kind+"\x00"+effect.Target] {
			return false
		}
	}
	return true
}
func sortedEffects(effects []executiongraph.Effect) []executiongraph.Effect {
	result := append([]executiongraph.Effect(nil), effects...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Kind+"\x00"+result[i].Target < result[j].Kind+"\x00"+result[j].Target
	})
	return result
}
func withinEffectCeilings(effects []executiongraph.Effect, ceilings []string) bool {
	allowed := map[string]bool{}
	for _, ceiling := range ceilings {
		allowed[ceiling] = true
	}
	for _, effect := range effects {
		if !allowed[effect.Kind] {
			return false
		}
	}
	return true
}
func withinScope(effects []executiongraph.Effect, paths []string) bool {
	for _, effect := range effects {
		if effect.Kind != "repository-write" && effect.Kind != "integration" {
			continue
		}
		target := path.Clean(effect.Target)
		if target == "." || path.IsAbs(target) || target != effect.Target || strings.HasPrefix(target, "../") {
			return false
		}
		covered := false
		for _, scope := range paths {
			cleanScope := path.Clean(scope)
			if target == cleanScope || strings.HasPrefix(target, cleanScope+"/") {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}
func topologicalIDs(agents []workflowdefinition.Agent) []string {
	remaining := map[string]workflowdefinition.Agent{}
	for _, agent := range agents {
		remaining[agent.ID] = agent
	}
	var ordered []string
	for len(remaining) > 0 {
		var ready []string
		for id, agent := range remaining {
			readyNow := true
			for _, dependency := range agent.DependsOn {
				if _, pending := remaining[dependency]; pending {
					readyNow = false
					break
				}
			}
			if readyNow {
				ready = append(ready, id)
			}
		}
		if len(ready) == 0 {
			return ordered
		}
		sort.Strings(ready)
		for _, id := range ready {
			ordered = append(ordered, id)
			delete(remaining, id)
		}
	}
	return ordered
}
func profileID(ref string) string {
	if ref == "policy-default" {
		return ""
	}
	return ref
}
func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func sortedUnique(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}
