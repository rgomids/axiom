package workflow

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Integration records one principal-controlled integration of an agent
// attempt's isolated branch into a principal-owned integration ref of a bound
// Repository (Spec 015 "Parallel work and integration", Plan TD-03). It is a
// nested Execution fact, not another store or lifecycle: the principal runner
// persists the intent under its fence before any Git effect and then the
// observed outcome. MergeSHA is only the intermediate merge result; FinalSHA
// exists only once required post-merge checks passed and the target ref was
// observed at that commit. A not_integrated record never claims a rollback:
// the target ref was simply observed unchanged by this integration.
type Integration struct {
	IntegrationID      string             `json:"integrationId"`
	RepositoryKey      string             `json:"repositoryKey"`
	SourceAttemptID    string             `json:"sourceAttemptId"`
	StageID            string             `json:"stageId"`
	AgentID            string             `json:"agentId"`
	SourceBranch       string             `json:"sourceBranch"`
	Worktree           string             `json:"worktree"`
	BaseSHA            string             `json:"baseSha"`
	TargetRef          string             `json:"targetRef"`
	ExpectedTargetHead string             `json:"expectedTargetHead"`
	CandidateSHA       string             `json:"candidateSha"`
	MergeSHA           string             `json:"mergeSha,omitempty"`
	FinalSHA           string             `json:"finalSha,omitempty"`
	ObservedTargetHead string             `json:"observedTargetHead,omitempty"`
	Checks             []IntegrationCheck `json:"checks,omitempty"`
	Status             IntegrationStatus  `json:"status"`
	Diagnostic         string             `json:"diagnostic,omitempty"`
	RunnerGeneration   uint64             `json:"runnerGeneration"`
	PreparedAt         time.Time          `json:"preparedAt"`
	EndedAt            *time.Time         `json:"endedAt,omitempty"`
	Resolution         *Disposition       `json:"resolution,omitempty"`
}

// IntegrationCheck is the observed result of one required post-merge check:
// a Workflow stage command re-run on the merged tree.
type IntegrationCheck struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exitCode"`
}

type IntegrationStatus string

const (
	IntegrationPrepared      IntegrationStatus = "prepared"
	IntegrationIntegrated    IntegrationStatus = "integrated"
	IntegrationNotIntegrated IntegrationStatus = "not_integrated"
	// IntegrationUnknown means the target ref effect cannot be established; it
	// is never retried automatically and blocks further work like an unknown
	// attempt.
	IntegrationUnknown IntegrationStatus = "unknown"
)

// Integration check statuses.
const (
	CheckPassed     = "passed"
	CheckFailed     = "failed"
	CheckTerminated = "terminated"
)

const MaxIntegrations = 32

var (
	// ErrIntegrationBusy serializes integration per Execution: one prepared
	// integration at a time.
	ErrIntegrationBusy = errors.New("execution_integration_in_progress")
	// ErrIntegrationStale reports an expected target head that is not the
	// recorded or observed one.
	ErrIntegrationStale = errors.New("execution_integration_stale_target")
	// ErrIntegrationChecksUnavailable refuses an integration whose source stage
	// declares no post-merge check: integration needs check Evidence.
	ErrIntegrationChecksUnavailable = errors.New("execution_integration_checks_unavailable")
	// ErrIntegrationConflict and ErrIntegrationTargetCheckedOut are reported
	// by the Git boundary: an unresolved merge conflict, and a target branch
	// checked out in some working tree.
	ErrIntegrationConflict         = errors.New("execution_integration_merge_conflict")
	ErrIntegrationTargetCheckedOut = errors.New("execution_integration_target_checked_out")
)

// IntegrationInput is the principal's integration intent. Every SHA is the
// value the principal observed immediately before this commit.
type IntegrationInput struct {
	IntegrationID      string
	Generation         uint64
	SourceAttemptID    string
	RepositoryKey      string
	RepositoryIdentity string
	SourceBranch       string
	Worktree           string
	BaseSHA            string
	TargetRef          string
	ExpectedTargetHead string
	CandidateSHA       string
	Now                time.Time
}

