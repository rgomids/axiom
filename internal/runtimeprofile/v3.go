package runtimeprofile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

// Local configuration format 3 (Specification 007 AXM-22 amendment BI-2,
// D1–D8) adds the Runtime-owned EffortDeclaration. It is a separate type from
// the historical format-1/2 Configuration, which remains only for narrow
// pre-stable recognition and retirement (PM-11). EncodeV3 is the only local
// configuration writer. Before the first stable release format 3 is the only
// canonical format; there is no v2-to-v3 migration and no v3-to-v2 downgrade
// (Specification 007 AXM-22 PM-1). Every current consumer is v3-only since
// BLD-001B.3c-2b.3.
const (
	FormatVersionV3          = 3
	EffortDeclarationVersion = 1
	maxEffortDeclarations    = 8
	maxEffortSources         = 32
	maxEffortKeys            = 16
	maxEffortKeyBytes        = 128
	maxEffortPathBytes       = 256
	maxDeclarationIDBytes    = 64
)

// Source-root anchors are closed (D6); host roots are resolved later through
// the existing trusted filesystem boundary and are never persisted.
const (
	EffortAnchorUser    = "user"
	EffortAnchorProject = "project"
	EffortAnchorManaged = "managed"
)

// ErrUnsupportedConfigurationFormat is the explicit refusal of a format a
// consumer does not read (D8). It remains an ErrInvalidConfiguration so
// existing v2-only handling is unchanged.
var ErrUnsupportedConfigurationFormat = fmt.Errorf("%w: unsupported_configuration_format", ErrInvalidConfiguration)

type ConfigurationV3 struct {
	FormatVersion int            `json:"formatVersion"`
	Revision      uint64         `json:"revision"`
	Runtimes      []RuntimeV3    `json:"runtimes"`
	ModelProfiles []ModelProfile `json:"modelProfiles"`
	Preferences   []Preference   `json:"preferences,omitempty"`
}

// RuntimeV3 is the format-2 Runtime plus its effort declarations. The
// enclosing Runtime supplies the declaration's Runtime identity.
type RuntimeV3 struct {
	ID                    string              `json:"id"`
	Adapter               string              `json:"adapter"`
	Enabled               bool                `json:"enabled"`
	AllowlistedProfileIDs []string            `json:"allowlistedProfileIds"`
	EffortDeclarations    []EffortDeclaration `json:"effortDeclarations,omitempty"`
}

// EffortDeclaration is configured effort-source identity for one executable
// and invocation mode. It is not capability Evidence: it carries no capability
// status, and effort intent stays in ModelProfile (ADR-0024).
type EffortDeclaration struct {
	DeclarationID         string                    `json:"declarationId"`
	DeclarationVersion    int                       `json:"declarationVersion"`
	RuntimeVersion        string                    `json:"runtimeVersion"`
	ExecutableDigest      string                    `json:"executableDigest"`
	InvocationMode        string                    `json:"invocationMode"`
	PrecedenceEstablished bool                      `json:"precedenceEstablished"`
	JSONKeys              []string                  `json:"jsonKeys"`
	EnvironmentKeys       []string                  `json:"environmentKeys"`
	Sources               []EffortSourceDeclaration `json:"sources"`
}

type EffortSourceDeclaration struct {
	Scope     string `json:"scope"`
	Anchor    string `json:"anchor"`
	Path      string `json:"path"`
	Format    string `json:"format"`
	Directory bool   `json:"directory"`
}

// VersionedConfiguration is the version-dispatched read result. Formats 1 and
// 2 are decoded by Decode with unchanged semantics and gain no declaration.
type VersionedConfiguration struct {
	FormatVersion int
	Configuration *Configuration
	V3            *ConfigurationV3
	V4            *ConfigurationV4
}

func DecodeVersioned(wire []byte) (VersionedConfiguration, error) {
	switch version := probeFormatVersion(wire); {
	case version == FormatVersion || version == FormatVersionV2:
		cfg, err := Decode(wire)
		if err != nil {
			return VersionedConfiguration{}, err
		}
		return VersionedConfiguration{FormatVersion: cfg.FormatVersion, Configuration: &cfg}, nil
	case version == FormatVersionV3:
		cfg, err := decodeConfigurationV3(wire)
		if err != nil {
			return VersionedConfiguration{}, err
		}
		return VersionedConfiguration{FormatVersion: FormatVersionV3, V3: &cfg}, nil
	case version == FormatVersionV4:
		cfg, err := DecodeV4(wire)
		if err != nil {
			return VersionedConfiguration{}, err
		}
		return VersionedConfiguration{FormatVersion: FormatVersionV4, V4: &cfg}, nil
	case version > FormatVersionV4:
		return VersionedConfiguration{}, ErrUnsupportedConfigurationFormat
	}
	return VersionedConfiguration{}, ErrInvalidConfiguration
}

