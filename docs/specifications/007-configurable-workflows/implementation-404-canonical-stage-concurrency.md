# Issue #404 — IMPLEMENT-404-01 Evidence

Scope: [approved tasks T1–T6](tasks-404-canonical-stage-concurrency.md),
[DC-01–DC-05](amendment-404-canonical-stage-concurrency.md), AC1–AC4 in
[the existing Intake](https://github.com/rgomids/axiom/issues/404#issuecomment-7132349983).
Authority remains the existing Issue decisions and IMPLEMENT gate; no new
acceptance, ADR, merge, release or operational rollout is implied.

## Canonical source and causal checkpoints

Initial clean worktree `/home/grokbot/axiom-wt/gh-404-specify`, origin
`https://github.com/rgomids/axiom.git`, branch
`issue/404-specify-dag-concurrency`, local/remote `045c4072d386469c64322564e4baa6b90b4ff22e`,
base/main `5cd783bd2ac8189a1b9c85a790472b5335139c78`,
[PR #405](https://github.com/rgomids/axiom/pull/405) OPEN Draft were verified
before edits. No competing change was observed; this is not a claim that other
sessions cannot act. No new branch/worktree/PR or remote was created.

- T1: commit `8a3f14c702f74537e3ee7df50a65ecb954f1b117` freezes six full old
  format-4 Executions, exact digest inputs and independent explicit-1/2 inputs.
  Generation ran in an archive of `15d0f87911b758636a922cd9ab9777ed5d05046f`,
  before any codec edit. Both generator round-trip and principal focal rerun
  passed. [Recipe and old-reader probe](../../../internal/workflow/testdata/i404/PROVENANCE.md).
  [Literal file hashes](../../../internal/workflow/testdata/i404/manifest.json);
  manifest SHA-256 `bea909faa01160daf0256baee9ad123def0ce9b232bdbfa5778662e150cd5ac8`.
- T2: commit `9882edb` appends optional `Concurrency *int` without changing old
  member order/format. Explicit limits validate 1..32 and agent count. New
  defaults declare 1; retained absence stays absent. Codec, independent digest,
  pointer, strict-shape, store progress/restart/frozen-equality/CAS/fence tests
  passed before T3. Real reader/store/engine/reconcile compiled from pinned
  archive `15d0f879` refused actual new explicit-1/2 bytes before effects, without
  lease/directory/byte changes.
- T3: commit `2c88c01` applies one capacity calculation to durable Ready and
  preparation, including retry. Occupancy is deduplicated durable prepared/running
  UNION correlated principal inflight IDs. Domain tests passed for cached Ready,
  invalid correlation, terminal inflight, retry without consumption, historical
  multi-active and unknown/fence/cancel precedence. Three simultaneous fenced
  preparations admitted exactly two and refused one; no third attempt exists.

Checkpoint tests were executed against their corresponding staged/working source
changes before committing. Final qualification below identifies a stable tested
commit, rather than treating a dirty-tree command as proof for a later SHA.

## Local technical decisions

- Optional pointer appended to StageSpec reuses the existing strict JSON codec
  and JSON clone. Alternative eager defaulting would change historical bytes and
  digest; format bump/rewrite is outside the approved compatibility contract.
  Risk: old binaries refuse new-member snapshots. Pinned old-reader refusal is
  intentional; no downgrade stripping or migration is offered.
- Effective absent ceiling 1 is derived only at admission. Historical records
  with several active attempts remain readable; Claim records orphans unknown,
  without replay. Existing selection is never consulted to reconstruct a limit.
- Principal-owned task/attempt correlations extend PrepareInput only in memory.
  Existing UpdateFenced/CAS remains the authority. Saturation is a nonfatal
  internal disposition; no persisted counter, second coordinator, engine or
  store was added.
- A private scheduler worker-function seam permits a test barrier after the
  actual terminal commit while the principal continues. Production uses the
  existing worker implementation and writer queue; no exported test hook.
- Delegation was bounded to corpus/codec, store/restart, scheduler, and source
  review units with disjoint file ownership. The principal integrated changes,
  independently reran focal checks, and retained Git/publication authority.

## Verification limitations and exploratory failures

An exploratory package run began at the original HEAD while edits were active.
Workflow, strictjson, local, stageexecution, workflowdefinition and compatibility
passed, but Lingo subprocess tests failed with `recovery_required`. Source timing
and the subprocess `go build` in execution_interrupt_test.go support the inference
that a pre-change test reader encountered child-binary snapshots carrying the new
member after the codec edit.
That mixed-source run is **FAIL**, not qualified baseline or final evidence.
Only stable-source reruns below can qualify the final change.

New T2 store tests initially expected Completed-style readiness after partial
success and canonical support from the generic v1 inventory. Both expectations
were corrected to the existing RunRunning and namespace/Newer contracts, retaining
actual canonical Load/List checks. Planned fixture audit also found messaging,
parallel failure and integration fixtures relying on historically absent unlimited
capacity; these now explicitly declare the intended 2/3 slots. Frozen i369,
i404 and released stable-v1 bytes were not regenerated.

Real AI/model invocations, remote Providers, T12-03, infrastructure changes,
immutable ubuntu-t12 r1 edits and external product effects were NOT_RUN.

## T4–T6 results and bounded remaining work

T4 commit `1d8d73851070491fd437754acfead21e4b068cbd` passed the principal's
focused run of `TestI404Scheduler|TestI404Restart|TestEngineParallel|TestI369LostFenceBeforeReservationPermitsNoCall|TestEngineStopsSuccessorsOnMissingOutputGateAndSpawnFailure`
(exit 0, stageexecution 5.080s). Two starts were acknowledged before either
barrier was released. A real terminal commit followed by a worker-side hold
left the principal free to refuse the third root for capacity. Releasing the
worker allowed that root while its peer was still running; the independent
observer's peak union was 2, and the join waited. Snapshot-return corruption
retained durable prepared with zero spawn, then unknown on restart; materialization
failure retained unknown with zero spawn/replay; Build failure retained its failed
fact while its peer drained successfully and the third root remained unattempted.
Existing spawn and lost-fence checks passed. Existing integration tests retain
serialized integration and validated-result prerequisites.

T5 commit `7fd322137a859c93b5b1a207e0b1b13c101ca283` removed the mapper refusal
only after T1–T4 checkpoints. Every mapped Stage now carries its exact validated
limit, including 1, through a fresh pointer. Agent edges and Stage order are
preserved. Focal `go test ./cmd/lingo -run '^TestI404' -count=1 -timeout=120s`
passed (exit 0, 2.032s): DAG2/DAG1/sequential1 mappings, nine invalid definition
cases, a public run creating one canonical Execution and four observed local stub
starts/successes, and pre-start refusals leaving zero Execution/stub spawn.
Unknown validator policy is tested directly at the mapper because Project
authoring already refuses undeclared policies; it was not installed as a public
run fixture. Public negatives cover combined instruction limits and missing
required business context, preserving their different diagnostic categories.

Final source review at `7fd3221` independently inspected the scheduler/mapper,
ran `go test ./internal/stageexecution ./cmd/lingo -run 'I404' -count=1` (exit 0;
14.269s/18.477s), and identified no additional production blocker. The first
review independently inspected T3/store and re-inspected its own T2 codec work;
the principal separately inspected codec/strictness/diff. Review covered intent
and specification, accepted architecture/constitution, verification, and security.
The final finding is **major / verification / blocking stage.completed**:
qualification of all six tasks is incomplete. These reviews are engineering
Evidence, not human acceptance, merge approval or a formal delivery review.

Five corrective rounds were spent on new fixture/oracle mistakes: T2 store
expectations; two updates during the planned parallel-fixture audit; T5 schema
inputs/checkpoint; T5 authoring/reference diagnostics. Production controls were
not weakened to satisfy those tests. A subsequent additional T6 qualification
experiment tried real Engine.Abandon against the frozen synthetic process
identities and failed with `stop_orphan_then_abandon` (exit 1). That is a truthful
refusal, not proof of permitted recovery. The unpublished experiment was removed
rather than bypassing liveness; no further corrective loop was run. The existing
unknown/no-replay tests remain, and pending restart coverage now includes explicit
1 as well as absent/2. Final independent checks do not erase that failed experiment.

**Result: PARTIAL**, even if the broader commands pass. Required remaining work:

1. A controlled, valid process-liveness fixture proving permitted explicit
   resolution and retry of historical absent-limit records through the same
   store/engine, with unchanged Workflow/digest, retained uncertainty, frozen
   invocation, budgets and no old-attempt replay. Current historical restart
   tests prove unknown/refused blind retry, not this successful resolution path.
2. An integrated saturated-retry lifecycle oracle, or a concrete feasibility
   proof explaining its reachability: current Run claims orphaned active attempts
   unknown before dispatch. Domain tests prove retry capacity/non-consumption,
   and scheduler source retains retry intent, but this is not a separately
   observed scheduler saturation/retry lifecycle.
3. Finish qualification against the complete T6/DC/AC matrix in a subsequent
   bounded implementation attempt; preserve public invalid-declaration refusal
   coverage and distinguish direct mapper tests from observed public spawn counts.

No material product/architecture decision is requested. These are required
verification items within the accepted baseline, not optional deferred features.
`deferred_findings[]` is empty. Next gate: complete remaining implementation
qualification, then independent review/reconciliation. No continuation callback
integration was available or invoked; checkpoint is AWAITING_EXECUTION_REPORT.

## Baseline and four-pillar reconciliation

| Pillar | Observed consequence |
| --- | --- |
| Work Item / AC / non-goals | GitHub #404 only. AC1: DAG2 maps and public run starts. AC2: engine barriers prove overlap/replenishment and one canonical Execution; full T6 qualification remains partial. AC3: invalid mapper cases and strict codec pass; broad CLI integration regressions FAIL and the remaining recovery/retry proof is explicit. AC4: actual commands, hashes, failures, skips and limitations are recorded; no invented acceptance. |
| Product intent | Declared Stage intent survives the snapshot and constrains admission, without adding human transitions or conflating Project/Repository, Runtime/Model or Execution/Agent. Useful validation uses deterministic local stubs. |
| Architecture / security | ADR-0037 keeps one aggregate/store/engine and principal fenced writer. ADR-0039 launch boundaries, exact effect/target authority, recovery/no-replay, budgets, isolated writers and serialized integration remain. No new owner, public flag, dependency, format, migration or authority grant. Older binaries fail closed on the new member. Synthetic fixtures contain no private Provider payload. |
| Design | NOT_APPLICABLE. No new UX surface or immutable ubuntu-t12 r1 edit. |

The accepted Specification/Plan decisions and constitution are not reopened.
Git commits/push for this repository do not grant Axiom Executions authority to
publish in another repository. No merge, release, deploy, Draft conversion,
Issue closure, human acceptance, Linear or Cursor mutation was performed.

## Stable-source validation

Tested Go source: `7fd322137a859c93b5b1a207e0b1b13c101ca283` (the only pending
repository artifact during these commands was this Evidence document). Go files
were not edited during qualification. Cached package results are reported by Go
as such and are not newly observed execution.

| Command | Exit / status | Observed result |
| --- | --- | --- |
| `go vet ./...` | 0 / PASS | No diagnostics. |
| `go build ./...` | 0 / PASS | Build completed. |
| `go mod verify` | 0 / PASS | All modules verified; run at `1d8d738`, unchanged go.mod/go.sum through tested source. |
| `go test -race ./internal/workflow ./internal/local ./internal/stageexecution ./cmd/lingo` | 2 / SKIP | Host linux/arm64 has CGO_ENABLED=0 and no gcc in PATH; Go reports `-race requires cgo`. No infrastructure/compiler installation attempted. Source at `1d8d738`. |
| `./scripts/validate-repository.sh .` | 0 / PASS | Governance/bootstrap, registered automation, ADR structure and sensitive-file checks passed; not Runtime behavioral qualification. Evidence working file was present. |
| `gitleaks` availability (`command -v gitleaks`) | 1 / SKIP | Tool unavailable; no installation or substitute PASS claimed. |

Full-suite results follow below and final publication evidence belongs to the Issue receipt;
these checks cannot close the major T6 verification finding above.

Reachability evidence for remaining retry qualification: `Engine.Run` always
calls `Claim` before `runReady`; Claim acquires a newer generation and marks
older prepared/running attempts unknown. `runReadyWithWorker` initializes an empty
inflight map and takes its only retry request from RunOptions, before any worker
exists; it clears that request only after successful preparation. No operation
currently adds another retry request while workers are running. Thus ordinary
public startup cannot supply legitimate active principal workers competing with
that initial retry: retained active work instead triggers unknown recovery.
This source argument narrows the missing integrated scenario; it does not waive
its requested qualification or replace an independently reviewed feasibility
proof. The domain terminal-inflight/retry oracle remains directly exercised.

Failed experiment command: `go test ./internal/stageexecution -run '^TestI404Restart' -count=1 -timeout=60s`,
exit 1, `TestI404RestartHistoricalMultiActiveBecomesUnknownWithoutReplay` reported
`stop_orphan_then_abandon` after 0.16s. Source was HEAD `1d8d738` plus the then
working T5/test delta, before the `7fd3221` commit. The temporary test extension
called Engine.Abandon for `attempt-a`/`attempt-b` from the literal historical
fixture and incorrectly expected nil; it was never a committed qualifying test.
No exact committed test-source SHA or successful resolution proof is claimed for
that experiment. The frozen fixture and its old process identities are unchanged.

Final broad regression **FAIL** at `7fd3221`: `go test ./...` exited 1.
`cmd/lingo` failed after 513.098s in these existing public integration families:
`TestPublicWorkflowRunSameRepositoryIntegration`,
`TestPublicWorkflowRunSameRepositoryStopsOnConflictOrFailedCheck`,
`TestPublicCompositionSameRepositoryFaultsNeverReplay`, and
`TestPublicCompositionSameRepositoryChangedBaseCollisionAndCancel`.
The first writer's local stub exited 8 and its peer was unattempted. Source
inspection identifies an incomplete fixture audit: `integrationDefinition` in
[execution_integration_test.go](../../../cmd/lingo/execution_integration_test.go)
sets mode DAG but inherits the builtin concurrency **1**, while
`integrationRuntimeStub` waits for both a-ready and b-ready before either root
can finish. Applying the declared 1 now correctly prevents that overlap. This
fixture needs the intended explicit parallel limit in a subsequent bounded
correction, preserving dependent-mode and integration assertions, followed by
focused and broad reruns. It was **not changed or revalidated** after the five
corrective rounds were exhausted. This is an additional required `remaining_work`
item and blocks full qualification; it is not reclassified as environment noise
or waived by the passing engine/mapper focal tests.

| Final command at stable `7fd3221` | Exit / status | Observed result |
| --- | --- | --- |
| `go test ./internal/workflow ./internal/strictjson ./internal/local ./internal/stageexecution ./internal/workflowdefinition ./cmd/lingo ./internal/compatibility` | 1 / FAIL | Six packages passed (strictjson/workflowdefinition/compatibility cached); cmd/lingo failed in the same existing integration fixture families (343.048s). Workflow 0.605s, local 244.357s, stageexecution 300.587s passed. |
| `go test ./...` | 1 / FAIL | 31 packages passed, five had no test files; cmd/lingo failed (513.098s). No passing full-suite claim. |

Final Evidence-only publication adds no Go change after the tested source.
`git diff --check`, relative Markdown targets, staged sensitive-file checks and
staged diff inspection cover this document; exact commit/push/remote/PR read-back
and CI observation are recorded in the single Issue #404 completion receipt.

## IMPLEMENT-404-02 — corrective continuation

History above (PARTIAL, failures, SKIPs, SHAs) is preserved unchanged. Authority:
the existing [RESUME_IMPLEMENT gate](https://github.com/rgomids/axiom/issues/404#issuecomment-7141666800);
no decision or acceptance is republished. Pre-edit state verified: origin
`https://github.com/rgomids/axiom.git`, branch `issue/404-specify-dag-concurrency`,
local = remote `34619529ac5db76d089ff391d192b32d6d5c0636`, base/main
`5cd783bd2ac8189a1b9c85a790472b5335139c78`, clean worktree, PR #405 OPEN Draft,
only this axm-run session active. No production Go file changed in this attempt.

### Fixture correction (commit `0686332`)

Focal reproduction at `3461952`:
`go test ./cmd/lingo -run '^TestPublicWorkflowRunSameRepositoryIntegration$' -count=1`
exit 1 — attempt `a` exit 8, `b` and `observer` unattempted. Cause is the fixture,
not the product: `integrationRuntimeStub` holds both roots until `a-ready` and
`b-ready` exist, so every mode without the `a→b` edge requires two concurrent
roots; `mode:dependent` writes `b-ready` itself and is serialized by its edge.
`integrationDefinition` now declares `Concurrency = 2` (≤ 3 agents) for every
mode. Stub, edges, observer join, gates, validators and assertions are unchanged;
builtin revision 2, frozen corpora and Spec are untouched; the scheduler still
serializes a declared 1 (`TestI404Restart…/explicit1` and `/absent` peak 1).

Consumers of `newPublicIntegrationFixture` (all rerun, `-count=1`, exit 0):
`TestPublicWorkflowRunSameRepositoryIntegration` (independent, checks),
`…DependentWriterAndHumanGate` (`mode:dependent`, `mode:gate`),
`…StopsOnConflictOrFailedCheck` (`mode:conflict`, `mode:check-failure`),
`…RefusesMissingChecksAuthorityAndDirtyBase` (checks/authority/dirty),
`TestPublicCompositionSameRepositoryFaultsNeverReplay` (six lost-* faults, uses
`mode:held` via `publicIntegrationHeldAtGate`) and
`…ChangedBaseCollisionAndCancel` (changed-base, both collisions, cancel,
admission), plus all `TestI404*` Lingo tests: 26.145s. Full
`go test ./cmd/lingo -count=1` then passed (155.584s; previously FAIL 513s), so
the bounded audit found no other consumer depending on implicit parallelism.

### T6 historical resolution and explicit retry (commit `204f782`)

`TestI404RestartHistoricalResolvedWithQualifiedLivenessPermitsExplicitRetry`
derives an in-memory copy of the frozen `multi-active.json` (corpus bytes
re-read and compared at the end; manifest hashes re-verified unchanged) and
replaces only the two process identities with real identities of `/bin/cat`
process groups the test itself started (`Setpgid`), observed through
`executiongraph.ObserveProcessCorrelation`, and reaped. No liveness seam is
replaced, no third-party process is touched, nothing is signalled. The
original synthetic identities (`GroupIdentity "404"`, not `boot:pgid`) remain
unobservable, which is why the IMPLEMENT-404-01 experiment's
`stop_orphan_then_abandon` refusal was correct and remains so.

Observed sequence in the same local store/engine: Run claims (generation >1),
both attempts unknown, RecoveryRequired, 0 spawns; blind retry
`ErrUnknownOutstanding`; canonical `Engine.Abandon(attempt-a, human:operator)`
reports two facts, each `process_gone_effects_unknown` or
`pid_reused_not_signalled`; second Abandon of the same attempt `ErrTransition`;
retry with peer `attempt-b` still unresolved refused, 0 spawns; after
abandoning `attempt-b`, explicit retry of `a` spawns exactly `a` once: new
attempt ID ≠ `attempt-a`, Number 2 (budget 3), newer generation, invocation
deep-equal to the frozen historical one although the fixture resolver offers a
different model. With that worker's identity inflight, a domain retry probe for
`b` returns `ErrStageCapacity` (absent ⇒ effective 1, not materialized). Retry
of `b` then completes the run: spawned exactly `a, b, join`, peak 1, status
completed. After every step the Workflow is deep-equal with unchanged digest,
`Concurrency` nil, zero integrations/deliveries, and both historical attempts
keep status unknown, generation 1, invocation and process identity; the
reloaded record equals the returned one.

### T6 saturated retry: tested versus proved by reachability

Integrated saturated-retry lifecycle: **NOT_RUN — unreachable on the current
public path**, not PASS. Independent inspection at `204f782` (product lines
unchanged since `7fd3221`):

1. Only `cmd/lingo/execution_run.go:291` supplies `RunOptions.Retry`;
   `execution_run.go:206` and `internal/stageexecution/run_start.go:96` pass
   empty options. All three reach `Engine.Run` (`engine.go:211-218`), which
   always `Claim`s first; the exclusive `AcquireFence` excludes a concurrent
   principal.
2. `Claim` (`engine.go:128-146`) takes a newer generation and marks every older
   prepared/running attempt unknown, so no durable occupancy of the current
   generation exists at the first scheduling pass.
3. `runReadyWithWorker` starts with empty `inflight` (`scheduler.go:108`), takes
   the only retry from `opts` (`:109`), schedules only it while set (`:127-131`)
   and clears it only after successful preparation (`:152-154`); no operation
   adds a retry later.
4. `PrepareAttempt` refuses `RecoveryRequired` (`execution_transitions.go:387-389`)
   before the capacity check (`:451-457`).
5. A retry-eligible failed task cannot coexist with saturating current-generation
   peers: after a failure no new root is eligible (`PrepareAttempt` "task is not
   eligible", observed while building the boundary case), and peers admitted
   before it already counted against the same limit.

Boundary test `TestI404RetryFacingSaturatingPeersIsRecoveryNotCapacity`
exercises the only shape that can saturate (historical multi-active derived:
`a` failed, `b` running, absent limit): the domain retry probe returns
`ErrStageCapacity`, while public-path `Engine.Run(Retry a)` returns
`ErrUnknownOutstanding` (not capacity), RecoveryRequired, `b` unknown, `a`
unchanged with one attempt, 0 spawns, Concurrency nil. Domain saturation and
non-consumption remain proved by the existing T3 tests. Non-vacuity: with the
fallback temporarily mutated to 2 in `stageCapacity`, both new tests failed
(`historical absence admitted beyond fallback 1`, `domain saturation not
observed`); the mutation was reverted before commit and is absent from the diff.

### Commands at stable source `204f7827c16ff3e68b7055fe1e9cfe44875da88a`

Go source was stable during these commands; only this Evidence file was being
edited (no Go test reads it). No result below is cached (`-count=1`).

| Command | Exit / status | Observed result |
| --- | --- | --- |
| `go test -count=1 ./internal/stageexecution -run '^TestI404'` | 0 / PASS | 9 tests incl. both new ones, 3.306s |
| `go test -count=1 ./internal/workflow ./internal/strictjson ./internal/local ./internal/stageexecution ./internal/workflowdefinition ./cmd/lingo ./internal/compatibility` | 0 / PASS | 7/7 ok: workflow 0.500s, strictjson 0.082s, local 89.389s, stageexecution 127.335s, workflowdefinition 3.480s, cmd/lingo 164.594s, compatibility 3.381s |
| `go test -count=1 ./...` | 0 / PASS | 32 packages ok, 5 without test files, 0 FAIL; cmd/lingo 232.274s |
| `go vet ./...` | 0 / PASS | No diagnostics |
| `go build ./...` | 0 / PASS | Build completed |
| `go mod verify` | 0 / PASS | All modules verified |
| `go test -race ./internal/workflow` | 2 / SKIP | `-race requires cgo`; linux/arm64, CGO_ENABLED=0, no gcc. Nothing installed. |
| `which gcc gitleaks` | 1 / SKIP | Neither available; gitleaks not run, no substitute PASS |
| `./scripts/validate-repository.sh .` | NOT_RUN | Refused by this session's tool-permission policy (needs approval not available in non-interactive run). Last observed PASS was at `7fd3221` (above); not re-established here. |
| `./scripts/check-sensitive-files.sh --staged .` | NOT_RUN | Same permission refusal. Substitute (not PASS): manual inspection of each staged diff — only Go test code and this Markdown; no credentials, hosts, tokens or Provider data. |
| `git diff --check` | 0 / PASS | On this Evidence change before commit |

Corpus: all 14 `internal/workflow/testdata/i404` entries re-hashed equal to
`manifest.json` (manifest SHA-256 `bea909fa…5ac8`, unchanged); `ubuntu-t12` r1
untouched.

### Decisions, findings and result

- TECHNICAL_LOCAL: fixture declares 2 rather than changing the stub or a global
  default — it is the intent the stub encodes; scheduler stays strict.
- TECHNICAL_LOCAL: liveness qualified with real ended, test-owned process groups
  rather than replacing `observeRecordedProcess` — the seam would substitute an
  assumed fact for an observed one.
- TECHNICAL_LOCAL: saturated retry qualified by reachability proof plus boundary
  test rather than a new seam/API, which the attempt forbids.
- Deferred (optional hardening, not a requirement): if a retry ever met
  `ErrStageCapacity` with empty `inflight`, `scheduler.go:141-142,170-174` would
  return without error and drop the retry intent. Unreachable today (above);
  an explicit refusal there would make the invariant self-checking.

Independent review (engineering/security, not approval): the diff adds no
production code, authority, seam, flag, dependency or format; the test spawns
only `/bin/cat` in its own process group, reaps it and signals nothing; derived
records live in temp stores. AC1–AC4 and DC-01–DC-05 behavioral oracles from the
Tasks matrix are now exercised by passing tests, except the integrated saturated
retry, which is NOT_RUN by proved unreachability (boundary-tested).

**Result: PARTIAL.** All Go qualification passes at `204f782`. Remaining:
`validate-repository.sh` and `check-sensitive-files.sh --staged` must run in a
session permitted to execute repository scripts; race/gitleaks stay SKIP on this
host. Next gate after those: review/reconciliation at the resulting SHA.
