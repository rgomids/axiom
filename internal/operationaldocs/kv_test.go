package operationaldocs

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func h(c string) string { return strings.Repeat(c, 64) }

func validMetadata() string {
	c := strings.Repeat("b", 40)
	return "formatVersion=2\nproduct=Axiom\nversion=0.2.0\nrevision=" + c[:12] + "\nsourceState=clean\nrelease=true\nplatform=macos\ngoos=darwin\narchitecture=arm64\nskillSetVersion=1\nsourceCommit=" + c + "\narchiveGeneration=2\ndocsInventorySha256=" + h("1") + "\ndocsProvenanceSha256=" + h("2") + "\ndocsManMapSha256=" + h("3") + "\nbinarySha256=" + h("4") + "\n"
}

func validReceipt() string {
	c := strings.Repeat("b", 40)
	gen := Generation(h("1"), h("2"), h("3"))
	return "formatVersion=2\ndestination=/Users/a b/bin/axiom\nsha256=" + h("4") + "\nproduct=Axiom\nversion=0.2.0\nrevision=" + c[:12] + "\nsourceState=clean\nrelease=true\nplatform=linux\ngoos=linux\narchitecture=amd64\nskillSetVersion=1\narchiveSha256=" + h("5") + "\nskillManifestSha256=" + h("6") + "\ninstalledAt=2026-10-20T10:00:00Z\nsourceCommit=" + c + "\ndocsGeneration=" + gen + "\ndocsBindingSha256=" + h("7") + "\ndocsInventorySha256=" + h("1") + "\ndocsProvenanceSha256=" + h("2") + "\ndocsManMapSha256=" + h("3") + "\n"
}

func mutations(wire string) []string {
	lines := strings.Split(strings.TrimSuffix(wire, "\n"), "\n")
	var out []string
	join := func(l []string) string { return strings.Join(l, "\n") + "\n" }
	for i := range lines {
		removed := append(append([]string{}, lines[:i]...), lines[i+1:]...)
		out = append(out, join(removed))
		dup := append(append([]string{}, lines[:i+1]...), lines[i:]...)
		out = append(out, join(dup))
		swap := append([]string{}, lines...)
		j := i + 1
		if j == len(lines) {
			j = i - 1
		}
		swap[i], swap[j] = swap[j], swap[i]
		out = append(out, join(swap))
		empty := append([]string{}, lines...)
		empty[i] = strings.SplitN(lines[i], "=", 2)[0] + "="
		out = append(out, join(empty))
	}
	out = append(out, join(append(append([]string{}, lines...), "extra=1")), wire[:len(wire)-1], strings.ReplaceAll(wire, "\n", "\r\n"), "\n"+wire, wire+"\n")
	return out
}

func TestMetadataRoundTripAndNegatives(t *testing.T) {
	m, err := ParseMetadata([]byte(validMetadata()))
	if err != nil || !m.Release || m.Platform != "macos" {
		t.Fatal(err)
	}
	for i, wire := range mutations(validMetadata()) {
		if _, err := ParseMetadata([]byte(wire)); err == nil {
			t.Fatalf("mutation %d accepted", i)
		}
	}
	for _, edit := range [][2]string{{"formatVersion=2", "formatVersion=1"}, {"release=true", "release=yes"}, {"sourceState=clean", "sourceState=dirty"}, {"goos=darwin", "goos=linux"}, {"archiveGeneration=2", "archiveGeneration=1"}, {"version=0.2.0", "version=01.2.0"}, {"revision=bbbbbbbbbbbb", "revision=aaaaaaaaaaaa"}, {"skillSetVersion=1", "skillSetVersion=2"}} {
		if _, err := ParseMetadata([]byte(strings.Replace(validMetadata(), edit[0], edit[1], 1))); err == nil {
			t.Fatalf("accepted %v", edit)
		}
	}
	if _, err := ParseMetadata([]byte(strings.Repeat("a", MaxMetadata+1))); err == nil || err.Error() != "quota_exceeded" {
		t.Fatal("oversize not refused as quota")
	}
}

