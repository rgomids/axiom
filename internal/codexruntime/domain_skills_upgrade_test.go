package codexruntime

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Issue #229 added the canonical axiom-project and axiom-work-item skills
// after v0.6.0, the last release that published six skills. A Codex or Claude
// root a six-skill release configured must converge additively: the six
// compatibility skills stay, the two domain skills are created, the receipt is
// replaced, and content Axiom never published is never overwritten.

var domainSkills = []string{"axiom-project", "axiom-work-item"}

// sixSkillRevision is the v0.6.0 skill set: the last sharedSkillHistory
// revision, pinned against the receipts v0.6.0 really wrote.
func sixSkillRevision(t *testing.T) skillSetRevision {
	t.Helper()
	revision := sharedSkillHistory[len(sharedSkillHistory)-1]
	if len(revision.skills) != 6 {
		t.Fatalf("last shared revision has %d skills", len(revision.skills))
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
		content, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil || digestOf(content) != digest {
			t.Fatalf("%s compatibility text drifted from v0.6.0: %v", name, err)
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

func runtimeServices(t *testing.T) []func(string) (Service, error) {
	t.Helper()
	return []func(string) (Service, error){New, NewClaude}
}

func TestSixSkillRootConvergesAdditivelyToDomainSkills(t *testing.T) {
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
			for _, skill := range before.Skills {
				want := "equivalent"
				if slices.Contains(domainSkills, skill.Name) {
					want = "missing"
				}
				if skill.State != want {
					t.Fatalf("%s state=%s want %s", skill.Name, skill.State, want)
				}
			}
			compatibility := map[string]string{}
			for name := range sixSkillRevision(t).skills {
				wire, _ := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
				compatibility[name] = string(wire)
			}
			installOrFail(t, service, Applied)
			for name, content := range compatibility {
				if wire, _ := os.ReadFile(filepath.Join(root, name, "SKILL.md")); string(wire) != content {
					t.Fatalf("compatibility skill %s changed", name)
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
