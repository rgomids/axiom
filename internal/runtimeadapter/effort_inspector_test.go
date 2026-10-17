//go:build unix

package runtimeadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rgomids/axiom/internal/executiongraph"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 AXM-22 BI-2/BI-2a: the narrow inspector derives the
// PE-05a EffortSourceDigest from a configured v3 declaration, trusted anchor
// roots and independent observations only. Supported hosts (macOS, Linux)
// provide native symlinks and FIFOs; fixture setup failure fails the test.

type effortInspectorFixture struct {
	base string
	req  EffortSourceRequest
	exe  string
}

func (f *effortInspectorFixture) declaration() *runtimeprofile.EffortDeclaration {
	return &f.req.Configuration.V3.Runtimes[0].EffortDeclarations[0]
}
func (f *effortInspectorFixture) profile() *runtimeprofile.ModelProfile {
	return &f.req.Configuration.V3.ModelProfiles[0]
}
func (f *effortInspectorFixture) root(anchor string) string {
	return map[string]string{"user": f.req.Roots.User, "project": f.req.Roots.Project, "managed": f.req.Roots.Managed}[anchor]
}
func (f *effortInspectorFixture) source(t *testing.T, scope string) string {
	t.Helper()
	for _, s := range f.declaration().Sources {
		if s.Scope == scope {
			return filepath.Join(f.root(s.Anchor), filepath.FromSlash(s.Path))
		}
	}
	t.Fatal("no source for scope", scope)
	return ""
}