func ValidateV3(c ConfigurationV3) error {
	if c.FormatVersion != FormatVersionV3 {
		return ErrInvalidConfiguration
	}
	// Every format-2 rule applies unchanged to the Runtime/Profile projection.
	if Validate(projectV2(c)) != nil {
		return ErrInvalidConfiguration
	}
	ids := map[string]bool{}
	for _, r := range c.Runtimes {
		if len(r.EffortDeclarations) > maxEffortDeclarations {
			return ErrInvalidConfiguration
		}
		applicability := map[string]bool{}
		for _, d := range r.EffortDeclarations {
			key := d.RuntimeVersion + "\x00" + d.ExecutableDigest + "\x00" + d.InvocationMode
			// Unique declarationId keeps the ProofReference mapping 1:1; one
			// declaration per applicability tuple keeps selection unambiguous.
			if !validEffortDeclaration(r.ID, d) || ids[d.DeclarationID] || applicability[key] {
				return ErrInvalidConfiguration
			}
			ids[d.DeclarationID] = true
			applicability[key] = true
		}
	}
	return nil
}

func validEffortDeclaration(runtimeID string, d EffortDeclaration) bool {
	if !validDeclarationID(d.DeclarationID) || d.DeclarationVersion != EffortDeclarationVersion || !SafeRuntimeVersion(d.RuntimeVersion) || !validDigest(d.ExecutableDigest) || d.InvocationMode != "noninteractive" || !d.PrecedenceEstablished {
		return false
	}
	if !validEffortKeys(d.JSONKeys, false) || !validEffortKeys(d.EnvironmentKeys, true) || len(d.Sources) == 0 || len(d.Sources) > maxEffortSources {
		return false
	}
	required := []string{"user", "project", "managed"}
	switch runtimeID {
	case "codex":
		// Codex sources are TOML; declared JSON keys would contradict them.
		if len(d.JSONKeys) != 0 {
			return false
		}
	case "claude":
		if len(d.JSONKeys) == 0 || !contains(d.EnvironmentKeys, "CLAUDE_CODE_EFFORT_LEVEL") {
			return false
		}
		required = append(required, "project-local", "managed-drop-in")
	default:
		return false
	}
	// One source per scope: two sources in a scope have no established order.
	scopes := map[string]bool{}
	for _, s := range d.Sources {
		if !validEffortSourceDeclaration(runtimeID, s) || scopes[s.Scope] {
			return false
		}
		scopes[s.Scope] = true
	}
	for _, scope := range required {
		if !scopes[scope] {
			return false
		}
	}
	return len(scopes) == len(required)
}

func validDeclarationID(v string) bool {
	return len(v) <= maxDeclarationIDBytes && validToken(v) && SafeIdentifier(v)
}

func validEffortKeys(keys []string, caseInsensitive bool) bool {
	if len(keys) > maxEffortKeys {
		return false
	}
	seen := map[string]bool{}
	for _, k := range keys {
		normalized := k
		if caseInsensitive {
			normalized = strings.ToUpper(k)
		}
		if !safeEffortKey(k) || seen[normalized] {
			return false
		}
		seen[normalized] = true
	}
	return true
}

