# Evidence 013

Classificação: **synthetic** (fixtures, executáveis sintéticos que falham se
invocados, roots isolados). Nenhuma credencial, variável de ambiente, arquivo
de autenticação, processo de fornecedor ou inferência foi usado ou alterado.
Baseline `c00156f`; Go do host do mantenedor (darwin).

## Antes/depois — comando original `govsm`

`axiom --json runtime binding create --runtime codex --profile codex-default
--model gpt-6-luna --effort-mode explicit --effort low`, binários construídos
de `c00156f` (antes) e desta branch (depois), roots vazios:

| | Antes | Depois |
|---|---|---|
| status / category | `validation_failure` / `candidate_invalid` | inalterados |
| `next` | ausente | `Add --capabilities <comma-list> …; --complexities stays optional. Declarative local validation only: authentication was not evaluated and is not the cause; …` |
| `diagnostics` | ausente | `[{code: field_required, field: --capabilities, rule: at least one declared capability is required, next: …}]` |
| authentication / modelSupport | `unknown` / `unconfirmed` | inalterados |

A saída humana (sem `--json`) mostra o mesmo `next` e `diagnostics`, derivada
do mesmo documento canônico.

## Checks executados

| Comando | Resultado |
|---|---|
| `go test ./... -count=1` | 34 pacotes `ok`; 1 FAIL pré-existente: `internal/executiongraph` `TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation`, reproduzido idêntico em worktree limpo de `c00156f`; pacote não depende do código alterado |
| `go vet ./...` | sem achados |
| `go test -race -count=1` em `runtimeapplication`, `runtimeprofile`, `codexruntime` e `cmd/lingo -run 'GuidedSetup\|RuntimeAuthoring\|RuntimeDeclaration'` | `ok` |
| `./scripts/validate-repository.sh .` | PASS em checkout limpo do commit (no worktree local falha só por `.claude/.DS_Store` ignorado, criado pelo Finder, fora desta mudança) |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `git diff --check` | limpo |

## Mapeamento de aceite

| Critério | Teste | Resultado |
|---|---|---|
| A1 | `TestAuthoringMissingCapabilitiesNamesFieldWithoutAuthentication`; `TestGuidedSetupGovsmFromScratchWithCodexAndClaude` (JSON + humano) | PASS |
| A2 | `TestAuthoringCapabilitiesSuppliedProducesPreviewOfUncertifiedModel`; jornada govsm | PASS |
| A3 | `TestAuthoringBindingMissingAndAbsentRuntimeAreNotModelEvidence`; jornada govsm | PASS |
| A4 | idem A2 (`gpt-6-luna` salvo, `modelSupport: unconfirmed`) | PASS |
| A5 | asserções de ausência de `OPENAI_API_KEY`/`ANTHROPIC_BASE_URL`/`CLAUDE_CODE`; executáveis sintéticos com `exit 97` nunca invocados | PASS |
| A6 | jornada govsm: 2 bindings, 2 Repositories, Provider github, contexto, política, `default-sdd` selecionado; validate: structure valid, runtime ready, work-item/execution ready | PASS |
| A7 | `TestGuidedSetupCompletesExistingIncompleteProject`: `runtime_policy_unavailable` → G-1 → read-back `confirmed`; restam `inventory_unavailable` (schema 3) e `capability_mapping_missing` (Provider none) | PASS |
| A8 | digests errados de Project/Workflow recusados sem mudar árvore; replays `stale_preview`; `binding_exists`/`profile_exists` | PASS |
| A9 | `TestGuidedSetup013UpgradesPublishedSetForCodexAndClaude`, `TestEveryPublishedSharedRevisionStaysOwned` | PASS |
| RD-08 | `TestProfileViolationsAgreeWithValidation` | PASS |

## Não verificado

- **T08 NOT_RUN:** conversa nativa real com Codex e Claude percorrendo o wizard
  (a skill é orientação para a IA operadora; testes cobrem contratos e conteúdo,
  não o comportamento conversacional do modelo).
- Autenticação real e inferência: fora de escopo por desenho; o resumo final
  as reporta como não verificadas.
