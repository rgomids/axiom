package capabilityproof

import (
	"context"
	"strings"

	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Observer is instantiated for exactly one invocation mode (confinement variant
// and Process). Current facts come from Base and Source on every observation,
// never from evidence itself.
type Observer struct {
	Base    runtimeprofile.Observer
	Service Service
	Targets []Target
}

func (o Observer) Observe(ctx context.Context, id string) (runtimeprofile.Observation, error) {
	if o.Base == nil {
		return runtimeprofile.Observation{}, ErrBlocked
	}
	seen := map[string]bool{}
	for _, t := range o.Targets {
		if t.Variant != o.Targets[0].Variant || t.Process != o.Targets[0].Process || seen[t.RuntimeID] {
			return runtimeprofile.Observation{}, ErrBlocked
		}
		seen[t.RuntimeID] = true
	}
	observation, err := o.Base.Observe(ctx, id)
	if err != nil {
		return observation, err
	}
	// Retain only Axiom's independently observed skill integration.
	integration := observation.CapabilityStatus["axiom-skills"]
	observation.CapabilityStatus = map[string]runtimeprofile.CapabilityStatus{}
	if integration == runtimeprofile.CapabilityProven {
		observation.CapabilityStatus["axiom-skills"] = integration
	}
	observation.NonInteractiveModelCapabilities = map[string]map[string]runtimeprofile.CapabilityStatus{}
	observation.ProfileProofs = nil
	for _, t := range o.Targets {
		if t.RuntimeID != id {
			continue
		}
		c, e, err := o.Service.read(ctx, t)
		if err != nil || !observation.Installed || !observation.Available || c.Binding.ExecutableDigest != observation.ExecutableDigest || c.Binding.Version != observation.Version || c.Model == nil || c.Model.ConfigurationRevision != observation.Revision {
			continue
		}
		statuses := map[string]runtimeprofile.CapabilityStatus{}
		for cap := range e.Capabilities {
			if st, _ := state(c, e, cap, o.Service.now()); st == Proven {
				statuses[cap] = runtimeprofile.CapabilityProven
				if cap != ModelInvocation && !strings.HasPrefix(cap, "reasoning-effort-") {
					observation.CapabilityStatus[cap] = runtimeprofile.CapabilityProven
				}
			}
		}
		observation.NonInteractiveModelCapabilities[c.Model.Model] = statuses
		effort := runtimeprofile.Effort{Mode: c.Model.EffortMode, Value: c.Model.EffortToken}
		proof := runtimeprofile.ProfileProof{RuntimeID: id, Model: c.Model.Model, Effort: effort, ExecutableDigest: c.Binding.ExecutableDigest, RuntimeVersion: c.Binding.Version, InvocationMode: "noninteractive", ObservationRevision: observation.Revision, Reference: c.EvidenceRef, EffortSourceDigest: c.Model.EffortSourceDigest, CapabilityStatus: statuses}
		if statuses[ModelInvocation] == runtimeprofile.CapabilityProven {
			proof.Authentication = runtimeprofile.CapabilityProven
			proof.ModelAvailability = runtimeprofile.CapabilityProven
		}
		if c.Model.EffortMode == "runtime-default" && statuses[ModelInvocation] == runtimeprofile.CapabilityProven || c.Model.EffortMode == "explicit" && statuses["reasoning-effort-"+c.Model.EffortToken] == runtimeprofile.CapabilityProven {
			proof.EffortSupport = runtimeprofile.CapabilityProven
			proof.NoOverride = runtimeprofile.CapabilityProven
		}
		observation.ProfileProofs = append(observation.ProfileProofs, proof)
	}
	return observation, nil
}

// ConfiguredSource intersects current facts with the freshly loaded canonical
// v3 configuration, preventing a removed, disabled or edited Profile from
// retaining proof through an otherwise valid evidence selection.
type ConfiguredSource struct {
	Source   BindingSource
	Consumer runtimeprofile.ConsumerV3
}

func (s ConfiguredSource) Current(ctx context.Context, t Target) (Current, error) {
	if s.Source == nil {
		return Current{}, ErrBlocked
	}
	c, err := s.Source.Current(ctx, t)
	if err != nil {
		return c, err
	}
	active := false
	if p, err := s.Consumer.AllowedProfile(t.RuntimeID, t.ProfileID); err == nil {
		active = true
		pd, err := runtimeprofile.ProfileDigest(p)
		if err != nil || c.Model == nil {
			return Current{}, ErrStalePreview
		}
		model := *c.Model
		model.ProfileDigest, model.Model = pd, p.Model
		model.EffortMode, model.EffortToken = p.Effort.Mode, p.Effort.Value
		model.ConfigurationRevision = s.Consumer.Revision()
		c.Model = &model
	}
	c.ProfileActive = c.ProfileActive && active
	return c, nil
}
