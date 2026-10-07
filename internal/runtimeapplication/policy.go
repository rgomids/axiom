// Package runtimeapplication composes portable Project policy with local Runtime
// configuration. It owns no credentials and performs no Runtime dispatch.
package runtimeapplication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var ErrBlocked = errors.New("project runtime resolution blocked")
var ErrStale = errors.New("project runtime resolution stale")

type Request struct {
	Role         string   `json:"role"`
	Complexity   string   `json:"complexity"`
	Capabilities []string `json:"capabilities"`
	RuntimeID    string   `json:"runtimeId,omitempty"`
}

type Preview struct {
	ProjectID             string                  `json:"projectId"`
	Request               Request                 `json:"request"`
	Choice                *runtimeprofile.Choice  `json:"choice,omitempty"`
	Blocker               *runtimeprofile.Blocker `json:"blocker,omitempty"`
	ProjectDigest         string                  `json:"projectDigest,omitempty"`
	ConfigurationDigest   string                  `json:"configurationDigest,omitempty"`
	ObservationDigest     string                  `json:"observationDigest,omitempty"`
	ConfigurationRevision uint64                  `json:"configurationRevision,omitempty"`
	ObservationRevisions  map[string]uint64       `json:"observationRevisions,omitempty"`
}

// Snapshot contains independently read portable and machine-local inputs.
// Observer must return current authoritative observations on each Load/Observe;
// an old captured inventory cannot establish freshness for a changed machine.
type Snapshot struct {
	Project       project.Project
	Configuration runtimeprofile.Configuration
	Observer      runtimeprofile.Observer
}
type Source interface {
	Load(context.Context, string) (Snapshot, error)
}
type Service struct{ source Source }

func New(source Source) Service { return Service{source: source} }

func (s Service) Preview(ctx context.Context, projectID string, request Request) (Preview, error) {
	result, _, err := s.preview(ctx, projectID, request)
	return result, err
}

// Binding is the concrete dispatch binding that Check re-derives from the same
// snapshot that reproduced the reviewed preview. It is internal: Preview and
// output carry only the configuration digest, never the credential reference.
type Binding struct {
	Choice              runtimeprofile.Choice
	CredentialReference string `json:"-"`
}

func (s Service) preview(ctx context.Context, projectID string, request Request) (Preview, runtimeprofile.Configuration, error) {
	result := Preview{}
	if !validRequest(request) || len(project.ValidateIdentity(projectID, "preview")) != 0 {
		return denyWith(result, "invalid_request")
	}
	result.ProjectID = projectID
	request.Capabilities = append([]string(nil), request.Capabilities...)
	sort.Strings(request.Capabilities)
	result.Request = request
	if s.source == nil {
		return denyWith(result, "inventory_unavailable")
	}
	snapshot, err := s.source.Load(ctx, projectID)
	if err != nil {
		return denyWith(result, "policy_unavailable")
	}
	if snapshot.Project.State().ID != projectID {
		return denyWith(result, "invalid_project")
	}
	wire, issues := manifest.Encode(snapshot.Project)
	if len(issues) != 0 {
		return denyWith(result, "invalid_project")
	}
	result.ProjectDigest = digest(wire)
	result.ConfigurationDigest, err = runtimeprofile.Digest(snapshot.Configuration)
	if err != nil {
		return denyWith(result, "invalid_configuration")
	}
	result.ConfigurationRevision = snapshot.Configuration.Revision
	cfg, code := projectConfiguration(snapshot.Project.State(), snapshot.Configuration, request)
	if code != "" {
		return denyWith(result, code)
	}
	if snapshot.Observer == nil {
		return denyWith(result, "inventory_unavailable")
	}
	result.ObservationRevisions = map[string]uint64{}
	captured := capture{observations: map[string]runtimeprofile.Observation{}}
	observed := make([]runtimeprofile.Observation, 0, len(cfg.Runtimes))
	for _, runtime := range cfg.Runtimes {
		observation, err := snapshot.Observer.Observe(ctx, runtime.ID)
		if err != nil {
			observation = runtimeprofile.Observation{RuntimeID: runtime.ID, Adapter: runtime.Adapter}
		}
		// Raw version text never crosses the presentation boundary.
		if !safeObservation(observation) {
			return denyWith(result, "invalid_observation")
		}
		result.ObservationRevisions[runtime.ID] = observation.Revision
		captured.observations[runtime.ID] = observation
		// Timestamp denotes collection, not machine revision; refreshed timestamps
		// alone do not invalidate an otherwise identical authoritative observation.
		if !observation.ObservedAt.IsZero() {
			observation.ObservedAt = time.Unix(1, 0).UTC()
		}
		observed = append(observed, observation)
	}
	sort.Slice(observed, func(i, j int) bool { return observed[i].RuntimeID < observed[j].RuntimeID })
	observationWire, marshalErr := json.Marshal(observed)
	if marshalErr != nil {
		return denyWith(result, "invalid_observation")
	}
	result.ObservationDigest = digest(observationWire)
	resolution, err := runtimeprofile.NewResolver(captured).Resolve(ctx, cfg, runtimeprofile.Request{
		ConfigurationRevision: cfg.Revision, Role: request.Role, Complexity: request.Complexity, Capabilities: request.Capabilities,
	})
	if err != nil {
		return denyWith(result, resolution.Blocker.Code)
	}
	result.Choice = resolution.Choice
	return result, snapshot.Configuration, nil
}

