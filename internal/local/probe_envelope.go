package local

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
)

var ErrUnsafeEnvelope = errors.New("unsafe probe envelope; recovery required")

// EnvelopePolicy names all repository and vendor roots known to the caller.
// Home and temporary roots are additionally obtained from the local environment.
// Policy never grants permission to override a prohibited root.
type EnvelopePolicy = capabilityproof.EnvelopePolicy
type EnvelopeLimits = capabilityproof.EnvelopeLimits
type EnvelopeManifest = capabilityproof.EnvelopeManifest

// Envelope contains host paths only locally; its manifest contains relative names.
type Envelope struct {
	policy   EnvelopePolicy
	path     string
	manifest EnvelopeManifest
	digest   string
	prepared bool
	mu       sync.Mutex
	claimed  bool
}

func within(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func checkEnvelopePolicy(policy EnvelopePolicy) error {
	root := policy.StateRoot
	if !filepath.IsAbs(root) || root != filepath.Clean(root) || root == filepath.VolumeName(root)+string(filepath.Separator) {
		return ErrUnsafeEnvelope
	}
	prohibited := append([]string{os.TempDir(), os.Getenv("TMPDIR")}, policy.RepositoryRoots...)
	if filepath.IsAbs("/tmp") {
		prohibited = append(prohibited, "/tmp")
	}
	prohibited = append(prohibited, policy.VendorHomes...)
	prohibited = append(prohibited, os.Getenv("CODEX_HOME"), os.Getenv("CLAUDE_CONFIG_DIR"))
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	// A state-root descendant of HOME is the sole permitted HOME exception.
	if root == home {
		return ErrUnsafeEnvelope
	}
	prohibited = append(prohibited, filepath.Join(home, ".codex"), filepath.Join(home, ".claude"))
	for _, denied := range prohibited {
		if denied == "" {
			continue
		}
		if canonical, err := filepath.EvalSymlinks(denied); err == nil && within(root, canonical) {
			return ErrUnsafeEnvelope
		}
		if !filepath.IsAbs(denied) || within(root, filepath.Clean(denied)) {
			return ErrUnsafeEnvelope
		}
	}
	// Reject every link component, including system-owned aliases.
	current := filepath.VolumeName(root) + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(root, current), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafeEnvelope
		}
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil || !os.IsNotExist(err) {
			return ErrUnsafeEnvelope
		}
	}
	return nil
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

// CreateEnvelope never overwrites a pre-existing probe and leaves interrupted
// preparation quarantined. An unused ready preparation can be reopened only by
// its exact manifest digest; a running probe requires its unique live claim.
func CreateEnvelope(policy EnvelopePolicy, id string, limits EnvelopeLimits) (*Envelope, error) {
	if _, err := InspectEnvelopePreparation(policy, id, limits); err != nil {
		return nil, err
	}
	return prepareEnvelope(policy, id, limits)
}

// InspectEnvelopePreparation validates a zero-write preparation and binds the
// existing state-root ancestors. Any existing probe identity is refused.
func InspectEnvelopePreparation(policy EnvelopePolicy, id string, limits EnvelopeLimits) (string, error) {
	if (policy.Variant != capabilityproof.ReadOnlyVariant && policy.Variant != capabilityproof.RepositoryWriteVariant) || !probeName(id) || limits.Timeout <= 0 || limits.MaxFiles <= 0 || limits.MaxBytes <= 0 || limits.MaxBytes > 1<<30 || limits.MaxFiles > 10000 {
		return "", ErrUnsafeEnvelope
	}
	if err := checkEnvelopePolicy(policy); err != nil {
		return "", err
	}
	if _, err := os.Lstat(filepath.Join(policy.StateRoot, "capability-probes", id)); !os.IsNotExist(err) {
		return "", ErrUnsafeEnvelope
	}
	identities := map[string]string{}
	for path := policy.StateRoot; ; path = filepath.Dir(path) {
		identity, err := DirectoryIdentity(path)
		if err != nil {
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				return "", ErrUnsafeEnvelope
			}
			identity = "absent"
		}
		identities[path] = identity
		if filepath.Dir(path) == path {
			break
		}
	}
	wire, err := json.Marshal(identities)
	if err != nil {
		return "", err
	}
	return digestEnvelope(wire), nil
}

