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
)

// Specification 007 BI-3 through the production CLI with isolated state and a
// synthetic executable that is never invoked. No production version reader is
// composed before BLD-001B.3c, so applicability stays unproven: the explicit
// migration fails closed without writing, never probing the Runtime.
func TestExecutableDeclarationMigrationFailsClosedWithoutVersionObserver(t *testing.T) {
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
	before, e := os.ReadFile(path)
	if e != nil || !strings.Contains(string(before), `"formatVersion":2`) {
		t.Fatalf("%s %v", before, e)
	}
	run("effort_configuration_unproven", "runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3", "--effort-declaration", declaration)
	run("migration_required", "runtime", "binding", "edit", "--runtime", "codex", "--effort-declaration", declaration)
	link := filepath.Join(base, "link.json")
	if e := os.Symlink(declaration, link); e != nil {
		t.Fatal(e)
	}
	run("invalid_input", "runtime", "binding", "migrate", "--runtime", "codex", "--target-format", "3", "--effort-declaration", link)
	run("invalid_input", "runtime", "binding", "migrate", "--runtime", "codex", "--effort-declaration", declaration)
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Fatal("refused migration changed the v2 record")
	}
}
