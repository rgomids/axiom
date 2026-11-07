package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/rgomids/axiom/internal/strictjson"
)

// Execution is the one canonical durable state of a new workflow run
// (ADR-0037). Attempts, invocation snapshots, authority, process identity,
// effects, messages and artifact references are nested records; there are no
// child Executions, Graphs, Stage Plans, Validator Policies or side ledgers.
const (
	ExecutionKind          = "execution"
	ExecutionFormatVersion = 4

	MaxExecutionBytes   = 256 * 1024
	MaxExecutionStages  = 32
	MaxStageAgents      = 32
	MaxTaskAttempts     = 10
	MaxAttemptEffects   = 32
	MaxAttemptArtifacts = 64
	MaxStageOutputs     = 64
	MaxInstructionBytes = 16 * 1024
	MaxDiagnosticBytes  = 1024
	MaxAttemptTimeout   = 24 * time.Hour
)

var (
	ErrExecutionInvalid = errors.New("execution_invalid")
	// ErrLegacyExecution marks bytes of an earlier Execution/Graph/Single
	// format. After the breaking cutover they are never decoded, resumed,
	// retried or converted.
	ErrLegacyExecution    = errors.New("execution_legacy_format_refused: breaking cutover admits only execution format 4; old attempts cannot resume")
	ErrStaleRunner        = errors.New("execution_stale_runner")
	ErrTransition         = errors.New("execution_transition_refused")
	ErrUnauthorizedEffect = errors.New("execution_unauthorized_effect")
	ErrUnknownOutstanding = errors.New("execution_unknown_effect_outstanding")
	ErrCancelled          = errors.New("execution_cancelled")
	// ErrRunnerHeld reports that another runner holds the exclusive lease.
	ErrRunnerHeld = errors.New("execution_runner_active")
)

// RunnerFence is an exclusive runner lease; Generation is its monotonic
// epoch, recorded in the Execution by ClaimRunner. It is an alias so storage
// and engine share the identical type without importing each other.
type RunnerFence = interface {
	Generation() uint64
	Close() error
}

type RunStatus string

const (
	RunReady            RunStatus = "ready"
	RunRunning          RunStatus = "running"
	RunAwaitingGate     RunStatus = "awaiting_gate"
	RunCompleted        RunStatus = "completed"
	RunRejected         RunStatus = "rejected"
	RunFailed           RunStatus = "failed"
	RunCancelled        RunStatus = "cancelled"
	RunRecoveryRequired RunStatus = "recovery_required"
)

type StageStatus string

const (
	StagePending      StageStatus = "pending"
	StageRunning      StageStatus = "running"
	StageAwaitingGate StageStatus = "awaiting_gate"
	StageCompleted    StageStatus = "completed"
	StageFailed       StageStatus = "failed"
	StageCancelled    StageStatus = "cancelled"
)

type AttemptStatus string

const (
	AttemptPrepared  AttemptStatus = "prepared"
	AttemptRunning   AttemptStatus = "running"
	AttemptSucceeded AttemptStatus = "succeeded"
	AttemptFailed    AttemptStatus = "failed"
	AttemptCancelled AttemptStatus = "cancelled"
	// AttemptUnknown means effects cannot be established (crash after
	// preparation/spawn, lost terminal evidence, uncertain stop). It blocks
	// successors and automatic respawn until a human disposition is recorded.
	AttemptUnknown AttemptStatus = "unknown"
)

type EffectState string

const (
	EffectConfirmed EffectState = "confirmed"
	EffectUnknown   EffectState = "unknown"
)

// WorkItemLink is the already-linked traceability identity. Provider
// projection is optional and never a dispatch prerequisite.
type WorkItemLink struct {
	Provider      string `json:"provider"`
	ID            string `json:"id"`
	RepositoryKey string `json:"repositoryKey"`
}

// RepositoryBinding names a Project repository by its portable key and
// canonical identity; machine-local paths are resolved at dispatch time.
type RepositoryBinding struct {
	Key      string `json:"key"`
	Identity string `json:"identity"`
}

// AgentSpec is a Workflow agent: explicit Runtime/model/effort/instructions
// and optional Profile input. Empty fields fall back by precedence.
type AgentSpec struct {
	Inputs          []string `json:"inputs,omitempty"`
	OutputScope     bool     `json:"outputScope,omitempty"`
	DependsOn       []string `json:"dependsOn,omitempty"`
	Outputs         []string `json:"outputs,omitempty"`
	TimeoutSeconds  int64    `json:"timeoutSeconds,omitempty"`
	MaximumAttempts int      `json:"maximumAttempts,omitempty"`
	ID              string   `json:"id"`
	Profile         string   `json:"profile,omitempty"`
	Runtime         string   `json:"runtime,omitempty"`
	Model           string   `json:"model,omitempty"`
	Effort          string   `json:"effort,omitempty"`
	Instructions    string   `json:"instructions,omitempty"`
	Effects         []string `json:"effects,omitempty"`
	// Repository is the bound Repository key the agent works in; empty means
	// the Work Item's Repository.
	Repository string `json:"repository,omitempty"`
	// Runtimes is the explicit user restriction of permitted Runtimes; empty
	// means unrestricted. It is intent, never a vendor capability claim.
	Runtimes []string `json:"runtimes,omitempty"`
}

