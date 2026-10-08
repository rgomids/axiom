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
	alias := read("axiom-work-item-run")
	start := "Require an explicit exact Work Item on every invocation."
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
	aliasBlock := section(alias, start, "\nInvoke only")
	if strings.TrimSpace(block) != strings.TrimSpace(start+aliasBlock) {
		t.Fatal("domain run and compatibility alias disagree")
	}
	assertReviewedRuntimePreview(t, "shared run", block)
	for _, required := range []string{
		"Project is optional", "explicit > session > default", "`--session <id>`",
		"CWD, Git, conversation, or recent activity", "authoritative resolver and validator",
		"`projectId`, `projectSource`", "`repositoryKey`, `provider`, `resource`, `externalId`, and `workItem`",
		"binds this full target", "Runtime/Profile policy", "repeat the identical selectors, `--session`",
		"resolved Project UUID is the same", "persisted exact Project UUID", "`workflow.gateCommand`",
	} {
		if !strings.Contains(block, required) {
			t.Fatalf("shared run missing %q", required)
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
