package codexruntime

import (
	"io/fs"
	"strings"
	"testing"
)

// Static consistency only. This is never behavioral evidence of a host reply.
func TestResponseContextGuidanceSharedAcrossPackagedSkills(t *testing.T) {
	var baseline string
	for _, name := range []string{"axiom-project", "axiom-runtime", "axiom-work-item", "axiom-workflow"} {
		wire, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		_, section, found := strings.Cut(string(wire), "### Response context header\n")
		if !found {
			t.Fatalf("missing header guidance: %s", name)
		}
		section, _, found = strings.Cut(section, "The host writes the normal conversational response itself:")
		if !found || strings.TrimSpace(section) == "" {
			t.Fatalf("missing presentation boundary: %s", name)
		}
		if baseline == "" {
			baseline = section
		} else if section != baseline {
			t.Fatalf("shared response context contract drifted: %s", name)
		}
	}
}
