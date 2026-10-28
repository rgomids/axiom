package codexruntime

import (
	"io/fs"
	"strings"
	"testing"
)

// Qualifies installed guidance, not probabilistic native compliance. These rules
// precede operational detail so routine answers do not inherit authority dumps.
func TestConversationalAttempt15Guidance(t *testing.T) {
	for name, rules := range map[string][]string{
		"axiom-project": {
			"Routine list/show/validate: `Nome (slug)` only",
			"NOT_ESTABLISHED does not mean non-unique",
			"inspect the corresponding owner/configuration first",
		},
		"axiom-workflow": {
			"Each independent case starts a new fact scope",
			"output existence AND correlation",
			"Do not transfer a validator from an example",
			"not evidence that Codex is unused",
			"complete sequential Stage order",
			"same actual inputs for natural and explicit entry",
		},
		"axiom-runtime": {
			"Each independent case starts a new fact scope",
			"not evidence that Codex is unused",
			"inspect the corresponding owner/configuration first",
		},
	} {
		wire, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		text := string(wire)
		for _, rule := range rules {
			at := strings.Index(text, rule)
			if at < 0 || at > 5500 {
				t.Errorf("%s missing early corrective rule %q", name, rule)
			}
		}
	}
}
