package runtimeprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/portableconfig"
)

const TombstoneKind = "runtime-profile-tombstone"
const EffortExplicit = "explicit"
const EffortRuntimeDefault = "runtime-default"

type Effort struct {
	Mode  string `json:"mode"`
	Value string `json:"value"`
}

// MarshalJSON omits effort only for historical format-1 Profiles.
func (p ModelProfile) MarshalJSON() ([]byte, error) {
	type historical ModelProfile
	var effort *Effort
	if p.Effort != (Effort{}) {
		e := p.Effort
		effort = &e
	}
	return json.Marshal(struct {
		historical
		Effort *Effort `json:"effort,omitempty"`
	}{historical(p), effort})
}
func (p *ModelProfile) UnmarshalJSON(w []byte) error {
	type historical ModelProfile
	var decoded struct {
		historical
		Effort *Effort `json:"effort,omitempty"`
	}
	d := json.NewDecoder(bytes.NewReader(w))
	d.DisallowUnknownFields()
	if err := d.Decode(&decoded); err != nil {
		return ErrInvalidConfiguration
	}
	*p = ModelProfile(decoded.historical)
	if decoded.Effort != nil {
		p.Effort = *decoded.Effort
	}
	// null/missing in v2 is rejected by Validate; any effort key in v1 is
	// separately rejected at version dispatch, including explicit null.
	return nil
}

func SafeIdentifier(v string) bool {
	lower := strings.ToLower(v)
	return validText(v) && portableconfig.SafeValue(v, "logical") && !portableconfig.ContainsSecretBearingValue(v) && !strings.ContainsAny(v, ";|&<>\"'(){}[]") && !strings.HasPrefix(lower, "sk-") && !strings.HasPrefix(lower, "ghp_") && !strings.HasPrefix(lower, "github_pat_")
}
func validV2Profile(p ModelProfile) bool {
	if !SafeIdentifier(p.ID) || !SafeIdentifier(p.Model) || !SafeIdentifier(p.RuntimeID) || !ValidEffort(p.Effort) {
		return false
	}
	for _, c := range p.Capabilities {
		if strings.HasPrefix(c, "reasoning-effort-") || !SafeIdentifier(c) {
			return false
		}
	}
	for _, c := range p.Complexities {
		if !SafeIdentifier(c) {
			return false
		}
	}
	return true
}
func ValidEffort(e Effort) bool {
	return e.Mode == EffortRuntimeDefault && e.Value == "default" || e.Mode == EffortExplicit && SafeIdentifier(e.Value)
}

type Tombstone struct {
	Kind                     string `json:"kind"`
	FormatVersion            int    `json:"formatVersion"`
	Revision                 uint64 `json:"revision"`
	PriorConfigurationDigest string `json:"priorConfigurationDigest"`
}

func ValidateTombstone(t Tombstone) error {
	if t.Kind != TombstoneKind || t.FormatVersion != 1 || t.Revision == 0 || !validDigest(t.PriorConfigurationDigest) {
		return ErrInvalidConfiguration
	}
	return nil
}
func EncodeTombstone(t Tombstone) ([]byte, error) {
	if err := ValidateTombstone(t); err != nil {
		return nil, err
	}
	w, err := json.Marshal(t)
	return append(w, '\n'), err
}
func DecodeTombstone(w []byte) (Tombstone, error) {
	var t Tombstone
	if len(w) == 0 || len(w) > MaxConfigurationBytes || !uniqueJSON(w) || !closedObject(w, []string{"kind", "formatVersion", "revision", "priorConfigurationDigest"}, nil) {
		return t, ErrInvalidConfiguration
	}
	d := json.NewDecoder(bytes.NewReader(w))
	d.DisallowUnknownFields()
	if d.Decode(&t) != nil || d.Decode(&struct{}{}) != io.EOF || ValidateTombstone(t) != nil {
		return Tombstone{}, ErrInvalidConfiguration
	}
	return t, nil
}