// OutputCheck freezes the installed typed-artifact validator contract. Evidence
// is an additional retention check, never a substitute for schema validation.
type OutputCheck struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Evidence bool   `json:"evidence,omitempty"`
}

type StageOutputInput struct {
	Name     string `json:"name"`
	StageID  string `json:"stageId"`
	Output   string `json:"output"`
	Required bool   `json:"required"`
}

// InputProvenance identifies the producer of an artifact supplied to another attempt.
// Repository key and bytes remain bound by the accompanying ArtifactRef.
type InputProvenance struct {
	ExecutionID string `json:"executionId"`
	StageID     string `json:"stageId"`
	AttemptID   string `json:"attemptId"`
}

func (p InputProvenance) Valid() bool {
	return ValidExecutionID(p.ExecutionID) && ValidToken(p.StageID) && ValidToken(p.AttemptID)
}

type FrozenInput struct {
	Provenance *InputProvenance `json:"provenance,omitempty"`
	Content    string           `json:"content,omitempty"`
	Digest     string           `json:"digest,omitempty"`
	Name       string           `json:"name"`
	Kind       string           `json:"kind"`
	Source     string           `json:"source"`
	Required   bool             `json:"required"`
	Project    *ProjectInput    `json:"project,omitempty"`
	Artifact   *ArtifactRef     `json:"artifact,omitempty"`
}
type ProjectInput struct {
	ID               string              `json:"id"`
	Slug             string              `json:"slug"`
	PortableRevision string              `json:"portableRevision"`
	Repositories     []RepositoryBinding `json:"repositories"`
}

type CommandCheck struct {
	ID             string   `json:"id"`
	Argv           []string `json:"argv"`
	TimeoutSeconds int64    `json:"timeoutSeconds"`
	Effects        []string `json:"effects"`
}
type StageSpec struct {
	Commands          []CommandCheck     `json:"commands,omitempty"`
	FrozenInputs      []FrozenInput      `json:"frozenInputs,omitempty"`
	StageOutputInputs []StageOutputInput `json:"stageOutputInputs,omitempty"`
	WorkItemInputs    []string           `json:"workItemInputs,omitempty"`
	DefinitionDigest  string             `json:"definitionDigest,omitempty"`
	OptionalOutputs   []string           `json:"optionalOutputs,omitempty"`
	ReviewRequired    bool               `json:"reviewRequired,omitempty"`
	OutputChecks      []OutputCheck      `json:"outputChecks,omitempty"`
	ID                string             `json:"id"`
	DependsOn         []string           `json:"dependsOn,omitempty"`
	Agents            []AgentSpec        `json:"agents"`
	Outputs           []string           `json:"outputs,omitempty"`
	BeforeGate        bool               `json:"beforeGate,omitempty"`
	HumanGate         bool               `json:"humanGate,omitempty"`
	TimeoutSeconds    int64              `json:"timeoutSeconds"`
}

// WorkflowSnapshot is the immutable selected Workflow bound at start; its
// digest is recomputed on every decode so edits cannot retarget a run.
type WorkflowSnapshot struct {
	ID       string      `json:"id"`
	Revision string      `json:"revision"`
	Digest   string      `json:"digest"`
	Stages   []StageSpec `json:"stages"`
}

type FieldProvenance struct {
	Field   string `json:"field"`
	Source  string `json:"source"`
	Profile string `json:"profile,omitempty"`
}

// Invocation is the frozen per-attempt effective selection (ADR-0037 §4).
// It is never edited; a changed selection needs a new attempt.
type Invocation struct {
	Runtime            string            `json:"runtime"`
	Executable         string            `json:"executable"`
	Model              string            `json:"model"`
	Effort             string            `json:"effort"`
	InstructionsDigest string            `json:"instructionsDigest"`
	Instructions       string            `json:"instructions"`
	Profile            string            `json:"profile,omitempty"`
	ProfileDigest      string            `json:"profileDigest,omitempty"`
	Provenance         []FieldProvenance `json:"provenance"`
}

// AttemptAuthority is the exact target and effect bound authorized before
// spawn. Messages and agent output can never extend it.
type AttemptAuthority struct {
	RepositoryKey string   `json:"repositoryKey"`
	Effects       []string `json:"effects"`
	Actor         string   `json:"actor"`
}

type ProcessIdentity struct {
	PID           int64  `json:"pid"`
	StartIdentity string `json:"startIdentity"`
	GroupIdentity string `json:"groupIdentity"`
}

type EffectFact struct {
	Kind   string      `json:"kind"`
	Target string      `json:"target"`
	State  EffectState `json:"state"`
}

type ArtifactRef struct {
	RetainedID    string `json:"retainedId,omitempty"`
	Name          string `json:"name"`
	RepositoryKey string `json:"repositoryKey"`
	Digest        string `json:"digest"`
	Size          int64  `json:"size"`
}

type Disposition struct {
	Actor      string    `json:"actor"`
	Decision   string    `json:"decision"`
	RecordedAt time.Time `json:"recordedAt"`
}

var ErrWorkspaceDispositionRequired = errors.New("workspace_disposition_required")

