// Package workflow owns canonical Execution and historical state validation.
package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/provenance"
)

const (
	FormatVersion   = 1
	WorkflowVersion = "mvp-v1-sequential"
	maxEvents       = 32
	maxReferences   = 16
	maxTextBytes    = 512
	stagePrefix     = "axiom:stage:"
)

var (
	ErrNotFound          = errors.New("execution not found")
	ErrConflict          = errors.New("execution conflict")
	ErrRecoveryRequired  = errors.New("execution recovery required")
	ErrWorkItemAmbiguous = errors.New("work item linkage ambiguous")
)

type Stage string

const (
	Intake         Stage = "intake"
	Specification  Stage = "specification"
	Clarification  Stage = "clarification"
	Plan           Stage = "plan"
	Tasks          Stage = "tasks"
	Implementation Stage = "implementation"
	Review         Stage = "review"
	Evidence       Stage = "evidence"
	Reconciliation Stage = "reconciliation"
	Completion     Stage = "completion"
)

var stages = [...]Stage{Intake, Specification, Clarification, Plan, Tasks, Implementation, Review, Evidence, Reconciliation, Completion}

func Stages() []Stage       { result := make([]Stage, len(stages)); copy(result, stages[:]); return result }
func (s Stage) Valid() bool { return stageIndex(s) >= 0 }
func stageIndex(stage Stage) int {
	for index, candidate := range stages {
		if stage == candidate {
			return index
		}
	}
	return -1
}

type Outcome string

const (
	OutcomePassed  Outcome = "pass"
	OutcomeFailed  Outcome = "fail"
	OutcomeResumed Outcome = "resumed"
	OutcomeFact    Outcome = "fact"
)

type ExecutionStatus string

const (
	ExecutionActive      ExecutionStatus = "active"
	ExecutionInterrupted ExecutionStatus = "interrupted"
	ExecutionCompleted   ExecutionStatus = "completed"
)

type Repository struct{ Key, Path string }
type Project struct {
	ID           string
	Repositories []Repository
}
type Resolver interface {
	Resolve(context.Context, string) (Project, string)
}

type WorkItem struct {
	Provider, Resource, ExternalID string
	URL, State                     string
}
type WorkItems interface {
	Load(context.Context, string, string, string, string, string) (WorkItem, error)
}

type Identity struct{ Product, Version, Revision, SourceState string }
type Reference struct{ Kind, ID, Digest string }
type Transition struct {
	Ledger        *StageLedger `json:"stageLedger,omitempty"`
	Revision      uint64
	From, To      Stage
	Outcome       Outcome
	RequestDigest string
	References    []Reference
	Next          string
	CommittedAt   time.Time
	Provenance    Identity
	Fact          *LifecycleFact `json:"fact,omitempty"`
}
type Terminal struct {
	Status           completion.Status
	ConfirmedEffects []string
}
type ProjectionRecord struct {
	ExecutionRevision   uint64
	Key                 string
	Intended, Confirmed []ProjectionEffect
	Complete            bool
}
type State struct {
	DeliveryDecision           *DeliveryDecision
	ReworkOf                   *ReworkLink
	Binding                    *WorkflowBinding
	StageOrdinal               int
	ExecutionID                string
	FormatVersion              int
	WorkflowVersion            string
	ProjectID, RepositoryKey   string
	WorkItem                   WorkItem
	RuntimeID                  string
	Stage                      Stage
	Revision                   uint64
	Status                     ExecutionStatus
	Transitions                []Transition
	CreatedAt, UpdatedAt       time.Time
	Provenance                 Identity
	Terminal                   *Terminal
	Projections                []ProjectionRecord
	StorageRevision            [sha256.Size]byte
	DeliveryProjectionRevision [sha256.Size]byte `json:"-"`
}
type ProjectionEffectKind string

const (
	CreateStageLabel      ProjectionEffectKind = "create_stage_label"
	AddStageLabel         ProjectionEffectKind = "add_stage_label"
	RemoveStageLabel      ProjectionEffectKind = "remove_stage_label"
	PostTransitionComment ProjectionEffectKind = "post_transition_comment"
)

