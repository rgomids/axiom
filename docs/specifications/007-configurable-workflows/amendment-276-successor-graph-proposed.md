# Specification 007 amendment — #276 successor Graph format (proposed)

## Status

**Proposed — Awaiting Human Approval, 2026-10-12.**
The maintainer approved PD-R276-01/B *direction* ([receipt](https://github.com/rgomids/axiom/pull/311#issuecomment-6125676685)).
This exact amendment and [ADR-0026](../../decisions/0026-successor-stage-dispatch-graph-format.md)
are not yet accepted, do not supersede the 2026-10-11 Phase 1 accepted
amendment, and do not authorize Implementation or real inference.

**Accepted, recorded 2026-10-13**, by the maintainer, for this exact artifact
(blob `53ac761f31e79dbff1c6ed9652de92f8556e7a6f` at `main` `bf3c621`) together
with ADR-0026; see the [#276 acceptance record](plan-tasks-276-stage-agent-execution.md#maintainer-acceptance-and-delivery-governance--recorded-2026-10-13).
Implementation is authorized within the accepted contracts; real inference
keeps its separate gate. The paragraph above is preserved as history.

## Scope and ownership

Complement only the successor-Graph version/codec contract for the public
stage dispatch capability of Issue #276. All Graph1/2 formats, exact-byte
digests, Profile effort/resolver/Execution3 delivered by #306 and its completed
I306-X05 are preserved without adding work to #306. #276 T01 owns successor
codec/version dispatch/validators; T04/T06/T07/T08/T09/T37 own consumers
and compatibility. No Workflows/Executions migration is implied.

## A1 — Versioned graph persistence

A new explicit successor format discriminator (candidate v3; exact number and
canonical wire representation must be confirmed in T01 reviewed test vectors)
distinguishes #276 graph publications from delivered Graph2. Successor has
closed bounded encoding and version-dispatched validation. Unknown/duplicate
fields, invalid versions or missing required state fail closed. The historical
Graph1/2 encodings, SHA-256 digests, identity, historical fixtures and reader
semantics cannot be rewritten, rebased, downgraded, rebound or reinterpreted.

Graph1/2 may be inspected under original reader contracts, but cannot be
mutated by #276 public commands. Unknown successor data does not fall back
to Graph2. Existing Execution3, StagePlan2, Workflow2 and local Profile-v2
are consumed without changing their format identifiers.

## A2 — Persisted fields and invariants

The successor Graph carries:
- `classificationPolicy`: frozen classifier revision.
- `dispatchAuthorityRef`: digest-bound protected dispatch-authority record.
- `stageConcurrency`: positive bounded publication-time concurrency.
- `cancelRequest`: identity, graph revision, actor, UTC time and operation
  authority, without implying a confirmed process stop.

A successor Attempt carries:
- `classificationReason`: closed reason set per accepted §3.5.
- `qualifyingRecordRef`: canonical coordination-record digest distinct
  from existing process-output `ResultReference`.
- `leaseEpoch`: monotonic positive epoch with fencing checks.
- Process correlation: PID plus process-start and group/job identity, to
  reject PID reuse and support bounded stop/reconciliation.
- `packetDigest`: bounded invocation task-packet identity.
- Append-only reconciliation facts beside that attempt, binding exact
  parent/revision/attempt, observations digest, authority, actor/time and
  classification; contradictory facts are surfaced, not overwritten.

T01 must freeze JSON naming, types, requiredness per status, size limits,
canonical encodings and invalid/valid vectors as separately reviewed contract
details. No implementation or format guessing is authorized by this proposal.

The existing I-1–I-6 semantic rules of the accepted Phase 1 remain:
every terminal status has a valid reason; a succeeded/result_recorded attempt
requires a qualifying successful record of the same graph/parent/revision/
attempt; blocked requires the corresponding blocker; scheduler, integration,
retry and status consume persisted classification; late records cannot
reclassify terminal attempts. The final inbox drain, classification, reason
and qualifying reference are committed atomically with exact-byte CAS.
Reconciliation facts never rewrite persisted status or promote success.

## A3 — Authority and transaction boundary

T07 owns a protected dispatch-authority record bound to the exact preview,
graph/ledger revision, first-attempt scope, actor/time and authorized effects.
It is committed with the successor graph, or with the single-agent ledger
under the preexisting ADR-0007 transaction/recovery boundary. Wrong revision,
torn/partial commit, stale preview, reused authority or absent record fails
closed. Cancellation, retry, reconcile, integrate and probe each require
their own separately scoped authority. Historical readers are not authority
to mutate historical graphs.

## A4 — Admission and backward compatibility

| Input | #276 mutation | Inspection / result |
|---|---|---|
| Graph1 | Deny | Historical read-only when supported |
| Delivered Graph2 without #276 successor fields | Deny | Historical read-only when supported |
| Approved successor with frozen Execution3 and Profile tuple, capability/auth/authority gates passing | Allow only within explicit dispatch authority | Exact bound operation |
| Unknown, missing or malformed successor version/required fields | Deny | Unsupported format; no fallback |

#306 amendment/ADR-0024 schema-2 Workflow, format-3 Execution, local-v2
Profile-owned effort, PE-05a and anti-fallback remain mandatory.
Historical `default-sdd` r1 bytes/digest and all six #306 corpus families
stay unchanged.

## A5 — Verification and Evidence

I276-T37 covers baseline §3.10's 15 cases plus the successor delta:
- Read historical formats without changing bytes/digests.
- Reject mutation of Graph1/Graph2 and old-reader admission of successor.
- Strict unknown/duplicate/version/requiredness rejections.
- Revalidate exact-bound authority after restart.
- Fail closed on missing/foreign result, late record, stale epoch, PID reuse,
  contradictory reconciliation or uncertain cancellation.
- Preserve atomic save and recoverability across torn multi-object commits.
- Enforce stage-concurrency and immutable frozen tuple requirements.

Tests are **planned**; no real Runtime, probe, smoke or inference is claimed.
T40 has an independent real-smoke authorization/acceptance gate.

## Compatibility and authority

This document becomes a material amendment only after exact-text human
acceptance and corresponding ADR-0018 partial-supersession annotations.
Until then, the accepted 2026-10-11 amendment/ADR-0025 and Plan Part I
retain their historical contract and Phase 2 remains Proposed.
