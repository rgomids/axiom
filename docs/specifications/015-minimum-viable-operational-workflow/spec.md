# Specification 015 — Minimum viable operational workflow

Status: **Proposed — correction-04**. Specification + Research only; no implementation authority or architecture acceptance.
Technical baseline: main `440e44a68638e08e901fd3f87f5b15cd395a6365`; correction-04 starts from fetched PR HEAD `cc709b7c0ecf4ba9b796d13ec692be0c55660a1d` on [PR #366](https://github.com/rgomids/axiom/pull/366).
Authority: [accepted Intake](https://linear.app/rgomids/issue/AXM-42#comment-274fd7e0) plus the **new human product direction** [7a717e9f](https://linear.app/rgomids/issue/AXM-42#comment-7a717e9f). [AR-01](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5708094263) supersedes the favorable assessment under the previous baseline. F1–F4 corrections and historical receipts remain archival facts.
Controlling review: [2026-10-30 five scoped decisions](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5719376804). These decisions authorize this reconciliation only; the complete replacement architecture remains Proposed.
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
- No historical Execution migration/conversion, old-attempt resume, active legacy engine, conversion service or required historical-reader UI.

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

Each Workflow stage MUST explicitly support one or more agents, each with configurable Runtime, model, reasoning effort and instructions. Parallel subagents are REQUIRED functionality. Profiles are optional reusable inputs; Axiom never selects a best model automatically by capability or complexity. Workflow stages and agent definitions drive dispatch directly. A transient in-memory schedule may resolve dependencies; no independently persisted Stage Plan, graph proposal lifecycle, Validator Policy or replacement Project-default policy object is required. Keep only authorized effect bounds at their owner and enforce them at dispatch/effect boundaries without duplicating policy objects.

New single-agent and multi-agent starts MUST use the same Execution state model and scheduler path. Do not fabricate graph children for a single agent. Delete the redundant ledger/store/codec paths at the safe breaking cutover; merely making them private fails this specification. Exact encoding and transaction primitives are reversible Plan choices if they meet these contracts.

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

Resolve each field in this order: explicit per-agent run override > explicit selected Workflow agent configuration > explicitly configured machine-local default. A Profile reference at any level supplies its tuple at that level; explicit fields at that same level replace the corresponding Profile defaults. Reject conflicting values at the same precedence and ambiguous/missing selections; never rank candidates or infer effort from complexity. Effort is either an explicit opaque adapter token or explicitly selected `runtime-default` (no Axiom effort selector, no asserted concrete vendor level). Missing effort is not implicit runtime-default. Freeze the effective instructions alongside Runtime/model/effort for each agent attempt. Record the safe input provenance, selected Profile identity/digest when used, and resulting tuple; an overridden tuple must not be presented as the unchanged Profile.

For example, configure local Runtime/model plus `runtime-default` once, then request an explicit effort for a run without creating a Profile. Alternatively select an existing Profile and explicitly replace its effort for that run. Both freeze the effective choice before dispatch under actual permission. Snapshot edits are forbidden; changed defaults or a new explicit selection require a new attempt with new provenance, never rebinding an existing attempt or blindly retrying unknown effects. Retry without a new selection retains the prior tuple and rechecks current authority. No inherited argv/env/vendor setting may silently win over the frozen choice: ambiguous/conflicting effective sources produce a sanitized diagnostic. Vendor rejection is an observed failure; no automatic model, effort, Profile or vendor substitution.

This is a proposed partial supersession of ADR-0024 Decision/Consequences and Spec 007 #306 PE-01–PE-05/PE-05a and AC-PE1/AC-PE3 **only where they require sole Profile effort, forbid direct input overrides, require two Profiles for different efforts, or project only Profile provenance**. Preserve truthful identity, bounded safe source inspection, immutable provenance and no fallback. Predictive proof/preflight removals are separately mapped in Research, not implied by this ownership change. Spec 007 WF-008 and its schema-2/Stage Plan annotations must point to the new prospective contract when accepted; do not re-enable historical Workflow schema-1 effort semantics as the new format.

Cutover: only new-format Executions may run after the pre-MVP breaking cutover. No migration, conversion, old-attempt resume or active legacy engine is required or permitted by this proposal. Preserve raw old state as an inert backup where feasible for investigation; do not reinterpret its provenance. Previously published receipts remain historical facts, not runtime compatibility obligations. Existing explicitly selected Profiles may supply inputs to a new attempt without conversion/rebinding of an old Execution. Before cutover inventory actual active/unknown attempts; safely stop owned processes where possible, or BLOCK cutover when effects cannot be established. Never replay unknown external mutations. No historical-reader UI or migration service is required. Exact new encoding/version remains a Plan choice after material acceptance.

Alternatives: sole Profile ownership preserves ADR-0024 but requires extra Profiles for each effort choice; automatic ephemeral Profile generation hides that ceremony while retaining redundant ownership. The recommended snapshot rule removes that requirement at the cost of explicit prospective supersession and per-attempt provenance. Ownership/compatibility becomes harder to change after persisted attempts exist. This recommendation is **not accepted architecture**.

## Parallel work and integration

For two or more agents editing the same Repository, recommend an isolated Git Worktree and independent branch per agent. The principal Orchestrator owns the integration branch, serializes cherry-picks/merges, resolves conflicts explicitly, runs post-merge checks and records final integration status before marking work integrated. Worktrees do not guarantee conflict-free merges or protect shared external state. Preserve aggregate CAS, runner fencing, attempt correlation and effect authorization across parallel work. This adds no Git management subsystem or delivery slice.

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

The short normative contract and research A–L inventory must identify removals/consolidations, exact evidence, ownership, simpler alternatives, concrete counterexamples, cutover and confidence. Each unresolved safety claim has a narrow S1 experiment and decision deadline. The deletion budget names what Plan must measure, without inventing reductions. Proposed contract supersession is explicit; accepted history is unchanged. Docs checks do not constitute product acceptance.

### Normative technical completion and later Product acceptance

During Specify, Plan, Tasks, Implement and review, automated validation is limited to unit tests, isolated deterministic stub/fault simulations and ordinary non-E2E repository/document checks. No automated real product E2E, live Codex+Claude invocation as a validation test, actual multi-repository external mutation or test PR publication. Documentation publication on this authorized PR is not a product test.

**Implementation Complete** means all S1–S3 code plus unit/isolated technical checks are complete. Slice completion requires its scoped technical evidence, never a real vendor demo. **Product Accepted** is a separate, later maintainer manual end-to-end journey after all S1–S3 code is complete: configure Project, bind Repositories, configure Workflow and agents/Runtimes, create/link Work Item, start/run, observe progress/logs, inspect trustworthy Evidence/results and diagnose real errors, including collaboration and review/rework. Synthetic checks establish technical contracts, not that manual acceptance.

The P-ACs below remain later manually verifiable product requirements; P-AC5/7 also require deterministic technical evidence.

| ID | Observable pass condition |
|---|---|
| P-AC1 | Configured Project with per-stage/per-agent Runtime, model, effort and instructions, optional Profiles and no automatic best-model selection + linked Work Item starts the ready default through one run action without Plan/Policy/graph ceremony; deleted services/stores/commands are evidenced, not merely hidden |
| P-AC2 | Real single-agent attempt produces truthful durable state/logs/artifacts and actionable failure/unknown-effect recovery |
| P-AC3 | Codex and Claude parallel subagents exchange a question/answer, safely integrate same-repository work under the principal Orchestrator, unblock dependent work and advance eligible stages automatically using one Execution/message path |
| P-AC4 | A Project with two Repositories exposes repository-specific outcomes/artifacts under the same Execution |
| P-AC5 | Safety obligations pass targeted fault/cancel/concurrency/unknown-effect checks; no fabricated CI, provider outcome or success |
| P-AC6 | Review/rework and optional PR use one path, preserving evidence and explicit effect authority without delivery-object ceremony |
| P-AC7 | Plan baseline versus final inventory demonstrates actual reductions of redundant commands/flags, mandatory fields/steps, independent objects, stores/formats and associated code/tests; exceptions meet the three-part survival proof |

**Negative tests:** trivial dispatch creates no Policy/Plan store or format; new starts create no duplicate Execution ledger; no repeated manual graph preview/apply; no mandatory vendor-compatibility/auth preflight; optional delivery requires no independent Packet/Publication/Decision lifecycle. Denial of unauthorized writes/messages/effects and refusal of blind replay must still work after removal.

Real two-repository Codex+Claude journey and R2/R3 qualification: **NOT_RUN**; reserved for later maintainer manual Product acceptance after S1–S3. No real E2E is required at individual slice completion.

## Delivery slices and safe breaking cutover

| Slice | Required outcome and deletion evidence |
|---|---|
| S1 | Remove first-dispatch blockers **and obsolete policy/plan/admission/storage/CLI abstractions**; establish canonical Execution for new starts; verify single-agent dispatch contracts with isolated stubs (technical coverage for P-AC1/2/5). Close research gaps before committing the dependent implementation contract; unresolved material choices return for human decision, not a permanent keep |
| S2 | Required parallel subagents, cross-Runtime messaging, dependent scheduling and automatic eligible stage transitions, verified with isolated stubs (technical coverage for P-AC3); delete duplicate coordination and stage/gate representations supported by evidence |
| S3 | Unified repository outcomes and simplified authorized optional PR/rework, verified with isolated repository/Provider stubs (technical coverage for P-AC4/6); retire duplicate delivery lifecycles |

Keep exactly **S1, S2, S3**. Plan/Tasks may order internal checks with explicit per-slice value, dependencies and scope freeze; do not create extra slices, Linear cards, Specs or recurring human gates for technical local choices. Optional discoveries become deferred findings. Only a new material architectural/product/design conflict returns for human decision.

The cutover is an S1 safety obligation, not another slice: inventory actual active/unknown attempts without importing private state, preserve raw backup where feasible and stop safely or BLOCK before switching to new-format-only Executions. Delete obsolete runtime readers/writers and conversion promises; retaining published receipt history does not require old runtime compatibility. Plan evidence covers inventory, backup feasibility, safe-stop/blocked disposition and isolated refusal of old-attempt resume and unknown-effect replay. Rollback must not replay effects, restart old attempts or point a prior binary at new-format state; no old-format operational fallback is promised.

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
| ADR-0024 and Spec 007 #306 | Partially supersede sole Profile ownership/no direct override and Profile-only projection for new attempts as specified above; retain inert historical meanings, immutable effective identity and no silent substitution |
| Historical compatibility: ADR-0024 Compatibility scope, Spec 007 #306 PE-04 / version matrix / compatibility corpus; ADR-0027 Decision bullets 5/9 | Propose replacing required operational old-format decoding/inspection with the pre-MVP new-Execution-only cutoff. Preserve historical bytes/meanings in feasible raw backup and published receipts; no conversion, resume or legacy engine. Inventory active/unknown attempts and safely stop or BLOCK before cutover; preserve protected publication, monotonic fencing and no replay |
| Direct dependencies (including ADR-0025 and protected publication ADR-0007) | Amend only Profile/dispatch/authority/persistence clauses actually contradicted by consolidation, as enumerated in research. Keep secret/target/publication safety outcomes; no wholesale implicit supersession |

**Recommendation:** accept one coherent structural replacement contract, with the smallest explicit amendment/supersession package required by ADR-0018, before implementation. Alternative: incremental internalization retains multi-object recovery/maintenance cost and fails the new direction. Trade-off: consolidation requires a safe breaking cutover and may reduce independently addressable historical interfaces; persistence cutover and public command retirement become costly to reverse. Encoding, helper layout and flag coalescing remain TECHNICAL_LOCAL, not new product gates.

The human has accepted the **direction**, not this exact architecture. Next gate is **re-review of revised Spec 015 and explicit human material acceptance of its architecture/contract evolution**. No Plan/Implement, merge/release/delivery acceptance or AXM-12 unblock is claimed here.

## Constitution check

Intent precedes implementation; reduction evidence is deterministic where possible; Project remains distinct from Repository. Safety, truthful evidence and human authority remain mandatory outcomes. Accepted contracts/history are preserved until explicit evolution. No new constitutional conflict is proposed; the material architecture decision remains pending acceptance.
