package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestExecutableSkillDiscoveryWithoutStateOrWorkflow(t *testing.T) {
	binary := filepath.Join(t.TempDir(), testExecutableName("axiom"))
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, output)
	}
	root := t.TempDir()
	for _, skill := range []string{"axiom-project-configure", "axiom-project-list", "axiom-project-show", "axiom-work-item-create", "axiom-work-item-run", "axiom-work-item-status"} {
		t.Run(skill, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, binary, "--json", "skill", "inspect", skill)
			command.Dir = root
			command.Env = append(os.Environ(), "LINGO_PROJECTS_ROOT=invalid-relative-root", "LINGO_STATE_ROOT=", "AXIOM_GH_BIN="+filepath.Join(root, "missing-gh"), "AXIOM_CODEX_SKILLS_ROOT="+filepath.Join(root, "skills"), "CLAUDE_CONFIG_DIR="+filepath.Join(root, "claude"))
			// Keep stdin open with no data: a guided path would block and hit the timeout.
			input, err := command.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			if err := command.Run(); err != nil {
				t.Fatalf("inspect: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
			}
			var got struct {
				Status string `json:"status"`
				Skill  struct {
					Name     string `json:"name"`
					Commands []struct {
						Arguments []map[string]any `json:"arguments"`
					} `json:"commands"`
				} `json:"skill"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != "success" || got.Skill.Name != skill || stderr.Len() != 0 {
				t.Fatalf("output=%s stderr=%s", stdout.String(), stderr.String())
			}
			for _, cmd := range got.Skill.Commands {
				for _, arg := range cmd.Arguments {
					if _, ok := arg["required"].(bool); !ok {
						t.Fatalf("required boolean absent: %v", arg)
					}
				}
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("inspection wrote state: %v %v", entries, err)
			}
		})
	}
}