type ProjectionEffect struct {
	Kind  ProjectionEffectKind `json:"kind"`
	Value string               `json:"value"`
}
type ProjectionObservation struct {
	Provider         string   `json:"provider"`
	Resource         string   `json:"resource"`
	IssueExternalID  string   `json:"issueExternalId"`
	IssueURL         string   `json:"issueUrl"`
	IssueState       string   `json:"issueState"`
	RepositoryLabels []string `json:"repositoryLabels"`
	IssueLabels      []string `json:"issueLabels"`
	CommentPresent   bool     `json:"commentPresent"`
}
type ProjectionErrorKind string

const (
	ProjectionUnavailable     ProjectionErrorKind = "provider_unavailable"
	ProjectionUnauthenticated ProjectionErrorKind = "provider_unauthenticated"
	ProjectionRateLimited     ProjectionErrorKind = "provider_rate_limited"
	ProjectionInvalidResponse ProjectionErrorKind = "provider_invalid_response"
	ProjectionAmbiguous       ProjectionErrorKind = "provider_ambiguous"
)

type ProjectionError struct {
	Kind                                     ProjectionErrorKind
	Retryable, Ambiguous, EffectNotCommitted bool
}

func (e *ProjectionError) Error() string { return string(e.Kind) }

type ProjectionPreview struct {
	ExecutionID       string                 `json:"executionId"`
	ExecutionRevision uint64                 `json:"executionRevision"`
	ProjectionKey     string                 `json:"projectionKey"`
	Stage             Stage                  `json:"stage"`
	LifecycleStage    WorkItemLifecycleStage `json:"lifecycleStage"`
	Label             string                 `json:"label"`
	Comment           string                 `json:"comment"`
	Effects           []ProjectionEffect     `json:"effects"`
	Observation       ProjectionObservation  `json:"observation"`
	ObservationDigest string                 `json:"observationDigest"`
	Digest            string                 `json:"digest"`
}

type Status = completion.Status

const (
	Succeeded        = completion.Success
	Failed           = completion.Failure
	ValidationFailed = completion.ValidationFailure
	Denied           = completion.DeniedAuthority
	Partial          = completion.Partial
	Interrupted      = completion.Interrupted
	Retryable        = completion.RetryableFailure
)

type Result struct {
	Status   Status
	Category string
	State    State
	Preview  *ProjectionPreview
}
type Target struct {
	ProjectSelector, RepositoryKey               string
	WorkItemProvider, WorkItemResource, WorkItem string
	ExecutionID, RuntimeID                       string
}

// Service resolves the existing linked Work Item for the canonical Execution engine.
type Service struct {
	resolver  Resolver
	workItems WorkItems
	source    provenance.Value
}

func New(resolver Resolver, workItems WorkItems, source provenance.Value) Service {
	return Service{resolver: resolver, workItems: workItems, source: source}
}

func (s Service) resolve(ctx context.Context, target Target) (Project, Repository, WorkItem, Result) {
	if s.resolver == nil || s.workItems == nil || !s.source.Valid() || !validText(target.ProjectSelector) || !validText(target.RepositoryKey) || !validWorkItemSelector(target.WorkItem) || target.RuntimeID != "" && !validText(target.RuntimeID) {
		return Project{}, Repository{}, WorkItem{}, result(ValidationFailed, "invalid_execution_input", State{})
	}
	project, category := s.resolver.Resolve(ctx, target.ProjectSelector)
	if category != "" {
		return Project{}, Repository{}, WorkItem{}, result(ValidationFailed, category, State{})
	}
	if !validText(project.ID) {
		return Project{}, Repository{}, WorkItem{}, result(ValidationFailed, "invalid_execution_input", State{})
	}
	var repository Repository
	for _, candidate := range project.Repositories {
		if candidate.Key == target.RepositoryKey {
			if repository.Key != "" {
				return Project{}, Repository{}, WorkItem{}, result(ValidationFailed, "repository_ambiguous", State{})
			}
			repository = candidate
		}
	}
	if repository.Key == "" {
		return Project{}, Repository{}, WorkItem{}, result(ValidationFailed, "repository_not_configured", State{})
	}
	item, err := s.workItems.Load(ctx, project.ID, target.RepositoryKey, target.WorkItemProvider, target.WorkItemResource, target.WorkItem)
	if errors.Is(err, ErrRecoveryRequired) {
		return Project{}, Repository{}, WorkItem{}, result(Failed, "recovery_required", State{})
	}
	if errors.Is(err, ErrWorkItemAmbiguous) {
		return Project{}, Repository{}, WorkItem{}, result(ValidationFailed, "work_item_ambiguous", State{})
	}
	if err != nil || !validWorkItem(item) || item.ExternalID != target.WorkItem || target.WorkItemProvider != "" && item.Provider != target.WorkItemProvider || target.WorkItemResource != "" && item.Resource != target.WorkItemResource {
		return Project{}, Repository{}, WorkItem{}, result(ValidationFailed, "work_item_not_linked", State{})
	}
	return project, repository, item, Result{}
}

