package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelpMapsEveryCodexSkillToPublicAxiomExecutable(t *testing.T) {
	var output bytes.Buffer
	if code := Help(&output); code != ExitSuccess {
		t.Fatalf("help code = %d", code)
	}
	for _, expected := range []string{"$axiom-project", "$axiom-work-item", "$axiom-workflow", "axiom [--human|--json] <command>", "--authorize-external"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("help missing %q: %s", expected, output.String())
		}
	}
	for _, legacy := range []string{"axiom-project-configure", "axiom-project-list", "axiom-project-show", "axiom-work-item-create", "axiom-work-item-run", "axiom-work-item-status"} {
		if strings.Contains(output.String(), legacy) {
			t.Fatalf("help advertises retired skill %s", legacy)
		}
	}
	if strings.Contains(output.String(), "lingo ") {
		t.Fatalf("help still presents lingo as the public executable: %s", output.String())
	}
}

func TestSkillHelpListsOnlyCurrentStableNames(t *testing.T) {
	var output bytes.Buffer
	if code := Run(context.Background(), []string{"skill", "inspect", "--help"}, nil, completionProvenance(t), &output); code != ExitSuccess {
		t.Fatalf("help=%d %s", code, output.String())
	}
	if !strings.Contains(output.String(), "axiom-project | axiom-work-item | axiom-workflow") {
		t.Fatalf("skill help inventory=%s", output.String())
	}
	for _, old := range []string{"list", "show", "create", "edit", "validate", "remove", "recover"} {
		if strings.Contains(output.String(), "workflow."+old) {
			t.Fatalf("help advertises removed route %s", old)
		}
	}
}