func TestReceiptRoundTripAndNegatives(t *testing.T) {
	r, err := ParseReceipt([]byte(validReceipt()))
	if err != nil || r.Destination != "/Users/a b/bin/axiom" {
		t.Fatal(err)
	}
	for i, wire := range mutations(validReceipt()) {
		if _, err := ParseReceipt([]byte(wire)); err == nil {
			t.Fatalf("mutation %d accepted", i)
		}
	}
	for _, edit := range [][2]string{{"destination=/Users/a b/bin/axiom", "destination=relative/axiom"}, {"destination=/Users/a b/bin/axiom", "destination=/a/../axiom"}, {"installedAt=2026-10-20T10:00:00Z", "installedAt=2026-02-30T10:00:00Z"}, {"docsInventorySha256=" + h("1"), "docsInventorySha256=" + h("9")}} {
		if _, err := ParseReceipt([]byte(strings.Replace(validReceipt(), edit[0], edit[1], 1))); err == nil {
			t.Fatalf("accepted %v", edit)
		}
	}
}

func validBinding() string {
	c := strings.Repeat("b", 40)
	return `{"schemaVersion":1,"binaryPath":"/opt/axiom/axiom","parentDevice":1,"parentInode":2,"binaryDevice":1,"binaryInode":3,"binarySha256":"` + h("4") + `","sourceCommit":"` + c + `","version":"0.2.0","archiveSha256":"` + h("5") + `","metadataSha256":"` + h("6") + `","generation":"` + h("7") + `","provenanceSha256":"` + h("8") + `","manMapSha256":"` + h("9") + `","skillManifestSha256":"` + h("a") + `"}` + "\n"
}

func TestBindingCanonicalHashAndNegatives(t *testing.T) {
	b, err := ParseBinding([]byte(validBinding()))
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := b.Wire()
	if string(wire) != validBinding() {
		t.Fatalf("emit order differs: %s", wire)
	}
	if got, _ := b.Hash(); got != Hash([]byte(validBinding())) {
		t.Fatal("hash not over compact JSON plus LF")
	}
	for _, edit := range [][2]string{{`"schemaVersion":1`, `"schemaVersion":2`}, {`"parentInode":2`, `"parentInode":-2`}, {`"parentInode":2`, `"parentInode":9007199254740992`}, {`"parentInode":2`, `"parentInode":2.5`}, {`"binaryPath":"/opt/axiom/axiom"`, `"binaryPath":"opt"`}, {`"version":"0.2.0"`, `"version":"v0.2.0"`}, {`"generation":"` + h("7") + `"`, `"generation":"` + strings.ToUpper(h("a")) + `"`}, {`"manMapSha256"`, `"manMapSha256x"`}, {`"binaryInode":3`, `"binaryInode":3,"binaryInode":4`}, {`"version":"0.2.0",`, ``}} {
		if _, err := ParseBinding([]byte(strings.Replace(validBinding(), edit[0], edit[1], 1))); err == nil {
			t.Fatalf("accepted %v", edit)
		}
	}
	if _, err := ParseBinding([]byte(validBinding() + validBinding())); err == nil {
		t.Fatal("trailing value accepted")
	}
}

func TestAcceptedPositiveVectorsDecodeAndRoundTrip(t *testing.T) {
	wire, err := os.ReadFile("../../docs/specifications/010-installed-operational-documentation/contract-vectors-proposed.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		PositiveRecords struct {
			Metadata   string          `json:"metadata"`
			Inventory  string          `json:"inventory"`
			Provenance string          `json:"provenance"`
			ManMap     string          `json:"manMap"`
			Binding    json.RawMessage `json:"binding"`
			Receipt    string          `json:"receipt"`
		} `json:"positiveRecords"`
	}
	if err := json.Unmarshal(wire, &vectors); err != nil {
		t.Fatal(err)
	}
	records := vectors.PositiveRecords
	if _, err := ParseMetadata([]byte(records.Metadata)); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if _, err := ParseReceipt([]byte(records.Receipt)); err != nil {
		t.Fatalf("receipt: %v", err)
	}
	for name, spec := range map[string]struct {
		text, header string
		fields       int
	}{"inventory": {records.Inventory, "path\tkind\tbytes\tsha256", 4}, "provenance": {records.Provenance, "output\tsource\tsourceCommit\tsourceSha256\ttransform", 5}, "man-map": {records.ManMap, "command\tpage\tanchor", 3}} {
		if _, err := rows([]byte(spec.text), spec.header, spec.fields, 1024); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(records.Binding, &members); err != nil {
		t.Fatal(err)
	}
	binding, err := ParseBinding(append(bytes.TrimSpace(records.Binding), '\n'))
	if err != nil {
		t.Fatalf("binding: %v", err)
	}
	if again, _ := binding.Wire(); len(again) == 0 || len(members) != len(bindingFields) {
		t.Fatal("binding field set differs from the accepted vector")
	}
}
