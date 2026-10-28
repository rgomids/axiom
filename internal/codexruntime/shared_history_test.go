package codexruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// publishedSharedRevisions pins the manifest digest of every skill set Axiom
// published through the shared Runtime integration. Each must remain either
// the embedded skill set or an entry of sharedSkillHistory, so no Runtime
// loses ownership of a revision it received. Add the new embedded manifest
// digest here whenever the skill text changes.
var publishedSharedRevisions = []string{
	"98c861ec8f4448f5e63299d576a11fd41c6cf3f8d43270bb060bd77e9ba327ec",
	"9edcd211cdf251a9d8ffd6e20e89809ed2ae51ac5d54ce0b9c0d927431fa2a5f",
	"ea2f03b88f72ee19efa6dd2cc18b34c0bdac3c7ef311a6f17b1504664f939f80",
	"5d8859d1dcdf6c60ee169555295d7b803e56d3742def4c90d2ab6f957eb481db",
	"d2b9dd436071e995744b2dc2f025b72ec142b9cfccf3b8c26c9e3a48d56f26a1",
	"7df1ad4de232ea88d3e71928d583c0fc108a72d8e2349864bcb51df3077827a3",
	"50641be48a20ac1d2af5076e708b17b99d977d1dd3fd2de8f50d20bd7d3f4524",
	"8cd8343f62485a83c8b8bed018bdc516dd8619319fc42924bfd94057cee52550",
	"bc34d5350dca38353f2016d818cf2f01453a83eb223b914917aa2605029a241e",
	"b6087a7adfe5c1edaad7ef78da6b180ce8bc62c385a18f7090fe946ac0aa5159",
	"22493d0810b50320ca293a773e34f893dfb1f5440f6c3d6235671f964c28cdf0",
	"dca4a1b02932801b910bd8288c1ffbbe6d158699740c2cbee3aaca5005788008",
	"473a9f20f888c61590009d97e291f72f49952166ab29a4d102e7c45feeb48b04",
	"7d6e1657179e2b4c9d374089d1b66bdf951afa6380dcb241465f0699f8fb09e6",
	"1a39e4386f8e099eb094518541c0cffa27bb28f0776fee1b64a165363bd09fee",
	"edab9912728cb5f897ac6121aa59fca618885582f676b9f1453d9a86e15a8f1a",
	"5616db2310b3dcca9e810bf47df2e3734bb91bcab618539a94da41956f5ff3d3",
	"2d1955609ab515082a1975281a865e1277f666ec4786ea91d1ae1bc457ab5f0a",
	"f1b5227368c98b3de89beb809d9d07b25b7ccc7369c5f63c8624fdbf8242a9e2",
	"b115f8b67efa3b9ae5e50588877e837d0c765f88b277682267ad23a80a2453d3",
	"ee748b4f34002ab29d758260e845bc71a317050275d76ff40d1eeb5fb66fcfc7",
	"90ffb0baa9fa7865b4d61376818d9a34693a3743e734e50144802120e28288cf",
	"36a9f24441cac44ee31d1b234df4aeccda827d80efb5872207d24ca6e648bfd3",
	"63e3d0b1efd87fc550ae9d8afe8ec47a80a07505e8bf387643647ae3be3c8191",
	"25629c20ab4b19db1aee61ee7289f65b3aceec4716da207c9b56fc1ea661ff46",
	"b6462d4a7b045b51859ff03b7c9c78398077bc9828675c28bb8c37e2e8167629",
	"67a16f700e7f8f798c7d739fa176ad42b25251b4b063cebb850563f4092cf8a2",
	"238503af1bf9bdbfb3cc6c054f7641f3992f4aac0cace989126ac6c7670bf3b7",
	"6551bec21b0efa51ebffaed160efc2975243a0b88f8d8b1872ddc5dd72a995e2",
	"ece4e12b21a57503f888f2bcb304a4147f522baefa5c78cf8b4e6529de0a85c8",
	"73e9fa8d6750660c7abaa1da0dee9b38b1f4d7c7e790ce7819d3cd48e60e02dd",
	"d87a6acbe77d27f0d16a77270cfe15129fd0122efb34da0595c3ee0863963ae1",
	"0198dbea20a157852bf131141b480938c2436120d82f87b8bc84b14fc30effb8",
	"e87403814f57b5160254e1fccaffcb64d1b9fba4a61035db9444dc16075a38e3",
	"508f3f1b1a31a403d5deba227cffef981bc67061812394420948290a78c03216",
	"d039a843c6289c3f91af73fab96e9e62c3cc1efafe6f08f07e163c925f5cc4b7",
	"df2d8dcb0f0ab4808240012918d806aa8e7a1472553101bd1ecd54aea25009ce",
	"725010f55b3b9af755ab6520913088e6479074467b3fb15c46086c1e64b3fc0c",
	"e29183f85292f1aeb7fe37ed6a5688e66583e293340d249666208d76b7526696",
	"b3607d79035da730689c871bdfba0f69420bfec367b086a35db3dc18eca7f4d4",
	"c3f11784611e706761d14617b8d91052681fa01b77df6439b747f47f537aa469",
	"0c406c39087e19663b75307e4cd9981844964466b348c29a4e94327dcd55154f",
	"53fad16966ec6b08c171d9f884eeda6c3ed04c6ed1c2ce8dcfbca54eec234aca",
	"0f1af5376d96699f7c9bb8df4b24bf264a3119f03ee86f366165cb0bb1a62680",
}