// legacy is an independently written PE-05a oracle: the same declaration
// expressed directly as runtimeadapter roots and a ResolvedProfile.
func (f *effortInspectorFixture) legacy() (EffortInspection, runtimeprofile.ResolvedProfile) {
	d, profile := f.declaration(), f.profile()
	hash := strings.Repeat("a", 64)
	p := runtimeprofile.ResolvedProfile{ProfileID: profile.ID, RuntimeID: profile.RuntimeID, Model: profile.Model, Effort: profile.Effort, ProfileDigest: hash, ConfigurationRevision: 1, ConfigurationDigest: hash, ExecutableDigest: f.req.Observed.ExecutableDigest, RuntimeVersion: f.req.Observed.RuntimeVersion, InvocationMode: f.req.Observed.InvocationMode, ObservationRevision: 1, ObservationDigest: hash, EffortSourceDigest: hash, CapabilityProofReference: "synthetic-proof"}
	legacy := EffortDeclaration{RuntimeID: profile.RuntimeID, RuntimeVersion: d.RuntimeVersion, InvocationMode: d.InvocationMode, ExecutableDigest: d.ExecutableDigest, ProofReference: d.DeclarationID, PrecedenceProven: d.PrecedenceEstablished, JSONKeys: d.JSONKeys, EnvironmentKeys: d.EnvironmentKeys}
	for _, s := range d.Sources {
		legacy.Sources = append(legacy.Sources, EffortSource{Scope: s.Scope, Root: f.root(s.Anchor), Relative: s.Path, Format: s.Format, Directory: s.Directory})
	}
	return EffortInspection{Consented: true, Declaration: legacy}, p
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func newEffortInspectorFixture(t *testing.T, id string) *effortInspectorFixture {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &effortInspectorFixture{base: base, exe: filepath.Join(base, "synthetic-executable")}
	writeFixtureFile(t, f.exe, "synthetic identity; never execute\n")
	if err := os.Chmod(f.exe, 0o700); err != nil {
		t.Fatal(err)
	}
	identity, err := ExecutableIdentity(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	f.req.Roots = EffortSourceRoots{User: filepath.Join(base, "home"), Project: filepath.Join(base, "project"), Managed: filepath.Join(base, "managed")}
	for _, root := range []string{f.req.Roots.User, f.req.Roots.Project, f.req.Roots.Managed} {
		if err := os.Mkdir(root, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	d := runtimeprofile.EffortDeclaration{DeclarationID: id + "-effort-sources", DeclarationVersion: runtimeprofile.EffortDeclarationVersion, RuntimeVersion: "synthetic-" + id + "-1", ExecutableDigest: identity, InvocationMode: "noninteractive", PrecedenceEstablished: true, JSONKeys: []string{}, EnvironmentKeys: []string{}}
	argv := []string{f.exe, "exec", "--model", "operator-model"}
	if id == "codex" {
		d.Sources = []runtimeprofile.EffortSourceDeclaration{
			{Scope: "user", Anchor: "user", Path: ".codex/config.toml", Format: "toml"},
			{Scope: "project", Anchor: "project", Path: ".codex/config.toml", Format: "toml"},
			{Scope: "managed", Anchor: "managed", Path: "managed_config.toml", Format: "toml"},
		}
	} else {
		// Synthetic JSON key: no Claude settings key is invented by fixtures.
		d.JSONKeys = []string{"fixtureEffortKey"}
		d.EnvironmentKeys = []string{"CLAUDE_CODE_EFFORT_LEVEL"}
		d.Sources = []runtimeprofile.EffortSourceDeclaration{
			{Scope: "user", Anchor: "user", Path: ".claude/settings.json", Format: "json"},
			{Scope: "project", Anchor: "project", Path: ".claude/settings.json", Format: "json"},
			{Scope: "project-local", Anchor: "project", Path: ".claude/settings.local.json", Format: "json"},
			{Scope: "managed", Anchor: "managed", Path: "managed-settings.json", Format: "json"},
			{Scope: "managed-drop-in", Anchor: "managed", Path: "managed-settings.d", Format: "json", Directory: true},
		}
		argv[1] = "--print"
	}
	cfg := runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 1,
		Runtimes:      []runtimeprofile.RuntimeV3{{ID: id, Adapter: id, Enabled: true, AllowlistedProfileIDs: []string{"test-profile"}, EffortDeclarations: []runtimeprofile.EffortDeclaration{d}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "test-profile", RuntimeID: id, Model: "operator-model", Effort: runtimeprofile.Effort{Mode: runtimeprofile.EffortRuntimeDefault, Value: "default"}, Capabilities: []string{"noninteractive"}}},
	}
	f.req.Consented = true
	f.req.Configuration = decodeV3Fixture(t, cfg)
	f.req.ProfileID = "test-profile"
	f.req.Observed = runtimeprofile.EffortApplicability{RuntimeID: id, RuntimeVersion: d.RuntimeVersion, ExecutableDigest: identity, InvocationMode: "noninteractive"}
	f.req.Invocation = executiongraph.Invocation{RuntimeID: id, Argv: argv, CWD: f.req.Roots.Project, Env: []string{"PATH=fixture"}}
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
	return f
}

// decodeV3Fixture reads the fixture through the approved version-aware read
// path (DecodeVersioned), never through a production writer.
func decodeV3Fixture(t *testing.T, cfg runtimeprofile.ConfigurationV3) runtimeprofile.VersionedConfiguration {
	t.Helper()
	wire, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	v, err := runtimeprofile.DecodeVersioned(wire)
	if err != nil || v.FormatVersion != runtimeprofile.FormatVersionV3 || v.V3 == nil {
		t.Fatalf("v3 fixture not readable through DecodeVersioned: %v", err)
	}
	return v
}

var canonicalDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

func compareWithPE05a(t *testing.T, f *effortInspectorFixture) (string, error) {
	t.Helper()
	legacy, p := f.legacy()
	gotWire, gotErr := inspectEffortProjection(context.Background(), f.req)
	wantWire, wantErr := legacy.effortProjection(context.Background(), p, f.req.Invocation)
	if !errors.Is(gotErr, wantErr) || !errors.Is(wantErr, gotErr) {
		t.Fatalf("refusal diverges from PE-05a: inspector %v, CheckEffortSources %v", gotErr, wantErr)
	}
	if !bytes.Equal(gotWire, wantWire) {
		t.Fatalf("canonical projection diverges:\n%s\n%s", gotWire, wantWire)
	}
	got, gotErr := InspectEffortSources(context.Background(), f.req)
	want, wantErr := legacy.CheckEffortSources(context.Background(), p, f.req.Invocation)
	if got != want || !errors.Is(gotErr, wantErr) || !errors.Is(wantErr, gotErr) {
		t.Fatalf("digest diverges: %q %v vs %q %v", got, gotErr, want, wantErr)
	}
	if gotErr == nil && (!canonicalDigest.MatchString(got) || got != effortHash(wantWire)) {
		t.Fatalf("digest representation %q", got)
	}
	return got, gotErr
}

func TestInspectEffortSourcesMatchesPE05aDigest(t *testing.T) {
	variants := []struct {
		name  string
		apply func(*testing.T, *effortInspectorFixture)
	}{
		{"baseline", func(*testing.T, *effortInspectorFixture) {}},
		{"absent-user-source", func(t *testing.T, f *effortInspectorFixture) {
			if err := os.Remove(f.source(t, "user")); err != nil {
				t.Fatal(err)
			}
		}},
		{"different-non-selector-bytes", func(t *testing.T, f *effortInspectorFixture) {
			content := "# other\nunrelated = true\n"
			if f.req.Observed.RuntimeID == "claude" {
				content = `{"unrelated":true}`
			}
			writeFixtureFile(t, f.source(t, "project"), content)
		}},
		{"explicit-profile-effort", func(t *testing.T, f *effortInspectorFixture) {
			f.profile().Effort = runtimeprofile.Effort{Mode: runtimeprofile.EffortExplicit, Value: "operator-token"}
			if f.req.Observed.RuntimeID == "codex" {
				f.req.Invocation.Argv = append(f.req.Invocation.Argv, "--config", "model_reasoning_effort=operator-token")
			} else {
				f.req.Invocation.Env = append(f.req.Invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=operator-token")
			}
		}},
		{"other-declaration-id", func(*testing.T, *effortInspectorFixture) {}},
	}
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			seen := map[string]string{}
			for _, v := range variants {
				f := newEffortInspectorFixture(t, id)
				v.apply(t, f)
				if v.name == "other-declaration-id" {
					f.declaration().DeclarationID = id + "-other-sources"
				}
				digest, err := compareWithPE05a(t, f)
				if err != nil {
					t.Fatalf("%s: %v", v.name, err)
				}
				// Identical inputs give identical digests.
				if again, err := InspectEffortSources(context.Background(), f.req); err != nil || again != digest {
					t.Fatalf("%s: not deterministic: %v", v.name, err)
				}
				seen[v.name] = digest
			}
			// Fresh fixtures differ in object identity, so every case must be
			// distinct; within one fixture each meaningful difference must be too.
			f := newEffortInspectorFixture(t, id)
			before, _ := compareWithPE05a(t, f)
			for name, change := range map[string]func(){
				"source-content-drift": func() {
					writeFixtureFile(t, f.source(t, "managed"), map[bool]string{true: "# drift\n", false: `{"drift":1}`}[id == "codex"])
				},
				"declaration-id": func() { f.declaration().DeclarationID = id + "-renamed" },
			} {
				change()
				after, err := compareWithPE05a(t, f)
				if err != nil || after == before {
					t.Fatalf("%s not bound into the digest: %v", name, err)
				}
				before = after
			}
			if id == "claude" {
				writeFixtureFile(t, filepath.Join(f.source(t, "managed-drop-in"), "20-extra.json"), "{}")
				after, err := compareWithPE05a(t, f)
				if err != nil || after == before {
					t.Fatal("drop-in membership not bound:", err)
				}
			}
			if len(seen) != len(variants) {
				t.Fatal("variant digests collided")
			}
		})
	}
}

func TestInspectEffortSourcesRefusalsMatchPE05a(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			cases := map[string]struct {
				want  error
				apply func(*testing.T, *effortInspectorFixture)
			}{
				"selector-in-source": {ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
					writeFixtureFile(t, f.source(t, "user"), map[bool]string{true: `model_reasoning_effort = "private-canary"`, false: `{"fixtureEffortKey":"private-canary"}`}[id == "codex"])
				}},
				"inherited-environment-override": {ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
					if id == "codex" {
						f.declaration().EnvironmentKeys = []string{"CODEX_FIXTURE_EFFORT"}
						f.req.Configuration = decodeV3Fixture(t, *f.req.Configuration.V3)
						f.req.Invocation.Env = append(f.req.Invocation.Env, "CODEX_FIXTURE_EFFORT=high")
					} else {
						f.req.Invocation.Env = append(f.req.Invocation.Env, "CLAUDE_CODE_EFFORT_LEVEL=high")
					}
				}},
				"unsupported-source-grammar": {ErrEffortConfigurationUnproven, func(t *testing.T, f *effortInspectorFixture) {
					writeFixtureFile(t, f.source(t, "project"), map[bool]string{true: "[[array]]\n", false: `[]`}[id == "codex"])
				}},
				"argv-effort-override": {ErrEffortOverrideConflict, func(t *testing.T, f *effortInspectorFixture) {
					f.req.Invocation.Argv = append(f.req.Invocation.Argv, "--effort", "high")
				}},
			}
			for name, c := range cases {
				f := newEffortInspectorFixture(t, id)
				c.apply(t, f)
				if _, err := compareWithPE05a(t, f); !errors.Is(err, c.want) || strings.Contains(err.Error(), "private-canary") {
					t.Fatalf("%s: %v", name, err)
				}
			}
		})
	}
}