- Interrupção entre etapas é coberta por preservação de efeitos confirmados e
  replays recusados, não por injeção de falha de publicação na jornada CLI
  (coberta em testes existentes de `publication_unconfirmed`).

## Revisão 2 — correções do review do PR #356 (HEAD revisado `1b1b983`)

### CR-356-01 — dependência circular no wizard

Investigação (binário desta branch, roots vazios):

- `project workflow list` sem `--project` → `validation_failure` (o parser
  exige `--project`); não há inspeção pública de definições sem Project.
- Com um Project recém-publicado sem política, `project workflow list
  --project` retorna `default-sdd` r1/r2 (`source: builtin`) e `project
  workflow show` expõe, por Agent, `capabilities`, `role`, `complexity`,
  `runtimeConstraints` e `profileRef`.
- Após `workflow select`, `project configure --policy-draft` é recusado:
  “Project Runtime policy editing requires schema 2 or 3”
  (`SelectWorkflow` eleva o schema para 4).
- Sem Workflow selecionado, `project validate` reporta
  `runtime_resolution_blocked / inventory_unavailable` mesmo com política
  válida. A revisão 1 atribuiu isso à ordem de criação, o que estava errado.

Resolução: jornada identidade → inspeção → Profiles → política → seleção →
validate, só com operações publicadas (decisão 1 do [Plan](plan.md)).
`TestGuidedSetupGovsmFromScratchWithCodexAndClaude` passou a:

- verificar `project list` vazio;
- escolher a revisão listada mais recente de `default-sdd` a partir do JSON de
  `list`;
- derivar `--capabilities read,repository-write`, `--complexities medium` e a
  preferência `engineer/medium` do JSON de `show`;
- aplicar a política com digest obsoleto recusado e sem efeitos, e read-back
  `confirmed`;
- selecionar o Workflow e terminar com structure valid, runtime ready e
  work-item/execution ready.

O import de `internal/workflowdefinition` foi removido.
`TestGuidedSetupCompletesExistingIncompleteProject` cobre metadata-only antes
da seleção, Runtime ready após a seleção, execução bloqueada só por
`capability_mapping_missing` (Provider `none`) e a recusa da G-1 após a seleção.

### CR-356-02 — `--enabled` em Profile

`inputDiagnostics` nomeia `field_invalid:--enabled`; categoria `invalid_input`
preservada. Cobertura:

- `TestAuthoringProfileEnabledNamesField` (create/edit/validate-candidate):
  zero escritas; auth/model/effort inalterados; binding continua aceitando
  `--enabled`.
- `TestProfileEnabledRefusalNamesFieldInJSONAndHuman`: parser de produção,
  saída JSON e humana (`field: --enabled`, `Remove --enabled`); árvore de
  estado inalterada.

Demais rejeições de `safeAuthoring` verificadas como inalcançáveis pelo parser
(decisão 4 do Plan).

### Checks da revisão 2

| Comando | Resultado |
|---|---|
| `go test ./... -count=1` | 34 pacotes `ok`; mesmo FAIL de `internal/executiongraph` `TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation`, reproduzido em `c00156f` neste host; o check `PR Checks` do CI passou em `1b1b983`, indicando falha específica do host local |
| `go vet ./...` | sem achados |
| `go test -race` em `runtimeapplication`, `runtimeprofile`, `codexruntime` e `cmd/lingo -run 'GuidedSetup\|ProfileEnabled\|RuntimeAuthoring\|RuntimeDeclaration'` | `ok` |
| `./scripts/validate-repository.sh .` | PASS em checkout limpo do commit |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `git diff --check` | limpo |

Pendências inalteradas: T08 (conversas nativas) NOT_RUN; DF-013-01 e
DF-013-02 registrados no Plan.

## Revisão 3 — Model Advisor (ampliação de escopo do mantenedor)

Base: `48d325b`. CI nessa base: `PR Checks` e `delivery-metadata` pass.

Implementação somente em orientação de skill (decisões 5–7 do [Plan](plan.md)):

- seção “Model Advisor” em `axiom-runtime`;
- oferta opcional na etapa 4 de `axiom-project`;
- reconciliação das linhas que vedavam recomendação sem fonte autoritativa;
- nenhuma mudança de domínio, CLI ou estado persistente.

