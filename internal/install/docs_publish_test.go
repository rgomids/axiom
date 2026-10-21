//go:build unix

package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/compatibility"
	"github.com/rgomids/axiom/internal/operationaldocs"
)

func v2Target(t *testing.T) Target {
	t.Helper()
	temp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	base := privateDirectory(t, temp, "v2")
	return Target{BinaryDir: filepath.Join(base, "bin"), ReceiptDir: filepath.Join(base, "receipts"), State: compatibility.Roots{State: filepath.Join(base, "state"), Projects: filepath.Join(base, "projects")}}
}

func v2Candidate(t *testing.T, binary, tag, version string) Candidate {
	t.Helper()
	a := newV2ArchiveFor(t, binary, tag)
	if version != "" {
		platform, _, arch := hostTuple()
		a.root = "axiom-" + version + "-" + platform + "-" + arch
		a.files["release-metadata.txt"] = []byte(strings.Replace(string(a.files["release-metadata.txt"]), "version=0.2.0", "version="+version, 1))
		a.seal()
	}
	archive, sums := a.write(t)
	candidate, err := LoadCandidate(archive, sums)
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func descriptorFor(c Candidate) operationaldocs.Descriptor {
	platform, _, arch := hostTuple()
	d := operationaldocs.Descriptor{Generation: c.Docs.Generation, SourceCommit: c.DocsMetadata.SourceCommit, Platform: platform, Architecture: arch, SkillsSHA256: strings.Repeat("c", 64)}
	d.Identity = operationaldocs.Association(d.Generation, d.SourceCommit, d.Platform, d.Architecture, d.SkillsSHA256)
	return d
}

func installV2(t *testing.T, target Target, candidate Candidate) InstallerResult {
	t.Helper()
	t.Cleanup(func() { removeTree(filepath.Dir(target.BinaryDir)) })
	return InstallReleasePOSIX(context.Background(), target, candidate, "")
}

func brief(r InstallerResult) string {
	return fmt.Sprintf("exit=%d stdout=%q stderr=%q phase=%s err=%v", r.ExitCode, r.Stdout, r.Stderr, r.UpgradePhase, r.UpgradeError)
}

func TestV2FreshInstallResolvesAndReinstallIsNoOp(t *testing.T) {
	target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
	if result := installV2(t, target, candidate); result.ExitCode != 0 || result.Stdout != "install_status=installed\n" {
		t.Fatal(brief(result))
	}
	destination := filepath.Join(target.BinaryDir, binaryName)
	resolved := operationaldocs.Resolve(destination, descriptorFor(candidate))
	if resolved.State != operationaldocs.StateVerified || resolved.Bundle.Generation != candidate.Docs.Generation {
		t.Fatalf("%+v", resolved)
	}
	receipt := read(t, filepath.Join(target.ReceiptDir, receiptName))
	if !strings.HasPrefix(receipt, "formatVersion=2\n") || !strings.Contains(receipt, "docsBindingSha256="+resolved.BindingSHA256) {
		t.Fatalf("receipt=%q", receipt)
	}
	before := treeSnapshot(t, target.BinaryDir)
	if result := installV2(t, target, candidate); result.ExitCode != 0 || result.Stdout != "install_status=unchanged\n" {
		t.Fatal(brief(result))
	}
	if after := treeSnapshot(t, target.BinaryDir); before != after {
		t.Fatal("no-op reinstall changed the namespace")
	}
}

func treeSnapshot(t *testing.T, root string) string {
	t.Helper()
	var out strings.Builder
	if err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		out.WriteString(p + " " + info.Mode().String() + " " + info.ModTime().String() + "\n")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestV2ReinstallRepairsMissingGeneration(t *testing.T) {
	target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
	installV2(t, target, candidate)
	generation := filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsGenerationsName, candidate.Docs.Generation)
	removeTree(generation)
	destination := filepath.Join(target.BinaryDir, binaryName)
	if got := operationaldocs.Resolve(destination, descriptorFor(candidate)); got.State == operationaldocs.StateVerified {
		t.Fatal("missing generation resolved")
	}
	if result := installV2(t, target, candidate); result.ExitCode != 0 {
		t.Fatal(brief(result))
	}
	if got := operationaldocs.Resolve(destination, descriptorFor(candidate)); got.State != operationaldocs.StateVerified {
		t.Fatalf("%+v", got)
	}
}

func TestV2UpgradeKeepsSurvivorOnlyForOwnDescriptor(t *testing.T) {
	target := v2Target(t)
	a := v2Candidate(t, "binary-A", "A", "0.2.0")
	installV2(t, target, a)
	b := v2Candidate(t, "binary-B", "B", "0.3.0")
	result := installV2(t, target, b)
	if result.ExitCode != 0 || result.UpgradeResult.Status != "success" {
		t.Fatal(brief(result))
	}
	destination := filepath.Join(target.BinaryDir, binaryName)
	if got := operationaldocs.Resolve(destination, descriptorFor(b)); got.State != operationaldocs.StateVerified {
		t.Fatalf("B: %+v", got)
	}
	if got := operationaldocs.Resolve(destination, descriptorFor(a)); got.State == operationaldocs.StateVerified {
		t.Fatal("A descriptor verified against B's binary")
	}
	if !strings.Contains(read(t, filepath.Join(target.ReceiptDir, receiptName)), "version=0.3.0") {
		t.Fatal("receipt not upgraded")
	}
	if down := InstallReleasePOSIX(context.Background(), target, a, ""); down.ExitCode == 0 {
		t.Fatal("downgrade accepted: " + brief(down))
	}
}

func TestV2FaultBoundariesNeverLeaveHalfCommit(t *testing.T) {
	for _, stage := range []string{"generation_staged", "generation_committed", "binding_published", "before_binary_commit"} {
		t.Run(stage, func(t *testing.T) {
			target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
			docsFault = func(s string) error {
				if s == stage {
					return errors.New("injected")
				}
				return nil
			}
			defer func() { docsFault = func(string) error { return nil } }()
			result := installV2(t, target, candidate)
			if result.ExitCode == 0 {
				t.Fatal(brief(result))
			}
			if posixExists(filepath.Join(target.BinaryDir, binaryName)) || posixExists(filepath.Join(target.ReceiptDir, receiptName)) || posixExists(filepath.Join(target.ReceiptDir, markerName)) {
				t.Fatal("pre-commit failure left a binary, receipt or marker")
			}
			docsFault = func(string) error { return nil }
			if retry := installV2(t, target, candidate); retry.ExitCode != 0 {
				t.Fatal(brief(retry))
			}
			if got := operationaldocs.Resolve(filepath.Join(target.BinaryDir, binaryName), descriptorFor(candidate)); got.State != operationaldocs.StateVerified {
				t.Fatalf("%+v", got)
			}
		})
	}
}

func TestV2ConflictingGenerationAndForeignFilesArePreserved(t *testing.T) {
	target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
	namespace := filepath.Join(target.BinaryDir, docsParentName, docsRootName)
	if err := os.MkdirAll(filepath.Join(namespace, docsGenerationsName, candidate.Docs.Generation, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(namespace, docsBindingsName), 0o700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(namespace, "foreign.txt")
	if err := os.WriteFile(foreign, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for dir := filepath.Dir(namespace); dir != filepath.Dir(target.BinaryDir); dir = filepath.Dir(dir) {
		_ = os.Chmod(dir, 0o700)
	}
	result := installV2(t, target, candidate)
	if result.ExitCode == 0 || posixExists(filepath.Join(target.BinaryDir, binaryName)) {
		t.Fatal(brief(result))
	}
	if read(t, foreign) != "mine\n" {
		t.Fatal("foreign file changed")
	}
}

func TestV2MigratesAV1InstallationPreservingForeignFiles(t *testing.T) {
	prior := hostRow
	hostRow = func() string { return testRow }
	tupleOverride = []string{"macos", "darwin", "arm64"}
	t.Cleanup(func() { hostRow, tupleOverride = prior, nil })
	target := v2Target(t)
	legacy := newBundle("1.0.0", []byte("binary-v1\n"))
	archive, checksums := legacy.write(t, privateDirectory(t, t.TempDir(), "release"))
	v1, err := LoadCandidate(archive, checksums)
	if err != nil {
		t.Fatal(err)
	}
	if result := installV2(t, target, v1); result.ExitCode != 0 || v1.Docs != nil {
		t.Fatal(brief(result))
	}
	foreign := filepath.Join(target.BinaryDir, "user-tool")
	if err := os.WriteFile(foreign, []byte("mine\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	next := v2Candidate(t, "binary-v2", "N", "1.1.0")
	if result := installV2(t, target, next); result.ExitCode != 0 || result.UpgradeResult.Status != "success" {
		t.Fatal(brief(result))
	}
	receipt := read(t, filepath.Join(target.ReceiptDir, receiptName))
	if !strings.HasPrefix(receipt, "formatVersion=2\n") || !strings.Contains(receipt, "version=1.1.0") {
		t.Fatalf("receipt=%q", receipt)
	}
	if read(t, foreign) != "mine\n" {
		t.Fatal("foreign file changed")
	}
	if again := installV2(t, target, next); again.Stdout != "install_status=unchanged\n" {
		t.Fatal(brief(again))
	}
}

func TestV2InstallationLockCoversDifferentReceiptRoots(t *testing.T) {
	target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
	other := target
	other.ReceiptDir = filepath.Join(filepath.Dir(target.ReceiptDir), "receipts-other")
	t.Cleanup(func() { removeTree(filepath.Dir(target.BinaryDir)) })
	results := make(chan InstallerResult, 2)
	for _, t2 := range []Target{target, other} {
		go func(tt Target) { results <- InstallReleasePOSIX(context.Background(), tt, candidate, "") }(t2)
	}
	first, second := <-results, <-results
	if first.ExitCode == 0 && second.ExitCode == 0 {
		t.Fatalf("both publishers succeeded: %s | %s", brief(first), brief(second))
	}
	if first.ExitCode != 0 && second.ExitCode != 0 {
		if posixExists(filepath.Join(target.BinaryDir, binaryName)) && posixExists(filepath.Join(target.ReceiptDir, receiptName)) && posixExists(filepath.Join(other.ReceiptDir, receiptName)) {
			t.Fatal("both receipts exist")
		}
	}
	owners := 0
	for _, dir := range []string{target.ReceiptDir, other.ReceiptDir} {
		if posixExists(filepath.Join(dir, receiptName)) {
			owners++
		}
	}
	if owners > 1 {
		t.Fatalf("%d receipt roots claim one binary", owners)
	}
}

func TestV2StaleDocumentationLockRefusesWithoutEffects(t *testing.T) {
	target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
	if result := installV2(t, target, candidate); result.ExitCode != 0 {
		t.Fatal(brief(result))
	}
	lock := filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsLockName)
	if err := os.Mkdir(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	removeTree(filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsGenerationsName, candidate.Docs.Generation))
	before := read(t, filepath.Join(target.ReceiptDir, receiptName))
	result := installV2(t, target, candidate)
	if result.ExitCode == 0 || read(t, filepath.Join(target.ReceiptDir, receiptName)) != before {
		t.Fatal(brief(result))
	}
	if posixExists(filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsGenerationsName, candidate.Docs.Generation)) {
		t.Fatal("publisher wrote through a held lock")
	}
}

func removeTree(path string) {
	_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			_ = os.Chmod(p, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(path)
}

// docsBinding builds the immutable binding for the installed binary identity.

func TestV2StageSwapAtFaultBoundaryPreservesForeignObject(t *testing.T) {
	for _, swap := range []string{"stage_replaced_by_directory", "stage_replaced_by_symlink", "ancestor_replaced"} {
		t.Run(swap, func(t *testing.T) {
			target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
			generations := filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsGenerationsName)
			foreign := filepath.Join(filepath.Dir(target.BinaryDir), "foreign")
			if err := os.MkdirAll(filepath.Join(foreign, "keep"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(foreign, "keep", "file"), []byte("mine\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			moved := filepath.Join(filepath.Dir(target.BinaryDir), "moved")
			docsFault = func(s string) error {
				if s != "generation_staged" {
					return nil
				}
				entries, err := os.ReadDir(generations)
				if err != nil || len(entries) != 1 {
					t.Errorf("stage not found: %v %v", entries, err)
					return nil
				}
				stage := filepath.Join(generations, entries[0].Name())
				switch swap {
				case "ancestor_replaced":
					_ = os.Chmod(stage, 0o700)
					if err := os.Rename(generations, moved); err != nil {
						t.Error(err)
					}
					if err := os.Symlink(foreign, generations); err != nil {
						t.Error(err)
					}
				default:
					_ = os.Chmod(stage, 0o700)
					if err := os.Rename(stage, moved); err != nil {
						t.Error(err)
					}
					if swap == "stage_replaced_by_symlink" {
						if err := os.Symlink(foreign, stage); err != nil {
							t.Error(err)
						}
					} else if err := os.Rename(foreign, stage); err != nil {
						t.Error(err)
					}
				}
				return nil
			}
			defer func() { docsFault = func(string) error { return nil } }()
			result := installV2(t, target, candidate)
			if result.ExitCode == 0 || posixExists(filepath.Join(target.BinaryDir, binaryName)) {
				t.Fatal("swap accepted: " + brief(result))
			}
			content := filepath.Join(foreign, "keep", "file")
			if swap == "stage_replaced_by_directory" {
				content = filepath.Join(generations, "keep", "file")
				entries, _ := os.ReadDir(generations)
				for _, e := range entries {
					if strings.HasPrefix(e.Name(), docsStagePrefix) {
						content = filepath.Join(generations, e.Name(), "keep", "file")
					}
				}
			}
			if read(t, content) != "mine\n" {
				t.Fatal("foreign object changed or removed")
			}
			_ = os.Remove(generations)
			if swap == "ancestor_replaced" {
				_ = os.Rename(moved, generations)
			}
		})
	}
}

func TestV2FailedPublicationRemovesOnlyItsOwnStage(t *testing.T) {
	target, candidate := v2Target(t), v2Candidate(t, "binary-A", "A", "")
	docsFault = func(s string) error {
		if s == "generation_staged" {
			return errors.New("injected")
		}
		return nil
	}
	defer func() { docsFault = func(string) error { return nil } }()
	if result := installV2(t, target, candidate); result.ExitCode == 0 {
		t.Fatal(brief(result))
	}
	entries, err := os.ReadDir(filepath.Join(target.BinaryDir, docsParentName, docsRootName, docsGenerationsName))
	if err != nil || len(entries) != 0 {
		t.Fatalf("stage left behind: %v %v", entries, err)
	}
}
