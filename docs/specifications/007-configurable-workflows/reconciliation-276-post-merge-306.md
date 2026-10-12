# #276 post-merge reconciliation — CR-012 / CR-013

## Post-decision update — 2026-10-12 (current)

PD-R276-01/**B direction approved** at the exact
[maintainer decision receipt](https://github.com/rgomids/axiom/pull/311#issuecomment-6125676685).
The alternatives and original Evidence below are preserved as historical
pre-approval analysis; their `Proposed / HUMAN_DECISION_REQUIRED` labels describe
the earlier observation and are not current decision-state labels.

A new detailed [ADR-0026](../../decisions/0026-successor-stage-dispatch-graph-format.md)
and [successor Specification amendment](amendment-276-successor-graph-proposed.md)
are **Proposed, awaiting separate exact-artifact acceptance**.
The existing 44 Tasks are reconciled as a new **Proposed** Task revision;
this does **not** authorize implementation, does not alter #306 and does not
supersede accepted ADR-0025 until ADR-0018 governance is completed.

CR-012 remains open pending acceptance/re-review of the detailed contracts
and their aligned Tasks. CR-013 remains resolved.
PR #311 stays Draft; Phase 1 remains accepted historical baseline,
Phase 2 proposed, Phase 3 not authorized.

## Status and authority — 2026-10-12

**PD-R276-01: Proposed / HUMAN_DECISION_REQUIRED.** This packet is analysis and
proposed reconciliation, not acceptance or authorization of a new format.
Phase 1 Accepted; Phase 2 Proposed / Awaiting Human Approval; Phase 3 Not Authorized;
PR #311 Draft. No #306 Task, accepted ADR/amendment or production code changes.

Observed PR #311 HEAD `3c2f436823740ce97c5616c0cb3106b9fea4dbe0`, base main
`5e1c995ab57ed3fbd54e5b72288607140d031e7d`. PR #310 MERGED at
2026-10-12 03:05:33 UTC (00:05:33 America/Sao_Paulo), merge `5e1c995`,
accepted source HEAD `dd6fc596af5389e56a9d4a7287e328d8dab8c871`.
The merge is an ancestor of #311 HEAD. Repository visibility was verified public.

[Phase 1 receipt](https://github.com/rgomids/axiom/pull/311#issuecomment-6109867861)
fixes Plan blob `13776ae4e5923b2c30439b2b12586b202d864c11`, amendment blob
`3c931f0d6005a8abfc870b5febf414606d2ffdaa`, ADR-0025 blob
`1da38d6905df8ad34c07d70b2204164b5aed0634`, and PD-1–PD-17.
[PD-OWN](https://github.com/rgomids/axiom/pull/310#issuecomment-6109651213)
keeps all 23 #306 Tasks with #306;
[Task acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6109821395)
and [technical acceptance](https://github.com/rgomids/axiom/pull/310#issuecomment-6117000624)
are independent receipts. They do not authorize #276 implementation.

Latest published #311 review is [5485913782](https://github.com/rgomids/axiom/pull/311#pullrequestreview-5485913782),
HEAD `24708aa`, CR-010/CR-011. CR-012/CR-013 are the preview findings supplied
in the current human request; no corresponding published review was found in
PR reviews, issue comments or inline comments. Their identifiers are retained
without inventing a remote review receipt.

## A. CR-012 — cause and executable contract

PD-16 and ADR-0025 Consequences accepted one joint Graph2 revision with #306/X05.
The Proposed T01 handoff never became part of X05's accepted implementation.
Completed X05 supplies Profile binding/effort checks, not the #276 persistence
contract. Reassigning more work to that completed Task would contradict its
accepted scope/history. Its delivery is not reopened.

| Source at reviewed HEAD | Actual contract / consequence |
| --- | --- |
| `internal/executiongraph/graph.go:22–98` | Parent IDs/revision/proposal and authority digests/effects; child envelope; attempts with status, timestamps, result/output, exit, ambiguity/cancellation-seen. No #276 field set. |
| `graph.go:234–295` | ValidGraph accepts formats 1/2, requires Profile controls for2 and excludes resolvedProfile for1. |
| `graph.go:307–335` | Closed bounded JSON; unknown members rejected; Graph2 additionally strict duplicate-key checks. Encode is struct-order JSON plus newline; StorageRevision hashes exact bytes. |
| `internal/local/graph_store.go:53–85,113–118,137,166` | Protected CAS on exact-byte SHA-256; common graphs/v1 storage namespace, filename SHA-256(parent ID), independent from payload format. Directory v1 is not Graph format1. |
| `internal/executiongraph/planner.go:203`, `integration.go:368`, `acceptance.go:161` | Proposal, envelope and Evidence graph hashes bind serialized content. New fields are not digest-neutral. |
| `internal/graphapplication/local.go:51,101` | Current production composition/dispatch requires Graph2; any successor requires explicit consumer changes. |
| `internal/executiongraph/scheduler.go:99,171,237–255` | Cancellation input is transient; exit0 currently succeeds without qualifying coordination; unknown/ambiguous retry refused; running-attempt fencing remains T02 work. |
| `internal/workflowcompiler/profile_chain_test.go:213–219,305–319` | Protected frozen Graph2 corpus/restart comparison; its graph has no attempts, so alone cannot prove #276 attempt behavior. |

No accepted extension registry, minor-version discriminator or sidecar protocol
for #276 exists in these sources. Existing family-version dispatch is a pattern,
not permission to adopt another version.

## B. PD-R276-01 — alternatives and recommendation

| Alternative | PD-16 / persisted formats | Digests, authority and history | Implementation / owner / tests / migration | Trade-off and human gate |
| --- | --- | --- | --- | --- |
| A. Explicit discriminated extension within Graph2 | Preserves number, but changes the closed delivered v2 contract; cannot be called compatible with old readers | Optional absent fields might preserve old encoding; populated fields change hashes and old readers reject them. No inferred authority | #276 complementary codec and all consumers need explicit discrimination, old/new reader tests and unsupported-extension refusal. No historical rewrite | Hidden incompatibility behind same major number; material accepted-contract/ownership decision required |
| B. Explicit successor graph revision | Changes joint-v2 PD-16 and ADR-0025 versioning fragment; preserves delivered Graph2 contract | Historical formats/digests unchanged. New format has new hashes and explicit dispatch authority; old readers refuse it | Proposed #276/T01 codec and consumers; T37 version matrix/corpus/restart tests. New publications only; no conversion of old graphs | Clearest closed-reader boundary; extra format and compatibility window. Human acceptance required before selecting number or implementing |
| C. Versioned bound operational records alongside Graph2 | Preserves Graph2 bytes, but changes accepted graph-resident classification/epoch contract in §3.5/A3/ADR-0025 | Binds IDs, graph/storage revision, envelope and authority digests; must atomically commit status and qualification. Missing/stale sidecar denies operation | #276 owns new record codecs/stores and multi-object recovery. Requires missing/torn/conflicting fact tests; no historical conversion | Good operational separation, but more join/recovery complexity and broader amendment; human acceptance required |
| D. Complementary joint-v2 delivery assigned to original owner | Ownership reconciliation alone cannot make strict Graph2 readers accept fields | Same incompatibility as A if v2 expanded; risks falsely redefining completed X05 | Separately approved complementary work, not reopening X05; codec and consumer tests still necessary | Reject as standalone solution: neither format compatibility nor PD-OWN expansion follows from merge |

**Recommendation: B, with complementary implementation owned by #276.** A new
explicit successor revision (candidate Graph3, number not adopted here) leaves
Graph1/2 closed definitions intact and makes dispatch admission unambiguous.
It changes PD-16 materially; this packet does not enact that change. C is viable
if the human instead prioritizes a separately versioned operational-state model,
but requires a larger change to accepted graph-resident classification semantics.
The recommendation is invalidated by an accepted extension mechanism or a new
human decision requiring Graph2-compatible operational records.

### Proposed field and invariant ownership (not accepted)

| Persistence location | Fields / semantics | Implementation owner after acceptance |
| --- | --- | --- |
| Successor Graph header | classificationPolicy identifies exact policy revision; dispatchAuthorityRef resolves protected authority; positive bounded stageConcurrency frozen at publication | T01 codec/validation; T07 authority binding; T25/T26 preview/publication |
| Successor Attempt | classificationReason closed set from §3.5; qualifyingRecordRef canonical digest, distinct from process ResultReference; leaseEpoch monotonic; PID plus process-start and group/job identity; packetDigest | T01 serialized fields; T03/T04 classifier/consumers; T06 fencing; T20 packet; T27/T43 process identity/confirmed stop |
| Successor Graph cancellation | cancelRequest binds request identity, graph revision, actor/time and exact operation authority; never implies confirmed stop | T01 codec; T29 request lifecycle; T27/T43 stop |
| Append-only facts next to Attempt, inside successor graph | attempt identity, observations digest, reconciliation classification, actor/time and operation authority; contradictory facts surfaced; no terminal-status rewrite | T01 codec; T08 semantics/store transaction; T30 apply |
| Separate dispatch-authority record | preview digest, graph/ledger and revision scope, first attempts, actor kind/time, exact effects; digest-bound reference committed with graph or single-agent ledger | T07 protected codec/store/commit; no new authority inferred from ChildEnvelope.authorityReference |
| Existing Execution3 / Evidence | Execution retains frozen Workflow/Profile binding; Evidence reports references and outcomes and never grants authority | Consume #306 Execution3; #276 Evidence consumers; no unrelated format revision |

Proposed scalar contract: policy is an opaque revision token; reasons use the
closed §3.5 reason vocabulary; record/packet/authority/observation references are
SHA-256 digests; epoch is positive uint64; process identity binds platform PID
plus opaque start and group/job identity; concurrency is a positive integer
bounded by admitted node count. Cancellation and reconciliation bind graph/attempt
identity, actor, UTC time and operation-authority digest. T01 must freeze platform
encodings, size limits, requiredness and canonical vectors before implementation; indicative Plan signatures are not wire contracts.
I-1–I-6 remain: terminal reasons required; succeeded/result_recorded binds a
same-parent/revision/attempt succeeded record; blocked binds blocker; consumers
read persisted facts only; late records cannot affect classification; graphs
without the policy marker are read-only to #276. Epoch fencing, PID-reuse checks,
atomic terminal status/reason/reference save, exact-preview drift checks and
bounded concurrency are additional mandatory invariants. Reconciliation never
promotes success. Single-agent ledger admission needs equally bound authority
and classification under an explicit ledger contract; it cannot bypass T01/T07.

### Compatibility, tests and gates before dispatch

- Keep Graph1/2 readers, original encodings, IDs, all historical fixtures and
  six-family #306 corpus/hash receipts. No fallback, reinterpretation, rewrite,
  downgrade, automatic migration or rebind. Historical graphs remain inspectable;
  #276 mutations refuse them. Newly authorized publications use the approved
  successor; historical operations retain their original bounded semantics.
- T37 adds exact old/new round-trip/hash, duplicate/unknown field, unsupported
  version, invalid/omitted classification, unresolved/wrong-attempt reference,
  late record, unknown retry, stale epoch/PID reuse, cancel uncertainty, bounded
  concurrency, torn authority/graph/ledger commit and restart regressions.
  Test old readers refuse the new family; update every mutating consumer explicitly.
- Mandatory order: human format/ownership decision → accepted bounded amendments
  and revised Tasks → separate implementation authority → T01 contract/codec →
  T04/T06/T07/T08/T09 and transitive consumers → T37 compatibility/integration.
  No new Task IDs or external #306 work required; existing DAG edges unchanged.
- Before real dispatch: IG-7 contract reconciliation and integration tests;
  supported IG-1 configuration; IG-5 tuple/PE-05a drift checks; IG-6 routes;
  G-3 fresh exact Runtime/model/effort/variant proof; preview-bound local/external
  authority; protected reservation, lease, packet, process containment and final
  drain. T40 requires its own real execution authorization and complete §9.4 proof.

### Decision requested and consequences

Accept or reject **PD-R276-01/B**: replace only PD-16's one-joint-v2/X05-serialization
fragment with explicit successor graph publication owned by #276. Preserve PD-OWN,
all #306 accepted contracts and completed Tasks, and #276 classification invariants.
After acceptance, prepare bounded evolution of Plan §3.5/§3.10/PD-16, amendment
A3/A6, ADR-0025 Consequences (ADR-0018 governance), and Part II T01/T04/T06/T07/
T08/T09/T37 plus transitive admission/dispatch consumers for renewed acceptance.
No accepted artifact is superseded now. Without decision, Phase 2 remains held
at its persistence contract gate and Phase 3 cannot begin.

## C. CR-013 and Tasks

### Unapplied Task reconciliation proposal

| Task | Proposed remaining responsibility and dependencies |
| --- | --- |
| I276-T01 | Define #276 field semantics, explicit successor-format codec/version dispatch and vectors under #276, only after PD-R276-01 acceptance; consume delivered Graph2 as immutable compatibility baseline. Replaces the unfulfilled handoff to completed X05. |
| I276-T04 | Domain classifier consumers and decode invariants consume T01's approved persistence contract, not future X05 additions. Existing T01/T02/T03 edges retained. |
| I276-T06 | #276 owns durable lease epoch and attempt process correlation; T01 prerequisite retained; X05 is a delivered Profile/effort input only. |
| I276-T07 | #276 owns protected dispatch-authority record and atomic graph/ledger binding; T01/T06 retained; delivered X04/X05 are baselines, not owners of the new record. |
| I276-T08 | #276 owns append-only reconciliation persistence and validation; T04/T06 transitively require T01 codec; no future X05 persistence delivery. |
| I276-T09 | Consume delivered X01/X03/X04/X05/T02; add refusal of Graph2 without #276 classification contract and exact successor-version admission after decision. |
| I276-T37 | Consume immutable X07 corpus and add #276 successor-format round-trip, historical hash preservation, unsupported-reader refusal and authority/restart regressions after T01/T09/T21/T26/T31. |
| I276-T16 / T21 | Consume delivered resolver and PE-05a source-inspection seam; #276 still owns acquisition/revalidation of real proof. |
| I276-T25–T31 / T33 / T36 / T38–T40 / T43 | Transitive consumers of T01: authority, supervisor, cancel, retry, integration and tests must bind the approved format. T29 owns durable cancel request; T25/T26 bind stageConcurrency. Real T40 still needs separate authority. |

This table is a proposal only. It changes no catalog scope, dependency edge,
batch prerequisite or accepted contract. Apply only after PD-R276-01 acceptance
and the required bounded contract/Task reconciliation.


The Plan's dated administrative note records delivered #306 contracts and
individual IG-1/5/6/7 state while preserving historical Part I technical text.
Part II has an administrative gate pointing here. Original Task catalog, DAG,
batches and traceability remain byte-identical to the reviewed HEAD. The proposed
T01 codec ownership and corresponding T04/T06/T07/T08/T09/T37 changes are recorded
above, not applied. T16/T21 consume the delivered proof-inspection seams;
T25–T31/T33/T36/T38–T40/T43 remain affected transitive consumers in the proposal.
No #306 Task is reopened or assigned more work. Task IDs, acceptance criteria,
Evidence requirements and authority gates remain preserved.

CR-013: **RESOLVED** for current operational status and PR metadata. CR-012:
**HUMAN_DECISION_REQUIRED**. This is reconciled planning, not dispatch delivery,
Task acceptance or an approved review. Next: human decision, revised contract
acceptance, then Code Review Guardian and separate acceptance of all 44 Tasks.

## D. Verification and limitations

A bounded read-only codec specialist audited fields, consumers and exact-byte
hashes; Orchestrator checked source references and integrated conclusions.
Delegation used inherited capability/effort and no concurrent edits or human decisions.
The table records checks executed during the preceding investigation turn;
its package-test Evidence remains valid at unchanged production HEAD. Publication
checks rerun in this follow-up are recorded separately below.
No vendor Runtime, paid inference, capability probe or operational smoke ran.
Remote #311 checks at `3c2f436`: mandatory conclusion checks SUCCESS; technical
suite jobs SKIPPED by documentation classification. These checks are not live
Runtime proof or checks of the new uncommitted documentation.


| Executed verification | Result / boundary |
| --- | --- |
| `env -u OPENAI_API_KEY -u CODEX_API_KEY go test ./internal/executiongraph ./internal/graphapplication ./internal/local ./internal/workflowcompiler` | PASS four existing package suites; controlled filesystem/fake executables, no real vendor inference |
| `scripts/validate-repository.sh .` | PASS repository bootstrap, ADR governance (26 ADRs), sensitive-file checks; optional vendor behavior remains unverified |
| In-memory Python integrity check | PASS 44 unique stable IDs, exact original dependency edges and acyclic DAG; accepted Part I technical body, amendment, ADR-0025 and #306 Tasks unchanged; local Markdown targets exist; all six-family trimmed-canonical SHA256SUMS match |
| `git diff --check` | PASS documentation whitespace |
| PR #311 metadata update and read-back | Exact intended body matched after trailing-newline normalization; HEAD/base/Draft unchanged. No review/comment published. |

The earlier investigation produced local proposals without commit/push authority.
The 2026-10-12 follow-up explicitly authorizes publication of this administrative
reconciliation and Decision Packet only; proposals dependent on PD-R276-01 are
not applied to the Task catalog or accepted contracts. No Issue/Linear transition,
implementation, dispatch,
merge or release occurred. Accepted Graph2 and all production/fixture files stay
unchanged. `gitleaks dir docs/specifications/007-configurable-workflows --no-banner --redact`
passed with no leaks. Final worktree review contains only the two scoped Markdown
artifacts; complete related diff reviewed and accepted source/fixture bytes unchanged.

## Publication scope — 2026-10-12 follow-up

Authorized publication: this Decision Packet and the Plan administrative notes.
The proposed successor-format/ownership change remains HUMAN_DECISION_REQUIRED;
no accepted PD-16, amendment or ADR evolution is enacted. Original 44 Task bodies,
DAG, batches and traceability are preserved. Earlier full Task rewrite was retained
outside the repository as an unapplied local proposal; no work was discarded.
No new delegation: existing audited Evidence remains valid and sources were
spot-checked at unchanged HEAD/main. No runtime or inference execution.

Publication uses the existing remote `feat/276-stage-agent-execution` branch.
Local checkout is detached because that local branch is attached to another
worktree; explicit `HEAD:refs/heads/feat/276-stage-agent-execution` publication
preserves that worktree. No force push or history rewrite.


Follow-up publication checks: administrative-only Plan comparison PASS; all
44 original Task bodies and prerequisite edges preserved, catalog/matrix/Mermaid
agreement and acyclic DAG PASS; accepted amendment/ADR/#306 Task bytes and six-family
corpus hashes preserved; local document targets PASS.
`scripts/validate-repository.sh .`, `git diff --check` and scoped
`gitleaks dir docs/specifications/007-configurable-workflows --no-banner --redact`
all PASS before commit. Exact
publication SHA and its separate CI receipt are reported in the PR body and
handoff; they cannot be inferred from the preceding-head checks above.
