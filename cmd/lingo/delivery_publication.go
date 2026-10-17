package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/githubissues"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

type deliveryEffects struct {
	git    gitworkspace.DeliveryGit
	github githubissues.Adapter
}

func (d deliveryEffects) ObserveDelivery(ctx context.Context, p workflow.DeliveryPreview) (workflow.DeliveryObservation, error) {
	o, err := d.git.ObserveDeliveryGit(ctx, p)
	if err != nil {
		return o, err
	}
	o.Candidates, o.Conclusive, err = d.github.ObserveDeliveryPR(ctx, p)
	return o, err
}
func (d deliveryEffects) PushDelivery(ctx context.Context, p workflow.DeliveryPreview) error {
	return d.git.PushDelivery(ctx, p)
}
func (d deliveryEffects) CreateDeliveryPR(ctx context.Context, p workflow.DeliveryPreview) error {
	return d.github.CreateDeliveryPR(ctx, p)
}

func readDeliveryRequest(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	wire, err := io.ReadAll(io.LimitReader(f, workflow.MaxDeliveryBytes+1))
	if err != nil {
		return err
	}
	return strictjson.Decode(wire, value, workflow.MaxDeliveryBytes)
}
func (s lifecycleService) deliveryMetadata(ctx context.Context, projectID, repository string, packet workflow.DeliveryPacket) (json.RawMessage, error) {
	resolved := s.installation.Resolve(ctx, projectID)
	if resolved.Status != local.ResolutionFound {
		return nil, workflow.ErrDelivery
	}
	portable, err := local.NewPortableStore(filepath.Dir(resolved.Project.Source))
	if err != nil {
		return nil, err
	}
	snapshot, _, err := portable.InspectWorkflows(ctx, resolved.Project.Slug)
	if err != nil {
		return nil, err
	}
	state := snapshot.Project().State()
	refs, _ := state.Policies.Value()
	docs := [][]byte{}
	for _, ref := range refs {
		b, err := readWorkflowContext(ctx, resolved.Project, state, ref)
		if err != nil {
			return nil, err
		}
		docs = append(docs, b)
	}
	policy, err := workitem.DecodeMetadataPolicies(docs)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		policy = &workitem.MetadataPolicy{}
	}
	digest, _ := workflow.DeliveryPacketDigest(packet)
	return githubissues.DeliveryMetadata(repository, digest, *policy, nil)
}
func (s lifecycleService) deliveryPublication(ctx context.Context, stores stageStores, t stageTarget, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	publisher := workflow.DeliveryPublisher{Store: stores.executions}
	var target workflow.DeliveryTarget
	var record workflow.DeliveryPublication
	var err error
	inspecting := in.Operation == "delivery-inspect" || in.Operation == "delivery-reconcile"
	if inspecting {
		record, err = stores.executions.LoadDeliveryPublication(ctx, t.state.ProjectID, in.Record)
		if err != nil {
			return s.stageError(err, nil)
		}
		target = record.Preview.Target
		if record.Preview.Packet.ExecutionID != t.state.ExecutionID {
			return s.stageError(workflow.ErrDelivery, nil)
		}
	} else if err = readDeliveryRequest(in.PlanFile, &target); err != nil {
		return s.stageError(err, nil)
	}
	if target.Kind == "github" {
		git, e := gitworkspace.NewDeliveryGit(ctx, t.repository.Path)
		if e != nil {
			return s.stageError(e, nil)
		}
		github, e := githubissues.New("")
		if e != nil {
			return s.stageError(e, nil)
		}
		publisher.Effects = deliveryEffects{git, github}
	}
	if inspecting {
		if in.Operation == "delivery-reconcile" {
			if !in.AuthorizeLocal || in.PreviewDigest != record.Preview.Digest() {
				return s.stageError(workflow.ErrDelivery, nil)
			}
			record, err = publisher.Reconcile(ctx, t.state, record)
		}
	} else {
		packet, e := workflow.ComposeDeliveryPacket(ctx, t.state, deliveryEvidence{stateRoot: s.stateRoot, stores: stores, repository: t.repository.Path})
		if e != nil {
			return s.stageError(e, nil)
		}
		if target.Kind == "github" {
			metadata, e := s.deliveryMetadata(ctx, t.state.ProjectID, target.Repository, packet)
			if e != nil {
				return s.stageError(e, nil)
			}
			target.Metadata = metadata
		}
		if in.Operation == "delivery-apply" && in.PreviewDigest != "" {
			prior, loadErr := stores.executions.LoadDeliveryPublication(ctx, t.state.ProjectID, in.PreviewDigest)
			if loadErr == nil {
				if !reflect.DeepEqual(prior.Preview.Target, target) {
					return s.stageError(workflow.ErrDelivery, nil)
				}
				record, err = publisher.Apply(ctx, t.state, prior.Preview, workflow.DeliveryAuthority{Digest: in.PreviewDigest, Local: in.AuthorizeLocal, Push: in.AuthorizePush, PR: in.AuthorizePR})
				if err != nil {
					return s.stageError(err, record)
				}
				if record.Status != "success" {
					return s.stageFailure(completion.Failure, "delivery_recovery_required", "Reconcile recorded effects", record.Next, record)
				}
				return s.stageSuccess("delivery_publication", "Confirmed delivery artifact", []string{record.Artifact}, record.Next, record)
			}
		}
		if in.Operation == "delivery-prepare" {
			git, e := gitworkspace.NewDeliveryGit(ctx, t.repository.Path)
			if e != nil {
				return s.stageError(e, nil)
			}
			proposal, e := git.PreviewDeliveryCommit(ctx, packet, target)
			if e != nil {
				return s.stageError(e, nil)
			}
			if in.PreviewDigest == "" && !in.AuthorizeLocal {
				return s.stagePreview(proposal.Digest(), "Review local commit preparation", "Authorize only this local commit and head ref; review a new publication preview afterwards", proposal)
			}
			updated, e := git.ApplyDeliveryCommit(ctx, packet, proposal, in.PreviewDigest, in.AuthorizeLocal)
			if e != nil {
				return s.stageError(e, nil)
			}
			return s.stageSuccess("delivery_commit_prepared", "Local reviewed commit prepared", []string{updated.HeadCommit}, "Use returned target to request a new publication preview", updated)
		}
		preview, e := publisher.Preview(ctx, t.state, packet, target)
		if e != nil {
			return s.stageError(e, nil)
		}
		if in.Operation == "delivery-preview" {
			return s.stagePreview(preview.Digest(), "Exact delivery effect envelope", "Review packet, destination and separate push/PR effects", preview)
		}
		if _, e = stores.executions.LoadDeliveryPacket(ctx, t.state.ProjectID, preview.PacketDigest); e != nil {
			return s.stageError(e, nil)
		}
		record, err = publisher.Apply(ctx, t.state, preview, workflow.DeliveryAuthority{Digest: in.PreviewDigest, Local: in.AuthorizeLocal, Push: in.AuthorizePush, PR: in.AuthorizePR})
	}
	if err != nil {
		return s.stageFailure(completion.Failure, "delivery_recovery_required", "Delivery is not confirmed", record.Next, record)
	}
	if record.Status != "success" {
		return s.stageFailure(completion.Failure, "delivery_recovery_required", "Inspect and reconcile recorded effects", record.Next, record)
	}
	return s.stageSuccess("delivery_publication", "Confirmed delivery artifact", []string{record.Artifact}, record.Next, record)
}
