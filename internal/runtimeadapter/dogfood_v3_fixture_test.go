//go:build unix

package runtimeadapter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

// BLD-001B.3c-2c.1: synthetic canonical-v3 preparation for the future dogfood
// cutover (BLD-001B.3c-2b.3). It is isolated from scripts/dogfood-poc.sh,
// which keeps its v1→v2 journey unchanged. Fixed-byte synthetic executables
// are hashed through ExecutableIdentity, never run; the same identity binds
// the synthetic registry entry and the Runtime-owned EffortDeclaration.
type dogfoodV3Fixture struct {
	base     string
	roots    EffortSourceRoots
	exe      map[string]string
	markers  map[string]string
	cfg      runtimeprofile.ConfigurationV3
	registry RegistryVersionObserver
}

var dogfoodV3Versions = map[string]string{"codex": runtimeprofiletest.CodexRuntimeVersion, "claude": runtimeprofiletest.ClaudeRuntimeVersion}

// dogfoodV3Profiles names one Profile per Runtime that MultiRuntime allowlists.
var dogfoodV3Profiles = map[string]string{"codex": runtimeprofiletest.CodexHighProfileID, "claude": runtimeprofiletest.ClaudeLowProfileID}

func newDogfoodV3Fixture(t *testing.T) *dogfoodV3Fixture {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &dogfoodV3Fixture{base: base, exe: map[string]string{}, markers: map[string]string{}, roots: EffortSourceRoots{User: filepath.Join(base, "home"), Project: filepath.Join(base, "project"), Managed: filepath.Join(base, "managed")}}
	for _, root := range []string{f.roots.User, f.roots.Project, f.roots.Managed} {
		if err := os.Mkdir(root, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	cfg := runtimeprofiletest.MultiRuntime()
	var entries []string
	for _, id := range []string{"claude", "codex"} {
		f.exe[id], f.markers[id] = syntheticExecutable(t, base, id)
		digest := identityOf(t, f.exe[id])
		d := runtimeprofiletest.CodexDeclaration()
		if id == "claude" {
			d = runtimeprofiletest.ClaudeDeclaration()
		}
		d.ExecutableDigest = digest
		cfg = runtimeprofiletest.WithDeclaration(cfg, id, d)
		entries = append(entries, registryEntry(id, dogfoodV3Versions[id], digest))
		for _, s := range d.Sources {
			path := filepath.Join(f.root(s.Anchor), filepath.FromSlash(s.Path))
			switch {
			case s.Directory:
				writeFixtureFile(t, filepath.Join(path, "10-base.json"), "{}")
			case s.Format == "toml":
				writeFixtureFile(t, path, "# no effort selector\n")
			default:
				writeFixtureFile(t, path, "{}")
			}
		}
	}
	f.cfg = cfg
	f.registry = mustRegistry(t, registryWire(entries...))
	return f
}

func (f *dogfoodV3Fixture) root(anchor string) string {
	return map[string]string{"user": f.roots.User, "project": f.roots.Project, "managed": f.roots.Managed}[anchor]
}

func (f *dogfoodV3Fixture) lookPath(name string) (string, error) {
	if path, ok := f.exe[name]; ok {
		return path, nil
	}
	return "", os.ErrNotExist
}

func (f *dogfoodV3Fixture) assertNotExecuted(t *testing.T) {
	t.Helper()
	for _, marker := range f.markers {
		assertNotExecuted(t, marker)
	}
}

// inspect derives the EffortSourceDigest for id's Profile over the effective
// invocation with the Profile effort, from the observed applicability.
func (f *dogfoodV3Fixture) inspect(t *testing.T, cfg runtimeprofile.ConfigurationV3, id string, a runtimeprofile.EffortApplicability) (string, error) {
	t.Helper()
	var profile runtimeprofile.ModelProfile
	for _, p := range cfg.ModelProfiles {
		if p.ID == dogfoodV3Profiles[id] {
			profile = p
		}
	}
	// The adapter stage builder accepts only adapter-supported vendor versions,
	// so the synthetic versions use the inspector's minimal invocation form.
	mode := map[string]string{"codex": "exec", "claude": "--print"}[id]
	base := executiongraph.Invocation{RuntimeID: id, Argv: []string{f.exe[id], mode, "--model", profile.Model}, CWD: f.roots.Project, Env: []string{"PATH=fixture"}}
	inv, err := withStageEffort(base, profile.Effort)
	if err != nil {
		t.Fatal(err)
	}
	return InspectEffortSources(context.Background(), EffortSourceRequest{Consented: true, Configuration: decodeV3Fixture(t, cfg), ProfileID: profile.ID, Observed: a, Roots: f.roots, Invocation: inv})
}

func TestDogfoodV3FixtureIsCanonical(t *testing.T) {
	f := newDogfoodV3Fixture(t)
	wire := runtimeprofiletest.Encode(t, f.cfg)
	sum := sha256.Sum256(wire)
	if got := runtimeprofiletest.Digest(t, f.cfg); got != hex.EncodeToString(sum[:]) {
		t.Fatalf("DigestV3 %s is not SHA-256 of EncodeV3", got)
	}
	decoded, err := runtimeprofile.DecodeVersioned(wire)
	if err != nil || decoded.FormatVersion != runtimeprofile.FormatVersionV3 || decoded.V3 == nil {
		t.Fatalf("decode: %+v %v", decoded, err)
	}
	if again := runtimeprofiletest.Encode(t, *decoded.V3); !bytes.Equal(again, wire) {
		t.Fatal("canonical v3 encoding is not stable across decode")
	}
	if decoded.V3.Revision != runtimeprofiletest.InitialRevision {
		t.Fatalf("revision %d", decoded.V3.Revision)
	}
	consumer := runtimeprofiletest.Consumer(t, f.cfg)
	for id := range f.exe {
		if _, err := consumer.EffortDeclaration(runtimeprofile.EffortApplicability{RuntimeID: id, RuntimeVersion: dogfoodV3Versions[id], ExecutableDigest: identityOf(t, f.exe[id]), InvocationMode: runtimeprofiletest.InvocationMode}); err != nil {
			t.Fatalf("%s consumer declaration: %v", id, err)
		}
	}
	// Committed fixture content is host-independent: only the test-time
	// identities embed the temporary paths.
	if bytes.Contains(runtimeprofiletest.Encode(t, runtimeprofiletest.MultiRuntime()), []byte(f.base)) {
		t.Fatal("shared fixture embeds a host path")
	}
	f.assertNotExecuted(t)
}

func TestDogfoodV3FixtureApplicability(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			f := newDogfoodV3Fixture(t)
			ctx := context.Background()
			a, err := ObserveEffortApplicability(ctx, f.lookPath, f.registry.Version, id)
			if err != nil {
				t.Fatal(err)
			}
			want := runtimeprofile.EffortApplicability{RuntimeID: id, RuntimeVersion: dogfoodV3Versions[id], ExecutableDigest: identityOf(t, f.exe[id]), InvocationMode: runtimeprofiletest.InvocationMode}
			if a != want {
				t.Fatalf("observed %+v, want %+v", a, want)
			}
			d, err := runtimeprofile.ApplicableEffortDeclaration(decodeV3Fixture(t, f.cfg), a)
			if err != nil || d.RuntimeVersion != a.RuntimeVersion || d.ExecutableDigest != a.ExecutableDigest {
				t.Fatalf("declaration %+v %v", d, err)
			}
			digest, err := f.inspect(t, f.cfg, id, a)
			if err != nil || len(digest) != 64 {
				t.Fatalf("effort sources: %q %v", digest, err)
			}

			// The declaration for one Runtime never applies to the other.
			other := map[string]string{"codex": "claude", "claude": "codex"}[id]
			crossed := a
			crossed.RuntimeID = other
			if _, err := runtimeprofile.ApplicableEffortDeclaration(decodeV3Fixture(t, f.cfg), crossed); !errors.Is(err, runtimeprofile.ErrEffortDeclarationUnproven) {
				t.Fatalf("implicit cross-Runtime selection: %v", err)
			}
			if _, err := f.registry.Version(ctx, other, a.ExecutableDigest); err == nil {
				t.Fatal("registry resolved another Runtime's identity")
			}

			// A changed Runtime version in the registry leaves the declaration
			// inapplicable; nothing falls back to the declared version.
			bumped := mustRegistry(t, registryWire(registryEntry(id, dogfoodV3Versions[id]+"-next", a.ExecutableDigest)))
			moved, err := ObserveEffortApplicability(ctx, f.lookPath, bumped.Version, id)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := runtimeprofile.ApplicableEffortDeclaration(decodeV3Fixture(t, f.cfg), moved); !errors.Is(err, runtimeprofile.ErrEffortDeclarationUnproven) {
				t.Fatalf("changed version applied: %v", err)
			}

			// A registry without this identity refuses.
			if _, err := ObserveEffortApplicability(ctx, f.lookPath, mustRegistry(t, registryWire()).Version, id); !errors.Is(err, ErrEffortConfigurationUnproven) {
				t.Fatalf("unknown registry digest: %v", err)
			}

			// A missing declaration stays unproven, even with a registered version.
			if _, err := runtimeprofile.ApplicableEffortDeclaration(decodeV3Fixture(t, runtimeprofiletest.WithoutDeclarations(f.cfg, id)), a); !errors.Is(err, runtimeprofile.ErrEffortDeclarationUnproven) {
				t.Fatalf("missing declaration: %v", err)
			}

			// Symlinked effort sources stay refused.
			source := filepath.Join(f.roots.Project, filepath.FromSlash(map[string]string{"codex": ".codex/config.toml", "claude": ".claude/settings.json"}[id]))
			real := source + ".real"
			if err := os.Rename(source, real); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, source); err != nil {
				t.Fatal(err)
			}
			if _, err := f.inspect(t, f.cfg, id, a); err == nil {
				t.Fatal("symlinked effort source accepted")
			}

			// Changed executable bytes change the identity: the registry and
			// the declaration no longer apply.
			if err := os.WriteFile(f.exe[id], []byte("#!/bin/sh\nexit 98\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err := ObserveEffortApplicability(ctx, f.lookPath, f.registry.Version, id); !errors.Is(err, ErrEffortConfigurationUnproven) {
				t.Fatalf("changed executable bytes: %v", err)
			}
			drifted := a
			drifted.ExecutableDigest = identityOf(t, f.exe[id])
			if _, err := runtimeprofile.ApplicableEffortDeclaration(decodeV3Fixture(t, f.cfg), drifted); !errors.Is(err, runtimeprofile.ErrEffortDeclarationUnproven) {
				t.Fatalf("drifted identity applied: %v", err)
			}
			f.assertNotExecuted(t)
		})
	}
}

// The preparation leaves the active dogfood journey and the historical corpus
// alone: the shared synthetic builders stay synthetic, and nothing here adds
// a production registry entry.
func TestDogfoodV3FixtureLeavesProductionRegistry(t *testing.T) {
	f := newDogfoodV3Fixture(t)
	embedded, err := EmbeddedVersionObserver()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(embedded.versions, map[versionKey]string{}) {
		t.Fatalf("production registry changed: %v", embedded.versions)
	}
	for id, exe := range f.exe {
		if _, err := ObserveEffortApplicability(context.Background(), f.lookPath, embedded.Version, id); !errors.Is(err, ErrEffortConfigurationUnproven) {
			t.Fatalf("%s synthetic executable %s resolved in production: %v", id, exe, err)
		}
	}
	f.assertNotExecuted(t)
}
