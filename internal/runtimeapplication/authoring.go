package runtimeapplication

import (
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
	State         string
	Revision      uint64
	Configuration runtimeprofile.Configuration
	// V3 is the strictly decoded record of an active-v3 snapshot.
	V3    *runtimeprofile.ConfigurationV3
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
	Cascade, ClearLegacyReferences               bool
	MigrationEfforts                             map[string]runtimeprofile.Effort
	// TargetFormat selects the explicit v2→v3 migration (only 3, BI-3.1).
	TargetFormat int
	// EffortDeclaration is the host path of one declaration file; it is read
	// once per attempt and never persisted or reported.
	EffortDeclaration               string
	ExpectedRevision, PreviewDigest string
	AuthorizeLocal                  bool
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
	// ReadDeclaration performs the single bounded, non-symlinked read of a
	// declaration file inside the trusted filesystem boundary.
	ReadDeclaration func(string) ([]byte, error)
	// Applicability passively observes the Runtime executable identity and
	// version a declaration must match; it never probes or proves capability.
	Applicability func(context.Context, string) (runtimeprofile.EffortApplicability, error)
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
	report := AuthoringReport{ImpactInventory: "incomplete: project references not evaluated", Authentication: "unknown", ModelSupport: "unproven", EffortSupport: "unproven"}
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
	if req.EffortDeclaration != "" {
		return s.authorDeclaration(ctx, req, snapshot, report)
	}
	if snapshot.State == "active-v3" && snapshot.V3 == nil {
		return deny("configuration_unavailable")
	}
	if req.Operation == "list" || req.Operation == "show" {
		if req.AuthorizeLocal || req.ExpectedRevision != "" || req.PreviewDigest != "" {
			return deny("invalid_authority")
		}
		view := snapshot.Configuration
		if snapshot.State == "active-v3" {
			view = v3View(*snapshot.V3)
		}
		projectLocal(&report, view, req)
		report.Category = "inspected"
		if snapshot.State == "fresh" || snapshot.State == "retired" {
			report.Next = "Use axiom-runtime to prepare explicit local configuration"
		}
		if snapshot.State == "active-v1" {
			report.Next = "Migrate explicitly with effort for every Profile"
		}
		return report, nil
	}
	// Format 3 is mutated only by declaration authoring; every other mutation
	// stays v2-only and refuses it, with no downgrade or implicit migration.
	if snapshot.State == "active-v3" {
		report.Next = newerFormatNext
		return deny("configuration_unavailable")
	}
	if snapshot.Revision == math.MaxUint64 {
		return deny("revision_overflow")
	}
	cfg, err := cloneConfiguration(snapshot.Configuration)
	if err != nil && snapshot.State != "fresh" && snapshot.State != "retired" {
		return deny("configuration_invalid")
	}
	if snapshot.State == "fresh" || snapshot.State == "retired" {
		cfg = runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2, Revision: snapshot.Revision}
	}
	effects := []string{}
	if req.Operation == "migrate" {
		if snapshot.State != "active-v1" || len(req.MigrationEfforts) != len(cfg.ModelProfiles) {
			return deny("migration_mapping_required")
		}
		for i := range cfg.ModelProfiles {
			effort, ok := req.MigrationEfforts[cfg.ModelProfiles[i].ID]
			if !ok {
				return deny("migration_mapping_required")
			}
			cfg.ModelProfiles[i].Effort = effort
		}
		for i := range cfg.Runtimes {
			if cfg.Runtimes[i].CredentialReference != "" {
				if !req.ClearLegacyReferences {
					return deny("legacy_reference_removal_required")
				}
				cfg.Runtimes[i].CredentialReference = ""
				effects = append(effects, "remove legacy credential reference")
			}
		}
		cfg.FormatVersion = runtimeprofile.FormatVersionV2
		effects = append(effects, "migrate local format 1 to 2")
	} else {
		if snapshot.State == "active-v1" {
			return deny("migration_required")
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
		effects = append(effects, req.Kind+" "+req.Operation)
	}
	if len(cfg.Runtimes) != 0 {
		cfg.Revision = snapshot.Revision + 1
		if runtimeprofile.Validate(cfg) != nil {
			return deny("candidate_invalid")
		}
	}
	if req.Operation == "validate-candidate" || req.Operation == "validate" {
		report.Category = "candidate_valid"
		projectLocal(&report, cfg, req)
		return report, nil
	}
	before := snapshot.Configuration
	before.Revision = cfg.Revision
	priorSemantic, _ := runtimeprofile.Digest(before)
	nextSemantic, _ := runtimeprofile.Digest(cfg)
	if snapshot.State == "active-v2" && priorSemantic != "" && priorSemantic == nextSemantic {
		report.Category = "unchanged"
		if req.AuthorizeLocal {
			return deny("preview_required")
		}
		return report, nil
	}
	var wire []byte
	if len(cfg.Runtimes) == 0 {
		if snapshot.State != "active-v2" {
			return deny("configuration_required")
		}
		prior, e := runtimeprofile.Digest(snapshot.Configuration)
		if e != nil {
			return deny("configuration_invalid")
		}
		wire, err = runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: "runtime-profile-tombstone", FormatVersion: 1, Revision: snapshot.Revision + 1, PriorConfigurationDigest: prior})
		effects = append(effects, "retire canonical configuration; preserve directories and vendor state")
	} else {
		wire, err = runtimeprofile.Encode(cfg)
	}
	if err != nil {
		return deny("candidate_invalid")
	}
	report.CandidateDigest = digest(wire)
	report.Effects = append(effects, candidateEffects(snapshot.Configuration, cfg)...)
	report.Category = "preview"
	projectLocal(&report, cfg, req)
	return s.confirm(ctx, req, snapshot.Token, wire, report, nil)
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
	report.Next = "Intent saved; operational model, effort and authentication proofs remain separate"
	return report, nil
}

