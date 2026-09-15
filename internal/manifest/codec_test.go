package manifest_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/manifest"
	"github.com/rgomids/axiom/internal/project"
)

const minimal = "schemaVersion: 1\nproject: {id: 12345678-1234-4abc-8def-123456789abc, slug: sample, name: Sample}\n"
const sentinel = "SYNTHETIC_DIAGNOSTIC_SENTINEL"

func decode(t *testing.T, source string) project.Project {
	t.Helper()
	p, issues := manifest.Decode([]byte(source))
	if len(issues) != 0 {
		t.Fatalf("decode: %v", issues)
	}
	return p
}
func encode(t *testing.T, p project.Project) []byte {
	t.Helper()
	b, issues := manifest.Encode(p)
	if len(issues) != 0 {
		t.Fatalf("encode: %v", issues)
	}
	return b
}
func reject(t *testing.T, source string) {
	t.Helper()
	input := []byte(source)
	before := bytes.Clone(input)
	p, issues := manifest.Decode(input)
	if len(issues) == 0 || p.Equivalent(p) {
		t.Fatal("invalid input returned a valid Project")
	}
	if !bytes.Equal(input, before) {
		t.Fatal("decode mutated input")
	}
	if strings.Contains(fmt.Sprint(issues), sentinel) {
		t.Fatal("diagnostic leaked sentinel")
	}
	_, again := manifest.Decode(input)
	if fmt.Sprint(again) != fmt.Sprint(issues) {
		t.Fatal("diagnostics unstable")
	}
}
func TestSchemaVersion(t *testing.T) {
	decode(t, minimal)
	reject(t, strings.Replace(minimal, "schemaVersion: 1\n", "", 1))
	for _, version := range []string{"0", "2", `"1"`, "true", "null", "1.0", "1e0", "999", "99999999999999999999999999", "01", "0x1", "+1", "1_0"} {
		t.Run(version, func(t *testing.T) {
			reject(t, strings.Replace(minimal, "schemaVersion: 1", "schemaVersion: "+version, 1))
		})
	}
}
func TestYAMLAbuse(t *testing.T) {
	for name, source := range map[string]string{
		"duplicate root":   minimal + "schemaVersion: 1\n",
		"duplicate nested": strings.Replace(minimal, "name: Sample", "name: Sample, name: Other", 1),
		"duplicate deep":   minimal + "integrations: [{key: i, transport: {id: a, id: b}}]",
		"unknown root":     minimal + sentinel + ": data",
		"unknown nested":   strings.Replace(minimal, "name: Sample", "name: Sample, "+sentinel+": data", 1),
		"unknown deep":     minimal + "integrations: [{key: i, transport: {id: a, " + sentinel + ": data}}]",
		"anchor":           minimal + "runtime: &" + sentinel + " {id: r}",
		"alias":            minimal + "runtime: *" + sentinel,
		"recursive alias":  minimal + "runtime: &a [*a]",
		"merge":            minimal + "runtime: {<<: {id: r}}",
		"quoted merge":     minimal + `runtime: {"<<": {id: r}}`,
		"tag":              minimal + "runtime: !" + sentinel + " unconfigured",
		"tag on container": minimal + "providers: !custom []",
		"multiple":         minimal + "---\n" + minimal,
		"empty second":     minimal + "---\n",
		"non string key":   minimal + "1: data",
		"complex key":      minimal + "? [a, b]\n: data",
		"root sequence":    "[" + sentinel + "]",
		"syntax":           minimal + "runtime: [" + sentinel,
		"null":             minimal + "runtime: null",
		"binary":           minimal + "businessContext: {text: !!binary YQ==}",
		"utf8":             minimal + "#\xff",
	} {
		t.Run(name, func(t *testing.T) { reject(t, source) })
	}
}
func TestGoldenRoundTrip(t *testing.T) {
	for _, name := range []string{"minimal", "configured", "unconfigured", "empty"} {
		t.Run(name, func(t *testing.T) {
			input, err := os.ReadFile("testdata/" + name + ".yaml")
			if err != nil {
				t.Fatal(err)
			}
			golden, err := os.ReadFile("testdata/" + name + ".golden.yaml")
			if err != nil {
				t.Fatal(err)
			}
			p := decode(t, string(input))
			b := encode(t, p)
			if !bytes.Equal(b, golden) {
				t.Fatalf("golden mismatch:\n%s", b)
			}
			q := decode(t, string(b))
			if !p.Equivalent(q) {
				t.Fatal("round trip lost intent")
			}
			if !bytes.Equal(b, encode(t, q)) || !bytes.Equal(b, encode(t, p)) {
				t.Fatal("encoding unstable")
			}
		})
	}
}
