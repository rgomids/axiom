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

## Historical remote and concurrent-contract baseline — c4cb5aa

- PR #310 OPEN/Draft, branch `feat/306-local-runtime-profiles`, base `main` at `5d52327b0367ba08d872b50e804df5c7d63a3e89`; initial reviewed head `823bce3141bae5e90951c5239181c46d88cfbfe9`. No new branch/PR.
- Open related PR #311 at `4c857ead8f4c7ec8d027df2d8eef746464b55d17` contains Proposed #276 planning, not production. It references #306's accepted Profile-owned effort; its proposed capability proof delivery remains externally owned. No other accepted contract is imported from that PR.
- Both branches introduce different ADR paths numbered 0024 and touch the two indexes. ADR-0018 identity is file-path based: preserve accepted `0024-model-profile-effort-ownership.md`; eventual integration requires index/identity reconciliation, not silent replacement or authority from a shared number. No #311 mutation here.

## Historical acceptance/Tasks reconciliation scope — c4cb5aa

- Six artifacts status/backlinks and the two indexes now reflect independent acceptance. Format/window/no-override/legacy-continuation design is accepted, not reopened as a pending concept. Exact Tasks and future-code/native security gates remain separate.
- ADR-0022 remains Accepted. Its sole Stage/Agent effort-owner fragment has the canonical reciprocal annotation to accepted ADR-0024; ADR-0024 records `Supersedes`. Spec 007 has bounded additive annotations at Agent requirements, effort semantics and Stage Plan DTO. All original historical paragraphs remain present.
- Spec 004's two-skill consolidation remains historical, with one additive inventory evolution note for accepted #303 and future #306 fourth skill. No source/catalog/installed inventory changes.
- [Tasks](tasks-306-local-runtime-profiles.md): 23 Not-started units, 16 confirmed local T units and seven transversal X units blocked by PD-OWN. Every unit includes scope/reference/components/dependencies/AC/negatives/Evidence/Done condition. Accepted formats mapped to explicit migrations; dependency DAG has no cycles.
- Recommendation: keep #306 local; assign a separately authorized bounded Workflow/compiler/Execution migration bundle, or explicitly assign that bundle to #306. **Pending Human Decision**, not assumed expansion. No new Issue or external operation.

## Sources and historical specialist verification

Repository `axiom-sdd`, `axiom-document`, `axiom-orchestrate`, security policy, ADR-0018 and directly affected approved artifacts/current interfaces were consulted. Three existing capability-scoped reviewers were reused read-only with inherited Orchestrator settings and bounded outputs: contracts/compatibility; security/native gates; onboarding/default workflow. No concrete model was hardcoded or higher capability selected; the tool does not expose a relative capability catalog, so no claim of optimized cheaper-profile selection is made. The Orchestrator integrated and checked their findings; none accepted artifacts or implemented work.

The current human request clarifies that private Core/Operational Protocol/SDD Playbook instructions are incorporated in its architecture/operational/SDD sections. They require no discovery, path request, external import or availability gate. The previous source-availability review point is withdrawn. This revision uses those incorporated instructions and repository technical contracts; no additional private content is invented.

## Historical documentary validation — c4cb5aa

Actual checks and receipts for the prior c4cb5aa revision are recorded below; current corrections have their separate validation section. No production tests, operational migration, Runtime process, authenticated probe, model inference or Provider effect is executed in this phase. Tests specified inside Tasks are future expected Evidence only.

- `./scripts/validate-repository.sh .` — PASS; 25 ADR structures and reciprocal governance accepted; vendor behavior explicitly SKIPPED/UNVERIFIED.
- Approved-blob/initial-worktree comparison — PASS: six independently approved blobs match exact commit/receipt; no post-acceptance material delta.
- Task structure/dependency/traceability check — PASS: 23 unique Not-started units, ten required fields each, 16 local/seven PD-OWN-gated transversal units, valid dependency references and acyclic DAG; all D1–D5/IA/LA/PE/ON/AC markers traced. Security review added pre-migration safety dependency, environment/source drift negatives and preserved nonmutating shared-lock semantics.
- Local Markdown target/anchor check — PASS: 215 local targets and 13 heading anchors.
- Source/schema/fixture and historical receipt preservation — PASS: seven selected source/schema/fixture baselines and prior Planning Evidence byte-identical to approved HEAD. All original Spec 004/007/ADR-0022 lines preserved in order with additive annotations only. Exact approved Git objects remain unchanged.
- Diff scope/whitespace — PASS: 13 Markdown artifacts only; staged sensitive-file check PASS; scoped gitleaks on Spec 002, Spec 007 and ADRs PASS, no detected leaks.
- PR metadata — PASS: Related-Issues #306, Completes-Issues none. Final historical c4cb5aa remote-head/read-back/CI receipt is [the prior PR review record](https://github.com/rgomids/axiom/pull/310#issuecomment-6109364967). Its source/ownership pendencies are superseded by the current receipt below, not new implementation authority.

## Current ownership reconciliation — 2026-10-11

