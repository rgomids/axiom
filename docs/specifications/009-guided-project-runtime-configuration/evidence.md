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

## Implement — orientação e gates de validação — 2026-10-18

### Authority, baseline e E01 — contratos observados

Tasks rev.1 T01–T09 aceitas no SHA
`6d5046440a2638630bfb729dfd8cff7184d74e23` pela
[decisão humana](https://github.com/rgomids/axiom/pull/329#issuecomment-6254729819)
e [recibo Linear](https://linear.app/rgomids/issue/AXM-33#comment-48abcf6e).
O prompt implement autoriza edição no escopo, testes, commit/push no PR único e
handoff bounded em AXM-33, sem merge/release/aceite ou transição canônica. Spec,
Plan, Tasks e todo o prefixo histórico deste Evidence permanecem intactos.

Read-back inicial: árvore limpa, branch
`docs/axm-33-guided-configuration-specification`, HEAD e head remoto/PR exatamente
`6d5046440a2638630bfb729dfd8cff7184d74e23`; PR #329 OPEN, base main.
`git log -6 --oneline`, `git ls-remote origin` e `gh pr view 329` não mostraram
commits de implementação nesta branch. Linear foi reobservado **In Progress**,
diferente do snapshot In Review do prompt; nenhum status foi alterado pelo
executor. Comentários de aprovação/prompt não fornecem uma Execution canônica.
Não há `axiom` no PATH; não se consultou nem enumerou estado privado para
reconstruir Execution. Execution/revisão/gates/estágio permanecem **unknown**;
nenhuma segunda Execution foi criada ou inferida.

Orquestração: cadeia causal T01→T02→T03→T04→T05; ativação T06 depende de G2,
T07 de T06/G3, T08 de T07/G4. Consolidação de bloqueios não satisfaz dependências
não observadas. Sem delegação: inferioridade estrita de modelo não comprovável.
Orchestrator executou explore/implement/validate/review/document; comandos de
validação independentes foram paralelizados. Nenhuma infraestrutura genérica,
engine, nova interface, dependência ou automation durável foi necessária:
Markdown embedded, testes e biblioteca padrão existentes bastam; biblioteca
externa adicionaria superfície sem requisito validado.

| Lacuna / dono | Fonte inspecionada e forma real | Pré-requisito / authority / confirmação |
| --- | --- | --- |
| Identidade, Repository, Provider / Project | `internal/cli/skill_inspect.go`, Project skill; list/show/validate, configure edit explícito | UUID/source/selector, binding por chave, preview completo; `project-id`, `preview-digest`, `authorize-local` somente do preview exato; show/validate após write |
| Policy / G-1 AXM-34 | `internal/projectapp/policy_{preview,apply}.go`, `internal/local/policy_edit_observation.go`; configure edit com policy-draft exclusivo; `{}` observa policy | Schema suportado, snapshots installation/source/Workflow/operational/runtime/readiness/locations, diff/impact/admission; tupla própria; policyEdit.readBack separa confirmed/drift/unverified/recovery_required do commit |
| Binding/Profile / G-2 | `internal/runtimeapplication/authoring.go`, Runtime skill; list/show/validate-candidate e autoria local | Snapshot local e candidate/preview digests; expected-revision/preview-digest/authorize-local; shared local scope; list/show read-back; suporte/auth não inferidos |
| Workflow / definição e seleção | Workflow skill, `internal/cli/skill_inspect.go`, `internal/projectapp/policy_workflow.go` | Definition list/show/validate/configuration.readiness; Project workflow.select separado com source/revision/digest e authority própria; seleção/revisão conferidas após write |
| Readiness / evaluator #231 | `internal/projectapp/readiness.go`, `internal/cli/readiness.go`, `internal/projectapp/policy_preview.go` | work-item/execution por operação; leitura não concede authority. G-1 usa `read_only_metadata_no_new_runtime_proof`; ausência de auth/proof material restringe conclusão |
| Auth / proof / produtores | `internal/runtimeadapter/authpreflight.go`, `internal/cli/runtime_capability.go`, `internal/runtimeadapter/runtime-versions.json` | Registry entries vazio não prova combinação. Auth status executa processos. Capability show pode exigir esforço/authority. Não executar prepare/probe/invalidate/login/inferência como descoberta; instalação/suporte/auth/proof separados |
| Work Item / Execution | Work Item skill, catálogo | Handoff só com refs/unknown, sem create/link/start/resume/run/stage dispatch/rebind. Backlog Linear não demonstra adapter instalado |
| Apresentação / AXM-32 | PR #328 OPEN/Draft, head `aba1c4d843e4b44a891531831aec2c79e95aae18`, mergedAt null; ADR-0030 lida nesse SHA | Entrega/par compatível não demonstrados. T06 BLOCKED; relay Markdown/Auxiliary atual preservado, sem cherry-pick, renderer ou mudança wire |

Fontes/ADRs relevantes confrontados: ADR-0004 (portátil/local), ADR-0020
(binding imutável), ADR-0022/0023 (donos), ADR-0024 e nota pre-stable
(effort/Profile), ADR-0025 (proof), ADR-0030 no dono da apresentação, Spec 009 e
Plan/Tasks aceitos. Nenhuma mudança de contrato/ADR proposta. Frases stale de
indisponibilidade G-1/G-2 e dispatch/delivery nas skills foram reconciliadas com
o catálogo de fonte, sempre condicionadas à instalação efetiva. Presença na
fonte não é evidência instalada.

### E02–E04 — orientação implementada

- Project agora compõe inventário, caminho completo conhecido e parada antes de
  modelo/effort/ID/template; registra inaccessible como não observado, distingue
  estados da policy e mantém Repository/remote/binding separados de Project.
- Project/Runtime orientam opções com fonte, campo aberto versus catálogo,
  candidato ID validado sem seleção unilateral, uma decisão material por pergunta,
  reuso de decisões válidas e opt-in parcial antes da coleta técnica.
- Cada dono mantém preview/authority por apply, write set/alcance e dependências;
  todos reobservam fatos materiais, invalidam preview/authority em drift e param
  cadeia incerta após read-back. Workflow compatível precede policy se necessário;
  nenhuma ordem fixa impossível, seleção/default implícito ou rebind.
- Runtime recebeu limites cumulativos AO-06/07/09 e checkpoint compatíveis com os
  três donos que já os tinham. Tetos não concedem efeito; deadline/correlação/
  attempt/contadores sobrevivem sessão, com parada antes sem progresso.
- Handoff Work Item mantém resultado/readiness/certeza/aceite e refs/unknown;
  nenhuma Execution ou mutação de Provider deriva de ready/partial/blocked.

Isto é implementação de **instruções distribuídas**. Assertions textuais comprovam
presença do contrato, não observação das decisões da IA. E02/E03/E04 comportamentais
(perguntas/eventos, previews reais, authorities por efeito e contadores em conversa)
continuam NOT_RUN. Não há snapshots/authority de Project real a publicar.

### E05 — testes e cenários por camada

Novo `TestGuidedConfigurationInstructionContracts` em
`internal/codexruntime/workflow_skill_content_test.go` cobre C01–C16 e invariantes
entre os quatro donos. Suites existentes de CLI verificam routing/catálogo/parser,
exclusividade de ownership e ausência de authority implícita; não implementam
resolver semântico. Novas entradas conversacionais e variantes estão em
[scenarios.md](../../../internal/codexruntime/testdata/guided-configuration/scenarios.md).
Formas e limites vêm dos produtores de E01; labels de snapshot são dados
sintéticos, nunca digests de wire nem novo schema de produto.

A primeira execução focada detectou duas obrigações reais de compatibilidade:
registro da revisão embedded substituída e fixture pré-AO derivado do texto atual.
Correção: append-only sharedSkillHistory e pin do novo manifest; fixture pré-AO
congelado com os **mesmos hashes históricos** verificados pelo teste, sem relaxar
nenhuma assertion. [Proveniência do fixture](../../../internal/codexruntime/testdata/pre-operator-autonomy/PROVENANCE.md).
Históricos/receipts/corpus existentes intactos. Manifest do novo conjunto:
`73e9fa8d6750660c7abaa1da0dee9b38b1f4d7c7e790ce7819d3cd48e60e02dd`;
skill-set/binary compatibility continuam 2/2 pelo contrato existente.

| Cenário | Contrato textual | Regressão determinística pertinente do dono | Conversa synthetic / nativa instalada / real |
| --- | --- | --- | --- |
| C01 | PASS | Readiness + instalação/reinstalação das quatro skills | NOT_RUN / NOT_RUN / não aplicável |
| C02 | PASS | Catálogo/routing/selector e policy identity drift | NOT_RUN / NOT_RUN / não aplicável |
| C03 | PASS | Configure/parser/Project readiness | NOT_RUN / NOT_RUN / não aplicável |
| C04 | PASS | Policy ausente/unconfigured e blockers canônicos | NOT_RUN / NOT_RUN / não aplicável |
| C05 | PASS | Catálogo/parser; não demonstra escolha semântica | NOT_RUN / NOT_RUN / não aplicável |
| C06 | PASS | Local authoring e PolicyV3DeclarationIsNotCapabilityProof | NOT_RUN / NOT_RUN / não aplicável |
| C07 | PASS | RoutingAloneNeverGrantsAuthority, authoring preview e policy exact apply | NOT_RUN por cada tipo de apply / NOT_RUN / não aplicável |
| C08 | PASS | PolicyPublicationRejectsEachStaleDependency, identity/content drift, AuthoringStaleCandidateAndNoOpZeroWrites | NOT_RUN por dimensão conversacional / NOT_RUN / não aplicável |
| C09 | PASS | PolicyReadBackTruthAndDrift, PolicyApplyTruthfulRecovery, AuthoringPublicationUncertaintyNotClaimedApplied | NOT_RUN / NOT_RUN / não aplicável |
| C10 | PASS | Contrato AO cumulativo; relógio/contadores de decisão da IA não observados | NOT_RUN / NOT_RUN / não aplicável |
| C11 | PASS | ExplicitOperationRoutingIsDeterministic, inspeção/catálogo | NOT_RUN / NOT_RUN / não aplicável |
| C12 | PASS | PolicyApplyExactReplayAndNoop e readiness; não prova caminho ready real | NOT_RUN synthetic / NOT_RUN / NOT_RUN real |
| C13 | PASS | PolicyPublicationGuardsAndPreservesLocalAbsence, readiness snapshots | NOT_RUN / NOT_RUN / não aplicável |
| C14 | PASS | PolicyWorkflowMissingTamperedAndReselected e Workflow binding | NOT_RUN / NOT_RUN / não aplicável |
| C15 | PASS | PolicyDraftStrictRejections, AuthoringLegacyDTOCanariesRedacted | NOT_RUN por draft/erro/Provider / NOT_RUN / não aplicável |
| C16 | PASS | AuthoringValidateCandidateValidatesSuppliedIntent; conversa/candidato não observados | NOT_RUN / NOT_RUN / não aplicável |

A coluna de regressão identifica controles existentes, **não** cobertura integral
conversacional do caso. Nem testes de strings nem green unit tests demonstram
ordem de perguntas, budgets da IA, zero comando injetado ou paridade semântica.
Filtros: pacote codexruntime/cli inteiro porque texto embedded altera instalação,
manifest/history, catálogo e routing; regressões dos donos executadas pelas suites
Go, sem mudar sua semântica para acomodar a jornada.

Tentativas sintéticas isoladas, sem instalação no operador, com orientações
embedded e fixture em stdin (não teste de par binário/skills instalado):

- `codex --version`: `codex-cli 0.162.1`; ajuda de exec inspecionada.
  `codex exec --ignore-user-config --ignore-rules --ephemeral --sandbox read-only
  --skip-git-repo-check --cd <scratch> --output-last-message <scratch-reply> -`:
  timeout local 55 s, nenhuma resposta final nem output stdout avaliável. Não há
  trace completo que comprove comportamento/eventos. Nenhum retry realizado.
- `claude --version`: `2.1.302`; ajuda inspecionada.
  `claude --bare --print --tools '' --no-session-persistence --strict-mcp-config
  --mcp-config '{"mcpServers":{}}'`, cwd scratch e stdin fixture: exit 1, sessão
  OAuth expirada sem refresh possível. Nenhum caso alcançou decisão avaliável;
  nenhum login/remediação/configuração de credenciais executado.

Invocações de avaliação **FAIL por ambiente/timeout**, casos **NOT_RUN**. Não são
findings comportamentais PASS ou FAIL. O pedido ao subject proibiu tools/efeitos,
mas não se atribui prova de confinamento ou zero ações a um trace ausente. Apenas
Claude teve tools desabilitadas por flag; sandbox Codex não equivale a tools off.
C15 canário foi inserido apenas no input efêmero; nenhum resultado de caso foi
obtido e não se declara PASS de não divulgação. Logs brutos não publicados.
SHA-256 do input efêmero conjunto (guidance + fixtures + canário sintético):
`24512c2742ab647ae69f928cfe86361bbe9e9d55017a9ff6f09fca9f9a3072b7`.
Novas observações exigem host funcional e trace de eventos verificável, sem
repetição automática de tentativa incerta.

### E06–E08 — gates externos e observações ausentes

T06/G2: #328 segue OPEN/Draft no SHA E01; contrato de apresentação ainda não
entregue como par compatível. Consumo/migração ativados **NOT_RUN/BLOCKED**;
nenhuma omissão material ou authority nova foi aceita por antecipação. Revalidar
entrega do dono e repetir C01/04/07/09/12/13 antes de alegar integração.

T07/G3: regressão de instalação em roots temporários é teste determinístico,
não sessão nativa com orientação instalada. T06 pendente e nenhuma sessão nativa
avaliável impedem concluir T07. Natural/explícita, hosts/alvos cruzados, rollback
para par anterior conhecido e comparação semântica C01–C16 permanecem NOT_RUN.
Não instalar/atualizar no operador para contornar isso. Rollback da orientação
não desfaz configuração; inversão suportada exige nova authority/read-back.

T08/G4: C12 real **BLOCKED/NOT_RUN**. Não há binário Axiom efetivamente instalado
observado, par entregue AXM-32, alvo real com Repository/Profile/Workflow
verificados, auth/proof exatos/frescos nem autorização humana específica do
preview/efeito. O prompt implement não concede essa última authority. Não
produzir proof/probe ou iniciar Execution para fabricar pré-requisitos. Menor
próxima ação: dono disponibilizar par compatível; depois operador indicar ambiente
adequado e observações válidas para preflight read-only. Somente após preview
real pedir authority exata. AXM-34 e AXM-12 retêm seus gates/aceites próprios.

### E09 — matriz de entrega e checkpoint

| Task | Resultado factual / critérios | Testes / Evidence / estado |
| --- | --- | --- |
| T01 | Contratos/fontes/limites e gates mapeados; lacunas localizadas | E01; DONE para inspeção de fonte, instalação real não alegada |
| T02 | Inventário/preflight e parada cedo implementados em Project | E02, C01–04/06/11/14 textuais PASS; comportamento NOT_RUN; IMPLEMENTED, validação incompleta |
| T03 | Decisões/opções/partial opt-in e preview/authority exatos em Project/Runtime | E03, C05–08/12/13/16 textuais PASS; comportamento NOT_RUN; IMPLEMENTED, validação incompleta |
| T04 | Workflow/stale/read-back/budgets/handoff nos quatro donos | E04, C07–10/13–15 textuais PASS; comportamento NOT_RUN; IMPLEMENTED, validação incompleta |
| T05 | Contratos C01–C16 e fixtures/variantes, regressão; tentativas de conversa sem resultado | E05; PARTIAL, não concluída sem observações sintéticas |
| T06 | Relay atual preservado; gate AXM-32 revalidado | E06; BLOCKED G2, integração NOT_RUN |
| T07 | Regressão determinística separada da paridade instalada | E07; BLOCKED T06/G3, paridade/rollback nativos NOT_RUN |
| T08 | Pré-requisitos/authority real não observados | E08; BLOCKED G4, C12 real NOT_RUN |
| T09 | Evidence incremental, revisão e handoff de bloqueio | E09; PARTIAL/handoff, conclusão formal depende T07; review independente/aceite pendentes |

GC-01: T01/T02 + inventário e C01/02/03/08/11/14/16;
GC-02: T02 + preflight antes de P3 e C03–06/10–14;
GC-03: T03 + fontes/decisão única e C01/04–06/16;
GC-04: T03/T04 + tupla por efeito/denial/stale e C02/03/07–09/11–16;
GC-05: T04 + read-back/readiness/síntese e C01/03/04/06/07/09/12/13;
GC-06: T02/T04 + limites/checkpoint/injeção e C02/09/10/15.
Todos têm contrato textual verificado; observação da jornada inteira continua
pendente. Nenhum GC é declarado aceito por essas assertions.

Checkpoint de retomada: objetivo é completar T01–T09 aceitas no PR #329;
autoridade permanece limitada ao prompt implement, sem merge/release/aceite.
Não existe alvo/attempt/budget de jornada real iniciado por este trabalho;
Execution/gate unknown, efeitos de configuração confirmados/incertos nenhum
observado pelo executor. Tentativas sintéticas encerradas sem resultado, sem
replay automático; nenhuma authority real persistida. Revalidar head/PR e G2
antes da próxima unidade. T05 precisa validate/conversa synthetic com trace;
T06 exige entrega AXM-32; T07 exige T06 e ambientes/instalação de teste autorizados;
T08 exige T07, observações exatas e nova authority por efeito. Não resetar budgets
de uma futura jornada a partir deste checkpoint de desenvolvimento.

Validação do conteúdo entregue:

| Check / comando | Resultado e alcance |
| --- | --- |
| `go test ./internal/codexruntime ./internal/cli` | PASS após correções de histórico/fixture; contratos C01–C16 e install/reinstall/upgrade nos dois destinos, sem provar conversa |
| `go test ./...` | PASS, incluindo cmd/lingo (172.194 s), local, projectapp, runtimeapplication, Workflow, install, compatibility e todos os demais pacotes; nenhuma suíte omitida |
| `go vet ./...` | PASS |
| `./scripts/validate-repository.sh .` | PASS: adapters/harness, automation (91 surfaces), ADRs (30), sensitive files e whitespace; cenários nativos do maintainer validator explicitamente SKIP/UNVERIFIED |
| `git diff --check` | PASS |
| Scanner consolidado `gitleaks` | NOT_RUN: comando não disponível; nenhuma instalação global realizada |
| Conversa synthetic Codex/Claude | NOT_RUN por caso; tentativas FAIL por timeout/auth, como E05 |
| T06 integração, T07 paridade instalada, T08 real C12 e AXM-12 E2E amplo | NOT_RUN; gates E06–E08, sem aceite derivado |

Review proporcional do Orchestrator com axiom-review: escopo confrontado com
GC/P/Tasks e ADRs, denial/stale/read-back, limites/auth/proof, superfícies de
paths/links/permissões, comentários novos, dependências e diff histórico.
Nenhum finding bloqueante de código/instrução identificado; lacunas de validação
comportamental e gates externos permanecem bloqueantes para **conclusão integral**,
não foram dispensadas. Nenhum comentário novo em Go de produção/teste, TODO/FIXME,
código comentado, dependência, comando/catálogo ou schema novo introduzido.
Remoção do comentário do fixture integra a alteração de sua fonte; não remove
comentários fora do diff. Self-review não é review independente nem aceite humano.
Checks finais de links/bytes/hygiene e publicação são registrados no checkpoint
abaixo e no comentário do head exato no PR/AXM, sem autoatribuir CI PASS.

Checkpoint final pré-publicação: verificação efêmera Python executada contra
`git show 6d504644:<artifact>` confirmou identidade byte a byte de Spec/Plan/Tasks
e prefixo append-only de Evidence; verificou fences Markdown, 114 links locais
nos paths alterados e ausência de linhas novas de comentário Go. PASS.
`git diff --check` PASS; árvore contém somente quatro skills, testes/histórico de
instalação e fixtures correspondentes, índice e este Evidence. Staged hygiene e
read-back de publicação/CI do SHA exato seguem no comentário bounded do mesmo PR.
CI ainda não observado para o commit futuro; não é PASS. Próximo gate é review
independente deste delta parcial e resolução dos blockers E05–E08 antes de
concluir Tasks ou aceitar produto. Sem merge/release/transição/aceite automático.


## Retomada r2 — 2026-10-18

### Authority, read-back e escopo incremental

Retomada explicitamente autorizada por Rafael em 18/10/2026 17:57 BRT,
com edição/testes/commit/push normal na mesma branch/PR #329; sem login/OAuth,
instalação, merge, force-push, aceite ou mudança de status/labels. Antes de editar,
foram lidos o Evidence integral e o
[handoff indicado](https://linear.app/rgomids/issue/AXM-33#comment-2b5f8e4a).
Somente referências e observações técnicas autorizadas para publicação são
registradas; nenhuma transcrição privada ou payload de Provider importado.

`git fetch origin`, `git log -5 --oneline`, ref remota e `gh pr view 329`
confirmaram HEAD local/remoto/PR `b8166be074520390ac5b9a27cec9201c0a56dfcb`,
árvore limpa e PR canônico OPEN/Draft, mesma branch e base main, sem commit novo.
CI desse SHA foi reobservado: PR Checks e delivery-metadata SUCCESS; isso não
herda PASS para o futuro commit r2. Linear AXM-33 permanece In Progress. Execution,
revisão/gates/estágio canônicos continuam unknown; nenhum segundo Execution criado.

T02–T04 já estavam implementadas. Não houve nova mudança de skill, Go, teste,
manifest/history, dependência ou automação. O delta r2 consiste em observações
comportamentais possíveis, Evidence append-only e reconciliação do índice.
Orchestrator executou validate/review/document; sem delegação de trabalho porque
inferioridade estrita de modelo não é comprovável. Subjects de avaliação são o
objeto do teste, não recebem trabalho de implementação/review ou authority. As
chamadas independentes de avaliação foram paralelizadas; diálogo causal foi
sequencial, branches após o prefixo comum. Biblioteca padrão e fixtures existentes
bastaram, sem infraestrutura genérica/runner durável ou nova biblioteca.

### E02–E05 r2 — observações agora avaliáveis

[Observations r2](observations-r2.md) contém método, variantes, esperado/observado,
recortes mínimos sanitizados, hashes dos inputs/replies e limites. Codex CLI
0.162.1 respondeu ao C04 isolado, aos seis batches C01–C16 (48 variantes × dois
contextos de entrada = 96 respostas) e a seis chamadas de diálogo sintético com
histórico reapresentado: escolha → preview → branches confirmed/denied/stale/
unverified. Todos exit 0, com trace completo e sem eventos de ferramentas/comandos.

Resultado: **PASS restrito às decisões observadas**. C07 cobre negativa por tipo
de apply; C08 cobre dez dimensões de drift; C09 cobre ack/readBack/drift; C10
preserva relógio/deadline e budgets fornecidos; C15 cobre draft/erro/Provider,
com ausência do canário nas respostas verificada deterministicamente. Perguntas,
opt-in, preservação de intenção, consentimento e síntese foram avaliados contra
GC-01–06; o diálogo distingue confirmação de conteúdo, commit e readiness.
Não se pede que o leitor confie em contadores autorrelatados: cada trace contém
somente início de thread/turno, mensagem do subject e fim do turno.

Não se declara execução real de apply, store, dispatch, auth ou proof. Read-only
sandbox não equivale a tools-off; zero chamadas é observação desses traces.
Snapshots/D0 são símbolos sintéticos e nunca foram submetidos a uma CLI Axiom.
As propostas de inspeção continuam condicionadas à metadata e pré-requisitos do
dono; não são comandos aprovados pela avaliação. Modelo efetivo não exposto pelo
trace, sem override/alegação de identidade. Entradas batch são snapshots de fases,
e o diálogo reapresenta histórico, não uma sessão nativa persistida.

Assim E02/E03/E04 deixam de ser inteiramente NOT_RUN **na camada sintética de
próxima decisão**; a matriz histórica acima continua válida para sua data. T05
avança, mas continua **PARTIAL** para a jornada integral com ferramentas/resultados
canônicos completos: não se comprovou coleta→draft tipado→preview literal→apply→
read-back em conversa instalada para cada variante. Essa limitação não se resolve
com mais assertions textuais. T07 conserva a validação instalada e a paridade.

Claude CLI 2.1.302: chamada C04 isolada, tools desabilitadas por flag, exit 1/OAuth
expirado novamente. Invocação **FAIL ambiente**; caso **NOT_RUN/BLOCKED** por
autenticação. Sem login, refresh, inspeção de credenciais ou retry automático.

### E06–E08 r2 — gates reobservados e tentativa da C12 real

`gh pr view 328` confirmou OPEN/Draft, mergedAt null, head ainda
`aba1c4d843e4b44a891531831aec2c79e95aae18`; AXM-32 consultada In Progress.
Entrega/par de apresentação compatível não demonstrados: **T06 BLOCKED G2**, sem
ativar migração/relay ou absorver renderer. **T07 BLOCKED T06/G3**; não instalar
no operador nem inferir paridade/rollback de testes de instalação em roots de teste.

C12 real foi tentada até o primeiro gate de observação passiva disponível:
`command -v`/`shutil.which('axiom')` não localizou Axiom no PATH. Nenhum caminho de
instalação alternativo foi fornecido. Não enumerar diretórios privados, instalar,
construir checkout e chamá-lo de instalação do usuário ou autenticar para superar
esse gate. A entrega de G-1 em fonte não demonstra operação efetivamente instalada.

| Pré-requisito C12 real | Observado r2 / menor próxima ação |
| --- | --- |
| T07 e par instalado compatível AXM-32 | Não demonstrados; dono deve entregar G2, depois instalação de teste sob autorização própria |
| Operação G-1 na instalação efetiva | Não observada; nenhum axiom no PATH; operador indicar instalação já existente ou autorizar instalação por processo próprio |
| Alvo real autorizado e Repository/Profile/Workflow correspondentes | Não fornecidos; indicar alvo e permitir inspeções canônicas read-only disponíveis |
| Auth/proof exatos, frescos e produtor quando necessário | Não observados para alvo real; autenticação do host Codex não é proof/auth do alvo; Claude expirado não foi remediado |
| Preview real e authority específica por efeito | Não existem nesta observação; solicitar authority somente após preview real verificável |

**T08 BLOCKED G4; C12 real NOT_RUN.** C12 synthetic PASS limitado não substitui
nenhum desses fatos. Nenhuma tentativa real de apply/dispatch ou efeito de
configuração incerto iniciado pelo executor. O prompt implement/retomada autoriza
o trabalho de repositório, não a publicação de um futuro preview de configuração.

### E09 r2 — matriz factual e próximo gate

| Task | Estado factual r2 | Evidence / lacuna restante |
| --- | --- | --- |
| T01 | DONE inspeção de fonte/contratos, read-back remoto renovado | E01 anterior; instalação real não alegada |
| T02 | IMPLEMENTED; decisões sintéticas pertinentes PASS limitado | E02 r2/C01–04/06/11/14; jornada instalada incompleta |
| T03 | IMPLEMENTED; decisões sintéticas pertinentes PASS limitado | E03 r2/C05–08/12/13/16 e diálogo escolha/preview; efeitos canônicos em conversa incompletos |
| T04 | IMPLEMENTED; decisões sintéticas pertinentes PASS limitado | E04 r2/C07–10/13–15 e branches; integração/continuidade nativa incompletas |
| T05 | PARTIAL, avanço de NOT_RUN para observações avaliáveis Codex | E05 r2/96 respostas e diálogo; interação integral canônica e ambiente Claude não demonstrados |
| T06 | BLOCKED G2 | E06 r2, AXM-32/#328 não entregue como par compatível |
| T07 | BLOCKED T06/G3 | E07 r2, paridade instalada/alvos cruzados/rollback NOT_RUN; instalação e auth não autorizadas |
| T08 | BLOCKED G4 | E08 r2, C12 real NOT_RUN pelos pré-requisitos acima |
| T09 | PARTIAL, handoff e revisão incremental entregues | E09 r2, conclusão formal ainda depende T07; review independente/aceite pendentes |

GC-01/02/03/04/05/06 receberam observações sintéticas de decisões conforme a
matriz de Observations r2. Nenhum GC é declarado integralmente aceito. Sem novo
finding bloqueante de instrução identificado pelo self-review proporcional; sem
alteração de código para acomodar teste. Limites de observação permanecem riscos
bloqueantes para conclusão integral. ADRs/Spec/Plan/Tasks e fronteiras dos donos
preservados; review independente não substituído por este relato.

Checkpoint: continuar o mesmo objetivo T01–T09/PR #329, sem recriar execução ou
refazer implementação existente. Próxima unidade depende do dono AXM-32 entregar
contrato/par compatível; depois, ambiente de teste instalado e funcional autorizado
para observar ferramentas canônicas e paridade. Para Claude, autenticação válida
precisa ser providenciada pelo operador sob processo próprio. Para C12 real, seguir
o gate E08 acima, sem pedir confirmação genérica antecipada ou simular proof.
Nenhum attempt/budget de jornada real iniciado; contadores do roteiro sintético
não são estado do operador, e a retomada não concede replay de efeito incerto.


Validação r2 do delta documental e da orientação inalterada:

| Check realmente executado | Resultado / alcance |
| --- | --- |
| `go test ./internal/codexruntime ./internal/cli` | PASS (cache válido); contrato embedded, catálogo/routing e instalação/reinstalação de teste |
| `go test ./internal/projectapp ./internal/runtimeapplication ./internal/local ./internal/compatibility` | PASS (cache válido); regressões de policy, autoria local, persistência/compatibilidade, sem mudar testes dos donos |
| `./scripts/validate-repository.sh .` | PASS; 91 superfícies de automação, 30 ADRs e hygiene; cenários nativos do maintainer validator SKIP/UNVERIFIED |
| Check efêmero Python de bytes/links | PASS; Spec/Plan/Tasks idênticos ao SHA aceito, prefixo Evidence de b8166be intacto, fences e 119 links/âncoras locais nos três documentos r2 |
| Trace/IDs/canário dos batches e diálogo | PASS no alcance de Observations r2; 96 IDs únicos/completos, zero tool events e canário ausente nas respostas |
| `git diff --check` | PASS |
| Scanner consolidado gitleaks | NOT_RUN: não disponível, nenhuma instalação realizada |
| `go test ./...` / `go vet ./...` nesta retomada | NOT_RUN novamente: nenhum Go/skill/dependência alterado; resultado local anterior e CI SUCCESS de b8166be preservados com seu SHA, não transferidos ao commit novo |
| Claude synthetic / T06 / T07 / C12 real | NOT_RUN/BLOCKED conforme E05–E08 r2, sem atribuir PASS por CI |

Revisão de engenharia/segurança do delta: apenas três documentos, sem mudança de
contrato/ADR, effect authority, permissões, scripts ou código/comentários Go.
Observações são produzidas com fontes públicas do checkout e fixtures sintéticas;
sem canário, logs de auth, paths privados ou payload de Provider publicados. Nenhum
finding bloqueante novo identificado; não é review independente nem aceite.
Staged hygiene, metadata do PR, commit/push/head e CI do SHA novo são confirmados
no comentário de publicação no PR/AXM; CI pendente não é PASS. O gate seguinte é
review independente deste delta limitado e resolução dos bloqueios registrados,
sem conclusão integral de Tasks ou transição automática.
