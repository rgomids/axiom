# Attempt-14 — Corrected installation qualified; native acceptance PARTIAL

2026-10-25, macOS arm64, native subscription CLIs outside the maintainer checkout.
Same open Draft [PR #343](https://github.com/rgomids/axiom/pull/343), branch
`docs/axm-39-conversational-presentation`. Initial local/remote/PR HEAD all
`e67018f9c3c52081993c1f9ae8ab154c70c457bc`; clean tree. Rechecked unchanged after
native sessions, before documentary edits. All Linear comments inventoried,
`hasNextPage: false`; attempt-14 dispatch present, no prior completion/conflicting
execution recorded. Process inventory scoped other Claude processes to other
working directories; only this executor observed in this worktree.

## Authority, baseline and ownership

[Native prompt](https://linear.app/rgomids/issue/AXM-39#comment-162ef5a3),
[attempt-13 receipt](https://linear.app/rgomids/issue/AXM-39#comment-47e84e2c),
[correction Evidence](evidence-corrections-attempt13.md), [Spec r1](spec.md),
[Plan r1](plan.md), [Tasks r1](tasks.md), [C01–C16](examples.md).
Rafael's bounded 25 October approximately 00:23 BRT authorization is a **delegated
human authorization report from Maestro**, not independent human authentication.
It covers supported reversible corrected installation, pinned subscription native
sessions, necessary minimal Project configuration and real C16 qualification.
No additional credential, billing, privilege, ambient-model or provider mutation.
The explicit newer test policy overrides historical Luna/No Sol instructions.
Single executor; no spawned agents, fallback, unpinned model, retry-for-PASS,
new CLI/domain/schema/admission/Execution semantics, Spec/ADR/Plan edits or PR.
Shared-note API again returned `McpServerError: Tool get_agent_context not found`;
issue/comment API worked, no connector repair attempted.

## Installation, backups and reversible qualification Project

Previous owned pair: `0.21.3-axm39.8`, source
`e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6`, both receipts current, docs verified.
Project/Workflow/Runtime differ from current source; Work Item unchanged.
Full four-skill shared history and compatibility 3 preserved in source; no receipt
hand edits. New **private, unpublished candidate** `0.21.3-axm39.14` uses the
existing prerelease build route from attempt-08: clean exact input HEAD, official
builder `--version`, no `--development`, no release ledger spoofing or publication.
Numeric prerelease identifier 14 follows installed 8; source identity independently
identifies corrected content. It does not claim a released stable version or tag.

```sh
./scripts/build-release-archives.sh --version 0.21.3-axm39.14 --format 2 --only macos-arm64 --output <private-candidate-root>
./scripts/install-release.sh --verify-only --helper <separately-built-verifier> --helper-sha256 <verified-helper-sha> --archive <candidate-archive> --checksums <SHA256SUMS> --bin-dir <owned-bin-root> --receipt-dir <owned-receipt-root>
<candidate-binary> --json upgrade --archive <candidate-archive> --checksums <SHA256SUMS> --bin-dir <owned-bin-root> --receipt-dir <owned-receipt-root>
# Same exact reviewed inputs, plus --preview-digest fbde6c88dadf15e24094eb2ebdf3d141d6cf65689b2e44cee48cf2711a2f3070 --authorize-local
axiom --json runtime claude install
```

All exit 0. Archive member paths/types checked before extraction; archived four
skills equal exact source bytes. Strict pinned verifier succeeded. Fresh backup
contains **97 objects / 72 files**: binary, installation receipt, complete owned
docs generations/bindings, eight skills and both skill receipts. SHA256, UID/GID,
modes and mtimes recorded; private backup manifest
`1bfe355a1f25f28cf158ba94f68a33e1fd7dba1712051aff705d394674421ef8`.
Separate restore rehearsal reconstructed every object and independently compared
bytes and metadata; PASS. Live baseline compared again before apply; unchanged.
Backup retained privately, no credential/unmanaged content copied. Restore was
rehearsed, **no live rollback** claimed because both supported applies succeeded.
If a future apply fails, inspect exact ledger/ownership before restoring this
complete bounded set; do not downgrade, mix pairs or fabricate receipts.

Supported upgrade preview: direct/compatible, `absent_v1`, no persisted state to
transition, no leftovers; eight effects: new docs generation/binding, binary,
installation receipt, three Codex skills and Codex skill receipt. Apply confirmed
all eight effects. Supported Claude install converged its recognized owned set.
No cross-root atomicity claimed. Both status calls: compatibility 3 / skill set 3,
all four `equivalent`, receipt `current`; docs `verified / ok`. Inspector loaded
remains `unknown`, distinct from session-level evidence below.

| Artifact | SHA-256 |
| --- | --- |
| Candidate archive | `e106e7186b7e7cf6d996742c449417175b563ef15db85c81c22a170e4e4e102e` |
| Installed binary | `74dc4641b69f3e41df60f5e5762b6ea19117f0b34d9ff6b62f779bf3fb13eb5b` |
| Embedded manifest | `df2d8dcb0f0ab4808240012918d806aa8e7a1472553101bd1ecd54aea25009ce` |
| Docs inventory skill manifest | `dbbabd8fa4f3b204bbc3913e113926e47a0341ab6a6f0a89b15942aea35d74ed` |
| Project skill, both roots | `52cde467850d0f51f10a219e5e8b0ae69a74bc22b65413501d0ef628a0e63050` |
| Workflow skill, both roots | `4df6fc93426cd9532a9f2a7cc2df6d58962c59eb9b0f91f9d032c73140f640b7` |
| Runtime skill, both roots | `39b21768a263f5702afd7f637e46e333d3cc87b60400bb24371e8cf2035fc02e` |
| Work Item skill, both roots | `ea5476f17d1962c0938968fa726a006a01c4d79913ff3988e3d11aaf64af49f3` |
| Codex receipt | `40f3ca7c89ffd9249109c5d1ebb1bc19e1da9d253d833f8bc8c8a8152d5311ba` |
| Claude receipt | `d84d7710517e82ffeba4a8902c7c72126f2211adf4592f79755271445b9d6299` |

Ambient real inventory initially empty. Created exactly one **disposable local
qualification Project through actual supported CLI**, isolated by documented
`LINGO_PROJECTS_ROOT` / `LINGO_STATE_ROOT`; actual newly initialized local-only Git
repository, provider `none`, no Runtime/Profile changes, no Workflow selection.
This is actual CLI configuration for a test journey, **not a production Project**
and not a synthetic fixture promoted to acceptance. Pre-change destinations absent;
source repository and ambient Project stores untouched.

```sh
axiom --json project configure --slug axm39-native-qualification --name 'Órbita AXM-39 — qualificação local' --repository qualification=<qualification-repository> --repository-remote qualification=none --work-item-provider none
# Repeat exact inputs plus --project-id 37b7e9af-ad73-4733-9b68-a3e73cb58d15 --preview-digest 2fc7a96517beb43b699f53a8b54160991630c340ee22accd3bc19773defe6f75 --authorize-local
```

Preview reviewed within the user's explicit configuration grant: only portable
Project and local bindings, no provider/credential/billing effect. Create/read-back
exit 0; UUID `37b7e9af-ad73-4733-9b68-a3e73cb58d15`, Project revision
`0042ed6958f09bde8416c94bab021336ea1fc2cc43674f0065096f6fc50a5961`.
Real list/show/validate/Workflow show executed independently in both native sessions.
`structure: valid`, `effective: blocked`; Work Item `capability_mapping_missing`;
Execution also `runtime_policy_unavailable / policy_unconfigured`. No policy was
configured merely to manufacture success. Builtin `default-sdd` r2 inspected,
not selected; digest `1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068`.
Workflow/Stage/Agent facts came from that real installed definition, not a fixture.

After both sessions ended, supported archive preview/apply used digest
`50b5688950b211f727a4f90ab6b23b478f4f244817a548bf8c5acd55c85ed57c`, exit 0.
Default isolated list empty, include-archived confirms only this archived UUID;
ambient list still empty. Private isolated records retained for audit/reversal;
no unsupported delete or canonical-file patch. Installed corrected pair intentionally
retained; post-native all eight files still byte-identical to source.

## E07 — Model and fresh loading gates

Read-only auth observations: `codex -m gpt-6.1-sol login status` reports ChatGPT;
`claude --model claude-haiku-5-5 auth status` reports first-party OAuth logged in.
These are not inference proof. CLI versions: Codex **0.163.0**, Claude **2.1.310**.
No login/refresh/credential read, export or edit. Every invocation, help/version,
preflight and resume explicitly pinned; no Luna, Sonnet, Opus or alternate model.

Codex one native conversation, init plus six resumes, each exit 0:

```text
codex -m gpt-6.1-sol -a never exec -s read-only --json --skip-git-repo-check -C <native-scratch> 'Reply exactly OK. Do not use tools or load skills.'
codex -m gpt-6.1-sol -a never exec resume --json --skip-git-repo-check 01a1711a-cead-7712-84b2-ac998a37c40c -
```

Initial minimal prompt caused inference before metadata inspection, explicitly
not pre-inference proof. No substantive input until supported native session JSONL
`turn_context` positively identified `model: gpt-6.1-sol`, `effort: medium`,
`approval_policy: never`, sandbox `read-only`. All **seven** turn contexts match;
no drift. This is supported runtime model/routing metadata, not independent backend
attestation or a model name inferred from the answer. Init stderr exactly
`Reading additional input from stdin...` plus one newline; no error. Full argv/captures
retained privately. Thread UUID above binds init, complete reads and every case.

Explicit `$axiom-project $axiom-workflow $axiom-runtime $axiom-work-item` entry,
then four separate complete `cat`/SHA256 commands. Each exit 0; each full installed
file is an exact substring of its actual tool output, without truncation. Executor
independently compared every file to input HEAD. Model's own final answer conservatively
said native loading/source verification unknown; it did not attest discovery or
source itself. Evidence establishes actual corrected bytes read in session context,
not stronger hidden native resolver/retention certification. No separate Codex
Skill-tool acknowledgement exists; that facet remains explicitly unverified.

Claude one native stream process, seven successful turns, exit 0, empty stderr:

```text
claude --model claude-haiku-5-5 -p --input-format stream-json --output-format stream-json --verbose --no-session-persistence --setting-sources user --settings <session-only-restricted-JSON> --strict-mcp-config --mcp-config '{"mcpServers":{}}' --no-chrome --permission-mode dontAsk --tools Read,Skill,Bash --add-dir <eight-owned-skill-directories> <qualification-root>
```

Session-only settings disable hooks, allow Read only for eight exact owned skill
files plus two explicit qualification captures, Skill only for four domain names,
and Bash only Project list/show/validate and Workflow list/show; deny write/edit,
Agent and web tools. Child environment drops inherited Anthropic/Claude model and
provider overrides by name, without exporting their values; host settings unchanged.
No permission bypass. First input only `Reply exactly OK. Do not use tools or load
skills.` caused minimal inference before metadata inspection. Subsequent substantive
turns were withheld until both init and first assistant model positively matched:

```json
{"type":"system","subtype":"init","model":"claude-haiku-5-5","session_id":"82c3f947-847a-4a95-b35d-7f2e9a6716a2","claude_code_version":"2.1.310","apiKeySource":"none","permissionMode":"dontAsk"}
```

Every captured assistant model exactly `claude-haiku-5-5`; all seven results
successful, each `permission_denials: []`. Six native Skill calls (four initial,
Project/Workflow again for explicit C16), four complete Read calls and four Bash
reads. Every numbered Read reconstructed by removing line prefixes matches its
full installed file exactly. Native Skill launch events/expansions plus full reads
prove this session consumed corrected revision; receipts alone were never loading
proof. No claim every internal context byte is independently retained indefinitely.

## Per-case verdicts and scope

PASS below is only for supplied facts and observed answers. **SCOPED_REPLAY is
synthetic input answered by real native inference, never full native PASS**.
LIVE is actual disposable CLI Project/installed definition, not production or E2E.
Natural and explicit C16 used identical real facts after the same live reads.

| Case | Codex | Claude | Scope / evidence and limitation |
| --- | --- | --- | --- |
| C01 | PASS | FAIL | LIVE + SCOPED_REPLAY, turns 02/03: Codex routine label; Claude gratuitous full UUID in both |
| C02 | PASS | PASS | SCOPED_REPLAY 03: independent name-only and slug-only observations, no invented second field |
| C03 | PASS | PASS | SCOPED_REPLAY 03: unnamed fallback, complete UUID and explanation; no mutation occurred |
| C04 | PASS | PASS | SCOPED_REPLAY 03: both complete colliding UUIDs and unresolved repeated name choice; no live collision created |
| C05 | PASS | PASS | SCOPED_REPLAY 03: complete requested UUID, supported exact selector and digest |
| C06 | PASS | PARTIAL | SCOPED_REPLAY 03: both inert label/NOT_ESTABLISHED; Claude opens “Não é alvo único”, a negative uniqueness claim not established by partial inventory |
| C07 | PASS | PARTIAL | LIVE 02 + replay 04: exact real Stage/Agent fields; Claude names validator but omits explicit output-existence/correlation completion condition; summary/details progressively supplied together when requested |
| C08 | PASS | PARTIAL | SCOPED_REPLAY 04: supplied content-check does not match installed validate-result; Codex preserves source gap; Claude says unrelated unavailable validator prevents criterion and calls gate pending without observation. Not a real failure injection |
| C09 | PASS | PARTIAL | LIVE 02 + replay 04: both exact policy-default; Claude asserts “Codex não está em uso” where usage is merely unestablished |
| C10 | PARTIAL | PARTIAL | SCOPED_REPLAY 04: good plan/auth/attempt/compatibility/billing/acceptance split, but both import Project codex-test declaration from independent C09; C10 only supplied resolved Profile, live Project policy was absent |
| C11 | PASS | PARTIAL | LIVE 02 + replay 04: exact blockers/no dispatch retained; Claude next action jumps to policy declaration rather than smallest inspection, though preview/authority retained |
| C12 | PASS | PASS | SCOPED_REPLAY 04: invalid result despite exit 0, timeout, authority/result blockers and unknown effects survive; reconcile before retry |
| C13 | PASS | PASS | SCOPED_REPLAY 04: conflicting/stale references, inaccessible/truncated source; neither infers freshness from r2 |
| C14 | PASS | PASS | SCOPED_REPLAY 04: no version registry/probe/certification demand or success guarantee |
| C15 | PASS | PARTIAL | Real definition plus synthetic DAG 04: Codex full sequential/gates and DAG; Claude omits complete sequential/gate fallback. No graphical renderer observed |
| C16 | PARTIAL | PARTIAL | LIVE facts 05/06: both real native natural/explicit turns; Codex semantics equivalent; Claude routine UUID and unsupported “aprovação explícita do Runtime” remain. GUI/narrow rendered-width/independent visual-harness facets unobserved |

C16 real processes and explicit native entry observed; serialized output is only
plain text. No screenshot, observed graphical surface, width measurement or
independent visual harness was produced. Native CLI captures do not close that
facet; requested narrow formatting and textual equivalence are not visual proof.
No fabricated screenshot, HTML transcript rendering or self-scoring model verdict.
This evidence is executor self-review, not independent acceptance/review.

Prior attempt-12 Claude FAIL C01/C02/C04 and PARTIAL C06/C09/C10/C13/C16 remain
unaltered at their original SHA. New C02/C04/C13 observations supersede nothing;
they cover only this input/session. Historical Codex scoped replay/C16 PARTIAL
and attempt-13 deterministic C01–C16 PASS_SCOPED retained, never relabeled native.

## E08 — Tasks, ACs, validation and decisions

T01–T06 historical PASS unchanged. T07 **PARTIAL**, E07-Codex/E07-Claude **PARTIAL**:
corrected installation and model/byte gates observed, real bounded CLI journey
achieved, but failed/partial answers and Codex resolver/UI facets remain.
T08/E08 **PARTIAL**: new observed reconciliation, native acceptance open.
AC1 **PARTIAL** (Claude C01 failure); AC2 **PARTIAL** (C07/C08/C15/C16 limits);
AC3 **PARTIAL** (C09/C10 and loaded-provenance limits); AC4 **PARTIAL** (C11/C16
next action limitations); AC5 **PARTIAL** (both pinned native conversations real,
full matrix/visual acceptance incomplete); AC6 **PASS_SCOPED_REVIEW**: Evidence-only
repository delta, no product/architecture/design/authority contract change.
No independent Review readiness, AXM-12 E2E, human acceptance, Done, merge or release.

| Actual local check | Result |
| --- | --- |
| `go test ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./internal/install` | PASS / 0; first three cached, install 62.501s |
| `./scripts/check-go-quality.sh format` | PASS / 0 |
| `go vet ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./internal/install` | PASS / 0 |
| `go build ./...` | PASS / 0 |
| `./scripts/validate-repository.sh .` | PASS / 0 |
| Relative links / historical bytes / production unchanged | PASS / 0; 150 relative links, historical prefixes and protected source bytes preserved |
| `git diff --check` | PASS / 0 |
| `./scripts/check-sensitive-files.sh .` and `--staged .` | PASS / 0 |
| `gitleaks git --pre-commit --staged --redact --no-banner --no-color .` | PASS / 0; no leaks |
| Exact pushed-head CI | Matching final-head callback only; input-head CI is not final-head CI |

No new production code/tests, full local `go test ./...` or race suite. Installed
source remains input HEAD e67018f; final Evidence commit is not misrepresented as
installed binary source. Corrections already explicitly instruct the failed
behavior; native violations alone do not establish a new deterministic code defect.
No prompt/skill patch or repeated inference performed to seek passing results.

technical_decisions[] (TECHNICAL_LOCAL, decision_maker executor-ai, scope_impact none):

- **IMPL14-TD-01**: official clean private prerelease candidate, coherent binary/docs/
  four-skill identity and supported digest-bound owned upgrade; alternative manual
  skill/receipt replacement or equivalent-version forcing rejected. Evidence:
  builder, upgrade preview/ledger, both receipts and hash table. Rollback: verified
  complete backup/isolated restoration; no live restore needed.
- **IMPL14-TD-02**: minimal OK gate before substantive native input; explicit exact
  model pins on every command/resume, supported runtime JSON metadata; alternative
  flag-only/model-style inference rejected. Evidence: seven Codex contexts, Claude
  init/assistant models. Reversible process flags only, no ambient configuration.
- **IMPL14-TD-03**: isolated real CLI qualification Project/provider none, no policy
  or Workflow mutation to manufacture readiness; alternative production Project
  writes and fictitious live fixture rejected. Evidence: CREATE preview/read-back,
  actual native reads and supported archive cleanup. Initial roots absent, ambient
  inventory unchanged; isolated archived record retained for audit/reversal.
- **IMPL14-TD-04**: bounded native turns with actual installed bytes and classified
  public test inputs; retain failures and UI/resolver gaps instead of oracle-based
  native/visual PASS or PASS-seeking retries. Evidence: per-case ledger and captured
  answers below; no new product requirement, independent reviewer or renderer.

DF-01 **DEFERRED_NOT_IMPLEMENTED** remains the historical predecessor index-summary
finding from [Specify Evidence](evidence-specify.md). No new deferred requirement;
GitHub #346 untouched. Failed native cases remain required in-scope work, not
optional deferred findings. Remaining: correct/reconcile observed answer failures
within accepted guidance, qualify C10 independent-source fidelity and missing
C16 narrow graphical/independent harness facet, then reconcile T07/T08/AC/E07/E08
and final-head checks. Next gate **RECONCILE_IMPLEMENT / AWAITING_EXECUTION_REPORT**.
One same-AXM stage.partial event with final SHA/CI and API read-back; no speculative
successor dispatch or claimed callback integration beyond this real receipt.

## Sanitized inputs and actual visible answers

Only classified public test prompts and visible answers follow; not raw private
chat/provider export. Private paths replaced by logical placeholders. Native
metadata/tool events and exact unredacted argv remain private. No output rewritten
to correct failures. Turn 01 loading and minimal turn 00 are documented above.

### Turn 02 — Real local Project and installed Workflow

Input (both runtimes; explicit entry uses native Skill calls in Claude):

```text
Liste meus Projects; mostre e valide o Project de qualificação local 37b7e9af-ad73-4733-9b68-a3e73cb58d15. Use CLI real read-only project list, project show --selector, project validate --project. Depois explique intake e implementation e detalhes exatos dos Agents no Workflow builtin default-sdd r2, digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068, via axiom --json project workflow show --project 37b7e9af-ad73-4733-9b68-a3e73cb58d15 --source builtin --workflow default-sdd --revision 2 --digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068. Não selecione Workflow nem configure policy. Distinga operação concluída, estrutura, blockers por operação e ausência de execução; Runtime/model/profile selecionado/resolvido/observado desconhecido quando ausente. Contexto: Project local real descartável criado pelo CLI em roots isolados para qualificação; não é Project de produção, nenhum Stage dispatch autorizado. Preserve Unicode e não exponha paths privados. Responda naturalmente; não avalie os próprios testes. C01/C07/C09/C11 live subset; casos de replay vêm depois.
```

Codex observed answer:

```text
Vou usar axiom-project e axiom-workflow já lidos: listar, mostrar, validar e inspecionar a definição exata. Sem seleção, configuração ou dispatch.


Encontrado um Project: **Órbita AXM-39 — qualificação local (axm39-native-qualification)**, ativo. Associação `qualification` disponível, vinculada e `local_only`; nenhuma integração desabilitada. Workflow ativo: `none`.

As quatro operações CLI concluíram com `success`, exit 0. Isso não significa prontidão para execução:

| Validação | Resultado |
|---|---|
| Estrutura | `valid` |
| Readiness efetiva | `blocked` |
| `work-item` | `capability_mapping_missing`, sujeito `work-item` |
| `execution` | Mesmo blocker, mais `runtime_policy_unavailable / policy_unconfigured` |

Avisos: `business_context_absent` e `technology_context_absent`. Nenhuma autoridade concedida pela validação.

Definição inspecionada: **Default bounded SDD**, schema 2, `source: builtin`, `workflowId: default-sdd`, revisão `2`, digest `1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068`. Ela **não está selecionada**. Project `37b7e9af-ad73-4733-9b68-a3e73cb58d15`, revisão `0042ed6958f09bde8416c94bab021336ea1fc2cc43674f0065096f6fc50a5961`.

Renderização gráfica não qualificada nesta superfície; segue representação textual completa. Todos os Stages são `sequential`, concorrência `1`, com um Agent `owner`, sem predecessores no próprio Stage:

| Ordem | Stage e propósito | Gate humano declarado |
|---|---|---|
| 1 | `intake`: normalizar intenção delimitada e alvo exato | Nenhum |
| 2 | `specification`: especificar resultado, requisitos, aceite e não objetivos | Nenhum |
| 3 | `clarification`: resolver dúvidas materiais e registrar decisões | Nenhum |
| 4 | `plan`: planejar arquitetura, escopo, validadores, riscos e rollback | `planning_authority`, `before` |
| 5 | `tasks`: decompor plano aprovado em unidades delimitadas | Nenhum |
| 6 | `implementation`: implementar plano aprovado, preservando mudanças do usuário | `implementation_authority`, `before` |
| 7 | `review`: revisar comportamento, arquitetura, segurança e validação | `review_started`, `before` |
| 8 | `evidence`: registrar evidência reproduzível e limitações | Nenhum |
| 9 | `reconciliation`: reconciliar contratos, documentação e achados | Nenhum |
| 10 | `completion`: montar pacote final e solicitar aceite | `human_acceptance`, `after` |

Essa ordem representa progressão declarada; satisfação dos gates não observada.

**Intake** recebe `source`, tipo `work-item`, origem `target`, obrigatório. **Implementation** recebe `source`, tipo `stage-output`, origem `tasks/result`, obrigatório. Ambos têm checkpoint e produzem `result`, tipo `artifact`, obrigatório. Critério `valid-result`: saída existente, correlacionada e aprovada pelo validador registrado `validate-result` (`artifact-schema`, política `builtin-sdd-v1`). Falha bloqueia; resultado desconhecido exige reconciliação; retry é explícito.

Detalhes exatos dos Agents desses dois Stages:

| Campo | `intake/owner` | `implementation/owner` |
|---|---|---|
| `role` / `complexity` | `engineer` / `medium` | `engineer` / `medium` |
| `dependsOn` | `[]` — nenhum | `[]` — nenhum |
| `inputs` / `outputs` | `["source"]` / `["result"]` | `["source"]` / `["result"]` |
| `capabilities` | `["read"]` | `["read","repository-write"]` |
| `runtimeConstraints` | `["codex","claude"]` | `["codex","claude"]` |
| `profileRef` | `policy-default` | `policy-default` |
| `timeoutSeconds` / `maximumAttempts` | `1800` / `2` | `1800` / `2` |
| `effectCeilings` | `["read","process","artifact-publish"]` | `["read","repository-write","process","artifact-publish"]` |
| `integrationOwner` / `validationOwner` | `true` / `true` | `true` / `true` |

Responsabilidade comum: produzir saídas declaradas, validar referências e permanecer no envelope aprovado. Role descreve responsabilidade; Agent identifica seu proprietário configurado. Os demais Stages declaram os mesmos campos de Agent de `intake`, cada qual no próprio escopo.

`codex` e `claude` são restrições declaradas; `policy-default` é intenção de resolução. Runtime, model e Profile concretos **selecionados, resolvidos ou observados: desconhecidos**. Schema 2 não declara esforço no Workflow. Disponibilidade dos validadores e resolução dos digests de contexto continuam como pré-requisitos anteriores à execução.

Nenhuma execução iniciada ou Stage despachado por estas chamadas; existência de execuções anteriores não foi consultada. Nenhuma seleção ou policy alterada. Próxima decisão permanece com você; nenhuma correção automática executada.
```

Claude observed answer:

```text
**Resultado:** o CLI retornou sucesso nas quatro leituras. A estrutura do Project é válida, mas o Project está **bloqueado** para execução. Nada foi executado nem selecionado.

**Projects (`project list`)**: 1 Project.
- Órbita AXM-39 — qualificação local (axm39-native-qualification), ID `37b7e9af-ad73-4733-9b68-a3e73cb58d15`, status ativo, `activeWorkflow: none`.

**Show (`project show --selector`)**: sucesso. Repositório `qualification` com disponibilidade `available`. Integrações desabilitadas: nenhuma.

**Validate (`project validate --project`)**: operação concluída com sucesso, por escopo:
- **Estrutura:** `valid`.
- **Efetivo:** `blocked`.
- **Repositório:** `qualification` vinculado, disponível, `local_only`.
- **Work Item (`work-item`):** `blocked`, motivo `capability_mapping_missing`. A capacidade `work-item` está `missing`.
- **Execution (`execution`):** `blocked`, motivos `capability_mapping_missing` e `runtime_policy_unavailable` (`policy_unconfigured`).
- **Warnings:** `business_context_absent`, `technology_context_absent`.
- **Runtime/model/profile:** política portátil não configurada. Selecionado, resolvido e observado: **desconhecido**. Não há Runtime, model ou Profile declarados.

**Workflow `default-sdd` r2 (`project workflow show`)**: inspeção concluída. Digest confirmado `1a7d6f56…4068`. Não há seleção: `activeWorkflow` do Project é `none`. Pré-requisitos declarados: checagem de Runtime/Profile/effort, de validadores e de digests de contexto antes da execução.

**intake**
- Propósito: normalizar a intenção limitada e o alvo exato do Work Item.
- Sequencial, concorrência 1, checkpoint `true`, sem gate humano.
- Entrada: `source` (work-item, alvo, obrigatória). Saída: `result` (artifact, obrigatória).
- Critério `valid-result` com validador `validate-result` (`artifact-schema`, política `builtin-sdd-v1`).
- Falha: `block`; desconhecido: `reconcile`; retry: `explicit`.
- Agent `owner`: papel `engineer`, complexidade `medium`, dependências nenhuma, capacidades `read`, Runtime constraints `codex` e `claude`, `profileRef: policy-default`, timeout 1800 s, máximo 2 tentativas, effect ceilings `read`, `process`, `artifact-publish`, `integrationOwner` e `validationOwner` `true`. Não há effort declarado no schema 2.

**implementation**
- Propósito: implementar apenas o plano aprovado e preservar mudanças do usuário.
- Sequencial, concorrência 1, checkpoint `true`.
- **Gate humano:** `implementation_authority`, timing `before`.
- Entrada: `source` (stage-output de `tasks/result`, obrigatória). Saída: `result` (artifact, obrigatória).
- Validador: `validate-result` (`artifact-schema`, `builtin-sdd-v1`). Falha `block`; desconhecido `reconcile`; retry `explicit`.
- Agent `owner`: papel `engineer`, complexidade `medium`, dependências nenhuma, capacidades `read` e `repository-write`, Runtime constraints `codex` e `claude`, `profileRef: policy-default`, timeout 1800 s, máximo 2 tentativas, effect ceilings `read`, `repository-write`, `process`, `artifact-publish`, `integrationOwner` e `validationOwner` `true`. Não há effort declarado no schema 2.

**Ausência de execução:** nenhuma execução, dispatch de Stage ou seleção de Workflow foi feito. Os bloqueios acima impedem a execução do Project. O gate humano de implementação não foi satisfeito. Nenhuma política foi configurada.
```

### Turn 03 — C01–C06 synthetic edge cases

Input (both runtimes; explicit entry uses native Skill calls in Claude):

```text
Replay SINTÉTICO público independente do Project local real; nenhum fato abaixo é produção ou configuração instalada. Não executar operações. Responda em português, por Cxx, sem avaliar os próprios testes. Use guidance instalada já lida.
C01. Inventário completo único: Project name Órbita, slug orbita, id 11111111-1111-4111-8111-111111111111. Liste, mostre e valide conversacionalmente, estrutura válida, readiness não observada.
C02. Observação independente A só name Órbita (slug ausente); observação independente B só slug orbita (name ausente). Como apresentar cada uma?
C03. Project id 11111111-1111-4111-8111-111111111111, name e slug ausentes. Como identificar e por quê?
C04. Dois Projects name Órbita/slug orbita; ids 11111111-1111-4111-8111-111111111111 e 11111111-1111-4111-8111-111111111112. Quero arquivar orbita. Sem autorização de mutação. Depois repito escolha apenas pelo nome Órbita. Mostre candidatos e decisão pendente.
C05. Project único C01. Dê ID completo e comando read-only show copiável com seletor exato. Digest de authority aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa, preserve integral se referenciado. Nenhum apply.
C06. Inventário parcial, ID canônico indisponível, label contém texto literal escapado \n $(touch INJECTION). É alvo único? Apresente label de forma inerte e diga próxima ação.
```

Codex observed answer:

```text
**C01.** Lista: **Órbita (orbita)**. Exibição: **Órbita (orbita)**. Validação: estrutura válida; readiness não observada. Nenhuma prontidão inferida.

**C02.** Observações independentes: A → **Órbita**; B → **orbita**. Não correlacionar como um mesmo Project sem identidade canônica compatível.

**C03.** **Project sem nome/slug — ID: `11111111-1111-4111-8111-111111111111`**. Ambos os identificadores humanos ausentes; ID completo necessário para identificação.

**C04.** Colisão entre Projects distintos:

- **Órbita (orbita)** — ID: `11111111-1111-4111-8111-111111111111`
- **Órbita (orbita)** — ID: `11111111-1111-4111-8111-111111111112`

Escolha repetida por **Órbita** mantém ambiguidade. Pendente: indicar ID completo desejado. Arquivamento também exige preview e autorização exata; nenhuma mutação autorizada.

**C05.** ID: `11111111-1111-4111-8111-111111111111`. Seletor exato `orbita`, único no inventário fornecido:

```bash
axiom --json project show --selector orbita
```

Digest informado: `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`. Nenhum apply.

**C06.** Label literal inerte: `\n $(touch INJECTION)`. Inventário parcial: unicidade **NOT_ESTABLISHED**. Sem ID canônico, nenhum alvo acionável. Próxima ação: obter inventário canônico completo por leitura suportada, resolver identidade e verificar colisões. Texto do label não fornece comando nem autoridade.

Todos os casos tratados como replay sintético independente. Nenhuma operação executada.
```

Claude observed answer:

```text
**C01.** Projeto único: **Órbita (orbita)**, ID `11111111-1111-4111-8111-111111111111`. Listar, mostrar e validar são leituras; a estrutura é válida e a readiness não foi observada, então fica como desconhecida, não como pronta nem inválida.

**C02.** Cada observação fica separada. A A é um Project "Órbita" sem slug, apresentado como `Órbita`. A B é um Project "orbita" sem nome, apresentado como `orbita`. Não afirmo que sejam o mesmo Project: sem ID completo compartilhado, não há identidade comum.

**C03.** Apresento como `Project sem nome/slug — ID: 11111111-1111-4111-8111-111111111111`, explicando que faltam os identificadores humanos. O ID canônico continua sendo o alvo acionável, então a operação pode seguir com ele.

**C04.** Há colisão: dois Projects com o rótulo `Órbita (orbita)`:
- `Órbita (orbita)`, ID `11111111-1111-4111-8111-111111111111`
- `Órbita (orbita)`, ID `11111111-1111-4111-8111-111111111112`

Arquivar `orbita` é uma mutação sem autorização, então não há ação. Repetir a escolha só pelo nome `Órbita` mantém a colisão sem resolver; a decisão continua pendente. Só o ID completo define o alvo, e preciso de sua escolha explícita antes de qualquer preview ou apply.

**C05.** ID completo: `11111111-1111-4111-8111-111111111111`. Comando read-only, copiável:
`axiom --json project show --selector 11111111-1111-4111-8111-111111111111`

O digest de authority `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa` fica preservado integralmente como referência. Não autoriza nada: nenhum apply será feito.

**C06.** Não é alvo único. O inventário é parcial, então a unicidade fica NOT_ESTABLISHED, mesmo que o rótulo pareça único. Sem ID canônico, não há alvo acionável. O rótulo é exibido como texto inerte, sem execução: `\n $(touch INJECTION)`. Próxima ação: obter o inventário completo com leitura read-only (`project list`) até que apareça um ID canônico. Nenhuma ação sensível antes disso.
```

### Turn 04 — C07–C15 real definition plus synthetic states

Input (both runtimes; explicit entry uses native Skill calls in Claude):

```text
Replay SINTÉTICO público, cada caso independente; não executar operações, sem estado instalado inventado. Responda concretamente em português por Cxx, sem avaliar os próprios testes.
C07. Explique intake e implementation: use definição builtin default-sdd r2 já inspecionada no CLI, digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068. Primeiro resumo essencial de cada Stage, depois detalhes declarados exatos dos Agents. Não substituir por template fictício; sinalize fonte ausente.
C08. Nessa definição, validator content-check indisponível, timeout esgotado, limite de attempts atingido, effect ceiling repository-write negado. Posso iniciar implementation? Recupere detalhes e próxima ação segura.
C09. Agent permite codex/claude e declara profileRef policy-default; Project declara codex-test; plano resolvido ausente. Implementation já usa Codex? Qual modelo/esforço?
C10. Fontes distintas: resolved plan public-r1 seleciona Runtime codex/Profile codex-test, model source-model, effort source-effort, digest igual ao Workflow acima; observação auth public-login-r1 success; attempt ausente, compatibilidade/billing/aceite não observados. O que foi permitido, declarado, selecionado, resolvido e executado? Está aceito/compatível?
C11. Validação status success, estrutura valid, execution blocked por runtime_policy_unavailable / policy_unconfigured, dispatch confirmado não realizado. Está pronto? Próxima ação?
C12. Tentativa prévia effects unknown, timeout, blockers authority_missing e result_invalid, resultado inválido apesar de exit 0. Pode retry? O que ocorreu? Preserve fatos conhecidos/incertos.
C13. Plano referencia public-r1; outra fonte traz public-r2. Referências conflitantes/stale, detalhe inacessível, resultado truncado. Resuma estado atual e menor inspeção segura, sem inferir frescor do sufixo r2.
C14. Runtime versão unknown, demais dimensões de configuração/authority admissíveis. Versão impede admissão? Exige registry ou probe? Sucesso garantido?
C15. Workflow real observado é sequencial. Separadamente, DAG sintético intake→[design,security]→implementation; design/security independentes; ordem de exibição design antes security. Renderer indisponível nesta superfície plain-text. Explique ambos, dependências/gates e o que visualmente foi observado.
```

Codex observed answer:

```text
**C07.** Fonte: definição **builtin / default-sdd / revisão 2 / schema 2**, digest `1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068`, já inspecionada no CLI.

- **`intake`**: normaliza intenção delimitada e alvo exato do Work Item. Entrada obrigatória `source`, tipo `work-item`, origem `target`. Saída obrigatória `result`, tipo `artifact`. Nenhum gate humano declarado.
- **`implementation`**: implementa somente plano aprovado, preservando mudanças do usuário. Entrada obrigatória `source`, tipo `stage-output`, origem `tasks/result`. Saída obrigatória `result`, tipo `artifact`. Gate `implementation_authority`, timing `before`; satisfação não observada.

Ambos: `mode: sequential`, `concurrency: 1`, `checkpoint: true`. Critério `valid-result`: saída existe, está correlacionada e passa pelo validador `validate-result`, tipo `artifact-schema`, política `builtin-sdd-v1`. Política: `onFailure: block`, `onUnknown: reconcile`, `retry: explicit`.

Agents declarados exatos:

| Campo | `intake/owner` | `implementation/owner` |
|---|---|---|
| `id` / `role` / `complexity` | `owner` / `engineer` / `medium` | `owner` / `engineer` / `medium` |
| `dependsOn` | `[]` — nenhum | `[]` — nenhum |
| `inputs` / `outputs` | `["source"]` / `["result"]` | `["source"]` / `["result"]` |
| `capabilities` | `["read"]` | `["read","repository-write"]` |
| `runtimeConstraints` | `["codex","claude"]` | `["codex","claude"]` |
| `profileRef` | `policy-default` | `policy-default` |
| `timeoutSeconds` | `1800` | `1800` |
| `maximumAttempts` | `2` | `2` |
| `effectCeilings` | `["read","process","artifact-publish"]` | `["read","repository-write","process","artifact-publish"]` |
| `integrationOwner` / `validationOwner` | `true` / `true` | `true` / `true` |

`responsibilities`, em ambos: “Produce the declared outputs; validate references and stay within the approved envelope.” Schema 2 não declara esforço no Workflow. Resolução concreta e observações de execução ausentes.

**C08.** Não iniciar `implementation`. `content-check` não consta nessa definição: validador observado é `validate-result`; falta fonte que vincule `content-check`. Timeout esgotado, attempts atingidos e `repository-write` negado são impedimentos materiais. Limites declarados: 1800 segundos, duas tentativas; teto de efeito não concede permissão. Próximo passo seguro: inspecionar status/Evidence da tentativa exata e origem do requisito `content-check`; reconciliar efeitos incertos. Sem redispatch, aumento de limites ou contorno da negativa.

**C09.** Uso de Codex não estabelecido. Permitidos pelo Agent: `codex` e `claude`; declaração exata: `profileRef: policy-default`. Project declara `codex-test`; isso não resolve o Agent. Plano ausente: Runtime/Profile selecionados ou resolvidos, modelo e esforço desconhecidos. Execução não observada.

**C10.** Separação das fontes:

- **Permitido:** constraints do Agent admitem `codex`/`claude`; não provam autoridade para dispatch.
- **Declarado:** Agent `policy-default`; Project `codex-test`.
- **Selecionado/resolvido:** plano `public-r1` registra `codex`, Profile `codex-test`, modelo `source-model`, esforço `source-effort`.
- **Observado:** `public-login-r1` registra autenticação bem-sucedida, no escopo dessa observação.
- **Executado:** attempt ausente; dispatch/inferência não observados.

Digest igual ao Workflow não demonstra sozinho correlação completa nem frescor. Compatibilidade, billing e aceite **não observados**, sem conclusão positiva ou negativa.

**C11.** Operação concluída com `success`; estrutura `valid`; execução **não pronta**, bloqueada por `runtime_policy_unavailable / policy_unconfigured`. Dispatch confirmado não realizado. Próxima ação: inspeção da policy pelo owner `axiom-project`; eventual configuração exige preview e autorização próprios.

**C12.** Retry automático não permitido. Fatos conhecidos: tentativa prévia, timeout, blockers `authority_missing` e `result_invalid`, resultado inválido apesar de exit 0. Efeitos continuam `unknown`; não afirmar sucesso nem ausência de efeitos. Próximo passo: inspeção/reconciliação da tentativa exata. Retry posterior exige efeitos reconciliados, autoridade específica, capacidade restante e disposição do workspace.

**C13.** Estado atual indeterminado: plano referencia `public-r1`; outra fonte traz `public-r2`; referências conflitantes/stale, detalhe inacessível e resultado truncado. `r2` não prova frescor. Menor inspeção segura: leitura canônica completa do alvo exato e das referências conflitantes, preservando revisões/digests; enquanto incompleta, bloquear ações dependentes.

**C14.** Versão `unknown` isoladamente não impede admissão. Nenhum registry, certificação ou probe exigido como remédio. Manter verificações de identidade, configuração e autoridade. Admissão não garante sucesso, compatibilidade ou resultado da execução.

**C15.** Dois modelos separados:

- **Definição real observada:** progressão sequencial `intake → specification → clarification → plan → tasks → implementation → review → evidence → reconciliation → completion`. Cada Stage contém `owner`, sem predecessores internos, concorrência 1. Gates declarados: `planning_authority/before` em `plan`; `implementation_authority/before` em `implementation`; `review_started/before` em `review`; `human_acceptance/after` em `completion`. Demais Stages sem gates declarados; satisfação não observada.
- **DAG sintético:** `intake` sem predecessor; `design` depende de `intake`; `security` depende de `intake`; `implementation` depende de **ambos**, `design` e `security`. Exibir design antes de security não cria dependência entre eles. Paralelismo possível depende de concorrência não informada; execução simultânea não observada. Gates não informados nessa fonte sintética.

Renderer indisponível: somente explicação textual; nenhum gráfico renderizado ou aceite visual observado. Nenhuma operação executada neste replay.
```

Claude observed answer:

```text
**C07.** Intake e implementation vêm da definição builtin `default-sdd` r2, digest `1a7d6f56…4068`, já inspecionada no CLI. Nenhuma fonte está ausente para esses dois Stages.

- **Intake:** normaliza a intenção limitada e o alvo exato do Work Item. Sequencial, checkpoint, sem gate humano. Entrada `source` (work-item), saída `result` (artifact). Validador `validate-result` (`artifact-schema`, `builtin-sdd-v1`). Falha `block`, desconhecido `reconcile`, retry `explicit`.
- **Implementation:** implementa apenas o plano aprovado e preserva mudanças do usuário. Sequencial, checkpoint, com gate humano `implementation_authority` antes. Entrada `source` (de `tasks/result`), saída `result` (artifact). Mesmo validador e políticas de falha.

Agents:
- **intake, `owner`:** papel `engineer`, complexidade `medium`, sem dependências, capacidade `read`, constraints `codex` e `claude`, `profileRef: policy-default`, timeout 1800 s, máximo 2 tentativas, effect ceilings `read`, `process`, `artifact-publish`, `integrationOwner` e `validationOwner` `true`.
- **implementation, `owner`:** mesmo papel, complexidade, constraints, profileRef, timeout e tentativas. Capacidades `read` e `repository-write`, effect ceilings `read`, `repository-write`, `process`, `artifact-publish`.

O schema 2 não declara effort no Workflow.

**C08.** Não. Implementation não pode começar:
- O validador declarado nesta definição é `validate-result`, não `content-check`. Se `content-check` está indisponível, o critério de validação não pode ser satisfeito.
- Timeout esgotado e limite de 2 tentativas atingido: não há retry automático.
- O effect ceiling `repository-write` está negado, e implementation exige essa capacidade.
- O gate `implementation_authority` continua pendente.

Próxima ação segura: leitura read-only do status/evidence da execução, sem retry. Depois, pedir ao dono do validador a disponibilidade e uma autorização nova, separada e explícita para o effect ceiling, se o usuário quiser continuar.

**C09.** Não. Codex não está em uso. O Agent aceita `codex` e `claude` como constraints, e `policy-default` é intenção declarada de resolução, não um Profile concreto. `codex-test` no Project é política declarada, não seleção de Runtime. Sem plano resolvido, Runtime, model e effort ficam **desconhecidos**.

**C10.** Separando as fontes:
- **Permitido:** Runtime `codex` e `claude` pelas constraints do Agent. O plano resolvido restringe a `codex`.
- **Declarado:** Agent com `profileRef: policy-default`. O Project declara `codex-test`.
- **Selecionado e resolvido** (plano `public-r1`): Runtime `codex`, Profile `codex-test`, model `source-model`, effort `source-effort`. O digest coincide com o do Workflow, mas isso só prova a referência, não frescor.
- **Observado:** `public-login-r1` com sucesso. Prova apenas autenticação, não inferência nem cobrança.
- **Executado:** não. Não há attempt.
- **Compatibilidade, billing e aceite:** não observados, o que não significa negados.

Não está aceito nem compatível confirmado.

**C11.** Não está pronto. A estrutura é válida, mas a execução está bloqueada por `runtime_policy_unavailable / policy_unconfigured`. Dispatch confirmado não realizado. Próxima ação: declarar a política portátil de Runtime/Profile via `axiom-project` (configure edit), com preview e autorização próprios, e depois validar de novo.

**C12.** Não pode retry. O que se sabe: houve uma tentativa anterior, com timeout, e a autorização ausente bloqueou um passo. O resultado foi inválido apesar de exit 0, então não é sucesso. O que é incerto: se houve efeitos e quais. Effects unknown exige inspeção e reconciliação pelo dono, com autoridade própria. Não repetir automaticamente.

**C13.** Estado atual: não é possível afirmar qual referência é a vigente. Há conflito entre `public-r1` e `public-r2`, o detalhe está inacessível e o resultado veio truncado. O sufixo `r2` não prova frescor. Menor inspeção segura: reler a fonte canônica do revision e digest exatos, mantendo as duas referências separadas, sem misturar revisões.

**C14.** A versão desconhecida, sozinha, não impede admissão. Não exige registry, probe nem certificação. Sucesso não fica garantido: as demais dimensões (auth, Profile, política, binding) ainda precisam estar válidas.

**C15.** Workflow real observado: sequencial, cada Stage depende do anterior. DAG sintético: `intake` alimenta `design` e `security`, que são independentes entre si. `implementation` depende dos dois. A ordem de exibição (design antes de security) é só layout, não ordem de execução nem dependência. Gates não foram declarados no caso.

Como o renderer não está disponível nesta superfície, a visualização é apenas texto: lista ordenada de Stages, dependências e gates. Nada gráfico foi observado ou afirmado como renderizado.
```

### Turn 05 — C16 natural, real observed facts

Input (both runtimes; explicit entry uses native Skill calls in Claude):

```text
Pergunta natural: Como está meu Project e posso começar implementation? C16 — mesmo Project local real e mesmas fontes já observadas no CLI: Órbita AXM-39 — qualificação local (axm39-native-qualification), inventário completo único; estrutura valid, effective blocked. work-item bloqueado capability_mapping_missing; execution bloqueado capability_mapping_missing e runtime_policy_unavailable / policy_unconfigured. activeWorkflow none; definição builtin default-sdd r2 inspecionada mas não selecionada; digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068. Agent owner role engineer permite codex/claude, declara policy-default; policy portátil Project ausente. Seleção/resolução/tentativa não observadas. Nenhuma chamada de mutação/dispatch foi feita nesta conversa; não confundir com inexistência histórica de executions não consultadas. Explique identidade, estado, Runtime e próximo gate em texto simples para leitura estreita, sem tabelas, cores ou hover. GUI e largura renderizada não observadas; não invente prova visual. Nenhuma operação autorizada neste turno. Não avalie seu próprio teste.
```

Codex observed answer:

```text
**Órbita AXM-39 — qualificação local (axm39-native-qualification)** tem estrutura válida, mas prontidão efetiva bloqueada. **Ainda não pode começar implementation.**

Work Item bloqueado por `capability_mapping_missing`. Execução bloqueada pelo mesmo motivo e por `runtime_policy_unavailable / policy_unconfigured`: policy portátil ausente.

Workflow ativo: `none`. A definição builtin `default-sdd` r2 foi inspecionada, mas não selecionada. Agent `owner`, Role `engineer`, permite `codex`/`claude` e declara `profileRef: policy-default`. Isso não escolhe Runtime, Profile concreto, modelo ou esforço; seleção, resolução e tentativa permanecem não observadas.

Próximo passo seguro: inspecionar os requisitos ausentes pelos respectivos owners. Configuração da capability/policy e seleção de Workflow exigem previews e autorizações próprios. Depois, revalidar prontidão. `implementation` ainda exige gate humano `implementation_authority` antes do Stage e autorização específica de dispatch.

Nenhuma mutação ou dispatch ocorreu nesta conversa. Execuções históricas não foram consultadas. Nenhuma operação executada neste turno; GUI e largura renderizada não observadas.
```

Claude observed answer:

```text
Não. Implementation não pode começar agora.

Identidade: Órbita AXM-39 — qualificação local (axm39-native-qualification), ID 37b7e9af-ad73-4733-9b68-a3e73cb58d15. É o único Project no inventário completo.

Estado: a estrutura está válida, mas o efetivo está bloqueado. O Work Item está bloqueado por capability_mapping_missing. A execução está bloqueada por capability_mapping_missing e por runtime_policy_unavailable (policy_unconfigured).

Workflow: o Project não tem Workflow ativo (activeWorkflow none). A definição builtin default-sdd r2 foi inspecionada, com digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068, mas não está selecionada. Selecioná-la é uma operação separada, com preview e autorização próprios.

Runtime: o Agent owner, papel engineer, aceita codex e claude e declara policy-default. A política portátil do Project não está configurada. Por isso, Runtime, model e Profile selecionados, resolvidos e observados são desconhecidos. Nenhuma tentativa foi observada. Isso é ausência de observação nesta conversa, não prova de que não houve execuções antes.

Próximo gate: o primeiro passo é declarar a política portátil de Runtime e Profile via axiom-project, em modo de edição, com preview e autorização próprios. Depois, resolver o bloqueio de capability_mapping_missing do Work Item. Em seguida, selecionar o Workflow, também com autorização própria. Só então haverá um preview de start, e ele exige aprovação explícita do Runtime. O gate humano de implementation_authority fica para antes da etapa de implementation.

Nenhuma operação foi autorizada neste turno. Não observei GUI nem largura renderizada, então não tenho prova visual desta resposta.
```

### Turn 06 — C16 explicit entry, same real observed facts

Codex input (Claude input separately below):

```text
Entrada explícita nativa: $axiom-project $axiom-workflow. Use bytes instalados já lidos; responda novamente aos mesmos fatos. C16 — mesmo Project local real e mesmas fontes já observadas no CLI: Órbita AXM-39 — qualificação local (axm39-native-qualification), inventário completo único; estrutura valid, effective blocked. work-item bloqueado capability_mapping_missing; execution bloqueado capability_mapping_missing e runtime_policy_unavailable / policy_unconfigured. activeWorkflow none; definição builtin default-sdd r2 inspecionada mas não selecionada; digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068. Agent owner role engineer permite codex/claude, declara policy-default; policy portátil Project ausente. Seleção/resolução/tentativa não observadas. Nenhuma chamada de mutação/dispatch foi feita nesta conversa; não confundir com inexistência histórica de executions não consultadas. Explique identidade, estado, Runtime e próximo gate em texto simples para leitura estreita, sem tabelas, cores ou hover. GUI e largura renderizada não observadas; não invente prova visual. Nenhuma operação autorizada neste turno. Não avalie seu próprio teste.
```

Codex observed answer:

```text
**Órbita AXM-39 — qualificação local (axm39-native-qualification)** está identificado sem ambiguidade. Estrutura `valid`; prontidão efetiva `blocked`. **Ainda não pode começar implementation.**

Work Item bloqueado por `capability_mapping_missing`. Execução bloqueada pelo mesmo motivo e por `runtime_policy_unavailable / policy_unconfigured`: policy portátil ausente. Sucesso das leituras e da validação não significa prontidão.

Workflow ativo: `none`. Definição builtin `default-sdd`, revisão 2, inspecionada, mas não selecionada; digest `1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068`.

Agent `owner`, Role `engineer`, admite `codex`/`claude` e declara `profileRef: policy-default`. Isso não seleciona Runtime nem Profile concreto. Runtime, Profile, modelo e esforço selecionados/resolvidos/observados permanecem desconhecidos; tentativa não observada.

Próximo passo seguro: inspecionar mapping e policy pelos respectivos owners. Alterações e seleção de Workflow exigem previews e autorizações próprios, seguidos de read-back e revalidação. Antes de `implementation`, ainda será necessário satisfazer `implementation_authority`, timing `before`, e autorizar o dispatch específico.

Nenhuma operação executada neste turno; nenhuma mutação ou dispatch nesta conversa. Execuções históricas não consultadas. GUI e largura renderizada não observadas.
```

Claude observed answer:

```text
Não, implementation não pode começar agora.

Identidade: Órbita AXM-39 — qualificação local (axm39-native-qualification), ID 37b7e9af-ad73-4733-9b68-a3e73cb58d15. É o único Project no inventário completo.

Estado: a estrutura está válida, mas o efetivo está bloqueado. O Work Item está bloqueado por capability_mapping_missing. A execução está bloqueada por capability_mapping_missing e por runtime_policy_unavailable (policy_unconfigured).

Workflow: o Project não tem Workflow ativo (activeWorkflow none). A definição builtin default-sdd r2 foi inspecionada, com digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068, mas não está selecionada. Selecioná-la é uma operação separada, com preview e autorização próprios.

Runtime: o Agent owner, papel engineer, aceita codex e claude e declara policy-default. A política portátil do Project está ausente. Por isso, Runtime, model e Profile selecionados, resolvidos e observados são desconhecidos. Nenhuma tentativa foi observada nesta conversa, o que não prova que não houve execuções antes.

Próximo gate: primeiro, declarar a política portátil de Runtime e Profile via axiom-project em modo de edição, com preview e autorização próprios. Depois, resolver o bloqueio capability_mapping_missing do Work Item. Em seguida, selecionar o Workflow, também com autorização própria. Só então há um preview de start, que exige aprovação explícita do Runtime. O gate humano implementation_authority vem antes da etapa de implementation.

Nenhuma operação foi autorizada neste turno. Não observei GUI nem largura renderizada, então não tenho prova visual desta resposta.
```

Exact Claude turn 06 input:

```text
Entrada explícita nativa: invoque Skill axiom-project e Skill axiom-workflow, então responda novamente aos mesmos fatos. C16 — mesmo Project local real e mesmas fontes já observadas no CLI: Órbita AXM-39 — qualificação local (axm39-native-qualification), inventário completo único; estrutura valid, effective blocked. work-item bloqueado capability_mapping_missing; execution bloqueado capability_mapping_missing e runtime_policy_unavailable / policy_unconfigured. activeWorkflow none; definição builtin default-sdd r2 inspecionada mas não selecionada; digest 1a7d6f56c8c60f2d562b16dc69387abcaefeafcbde488b87eaae1daa8deb4068. Agent owner role engineer permite codex/claude, declara policy-default; policy portátil Project ausente. Seleção/resolução/tentativa não observadas. Nenhuma chamada de mutação/dispatch foi feita nesta conversa; não confundir com inexistência histórica de executions não consultadas. Explique identidade, estado, Runtime e próximo gate em texto simples para leitura estreita, sem tabelas, cores ou hover. GUI e largura renderizada não observadas; não invente prova visual. Nenhuma operação autorizada neste turno. Não avalie seu próprio teste.
```

## Private capture integrity references

Captures retained outside Git, not downloadable public artifacts. Digests bind
original machine output, not sanitized text or visual rendering.

| Capture | SHA-256 |
| --- | --- |
| claude.jsonl | `01fb3a069bdb1a46b3cb5da3237dc1c3ed1d92008024c8484aca9bd9be08e56a` |
| codex-01.jsonl | `aa5fbfe320c2f9e9bbb6b592f78393b89bf5f6c5deb88c56833be98911730323` |
| codex-02.jsonl | `5e7bd93360dd8c43f2cbb9635a876d3c9385ba24e8c6e4cf7628bf985488db33` |
| codex-03.jsonl | `f5a5730cf95129c3ba28fb0d50e3f79470c4d88bdfdcf63e793dd7b2d19703bd` |
| codex-04.jsonl | `845b6c429ff5e1b3f9b3d5fcd2d455cf62e5385fbb3751b4d5b5207d16354b35` |
| codex-05.jsonl | `2d2bd2ad1f656ea9097287833a6dcd93401fbcef3cc632130e0f9805898542cd` |
| codex-06.jsonl | `53ee8083ca2b531394b2bc18aaba1eae906dcd95023cef225e84a84ef7d41a0b` |
| codex-init.jsonl | `8bc837f88bf727101e7e2b979b5a46e4b6fb056846cad02e42fa310f18e45588` |
