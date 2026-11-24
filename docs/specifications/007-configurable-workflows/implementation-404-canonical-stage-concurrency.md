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