### Testes por tipo

| Tipo | Teste | Resultado |
|---|---|---|
| Determinístico (guidance × fixtures) | `TestModelAdvisorFixturesAnchoredInShippedGuidance`: cada MA-01…MA-10 de `testdata/model-advisor/scenarios.md` exige as cláusulas da skill que o governam | PASS |
| Determinístico (guarda) | `TestModelAdvisorShipsNoStaticModelCatalog`: nenhuma família comercial de modelos nem preço nas quatro skills ou nas fixtures; autoverificação de que a guarda detecta exemplos | PASS |
| Integração CLI | MA-04 (modelo desconhecido salvo, `modelSupport: unconfirmed`), MA-06 (`binding_exists`/`profile_exists`), MA-07 (preview → autoridade exata → apply → read-back; replay `stale_preview`), MA-10 (jornada completa) pelos testes `GuidedSetup*` | PASS |
| Upgrade de skills | `TestEveryPublishedSharedRevisionStaysOwned` e `TestGuidedSetup013UpgradesPublishedSetForCodexAndClaude` com o novo manifesto `6b3bef42…` | PASS |
| Conversacional nativo | Codex/Claude seguindo o Advisor com pesquisa real | **NOT_RUN** (T13) |

### Estado dos critérios MA

MA-01 a MA-10 estão **especificados e ancorados**: fixture e cláusula verificadas,
e, para MA-04, 06, 07 e 10, também o comportamento da CLI. O comportamento
conversacional de recomendar, comparar e degradar **não está verificado** sem
conversa nativa: as fixtures definem o oráculo, não comprovam a resposta de um
modelo.

### Checks da revisão 3

| Comando | Resultado |
|---|---|
| `go test ./... -count=1` | 34 pacotes `ok`; mesmo FAIL host-específico de `internal/executiongraph` (reproduzido em `c00156f` neste host; passa no CI) |
| `go vet ./...` | sem achados |
| `go test -race` em `codexruntime`, `runtimeapplication` e `cmd/lingo -run 'GuidedSetup\|ProfileEnabled'` | `ok` |
| `./scripts/validate-repository.sh .` | PASS em checkout limpo do commit |
| `./scripts/check-sensitive-files.sh .` | PASS |
| `git diff --check` | limpo |

Nenhuma pesquisa externa, autenticação, inferência ou consumo de crédito foi
executada para esta revisão.

## MA-356-01 — avaliação comportamental do Model Advisor

Finding do review `5652131303` no HEAD `e0a05f2`: o teste de ancoragem provava
apenas a presença de texto nas skills. Esta seção registra execução real.

### Mecanismo

[`scripts/model-advisor-eval.py`](../../../scripts/model-advisor-eval.py)
(decisões 8–12 do [Plan](plan.md)). Para cada cenário:

- workspace isolado com `axiom-project` e `axiom-runtime` copiados byte a byte
  (digest no relatório);
- `context/workflow-show.json` gerado pelo binário real em root isolado
  (`default-sdd` r2; sha256 `2b1ccc10…`);
- fontes sintéticas S-A, S-B e S-C (fornecedores e modelos fictícios);
- uma invocação do agente com modelo fixado, somente leitura;
- avaliador determinístico sobre a prosa e o anexo `ADVISOR-RESULT`.

As autotestes offline do avaliador
([`test_model_advisor_eval.py`](../../../scripts/test_model_advisor_eval.py))
usam respostas escritas à mão **apenas para testar o avaliador**; não são
Evidence do Advisor.

Autorização do mantenedor nesta sessão: até 15 invocações (14 iniciais + 1
após um defeito do harness), e depois o uso de 2 delas para re-execução.
Foram usadas 12 requisições: 10 respondidas pelo Claude e 2 recusadas pelo
Codex antes da inferência.

| Runtime | CLI | Comando fixado | Modelo observado |
|---|---|---|---|
| Claude | 2.1.310 | `claude --model claude-haiku-5-5 -p … --tools Read --strict-mcp-config --setting-sources project --no-session-persistence` | `claude-haiku-5-5` em init e em todas as mensagens (exigido pelo harness) |
| Codex | 0.163.0 | `codex -m gpt-6.1-luna -a never exec -s read-only --json --ephemeral` | recusado: 400 `model_not_found` — “The 'gpt-6.1-luna' model is not supported when using Codex with a ChatGPT account.” Nenhum modelo substituto foi tentado |

