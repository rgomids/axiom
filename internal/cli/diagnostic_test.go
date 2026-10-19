package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

func TestOperatorDiagnosticRenderingIsAdditiveAndBounded(t *testing.T) {
	c := presentationResult(t, completion.Facts{ValidationFailed: true}, "Refused", []string{"execution:synthetic"}, "Inspect state", "unchanged")
	for _, payload := range []any{map[string]any{"single": map[string]string{"status": "unknown"}}, map[string]any{"graph": map[string]string{"status": "partial"}}} {
		d := &Diagnostic{SchemaVersion: 1, Operation: "workflow stage run", Category: "capability_unknown", Capability: "unknown", Options: []DiagnosticOption{{ID: "inspect", Availability: "conditional", Operation: "workflow stage status"}}}
		base, _ := renderCompletionJSON(*c)
		wire := bytes.TrimSpace(base)
		stage, _ := json.Marshal(payload)
		diag, _ := json.Marshal(boundedDiagnostic(d))
		wire = append(wire[:len(wire)-1], []byte(",\"category\":\"capability_unknown\",\"stage\":"+string(stage)+",\"diagnostic\":"+string(diag)+"}")...)
		var human bytes.Buffer
		if presentEvent(&human, humanOutput, c.Status(), wire, 1<<20) != ExitFailure {
			t.Fatal("changed refusal exit")
		}
		rendered, err := renderTerminal(wire)
		if err != nil || human.String() != string(rendered) {
			t.Fatal("JSON/Markdown diverged")
		}
		for _, v := range []string{"unknown", "diagnostic", "schemaVersion", "capability_unknown", "unchanged"} {
			if !strings.Contains(human.String(), v) {
				t.Fatalf("lost %s", v)
			}
		}
	}
	for _, d := range []*Diagnostic{
		{SchemaVersion: 2},
		{SchemaVersion: 1, Observed: []string{strings.Repeat("x", 16*1024)}},
		{SchemaVersion: 1, Options: make([]DiagnosticOption, 4)},
		{SchemaVersion: 1, Options: []DiagnosticOption{{Availability: "unavailable", Operation: "invented"}}},
	} {
		if boundedDiagnostic(d) != nil {
			t.Fatal("unbounded/invalid enrichment accepted")
		}
	}
}

func TestOperatorDiagnosticCannotSuppressRefusalAtOutputLimit(t *testing.T) {
	c := presentationResult(t, completion.Facts{ValidationFailed: true}, "Refused", nil, "Inspect state", "")
	wire, _ := renderCompletionJSON(*c)
	d := &Diagnostic{SchemaVersion: 1, Observed: []string{strings.Repeat("x", 1024)}}
	var out bytes.Buffer
	if code := presentDiagnosticEvent(&out, jsonOutput, c.Status(), wire, len(wire), d); code != ExitFailure || !bytes.Equal(out.Bytes(), wire) {
		t.Fatal("enrichment suppressed canonical refusal")
	}
}