// prepareEnvelope accepts an already policy-validated state root.
func prepareEnvelope(policy EnvelopePolicy, id string, limits EnvelopeLimits) (*Envelope, error) {
	state, err := CreateOwnedDirectory(policy.StateRoot)
	if err != nil {
		return nil, err
	}
	defer state.Close()
	if err := state.Mkdir("capability-probes"); err != nil && !os.IsExist(err) {
		return nil, err
	}
	parent, err := OpenOwnedDirectory(filepath.Join(policy.StateRoot, "capability-probes"))
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	locks, err := lockRoots(true, state.root, parent.root)
	if err != nil {
		return nil, err
	}
	defer closeFiles(locks)
	if _, err := parent.Lstat(cleanupIntentName(id)); !os.IsNotExist(err) {
		return nil, ErrUnsafeEnvelope
	}
	if err := parent.Mkdir(id); err != nil {
		return nil, err
	}
	path := filepath.Join(policy.StateRoot, "capability-probes", id)
	directory, err := OpenOwnedDirectory(path)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	envelopeLock, err := lockRoots(true, directory.root)
	if err != nil {
		return nil, err
	}
	defer closeFiles(envelopeLock)
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	allowed := []string{"workspace", "inbox"}
	negative := []string{"sentinel"}
	if policy.Variant == capabilityproof.ReadOnlyVariant {
		allowed = []string{"inbox"}
		negative = []string{"sentinel", "workspace"}
	}
	manifest := EnvelopeManifest{Variant: policy.Variant, NegativeTargets: negative, Phase: "preparing", Version: 1, ProbeID: id, Nonce: hex.EncodeToString(nonce), Revision: hex.EncodeToString(nonce), CreatedAt: time.Now().UTC(), AllowedRoots: allowed, Sentinel: "sentinel", Limits: limits, Identities: map[string]string{}}
	identity, err := envelopeIdentity(path)
	if err != nil {
		return nil, err
	}
	manifest.Identities["."] = identity
	wire, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if err := directory.CreateExclusive("envelope.json", wire, 0600); err != nil {
		return nil, err
	}
	if err := directory.Sync(); err != nil {
		return nil, err
	}
	checkpoint := func() error {
		next, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		if err := publishFile(context.Background(), directory.root, "envelope.json", wire, next, false, publicationHooks{}); err != nil {
			return err
		}
		wire = next
		return directory.StillAtPath()
	}
	for _, name := range []string{"workspace", "inbox", "sentinel"} {
		if err := directory.Mkdir(name); err != nil {
			return nil, err
		}
		identity, err := envelopeIdentity(filepath.Join(path, name))
		if err != nil {
			return nil, err
		}
		manifest.Identities[name] = identity
		if err := checkpoint(); err != nil {
			return nil, err
		}
	}
	workspace, err := OpenOwnedDirectory(filepath.Join(path, "workspace"))
	if err != nil {
		return nil, err
	}
	if err = workspace.Mkdir(".git"); err != nil {
		workspace.Close()
		return nil, err
	}
	if err := workspace.CreateExclusive("nonce", []byte(manifest.Nonce), 0600); err != nil {
		workspace.Close()
		return nil, err
	}
	workspace.Close()
	git, err := OpenOwnedDirectory(filepath.Join(path, "workspace", ".git"))
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"objects", "refs"} {
		if err = git.Mkdir(name); err != nil {
			git.Close()
			return nil, err
		}
	}
	if err = git.CreateExclusive("HEAD", []byte("ref: refs/heads/probe\n"), 0600); err != nil {
		git.Close()
		return nil, err
	}
	if err = git.CreateExclusive("config", []byte("[core]\n\trepositoryformatversion = 0\n\tbare = false\n"), 0600); err != nil {
		git.Close()
		return nil, err
	}
	git.Close()
	manifest.Phase = "ready"
	if err := checkpoint(); err != nil {
		return nil, err
	}
	e := &Envelope{policy: policy, path: path, manifest: manifest, digest: digestEnvelope(wire), prepared: true}
	if err := e.verifyObjects(); err != nil {
		return nil, err
	}
	return e, nil
}