func prepareProjectionPreview(state State, lifecycle LifecycleProjection, observation ProjectionObservation) (ProjectionPreview, error) {
	if !validProjectionObservation(state.WorkItem, observation) {
		return ProjectionPreview{}, &ProjectionError{Kind: ProjectionInvalidResponse}
	}
	observation.RepositoryLabels = sorted(observation.RepositoryLabels)
	observation.IssueLabels = sorted(observation.IssueLabels)
	observedStage, legacyStage, err := observedLifecycleStage(observation.IssueLabels)
	if err != nil {
		return ProjectionPreview{}, err
	}
	// Zero stage markers are the bootstrap of a Work Item this Execution never
	// projected. Once any projection was established, losing the marker is drift.
	if observedStage == "" && projectionEstablished(state) {
		return ProjectionPreview{}, ErrRecoveryRequired
	}
	label := lifecycleLabel(lifecycle.Stage)
	key := projectionKey(state.ExecutionID, state.Revision)
	comment := transitionComment(state, key)
	if comment == "" {
		return ProjectionPreview{}, ErrRecoveryRequired
	}
	desired := append([]string{label}, lifecycleFlags(lifecycle.Conditions)...)
	effects := make([]ProjectionEffect, 0, 12)
	for _, wanted := range desired {
		if !contains(observation.RepositoryLabels, wanted) {
			effects = append(effects, ProjectionEffect{Kind: CreateStageLabel, Value: wanted})
		}
		if !contains(observation.IssueLabels, wanted) {
			effects = append(effects, ProjectionEffect{Kind: AddStageLabel, Value: wanted})
		}
	}
	if observedStage != "" && (observedStage != label || legacyStage) {
		effects = append(effects, ProjectionEffect{Kind: RemoveStageLabel, Value: observedStage})
	}
	for _, current := range observation.IssueLabels {
		if validLifecycleFlag(current) && !contains(desired, current) {
			effects = append(effects, ProjectionEffect{Kind: RemoveStageLabel, Value: current})
		}
	}
	if !observation.CommentPresent {
		effects = append(effects, ProjectionEffect{Kind: PostTransitionComment, Value: comment})
	}
	if len(effects) > 16 {
		return ProjectionPreview{}, ErrRecoveryRequired
	}
	for _, effect := range effects {
		if !validEffect(effect) {
			return ProjectionPreview{}, ErrRecoveryRequired
		}
	}
	observationDigest := digest(observation)
	preview := ProjectionPreview{ExecutionID: state.ExecutionID, ExecutionRevision: state.Revision, ProjectionKey: key, Stage: state.Stage, LifecycleStage: lifecycle.Stage, Label: label, Comment: comment, Effects: effects, Observation: observation, ObservationDigest: observationDigest}
	preview.Digest = digest(preview)
	return preview, nil
}

func validProjectionObservation(item WorkItem, observation ProjectionObservation) bool {
	if observation.Provider != item.Provider || observation.Resource != item.Resource || observation.IssueExternalID != item.ExternalID || observation.IssueURL != item.URL || observation.IssueState != "OPEN" && observation.IssueState != "CLOSED" || len(observation.RepositoryLabels) > 100 || len(observation.IssueLabels) > 100 {
		return false
	}
	for _, labels := range [][]string{observation.RepositoryLabels, observation.IssueLabels} {
		seen := make(map[string]struct{}, len(labels))
		for _, label := range labels {
			if label == "" || len(label) > 256 {
				return false
			}
			if _, exists := seen[label]; exists {
				return false
			}
			seen[label] = struct{}{}
		}
	}
	return true
}

