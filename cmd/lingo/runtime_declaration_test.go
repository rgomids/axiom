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

// Specification 007 PM-1/PM-3 through the production CLI with isolated state
// and a synthetic executable that is never invoked. Existing authoring keeps
// writing format 2 until BLD-001B.3c-2; no v2→v3 migration exists, and a
// declaration is authored only on a format-3 record. No production version
// reader is composed before BLD-001B.3c-2, so declaration authoring fails
// closed without writing, never probing the Runtime.
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
	run := func(expected string, args ...string) map[string]any {
		cmd := exec.Command(binary, append([]string{"--json"}, args...)...)
		cmd.Env = append(os.Environ(), "PATH="+bin, "LINGO_STATE_ROOT="+state, "LINGO_PROJECTS_ROOT="+filepath.Join(base, "projects"), "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(base, "skills"))
		b, _ := cmd.CombinedOutput()
		var out struct {
			Category string         `json:"category"`
			Report   map[string]any `json:"runtimeAuthoring"`
		}
		if json.Unmarshal(b, &out) != nil || out.Category != expected || bytes.Contains(b, []byte("invoked")) || bytes.Contains(b, []byte(base)) {
			t.Fatalf("%v expected %s got %s", args, expected, b)
		}
		return out.Report
	}
	create := []string{"runtime", "binding", "create", "--runtime", "codex", "--profile", "codex-one", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "axiom-skills"}
	preview := run("preview", create...)
	run("applied", append(create, "--expected-revision", preview["expectedRevision"].(string), "--preview-digest", preview["previewDigest"].(string), "--authorize-local")...)
	path := filepath.Join(state, "runtime-profiles", "v1", "configuration.json")
	v2, e := os.ReadFile(path)
	if e != nil || !strings.Contains(string(v2), `"formatVersion":2`) {
		t.Fatalf("%s %v", v2, e)
	}
	// No v2→v3 migration surface: the flag is unknown and a declaration over
	// a format-2 record is refused without migrating or converting it.
	for _, args := range [][]string{{"runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3", "--effort-declaration", declaration}, {"runtime", "binding", "migrate", "--runtime", "codex", "--effort-declaration", declaration}} {
		cmd := exec.Command(binary, append([]string{"--json"}, args...)...)
		cmd.Env = append(os.Environ(), "PATH="+bin, "LINGO_STATE_ROOT="+state, "LINGO_PROJECTS_ROOT="+filepath.Join(base, "projects"), "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(base, "skills"))
		b, e := cmd.CombinedOutput()
		if e == nil || !bytes.Contains(b, []byte(`"status":"validation_failure"`)) || bytes.Contains(b, []byte("runtimeAuthoring")) {
			t.Fatalf("%v parsed: %s", args, b)
		}
	}
	run("configuration_unavailable", "runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", declaration)
	if after, _ := os.ReadFile(path); !bytes.Equal(v2, after) {
		t.Fatal("refusal changed the v2 record")
	}
	// A format-3 record (seeded directly; authoring activates it in 3c-2).
	v3, e := runtimeprofile.EncodeV3(runtimeprofile.ConfigurationV3{FormatVersion: runtimeprofile.FormatVersionV3, Revision: 2,
		Runtimes:      []runtimeprofile.RuntimeV3{{ID: "codex", Adapter: "codex", Enabled: true, AllowlistedProfileIDs: []string{"codex-one"}}},
		ModelProfiles: []runtimeprofile.ModelProfile{{ID: "codex-one", RuntimeID: "codex", Model: "operator-model", Capabilities: []string{"axiom-skills"}, Effort: runtimeprofile.Effort{Mode: "runtime-default", Value: "default"}}}})
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, v3, 0o600); e != nil {
		t.Fatal(e)
	}
	run("effort_configuration_unproven", "runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", declaration)
	link := filepath.Join(base, "link.json")
	if e := os.Symlink(declaration, link); e != nil {
		t.Fatal(e)
	}
	run("invalid_input", "runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", link)
	if after, _ := os.ReadFile(path); !bytes.Equal(v3, after) {
		t.Fatal("refused declaration authoring changed the v3 record")
	}
}
