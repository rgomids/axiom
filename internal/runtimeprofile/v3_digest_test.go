package runtimeprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
)

// DigestV3 golden vectors (PM-8). Each canonical byte file was written by the
// unmodified EncodeV3 and each digest is SHA-256 over that file.
func digestV3Vectors() []struct {
	name, file, digest string
	cfg                ConfigurationV3
} {
	codexOnly := testV3()
	codexOnly.Runtimes[1].EffortDeclarations = nil
	claudeOnly := testV3()
	claudeOnly.Runtimes[0].EffortDeclarations = nil
	multiple := testV3()
	second := testCodexDeclaration()
	second.DeclarationID, second.RuntimeVersion, second.ExecutableDigest = "codex-effort-sources-2", "synthetic-codex-2", strings.Repeat("3", 64)
	multiple.Runtimes[0].EffortDeclarations = append(multiple.Runtimes[0].EffortDeclarations, second)
	preferences := testV3()
	preferences.Preferences = append(preferences.Preferences, Preference{Role: "review", Complexity: "low", ModelProfileID: "claude-low"})
	return []struct {
		name, file, digest string
		cfg                ConfigurationV3
	}{
		{"codex declaration", "testdata/digest-v3/codex-declaration.json", "5765a5115873af46b2a417b7a9423f719ebf3597a75dd746bca2c5a76fef6f15", codexOnly},
		{"claude declaration", "testdata/digest-v3/claude-declaration.json", "9addda042d6bc65165427a94ab5587c7abf7f85560928273cc4174ce3d0a7231", claudeOnly},
		{"multiple runtimes", "testdata/configuration-v3.json", "e5bb920d17b844809f872f0565c993b86b49b9b49291d433b6a336bb7e681d40", testV3()},
		{"multiple declarations", "testdata/digest-v3/multiple-declarations.json", "4d705a218588d345871c9207d2623f9f4521bb3e711514e446d9bdbffc853da9", multiple},
		{"nonempty preferences", "testdata/digest-v3/preferences.json", "2b1cb1c6f220c44e6e1e8bafa0e898b0155f2f46582454ef8cdc7a6e39235ac6", preferences},
	}
}

// reversedV3 reverses every set-like collection EncodeV3 orders.
func reversedV3(c ConfigurationV3) ConfigurationV3 {
	n := canonicalV3(c)
	reverse := func(n int, swap func(i, j int)) {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			swap(i, j)
		}
	}
	reverse(len(n.Runtimes), func(i, j int) { n.Runtimes[i], n.Runtimes[j] = n.Runtimes[j], n.Runtimes[i] })
	for _, r := range n.Runtimes {
		reverse(len(r.EffortDeclarations), func(i, j int) {
			r.EffortDeclarations[i], r.EffortDeclarations[j] = r.EffortDeclarations[j], r.EffortDeclarations[i]
		})
		for _, d := range r.EffortDeclarations {
			reverse(len(d.Sources), func(i, j int) { d.Sources[i], d.Sources[j] = d.Sources[j], d.Sources[i] })
			reverse(len(d.JSONKeys), func(i, j int) { d.JSONKeys[i], d.JSONKeys[j] = d.JSONKeys[j], d.JSONKeys[i] })
			reverse(len(d.EnvironmentKeys), func(i, j int) {
				d.EnvironmentKeys[i], d.EnvironmentKeys[j] = d.EnvironmentKeys[j], d.EnvironmentKeys[i]
			})
		}
	}
	reverse(len(n.ModelProfiles), func(i, j int) { n.ModelProfiles[i], n.ModelProfiles[j] = n.ModelProfiles[j], n.ModelProfiles[i] })
	reverse(len(n.Preferences), func(i, j int) { n.Preferences[i], n.Preferences[j] = n.Preferences[j], n.Preferences[i] })
	return n
}