func safeEffortKey(v string) bool {
	if v == "" || len(v) > maxEffortKeyBytes {
		return false
	}
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// Scope classes mirror the recognized PE-05a sources; the anchor must be the
// root class that owns the scope.
var effortScopeAnchor = map[string]string{
	"user": EffortAnchorUser, "project": EffortAnchorProject, "project-local": EffortAnchorProject,
	"managed": EffortAnchorManaged, "managed-drop-in": EffortAnchorManaged,
}

func validEffortSourceDeclaration(runtimeID string, s EffortSourceDeclaration) bool {
	if anchor, ok := effortScopeAnchor[s.Scope]; !ok || s.Anchor != anchor || !ValidEffortSourcePath(s.Path) {
		return false
	}
	if s.Directory || s.Scope == "managed-drop-in" {
		return runtimeID == "claude" && s.Directory && s.Scope == "managed-drop-in" && s.Format == "json"
	}
	base := path.Base(s.Path)
	if runtimeID == "codex" {
		return s.Format == "toml" && (base == "config.toml" || base == "managed_config.toml")
	}
	return runtimeID == "claude" && s.Format == "json" && (base == "settings.json" || base == "settings.local.json" || base == "managed-settings.json")
}

// ValidEffortSourcePath is the lexical form of a persisted source path: a
// non-empty, canonical, slash-separated path relative to its anchor. Lexical
// validity does not prove filesystem safety; resolution must still use the
// existing trusted boundary, which refuses symlinks and special files.
func ValidEffortSourcePath(p string) bool {
	if p == "" || len(p) > maxEffortPathBytes || path.Clean(p) != p || strings.HasPrefix(p, "/") {
		return false
	}
	for _, segment := range strings.Split(p, "/") {
		// Rejects ".", "..", empty segments and Windows-normalized trailing
		// dots/spaces; the character set excludes "\\", ":" (drives, schemes,
		// streams), control characters and non-ASCII normalization forms.
		if segment == "" || segment == "." || segment == ".." || strings.HasSuffix(segment, ".") || strings.HasPrefix(segment, " ") || strings.HasSuffix(segment, " ") {
			return false
		}
		for _, r := range segment {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' || r == ' ') {
				return false
			}
		}
	}
	return true
}

func projectV2(c ConfigurationV3) Configuration {
	runtimes := make([]Runtime, len(c.Runtimes))
	for i, r := range c.Runtimes {
		runtimes[i] = Runtime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs}
	}
	return Configuration{FormatVersion: FormatVersionV2, Revision: c.Revision, Runtimes: runtimes, ModelProfiles: c.ModelProfiles, Preferences: c.Preferences}
}

func probeFormatVersion(wire []byte) int {
	var probe struct {
		FormatVersion *int `json:"formatVersion"`
	}
	if len(wire) == 0 || len(wire) > MaxConfigurationBytes || !uniqueJSON(wire) || json.Unmarshal(wire, &probe) != nil || probe.FormatVersion == nil {
		return 0
	}
	return *probe.FormatVersion
}

func decodeConfigurationV3(wire []byte) (ConfigurationV3, error) {
	if len(wire) == 0 || len(wire) > MaxConfigurationBytes || !uniqueJSON(wire) || !closedConfigurationV3(wire) {
		return ConfigurationV3{}, ErrInvalidConfiguration
	}
	d := json.NewDecoder(bytes.NewReader(wire))
	d.DisallowUnknownFields()
	var c ConfigurationV3
	if d.Decode(&c) != nil || d.Decode(&struct{}{}) != io.EOF || ValidateV3(c) != nil {
		return ConfigurationV3{}, ErrInvalidConfiguration
	}
	return c, nil
}

// EncodeV3 is the canonical v3 writer used by local authoring (PM-1).
func EncodeV3(c ConfigurationV3) ([]byte, error) {
	n := canonicalV3(c)
	if err := ValidateV3(n); err != nil {
		return nil, err
	}
	wire, err := json.Marshal(n)
	if err != nil || len(wire)+1 > MaxConfigurationBytes {
		return nil, ErrInvalidConfiguration
	}
	return append(wire, '\n'), nil
}

// DigestV3 is the canonical v3 configuration identity (Specification 007 AXM-22
// clause PM-8): SHA-256 over the exact EncodeV3 bytes, so every encoded field,
// including Runtime-owned EffortDeclarations, participates and EncodeV3 is the
// only canonicalization. It hashes no source contents, host paths, executable
// bytes or Evidence (EffortSourceDigest, ModeDigest and Evidence digests stay
// separate) and is independent of the historical format-1/2 Digest. Retirement
// of an active v3 record binds priorConfigurationDigest to this value from
// BLD-001B.3c-2b; until then the existing tombstone contract is unchanged.
func DigestV3(c ConfigurationV3) (string, error) {
	if err := ValidateV3(c); err != nil {
		return "", err
	}
	wire, err := EncodeV3(c)
	if err != nil {
		return "", err
	}
	return hash(wire), nil
}

