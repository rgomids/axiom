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