// ProfileProof is injected authoritative evidence consumption, not acquisition.
// Unknown facts never become proof because a configuration was successfully saved.
type ProfileProof struct {
	CapabilityStatus    map[string]CapabilityStatus `json:"capabilityStatus"`
	RuntimeID           string                      `json:"runtimeId"`
	Model               string                      `json:"model"`
	Effort              Effort                      `json:"effort"`
	ExecutableDigest    string                      `json:"executableDigest"`
	RuntimeVersion      string                      `json:"runtimeVersion"`
	InvocationMode      string                      `json:"invocationMode"`
	ObservationRevision uint64                      `json:"observationRevision"`
	Reference           string                      `json:"reference"`
	EffortSourceDigest  string                      `json:"effortSourceDigest"`
	Authentication      CapabilityStatus            `json:"authentication"`
	ModelAvailability   CapabilityStatus            `json:"modelAvailability"`
	EffortSupport       CapabilityStatus            `json:"effortSupport"`
	NoOverride          CapabilityStatus            `json:"noOverride"`
}
type ResolvedProfile struct {
	ProfileID                string `json:"profileId"`
	RuntimeID                string `json:"runtimeId"`
	Model                    string `json:"model"`
	Effort                   Effort `json:"effort"`
	ProfileDigest            string `json:"profileDigest"`
	ConfigurationRevision    uint64 `json:"configurationRevision"`
	ConfigurationDigest      string `json:"configurationDigest"`
	ExecutableDigest         string `json:"executableDigest"`
	RuntimeVersion           string `json:"runtimeVersion"`
	InvocationMode           string `json:"invocationMode"`
	ObservationRevision      uint64 `json:"observationRevision"`
	ObservationDigest        string `json:"observationDigest"`
	EffortSourceDigest       string `json:"effortSourceDigest"`
	CapabilityProofReference string `json:"capabilityProofReference"`
}

func ValidateResolvedProfile(p ResolvedProfile) error {
	if !SafeIdentifier(p.ProfileID) || !SafeIdentifier(p.RuntimeID) || !SafeIdentifier(p.Model) || !ValidEffort(p.Effort) || p.ConfigurationRevision == 0 || p.ObservationRevision == 0 || !SafeRuntimeVersion(p.RuntimeVersion) || p.InvocationMode != "noninteractive" || !SafeIdentifier(p.CapabilityProofReference) {
		return ErrInvalidConfiguration
	}
	for _, d := range []string{p.ProfileDigest, p.ConfigurationDigest, p.ExecutableDigest, p.ObservationDigest, p.EffortSourceDigest} {
		if !validDigest(d) {
			return ErrInvalidConfiguration
		}
	}
	return nil
}
func resolveProfile(cfg Configuration, p ModelProfile, o Observation, capabilities []string) (ResolvedProfile, string) {
	code := "effort_unproven"
	for _, proof := range o.ProfileProofs {
		if proof.RuntimeID != p.RuntimeID || proof.Model != p.Model || proof.Effort != p.Effort || proof.ExecutableDigest != o.ExecutableDigest || proof.RuntimeVersion != o.Version || proof.InvocationMode != "noninteractive" || proof.ObservationRevision != o.Revision {
			continue
		}
		if proof.Authentication != CapabilityProven {
			code = "authentication_unproven"
			continue
		}
		if proof.ModelAvailability != CapabilityProven {
			code = "model_unproven"
			continue
		}
		if proof.EffortSupport != CapabilityProven {
			continue
		}
		if proof.NoOverride != CapabilityProven || !validDigest(proof.EffortSourceDigest) {
			code = "effort_configuration_unproven"
			continue
		}
		proven := true
		for _, c := range capabilities {
			if proof.CapabilityStatus[c] != CapabilityProven {
				proven = false
			}
		}
		if !proven {
			code = "capability_unproven"
			continue
		}
		cd, _ := Digest(cfg)
		pd, _ := ProfileDigest(p)
		normalized := o
		// Collection time is not identity; the exact observation revision binds freshness.
		normalized.ObservedAt = time.Unix(1, 0).UTC()
		ow, _ := json.Marshal(normalized)
		r := ResolvedProfile{p.ID, p.RuntimeID, p.Model, p.Effort, pd, cfg.Revision, cd, o.ExecutableDigest, o.Version, proof.InvocationMode, o.Revision, hash(ow), proof.EffortSourceDigest, proof.Reference}
		if ValidateResolvedProfile(r) != nil {
			return ResolvedProfile{}, "invalid_observation"
		}
		return r, ""
	}
	return ResolvedProfile{}, code
}
func hash(w []byte) string { d := sha256.Sum256(w); return hex.EncodeToString(d[:]) }
func validDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// uniqueJSON rejects duplicate keys at every depth and excessive nesting.
func uniqueJSON(w []byte) bool {
	d := json.NewDecoder(bytes.NewReader(w))
	d.UseNumber()
	var read func(int) bool
	read = func(depth int) bool {
		if depth > 32 {
			return false
		}
		t, e := d.Token()
		if e != nil {
			return false
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return true
		}
		if delim == '{' {
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				key, ok := k.(string)
				if e != nil || !ok || seen[key] {
					return false
				}
				seen[key] = true
				if !read(depth + 1) {
					return false
				}
			}
			t, e = d.Token()
			return e == nil && t == json.Delim('}')
		}
		if delim == '[' {
			for d.More() {
				if !read(depth + 1) {
					return false
				}
			}
			t, e = d.Token()
			return e == nil && t == json.Delim(']')
		}
		return false
	}
	if !read(0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}