// The request carries no ProfileProof, ResolvedProfile, capability status or
// Evidence, and the result is only a digest: nothing is promoted to proven.
func TestInspectEffortSourcesIndependentOfCapabilityProof(t *testing.T) {
	forbidden := []reflect.Type{reflect.TypeOf(runtimeprofile.ProfileProof{}), reflect.TypeOf(runtimeprofile.ResolvedProfile{}), reflect.TypeOf(runtimeprofile.Observation{}), reflect.TypeOf(runtimeprofile.CapabilityStatus(""))}
	var walk func(reflect.Type, string)
	walk = func(ty reflect.Type, at string) {
		for ty.Kind() == reflect.Pointer || ty.Kind() == reflect.Slice {
			ty = ty.Elem()
		}
		for _, bad := range forbidden {
			if ty == bad {
				t.Fatalf("%s depends on %s", at, bad)
			}
		}
		if strings.Contains(ty.PkgPath(), "capabilityproof") {
			t.Fatalf("%s depends on capability Evidence", at)
		}
		if ty.Kind() == reflect.Struct {
			for i := 0; i < ty.NumField(); i++ {
				walk(ty.Field(i).Type, at+"."+ty.Field(i).Name)
			}
		}
	}
	walk(reflect.TypeOf(EffortSourceRequest{}), "EffortSourceRequest")
	if out := reflect.TypeOf(InspectEffortSources); out.NumOut() != 2 || out.Out(0).Kind() != reflect.String {
		t.Fatal("inspector must return only a digest and an error")
	}
	// A declarationId equal to a would-be proof reference identifies the
	// declaration only; inspection succeeds with no proof present anywhere.
	f := newEffortInspectorFixture(t, "codex")
	f.declaration().DeclarationID = "synthetic-proof"
	if _, err := compareWithPE05a(t, f); err != nil {
		t.Fatal(err)
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		entry := info.Mode().String() + info.ModTime().String()
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry += string(b)
		}
		out[path] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestInspectEffortSourcesIsReadOnly(t *testing.T) {
	for _, id := range []string{"codex", "claude"} {
		f := newEffortInspectorFixture(t, id)
		if err := os.Remove(f.source(t, "user")); err != nil {
			t.Fatal(err)
		}
		before := snapshotTree(t, f.base)
		if _, err := InspectEffortSources(context.Background(), f.req); err != nil {
			t.Fatal(err)
		}
		after := snapshotTree(t, f.base)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("inspection created or mutated a source")
		}
	}
}

