package manifest

import (
	"io"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// These tests exercise the dependency directly, independently of DTO decoding.
func TestParserContractNodeEvidence(t *testing.T) {
	for _, source := range []string{"a: 1\na: 2\n", "outer: {inner: {x: one, x: two}}"} {
		var n yaml.Node
		if err := yaml.Unmarshal([]byte(source), &n); err != nil {
			t.Fatal(err)
		}
		m := n.Content[0]
		for len(m.Content) == 2 {
			m = m.Content[1]
		}
		if len(m.Content) != 4 || m.Content[0].Value != m.Content[2].Value {
			t.Fatal("duplicate evidence lost")
		}
	}
	var n yaml.Node
	if err := yaml.Unmarshal([]byte("a: &ref {x: value}\nb: *ref\nc: {<<: *ref}\nd: !custom data"), &n); err != nil {
		t.Fatal(err)
	}
	c := n.Content[0].Content
	if c[1].Anchor != "ref" || c[3].Kind != yaml.AliasNode || c[3].Alias != c[1] || c[5].Content[0].Tag != "!!merge" || c[7].Tag != "!custom" {
		t.Fatal("forbidden syntax evidence lost")
	}
}

func TestParserContractScalarTyping(t *testing.T) {
	for _, tc := range []struct{ source, tag, value string }{
		{"1", "!!int", "1"}, {"01", "!!int", "01"}, {"0x1", "!!int", "0x1"},
		{"1.0", "!!float", "1.0"}, {"true", "!!bool", "true"}, {"null", "!!null", "null"},
		{`"1"`, "!!str", "1"}, {"yes", "!!str", "yes"}, {"on", "!!str", "on"},
		{"2026-09-15", "!!timestamp", "2026-09-15"}, {"!!str 1", "!!str", "1"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			var n yaml.Node
			if err := yaml.Unmarshal([]byte(tc.source), &n); err != nil {
				t.Fatal(err)
			}
			if n.Content[0].Tag != tc.tag || n.Content[0].Value != tc.value {
				t.Fatal("scalar evidence changed")
			}
		})
	}
}

func TestParserContractStreamAndHostileBounds(t *testing.T) {
	d := yaml.NewDecoder(strings.NewReader("a: b\n---\nc: d\n"))
	var n yaml.Node
	if d.Decode(&n) != nil || d.Decode(&n) != nil || d.Decode(&n) != io.EOF {
		t.Fatal("stream boundary lost")
	}
	// Dependency has hard flow/indent depth caps; node parsing does not expand aliases.
	for _, s := range []string{strings.Repeat("[", 10001) + "x" + strings.Repeat("]", 10001), "a: *missing", "a: [unterminated"} {
		if err := yaml.Unmarshal([]byte(s), &n); err == nil {
			t.Fatal("hostile syntax accepted")
		}
	}
	var wide yaml.Node
	if err := yaml.Unmarshal([]byte("["+strings.Repeat("x,", 20000)+"]"), &wide); err != nil || len(wide.Content[0].Content) != 20000 {
		t.Fatal("wide node evidence unavailable")
	}
}

func TestTreeLimitBoundaries(t *testing.T) {
	for _, depth := range []int{MaxDepth, MaxDepth + 1} {
		var n yaml.Node
		source := strings.Repeat("[", depth-1) + "x" + strings.Repeat("]", depth-1)
		if err := yaml.Unmarshal([]byte(source), &n); err != nil {
			t.Fatal(err)
		}
		issues := validateTree(n.Content[0])
		if (len(issues) == 0) != (depth == MaxDepth) {
			t.Fatal("depth boundary off by one")
		}
	}
}
