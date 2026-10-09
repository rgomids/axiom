package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
)

// Issue #186: the installer runs as the candidate release, so it can publish
// the candidate's Codex skill-set receipt itself instead of ending partial
// with refresh_required after every skill text change.

const selfRevision = "123456789abc"

// selfBundle is a candidate whose skill files are exactly this binary's
// embedded skills, as every real release archive is for its own binary.
func selfBundle(t *testing.T, version string) *bundle {
	t.Helper()
	b := newBundle(version, []byte("binary "+version+"\n"))
	for _, name := range skillNames {
		wire, err := os.ReadFile(filepath.Join("..", "codexruntime", "skills", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		b.skills[name] = wire
	}
	return b
}

func selfBuildFor(version string) Build {
	return Build{Release: true, Version: version, Revision: selfRevision, SourceState: "clean"}
}

// publishedCodexReceipt is the exact Codex receipt a published release wrote.
func publishedCodexReceipt(t *testing.T, tag string) []byte {
	t.Helper()
	wire, err := os.ReadFile(filepath.Join("..", "codexruntime", "testdata", "published-receipts", tag, "codex.receipt"))
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

// upgradeAsCandidate installs 1.0.0 over an older Axiom-owned Codex root
// (earlier owned skill revisions, some skills missing, an earlier receipt) and
// previews 1.1.0 as the candidate binary.
func upgradeAsCandidate(t *testing.T, receipt []byte) (installation, string, Candidate, Preview) {
	t.Helper()
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	legacy := map[string][]byte{}
	for _, name := range []string{"axiom-project-configure", "axiom-project-show", "axiom-work-item-create", "axiom-work-item-run"} {
		wire, err := os.ReadFile(filepath.Join("..", "compatibility", "testdata", "poc-v0.1.0-poc.1", "skills", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		legacy[name] = wire
	}
	root := installed.withSkillsRoot(t, legacy)
	if receipt != nil {
		writeFile(t, filepath.Join(root, codexruntime.SkillSetReceiptName), receipt, 0o600)
	}
	installed.target.Self = selfBuildFor("1.1.0")
	candidate := installed.candidate(t, selfBundle(t, "1.1.0"))
	preview, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	return installed, root, candidate, preview
}

func TestUpgradeAsCandidatePublishesSkillSetReceiptLast(t *testing.T) {
	installed, root, candidate, preview := upgradeAsCandidate(t, publishedCodexReceipt(t, "v0.1.1"))
	want, err := codexruntime.CurrentReceipt()
	if err != nil {
		t.Fatal(err)
	}
	last := preview.Effects[len(preview.Effects)-1]
	if len(preview.Effects) != 2+len(skillNames)+4+1 || last.Kind != "skill_receipt" || last.Expected != digest(publishedCodexReceipt(t, "v0.1.1")) || last.Next != digest(want) || last.Target != filepath.Join(root, codexruntime.SkillSetReceiptName) {
		t.Fatalf("effects=%+v", preview.Effects)
	}
	authority, _ := Authorize(preview, preview.Digest)
	result, err := NewService().Apply(context.Background(), preview, authority)
	if err != nil || result.Status != "success" || result.SkillReceipt != SkillReceiptCurrent || len(result.Ledger) != len(preview.Effects) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if read(t, filepath.Join(root, codexruntime.SkillSetReceiptName)) != string(want) {
		t.Fatal("receipt not published")
	}
	assertMode(t, filepath.Join(root, codexruntime.SkillSetReceiptName), 0o600)
	service, err := codexruntime.New(root)
	if err != nil {
		t.Fatal(err)
	}
	if status := service.Inspect(context.Background()); status.Status != codexruntime.Ready {
		t.Fatalf("codex status after upgrade = %+v", status)
	}
	again, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || len(again.Effects) != 0 {
		t.Fatalf("rerun preview=%+v err=%v", again, err)
	}
	if strays := stageEntries(t, root); len(strays) != 0 {
		t.Fatalf("stage leftovers: %v", strays)
	}
}

func TestUpgradeAsCandidateCreatesAbsentReceiptOnlyForConfiguredRoot(t *testing.T) {
	_, root, _, preview := upgradeAsCandidate(t, nil)
	last := preview.Effects[len(preview.Effects)-1]
	if last.Kind != "skill_receipt" || last.Expected != absentRevision {
		t.Fatalf("effects=%+v", preview.Effects)
	}
	authority, _ := Authorize(preview, preview.Digest)
	if result, err := NewService().Apply(context.Background(), preview, authority); err != nil || result.Status != "success" || result.SkillReceipt != SkillReceiptCurrent {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(filepath.Join(root, codexruntime.SkillSetReceiptName)); err != nil {
		t.Fatal(err)
	}

	unconfigured := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	unconfigured.target.SkillsRoot = filepath.Join(filepath.Dir(unconfigured.target.BinaryDir), "skills")
	unconfigured.target.Self = selfBuildFor("1.1.0")
	plain, err := NewService().Preview(context.Background(), unconfigured.target, unconfigured.candidate(t, selfBundle(t, "1.1.0")))
	if err != nil || plain.Skills != SkillsNotConfigured || len(plain.Effects) != 2 {
		t.Fatalf("unconfigured preview=%+v err=%v", plain, err)
	}
	if _, err := os.Lstat(unconfigured.target.SkillsRoot); !os.IsNotExist(err) {
		t.Fatal("unconfigured Codex root created")
	}
}

func TestUpgradeAsCandidatePreservesUnrecognizedReceipt(t *testing.T) {
	foreign := []byte("formatVersion=1\nskillSetVersion=9\n")
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	root := installed.withSkillsRoot(t, map[string][]byte{"axiom-project-configure": v060Skill(t, "axiom-project-configure")})
	writeFile(t, filepath.Join(root, codexruntime.SkillSetReceiptName), foreign, 0600)
	installed.target.Self = selfBuildFor("1.1.0")
	preview, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, selfBundle(t, "1.1.0")))
	if err == nil || len(preview.Effects) > 2 {
		t.Fatal("foreign receipt permitted retirement")
	}
	if read(t, filepath.Join(root, codexruntime.SkillSetReceiptName)) != string(foreign) {
		t.Fatal("unrecognized receipt changed")
	}
}

func TestUpgradeNotRunningAsCandidateKeepsRefreshRequired(t *testing.T) {
	for name, self := range map[string]Build{
		"other revision": {Release: true, Version: "1.1.0", Revision: "000000000000", SourceState: "clean"},
		"development":    {Release: false, Version: "1.1.0", Revision: selfRevision, SourceState: "clean"},
		"dirty":          {Release: true, Version: "1.1.0", Revision: selfRevision, SourceState: "dirty"},
		"older binary":   selfBuildFor("1.0.0"),
		"unset":          {},
	} {
		t.Run(name, func(t *testing.T) {
			current := newBundle("1.0.0", []byte("old-binary\n"))
			installed := install(t, current)
			installed.withSkillsRoot(t, current.skills)
			installed.target.Self = self
			preview, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, selfBundle(t, "1.1.0")))
			if err != nil {
				t.Fatal(err)
			}
			for _, effect := range preview.Effects {
				if effect.Kind == "skill_receipt" {
					t.Fatalf("receipt derived without proof of candidate identity: %+v", effect)
				}
			}
			authority, _ := Authorize(preview, preview.Digest)
			if result, err := NewService().Apply(context.Background(), preview, authority); err != nil || result.Status != "partial" || result.SkillReceipt != SkillReceiptRefreshRequired {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

// A candidate whose skill files differ from the running binary's embedded
// skills is not this binary, whatever its metadata claims.
func TestUpgradeCandidateIdentityRequiresEmbeddedSkills(t *testing.T) {
	candidate := selfBundle(t, "1.1.0")
	candidate.skills["axiom-work-item"] = []byte("different\n")
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	if selfBuildFor("1.1.0").isCandidate(installed.candidate(t, candidate)) {
		t.Fatal("candidate with different skills accepted as this binary")
	}
	if !selfBuildFor("1.1.0").isCandidate(installed.candidate(t, selfBundle(t, "1.1.0"))) {
		t.Fatal("exact candidate rejected")
	}
}

func TestUpgradeAsCandidateResumesAfterInterruptionAroundReceipt(t *testing.T) {
	// Interrupt after the last skill file (the receipt is still unpublished)
	// and after the receipt itself.
	for _, stop := range []string{"skill:" + skillNames[len(skillNames)-1], "skill_receipt"} {
		t.Run(stop, func(t *testing.T) {
			installed, root, candidate, preview := upgradeAsCandidate(t, publishedCodexReceipt(t, "v0.1.1"))
			interrupted := Service{afterEffect: func(label string) error {
				if label == stop {
					return errors.New("injected interruption")
				}
				return nil
			}}
			authority, _ := Authorize(preview, preview.Digest)
			if result, err := interrupted.Apply(context.Background(), preview, authority); err == nil || result.Status != "partial" {
				t.Fatalf("interrupted result=%+v err=%v", result, err)
			}
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil || !resume.Resume {
				t.Fatalf("resume preview=%+v err=%v", resume, err)
			}
			wantEffects := 5
			if stop == "skill_receipt" {
				wantEffects = 0
			}
			if len(resume.Effects) != wantEffects {
				t.Fatalf("resume effects=%+v", resume.Effects)
			}
			authority, err = Authorize(resume, resume.Digest)
			if err != nil {
				t.Fatal(err)
			}
			if result, err := NewService().Apply(context.Background(), resume, authority); err != nil || result.Status != "success" || result.SkillReceipt != SkillReceiptCurrent {
				t.Fatalf("resumed result=%+v err=%v", result, err)
			}
			if again, err := NewService().Preview(context.Background(), installed.target, candidate); err != nil || len(again.Effects) != 0 || again.Resume {
				t.Fatalf("after resume preview=%+v err=%v", again, err)
			}
			if strays := stageEntries(t, root); len(strays) != 0 {
				t.Fatalf("stage leftovers: %v", strays)
			}
		})
	}
}

func TestUpgradeReceiptStageLeftoverIsRecoveredOnlyWithMarker(t *testing.T) {
	installed, root, candidate, preview := upgradeAsCandidate(t, publishedCodexReceipt(t, "v0.1.1"))
	interrupted := Service{afterEffect: func(label string) error {
		if label == "skill:"+skillNames[len(skillNames)-1] {
			// A crash during receipt staging leaves its private stage behind.
			writeFile(t, filepath.Join(root, codexruntime.UpgradeStagePrefix+"receipt.crashed"), []byte("partial"), 0o600)
			return errors.New("injected interruption")
		}
		return nil
	}}
	authority, _ := Authorize(preview, preview.Digest)
	if _, err := interrupted.Apply(context.Background(), preview, authority); err == nil {
		t.Fatal("interruption not reported")
	}
	resume, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || len(resume.Leftovers) != 1 || !strings.HasSuffix(resume.Leftovers[0], "receipt.crashed") {
		t.Fatalf("resume=%+v err=%v", resume, err)
	}
	authority, _ = Authorize(resume, resume.Digest)
	if result, err := NewService().Apply(context.Background(), resume, authority); err != nil || result.Status != "success" {
		t.Fatalf("resumed result=%+v err=%v", result, err)
	}
	if strays := stageEntries(t, root); len(strays) != 0 {
		t.Fatalf("stage leftovers: %v", strays)
	}

	// Without the operation marker a stray stage is not this operation's.
	writeFile(t, filepath.Join(root, codexruntime.UpgradeStagePrefix+"receipt.unknown"), []byte("x"), 0o600)
	next := selfBundle(t, "1.2.0")
	next.binary = []byte("binary 1.2.0\n")
	installed.target.Self = selfBuildFor("1.2.0")
	if _, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, next)); category(err) != "recovery_required" {
		t.Fatalf("stray stage without marker: %v", err)
	}
}

func stageEntries(t *testing.T, root string) []string {
	t.Helper()
	strays := []string{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && strings.HasPrefix(entry.Name(), codexruntime.UpgradeStagePrefix) {
			strays = append(strays, path)
		}
		return nil
	})
	return strays
}

// The v0.1.1 archive's skills-manifest.txt listed five skills (sha256
// 98ba8954…, recorded as skillManifestSha256 in its installation receipt). A
// root holding exactly that set must reconstruct the same digest even though
// this binary knows a sixth skill name.
func TestInstalledSkillManifestOmitsSkillsAnEarlierReleaseDidNotHave(t *testing.T) {
	inventory := codexruntime.UpgradeInventory{Configured: true, Skills: []codexruntime.UpgradeSkill{
		{Name: "axiom-project-configure", SHA256: "b0d16b482ca3a08c20f8c5d7154573db825a563f458b3bf5284570fa263e4ad3"},
		{Name: "axiom-project-list"},
		{Name: "axiom-project-show", SHA256: "048975860d658e8134eb498c7cd5158104338a2629dd8a8cda4a2ecb0bb99c09"},
		{Name: "axiom-work-item-create", SHA256: "85e4f4badff647a337acf59e96d07909c9a3c47ce5e8df65fac08395c8aac8c4"},
		{Name: "axiom-work-item-run", SHA256: "f0fd487fbbfcfb87bee9906244fb1fedbc0dd5bcbd1fb01594dc3df0c8e5e897"},
		{Name: "axiom-work-item-status", SHA256: "6a139090ff66759b64e630181373c32ff51ad2672d90d26184b6f81e9879b7d9"},
	}}
	if got := installedSkillManifest(inventory); got != "98ba89541007cf2838fd5afeb641477ded380ee361591787bdea395617671ed2" {
		t.Fatalf("five-skill manifest digest = %s", got)
	}
	if got := installedSkillManifest(codexruntime.UpgradeInventory{Skills: []codexruntime.UpgradeSkill{{Name: "axiom-project-list"}}}); got != "" {
		t.Fatalf("empty set manifest = %q", got)
	}
}
