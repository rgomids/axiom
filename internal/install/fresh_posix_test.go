//go:build !windows

package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/local"
)

func freshPOSIXFixture(t *testing.T) (Target, Candidate) {
	t.Helper()
	prior := hostRow
	hostRow = func() string { return testRow }
	t.Cleanup(func() { hostRow = prior })
	temp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	base := privateDirectory(t, temp, "fresh")
	archive, checksums := newBundle("1.0.0", []byte("binary\n")).write(t, base)
	candidate, err := LoadCandidate(archive, checksums)
	if err != nil {
		t.Fatal(err)
	}
	return Target{BinaryDir: filepath.Join(base, "bin"), ReceiptDir: filepath.Join(base, "receipts"), State: compatibility.Roots{State: filepath.Join(base, "state"), Projects: filepath.Join(base, "projects")}}, candidate
}

func TestPOSIXFreshInstallAndUnchangedBypassState(t *testing.T) {
	target, candidate := freshPOSIXFixture(t)
	result := InstallReleasePOSIX(context.Background(), target, candidate, "")
	if result.ExitCode != 0 || result.Stdout != "install_status=installed\n" || result.Stderr != "" {
		t.Fatalf("result=%+v", result)
	}
	receiptBefore := read(t, filepath.Join(target.ReceiptDir, receiptName))
	if _, err := os.Lstat(filepath.Join(target.ReceiptDir, markerName)); !os.IsNotExist(err) {
		t.Fatalf("marker: %v", err)
	}
	writeFile(t, target.State.State, []byte("unsafe state\n"), 0o600)
	result = InstallReleasePOSIX(context.Background(), target, candidate, "")
	if result.ExitCode != 0 || result.Stdout != "install_status=unchanged\n" {
		t.Fatalf("result=%+v", result)
	}
	if got := read(t, filepath.Join(target.ReceiptDir, receiptName)); got != receiptBefore {
		t.Fatal("unchanged receipt mutated")
	}
	for path, mode := range map[string]os.FileMode{filepath.Join(target.BinaryDir, binaryName): 0o700, filepath.Join(target.ReceiptDir, receiptName): 0o600} {
		info, err := os.Lstat(path)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("mode=%v err=%v", info, err)
		}
	}
}

