package runtimeprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

const v3Executable = "1111111111111111111111111111111111111111111111111111111111111111"

func testCodexDeclaration() EffortDeclaration {
	return EffortDeclaration{DeclarationID: "codex-effort-sources", DeclarationVersion: EffortDeclarationVersion, RuntimeVersion: "synthetic-codex-1", ExecutableDigest: v3Executable, InvocationMode: "noninteractive", PrecedenceEstablished: true, JSONKeys: []string{}, EnvironmentKeys: []string{}, Sources: []EffortSourceDeclaration{
		{Scope: "user", Anchor: EffortAnchorUser, Path: ".codex/config.toml", Format: "toml"},
		{Scope: "project", Anchor: EffortAnchorProject, Path: ".codex/config.toml", Format: "toml"},
		{Scope: "managed", Anchor: EffortAnchorManaged, Path: "managed_config.toml", Format: "toml"},
	}}
}

// Synthetic JSON key: no Claude settings key is invented by these fixtures.
func testClaudeDeclaration() EffortDeclaration {
	return EffortDeclaration{DeclarationID: "claude-effort-sources", DeclarationVersion: EffortDeclarationVersion, RuntimeVersion: "synthetic-claude-1", ExecutableDigest: strings.Repeat("2", 64), InvocationMode: "noninteractive", PrecedenceEstablished: true, JSONKeys: []string{"fixtureEffortKey"}, EnvironmentKeys: []string{"CLAUDE_CODE_EFFORT_LEVEL"}, Sources: []EffortSourceDeclaration{
		{Scope: "user", Anchor: EffortAnchorUser, Path: ".claude/settings.json", Format: "json"},
		{Scope: "project", Anchor: EffortAnchorProject, Path: ".claude/settings.json", Format: "json"},
		{Scope: "project-local", Anchor: EffortAnchorProject, Path: ".claude/settings.local.json", Format: "json"},
		{Scope: "managed", Anchor: EffortAnchorManaged, Path: "managed-settings.json", Format: "json"},
		{Scope: "managed-drop-in", Anchor: EffortAnchorManaged, Path: "managed-settings.d", Format: "json", Directory: true},
	}}
}

func testV3() ConfigurationV3 {
	v2 := testV2()
	c := ConfigurationV3{FormatVersion: FormatVersionV3, Revision: v2.Revision, ModelProfiles: v2.ModelProfiles, Preferences: v2.Preferences}
	for _, r := range v2.Runtimes {
		c.Runtimes = append(c.Runtimes, RuntimeV3{ID: r.ID, Adapter: r.Adapter, Enabled: r.Enabled, AllowlistedProfileIDs: r.AllowlistedProfileIDs})
	}
	c.Runtimes[0].EffortDeclarations = []EffortDeclaration{testCodexDeclaration()}
	c.Runtimes[1].EffortDeclarations = []EffortDeclaration{testClaudeDeclaration()}
	return c
}