func digestV3(t *testing.T, c ConfigurationV3) string {
	t.Helper()
	d, err := DigestV3(c)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDigestV3GoldenVectors(t *testing.T) {
	for _, v := range digestV3Vectors() {
		golden, err := os.ReadFile(v.file)
		if err != nil {
			t.Fatal(v.name, err)
		}
		if w := encodeV3(t, v.cfg); string(w) != string(golden) {
			t.Fatalf("%s: canonical v3 bytes changed\n%s", v.name, w)
		}
		sum := sha256.Sum256(golden)
		d := digestV3(t, v.cfg)
		if d != v.digest || hex.EncodeToString(sum[:]) != v.digest {
			t.Fatalf("%s: DigestV3 = %s, want SHA-256(EncodeV3) %s", v.name, d, v.digest)
		}
		// Deterministic repeated encoding and EncodeV3-defined order insensitivity.
		if digestV3(t, v.cfg) != d || digestV3(t, reversedV3(v.cfg)) != d {
			t.Fatalf("%s: DigestV3 is not canonical", v.name)
		}
	}
}

// Every valid semantic change, including one confined to an EffortDeclaration,
// changes the digest; mutations the closed schema forbids are refused.
func TestDigestV3SemanticDrift(t *testing.T) {
	base := digestV3(t, testV3())
	codex := func(c *ConfigurationV3) *EffortDeclaration { return &c.Runtimes[0].EffortDeclarations[0] }
	claude := func(c *ConfigurationV3) *EffortDeclaration { return &c.Runtimes[1].EffortDeclarations[0] }
	valid := map[string]func(*ConfigurationV3){
		"revision": func(c *ConfigurationV3) { c.Revision++ },
		"runtime identity set": func(c *ConfigurationV3) {
			c.Runtimes, c.ModelProfiles, c.Preferences = c.Runtimes[:1], c.ModelProfiles[:1], c.Preferences[:1]
		},
		"runtime enabled":      func(c *ConfigurationV3) { c.Runtimes[1].Enabled = false },
		"profile model":        func(c *ConfigurationV3) { c.ModelProfiles[0].Model = "other-codex-model" },
		"profile effort value": func(c *ConfigurationV3) { c.ModelProfiles[0].Effort = Effort{EffortExplicit, "low"} },
		"profile effort mode":  func(c *ConfigurationV3) { c.ModelProfiles[0].Effort = Effort{EffortRuntimeDefault, "default"} },
		"preferences":          func(c *ConfigurationV3) { c.Preferences[0].Role = "review" },
		"preferences absent":   func(c *ConfigurationV3) { c.Preferences = nil },
		"declarationId":        func(c *ConfigurationV3) { codex(c).DeclarationID = "codex-effort-renamed" },
		"runtimeVersion":       func(c *ConfigurationV3) { codex(c).RuntimeVersion = "synthetic-codex-2" },
		"executableDigest":     func(c *ConfigurationV3) { codex(c).ExecutableDigest = strings.Repeat("4", 64) },
		"source path":          func(c *ConfigurationV3) { codex(c).Sources[0].Path = ".codex/alt/config.toml" },
		"source scope and anchor": func(c *ConfigurationV3) {
			// user ↔ managed: each source keeps its path; scope and anchor move.
			s := codex(c).Sources
			s[0].Scope, s[0].Anchor, s[2].Scope, s[2].Anchor = s[2].Scope, s[2].Anchor, s[0].Scope, s[0].Anchor
		},
		"json key":           func(c *ConfigurationV3) { claude(c).JSONKeys = []string{"otherFixtureKey"} },
		"json key inventory": func(c *ConfigurationV3) { claude(c).JSONKeys = append(claude(c).JSONKeys, "otherFixtureKey") },
		"environment key inventory": func(c *ConfigurationV3) {
			codex(c).EnvironmentKeys = []string{"CODEX_FIXTURE_EFFORT"}
		},
		"declaration removed": func(c *ConfigurationV3) { c.Runtimes[1].EffortDeclarations = nil },
		"declaration added": func(c *ConfigurationV3) {
			d := testCodexDeclaration()
			d.DeclarationID, d.RuntimeVersion = "codex-effort-sources-2", "synthetic-codex-2"
			c.Runtimes[0].EffortDeclarations = append(c.Runtimes[0].EffortDeclarations, d)
		},
	}
	seen := map[string]string{base: "base"}
	for name, mutate := range valid {
		c := testV3()
		mutate(&c)
		d := digestV3(t, c)
		if previous, ok := seen[d]; ok {
			t.Fatalf("%s: digest equals %s", name, previous)
		}
		seen[d] = name
	}
	invalid := map[string]func(*ConfigurationV3){
		"runtime renamed":        func(c *ConfigurationV3) { c.Runtimes[0].ID = "codex-2" },
		"declarationVersion":     func(c *ConfigurationV3) { codex(c).DeclarationVersion = 2 },
		"invocationMode":         func(c *ConfigurationV3) { codex(c).InvocationMode = "interactive" },
		"precedence unknown":     func(c *ConfigurationV3) { codex(c).PrecedenceEstablished = false },
		"anchor without scope":   func(c *ConfigurationV3) { codex(c).Sources[0].Anchor = EffortAnchorManaged },
		"required source absent": func(c *ConfigurationV3) { codex(c).Sources = codex(c).Sources[:2] },
		"absolute source path":   func(c *ConfigurationV3) { codex(c).Sources[0].Path = "/etc/codex/config.toml" },
		"ambiguous declaration": func(c *ConfigurationV3) {
			d := testCodexDeclaration()
			d.DeclarationID = "codex-effort-sources-2"
			c.Runtimes[0].EffortDeclarations = append(c.Runtimes[0].EffortDeclarations, d)
		},
		"format version": func(c *ConfigurationV3) { c.FormatVersion = FormatVersionV2 },
		"zero revision":  func(c *ConfigurationV3) { c.Revision = 0 },
	}
	for name, mutate := range invalid {
		c := testV3()
		mutate(&c)
		if d, err := DigestV3(c); d != "" || !errors.Is(err, ErrInvalidConfiguration) {
			t.Fatalf("%s: malformed v3 digested: %q, %v", name, d, err)
		}
	}
	if d, err := DigestV3(ConfigurationV3{}); d != "" || !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("zero configuration digested: %q, %v", d, err)
	}
}

