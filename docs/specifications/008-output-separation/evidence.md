# AXM-32 — Specify Evidence

## Original specify investigation — historical baseline

Documentation-only investigation on 2026-10-18, baseline
`14cb721ac7b8ce09a7f20f2bc2175adf61b7934b` (v0.19.0).
[Specification](spec.md) and
[ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md)
remain Proposed. Evidence here proves source inspection/document integrity only.
No code, tests, adapters, installed skills, operational state or authority changed.

Initial checkout was clean and detached at the baseline; `git ls-remote origin
refs/heads/main` matched. Both open PRs were inspected by title, body and branch
with `gh pr list --repo rgomids/axiom --state open --limit 100 --json
number,title,headRefName,body`: #317 and #327 were unrelated to AXM-32.
Search results containing #327 were not assumed to identify an AXM-32 execution.
`git branch -a` and `git ls-remote --heads origin '*axm-32*' '*AXM-32*'`
showed no AXM-32 branch. Created `docs/axm-32-output-separation-specification`.
Recheck before opening a single draft PR; no empty PR/commit is authorized.

`gh repo view rgomids/axiom --json visibility` confirmed PUBLIC. Only public
repository sources and sanitized user-supplied scope inform these artifacts;
no private Linear/Notion contents or raw personal test data were imported.
The supplied Linear/gate facts were not independently revalidated. No canonical
Execution was observable, reconstructed or advanced; other-host concurrency
cannot be proven absent from a provider PR search.

## Confirmed source observations

- `gh issue view 232 --repo rgomids/axiom --json title,state,url,body` confirms
  closed/accepted history, human decision dated 2026-10-09 and stable v0.13.0.
  `gh pr view 291 --repo rgomids/axiom --json state,mergeCommit,mergedAt`
  confirms merge `2d7a7cdf24feb4312321e44755657d4e0b11b246`,
  2026-10-09T22:51:14Z. Historical Evidence is unchanged.
- `gh pr view 325 --repo rgomids/axiom --json state,mergedAt,mergeCommit,title,body,url`
  confirms MERGED, 2026-10-18T01:57:12Z,
  `5b6bd0b354ec38794be33fd8847333bf5132049f`.
  The integrated files, not the PR narrative alone, confirm exact relay and
  separate Auxiliary explanation. Existing Specification/Plan acceptance records
  remain preserved; AXM-31's synthetic evidence is not native-host proof.
- `gh issue view 305 --repo rgomids/axiom --json title,state,url,body` confirms
  the separate open Project Runtime/Profile policy-editing gap. This proposal
  neither implements nor masks it and does not change dependencies.
- Source inspection of `internal/cli/{cli,completion,presentation,readiness}.go`
  confirms leading output flags, default human mode, exact Markdown over canonical
  JSON, fallback dumps, separate readiness payload and existing exit mapping.
- `cmd/lingo/project_lifecycle.go` returns successful structural validation when
  `report.Structure` is valid and includes the readiness report.
  `cmd/lingo/bootstrap_readiness_test.go` contains an existing synthetic case with
  successful validation, effective partial, Work Item ready and Execution blocked
  by `runtime_policy_unavailable`. Read as source, **not executed here**. This does
  not prove the reported personal Project had those exact facts.
- Four `internal/codexruntime/skills/*/SKILL.md` instruction sources and their
  routing/content/history tests were inspected. Shared Runtime integration tests
  cover both Codex and Claude; no separate `internal/clauderuntime` package exists
  in this checkout. Future native-host behavior remains unverified.
- Consulted constitution, contribution/security/architecture/quality/documentation
  and automation policies, contribution/PR templates, commands/tracking reference,
  Specification 004/#232 Evidence, Specification 007 and AXM-31 amendment/Plan,
  ADR-0003/0006/0018/0022/0025. Exact conflict mapping is in the Specification.

## Original documentary checks — historical baseline

No future behavior acceptance is implied by these checks. Link validation is
scoped to new artifacts and added
index links; public issue/PR references are verified with the read-only GitHub
queries above. No external website crawling or speculative capability lookup is
required. One-off link checking runs outside the tracked tree.

| Check executed | Result and limit |
|---|---|
| `./scripts/validate-repository.sh .` | PASS: repository bootstrap, maintainer harness structure, Claude adapters, automation inventory, label policy, ADR structure and sensitive-file checks; native Runtime behavioral scenarios explicitly skipped by the validator |
| `python3 scripts/check-adr-governance.py .` | PASS: 31 ADRs, Proposed status and index reconciled; structure does not establish semantic acceptance |
| Scoped one-off Python link check | PASS: 44 local links including heading anchors and added index entries; target existence/heading normalization checked without a new versioned script |
| `git diff --check`, `git diff --cached --check`, `./scripts/check-sensitive-files.sh --staged .` | PASS: whitespace and staged sensitive-file scan; staged paths/content reviewed before commit |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN, no installation attempted; repository sensitive-file scanner remains applicable |

Review found only five intended Markdown paths: the Specification, this Evidence,
ADR proposal and their two indexes. No accepted specification/ADR/history,
commands reference, executable, tests, adapters, automation registry, dependencies
or installed skill bytes are modified. Any later commit/PR reference is carried
by Git/PR metadata; it is not fabricated inside its own commit.

## Review and limitations

Self-review checks source hierarchy, historical acceptance, absence of an accepted
ADR conflict, status/readiness separation, resolved D1–D3 choices, single-PR scope,
semantic matrix, secret/injection controls and public-data classification.
This is not independent review or human acceptance. D1–D3 are resolved choices;
R1 producer/consumer coverage remains unverified before Plan. Exact revised
Specification/ADR acceptance remains pending; no schema gap is claimed resolved.

**NOT_RUN:** AC1–AC9 implementation, terminal/golden/channel tests, JSON consumer
regressions, security canary tests, installed-skill upgrade/integration journeys,
native Codex/Claude conversations and AXM-12 E2E. They require future approved
contracts/implementation and, for real Runtime effects, separate authority.
No additional Go suite is needed to validate a specification-only change.

## D1–D3 specify refinement — 2026-10-18