func encodeV3(t *testing.T, c ConfigurationV3) []byte {
	t.Helper()
	w, err := encodeConfigurationV3(c)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// Golden values recorded from the unmodified format-2 codec at 825b88c.
func TestV3HistoricalV2BytesDigestAndUnprovenDeclaration(t *testing.T) {
	w, err := Encode(testV2())
	sum := sha256.Sum256(w)
	if err != nil || hex.EncodeToString(sum[:]) != "08a005eafe66a7b90127c33c9695f5fb51baccba6e87b82a2abcecd439cf1b53" {
		t.Fatalf("historical v2 bytes changed: %v", err)
	}
	if d, err := Digest(testV2()); err != nil || d != "8a7f9a6ee81988d6f37ddf48282f166e6d7ba73cc5c5c3776cd178a7f053d19e" {
		t.Fatalf("historical v2 digest changed: %v", err)
	}
	for _, cfg := range []Configuration{testConfiguration(), testV2()} {
		wire, _ := Encode(cfg)
		v, err := DecodeVersioned(wire)
		if err != nil || v.FormatVersion != cfg.FormatVersion || v.V3 != nil || v.Configuration == nil || !reflect.DeepEqual(*v.Configuration, cfg) {
			t.Fatalf("historical dispatch %d: %v", cfg.FormatVersion, err)
		}
		again, _ := Encode(*v.Configuration)
		if string(again) != string(wire) || strings.Contains(string(wire), "effortDeclarations") {
			t.Fatal("historical record rewritten or given a synthetic declaration")
		}
		// D3: valid history is not corruption; its effort configuration is unproven.
		if _, err := ApplicableEffortDeclaration(v, EffortApplicability{"codex", "synthetic-codex-1", v3Executable, "noninteractive"}); !errors.Is(err, ErrEffortDeclarationUnproven) {
			t.Fatal("historical record gained a declaration")
		}
	}
}

func TestV3CanonicalCodecAndVersionDispatch(t *testing.T) {
	c := testV3()
	w := encodeV3(t, c)
	golden, err := os.ReadFile("testdata/configuration-v3.json")
	if err != nil || string(golden) != string(w) {
		t.Fatalf("canonical v3 bytes changed: %v\n%s", err, w)
	}
	v, err := DecodeVersioned(w)
	if err != nil || v.FormatVersion != FormatVersionV3 || v.Configuration != nil || v.V3 == nil {
		t.Fatalf("dispatch: %v", err)
	}
	if !reflect.DeepEqual(*v.V3, canonicalV3(c)) || string(encodeV3(t, *v.V3)) != string(w) {
		t.Fatal("round trip")
	}
	// Input order never changes canonical bytes.
	p := testV3()
	p.Runtimes[0], p.Runtimes[1] = p.Runtimes[1], p.Runtimes[0]
	p.ModelProfiles[0], p.ModelProfiles[1] = p.ModelProfiles[1], p.ModelProfiles[0]
	s := p.Runtimes[0].EffortDeclarations[0].Sources
	s[0], s[4] = s[4], s[0]
	p.Runtimes[1].EffortDeclarations[0].JSONKeys = nil
	if string(encodeV3(t, p)) != string(w) {
		t.Fatal("non-deterministic v3 encoding")
	}
	// ModelProfile keeps effort ownership (ADR-0024): identical Profile identity.
	for i, profile := range v.V3.ModelProfiles {
		want := canonicalV3(c).ModelProfiles[i]
		got, _ := ProfileDigest(profile)
		expected, _ := ProfileDigest(want)
		if profile.Effort != want.Effort || got != expected {
			t.Fatal("v3 changed Profile effort intent")
		}
	}
	// D7: declarationId maps 1:1 to the existing ProofReference contract.
	refs := map[string]string{}
	for _, r := range v.V3.Runtimes {
		for _, d := range r.EffortDeclarations {
			if d.ProofReference() != d.DeclarationID || refs[d.ProofReference()] != "" {
				t.Fatal("declaration/ProofReference mapping is not 1:1")
			}
			refs[d.ProofReference()] = r.ID
		}
	}
	if len(refs) != 2 {
		t.Fatal(refs)
	}
	for _, version := range []string{"4", "99"} {
		if _, err := DecodeVersioned([]byte(`{"formatVersion":` + version + `,"revision":1}`)); !errors.Is(err, ErrUnsupportedConfigurationFormat) {
			t.Fatal("unsupported version accepted", version, err)
		}
	}
	for _, wire := range []string{`{"formatVersion":0}`, `{"revision":1}`, `{"formatVersion":"3"}`, `[]`, ``} {
		if _, err := DecodeVersioned([]byte(wire)); !errors.Is(err, ErrInvalidConfiguration) || errors.Is(err, ErrUnsupportedConfigurationFormat) {
			t.Fatal("malformed version accepted", wire, err)
		}
	}
}

// D8: the format-1/2 codec refuses v3 explicitly and never downgrades it.
func TestV3RefusedByV2OnlyCodec(t *testing.T) {
	w := encodeV3(t, testV3())
	cfg, err := Decode(w)
	if !errors.Is(err, ErrUnsupportedConfigurationFormat) || !errors.Is(err, ErrInvalidConfiguration) || !reflect.DeepEqual(cfg, Configuration{}) {
		t.Fatalf("v2 reader accepted v3: %v", err)
	}
	downgrade := strings.Replace(string(w), `"formatVersion":3`, `"formatVersion":2`, 1)
	if _, err := Decode([]byte(downgrade)); err == nil {
		t.Fatal("v3 declarations read as v2")
	}
	projected := projectV2(testV3())
	projected.FormatVersion = FormatVersionV3
	if err := Validate(projected); !errors.Is(err, ErrUnsupportedConfigurationFormat) {
		t.Fatal(err)
	}
	if _, err := Encode(projected); !errors.Is(err, ErrUnsupportedConfigurationFormat) {
		t.Fatal("v2 writer emitted format 3")
	}
	if _, err := Digest(projected); !errors.Is(err, ErrUnsupportedConfigurationFormat) {
		t.Fatal(err)
	}
	// A malformed v2 record keeps its historical classification.
	if _, err := Decode([]byte(`{"formatVersion":2,"revision":1,"runtimes":[]}`)); !errors.Is(err, ErrInvalidConfiguration) || errors.Is(err, ErrUnsupportedConfigurationFormat) {
		t.Fatal(err)
	}
}

func TestV3EffortDeclarationApplicability(t *testing.T) {
	w := encodeV3(t, testV3())
	v, err := DecodeVersioned(w)
	if err != nil {
		t.Fatal(err)
	}
	match := EffortApplicability{"codex", "synthetic-codex-1", v3Executable, "noninteractive"}
	d, err := ApplicableEffortDeclaration(v, match)
	if err != nil || d.DeclarationID != "codex-effort-sources" {
		t.Fatal(d, err)
	}
	for name, a := range map[string]EffortApplicability{
		"runtime":    {"claude", match.RuntimeVersion, match.ExecutableDigest, match.InvocationMode},
		"executable": {match.RuntimeID, match.RuntimeVersion, strings.Repeat("3", 64), match.InvocationMode},
		"version":    {match.RuntimeID, "synthetic-codex-2", match.ExecutableDigest, match.InvocationMode},
		"mode":       {match.RuntimeID, match.RuntimeVersion, match.ExecutableDigest, "interactive"},
		"unknown":    {"other", match.RuntimeVersion, match.ExecutableDigest, match.InvocationMode},
	} {
		if _, err := ApplicableEffortDeclaration(v, a); !errors.Is(err, ErrEffortDeclarationUnproven) {
			t.Fatal("inapplicable declaration selected:", name)
		}
	}
	// A vendor CLI version never creates a declaration.
	none := testV3()
	none.Runtimes[0].EffortDeclarations = nil
	nv, err := DecodeVersioned(encodeV3(t, none))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplicableEffortDeclaration(nv, match); !errors.Is(err, ErrEffortDeclarationUnproven) {
		t.Fatal("declaration synthesized from version")
	}
}

// A declaration carries no capability status and cannot satisfy a proof.
func TestV3DeclarationIsNeverCapabilityProof(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(EffortDeclaration{}), reflect.TypeOf(EffortSourceDeclaration{})} {
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i).Type
			if f == reflect.TypeOf(CapabilityStatus("")) || f.Kind() == reflect.Map {
				t.Fatal("declaration carries capability state:", typ.Field(i).Name)
			}
		}
	}
	c := testV3()
	cfg := projectV2(c)
	d := c.Runtimes[0].EffortDeclarations[0]
	o := testObservation("codex", "codex", true)
	o.ExecutableDigest = d.ExecutableDigest
	o.Version = d.RuntimeVersion
	// Same reference string, but only a declared (not proven) effort: blocked.
	o.ProfileProofs = []ProfileProof{{RuntimeID: "codex", Model: cfg.ModelProfiles[0].Model, Effort: cfg.ModelProfiles[0].Effort, ExecutableDigest: o.ExecutableDigest, RuntimeVersion: o.Version, InvocationMode: "noninteractive", ObservationRevision: o.Revision, Reference: d.ProofReference(), EffortSourceDigest: strings.Repeat("a", 64), Authentication: CapabilityProven, ModelAvailability: CapabilityProven, EffortSupport: CapabilityDeclared, NoOverride: CapabilityProven}}
	if code := ProfileReadiness(cfg, cfg.ModelProfiles[0], o); code != "effort_unproven" {
		t.Fatal("declarationId proved effort:", code)
	}
}

