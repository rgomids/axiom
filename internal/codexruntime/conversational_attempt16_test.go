package codexruntime

import (
	"io/fs"
	"strings"
	"testing"
)

// Packaged output obligations only; native and visual evidence remain separate.
func TestConversationalAttempt16AnswerAudit(t *testing.T) {
	for name, obligations := range map[string][]string{
		"axiom-project":  {"even in an explanation of internal correlation", "command status, structural validity and execution readiness together"},
		"axiom-workflow": {"NOT_OBSERVED / NOT_ESTABLISHED", "never abbreviate a Workflow digest", "command status, structural validity and execution readiness together"},
		"axiom-runtime":  {"NOT_OBSERVED / NOT_ESTABLISHED", "command status, structural validity and execution readiness together"},
	} {
		wire, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		for _, obligation := range obligations {
			if !strings.Contains(string(wire), obligation) {
				t.Errorf("%s missing answer audit: %s", name, obligation)
			}
		}
	}
}
