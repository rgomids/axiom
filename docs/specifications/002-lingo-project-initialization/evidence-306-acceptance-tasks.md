# Evidence — Issue #306 acceptance reconciliation and Tasks

## Authority receipt — 2026-10-11

[Explicit independent six-artifact acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6109183466), posted by maintainer `rgomids` at 2026-10-11 12:48:06 UTC (09:48:06 America/Sao_Paulo), is fixed to commit `823bce3141bae5e90951c5239181c46d88cfbfe9`. The current human request separately authorizes status/supersession/index reconciliation and complete Tasks preparation/publication on the same PR branch. It does not accept Tasks or authorize implementation.

| Historical approved artifact | Exact approved Git blob | Decision |
| --- | --- | --- |
| [plan-306-local-runtime-profiles.md](https://github.com/rgomids/axiom/blob/823bce3141bae5e90951c5239181c46d88cfbfe9/docs/specifications/002-lingo-project-initialization/plan-306-local-runtime-profiles.md) | `d4886e9c282ee12d3526566d0ddfc31f8fffb391` | Accepted |
| [amendment-306-local-runtime-authoring.md](https://github.com/rgomids/axiom/blob/823bce3141bae5e90951c5239181c46d88cfbfe9/docs/specifications/002-lingo-project-initialization/amendment-306-local-runtime-authoring.md) | `c504cd5204e38eacafe18ae90179c6b646d0cd1e` | Accepted |
| [security-review-306-local-runtime-profiles.md](https://github.com/rgomids/axiom/blob/823bce3141bae5e90951c5239181c46d88cfbfe9/docs/specifications/002-lingo-project-initialization/security-review-306-local-runtime-profiles.md) | `d7105750fa50f9c92179d330ab604a817d2c1d71` | Accepted design baseline |
| [0023-local-runtime-configuration-surface.md](https://github.com/rgomids/axiom/blob/823bce3141bae5e90951c5239181c46d88cfbfe9/docs/decisions/0023-local-runtime-configuration-surface.md) | `f39a75b1d227016bbd19b096bfae326ad92049f1` | Accepted |
| [amendment-306-profile-effort-onboarding.md](https://github.com/rgomids/axiom/blob/823bce3141bae5e90951c5239181c46d88cfbfe9/docs/specifications/007-configurable-workflows/amendment-306-profile-effort-onboarding.md) | `4bc5b65830fb9ec885bdd53c2484f53c13a02b86` | Accepted |
| [0024-model-profile-effort-ownership.md](https://github.com/rgomids/axiom/blob/823bce3141bae5e90951c5239181c46d88cfbfe9/docs/decisions/0024-model-profile-effort-ownership.md) | `c9753a515ef43d2a84bae2156b1c6f8eac170dd1` | Accepted |

The remote PR HEAD and local HEAD matched this commit before edits; all six exact blobs were compared with the receipt and local file bytes. No post-acceptance delta existed. Their historical Git objects stay addressable; reconciled files at the new revision record authority/status, not a retroactive change to that receipt. [Prior Planning Evidence](evidence-306-planning-reconciliation.md) and all earlier receipts remain unchanged.

## Remote and concurrent-contract baseline

- PR #310 OPEN/Draft, branch `feat/306-local-runtime-profiles`, base `main` at `5d52327b0367ba08d872b50e804df5c7d63a3e89`; initial reviewed head `823bce3141bae5e90951c5239181c46d88cfbfe9`. No new branch/PR.
- Open related PR #311 at `4c857ead8f4c7ec8d027df2d8eef746464b55d17` contains Proposed #276 planning, not production. It references #306's accepted Profile-owned effort; its proposed capability proof delivery remains externally owned. No other accepted contract is imported from that PR.
- Both branches introduce different ADR paths numbered 0024 and touch the two indexes. ADR-0018 identity is file-path based: preserve accepted `0024-model-profile-effort-ownership.md`; eventual integration requires index/identity reconciliation, not silent replacement or authority from a shared number. No #311 mutation here.

## Reconciliation scope

- Six artifacts status/backlinks and the two indexes now reflect independent acceptance. Format/window/no-override/legacy-continuation design is accepted, not reopened as a pending concept. Exact Tasks and future-code/native security gates remain separate.
- ADR-0022 remains Accepted. Its sole Stage/Agent effort-owner fragment has the canonical reciprocal annotation to accepted ADR-0024; ADR-0024 records `Supersedes`. Spec 007 has bounded additive annotations at Agent requirements, effort semantics and Stage Plan DTO. All original historical paragraphs remain present.
- Spec 004's two-skill consolidation remains historical, with one additive inventory evolution note for accepted #303 and future #306 fourth skill. No source/catalog/installed inventory changes.
- [Tasks](tasks-306-local-runtime-profiles.md): 23 Not-started units, 16 confirmed local T units and seven transversal X units blocked by PD-OWN. Every unit includes scope/reference/components/dependencies/AC/negatives/Evidence/Done condition. Accepted formats mapped to explicit migrations; dependency DAG has no cycles.
- Recommendation: keep #306 local; assign a separately authorized bounded Workflow/compiler/Execution migration bundle, or explicitly assign that bundle to #306. **Pending Human Decision**, not assumed expansion. No new Issue or external operation.

## Sources and bounded specialist verification

Repository `axiom-sdd`, `axiom-document`, `axiom-orchestrate`, security policy, ADR-0018 and directly affected approved artifacts/current interfaces were consulted. Three existing capability-scoped reviewers were reused read-only with inherited Orchestrator settings and bounded outputs: contracts/compatibility; security/native gates; onboarding/default workflow. No concrete model was hardcoded or higher capability selected; the tool does not expose a relative capability catalog, so no claim of optimized cheaper-profile selection is made. The Orchestrator integrated and checked their findings; none accepted artifacts or implemented work.

Mandatory named sources “00 - Axiom Core”, “01 - Protocolo Operacional”, “02 - Playbook SDD” were not located in bounded repository/connected Notion search. Source paths/links have been requested; verification remains pending. Repository SDD conventions were used rather than inventing unavailable content. No private provider content was imported: only this lookup limitation is recorded. Exact Playbook conformance cannot yet be asserted.

## Documentary validation

Actual checks and receipts for this revision are recorded below. No production tests, operational migration, Runtime process, authenticated probe, model inference or Provider effect is executed in this phase. Tests specified inside Tasks are future expected Evidence only.

- `./scripts/validate-repository.sh .` — PASS; 25 ADR structures and reciprocal governance accepted; vendor behavior explicitly SKIPPED/UNVERIFIED.
- Approved-blob/initial-worktree comparison — PASS: six independently approved blobs match exact commit/receipt; no post-acceptance material delta.
- Task structure/dependency/traceability check — PASS: 23 unique Not-started units, ten required fields each, 16 local/seven PD-OWN-gated transversal units, valid dependency references and acyclic DAG; all D1–D5/IA/LA/PE/ON/AC markers traced. Security review added pre-migration safety dependency, environment/source drift negatives and preserved nonmutating shared-lock semantics.
- Local Markdown target/anchor check — PASS: 215 local targets and 13 heading anchors.
- Source/schema/fixture and historical receipt preservation — PASS: seven selected source/schema/fixture baselines and prior Planning Evidence byte-identical to approved HEAD. All original Spec 004/007/ADR-0022 lines preserved in order with additive annotations only. Exact approved Git objects remain unchanged.
- Diff scope/whitespace — PASS: 13 Markdown artifacts only; staged sensitive-file check PASS; scoped gitleaks on Spec 002, Spec 007 and ADRs PASS, no detected leaks.
- PR metadata — PASS: Related-Issues #306, Completes-Issues none. Publication remote-head and final read-back — pending publication; exact final commit/Task blob/CI classification will be recorded in the PR review record.

## Limits and stop

Accepted design is not implemented or certified. Synthetic assertions do not prove actual model availability/auth/effort or native vendor conversation. G-3 proof/#305 policy/#276 dispatch/#277 rework/#278 real acceptance remain separate. PD-OWN and missing exact source-convention verification remain explicit review points. **Awaiting Human Approval — Tasks.** All units Not started; no Ready transition/merge/release/Issue state or label/milestone/Linear mutation.
