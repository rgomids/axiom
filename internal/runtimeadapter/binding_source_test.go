//go:build unix

package runtimeadapter

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 BLD-001B.2b: the production BindingSource observes
// the complete Binding and ModelBinding from injected, deterministic sources
// and synthetic non-executed files, then selects Evidence through AXM-21. No
// test runs a Runtime, probes or invokes a model.

var bindingNow = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

var bindingVersions = map[string]string{"codex": "0.159.1", "claude": "2.1.296"}

type bindingScanner struct {
	records []capabilityproof.StoredEvidence
	scans   int
	hook    func(context.Context) error
}

func (s *bindingScanner) Scan(ctx context.Context, visit func(capabilityproof.StoredEvidence) error) error {
	s.scans++
	for _, r := range s.records {
		if s.hook != nil {
			if err := s.hook(ctx); err != nil {
				return err
			}
		}
		if err := visit(r); err != nil {
			return err
		}
	}
	return nil
}

type bindingFixture struct {
	runtime, base, exe, axiom string
	cfg                       runtimeprofile.ConfigurationV3
	roots                     EffortSourceRoots
	version                   string
	scanner                   *bindingScanner
	source                    BindingSource
}

func newBindingFixture(t *testing.T, runtime string) *bindingFixture {
	t.Helper()
	inspector := newEffortInspectorFixture(t, runtime)
	// Stage paths live under a neutral root: PE-05a refuses any stage argument
	// containing "effort", and test temp names embed the test name.
	neutral, err := os.MkdirTemp("", "axm22-stage-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(neutral) })
	if neutral, err = filepath.EvalSymlinks(neutral); err != nil {
		t.Fatal(err)
	}
	f := &bindingFixture{runtime: runtime, base: neutral, roots: inspector.req.Roots, version: bindingVersions[runtime], scanner: &bindingScanner{}}
	f.exe = filepath.Join(f.base, "bin", runtime)
	f.axiom = filepath.Join(f.base, "bin", "axiom")
	for _, path := range []string{f.exe, f.axiom} {
		writeFixtureFile(t, path, "synthetic "+filepath.Base(path)+"; never execute\n")
		if err := os.Chmod(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	f.cfg = *inspector.req.Configuration.V3
	d := &f.cfg.Runtimes[0].EffortDeclarations[0]
	d.RuntimeVersion = f.version
	d.ExecutableDigest = f.identity(t, f.exe)
	f.cfg.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: runtimeprofile.EffortExplicit, Value: "high"}
	f.source = BindingSource{
		Configuration: func(context.Context) (runtimeprofile.VersionedConfiguration, error) {
			return decodeV3Fixture(t, f.cfg), nil
		},
		LookPath: func(name string) (string, error) {
			if name != runtime {
				return "", os.ErrNotExist
			}
			return f.exe, nil
		},
		Version: func(_ context.Context, id, digest string) (string, error) {
			if id != runtime || digest == "" {
				return "", errors.New("unobserved")
			}
			return f.version, nil
		},
		AxiomPath: f.axiom,
		Stage: func(context.Context, capabilityproof.Target) (StageContext, error) {
			return StageContext{Workspace: filepath.Join(f.base, "work"), Inbox: filepath.Join(f.base, "inbox"), ParentEnvironment: []string{"PATH=fixture"}}, nil
		},
		EffortRoots:     func(context.Context) (EffortSourceRoots, error) { return f.roots, nil },
		EffortConsented: true,
		Selector:        capabilityproof.Selector{Evidence: f.scanner, Now: func() time.Time { return bindingNow }},
	}
	return f
}

func (f *bindingFixture) identity(t *testing.T, path string) string {
	t.Helper()
	digest, err := ExecutableIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func (f *bindingFixture) target(variant string, process bool) capabilityproof.Target {
	return capabilityproof.Target{RuntimeID: f.runtime, ProfileID: "test-profile", Variant: variant, Process: process}
}

// expected computes the identity through independent canonical oracles: the
// adapter ModeDigest, and PE-05a CheckEffortSources over the dispatch-time
// BuildFrozenStageInvocation.
func (f *bindingFixture) expected(t *testing.T, target capabilityproof.Target) (capabilityproof.Binding, capabilityproof.ModelBinding) {
	t.Helper()
	profile := f.cfg.ModelProfiles[0]
	request := StageInvocationRequest{Profile: CommandProfile{RuntimeID: f.runtime, ModelProfileID: profile.ID, Executable: f.exe, Model: profile.Model}, Version: f.version, Workspace: filepath.Join(f.base, "work"), Inbox: filepath.Join(f.base, "inbox"), AxiomPath: f.axiom, Process: target.Process, RepositoryWrite: target.Variant == capabilityproof.RepositoryWriteVariant, ParentEnvironment: []string{"PATH=fixture"}}
	mode, err := ModeDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := AdapterVersion(f.runtime)
	if err != nil {
		t.Fatal(err)
	}
	profileDigest, err := runtimeprofile.ProfileDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	d := f.cfg.Runtimes[0].EffortDeclarations[0]
	hash := strings.Repeat("a", 64)
	resolved := runtimeprofile.ResolvedProfile{ProfileID: profile.ID, RuntimeID: f.runtime, Model: profile.Model, Effort: profile.Effort, ProfileDigest: hash, ConfigurationRevision: f.cfg.Revision, ConfigurationDigest: hash, ExecutableDigest: d.ExecutableDigest, RuntimeVersion: f.version, InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: hash, EffortSourceDigest: hash, CapabilityProofReference: "synthetic-proof"}
	legacy := EffortDeclaration{RuntimeID: f.runtime, RuntimeVersion: d.RuntimeVersion, InvocationMode: d.InvocationMode, ExecutableDigest: d.ExecutableDigest, ProofReference: d.DeclarationID, PrecedenceProven: true, JSONKeys: d.JSONKeys, EnvironmentKeys: d.EnvironmentKeys}
	roots := map[string]string{"user": f.roots.User, "project": f.roots.Project, "managed": f.roots.Managed}
	for _, s := range d.Sources {
		legacy.Sources = append(legacy.Sources, EffortSource{Scope: s.Scope, Root: roots[s.Anchor], Relative: s.Path, Format: s.Format, Directory: s.Directory})
	}
	inv, err := BuildFrozenStageInvocation(request, resolved)
	if err != nil {
		t.Fatal(err)
	}
	effort, err := EffortInspection{Consented: true, Declaration: legacy}.CheckEffortSources(context.Background(), resolved, inv)
	if err != nil {
		t.Fatal(err)
	}
	return capabilityproof.Binding{RuntimeID: f.runtime, ExecutableDigest: f.identity(t, f.exe), Version: f.version, ModeDigest: mode, Variant: target.Variant, AdapterVersion: adapter, AxiomDigest: f.identity(t, f.axiom)},
		capabilityproof.ModelBinding{ProfileID: profile.ID, ProfileDigest: profileDigest, Model: profile.Model, EffortMode: profile.Effort.Mode, EffortToken: profile.Effort.Value, ConfigurationRevision: f.cfg.Revision, EffortSourceDigest: effort}
}

func bindingEvidence(t *testing.T, b capabilityproof.Binding, m capabilityproof.ModelBinding, observed time.Time) capabilityproof.StoredEvidence {
	t.Helper()
	proven := capabilityproof.Result{State: capabilityproof.Proven, Facts: capabilityproof.ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}}
	caps := map[string]capabilityproof.Result{}
	for _, c := range []string{capabilityproof.Read, capabilityproof.Process, capabilityproof.Noninteractive, capabilityproof.AxiomCoordination, capabilityproof.ModelInvocation, "reasoning-effort-" + m.EffortToken} {
		caps[c] = proven
	}
	if b.Variant == capabilityproof.RepositoryWriteVariant {
		caps[capabilityproof.RepositoryWrite] = proven
	}
	model := m
	e := capabilityproof.Evidence{SchemaVersion: 1, Binding: b, Model: &model, ObservedAt: observed, ProbePacketDigest: strings.Repeat("b", 64), Capabilities: caps, Confinement: capabilityproof.Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}}
	if !capabilityproof.ValidEvidence(e) {
		t.Fatal("synthetic Evidence invalid")
	}
	ref, err := capabilityproof.Digest(e)
	if err != nil {
		t.Fatal(err)
	}
	return capabilityproof.StoredEvidence{Ref: ref, Evidence: e}
}

// seed stores Evidence for the fixture's current identity at target.
func (f *bindingFixture) seed(t *testing.T, target capabilityproof.Target) capabilityproof.StoredEvidence {
	t.Helper()
	b, m := f.expected(t, target)
	stored := bindingEvidence(t, b, m, bindingNow.Add(-time.Minute))
	f.scanner.records = append(f.scanner.records, stored)
	return stored
}

func TestBindingSourceObservesCompleteIdentity(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newBindingFixture(t, runtime)
			modes := map[capabilityproof.Target]string{}
			for _, variant := range []string{capabilityproof.ReadOnlyVariant, capabilityproof.RepositoryWriteVariant} {
				for _, process := range []bool{false, true} {
					target := f.target(variant, process)
					f.scanner.records = nil
					stored := f.seed(t, target)
					c, err := f.source.Current(context.Background(), target)
					if err != nil {
						t.Fatalf("%s process=%v: %v", variant, process, err)
					}
					wantBinding, wantModel := f.expected(t, target)
					if c.Binding != wantBinding || c.Model == nil || *c.Model != wantModel || c.EvidenceRef != stored.Ref || !c.ProfileActive {
						t.Fatalf("%s process=%v: got %+v %+v %q, want %+v %+v %q", variant, process, c.Binding, c.Model, c.EvidenceRef, wantBinding, wantModel, stored.Ref)
					}
					if c.Binding.AdapterVersion != runtime+"-adapter-v1" || c.Binding.Version != bindingVersions[runtime] {
						t.Fatalf("non-canonical adapter/version: %+v", c.Binding)
					}
					modes[target] = c.Binding.ModeDigest
				}
			}
			read, write := capabilityproof.ReadOnlyVariant, capabilityproof.RepositoryWriteVariant
			if modes[f.target(read, false)] == modes[f.target(write, false)] {
				t.Fatal("variants share a ModeDigest")
			}
			// Claude widens Bash under Process; Codex emits an identical invocation.
			differs := modes[f.target(read, false)] != modes[f.target(read, true)] && modes[f.target(write, false)] != modes[f.target(write, true)]
			if differs != (runtime == "claude") {
				t.Fatalf("%s Process mode identities: differ=%v", runtime, differs)
			}
		})
	}
}

