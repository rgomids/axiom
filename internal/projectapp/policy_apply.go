package projectapp

import (
	"bytes"
	"context"
	"encoding/hex"
)

type PolicyEditResult struct {
	Outcome     EditOutcome
	Failure     EditFailure
	Proposal    PolicyEditProposal
	Publication EditPublicationResult
	ReadBack    *PolicyEditReadBack
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
	if result.Outcome == EditPublished || result.Outcome == EditUnchanged {
		result.ReadBack = readBackPolicyEdit(context.WithoutCancel(ctx), ports, proposal)
	}
	return result
}

type PolicyEditReadBack struct {
	Status                   string             `json:"status"`
	ExpectedPortableRevision string             `json:"expectedPortableRevision"`
	ExpectedLocalRevision    string             `json:"expectedLocalRevision"`
	ReadinessChanged         bool               `json:"readinessChanged"`
	Observed                 *PolicyEditPreview `json:"observed,omitempty"`
}

func readBackPolicyEdit(ctx context.Context, ports PolicyEditPorts, expected PolicyEditProposal) *PolicyEditReadBack {
	portable, _ := expected.edit.local.PortableRevision.Digest()
	local, _ := ObserveLocalRevision(expected.edit.localWire).Digest()
	out := &PolicyEditReadBack{Status: "unverified", ExpectedPortableRevision: hex.EncodeToString(portable[:]), ExpectedLocalRevision: hex.EncodeToString(local[:])}
	draft, failure := DecodePolicyDraft([]byte("{}"))
	if failure != EditOK {
		return out
	}
	observed, failure := PreviewPolicyEdit(ctx, ports, EditIntent{Selector: expected.preview.ProjectID, Policy: &draft})
	if failure != EditOK {
		if failure == EditRecoveryRequired {
			out.Status = "recovery_required"
		}
		return out
	}
	preview := observed.Preview()
	out.Observed = &preview
	if observed.dependencies.Locations != expected.dependencies.Locations || observed.preview.ProjectID != expected.preview.ProjectID || observed.edit.preview.PortableDestination != expected.edit.preview.PortableDestination || observed.edit.preview.LocalDestination != expected.edit.preview.LocalDestination || !bytes.Equal(observed.edit.observedManifest, expected.edit.manifest) || !bytes.Equal(observed.edit.observedLocalWire, expected.edit.localWire) {
		out.Status = "drift"
		return out
	}
	out.Status = "confirmed"
	out.ReadinessChanged = observed.dependencies.Runtime != expected.dependencies.Runtime || observed.dependencies.Workflow != expected.dependencies.Workflow || observed.dependencies.Operational != expected.dependencies.Operational || observed.dependencies.Readiness != expected.dependencies.Readiness
	return out
}

type PolicyEditReport struct {
	Outcome        EditOutcome           `json:"outcome"`
	Preview        PolicyEditPreview     `json:"preview"`
	Publication    EditPublicationStatus `json:"publication,omitempty"`
	LocalCommitted bool                  `json:"localCommitted"`
	ReadBack       *PolicyEditReadBack   `json:"readBack,omitempty"`
}