func TestV3StrictDeclarationValidation(t *testing.T) {
	type mutation func(*ConfigurationV3)
	codex := func(f func(*EffortDeclaration)) mutation {
		return func(c *ConfigurationV3) { f(&c.Runtimes[0].EffortDeclarations[0]) }
	}
	claude := func(f func(*EffortDeclaration)) mutation {
		return func(c *ConfigurationV3) { f(&c.Runtimes[1].EffortDeclarations[0]) }
	}
	keys := func(n int) []string {
		out := []string{}
		for i := 0; i < n; i++ {
			out = append(out, "KEY_"+strings.Repeat("A", i+1))
		}
		return out
	}
	cases := map[string]mutation{
		"missing declarationId":    codex(func(d *EffortDeclaration) { d.DeclarationID = "" }),
		"unsafe declarationId":     codex(func(d *EffortDeclaration) { d.DeclarationID = "Codex/Sources" }),
		"oversized declarationId":  codex(func(d *EffortDeclaration) { d.DeclarationID = strings.Repeat("a", 65) }),
		"duplicate declarationId":  claude(func(d *EffortDeclaration) { d.DeclarationID = "codex-effort-sources" }),
		"declaration version 0":    codex(func(d *EffortDeclaration) { d.DeclarationVersion = 0 }),
		"declaration version 2":    codex(func(d *EffortDeclaration) { d.DeclarationVersion = 2 }),
		"runtime version":          codex(func(d *EffortDeclaration) { d.RuntimeVersion = "" }),
		"secret runtime version":   codex(func(d *EffortDeclaration) { d.RuntimeVersion = "token=synthetic" }),
		"executable identity":      codex(func(d *EffortDeclaration) { d.ExecutableDigest = "/usr/local/bin/codex" }),
		"invocation mode":          codex(func(d *EffortDeclaration) { d.InvocationMode = "interactive" }),
		"precedence not set":       codex(func(d *EffortDeclaration) { d.PrecedenceEstablished = false }),
		"ambiguous precedence":     codex(func(d *EffortDeclaration) { d.Sources[1] = d.Sources[0] }),
		"incomplete scopes":        codex(func(d *EffortDeclaration) { d.Sources = d.Sources[:2] }),
		"no sources":               codex(func(d *EffortDeclaration) { d.Sources = nil }),
		"extra claude scope":       codex(func(d *EffortDeclaration) { d.Sources = append(d.Sources, testClaudeDeclaration().Sources[2]) }),
		"unknown anchor":           codex(func(d *EffortDeclaration) { d.Sources[0].Anchor = "home" }),
		"contradictory anchor":     codex(func(d *EffortDeclaration) { d.Sources[0].Anchor = EffortAnchorProject }),
		"unknown scope":            codex(func(d *EffortDeclaration) { d.Sources[0].Scope = "system" }),
		"wrong runtime format":     codex(func(d *EffortDeclaration) { d.Sources[0].Format = "json" }),
		"unrecognized file":        claude(func(d *EffortDeclaration) { d.Sources[0].Path = ".claude/credentials.json" }),
		"codex directory":          codex(func(d *EffortDeclaration) { d.Sources[2].Directory = true }),
		"drop-in not directory":    claude(func(d *EffortDeclaration) { d.Sources[4].Directory = false }),
		"codex json keys":          codex(func(d *EffortDeclaration) { d.JSONKeys = []string{"fixtureEffortKey"} }),
		"claude without json keys": claude(func(d *EffortDeclaration) { d.JSONKeys = nil }),
		"claude without env key":   claude(func(d *EffortDeclaration) { d.EnvironmentKeys = []string{"OTHER_KEY"} }),
		"invalid key":              claude(func(d *EffortDeclaration) { d.JSONKeys = []string{"effort.level"} }),
		"duplicate env key": claude(func(d *EffortDeclaration) {
			d.EnvironmentKeys = []string{"CLAUDE_CODE_EFFORT_LEVEL", "claude_code_effort_level"}
		}),
		"too many keys": claude(func(d *EffortDeclaration) { d.JSONKeys = keys(17) }),
		"oversized key": claude(func(d *EffortDeclaration) { d.JSONKeys = []string{strings.Repeat("k", 129)} }),
		"contradictory declaration": func(c *ConfigurationV3) {
			d := testCodexDeclaration()
			d.DeclarationID = "codex-second"
			c.Runtimes[0].EffortDeclarations = append(c.Runtimes[0].EffortDeclarations, d)
		},
		"too many declarations": func(c *ConfigurationV3) {
			for i := 0; i < maxEffortDeclarations; i++ {
				d := testCodexDeclaration()
				d.DeclarationID = "codex-" + strings.Repeat("x", i+1)
				d.RuntimeVersion = "synthetic-" + strings.Repeat("x", i+1)
				c.Runtimes[0].EffortDeclarations = append(c.Runtimes[0].EffortDeclarations, d)
			}
		},
		"wrong runtime identity": func(c *ConfigurationV3) {
			c.Runtimes[0].EffortDeclarations = []EffortDeclaration{testClaudeDeclaration()}
			c.Runtimes[1].EffortDeclarations = nil
		},
		"v2 profile rule": func(c *ConfigurationV3) { c.ModelProfiles[0].Effort = Effort{} },
	}
	for name, mutate := range cases {
		c := testV3()
		mutate(&c)
		if ValidateV3(c) == nil {
			t.Fatal("invalid declaration accepted:", name)
		}
		if _, err := encodeConfigurationV3(c); err == nil {
			t.Fatal("invalid declaration encoded:", name)
		}
	}
	badPaths := map[string]string{
		"empty path":        "",
		"absolute posix":    "/etc/codex/config.toml",
		"windows drive":     "C:/codex/config.toml",
		"windows separator": `.codex\config.toml`,
		"unc":               "//server/share/config.toml",
		"unc backslash":     `\\server\share\config.toml`,
		"traversal":         "../config.toml",
		"inner traversal":   ".codex/../config.toml",
		"dot segment":       "./config.toml",
		"empty segment":     ".codex//config.toml",
		"trailing slash":    ".codex/config.toml/",
		"uri scheme":        "file:config.toml",
		"nul":               ".codex\x00/config.toml",
		"control":           ".codex\n/config.toml",
		"non-ascii":         ".cödex/config.toml",
		"trailing dot":      ".codex./config.toml",
		"trailing space":    ".codex /config.toml",
		"home expansion":    "~/config.toml",
		"oversized path":    strings.Repeat("a/", 128) + "config.toml",
	}
	for name, p := range badPaths {
		c := testV3()
		c.Runtimes[0].EffortDeclarations[0].Sources[0].Path = p
		if ValidEffortSourcePath(p) || ValidateV3(c) == nil {
			t.Fatal("unsafe source path accepted:", name)
		}
	}
	if !ValidEffortSourcePath("Library/Application Support/managed-settings.json") || !ValidEffortSourcePath("managed-settings.d") {
		t.Fatal("valid relative path rejected")
	}
}

