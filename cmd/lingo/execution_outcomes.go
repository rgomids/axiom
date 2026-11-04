package main

import (
	"context"

	"github.com/rgomids/axiom/internal/gitworkspace"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/workflow"
)

// repositoryOutcomes observes current Evidence without writing state, spawning
// a Runtime or calling a Provider. Errors stay bounded and contain no paths.
func (c executionComposition) repositoryOutcomes(ctx context.Context, x workflow.Execution) []workflow.RepositoryOutcome {
	outcomes := x.RepositoryOutcomes()
	artifacts, err := local.NewArtifactStore(c.s.stateRoot)
	validator := local.NewWorkflowReferenceValidator(artifacts)
	for k := range outcomes {
		o := &outcomes[k]
		path := ""
		for _, bound := range x.Repositories {
			if bound.Key == o.RepositoryKey {
				for _, r := range c.project.Repositories {
					if r.Key == bound.Key && repositoryIdentity(r) == bound.Identity {
						path = r.Path
					}
				}
			}
		}
		// An unclaimed Repository has nothing to observe, so its path is not read.
		if x.Claims(*o) && (path == "" || local.CheckPublicationDirectory(path) != nil) {
			o.Blockers = append(o.Blockers, "repository_binding_mismatch")
		}
		for _, r := range o.Results {
			if r.Current && r.Status == workflow.AttemptSucceeded {
				a, ref, _ := x.FindAttempt(r.AttemptID)
				for _, artifact := range r.Artifacts {
					if err != nil || validator.ObserveExecutionArtifact(ctx, x, ref, *a, artifact) != nil {
						o.Blockers = append(o.Blockers, r.AttemptID+":"+artifact.Name+":artifact_unverified")
					}
				}
			}
		}
		latest := map[string]int{}
		for j, i := range o.Integrations {
			latest[i.TargetRef] = j
		}
		for j, i := range o.Integrations {
			if i.Status == workflow.IntegrationIntegrated && (path == "" || gitworkspace.ObserveIntegration(ctx, path, i, latest[i.TargetRef] == j) != nil) {
				o.Blockers = append(o.Blockers, i.IntegrationID+":git_result_unverified")
			}
		}
		if ctx.Err() != nil {
			o.Blockers = append(o.Blockers, "observation_cancelled")
		}
		o.ReadinessObserved = true
		o.Ready = len(o.Blockers) == 0
	}
	return outcomes
}
