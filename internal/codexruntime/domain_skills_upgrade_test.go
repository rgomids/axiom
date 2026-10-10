package codexruntime

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// The pre-MVP decision supersedes #229's additive installation: historical
// owned six-skill roots converge to three canonical skills without deleting edits.

var domainSkills = []string{"axiom-project", "axiom-work-item", "axiom-workflow"}

// sixSkillRevision is the v0.6.0 skill set: the last six-skill
// sharedSkillHistory revision, pinned against the receipts v0.6.0 really wrote.
func sixSkillRevision(t *testing.T) skillSetRevision {
	t.Helper()
	var revision skillSetRevision
	for _, candidate := range sharedSkillHistory {
		if len(candidate.skills) == 6 {
			revision = candidate
		}
	}
	if len(revision.skills) != 6 {
		t.Fatal("no six-skill shared revision")
	}
	for _, name := range domainSkills {
		if _, ok := revision.skills[name]; ok {
			t.Fatalf("six-skill revision already lists %s", name)
		}
	}
	return revision
}

// seedSixSkillRoot writes the v0.6.0 skill files and that release's own
// receipt for service's Runtime, as v0.6.0 first-run left the root.
func seedSixSkillRoot(t *testing.T, service Service, root string) {
	t.Helper()
	revision := sixSkillRevision(t)
	for name, digest := range revision.skills {
		content := v060SkillText(t, name)
		if digestOf(content) != digest {
			t.Fatalf("%s compatibility text drifted from v0.6.0", name)
		}
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name, "SKILL.md"), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	receipt := publishedReceipt(t, "v0.6.0", service.Runtime(), root)
	if wire, err := service.integration.receiptFor(root, revision); err != nil || string(wire) != string(receipt) {
		t.Fatalf("%s v0.6.0 receipt is not the six-skill revision receipt: %v", service.Runtime(), err)
	}
	if err := os.WriteFile(filepath.Join(root, receiptName), receipt, 0o600); err != nil {
		t.Fatal(err)
	}
}

func v060SkillText(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", "published-skills", "v0.6.0", name, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func runtimeServices(t *testing.T) []func(string) (Service, error) {
	t.Helper()
	return []func(string) (Service, error){New, NewClaude}
}

func TestSixSkillRootConvergesToOnlyDomainSkills(t *testing.T) {
	for _, newService := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, err := newService(root)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(service.Runtime(), func(t *testing.T) {
			seedSixSkillRoot(t, service, root)
			before := service.Inspect(context.Background())
			if before.Status != Missing || before.Receipt != ReceiptLegacy || len(before.Conflicts) != 0 {
				t.Fatalf("six-skill root before = %#v", before)
			}
			installOrFail(t, service, Applied)
			for name := range sixSkillRevision(t).skills {
				if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
					t.Fatalf("retired %s remains: %v", name, err)
				}
			}
			if got := service.Inspect(context.Background()); got.Status != Ready || got.Receipt != ReceiptCurrent || len(got.Skills) != len(skillNames) {
				t.Fatalf("after install = %#v", got)
			}
			tree := skillTree(t, root)
			installOrFail(t, service, Unchanged)
			if skillTree(t, root) != tree {
				t.Fatal("rerun changed a converged root")
			}
		})
	}
}

func TestSixSkillRootNeverAdoptsForeignOrModifiedContent(t *testing.T) {
	cases := map[string]struct {
		artifact, content, state string
	}{
		"foreign domain skill":         {"axiom-project", "operator skill\n", "foreign"},
		"modified compatibility skill": {"axiom-project-show", "operator edit\n", "modified"},
	}
	for name, test := range cases {
		for _, newService := range runtimeServices(t) {
			root := privateSkillRoot(t)
			service, err := newService(root)
			if err != nil {
				t.Fatal(err)
			}
			t.Run(name+"/"+service.Runtime(), func(t *testing.T) {
				seedSixSkillRoot(t, service, root)
				directory := filepath.Join(root, test.artifact)
				if err := os.MkdirAll(directory, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte(test.content), 0o600); err != nil {
					t.Fatal(err)
				}
				before := skillTree(t, root)
				// The install lock is persistent coordination state, not content.
				lock := filepath.Join(root, installLockName)
				got := service.Install(context.Background())
				if got.Status != Failed || got.Category != service.integration.category("skill_conflict") || len(got.Conflicts) != 1 || got.Conflicts[0] != (Conflict{Artifact: test.artifact + "/SKILL.md", State: test.state, Digest: digestOf([]byte(test.content))}) {
					t.Fatalf("install = %#v", got)
				}
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
				if after := skillTree(t, root); after != before {
					t.Fatalf("refused convergence changed the root:\n%s\n---\n%s", before, after)
				}
			})
		}
	}
}

