# Plan and Tasks — #276 run and coordinate stage agents through CLI and Runtime skills

## Status

**Phase 1 (Plan) — Proposed, pending human review.** This document is the
Plan for [#276](https://github.com/rgomids/axiom/issues/276) / Linear AXM-8.
It does not authorize Tasks, implementation, Runtime dispatch, real
inference, Provider effects, merge, release or human acceptance. Part II
(Tasks) is intentionally empty until this Plan and its decisions (§12) are
approved.

| Item | Value |
|---|---|
| Work Item | [#276](https://github.com/rgomids/axiom/issues/276) / Linear AXM-8 (operational status lives in Linear) |
| Parent | [Epic #15](https://github.com/rgomids/axiom/issues/15); Spec 007 requirement **WF-009**, acceptance **AC-009** ([spec.md:62](spec.md), [spec.md:547](spec.md)) |
| Contract consumed | [Spec 007](spec.md) (Accepted 2026-10-09), [amendment #303](amendment-303-workflow-skill.md) + [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md) (Accepted 2026-10-10), [ADR-0003](../../decisions/0003-lingo-as-axiom-local-control-plane.md), [ADR-0004](../../decisions/0004-portable-project-manifest.md), [ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md), [ADR-0007](../../decisions/0007-local-publication-and-recovery-protocol.md), [ADR-0008](../../decisions/0008-minimal-machine-local-execution-record.md), [ADR-0009](../../decisions/0009-parent-child-execution-graph.md), [ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md) |
| Contract changes proposed | Only through a bounded Spec 007 addendum and an ADR-0023 candidate (§12), each requiring separate human acceptance before dependent Tasks |
| Satisfied predecessors | #274 (binding), #275 (stage compiler, v0.15.0), #303 (skill surface, v0.16.0), #272 (auth preflight), #230 (resource lifecycle), #271 (contracts) — implementation baseline only |
| Parallel work | [#306](https://github.com/rgomids/axiom/issues/306) (G-2 local Runtime/Profile authoring) — boundary in §10 |
| Successors (not in scope) | #277 delivery/rework, #278 / AXM-12 R-2 native and R-3 human MVP acceptance |
| Code baseline | `main` `5d52327` (v0.16.0) |
| Lifecycle | One branch (`feat/276-stage-agent-execution`) and one Draft PR for Plan → Tasks → Implementation, each separately approved |
| Review | Independent adversarial architecture/security review of the first draft; findings incorporated (§12.4) |

Observed statements cite `file:line` on the baseline. Statements marked
*(inferred)* must be confirmed in the named Task. **D-n/PD-n** items need
explicit human approval (§12). **IG-n** items are integration gates outside
#276 (§10).

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
Spec 007 records the gap as "S9 runner is checkout-only" ([spec.md:528](spec.md)).

### 1.2 Expected user-visible behavior

From the installed `axiom` CLI, or from the `axiom-work-item` skill in
either Codex (`$axiom-work-item`) or Claude (`/axiom-work-item`), a developer
working on a Work Item whose Execution is bound to an immutable workflow
revision can:

1. preview dispatch of a planned stage (graph or single-agent), reviewing
   the exact proposal, Runtime/Profile resolutions, effects, workspaces,
   confinement level and validators;
2. confirm it once. The confirmation is persisted, revision- and
   digest-bound, together with the graph and correlation, **before** any
   process starts. It covers the first attempt of every child of that graph
   revision and nothing else;
3. let Axiom dispatch ready children (Codex and/or Claude) concurrently only
   where allowed, without the initiating Runtime session having to stay
   alive;
4. observe parent/child progress, blockers, coordination records and
   dependency state from either Runtime or the CLI, including after the
   initiating session ends;
5. cancel (prevent new dispatch and request bounded stop), reconcile
   uncertain attempts before any retry, retry explicitly, and request the
   separately authorized integration step;
6. receive truthful `denied_authority`, `validation_failure`, `interrupted`,
   `failure/recovery_required` and `partial` outcomes, never invented
   success or observed stop.

Child agents exchange only bounded, typed, lineage-checked records (question
/answer, contract proposal/acceptance, blocker, dependency resolution,
artifact publication, progress, result).

### 1.3 Included capabilities

- Public application service composing the existing graph, scheduler,
  workspace, integration, coordination and Runtime-policy services for one
  Work Item-bound Execution stage.
- CLI operations under the existing `axiom workflow` Execution family
  (Spec 007 table, [spec.md:418-419](spec.md)) and thin `axiom-work-item`
  skill routes.
- Child-side coordination publish/read with lineage, revision, scope,
  provenance and capacity enforcement at the entrypoint; deterministic
  orchestrator consumption.
- Durable dispatch authority for graph **and** single-agent paths
  (amendment gap **G-5**, [amendment:323](amendment-303-workflow-skill.md)).
- Cancellation per Spec 007 semantics, interruption, stale/orphan detection,
  append-only reconciliation and explicit retry.
- Feature help/result metadata, docs, deterministic tests and an early
  bounded real two-Runtime smoke procedure (executed only under separate
  authorization and after its integration gates).

### 1.4 Non-goals

UI/dashboard; new control plane, long-lived daemon or broker; general
distributed workflow engine; HTTP/MCP server; API-billing prerequisite;
automatic fallback, merge, push, PR or release (#277); Runtime/Profile
authoring or credential resolvers (#306); Project policy editing (#305);
authoritative capability/effort observation beyond the existing boundary
(IG-4); new global Doctor (#231); #241 CI/CD hardening; native
conversational R-2 and human R-3 acceptance (#278).

### 1.5 Relationship to the MVP journey

Spec 007 dependency order is `#273 → (#274, #275) → #276 → #277 → #278`
([spec.md:561-562](spec.md)). #276 turns the bound Execution and the stage
proposal into executed, coordinated, integrated stage outcomes. It produces
technical Evidence (deterministic/synthetic plus one early real engineering
smoke). It does **not** produce R-2/R-3 acceptance; those remain with #278.

## 2. Current-state assessment (observed)

### 2.1 Reusable components

| Area | Component | Evidence |
|---|---|---|
| Binding | `workflow start` creates an Execution with immutable `WorkflowBinding`; `workflow status/resume` | [execution_target.go:54-112](../../../cmd/lingo/execution_target.go), `internal/workflow/service.go:281,460` |
| Stage compilation | `workflowcompiler.Compile` → `ExecutionKind` `graph` with `*executiongraph.Proposal`, or `single`; `PlanStage` → `StagePlan` (refs, resolutions, validators, gates, blockers) | `internal/workflowcompiler/compiler.go:145`, `stage_plan.go:27-117` |
| Public plan op | `workflow stage plan` (read-only, digest-bound, re-checks Execution revision/binding) | [workflow_stage_plan.go:43-112](../../../cmd/lingo/workflow_stage_plan.go) |
| Graph model | Parent/children, envelopes, attempts; statuses not-started/running/blocked/succeeded/failed/cancelled/skipped/unknown; `AmbiguousEffect`, `CancellationSeen`; `AuthorityReferences map[string]string` | `internal/executiongraph/graph.go:29-100` |
| Graph publication | `GraphService.Publish(PublicationRequest)` validates proposal, authority subset, resolutions, workspaces; create + reload + digest compare | `internal/executiongraph/graph.go:126-209` |
| Planner rules | DAG, one integration+validation owner, transitive dependency, unsafe overlap, authority subset, limits (32 nodes, 1 MiB, 1–10 attempts, ≤24h) | `internal/executiongraph/planner.go:17-23,73-93,117-170` |
| Scheduler | Ready set, non-conflicting batch, **attempt persisted as `running` before process start**, terminal save after `wg.Wait`, explicit retry list, unknown/ambiguous block retry, `CancelRequested` blocks undispatched children | `internal/executiongraph/scheduler.go:94-208,210-262` |
| Integration | Digest-bound preview, `IntegrationAuthority`, single persisted attempt, optional waivers | `internal/executiongraph/integration.go:215,280,332` |
| Composition | `graphapplication.LocalService`: PrepareWorkspaces, DispatchReady, PreviewIntegration, ExecuteIntegration, BuildEvidence; policy guard re-checks preview, executable digest, credential reference and subscription auth per dispatch | `internal/graphapplication/local.go:49-150`, `runtime_policy.go:87-161` |
| Runtime adapters | Codex `exec --model`, Claude `--print --model`; effort only when capability-proven; absolute executable; no shell | `internal/runtimeadapter/adapters.go:92-150`, `scheduler.go:289` |
| Auth preflight | `AuthPreflight.Check` per child invocation; `authentication_blocked`; credential references refused as `credential_reference_configured` in subscription mode | `internal/runtimeadapter/authpreflight.go:162`; `runtime_policy.go:117`; [#272 record](../004-mvp-v1-baseline/issue-272-cli-subscription-auth-preflight.md) |
| Runtime policy | `runtimeapplication.Service` Preview/Check over Project allowlist ∩ local `runtimeprofile.Configuration` ∩ observation | `cmd/lingo/runtime_policy.go:80-104`, `internal/runtimeprofile/runtimeprofile.go:41-70` |
| Coordination | Nine kinds, per-kind fields, hash-chained per-child streams, credential/chat-marker rejection, 64 records/stream, 64 KiB/record, `AcceptanceVerifier` | `internal/coordination/coordination.go:42-52,78,115,209,301-347` |
| Stores | `local.GraphStore` (CAS `Save`, `ErrConflict`), `local.CoordinationStore` (private dirs, lock, chain re-check) | `internal/local/graph_store.go`, `internal/local/coordination_store.go:97` |
| Skills | One embedded set for Codex and Claude; `axiom-work-item` owns run/status/plan | `internal/codexruntime/runtime.go:15`; ADR-0022:58-67 |
| Result protocol | Canonical completion envelope + operation payloads; preview → `--preview-digest` + `--authorize-local` | `internal/cli/completion.go:26-33,282-289`; `internal/cli/argument_requirements.go:36-41` |

### 2.2 Gaps for #276

| # | Gap | Evidence | Disposition |
|---|---|---|---|
| G1 | No production path from `StagePlan`/compile `Result` to `GraphService.Publish` (`PublicationRequest` builder: workspaces, authorities, resolutions); `StagePlan` keeps only refs, so publication must recompile and compare digests | `stage_plan.go:94-109`; runner does it by hand at `s9-graph-runner.go:309-323` | Extend (application) |
| G2 | `LocalService` is not wired into `cmd/lingo`/`internal/cli`; it holds the graph in memory and has no load/resume constructor | `local.go:49-150`; no imports in `cmd/lingo` | Extend |
| G3 | No public `stage run/status/retry/cancel/reconcile/integrate` or `coordination list/publish` operations | `internal/cli/commands.go:26-83` | New (CLI + skill) |
| G4 | No task delivery to children: `Invocation` and `OSProcessRunner` have no stdin; the runner appends the prompt to `Arguments` | `scheduler.go:22-31,361-366`; `s9-graph-runner.go:249` | Extend (`executiongraph` + adapter) |
| G5 | No confinement derived from effects; the argument guard is a **denylist** of selection options and does not reject permission/sandbox bypass flags | `runtime_policy.go:207-232`; runner flags in `evidence-s9-rc2/envelopes/D-graph-spec-run03.json` | Extend (adapter), PD-4 |
| G6 | `CommandProfile` has no production source; executable is observed via `PATH`; no `CredentialResolver` implementation | `cmd/lingo/runtime_policy.go:87-104`; `adapters.go:56-75` | Derive from existing config + Axiom constants (§3.5) |
| G7 | Coordination store exposes only `Latest`/`Publish`; no stream read; no child-side entrypoint; `validInput` accepts an empty `AttemptID`; records carry no scope/authority fields | `coordination.go:115,301-307` | Extend |
| G8 | **Defect:** `readyChildren` does not block a child whose last attempt is still `running`; with retry authorized after a crash it would redispatch | `scheduler.go:237-255` (confirmed by two independent reads) | Fix + regression |
| G9 | No detection of stale/orphaned `running` attempts and no reconciliation; no graph resume after process death; no process-group control, so killing the caller leaves children running | grep "reconcil" only in `executiongraph`; `scheduler.go:361-381` | New (application + runner) |
| G10 | Timeouts/interrupts always yield `unknown` (`Stopped: false` hard-coded) | `scheduler.go:378-381` | Keep truthful; document |
| G11 | Validator `policyRef` resolves only `builtin-sdd-v1`; repository test commands must be "reviewable validator policy artifacts under separate authority"; no contract maps graph integration validators to `gitworkspace.ValidationCommand` | `internal/local/workflow_references.go:93-116`; [spec.md:176-180](spec.md) | PD-7 |
| G12 | Stage-plan approval is conversational/Plan-document only; G-5 requires durable approval binding before real dispatch | amendment:323; [issue-275-implementation.md:310-312](issue-275-implementation.md) | PD-3 |
| G13 | Subscription auth is opt-in (`SubscriptionAuth`); making it mandatory is pending a human decision | #272 record:124-127 | PD-5 |
| G14 | Children inherit the full parent environment when `Env` is empty (`os.Environ()`), conflicting with ADR-0009 minimum credential capability | `scheduler.go:336-357` | Extend (§3.5) |
| G15 | No feature help/result metadata, docs or skill routes for dispatch; skills say dispatch is unavailable | `internal/cli/skill_inspect.go:60-215`; `axiom-work-item/SKILL.md` | New |
| G16 | Concurrency cap is enforced only as compile-time dependency edges, not re-checked at dispatch | `compiler.go:397`; `scheduler.go:163-185` | Regression test + defensive check |
| G17 | The production observer proves only `axiom-skills`; stages needing other capabilities (e.g. built-in `read`) or explicit effort are refused on a real machine | [issue-275-implementation.md:303-309](issue-275-implementation.md); `docs/commands.md:603-610` | **IG-4** (outside #276) |

## 3. Proposed architecture

### 3.1 Containers and boundaries (C4 level 2, delta only)

The existing container view in `docs/architecture/` is unchanged: one local
`axiom` (Lingo) binary over machine-local state. #276 adds no new container
or service; it adds one application component, a bounded supervisor mode of
the same binary (PD-2) and CLI/skill entrypoints.

```text
 Codex session ──$axiom-work-item──┐
 Claude session ─/axiom-work-item──┤      (initiating surfaces: operator only)
 Terminal ───────── axiom ─────────┤
                                   ▼
             internal/cli  (parse, help, render; no domain rules)
                                   ▼
        cmd/lingo composition ──► stageexecution.Service      (NEW application component)
                                   │   ├─ workflow.Service          binding, gates, ledger          existing
                                   │   ├─ workflowcompiler          recompile + digest compare      existing
                                   │   ├─ runtimeapplication        policy Preview/Check            existing
                                   │   ├─ executiongraph            Publish, Scheduler, Integration existing (+G4, G8)
                                   │   ├─ graphapplication          LocalService, policy guard      extended (load/resume)
                                   │   ├─ coordination              validate, chain                 extended (stream read)
                                   │   └─ runtimeadapter            argv, confinement, auth         extended
                                   ▼
             internal/local stores: workflow, graph, coordination, runtime profile (read-only here)
                                   ▲
   axiom stage supervisor (same binary, detached, one per run, exits when done; PD-2)
        └─ spawns children in their own process group ──► codex exec / claude --print
                                                            stdin: bounded task packet
                                                            writes: workspace + own attempt inbox
                                                            calls: axiom workflow coordination publish|list
```

`stageexecution` *(name provisional; Tasks decide the package)* owns
orchestration **composition**, not rules: every rule stays in the package
that already owns it. CLI handlers and skills pass request DTOs and render
results only.

### 3.2 Process model (PD-2)

Children of real stages run for minutes to hours (per-child timeout up to
24h, `planner.go:17-23`). The initiating Runtime runs `axiom` inside a tool
call with its own, much shorter limit (Claude Code shell calls default to 2
minutes and cap at 10 *(documented vendor behavior; verify per observed
version in Tasks)*; Codex limits *unverified*). A foreground call that waits
for children would be killed by the host and orphan every child.

Recommended: **detached bounded supervisor (same binary), no daemon.**

- `stage run` apply, after committing the graph and authority (§3.3),
  starts one detached `axiom` supervisor process for that graph run (hidden
  internal entrypoint, not a public command), records its correlation
  (PID, process start identity, lease epoch) and returns immediately with
  `ExecutionRef`, graph revision and `next`.
- The supervisor acquires the run lease, dispatches ready non-conflicting
  batches through the existing scheduler, ingests coordination (§3.4),
  persists terminal attempts and exits when nothing more is dispatchable
  under the existing authority, or on cancellation. It never initiates
  retries, reconciliation, integration or any effect needing new authority.
- Children run in their own process group (POSIX) / job object (Windows),
  so cancellation and supervisor-death handling can address the group;
  observed stop is still not claimed unless the group is confirmed gone.
- All truth lives in the stores. Any later `axiom` call from any Runtime or
  terminal reads it; the initiating session may exit at any time.

Alternative A (foreground bounded step inside the caller's tool call) is
simpler but infeasible for realistic child durations; it stays viable only
for terminal use and is the fallback if PD-2 is rejected, with per-child
timeout capped below the host limit.

The supervisor adds a background process lifecycle and a new lock (§3.6),
so it belongs in the **ADR-0023 candidate** (§12).

### 3.3 Dispatch authority (PD-3, closes G-5)

`stage run` follows the standard preview/apply pattern:

- **Preview** (no authority flags): recompiles the stage under the bound
  revision, compares with the reviewed `StagePlan` digest and
  `planDocumentDigest`, re-observes Runtime policy, checks gates, resolves
  workspaces and confinement and returns a **dispatch preview** whose digest
  binds: `ExecutionRef`, `workflowRef`, `stageId`, `planDocumentDigest`,
  `plan.digest`, graph proposal digest (graph path), every `Resolution`
  (Runtime, Profile, configuration/observation revisions, effective effort,
  executable digest), effects, workspaces, confinement level per child,
  validator policy reference (PD-7), attempt limits, parent authority
  ceiling and the digest of the `axiom` binary given to children.
- **Apply** (`--preview-digest` + `--authorize-local` + `--expected-revision`):
  re-observes everything, refuses drift with `denied_authority/stale_preview`,
  then commits a **dispatch-authority record** {preview digest, scope
  "first attempt of each child of graph revision R" or "the single attempt",
  actor kind `human-operator`, time} and the graph (graph path) or the stage
  ledger attempt binding (single path). Graph children reference the record
  through the existing `AuthorityReferences` string per node; the record
  itself is a bounded machine-local artifact (additive format, decided in
  Tasks and the ADR-0023 candidate).
- The confirmation covers **only** the first attempts in that scope. Retry,
  reconcile, cancel and integrate each require their own preview and
  `--authorize-local`. Continuing dispatch of already-authorized first
  attempts (by the supervisor) needs no new confirmation but re-checks
  drift and stops on any.
- Workflow gates declared `before` the stage must already be satisfied
  through the existing `workflow fact` path; dispatch authority never
  substitutes for them, and agent records cannot commit authority
  ([spec.md:205-208](spec.md)).
- Skills pass `--authorize-local` only after explicit human confirmation in
  the conversation, as for every existing mutation; a skill never
  re-authorizes on its own.

### 3.4 Coordination ingress and orchestrator consumption (PD-4, PD-9)

Children never write canonical Axiom state directly.

- Each running attempt gets a private **attempt inbox** under the state
  root. `axiom workflow coordination publish`, run inside a child, resolves
  its identity from its working directory (child workspace) plus attempt
  correlation in its environment, validates kind/fields/size with the
  existing `coordination` rules, requires a non-empty attempt ID, and writes
  a candidate only into its own inbox.
- The supervisor ingests candidates **concurrently while children run** and
  performs a **final drain of an attempt's inbox before persisting that
  attempt's terminal state** (hook between process exit and terminal
  `Save`). Ingestion re-validates lineage (parent, child, graph revision,
  attempt owned by the current lease epoch), chain revision/previous digest,
  capacity, and cross-stream references (`answer.question_reference`,
  `contract_acceptance.contract_reference` must name an existing record of
  the same parent/graph revision), then publishes through the existing
  `coordination.Service` and `local.CoordinationStore`. Rejected candidates
  become bounded diagnostics, never canonical records. Candidates from a
  previous lease epoch (orphans) are quarantined, not ingested.
- Reads (`coordination list/show`) read canonical streams only; a child may
  poll with a bounded `--wait` shorter than its own timeout. The task packet
  tells an asker to stop waiting, publish a `blocker`
  (`required_action: answer <record>`) and exit before its timeout, so an
  unanswered question becomes a blocked child, not an `unknown` attempt.
- **Orchestrator consumption (deterministic, PD-9):**
  1. a `question_request` is *resolved* when a valid `answer` from another
     child references it; unresolved questions appear in `stage status`;
  2. a child that exits after publishing a `blocker` is reported blocked
     with the record's `required_action`; its dependents stay
     `dependencies_unsatisfied`;
  3. a required child is integration-eligible only if its attempt
     succeeded **and** it published a `result` record correlated to that
     attempt; results feed the integration owner as compiled inputs
     (validated refs/digests, never raw text);
  4. `progress`, `artifact_publication`, `contract_*` and
     `dependency_resolution` are surfaced in status and Evidence with their
     correlation; they never mutate graph, authority or gates
     ([ADR-0009](../../decisions/0009-parent-child-execution-graph.md):104-113).
- Provenance is "attempt inbox of child X attempt Y, lease epoch E",
  recorded truthfully. Under the bounded local threat model
  ([ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md))
  a same-user process can still forge files; Axiom does not claim
  cryptographic identity. How far a child is actually confined depends on
  the Runtime (§3.5, PD-4).

### 3.5 Child invocation, confinement and environment (G4–G6, G14)

- **Command profile source:** Runtime ID, Profile, model and credential
  reference from the existing `runtimeprofile.Configuration`; executable
  from the existing `PATH` observation, pinned by `executableDigest`.
  `Arguments` and `OutputMax` are **Axiom constants per Runtime** owned by
  the adapter, not operator configuration. No configuration field or file
  is added; #306 may later supply reviewed invocation parameters through the
  same `CommandProfile` builder (IG-3).
- **Argument allowlist:** the adapter builds argv only from its constant
  allowlist plus confinement flags it appends last; anything else is
  refused. Permission or sandbox bypass options of either Runtime are never
  emitted and are explicitly rejected in tests.
- **Confinement level (PD-4):** derived from the child's effects:
  `read` → read-only; `repository-write` → write to its worktree; every
  child that coordinates also needs `process` (to run `axiom`) and write to
  its own inbox, made explicit in the envelope. Codex: `--sandbox
  workspace-write` + `--add-dir <inbox>` (OS-enforced sandbox). Claude: the
  equivalent permission flags restrict tool use but are **not** an OS
  sandbox; unless an enforceable sandbox is observed for the installed
  version, a Claude child is recorded with confinement `advisory`, shown in
  the dispatch preview so the human decides with that knowledge. Exact
  flags per observed version are confirmed in Tasks *(inferred)*.
- **Task packet on stdin:** role, responsibilities, instructions, declared
  inputs (validated predecessor result refs/digests), expected outputs,
  criteria, allowed paths/effects, coordination protocol and limits, the
  absolute path and digest of the `axiom` binary to call, and prohibitions
  (no commits, push, Provider calls, credential access, nested agents).
  Derived only from the compiled `ExecutionInput` and envelope; its digest
  is recorded on the attempt. No raw chat is replayed. Requires adding
  stdin to `Invocation`/`OSProcessRunner` (`executiongraph`).
- **Environment allowlist (ADR-0009 minimum credential capability):** the
  child environment is constructed, never inherited wholesale: a fixed
  allowlist (`HOME`, `USER`, `LOGNAME`, `PATH`, `SHELL`, `TMPDIR`, locale,
  and Runtime login-location variables such as `CODEX_HOME` and
  `CLAUDE_CONFIG_DIR` when present) plus Axiom correlation variables.
  Ambient tokens such as `GH_TOKEN` are dropped. The auth preflight runs on
  exactly this environment. Tests cover both failure modes (leaked token;
  missing login variables breaking subscription auth).

### 3.6 Leases, locks and multi-object commits

- The **run lease** is a non-blocking exclusive OS lock per graph, held by
  the supervisor, with a monotonically increasing **lease epoch** recorded
  on each attempt it starts. It is first (broadest) in the ADR-0007 total
  lock order, before graph CAS and coordination stream locks; `stage
  status` probes it non-blockingly and never waits.
- A `running` attempt whose epoch's lease is free is **stale**. If its
  recorded process group is still alive it is an **orphan**: status reports
  it, ingestion quarantines its inbox, and retry stays blocked until the
  operator stops it and reconciles.
- Committing graph + dispatch authority, and later recording a stage
  outcome in the Execution ledger, are multi-object commits; they follow
  ADR-0007 single-commit-point and recovery-record rules (exact objects
  decided in Tasks and the ADR-0023 candidate).

### 3.7 Reconciliation, retry and cancellation

- **Reconciliation is append-only.** `stage reconcile` preview gathers
  observations (lease state, process liveness, workspace status vs base,
  canonical records, inbox quarantine); apply records a reconciliation fact
  {attempt, observations digest, classification, actor, time} next to the
  attempt. It never rewrites an attempt's status: `unknown` stays `unknown`,
  ambiguous effects stay recorded, and inference spend is never "no
  effect". The scheduler reads the fact to unblock an explicit retry (PD-10
  decides where the fact lives).
- **Retry** (preview/apply) creates a new attempt under the same child only
  with reconciled prior attempts, remaining `MaximumAttempts` and explicit
  authority; workspace disposition after effects is explicit (PD-8).
  Exhaustion needs a new stage plan and authority ([spec.md:196-202](spec.md)).
- **Cancellation** follows Spec 007 and ADR-0009 (PD-6): `stage cancel`
  (preview/apply) persists a cancel request on the graph (additive), which
  makes the scheduler stop new dispatch (existing `CancelRequested` →
  `cancelled_before_dispatch`) and makes the supervisor request a bounded
  stop of running children's process groups. Attempts end `cancelled` only
  when stop is confirmed (group gone and no ambiguous effect); otherwise
  `unknown` with `CancellationSeen`
  ([ADR-0009](../../decisions/0009-parent-child-execution-graph.md):117-119).
  The result reports observed facts, never assumed stop. The existing
  Execution-level `workflow cancel` stays `invalid_command`.

### 3.8 Single-agent stages (PD-11)

`executionKind: single` uses the existing sequential path without a graph
parent ([spec.md:347-350](spec.md); ADR-0020:52-55). Recommended: #276
exposes the same `stage run/status/cancel/reconcile/retry` operations for
it, binding one attempt in the stage ledger; the sequential executor is
extended to dispatch that attempt through the same adapter, packet,
confinement and supervisor. No graph, integration child or second process
is fabricated. Alternative: keep single-agent stages non-dispatching in
#276 (smaller, but the default SDD workflow's stages are single-agent, so
the MVP journey would stay manual).

## 4. Execution lifecycle

```text
Work Item ─ workflow start (existing, #274) ─► Execution + immutable WorkflowBinding
   └─ workflow stage plan (existing, #275) ─► StagePlan digest (no authority)
        └─ workflow stage run [preview] ─► dispatch preview digest (+confinement levels)
             └─ workflow stage run [apply + authorize-local]
                  ├─ re-check Execution revision, binding digest, gates, policy, observation, auth
                  ├─ commit dispatch-authority record + graph (or ledger attempt)   ← commit point
                  ├─ prepare isolated worktrees for writers
                  └─ start detached supervisor; return ExecutionRef + next
                         supervisor: lease(epoch) → persist attempt `running` → spawn group
                                     → ingest inbox → final drain → persist terminal → release
        └─ workflow stage status ─► children, attempts, blockers, records, stale/orphan, next
        └─ workflow coordination list/show ─► canonical records
        └─ workflow stage cancel     [preview/apply] ─► stop new dispatch; bounded stop request
        └─ workflow stage reconcile  [preview/apply] ─► append reconciliation fact
        └─ workflow stage retry      [preview/apply] ─► new attempt under same child
        └─ workflow stage integrate  [preview/apply] ─► existing Integration preview/execute
             └─ validators (PD-7) → deterministic roll-up → stage outcome in Execution ledger
        └─ workflow status / advance (existing) ─► next stage or #277 delivery
```

| From | Event | To | Notes |
|---|---|---|---|
| planned | apply with drift | unchanged | `denied_authority/stale_preview`; nothing written |
| graph published | Runtime/auth/capability blocked | child `blocked` (`runtime_unresolvable`, `authentication_blocked`) | no attempt, no process |
| attempt `running` | exit 0 + correlated `result` record | `succeeded`, integration-eligible | output untrusted until integration validators pass |
| attempt `running` | exit 0 without `result` | `succeeded`, not integration-eligible | roll-up blocked (`result_missing`) |
| attempt `running` | exit after `blocker` | per exit code; child reported blocked | dependents stay blocked; `required_action` surfaced |
| attempt `running` | exit ≠ 0 | `failed` | dependents blocked; retry needs authority + capacity |
| attempt `running` | timeout | `unknown` | `Stopped` not observable (G10) |
| attempt `running` | cancel, group confirmed gone, no ambiguous effect | `cancelled` | otherwise `unknown` + `CancellationSeen` |
| attempt `running` | supervisor dies | stays `running`; lease free → **stale** (or **orphan** if group alive) | `failure/recovery_required`; G8 fix blocks redispatch |
| stale/unknown | reconcile apply | unchanged + reconciliation fact | never `succeeded` by inference |
| reconciled | retry apply | new attempt | explicit workspace disposition (PD-8) |
| all required eligible | integrate apply | integration attempt | single persisted attempt; validators; roll-up |
| missing/conflicting results | — | parent `blocked` | no roll-up success |

## 5. Public interface design

All operations live under the existing `axiom workflow` Execution family
and use existing selectors (`--project --repository --work-item --execution`),
`--expected-revision`, `--preview-digest`, `--authorize-local`, `--json`,
the canonical completion envelope and exit codes
([cli.go:23-27](../../../internal/cli/cli.go)). Flag spellings are final
only in Tasks.

| Operation | Kind | Mutates | Owning skill / mode | Spec 007 basis |
|---|---|---|---|---|
| `workflow start/status/resume/fact/advance` | existing | as today | `axiom-work-item` `run`/`status` | #274 |
| `workflow stage plan` | existing (next text updated) | no | `axiom-work-item` `plan` | #275 |
| `workflow stage run` | **new** (preview/apply) | yes (authority, graph, supervisor) | `axiom-work-item` `stage.run` | table row `stage run` |
| `workflow stage status` | **new** read | no | `axiom-work-item` `stage.status` | addendum (PD-1) |
| `workflow stage cancel` | **new** (preview/apply) | yes (cancel and stop request) | `axiom-work-item` `stage.cancel` | table row `stage cancel`; PD-6 |
| `workflow stage retry` | **new** (preview/apply) | yes | `axiom-work-item` `stage.retry` | table row `stage retry` |
| `workflow stage reconcile` | **new** (preview/apply) | yes (reconciliation facts only) | `axiom-work-item` `stage.reconcile` | addendum (PD-1) |
| `workflow stage integrate` | **new** (preview/apply) | yes (local integration effect) | `axiom-work-item` `stage.integrate` | addendum (PD-1) |
| `workflow coordination list/show` | **new** read (`--wait` bounded) | no | `axiom-work-item` `coordination.list` | table row `coordination list`; `show` by addendum |
| `workflow coordination publish` | **new**, child context only | own inbox only | none (used by child processes per packet) | table row `coordination publish` |
| supervisor entrypoint | **new**, hidden/internal | yes (as already authorized) | none | ADR-0023 candidate |
| `workflow cancel` | existing unsupported (`invalid_command`) | no | unchanged | #230 F-02 |

Rules:

- Read operations take no authority argument. Mutations require preview,
  `--expected-revision`, `--preview-digest` and `--authorize-local`.
- `coordination publish` outside a running child context (no matching
  workspace, attempt or lease epoch) fails closed with `denied_authority`,
  including from a human shell or the initiating session.
- Results use the existing refusal table categories
  ([spec.md:448-456](spec.md)) plus operation-specific categories added by
  the PD-1 addendum; no new completion statuses.
- Skills relay the default human rendering and read follow-up values from
  labelled fields, as today; they never compute digests or decide policy.

## 6. Agent orchestration

- **Topology:** one parent (the Execution's stage) and up to 32 children
  from the compiled proposal; one integration+validation owner depending
  transitively on all others (planner rules).
- **Cross-Runtime communication:** only typed records on per-child streams;
  peers read each other's canonical streams; cross-stream references are
  validated at ingestion; orchestrator consumption per §3.4.
- **Ordering and concurrency:** dependencies from the compiled DAG; stage
  `concurrency` already materialized as edges (`compiler.go:397`); the
  scheduler batches only non-conflicting effects; repository writers get
  isolated worktrees; a defensive dispatch-time check refuses a batch larger
  than the recorded stage concurrency (G16) *(inferred: needs the value on
  the graph envelope as an additive field)*.
- **Integration and aggregation:** existing digest-bound Integration preview
  and execute, validators (PD-7), deterministic parent roll-up requiring
  integration-eligible required children.
- **Retry/recovery boundaries:** §3.7.

## 7. Security and authority

| Concern | Control |
|---|---|
| Authorization before dispatch | Dispatch preview + `--authorize-local`; dispatch-authority record committed with the graph/ledger before any attempt; scope limited to first attempts; gates checked |
| Runtime availability / Profile resolution | `runtimeapplication.Check` at preview, apply and per dispatch (existing guard); no-match/disabled/unavailable → blocker, never fallback; initiating Runtime has no preference |
| Capability proof | Existing observation boundary only; unproven capability blocks (IG-4) |
| Authentication | `AuthPreflight` per invocation on the constructed environment; subscription mode mandatory for public dispatch (PD-5); credential references refused (`credential_reference_configured`, existing) |
| Environment | Allowlisted construction (§3.5); no ambient tokens |
| Arguments / confinement | Adapter-owned constant allowlist + appended confinement; bypass options rejected; per-child confinement level shown in preview (PD-4) |
| Execution lineage | Child/attempt/graph revision/lease epoch checked at every entrypoint; empty attempt refused; forged or foreign lineage → `denied_authority` |
| Workflow revision | Recompile under bound revision; binding/observation digest compare as in `workflow_stage_plan.go:105-111`; mismatch → `stale_preview`; corrupt snapshot → `recovery_required` |
| Coordination provenance | Attempt inbox + concurrent ingestion + final drain + epoch fencing; credential/chat-marker filters retained; records never mutate graph/authority/gates |
| Scope and capability | Child effects ⊆ parent authority (`ErrAuthoritySubset`); `process` effect explicit for coordinating children |
| Capacity | 32 nodes, 64 records/stream, 64 KiB/record, attempts 1–10, timeout ≤ 24h; enforced at ingestion before canonical publish |
| External effects | Runtime inference (by the operator's apply) and the local integration effect only; no push/PR/Provider writes (#277) |
| Unknown outcomes | Stay `unknown`; retry blocked until reconciliation; stale/orphan never redispatched |
| Concurrent operators | Run lease + store CAS (`ErrConflict`) → `denied_authority/stale_preview` or `retryable_failure/temporarily_unavailable` with next action |
| Binary pinning | Children call the `axiom` binary whose path and digest are in the packet and preview |

Phase 3 security review covers: stdin packet handling, inbox permissions and
path confinement (POSIX and Windows), adapter allowlist and bypass-option
rejection, environment construction, ingestion parser hardening, supervisor
detachment and process-group handling, and Evidence redaction.

## 8. Failure scenarios

| Scenario | Detection point | Outcome (status / category) | Preserved truth / next |
|---|---|---|---|
| Runtime unavailable / disabled | policy Check at preview, apply, dispatch | `validation_failure` / `runtime_unresolvable` | no graph or no attempt; configure through the #306 path |
| Capability or effort unproven | compiler / policy | `validation_failure` / `runtime_unresolvable` or `unsupported_effort` | no dispatch (IG-4) |
| Authentication unavailable | `AuthPreflight` | child blocked `authentication_blocked` | no process; login next action |
| Credential reference configured | existing guard | `authentication_blocked` / `credential_reference_configured` | blocked until #306 (IG-2) |
| Invalid or stale workflow revision | recompile + binding digest | `denied_authority` / `stale_preview`; corrupt snapshot → `failure` / `recovery_required` | no dispatch |
| Invalid parent/child lineage | publish entrypoint, ingestion, retry/reconcile | `denied_authority` / `authority_denied` | candidate discarded with diagnostic |
| Unauthorized operation | missing/foreign digest or authority | `denied_authority` | no effects |
| Coordination capacity exceeded | ingestion (`ErrCapacity`) | child-side `validation_failure`; parent status shows capacity blocker | stream unchanged |
| Dependency conflict | planner / scheduler | `ErrUnsafeOverlap` at plan, or `effect_conflict_serialized` at dispatch | serialized or blocked before dispatch |
| Unanswered question | packet protocol | asker publishes `blocker`, exits | child blocked, not `unknown` |
| Interrupted Runtime / timeout | supervisor | attempt `unknown`; `interrupted` only if no ambiguous effect, else `failure/recovery_required` | retained attempt; reconcile |
| Initiating session exits | none needed | no change | supervisor continues; status from any surface |
| Supervisor killed | status with free lease | `failure` / `recovery_required` (stale or orphan) | reconcile; G8 fix prevents redispatch |
| Unknown execution outcome | `stage status` | `failure` / `recovery_required` | retry blocked until reconcile |
| Conflicting concurrent operations | lease, graph CAS, coordination chain | `denied_authority` / `stale_preview` or `retryable_failure` / `temporarily_unavailable` | no partial overwrite |
| Primary effect confirmed, projection failed | after commit point | `partial` / `confirmed_effect_reconciliation_required` | references + next |

## 9. Validation strategy

### 9.1 Deterministic tests

- **Unit:** G8 stale-running block; dispatch-preview digest composition and
  drift; publication-request builder; dispatch-authority scope; packet
  builder bounds/digest; stdin delivery; argument allowlist and
  bypass-option rejection per Runtime; environment construction (token
  dropped, login variables kept); ingestion validator for **all nine
  kinds** (lineage, chain, capacity, cross-stream references,
  empty/foreign/non-running attempt, stale epoch, oversized, credential-like
  values); append-only reconciliation facts; cancel classification.
- **Application integration** (`stageexecution` with fake `ProcessRunner`
  and temp stores): graph and single paths; concurrent independent
  children; serialized conflicts; dependency blocking; blocker propagation;
  result-required roll-up; final drain before terminal save;
  kill-between-persist-and-spawn; supervisor death → stale/orphan; resume
  from `GraphStore.Load`; reconcile → retry; cancel before/while running;
  integration and roll-up; stale revision; authority escalation (child
  effects > ceiling); capacity.
- **CLI blackbox** (`cmd/lingo`, controlled executables standing in for
  `codex`/`claude`, existing patterns in `cmd/lingo/blackbox_test.go`):
  preview/apply round trips, detached supervisor lifecycle, exit codes,
  canonical envelopes, `--json` parity, help metadata,
  `coordination publish` denied outside child context.
- **Skill contract/routing:** extend `internal/cli/domain_skill_routing_test.go`,
  `internal/cli/skill_catalog_contract_test.go`, `cmd/lingo/skill_inspect_test.go`,
  `internal/codexruntime/execution_target_skills_test.go`,
  `internal/codexruntime/domain_skills_upgrade_test.go`; Codex and Claude
  installs carry identical content; published receipt/upgrade fixtures updated.
- **Graph/scheduling regressions:** existing `internal/executiongraph`,
  `internal/graphapplication`, `internal/local` suites unchanged and green;
  graph format compatibility for additive fields.
- **Cross-surface parity (synthetic):** start from a Codex-shaped skill
  invocation, inspect/cancel/reconcile from a Claude-shaped one and from the
  CLI (and vice versa); identical canonical records and parent/child
  identity across supervisor restart.

All deterministic Evidence is classified **synthetic**, never operational
acceptance.

### 9.2 Controlled Runtime integration

Test-built fake `codex`/`claude` executables read the stdin packet, publish
coordination through the real `axiom` binary, wait, block or exit with
scripted outcomes, exercising inbox confinement, ingestion, drain and
orchestrator consumption end to end.

### 9.3 Repository checks

`go test ./...`, `./scripts/check-go-quality.sh`,
`./scripts/test-check-project-domain.sh` (architecture boundaries),
`./scripts/test-codex-skills.sh`, `./scripts/validate-repository.sh .`,
`./scripts/check-sensitive-files.sh --staged .`, `git diff --check`.

### 9.4 Early bounded real Codex + Claude smoke (separately authorized)

Run after increments I2–I5 (§11), before the remaining surface work, to
validate PD-2/PD-4 with real Runtimes. **Preconditions:** an explicit human
authorization naming the envelope below, and integration gates IG-1 and
IG-4 satisfied for the smoke's stage (§10); otherwise the smoke is not run
and the reason is recorded.

| Field | Requirement |
|---|---|
| Authorization | Exact envelope: `axiom` build (version, commit, binary SHA-256), lab repository path (scratch repository, never the Axiom repository), workflow definition digest, stage ID, children (Runtime, Profile, model, effort), effects (`read`, `repository-write` in isolated worktrees, `process`; no push/PR/Provider), max attempts 1, per-child timeout, coordination budget, Evidence directory, approver and time |
| Auth mode | Subscription login only (`codex login status`, `claude auth status --json` → `subscription_observed`); no API keys or environment overrides |
| Runtime versions | Recorded from `--version` and executable digests at preview and dispatch; confinement level per child |
| Topology | dag stage with two **independent, concurrent** children: the Codex child publishes a bounded `question_request`; the Claude child publishes the `answer` with `question_reference`; both publish `result`; an integration owner child depends on both and consumes their results as compiled inputs; the stage integrates locally |
| Orchestrator check | `stage status` shows the question resolved, both results consumed and roll-up eligible, before and after the initiating session exits |
| Initiating surface | Start from one Runtime's skill, inspect/integrate from the other's (AC-2) |
| Correlation | Execution ID, graph revision, child IDs, attempt IDs, lease epoch, record IDs/digests, dispatch preview digest |
| Artifacts | Bounded: graph JSON, coordination streams, attempt metadata, validator results, redacted output excerpts; no credentials, account IDs or raw transcripts |
| Classification | `engineering_smoke_real` — not R-2, not R-3, not #278 acceptance |

While IG-1 is open the smoke **is not run** with hand-edited state: this
Plan neither introduces nor endorses a temporary configuration path.

## 10. Dependencies and integration gates

**Consumed, unchanged:** `runtimeprofile.Configuration` read through
`local.NewRuntimeProfileStore`, `runtimeapplication.Service`, the
observation sources in `cmd/lingo/runtime_policy.go`, the #272 preflight.
#276 never writes the Runtime/Profile store, never adds a Profile format
field, and never adds a second registry or temporary configuration path.

**Owned by #306, not touched by #276:** local Runtime/Profile authoring,
credential-reference storage and resolver, Profile invocation parameters,
`axiom-workflow` readiness guidance.

| Gate | Owner | #276 impact | Mitigation |
|---|---|---|---|
| IG-1 Supported local Profile authoring | #306 (its AC-1) | Real smoke and operational AC-1 Evidence need a clean machine configured through supported commands | Deterministic delivery proceeds with existing test fixtures (test-only, never product paths); real smoke waits |
| IG-2 Credential references | #306 security review | Dispatch with `CredentialReference` stays refused | Existing `credential_reference_configured` refusal |
| IG-3 Profile invocation parameters | #306 | Adapter uses Axiom constants only | Compose through the `CommandProfile` builder after #306 review |
| IG-4 Authoritative capability/effort observation (amendment G-3) | **Unassigned**: #275 records it as #272 scope; #306 mentions observed capabilities | On a real machine, stages needing capabilities other than `axiom-skills` (e.g. built-in `read`) are refused, so no real stage can dispatch | **Genuine blocker for real AC-1/AC-3/AC-6 Evidence**; needs a human ownership decision (D-1). #276 does not invent capability proof |

**Likely file conflicts with #306:** `internal/cli/commands.go`,
`internal/cli/cli.go` flag sets, `internal/cli/argument_requirements.go`,
`internal/cli/help_requirements.go`, `internal/cli/skill_inspect.go`,
`docs/commands.md`, `internal/codexruntime/testdata/published-*`,
`cmd/lingo/runtime_policy.go`. Mitigation: #276 commands in new files,
shared-table edits as single appended entries, rebase on `main` at each
phase boundary, coordinate fixture regeneration order across the two PRs.

Other: #230 (graph worktrees retained while referenced; cleanup unchanged),
#133/#232 (register help/rendering metadata only), #277 consumes stage
outcomes, #278 owns R-2/R-3.

## 11. Implementation strategy (increments)

| # | Outcome | Components | Depends on | Validation | Risks |
|---|---|---|---|---|---|
| I1 | Contract gate: decisions recorded; Spec 007 addendum (PD-1, PD-6 categories, PD-7, PD-9) and ADR-0023 draft (PD-2, PD-3 record, PD-4, PD-10, lease order) as **Proposed**, accepted by a human before dependent increments | docs only | Plan approval | doc checks | scope creep |
| I2 | Safety baseline: G8 fix; stdin in `Invocation`; process-group runner; lease + epoch; additive graph fields; publication-request builder from recompiled stage | `executiongraph`, `graphapplication`, `local` | I1 | unit + store regressions + format compatibility | graph format compatibility |
| I3 | `stageexecution` service: preview/apply dispatch, dispatch-authority record, supervisor, graph and single paths, status, resume after restart | new component, `cmd/lingo` composition | I2 | application integration | supervisor portability (Windows) |
| I4 | Adapter: constant argv allowlist, confinement, packet, environment allowlist, mandatory subscription auth | `runtimeadapter`, `graphapplication` | I2 | unit + controlled Runtime | vendor flag drift |
| I5 | Coordination: inbox, publish entrypoint, concurrent ingestion + final drain, stream read/list/show/wait, orchestrator consumption | `coordination`, `local`, service | I3, I4 | all-nine-kinds + negative tests | ingestion races |
| — | **Early real smoke** (§9.4), only if IG-1/IG-4 are satisfied and it is separately authorized | — | I3–I5 | real Evidence | usage limits; vendor confinement |
| I6 | Cancel, reconcile, retry, integrate; validator policy (PD-7) | service, `gitworkspace` | I3, I5 | integration + blackbox | workspace disposition |
| I7 | CLI + help/result metadata + `axiom-work-item` routes; docs (`docs/commands.md`, skill text, receipts) | `internal/cli`, `internal/codexruntime`, docs | I3–I6 | blackbox + skill contract | #306 merge conflicts |
| I8 | Implementation record, Evidence, final validation, review; installed-path smoke when gates allow | docs | all | full checks | gate timing |

## 12. Risks and decisions

### 12.1 Decisions requiring human approval

| ID | Decision | Alternatives | Recommendation | Harder to change later |
|---|---|---|---|---|
| D-1 | Ownership of IG-4 (authoritative capability/effort observation) | (a) new bounded follow-up under #272 scope; (b) fold into #306; (c) extend #276 with its own Spec/security review | (a); #276 proceeds deterministically and its real Evidence waits | real-dispatch readiness of every stage |
| PD-1 | Spec 007 addendum adding `stage status`, `stage reconcile`, `stage integrate`, `coordination show` and their categories | Fold into `workflow status` and `stage run` modes | Addendum with separate leaves (one authority per mutation) | public command names |
| PD-2 | Detached bounded supervisor (same binary) per graph run | Foreground step inside the caller's tool call (timeout-capped) | Supervisor; ADR-0023 | process model |
| PD-3 | Dispatch authority = `stage run` preview/apply committing a dispatch-authority record (graph via `AuthorityReferences`; single via ledger attempt), scoped to first attempts | New fact kind via `workflow fact` | Preview/apply record; no new approval mechanism | approval record shape |
| PD-4 | Attempt inbox + supervisor ingestion; adapter-owned argv allowlist and effect-derived confinement; Claude children marked `advisory` unless an enforceable sandbox is observed | Children write the canonical store directly; or disable Claude coordination until an OS sandbox is mandatory | Inbox + allowlist + truthful confinement level shown at preview | child trust boundary |
| PD-5 | Subscription auth mandatory for public dispatch | Keep opt-in | Mandatory, fail closed | auth posture |
| PD-6 | Implement `stage cancel` per Spec 007/ADR-0009 (persisted cancel request; bounded group stop; `cancelled` only when confirmed) | Explicit unsupported with an existing category | Implement (spec and scheduler semantics exist) | cancellation contract |
| PD-7 | Integration validators come from a separate, reviewed validator-policy artifact with its own preview/`--authorize-local`, referenced by digest in the dispatch preview (Spec 007 addendum) | Commands embedded in the agent-prepared Plan document; or only builtin/human-review validators in #276 | Separate artifact (matches spec.md:176-180) | validator contract |
| PD-8 | Retry after effects requires explicit, authorized workspace disposition (discard/new worktree) | Retry in place only on a clean workspace | Explicit disposition; never auto-reset | retry semantics |
| PD-9 | Deterministic orchestrator consumption rules (§3.4): question resolution, blocker propagation, result-required integration eligibility | Records as Evidence only | Rules as specified | roll-up semantics |
| PD-10 | Reconciliation facts as append-only additive records next to graph attempts (ADR-0009 format extension) | Stage-ledger records | Graph-side additive records | record formats |
| PD-11 | Single-agent stages dispatch through the extended sequential executor with the same adapter/supervisor | Single path stays non-dispatching in #276 | Dispatch (the default workflow is single-agent) | sequential executor contract |

**ADR candidate:** **ADR-0023 (Proposed)** "Public stage dispatch:
supervisor process, dispatch authority, child coordination ingress and
reconciliation records", covering PD-2, the PD-3 record shape, PD-4, PD-10
and the lease position in the ADR-0007 lock order. Drafted in I1; it stays
Proposed until explicit human acceptance.

### 12.2 Risks

| Risk | Type | Mitigation |
|---|---|---|
| IG-4 stays unowned → no real stage can dispatch | dependency | D-1 now; deterministic delivery continues |
| Claude confinement is advisory: a child may write outside its inbox/workspace | security | truthful `advisory` level in preview; ingestion validation; records grant nothing; human decides per dispatch |
| Vendor flags differ by version | cross-Runtime | allowlist per observed version; record versions; refuse unknown versions *(inferred)* |
| Same-user forgery of inbox files | security | ADR-0005 boundary stated; epoch fencing; no authority from records |
| Detached supervisor portability (Windows job objects, macOS) | operational | I3 platform tests; fail closed if detachment unsupported |
| Additive graph/attempt fields break readers | architecture | compatibility tests in `internal/local`; additive only |
| Descendants survive stop | operational | process groups; `unknown` unless confirmed |
| #306 merge conflicts | parallel development | §10 mitigations |
| Usage limits in real smoke (S9 runs 01/02) | operational | bounded retry only under new authority |

### 12.3 Open questions (resolved in Tasks unless escalated)

1. Exact Codex/Claude flags for stdin input, inbox-only write grants and an
   observable sandbox on the installed versions.
2. Exact objects and recovery record for the multi-object commits (§3.6).
3. Detachment primitive per platform for the supervisor.

### 12.4 Review disposition

The first draft was reviewed adversarially. Accepted and incorporated:
the argument guard is a denylist (G5, §3.5); concurrent ingestion with a
final drain (§3.4); orchestrator consumption defined (PD-9); append-only
reconciliation and epoch fencing (§3.6–3.7); cancellation is already
specified (PD-6 revised); dispatch-authority scope and the single path
(PD-3); validator policy under separate authority (PD-7 revised); the
foreground process model is infeasible (PD-2 revised); environment
allowlist (§3.5); lease in the ADR-0007 lock order (§3.6); no pre-#306
hand-configured smoke (§9.4); stdin touches `executiongraph` (G4); the
existing `credential_reference_configured` refusal is reused; the child
binary is pinned; an AC-2 cross-surface case; the single path promoted to
PD-11; empty attempt IDs refused. Added independently: the IG-4
capability-observation blocker.

## 13. Acceptance traceability

| #276 acceptance criterion | Planned components | Validation | Expected Evidence |
|---|---|---|---|
| AC-1 Installed path starts configured Codex+Claude children without an acceptance-only runner or internal-state edits | `stage run` CLI/skill, `stageexecution`, supervisor, adapter packet/confinement, publication builder | CLI blackbox with controlled executables against the installed binary | synthetic blackbox report; **operational** installed smoke only after IG-1 and IG-4 |
| AC-2 Either Runtime initiates the same workflow; restart/status/resume use canonical records and keep parent/child identity | one `axiom-work-item` content for both Runtimes, `stage status`, `GraphStore.Load`, lease/epoch | skill contract + cross-surface parity + supervisor-restart tests; smoke starts in one Runtime and continues in the other | parity report; stable graph/attempt IDs across restart |
| AC-3 Real cross-Runtime Q/A consumed by the orchestrator; tests cover nine kinds, forged lineage, stale revision, authority escalation, capacity | inbox ingestion, cross-stream references, PD-9 consumption | unit/integration for all nine kinds and negative cases; real smoke Q/A | test report; smoke records with digests and correlation |
| AC-4 Independent children overlap only when allowed; conflicts serialize/block; integration and roll-up preserve validation/authority | scheduler/planner + dispatch-time concurrency check, `stage integrate`, PD-7 validators, result-required roll-up | scheduling regressions, overlap timing with fake runners, integration tests | regression results; smoke overlap timestamps |
| AC-5 Denied, failed, interrupted, unknown, recovery-required distinguishable; uncertain attempts reconciled before retry | G8 fix, stale/orphan detection, `stage cancel/reconcile/retry`, §8 table | per-outcome integration + blackbox; supervisor-kill tests | outcome matrix report |
| AC-6 Early real smoke records auth mode, versions, graph, coordination, bounded artifacts; synthetic never classified as operational | §9.4 procedure and classification fields | Evidence field check; human review | smoke Evidence directory (path set in Tasks) classified `engineering_smoke_real`, or a recorded not-run reason while gates are open |

Spec 007 AC-009 (parity, restart lineage, nine coordination kinds, forged
lineage and stale authority denied, unknown reconciled) is covered by AC-2,
AC-3 and AC-5.

---

# Part II — Tasks

**Not yet authorized.** Tasks will be added to this file in Phase 2, on the
same branch and PR, only after human approval of this Plan, D-1 and PD-1 to
PD-11.
