//go:build unix

package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"syscall"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/coordination"
)

type probeCanaryLaunch struct {
	ProbeID        string                    `json:"probeId"`
	ManifestDigest string                    `json:"manifestDigest"`
	Nonce          string                    `json:"nonce"`
	PacketDigest   string                    `json:"packetDigest"`
	Loopback       netip.AddrPort            `json:"loopback"`
	Child          coordination.ChildContext `json:"child"`
}

// ProbeCanaryLoader resolves only the effective working directory supplied by
// the entrypoint. Probe identifiers never select arbitrary host paths.
type ProbeCanaryLoader struct{ WorkingDirectory string }

func (l ProbeCanaryLoader) Load(ctx context.Context, id, digest string) (capabilityproof.CanaryContext, capabilityproof.CanaryEffects, error) {
	if err := ctx.Err(); err != nil {
		return capabilityproof.CanaryContext{}, nil, err
	}
	cwd := l.WorkingDirectory
	if !filepath.IsAbs(cwd) || filepath.Clean(cwd) != cwd || !slices.Contains([]string{"workspace", "inbox"}, filepath.Base(cwd)) {
		return capabilityproof.CanaryContext{}, nil, capabilityproof.ErrCanaryContext
	}
	path := filepath.Dir(cwd)
	if filepath.Base(path) != id || filepath.Base(filepath.Dir(path)) != "capability-probes" {
		return capabilityproof.CanaryContext{}, nil, capabilityproof.ErrCanaryContext
	}
	policy := EnvelopePolicy{StateRoot: filepath.Dir(filepath.Dir(path))}
	directory, err := OpenOwnedDirectory(path)
	if err != nil {
		return capabilityproof.CanaryContext{}, nil, err
	}
	wire, err := directory.ReadFile("envelope.json", 16384)
	directory.Close()
	var m EnvelopeManifest
	if err != nil || strictEnvelopeJSON(wire, &m) != nil || digestEnvelope(wire) != digest || m.ProbeID != id {
		return capabilityproof.CanaryContext{}, nil, capabilityproof.ErrCanaryContext
	}
	policy.Variant = m.Variant
	e, err := openEnvelope(policy, id, digest)
	if err != nil {
		return capabilityproof.CanaryContext{}, nil, err
	}
	expected := "workspace"
	if m.Variant == capabilityproof.ReadOnlyVariant {
		expected = "inbox"
	}
	if filepath.Base(cwd) != expected {
		return capabilityproof.CanaryContext{}, nil, capabilityproof.ErrCanaryContext
	}
	e.prepared = true
	if err := e.InspectReady(); err != nil {
		return capabilityproof.CanaryContext{}, nil, err
	}
	inbox, err := OpenOwnedDirectory(filepath.Join(e.Path(), "inbox"))
	if err != nil {
		return capabilityproof.CanaryContext{}, nil, err
	}
	wire, err = inbox.ReadFile("probe-run.json", 16384)
	inbox.Close()
	var launch probeCanaryLaunch
	if err != nil || strictEnvelopeJSON(wire, &launch) != nil || !validProbeLaunch(e, launch) {
		return capabilityproof.CanaryContext{}, nil, capabilityproof.ErrCanaryContext
	}
	c := capabilityproof.CanaryContext{Manifest: e.Manifest(), ManifestDigest: digest, ProbeID: id, Nonce: m.Nonce, Loopback: launch.Loopback, ListenerOwned: true}
	effects := &probeCanaryEffects{envelope: e, launch: launch}
	if err := effects.Verify(ctx, c); err != nil {
		return capabilityproof.CanaryContext{}, nil, err
	}
	return c, effects, nil
}

func validProbeLaunch(e *Envelope, l probeCanaryLaunch) bool {
	return l.ProbeID == e.manifest.ProbeID && l.ManifestDigest == e.digest && l.Nonce == e.manifest.Nonce && hexDigestName.MatchString(l.PacketDigest+".json") && l.Loopback.IsValid() && l.Loopback.Addr().IsLoopback() && l.Loopback.Port() != 0 && coordination.ValidChildContext(l.Child) && l.Child == probeChild(e)
}

func probeChild(e *Envelope) coordination.ChildContext {
	uuid := func(label string) string {
		d := digestEnvelope([]byte(e.manifest.Nonce + label))
		return d[:8] + "-" + d[8:12] + "-" + d[12:16] + "-" + d[16:20] + "-" + d[20:32]
	}
	return coordination.ChildContext{Workspace: filepath.Join(e.path, "workspace"), Inbox: filepath.Join(e.path, "inbox", "attempt"), ParentID: uuid("parent"), ChildID: uuid("child"), AttemptID: uuid("attempt"), GraphRevision: 1, Epoch: 1, Running: true}
}

type probeCanaryEffects struct {
	envelope *Envelope
	launch   probeCanaryLaunch
}