func TestBindingSourceRuntimeDefaultEffort(t *testing.T) {
	f := newBindingFixture(t, "codex")
	f.cfg.ModelProfiles[0].Effort = runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}
	target := f.target(capabilityproof.ReadOnlyVariant, false)
	stored := f.seed(t, target)
	c, err := f.source.Current(context.Background(), target)
	if err != nil || c.EvidenceRef != stored.Ref || c.Model.EffortMode != runtimeprofile.EffortRuntimeDefault {
		t.Fatalf("runtime-default: %+v %v", c, err)
	}
}

func TestBindingSourceFailsClosed(t *testing.T) {
	read := capabilityproof.ReadOnlyVariant
	cases := []struct {
		name    string
		runtime string
		mutate  func(*testing.T, *bindingFixture, *capabilityproof.Target)
		want    error
	}{
		{"missing-runtime", "codex", func(_ *testing.T, f *bindingFixture, t *capabilityproof.Target) { t.RuntimeID = "claude" }, capabilityproof.ErrBlocked},
		{"missing-profile-id", "codex", func(_ *testing.T, f *bindingFixture, t *capabilityproof.Target) { t.ProfileID = "" }, capabilityproof.ErrBlocked},
		{"unknown-variant", "codex", func(_ *testing.T, f *bindingFixture, t *capabilityproof.Target) { t.Variant = "unconfined" }, capabilityproof.ErrBlocked},
		{"disabled-runtime", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) { f.cfg.Runtimes[0].Enabled = false }, capabilityproof.ErrBlocked},
		{"missing-profile", "claude", func(_ *testing.T, f *bindingFixture, t *capabilityproof.Target) { t.ProfileID = "other-profile" }, capabilityproof.ErrBlocked},
		{"executable-unavailable", "codex", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			if err := os.Remove(f.exe); err != nil {
				t.Fatal(err)
			}
		}, capabilityproof.ErrBlocked},
		{"executable-digest-drift", "claude", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			writeFixtureFile(t, f.exe, "replaced executable\n")
			f.cfg.Runtimes[0].EffortDeclarations[0].ExecutableDigest = f.identity(t, f.exe)
		}, capabilityproof.ErrEvidenceMissing},
		{"invalid-executable-applicability", "codex", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			writeFixtureFile(t, f.exe, "replaced executable\n")
		}, ErrEffortConfigurationUnproven},
		{"unsupported-runtime-version", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) { f.version = "0.159.2" }, capabilityproof.ErrBlocked},
		{"unobserved-runtime-version", "claude", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) { f.version = "" }, capabilityproof.ErrBlocked},
		{"version-drift-from-declaration", "claude", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.cfg.Runtimes[0].EffortDeclarations[0].RuntimeVersion = "2.1.295"
		}, ErrEffortConfigurationUnproven},
		{"process-mismatch", "claude", func(_ *testing.T, f *bindingFixture, t *capabilityproof.Target) { t.Process = true }, capabilityproof.ErrEvidenceMissing},
		{"variant-mismatch", "codex", func(_ *testing.T, f *bindingFixture, t *capabilityproof.Target) {
			t.Variant = capabilityproof.RepositoryWriteVariant
		}, capabilityproof.ErrEvidenceMissing},
		{"axiom-digest-drift", "codex", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			writeFixtureFile(t, f.axiom, "other pinned axiom\n")
		}, capabilityproof.ErrEvidenceMissing},
		{"axiom-unavailable", "claude", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.source.AxiomPath = "relative/axiom"
		}, capabilityproof.ErrBlocked},
		{"profile-digest-drift", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.cfg.ModelProfiles[0].Capabilities = append(f.cfg.ModelProfiles[0].Capabilities, "read")
		}, capabilityproof.ErrEvidenceMissing},
		{"profile-effort-drift", "claude", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.cfg.ModelProfiles[0].Effort.Value = "low"
		}, capabilityproof.ErrEvidenceMissing},
		{"configuration-revision-drift", "claude", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) { f.cfg.Revision = 2 }, capabilityproof.ErrEvidenceMissing},
		{"effort-source-drift", "codex", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			writeFixtureFile(t, filepath.Join(f.roots.Project, ".codex", "config.toml"), "# other\nunrelated = true\n")
		}, capabilityproof.ErrEvidenceMissing},
		{"effort-override-conflict", "claude", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			writeFixtureFile(t, filepath.Join(f.roots.Project, ".claude", "settings.json"), `{"fixtureEffortKey":"max"}`)
		}, ErrEffortOverrideConflict},
		{"missing-declaration", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.cfg.Runtimes[0].EffortDeclarations = nil
		}, ErrEffortConfigurationUnproven},
		{"v2-configuration", "claude", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.source.Configuration = func(context.Context) (runtimeprofile.VersionedConfiguration, error) {
				return runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV2, Configuration: &runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2, Revision: 1}}, nil
			}
		}, ErrEffortConfigurationUnproven},
		{"configuration-unreadable", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.source.Configuration = func(context.Context) (runtimeprofile.VersionedConfiguration, error) {
				return runtimeprofile.VersionedConfiguration{}, runtimeprofile.ErrUnsupportedConfigurationFormat
			}
		}, capabilityproof.ErrBlocked},
		{"effort-not-consented", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) { f.source.EffortConsented = false }, ErrEffortConfigurationUnproven},
		{"unsafe-effort-root", "claude", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			if err := os.Chmod(f.roots.User, 0o770); err != nil {
				t.Fatal(err)
			}
		}, ErrEffortConfigurationUnproven},
		{"symlinked-effort-root", "codex", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			link := filepath.Join(f.base, "managed-link")
			if err := os.Symlink(f.roots.Managed, link); err != nil {
				t.Fatalf("native symlink setup failed on a supported host: %v", err)
			}
			f.roots.Managed = link
		}, ErrEffortConfigurationUnproven},
		{"missing-stage-context", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.source.Stage = func(context.Context, capabilityproof.Target) (StageContext, error) { return StageContext{}, nil }
		}, capabilityproof.ErrBlocked},
		{"adapter-version-mismatch", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.scanner.records[0] = rebind(f.scanner.records[0], func(b *capabilityproof.Binding, _ *capabilityproof.ModelBinding) {
				b.AdapterVersion = "codex-adapter-v0"
			})
		}, capabilityproof.ErrEvidenceMissing},
		{"mode-digest-drift", "claude", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.scanner.records[0] = rebind(f.scanner.records[0], func(b *capabilityproof.Binding, _ *capabilityproof.ModelBinding) {
				b.ModeDigest = strings.Repeat("c", 64)
			})
		}, capabilityproof.ErrEvidenceMissing},
		{"missing-evidence", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) { f.scanner.records = nil }, capabilityproof.ErrEvidenceMissing},
		{"multiple-evidence", "claude", func(t *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			first := f.scanner.records[0].Evidence
			f.scanner.records = append(f.scanner.records, bindingEvidence(t, first.Binding, *first.Model, bindingNow.Add(-2*time.Minute)))
		}, capabilityproof.ErrEvidenceAmbiguous},
		{"corrupt-evidence", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.scanner.records[0].Ref = strings.Repeat("d", 64)
		}, capabilityproof.ErrBlocked},
		{"cancel-during-selection", "claude", nil, context.Canceled},
		{"deadline-during-selection", "codex", func(_ *testing.T, f *bindingFixture, _ *capabilityproof.Target) {
			f.scanner.hook = func(context.Context) error { return context.DeadlineExceeded }
		}, context.DeadlineExceeded},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newBindingFixture(t, c.runtime)
			target := f.target(read, false)
			f.seed(t, target)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if c.mutate != nil {
				c.mutate(t, f, &target)
			} else {
				f.scanner.hook = func(context.Context) error { cancel(); return ctx.Err() }
			}
			got, err := f.source.Current(ctx, target)
			if !errors.Is(err, c.want) || got != (capabilityproof.Current{}) {
				t.Fatalf("got %+v, %v; want %v", got, err, c.want)
			}
			if c.want == capabilityproof.ErrEvidenceMissing && errors.Is(err, capabilityproof.ErrEvidenceAmbiguous) || c.want == capabilityproof.ErrBlocked && (errors.Is(err, capabilityproof.ErrEvidenceMissing) || errors.Is(err, capabilityproof.ErrEvidenceAmbiguous)) {
				t.Fatalf("blocker category conflated: %v", err)
			}
			// Selection runs only after a complete identity was observed.
			if errors.Is(err, ErrEffortConfigurationUnproven) || errors.Is(err, ErrEffortOverrideConflict) || c.name != "corrupt-evidence" && err == capabilityproof.ErrBlocked {
				if f.scanner.scans != 0 {
					t.Fatalf("Evidence scanned before identity was complete")
				}
			}
		})
	}
}

