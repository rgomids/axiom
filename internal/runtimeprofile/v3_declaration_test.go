package runtimeprofile

import (
	"errors"
	"strings"
	"testing"
)

const codexDeclarationJSON = `{"declarationId":"codex-effort-sources","declarationVersion":1,"runtimeVersion":"synthetic-codex-1","executableDigest":"1111111111111111111111111111111111111111111111111111111111111111","invocationMode":"noninteractive","precedenceEstablished":true,"jsonKeys":[],"environmentKeys":[],"sources":[{"scope":"user","anchor":"user","path":".codex/config.toml","format":"toml","directory":false},{"scope":"project","anchor":"project","path":".codex/config.toml","format":"toml","directory":false},{"scope":"managed","anchor":"managed","path":"managed_config.toml","format":"toml","directory":false}]}`

// declarationBase is a canonical v3 record authored directly in format 3; the
// pre-MVP policy has no v2 source and no migration (Specification 007 PM-1).
func declarationBase(runtimeID string, d EffortDeclaration) ConfigurationV3 {
	c := ConfigurationV3{FormatVersion: FormatVersionV3, Revision: 5,
		Runtimes:      []RuntimeV3{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-high"}}, {ID: "claude", Adapter: "claude", Enabled: false, AllowlistedProfileIDs: []string{"claude-low"}}},
		ModelProfiles: []ModelProfile{{ID: "codex-high", RuntimeID: "codex", Model: "local-codex-model", Capabilities: []string{"go"}, Complexities: []string{"high"}, Effort: Effort{Mode: "explicit", Value: "high"}}, {ID: "claude-low", RuntimeID: "claude", Model: "local-claude-model", Capabilities: []string{"go"}, Complexities: []string{"low"}, Effort: Effort{Mode: "runtime-default", Value: "default"}}},
		Preferences:   []Preference{{Role: "implementation", Complexity: "high", ModelProfileID: "codex-high"}},
	}
	for i := range c.Runtimes {
		if c.Runtimes[i].ID == runtimeID {
			c.Runtimes[i].EffortDeclarations = []EffortDeclaration{d}
		}
	}
	return c
}

func codexDeclaration(t *testing.T) EffortDeclaration {
	t.Helper()
	d, err := DecodeEffortDeclaration([]byte(codexDeclarationJSON), "codex")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// BI-3.1: one closed declaration object; nothing defaulted, aliased or extra.
func TestDecodeEffortDeclarationIsStrict(t *testing.T) {
	d := codexDeclaration(t)
	if d.ProofReference() != "codex-effort-sources" || len(d.Sources) != 3 {
		t.Fatalf("%+v", d)
	}
	refused := map[string]string{
		"unknown":        strings.Replace(codexDeclarationJSON, `"jsonKeys":[]`, `"jsonKeys":[],"capability":"proven"`, 1),
		"duplicate":      strings.Replace(codexDeclarationJSON, `"declarationVersion":1`, `"declarationVersion":1,"declarationVersion":1`, 1),
		"case-alias":     strings.Replace(codexDeclarationJSON, `"declarationId"`, `"DeclarationId"`, 1),
		"missing":        strings.Replace(codexDeclarationJSON, `"precedenceEstablished":true,`, ``, 1),
		"unversioned":    strings.Replace(codexDeclarationJSON, `"declarationVersion":1`, `"declarationVersion":2`, 1),
		"malformed":      codexDeclarationJSON[:40],
		"trailing":       codexDeclarationJSON + `{}`,
		"null":           `null`,
		"array":          `[` + codexDeclarationJSON + `]`,
		"empty":          ``,
		"oversized":      codexDeclarationJSON + strings.Repeat(" ", MaxConfigurationBytes),
		"source-missing": strings.Replace(codexDeclarationJSON, `{"scope":"managed","anchor":"managed","path":"managed_config.toml","format":"toml","directory":false}`, `{"scope":"managed","anchor":"managed","path":"managed_config.toml","format":"toml"}`, 1),
		"absolute-path":  strings.Replace(codexDeclarationJSON, `"path":"managed_config.toml"`, `"path":"/etc/managed_config.toml"`, 1),
		"runtime-field":  strings.Replace(codexDeclarationJSON, `"jsonKeys":[]`, `"jsonKeys":[],"runtimeId":"codex"`, 1),
	}
	for name, wire := range refused {
		if _, err := DecodeEffortDeclaration([]byte(wire), "codex"); !errors.Is(err, ErrInvalidConfiguration) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
	// The enclosing Runtime's rules apply: a Codex declaration is not Claude's.
	if _, err := DecodeEffortDeclaration([]byte(codexDeclarationJSON), "claude"); err == nil {
		t.Fatal("declaration accepted for another Runtime")
	}
}

// BI-3.1: add, replace by declarationId, refuse ambiguity and duplicates.
func TestWithEffortDeclaration(t *testing.T) {
	base := declarationBase("codex", codexDeclaration(t))
	if ValidateV3(base) != nil {
		t.Fatal("invalid base")
	}
	other := codexDeclaration(t)
	other.DeclarationID = "codex-next"
	other.RuntimeVersion = "synthetic-codex-2"
	added, replaced, err := WithEffortDeclaration(base, "codex", other)
	if err != nil || replaced || added.Revision != base.Revision+1 || len(added.Runtimes[1].EffortDeclarations)+len(added.Runtimes[0].EffortDeclarations) != 2 {
		t.Fatalf("add: %+v %v %v", added, replaced, err)
	}
	same := codexDeclaration(t)
	same.JSONKeys = []string{}
	same.EnvironmentKeys = []string{"CODEX_EFFORT"}
	updated, replaced, err := WithEffortDeclaration(base, "codex", same)
	if err != nil || !replaced || updated.Revision != base.Revision+1 {
		t.Fatalf("replace: %v %v", replaced, err)
	}
	for _, r := range updated.Runtimes {
		if r.ID == "codex" && (len(r.EffortDeclarations) != 1 || r.EffortDeclarations[0].EnvironmentKeys[0] != "CODEX_EFFORT") {
			t.Fatalf("replacement not applied: %+v", r)
		}
	}
	ambiguous := codexDeclaration(t)
	ambiguous.DeclarationID = "codex-alias"
	if _, _, err := WithEffortDeclaration(base, "codex", ambiguous); err == nil {
		t.Fatal("ambiguous applicability tuple accepted")
	}
	if _, _, err := WithEffortDeclaration(base, "absent", other); err == nil {
		t.Fatal("missing Runtime accepted")
	}
	// A declaration is added to a v3 Runtime that has none yet.
	empty := declarationBase("", EffortDeclaration{})
	if first, replaced, err := WithEffortDeclaration(empty, "codex", codexDeclaration(t)); err != nil || replaced || len(first.Runtimes[0].EffortDeclarations) != 1 {
		t.Fatalf("first declaration: %v %v", replaced, err)
	}
	claude := declarationBase("claude", claudeDeclaration(t))
	stolen := codexDeclaration(t)
	stolen.DeclarationID = "claude-effort-sources"
	if _, _, err := WithEffortDeclaration(claude, "codex", stolen); err == nil {
		t.Fatal("declarationId owned by another Runtime accepted")
	}
}

// PM-1: no v2 → v3 conversion exists; format 1/2 records never become a v3
// record, and ApplicableEffortDeclaration never selects for them.
func TestNoHistoricalFormatBecomesV3(t *testing.T) {
	v2 := Configuration{FormatVersion: FormatVersionV2, Revision: 4,
		Runtimes:      []Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-high"}}},
		ModelProfiles: []ModelProfile{{ID: "codex-high", RuntimeID: "codex", Model: "local-codex-model", Capabilities: []string{"go"}, Complexities: []string{"high"}, Effort: Effort{Mode: "explicit", Value: "high"}}},
	}
	wire, err := Encode(v2)
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeVersioned(wire)
	if err != nil || v.V3 != nil || v.FormatVersion != FormatVersionV2 {
		t.Fatalf("v2 reinterpreted: %+v %v", v, err)
	}
	if _, err := ApplicableEffortDeclaration(v, EffortApplicability{RuntimeID: "codex", RuntimeVersion: "synthetic-codex-1", ExecutableDigest: strings.Repeat("1", 64), InvocationMode: "noninteractive"}); !errors.Is(err, ErrEffortDeclarationUnproven) {
		t.Fatal("declaration synthesized for a v2 record")
	}
}

func claudeDeclaration(t *testing.T) EffortDeclaration {
	t.Helper()
	wire := `{"declarationId":"claude-effort-sources","declarationVersion":1,"runtimeVersion":"synthetic-claude-1","executableDigest":"2222222222222222222222222222222222222222222222222222222222222222","invocationMode":"noninteractive","precedenceEstablished":true,"jsonKeys":["fixtureEffortKey"],"environmentKeys":["CLAUDE_CODE_EFFORT_LEVEL"],"sources":[{"scope":"managed","anchor":"managed","path":"managed-settings.json","format":"json","directory":false},{"scope":"managed-drop-in","anchor":"managed","path":"managed-settings.d","format":"json","directory":true},{"scope":"project","anchor":"project","path":".claude/settings.json","format":"json","directory":false},{"scope":"project-local","anchor":"project","path":".claude/settings.local.json","format":"json","directory":false},{"scope":"user","anchor":"user","path":".claude/settings.json","format":"json","directory":false}]}`
	d, err := DecodeEffortDeclaration([]byte(wire), "claude")
	if err != nil {
		t.Fatal(err)
	}
	return d
}
