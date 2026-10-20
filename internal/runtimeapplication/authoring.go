package runtimeapplication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/portableconfig"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// LocalSnapshot is internal machine state. Only AuthoringReport crosses the
// presentation boundary; legacy configuration and host identities never do.
type LocalSnapshot struct {
	State    string
	Revision uint64
	// Legacy is a recognized pre-stable active-v1/active-v2 record. It is read
	// only to bind its explicit retirement (Specification 007 PM-11, A1).
	Legacy runtimeprofile.Configuration
	// V3 is a historical active-v3 record, read only to bind its explicit
	// retirement; it is never current configuration (AXM-36 D2).
	V3 *runtimeprofile.ConfigurationV3
	// V4 is the strictly decoded current record of an active-v4 snapshot.
	V4    *runtimeprofile.ConfigurationV4
	Token string
}
type AuthoringRecovery interface {
	Recover(context.Context, AuthoringRequest) (AuthoringReport, error)
}
type AuthoringStore interface {
	Inspect(context.Context) (LocalSnapshot, error)
	Publish(context.Context, string, []byte) (LocalSnapshot, error)
}
type AuthoringRequest struct {
	Kind, Operation, RuntimeID, ProfileID, Model string
	Effort                                       runtimeprofile.Effort
	Capabilities, Complexities                   []string
	Enabled                                      *bool
	Cascade                                      bool
	ExpectedRevision, PreviewDigest              string
	AuthorizeLocal                               bool
}
type ProfileView struct {
	ID        string                `json:"id"`
	RuntimeID string                `json:"runtimeId"`
	Model     string                `json:"model"`
	Effort    runtimeprofile.Effort `json:"effort"`
	Eligible  bool                  `json:"eligible"`
}
type RuntimeView struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}
type AuthoringReport struct {
	KnownImpacts     []LocalImpact     `json:"knownImpacts,omitempty"`
	DetectedRuntimes []DetectedRuntime `json:"detectedRuntimes,omitempty"`
	State            string            `json:"state"`
	Revision         uint64            `json:"revision"`
	Category         string            `json:"category"`
	ExpectedRevision string            `json:"expectedRevision,omitempty"`
	PreviewDigest    string            `json:"previewDigest,omitempty"`
	CandidateDigest  string            `json:"candidateDigest,omitempty"`
	Effects          []string          `json:"effects,omitempty"`
	Profiles         []ProfileView     `json:"profiles,omitempty"`
	Runtimes         []RuntimeView     `json:"runtimes,omitempty"`
	ImpactInventory  string            `json:"impactInventory"`
	Authentication   string            `json:"authentication"`
	ModelSupport     string            `json:"modelSupport"`
	EffortSupport    string            `json:"effortSupport"`
	Next             string            `json:"next,omitempty"`
}
type Authoring struct {
	Store AuthoringStore
	// Installed is passive executable discovery, never authentication/probing.
	Installed func(string) bool
	Impacts   func(context.Context) ([]LocalImpact, string)
	Passive   func(context.Context) []DetectedRuntime
}
type LocalImpact struct {
	ProjectID string `json:"projectId"`
	ProfileID string `json:"profileId"`
}
type DetectedRuntime struct {
	RuntimeID   string `json:"runtimeId"`
	Installed   bool   `json:"installed"`
	Integration string `json:"integration"`
}

var ErrAuthoring = errors.New("local runtime authoring blocked")

const newerFormatNext = "Local configuration format is newer than this authoring path supports; it is preserved unchanged, with no downgrade or migration"

