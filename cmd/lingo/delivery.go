package main

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"reflect"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/detailartifact"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

type deliveryEvidence struct {
	stateRoot  string
	stores     stageStores
	repository string
}

func (d deliveryEvidence) Reference(ctx context.Context, state workflow.State, ref workflow.Reference) error {
	store, err := local.NewArtifactStore(d.stateRoot)
	if err != nil {
		return err
	}
	return local.NewWorkflowReferenceValidator(store).Validate(ctx, state.ExecutionID, d.repository, ref)
}

func (d deliveryEvidence) Artifact(ctx context.Context, state workflow.State, stage workflowdefinition.Stage, output workflowdefinition.Output, ref workflow.Reference) (detailartifact.Artifact, error) {
	store, err := local.NewArtifactStore(d.stateRoot)
	if err != nil {
		return detailartifact.Artifact{}, err
	}
	validator := local.NewWorkflowReferenceValidator(store)
	if err = validator.ValidateStageOutput(ctx, state, d.repository, stage, output, workflowdefinition.Validator{Kind: "artifact-schema", PolicyRef: "builtin-sdd-v1"}, ref); err != nil {
		return detailartifact.Artifact{}, err
	}
	for _, criterion := range stage.CompletionCriteria {
		if criterion.OutputRef != output.ID {
			continue
		}
		for _, policy := range stage.Validators {
			if policy.ID == criterion.ValidatorRef {
				if err = validator.ValidateStageOutput(ctx, state, d.repository, stage, output, policy, ref); err != nil {
					return detailartifact.Artifact{}, err
				}
			}
		}
	}
	return store.Read(ctx, ref.ID)
}

func (d deliveryEvidence) Stage(ctx context.Context, state workflow.State, ordinal int) (workflow.DeliveryStage, error) {
	result := workflow.DeliveryStage{}
	doc, issues := workflowdefinition.Decode(state.Binding.Snapshot)
	if len(issues) > 0 || ordinal >= len(doc.Definition.Stages) {
		return result, workflow.ErrDelivery
	}
	stage := doc.Definition.Stages[ordinal]
	parent, err := d.stores.graphs.FindStageGraph(ctx, state.ProjectID, state.ExecutionID, stage.ID)
	if errors.Is(err, local.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		prefix := state
		prefix.Stage = workflow.Stage(stage.ID)
		prefix.StageOrdinal = ordinal
		run, err := loadStageSingle(ctx, d.stores.singles, prefix)
		if err != nil {
			return result, err
		}
		if run == nil {
			result.Blocker = "single_stage_run_missing"
			return result, nil
		}
		result.SingleDigest = hex.EncodeToString(run.StorageRevision[:])
		if len(run.Attempts) > 0 {
			copy := run.Attempts[len(run.Attempts)-1]
			result.SingleAttempt = &copy
		}
		// Readiness describes another dispatch, so a settled success is expected here.
		pending, blocker := executiongraph.SingleStageReadiness(*run)
		if pending || blocker != string(executiongraph.AttemptSucceeded) || len(run.Attempts) == 0 {
			result.Blocker = "single_stage_incomplete"
			return result, nil
		}
		last := run.Attempts[len(run.Attempts)-1]
		if last.Status != executiongraph.AttemptSucceeded || last.ClassificationReason != executiongraph.ResultRecorded {
			result.Blocker = "single_stage_incomplete"
			return result, nil
		}
		authority, err := d.stores.singles.LoadAuthority(ctx, *run, run.Authorities[len(run.Authorities)-1])
		if err != nil || executiongraph.ValidateSingleStageBindingAuthority(*run, authority) != nil {
			result.Blocker = "single_stage_authority_unverified"
			return result, nil
		}
		coordinationStore, e := local.NewCoordinationStore(d.stateRoot, state.ProjectID)
		if e != nil {
			return result, e
		}
		records, e := (coordination.Reader{Store: coordinationStore}).List(ctx, coordination.Filter{ParentID: state.ExecutionID, ChildID: state.ExecutionID, AttemptID: last.AttemptID})
		if e != nil {
			return result, e
		}
		projected := make([]executiongraph.AttemptRecord, 0, len(records))
		for _, record := range records {
			projected = append(projected, coordination.ClassifierRecord(record))
		}
		latest, verified := executiongraph.LatestAttemptRecord(executiongraph.AttemptCorrelation{ParentID: state.ExecutionID, ChildID: state.ExecutionID, AttemptID: last.AttemptID, GraphRevision: run.Binding.WorkflowRevision}, projected)
		if !verified || latest.Digest != last.QualifyingRecordRef || latest.Outcome != "succeeded" {
			result.Blocker = "single_stage_coordination_unverified"
			return result, nil
		}
		return result, nil
	}
	if err != nil {
		return result, err
	}
	graph, err := d.stores.graphs.LoadSuccessor(ctx, state.ProjectID, parent)
	if err != nil {
		return result, err
	}
	latestIntegration := ""
	result.Blocker = "integration_pending"
	for _, child := range graph.Children {
		result.Children = append(result.Children, workflow.DeliveryChild{ChildID: child.ExecutionID, Attempts: append([]executiongraph.SuccessorAttempt{}, child.Attempts...)})
		if child.Envelope.IntegrationOwner && len(child.Attempts) > 0 {
			latestIntegration = child.Attempts[len(child.Attempts)-1].AttemptID
		}
	}
	records, err := canonicalRecords(ctx, d.stateRoot, graph)
	if err != nil {
		return result, err
	}
	outcomes, err := d.stores.executions.LoadStageOutcomes(ctx, state.ProjectID, state.ExecutionID)
	if err != nil {
		return result, err
	}
	for ref, outcome := range outcomes {
		if outcome.Rollup.IntegrationAttemptID != latestIntegration || outcome.StageOrdinal != ordinal || outcome.StageID != stage.ID || outcome.Rollup.ParentID != parent {
			continue
		}
		if result.Outcome != nil {
			return result, workflow.ErrDelivery
		}
		receipt, err := d.stores.graphs.LoadIntegrationReceipt(ctx, state.ProjectID, parent, outcome.IntegrationReceiptRef)
		if err != nil {
			return result, err
		}
		observed, err := executiongraph.BuildStageOutcome(graph, outcome.IntegrationReceiptRef, receipt, outcome.RecordedAt)
		if err != nil || !reflect.DeepEqual(observed, outcome) || outcome.ExecutionRevision > state.Revision {
			return result, workflow.ErrDelivery
		}
		result.Blocker = ""
		if outcome.Rollup.Status != "success" {
			copy := outcome
			result.Outcome = &copy
			result.OutcomeRef = ref
			result.GraphDigest = outcome.GraphDigest
			result.ReceiptRef = outcome.IntegrationReceiptRef
			continue
		}
		for _, child := range graph.Children {
			if child.Envelope.IntegrationOwner || child.Envelope.Optional && waivedChild(receipt, child.ExecutionID) {
				continue
			}
			if len(child.Attempts) == 0 {
				return result, workflow.ErrDelivery
			}
			last := child.Attempts[len(child.Attempts)-1]
			record, ok := executiongraph.LatestAttemptRecord(executiongraph.AttemptCorrelation{ParentID: parent, ChildID: child.ExecutionID, AttemptID: last.AttemptID, GraphRevision: graph.Parent.GraphRevision}, records)
			if !ok || record.Outcome != "succeeded" || record.Digest != last.QualifyingRecordRef {
				return result, workflow.ErrDelivery
			}
		}
		manager, projection, err := (gitIntegration{repository: d.repository}).manager(ctx, graph)
		if err != nil {
			return result, err
		}
		integration, err := integrationProjection(projection)
		if err != nil {
			return result, err
		}
		tree, diffDigest, err := manager.DeliveryTree(ctx, integration)
		if err != nil || executiongraph.IntegrationResultDigest(tree) != outcome.IntegrationResultReference {
			return result, workflow.ErrDelivery
		}
		result.ResultTree, result.DiffDigest = tree, diffDigest
		copy := outcome
		result.Outcome = &copy
		result.OutcomeRef = ref
		result.GraphDigest = outcome.GraphDigest
		result.ReceiptRef = outcome.IntegrationReceiptRef
	}
	return result, nil
}