func TestEveryPublishedSharedRevisionStaysOwned(t *testing.T) {
	current, err := currentRevision()
	if err != nil {
		t.Fatal(err)
	}
	currentDigest, err := current.manifestDigest()
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{currentDigest: true}
	for _, revision := range sharedSkillHistory {
		digest, err := revision.manifestDigest()
		if err != nil {
			t.Fatalf("incomplete shared revision: %v", err)
		}
		known[digest] = true
	}
	for _, digest := range publishedSharedRevisions {
		if !known[digest] {
			t.Fatalf("published skill set %s is neither embedded nor in sharedSkillHistory (embedded manifest is %s): append the replaced revision to sharedSkillHistory", digest, currentDigest)
		}
	}
	if publishedSharedRevisions[len(publishedSharedRevisions)-1] != currentDigest {
		t.Fatalf("embedded skill set %s is not pinned in publishedSharedRevisions", currentDigest)
	}
}

// The Codex-only history predates the shared history and is frozen, so a
// revision cannot be registered for Codex alone.
func TestRuntimeOnlyHistoryIsFrozen(t *testing.T) {
	count := 0
	for _, digests := range legacySkillDigests {
		count += len(digests)
	}
	if count != 28 || len(legacyReceiptWires) != 4 {
		t.Fatalf("Codex-only history changed (%d skills, %d receipts): register new revisions in sharedSkillHistory", count, len(legacyReceiptWires))
	}
	if len(claudeIntegration.legacySkills) != 0 || len(claudeIntegration.legacyReceipts) != 0 {
		t.Fatal("Claude gained a Runtime-only history")
	}
}

// nextRelease simulates the binary of Axiom N+1: every skill text changes and
// revision N, the embedded skill set, is appended to sharedSkillHistory as the
// release discipline requires. It returns the N+1 skill contents.
func nextRelease(t *testing.T) map[string][]byte {
	t.Helper()
	previous, err := currentRevision()
	if err != nil {
		t.Fatal(err)
	}
	next := fstest.MapFS{}
	contents := map[string][]byte{}
	for _, name := range skillNames {
		content, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		changed := append(append([]byte(nil), content...), []byte("\nRevision N+1 wording.\n")...)
		next["skills/"+name+"/SKILL.md"] = &fstest.MapFile{Data: changed, Mode: 0o600}
		contents[name] = changed
	}
	savedFiles, savedHistory := skillFiles, sharedSkillHistory
	skillFiles = next
	sharedSkillHistory = append(append([]skillSetRevision(nil), savedHistory...), previous)
	t.Cleanup(func() { skillFiles, sharedSkillHistory = savedFiles, savedHistory })
	return contents
}

func skillTree(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		line := strings.TrimPrefix(path, root) + " " + info.Mode().String()
		if info.Mode().IsRegular() {
			content, _ := os.ReadFile(path)
			sum := sha256.Sum256(content)
			line += " " + hex.EncodeToString(sum[:])
		}
		lines = append(lines, line)
		return nil
	})
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func installOrFail(t *testing.T, service Service, status Status) {
	t.Helper()
	if got := service.Install(context.Background()); got.Status != status {
		t.Fatalf("%s install = %#v, want %s", service.Runtime(), got, status)
	}
}