func digestEnvelope(wire []byte) string {
	sum := sha256.Sum256(wire)
	return hex.EncodeToString(sum[:])
}
func (e *Envelope) Path() string           { return e.path }
func (e *Envelope) ManifestDigest() string { return e.digest }

// InspectReady observes a live prepared envelope without creating objects or
// accepting cleanup recovery state as a runnable envelope.
func (e *Envelope) InspectReady() error {
	if e == nil {
		return ErrUnsafeEnvelope
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.prepared || e.manifest.Phase != "ready" {
		return ErrUnsafeEnvelope
	}
	if err := e.verify(); err != nil {
		return err
	}
	intent, err := readEnvelopeIntent(e.policy, e.manifest.ProbeID)
	if err != nil || intent != nil {
		return ErrUnsafeEnvelope
	}
	if _, err := os.Lstat(filepath.Join(e.path, probeQuarantineName)); !os.IsNotExist(err) {
		return ErrUnsafeEnvelope
	}
	if !e.claimed {
		d, err := OpenOwnedDirectory(e.path)
		if err != nil {
			return err
		}
		defer d.Close()
		if _, err := d.Lstat(probeClaimName); !os.IsNotExist(err) {
			return ErrUnsafeEnvelope
		}
	}
	return nil
}

const probeClaimName = ".axiom-recovery-probe-claimed"
const probeQuarantineName = ".axiom-recovery-probe-quarantined"

// OpenPreparedEnvelope reads an unused ready preparation by its exact manifest
// digest. A claimed, quarantined, interrupted or cleanup envelope never reopens
// for execution; OpenQuarantinedEnvelope remains cleanup-only.
func OpenPreparedEnvelope(policy EnvelopePolicy, id, digest string) (*Envelope, error) {
	e, err := openEnvelope(policy, id, digest)
	if err != nil || e.manifest.Phase != "ready" {
		return nil, ErrUnsafeEnvelope
	}
	e.prepared = true
	if err := e.InspectReady(); err != nil {
		return nil, err
	}
	d, err := OpenOwnedDirectory(filepath.Join(e.path, "inbox"))
	if err != nil {
		return nil, err
	}
	defer d.Close()
	entries, err := d.ReadDir()
	if err != nil || len(entries) != 0 {
		return nil, ErrUnsafeEnvelope
	}
	return e, nil
}

func (e *Envelope) claim() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.prepared || e.claimed {
		return ErrUnsafeEnvelope
	}
	if err := e.verify(); err != nil {
		return err
	}
	d, err := OpenOwnedDirectory(e.path)
	if err != nil {
		return err
	}
	defer d.Close()
	info, err := d.root.Stat(".")
	if err != nil {
		return err
	}
	identity, err := envelopeObjectIdentity(info)
	if err != nil || identity != e.manifest.Identities["."] {
		return ErrUnsafeEnvelope
	}
	if err = d.CreateExclusive(probeClaimName, []byte(e.digest), 0600); err != nil {
		e.prepared = false
		return err
	}
	e.claimed = true
	if err := d.Sync(); err != nil {
		e.prepared = false
		return err
	}
	return nil
}

