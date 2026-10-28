package codexruntime

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Specification 013: the v0.22.0 shared set upgrades in place for both
// Runtimes, and the guided setup/diagnostic guidance ships in each.
func TestGuidedSetup013UpgradesPublishedSetForCodexAndClaude(t *testing.T) {
	var previous skillSetRevision
	for _, r := range sharedSkillHistory {
		if d, err := r.manifestDigest(); err == nil && d == "e29183f85292f1aeb7fe37ed6a5688e66583e293340d249666208d76b7526696" {
			previous = r
		}
	}
	if len(previous.skills) != 4 {
		t.Fatal("missing v0.22.0 shared revision")
	}
	for _, constructor := range runtimeServices(t) {
		root := privateSkillRoot(t)
		service, err := constructor(root)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(service.Runtime(), func(t *testing.T) {
			for name, digest := range previous.skills {
				wire, err := os.ReadFile("testdata/pre-guided-setup/" + name + ".md")
				if err != nil || digestOf(wire) != digest {
					t.Fatal("historical bytes drift", name, err)
				}
				if err := os.MkdirAll(filepath.Join(root, name), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, name, "SKILL.md"), wire, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			receipt, err := service.integration.receiptFor(root, previous)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, receiptName), receipt, 0o600); err != nil {
				t.Fatal(err)
			}
			installOrFail(t, service, Applied)
			if got := service.Inspect(context.Background()); got.Status != Ready || got.Receipt != ReceiptCurrent {
				t.Fatal(got)
			}
			project, _ := os.ReadFile(filepath.Join(root, "axiom-project", "SKILL.md"))
			runtime, _ := os.ReadFile(filepath.Join(root, "axiom-runtime", "SKILL.md"))
			for _, want := range []string{"## Complete guided setup", "machine-local and shared by every Project", "`default-sdd` revisions as an explicit option", "Authentication ready", "never ready", "partial bootstrap", "derive Profile needs\n   only from its Agents", "Do this before selecting"} {
				if !strings.Contains(string(project), want) {
					t.Fatal("axiom-project lacks", want)
				}
			}
			for _, want := range []string{"## Refusal diagnostics", "runtimeAuthoring.diagnostics", "None of these is authentication evidence", "`candidate_valid` proves local declaration only"} {
				if !strings.Contains(string(runtime), want) {
					t.Fatal("axiom-runtime lacks", want)
				}
			}
		})
	}
}

// The wizard coordinates owners; it never names an operation the binary
// does not expose for this skill set.
func TestGuidedSetupNamesOnlyOwnedOperations(t *testing.T) {
	project, err := fs.ReadFile(skillFiles, "skills/axiom-project/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	section := string(project)
	section = section[strings.Index(section, "## Complete guided setup"):strings.Index(section, "## configure")]
	for _, forbidden := range []string{"setup-ready", "--authorize-local\n", "auth preflight automatically", "OPENAI_API_KEY"} {
		if strings.Contains(section, forbidden) {
			t.Fatal("guided setup invents", forbidden)
		}
	}
	for _, flag := range []string{"--runtime", "--model-profile", "--runtime-preference", "--repository-remote", "--policy-draft", "--technology", "--documentation", "--business-context", "--context-source", "--glossary"} {
		if !strings.Contains(section, flag) {
			t.Fatal("guided setup omits", flag)
		}
	}
}
