package runtimeprofile

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const codexDeclarationJSON = `{"declarationId":"codex-effort-sources","declarationVersion":1,"runtimeVersion":"synthetic-codex-1","executableDigest":"1111111111111111111111111111111111111111111111111111111111111111","invocationMode":"noninteractive","precedenceEstablished":true,"jsonKeys":[],"environmentKeys":[],"sources":[{"scope":"user","anchor":"user","path":".codex/config.toml","format":"toml","directory":false},{"scope":"project","anchor":"project","path":".codex/config.toml","format":"toml","directory":false},{"scope":"managed","anchor":"managed","path":"managed_config.toml","format":"toml","directory":false}]}`

func migrationSource() Configuration {
	return Configuration{FormatVersion: FormatVersionV2, Revision: 4,
		Runtimes:      []Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-high"}}, {ID: "claude", Adapter: "claude", Enabled: false, AllowlistedProfileIDs: []string{"claude-low"}}},
		ModelProfiles: []ModelProfile{{ID: "codex-high", RuntimeID: "codex", Model: "local-codex-model", Capabilities: []string{"go"}, Complexities: []string{"high"}, Effort: Effort{Mode: "explicit", Value: "high"}}, {ID: "claude-low", RuntimeID: "claude", Model: "local-claude-model", Capabilities: []string{"go"}, Complexities: []string{"low"}, Effort: Effort{Mode: "runtime-default", Value: "default"}}},
		Preferences:   []Preference{{Role: "implementation", Complexity: "high", ModelProfileID: "codex-high"}},
	}
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

// BI-3.2 item 6: format 3, revision+1, every Runtime/Profile/allowlist/
// Preference unchanged and only the named Runtime gains the declaration.
func TestMigrateV2ToV3PreservesSource(t *testing.T) {
	source := migrationSource()
	sourceWire, err := Encode(source)
	if err != nil {
		t.Fatal(err)
	}
	sourceDigest, _ := Digest(source)
	profileDigests := map[string]string{}
	for _, p := range source.ModelProfiles {
		profileDigests[p.ID], _ = ProfileDigest(p)
	}
	next, err := MigrateV2ToV3(source, "codex", codexDeclaration(t))
	if err != nil {
		t.Fatal(err)
	}
	if next.FormatVersion != FormatVersionV3 || next.Revision != source.Revision+1 || len(next.Runtimes) != 2 || len(next.ModelProfiles) != 2 || len(next.Preferences) != 1 {
		t.Fatalf("%+v", next)
	}
	for _, r := range next.Runtimes {
		if r.ID == "codex" && (len(r.EffortDeclarations) != 1 || r.EffortDeclarations[0].DeclarationID != "codex-effort-sources") {
			t.Fatalf("declaration not bound: %+v", r)
		}
		if r.ID == "claude" && r.EffortDeclarations != nil {
			t.Fatal("declaration invented for another Runtime")
		}
	}
	for _, p := range next.ModelProfiles {
		if d, _ := ProfileDigest(p); d != profileDigests[p.ID] {
			t.Fatalf("Profile digest changed for %s", p.ID)
		}
	}
	wire, err := EncodeV3(next)
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeVersioned(wire)
	if err != nil || v.V3 == nil {
		t.Fatal(err)
	}
	if again, _ := EncodeV3(*v.V3); !bytes.Equal(again, wire) {
		t.Fatal("v3 candidate is not canonical")
	}
	// Historical v2 bytes and digest are untouched by migration.
	if after, _ := Encode(source); !bytes.Equal(after, sourceWire) {
		t.Fatal("v2 source re-encoded differently")
	}
	if after, _ := Digest(source); after != sourceDigest {
		t.Fatal("v2 digest changed")
	}
	// Other Runtimes stay unproven; no declaration is selected by version alone.
	if _, err := ApplicableEffortDeclaration(v, EffortApplicability{RuntimeID: "claude", RuntimeVersion: "synthetic-codex-1", ExecutableDigest: strings.Repeat("1", 64), InvocationMode: "noninteractive"}); !errors.Is(err, ErrEffortDeclarationUnproven) {
		t.Fatal("declaration applied to another Runtime")
	}
	if _, err := ApplicableEffortDeclaration(v, EffortApplicability{RuntimeID: "codex", RuntimeVersion: "synthetic-codex-1", ExecutableDigest: strings.Repeat("9", 64), InvocationMode: "noninteractive"}); !errors.Is(err, ErrEffortDeclarationUnproven) {
		t.Fatal("version alone selected a declaration")
	}
}

func TestMigrateV2ToV3Refusals(t *testing.T) {
	d := codexDeclaration(t)
	v1 := migrationSource()
	v1.FormatVersion = FormatVersion
	missing := migrationSource()
	missing.Runtimes = missing.Runtimes[1:]
	missing.ModelProfiles = missing.ModelProfiles[1:]
	missing.Preferences = nil
	for name, source := range map[string]Configuration{"v1": v1, "runtime-missing": missing, "empty": {}} {
		if _, err := MigrateV2ToV3(source, "codex", d); err == nil {
			t.Errorf("%s migrated", name)
		}
	}
}

// BI-3.1: add, replace by declarationId, refuse ambiguity and duplicates.
func TestWithEffortDeclaration(t *testing.T) {
	base, err := MigrateV2ToV3(migrationSource(), "codex", codexDeclaration(t))
	if err != nil {
		t.Fatal(err)
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
	claude, err := MigrateV2ToV3(migrationSource(), "claude", claudeDeclaration(t))
	if err != nil {
		t.Fatal(err)
	}
	stolen := codexDeclaration(t)
	stolen.DeclarationID = "claude-effort-sources"
	if _, _, err := WithEffortDeclaration(claude, "codex", stolen); err == nil {
		t.Fatal("declarationId owned by another Runtime accepted")
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
