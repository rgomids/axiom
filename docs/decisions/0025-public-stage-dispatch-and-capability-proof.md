# ADR-0025 — Public stage dispatch supervision, capability proof and coordination ingress

## Status

Proposed, 2026-10-11, for human review on Draft PR
[#311](https://github.com/rgomids/axiom/pull/311) (Issue
[#276](https://github.com/rgomids/axiom/issues/276)). It is not accepted. It
authorizes no implementation, dispatch, inference, merge or release. The
number 0025 avoids collision with ADR-0023 and ADR-0024, both of which
PR #310 proposes and which were accepted there on 2026-10-11 but are not yet
merged.

## Context

[ADR-0009](0009-parent-child-execution-graph.md) defines the parent/child
graph, its outcomes and its cancellation semantics.
[ADR-0007](0007-local-publication-and-recovery-protocol.md) defines
publication, lock order and recovery.
[ADR-0005](0005-bounded-local-filesystem-threat-model.md) bounds the local
threat model. The graph and coordination code is reachable only from a
checkout-built runner.

Making it public raises four long-lived questions:

1. **Initiating sessions are short-lived.** The Runtime tool calls that start
   a stage time out long before child agents finish.
2. **Authority must be recorded before dispatch.** Amendment #303 G-5 calls for
   durable dispatch authority before any child starts.
3. **Children must not write canonical state.** They still need to exchange
   records.
4. **Capability claims need proof.** The production observer proves only
   `axiom-skills`, so real stages would be refused. Capability must be proven,
   not assumed (maintainer decision D-1 on PR #311).

## Decision

Subject to human acceptance:

1. **Supervisor.** A `stage run` apply starts one detached supervisor process
   of the same `axiom` binary per graph run. It is not a daemon. The
   supervisor holds a non-blocking per-graph run lease with a monotonically
   increasing epoch, placed first in the ADR-0007 lock order. It dispatches
   only work that is already authorized. It exits when no more work is
   dispatchable or when cancelled. Children run in their own process group or
   job object.
2. **Dispatch authority record.** The record is committed with the graph, or
   with the single-agent ledger attempt, before any attempt starts. Its scope
   is the first attempt of each child of one graph revision. Every other
   mutation needs its own authority. Vendor inference needs external authority
   in addition to local authority.
3. **Coordination ingress.** Each child writes candidates only into its own
   attempt inbox. The lease holder ingests them concurrently. It drains each
   attempt's inbox one last time before that attempt's terminal save, and
   quarantines inboxes from earlier epochs. Records never mutate graph state,
   authority or gates.
4. **Persisted classification.** At terminal save, the attempt's process
   result and drained records are classified once into existing ADR-0009
   statuses (`cancelled`, `unknown`, `blocked`, `failed`, `succeeded`).
   Records are ordered by the supervisor-assigned stream revision. Each
   attempt is saved on its own. Integration is bound to the qualifying
   `result` record. A retry after `unknown` requires an appended
   reconciliation fact. The reason and the
   qualifying record reference are saved in the same commit, and the exit code
   is retained. The scheduler, integration and status read only these
   persisted fields. Reconciliation appends facts and never rewrites a status.
5. **Capability proof.**
   - Execution capabilities are proven only by an authorized behavioral probe
     in the exact dispatch mode, verified by Axiom's own observation of
     effects.
   - The resulting evidence is bound to Runtime, executable digest, version,
     invocation mode and, where applicable, model and effort. It is
     revalidated at preview and at dispatch, and expires after a fixed age.
   - Interface observation can only refute a capability.
   - Installation, version, authentication and Profile declaration never prove
     one.
   - Capability, effect authorization and confinement are recorded and shown
     separately.
   - Confinement is reported as `os-enforced`, `advisory` or `unknown`, as
     observed. `os-enforced` needs a canary-observed denial bracketed by a
     same-identity positive control (pre- and post-run writes to the same
     unchanged target outside the sandbox). Otherwise the result is
     `unknown`.
   - Each confinement variant is probed with its own tasks in its exact
     dispatch mode; `read-only` never needs a workspace write. Evidence is
     variant-bound.
   - Probes mutate only an Axiom-owned envelope whose manifest digest is
     authorized; untested roots are reported `not_tested`.
   - Temp roots can be tested only by a separately authorized extended
     probe.
   - User homes, vendor homes and external networks are never probed;
     network checks use an Axiom-owned loopback listener only.
   - Interrupted probes quarantine their envelope and yield `unknown`
     evidence.
   - Evidence keeps digests and categories only: no host paths,
     environment values, configuration, credentials or transcripts.
6. **Consumption of the accepted #306 contract.** Dispatch admission follows
   the accepted amendment-306 and ADR-0024: schema-2 Workflows, format-3
   Executions, frozen `resolvedProfile` tuples and PE-05a effort-source
   checks. Legacy contracts are refused, never reinterpreted.

## Alternatives considered

- **Foreground dispatch inside the caller's tool call.** Simplest, but it is
  killed by host tool timeouts and orphans children.
- **Long-lived daemon or broker.** It adds a control plane, which ADR-0003
  forbids without necessity.
- **Children writing the canonical coordination store.** This widens the
  child write surface to shared state.
- **Classification at render time only.** Status, the scheduler and
  integration could then disagree, which is the defect named in CR-001.
- **A separate qualification field that leaves `succeeded` as exit 0.** Both
  the scheduler and integration would need new rules.
- **Capability from interface or version observation.** This is not proof,
  and it is excluded by the maintainer's direction.
- **Probing before every dispatch.** It is correct but repeats inference cost.

## Consequences

- There are format changes, versioned together with the #306 graph v2 rather
  than added silently:
  - attempt classification fields, the lease epoch and process correlation;
  - reconciliation facts;
  - the dispatch-authority record;
  - a new machine-local capability evidence store.
- `succeeded` now means the child's contract is fulfilled, not only that the
  process exited 0.
- Write confinement for Claude children is advisory unless proven. The human
  sees this before authorizing dispatch.
- A same-user forgery risk remains, within ADR-0005. Records and evidence
  grant no authority.
- Probes consume vendor inference and require explicit consent.

## Revisit when

- a Runtime offers verifiable OS confinement for all children;
- vendors provide machine-readable capability declarations whose integrity
  can be verified; or
- multi-machine execution is approved.
