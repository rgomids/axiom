package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeadapter"
	"github.com/rgomids/axiom/internal/runtimeprofile"
)

// Specification 007 PM-1/PM-9/PM-11 through the production CLI with isolated
// state and a synthetic executable that is never invoked: canonical v3 is
// authored directly, the last Runtime retires through the existing tombstone
// bound to DigestV3, a pre-stable record is only retired as a whole, and no
// migration surface exists. No production version is registered, so
// declaration authoring fails closed without writing or probing the Runtime.
func TestExecutableDeclarationAuthoringWithoutMigration(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "axiom")
	if b, e := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); e != nil {
		t.Fatalf("build: %v %s", e, b)
	}
	base := t.TempDir()
	state, bin := filepath.Join(base, "state"), filepath.Join(base, "bin")
	if e := os.Mkdir(bin, 0o700); e != nil {
		t.Fatal(e)
	}
	executable := filepath.Join(bin, "codex")
	if e := os.WriteFile(executable, []byte("#!/bin/sh\necho invoked >&2\nexit 97\n"), 0o700); e != nil {
		t.Fatal(e)
	}
	digest, e := runtimeadapter.ExecutableIdentity(executable)
	if e != nil {
		t.Fatal(e)
	}
	declaration := filepath.Join(base, "declaration.json")
	wire := `{"declarationId":"codex-effort-sources","declarationVersion":1,"runtimeVersion":"synthetic-codex-1","executableDigest":"` + digest + `","invocationMode":"noninteractive","precedenceEstablished":true,"jsonKeys":[],"environmentKeys":[],"sources":[{"scope":"user","anchor":"user","path":".codex/config.toml","format":"toml","directory":false},{"scope":"project","anchor":"project","path":".codex/config.toml","format":"toml","directory":false},{"scope":"managed","anchor":"managed","path":"managed_config.toml","format":"toml","directory":false}]}`
	if e := os.WriteFile(declaration, []byte(wire), 0o644); e != nil {
		t.Fatal(e)
	}
	environment := func(root string) []string {
		return append(os.Environ(), "PATH="+bin, "LINGO_STATE_ROOT="+root, "LINGO_PROJECTS_ROOT="+filepath.Join(base, "projects"), "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(base, "skills"))
	}
	runIn := func(root, expected string, args ...string) map[string]any {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--json"}, args...)...)
		cmd.Env = environment(root)
		b, _ := cmd.CombinedOutput()
		var out struct {
			Category string         `json:"category"`
			Status   string         `json:"status"`
			Report   map[string]any `json:"runtimeAuthoring"`
		}
		if json.Unmarshal(b, &out) != nil || (expected != "invalid_command" && out.Category != expected || expected == "invalid_command" && (out.Status != "validation_failure" || out.Report != nil)) || bytes.Contains(b, []byte("invoked")) || bytes.Contains(b, []byte(base)) {
			t.Fatalf("%v expected %s got %s", args, expected, b)
		}
		return out.Report
	}
	run := func(expected string, args ...string) map[string]any {
		t.Helper()
		return runIn(state, expected, args...)
	}
	apply := func(root string, args ...string) map[string]any {
		t.Helper()
		preview := runIn(root, "preview", args...)
		return runIn(root, "applied", append(args, "--expected-revision", preview["expectedRevision"].(string), "--preview-digest", preview["previewDigest"].(string), "--authorize-local")...)
	}
	path := filepath.Join(state, "runtime-profiles", "v1", "configuration.json")
	decode := func(p string) *runtimeprofile.ConfigurationV4 {
		t.Helper()
		b, e := os.ReadFile(p)
		v, de := runtimeprofile.DecodeVersioned(b)
		if e != nil || de != nil || v.V4 == nil {
			t.Fatalf("not canonical v4: %s %v %v", b, e, de)
		}
		return v.V4
	}
	// Fresh state initializes canonical v3 directly.
	create := []string{"runtime", "binding", "create", "--runtime", "codex", "--profile", "codex-one", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "axiom-skills"}
	apply(state, create...)
	if cfg := decode(path); cfg.Revision != 1 || len(cfg.Runtimes) != 1 {
		t.Fatalf("created %+v", cfg)
	}
	// No migration surface: the commands and flags are unknown.
	for _, args := range [][]string{{"runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3", "--effort-declaration", declaration}, {"runtime", "profile", "migrate", "--profile-effort", "codex-one:runtime-default:default"}} {
		cmd := exec.Command(binary, append([]string{"--json"}, args...)...)
		cmd.Env = environment(state)
		b, e := cmd.CombinedOutput()
		if e == nil || !bytes.Contains(b, []byte(`"category":"invalid_command"`)) || bytes.Contains(b, []byte(`"runtimeAuthoring":{`)) {
			t.Fatalf("%v parsed: %s", args, b)
		}
	}
	v3, _ := os.ReadFile(path)
	// The embedded registry has no production entry: the observed version is
	// unproven and nothing is written or executed.
	run("invalid_command", "runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", declaration)
	link := filepath.Join(base, "link.json")
	if e := os.Symlink(declaration, link); e != nil {
		t.Fatal(e)
	}
	run("invalid_command", "runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", link)
	if after, _ := os.ReadFile(path); !bytes.Equal(v3, after) {
		t.Fatal("refused declaration authoring changed the v3 record")
	}
	// Structural v3 authoring needs no capability proof.
	apply(state, "runtime", "binding", "disable", "--runtime", "codex")
	if cfg := decode(path); cfg.Revision != 2 || cfg.Runtimes[0].Enabled {
		t.Fatalf("disabled %+v", cfg)
	}
	apply(state, "runtime", "binding", "enable", "--runtime", "codex")
	prior := decode(path)
	if prior.Revision != 3 || !prior.Runtimes[0].Enabled {
		t.Fatalf("enabled %+v", prior)
	}
	// Removing the last Runtime publishes the existing tombstone bound to
	// DigestV3 of the exact prior record; no empty v3 configuration exists.
	run("compound_removal_required", "runtime", "binding", "remove", "--runtime", "codex")
	apply(state, "runtime", "binding", "remove", "--runtime", "codex", "--cascade")
	b, _ := os.ReadFile(path)
	tomb, e := runtimeprofile.DecodeTombstone(b)
	if want, _ := runtimeprofile.DigestV4(*prior); e != nil || tomb.Revision != 4 || tomb.FormatVersion != 1 || tomb.PriorConfigurationDigest != want {
		t.Fatalf("retirement %s %v", b, e)
	}
	// Fresh explicit reconfiguration after retirement; nothing is resurrected.
	run("inspected", "runtime", "binding", "list")
	apply(state, create...)
	if cfg := decode(path); cfg.Revision != 5 {
		t.Fatalf("recreated %+v", cfg)
	}

	// PM-11 A1: a pre-stable record is refused for every current use and is
	// retired only as a whole, with the historical Digest.
	legacyState := filepath.Join(base, "legacy-state")
	legacy := runtimeprofile.Configuration{FormatVersion: runtimeprofile.FormatVersion, Revision: 1,
		Runtimes:      []runtimeprofile.Runtime{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-default"}}, {ID: "claude", Adapter: "claude", Enabled: true, AllowlistedProfileIDs: []string{"claude-default"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-default", RuntimeID: "codex", Model: "local-codex", Capabilities: []string{"go"}}, {ID: "claude-default", RuntimeID: "claude", Model: "local-claude", Capabilities: []string{"go"}}},
	}
	source := writeLegacyRuntimeProfile(t, legacyState, legacy)
	legacyPath := filepath.Join(legacyState, "runtime-profiles", "v1", "configuration.json")
	runIn(legacyState, "unsupported_configuration_format", create...)
	runIn(legacyState, "unsupported_configuration_format", "runtime", "binding", "disable", "--runtime", "codex")
	runIn(legacyState, "binding_missing", "runtime", "binding", "remove", "--runtime", "other", "--cascade")
	runIn(legacyState, "compound_removal_required", "runtime", "binding", "remove", "--runtime", "codex")
	preview := runIn(legacyState, "preview", "runtime", "binding", "remove", "--runtime", "codex", "--cascade")
	effects, _ := json.Marshal(preview["effects"])
	historical, _ := runtimeprofile.Digest(legacy)
	for _, want := range []string{"retire ENTIRE historical local configuration", "retire Runtime claude", "retire Profile claude-default", historical} {
		if !strings.Contains(string(effects), want) {
			t.Fatalf("legacy retirement preview lacks %q: %s", want, effects)
		}
	}
	runIn(legacyState, "local_authority_required", "runtime", "binding", "remove", "--runtime", "codex", "--cascade", "--expected-revision", preview["expectedRevision"].(string), "--preview-digest", preview["previewDigest"].(string))
	if after, _ := os.ReadFile(legacyPath); !bytes.Equal(after, source) {
		t.Fatal("refused legacy operations changed the pre-stable bytes")
	}
	apply(legacyState, "runtime", "binding", "remove", "--runtime", "codex", "--cascade")
	b, _ = os.ReadFile(legacyPath)
	if tomb, e := runtimeprofile.DecodeTombstone(b); e != nil || tomb.Revision != 2 || tomb.PriorConfigurationDigest != historical {
		t.Fatalf("legacy retirement %s %v", b, e)
	}
	apply(legacyState, create...)
	if cfg := decode(legacyPath); cfg.Revision != 3 || len(cfg.Runtimes) != 1 || cfg.ModelProfiles[0].ID != "codex-one" {
		t.Fatalf("explicit v3 after legacy retirement %+v", cfg)
	}
}
