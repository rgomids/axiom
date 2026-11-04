# Tasks — #369 bounded pre-spawn launch correction

Revision 1, 2026-11-04. Stage: TASKS / attempt-01. **Eight tasks, all PLANNED**;
none implemented or completed. Design: **NOT_APPLICABLE** (CLI; no graphical UI).

## Authority, baseline and four pillars

One [Work Item #369](https://github.com/rgomids/axiom/issues/369), branch
`fix/369-launch-refused`, [draft PR #370](https://github.com/rgomids/axiom/pull/370).
This decomposition consumes the approved [amendment revision 1](amendment-369-pre-spawn-launch.md),
[Plan revision 1](plan-369-pre-spawn-launch.md), [Plan Evidence](evidence-plan-369.md),
[human baseline](https://github.com/rgomids/axiom/issues/369#issuecomment-6621735082),
[Specification and HD-369-D approval](https://github.com/rgomids/axiom/issues/369#issuecomment-6621938832),
and [Plan receipt](https://github.com/rgomids/axiom/issues/369#issuecomment-6622099603).
PR review [5763007250](https://github.com/rgomids/axiom/pull/370#pullrequestreview-5763007250)
favors Tasks, without merge authority. Historical pending wording in Specify
is superseded only by that explicit HD-369-D approval; approved bytes stay intact.

Before writing: clean worktree; local, remote branch and PR HEAD all
`9fad368ded09fc2f70e0b88d23fef5c4588869e7`; fetched main and PR base SHA both
`375960907ce1ba4799830e2f0a90ae978598ea23`. PR open/draft, same repository.

| Pillar | Check and consequence |
| --- | --- |
| Work Item / AC / non-goals | A: bounded valid TOML versus selectors; B: persisted closed redacted failures; C: pure effective diagnosis; D: prospective lifetime refusal accounting. All four required. No implementation in this stage. |
| Product intent | Truthful operational control and explicit recovery under the [constitution](../../product/constitution.md); readiness is not authentication or inference. Project != Repository, Runtime != Model, Execution != Agent. |
| Architecture / security | Preserve [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md), retained ADR-0024 / Spec 007 controls, strict format 4, one aggregate/store/engine, append-only history, frozen selectors, bounded output, source/path identity and double reads. [Security](../../../.agents/policies/security.md) and [quality](../../../.agents/policies/quality.md) gates apply. No new ADR or material change is proposed. |
| Design | NOT_APPLICABLE: CLI; route/help/text/JSON consistency remains required engineering work. |

## Execution rules shared by every task

Owners below are accountable implementation roles/packages, not new agents or
assigned people. Repeat targeted caller/test/fixture/config/export discovery from
the Plan inventory before edits; use `GRAPHIFY_BACKEND=none` with the external
Axiom graph first, then `rg` (or grep if unavailable). Confirm current sources;
graph edges are navigation aids. Keep all work on the canonical branch/PR.

Each task requires its dependencies' reviewed outputs and passing targeted checks.
Record a task receipt in a later `evidence-implementation-369.md`, section matching
the stable ID: baseline/commit, changed paths, AC/Plan mapping, exact commands and
exit results, fixture names, positive/negative observations, review findings,
limitations and status. That file is an expected implementation output, **not
created or claimed as Evidence here**. Failed/unexecuted checks stay FAIL/NOT_RUN.
Tests below are future commands, not Tasks-stage results. Use synthetic temporary
HOME/managed/Project/repository roots, fake runners and fake executable identities;
inspect subprocess fixtures before running suites. Never read personal configs,
credentials or invoke vendors, auth probes, inference or network from tests.

All tasks exclude: new ledger/store, auto retries, auth prerequisite, r2 budget or
definition/digest changes, historical refunds/reclassification, profile-wide scans,
external Project mutation, Claude model-specific exception, merge/release/deploy,
acceptance or Issue closure. Never modify frozen historical corpus bytes. No
ceremonial APIs, generic validation framework or speculative options are needed.

Stop the affected task on a broken resource bound, raw-data disclosure, historical
incompatibility, identity/fence/authority bypass, impure diagnosis or requirement
for a material contract change. Preserve its Evidence and report on #369; continue
independent authorized work. A material change requires human decision. A local
implementation correction does not. Task completion requires executable evidence
and review, not the existence of this checklist. Do not start P1 in this run.

## Dependencies and execution order

| ID | Depends on | Causal edge / independently reviewable outcome |
| --- | --- | --- |
| I369-P1 | Approved Plan | Freeze old bytes, actual old-reader behavior and budget oracle before additive writers. |
| I369-P2 | P1 | Closed failure schema must respect proved compatibility constraints. |
| I369-P3 | Approved Plan + dependency proof | Parser replacement is independently verifiable; no B/C/D prerequisite. |
| I369-P4a | P2 | Pure assembly/inspection uses stable typed failure codes; can use current parser until P4b. |
| I369-P4b | P4a, P3 | Public diagnose needs pure composition and corrected TOML behavior. |
| I369-P5a | P1, P2 | Accounting schema/transitions need baseline and canonical failure qualification. |
| I369-P5b | P5a | Engine/store must enforce the same durable transitions and recovery. No parser or diagnose dependency. |
| I369-P6 | P3, P4b, P5b | Integrate all AC, final dispatch parity, docs/installed contracts and regression Evidence. |

```text
Plan ── P1 ── P2 ── P5a ── P5b ──┐
                 └─ P4a ── P4b ──┼─ P6
Plan/proof ── P3 ────────────┘     │
              └──────────────────┘
```

Safety-critical path: P1 → P2 → P5a → P5b → P6 (old-reader proof through durable
reservation/recovery). P1 → P2 → P4a → P4b → P6 is a converging path of equal task
count; elapsed critical path depends on execution duration, not estimated here.
P3 runs independently of P1/P2/P5. After P2, P4a and P5a can proceed independently;
P4b and P5b can proceed independently when their own inputs pass. Coordinate shared
`cmd/lingo` and launcher edits; parallel eligibility does not authorize conflicting
unreviewed writes. No agent delegation is required by this graph.

## I369-P1 — Compatibility and accounting baseline

**Objective / mapping:** P1, AC-B/D. Establish immutable historical and downgrade
oracles before changing writers. **Owner:** workflow/local/compatibility test owner.
**Preconditions:** verified approved baseline and Plan; no production schema edit.

**Scope:** `internal/workflow/execution*_test.go`, `internal/local/execution_store_test.go`,
`historical_execution_fixtures_test.go`, `internal/compatibility` and existing testdata.
Discover strict codec consumers in `internal/strictjson`, CLI admission and engine
fixtures from the Plan inventory. Read baseline decoder/source into an isolated
local harness pinned to `9fad368ded09fc2f70e0b88d23fef5c4588869e7`; record source
SHA and harness command. Reuse fixtures; do not introduce a production legacy reader.

**Outputs:** immutable format-4 legacy terminal/active bytes with digests; candidate
new-policy/failure bytes; a real compiled baseline decoder harness and effect spies;
a budget transition table used as an implementation oracle; joined-error/cancel
baseline assertions. Store only sanitized reproducible test material.

**Acceptance:** current reader accepts every legitimate old record; absent policy
retains record-count budgeting. Actual old reader rejects proposed extra policy and
failure members before any runner/prepare/lease/network effect or directory change;
strict unknown fields/policy fail closed. Candidate-byte proof is rerun against
real writer bytes in P6. Old terminal byte round trips and historical budget
exhaustion remain unchanged. Cover maximumAttempts 1, 2, 10; zero/one/two refusals,
charged exhaustion and max+2 records; pending conservatively occupies a charged
slot. Joined Invalid+Unsafe and cancellation establish the classification oracle.

**Checks / negatives:** `go test ./internal/workflow ./internal/local ./internal/strictjson ./internal/compatibility -count=1`;
`go test ./internal/stageexecution -count=1`; execute and record the pinned isolated
old-reader harness. Include the quality-policy inventory/canonical-location and
append-only corpus guards, unknown fields/enums, malformed bytes and zero-effect
admission rejection. No unknown field may be silently dropped before validation.

**Evidence:** P1 receipt includes fixture hashes, baseline source/compile command,
old/new decode results, zero-effect snapshots and counter examples. **Non-goals /
stop:** no runtime change, snapshot rewrite or migration; inability to prove
format-4 safety blocks P2/P5 writers, not P3 research/integration preparation.

## I369-P2 — Typed launch failures through canonical projections

**Objective / mapping:** P2, AC-B. Preserve precise redacted failure facts from
adapter/launcher to attempts, CLI/status and reconciliation. **Owner:** workflow,
runtimeadapter, stageexecution and CLI projection owner. **Precondition:** P1 proof.

**Scope:** `internal/workflow/execution.go` and transitions, `internal/runtimeadapter`
error boundaries, `internal/stageexecution/runtime_launcher.go`, `engine.go` and
`Reconcile`, `internal/local/execution_store.go`, `cmd/lingo/execution_run.go` and
existing output tests. Search every Build return and consumer before editing.

**Outputs:** closed reason/phase/source values and typed Is/As-compatible errors;
optional canonical launchFailure encoding with omitempty; strict enum validation,
history immutability and clone safety; static remediation and broad diagnostic
compatibility; tests for every exit/sink. No string-based error classifier.

**Acceptance:** use exactly Plan B reasons/phases/source classes (including omitted
unknown source). Cancellation wins; joined Invalid precedes Unsafe in either order;
specific typed selector/identity/path/permission/executable causes precede broad
unsafe. Permission needs typed evidence. Unknown builder failure is
builder_failed/invocation_build, never eligible for deterministic refusal. Spawn and
runtime failures remain distinct, with no inferred login failure. Preserve
errors.Is(ErrRuntimeLaunch), broad launch_refused and existing adapter sentinels.
Persisted facts survive codec/status/reconcile without reparsing Diagnostic.
Old records with no detail remain unavailable, without backfill. Detail added to
active legacy-policy bytes must explicitly test/document old-reader refusal while
retaining legacy accounting; terminal historical bytes stay untouched.

**Checks / negatives:** `go test ./internal/runtimeadapter ./internal/workflow ./internal/stageexecution ./internal/local ./internal/cli ./cmd/lingo -count=1`.
Round trip every code and reject unknown enum/field. Inject canary paths, tokens,
values, key names and child output into nested/joined causes; assert absence from
persisted JSON, CLI text/JSON and captured logs. Test cancellation, ambiguous I/O,
unclassified builder error, all Build exits, reconciliation and protected rewrites.

**Evidence:** P2 receipt maps Build exits → code → sink with redaction results and
P1 compatibility replay. **Non-goals / stop:** no accounting-policy writer, raw error
serialization or alternate log store; stop on leakage or broad compatibility loss.

## I369-P3 — Bounded standards TOML decoding and selector traversal

**Objective / mapping:** P3, AC-A (typed integration AC-B). **Owner:** runtimeadapter.
**Preconditions:** approved exact dependency/source proof from Plan Evidence; repeat
checksum/advisory/license review before adoption, without changing the pinned choice
silently. Independent of P1/P2; retain existing adapter error contracts until P2 joins.

**Scope:** `go.mod`, `go.sum`, `internal/runtimeadapter/effort_sources.go`,
`effort_v4.go` and parser/operational/frozen invocation tests; discover every
`tomlEffortSelectorsBounded`/`observeEffortFacts` and nestedProfiles caller.

**Outputs:** exact `github.com/BurntSushi/toml v1.6.1-0.20260818200445-d733fc535e4a`,
explicit `Decoder.MaxDepth(64)` before Decode, bounded iterative semantic walk;
remove superseded narrow grammar; port Plan's isolated proof into repository tests.
Preserve source-bound review references and fixtures, not only happy-path decoding.

**Acceptance:** decoder owns syntax/duplicates; parser-time combined recursion <=128
before excessive descent/allocation, including dotted/table paths. MaxDepth(64)
bounds separate array/table counters conservatively; acceptance at depth128 is not
required. Semantic depth <=32, nodes <=65,536, counting root/containers/keys/scalars
consistently; 1 MiB input and 64 managed drop-ins remain enforced. Valid bounded
arrays, multiline arrays/strings, inline tables, arrays of tables and dates pass
independently of selector detection. Quoted/dotted/nested selectors conflict even
at equal values; comments/string mentions do not. nestedProfiles=false rejects
profiles at every depth/form. Preserve rooted no-symlink paths, identity/binding,
double-read/inventory drift checks. Malformed/duplicate is invalid; resource/source
excess unsafe; no raw decoder context escapes.

**Checks / negatives:** `go test ./internal/runtimeadapter -count=1`.
Port array64/65/129/500000, mixed inline-array, dotted129 and duplicate proof cases;
add combined128/129, inline/table contexts, semantic32/33 and65536/65537,
1MiB/over-limit,64/65 drop-ins, malformed UTF-8/tables, replacement between reads,
symlinked ancestor/file and nestedProfiles=false cases. notify alone passes;
notify+effort conflicts. Keep Claude Sonnet modelSettings effort with frozen Haiku
as selector_conflict. Ensure hostile inputs refuse without panic/unbounded recursion.

**Evidence:** P3 receipt records module/checksums/source guard, resource fixture
results, caller coverage and security regressions. **Non-goals / stop:** no custom
pre-parser, runtime Python, unstable AST or Claude exception; stop parser adoption
if the source guard or hostile-input proof fails; independent B/D work continues.

## I369-P4a — Pure invocation inspection boundary

**Objective / mapping:** P4, AC-C core. **Owner:** runtimeadapter/stageexecution
composition owner. **Precondition:** P2 closed failure contract; P3 not required.

**Scope:** `internal/runtimeadapter/stage_invocation.go`, `frozen_invocation.go`,
`effort_v4.go`, `operational_source.go`; `internal/stageexecution/runtime_launcher.go`;
read-only path derivation seams in `cmd/lingo/execution_run.go`. Inspect Build,
Collect, effortInvocation and invocation assembly callers; reuse existing pure APIs.

**Outputs:** minimal separation of planned path/argv/env/CWD assembly and bounded
selector inspection from materialization; dispatch retains actual path checks.
Tests establish the pure boundary against existing parser behavior.

**Acceptance:** same frozen Runtime/model/effort/instructions, roots/environment and
CWD rules feed inspect and dispatch. No MkdirAll, artifact-store construction,
Build call, command validation/collection, subprocess, network, provider, lease,
CAS/store writer or install/recovery writer in inspection. Canonical read-only loads
are allowed. Absent planned inbox/CWD yields not_evaluated for materialized checks;
existing ancestors may be inspected read-only. Inaccessible is not assumed absent;
partial results cannot become admissible; known conflicts can still be reported.
Authentication remains not_evaluated, inference not_run. Final actual dispatch
rechecks are preserved; no cached inspection receipt/admission token.

**Checks / negatives:** `go test ./internal/runtimeadapter ./internal/stageexecution ./cmd/lingo -count=1`.
Use before/after byte/mode/tree snapshots and fail-on-call spies for each prohibited
effect; absent inbox/roots, permission denial, cancellation and source replacement.
Assert tuple parity, child markers and bounded output. P4b/P6 repeat with P3 and
new accounting; P4a alone does not claim complete AC-C.

**Evidence:** P4a receipt documents call boundary, spies/snapshots and same-tuple
fixtures. **Non-goals / stop:** no CLI route yet, auth probe or redesign of resolver;
stop on filesystem mutation or silently weakened dispatch checks.

## I369-P4b — Read-only diagnose route and truthful scope projections

**Objective / mapping:** P4, AC-C public contract, AC-A/B integration. **Owner:**
cmd/lingo, cli, runtimeapplication, projectapp. **Preconditions:** P4a and P3 pass.

**Scope:** `cmd/lingo/execution_run.go`, `internal/cli/workflow_stage.go` command,
requirements/help/SkillSpecs/JSON tests; `internal/runtimeapplication/invocation.go`,
`authoring.go`; `internal/projectapp/readiness.go`; associated recorder fixtures.

**Outputs:** existing WorkflowStageService route `workflow stage diagnose` selecting
Project/Execution/stage/agent and explicit attempt when ambiguous; explicit
`--new-selection` preview via ResolveInvocation; five separate scope dimensions,
help/text/JSON metadata and readiness/profile disclosures.

**Acceptance:** default uses persisted frozen attempt/provenance despite changed
defaults. Preview uses same resolver/flags as explicit retry, labels prospective
selection, never rebinds history or grants retry authority. Structural readiness
and configuration eligibility retain their meanings and disclose invocation
not_evaluated plus diagnosis guidance. Invocation is admissible/refused/not_evaluated;
authentication=not_evaluated, inference=not_run. Safe provenance omits host/private
paths, raw instructions/config/environment. Route bypasses effectful CLI launcher
construction and run/retry/admit. Missing/ambiguous context returns safe explicit
guidance; no profile-wide scan or mandatory diagnostic preflight.

**Checks / negatives:** `go test ./internal/projectapp ./internal/runtimeapplication ./internal/cli ./cmd/lingo -count=1`;
`go test ./internal/runtimeadapter ./internal/stageexecution -count=1`.
Cover route/flags/requirements/JSON/text parity, ambiguity, frozen versus preview
selection, invalid config/selector and Claude case; reuse all P4a no-effect spies
through the actual CLI entry, including missing output directories. A previous
admissible result must not bypass dispatch after changed sources/authority.

**Evidence:** P4b receipt includes CLI projections and full entry-point tree/byte
snapshots, no-effect counts, five dimensions and dispatch parity. **Non-goals /
stop:** no credentials/vendor status or persisted diagnosis; stop if any dimension
implies auth/inference success or a preview mutates/rebinds an attempt.

## I369-P5a — Prospective policy and bounded canonical transitions

**Objective / mapping:** P5, AC-D aggregate contract. **Owner:** workflow/local codec
and history owner. **Preconditions:** P1, P2; no P3/P4 dependency.

**Scope:** `internal/workflow/execution.go`, `execution_transitions.go` including
Validate/PrepareAttempt/Ready/retry projections; `internal/local/execution_store.go`
protected history and cloning; inventory/current-writer/strict codec tests.

**Outputs:** frozen new-Execution policy `spawn-boundary-v1`; optional attempt
accounting pending/charged/deterministic_refusal; required new-policy metadata,
reservation/no-spawn/lineage transitions and protected history tests. Implement
reader/validator/history support together; do not enable a partially supported writer.

**Acceptance:** absent policy uses entire legacy state machine, unknown nonempty
policy refuses. New writers alone freeze new policy. Counts derive from bounded
canonical attempts; no duplicated counter/store. Pending consumes budget
conservatively; charged never becomes refusal. maximumAttempts<=10 unchanged;
records<=maximumAttempts+2; prepare denied if charged budget or two lifetime
refusals exhausted, with no third append. Two slots never reset on selection,
restart/resume/status or successful spawn. New attempts carry distinct ID/ordinal,
parent (first absent), actor/time, generation, frozen invocation/provenance,
authority and preserve workspace disposition. Refusal requires closed deterministic
validator failure, engine no-spawn assertion and materialization disposition;
unknown builder/cancel/revocation never qualifies. Freeze policy/accounting/lineage
and reason/phase/source once recorded; preserve clone isolation for any new pointer
or slice fields. Historical terminal bytes/budgets and active legacy policy unchanged.

**Checks / negatives:** `go test ./internal/workflow ./internal/local ./internal/strictjson ./internal/compatibility -count=1`.
Use P1 max1/2/10 tables, missing mandatory metadata, unknown enum/policy/field,
max+2 and third-refusal denial, no reset, illegal charged→refusal transition,
history rewrites and snapshot aliasing. Replay old bytes and old-reader candidate
rejection; inventory current writer/canonical location and frozen corpus unchanged.

**Evidence:** P5a receipt includes state transition/count truth table and actual
writer byte compatibility results. **Non-goals / stop:** no engine activation,
new format/ledger or historical reclassification; stop writer if strict format-4
or legitimate old-state compatibility cannot hold within approved contracts.

## I369-P5b — Fenced charged reservation, explicit retry and crash recovery

**Objective / mapping:** P5, AC-D engine and recovery; AC-B persistence. **Owner:**
stageexecution/local and cmd/lingo retry owner. **Precondition:** P5a passes.

**Scope:** `internal/stageexecution/engine.go` runAttempt/commit/recordInterrupt/
watchCancel/Reconcile, `scheduler.go`; `internal/local/execution_store.go`,
`recovery.go`; `cmd/lingo/execution_run.go` run/retry/existingLauncher and tests.

**Outputs:** engine-only deterministic no-spawn commit; durable charged reservation
before Runner; explicit retry lineage/authority/workspace handling; coordinated
scheduler/readiness/cancel/reconciliation and crash/fence tests. Enable prospective
new-Execution writes only with complete reader/store/engine support.

**Acceptance:** after Build/materialization, final frozen selector/path/identity,
authority/cancel/generation checks precede reservation/Runner. Only successful
UpdateFenced reservation permits exactly one RunObserved call. CAS/fence failure
permits none. Known deterministic pre-call failure commits terminal refusal with
local materialization fact; unknown builder/cancel/revocation consumes charged
budget conservatively without claiming spawn. Refusal->corrected attempt needs
explicit authorized retry, new ID/parent/frozen provenance and workspace preserve
disposition; new-selection never edits prior snapshot. Unknown outcomes require
existing explicit resolution and safe workspace disposition before retry. Active
foreign lease, stale revision/generation or unconfirmed effects refuse. Exhausted
refusal allowance requires a fresh explicitly authorized Execution, no inherited authority. Scheduler
never loops automatically over refusals or unknown attempts.

Crash before prepare commit publishes no attempt; after prepare before resolution,
pending becomes unknown and counts conservatively; after reservation, before Runner
or start callback, remains charged unknown. Lost terminal/command output retains
unknown effects/process identity safeguards. Never infer no-spawn from absent PID,
refund unknown or replay automatically. Confirmed spawn failure and runtime failure
remain charged (phase spawn/runtime); missing inference is not free credit.

**Checks / negatives:** `go test ./internal/workflow ./internal/stageexecution ./internal/local ./cmd/lingo -count=1`;
`go test -race ./internal/workflow ./internal/stageexecution ./internal/local -count=1`.
Inject crashes before/after prepare, reservation, Runner/start callback and terminal
save; assert durable bytes/call counts. Test lost CAS/stale lease/fence/generation,
cancel/revocation at boundaries, repository/source/executable replacement, retry
lineage/workspace disposition, one then two lifetime refusals across success and
restart, exhausted budgets and historical default-r2 behavior. No real vendors.

**Evidence:** P5b receipt includes crash timeline table, durable reservation before
call proof, retry authority and no replay assertions, race results. **Non-goals /
stop:** no auth gate, budget increase or historical rescue; stop on any possible
Runner call without committed charge or stale authority/fence.

## I369-P6 — Integrated AC matrix, documentation and final Evidence

**Objective / mapping:** P6, AC-A–D. **Owner:** integration/CLI/docs/compatibility
review owner. **Preconditions:** P3, P4b, P5b passing receipts; all other tasks transitively.

**Scope / outputs:** cross-package synthetic engine/CLI integration tests; updated
`docs/commands.md`, canonical CLI help/docs metadata and
`internal/codexruntime/skills/axiom-work-item/SKILL.md`; discover installed/copied
skill consumers and renderer contracts via existing install/codexruntime tests.
Final `evidence-implementation-369.md` maps every AC/Plan matrix row to test/fixture,
result and commit, plus security/engineering review and rollback limits. Do not edit
historical Specify/Plan to claim behavior. Documentation may be drafted alongside
implementation, but final parity waits for merged task outputs on this branch.

**Acceptance:** full Plan verification matrix covered: TOML syntax vs selector,
typed failure through actual persisted attempts/status/reconcile, pure CLI diagnose
against frozen/prospective dispatch, charged vs refusal lifetime budgets under
crash/race/authority changes. Rerun actual baseline decoder harness on actual new
writer bytes; reject before all effects, including active legacy failure additions.
New reader accepts old corpus, strict unknown policy/fields refuse, inventory guards
pass and no frozen byte is edited. Cross-package canary redaction and no-effect
negative checks pass. Installed skill, help, docs, text/JSON disclose five scopes,
explicit retry and downgrade refusal. No field stripping or historical refund as
rollback: after new-policy writes retain capable reader or stop affected executions.
No material ADR conflict; no blocking engineering/security review finding unresolved.

**Checks:** after inspecting subprocess fixtures, run:

```sh
go test ./internal/runtimeadapter -count=1
go test ./internal/workflow ./internal/stageexecution -count=1
go test ./internal/local ./internal/strictjson ./internal/compatibility -count=1
go test ./internal/projectapp ./internal/runtimeapplication ./internal/cli ./cmd/lingo -count=1
go test ./internal/install ./internal/codexruntime -count=1
go test -race ./internal/workflow ./internal/stageexecution ./internal/local -count=1
go test ./... -count=1
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
./scripts/check-sensitive-files.sh --staged .
git diff --cached --check
```

Also run the P1 harness using its recorded exact command and documentation local
link/marker assertions. Record platform/CI observations explicitly; unavailable
checks are NOT_RUN, never inferred passing. New writer coverage must exercise
`TestEveryV1WriterIsRecognizedByInventory`; release snapshot freezing remains a
later separately authorized release operation, not a task here. No AI/vendor battery.

**Evidence:** final matrix and task receipts distinguish unit/integration/source
proofs, CI pending results, limitations and unresolved risks. Apply Axiom review
and Ponytail-review against actual connected consumers. **Non-goals / stop:** no
release/merge/acceptance/closure task; an unproved AC or failing safety negative
blocks implementation-completed claims and only the dependent later gate.

## Tasks-stage decisions, review and handoff

technical_decisions[]:
- TD-369-T1 (TECHNICAL_LOCAL): eight tasks retain P1–P6; split P4 at pure core/public
  route and P5 at aggregate/engine boundary because each has distinct tests and
  review owners. Fewer tasks would hide effects/codec gates; finer splits add no
  independently useful acceptance boundary.
- TD-369-T2 (TECHNICAL_LOCAL): P3 is independent; P4a can establish purity with the
  current parser, while P4b/P6 require P3. B/D have no parser dependency. This
  preserves Plan integration requirements without artificial serialization.
- TD-369-T3 (TECHNICAL_LOCAL): one later implementation Evidence file with per-ID
  receipts rather than eight tracking files; no new automation/store. Actual old
  reader harness is local/test-only, source-pinned, then rerun on real writer bytes.

deferred_findings[]: empty. No indispensable out-of-scope change identified.

Tasks review checks AC coverage, dependency causality, owners, executable commands,
negative tests, authority and stop conditions; see [Tasks Evidence](evidence-tasks-369.md).
This review is of decomposition, not implemented behavior. Next gate: **implement**,
after Tasks review/gates; do not begin it in this Tasks-only run. Later implementation
must still satisfy review/reconciliation and separate human publication authority.