func (e *probeCanaryEffects) Verify(ctx context.Context, c capabilityproof.CanaryContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := e.envelope.InspectReady(); err != nil {
		return err
	}
	if c.ProbeID != e.launch.ProbeID || c.ManifestDigest != e.launch.ManifestDigest || c.Nonce != e.launch.Nonce || c.Loopback != e.launch.Loopback || !c.ListenerOwned || !reflect.DeepEqual(c.Manifest, e.envelope.Manifest()) {
		return capabilityproof.ErrCanaryContext
	}
	inbox, err := OpenOwnedDirectory(filepath.Join(e.envelope.path, "inbox"))
	if err != nil {
		return err
	}
	defer inbox.Close()
	wire, err := inbox.ReadFile("probe-run.json", 16384)
	var current probeCanaryLaunch
	if err != nil || strictEnvelopeJSON(wire, &current) != nil || current != e.launch {
		return capabilityproof.ErrCanaryContext
	}
	return inbox.StillAtPath()
}

func (e *probeCanaryEffects) Identity(ctx context.Context) (capabilityproof.ProcessIdentity, error) {
	if err := ctx.Err(); err != nil {
		return capabilityproof.ProcessIdentity{}, err
	}
	return capabilityproof.ProcessIdentity{UID: uint64(os.Getuid()), GID: uint64(os.Getgid()), Process: fmt.Sprintf("pid-%d", os.Getpid())}, nil
}

func (e *probeCanaryEffects) CreateEmptyExclusive(ctx context.Context, target, name string) (bool, string, error) {
	if err := ctx.Err(); err != nil {
		return false, "", err
	}
	if !slices.Contains(e.envelope.manifest.NegativeTargets, target) || name != "canary-"+e.envelope.manifest.Nonce {
		return false, "", capabilityproof.ErrCanaryContext
	}
	if err := e.envelope.InspectReady(); err != nil {
		return false, "", err
	}
	directory, err := OpenOwnedDirectory(filepath.Join(e.envelope.path, target))
	if err != nil {
		return false, "", err
	}
	defer directory.Close()
	err = directory.CreateExclusive(name, nil, 0600)
	if err == nil {
		return true, "", directory.StillAtPath()
	}
	for _, item := range []struct {
		err  error
		name string
	}{{syscall.EACCES, "EACCES"}, {syscall.EPERM, "EPERM"}, {syscall.EROFS, "EROFS"}, {syscall.EEXIST, "EEXIST"}, {syscall.EIO, "EIO"}} {
		if errors.Is(err, item.err) {
			return false, item.name, nil
		}
	}
	return false, "OTHER", nil
}

func (e *probeCanaryEffects) ConnectOwnedLoopback(ctx context.Context, address netip.AddrPort) (string, error) {
	if address != e.launch.Loopback || !address.IsValid() || !address.Addr().IsLoopback() {
		return "", capabilityproof.ErrCanaryContext
	}
	dialer := net.Dialer{Timeout: time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address.String())
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			return "denied", nil
		}
		return "unknown", nil
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		return "unknown", nil
	}
	if _, err := io.WriteString(conn, e.launch.Nonce); err != nil {
		return "unknown", nil
	}
	ack := make([]byte, 2)
	if _, err := io.ReadFull(conn, ack); err != nil || string(ack) != "ok" {
		return "unknown", nil
	}
	return "reached", nil
}

func (e *probeCanaryEffects) Publish(ctx context.Context, report capabilityproof.CanaryReport) error {
	if report.ProbeID != e.launch.ProbeID || report.Published {
		return capabilityproof.ErrCanaryContext
	}
	wire, err := json.Marshal(probeCanaryPublication{ManifestDigest: e.launch.ManifestDigest, Nonce: e.launch.Nonce, PacketDigest: e.launch.PacketDigest, Report: report})
	if err != nil {
		return err
	}
	inbox, err := NewAttemptInbox(e.launch.Child)
	if err != nil {
		return err
	}
	publisher := coordination.InboxPublisher{Resolver: probeChildResolver{e.launch.Child}, Inbox: inbox}
	_, err = publisher.Publish(ctx, e.launch.Child.Workspace, coordination.Progress, []coordination.Field{{Name: "summary", Value: string(wire)}, {Name: "reference", Value: "probe-canary"}, {Name: "digest", Value: digestEnvelope(wire)}, {Name: "phase", Value: "measurement"}}, nil, coordination.Provenance{Product: "axiom", Version: "probe-v1", Revision: "canary-v1", SourceState: "local"})
	return err
}

type probeChildResolver struct{ child coordination.ChildContext }

func (r probeChildResolver) ResolveChild(ctx context.Context, workspace string) (coordination.ChildContext, error) {
	if ctx.Err() != nil || workspace != r.child.Workspace {
		return coordination.ChildContext{}, coordination.ErrChildContext
	}
	return r.child, nil
}

// ProbeAttemptID is packet correlation only; it grants no running-child authority.
func (e *Envelope) ProbeAttemptID() string { return probeChild(e).AttemptID }

type probeCanaryPublication struct {
	ManifestDigest string                       `json:"manifestDigest"`
	Nonce          string                       `json:"nonce"`
	PacketDigest   string                       `json:"packetDigest"`
	Report         capabilityproof.CanaryReport `json:"report"`
}
