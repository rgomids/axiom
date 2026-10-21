package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type v2Archive struct {
	files   map[string][]byte
	modes   map[string]int64
	root    string
	format  tar.Format
	extra   []byte
	trailer []byte
	pax     map[string]string
}

func newV2Archive(t *testing.T) *v2Archive {
	t.Helper()
	commit := strings.Repeat("b", 40)
	files := map[string][]byte{binaryName: []byte("binary"), "LICENSE": []byte("license\n"), "install.sh": []byte("#!/bin/sh\n")}
	var skillLines strings.Builder
	skillLines.WriteString("formatVersion=1\nskillSetVersion=1\nbinaryCompatibility=1\n")
	for _, name := range skillNames {
		wire := []byte("skill " + name + "\n")
		files["skills/"+name+"/SKILL.md"] = wire
		fmt.Fprintf(&skillLines, "skill.%s=%s\n", name, digest(wire))
	}
	files["skills-manifest.txt"] = []byte(skillLines.String())
	var inventory, provenance strings.Builder
	inventory.WriteString("formatVersion=1\npath\tkind\tbytes\tsha256\n")
	provenance.WriteString("formatVersion=1\noutput\tsource\tsourceCommit\tsourceSha256\ttransform\n")
	names := []string{"docs/man/man1/axiom.1"}
	for _, n := range []string{"index", "concepts", "installation", "project", "runtime", "workflow", "work-item", "configuration", "troubleshooting", "cli", "notices"} {
		names = append(names, "docs/text/"+n+".md")
	}
	sort.Strings(names)
	for _, name := range names {
		body := []byte("# " + name + "\n")
		kind := "markdown"
		if strings.HasSuffix(name, ".1") {
			body, kind = []byte(".TH AXIOM 1\n.SH ROOT\nx\n"), "man1"
		}
		files[name] = body
		fmt.Fprintf(&inventory, "%s\t%s\t%d\t%s\n", name, kind, len(body), digest(body))
		fmt.Fprintf(&provenance, "%s\tdocs/operational/index.md\t%s\t%s\toperational-synthesis-v1\n", name, commit, strings.Repeat("e", 64))
	}
	files["docs/inventory.tsv"] = []byte(inventory.String())
	files["docs/provenance.tsv"] = []byte(provenance.String())
	files["docs/man-map.tsv"] = []byte("formatVersion=1\ncommand\tpage\tanchor\naxiom\tdocs/man/man1/axiom.1\tROOT\n")
	files["release-metadata.txt"] = []byte("formatVersion=2\nproduct=Axiom\nversion=0.2.0\nrevision=" + commit[:12] + "\nsourceState=clean\nrelease=true\nplatform=linux\ngoos=linux\narchitecture=amd64\nskillSetVersion=1\nsourceCommit=" + commit + "\narchiveGeneration=2\ndocsInventorySha256=" + digest(files["docs/inventory.tsv"]) + "\ndocsProvenanceSha256=" + digest(files["docs/provenance.tsv"]) + "\ndocsManMapSha256=" + digest(files["docs/man-map.tsv"]) + "\nbinarySha256=" + digest(files[binaryName]) + "\n")
	a := &v2Archive{files: files, modes: map[string]int64{}, root: "axiom-0.2.0-linux-amd64", format: tar.FormatUSTAR}
	a.seal()
	return a
}

func (a *v2Archive) seal() {
	delete(a.files, "MANIFEST.sha256")
	names := make([]string, 0, len(a.files))
	for n := range a.files {
		names = append(names, n)
	}
	sort.Strings(names)
	var m strings.Builder
	for _, n := range names {
		fmt.Fprintf(&m, "%s  %s\n", digest(a.files[n]), n)
	}
	a.files["MANIFEST.sha256"] = []byte(m.String())
}