func cloneConfiguration(c runtimeprofile.Configuration) (runtimeprofile.Configuration, error) {
	if c.FormatVersion == 0 {
		return c, nil
	}
	b, e := runtimeprofile.Encode(c)
	if e != nil {
		return c, e
	}
	return runtimeprofile.Decode(b)
}
func safeAuthoring(r AuthoringRequest) bool {
	if r.Kind != "profile" && r.Kind != "binding" {
		return false
	}
	switch r.Operation {
	case "list", "show", "create", "edit", "remove", "enable", "disable", "validate", "validate-candidate", "migrate", "recover":
	default:
		return false
	}
	if r.Cascade && r.Operation != "remove" || (r.ClearLegacyReferences || len(r.MigrationEfforts) > 0) && r.Operation != "migrate" {
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
	for id, e := range r.MigrationEfforts {
		if !runtimeprofile.SafeIdentifier(id) || !runtimeprofile.ValidEffort(e) {
			return false
		}
	}
	if !safeDeclarationRequest(r) {
		return false
	}
	return len(r.Capabilities) <= 32 && len(r.Complexities) <= 32 && len(r.MigrationEfforts) <= 32
}
func editCandidate(c *runtimeprofile.Configuration, r AuthoringRequest, installed func(string) bool) error {
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
			c.Runtimes = append(c.Runtimes, runtimeprofile.Runtime{ID: r.RuntimeID, Adapter: r.RuntimeID, Enabled: enabled, AllowlistedProfileIDs: []string{r.ProfileID}})
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
func removeProfile(c *runtimeprofile.Configuration, index int) {
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
func projectLocal(out *AuthoringReport, c runtimeprofile.Configuration, r AuthoringRequest) {
	for _, rt := range c.Runtimes {
		if r.Kind == "binding" && r.RuntimeID != "" && rt.ID != r.RuntimeID {
			continue
		}
		id := rt.ID
		if !runtimeprofile.SafeIdentifier(id) {
			id = "redacted"
		}
		out.Runtimes = append(out.Runtimes, RuntimeView{id, rt.Enabled})
	}
	for _, p := range c.ModelProfiles {
		if r.Kind == "profile" && r.ProfileID != "" && p.ID != r.ProfileID {
			continue
		}
		model := p.Model
		if !runtimeprofile.SafeIdentifier(model) || !portableconfig.SafeValue(model, "logical") || portableconfig.ContainsSecretBearingValue(model) {
			model = "redacted"
		}
		eligible := false
		for _, rt := range c.Runtimes {
			if rt.ID == p.RuntimeID && rt.Enabled {
				for _, id := range rt.AllowlistedProfileIDs {
					if id == p.ID {
						eligible = true
					}
				}
			}
		}
		id, runtimeID, effort := p.ID, p.RuntimeID, p.Effort
		if !runtimeprofile.SafeIdentifier(id) {
			id = "redacted"
		}
		if !runtimeprofile.SafeIdentifier(runtimeID) {
			runtimeID = "redacted"
		}
		if !runtimeprofile.ValidEffort(effort) {
			effort = runtimeprofile.Effort{Mode: "unconfigured", Value: "unconfigured"}
		}
		out.Profiles = append(out.Profiles, ProfileView{ID: id, RuntimeID: runtimeID, Model: model, Effort: effort, Eligible: eligible})
	}
	sort.Slice(out.Profiles, func(i, j int) bool { return strings.Compare(out.Profiles[i].ID, out.Profiles[j].ID) < 0 })
	sort.Slice(out.Runtimes, func(i, j int) bool { return out.Runtimes[i].ID < out.Runtimes[j].ID })
}

func candidateEffects(prior, next runtimeprofile.Configuration) []string {
	effects := []string{}
	ids := map[string]bool{}
	for _, p := range next.ModelProfiles {
		ids[p.ID] = true
	}
	for _, p := range prior.ModelProfiles {
		if !ids[p.ID] && runtimeprofile.SafeIdentifier(p.ID) {
			effects = append(effects, "remove Profile "+p.ID)
		}
	}
	for _, preference := range prior.Preferences {
		if !ids[preference.ModelProfileID] && runtimeprofile.SafeIdentifier(preference.ModelProfileID) {
			effects = append(effects, "remove local preference referring to Profile "+preference.ModelProfileID)
		}
	}
	for _, r := range next.Runtimes {
		effects = append(effects, fmt.Sprintf("Runtime %s enabled=%t allowlist=%s", r.ID, r.Enabled, strings.Join(r.AllowlistedProfileIDs, ",")))
	}
	for _, p := range next.ModelProfiles {
		effects = append(effects, "Profile "+p.ID+" capabilities="+strings.Join(p.Capabilities, ",")+" complexities="+strings.Join(p.Complexities, ","))
	}
	sort.Strings(effects)
	return effects
}