// Manifest returns detached metadata; callers cannot change cleanup identity.
func (e *Envelope) Manifest() EnvelopeManifest {
	m := e.manifest
	m.AllowedRoots = append([]string(nil), m.AllowedRoots...)
	m.NegativeTargets = append([]string(nil), m.NegativeTargets...)
	m.Identities = make(map[string]string, len(e.manifest.Identities))
	for name, identity := range e.manifest.Identities {
		m.Identities[name] = identity
	}
	return m
}

func (e *Envelope) verify() error {
	if !validEnvelopeManifest(e.manifest, e.manifest.ProbeID) || e.manifest.Variant != e.policy.Variant {
		return ErrUnsafeEnvelope
	}
	if err := checkEnvelopePolicy(e.policy); err != nil {
		return err
	}
	return e.verifyObjects()
}

func (e *Envelope) verifyObjects() error {
	intent, intentErr := readEnvelopeIntent(e.policy, e.manifest.ProbeID)
	if intentErr != nil || intent != nil && intent.Digest != e.digest {
		return ErrUnsafeEnvelope
	}
	if _, err := os.Lstat(e.path); os.IsNotExist(err) && intent != nil {
		return nil
	}
	directory, err := OpenOwnedDirectory(e.path)
	if err != nil {
		return err
	}
	defer directory.Close()
	wire, err := directory.ReadFile("envelope.json", 16384)
	if err != nil || digestEnvelope(wire) != e.digest {
		if intent == nil {
			return ErrUnsafeEnvelope
		}
		if _, statErr := directory.Lstat("envelope.json"); !os.IsNotExist(statErr) {
			return ErrUnsafeEnvelope
		}
	}
	for name, identity := range e.manifest.Identities {
		observed, err := envelopeIdentity(filepath.Join(e.path, name))
		if os.IsNotExist(err) && name != "." && intent != nil {
			continue
		}
		if err != nil || observed != identity {
			return ErrUnsafeEnvelope
		}
	}
	return directory.StillAtPath()
}

// OpenQuarantinedEnvelope reopens leftovers for explicit cleanup only. It does
// not authorize resuming a probe or promote interrupted evidence to proven.
func OpenQuarantinedEnvelope(policy EnvelopePolicy, id, digest string) (*Envelope, error) {
	return openEnvelope(policy, id, digest)
}

func openEnvelope(policy EnvelopePolicy, id, digest string) (*Envelope, error) {
	if !probeName(id) || digest == "" {
		return nil, ErrUnsafeEnvelope
	}
	if err := checkEnvelopePolicy(policy); err != nil {
		return nil, err
	}
	path := filepath.Join(policy.StateRoot, "capability-probes", id)
	directory, err := OpenOwnedDirectory(path)
	if err != nil {
		if _, statErr := os.Lstat(path); !os.IsNotExist(statErr) {
			return nil, err
		}
		intent, intentErr := readEnvelopeIntent(policy, id)
		if intentErr != nil || intent == nil || intent.Digest != digest {
			return nil, ErrUnsafeEnvelope
		}
		e := &Envelope{policy: policy, path: path, manifest: intent.Manifest, digest: digest}
		return e, e.verify()
	}
	defer directory.Close()
	wire, err := directory.ReadFile("envelope.json", 16384)
	var manifest EnvelopeManifest
	if err != nil {
		intent, intentErr := readEnvelopeIntent(policy, id)
		if intentErr != nil || intent == nil || intent.Digest != digest {
			return nil, ErrUnsafeEnvelope
		}
		if _, statErr := directory.Lstat("envelope.json"); !os.IsNotExist(statErr) {
			return nil, ErrUnsafeEnvelope
		}
		manifest = intent.Manifest
	} else {
		if digestEnvelope(wire) != digest || strictEnvelopeJSON(wire, &manifest) != nil {
			return nil, ErrUnsafeEnvelope
		}
	}
	if !validEnvelopeManifest(manifest, id) || manifest.Variant != policy.Variant {
		return nil, ErrUnsafeEnvelope
	}
	e := &Envelope{policy: policy, path: path, manifest: manifest, digest: digest}
	return e, e.verify()
}

