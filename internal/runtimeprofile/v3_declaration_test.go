package runtimeprofile

import (
	"encoding/json"
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
	var d EffortDeclaration
	if err := json.Unmarshal([]byte(codexDeclarationJSON), &d); err != nil {
		t.Fatal(err)
	}
	return d
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
}

func claudeDeclaration(t *testing.T) EffortDeclaration {
	t.Helper()
	wire := `{"declarationId":"claude-effort-sources","declarationVersion":1,"runtimeVersion":"synthetic-claude-1","executableDigest":"2222222222222222222222222222222222222222222222222222222222222222","invocationMode":"noninteractive","precedenceEstablished":true,"jsonKeys":["fixtureEffortKey"],"environmentKeys":["CLAUDE_CODE_EFFORT_LEVEL"],"sources":[{"scope":"managed","anchor":"managed","path":"managed-settings.json","format":"json","directory":false},{"scope":"managed-drop-in","anchor":"managed","path":"managed-settings.d","format":"json","directory":true},{"scope":"project","anchor":"project","path":".claude/settings.json","format":"json","directory":false},{"scope":"project-local","anchor":"project","path":".claude/settings.local.json","format":"json","directory":false},{"scope":"user","anchor":"user","path":".claude/settings.json","format":"json","directory":false}]}`
	var d EffortDeclaration
	if err := json.Unmarshal([]byte(wire), &d); err != nil {
		t.Fatal(err)
	}
	return d
}