func (s Authoring) Run(ctx context.Context, req AuthoringRequest) (AuthoringReport, error) {
	report := AuthoringReport{ImpactInventory: "incomplete: project references not evaluated", Authentication: "unknown", ModelSupport: "unconfirmed", EffortSupport: "unconfirmed"}
	deny := func(code string) (AuthoringReport, error) { report.Category = code; return report, ErrAuthoring }
	if s.Store == nil || !safeAuthoring(req) {
		return deny("invalid_input")
	}
	if s.Impacts != nil {
		report.KnownImpacts, report.ImpactInventory = s.Impacts(ctx)
		for i := range report.KnownImpacts {
			if !runtimeprofile.SafeIdentifier(report.KnownImpacts[i].ProfileID) {
				report.KnownImpacts[i].ProfileID = "redacted"
			}
			if !runtimeprofile.SafeIdentifier(report.KnownImpacts[i].ProjectID) {
				report.KnownImpacts[i].ProjectID = "redacted"
			}
		}
	}
	if s.Passive != nil {
		report.DetectedRuntimes = s.Passive(ctx)
	}
	if req.Operation == "recover" {
		recovery, ok := s.Store.(AuthoringRecovery)
		if !ok {
			return deny("recovery_unavailable")
		}
		return recovery.Recover(ctx, req)
	}
	snapshot, err := s.Store.Inspect(ctx)
	if err != nil {
		if errors.Is(err, runtimeprofile.ErrUnsupportedConfigurationFormat) {
			report.Next = newerFormatNext
		}
		return deny("configuration_unavailable")
	}
	report.State, report.Revision = snapshot.State, snapshot.Revision
	if snapshot.State == "unsafe" || snapshot.State == "pending" {
		report.Next = "Use axiom-runtime scoped recovery; preserve canonical state and prepare fresh exact authority"
		return deny("recovery_required")
	}
	report.ExpectedRevision = snapshot.Token
	if req.AuthorizeLocal && req.ExpectedRevision != snapshot.Token {
		return deny("stale_preview")
	}
	if snapshot.State == "active-v1" || snapshot.State == "active-v2" || snapshot.State == "active-v3" {
		return s.retireLegacy(ctx, req, snapshot, report)
	}
	if snapshot.State == "active-v4" && snapshot.V4 == nil || snapshot.State != "active-v4" && snapshot.State != "fresh" && snapshot.State != "retired" {
		return deny("configuration_unavailable")
	}
	prior := runtimeprofile.ConfigurationV4{FormatVersion: runtimeprofile.FormatVersionV4, Revision: snapshot.Revision}
	if snapshot.State == "active-v4" {
		prior = *snapshot.V4
	}
	if req.Operation == "list" || req.Operation == "show" {
		if req.AuthorizeLocal || req.ExpectedRevision != "" || req.PreviewDigest != "" {
			return deny("invalid_authority")
		}
		projectLocal(&report, v4Runtimes(prior), prior.ModelProfiles, req)
		report.Category = "inspected"
		if snapshot.State == "fresh" || snapshot.State == "retired" {
			report.Next = "Use axiom-runtime to prepare explicit local configuration"
		}
		return report, nil
	}
	if snapshot.Revision == math.MaxUint64 {
		return deny("revision_overflow")
	}
	// Fresh and retired state start an explicit new canonical v4 record; a
	// retired record is never resurrected (Specification 007 PM-11).
	cfg, err := cloneV4(prior)
	if err != nil {
		return deny("configuration_invalid")
	}
	edit := req
	if req.Operation == "validate" || req.Operation == "validate-candidate" {
		edit.Operation = "create"
		if req.Kind == "profile" {
			for _, p := range cfg.ModelProfiles {
				if p.ID == req.ProfileID {
					edit.Operation = "edit"
				}
			}
		} else {
			for _, r := range cfg.Runtimes {
				if r.ID == req.RuntimeID {
					edit.Operation = "edit"
				}
			}
		}
	}
	if err := editCandidate(&cfg, edit, s.Installed); err != nil {
		return deny(err.Error())
	}
	effects := []string{req.Kind + " " + req.Operation}
	if len(cfg.Runtimes) != 0 {
		cfg.Revision = snapshot.Revision + 1
		if runtimeprofile.ValidateV4(cfg) != nil {
			return deny("candidate_invalid")
		}
	}
	if req.Operation == "validate-candidate" || req.Operation == "validate" {
		report.Category = "candidate_valid"
		projectLocal(&report, v4Runtimes(cfg), cfg.ModelProfiles, req)
		return report, nil
	}
	if snapshot.State == "active-v4" && len(cfg.Runtimes) != 0 {
		unchanged := cfg
		unchanged.Revision = prior.Revision
		before, beforeErr := runtimeprofile.EncodeV4(prior)
		after, afterErr := runtimeprofile.EncodeV4(unchanged)
		if beforeErr == nil && afterErr == nil && bytes.Equal(before, after) {
			report.Category = "unchanged"
			if req.AuthorizeLocal {
				return deny("preview_required")
			}
			return report, nil
		}
	}
	var wire []byte
	if len(cfg.Runtimes) == 0 {
		// Removing the final Runtime retires the canonical record explicitly;
		// no empty v4 configuration exists.
		if snapshot.State != "active-v4" {
			return deny("configuration_required")
		}
		priorDigest, e := runtimeprofile.DigestV4(prior)
		if e != nil {
			return deny("configuration_invalid")
		}
		wire, err = runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: snapshot.Revision + 1, PriorConfigurationDigest: priorDigest})
		effects = append(effects, "retire canonical configuration; preserve directories and vendor state", "tombstone binds prior configuration digest "+priorDigest)
	} else {
		wire, err = runtimeprofile.EncodeV4(cfg)
	}
	if err != nil {
		return deny("candidate_invalid")
	}
	report.CandidateDigest = digest(wire)
	report.Effects = append(effects, candidateEffects(prior.ModelProfiles, prior.Preferences, v4Runtimes(cfg), cfg.ModelProfiles)...)
	report.Category = "preview"
	projectLocal(&report, v4Runtimes(cfg), cfg.ModelProfiles, req)
	return s.confirm(ctx, req, snapshot.Token, wire, report, nil)
}

