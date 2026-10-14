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

// safeDeclarationRequest admits the two BI-3.1 combinations only:
// `binding migrate --target-format 3` and `binding edit`, each with --runtime
// and exactly one --effort-declaration, and no other candidate field.
func safeDeclarationRequest(r AuthoringRequest) bool {
	if r.TargetFormat != 0 && (r.TargetFormat != runtimeprofile.FormatVersionV3 || r.Kind != "binding" || r.Operation != "migrate" || r.EffortDeclaration == "") {
		return false
	}
	if r.EffortDeclaration == "" {
		return true
	}
	if r.Kind != "binding" || r.RuntimeID == "" || r.ProfileID != "" || len(r.EffortDeclaration) > maxDeclarationPathBytes || strings.ContainsRune(r.EffortDeclaration, 0) {
		return false
	}
	if r.Model != "" || r.Effort != (runtimeprofile.Effort{}) || r.Capabilities != nil || r.Complexities != nil || r.Enabled != nil || r.Cascade || r.ClearLegacyReferences || len(r.MigrationEfforts) > 0 {
		return false
	}
	switch r.Operation {
	case "migrate":
		return r.TargetFormat == runtimeprofile.FormatVersionV3
	case "edit":
		return true
	}
	return false
}

// authorDeclaration is the BI-3 write path: explicit v2→v3 migration or
// declaration authoring on active-v3. Each attempt inspects the snapshot, reads
// the declaration once, observes the executable and rebuilds the candidate, so
// any drift since the preview changes its digest and is stale_preview.
func (s Authoring) authorDeclaration(ctx context.Context, req AuthoringRequest, snapshot LocalSnapshot, report AuthoringReport) (AuthoringReport, error) {
	deny := func(code string) (AuthoringReport, error) { report.Category = code; return report, ErrAuthoring }
	switch {
	case req.Operation == "migrate" && snapshot.State == "active-v1":
		report.Next = "Migrate local format 1 to 2 first with explicit effort for every Profile"
		return deny("migration_required")
	case req.Operation == "migrate" && snapshot.State != "active-v2":
		return deny("migration_mapping_required")
	case req.Operation == "edit" && (snapshot.State != "active-v3" || snapshot.V3 == nil):
		report.Next = "Declarations require local format 3; migrate explicitly with `--target-format 3`"
		return deny("migration_required")
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
	var candidate runtimeprofile.ConfigurationV3
	var prior runtimeprofile.Configuration
	effects := []string{}
	if req.Operation == "migrate" {
		source, err := runtimeprofile.Digest(snapshot.Configuration)
		if err != nil {
			return deny("candidate_invalid")
		}
		if candidate, err = runtimeprofile.MigrateV2ToV3(snapshot.Configuration, req.RuntimeID, declaration); err != nil {
			return deny("candidate_invalid")
		}
		prior = snapshot.Configuration
		effects = append(effects, "migrate local format 2 to 3", fmt.Sprintf("source format 2 revision %d digest %s", snapshot.Revision, source), "add effort declaration "+declaration.DeclarationID)
	} else {
		var replaced bool
		if candidate, replaced, err = runtimeprofile.WithEffortDeclaration(*snapshot.V3, req.RuntimeID, declaration); err != nil {
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
		prior = v3View(*snapshot.V3)
		action := "add"
		if replaced {
			action = "replace"
		}
		effects = append(effects, action+" effort declaration "+declaration.DeclarationID)
	}
	wire, err := runtimeprofile.EncodeV3(candidate)
	if err != nil {
		return deny("candidate_invalid")
	}
	effects = append(effects, fmt.Sprintf("Runtime %s effort declaration %s applies to observed version %s executable %s mode %s", req.RuntimeID, declaration.DeclarationID, observed.RuntimeVersion, observed.ExecutableDigest, observed.InvocationMode))
	report.CandidateDigest = digest(wire)
	report.Effects = append(effects, candidateEffects(prior, v3View(candidate))...)
	report.Category = "preview"
	projectLocal(&report, v3View(candidate), req)
	return s.confirm(ctx, req, snapshot.Token, wire, report, func() bool {
		again, ok := observe()
		return ok && again == observed
	})
}

// v3View is a presentation-only projection of a format-3 record onto the
// existing views and effect lines. It is never encoded or published, so it
// cannot down-convert a record.
func v3View(c runtimeprofile.ConfigurationV3) runtimeprofile.Configuration {
	view := runtimeprofile.Configuration{FormatVersion: c.FormatVersion, Revision: c.Revision, ModelProfiles: c.ModelProfiles, Preferences: c.Preferences}
	for _, r := range c.Runtimes {
		view.Runtimes = append(view.Runtimes, runtimeprofile.Runtime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs})
	}
	return view
}
