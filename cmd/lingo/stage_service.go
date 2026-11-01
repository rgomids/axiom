package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/workflow"
)

var _ cli.WorkflowStageService = lifecycleService{}

// WorkflowStage routes every operational consumer to the canonical Execution.
// Legacy Graph/Single ledgers are never opened, including for inspection.
func (s lifecycleService) WorkflowStage(ctx context.Context, in cli.WorkflowStageInput) (out cli.WorkflowStageResponse) {
	defer func() {
		if d := out.Result.Diagnostic; d != nil {
			switch in.Operation {
			case "execution-run":
				d.Operation = "workflow run"
			case "status", "cancel", "retry", "reconcile":
				d.Operation = "workflow stage " + in.Operation
			case "coordination-list", "coordination-show", "coordination-publish":
				d.Operation = "workflow coordination " + strings.TrimPrefix(in.Operation, "coordination-")
			default:
				// Never invent an operation for delivery or unsupported input.
				out.Result.Diagnostic = nil
			}
		}
	}()
	if ctx.Err() != nil {
		return s.stageFailure(completion.Interrupted, "workflow_cancelled", "Operation cancelled", "Inspect Execution status", nil)
	}
	switch in.Operation {
	case "execution-run", "status", "cancel", "retry", "reconcile":
		return s.executionStage(ctx, in)
	case "coordination-publish":
		return s.stagePublish(ctx, in)
	case "coordination-list", "coordination-show", "delivery-packet":
		return s.executionRead(ctx, in)
	}
	return s.stageFailure(completion.ValidationFailure, "unsupported_execution_operation", "This operation is not implemented for canonical Executions", "Inspect workflow run/status; delivery effects require the approved S3 implementation", nil)
}

func (s lifecycleService) executionRead(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	selector := in.Project
	if selector == "" {
		id, failure := s.EffectiveProject(ctx, "")
		if id == "" {
			return cli.WorkflowStageResponse{Result: failure}
		}
		selector = id
	}
	if blocked := s.gate(ctx, selector, projectapp.AdmitExecutionStatus); blocked != nil {
		return cli.WorkflowStageResponse{Result: *blocked}
	}
	project := s.installation.Resolve(ctx, selector)
	if project.Status != local.ResolutionFound {
		return s.stageFailure(completion.ValidationFailure, project.Category, "Project could not be resolved", "Inspect project context", nil)
	}
	store, err := local.NewExecutionStore(s.stateRoot)
	if err != nil {
		return s.stageError(err, nil)
	}
	c := executionComposition{s: s, store: store, project: project.Project}
	x, err := store.Load(ctx, project.Project.ID, in.Execution)
	if err != nil {
		return c.failure(err, nil)
	}
	if in.Repository != "" && in.Repository != x.WorkItem.RepositoryKey || in.ExpectedRevision != 0 && in.ExpectedRevision != x.Revision {
		return s.stageFailure(completion.ValidationFailure, "execution_target_mismatch", "Execution selectors differ", "Use selectors from canonical status", nil)
	}
	if in.Operation == "delivery-packet" {
		if in.AuthorizeLocal || in.PreviewDigest != "" || in.Record != "" {
			return s.stageFailure(completion.ValidationFailure, "unsupported_execution_operation", "Canonical readiness is read-only", "Delivery publication is S3; no legacy packet is read or retained", nil)
		}
		blockers := []string{}
		if x.Status != workflow.RunCompleted {
			blockers = append(blockers, "execution_not_completed")
		}
		artifacts := []workflow.ArtifactRef{}
		for _, stage := range x.Stages {
			for _, task := range stage.Tasks {
				if len(task.Attempts) == 0 {
					blockers = append(blockers, stage.StageID+":"+task.AgentID+":attempt_missing")
					continue
				}
				a := task.Attempts[len(task.Attempts)-1]
				if a.Status != workflow.AttemptSucceeded {
					blockers = append(blockers, stage.StageID+":"+task.AgentID+":"+string(a.Status))
				}
				for _, effect := range a.Effects {
					if effect.State == workflow.EffectUnknown {
						blockers = append(blockers, a.AttemptID+":unknown_effect")
					}
				}
				artifacts = append(artifacts, a.Artifacts...)
			}
		}
		return s.stageSuccess("execution_readiness", "Canonical Execution readiness", []string{"execution:" + x.ExecutionID}, "Readiness grants no delivery authority; publication is implemented in S3", map[string]any{"execution": x.ExecutionID, "revision": x.Revision, "ready": len(blockers) == 0, "blockers": blockers, "artifacts": artifacts, "deliveryAuthorized": false})
	}
	records := []workflow.Message{}
	for _, m := range x.Messages {
		if in.Record != "" && m.ID != in.Record || in.Attempt != "" && m.SenderAttempt != in.Attempt || in.Child != "" && m.Sender != in.Child || in.Kind != "" && string(m.Kind) != in.Kind {
			continue
		}
		records = append(records, m)
	}
	if in.Operation == "coordination-show" && len(records) != 1 {
		return s.stageError(coordination.ErrRecordNotFound, nil)
	}
	if in.Wait != 0 || in.After != 0 {
		return s.stageFailure(completion.ValidationFailure, "unsupported_execution_operation", "Legacy stream cursors are not Execution message selectors", "Inspect canonical messages without --wait/--after", nil)
	}
	return s.stageSuccess("execution_messages", "Attempt-owned Execution messages", []string{"execution:" + x.ExecutionID}, "Messages grant no authority", map[string]any{"records": records, "revision": x.Revision})
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
		{coordination.ErrChildContext, completion.DeniedAuthority, "child_context_required", "Publish only from a running stage child's own task context"},
		{coordination.ErrRecordNotFound, completion.ValidationFailure, "record_not_found", "List the child's canonical records first"},
		{coordination.ErrInvalidWait, completion.ValidationFailure, "invalid_wait", "Use a positive --wait no longer than 30s"},
		{local.ErrConflict, completion.RetryableFailure, "temporarily_unavailable", "A concurrent operation holds the lease; retry the read first"},
		{local.ErrRecoveryRequired, completion.Failure, "recovery_required", "Run recovery inspect and apply the reviewed plan"},
	}
	for _, r := range rules {
		if errors.Is(err, r.target) {
			return s.stageFailure(r.status, r.category, "Stage operation refused", r.next, payload)
		}
	}
	return s.stageFailure(completion.Failure, "stage_operation_failed", "Stage operation failed", "Inspect status; never repeat an authorized mutation blindly", payload)
}

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

// decisionNext describes retained historical delivery facts without loading an
// old ledger or granting an effect. Delivery mutation is S3-owned.
func decisionNext(d workflow.DeliveryDecision) string {
	if d.Outcome == "reject" {
		return "prepare separately authorized rework using the recorded corrective reason and Plan"
	}
	return "delivery accepted; closure, merge and release remain separate"
}