// Quarantine revokes the live preparation handle and preserves every object.
// Leftovers are quarantined by default, including crash-before-marker cases.
func (e *Envelope) Quarantine() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.prepared = false
	if err := e.verify(); err != nil {
		return err
	}
	intent, err := readEnvelopeIntent(e.policy, e.manifest.ProbeID)
	if err != nil || intent != nil {
		return err
	}
	d, err := OpenOwnedDirectory(e.path)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.CreateExclusive(probeQuarantineName, []byte(e.digest), 0600); err != nil && !os.IsExist(err) {
		return err
	}
	return d.Sync()
}

type EnvelopeCleanupPreview struct {
	envelope   *Envelope
	identities map[string]string
	names      []string
}

// PreviewCleanup snapshots each exact object. Links, foreign metadata and
// unexpected top-level names fail closed; cleanup never uses RemoveAll.
func (e *Envelope) PreviewCleanup() (*EnvelopeCleanupPreview, error) {
	if err := e.verify(); err != nil {
		return nil, err
	}
	return e.previewCleanupObjects()
}

func (e *Envelope) previewCleanupObjects() (*EnvelopeCleanupPreview, error) {
	if err := e.verifyObjects(); err != nil {
		return nil, err
	}
	p := &EnvelopeCleanupPreview{envelope: e, identities: map[string]string{}}
	if _, err := os.Lstat(e.path); os.IsNotExist(err) {
		return p, nil
	}
	var totalBytes int64
	var walk func(string) error
	walk = func(relative string) error {
		path := filepath.Join(e.path, relative)
		directory, err := OpenOwnedDirectory(path)
		if err != nil {
			return err
		}
		defer directory.Close()
		entries, err := directory.ReadDir()
		if err != nil {
			return err
		}
		for _, entry := range entries {
			name := filepath.Join(relative, entry.Name())
			if relative == "." && entry.Name() != "workspace" && entry.Name() != "inbox" && entry.Name() != "sentinel" && entry.Name() != "envelope.json" && !protocolName(entry.Name()) {
				return ErrUnsafeEnvelope
			}
			if relative == "." && (entry.Name() == "workspace" || entry.Name() == "inbox" || entry.Name() == "sentinel") && e.manifest.Identities[entry.Name()] == "" {
				return ErrUnsafeEnvelope
			}
			info, err := directory.Lstat(entry.Name())
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := walk(name); err != nil {
					return err
				}
			} else {
				if !info.Mode().IsRegular() {
					return ErrUnsafeEnvelope
				}
				if name != "envelope.json" && !protocolName(entry.Name()) {
					if info.Size() < 0 || info.Size() > e.manifest.Limits.MaxBytes-totalBytes {
						return ErrUnsafeEnvelope
					}
					totalBytes += info.Size()
				}
				file, err := directory.root.OpenFile(entry.Name(), os.O_RDONLY|noFollow, 0)
				if err != nil {
					return err
				}
				observed, statErr := file.Stat()
				safe := statErr == nil && observed.Mode().IsRegular() && os.SameFile(info, observed) && ownedByUser(observed) && !forbiddenPermissions(observed, 0077) && singleLink(file, observed) && checkPrivateACL(file) == nil
				file.Close()
				if !safe {
					return ErrUnsafeEnvelope
				}
			}
			identity, err := envelopeIdentity(filepath.Join(e.path, name))
			if err != nil {
				return err
			}
			p.identities[name] = identity
			p.names = append(p.names, name)
			if len(p.names) > e.manifest.Limits.MaxFiles+4 {
				return ErrUnsafeEnvelope
			}
		}
		return directory.StillAtPath()
	}
	if err := walk("."); err != nil {
		return nil, err
	}
	// Preserve the recovery manifest until the other entries are removed.
	for i, name := range p.names {
		if name == "envelope.json" {
			p.names = append(append(p.names[:i:i], p.names[i+1:]...), name)
			break
		}
	}
	if intent, err := readEnvelopeIntent(e.policy, e.manifest.ProbeID); err != nil {
		return nil, err
	} else if intent != nil {
		for name, identity := range p.identities {
			if intent.Identities[name] != identity {
				return nil, ErrUnsafeEnvelope
			}
		}
	}
	return p, nil
}

