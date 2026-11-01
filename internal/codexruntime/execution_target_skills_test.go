package codexruntime

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkItemRunExecutionTargetParity(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		content, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	domain := read("axiom-work-item")
	section := func(text, begin, end string) string {
		t.Helper()
		_, remainder, ok := strings.Cut(text, begin)
		if !ok {
			t.Fatalf("missing run boundary %q", begin)
		}
		block, _, ok := strings.Cut(remainder, end)
		if !ok {
			t.Fatalf("missing run boundary %q", end)
		}
		return block
	}
	block := section(domain, "## run\n\n", "\n## status")
	for _, required := range []string{"exact linked Work Item", "Project-scoped Repository", "Lingo resolves and validates selectors", "persisted Project", "`--execution`", "Old start/advance/fact/resume"} {
		if !strings.Contains(block, required) {
			t.Fatalf("shared canonical run missing %q", required)
		}
	}

}

func TestExecutionTargetPreviousSkillSetOwnedByBothRuntimes(t *testing.T) {
	// Full eight-skill manifest immediately before #137; pinning the whole
	// revision catches incomplete history even when changed skills are known.
	const baseline = "b6087a7adfe5c1edaad7ef78da6b180ce8bc62c385a18f7090fe946ac0aa5159"
	var previous skillSetRevision
	for _, revision := range sharedSkillHistory {
		digest, err := revision.manifestDigest()
		if err == nil && digest == baseline {
			previous = revision
		}
	}
	if len(previous.skills) != 8 {
		t.Fatal("complete pre-execution-target skill set missing from shared history")
	}
	for _, runtime := range []integration{codexIntegration, claudeIntegration} {
		root := privateSkillRoot(t)
		for name, digest := range previous.skills {
			if !runtime.knownDigest(name, digest) {
				t.Fatalf("%s lost ownership of %s", runtime.runtime, name)
			}
		}
		wire, err := runtime.receiptFor(root, previous)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, receiptName), wire, 0o600); err != nil {
			t.Fatal(err)
		}
		if !runtime.matchesLegacyReceipt(root) {
			t.Fatalf("%s lost ownership of baseline receipt", runtime.runtime)
		}
	}
}
