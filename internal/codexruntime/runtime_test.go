package codexruntime

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedSkillsUseSupportedNamesAndThinEntrypoints(t *testing.T) {
	validName := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	for _, name := range skillNames {
		if !validName.MatchString(name) {
			t.Fatalf("unsupported skill name: %q", name)
		}
		content, err := skillFiles.ReadFile("skills/" + name + "/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if !strings.Contains(text, "name: "+name) || !strings.Contains(text, "lingo --json") {
			t.Fatalf("skill is not a named thin Lingo entrypoint: %s", name)
		}
	}
}

func TestInstallUpgradesExactPriorAxiomSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	directory := filepath.Join(root, "axiom-project-show")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	legacy := `---
name: axiom-project-show
description: Inspect or resolve a configured Axiom Project through Lingo from any working directory.
---

# Show Axiom Project

Collect a Project slug or ID when absent. Run ` + "`lingo project show`" + ` with that
selector and report its structured result. Never infer a repository from Codex's
current working directory.
`
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Applied {
		t.Fatalf("upgrade = %#v", got)
	}
	data, err := os.ReadFile(filepath.Join(directory, "SKILL.md"))
	if err != nil || !strings.Contains(string(data), "lingo --json project show") {
		t.Fatalf("skill not upgraded: %q, %v", data, err)
	}
}

func TestInstallAndInspectGlobalSkills(t *testing.T) {
	service, err := New(filepath.Join(t.TempDir(), "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Inspect(context.Background()); got.Status != Missing {
		t.Fatalf("initial status = %#v", got)
	}
	if got := service.Install(context.Background()); got.Status != Applied || got.Category != "codex_configured" {
		t.Fatalf("install = %#v", got)
	}
	if got := service.Inspect(context.Background()); got.Status != Ready || got.Category != "codex_ready" {
		t.Fatalf("inspect = %#v", got)
	}
	if got := service.Install(context.Background()); got.Status != Unchanged || got.Category != "codex_already_configured" {
		t.Fatalf("rerun = %#v", got)
	}
}

func TestInstallRefusesConflictAndRollsBackCurrentAttempt(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(filepath.Join(root, "axiom-work-item-create"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "axiom-work-item-create", "SKILL.md"), []byte("unowned"), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "codex_skill_conflict" {
		t.Fatalf("install = %#v", got)
	}
	for _, name := range []string{"axiom-project-configure", "axiom-project-show"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("partial skill retained: %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "axiom-work-item-create", "SKILL.md"))
	if err != nil || string(data) != "unowned" {
		t.Fatalf("conflicting skill changed: %q, %v", data, err)
	}
}

func TestInstallRejectsSymlinkSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "axiom-project-configure")); err != nil {
		t.Fatal(err)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed {
		t.Fatalf("symlink accepted: %#v", got)
	}
}
