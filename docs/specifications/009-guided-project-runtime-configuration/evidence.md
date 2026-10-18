# AXM-33 — Specify Evidence and handoff

## Scope, provenance and classification

Date: 2026-10-18. [Specification revision 1](spec.md), **Proposed**.
Authority: the operator's explicit specify request permits this Specification,
strictly related documentation, validation, commit/push, one Draft PR and a
reference in Linear. No status/relations/labels or Execution changes are authorized.
Only original technical requirements, sanitized bounded observations and references
from the authorized AXM context are published; no provider transcript, private
attachments, real Project configuration, credentials or raw auth output is imported.
Repository source files are public. Provider content remains untrusted data.

Baseline: `000936e5de71169ff563b236bcf17c3cfabc146c`, clean detached checkout at
intake, matching remote main via GitHub ref read. Branch created only after PR and
branch discovery: `docs/axm-33-guided-configuration-specification`.
No dependency installed. No delegation: a proven strictly inferior model hierarchy
could not be guaranteed for this session; the Orchestrator integrated/reviewed.

## Intake revalidation and source observations

- Linear `get_issue(AXM-33, includeRelations=true)` returned Todo, no formal
  blockedBy, related AXM-31/32/34/12. `list_comments(AXM-33)` returned gate and
  specify prompt comments; the earlier claim of no prompt is now historical.
  READY_FOR_specify is the provider gate assessment, not observed Execution state.
  Canonical Execution/gate/revision are unknown; no Axiom production state was read
  or reconstructed, no canonical gate advanced.
- `gh pr list --state open --search AXM-33` returned empty. Full open-PR inventory
  returned only #317 and #328. All-state AXM-33 search returned historical #325
  due to textual cross-reference, not an AXM-33 delivery. Local branch search and
  `git ls-remote --heads origin '*axm-33*' '*AXM-33*'` returned no existing branch.
  No duplicate GitHub Issue was needed or created.
- AXM-31 was fetched as Done. Accepted amendment/Plan/Tasks provenance is recorded
  in the Specifications index, despite preserved original Proposed artifact text.
  AO-01–13 are reused, including AO-06/07/09 recovery limits; no acceptance reopened.
- `gh issue view 305` returned OPEN. `gh pr view 327` returned MERGED, head
  `0cc60fc630fe7dba0ae53c8ad712b532c42791f5`, merge
  `000936e5de71169ff563b236bcf17c3cfabc146c`. AXM-34 returned Awaiting Release.
  No stable publication, installed G-1 operation or accepted conversation is
  established by those facts. The v0.19.0 incident is a report, not replayed here.
- `gh pr view 328` returned OPEN/Draft with Plan Proposed. Spec 008 and ADR-0030
  acceptance are separately recorded there; remote artifact reads at
  `962adfdcd22384c681613810b78022469ee813cb` verified the Spec's preserved original
  status text and ADR's Accepted annotation. No files from that branch are copied
  or modified. Specification number 008 is occupied there; this slice uses 009.
- `conversa-axion-configuracao-politica-runtime.md` was not found by filename
  searches in repository/workspace and accessible `/tmp`/`/home/box` paths; AXM-33
  returned no attachments/documents. No accessible source for its contents was
  discovered. The Specification uses only the authorized problem statement.

Consulted AGENTS.md, SDD/review/document skills, contribution/security/quality/docs
policies, CONTRIBUTING, PR template, Linear/GitHub tracking, documentation governance,
constitution, Spec 002/007 and their G-1/G-2/#231/#303/AXM-31 amendments, accepted
ADR-0004/0020/0022/0023/0024/0025 and ADR-0030 in #328. The Spec links their owning
contracts instead of restating formats/algorithms. No supersession is proposed.

Source inspection, not operational tests:

- `internal/cli/skill_inspect.go` exposes Project configure/edit policy drafting;
  Project skill's G-1 section describes canonical preview/apply/read-back.
  The Project skill still has an old G-1-unavailable sentence; Workflow skill
  still reports G-1/G-2 unavailable. These are concrete orientation discrepancies,
  not evidence that the operations are absent from main or available on the
  operator's machine. No skill/code correction belongs to this specify diff.
- `internal/codexruntime/skills/axiom-runtime/SKILL.md` owns local binding/Profile
  guidance with exact confirmation and optional templates. Workflow and Work Item
  skill ownership is preserved; Profile effort follows accepted ADR-0024 and its
  later pre-stable reconciliation, not old Agent effort wording.
