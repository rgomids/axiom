package local

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeEnvelopeStrictJSON(t *testing.T) {
	for _, wire := range []string{`{"version":1,"version":1}`, `{"unknown":1}`, `{"identities":{".":"one",".":"two"}}`, `{} {}`} {
		var manifest EnvelopeManifest
		if strictEnvelopeJSON([]byte(wire), &manifest) == nil {
			t.Fatal("unsafe manifest accepted", wire)
		}
	}
}

func TestProbeEnvelopeInterruptedCleanupRecovery(t *testing.T) {
	for _, stage := range []string{"directory", "manifest", "root"} {
		t.Run(stage, func(t *testing.T) {
			e := envelopeFixture(t)
			preview, err := e.PreviewCleanup()
			if err != nil {
				t.Fatal(err)
			}
			if err := preview.ensureCleanupIntent(); err != nil {
				t.Fatal(err)
			}
			if stage == "directory" {
				if err := os.Remove(filepath.Join(e.path, "sentinel")); err != nil {
					t.Fatal(err)
				}
			} else {
				for _, name := range preview.names {
					if err := os.Remove(filepath.Join(e.path, name)); err != nil {
						t.Fatal(err)
					}
				}
				if stage == "root" {
					if err := os.Remove(e.path); err != nil {
						t.Fatal(err)
					}
				}
			}
			recovered, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
			if err != nil {
				t.Fatal(err)
			}
			next, err := recovered.PreviewCleanup()
			if err != nil {
				t.Fatal(err)
			}
			if err := next.Apply(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(e.path); !os.IsNotExist(err) {
				t.Fatal("root remained", err)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(e.path), cleanupIntentName(e.manifest.ProbeID))); !os.IsNotExist(err) {
				t.Fatal("intent remained", err)
			}
		})
	}
}

func TestProbeEnvelopeInterruptedPreparationRecovery(t *testing.T) {
	e := envelopeFixture(t)
	if err := os.RemoveAll(filepath.Join(e.path, "workspace")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(e.path, "inbox")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(e.path, "sentinel")); err != nil {
		t.Fatal(err)
	}
	pending := e.Manifest()
	pending.Phase = "preparing"
	pending.Identities = map[string]string{".": pending.Identities["."]}
	wire, err := json.Marshal(pending)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.path, "envelope.json"), wire, 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, digestEnvelope(wire))
	if err != nil {
		t.Fatal(err)
	}
	preview, err := recovered.PreviewCleanup()
	if err != nil {
		t.Fatal(err)
	}
	if err := preview.Apply(); err != nil {
		t.Fatal(err)
	}
}

func envelopeFixture(t *testing.T) *Envelope {
	return envelopeVariantFixture(t, "repository-write")
}

