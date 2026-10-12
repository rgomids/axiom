package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/runtimeapplication"
)

// Production CLI authoring, isolated state and synthetic installed executables.
// This exercises real Axiom filesystem writes, never vendor execution/proof.
func TestExecutableRuntimeAuthoringRestartAndRetirement(t *testing.T) {
	binary := filepath.Join(t.TempDir(), testExecutableName("axiom"))
	build := exec.Command("go", "build", "-o", binary, ".")
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatalf("build: %v %s", e, b)
	}
	for _, installed := range [][]string{nil, {"codex"}, {"claude"}, {"codex", "claude"}} {
		t.Run(strings.Join(installed, "-")+"inventory", func(t *testing.T) {
			base := t.TempDir()
			state := filepath.Join(base, "state")
			bin := filepath.Join(base, "bin")
			if e := os.Mkdir(bin, 0700); e != nil {
				t.Fatal(e)
			}
			for _, id := range installed {
				name := id
				if runtime.GOOS == "windows" {
					name += ".exe"
				}
				if e := os.WriteFile(filepath.Join(bin, name), []byte("synthetic executable: must never be invoked\n"), 0700); e != nil {
					t.Fatal(e)
				}
			}
			run := func(expected string, args ...string) runtimeapplication.AuthoringReport {
				cmd := exec.Command(binary, append([]string{"--json"}, args...)...)
				cmd.Env = append(os.Environ(), "PATH="+bin, "LINGO_STATE_ROOT="+state, "LINGO_PROJECTS_ROOT="+filepath.Join(base, "projects"), "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(base, "skills"))
				b, e := cmd.CombinedOutput()
				var out struct {
					Category string                             `json:"category"`
					Report   runtimeapplication.AuthoringReport `json:"runtimeAuthoring"`
				}
				if json.Unmarshal(b, &out) != nil || out.Category != expected {
					t.Fatalf("%v expected %s got %s err=%v", args, expected, b, e)
				}
				if bytes.Contains(b, []byte("private-canary")) {
					t.Fatal("rejected input leaked")
				}
				return out.Report
			}
			inspected := run("inspected", "runtime", "binding", "list")
			if inspected.State != "fresh" {
				t.Fatal(inspected)
			}
			if _, e := os.Stat(state); !os.IsNotExist(e) {
				t.Fatal("passive read wrote state")
			}
			run("invalid_input", "runtime", "binding", "create", "--runtime", "codex", "--profile", "sample", "--model", "token=private-canary", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "axiom-skills")
			if len(installed) == 0 {
				run("runtime_absent", "runtime", "binding", "create", "--runtime", "codex", "--profile", "sample", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "axiom-skills")
				return
			}
			apply := func(args []string) runtimeapplication.AuthoringReport {
				preview := run("preview", args...)
				if _, e := os.Stat(state); os.IsNotExist(e) && preview.State != "fresh" {
					t.Fatal("lost state")
				}
				auth := append(append([]string{}, args...), "--expected-revision", preview.ExpectedRevision, "--preview-digest", preview.PreviewDigest, "--authorize-local")
				result := run("applied", auth...)
				run("stale_preview", auth...)
				return result
			}
			var last runtimeapplication.AuthoringReport
			for _, id := range installed {
				last = apply([]string{"runtime", "binding", "create", "--runtime", id, "--profile", id + "-one", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "axiom-skills"})
				for _, name := range []string{"two", "three", "four"} {
					last = apply([]string{"runtime", "profile", "create", "--runtime", id, "--profile", id + "-" + name, "--model", "operator-model", "--effort-mode", "explicit", "--effort", "operator-token", "--capabilities", "axiom-skills"})
				}
			}
			view := run("inspected", "runtime", "profile", "list")
			if len(view.Profiles) != 4*len(installed) || view.Revision != uint64(4*len(installed)) {
				t.Fatal("restart mismatch", view)
			}
			before, e := os.ReadFile(filepath.Join(state, "runtime-profiles", "v1", "configuration.json"))
			if e != nil {
				t.Fatal(e)
			}
			run("compound_removal_required", "runtime", "binding", "remove", "--runtime", installed[0])
			after, _ := os.ReadFile(filepath.Join(state, "runtime-profiles", "v1", "configuration.json"))
			if !bytes.Equal(before, after) {
				t.Fatal("refusal changed bytes")
			}
			for _, id := range installed {
				last = apply([]string{"runtime", "binding", "remove", "--runtime", id, "--cascade"})
			}
			if last.State != "retired" {
				t.Fatal("last removal did not tombstone", last)
			}
			recreated := apply([]string{"runtime", "binding", "create", "--runtime", installed[0], "--profile", "recreated", "--model", "operator-model", "--effort-mode", "runtime-default", "--effort", "default", "--capabilities", "axiom-skills"})
			if recreated.Revision != last.Revision+1 {
				t.Fatal("generation reset")
			}
		})
	}
}
