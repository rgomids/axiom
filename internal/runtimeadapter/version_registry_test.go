//go:build unix

package runtimeadapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/capabilityproof"
	"github.com/rgomids/axiom/internal/runtimeprofile"
	"github.com/rgomids/axiom/internal/runtimeprofile/runtimeprofiletest"
)

// syntheticExecutable writes harmless fixed bytes that record any execution in
// a sibling marker; tests assert the marker never appears.
func syntheticExecutable(t *testing.T, dir, runtimeID string) (path, marker string) {
	t.Helper()
	path, marker = filepath.Join(dir, "bin", runtimeID), filepath.Join(dir, "executed-"+runtimeID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n: >\""+marker+"\"\nexit 97\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path, marker
}

func assertNotExecuted(t *testing.T, marker string) {
	t.Helper()
	if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("synthetic executable ran or marker unreadable: %v", err)
	}
}

func registryWire(entries ...string) []byte {
	return []byte(`{"schema":"` + VersionRegistrySchema + `","entries":[` + strings.Join(entries, ",") + `]}`)
}

func registryEntry(runtimeID, version, digest string) string {
	return fmt.Sprintf(`{"runtimeId":%q,"version":%q,"executableDigest":%q}`, runtimeID, version, digest)
}

func mustRegistry(t *testing.T, wire []byte) RegistryVersionObserver {
	t.Helper()
	o, err := decodeVersionRegistry(wire)
	if err != nil {
		t.Fatalf("synthetic registry: %v", err)
	}
	return o
}

// PM-5: the production registry is the embedded file; it decodes, holds no
// fabricated executable identity, and refuses every lookup.
func TestEmbeddedVersionRegistryIsEmptyAndRefuses(t *testing.T) {
	o, err := EmbeddedVersionObserver()
	if err != nil {
		t.Fatalf("embedded registry: %v", err)
	}
	if len(o.versions) != 0 {
		t.Fatalf("production registry carries unapproved entries: %v", o.versions)
	}
	for _, id := range []string{"codex", "claude"} {
		for _, digest := range []string{runtimeprofiletest.CodexExecutableDigest, runtimeprofiletest.ClaudeExecutableDigest} {
			if v, err := o.Version(context.Background(), id, digest); v != "" || !errors.Is(err, capabilityproof.ErrBlocked) {
				t.Fatalf("%s/%s: %q %v", id, digest, v, err)
			}
		}
	}
	if v, err := (RegistryVersionObserver{}).Version(context.Background(), "codex", runtimeprofiletest.CodexExecutableDigest); v != "" || !errors.Is(err, capabilityproof.ErrBlocked) {
		t.Fatalf("zero observer: %q %v", v, err)
	}
}