[PD-OWN human acceptance — Option A](https://github.com/rgomids/axiom/pull/310#issuecomment-6109470563) records the current maintainer request against reviewed remote/local HEAD `c4cb5aa83fdf82acc91c9d6fc4f0a6012f04dd63`. This is an ownership acceptance, not acceptance of corrected Tasks or a formal GitHub review.

- **Delivery A:** #306 exclusively owns T01–T16 local Runtime/Profile authoring and onboarding. T16's local gate depends on T14/T15 only. It may conclude its local technical Evidence with accurate external blockers; mandatory local security/native/compatibility checks are not removed.
- **Delivery B:** X01–X07 remain a separately proposed technical delivery with owner/identity/lifecycle/Tasks and implementation authority to be established independently. No assignment to #306/#276 or new Issue. Existing A→B producer dependencies remain; no local T unit depends on any X unit.
- **GATE-I306-INTEGRATED:** separate Not-evaluated review gate consuming A/T16 + B/X07 + G-3/#276/#278 receipts and #305/#277/G-4 when required. Covers Workflow2/compiler/Execution/graph/default-sdd r2, exact proven model/effort/no-override availability, dispatch and real E2E. It is not Task24, a second dispatcher, permission or automatic acceptance.
- Stable interfaces are documented: local active2/tombstone/v1 codecs/digests/generation; conditional authoring/result; Profile-first projection/proof-source contracts; policy/readiness; skill/onboarding. Local injected consumer tests/unknown blockers cannot certify future compiler/snapshot/dispatch support.
- [Tasks](tasks-306-local-runtime-profiles.md) retain 23 unique IDs and Not-started status. Only T16→X07 is removed from Task prerequisite edges; its integrated assertion moves to the separate gate. T03's local producer/interface wording clarifies its already bounded scope without promising Delivery B operations.
- All six reconciled accepted files remain byte-identical to pre-edit c4cb5aa; exact approved 823bce3 Git blobs and the table above are preserved. No accepted ADR decision/supersession, historical Specification, schema, fixture, source or prior Planning Evidence is edited. Only Tasks, this Evidence and affected index status references change.
- No delegation: the bounded coupled correction is smaller and safer to integrate/check directly in the Orchestrator; no model or effort selection is simulated.

## Current concurrent PR and integration handoff

PR #310 was OPEN/Draft at `c4cb5aa`, branch `feat/306-local-runtime-profiles`, base main `5d52327b0367ba08d872b50e804df5c7d63a3e89`; clean local state matched remote before edits. All prior commits remain.

Read-only PR #311 observation: HEAD `52971d0f63b67427107011c874acf3d742e0e800` now proposes `docs/decisions/0025-public-stage-dispatch-and-capability-proof.md`. Its Status explicitly says Proposed/not accepted and explains renumbering from the earlier 0024 proposal. The numerical collision in the historical snapshot above has already been removed by that PR's own work; accepted #306 ADR-0024 path/receipt stays unchanged. ADR-0018 identity-by-path and reciprocal metadata remain valid. Shared index/reference changes still need reconciliation at eventual integration; no #311 mutation or proposal acceptance here.

#305 retains policy authoring; G-3 retains positive proof acquisition; #276 retains public dispatch/scheduler/coordination; #277 retains delivery/rework; #278 retains actual E2E acceptance. X05 supplies only the separately authorized snapshot/effort guard seam; no duplicated proof acquisition or dispatcher. The gate requires exact matched provenance, explicit authority and separate native classification; unknown/stale/missing evidence blocks full readiness.

## Current documentary validation

- Task IDs/fields/AC traceability/DAG — PASS: same 23 unique IDs, ten fields each, all Not started; D1–D5/LA/PE/ON/AC/IA markers present; acyclic. The only prerequisite delta is removal of T16→X07; every A dependency stays within A and every B prerequisite is retained. Integrated gate is separate/Not evaluated.
- Local Markdown targets/anchors — PASS: 133 targets/two anchors. Index references updated only for accepted ownership; ADR-0018 reciprocal metadata untouched and repository ADR governance PASS.
- Preservation — PASS: all six reconciled accepted files byte-identical to c4cb5aa; original six 823bce3 receipt rows/blob identities unchanged. Prior Planning Evidence and historical Spec004/007/ADR22 unchanged; no source/schema/fixture changes.
- Repository validator — PASS (25 ADRs; vendor behavior SKIPPED/UNVERIFIED). Diff scope/whitespace — PASS: four authorized Markdown paths only. Staged sensitive-file and scoped gitleaks — PASS, no detected leaks. PR metadata — PASS (Related-Issues #306; Completes-Issues none).
- PR metadata/head/read-back/CI classification — final exact revision and Tasks blob recorded in the new PR review comment after publication.

No production/Runtime test, authenticated vendor probe/inference, operational migration, local configuration or Provider change runs in this phase. Documentary checks prove document consistency only.

## Current limits and stop

PD-OWN Option A is accepted. The corrected artifact remains **Proposed — Awaiting Human Approval — Tasks**. Real model/auth/effort support, Delivery B implementation, #305/G-3/#276/#277/#278/G-4 receipts and nominal Delivery B owner/future lifecycle remain external; they do not prevent Delivery A's independently bounded Evidence. No private-document availability pendency. All Tasks Not started, gate Not evaluated; no new Issue, Ready transition, formal review, merge, release, operational state or MVP acceptance.
