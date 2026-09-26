package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/compatibility"
)

const testRow = "macos-27:darwin:arm64"

type bundle struct {
	version string
	binary  []byte
	skills  map[string][]byte
	files   map[string][]byte
	links   map[string]string
	skip    map[string]bool
}

func newBundle(version string, binary []byte) *bundle {
	skills := map[string][]byte{}
	for _, name := range skillNames {
		skills[name] = []byte("skill " + name + " " + version + "\n")
	}
	return &bundle{version: version, binary: binary, skills: skills, files: map[string][]byte{}, links: map[string]string{}, skip: map[string]bool{}}
}

func (b *bundle) contents() map[string][]byte {
	files := map[string][]byte{
		"lingo":                b.binary,
		"LICENSE":              []byte("license\n"),
		"release-metadata.txt": []byte(fmt.Sprintf("formatVersion=1\nproduct=Axiom\nversion=%s\nrevision=123456789abc\nsourceState=clean\nrelease=true\nplatform=macos-27\ngoos=darwin\narchitecture=arm64\nskillSetVersion=1\n", b.version)),
	}
	manifest := "formatVersion=1\nskillSetVersion=1\nbinaryCompatibility=1\n"
	for _, name := range skillNames {
		files["skills/"+name+"/SKILL.md"] = b.skills[name]
		manifest += "skill." + name + "=" + digest(b.skills[name]) + "\n"
	}
	files["skills-manifest.txt"] = []byte(manifest)
	for name, wire := range b.files {
		files[name] = wire
	}
	return files
}

