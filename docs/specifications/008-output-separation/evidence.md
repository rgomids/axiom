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
