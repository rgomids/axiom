# Specification 007 amendment — #276 public stage dispatch, capability proof and coordination state

## Status

**Proposed, 2026-10-11, for human review.** Prepared with the
[#276 Plan](plan-tasks-276-stage-agent-execution.md) on Draft PR
[#311](https://github.com/rgomids/axiom/pull/311). The amendment is not
accepted. It changes no accepted text of [Specification 007](spec.md)
until a human accepts it, and acceptance would not authorize
implementation, dispatch, inference, merge or release. The companion
decision is [ADR-0024](../../decisions/0024-public-stage-dispatch-and-capability-proof.md)
(Proposed).

## Scope

This amendment **adds** contracts for WF-009 / AC-009, which Specification 007
assigns to #276 ([spec.md:62](spec.md), [spec.md:418-419](spec.md),
[spec.md:547](spec.md)). It does not modify:

- workflow definition ownership or revision binding (ADR-0020);
- the Project-owned or local Profile stores;
- the `runtimeprofile.Observation` contract;
- the HD-005–HD-008 skill surface.

The Profile-owned **effort** change is **out of scope**. It belongs to
PR #310 / #306, under the maintainer's D3 direction and that PR's mandatory
change 2. This amendment states only how #276 consumes the outcome (§A6).

## A1 — Public operations (additions to the boundary table)

| Operation | Request | Response / owner |
|---|---|---|
| `workflow stage status` | ExecutionRef, stage | Persisted children, attempts, statuses, classification reasons, record refs, stale/orphan, next; #276 |
| `workflow stage reconcile` | ExecutionRef, stage, attempt refs; preview/apply | Observations digest; appended reconciliation fact; #276 |
| `workflow stage integrate` | ExecutionRef, stage, graph revision; preview/apply | Existing Integration preview/execute; #276 |
| `workflow coordination show` | ExecutionRef, record ID | One canonical record; #276 |
| `runtime <codex\|claude> capability show` | Runtime, optional Profile | Capability states per binding; #276 |
| `runtime <codex\|claude> capability probe` | Runtime, Profile; preview/apply | Evidence record or inconclusive outcome; #276 |
| `runtime <codex\|claude> capability invalidate` | Runtime, evidence ref; preview/apply | Evidence state `unknown`; #276 |

These operations also complete the existing table rows `stage run/retry/cancel`
and `coordination list/publish`. `coordination publish` is valid only from a
running child context. Vendor-inference operations (`stage run`,
`stage retry`, `capability probe`) require external authority in addition to
local authority (Plan PD-12).

Additional categories use the existing completion statuses:

| Category | Completion status |
|---|---|
| `capability_unknown`, `capability_unsupported`, `capability_stale`, `workflow_effort_override_unsupported`, `result_missing` (operation-level) | `validation_failure` |
| `unsupported_graph_format` | `failure` |
| `cancellation_requested` | `success`, describing an observed request, never an observed stop |

## A2 — Execution capability vocabulary and proof

A child's **required capabilities** are the union of:

- its declared `capabilities`;
- `repository-write` and `process` when those effects are authorized;
- `noninteractive` and `axiom-coordination`.

| Capability | Meaning (within the dispatch worktree) |
|---|---|
| `read` | read files |
| `repository-write` | create and edit files |
| `process` | run commands |
| `noninteractive` | run headless from the task packet on stdin, with no prompts and a meaningful exit status |
| `axiom-coordination` | run the pinned `axiom` binary and publish into its own attempt inbox |

Model and effort support are recorded separately per exact model: as
`model-invocation` and as `reasoning-effort-<token>` in the existing
per-model slot.

**Evidence binding.**

- Every record binds Runtime ID, executable digest, observed version,
  invocation-mode digest (adapter argv template, confinement variant and
  environment allowlist) and adapter version.
- Model-scoped results also bind model and Profile effort.

**Proof rules.**

1. `proven` only from a behavioral calibration probe in the exact dispatch
   mode. Axiom verifies the probe by observing the filesystem, inbox and exit
   status, never by trusting what the agent says.
2. Installation, version, authentication, help output and Profile declaration
   never yield `proven`.
3. Interface observation (`--version`, `--help`) can yield only `unsupported`,
   when a required option or mode is absent.
4. Any of the following leaves a capability `unknown`: no evidence; a changed
   binding element; evidence older than the fixed maximum age; explicit
   invalidation; a disabled or removed Profile; an inconclusive probe;
   unproven authentication.
5. Preview and every dispatch revalidate the binding. Drift between preview
   and dispatch returns `denied_authority/stale_preview`. Any required
   capability that is not `proven` blocks dispatch, and nothing falls back.
6. Authentication failure is reported as `authentication_blocked`. It never
   changes capability evidence.

Implicit and model-scoped capabilities are checked against evidence for the
child's confinement variant. They do not require new Profile declarations.
Every coordinating variant is write-capable for the attempt inbox.

Confinement and network posture are measured by a deterministic `axiom`
canary run inside the child. The model's own claims never count. The
observed writable set is recorded.

Capability, effect authorization and confinement are distinct. The dispatch
preview shows all three per child, including whether write confinement is
`os-enforced` or `advisory`. Proof never grants authority.

## A3 — Coordination-to-graph state

At terminal save, after the final inbox drain, Axiom classifies each attempt
once. It persists the status, a classification reason and the qualifying
record digest in the same commit, and saves each attempt as soon as it is
classified.

Records are ordered by the canonical stream `Revision` that the supervisor
assigns in per-attempt candidate sequence. "Latest within the attempt" means
the highest such `Revision` for that attempt and the current graph revision.
`result.outcome` is a required enum (`succeeded` | `failed`). Stream capacity
reserves one terminal slot per attempt.

The first matching rule applies:

0. Cancel requested, stop confirmed, no ambiguous effect → `cancelled`.
1. Unknown process outcome → `unknown`. A timeout with confirmed stop and no
   ambiguous effect → `failed` (`timeout_stopped`).
2. The attempt's latest of {`blocker`, `result`} is a `blocker` → `blocked`.
3. Exit code ≠ 0 → `failed`.
4. Exit code 0 without a correlated `result` → `failed` (`result_missing`).
5. Exit code 0 with a `result` whose outcome is not `succeeded` → `failed`.
6. Otherwise → `succeeded`.

The exit code is retained. Scheduler readiness, integration eligibility and
status read only these persisted values. Records that arrive after the
terminal save have no effect. Reconciliation appends facts and never changes
a persisted status. Graphs without classification fields are read-only to
#276 operations.

Integration additionally requires `result_recorded` and a child result bound
to the qualifying record. Integration attempts carry their own reasons. A
graph-level classification-policy marker distinguishes #276 graphs.

Retry may follow `failed`, `blocked` or `cancelled` under explicit
authority. It may follow `unknown` or an ambiguous outcome only once a
reconciliation fact exists for that attempt; this changes the scheduler,
which refuses those outcomes unconditionally today.

## A4 — Dispatch authority

A `stage run` apply commits a dispatch-authority record with the graph, or with
the single-agent ledger attempt. The record holds the preview digest, its scope
(the first attempt of each child of one graph revision), the actor kind and
the time. This satisfies amendment #303 gap G-5. Retry, reconcile, cancel,
integrate and probe each require their own authority.

## A5 — Validator policy

Integration validators that run repository commands come from a separate
validator-policy artifact. The artifact is reviewed under its own preview and
authority, and the dispatch preview references it by digest
([spec.md:176-180](spec.md)). It is never taken from an agent-prepared Plan
document.

## A6 — Effort consumption (dependent on PR #310)

#276 reads effort from the resolved local Model Profile. The workflow
`effort` value is consumed as follows:

- `runtime-default` stays dispatchable;
- an explicit agent effort in a pinned revision is refused for new dispatch
  with `workflow_effort_override_unsupported`, and is never rewritten.

The schema and Specification text that retire agent effort are owned by
PR #310 and require their own acceptance.

## Compatibility

All format changes are additive:

- attempt classification fields;
- reconciliation facts;
- the dispatch-authority record;
- the stage-concurrency envelope field;
- the new capability evidence store.

Historical graphs, Executions and pinned revisions are not rewritten.

## Traceability

Plan §3.3–§3.10 and §12.2 (PD-1, PD-3, PD-7, PD-9, PD-12, PD-13, PD-15).
Maintainer direction:
[PR #311](https://github.com/rgomids/axiom/pull/311#issuecomment-6108112930)
and [PR #310](https://github.com/rgomids/axiom/pull/310#issuecomment-6108025068).