func (a *v2Archive) write(t *testing.T) (string, string) {
	t.Helper()
	var raw bytes.Buffer
	zw := gzip.NewWriter(&raw)
	tw := tar.NewWriter(zw)
	dirs := map[string]bool{}
	names := make([]string, 0, len(a.files))
	for n := range a.files {
		names = append(names, n)
	}
	sort.Strings(names)
	writeDir := func(d string) {
		if !dirs[d] {
			dirs[d] = true
			if err := tw.WriteHeader(&tar.Header{Name: d + "/", Typeflag: tar.TypeDir, Mode: 0o700, Format: a.format}); err != nil {
				t.Fatal(err)
			}
		}
	}
	writeDir(a.root)
	for _, n := range names {
		parts := strings.Split(filepath.Dir(n), "/")
		for i := range parts {
			if parts[i] != "." {
				writeDir(a.root + "/" + strings.Join(parts[:i+1], "/"))
			}
		}
		mode := v2Mode(n)
		if override, ok := a.modes[n]; ok {
			mode = override
		}
		if err := tw.WriteHeader(&tar.Header{Name: a.root + "/" + n, Typeflag: tar.TypeReg, Mode: mode, Size: int64(len(a.files[n])), Format: a.format, PAXRecords: a.pax}); err != nil {
			t.Fatal(err)
		}
		tw.Write(a.files[n])
	}
	tw.Close()
	zw.Close()
	raw.Write(a.trailer)
	dir := t.TempDir()
	archive := filepath.Join(dir, a.root+".tar.gz")
	os.WriteFile(archive, raw.Bytes(), 0o600)
	sums := filepath.Join(dir, "SHA256SUMS")
	os.WriteFile(sums, []byte(digest(raw.Bytes())+"  "+filepath.Base(archive)+"\n"), 0o600)
	return archive, sums
}

func TestLoadCandidateV2Accepts(t *testing.T) {
	archive, sums := newV2Archive(t).write(t)
	c, err := LoadCandidate(archive, sums)
	if err != nil {
		t.Fatal(err)
	}
	if c.Docs == nil || c.DocsMetadata == nil || c.Docs.Generation == "" || c.Values["platform"] != "linux" || len(c.Skills) != 4 {
		t.Fatalf("%+v", c)
	}
}

func TestLoadCandidateV2Refusals(t *testing.T) {
	cases := map[string]func(*v2Archive){
		"extra file":          func(a *v2Archive) { a.files["docs/text/extra.md"] = []byte("x\n"); a.seal() },
		"unlisted extra root": func(a *v2Archive) { a.files["notes.txt"] = []byte("x\n"); a.seal() },
		"missing control":     func(a *v2Archive) { delete(a.files, "docs/man-map.tsv"); a.seal() },
		"binary swapped":      func(a *v2Archive) { a.files[binaryName] = []byte("other"); a.seal() },
		"content swapped":     func(a *v2Archive) { a.files["docs/text/index.md"] = []byte("# changed\n"); a.seal() },
		"manifest tampered":   func(a *v2Archive) { a.files["MANIFEST.sha256"] = append(a.files["MANIFEST.sha256"], '\n') },
		"binary mode":         func(a *v2Archive) { a.modes[binaryName] = 0o755 },
		"setuid":              func(a *v2Archive) { a.modes[binaryName] = 0o4700 },
		"text mode":           func(a *v2Archive) { a.modes["LICENSE"] = 0o644 },
		"wrong root":          func(a *v2Archive) { a.root = "axiom-0.2.0-linux-arm64" },
		"pax record":          func(a *v2Archive) { a.format = tar.FormatPAX; a.pax = map[string]string{"comment": "x"} },
		"gnu format":          func(a *v2Archive) { a.format = tar.FormatGNU },
		"uppercase path":      func(a *v2Archive) { a.files["docs/text/BAD.md"] = []byte("x\n"); a.seal() },
		"trailing garbage":    func(a *v2Archive) { a.trailer = []byte("garbage") },
		"metadata v1 keys": func(a *v2Archive) {
			a.files["release-metadata.txt"] = bytes.Replace(a.files["release-metadata.txt"], []byte("formatVersion=2"), []byte("formatVersion=1"), 1)
			a.seal()
		},
		"metadata docs hash": func(a *v2Archive) {
			a.files["release-metadata.txt"] = bytes.Replace(a.files["release-metadata.txt"], []byte("docsManMapSha256="+digest(a.files["docs/man-map.tsv"])), []byte("docsManMapSha256="+strings.Repeat("0", 64)), 1)
			a.seal()
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			a := newV2Archive(t)
			mutate(a)
			archive, sums := a.write(t)
			if _, err := LoadCandidate(archive, sums); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}
