package codexruntime

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func preRenderedWorkflow(t *testing.T) skillSetRevision {
	t.Helper()
	previous := skillSetRevision{}
	for _, revision := range sharedSkillHistory {
		if revision.skills["axiom-workflow"] == "580ca950da69046632543523d41447544e979f9f2237278ac39377772fc4f5e5" {
			previous = revision
		}
	}
	if previous.skills["axiom-workflow"] != "580ca950da69046632543523d41447544e979f9f2237278ac39377772fc4f5e5" {
		t.Fatal("missing exact AXM-38 baseline")
	}
	return previous
}
func seedPreRenderedWorkflow(t *testing.T, service Service, root string) {
	t.Helper()
	previous := preRenderedWorkflow(t)
	for name, digest := range previous.skills {
		b, err := os.ReadFile("testdata/pre-rendered-workflow/" + name + ".md")
		if err != nil || digestOf(b) != digest {
			t.Fatal("historical bytes drift", name, err)
		}
		if err = os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, name, "SKILL.md"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	receipt, err := service.integration.receiptFor(root, previous)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, receiptName), receipt, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestRenderedWorkflowSharedUpgradeAndInterruption(t *testing.T) {
	for _, constructor := range runtimeServices(t) {
		for _, scenario := range []string{"owned", "modified", "unowned", "interrupted"} {
			root := privateSkillRoot(t)
			service, err := constructor(root)
			if err != nil {
				t.Fatal(err)
			}
			t.Run(service.Runtime()+"/"+scenario, func(t *testing.T) {
				seedPreRenderedWorkflow(t, service, root)
				if scenario == "modified" || scenario == "unowned" {
					wire := []byte("Synthetic operator edit; preserve this data\n")
					if err := os.WriteFile(filepath.Join(root, "axiom-workflow", "SKILL.md"), wire, 0600); err != nil {
						t.Fatal(err)
					}
					if scenario == "unowned" {
						if err := os.Remove(filepath.Join(root, receiptName)); err != nil {
							t.Fatal(err)
						}
					}
					before := skillTree(t, root)
					result := service.Install(context.Background())
					if result.Status != Failed || len(result.Conflicts) == 0 {
						t.Fatal("adopted modified/unowned skill", result)
					}
					if err := os.Remove(filepath.Join(root, installLockName)); err != nil {
						t.Fatal(err)
					}
					if before != skillTree(t, root) {
						t.Fatal("refusal changed operator content")
					}
					return
				}
				if scenario == "interrupted" {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					service.afterSkill = func(string) { cancel() }
					result := service.Install(ctx)
					if result.Status != Partial {
						t.Fatal("interruption claimed success", result)
					}
					if service.Inspect(context.Background()).Status == Ready {
						t.Fatal("mixed inventory claimed ready")
					}
					service.afterSkill = nil
				}
				installOrFail(t, service, Applied)
				current, err := currentRevision()
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range skillNames {
					installed, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
					embedded, readErr := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
					if err != nil || readErr != nil || string(installed) != string(embedded) || digestOf(installed) != current.skills[name] {
						t.Fatal("consumer bytes differ", name, err, readErr)
					}
				}
				before := skillTree(t, root)
				installOrFail(t, service, Unchanged)
				if before != skillTree(t, root) {
					t.Fatal("no-op mutated skill tree")
				}
			})
		}
	}
}

func conversationalPrevious(t *testing.T) skillSetRevision {
	t.Helper()
	for _, r := range sharedSkillHistory {
		digest, err := r.manifestDigest()
		if err == nil && digest == "508f3f1b1a31a403d5deba227cffef981bc67061812394420948290a78c03216" {
			if len(r.skills) != 4 || r.binaryCompatibility != "3" || r.skillSetVersion != "3" {
				t.Fatal("incomplete previous set")
			}
			return r
		}
	}
	t.Fatal("missing AXM-39 replaced full set")
	return skillSetRevision{}
}
func seedConversationalPrevious(t *testing.T, service Service, root string) {
	t.Helper()
	r := conversationalPrevious(t)
	for name, digest := range r.skills {
		wire, err := os.ReadFile("testdata/pre-conversational-presentation/" + name + ".md")
		if err != nil || digestOf(wire) != digest {
			t.Fatal("historical bytes drift", name, err)
		}
		if err = os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, name, "SKILL.md"), wire, 0600); err != nil {
			t.Fatal(err)
		}
	}
	receipt, err := service.integration.receiptFor(root, r)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, receiptName), receipt, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestOperatorConversational012UpgradeConflictInterruptionAndRestoration(t *testing.T) {
	for _, constructor := range runtimeServices(t) {
		for _, scenario := range []string{"owned", "modified", "foreign", "interrupted"} {
			root := privateSkillRoot(t)
			service, err := constructor(root)
			if err != nil {
				t.Fatal(err)
			}
			t.Run(service.Runtime()+"/"+scenario, func(t *testing.T) {
				seedConversationalPrevious(t, service, root)
				stateRoot := t.TempDir()
				// State fixtures are outside the integration boundary, never global roots.
				for _, name := range []string{"Project", "Workflow", "Execution", "Profile"} {
					if err := os.WriteFile(filepath.Join(stateRoot, name), []byte("synthetic immutable revision 17"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(root, "foreign-note.txt"), []byte("preserve foreign file"), 0600); err != nil {
					t.Fatal(err)
				}
				stateBefore := skillTree(t, stateRoot)
				backup := root + "-previous"
				seedConversationalPrevious(t, service, backup)
				// Claude receipts bind the eventual root, not the backup location.
				receipt, err := os.ReadFile(filepath.Join(root, receiptName))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(backup, receiptName), receipt, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(backup, "foreign-note.txt"), []byte("preserve foreign file"), 0600); err != nil {
					t.Fatal(err)
				}
				before := skillTree(t, root)
				if scenario == "modified" || scenario == "foreign" {
					if err := os.WriteFile(filepath.Join(root, "axiom-project", "SKILL.md"), []byte("synthetic operator content"), 0600); err != nil {
						t.Fatal(err)
					}
					if scenario == "foreign" {
						if err := os.Remove(filepath.Join(root, receiptName)); err != nil {
							t.Fatal(err)
						}
					}
					before = skillTree(t, root)
					got := service.Install(context.Background())
					if got.Status != Failed || len(got.Conflicts) == 0 {
						t.Fatal("overwrote operator content", got)
					}
					if err := os.Remove(filepath.Join(root, installLockName)); err != nil {
						t.Fatal(err)
					}
					if before != skillTree(t, root) || stateBefore != skillTree(t, stateRoot) {
						t.Fatal("refusal changed bytes/state")
					}
					return
				}
				if scenario == "interrupted" {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					service.afterSkill = func(string) { cancel() }
					if got := service.Install(ctx); got.Status != Partial {
						t.Fatal("interruption claimed success", got)
					}
					if service.Inspect(context.Background()).Status == Ready {
						t.Fatal("mixed set claimed ready")
					}
					service.afterSkill = nil
				}
				installOrFail(t, service, Applied)
				if got := service.Inspect(context.Background()); got.Status != Ready || got.Receipt != ReceiptCurrent {
					t.Fatal(got)
				}
				for _, name := range skillNames {
					wire, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
					embedded, readErr := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
					if err != nil || readErr != nil || string(wire) != string(embedded) {
						t.Fatal("final bytes differ", name, err, readErr)
					}
				}
				tree := skillTree(t, root)
				installOrFail(t, service, Unchanged)
				if tree != skillTree(t, root) {
					t.Fatal("reinstall changed bytes")
				}
				if wire, err := os.ReadFile(filepath.Join(root, "foreign-note.txt")); err != nil || string(wire) != "preserve foreign file" {
					t.Fatal("foreign file changed")
				}
				// Simulate separately authorized compatible pair restoration via exact
				// directory backup, following the existing output-separation precedent.
				if err := os.Rename(root, root+"-upgraded"); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(backup, root); err != nil {
					t.Fatal(err)
				}
				if before != skillTree(t, root) || stateBefore != skillTree(t, stateRoot) {
					t.Fatal("pair restoration changed historical bytes/state")
				}
			})
		}
	}
}
