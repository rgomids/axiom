// Package capabilityproof keeps only the read-only codec of retired, historical
// capability Evidence records so local state inventory can classify them
// (ADR-0032). Nothing here admits, selects or dispatches.
package capabilityproof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/rgomids/axiom/internal/portableconfig"
	"github.com/rgomids/axiom/internal/workflowdefinition"
)

const (
	Read                   = "read"
	RepositoryWrite        = "repository-write"
	Process                = "process"
	Noninteractive         = "noninteractive"
	AxiomCoordination      = "axiom-coordination"
	ModelInvocation        = "model-invocation"
	ReadOnlyVariant        = "read-only"
	RepositoryWriteVariant = "repository-write"
)

type State string

const (
	Proven      State = "proven"
	Unsupported State = "unsupported"
	Unknown     State = "unknown"
)

type Binding struct {
	RuntimeID        string `json:"runtimeId"`
	ExecutableDigest string `json:"executableDigest"`
	Version          string `json:"version"`
	ModeDigest       string `json:"modeDigest"`
	Variant          string `json:"variant"`
	AdapterVersion   string `json:"adapterVersion"`
	AxiomDigest      string `json:"axiomDigest"`
}

// ModelBinding contains only the frozen, redacted resolvedProfile elements.
type ModelBinding struct {
	ProfileID             string `json:"profileId"`
	ProfileDigest         string `json:"profileDigest"`
	Model                 string `json:"model"`
	EffortMode            string `json:"effortMode"`
	EffortToken           string `json:"effortToken"`
	ConfigurationRevision uint64 `json:"configurationRevision"`
	EffortSourceDigest    string `json:"effortSourceDigest"`
}
type ProbeFacts struct {
	AuthenticationProven bool `json:"authenticationProven"`
	BehavioralProbe      bool `json:"behavioralProbe"`
	Verified             bool `json:"verified"`
	InterfaceAbsent      bool `json:"interfaceAbsent"`
	Inconclusive         bool `json:"inconclusive"`
}
type Result struct {
	State State      `json:"state"`
	Facts ProbeFacts `json:"facts"`
}
type Confinement struct {
	ControlsVerified       bool     `json:"controlsVerified"`
	CanaryIdentityMatched  bool     `json:"canaryIdentityMatched"`
	PositiveWritesVerified bool     `json:"positiveWritesVerified"`
	EveryNegativeDenied    bool     `json:"everyNegativeDenied"`
	OSSandboxObserved      bool     `json:"osSandboxObserved"`
	Level                  string   `json:"level"`
	Scope                  []string `json:"scope"`
	NotTested              []string `json:"notTested"`
}
type Evidence struct {
	SchemaVersion     int               `json:"schemaVersion"`
	Binding           Binding           `json:"binding"`
	Model             *ModelBinding     `json:"model,omitempty"`
	Capabilities      map[string]Result `json:"capabilities"`
	Confinement       Confinement       `json:"confinement"`
	ObservedAt        time.Time         `json:"observedAt"`
	ProbePacketDigest string            `json:"probePacketDigest"`
	InvalidatedReason string            `json:"invalidatedReason,omitempty"`
}

