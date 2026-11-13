package workflow

import (
	"path/filepath"
	"reflect"
	"slices"
	"time"
)

// SourceCapture is a versioned, nested fact on the exact attempt. Workspace and
// Authority on that attempt freeze repository/ref/base and the human grant.
// Intent precedes index/object writes; CandidateSHA precedes the source CAS.
// An interrupted capture is observed, never replayed, even if the ref stayed at base.
type SourceCapture struct {
	Version          int                 `json:"version"`
	Path             string              `json:"path"`
	RunnerGeneration uint64              `json:"runnerGeneration"`
	PreparedAt       time.Time           `json:"preparedAt"`
	CandidateSHA     string              `json:"candidateSha,omitempty"`
	Observation      *CaptureObservation `json:"observation,omitempty"`
	Recovery         *CaptureObservation `json:"recovery,omitempty"`
}

type CaptureObservation struct {
	Outcome        string    `json:"outcome"` // captured, refused, unknown; internal facts only
	HeadSHA        string    `json:"headSha,omitempty"`
	IndexDigest    string    `json:"indexDigest,omitempty"`
	WorktreeDigest string    `json:"worktreeDigest,omitempty"`
	ObservedAt     time.Time `json:"observedAt"`
}

func (e *Execution) PrepareCapture(in IntegrationInput, path string) error {
	if err := e.fence(in.Generation); err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	if e.Status == RunRecoveryRequired {
		return ErrUnknownOutstanding
	}
	a, ref, ok := e.FindAttempt(in.SourceAttemptID)
	if !ok || a.Status != AttemptSucceeded || a.Capture != nil || !e.RequiresIntegration(in.SourceAttemptID) || a.WorkspaceUncertain {
		return ErrTransition
	}
	w := a.Workspace
	if w == nil || w.RepositoryKey != in.RepositoryKey || w.RepositoryIdentity != in.RepositoryIdentity || w.SourceBranch != in.SourceBranch || w.Worktree != in.Worktree || w.BaseSHA != in.BaseSHA || w.TargetRef != in.TargetRef {
		return ErrTransition
	}
	if !validCapturePath(path, w.Worktree) || a.Authority.RepositoryKey != w.RepositoryKey || !slices.Contains(a.Authority.Effects, "workspace_write") || !slices.Contains(a.Authority.Effects, "validation_command") {
		return ErrUnauthorizedEffect
	}
	spec := e.Workflow.Stages[e.stageIndex(ref.StageID)]
	if len(spec.Commands) == 0 {
		return ErrIntegrationChecksUnavailable
	}
	if spec.HumanGate && e.Stages[e.stageIndex(ref.StageID)].Gate == nil {
		return ErrTransition
	}
	for _, i := range e.Integrations {
		if i.SourceAttemptID == a.AttemptID || i.Status == IntegrationPrepared {
			return ErrIntegrationBusy
		}
	}
	if accepted := e.AcceptedTargetHead(in.RepositoryKey, in.TargetRef); accepted != "" && accepted != in.ExpectedTargetHead {
		return ErrIntegrationStale
	}
	a.Capture = &SourceCapture{Version: 1, Path: path, RunnerGeneration: in.Generation, PreparedAt: in.Now.UTC()}
	e.touch(in.Now)
	return e.Validate()
}

func validCapturePath(path, name string) bool {
	return len(path) <= 4096 && filepath.IsAbs(path) && filepath.Clean(path) == path && filepath.Base(path) == name && boundedText(path, 4096)
}

func (e *Execution) RecordCaptureCandidate(id string, generation uint64, candidate string, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	a, _, ok := e.FindAttempt(id)
	if !ok || a.Capture == nil || a.Capture.RunnerGeneration != generation || a.Capture.CandidateSHA != "" || a.Capture.Observation != nil || !ValidObjectID(candidate) || candidate == a.Workspace.BaseSHA || e.Cancel != nil {
		return ErrTransition
	}
	a.Capture.CandidateSHA = candidate
	e.touch(now)
	return e.Validate()
}