func rebind(s capabilityproof.StoredEvidence, edit func(*capabilityproof.Binding, *capabilityproof.ModelBinding)) capabilityproof.StoredEvidence {
	e := s.Evidence
	model := *e.Model
	edit(&e.Binding, &model)
	e.Model = &model
	ref, _ := capabilityproof.Digest(e)
	return capabilityproof.StoredEvidence{Ref: ref, Evidence: e}
}

func TestBindingSourcePreservesCancellationBeforeObservation(t *testing.T) {
	f := newBindingFixture(t, "codex")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.source.Current(ctx, f.target(capabilityproof.ReadOnlyVariant, false)); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	expired, stop := context.WithDeadline(context.Background(), bindingNow)
	defer stop()
	if _, err := f.source.Current(expired, f.target(capabilityproof.ReadOnlyVariant, false)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

// Stored Evidence never supplies current identity: a record that differs from
// the observation in any field is not adopted, even when it is the only one.
func TestBindingSourceNeverDerivesIdentityFromEvidence(t *testing.T) {
	f := newBindingFixture(t, "claude")
	target := f.target(capabilityproof.ReadOnlyVariant, false)
	b, m := f.expected(t, target)
	other := bindingEvidence(t, b, m, bindingNow.Add(-time.Minute))
	other = rebind(other, func(_ *capabilityproof.Binding, m *capabilityproof.ModelBinding) {
		m.EffortSourceDigest = strings.Repeat("e", 64)
	})
	f.scanner.records = []capabilityproof.StoredEvidence{other}
	if _, err := f.source.Current(context.Background(), target); !errors.Is(err, capabilityproof.ErrEvidenceMissing) {
		t.Fatalf("got %v", err)
	}
	observed, observedModel, err := f.source.observe(context.Background(), target)
	if err != nil || observed != b || observedModel != m {
		t.Fatalf("observation depends on Evidence: %+v %+v %v", observed, observedModel, err)
	}
}

// The declarationId is configuration identity, never capability Evidence: even
// one spelled like an EvidenceRef selects nothing and proves nothing.
func TestBindingSourceDeclarationIsNotCapability(t *testing.T) {
	f := newBindingFixture(t, "codex")
	target := f.target(capabilityproof.ReadOnlyVariant, false)
	stored := f.seed(t, target)
	f.scanner.records = nil
	f.cfg.Runtimes[0].EffortDeclarations[0].DeclarationID = stored.Ref
	if _, err := f.source.Current(context.Background(), target); !errors.Is(err, capabilityproof.ErrEvidenceMissing) {
		t.Fatalf("got %v", err)
	}
	service := capabilityproof.Service{Source: f.source, Store: bindingStore{}, Now: func() time.Time { return bindingNow }}
	if ref, code, err := service.Check(context.Background(), target, []string{capabilityproof.Read}, false, false, ""); ref != "" || code != "capability_unknown" || !errors.Is(err, capabilityproof.ErrBlocked) {
		t.Fatalf("declaration promoted to capability: %q %q %v", ref, code, err)
	}
}

type bindingStore map[string]capabilityproof.Evidence

func (s bindingStore) Load(_ context.Context, ref string) (capabilityproof.Evidence, error) {
	e, ok := s[ref]
	if !ok {
		return capabilityproof.Evidence{}, capabilityproof.ErrInvalidEvidence
	}
	return e, nil
}
func (bindingStore) Invalidate(context.Context, string, string, [sha256.Size]byte) error {
	return capabilityproof.ErrAuthority
}

func v2Projection(c runtimeprofile.ConfigurationV3) runtimeprofile.Configuration {
	cfg := runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2, Revision: c.Revision, ModelProfiles: c.ModelProfiles, Preferences: c.Preferences}
	for _, r := range c.Runtimes {
		cfg.Runtimes = append(cfg.Runtimes, runtimeprofile.Runtime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs})
	}
	return cfg
}

