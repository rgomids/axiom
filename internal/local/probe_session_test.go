//go:build unix

package local

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
)

func openProbeSession(t *testing.T, e *Envelope) capabilityproof.ProbeEnvelope {
	t.Helper()
	s, err := (PreparedProbeEnvelopes{Envelope: e}).Open(context.Background(), capabilityproof.ProbePreparation{Manifest: e.Manifest(), ManifestDigest: e.ManifestDigest(), Invocation: executiongraph.Invocation{Stdin: []byte("synthetic-packet")}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Quarantine(context.Background()) })
	return s
}

func TestProbeCanaryLoaderRefusesDrift(t *testing.T) {
	for _, name := range []string{"id", "digest", "cwd", "variant-cwd", "manifest", "symlink", "identity", "nonce", "external-network", "child", "launch-symlink"} {
		t.Run(name, func(t *testing.T) {
			e := envelopeFixture(t)
			openProbeSession(t, e)
			cwd, id, digest := filepath.Join(e.Path(), "workspace"), e.manifest.ProbeID, e.digest
			launchPath := filepath.Join(e.Path(), "inbox", "probe-run.json")
			switch name {
			case "id":
				id = "other-probe"
			case "digest":
				digest = strings.Repeat("0", 64)
			case "cwd":
				cwd = e.Path()
			case "variant-cwd":
				cwd = filepath.Join(e.Path(), "inbox")
			case "manifest":
				if err := os.WriteFile(filepath.Join(e.Path(), "envelope.json"), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink", "identity":
				path := filepath.Join(e.Path(), "sentinel")
				if err := os.Rename(path, path+"-old"); err != nil {
					t.Fatal(err)
				}
				var err error
				if name == "symlink" {
					err = os.Symlink(path+"-old", path)
				} else {
					err = os.Mkdir(path, 0700)
				}
				if err != nil {
					t.Fatal(err)
				}
			case "launch-symlink":
				if err := os.Rename(launchPath, launchPath+"-old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(launchPath+"-old", launchPath); err != nil {
					t.Fatal(err)
				}
			default:
				wire, err := os.ReadFile(launchPath)
				var launch probeCanaryLaunch
				if err != nil || json.Unmarshal(wire, &launch) != nil {
					t.Fatal("launch unavailable", err)
				}
				switch name {
				case "nonce":
					launch.Nonce = strings.Repeat("0", 64)
				case "external-network":
					launch.Loopback = netip.AddrPortFrom(netip.MustParseAddr("192.0.2.1"), launch.Loopback.Port())
				case "child":
					launch.Child.Epoch++
				}
				wire, err = json.Marshal(launch)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(launchPath, wire, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := (ProbeCanaryLoader{WorkingDirectory: cwd}).Load(context.Background(), id, digest); err == nil {
				t.Fatal("unsafe canary context accepted")
			}
		})
	}
}

func TestProbeSessionRejectsPacketAndAttemptDrift(t *testing.T) {
	for _, name := range []string{"packet", "attempt", "unknown-denial"} {
		t.Run(name, func(t *testing.T) {
			e := envelopeFixture(t)
			s := openProbeSession(t, e)
			ctx := context.Background()
			c, effects, err := (ProbeCanaryLoader{WorkingDirectory: filepath.Join(e.Path(), "workspace")}).Load(ctx, e.manifest.ProbeID, e.digest)
			if err != nil {
				t.Fatal(err)
			}
			native := effects.(*probeCanaryEffects)
			if name == "packet" {
				native.launch.PacketDigest = strings.Repeat("0", 64)
			}
			if name == "attempt" {
				native.launch.Child.AttemptID = native.launch.Child.ParentID
			}
			identity, err := effects.Identity(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := effects.Publish(ctx, capabilityproof.CanaryReport{ProbeID: c.ProbeID, Identity: identity, Attempts: []capabilityproof.CanaryAttempt{{Target: "sentinel", Errno: "EIO"}}, Loopback: "unknown"}); err != nil && name != "attempt" {
				t.Fatal(err)
			}
			v, err := s.Verify(ctx, executiongraph.ProcessResult{})
			if err != nil || !v.Inconclusive || v.ModelInvoked || v.Positive.CanaryExecuted {
				t.Fatalf("unverified publication promoted: %+v %v", v, err)
			}
		})
	}
}

func TestProbeSessionCannotResumeRecoveryHandle(t *testing.T) {
	e := envelopeFixture(t)
	recovered, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.InspectReady(); err == nil {
		t.Fatal("recovery handle acquired running authority")
	}
	if _, err := (PreparedProbeEnvelopes{Envelope: recovered}).Open(context.Background(), capabilityproof.ProbePreparation{Manifest: e.Manifest(), ManifestDigest: e.digest, Invocation: executiongraph.Invocation{Stdin: []byte("packet")}}); err == nil {
		t.Fatal("recovery handle resumed")
	}
	s := openProbeSession(t, e)
	if _, err := (PreparedProbeEnvelopes{Envelope: e}).Open(context.Background(), capabilityproof.ProbePreparation{Manifest: e.Manifest(), ManifestDigest: e.digest, Invocation: executiongraph.Invocation{Stdin: []byte("packet")}}); err == nil {
		t.Fatal("used envelope resumed")
	}
	if err := s.Quarantine(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, err := recovered.PreviewCleanup()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Apply(); err != nil {
		t.Fatal(err)
	}
}

func TestProbePreparedAbandonmentRevokesHandle(t *testing.T) {
	e := envelopeFixture(t)
	if err := e.Quarantine(); err != nil {
		t.Fatal(err)
	}
	if err := e.InspectReady(); err == nil {
		t.Fatal("abandoned preparation retained dispatch authority")
	}
	recovered, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
	if err != nil {
		t.Fatal(err)
	}
	p, err := recovered.PreviewCleanup()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Apply(); err != nil {
		t.Fatal(err)
	}
}

func TestProbeNativeControlObservesUnchangedObject(t *testing.T) {
	e := envelopeFixture(t)
	s := openProbeSession(t, e)
	pre, err := s.Control(context.Background(), "sentinel")
	if err != nil || !pre.Created || !pre.Removed || !pre.RemovalVerified || !pre.Snapshot.ACLReadable || pre.Snapshot.ACLDigest == "" || !pre.Snapshot.SymlinkFree || pre.Snapshot.ReadOnly {
		t.Fatalf("control unavailable: %+v %v", pre, err)
	}
	post, err := s.Control(context.Background(), "sentinel")
	if err != nil || pre != post {
		t.Fatalf("control drift: %+v %+v %v", pre, post, err)
	}
	if _, err := s.Control(context.Background(), "../outside"); err == nil {
		t.Fatal("control accepted path outside manifest")
	}
}

func TestProbeOpenFailureQuarantinesPartialPreparation(t *testing.T) {
	for _, stage := range []string{"launch", "partial-inbox", "complete-inbox", "final-inspection"} {
		t.Run(stage, func(t *testing.T) {
			e := envelopeFixture(t)
			preparation := capabilityproof.ProbePreparation{Manifest: e.Manifest(), ManifestDigest: e.digest, Invocation: executiongraph.Invocation{Stdin: []byte("synthetic-packet")}}
			injected := error(syscall.ENOSPC)
			if stage == "final-inspection" {
				injected = ErrUnsafeEnvelope
			}
			create := func(inbox *AttemptInbox, ctx context.Context, child coordination.ChildContext) error {
				if _, err := os.Stat(filepath.Join(e.path, "inbox", "probe-run.json")); err != nil {
					t.Fatal("launch not persisted before fault", err)
				}
				switch stage {
				case "partial-inbox":
					if err := os.Mkdir(child.Inbox, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(child.Inbox, "binding.json"), []byte("{"), 0600); err != nil {
						t.Fatal(err)
					}
				case "complete-inbox", "final-inspection":
					if err := inbox.Create(ctx, child); err != nil {
						t.Fatal(err)
					}
					if stage == "final-inspection" {
						preview, err := e.PreviewCleanup()
						if err != nil {
							t.Fatal(err)
						}
						return preview.ensureCleanupIntent()
					}
				}
				return injected
			}
			s, err := (PreparedProbeEnvelopes{Envelope: e}).open(context.Background(), preparation, create)
			if s != nil || !errors.Is(err, injected) || e.prepared || e.InspectReady() == nil {
				t.Fatalf("partial preparation retained authority: %v %v prepared=%v", s, err, e.prepared)
			}
			launchWire, err := os.ReadFile(filepath.Join(e.path, "inbox", "probe-run.json"))
			var launch probeCanaryLaunch
			if err != nil || json.Unmarshal(launchWire, &launch) != nil {
				t.Fatal("launch lost", err)
			}
			conn, err := net.DialTimeout("tcp4", launch.Loopback.String(), time.Second)
			if err == nil {
				conn.Close()
				t.Fatal("failed open leaked listener")
			}
			if _, err := (PreparedProbeEnvelopes{Envelope: e}).Open(context.Background(), preparation); err == nil {
				t.Fatal("failed open reused preparation")
			}
			if stage != "launch" {
				if _, err := os.Stat(filepath.Join(launch.Child.Inbox, "binding.json")); err != nil {
					t.Fatal("partial objects removed implicitly", err)
				}
			}
			inventory, err := InspectStateInventory(context.Background(), e.policy.StateRoot)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, entry := range inventory.Entries {
				if entry.Relative == filepath.Join("capability-probes", e.manifest.ProbeID) {
					found = entry.Kind == InventoryRecovery
				}
			}
			if !found {
				t.Fatal("failed preparation missing from recovery inventory")
			}
			recovered, err := OpenQuarantinedEnvelope(e.policy, e.manifest.ProbeID, e.digest)
			if err != nil || recovered.InspectReady() == nil {
				t.Fatal("recovery acquired authority", err)
			}
			preview, err := recovered.PreviewCleanup()
			if err != nil {
				t.Fatal(err)
			}
			if err := preview.Apply(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(e.path); !os.IsNotExist(err) {
				t.Fatal("exact recovery cleanup failed", err)
			}
		})
	}
}
