package runtimeapplication

import (
	"reflect"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// ConsumerFromRecord classifies one inspected local record state (as
// local.RuntimeProfileStore.Inspect reports it). Only an active, valid
// format-4 record yields a consumer; every other state returns its blocker
// code and is never a default. Historical formats require explicit retirement.
func ConsumerFromRecord(state string, cfg *runtimeprofile.ConfigurationV4) (runtimeprofile.ConsumerV4, string) {
	switch state {
	case "active-v4":
	case "fresh", "retired", "pending", "unsafe", "active-v1", "active-v2", "active-v3":
		return runtimeprofile.ConsumerV4{}, localStateCategory(LocalStateError{State: state})
	default:
		return runtimeprofile.ConsumerV4{}, "configuration_unsafe"
	}
	if cfg == nil {
		return runtimeprofile.ConsumerV4{}, "invalid_configuration"
	}
	consumer, err := runtimeprofile.NewConsumerV4(*cfg)
	if err != nil {
		return runtimeprofile.ConsumerV4{}, "invalid_configuration"
	}
	return consumer, ""
}

// PolicyCandidate is the single policy-admissible Runtime/Profile for one
// request. Requested capabilities are matched only against the Profile's
// declared capabilities: it carries no observation and no proof.
type PolicyCandidate struct {
	ProjectID     string                       `json:"projectId"`
	ProjectDigest string                       `json:"projectDigest"`
	Request       Request                      `json:"request"`
	Profile       runtimeprofile.ProfileIntent `json:"profile"`
}

// EvaluatePolicy intersects portable Project policy with the consumed
// configuration using the exact Preview blocker codes. It selects nothing
// implicitly: several admissible Profiles are ambiguous_match.
func EvaluatePolicy(p project.Project, consumer runtimeprofile.ConsumerV4, request Request) (PolicyCandidate, *runtimeprofile.Blocker) {
	state := p.State()
	if !validRequest(request) || len(project.ValidateIdentity(state.ID, "preview")) != 0 {
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "invalid_request"}
	}
	request.Capabilities = append([]string(nil), request.Capabilities...)
	sort.Strings(request.Capabilities)
	wire, issues := manifest.Encode(p)
	if len(issues) != 0 {
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "invalid_project"}
	}
	if consumer.Digest() == "" {
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "invalid_configuration"}
	}
	if state.SchemaVersion != 1 && !project.MultiRuntimeSchema(state.SchemaVersion) {
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "invalid_project"}
	}
	if !project.RuntimePolicyDeclared(state) {
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "policy_unconfigured"}
	}
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
	permitted := map[string]bool{}
	seen := []string{}
	candidates := []runtimeprofile.ProfileIntent{}
	for _, runtime := range consumer.Configuration().Runtimes {
		if !allowed[runtime.ID] || request.RuntimeID != "" && request.RuntimeID != runtime.ID {
			continue
		}
		for _, id := range runtime.AllowlistedProfileIDs {
			profile, err := consumer.AllowedProfile(runtime.ID, id)
			if err != nil {
				// A disabled Runtime or inconsistent reference is never a candidate.
				continue
			}
			ref, _ := portable[id].RuntimeRef.Value()
			model, _ := portable[id].Model.Value()
			if _, ok := portable[id]; !ok || ref != runtime.ID || model != profile.Model {
				continue
			}
			permitted[id] = true
			seen = append(seen, runtime.ID)
			if request.ModelProfileID != "" && request.ModelProfileID != id || !declaresRequest(profile, request) {
				continue
			}
			intent, err := consumer.ProfileIntent(runtime.ID, id)
			if err != nil {
				return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "invalid_configuration"}
			}
			candidates = append(candidates, intent)
		}
	}
	if len(permitted) == 0 {
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "no_allowed_match"}
	}
	// A portable preference whose Profile is locally unavailable still blocks;
	// another candidate is never substituted.
	preferred := ""
	if preferences, ok := state.RuntimePreferences.Value(); ok && request.ModelProfileID == "" {
		for _, preference := range preferences {
			if preference.Role != request.Role || preference.Complexity != request.Complexity {
				continue
			}
			if !permitted[preference.ModelProfileRef] {
				return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "no_allowed_match"}
			}
			if preferred == "" {
				preferred = preference.ModelProfileRef
			}
		}
	}
	if preferred != "" {
		filtered := candidates[:0]
		for _, candidate := range candidates {
			if candidate.ProfileID == preferred {
				filtered = append(filtered, candidate)
			}
		}
		candidates = filtered
	}
	switch len(candidates) {
	case 0:
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "no_allowed_match", RuntimeIDs: uniqueSorted(seen)}
	case 1:
	default:
		ids := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			ids = append(ids, candidate.ProfileID)
		}
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "ambiguous_match", ProfileIDs: uniqueSorted(ids)}
	}
	if !runtimeprofile.SafeIdentifier(candidates[0].ProfileID) || !runtimeprofile.SafeIdentifier(candidates[0].Model) {
		return PolicyCandidate{}, &runtimeprofile.Blocker{Code: "unsafe_configuration"}
	}
	return PolicyCandidate{ProjectID: state.ID, ProjectDigest: digest(wire), Request: request, Profile: candidates[0]}, nil
}