type CommandResult struct {
	Status     string           `json:"status"`
	PreparedAt time.Time        `json:"preparedAt"`
	StartedAt  *time.Time       `json:"startedAt,omitempty"`
	EndedAt    *time.Time       `json:"endedAt,omitempty"`
	Process    *ProcessIdentity `json:"process,omitempty"`
	ID         string           `json:"id"`
	ArgvDigest string           `json:"argvDigest"`
	ExitCode   int              `json:"exitCode"`
}
type Attempt struct {
	Accounting           string           `json:"accounting,omitempty"`
	ParentAttemptID      string           `json:"parentAttemptId,omitempty"`
	NoSpawn              bool             `json:"noSpawn,omitempty"`
	Materialization      string           `json:"materialization,omitempty"`
	LaunchFailure        *LaunchFailure   `json:"launchFailure,omitempty"`
	Commands             []CommandResult  `json:"commands,omitempty"`
	WorkspaceDisposition string           `json:"workspaceDisposition,omitempty"`
	AttemptID            string           `json:"attemptId"`
	Number               int              `json:"number"`
	RunnerGeneration     uint64           `json:"runnerGeneration"`
	Invocation           Invocation       `json:"invocation"`
	Authority            AttemptAuthority `json:"authority"`
	Status               AttemptStatus    `json:"status"`
	PreparedAt           time.Time        `json:"preparedAt"`
	StartedAt            *time.Time       `json:"startedAt,omitempty"`
	EndedAt              *time.Time       `json:"endedAt,omitempty"`
	Process              *ProcessIdentity `json:"process,omitempty"`
	ExitCode             *int             `json:"exitCode,omitempty"`
	Diagnostic           string           `json:"diagnostic,omitempty"`
	Effects              []EffectFact     `json:"effects,omitempty"`
	Artifacts            []ArtifactRef    `json:"artifacts,omitempty"`
	Resolution           *Disposition     `json:"resolution,omitempty"`
}

type TaskRun struct {
	AgentID  string    `json:"agentId"`
	Attempts []Attempt `json:"attempts,omitempty"`
}

type StageRun struct {
	StageID        string        `json:"stageId"`
	Status         StageStatus   `json:"status"`
	Tasks          []TaskRun     `json:"tasks"`
	Gate           *Disposition  `json:"gate,omitempty"`
	BeforeApproval *Disposition  `json:"beforeApproval,omitempty"`
	Review         *ResultReview `json:"review,omitempty"`
}

type ResultReview struct {
	Actor        string      `json:"actor"`
	ResultDigest string      `json:"resultDigest"`
	Evidence     ArtifactRef `json:"evidence"`
	RecordedAt   time.Time   `json:"recordedAt"`
}

type CancelIntent struct {
	Actor       string    `json:"actor"`
	RequestedAt time.Time `json:"requestedAt"`
}

