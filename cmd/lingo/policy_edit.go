package main

import (
	"github.com/rgomids/axiom/internal/cli"
	"github.com/rgomids/axiom/internal/completion"
	"github.com/rgomids/axiom/internal/projectapp"
)

func (s lifecycleService) policyEditResult(out projectapp.PolicyEditResult) cli.Result {
	if out.Outcome == projectapp.EditRejected {
		return editFailure(out.Failure, s.provenance)
	}
	preview := out.Proposal.Preview()
	facts := completion.Facts{Completed: true}
	message, next := "Project Runtime policy preview ready", "Review policy, Workflow impact and readiness; repeat the same draft with --project-id "+preview.ProjectID+" --preview-digest "+preview.Digest+" --authorize-local only after exact confirmation"
	switch out.Outcome {
	case projectapp.EditDenied, projectapp.EditConflicted:
		facts = completion.Facts{AuthorityDenied: true}
		message, next = "Project Runtime policy authority or compatibility refused", "Resolve exact Workflow references or drift, then review and authorize a fresh preview"
	case projectapp.EditPublished, projectapp.EditUnchanged:
		message, next = "Project Runtime policy published and read back", "Review observed policy and project validate before any future Execution"
		if out.Outcome == projectapp.EditUnchanged {
			message = "Project Runtime policy unchanged and read back"
		}
		if out.ReadBack == nil || out.ReadBack.Status != "confirmed" {
			facts = completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}
			message, next = "Project Runtime policy commit confirmed; read-back is unverified or changed", "Inspect current Project state and recovery; preserve the confirmed outcome and do not retry the mutation blindly"
		} else if out.ReadBack.ReadinessChanged {
			next = "Policy bytes are confirmed; local readiness snapshots changed after preview, so review the observed blockers"
		}
	case projectapp.EditPartial:
		facts = completion.Facts{RequestedEffectConfirmed: true, SecondaryFailure: true}
		message, next = "Portable Runtime policy committed; local bookkeeping or recovery remains incomplete", "Run axiom recovery inspect; preserve state and obtain separate recovery authority, without reapplying this edit"
	case projectapp.EditRecovery:
		facts = completion.Facts{Failed: true}
		message, next = "Project Runtime policy publication requires recovery", "Run axiom recovery inspect; commit state is uncertain, so do not retry blindly"
	case projectapp.EditFailedPublish:
		facts = completion.Facts{Failed: true}
		message, next = "Project Runtime policy publication failed", "Inspect preserved state and request a fresh preview; unsupported sources remain refused"
	}
	result := canonicalCompletion(facts, message, []string{"project:" + preview.ProjectID}, next, s.provenance)
	result.PolicyEdit = &projectapp.PolicyEditReport{Outcome: out.Outcome, Preview: preview, Publication: out.Publication.Status, LocalCommitted: out.Publication.LocalCommitted || out.Publication.Status == projectapp.EditPublicationApplied, ReadBack: out.ReadBack}
	return result
}