func envelopeVariantFixture(t *testing.T, variant string) *Envelope {
	t.Helper()
	base := os.Getenv("AXIOM_TEST_PROBE_STATE_ROOT")
	if base == "" {
		t.Skip("requires explicit non-repository, non-temporary private test state root")
	}
	state, err := os.MkdirTemp(base, "probe-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(state) })
	e, err := CreateEnvelope(EnvelopePolicy{Variant: variant, StateRoot: state}, "probe-one", EnvelopeLimits{Timeout: time.Minute, MaxFiles: 64, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestProbeEnvelopeVariantRoots(t *testing.T) {
	for _, variant := range []string{"read-only", "repository-write"} {
		t.Run(variant, func(t *testing.T) {
			e := envelopeVariantFixture(t, variant)
			m := e.Manifest()
			if !validEnvelopeManifest(m, m.ProbeID) {
				t.Fatal("invalid created manifest")
			}
			if variant == "read-only" {
				if len(m.AllowedRoots) != 1 || m.AllowedRoots[0] != "inbox" || len(m.NegativeTargets) != 2 {
					t.Fatal("read-only roots broadened")
				}
				m.AllowedRoots = append(m.AllowedRoots, "workspace")
			} else {
				m.AllowedRoots = []string{"inbox"}
			}
			if validEnvelopeManifest(m, m.ProbeID) {
				t.Fatal("changed variant roots accepted")
			}
		})
	}
}

func TestProbeEnvelopeLifecycle(t *testing.T) {
	e := envelopeFixture(t)
	if len(e.ManifestDigest()) != 64 {
		t.Fatal("missing manifest digest")
	}
	wire, err := os.ReadFile(filepath.Join(e.path, "envelope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if digestEnvelope(wire) != e.ManifestDigest() {
		t.Fatal("unstable digest")
	}
	if _, err := os.Stat(filepath.Join(e.path, "workspace", ".git", "HEAD")); err != nil {
		t.Fatal("scratch git layout", err)
	}
	if err := os.WriteFile(filepath.Join(e.path, "inbox", "result"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	preview, err := e.PreviewCleanup()
	if err != nil {
		t.Fatal(err)
	}
	// Preview alone preserves the quarantined envelope.
	if _, err := os.Stat(e.path); err != nil {
		t.Fatal(err)
	}

	if err := e.Quarantine(); err != nil {
		t.Fatal(err)
	}
	recovered, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.ManifestDigest() != e.ManifestDigest() {
		t.Fatal("quarantine identity changed")
	}
	if _, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, "wrong-digest"); err == nil {
		t.Fatal("unbound recovery accepted")
	}
	fresh, err := e.PreviewCleanup()
	if err != nil {
		t.Fatal(err)
	}
	if err := preview.Apply(); err == nil {
		t.Fatal("pre-quarantine cleanup preview accepted after persisted quarantine")
	}
	if err := fresh.Apply(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(e.path); !os.IsNotExist(err) {
		t.Fatal("cleanup not verified", err)
	}
}

func TestProbeEnvelopeRejectsUnsafePolicy(t *testing.T) {
	for _, root := range []string{"/tmp/probes", os.TempDir(), "relative", "/", filepath.Join(t.TempDir(), "state")} {
		t.Run(root, func(t *testing.T) {
			if _, err := CreateEnvelope(EnvelopePolicy{Variant: "repository-write", StateRoot: root}, "probe", EnvelopeLimits{Timeout: time.Minute, MaxFiles: 8, MaxBytes: 1024}); err == nil {
				t.Fatal("unsafe root accepted")
			}
		})
	}
	for _, id := range []string{"..", "../probe", "probe/x", "probe\\x", ""} {
		if probeName(id) {
			t.Fatal("unsafe id", id)
		}
	}
}

func TestProbeEnvelopePreservesExistingPath(t *testing.T) {
	e := envelopeFixture(t)
	if _, err := CreateEnvelope(e.policy, e.manifest.ProbeID, e.manifest.Limits); err == nil {
		t.Fatal("existing path accepted")
	}
	if err := e.verify(); err != nil {
		t.Fatal("existing envelope changed", err)
	}
}

func TestProbeEnvelopeRefusesReplacedDirectory(t *testing.T) {
	e := envelopeFixture(t)
	original := filepath.Join(e.path, "sentinel")
	if err := os.Rename(original, original+"-original"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(original, 0700); err != nil {
		t.Fatal(err)
	}
	if err := e.verify(); err == nil {
		t.Fatal("replaced directory accepted")
	}
	if _, err := os.Stat(original + "-original"); err != nil {
		t.Fatal("original lost", err)
	}
}

func TestProbeEnvelopeRefusesStaleCleanupAndSymlinks(t *testing.T) {
	e := envelopeFixture(t)
	preview, err := e.PreviewCleanup()
	if err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(external, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(e.path, "inbox", "escape")
	if err := os.Symlink(external, target); err != nil {
		t.Skip("symlink unavailable", err)
	}
	if _, err := e.PreviewCleanup(); err == nil {
		t.Fatal("symlink accepted")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = e.PreviewCleanup()
	if err != nil {
		t.Fatal(err)
	}
	if err := preview.Apply(); err == nil {
		t.Fatal("stale preview accepted")
	}
	wire, err := os.ReadFile(external)
	if err != nil || string(wire) != "preserve" {
		t.Fatal("outside target changed")
	}
}

func TestProbeEnvelopeRejectsSymlinkedStateComponent(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	if _, err := CreateEnvelope(EnvelopePolicy{Variant: "repository-write", StateRoot: filepath.Join(link, "state")}, "probe", EnvelopeLimits{Timeout: time.Minute, MaxFiles: 64, MaxBytes: 1 << 20}); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := os.Stat(filepath.Join(target, "state")); !os.IsNotExist(err) {
		t.Fatal("mutation through link")
	}
}
