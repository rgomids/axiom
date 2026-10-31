# ADR-0027 — Persist single-stage attempts without rewriting Execution3

## Status

Accepted on 2026-10-16 by explicit maintainer decision in
[AXM-8](https://linear.app/rgomids/issue/AXM-8/mvp-run-and-coordinate-stage-agents-through-cli-and-runtime-skills)
(Linear comment `fee15434-632b-4239-8d14-55653cab0ad7`). Option A
(SINGLE-PERSISTENCE-01) is selected; alternative B (Execution4) is not
approved. Acceptance authorizes implementing this persistence contract on
PR #320; no operational acceptance, merge or release is inferred. The accepted 44
Tasks and existing ADRs remain in force; no historical artifact is rewritten.

## Context

Plan #276 §3.11 and T26–T31 require a single-agent attempt in a stage ledger,
with persisted classification, cancellation, epoch, recovery and explicit retry.
There must be no fabricated Graph or integration child. ADR-0025 requires the
authority and ledger attempt to commit before spawn. The successor amendment A1
consumes the existing Execution3 contract without changing its identifier.

The delivered Execution3 is a workflow-gate ledger. `workflow.State` has no
attempt collection; `workflow.StageLedger` contains stage inputs/outputs and
validator results. Its transition replay requires the exact derived revision,
stage, status and update time. It cannot record a child start, process identity,
packet, attempt classifier or cancellation without a different persistence
contract. A start must not fabricate a workflow failure, human fact, technical
completion or gate advance.

The existing `WorkflowStore.CommitSingleDispatch` binds its authority to the
Execution3 wire digest, requires a changed ledger generation, and refuses an
authority newer than that ledger's update time. A valid existing Execution3
cannot be made into a new attempt generation by changing only its timestamp or
revision. The current single interfaces are preparation seams, not a complete
durable attempt store. Extending a closed codec under the same discriminator,
dropping generation fencing, or reconstructing status from child output would
hide this gap.

Executable evidence: `internal/local/single_dispatch_contract_test.go`,
`TestSingleDispatchCannotInventAnExecution3AttemptGeneration`, against the
unchanged #306 fixture. It proves unchanged-generation refusal, authority-time
refusal, invalid timestamp/revision-only changes, closed-codec refusal of an
attempt field, and preservation of stored workflow truth. The implementation
record contains actual commands/results. This is a bounded persistence decision,
not a request to reapprove all 44 Tasks.

## Decision

Option A: add an explicit single-stage operational ledger and a
versioned single-stage authority contract, while leaving Execution3 and all
Graph formats unchanged.

> Superseded by [ADR-0037 — One canonical Execution for new workflow runs](0037-one-canonical-execution-for-new-runs.md#decision), accepted 2026-10-30.
> Option A is superseded for new executions: the previously rejected alternative B direction is selected as a new versioned Execution with nested attempts. Existing bytes, no fake workflow transition or Graph node, no automatic migration/redispatch, cancel/attempt identity and authority before spawn remain in force.

The proposed contract is:

- A new private `single-stage-runs/v1/<project>/<scope-digest>.json` record,
  kind `single_stage_run`, format 1. The scope digest binds the exact Project,
  Repository, Work Item, Execution ID, stage ID and stage ordinal using the
  repository's canonical SHA-256 convention. It never names a Graph parent.
- Immutable binding to the reviewed Execution3 storage generation, Workflow
  revision, compiled Stage Plan/inputs, frozen v3 Profile, policy, capability
  Evidence, pinned binary and exact workspace/inbox. A changed binding requires
  a new reviewed operation; it is never silently re-resolved or migrated.
- A single Runtime execution identity equal to the canonical Execution ID,
  with distinct per-attempt IDs. No integration owner or synthetic Graph node.
  Attempts use the existing shared bounded attempt/status/classifier vocabulary,
  correlation, process identity, packet digest and append-only reconciliation
  facts. Limits remain 1–10 attempts and at most 24 hours per attempt.
- A monotonic run epoch, persisted cancellation request and explicit current
  authority references. Status reads persisted fields only; uncertain stop is
  unknown, reconciliation cannot promote success, and retry needs its own
  authority and explicit workspace disposition after effects.
- A distinct version-2 **single-stage** dispatch-authority codec binds the
  operational ledger's generation as well as the unchanged Execution3 source
  generation and exact reviewed preview. Existing version-1 receipt meanings,
  readers and historical bytes remain unchanged. Graph dispatch keeps its
  existing version-1 contract. Unknown versions fail closed, without fallback.
- Protected ledger/authority publication and exact read-back before any process,
  using ADR-0007 recovery and lock order. The run lease is acquired before
  publication/CAS locks. Every save, drain and reset effect checks the current
  epoch, binding, authority scope and owned workspace. Inert retained inputs
  grant no authority; partial commits remain recoverable, never dispatched.
- The same bounded detached supervisor, adapter, stdin packet, authentication,
  immediate capability/effort checks, ingestion and classifier as Graph3.
  Its single-stage entrypoint accepts only the protected single-stage receipt.
  No public `stage resume` command is introduced; `stage run` resumes only
  already-authorized eligible work.
- Workflow-gate advancement and validation remain separate human/domain
  operations. An attempt result never rewrites the Execution3 binding or
  fabricates a workflow transition. A concurrent workflow-generation change
  cannot authorize a child effect and leaves explicit recovery when relevant.
- The new directory, kind, codec/version admission and actual writer must be
  added to the inventory and compatibility guards. New canonical vectors and
  publication/kill-window tests are required before claiming implementation;
  existing Graph1/2 and #306 corpus data stay immutable. No automatic conversion
  of historical single dispatch receipts is authorized.

Acceptance of this proposal would supply the missing bounded persistence
contract for the existing #276 Tasks. It would authorize no real vendor effect,
Provider mutation, task/state transition, merge, release or T40 run.

## Alternatives considered

| Alternative | Trade-offs |
| --- | --- |
| A — separate versioned single-stage ledger and authority (recommended) | Preserves workflow-gate truth and historical codecs; adds a store, inventory kind, receipt codec and multi-object recovery boundary. Native local storage only, without a new dependency or service. |
| B — explicit Execution4 with attempt fields | Puts workflow gates and runtime attempts in one CAS record; requires a broader Execution compatibility/admission change, new migration policy and evolution of the accepted unchanged-Execution3 constraint. Larger coupling and rollback cost. |
| C — only first-attempt receipt, unchanged Execution3 | Safe refusal can remain, but it cannot satisfy persisted single status/cancel/recovery/retry. It is a partial delivery, not an implementation of T26–T31. |

A Graph3 with one fake child is not an alternative: it contradicts the accepted
single-stage model. Treating child streams or Provider state as the ledger,
relaxing the closed codec, or recording fake workflow events is also rejected.

## Consequences

### Positive

Explicit attempt truth, with the same authority and classification invariants
as graph runs, preserves the separation between Execution and Agent. Execution3
and accepted graph history remain stable. Recovery and compatibility are
testable without real Runtime calls or broader filesystem/process permissions.

### Negative / trade-offs

The additional persisted kind and authority codec become public compatibility
obligations once written/released. Atomic publication, admission and recovery
must cover both the immutable workflow source and operational ledger; this is
the part that becomes expensive to change later. A new state store is more code
than a timestamp update, but avoids misrepresenting workflow truth. Current
empty-registry, unproven capability and Darwin observation gates still apply.

## Revisit when

The maintainer selects another explicit persistence contract, or a future
Execution-family design intentionally unifies gate and attempt records under a
new discriminator and approved migration policy. A passing component test,
green CI or an available preparation interface cannot resolve this decision.