const legacyNext = "Historical local configuration is not used and is never migrated; retire it explicitly with `axiom runtime binding remove --runtime <id> --cascade`, then create canonical configuration"

// historicalRecord is the retirement view of a recognized historical record.
type historicalRecord struct {
	formatVersion int
	revision      uint64
	digest        string
	runtimes      []localRuntime
	profiles      []runtimeprofile.ModelProfile
	preferences   []runtimeprofile.Preference
}

func historicalSnapshot(snapshot LocalSnapshot) (historicalRecord, bool) {
	if snapshot.State == "active-v3" {
		if snapshot.V3 == nil || runtimeprofile.ValidateV3(*snapshot.V3) != nil || snapshot.V3.Revision != snapshot.Revision {
			return historicalRecord{}, false
		}
		d, err := runtimeprofile.DigestV3(*snapshot.V3)
		runtimes := make([]localRuntime, 0, len(snapshot.V3.Runtimes))
		for _, r := range snapshot.V3.Runtimes {
			runtimes = append(runtimes, localRuntime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, Allowlisted: r.AllowlistedProfileIDs})
		}
		return historicalRecord{runtimeprofile.FormatVersionV3, snapshot.V3.Revision, d, runtimes, snapshot.V3.ModelProfiles, snapshot.V3.Preferences}, err == nil
	}
	legacy := snapshot.Legacy
	if runtimeprofile.Validate(legacy) != nil || legacy.Revision != snapshot.Revision {
		return historicalRecord{}, false
	}
	d, err := runtimeprofile.Digest(legacy)
	runtimes := make([]localRuntime, 0, len(legacy.Runtimes))
	for _, r := range legacy.Runtimes {
		runtimes = append(runtimes, localRuntime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, Allowlisted: r.AllowlistedProfileIDs})
	}
	return historicalRecord{legacy.FormatVersion, legacy.Revision, d, runtimes, legacy.ModelProfiles, legacy.Preferences}, err == nil
}