// Apply explicitly approves this preview only. Any replaced or added object
// invalidates it before removal; partial failure is reported as recovery required.
func (p *EnvelopeCleanupPreview) Apply() error {
	if p == nil || p.envelope == nil {
		return ErrUnsafeEnvelope
	}
	e := p.envelope
	state, err := OpenOwnedDirectory(e.policy.StateRoot)
	if err != nil {
		return err
	}
	defer state.Close()
	parent, err := OpenOwnedDirectory(filepath.Dir(e.path))
	if err != nil {
		return err
	}
	defer parent.Close()
	roots := []*os.Root{state.root, parent.root}
	if _, err := os.Lstat(e.path); err == nil {
		directory, err := OpenOwnedDirectory(e.path)
		if err != nil {
			return err
		}
		defer directory.Close()
		roots = append(roots, directory.root)
	} else if !os.IsNotExist(err) {
		return err
	}
	locks, err := lockRoots(true, roots...)
	if err != nil {
		return err
	}
	defer closeFiles(locks)
	current, err := p.envelope.PreviewCleanup()
	if err != nil {
		return err
	}
	return p.applyObjects(current)
}

func (p *EnvelopeCleanupPreview) applyObjects(current *EnvelopeCleanupPreview) error {
	if len(current.identities) != len(p.identities) {
		return ErrUnsafeEnvelope
	}
	for name, identity := range p.identities {
		if current.identities[name] != identity {
			return ErrUnsafeEnvelope
		}
	}
	if err := p.ensureCleanupIntent(); err != nil {
		return err
	}
	for _, name := range p.names {
		parent, err := OpenOwnedDirectory(filepath.Dir(filepath.Join(p.envelope.path, name)))
		if err != nil {
			return err
		}
		identity, err := envelopeIdentity(filepath.Join(p.envelope.path, name))
		if err == nil && identity == p.identities[name] {
			err = parent.Remove(filepath.Base(name))
		} else {
			err = ErrUnsafeEnvelope
		}
		parent.Close()
		if err != nil {
			return fmt.Errorf("%w: cleanup incomplete", ErrUnsafeEnvelope)
		}
	}
	parent, err := OpenOwnedDirectory(filepath.Dir(p.envelope.path))
	if err != nil {
		return err
	}
	defer parent.Close()
	identity, err := envelopeIdentity(p.envelope.path)
	if !os.IsNotExist(err) {
		if err != nil || identity != p.envelope.manifest.Identities["."] {
			return ErrUnsafeEnvelope
		}
		if err := parent.Remove(filepath.Base(p.envelope.path)); err != nil {
			return err
		}
	}
	if _, err := parent.Lstat(filepath.Base(p.envelope.path)); !os.IsNotExist(err) {
		return ErrUnsafeEnvelope
	}
	if err := parent.Remove(cleanupIntentName(p.envelope.manifest.ProbeID)); err != nil {
		return err
	}
	return parent.Sync()
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
	case capabilityproof.ReadOnlyVariant:
		return slices.Equal(m.AllowedRoots, []string{"inbox"}) && slices.Equal(m.NegativeTargets, []string{"sentinel", "workspace"})
	case capabilityproof.RepositoryWriteVariant:
		return slices.Equal(m.AllowedRoots, []string{"workspace", "inbox"}) && slices.Equal(m.NegativeTargets, []string{"sentinel"})
	default:
		return false
	}
}
