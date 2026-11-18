# AXM-12 — Validação E2E da v0.24.2

17/11/2026, America/Sao_Paulo. Resultado **PARTIAL**. Um ciclo, contra a
distribuição oficial. Último build testado: v0.24.2, commit
`2088df13a38243140ff94917eeeb45e2c2364dbd`. Nenhum defeito novo de produto
confirmado. [Release](https://github.com/rgomids/axiom/releases/tag/v0.24.2) ·
[PR #401](https://github.com/rgomids/axiom/pull/401) ·
[Issue #278](https://github.com/rgomids/axiom/issues/278) ·
[AXM-12](https://linear.app/rgomids/issue/AXM-12)

Critérios da [Spec 015](spec.md) com decisão registrada no
[ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md).
Seleção e revisões imutáveis seguem o
[ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md).
A matriz do checkpoint 03 do laboratório e a
[Evidence v0.24.1](evidence-axm-12-e2e-v0241.md) serviram de referência histórica.

## Distribuição e ambiente

- macOS 27.2.1 arm64, Go 1.26.1. Binário executado em
  `/private/tmp/axiom-v0242.zKXjfLMg/bin/axiom`, versão `0.24.2`,
  revisão `2088df13a382`, `sourceState: clean`. SHA-256
  `112f953b69e36c17253f3ba0e88a6a6be8bfb356d530b848e121e622d89b640a`,
  coincidente com `release-metadata.txt` e `axiom --json docs inspect`.
- `SHA256SUMS` e pin do verificador Darwin conferidos com os digests da API
  pública da release. Seis assets conferidos com `shasum -a 256 -c SHA256SUMS`.
  No macOS, `MANIFEST.sha256` conferiu os 38 membros e o instalador oficial
  verificou e instalou o bundle. Linux recebeu verificação de hash.
- A primeira instalação recusou o alias `/tmp` com `symlink destination refused`.
  O mesmo procedimento passou usando o caminho físico `/private/tmp`.
  `docs inspect` retornou `verified`. `first-run` instalou
  `axiom-project`, `axiom-runtime`, `axiom-work-item` e `axiom-workflow`
  em raízes isoladas do Codex e do Claude, com os oito arquivos iguais aos
  digests de `skills-manifest.txt`.
- Codex CLI 0.166.0, login ChatGPT, `gpt-6.1-sol` com esforço `low`.
  Claude Code 2.1.325, login local `claude.ai`, `claude-haiku-5-5` com
  esforço `low`. Probes dos dois modelos retornaram os marcadores esperados.
  O ambiente Claude foi executado sem o override de token OAuth herdado.
  A execução usou os logins observados e os modelos fixados, sem fallback API.
- **12 invocações de CLI com inferência** neste ciclo, segundo a unidade de
  contagem da rodada anterior. Dois probes, sete processos de Stage
  (quatro paralelos e três do fluxo padrão) e três sessões iniciadoras
  (dois Codex, uma Claude), incluindo sessões com mais de um turno.
  Nenhum processo de Stage foi iniciado pelas três sessões iniciadoras.
- Mesmo Project `axiom-mvp-e2e-20261106` (`e4e2b26f-ca22-4258-96a8-fbe3d2c650ad`),
  instalado em estado novo pela CLI a partir da fonte portátil copiada.
  Clones independentes dos Repositories `library` e `cli`, com os Issues
  existentes #1 vinculados localmente por preview → aplicação exata.
  Para recuperação, foi usada uma cópia separada do registro anterior.
  Binário, estado, skills e checkouts originais do operador foram preservados.

## Matriz T01–T15

Referências completas de Execution e vínculo observável:

| Referência | Execution ID | WorkflowBinding observado em `workflow stage status` |
|---|---|---|
| P | `16f0eef1-22dc-4d7f-b09a-53907525c535` | `e2e-parallel` r8, `sha256:00c85cb2aa37a785b988b4f9c4b1a072fb4d040249331e805bb8fb50ab025e0b` |
| D | `a82f9b7d-5963-4f4c-8167-0fef75ebf82a` | `default-sdd` r2, `sha256:7b85c02f27dc8d51097695bbd61b6818055c3a09f2b6737dbd84658b33c207d0` |
| U (registro histórico, operação nova) | `8d1c1b2c-0c79-4441-b63d-00f8445f5d11` | `minimal` r2, `sha256:f61f1d484cfa26c97390f2d28795d22e8add17a1fa2faa9860db4a2180450bb7` |

| Cenário | Resultado | Execution | Evidence deste ciclo |
|---|---|---|---|
| T01 · readiness | PASS | — | `project validate`: estrutura válida, `effective: ready`, ambos os Repositories vinculados e disponíveis, Integration de Work Item pronta. |
| T02 · Runtimes reais | PASS | P, D | Probes nativos passaram. P registrou Codex `gpt-6.1-sol` e Claude `claude-haiku-5-5` com tentativas bem-sucedidas, ambos em `low`. |
| T03 · Project/Repository/Work Item | PASS no caminho de reutilização | P, D | Instalação e bindings no estado isolado. `work-item select` vinculou ambos os Issues existentes com `--authorize-local`; lista devolveu os dois links `OPEN`. Criação de Issue remota `NOT_RUN`. |
| T04 · fluxo padrão | PASS até o gate | D | `intake → specification → clarification` concluídos; `plan` aguardou `planning_authority`, sem tentativa nessa Stage. |
| T05 · paralelismo e Q→A | PASS | P | `planner` → `lib-codex` e `cli-claude` → `integrator`. Pergunta `cli-q-format-label` às 23:27:11Z (tentativa `3de52e5e-5c27-41cc-a12a-3f753edf883c`); resposta `lib-answer-format-label` às 23:27:30Z (tentativa `edeefd28-7955-41be-b063-e6fb7dc5473e`) com `replyTo` correto. |
| T06 · dois Repositories | PASS no cenário de coordenação | P | `repositoryOutcomes` separou `library/lib-result` de `cli/cli-result` e `cli/result`. O cenário não produziu integração de commits. |
| T07 · revisão imutável | PASS para troca de seleção | P, D | Seleção mudou via preview e digest exatos enquanto P estava `running`, rev. 14. Consultas em processos novos mantiveram o mesmo vínculo de P até rev. 20. D usou o novo `default-sdd` r2. Publicação de uma revisão inédita `NOT_RUN`. |
| T08 · iniciação nativa | FAIL_ENVIRONMENT / MATERIAL_DECISION_REQUIRED | — | Claude parou em três recusas do Bash antes do dispatch. Codex usou o binário correto, mas a cópia do estado ficou obsoleta quando a seleção mudou na fonte compartilhada e retornou `work_item_not_linked`. Novo teste direto com estado consistente isolou a recusa do sandbox (F-17 abaixo). |
| T09 · integração e verificações Go | EVIDENCE_GAP | P | `go vet ./...` e `go test -count=1 ./...` passaram nos clones `library` (`c05e1b9`) e `cli` (`beda8f6`). Faltou rodada operacional de integração serial com verificações em commits novos. Conflito/falha de verificação exercitados em testes sintéticos. |
| T10 · recusas e recuperação | PASS no recorte / EVIDENCE_GAP | U | Sem autoridade de inferência: `authority_required`, sem Execution. F-2/F-18 passaram. Reconciliação preservou a mesma tentativa `unknown` (`89bb7de6-0acc-4e56-9bea-e043e793c893`). Retry recusou `recovery_required`, sem tentativa adicional. Cancelamento e conflito foram exercitados em testes sintéticos, sem repetir as operações nativas. |
| T11 · rejeição e retrabalho | BLOCKED_HUMAN_AUTHORITY | P | P chegou ao gate humano de `review`. Nenhuma rejeição, aprovação ou criação de rework simulada. |
| T12 · entrega | BLOCKED_HUMAN_AUTHORITY | P | `workflow delivery packet` retornou `execution_readiness` com ambos os Repositories bloqueados por `execution_not_completed`. Nenhuma publicação autorizada pela consulta. |
| T13 · P-AC5 | EVIDENCE_GAP | P, U | Recusas, vínculo congelado e efeito desconhecido observados. Testes direcionados cobriram conflito, falha de check, cancelamento e retry. Cobertura de falhas e concorrência operacional incompleta. |
| T14 · P-AC7 | EVIDENCE_GAP | P, D | O fluxo usou uma Execution por run e consulta direta. A comparação integral entre inventário inicial e final de comandos, campos, formatos, objetos, stores e código não foi repetida. |
| T15 · decisão de aceite | BLOCKED_HUMAN_AUTHORITY | — | Parecer técnico `NO_GO`; decisão de produto pendente. |

## Retestes e classificação

| ID | Severidade / classificação | Contrato, causa, correção e reteste |
|---|---|---|
| F-2 | Menor, histórico · PASS | A recuperação anunciava um resume impossível para uma Stage cujo gate aguardaria um dependente do writer integrado. #401, commit `51d576c98666e5fe1b6237fcc8579fef131460ba`, alinhou status e preflight. Selecionado `e2e-same-repo-gate` r1 (digest de autoria `bd9ced99c78c518aa306de92c9f61879c948b950299782b1aad8a1be6278e7f1`), novo run com os três Profiles e autoridades de comandos/inferência recusou `integration_refused`, sem ID novo e sem mudança de refs. O registro histórico `735d9b2b-1324-4031-a341-8540b5ccd884`, lido pela release, orientou mover o gate para Stage posterior e começar outra Execution. Impacto verificado: nenhum caminho de replay sugerido. |
| F-18 | Diagnóstico, histórico · PASS | Erros de permissão em estado ou worktree eram genéricos ou atribuíam localização sem prova. Mesmo commit #401. `sandbox-exec` negou gravações apenas na raiz do estado de teste; run mínimo devolveu `local_permission_denied`, nomeou as duas localizações possíveis e orientou inspeção antes de repetir. Lista de Executions idêntica antes e depois. |
| F-17 | MATERIAL_DECISION_REQUIRED | `codex sandbox -P :workspace` recusou escrita em diretório de teste fora das raízes graváveis. Com o binário absoluto, estado consistente e dois links comprovados, um run mínimo sob esse sandbox retornou `local_permission_denied`; inventário de Executions idêntico. O teste direto confirma a barreira de escrita, com as sessões iniciadoras classificadas em T08. |
| F-16 | EVIDENCE_GAP conhecido | Projeções públicas conferidas até IDs e digests dos artefatos retidos. Leitura pública de conteúdo segue lacuna histórica, sem reteste dedicado. |

## Reprodução e entrega

- Instalação com `sh scripts/install.sh --version v0.24.2 --helper "$verifier"
  --helper-sha256 "$verifier_sha" --bin-dir "$scratch/bin" --receipt-dir
  "$scratch/receipt"`, com `scratch` em caminho físico. Verificação com
  `axiom version`, `axiom --json docs inspect` e `axiom --json first-run`
  com `AXIOM_CODEX_SKILLS_ROOT` e `CLAUDE_CONFIG_DIR` isolados.
- Estado isolado por `LINGO_STATE_ROOT` e `LINGO_PROJECTS_ROOT`. O procedimento
  usa `project install` com o diretório portátil como `--source` e os dois
  clones como `--repository`; `work-item select` e `project workflow select`
  com seus digests de preview e `--authorize-local`; `workflow run` com
  Project/Repository/Work Item, Profiles por Stage/Agent e `--authorize-external`.
  Read-back com `workflow stage status`, `workflow coordination list` e
  `workflow delivery packet`. Para o teste F-18, executar o run mínimo sob
  `sandbox-exec -p '(version 1) (allow default) (deny file-write* (subpath "<raiz-do-estado>"))'`.
- Testes sintéticos em `2088df1`: `go vet ./...` e
  `go test -count=1 ./cmd/lingo ./internal/stageexecution -run 'AfterGate|UnwritableState|StopsOnConflictOrFailedCheck|CancelDuringSpawn|ChangedBaseCollisionAndCancel|ReconcileNeverSignalsAReusedPID|RetryStopsAtAttemptLimit'`
  passaram. Checks da release no GitHub bem-sucedidos, inclusive publicação,
  suites nativas macOS/Linux e smoke macOS. Logs e respostas brutas ficaram
  fora do Git no diretório temporário da rodada.
- Checkout de entrega em HEAD destacado no commit da release. Única alteração
  local é este registro. Sem correção, build adicional, branch, commit ou PR
  novo. Limite de 12 invocações do ciclo atingido. Sem bug confirmado que
  justificasse ciclo de correção.

**Recomendação técnica: NO_GO para o gate de aceite.** Requer iniciação
nativa comprovada ou decisão explícita sobre F-17, integração operacional
da v0.24.2 com checks, cobertura das lacunas T13/T14 e exercício de T11/T12
com autoridade humana. Evidence nativa de fornecedor e operacional local
está registrada acima. Aceite do produto, entrega e encerramento de AXM-12
permanecem pendentes.

## Ciclo 2 — permissões dos Runtimes e build local

> Correção CORRECT-403-01: a afirmação histórica de autorização no parágrafo
> abaixo não tem fonte primária localizada. Não comprova aceite da ADR-0039;
> ver a investigação ao final. Resultados observados não suprem autoridade.

O mantenedor autorizou o padrão de permissões irrestritas para Codex e Claude
em todas as Stages de todos os Projects, registrado no
[ADR-0039](../../decisions/0039-unrestricted-runtime-stage-processes.md).
O adaptador de Stage aplica as flags dos fornecedores. Uma sessão nativa
iniciada separadamente usa as permissões selecionadas na sua abertura.

- **T08, reteste nativo na release:** `claude -p --dangerously-skip-permissions`
  e `codex -a never exec --sandbox danger-full-access` iniciaram o binário
  oficial com os dois roots isolados explícitos. As primeiras tentativas
  (`64cedf36-258f-4d30-b8bf-fcab70032108` e
  `7ae7cd76-fa5c-4867-a176-d29504fac7b7`) despacharam sem prompt, mas falharam
  com `missing_output: result` porque a instrução da fixture mandava apenas
  responder sem escrever. Com instrução alinhada ao resultado obrigatório,
  ambas completaram `minimal` r2. Claude
  `9cca05a7-36dd-413e-b165-0566d6a84078`, tentativa
  `0248a920-db93-4284-87b0-85b06424517a`, zero recusas de permissão. Codex
  `e89bbf61-37ce-4ab9-ba29-aa822225d148`.
- **Build local:** instalador de desenvolvimento com `AXIOM_BIN_DIR` e
  `AXIOM_INSTALL_STATE_ROOT` isolados. Binário
  `/private/tmp/axiom-v0242.zKXjfLMg/dev/bin/axiom`, `development`,
  `204740d0b69ca7a3733c2b97d3cf39bdd59001e0`, `clean`, SHA-256
  `962658a78bc903dd628f19376d86ad35c721b3af89fc033bbc02f7cd551c6abb`.
- **T09, PASS:** Execution `2b269dfb-7c86-407a-9947-9070e4656246`,
  `e2e-same-repo` r1,
  `sha256:1fcd9b9914c64cb6239dfa7b75fd9fd2da312c0b494dc760b2b0ac9ac4b1aa1b`.
  Codex e Claude fizeram commits `baf45bb` e `fa58e89` em worktrees
  isolados. O engine integrou `c05e1b9 → d21f0e2 → c8d07c8` e os dois
  `post-merge` passaram. `go vet ./...` e `go test -count=1 ./...`
  passaram no archive de `c8d07c8`. O leitor dependente concluiu. A execução
  está em `awaiting_gate` em `review`, resultado de `build`
  `sha256:4988340f5f43f9c3426132492cfbc053f2255b04326cffd689846ba84ec5757e`.
- **T10, cancelamento e autoridade de comandos:** a mesma composição sem
  `--authorize-commands` recusou `commands_unauthorized`. Execution
  `936d9d47-16a4-4404-a38a-3ed3ec743e56`, `minimal` r2, teve seu
  processo Claude ativo cancelado por `workflow stage cancel --authorize-local`.
  A tentativa `1f487201-51f2-45bd-9958-51c9bc2b82f1` retornou
  `group_stop_confirmed`; o PID deixou de existir.
- **T07, autoria:** `project workflow edit` publicou `e2e-same-repo` r4
  com preview e digests exatos. Após seleção de r4, a Execution do T09
  continuava em r1 com o mesmo digest.
- **T14, redução de arquivos abaixo da meta:** a contagem histórica S3.4
  registrou −25 arquivos Go de produção. Frente à mesma baseline
  `0a65d9a90e90`, tanto a release quanto o candidato têm −8, contra a
  meta de −15. O candidato tem −7.823 linhas físicas e −129 tipos
  exportados (metas de −4.000 e −40). A comparação do conjunto alterado e do
  repositório inteiro produziu os mesmos deltas. O script histórico foi
  reproduzido em scratch com a revisão do candidato e a escrita do
  relatório histórico desativada.
- **Contagem:** 12 invocações de CLI com inferência neste ciclo. Quatro
  sessões nativas e seus quatro processos de Stage na release, três
  Stages no reteste de integração e uma no cancelamento no candidato.

## Ciclo 3 — conflito e regressões no mesmo build

- **T07/T10, PASS:** `e3765d73-0a8b-4b79-bb59-b177867108fd` iniciou na
  nova `e2e-same-repo` r4, digest de Execution
  `sha256:b22e39c4c208119f02fd189a3f1b8dbeb6535a6e70869fc1302bf7aec3bdf54c`.
  Dois writers reais modificaram o mesmo arquivo. O primeiro integrou até
  `26873ec`, com verificação pós-merge aprovada. O segundo retornou
  `not_integrated` / `merge_conflict`. A ref de integração ficou em
  `26873ec`, o leitor não despachou e nenhuma tentativa foi repetida.
- **F-2/F-18, PASS:** status do registro histórico continua orientando
  nova Execution com gate em Stage posterior. A negação de escrita do
  sistema em estado isolado ainda retorna `local_permission_denied`,
  com lista de Executions idêntica antes e depois.
- **T08, PASS no build local:** as sessões nativas completaram
  `minimal` r2, Claude `2ab5b180-1e84-486f-9717-7d81561d3a48` e
  Codex `eb502bc8-be0c-4bac-9391-e861a30ccab2`. Os digests dos resultados
  retidos coincidiram com o marcador esperado `AXM12-CANDIDATE-OK`
  (Claude escreveu uma quebra de linha final). Claude registrou zero
  recusas de permissão.
- Validação do código em `204740d`: `go vet ./...`,
  `go test -count=1 ./...`, testes direcionados do launcher e
  `./scripts/validate-repository.sh .` passaram. Os processos de
  inferência real acima complementam os testes sintéticos.
- Seis invocações neste ciclo, 30 acumuladas. A execução de entrega
  `2b269dfb-7c86-407a-9947-9070e4656246` segue em `review`, e o caso
  de retrabalho `16f0eef1-22dc-4d7f-b09a-53907525c535` aguarda a
  decisão humana sobre o resultado de `build`
  `sha256:f255f88f7ed461f032197b8567846a276dddb21228e31763b636e0fb0326f48d`.

## Decisões do mantenedor após o ciclo 3

- **T11:** o mantenedor pediu conservar o resultado útil no Draft PR.
  A resposta correlacionada confere com a implementação de `FormatLabel`
  no commit `c05e1b9`: entrada vazia retorna `[]` e `  go ` retorna
  `[GO]`. O resultado fica como Evidence de colaboração. Permanece
  pendente o exercício de rejeição e retrabalho.
- **T14:** o mantenedor manteve a meta de arquivos pendente e autorizou
  enviar um Draft PR para revisão com essa limitação registrada.
- **T12:** o mantenedor autorizou abrir `review` de
  `2b269dfb-7c86-407a-9947-9070e4656246`. A aprovação foi registrada.
  A primeira chamada recusou o dispatch com `commands_unauthorized`;
  o read-back confirmou a Execution em `ready`, com zero tentativas de
  review. A retomada com a autoridade dos checks já autorizados completou
  o review com Codex, tentativa `18eb7aee-b875-49df-9bc7-9dc3ed941965`,
  e levou a Execution a `completed` (rev. 34). A consulta
  `workflow delivery packet` retornou `ready: true` para ambos os
  Repositories, sem blockers. Push/PR do Axiom foram autorizados para
  revisão do código. Efeitos nos Repositories de teste seguem pendentes de
  autorização exata. Uma invocação adicional, 31 acumuladas.


## CORRECT-403-01 — authority reconciliation

Checkpoint: 2026-11-18 UTC; outcome **PARTIAL / HUMAN_DECISION_REQUIRED**.
Initial local/remote HEAD `6e1dd7682f23ba9d60b101246c9f48a832502682`, branch
`agent/axm-12-runtime-permissions`, base/main
`2088df13a38243140ff94917eeeb45e2c2364dbd`. Fetch confirmed exact matches,
clean tree, no additional remote commits; PR #403 open/draft, not merged.
The existing prompt/dispatch is this attempt, not a second correction run.

### Bounded investigation and findings

Inspected all 74 available AXM-12 comments (pagination complete), its description,
all 3 Issue #278 comments, PR #403 discussion/review and metadata, PR #382
reviews, and versioned ADR-0039 history (`204740d`, sole introducing commit).
Graphify was queried with backend `none` for the contract map, then the actual
Spec 015, ADR-0037, ADR-0009, ADR-0039 and Evidence were read directly. The
bounded search located no primary human act establishing actor/date and all of:
Codex + Claude, writing + read-only Stages, every Project, and host access outside
worktrees (filesystem, network and available credentials). Absence in these
sources does not prove no decision exists elsewhere. No private conversation
export or inaccessible source was assumed available or copied into the repo.

- **CR-403-F2: OPEN / EVIDENCE_GAP.** ADR, commit authorship, PR text, passing
  tests and the execution dispatch are not the missing human approval.
- **CR-403-F1: OPEN / BLOCKED_BY_AUTHORITY.** Spec 015 non-goals and safety
  requirements preserve target/write/message boundaries; ADR-0037 context and
  decisions 5–6 retain real permission/target/effect checks and non-authorizing
  messages. ADR-0039's process-level exception is not an accepted amendment.
  Spec 015 and ADR-0037 were therefore not amended; Runtime code is unchanged.

Public evidence: [binding review](https://github.com/rgomids/axiom/pull/403#pullrequestreview-5949116179),
[Issue #278 checkpoints](https://github.com/rgomids/axiom/issues/278#issuecomment-6696032408),
and [PR #382 re-review](https://github.com/rgomids/axiom/pull/382#pullrequestreview-5808695738).
The latter preserves the no-write guard. Its separate D1 approval concerns only
exact-grant delivery inspection without replay; it grants no global permissions.

### Exact material decision required and proposed amendment

The maintainer must either provide the permanent primary original approval
(with actor, timestamp and exact scope), or make a new prospective decision on
this concrete proposal; neither is implied by permission to run CORRECT:

> Permit Codex and Claude Stage processes, including read-only Stages, in every
> Project to run with vendor approvals/sandbox disabled and with their OS user's
> filesystem, network and credential access outside the assigned worktree.
> Axiom target/effect/gate checks govern only actions mediated by Axiom; direct
> Runtime/subprocess effects are not confined by those checks. Where confinement
> is required, use a dedicated OS identity or external isolation. Environment
> filtering and a read-only Stage designation are not OS security boundaries.

If approved, record the dated permanent decision on ADR-0039 and explicitly
amend Spec 015's safety/target/write/message clauses and ADR-0037's preserved
permission/target/effect clauses to distinguish Axiom-mediated operations from
unconfined direct process effects. Annotate the affected ADR-0009 fragments and
reconcile the ADR index. Keep frozen invocation/target identity, single-writer
integration, cancellation, unknown-effect/no-replay rules, and all exact grants
for Axiom-mediated operations. This paragraph is a proposal, not supersession.
A narrower choice retains the accepted boundary and requires its own scoped
implementation assessment; no unilateral launcher reversal was made here.

### Verification and remaining work

Documentation-only correction; no new runtime mechanisms, dependencies or flags.
Local commands/results are recorded below and the completion receipt binds the
published commit/remote SHA and CI observation. No real Runtime/E2E/vendor call,
external product effect, merge, release or acceptance is authorized by this run.
T11/T12/T14/T15 remain pending; historical test observations above are unchanged.
Next gate: material decision/source, dependent normative reconciliation, re-review.

Validation at initial HEAD `6e1dd7682f23ba9d60b101246c9f48a832502682`
with this documentation diff applied (dirty development source):

| Command/check | Result | Limits |
| --- | --- | --- |
| `python3 scripts/check-adr-governance.py .` | PASS | Structure only; F1/F2 remain open |
| `./scripts/validate-repository.sh .` | PASS | Repository bootstrap, governance and sensitive-file checks |
| `./scripts/test-docs-offline.sh` | FAIL then PASS after link correction | Initial relative ADR link was absent from installed corpus; replaced with public review link |
| Offline qualification | PARTIAL | Linux arm64; native Runtimes and network/source denial NOT_RUN; pinned v1 installer UNVERIFIED |
| `git diff --check` | PASS | Documentation whitespace |
| Go suite / vet / real E2E | NOT_RUN | No Go changes; no real Runtime or product effects authorized |
| Gitleaks | NOT_RUN | Executable unavailable; no dependency installed |

The staged sensitive-file check and final remote identity are reported in the
completion receipt. Prior CI PASS applies only to the initial HEAD; it is not
claimed for this correction commit. Self-review against Axiom/Ponytail review
discipline found no additional blocker in this documentation delta; unresolved
F1/F2 still prevent technical acceptance of the overall PR. PR metadata was
reconciled to remove the unverified global approval/checklist claim.

## CORRECT-403-02 — proposed normative reconciliation

This session executes [dispatch 110019cf](https://linear.app/rgomids/issue/AXM-12#comment-110019cf)
and [prompt c445908d](https://linear.app/rgomids/issue/AXM-12#comment-c445908d).
The earlier [ownership checkpoint](https://linear.app/rgomids/issue/AXM-12#comment-ed6052e8)
made no changes; the operator explicitly confirmed this session owns that dispatch.
Preflight fetched origin: clean canonical worktree, branch
`agent/axm-12-runtime-permissions`, local/remote/PR HEAD
`45c597b1f0736757f550c0a558b386884065e741`, base/main
`2088df13a38243140ff94917eeeb45e2c2364dbd`, PR open/Draft, not merged.
Linear read-back verified prior receipt, prompt, dispatch and the
[prospective authority of 2026-11-17 23:35 BRT](https://linear.app/rgomids/issue/AXM-12#comment-4ce47d42).
No other execution completion was present in the retrieved comments.

- **F1: DOCUMENTED_PROPOSED / AWAITING_MATERIAL_ACCEPTANCE.** ADR-0039's
  proposed-supersession map now names Spec 015's affected target/effect clauses,
  ADR-0037 Context/Decision 5 with Decision 6 retained, and ADR-0009 decisions
  6–7/9 solely for direct process confinement. Spec 015 has an explicit pending
  amendment; reciprocal ADR annotations preserve historical text and Accepted
  statuses. The index retains Proposed. No effective supersession is claimed.
- **F2: OPEN / EVIDENCE_GAP; ORIGINAL_APPROVAL_NOT_FOUND** remains the factual
  result of CORRECT-403-01's bounded investigation, not a newly exhaustive
  search. **PROSPECTIVE_RECONCILIATION_AUTHORITY_VERIFIED** authorizes drafting
  these proposals only. No global historical or prospective material approval,
  decision.approved event or promotion of ADR-0039 was inferred.

Security review inspected `internal/runtimeadapter/stage_invocation.go` at the
initial HEAD: both vendor flags are unconditional for Codex/Claude; the
`RepositoryWrite=false` branches change CWD/arguments, not OS privileges.
`StageEnvironment` drops unlisted environment entries but retains host context;
it cannot prevent reading credential files, other directories or network access
available to that user. Documentation now states those limits and that
`danger-full-access` is not a permanent mitigation. A required boundary needs
an appropriately restricted host/user or real external sandbox; identity alone
is insufficient for shared readable resources/network. No isolation was tested
or newly supplied by this documentation change.

TECHNICAL_LOCAL decision: use pending amendments and reciprocal navigation,
rather than changing the launcher or accepted historical clauses. Rationale:
this is the smallest complete documentary proposal within the authorization;
Runtime hardening/reversal requires a separate material scope assessment.
Validation: ADR governance, repository and documentation checks plus diff review.
No new optional finding or mechanism was introduced; deferred_findings is empty.
UX/design: NOT_APPLICABLE. Project/Repository, Runtime/Model, Execution/Agent,
Provider/Transport and Evidence/conversation boundaries remain unchanged.

Documentary reconciliation is completed as a proposal; the existing launcher
remains on this unmerged PR, with material policy acceptance and implementation
integration into main still pending. The semantic conflict is documented, not
waived or resolved by structural checks. Exact mediated grants/messages/effects,
external delivery, recovery, cancellation, fencing and no replay remain required.
No additional blocker was identified in the documentary delta during scoped
Axiom/Ponytail self-review. This does not clear overall PR technical acceptance.

Validation runs on Linux aarch64, Python 3.12.3, at the initial HEAD plus this
uncommitted documentation delta. Final commit/remote SHA, CI and receipt read-back
are bound by the completion receipt on the same AXM-12; no self-referential SHA
is fabricated here.

| Command/check | Result | Limits |
| --- | --- | --- |
| `python3 scripts/check-adr-governance.py .` | PASS | 40 ADRs; structural only, no material acceptance |
| `./scripts/test-docs-offline.sh` | PASS | Linux arm64; local archive/install, 26 corpus files, link/hash/refusal checks |
| Offline qualification | PARTIAL | Network/source denial and native Runtime observation NOT_RUN; pinned v1 installer UNVERIFIED |
| `git diff --check` | PASS | Documentation whitespace |
| Go suite / vet / real E2E / T11/T12/T14/T15 | NOT_RUN | No Go changes; vendor/product tests outside this round |
| Gitleaks | NOT_RUN | Executable unavailable; not installed |

`./scripts/validate-repository.sh .`: PASS (bootstrap, ADR/automation governance,
maintainer harness and sensitive-file checks); Runtime behavioral scenarios
explicitly skipped/UNVERIFIED. `./scripts/check-sensitive-files.sh --staged .`:
PASS before commit after inspecting all six staged documentation paths/content.
Next gate: re-review AFTER completion receipt, then a separate explicit material
human policy gate. No cr-run, Runtime inference, merge, release, external product
delivery or AXM-12/#278/R-2/R-3/MVP acceptance was performed.