func waivedChild(receipt executiongraph.IntegrationReceipt, child string) bool {
	for _, w := range receipt.OptionalWaivers {
		if w.ChildID == child {
			return true
		}
	}
	return false
}

func (s lifecycleService) deliveryPacket(ctx context.Context, stores stageStores, target stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	if in.Record != "" {
		if in.AuthorizeLocal || in.PreviewDigest != "" {
			return s.stageFailure(completion.ValidationFailure, "conflicting_packet_inputs", "Packet inspection is read-only", "Omit retention authority when inspecting a digest", nil)
		}
		packet, err := stores.executions.LoadDeliveryPacket(ctx, target.state.ProjectID, in.Record)
		if err != nil {
			return s.stageError(err, nil)
		}
		if packet.ExecutionID != target.state.ExecutionID {
			return s.stageError(workflow.ErrDelivery, nil)
		}
		return s.stageSuccess("delivery_packet", "Immutable delivery packet", []string{"delivery-packet:" + in.Record}, packet.Next, map[string]any{"packetDigest": in.Record, "packet": packet})
	}
	packet, err := workflow.ComposeDeliveryPacket(ctx, target.state, deliveryEvidence{stateRoot: s.stateRoot, stores: stores, repository: target.repository.Path})
	if err != nil {
		return s.stageError(err, nil)
	}
	digest, err := workflow.DeliveryPacketDigest(packet)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.AuthorizeLocal || in.PreviewDigest != "" {
		if !in.AuthorizeLocal || in.PreviewDigest != digest {
			return s.stageFailure(completion.DeniedAuthority, "delivery_packet_authority_required", "Packet retention needs the exact preview digest and local authority", "Review current packet readiness before retaining", nil)
		}
		if _, err = stores.executions.RecordDeliveryPacket(ctx, target.state, packet); err != nil {
			return s.stageError(err, nil)
		}
	}
	return s.stageSuccess("delivery_packet", "Canonical delivery readiness", []string{"delivery-packet:" + digest}, packet.Next, map[string]any{"packetDigest": digest, "packet": packet})
}
