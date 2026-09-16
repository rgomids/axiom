package local

import (
	"strings"
	"testing"
)

func TestTokenLimitsAtBoundary(t *testing.T) {
	// Root counts as depth/node 1. Keys count as nodes, closing delimiters do not.
	for _, input := range []string{
		strings.Repeat("[", MaxRecordDepth-1) + "0" + strings.Repeat("]", MaxRecordDepth-1),
		"[" + strings.Repeat("0,", MaxRecordNodes-2) + "0]",
	} {
		if _, issues := parseRecord([]byte(input)); len(issues) > 0 {
			t.Fatal(issues)
		}
	}
	for _, input := range []string{`{"a":1,"a":2}`, `{"a":{"b":1,"b":2}}`, `{"a":[{"b":1,"\u0062":2}]}`} {
		_, issues := parseRecord([]byte(input))
		if len(issues) == 0 || issues[0].Code != "duplicate_field" {
			t.Fatal("nested duplicate not detected")
		}
	}
}