// Check re-reads all inputs and accepts only the exact previewed decision.
// It never substitutes a newly resolved choice on drift. The returned binding
// comes from the configuration whose digest the reviewed preview carries.
func (s Service) Check(ctx context.Context, expected Preview) (Binding, error) {
	if expected.Choice == nil || expected.Blocker != nil {
		return Binding{}, ErrBlocked
	}
	current, cfg, err := s.preview(ctx, expected.ProjectID, expected.Request)
	if err != nil || !reflect.DeepEqual(current, expected) {
		return Binding{}, ErrStale
	}
	choice := *expected.Choice
	choice.Capabilities = append([]string(nil), choice.Capabilities...)
	for _, runtime := range cfg.Runtimes {
		if runtime.ID == choice.RuntimeID && runtime.Adapter == choice.Adapter {
			return Binding{Choice: choice, CredentialReference: runtime.CredentialReference}, nil
		}
	}
	return Binding{}, ErrStale
}

// Availability is the request-independent readiness projection of the same
// portable/local intersection Preview uses (Issue #231). It selects nothing:
// "" means at least one allowed, enabled Runtime with an allowed profile is
// currently observed installed and available. Otherwise it returns the #140
// blocker code (policy_unconfigured, no_allowed_match, runtime_unavailable, ...).
// Role-specific preferences and capabilities are checked only by Preview.
func (s Service) Availability(ctx context.Context, projectID string) string {
	if len(project.ValidateIdentity(projectID, "preview")) != 0 {
		return "invalid_request"
	}
	if s.source == nil {
		return "inventory_unavailable"
	}
	snapshot, err := s.source.Load(ctx, projectID)
	if err != nil {
		return "policy_unavailable"
	}
	if snapshot.Project.State().ID != projectID {
		return "invalid_project"
	}
	cfg, code := projectConfiguration(snapshot.Project.State(), snapshot.Configuration, Request{})
	if code != "" {
		return code
	}
	if snapshot.Observer == nil {
		return "inventory_unavailable"
	}
	for _, runtime := range cfg.Runtimes {
		if !runtime.Enabled {
			continue
		}
		observation, err := snapshot.Observer.Observe(ctx, runtime.ID)
		if err == nil && observation.Installed && observation.Available {
			return ""
		}
	}
	return "runtime_unavailable"
}

// Digest is an operator review token, not authority for external effects.
func (p Preview) Digest() string { wire, _ := json.Marshal(p); return digest(wire) }
func denyWith(p Preview, code string) (Preview, runtimeprofile.Configuration, error) {
	p, err := deny(p, code)
	return p, runtimeprofile.Configuration{}, err
}
func deny(p Preview, code string) (Preview, error) {
	p.Choice = nil
	p.Blocker = &runtimeprofile.Blocker{Code: code}
	return p, ErrBlocked
}
func digest(wire []byte) string { d := sha256.Sum256(wire); return hex.EncodeToString(d[:]) }

type capture struct {
	observations map[string]runtimeprofile.Observation
}

func (c capture) Observe(_ context.Context, id string) (runtimeprofile.Observation, error) {
	o, ok := c.observations[id]
	if !ok {
		return o, ErrBlocked
	}
	return o, nil
}

