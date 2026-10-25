package codexruntime

import (
	"io/fs"
	"strings"
	"testing"
)

// These assertions qualify packaged instructions, not native model compliance.
// Keep attempt-12 outputs and all prior fixtures immutable.
func TestConversationalAttempt13Guidance(t *testing.T) {
	for name, requirements := range map[string][]string{
		"axiom-project": {
			"For a routine unique Project, answer `Órbita (orbita)`",
			"A missing name OR slug alone never requires displaying the UUID",
			"Separate observations stay separate",
			"11111111-1111-4111-8111-111111111111",
			"11111111-1111-4111-8111-111111111112",
			"actually print BOTH complete canonical IDs",
			"Partial inventory means uniqueness is NOT_ESTABLISHED",
			"Without a canonical ID there is no actionable target",
		},
		"axiom-workflow": {
			"Retain the exact Agent declaration `profileRef: policy-default`",
			"NOT_OBSERVED is not FALSE",
			"Compatibility has its own source",
			"A revision label such as `public-r2` does not establish freshness",
			"Natural and explicit entry must preserve the same facts",
		},
		"axiom-runtime": {
			"Retain the exact Agent declaration `profileRef: policy-default`",
			"NOT_OBSERVED is not FALSE",
			"Compatibility has its own source",
			"A revision label such as `public-r2` does not establish freshness",
		},
	} {
		wire, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range requirements {
			if !strings.Contains(string(wire), want) {
				t.Errorf("%s missing corrective rule %q", name, want)
			}
		}
	}
}
