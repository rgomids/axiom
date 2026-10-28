# Specification 014 — Operabilidade da CLI por modelos menores

## Status e autoridade

**Proposed — revisão 1, 2026-10-28.** Implementação local sob pedido explícito
do mantenedor; sem merge, release ou publicação. Aceite humano e qualificação
empírica com modelos menores permanecem pendentes e não são substituídos por
testes determinísticos. Baseline: `6ccfbfc` (v0.22.3).

## Problema

Teste com Axiom 0.22.3, Project `govsm`, Repository `core`, destino
`RGomides/GOVSM`, objetivo: Work Item para README em convenções Go.
`project validate` retornou `ready`; três tentativas de `work-item create`
falharam e nenhum preview foi confirmado.

## Evidências (reproduzidas com Provider fake, sem mutação externa)

| # | Chamada | Resultado em 0.22.3 | Causa demonstrada |
|---|---|---|---|
| 1 | `--title`, `--body` | `validation_failure` "Explicit selector input is invalid", sem nomear a flag | Flags inexistentes; o modelo não tinha o contrato do comando: `skill inspect axiom-work-item` devolve ~93 KB para 26 operações. |
| 2 | sem `desired_outcome` | `validation_failure` com `questions` | Comportamento correto; não havia código estável para ramificar. |
| 3 | `--intent` + 7 `--elaborated-section` | `failure` | **Não** é conflito `--intent`/`problem`: a mesma chamada gera preview com sucesso (`--intent` é ignorado quando `problem` é suprido). No preview, `failure` só é produzido por falha do Provider na classificação (`ClassifyDocument` → `providerFailure`) ou do estado local; o envelope JSON omitia a categoria e `provider_unauthenticated`/`provider_invalid_response` caíam no texto genérico. A falha exata do Provider no ambiente do incidente não é reproduzível sem o host original. |

Causa transversal: o envelope canônico não expunha a categoria do domínio, e o
skill exigia que o modelo inferisse o comando a partir de documentação extensa.

## Decisão (menor alteração coerente)

- **D1 — diagnóstico no campo existente `details`.** Recusas de parser de
  comandos de Work Item/workflow levam `<code>` ou `<code>:--<flag>`
  (`unknown_input`, `duplicate_input`, `missing_value`, `unsupported_syntax`,
  `invalid_value`, `conflicting_input`, `missing_required_input`). Resultados não
  bem-sucedidos de Work Item levam sua categoria. Nenhum campo, envelope, status
  ou exit code novo (ADR-0030 D3); precedente: `project_lifecycle` já usa
  `details` para código estável. O nome só é ecoado se for nome de flag bem
  formado.
- **D2 — textos específicos** para categorias de Work Item antes genéricas
  (`provider_unauthenticated`, `provider_invalid_response`, `provider_unavailable`
  não-retentável, `invalid_draft_input`, `repository_not_configured`,
  `project_not_found` etc.), sem afirmar efeito que não foi observado.
- **D3 — introspecção restrita:** `skill inspect <skill> --operation <name>`
  devolve o mesmo payload filtrado (create: ~9 KB). Sem novo registro: deriva
  das mesmas `skillOperationSpecs` e FlagSets.
- **D4 — condições das seções** de `create` em `requiredWhen` (metadado de
  descoberta; não altera validação), incluindo o mapeamento
  `--desired-outcome` ↔ `--elaborated-section desired_outcome=`.
- **D5 — skills:** os quatro skills carregam o contrato por operação antes de
  executar e corrigem apenas o input nomeado em `details`; `axiom-work-item`
  preserva identificadores confirmados, avança por leitura/preview sem pedir
  confirmação e para em códigos de Provider/configuração. Distribuição continua
  um `SKILL.md` por skill; revisão anterior registrada em `sharedSkillHistory`.

Não adotado: skill multiarquivo/referências (muda o contrato de distribuição,
receipts e instaladores); novo campo `category`/`code` no envelope (exigiria
proposta aditiva aceita sob ADR-0030); tornar `--intent` + `problem` conflito
(mudança de comportamento de domínio sem evidência de falha).

## Aceite

- AC-1 `--title` é recusado com `details=unknown_input:--title`, sem efeito.
- AC-2 Ausência de `--provider-repository` → `missing_required_input:--provider-repository`.
- AC-3 Seções faltantes → `details=draft_incomplete` e `questions`.
- AC-4 Falha de Provider → `details=provider_*` e texto/próxima ação específicos.
- AC-5 Inspeção restrita contém só `create`/`select`, sem `--title`, e é ≥5× menor.
- AC-6 Preview preserva Project/Repository/`owner/repository`; autoridade sem
  digest ou com digest divergente é recusada; create autorizado cria uma vez;
  repetição não duplica; `work-item list` lê o vínculo.
- AC-7 Skills embutidos idênticos para Codex e Claude; histórico de revisões
  mantém propriedade de instalações existentes.

Evidência automatizada: `cmd/lingo/small_model_journey_test.go`, testes
existentes de `internal/cli`, `internal/codexruntime` e `cmd/...`.

## Pendências

- Qualificação empírica com modelos menores (conclusão end-to-end, intervenções
  corretivas, chamadas inválidas, tokens, latência) comparando 0.22.3 com esta
  revisão: **NOT_RUN** — sem ambiente/autorização nesta entrega.
- Causa exata da falha 3 no host original: não observada; reexecutar com o
  binário novo exibirá o código em `details`.
- `details` em recusas de parser de comandos Project/Runtime fora dos seletores
  de Work Item/workflow não foi ampliado nesta revisão.
