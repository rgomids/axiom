package codexruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rgomids/axiom/internal/testfs"
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
				var err error
				switch test {
				case "receipt":
					err = os.WriteFile(filepath.Join(root, receiptName), []byte("foreign malformed receipt\n"), 0600)
				case "foreign":
					err = os.WriteFile(path, []byte("operator edit\n"), 0600)
				case "extra":
					err = os.WriteFile(filepath.Join(root, retiredSkillNames[0], "notes"), []byte("operator content\n"), 0600)
				case "linked":
					if err = os.Remove(path); err == nil {
						err = testfs.Symlink(t, filepath.Join(root, retiredSkillNames[1], "SKILL.md"), path)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if test == "linked" {
					if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
						t.Fatalf("link fixture not created: %v %v", info, err)
					}
				}
				before := skillTree(t, root)
				if result := service.Install(context.Background()); result.Status != Failed {
					t.Fatalf("result=%+v", result)
				}
				if err := os.Remove(filepath.Join(root, installLockName)); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
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

var errInjectedRetirementInterruption = errors.New("injected retirement interruption")

// interruptRetirement runs one removal through the anchored root and stops it
// at the first verification for which gone(root) holds, simulating a process
// interruption at that internal boundary.
func interruptRetirement(t *testing.T, service Service, root, name string, gone func() bool) {
	t.Helper()
	anchored, identity, err := anchoredRoot(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	verify := func() error {
		if gone() {
			return errInjectedRetirementInterruption
		}
		return identity.verify(anchored)
	}
	if err := service.integration.removeRetiredIn(anchored, root, name, "", verify); err == nil {
		t.Fatal("injected interruption did not stop the removal")
	}
}

func absent(path string) func() bool {
	return func() bool {
		_, err := os.Lstat(path)
		return os.IsNotExist(err)
	}
}

func inventoryState(t *testing.T, service Service, name string) (SkillSetState, string) {
	t.Helper()
	inventory, err := service.OwnershipInventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range inventory.Skills {
		if skill.Name == name {
			return inventory.State, skill.State
		}
	}
	t.Fatalf("%s not inventoried", name)
	return "", ""
}

// An interruption between the two removals of one entry, or after the
// directory but before its proof, converges on rerun with or without the
// historical receipt, because the proof precedes the first deletion.
func TestRetirementInterruptedWithinEntryResumesWithDurableProof(t *testing.T) {
	for _, receipt := range []bool{true, false} {
		for _, stop := range []string{"content", "directory"} {
			for _, factory := range runtimeServices(t) {
				root := privateSkillRoot(t)
				service, _ := factory(root)
				name := retiredSkillNames[0]
				t.Run(service.Runtime()+map[bool]string{true: "/receipt/", false: "/no-receipt/"}[receipt]+stop, func(t *testing.T) {
					seedEightSkillRoot(t, service, root)
					historical, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
					if err != nil {
						t.Fatal(err)
					}
					if !receipt {
						if err := os.Remove(filepath.Join(root, receiptName)); err != nil {
							t.Fatal(err)
						}
					}
					gone := absent(filepath.Join(root, name, "SKILL.md"))
					if stop == "directory" {
						gone = absent(filepath.Join(root, name))
					}
					interruptRetirement(t, service, root, name, gone)
					proof, err := os.ReadFile(filepath.Join(root, retirementProofName(name)))
					if err != nil || string(proof) != string(historical) {
						t.Fatalf("durable proof missing: %v", err)
					}
					if stop == "content" {
						entries, err := os.ReadDir(filepath.Join(root, name))
						if err != nil || len(entries) != 0 {
							t.Fatalf("expected empty interrupted entry: %v %v", entries, err)
						}
						if set, state := inventoryState(t, service, name); set != SkillSetUpgradable || state != "retiring" {
							t.Fatalf("inventory=%s/%s", set, state)
						}
					}
					if status := service.Inspect(context.Background()); status.Status == Ready {
						t.Fatalf("interrupted retirement reported ready: %+v", status)
					}
					installOrFail(t, service, Applied)
					if _, err := os.Lstat(filepath.Join(root, retirementProofName(name))); !os.IsNotExist(err) {
						t.Fatal("retirement proof left after convergence")
					}
					assertOnlyCanonical(t, service, root)
				})
			}
		}
	}
}

// Without a receipt or proof an empty directory is not Axiom's, and a proof
// is honored only for its own name and only with known bytes.
func TestRetirementRefusesEmptyDirectoryWithoutSpecificProof(t *testing.T) {
	for _, test := range []string{"none", "foreign-proof", "other-name-proof", "extra-file"} {
		for _, factory := range runtimeServices(t) {
			root := privateSkillRoot(t)
			service, _ := factory(root)
			name := retiredSkillNames[0]
			t.Run(service.Runtime()+"/"+test, func(t *testing.T) {
				seedEightSkillRoot(t, service, root)
				if err := os.Remove(filepath.Join(root, receiptName)); err != nil {
					t.Fatal(err)
				}
				var err error
				switch test {
				case "none", "foreign-proof", "other-name-proof":
					err = os.Remove(filepath.Join(root, name, "SKILL.md"))
				case "extra-file":
					interruptRetirement(t, service, root, name, absent(filepath.Join(root, name, "SKILL.md")))
					err = os.WriteFile(filepath.Join(root, name, "notes"), []byte("operator content\n"), 0600)
				}
				if err != nil {
					t.Fatal(err)
				}
				switch test {
				case "foreign-proof":
					err = os.WriteFile(filepath.Join(root, retirementProofName(name)), []byte("operator content\n"), 0600)
				case "other-name-proof":
					var other []byte
					if other, err = os.ReadFile(filepath.Join(root, retiredSkillNames[1], "SKILL.md")); err == nil {
						err = os.WriteFile(filepath.Join(root, retirementProofName(name)), other, 0600)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if set, state := inventoryState(t, service, name); set != SkillSetForeign || state != "foreign" {
					t.Fatalf("inventory=%s/%s", set, state)
				}
				before := skillTree(t, root)
				if result := service.Install(context.Background()); result.Status != Failed {
					t.Fatalf("result=%+v", result)
				}
				if err := os.Remove(filepath.Join(root, installLockName)); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				if skillTree(t, root) != before {
					t.Fatal("unproven content changed")
				}
				for _, path := range []string{filepath.Join(root, name), filepath.Join(root, retirementProofName(name))} {
					if _, err := os.Lstat(path); err != nil && test != "none" {
						t.Fatalf("%s removed: %v", path, err)
					}
				}
			})
		}
	}
}

// Content replaced after the proof was recorded is never deleted, and the
// stale proof cannot authorize removing it on rerun.
func TestRetirementProofDoesNotWidenAuthorityAfterReplacement(t *testing.T) {
	for _, factory := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, _ := factory(root)
		name := retiredSkillNames[0]
		t.Run(service.Runtime(), func(t *testing.T) {
			seedEightSkillRoot(t, service, root)
			if err := os.Remove(filepath.Join(root, receiptName)); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, name, "SKILL.md")
			replaced := false
			interruptRetirement(t, service, root, name, func() bool {
				if _, err := os.Lstat(filepath.Join(root, retirementProofName(name))); err == nil && !replaced {
					replaced = true
					if err := os.WriteFile(path, []byte("operator edit\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return false
			})
			if !replaced {
				t.Fatal("replacement was not injected after the proof")
			}
			if content, err := os.ReadFile(path); err != nil || string(content) != "operator edit\n" {
				t.Fatalf("replaced content deleted: %v", err)
			}
			if result := service.Install(context.Background()); result.Status != Failed {
				t.Fatalf("rerun=%+v", result)
			}
			if content, err := os.ReadFile(path); err != nil || string(content) != "operator edit\n" {
				t.Fatalf("replaced content deleted on rerun: %v", err)
			}
		})
	}
}

// The upgrade session sees the interrupted entry as owned with its proof as
// a leftover, and its retirement converges and is idempotent.
func TestUpgradeSessionResumesInterruptedRetirementWithoutReceipt(t *testing.T) {
	for _, factory := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, _ := factory(root)
		name := retiredSkillNames[0]
		t.Run(service.Runtime(), func(t *testing.T) {
			seedEightSkillRoot(t, service, root)
			if err := os.Remove(filepath.Join(root, receiptName)); err != nil {
				t.Fatal(err)
			}
			interruptRetirement(t, service, root, name, absent(filepath.Join(root, name, "SKILL.md")))
			inventory, err := service.InspectUpgrade(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(inventory.Leftovers) != 1 || inventory.Leftovers[0] != filepath.Join(root, retirementProofName(name)) {
				t.Fatalf("leftovers=%v", inventory.Leftovers)
			}
			session, err := service.LockForUpgrade()
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			anchored, err := session.Inspect(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, skills := range [][]UpgradeSkill{inventory.Skills, anchored.Skills} {
				for _, skill := range skills {
					if skill.Name == name && (!skill.Directory || !skill.Owned || skill.SHA256 != "") {
						t.Fatalf("interrupted entry=%+v", skill)
					}
				}
			}
			for range 2 {
				if err := session.RemoveRetiredSkill(name, ""); err != nil {
					t.Fatal(err)
				}
			}
			for _, path := range []string{filepath.Join(root, name), filepath.Join(root, retirementProofName(name))} {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatalf("%s remains: %v", path, err)
				}
			}
		})
	}
}