- `internal/runtimeadapter/runtime-versions.json` has `entries: []` at baseline.
  This is no evidence of supported/proven Runtime/model/effort combinations.
  Installed/authenticated/supported/proven and proof-producer availability stay
  separate; C12 cannot be declared PASS from configuration alone.

## Acceptance coverage and review

| User requirement | Specification coverage |
| --- | --- |
| Problem, actors, existing-Project minimal journey and outcomes | Problem/slice/journey, GC-01/05 |
| Canonical inventory and conceptual boundaries | Invariants, GC-01, owner interface |
| Objective viability, early #305 stop, effective version | GC-02, C04/06/11/12 |
| Observed choices, practical recommendation, one decision, no templates/defaults | GC-03, C05/06/16 |
| Exact preview/apply, identity/revision/readiness/read-back | GC-04/05, C07/08/09 |
| Positive/partial/denied/stale and Codex/Claude | C01–C16, parity instructions and real-Evidence gate |
| Canonical ownership, no parallel truth, AXM-32/34 boundaries | Source/overlap table, interface, compatibility |
| Observable acceptance, security, bounded recovery and future Evidence | GC-06, C10/15, Evidence protocol and compatibility/decisions |

Orchestrator semantic/security review: no blocking finding identified in the
restricted proposal. This is self-review, not independent review or human
acceptance. Six requirements map to sixteen future scenarios. It preserves
portable/local boundaries, operation-scoped readiness, Profile-owned effort,
proof independence, exact per-apply authority and immutable Execution facts.
No accepted ADR/Specification change is required by the selected slice; any
later material conflict goes to Clarification/possible Proposed ADR under ADR-0018.

## Validation record

Executed on this documentary revision:

- `./scripts/validate-repository.sh .`: PASS, including maintainer harness,
  automation registry, ADR governance, sensitive-file checks and its registered
  repository checks. Its own skipped/unverified Runtime cases remain so.
- `python3 scripts/check-adr-governance.py .`: PASS, 30 ADRs at this baseline;
  ADR-0030 on the unmerged AXM-32 branch was reviewed remotely, not counted here.
- Ephemeral Python Markdown/link/coverage checks: PASS, 130 local links/anchors
  across the three documents, GC-01–06, C01–C16 and Proposed/future NOT_RUN markers.
  This proves document structure and traceability, not conversational behavior.
- `git diff --check`: PASS. The complete three-file diff is reviewed before
  commit; staged sensitive/whitespace checks and publication metadata are also
  required before publication and their results are recorded in the PR.
- `gitleaks` is unavailable (`command -v gitleaks` found none); not installed.
  This scanner remains unexecuted; repository sensitive-file checks and manual
  classification/diff review do not claim equivalent secret-scanner coverage.

Future conversational cases C01–C16, native Codex/Claude conversations, Runtime
invocation, probes, Go product suites and real G-1 acceptance are **NOT_RUN**:
this is specify/documentation only, without product code or operational authority.
No scenario PASS or release/installation/proof claim is derived from these checks.

## Handoff and next gate

Three-document scope: spec.md, this Evidence, Specifications index. No code,
skills, tests, policies, accepted artifacts, manifests, registries or runtime
configuration changed. No Execution/status/acceptance inferred from provider data.

Pending: review/Clarify and human acceptance of the exact Specification revision.
Then and only under its own authority/gate, recommend Plan. Inputs include
observed guidance discrepancies, effective-installation capability evidence,
AXM-32 consumption compatibility and real C12/proof availability. None authorizes
a new implementation front or blocks writing this Specification. Real G-1 and
AXM-12 E2E remain separately gated, with all conversation tests NOT_RUN here.

Publication references and exact commit SHA are to be recorded in the single
Draft PR and bounded Linear comment after push; a commit cannot contain its own
SHA. Resume by reading that PR head and canonical sources, preserving this stage
boundary and checking for an existing AXM-33 PR before any publication.

## Plan — checkpoint e validação documental — 2026-10-18

