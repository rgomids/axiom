//go:build unix

package operationaldocs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fixture struct {
	t          *testing.T
	exe        string
	descriptor Descriptor
	root       string
}

func fixtureBundle(tag string) map[string][]byte {
	files := map[string][]byte{}
	var inventory, provenance strings.Builder
	inventory.WriteString("formatVersion=1\npath\tkind\tbytes\tsha256\n")
	provenance.WriteString("formatVersion=1\noutput\tsource\tsourceCommit\tsourceSha256\ttransform\n")
	names := []string{"docs/man/man1/axiom.1"}
	for _, n := range []string{"index", "concepts", "installation", "project", "runtime", "workflow", "work-item", "configuration", "troubleshooting", "cli", "notices"} {
		names = append(names, "docs/text/"+n+".md")
	}
	sortStrings(names)
	for _, name := range names {
		body := []byte("# " + name + " " + tag + "\n")
		if strings.HasSuffix(name, ".1") {
			body = []byte(".TH AXIOM 1\n.SH ROOT\n" + tag + "\n")
		}
		kind := "markdown"
		if strings.HasSuffix(name, ".1") {
			kind = "man1"
		}
		files[name] = body
		fmt.Fprintf(&inventory, "%s\t%s\t%d\t%s\n", name, kind, len(body), Hash(body))
	}
	commit := strings.Repeat("b", 40)
	for _, name := range names {
		fmt.Fprintf(&provenance, "%s\tdocs/operational/index.md\t%s\t%s\toperational-synthesis-v1\n", name, commit, strings.Repeat("e", 64))
	}
	files["docs/inventory.tsv"] = []byte(inventory.String())
	files["docs/provenance.tsv"] = []byte(provenance.String())
	files["docs/man-map.tsv"] = []byte("formatVersion=1\ncommand\tpage\tanchor\naxiom\tdocs/man/man1/axiom.1\tROOT\n")
	return files
}

func sortStrings(s []string) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func platform() (string, string) {
	if runtime.GOOS == "darwin" {
		return "macos", runtime.GOARCH
	}
	return "linux", runtime.GOARCH
}

func newFixture(t *testing.T, binary, tag string) *fixture {
	t.Helper()
	if _, arch := platform(); arch != "arm64" && arch != "amd64" {
		t.Skip("unsupported architecture")
	}
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(base, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, exe: filepath.Join(bin, "axiom")}
	if err := os.WriteFile(f.exe, []byte(binary), 0o700); err != nil {
		t.Fatal(err)
	}
	f.publish(tag)
	return f
}

func (f *fixture) publish(tag string) {
	f.t.Helper()
	files := fixtureBundle(tag)
	commit := strings.Repeat("b", 40)
	bundle, err := Validate(files, commit)
	if err != nil {
		f.t.Fatal(err)
	}
	plat, arch := platform()
	f.descriptor = Descriptor{Generation: bundle.Generation, SourceCommit: commit, Platform: plat, Architecture: arch, SkillsSHA256: strings.Repeat("c", 64)}
	f.descriptor.Identity = Association(f.descriptor.Generation, commit, plat, arch, f.descriptor.SkillsSHA256)
	namespace := filepath.Join(filepath.Dir(f.exe), ".axiom-docs", "axiom")
	f.root = filepath.Join(namespace, "generations", bundle.Generation)
	for name, wire := range bundle.Files {
		path := filepath.Join(f.root, filepath.FromSlash(name))
		must(f.t, os.MkdirAll(filepath.Dir(path), 0o700))
		must(f.t, os.WriteFile(path, wire, 0o400))
	}
	for dir := namespace; dir != filepath.Dir(filepath.Dir(f.exe)); dir = filepath.Dir(dir) {
		must(f.t, os.Chmod(dir, 0o700))
	}
	f.bind(bundle)
}

