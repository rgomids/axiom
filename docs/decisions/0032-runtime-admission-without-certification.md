# ADR-0032 — Runtime admission without capability certification

## Status

Accepted on 2026-10-19 by [explicit maintainer acceptance](https://github.com/rgomids/axiom/pull/334#issuecomment-6278024276)
of this ADR and the AXM-36 amendment at commit `b4baf106f02b89028149ff57b01bf960524692f8`.
The acceptance includes exactly the partial-supersession map below. This
annotation records that decision; it grants no Implementation, dispatch,
inference, merge, release or Execution acceptance. The separately authorized
[Plan](../specifications/007-configurable-workflows/plan-axm36-runtime-admission.md)
remains Proposed. The original proposal wording below is preserved as history;
its pending-acceptance statements describe the pre-acceptance revision.

Historical status:

Proposed, 2026-10-19. Explicit human acceptance of this exact proposal and the
[AXM-36 amendment](../specifications/007-configurable-workflows/amendment-axm36-runtime-admission.md)
is pending before Plan/Implementation. The product direction is decided;
architectural reconciliation is not accepted. No dispatch, inference, merge,
release or Execution acceptance is authorized here.

## Context

ADR-0025 decision 5 requires authorized behavioral proof bound to executable,
version, mode, model/effort and freshness. Specification 007 PM-5/PM-5a obtains
the version from an embedded exact-digest registry. The inspected registry is
empty; a normal installation can fail admission independently of its actual
ability to perform the requested operation. Profile effort resolution also
requires proof and an exact-identity declaration, so removing only the initial
registry lookup would leave equivalent certification gates downstream.

The maintainer's AXM-36 product decision of 2026-10-19 removes certification
as an Axiom responsibility for Codex and Claude. The reported login observation
and capability_unknown do not establish Runtime incompatibility or inference
success. The proposal separates operational identity, authorization and actual
outcome from claims about vendor capability.

## Decision

Proposed: admit the user-selected Runtime through installed executable,
configured Profile/model/effort, observable authentication, Project Runtime
Policy, Workflow/Agent binding and specific Execution authority; observe and
classify actual operational failures without a version registry, executable
allowlist, CapabilityProven or prior behavioral probe.

RA-1–RA-5 of the linked amendment define the behavior and verification
obligations. Exact executable and configuration identity remain preview/apply
integrity controls, never compatibility certification. Upgrades do not require
qualification. Preserve effort intent and safe effective-source inspection
without exact-version declarations as eligibility gates. No implicit fallback,
permission expansion or inferred sandbox guarantee is introduced.

Retain supervisor, durable pre-spawn authority, inbox ingress, terminal-save
classification, immutable history, retry/reconciliation and canonical-state
protection. Unknown/partial effects prevent automatic redispatch. Login and
exit 0 alone never satisfy a Stage contract. CLI and thin skills expose the
same sanitized Evidence, certainty and actionable next step.

## Supersedes

Effective on the acceptance above, limited to the fragments listed below.
All unrelated decisions remain in force. The following proposal introduction
is historical; the map itself was explicitly accepted at the recorded SHA.

Historical introduction:

This section records **intended**, not effective supersession. In accordance
with ADR-0018, no `Supersedes` section or annotation on an accepted ADR is added
until human acceptance. On acceptance, convert this map to that section and
annotate only the affected historical fragments with the actual acceptance
date; preserve their text and all unrelated decisions.

- [ADR-0025 decision 5](0025-public-stage-dispatch-and-capability-proof.md#decision):
  replace mandatory probe-derived capability admission, exact binding/freshness
  requalification and associated mandatory probe lifecycle. Keep separation of
  authority/confinement, honest reporting, redaction and no unauthorized probe
  effects. Its probe-specific Consequences/store expectations become historical;
  decisions 1–4 and supervisor/coordination/classification remain accepted.
  Decision 6's reference to #306 follows only the proof-related Specification
  reconciliation below; immutable bindings and effort ownership remain.
- [Specification 007 binding amendment](../specifications/007-configurable-workflows/amendment-276-axm22-binding-identity.md):
  replace PM-5/PM-5a registry requirements and unmatched-identity refusal;
  PM-7 proof prerequisites, PM.4 proof composition and PM-13 version-evidence
  eligibility. BI-2/BI-2a and PM-2 exact-identity declaration obligations change
  only as necessary to eliminate recertification. Keep epoch, source trust,
  no-override and identity/authority semantics. RA-6 names bounded fragments.
- [Specification 007 effort amendment](../specifications/007-configurable-workflows/amendment-306-profile-effort-onboarding.md):
  PE-03/04/05/05a/06 proof requirements/reference, exact-version evidence and
  per-model effort certification; keep Profile ownership, selection,
  adapter encoding, inspection consent and override guards.
- [Specification 007 dispatch amendment](../specifications/007-configurable-workflows/amendment-276-stage-dispatch.md):
  A2/A6 proof gates and tuple proof reference, including mandatory qualification
  of implicit and model-scoped capabilities. Keep effect requirements,
  preview integrity, authentication and authority. Retired proof-only tasks,
  CLI/skill guidance and stores require prospective reconciliation, not a
  reinterpretation of historical Evidence.

No wholesale supersession of ADR-0024, execution graph formats or supervisor
contracts is proposed. RA-6 records direct code consumers and unresolved
versioning impacts; this ADR neither chooses a new wire format nor accepts
legacy tuples with fabricated proof.

## Alternatives considered

| Alternative | Trade-offs and disposition |
| --- | --- |
| Operational admission and classified failure (recommended) | Less per-installation coupling, no qualification inference cost, portable installed journey; more failures surface during real authorized work and require strong diagnostics/recovery |
| Populate registry and automate probes | Preserves historical gate but costs maintenance/inference, couples installations and upgrades to certification, and contradicts the decided product direction; not a candidate for implementation |
| Remove only registry check or treat unknown capability as proven | Small initial diff but leaves Profile/effort proof gates, lies about Evidence and fails structural simplification; rejected |
| Remove authority and source/identity checks with proof | Lower admission complexity but weakens intent/security and permits stale authorization; rejected |

## Consequences

Less complexity in admission comes with responsibility for concrete adapter
integration tests, installed synthetic journeys and operator-facing diagnosis.
A Runtime can accept an invocation and still fail or ignore a requested
feature; Axiom reports only observed facts and contract validation, without
claiming vendor internals or billing. Unknown effects remain expensive to
recover and cannot be hidden behind automatic retry.

Authorization, identity and configuration safety still couple preview to
dispatch. Removing proof structurally affects Profile tuples, stores,
commands, adapters, workflow compilation and retry. Plan must select bounded
format/retirement changes, preserve historical inspection/Evidence and retain
negative security tests. That coordinated boundary is harder to reverse than
a local check removal; reinstating mandatory certification would need a new
product and architecture decision, never a quiet adapter restriction.

## Revisit when

Actual authorized runs expose an interface or safety boundary the adapter
cannot honor. Review the concrete invocation/control or supported operation,
with observable Evidence. Do not turn a version bump, missing receipt or
unknown vendor feature into recertification. An independent change in product
scope or threat model needs explicit human decision.

## Acceptance and next stage

Human acceptance must identify this ADR and amendment revision/commit, the
partial-supersession boundary and remaining design obligations. Product approval,
PR review, merge or green CI alone cannot infer that acceptance. After it is
recorded, reconcile historical annotations/index prospectively and request
separate authority for Plan. Implementation, real runtime acceptance and
release remain outside specify.