type Execution struct {
	AttemptPolicy    string              `json:"attemptPolicy,omitempty"`
	Rejection        *ResultRejection    `json:"rejection,omitempty"`
	ReworkOf         *RejectedSource     `json:"reworkOf,omitempty"`
	Kind             string              `json:"kind"`
	FormatVersion    int                 `json:"formatVersion"`
	ExecutionID      string              `json:"executionId"`
	ProjectID        string              `json:"projectId"`
	WorkItem         WorkItemLink        `json:"workItem"`
	Repositories     []RepositoryBinding `json:"repositories"`
	Workflow         WorkflowSnapshot    `json:"workflow"`
	Revision         uint64              `json:"revision"`
	RunnerGeneration uint64              `json:"runnerGeneration"`
	Status           RunStatus           `json:"status"`
	Cancel           *CancelIntent       `json:"cancel,omitempty"`
	Stages           []StageRun          `json:"stages"`
	Messages         []Message           `json:"messages,omitempty"`
	// Integrations are principal-controlled Git integrations of attempt
	// branches (S2.3); append-only intent and immutable terminal observations.
	// Absent for runs that integrate nothing.
	Integrations []Integration       `json:"integrations,omitempty"`
	Deliveries   []ExecutionDelivery `json:"deliveries,omitempty"`
	// Selections are the explicit run overrides given by the start action.
	// They are immutable inputs of every later attempt's resolution; changing
	// them needs an explicitly authorized new selection on a retry.
	Selections []Selection `json:"selections,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

// TaskRef addresses one agent task of one stage.
type TaskRef struct {
	StageID string
	AgentID string
}

type NewExecutionInput struct {
	ExecutionID  string
	ProjectID    string
	WorkItem     WorkItemLink
	Repositories []RepositoryBinding
	WorkflowID   string
	Revision     string
	Stages       []StageSpec
	Selections   []Selection
	Now          time.Time
}

// Selection is one explicit run override of an invocation field for an
// agent. An empty StageID applies to that agent ID in every stage.
type Selection struct {
	StageID string `json:"stageId,omitempty"`
	AgentID string `json:"agentId"`
	Field   string `json:"field"`
	Value   string `json:"value"`
}

// SelectionFields are the run-overridable invocation inputs; profile selects
// an optional reusable tuple at the run level.
var SelectionFields = []string{"runtime", "model", "effort", "instructions", "profile"}

func validateSelections(e Execution) error {
	if len(e.Selections) > MaxExecutionStages*MaxStageAgents*len(SelectionFields) {
		return invalid("selections")
	}
	seen := map[Selection]bool{}
	for _, s := range e.Selections {
		key := Selection{StageID: s.StageID, AgentID: s.AgentID, Field: s.Field}
		if seen[key] || !optionalToken(s.StageID) || !ValidToken(s.AgentID) || s.Value == "" {
			return invalid("selection %s.%s", s.AgentID, s.Field)
		}
		seen[key] = true
		switch s.Field {
		case "runtime", "effort", "profile":
			if !ValidToken(s.Value) {
				return invalid("selection %s.%s", s.AgentID, s.Field)
			}
		case "model":
			if !boundedText(s.Value, 256) {
				return invalid("selection %s.%s", s.AgentID, s.Field)
			}
		case "instructions":
			if !boundedText(s.Value, MaxInstructionBytes) {
				return invalid("selection %s.%s", s.AgentID, s.Field)
			}
		default:
			return invalid("selection field %q", s.Field)
		}
		matched := false
		for _, stage := range e.Workflow.Stages {
			for _, agent := range stage.Agents {
				matched = matched || agent.ID == s.AgentID && (s.StageID == "" || s.StageID == stage.ID)
			}
		}
		if !matched {
			return invalid("selection %s.%s names no Workflow agent", s.AgentID, s.Field)
		}
	}
	return nil
}

// NewExecution validates the Workflow (unique IDs, known dependencies, no
// cycles, bounded agents) before any attempt can exist.
func NewExecution(in NewExecutionInput) (Execution, error) {
	snapshot := WorkflowSnapshot{ID: in.WorkflowID, Revision: in.Revision, Stages: cloneStages(in.Stages)}
	digest, err := workflowDigest(snapshot)
	if err != nil {
		return Execution{}, err
	}
	snapshot.Digest = digest
	if err := validateWorkflow(snapshot, nil); err != nil {
		return Execution{}, err
	}
	now := in.Now.UTC()
	e := Execution{
		Kind: ExecutionKind, FormatVersion: ExecutionFormatVersion,
		ExecutionID: in.ExecutionID, ProjectID: in.ProjectID, WorkItem: in.WorkItem,
		Repositories: append([]RepositoryBinding(nil), in.Repositories...),
		Workflow:     snapshot, Revision: 1, Status: RunReady, CreatedAt: now, UpdatedAt: now,
		Selections: append([]Selection(nil), in.Selections...),
	}
	for _, stage := range snapshot.Stages {
		run := StageRun{StageID: stage.ID, Status: StagePending}
		for _, agent := range stage.Agents {
			run.Tasks = append(run.Tasks, TaskRun{AgentID: agent.ID})
		}
		e.Stages = append(e.Stages, run)
	}
	e.refreshStageStatuses()
	e.Status = e.derivedStatus()
	if err := e.Validate(); err != nil {
		return Execution{}, err
	}
	return e, nil
}

func cloneStages(stages []StageSpec) []StageSpec {
	wire, _ := json.Marshal(stages)
	var out []StageSpec
	_ = json.Unmarshal(wire, &out)
	return out
}

func workflowDigest(s WorkflowSnapshot) (string, error) {
	wire, err := json.Marshal(struct {
		ID       string      `json:"id"`
		Revision string      `json:"revision"`
		Stages   []StageSpec `json:"stages"`
	}{s.ID, s.Revision, s.Stages})
	if err != nil {
		return "", ErrExecutionInvalid
	}
	sum := sha256.Sum256(wire)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// DigestText is the canonical digest form used for instructions and artifacts.
func DigestText(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// EncodeExecution validates before producing the only canonical bytes.
func EncodeExecution(e Execution) ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	wire, err := json.Marshal(e)
	if err != nil {
		return nil, ErrExecutionInvalid
	}
	wire = append(wire, '\n')
	if len(wire) > MaxExecutionBytes {
		return nil, fmt.Errorf("%w: encoded execution exceeds %d bytes", ErrExecutionInvalid, MaxExecutionBytes)
	}
	return wire, nil
}

// DecodeExecution never reinterprets other formats: any other discriminator
// or version is refused, and recognizable earlier formats report the cutover.
func DecodeExecution(wire []byte) (Execution, error) {
	if len(wire) == 0 || len(wire) > MaxExecutionBytes {
		return Execution{}, ErrExecutionInvalid
	}
	var header struct {
		Kind          *string `json:"kind"`
		FormatVersion *int    `json:"formatVersion"`
	}
	if err := json.Unmarshal(wire, &header); err != nil {
		return Execution{}, ErrExecutionInvalid
	}
	if header.Kind == nil || *header.Kind != ExecutionKind || header.FormatVersion == nil || *header.FormatVersion != ExecutionFormatVersion {
		// Every earlier Execution family (including the unkinded executions/v1
		// formats up to 5), Graph and single-stage ledger lacks this
		// discriminator/version pair.
		if header.Kind == nil || *header.Kind != ExecutionKind || (header.FormatVersion != nil && *header.FormatVersion < ExecutionFormatVersion) {
			return Execution{}, ErrLegacyExecution
		}
		return Execution{}, ErrExecutionInvalid
	}
	var e Execution
	if err := strictjson.Decode(wire, &e, MaxExecutionBytes); err != nil {
		return Execution{}, ErrExecutionInvalid
	}
	if err := e.Validate(); err != nil {
		return Execution{}, err
	}
	return e, nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrExecutionInvalid}, args...)...)
}

// ValidToken bounds every identifier stored in an Execution.
func ValidToken(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':') {
			return false
		}
	}
	return value != "." && value != ".."
}

// ValidExecutionID matches the 36-character identity used by the store and
// runner lease addresses.
func ValidExecutionID(value string) bool { return len(value) == 36 && ValidToken(value) }

func boundedText(value string, limit int) bool {
	if len(value) > limit {
		return false
	}
	for _, c := range value {
		if c == 0 {
			return false
		}
	}
	return true
}

func (e Execution) Validate() error {
	if e.AttemptPolicy != "" && e.AttemptPolicy != SpawnBoundaryPolicy {
		return invalid("attempt policy")
	}
	if e.Kind != ExecutionKind || e.FormatVersion != ExecutionFormatVersion {
		return invalid("discriminator")
	}
	if !ValidExecutionID(e.ExecutionID) || !ValidToken(e.ProjectID) || e.Revision == 0 {
		return invalid("identity")
	}
	if !ValidToken(e.WorkItem.Provider) || !ValidToken(e.WorkItem.ID) || !ValidToken(e.WorkItem.RepositoryKey) {
		return invalid("work item link")
	}
	repos := map[string]bool{}
	if len(e.Repositories) == 0 || len(e.Repositories) > 16 {
		return invalid("repositories")
	}
	for _, r := range e.Repositories {
		if !ValidToken(r.Key) || !boundedText(r.Identity, 512) || r.Identity == "" || repos[r.Key] {
			return invalid("repository binding")
		}
		repos[r.Key] = true
	}
	if !repos[e.WorkItem.RepositoryKey] {
		return invalid("work item repository is not bound")
	}
	if e.CreatedAt.IsZero() || e.UpdatedAt.Before(e.CreatedAt) {
		return invalid("timestamps")
	}
	if err := validateWorkflow(e.Workflow, repos); err != nil {
		return err
	}
	if len(e.Stages) != len(e.Workflow.Stages) {
		return invalid("stage runs")
	}
	for i, run := range e.Stages {
		spec := e.Workflow.Stages[i]
		if run.StageID != spec.ID || len(run.Tasks) != len(spec.Agents) || !validStageStatus(run.Status) || run.Status != e.derivedStageStatus(i) {
			return invalid("stage %q", run.StageID)
		}
		if run.Gate != nil && ((!spec.HumanGate && !spec.BeforeGate) || !validDisposition(*run.Gate) || run.Gate.Decision != DispositionApproved || len(run.Gate.Actor) < 7 || run.Gate.Actor[:6] != "human:") {
			return invalid("stage %q gate", run.StageID)
		}
		if run.BeforeApproval != nil && (!spec.BeforeGate || !humanDisposition(*run.BeforeApproval)) {
			return invalid("before approval")
		}
		if run.Review != nil {
			digest, err := e.StageResultDigest(run.StageID)
			r := run.Review
			if err != nil || !spec.ReviewRequired || r.ResultDigest != digest || !humanActor(r.Actor) || r.RecordedAt.IsZero() || r.Evidence.RetainedID == "" || r.Evidence.RepositoryKey != e.WorkItem.RepositoryKey || !validPrefixedDigest(r.Evidence.Digest) {
				return invalid("result review")
			}
		}
		for j, task := range run.Tasks {
			if task.AgentID != spec.Agents[j].ID {
				return invalid("task %q", task.AgentID)
			}
			if err := e.validateAccounting(task, spec.Agents[j]); err != nil {
				return err
			}
			for k, attempt := range task.Attempts {
				if err := validateAttempt(attempt, k+1, e.RunnerGeneration, repos); err != nil {
					return err
				}
				if len(attempt.Commands) > len(spec.Commands) {
					return invalid("command results")
				}
				if attempt.Status == AttemptSucceeded && len(attempt.Commands) != len(spec.Commands) {
					return invalid("missing command results")
				}
				for k, result := range attempt.Commands {
					wire, _ := json.Marshal(spec.Commands[k])
					if result.ID != spec.Commands[k].ID || result.ArgvDigest != DigestText(wire) || attempt.Status == AttemptSucceeded && (result.ExitCode != 0 || result.Status != "completed") {
						return invalid("command result")
					}
					if result.PreparedAt.IsZero() || (result.Status != "prepared" && result.Status != "running" && result.Status != "completed") || result.Status == "prepared" && (result.Process != nil || result.StartedAt != nil || result.EndedAt != nil) || result.Status == "running" && (result.Process == nil || result.StartedAt == nil || result.EndedAt != nil) || result.Status == "completed" && (result.Process == nil || result.StartedAt == nil || result.EndedAt == nil) {
						return invalid("command process facts")
					}
					if result.Process != nil && (result.Process.PID <= 1 || result.Process.StartIdentity == "" || result.Process.GroupIdentity == "") {
						return invalid("command identity")
					}
				}
				seen := map[string]bool{}
				for _, artifact := range attempt.Artifacts {
					if seen[artifact.Name] {
						return invalid("duplicate attempt artifact")
					}
					seen[artifact.Name] = true
					for _, check := range spec.OutputChecks {
						if check.Name == artifact.Name && artifact.RetainedID == "" {
							return invalid("validated artifact requires retained identity")
						}
					}
				}
				if attempt.Status == AttemptSucceeded {
					agent := spec.Agents[j]
					for _, name := range spec.Outputs {
						if (!agent.OutputScope || slices.Contains(agent.Outputs, name)) && !seen[name] {
							return invalid("succeeded attempt missing required output")
						}
					}
				}
			}
		}
	}
	if e.Cancel != nil && (!ValidToken(e.Cancel.Actor) || e.Cancel.RequestedAt.IsZero()) {
		return invalid("cancel intent")
	}
	if err := validateMessages(e); err != nil {
		return err
	}
	if err := validateRework(e); err != nil {
		return err
	}
	if err := validateSelections(e); err != nil {
		return err
	}
	if err := validateDeliveries(e); err != nil {
		return err
	}
	if err := validateIntegrations(e); err != nil {
		return err
	}
	if !validRunStatus(e.Status) || e.Status != e.derivedStatus() {
		return invalid("status %q does not match its facts", e.Status)
	}
	wire, err := json.Marshal(e)
	if err != nil || len(wire)+1 > MaxExecutionBytes {
		return invalid("aggregate bounds")
	}
	return nil
}

func validateWorkflow(w WorkflowSnapshot, repos map[string]bool) error {
	if !ValidToken(w.ID) || !ValidToken(w.Revision) || len(w.Stages) == 0 || len(w.Stages) > MaxExecutionStages {
		return invalid("workflow")
	}
	digest, err := workflowDigest(w)
	if err != nil || digest != w.Digest {
		return invalid("workflow digest")
	}
	index := map[string]int{}
	for i, s := range w.Stages {
		if !ValidToken(s.ID) {
			return invalid("stage id")
		}
		if _, dup := index[s.ID]; dup {
			return invalid("duplicate stage %q", s.ID)
		}
		index[s.ID] = i
		if len(s.Agents) == 0 || len(s.Agents) > MaxStageAgents || len(s.Outputs)+len(s.OptionalOutputs) > MaxStageOutputs {
			return invalid("stage %q agents/outputs", s.ID)
		}

		if s.TimeoutSeconds <= 0 || s.TimeoutSeconds > int64(MaxAttemptTimeout/time.Second) {
			return invalid("stage %q timeout", s.ID)
		}
		if len(s.OutputChecks) > MaxStageOutputs || (len(s.OutputChecks) != 0 && len(s.OutputChecks) != len(s.Outputs)+len(s.OptionalOutputs)) || (len(s.OutputChecks) != 0 && !validPrefixedDigest(s.DefinitionDigest)) || (len(s.OutputChecks) == 0 && s.DefinitionDigest != "") {
			return invalid("output validator bounds/digest")
		}
		if len(s.Commands) > 64 {
			return invalid("command bounds")
		}
		commandIDs := map[string]bool{}
		for _, c := range s.Commands {
			if !ValidToken(c.ID) || commandIDs[c.ID] || len(c.Argv) == 0 || len(c.Argv) > 64 || c.TimeoutSeconds <= 0 || c.TimeoutSeconds > int64(MaxAttemptTimeout/time.Second) {
				return invalid("command")
			}
			commandIDs[c.ID] = true
			for _, arg := range c.Argv {
				if arg == "" || !boundedText(arg, 4096) {
					return invalid("command argv")
				}
			}
			if validEffectKinds(c.Effects) != nil || !slices.Contains(c.Effects, "validation_command") {
				return invalid("command effects")
			}
		}
		inputs := map[string]bool{}
		if len(s.WorkItemInputs) > 64 {
			return invalid("stage input bounds")
		}
		for _, name := range s.WorkItemInputs {
			if !ValidToken(name) || inputs[name] {
				return invalid("stage input")
			}
			inputs[name] = true
		}
		for _, input := range s.FrozenInputs {
			if !ValidToken(input.Name) || inputs[input.Name] || !boundedText(input.Source, 256) || input.Source == "" {
				return invalid("frozen input")
			}
			if input.Kind == "project-context" {
				if input.Artifact != nil || input.Provenance != nil {
					return invalid("project input")
				}
				if !boundedText(input.Content, MaxInstructionBytes) || input.Content != "" && input.Digest != DigestText([]byte(input.Content)) {
					return invalid("context digest/bounds")
				}
				if input.Project != nil && (!ValidToken(input.Project.ID) || len(input.Project.Repositories) > 16 || !boundedText(input.Project.PortableRevision, 256)) {
					return invalid("project context")
				}
			} else if input.Kind == "work-item" {
				if input.Artifact != nil || input.Project != nil || input.Provenance != nil || input.Source != "target" {
					return invalid("work item input")
				}
				if input.Content == "" || !boundedText(input.Content, MaxInstructionBytes) || input.Digest != DigestText([]byte(input.Content)) {
					return invalid("work item digest/bounds")
				}
			} else if input.Kind == "artifact" {
				if input.Project != nil {
					return invalid("artifact input")
				}
				if (input.Artifact == nil) != (input.Provenance == nil) || input.Provenance != nil && !input.Provenance.Valid() {
					return invalid("artifact provenance")
				}
				if input.Artifact != nil && (!ValidToken(input.Artifact.RetainedID) || !validPrefixedDigest(input.Artifact.Digest) || input.Artifact.RepositoryKey == "") {
					return invalid("artifact reference")
				}
			} else {
				return invalid("frozen input kind")
			}
			inputs[input.Name] = true
		}
		if len(s.WorkItemInputs)+len(s.StageOutputInputs)+len(s.FrozenInputs) > 64 {
			return invalid("stage input bounds")
		}
		for _, input := range s.StageOutputInputs {
			if !ValidToken(input.Name) || inputs[input.Name] {
				return invalid("stage output input")
			}
			found := false
			for j := 0; j < i; j++ {
				if w.Stages[j].ID != input.StageID {
					continue
				}
				for _, out := range append(append([]string(nil), w.Stages[j].Outputs...), w.Stages[j].OptionalOutputs...) {
					found = found || out == input.Output
				}
			}
			if !found {
				return invalid("unavailable stage output input")
			}
			inputs[input.Name] = true
		}
		checks := map[string]bool{}
		for _, check := range s.OutputChecks {
			found := false
			for _, name := range append(append([]string(nil), s.Outputs...), s.OptionalOutputs...) {
				found = found || name == check.Name
			}
			if !found || checks[check.Name] || (check.Kind != "artifact" && check.Kind != "evidence" && check.Kind != "decision" && check.Kind != "result") {
				return invalid("output validator")
			}
			checks[check.Name] = true
		}
		agents := map[string]bool{}
		for _, a := range s.Agents {
			if !ValidToken(a.ID) || agents[a.ID] || !optionalToken(a.Profile) || !optionalToken(a.Runtime) || !boundedText(a.Model, 256) || !optionalToken(a.Effort) || !boundedText(a.Instructions, MaxInstructionBytes) {
				return invalid("stage %q agent %q", s.ID, a.ID)
			}
			agents[a.ID] = true
			// The pre-binding snapshot check (nil repos) validates syntax only.
			if !optionalToken(a.Repository) || repos != nil && a.Repository != "" && !repos[a.Repository] {
				return invalid("stage %q agent %q repository", s.ID, a.ID)
			}
			if a.MaximumAttempts < 0 || a.MaximumAttempts > MaxTaskAttempts || a.TimeoutSeconds < 0 || a.TimeoutSeconds > int64(MaxAttemptTimeout/time.Second) {
				return invalid("agent limits")
			}
			seenInputs := map[string]bool{}
			if len(a.Inputs) > 64 {
				return invalid("agent input bounds")
			}
			for _, name := range a.Inputs {
				if !inputs[name] || seenInputs[name] {
					return invalid("agent input")
				}
				seenInputs[name] = true
			}
			seen := map[string]bool{}
			for _, out := range a.Outputs {
				if !ValidToken(out) || seen[out] {
					return invalid("agent output")
				}
				seen[out] = true
				found := false
				for _, output := range append(append([]string(nil), s.Outputs...), s.OptionalOutputs...) {
					found = found || out == output
				}
				if !found {
					return invalid("dangling agent output")
				}
			}

			if err := validEffectKinds(a.Effects); err != nil {
				return err
			}
			permitted := map[string]bool{}
			for _, r := range a.Runtimes {
				if !ValidToken(r) || permitted[r] {
					return invalid("stage %q agent %q runtimes", s.ID, a.ID)
				}
				permitted[r] = true
			}
		}
		// Validate the complete agent DAG before any attempt or filesystem effect.
		color := map[string]int{}
		var visitAgent func(string) bool
		visitAgent = func(id string) bool {
			if !agents[id] || color[id] == 1 {
				return false
			}
			if color[id] == 2 {
				return true
			}
			color[id] = 1
			for _, a := range s.Agents {
				if a.ID == id {
					seen := map[string]bool{}
					for _, dep := range a.DependsOn {
						if seen[dep] || !visitAgent(dep) {
							return false
						}
						seen[dep] = true
					}
				}
			}
			color[id] = 2
			return true
		}
		for _, a := range s.Agents {
			if !visitAgent(a.ID) {
				return invalid("cyclic or dangling agent dependency")
			}
		}
		outputs := map[string]bool{}
		for _, o := range append(append([]string(nil), s.Outputs...), s.OptionalOutputs...) {
			if !ValidToken(o) || outputs[o] {
				return invalid("stage %q output", s.ID)
			}
			outputs[o] = true
			owned := false
			for _, agent := range s.Agents {
				if !agent.OutputScope {
					owned = true
				}
				for _, assigned := range agent.Outputs {
					owned = owned || assigned == o
				}
			}
			if !owned {
				return invalid("unowned required output")
			}
		}
	}
	// Dependencies must name known stages and form a DAG; this runs before any
	// attempt can be prepared.
	state := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		switch state[id] {
		case 1:
			return false
		case 2:
			return true
		}
		state[id] = 1
		for _, dep := range w.Stages[index[id]].DependsOn {
			if _, ok := index[dep]; !ok || dep == id || !visit(dep) {
				return false
			}
		}
		state[id] = 2
		return true
	}
	for _, s := range w.Stages {
		if !visit(s.ID) {
			return invalid("stage %q has an unknown or cyclic dependency", s.ID)
		}
	}
	return nil
}

func optionalToken(v string) bool { return v == "" || ValidToken(v) }

var attemptEffectKinds = map[string]bool{"workspace_write": true, "validation_command": true, "network": true}

func validEffectKinds(kinds []string) error {
	seen := map[string]bool{}
	for _, k := range kinds {
		if !attemptEffectKinds[k] || seen[k] {
			return invalid("effect kind %q", k)
		}
		seen[k] = true
	}
	return nil
}

func validateAttempt(a Attempt, number int, generation uint64, repos map[string]bool) error {
	if a.LaunchFailure != nil && (!a.LaunchFailure.Valid() || a.Status != AttemptFailed || a.StartedAt != nil || a.Process != nil) {
		return invalid("launch failure")
	}
	if a.WorkspaceDisposition != "" && (number == 1 || a.WorkspaceDisposition != "preserve") {
		return invalid("workspace disposition")
	}
	if !ValidToken(a.AttemptID) || a.Number != number || a.RunnerGeneration == 0 || a.RunnerGeneration > generation {
		return invalid("attempt %q identity", a.AttemptID)
	}
	if err := validateInvocation(a.Invocation); err != nil {
		return err
	}
	if !repos[a.Authority.RepositoryKey] || !ValidToken(a.Authority.Actor) {
		return invalid("attempt %q authority", a.AttemptID)
	}
	if err := validEffectKinds(a.Authority.Effects); err != nil {
		return err
	}
	if a.PreparedAt.IsZero() || !boundedText(a.Diagnostic, MaxDiagnosticBytes) {
		return invalid("attempt %q facts", a.AttemptID)
	}
	switch a.Status {
	case AttemptPrepared:
		if a.StartedAt != nil || a.EndedAt != nil || a.Process != nil || a.ExitCode != nil {
			return invalid("attempt %q prepared facts", a.AttemptID)
		}
	case AttemptRunning:
		if a.StartedAt == nil || a.Process == nil || a.EndedAt != nil {
			return invalid("attempt %q running facts", a.AttemptID)
		}
	case AttemptSucceeded:
		if a.EndedAt == nil || a.ExitCode == nil || *a.ExitCode != 0 {
			return invalid("attempt %q success facts", a.AttemptID)
		}
	case AttemptFailed, AttemptCancelled, AttemptUnknown:
	default:
		return invalid("attempt %q status", a.AttemptID)
	}
	if a.Process != nil && (a.Process.PID <= 1 || !boundedText(a.Process.StartIdentity, 256) || !boundedText(a.Process.GroupIdentity, 256)) {
		return invalid("attempt %q process", a.AttemptID)
	}
	if len(a.Effects) > MaxAttemptEffects || len(a.Artifacts) > MaxAttemptArtifacts {
		return invalid("attempt %q bounds", a.AttemptID)
	}
	allowed := map[string]bool{}
	for _, k := range a.Authority.Effects {
		allowed[k] = true
	}
	for _, f := range a.Effects {
		// CompleteAttempt downgrades any unknown effect to an unknown attempt;
		// success with an unknown effect is never a writer-produced fact.
		if !allowed[f.Kind] || !boundedText(f.Target, 512) || (f.State != EffectConfirmed && f.State != EffectUnknown) || f.State == EffectUnknown && a.Status == AttemptSucceeded {
			return invalid("attempt %q effect", a.AttemptID)
		}
	}
	for _, r := range a.Artifacts {
		if !optionalToken(r.RetainedID) || !ValidToken(r.Name) || r.RepositoryKey != a.Authority.RepositoryKey || !validPrefixedDigest(r.Digest) || r.Size < 0 {
			return invalid("attempt %q artifact", a.AttemptID)
		}
	}
	if a.Resolution != nil && (a.Status != AttemptUnknown || !validDisposition(*a.Resolution)) {
		return invalid("attempt %q resolution", a.AttemptID)
	}
	return nil
}

func validateInvocation(i Invocation) error {
	if !ValidToken(i.Runtime) || i.Executable == "" || !boundedText(i.Executable, 1024) || i.Model == "" || !boundedText(i.Model, 256) || !ValidToken(i.Effort) {
		return invalid("invocation")
	}
	if !boundedText(i.Instructions, MaxInstructionBytes) || i.InstructionsDigest != DigestText([]byte(i.Instructions)) {
		return invalid("invocation instructions")
	}
	if !optionalToken(i.Profile) || (i.Profile == "") != (i.ProfileDigest == "") || (i.ProfileDigest != "" && !validPrefixedDigest(i.ProfileDigest)) {
		return invalid("invocation profile")
	}
	fields := map[string]bool{}
	for _, p := range i.Provenance {
		if !invocationFields[p.Field] || fields[p.Field] || !invocationSources[p.Source] || !optionalToken(p.Profile) {
			return invalid("invocation provenance")
		}
		fields[p.Field] = true
	}
	if len(fields) != len(invocationFields) {
		return invalid("invocation provenance incomplete")
	}
	return nil
}

var invocationFields = map[string]bool{"runtime": true, "model": true, "effort": true, "instructions": true}
var invocationSources = map[string]bool{"run_override": true, "workflow_agent": true, "local_default": true}

func validPrefixedDigest(v string) bool {
	if len(v) != len("sha256:")+64 || v[:7] != "sha256:" {
		return false
	}
	_, err := hex.DecodeString(v[7:])
	return err == nil
}

func validDisposition(d Disposition) bool {
	return ValidToken(d.Actor) && ValidToken(d.Decision) && !d.RecordedAt.IsZero()
}

func validStageStatus(s StageStatus) bool {
	switch s {
	case StagePending, StageRunning, StageAwaitingGate, StageCompleted, StageFailed, StageCancelled:
		return true
	}
	return false
}

func validRunStatus(s RunStatus) bool {
	switch s {
	case RunReady, RunRunning, RunAwaitingGate, RunCompleted, RunRejected, RunFailed, RunCancelled, RunRecoveryRequired:
		return true
	}
	return false
}

// AttemptLimit preserves the default for minimal Workflow agents; configured
// definitions always carry their explicit maximum.
func (a AgentSpec) AttemptLimit() int {
	if a.MaximumAttempts != 0 {
		return a.MaximumAttempts
	}
	return MaxTaskAttempts
}