The [GitHub human decision](https://github.com/rgomids/axiom/pull/328#issuecomment-6245936652)
(created 2026-10-18T05:06:23Z) and matching
[Linear human decision](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-43e5ffd1)
(created 2026-10-18T05:06:51Z) were read directly, together with AXM-32 and
[review #5561505065](https://github.com/rgomids/axiom/pull/328#pullrequestreview-5561505065).
The user explicitly authorized publication of these D1–D3 choices in this public
repository. Only their bounded decision content and links are incorporated;
unrelated Linear content and raw conversations are not imported. Original
investigation/validation observations above remain dated historical evidence.

Before edits, `gh pr view 328` confirmed OPEN/Draft, branch
`docs/axm-32-output-separation-specification`, head
`e59480d432403104fccdd7ba61f8f9b46752d3c2`.
`git fetch origin docs/axm-32-output-separation-specification` and `git rev-parse`
confirmed local HEAD and the fetched remote branch matched that reviewed head;
the checkout was clean, with no intervening remote edits to reconcile.
`gh pr list --state open` confirmed #328 as the sole open AXM-32 PR.
`gh repo view --json visibility` reconfirmed PUBLIC. The resulting revision is
identified by Git history and PR head metadata, avoiding a self-referential SHA.

Linear currently reports In Progress. This provider operational status neither
proves work start/completion nor projects a canonical Execution stage. No local
Execution was reconstructed or advanced; no provider lifecycle metadata changed.

Refinement reconciles native human CLI defaults in TTY/pipes and explicit JSON
(D1), coordinated replacement of default Markdown/literal relay/Auxiliary within
AXM-32 across binary and all four installed Codex/Claude skills (D2), and current
typed wire-contract preservation without speculative schema evolution (D3).
The fact-by-fact R1 producer/consumer inventory is still outstanding before Plan;
proven material gaps require explicit proposal review/acceptance with affected
consumers/tests before code changes. No JSON coverage or implementation is claimed.

Only the Specification, Proposed ADR, this Evidence and their two stale index
summaries are revised. PR #328's stale decision/lifecycle description is reconciled
in place. Accepted documents, historical Evidence files, executable code, tests,
scripts and installed skills are untouched. The ADR remains Proposed; no formal
supersession or contract acceptance is inferred. All AC1–AC9 remain NOT_RUN.

No subagents were spawned: the edits are coupled and no model strictly inferior
to the Orchestrator was verifiable. Deterministic validation and self-review are
performed by the Orchestrator; no independent review is claimed.

### Refinement validation

Results below concern documentary integrity only; they cannot establish behavioral
acceptance, native-host fidelity, final human approval or merge readiness.

| Check executed for refinement | Result and limit |
|---|---|
| `./scripts/validate-repository.sh .` | PASS; includes harness/adapter structure, automation registry, ADR/label/security checker regressions and worktree scan. Native Runtime behavior explicitly skipped/UNVERIFIED |
| `python3 scripts/check-adr-governance.py .` | PASS, 31 ADRs; structural status/index reconciliation only |
| Scoped one-off Python link/anchor check in `/tmp` | PASS, 43 local links/anchors across the three artifacts and revised index references; no tracked script added |
| `git diff --check`, `git diff --cached --check` | PASS; staged paths/content reviewed for only five authorized Markdown files |
| `./scripts/check-sensitive-files.sh --staged .` | PASS; no secrets/private unrelated content identified in documentary self-review |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN, no dependency installed |

Self-review found no blocking documentary finding: resolved choices, flags/pipes,
no extra formatting LLM call, authority/status, migration/rollback, historical
acceptance and scope remain coherent. These checks do not claim JSON R1 coverage,
AC1–AC9 execution, native smoke, implementation completion or independent review.
Explicit human re-review/acceptance of the resulting Specification/ADR revision
is still required; the same PR remains Draft.

## Clarify investigation — 2026-10-18

This section supersedes earlier pending-Spec-acceptance observations only as a
current authority record; original dated specify Evidence remains historical.
[Specification 008 was explicitly accepted](https://github.com/rgomids/axiom/pull/328#issuecomment-6246060142)
at `18d29a6c5446357defb2563483cfac180dcc4043`. Its bytes and ADR-0030's bytes
are unchanged by this clarification. The ADR remains Proposed, Git blob
`295084e6b6afa054b2f07dcaf5e73563ed7e7644`. D1–D3 are not reopened.
[Clarify inventory](clarify.md) owns the producer/emitter/consumer table, required
R1 fact matrix, G1/G2 and bounded separate proposals, and the exact pending ADR
question. Outcome: **NEEDS_HUMAN_DECISION**, not READY_FOR_plan.

Before editing, `git status --short` was empty; branch was
`docs/axm-32-output-separation-specification`, HEAD was the accepted SHA above.
`gh pr view 328 --repo rgomids/axiom --json state,isDraft,headRefName,headRefOid,body,comments,reviews,statusCheckRollup,url`
confirmed OPEN/Draft and matching head. Comments verified explicit Spec-only
acceptance, separate ADR authority, D1–D3 and round-two review. `gh pr list
--repo rgomids/axiom --state open --limit 100 --json number,title,headRefName,body`
showed #328 as the only AXM-32 PR; #317 and #327 are unrelated. `git ls-remote
origin refs/heads/docs/axm-32-output-separation-specification` matched before
editing and before publication preparation. No concurrent remote edit was seen;
this does not prove absence of an external canonical Execution.

Read-only Linear issue/comment queries directly confirmed the supplied
[Spec-only acceptance](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-efd06110)
and [READY_FOR_clarify gate](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-6eaf4304).
Operational In Progress and no Blocked by do not reveal canonical Execution
identity/revision/gate. No provider state or canonical transition is written.
Only bounded user-authorized decisions/links are incorporated; no unrelated
private Linear text or personal incident data is imported. `gh repo view
rgomids/axiom --json visibility` reconfirmed PUBLIC.

### Inspection and documentary validation

| Executed check / observation | Actual result and limit |
|---|---|
| Source/DTO trace at accepted SHA | G1: `reconcileAttempt` returns `provider_create_ambiguous`; `workItemResult` retains Category but the no-payload common completion drops it. Comment ambiguity was separately traced through `applyChange`: ChangePreview keeps category; it is not a G1 example. G2: `boundedTransfer` slices three full lists before serialization at 64. Tests named in clarify were read, not executed. |
| One-off Python static assertions, outside tracked tree | PASS: exact G1 return/category-carrier/common-DTO/fallback snippets and G2 cap/slice snippets checked against source. This checks source premises, not an end-to-end mutation or a >64-object transfer. |
| `./scripts/validate-repository.sh .` | PASS: bootstrap/harness/Claude adapter/automation/label/ADR/sensitive-file validation and checker regressions; native Runtime behavioral scenarios explicitly SKIP/UNVERIFIED. |
| `python3 scripts/check-adr-governance.py .` | PASS, 31 ADRs; structural only, no semantic/human acceptance implied. |
| One-off local documentary link/anchor check | PASS, 41 scoped local links/anchors; clarify links, new Evidence cross-links and updated AXM-32 index links checked for targets/anchors. No durable script or automation added. |
| `git diff --check`, staged diff and sensitive-file check | PASS for working/staged whitespace and staged sensitive-file scan; staged paths/content reviewed. Only clarify, this appended Evidence and the Specification index changed. |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN, no installation attempted. Repository scanner and content classification review remain applicable. |
| Byte identity check against `git show 18d29a6:<path>` | PASS: approved Spec and Proposed ADR unchanged byte-for-byte. |
| `gh pr checks 328 --repo rgomids/axiom` before edits | PR Checks and delivery-metadata PASS on `18d29a6`; these are not results for the future clarify commit. Publication SHA/checks are reported in the same PR metadata. |

Self-review reconciled the constitution, documentation/security/contribution
policies, #232 acceptance, AXM-31 diagnostic/relay contract and ADR-0018 with
ADR-0030. No accepted ADR fragment requiring supersession was identified. The
Spec is untouched; the index now records its already granted acceptance separately
from the Proposed ADR. No independent review, schema acceptance or implementation
readiness is claimed. No subagents: no strict model inferiority proof was available.

**NOT_RUN:** Go suites, AC1–AC9 implementation, changed renderers/handlers/DTOs,
JSON consumer regressions, security canaries, installed-skill upgrade/integration,
real >64-object transfer, native Codex/Claude conversations and AXM-12 E2E.
No production skills, schemas, code, tests, Provider/Runtime execution, release,
merge or lifecycle metadata changed. Unknown concrete payload variants and external
consumers remain explicit in clarify. ADR acceptance and separate gap decisions,
with sufficient deterministic coverage, precede any subsequent Plan gate.

## Plan preparation — 2026-10-18

[Plan](plan.md) is **Proposed for review**, not accepted. This dated section
supersedes only older pending-ADR/pre-Plan-hold statements as current authority
context. Specification 008 acceptance at `18d29a6` and the subsequent
[ADR acceptance / G1–G2 watch decision](https://github.com/rgomids/axiom/pull/328#issuecomment-6250321308)
were read directly through GitHub. Read-only Linear issue/comment queries confirmed
[READY_FOR_plan](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-9e16edb9),
the later decision, In Progress and no Blocked by. These observations neither
identify a canonical Execution nor prove its absence; none is created or advanced.
Only already public/user-authorized decisions and links are incorporated here.

### Baseline and scope

Initial clean checkout was `1d26b0777b787c796945331e9a55d7946896e451` on
`docs/axm-32-output-separation-specification`. `gh pr view 328` and `git fetch`
observed remote `d4db1c3fdf625c6263751593add5ba8719fdd9b3`; `git merge --ff-only
FETCH_HEAD` incorporated the later acceptance/operational annotations without
rewriting them. The checkout has no remote-tracking ref for this branch, so
subsequent comparisons use fetched HEAD / `git ls-remote`, not an assumed
`origin/<branch>` ref. `gh pr list --head ... --state all` and a bounded open-PR
AXM-32 search showed only #328, OPEN/Draft. `gh repo view --json visibility`
confirmed PUBLIC. Before writing, `git ls-remote` still matched `d4db1c3`.

Read SDD/orchestration/review skills, security/documentation policies,
Documentation Governance, constitution, Linear/GitHub governance, existing Plan
format, Spec/ADR and clarify inventory. Spot-checked `emitResponse`,
`parseOutputMode`, `Run` vs `RunInteractive`, canonical completion/readiness DTOs,
`presentEvent` / Markdown fallback, all four skill consumption obligations and
shared Codex/Claude embedding, manifest/history/receipts/upgrade boundaries.
No new product capability or host invocation interface was inferred.

No delegation: strict model inferiority was not provable and this bounded
investigation could be integrated locally. Explore/plan/review/validate stayed
with the Orchestrator; no independent review is claimed. Documentary self-review
found no blocking contradiction with Spec 008 or Accepted ADR-0030; the draft
corrected a source-function reference to `parseOutputMode`. G1/G2 are observable
risks only, without speculative correction or waiver of essential safety checks.

### Documentary validation and limits

| Executed check | Result / scope |
|---|---|
| `./scripts/validate-repository.sh .` | PASS: repository/harness/adapters/automation/label/ADR/sensitive-file checks and validator regressions. Native Runtime behavioral scenarios explicitly SKIP/UNVERIFIED. No product behavioral suite was invoked. |
| `python3 scripts/check-adr-governance.py .` | PASS, 31 ADRs; structural validation, not semantic or human acceptance. |
| Scoped one-off Python local link/anchor and source-path checks | Checks the Plan and appended plan Evidence targets plus named concrete source/test paths; no durable automation added. Results recorded with publication checks below. |
| `git diff --check` | PASS on the documentary worktree; staged whitespace/content/classification and sensitive-file review required again before commit. |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN; no dependency installed. Repository sensitive-file checks and manual public-content review remain applicable. |
| `gh pr view 328 --json statusCheckRollup` at baseline `d4db1c3` | PR Checks and both delivery-metadata observations SUCCESS. These results apply only to that pre-Plan SHA, not to the future Plan commit. |

**NOT_RUN:** implementation, Go product suites, AC1–AC9 behavioral/equivalence/
security-canary tests, installed-skill upgrade/rollback smoke, prepared-artifact
smoke, native Codex/Claude conversations, actual G1/G2 incident reproduction,
Provider/Runtime effects and AXM-12 E2E. Future test surfaces and bounded semantic
oracle are a strategy, not delivered tests or complete natural-language coverage.
No code, production skills/tests, accepted Spec/ADR, persisted state, Linear/status
labels, Execution lifecycle, merge or release is changed. Only Plan and this
appended Evidence are versioned in this stage. `tasks` is a candidate next stage
only after separate review/acceptance of the exact Plan and its own authority/gate.

Publication preparation: one-off checks passed for **6** scoped local links/anchors
and **21** explicit source paths. Byte comparison passed for Spec against accepted
`18d29a6`, and Spec/ADR/clarify against `d4db1c3`. A fresh `gh pr view` and
`git ls-remote` still reported OPEN/Draft #328 at `d4db1c3`; no concurrent delta
was observed. `./scripts/delivery-issues.sh check-pr --body <temporary-body>
--title <temporary-title>` passed (`metadata=declared`, related/completes `none`).
Staged review confirmed only the two authorized documentary paths, with no
private/raw provider data or secrets. `git diff --cached --check`
and `./scripts/check-sensitive-files.sh --staged .` passed. Remote Plan-commit CI
and published HEAD are recorded on the same PR after push; they are not inferred
from baseline results.

## Plan correction — R1 decision and index, 2026-10-18

### Authority, interpretation and round-3 response

Read the full [subsequent human R1 decision](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-f5c308fd)
(comment `f5c308fd-795c-44cc-a52a-0b7ba6a6ed84`) before editing. The user authorizes
publishing its bounded documentary reconciliation, not raw provider discussion.
It confirms that [Clarify](clarify.md)'s concrete inventory of producers, consumers,
facts and gaps satisfies ADR-0030's pre-Plan requirement **for planning only**.
This resolves the interpretation requested by [round 3](https://github.com/rgomids/axiom/pull/328#pullrequestreview-5563186345);
it does not prove every dynamic path, assert universally lossless JSON, waive
R1/AC1–AC9 or accept the [Plan](plan.md).

ADR-0030 D3 requires an inventory with concrete producer/consumer Evidence;
Clarify supplies that inventory and explicitly distinguishes `covered`,
`gap_confirmed` and `unverified`. Inventory sufficiency and universal behavioral
proof are different claims. No unavoidable contradiction with the accepted
Spec/ADR was identified; their behavioral obligations remain binding. This dated
section and Plan's updated gate context supersede only earlier pending-inventory
interpretations as current planning context, preserving historical Evidence and
Clarify bytes. The earlier G1/G2 decision alone was not treated as resolving R1.

| Round-3 item | Documentary response for renewed review |
|---|---|
| [R1 gate comment](https://github.com/rgomids/axiom/pull/328#discussion_r4307209020) | Plan links the later human decision and states inventory sufficiency only for planning, with gaps/dynamic variants unverified and future behavioral Evidence still required. |
| [CR-001 — Minor index comment](https://github.com/rgomids/axiom/pull/328#discussion_r4307209016) | Specifications index now records ADR-0030 Accepted (2026-10-18), its human acceptance and later R1 decision, and Plan Proposed. Obsolete future ADR/G1–G2 preconditions are removed; the decisions index already agrees and is unchanged. |

Both requests have documentary responses; no thread was resolved and no new
review or acceptance is claimed. D1–D3, flags/channels, status vs readiness,
consent/digests/read-back, exits, provenance and effects remain unchanged. G1
(`provider_create_ambiguous` lost on a no-payload JSON path) and G2 (64-item transfer
view) remain conditional observations. Never blindly replay uncertain external
mutations or call a truncated ledger complete. Concrete impact requires bounded
reproduction/Evidence and a separate focused correction with review/authority;
no schema, fix or Work Item is authorized here. #232/AXM-11 and AXM-31 acceptance
history is preserved; #305 and AXM-12/#278 remain outside this authority.

### Baseline, scope and validation

Clean checkout, `gh pr view 328` and `git ls-remote` agreed on
`add52daf16bb27d2189484485bae9c4ebea4938b`, branch
`docs/axm-32-output-separation-specification`, OPEN/Draft. Search for AXM-32 PRs
and the Linear attachment identified #328 as the sole open work PR; other search
hits were merged history. Before edits, remote HEAD still matched. The public
repository classification was verified. Baseline CI now shows PR Checks and both
delivery-metadata observations SUCCESS; the round-3 pending result is historical,
not a result for this correction's future SHA. Linear remains In Progress with
no Blocked by; no operational metadata is changed or canonical Execution inferred.

Only Plan, this appended Evidence and the Specifications index change. Clarify,
accepted Spec/ADR and the decisions index remain byte-identical to baseline.
No delegation: this bounded correction uses a linear path; no independent review
is claimed. Documentary/security self-review checks scope, public-content
classification, retained unknowns and unchanged authority. No provider text dump,
private incident data, credential or secret is included.

| Executed check | Result / scope |
|---|---|
| `./scripts/validate-repository.sh .` | PASS: repository/harness/adapters/automation/label/ADR/sensitive-file validation and validator regressions; native Runtime behavior explicitly SKIP/UNVERIFIED. |
| `python3 scripts/check-adr-governance.py .` | PASS, 31 ADRs; structural, not behavioral or human acceptance. |
| One-off Python local link/anchor check | PASS, 117 local links/anchors across all three changed documents. Remote decision/review targets were read through authenticated GitHub/Linear tools; no general external-link crawler run. |
| One-off byte/scope assertions | PASS: Spec identical to accepted `18d29a6`; Spec/ADR/Clarify/decisions index identical to pre-correction `add52da`; exactly three authorized documentary paths changed. |
| `git diff --check` | PASS; limited diff and public-content classification reviewed. |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN; no dependency installed. |

Publication preparation: staged diff/content/classification review,
`git diff --cached --check` and `./scripts/check-sensitive-files.sh --staged .`
passed. `./scripts/delivery-issues.sh check-pr --body <temporary-body> --title
<temporary-title>` passed (`metadata=declared`, related/completes `none`).
Published SHA and exact-SHA CI observations are recorded in the same PR update
after push, separate from baseline CI.

**NOT_RUN:** implementation, Go product suites, R1/AC1–AC9 behavioral/equivalence/
security checks, prepared-artifact/upgrade/rollback smoke, real Codex/Claude native
conversations, G1/G2 incident reproduction, Provider/Runtime effects and AXM-12 E2E.
The Plan remains Proposed. Next action is a new Plan review and explicit human
acceptance of the updated SHA before any `tasks` stage and its own gate/authority;
this execution stops after documentary publication and verification.

## Tasks preparation — 2026-10-18

### Authority and documentary result

The maintainer explicitly [accepted the Plan](https://github.com/rgomids/axiom/pull/328#issuecomment-6253809891)
at `962adfdcd22384c681613810b78022469ee813cb`, also confirmed by the
[Linear acceptance](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-42cacc85).
Read that decision, the [round-4 review](https://github.com/rgomids/axiom/pull/328#pullrequestreview-5564396763),
Spec/ADR, Clarify, Plan and current SDD/documentation/security governance before
editing. This later acceptance supersedes earlier pending-Plan gate statements
as current context; their historical bytes remain. Authority covers only Tasks
preparation/publication in the same Draft PR, not implementation or Tasks acceptance.

[Tasks](tasks.md) proposes eight stable IDs mapping all five Plan phases and
R1–R4/AC1–AC9, with objective, scope/impact, dependencies, completion evidence,
risks and gates. T03/T04 are the only potential parallel delivery units, after
T02 and with disjoint ownership; T05 integrates them before distribution/rollback.
T07 isolated smoke and T08 separately authorized native observations are distinct.
All are future units, not executions. D1–D3 and historical #232/AXM-11/AXM-31
acceptance are preserved; no #305 or AXM-12/#278 delivery is implied. G1/G2 stay
conditional observations with bounded impact/decision escalation, no preventive
fix/schema/migration/new Work Item. Technical completion never implies readiness.

Initial `git status`, `git fetch origin`, `gh pr view 328` and `git ls-remote`
agreed on a clean checkout and head `962adfdcd22384c681613810b78022469ee813cb`,
branch `docs/axm-32-output-separation-specification`, OPEN/Draft. Open AXM-32 PR
search returned only #328; Linear attachment agreed. Remote was re-read immediately
before editing and still matched; no concurrent delta was observed. No accessible
remote record inspected established another active Tasks writer. Canonical Execution,
its revision/gate and execution in other hosts remain unknown; no Runtime/store
was invoked or truth reconstructed. Linear remains In Progress, without blockedBy;
no status/relations/labels changed. Baseline CI was re-read: PR Checks and returned
delivery-metadata checks SUCCESS **only for `962adfd`**, not for this new change.

Only `tasks.md`, this appended Evidence and the Specifications index change.
Plan/Spec/Clarify/ADR/decisions index bytes are preserved. Index records the later
Plan acceptance and Tasks Proposed. GitHub confirms repository PUBLIC; user
publication authority covers these bounded decision summaries/references, not
raw provider comments or personal data. No raw chat, private incident record or
secret is imported. No delegation: a strictly inferior model hierarchy could not
be verified; documentary self-review is not independent review.

### Checks and limits

| Executed check | Result / scope |
|---|---|
| `./scripts/validate-repository.sh .` | PASS: documentary/bootstrap/harness/adapters/automation/label/ADR/sensitive-file checks and their validator regressions. Native behavioral scenarios explicitly SKIP/UNVERIFIED; no product behavior claimed. |
| `python3 scripts/check-adr-governance.py .` | PASS: 31 ADRs, structural only. |
| One-off local links/anchors check | PASS across Tasks/Evidence/index; remote authority/review targets read via authenticated GitHub/Linear. No general remote-link crawler. |
| One-off byte/scope/coverage assertions | PASS: Spec identical to accepted `18d29a6`; Plan/Spec/Clarify/ADR/decisions index identical to `962adfd`; eight unique ordered IDs, mandatory Task fields and R1–R4/AC1–AC9 coverage rows. Structural coverage is not behavioral proof. |
| Plan→Tasks and security self-review | No blocking documentary finding identified: all five phases and gates map to units; no implicit contract/authority change or unclassified content. Dynamic coverage remains unverified. |
| `git diff --check` | PASS, scoped content/diff inspected. |
| `command -v gitleaks` | Unavailable, dedicated secret scanner NOT_RUN; no dependency installed. Sensitive-file checks are not a substitute for a consolidated scanner. |

Publication preflight PASS: staged content/classification/path review (only the
three authorized Markdown paths), append-only Evidence assertion,
`git diff --cached --check`, `./scripts/check-sensitive-files.sh --staged .` and
`./scripts/delivery-issues.sh check-pr --body <temporary-body> --title <temporary-title>`
(`metadata=declared`, related/completes `none`). Commit SHA, remote read-back and
exact-SHA CI observation are recorded on the [same PR #328](https://github.com/rgomids/axiom/pull/328)
after publication, avoiding a self-referential commit SHA or attributing old CI
to new bytes. Stop publication if head/concurrency/scope differs.

**NOT_RUN:** implementation; Go product suites; R1/AC1–AC9 behavior, semantic
oracles and security canaries; pipelines/broken-write behavior; prepared-artifact,
installed-skill, upgrade/rollback smoke; native Codex/Claude conversations;
Provider/Runtime effects; G1/G2 incident reproduction; AXM-12 E2E. Documentation
checks only establish artifact integrity. Residual risks: dynamic JSON variants,
unknown external human-text consumers, native-host omission/injection and mixed
installed revisions. G1/G2 cannot be called resolved or harmless without evidence.

**Next:** human review and explicit approval of Tasks at the published SHA, then
separate bounded authority before implement. No Tasks acceptance, canonical
Execution transition, merge, release or product completion is recorded here.

## Implement preflight and recovery handoff — 2026-10-18

### Authority and observed state

The maintainer [approved all eight Tasks and authorized implementation](https://github.com/rgomids/axiom/pull/328#issuecomment-6253982994)
at `2efacfdb3957c458df84514b0e2a4ac532cae4c8`. This supersedes the preceding
pending-Tasks statements as current authority; their historical evidence remains.
No renewed Spec, ADR, Plan or Tasks approval is requested. Implementation authority
does not establish the location, identity, revision or concurrency of an Execution.

Clean checkout, `git rev-parse HEAD`, `git ls-remote origin
refs/heads/docs/axm-32-output-separation-specification` and `gh pr view 328`
agreed on that exact SHA, the expected branch and OPEN/Draft #328. Repeated the
remote/PR checks immediately before this append; no delta was observed. Read the
current PR comments/reviews, exact authorization, Linear issue/relations and
discussion. #328 is the only open AXM-32 work PR identified; search also returned
unrelated #329, which was excluded by identity. Linear is In Progress with no
blockedBy; those facts are not canonical workflow truth. GitHub reports PUBLIC.
Only bounded authorized references and locally observed results are recorded here.

**Unresolved preflight: canonical workflow truth unavailable (`recovery_required`).**
`LINGO_STATE_ROOT`, `LINGO_PROJECTS_ROOT` and `XDG_STATE_HOME` are unset.
`cmd/lingo/main.go::stateRoot` and `internal/local/state_root.go::NativeStateRoot`
therefore resolve this Linux session's state to `/home/box/.local/state/lingo`;
that directory does not exist. The default Project root `/home/box/.axiom/projects`
does not exist either. No repository-local checkpoint was identified in the
expected `.axiom` locations or tracked checkpoint/Execution search. `git worktree
list` identifies this clean worktree and separate AXM-33/AXM-34 worktrees; it does
not establish the absence of another host's Execution. No inspected remote record
supplies a canonical Execution ID, root, revision and resumable checkpoint.

This is an unavailable local observation, **not proof that no Execution exists**.
No store was initialized, Execution created/reconstructed/advanced, gate inferred
from approvals, Provider mutation replayed or operational metadata changed. The
requested checkpoint/root clarification remains unanswered at this checkpoint.
[Contribution governance](../../../CONTRIBUTING.md#work-item-lifecycle-governance)
requires inspection/reconciliation and human decision when local workflow truth
is unavailable; [Tasks entry](tasks.md#entrada-e-autoridade) requires handoff on
absence/contradiction. Do not bypass those rules or create a second Execution.
The only change here is this append, preserving useful read-only preflight work.

### T01 baseline inventory for resumption

These are current-source observations at `2efacfdb`, supplementing
[Clarify's producer/fact inventory](clarify.md#producer-and-emission-inventory).
They do not complete T01's frozen typed obligations or behavioral gates.
Invocation supplies operation identity where the DTO lacks it; unavailable
payloads remain unknown. No typed fact may be extracted from `result`/`next`.

| Family | Current emission boundary / material typed members | Consumer and resumption obligation |
|---|---|---|
| Common/parser/version | `completion.go`, `cli.go`: `completionEvent`, `Diagnostic`; status, refs/details, provenance | `WriteCompletion`, executable and JSON callers; retain invocation, exits and unknown target; do not add an envelope/category speculatively. |
| Project show/list/bootstrap/install | `completion.go`: `ProjectView`, `ProjectListView`, `BootstrapView`, `RuntimeView` | Project/Runtime skills and installed-set tests; preserve IDs, bindings, availability, ownership, conflicts, receipts/digests. |
| Validation/preflight/admission | `readiness.go`: `ReadinessReport`, `OperationReadiness`, `AdmissionDecision`, `ProjectStateView` | Project/Work Item prerequisites; success/0 is independent of structure/effective/operation blockers and grantsAuthority=false. Preserve every scoped blocker/warning. |
| Configure/edit/operational/Integration | `completion.go`, `operational.go`, `integration.go`: typed previews/reports and optional diagnostic | Project skill and setup smoke; proposed effects are not confirmed effects. Preserve exact digest, revisions, boundary and category where emitted. |
| Work Item/lifecycle | `completion.go`, `work_item_lifecycle.go`: draft/selection/change, questions, WorkItemView/list | Work Item skill/lifecycle suites; empty list differs from omitted list. Preserve target, exact consent/revision/digest, partial linkage and uncertainty. G1 remains conditional. |
| Execution/projection/list | `completion.go`, `execution_lifecycle.go`: WorkflowView, ProjectionPreview, ExecutionSummaryView | Work Item skill; preserve Execution ID, current gate, revision, transitions/refs and projection state. Category omission is not permission for prose parsing. |
| Workflow authoring/stage/plan/delivery | `workflow_authoring.go`, `workflow_stage.go`, `workflow_stage_plan.go`: WorkflowReport, stage payload, stagePlanEvent | Workflow authoring/Work Item execution skills; inventory each concrete `any` variant before claiming coverage; retain exact preview/policy digests, effects, Evidence and authority. |
| Runtime authoring/resolution/auth/capability | `runtime_authoring.go`, `runtime_resolution_completion.go`, `runtime_capability.go` | Runtime skill and execution preflight; typed AuthoringReport/resolution/AuthReport plus concrete capability variants. Preserve sanitized observations, revision/digests/impacts; capability is not dispatch consent. |
| Context | `project_context.go`: EffectiveContext | Project/Work Item prerequisite readers; context/Provider/workflow text is data, never host instructions or permission. |
| Maintenance | `maintenance.go` and `cmd/lingo/maintenance.go`: transfer/cleanup/recovery/retirement/upgrade views | Runtime recovery/direct callers; preserve certainty, refs, digests and known limits. G2 remains conditional, with no preventive repair or completeness claim. |
| Help / skill inspection / POC | `help.go`, `skill_inspect.go`, legacy `cli.go::emit` | Help remains human text with output flags; skill inspection has its own payload. Legacy statuses/writer guarantees require separate classification, not normalization to MVP. |

Concrete consumer search: `internal/cli` completion/presentation/parser/routing
tests, `cmd/lingo` blackbox/help/skill-protocol/operator journeys,
`internal/codexruntime` content/history/receipts/upgrade tests, `scripts` and
`.github/workflows`. Identified production consumers are the four shared embedded
skills; `integration.go::NewClaude` reuses those sources with a distinct root and
receipt. They currently request human output and require literal relay; all four
need coordinated replacement. Current manifest is skill-set `2`, binary
compatibility `2`; no bump or mismatch enforcement is claimed by this inspection.

Two concrete human-text automation consumers need migration: the human-version
assertion in `scripts/test-install-axiom.sh`, and `version_of()` in
`scripts/test-upgrade-journeys.sh`, which parses historical provenance text with
`sed`. The former must retain a native human assertion; version extraction for
automation must use explicit compatible JSON. `scripts/smoke-prepared-artifact.py`
already prepends `--json` and decodes with `json.loads`; its independent 64KiB
capture limit must not be confused with emitter budgets or G2.
`scripts/test-codex-skills.sh` invokes validators/Go tests, not a result parser.
External human-text consumers and installations outside this environment remain
unknown. `Run` prepends JSON for programmatic callers; `RunInteractive` preserves
the explicit/default selection boundary. No consumers were modified yet.

Baseline: 11 JSON presentation fixtures; SHA-256 over lexically sorted relative
path bytes + NUL + exact file bytes + NUL, repeated for each fixture:
`c847e6c9d3989169141d4f26174d87a8c9950be600e2c9ccf959781b61967146`.
Existing fixtures and accepted artifacts remain unchanged. Baseline shared skill
SHA-256 values, computed directly from each source's bytes:

| Skill | SHA-256 |
|---|---|
| axiom-project | `d676740525a254105521e1bf426284cc0a457830183da6ed6f8057dc7a8dfc26` |
| axiom-runtime | `7a3d90f74a9e5eb3f6580ca0bb9f55b8983db2fda9ea69feca2bae72499fef35` |
| axiom-work-item | `965df8b3435c4afa3fd674ceae86ce98ecae405440a00854478610c74ef18c39` |
| axiom-workflow | `c7c5dec9abcf5dd133036ef97c69203b52ca7d6aee1aa50dbf9d3616680f4773` |

### Task → files → dependencies → validation → Evidence

All eight Tasks remain **incomplete**. This matrix records the direct entry
blocker and each downstream gate rather than declaring a subset delivered.
There is no product implementation or new semantic oracle in this checkpoint.

| Task / AC mapping | Files/front at resume | Dependency and precise current impediment | Required validation / Evidence still owed |
|---|---|---|---|
| T01 / R1–R4, AC1–AC9 inputs | Clarify/Evidence, `internal/cli/testdata/presentation`, typed test fixtures | Entry reconciliation: canonical Execution/checkpoint/root/revision unavailable; inventory above is partial preparation only | Complete named concrete variants, typed obligation fixtures for every Spec matrix row including negative cases; freeze baseline and unknowns. |
| T02 / R1–R4, AC1–AC6 | `internal/cli/{presentation,readiness,cli}.go`, `cmd/lingo` validation tests, Project skill | T01 exit not established | Seven readiness variants, success/0 with Execution policy blocker, native snapshots, typed host assertions and one service call across channel selections. |
| T03 / R1/R2/R4, AC1/2/5/6 | CLI emitters/parser/help, executable boundary, native snapshots | T02 vertical slice not implemented/proved | Every semantic row; TTY/pipe/NO_COLOR/width, flags, short/broken writes, unchanged JSON/status/exits and no replay. |
| T04 / R1/R3/R4, AC3/4/7 | Four shared skills, content/routing/skill-protocol tests | T02; disjoint from T03 only after gate | Natural/explicit internal JSON, concise faithful synthesis, malformed/contradictory observations stop dependent effects; both isolated host roots. |
| T05 / R1–R4, AC1–AC8 | CLI/completion/readiness/operator/adversarial and isolated skill-tool suites | T03 and T04 integrated gates unavailable | Typed claims and contradictory-prose negatives, exact tokens, consent/drift/read-back/effect counters, ANSI/bidi/injection/secret canaries. |
| T06 / AC9, compatibility AC1/3/6 | `internal/codexruntime` manifest/history/receipts/upgrade tests | T05 equivalence/security exit unavailable | Both hosts' old/old, new/new and mixed-pair enforcement, protected/interrupted upgrade, four digests, no-op, modified/unsafe preservation and actual pair restoration. |
| T07 / AC1–AC9 consolidation | `docs/commands.md`, affected consumers/scripts, Evidence | T06 compatible-set/rollback proof unavailable | Prepared-artifact smoke, affected suites and checks, exact binary/skills, immutable pins/state corpus, document only implemented behavior. |
| T08 / AC3/4/5/7/8 | Isolated native fixtures and sanitized Evidence | T07 prepared set unavailable; verify own native gate/authority then | Native success, blocked validation, denied consent and partial/interrupted recovery in Codex and Claude; bounded expected/observed facts and tool traces; independent review/human decision. |

### Validation and resumption limits

Executed baseline checks, before any code change:

| Check | Observed result / limit |
|---|---|
| `go test ./internal/cli ./internal/completion ./internal/projectapp ./internal/codexruntime ./internal/compatibility` | PASS, exit 0. Tests establish the existing Markdown/relay and compatibility baseline, not the accepted future ACs. |
| `go test ./cmd/lingo` | PASS, exit 0, 204.876s. Existing executable/operator/readiness/authority regressions; no new AXM-32 behavior tested. |
| `./scripts/validate-repository.sh .` | PASS, exit 0: harness/adapters, registered automation, labels, ADR structure and sensitive-file checks. Native behavioral scenarios explicitly SKIP/UNVERIFIED. |
| One-off append-only/byte/link assertions; `git diff --check` | PASS: Evidence prefix preserved, Spec/Clarify/Plan/Tasks/ADR unchanged, three added local links/anchors valid; whitespace clean. |
| Engineering/security self-review | Only appended Evidence changes. Source hierarchy, authority, status/readiness separation, public-content classification and no schema/domain/dependency/comment changes checked. Workflow-state finding remains unresolved; this is not independent review. |
| Python standard-library fixture/skill hash calculation | PASS: 11 fixture files and four exact source digests above. No new automation/dependency added. |
| `gh pr checks 328` at `2efacfdb` | PASS: PR Checks run `39156896870` and delivery-metadata run `39156935788`. Baseline only; do not attribute to the checkpoint commit. |
| `command -v axiom` | Unavailable. No binary installed or state initialized to manufacture workflow truth. |
| `command -v codex`; `command -v claude` | Both executables found; native conversations NOT_RUN because T07's verified set does not exist. Availability is not native behavioral Evidence. |
| `command -v gitleaks` | Unavailable; dedicated scanner NOT_RUN; no global dependency installed. |

Checkpoint documentary checks and exact-SHA CI are recorded on the same PR after
execution/publication. AC1–AC9/R1–R4 future behavior, native rendering/semantic
equivalence/security cases, mixed-pair/rollback proof, prepared smoke and native
conversations are **NOT_RUN** against an AXM-32 implementation. G1/G2 incident
reproduction NOT_RUN; neither is promoted to a preventive blocker. No real
Provider/Runtime effects or AXM-12 E2E were run. No independent review or acceptance
is claimed. Orchestration/review/document skills were consulted; no delegation
because strict model inferiority could not be established. No code/comment,
dependency, schema, domain, status rule, accepted decision or installed set changes.

**RESUME:** obtain the canonical state root/checkpoint for AXM-32 and inspect its
Execution ID, revision, current gate, prerequisites and active ownership before
resuming. If that truth is missing on the owning host too, reconcile through the
governed recovery decision instead of constructing it from this record. Preserve
the same branch/PR and all eight dependencies above. The request is for missing
workflow facts, not another implementation approval. Then finish T01 and proceed
through T08; request independent engineering/security review and human acceptance
of the resulting exact artifacts. No merge, release, closure or automatic advance.

## Resume r2 — new box state root and Execution binding — 2026-10-18

### Authority and root creation

Rafael's resume-r2 instruction (2026-10-18, 16:31 BRT) explicitly authorizes
creating a **new**, box-local canonical Lingo state root for AXM-32. This
supersedes the previous checkpoint's request for an existing root on this box;
it says nothing about Executions on other hosts. The implementation authority
for all eight Tasks remains the [existing approval](https://github.com/rgomids/axiom/pull/328#issuecomment-6253982994)
at `2efacfdb3957c458df84514b0e2a4ac532cae4c8`; no renewal is requested.

Pre-edit checks: clean worktree, expected branch, local HEAD, remote
`refs/heads/docs/axm-32-output-separation-specification`, and OPEN/Draft PR #328
all agree on `d555dc8fe0e027b8eb9803ba1c3650865f4c840c`. Read the PR body,
comments and reviews, including the exact Tasks authorization. The open-PR search
for AXM-32 returns #328. GitHub reports the repository PUBLIC. Read Spec,
Clarify, Plan, Tasks, ADR-0030, contribution/security/quality/automation policies
and current CLI/Runtime implementation. Historical accepted artifacts remain
unchanged; `d555dc8` remains the prior checkpoint reference.

`LINGO_STATE_ROOT` and `XDG_STATE_HOME` are unset. The actual resolver is
`cmd/lingo/main.go::stateRoot` → `internal/local/state_root.go::NativeStateRoot`:
on Linux its canonical default is `/home/box/.local/state/lingo` (the basename
is **lingo**, not axiom). After verifying that this path neither existed nor
was a symlink, a Python standard-library one-off outside tracked automation
created that directory with `mkdir(mode=0o700)`. Read-back: mode `0700`, zero
entries. The default Project root `/home/box/.axiom/projects` is absent. The new
root is local operational state, not a tracked artifact. No existing state was
overwritten, imported, reconstructed or removed.

### Concrete remaining Execution prerequisite

Built the exact clean checkpoint with
`go build -o /tmp/axiom-axm32-baseline ./cmd/lingo` (PASS, exit 0).
The build reports revision `d555dc8fe0e0`, sourceState `clean`.
The following bounded observations were decoded as JSON, with actual process
exit codes captured independently; all had empty stderr:

| Invocation after `/tmp/axiom-axm32-baseline` | Exit / status | Observed fact |
|---|---|---|
| `--json project list` | 0 / success | `projects: []`, No configured Projects. |
| `--json workflow list --project axiom` | 1 / validation_failure | No configured Project by that selector; this is not an empty Execution-list proof. `axiom` was an inspection candidate, never an inferred canonical identity. |
| `--json workflow start --project axiom --repository main --work-item linear:rgomids#AXM-32 --role implementation --complexity high --capabilities axiom-skills` | 1 / validation_failure | Explicit selector input is invalid; rejected before application dispatch. No Execution created. |

The last invocation is a bounded capability/refusal check, **not** an attempt to
substitute a guessed Project/Repository for canonical identity. Source confirms
the independent limitation: `internal/cli/cli.go::parseWorkItemSelector` accepts
only `github:<owner>/<repository>#<positive integer>`;
`internal/projectapp/capability.go::SupportedProviders` contains only GitHub.
`workflow start --help` and `project configure --help` were inspected. Current
`cmd/lingo/execution_target.go::WorkflowStart` also requires configured Project,
validated Work Item linkage and an exact reviewed Runtime preview. A fresh root
alone cannot supply those facts or consent.

`gh issue list --state all --search 'AXM-32 in:title'` returned no matches.
That bounded search does not prove no differently titled mapping exists. AXM-32's
provided identity is Linear, and no authorized GitHub Work Item binding was
provided or discovered. PR #328 is not silently treated as a Work Item. No
mirror Issue, Project, linkage, Profile, Runtime policy or Execution was created;
no local workflow gate or Provider metadata was changed. Root read-back after
the checks still has zero entries, establishing no active Execution **inside
this newly created root**, without any assertion about other hosts/roots.

A material clarification was requested: supply an existing authorized GitHub
Work Item binding for this Execution, or explicitly decide that this delivery
uses the approved SDD artifacts without a Lingo-managed Execution for Linear.
This is distinct from the already-resolved root authorization and already-granted
implementation approval. It does not propose implementing a Linear adapter,
expanding schema/domain scope, or using a mirrored backlog by convenience.
The [Tasks entry](tasks.md#entrada-e-autoridade) says to hand off on missing or
contradictory local workflow facts; [contribution governance](../../../CONTRIBUTING.md#work-item-lifecycle-governance)
requires human decision on insufficient truth. No registered Execution/revision/
gate is claimed from Repository or Provider history.

### Task → files → dependencies → validation → Evidence

No product implementation or presentation test fixture was changed in r2.
This is a precise entry-prerequisite report for **all eight Tasks**, not a claim
that a subset implements AXM-32. The [previous matrix](#task--files--dependencies--validation--evidence)
continues to describe the required tests; its missing-root condition is resolved
by the creation above. Remaining gates are:

| Task / requirements | Intended files / front | Dependency and current result | Validation / Evidence still required |
|---|---|---|---|
| T01 / R1–R4, AC1–AC9 inputs | Clarify/Evidence; CLI typed obligation fixtures | INCOMPLETE: new root reconciled; Execution identity/binding decision outstanding. Read-only inventory remains preparation. | Freeze all concrete variants and typed obligations, preserve baseline hashes, classify unknown consumers. |
| T02 / R1–R4, AC1–AC6 | CLI presentation/readiness; Project skill; command tests | NOT_RUN: T01 exit unavailable. | Seven readiness variants, success/0 with policy-blocked Execution, one-call counters and channel/host assertions. |
| T03 / R1/R2/R4, AC1/2/5/6 | CLI emitters/parser/help; native snapshots | NOT_RUN: T02 exit unavailable. | Entire semantic matrix, unchanged JSON, TTY/pipe/NO_COLOR/width, write failures and zero replay. |
| T04 / R1/R3/R4, AC3/4/7 | Four shared skills and routing/protocol tests | NOT_RUN: T02 exit unavailable. | Internal JSON, faithful synthesis, invalid observations stop dependent effects; both isolated host roots. |
| T05 / R1–R4, AC1–AC8 | Typed oracles; consent/security/integration suites | NOT_RUN: T03/T04 exits unavailable. | Positive/negative semantic claims, exact tokens, drift/read-back/effect counters, ANSI/bidi/injection/redaction. |
| T06 / AC9; AC1/3/6 compatibility | Runtime manifest/history/receipts/upgrades | NOT_RUN: T05 exit unavailable. | Old/new pair matrix in both hosts, protected upgrade, no-op/interruption and actual pair restoration. |
| T07 / AC1–AC9 consolidation | Commands/consumers; prepared smoke; Evidence | NOT_RUN: T06 exit unavailable. | Verified compatible binary/four-skill set, checks, immutable pins/state corpus and truthful references. |
| T08 / AC3/4/5/7/8 | Native isolated observations; sanitized Evidence | NOT_RUN: T07 set unavailable; its separate native-authority gate still applies. | Success, blocked validation, denied consent and partial/interrupted recovery in each host; independent review/human decision. |

### Validation and delivery limits

- PASS: `go test ./internal/cli -run
  'Test.*(ExecutionTarget|WorkflowStart|WorkItemSelector|Strict)' -count=1`,
  exit 0. This is the existing selector/parser regression boundary, not new
  AXM-32 implementation evidence.
- PASS: baseline binary build, JSON parsing/independent exits, root mode/empty
  read-back and bounded remote identity/public-classification inspection above.
- AC1–AC9/R1–R4 implementation, new JSON/terminal equivalence/snapshots,
  semantic/consent/security tests, pair upgrades/rollback, prepared smoke and
  native Codex/Claude observations: NOT_RUN against a new implementation because
  no implementation exists here. Baseline CI is not reused as behavioral proof.
- PASS: `./scripts/validate-repository.sh .`, exit 0; native behavioral
  scenarios explicitly SKIP/UNVERIFIED. PASS: `git diff --check` and a Python
  byte comparison proving append-only Evidence and unchanged Spec/Clarify/Plan/
  Tasks/ADR. These are documentary checks, not behavioral acceptance.
- `gitleaks` remains unavailable: dedicated scan NOT_RUN. Staged sensitive-file
  checks and exact-checkpoint CI are recorded with publication. No raw Provider/
  chat content or secret is included.
- Review/documentation skills were applied to this bounded Evidence append.
  No delegation: strict model inferiority remains unproved. No independent
  review, human acceptance, merge/release/closure or completion claim.
- G1/G2 remain conditional, unreproduced and unchanged. This prerequisite is
  neither G1 nor G2, and no preventive correction, dependency, schema, domain,
  code comment, automation surface or installed skill change was introduced.

Resume on the same branch/PR using the existing new root; **do not create it
again or start a second Execution**. Inspect it for intervening state first.
Resolve only the Execution-binding/delivery-mode decision above, then satisfy
T01 → T02 → (T03/T04) → T05 → T06 → T07 → authorized T08. Independent review
and human acceptance remain subsequent gates. Do not ask for the former absent
root or renew the existing eight-Task implementation approval.


## Resume r3 — authorized GitHub binding and local preflight — 2026-10-18

Rafael authorized this resume on 2026-10-18 at 18:08 BRT from
`aba1c4d843e4b44a891531831aec2c79e95aae18`, on the same branch and Draft
PR #328. The eight-Task implementation authority at `2efacfdb3957c458df84514b0e2a4ac532cae4c8`
was read back from [the approval](https://github.com/rgomids/axiom/pull/328#issuecomment-6253982994).
Local HEAD, remote branch and OPEN/Draft PR HEAD agreed before editing; the
worktree was clean. The PR conversation/reviews and current accepted inputs were
inspected. Historical Proposed language is not a request to renew their approval.

**Authorized exception:** use existing public GitHub
[issue #330](https://github.com/rgomids/axiom/issues/330) as the Lingo Work Item
only for this implementation-validation context. Its body confirms the exception.
Operational authority remains **Linear AXM-32**. This neither adopts GitHub as a
second operational backlog nor authorizes changing Provider status/labels,
closing the issue or accepting work. The former missing GitHub binding is resolved.

The existing `/home/box/.local/state/lingo` was reused for inspection: mode 0700,
zero entries before commands. No active Execution exists inside that inspected
root. No assertion concerns another machine/root. No second Execution or manual
store reconstruction was attempted.

Built the clean baseline with `go build -o /tmp/axiom-axm32-r3 ./cmd/lingo`:
PASS, exit 0, provenance revision `aba1c4d843e4`, sourceState clean.
Read-only commands against that baseline establish:

| Command after the baseline executable | Result / exit | Meaning |
|---|---|---|
| `--json project configure --slug axiom-axm32-validation --name "Axiom AXM-32 validation" --repository main=/workspace/wt/axm-32 --work-item-provider github` | success / 0 | Preview only; single discovered remote `https://github.com/rgomids/axiom`; portable/local revisions absent; proposed local effects are publication of Project and bindings. Runtime policy absent, candidates empty. |
| `--json runtime profile validate` | validation_failure / 1 | No valid machine-local Runtime Profile configuration. |
| `--json runtime binding list` and `--json runtime profile list` | success / 0 each | Configuration state fresh, revision 0; Codex and Claude executables detected, integration unobserved; authentication unknown, model/effort support unproven. Detection does not select a Runtime/model. |
| `--json runtime codex status` and `--json runtime claude status` | validation_failure / 1 each | All four skills missing, receipt absent, skillSetVersion 2 / binaryCompatibility 2 in each host. |

An exploratory `runtime <host> inspect` is not a supported command: both returned
legacy invalid_command / 1. Subsequent actual help discovery identified `status`
as the supported read-only inspection; those canonical results appear above.
No mutation was replayed. Project CREATE, runtime binding/profile authoring,
installation and workflow-start help and the implementation's exact preview/check
requirements were inspected. No preview has been applied at this checkpoint.
Runtime/model/effort selection was requested because no existing local Profile
provides those facts; not a renewal of implementation, root or #330 authority.

### T01 baseline preparation and consumer treatment

This is preparation, not T01 completion or permission to skip its Execution entry
prerequisite. No presentation code or production skill was changed.
The existing producer/DTO/fact inventory in [Clarify](clarify.md#producer-and-emission-inventory)
was checked against current emitters. The following named consumers must migrate
with the coordinated set:

| Consumer / current boundary | Treatment / required proof |
|---|---|
| `internal/cli/presentation_test.go`, `testdata/presentation/*.md` | Replace Markdown human snapshots with native text; preserve JSON bytes and material-field assertions. Existing seven completion statuses and eleven operation scenarios cover no-op, partial, denied, interrupted, recovery, retryable and hostile values; readiness variants remain to add. |
| `internal/cli/domain_skill_routing_test.go` | Currently rejects leading JSON in routing rows as outside human-first; migrate machine routing assertions to explicit JSON while preserving domain ownership/selectors. |
| `cmd/lingo/skill_protocol_test.go` | Parse actual skill invocation examples; preserve exact start preview/selector protocol while switching internal consumption to JSON. |
| `internal/codexruntime/runtime_test.go`, `workflow_skill_content_test.go` | Current exact Markdown/explicit-user-JSON/Auxiliary assertions must be replaced by canonical internal JSON plus faithful host synthesis checks. |
| `internal/codexruntime/{manifest.go,shared_history_test.go,published_receipts_test.go,domain_skills_upgrade_test.go}` | Preserve published digests/receipts and add replaced revision recognition; prove both host roots, compatible pairs and protected rollback. Current manifest versions both 2 are baseline facts, not evidence of mixed-pair safety. |
| `scripts/smoke-prepared-artifact.py::run` | Already prepends explicit JSON and parses JSON. Its independent 64KiB capture budget must remain distinct from emitter budgets and G2. |
| `scripts/test-codex-skills.sh` | Validator/Go-suite caller; migrate affected contract assertions, not a human-output JSON parser. |
| Help/version/skill-inspect/legacy POC | Classify independently; help stays human; legacy status/error transport is not silently converted to MVP DTOs. Skill inspect canonical completion uses presentEvent. |
| Unknown downstream callers | External human-text parsers/installed revisions are unobserved; no universal consumer compatibility claim. |

The canonical `presentEvent` boundary receives an already-built wire and performs
one write; presentation cannot call a use case again. `presentDiagnosticEvent`
appends only bounded optional diagnostic enrichment before that boundary.
Actual emitter limits observed include common/readiness 16KiB, Work Item 128KiB,
Project/Execution lists 2MiB, stage-plan 4MiB, Runtime authoring/capability 256KiB;
these are not a new universal result schema or a change to limits.

Before any product edit a one-off Python standard-library inventory captured
SHA-256 for each baseline file outside tracked automation. Group hashes below use
UTF-8 JSON of the sorted path-to-SHA256 mapping with compact separators:

| Immutable baseline group | Files | Mapping SHA-256 |
|---|---:|---|
| `internal/cli/testdata/presentation/*.json` | 11 | `cfcd029243d0a19d1d7b05648d6abfee77d25cc2da9cf66eab575bae8562f316` |
| `internal/compatibility/testdata/stable-v1/snapshots/` recursive regular files | 393 | `0905dde3166ef505ce10b5ce1d2666f5ac5a97887cb706202d6fd0731dbd2ca3` |
| Four embedded `internal/codexruntime/skills/*/SKILL.md` sources | 4 | `e93613cd40b1038bd78d28a55e4369dd18a87db15226f3a273d396c87c050e02` |

PASS: `go test ./internal/cli ./internal/codexruntime ./cmd/lingo`, exit 0.
These existing baseline tests do not prove AXM-32 behavior. Native observations,
new typed obligations/readiness matrix and compatibility/rollback of the new set
remain NOT_RUN. G1/G2 remain conditional and unchanged; no occurrence reproduced.
No delegation: the runtime exposes overrides, but strict model inferiority to the
Orchestrator is unproved. No independent review is claimed.


### r3 Task and acceptance gate checkpoint

| Task | Files/front | Dependency / precise outstanding gate | Required validation / acceptance state |
|---|---|---|---|
| T01 | Clarify/Evidence; presentation typed obligations | INCOMPLETE: baseline captured; no configured Runtime/Profile from which to produce a valid start preview and create the authorized local Execution. Project setup is preview-only. | Freeze full semantic/readiness/channel obligations after entry; AC1–AC9 inputs incomplete. |
| T02 | CLI readiness/presentation and Project skill/tests | NOT_RUN: T01 exit not met. | Seven readiness variants, status success/0 vs scoped Execution blocker, one-call/channel checks; AC1–AC6. |
| T03 | All canonical CLI emitters/native snapshots | NOT_RUN: T02 exit not met. | Native semantic matrix, JSON unchanged, writer failure, TTY/pipe/NO_COLOR; AC1/2/5/6. |
| T04 | Four shared skills and routing/protocol integration | NOT_RUN: T02 exit not met. | Internal JSON, bounded faithful synthesis and fail-closed dependent actions in both hosts; AC3/4/7. |
| T05 | Typed semantic oracle, consent/security tests | NOT_RUN: T03/T04 exits not met. | Positive/negative facts/tokens, drift/read-back, effect counters and security canaries; AC1–AC8. |
| T06 | Manifest/history/receipts and upgrade tests | NOT_RUN: T05 exit not met. | Old/new pairs, protected interrupted upgrade and actual pair rollback in both hosts; AC9. |
| T07 | Command/reference consumers and prepared smoke | NOT_RUN: T06 exit not met. | Exact binary/four-skill set, prepared smoke, complete checks and AC mapping. |
| T08 | Isolated native observations and sanitized Evidence | NOT_RUN: T07 and separate native-observation authority gates not met. | Success, blocked validation, denied consent, partial/interrupted recovery in each host; independent review/human decision pending. |

No Task or R1–R4/AC1–AC9 behavior is declared complete. The smallest missing
input is the explicit Runtime/model/effort selection for the local configuration;
then use the actual authoring previews and exact digests, read back configuration,
configure the Project with that policy, link #330 through supported commands,
inspect Executions again, and start only if none is active. Preserve local gates;
accepted repository artifacts do not constitute a fabricated transition history.
The entry remains a Runtime configuration prerequisite, not G1/G2 or a request to
renew the already authorized implementation. No new adapter, schema/domain rule,
remote infrastructure, secret, dependency, code comment or durable automation was
introduced. Review/documentation skills were applied to this bounded append;
self-review is not independent review. No merge/release/closure or automatic acceptance.


Final r3 local verification: PASS `go test ./...` (exit 0); PASS
`./scripts/validate-repository.sh .` (exit 0; native behavioral scenarios explicitly
SKIP/UNVERIFIED); PASS `git diff --check`; PASS byte comparison for all captured
JSON/state/skill/accepted-artifact files; PASS read-back that the reused canonical
state root still has zero entries. `gitleaks` is unavailable: dedicated scanner
NOT_RUN; repository/staged sensitive-file checks remain required for publication.
Only this append changes tracked content. Exact-checkpoint CI must be queried on
the published SHA and must not be inherited from `aba1c4d`.
