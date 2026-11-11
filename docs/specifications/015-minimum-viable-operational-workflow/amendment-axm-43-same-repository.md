# AXM-43 — Same-repository workflow run integration amendment

Revision 2, 2026-11-11. Status: **option A approved by Rafael**; not implemented.
Implementation follow-up: [AXM-43 implementation Evidence](evidence-axm-43-implementation.md)
records the later code and validation. The analysis below describes the inspected
pre-implementation base and preserves Revision 2's approval history.
Revision 1 (same day) proposed options A/B/C as `DECISION_PENDING (Rafael)`.
Decision, 2026-11-11 11:50 BRT, literal: “Aprovo a opção A da P-AC3 na #390”
([decision record](https://github.com/rgomids/axiom/pull/390#issuecomment-6799056487)).
Related: AXM-43 (AXM-1 sub-issue), [#278](https://github.com/rgomids/axiom/issues/278).
Inspected base: `e5b9d10880cfc210356b8c49d639b1718b494e01` (v0.23.6).
The approval authorizes this Specification amendment and its Plan/Tasks. It is
not P-AC3 implementation or acceptance. No historical Specification or accepted
ADR is amended in place.

## Contract analysis before code

The automatic composition itself fits the approved direction. It does not need
an additional public flag, command, store or schema family. However, a safe
production implementation also needs a durable **pre-spawn workspace binding**
and an explicit enrollment/eligibility rule. Neither is an existing persisted
contract. Revision 1 therefore asked for approval of the bounded amendment below
before wiring, rather than treating a new crash/recovery contract as an incidental
launcher detail. Rafael approved it (option A). This is not a claim that ADR-0037
forbids extending its nested records.

Exact applicable clauses:

| Source | Exact text | Implication |
| --- | --- | --- |
| [Spec 015, Parallel work and integration](spec.md#parallel-work-and-integration) | “recommend an isolated Git Worktree and independent branch per agent” | Isolation is already intended; no independent Git service is justified. |
| Same section | “records final integration status before marking work integrated” | Agent success cannot stand in for integration success. |
| [Spec 015 P-AC3](spec.md#normative-technical-completion-and-later-product-acceptance) | “safely integrate same-repository work under the principal Orchestrator” | The public run composition must reach the integration engine. |
| [Plan TD-03](plan.md#td-03--scheduling-messages-and-integration) | “Record repository identity, base SHA and agent result SHA” | The principal needs trustworthy source/base correlation. |
| Same section | “Dirty/changed base or conflicts stop integration explicitly.” | Never derive a fresh base after an interrupted attempt or overwrite a checked-out branch. |
| Same section | “Persist attempt preparation under fence before launch” | Freeze workspace identity before materialization/spawn; recovery cannot guess it from agent output. |
| [ADR-0037 decision 1](../../decisions/0037-one-canonical-execution-for-new-runs.md#decision) | “One Execution aggregate.” | Workspace binding belongs in the existing aggregate. |
| ADR-0037 decision 3 | “A crash after preparation or spawn without correlated terminal evidence is `unknown`; it is never respawned automatically.” | A workspace creation or integration gap cannot authorize replay. |
| [Spec 015 safety obligations](spec.md#safety-proof-obligations) | “authorization cannot be expanded by agent output or stale state” | Do not infer branch ownership or check authority from a successful agent. |

### Concrete gap at the inspected base

- `internal/workflow/execution.go:Attempt` freezes invocation and repository/effect
  authority, but no worktree, source ref, base or integration target. Repository
  binding is key + identity, without branch ownership.
- `internal/workflow/execution_integration.go:PrepareIntegration` only admits a
  **succeeded** attempt, a committed candidate distinct from base, write/check
  authority, actual stage commands and an approved human gate. Its existing
  `IntegrationPrepared` cannot represent pre-spawn workspace reservation. Reusing
  it would change those invariants, not merely add a production caller.
- `internal/stageexecution/runtime_launcher.go:forAttempt` selects a repository
  root, not an attempt root. `cmd/lingo/execution_run.go:plannedLauncher` supplies
  those roots. `RuntimeLauncher.Build` and `Collect` would both need the frozen
  attempt workspace; `Inspect` must remain read-only.
- `internal/stageexecution/scheduler.go:runReady` holds the runner lease.
  `Engine.Integrate` acquires that lease independently. Calling it inside the
  loop without extracting a lease-held operation cannot compose the two safely.
- `internal/workflow/execution_transitions.go:Ready` accepts succeeded same-stage
  dependencies; `derivedStageStatus` can complete a stage with no integration
  records. A principal crash after result commit but before integration intent
  leaves no persisted distinction between ordinary success and integration owed.
  Merely appending integrations after `Run` would let successors run too early.
- CLI outcomes already re-observe recorded Git integration facts. They cannot
  prove that an integration which was never recorded was required or completed.

**Counterexample:** two writers finish successfully; the principal dies after
result commit, before any integration record. A new run currently sees validated
results and may launch the dependent agent in the original repository. That
repository does not contain their isolated commits. Retrofitting a guess from
branch naming or the current Workflow cannot establish which old attempts were
enrolled, their original base, or authority to create/adopt those refs.

This is why no independently useful runtime slice is shipped in this amendment PR.
Changing only CWD would strand results; integrating only after `Run` would leave
the scheduling hole; modifying existing integration admission would weaken its
validated-result invariant. The existing primitives remain usable and tested.
DF-S23-07 does not itself block automatic integration: a new public resolution
command is explicitly excluded from the recommendation.

## Approved Specification amendment (option A)

Approved by Rafael on 2026-11-11 11:50 BRT
([decision record](https://github.com/rgomids/axiom/pull/390#issuecomment-6799056487)).
These rules bind the AXM-43 implementation; none is implemented yet.

1. Enroll **new Executions only** automatically when a Stage contains at least
   two agents with `workspace_write` targeting the same resolved Repository key.
   Count configured writers, including dependent writers, not completion order.
   Existing Executions with no enrollment retain their historical semantics;
   do not reinterpret or replay their attempts.
2. Persist immutable enrollment and per-attempt workspace/base/source/target
   correlation inside Execution format 4 before any worktree/ref creation or
   spawn. Exact optional field layout is an implementation choice after approval;
   maintain bounded validation, immutable transitions and old-record decoding.
   No additional store, sidecar manifest, Git-ref authority ledger or schema family.
3. Principal-generated refs and a private confined sibling worktree root are
   scoped to Execution + Repository + attempt. Never adopt an existing collision.
   Use a new principal-owned target, never the operator's checked-out branch.
   Record initial target HEAD and refuse dirty/changed base. Retain all worktrees,
   branches and commits on success or failure; no reset, deletion or cleanup.
4. Require configured post-merge checks and existing `validation_command`
   authority (`--authorize-commands` already exists). Missing authority/checks
   refuses before launching enrolled writers. No implicit grant or placeholder
   check. Instruct isolated writers to commit their result on their assigned
   branch; uncommitted/dirty output cannot integrate.
5. Integrate validated results serially under the principal's existing runner
   fence. Preserve `Integrations`, expected target HEAD, CAS, target observation,
   post-merge command checks and cancellation semantics. A failed check/conflict
   remains `not_integrated`; uncertain effects remain `unknown`, without replay.
6. Enrolled succeeded writers unblock dependencies and stage advancement only
   after required integration is recorded. Dependent repository readers/writers
   receive the accepted integrated tree. Preserve gates; integration must not
   synthesize human approval. No automatic resume from conflict/check failure.
7. Crash after workspace intent or before a terminal/integration observation
   preserves uncertainty and stops dependent work. Never reconstruct authority
   from refs, paths, messages or runtime output. Public recovery remains diagnosis
   and safe stop; adding public `ResolveIntegration` is a separate decision.

## Decision and considered alternatives

| Option | Outcome | Trade-off |
| --- | --- | --- |
| A — Nested enrollment/workspace binding and eligibility amendment | **Approved** | One owner, no new public ceremony; adds persisted semantics and requires compatibility/fault tests. |
| B — Keep current aggregate unchanged; infer ownership from deterministic refs/paths | Not chosen | Smaller initial diff, but either loses frozen enrollment/base across crashes or introduces a second authority protocol. |
| C — New explicit integrate/resolve command or launcher manifest | Not chosen | Makes target selection visible, but adds public ceremony or a separate state owner; requires a wider contract decision. |

Option A, including new-Execution-only enrollment and nested immutable binding,
is approved as a bounded realization of ADR-0037; no new ADR is needed or marked
accepted here. A prospective ADR is required only if implementation would change
an accepted architectural clause; that stops as `DECISION_PENDING (Rafael)`.
Persisted binding and eligibility semantics become harder to change once real
attempts exist. P-AC3/P-AC4 remain full requirements, not reduced criteria.

## Authorized implementation Plan and Tasks

Implementation lands in a separate PR linked to AXM-43, stacked on this one. It
follows these tasks and the approved rules above; anything outside them (new
public flag/command, store, ADR or other contract change) needs a new decision.

1. Add bounded optional enrollment/binding to the existing Execution; validate
   immutable identity, compatibility, dependency readiness and truthful status.
   Extend store transition guards and inventory/corpus checks as required.
2. Reuse the hardened Git boundary to observe clean base, create new refs and
   worktrees only after intent, bind launcher Build/Inspect/Collect to the same
   attempt workspace, and preserve attempt/message correlation.
3. Extract the lease-held integration operation for the existing principal loop;
   keep standalone `Integrate` behavior. Recheck Project/repository admission and
   effects before integration/checks/ref advance. Wait for running peers without
   starving their fenced result/message callbacks.
4. Wire the actual `workflow run` composition. Reconcile CLI status/outcomes,
   canonical runtime skills, operational docs, help/man and installed metadata
   together. Do not advertise unimplemented behavior as installed functionality.
5. Exercise the public composition using fake runners, temporary Git repositories
   and synthetic state roots: parallel writers and dependent workspace contents;
   same-file conflict; failed post-merge check; lost result/intent/advance outcome;
   dirty or changed base; collision; missing checks/authority; cancellation; gate;
   old unenrolled Execution. Assert no successor or unauthorized/double effect.

Acceptance for this docs-only PR: cited contract analysis, recorded decision,
considered alternatives, authorized Plan/Tasks, executed check evidence and one
open review PR. [Evidence](evidence-axm-43-contract.md) reports existing tests
separately from the still-unimplemented public composition.
