package codexruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func seedEightSkillRoot(t *testing.T, service Service, root string) {
	t.Helper()
	revision := sharedSkillHistory[len(sharedSkillHistory)-1]
	if len(revision.skills) != 8 {
		t.Fatal("missing frozen eight-skill revision")
	}
	for name, sha := range revision.skills {
		content, err := os.ReadFile(filepath.Join("testdata", "published-skills", "v0.10.0", name, "SKILL.md"))
		if err != nil || digestOf(content) != sha {
			t.Fatalf("historical %s digest differs: %v", name, err)
		}
		writePrivateSkill(t, root, name, content)
	}
	receipt, err := service.integration.receiptFor(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, receiptName), receipt, 0600); err != nil {
		t.Fatal(err)
	}
}

func assertOnlyCanonical(t *testing.T, service Service, root string) {
	t.Helper()
	for _, name := range retiredSkillNames {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("retired %s still present", name)
		}
	}
	status := service.Inspect(context.Background())
	if status.Status != Ready || len(status.Skills) != 2 {
		t.Fatalf("status=%+v", status)
	}
	installOrFail(t, service, Unchanged)
}

func TestEightSkillRetirementAndInterruptedCleanupBothRuntimes(t *testing.T) {
	for _, factory := range runtimeServices(t) {
		for _, interrupted := range []bool{false, true} {
			root := privateSkillRoot(t)
			service, _ := factory(root)
			t.Run(service.Runtime()+map[bool]string{false: "/eight", true: "/empty-interrupted"}[interrupted], func(t *testing.T) {
				seedEightSkillRoot(t, service, root)
				if interrupted {
					if err := os.Remove(filepath.Join(root, retiredSkillNames[0], "SKILL.md")); err != nil {
						t.Fatal(err)
					}
				}
				installOrFail(t, service, Applied)
				assertOnlyCanonical(t, service, root)
			})
		}
	}
}

func TestRetirementInterruptionBetweenEntriesRerunsDeterministically(t *testing.T) {
	for _, factory := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, _ := factory(root)
		seedEightSkillRoot(t, service, root)
		ctx, cancel := context.WithCancel(context.Background())
		service.afterSkill = func(name string) {
			if name == retiredSkillNames[0] {
				cancel()
			}
		}
		if result := service.Install(ctx); result.Status != Partial {
			t.Fatalf("result=%+v", result)
		}
		service.afterSkill = nil
		installOrFail(t, service, Applied)
		assertOnlyCanonical(t, service, root)
	}
}

func TestRetirementRefusesMalformedReceiptAndForeignLegacyContent(t *testing.T) {
	for _, test := range []string{"receipt", "foreign", "extra", "linked"} {
		for _, factory := range runtimeServices(t) {
			root := privateSkillRoot(t)
			service, _ := factory(root)
			t.Run(service.Runtime()+"/"+test, func(t *testing.T) {
				seedEightSkillRoot(t, service, root)
				path := filepath.Join(root, retiredSkillNames[0], "SKILL.md")
				switch test {
				case "receipt":
					os.WriteFile(filepath.Join(root, receiptName), []byte("foreign malformed receipt\n"), 0600)
				case "foreign":
					os.WriteFile(path, []byte("operator edit\n"), 0600)
				case "extra":
					os.WriteFile(filepath.Join(root, retiredSkillNames[0], "notes"), []byte("operator content\n"), 0600)
				case "linked":
					os.Remove(path)
					os.Symlink(filepath.Join(root, retiredSkillNames[1], "SKILL.md"), path)
				}
				before := skillTree(t, root)
				if result := service.Install(context.Background()); result.Status != Failed {
					t.Fatalf("result=%+v", result)
				}
				os.Remove(filepath.Join(root, installLockName))
				if skillTree(t, root) != before {
					t.Fatal("conflicting content changed")
				}
			})
		}
	}
}

func TestRetirementRechecksEachEntryBeforeDeletion(t *testing.T) {
	root := privateSkillRoot(t)
	service, _ := New(root)
	seedEightSkillRoot(t, service, root)
	altered := filepath.Join(root, retiredSkillNames[1], "SKILL.md")
	service.afterSkill = func(name string) {
		if name == retiredSkillNames[0] {
			if err := os.WriteFile(altered, []byte("changed during cleanup\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	result := service.Install(context.Background())
	if result.Status != Partial {
		t.Fatalf("result=%+v", result)
	}
	content, err := os.ReadFile(altered)
	if err != nil || string(content) != "changed during cleanup\n" {
		t.Fatal("changed entry deleted")
	}
	service.afterSkill = nil
	if result := service.Install(context.Background()); result.Status != Failed {
		t.Fatalf("rerun=%+v", result)
	}
}
