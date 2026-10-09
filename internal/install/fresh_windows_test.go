package install

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/codexruntime"
	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/local"
)

func windowsCandidate(t *testing.T, version string) Candidate {
	t.Helper()
	return windowsBundleCandidate(t, newBundle(version, []byte("Windows binary "+version)))
}

func windowsBundleCandidate(t *testing.T, b *bundle) Candidate {
	t.Helper()
	metadata := string(b.contents()["release-metadata.txt"])
	metadata = strings.NewReplacer("platform=macos-27", "platform=windows", "goos=darwin", "goos=windows", "architecture=arm64", "architecture=amd64").Replace(metadata)
	b.files["release-metadata.txt"] = []byte(metadata)
	archive, sums := b.write(t, t.TempDir())
	candidate, err := LoadCandidate(archive, sums)
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func windowsTarget(t *testing.T) Target {
	// Exercise filesystem/install mechanics with a deterministic host fixture.
	// Real workstation/Server eligibility is checked separately.
	previous := hostRow
	hostRow = func() string { return "windows:windows:amd64" }
	t.Cleanup(func() { hostRow = previous })
	base := t.TempDir()
	return Target{BinaryDir: filepath.Join(base, "bin with spaces"), ReceiptDir: filepath.Join(base, "receipt"), State: compatibility.Roots{Projects: filepath.Join(base, "projects"), State: filepath.Join(base, "state")}}
}

func TestWindowsFreshInstallReinstallAndUpgrade(t *testing.T) {
	target := windowsTarget(t)
	first := windowsCandidate(t, "1.0.0")
	if result, err := InstallRelease(context.Background(), target, first); err != nil || result.Status != "installed" {
		t.Fatalf("fresh: %s %v", result.Status, err)
	}
	before, err := os.ReadFile(filepath.Join(target.ReceiptDir, receiptName))
	if err != nil {
		t.Fatal(err)
	}
	if result, err := InstallRelease(context.Background(), target, first); err != nil || result.Status != "unchanged" {
		t.Fatalf("repeat: %s %v", result.Status, err)
	}
	next := windowsCandidate(t, "1.1.0")
	if result, err := InstallRelease(context.Background(), target, next); err != nil || result.Status != "success" || result.Preservation != "" {
		t.Fatalf("upgrade: %s %v", result.Status, err)
	}
	after, err := os.ReadFile(filepath.Join(target.ReceiptDir, receiptName))
	if err != nil {
		t.Fatal(err)
	}
	oldFields, _ := parseReceipt(before)
	newFields, err := parseReceipt(after)
	if err != nil || oldFields["installedAt"] != newFields["installedAt"] || newFields["version"] != "1.1.0" {
		t.Fatalf("receipt: %v %v", newFields, err)
	}
	wire, err := os.ReadFile(filepath.Join(target.BinaryDir, binaryName))
	if err != nil || !bytes.Equal(wire, next.Binary) {
		t.Fatalf("binary: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target.ReceiptDir, markerName)); !os.IsNotExist(err) {
		t.Fatalf("marker remains: %v", err)
	}
}

func TestWindowsInstallReleasePropagatesRecognizedPOCPreservation(t *testing.T) {
	target := windowsTarget(t)
	first := windowsCandidate(t, "1.0.0")
	if _, err := InstallRelease(context.Background(), target, first); err != nil {
		t.Fatal(err)
	}
	installed := installation{target: target}
	installed.withPOCState(t)
	target.Archive = filepath.Join(filepath.Dir(target.ReceiptDir), "archive")
	installed.target = target
	next := windowsCandidate(t, "1.1.0")
	result, err := InstallRelease(context.Background(), target, next)
	if err != nil || result.Status != "success" || result.Preservation != installed.archivePath(t) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := os.Stat(filepath.Join(result.Preservation, "manifest.json")); err != nil {
		t.Fatalf("preservation manifest: %v", err)
	}
}

func TestWindowsFreshInstallPreservesForeignBinary(t *testing.T) {
	target := windowsTarget(t)
	root, err := local.CreateOwnedDirectory(target.BinaryDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.CreateExclusive(binaryName, []byte("foreign"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallRelease(context.Background(), target, windowsCandidate(t, "1.0.0")); category(err) != "foreign_binary_preserved" {
		t.Fatalf("foreign: %v", err)
	}
	wire, _ := root.ReadFile(binaryName, 100)
	if string(wire) != "foreign" {
		t.Fatal("foreign binary changed")
	}
}

func TestWindowsFreshInstallFinalizesExactInterruptedReceipt(t *testing.T) {
	target := windowsTarget(t)
	candidate := windowsCandidate(t, "1.0.0")
	if _, err := InstallRelease(context.Background(), target, candidate); err != nil {
		t.Fatal(err)
	}
	root, err := local.OpenOwnedDirectory(target.ReceiptDir)
	if err != nil {
		t.Fatal(err)
	}
	marker := []byte("formatVersion=1\nstage=prepare\narchiveSha256=" + candidate.ArchiveSHA256 + "\n")
	if err := root.CreateExclusive(markerName, marker, 0o600); err != nil {
		t.Fatal(err)
	}
	root.Close()
	if result, err := InstallRelease(context.Background(), target, candidate); err != nil || result.Status != "installed" {
		t.Fatalf("resume: %s %v", result.Status, err)
	}
}

func TestWindowsUpgradePartialPreservesNonReceiptCauseAndLedger(t *testing.T) {
	target := windowsTarget(t)
	if result, err := InstallRelease(context.Background(), target, windowsCandidate(t, "1.0.0")); err != nil || result.Status != "installed" {
		t.Fatalf("fresh: %s %v", result.Status, err)
	}
	candidate := windowsCandidate(t, "1.1.0")
	service := NewService()
	preview, err := service.Preview(context.Background(), target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := Authorize(preview, preview.Digest)
	if err != nil {
		t.Fatal(err)
	}
	service.afterEffect = func(kind string) error {
		if kind == "binary" {
			// Replace only this fixture's marker with a nonempty directory
			// so final cleanup fails after confirmed binary and receipt effects.
			marker := filepath.Join(target.ReceiptDir, markerName)
			if err := os.Remove(marker); err != nil {
				return err
			}
			if err := os.Mkdir(marker, 0o700); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(marker, "blocker"), []byte("fixture"), 0o600)
		}
		return nil
	}
	result, err := service.Apply(context.Background(), preview, authority)
	if category(err) != "marker_cleanup_failed" || result.Status != "partial" || result.SkillReceipt != "" || len(result.Ledger) != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	for _, entry := range result.Ledger {
		if !entry.Confirmed {
			t.Fatalf("unconfirmed ledger: %+v", result.Ledger)
		}
	}
	wire, readErr := os.ReadFile(filepath.Join(target.BinaryDir, binaryName))
	if readErr != nil || !bytes.Equal(wire, candidate.Binary) {
		t.Fatalf("binary: %v", readErr)
	}
	// InstallRelease must preserve a real service error rather than synthesize
	// a skill-receipt error. This interrupted marker remains recovery evidence.
	_, err = InstallRelease(context.Background(), target, candidate)
	var installErr *Error
	if !errors.As(err, &installErr) || strings.HasPrefix(installErr.Category, "skill_receipt_") {
		t.Fatalf("recovery cause lost: %v", err)
	}
}

func TestWindowsInstallReleaseReceiptPartialCategories(t *testing.T) {
	for _, receipt := range []string{SkillReceiptRefreshRequired, SkillReceiptConflict} {
		t.Run(receipt, func(t *testing.T) {
			target := windowsTarget(t)
			first := windowsCandidate(t, "1.0.0")
			if _, err := InstallRelease(context.Background(), target, first); err != nil {
				t.Fatal(err)
			}
			target.SkillsRoot = filepath.Join(filepath.Dir(target.ReceiptDir), "skills")
			root, err := local.CreateOwnedDirectory(target.SkillsRoot)
			if err != nil {
				t.Fatal(err)
			}
			root.Close()
			for name, wire := range first.SkillFiles {
				directory, err := local.CreateOwnedDirectory(filepath.Join(target.SkillsRoot, name))
				if err != nil {
					t.Fatal(err)
				}
				err = directory.CreateExclusive("SKILL.md", wire, 0o600)
				directory.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			if receipt == SkillReceiptConflict {
				writeFile(t, filepath.Join(target.SkillsRoot, codexruntime.SkillSetReceiptName), []byte("foreign receipt\n"), 0o600)
			}
			next := windowsCandidate(t, "1.1.0")
			if receipt == SkillReceiptConflict {
				next = windowsBundleCandidate(t, selfBundle(t, "1.1.0"))
				target.Self = selfBuildFor("1.1.0")
			}
			result, err := InstallRelease(context.Background(), target, next)
			if result.Status != "partial" || category(err) != "skill_receipt_"+receipt {
				t.Fatalf("status=%s err=%v", result.Status, err)
			}
			wire, readErr := os.ReadFile(filepath.Join(target.BinaryDir, binaryName))
			if readErr != nil || !bytes.Equal(wire, next.Binary) {
				t.Fatalf("binary: %v", readErr)
			}
		})
	}
}
