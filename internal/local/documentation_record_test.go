package local_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/local"
	"github.com/rgomids/axiom/internal/projectapp"
)

func documentationRecord(t *testing.T) local.Record {
	t.Helper()
	r, issues := local.DecodeRecord(fixture(t, "documentation-v2"))
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return r
}

func TestFormatTwoRoundTripsDocumentationBindings(t *testing.T) {
	r := documentationRecord(t)
	bindings := r.State().Documentation
	if len(bindings) != 1 || bindings[0].SourceKey != "product-notes" || bindings[0].Observation.Availability != projectapp.Available {
		t.Fatalf("binding lost: %+v", bindings)
	}
	wire, issues := local.EncodeRecord(r)
	if len(issues) != 0 || !bytes.Contains(wire, []byte(`"formatVersion":2`)) {
		t.Fatalf("format 2 not written: %s %v", wire, issues)
	}
	again, issues := local.DecodeRecord(wire)
	if len(issues) != 0 || !bytes.Equal(must(t, again), wire) {
		t.Fatal("format 2 round trip unstable")
	}
}

func must(t *testing.T, r local.Record) []byte {
	t.Helper()
	b, issues := local.EncodeRecord(r)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	return b
}

func TestFormatOneStaysFormatOneWithoutDocumentation(t *testing.T) {
	for _, name := range []string{"minimal", "configured"} {
		r, issues := local.DecodeRecord(fixture(t, name))
		if len(issues) != 0 {
			t.Fatal(issues)
		}
		wire := must(t, r)
		if !bytes.Contains(wire, []byte(`"formatVersion":1`)) || bytes.Contains(wire, []byte("documentationBindings")) {
			t.Fatalf("format 1 changed: %s", wire)
		}
	}
	// Removing every documentation binding returns to format 1, never an empty format 2.
	state := documentationRecord(t).State()
	state.Documentation = nil
	r, issues := local.NewRecord(state)
	if len(issues) != 0 || !bytes.Contains(must(t, r), []byte(`"formatVersion":1`)) {
		t.Fatal("empty bindings did not use format 1")
	}
}

func TestFormatTwoRejectsInvalidDocumentationBindings(t *testing.T) {
	base := fixture(t, "documentation-v2")
	mutate := func(edit func(map[string]any)) []byte {
		var tree map[string]any
		if err := json.Unmarshal(base, &tree); err != nil {
			t.Fatal(err)
		}
		edit(tree)
		b, _ := json.Marshal(tree)
		return b
	}
	binding := func(tree map[string]any) map[string]any {
		return tree["documentationBindings"].([]any)[0].(map[string]any)
	}
	for name, input := range map[string][]byte{
		"format 1 with bindings": mutate(func(m map[string]any) { m["formatVersion"] = 1 }),
		"format 2 without":       mutate(func(m map[string]any) { delete(m, "documentationBindings") }),
		"format 2 empty":         mutate(func(m map[string]any) { m["documentationBindings"] = []any{} }),
		"document body":          mutate(func(m map[string]any) { binding(m)["content"] = "synthetic body" }),
		"content digest":         mutate(func(m map[string]any) { binding(m)["contentDigest"] = strings.Repeat("a", 64) }),
		"relative path":          mutate(func(m map[string]any) { binding(m)["explicitPath"] = "notes/product.md" }),
		"bad identity":           mutate(func(m map[string]any) { binding(m)["canonicalIdentity"] = "fs:" + strings.Repeat("a", 64) }),
		"uppercase identity":     mutate(func(m map[string]any) { binding(m)["canonicalIdentity"] = "file:" + strings.Repeat("A", 64) }),
		"bad key":                mutate(func(m map[string]any) { binding(m)["sourceKey"] = "../notes" }),
		"bad observation": mutate(func(m map[string]any) {
			binding(m)["observation"] = map[string]any{"availability": "maybe", "basis": "x", "observedAt": "x"}
		}),
		"duplicate key": mutate(func(m map[string]any) {
			list := m["documentationBindings"].([]any)
			m["documentationBindings"] = append(list, list[0])
		}),
		"newer format": mutate(func(m map[string]any) { m["formatVersion"] = 3 }),
	} {
		t.Run(name, func(t *testing.T) { rejected(t, input) })
	}
}
