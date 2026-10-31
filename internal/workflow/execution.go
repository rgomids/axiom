package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	MaxStageAgents      = 8
	MaxTaskAttempts     = 10
	MaxAttemptEffects   = 32
	MaxAttemptArtifacts = 32
	MaxStageOutputs     = 16
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
)

type RunStatus string

const (
	RunReady            RunStatus = "ready"
	RunRunning          RunStatus = "running"
	RunAwaitingGate     RunStatus = "awaiting_gate"
	RunCompleted        RunStatus = "completed"
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
	ID           string   `json:"id"`
	Profile      string   `json:"profile,omitempty"`
	Runtime      string   `json:"runtime,omitempty"`
	Model        string   `json:"model,omitempty"`
	Effort       string   `json:"effort,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	Effects      []string `json:"effects,omitempty"`
}

type StageSpec struct {
	ID             string      `json:"id"`
	DependsOn      []string    `json:"dependsOn,omitempty"`
	Agents         []AgentSpec `json:"agents"`
	Outputs        []string    `json:"outputs,omitempty"`
	HumanGate      bool        `json:"humanGate,omitempty"`
	TimeoutSeconds int64       `json:"timeoutSeconds"`
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

type Attempt struct {
	AttemptID        string           `json:"attemptId"`
	Number           int              `json:"number"`
	RunnerGeneration uint64           `json:"runnerGeneration"`
	Invocation       Invocation       `json:"invocation"`
	Authority        AttemptAuthority `json:"authority"`
	Status           AttemptStatus    `json:"status"`
	PreparedAt       time.Time        `json:"preparedAt"`
	StartedAt        *time.Time       `json:"startedAt,omitempty"`
	EndedAt          *time.Time       `json:"endedAt,omitempty"`
	Process          *ProcessIdentity `json:"process,omitempty"`
	ExitCode         *int             `json:"exitCode,omitempty"`
	Diagnostic       string           `json:"diagnostic,omitempty"`
	Effects          []EffectFact     `json:"effects,omitempty"`
	Artifacts        []ArtifactRef    `json:"artifacts,omitempty"`
	Resolution       *Disposition     `json:"resolution,omitempty"`
}

type TaskRun struct {
	AgentID  string    `json:"agentId"`
	Attempts []Attempt `json:"attempts,omitempty"`
}

type StageRun struct {
	StageID string       `json:"stageId"`
	Status  StageStatus  `json:"status"`
	Tasks   []TaskRun    `json:"tasks"`
	Gate    *Disposition `json:"gate,omitempty"`
}

type CancelIntent struct {
	Actor       string    `json:"actor"`
	RequestedAt time.Time `json:"requestedAt"`
}

type Execution struct {
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
	CreatedAt        time.Time           `json:"createdAt"`
	UpdatedAt        time.Time           `json:"updatedAt"`
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
	Now          time.Time
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
	now := in.Now.UTC()
	e := Execution{
		Kind: ExecutionKind, FormatVersion: ExecutionFormatVersion,
		ExecutionID: in.ExecutionID, ProjectID: in.ProjectID, WorkItem: in.WorkItem,
		Repositories: append([]RepositoryBinding(nil), in.Repositories...),
		Workflow:     snapshot, Revision: 1, Status: RunReady, CreatedAt: now, UpdatedAt: now,
	}
	for _, stage := range snapshot.Stages {
		run := StageRun{StageID: stage.ID, Status: StagePending}
		for _, agent := range stage.Agents {
			run.Tasks = append(run.Tasks, TaskRun{AgentID: agent.ID})
		}
		e.Stages = append(e.Stages, run)
	}
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
		if run.Gate != nil && (!spec.HumanGate || !validDisposition(*run.Gate)) {
			return invalid("stage %q gate", run.StageID)
		}
		for j, task := range run.Tasks {
			if task.AgentID != spec.Agents[j].ID || len(task.Attempts) > MaxTaskAttempts {
				return invalid("task %q", task.AgentID)
			}
			for k, attempt := range task.Attempts {
				if err := validateAttempt(attempt, k+1, e.RunnerGeneration, repos); err != nil {
					return err
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
	if !validRunStatus(e.Status) || e.Status != e.derivedStatus() {
		return invalid("status %q does not match its facts", e.Status)
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
		if len(s.Agents) == 0 || len(s.Agents) > MaxStageAgents || len(s.Outputs) > MaxStageOutputs {
			return invalid("stage %q agents/outputs", s.ID)
		}
		if s.TimeoutSeconds <= 0 || time.Duration(s.TimeoutSeconds)*time.Second > MaxAttemptTimeout {
			return invalid("stage %q timeout", s.ID)
		}
		agents := map[string]bool{}
		for _, a := range s.Agents {
			if !ValidToken(a.ID) || agents[a.ID] || !optionalToken(a.Profile) || !optionalToken(a.Runtime) || !boundedText(a.Model, 256) || !optionalToken(a.Effort) || !boundedText(a.Instructions, MaxInstructionBytes) {
				return invalid("stage %q agent %q", s.ID, a.ID)
			}
			agents[a.ID] = true
			if err := validEffectKinds(a.Effects); err != nil {
				return err
			}
		}
		outputs := map[string]bool{}
		for _, o := range s.Outputs {
			if !ValidToken(o) || outputs[o] {
				return invalid("stage %q output", s.ID)
			}
			outputs[o] = true
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
		if !allowed[f.Kind] || !boundedText(f.Target, 512) || (f.State != EffectConfirmed && f.State != EffectUnknown) {
			return invalid("attempt %q effect", a.AttemptID)
		}
	}
	for _, r := range a.Artifacts {
		if !ValidToken(r.Name) || r.RepositoryKey != a.Authority.RepositoryKey || !validPrefixedDigest(r.Digest) || r.Size < 0 {
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
	case RunReady, RunRunning, RunAwaitingGate, RunCompleted, RunFailed, RunCancelled, RunRecoveryRequired:
		return true
	}
	return false
}
