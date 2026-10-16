package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/coordination"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

type probeFakeSettings struct {
	Runtime, Mode string
	Auth          bool
}

func probeFakeProcess() (bool, int) {
	if len(os.Args) > 1 && os.Args[1] == "__probe-canary" {
		handled, code := probeCanaryCommand(context.Background(), os.Args[1:], local.ProbeCanaryLoader{WorkingDirectory: probeWorkingDirectory()}, os.Stdout)
		return handled, code
	}
	if handled, code := extendedCanaryCommand(os.Args[1:], probeWorkingDirectory()); handled {
		return handled, code
	}
	name := filepath.Base(os.Args[0])
	if name != "codex" && name != "claude" {
		return false, 0
	}
	wire, err := os.ReadFile(filepath.Join(filepath.Dir(os.Args[0]), "fake-probe-runtime.json"))
	var cfg probeFakeSettings
	if err != nil || json.Unmarshal(wire, &cfg) != nil || cfg.Runtime != name {
		return false, 0
	}
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		version := "0.159.1"
		if name == "claude" {
			version = "2.1.296"
		}
		_, err = io.WriteString(os.Stdout, version+"\n")
		if err != nil {
			return true, 1
		}
		return true, 0
	}
	if len(os.Args) > 2 && (os.Args[1] == "login" || os.Args[1] == "auth") {
		if !cfg.Auth {
			return true, 1
		}
		status := "Logged in using ChatGPT"
		if name == "claude" {
			status = `{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"firstParty"}`
		}
		_, err = io.WriteString(os.Stdout, status+"\n")
		if err != nil {
			return true, 1
		}
		return true, 0
	}
	if cfg.Mode == "interrupted" || cfg.Mode == "timeout" {
		if err := os.WriteFile(filepath.Join(filepath.Dir(os.Args[0]), "fake-invocation-started"), nil, 0600); err != nil {
			return true, 9
		}
		time.Sleep(30 * time.Second)
		return true, 7
	}
	if cfg.Mode == "failure" || cfg.Mode == "refusal" || cfg.Mode == "usage-limit" {
		if cfg.Mode != "failure" {
			_, _ = io.WriteString(os.Stdout, cfg.Mode)
			return true, 0
		}
		_, _ = io.WriteString(os.Stdout, `{"modelInvoked":true,"effortVerified":true}`)
		return true, 7
	}
	if cfg.Mode == "claims" {
		_, _ = io.WriteString(os.Stdout, `{"modelInvoked":true,"effortVerified":true,"canaryExecuted":true,"capabilities":"proven"}`)
		return true, 0
	}
	var packet runtimeadapter.TaskPacket
	if json.NewDecoder(os.Stdin).Decode(&packet) != nil || len(packet.Inputs) != 1 {
		return true, 2
	}
	if fields := strings.Fields(packet.Instructions); len(fields) > 6 && fields[4] == "__probe-canary-extended" {
		args := fields[5:8]
		switch cfg.Mode {
		case "extended-skip":
			return true, 0
		case "extended-residue":
			_ = os.WriteFile(args[2], []byte("not empty"), 0600)
			return true, 0
		}
		cmd := exec.Command(packet.Coordination.AxiomPath, append([]string{"__probe-canary-extended"}, args...)...)
		cmd.Dir = probeWorkingDirectory()
		cmd.Env = os.Environ()
		if cmd.Run() != nil {
			return true, 14
		}
		return true, 0
	}
	root := filepath.Dir(probeWorkingDirectory())
	wire, err = os.ReadFile(filepath.Join(root, "workspace", "nonce"))
	if err != nil {
		return true, 3
	}
	if os.WriteFile(filepath.Join(root, "inbox", "read-nonce"), wire, 0600) != nil {
		return true, 4
	}
	manifestWire, err := os.ReadFile(filepath.Join(root, "envelope.json"))
	var manifest local.EnvelopeManifest
	sum := sha256.Sum256(manifestWire)
	if err != nil || json.Unmarshal(manifestWire, &manifest) != nil || hex.EncodeToString(sum[:]) != packet.Inputs[0].Digest || manifest.ProbeID != packet.AgentID {
		return true, 5
	}
	if manifest.Variant == capabilityproof.RepositoryWriteVariant {
		path := filepath.Join(root, "workspace", "probe-work")
		if os.WriteFile(path, append([]byte("created:"), wire...), 0600) != nil {
			return true, 6
		}
		if os.WriteFile(path, []byte("created:"+string(wire)+";edited:"+string(wire)), 0600) != nil {
			return true, 6
		}
	}
	if cfg.Mode == "forged-report" {
		return true, forgeProbePublication(root, manifest)
	}
	cmd := exec.Command(packet.Coordination.AxiomPath, "__probe-canary", packet.AgentID, packet.Inputs[0].Digest)
	cmd.Dir = probeWorkingDirectory()
	cmd.Env = os.Environ()
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if cmd.Run() != nil {
		return true, 8
	}
	return true, 0
}

