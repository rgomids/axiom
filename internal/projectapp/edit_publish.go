package projectapp

import (
	"context"
)

// Issue #230 I230-T03 (delivering Issue #132 I132-T02): stale-safe authorized
// publication of a reviewed Project EDIT. Authority is never carried from the
// preview: replay rebuilds the complete candidate from fresh observations and
// the same explicit intent, and publishes only when the fresh Project ID and
// complete-envelope digest equal the reviewed ones. Persistence, CAS,
// confinement, locking and cross-store recovery belong to the local adapter
// behind EditPublisher (ADR-0007).

// EditAuthority is the exact replay tuple of a reviewed EDIT preview.
type EditAuthority struct {
	ProjectID      string
	PreviewDigest  string
	AuthorizeLocal bool
}

// Supplied reports whether any replay input is present.
func (a EditAuthority) Supplied() bool {
	return a.ProjectID != "" || a.PreviewDigest != "" || a.AuthorizeLocal
}

// ValidateEditAuthority rejects a partial replay tuple before any state read.
func ValidateEditAuthority(authority EditAuthority) EditFailure {
	if !authority.Supplied() {
		return EditOK
	}
	if authority.ProjectID == "" || authority.PreviewDigest == "" || !authority.AuthorizeLocal ||
		!boundedSetupText(authority.ProjectID, maxSetupRevisionBytes) || !boundedSetupText(authority.PreviewDigest, maxSetupRevisionBytes) {
		return EditIncompleteAuthority
	}
	return EditOK
}

// EditPublicationRequest names the complete candidates and the exact observed
// generations they replace. An empty PortableNext means the portable Project
// is unchanged and is only re-verified against PortableExpected.
type EditPublicationRequest struct {
	ProjectID           string
	Slug                string
	PortableDestination string
	LocalDestination    string
	PortableExpected    []byte
	PortableNext        []byte
	LocalExpected       []byte
	LocalNext           []byte
}

type EditPublicationStatus string

const (
	// EditPublicationApplied: every requested object was published and
	// confirmed; no recovery state remains.
	EditPublicationApplied EditPublicationStatus = "applied"
	// EditPublicationConflict: an exact precondition no longer held (stale
	// observation or a concurrent writer). Nothing was overwritten.
	EditPublicationConflict EditPublicationStatus = "conflict"
	// EditPublicationFailed: nothing was published and no recovery state
	// remains.
	EditPublicationFailed EditPublicationStatus = "failed"
	// EditPublicationRecoveryRequired: no effect is confirmed, but owned
	// recovery state was left for recovery inspect to classify.
	EditPublicationRecoveryRequired EditPublicationStatus = "recovery_required"
	// EditPublicationPartial: the portable publication is confirmed and the
	// local publication is not, or its recovery state could not be retired.
	// Owned recovery state names both generations.
	EditPublicationPartial EditPublicationStatus = "partial"
)

type EditPublicationResult struct {
	Status   EditPublicationStatus
	Category string
	// LocalCommitted is true when the local record was also confirmed and
	// only recovery bookkeeping remains.
	LocalCommitted bool
}

// EditPublisher publishes one authorized EDIT candidate with exact CAS
// preconditions. It never rolls back a confirmed portable commit.
type EditPublisher interface {
	PublishEdit(context.Context, EditPublicationRequest) EditPublicationResult
}

type EditOutcome string

const (
	EditPreviewed     EditOutcome = "previewed"
	EditRejected      EditOutcome = "rejected"
	EditDenied        EditOutcome = "denied"
	EditUnchanged     EditOutcome = "unchanged"
	EditPublished     EditOutcome = "published"
	EditConflicted    EditOutcome = "conflict"
	EditPartial       EditOutcome = "partial"
	EditFailedPublish EditOutcome = "failed"
	EditRecovery      EditOutcome = "recovery_required"
)

// EditResult is the canonical outcome of one EDIT request. Proposal is the
// fresh complete proposal whenever one could be built.
type EditResult struct {
	Outcome     EditOutcome
	Failure     EditFailure
	Proposal    EditProposal
	Publication EditPublicationResult
}

// PublicationRequest returns the exact CAS request for this proposal.
func (p EditProposal) PublicationRequest() EditPublicationRequest {
	request := EditPublicationRequest{
		ProjectID: p.preview.ProjectID, Slug: p.preview.Slug,
		PortableDestination: p.preview.PortableDestination, LocalDestination: p.preview.LocalDestination,
		PortableExpected: append([]byte(nil), p.observedManifest...),
		LocalExpected:    append([]byte(nil), p.observedLocalWire...),
		LocalNext:        append([]byte(nil), p.localWire...),
	}
	if p.portableChanged {
		request.PortableNext = append([]byte(nil), p.manifest...)
	}
	return request
}

// ApplyEdit previews or, with a complete authority tuple, publishes one EDIT.
// Every request rebuilds the proposal from fresh observations; a Project ID
// or digest that differs from the fresh preview is denied with zero writes.
func ApplyEdit(ctx context.Context, ports EditPorts, publisher EditPublisher, intent EditIntent, authority EditAuthority) EditResult {
	if failure := ValidateEditAuthority(authority); failure != EditOK {
		return EditResult{Outcome: EditRejected, Failure: failure}
	}
	proposal, failure := PreviewEdit(ctx, ports, intent)
	if failure != EditOK {
		return EditResult{Outcome: EditRejected, Failure: failure}
	}
	if !authority.Supplied() {
		return EditResult{Outcome: EditPreviewed, Proposal: proposal}
	}
	preview := proposal.preview
	if authority.ProjectID != preview.ProjectID || authority.PreviewDigest != preview.Digest {
		return EditResult{Outcome: EditDenied, Proposal: proposal}
	}
	if len(preview.Effects) == 0 {
		return EditResult{Outcome: EditUnchanged, Proposal: proposal}
	}
	if publisher == nil {
		return EditResult{Outcome: EditRejected, Failure: EditUnavailable, Proposal: proposal}
	}
	if ctx.Err() != nil {
		return EditResult{Outcome: EditRejected, Failure: EditCancelled, Proposal: proposal}
	}
	published := publisher.PublishEdit(ctx, proposal.PublicationRequest())
	result := EditResult{Proposal: proposal, Publication: published}
	switch published.Status {
	case EditPublicationApplied:
		result.Outcome = EditPublished
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
