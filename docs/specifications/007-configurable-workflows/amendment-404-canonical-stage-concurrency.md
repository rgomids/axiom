# Issue #404 — Canonical Stage concurrency

## Status, authority and bounded delta

SPECIFY-404-02, 2026-11-24. Normative amendment within the scope
[authorized by Rafael](https://github.com/rgomids/axiom/issues/404#issuecomment-7133606313),
subject to amendment/compatibility qualification. This records requirements, not
implementation approval, behavioral PASS, merge or human delivery acceptance.
The [Intake AC1–AC4](https://github.com/rgomids/axiom/issues/404#issuecomment-7132349983),
[original baseline](https://github.com/rgomids/axiom/issues/404#issuecomment-7132672246),
[historical BLOCKED receipt](https://github.com/rgomids/axiom/issues/404#issuecomment-7132912608)
and [requalification readiness](https://github.com/rgomids/axiom/issues/404#issuecomment-7133640003)
remain history; none is republished as a new decision.

Affected Spec 007 clauses: [WF-005/006 Stage and agent semantics](spec.md#stage-and-agent-semantics-wf-005006)
(concurrency ceiling, immutable revision and DAG dependencies), and
[WF-007/008/010 Stage planning and Runtime boundary](spec.md#stage-planning-and-runtime-boundary-wf-007008010)
(lossless transfer, permitted overlap, isolated writers and integration).
Read them with accepted [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md)
and its prior supersessions: one Execution, store and engine own new runs.
Historical graph ceilings are not a second owner or an implementation option.

Delta from SPECIFY-404-01: mapper/tests-only is replaced by preservation and
application of the declared Stage limit in canonical StageSpec, WorkflowSnapshot,
Ready/scheduling, mapper and future tests. This is the authorized Spec 007
amendment, not a new numbered Specification or ADR. Design: NOT_APPLICABLE.
No new interface, flag, service, launch policy or authority mechanism is needed.

## Source qualification at the effective HEAD

Inspection HEAD `b42d6e5e56f1ef51e0f4153cd024473d973494f8`; production sources
are unchanged from base `5cd783bd2ac8189a1b9c85a790472b5335139c78`.
[Earlier source Evidence](issue-404-dag-concurrency-specify.md#contracts-and-verified-source-path)
is preserved. Exact source, rather than graph freshness, supports these findings:

| Source | Consequence |
| --- | --- |
| [Mapper](../../../cmd/lingo/execution_run.go), `mapWorkflowDefinition` | Encode validates first; `s.Concurrency != 1` refuses; StageSpec has no destination for the limit. Removing the guard alone loses the limit. |
| [Definition validation](../../../internal/workflowdefinition/validation.go) and its schemas | Existing schema bounds, agent-count bound, sequential 1, cycles, references and owner requirements remain applicable. |
| [Execution](../../../internal/workflow/execution.go), StageSpec, cloneStages, workflowDigest, validateWorkflow, Encode/DecodeExecution | Format 4, kind execution; StageSpec lacks a ceiling. Digest hashes JSON of ID/revision/Stages, including member order from Go structs. Clone uses JSON. Decode recomputes digest and validates. |
| [Strict codec](../../../internal/strictjson/shape.go) | Bounded strict decoding plus exact round-trip shape rejects unknown/mis-cased members, missing required members and null substitutions. A new required/default-emitting member would break existing canonical bytes. |
| [Store](../../../internal/local/execution_store.go), read, validateExecutionUpdate | Reads use canonical decode; updates preserve Workflow by deep equality and use exact-byte CAS, protected publication and runner fencing. Normal progress must not rewrite a snapshot to insert a default. |
| [Ready/PrepareAttempt](../../../internal/workflow/execution_transitions.go) | Ready returns all dependency-eligible unattempted tasks. Preparation rechecks readiness under generation fencing; retry has a distinct path that also needs the bound. |
| [runReady](../../../internal/stageexecution/scheduler.go) | Principal prepares sequentially, starts workers, owns inflight refs and services their commits. A precomputed ready list alone cannot enforce capacity. |
| [Claim/orphan recovery](../../../internal/stageexecution/engine.go) | Older prepared/running attempts become unknown, not automatically respawned. Existing integration/delivery recovery remains separate and mandatory. |

Spec 011 VP-04/E3 is a presentation fixture only. Reported ubuntu-t12 r1
refusal and zero processes were not rerun. Its digest remains
`80b733ae0281f99c4b5f6714db078952580242dbd7fd6b05e623adb86f550df8`.

## Normative behavior

**DC-01 — Preserve intent.** Every newly mapped valid Stage MUST carry its
declared positive integer concurrency unchanged into the immutable canonical
WorkflowSnapshot. Its bounds MUST satisfy definition/schema and agent-count
constraints. Sequential remains 1. DAG 2 MUST NOT be refused solely for that
value; separate unsupported controls still refuse. No artificial edges, changed
revision, inferred owner, reduced declared value or historical engine may hide
missing enforcement. Definition array Stage ordering remains unchanged.

**DC-02 — Apply capacity at admission.** For each Stage, the number of occupied
agent-attempt slots MUST NOT increase above the effective ceiling. Dependency
eligibility is necessary, not proof of admission or real concurrent execution.
Ready, preparation (including explicit retry) and runReady MUST agree on the
same persisted limit and current occupancy, with admission rechecked at the
canonical fenced mutation before a launch. A cached list or a worker-local
counter cannot independently grant capacity. Re-evaluate after each admission
and as capacity returns; a runnable independent root can use the released slot
without adding dependencies. This does not promise wall-clock overlap, fairness
machinery or launching work while another existing control blocks it.

Occupation starts with durable `prepared`, before process spawn, and includes
`running`, validation/output collection and any still-inflight worker for that
attempt. Count the union of durable active attempts and principal inflight refs,
without double counting an attempt; a terminal commit alone must not release a
slot while its worker still performs attempt work. Release after terminal facts
and worker completion are observed, including launch failure with no process.
No release may assume a process exited from an absent callback. Unknown is not
a free-slot permission: existing recovery-required/unknown-effect controls stop
new dispatch until their existing explicit resolution contract permits it.
Integration remains governed by its existing serialized/validated contract;
completion of an agent slot does not imply its results have been integrated.

**DC-03 — Preserve controls.** Capacity is an additional ceiling, never a grant.
DAG prerequisites and joins require validated predecessor results and required
integration; gates, inputs, output checks, effect conflicts, repository writer
isolation, integration ownership, timeouts, budgets, cancellation, failures,
unknown/no blind replay, invocation and target authority remain authoritative.
Apply the narrower existing authorized restriction wherever present. Cancellation
or failure stops further dispatch while already running peers retain truthful
outcomes. Stale generations/CAS conflicts cannot duplicate preparation or spawn.
Messages do not grant permission. [ADR-0039](../../decisions/0039-unrestricted-runtime-stage-processes.md)
and [Spec 015](../015-minimum-viable-operational-workflow/spec.md) launch/authority
boundaries are unchanged; a ceiling does not confine Runtime subprocesses.

## Persisted compatibility and defaults

**DC-04 — Additive, presence-aware compatibility.** Keep canonical kind/format 4
and store ownership. Extend the Stage snapshot additively with an optional
positive integer limit; new mapping MUST emit it explicitly, including 1.
Existing canonical snapshots without that member MUST remain readable,
inspectable and resumable through the same store/engine without migration or
snapshot rewriting. Absence is a historical encoding distinction, not evidence
that a definition declared 1. Never fetch current Workflow configuration to
reconstruct historical intent or insert a value into a frozen snapshot.

For an absent limit, use a conservative **effective dispatch ceiling of 1**, as
an execution-time fallback only. This can narrow old unconstrained scheduling;
it cannot increase old concurrency or authority. It is not a claim about a
historical default. Retained records already containing multiple active attempts
remain valid/readable: do not cancel, rewrite or reject their history solely for
this new fallback. Admit no additional work while occupancy is at or above 1;
recover orphaned attempts as unknown through the existing protocol. Resolve
unknowns explicitly before eligible future work; never replay old attempts.

**DC-05 — Preserve hashing and strictness.** For absent-limit snapshots, preserve
the original Stage member order and serialization of all existing fields,
absence of the new member, and original digest input. Validate the stored digest
before applying an effective fallback. Decode/encode, cloning and routine
fenced progress MUST preserve the frozen Workflow and digest. `omitempty` alone
is insufficient: eager defaulting would emit 1, change the shape/hash and violate
store equality. A presence-preserving optional representation can leave absent
bytes unchanged and hash explicit new values. Exact representation is a local
implementation choice subject to these oracles, not a mandated abstraction.

Explicit zero, negative, fractional, null, excessive/above-agent limits, duplicate
or mis-cased/unknown fields and tampered digests MUST be refused by the relevant
strict/bounded validation path. Absence MUST NOT hide an explicit invalid value.
Changing an explicit limit without its digest must fail validation; matching a
recomputed digest does not authorize modifying an existing store snapshot.

Compatibility direction: the amended reader must accept existing canonical
format-4 records, not legacy pre-cutover formats. The current strict binary
will refuse records carrying the new member; downgrade consumption of new
records is unsupported and must fail closed, never strip the member. No format
bump, legacy resume, conversion, destructive migration or forced rollback rewrite
is authorized. ADR-0037's legacy cutover does not permit breaking present
canonical records. If these additive compatibility oracles cannot be met,
stop dependent work for a material decision; do not silently choose a new format.

Source-based feasibility: workflowDigest marshals StageSpec without normalizing;
strictjson compares input against the marshaled shape; store compares frozen
Workflow by value. Keeping absence through all three paths avoids the identified
hash/shape/equality break. This establishes a compatible design path, **not** a
performed codec/resume proof. Implementation qualification must supply frozen
pre-change bytes and digests, not fixtures regenerated by the amended encoder.
No new ADR is presently indispensable; a contradictory implementation finding
reopens this gate.

## Acceptance and deterministic future Evidence

Origin ACCEPTED: Intake and the linked scope decision. AC remain AC1–AC4;
these cases are means of proving them, not additional product scope.
All implementation cases below are **NOT_RUN** in this documentation attempt.
Use local synthetic definitions/stores and controlled stub launchers only.

| AC / case | Deterministic oracle |
| --- | --- |
| AC1/AC2: valid DAG 2, A/B independent, C join and declared owner | Mapper preserves exact limit/edges/controls; canonical run creates one Execution and starts stub attempts. Barrier observations can prove actual A/B overlap; Ready output cannot. C waits for both validated/integrated predecessors. Start is not completion or inference success. |
| AC2: three independent roots and join, limit 2 | Hold two prepared/running stub attempts at barriers; third remains unprepared/unspawned. Finish/join one, observe third admitted; peak union occupancy <=2. Observe failures before spawn and after terminal commit/before worker return. No sleeps as concurrency proof. |
| AC2/AC3: dependencies and controls | Preserve DAG edges, Stage order, joins, gates, validators, effects, isolated writers and integration. Conflicting effects serialize or refuse under existing contracts. A free slot cannot bypass them; supported independent work can progress after capacity returns. |
| AC3: invalid declarations | Bounds (zero, negative, over schema bound/agent count), sequential 2, cycles, dangling refs and invalid owners remain refused. Compare store and spawn counts: pre-start refusal creates no Execution/process. Valid limit with unsupported validator/missing context remains refused for its own reason. |
| AC3: sequential 1 and DAG 1 | Mapping/start and existing controls remain intact; no artificial DAG edges; peak admitted occupancy <=1. |
| AC2/AC3: codec/digest | Frozen pre-change full canonical fixtures (single, independent DAG and existing multi-active history): decode/encode and clone preserve Workflow/digest and permit fenced progress without snapshot edit. Explicit 1/2 round-trip; invalid wire values/shape/digest fail; explicit limit tampering fails. Current old-reader refusal of new bytes is documented, not migration. |
| AC2/AC3: restart/resume | Persist explicit 2, restart same store and retain ceiling; absent limit uses fallback 1 without materializing it. Old multi-active records remain readable; orphaned prepared/running become unknown; no spawn/replay before permitted resolution. Stale fence/CAS and concurrent admission cannot exceed capacity. Include retry path. |
| AC2/AC3: cancellation/failure/unknown/integration | No new admission under existing stop conditions; peers drain with honest results; unintegrated/unknown results never unblock join merely because a slot returned. Existing retry/authority and integration/delivery contracts remain intact. |
| AC4 | PR/Issue Evidence records command, exit, fixture/SHA, observed counts/results and PASS/FAIL/SKIP/NOT_RUN/UNVERIFIED separately. No behavioral PASS from source review or documentation validation. |

## Proportional choices, risks and non-goals

Technical local choices: additive presence-preserving snapshot extension over a
format bump/rewrite; runtime fallback 1 for absent records over inferring a lost
value or unbounded new admission; enforcement within the existing fenced
aggregate/engine over a parallel coordinator. These meet the authorized behavior
with no new owner. Costs: strict older readers cannot read new records; old
records may progress more slowly; every constructor/clone/codec/admission path
must preserve the distinction. Future deterministic tests, not text inspection,
resolve these risks. Serialization/default mistakes and retry/stale-ready races
are the principal regression risks.

Rejected alternatives: guard removal alone loses the ceiling; synthetic edges
change DAG intent; historical graph reuse contradicts ADR-0037; format bump or
rewriting old snapshots breaks authorized compatibility. New public policy,
mechanism, infrastructure or ADR is not needed for the bounded path described.
A material incompatibility or ownership change is HUMAN_DECISION_REQUIRED;
there is no authorization to draft/adopt an ADR as a workaround.

Non-goals: no production/test edits in SPECIFY, PLAN/TASKS/IMPLEMENT, immutable
ubuntu-t12 r1 edit, CLI flag, real models/inference, T12 battery 03, external
product effects, launch-policy changes, merge/release/deploy, ready conversion,
Issue closure/acceptance or Linear/Cursor action. No optional deferred finding.

## Documentation Evidence and handoff

Repository/relative-link, whitespace and staged sensitive-file checks are
recorded with their actual commands and exits in the Issue completion receipt.
They validate documents/governance only. Product behavior and future compatibility
tests remain NOT_RUN/UNVERIFIED; Gitleaks availability is reported separately.
Review applies Axiom and Ponytail discipline to the source-backed amendment;
no accepted ADR contradiction was identified in this bounded specification.

Next gate: qualification of this amendment and compatibility oracles; CLARIFY
only for a material unresolved question, otherwise a proportional PLAN after
orchestration qualification. This attempt does not run that stage. Operational
checkpoint is AWAITING_EXECUTION_REPORT until the orchestrator consumes the
published receipt; no continuation integration or dispatch is assumed.
