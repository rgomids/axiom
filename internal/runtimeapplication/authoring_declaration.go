package runtimeapplication

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

const maxDeclarationPathBytes = 4096

// safeDeclarationRequest admits the only declaration combination:
// `binding edit` with --runtime and exactly one --effort-declaration, and no
// other candidate field. There is no format transition (Specification 007 PM-1).
func safeDeclarationRequest(r AuthoringRequest) bool {
	if r.EffortDeclaration == "" {
		return true
	}
	if r.Kind != "binding" || r.Operation != "edit" || r.RuntimeID == "" || r.ProfileID != "" || len(r.EffortDeclaration) > maxDeclarationPathBytes || strings.ContainsRune(r.EffortDeclaration, 0) {
		return false
	}
	return r.Model == "" && r.Effort == (runtimeprofile.Effort{}) && r.Capabilities == nil && r.Complexities == nil && r.Enabled == nil && !r.Cascade
}

// authorDeclaration is the declaration write path on an active-v3 record. Each
// attempt inspects the snapshot, reads the declaration once, observes the
// executable and rebuilds the candidate, so any drift since the preview changes
// its digest and is stale_preview. A format 1/2 record is never migrated or
// converted (PM-1).
func (s Authoring) authorDeclaration(ctx context.Context, req AuthoringRequest, snapshot LocalSnapshot, report AuthoringReport) (AuthoringReport, error) {
	deny := func(code string) (AuthoringReport, error) { report.Category = code; return report, ErrAuthoring }
	if snapshot.State != "active-v3" || snapshot.V3 == nil {
		report.Next = "Effort declarations require an active canonical configuration; create one explicitly first"
		return deny("configuration_unavailable")
	}
	if snapshot.Revision == math.MaxUint64 {
		return deny("revision_overflow")
	}
	if s.ReadDeclaration == nil {
		return deny("invalid_input")
	}
	input, err := s.ReadDeclaration(req.EffortDeclaration)
	if err != nil {
		return deny("invalid_input")
	}
	declaration, err := runtimeprofile.DecodeEffortDeclaration(input, req.RuntimeID)
	if err != nil {
		return deny("invalid_input")
	}
	observe := func() (runtimeprofile.EffortApplicability, bool) {
		if s.Applicability == nil {
			return runtimeprofile.EffortApplicability{}, false
		}
		observed, err := s.Applicability(ctx, req.RuntimeID)
		// Applicability only: an exact observed identity, never a capability.
		return observed, err == nil && observed.RuntimeID == req.RuntimeID && observed.RuntimeVersion == declaration.RuntimeVersion && observed.ExecutableDigest == declaration.ExecutableDigest && observed.InvocationMode == declaration.InvocationMode
	}
	observed, ok := observe()
	if !ok {
		report.Next = "Author a declaration for the independently observed Runtime version and executable"
		return deny("effort_configuration_unproven")
	}
	candidate, replaced, err := runtimeprofile.WithEffortDeclaration(*snapshot.V3, req.RuntimeID, declaration)
	if err != nil {
		return deny("candidate_invalid")
	}
	unchanged := candidate
	unchanged.Revision = snapshot.V3.Revision
	before, beforeErr := runtimeprofile.EncodeV3(*snapshot.V3)
	after, afterErr := runtimeprofile.EncodeV3(unchanged)
	if beforeErr == nil && afterErr == nil && bytes.Equal(before, after) {
		report.Category = "unchanged"
		if req.AuthorizeLocal {
			return deny("preview_required")
		}
		return report, nil
	}
	action := "add"
	if replaced {
		action = "replace"
	}
	effects := []string{action + " effort declaration " + declaration.DeclarationID}
	wire, err := runtimeprofile.EncodeV3(candidate)
	if err != nil {
		return deny("candidate_invalid")
	}
	effects = append(effects, fmt.Sprintf("Runtime %s effort declaration %s applies to observed version %s executable %s mode %s", req.RuntimeID, declaration.DeclarationID, observed.RuntimeVersion, observed.ExecutableDigest, observed.InvocationMode))
	report.CandidateDigest = digest(wire)
	report.Effects = append(effects, candidateEffects(snapshot.V3.ModelProfiles, snapshot.V3.Preferences, v3Runtimes(candidate), candidate.ModelProfiles)...)
	report.Category = "preview"
	projectLocal(&report, v3Runtimes(candidate), candidate.ModelProfiles, req)
	return s.confirm(ctx, req, snapshot.Token, wire, report, func() bool {
		again, ok := observe()
		return ok && again == observed
	})
}