// IntegrationResult is the observed outcome of a prepared integration.
type IntegrationResult struct {
	Status             IntegrationStatus
	MergeSHA           string
	FinalSHA           string
	ObservedTargetHead string
	Checks             []IntegrationCheck
	Diagnostic         string
	Now                time.Time
}

// ValidObjectID accepts a full lowercase SHA-1 or SHA-256 Git object name.
func ValidObjectID(v string) bool {
	if len(v) != 40 && len(v) != 64 {
		return false
	}
	for _, c := range v {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// ValidBranchRef accepts a bounded refs/heads/ name whose components are
// tokens; it excludes "..", "@{", lock suffixes and every shell/glob byte.
func ValidBranchRef(v string) bool {
	name, ok := strings.CutPrefix(v, "refs/heads/")
	if !ok || len(v) > 200 || name == "" {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if !ValidToken(part) || strings.Contains(part, "..") || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") || strings.Contains(part, ":") {
			return false
		}
	}
	return true
}

// ValidWorktreeName is one path component under the principal-authorized
// worktree root; it can never traverse or name the root itself.
func ValidWorktreeName(v string) bool {
	return ValidToken(v) && !strings.HasPrefix(v, ".") && !strings.Contains(v, ":")
}

func unresolvedUnknownIntegration(i Integration) bool {
	return i.Status == IntegrationUnknown && i.Resolution == nil
}

func (e *Execution) integration(id string) *Integration {
	for k := range e.Integrations {
		if e.Integrations[k].IntegrationID == id {
			return &e.Integrations[k]
		}
	}
	return nil
}

// AcceptedTargetHead is the last FinalSHA recorded for a repository target,
// or "" when this Execution has integrated nothing there.
func (e Execution) AcceptedTargetHead(repositoryKey, targetRef string) string {
	head := ""
	for _, i := range e.Integrations {
		if i.Status == IntegrationIntegrated && i.RepositoryKey == repositoryKey && i.TargetRef == targetRef {
			head = i.FinalSHA
		}
	}
	return head
}

// PrepareIntegration persists the principal's intent before any Git effect.
// Only the runner fence holder can call it; a child attempt holds no fence and
// no message changes it. The source must be a succeeded attempt authorized to
// write that Repository and run its stage's validation commands.
func (e *Execution) PrepareIntegration(in IntegrationInput) error {
	if err := e.fence(in.Generation); err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	if e.Status == RunRecoveryRequired {
		return ErrUnknownOutstanding
	}
	if !ValidToken(in.IntegrationID) || e.integration(in.IntegrationID) != nil || len(e.Integrations) >= MaxIntegrations {
		return fmt.Errorf("%w: integration identity", ErrTransition)
	}
	if !ValidBranchRef(in.SourceBranch) || !ValidBranchRef(in.TargetRef) || in.SourceBranch == in.TargetRef || !ValidWorktreeName(in.Worktree) {
		return fmt.Errorf("%w: integration refs", ErrTransition)
	}
	if !ValidObjectID(in.BaseSHA) || !ValidObjectID(in.ExpectedTargetHead) || !ValidObjectID(in.CandidateSHA) || in.CandidateSHA == in.BaseSHA {
		return fmt.Errorf("%w: integration object identity", ErrTransition)
	}
	bound := false
	for _, r := range e.Repositories {
		bound = bound || r.Key == in.RepositoryKey && r.Identity == in.RepositoryIdentity
	}
	if !bound {
		return fmt.Errorf("%w: repository binding", ErrTransition)
	}
	attempt, ref, ok := e.FindAttempt(in.SourceAttemptID)
	if !ok || attempt.Status != AttemptSucceeded {
		return fmt.Errorf("%w: source attempt has no validated result", ErrTransition)
	}
	if attempt.Authority.RepositoryKey != in.RepositoryKey || !slices.Contains(attempt.Authority.Effects, "workspace_write") || !slices.Contains(attempt.Authority.Effects, "validation_command") {
		return fmt.Errorf("%w: source attempt is not authorized to write and validate this Repository", ErrUnauthorizedEffect)
	}
	i := e.stageIndex(ref.StageID)
	spec := e.Workflow.Stages[i]
	if len(spec.Commands) == 0 {
		return ErrIntegrationChecksUnavailable
	}
	if spec.HumanGate && e.Stages[i].Gate == nil {
		return fmt.Errorf("%w: human gate pending", ErrTransition)
	}
	for _, other := range e.Integrations {
		if other.Status == IntegrationPrepared {
			return ErrIntegrationBusy
		}
		if other.RepositoryKey == in.RepositoryKey && other.TargetRef == in.TargetRef && other.CandidateSHA == in.CandidateSHA && other.Status == IntegrationIntegrated {
			return fmt.Errorf("%w: candidate already integrated", ErrTransition)
		}
		// The principal asserts which branch carries which attempt's work; once
		// recorded, that association cannot be reused for another attempt.
		if (other.SourceBranch == in.SourceBranch || other.Worktree == in.Worktree || other.SourceAttemptID == in.SourceAttemptID) && (other.SourceAttemptID != in.SourceAttemptID || other.SourceBranch != in.SourceBranch || other.Worktree != in.Worktree) {
			return fmt.Errorf("%w: branch, worktree and attempt association mismatch", ErrTransition)
		}
	}
	if accepted := e.AcceptedTargetHead(in.RepositoryKey, in.TargetRef); accepted != "" && accepted != in.ExpectedTargetHead {
		return ErrIntegrationStale
	}
	e.Integrations = append(e.Integrations, Integration{
		IntegrationID: in.IntegrationID, RepositoryKey: in.RepositoryKey, SourceAttemptID: in.SourceAttemptID,
		StageID: ref.StageID, AgentID: ref.AgentID, SourceBranch: in.SourceBranch, Worktree: in.Worktree,
		BaseSHA: in.BaseSHA, TargetRef: in.TargetRef, ExpectedTargetHead: in.ExpectedTargetHead, CandidateSHA: in.CandidateSHA,
		Status: IntegrationPrepared, RunnerGeneration: in.Generation, PreparedAt: in.Now.UTC(),
	})
	e.touch(in.Now)
	return e.Validate()
}

// FinishIntegration records the observed outcome of the fence holder's own
// prepared integration. integrated requires every stage check to have passed
// on the merge result and the target observed at FinalSHA == MergeSHA.
func (e *Execution) FinishIntegration(id string, generation uint64, r IntegrationResult) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	i := e.integration(id)
	if i == nil || i.Status != IntegrationPrepared || i.RunnerGeneration != generation {
		return ErrTransition
	}
	if r.Status != IntegrationIntegrated && r.Status != IntegrationNotIntegrated && r.Status != IntegrationUnknown {
		return ErrTransition
	}
	at := r.Now.UTC()
	i.Status, i.MergeSHA, i.FinalSHA, i.ObservedTargetHead = r.Status, r.MergeSHA, r.FinalSHA, r.ObservedTargetHead
	i.Checks, i.Diagnostic, i.EndedAt = append([]IntegrationCheck(nil), r.Checks...), clip(r.Diagnostic), &at
	e.touch(r.Now)
	return e.Validate()
}