// retireLegacy is the only operation over a recognized historical
// active-v1/v2/v3 record (Specification 007 PM-11, A1; AXM-36 D2): `binding
// remove --runtime <id> --cascade` retires the ENTIRE record through the
// existing tombstone, bound to the historical digest of the exact prior record.
// The named Runtime must exist in it. Nothing is migrated, partially edited,
// converted or created; list/show only report the state. Every other operation
// is unsupported_configuration_format.
func (s Authoring) retireLegacy(ctx context.Context, req AuthoringRequest, snapshot LocalSnapshot, report AuthoringReport) (AuthoringReport, error) {
	deny := func(code string) (AuthoringReport, error) { report.Category = code; return report, ErrAuthoring }
	report.Next = legacyNext
	legacy, ok := historicalSnapshot(snapshot)
	if !ok {
		return deny("configuration_invalid")
	}
	if req.Operation == "list" || req.Operation == "show" {
		if req.AuthorizeLocal || req.ExpectedRevision != "" || req.PreviewDigest != "" {
			return deny("invalid_authority")
		}
		report.Category = "inspected"
		return report, nil
	}
	if req.Kind != "binding" || req.Operation != "remove" {
		return deny("unsupported_configuration_format")
	}
	found := false
	for _, r := range legacy.runtimes {
		found = found || r.ID == req.RuntimeID
	}
	if req.RuntimeID == "" || !found {
		return deny("binding_missing")
	}
	if !req.Cascade {
		return deny("compound_removal_required")
	}
	if snapshot.Revision == math.MaxUint64 {
		return deny("revision_overflow")
	}
	priorDigest := legacy.digest
	wire, err := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: snapshot.Revision + 1, PriorConfigurationDigest: priorDigest})
	if err != nil {
		return deny("candidate_invalid")
	}
	effects := []string{
		fmt.Sprintf("retire ENTIRE historical local configuration format %d revision %d: --cascade on a historical record retires every Runtime and Profile, not only Runtime %s", legacy.formatVersion, legacy.revision, req.RuntimeID),
		"publish tombstone binding prior configuration digest " + priorDigest + "; preserve directories and vendor state",
		"no canonical configuration is created and nothing is migrated; create one explicitly afterwards",
	}
	for _, r := range legacy.runtimes {
		effects = append(effects, fmt.Sprintf("retire Runtime %s enabled=%t allowlist=%s", safeView(r.ID), r.Enabled, strings.Join(safeViews(r.Allowlisted), ",")))
	}
	for _, p := range legacy.profiles {
		effects = append(effects, "retire Profile "+safeView(p.ID)+" runtime="+safeView(p.RuntimeID))
	}
	for _, preference := range legacy.preferences {
		effects = append(effects, "retire local preference "+safeView(preference.Role)+"/"+safeView(preference.Complexity)+" referring to Profile "+safeView(preference.ModelProfileID))
	}
	sort.Strings(effects[3:])
	report.CandidateDigest = digest(wire)
	report.Effects = effects
	report.Category = "preview"
	projectLocal(&report, legacy.runtimes, legacy.profiles, AuthoringRequest{Kind: "binding"})
	result, err := s.confirm(ctx, req, snapshot.Token, wire, report, nil)
	if err == nil && result.Category == "applied" {
		result.Next = "Historical configuration retired; create canonical configuration explicitly with axiom-runtime"
	}
	return result, err
}

func safeView(v string) string {
	if !runtimeprofile.SafeIdentifier(v) {
		return "redacted"
	}
	return v
}

func safeViews(values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = safeView(v)
	}
	return out
}

// confirm binds the preview envelope and publishes only under exact local
// authority. revalidate, when set, re-observes inputs immediately before the
// write; any drift is stale_preview and nothing is written.
func (s Authoring) confirm(ctx context.Context, req AuthoringRequest, token string, wire []byte, report AuthoringReport, revalidate func() bool) (AuthoringReport, error) {
	deny := func(code string) (AuthoringReport, error) { report.Category = code; return report, ErrAuthoring }
	report.PreviewDigest = ""
	envelope, _ := json.Marshal(report)
	report.PreviewDigest = digest(envelope)
	if !req.AuthorizeLocal {
		if req.ExpectedRevision != "" || req.PreviewDigest != "" {
			return deny("local_authority_required")
		}
		return report, nil
	}
	if req.ExpectedRevision != token || req.PreviewDigest != report.PreviewDigest {
		return deny("stale_preview")
	}
	if revalidate != nil && !revalidate() {
		return deny("stale_preview")
	}
	actual, err := s.Store.Publish(ctx, token, wire)
	if err != nil {
		report.Next = "Inspect scoped local recovery before preparing a new preview; publication is unconfirmed"
		return deny("publication_unconfirmed")
	}
	report.State, report.Revision = actual.State, actual.Revision
	report.Category = "applied"
	report.Next = "Intent saved; executable, authentication and effort sources are observed at each preview and before each attempt; vendor model and effort support stay unconfirmed until a real attempt"
	return report, nil
}