// canonicalV3 orders every set-like collection; declaration source order never
// carries precedence, which the adapter owns.
func canonicalV3(c ConfigurationV3) ConfigurationV3 {
	n := c
	n.Runtimes = make([]RuntimeV3, len(c.Runtimes))
	for i, r := range c.Runtimes {
		r.AllowlistedProfileIDs = sortedStrings(r.AllowlistedProfileIDs)
		declarations := make([]EffortDeclaration, len(r.EffortDeclarations))
		for j, d := range r.EffortDeclarations {
			declarations[j] = cloneDeclaration(d)
			declarations[j].JSONKeys = nonNil(sortedStrings(d.JSONKeys))
			declarations[j].EnvironmentKeys = nonNil(sortedStrings(d.EnvironmentKeys))
			sort.Slice(declarations[j].Sources, func(a, b int) bool { return declarations[j].Sources[a].Scope < declarations[j].Sources[b].Scope })
		}
		sort.Slice(declarations, func(a, b int) bool { return declarations[a].DeclarationID < declarations[b].DeclarationID })
		if len(declarations) == 0 {
			declarations = nil
		}
		r.EffortDeclarations = declarations
		n.Runtimes[i] = r
	}
	sort.Slice(n.Runtimes, func(a, b int) bool { return n.Runtimes[a].ID < n.Runtimes[b].ID })
	n.ModelProfiles = append([]ModelProfile(nil), c.ModelProfiles...)
	for i := range n.ModelProfiles {
		n.ModelProfiles[i].Capabilities = sortedStrings(n.ModelProfiles[i].Capabilities)
		n.ModelProfiles[i].Complexities = sortedStrings(n.ModelProfiles[i].Complexities)
	}
	sort.Slice(n.ModelProfiles, func(a, b int) bool { return n.ModelProfiles[a].ID < n.ModelProfiles[b].ID })
	n.Preferences = append([]Preference(nil), c.Preferences...)
	sort.Slice(n.Preferences, func(a, b int) bool {
		return n.Preferences[a].Role+"\x00"+n.Preferences[a].Complexity < n.Preferences[b].Role+"\x00"+n.Preferences[b].Complexity
	})
	if len(n.Preferences) == 0 {
		n.Preferences = nil
	}
	return n
}

func cloneDeclaration(d EffortDeclaration) EffortDeclaration {
	d.JSONKeys = append([]string(nil), d.JSONKeys...)
	d.EnvironmentKeys = append([]string(nil), d.EnvironmentKeys...)
	d.Sources = append([]EffortSourceDeclaration(nil), d.Sources...)
	return d
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// requiredObject is closedObject with every member present: v3 declaration
// fields have no implicit defaults.
func requiredObject(w []byte, keys []string, nested map[string]func([]byte) bool) bool {
	var obj map[string]json.RawMessage
	if !closedObject(w, keys, nested) || json.Unmarshal(w, &obj) != nil || len(obj) != len(keys) {
		return false
	}
	return true
}

func closedConfigurationV3(w []byte) bool {
	effort := func(v []byte) bool { return closedObject(v, []string{"mode", "value"}, nil) }
	profile := func(v []byte) bool {
		return closedObject(v, []string{"id", "runtimeId", "model", "capabilities", "complexities", "effort"}, map[string]func([]byte) bool{"effort": effort})
	}
	// An explicit empty or null declaration list would be a second spelling of
	// "no declaration"; only omission is canonical.
	declarations := func(v []byte) bool {
		var items []json.RawMessage
		return json.Unmarshal(v, &items) == nil && len(items) > 0 && closedArray(v, closedEffortDeclaration)
	}
	runtime := func(v []byte) bool {
		return closedObject(v, []string{"id", "adapter", "enabled", "allowlistedProfileIds", "effortDeclarations"}, map[string]func([]byte) bool{"effortDeclarations": declarations})
	}
	preference := func(v []byte) bool { return closedObject(v, []string{"role", "complexity", "modelProfileId"}, nil) }
	return closedObject(w, []string{"formatVersion", "revision", "runtimes", "modelProfiles", "preferences"}, map[string]func([]byte) bool{"runtimes": func(v []byte) bool { return closedArray(v, runtime) }, "modelProfiles": func(v []byte) bool { return closedArray(v, profile) }, "preferences": func(v []byte) bool { return closedArray(v, preference) }})
}

// closedEffortDeclaration is the strict wire form of one declaration: every
// member present, no unknown, duplicate or case-alias member (BI-2a item 2).
func closedEffortDeclaration(v []byte) bool {
	list := func(v []byte) bool { return closedArray(v, func([]byte) bool { return true }) }
	source := func(v []byte) bool {
		return requiredObject(v, []string{"scope", "anchor", "path", "format", "directory"}, nil)
	}
	return requiredObject(v, []string{"declarationId", "declarationVersion", "runtimeVersion", "executableDigest", "invocationMode", "precedenceEstablished", "jsonKeys", "environmentKeys", "sources"}, map[string]func([]byte) bool{
		"jsonKeys": list, "environmentKeys": list, "sources": func(v []byte) bool { return closedArray(v, source) },
	})
}