func TestRegistryVersionObserverExactLookup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	codexExe, codexMarker := syntheticExecutable(t, dir, "codex")
	claudeExe, claudeMarker := syntheticExecutable(t, dir, "claude")
	codexDigest, claudeDigest := identityOf(t, codexExe), identityOf(t, claudeExe)
	entries := []string{registryEntry("claude", runtimeprofiletest.ClaudeRuntimeVersion, claudeDigest), registryEntry("codex", runtimeprofiletest.CodexRuntimeVersion, codexDigest)}
	o := mustRegistry(t, registryWire(entries...))
	if v, err := o.Version(ctx, "codex", codexDigest); err != nil || v != runtimeprofiletest.CodexRuntimeVersion {
		t.Fatalf("codex: %q %v", v, err)
	}
	if v, err := o.Version(ctx, "claude", claudeDigest); err != nil || v != runtimeprofiletest.ClaudeRuntimeVersion {
		t.Fatalf("claude: %q %v", v, err)
	}
	refusals := map[string][2]string{
		"wrong-runtime":   {"claude", codexDigest},
		"wrong-digest":    {"codex", claudeDigest},
		"unknown-digest":  {"codex", strings.Repeat("a", 64)},
		"unknown-runtime": {runtimeprofiletest.UnknownRuntimeID, codexDigest},
		"uppercase":       {"codex", strings.ToUpper(codexDigest)},
		"empty":           {"codex", ""},
	}
	for name, c := range refusals {
		if v, err := o.Version(ctx, c[0], c[1]); v != "" || !errors.Is(err, capabilityproof.ErrBlocked) {
			t.Errorf("%s: %q %v", name, v, err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if v, err := o.Version(canceled, "codex", codexDigest); v != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %q %v", v, err)
	}
	assertNotExecuted(t, codexMarker)
	assertNotExecuted(t, claudeMarker)
}

func TestDecodeVersionRegistryFailsClosed(t *testing.T) {
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	valid := registryEntry("codex", "1.0.0", a)
	many := make([]string, maxVersionRegistryEntries+1)
	for i := range many {
		many[i] = registryEntry("codex", "1.0.0", fmt.Sprintf("%064x", i))
	}
	cases := map[string][]byte{
		"empty-bytes":         nil,
		"malformed":           []byte(`{"schema":`),
		"trailing":            append(registryWire(valid), []byte(`{}`)...),
		"unsupported-schema":  []byte(`{"schema":"axiom-runtime-version-registry/v2","entries":[]}`),
		"missing-schema":      []byte(`{"entries":[]}`),
		"missing-entries":     []byte(`{"schema":"` + VersionRegistrySchema + `"}`),
		"null-entries":        []byte(`{"schema":"` + VersionRegistrySchema + `","entries":null}`),
		"unknown-field":       []byte(`{"schema":"` + VersionRegistrySchema + `","entries":[],"trusted":true}`),
		"unknown-entry-field": registryWire(`{"runtimeId":"codex","version":"1.0.0","executableDigest":"` + a + `","capabilities":["code"]}`),
		"case-alias":          []byte(`{"Schema":"` + VersionRegistrySchema + `","entries":[]}`),
		"case-alias-entry":    registryWire(`{"runtimeID":"codex","version":"1.0.0","executableDigest":"` + a + `"}`),
		"duplicate-field":     []byte(`{"schema":"` + VersionRegistrySchema + `","schema":"` + VersionRegistrySchema + `","entries":[]}`),
		"duplicate-entry":     registryWire(valid, valid),
		"conflicting-version": registryWire(valid, registryEntry("codex", "2.0.0", a)),
		"digest-reused":       registryWire(registryEntry("claude", "1.0.0", a), valid),
		"unordered":           registryWire(registryEntry("codex", "1.0.0", b), valid),
		"unknown-runtime":     registryWire(registryEntry("other", "1.0.0", a)),
		"invalid-runtime":     registryWire(registryEntry("Codex", "1.0.0", a)),
		"missing-version":     registryWire(`{"runtimeId":"codex","executableDigest":"` + a + `"}`),
		"empty-version":       registryWire(registryEntry("codex", "", a)),
		"unsafe-version":      registryWire(registryEntry("codex", "1.0\n", a)),
		"secret-version":      registryWire(registryEntry("codex", "token-1", a)),
		"short-digest":        registryWire(registryEntry("codex", "1.0.0", a[:63])),
		"uppercase-digest":    registryWire(registryEntry("codex", "1.0.0", strings.ToUpper(a))),
		"non-hex-digest":      registryWire(registryEntry("codex", "1.0.0", strings.Repeat("g", 64))),
		"non-string-version":  registryWire(`{"runtimeId":"codex","version":1,"executableDigest":"` + a + `"}`),
		"too-many-entries":    registryWire(many...),
		"oversized":           registryWire(strings.Repeat(" ", maxVersionRegistryBytes)),
		"non-object":          []byte(`[]`),
	}
	for name, wire := range cases {
		if o, err := decodeVersionRegistry(wire); !errors.Is(err, capabilityproof.ErrBlocked) || o.versions != nil {
			t.Errorf("%s: %v %v", name, o.versions, err)
		}
	}
	if _, err := decodeVersionRegistry(registryWire(valid, registryEntry("codex", "1.0.0", b))); err != nil {
		t.Fatalf("canonical order: %v", err)
	}
}

// The observer reads only its registry: a declaration naming a version, a
// Profile or stored Evidence never produces one, and a registered version is
// not a capability or a declaration.
func TestRegistryVersionIsNotDeclarationOrCapability(t *testing.T) {
	ctx := context.Background()
	exe, marker := syntheticExecutable(t, t.TempDir(), "codex")
	digest := identityOf(t, exe)
	d := runtimeprofiletest.CodexDeclaration()
	d.ExecutableDigest = digest
	cfg := runtimeprofiletest.WithDeclaration(runtimeprofiletest.CodexOnly(), "codex", d)
	empty := mustRegistry(t, registryWire())
	look := func(string) (string, error) { return exe, nil }
	if _, err := ObserveEffortApplicability(ctx, look, empty.Version, "codex"); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatalf("declaration-derived version: %v", err)
	}
	registered := mustRegistry(t, registryWire(registryEntry("codex", "synthetic-codex-unrelated", digest)))
	a, err := ObserveEffortApplicability(ctx, look, registered.Version, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtimeprofile.ApplicableEffortDeclaration(decodeV3Fixture(t, cfg), a); !errors.Is(err, runtimeprofile.ErrEffortDeclarationUnproven) {
		t.Fatalf("registry version created a declaration: %v", err)
	}
	assertNotExecuted(t, marker)
}

// The registry observer satisfies the existing BindingSource Version seam
// unchanged; an unregistered identity blocks the binding.
func TestBindingSourceConsumesRegistryVersionObserver(t *testing.T) {
	for _, runtime := range []string{"codex", "claude"} {
		t.Run(runtime, func(t *testing.T) {
			f := newBindingFixture(t, runtime)
			o := mustRegistry(t, registryWire(registryEntry(runtime, f.version, f.identity(t, f.exe))))
			f.source.Version = o.Version
			target := f.target(capabilityproof.ReadOnlyVariant, false)
			stored := f.seed(t, target)
			c, err := f.source.Current(context.Background(), target)
			wantBinding, wantModel := f.expected(t, target)
			if err != nil || c.Binding != wantBinding || c.Model == nil || *c.Model != wantModel || c.EvidenceRef != stored.Ref {
				t.Fatalf("%+v %v", c, err)
			}
			f.source.Version = mustRegistry(t, registryWire()).Version
			if _, err := f.source.Current(context.Background(), target); !errors.Is(err, capabilityproof.ErrBlocked) {
				t.Fatalf("unregistered identity: %v", err)
			}
		})
	}
}

func identityOf(t *testing.T, path string) string {
	t.Helper()
	digest, err := ExecutableIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
