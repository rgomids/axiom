package cli

import (
	"strings"
	"testing"

	"github.com/rgomids/axiom/internal/completion"
)

// Independent synthetic answers and negative controls; no native/visual claim.
func TestPresentationAttempt15SourceIsolation(t *testing.T) {
	cases := []struct {
		name, answer string
		must, forbid []string
	}{
		{"C06", "Unicidade NOT_ESTABLISHED. Inventário parcial; ID desconhecido; sem alvo acionável.", []string{"NOT_ESTABLISHED", "Inventário parcial", "ID desconhecido", "sem alvo acionável"}, []string{"não é único", "alvo confirmado"}},
		{"C07-C08", "Fonte installed-r1: intake → implementation. validate-result; conclusão exige existência e correlação do output. Fonte replay-r7: content-check indisponível; timeout 30s; efeitos desconhecidos. Gate não observado.", []string{"installed-r1", "validate-result", "existência e correlação", "replay-r7", "content-check indisponível", "timeout 30s", "efeitos desconhecidos", "Gate não observado"}, []string{"gate pendente", "validação falhou", "installed-r1 usa content-check"}},
		{"C09", "Agent r9 permite codex/claude; profileRef: policy-default. Project r9 declara codex-test. Seleção, resolução e execução não observadas.", []string{"Agent r9", "permite codex/claude", "profileRef: policy-default", "Project r9 declara codex-test", "não observadas"}, []string{"Codex não está em uso", "Codex selecionado"}},
		{"C10", "Somente fonte C10: Plan r18 resolve codex/codex-test; Profile r4 fornece modelo sample-model e effort runtime-default. Login observado em auth-a. Project policy NOT_OBSERVED; attempt, compatibility, billing, acceptance NOT_OBSERVED.", []string{"Plan r18", "Profile r4", "Login observado em auth-a", "Project policy NOT_OBSERVED", "attempt, compatibility, billing, acceptance NOT_OBSERVED"}, []string{"Project declara codex-test", "compatibility FALSE", "pronto para executar"}},
		{"C11", "Comando success/0; execution blocked: capability_mapping_missing; runtime_policy_unavailable / policy_unconfigured. Sem dispatch nesta observação; histórico desconhecido. Próximo: inspecionar owner/configuração; alteração posterior requer preview/authority.", []string{"success/0", "execution blocked", "capability_mapping_missing", "runtime_policy_unavailable / policy_unconfigured", "Sem dispatch nesta observação", "histórico desconhecido", "inspecionar owner/configuração", "preview/authority"}, []string{"aprovar Runtime", "declare policy agora", "nunca executou"}},
		{"C15-C16", "Fonte r1: intake → specification → plan → implementation → review → delivery; gate humano before delivery; intake sem gate declarado. DAG: a e b independentes; c depende de a,b. Órbita (orbita); bloqueado; inspecionar configuração antes de preview. Mesmo input natural/skill. Renderer não observado; fallback textual; GUI NOT_RUN.", []string{"intake → specification → plan → implementation → review → delivery", "gate humano before delivery", "intake sem gate declarado", "a e b independentes", "c depende de a,b", "Órbita (orbita)", "bloqueado", "inspecionar configuração antes de preview", "Mesmo input natural/skill", "fallback textual", "GUI NOT_RUN"}, []string{"gráfico verificado", "Runtime aprovado"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			context := presentation012Context{must: c.must, forbid: c.forbid}
			context.event.Status = completion.Success
			context.event.Provenance.SourceState = "clean"
			answer := "(success) " + c.answer
			if err := checkPresentation012(context, answer); err != nil {
				t.Fatal(err)
			}
			for _, fact := range c.must {
				if checkPresentation012(context, strings.ReplaceAll(answer, fact, "[omitted]")) == nil {
					t.Fatalf("accepted omitted fact %q", fact)
				}
			}
			for _, invention := range c.forbid {
				if checkPresentation012(context, answer+" "+invention) == nil {
					t.Fatalf("accepted invention %q", invention)
				}
			}
		})
	}
}
