# Plan — #369 bounded pre-spawn launch correction

Revision 1, 2026-11-04. Stage: PLAN / attempt-01. Design: NOT_APPLICABLE
(CLI; no graphical UI). This document plans behavior; it implements none.

## Baseline and authority

[Issue #369](https://github.com/rgomids/axiom/issues/369),
[draft PR #370](https://github.com/rgomids/axiom/pull/370), branch
`fix/369-launch-refused`. Inspected local and remote HEAD before edits:
`bb5c3759a4fce98fb9ee0d19bab4a44d6318f72b`; worktree clean; PR open/draft,
base main. Accepted [amendment revision 1](amendment-369-pre-spawn-launch.md)
and [Specify receipt](https://github.com/rgomids/axiom/issues/369#issuecomment-6621881028)
are preserved. The subsequent [HD-369-D approval](https://github.com/rgomids/axiom/issues/369#issuecomment-6621938832)
authorizes AC-D: its historical pending language is not a current blocker.
Complete Issue comment timeline and PR metadata were read before editing.

[ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md),
[quality policy](../../../.agents/policies/quality.md),
[security policy](../../../.agents/policies/security.md), Spec 007 PE-05a and
retained ADR-0024 controls continue to apply. No ledger, store, architectural
cutover, default workflow budget change, historical refund or new ADR is proposed.
The amendment supersedes the original Plan's “no new parser dependency” choice
only within AC-A. No private config or govsm access, vendor execution, inference,
merge, release, acceptance or Issue closure is included.

## Observed dependency and consumer inventory

References below are existing files/functions at the baseline, not proposed APIs.
Graphify was queried first using `GRAPHIFY_BACKEND=none`; its broad result was
truncated and used only as a navigation aid. Source inspection establishes facts.
`rg` is unavailable in this environment; targeted grep/find supplied caller checks.

| Owner / existing reference | Existing contract and affected consumers |
| --- | --- |
| `internal/runtimeadapter/effort_sources.go`: `observeEffortFacts`, `tomlEffortSelectorsBounded`, `readEffortSource`, `observeEffortSource`, `effortPathBinding` | Central argv/env/config observation, narrow TOML parser, double-read/binding, managed inventory; preserve source recognition and bounded reads. TOML parser directly feeds `observeEffortFacts` and parser tests. |
| `internal/runtimeadapter/effort_v4.go`: `fixedEffortRules`, `inspectEffortV4`, `InspectEffortSourcesV4`, `StageEffortRoots.ResolveOperational`, `VerifyChildEquivalenceV4` | Fixed user/project/managed sources; Claude project-local/drop-ins. Native effort inspector and child-equivalence consumers share the parser. Operational Codex uses nested profiles; preserve false-mode behavior of direct/legacy-scope tests. |
| `internal/runtimeadapter/frozen_invocation.go`: `CheckFrozenSelectors` | Launcher calls this for frozen Runtime/model/effort and executable identity. No vendor prediction/authentication. |
| `internal/runtimeadapter/stage_invocation.go`: `BuildStageInvocation`, `StageEnvironment`, `AttemptCorrelation.Environment` | Pure argv/CWD/environment assembly, child markers, selector-relocation refusal. Consumers: RuntimeLauncher.Build/Collect, `operational_source.go:effortInvocation`, stage invocation/path/v4 tests. Keep signatures where possible; reuse pure assembly. |
| `internal/runtimeadapter/stage_effort.go`: `StageEffortInspector.CheckEffortSources`; `operational_source.go:ObserveOperational` | Retained inspection/digest/equivalence consumers must retain errors.Is compatibility, synthetic tests, redaction and cancellation semantics. Do not widen to auth checks. |
| `internal/stageexecution/runtime_launcher.go`: `RuntimeLauncher.Build`, `inbox`, `cleanRuntimePath`, `Collect` | Build checks paths, creates attempt/send/receive dirs, assembles invocation, resolves roots/checks selectors, then serializes input. All fail paths currently collapse to ErrRuntimeLaunch. Collect uses StageEnvironment for authorized commands; preserve bounded output and child markers. |
| `internal/stageexecution/engine.go`: `runAttempt`, `commit`, `recordInterrupt`, `watchCancel`, `Reconcile`; `scheduler.go:runReady` / RunOptions.Retry | Build -> cancellation/admission check -> Runner.RunObserved -> MarkStarted -> Collect -> terminal commit. Build error currently MarkSpawnFailed with constant. New checks/reservation belong inside the fence, not Launcher or CLI. |
| `internal/workflow/execution.go`: Execution/Attempt, Validate, EncodeExecution/DecodeExecution; `execution_transitions.go`: PrepareAttempt/MarkSpawnFailed/MarkStarted/MarkUnknown/MarkCancelled | One nested aggregate; attempts currently limited by record count. New policy must affect validation, preparation, retry, readiness and terminal/recovery transitions together. `execution_transitions.go:Ready` and task projection must agree with counters. |
| `internal/local/execution_store.go`: Load/read, AcquireRunner/AcquireFence, Save, UpdateFenced, preservesExecutionHistory | Strict codec, revision CAS, lease generation, immutable frozen identity/terminal history. Extend existing protected fields and transition checks, never introduce another accounting owner. `cloneExecutionStages` shallow-copies Attempt: prefer scalar additions or deep-clone new pointer/slice metadata to preserve snapshot isolation. `internal/strictjson/shape.go:Decode` re-marshals member shape: absent additions require omitempty. |
| `internal/local/inventory.go`, `recovery.go`, state compatibility tests; `internal/compatibility` stable corpus | Recognize same execution location/kind/format, preserve legacy historical recognition. Add new writer/corpus cases; never edit frozen snapshots or re-enable retired execution formats. |
| `internal/projectapp/readiness.go`: ProjectReadiness.Evaluate/ReadinessReport/RuntimeReadiness | Structural readiness consumed by project validation and preflights; disclose invocation not evaluated without making readiness a vendor/admission probe. |
| `internal/runtimeapplication/authoring.go`: ProfileView/projectLocal; `invocation.go:ResolveInvocation` | Eligible currently means enabled/allowlisted configuration. Keep meaning, add explicit scope; resolver owns field precedence and provenance for prospective preview/retry. |
| `cmd/lingo/execution_run.go`: executionStage, executionComposition.run/retry/resolve/launcher/existingLauncher, executionPayload/executionNext/outcome/failure | launcher ALREADY MkdirAlls outputs and constructs artifact store. Diagnose must bypass it and all admission/CAS/lease paths; split read-only path derivation from directory/store creation. JSON/text projections consume canonical attempts, not diagnostic parsing. |
| `internal/cli/workflow_stage.go`: action/operation tables, workflowStageCommands/FlagSet/Requirements/runWorkflowStage/SkillSpecs | Add diagnose through existing route/service/envelope; update command/help metadata and route/JSON parity tests. No alternate CLI family. |
| `docs/commands.md`, `internal/cli` help/docs metadata, `internal/codexruntime/skills/axiom-work-item/SKILL.md` | Reconcile installed docs and embedded operational skill for diagnose, scopes, budget/retry semantics; run documentation/renderer contracts. Historical testdata snapshots stay immutable. |

Implementation must repeat a targeted caller search before changing a shared
signature. No exported API is broadened merely to make tests convenient.

## Technical decisions and implementation sequence

All decisions here are TECHNICAL_LOCAL unless explicitly HD-369-D. The sequence
is input to a later Tasks artifact, not authorization to implement during PLAN.

1. **P1 — freeze compatibility and counter examples first (B/D).** Add fixtures
   for original format-4 bytes, old binary decoder behavior, budget exhaustion,
   joined errors and existing cancellation. Establish strict schema behavior
   before writer changes. Preserve current historical execution semantics.
2. **P2 — carry typed failures (B).** Introduce small closed reason/phase values
   in workflow and a typed launcher error compatible with errors.Is(ErrRuntimeLaunch).
   Preserve cause classification at adapter/path boundaries. Persist safe codes
   on canonical attempts and project them through status/reconcile/JSON/text.
   Keep old Diagnostic broad family; no new log store.
3. **P3 — replace TOML grammar (A), after preserving the decoder security proof.** Remove scalar
   special cases and key splitting once a compliant bounded decoder is selected.
   Decode into ordinary TOML values, then bounded semantic inspection. This
   increment alone does not require D or diagnosis. See dependency evidence below.
4. **P4 — shared pure inspection and optional diagnose (C).** Reuse P2 codes and
   P3 parser, extract invocation assembly/selector checks from effectful Build.
   Route read-only diagnosis around CLI launcher creation. Disclose scopes in
   Project/profile views. Recheck at dispatch, never cache an admission verdict.
5. **P5 — prospective accounting (D), after P1/P2.** Freeze policy on new
   Executions, introduce bounded attempt accounting/lineage and charged reservation
   transitions in the existing aggregate. Change engine and history guard together;
   do not ship a writer whose reader/validation/recovery disagrees. Integrate with
   cancellation, explicit retry/new-selection and scheduler. D is approved;
   parser investigation does not block this independent increment.
6. **P6 — integrated verification/docs.** Run the matrix, review authority,
   redaction and compatibility; update installed command/skill contracts. Release
   only after all four ACs have behavioral Evidence and normal later gates.

No generic validation framework, new execution kind, second store, automatic
retry, auth prerequisite, profile-wide scan or Claude per-model exception is
needed. Alternatives rejected: notify-only parsing (incomplete TOML), a second
budget ledger (divergent authority), diagnostic invoking Build (writes), inferring
failure categories from strings (unstable/untrusted), and increasing r2 budgets
(masking the defect). A material contract conflict discovered by implementation
blocks only that dependent change and is reported on #369, never self-approved.

## A — decoder and semantic contract

Dependency research and executed source-level evidence are in
[dependency evidence](evidence-plan-369.md). The initially preferred exact
candidate was `github.com/pelletier/go-toml/v2 v2.4.3`; it is rejected because
its parser-time bound is 10000, despite the advisory fix. Select exact maintained
upstream `github.com/BurntSushi/toml v1.6.1-0.20260818200445-d733fc535e4a`, MIT,
with explicit `Decoder.MaxDepth(64)`. Source guards independently bound arrays
and inline-table contexts; configured64 bounds their combined recursion below128.
The isolated proof is PASS; integration and semantic traversal remain planned. No custom TOML pre-parser,
unstable AST dependency, runtime Python, or post-decode-only mitigation is accepted.

Implementation acceptance for the selected decoder: syntax and duplicate/table
validation are owned by the maintained decoder; nesting guard executes before
recursive descent/allocation above 128. Source review plus synthetic 128/129 and
mixed inline-array nesting tests must demonstrate the bound. A dotted key/table
path must also stay bounded before constructing a deep decoded tree; reject
resource excess conservatively. Then traverse keys/arrays/maps with semantic depth
<=32 and nodes <=65536. Count root, containers, keys and scalar values consistently
in tests; use a bounded worklist rather than unbounded recursion. Guard 1 MiB
before decode; preserve 64 managed drop-ins, existing source count limits,
symlink/ancestor/file identity and double-read/inventory-drift controls.

Detect model_reasoning_effort anywhere in relevant maps, quoted/dotted keys,
inline tables and arrays of tables; strings/comments mentioning it are data.
Equal-valued competing selectors still conflict. nestedProfiles=false must reject
profiles at every depth/form rather than granting new scope. Resource/identity
refusal is unsafe; malformed/duplicate TOML is invalid. Update the current valid
array/inline/date refusal expectations while retaining malformed/security cases.
Claude Sonnet modelSettings.effortLevel with frozen Haiku remains selector_conflict:
no evidence of noninterference or authority to weaken that rule was obtained.

## B — failure representation, safety and persisted compatibility

Proposed minimal fields: optional `launchFailure` on Attempt, containing closed
`reason`, `phase` and optional closed `source` class, never raw error text. Reasons:
configuration_invalid, selector_conflict, configuration_unsafe, path_unsafe,
permission_denied, identity_unsafe, executable_unavailable, builder_failed.
Phases: invocation_build, source_resolution, selector_inspection, spawn, runtime.
Source classes: user/project/project-local/managed/managed-drop-in/argv/environment;
omit when unknown. A static mapping supplies redacted remediation at projection;
do not persist arbitrary remediation strings or keys. Cancellation remains
cancelled; runtime nonzero remains observed runtime failure, never presumed login.
Do not force arbitrary runtime failures into a configuration category.

Use errors.Is/As, ctx cancellation first, Invalid before joined Unsafe, then
specific selector/identity/path/permission/executable categories before broad
unsafe. Permission requires typed os.ErrPermission/fs.PathError cause; no errno
message matching. Preserve existing broad sentinel matches. Unknown builder error
maps builder_failed/invocation_build and is NOT deterministic free refusal.
Only known closed validator failures qualify for deterministic classification;
Build's mere return-before-Runner is insufficient evidence of safe accounting.
Audit every Build return and engine projection; no `err.Error()` or `%v` reaches
CLI, attempt bytes or logs. Keep internal error causes only where needed for Is/As;
never serialize them. Test path/token/config-value canaries in every output sink.

Use format 4 and existing namespace; new fields tagged omitempty. New reader must
accept every legitimately written old record, with absent policy meaning legacy
record-count accounting and absent failure meaning unavailable historical detail.
Never backfill typed fields by parsing Diagnostic. New policy exact string:
`spawn-boundary-v1`; absent is legacy, unknown nonempty policy is invalid.
New writers freeze policy at creation; optional fields are mandatory semantically
for new-policy attempts. The old strict decoder rejects unknown policy/failure
members before execution effects. This is a fail-closed downgrade, not permission
to execute the new bytes with an old binary. Extend history guards to freeze policy,
accounting evidence, lineage, reason/phase/source once recorded. Historical terminal
records remain byte-equivalent; historical mutable executions keep old policy and
must not gain fields that silently reinterpret budget. If adding B detail to a
legacy-policy active record, disclose that old decoder may refuse the updated
record; test the boundary, never silently migrate it.

P1 must execute an actual baseline decoder harness against candidate new bytes,
not merely assert DisallowUnknownFields behavior. Loading unsupported bytes must
produce zero AcquireRunner/prepare/spawn/network calls and unchanged directories.
If this cannot preserve ADR-0037/strict codec/quality contracts, stop that writer
and report the exact material incompatibility. No new format is assumed necessary.

## C — read-only diagnosis and dispatch parity

Proposed route: `workflow stage diagnose --project <id> --execution <id>
--stage <id> --agent <id>` through existing WorkflowStageService. Default selects
the requested persisted attempt's frozen invocation and provenance; ambiguity
requires explicit attempt selection. `--new-selection` plus existing selection
flags is a clearly labeled prospective preview through ResolveInvocation using
current sources. It never changes/rebinds the prior attempt and grants no retry
authority. Reuse safe logical provenance fields; omit executable/private paths,
instructions/config contents and arbitrary environment values from output.

Split pure assembly from materialization. Derive planned inbox paths without
MkdirAll, artifact store construction, or launcher Build. Read only canonical
Execution/Project/configuration and recognized consented runtime config sources;
never credential files. New read-only app branch must not call run/retry/admit,
AcquireRunner, UpdateFenced, CAS, installation/recovery writers, subprocess, vendor,
network, collection or command validation. Existing Load/read-only inspection is
allowed; constructing mutation-capable services with directory creation is not.

Return five explicit dimensions: Project structural readiness, profile
configuration eligibility, effective local invocation admissibility
(admissible/refused/not_evaluated), authentication=not_evaluated, inference=not_run.
Missing would-be inbox/CWD means corresponding materialized-path checks remain
not_evaluated; inspect existing ancestors without creating placeholders. Do not
infer inaccessible vs absent source, working root or executable. Partial checks
cannot aggregate to admissible. A known selector conflict may be reported even
if other checks remain not_evaluated. Project validate and profile listings retain
ready/eligible meaning but explicitly disclose invocation not_evaluated and how
to diagnose; no shell auth observation upgrades child authentication to success.

Effectful dispatch performs materialization and final same-tuple/environment/CWD
selector/path/identity checks immediately before charged reservation/Runner;
recheck authority/cancel/generation there. A previous diagnostic is never an
admission token. File races still use double-read/source identity controls;
point-in-time inspection does not promise vendor behavior or eliminate TOCTOU.

## D — policy invariants, transitions and crash timeline

HD-369-D defines maximumAttempts as charged launch attempts, including spawn
failure, runtime failure and unresolved spawn outcomes, not inference successes.
One task lifetime has <=2 deterministic refusal records, regardless of selections,
restart/resume/stage status or successful spawns. Record bound <=maximumAttempts+2,
with existing maximumAttempts<=10 unchanged. Deny preparation if charged budget or
the two-refusal allowance is exhausted; never append a third refusal record.
Compute counts from bounded canonical attempt accounting, not duplicated counters.

Proposed optional per-attempt accounting enum: pending / charged /
deterministic_refusal, only on new-policy executions. Pending is durable preparation
with unresolved outcome and consumes a budget slot conservatively until a fenced
known deterministic failure is committed. It is not an available free retry.
Persist reservation metadata/time before Runner; charged is monotonic, never
converted back to refusal. Policy absence keeps the entire legacy state machine.

| Boundary | Durable operation and admissible next action |
| --- | --- |
| Before PrepareAttempt | Validate task limits, authority, frozen tuple, parent/disposition and fence. Rejection appends nothing; explicit caller fixes/retries. |
| PrepareAttempt | Append new ID, ordinal, parent attempt ID (first absent), actor/time, generation, immutable invocation/provenance/authority/workspace disposition, accounting pending under CAS. No automatic retry. |
| Known deterministic Build/inspection refusal before runner call | Engine alone commits failed terminal attempt + launchFailure + deterministic_refusal, no-spawn assertion and local materialization disposition. This consumes one lifetime refusal; no charged reservation exists. |
| Unknown builder error / cancellation / revoked authority | Never free refusal. Conservatively commit charged disposition and failed/cancelled status, without claiming spawn occurred. If terminal commit fails, pending/unknown still consumes budget. No bypass of revoked authority to run. |
| Build succeeds / final recheck succeeds | UpdateFenced commits charged reservation and reservation time; after successful commit only, invoke Runner.RunObserved exactly once. Failed CAS/stale fence prevents call. |
| Crash before preparation commit | No published attempt; publication recovery applies. Never infer success from temporary bytes. |
| Crash after preparation, before refusal/reservation commit | Pending becomes unknown under reconciliation and counts conservatively. No PID is no proof of no-spawn. Explicit abandon/disposition only, no replay. |
| Crash after reservation, before Runner or before start callback | Charged unknown; cannot prove where crash occurred. Never refund or automatically spawn. |
| Spawn fails / starts / terminal commit lost | Keep charged; phase spawn / runtime as applicable. Preserve process identity and existing unknown/command recovery rules; missing terminal result is not free. |

Retry of known terminal refusal remains explicit human-authorized RunOptions.Retry,
new record/parent lineage, new actor/time and current fence. Preserve invocation
unless explicit --new-selection freezes another tuple using existing precedence.
Validate current target/effects authorization and workspace preservation; no
inherited authority from old attempts/new Executions. Unknown outcomes require
existing explicit resolution and safe workspace disposition before retry. Stale
revision/generation, active foreign lease, revoked authority or unconfirmed effects
refuse. Every path checks both lifetime refusal and charged/total bounds.

New-policy failed Build still stops progression; no scheduler auto-resume loop.
Existing historical exhausted/terminal executions remain exhausted/terminal even
when diagnostics look pre-spawn. Upgrade reads old format4; downgrade only reads
unchanged historical supported bytes, rejects new-policy bytes before effects.
Do not bump default-sdd-r2.json or change its definition digest.

## Traceability and executable Evidence strategy

All tests below are PLANNED unless explicitly listed as executed in the evidence
artifact. Use temporary synthetic HOME, managed, repository and executable roots;
no real vendor calls. Reuse effortInspectorFixture, launcher/engine fake runners,
local CAS/crash hooks, CLI recorder fixtures and immutable compatibility corpus.

| AC / priority | Proposed checks / code owners | Observable Evidence and safety negatives |
| --- | --- | --- |
| A / P0 | runtimeadapter notify array alone; notify+model_reasoning_effort; inline/quoted/dotted/multiline/date/array-table forms; selector in every nested map; comments/strings do not conflict | Valid unrelated forms pass; real/equal-valued selector conflicts; nestedProfiles=false still restricted. Original synthetic Codex case distinguishes syntax from selector. |
| A / P0 | Decoder source bound, 128/129 mixed containers; semantic 32/33, 65536/65537; duplicate keys/tables, malformed/UTF-8, >1MiB; 64/65 drop-ins | No stack overflow/unbounded parse before semantic walk; rejected inputs return safe typed code, not leaked decoder context. Test deep dotted paths too. |
| A/B / P0 | symlinked ancestors/file, replacement between reads, changed binding/inventory/executable, readonly/permission roots | Unsafe/identity/permission categories only with typed evidence; never accepts changed source or weakens containment. |
| B / P0 | All launcher fail exits; Is/As nested/joined Invalid+Unsafe in either order, unknown error, cancellation, spawn/runtime outcomes | Persisted typed reason/phase survives codec/status/reconcile; launch_refused compatibility; cancellation remains distinct. |
| B / P0 | Inject synthetic path/token/value/child-output canary in every underlying error | Canary absent from text, CLI JSON, persisted JSON and captured logs; remediation/source classes static. |
| C / P0 | Same frozen tuple diagnose vs dispatch; altered defaults, explicit new-selection preview, synthetic Sonnet effort with Haiku | Frozen snapshot never rebinds. Claude remains selector_conflict, no claim of model-specific noninterference. |
| C / P0 | Snapshot bytes/modes/dir trees before/after; missing inbox, missing roots, cancelled context; panic spies for mutation/runner/network/vendor | No dirs/inbox/store/CAS/lease/provider/subprocess/network effects; missing materialized paths not_evaluated, auth unknown/inference not_run; no host HOME inspection. |
| D / P0 | Refusal -> explicit corrected retry; two lifetime refusals then prepare refused; selection/restart/resume/success between refusals | Distinct IDs/ordinals/parent/actor/time; two lifetime slots do not reset; max+2 enforced without third append. |
| D / P0 | Confirmed spawn failure/runtime failure/unknown builder/cancel/revocation | Charged conservative accounting; only closed fenced deterministic pre-call failure gets refusal. No inference-success counter; no automatic retries. |
| D / P0 | Crash hooks before/after prepare, reservation, Runner, start callback/terminal save; lost output/command result | No durable reservation -> no runner call; pending/charged unknown blocks replay; never derives no-spawn from missing PID. |
| B/D / P0 | Strict unknown enum/field/missing mandatory new-policy data; old reader harness; current old-policy round trip; corpus guards; stale CAS/lease/generation and history rewrites | New reader accepts old bytes, old reader refuses new bytes before effects, no refund/reclassification/migration; frozen history/policy/accounting immutable. |
| C / P1 | Project/profile help/text/JSON, route requirements/unknown flags, installed docs/skill renderer | ready/eligible explicitly scope-limited, diagnosis has same canonical envelope; no authentication/inference success inferred. |

Exact safe commands for later implementation, after inspecting added test names:

```sh
go test ./internal/runtimeadapter -count=1
go test ./internal/workflow ./internal/stageexecution -count=1
go test ./internal/local ./internal/strictjson ./internal/compatibility -count=1
go test ./internal/projectapp ./internal/runtimeapplication ./internal/cli ./cmd/lingo -count=1
go test ./internal/install ./internal/codexruntime -count=1
go test -race ./internal/workflow ./internal/stageexecution ./internal/local -count=1
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
./scripts/check-sensitive-files.sh --staged .
git diff --cached --check
```

Inspect package test subprocesses before broad execution; real process fixtures
may run only synthetic/local helpers, never vendor binaries. Full suite/platform
checks belong to later implementation CI. Persisted writer release guard remains
`AXIOM_FREEZE_STATE_CORPUS=<release> go test ./internal/local -run TestEveryV1WriterIsRecognizedByInventory`;
do not run it during PLAN or change stable snapshots to make a failure pass.
No AI/vendor battery is needed for this stage; exact user-approved model pins
apply if separately authorized later. No production/inference validation claimed.

## Rollout, rollback and handoff

Ship reader/validator/history protection before enabling new-policy writers in the
same tested release; no feature flag/config option needed. Deployment success
requires A-D matrix passes, unchanged historical corpus, downgrade refusal before
effects, installed docs/CLI parity and review of security/accounting races.
A regression in old read compatibility, parse bound, redaction, no-effect diagnosis,
reservation durability or authorization is a stop condition for the dependent
increment. Preserve the pinned source-bound proof and regression fixtures before parser adoption.

Rollback before new writes is an ordinary code revert. After new-policy writes,
old binaries must refuse those records; operational rollback requires retaining a
capable reader or stopping affected executions, preserving all bytes/outputs.
Never strip fields, rewrite terminal history, refund unknown outcomes or downgrade
in place. A future conversion/cutover would be a separate material decision.

Tasks stage consumes P1–P6, the inventory and matrix to create bounded tasks with
preconditions and Evidence; it must port the isolated parser proof into P3 regression tests, without holding
independent B/C/D work. There is no remaining material decision or parser research
blocker; passing an isolated proof does not establish integration correctness.
Versioned artifacts are this Plan, its Evidence and the specifications index link.
Issue completion receipt supplies final commit/push/read-back/CI observations.
Next gate after verified publication: TASKS. No continuation integration is assumed;
without an actual authorized callback, report AWAITING_EXECUTION_REPORT.