func TestBindingSourceServiceCompatibility(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newBindingFixture(t, runtime)
			target := f.target(capabilityproof.RepositoryWriteVariant, true)
			stored := f.seed(t, target)
			store := bindingStore{stored.Ref: stored.Evidence}
			configured := capabilityproof.ConfiguredSource{Source: f.source, Configuration: v2Projection(f.cfg)}
			direct, err := f.source.Current(context.Background(), target)
			if err != nil {
				t.Fatal(err)
			}
			through, err := configured.Current(context.Background(), target)
			if err != nil || through.Binding != direct.Binding || *through.Model != *direct.Model || through.EvidenceRef != direct.EvidenceRef || !through.ProfileActive {
				t.Fatalf("ConfiguredSource diverges: %+v %v", through, err)
			}
			service := capabilityproof.Service{Source: configured, Store: store, Now: func() time.Time { return bindingNow }}
			ref, code, err := service.Check(context.Background(), target, []string{capabilityproof.Read}, true, true, "")
			if err != nil || code != "" || ref != stored.Ref {
				t.Fatalf("Check: %q %q %v", ref, code, err)
			}
			if _, code, err := service.Check(context.Background(), target, []string{capabilityproof.Read}, true, false, ""); code != "capability_stale" || !errors.Is(err, capabilityproof.ErrBlocked) {
				t.Fatalf("Process contradiction admitted: %q %v", code, err)
			}
			// A Runtime disabled in the freshly loaded configuration deactivates
			// the Profile without promoting the observation.
			disabled := v2Projection(f.cfg)
			disabled.Runtimes[0].Enabled = false
			service.Source = capabilityproof.ConfiguredSource{Source: f.source, Configuration: disabled}
			if _, code, err := service.Check(context.Background(), target, []string{capabilityproof.Read}, true, true, ""); code != "capability_unknown" || !errors.Is(err, capabilityproof.ErrBlocked) {
				t.Fatalf("inactive Profile admitted: %q %v", code, err)
			}
		})
	}
}
