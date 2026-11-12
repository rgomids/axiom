# CORRECT-399-01 — Capture recovery decision pending

2026-11-12. Inspected source: `74d5cf393eabbb3ada9d69f198da0b41dba5aabb`,
[PR #399](https://github.com/rgomids/axiom/pull/399),
[CR-399-F1](https://github.com/rgomids/axiom/pull/399#pullrequestreview-5873043544).
Status: **BLOCKED pending bounded integration-contract decision**. This report
does not implement the correction, close the finding, or authorize merge.

## Confirmed defect and constraints

`Engine.integrateHeld` calls `CommitPending` before observing the target and
before persisting `PrepareIntegration`. `Integrator.CommitPending` stages the
writer's files, writes a commit and advances its assigned branch by CAS. A
target observation failure, rejected preparation, lost fence or interruption
can therefore leave a principal-created candidate with no capture record.
Reentry sees `head != base` and treats it like a writer-created commit.
The original `ErrIntegrationRefused` comment promised no recorded intent and no
Git effect; that promise is false on this path. This checkpoint corrects the
comments and refusal documentation only, without changing runtime behavior.
The other corrections in #399 are unaffected.

The [approved AXM-43 amendment](amendment-axm-43-same-repository.md), rule 2,
requires nested Execution ownership and explicitly excludes another store,
sidecar manifest or Git-ref authority ledger. Rules 5–7 require fenced
integration, truthful uncertainty and no reconstruction of authority from refs.
[ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md)
also requires one aggregate/store. A private Git journal is therefore not an
authorized local implementation shortcut.

The current `PrepareIntegration` and `validateIntegrations` require an existing,
non-base `CandidateSHA`. A dirty worktree at base cannot satisfy that invariant
before capture. Merely moving `ObserveTarget` earlier closes one failure path;
it does not close crash or preparation failure. Changing the error text alone
does not make the mutation recoverable. A fabricated candidate SHA is not
evidence and must not be used to bypass preparation validation.

## Proposed decision, not adopted

**Recommend:** permit a bounded capture intent nested in the existing Execution,
under its existing store/CAS/runner fence. Keep existing public status names,
Workflow schema, authority, gates and integration checks. Explicitly authorize
the additional pre-candidate persisted phase and its recovery semantics before
implementation; do not silently reinterpret historical records.

| Alternative | Trade-off |
| --- | --- |
| Nested capture intent (recommended, pending authority) | Preserves the single truth owner; requires correlated intent/outcome fields, immutable transition validation, strict codec compatibility and recovery tests. |
| Require writer commits and refuse all pending work | Retains the historical committed-candidate contract but withdraws the Codex pending-work capability delivered in #399; requires a product/contract decision. |
| Independent journal or ref ledger | Could record capture independently, but contradicts the approved no-sidecar/single-store boundary; excluded from this correction's authority. |

The nested proposal must persist exact source-attempt/repository/worktree/branch/
base correlation before the first Git mutation. Before source-ref CAS it must
also persist the exact generated candidate, so recovery can distinguish an
observed known effect from an uncorrelated one. Preserve the writer's pending
files and index; do not reset, clean, delete or advance a foreign ref. A lost
runner must not repeat capture automatically. Observations alone cannot grant
authority or turn an uncertain capture into a writer-owned commit.

Once real Executions contain capture facts, their correlation, immutability and
recovery meaning become compatibility obligations. The decision is material
because `prepared` currently certifies an already observed committed candidate;
using it before capture changes that persisted invariant even without adding a
status name. No ADR or accepted specification is amended by this proposal.

## Implementation and validation after authorization

Affected producers/consumers: `internal/workflow/execution_integration.go`,
Execution codec and transition guards in `internal/local`,
`internal/stageexecution/integration.go` and runner-loss recovery,
`internal/gitworkspace/integration.go`, their fault fixtures and directly
affected operational/refusal documentation. Reuse the existing aggregate/store;
do not change command environment, artifact locking, command admission or agents.

Required faults: refused intent before any Git mutation; target observation
failure; cancellation/lost fence before capture and before source CAS; crash
after staging/object creation, after candidate recording, after source CAS and
before integration preparation/outcome recording. Reload through the real
store, observe exact source/target identities, distinguish known and unknown
outcomes, and assert no blind replay. Include moved source/target refs, executable
Git configuration, staged/unstaged/untracked file preservation, writer-owned
commits and repeated entry. Retain existing post-merge checks and target CAS.

No correction acceptance test can be reported PASS until that protocol exists.
Baseline test results and delivery SHA are reported in the stage receipt; they
do not close CR-399-F1. Real-vendor E2E evidence remains historical. No external
inference, merge, release, deployment or Product acceptance is authorized here.
