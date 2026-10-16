package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutableCapabilityRefusalsAndPreparation(t *testing.T) {
	parent := os.Getenv("AXIOM_TEST_PROBE_STATE_ROOT")
	if parent == "" {
		t.Skip("requires private synthetic probe state outside repositories and temporary roots")
	}
	base, err := os.MkdirTemp(parent, "capability-cli-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(base) })
	binary := filepath.Join(t.TempDir(), "axiom")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, output)
	}
	fakeBin := filepath.Join(base, "bin")
	if err := os.Mkdir(fakeBin, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(base, "unexpected-runtime-invocation")
	for _, runtime := range []string{"codex", "claude"} {
		if err := os.WriteFile(filepath.Join(fakeBin, runtime), []byte("#!/bin/sh\n: > \"$AXIOM_FAKE_MARKER\"\nexit 99\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	state := filepath.Join(base, "state")
	env := append(os.Environ(), "LINGO_STATE_ROOT="+state, "LINGO_PROJECTS_ROOT="+filepath.Join(base, "projects"), "PATH="+fakeBin+":/usr/bin:/bin", "AXIOM_FAKE_MARKER="+marker)
	run := func(args []string, category string, success bool) map[string]any {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"--json"}, args...)...)
		cmd.Env = env
		wire, err := cmd.CombinedOutput()
		if (err == nil) != success {
			t.Fatalf("args=%v error=%v output=%s", args, err, wire)
		}
		var event map[string]any
		if json.Unmarshal(wire, &event) != nil || event["category"] != category {
			t.Fatalf("category=%s args=%v output=%s", category, args, wire)
		}
		return event
	}
	for _, runtime := range []string{"codex", "claude"} {
		for _, op := range []string{"show", "invalidate", "probe", "probe-extended"} {
			args := []string{"runtime", runtime, "capability", op, "--profile", "synthetic", "--variant", "repository-write"}
			category := "capability_unknown"
			if op == "probe" || op == "probe-extended" {
				category = "probe_authority_required"
				args = append(args, "--probe-id", "fake-probe", "--manifest-digest", strings.Repeat("a", 64), "--preview-digest", strings.Repeat("b", 64), "--authorize-local")
			}
			if op == "probe-extended" {
				args = append(args, "--roots", "tmpdir")
			}
			run(args, category, false)
		}
	}
	if _, err := os.Lstat(state); !os.IsNotExist(err) {
		t.Fatal("refused command wrote state", err)
	}
	args := []string{"runtime", "codex", "capability", "prepare", "--profile", "synthetic", "--variant", "repository-write", "--probe-id", "fake-preparation", "--repository-root", filepath.Join(base, "repository"), "--workspace-root", filepath.Join(base, "workspaces")}
	p := run(args, "capability_preparation_preview", true)
	if _, err := os.Lstat(state); !os.IsNotExist(err) {
		t.Fatal("preparation preview wrote state", err)
	}
	digest := p["capability"].(map[string]any)["previewDigest"].(string)
	apply := append(append([]string(nil), args...), "--preview-digest", digest)
	run(append(append([]string(nil), apply...), "--authorize-local"), "preparation_authority_required", false)
	r := run(append(apply, "--authorize-prepare"), "capability_envelope_prepared", true)
	if r["capability"].(map[string]any)["manifestDigest"] == "" {
		t.Fatal("missing preparation receipt")
	}
	if _, err := os.Lstat(marker); !os.IsNotExist(err) {
		t.Fatal("capability refusal/preparation invoked a Runtime", err)
	}
}
