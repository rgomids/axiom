package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
	"github.com/rgomids/axiom/internal/stageexecution"
)

// Specification 007 AXM-22: the production capability composition. Every test
// uses synthetic local state and synthetic executable files that are never
// executed; no Runtime, probe, inference or Provider effect occurs.

func TestProductionCapabilityCompositionFailsClosed(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	looked := 0
	s, err := productionCapabilityService(root, func(string) (string, error) { looked++; return "", os.ErrNotExist }, stageOperation{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Store.(local.CapabilityStore); !ok {
		t.Fatal("production composition must use the canonical local store")
	}
	source, ok := s.Source.(runtimeadapter.BindingSource)
	if !ok || source.EffortConsented || source.Selector.Evidence == nil {
		t.Fatalf("production source not composed: %#v", s.Source)
	}
	if _, ok := source.Selector.Evidence.(local.CapabilityStore); !ok {
		t.Fatal("selection must scan the canonical store")
	}
	if s.ProbeAdapter == nil || s.ProbeEnvelopes == nil || s.ProbeWriter == nil || s.ProbeAuth == nil {
		t.Fatal("production ports must be present and fail closed without preparation")
	}
	ctx := context.Background()
	for _, variant := range []string{capabilityproof.ReadOnlyVariant, capabilityproof.RepositoryWriteVariant} {
		target := capabilityproof.Target{RuntimeID: "codex", ProfileID: "synthetic", Variant: variant, Process: true}
		view, err := s.Show(ctx, target)
		if !errors.Is(err, capabilityproof.ErrBlocked) || view.EvidenceRef != "" {
			t.Fatalf("fresh state: %+v, %v", view, err)
		}
		for cap, state := range view.States {
			if state != capabilityproof.Unknown || view.Blockers[cap] != "capability_unknown" {
				t.Fatalf("%s unexpectedly proven: %+v", cap, view)
			}
		}
		for _, expectedRef := range []string{"", "synthetic-reviewed-reference"} {
			ref, code, err := s.Check(ctx, target, nil, variant == capabilityproof.RepositoryWriteVariant, true, expectedRef)
			if ref != "" || code != "capability_unknown" || !errors.Is(err, capabilityproof.ErrBlocked) {
				t.Fatalf("check = %q %q %v", ref, code, err)
			}
		}
		if _, err := s.PreviewProbe(ctx, target); !errors.Is(err, capabilityproof.ErrBlocked) {
			t.Fatalf("unconfigured probe = %v", err)
		}
	}
	// Fresh configuration blocks before any executable lookup, and neither
	// composition nor blocked reads/checks create even local state.
	if looked != 0 {
		t.Fatal("blocked composition resolved an executable")
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("blocked composition created state: %v", err)
	}
}

func TestProductionCapabilityCompositionRejectsUnsafeRoot(t *testing.T) {
	if _, err := productionCapabilityService("relative-state", nil, stageOperation{}); !errors.Is(err, local.ErrUnsafe) {
		t.Fatalf("unsafe root = %v", err)
	}
	f := newCompositionFixture(t)
	for name, mutate := range map[string]func(*capabilityDependencies){
		"lookup":  func(d *capabilityDependencies) { d.lookPath = nil },
		"version": func(d *capabilityDependencies) { d.version = nil },
		"axiom":   func(d *capabilityDependencies) { d.axiomPath = "axiom" },
		"stage":   func(d *capabilityDependencies) { d.stage = nil },
		"roots":   func(d *capabilityDependencies) { d.effortRoots = nil },
	} {
		d := f.dependencies()
		mutate(&d)
		if _, err := composeCapabilityService(d); !errors.Is(err, capabilityproof.ErrBlocked) {
			t.Fatalf("%s: incomplete composition = %v", name, err)
		}
	}
}

// compositionFixture is a canonical v3 Codex configuration with an applicable
// declaration, synthetic never-executed executables, trusted empty effort
// roots, an adapter-supported synthetic version, a registry entry injected through the composition seam
// (never the embedded registry) and the canonical local Evidence store.
type compositionFixture struct {
	base, state, exe, axiom string
	roots                   runtimeadapter.EffortSourceRoots
	cfg                     runtimeprofile.ConfigurationV3
	version                 string
	registered              map[[2]string]string
	stage                   runtimeadapter.StageContext
	consented               bool
	profiles                local.RuntimeProfileStore
	evidence                local.CapabilityStore
}

func newCompositionFixture(t *testing.T) *compositionFixture {
	t.Helper()
	base, err := os.MkdirTemp("", "axm22-composition-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(base) })
	if base, err = filepath.EvalSymlinks(base); err != nil {
		t.Fatal(err)
	}
	f := &compositionFixture{base: base, state: filepath.Join(base, "state"), exe: filepath.Join(base, "bin", "codex"), axiom: filepath.Join(base, "bin", "axiom"), version: "0.159.1", consented: true}
	f.roots = runtimeadapter.EffortSourceRoots{User: filepath.Join(base, "home"), Project: filepath.Join(base, "project"), Managed: filepath.Join(base, "managed")}
	for _, dir := range []string{filepath.Join(base, "bin"), f.roots.User, f.roots.Project, f.roots.Managed, filepath.Join(base, "work"), filepath.Join(base, "inbox")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{f.exe, f.axiom} {
		writeComposition(t, path, "synthetic "+filepath.Base(path)+"; never execute\n", 0o700)
	}
	f.stage = runtimeadapter.StageContext{Workspace: filepath.Join(base, "work"), Inbox: filepath.Join(base, "inbox"), ParentEnvironment: []string{"HOME=" + f.roots.User, "PATH=/usr/bin:/bin"}}
	f.registered = map[[2]string]string{{"codex", f.identity(t, f.exe)}: f.version}
	d := runtimeprofiletest.CodexDeclaration()
	d.ExecutableDigest, d.RuntimeVersion = f.identity(t, f.exe), f.version
	f.cfg = runtimeprofiletest.WithDeclaration(runtimeprofiletest.CodexOnly(), "codex", d)
	if f.profiles, err = local.NewRuntimeProfileStore(f.state); err != nil {
		t.Fatal(err)
	}
	if f.evidence, err = local.NewCapabilityStore(f.state); err != nil {
		t.Fatal(err)
	}
	f.publish(t, func(*runtimeprofile.ConfigurationV3) {})
	return f
}

func writeComposition(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func (f *compositionFixture) identity(t *testing.T, path string) string {
	t.Helper()
	digest, err := runtimeadapter.ExecutableIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

// publish authors the next canonical revision through the real store with an
// exact reviewed token, as v3 authoring does.
func (f *compositionFixture) publish(t *testing.T, mutate func(*runtimeprofile.ConfigurationV3)) {
	t.Helper()
	ctx := context.Background()
	snapshot, err := f.profiles.Inspect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mutate(&f.cfg)
	f.cfg.Revision = snapshot.Revision + 1
	wire, err := runtimeprofile.EncodeV3(f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.profiles.Publish(ctx, snapshot.Token, wire); err != nil {
		t.Fatal(err)
	}
}

func (f *compositionFixture) dependencies() capabilityDependencies {
	return capabilityDependencies{
		stateRoot: f.state,
		lookPath: func(name string) (string, error) {
			if name != "codex" {
				return "", os.ErrNotExist
			}
			return f.exe, nil
		},
		version: func(ctx context.Context, id, digest string) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			if v, ok := f.registered[[2]string{id, digest}]; ok {
				return v, nil
			}
			return "", capabilityproof.ErrBlocked
		},
		axiomPath: f.axiom,
		stage: func(context.Context, capabilityproof.Target) (runtimeadapter.StageContext, error) {
			return f.stage, nil
		},
		// The production PM-13 resolver (variant, child HOME, repository
		// root); only the platform managed directory is replaced by the
		// fixture's trusted synthetic root.
		effortRoots: func(ctx context.Context, t capabilityproof.Target, env []string) (runtimeadapter.EffortSourceRoots, error) {
			roots, err := runtimeadapter.StageEffortRoots{Repository: f.roots.Project}.Resolve(ctx, t, env)
			roots.Managed = f.roots.Managed
			return roots, err
		},
		effortConsented: f.consented,
	}
}

func (f *compositionFixture) service(t *testing.T) capabilityproof.Service {
	t.Helper()
	s, err := composeCapabilityService(f.dependencies())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func compositionTarget(profile string, variant string, process bool) capabilityproof.Target {
	return capabilityproof.Target{RuntimeID: "codex", ProfileID: profile, Variant: variant, Process: process}
}

// expected computes the identity with the independent canonical oracles —
// ModeDigest over the effective request and PE-05a CheckEffortSources over the
// dispatch-time frozen invocation — never through BindingSource.
func (f *compositionFixture) expected(t *testing.T, target capabilityproof.Target) (capabilityproof.Binding, capabilityproof.ModelBinding) {
	t.Helper()
	var profile runtimeprofile.ModelProfile
	for _, p := range f.cfg.ModelProfiles {
		if p.ID == target.ProfileID {
			profile = p
		}
	}
	request := runtimeadapter.StageInvocationRequest{Profile: runtimeadapter.CommandProfile{RuntimeID: "codex", ModelProfileID: profile.ID, Executable: f.exe, Model: profile.Model}, Version: f.version, Workspace: f.stage.Workspace, Inbox: f.stage.Inbox, AxiomPath: f.axiom, Process: target.Process, RepositoryWrite: target.Variant == capabilityproof.RepositoryWriteVariant, ParentEnvironment: f.stage.ParentEnvironment}
	mode, err := runtimeadapter.ModeDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	adapter, _ := runtimeadapter.AdapterVersion("codex")
	profileDigest, err := runtimeprofile.ProfileDigest(profile)
	if err != nil {
		t.Fatal(err)
	}
	d := f.cfg.Runtimes[0].EffortDeclarations[0]
	hash := strings.Repeat("a", 64)
	resolved := runtimeprofile.ResolvedProfile{ProfileID: profile.ID, RuntimeID: "codex", Model: profile.Model, Effort: profile.Effort, ProfileDigest: hash, ConfigurationRevision: f.cfg.Revision, ConfigurationDigest: hash, ExecutableDigest: f.identity(t, f.exe), RuntimeVersion: f.version, InvocationMode: "noninteractive", ObservationRevision: 1, ObservationDigest: hash, EffortSourceDigest: hash, CapabilityProofReference: "synthetic-proof"}
	legacy := runtimeadapter.EffortDeclaration{RuntimeID: "codex", RuntimeVersion: d.RuntimeVersion, InvocationMode: d.InvocationMode, ExecutableDigest: d.ExecutableDigest, ProofReference: d.DeclarationID, PrecedenceProven: true, JSONKeys: d.JSONKeys, EnvironmentKeys: d.EnvironmentKeys}
	roots := map[string]string{"user": f.roots.User, "project": f.roots.Project, "managed": f.roots.Managed}
	for _, s := range d.Sources {
		legacy.Sources = append(legacy.Sources, runtimeadapter.EffortSource{Scope: s.Scope, Root: roots[s.Anchor], Relative: s.Path, Format: s.Format, Directory: s.Directory})
	}
	inv, err := runtimeadapter.BuildFrozenStageInvocation(request, resolved)
	if err != nil {
		t.Fatal(err)
	}
	effort, err := runtimeadapter.EffortInspection{Consented: true, Declaration: legacy}.CheckEffortSources(context.Background(), resolved, inv)
	if err != nil {
		t.Fatal(err)
	}
	return capabilityproof.Binding{RuntimeID: "codex", ExecutableDigest: f.identity(t, f.exe), Version: f.version, ModeDigest: mode, Variant: target.Variant, AdapterVersion: adapter, AxiomDigest: f.identity(t, f.axiom)},
		capabilityproof.ModelBinding{ProfileID: profile.ID, ProfileDigest: profileDigest, Model: profile.Model, EffortMode: profile.Effort.Mode, EffortToken: profile.Effort.Value, ConfigurationRevision: f.cfg.Revision, EffortSourceDigest: effort}
}

// record stores synthetic Evidence for the current identity of target in the
// canonical store. caps lists the proven capabilities; packet distinguishes
// otherwise identical records.
func (f *compositionFixture) record(t *testing.T, target capabilityproof.Target, observed time.Time, packet string, caps ...string) string {
	t.Helper()
	b, m := f.expected(t, target)
	proven := capabilityproof.Result{State: capabilityproof.Proven, Facts: capabilityproof.ProbeFacts{AuthenticationProven: true, BehavioralProbe: true, Verified: true}}
	results := map[string]capabilityproof.Result{}
	for _, c := range caps {
		results[c] = proven
	}
	e := capabilityproof.Evidence{SchemaVersion: 1, Binding: b, Model: &m, ObservedAt: observed, ProbePacketDigest: strings.Repeat(packet, 64), Capabilities: results, Confinement: capabilityproof.Confinement{Level: "unknown", NotTested: []string{"home", "vendor-home", "tmpdir", "system-temp", "external-network"}}}
	ref, err := f.evidence.Create(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

var writeCaps = []string{capabilityproof.Read, capabilityproof.RepositoryWrite, capabilityproof.Noninteractive, capabilityproof.AxiomCoordination, capabilityproof.ModelInvocation, "reasoning-effort-high"}

// Positive synthetic flow: the composed production boundary observes the
// current identity independently, the canonical digests agree, the unique
// matching EvidenceRef is selected and only proven capabilities are admitted.
func TestCapabilityCompositionSyntheticPositive(t *testing.T) {
	ctx := context.Background()
	f := newCompositionFixture(t)
	target := compositionTarget(runtimeprofiletest.CodexHighProfileID, capabilityproof.RepositoryWriteVariant, false)
	ref := f.record(t, target, time.Now().Add(-time.Minute), "b", writeCaps...)
	s := f.service(t)
	view, err := s.Show(ctx, target)
	wantBinding, wantModel := f.expected(t, target)
	if err != nil || view.EvidenceRef != ref || view.Binding != wantBinding || view.Model == nil || *view.Model != wantModel {
		t.Fatalf("preview %+v %v", view, err)
	}
	for _, cap := range writeCaps {
		if view.States[cap] != capabilityproof.Proven {
			t.Fatalf("%s not proven: %+v", cap, view.States)
		}
	}
	if got, code, err := s.Check(ctx, target, []string{capabilityproof.Read}, true, false, ref); err != nil || got != ref || code != "" {
		t.Fatalf("check %q %q %v", got, code, err)
	}
	// A selected EvidenceRef does not bypass evaluation: an unproven requested
	// capability still blocks.
	if _, code, err := s.Check(ctx, target, []string{"synthetic-unproven"}, true, false, ref); err == nil || code == "" {
		t.Fatal("unproven capability admitted")
	}
	// Process=false Evidence never satisfies a Process=true request.
	processTarget := compositionTarget(runtimeprofiletest.CodexHighProfileID, capabilityproof.RepositoryWriteVariant, true)
	if _, _, err := s.Check(ctx, processTarget, nil, true, true, ""); err == nil {
		t.Fatal("process capability inferred from Process=false Evidence")
	}
	if _, _, err := s.Check(ctx, target, nil, true, true, ref); err == nil {
		t.Fatal("process mode differs from the observed Target")
	}
	// The read-only variant is another identity, and PM-13 blocks read-only
	// children even when matching Evidence exists.
	readOnly := compositionTarget(runtimeprofiletest.CodexHighProfileID, capabilityproof.ReadOnlyVariant, false)
	if _, _, err := s.Check(ctx, readOnly, nil, false, false, ""); err == nil {
		t.Fatal("variant inferred from repository-write Evidence")
	}
	// The runtime-default Profile is a distinct ModelBinding.
	if _, _, err := s.Check(ctx, compositionTarget(runtimeprofiletest.CodexDefaultProfileID, capabilityproof.RepositoryWriteVariant, false), nil, true, false, ""); err == nil {
		t.Fatal("Profile inferred from another Profile's Evidence")
	}
}

// Preview, check and the dispatch-time check re-observe the same intended
// identity; drift at any step makes the reviewed reference stale.
func TestCapabilityCompositionPreviewDispatchConsistency(t *testing.T) {
	ctx := context.Background()
	drifts := map[string]func(*testing.T, *compositionFixture){
		"configuration": func(t *testing.T, f *compositionFixture) { f.publish(t, func(*runtimeprofile.ConfigurationV3) {}) },
		"effort-source": func(t *testing.T, f *compositionFixture) {
			os.MkdirAll(filepath.Join(f.roots.Project, ".codex"), 0o700)
			writeComposition(t, filepath.Join(f.roots.Project, ".codex", "config.toml"), "model = \"synthetic-codex-model\"\n", 0o600)
		},
		"axiom": func(t *testing.T, f *compositionFixture) {
			os.Remove(f.axiom)
			writeComposition(t, f.axiom, "replaced axiom; never execute\n", 0o700)
		},
	}
	for name, drift := range drifts {
		for step := range 2 {
			t.Run(name+[]string{"-after-preview", "-before-dispatch"}[step], func(t *testing.T) {
				f := newCompositionFixture(t)
				target := compositionTarget(runtimeprofiletest.CodexHighProfileID, capabilityproof.RepositoryWriteVariant, false)
				ref := f.record(t, target, time.Now().Add(-time.Minute), "b", writeCaps...)
				s := f.service(t)
				if view, err := s.Show(ctx, target); err != nil || view.EvidenceRef != ref {
					t.Fatalf("preview %v", err)
				}
				if step == 1 {
					if got, _, err := s.Check(ctx, target, nil, true, false, ref); err != nil || got != ref {
						t.Fatalf("check %v", err)
					}
				}
				drift(t, f)
				if got, _, err := s.Check(ctx, target, nil, true, false, ref); err == nil || got != "" {
					t.Fatalf("drifted identity admitted: %q", got)
				}
			})
		}
	}
}

// Negative integration matrix: every scenario fails closed with no proof.
func TestCapabilityCompositionNegativeMatrix(t *testing.T) {
	high := runtimeprofiletest.CodexHighProfileID
	write := capabilityproof.RepositoryWriteVariant
	cases := map[string]struct {
		target func() capabilityproof.Target
		setup  func(*testing.T, *compositionFixture)
		ctx    func() (context.Context, context.CancelFunc)
	}{
		"missing-runtime": {target: func() capabilityproof.Target {
			tg := compositionTarget(high, write, false)
			tg.RuntimeID = "claude"
			return tg
		}},
		"disallowed-profile": {target: func() capabilityproof.Target {
			return compositionTarget(runtimeprofiletest.UnknownProfileID, write, false)
		}},
		"unknown-variant": {target: func() capabilityproof.Target { return compositionTarget(high, "unconfined", false) }},
		"disabled-runtime": {setup: func(t *testing.T, f *compositionFixture) {
			f.publish(t, func(c *runtimeprofile.ConfigurationV3) { c.Runtimes[0].Enabled = false })
		}},
		"different-profile": {setup: func(t *testing.T, f *compositionFixture) {
			f.publish(t, func(c *runtimeprofile.ConfigurationV3) { c.ModelProfiles[0].Model = "synthetic-other-model" })
		}},
		"changed-revision": {setup: func(t *testing.T, f *compositionFixture) { f.publish(t, func(*runtimeprofile.ConfigurationV3) {}) }},
		"missing-declaration": {setup: func(t *testing.T, f *compositionFixture) {
			f.publish(t, func(c *runtimeprofile.ConfigurationV3) { c.Runtimes[0].EffortDeclarations = nil })
		}},
		"inapplicable-declaration": {setup: func(t *testing.T, f *compositionFixture) {
			f.publish(t, func(c *runtimeprofile.ConfigurationV3) {
				c.Runtimes[0].EffortDeclarations[0].ExecutableDigest = strings.Repeat("1", 64)
			})
		}},
		"explicit-effort-conflict": {setup: func(t *testing.T, f *compositionFixture) {
			os.MkdirAll(filepath.Join(f.roots.User, ".codex"), 0o700)
			writeComposition(t, filepath.Join(f.roots.User, ".codex", "config.toml"), "model_reasoning_effort = \"low\"\n", 0o600)
		}},
		"changed-effort-source": {setup: func(t *testing.T, f *compositionFixture) {
			writeComposition(t, filepath.Join(f.roots.Managed, "managed_config.toml"), "model = \"synthetic-codex-model\"\n", 0o600)
		}},
		"replaced-executable": {setup: func(t *testing.T, f *compositionFixture) {
			digest := f.identity(t, f.exe)
			os.Remove(f.exe)
			writeComposition(t, f.exe, "replaced; never execute\n", 0o700)
			f.registered[[2]string{"codex", f.identity(t, f.exe)}] = f.registered[[2]string{"codex", digest}]
		}},
		"missing-executable": {setup: func(t *testing.T, f *compositionFixture) { os.Remove(f.exe) }},
		"symlinked-executable": {setup: func(t *testing.T, f *compositionFixture) {
			target := filepath.Join(f.base, "bin", "codex-real")
			os.Rename(f.exe, target)
			os.Symlink(target, f.exe)
		}},
		"changed-runtime-version": {setup: func(t *testing.T, f *compositionFixture) {
			f.registered[[2]string{"codex", f.identity(t, f.exe)}] = "synthetic-codex-2"
		}},
		"unknown-registry-digest": {setup: func(t *testing.T, f *compositionFixture) { f.registered = map[[2]string]string{} }},
		"registry-runtime-mismatch": {setup: func(t *testing.T, f *compositionFixture) {
			f.registered = map[[2]string]string{{"claude", f.identity(t, f.exe)}: f.version}
		}},
		"changed-axiom": {setup: func(t *testing.T, f *compositionFixture) {
			os.Remove(f.axiom)
			writeComposition(t, f.axiom, "other axiom; never execute\n", 0o700)
		}},
		"changed-mode": {target: func() capabilityproof.Target {
			return compositionTarget(high, capabilityproof.ReadOnlyVariant, false)
		}},
		"codex-home-override": {setup: func(t *testing.T, f *compositionFixture) {
			f.stage.ParentEnvironment = append(f.stage.ParentEnvironment, "CODEX_HOME="+f.roots.User)
		}},
		"missing-child-home": {setup: func(t *testing.T, f *compositionFixture) {
			f.stage.ParentEnvironment = []string{"PATH=/usr/bin:/bin"}
		}},
		"changed-process": {target: func() capabilityproof.Target { return compositionTarget(high, write, true) }},
		"no-evidence": {setup: func(t *testing.T, f *compositionFixture) {
			os.RemoveAll(filepath.Join(f.state, "capability-evidence"))
		}},
		"multiple-evidence": {setup: func(t *testing.T, f *compositionFixture) {
			f.record(t, compositionTarget(high, write, false), time.Now().Add(-2*time.Minute), "c", writeCaps...)
		}},
		"corrupt-evidence": {setup: func(t *testing.T, f *compositionFixture) {
			writeComposition(t, filepath.Join(f.state, "capability-evidence", "v1", strings.Repeat("d", 64)+".json"), "{corrupt", 0o600)
		}},
		"unsafe-effort-root": {setup: func(t *testing.T, f *compositionFixture) { os.Chmod(f.roots.Project, 0o777) }},
		"symlinked-effort-root": {setup: func(t *testing.T, f *compositionFixture) {
			moved := f.roots.Project + "-real"
			os.Rename(f.roots.Project, moved)
			os.Symlink(moved, f.roots.Project)
		}},
		"unauthorized-inspection": {setup: func(t *testing.T, f *compositionFixture) { f.consented = false }},
		"unsupported-local-config": {setup: func(t *testing.T, f *compositionFixture) {
			os.WriteFile(filepath.Join(f.state, "runtime-profiles", "v1", "configuration.json"), []byte(`{"formatVersion":4}`), 0o600)
		}},
		"retired-local-config": {setup: func(t *testing.T, f *compositionFixture) {
			ctx := context.Background()
			snapshot, _ := f.profiles.Inspect(ctx)
			digest, _ := runtimeprofile.DigestV3(*snapshot.V3)
			tomb, _ := runtimeprofile.EncodeTombstone(runtimeprofile.Tombstone{Kind: runtimeprofile.TombstoneKind, FormatVersion: 1, Revision: snapshot.Revision + 1, PriorConfigurationDigest: digest})
			if _, err := f.profiles.Publish(ctx, snapshot.Token, tomb); err != nil {
				t.Fatal(err)
			}
		}},
		"pending-recovery": {setup: func(t *testing.T, f *compositionFixture) {
			writeComposition(t, filepath.Join(f.state, "runtime-profiles", "v1", ".axiom-stage-file-synthetic"), "{}", 0o600)
		}},
		"missing-authorization-epoch": {setup: func(t *testing.T, f *compositionFixture) {
			os.Remove(filepath.Join(f.state, "runtime-profiles", "v1", "authorization-epoch.json"))
		}},
		"expired-evidence": {setup: func(t *testing.T, f *compositionFixture) {
			os.RemoveAll(filepath.Join(f.state, "capability-evidence"))
			f.record(t, compositionTarget(high, write, false), time.Now().Add(-capabilityproof.MaxAge-time.Hour), "e", writeCaps...)
		}},
		"invalidated-evidence": {setup: func(t *testing.T, f *compositionFixture) {
			ctx := context.Background()
			s := f.service(t)
			preview, err := s.PreviewInvalidate(ctx, compositionTarget(high, write, false))
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Invalidate(ctx, preview, capabilityproof.LocalAuthority{Actor: "synthetic-operator", Preview: preview}); err != nil {
				t.Fatal(err)
			}
		}},
		"canceled": {ctx: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, func() {}
		}},
		"deadline-exceeded": {ctx: func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newCompositionFixture(t)
			baseline := compositionTarget(high, write, false)
			ref := f.record(t, baseline, time.Now().Add(-time.Minute), "b", writeCaps...)
			if got, _, err := f.service(t).Check(context.Background(), baseline, nil, true, false, ref); err != nil || got != ref {
				t.Fatalf("baseline not admitted: %v", err)
			}
			if c.setup != nil {
				c.setup(t, f)
			}
			target := baseline
			if c.target != nil {
				target = c.target()
			}
			ctx, cancel := context.Background(), context.CancelFunc(func() {})
			if c.ctx != nil {
				ctx, cancel = c.ctx()
			}
			defer cancel()
			s := f.service(t)
			write, process := target.Variant == capabilityproof.ReadOnlyVariant, target.Process
			for _, expected := range []string{"", ref} {
				got, code, err := s.Check(ctx, target, nil, write, process, expected)
				if err == nil || got != "" || code == "" {
					t.Fatalf("%s admitted: ref %q code %q", name, got, code)
				}
			}
			view, err := s.Show(ctx, target)
			for cap, state := range view.States {
				if state == capabilityproof.Proven && err != nil {
					t.Fatalf("%s: %s proven through a failed observation", name, cap)
				}
			}
		})
	}
}

// A declaration never proves support: with no Evidence an applicable
// declaration and a clean identity still leave every capability unknown.
func TestCapabilityCompositionDeclarationIsNotProof(t *testing.T) {
	f := newCompositionFixture(t)
	target := compositionTarget(runtimeprofiletest.CodexHighProfileID, capabilityproof.RepositoryWriteVariant, false)
	view, err := f.service(t).Show(context.Background(), target)
	if err == nil || view.EvidenceRef != "" {
		t.Fatalf("declaration selected Evidence: %+v %v", view, err)
	}
	for cap, state := range view.States {
		if state == capabilityproof.Proven {
			t.Fatalf("%s proven by a declaration", cap)
		}
	}
}

// The production composition over a valid canonical v3 configuration and a
// resolvable synthetic executable blocks at the empty embedded registry; the
// production PM-13 inputs still refuse without operation consent, for a
// read-only Target and without a trusted repository root.
func TestProductionCapabilityCompositionEmbeddedRegistryBlocks(t *testing.T) {
	ctx := context.Background()
	f := newCompositionFixture(t)
	target := compositionTarget(runtimeprofiletest.CodexHighProfileID, capabilityproof.RepositoryWriteVariant, false)
	ref := f.record(t, target, time.Now().Add(-time.Minute), "b", writeCaps...)
	op := stageOperation{repository: f.roots.Project, workspaceRoot: f.base, consented: true}
	s, err := productionCapabilityService(f.state, f.dependencies().lookPath, op)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"", ref} {
		if got, code, err := s.Check(ctx, target, nil, true, false, expected); err == nil || got != "" || code != "capability_unknown" {
			t.Fatalf("production admitted synthetic identity: %q %q %v", got, code, err)
		}
	}
	source := s.Source.(runtimeadapter.BindingSource)
	if !source.EffortConsented {
		t.Fatal("operation consent not composed")
	}
	if s, _ := productionCapabilityService(f.state, f.dependencies().lookPath, stageOperation{repository: f.roots.Project, workspaceRoot: f.base}); s.Source.(runtimeadapter.BindingSource).EffortConsented {
		t.Fatal("consent without --authorize-local")
	}
	// The operation context is its own namespaces and inherited environment.
	c, err := op.context(ctx, target)
	if err != nil || c.Workspace != filepath.Join(f.base, "worktrees") || c.Inbox != filepath.Join(f.base, "inboxes") || len(c.ParentEnvironment) != len(os.Environ()) {
		t.Fatalf("operation context %+v %v", c, err)
	}
	if _, err := (stageOperation{workspaceRoot: "relative"}).context(ctx, target); !errors.Is(err, capabilityproof.ErrBlocked) {
		t.Fatalf("relative workspace root: %v", err)
	}
	// With the synthetic registry injected, production PM-13 roots refuse
	// unconsented inspection, a read-only Target and an untrusted repository.
	for name, mutate := range map[string]func(*capabilityDependencies, *capabilityproof.Target){
		"unconsented": func(d *capabilityDependencies, _ *capabilityproof.Target) { d.effortConsented = false },
		"read-only": func(d *capabilityDependencies, t *capabilityproof.Target) {
			d.effortRoots = runtimeadapter.StageEffortRoots{Repository: f.roots.Project}.Resolve
			t.Variant = capabilityproof.ReadOnlyVariant
		},
		"untrusted-repository": func(d *capabilityDependencies, _ *capabilityproof.Target) {
			d.effortRoots = runtimeadapter.StageEffortRoots{Repository: "repository"}.Resolve
		},
	} {
		d, tg := f.dependencies(), target
		mutate(&d, &tg)
		if _, _, err := mustCompose(t, d).Check(ctx, tg, nil, tg.Variant == capabilityproof.RepositoryWriteVariant, false, ""); err == nil {
			t.Fatalf("%s admitted", name)
		}
	}
}

// The supervisor's dispatch inspector is consented by committed authority,
// resolves PM-13 roots and admits only reviewed repository-write workspaces.
func TestProductionEffortInspectorComposition(t *testing.T) {
	f := newCompositionFixture(t)
	children := []stageexecution.ChildRights{
		{Workspace: filepath.Join(f.base, "worktrees", "a"), Effects: []executiongraph.Effect{{Kind: "repository-write"}}},
		{Workspace: filepath.Join(f.base, "worktrees", "b"), Effects: []executiongraph.Effect{{Kind: "process"}}},
	}
	i, err := productionEffortInspector(f.state, f.roots.Project, children)
	if err != nil {
		t.Fatal(err)
	}
	if !i.Consented || i.Roots == nil || i.Configuration == nil || i.Version == nil || len(i.Workspaces) != 1 || i.Workspaces[0] != children[0].Workspace {
		t.Fatalf("inspector %+v", i)
	}
	if _, err := productionEffortInspector("relative", f.roots.Project, children); err == nil {
		t.Fatal("unsafe state root")
	}
}

func mustCompose(t *testing.T, d capabilityDependencies) capabilityproof.Service {
	t.Helper()
	s, err := composeCapabilityService(d)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
