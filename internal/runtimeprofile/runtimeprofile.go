// Package runtimeprofile owns machine-local Runtime/Model Profile resolution.
// Concrete model names and credential references are configuration data, not
// Axiom domain identities.
package runtimeprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	FormatVersion         = 1
	FormatVersionV2       = 2
	MaxConfigurationBytes = 256 << 10
	maxRuntimes           = 8
	maxProfiles           = 32
	maxTextBytes          = 256
)

var (
	ErrInvalidConfiguration = errors.New("invalid runtime profile configuration")
	ErrStaleConfiguration   = errors.New("stale runtime profile configuration")
	ErrNoMatch              = errors.New("no allowed runtime profile match")
	ErrAmbiguousMatch       = errors.New("multiple allowed runtime profile matches")
)

type Runtime struct {
	ID                    string   `json:"id"`
	Adapter               string   `json:"adapter"`
	Enabled               bool     `json:"enabled"`
	AllowlistedProfileIDs []string `json:"allowlistedProfileIds"`
	CredentialReference   string   `json:"credentialReference,omitempty"`
}

type ModelProfile struct {
	Effort       Effort   `json:"-"`
	ID           string   `json:"id"`
	RuntimeID    string   `json:"runtimeId"`
	Model        string   `json:"model"`
	Capabilities []string `json:"capabilities"`
	Complexities []string `json:"complexities,omitempty"`
}

type Preference struct {
	Role           string `json:"role"`
	Complexity     string `json:"complexity"`
	ModelProfileID string `json:"modelProfileId"`
}

type Configuration struct {
	FormatVersion int            `json:"formatVersion"`
	Revision      uint64         `json:"revision"`
	Runtimes      []Runtime      `json:"runtimes"`
	ModelProfiles []ModelProfile `json:"modelProfiles"`
	Preferences   []Preference   `json:"preferences,omitempty"`
}

type Request struct {
	ConfigurationRevision uint64   `json:"configurationRevision"`
	Role                  string   `json:"role"`
	Complexity            string   `json:"complexity"`
	Capabilities          []string `json:"capabilities"`
	ModelProfileID        string   `json:"modelProfileId,omitempty"`
}

type Choice struct {
	ResolvedProfile       *ResolvedProfile   `json:"resolvedProfile,omitempty"`
	FrozenProfile         *ResolvedProfileV4 `json:"frozenProfile,omitempty"`
	RuntimeID             string             `json:"runtimeId"`
	Adapter               string             `json:"adapter"`
	ModelProfileID        string             `json:"modelProfileId"`
	Model                 string             `json:"model"`
	Capabilities          []string           `json:"capabilities"`
	RuntimeVersion        string             `json:"runtimeVersion,omitempty"`
	ExecutableDigest      string             `json:"executableDigest,omitempty"`
	ConfigurationRevision uint64             `json:"configurationRevision"`
	ObservationRevision   uint64             `json:"observationRevision"`
}

type Blocker struct {
	Code       string   `json:"code"`
	RuntimeIDs []string `json:"runtimeIds,omitempty"`
	ProfileIDs []string `json:"profileIds,omitempty"`
}

func Validate(cfg Configuration) error {
	// Configuration is the format-1/2 type; format 3 is ConfigurationV3 only.
	if cfg.FormatVersion > FormatVersionV2 {
		return ErrUnsupportedConfigurationFormat
	}
	if (cfg.FormatVersion != FormatVersion && cfg.FormatVersion != FormatVersionV2) || cfg.Revision == 0 || len(cfg.Runtimes) == 0 || len(cfg.Runtimes) > maxRuntimes || len(cfg.ModelProfiles) == 0 || len(cfg.ModelProfiles) > maxProfiles {
		return ErrInvalidConfiguration
	}
	runtimes := make(map[string]Runtime, len(cfg.Runtimes))
	for _, runtime := range cfg.Runtimes {
		if cfg.FormatVersion == FormatVersionV2 && (runtime.CredentialReference != "" || len(runtime.AllowlistedProfileIDs) == 0) {
			return ErrInvalidConfiguration
		}
		if !validToken(runtime.ID) || (runtime.Adapter != "codex" && runtime.Adapter != "claude") || runtime.ID != runtime.Adapter || !validReference(runtime.CredentialReference) || len(runtime.AllowlistedProfileIDs) > maxProfiles {
			return ErrInvalidConfiguration
		}
		if _, exists := runtimes[runtime.ID]; exists || !validTokens(runtime.AllowlistedProfileIDs, false) {
			return ErrInvalidConfiguration
		}
		runtimes[runtime.ID] = runtime
	}
	profiles := make(map[string]ModelProfile, len(cfg.ModelProfiles))
	for _, profile := range cfg.ModelProfiles {
		if cfg.FormatVersion == FormatVersion && profile.Effort != (Effort{}) {
			return ErrInvalidConfiguration
		}
		if cfg.FormatVersion == FormatVersionV2 && !validV2Profile(profile) {
			return ErrInvalidConfiguration
		}
		if !validToken(profile.ID) || !validText(profile.Model) || !validTokens(profile.Capabilities, false) || !validTokens(profile.Complexities, true) {
			return ErrInvalidConfiguration
		}
		if _, exists := runtimes[profile.RuntimeID]; !exists {
			return ErrInvalidConfiguration
		}
		if _, exists := profiles[profile.ID]; exists {
			return ErrInvalidConfiguration
		}
		profiles[profile.ID] = profile
	}
	for _, runtime := range cfg.Runtimes {
		for _, profileID := range runtime.AllowlistedProfileIDs {
			if profile, exists := profiles[profileID]; !exists || profile.RuntimeID != runtime.ID {
				return ErrInvalidConfiguration
			}
		}
	}
	seenPreferences := map[string]bool{}
	for _, preference := range cfg.Preferences {
		key := preference.Role + "\x00" + preference.Complexity
		if !validToken(preference.Role) || !validToken(preference.Complexity) || !validToken(preference.ModelProfileID) || seenPreferences[key] {
			return ErrInvalidConfiguration
		}
		if _, exists := profiles[preference.ModelProfileID]; !exists {
			return ErrInvalidConfiguration
		}
		seenPreferences[key] = true
	}
	return nil
}

