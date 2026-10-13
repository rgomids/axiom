package local

import (
	"encoding/json"
	"github.com/rgomids/axiom/internal/workflowdefinition"
	"os"
	"path/filepath"
)

// The cleanup intent is outside the envelope so it survives the final
// manifest removal and can confirm an interrupted directory deletion.
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

func readEnvelopeIntent(policy EnvelopePolicy, id string) (*envelopeCleanupIntent, error) {
	parent, err := OpenOwnedDirectory(filepath.Join(policy.StateRoot, "capability-probes"))
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

func (p *EnvelopeCleanupPreview) ensureCleanupIntent() error {
	e := p.envelope
	intent, err := readEnvelopeIntent(e.policy, e.manifest.ProbeID)
	if err != nil {
		return err
	}
	if intent != nil {
		if intent.Digest != e.digest {
			return ErrUnsafeEnvelope
		}
		for name, identity := range p.identities {
			if intent.Identities[name] != identity {
				return ErrUnsafeEnvelope
			}
		}
		return nil
	}
	wire, err := json.Marshal(envelopeCleanupIntent{Version: 1, Manifest: e.Manifest(), Digest: e.digest, Identities: p.identities})
	if err != nil {
		return err
	}
	parent, err := OpenOwnedDirectory(filepath.Dir(e.path))
	if err != nil {
		return err
	}
	defer parent.Close()
	if err := parent.CreateExclusive(cleanupIntentName(e.manifest.ProbeID), wire, 0600); err != nil {
		return err
	}
	return parent.Sync()
}
