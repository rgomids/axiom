package main

import (
	"context"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

type deliveryDecisionRequest struct {
	PacketDigest      string               `json:"packetDigest"`
	PublicationDigest string               `json:"publicationDigest"`
	Outcome           string               `json:"outcome"`
	Actor             string               `json:"actor"`
	Reason            string               `json:"reason"`
	Authority         workflow.Reference   `json:"authority"`
	Evidence          []workflow.Reference `json:"evidence"`
}

func (s lifecycleService) deliveryDecision(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	if in.PlanFile == "" {
		d, err := stores.executions.LoadDeliveryDecision(ctx, t.state.ProjectID, t.state.ExecutionID)
		if err != nil {
			return s.stageError(err, nil)
		}
		return s.stageSuccess("delivery_decision", "Immutable human delivery decision", []string{d.Digest()}, decisionNext(d), d)
	}
	var request deliveryDecisionRequest
	if err := readDeliveryRequest(in.PlanFile, &request); err != nil {
		return s.stageError(err, nil)
	}
	p, err := stores.executions.LoadDeliveryPacket(ctx, t.state.ProjectID, request.PacketDigest)
	if err != nil {
		return s.stageError(err, nil)
	}
	current, err := workflow.ComposeDeliveryPacket(ctx, t.state, deliveryEvidence{stateRoot: s.stateRoot, stores: stores, repository: t.repository.Path})
	if err != nil {
		return s.stageError(err, nil)
	}
	ref, _ := workflow.DeliveryPacketDigest(current)
	if ref != request.PacketDigest {
		return s.stageError(workflow.ErrDelivery, nil)
	}
	var publication *workflow.DeliveryPublication
	if request.PublicationDigest != "" {
		record, e := stores.executions.LoadDeliveryPublication(ctx, t.state.ProjectID, request.PublicationDigest)
		if e != nil {
			return s.stageError(e, nil)
		}
		publication = &record
	}
	artifacts, err := local.NewArtifactStore(s.stateRoot)
	if err != nil {
		return s.stageError(err, nil)
	}
	d, err := workflow.PrepareDeliveryDecision(ctx, t.state, p, publication, request.Outcome, request.Actor, request.Reason, request.Authority, request.Evidence, local.NewWorkflowReferenceValidator(artifacts), t.repository.Path)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" && !in.AuthorizeLocal && !in.AuthorizeHuman {
		return s.stagePreview(d.Digest(), "Review exact human delivery decision", "Record only with explicit human and local authority over this preview", d)
	}
	d, err = workflow.ApplyDeliveryDecision(ctx, stores.executions, t.state, d, in.PreviewDigest, in.AuthorizeLocal, in.AuthorizeHuman)
	if err != nil {
		return s.stageError(err, d)
	}
	return s.stageSuccess("delivery_decision", "Human delivery decision recorded", []string{d.Digest()}, decisionNext(d), d)
}
func decisionNext(d workflow.DeliveryDecision) string {
	if d.Outcome == "reject" {
		return "prepare separately authorized rework using the recorded corrective reason and Plan"
	}
	return "delivery accepted; closure, merge and release remain separate"
}
func (s lifecycleService) deliveryRework(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	var input workflow.ReworkInput
	if err := readDeliveryRequest(in.PlanFile, &input); err != nil {
		return s.stageError(err, nil)
	}
	d, err := stores.executions.LoadDeliveryDecision(ctx, t.state.ProjectID, t.state.ExecutionID)
	if err != nil {
		return s.stageError(err, nil)
	}
	provider, resource, id, ok := cli.ParseWorkItemSelector(in.WorkItem)
	if !ok {
		return s.stageError(workflow.ErrDelivery, nil)
	}
	target := workflow.Target{ProjectSelector: t.state.ProjectID, RepositoryKey: in.Repository, WorkItemProvider: provider, WorkItemResource: resource, WorkItem: id}
	p, err := s.workflows.PreviewDeliveryRework(ctx, t.state, d, input, target)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.PreviewDigest == "" && !in.AuthorizeLocal && !in.AuthorizeExecution {
		return s.stagePreview(p.Digest(), "Review new linked Execution and independent authority", "Authorize the new Execution only; all required workflow gates restart", p)
	}
	next, err := workflow.ApplyDeliveryRework(ctx, stores.executions, t.state, p, in.PreviewDigest, in.AuthorizeLocal, in.AuthorizeExecution)
	if err != nil {
		return s.stageFailure(completion.Failure, "delivery_rework_recovery_required", "New active Execution is not confirmed", "Inspect the original and replay the same authorized rework proposal; never allocate another ID", p)
	}
	return s.stageSuccess("delivery_rework", "New active Execution confirmed", []string{next.ExecutionID, p.Digest()}, "Start at the first required gate with independent authority", map[string]any{"execution": next, "reworkOf": next.ReworkOf, "previewDigest": p.Digest()})
}