// cloneV4 is an independent canonical copy; an empty fresh/retired start has
// no record to copy.
func cloneV4(c runtimeprofile.ConfigurationV4) (runtimeprofile.ConfigurationV4, error) {
	if len(c.Runtimes) == 0 {
		return runtimeprofile.ConfigurationV4{FormatVersion: runtimeprofile.FormatVersionV4, Revision: c.Revision}, nil
	}
	wire, err := runtimeprofile.EncodeV4(c)
	if err != nil {
		return c, err
	}
	return runtimeprofile.DecodeV4(wire)
}
func safeAuthoring(r AuthoringRequest) bool {
	if r.Kind != "profile" && r.Kind != "binding" {
		return false
	}
	switch r.Operation {
	case "list", "show", "create", "edit", "remove", "enable", "disable", "validate", "validate-candidate", "recover":
	default:
		return false
	}
	if r.Cascade && r.Operation != "remove" {
		return false
	}
	fields := r.Model != "" || r.Effort != (runtimeprofile.Effort{}) || r.Capabilities != nil || r.Complexities != nil
	if fields && r.Operation != "create" && r.Operation != "edit" && r.Operation != "validate" && r.Operation != "validate-candidate" {
		return false
	}
	if r.Kind == "profile" && r.Enabled != nil {
		return false
	}
	if r.Enabled != nil && r.Operation != "create" && r.Operation != "edit" && r.Operation != "validate" && r.Operation != "validate-candidate" {
		return false
	}
	for _, v := range []string{r.RuntimeID, r.ProfileID} {
		if v != "" && !portableconfig.SafeValue(v, "token") {
			return false
		}
	}
	if r.Model != "" && (!runtimeprofile.SafeIdentifier(r.Model) || !portableconfig.SafeValue(r.Model, "logical") || portableconfig.ContainsSecretBearingValue(r.Model)) {
		return false
	}
	if r.Effort.Mode != "" || r.Effort.Value != "" {
		if !runtimeprofile.ValidEffort(r.Effort) {
			return false
		}
	}
	for _, v := range append(append([]string{}, r.Capabilities...), r.Complexities...) {
		if !runtimeprofile.SafeIdentifier(v) || !portableconfig.SafeValue(v, "token") {
			return false
		}
	}
	return len(r.Capabilities) <= 32 && len(r.Complexities) <= 32
}
func editCandidate(c *runtimeprofile.ConfigurationV4, r AuthoringRequest, installed func(string) bool) error {
	fail := func(code string) error { return errors.New(code) }
	if r.Operation == "validate" || r.Operation == "validate-candidate" {
		return nil
	}
	if r.Kind == "binding" {
		index := -1
		for i, x := range c.Runtimes {
			if x.ID == r.RuntimeID {
				index = i
			}
		}
		switch r.Operation {
		case "create":
			if index >= 0 {
				return fail("binding_exists")
			}
			if r.RuntimeID != "codex" && r.RuntimeID != "claude" {
				return fail("runtime_unsupported")
			}
			if installed == nil || !installed(r.RuntimeID) {
				return fail("runtime_absent")
			}
			enabled := true
			if r.Enabled != nil {
				enabled = *r.Enabled
			}
			c.Runtimes = append(c.Runtimes, runtimeprofile.RuntimeV4{ID: r.RuntimeID, Adapter: r.RuntimeID, Enabled: enabled, AllowlistedProfileIDs: []string{r.ProfileID}})
			c.ModelProfiles = append(c.ModelProfiles, runtimeprofile.ModelProfile{ID: r.ProfileID, RuntimeID: r.RuntimeID, Model: r.Model, Effort: r.Effort, Capabilities: r.Capabilities, Complexities: r.Complexities})
		case "edit", "enable", "disable":
			if index < 0 {
				return fail("binding_missing")
			}
			if r.ProfileID != "" || r.Model != "" || r.Effort != (runtimeprofile.Effort{}) || r.Capabilities != nil || r.Complexities != nil {
				return fail("invalid_binding_candidate")
			}
			if r.Operation == "edit" && r.Enabled == nil {
				return fail("candidate_required")
			}
			enabled := r.Operation == "enable"
			if r.Enabled != nil {
				enabled = *r.Enabled
			}
			c.Runtimes[index].Enabled = enabled
		case "remove":
			if index < 0 || !r.Cascade {
				return fail("compound_removal_required")
			}
			c.Runtimes = append(c.Runtimes[:index], c.Runtimes[index+1:]...)
			for i := len(c.ModelProfiles) - 1; i >= 0; i-- {
				if c.ModelProfiles[i].RuntimeID == r.RuntimeID {
					removeProfile(c, i)
				}
			}
		default:
			return fail("invalid_operation")
		}
		return nil
	}
	index := -1
	for i, x := range c.ModelProfiles {
		if x.ID == r.ProfileID {
			index = i
		}
	}
	switch r.Operation {
	case "create":
		if index >= 0 {
			return fail("profile_exists")
		}
		rt := -1
		for i, x := range c.Runtimes {
			if x.ID == r.RuntimeID {
				rt = i
			}
		}
		if rt < 0 {
			return fail("binding_missing")
		}
		c.ModelProfiles = append(c.ModelProfiles, runtimeprofile.ModelProfile{ID: r.ProfileID, RuntimeID: r.RuntimeID, Model: r.Model, Effort: r.Effort, Capabilities: r.Capabilities, Complexities: r.Complexities})
		c.Runtimes[rt].AllowlistedProfileIDs = append(c.Runtimes[rt].AllowlistedProfileIDs, r.ProfileID)
	case "edit":
		if index < 0 {
			return fail("profile_missing")
		}
		p := &c.ModelProfiles[index]
		if r.RuntimeID != "" && r.RuntimeID != p.RuntimeID {
			return fail("profile_runtime_mismatch")
		}
		if r.Model != "" {
			p.Model = r.Model
		}
		if r.Effort.Mode != "" {
			p.Effort = r.Effort
		}
		if r.Capabilities != nil {
			p.Capabilities = r.Capabilities
		}
		if r.Complexities != nil {
			p.Complexities = r.Complexities
		}
	case "remove":
		if index < 0 {
			return fail("profile_missing")
		}
		runtimeID := c.ModelProfiles[index].RuntimeID
		removeProfile(c, index)
		for i := len(c.Runtimes) - 1; i >= 0; i-- {
			if c.Runtimes[i].ID == runtimeID && len(c.Runtimes[i].AllowlistedProfileIDs) == 0 {
				if !r.Cascade {
					return fail("compound_removal_required")
				}
				c.Runtimes = append(c.Runtimes[:i], c.Runtimes[i+1:]...)
			}
		}
	case "enable", "disable":
		if index < 0 {
			return fail("profile_missing")
		}
		for i := range c.Runtimes {
			if c.Runtimes[i].ID != c.ModelProfiles[index].RuntimeID {
				continue
			}
			ids := c.Runtimes[i].AllowlistedProfileIDs
			found := -1
			for j, id := range ids {
				if id == r.ProfileID {
					found = j
				}
			}
			if r.Operation == "enable" && found < 0 {
				ids = append(ids, r.ProfileID)
			}
			if r.Operation == "disable" && found >= 0 {
				ids = append(ids[:found], ids[found+1:]...)
			}
			c.Runtimes[i].AllowlistedProfileIDs = ids
		}
	default:
		return fail("invalid_operation")
	}
	return nil
}
func removeProfile(c *runtimeprofile.ConfigurationV4, index int) {
	id := c.ModelProfiles[index].ID
	c.ModelProfiles = append(c.ModelProfiles[:index], c.ModelProfiles[index+1:]...)
	for i := range c.Runtimes {
		ids := c.Runtimes[i].AllowlistedProfileIDs
		for j := len(ids) - 1; j >= 0; j-- {
			if ids[j] == id {
				ids = append(ids[:j], ids[j+1:]...)
			}
		}
		c.Runtimes[i].AllowlistedProfileIDs = ids
	}
	for i := len(c.Preferences) - 1; i >= 0; i-- {
		if c.Preferences[i].ModelProfileID == id {
			c.Preferences = append(c.Preferences[:i], c.Preferences[i+1:]...)
		}
	}
}

