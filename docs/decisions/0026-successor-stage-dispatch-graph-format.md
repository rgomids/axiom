# ADR-0026 — Successor execution Graph for public stage dispatch

## Status

**Accepted, recorded 2026-10-13**, by the maintainer, for this exact artifact
(blob `1a99a571ee74656a23fa294dfae7e3bc63bc2469` at `main` `bf3c621`); see the
[#276 acceptance record](../specifications/007-configurable-workflows/plan-tasks-276-stage-agent-execution.md#maintainer-acceptance-and-delivery-governance--recorded-2026-10-13).
The paragraph below is preserved as history. After acceptance, only ADR-0018
supersession bookkeeping changed: this Status order, the supersession heading
and note, and the `## Supersedes` backlink; the decision text is unchanged.

Proposed on 2026-10-12 for acceptance of this exact artifact. The maintainer
approved the architectural **direction** PD-R276-01/B
([receipt](https://github.com/rgomids/axiom/pull/311#issuecomment-6125676685)); that approval is not approval of this detailed ADR,
its wire representation, the amended Specification, Tasks, or implementation.
No supersession is currently in force.

## Context

#306 / PR #310 delivered a closed Graph format 2 that records resolved Profile
bindings and strict JSON with exact-byte SHA-256 storage revisions. The accepted
#276 Phase 1 PD-16 and ADR-0025 anticipated one joint Graph2 revision carrying
#276 fields, yet completed I306-X05 does not serialize classification reason,
qualifying coordination digest, epochs, process correlation, authority/cancel
markers, concurrency and reconciliation facts. Previously approved historical
Graph1/2 readers reject unknown members; extending their persisted meaning
silently would break versioned contracts or preserve misleading digests.

[PD-R276-01 Decision Packet](../specifications/007-configurable-workflows/reconciliation-276-post-merge-306.md)
records alternatives and Evidence. Issue #276 retains public dispatch and
Runtime capability proof; #306's 23 completed Tasks are not reopened.

## Decision (proposed detailed contract)

1. **Introduce an explicit successor Graph format family** for new authorized
   #276 publications (candidate format version 3, exact version/encoding to be
   fixed by the reviewed T01 vectors). Graph1/2 bytes, identifiers, historic
   digests, readers, fixtures and frozen corpus remain unchanged and readable
   under their original semantics. No automatic conversion, rebind or migration.
2. **Ownership:** #276/T01 owns successor structs, strict codec, format/version
   dispatch, validation and canonical vectors. #306/I306-X05 owns delivered
   Graph2 and PE-05a only, without supplementary work. #276/T04/T06/T07/T08/
   T09/T37 own the respective consumers, persisted epochs, protected authority
   record, append-only facts, admission and compatibility integration.
3. **State and classification:** successor Graph records the policy revision,
   dispatch-authority reference and bounded stage concurrency. Each Attempt
   stores a closed classification reason, qualifying canonical record digest,
   monotonic lease epoch, PID/process-start/group identity and packet digest;
   persistent cancellation requests bind actor/authority, and reconciliation
   facts are append-only next to the affected attempt. Precise JSON names,
   sizes and state-specific requiredness remain part of T01's proposed contract.
4. **Invariants:** preserve Phase 1 I-1–I-6: terminal reasons and qualifying
   records must correspond to the same parent/revision/attempt; one atomic CAS
   terminal save after final drain; consumers never reclassify on rendering;
   late records do not rewrite terminal statuses; reconciliation does not
   promote success. Authority, lease fencing, process identity and unknown
   outcomes fail closed. Separate dispatch-authority records bind exact preview,
   graph/ledger revision, first-attempt scope, effects and actor, with protected
   commit/recovery per ADR-0007.
5. **Admission:** #276 mutating commands reject Graph1 and delivered Graph2,
   regardless of whether historical readers can inspect them. Only the
   separately accepted successor family can be used by new #276 mutation
   paths, after #306 Execution3/Workflow2/Profile-v2 and effort checks,
   capability proof, authentication, preview-bound authorities and lease
   gates. Unknown/disallowed versions never fall back.
6. **Compatibility:** T37 verifies old/new round-trips and exact digests,
   unsupported-reader refusals, strict unknown/duplicate fields, recovery
   crash windows, torn multi-object commits, restart, wrong-attempt results,
   stale epoch/PID reuse, cancellation uncertainty and concurrency bounds.
   Historical Graphs remain immutable; no rewrite of the six-family #306
   compatibility corpus or of `default-sdd` r1.

This ADR specifies no permission for implementation, real Runtime inference
or new external effects.

## Partial supersession — effective since acceptance, 2026-10-13

The conditional wording below is preserved as proposed; acceptance recorded
on 2026-10-13 made this partial supersession effective, as listed in
[Supersedes](#supersedes).

**When and only when accepted**, this ADR would supersede the bounded
**joint-Graph2/X05 version/ownership fragment** of
[ADR-0025 Consequences](0025-public-stage-dispatch-and-capability-proof.md#consequences),
the corresponding PD-16/§3.10 assertion and amendment #276 A6 sentence.
The rest of ADR-0025 (supervision, authorization, coordination, classifier,
proof and safety) remains in force. Upon exact-artifact acceptance, create
the ADR-0018 `## Supersedes` backlink and historical annotation in ADR-0025,
and update the ADR index's partial-supersession note. Until then, none of
those accepted artifacts is silently replaced.

## Supersedes

- [ADR-0025 Consequences](0025-public-stage-dispatch-and-capability-proof.md#consequences):
  only the joint-Graph2/I306-X05 version-and-ownership fragment (format
  changes "versioned together with the #306 graph v2"). Successor Graph
  ownership belongs to #276 under this decision. Supervision, authority,
  coordination, classification, capability proof and security/safety
  decisions of ADR-0025 remain in force.
- [Plan #276 Part I](../specifications/007-configurable-workflows/plan-tasks-276-stage-agent-execution.md):
  only the corresponding PD-16/§3.10 joint-Graph2 assertion, and the
  corresponding [amendment #276](../specifications/007-configurable-workflows/amendment-276-stage-dispatch.md)
  A6 sentence; their wording is preserved as history.

Acceptance: [#276 acceptance record](../specifications/007-configurable-workflows/plan-tasks-276-stage-agent-execution.md#maintainer-acceptance-and-delivery-governance--recorded-2026-10-13).

## Alternatives considered

- A — same-number Graph2 extension: rejected as a preferred path because
  delivered readers enforce closed JSON and extension changes wire/digest
  semantics without a distinct format discriminator.
- B — explicit successor family: recommended/approved as direction, with
  additional codec, migration policy and consumer work isolated to #276.
- C — bound sidecar/operational ledger: viable but requires broader multi-object
  recovery and changes the accepted graph-resident classification boundary.
- D — late reassignment to completed I306-X05: ownership repair alone does
  not resolve strict-reader compatibility, and reopens accepted completed work.

## Consequences and gates

New successor-format maintenance and compatibility testing are #276 costs;
Graph1/2 remain byte-compatible. The separately proposed
[Spec amendment](../specifications/007-configurable-workflows/amendment-276-successor-graph-proposed.md)
must be accepted together with this exact ADR before Tasks are approved.
The 44 Tasks remain Proposed; Phase 3 Implementation and T40 real smoke
require later independent authorization.

## Revisit when

A formally accepted compatible extension mechanism, major Graph format
migration requirement or separately authorized operational-state architecture
invalidates the decision's assumptions.
