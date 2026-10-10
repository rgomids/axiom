# Specification 007 — Configurable Project workflows

## Status and delivery boundary

Accepted on 2026-10-09 by the Axiom maintainer, with the exact reviewed
Specification revision and HD-001–HD-004 approval recorded in
[Issue #271](https://github.com/rgomids/axiom/issues/271#issuecomment-6074127314).
The document originated on 2026-10-08 under
[#271](https://github.com/rgomids/axiom/issues/271) and
[Epic #15](https://github.com/rgomids/axiom/issues/15).
[ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md)
is also accepted by that explicit human decision. This approval covers the
versioned contract and reviewed decision options, not implementation, Runtime
dispatch, Provider effects, merge, publication or MVP acceptance. Consumers
still require bounded implementation authority and their own Evidence; passing
fixture checks establishes document consistency only.

Audit baseline: `c7260797aad7865419f555eea94a2eef7ae78285`; authoring inspection:
`73dce6df0ab590b37df0e5a19b1483737bf1ca1e` (main). Existing
`internal/workflow/service.go` fixes `mvp-v1-sequential`; Project schema 3 has no
workflow selection. `internal/executiongraph/planner.go`, `graph.go`,
`scheduler.go`, `integration.go`, `internal/coordination/coordination.go` and
`internal/graphapplication/runtime_policy.go` are reusable foundations, not proof
of the missing public journey. Historical Specification 004 T01–T40 and S8/S9
Evidence keep their identities, scope and approval history.

## Intent, limits and constitution check

A developer operating a locally installed Project through CLI, Codex or Claude
needs to author/select a workflow and execute a Work Item against an inspectable
revision, even after the Project changes. A bounded ordered stage list with a
finite per-stage agent DAG is sufficient. There are no stage loops, arbitrary
expressions/scripts, dynamic spawning, distributed scheduling, UI, new control
plane, API-billing prerequisite, automatic merge/release/deploy or #241 work.
Specification 004's metadata-policy exclusion of a generic workflow engine
remains valid: this new contract consumes its fixed lifecycle, gates and graph
primitives, and does not replace metadata policy with user-defined labels.

Constitution I–VII: intent, requirements, deterministic examples, human gates,
alternatives and bounded evidence precede implementation; no new dependency or
external service is adopted. Constitution VIII: Project identity and Repository
target remain separate. Runtime != Model, Role != Model, Execution != Agent,
Skill != workflow truth, Evidence != raw chat. ADR-0003 owns the local control
plane; ADR-0004 separates portable intent from local state; ADR-0005/0006/0007
continue to own confinement, artifacts and protected publication/recovery.

## Requirements and unique implementation owners

Each requirement is normative only after approval. A follow-up owns the stated
slice; related consumers do not duplicate that responsibility.

| ID | Required behavior | Primary owner | Acceptance |
|---|---|---|---|
| WF-001 | Portable definition, strict schema and explicit Project selection | #273 | AC-001 |
| WF-002 | Immutable revision identity, safe edit and reference-aware retention | #273 | AC-002 |
| WF-003 | Atomic local Execution binding and R1/R2 isolation | #274 | AC-003 |
| WF-004 | Legacy sequential read/resume, fail-closed new formats | #274 | AC-004 |
| WF-005 | Complete bounded stage/agent contracts and default SDD | #273 | AC-005 |
| WF-006 | Stage criteria, validators, human facts and lifecycle projection | #274 | AC-006 |
| WF-007 | Deterministic stage compilation, single-agent and independent agents | #275 | AC-007 |
| WF-008 | Per-agent policy/Profile/effort resolution and context binding | #275 | AC-008 |
| WF-009 | Installed CLI/skill dispatch, coordination, status and recovery | #276 | AC-009 |
| WF-010 | Exact local-login/auth readiness and early real probe | #272 | AC-010 |
| WF-011 | Delivery packet/publication and explicit reject/rework/accept | #277 | AC-011 |
| WF-012 | Complete current-head operational evidence and decision packet | #278 | AC-012 |
| WF-013 | Feature command metadata and canonical human/JSON rendering | #133 / #232 (help / rendering respectively) | AC-013 |
| WF-014 | Existing Project resource lifecycle acceptance stays independent | #230 | AC-014 |

## Portable schema and selection (WF-001/002/005)

[workflow.schema.json](workflow.schema.json) is the proposed JSON Schema Draft
2020-12 boundary. [Default](examples/default-sdd-r1.json),
[custom R1](examples/custom-r1.json), and [custom R2](examples/custom-r2.json)
are complete versioned fixtures. No example is installed by this PR.

[Boundary examples](boundary-examples.json) include Project manifest **fragments**,
immutable binding fixtures and the acyclic Issue dependency map. They are not
complete Project manifests or records produced by the product. Placeholder IDs
and observation digests carry no authority or operational Evidence.

Definition identity is `(workflowId, revision, digest)`. `workflowId` is a
Project-scoped logical key; `revision` is a positive integer with a single
content assignment; digest is lowercase SHA-256 of validated content. Copying
identical content into another Project does not copy Execution identities or
authority. `schemaVersion` is codec compatibility, independent of revision.
Human-readable names do not establish identity.

Portable `workflows/index.json` has exactly `{schemaVersion: 1, revisions: []}`;
each entry is `{workflowId, revision, digest, state}` with `state` `published` or
`retired`. Entries are unique by `(workflowId, revision)` and sorted by that pair;
the index retains retired identity/digest assignments, so removal cannot allow
revision reuse with different content. Bound: 1024 entries / 1 MiB; exhaustion
blocks before publication, never prunes old assignments. The next revision is
greater than every indexed revision of that workflow; gaps are valid. Import
rejects conflicting assignments before effects. Built-ins have a release-owned
index and cannot be retired by Project operations.

Proposed Project schema 4 adds `workflowSelection` with exactly `workflowId`,
`revision`, `digest` and `source` (`builtin` or `project`). Project definitions
are explicit companion files in the portable Project working copy,
`workflows/<workflowId>/<revision>.json`; they belong to the Project, not one
associated Repository or the user-global skill directory. Selection is an exact
reference, never `latest`, CWD inference or an absolute path. Definition paths
are derived only from validated keys; file links/replacements obey ADR-0005.
The built-in source is the release-bundled `default-sdd` revision 1 fixture;
its selection is explicit and its full content is copied into new local bindings.
Selecting it requires no custom authoring or mandatory multi-agent setup.

Schema 1/2/3 Projects remain readable without rewriting or implicit selection.
Legacy resume uses its legacy record; a new configurable start on an unselected
Project returns `workflow_selection_required`. An explicit configure/select
preview may upgrade to schema 4, preserving all prior fields, presence semantics
and references. #273 MUST extend the local inventory and compatibility corpus,
including schema 1/2/3 writers, before release. A downgrade to a reader that does
not support schema 4 fails safely; it must not delete or reinterpret new data.

Authoring parses strict UTF-8 JSON: reject duplicate keys, unknown properties,
malformed/unsupported schema, unknown enum values and out-of-range integers.
No floats or null values appear in this schema. All normative fields are required;
empty lists explicitly mean none. No hidden defaults are injected by consumers.
IDs match `^[a-z][a-z0-9-]{0,63}$`. Limits: 1 MiB definition/canonical snapshot,
1–32 stages, 1–32 agents per stage, 1–32 maximum concurrent agents, 1–10 attempts,
1–86400 seconds timeout, 1–16384 UTF-8 bytes per instruction/purpose/criterion,
at most 64 entries per other list. Byte limits supplement Schema character limits.
Larger instructions/context use typed digest-bound artifact references.

Digest calculation: validate first; canonicalize the **whole definition**, with
no excluded field, using RFC 8785 JCS; SHA-256 canonical UTF-8 bytes, without BOM
or trailing newline. Definition has no self-digest. Preserve Unicode without
normalization, array order and instruction whitespace. Object key/indentation
changes do not change digest; changing content, revision, order, criterion or
instructions does. The included checker covers the fixtures' ASCII/integer
subset; #273 must use full JCS conformance vectors, not Python JSON formatting
as a general implementation. Existing graph/proposal digests retain their own
codec; this choice does not reinterpret historical digest algorithms.

An edit publishes a new complete revision; existing revision overwrite is denied.
Same identity/content replay is a no-op; same identity/different digest is a
conflict. Preview binds expected Project revision, expected current selection,
definition content/digest and complete local effect list. Validate the definition
publication before selection. Selection can reference only a confirmed readable
and indexed published revision. The order is content publication, index append,
then selection, each with expected-revision checks and read-back. Unindexed
published content is not selectable; index/content interruption needs explicit
recovery, not inferred success. If selection fails after revision publication, report `partial` with
the confirmed revision and actionable next step; never claim rollback. ADR-0007
owns interruption, fencing, atomic record publication and recovery.

Removal is explicit preview/apply only. Selected or locally referenced revisions
are retained, including inactive Execution, rejection, recovery and Evidence
references. Built-ins are read-only. Removing an unreferenced portable definition
retires its index entry before content removal and cannot invalidate retained
local snapshots; failed content removal leaves a truthful retired entry and
confirmed effects. No background pruning or inferred
reference repair. Missing/ambiguous indexes block cleanup. Portable import
revalidates content; it never imports machine-local authority or Execution state.

## Stage and agent semantics (WF-005/006)

Stages run in definition array order, exactly one active stage. Each declares:
stable `id`, `purpose`, `instructions`, typed `inputs`, typed `outputs`, explicit
`completionCriteria`, `validators`, `humanGates`, `failurePolicy`, `phase`,
`checkpoint`, `mode`, `concurrency` and complete `agents`. Within a revision,
stage IDs and agent IDs are unique at their scopes. Rename/change creates a new
revision, never retargets retained history.

Inputs name `work-item`, `project-context`, `artifact` or `stage-output` sources;
stage-output names exactly `<prior-stage-id>/<output-id>`. Inputs from current,
later or missing stages fail validation. Project context keys are logical
registered document/business-context references; required missing/unreadable
sources block planning. Outputs declare ID, kind (`artifact`, `evidence`,
`decision`, `result`) and requiredness. They are references to bounded typed
artifacts, never arbitrary host paths, credential content, chat or hidden reasoning.
Each criterion references a declared output and a declared validator. Required
outputs and **all** criteria must pass; criteria are not agent self-certification.
Validators are domain-owned versioned `artifact-schema`, `evidence-check` or
`human-review` adapters. Their configuration is a logical policy reference;
executables/argv cannot be supplied as definition strings. Actual repository
test commands are reviewable validator policy artifacts under separate authority.
Schema validation alone does not prove a validator is available or has run.

Agent `role` and `responsibilities` describe work, separate from Runtime/Profile.
Each names inputs/output responsibilities, dependencies, capability requirements,
`complexity` (`low`, `medium`, `high`), Runtime constraints, Profile reference (or `policy-default`), effort,
timeout/attempt limits and allowed effect **ceilings**. Every required stage
output must have an accountable agent. Agent input/output names resolve to that
stage's declarations. Each dependency also supplies its predecessor's validated
output references/digests as explicit compiled child inputs; the integrator cannot
consume raw chat or guessed files instead. No nested agents or implied delegation.
Dependencies use
same-stage IDs, with no self-edge, dangling edge or cycle. `sequential` requires
concurrency 1 and every non-first agent depending on its immediate predecessor;
`dag` allows independent nodes. Stage limits and the approved Plan may narrow
effects/attempts/concurrency; they cannot exceed definition or operator ceilings.

`failurePolicy` fixes `onFailure: block`, `onUnknown: reconcile`, and
`retry: explicit`. Fail/timeout blocks dependent work; cancellation prevents new
dispatch, requests bounded stopping and preserves confirmed/unknown effects.
Retry is a new attempt under the same child only after explicit authority,
reconciled effects and available attempt capacity. Exhaustion requires a new
bounded proposal/authority, not counter reset. Unsupported pause/cancel must be
reported as unsupported, never observed success. Resume is not process replay.

Gate kinds reuse `planning_authority`, `implementation_authority`,
`review_started`, `human_acceptance` with `before` or `after` timing. Human facts
bind the exact Execution/stage/record revision, workflow digest, applicable
plan/delivery digest, actor and Evidence reference. `planning_authority` requires
Specification/decision refs; `implementation_authority` Plan/Tasks refs;
`review_started` artifact/Evidence/PR ref; `human_acceptance` validated delivery
Evidence ref. Reference existence alone cannot grant authority. Protected domain
prerequisites cannot be weakened by omitting gates in a custom workflow. Only
human-authenticated application operations commit such facts; prompts and typed
agent messages cannot. Reject inappropriate timing/phase combinations at authoring.

Exactly one planning-authority gate precedes the first planning stage, one
implementation-authority gate precedes the first implementation stage, and one
review-started gate precedes the first review stage. Exactly one acceptance gate
follows the final completion stage. A custom workflow cannot delay these gates
until later stages or duplicate them to reinterpret authority.

## Fixed lifecycle projection (WF-006)

`phase` values are `intake`, `specification`, `planning`, `implementation`,
`review`, `completion`, monotonically ordered with at least one of each. A phase's
last stage has `checkpoint: true`; earlier stages of it have false. Required
outputs/validators at each checkpoint establish its milestone. This bounded
profile allows different names/counts/agents without inventing a Provider lifecycle.

| Canonical local facts/frontier | Existing lifecycle value |
|---|---|
| Intake is active | `intake` |
| Specification is active | `specifying` |
| Specification checkpoint passed, no planning authority | `specified` |
| Planning authority recorded, planning not finished | `planning` |
| Planning checkpoint passed, no implementation authority | `planned` |
| Implementation authority recorded, implementation not finished | `implementing` |
| Implementation checkpoint passed, no review-started fact | `implemented` |
| Review started, remaining review/evidence/reconciliation incomplete | `reviewing` |
| Review checkpoint and completion validation passed, acceptance pending | `reviewed` |
| Exact final packet explicitly human-accepted, terminal success committed | `accepted` |

All preceding milestones are required in each row. When a checkpoint just passed
and the next stage is active, its required before gate keeps the projection at
the previous milestone until the gate is satisfied. Completion preparation before
its validator passes remains `reviewing`. `blocked`, `needs_decision`,
`needs_approval`, `recovery_required` remain auxiliary conditions, not stages.
Within an Execution, milestones never rewind. Reject/rework uses a new linked
Execution as specified below. For a Work Item with multiple runs, use its explicit
active Execution link; after it terminates use the last explicitly linked run,
never timestamp guessing. Ambiguous linkage returns recovery required.

Only these ten existing lifecycle values may reach metadata-policy projection.
Custom stage names appear in bounded status/transition details, never generate
`axiom:stage:<custom-id>` labels. Existing legacy stage-label behavior is preserved
for legacy records. Existing metadata policy chooses the approved lifecycle labels;
this proposal creates no labels or new Provider states. Projection key binds
Execution ID + committed local revision; projection needs separate exact effect
authority. Provider success/failure cannot change canonical milestones. Partial
projection retains confirmed local truth. Provider labels/comments never recover
missing local state or establish human acceptance.

## Binding, persistence and compatibility (WF-003/004)

New configurable Execution record format 2 adds an immutable `WorkflowBinding`
and stage execution ledger to the existing local authority. Do not add configurable
fields to format 1 records. `WorkflowBinding` contains Project ID, workflow ID,
schema/revision/digest, source, canonical snapshot reference/digest, selection
observation/Project revision and binding timestamp. The snapshot and binding are
published/read back before an Execution becomes dispatchable. Interrupted multi-
record publication cannot produce an active binding from an unconfirmed snapshot;
use ADR-0007 recovery, exposing confirmed effects and `recovery_required`.

The stage ledger records stage ID, ordered stage ordinal, state revision, criteria
and validator result refs, human facts, transition provenance, optional graph
parent/revision, child/attempt refs and confirmed effects. Context observations
contain logical source + digest + bounded local artifact ref; local paths,
credentials and process details never enter the portable definition. Binding is
unchanged for retries, resume, configuration editing or Profile unavailability.

At start, resolve effective Project with #233 precedence and exact #137 target;
read selected definition and context; check operation-scoped readiness. Preview
binds resolved Project/source revision, Repository key/local binding revision,
Work Item Provider/resource/external ID, workflow selection/digest, context
digests, policy/Profile/configuration and capability observation revisions, Plan,
effects and expiry. Drift in any bound observation denies apply before effects.
Allocate opaque identity, confirm snapshot/binding, then allow stage planning.
Starting does not itself grant dispatch or Provider mutation authority.

Resume reads the retained canonical snapshot, never current Project selection or
a fresh built-in. For Execution A on R1, selection/edit to R2 has no effect on A;
new B uses R2 only after a new start preview. Missing snapshot, digest mismatch,
unsupported schema, corrupt ledger, uncertain publication or lineage mismatch
returns `recovery_required` without dispatch. A readable portable copy cannot
silently repair a missing snapshot; explicit recovery may restore **identical**
verified bytes under a preview bound to the damaged state, preserving history.
Runtime readiness is reobserved; policy revocation blocks dispatch. Newly approved
resolution is a graph/attempt proposal under the same definition, never fallback.

| Stored case | Required read/resume behavior |
|---|---|
| Valid format 1 / `mvp-v1-sequential` | Existing codec, stages, gates, lifecycle and resume unchanged; report legacy kind; no snapshot, parent, stage ledger or selection invented |
| Preserved POC history | Existing inert preservation; no opt-in #175 migration added |
| Supported format 2 with valid snapshot | Read/status offline; resume exact pinned contract after readiness/authority validation |
| Unsupported future version | Explicit unsupported/recovery result; no overwrite or downgrade |
| Missing/corrupt/tampered format 2 dependency | Fail closed; read-only inspection of known facts; no default substitution |

Retention includes terminal executions and their decisions, snapshots, stage
context, graph/attempts and artifacts while referenced for review/resume/recovery/
Evidence. Cleanup is reference-aware, explicitly previewed and separately
authorized; unknown reference sets block it. No automatic TTL. Format 2's
compatibility window is all schema 1 definition revisions written by the first
configurable release onward; format 1 remains supported across that window.
Future changes need an explicit forward path and frozen corpus; #274 adds
inventory/writer/upgrade tests under the persisted-state quality policy.

## Stage planning and Runtime boundary (WF-007/008/010)

For each multi-agent stage, #275 compiles stage contract + bounded approved Plan
to existing `ApprovedPlan`/`WorkUnit` → `Proposal` → child envelopes. No second
scheduler. The stage work Plan is an explicitly approved bounded execution plan,
not proof of engineering implementation authority; early analysis stages may have
read-only Plans while implementation remains gated on engineering Plan/Tasks.
Node keys correlate `<stage-id>:<agent-id>` (valid existing graph tokens); opaque Execution IDs are
allocated separately. Preserve instruction/context/artifact digests in an
Axiom-owned bounded stage-input artifact, referenced by each envelope. Extend
versioned boundary DTOs for correlation; do not put rules in prompts.
Logical stage-output source syntax is resolved into validated graph artifact/input
tokens; do not pass slash-containing source names into graph token fields.
Role, complexity, capabilities, dependencies, I/O, timeout and attempt controls map
losslessly to existing WorkUnit fields. Effect targets and Repository paths come
from the bounded approved Plan, not strings inferred by an agent. The only allowed
effect ceiling kinds are the existing `read`, `repository-write`, `process`,
`artifact-publish`, `integration`; any compiled effect must be both declared and
explicitly authorized. An integration flag does not grant an integration effect.

Exactly one declared agent in a `dag` stage is integration owner and validator
owner and depends transitively on every other required agent. Disconnected or
optional unwaived outcomes cannot yield parent success. Effects are a child
subset of parent authority, checked with existing graph rules. Dependency-free
nodes overlap only if concurrency permits, their effects do not conflict and
repository writers have isolated workspaces; otherwise serialize or block before
dispatch. Integration applies only reviewed authorized changes and combined
validators; missing/conflicting results block roll-up. Stage success requires
required children and integration success plus validators and applicable gates.

A `sequential` stage with **one agent** uses the existing sequential executor
extended for the pinned stage contract, with no graph required. Its sole agent
owns validation and integration of its own output; no second LLM/process/child is
fabricated. The stage ledger explicitly says `executionKind: single`, has no
graph parent and still binds its bounded attempt/result. Multi-agent sequential
stages use the graph with a predecessor chain and final integration owner.
#274 owns ledger admission; #275 owns compilation/one-agent adapter contract;
#276 owns dispatch composition. New graph correlation must not synthesize
lineage into legacy records. Graph terminal result alone does not accept the
Work Item or complete the whole workflow.

Per-agent Runtime selection intersects stage constraints, Project allowlist and
operator-installed available capability-proven Profiles. `policy-default` means
the existing deterministic Project resolver; tie/no-match/disabled/unavailable
returns a blocker, never invented ranking. Initiating Runtime has no preference
over child choice. Concrete model IDs resolve from machine-local Profile bindings;
portable workflow references Profile keys, never vendor account credentials.

Effort contract: `{mode: "runtime-default", value: "default"}` sends no override;
`{mode: "explicit", value: <token>}` requires capability-proven support for the
resolved Runtime version + model + invocation mode. Tokens are opaque adapter
values, not portable rankings. No conversion between Codex and Claude levels.
The fixtures use `medium`/`high` as **conditional requests**, not proof that any
installed model supports them. Unknown/unsupported effort returns
`unsupported_effort` before dispatch, with requested/observed values and next
action. #275 records exact argv/config mapping and adapter tests; #272 validates
effective local CLI authentication without hidden API prerequisites. Observation
drift demands a fresh authorized proposal. Default is usable without selecting a
concrete model/effort; missing local Runtime setup remains an honest readiness gap.

## Public operations and boundary DTOs (WF-009/011/013)

These are **proposed interfaces**, not commands available at this revision.
Keep authoring under `axiom project workflow <operation>` and
~~thin `axiom-project` operations `workflow.<operation>`.~~ Execution uses
existing `axiom workflow` command family and thin `axiom-work-item`;
~~do not create an `axiom-workflow` skill.~~
~~Project ownership gives one discoverable configure/show entrypoint and reuses
preview/apply. A dedicated top-level workflow management surface would
separate authoring conceptually but add selector/skill overlap; revisit only
if independently managed cross-Project definitions become approved.~~

> **Conversational skill-surface supersession (accepted 2026-10-10):** the
> historical stricken text is preserved for traceability. The approved
> [Specification 007 amendment](amendment-303-workflow-skill.md) and
> [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md)
> replace **only** the skill-routing restriction: `axiom-workflow` configures
> workflow definitions, `axiom-project` selects the active Project revision,
> and `axiom-work-item` owns Work Item-bound execution/plan. The original
> Project-owned CLI authoring and `axiom workflow *` commands, authority,
> result contracts and immutable binding remain unchanged. The maintainer's
> formal exact-revision decision is recorded in
> [Issue #303](https://github.com/rgomids/axiom/issues/303#issuecomment-6099596104).
> This is contract approval, **not** implementation/merge/release authority.

All requests use explicit/effectively resolved Project, no CWD/chat fallback.
Read-only operations have no authority argument. Mutating operations have preview
and apply forms; apply requires `expectedRevision`, `previewDigest`, an exact
authority reference and explicit operator confirmation, through existing
application authority machinery. IDs below are semantic JSON DTO field names;
existing adapters may retain aliases but must map losslessly, not implement a
second public result protocol. No flag spelling is claimed installed here.

| Operation | Required request data | Response data / primary owner |
|---|---|---|
| `project workflow list/show` | Project; show exact WorkflowRef | references/definition/schema/readiness; #273 |
| `project workflow validate` | Project, complete definition | digest, ordered diagnostics, unresolved runtime prerequisites; #273 |
| `project workflow create/edit` | Project, definition, expected Project revision; edit prior WorkflowRef | preview/effects; confirmed new immutable WorkflowRef; #273 |
| `project workflow select/remove` | Project, exact WorkflowRef, expected selection/revision | preview/reference blockers; confirmed selection/removal; #273 |
| `workflow start` | TargetRef, expected Project revision/selection, context observations | start preview; ExecutionRef + WorkflowBinding; #274 |
| `workflow status/resume` | ExecutionRef; resume expected ledger revision | pinned stage, criteria, validators, gates, conditions, next action; #274 |
| `workflow stage plan` | ExecutionRef, stage ID, expected revision; PlanRef for graph path, bounded scope for single path | deterministic proposal/digest, resolved controls, blockers; #275 |
| `workflow stage run/retry/cancel` | ExecutionRef, stage/graph/attempt refs, reviewed proposal, expected revisions | confirmed dispatch/attempt/stop observations; #276 |
| `workflow coordination list/publish` | ExecutionRef, graph/child/attempt correlation; publish typed bounded record | existing typed record/result refs; #276 |
| `workflow delivery show/publish` | ExecutionRef, packet digest; publish exact repository/base/head/remote effects | packet/readiness; confirmed PR/artifact ref or uncertain effects; #277 |
| `workflow decision` | ExecutionRef, packet digest, decision `accept`/`reject`, reason/Evidence/actor | committed fact or rejection and rework proposal ref; #277 |
| `workflow rework` | rejected ExecutionRef/decision, selected exact WorkflowRef, bounded new scope/Plan | preview; new ExecutionRef + immutable predecessor linkage; #277 |

`WorkflowRef = {workflowId, revision, digest, source}`.
`TargetRef = {projectId, repositoryKey, provider, resource, externalId}`.
`ExecutionRef = {executionId, revision}`.
`PlanRef = {id, revision, digest}`.
`Observation = {kind, id, revision, digest}`.
`Preview = {digest, expectedRevision, observations[], expiresAt, effects[]}`.
`StagePlan = {executionId, workflowRef, stageId, stageInputRef, graphProposalRef?,
executionKind, resolutions[], validatorRefs[], gateRefs[], blockers[]}`.
`Resolution = {agentId, runtimeId, modelProfileId, configurationRevision,
observationRevision, requestedEffort, effectiveEffort, capabilityEvidenceRef}`.
These fields are required except graphProposalRef for a single-agent path;
resolutions exist only when successful. No unresolved placeholder is dispatchable.

Canonical result retains existing completion `status`, provenance-marked
`result`, `references`, `next`, `details`, `provenance`; application payload adds
`category`, `confirmedEffects`, `workflowRef`, `executionRef`, `stageId`,
`conditions`, `preview`/`plan`/`packet` as applicable. Absent non-applicable
payloads are omitted. Never add `unknown` or `blocked` to completion.Status:
these are conditions/attempt outcomes/categories. Status success on a read or
preview means that operation succeeded, not workflow success. `details` points
to bounded machine-local diagnostics; both Runtime skills call the same service.
#133 owns help rendering metadata, #232 human rendering; each feature registers
its own commands and cases. JSON remains complete and canonical.

| Refusal/failure case | Completion status | Category / preserved truth |
|---|---|---|
| Duplicate/unknown/schema/cycle/dangling/phase/limit failure | `validation_failure` | `invalid_workflow`; path diagnostics; no effects |
| Required selection/input/validator/gate absent | `validation_failure` | `workflow_selection_required` / `stage_prerequisite_missing`; needs approval where applicable |
| Preview/target/policy/context/graph revision drift or missing authority | `denied_authority` | `stale_preview` / `authority_denied`; no new effects |
| No-match/unsupported effort | `validation_failure` | `runtime_unresolvable` / `unsupported_effort`; no dispatch |
| Proven safe transient pre-effect unavailable service | `retryable_failure` | `temporarily_unavailable`; explicit next action |
| Interrupted process with confirmed no ambiguous effects | `interrupted` | `interrupted`; retained binding/attempt |
| Lost acknowledgement/unknown effect/corrupt snapshot | `failure` | `recovery_required`; unknown retained; no blind retry |
| Confirmed primary effect + secondary local/projection failure | `partial` | `confirmed_effect_reconciliation_required`; references and next required |
| Valid preview/read/no-op/confirmed operation | `success` | operation-specific category; explicit scope |

## Delivery, rejection and parent completion (WF-011)

Packet binds workflow, Work Item, parent/children/attempts, integration tree/diff,
artifact and validation refs, review findings, decisions, blockers and provenance.
Readiness requires every required stage/checkpoint, validator and review result,
no unresolved Blocker/Major unless exact human scope waiver is recorded. Packet
creation does not publish. PR publication preview binds repository, base/head
commit, remote target, packet digest, Work Item and exact effects; authority for
local editing never grants push/PR/merge/release. Ambiguous publication is
reconciled by exact remote identity before retry, never duplicated blindly.

Agent/stage/graph success is technical completion. Workflow terminal success and
`accepted` require exact final packet acceptance by the human; green checks or
Provider closure do not satisfy it. Rejection records human actor, reason,
packet/Execution/revision and Evidence and preserves the completed work and
confirmed effects. A rejected run is terminal with delivery decision `rejected`,
not success/accepted. It remains inspectable; no attempts or transitions rewind.

Rework creates a **new Execution**, linked by `reworkOf` to the rejected run and
decision. Preview explicitly chooses retained R1 or a newly selected R2 and
bounded scope/Plan; no automatic workflow switch, Runtime fallback or reused
authority. Prior artifacts may be new inputs only when digest-bound and validated;
prior authority/acceptance does not carry over. The new run follows all required
phase/checkpoint gates; no arbitrary jump into implementation. The explicit active
Work Item link changes only on confirmed new binding. Uncertain linkage blocks
start. #277 owns decision/linking use cases; #274 owns immutable binding admission.

## Architectural decisions — explicit maintainer acceptance

| Decision | Options and recommendation | Trade-offs | Human owner / affected Issues |
|---|---|---|---|
| HD-001 / ADR-0020 | Portable Project companion definitions + schema 4 selection + local snapshots (accepted); inline manifest; global mutable catalog; digest pointer only | Companion portability/reference upkeep vs inline size; snapshots cost storage but allow offline resume; global catalog adds authority; pointer only loses retained semantics | Axiom maintainer; #273/#274 and consumers #275–#278 |
| HD-002 / public surface | ~~Extend Project authoring + Work Item execution (accepted)~~ **as the exclusive skill surface** (historical choice superseded by [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md) and the [accepted amendment](amendment-303-workflow-skill.md)); dedicated workflow skill/CLI tree was the original alternative. **CLI and domain ownership remain as accepted.** | Small existing entrypoints vs longer Project command tree; dedicated skill duplicates discovery/selection (historical assessment; only the configuration *skill*, not a new CLI tree, is now adopted). | Axiom maintainer approving this Specification; #273/#133/#232/#276. Follow-up skill-surface decision formally accepted 2026-10-10 in [Issue #303](https://github.com/rgomids/axiom/issues/303#issuecomment-6099596104). |
| HD-003 / rework | New linked Execution (accepted); rewind old stages; mutate completed attempts | More records/repeated gates; preserves immutable provenance vs ambiguous rewind and lost evidence | Axiom maintainer approving ADR-0020 and this Specification; #274/#277/#278 |
| HD-004 / canonical content | JCS + SHA-256 (accepted); exact source bytes; implementation-specific marshaler | Conformance work vs formatting-sensitive edits or cross-client drift; does not change legacy graph hashes | Axiom maintainer approving ADR-0020; #273/#274/#275 |

Record decision date, actor, exact artifact revision, accepted option and any
conditions in a versioned decision record or linked explicit human review.
The approval ledger is **Accepted, 2026-10-09**: HD-001 (Project companion
workflow definitions and snapshots), HD-002 (Project authoring / Work Item
execution surface), HD-003 (new linked Execution for rework), HD-004 (JCS + SHA-256),
all without added conditions, under the explicit maintainer decision in
[Issue #271](https://github.com/rgomids/axiom/issues/271#issuecomment-6074127314).
Reviewed source revision: `998e763cf82c4cb443d0d4d0051c6e82bcdf8d72`;
Specification blob: `b68072e9909793f24786c343e4b9d3b83064b217`;
ADR-0020 blob: `3f913ed027ec90546ca96d436d538335590e4279`.
Future changes to these choices require a separate human decision. Completion of
the documentation does not assert implementation or MVP acceptance.

## Epic capability and acceptance coverage

This maps every capability row in Epic #15 section C and K1–K12; proven scope is
kept separate from remaining work. Each remaining slice has exactly one primary
Issue, with final operational proof owned by #278.

| Epic capability | Existing contract/evidence boundary | Remaining primary owner / requirement |
|---|---|---|
| Install/first-run/discovery, K1 | `internal/install`, `runtimebootstrap`, 004 S2/S9; no new host run | #278 / WF-012; #265 retains its separate Windows check |
| Project identity/repos/bindings/Provider/Integration, K2 | 002, #231/#230 implementation | #230 / WF-014 acceptance; #278 journey |
| Business context/glossary/docs/technology summaries, K2 | 002 schema 3, `internal/project/context_test.go` | #275 / WF-008 consumption; #278 journey |
| Runtime/Profile policy, K2/K4 | 002 policy v2, #140, `internal/runtimeprofile` | #275 / WF-008 stage resolution |
| Default/custom workflow and full stage schema, K3/K4 | Not delivered by fixed `mvp-v1-sequential` | #273 / WF-001/005 |
| Safe edits/immutable revisions, K3/K5 | Existing Project preview/publication primitives only | #273 / WF-002 |
| Work Item target/lifecycle/gates, K6 | #137/#138/#233, 004 workflow-gates and tests | #274 / WF-006 pinned integration |
| Execution R1/R2 preservation, K5 | ADR-0008 workflowVersion is not definition content | #274 / WF-003/004 |
| Stage roles/instructions/count/topology/effort, K4 | 004 S8 ApprovedPlan/WorkUnit only | #275 / WF-007/008; schema authoring #273 |
| Scheduling/isolation/integration, K8 | ADR-0009; graph scheduler/integration tests; historical T36 | #275 / WF-007 compilation; #276 dispatch |
| Structured coordination, K8 | Nine existing typed kinds; `internal/coordination`; historical exchanges | #276 / WF-009 public connection |
| Public CLI/skill launch from either Runtime, K6 | Fixed sequential skill; S9 runner is checkout-only | #276 / WF-009 installed path |
| Local subscription CLI invocation, K7 | Runtime adapters; login observations are not inference proof | #272 / WF-010; #276 smoke; #278 final proof |
| Progress/blockers/decisions/Evidence/provenance, K9 | Separate workflow/graph status and provenance | #274 / WF-006 ledger; #276 correlated progress |
| Packet/authorized PR/accept/reject/rework, K10/K11 | Integration/completion/human facts; metadata is not PR creation | #277 / WF-011 |
| Command discovery and results, K9 | Existing result contract and global help | #133 help, #232 rendering / WF-013 |
| Current-head expanded acceptance, K12 | Historical 004 S8/S9 remain revision-bound | #278 / WF-012; explicit #15 human decision |

## Acceptance matrix and reproducible validation

| ID | Required test/review evidence | Owner |
|---|---|---|
| AC-001 | Strict schema, complete selection preview, v1/v2/v3 preserved; new start refuses absence | #273 |
| AC-002 | Formatting-equivalent digest; meaningful edit changes digest; conflicting overwrite rejected; selected/referenced removal blocked | #273 |
| AC-003 | A pins R1; select R2; restart A still R1; B pins R2; drift refuses apply before effects | #274 |
| AC-004 | Real legacy writer corpus loads/resumes unchanged; no synthetic binding; corrupt/future format refuses dispatch | #274 |
| AC-005 | Default sequential stage list usable with one agent; custom Codex+Claude dependencies and single stage validated | #273 |
| AC-006 | All ten lifecycle rows; custom IDs produce no labels; missing output/validator/human fact blocks; Provider cannot advance state | #274 |
| AC-007 | Same contract/Plan/observations gives same proposal; independent nodes allowed; cycle/dangling/overlap/limit rejected; no mandatory extra LLM | #275 |
| AC-008 | Logical context digests bound; no-match/unavailable/unsupported effort fails; child envelopes preserve controls; no implicit fallback | #275 |
| AC-009 | CLI/Codex/Claude parity, restart lineage, nine coordination kinds, forged lineage and stale authority denied, unknown reconciled | #276 |
| AC-010 | Installed exact invocation/auth observations plus separately authorized bounded real probe; no login-only inference claim | #272 |
| AC-011 | One packet; publication preview/denial/replay/drift/unknown tested; reject creates linked new run preserving R1; accept exact packet only | #277 |
| AC-012 | Every Epic K row has current-head real/deterministic Evidence or blocking unmet result; human decision separate | #278 |
| AC-013 | Registered commands/results discoverable, human/JSON semantic parity and provenance; no feature-owned prompt semantics | #133 / #232 |
| AC-014 | Delivered lifecycle acceptance or exact recorded stability gate; no inference from Issue labels/merge | #230 |

#271 validation is document/reference/schema-example consistency, not the above
implementation evidence. Run `python docs/specifications/007-configurable-workflows/validate_examples.py`
and repository governance checks. The fixture checker validates the strict schema
subset used here, typed references, bounded topology, lifecycle checkpoint/gate
coverage, JCS-subset digests, R1/R2 and negative mutations. It does not call the
product, Runtime, Provider or network. #273 must run a full Draft 2020-12 validator
and JCS vectors; #274–#278 own their independent executable acceptance tests.
The dependency graph is #271+human decisions → #273 → (#274, #275) → #276 →
#277 → #278, with #272 and #230 stability joining #276, and #133/#232/#230
acceptance joining #278. No edge returns to #271; no #241 dependency is added.

## External references and bounded use

Primary references checked 2026-10-08; they inform this proposal, not architecture
adoption or installed-version support claims:

- [RFC 8785 JCS](https://www.rfc-editor.org/rfc/rfc8785.html): deterministic JSON
  bytes for digests; retain Unicode and distinguish content from formatting.
- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12): explicit
  versioned structural validation; DAG/authority still need semantic checks.
- [Temporal workflow definitions](https://docs.temporal.io/workflow-definition):
  changing a definition can affect execution determinism. Axiom uses retained
  local snapshots; no Temporal SDK, service or replay engine is adopted.
- [Open Workflow DSL](https://github.com/open-workflow-specification/specification/blob/main/dsl-reference.md):
  separate definition identity/version and explicit task dependencies. Arbitrary
  expressions, scripts, nested workflows and distributed features are out of scope.
- [Codex configuration](https://developers.openai.com/codex/config-reference/)
  and [Claude model configuration](https://code.claude.com/docs/en/model-config):
  effort depends on model/client/version. Exact adapter observations take priority
  over examples; no portable vendor ranking or silent effort coercion.
- [Codex authentication](https://developers.openai.com/codex/auth/) and
  [Claude authentication](https://code.claude.com/docs/en/authentication):
  local login and effective execution/billing path are distinct. #272 owns proof.