func TestPOSIXFreshInterruptionProtocol(t *testing.T) {
	for _, stage := range []string{"before_binary", "after_binary"} {
		t.Run(stage, func(t *testing.T) {
			target, candidate := freshPOSIXFixture(t)
			result := InstallReleasePOSIX(context.Background(), target, candidate, stage)
			if result.ExitCode != 75 {
				t.Fatalf("result=%+v", result)
			}
			wantStage := "prepare"
			wantOutput := "install_error: injected pre-commit interruption\n"
			if stage == "after_binary" {
				wantStage = "binary_committed"
				wantOutput = "install_status=partial\n"
			}
			if result.Stderr != wantOutput || result.Stdout != "" {
				t.Fatalf("result=%+v", result)
			}
			if stage == "before_binary" {
				if posixExists(filepath.Join(target.ReceiptDir, markerName)) {
					t.Fatal("precommit marker retained")
				}
				result = InstallReleasePOSIX(context.Background(), target, candidate, "")
				if result.ExitCode != 0 || result.Stdout != "install_status=installed\n" {
					t.Fatalf("retry=%+v", result)
				}
				return
			}
			marker := read(t, filepath.Join(target.ReceiptDir, markerName))
			want := "formatVersion=1\nstage=" + wantStage + "\narchiveSha256=" + candidate.ArchiveSHA256 + "\n"
			if marker != want {
				t.Fatalf("marker=%q want=%q", marker, want)
			}
			result = InstallReleasePOSIX(context.Background(), target, candidate, "")
			if result.Stderr != "install_error: recovery_required\n" || result.ExitCode != 1 {
				t.Fatalf("result=%+v", result)
			}
			if got := read(t, filepath.Join(target.ReceiptDir, markerName)); got != marker {
				t.Fatal("marker changed")
			}
			if _, err := os.Lstat(filepath.Join(target.ReceiptDir, receiptName)); !os.IsNotExist(err) {
				t.Fatalf("receipt: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(target.ReceiptDir, lockName)); !os.IsNotExist(err) {
				t.Fatalf("lock: %v", err)
			}
		})
	}
}

func TestPOSIXFreshRefusalsPreserveExistingObjects(t *testing.T) {
	tests := []struct {
		name, want string
		setup      func(*testing.T, Target, Candidate)
	}{
		{"foreign binary", "foreign binary preserved", func(t *testing.T, target Target, _ Candidate) {
			privateDirectory(t, filepath.Dir(target.BinaryDir), "bin")
			writeFile(t, filepath.Join(target.BinaryDir, binaryName), []byte("foreign"), 0o700)
		}},
		{"foreign receipt", "foreign receipt preserved", func(t *testing.T, target Target, _ Candidate) {
			privateDirectory(t, filepath.Dir(target.ReceiptDir), "receipts")
			writeFile(t, filepath.Join(target.ReceiptDir, receiptName), []byte("foreign"), 0o600)
		}},
		{"unsafe root", "unsafe destination ownership, permissions, ACL, or type", func(t *testing.T, target Target, _ Candidate) {
			privateDirectory(t, filepath.Dir(target.ReceiptDir), "receipts")
			if err := os.Chmod(target.ReceiptDir, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink root", "symlink destination refused", func(t *testing.T, target Target, _ Candidate) {
			outside := privateDirectory(t, filepath.Dir(target.BinaryDir), "outside")
			if err := os.Symlink(outside, target.BinaryDir); err != nil {
				t.Fatal(err)
			}
		}},
		{"lock", "concurrent installation refused", func(t *testing.T, target Target, _ Candidate) {
			privateDirectory(t, filepath.Dir(target.ReceiptDir), "receipts")
			privateDirectory(t, target.ReceiptDir, lockName)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target, candidate := freshPOSIXFixture(t)
			test.setup(t, target, candidate)
			result := InstallReleasePOSIX(context.Background(), target, candidate, "")
			if result.ExitCode != 1 || result.Stderr != "install_error: "+test.want+"\n" {
				t.Fatalf("result=%+v", result)
			}
			if test.name == "unsafe root" {
				info, _ := os.Lstat(target.ReceiptDir)
				if info.Mode().Perm() != 0o755 {
					t.Fatal("root repaired")
				}
			}
			if test.name == "foreign binary" {
				if read(t, filepath.Join(target.BinaryDir, binaryName)) != "foreign" {
					t.Fatal("binary changed")
				}
				if posixExists(target.ReceiptDir) {
					t.Fatal("receipt root created before refusal")
				}
			}
		})
	}
}

func TestPOSIXOwnedUpgradeUsesProtectedService(t *testing.T) {
	current := posixInstallation(t, newBundle("1.0.0", []byte("old binary\n")))
	candidate := current.candidate(t, newBundle("1.1.0", []byte("new binary\n")))
	result := InstallReleasePOSIX(context.Background(), current.target, candidate, "")
	if result.ExitCode != 0 || result.UpgradePhase != "apply" || result.UpgradeResult.Status != "success" || result.UpgradeError != nil {
		t.Fatalf("result=%+v", result)
	}
	if read(t, filepath.Join(current.target.BinaryDir, binaryName)) != "new binary\n" {
		t.Fatal("upgrade missing")
	}
}

func TestPOSIXOwnedModifiedAndDivergentReceiptsRefused(t *testing.T) {
	for _, kind := range []string{"modified", "divergent", "hardlink", "mode"} {
		t.Run(kind, func(t *testing.T) {
			current := posixInstallation(t, newBundle("1.0.0", []byte("old binary\n")))
			candidate := current.candidate(t, current.current)
			destination := filepath.Join(current.target.BinaryDir, binaryName)
			receipt := filepath.Join(current.target.ReceiptDir, receiptName)
			want := "modified binary preserved"
			switch kind {
			case "modified":
				writeFile(t, destination, []byte("changed\n"), 0o700)
			case "divergent":
				want = "divergent receipt preserved"
				wire := read(t, receipt)
				writeFile(t, receipt, []byte(strings.Replace(wire, "revision=123456789abc", "revision=abcdef123456", 1)), 0o600)
			case "hardlink":
				want = "hard-linked installation preserved"
				if err := os.Link(destination, filepath.Join(current.target.BinaryDir, "linked")); err != nil {
					t.Fatal(err)
				}
			case "mode":
				want = "unsafe installation permissions"
				if err := os.Chmod(destination, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			binaryBefore := read(t, destination)
			receiptBefore := read(t, receipt)
			result := InstallReleasePOSIX(context.Background(), current.target, candidate, "")
			if result.Stderr != "install_error: "+want+"\n" || result.ExitCode != 1 {
				t.Fatalf("result=%+v", result)
			}
			if read(t, destination) != binaryBefore || read(t, receipt) != receiptBefore {
				t.Fatal("refused installation mutated")
			}
		})
	}
}

func posixInstallation(t *testing.T, b *bundle) installation {
	t.Helper()
	current := install(t, b)
	original := current.target.BinaryDir
	var err error
	current.target.BinaryDir, err = filepath.EvalSymlinks(current.target.BinaryDir)
	if err != nil {
		t.Fatal(err)
	}
	current.target.ReceiptDir, err = filepath.EvalSymlinks(current.target.ReceiptDir)
	if err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(current.target.ReceiptDir, receiptName)
	wire := read(t, receipt)
	writeFile(t, receipt, []byte(strings.Replace(wire, "destination="+original+"/axiom", "destination="+current.target.BinaryDir+"/axiom", 1)), 0o600)
	return current
}

func TestPOSIXLexicalAndSharedDestinations(t *testing.T) {
	for _, sameRoot := range []bool{false, true} {
		t.Run(map[bool]string{false: "trailing slash", true: "shared root"}[sameRoot], func(t *testing.T) {
			target, candidate := freshPOSIXFixture(t)
			target.BinaryDir += "/"
			if sameRoot {
				target.ReceiptDir = target.BinaryDir
			}
			result := InstallReleasePOSIX(context.Background(), target, candidate, "")
			if result.Stdout != "install_status=installed\n" || result.ExitCode != 0 {
				t.Fatalf("result=%+v", result)
			}
			wire := read(t, filepath.Join(target.ReceiptDir, receiptName))
			if !strings.Contains(wire, "destination="+target.BinaryDir+"/axiom\n") {
				t.Fatal("lexical destination lost")
			}
			result = InstallReleasePOSIX(context.Background(), target, candidate, "")
			if result.Stdout != "install_status=unchanged\n" || result.ExitCode != 0 {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestPOSIXExactUpgradeResume(t *testing.T) {
	current := posixInstallation(t, newBundle("1.0.0", []byte("old\n")))
	candidate := current.candidate(t, newBundle("1.1.0", []byte("new\n")))
	service := NewService()
	preview, err := service.Preview(context.Background(), current.target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := Authorize(preview, preview.Digest)
	if err != nil {
		t.Fatal(err)
	}
	service.afterEffect = func(string) error { return os.ErrInvalid }
	partial, err := service.Apply(context.Background(), preview, authority)
	if err == nil || partial.Status != "partial" {
		t.Fatalf("result=%+v err=%v", partial, err)
	}
	other := current.candidate(t, newBundle("1.2.0", []byte("other\n")))
	refused := InstallReleasePOSIX(context.Background(), current.target, other, "")
	if refused.Stderr != "install_error: recovery_required\n" {
		t.Fatalf("result=%+v", refused)
	}
	result := InstallReleasePOSIX(context.Background(), current.target, candidate, "")
	if result.ExitCode != 0 || result.UpgradeResult.Status != "success" || !result.UpgradePreview.Resume {
		t.Fatalf("result=%+v", result)
	}
}

func TestPOSIXConfigurationPreflightRunsOnlyForUpgrade(t *testing.T) {
	target, candidate := freshPOSIXFixture(t)
	calls := 0
	unavailable := func() error { calls++; return os.ErrInvalid }
	for _, want := range []string{"installed", "unchanged"} {
		result := InstallReleasePOSIX(context.Background(), target, candidate, "", unavailable)
		if result.Stdout != "install_status="+want+"\n" || calls != 0 {
			t.Fatalf("result=%+v calls=%d", result, calls)
		}
	}
	archive, checksums := newBundle("1.1.0", []byte("new\n")).write(t, filepath.Dir(target.BinaryDir))
	next, err := LoadCandidate(archive, checksums)
	if err != nil {
		t.Fatal(err)
	}
	result := InstallReleasePOSIX(context.Background(), target, next, "", unavailable)
	if calls != 1 || result.ExitCode != 1 || result.UpgradePhase != "preview" || result.UpgradeError != os.ErrInvalid {
		t.Fatalf("result=%+v calls=%d", result, calls)
	}
	if read(t, filepath.Join(target.BinaryDir, binaryName)) != string(candidate.Binary) {
		t.Fatal("preflight failure changed binary")
	}
}

func TestPOSIXCleanupPreservesChangedObjects(t *testing.T) {
	for _, change := range []string{"bytes", "file identity", "root identity", "lock identity"} {
		t.Run(change, func(t *testing.T) {
			target, _ := freshPOSIXFixture(t)
			directory, err := local.CreateOwnedDirectory(target.ReceiptDir)
			if err != nil {
				t.Fatal(err)
			}
			defer directory.Close()
			wire := []byte("owned\n")
			if change == "lock identity" {
				if err := directory.Mkdir(lockName); err != nil {
					t.Fatal(err)
				}
				identity, err := directory.Lstat(lockName)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(target.ReceiptDir, lockName), filepath.Join(target.ReceiptDir, "old-lock")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(target.ReceiptDir, lockName), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := posixRemoveLock(directory, identity); err == nil {
					t.Fatal("changed lock removed")
				}
				if !posixExists(filepath.Join(target.ReceiptDir, lockName)) {
					t.Fatal("replacement lost")
				}
				return
			}
			if err := directory.CreateExclusive("stage", wire, 0o600); err != nil {
				t.Fatal(err)
			}
			identity, err := directory.Lstat("stage")
			if err != nil {
				t.Fatal(err)
			}
			checkPath := filepath.Join(target.ReceiptDir, "stage")
			switch change {
			case "bytes":
				writeFile(t, checkPath, []byte("changed\n"), 0o600)
			case "file identity":
				if err := os.Rename(checkPath, checkPath+".old"); err != nil {
					t.Fatal(err)
				}
				writeFile(t, checkPath, wire, 0o600)
			case "root identity":
				if err := os.Rename(target.ReceiptDir, target.ReceiptDir+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(target.ReceiptDir, 0o700); err != nil {
					t.Fatal(err)
				}
				checkPath = filepath.Join(target.ReceiptDir+".old", "stage")
			}
			if err := posixRemoveFile(directory, "stage", identity, wire, maxReceiptBytes); err == nil {
				t.Fatal("changed object removed")
			}
			if !posixExists(checkPath) {
				t.Fatal("preserved object lost")
			}
		})
	}
}