// DigestV3 is additive: the historical Digest, ProfileDigest and Profile effort
// ownership keep their values, and a declaration never becomes proof.
func TestDigestV3PreservesHistoricalIdentities(t *testing.T) {
	c := testV3()
	if d, err := Digest(testV2()); err != nil || d != "8a7f9a6ee81988d6f37ddf48282f166e6d7ba73cc5c5c3776cd178a7f053d19e" {
		t.Fatalf("historical Digest changed: %v", err)
	}
	if h, _ := Digest(projectV2(c)); h == digestV3(t, c) {
		t.Fatal("DigestV3 reinterpreted the historical Digest")
	}
	profiles := map[string]string{"codex-high": "4d99ee2d946784844a6ad5d66ce48e866b27add27f84383c74d13708ab0d12d0", "claude-low": "c46f48de857d99eb63fa3631a315b2d0fa94c4038fb2516a60d9e234f1a53bac"}
	for _, p := range c.ModelProfiles {
		if d, err := ProfileDigest(p); err != nil || d != profiles[p.ID] {
			t.Fatalf("ProfileDigest %s = %s, %v", p.ID, d, err)
		}
	}
	// A declaration-only change leaves every Profile identity intact.
	changed := testV3()
	changed.Runtimes[0].EffortDeclarations[0].RuntimeVersion = "synthetic-codex-2"
	for i, p := range changed.ModelProfiles {
		if a, _ := ProfileDigest(p); a != profiles[c.ModelProfiles[i].ID] {
			t.Fatal("declaration drift changed a Profile identity")
		}
	}
	if wire := string(encodeV3(t, c)); strings.Contains(wire, "capabilityStatus") || strings.Contains(wire, "evidence") {
		t.Fatal("v3 canonical bytes carry capability proof")
	}
}
