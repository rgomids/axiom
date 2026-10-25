package cli

import (
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

// Independent public observations and native-failure-shaped negative answers.
// This qualifies the oracle, not native compliance or a renderer.
func TestPresentationAttempt16FidelityNegatives(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	for _, tc := range []struct {
		name, good   string
		must, forbid []string
	}{
		{"C01-C02", "Órbita (orbita); ID canônico mantido internamente. Nome apenas: Órbita. Slug apenas: orbita.", []string{"Órbita (orbita)", "Nome apenas: Órbita", "Slug apenas: orbita"}, []string{id}},
		{"C09", "Agent r9 permite codex/claude; profileRef: policy-default. Project r9 declara codex-test. Seleção, resolução e execução: NOT_OBSERVED / NOT_ESTABLISHED.", []string{"Agent r9 permite codex/claude", "profileRef: policy-default", "Project r9 declara codex-test", "NOT_OBSERVED / NOT_ESTABLISHED"}, []string{"nenhum Runtime usado", "Codex não está em uso"}},
		{"C11", "status: success; estrutura válida; execution: blocked. capability_mapping_missing; runtime_policy_unavailable / policy_unconfigured. Sem dispatch nesta invocação; histórico desconhecido. Inspecionar owner/configuração antes de preview/alteração autorizada.", []string{"status: success", "estrutura válida", "execution: blocked", "capability_mapping_missing", "runtime_policy_unavailable / policy_unconfigured", "nesta invocação", "histórico desconhecido", "Inspecionar owner/configuração antes de preview"}, []string{"nunca executou", "pronto para executar"}},
		{"C15", "project/public-dag r1; digest " + digest + "; não selecionado. analysis: intake → [design, security] → implementation; design/security independentes; implementation depende de ambos. Gates vazios; fallback textual; renderer indisponível.", []string{"project/public-dag r1", digest, "não selecionado", "intake → [design, security] → implementation", "design/security independentes", "implementation depende de ambos", "Gates vazios", "renderer indisponível"}, []string{"renderizado com sucesso"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := presentation012Context{must: tc.must, forbid: tc.forbid}
			context.event.Status = completion.Success
			context.event.Provenance.SourceState = "clean"
			answer := "(success) " + tc.good
			if err := checkPresentation012(context, answer); err != nil {
				t.Fatal(err)
			}
			for _, fact := range tc.must {
				if checkPresentation012(context, strings.ReplaceAll(answer, fact, "[omitted]")) == nil {
					t.Fatalf("accepted omission: %s", fact)
				}
			}
			for _, bad := range tc.forbid {
				if checkPresentation012(context, answer+" "+bad) == nil {
					t.Fatalf("accepted invention: %s", bad)
				}
			}
			if tc.name == "C15" && checkPresentation012(context, strings.ReplaceAll(answer, digest, "01234567…")) == nil {
				t.Fatal("accepted truncated operational digest")
			}
		})
	}
}