// write builds a release-shaped archive and SHA256SUMS in dir.
func (b *bundle) write(t *testing.T, dir string) (string, string) {
	t.Helper()
	files := b.contents()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var manifest strings.Builder
	for _, name := range names {
		if !b.skip[name] {
			fmt.Fprintf(&manifest, "%s  %s\n", digest(files[name]), name)
		}
	}
	files["MANIFEST.sha256"] = []byte(manifest.String())
	names = append(names, "MANIFEST.sha256")
	root := "axiom-" + b.version + "-macos-27-arm64"
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	_ = writer.WriteHeader(&tar.Header{Name: root + "/", Typeflag: tar.TypeDir, Mode: 0o700})
	for _, name := range names {
		wire := files[name]
		if err := writer.WriteHeader(&tar.Header{Name: root + "/" + name, Typeflag: tar.TypeReg, Mode: 0o600, Size: int64(len(wire))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(wire); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range b.links {
		_ = writer.WriteHeader(&tar.Header{Name: root + "/" + name, Typeflag: tar.TypeSymlink, Linkname: target})
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(dir, root+".tar.gz")
	writeFile(t, archivePath, archive.Bytes(), 0o600)
	checksums := filepath.Join(dir, "SHA256SUMS")
	writeFile(t, checksums, []byte(digest(archive.Bytes())+"  "+filepath.Base(archivePath)+"\n"), 0o600)
	return archivePath, checksums
}

type installation struct {
	target  Target
	current *bundle
}

// install mirrors install-release.sh's published owned state for version.
func install(t *testing.T, current *bundle) installation {
	t.Helper()
	t.Cleanup(func() { hostRow = defaultHostRow })
	hostRow = func() string { return testRow }
	base := privateDirectory(t, t.TempDir(), "install")
	target := Target{BinaryDir: privateDirectory(t, base, "bin"), ReceiptDir: privateDirectory(t, base, "receipts"), State: compatibility.Roots{State: filepath.Join(base, "state"), Projects: filepath.Join(base, "projects")}}
	archive, checksums := current.write(t, privateDirectory(t, base, "release-"+current.version))
	candidate, err := LoadCandidate(archive, checksums)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(target.BinaryDir, binaryName), current.binary, 0o700)
	writeFile(t, filepath.Join(target.ReceiptDir, receiptName), expectedReceipt(filepath.Join(target.BinaryDir, binaryName), candidate, "2026-09-20T12:00:00Z"), 0o600)
	return installation{target: target, current: current}
}

func (i installation) candidate(t *testing.T, next *bundle) Candidate {
	t.Helper()
	archive, checksums := next.write(t, t.TempDir())
	candidate, err := LoadCandidate(archive, checksums)
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func (i installation) withSkills(t *testing.T, skills map[string][]byte) {
	t.Helper()
	root := privateDirectory(t, filepath.Dir(i.target.BinaryDir), "skills")
	for name, wire := range skills {
		writeFile(t, filepath.Join(privateDirectory(t, root, name), "SKILL.md"), wire, 0o600)
	}
}

var defaultHostRow = hostRow

func category(err error) string {
	var upgradeErr *Error
	if errors.As(err, &upgradeErr) {
		return upgradeErr.Category
	}
	return ""
}

func TestLoadCandidateMirrorsInstallerVerification(t *testing.T) {
	valid := newBundle("1.1.0", []byte("binary\n"))
	archive, checksums := valid.write(t, t.TempDir())
	candidate, err := LoadCandidate(archive, checksums)
	if err != nil || candidate.Version != "1.1.0" || len(candidate.Skills) != 5 {
		t.Fatalf("candidate=%+v err=%v", candidate, err)
	}
	tests := []struct {
		name   string
		mutate func(*testing.T, *bundle, string, string)
		want   string
	}{
		{"checksum mismatch", func(t *testing.T, _ *bundle, archive, checksums string) {
			writeFile(t, checksums, []byte(strings.Repeat("0", 64)+"  "+filepath.Base(archive)+"\n"), 0o600)
		}, "checksum_mismatch"},
		{"missing checksum", func(t *testing.T, _ *bundle, _, checksums string) {
			writeFile(t, checksums, []byte(strings.Repeat("0", 64)+"  other.tar.gz\n"), 0o600)
		}, "checksum_unavailable"},
		{"duplicate checksum", func(t *testing.T, _ *bundle, archive, checksums string) {
			line := read(t, checksums)
			writeFile(t, checksums, []byte(line+line), 0o600)
		}, "checksum_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := newBundle("1.1.0", []byte("binary\n"))
			archive, checksums := item.write(t, t.TempDir())
			test.mutate(t, item, archive, checksums)
			if _, err := LoadCandidate(archive, checksums); category(err) != test.want {
				t.Fatalf("error=%v want %s", err, test.want)
			}
		})
	}
	for name, mutate := range map[string]func(*bundle){
		"symlink entry": func(b *bundle) { b.links["skills/link"] = "/etc/passwd" },
		"unlisted file": func(b *bundle) { b.files["extra.txt"] = []byte("x\n"); b.skip["extra.txt"] = true },
		"dirty release": func(b *bundle) {
			b.version = "1.1.0"
			b.files["release-metadata.txt"] = []byte("formatVersion=1\nproduct=Axiom\nversion=1.1.0\nrevision=123456789abc\nsourceState=dirty\nrelease=true\nplatform=macos-27\ngoos=darwin\narchitecture=arm64\nskillSetVersion=1\n")
		},
		"injection version": func(b *bundle) { b.version = "1.1.0;$(touch pwned)" },
		"skill manifest skewed": func(b *bundle) {
			b.files["skills-manifest.txt"] = []byte("formatVersion=1\nskillSetVersion=2\nbinaryCompatibility=1\n")
		},
	} {
		t.Run(name, func(t *testing.T) {
			item := newBundle("1.1.0", []byte("binary\n"))
			mutate(item)
			archive, checksums := item.write(t, t.TempDir())
			if _, err := LoadCandidate(archive, checksums); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	if _, err := os.Stat("pwned"); !os.IsNotExist(err) {
		t.Fatal("attack string was executed")
	}
}

func TestUpgradeOrdersConfirmedEffectsAndPreservesInstalledAt(t *testing.T) {
	current := newBundle("1.0.0", []byte("old-binary\n"))
	installed := install(t, current)
	next := newBundle("1.1.0", []byte("new-binary;$(not-executed)\n"))
	installed.withSkills(t, next.skills)
	installed.target.SkillsRoot = filepath.Join(filepath.Dir(installed.target.BinaryDir), "skills")
	candidate := installed.candidate(t, next)
	service := NewService()
	preview, err := service.Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Effects) != 2 || preview.Effects[0].Kind != "binary" || preview.Effects[1].Kind != "receipt" || preview.SourceVersion != "1.0.0" || preview.TargetVersion != "1.1.0" || preview.Skills != SkillsMatch {
		t.Fatalf("preview=%+v", preview)
	}
	if _, err := Authorize(preview, "stale"); err == nil {
		t.Fatal("stale digest authorized upgrade")
	}
	authority, _ := Authorize(preview, preview.Digest)
	result, err := service.Apply(context.Background(), preview, authority)
	if err != nil || result.Status != "success" || len(result.Ledger) != 2 || !result.Ledger[0].Confirmed || !result.Ledger[1].Confirmed {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if read(t, filepath.Join(installed.target.BinaryDir, binaryName)) != string(next.binary) {
		t.Fatal("binary not upgraded")
	}
	receipt := read(t, filepath.Join(installed.target.ReceiptDir, receiptName))
	if !strings.Contains(receipt, "version=1.1.0\n") || !strings.Contains(receipt, "installedAt=2026-09-20T12:00:00Z\n") {
		t.Fatalf("receipt=%s", receipt)
	}
	assertMode(t, filepath.Join(installed.target.BinaryDir, binaryName), 0o700)
	assertMode(t, filepath.Join(installed.target.ReceiptDir, receiptName), 0o600)
	for _, name := range []string{markerName, lockName} {
		if _, err := os.Lstat(filepath.Join(installed.target.ReceiptDir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s left behind", name)
		}
	}
	noOp, err := service.Preview(context.Background(), installed.target, candidate)
	if err != nil || len(noOp.Effects) != 0 {
		t.Fatalf("equivalent upgrade is not a no-op: %+v %v", noOp, err)
	}
	if _, err := Authorize(noOp, noOp.Digest); err == nil {
		t.Fatal("no-op authorized a mutation")
	}
}

func TestUpgradeReportsIncompatibleSkillsAsPartial(t *testing.T) {
	current := newBundle("1.0.0", []byte("old-binary\n"))
	installed := install(t, current)
	installed.withSkills(t, current.skills)
	installed.target.SkillsRoot = filepath.Join(filepath.Dir(installed.target.BinaryDir), "skills")
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	service := NewService()
	preview, err := service.Preview(context.Background(), installed.target, candidate)
	if err != nil || preview.Skills != SkillsRequireInstall {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	authority, _ := Authorize(preview, preview.Digest)
	result, err := service.Apply(context.Background(), preview, authority)
	if err != nil || result.Status != "partial" || result.Skills != SkillsRequireInstall || len(result.Ledger) != 2 {
		t.Fatalf("mixed binary/skill outcome not partial: %+v %v", result, err)
	}
}

func TestUpgradeInterruptionAfterBinaryIsPartialAndResumable(t *testing.T) {
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	next := newBundle("1.1.0", []byte("new-binary\n"))
	candidate := installed.candidate(t, next)
	service := NewService()
	preview, _ := service.Preview(context.Background(), installed.target, candidate)
	authority, _ := Authorize(preview, preview.Digest)
	service.afterEffect = func(kind string) error {
		if kind == "binary" {
			return errors.New("injected interruption")
		}
		return nil
	}
	result, err := service.Apply(context.Background(), preview, authority)
	if err == nil || result.Status != "partial" || len(result.Ledger) != 1 || result.Ledger[0].Kind != "binary" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	marker := read(t, filepath.Join(installed.target.ReceiptDir, markerName))
	if !strings.Contains(marker, "stage=binary_committed\n") || !strings.Contains(marker, "archiveSha256="+candidate.ArchiveSHA256+"\n") {
		t.Fatalf("marker=%s", marker)
	}
	other := installed.candidate(t, newBundle("1.2.0", []byte("other\n")))
	if _, err := NewService().Preview(context.Background(), installed.target, other); category(err) != "recovery_required" {
		t.Fatalf("different archive resumed an interrupted upgrade: %v", err)
	}
	resume, err := NewService().Preview(context.Background(), installed.target, candidate)
	if err != nil || !resume.Resume || len(resume.Effects) != 1 || resume.Effects[0].Kind != "receipt" {
		t.Fatalf("resume=%+v err=%v", resume, err)
	}
	resumeAuthority, _ := Authorize(resume, resume.Digest)
	final, err := NewService().Apply(context.Background(), resume, resumeAuthority)
	if err != nil || final.Status != "success" || len(final.Ledger) != 1 {
		t.Fatalf("final=%+v err=%v", final, err)
	}
	if _, err := os.Lstat(filepath.Join(installed.target.ReceiptDir, markerName)); !os.IsNotExist(err) {
		t.Fatal("marker remained after resumed success")
	}
}

func TestUpgradeRefusalsHaveZeroEffects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *installation) Candidate
		want   string
	}{
		{"downgrade", func(t *testing.T, i *installation) Candidate {
			return i.candidate(t, newBundle("0.9.0", []byte("older\n")))
		}, "downgrade_refused"},
		{"divergent equivalent version", func(t *testing.T, i *installation) Candidate {
			return i.candidate(t, newBundle("1.0.0", []byte("different\n")))
		}, "divergent_equivalent_version"},
		{"modified binary", func(t *testing.T, i *installation) Candidate {
			writeFile(t, filepath.Join(i.target.BinaryDir, binaryName), []byte("tampered\n"), 0o700)
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "binary_modified"},
		{"unsupported host", func(t *testing.T, i *installation) Candidate {
			hostRow = func() string { return "ubuntu-26.04:linux:amd64" }
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "unsupported_host"},
		{"concurrent installation", func(t *testing.T, i *installation) Candidate {
			privateDirectory(t, i.target.ReceiptDir, lockName)
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "installation_busy_or_interrupted"},
		{"installer interrupted", func(t *testing.T, i *installation) Candidate {
			writeFile(t, filepath.Join(i.target.ReceiptDir, markerName), []byte("formatVersion=1\nstage=prepare\narchiveSha256="+strings.Repeat("a", 64)+"\n"), 0o600)
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "recovery_required"},
		{"hard-linked binary", func(t *testing.T, i *installation) Candidate {
			if err := os.Link(filepath.Join(i.target.BinaryDir, binaryName), filepath.Join(t.TempDir(), "alias")); err != nil {
				t.Skip(err)
			}
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "unsafe_binary"},
		{"symlinked receipt", func(t *testing.T, i *installation) Candidate {
			path := filepath.Join(i.target.ReceiptDir, receiptName)
			wire := read(t, path)
			outside := filepath.Join(t.TempDir(), "receipt")
			writeFile(t, outside, []byte(wire), 0o600)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "owned_receipt_required"},
		{"POC state", func(t *testing.T, i *installation) Candidate {
			state := privateDirectory(t, filepath.Dir(i.target.State.State), "state")
			workflows := privateDirectory(t, privateDirectory(t, state, "workflows"), "46f9e9bf-9da2-4769-b9b7-f058f0ab6e90")
			wire, err := os.ReadFile(filepath.Join("..", "compatibility", "testdata", "poc-v0.1.0-poc.1", "state", "workflows", "46f9e9bf-9da2-4769-b9b7-f058f0ab6e90", "main-7.json"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(workflows, "main-7.json"), wire, 0o600)
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "state_incompatible"},
		{"permissive target directory", func(t *testing.T, i *installation) Candidate {
			if err := os.Chmod(i.target.BinaryDir, 0o755); err != nil {
				t.Fatal(err)
			}
			return i.candidate(t, newBundle("1.1.0", []byte("new\n")))
		}, "unsafe_target"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
			candidate := test.mutate(t, &installed)
			before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
			if _, err := NewService().Preview(context.Background(), installed.target, candidate); category(err) != test.want {
				t.Fatalf("error=%v want %s", err, test.want)
			}
			if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
				t.Fatal("refused upgrade changed owned state")
			}
		})
	}
}

func TestUpgradeStaleAuthorityAndSpaceHaveZeroEffects(t *testing.T) {
	installed := install(t, newBundle("1.0.0", []byte("old-binary\n")))
	candidate := installed.candidate(t, newBundle("1.1.0", []byte("new-binary\n")))
	full := Service{availableSpace: func(string) (uint64, error) { return 10, nil }}
	if _, err := full.Preview(context.Background(), installed.target, candidate); category(err) != "insufficient_space" {
		t.Fatalf("space error=%v", err)
	}
	service := NewService()
	preview, err := service.Preview(context.Background(), installed.target, candidate)
	if err != nil {
		t.Fatal(err)
	}
	authority, _ := Authorize(preview, preview.Digest)
	receipt := filepath.Join(installed.target.ReceiptDir, receiptName)
	writeFile(t, receipt, []byte(strings.Replace(read(t, receipt), "installedAt=2026-09-20T12:00:00Z", "installedAt=2026-09-21T12:00:00Z", 1)), 0o600)
	before := snapshot(t, filepath.Dir(installed.target.BinaryDir))
	if result, err := service.Apply(context.Background(), preview, authority); category(err) != "authority_denied" || len(result.Ledger) != 0 {
		t.Fatalf("stale authority result=%+v err=%v", result, err)
	}
	if after := snapshot(t, filepath.Dir(installed.target.BinaryDir)); after != before {
		t.Fatal("stale authority changed state")
	}
}

func TestSemverPrecedence(t *testing.T) {
	for _, test := range []struct {
		left, right string
		want        int
	}{
		{"1.0.0", "1.0.0", 0}, {"1.0.1", "1.0.0", 1}, {"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0-rc.2", "1.0.0-rc.10", -1}, {"1.0.0-alpha", "1.0.0-alpha.1", -1}, {"1.0.0-1", "1.0.0-alpha", -1},
		{"2.0.0", "10.0.0", -1}, {"1.0.0+build", "1.0.0", 0},
	} {
		if got := compareSemver(test.left, test.right); got != test.want {
			t.Fatalf("compare(%s,%s)=%d want %d", test.left, test.right, got, test.want)
		}
	}
}

func privateDirectory(t *testing.T, parent, name string) string {
	t.Helper()
	path := filepath.Join(parent, name)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFile(t *testing.T, path string, wire []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, wire, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(wire)
}

func assertMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != mode {
		t.Fatalf("mode of %s = %v want %v (%v)", path, info.Mode(), mode, err)
	}
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	var builder strings.Builder
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		builder.WriteString(relative + info.Mode().String())
		if info.Mode().IsRegular() {
			wire, _ := os.ReadFile(path)
			builder.Write(wire)
		}
		return nil
	})
	return builder.String()
}
