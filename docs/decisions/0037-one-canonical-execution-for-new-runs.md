# ADR-0037 — One canonical Execution for new workflow runs

## Status

Accepted on 2026-10-30 by the [human material decision on PR #366](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5720318529),
which explicitly approved [Specification 015](../specifications/015-minimum-viable-operational-workflow/spec.md)
correction-04 and its proposed architecture and contract evolution, including
the exact amendment map of the
[AXM-42 research](../research/axm-42-minimum-viable-workflow-investigation.md#exact-amendment-implications-proposed-history-preserved).
This ADR records that decision and the [Plan](../specifications/015-minimum-viable-operational-workflow/plan.md)
technical choices TD-01–TD-04; it introduces no new material direction. It
grants no merge, release, cutover of real installed state or Product
Accepted. S1 implementation authority is recorded separately on AXM-42.

## Context

The operational path keeps three durable truth owners (WorkflowStore,
GraphStore, SingleStageStore), a mandatory operator Plan document and reviewed
Validator Policy, predictive capability/complexity selection, a mandatory
authentication preflight and repeated preview/apply ceremonies. Spec 015
requires deleting these rather than hiding them, while preserving the proven
safety outcomes: exact target/effect authority, single-writer fencing,
process-identity safeguards, truthful unknown effects and no blind replay.

## Decision

For **new** workflow runs:

1. **One Execution aggregate.** A versioned `Execution` (`kind: execution`,
   `formatVersion: 4`) in `internal/workflow` owns Workflow binding and digest,
   stage progress, nested agent attempts, frozen invocation snapshots,
   attempt authority, process correlation, cancellation, confirmed/unknown
   effects, bounded messages and artifact references. Attempts are nested
   records, not child Executions, Graph nodes or a separate ledger.
2. **One store.** `internal/local` persists it under `executions/v2` with
   strict bounded decoding, rooted no-symlink paths, ADR-0007 protected
   replacement, exact-byte revision CAS and a runner lease whose monotonic
   generation is recorded in the aggregate. Lock order is runner lease, then
   aggregate lock. Old bytes are never decoded as the new type.
3. **One engine.** `internal/stageexecution` runs one or many agents through
   one fenced loop: persist the prepared attempt before spawn, observe process
   facts, validate outputs, commit results and make eligible successors
   runnable automatically. A crash after preparation or spawn without
   correlated terminal evidence is `unknown`; it is never respawned
   automatically. Human gates and unknown effects stop successors.
4. **Invocation ownership.** The frozen per-attempt invocation snapshot is
   the sole effective Runtime/model/effort/instructions authority. Per field:
   explicit per-agent run override > explicit Workflow agent value >
   explicitly configured machine-local default; a Profile supplies optional
   defaults at its level and explicit fields at that level replace them.
   Same-level conflicts and missing values refuse. Effort is an opaque token or
   explicit `runtime-default`. No automatic fallback, ranking or generated
   Profile.
5. **No predictive admission.** Normal runs perform no mandatory
   authentication/compatibility preflight and no capability/complexity
   selection; vendor, spawn and auth errors are observed attempt failures.
   Real permission, target and effect checks remain.
6. **Messages.** One private attempt-owned ingress carries bounded
   question/answer/blocker/result facts bound to launcher-issued attempt
   context and runner generation. Messages never grant authority.
7. **Breaking cutover.** Only new-format Executions run after the pre-MVP
   cutover. No migration, conversion, old-attempt resume or active legacy
   engine. Raw old state is preserved as inert backup where feasible;
   active/unknown legacy attempts are stopped safely or the cutover is BLOCKED.

## Supersedes

- [ADR-0009 parent, child and graph semantics](0009-parent-child-execution-graph.md#parent-child-and-graph-semantics):
  only durable child Executions and separate graph lifecycle for new runs.
- [ADR-0020 exact relationship to accepted decisions](0020-workflow-definition-revision-binding.md#exact-relationship-to-accepted-decisions):
  only the separate sequential/graph executor assertion; immutable definition
  binding and rework linkage remain.
- [ADR-0024 Decision](0024-model-profile-effort-ownership.md#decision) and
  [compatibility scope](0024-model-profile-effort-ownership.md#compatibility-and-evolution-scope):
  only sole Profile effort ownership, the prohibition on direct overrides,
  Profile-only projection and mandatory operational old-format readers.
- [ADR-0025 Decision](0025-public-stage-dispatch-and-capability-proof.md#decision):
  only the per-graph supervisor/single-ledger split, the separate dispatch
  authority record and graph-owned coordination ingress for new runs.
- [ADR-0026 Decision](0026-successor-stage-dispatch-graph-format.md#decision-proposed-detailed-contract):
  only successor Graph as operational owner, graph CAS and graph admission for
  new runs; Graph1–4 bytes remain inert history.
- [ADR-0027 Decision](0027-single-stage-attempt-persistence.md#decision):
  only Option A's separate single-stage ledger and authority codec for new
  runs, and the rejection of alternative B.
- [ADR-0032 Decision](0032-runtime-admission-without-certification.md#decision):
  only the Project Runtime Policy selection references; no certification is
  reintroduced.
- [Specification 007](../specifications/007-configurable-workflows/spec.md),
  [amendment #276](../specifications/007-configurable-workflows/amendment-276-stage-dispatch.md),
  [amendment #306](../specifications/007-configurable-workflows/amendment-306-profile-effort-onboarding.md)
  and the [Spec 004 auth preflight](../specifications/004-mvp-v1-baseline/issue-272-cli-subscription-auth-preflight.md):
  only the clauses mapped by the AXM-42 research amendment table.

Retained without change: ADR-0003/0005/0006 boundaries, ADR-0007 publication
outcomes and ADR-0028 process-group supervision limits.

## Alternatives considered

| Option | Trade-off |
| --- | --- |
| Keep Graph/Single and add a facade | Two authority writers and recovery paths remain; fails Spec 015 |
| New database or event sourcing | New dependency without evidence |
| One bounded JSON aggregate (selected) | Simpler ownership; whole-record rewrite and contention cost, reconsidered only with measured evidence |

## Consequences

- Positive: one writer, one recovery path, one run action, no mandatory
  Plan/Policy/Profile ceremony.
- Negative: breaking cutover; persisted attempt layout and retired public
  verbs become costly to reverse; old-format operational inspection is gone.

## Revisit when

- measured aggregate size or contention exceeds bounded storage limits;
- parallel S2 completions show writer starvation;
- a stable/LTS release begins strict compatibility obligations.
