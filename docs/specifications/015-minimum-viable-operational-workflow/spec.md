# Specification 015 — Minimum viable operational workflow

Status: **Proposed** (AXM-42 Specify). Not approved. Does not authorize Implementation, ADR rewrite, merge, release or MVP acceptance.  
Baseline HEAD for investigation: `440e44a68638e08e901fd3f87f5b15cd395a6365`.  
Product baseline: human-approved AXM-42 Intake (workflow orchestration/coordination; Axiom owns operational behavior only).  
Evidence: [research investigation](../../research/axm-42-minimum-viable-workflow-investigation.md).

## Problem

The configurable Execution path can schedule and supervise Codex/Claude work, but the **operator-facing ceremony** before first dispatch exceeds the minimum needed for Axiom-owned safety: mandatory manual Stage Plan, mandatory Validator Policy digest **without a public publisher**, ten-stage SDD as the ready path, and manual advance between eligible stages. Certification was already removed (ADR-0032); remaining friction is mostly owned selection, authority CAS and missing product defaults.

## Goal

Define the **minimum operational contracts** so that after Project configuration, a Work Item can reach a **genuine single-agent Runtime dispatch** with durable status/logs, then extend to cross-Runtime collaboration, multi-Repository outcomes and optional authorized PR — without certifying vendor capabilities or promising task success.

## Non-goals

- Plugin SDK, general policy engines, distributed control plane.
- Model-support certification, executable allowlists-as-compatibility, behavioral probes as admission.
- Weakening process safety (leases, uncertain-effect fences, secret hygiene, write confinement).
- Replacing live R2/R3 qualification with synthetic CI alone.
- Silent supersession of accepted ADRs (map only; human acceptance required).

## Actors

- Operator (human): configures Project, grants external/human authority, accepts delivery.
- Axiom (Lingo): loads/validates own state, schedules stages, dispatches/supervises, records truthfully.
- Runtime agents (Codex/Claude): perform work under attempt confinement; not certified by Axiom.
- Providers (GitHub/Linear when supported): Work Item identity and optional PR/push effects.

## Slim contracts (Proposed)

### Project configuration defaults

- Portable Project: repositories (N≥1), Work Item source binding, selected Workflow ref, Runtime policy (allowed runtimes + profile allowlist).
- Machine-local: Runtime bindings/Profiles (v4), auth observations, workspace preferences.
- Defaults after first successful configure: selected minimal Workflow; default Profile per Runtime when unambiguous; Project-default validator policy (empty commands or reviewed builtin).

### Work Item

- Execution is WI-scoped for traceability.
- Start requires a **linked** Work Item identity; create-via-Provider remains available but is not the only path if an issue already exists.

### Runtime selection

- Admit on: installed executable identity in preview, configured Profile/model/effort, observed auth, Project allowlist match, Execution authority.
- Do **not** admit on capability certification (ADR-0032).
- On failure: record observed diagnostic + next action; do not preemptively invent incompatibility.

### Workflow / Stage / Agent execution

- **Ready default:** a minimal Workflow (Proposed shape: Intake → one work stage → Completion, or a single work stage with optional human accept). Full SDD (≤10 stages) remains a selectable template, not the only path.
- Stage binding declares agent role, responsibilities, effects ceiling, inputs/outputs.
- **Single-agent:** Axiom derives the Stage Plan from the bound stage (INTERNALIZE). Operator-supplied Plan remains OPTIONAL/ADVANCED.
- **Multi-agent:** explicit plan or generated graph preview remains required before apply.
- Eligible stages **auto-advance** when validators/outputs satisfied and no humanGate fires; human interaction only for material gates, missing input, or external/sensitive permission.

### Message channel

- Attempt inbox/outbox under supervisor; records grant no authority and cannot satisfy human gates.
- Cross-Runtime exchange uses the same channel; publish outside child attempt context remains denied.

### State and logs

- Durable: Execution identity, revision, attempt id/status, logs, artifacts refs, errors, next action.
- Graph vs single-stage ledgers remain internal (ADR-0026/0027); happy path does not require operator to choose format.

### Errors

- Categories stay actionable (`validator_policy_unavailable` eliminated for default-empty policy path; replace with clear Project-default messaging).
- Never fabricate success, CI, merge or Provider state.

### Safe retry

- Preserve receipt/lease/epoch/reconcile (ADR-0028). Unknown/partial external effects block automatic redispatch.
- Explicit disposition for workspace reset.

### Delivery

- Technical stage completion ≠ human acceptance.
- Packet/publication/decision/rework remain the boundary for push/PR; optional until operator requests delivery.
- Delivery never silently closes WI or merges.

## Invariants (preserve)

Secrets out of portable manifests/logs; no unapproved destructive/external/costly effects; no writes outside authorized target; no fabricated evidence; sufficient attempt identity for recovery; no blind repeat of uncertain external mutation.

## Acceptance criteria (Specify-level Proposed)

1. Investigation matrix A–L published with file:line evidence at a named SHA.
2. Spec distinguishes REMOVE vs INTERNALIZE vs OPTIONAL vs PRESERVE with counterarguments.
3. Vertical slices S1–S3 defined with observable AC; S1 = first dispatch without manual Plan/Validator Policy authoring.
4. Explicit MATERIAL human decisions listed before Implement.
5. Live Codex/Claude E2E listed NOT_RUN until separately authorized.

## Constitution check

Aligns with intent-before-implementation, honest evidence, least privilege, and smallest coherent change. Conflicts with treating full SDD ritual as mandatory for every task — resolved by making SDD optional configuration (Constitution II allows lightweight phases).

## Open MATERIAL decisions

1. Validator Policy: Project-default auto vs mandatory public author command.
2. Single-agent Plan: fully derived vs retained explicit file.
3. Default Workflow: minimal vs keep SDD-10/r1 as shipped default.
4. Auto-advance: on for all non-humanGate stages vs opt-in.
5. Provider: may local experimentation use a stub/local WI identity without Provider create?
