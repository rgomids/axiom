package codexruntime

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
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
	if got := service.Inspect(context.Background()); got.Status != Missing || len(got.Skills) != 5 || got.Skills[0].State != "missing" {
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
	if err := os.Remove(filepath.Join(service.root, receiptName)); err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Applied || got.Category != "codex_configured" {
		t.Fatalf("receipt completion = %#v", got)
	}
}

func TestInstallRefusesAmbiguousLegacyLockState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".axiom-skill-set.lock"), 0o700); err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "recovery_required" {
		t.Fatalf("ambiguous lock = %#v", got)
	}
	if info, err := os.Stat(filepath.Join(root, ".axiom-skill-set.lock")); err != nil || !info.IsDir() {
		t.Fatalf("ambiguous lock changed: %v, %v", info, err)
	}
	for _, skill := range skillNames {
		if _, err := os.Stat(filepath.Join(root, skill)); !os.IsNotExist(err) {
			t.Fatalf("ambiguous install published %s: %v", skill, err)
		}
	}
}

func TestInstallPreservesUnknownLockContent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, installLockName)
	if err := os.WriteFile(lock, []byte("unknown\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "recovery_required" {
		t.Fatalf("unknown lock = %#v", got)
	}
	wire, err := os.ReadFile(lock)
	if err != nil || string(wire) != "unknown\n" {
		t.Fatalf("unknown lock changed: %q, %v", wire, err)
	}
}

func TestInstallProcessLockRefusesActiveAndResumesAfterSIGKILL(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	command := exec.Command(os.Args[0], "-test.run=^TestInstallProcessHelper$")
	command.Env = append(os.Environ(), "AXIOM_CODEX_INSTALL_HELPER=hold", "AXIOM_CODEX_INSTALL_ROOT="+root)
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "locked" {
		t.Fatalf("lock barrier not reached: %q, %v", line, err)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "codex_skill_install_concurrent" {
		t.Fatalf("active lock = %#v", got)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = command.Wait()
	if got := service.Install(context.Background()); got.Status != Applied || got.Category != "codex_configured" {
		t.Fatalf("abandoned lock resume = %#v", got)
	}
	if got := service.Inspect(context.Background()); got.Status != Ready {
		t.Fatalf("resumed install = %#v", got)
	}
}

func TestInstallProcessHelper(t *testing.T) {
	if os.Getenv("AXIOM_CODEX_INSTALL_HELPER") != "hold" {
		return
	}
	service, err := New(os.Getenv("AXIOM_CODEX_INSTALL_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	service.afterSkill = func(string) {
		fmt.Fprintln(os.Stdout, "locked")
		_ = os.Stdout.Sync()
		time.Sleep(time.Hour)
	}
	result := service.Install(context.Background())
	if result.Status != Applied {
		t.Fatal(result)
	}
}

func TestInstallAndInspectRejectUnsafeRootPermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o770); err != nil {
		t.Fatal(err)
	}
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "codex_skill_root_unavailable" {
		t.Fatalf("unsafe install root = %#v", got)
	}
	if got := service.Inspect(context.Background()); got.Status != Failed || got.Category != "codex_skill_root_unavailable" {
		t.Fatalf("unsafe inspect root = %#v", got)
	}
	info, err := os.Stat(root)
	if err != nil || info.Mode().Perm() != 0o770 {
		t.Fatalf("unsafe root changed: %v, %v", info, err)
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
	} else if len(got.Skills) != 5 || got.Skills[2].State != "modified_or_foreign" {
		t.Fatalf("conflict detail = %#v", got.Skills)
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

func TestInspectRejectsHardLinkedOwnedSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Applied {
		t.Fatalf("install = %#v", got)
	}
	path := filepath.Join(root, "axiom-project-show", "SKILL.md")
	if err := os.Link(path, filepath.Join(t.TempDir(), "skill-copy")); err != nil {
		t.Fatal(err)
	}
	if got := service.Inspect(context.Background()); got.Status != Missing || got.Skills[1].State != "modified_or_foreign" {
		t.Fatalf("hard link inspection = %#v", got)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "codex_skill_conflict" {
		t.Fatalf("hard link install = %#v", got)
	}
}

func TestInspectRejectsUnsafeSkillPermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := service.Install(context.Background()); got.Status != Applied {
		t.Fatalf("install = %#v", got)
	}
	directory := filepath.Join(root, "axiom-project-show")
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := service.Inspect(context.Background()); got.Status != Missing || got.Skills[1].State != "modified_or_foreign" {
		t.Fatalf("permission inspection = %#v", got)
	}
	if got := service.Install(context.Background()); got.Status != Failed || got.Category != "codex_skill_conflict" {
		t.Fatalf("permission install = %#v", got)
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

func TestSkillManifestIsClosedVersionedAndComplete(t *testing.T) {
	manifest, err := CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.FormatVersion != 1 || manifest.SkillSetVersion != SkillSetVersion || manifest.BinaryCompatibility != BinaryCompatibility || len(manifest.Skills) != 5 {
		t.Fatalf("manifest = %#v", manifest)
	}
	seen := map[string]bool{}
	for _, skill := range manifest.Skills {
		if seen[skill.Name] || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(skill.SHA256) {
			t.Fatalf("invalid skill manifest entry: %#v", skill)
		}
		seen[skill.Name] = true
	}
}

func TestInspectReportsBinaryCompatibilityAndPartialResume(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	service, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	service.afterSkill = func(string) {
		calls++
		if calls == 2 {
			service.afterSkill = nil
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.afterSkill = func(string) {
		calls++
		if calls == 2 {
			cancel()
		}
	}
	if got := service.Install(ctx); got.Status != Partial || got.Category != "codex_skill_install_partial" {
		t.Fatalf("partial install = %#v", got)
	}
	if got := service.Inspect(context.Background()); got.Status != Missing {
		t.Fatalf("partial inspection = %#v", got)
	}
	if got := service.Install(context.Background()); got.Status != Applied {
		t.Fatalf("resume = %#v", got)
	}
	incompatible, err := NewForBinary(root, "2")
	if err != nil {
		t.Fatal(err)
	}
	if got := incompatible.Inspect(context.Background()); got.Status != Incompatible || got.Category != "codex_binary_skill_incompatible" {
		t.Fatalf("compatibility = %#v", got)
	}
}
