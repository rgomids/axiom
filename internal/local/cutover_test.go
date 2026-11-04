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

func TestCutoverUnsafeRootsAndUnreadableRecordsBlock(t *testing.T) {
	for _, name := range []string{"linked family ancestor", "linked known root", "unreadable record", "unreadable directory"} {
		t.Run(name, func(t *testing.T) {
			root := legacyFixture(t)
			switch name {
			case "linked family ancestor", "linked known root":
				family := "graphs"
				if name == "linked known root" {
					family = "projects"
				}
				if family == "graphs" {
					if err := os.Rename(filepath.Join(root, family), filepath.Join(t.TempDir(), "old")); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(t.TempDir(), filepath.Join(root, family)); err != nil {
					t.Fatal(err)
				}
			case "unreadable record":
				if err := os.Chmod(filepath.Join(root, "graphs/v1/proj/g.json"), 0); err != nil {
					t.Fatal(err)
				}
			case "unreadable directory":
				if err := os.Chmod(filepath.Join(root, "graphs/v1/proj"), 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "graphs/v1/proj"), 0o700) })
			}
			inv, err := InspectCutover(context.Background(), root)
			if err == nil && !inv.Blocked() {
				t.Fatalf("unsafe inventory permitted: %+v", inv)
			}
			if err == nil && CutoverDecision(inv, disposition(inv)) == nil {
				t.Fatal("unsafe cutover allowed")
			}
		})
	}
}

func TestCutoverBackupRefusesForgedPathsAndChangedSource(t *testing.T) {
	root := legacyFixture(t)
	inv, err := InspectCutover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside", "/absolute", "graphs/v1/../../outside", "projects/foreign.json"} {
		forged := CutoverInventory{Files: []CutoverFile{{Path: path}}}
		if err := BackupLegacyState(context.Background(), root, privateTestRoot(t), forged); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("forged path %q: %v", path, err)
		}
		if err := VerifyLegacyBackup(privateTestRoot(t), forged); !errors.Is(err, ErrUnsafe) {
			t.Fatalf("forged verify %q: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, inv.Files[0].Path), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := BackupLegacyState(context.Background(), root, privateTestRoot(t), inv); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed bytes backed up: %v", err)
	}
}

func TestCutoverBackupVerificationRefusesSymlink(t *testing.T) {
	root := legacyFixture(t)
	inv, err := InspectCutover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	backup := privateTestRoot(t)
	if err := BackupLegacyState(context.Background(), root, backup, inv); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(backup, inv.Files[0].Path)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, inv.Files[0].Path), path); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLegacyBackup(backup, inv); !errors.Is(err, ErrConflict) {
		t.Fatalf("linked backup certified: %v", err)
	}
}

