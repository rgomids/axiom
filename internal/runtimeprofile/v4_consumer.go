package runtimeprofile

import "time"

type ConsumerV4 struct {
	cfg    ConfigurationV4
	digest string
}

type OperationalObservation struct {
	RuntimeID            string    `json:"runtimeId"`
	Adapter              string    `json:"adapter"`
	Installed            bool      `json:"installed"`
	Available            bool      `json:"available"`
	ExecutableDigest     string    `json:"executableDigest"`
	InvocationMode       string    `json:"invocationMode"`
	Revision             uint64    `json:"revision"`
	SubscriptionObserved bool      `json:"subscriptionObserved"`
	EffortSourceDigest   string    `json:"effortSourceDigest"`
	ObservedAt           time.Time `json:"-"`
}

func NewConsumerV4(cfg ConfigurationV4) (ConsumerV4, error) {
	digest, err := DigestV4(cfg)
	if err != nil {
		return ConsumerV4{}, err
	}
	return ConsumerV4{cfg: canonicalV4(cfg), digest: digest}, nil
}

func (c ConsumerV4) Configuration() ConfigurationV4 { return canonicalV4(c.cfg) }
func (c ConsumerV4) Digest() string                 { return c.digest }
func (c ConsumerV4) Revision() uint64               { return c.cfg.Revision }

func (c ConsumerV4) EnabledRuntime(runtimeID string) (RuntimeV4, error) {
	if c.digest == "" {
		return RuntimeV4{}, ErrInvalidConfiguration
	}
	for _, r := range c.cfg.Runtimes {
		if r.ID != runtimeID {
			continue
		}
		if !r.Enabled {
			return RuntimeV4{}, ErrRuntimeDisabled
		}
		r.AllowlistedProfileIDs = append([]string(nil), r.AllowlistedProfileIDs...)
		return r, nil
	}
	return RuntimeV4{}, ErrRuntimeAbsent
}

func (c ConsumerV4) AllowedProfile(runtimeID, profileID string) (ModelProfile, error) {
	r, err := c.EnabledRuntime(runtimeID)
	if err != nil {
		return ModelProfile{}, err
	}
	for _, p := range c.cfg.ModelProfiles {
		if p.ID != profileID {
			continue
		}
		if p.RuntimeID != r.ID {
			return ModelProfile{}, ErrProfileRuntimeMismatch
		}
		if !contains(r.AllowlistedProfileIDs, p.ID) {
			return ModelProfile{}, ErrProfileNotAllowlisted
		}
		p.Capabilities = append([]string(nil), p.Capabilities...)
		p.Complexities = append([]string(nil), p.Complexities...)
		return p, nil
	}
	return ModelProfile{}, ErrProfileMissing
}

func (c ConsumerV4) ObservedProfile(runtimeID, profileID string, o OperationalObservation) (ResolvedProfileV4, error) {
	p, err := c.AllowedProfile(runtimeID, profileID)
	if err != nil {
		return ResolvedProfileV4{}, err
	}
	if o.RuntimeID != runtimeID || o.Adapter != runtimeID || o.Revision != c.cfg.Revision || o.ObservedAt.IsZero() || !o.Installed || !o.Available || !o.SubscriptionObserved || o.InvocationMode != "noninteractive" || !validDigest(o.ExecutableDigest) || !validDigest(o.EffortSourceDigest) {
		return ResolvedProfileV4{}, ErrNoMatch
	}
	od, err := operationalObservationDigest(o)
	if err != nil {
		return ResolvedProfileV4{}, err
	}
	pd, err := ProfileDigest(p)
	if err != nil {
		return ResolvedProfileV4{}, err
	}
	resolved := ResolvedProfileV4{ProfileID: p.ID, RuntimeID: p.RuntimeID, Model: p.Model, Effort: p.Effort, ProfileDigest: pd, ConfigurationRevision: c.cfg.Revision, ConfigurationDigest: c.digest, ExecutableDigest: o.ExecutableDigest, InvocationMode: o.InvocationMode, ObservationRevision: o.Revision, ObservationDigest: od, EffortSourceDigest: o.EffortSourceDigest}
	if err := ValidateResolvedProfileV4(resolved); err != nil {
		return ResolvedProfileV4{}, err
	}
	return resolved, nil
}