func (f *fixture) bind(bundle Bundle) {
	digest, info, err := hashBinary(f.exe)
	must(f.t, err)
	parent, err := os.Lstat(filepath.Dir(f.exe))
	must(f.t, err)
	pdev, pino := devIno(parent)
	bdev, bino := devIno(info)
	b := Binding{1, f.exe, pdev, pino, bdev, bino, digest, f.descriptor.SourceCommit, "0.2.0", strings.Repeat("5", 64), strings.Repeat("6", 64), bundle.Generation, bundle.ProvenanceSHA256, bundle.ManMapSHA256, strings.Repeat("a", 64)}
	wire, _ := b.Wire()
	dir := filepath.Join(filepath.Dir(f.exe), ".axiom-docs", "axiom", "bindings", digest)
	must(f.t, os.MkdirAll(dir, 0o700))
	must(f.t, os.WriteFile(filepath.Join(dir, Hash(wire)+".json"), wire, 0o400))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestResolveVerifiedAndSnapshot(t *testing.T) {
	f := newFixture(t, "binary-A", "A")
	got := Resolve(f.exe, f.descriptor)
	if got.State != StateVerified || got.Reason != "ok" || got.Bundle.Generation != f.descriptor.Generation {
		t.Fatalf("%+v", got)
	}
	if string(got.Bundle.Files["docs/text/index.md"]) != "# docs/text/index.md A\n" || !strings.HasSuffix(got.Locator("docs/text/index.md"), "index.md") {
		t.Fatal("snapshot or locator wrong")
	}
	link := filepath.Join(filepath.Dir(filepath.Dir(f.exe)), "alias")
	must(t, os.Symlink(f.exe, link))
	if via := Resolve(link, f.descriptor); via.State != StateVerified || via.Binary != f.exe {
		t.Fatalf("alias: %+v", via)
	}
}

func TestResolveRefusals(t *testing.T) {
	cases := []struct {
		name, state, reason string
		mutate              func(*fixture)
	}{
		{"unqualified", StateUnknown, "source_unqualified", func(f *fixture) { f.descriptor = Descriptor{} }},
		{"wrong host", StateUnknown, "source_unqualified", func(f *fixture) {
			f.descriptor.Architecture = "riscv"
		}},
		{"C2 disk bytes replaced under running A", StateAbsent, "binding_absent", func(f *fixture) { must(f.t, os.WriteFile(f.exe, []byte("binary-B"), 0o700)) }},
		{"B installed with its own generation", StateInvalid, "identity_mismatch", func(f *fixture) {
			old := f.descriptor
			must(f.t, os.WriteFile(f.exe, []byte("binary-B"), 0o700))
			f.publish("B")
			f.descriptor = old
		}},
		{"moved binary", StateAbsent, "binding_absent", func(f *fixture) {
			moved := filepath.Join(filepath.Dir(f.exe), "moved")
			must(f.t, os.Rename(f.exe, moved))
			f.exe = moved
		}},
		{"content tampered", StateInvalid, "hash_mismatch", func(f *fixture) {
			p := filepath.Join(f.root, "docs/text/index.md")
			must(f.t, os.Chmod(p, 0o600))
			must(f.t, os.WriteFile(p, []byte("# tampered\n"), 0o400))
		}},
		{"extra file", StateInvalid, "inventory_mismatch", func(f *fixture) {
			must(f.t, os.WriteFile(filepath.Join(f.root, "docs/text/extra.md"), []byte("x\n"), 0o400))
		}},
		{"missing generation", StateAbsent, "binding_absent", func(f *fixture) { must(f.t, os.RemoveAll(f.root)) }},
		{"group writable namespace", StateInvalid, "unsafe_permissions", func(f *fixture) {
			must(f.t, os.Chmod(filepath.Join(filepath.Dir(f.exe), ".axiom-docs", "axiom"), 0o770))
		}},
		{"symlinked content", StateInvalid, "unsafe_path", func(f *fixture) {
			p := filepath.Join(f.root, "docs/text/index.md")
			must(f.t, os.Remove(p))
			must(f.t, os.Symlink(f.exe, p))
		}},
		{"corrupt binding", StateInvalid, "malformed_record", func(f *fixture) {
			digest, _, _ := hashBinary(f.exe)
			dir := filepath.Join(filepath.Dir(f.exe), ".axiom-docs", "axiom", "bindings", digest)
			entries, _ := os.ReadDir(dir)
			must(f.t, os.Chmod(filepath.Join(dir, entries[0].Name()), 0o600))
			must(f.t, os.WriteFile(filepath.Join(dir, entries[0].Name()), []byte("{}\n"), 0o400))
		}},
		{"binding conflict", StateInvalid, "binding_conflict", func(f *fixture) {
			digest, info, _ := hashBinary(f.exe)
			parent, _ := os.Lstat(filepath.Dir(f.exe))
			pdev, pino := devIno(parent)
			bdev, bino := devIno(info)
			b := Binding{1, f.exe, pdev, pino, bdev, bino, digest, f.descriptor.SourceCommit, "0.3.0", strings.Repeat("5", 64), strings.Repeat("6", 64), f.descriptor.Generation, strings.Repeat("1", 64), strings.Repeat("2", 64), strings.Repeat("a", 64)}
			wire, _ := b.Wire()
			must(f.t, os.WriteFile(filepath.Join(filepath.Dir(f.exe), ".axiom-docs", "axiom", "bindings", digest, Hash(wire)+".json"), wire, 0o400))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t, "binary-A", "A")
			c.mutate(f)
			got := Resolve(f.exe, f.descriptor)
			if got.State != c.state || got.Reason != c.reason {
				t.Fatalf("got %s/%s want %s/%s", got.State, got.Reason, c.state, c.reason)
			}
			if got.Root != "" || len(got.Bundle.Files) != 0 {
				t.Fatal("refusal exposed paths or bytes")
			}
		})
	}
}

func TestResolveNeverWrites(t *testing.T) {
	f := newFixture(t, "binary-A", "A")
	before := snapshotTree(t, filepath.Dir(f.exe))
	Resolve(f.exe, f.descriptor)
	if after := snapshotTree(t, filepath.Dir(f.exe)); before != after {
		t.Fatal("resolver changed the tree")
	}
}

func snapshotTree(t *testing.T, root string) string {
	var out strings.Builder
	must(t, filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		fmt.Fprintf(&out, "%s %v %d %v\n", p, info.Mode(), info.Size(), info.ModTime().UnixNano())
		return err
	}))
	return out.String()
}
