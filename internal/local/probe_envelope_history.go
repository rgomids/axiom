package local

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/rgomids/axiom/internal/workflowdefinition"
)

var ErrUnsafeEnvelope = errors.New("unsafe probe envelope; recovery required")

type EnvelopeLimits struct {
	Timeout  time.Duration `json:"timeout"`
	MaxFiles int           `json:"maxFiles"`
	MaxBytes int64         `json:"maxBytes"`
}

type EnvelopeManifest struct {
	Variant         string            `json:"variant"`
	NegativeTargets []string          `json:"negativeTargets"`
	Phase           string            `json:"phase"`
	Version         int               `json:"version"`
	ProbeID         string            `json:"probeId"`
	Nonce           string            `json:"nonce"`
	Revision        string            `json:"creationRevision"`
	CreatedAt       time.Time         `json:"createdAt"`
	AllowedRoots    []string          `json:"allowedRoots"`
	Sentinel        string            `json:"sentinel"`
	Limits          EnvelopeLimits    `json:"limits"`
	Identities      map[string]string `json:"identities"`
}

type envelopeCleanupIntent struct {
	Version    int               `json:"version"`
	Manifest   EnvelopeManifest  `json:"manifest"`
	Digest     string            `json:"digest"`
	Identities map[string]string `json:"identities"`
}

func cleanupIntentName(id string) string { return ".probe-cleanup-" + id + ".json" }

func strictEnvelopeJSON(wire []byte, target any) error {
	return workflowdefinition.StrictJSONBounded(wire, target, 16<<20)
}

func digestEnvelope(wire []byte) string {
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}

func probeName(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func readEnvelopeIntent(stateRoot, id string) (*envelopeCleanupIntent, error) {
	parent, err := OpenOwnedDirectory(filepath.Join(stateRoot, "capability-probes"))
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	if _, err := parent.Lstat(cleanupIntentName(id)); os.IsNotExist(err) {
		return nil, nil
	}
	wire, err := parent.ReadFile(cleanupIntentName(id), 16<<20)
	if err != nil {
		return nil, err
	}
	var intent envelopeCleanupIntent
	if strictEnvelopeJSON(wire, &intent) != nil || intent.Version != 1 || !validEnvelopeManifest(intent.Manifest, id) {
		return nil, ErrUnsafeEnvelope
	}
	encoded, err := json.Marshal(intent.Manifest)
	if err != nil || digestEnvelope(encoded) != intent.Digest {
		return nil, ErrUnsafeEnvelope
	}
	for name, identity := range intent.Identities {
		if !filepath.IsLocal(name) || filepath.Clean(name) != name || name == "." || identity == "" {
			return nil, ErrUnsafeEnvelope
		}
		first := name
		for filepath.Dir(first) != "." {
			first = filepath.Dir(first)
		}
		if first != "envelope.json" && first != "workspace" && first != "inbox" && first != "sentinel" && !protocolName(first) {
			return nil, ErrUnsafeEnvelope
		}
	}
	return &intent, nil
}

func validEnvelopeManifest(m EnvelopeManifest, id string) bool {
	if m.Version != 1 || m.ProbeID != id || len(m.Nonce) != 64 || !probeName(m.Nonce) || m.Revision != m.Nonce || m.CreatedAt.IsZero() || m.Limits.Timeout <= 0 || m.Limits.MaxFiles <= 0 || m.Limits.MaxFiles > 10000 || m.Limits.MaxBytes <= 0 || m.Limits.MaxBytes > 1<<30 || m.Sentinel != "sentinel" || (m.Phase != "ready" && m.Phase != "preparing") || len(m.Identities) < 1 || len(m.Identities) > 4 || m.Phase == "ready" && len(m.Identities) != 4 || m.Identities["."] == "" {
		return false
	}
	for name, identity := range m.Identities {
		if (name != "." && name != "workspace" && name != "inbox" && name != "sentinel") || identity == "" {
			return false
		}
	}
	switch m.Variant {
	case "read-only":
		return slices.Equal(m.AllowedRoots, []string{"inbox"}) && slices.Equal(m.NegativeTargets, []string{"sentinel", "workspace"})
	case "repository-write":
		return slices.Equal(m.AllowedRoots, []string{"workspace", "inbox"}) && slices.Equal(m.NegativeTargets, []string{"sentinel"})
	default:
		return false
	}
}