// MarkIntegrationUnknown records a prepared integration left by a lost
// runner: its target ref effect has no correlated observation.
func (e *Execution) MarkIntegrationUnknown(id string, generation uint64, diagnostic string, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	i := e.integration(id)
	if i == nil || i.Status != IntegrationPrepared {
		return ErrTransition
	}
	at := now.UTC()
	i.Status, i.Diagnostic, i.EndedAt = IntegrationUnknown, clip(diagnostic), &at
	e.touch(now)
	return e.Validate()
}

// ResolveIntegration records the human abandonment of an unknown
// integration. The uncertain effect stays recorded and is never replayed.
func (e *Execution) ResolveIntegration(id, actor string, now time.Time) error {
	i := e.integration(id)
	if i == nil || !unresolvedUnknownIntegration(*i) || !humanActor(actor) {
		return ErrTransition
	}
	i.Resolution = &Disposition{Actor: actor, Decision: DispositionAbandoned, RecordedAt: now.UTC()}
	e.touch(now)
	return e.Validate()
}

func validateIntegrations(e Execution) error {
	if len(e.Integrations) > MaxIntegrations {
		return invalid("integrations")
	}
	seen := map[string]bool{}
	association := map[string]string{}
	prepared := 0
	for _, i := range e.Integrations {
		if !ValidToken(i.IntegrationID) || seen[i.IntegrationID] {
			return invalid("integration identity")
		}
		seen[i.IntegrationID] = true
		bound := i.SourceAttemptID + "\x00" + i.SourceBranch + "\x00" + i.Worktree
		for _, key := range []string{"attempt:" + i.SourceAttemptID, "branch:" + i.SourceBranch, "worktree:" + i.Worktree} {
			if previous, ok := association[key]; ok && previous != bound {
				return invalid("integration %q association", i.IntegrationID)
			}
			association[key] = bound
		}
		attempt, ref, ok := e.FindAttempt(i.SourceAttemptID)
		if !ok || ref.StageID != i.StageID || ref.AgentID != i.AgentID || attempt.Authority.RepositoryKey != i.RepositoryKey {
			return invalid("integration %q source", i.IntegrationID)
		}
		if !ValidBranchRef(i.SourceBranch) || !ValidBranchRef(i.TargetRef) || i.SourceBranch == i.TargetRef || !ValidWorktreeName(i.Worktree) {
			return invalid("integration %q refs", i.IntegrationID)
		}
		if !ValidObjectID(i.BaseSHA) || !ValidObjectID(i.ExpectedTargetHead) || !ValidObjectID(i.CandidateSHA) || i.CandidateSHA == i.BaseSHA {
			return invalid("integration %q objects", i.IntegrationID)
		}
		for _, v := range []string{i.MergeSHA, i.FinalSHA, i.ObservedTargetHead} {
			if v != "" && !ValidObjectID(v) {
				return invalid("integration %q objects", i.IntegrationID)
			}
		}
		if i.RunnerGeneration == 0 || i.RunnerGeneration > e.RunnerGeneration || i.PreparedAt.IsZero() || !boundedText(i.Diagnostic, MaxDiagnosticBytes) {
			return invalid("integration %q facts", i.IntegrationID)
		}
		spec := e.Workflow.Stages[e.stageIndex(i.StageID)]
		if len(i.Checks) > len(spec.Commands) {
			return invalid("integration %q checks", i.IntegrationID)
		}
		passed := len(i.Checks) == len(spec.Commands)
		for k, c := range i.Checks {
			if c.ID != spec.Commands[k].ID || (c.Status != CheckPassed && c.Status != CheckFailed && c.Status != CheckTerminated) || (c.Status == CheckPassed) != (c.ExitCode == 0) {
				return invalid("integration %q check", i.IntegrationID)
			}
			passed = passed && c.Status == CheckPassed
		}
		switch i.Status {
		case IntegrationPrepared:
			prepared++
			if i.EndedAt != nil || i.MergeSHA != "" || i.FinalSHA != "" || i.ObservedTargetHead != "" || len(i.Checks) != 0 {
				return invalid("integration %q prepared facts", i.IntegrationID)
			}
		case IntegrationIntegrated:
			if i.EndedAt == nil || !passed || i.MergeSHA == "" || i.FinalSHA != i.MergeSHA || i.ObservedTargetHead != i.FinalSHA {
				return invalid("integration %q integrated facts", i.IntegrationID)
			}
		case IntegrationNotIntegrated, IntegrationUnknown:
			if i.EndedAt == nil || i.FinalSHA != "" || i.Diagnostic == "" {
				return invalid("integration %q outcome facts", i.IntegrationID)
			}
		default:
			return invalid("integration %q status", i.IntegrationID)
		}
		if i.Resolution != nil && (i.Status != IntegrationUnknown || !humanActor(i.Resolution.Actor) || i.Resolution.Decision != DispositionAbandoned || i.Resolution.RecordedAt.IsZero()) {
			return invalid("integration %q resolution", i.IntegrationID)
		}
	}
	if prepared > 1 {
		return invalid("integrations are not serialized")
	}
	return nil
}