func forgeProbePublication(root string, manifest local.EnvelopeManifest) int {
	wire, err := os.ReadFile(filepath.Join(root, "inbox", "probe-run.json"))
	var launch struct {
		ManifestDigest string                    `json:"manifestDigest"`
		Nonce          string                    `json:"nonce"`
		PacketDigest   string                    `json:"packetDigest"`
		Child          coordination.ChildContext `json:"child"`
	}
	if err != nil || json.Unmarshal(wire, &launch) != nil || launch.Nonce != manifest.Nonce {
		return 10
	}
	report := capabilityproof.CanaryReport{ProbeID: manifest.ProbeID, Identity: capabilityproof.ProcessIdentity{UID: uint64(os.Getuid()), GID: uint64(os.Getgid()), Process: "fabricated-canary"}, Loopback: "denied"}
	for _, target := range manifest.NegativeTargets {
		report.Attempts = append(report.Attempts, capabilityproof.CanaryAttempt{Target: target, Errno: "EACCES"})
	}
	publication := struct {
		ManifestDigest string                       `json:"manifestDigest"`
		Nonce          string                       `json:"nonce"`
		PacketDigest   string                       `json:"packetDigest"`
		Report         capabilityproof.CanaryReport `json:"report"`
	}{launch.ManifestDigest, launch.Nonce, launch.PacketDigest, report}
	wire, err = json.Marshal(publication)
	if err != nil {
		return 11
	}
	digest := sha256.Sum256(wire)
	inbox, err := local.NewAttemptInbox(launch.Child)
	if err != nil {
		return 12
	}
	publisher := coordination.InboxPublisher{Resolver: forgedProbeResolver{launch.Child}, Inbox: inbox}
	if _, err := publisher.Publish(context.Background(), launch.Child.Workspace, coordination.Progress, []coordination.Field{{Name: "summary", Value: string(wire)}, {Name: "reference", Value: "probe-canary"}, {Name: "digest", Value: hex.EncodeToString(digest[:])}, {Name: "phase", Value: "measurement"}}, nil, coordination.Provenance{Product: "axiom", Version: "probe-v1", Revision: "canary-v1", SourceState: "local"}); err != nil {
		return 13
	}
	return 0
}

type forgedProbeResolver struct{ child coordination.ChildContext }

func (r forgedProbeResolver) ResolveChild(context.Context, string) (coordination.ChildContext, error) {
	return r.child, nil
}

type observedProbeEnvelopes struct {
	capabilityproof.ProbeEnvelopes
	verify func(context.Context, capabilityproof.ProbeEnvelope, executiongraph.ProcessResult) (capabilityproof.ProbeVerification, error)
}