var ErrInvalidEvidence = errors.New("invalid capability evidence")
var token = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,127}$`)
var digest = regexp.MustCompile(`^[a-f0-9]{64}$`)

func safe(s string) bool {
	return !portableconfig.ContainsSecretBearingValue(s) && !strings.HasPrefix(s, "github_pat_") && !strings.HasPrefix(s, "gho_") && !strings.HasPrefix(s, "ghs_") && !strings.HasPrefix(s, "sk-") && !strings.HasPrefix(s, "ghp_") && !strings.HasPrefix(s, "AKIA") && token.MatchString(s) && !strings.Contains(strings.ToLower(s), "secret") && !strings.Contains(strings.ToLower(s), "token=")
}
func ValidCapability(c string) bool {
	return c == Read || c == RepositoryWrite || c == Process || c == Noninteractive || c == AxiomCoordination || c == ModelInvocation || strings.HasPrefix(c, "reasoning-effort-") && safe(strings.TrimPrefix(c, "reasoning-effort-"))
}
func ValidBinding(b Binding) bool {
	return safe(b.RuntimeID) && safe(b.Version) && safe(b.AdapterVersion) && digest.MatchString(b.ExecutableDigest) && digest.MatchString(b.ModeDigest) && digest.MatchString(b.AxiomDigest) && (b.Variant == ReadOnlyVariant || b.Variant == RepositoryWriteVariant)
}
func ValidEvidence(e Evidence) bool {
	if e.SchemaVersion != 1 || !ValidBinding(e.Binding) || e.ObservedAt.IsZero() || !digest.MatchString(e.ProbePacketDigest) || len(e.Capabilities) == 0 || len(e.Capabilities) > 64 {
		return false
	}
	if e.InvalidatedReason != "" && e.InvalidatedReason != "explicit" && e.InvalidatedReason != "profile-disabled" && e.InvalidatedReason != "profile-removed" {
		return false
	}
	if e.Model != nil {
		m := e.Model
		if !safe(m.ProfileID) || !safe(m.Model) || !safe(m.EffortToken) || !digest.MatchString(m.ProfileDigest) || !digest.MatchString(m.EffortSourceDigest) || m.ConfigurationRevision == 0 || (m.EffortMode != "explicit" && m.EffortMode != "runtime-default") || (m.EffortMode == "runtime-default" && m.EffortToken != "default") {
			return false
		}
	}
	for c, r := range e.Capabilities {
		if c == RepositoryWrite && r.State == Proven && e.Binding.Variant == ReadOnlyVariant {
			return false
		}
		if !ValidCapability(c) || (r.State != Proven && r.State != Unsupported && r.State != Unknown) {
			return false
		}
		if (c == ModelInvocation || strings.HasPrefix(c, "reasoning-effort-")) && e.Model == nil {
			return false
		}
		if strings.HasPrefix(c, "reasoning-effort-") && r.State == Proven && strings.TrimPrefix(c, "reasoning-effort-") != e.Model.EffortToken {
			return false
		}
		f := r.Facts
		if r.State == Proven && (!f.BehavioralProbe || !f.AuthenticationProven || !f.Verified || f.InterfaceAbsent || f.Inconclusive) {
			return false
		}
		if r.State == Unsupported && !f.InterfaceAbsent && (!f.BehavioralProbe || !f.AuthenticationProven || !f.Verified || f.Inconclusive) {
			return false
		}
	}
	if e.Confinement.Level != "unknown" && e.Confinement.Level != "advisory" && e.Confinement.Level != "os-enforced" {
		return false
	}
	if e.Confinement.Level == "os-enforced" && (!e.Confinement.ControlsVerified || !e.Confinement.CanaryIdentityMatched || !e.Confinement.PositiveWritesVerified || !e.Confinement.EveryNegativeDenied || !e.Confinement.OSSandboxObserved || len(e.Confinement.Scope) == 0) {
		return false
	}
	if len(e.Confinement.Scope) > 4 || len(e.Confinement.NotTested) > 5 {
		return false
	}
	// Fixed categories avoid retaining free-form paths, account data or output.
	seenScope := map[string]bool{}
	for _, s := range e.Confinement.Scope {
		if seenScope[s] {
			return false
		}
		seenScope[s] = true
		if s != "sentinel" && s != "workspace" && s != "inbox" && s != "loopback" {
			return false
		}
	}
	if e.Confinement.Level == "os-enforced" && (!seenScope["sentinel"] || e.Binding.Variant == ReadOnlyVariant && !seenScope["workspace"]) {
		return false
	}
	seenNotTested := map[string]bool{}
	for _, s := range e.Confinement.NotTested {
		if seenNotTested[s] {
			return false
		}
		seenNotTested[s] = true
		if s != "home" && s != "vendor-home" && s != "tmpdir" && s != "system-temp" && s != "external-network" {
			return false
		}
	}
	for _, required := range []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"} {
		if !seenNotTested[required] {
			return false
		}
	}
	// B0 has no extended-temp probe authority. All user/vendor/temp roots and
	// external networks must remain explicitly untested in its evidence.
	for _, root := range []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"} {
		if !slices.Contains(e.Confinement.NotTested, root) {
			return false
		}
	}
	if e.Confinement.Level == "os-enforced" && (!slices.Contains(e.Confinement.Scope, "sentinel") || e.Binding.Variant == ReadOnlyVariant && !slices.Contains(e.Confinement.Scope, "workspace")) {
		return false
	}
	return true
}
func EncodeEvidence(e Evidence) ([]byte, error) {
	if !ValidEvidence(e) {
		return nil, ErrInvalidEvidence
	}
	w, err := json.Marshal(e)
	if err != nil || len(w) > 65536 {
		return nil, ErrInvalidEvidence
	}
	return w, nil
}
func DecodeEvidence(w []byte) (Evidence, error) {
	var e Evidence
	if workflowdefinition.StrictJSONBounded(w, &e, 65536) != nil || !ValidEvidence(e) {
		return Evidence{}, ErrInvalidEvidence
	}
	return e, nil
}
func Digest(e Evidence) (string, error) {
	w, err := EncodeEvidence(e)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(w)
	return hex.EncodeToString(d[:]), nil
}
