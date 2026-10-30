# Specification 015 — Minimum viable operational workflow

Status: **Proposed — correction-03**. Specification + Research only; no implementation authority or architecture acceptance.
Technical baseline: main `440e44a68638e08e901fd3f87f5b15cd395a6365`; correction-03 starts from confirmed PR HEAD `3558bba51b0af11091f3d41305af264ba060d7bc` on [PR #366](https://github.com/rgomids/axiom/pull/366).
Authority: [accepted Intake](https://linear.app/rgomids/issue/AXM-42#comment-274fd7e0) plus the **new human product direction** [7a717e9f](https://linear.app/rgomids/issue/AXM-42#comment-7a717e9f). [AR-01](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5708094263) supersedes the favorable assessment under the previous baseline. F1–F4 corrections and historical receipts remain archival facts.
Inspectable code inventory, deletion candidates, contract clauses and evidence gaps: [research](../../research/axm-42-minimum-viable-workflow-investigation.md).

## Problem

Axiom's operational path has accumulated separate policy/plan objects, execution ledgers, coordination and delivery lifecycles, and repeated admission/preview ceremonies. Hiding those structures behind a shorter CLI retains their maintenance and recovery costs. Their existence, tests or historical acceptance do not establish that they remain necessary.

## Goal

**Prefer delete to hide/internalize.** Remove or consolidate redundant concepts, services, stores, persistence formats and commands while delivering the smallest useful control plane: a configured Project with 1..N Repositories, a Work Item, default/custom Workflow with optional SDD template, Codex/Claude dispatch and collaboration, automatic eligible progression, truthful results/error recovery, and optional authorized reviewed delivery.

A mechanism may survive only with evidence of its distinct Axiom-owned invariant, the concrete failure if removed, and why a meaningfully simpler mechanism cannot satisfy it. Missing evidence means a bounded investigation, not indefinite preservation.

## Non-goals and boundaries

- No implementation, new ADR, accepted-record rewrite, new Work Item/PR, merge, release or live inference in this correction.
- No model capability certification, predicted vendor compatibility or code-quality guarantee; no universal preflight/test battery for the MVP.
- No plugin framework, new infrastructure or distributed orchestration guarantees.
- No loss of authorized target boundaries, secret protection, real process cancellation, uncertain-effect visibility or material-effect authorization.
- No permanent dual-mode persistence or compatibility wrapper as the default migration strategy.

## Architecture target (Proposed)

Use these responsibilities with the fewest representations; the names below do not require separate services, tables or stores.

| Responsibility | Minimum representation and ownership |
|---|---|
| Project | Durable portable repository associations, selected Workflow and explicit user restrictions; machine-local executable/workspace/credential bindings remain outside portable content |
| Workflow | Durable definition of stages, agents, dependencies, inputs/outputs and authorized effect bounds; full SDD is an optional template |
| Work Item | Traceable identity and intent, linked to an existing supported source without mandatory create-via-Provider; no new Provider abstraction |
| Execution | **One canonical durable state** for Workflow binding, stage progress, attempts, outcomes, authorization facts and repository-scoped artifact references |
| Agent attempt | Nested Execution identity/status/process correlation and confirmed/unknown effects, rather than another independently versioned lifecycle/store |
| Messages and artifacts | Attempt-owned message log and artifact references associated with Execution; content is not authority and does not require duplicate graph/coordination objects |

Workflow stages and agent definitions drive dispatch directly. A transient in-memory schedule may resolve dependencies; no independently persisted Stage Plan, graph proposal lifecycle, Validator Policy or replacement Project-default policy object is required. Keep only authorized effect bounds at their owner and enforce them at dispatch/effect boundaries without duplicating policy objects.

New single-agent and multi-agent starts MUST use the same Execution state model and scheduler path. Do not fabricate graph children for a single agent. Delete the redundant ledger/store/codec paths after bounded legacy retirement; merely making them private fails this specification. Exact encoding and transaction primitives are reversible Plan choices if they meet these contracts.

## Normal operation

1. Configure Project repositories, Workflow and local Runtime target once; resolve unambiguous defaults or ask for a missing user selection. Do not mandate a Profile generation, complexity taxonomy or model/effort capability catalogue.
2. From a Work Item, **one run action** resolves current Axiom-owned inputs/authority and starts an attempt. Internal atomic publication/concurrency protection replaces repeated public preview/apply/digest/expected-revision checkpoints. Optional inspection must not become another run prerequisite.
3. Observe real process/output/error facts; make eligible successors runnable automatically. The ready default needs only a work stage and completion outcome; additional stages/gates must serve configured work or actual authority. Full SDD remains selectable.
4. Exchange attempt-owned messages between Codex and Claude through one channel. Authenticate sender/recipient scope at process boundaries; no repeated grant/receipt ceremony for each message or intermediate handoff object. Messages never grant permissions or satisfy human gates.
5. Show state, logs, artifact references, confirmed effects, unknown effects and next safe action. Exit zero or an agent assertion alone does not prove stage completion or human acceptance.
6. Expose one delivery/rework path over Execution results and review findings. Remove independent Packet/Publication/Decision/Rework lifecycle prerequisites. Record actual authorization and observed push/PR results in Execution; rejection preserves prior evidence and links subsequent work without another public object ceremony. No silent merge, release or Work Item closure.

Axiom validates its own input identity, repository targets and real permissions. User-selected Runtime restrictions are intent, never vendor capability certification. Vendor authentication, model/effort rejection, unavailable executable or other spawn/vendor errors must produce actionable observed diagnostics during the authorized attempt; no mandatory compatibility/auth probe before normal dispatch. Do not claim external sandbox, network or vendor behavior is guaranteed.

Optional stage-defined validation commands may run only with authority for their actual effects. They provide concrete output checks without an independent Policy publisher/store/digest. A stage without such commands must not require a placeholder Policy.

Work Item linking remains the normal traceability path. A provider-free local identity is not required: justify a concrete blocked use case and reconcile existing identity contracts before adopting it. Resolve the research gap in S1; do not build speculative provider orchestration.

## Invocation ownership and precedence (Proposed; F5)

For new executions, the **frozen per-attempt invocation snapshot within Execution** is the sole effective Runtime/model/effort authority. A named Profile is an optional reusable input, not a mandatory generated artifact. Selecting Profile B instead of A selects B's whole configured tuple; directly requesting another effort while using A is an override of A's input, not selection of a different Profile. Under accepted ADR-0024 the latter is forbidden. This proposal deliberately changes that boundary and requires explicit human material acceptance before implementation.

Resolve each field in this order: explicit run selection > selected Workflow agent default > machine-local invocation default. A Profile reference at any level supplies its tuple at that level; explicit fields at that same level replace the corresponding Profile defaults. Reject conflicting values at the same precedence and ambiguous/missing selections; never rank candidates or infer effort from complexity. Effort is either an explicit opaque adapter token or explicitly selected `runtime-default` (no Axiom effort selector, no asserted concrete vendor level). Missing effort is not implicit runtime-default. Record the safe input provenance, selected Profile identity/digest when used, and resulting tuple; an overridden tuple must not be presented as the unchanged Profile.

For example, configure local Runtime/model plus `runtime-default` once, then request an explicit effort for a run without creating a Profile. Alternatively select an existing Profile and explicitly replace its effort for that run. Both freeze the effective choice before dispatch under actual permission. Snapshot edits are forbidden; changed defaults or a new explicit selection require a new attempt with new provenance, never rebinding an existing attempt or blindly retrying unknown effects. Retry without a new selection retains the prior tuple and rechecks current authority. No inherited argv/env/vendor setting may silently win over the frozen choice: ambiguous/conflicting effective sources produce a sanitized diagnostic. Vendor rejection is an observed failure; no automatic model, effort, Profile or vendor substitution.

This is a proposed partial supersession of ADR-0024 Decision/Consequences and Spec 007 #306 PE-01–PE-05/PE-05a and AC-PE1/AC-PE3 **only where they require sole Profile effort, forbid direct input overrides, require two Profiles for different efforts, or project only Profile provenance**. Preserve truthful identity, bounded safe source inspection, immutable provenance and no fallback. Predictive proof/preflight removals are separately mapped in Research, not implied by this ownership change. Spec 007 WF-008 and its schema-2/Stage Plan annotations must point to the new prospective contract when accepted; do not re-enable historical Workflow schema-1 effort semantics as the new format.

Compatibility: old Workflow/Profile/Execution/graph bytes and meanings remain unchanged; no resume can manufacture snapshot provenance or reinterpret historical requested effort as a current default. Existing explicitly selected Profiles may supply unchanged defaults to a **new** attempt with recorded lineage; no auto-generation, silent conversion or rebinding. Existing attempts remain governed by their original frozen contract pending the bounded S1–S3 disposition below. Unsupported old state blocks safely rather than enabling a parallel legacy engine. Exact encoding/version is a Plan choice after material acceptance.

Alternatives: sole Profile ownership preserves ADR-0024 but requires extra Profiles for each effort choice; automatic ephemeral Profile generation hides that ceremony while retaining redundant ownership. The recommended snapshot rule removes that requirement at the cost of explicit prospective supersession and per-attempt provenance. Ownership/compatibility becomes harder to change after persisted attempts exist. This recommendation is **not accepted architecture**.

## Safety proof obligations

These are required outcomes, **not** mandates for historical leases, epochs, receipts or separate reconcile entities:

- Portable configuration, logs, messages and artifact references expose no secrets.
- Writes and messages cannot escape the authorized repository/attempt targets; authorization cannot be expanded by agent output or stale state.
- Stop/cancel controls the owned process scope and reports observable uncertainty honestly, including process identity reuse/escape limitations.
- Concurrent/stale writers cannot authorize conflicting work. Interrupted dispatch retains sufficient attempt/process identity to distinguish confirmed, failed and unknown effects.
- Unknown external mutations block blind automatic replay. Recovery reports what is known and the smallest safe action; it never promises exactly-once execution.
- Push/PR/merge/release, destructive actions and material cost require applicable explicit authority. Existing bounded authority need not be requested again ceremonially; changes to the authorized target/effect require new authority.

## Acceptance

### Specify completion (this correction)

The short normative contract and research A–L inventory must identify removals/consolidations, exact evidence, ownership, simpler alternatives, concrete counterexamples, migration and confidence. Each unresolved safety claim has a narrow S1 experiment and decision deadline. The deletion budget names what Plan must measure, without inventing reductions. Proposed contract supersession is explicit; accepted history is unchanged. Docs checks do not constitute product acceptance.

### Product acceptance (later authorized implementation)

| ID | Observable pass condition |
|---|---|
| P-AC1 | Configured Project + linked Work Item starts the ready default through one run action without Plan/Policy/graph ceremony; deleted services/stores/commands are evidenced, not merely hidden |
| P-AC2 | Real single-agent attempt produces truthful durable state/logs/artifacts and actionable failure/unknown-effect recovery |
| P-AC3 | Codex and Claude exchange a question/answer, unblock dependent work and advance eligible stages automatically using one Execution/message path |
| P-AC4 | A Project with two Repositories exposes repository-specific outcomes/artifacts under the same Execution |
| P-AC5 | Safety obligations pass targeted fault/cancel/concurrency/unknown-effect checks; no fabricated CI, provider outcome or success |
| P-AC6 | Review/rework and optional PR use one path, preserving evidence and explicit effect authority without delivery-object ceremony |
| P-AC7 | Plan baseline versus final inventory demonstrates actual reductions of redundant commands/flags, mandatory fields/steps, independent objects, stores/formats and associated code/tests; exceptions meet the three-part survival proof |

**Negative tests:** trivial dispatch creates no Policy/Plan store or format; new starts create no duplicate Execution ledger; no repeated manual graph preview/apply; no mandatory vendor-compatibility/auth preflight; optional delivery requires no independent Packet/Publication/Decision lifecycle. Denial of unauthorized writes/messages/effects and refusal of blind replay must still work after removal.

Real two-repository Codex+Claude E2E and R2/R3 qualification: **NOT_RUN**, pending later explicit execution authority. Synthetic checks alone cannot pass P-AC2–P-AC6's real execution scenarios.

## Delivery slices and bounded migration

| Slice | Required outcome and deletion evidence |
|---|---|
| S1 | Remove first-dispatch blockers **and obsolete policy/plan/admission/storage/CLI abstractions**; establish canonical Execution for new starts; demonstrate first real single-agent dispatch when authorized (P-AC1/2/5). Close research gaps before committing the dependent implementation contract; unresolved material choices return for human decision, not a permanent keep |
| S2 | Minimal cross-Runtime messaging, dependent scheduling and automatic eligible stage transitions (P-AC3); delete duplicate coordination and stage/gate representations supported by evidence |
| S3 | Unified repository outcomes and simplified authorized optional PR/rework (P-AC4/6); retire duplicate delivery lifecycles |
| Migration, only if indispensable | Inventory actual preexisting records in Plan without importing private data. Preserve original bytes as inert archives; stop new writes to retired formats at S1 cutover. Resolve active/uncertain attempts before conversion or leave them explicitly blocked for safe disposition. Convert inspectable history to canonical references with verified lineage; retire operational legacy readers/writers by completion of S3, before MVP qualification. If unsafe, block cutover for a concrete decision rather than extend dual mode indefinitely |

Rollback may preserve archived bytes and restore the prior binary before cutover; it must not replay external effects or resume legacy writers against new-format executions. Plan must establish backup/read-back/rollback evidence and the actual population needing migration; no migration service is assumed.

## Contract evolution and real acceptance gate

The [research contract map](../../research/axm-42-minimum-viable-workflow-investigation.md) identifies precise clauses and dependent amendments. Proposed implications:

| Accepted contract | Proposed change; what remains |
|---|---|
| Spec 007 WF-005/006/007/008/010/011, stage/ledger/interface/delivery sections | Partially supersede separate StagePlan/PlanRef, validator-policy authority objects, dual single/graph execution, mandatory phase/gate profile and preview-based run/delivery ceremonies. Retain Workflow intent, Work Item/repository identity, immutable provenance, actual authority and honest outcomes |
| ADR-0009 | Partially supersede graph-specific execution/coordination architecture with one Execution and transient dependency schedule; retain dependency ordering and attempt ownership |
| ADR-0020 | Partially amend binding publication/retention mechanisms where multi-object ceremonies are replaced; retain exact Workflow provenance so configuration edits cannot silently retarget running work |
| ADR-0026 | Supersede separate successor-Graph persistence/admission for new runs; historical Graph bytes remain archives, not a permanent second operating model |
| ADR-0027 | Supersede independent single-stage operational ledger/authority codec for new runs; attempts belong to canonical Execution |
| ADR-0028 | Retain the limited observable process-control boundary; amend any required historical stop-receipt representation only where consolidated. No stronger process containment guarantee |
| ADR-0032 | Preserve no-certification direction; amend remaining mandatory admission/configuration ceremony where direct authorized execution and observed error suffice |
| ADR-0024 and Spec 007 #306 | Partially supersede sole Profile ownership/no direct override and Profile-only projection for new attempts as specified above; retain old meanings, immutable effective identity and no silent substitution |
| Direct dependencies (including ADR-0025 and protected publication ADR-0007) | Amend only Profile/dispatch/authority/persistence clauses actually contradicted by consolidation, as enumerated in research. Keep secret/target/publication safety outcomes; no wholesale implicit supersession |

**Recommendation:** accept one coherent structural replacement contract, with the smallest explicit amendment/supersession package required by ADR-0018, before implementation. Alternative: incremental internalization retains multi-object recovery/maintenance cost and fails the new direction. Trade-off: consolidation requires bounded migration and may reduce independently addressable historical interfaces; persistence cutover and public command retirement become costly to reverse. Encoding, helper layout and flag coalescing remain TECHNICAL_LOCAL, not new product gates.

The human has accepted the **direction**, not this exact architecture. Next gate is **re-review of revised Spec 015 and explicit human material acceptance of its architecture/contract evolution**. No Plan/Implement, merge/release/delivery acceptance or AXM-12 unblock is claimed here.

## Constitution check

Intent precedes implementation; reduction evidence is deterministic where possible; Project remains distinct from Repository. Safety, truthful evidence and human authority remain mandatory outcomes. Accepted contracts/history are preserved until explicit evolution. No new constitutional conflict is proposed; the material architecture decision remains pending acceptance.