func SupportedRuntime(id string) bool { return id == "codex" || id == "claude" }

func ValidState(state State) bool {
	if state.FormatVersion == ConfiguredFormatVersion || state.FormatVersion == ProfileConfiguredFormatVersion || state.FormatVersion == OperationalConfiguredFormatVersion {
		return validConfiguredState(state)
	}
	if state.Binding != nil || state.StageOrdinal != 0 {
		return false
	}
	if !validOpaqueID(state.ExecutionID) || state.FormatVersion != FormatVersion || state.WorkflowVersion != WorkflowVersion || !validText(state.ProjectID) || !validText(state.RepositoryKey) || !validWorkItem(state.WorkItem) || !validText(state.RuntimeID) || !state.Stage.Valid() || state.Revision == 0 || !validExecutionStatus(state.Status) || state.CreatedAt.IsZero() || state.UpdatedAt.Before(state.CreatedAt) || !validIdentity(state.Provenance) || len(state.Transitions) > maxEvents || len(state.Projections) > maxEvents {
		return false
	}
	if uint64(len(state.Transitions))+1 != state.Revision {
		return false
	}
	derivedStage := Intake
	derivedStatus := ExecutionActive
	lastCommittedAt := state.CreatedAt
	for index, event := range state.Transitions {
		if event.Ledger != nil {
			return false
		}
		if event.Revision != uint64(index+2) || !event.From.Valid() || !event.To.Valid() || event.Outcome != OutcomePassed && event.Outcome != OutcomeFailed && event.Outcome != OutcomeResumed && event.Outcome != OutcomeFact || event.RequestDigest == "" || event.CommittedAt.IsZero() || !validIdentity(event.Provenance) || len(event.References) > maxReferences || !validOptionalText(event.Next) {
			return false
		}
		if event.From != derivedStage || !validTransitionShape(event, derivedStage, derivedStatus) {
			return false
		}
		if event.CommittedAt.Before(lastCommittedAt) {
			return false
		}
		lastCommittedAt = event.CommittedAt
		switch event.Outcome {
		case OutcomePassed:
			derivedStage = event.To
			if event.From == Completion {
				derivedStatus = ExecutionCompleted
			}
		case OutcomeFailed:
			derivedStatus = ExecutionInterrupted
		case OutcomeResumed:
			derivedStatus = ExecutionActive
		case OutcomeFact:
			if event.Fact == nil {
				return false
			}
		}
		for _, reference := range event.References {
			if !validReference(reference) {
				return false
			}
		}
		if event.Outcome != OutcomeFact && event.Fact != nil {
			return false
		}
	}
	if state.Stage != derivedStage || state.Status != derivedStatus {
		return false
	}
	if len(state.Transitions) != 0 && !state.UpdatedAt.Equal(lastCommittedAt) {
		return false
	}
	if state.Status == ExecutionCompleted && (state.Stage != Completion || state.Terminal == nil || state.Terminal.Status != completion.Success) {
		return false
	}
	if state.Status == ExecutionCompleted && (len(state.Terminal.ConfirmedEffects) != 1 || state.Terminal.ConfirmedEffects[0] != "local_execution_transition") {
		return false
	}
	if state.Status != ExecutionCompleted && state.Terminal != nil {
		return false
	}
	projectionRevisions := make(map[uint64]struct{}, len(state.Projections))
	for _, record := range state.Projections {
		if record.ExecutionRevision < 2 || record.ExecutionRevision > state.Revision || record.Key != projectionKey(state.ExecutionID, record.ExecutionRevision) || len(record.Intended) > 16 || len(record.Confirmed) > len(record.Intended) || record.Complete != effectsConfirmed(record.Intended, record.Confirmed) {
			return false
		}
		if _, exists := projectionRevisions[record.ExecutionRevision]; exists || hasDuplicateEffects(record.Intended) || hasDuplicateEffects(record.Confirmed) || !validProjectionRecord(state, record) {
			return false
		}
		projectionRevisions[record.ExecutionRevision] = struct{}{}
		for _, effect := range append(cloneEffects(record.Intended), record.Confirmed...) {
			if !validEffect(effect) {
				return false
			}
		}
	}
	return true
}