func projectConfiguration(state project.State, local runtimeprofile.Configuration, request Request) (runtimeprofile.Configuration, string) {
	if state.SchemaVersion != 1 && !project.MultiRuntimeSchema(state.SchemaVersion) {
		return runtimeprofile.Configuration{}, "invalid_project"
	}
	if !project.RuntimePolicyDeclared(state) {
		return runtimeprofile.Configuration{}, "policy_unconfigured"
	}
	allowed := map[string]bool{}
	for _, id := range project.AllowedRuntimes(state) {
		allowed[id] = true
	}
	portableProfiles, _ := state.ModelProfiles.Value()
	profiles := map[string]project.ModelProfile{}
	for _, profile := range portableProfiles {
		if profile.State.Form() != project.NotConfigured {
			profiles[profile.Key] = profile
		}
	}
	cfg := runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersion, Revision: local.Revision}
	for _, runtime := range local.Runtimes {
		if !allowed[runtime.ID] || request.RuntimeID != "" && request.RuntimeID != runtime.ID {
			continue
		}
		runtime.AllowlistedProfileIDs = append([]string(nil), runtime.AllowlistedProfileIDs...)
		permitted := make([]string, 0, len(runtime.AllowlistedProfileIDs))
		for _, id := range runtime.AllowlistedProfileIDs {
			portable, ok := profiles[id]
			if !ok {
				continue
			}
			ref, _ := portable.RuntimeRef.Value()
			model, _ := portable.Model.Value()
			for _, profile := range local.ModelProfiles {
				if profile.ID == id && profile.RuntimeID == runtime.ID && ref == runtime.ID && profile.Model == model {
					permitted = append(permitted, id)
					cfg.ModelProfiles = append(cfg.ModelProfiles, profile)
				}
			}
		}
		if len(permitted) == 0 {
			continue
		}
		runtime.AllowlistedProfileIDs = permitted
		cfg.Runtimes = append(cfg.Runtimes, runtime)
	}
	if len(cfg.Runtimes) == 0 || len(cfg.ModelProfiles) == 0 {
		return cfg, "no_allowed_match"
	}
	// A portable preference whose profile is locally unavailable still blocks;
	// do not remove it and accidentally choose another profile.
	if preferences, ok := state.RuntimePreferences.Value(); ok {
		for _, preference := range preferences {
			if preference.Role != request.Role || preference.Complexity != request.Complexity {
				continue
			}
			if preference.Role == "" || preference.Complexity == "" {
				return cfg, "invalid_project"
			}
			cfg.Preferences = append(cfg.Preferences, runtimeprofile.Preference{Role: preference.Role, Complexity: preference.Complexity, ModelProfileID: preference.ModelProfileRef})
		}
	}
	// Missing preferred profiles block; another candidate is never substituted.
	for _, preference := range cfg.Preferences {
		exists := false
		for _, profile := range cfg.ModelProfiles {
			if profile.ID == preference.ModelProfileID {
				exists = true
			}
		}
		if !exists {
			return cfg, "no_allowed_match"
		}
	}
	return cfg, ""
}
func validRequest(r Request) bool {
	if !token(r.Role) || !token(r.Complexity) || r.RuntimeID != "" && !token(r.RuntimeID) || len(r.Capabilities) == 0 || len(r.Capabilities) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, c := range r.Capabilities {
		if !token(c) || seen[c] {
			return false
		}
		seen[c] = true
	}
	return true
}
func token(v string) bool {
	if v == "" || len(v) > 256 {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func safeVersion(v string) bool {
	if len(v) > 128 || strings.TrimSpace(v) != v {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune(" ._-()+", c)) {
			return false
		}
	}
	lower := strings.ToLower(v)
	for _, word := range []string{"token", "secret", "password", "bearer", "api_key", "ghp_", "sk-"} {
		if strings.Contains(lower, word) {
			return false
		}
	}
	return true
}

func safeObservation(o runtimeprofile.Observation) bool {
	if !token(o.RuntimeID) || !token(o.Adapter) || !safeVersion(o.Version) || len(o.CapabilityStatus) > 32 {
		return false
	}
	for capability, status := range o.CapabilityStatus {
		if !token(capability) || (status != runtimeprofile.CapabilityDeclared && status != runtimeprofile.CapabilityProven) {
			return false
		}
	}
	return true
}
