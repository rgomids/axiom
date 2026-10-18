package codexruntime

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func outputSeparationPrevious(t *testing.T) skillSetRevision {
	t.Helper()
	previous := sharedSkillHistory[len(sharedSkillHistory)-1]
	if previous.skillSetVersion != "2" || previous.binaryCompatibility != "2" || len(previous.skills) != 4 {
		t.Fatal("missing complete replaced presentation revision")
	}
	return previous
}

func seedOutputSeparationPrevious(t *testing.T, destination string, receipt []byte) {
	t.Helper()
	previous := outputSeparationPrevious(t)
	for _, name := range skillNames {
		wire, err := os.ReadFile(filepath.Join("testdata", "output-separation-v2", name, "SKILL.md"))
		if err != nil || digestOf(wire) != previous.skills[name] {
			t.Fatalf("replaced %s bytes: %v", name, err)
		}
		if err := os.MkdirAll(filepath.Join(destination, name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, name, "SKILL.md"), wire, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(destination, receiptName), receipt, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestOutputSeparationCoordinatedUpgradeAndPairRestoration(t *testing.T) {
	for _, constructor := range runtimeServices(t) {
		root := filepath.Join(t.TempDir(), "skills")
		service, err := constructor(root)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(service.Runtime(), func(t *testing.T) {
			previous := outputSeparationPrevious(t)
			receipt, err := service.integration.receiptFor(root, previous)
			if err != nil {
				t.Fatal(err)
			}
			seedOutputSeparationPrevious(t, root, receipt)
			backup := root + "-previous"
			seedOutputSeparationPrevious(t, backup, receipt)
			before := skillTree(t, root)
			state := filepath.Join(t.TempDir(), "operational-state")
			if err := os.WriteFile(state, []byte("synthetic revision 17\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if got := service.Inspect(context.Background()); got.Status == Ready || got.Receipt != ReceiptLegacy || len(got.Conflicts) != 0 {
				t.Fatalf("old skills claimed compatible: %+v", got)
			}
			ctx, cancel := context.WithCancel(context.Background())
			service.afterSkill = func(string) { cancel() }
			if got := service.Install(ctx); got.Status != Partial {
				t.Fatalf("interruption: %+v", got)
			}
			if got := service.Inspect(context.Background()); got.Status == Ready {
				t.Fatal("mixed partial pair claimed ready")
			}
			service.afterSkill = nil
			installOrFail(t, service, Applied)
			if got := service.Inspect(context.Background()); got.Status != Ready || got.Receipt != ReceiptCurrent {
				t.Fatal(got)
			}
			for _, name := range skillNames {
				installed, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
				current, readErr := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
				if err != nil || readErr != nil || !bytes.Equal(installed, current) {
					t.Fatalf("new pair %s bytes differ", name)
				}
			}
			tree := skillTree(t, root)
			installOrFail(t, service, Unchanged)
			if tree != skillTree(t, root) {
				t.Fatal("no-op reinstall changed current pair")
			}
			older := service
			older.binaryCompatibility = "2"
			if got := older.Inspect(context.Background()); got.Status != Incompatible {
				t.Fatalf("old consumer accepted new presentation: %+v", got)
			}
			if err := os.Rename(root, root+"-upgraded"); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(backup, root); err != nil {
				t.Fatal(err)
			}
			if before != skillTree(t, root) {
				t.Fatal("pair restoration changed previous bytes or receipt")
			}
			if wire, err := os.ReadFile(state); err != nil || string(wire) != "synthetic revision 17\n" {
				t.Fatal("presentation upgrade/restore changed operational state")
			}
		})
	}
}
