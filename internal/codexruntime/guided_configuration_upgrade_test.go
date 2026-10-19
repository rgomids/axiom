package codexruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGuidedConfigurationSyntheticPackageRollbackBothHosts(t *testing.T) {
	for _, constructor := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, err := constructor(root)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(service.Runtime(), func(t *testing.T) {
			installOrFail(t, service, Applied)
			files := map[string][]byte{}
			for _, name := range skillNames {
				files[name], err = os.ReadFile(filepath.Join(root, name, "SKILL.md"))
				if err != nil {
					t.Fatal(err)
				}
			}
			receipt, err := os.ReadFile(filepath.Join(root, receiptName))
			if err != nil {
				t.Fatal(err)
			}
			incompatible := service
			incompatible.binaryCompatibility = "synthetic-incompatible"
			if got := incompatible.Inspect(context.Background()); got.Status == Ready {
				t.Fatal("incompatible binary/skill pair reported ready")
			}
			session, err := service.LockForUpgrade()
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			changed := map[string][]byte{}
			for _, name := range skillNames {
				changed[name] = append(append([]byte{}, files[name]...), []byte("\nSynthetic candidate package\n")...)
				if err := session.PublishSkill(name, changed[name], digestOf(files[name])); err != nil {
					t.Fatal(err)
				}
			}
			if got := service.Inspect(context.Background()); got.Status == Ready {
				t.Fatal("mixed package with old receipt reported ready")
			}
			for _, name := range skillNames {
				if err := session.PublishSkill(name, files[name], digestOf(files[name])); err != ErrUpgradeConflict {
					t.Fatalf("stale rollback expected conflict: %v", err)
				}
				wire, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
				if err != nil || string(wire) != string(changed[name]) {
					t.Fatal("refused rollback changed candidate content", err)
				}
				if err := session.PublishSkill(name, files[name], digestOf(changed[name])); err != nil {
					t.Fatal(err)
				}
			}
			if err := session.PublishReceipt(receipt, digestOf(receipt)); err != nil {
				t.Fatal(err)
			}
			session.Close()
			if got := service.Inspect(context.Background()); got.Status != Ready || got.Receipt != ReceiptCurrent {
				t.Fatalf("known compatible package was not restored: %+v", got)
			}
			for _, name := range skillNames {
				wire, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
				if err != nil || string(wire) != string(files[name]) {
					t.Fatal("restored skill differs from known package", name, err)
				}
			}
			installOrFail(t, service, Unchanged)
		})
	}
}