func Digest(cfg Configuration) (string, error) {
	if err := Validate(cfg); err != nil {
		return "", err
	}
	normalized := cfg
	normalized.Runtimes = sortedRuntimes(cfg.Runtimes)
	normalized.ModelProfiles = append([]ModelProfile(nil), cfg.ModelProfiles...)
	normalized.Preferences = append([]Preference(nil), cfg.Preferences...)
	sort.Slice(normalized.ModelProfiles, func(i, j int) bool { return normalized.ModelProfiles[i].ID < normalized.ModelProfiles[j].ID })
	for index := range normalized.Runtimes {
		normalized.Runtimes[index].AllowlistedProfileIDs = sortedStrings(normalized.Runtimes[index].AllowlistedProfileIDs)
	}
	for index := range normalized.ModelProfiles {
		normalized.ModelProfiles[index].Capabilities = sortedStrings(normalized.ModelProfiles[index].Capabilities)
		normalized.ModelProfiles[index].Complexities = sortedStrings(normalized.ModelProfiles[index].Complexities)
	}
	sort.Slice(normalized.Preferences, func(i, j int) bool {
		left := normalized.Preferences[i].Role + "\x00" + normalized.Preferences[i].Complexity
		right := normalized.Preferences[j].Role + "\x00" + normalized.Preferences[j].Complexity
		return left < right
	})
	wire, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(wire)
	return hex.EncodeToString(digest[:]), nil
}

func Encode(cfg Configuration) ([]byte, error) {
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	wire, err := json.Marshal(cfg)
	if err != nil || len(wire)+1 > MaxConfigurationBytes {
		return nil, ErrInvalidConfiguration
	}
	return append(wire, '\n'), nil
}

// Decode is the format-1/2 reader used by every existing consumer. A newer
// format is refused explicitly, never downgraded or partially read (D8).
func Decode(wire []byte) (Configuration, error) {
	cfg, err := decodeHistorical(wire)
	if err != nil && probeFormatVersion(wire) > FormatVersionV2 {
		return Configuration{}, ErrUnsupportedConfigurationFormat
	}
	return cfg, err
}

func decodeHistorical(wire []byte) (Configuration, error) {
	if len(wire) == 0 || len(wire) > MaxConfigurationBytes || !uniqueJSON(wire) {
		return Configuration{}, ErrInvalidConfiguration
	}
	decoder := json.NewDecoder(strings.NewReader(string(wire)))
	decoder.DisallowUnknownFields()
	var cfg Configuration
	if err := decoder.Decode(&cfg); err != nil {
		return Configuration{}, ErrInvalidConfiguration
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || Validate(cfg) != nil {
		return Configuration{}, ErrInvalidConfiguration
	}
	if cfg.FormatVersion == FormatVersionV2 {
		if !closedConfigurationV2(wire) {
			return Configuration{}, ErrInvalidConfiguration
		}
		var raw struct {
			Runtimes []map[string]json.RawMessage `json:"runtimes"`
		}
		if json.Unmarshal(wire, &raw) != nil {
			return Configuration{}, ErrInvalidConfiguration
		}
		for _, r := range raw.Runtimes {
			if _, exists := r["credentialReference"]; exists {
				return Configuration{}, ErrInvalidConfiguration
			}
		}
	}
	if cfg.FormatVersion == FormatVersion {
		var raw struct {
			Profiles []map[string]json.RawMessage `json:"modelProfiles"`
		}
		if json.Unmarshal(wire, &raw) != nil {
			return Configuration{}, ErrInvalidConfiguration
		}
		for _, p := range raw.Profiles {
			for key := range p {
				if strings.EqualFold(key, "effort") {
					return Configuration{}, ErrInvalidConfiguration
				}
			}
		}
	}
	return cfg, nil
}

func sortedRuntimes(values []Runtime) []Runtime {
	result := append([]Runtime(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func validTokens(values []string, allowEmpty bool) bool {
	if !allowEmpty && len(values) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !validToken(value) || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func validToken(value string) bool {
	if value == "" || len(value) > maxTextBytes {
		return false
	}
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.') {
			return false
		}
	}
	return true
}

func validText(value string) bool {
	return value != "" && len(value) <= maxTextBytes && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func validReference(value string) bool {
	return value == "" || (validText(value) && !strings.Contains(value, "="))
}

func (b Blocker) Error() string { return fmt.Sprintf("runtime resolution blocked: %s", b.Code) }
