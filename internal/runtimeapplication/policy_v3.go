package runtimeapplication

import (
	"errors"
	"reflect"
	"sort"
	"strings"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 BLD-001B.3c-2b.2 prepared the format-3 consumers
// below and BLD-001B.3c-2b.3 activated them in Preview, Check, Availability
// and readiness. They read runtimeprofile.ConsumerV3 directly and never
// project onto the format-1/2 Configuration.

// ConsumerFromRecordV3 classifies one inspected local record state (as
// local.RuntimeProfileStore.Inspect reports it) through RecordStateV3.
// Only an active, valid format-3 record yields a consumer; every other state
// returns the established blocker code and is never a default.
func ConsumerFromRecordV3(state string, cfg *runtimeprofile.ConfigurationV3) (runtimeprofile.ConsumerV3, string) {
	record := runtimeprofile.RecordStateV3(0)
	switch state {
	case "fresh":
		record = runtimeprofile.RecordMissingV3
	case "retired":
		record = runtimeprofile.RecordRetiredV3
	case "active-v3":
		record = runtimeprofile.RecordActiveV3
	case "active-v1", "active-v2", "pending", "unsafe":
		// Historical formats are refused, never upgraded or reinterpreted.
		return runtimeprofile.ConsumerV3{}, localStateCategory(LocalStateError{State: state})
	default:
		return runtimeprofile.ConsumerV3{}, "configuration_unsafe"
	}
	if record != runtimeprofile.RecordActiveV3 {
		cfg = nil
	}
	consumer, err := runtimeprofile.NewConsumerV3(record, cfg)
	switch {
	case errors.Is(err, runtimeprofile.ErrConfigurationMissingV3):
		return consumer, "configuration_missing"
	case errors.Is(err, runtimeprofile.ErrConfigurationRetiredV3):
		return consumer, "configuration_retired"
	case err != nil:
		return consumer, "invalid_configuration"
	}
	return consumer, ""
}

// PolicyCandidateV3 is the single policy-admissible Runtime/Profile for one
// request. Requested capabilities are matched only against the Profile's
// declared capabilities: it carries no observation, ResolvedProfile or proof,
// and an EffortDeclaration never contributes to it.
type PolicyCandidateV3 struct {
	ProjectID     string                         `json:"projectId"`
	ProjectDigest string                         `json:"projectDigest"`
	Request       Request                        `json:"request"`
	Profile       runtimeprofile.ProfileIntentV3 `json:"profile"`
}

// EvaluatePolicyV3 intersects portable Project policy with the consumed
// format-3 configuration using the exact Preview blocker codes. It selects
// nothing implicitly: several admissible Profiles are ambiguous_match.
func EvaluatePolicyV3(p project.Project, consumer runtimeprofile.ConsumerV3, request Request) (PolicyCandidateV3, *runtimeprofile.Blocker) {
	state := p.State()
	if !validRequest(request) || len(project.ValidateIdentity(state.ID, "preview")) != 0 {
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "invalid_request"}
	}
	request.Capabilities = append([]string(nil), request.Capabilities...)
	sort.Strings(request.Capabilities)
	wire, issues := manifest.Encode(p)
	if len(issues) != 0 {
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "invalid_project"}
	}
	if consumer.Digest() == "" {
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "invalid_configuration"}
	}
	if state.SchemaVersion != 1 && !project.MultiRuntimeSchema(state.SchemaVersion) {
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "invalid_project"}
	}
	if !project.RuntimePolicyDeclared(state) {
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "policy_unconfigured"}
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
	candidates := []runtimeprofile.ProfileIntentV3{}
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
				return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "invalid_configuration"}
			}
			candidates = append(candidates, intent)
		}
	}
	if len(permitted) == 0 {
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "no_allowed_match"}
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
				return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "no_allowed_match"}
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
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "no_allowed_match", RuntimeIDs: uniqueSorted(seen)}
	case 1:
	default:
		ids := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			ids = append(ids, candidate.ProfileID)
		}
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "ambiguous_match", ProfileIDs: uniqueSorted(ids)}
	}
	if !runtimeprofile.SafeIdentifier(candidates[0].ProfileID) || !runtimeprofile.SafeIdentifier(candidates[0].Model) {
		return PolicyCandidateV3{}, &runtimeprofile.Blocker{Code: "unsafe_configuration"}
	}
	return PolicyCandidateV3{ProjectID: state.ID, ProjectDigest: digest(wire), Request: request, Profile: candidates[0]}, nil
}

// CheckPolicyV3 re-evaluates current inputs and accepts only the exact
// reviewed candidate, including its DigestV3 and ProfileDigest. It never
// substitutes a newly evaluated candidate on drift.
func CheckPolicyV3(p project.Project, consumer runtimeprofile.ConsumerV3, expected PolicyCandidateV3) error {
	current, blocker := EvaluatePolicyV3(p, consumer, expected.Request)
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

// RuntimeCandidateV3 discloses one enabled Runtime and the effort intent of
// each Profile it allowlists. It is a readiness disclosure, not a selection.
type RuntimeCandidateV3 struct {
	RuntimeID string                           `json:"runtimeId"`
	Profiles  []runtimeprofile.ProfileIntentV3 `json:"profiles"`
}

// ReadinessCandidatesV3 projects every enabled Runtime with its allowed
// Profiles' model and effort intent from an inspected record. It selects no
// Runtime and reads no EffortDeclaration. A non-active or invalid record
// returns its distinct blocker code and no candidates.
func ReadinessCandidatesV3(state string, cfg *runtimeprofile.ConfigurationV3) ([]RuntimeCandidateV3, string) {
	consumer, code := ConsumerFromRecordV3(state, cfg)
	if code != "" {
		return nil, code
	}
	candidates := []RuntimeCandidateV3{}
	for _, runtime := range consumer.Configuration().Runtimes {
		if !runtime.Enabled {
			continue
		}
		candidate := RuntimeCandidateV3{RuntimeID: runtime.ID, Profiles: []runtimeprofile.ProfileIntentV3{}}
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
