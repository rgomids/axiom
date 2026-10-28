package codexruntime

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Specification 013 revision 3: each Model Advisor fixture case is anchored
// to the guidance that governs it. These checks prove the shipped guidance and
// fixtures agree; they do not prove how a model converses (native NOT_RUN).
var modelAdvisorClauses = map[string][]string{
	"MA-01": {"record URL and retrieval date", "For each candidate keep", "one primary recommendation tied to\nthe Agent needs", "Mark undated, stale or\nconflicting information"},
	"MA-02": {"Economy never defaults to the most capable model", "never drops below a documented need", "at the minimum sufficient"},
	"MA-03": {"The Orchestrator needs enough capacity", "never\nneed more capacity than the Orchestrator", "Choose effort independently of the\nmodel"},
	"MA-04": {"the saved model stays `modelSupport: unconfirmed`", "never replace or rank the requested model"},
	"MA-05": {"Degraded mode", "never\nsimulate research and never block the journey", "Never use your own model identity, training memory, a static list"},
	"MA-06": {"Prefer reusing a compatible existing Profile", "never overwrite or rename one"},
	"MA-07": {"A recommendation is a proposal, not a mutation or a certification", "`validate-candidate`, preview, exact authority, apply and\nread-back"},
	"MA-08": {"API per-token price is not the marginal cost", "Cost that\ncannot be compared is `unknown`"},
	"MA-09": {"say the Orchestrator/subagent hierarchy is not\nestablished", "never claim a combination satisfies it"},
	"MA-10": {"optional help, never a precondition", "offer the `axiom-runtime` Model Advisor", "its recommendation changes nothing by itself"},
}

func TestModelAdvisorFixturesAnchoredInShippedGuidance(t *testing.T) {
	runtime, err := fs.ReadFile(skillFiles, "skills/axiom-runtime/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	project, err := fs.ReadFile(skillFiles, "skills/axiom-project/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	guidance := string(runtime) + "\n" + string(project)
	fixture, err := os.ReadFile("testdata/model-advisor/scenarios.md")
	if err != nil {
		t.Fatal(err)
	}
	for id, clauses := range modelAdvisorClauses {
		if !strings.Contains(string(fixture), "| "+id+" |") {
			t.Fatal("fixture lacks", id)
		}
		for _, clause := range clauses {
			if !strings.Contains(guidance, clause) {
				t.Fatalf("%s: guidance lacks %q", id, clause)
			}
		}
	}
	if got := regexp.MustCompile(`\| MA-\d\d \|`).FindAllString(string(fixture), -1); len(got) != len(modelAdvisorClauses) {
		t.Fatal("unanchored fixture cases", got)
	}
}

// No static model registry: shipped guidance and fixtures name no commercial
// model family and carry no price figures.
func TestModelAdvisorShipsNoStaticModelCatalog(t *testing.T) {
	commercial := regexp.MustCompile(`(?i)\b(gpt-|o[134]-|claude-(opus|sonnet|haiku)|opus|sonnet|haiku|gemini)|\$\s?\d`)
	for _, probe := range []string{"use claude-sonnet-9", "gpt-9-mini", "costs $3"} {
		if !commercial.MatchString(probe) {
			t.Fatal("guard cannot detect", probe)
		}
	}
	fixture, _ := os.ReadFile("testdata/model-advisor/scenarios.md")
	for _, name := range skillNames {
		wire, err := fs.ReadFile(skillFiles, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		if m := commercial.FindString(string(wire)); m != "" {
			t.Fatalf("%s names %q", name, m)
		}
	}
	if m := commercial.FindString(string(fixture)); m != "" {
		t.Fatalf("fixture names %q", m)
	}
}
