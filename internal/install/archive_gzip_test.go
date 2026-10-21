package install

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Recompress one tar stream across gzip members, not concatenated tar files.
func gzipMembers(t *testing.T, raw []byte, cuts ...int) []byte {
	t.Helper()
	var out bytes.Buffer
	start := 0
	for _, end := range append(cuts, len(raw)) {
		w := gzip.NewWriter(&out)
		if _, err := w.Write(raw[start:end]); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		start = end
	}
	return out.Bytes()
}

func tarBytes(t *testing.T, archive string) []byte {
	t.Helper()
	wire, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(wire))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func replaceArchive(t *testing.T, archive, sums string, wire []byte) {
	t.Helper()
	writeFile(t, archive, wire, 0o600)
	writeFile(t, sums, []byte(digest(wire)+"  "+filepath.Base(archive)+"\n"), 0o600)
}

func TestLegacyGzipMembers(t *testing.T) {
	for _, split := range []string{"header", "body", "metadata"} {
		t.Run(split, func(t *testing.T) {
			archive, sums := newBundle("1.1.0", bytes.Repeat([]byte("binary"), 100)).write(t, t.TempDir())
			raw := tarBytes(t, archive)
			cut := 17
			if split == "body" {
				cut = bytes.Index(raw, []byte("binarybinary")) + 3
			}
			if split == "metadata" {
				cut = bytes.Index(raw, []byte("formatVersion=1\nproduct=Axiom")) + 7
			}
			replaceArchive(t, archive, sums, gzipMembers(t, raw, cut))
			c, err := LoadCandidate(archive, sums)
			if err != nil || c.Values["platform"] != "macos-27" {
				t.Fatalf("legacy split %s: %v", split, err)
			}
		})
	}
}

func TestGzipCorruptionRefused(t *testing.T) {
	for _, version := range []int{1, 2} {
		for _, mutation := range []string{"truncated", "crc", "split crc", "extra member", "trailing"} {
			t.Run(string(rune('0'+version))+"/"+mutation, func(t *testing.T) {
				var archive, sums string
				if version == 1 {
					archive, sums = newBundle("1.1.0", []byte("binary")).write(t, t.TempDir())
				} else {
					archive, sums = newV2Archive(t).write(t)
				}
				raw := tarBytes(t, archive)
				wire := gzipMembers(t, raw)
				switch mutation {
				case "truncated":
					wire = wire[:len(wire)-3]
				case "crc":
					wire[len(wire)-8] ^= 1
				case "split crc":
					wire = gzipMembers(t, raw, 17)
					wire[len(wire)-8] ^= 1
				case "extra member":
					if version == 1 {
						return
					} // Legacy allows gzip members of one tar stream.
					wire = append(wire, gzipMembers(t, nil)...)
				case "trailing":
					wire = append(wire, []byte("garbage")...)
				}
				replaceArchive(t, archive, sums, wire)
				if _, err := LoadCandidate(archive, sums); err == nil {
					t.Fatal("accepted corrupt/ambiguous gzip")
				}
			})
		}
	}
}

func TestV2SplitGzipRefused(t *testing.T) {
	archive, sums := newV2Archive(t).write(t)
	replaceArchive(t, archive, sums, gzipMembers(t, tarBytes(t, archive), 17))
	if _, err := LoadCandidate(archive, sums); err == nil {
		t.Fatal("accepted split v2 gzip")
	}
}

func TestLegacySplitGzipRetainsArchiveRefusals(t *testing.T) {
	for _, mutation := range []string{"path", "link", "limit", "checksum"} {
		t.Run(mutation, func(t *testing.T) {
			b := newBundle("1.1.0", []byte("binary"))
			switch mutation {
			case "path":
				b.files["../escape"] = []byte("x")
			case "link":
				b.links["link"] = "/foreign"
			case "limit":
				b.files["LICENSE"] = bytes.Repeat([]byte("x"), maxBundleFile+1)
			case "checksum":
				b.skills[skillNames[0]] = []byte("changed")
				b.files["skills-manifest.txt"] = []byte("invalid")
			}
			archive, sums := b.write(t, t.TempDir())
			replaceArchive(t, archive, sums, gzipMembers(t, tarBytes(t, archive), 17))
			if _, err := LoadCandidate(archive, sums); err == nil {
				t.Fatal("accepted invalid legacy archive")
			}
		})
	}
}

func TestGzipAmbiguousTarRefused(t *testing.T) {
	for _, version := range []int{1, 2} {
		for _, mutation := range []string{"complete tars", "duplicate directory"} {
			t.Run(string(rune('0'+version))+"/"+mutation, func(t *testing.T) {
				var archive, sums string
				if version == 1 {
					archive, sums = newBundle("1.1.0", []byte("binary")).write(t, t.TempDir())
				} else {
					archive, sums = newV2Archive(t).write(t)
				}
				raw := tarBytes(t, archive)
				if mutation == "complete tars" {
					wire := append(gzipMembers(t, raw), gzipMembers(t, raw)...)
					replaceArchive(t, archive, sums, wire)
				} else {
					raw = append(append(append([]byte{}, raw[:512]...), raw[:512]...), raw[512:]...)
					replaceArchive(t, archive, sums, gzipMembers(t, raw))
				}
				if _, err := LoadCandidate(archive, sums); err == nil {
					t.Fatal("accepted ambiguous tar")
				}
			})
		}
	}
}
