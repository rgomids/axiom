package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/githubissues"
	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
	"github.com/rgomids/axiom/internal/strictjson"
	"github.com/rgomids/axiom/internal/workflow"
	"github.com/rgomids/axiom/internal/workitem"
)

// deliveryRemote composes the existing Git and Provider effect boundaries;
// it owns no state, authorization, retry or recovery lifecycle.
type deliveryRemote struct {
	git      gitworkspace.DeliveryGit
	provider githubissues.Adapter
}

func (d deliveryRemote) ObserveDelivery(ctx context.Context, p workflow.DeliveryPreview) (workflow.DeliveryObservation, error) {
	token, err := d.provider.DeliveryGitCredential(ctx)
	if err != nil {
		return workflow.DeliveryObservation{}, err
	}
	git, err := d.git.WithGitHubCredential(token)
	if err != nil {
		return workflow.DeliveryObservation{}, err
	}
	o, err := git.ObserveDeliveryGit(ctx, p)
	if err != nil {
		return o, err
	}
	candidates, conclusive, err := d.provider.ObserveDeliveryPR(ctx, p)
	o.Candidates = candidates
	o.Conclusive = o.Conclusive && conclusive && err == nil
	return o, err
}
func (d deliveryRemote) PushDelivery(ctx context.Context, p workflow.DeliveryPreview) error {
	token, err := d.provider.DeliveryGitCredential(ctx)
	if err != nil {
		return errors.Join(workflow.ErrDeliveryRefused, err)
	}
	git, err := d.git.WithGitHubCredential(token)
	if err != nil {
		return errors.Join(workflow.ErrDeliveryRefused, err)
	}
	return git.PushDelivery(ctx, p)
}
func (d deliveryRemote) CreateDeliveryPR(ctx context.Context, p workflow.DeliveryPreview) error {
	return d.provider.CreateDeliveryPR(ctx, p)
}

func newDeliveryRemote(ctx context.Context, path string) (workflow.DeliveryEffects, error) {
	git, err := gitworkspace.NewDeliveryGit(ctx, path)
	if err != nil {
		return nil, err
	}
	provider, err := githubissues.New(os.Getenv("AXIOM_GH_BIN"))
	if err != nil {
		return nil, err
	}

	return deliveryRemote{git: git, provider: provider}, nil
}