### Resultados por cenário (relatórios em [`evidence-model-advisor/`](evidence-model-advisor/))

| Cenário | Entrada / fontes | Claude (skill `453ea875…`) | Claude após correção (skill `8d2e9568…`) | Codex |
|---|---|---|---|---|
| MA-01 | Codex e Claude; S-A, S-B, S-C | **FAIL** — recomendou `va-compact`/`vb-swift` como primários com alternativas, status corretos, S-C sinalizada como conflitante, custo de assinatura `unknown`; mas citou “S-A”/“S-B” em vez da URL | **PASS** | BLOCKED |
| MA-02 | economia; S-A, S-B | **PASS** — `va-compact` e `vb-swift`, sem escolher o mais capaz | não reexecutado | BLOCKED |
| MA-05 | sem pesquisa | **PASS** — modo degradado, nenhum modelo nomeado, entrada manual oferecida | não reexecutado | BLOCKED |
| MA-08 | assinatura Codex; S-A com preços de API | **FAIL** — preço de API não apresentado como custo de assinatura, custo `unknown`; citação por rótulo em vez de URL | **PASS** | BLOCKED |
| MA-09 | Orchestrator `vb-deep` e subagentes `va-compact` sem comparação | registrado FAIL por falso positivo do avaliador (“two va-models”); **PASS** no re-grade — hierarquia “not established” na prosa e no anexo | — | BLOCKED |
| MA-10a | aceitar recomendação; S-A | **PASS** — oferta do Advisor, próximo passo `validate-candidate`/preview, zero mutações | — | BLOCKED |
| MA-10b | ignorar o Advisor e informar `vz-direct` | **PASS** — intenção preservada, status `unconfirmed`/`unknown`, nenhuma recomendação concorrente | — | BLOCKED |

Defeitos encontrados e tratamento:

1. **Harness (tentativa 1, `claude-01-harness-defect.json`):** a ferramenta
   Read não lista diretórios, e o prompt não nomeava os arquivos de fontes.
   O agente entrou corretamente em modo degradado, sem simular pesquisa.
   Correção: o prompt nomeia os arquivos.
2. **Avaliador:** a regra “no apply” casava com a menção “before any apply”, e
   a família “va-models” era lida como ID de modelo. Ambas foram corrigidas
   com autotestes; o MA-09 foi reavaliado sem nova inferência.
3. **Advisor (comportamento real):** citação de fonte por rótulo local. A
   skill passou a exigir “source cited by its URL (never a local file name or
   label)” (MA-R11). MA-01 e MA-08 foram reexecutados e passaram.

### Matriz por tipo de validação

| Tipo | Escopo | Estado |
|---|---|---|
| Static guidance | Cláusulas presentes; guarda contra catálogo | PASS (Go tests) |
| Deterministic CLI | Diagnósticos, preview/autoridade/read-back, jornada `govsm` | PASS (Go tests) |
| Behavioral evaluation | Respostas reais aos cenários prioritários | Claude: PASS em 7/7 após correção (MA-02/05/10a/10b na bateria, MA-09 por re-grade, MA-01/08 por re-execução); Codex: BLOCKED |
| Native wizard journey | Conversa completa do wizard com o Advisor, comandos reais e pesquisa web | NOT_RUN (T14) |
| Vendor availability | Disponibilidade real de modelos na conta | NOT_RUN (fora de escopo; exige observação autorizada) |

### Limitações

- **Prioridade sobre jornada:** a avaliação cobre a etapa 4 com estado do
  wizard fornecido, não a jornada nativa completa nem pesquisa web real.
- **Uma amostra:** uma resposta por cenário, com um único modelo pequeno;
  não há medida de variância.
- **Instrumentação:** o anexo JSON foi pedido pelo avaliador e pode
  influenciar a resposta.
- **Cobertura do avaliador:** não verifica a fidelidade ao estado do wizard
  (DF-013-05).
- **Codex sem Evidence comportamental** (DF-013-06).
