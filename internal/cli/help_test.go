package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpMapsEveryCodexSkillToLingo(t *testing.T) {
	var output bytes.Buffer
	if code := Help(&output); code != ExitSuccess {
		t.Fatalf("help code = %d", code)
	}
	for _, expected := range []string{"$axiom-project-configure", "$axiom-project-show", "$axiom-work-item-create", "$axiom-work-item-run", "$axiom-work-item-status", "lingo --json", "--authorize-external"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("help missing %q: %s", expected, output.String())
		}
	}
}