func assertConverged(t *testing.T, service Service, root string, contents map[string][]byte) {
	t.Helper()
	for name, content := range contents {
		actual, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
		if err != nil || string(actual) != string(content) {
			t.Fatalf("%s %s not at N+1: %v", service.Runtime(), name, err)
		}
	}
	if got := service.Inspect(context.Background()); got.Status != Ready {
		t.Fatalf("%s inspect = %#v", service.Runtime(), got)
	}
	receipt, err := service.integration.receipt(root)
	if err != nil || !matchesPrivateFile(filepath.Join(root, receiptName), receipt) {
		t.Fatalf("%s receipt not refreshed to N+1: %v", service.Runtime(), err)
	}
	if inventory, err := service.Inventory(context.Background()); err != nil || inventory.State != SkillSetCurrent {
		t.Fatalf("%s inventory = %#v, %v", service.Runtime(), inventory, err)
	}
}

func TestClaudeRevisionNConvergesAfterUpgradeToNPlusOne(t *testing.T) {
	root := filepath.Join(t.TempDir(), "claude", "skills")
	claude, err := NewClaude(root)
	if err != nil {
		t.Fatal(err)
	}
	installOrFail(t, claude, Applied)
	contents := nextRelease(t)
	if inventory, _ := claude.Inventory(context.Background()); inventory.State != SkillSetUpgradable || inventory.Receipt != "legacy" {
		t.Fatalf("revision N inventory under N+1 = %#v", inventory)
	}
	installOrFail(t, claude, Applied)
	assertConverged(t, claude, root, contents)
	before := skillTree(t, root)
	installOrFail(t, claude, Unchanged)
	if skillTree(t, root) != before {
		t.Fatal("idempotent rerun changed the Claude root")
	}
}

