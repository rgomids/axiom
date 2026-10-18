package projectapp

import "context"

type PolicyEditResult struct {
	Outcome     EditOutcome
	Failure     EditFailure
	Proposal    PolicyEditProposal
	Publication EditPublicationResult
}

func (p PolicyEditProposal) publicationRequest() EditPublicationRequest {
	e := p.edit
	dependencies := p.dependencies
	return EditPublicationRequest{
		PolicyDependencies: &dependencies,
		ProjectID:          e.preview.ProjectID, Slug: e.preview.Slug,
		PortableDestination: e.preview.PortableDestination, LocalDestination: e.preview.LocalDestination,
		PortableExpected: append([]byte(nil), e.observedManifest...), PortableNext: append([]byte(nil), e.manifest...),
		LocalExpected: append([]byte(nil), e.observedLocalWire...), LocalNext: append([]byte(nil), e.localWire...),
	}
}

func ApplyPolicyEdit(ctx context.Context, ports PolicyEditPorts, publisher EditPublisher, intent EditIntent, authority EditAuthority) PolicyEditResult {
	if failure := ValidateEditAuthority(authority); failure != EditOK {
		return PolicyEditResult{Outcome: EditRejected, Failure: failure}
	}
	proposal, failure := PreviewPolicyEdit(ctx, ports, intent)
	if failure != EditOK {
		return PolicyEditResult{Outcome: EditRejected, Failure: failure}
	}
	result := PolicyEditResult{Outcome: EditPreviewed, Proposal: proposal}
	if !authority.Supplied() {
		return result
	}
	if authority.ProjectID != proposal.preview.ProjectID || authority.PreviewDigest != proposal.preview.Digest {
		result.Outcome = EditDenied
		return result
	}
	if !proposal.preview.Workflow.Compatible || !proposal.preview.Admission.Allowed {
		result.Outcome = EditDenied
		return result
	}
	if publisher == nil {
		result.Outcome, result.Failure = EditRejected, EditUnavailable
		return result
	}
	if ctx.Err() != nil {
		result.Outcome, result.Failure = EditRejected, EditCancelled
		return result
	}
	result.Publication = publisher.PublishEdit(ctx, proposal.publicationRequest())
	switch result.Publication.Status {
	case EditPublicationApplied:
		result.Outcome = EditPublished
	case EditPublicationUnchanged:
		result.Outcome = EditUnchanged
	case EditPublicationConflict:
		result.Outcome = EditConflicted
	case EditPublicationPartial:
		result.Outcome = EditPartial
	case EditPublicationRecoveryRequired:
		result.Outcome = EditRecovery
	default:
		result.Outcome = EditFailedPublish
	}
	return result
}
