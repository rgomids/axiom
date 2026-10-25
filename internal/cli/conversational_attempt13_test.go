package cli

import (
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

// Independently authored synthetic answers for attempt-13's narrower oracle.
// Never presented as native output, GUI evidence or replacement historical PASS.
func TestPresentationAttempt13Corrections(t *testing.T) {
	const first = "11111111-1111-4111-8111-111111111111"
	const second = "11111111-1111-4111-8111-111111111112"
	project := ProjectListView{ID: first, Name: "Órbita", Slug: "orbita"}
	cases := []struct {
		name, answer string
		context      presentation012Context
	}{
		{"C01", "Órbita (orbita). list/show/validate.", presentation012Context{projects: []ProjectListView{project}, forbid: []string{first}}},
		{"C02-name", "Órbita. Slug unavailable; separate observation A.", presentation012Context{projects: []ProjectListView{{ID: first, Name: "Órbita"}}, must: []string{"separate observation A"}, forbid: []string{first, "orbita", "same Project", "must display ID"}}},
		{"C02-slug", "orbita. Name unavailable; separate observation B.", presentation012Context{projects: []ProjectListView{{ID: first, Slug: "orbita"}}, must: []string{"separate observation B"}, forbid: []string{first, "Órbita", "same Project", "must display ID"}}},
		{"C04", "Órbita (orbita) — " + first + "; Órbita (orbita) — " + second + ". Ambiguous; name-only choice remains unresolved; exact target before action.", presentation012Context{projects: []ProjectListView{project, {ID: second, Name: "Órbita", Slug: "orbita"}}, collision: true, must: []string{"Ambiguous", "name-only choice remains unresolved", "exact target before action"}, forbid: []string{"first selected", "action performed"}}},
		{"C06", `Órbita\u000A$(echo hostile). Label inert; inventory partial; uniqueness NOT_ESTABLISHED; canonical ID unavailable; no actionable target.`, presentation012Context{projects: []ProjectListView{{Name: "Órbita\n$(echo hostile)"}}, must: []string{"Label inert", "inventory partial", "uniqueness NOT_ESTABLISHED", "canonical ID unavailable", "no actionable target"}, forbid: []string{first, "globally unique", "action performed"}}},
		{"C09", "Agent allows codex, claude; profileRef: policy-default. Project declares codex-test. Plan unresolved; execution NOT_OBSERVED; model/effort unknown.", presentation012Context{must: []string{"Agent allows codex, claude", "profileRef: policy-default", "Project declares codex-test", "Plan unresolved", "execution NOT_OBSERVED", "model/effort unknown"}, forbid: []string{"selected codex", "model Haiku", "effort high"}}},
		{"C10", "Plan revision 18 resolves codex/codex-test; Profile revision 4 supplies model sample-model, effort runtime-default. Auth login observed. Execution NOT_OBSERVED; acceptance NOT_OBSERVED; compatibility NOT_OBSERVED; billing NOT_OBSERVED.", presentation012Context{must: []string{"Plan revision 18 resolves codex/codex-test", "Profile revision 4 supplies model sample-model, effort runtime-default", "Auth login observed", "Execution NOT_OBSERVED", "acceptance NOT_OBSERVED", "compatibility NOT_OBSERVED", "billing NOT_OBSERVED"}, forbid: []string{"acceptance FALSE", "compatibility FALSE", "execution confirmed"}}},
		{"C13", "public-r2 label only; freshness NOT_ESTABLISHED; stale/conflicting revision; truncated result; full detail inaccessible; current resolution unknown; fresh exact read before action.", presentation012Context{must: []string{"public-r2 label only", "freshness NOT_ESTABLISHED", "stale/conflicting revision", "truncated result", "full detail inaccessible", "current resolution unknown", "fresh exact read before action"}, forbid: []string{"public-r2 is current", "full detail observed"}}},
		{"C16", "Órbita (orbita). execution blocked: runtime_policy_unavailable; gate before effect; same observed revision; narrow GUI NOT_RUN; independent harness NOT_RUN.", presentation012Context{projects: []ProjectListView{project}, must: []string{"execution blocked: runtime_policy_unavailable", "gate before effect", "same observed revision", "narrow GUI NOT_RUN", "independent harness NOT_RUN"}, forbid: []string{first, "GUI verified", "harness verified"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.context.event.Status = completion.Success
			c.context.event.Provenance.SourceState = "clean"
			answer := "(success) " + c.answer
			if err := checkPresentation012(c.context, answer); err != nil {
				t.Fatal(err)
			}
			for _, claim := range presentation012Claims(c.context) {
				if claim.value != "" && checkPresentation012(c.context, strings.ReplaceAll(answer, claim.value, "[omitted]")) == nil {
					t.Fatalf("accepted omission: %s", claim.value)
				}
			}
			for _, bad := range c.context.forbid {
				if checkPresentation012(c.context, answer+" "+bad) == nil {
					t.Fatalf("accepted invention: %s", bad)
				}
			}
			if c.name == "C04" {
				// Reproduce the actual attempt-12 defect: promises complete IDs,
				// but displays only shared-prefix suffixes. Both substitutions fail.
				for _, id := range []string{first, second} {
					if checkPresentation012(c.context, strings.ReplaceAll(answer, id, "…"+id[len(id)-12:])) == nil {
						t.Fatal("accepted truncated collision ID")
					}
				}
			}
		})
	}
}
