# Issue #404 Plan — Canonical Stage concurrency

## Revision, authority and scope

PLAN-404-01, revision 1, 2026-11-24. Implementation plan for qualification;
**documentation only**, no Tasks or implementation authority from this artifact.
Behavior is the accepted [DC-01–DC-05 amendment](amendment-404-canonical-stage-concurrency.md),
[Spec 007](spec.md), [scope decision](https://github.com/rgomids/axiom/issues/404#issuecomment-7133606313)
and [Specification acceptance at a869b8c](https://github.com/rgomids/axiom/issues/404#issuecomment-7134082942).
[Readiness](https://github.com/rgomids/axiom/issues/404#issuecomment-7134106824)
authorizes this Plan alone. [SPECIFY-404-02 receipt](https://github.com/rgomids/axiom/issues/404#issuecomment-7133909888)
and [SPECIFY-404-01 historical note](issue-404-dag-concurrency-specify.md)
remain unchanged; the earlier impediment is history, not a reopened gate.
AC1–AC4 originate in the [accepted Intake](https://github.com/rgomids/axiom/issues/404#issuecomment-7132349983).
Design: NOT_APPLICABLE.

Pre-edit inspection: clean existing worktree `/home/grokbot/axiom-wt/gh-404-specify`,
branch `issue/404-specify-dag-concurrency`, origin `https://github.com/rgomids/axiom.git`;
local and remote HEAD `a869b8c40cfe45c2ea8c8ef746945503145a34ff`, main/base
`5cd783bd2ac8189a1b9c85a790472b5335139c78`, [PR #405](https://github.com/rgomids/axiom/pull/405)
open Draft. Existing diff: three documentation files, no production/test change.
Issue comments and Spec 007 filenames contained no competing Plan/result for
this attempt. This does not prove external sessions inactive; recheck remote
HEAD and Issue events before publication, preserve and reconcile divergence.
No new branch, worktree or PR is needed.

Only the canonical [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md)
Execution/store/engine path is in scope. Read [Spec 015](../015-minimum-viable-operational-workflow/spec.md)
with its accepted reconciliation, not its preserved Proposed status language.
[ADR-0039](../../decisions/0039-unrestricted-runtime-stage-processes.md) remains effective.
No legacy Graph coordinator, new engine/store, runtime policy, public CLI flag,
artificial dependencies or new ADR. `ubuntu-t12` r1 and its digest
`80b733ae0281f99c4b5f6714db078952580242dbd7fd6b05e623adb86f550df8`
are untouched. Battery 02 BLOCKED/NOT_RUN and zero processes are reported history,
not reproduced here. No battery 03, real inference or product external effects.

## Source inventory and internal interfaces

Exact files were inspected at the starting HEAD; production is unchanged from
base. A bounded `GRAPHIFY_BACKEND=none graphify query` located the canonical
symbols; graph freshness/inferred edges do not establish operational behavior.

| Location / observed contract | Planned change or integration point |
| --- | --- |
| [execution.go](../../../internal/workflow/execution.go): StageSpec, WorkflowSnapshot, cloneStages, workflowDigest, validateWorkflow, Encode/DecodeExecution | Append optional concurrency member, validate explicit bounds; preserve existing serialization, digest and cloning. No default insertion in constructors/decoder for absent input. |
| [strictjson/shape.go](../../../internal/strictjson/shape.go) | Reuse bounded strict decode and exact shape comparison unchanged if oracles pass. Pointer null omits on re-marshal, so explicit null must fail shape comparison. Do not relax shared strictness. |
| [execution_store.go](../../../internal/local/execution_store.go): read, validateExecutionUpdate, UpdateFenced | Keep frozen Workflow deep equality, runner fence, exact-byte CAS, lock order and protected replacement. Test the extension through this store; no equality exception for defaults. |
| [execution_transitions.go](../../../internal/workflow/execution_transitions.go): Ready, PrepareInput, PrepareAttempt | Shared effective ceiling/occupancy calculation; durable admission check for ordinary preparation and explicit retry, with current principal inflight facts at engine admission. |
| [scheduler.go](../../../internal/stageexecution/scheduler.go): runReady, attemptStore, attemptFinished | Keep single principal writer and worker commit queue. Track exact attempt identity until finished observation, re-evaluate capacity per admission and release. |
| [engine.go](../../../internal/stageexecution/engine.go): prepareAttempt, runAttempt, Claim/orphanedAttempts | Pass principal occupancy to fenced preparation; preserve prepared-before-spawn, failures, recovery unknown/no replay and invocation freezing. No alternate dispatch path. |
| [execution_run.go](../../../cmd/lingo/execution_run.go): selectedWorkflow/mapWorkflowDefinition | Keep definition Encode validation, remove only concurrency !=1 refusal after enforcement exists; transfer exact value including 1. Other refusals and ordered Stage/dependency mapping remain. |
| [workflowdefinition/validation.go](../../../internal/workflowdefinition/validation.go), [schema 2](workflow-v2.schema.json) | Retain 1..32, <=agent count, sequential 1, cycles/references/owners and all other validation. These are input authority, not scheduler code. |
| [execution_default.go](../../../internal/workflow/execution_default.go), [run_start.go](../../../internal/stageexecution/run_start.go) | Audit minimal default construction and all start consumers. Newly constructed default Stage explicitly carries 1; retained default snapshots are never regenerated. NewExecution clones supplied presence rather than inventing historical intent. |

Constructor/fixture audit includes `twoStageExecution`, `messagingExecution`,
`storeTestExecution`, `newEngineFixture`/`newEngineFixturePolicy`, `parallelStages`,
`collaborationStages`, `newExecutionRunFixture`, `criterionWorkflow`, `completeFirstStage`
and StageSpec literals in workflow selection/accounting/workspace/integration,
local store/output/rework/workspace, stageexecution capture/delivery/exchange/
integration/repository-outcomes/runtime-launcher/run-start and CLI freeze/reference
checks. Set explicit intended limits in fixtures testing new parallel behavior;
do not bulk-default historical fixtures or weaken their assertions. No production
caller beyond this bounded path is assumed exempt; TASKS must retain this audit.

## Local decisions and compatibility strategy

**TD-01 — Optional integer, no custom codec.** Plan `Concurrency *int` with
`json:"concurrency,omitempty"`, appended after the existing StageSpec members.
Nil represents absent, non-nil 1 remains explicit, non-nil 0 remains invalid.
Use a fresh value per mapped Stage, avoid shared mutable pointer aliases;
existing JSON clone provides independent values. An int+omitempty conflates
zero/absence; int+presence bool requires extra codec work; custom marshal or
format bump adds risk without a validated need. No extra dependency is needed.
These are TECHNICAL_LOCAL choices, source-qualified, executable UNVERIFIED.

Keep `kind: execution`, `formatVersion: 4`, namespace and every old struct member
order/tag unchanged. For nil, encoder and digest input must exactly match the
pre-change encoder. For present values, digest includes concurrency. Validation
checks the stored digest first, then explicit integer range 1..32 and <=agents;
mode/sequential correctness stays at validated definition mapping (StageSpec has
no mode and gains none). Never reject a historical record solely because its
active count exceeds the fallback; bounds apply to the explicit declaration,
capacity to new admission. Never add active-count rejection to general Validate.

Strict decoding must distinguish absent from explicit null/zero. Reuse existing
bounded duplicate-key/unknown-field decoder, integer decoding and shape check:
null cannot equal the omitted canonical member; mis-cased aliases change shape;
fractions/overflow/string/bool reject integer decoding; negative, zero, >32 and
>agent count fail semantic validation. Test duplicates even when values match,
and unknown nested fields. Do not assume shape comparison checks numeric value:
digest/semantic checks remain necessary. Tampering a limit without its digest
fails decode; recomputing the digest still cannot mutate a stored Workflow under
frozen equality/CAS. Removing a present member is also a frozen-state mutation.

**TD-02 — Preserve absence; derive fallback only for admission.** A small helper
returns declared value or effective 1, without mutation. Decoder, clone, progress,
restart and inspection retain nil and original Workflow digest. No lookup of
current Workflow/selection, inference of prior declared value or rewriting of
history. Historical multi-active records load normally; no new preparation at
or above effective 1. Claim still marks orphaned prepared/running attempts unknown;
unknown/recovery-required blocks dispatch until existing explicit resolution,
never replay/adopt a process. Explicit 2 persists across restart unchanged.
Alternatives rejected: eager 1 mutates frozen history, current-definition lookup
invents historical intent, and unbounded absence violates the approved fallback.

Compatibility proof is the first implementation dependency, not a conclusion of
this Plan. Freeze sanitized complete canonical format-4 bytes using the unchanged
baseline encoder at `a869b8c` (production-equivalent to `5cd783b`) before editing
it, with fixed IDs/time and no host/provider payload. Cover single, independent
DAG, terminal, prepared/running and multi-active history. Record generator source
SHA, full-file SHA-256 and Workflow digest as literals independent of the amended
encoder. Reuse `internal/workflow/testdata/i369` and `execution_369_test.go`'s
byte-hash/round-trip pattern without changing those fixtures; put #404 additions
in a separate issue-specific testdata directory. Do not regenerate expected
bytes/digests with the modified encoder. Also pin explicit 1/2 expected digest
input independently (literal ordered JSON + SHA-256) to avoid a self-oracle.

Prove unchanged bytes on historical decode/encode, nil through clone, independent
pointer mutation safety, and unchanged frozen Workflow/digest through actual
fenced store progress and reload (whole Execution bytes legitimately change on
progress). Exercise canonical-location inventory and append-only stable corpus
checks under the quality policy; do not edit/prune released snapshots. No new
namespace/inventory kind or migration service. A new writer sample may extend
its own future release corpus only through established freeze procedure.
Use the pre-change reader at its pinned source in an isolated test fixture to
prove refusal of new-member bytes; this is an old-reader refusal oracle, not
promised downgrade. Do not import operational state or invoke an installed model.

## Admission and worker lifetime

**TD-03 — One capacity rule inside existing fencing.** Keep Ready's public
no-argument use for durable-state eligibility, constrained by durable capacity;
use the same small calculation with principal inflight IDs at engine admission.
A minimal internal extension to PrepareInput carries current inflight attempt
IDs (not a persisted field, CLI option, counter authority or worker-supplied
claim). Engine.prepareAttempt passes them to the `UpdateFenced` mutation;
PrepareAttempt checks their union with the latest aggregate's active attempts,
for both new tasks and retry. Direct domain preparations without workers still
check durable occupancy. No independent store or worker may authorize admission;
all actual runReady dispatches pass through the principal/fenced path.
Cached Ready lists or a worker-local counter cannot cover stale state or terminal
inflight windows; a separate coordinator/legacy Graph would add an owner contrary
to ADR-0037. Reuse of the principal queue is the smallest complete alternative.

For Stage S, occupancy is the cardinality of distinct attempt IDs in:
`durable prepared/running(S) UNION principal inflight(S)`.
Use identity bound to stage/task/attempt, not TaskRef alone (retry reuses a task).
Validate correlation of supplied IDs against the current Execution; deduplicate
and retain terminal inflight attempts. Durable status, not charged budget alone,
is the occupancy source. AccountingPending and validation/output collection
remain occupied; budget refund does not itself free capacity. Unknown retains
its existing global stop semantics, never a capacity credit.

Ready may expose at most remaining durable capacity in Workflow order; it cannot
reserve those slots. Before each preparation, recompute against current aggregate
and principal inflight; recheck in the fenced mutation, then persist prepared
before spawn. A list computed earlier grants nothing. Return/recognize an internal
capacity-unavailable disposition without conflating ordinary saturation with a
fatal worker/control error or a public CLI contract. Stop filling that Stage,
service existing commits/finished events, then recompute. Preserve an explicit
retry request until admitted or definitively refused, rather than consuming it
on temporary saturation; retry cannot bypass the ceiling, unknown resolution,
current authority, budget or frozen invocation/new-selection requirements.

After preparation, register inflight identity before starting worker; remove it
only on principal observation of attemptFinished. A terminal commit received on
attemptStore's queue still occupies inflight until the worker finishes. Output
collection/validation and callbacks must finish before that observation. Conversely,
worker return without terminal facts does not release durable prepared/running;
retain unknown/recovery treatment, not guessed success. No-process Build/spawn
failures need their real terminal fact plus worker finish; materialization failure
before worker registration retains prepared/unknown durable facts and existing
stop semantics. Snapshot/commit error after preparation must not spawn or leak a
fake free slot. CAS/fence refusal never produces a spawn from stale request state.

Recompute after each admitted attempt and each serviced commit/finished event;
when one of two slots is genuinely released, a third independent root can start
while the peer remains active if existing controls permit. No whole-batch barrier,
polling sleeps or fairness/parallel-coordination subsystem. Normal saturation
waits on existing events; true failure/cancel stops new admissions while peers
drain through the existing commit queue. Preserve first-error handling and lease
ownership until safe drain. Integration stays serialized under the principal's
existing no-inflight condition; free capacity cannot bypass validated dependencies,
required integration, gates, unresolved effects or failure/cancellation.

## Causal implementation sequence and handoff boundaries

This is ordering for future TASKS, not a Tasks artifact or execution dispatch.

1. Freeze baseline corpus/oracles first, audit literals/callers and current tests.
   Prove the proposed omission/hash/strictness path before dependent mapper work.
2. Add optional snapshot member and validation; prove codec/clone/digest/store
   frozen-equality, CAS and canonical inventory compatibility. Keep absence intact.
3. Apply shared capacity in Ready and fenced preparation including retry; connect
   principal attempt identities, terminal/finished lifetime and replenishment in
   runReady/prepareAttempt. Preserve Claim/recovery and all control boundaries.
4. Map exact declared values, including default/new sequential 1, only with
   enforcement available. Preserve ordered edges/controls and independent refusals.
   Adjust new-behavior fixtures explicitly, leaving historical corpus unchanged.
5. Run deterministic contract/regression checks, engineering/security review and
   reconcile Evidence/docs against the final implementation SHA. Do not declare
   completion from mapping alone. Future Tasks must consume the qualified Plan
   and name these dependencies/oracles without expanding AC or architecture.

## Verification matrix (all implementation cases NOT_RUN)

Use existing local stores, stubLauncher/taskLauncher, fakeRunner, observedWriter,
controlled collectors and explicit channel barriers. Deadlines only bound a hung
test; elapsed time or sleeps never prove concurrency. Record admitted identities,
spawn events, terminal commits, finished observations and peak union occupancy.

| DC / AC | Fixture, owning tests and observable oracle |
| --- | --- |
| DC-01; AC1/2 | CLI execution_run_test mapper/public-run fixture: valid DAG 2 A/B independent, C declared validation/integration owner joining both. Exact 2/edges/controls frozen; one canonical Execution created. Both stub start acknowledgments arrive before either release: actual overlap. C stays unprepared until both validated and required integration complete. Start/overlap is not real inference or product completion. |
| DC-02; AC2 | scheduler_test three roots A/B/D plus owner join C, limit 2: hold two at barriers, record principal scheduling checkpoint and no third prepare/spawn; release one and observe D while peer held. Peak union <=2 at every admission/commit/finish; join waits for all three. Checking Ready output alone fails this oracle. |
| DC-02/03; AC2/3 | Hold Collect/output validation; then hold worker after terminal commit using a test-only worker-side post-commit barrier that leaves the principal free to schedule/service commits. Third stays unprepared until finished observed. Hold Build failure before spawn: zero process, durable disposition plus worker end required; failure still stops subsequent work, so capacity release grants no launch. Include post-prepare snapshot/materialization/commit errors and no slot leak or blind replay. |
| DC-01/03; AC3 | Definition/CLI table: 0, negative, fraction, >32, >agent count, sequential 2, cycle, dangling dependency, invalid owner; no Execution or spawn on pre-start refusal. DAG 2 with unsupported validator or missing required context still refuses for its independent reason. |
| DC-01/03; AC3 | DAG 1 independent roots and sequential 1 chain, public start and domain/scheduler tests: peak <=1, declared edges unchanged, gate/output/timeout/budget behaviors retained. |
| DC-04/05; AC2/3 | execution_test + frozen #404 corpus: old exact bytes/hash, nil clone/round-trip; explicit 1/2 hashes/round-trip; malformed wire null/0/negative/fraction/overflow/excess/duplicate/mis-case/unknown, removed required members and tampering refuse. Old reader refuses new bytes; no downgrade stripping. |
| DC-04/05; AC2/3 | execution_store_test actual load/update/reload: unchanged Workflow/digest on progress; changed/removed limit even with recomputed digest refused; stale CAS/fence cannot commit or spawn. Canonical-location inventory and append-only stable corpus retain recognition. |
| DC-02/04; AC2/3 | Restart explicit 2/absent pending and terminal snapshots; mutate current Workflow selection to prove no lookup/rebinding. Historical multi-active loads, no additional admission at fallback 1. Claim orphans to unknown, no replay; permitted explicit reconciliation/retry keeps absence, budget and invocation and rechecks union capacity. Include stale generation and simultaneous admission attempts. |
| DC-03; AC2/3 | Reuse scheduler stops, engine cancellation/unknown, exchange, integration/workspace and delivery tests: failure/cancel stops new work, peers drain truthfully, stale callback cannot write; missing outputs/unintegrated/unknown results do not unblock join. Gates, nonconflicting effects, writer isolation, serialized integration, timeouts and frozen retry remain. A ceiling never authorizes conflicting writers. |
| DC-01–05; AC4 | Evidence per SHA/command/exit/fixture/count/observation on same PR/Issue, PASS/FAIL/SKIP/NOT_RUN/UNVERIFIED separated. Docs/CI success never marks implementation AC PASS. |

Future commands, **NOT_RUN in PLAN**: targeted `go test ./internal/workflow
./internal/strictjson ./internal/local ./internal/stageexecution ./internal/workflowdefinition
./cmd/lingo` with focused `-run` selections for the cases above, plus
`go test ./internal/compatibility`; then `go test ./...`, `go vet ./...`,
`go build ./...`, `go mod verify` and `./scripts/validate-repository.sh .`.
Use targeted `go test -race ./internal/stageexecution ./internal/local` for
admission/commit races on supported host. No new dependency, real Runtime or
external Repository/Provider mutation. Report unavailable checks as such.

## Rollout, rollback, security and stop criteria

No operational rollout in PLAN. Future code stays on this PR until separately
qualified; first use of the amended writer can create new-member snapshots.
Code rollback must preserve those bytes and use a reader that understands them,
or leave them inert until the fixed reader returns. A pre-change binary must
not consume/strip/rewrite new snapshots. No automatic migration, old-attempt
restart, immutable Workflow edit or downgrade is authorized.

Principal risks: accidental eager defaults/hash drift, pointer aliasing, stale
Ready/retry over-admission, early release at terminal commit, stranded preparation,
and tests that silently relied on absent unlimited parallelism. Bound scan cost
by existing 32-agent/attempt limits; no performance subsystem. The deterministic
oracles above target these risks. Historical fallback can reduce throughput;
this is specified behavior, not permission to infer lost intent.

Retain security controls at Axiom-mediated boundaries, exact target/effect grants,
writer workspaces, environment filtering, rooted storage, lock order and protected
publication. Worktrees/effect labels do not provide OS confinement under ADR-0039.
No logs, secrets, lab files or host credentials enter fixtures/Evidence. Models,
new permissions, spending and delivery effects require their own authority.

Stop dependent work if omission/digest/frozen progress cannot be preserved,
incompatible migration/format bump or new durable ADR/owner is necessary, or
accepted controls would need relaxation. Preserve safe independent work and report
the exact blocked action, source/test evidence, viable alternatives and human
decision required. No self-approval, new ADR, synthetic dependency workaround or
scope expansion. Remote divergence/credential/push failure means preserve local
commits and PARTIAL/BLOCKED; never force or overwrite.

## Constitution, planning Evidence and next gate

Constitution I–VIII check: accepted intent precedes implementation; scope stays
within DC-01–05; one canonical owner, Project/Repository distinction, deterministic
oracles, preserved history, exact authority and explicit compatibility stop gates
remain. Local representation/helper choices introduce no material architecture
or product decision. Documentation governance: issue-specific Plan under Spec 007,
no overwrite of another delivery Plan, lifecycle index or accepted historical text.
No optional deferred finding or indispensable new ADR identified by source review.

Only planning checks run in this attempt: repository/document governance, links,
whitespace, staged paths/content and sensitive-file checks, with actual results in
the completion receipt on Issue #404. Dedicated scanner availability is reported
separately. Implementation/codec/resume tests, Go regression/vet and real inference/
T12 battery 03 remain **NOT_RUN**; behavior and executable compatibility remain
**UNVERIFIED**. No implementation AC is PASS from this Plan.

Planning Evidence: `./scripts/validate-repository.sh .` exited 0 (governance/
bootstrap only; Runtime behavioral scenarios explicitly skipped). Plan relative
Markdown targets/anchors, `git diff --cached --check` and
`./scripts/check-sensitive-files.sh --staged .` passed with exit 0. Gitleaks is
unavailable (`command -v gitleaks`, exit 1): dedicated scanner SKIP. Staged
content and source-connected Plan review under Axiom/Ponytail found no blocking
Plan/security or accepted-ADR conflict; this is not executable compatibility proof.

Next gate: orchestration qualification of this versioned Plan, then proportional
TASKS with sufficient Evidence. No Tasks/Implement/continuation dispatch here.
Checkpoint **AWAITING_EXECUTION_REPORT** pending consumption of the Issue receipt;
comment publication is not an authenticated orchestration continuation.
