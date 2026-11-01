package runtimeapplication

import "github.com/rgomids/axiom/internal/runtimeprofile"

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