func (e observedProbeEnvelopes) Open(ctx context.Context, p capabilityproof.ProbePreparation) (capabilityproof.ProbeEnvelope, error) {
	s, err := e.ProbeEnvelopes.Open(ctx, p)
	return observedProbeEnvelope{ProbeEnvelope: s, verify: e.verify}, err
}

type observedProbeEnvelope struct {
	capabilityproof.ProbeEnvelope
	verify func(context.Context, capabilityproof.ProbeEnvelope, executiongraph.ProcessResult) (capabilityproof.ProbeVerification, error)
}

func (e observedProbeEnvelope) Verify(ctx context.Context, r executiongraph.ProcessResult) (capabilityproof.ProbeVerification, error) {
	return e.verify(ctx, e.ProbeEnvelope, r)
}

func TestPreparedProductionProbeForgedReport(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, variant := range []string{capabilityproof.ReadOnlyVariant, capabilityproof.RepositoryWriteVariant} {
			t.Run(runtimeID+"/"+variant, func(t *testing.T) {
				_, s, target, e := newPreparedProbeFixture(t, runtimeID, variant, "forged-report")
				verified := false
				s.ProbeEnvelopes = observedProbeEnvelopes{ProbeEnvelopes: s.ProbeEnvelopes, verify: func(ctx context.Context, session capabilityproof.ProbeEnvelope, r executiongraph.ProcessResult) (capabilityproof.ProbeVerification, error) {
					if r.ExitCode != 0 || r.TimedOut || r.OutcomeUnknown {
						t.Fatalf("forger failed: %+v", r)
					}
					for _, target := range e.Manifest().NegativeTargets {
						if _, err := os.Lstat(filepath.Join(e.Path(), target, "canary-"+e.Manifest().Nonce)); !os.IsNotExist(err) {
							t.Fatal("forger touched negative target", err)
						}
					}
					v, err := session.Verify(ctx, r)
					if err != nil || len(v.Negatives) != len(e.Manifest().NegativeTargets) || !v.Positive.ReadNonce || !v.Positive.InboxPublished {
						t.Fatalf("forged report did not reach verification: %+v %v", v, err)
					}
					if !v.Inconclusive || v.Positive.CanaryExecuted || v.ModelInvoked || v.EffortVerified {
						t.Fatalf("fabricated origin promoted: %+v", v)
					}
					for _, n := range v.Negatives {
						if n.CanaryRan || n.Canary.Errno != "EACCES" {
							t.Fatalf("fabricated denial promoted: %+v", n)
						}
					}
					verified = true
					return v, err
				}}
				p, err := s.PreviewProbe(context.Background(), target)
				if err != nil {
					t.Fatal(err)
				}
				out, err := s.Probe(context.Background(), p, probeAuthority(p))
				if err != nil || !verified || !out.Inconclusive || out.EvidenceRef == "" || out.Evidence.Confinement.Level != "unknown" {
					t.Fatalf("forged outcome: %+v %v", out, err)
				}
				for cap, result := range out.Evidence.Capabilities {
					if result.State != capabilityproof.Unknown {
						t.Fatalf("forger promoted %s: %+v", cap, result)
					}
				}
			})
		}
	}
}

