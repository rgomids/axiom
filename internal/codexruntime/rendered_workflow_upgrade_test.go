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
	previous := sharedSkillHistory[len(sharedSkillHistory)-1]
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
