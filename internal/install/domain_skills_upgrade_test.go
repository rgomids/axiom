package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/compatibility"
)

// Issue #229: v0.6.0 was the last release whose archive carried six skills.
// An owned upgrade from such an installation to the eight-skill candidate
// keeps the compatibility skills, replaces only the owned axiom-work-item-run
// text whose start protocol #140 changed, creates the two domain skills, and
// never adopts content Axiom did not publish.

const v060SkillManifestSHA256 = "d7190ebb175652c1080288a2f293a1ba70d101d7874880c6eca91b1092dedc3a"

var (
	sixReleaseSkills = skillNames[:6]
	domainSkillNames = []string{"axiom-project", "axiom-work-item"}
)

// v060SkillManifest is the skills-manifest.txt of the published v0.6.0
// archives: the six skills in sorted order with the digests of their
// unchanged compatibility text.
func v060SkillManifest(t *testing.T, skills map[string][]byte) []byte {
	t.Helper()
	manifest := "formatVersion=1\nskillSetVersion=1\nbinaryCompatibility=1\n"
	for _, name := range slices.Sorted(slices.Values(sixReleaseSkills)) {
		manifest += "skill." + name + "=" + digest(skills[name]) + "\n"
	}
	if digest([]byte(manifest)) != v060SkillManifestSHA256 {
		t.Fatalf("reconstructed v0.6.0 manifest differs from the published one:\n%s", manifest)
	}
	return []byte(manifest)
}

// installSixSkillRelease mirrors what the v0.6.0 installer and first-run left:
// the binary, its installation receipt naming the six-skill manifest, and a
// Codex root holding the six skills plus v0.6.0's Codex skill-set receipt.
func installSixSkillRelease(t *testing.T) (installation, string) {
	t.Helper()
	t.Cleanup(func() { hostRow = defaultHostRow })
	hostRow = func() string { return testRow }
	self := selfBundle(t, "0.6.0")
	previous := &bundle{version: "0.6.0", binary: []byte("binary 0.6.0\n"), skills: map[string][]byte{}}
	for _, name := range sixReleaseSkills {
		previous.skills[name] = self.skills[name]
		if slices.Contains(changedCompatibilitySkills, name) {
			previous.skills[name] = v060Skill(t, name)
		}
	}
	lines, _, err := parseMetadata(newBundle("0.6.0", nil).contents()["release-metadata.txt"])
	if err != nil {
		t.Fatal(err)
	}
	base := privateDirectory(t, t.TempDir(), "install")
	target := Target{BinaryDir: privateDirectory(t, base, "bin"), ReceiptDir: privateDirectory(t, base, "receipts"), State: compatibility.Roots{State: filepath.Join(base, "state"), Projects: filepath.Join(base, "projects")}}
	released := Candidate{ArchiveSHA256: digest([]byte("v0.6.0 archive")), Binary: previous.binary, Metadata: lines, SkillManifestSHA256: digest(v060SkillManifest(t, previous.skills))}
	writeFile(t, filepath.Join(target.BinaryDir, binaryName), previous.binary, 0o700)
	writeFile(t, filepath.Join(target.ReceiptDir, receiptName), expectedReceipt(filepath.Join(target.BinaryDir, binaryName), released, "2026-10-01T12:00:00Z"), 0o600)
	installed := installation{target: target, current: previous}
	root := installed.withSkillsRoot(t, previous.skills)
	writeFile(t, filepath.Join(root, codexruntime.SkillSetReceiptName), publishedCodexReceipt(t, "v0.6.0"), 0o600)
	installed.target.Self = selfBuildFor("1.1.0")
	return installed, root
}

// changedCompatibilitySkills are the six-release skills whose embedded text
// differs from what v0.6.0 published (#140 changed run; #230 changed status).
var changedCompatibilitySkills = []string{"axiom-work-item-run", "axiom-work-item-status"}