func validProjectionRecord(state State, record ProjectionRecord) bool {
	projected := projectionSnapshot(state, record.ExecutionRevision)
	lifecycle, err := DeriveLifecycle(projected)
	if err == nil {
		label := lifecycleLabel(lifecycle.Stage)
		desired := append([]string{label}, lifecycleFlags(lifecycle.Conditions)...)
		if validProjectionEffects(record, desired, transitionComment(projected, record.Key), validProjectionLabel) {
			return true
		}
	}

	if state.DeliveryDecision != nil {
		state.DeliveryDecision = nil
		if validProjectionRecord(state, record) {
			return true
		}
	}

	// Persisted S4 records remain readable without rewriting their labels,
	// comments, or Evidence. New previews never create these legacy effects.
	legacyLabel := stagePrefix + string(projected.Stage)
	return validProjectionEffects(record, []string{legacyLabel}, legacyTransitionComment(projected, record.Key), validLegacyStageLabel)
}

func validProjectionEffects(record ProjectionRecord, desired []string, comment string, removable func(string) bool) bool {
	for _, effect := range append(cloneEffects(record.Intended), record.Confirmed...) {
		switch effect.Kind {
		case CreateStageLabel, AddStageLabel:
			if !contains(desired, effect.Value) {
				return false
			}
		case RemoveStageLabel:
			if !removable(effect.Value) || contains(desired, effect.Value) {
				return false
			}
		case PostTransitionComment:
			if effect.Value != comment {
				return false
			}
		}
	}
	return true
}

func projectionSnapshot(state State, revision uint64) State {
	projected := cloneState(state)
	projected.Stage = Intake
	if state.Binding != nil {
		doc, _ := bindingDocument(state)
		projected.Stage = Stage(doc.Definition.Stages[0].ID)
		projected.StageOrdinal = 0
	}
	projected.Status = ExecutionActive
	projected.Revision = revision
	projected.Transitions = projected.Transitions[:revision-1]
	projected.UpdatedAt = projected.CreatedAt
	projected.Terminal = nil
	projected.Projections = nil
	if revision != state.Revision {
		projected.DeliveryDecision = nil
	}
	for _, event := range projected.Transitions {
		if event.Ledger != nil && event.Outcome == OutcomePassed && event.From != event.To {
			projected.StageOrdinal++
		}
		projected.Stage = event.To
		projected.UpdatedAt = event.CommittedAt
		switch event.Outcome {
		case OutcomePassed:
			projected.Status = ExecutionActive
			if event.From == Completion && state.Binding == nil || state.Binding != nil && event.Outcome == OutcomePassed && event.From == event.To {
				projected.Status = ExecutionCompleted
				projected.Terminal = &Terminal{Status: completion.Success, ConfirmedEffects: []string{"local_execution_transition"}}
			}
		case OutcomeFailed:
			projected.Status = ExecutionInterrupted
		case OutcomeResumed:
			projected.Status = ExecutionActive
		}
	}
	return projected
}

func validTransitionShape(event Transition, stage Stage, status ExecutionStatus) bool {
	if event.Outcome == OutcomeFact {
		return event.To == stage && event.Fact != nil
	}
	if status == ExecutionCompleted {
		return false
	}
	if status == ExecutionInterrupted {
		return event.Outcome == OutcomeResumed && event.To == stage
	}
	if event.Outcome == OutcomeResumed {
		return false
	}
	if event.Outcome == OutcomeFailed {
		return event.To == stage
	}
	index := stageIndex(stage)
	if index == len(stages)-1 {
		return event.To == Completion
	}
	return event.To == stages[index+1]
}

func hasDuplicateEffects(effects []ProjectionEffect) bool {
	seen := make(map[ProjectionEffect]struct{}, len(effects))
	for _, effect := range effects {
		if _, exists := seen[effect]; exists {
			return true
		}
		seen[effect] = struct{}{}
	}
	return false
}

