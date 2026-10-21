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
