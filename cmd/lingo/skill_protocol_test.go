package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
)

// workItemRunStartProtocol reads the start protocol that the installed
// axiom-work-item-run skill prescribes: the fenced command block it teaches.
// Black-box tests execute those exact templates, so the skill text and the
// binary contract cannot drift apart.
func workItemRunStartProtocol(t *testing.T) [][]string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "skills")
	service, err := codexruntime.New(root)
	if err != nil {
		t.Fatal(err)
	}
	if result := service.Install(context.Background()); result.Status != codexruntime.Applied {
		t.Fatalf("install skills: %+v", result)
	}
	content, err := os.ReadFile(filepath.Join(root, "axiom-work-item-run", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, block, found := strings.Cut(string(content), "Starting is a reviewed two-step protocol")
	if !found {
		t.Fatal("skill has no start protocol")
	}
	_, block, _ = strings.Cut(block, "```text\n")
	block, _, _ = strings.Cut(block, "```")
	protocol := [][]string{}
	for _, line := range strings.Split(strings.TrimSpace(block), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != "axiom" || fields[1] != "--json" {
			t.Fatalf("protocol line %q is not an axiom --json command", line)
		}
		protocol = append(protocol, fields[2:])
	}
	if len(protocol) != 3 {
		t.Fatalf("protocol = %v", protocol)
	}
	return protocol
}

// expandProtocol substitutes the skill's placeholders; an unknown placeholder
// fails, so the skill cannot require an input the caller cannot supply.
func expandProtocol(t *testing.T, template []string, selectors []string, values map[string]string) []string {
	t.Helper()
	args := []string{}
	for _, field := range template {
		switch {
		case field == "<selectors>":
			args = append(args, selectors...)
		case strings.HasPrefix(field, "<") && strings.HasSuffix(field, ">"):
			value, ok := values[strings.Trim(field, "<>")]
			if !ok {
				t.Fatalf("protocol placeholder %s has no value", field)
			}
			args = append(args, value)
		default:
			args = append(args, field)
		}
	}
	return args
}