func TestV3StrictWire(t *testing.T) {
	w := string(encodeV3(t, testV3()))
	for name, candidate := range map[string]string{
		"unknown declaration field": strings.Replace(w, `"declarationVersion":1`, `"declarationVersion":1,"evidenceRef":"proof"`, 1),
		"duplicate field":           strings.Replace(w, `"declarationVersion":1`, `"declarationVersion":1,"declarationVersion":1`, 1),
		"case alias":                strings.Replace(w, `"declarationId"`, `"DeclarationId"`, 1),
		"missing required field":    strings.Replace(w, `,"directory":false`, ``, 1),
		"unknown source field":      strings.Replace(w, `"directory":false`, `"directory":false,"root":"/home/operator"`, 1),
		"absolute root persisted":   strings.Replace(w, `"anchor":"user"`, `"anchor":"/home/operator"`, 1),
		"null keys":                 strings.Replace(w, `"jsonKeys":[]`, `"jsonKeys":null`, 1),
		"string version":            strings.Replace(w, `"declarationVersion":1`, `"declarationVersion":"1"`, 1),
		"credential reference":      strings.Replace(w, `"adapter":"codex"`, `"adapter":"codex","credentialReference":"keychain:codex"`, 1),
		"runtime id in declaration": strings.Replace(w, `"declarationVersion":1`, `"declarationVersion":1,"runtimeId":"codex"`, 1),
		"trailing document":         w + `{}`,
		"oversized":                 strings.Replace(w, `"revision":7`, `"revision":7,"pad":"`+strings.Repeat("x", MaxConfigurationBytes)+`"`, 1),
	} {
		if _, err := DecodeVersioned([]byte(candidate)); err == nil {
			t.Fatal("invalid v3 wire accepted:", name)
		}
	}
	none := testV3()
	none.Runtimes[0].EffortDeclarations = nil
	base := string(encodeV3(t, none))
	if _, err := DecodeVersioned([]byte(base)); err != nil {
		t.Fatal(err)
	}
	// Only omission spells "no declaration"; null and [] are refused.
	for _, spelling := range []string{"null", "[]"} {
		candidate := strings.Replace(base, `"allowlistedProfileIds":["codex-high"]`, `"allowlistedProfileIds":["codex-high"],"effortDeclarations":`+spelling, 1)
		if candidate == base {
			t.Fatal("fixture")
		}
		if _, err := DecodeVersioned([]byte(candidate)); err == nil {
			t.Fatal("second spelling of no declaration accepted:", spelling)
		}
	}
}
