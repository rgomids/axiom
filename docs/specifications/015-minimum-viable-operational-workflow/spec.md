# Specification 015 — Minimum viable operational workflow

Status: **Proposed** (AXM-42 Specify correction-01). Not approved. Does not authorize Implementation, accepted-ADR rewrite, merge, release or MVP acceptance.  
Investigation baseline SHA (research snapshot): `440e44a68638e08e901fd3f87f5b15cd395a6365`.  
Specify PR head at last analytical review: `74b668460ef8f53de7323fce54f2f0adc61d646d` ([PR #366](https://github.com/rgomids/axiom/pull/366)).  
Product baseline: human-approved AXM-42 Intake (workflow orchestration/coordination; Axiom owns operational behavior only) — Linear comment [`274fd7e0-e74a-4b70-9afd-14f5bf77cb6f`](https://linear.app/rgomids/issue/AXM-42#comment-274fd7e0).  
Evidence: [research investigation](../../research/axm-42-minimum-viable-workflow-investigation.md).

## Problem

The configurable Execution path can schedule and supervise Codex/Claude work, but the **operator-facing ceremony** before first dispatch exceeds the minimum needed for Axiom-owned safety: mandatory manual Stage Plan, mandatory Validator Policy digest **without a public publisher**, ten-stage SDD as the ready path, and manual advance between eligible stages. Certification was already removed (ADR-0032); remaining friction is mostly owned selection ceremony and missing product defaults — not vendor capability proof.

## Goal

Define the **minimum operational contracts** so that after Project configuration, a linked Work Item can reach a **genuine Runtime dispatch** with durable status/logs in **one natural user action**, then extend to cross-Runtime collaboration, multi-Repository outcomes and optional authorized PR — without certifying vendor capabilities or promising task success.

## Non-goals

- Plugin SDK, general policy engines, distributed control plane.
- Model-support certification, executable allowlists-as-compatibility, behavioral probes as admission.
- Weakening process safety (leases, uncertain-effect fences, secret hygiene, write confinement, supervisor/cancel/recovery).
- Replacing live R2/R3 qualification with synthetic CI alone.
- Silent supersession of accepted ADRs (explicit proposed impact only; human acceptance required before historical rewrite).

## Actors

- Operator (human): configures Project, grants external/human authority, accepts delivery.
- Axiom (Lingo): loads/validates own state, schedules stages, dispatches/supervises, records truthfully.
- Runtime agents (Codex/Claude): perform work under attempt confinement; not certified by Axiom.
- Providers (GitHub/Linear when supported): Work Item identity and optional PR/push effects.

## Slim contracts (Proposed)

### Separation of concerns (normative)

Dispatch eligibility MUST separate:

| Class | Examples | Timing |
|-------|----------|--------|
| **(a) Axiom-owned integrity** | Linked Work Item identity; Execution/Workflow binding coherence; secrets out of portable manifests; authorized target/write bounds; supervisor lease/cancel/recovery identity; communication identity; explicit authorization for sensitive/remote/destructive/costly effects | Verified **before** dispatch when they are Axiom's own input/state/permission |
| **(b) Configuration defaults** | Selected minimal Workflow; default Profile/model/effort when unambiguous or user-selected; internally derived Stage Plan / multi-agent graph | Resolved **internally** without operator-authored Plan file, public Validator Policy digest, Project-default policy *object* ceremony, or per-stage manual graph preview |
| **(c) External Runtime authentication / vendor compatibility / environment** | Vendor login success; model/effort vendor support; network/sandbox/host behavior; code correctness | **Not** speculative pre-gates. Observed on **actual authorized Runtime failure**; recorded as diagnostic + next safe action |

User-selected Runtime/Profile **restrictions** remain valid when configured as **intent/permission** (Project allowlist of permitted runtimes/profiles). They MUST NOT be repurposed as vendor-capability certification or predictive compatibility probes (ADR-0032).

### Project configuration defaults

- Portable Project: repositories (`N≥1`, product AC assumes two for S3), Work Item source binding, selected Workflow ref, optional Runtime permission allowlist (intent).
- Machine-local: Runtime bindings/Profiles, workspace preferences.
- After first successful configure, happy path uses: **minimal ready Workflow** (Intake-settled); straightforward default Profile/model/effort per Runtime when unambiguous, or explicit user selection — **no extraneous required setup**.
- **No** mandatory Project-default Validator Policy object, public policy publisher, or digest handoff on the happy path.

### Work Item

- Execution is WI-scoped for traceability.
- Start requires a **linked** Work Item identity; create-via-Provider remains available but is not the only path if an issue already exists.
- Optional local/stub WI identity for experimentation without Provider create remains an **open implementation choice** (see Open decisions) — not a reopen of the coordination baseline.

### Runtime selection

- Axiom verifies **own** binding/permission: configured Runtime executable identity present for the selected binding, Profile/model/effort selected (user or default), Project permission allowlist match when configured, Execution authority for the intended effect class.
- Axiom does **not** admit or refuse on capability certification, behavioral probes, or predicted vendor auth/compat (ADR-0032).
- On external failure during authorized spawn/run: record observed diagnostic, last confirmed state, known/unknown effects, next safe action and retry restrictions; do not preemptively invent incompatibility.

### Workflow / Stage / Agent execution

- **Ready default (Intake-settled):** a **minimal Workflow** (Proposed shape: Intake → one work stage → Completion, or a single work stage with optional human accept). Full SDD (≤10 stages) remains a **selectable** template, not the shipped happy-path default.
- Stage binding declares agent role, responsibilities, effects ceiling, inputs/outputs.
- **Operator-authored Plan:** NOT required for dispatch. Axiom **derives** the Stage Plan from the bound stage (**INTERNAL**). Explicit operator Plan remains **OPTIONAL/ADVANCED**.
- **Multi-agent:** dependency-safe plan/graph MAY remain **INTERNAL** (derived from Workflow stage binding). Per-stage **manual** graph preview/digest is **NOT** required on the happy path before apply.
- **Eligible stages auto-advance** (Intake-settled) when Axiom-owned validators/outputs are satisfied and no humanGate fires; human interaction only for material gates, missing Axiom-owned input, or explicit external/sensitive permission.
- **Custom/advanced validation:** OPTIONAL. MUST NOT silently execute arbitrary operator or agent commands; reviewed command policies remain an advanced path with explicit authority — never a silent happy-path requirement.

### Message channel

- Attempt inbox/outbox under supervisor; records grant no authority and cannot satisfy human gates.
- Cross-Runtime exchange uses the same channel; publish outside child attempt context remains denied.

### State and logs

- Durable and user-observable: Execution identity, revision, attempt id/status, logs, artifact refs, errors, next action.
- Graph vs single-stage ledgers remain internal (ADR-0026/0027); happy path does not require the operator to choose format.

### Errors and completion (normative)

- Failure yields a **concrete diagnostic**, last confirmed state, known vs unknown effects, next safe action, and retry restrictions.
- Completion and failure are defined from **Axiom-owned observations** (attempt status, logs, artifacts, Provider/CI facts when actually observed) — **never** from hallucinated “AI succeeded” claims, fabricated CI, merge or Provider state.
- Categories stay actionable; eliminate happy-path `validator_policy_unavailable` caused solely by missing public publisher/digest ceremony.

### Safe retry

- Preserve receipt/lease/epoch/reconcile (ADR-0028). Unknown/partial external effects block automatic redispatch.
- Explicit disposition for workspace reset.

### Delivery

- Technical stage completion ≠ human acceptance.
- Packet/publication/decision/rework remain the boundary for push/PR; optional until operator requests delivery under authority.
- Delivery never silently closes WI or merges.

## Invariants (preserve)

Secrets out of portable manifests/logs; no unapproved destructive/external/costly effects; no writes outside authorized target; no fabricated evidence; sufficient attempt identity for recovery; no blind repeat of uncertain external mutation; communication identity integrity; supervisor cancel/recovery.

## Acceptance criteria

### A. Specify deliverable checklist (this PR / stage)

Checklist for the Specify *document* delivery — **not** product MVP acceptance:

1. Investigation matrix A–L published with file:line evidence at a named SHA.
2. Spec distinguishes REMOVE vs INTERNALIZE vs OPTIONAL vs PRESERVE with counterarguments.
3. Vertical slices S1–S3 defined with normative product AC below; happy path excludes manual Plan / Validator Policy digest / ceremonial transition.
4. Intake-settled baseline vs truly open implementation choices explicitly separated (no reopen of settled product direction as a mandatory new human gate).
5. Live Codex/Claude E2E listed **NOT_RUN** until separately authorized; R2/R3 live tests remain **NOT_RUN** until real authorized execution.

### B. Product acceptance criteria (normative; govern later Implement / qualification)

Observable user scenarios. Passing synthetic docs/CI alone does **not** satisfy these. Live R2/R3 remain **NOT_RUN** until authorized.

**Shared fixture intent:** Project with **two** Repositories + Work Item source + locally available Codex and Claude.

| ID | Scenario (user-observable) | Pass when | Fail when |
|----|----------------------------|-----------|-----------|
| **P-AC1** | From a linked Work Item, start the **default** Workflow in **one natural user action** | Dispatch path starts without operator-authored Plan, manually published Validator Policy/digest, Project-default policy object ceremony, or ceremonial stage transition | Operator must author Plan/Policy/graph digest or manually advance eligible stages on the happy path |
| **P-AC2** | Single agent dispatches and reports observed state | Durable attempt status/logs/artifacts reflect Axiom-observed outcomes | Success claimed without attempt evidence; fabricated Provider/CI |
| **P-AC3** | Two Runtimes exchange a structured question/answer; dependent work unblocks; eligible stages advance **without** “continue” | Inbox records under supervisor; auto-advance of eligible stages; no ceremonial human continue | Cross-Runtime blocked by missing manual Plan/preview; eligible stages stuck awaiting ceremonial continue |
| **P-AC4** | Changes/results/artifacts traceable per Repository | Per-repo artifact/outcome refs inspectable | Orphan or unscoped outcomes without repo linkage |
| **P-AC5** | Failure path | Concrete diagnostic, last confirmed state, known/unknown effects, next safe action, retry restrictions | Opaque failure; speculative pre-refusal without execution attempt when only vendor auth/compat is at issue |
| **P-AC6** | Review/rework loop and optional external PR | Only under explicit authority; no silent merge/close | Push/PR/merge/close without authorization |

**Slice mapping (clear tests):**

| Slice | Covers | Primary checks |
|-------|--------|----------------|
| **S1** | Single-agent first dispatch | P-AC1, P-AC2, P-AC5 (single Runtime) |
| **S2** | Dual-Runtime collaboration + auto-advance | P-AC1–P-AC3, P-AC5 |
| **S3** | Multi-repo outcomes + optional authorized PR | P-AC1–P-AC6 |

Live dual-Runtime / multi-repo qualification (**R2/R3**) remains **NOT_RUN** until real authorized execution produces Evidence.

## Constitution check

Aligns with intent-before-implementation, honest evidence, least privilege, and smallest coherent change. Conflicts with treating full SDD ritual as mandatory for every task — resolved by making SDD optional configuration (Constitution II allows lightweight phases).

## Baseline settled by Intake (do not reopen as mandatory human gates)

The following are **accepted product direction** for AXM-42 (Intake). They are **not** open options for re-debate in Specify:

1. Ship / select a **minimal default Workflow** (full SDD remains optional template).
2. **Automatic progression** of eligible stages (no ceremonial “continue” / manual advance when eligible).
3. Happy path does **not** require operator-authored Plan, manually published Validator Policy/digest, or public policy authoring ceremony.
4. **No external Runtime capability certification**; diagnose vendor auth/compat on observed failure (ADR-0032 aligned).
5. Preserve Axiom-owned integrity, secrets, authorized bounds, supervisor/cancel/recovery, communication identity, and explicit authorization for sensitive/remote/destructive/costly effects.

Reversible technical encoding of these directions is delegated to the Orchestrator during Plan/Implement within this baseline.

## Open MATERIAL / implementation choices (truly unresolved)

Only items that are **not** settled by Intake and may need human input if they expand product/architecture beyond the baseline:

1. **Persisted Execution / schema migration and own-effect boundaries** when introducing the slim happy path alongside historical graph/single-stage ledgers (ADR-0026/0027) — how far new starts diverge vs inspect/cancel/reconcile of old Executions.
2. **Optional local/stub Work Item identity** without Provider create for experimentation — only if it changes Provider/traceability product contracts beyond “link existing WI”.
3. **Concrete supersession text** for accepted Spec 007 / related ADRs where operator-facing Plan + validator-policy gates are currently normative — human must accept an amendment/supersession **before** historical artifacts are rewritten (see impact map). Do not silently accept/supersede.

Technical details (CLI flag coalescing, internal plan derivation mechanics, empty-vs-absent validator representation, message schema tweaks) are **not** HUMAN_DECISION_REQUIRED when reversible and within baseline.

## Spec / ADR impact map (Proposed — do not rewrite historical ADRs here)

| Artifact | Material tension with Intake baseline | Proposed handling (requires human accept before historical rewrite) |
|----------|----------------------------------------|---------------------------------------------------------------------|
| [Spec 007](../007-configurable-workflows/spec.md) | Operator-facing `workflow stage plan`, PlanRef/StagePlan, and reviewable validator-policy artifacts as part of the configurable journey | **Proposed supersession (operator ceremony only):** happy-path Plan/validator digest become INTERNAL/OPTIONAL; graph safety, authority ceilings, humanGates, delivery auth unchanged. Amend 007 after explicit acceptance — 015 does not rewrite 007. |
| ADR-0032 | Aligned (no certification) | Keep |
| ADR-0009 / 0026 / 0027 / 0028 | Internal graph/single persistence, parent-child, retry | **Preserve internals**; UX façade hides dual ledger. Migration/effect boundary = open item (1) above — not silent ledger rewrite |
| ADR-0020 | Workflow definition revision binding before Execution admission | Keep binding integrity; slim path still pins WorkflowRef/snapshot |
| ADR-0003 / 0005 | Control plane / related | Keep |
| ADR-0025 fragments limited by 0032 | — | No further silent supersession |

**What human approval would decide** (when Spec 015 itself is accepted): whether to authorize Plan/Implement of the slim path and a precise Spec 007 / ADR amendment package — **not** whether minimal default / auto-advance / no Plan-digest ceremony / no certification are desirable (already Intake-settled).
