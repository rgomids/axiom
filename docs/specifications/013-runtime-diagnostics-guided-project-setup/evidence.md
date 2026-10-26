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