func (e *Execution) ObserveCapture(id string, generation uint64, observation CaptureObservation) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	a, _, ok := e.FindAttempt(id)
	if !ok || a.Capture == nil {
		return ErrTransition
	}
	if a.Capture.Observation == nil {
		a.Capture.Observation = &observation
	} else {
		if generation <= a.Capture.RunnerGeneration || a.Capture.Recovery != nil {
			return ErrTransition
		}
		a.Capture.Recovery = &observation
	}
	e.touch(observation.ObservedAt)
	return e.Validate()
}

// PreservesCaptureHistory is used by the canonical store, including direct Save.
func PreservesCaptureHistory(prior, next *SourceCapture) bool {
	if prior == nil {
		return next == nil || next.CandidateSHA == "" && next.Observation == nil && next.Recovery == nil
	}
	if next == nil {
		return false
	}
	if prior.CandidateSHA == "" && next.CandidateSHA != "" && next.Observation != nil {
		return false
	}
	if next.Recovery != nil && prior.Observation == nil {
		return false
	}
	frozen := *next
	if prior.Recovery == nil {
		frozen.Recovery = nil
	}
	if prior.CandidateSHA == "" && prior.Observation == nil {
		frozen.CandidateSHA = ""
	}
	if prior.Observation == nil {
		frozen.Observation = nil
	}
	return reflect.DeepEqual(*prior, frozen)
}

func validateCapture(e Execution, a Attempt) error {
	c := a.Capture
	if c == nil {
		return nil
	}
	if !slices.Contains(a.Authority.Effects, "workspace_write") || !slices.Contains(a.Authority.Effects, "validation_command") {
		return invalid("capture authority")
	}
	if c.Version != 1 || a.Workspace == nil || !e.RequiresIntegration(a.AttemptID) || a.Status != AttemptSucceeded || !validCapturePath(c.Path, a.Workspace.Worktree) || c.RunnerGeneration == 0 || c.RunnerGeneration > e.RunnerGeneration || c.PreparedAt.IsZero() {
		return invalid("capture identity")
	}
	if c.CandidateSHA != "" && (!ValidObjectID(c.CandidateSHA) || c.CandidateSHA == a.Workspace.BaseSHA) {
		return invalid("capture candidate")
	}
	if c.Recovery != nil && c.Observation == nil {
		return invalid("capture recovery without observation")
	}
	for _, o := range []*CaptureObservation{c.Observation, c.Recovery} {
		if o == nil {
			continue
		}
		if o.ObservedAt.IsZero() || o.HeadSHA != "" && !ValidObjectID(o.HeadSHA) || o.IndexDigest != "" && !validPrefixedDigest(o.IndexDigest) || o.WorktreeDigest != "" && !validPrefixedDigest(o.WorktreeDigest) {
			return invalid("capture observation")
		}
		switch o.Outcome {
		case "captured":
			if c.CandidateSHA == "" || o.HeadSHA != c.CandidateSHA || o.IndexDigest == "" || o.WorktreeDigest == "" {
				return invalid("capture success")
			}
		case "refused":
			if o.HeadSHA != a.Workspace.BaseSHA || o.IndexDigest == "" || o.WorktreeDigest == "" {
				return invalid("capture refusal")
			}
		case "unknown":
		default:
			return invalid("capture outcome")
		}
	}
	return nil
}

// CheckCaptureFence is a fenced store checkpoint immediately before a Git
// mutation. It cannot authorize a different capture or a replay after outcome.
func (e *Execution) CheckCaptureFence(id string, generation uint64, now time.Time) error {
	if err := e.fence(generation); err != nil {
		return err
	}
	if e.Cancel != nil {
		return ErrCancelled
	}
	a, _, ok := e.FindAttempt(id)
	if !ok || a.Capture == nil || a.Capture.RunnerGeneration != generation || a.Capture.Observation != nil || a.WorkspaceUncertain {
		return ErrTransition
	}
	e.touch(now)
	return e.Validate()
}

func uncertainCapture(c *SourceCapture) bool {
	return c != nil && (c.Observation != nil && c.Observation.Outcome != "captured" || c.Recovery != nil && c.Recovery.Outcome != "captured")
}