func newPreparedProbeFixture(t *testing.T, runtimeID, variant, mode string) (*compositionFixture, capabilityproof.Service, capabilityproof.Target, *local.Envelope) {
	t.Helper()
	parent := os.Getenv("AXIOM_TEST_PROBE_STATE_ROOT")
	if parent == "" {
		t.Skip("requires private synthetic probe state outside repositories and temporary roots")
	}
	base, err := os.MkdirTemp(parent, "production-probe-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(base) })
	f := &compositionFixture{base: base, state: filepath.Join(base, "state"), exe: filepath.Join(base, "bin", runtimeID), axiom: filepath.Join(base, "bin", "axiom"), version: "0.159.1", consented: true}
	f.roots = runtimeadapter.EffortSourceRoots{User: filepath.Join(base, "home"), Project: filepath.Join(base, "repository"), Managed: filepath.Join(base, "managed")}
	for _, path := range []string{filepath.Dir(f.exe), f.roots.User, f.roots.Project, f.roots.Managed} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{f.exe, f.axiom} {
		src, err := os.Open(executable)
		if err != nil {
			t.Fatal(err)
		}
		dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if err != nil {
			src.Close()
			t.Fatal(err)
		}
		_, copyErr := io.Copy(dst, src)
		err = errors.Join(copyErr, src.Close(), dst.Close())
		if err != nil {
			t.Fatal(err)
		}
	}
	settings, _ := json.Marshal(probeFakeSettings{Runtime: runtimeID, Mode: mode, Auth: true})
	if err := os.WriteFile(filepath.Join(filepath.Dir(f.exe), "fake-probe-runtime.json"), settings, 0600); err != nil {
		t.Fatal(err)
	}
	declaration := runtimeprofiletest.CodexDeclaration()
	f.cfg = runtimeprofiletest.CodexOnly()
	profile := runtimeprofiletest.CodexHighProfileID
	if runtimeID == "claude" {
		f.version = "2.1.296"
		declaration = runtimeprofiletest.ClaudeDeclaration()
		f.cfg = runtimeprofiletest.ClaudeOnly()
		profile = runtimeprofiletest.ClaudeLowProfileID
	}
	declaration.ExecutableDigest, declaration.RuntimeVersion = f.identity(t, f.exe), f.version
	f.cfg = runtimeprofiletest.WithDeclaration(f.cfg, runtimeID, declaration)
	f.registered = map[[2]string]string{{runtimeID, f.identity(t, f.exe)}: f.version}
	f.stage = runtimeadapter.StageContext{Workspace: filepath.Join(base, "unused-work"), Inbox: filepath.Join(base, "unused-inbox"), ParentEnvironment: []string{"HOME=" + f.roots.User, "PATH=/usr/bin:/bin"}}
	f.profiles, err = local.NewRuntimeProfileStore(f.state)
	if err != nil {
		t.Fatal(err)
	}
	f.evidence, err = local.NewCapabilityStore(f.state)
	if err != nil {
		t.Fatal(err)
	}
	f.publish(t, func(*runtimeprofile.ConfigurationV3) {})
	timeout := 15 * time.Second
	if mode == "timeout" {
		timeout = 5 * time.Second
	}
	e, err := prepareCapabilityEnvelope(context.Background(), probePreparationAuthority{Actor: "synthetic-operator", Local: true, ProbeID: "synthetic-probe", Policy: local.EnvelopePolicy{StateRoot: f.state, Variant: variant, RepositoryRoots: []string{f.roots.Project}, VendorHomes: []string{filepath.Join(f.roots.User, ".codex"), filepath.Join(f.roots.User, ".claude")}}, Limits: local.EnvelopeLimits{Timeout: timeout, MaxFiles: 64, MaxBytes: 1 << 20}})
	if err != nil {
		t.Fatal(err)
	}
	d := f.dependencies()
	d.lookPath = func(name string) (string, error) {
		if name != runtimeID {
			return "", os.ErrNotExist
		}
		return f.exe, nil
	}
	d.envelope = e
	d.effortRoots = func(context.Context, capabilityproof.Target, []string) (runtimeadapter.EffortSourceRoots, error) {
		return f.roots, nil
	}
	s, err := composeCapabilityService(d)
	if err != nil {
		t.Fatal(err)
	}
	target := capabilityproof.Target{RuntimeID: runtimeID, ProfileID: profile, Variant: variant, Process: true}
	return f, s, target, e
}

func probeAuthority(p capabilityproof.ProbePreview) capabilityproof.ProbeAuthority {
	return capabilityproof.ProbeAuthority{Actor: "synthetic-operator", Local: true, External: true, Preview: p}
}

func probeTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String() + info.ModTime().String()
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			value += target
		} else if !d.IsDir() {
			wire, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(wire)
			value += hex.EncodeToString(sum[:])
		}
		result[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPreparedProductionProbeVariants(t *testing.T) {
	for _, runtimeID := range []string{"codex", "claude"} {
		for _, variant := range []string{capabilityproof.ReadOnlyVariant, capabilityproof.RepositoryWriteVariant} {
			t.Run(runtimeID+"/"+variant, func(t *testing.T) {
				f, s, target, e := newPreparedProbeFixture(t, runtimeID, variant, "success")
				s.ProbeEnvelopes = observedProbeEnvelopes{ProbeEnvelopes: s.ProbeEnvelopes, verify: func(ctx context.Context, session capabilityproof.ProbeEnvelope, r executiongraph.ProcessResult) (capabilityproof.ProbeVerification, error) {
					v, err := session.Verify(ctx, r)
					if err != nil || r.ExitCode != 0 || !v.Positive.ReadNonce || !v.Positive.InboxPublished || len(v.Negatives) != len(e.Manifest().NegativeTargets) {
						t.Fatalf("positive fake effects missing: %+v %+v %v", r, v, err)
					}
					if variant == capabilityproof.RepositoryWriteVariant && (!v.Positive.WorkspaceCreated || !v.Positive.WorkspaceEdited) {
						t.Fatal("positive workspace effects missing")
					}
					for _, n := range v.Negatives {
						if !n.Canary.Created || n.Canary.Errno != "" || n.CanaryRan {
							t.Fatalf("unexpected canary observation: %+v", n)
						}
					}
					return v, err
				}}
				before := probeTree(t, f.base)
				p, err := s.PreviewProbe(context.Background(), target)
				if err != nil {
					t.Fatal(err)
				}
				if after := probeTree(t, f.base); !reflect.DeepEqual(before, after) {
					t.Fatal("preview changed filesystem")
				}
				if p.Preparation.ManifestDigest != e.ManifestDigest() || p.Preparation.Invocation.CWD != filepath.Join(e.Path(), map[bool]string{true: "inbox", false: "workspace"}[variant == capabilityproof.ReadOnlyVariant]) {
					t.Fatal("wrong envelope invocation")
				}
				var packet runtimeadapter.TaskPacket
				if err := json.Unmarshal(p.Preparation.Invocation.Stdin, &packet); err != nil {
					t.Fatal(err)
				}
				if packet.Coordination.AttemptID != e.ProbeAttemptID() || packet.Coordination.AxiomPath != f.axiom || packet.Coordination.AxiomDigest != p.Preparation.Current.Binding.AxiomDigest || packet.Coordination.Inbox != filepath.Join(e.Path(), "inbox", "attempt") || len(packet.Inputs) != 1 || packet.Inputs[0].Digest != e.ManifestDigest() || !reflect.DeepEqual(packet.AllowedPaths, e.Manifest().AllowedRoots) || len(packet.AllowedEffects) < 3 {
					t.Fatal("packet does not bind the prepared envelope and pinned executable")
				}
				if variant == capabilityproof.ReadOnlyVariant && strings.Contains(packet.Instructions, "probe-work") {
					t.Fatal("read-only packet requires workspace write")
				}
				outside := map[string]map[string]string{}
				for _, root := range []string{f.roots.Project, f.roots.User, f.roots.Managed, filepath.Dir(f.exe)} {
					outside[root] = probeTree(t, root)
				}
				out, err := s.Probe(context.Background(), p, probeAuthority(p))
				if err != nil || !out.Inconclusive {
					t.Fatalf("outcome %+v error %v", out, err)
				}
				for cap, result := range out.Evidence.Capabilities {
					if result.State != capabilityproof.Unknown {
						t.Fatalf("%s promoted without independent attestation: %+v", cap, out.Evidence)
					}
				}
				if out.Evidence.Confinement.Level != "unknown" {
					t.Fatalf("unattested confinement = %+v", out.Evidence.Confinement)
				}
				stored, err := f.evidence.Load(context.Background(), out.EvidenceRef)
				if err != nil {
					t.Fatal("canonical Evidence mismatch", err)
				}
				wire, err := capabilityproof.EncodeEvidence(stored)
				if err != nil {
					t.Fatal(err)
				}
				original, err := capabilityproof.EncodeEvidence(out.Evidence)
				id, digestErr := capabilityproof.Digest(stored)
				if err != nil || digestErr != nil || !bytes.Equal(wire, original) || id != out.EvidenceRef || !stored.ObservedAt.Equal(out.Evidence.ObservedAt) {
					t.Fatal("canonical Evidence bytes or content address mismatch", err, digestErr)
				}
				if strings.Contains(string(wire), f.base) || strings.Contains(string(wire), "modelInvoked") {
					t.Fatal("raw output or host paths persisted")
				}
				if _, err := os.Lstat(e.Path()); !os.IsNotExist(err) {
					t.Fatal("envelope not cleaned", err)
				}
				for root, before := range outside {
					if !reflect.DeepEqual(before, probeTree(t, root)) {
						t.Fatal("probe changed an out-of-envelope fixture root", root)
					}
				}
			})
		}
	}
}

func TestPreparedProductionProbeRefusals(t *testing.T) {
	cases := []string{"local", "external", "different-preview", "auth", "manifest", "symlink", "replacement", "profile", "environment", "executable", "axiom", "configuration-v1", "configuration-v2", "claims", "failure", "refusal", "usage-limit", "interrupted", "timeout", "cleanup"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			mode := "success"
			if name == "timeout" {
				mode = "timeout"
			}
			f, s, target, e := newPreparedProbeFixture(t, "codex", capabilityproof.RepositoryWriteVariant, mode)
			p, err := s.PreviewProbe(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			a := probeAuthority(p)
			expected := error(nil)
			ctx := context.Background()
			switch name {
			case "local":
				a.Local = false
				expected = capabilityproof.ErrAuthority
			case "external":
				a.External = false
				expected = capabilityproof.ErrAuthority
			case "different-preview":
				a.Preview.Cost = "changed"
				expected = capabilityproof.ErrAuthority
			case "auth", "claims", "failure", "refusal", "usage-limit", "interrupted":
				cfg := probeFakeSettings{Runtime: "codex", Mode: name, Auth: name != "auth"}
				wire, _ := json.Marshal(cfg)
				if err := os.WriteFile(filepath.Join(filepath.Dir(f.exe), "fake-probe-runtime.json"), wire, 0600); err != nil {
					t.Fatal(err)
				}
				if name == "auth" {
					expected = capabilityproof.ErrAuthentication
				}
				if name == "interrupted" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, time.Minute)
					defer cancel()
					go func() {
						ticker := time.NewTicker(10 * time.Millisecond)
						defer ticker.Stop()
						for {
							select {
							case <-ctx.Done():
								return
							case <-ticker.C:
								if _, err := os.Stat(filepath.Join(filepath.Dir(f.exe), "fake-invocation-started")); err == nil {
									cancel()
									return
								}
							}
						}
					}()
				}
			case "manifest":
				path := filepath.Join(e.Path(), "envelope.json")
				wire, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(wire, ' '), 0600); err != nil {
					t.Fatal(err)
				}
				expected = capabilityproof.ErrStalePreview
			case "symlink", "replacement":
				path := filepath.Join(e.Path(), "sentinel")
				if err := os.Rename(path, path+"-old"); err != nil {
					t.Fatal(err)
				}
				if name == "symlink" {
					err = os.Symlink(path+"-old", path)
				} else {
					err = os.Mkdir(path, 0700)
				}
				if err != nil {
					t.Fatal(err)
				}
				expected = capabilityproof.ErrStalePreview
			case "profile":
				f.publish(t, func(c *runtimeprofile.ConfigurationV3) { c.ModelProfiles[0].Model = "changed-model" })
				expected = capabilityproof.ErrStalePreview
			case "environment":
				f.stage.ParentEnvironment = append(f.stage.ParentEnvironment, "LANG=C")
				expected = capabilityproof.ErrStalePreview
			case "executable", "axiom":
				path := f.exe
				if name == "axiom" {
					path = f.axiom
				}
				if err := os.WriteFile(path, []byte("replaced-executable"), 0700); err != nil {
					t.Fatal(err)
				}
				expected = capabilityproof.ErrStalePreview
			case "configuration-v1", "configuration-v2":
				path := filepath.Join(f.state, "runtime-profiles", "v1", "configuration.json")
				version := "1"
				if name == "configuration-v2" {
					version = "2"
				}
				if err := os.WriteFile(path, []byte(`{"formatVersion":`+version+`}`), 0600); err != nil {
					t.Fatal(err)
				}
				expected = capabilityproof.ErrStalePreview
			case "cleanup":
				path := filepath.Join(e.Path(), "workspace", "foreign")
				if err := os.Symlink(f.roots.Project, path); err != nil {
					t.Fatal(err)
				}
			}
			before := probeTree(t, f.base)
			out, err := s.Probe(ctx, p, a)
			if name == "timeout" {
				if _, startedErr := os.Stat(filepath.Join(filepath.Dir(f.exe), "fake-invocation-started")); startedErr != nil {
					t.Fatal("timeout did not reach the fake invocation", startedErr, err)
				}
			}
			if name == "interrupted" {
				if _, startedErr := os.Stat(filepath.Join(filepath.Dir(f.exe), "fake-invocation-started")); startedErr != nil || !errors.Is(ctx.Err(), context.Canceled) {
					t.Fatalf("did not cancel the running fake invocation: %v, %v, %v", startedErr, ctx.Err(), err)
				}
			}
			if expected != nil {
				if !errors.Is(err, expected) || out.EvidenceRef != "" {
					t.Fatalf("expected %v got %+v %v", expected, out, err)
				}
				if after := probeTree(t, f.base); !reflect.DeepEqual(before, after) {
					t.Fatal("refused apply changed filesystem")
				}
				return
			}
			if name == "cleanup" {
				if err == nil || out.EvidenceRef != "" {
					t.Fatalf("unsafe cleanup succeeded: %+v %v", out, err)
				}
				if _, err := os.Lstat(e.Path()); err != nil {
					t.Fatal("quarantine removed", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !out.Inconclusive {
				t.Fatalf("%s promoted: %+v", name, out)
			}
			for cap, result := range out.Evidence.Capabilities {
				if result.State != capabilityproof.Unknown {
					t.Fatalf("%s: %s promoted", name, cap)
				}
			}
			_, statErr := os.Lstat(e.Path())
			if (name == "interrupted" || name == "timeout") && statErr != nil {
				t.Fatal("interruption lost quarantine", statErr)
			}
			if (name == "interrupted" || name == "timeout") && e.InspectReady() == nil {
				t.Fatal("quarantine retained dispatch authority")
			}
			if name != "interrupted" && name != "timeout" && !os.IsNotExist(statErr) {
				t.Fatal("completed fake residue", statErr)
			}
		})
	}
}

func TestPrepareEnvelopeRequiresIndependentLocalAuthority(t *testing.T) {
	for _, a := range []probePreparationAuthority{{Local: false, Actor: "operator"}, {Local: true}, {Local: true, Actor: "bad\nactor"}} {
		if e, err := prepareCapabilityEnvelope(context.Background(), a); e != nil || !errors.Is(err, capabilityproof.ErrAuthority) {
			t.Fatal(e, err)
		}
	}
}