func (s lifecycleService) executionDelivery(ctx context.Context, in cli.WorkflowStageInput) cli.WorkflowStageResponse {
	denied := func() cli.WorkflowStageResponse {
		return s.stageFailure(completion.DeniedAuthority, "delivery_authority_required", "Exact delivery authority or binding refused", "Inspect Execution and review the exact request; never replay an unknown effect", nil)
	}
	if os.Getenv("AXIOM_ATTEMPT_ID") != "" || os.Getenv("AXIOM_EXECUTION_ID") != "" {
		return denied()
	}
	selector := in.Project
	if selector == "" {
		id, failure := s.EffectiveProject(ctx, "")
		if id == "" {
			return cli.WorkflowStageResponse{Result: failure}
		}
		selector = id
	}
	if blocked := s.gate(ctx, selector, projectapp.AdmitExecutionAdvance); blocked != nil {
		return cli.WorkflowStageResponse{Result: *blocked}
	}
	resolved := s.installation.Resolve(ctx, selector)
	if resolved.Status != local.ResolutionFound {
		return denied()
	}
	store, err := local.NewExecutionStore(s.stateRoot)
	if err != nil {
		return s.stageError(err, nil)
	}
	x, err := store.Load(ctx, resolved.Project.ID, in.Execution)
	if err != nil {
		return s.stageError(err, nil)
	}
	c := executionComposition{s: s, store: store, project: resolved.Project}
	var r workflow.DeliveryRequest
	if in.InspectDelivery >= 0 {
		if in.DeliveryRequest != "" || in.InspectDelivery >= len(x.Deliveries) {
			return denied()
		}
		r = x.Deliveries[in.InspectDelivery].Request
	} else if strictjson.Decode([]byte(in.DeliveryRequest), &r, workflow.MaxExecutionBytes) != nil || !r.Valid() {
		return denied()
	} else if r.Target.PR && strings.TrimSpace(string(r.Target.Metadata)) == "{}" {
		// No public surface builds the PR metadata preview, so an empty object
		// is normalized deterministically here; the returned request and its
		// grant digest bind the filled metadata.
		metadata, err := githubissues.DeliveryMetadata(r.Target.Repository, r.Target.DiffDigest, workitem.MetadataPolicy{}, nil)
		if err != nil {
			return denied()
		}
		r.Target.Metadata = metadata
	}
	if r.ExecutionID != x.ExecutionID || r.ProjectID != x.ProjectID {
		return denied()
	}
	check := func(ctx context.Context, current workflow.Execution) error {
		if err := c.admit(ctx, current); err != nil {
			return err
		}
		bound := false
		for _, b := range current.Repositories {
			bound = bound || b.Key == r.RepositoryKey && b.Identity == r.RepositoryIdentity
		}
		if !bound {
			return workflow.ErrUnauthorizedEffect
		}
		latest := s.installation.Resolve(ctx, current.ProjectID)
		path := ""
		frozen := ""
		for _, b := range c.project.Repositories {
			if b.Key == r.RepositoryKey && repositoryIdentity(b) == r.RepositoryIdentity {
				frozen = b.Path
			}
		}
		for _, b := range latest.Project.Repositories {
			if b.Key == r.RepositoryKey && repositoryIdentity(b) == r.RepositoryIdentity && b.Path == frozen {
				path = b.Path
			}
		}
		if path == "" || local.CheckPublicationDirectory(path) != nil || current.Cancel != nil && in.InspectDelivery < 0 {
			return workflow.ErrUnauthorizedEffect
		}
		if in.InspectDelivery < 0 {
			for _, o := range c.repositoryOutcomes(ctx, current) {
				if !o.Ready {
					return workflow.ErrUnauthorizedEffect
				}
			}
		}
		return ctx.Err()
	}
	if err = check(ctx, x); err != nil {
		return denied()
	}
	if in.PreviewDigest == "" {
		if in.InspectDelivery >= 0 || in.AuthorizeLocal || in.AuthorizeHuman || in.AuthorizePush || in.AuthorizePR || r.Revision != x.Revision {
			return denied()
		}
		return s.stageSuccess("delivery_request", "Exact request for operator review", nil, "Review target and digest; explicit authority is required for the current phase", map[string]any{"request": r, "grantDigest": r.Digest(), "deliveryAuthorized": false})
	}
	grant := workflow.DeliveryGrant{Digest: in.PreviewDigest, Actor: executionActor("human:"), Push: in.AuthorizePush, PR: in.AuthorizePR}
	if !in.AuthorizeHuman || !in.AuthorizeLocal || !grant.Covers(r) || in.InspectDelivery < 0 && r.Revision != x.Revision {
		return denied()
	}
	path := ""
	for _, b := range resolved.Project.Repositories {
		if b.Key == r.RepositoryKey {
			path = b.Path
		}
	}
	factory := s.deliveryEffects
	if factory == nil {
		factory = newDeliveryRemote
	}
	effects, err := factory(ctx, path)
	if err != nil {
		return s.stageError(err, nil)
	}
	if in.InspectDelivery >= 0 {
		x, err = c.engine(nil).InspectDelivery(ctx, x.ProjectID, x.ExecutionID, in.InspectDelivery, grant, effects, check)
	} else {
		x, err = c.engine(nil).Deliver(ctx, r, grant, effects, check)
	}
	if err != nil {
		if errors.Is(err, workflow.ErrUnauthorizedEffect) || errors.Is(err, workflow.ErrDeliveryRefused) {
			return denied()
		}
		return s.stageFailure(completion.Failure, "delivery_recovery_required", "Delivery did not confirm", "Inspect the exact recorded target under its original grant; no automatic replay", executionPayload(x, nil))
	}
	summary := "Execution delivery phase recorded"
	if in.InspectDelivery >= 0 {
		summary = "Delivery observation recorded; outcome remains unknown"
	}
	return s.stageSuccess("execution_delivery", summary, []string{"execution:" + x.ExecutionID}, "Unknown observations require separate human disposition; merge/release/closure remain separate", executionPayload(x, nil))
}
