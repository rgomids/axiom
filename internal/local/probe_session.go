//go:build unix

package local

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"golang.org/x/sys/unix"
)

// PreparedProbeEnvelopes accepts only the live T13 preparation handle. It never
// reopens abandoned envelopes for dispatch; their public opener is recovery-only.
type PreparedProbeEnvelopes struct{ Envelope *Envelope }

func (p PreparedProbeEnvelopes) Open(ctx context.Context, preparation capabilityproof.ProbePreparation) (capabilityproof.ProbeEnvelope, error) {
	e := p.Envelope
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := e.InspectReady(); err != nil {
		return nil, err
	}
	if preparation.ManifestDigest != e.digest || !reflect.DeepEqual(preparation.Manifest, e.Manifest()) || len(preparation.Invocation.Stdin) == 0 {
		return nil, capabilityproof.ErrStalePreview
	}
	directory, err := OpenOwnedDirectory(filepath.Join(e.path, "inbox"))
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir()
	if err != nil || len(entries) != 0 {
		return nil, ErrUnsafeEnvelope
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	launch := probeCanaryLaunch{ProbeID: e.manifest.ProbeID, ManifestDigest: e.digest, Nonce: e.manifest.Nonce, PacketDigest: digestEnvelope(preparation.Invocation.Stdin), Loopback: listener.Addr().(*net.TCPAddr).AddrPort(), Child: probeChild(e)}
	wire, err := json.Marshal(launch)
	if err == nil {
		err = directory.CreateExclusive("probe-run.json", wire, 0600)
	}
	if err != nil {
		listener.Close()
		return nil, err
	}
	inbox, err := NewAttemptInbox(launch.Child)
	if err == nil {
		err = inbox.Create(ctx, launch.Child)
	}
	if err != nil {
		listener.Close()
		return nil, err
	}
	if err = e.InspectReady(); err != nil {
		listener.Close()
		return nil, err
	}
	s := &probeSession{envelope: e, launch: launch, inbox: inbox, listener: listener, done: make(chan struct{})}
	go s.observeLoopback()
	return s, nil
}

type probeSession struct {
	envelope *Envelope
	launch   probeCanaryLaunch
	inbox    *AttemptInbox
	listener net.Listener
	done     chan struct{}
	mu       sync.Mutex
	reached  bool
	close    sync.Once
}

func (s *probeSession) observeLoopback() {
	defer close(s.done)
	for range 8 {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		if conn.SetDeadline(time.Now().Add(time.Second)) != nil {
			conn.Close()
			continue
		}
		nonce := make([]byte, 64)
		_, readErr := io.ReadFull(conn, nonce)
		if readErr == nil && string(nonce) == s.launch.Nonce {
			s.mu.Lock()
			s.reached = true
			s.mu.Unlock()
			_, _ = io.WriteString(conn, "ok")
		}
		conn.Close()
	}
}

func (s *probeSession) stopListener() {
	s.close.Do(func() { s.listener.Close(); <-s.done })
}

func (s *probeSession) Control(ctx context.Context, target string) (capabilityproof.Control, error) {
	if err := ctx.Err(); err != nil {
		return capabilityproof.Control{}, err
	}
	return capabilityproof.MeasureControl(s.envelope.Manifest(), target, probeControlEffects{s.envelope})
}

func (s *probeSession) Verify(ctx context.Context, result executiongraph.ProcessResult) (capabilityproof.ProbeVerification, error) {
	v := capabilityproof.ProbeVerification{ProbeID: s.launch.ProbeID, ManifestDigest: s.launch.ManifestDigest, Nonce: s.launch.Nonce, PacketDigest: s.launch.PacketDigest, Inconclusive: true}
	if err := ctx.Err(); err != nil {
		return v, err
	}
	if err := s.envelope.InspectReady(); err != nil {
		return v, err
	}
	var report capabilityproof.CanaryReport
	reports := 0
	err := s.inbox.Consume(ctx, s.launch.Child, true, func(c coordination.Candidate) (string, error) {
		r := c.Record
		if !coordination.ValidRecord(r) || c.Epoch != s.launch.Child.Epoch || r.Kind != coordination.Progress || r.ParentID != s.launch.Child.ParentID || r.ChildID != s.launch.Child.ChildID || r.AttemptID != s.launch.Child.AttemptID || r.GraphRevision != s.launch.Child.GraphRevision || len(r.Fields) != 4 {
			return "invalid_record", nil
		}
		fields := map[string]string{}
		for _, f := range r.Fields {
			fields[f.Name] = f.Value
		}
		var publication probeCanaryPublication
		if fields["reference"] != "probe-canary" || fields["phase"] != "measurement" || fields["digest"] != digestEnvelope([]byte(fields["summary"])) || strictEnvelopeJSON([]byte(fields["summary"]), &publication) != nil || publication.ManifestDigest != s.launch.ManifestDigest || publication.Nonce != s.launch.Nonce || publication.PacketDigest != s.launch.PacketDigest {
			return "invalid_record", nil
		}
		report = publication.Report
		reports++
		return "accepted", nil
	})
	if err != nil {
		return v, err
	}
	if result.ExitCode != 0 || result.TimedOut || result.Stopped || result.OutcomeUnknown || result.EffectAmbiguous || reports != 1 || report.ProbeID != s.launch.ProbeID || report.Published || report.Identity.UID != uint64(os.Getuid()) || report.Identity.GID != uint64(os.Getgid()) || len(report.Attempts) != len(s.envelope.manifest.NegativeTargets) {
		return v, nil
	}
	s.mu.Lock()
	reached := s.reached
	s.mu.Unlock()
	if report.Loopback == "reached" && !reached {
		return v, nil
	}
	for i, target := range s.envelope.manifest.NegativeTargets {
		attempt := report.Attempts[i]
		if attempt.Target != target {
			return v, nil
		}
		d, err := OpenOwnedDirectory(filepath.Join(s.envelope.path, target))
		if err != nil {
			return v, err
		}
		name := "canary-" + s.launch.Nonce
		if attempt.Created {
			wire, err := d.ReadFile(name, 1)
			d.Close()
			if err != nil || len(wire) != 0 || attempt.Errno != "" {
				return v, nil
			}
		} else {
			_, err := d.Lstat(name)
			d.Close()
			if !os.IsNotExist(err) || !slices.Contains([]string{"EACCES", "EPERM", "EROFS"}, attempt.Errno) {
				return v, nil
			}
		}
		v.Negatives = append(v.Negatives, capabilityproof.NegativeMeasurement{Target: target, Canary: attempt, Identity: report.Identity, CanaryRan: true})
	}
	inbox, err := OpenOwnedDirectory(filepath.Join(s.envelope.path, "inbox"))
	if err != nil {
		return v, err
	}
	wire, readErr := inbox.ReadFile("read-nonce", 128)
	inbox.Close()
	read := readErr == nil && string(wire) == s.launch.Nonce
	write := false
	if s.envelope.manifest.Variant == capabilityproof.RepositoryWriteVariant {
		workspace, err := OpenOwnedDirectory(filepath.Join(s.envelope.path, "workspace"))
		if err != nil {
			return v, err
		}
		wire, err = workspace.ReadFile("probe-work", 256)
		workspace.Close()
		write = err == nil && string(wire) == "created:"+s.launch.Nonce+";edited:"+s.launch.Nonce
	}
	v.Positive = capabilityproof.PositiveSteps{ReadNonce: read, WorkspaceCreated: write, WorkspaceEdited: write, CanaryExecuted: true, InboxPublished: true, ResultPublished: read, Noninteractive: true}
	v.ModelInvoked = read
	v.EffortVerified = read
	v.Inconclusive = !read || s.envelope.manifest.Variant == capabilityproof.RepositoryWriteVariant && !write
	return v, s.envelope.InspectReady()
}

func (s *probeSession) Cleanup(ctx context.Context) error {
	s.stopListener()
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := s.envelope.PreviewCleanup()
	if err != nil {
		return err
	}
	return p.Apply()
}

func (s *probeSession) Quarantine(ctx context.Context) error {
	s.stopListener()
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.envelope.Quarantine()
}

type probeControlEffects struct{ envelope *Envelope }

func (e probeControlEffects) Snapshot(target string) (capabilityproof.ObjectSnapshot, error) {
	if !slices.Contains(e.envelope.manifest.NegativeTargets, target) {
		return capabilityproof.ObjectSnapshot{}, capabilityproof.ErrCanaryContext
	}
	if err := e.envelope.InspectReady(); err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	d, err := OpenOwnedDirectory(filepath.Join(e.envelope.path, target))
	if err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	defer d.Close()
	file, err := d.root.Open(".")
	if err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	defer file.Close()
	if err := checkPrivateACL(file); err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	var filesystem unix.Statfs_t
	if err := unix.Fstatfs(int(file.Fd()), &filesystem); err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	info, err := file.Stat()
	if err != nil {
		return capabilityproof.ObjectSnapshot{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return capabilityproof.ObjectSnapshot{}, ErrUnsafeEnvelope
	}
	return capabilityproof.ObjectSnapshot{Device: uint64(stat.Dev), Inode: uint64(stat.Ino), Owner: uint64(stat.Uid), Mode: uint64(info.Mode()), ACLDigest: digestEnvelope([]byte("no-extended-acl")), ACLReadable: true, ReadOnly: probeFilesystemReadOnly(filesystem), SymlinkFree: true, ResolvedTarget: target}, d.StillAtPath()
}

func (e probeControlEffects) Identity() (uint64, uint64, error) {
	return uint64(os.Getuid()), uint64(os.Getgid()), nil
}

func (e probeControlEffects) CreateRemoveVerify(target string) (bool, bool, bool, error) {
	if !slices.Contains(e.envelope.manifest.NegativeTargets, target) {
		return false, false, false, capabilityproof.ErrCanaryContext
	}
	if err := e.envelope.InspectReady(); err != nil {
		return false, false, false, err
	}
	d, err := OpenOwnedDirectory(filepath.Join(e.envelope.path, target))
	if err != nil {
		return false, false, false, err
	}
	defer d.Close()
	name := "control-" + e.envelope.manifest.Nonce
	if err = d.CreateExclusive(name, nil, 0600); err != nil {
		return false, false, false, err
	}
	wire, err := d.ReadFile(name, 1)
	if err != nil || len(wire) != 0 {
		return true, false, false, ErrUnsafeEnvelope
	}
	if err = d.Remove(name); err != nil {
		return true, false, false, err
	}
	_, err = d.Lstat(name)
	if !errors.Is(err, os.ErrNotExist) {
		return true, true, false, ErrUnsafeEnvelope
	}
	return true, true, true, d.StillAtPath()
}

var _ capabilityproof.ProbeEnvelopes = PreparedProbeEnvelopes{}
var _ capabilityproof.ProbeEnvelope = (*probeSession)(nil)
var _ capabilityproof.CanaryEffects = (*probeCanaryEffects)(nil)