func TestInspectEffortSourcesFailsClosed(t *testing.T) {
	other := strings.Repeat("b", 64)
	type mutation func(*testing.T, *effortInspectorFixture)
	reconfigure := func(edit func(*runtimeprofile.EffortDeclaration)) mutation {
		return func(t *testing.T, f *effortInspectorFixture) {
			// A struct edit bypasses decoding; selection revalidates the v3 rules.
			edit(f.declaration())
		}
	}
	cases := map[string]mutation{
		"unconsented": func(_ *testing.T, f *effortInspectorFixture) { f.req.Consented = false },
		"missing-declaration": func(_ *testing.T, f *effortInspectorFixture) {
			f.req.Configuration.V3.Runtimes[0].EffortDeclarations = nil
		},
		"multiple-applicable": func(_ *testing.T, f *effortInspectorFixture) {
			r := &f.req.Configuration.V3.Runtimes[0]
			twin := *f.declaration()
			twin.DeclarationID = "twin-effort-sources"
			r.EffortDeclarations = append(r.EffortDeclarations, twin)
		},
		"wrong-observed-executable": func(_ *testing.T, f *effortInspectorFixture) { f.req.Observed.ExecutableDigest = other },
		"declaration-for-other-executable": func(_ *testing.T, f *effortInspectorFixture) {
			f.declaration().ExecutableDigest = other
			f.req.Observed.ExecutableDigest = other
		},
		"wrong-runtime-version":      func(_ *testing.T, f *effortInspectorFixture) { f.req.Observed.RuntimeVersion = "synthetic-other-2" },
		"wrong-runtime-identity":     func(_ *testing.T, f *effortInspectorFixture) { f.req.Observed.RuntimeID = "other" },
		"invocation-runtime-drift":   func(_ *testing.T, f *effortInspectorFixture) { f.req.Invocation.RuntimeID = "other" },
		"wrong-invocation-mode":      func(_ *testing.T, f *effortInspectorFixture) { f.req.Observed.InvocationMode = "interactive" },
		"argv-mode-drift":            func(_ *testing.T, f *effortInspectorFixture) { f.req.Invocation.Argv[1] = "--interactive" },
		"argv-model-drift":           func(_ *testing.T, f *effortInspectorFixture) { f.req.Invocation.Argv[3] = "other-model" },
		"wrong-declaration-version":  reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.DeclarationVersion = 2 }),
		"traversal-path":             reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources[0].Path = "../outside/config.toml" }),
		"absolute-path":              reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources[0].Path = "/etc/config.toml" }),
		"home-relative-path":         reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources[0].Path = "~/config.toml" }),
		"unsupported-anchor":         reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources[0].Anchor = "home" }),
		"anchor-scope-mismatch":      reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources[0].Anchor = "managed" }),
		"unsupported-format":         reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources[0].Format = "yaml" }),
		"unsupported-file":           reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources[0].Path = "credentials.json" }),
		"unsupported-key":            reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.EnvironmentKeys = []string{"BAD KEY"} }),
		"precedence-not-established": reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.PrecedenceEstablished = false }),
		"ambiguous-scope": reconfigure(func(d *runtimeprofile.EffortDeclaration) {
			d.Sources = append(d.Sources, d.Sources[0])
			d.Sources[len(d.Sources)-1].Path = "other/" + d.Sources[0].Path
		}),
		"missing-required-scope": reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.Sources = d.Sources[1:] }),
		"unknown-profile":        func(_ *testing.T, f *effortInspectorFixture) { f.req.ProfileID = "other-profile" },
		"missing-user-root":      func(_ *testing.T, f *effortInspectorFixture) { f.req.Roots.User = "" },
		"relative-root":          func(_ *testing.T, f *effortInspectorFixture) { f.req.Roots.Project = "project" },
		"unclean-root":           func(_ *testing.T, f *effortInspectorFixture) { f.req.Roots.Project += "/../project" },
		// PM-13: only a missing managed root may be an absence fact
		// (TestTrustedEffortAnchorAbsentManaged); user and project must exist.
		"absent-user-root": func(t *testing.T, f *effortInspectorFixture) {
			if err := os.RemoveAll(f.req.Roots.User); err != nil {
				t.Fatal(err)
			}
		},
		"absent-project-root": func(t *testing.T, f *effortInspectorFixture) {
			if err := os.RemoveAll(f.req.Roots.Project); err != nil {
				t.Fatal(err)
			}
		},
		"root-is-file": func(t *testing.T, f *effortInspectorFixture) {
			f.req.Roots.Managed = f.exe
		},
		"symlinked-root": func(t *testing.T, f *effortInspectorFixture) {
			link := filepath.Join(f.base, "project-link")
			if err := os.Symlink(f.req.Roots.Project, link); err != nil {
				t.Fatal("symlink setup failed on a supported host:", err)
			}
			f.req.Roots.Project = link
		},
		"symlink-escape": func(t *testing.T, f *effortInspectorFixture) {
			outside := filepath.Join(f.base, "outside")
			writeFixtureFile(t, filepath.Join(outside, "settings.json"), "{}")
			writeFixtureFile(t, filepath.Join(outside, "config.toml"), "# no effort selector\n")
			dir := filepath.Dir(f.source(t, "user"))
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, dir); err != nil {
				t.Fatal("symlink setup failed on a supported host:", err)
			}
		},
		"group-writable-root": func(t *testing.T, f *effortInspectorFixture) {
			chmodFixture(t, f.req.Roots.User, 0o770)
		},
		"sticky-world-writable-root": func(t *testing.T, f *effortInspectorFixture) {
			chmodFixture(t, f.req.Roots.User, 0o1777)
		},
		"unsafe-ancestor": func(t *testing.T, f *effortInspectorFixture) {
			chmodFixture(t, f.base, 0o777)
		},
		"special-file-source": func(t *testing.T, f *effortInspectorFixture) {
			path := f.source(t, "project")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Fatal("fifo setup failed on a supported host:", err)
			}
		},
		"directory-where-file-required": func(t *testing.T, f *effortInspectorFixture) {
			path := f.source(t, "project")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		},
	}
	claudeOnly := map[string]mutation{
		"missing-required-environment-key": reconfigure(func(d *runtimeprofile.EffortDeclaration) { d.EnvironmentKeys = []string{"OTHER_EFFORT_KEY"} }),
		"missing-explicit-environment-value": func(_ *testing.T, f *effortInspectorFixture) {
			f.profile().Effort = runtimeprofile.Effort{Mode: runtimeprofile.EffortExplicit, Value: "operator-token"}
		},
		"drop-in-non-json": func(t *testing.T, f *effortInspectorFixture) {
			writeFixtureFile(t, filepath.Join(f.source(t, "managed-drop-in"), "notes.txt"), "")
		},
	}
	for _, id := range []string{"codex", "claude"} {
		t.Run(id, func(t *testing.T) {
			run := func(name string, m mutation) {
				f := newEffortInspectorFixture(t, id)
				if _, err := InspectEffortSources(context.Background(), f.req); err != nil {
					t.Fatalf("%s: baseline refused: %v", name, err)
				}
				m(t, f)
				if digest, err := InspectEffortSources(context.Background(), f.req); !errors.Is(err, ErrEffortConfigurationUnproven) || digest != "" {
					t.Fatalf("%s: accepted or wrong refusal: %q %v", name, digest, err)
				}
			}
			for name, m := range cases {
				run(name, m)
			}
			if id == "claude" {
				for name, m := range claudeOnly {
					run(name, m)
				}
			}
		})
	}
}