func transitionComment(state State, key string) string {
	event := state.Transitions[len(state.Transitions)-1]
	lifecycle, err := DeriveLifecycle(state)
	if err != nil {
		return ""
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "<!-- axiom:workflow-projection:%s -->\n", key)
	fmt.Fprintf(&builder, "Axiom lifecycle projection\n\n- Lifecycle: `%s`\n- Canonical gate: `%s`\n- Outcome: `%s`\n", lifecycle.Stage, state.Stage, event.Outcome)
	if len(event.References) != 0 {
		builder.WriteString("- References:")
		for _, reference := range event.References {
			fmt.Fprintf(&builder, " `%s:%s`", reference.Kind, reference.ID)
		}
		builder.WriteByte('\n')
	}
	if event.Next != "" {
		fmt.Fprintf(&builder, "- Next: %s\n", event.Next)
	}
	fmt.Fprintf(&builder, "\n_Axiom %s · %s · %s · %s_", event.Provenance.Version, event.Provenance.Revision, event.Provenance.SourceState, state.ExecutionID)
	return builder.String()
}
func legacyTransitionComment(state State, key string) string {
	event := state.Transitions[len(state.Transitions)-1]
	var builder strings.Builder
	fmt.Fprintf(&builder, "<!-- axiom:workflow-projection:%s -->\n", key)
	fmt.Fprintf(&builder, "Axiom workflow transition\n\n- Stage: `%s`\n- Outcome: `%s`\n", state.Stage, event.Outcome)
	if len(event.References) != 0 {
		builder.WriteString("- References:")
		for _, reference := range event.References {
			fmt.Fprintf(&builder, " `%s:%s`", reference.Kind, reference.ID)
		}
		builder.WriteByte('\n')
	}
	if event.Next != "" {
		fmt.Fprintf(&builder, "- Next: %s\n", event.Next)
	}
	fmt.Fprintf(&builder, "\n_Axiom %s · %s · %s · %s_", event.Provenance.Version, event.Provenance.Revision, event.Provenance.SourceState, state.ExecutionID)
	return builder.String()
}
func projectionKey(executionID string, revision uint64) string {
	value := sha256.Sum256([]byte(fmt.Sprintf("axiom:workflow-projection:v1\x00%s\x00%d", executionID, revision)))
	return hex.EncodeToString(value[:])
}
func validIdentity(value Identity) bool {
	return value.Product == provenance.Product && validText(value.Version) && validText(value.Revision) && (value.SourceState == string(provenance.Clean) || value.SourceState == string(provenance.Dirty) || value.SourceState == string(provenance.Unknown))
}
func validExecutionStatus(value ExecutionStatus) bool {
	return value == ExecutionActive || value == ExecutionInterrupted || value == ExecutionCompleted
}
func validWorkItem(item WorkItem) bool {
	return validText(item.Provider) && validText(item.Resource) && validText(item.ExternalID) && validText(item.URL) && (item.State == "OPEN" || item.State == "CLOSED")
}
func validReference(reference Reference) bool {
	switch reference.Kind {
	case "artifact", "evidence", "specification", "decision", "plan", "tasks", "pull_request":
		return validText(reference.ID) && validDigest(reference.Digest)
	default:
		return false
	}
}
func validEffect(effect ProjectionEffect) bool {
	if effect.Kind != CreateStageLabel && effect.Kind != AddStageLabel && effect.Kind != RemoveStageLabel && effect.Kind != PostTransitionComment {
		return false
	}
	if effect.Kind == PostTransitionComment {
		return validMultiline(effect.Value, 16*1024)
	}
	return validProjectionLabel(effect.Value)
}
func validStageEffect(value string) bool {
	if !strings.HasPrefix(value, stagePrefix) {
		return false
	}
	stage := WorkItemLifecycleStage(strings.TrimPrefix(value, stagePrefix))
	return stage.Valid() && value == lifecycleLabel(stage)
}
func validProjectionLabel(value string) bool {
	return validStageEffect(value) || validLifecycleFlag(value) || validLegacyStageLabel(value)
}
func validLegacyStageLabel(value string) bool {
	if !strings.HasPrefix(value, stagePrefix) {
		return false
	}
	stage := Stage(strings.TrimPrefix(value, stagePrefix))
	return stage.Valid() && value == stagePrefix+string(stage)
}