// Codex and Claude roots configured by v0.6.0 converge to the same embedded
// skill bytes.
func TestCodexAndClaudeSixSkillRootsConvergeToSameSkillSet(t *testing.T) {
	trees := []string{}
	for _, newService := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, err := newService(root)
		if err != nil {
			t.Fatal(err)
		}
		seedSixSkillRoot(t, service, root)
		installOrFail(t, service, Applied)
		files := ""
		for _, name := range skillNames {
			wire, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
			if err != nil {
				t.Fatal(err)
			}
			files += name + " " + digestOf(wire) + "\n"
		}
		trees = append(trees, files)
	}
	if trees[0] != trees[1] {
		t.Fatalf("Codex and Claude diverge:\n%s\n%s", trees[0], trees[1])
	}
}

func seedPublishedV015Root(t *testing.T, service Service, root string) {
	t.Helper()
	for _, name := range []string{"axiom-project", "axiom-work-item"} {
		wire, err := os.ReadFile(filepath.Join("testdata", "published-skills", "v0.15.0", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name, "SKILL.md"), wire, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, receiptName), publishedReceipt(t, "v0.15.0", service.Runtime(), root), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPublishedV015ConvergesToThreeSkillsInBothRuntimes(t *testing.T) {
	for _, constructor := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, err := constructor(root)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(service.Runtime(), func(t *testing.T) {
			seedPublishedV015Root(t, service, root)
			oldReceipt, err := os.ReadFile(filepath.Join(root, receiptName))
			if err != nil {
				t.Fatal(err)
			}
			if before := service.Inspect(context.Background()); before.Receipt != ReceiptLegacy || len(before.Conflicts) != 0 {
				t.Fatalf("before: %+v", before)
			}
			installOrFail(t, service, Applied)
			if after := service.Inspect(context.Background()); after.Status != Ready || after.Receipt != ReceiptCurrent || len(after.Skills) != 3 {
				t.Fatalf("after: %+v", after)
			}
			for _, name := range skillNames {
				actual, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
				if err != nil {
					t.Fatal(err)
				}
				expected, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(actual, expected) {
					t.Fatalf("%s embedded bytes differ", name)
				}
			}
			tree := skillTree(t, root)
			installOrFail(t, service, Unchanged)
			if skillTree(t, root) != tree {
				t.Fatal("reinstall changed root")
			}
			if !bytes.Equal(oldReceipt, publishedReceipt(t, "v0.15.0", service.Runtime(), root)) {
				t.Fatal("historical fixture changed")
			}
		})
	}
}

func TestPublishedV015RefusesForeignWorkflowWithoutContentEffects(t *testing.T) {
	for _, constructor := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, err := constructor(root)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(service.Runtime(), func(t *testing.T) {
			seedPublishedV015Root(t, service, root)
			if err := os.Mkdir(filepath.Join(root, "axiom-workflow"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "axiom-workflow", "SKILL.md"), []byte("operator-owned workflow\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			before := skillTree(t, root)
			result := service.Install(context.Background())
			if result.Status != Failed || result.Category != service.integration.category("skill_conflict") {
				t.Fatalf("%+v", result)
			}
			if err := os.Remove(filepath.Join(root, installLockName)); err != nil {
				t.Fatal(err)
			}
			if skillTree(t, root) != before {
				t.Fatal("conflict changed content")
			}
		})
	}
}