// CheckPolicy re-evaluates current inputs and accepts only the exact reviewed
// candidate, including its configuration and Profile digests. It never
// substitutes a newly evaluated candidate on drift.
func CheckPolicy(p project.Project, consumer runtimeprofile.ConsumerV4, expected PolicyCandidate) error {
	current, blocker := EvaluatePolicy(p, consumer, expected.Request)
	if blocker != nil || !reflect.DeepEqual(current, expected) {
		return ErrStale
	}
	return nil
}

// declaresRequest applies the format-2 request rules to a Profile's declared
// intent: capability declaration is a selection filter, never proof.
func declaresRequest(p runtimeprofile.ModelProfile, r Request) bool {
	if len(p.Complexities) > 0 && !contains(p.Complexities, r.Complexity) {
		return false
	}
	for _, c := range r.Capabilities {
		if strings.HasPrefix(c, "reasoning-effort-") || !contains(p.Capabilities, c) {
			return false
		}
	}
	return true
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	out := []string{}
	for _, v := range values {
		if !contains(out, v) {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// RuntimeCandidate discloses one enabled Runtime and the effort intent of
// each Profile it allowlists. It is a readiness disclosure, not a selection.
type RuntimeCandidate struct {
	RuntimeID string                         `json:"runtimeId"`
	Profiles  []runtimeprofile.ProfileIntent `json:"profiles"`
}

// ReadinessCandidates projects every enabled Runtime with its allowed
// Profiles' model and effort intent from an inspected record. It selects no
// Runtime. A non-active or invalid record returns its distinct blocker code
// and no candidates.
func ReadinessCandidates(state string, cfg *runtimeprofile.ConfigurationV4) ([]RuntimeCandidate, string) {
	consumer, code := ConsumerFromRecord(state, cfg)
	if code != "" {
		return nil, code
	}
	candidates := []RuntimeCandidate{}
	for _, runtime := range consumer.Configuration().Runtimes {
		if !runtime.Enabled {
			continue
		}
		candidate := RuntimeCandidate{RuntimeID: runtime.ID, Profiles: []runtimeprofile.ProfileIntent{}}
		for _, id := range runtime.AllowlistedProfileIDs {
			intent, err := consumer.ProfileIntent(runtime.ID, id)
			if err != nil {
				return nil, "invalid_configuration"
			}
			candidate.Profiles = append(candidate.Profiles, intent)
		}
		candidates = append(candidates, candidate)
	}
	return candidates, ""
}

func EvaluateAvailability(p project.Project, consumer runtimeprofile.ConsumerV4) string {
	if !p.Equivalent(p) {
		return "invalid_project"
	}
	if !project.RuntimePolicyDeclared(p.State()) {
		return "policy_unconfigured"
	}
	if consumer.Digest() == "" {
		return "invalid_configuration"
	}
	permitted := permittedProfiles(p.State(), consumer)
	if len(permitted) == 0 {
		return "no_allowed_match"
	}
	for _, runtime := range consumer.Configuration().Runtimes {
		if runtime.Enabled && len(permitted[runtime.ID]) != 0 {
			return "inventory_unavailable"
		}
	}
	return "runtime_unavailable"
}