func TestCutoverUnknownNestedNamespacesAndUnsafeRootRefuse(t *testing.T) {
	for _, name := range []string{"unknown version", "post-cutover execution", "unsafe state root"} {
		t.Run(name, func(t *testing.T) {
			root := legacyFixture(t)
			switch name {
			case "unknown version":
				if err := os.Mkdir(filepath.Join(root, "graphs", "unknown"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "post-cutover execution":
				if err := os.Mkdir(filepath.Join(root, "executions", "v2"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "unsafe state root":
				if err := os.Chmod(root, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			inv, err := InspectCutover(context.Background(), root)
			if err == nil && !inv.Blocked() {
				t.Fatalf("incomplete inventory allowed: %+v", inv)
			}
		})
	}
}

func TestCutoverActiveUnknownBytesRequireDispositionWithoutDecoding(t *testing.T) {
	for _, status := range []string{"active", "unknown", "incomplete"} {
		t.Run(status, func(t *testing.T) {
			root := legacyFixture(t)
			// Inert arbitrary bytes; inspection must never parse or resume this record.
			record := filepath.Join(root, "graphs", "v1", "proj", "legacy.json")
			if err := os.WriteFile(record, []byte(`{"status":"`+status+`"}`), 0600); err != nil {
				t.Fatal(err)
			}
			inv, err := InspectCutover(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			d := disposition(inv)
			d.AttemptsDisposed = false
			if err := CutoverDecision(inv, d); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatal("undisposed bytes allowed cutover", err)
			}
		})
	}
}

func TestCutoverUnexpectedKnownRootFileAndEmptyHumanBlock(t *testing.T) {
	root := legacyFixture(t)
	if err := os.WriteFile(filepath.Join(root, "projects"), []byte("unexpected"), 0600); err != nil {
		t.Fatal(err)
	}
	inv, err := InspectCutover(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !inv.Blocked() {
		t.Fatal("known root file omitted from blockers")
	}
	d := disposition(inv)
	d.Actor = "human:"
	if err := CutoverDecision(CutoverInventory{}, d); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatal(err)
	}
}

func TestCutoverBackupRequiresCompleteCurrentInventory(t *testing.T) {
	for _, name := range []string{"omitted source", "added source", "extra backup", "cancelled inventory"} {
		t.Run(name, func(t *testing.T) {
			root := legacyFixture(t)
			inv, err := InspectCutover(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			backup := privateTestRoot(t)
			switch name {
			case "omitted source":
				inv.Files = inv.Files[:len(inv.Files)-1]
			case "added source":
				if err := os.WriteFile(filepath.Join(root, "graphs/v1/proj/new.json"), []byte("new"), 0600); err != nil {
					t.Fatal(err)
				}
			case "extra backup":
				if err := BackupLegacyState(context.Background(), root, backup, inv); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(backup, "graphs/v1/proj/new.json"), []byte("new"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := VerifyLegacyBackup(backup, inv); !errors.Is(err, ErrConflict) {
					t.Fatalf("unexpected backup entry certified: %v", err)
				}
				return
			case "cancelled inventory":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if err := BackupLegacyState(ctx, root, backup, inv); err == nil {
					t.Fatal("cancelled inventory certified")
				}
				return
			}
			if err := BackupLegacyState(context.Background(), root, backup, inv); !errors.Is(err, ErrConflict) {
				t.Fatalf("incomplete inventory certified: %v", err)
			}
			entries, err := os.ReadDir(backup)
			if err != nil || len(entries) != 0 {
				t.Fatalf("incomplete inventory wrote backup: %v %v", entries, err)
			}
		})
	}
}

func TestCutoverCancelledEmptyInventoryCannotCertifyBackup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root, backup := privateTestRoot(t), privateTestRoot(t)
	if _, err := InspectCutover(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled inspection: %v", err)
	}
	if err := BackupLegacyState(ctx, root, backup, CutoverInventory{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled empty backup: %v", err)
	}
}

// historicalRecognizedRootsFixture adds the three state roots the inventory
// recognizes but the original cutover catalog omitted (CR-S1-06). Evidence and
// Project context bytes come from the stable-v1 corpus; the probe envelope is
// opaque bytes, because the cutover never decodes legacy state.
func historicalRecognizedRootsFixture(t *testing.T) string {
	t.Helper()
	root := legacyFixture(t)
	corpus := "../compatibility/testdata/stable-v1/snapshots/v0.17.0-issue277.projection/state/"
	files := map[string]string{
		"capability-probes/probe-1/envelope.json":            `{"version":1,"probeId":"probe-1"}`,
		"capability-probes/probe-1/" + probeQuarantineMarker: "digest",
		"capability-probes/probe-1/workspace/out.txt":        "runtime output",
		"capability-probes/probe-1/inbox/request.json":       `{}`,
	}
	for _, path := range []string{historicalEvidencePath, "project-context/v1/default.json"} {
		wire, err := os.ReadFile(filepath.FromSlash(corpus + path))
		if err != nil {
			t.Fatal(err)
		}
		files[path] = string(wire)
	}
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "capability-probes", "probe-1", "sentinel"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

const historicalEvidencePath = "capability-evidence/v1/bf9eb5b0cce3422e44daacb4a34c1e0972cc2ed25c0fa5c34fd1294839d6f5b1.json"

func TestCutoverCatalogCoversEveryInventoriedStateRoot(t *testing.T) {
	for _, name := range StateRootDirectories {
		if !cutoverKnownRoots[name] {
			t.Errorf("state root %q is inventoried but unknown to cutover", name)
		}
	}
}

func TestCutoverHistoricalRecognizedRootsAreInventoriedAndBackedUp(t *testing.T) {
	ctx := context.Background()
	root := historicalRecognizedRootsFixture(t)
	inv, err := InspectCutover(ctx, root)
	if err != nil || inv.Blocked() {
		t.Fatalf("legitimate historical state root blocked: %v %v", err, inv.Blockers)
	}
	paths := map[string]bool{}
	for _, f := range inv.Files {
		paths[f.Path] = true
	}
	for _, want := range []string{
		historicalEvidencePath,
		"capability-probes/probe-1/envelope.json",
		"capability-probes/probe-1/" + probeQuarantineMarker,
		"capability-probes/probe-1/workspace/out.txt",
		"capability-probes/probe-1/inbox/request.json",
	} {
		if !paths[want] {
			t.Errorf("historical bytes omitted from inventory: %s", want)
		}
	}
	if paths["project-context/v1/default.json"] || len(inv.Files) != 3+5 {
		t.Fatalf("live Project context must stay in place, outside the legacy inventory: %+v", inv.Files)
	}
	backup := privateTestRoot(t)
	if err := BackupLegacyState(ctx, root, backup, inv); err != nil {
		t.Fatal(err)
	}
	for _, f := range inv.Files {
		src, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Path)))
		dst, err := os.ReadFile(filepath.Join(backup, filepath.FromSlash(f.Path)))
		if err != nil || string(src) != string(dst) {
			t.Fatalf("backup of %s differs: %v", f.Path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(backup, "project-context")); !os.IsNotExist(err) {
		t.Fatalf("live Project context copied into legacy backup: %v", err)
	}
	if again, _ := InspectCutover(ctx, root); again.Digest() != inv.Digest() {
		t.Fatal("inspection or backup modified historical state")
	}
	if err := CutoverDecision(inv, disposition(inv)); err != nil {
		t.Fatalf("fully disposed historical cutover blocked: %v", err)
	}
}

func TestCutoverRecognizedRootsStillFailClosed(t *testing.T) {
	probe := filepath.Join("capability-probes", "probe-1")
	cases := map[string]func(t *testing.T, root string) error{
		"truly unknown root": func(_ *testing.T, root string) error {
			return os.Mkdir(filepath.Join(root, "capability-evidence-v2"), 0o700)
		},
		"unknown evidence namespace": func(_ *testing.T, root string) error {
			return os.Mkdir(filepath.Join(root, "capability-evidence", "v2"), 0o700)
		},
		"interrupted probe cleanup": func(_ *testing.T, root string) error {
			return os.WriteFile(filepath.Join(root, "capability-probes", ".probe-cleanup-probe-1.json"), []byte("{}"), 0o600)
		},
		"probe staging leftover": func(_ *testing.T, root string) error {
			return os.WriteFile(filepath.Join(root, probe, "workspace", ".axiom-stage-x"), []byte("x"), 0o600)
		},
		"quarantine marker at wrong depth": func(_ *testing.T, root string) error {
			return os.WriteFile(filepath.Join(root, probe, "workspace", probeQuarantineMarker), []byte("x"), 0o600)
		},
		"quarantine marker as directory": func(_ *testing.T, root string) error {
			if err := os.Remove(filepath.Join(root, probe, probeQuarantineMarker)); err != nil {
				return err
			}
			return os.Mkdir(filepath.Join(root, probe, probeQuarantineMarker), 0o700)
		},
		"probe workspace symlink": func(t *testing.T, root string) error {
			return os.Symlink(t.TempDir(), filepath.Join(root, probe, "workspace", "link"))
		},
		"evidence interrupted publication": func(_ *testing.T, root string) error {
			return os.WriteFile(filepath.Join(root, "capability-evidence", "v1", ".axiom-recovery-x"), []byte("{}"), 0o600)
		},
		"Project context replaced by a file": func(_ *testing.T, root string) error {
			if err := os.RemoveAll(filepath.Join(root, "project-context")); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(root, "project-context"), []byte("x"), 0o600)
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			root := historicalRecognizedRootsFixture(t)
			if err := setup(t, root); err != nil {
				t.Fatal(err)
			}
			inv, err := InspectCutover(context.Background(), root)
			if err == nil && !inv.Blocked() {
				t.Fatalf("%s did not block: %+v", name, inv)
			}
			if err == nil && CutoverDecision(inv, disposition(inv)) == nil {
				t.Fatal("unsafe cutover allowed")
			}
		})
	}
}