func ProfileDigest(p ModelProfile) (string, error) {
	if !validV2Profile(p) {
		return "", ErrInvalidConfiguration
	}
	p.Capabilities = sortedStrings(p.Capabilities)
	p.Complexities = sortedStrings(p.Complexities)
	w, err := json.Marshal(p)
	return hash(w), err
}

func SafeRuntimeVersion(v string) bool {
	if !validText(v) || len(v) > 128 || portableconfig.ContainsSecretBearingValue(v) {
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

// ProfileReadiness consumes injected evidence without selecting or dispatching.
func ProfileReadiness(cfg Configuration, p ModelProfile, o Observation) string {
	if cfg.FormatVersion != FormatVersionV2 {
		return "migration_required"
	}
	if !validV2Profile(p) {
		return "unsafe_configuration"
	}
	if len(o.ProfileProofs) == 0 {
		return "authentication_unproven"
	}
	_, code := resolveProfile(cfg, p, o, nil)
	return code
}

// Exact member names close encoding/json's case-insensitive struct aliases.
func closedObject(w []byte, allowed []string, nested map[string]func([]byte) bool) bool {
	var obj map[string]json.RawMessage
	if json.Unmarshal(w, &obj) != nil || obj == nil {
		return false
	}
	for k, v := range obj {
		found := false
		for _, a := range allowed {
			found = found || k == a
		}
		if !found {
			return false
		}
		if check := nested[k]; check != nil && !check(v) {
			return false
		}
	}
	return true
}
func closedArray(w []byte, check func([]byte) bool) bool {
	var items []json.RawMessage
	if json.Unmarshal(w, &items) != nil || items == nil {
		return false
	}
	for _, v := range items {
		if !check(v) {
			return false
		}
	}
	return true
}
func closedConfigurationV2(w []byte) bool {
	effort := func(v []byte) bool { return closedObject(v, []string{"mode", "value"}, nil) }
	profile := func(v []byte) bool {
		return closedObject(v, []string{"id", "runtimeId", "model", "capabilities", "complexities", "effort"}, map[string]func([]byte) bool{"effort": effort})
	}
	runtime := func(v []byte) bool {
		return closedObject(v, []string{"id", "adapter", "enabled", "allowlistedProfileIds"}, nil)
	}
	preference := func(v []byte) bool { return closedObject(v, []string{"role", "complexity", "modelProfileId"}, nil) }
	return closedObject(w, []string{"formatVersion", "revision", "runtimes", "modelProfiles", "preferences"}, map[string]func([]byte) bool{"runtimes": func(v []byte) bool { return closedArray(v, runtime) }, "modelProfiles": func(v []byte) bool { return closedArray(v, profile) }, "preferences": func(v []byte) bool { return closedArray(v, preference) }})
}