func TestModifiedClaudeRevisionNIsRefusedAfterUpgrade(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, root string)
	}{
		{"user edit", func(t *testing.T, root string) {
			path := filepath.Join(root, "axiom-work-item", "SKILL.md")
			content, _ := os.ReadFile(path)
			if err := os.WriteFile(path, append(content, []byte("my local change\n")...), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"foreign skill", func(t *testing.T, root string) {
			if err := os.WriteFile(filepath.Join(root, "axiom-work-item", "SKILL.md"), []byte("---\nname: axiom-work-item-run\n---\nforeign\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"ambiguous ownership", func(t *testing.T, root string) {
			if err := os.WriteFile(filepath.Join(root, "axiom-work-item", "notes.md"), []byte("extra\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"unsafe mode", func(t *testing.T, root string) {
			if err := os.Chmod(filepath.Join(root, "axiom-work-item", "SKILL.md"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "claude", "skills")
			claude, err := NewClaude(root)
			if err != nil {
				t.Fatal(err)
			}
			installOrFail(t, claude, Applied)
			test.mutate(t, root)
			before := skillTree(t, root)
			nextRelease(t)
			if got := claude.Install(context.Background()); got.Status != Failed || got.Category != "claude_skill_conflict" {
				t.Fatalf("install = %#v", got)
			}
			if skillTree(t, root) != before {
				t.Fatalf("refused upgrade changed the Claude root:\n%s\n---\n%s", before, skillTree(t, root))
			}
		})
	}
}

// Axiom N publishes both Runtimes; `axiom upgrade` to N+1 publishes only the
// Codex root and leaves its receipt at N; first-run with N+1 converges both.
func TestCodexAndClaudeConvergeTogetherAfterUpgrade(t *testing.T) {
	home := t.TempDir()
	codexRoot, claudeRoot := filepath.Join(home, ".agents", "skills"), filepath.Join(home, ".claude", "skills")
	codex, err := New(codexRoot)
	if err != nil {
		t.Fatal(err)
	}
	claude, err := NewClaude(claudeRoot)
	if err != nil {
		t.Fatal(err)
	}
	installOrFail(t, codex, Applied)
	installOrFail(t, claude, Applied)
	previous, err := currentRevision()
	if err != nil {
		t.Fatal(err)
	}
	contents := nextRelease(t)
	session, err := codex.LockForUpgrade()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range skillNames {
		if err := session.PublishSkill(name, contents[name], previous.skills[name]); err != nil {
			t.Fatalf("upgrade publish %s: %v", name, err)
		}
	}
	session.Close()
	if got := codex.Inspect(context.Background()); got.Status != Partial {
		t.Fatalf("codex after upgrade = %#v, want stale receipt", got)
	}
	for _, service := range []Service{codex, claude} {
		installOrFail(t, service, Applied)
	}
	assertConverged(t, codex, codexRoot, contents)
	assertConverged(t, claude, claudeRoot, contents)
	codexBefore, claudeBefore := skillTree(t, codexRoot), skillTree(t, claudeRoot)
	for _, service := range []Service{codex, claude} {
		installOrFail(t, service, Unchanged)
	}
	if skillTree(t, codexRoot) != codexBefore || skillTree(t, claudeRoot) != claudeBefore {
		t.Fatal("idempotent rerun changed a Runtime root")
	}
}

// An interrupted Claude install or upgrade leaves some skills at N, one
// already at N+1, one missing and no receipt; the N+1 rerun converges it.
func TestPartialClaudeStateConvergesAfterUpgrade(t *testing.T) {
	root := filepath.Join(t.TempDir(), "claude", "skills")
	claude, err := NewClaude(root)
	if err != nil {
		t.Fatal(err)
	}
	installOrFail(t, claude, Applied)
	contents := nextRelease(t)
	if err := os.Remove(filepath.Join(root, receiptName)); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "axiom-work-item")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "axiom-project", "SKILL.md"), contents["axiom-project"], 0o600); err != nil {
		t.Fatal(err)
	}
	installOrFail(t, claude, Applied)
	assertConverged(t, claude, root, contents)
	installOrFail(t, claude, Unchanged)
}

func TestClaudeReceiptMustBeAxiomEvidenceForThisRoot(t *testing.T) {
	previous, err := currentRevision()
	if err != nil {
		t.Fatal(err)
	}
	elsewhere, err := claudeReceiptBytes("/elsewhere/.claude/skills", previous)
	if err != nil {
		t.Fatal(err)
	}
	codexWire, err := codexReceiptBytes("", previous)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		wire []byte
		mode os.FileMode
	}{
		{"receipt of another root", elsewhere, 0o600},
		{"Codex receipt", codexWire, 0o600},
		{"foreign receipt", []byte("formatVersion=1\nruntime=claude\n"), 0o600},
		{"unsafe receipt mode", nil, 0o644},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "claude", "skills")
			claude, err := NewClaude(root)
			if err != nil {
				t.Fatal(err)
			}
			installOrFail(t, claude, Applied)
			path := filepath.Join(root, receiptName)
			if test.wire != nil {
				if err := os.WriteFile(path, test.wire, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chmod(path, test.mode); err != nil {
				t.Fatal(err)
			}
			before := skillTree(t, root)
			nextRelease(t)
			if got := claude.Install(context.Background()); got.Status != Failed || got.Category != "claude_skill_conflict" {
				t.Fatalf("install = %#v", got)
			}
			if skillTree(t, root) != before {
				t.Fatal("skills or receipt changed beside an unrecognized receipt")
			}
		})
	}
}

// A pre-shared Codex-only revision is still not Claude-owned after an upgrade.
func TestSharedHistoryDoesNotAdoptCodexOnlyRevisionsForClaude(t *testing.T) {
	legacy, err := os.ReadFile(filepath.Join("..", "compatibility", "testdata", "poc-v0.1.0-poc.1", "skills", "axiom-project-show", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "claude", "skills")
	if err := os.MkdirAll(filepath.Join(root, "axiom-project-show"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "axiom-project-show", "SKILL.md"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	nextRelease(t)
	claude, err := NewClaude(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := claude.Install(context.Background()); got.Status != Failed || got.Category != "claude_skill_conflict" {
		t.Fatalf("install = %#v", got)
	}
	if content, _ := os.ReadFile(filepath.Join(root, "axiom-project-show", "SKILL.md")); string(content) != string(legacy) {
		t.Fatal("Codex-only revision was replaced in the Claude root")
	}
}
