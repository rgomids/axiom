# Plan and Tasks — #276 run and coordinate stage agents through CLI and Runtime skills

## Status

**Phase 1 (Plan), revision 2 — Proposed, pending human review.** This
document is the Plan for [#276](https://github.com/rgomids/axiom/issues/276) /
Linear AXM-8. It does not authorize Tasks, implementation, Runtime dispatch,
real inference, Provider effects, merge, release or human acceptance. Part II
(Tasks) stays empty until this Plan, its decisions (§12) and the proposed
contracts it depends on are approved.

| Item | Value |
|---|---|
| Work Item | [#276](https://github.com/rgomids/axiom/issues/276) / Linear AXM-8 (operational status lives in Linear) |
| Parent | [Epic #15](https://github.com/rgomids/axiom/issues/15); Spec 007 requirement **WF-009**, acceptance **AC-009** ([spec.md:62](spec.md), [spec.md:547](spec.md)) |
| Contract consumed | [Spec 007](spec.md) (Accepted 2026-10-09), [amendment #303](amendment-303-workflow-skill.md) + [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md) (Accepted 2026-10-10), [ADR-0003](../../decisions/0003-lingo-as-axiom-local-control-plane.md), [ADR-0004](../../decisions/0004-portable-project-manifest.md), [ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md), [ADR-0007](../../decisions/0007-local-publication-and-recovery-protocol.md), [ADR-0008](../../decisions/0008-minimal-machine-local-execution-record.md), [ADR-0009](../../decisions/0009-parent-child-execution-graph.md), [ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md) |
| Contracts proposed by this PR | [Spec 007 amendment #276](amendment-276-stage-dispatch.md) (**Proposed**) and [ADR-0024](../../decisions/0024-public-stage-dispatch-and-capability-proof.md) (**Proposed**). Neither is accepted; each needs its own human decision |
| Maintainer decisions incorporated | **D-1 resolved** ([PR #311 comment](https://github.com/rgomids/axiom/pull/311#issuecomment-6108112930)): capability proof for stage execution is a #276 deliverable. #306 directions D2/D3 ([PR #310 comment](https://github.com/rgomids/axiom/pull/310#issuecomment-6108025068)): vendor-managed authentication only; `effort` belongs to the local Model Profile |
| Satisfied predecessors | #274 (binding), #275 (stage compiler, v0.15.0), #303 (skill surface, v0.16.0), #272 (auth preflight), #230 (resource lifecycle), #271 (contracts) — implementation baseline only |
| Parallel work | [#306](https://github.com/rgomids/axiom/issues/306) / [PR #310](https://github.com/rgomids/axiom/pull/310) (local Runtime/Profile authoring, Profile-owned effort, `axiom-runtime` skill, ADR-0023) — boundary in §10 |
| Successors (not in scope) | #277 delivery/rework, #278 / AXM-12 R-2 native and R-3 human MVP acceptance |
| Code baseline | `main` `5d52327` (v0.16.0); Plan revision 1 at `0f5ba21` |
| Lifecycle | One branch (`feat/276-stage-agent-execution`) and one Draft PR (#311) for Plan → Tasks → Implementation, each separately approved |
| Review | Revision 1: independent adversarial review (§12.5). Revision 2: maintainer REQUEST CHANGES on PR #311 (D-1, CR-001, #310 reconciliation, ADR collision, metadata) |

Observed statements cite `file:line` on the baseline. Statements marked
*(inferred)* must be confirmed in the named Task. **PD-n** items need explicit
human approval (§12). **IG-n** items are integration gates owned outside
#276 (§10). Vendor interface facts come only from local `--version`/`--help`
output of the installed CLIs (Codex CLI `0.159.1`, Claude Code `2.1.296`,
observed 2026-10-11, no inference, no authentication) and must be
re-observed at implementation time.

---

# Part I — Plan

## 1. Scope and objective

### 1.1 Problem

Axiom already contains a tested parent/child Execution Graph, scheduler,
workspace isolation, integration, nine typed coordination record kinds and
their stores. None of it is reachable from the installed product: the only
caller of `internal/graphapplication` is the checkout-built, `//go:build
ignore` acceptance runner
([s9-graph-runner.go:1-6](../../../scripts/acceptance/s9-graph-runner.go)),
and `workflow stage plan` says its proposal is "not dispatchable in this
release (#276)" ([workflow_stage_plan.go:112](../../../cmd/lingo/workflow_stage_plan.go)).
In addition, the production observer proves only `axiom-skills`
([observation.go:17-21,107-128](../../../internal/runtimeadapter/observation.go)),
so even a wired dispatch path would refuse every real stage that needs
`read`, `repository-write` or `process`
([issue-275-implementation.md:303-309](issue-275-implementation.md)).

### 1.2 Expected user-visible behavior

From the installed `axiom` CLI, or from the `axiom-work-item` skill in either
Codex (`$axiom-work-item`) or Claude (`/axiom-work-item`), a developer whose
Work Item Execution is bound to an immutable workflow revision can:

1. establish, with explicit authorization, bounded proof that each installed
   and configured Runtime binding can actually read, edit and run commands in
   an authorized worktree non-interactively, and see what is `proven`,
   `unsupported` or `unknown`;
2. preview dispatch of a planned stage and review, per child: Runtime,
   Profile, model and Profile-owned effort, required capabilities and their
   proof, **authorized effects**, the **enforced confinement** (OS-enforced
   or advisory) and validators;
3. confirm it once; the confirmation is persisted with the graph before any
   process starts and covers only the first attempt of each child;
4. let Axiom dispatch ready Codex and/or Claude children concurrently only
   where allowed, without the initiating Runtime session staying alive;
5. observe parent/child progress, blockers, coordination and dependencies
   from either Runtime or the CLI, including after the initiating session ends;
6. cancel, reconcile uncertain attempts, retry explicitly and request the
   separately authorized integration step;
7. receive truthful `denied_authority`, `validation_failure`, `interrupted`,
   `failure/recovery_required` and `partial` outcomes.

Agentic capability is not machine-wide authority: children write only in
their approved worktree and their own coordination inbox, run commands only
within the approved envelope, and never push, open PRs, merge, publish or call
Providers.

### 1.3 Included capabilities

- Public application service composing graph, scheduler, workspace,
  integration, coordination and Runtime-policy services for one Work
  Item-bound Execution stage, graph or single-agent.
- **Capability observation and proof (IG-4 / amendment G-3, D-1)** for the
  capabilities stage execution needs, bound to Runtime, executable identity,
  version, invocation mode and, where applicable, model and effort (§3.9).
- CLI operations under `axiom workflow` (Execution family) and `axiom
  runtime <id> capability` (§5), with thin skill routes.
- Child coordination publish/read with lineage, revision, scope, provenance
  and capacity enforcement; **durable coordination-to-graph classification**
  (CR-001, §3.5).
- Durable dispatch authority for graph and single-agent paths (G-5).
- Cancellation per Spec 007, interruption, stale/orphan detection,
  append-only reconciliation, explicit retry.
- Help/result metadata, docs, deterministic tests and a bounded real
  two-Runtime smoke procedure executed only under separate authorization.

### 1.4 Non-goals

UI/dashboard; long-lived daemon, broker, HTTP/MCP server or new control
plane; general distributed workflow engine; API-billing prerequisite;
automatic fallback, merge, push, PR or release (#277); local Runtime/Profile
authoring, onboarding templates, Profile-owned effort storage and the
`axiom-runtime` skill (#306 / PR #310); credential capture or management;
Project policy editing (#305); new global Doctor (#231); #241 hardening;
R-2/R-3 acceptance (#278). Capability proof covers only what stage execution
requires (§3.9), not a general vendor capability catalog.

### 1.5 Relationship to the MVP journey

Spec 007 order is `#273 → (#274, #275) → #276 → #277 → #278`
([spec.md:561-562](spec.md)); #306 joins as the local-configuration
prerequisite. #276 turns a bound Execution and a stage proposal into
proven-capable, authorized, executed, coordinated and integrated stage
outcomes. It produces deterministic/synthetic Evidence plus one early real
engineering smoke; R-2/R-3 remain with #278.

## 2. Current-state assessment (observed)

### 2.1 Reusable components

| Area | Component | Evidence |
|---|---|---|
| Binding | `workflow start` creates an Execution with immutable `WorkflowBinding`; `workflow status/resume` | [execution_target.go:54-112](../../../cmd/lingo/execution_target.go), `internal/workflow/service.go:281,460` |
| Stage compilation | `workflowcompiler.Compile` → `graph` proposal or `single`; `PlanStage` → `StagePlan`; requested effort currently taken from `agent.Effort` | `internal/workflowcompiler/compiler.go:145,186`, `stage_plan.go:27-137` |
| Public plan op | `workflow stage plan` (read-only, digest-bound) | [workflow_stage_plan.go:43-112](../../../cmd/lingo/workflow_stage_plan.go) |
| Graph model | Parent/children, envelopes, attempts; statuses not-started/running/blocked/succeeded/failed/cancelled/skipped/unknown; `Attempt{Status, ExitCode, ResultReference, AmbiguousEffect, CancellationSeen}`; `AuthorityReferences map[string]string` | `internal/executiongraph/graph.go:29-100` |
| Graph publication | `GraphService.Publish(PublicationRequest)` with create + reload + digest compare | `internal/executiongraph/graph.go:126-209` |
| Planner rules | DAG, one integration+validation owner, unsafe overlap, authority subset, limits | `internal/executiongraph/planner.go:17-23,73-93,117-170` |
| Scheduler | Ready set from last-attempt status; non-conflicting batch; attempt persisted `running` before spawn; terminal save after `wg.Wait`; explicit retries; `CancelRequested` | `internal/executiongraph/scheduler.go:94-262` |
| Integration | Digest-bound preview/execute, single persisted attempt | `internal/executiongraph/integration.go:215,280,332` |
| Composition | `graphapplication.LocalService` + per-dispatch policy guard (preview, executable digest, credential reference, subscription auth) | `internal/graphapplication/local.go:49-150`, `runtime_policy.go:87-161` |
| Adapters | Codex `exec --model`, Claude `--print --model`; effort only when capability-proven | `internal/runtimeadapter/adapters.go:92-150` |
| Observation contract | `Observation{Installed, Available, Version, ExecutableDigest, CapabilityStatus, NonInteractiveModelCapabilities}`; `supports` requires Profile declaration **and** `CapabilityProven`; effort additionally requires per-model proof and a known version | `internal/runtimeprofile/runtimeprofile.go:37-38,70-86,297-313` |
| Auth preflight | Per-invocation subscription preflight; `credential_reference_configured` refusal | `internal/runtimeadapter/authpreflight.go:162`; `runtime_policy.go:117` |
| Coordination | Nine kinds, hash-chained per-child streams, filters, 64 records/stream, 64 KiB/record | `internal/coordination/coordination.go:42-52,78,115,209,301-347` |
| Stores | `local.GraphStore` (CAS), `local.CoordinationStore` (lock, chain re-check) | `internal/local/graph_store.go`, `internal/local/coordination_store.go:97` |
| Skills | One embedded set for Codex and Claude; `axiom-work-item` owns run/status/plan | `internal/codexruntime/runtime.go:15`; ADR-0022:58-67 |

### 2.2 Gaps for #276

| # | Gap | Evidence | Disposition |
|---|---|---|---|
| G1 | No production `StagePlan` → `GraphService.Publish` path | `stage_plan.go:94-109`; `s9-graph-runner.go:309-323` | Extend |
| G2 | `LocalService` not wired; in-memory graph; no load/resume | `local.go:49-150` | Extend |
| G3 | No public stage/coordination/capability operations | `internal/cli/commands.go:26-83` | New |
| G4 | No stdin on `Invocation`/`OSProcessRunner`; prompt appended to argv by runner | `scheduler.go:22-31,361-366`; `s9-graph-runner.go:249` | Extend |
| G5 | Argument guard is a denylist of selection options; does not reject permission/sandbox bypass options | `runtime_policy.go:207-232` | Replace by allowlist (§3.6) |
| G6 | No production `CommandProfile` source; no `CredentialResolver` | `cmd/lingo/runtime_policy.go:87-104`; `adapters.go:56-75` | Derive (§3.6) |
| G7 | Coordination store: only `Latest`/`Publish`; empty `AttemptID` accepted | `coordination.go:115,301-307` | Extend |
| G8 | **Defect:** `readyChildren` redispatches a child whose last attempt is `running` when retry is authorized | `scheduler.go:237-255` | Fix + regression |
| G9 | No stale/orphan detection, reconciliation or process-group control | `scheduler.go:361-381` | New |
| G10 | Timeouts yield `unknown` (`Stopped: false`) | `scheduler.go:378-381` | Keep truthful |
| G11 | Validators resolve only `builtin-sdd-v1`; repository commands need separate authority | `workflow_references.go:93-116`; [spec.md:176-180](spec.md) | PD-7 |
| G12 | Plan approval not durable (G-5) | amendment:323 | PD-3 |
| G13 | Subscription auth opt-in | #272 record:124-127 | PD-5 |
| G14 | Children inherit full `os.Environ()` | `scheduler.go:336-357` | Allowlist (§3.6) |
| G15 | No help/result metadata or skill routes | `skill_inspect.go:60-215` | New |
| G16 | Concurrency only compile-time | `compiler.go:397` | Defensive check |
| G17 | **Capability proof missing:** observer proves only `axiom-skills`; version always unknown; no proof for `read`, `repository-write`, `process` or per-model effort | `observation.go:83-128`; `docs/commands.md:603-610` | **#276 deliverable (§3.9, D-1)** |
| G18 | Coordination outcomes have no durable effect on graph state: scheduler readiness and integration read only attempt status/exit code | `scheduler.go:210-262`; `integration.go:332` | **CR-001 (§3.5)** |
| G19 | Effort is a Workflow agent field (`agent.Effort` → `RequestedEffort`), contradicting the #306 D3 direction (effort owned by local Model Profile) | `compiler.go:186`; `stage_plan.go:134-137`; `workflowdefinition/definition.go:102-117` | Consume #310 amendment (IG-5, §3.10) |

## 3. Proposed architecture

### 3.1 Containers and boundaries (C4 level 2, delta only)

The container view in `docs/architecture/` is unchanged: one local `axiom`
binary over machine-local state. #276 adds one application component
(`stageexecution`, name provisional), one capability-proof component, a
bounded supervisor mode of the same binary, and CLI/skill entrypoints.

```text
 Codex/Claude session (skills) ─┐   Terminal ─┐        (operator surfaces only)
                                ▼             ▼
        internal/cli  (parse, help, render; no domain rules)
                                ▼
 cmd/lingo composition ─► stageexecution.Service   ─┬─ workflow.Service (binding, gates, ledger)
                                                    ├─ workflowcompiler (recompile, digest compare)
                                                    ├─ runtimeapplication (Project ∩ local Profile ∩ observation)
                                                    │      └─ capability observer ◄── capabilityproof (NEW)
                                                    ├─ executiongraph (Publish, Scheduler, Integration, classifier)
                                                    ├─ graphapplication (LocalService, guard)
                                                    ├─ coordination (validate, chain, stream read)
                                                    └─ runtimeadapter (argv allowlist, confinement, env, packet, auth)
                                ▼
 internal/local stores: workflow, graph (+classification, reconciliation), coordination,
                        capability evidence (NEW), runtime profile (read-only here; authored by #306)
                                ▲
 axiom stage supervisor (same binary, detached, one per run) ─► children in own process group
                                                                stdin packet; writes: worktree + own inbox
```

Every rule stays with its current owner; the new components compose. CLI
handlers and skills only pass DTOs and render results.

### 3.2 Process model (PD-2)

Children run minutes to hours (per-child timeout ≤ 24h, `planner.go:17-23`).
The initiating Runtime runs `axiom` inside a tool call with a much shorter
limit (Claude Code shell calls default to 2 minutes and cap at 10
*(documented vendor behavior; verify)*; Codex limit *unverified*), so a
foreground wait would be killed and orphan its children.

Recommended: **detached bounded supervisor (same binary), no daemon.**
`stage run` apply commits authority + graph (§3.3), starts one detached
supervisor for that run (hidden internal entrypoint), records its
correlation (PID, process start identity, lease epoch) and returns. The
supervisor holds the run lease, dispatches ready non-conflicting batches,
ingests coordination, classifies and persists terminal attempts (§3.5) and
exits when nothing more is dispatchable under existing authority or on
cancellation. It never retries, reconciles, integrates or performs any
effect needing new authority. Children run in their own process group
(POSIX) / job object (Windows). Alternative: foreground step capped below
the host tool limit (terminal use only).

### 3.3 Dispatch authority (PD-3, PD-12; closes G-5)

- **Preview** (no authority flags) recompiles under the bound revision,
  compares the reviewed `StagePlan`/`planDocumentDigest`, re-observes policy
  **and capability evidence** (§3.9), checks gates and returns a dispatch
  preview whose digest binds `ExecutionRef`, `workflowRef`, `stageId`,
  plan digests, graph proposal digest, each `Resolution` (Runtime, Profile,
  model, Profile-owned effort, configuration/observation revisions,
  executable digest, version), capability evidence digests, effects,
  workspaces, confinement per child, validator policy reference (PD-7),
  attempt limits, parent ceiling and the `axiom` binary digest given to
  children.
- **The preview presents the decision surface per child:** what it may read
  (worktree; reads outside are *not* OS-confined for either Runtime —
  §3.6), where it may write (worktree + own inbox), which commands it may
  run (within the worktree, under the confinement shown), the confinement
  level per Runtime (`os-enforced` / `advisory`), the probe-observed writable
  set and network posture, and what is prohibited (push, PR, merge, Provider
  calls, credential access, nested agents). The preview states whether each
  prohibition is **enforced or policy-only** for that child (§3.6).
- **Apply** (`--preview-digest`, `--expected-revision`, `--authorize-local`
  and, because it starts vendor inference, `--authorize-external`; PD-12)
  re-observes everything, refuses drift (`denied_authority/stale_preview`),
  then commits a **dispatch-authority record** {preview digest, scope =
  first attempt of each child of graph revision R (or the single attempt),
  actor kind `human-operator`, time} with the graph (or ledger attempt).
  Children reference it through the existing `AuthorityReferences`.
- Retry, reconcile, cancel, integrate and capability probe each need their
  own preview and authority. The supervisor continuing already-authorized
  first attempts re-checks drift (including capability evidence) and stops on
  any. Gates declared `before` the stage must already be satisfied via
  `workflow fact`; agent records never commit authority
  ([spec.md:205-208](spec.md)). Skills pass authority flags only after
  explicit human confirmation.

### 3.4 Coordination ingress (PD-4)

Children never write canonical Axiom state. `axiom workflow coordination
publish`, run inside a child, resolves identity from its workspace plus
attempt correlation, validates with existing `coordination` rules, requires a
non-empty attempt ID and writes a candidate only into its private **attempt
inbox**. The supervisor ingests candidates **concurrently** and performs a
**final drain before the terminal save** (§3.5). Ingestion re-validates
lineage (parent, child, graph revision, attempt owned by the current lease
epoch), chain revision/previous digest, capacity and cross-stream references
(`answer.question_reference`, `contract_acceptance.contract_reference`), then
publishes through `coordination.Service` and `local.CoordinationStore`.
Rejected candidates become bounded diagnostics; candidates from a previous
epoch are quarantined. Reads use canonical streams only (`--wait` bounded and
shorter than the child timeout). The packet tells an asker to publish a
`blocker` and exit rather than wait past its timeout. Provenance is recorded
as "inbox of child X, attempt Y, epoch E"; under
[ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md) a
same-user process can still forge files, so records never grant authority
and confinement is reported truthfully (§3.6).

### 3.5 Coordination-to-graph state (CR-001, PD-9)

Coordination outcomes become **persisted graph state**, computed once and
read by every consumer. Rendering never reinterprets them.

**Record ordering and capacity (prerequisite).**

- Each inbox candidate carries a per-attempt sequence number. The supervisor
  publishes accepted candidates in that order and assigns the canonical
  stream `Revision`, so a child never needs to know the stream head
  (`coordination.go:140-151`).
- "Latest within the attempt" means the highest canonical `Revision` whose
  `AttemptID` equals the attempt and whose graph revision is current.
  Rejected or quarantined candidates never count.
- `result.outcome` becomes a required enum (`succeeded` / `failed`).
  Today it is free text (`coordination.go:337`).
- Stream capacity is reserved per attempt. The existing 64-record per-stream
  cap (`coordination.go:27,153`) is divided by `MaximumAttempts`, and one
  slot per attempt is reserved for its terminal `result` or `blocker`, so a
  late retry cannot be starved.

**Component and interface.**

- `executiongraph` gains a pure classifier `ClassifyAttempt(process
  ProcessResult, cancelRequested bool, drained []coordination.Record) →
  (AttemptStatus, ClassificationReason, qualifyingRecordRef)` *(signature
  indicative)*.
- The supervisor calls it **after the final inbox drain and before the
  terminal save**, and persists the result **in the same CAS save**.
- Each attempt is saved as soon as it is classified, not once per batch
  (today it is per batch after `wg.Wait`, `scheduler.go:198-205`).
- `Attempt` gains two additive fields:
  - `classificationReason`;
  - `qualifyingRecordRef`, the digest of the qualifying canonical record.
    It is distinct from the existing `ResultReference`, which stays the
    process-output digest.
- The graph gains a `classificationPolicy` marker set at publication, which
  identifies graphs created by #276.
- `ExitCode` is always retained.

**Rules (ordered, first match wins):**

| # | Condition | Persisted status | Reason |
|---|---|---|---|
| 0 | Cancel requested, stop of the process group confirmed, no ambiguous effect | `cancelled` | `cancel_confirmed` |
| 1 | Process outcome unknown (timeout or interruption without confirmed stop, lost supervisor) | `unknown` | `process_outcome_unknown` |
| 1b | Timeout with confirmed stop and no ambiguous effect | `failed` | `timeout_stopped` |
| 2 | The latest of {`blocker`, `result`} within the attempt is a `blocker` | `blocked` (existing status) | `blocker_published` |
| 3 | Exit code ≠ 0 | `failed` | `process_failed` |
| 4 | Exit 0, no correlated `result` | `failed` | `result_missing` |
| 5 | Exit 0, `result.outcome` = `failed` | `failed` | `result_not_successful` |
| 6 | Exit 0, `result.outcome` = `succeeded` | `succeeded` | `result_recorded` |

Integration attempts are not child work. They keep the existing integration
logic and record their own reasons (`integration_applied`,
`integration_failed`, `integration_unknown`), so I-1 holds for them too.

**Required scheduler and integration changes (explicit):**

| Area | Today | Change |
|---|---|---|
| Dependency readiness | Satisfied when the last attempt is `succeeded` (`scheduler.go:210-235`) | Unchanged. A `blocked` or `failed/result_missing` child therefore never unlocks dependents |
| Redispatch of `running` | Not gated (G8) | Never ready while `running` |
| Retry eligibility | Any non-succeeded status with `RetryChildIDs` passes, except `unknown`/ambiguous, which are refused unconditionally (`scheduler.go:237-255`); `unknown` always sets `AmbiguousEffect` (`scheduler.go:182`) | Retry allowed from `failed`, `blocked` and `cancelled` with authority. From `unknown`/ambiguous only when an appended reconciliation fact for that attempt exists (§3.8). `skipped` never retried |
| Integration eligibility | `validChildResult` checks last attempt `succeeded` and caller-supplied `ChildResult` (`integration.go:343-357`); `integrationAttemptAvailable` refuses unknown/ambiguous (`integration.go:332-340`) | Also requires `classificationReason == result_recorded` and the `ChildResult` digest bound to `qualifyingRecordRef`. On graphs without the classification policy marker, #276 refuses to integrate |
| Status | — | Renders persisted fields only |

**Consequences:**

- **A child exiting 0 after a `blocker` stays blocked.** It is persisted as
  `blocked`, its dependents remain `dependencies_unsatisfied`, and leaving
  that state requires an authorized `stage retry` (a new attempt) after the
  blocker's `required_action` is addressed.
- **A required child exiting 0 without a `result` is not
  integration-eligible.** It is persisted as `failed/result_missing`, so its
  dependents do not start and integration preview refuses roll-up.

**Restart, recovery and reconciliation:**

- Classification happens only at the terminal save. If the supervisor dies
  between the child's exit and the save, the attempt stays `running`, becomes
  stale once the lease is free, and its inbox is quarantined. Only rule 1 can
  later describe it.
- Reconciliation appends a fact and never rewrites a persisted status (§3.8).
  Quarantined content is diagnostics only.
- Records that arrive after the terminal save are rejected.
- Resume reads persisted fields only, so status, dispatch and integration
  agree before and after restart.

**Invariants (tested):**

- I-1: every terminal attempt of a graph carrying the classification policy
  marker has a non-empty `classificationReason`.
- I-2: `succeeded` implies `result_recorded` and a resolvable `result`
  record (`outcome = succeeded`) of the same attempt, graph revision and
  parent, whose digest equals `qualifyingRecordRef`.
- I-3: `blocked/blocker_published` implies a resolvable `blocker` record of
  that attempt.
- I-4: scheduler readiness, retry eligibility, integration eligibility and
  status read only persisted fields and reconciliation facts. No consumer
  calls the classifier.
- I-5: no record published after an attempt's terminal save affects it.
- I-6: graphs without the classification policy marker (for example
  historical S9 runner graphs) stay readable through version-tolerant
  decoding but are refused by #276 mutating operations with
  `recovery_required/unsupported_graph_format`.

### 3.6 Child invocation, confinement and environment (G4–G6, G14)

- **Command profile:** Runtime, Profile, model, effort (Profile-owned, §3.10)
  from the local configuration authored by #306; executable from `PATH`
  observation pinned by `executableDigest`; `Arguments` and `OutputMax` are
  **Axiom constants per Runtime and mode**. No new configuration path.
- **Argument allowlist:** argv is built only from the adapter's constant
  allowlist plus confinement options appended last; anything else is refused.
  Bypass options observed in the installed help output (Codex
  `--dangerously-bypass-approvals-and-sandbox` and `--sandbox
  danger-full-access`; Claude `--dangerously-skip-permissions`,
  `--allow-dangerously-skip-permissions` and `--permission-mode
  bypassPermissions`) are never emitted and are rejected in tests, as are
  `--bare` (Claude: disables OAuth, so incompatible with subscription auth
  per help text) and options that select alternative configuration
  (`--profile`, `--settings`, `--config` except adapter-owned effort).
- **Observed vendor confinement surfaces** (help output only; proof is by
  probe, §3.9):
  - Codex `exec`: prompt from stdin with `-`; `--sandbox read-only |
    workspace-write` (OS sandbox for model-generated commands); `--add-dir`
    for extra writable directories; `--ignore-user-config`, `--ephemeral`.
  - Claude `--print`: `--input-format`, `--permission-mode` (`dontAsk`,
    `acceptEdits`, …), `--permission-prompts none`, `--allowedTools` /
    `--disallowedTools`, `--add-dir`. No OS sandbox option appears in the
    help output.
- **Confinement variants:**
  - Every coordinating child must be able to write its own inbox, so no
    variant can be write-free.
  - Codex children use `workspace-write`:
    - `read`-only children run with the attempt inbox as the only writable
      root (inbox as working root via `-C`, the dispatch worktree read
      through its path);
    - `repository-write` children run with the worktree as working root plus
      `--add-dir <inbox>`.
  - The exact layout is fixed in Tasks and proven per variant by the probe
    (§3.9). Codex help describes `--add-dir` only as "writable alongside the
    primary workspace".
  - Claude tool restrictions (`--permission-mode dontAsk`, `--allowedTools`,
    `--add-dir`) are **advisory** unless the probe demonstrates enforcement
    for the exact binding.
- **What is and is not enforced (stated per Runtime in the preview):**
  - Reads are **not OS-confined** for either Runtime.
  - Codex writes are OS-confined to the writable set **the probe actually
    observed**. This may include `$TMPDIR` or `/tmp`
    *(unverified; recorded, never assumed)*.
  - Claude writes are advisory.
  - Because `HOME` is passed for vendor login, an advisory child can reach
    user-level tools and credentials such as SSH keys or a `gh` login, and
    could in principle push or open a PR. The prohibition on push, PR, merge,
    Provider calls and credential access is therefore **policy, not
    enforcement**, for advisory children, and the preview says so.
  - Network access is a separate dimension. It is pinned per variant where the
    Runtime offers it and probed and reported like writes.
- **Task packet on stdin:** role, responsibilities, instructions, validated
  input refs/digests, outputs, criteria, allowed paths/effects, coordination
  protocol and limits (including "publish `result` before exiting 0"), the
  pinned `axiom` path and digest, and prohibitions. No effort, model or
  credentials in the packet; no raw chat. Packet digest recorded on the
  attempt.
- **Environment allowlist (ADR-0009 minimum credential capability):** the
  child environment is constructed from a **closed list**:
  - `HOME`, `USER`, `LOGNAME`, `PATH`, `SHELL`, `TMPDIR`;
  - `LANG` and `LC_*`;
  - `CODEX_HOME` (Codex only) and `CLAUDE_CONFIG_DIR` (Claude only), each
    only when already present;
  - the Axiom correlation variables.

  Any change to the list is a contract change. Ambient tokens (`GH_TOKEN`, provider API keys and similar) are
  dropped. Authentication stays vendor-managed (#306 D2). Axiom never reads,
  copies or stores login material. The auth preflight runs on exactly this
  environment.

### 3.7 Leases, locks and multi-object commits

The **run lease** is a non-blocking exclusive OS lock per graph with a
monotonically increasing **lease epoch** recorded on attempts. It is first in
the ADR-0007 total lock order, before graph CAS and coordination locks.
Status probes it non-blockingly. A `running` attempt whose lease is free is
**stale**. If its process group is still alive it is an **orphan**: it is
reported, its inbox is quarantined, and retry stays blocked until it is
stopped and reconciled. Authority + graph and stage outcome + ledger are
multi-object commits under ADR-0007 commit-point/recovery rules (exact
objects in ADR-0024 and Tasks).

### 3.8 Reconciliation, retry and cancellation

- **Reconciliation is append-only** (PD-10). Preview gathers observations
  (lease, process liveness, worktree status, canonical records, quarantined
  inbox). Apply appends {attempt, observations digest, classification,
  actor, time} next to the attempt and never changes its status. The
  scheduler reads the fact to allow an explicit retry.
- **Retry** creates a new attempt under the same child. It needs remaining
  `MaximumAttempts`, fresh capability evidence, its own authority and an
  explicit workspace disposition after effects (PD-8). Eligibility by prior
  status:
  - `failed`, `blocked` or `cancelled`: retry is allowed;
  - `unknown` or ambiguous: allowed only once an appended reconciliation fact
    exists for that attempt. This is a **scheduler change**, because those
    attempts are refused unconditionally today (`scheduler.go:239-246`);
  - `skipped`: never retried.
- **Cancellation** (PD-6) persists a cancel request. The scheduler stops new
  dispatch (`CancelRequested` → `cancelled_before_dispatch`), and the
  supervisor requests a bounded process-group stop. An attempt becomes
  `cancelled` only when the stop is confirmed and there is no ambiguous
  effect; otherwise it is `unknown` + `CancellationSeen`
  ([ADR-0009](../../decisions/0009-parent-child-execution-graph.md):117-119).
  Execution-level `workflow cancel` stays `invalid_command`.

### 3.9 Capability observation and proof (IG-4 / G-3; D-1; PD-13, PD-14)

**Three separate questions, three separate records:**

| Question | Source | Never inferred from |
|---|---|---|
| Is the Runtime/Profile **declared** for this work? | Project allowlist ∩ local Profile (#306) | — |
| **Can** this exact Runtime binding do it? (capability) | Capability evidence (this section) | installation, version, authentication, Profile declaration, help text |
| **May** it do it here? (effect) | Dispatch authority + effect ceilings | capability proof |
| What is **enforced** if it tries more? (confinement) | Probe-verified confinement level | vendor documentation, flags alone |

**Required capability vocabulary (amendment #276).** A child's required set
is its declared `capabilities` (e.g. `read`, `repository-write`), plus
`repository-write` and `process` when the corresponding effects are
authorized, plus the implicit execution capabilities `noninteractive`
(headless run from the stdin packet with a meaningful exit status and no
prompts) and `axiom-coordination` (runs the pinned `axiom` binary and
publishes into its own inbox). `read` means reading files in the dispatch
worktree, `repository-write` means creating and editing files in it, and
`process` means running commands in it. Model and effort support are
recorded **separately** as per-model `model-invocation` and
`reasoning-effort-<token>` (existing
`NonInteractiveModelCapabilities` slot).

**Where each check happens.**

- **Declared capabilities** go through the existing `runtimeprofile`
  resolver. It requires both the Profile declaration and `CapabilityProven`
  (`runtimeprofile.go:297-313`), unchanged.
- **Implicit capabilities** (`noninteractive`, `axiom-coordination`, and the
  effect-derived `repository-write`/`process`) and **model-scoped
  capabilities** are checked by the #276 capability service against evidence
  for the child's confinement variant. They are not added to the resolver
  request, so existing #306 Profiles need not declare them. Today the
  resolver checks the per-model slot only for `reasoning-effort-*`.
- The observer is instantiated per confinement variant, because
  `CapabilityStatus` is a flat per-Runtime map.
- `CapabilityEvidenceRef` in the stage plan and preview becomes the evidence
  record digest. Today it is the observation digest (`stage_plan.go:136`).

**Evidence binding.** One evidence record per {Runtime ID, executable
digest, observed version, invocation-mode digest (constant argv template +
confinement variant `read-only` or `workspace-write` + environment allowlist
digest), Axiom adapter version}. Model-scoped results also bind {model,
Profile-owned effort token}. Each record stores per-capability state, the
verification facts, confinement observed, `observedAt` and the probe packet
digest. It stores no transcripts, credentials or account identifiers, and
lives in a new machine-local store owned by #276. It does **not** change the
#306 Profile format.

**Observation levels:**

1. **Interface observation (local, no inference).** `<exe> --version` and
   `--help` (or `exec --help`) are run under the constructed environment. The
   result can only establish `unsupported`, when a required option or mode is
   absent for the observed version, or confirm that a probe is possible. It
   never yields `proven`.
2. **Behavioral calibration probe (vendor inference; separately
   authorized).** `axiom runtime <codex|claude> capability probe --profile
   <id>` preview/apply runs the Runtime in the **exact dispatch invocation
   mode** (same adapter, argv allowlist, environment and packet format), once
   per confinement variant. It runs in an Axiom-created disposable scratch git
   repository under the state root, never a user repository, with a bounded
   timeout, one attempt and no retry. The canary tasks are: read a file
   containing a nonce, create and edit a file with derived content, run a
   command (the pinned `axiom` canary/coordination publish, which proves
   `process` and `axiom-coordination`), exit 0 after publishing a `result`,
   and attempt a write to a sentinel directory inside the probe root but
   outside the allowed write set.

   **Axiom verifies outcomes by inspecting the filesystem, inbox and exit
   status, never by trusting the agent's statements.**

   Confinement is measured by the **deterministic `axiom` canary** that the
   child is asked to run, never by the model's own attempt.
   - The canary tries writes to the sentinel, `$TMPDIR`, `/tmp`, `$HOME` and
     the vendor home, and one network connection.
   - It records each errno or result in its inbox record.
   - The observed writable set and network posture are stored as evidence.
   - `os-enforced` is concluded only when the canary itself ran and was
     denied outside the allowed set.
   - If the canary did not run, or the model refused, the result is
     `unknown`, never `os-enforced`.

   Each confinement variant is probed separately. Model invocation with the Profile's model/effort is
   recorded as model-scoped evidence.

**States** reported by `capability show`, preview and blockers:

- `proven`: matching, fresh evidence with verified facts;
- `unsupported`: required interface absent, or the probe verifiably failed
  the capability in the correct mode with authentication proven;
- `unknown`: no evidence, stale evidence (any binding element changed),
  expired evidence (older than a fixed maximum age; PD-13), explicitly
  invalidated evidence (`capability invalidate`, or the Profile disabled or
  removed through #306), an inconclusive probe (timeout, interruption,
  usage limit), or authentication not proven during the probe.

The `runtimeprofile.Observation` contract is unchanged. The observer sets
`CapabilityStatus[c] = proven`, or the per-model entries, only from matching
fresh evidence. `unsupported` and `unknown` are reported through #276
blockers (`capability_unsupported`, `capability_unknown`, `capability_stale`)
and are never mapped to `proven`.

**Revalidation.** Preview and every dispatch (including the supervisor's
continuation and retries) recompute the binding tuple:
- executable digest, which hashes the resolved target file; for a launcher
  under `~/.local/bin`, that is the launcher itself, and a change to what it
  launches is caught by `--version` and the probe binding;
- `--version`;
- mode digest;
- adapter version;
- pinned `axiom` binary digest;
- Profile model/effort and configuration revision.
Any mismatch makes the capability `unknown` and blocks with
`capability_stale`. A mismatch between preview and dispatch is reported as
`denied_authority/stale_preview`.

**Fail-closed rule.** Any required capability not `proven` blocks dispatch
with an actionable blocker naming the Runtime, binding, capability, state and
next action (`axiom runtime <id> capability probe …`, which needs authority).
There is no fallback to another Runtime/Profile and no dispatch "to see if it
works".

**Authentication is independent.** A failed or unproven auth preflight blocks
with `authentication_blocked`. It does not create, change or invalidate
capability evidence, and a capability probe cannot run without proven
subscription auth.

**Gap handling.** If a capability cannot be demonstrated for a Runtime
version (for example, Claude without observable OS write confinement), the
outcome is recorded truthfully (`advisory`, or `unsupported` for a missing
interface) and surfaced in preview. Nothing is bypassed or promoted.

### 3.10 Effort and model (reconciliation with #306 D3; IG-5, PD-15)

- Effort is owned by the local Model Profile (#306). The workflow references
  a Profile and does not specify or override effort. `complexity` stays a
  task requirement independent of effort.
- #276 consumes the Profile's effort. The resolution, dispatch preview and
  envelope `Controls.ReasoningEffort` carry the Profile effort, and the task
  packet carries none. The adapter emits it only with matching per-model
  evidence (`reasoning-effort-<token>` proven for the exact model, version
  and mode). Missing or non-matching effort evidence blocks with
  `capability_unknown`; it never falls back silently to the Runtime default.
- Compatibility: agents with `effort {mode: runtime-default}`, which covers
  every built-in `default-sdd` stage, are unaffected. A pinned historical
  revision with an explicit agent effort is never rewritten. A new dispatch
  under it is refused with `workflow_effort_override_unsupported` and
  directed to a new revision. Existing Executions keep their bindings.
- The Spec 007 / `default-sdd` schema change that removes or retires agent
  effort is **owned by PR #310** per its maintainer decision (mandatory change
  2). #276 depends on it (IG-5) and does not author it. Amendment #276 states
  only the dispatch-side consumption rule.

### 3.11 Single-agent stages (PD-11)

`executionKind: single` keeps the sequential path with no graph parent
([spec.md:347-350](spec.md); ADR-0020:52-55). Recommended: the same `stage
run/status/cancel/reconcile/retry` operations bind one attempt in the stage
ledger and dispatch it through the same adapter, packet, confinement,
capability proof, classifier (§3.5) and supervisor. No graph, integration
child or second process is fabricated.

## 4. Execution lifecycle

```text
(once per binding) runtime <id> capability probe [preview/apply, local+external authority] ─► evidence
Work Item ─ workflow start (#274) ─► Execution + immutable WorkflowBinding
   └─ workflow stage plan (#275) ─► StagePlan digest (no authority)
        └─ stage run [preview] ─► per-child rights, capability states, confinement, digest
             └─ stage run [apply + local + external authority]
                  ├─ re-check revisions, binding, gates, policy, capability evidence, auth
                  ├─ commit dispatch-authority record + graph (or ledger attempt)  ← commit point
                  ├─ prepare isolated worktrees
                  └─ start detached supervisor ─► lease(epoch) → attempt `running` → spawn group
                        → ingest → final drain → ClassifyAttempt → terminal CAS save → release
        └─ stage status / coordination list ─► persisted status, reasons, records, stale/orphan
        └─ stage cancel | reconcile | retry | integrate [preview/apply]
             └─ integration reads persisted eligibility → validators (PD-7) → roll-up → ledger
        └─ workflow status / advance (#274) ─► next stage or #277
```

| From | Event | To (persisted) | Notes |
|---|---|---|---|
| planned | apply with drift (incl. capability) | unchanged | `denied_authority/stale_preview` |
| published | capability not proven | child `blocked` (`capability_unknown/unsupported/stale`) | no attempt, no process |
| published | auth unproven | child `blocked` (`authentication_blocked`) | evidence untouched |
| `running` | exit 0 + `result` succeeded | `succeeded/result_recorded` | eligible |
| `running` | exit 0 after `blocker` | `blocked/blocker_published` | dependents not dispatched |
| `running` | exit 0, no `result` | `failed/result_missing` | dependents and integration blocked |
| `running` | exit ≠ 0 | `failed/process_failed` | retry needs authority |
| `running` | timeout / interruption | `unknown/process_outcome_unknown` | reconcile first |
| `running` | cancel confirmed, no ambiguity | `cancelled` | else `unknown` + `CancellationSeen` |
| `running` | supervisor dies | stays `running` → stale/orphan | `failure/recovery_required`; G8 fix |
| stale/unknown | reconcile | unchanged + fact | never upgraded |
| `failed`/`blocked`/reconciled `unknown` | retry | new attempt | PD-8 disposition |
| all required `succeeded` | integrate | integration attempt | roll-up |

## 5. Public interface design

Existing selectors, `--expected-revision`, `--preview-digest`,
`--authorize-local`, `--authorize-external`, `--json`, canonical envelope and
exit codes ([cli.go:23-27](../../../internal/cli/cli.go)). Flag spellings are
final only in Tasks.

| Operation | Kind | Mutates | Authority | Owning skill | Contract basis |
|---|---|---|---|---|---|
| `workflow start/status/resume/fact/advance` | existing | as today | as today | `axiom-work-item` | #274 |
| `workflow stage plan` | existing | no | none | `axiom-work-item` | #275 |
| `workflow stage run` | new | authority, graph, supervisor | local + external | `axiom-work-item` | Spec 007 table |
| `workflow stage status` | new read | no | none | `axiom-work-item` | amendment #276 |
| `workflow stage cancel` | new | cancel/stop request | local | `axiom-work-item` | Spec 007 table; PD-6 |
| `workflow stage retry` | new | new attempt | local + external | `axiom-work-item` | Spec 007 table |
| `workflow stage reconcile` | new | append fact | local | `axiom-work-item` | amendment #276 |
| `workflow stage integrate` | new | local integration | local | `axiom-work-item` | amendment #276 |
| `workflow coordination list/show` | new read | no | none | `axiom-work-item` | Spec 007 table + amendment |
| `workflow coordination publish` | new, child-only | own inbox | attempt context | none (packet) | Spec 007 table |
| `runtime <id> capability show` | new read | no | none | `axiom-runtime` once #310 lands (PD-14) | amendment #276 |
| `runtime <id> capability probe` | new | evidence; vendor inference | local + external | `axiom-runtime` once #310 lands (PD-14) | amendment #276 |
| `runtime <id> capability invalidate` | new | evidence state | local | `axiom-runtime` once #310 lands (PD-14) | amendment #276 |
| supervisor entrypoint | hidden | as authorized | inherited, scoped | none | ADR-0024 |
| `workflow cancel` | existing unsupported | no | — | unchanged | #230 F-02 |

`coordination publish` outside a running child context fails closed. Results
use the existing refusal categories ([spec.md:448-456](spec.md)) plus the
operation categories listed in the amendment. No new completion status is
added. Skills relay human rendering and never compute digests or policy.

## 6. Agent orchestration

- **Topology.** One parent and up to 32 children. There is one
  integration+validation owner, and it depends transitively on all other
  children.
- **Cross-Runtime communication.** Children communicate only through typed
  records on per-child streams. Peers read canonical streams, and
  cross-stream references are validated at ingestion.
- **Ordering and concurrency.** Dependencies come from the compiled DAG.
  Stage `concurrency` is expressed as edges. Batches contain only
  non-conflicting effects, and writers use isolated worktrees. A defensive
  dispatch-time check enforces the recorded stage concurrency (G16)
  *(inferred: additive envelope field)*.
- **Orchestrator consumption.** It is deterministic and persisted (§3.5).
  Integration consumes only `succeeded/result_recorded` results, and does so
  as compiled inputs.
- **Retry and recovery.** See §3.8. Every retry or recovery revalidates
  capability evidence.

## 7. Security and authority

| Concern | Control |
|---|---|
| Authorization before dispatch | Dispatch preview + local and external authority; record committed before any attempt; first-attempt scope; gates checked |
| Capability vs effect vs confinement | Three separate records shown per child in preview; proof never grants authority; authority never implies proof |
| Capability proof | Probe-verified evidence only; installation, version, authentication, Profile declaration or help output never promoted to `proven` |
| Runtime / Profile resolution | `runtimeapplication.Check` at preview, apply and each dispatch; no fallback; initiating Runtime has no preference |
| Model / effort | Profile-owned (#306); used only with matching per-model evidence; no cross-vendor mapping |
| Authentication | Vendor-managed (#306 D2); subscription preflight mandatory (PD-5); no credential capture; credential references refused |
| Read/write/process scope | Writes: worktree + own inbox (Codex OS-enforced; Claude advisory unless proven). Reads: not OS-confined, stated in preview. Commands: run in worktree under the shown confinement |
| Arguments / environment | Constant allowlist; bypass options rejected; constructed environment without ambient tokens |
| Lineage / revision | Child/attempt/graph revision/epoch checks; recompile under bound revision; digest compare |
| Coordination provenance | Inbox + concurrent ingestion + final drain + epoch fencing; records grant nothing |
| Capacity | 32 nodes, 64 records/stream, 64 KiB/record, attempts 1–10, timeout ≤ 24h; probe bounded |
| External effects | Vendor inference (stage run/retry/probe, with external authority) and the local integration effect only. No push, PR, merge, release or Provider writes |
| Unknown outcomes | Stay `unknown`; retry only after reconciliation; stale/orphan never redispatched |
| Concurrency | Lease + CAS → `stale_preview` or `temporarily_unavailable` |
| Evidence hygiene | No credentials, account IDs, login material or transcripts; bounded digests and redacted excerpts only |

The Phase 3 security review covers stdin packet handling, inbox and probe-root
permissions (POSIX and Windows), the allowlist and bypass rejection,
environment construction, ingestion hardening, supervisor detachment and
process groups, capability evidence integrity and binding, and redaction.

## 8. Failure scenarios

| Scenario | Detection | Outcome (status / category) | Next |
|---|---|---|---|
| Runtime unavailable/disabled | policy Check | `validation_failure` / `runtime_unresolvable` | configure via #306 |
| Capability unknown (no/stale/expired/invalidated evidence) | capability observer | `validation_failure` / `capability_unknown` or `capability_stale` | authorized probe |
| Capability unsupported | interface observation or probe | `validation_failure` / `capability_unsupported` | different Profile/Runtime by explicit choice |
| Model/effort unproven | per-model evidence | `validation_failure` / `unsupported_effort` or `capability_unknown` | probe or change Profile (#306) |
| Explicit workflow effort in pinned revision | compiler | `validation_failure` / `workflow_effort_override_unsupported` | new revision |
| Authentication unavailable | preflight | `authentication_blocked` | vendor login (user-run) |
| Credential reference configured | guard | `authentication_blocked` / `credential_reference_configured` | #306 |
| Stale/invalid revision | recompile + digest | `denied_authority` / `stale_preview`; corrupt → `recovery_required` | new preview |
| Forged lineage | publish, ingestion, retry/reconcile | `denied_authority` | none |
| Unauthorized operation / scope escalation | authority check, planner subset | `denied_authority` | none |
| Capacity exceeded | ingestion | `validation_failure` (child); capacity blocker (parent) | — |
| Dependency/effect conflict | planner/scheduler | `ErrUnsafeOverlap` / `effect_conflict_serialized` | — |
| Blocker then exit 0 | classifier | child `blocked/blocker_published`; dependents not dispatched | address required action; retry |
| Exit 0 without `result` | classifier | `failed/result_missing`; integration refused | retry |
| Interrupted / timeout | supervisor | `unknown`; `interrupted` only without ambiguous effect, else `failure/recovery_required` | reconcile |
| Supervisor killed | free lease | `failure/recovery_required` (stale/orphan) | stop orphan; reconcile |
| Concurrent operations | lease, CAS, chain | `stale_preview` / `temporarily_unavailable` | retry operation |
| Probe inconclusive (usage limit, timeout) | probe | capability `unknown` | new authorized probe |
| Confirmed effect, projection failed | after commit | `partial` / `confirmed_effect_reconciliation_required` | reconcile |

## 9. Validation strategy

### 9.1 Required deterministic coverage

- **Capabilities.** Cover `proven`, `unknown` (none, stale, expired,
  invalidated, inconclusive), `unsupported` (interface absent, verified
  failure), and evidence that is stale or revoked between preview and
  dispatch. Cover the Runtime, version, executable digest, mode and
  model/effort binding dimensions. Prove that installation, version,
  authentication, help output or Profile declaration alone never yield
  `proven`, and that authentication failure leaves evidence unchanged.
- **Runtime differences.** Cover per-Runtime argv allowlists, rejection of
  bypass options, the confinement variants, and Codex `os-enforced` versus
  Claude `advisory` reporting. Unknown versions fail closed.
- **Drift.** Cover policy, observation, capability, revision and binary drift
  between preview and dispatch, and between dispatch batches.
- **Scope.** Cover child effects above the ceiling, writes outside the
  worktree or inbox in probe fixtures, publishing outside the child context,
  and forged or foreign or empty attempts.
- **Coordination.** Cover all nine kinds, chain, capacity, cross-stream
  references, epoch fencing and late records.
- **CR-001.**
  - The classifier table, row by row, including rows 0 and 1b.
  - Per-attempt sequence ordering and the "latest within the attempt" rule.
  - `result.outcome` enum validation.
  - The per-attempt capacity reserve.
  - Per-attempt terminal saves.
  - Integration bound to `qualifyingRecordRef`.
  - Retry from a reconciled `unknown` only.
  - The graph classification-policy marker.
  - A blocker followed by exit 0 keeps dependents undispatched.
  - Exit 0 without a result is refused by integration.
  - Invariants I-1 to I-6.
  - Persisted state survives a restart with no reinterpretation.
  - Reconciliation never upgrades a status.
  - The kill windows: between exit and drain, between drain and save, and
    between save and lease release.
- **Recovery.** Cover stale and orphan states, the G8 regression, resume from
  the store, cancel before and during a run, and retry eligibility.
- **Compatibility.**
  - Pinned revisions with `runtime-default` effort are dispatchable; those
    with explicit effort are refused for new dispatch and never rewritten.
  - Existing Executions are untouched.
  - Historical graphs are readable but refused for mutation.
  - Graph and attempt formats stay additive.
- **Credentials and Evidence.** Cover environment construction (tokens
  dropped, login variables kept), the absence of credential or login
  material in packets, records, evidence, logs and Evidence, and redaction.

### 9.2 Test levels

- **Unit tests** for each rule above.
- **Application integration** for `stageexecution` and capability proof,
  using fake runners and temporary stores.
- **CLI blackbox** with test-built fake `codex`/`claude` executables that
  read the stdin packet and script outcomes, including the probe canaries
  and sentinel writes. These run against the installed binary layout.
- **Skill contract and routing** in `internal/cli/domain_skill_routing_test.go`,
  `skill_catalog_contract_test.go`, `cmd/lingo/skill_inspect_test.go` and
  `internal/codexruntime/*skills*_test.go`, with Codex and Claude content
  parity and updated receipt fixtures.
- **Existing suites** for `executiongraph`, `graphapplication` and `local`
  must stay green.

Every deterministic or fake-executable result is classified as
**synthetic**. None of it is operational capability proof or acceptance.

### 9.3 Repository checks

- `go test ./...`
- `./scripts/check-go-quality.sh`
- `./scripts/test-check-project-domain.sh`
- `./scripts/test-codex-skills.sh`
- `python3 scripts/check-adr-governance.py .`
- `./scripts/validate-repository.sh .`
- `./scripts/check-sensitive-files.sh --staged .`
- `git diff --check`

### 9.4 Bounded real Codex + Claude smoke (separately authorized)

The smoke has five preconditions:

1. IG-1: #306 supported authoring is delivered and used to configure a clean
   state. No hand-edited state.
2. Capability probes are separately authorized and recorded as `proven` for
   each child binding.
3. Both installed Runtimes are already authenticated by their vendors.
4. A human authorization names the exact envelope below.
5. Increments I2–I6 are complete.

| Field | Requirement |
|---|---|
| Envelope | `axiom` version/commit/binary SHA-256; scratch lab repository (never the Axiom repository); workflow digest; stage; children (Runtime, Profile, model, Profile effort); effects (`read`, `repository-write` in isolated worktrees, `process`); max attempts 1; timeouts; coordination budget; Evidence directory; approver and time |
| Auth mode | Subscription (`codex login status`, `claude auth status --json` → `subscription_observed`); no API keys or environment overrides |
| Versions / identity | `--version`, executable digests, mode digests, capability evidence digests, confinement level per child |
| Topology | Codex and Claude children are independent and concurrent. Codex publishes a `question_request`. Claude answers with a `question_reference`. Both publish `result`. The integration owner consumes both results as compiled inputs and the stage integrates locally |
| CR-001 check | Persisted statuses and reasons, dependents' readiness, and integration eligibility are recorded before and after the initiating session exits |
| Initiating surface | Start from one Runtime's skill; inspect and integrate from the other's (AC-2) |
| Artifacts | Bounded: graph JSON, streams, attempt metadata, capability evidence, validator results, redacted excerpts. No credentials, account IDs or transcripts |
| Classification | `engineering_smoke_real`. It is not R-2, not R-3, and not #278 acceptance |

If a precondition is unmet, the smoke is not run and the reason is recorded.

## 10. Dependencies and integration gates

**Consumed, not authored by #276:** the local Runtime/Profile configuration
and its authoring, onboarding and templates (#306/PR #310), the
`runtimeapplication` policy, the #272 preflight and the `runtimeprofile`
observation contract (unchanged).

| Gate | Owner | #276 impact | Mitigation |
|---|---|---|---|
| IG-1 Supported local Profile authoring and onboarding | #306 / PR #310 | Real installed dispatch, probes and the smoke need a clean, supported configuration | Deterministic delivery uses test fixtures only (never product paths); no temporary configuration path |
| IG-2 Credential handling | #306 (D2: vendor-managed only) | No credential resolver; references refused | Existing refusal |
| IG-3 Profile invocation parameters | #306 | Adapter constants only | Compose through the `CommandProfile` builder after review |
| IG-5 Profile-owned effort and the Spec 007 / `default-sdd` effort amendment | PR #310 (maintainer decision, mandatory change 2) | #276 reads effort from the Profile; agent-effort retirement is not authored here | §3.10 consumption rule; Tasks that depend on effort wait for that amendment's acceptance |
| IG-6 `axiom-runtime` skill | PR #310 (D4) | Capability show/probe/invalidate skill route | PD-14 |

IG-4 (capability proof) is **no longer a gate**. It is a #276 deliverable
(§3.9).

Likely file conflicts with PR #310:

- `internal/cli/commands.go` and `internal/cli/cli.go` flag sets;
- `internal/cli/argument_requirements.go` and
  `internal/cli/help_requirements.go`;
- `internal/cli/skill_inspect.go`;
- `docs/commands.md`;
- `docs/decisions/README.md` (ADR-0023 and ADR-0024 entries);
- `docs/specifications/README.md`;
- `internal/codexruntime/testdata/published-*`;
- `cmd/lingo/runtime_policy.go`;
- `internal/runtimeprofile/*` (#276 only reads it).

Mitigation: put new commands in new files, make shared tables append-only,
rebase at phase boundaries, and coordinate the order of fixture
regeneration.

## 11. Implementation strategy (increments)

| # | Outcome | Components | Depends on | Validation | Risks |
|---|---|---|---|---|---|
| I1 | Contract gate: amendment #276 and ADR-0024 accepted or revised; PDs decided; IG-5 status known | docs | Plan approval | doc checks | scope creep |
| I2 | Safety baseline: G8; stdin; process groups; lease and epoch; additive attempt fields; `ClassifyAttempt`; publication builder | `executiongraph`, `graphapplication`, `local` | I1 | unit, CR-001 invariants, format compatibility | format compatibility |
| I3 | Capability proof: interface observation, evidence store, probe (fake executables), observer integration, revalidation, show/probe/invalidate | `capabilityproof` (new), `runtimeadapter`, `local`, `cmd/lingo` | I1, I2 | capability matrix | vendor interface drift |
| I4 | Adapter: argv allowlist, confinement variants, packet, environment allowlist, mandatory subscription auth, Profile-owned effort | `runtimeadapter`, `graphapplication` | I2, I3, IG-5 for effort | unit, controlled Runtime | flag drift |
| I5 | `stageexecution`: preview/apply, authority record, supervisor, graph and single paths, status, resume | new component, composition | I2–I4 | application integration | supervisor portability |
| I6 | Coordination: inbox, publish, ingestion, drain, list/show/wait | `coordination`, `local`, service | I5 | nine kinds, negative tests | ingestion races |
| — | **Real capability probes and smoke** (§9.4), only after IG-1 and separate authorization | — | I3–I6 | real Evidence | usage limits; advisory confinement |
| I7 | Cancel, reconcile, retry, integrate; validator policy (PD-7) | service, `gitworkspace` | I5, I6 | integration, blackbox | workspace disposition |
| I8 | CLI and skill routes (`axiom-work-item`; `axiom-runtime` per PD-14); docs; receipts | `internal/cli`, `internal/codexruntime`, docs | I3–I7, IG-6 | blackbox, skill contract | #310 conflicts |
| I9 | Implementation record, Evidence, final validation, review | docs | all | full checks | gate timing |

## 12. Risks and decisions

### 12.1 Resolved

| ID | Decision | Source |
|---|---|---|
| D-1 | Capability proof needed for stage execution (IG-4/G-3) is in #276 and PR #311. No new Issue and no transfer to #272/#306 | [Maintainer, PR #311](https://github.com/rgomids/axiom/pull/311#issuecomment-6108112930) |
| #306 D2/D3 (consumed) | Vendor-managed authentication; effort is owned by the local Model Profile with no Workflow override; complexity is independent | [Maintainer, PR #310](https://github.com/rgomids/axiom/pull/310#issuecomment-6108025068) (conceptual direction; #310 artifacts not yet accepted) |

### 12.2 Pending decisions (none approved; each blocks the Tasks that depend on it)

| ID | Decision | Alternatives | Recommendation | Trade-offs / risks | If approved / if rejected | Blocks Phase 2? |
|---|---|---|---|---|---|---|
| PD-1 | Accept the amendment #276 surface: `stage status/reconcile/integrate`, `coordination show`, `runtime <id> capability show/probe/invalidate`, categories | Fold into existing commands as modes | Accept separate leaves | More commands, clearer authority per mutation | Approve: CLI tasks proceed. Reject: redesign §5 | Yes (CLI/skill tasks) |
| PD-2 | Detached bounded supervisor per run | Foreground step capped below the host tool limit | Supervisor (ADR-0024) | Background lifecycle and portability vs. surviving tool timeouts | Approve: I5 as planned. Reject: terminal-only long runs; skill-initiated runs limited to short children | Yes (I5) |
| PD-3 | Dispatch-authority record committed with graph/ledger, first-attempt scope | New `workflow fact` kind | Record via preview/apply | Record format (additive) vs. new approval mechanism | Approve: G-5 closed this way. Reject: fact-kind design needed | Yes (I5) |
| PD-4 | Attempt inbox + supervisor ingestion; truthful confinement (`advisory` for Claude unless proven) | Children write canonical store directly; or disable Claude children until OS confinement is proven | Inbox + truthful level, human decides per dispatch | Same-user forgery remains (ADR-0005); Claude writes not OS-confined | Approve: Claude usable with disclosed advisory confinement. Reject (disable variant): Claude dispatch blocked until proof, weakening AC-1/AC-3 | Yes (I6) |
| PD-5 | Subscription auth mandatory for public dispatch and probes | Opt-in | Mandatory | Excludes API-key users (consistent with #306 D2) | Approve: fail closed. Reject: auth modes must be specified | Yes (I4) |
| PD-6 | Implement `stage cancel` per Spec 007/ADR-0009 | Explicit unsupported | Implement | Process-group stop complexity | Approve: cancel tasks. Reject: unsupported with an existing category | No (isolated task) |
| PD-7 | Integration validators from a separate, reviewed validator-policy artifact with its own authority | Commands in the Plan document; builtin/human-review validators only in #276 | Separate artifact | One more authority step vs. agent-authored commands | Approve: amendment covers it. Reject (builtin-only): repository tests cannot gate integration | Yes (I7) |
| PD-8 | Retry after effects requires explicit, authorized workspace disposition | Retry only on a clean workspace | Explicit disposition | Friction vs. no silent resets | Approve/reject changes retry tasks only | No |
| PD-9 | CR-001 classifier at terminal save into existing statuses (`blocked`, `failed`) with persisted reason and record refs; retry allowed from `blocked` | Separate qualification field leaving `succeeded`; status-only rendering (rejected by CR-001) | Classifier into existing statuses | `succeeded` now means "contract fulfilled", not "exit 0" (exit code retained) | Approve: §3.5 as written. Reject: qualification-field variant with scheduler and integration changes | Yes (I2) |
| PD-10 | Append-only reconciliation facts stored next to graph attempts | Stage-ledger records | Graph-side | ADR-0009 format extension | Format choice only | Yes (I7) |
| PD-11 | Single-agent stages dispatch through the extended sequential executor | Non-dispatching single path | Dispatch | Larger scope; needed because `default-sdd` is all single-agent | Reject: MVP default journey stays manual | Yes (I5) |
| PD-12 | Vendor inference (`stage run`, `stage retry`, `capability probe`) requires `--authorize-external` in addition to `--authorize-local` | Local authority only | Both | Extra confirmation vs. truthful external-effect consent | Approve: preview names inference. Reject: local only | Yes (I3, I5) |
| PD-13 | Capability proof = interface observation (can only refute) + authorized behavioral probe (can prove), bound per §3.9, with a fixed maximum evidence age (proposed 30 days) | Interface observation as proof (prohibited by D-1 direction); a probe before every dispatch | Probe-based, reusable while fresh and binding-matched | Probe cost and inference consent vs. per-dispatch cost; exact max age | Approve: I3 as planned. Reject: alternative proof needs design | Yes (I3) |
| PD-14 | Capability commands live under `axiom runtime <id> capability`, owned by #276. The skill route is `axiom-runtime` once PR #310 delivers it; until then, CLI only, with `axiom-work-item` blockers naming the CLI next action | Route via `axiom-work-item`; wait for #310 | CLI now, `axiom-runtime` route later | Temporary CLI-only diagnostics; cross-PR skill coordination | Approve: no skill duplication. Reject: `axiom-work-item` gains capability routes | No (route-only) |
| PD-15 | #276 consumes Profile-owned effort; the Spec 007/`default-sdd` effort amendment is authored by PR #310 (IG-5); pinned explicit-effort revisions refused for new dispatch | #276 authors the Spec 007 effort amendment | #310 authors it, per its decision | Cross-PR sequencing | Approve: effort tasks wait for IG-5. Reject: #276 adds an effort amendment, coordinated with #310 | Yes (effort parts of I4) |

The Plan itself, [amendment #276](amendment-276-stage-dispatch.md) and
[ADR-0024](../../decisions/0024-public-stage-dispatch-and-capability-proof.md)
each need explicit human acceptance before Phase 2.

### 12.3 Risks

| Risk | Type | Mitigation |
|---|---|---|
| Claude write confinement is advisory | security | Truthful level in preview; probe sentinel; human decides; PD-4 alternative |
| Reads are not OS-confined for either Runtime | security | Stated in preview; no secrets in worktrees (#306 D2); environment allowlist |
| Vendor flags change across versions | cross-Runtime | Interface observation per version; mode digest in evidence; unknown versions fail closed |
| A probe proves behavior only for the canary, not every task | proof scope | Proof is scoped to the vocabulary definitions; evidence binds mode; revalidated |
| Probe cost and usage limits | operational | One bounded probe per binding/variant; reused while fresh |
| Same-user forgery of inbox or evidence files | security | ADR-0005 boundary; epoch fencing; evidence digest checked against binding |
| Supervisor portability | operational | Platform tests; fail closed |
| Additive formats break readers | architecture | Compatibility tests |
| PR #310 sequencing (IG-1, IG-5, IG-6) | dependency | Gates explicit; deterministic work proceeds |
| Merge conflicts with PR #310 | parallel development | §10 mitigations |

### 12.4 Open questions (resolved in Tasks unless escalated)

1. Whether Claude settings-based sandboxing can be enabled through an
   adapter-owned option without re-opening `--settings` as a selection path,
   and whether the probe can prove it.
2. The exact objects and recovery record for the multi-object commits.
3. The detachment primitive per platform.

### 12.5 Review disposition

- **Revision 1.** The independent review findings were incorporated: the
  denylist guard, concurrent ingestion, append-only reconciliation, the
  specified cancel, authority scope, separate validator authority, the
  supervisor model, the environment allowlist, the lease order, no
  pre-#306 smoke, stdin, reuse of existing categories, binary pinning, the
  AC-2 cross-surface case, the single-path PD and empty attempt IDs.
- **Revision 2, second independent review.** These findings were
  incorporated:
  - the `read` variant must still write its inbox;
  - explicit scheduler and integration changes for retry and eligibility;
  - a `cancelled` classifier row;
  - per-attempt record ordering, an enumerated outcome and a per-attempt
    capacity reserve;
  - integration bound to the qualifying record;
  - where implicit and model-scoped capabilities are checked;
  - confinement measured by a deterministic canary, with the observed
    writable set and network posture;
  - advisory prohibitions stated as policy rather than enforcement;
  - the graph marker;
  - per-attempt saves;
  - a closed environment list and a complete revalidation tuple;
  - no silent effort fallback.
- **Revision 2.** The maintainer REQUEST CHANGES was addressed as follows:
  - D-1: IG-4 is now a #276 deliverable (§3.9).
  - Security boundary and preview rights (§3.3, §3.6, §7).
  - Reconciliation with #306 (§3.10, §10).
  - CR-001: persisted classification and invariants (§3.5).
  - The ADR collision: #276 now uses ADR-0024.
  - Validation coverage (§9.1).
  - Delivery metadata in the PR body.

## 13. Acceptance traceability

| #276 acceptance criterion | Planned components | Validation | Expected Evidence |
|---|---|---|---|
| AC-1 Installed path starts configured Codex+Claude children without an acceptance-only runner or internal-state edits | `stage run`, `stageexecution`, supervisor, adapter, capability proof (§3.9) | CLI blackbox with fake executables against the installed layout | Synthetic report. **Operational** smoke only after IG-1 and authorized probes |
| AC-2 Either Runtime initiates the same workflow; restart/status/resume keep identity | Shared skill content, `stage status`, `GraphStore.Load`, lease/epoch, persisted classification | Parity and restart tests; smoke crosses surfaces | Parity report; stable IDs |
| AC-3 Real cross-Runtime Q/A consumed by orchestrator; nine kinds, forged lineage, stale revision, authority escalation, capacity | Ingestion, cross-stream references, classifier (§3.5) | Unit/integration for all kinds and negatives; smoke Q/A | Test report; smoke records with digests |
| AC-4 Overlap only when allowed; conflicts serialize/block; integration and roll-up preserve validation/authority | Scheduler, concurrency check, persisted eligibility, `stage integrate`, PD-7 | Scheduling regressions, CR-001 tests, integration tests | Regression results; smoke overlap timestamps |
| AC-5 Denied, failed, interrupted, unknown and recovery-required are distinguishable; reconcile before retry | G8 fix, classifier reasons, stale/orphan, cancel/reconcile/retry | Per-outcome and kill-window tests | Outcome matrix report |
| AC-6 Early real smoke records auth mode, versions, graph, coordination and bounded artifacts; synthetic never operational | §9.4, capability evidence, classification fields | Evidence field check; human review | `engineering_smoke_real` Evidence, or a recorded not-run reason |
| (D-1) Capabilities needed for real agentic development are proven, not assumed | §3.9 interface observation, probe, evidence store, revalidation, fail-closed | Capability matrix tests; authorized real probes | Evidence records per binding; synthetic vs. real classification |

Spec 007 AC-009 is covered by AC-2, AC-3 and AC-5.

---

# Part II — Tasks

**Not yet authorized.** Tasks will be added to this file in Phase 2, on the
same branch and PR, only after human approval of this Plan, amendment #276,
ADR-0024 and the pending decisions in §12.2.