func v060Skill(t *testing.T, name string) []byte {
	t.Helper()
	wire, err := os.ReadFile(filepath.Join("..", "codexruntime", "testdata", "published-skills", "v0.6.0", name, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func TestUpgradeFromSixSkillReleaseAddsOnlyDomainSkills(t *testing.T) {
	installed, root := installSixSkillRelease(t)
	candidate := installed.candidate(t, selfBundle(t, "1.1.0"))
	preview, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || preview.Skills != SkillsPublish || preview.SourceVersion != "0.6.0" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	kinds := []string{}
	for _, effect := range preview.Effects {
		kinds = append(kinds, strings.TrimSuffix(effect.Kind+":"+effect.Name, ":"))
		replacesChanged := slices.Contains(changedCompatibilitySkills, effect.Name) && effect.Expected == digest(v060Skill(t, effect.Name))
		if effect.Kind == "skill" && !replacesChanged && (!slices.Contains(domainSkillNames, effect.Name) || effect.Expected != absentRevision) {
			t.Fatalf("upgrade touches compatibility skill or replaces content: %+v", effect)
		}
	}
	if want := []string{"binary", "receipt", "skill:axiom-work-item-run", "skill:axiom-work-item-status", "skill:axiom-project", "skill:axiom-work-item", "skill_receipt"}; !slices.Equal(kinds, want) {
		t.Fatalf("effects=%v want %v", kinds, want)
	}
	if last := preview.Effects[len(preview.Effects)-1]; last.Expected != digest(publishedCodexReceipt(t, "v0.6.0")) {
		t.Fatalf("receipt effect=%+v", last)
	}
	authority, _ := Authorize(preview, preview.Digest)
	result, err := NewService().Apply(context.Background(), preview, authority)
	if err != nil || result.Status != "success" || result.SkillReceipt != SkillReceiptCurrent {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	assertSkills(t, root, candidate.SkillFiles)
	service, _ := codexruntime.New(root)
	if status := service.Inspect(context.Background()); status.Status != codexruntime.Ready || len(status.Skills) != len(skillNames) {
		t.Fatalf("codex status=%+v", status)
	}
	again, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || len(again.Effects) != 0 || again.Skills != SkillsMatch {
		t.Fatalf("rerun preview=%+v err=%v", again, err)
	}
}

func TestUpgradeFromSixSkillReleaseResumesAtEachDomainSkill(t *testing.T) {
	for _, stop := range domainSkillNames {
		t.Run(stop, func(t *testing.T) {
			installed, root := installSixSkillRelease(t)
			candidate := installed.candidate(t, selfBundle(t, "1.1.0"))
			preview, _ := NewService().Preview(context.Background(), installed.target, candidate)
			authority, _ := Authorize(preview, preview.Digest)
			interrupted := Service{afterEffect: func(label string) error {
				if label == "skill:"+stop {
					return errors.New("injected interruption")
				}
				return nil
			}}
			if result, err := interrupted.Apply(context.Background(), preview, authority); err == nil || result.Status != "partial" {
				t.Fatalf("interrupted result=%+v err=%v", result, err)
			}
			marker := read(t, filepath.Join(installed.target.ReceiptDir, markerName))
			for _, name := range domainSkillNames {
				if !strings.Contains(marker, "skill."+name+"="+absentRevision+"\n") {
					t.Fatalf("marker lacks absent expectation for %s: %s", name, marker)
				}
			}
			resume, err := NewService().Preview(context.Background(), installed.target, candidate)
			if err != nil || !resume.Resume {
				t.Fatalf("resume=%+v err=%v", resume, err)
			}
			authority, _ = Authorize(resume, resume.Digest)
			if result, err := NewService().Apply(context.Background(), resume, authority); err != nil || result.Status != "success" {
				t.Fatalf("resumed result=%+v err=%v", result, err)
			}
			assertSkills(t, root, candidate.SkillFiles)
			if again, err := NewService().Preview(context.Background(), installed.target, candidate); err != nil || len(again.Effects) != 0 || again.Resume {
				t.Fatalf("after resume preview=%+v err=%v", again, err)
			}
		})
	}
}

func TestUpgradeFromSixSkillReleaseNeverAdoptsForeignOrModifiedSkills(t *testing.T) {
	for name, artifact := range map[string]string{
		"foreign domain skill":         "axiom-project",
		"modified compatibility skill": "axiom-project-show",
	} {
		t.Run(name, func(t *testing.T) {
			installed, root := installSixSkillRelease(t)
			if err := os.MkdirAll(filepath.Join(root, artifact), 0o700); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(root, artifact, "SKILL.md"), []byte("operator content\n"), 0o600)
			before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
			if _, err := NewService().Preview(context.Background(), installed.target, installed.candidate(t, selfBundle(t, "1.1.0"))); category(err) != "skill_conflict" {
				t.Fatalf("error=%v", err)
			}
			if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
				t.Fatal("refused upgrade changed owned state")
			}
		})
	}
}

// An installed eight-skill set reconstructs the manifest that
// build-release-archives.sh writes (sorted), so the next upgrade still
// recognizes the whole set as owned.
func TestInstalledSkillManifestUsesReleaseOrderForDomainSkills(t *testing.T) {
	self := selfBundle(t, "1.1.0")
	inventory := codexruntime.UpgradeInventory{Configured: true}
	release := "formatVersion=1\nskillSetVersion=1\nbinaryCompatibility=1\n"
	for _, name := range skillNames {
		inventory.Skills = append(inventory.Skills, codexruntime.UpgradeSkill{Name: name, SHA256: digest(self.skills[name])})
	}
	for _, name := range slices.Sorted(slices.Values(skillNames)) {
		release += "skill." + name + "=" + digest(self.skills[name]) + "\n"
	}
	if slices.IsSorted(skillNames) {
		t.Fatal("inventory order no longer differs from release order; this test no longer guards anything")
	}
	if got := installedSkillManifest(inventory); got != digest([]byte(release)) {
		t.Fatalf("eight-skill manifest digest = %s, release manifest %s", got, digest([]byte(release)))
	}
}

// The candidate archive's closed skill set is exactly the embedded one.
func TestArchiveSkillSetIsTheEmbeddedSkillSet(t *testing.T) {
	embedded, err := codexruntime.EmbeddedSkillDigests()
	if err != nil {
		t.Fatal(err)
	}
	if len(embedded) != len(skillNames) {
		t.Fatalf("embedded=%d archive=%d", len(embedded), len(skillNames))
	}
	for _, name := range skillNames {
		if _, ok := embedded[name]; !ok {
			t.Fatalf("archive skill %s is not embedded", name)
		}
	}
	if !slices.Equal(sixReleaseSkills, []string{"axiom-project-configure", "axiom-project-list", "axiom-project-show", "axiom-work-item-create", "axiom-work-item-run", "axiom-work-item-status"}) {
		t.Fatalf("published skills are no longer a prefix of the archive set: %v", skillNames)
	}
}