func chmodFixture(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if mode&0o1000 != 0 {
		mode = mode&0o777 | os.ModeSticky
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o700) })
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm()&0o022 == 0 {
		t.Fatal("permission setup not effective:", err)
	}
}

// History and vendor versions never create or infer a declaration.
func TestInspectEffortSourcesNoInferredDeclaration(t *testing.T) {
	f := newEffortInspectorFixture(t, "codex")
	v3 := *f.req.Configuration.V3
	v2 := runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersionV2, Revision: v3.Revision, ModelProfiles: v3.ModelProfiles}
	for _, r := range v3.Runtimes {
		v2.Runtimes = append(v2.Runtimes, runtimeprofile.Runtime{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs})
	}
	if err := runtimeprofile.Validate(v2); err != nil {
		t.Fatal(err)
	}
	f.req.Configuration = runtimeprofile.VersionedConfiguration{FormatVersion: runtimeprofile.FormatVersionV2, Configuration: &v2}
	if _, err := InspectEffortSources(context.Background(), f.req); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("historical v2 record gained a declaration:", err)
	}
	// A vendor-looking version with no declaration is still unproven.
	f = newEffortInspectorFixture(t, "codex")
	f.req.Configuration.V3.Runtimes[0].EffortDeclarations = nil
	f.req.Observed.RuntimeVersion = "0.159.1"
	if _, err := InspectEffortSources(context.Background(), f.req); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("declaration inferred from a vendor version:", err)
	}
	f.req.Configuration = runtimeprofile.VersionedConfiguration{}
	if _, err := InspectEffortSources(context.Background(), f.req); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("empty configuration accepted:", err)
	}
}

func TestInspectEffortSourcesContextBoundary(t *testing.T) {
	f := newEffortInspectorFixture(t, "claude")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := InspectEffortSources(cancelled, f.req); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("cancelled context accepted:", err)
	}
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	if _, err := InspectEffortSources(expired, f.req); !errors.Is(err, ErrEffortConfigurationUnproven) {
		t.Fatal("expired deadline accepted:", err)
	}
}

// A root replaced at its pathname is a different identity; the inspector
// compares the root binding before and after reading sources.
func TestTrustedEffortRootIdentity(t *testing.T) {
	f := newEffortInspectorFixture(t, "codex")
	before, err := trustedEffortRoot(f.req.Roots.User)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := trustedEffortRoot(f.req.Roots.User); err != nil || again != before {
		t.Fatal("root binding not stable:", err)
	}
	if err := os.Rename(f.req.Roots.User, f.req.Roots.User+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(f.req.Roots.User, 0o700); err != nil {
		t.Fatal(err)
	}
	if after, err := trustedEffortRoot(f.req.Roots.User); err != nil || after == before {
		t.Fatal("replaced root kept its identity:", err)
	}
}