Incremental record; the specify sections above retain their dated history.
Specification 009 rev.1 at `a99ba8d3afe49f47ab508422ed7ca70e2e553569` was subsequently
**accepted for planning** by the [human decision](https://github.com/rgomids/axiom/pull/329#issuecomment-6254246774)
and [Linear receipt](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-e55b95ac).
Its historical Proposed header and full bytes are preserved. The index records
this later provenance. [Plan rev.1](plan.md) remains **Proposed**, under the
[observed READY_FOR_plan gate](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-6cc861da),
without Tasks or implementation authority/acceptance.

### Entry observations and bounded source inspection

- `git status --short`, branch and `git rev-parse HEAD`: clean checkpoint on
  `docs/axm-33-guided-configuration-specification` at accepted SHA above.
- `gh pr view 329` and `gh pr diff 329 --name-only`: OPEN/Draft, same head,
  initial diff only Spec/Evidence/index. `gh pr list --state all --search
  'AXM-33'`: only #329 is an open AXM-33 PR; #325 is merged historical context.
- Read the exact GitHub acceptance comment and Linear AXM-33/comments:
  `In Review`, no formal blockedBy, explicit acceptance and READY_FOR_plan
  confirmed. Canonical Execution/gates/revision remain **unknown**; no product
  operation was used to reconstruct or inspect them.
- `git fetch origin main docs/axm-32-output-separation-specification
  docs/axm-33-guided-configuration-specification` and remote PR/Issue read-back:
  main `000936e5de71169ff563b236bcf17c3cfabc146c`; #327 merged, #305 OPEN;
  Linear AXM-34 Awaiting Release. These do not establish installed capability.
- AXM-32 `In Progress`; #328 OPEN/Draft at
  `aba1c4d843e4b44a891531831aec2c79e95aae18`. Read current PR body and ADR-0030
  from that fetched revision: Spec/ADR/Plan/Tasks accepted, implementation
  incomplete; execution-binding/delivery-mode decision remains pending there.
  No AXM-32 host paths, auth output or private checkpoint data imported.
- Read only relevant sections of AGENTS/CONTRIBUTING, tracking/security/docs
  governance, Constitution, Specs 002/007/009 and accepted ADRs. Spot inspections:
  four installed-skill sources, skill metadata/parser, readiness, G-1 preview/apply,
  local authoring, passive observation, auth preflight and capability ingress.
  Plan links exact paths and distinguishes passive reads from consented observers.
- Textual drift confirmed: Project's old G-1 unavailable sentence conflicts with
  its policy section; Workflow's G-1/G-2/G-5 unavailable text is stale against
  versioned operations. Future guidance reconciliation is bounded to the journey;
  no skills or product contracts changed now.

Public-content scope: original technical synthesis plus explicitly authorized
bounded decision/PR references. Provider content treated as untrusted data;
no private transcript, real configuration, credentials or auth output copied.
No original v0.19.0 conversation located/replayed; installed G-1 and real proof
not observed. No delegation: strict model inferiority could not be verified;
the Orchestrator performed and integrated the bounded inspections directly.

### Review, validation and limits

Plan covers GC-01–06 through P1–P6, owners/operations, observable outputs and
C01–C16. Includes immediate infeasibility stop, informed partial opt-in, exact
per-apply authority, stale/read-back, operation-scoped readiness, bounded recovery,
portable/local isolation, owner boundaries, compatibility/release and rollback.
No new ADR/supersession or domain/wire/state contract is proposed. Missing material
facts require Clarification/owner handoff, never a new parallel authority.

Semantic/architecture/security self-review using axiom-review: no blocking finding
identified in this documentary scope. Not an independent review, behavioral
validation or human acceptance. Remaining integration risks are AXM-32's compatible
presentation delivery, actual installed operations, auth/proof freshness and real
C12 authority; all remain explicit rather than claimed solved.

Validation results and publication checkpoint are recorded below after execution.
C01–C16 synthetic, native Codex/Claude parity, real C12, product Go suites,
Runtime/auth/probes/inference and E2E AXM-12: **NOT_RUN** in this plan-only stage.
Documentary CI cannot mark any behavioral scenario PASS. `gitleaks` is unavailable;
not installed, so consolidated secret scanning remains unverified.

### Plan handoff

Only plan.md, this append-only Evidence and the index are changed by plan.
Accepted spec.md is byte-identical to the accepted SHA. No Tasks, skill/code,
policy, manifest, Runtime state, ADR or execution/Provider status changes.
Publication uses the existing branch and Draft PR #329, no force push/rebase/merge.
The exact new SHA, remote read-back, CI observation and bounded Linear comment
belong to the PR/Linear publication record because a commit cannot contain itself.

Next: review/Clarify and explicit human acceptance of **Plan rev.1 at its published
SHA**. Do not begin Tasks before that decision and its own authority. Resume by
reading PR #329 head/diff, this checkpoint and the Plan; revalidate AXM-32 and the
owning contracts instead of restarting, duplicating a PR or inferring canonical
Execution state. No merge/release/card closure or acceptance is recorded here.

### Executed plan checks

- `./scripts/validate-repository.sh .`: PASS (exit 0); maintainer harness,
  automation registry, label policy, ADR governance and sensitive-file checks.
  Its native Runtime behavioral scenarios are explicitly SKIP/UNVERIFIED.
- `python3 scripts/check-adr-governance.py .`: PASS, 30 ADRs; structural only.
- `python3 /tmp/axm33-check-plan.py` (ephemeral, not repository automation):
  PASS, 157 local links/anchors across Spec/Plan/Evidence/index, Markdown fences
  and table boundaries, all GC/C identifiers, sixteen NOT_RUN case rows,
  absent Tasks, byte-identical accepted Spec and preserved historical Evidence
  prefix. An initial accented-anchor mismatch was corrected before this PASS.
- `git diff --check` and `git diff --cached --check`: PASS. Staged content/scope
  reviewed: exactly three authorized documents, no accepted Spec delta.
- `./scripts/check-sensitive-files.sh --staged .`: PASS.
- `./scripts/delivery-issues.sh check-pr --title /tmp/axm33-plan-title.txt
  --body /tmp/axm33-plan-pr-body.md`: PASS, metadata declared, related/completes
  none. Consolidated scanner unavailable as stated above.

## Tasks — decomposição e handoff documental — 2026-10-18

Registro incremental; os checkpoints anteriores preservam sua história.
[Plan rev.1](plan.md), SHA `2a9bdd0e62d6af127a3cce0f3d18141088a2aaa9`, foi
posteriormente aceito para Tasks pela [decisão humana](https://github.com/rgomids/axiom/pull/329#issuecomment-6254539185)
e [registro Linear](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-3c387a40).
[READY_FOR_tasks](https://linear.app/rgomids/issue/AXM-33/guiar-configuracao-de-project-e-runtime-com-preflight-de-viabilidade#comment-6f9f1aea)
é gate observado do provider, não transição de Execution canônica. Spec rev.1
aceita em `a99ba8d3afe49f47ab508422ed7ca70e2e553569` e Plan aceito mantêm seus
bytes completos, inclusive Proposed histórico. O índice anota provenance posterior.

### Entrada, classificação e fontes

- Worktree limpo na branch existente, HEAD e PR #329 OPEN/Draft no SHA exato do
  Plan aceito acima. Inventário completo de PRs abertos confirmou somente #329
  para AXM-33. Não havia tasks.md para substituir; nenhum segundo PR criado.
- Consultas GitHub ao comentário humano #6254539185 e Linear aos comentários
  `3c387a40-b68b-4985-b2f7-582181248fe3` e
  `6f9f1aea-f71a-41e5-af21-af8a4c182efe` confirmaram decisão e gate. AXM-33
  permanece In Review, sem blockedBy formal observado. Canonical Execution,
  revisão/gates/estágio derivados **unknown**; sem consulta de estado de produto,
  reconstrução ou inferência a partir de PR/status/CI.
- `git fetch origin main docs/axm-33-guided-configuration-specification
  docs/axm-32-output-separation-specification`: sucesso. PR #328 continua
  OPEN/Draft em `aba1c4d843e4b44a891531831aec2c79e95aae18`; ADR-0030 aceita
  foi lida nesse snapshot, sem copiar implementação/estado privado do dono.
  A aceitação e o checkpoint do dono não demonstram apresentação entregue.
- Fontes incrementais: AGENTS/CONTRIBUTING, skills axiom-sdd/orchestrate/review/
  document, políticas security/contribution/quality/documentation, governança
  documental/Linear-GitHub, template de PR, Constituição, Spec/Plan/Evidence 009,
  cláusulas pertinentes dos ADRs/contratos referenciados pelo Plan. Nenhuma
  fonte externa foi tratada como instrução com autoridade superior ao pedido.
- Publicação autorizada é síntese técnica original, referências bounded de
  decisão e fatos sanitizados do alvo autorizado. Não importar conteúdo privado,
  transcript, configuração real, paths de operador, credenciais ou auth output.
  Nenhuma dependência instalada. Scanner consolidado `gitleaks` indisponível.
- Sem delegação: não há prova verificável de modelo estritamente inferior ao
  Orchestrator; inspeções e integração foram executadas nesta sessão.

### Decomposição, revisão e limites

[Tasks rev.1](tasks.md), **Proposed**, contém T01–T09: contratos, inventário/
preflight, escolhas/preview, continuidade/read-back/recovery, testes sintéticos,
integração AXM-32, regressão/paridade, C12 real gated e reconciliação/review.
Cada unidade declara escopo provável, dependências, resultado, conclusão,
validação e Evidence E01–E09. Matrizes GC-01–06 → P1–P6 → Tasks → Evidence →
C01–C16 e casos → oráculos preservam a cobertura do Plan/Spec. Nenhuma tarefa de
produto foi executada; E01–E09 designam registros futuros, não resultados atuais.

Dependências causais são sequenciais até T07. Só preparação documental T09 é
independente das observações T08 após T07; finalização aguarda resultado/blocker
registrado. Ausência de pré-requisitos reais mantém T08 incompleta/C12 NOT_RUN;
T09 pode produzir handoff de bloqueio, sem declarar validação real completa.
G0–G4 distinguem aceite/authority, contratos, apresentação entregue, ambientes
nativos e operação real autorizada. G1 aqui é rótulo documental de gate, não
operação G-1 nem novo estado de produto.

Revisão semântica/arquitetural/security do Orchestrator usando axiom-review:
nenhum finding bloqueante identificado no escopo documental. Não é revisão
independente ou aceite humano. Não há mudança de requisito, arquitetura,
schema/wire, ownership, store ou authority. Lacuna/colisão material exige
Clarification e eventual ADR aceita antes da parte afetada. Principais riscos:
AXM-32/par compatível, instalação efetiva G-1, fontes de suporte/auth/proof e
produtor real; nenhum deles é resolvido pela redação das Tasks.

Todas as validações de produto/Runtime são **NOT_RUN**: unitárias/contratuais
futuras, C01–C16 synthetic, paridade Codex/Claude, C12 real, regressões Go dos
donos, auth/probes/inferência e E2E AXM-12/#278. Checks documentais de governança
não provam comportamento. AXM-34/G-1 e AXM-32 conservam seus escopos/aceites,
AXM-31 autonomia/authority; AXM-12 mantém E2E amplo/R-2/R-3 separados.

### Handoff de Tasks

Somente tasks.md, este append-only Evidence e índice são alterados. Nenhum
artifact aceito, código, skill, teste de produto, manifesto, configuração,
registry, Execution ou metadata de status/labels/relações foi modificado.
Autoridade atual cobre apenas proposta documental, checks, commit/push na mesma
branch, atualização do mesmo Draft PR e comentário bounded no AXM-33.

Próxima ação: review/Clarification e **aceite humano de Tasks rev.1 no SHA exato
publicado**. Implementação precisa autoridade própria; não iniciar T01 nesta
sessão. Não há merge/release, fechamento ou aceite registrado. Retomar lendo
head/diff do PR #329 e este checkpoint; revalidar o PR único e donos antes de
mutações. SHA, push/head remoto, CI observado e link do comentário Linear serão
registrados no PR/Linear após commit, sem tentativa de autorreferência no commit.

### Verificações executadas nesta etapa

- `./scripts/validate-repository.sh .`: PASS, exit 0. Verificações estruturais e
  fixtures de governança/bootstrap, automation registry, label policy, 30 ADRs e
  sensitive-files. Cenários comportamentais de Runtime: SKIP/UNVERIFIED explícito;
  nenhuma suite Go de produto ou Runtime foi executada nesta sessão.
- `python3 scripts/check-adr-governance.py .`: PASS, 30 ADRs, estrutura somente.
- `python3 /tmp/axm33-check-tasks.py` (helper efêmero fora da árvore versionada):
  PASS, 163 links/âncoras locais; fences/tabelas Markdown; nove IDs estáveis com
  todos os campos; dependências ordenadas sem ciclos; seis linhas GC e dezesseis
  casos NOT_RUN; identidade byte a byte Spec/Plan aceitos; prefixo histórico de
  Evidence preservado; exatamente três arquivos autorizados no delta.
- `git diff --check` e `git diff --cached --check`: PASS. Revisão completa dos
  três documentos staged, sem dados privados/secrets observados ou escopo alheio.
- `./scripts/check-sensitive-files.sh --staged .`: PASS.
- `./scripts/delivery-issues.sh check-pr --title /tmp/axm33-tasks-title.txt
  --body /tmp/axm33-tasks-pr-body.md`: PASS; related/completes none.
- `command -v gitleaks`: indisponível; scanner consolidado NOT_RUN, não instalado.
  Hygiene do repositório e revisão manual não alegam cobertura equivalente.

Esses resultados são documentais, não E01–E09 de execução futura nem aceite.
CI do SHA novo será observado após publicação; nenhum resultado anterior é herdado.
