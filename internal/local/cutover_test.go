package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Isolated fixtures only: no real installed state is read.
func legacyFixture(t *testing.T) string {
	t.Helper()
	root := privateTestRoot(t)
	for path, content := range map[string]string{
		"executions/v1/proj/a.json":        `{"formatVersion":5}`,
		"graphs/v1/proj/g.json":            `{"formatVersion":4,"kind":"graph"}`,
		"single-stage-runs/v1/proj/s.json": `{"kind":"single_stage_run","formatVersion":2}`,
	} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func disposition(inv CutoverInventory) CutoverDisposition {
	return CutoverDisposition{InventoryDigest: inv.Digest(), BackupVerified: true, AttemptsDisposed: true, Actor: "human:maintainer"}
}

func TestCutoverBackupIsByteVerifiedAndSourceUntouched(t *testing.T) {
	ctx := context.Background()
	root := legacyFixture(t)
	inv, err := InspectCutover(ctx, root)
	if err != nil || inv.Blocked() || len(inv.Files) != 3 {
		t.Fatalf("inventory: %v %+v", err, inv)
	}
	backup := privateTestRoot(t)
	if err := BackupLegacyState(ctx, root, backup, inv); err != nil {
		t.Fatal(err)
	}
	again, _ := InspectCutover(ctx, root)
	if again.Digest() != inv.Digest() {
		t.Fatal("backup or inventory modified legacy state")
	}
	if err := CutoverDecision(inv, disposition(inv)); err != nil {
		t.Fatalf("fully disposed cutover blocked: %v", err)
	}
	tampered := filepath.Join(backup, "graphs", "v1", "proj", "g.json")
	if err := os.WriteFile(tampered, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLegacyBackup(backup, inv); !errors.Is(err, ErrConflict) {
		t.Fatalf("tampered backup verified: %v", err)
	}
	if err := BackupLegacyState(ctx, root, backup, inv); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("non-empty backup target accepted: %v", err)
	}
	inside := filepath.Join(root, "executions", "backup")
	_ = os.Mkdir(inside, 0o700)
	if err := BackupLegacyState(ctx, root, inside, inv); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("backup inside state root accepted: %v", err)
	}
}

func TestCutoverBlocksActiveUnknownAndIncompleteInventory(t *testing.T) {
	ctx := context.Background()
	cases := map[string]func(t *testing.T, root string) func(){
		"held runner lease": func(t *testing.T, root string) func() {
			s, _ := NewRunLeaseStore(root)
			lease, err := s.Acquire(ctx, "proj", "00000000-0000-4000-8000-000000000042", 0)
			if err != nil {
				t.Fatal(err)
			}
			return func() { lease.Close() }
		},
		"interrupted publication": func(t *testing.T, root string) func() {
			_ = os.WriteFile(filepath.Join(root, "graphs", "v1", "proj", ".axiom-recovery-x"), []byte("{}"), 0o600)
			return func() {}
		},
		"unknown entry": func(t *testing.T, root string) func() {
			_ = os.Mkdir(filepath.Join(root, "mystery"), 0o700)
			return func() {}
		},
		"symlink": func(t *testing.T, root string) func() {
			_ = os.Symlink(t.TempDir(), filepath.Join(root, "executions", "v1", "link"))
			return func() {}
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			root := legacyFixture(t)
			cleanup := setup(t, root)
			defer cleanup()
			inv, err := InspectCutover(ctx, root)
			if err != nil {
				t.Fatal(err)
			}
			if !inv.Blocked() {
				t.Fatalf("%s did not block: %+v", name, inv)
			}
			if err := CutoverDecision(inv, disposition(inv)); !errors.Is(err, ErrRecoveryRequired) || !strings.Contains(err.Error(), "BLOCKED") {
				t.Fatalf("decision: %v", err)
			}
		})
	}
}

func TestCutoverDecisionRequiresExactHumanDisposition(t *testing.T) {
	inv, err := InspectCutover(context.Background(), legacyFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*CutoverDisposition){
		"agent actor":         func(d *CutoverDisposition) { d.Actor = "agent:codex" },
		"stale inventory":     func(d *CutoverDisposition) { d.InventoryDigest = strings.Repeat("0", 64) },
		"no backup":           func(d *CutoverDisposition) { d.BackupVerified = false },
		"attempts undisposed": func(d *CutoverDisposition) { d.AttemptsDisposed = false },
	} {
		d := disposition(inv)
		mutate(&d)
		if err := CutoverDecision(inv, d); err == nil {
			t.Errorf("%s: cutover allowed", name)
		}
	}
	d := disposition(inv)
	d.BackupVerified, d.BackupInfeasible = false, "disk full; operator accepted loss of raw history"
	if err := CutoverDecision(inv, d); err != nil {
		t.Fatalf("recorded infeasible backup: %v", err)
	}
}
