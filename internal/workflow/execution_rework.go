package workflow

import (
	"encoding/json"
	"reflect"
	"time"
)

// ResultRejection is a human disposition of exact successful process facts,
// never a rewrite of those facts or an authority to close, merge or deliver.
type ResultRejection struct {
	StageID string `json:"stageId"`
	ResultReview
}

// RejectedSource carries provenance only. No invocation or authority is copied.
type RejectedSource struct {
	ExecutionID     string `json:"executionId"`
	StageID         string `json:"stageId"`
	RepositoryKey   string `json:"repositoryKey"`
	ResultDigest    string `json:"resultDigest"`
	RejectionDigest string `json:"rejectionDigest"`
}

func (e Execution) RejectedSource() (RejectedSource, error) {
	if e.Rejection == nil || e.Validate() != nil {
		return RejectedSource{}, ErrTransition
	}
	wire, err := json.Marshal(e.Rejection)
	if err != nil {
		return RejectedSource{}, ErrTransition
	}
	return RejectedSource{ExecutionID: e.ExecutionID, StageID: e.Rejection.StageID, RepositoryKey: e.WorkItem.RepositoryKey, ResultDigest: e.Rejection.ResultDigest, RejectionDigest: DigestText(wire)}, nil
}

// RejectResult stops progression without changing any observed result. The
// application must authenticate the human and observe retained Evidence first.
func (e *Execution) RejectResult(stage, actor, digest string, evidence ArtifactRef, now time.Time) error {
	if !humanActor(actor) || e.Rejection != nil || e.Cancel != nil || e.Status == RunRecoveryRequired || hasActiveAttempt(*e) {
		return ErrTransition
	}
	for _, i := range e.Integrations {
		if i.Status == IntegrationPrepared {
			return ErrTransition
		}
	}
	for _, d := range e.Deliveries {
		if d.Status == "prepared" {
			return ErrTransition
		}
	}
	i := e.stageIndex(stage)
	if i < 0 || e.Stages[i].Review != nil {
		return ErrTransition
	}
	actual, err := e.StageResultDigest(stage)
	if err != nil || digest != actual || !validRejectionEvidence(evidence, e.WorkItem.RepositoryKey) || now.IsZero() {
		return ErrTransition
	}
	e.Rejection = &ResultRejection{StageID: stage, ResultReview: ResultReview{Actor: actor, ResultDigest: digest, Evidence: evidence, RecordedAt: now.UTC()}}
	e.touch(now)
	return e.Validate()
}

func validRejectionEvidence(a ArtifactRef, repo string) bool {
	return ValidToken(a.RetainedID) && a.Name == "human-review" && a.RepositoryKey == repo && validPrefixedDigest(a.Digest) && a.Size >= 0
}

func validateRework(e Execution) error {
	if r := e.Rejection; r != nil {
		digest, err := e.StageResultDigest(r.StageID)
		if err != nil || digest != r.ResultDigest || !humanActor(r.Actor) || r.RecordedAt.IsZero() || !validRejectionEvidence(r.Evidence, e.WorkItem.RepositoryKey) || hasActiveAttempt(e) || e.Cancel != nil || e.Stages[e.stageIndex(r.StageID)].Review != nil {
			return ErrExecutionInvalid
		}
		for _, s := range e.Stages {
			for _, t := range s.Tasks {
				for _, a := range t.Attempts {
					if unresolvedUnknown(a) {
						return ErrExecutionInvalid
					}
				}
			}
		}
		for _, i := range e.Integrations {
			if i.Status == IntegrationPrepared || unresolvedUnknownIntegration(i) {
				return ErrExecutionInvalid
			}
		}
		for _, d := range e.Deliveries {
			if d.Status == "prepared" || d.Status == "unknown" {
				return ErrExecutionInvalid
			}
		}
	}
	if r := e.ReworkOf; r != nil {
		if !ValidExecutionID(r.ExecutionID) || r.ExecutionID == e.ExecutionID || !ValidToken(r.StageID) || r.RepositoryKey != e.WorkItem.RepositoryKey || !validPrefixedDigest(r.ResultDigest) || !validPrefixedDigest(r.RejectionDigest) {
			return ErrExecutionInvalid
		}
	}
	return nil
}

// ValidReworkStart is checked by the canonical store while holding its project
// lock. The source stays untouched; the destination must be a pristine run.
func ValidReworkStart(source, next Execution) bool {
	link, err := source.RejectedSource()
	if err != nil || next.ReworkOf == nil || *next.ReworkOf != link || source.ProjectID != next.ProjectID || source.WorkItem != next.WorkItem || !reflect.DeepEqual(source.Repositories, next.Repositories) {
		return false
	}
	clean, err := NewExecution(NewExecutionInput{ExecutionID: next.ExecutionID, ProjectID: next.ProjectID, WorkItem: next.WorkItem, Repositories: next.Repositories, WorkflowID: next.Workflow.ID, Revision: next.Workflow.Revision, Stages: next.Workflow.Stages, Selections: next.Selections, Now: next.CreatedAt})
	if err != nil {
		return false
	}
	clean.ReworkOf = next.ReworkOf
	if next.AttemptPolicy == SpawnBoundaryPolicy {
		clean.AttemptPolicy = next.AttemptPolicy
	}
	return reflect.DeepEqual(clean, next)
}