// observedLifecycleStage returns the single observed stage marker, or "" when
// the Issue carries none. Invalid, unknown, or multiple markers fail closed;
// callers decide whether zero markers is bootstrap or drift.
func observedLifecycleStage(labels []string) (string, bool, error) {
	var observed string
	legacy := false
	for _, label := range labels {
		if strings.HasPrefix(label, "axiom:") && !strings.HasPrefix(label, stagePrefix) && !validLifecycleFlag(label) {
			return "", false, ErrRecoveryRequired
		}
		if !strings.HasPrefix(label, stagePrefix) {
			continue
		}
		if !validStageEffect(label) && !validLegacyStageLabel(label) {
			return "", false, ErrRecoveryRequired
		}
		if observed != "" {
			return "", false, ErrRecoveryRequired
		}
		observed = label
		legacy = !validStageEffect(label)
	}
	return observed, legacy, nil
}

// projectionEstablished reports whether the Execution ledger proves a stage
// marker was once present on the Issue: a complete record, or any confirmed
// effect other than repository label creation (stage addition precedes removal
// and comment effects in every preview).
func projectionEstablished(state State) bool {
	for _, record := range state.Projections {
		if record.Complete {
			return true
		}
		for _, effect := range record.Confirmed {
			if effect.Kind != CreateStageLabel {
				return true
			}
		}
	}
	return false
}
func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
func validText(value string) bool {
	return validMultiline(value, maxTextBytes) && !strings.Contains(value, "\n")
}
func validOptionalText(value string) bool { return value == "" || validText(value) }
func validMultiline(value string, limit int) bool {
	if value == "" || len(value) > limit || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, current := range value {
		if unicode.IsControl(current) && current != '\n' {
			return false
		}
	}
	return true
}
func validOpaqueID(value string) bool {
	parts := strings.Split(value, "-")
	if len(parts) != 5 || len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		return false
	}
	_, err := hex.DecodeString(strings.Join(parts, ""))
	return err == nil
}
func digest(value any) string {
	wire, _ := json.Marshal(value)
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func containsEffect(values []ProjectionEffect, wanted ProjectionEffect) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func effectsConfirmed(intended, confirmed []ProjectionEffect) bool {
	if len(intended) != len(confirmed) {
		return false
	}
	for _, effect := range intended {
		if !containsEffect(confirmed, effect) {
			return false
		}
	}
	return true
}
func sorted(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
func cloneEffects(values []ProjectionEffect) []ProjectionEffect {
	return append([]ProjectionEffect(nil), values...)
}
func cloneReferences(values []Reference) []Reference { return append([]Reference(nil), values...) }
func cloneState(state State) State {
	if state.Binding != nil {
		b := *state.Binding
		if b.ResolvedProfile != nil {
			frozen := *b.ResolvedProfile
			b.ResolvedProfile = &frozen
		}
		if b.FrozenProfile != nil {
			frozen := *b.FrozenProfile
			b.FrozenProfile = &frozen
		}
		b.Snapshot = append([]byte(nil), b.Snapshot...)
		b.Contexts = make(map[string]ContextObservation, len(state.Binding.Contexts))
		for key, c := range state.Binding.Contexts {
			c.Content = append([]byte(nil), c.Content...)
			b.Contexts[key] = c
		}
		state.Binding = &b
	}
	state.Transitions = append([]Transition(nil), state.Transitions...)
	for index := range state.Transitions {
		if state.Transitions[index].Ledger != nil {
			wire, _ := json.Marshal(state.Transitions[index].Ledger)
			var ledger StageLedger
			_ = json.Unmarshal(wire, &ledger)
			state.Transitions[index].Ledger = &ledger
		}
		state.Transitions[index].References = cloneReferences(state.Transitions[index].References)
		if state.Transitions[index].Fact != nil {
			fact := *state.Transitions[index].Fact
			state.Transitions[index].Fact = &fact
		}
	}
	state.Projections = append([]ProjectionRecord(nil), state.Projections...)
	for index := range state.Projections {
		state.Projections[index].Intended = cloneEffects(state.Projections[index].Intended)
		state.Projections[index].Confirmed = cloneEffects(state.Projections[index].Confirmed)
	}
	if state.Terminal != nil {
		terminal := *state.Terminal
		terminal.ConfirmedEffects = append([]string(nil), state.Terminal.ConfirmedEffects...)
		state.Terminal = &terminal
	}
	return state
}
func result(status Status, category string, state State) Result {
	return Result{Status: status, Category: category, State: state}
}
