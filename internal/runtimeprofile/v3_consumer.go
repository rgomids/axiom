package runtimeprofile

import "errors"

// ConsumerV3 is the typed, read-only consumer boundary over one validated
// canonical format-3 record (Specification 007 AXM-22 BLD-001B.3c-2b.1). It
// reads ConfigurationV3 directly: there is no projection onto the format-1/2
// Configuration, Runtime-owned EffortDeclarations are retained, and identity
// is DigestV3. It never selects an alternative Runtime or Profile, applies no
// implicit default and proves no capability. No production consumer is wired
// to it until coordinated v3-only activation (BLD-001B.3c-2b.3).
type ConsumerV3 struct {
	cfg    ConfigurationV3
	digest string
}

// RecordStateV3 is the explicit local record state a consumer observed. It
// keeps an absent record, a retired record (tombstone after removal of the
// final Runtime) and an invalid record distinct; none of them is a default.
type RecordStateV3 int

const (
	RecordMissingV3 RecordStateV3 = iota + 1
	RecordRetiredV3
	RecordActiveV3
)

// Consumer refusals. Codes already used by runtime authoring keep their
// existing spelling; ErrInvalidConfiguration and ErrUnsupportedConfigurationFormat
// remain the invalid and historical-format refusals.
var (
	ErrConfigurationMissingV3 = errors.New("runtime_profile_configuration_missing")
	ErrConfigurationRetiredV3 = errors.New("runtime_profile_configuration_retired")
	ErrRuntimeAbsent          = errors.New("runtime_absent")
	ErrRuntimeDisabled        = errors.New("runtime_disabled")
	ErrProfileMissing         = errors.New("profile_missing")
	ErrProfileRuntimeMismatch = errors.New("profile_runtime_mismatch")
	ErrProfileNotAllowlisted  = errors.New("profile_not_allowlisted")
)

// ProfileIntentV3 is a ModelProfile's effort intent bound to its exact
// Runtime and the configuration identity it was read from. It carries no
// EffortDeclaration (the Runtime owns those), no observation and no proof.
type ProfileIntentV3 struct {
	RuntimeID             string
	Adapter               string
	ProfileID             string
	Model                 string
	Effort                Effort
	ProfileDigest         string
	ConfigurationRevision uint64
	ConfigurationDigest   string
}

// NewConsumerV3 accepts only an active record holding a valid format-3
// configuration. A configuration supplied with a missing or retired state, or
// an active state without one, is inconsistent and refused as invalid.
func NewConsumerV3(state RecordStateV3, cfg *ConfigurationV3) (ConsumerV3, error) {
	switch {
	case state == RecordMissingV3 && cfg == nil:
		return ConsumerV3{}, ErrConfigurationMissingV3
	case state == RecordRetiredV3 && cfg == nil:
		return ConsumerV3{}, ErrConfigurationRetiredV3
	case state != RecordActiveV3 || cfg == nil || ValidateV3(*cfg) != nil:
		return ConsumerV3{}, ErrInvalidConfiguration
	}
	digest, err := DigestV3(*cfg)
	if err != nil {
		return ConsumerV3{}, ErrInvalidConfiguration
	}
	return ConsumerV3{cfg: canonicalV3(*cfg), digest: digest}, nil
}

// ConsumeVersioned admits only format 3. Historical format-1/2 records are
// refused, never upgraded; there is no v2-to-v3 migration (PM-1).
func ConsumeVersioned(v VersionedConfiguration) (ConsumerV3, error) {
	if v.FormatVersion != FormatVersionV3 || v.Configuration != nil {
		if v.FormatVersion == FormatVersion || v.FormatVersion == FormatVersionV2 {
			return ConsumerV3{}, ErrUnsupportedConfigurationFormat
		}
		return ConsumerV3{}, ErrInvalidConfiguration
	}
	return NewConsumerV3(RecordActiveV3, v.V3)
}

func (c ConsumerV3) Revision() uint64 { return c.cfg.Revision }

// Digest is DigestV3 of the consumed configuration.
func (c ConsumerV3) Digest() string { return c.digest }

// Configuration returns an independent canonical copy, including every
// EffortDeclaration.
func (c ConsumerV3) Configuration() ConfigurationV3 { return canonicalV3(c.cfg) }

// EnabledRuntime returns exactly runtimeID when it exists and is enabled.
func (c ConsumerV3) EnabledRuntime(runtimeID string) (RuntimeV3, error) {
	if c.digest == "" {
		return RuntimeV3{}, ErrInvalidConfiguration
	}
	for _, r := range c.cfg.Runtimes {
		if r.ID != runtimeID {
			continue
		}
		if !r.Enabled {
			return RuntimeV3{}, ErrRuntimeDisabled
		}
		r.AllowlistedProfileIDs = append([]string(nil), r.AllowlistedProfileIDs...)
		declarations := make([]EffortDeclaration, len(r.EffortDeclarations))
		for i, d := range r.EffortDeclarations {
			declarations[i] = cloneDeclaration(d)
		}
		if len(declarations) == 0 {
			declarations = nil
		}
		r.EffortDeclarations = declarations
		return r, nil
	}
	return RuntimeV3{}, ErrRuntimeAbsent
}

// AllowedProfile returns exactly profileID when the enabled Runtime runtimeID
// owns and allowlists it.
func (c ConsumerV3) AllowedProfile(runtimeID, profileID string) (ModelProfile, error) {
	runtime, err := c.EnabledRuntime(runtimeID)
	if err != nil {
		return ModelProfile{}, err
	}
	for _, p := range c.cfg.ModelProfiles {
		if p.ID != profileID {
			continue
		}
		if p.RuntimeID != runtime.ID {
			return ModelProfile{}, ErrProfileRuntimeMismatch
		}
		if !contains(runtime.AllowlistedProfileIDs, p.ID) {
			return ModelProfile{}, ErrProfileNotAllowlisted
		}
		p.Capabilities = append([]string(nil), p.Capabilities...)
		p.Complexities = append([]string(nil), p.Complexities...)
		return p, nil
	}
	return ModelProfile{}, ErrProfileMissing
}

// ProfileIntent projects the allowed Profile's model and effort intent.
func (c ConsumerV3) ProfileIntent(runtimeID, profileID string) (ProfileIntentV3, error) {
	p, err := c.AllowedProfile(runtimeID, profileID)
	if err != nil {
		return ProfileIntentV3{}, err
	}
	runtime, _ := c.EnabledRuntime(runtimeID)
	pd, err := ProfileDigest(p)
	if err != nil {
		return ProfileIntentV3{}, ErrInvalidConfiguration
	}
	return ProfileIntentV3{RuntimeID: runtime.ID, Adapter: runtime.Adapter, ProfileID: p.ID, Model: p.Model, Effort: p.Effort, ProfileDigest: pd, ConfigurationRevision: c.cfg.Revision, ConfigurationDigest: c.digest}, nil
}

// EffortDeclaration returns the enabled Runtime's declaration applicable to
// the independently observed identity a. A missing or inapplicable
// declaration is effort_configuration_unproven, never evidence of support.
func (c ConsumerV3) EffortDeclaration(a EffortApplicability) (EffortDeclaration, error) {
	if _, err := c.EnabledRuntime(a.RuntimeID); err != nil {
		return EffortDeclaration{}, err
	}
	cfg := c.cfg
	return ApplicableEffortDeclaration(VersionedConfiguration{FormatVersion: FormatVersionV3, V3: &cfg}, a)
}
