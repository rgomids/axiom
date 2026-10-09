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

	"github.com/rgomids/axiom/internal/cli"
)

func TestHierarchicalHelpBlackboxPrecedesComposition(t *testing.T) {
	binary := filepath.Join(t.TempDir(), testExecutableName("axiom"))
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, output)
	}
	for _, roots := range []string{"invalid", "absent", "existing"} {
		t.Run(roots, func(t *testing.T) {
			base := t.TempDir()
			home, bin, cwd := filepath.Join(base, "home"), filepath.Join(base, "bin"), filepath.Join(base, "cwd")
			for _, path := range []string{home, bin, cwd} {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			marker := filepath.Join(base, "runtime-executed")
			// Shell builtins keep the trap effective even with PATH restricted to
			// this directory: an attempted Runtime/Provider execution leaves a mark.
			script := []byte("#!/bin/sh\nprintf called > '" + marker + "'\nexit 99\n")
			if runtime.GOOS == "windows" {
				script = []byte("@echo called>\"" + marker + "\"\r\n@exit /b 99\r\n")
			}
			for _, name := range []string{"codex", "claude", "gh", "git"} {
				if err := os.WriteFile(filepath.Join(bin, testRuntimeName(name)), script, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			projects, state, skills := filepath.Join(base, "projects"), filepath.Join(base, "state"), filepath.Join(base, "skills")
			if roots == "invalid" {
				projects, state, skills = "relative/projects", "relative/state", "relative/skills"
			}
			if roots == "existing" {
				for _, path := range []string{projects, state, skills} {
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(path, "sentinel"), []byte("preserve\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			environment := []string{
				"HOME=" + home, "USERPROFILE=" + home, "LOCALAPPDATA=" + filepath.Join(home, "AppData", "Local"),
				"PATH=" + bin, "CLAUDE_CONFIG_DIR=" + filepath.Join(home, "claude"), "XDG_STATE_HOME=" + filepath.Join(home, "xdg"),
				"LINGO_PROJECTS_ROOT=" + projects, "LINGO_STATE_ROOT=" + state, "AXIOM_CODEX_SKILLS_ROOT=" + skills,
				"AXIOM_GH_BIN=" + filepath.Join(bin, testRuntimeName("gh")),
			}
			if runtime.GOOS == "windows" {
				environment = append(environment, "SYSTEMROOT="+os.Getenv("SYSTEMROOT"), "PATHEXT=.COM;.EXE;.BAT;.CMD")
			}
			before := snapshotTrees(t, base)
			for _, path := range [][]string{
				nil, {"project"}, {"project", "configure"}, {"project", "context"}, {"project", "context", "session-set"},
				{"work-item", "create"}, {"workflow", "start"}, {"integration", "remove"},
				{"runtime"}, {"runtime", "codex", "install"}, {"runtime", "claude", "install"}, {"runtime", "profile", "preview"},
				{"first-run"}, {"skill", "inspect"}, {"artifact", "cleanup"}, {"recovery", "apply"}, {"upgrade"}, {"version"},
			} {
				var baseline []byte
				for _, helpFlag := range []string{"--help", "-h"} {
					args := append(append([]string{"--json"}, path...), helpFlag)
					command := exec.Command(binary, args...)
					command.Dir, command.Env = cwd, environment
					var stderr bytes.Buffer
					command.Stderr = &stderr
					output, err := command.Output()
					if err != nil {
						t.Fatalf("%v: %v stdout=%s stderr=%s", args, err, output, &stderr)
					}
					if stderr.Len() != 0 || !strings.Contains(string(output), "Usage:\n") || json.Valid(output) {
						t.Fatalf("%v did not yield isolated human help: stdout=%s stderr=%s", args, output, &stderr)
					}
					if baseline == nil {
						baseline = output
					} else if !bytes.Equal(baseline, output) {
						t.Errorf("help aliases differ for %v", path)
					}
				}
			}
			for _, mode := range []string{"--json", "--human"} {
				command := exec.Command(binary, mode, "runtime", "codex", "unknown", "--help")
				command.Dir, command.Env = cwd, environment
				output, err := command.CombinedOutput()
				exit, ok := err.(*exec.ExitError)
				if !ok || exit.ExitCode() != cli.ExitFailure {
					t.Fatalf("unknown command exit: %v output=%s", err, output)
				}
				if !strings.Contains(string(output), "axiom runtime codex --help") {
					t.Errorf("missing nearest command guidance: %s", output)
				}
				if mode == "--json" && !json.Valid(output) {
					t.Errorf("invalid JSON: %s", output)
				}
			}
			for _, alias := range []string{"--help", "-h"} {
				for _, mode := range []string{"--json", "--human"} {
					for _, test := range []struct {
						args  []string
						valid bool
					}{
						{[]string{"--session=Session_1", "project", "list", alias}, true},
						{[]string{"--session=", "project", "list", alias}, false},
						{[]string{"--session", "bad/value", "project", "list", alias}, false},
						{[]string{alias, "--session"}, false},
						{[]string{"--session=one", "--session=two", "project", "list", alias}, false},
						{[]string{"--session", alias, "project", "list", alias}, true},
					} {
						args := append([]string{mode}, test.args...)
						command := exec.Command(binary, args...)
						command.Dir, command.Env = cwd, environment
						output, err := command.CombinedOutput()
						if test.valid {
							if err != nil || !strings.Contains(string(output), "Usage:") || json.Valid(output) {
								t.Fatalf("session help %v: %v %s", args, err, output)
							}
						} else {
							exit, ok := err.(*exec.ExitError)
							if !ok || exit.ExitCode() != cli.ExitFailure || !strings.Contains(string(output), "validation_failure") || !strings.Contains(string(output), cli.HelpCommand(test.args)) || mode == "--json" && !json.Valid(output) {
								t.Fatalf("invalid session %v: %v %s", args, err, output)
							}
						}
					}
				}
			}
			if after := snapshotTrees(t, base); !bytes.Equal(before, after) {
				t.Fatal("help modified home, state, skills, cwd or executed a Runtime/Provider")
			}
		})
	}
}