// localRuntime is the presentation input of one Runtime: canonical v4
// Runtimes and, for the retirement preview only, historical ones.
type localRuntime struct {
	ID, Adapter string
	Enabled     bool
	Allowlisted []string
}

func v4Runtimes(c runtimeprofile.ConfigurationV4) []localRuntime {
	out := make([]localRuntime, 0, len(c.Runtimes))
	for _, r := range c.Runtimes {
		out = append(out, localRuntime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, Allowlisted: r.AllowlistedProfileIDs})
	}
	return out
}

func projectLocal(out *AuthoringReport, runtimes []localRuntime, profiles []runtimeprofile.ModelProfile, r AuthoringRequest) {
	for _, rt := range runtimes {
		if r.Kind == "binding" && r.RuntimeID != "" && rt.ID != r.RuntimeID {
			continue
		}
		out.Runtimes = append(out.Runtimes, RuntimeView{safeView(rt.ID), rt.Enabled})
	}
	for _, p := range profiles {
		if r.Kind == "profile" && r.ProfileID != "" && p.ID != r.ProfileID {
			continue
		}
		model := p.Model
		if !runtimeprofile.SafeIdentifier(model) || !portableconfig.SafeValue(model, "logical") || portableconfig.ContainsSecretBearingValue(model) {
			model = "redacted"
		}
		eligible := false
		for _, rt := range runtimes {
			if rt.ID == p.RuntimeID && rt.Enabled {
				for _, id := range rt.Allowlisted {
					if id == p.ID {
						eligible = true
					}
				}
			}
		}
		effort := p.Effort
		if !runtimeprofile.ValidEffort(effort) {
			effort = runtimeprofile.Effort{Mode: "unconfigured", Value: "unconfigured"}
		}
		out.Profiles = append(out.Profiles, ProfileView{ID: safeView(p.ID), RuntimeID: safeView(p.RuntimeID), Model: model, Effort: effort, Eligible: eligible})
	}
	sort.Slice(out.Profiles, func(i, j int) bool { return strings.Compare(out.Profiles[i].ID, out.Profiles[j].ID) < 0 })
	sort.Slice(out.Runtimes, func(i, j int) bool { return out.Runtimes[i].ID < out.Runtimes[j].ID })
}

