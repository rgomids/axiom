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

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

var ErrBlocked = errors.New("project runtime resolution blocked")
var ErrStale = errors.New("project runtime resolution stale")

type Request struct {
	Role           string   `json:"role"`
	Complexity     string   `json:"complexity"`
	Capabilities   []string `json:"capabilities"`
	RuntimeID      string   `json:"runtimeId,omitempty"`
	ModelProfileID string   `json:"modelProfileId,omitempty"`
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
// Observer must return current authoritative observations on each call; an
// old captured inventory cannot establish freshness for a changed machine.
// Consumer is the canonical format-4 record (AXM-36 D2).
type Snapshot struct {
	Project  project.Project
	Consumer runtimeprofile.ConsumerV4
	Observer runtimeprofile.OperationalObserver
}
type Source interface {
	Load(context.Context, string) (Snapshot, error)
}
type Service struct{ source Source }

func New(source Source) Service { return Service{source: source} }

func (s Service) Preview(ctx context.Context, projectID string, request Request) (Preview, error) {
	return s.preview(ctx, projectID, request)
}

// Binding is the concrete dispatch binding that Check re-derives from the same
// snapshot that reproduced the reviewed preview. Canonical configuration
// carries no credential reference, so CredentialReference stays empty.
type Binding struct {
	Choice              runtimeprofile.Choice
	CredentialReference string `json:"-"`
}

func (s Service) preview(ctx context.Context, projectID string, request Request) (Preview, error) {
	result := Preview{}
	if !validRequest(request) || len(project.ValidateIdentity(projectID, "preview")) != 0 {
		return deny(result, "invalid_request")
	}
	result.ProjectID = projectID
	request.Capabilities = append([]string(nil), request.Capabilities...)
	sort.Strings(request.Capabilities)
	result.Request = request
	if s.source == nil {
		return deny(result, "inventory_unavailable")
	}
	snapshot, err := s.source.Load(ctx, projectID)
	if err != nil {
		return deny(result, localStateCategory(err))
	}
	if snapshot.Project.State().ID != projectID {
		return deny(result, "invalid_project")
	}
	wire, issues := manifest.Encode(snapshot.Project)
	if len(issues) != 0 {
		return deny(result, "invalid_project")
	}
	result.ProjectDigest = digest(wire)
	consumer := snapshot.Consumer
	if consumer.Digest() == "" {
		return deny(result, "invalid_configuration")
	}
	result.ConfigurationDigest = consumer.Digest()
	result.ConfigurationRevision = consumer.Revision()
	// Selection is the exact policy intersection; several admissible Profiles
	// are ambiguous_match and nothing is selected implicitly.
	candidate, blocker := EvaluatePolicy(snapshot.Project, consumer, request)
	if blocker != nil {
		return deny(result, blocker.Code)
	}
	if snapshot.Observer == nil {
		return deny(result, "inventory_unavailable")
	}
	runtimeID := candidate.Profile.RuntimeID
	observation, err := snapshot.Observer.ObserveOperational(ctx, runtimeID, candidate.Profile.ProfileID)
	if err != nil {
		return deny(result, observationCategory(ctx, err))
	}
	result.ObservationRevisions = map[string]uint64{runtimeID: observation.Revision}
	resolved, err := consumer.ObservedProfile(runtimeID, candidate.Profile.ProfileID, observation)
	if err != nil {
		return deny(result, "invalid_observation")
	}
	result.ObservationDigest = resolved.ObservationDigest
	choice := runtimeprofile.Choice{FrozenProfile: &resolved, RuntimeID: runtimeID, Adapter: candidate.Profile.Adapter, ModelProfileID: resolved.ProfileID, Model: resolved.Model, Capabilities: append([]string(nil), request.Capabilities...), ExecutableDigest: resolved.ExecutableDigest, ConfigurationRevision: resolved.ConfigurationRevision, ObservationRevision: resolved.ObservationRevision}
	result.Choice = &choice
	return result, nil
}

func observationCategory(ctx context.Context, err error) string {
	switch {
	case ctx.Err() != nil:
		return "observation_cancelled"
	case errors.Is(err, runtimeprofile.ErrExecutableUnavailable):
		return "executable_unavailable"
	case errors.Is(err, runtimeprofile.ErrInspectionConsentRequired):
		return "inspection_consent_required"
	case errors.Is(err, runtimeprofile.ErrEffortOverrideConflict):
		return "effort_override_conflict"
	case errors.Is(err, runtimeprofile.ErrConfigurationUnsafe):
		return "configuration_unsafe"
	}
	return "invalid_observation"
}

// Check re-reads all inputs and accepts only the exact previewed decision.
// It never substitutes a newly resolved choice on drift. The returned binding
// comes from the configuration whose digest the reviewed preview carries.
func (s Service) Check(ctx context.Context, expected Preview) (Binding, error) {
	if expected.Choice == nil || expected.Blocker != nil {
		return Binding{}, ErrBlocked
	}
	current, err := s.preview(ctx, expected.ProjectID, expected.Request)
	if err != nil || !reflect.DeepEqual(current, expected) {
		return Binding{}, ErrStale
	}
	choice := *expected.Choice
	if choice.FrozenProfile != nil {
		p := *choice.FrozenProfile
		choice.FrozenProfile = &p
	}
	choice.Capabilities = append([]string(nil), choice.Capabilities...)
	return Binding{Choice: choice}, nil
}

// Availability is the request-independent readiness projection of the same
// portable/local intersection Preview uses (Issue #231). It selects nothing:
// "" means at least one allowed, enabled Runtime with an allowed profile has a
// currently resolvable executable. Otherwise it returns the #140 blocker code
// (policy_unconfigured, no_allowed_match, runtime_unavailable, ...).
// Role-specific preferences, capabilities and effort sources are checked only
// by Preview; authentication is observed before each attempt.
func (s Service) Availability(ctx context.Context, projectID string) string {
	if len(project.ValidateIdentity(projectID, "preview")) != 0 {
		return "invalid_request"
	}
	if s.source == nil {
		return "inventory_unavailable"
	}
	snapshot, err := s.source.Load(ctx, projectID)
	if err != nil {
		return localStateCategory(err)
	}
	if snapshot.Project.State().ID != projectID {
		return "invalid_project"
	}
	consumer := snapshot.Consumer
	if consumer.Digest() == "" {
		return "invalid_configuration"
	}
	state := snapshot.Project.State()
	if state.SchemaVersion != 1 && !project.MultiRuntimeSchema(state.SchemaVersion) {
		return "invalid_project"
	}
	if !project.RuntimePolicyDeclared(state) {
		return "policy_unconfigured"
	}
	permitted := permittedProfiles(state, consumer)
	if len(permitted) == 0 {
		return "no_allowed_match"
	}
	if snapshot.Observer == nil {
		return "inventory_unavailable"
	}
	blocker := "runtime_unavailable"
	for _, runtime := range consumer.Configuration().Runtimes {
		if !runtime.Enabled || len(permitted[runtime.ID]) == 0 {
			continue
		}
		observation, err := snapshot.Observer.ObserveExecutable(ctx, runtime.ID)
		if err != nil || !observation.Installed || !observation.Available {
			continue
		}
		if observation.RuntimeID != runtime.ID || observation.Adapter != runtime.Adapter || observation.Revision != consumer.Revision() || observation.ObservedAt.IsZero() {
			blocker = "stale_observation"
			continue
		}
		return ""
	}
	return blocker
}

// permittedProfiles is the request-independent policy intersection: every
// Runtime the Project allows with the Profiles it owns and allowlists whose
// portable reference names the same Runtime and model. Enablement is checked
// by the caller, so a disabled Runtime stays runtime_unavailable.
func permittedProfiles(state project.State, consumer runtimeprofile.ConsumerV4) map[string][]string {
	allowed := map[string]bool{}
	for _, id := range project.AllowedRuntimes(state) {
		allowed[id] = true
	}
	portableProfiles, _ := state.ModelProfiles.Value()
	portable := map[string]project.ModelProfile{}
	for _, profile := range portableProfiles {
		if profile.State.Form() != project.NotConfigured {
			portable[profile.Key] = profile
		}
	}
	cfg := consumer.Configuration()
	out := map[string][]string{}
	for _, runtime := range cfg.Runtimes {
		if !allowed[runtime.ID] {
			continue
		}
		for _, id := range runtime.AllowlistedProfileIDs {
			p, ok := portable[id]
			if !ok {
				continue
			}
			ref, _ := p.RuntimeRef.Value()
			model, _ := p.Model.Value()
			for _, profile := range cfg.ModelProfiles {
				if profile.ID == id && profile.RuntimeID == runtime.ID && ref == runtime.ID && model == profile.Model {
					out[runtime.ID] = append(out[runtime.ID], id)
				}
			}
		}
	}
	return out
}

// Digest is an operator review token, not authority for external effects.
func (p Preview) Digest() string { wire, _ := json.Marshal(p); return digest(wire) }
func deny(p Preview, code string) (Preview, error) {
	p.Choice = nil
	p.Blocker = &runtimeprofile.Blocker{Code: code}
	return p, ErrBlocked
}
func digest(wire []byte) string { d := sha256.Sum256(wire); return hex.EncodeToString(d[:]) }

func validRequest(r Request) bool {
	if !token(r.Role) || !token(r.Complexity) || r.RuntimeID != "" && !token(r.RuntimeID) || r.ModelProfileID != "" && !token(r.ModelProfileID) || len(r.Capabilities) == 0 || len(r.Capabilities) > 32 {
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
