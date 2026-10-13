# #276 / I276-T01 — Successor Graph field contract appendix

This appendix records the concrete T01 encoding implemented under the explicit
maintainer task instruction. It does not change the normative text or status of
[ADR-0026](../../decisions/0026-successor-stage-dispatch-graph-format.md), the
[successor amendment](amendment-276-successor-graph-proposed.md), or historical
Graph1/2. It supplies the field contract and reviewable vectors requested by
[Task T01](plan-tasks-276-stage-agent-execution.md#i276-t01--specify-276-successor-graph-codec-field-contract-and-version-dispatch).
Contract acceptance and operational acceptance remain separate from local test
results. No real inference or dispatch is established here.

## Encoding and version boundary

`formatVersion` is the integer **3**. `SuccessorGraph`, `SuccessorChild` and
`SuccessorAttempt` are separate types in `internal/executiongraph/successor.go`.
`DecodeGraphDocument` dispatches inspection of versions 1/2 to the unmodified
historical reader, and version 3 to `DecodeSuccessorGraph`. Its writer accepts
exactly one historical or successor value. Unknown, missing, duplicate or
malformed discriminators fail; there is no conversion, migration, downgrade or
fallback. The historical reader/writer still refuses version 3.

Canonical writing uses Go JSON struct-field order, compact UTF-8 JSON and one
trailing LF. Times use RFC3339Nano, with zero UTC offset required. Storage
revisions are SHA-256 of **exact input bytes**, including whitespace/LF; the
nonserialized `StorageRevision` is never encoded. Rewriting accepted
noncanonical whitespace produces canonical output, not a claim of identical
input digest. Readers are bounded to 1 MiB and use the existing bounded strict
JSON scan (depth 32, 100,000 nodes, UTF-8/JCS validity) plus exact member names
and required member/type checks. Unknown/duplicate fields at any depth,
case aliases, scalar nulls and missing nonoptional members fail. Optional
members must be absent rather than explicit null or an empty omitted value.
Empty list fields may be `[]` or `null`; each representation has its own bytes.

Every new digest is exactly 64 lowercase hexadecimal characters. Every opaque
ID is the existing 36-character lowercase hexadecimal UUID-shaped encoding.
Bounded identities are nonempty strings of at most 512 bytes, without NUL,
CR/LF or surrounding whitespace. Integers have their Go signed/unsigned bounds;
negative unsigned values and fractional integers fail decoding.

## Graph and child fields

| Member | Type | Requiredness / domain |
|---|---|---|
| `formatVersion` | integer | Required, exactly 3 |
| `parent` | existing `ParentExecution` object | Required; original parent identity, proposal/authority digests, positive graph revision, authority effects and integration child; `createdAt` must be nonzero UTC |
| `children` | array of successor children | Required, 1–32; original Graph2 topology, same Project, authority ceilings, exactly one validation/integration owner and acyclic dependencies |
| `classificationPolicy` | string | Required, exactly `stage-attempt-v1`; frozen at publication |
| `dispatchAuthorityRef` | digest | Required in every state; references the protected T07 authority record; a digest alone grants no authority |
| `stageConcurrency` | uint32 | Required, 1–64; frozen publication ceiling, not proof that parallel dispatch is safe |
| `cancelRequest` | cancellation object | Optional before cancellation; once present, cannot be removed or changed |

Child `executionId`, `parentId`, `graphRevision`, `nodeKey`, `envelope` and
`attempts` remain required members. The envelope consumes the delivered Graph2
resolved Profile tuple and controls without changing their encoding. Child
identity, topology, envelope and order are frozen across updates. An empty
`attempts` list represents an undispatched child. Maximum attempts comes from
the immutable envelope. No consumer automatically copies historical Graphs into
this family; the internal envelope-validation projection carries no Attempts
and does not expose a conversion API.

## Attempt fields and state

| Member | Type | Requiredness / domain |
|---|---|---|
| `attemptId`, `number`, `status` | opaque ID, uint32, status string | Required; globally unique identity, consecutive number starting at 1; supported persisted statuses are `running`, `blocked`, `succeeded`, `failed`, `cancelled`, `unknown` |
| `startedAt` | UTC time | Required for every recorded attempt, no earlier than graph creation or the previous attempt finish |
| `finishedAt` | UTC time | Absent for running; required terminal, no earlier than start |
| `resultReference`, `outputDigest` | digest | Optional process-output references, distinct from coordination qualification |
| `exitCode` | integer | Required, including **zero**; preserved rather than omitted in Graph3 |
| `ambiguousEffect` | boolean | Required; true only for unknown status; never evidence of success/confirmed cancellation |
| `cancellationSeen` | boolean | Optional, true when cancellation observed; required true for `cancel_confirmed`; any true value requires the graph cancel request |
| `classificationReason` | closed string enum below | Absent while running; required terminal |
| `qualifyingRecordRef` | digest | Required exactly for `result_recorded`, `blocker_published`, `result_not_successful`; absent otherwise |
| `leaseEpoch` | uint64 | Required positive; nondecreasing across attempts of a child; frozen for an existing attempt |
| `process` | process-correlation object | Required running; optional terminal (for example a launch failure); absence is never proof of process stop |
| `packetDigest` | digest | Required in every recorded attempt; frozen invocation-packet identity |
| `reconciliationFacts` | array | Required, at most 64; append-only; running facts admitted only through T08 stale reconciliation under an acquired run lease |

`not-started` and `skipped` remain historical domain vocabulary, but are not
classified process outcomes in this family: absence of a recorded attempt
represents work not dispatched. They cannot bypass terminal reason validation.
A running attempt is always the last attempt of its child. IDs cannot collide
with parent, children, other attempts or reconciliation facts.

| Reason | Status | Qualification / extra requirement |
|---|---|---|
| `cancel_confirmed` | cancelled | Cancellation seen, persisted cancel request no later than terminal time, no ambiguous effect |
| `process_outcome_unknown` | unknown | No qualifying record can prove a known process outcome |
| `timeout_stopped` | failed | No qualifying record required |
| `blocker_published` | blocked | Same-attempt canonical blocker digest |
| `process_failed` | failed | Nonzero exit code |
| `result_missing` | failed | Zero exit code, no qualifying record |
| `result_not_successful` | failed | Zero exit code, same-attempt failed result digest |
| `result_recorded` | succeeded | Zero exit code, same-attempt successful result digest |
| `integration_applied` | succeeded | Integration owner only, zero exit code, no coordination qualification |
| `integration_failed` | failed | Integration owner only, no coordination qualification |
| `integration_unknown` | unknown | Integration owner only, no coordination qualification |

Integration owners use only the three integration reasons; other children use
only the eight process/coordination reasons. No new classifier rule is added.
`ValidateSuccessorRecords` independently resolves qualifying digests against
accepted canonical record projections, checking parent, child, graph revision,
attempt, record kind, nonzero canonical revision and result outcome. Merely
passing the codec is not proof that references exist or that authority is valid.

## Correlation, cancellation and reconciliation objects

| Object / member | Type | Requiredness / domain |
|---|---|---|
| process `pid` | int64 | Required positive |
| process `startIdentity`, `groupIdentity` | bounded identity | Required OS start identity and process-group/job identity; observation/fencing is T06/T43, not inferred by the codec |
| cancel `requestId` | opaque ID | Required |
| cancel `parentId`, `graphRevision` | opaque ID, uint64 | Required, exact graph lineage |
| cancel `actorKind`, `actor` | string, bounded identity | Required; kind exactly `human-operator` |
| cancel `requestedAt` | UTC time | Required, no earlier than graph creation; does not assert process stop |
| cancel `authorityRef` | digest | Required, separately scoped operation authority |
| fact `factId` | opaque ID | Required, globally unique |
| fact `parentId`, `childId`, `graphRevision`, `attemptId` | lineage IDs / uint64 | Required, exact affected graph/child/attempt |
| fact `observationsDigest` | digest | Required; bounded observations identity, not raw logs |
| fact `classification` | closed enum | Required: `retry_safe` or `recovery_required`; neither is success |
| fact `authorityRef` | digest | Required, separate reconcile authority |
| fact `actorKind`, `actor` | string, bounded identity | Required; kind exactly `human-operator` |
| fact `recordedAt` | UTC time | Required, no earlier than terminal time or previous fact |

`ValidateSuccessorUpdate` validates both states without serialization. It freezes
publication fields and previous attempts, preserves exact prior fact prefixes,
and permits only a running attempt's terminal save and new attempts/facts. T08 also permits facts on stale running attempts without rewriting their status; a running attempt remains ineligible for dispatch. The local fact operation acquires the run lease first, refuses a live holder, and rechecks observations before CAS.
Process identity, epoch, packet identity and start time cannot change during a
terminal save. A terminal status/reason/reference cannot be rewritten. New
contradictory facts are retained explicitly; no last-writer-wins reconciliation
or success promotion occurs. CAS, canonical stream resolution, run leases,
protected authority and transaction recovery remain the responsibilities of
T04/T06/T07/T08 and their consumers, not claims made by this codec.

## Invariants and executable vectors

| Invariant | T01 boundary / consumer boundary |
|---|---|
| I-1 | Domain validator enforces terminal status/reason matrix |
| I-2 | Codec requires successful-result digest; record validator checks exact lineage/kind/outcome; consumer resolves canonical storage |
| I-3 | Codec requires blocker digest; record validator checks canonical blocker and lineage |
| I-4 | Persisted fields/facts are exported; validators never call the classifier; scheduler/integration/status consumers are later Tasks |
| I-5 | Update validator freezes terminal state and fact prefixes; final-drain/CAS timing is later consumer work |
| I-6 | Explicit version dispatch keeps historical inspection separate; historical bytes cannot enter successor decoder; public mutation admission is T09 |

`internal/executiongraph/testdata/successor/valid/` contains canonical publication,
running, every terminal reason, epoch progression, cancel-request and
reconciliation vectors. `invalid/` contains missing reason/qualification/exit
code, unknown reason/version/field, duplicate field, wrong case, missing policy,
decreasing positive epoch, zero concurrency and invalid running correlation.
Update tests additionally reject mutated/deleted facts, changed terminal state,
changed concurrency and rewritten epochs. Record tests reject missing/foreign
qualification. `historical-digests.json` freezes SHA-256 expectations for the
existing Graph1 stable corpus and delivered Graph2 fixture, without changing
those files. Tests require exact historical byte round-trips and old-reader
refusal of every valid successor vector.