func candidateEffects(priorProfiles []runtimeprofile.ModelProfile, priorPreferences []runtimeprofile.Preference, nextRuntimes []localRuntime, nextProfiles []runtimeprofile.ModelProfile) []string {
	effects := []string{}
	ids := map[string]bool{}
	for _, p := range nextProfiles {
		ids[p.ID] = true
	}
	for _, p := range priorProfiles {
		if !ids[p.ID] && runtimeprofile.SafeIdentifier(p.ID) {
			effects = append(effects, "remove Profile "+p.ID)
		}
	}
	for _, preference := range priorPreferences {
		if !ids[preference.ModelProfileID] && runtimeprofile.SafeIdentifier(preference.ModelProfileID) {
			effects = append(effects, "remove local preference referring to Profile "+preference.ModelProfileID)
		}
	}
	for _, r := range nextRuntimes {
		effects = append(effects, fmt.Sprintf("Runtime %s enabled=%t allowlist=%s", r.ID, r.Enabled, strings.Join(r.Allowlisted, ",")))
	}
	for _, p := range nextProfiles {
		effects = append(effects, "Profile "+p.ID+" capabilities="+strings.Join(p.Capabilities, ",")+" complexities="+strings.Join(p.Complexities, ","))
	}
	sort.Strings(effects)
	return effects
}
