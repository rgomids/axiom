# IF-01 correction and IF-02 triage: independent full S1 re-review

Attempt `IF-01-CORRECTION-CLAUDE` on [AXM-42](https://linear.app/rgomids/issue/AXM-42) /
[PR #366](https://github.com/rgomids/axiom/pull/366), from
[review 5732338943](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5732338943)
(analytical verdict `REVIEW_INCOMPLETE`, native COMMENT, no confirmed Blocker
or Major). Executor: Claude Opus 5.5 (`claude-opus-5-5`, effort medium), direct
work with no delegation. This is a bounded correction, not a review. Model
diversity is still not achieved: the executor, the implementer and the reviewer
share a model family. The next gate is `DIVERSE_MODEL_HUNK_REVIEW`; this
correction does not make the PR pre-merge ready.
[`correction-cr-s1-07.md`](correction-cr-s1-07.md) and earlier Evidence stay
historical.

## Identity

| Item | Value |
| --- | --- |
| Base / merge-base | `440e44a68638e08e901fd3f87f5b15cd395a6365` |
| HEAD before correction (local = origin = PR head = reviewed SHA; worktree clean apart from ignored `tmp/` and `scripts/__pycache__/`) | `15e4d24b57614df46832e8a83f83309fe211b4c5` |
| IF-01 fix and tests commit | `d0ab29afa628cec83edb58004b4724ff42b6a45a` |
| Evidence commit / final HEAD | recorded in the PR reply and the AXM-42 event |

## Findings and resolution

| ID | Severity | Status | Change | Proof |
| --- | --- | --- | --- | --- |
| IF-01 | Minor | FIXED, TESTED (red before, green after) | `local.ExecutionReferenceNamespace` (`internal/local/execution_list.go`) recognizes exactly two inventory entries: `executions/v2` classified `unsupported_newer`, and `executions/v2-attempt-outputs` classified `unknown`. It accepts them only while each is still a private, owned, non-symlinked directory below the state root. `ObserveWorkflows` (`cmd/lingo/workflow_authoring.go`) is its only caller and sets only those entries aside, so the strict `ExecutionStore.List` scan becomes reachable and stays the reference authority. `InspectStateInventory`, `InventoryKind.Supported`, `InspectCutover` and every other inventory consumer are unchanged | CLI black-box `TestWorkflowRemoveWithCanonicalExecutions` (`cmd/lingo/workflow_reference_test.go`), below |
| IF-02 | Nit (display only) | NOT REPRODUCED; no production change; DEFERRED_NOT_IMPLEMENTED (ordering nit) | None. A probe test is added | `TestExecutionStoreRefusedCompletionReturnsPersistedState` (`internal/local/execution_store_test.go`), below |

### IF-01: before and after

Hypothesis verified, both statically and by test. `eachVersionDirectory`
(`internal/local/inventory.go`) reports `executions/v2` as `unsupported_newer`
without descending into it. `executions/v2-attempt-outputs` does not match
`v<N>` and is reported `unknown`. `ObserveWorkflows` set
`ReferencesChecked=false` for any unsupported entry, so once one canonical
Execution existed, every `project workflow remove` was refused with
`reference_inventory_unknown` before `ExecutionStore.List` ran. The refusal
was fail-closed, so no unsafe removal was possible, but it was a functional
regression against base, where `executions/v1` was supported.

Red run at `15e4d24` plus the new test only, with the inventory the test
observed:

```text
if01-inventory executions/v2 unsupported_newer
if01-inventory executions/v2-attempt-outputs unknown
if01-inventory projects/<id>/installation.json installation_record
[remove --workflow custom --revision 1 ...] => {"status":"validation_failure",...,"category":"reference_inventory_unknown",...} (exit 1); want revision_in_use
--- FAIL: TestWorkflowRemoveWithCanonicalExecutions
```

Green run after the fix (`go test ./cmd/lingo -run TestWorkflowRemoveWithCanonicalExecutions -count=1 -v`):

```text
if01-evidence [remove] => category=previewed exit=0                    # (iv) blank state, no executions/ directory
if01-inventory executions/v2 unsupported_newer                          # cutover classification unchanged
if01-inventory executions/v2-attempt-outputs unknown
if01-evidence [remove] => category=revision_in_use exit=1              # (ii)+(v) referenced, 25 Executions, match sorts last
if01-evidence [remove] => category=previewed exit=0
if01-evidence [remove] => category=revision_in_use exit=1              # (ii) with presented authorization
if01-evidence [remove] => category=previewed exit=0                    # (i) unreferenced
if01-evidence [remove] => category=applied exit=0                      # (i) retired
if01-evidence fail-closed corrupt referenced record                    # (iii) each: reference_inventory_unknown, exit 1
if01-evidence fail-closed newer record format
if01-evidence fail-closed unknown entry in project namespace
if01-evidence fail-closed pending protocol state
if01-evidence fail-closed unsupported newer namespace                  # executions/v3
if01-evidence fail-closed unknown executions sibling                   # executions/v2-other
if01-evidence fail-closed unreadable project namespace                 # mode 000
if01-evidence fail-closed symlinked v2 namespace
if01-evidence fail-closed symlinked attempt-output namespace
if01-evidence [remove] => category=previewed exit=0                    # restored state valid again
if01-evidence [remove] => category=revision_in_use exit=1              # referenced revision still held
--- PASS: TestWorkflowRemoveWithCanonicalExecutions
```

Scenario mapping:

- (i) An unreferenced revision is previewed and retired while canonical
  Executions and attempt outputs exist.
- (ii) The referenced revision refuses with `revision_in_use`, at preview and
  with presented authorization. A fingerprint of the state root and the
  portable root (path, mode, bytes, link target) is unchanged.
- (iii) Each damage case refuses with the bounded category
  `reference_inventory_unknown`, and the fingerprint is unchanged.
- (iv) A blank state with no `executions/` directory previews as before.
- (v) 24 Executions reference a different revision of the same Workflow; the
  only match is the last record `List` reads.

The Executions are real format-4 records published through
`ExecutionStore.Create`, not hand-written JSON. The existing
`TestExecutableWorkflowDefaultCustomEditSelectAndRetire`, including its
`unknown-reference.json` refusal, still passes.

Mutation check: with the private-directory verification in
`ExecutionReferenceNamespace` replaced by `return true`, the
`symlinked attempt-output namespace` case fails (`exit 0; want
reference_inventory_unknown`). The first version of the test kept its holding
directory inside the state root, so an unrelated unknown entry masked that case.
The holding directory now lives outside the state root, and the mutation was
reverted. A symlinked `executions/v2` is refused by `List` itself:
`existingPrivateChild` returns `unsafe project storage`, confirmed with an
isolated, uncommitted probe.

### IF-02: triage

The review claimed that `CompleteAttempt` writes `EndedAt`, `ExitCode`,
`Effects`, `Artifacts` and `Diagnostic` before it can return `ErrTransition`
(`execution_transitions.go:431-434`), and that `cloneExecutionStages`
(`execution_store.go:346-357`) copies `Commands` shallowly, so `Update` could
return an already-mutated `current`.

The first half is true of the working copy. The second half does not hold:

- `Update` applies `mutate` to `next`, whose stage, task and attempt slices are
  fresh copies.
- On any mutate error, `Update` returns `current`, the aggregate just decoded by
  `Load`, never `next`.
- `CompleteAttempt` reassigns slice and pointer fields; it never writes through
  the shared `Commands` backing array.
- The engine's terminal closure returns that error, and `Engine.attempt`
  propagates the returned `current`.

`TestExecutionStoreRefusedCompletionReturnsPersistedState` first proves that it
reaches the refusal branch: on a decoded working copy, `CompleteAttempt` with
mismatched command results returns `ErrTransition` with `EndedAt` and
`Diagnostic` already written. Through `Update`, it then shows that both the
returned and the reloaded aggregates encode byte-identically to the persisted
record. No production change is made. The write-before-refuse ordering inside
the domain transition is recorded as DF-IF-02 below.

## technical_decisions[]

- **TD-IF-1:** Scope the exception to the Workflow-reference observation
  through one exported predicate called only by `ObserveWorkflows`. Do not mark
  `v2` as supported. Rejected alternatives: adding `executions/v2` to the
  supported inventory kinds, or descending into it in `InspectStateInventory`.
  Either would change `InspectCutover` and the pre-switch BLOCK that IR-08 and
  CR-S1-06 rely on. Rollback: revert the IF-01 commit, which restores
  fail-closed refusal.
- **TD-IF-2:** Match the exact relative path together with the exact kind the
  inventory assigns today. If the inventory ever classifies these entries
  differently, the predicate stops matching and retirement fails closed again.
  Deeper entries, `v3` and other siblings are not exempt.
- **TD-IF-3:** Require each exempt namespace to be a private, owned,
  non-symlinked directory at observation time. `List` independently re-opens
  `executions/v2/<project>` through anchored private opens and fails on any
  unexpected, pending, corrupt or newer record. Attempt outputs carry no
  Workflow reference and are never read for references. The check therefore
  only keeps an unexpected shape from being treated as canonical.
- **TD-IF-4:** Only the selected Project's canonical namespace is scanned.
  Executions of other Projects reference their own Project's Workflows, and a
  damaged foreign Project namespace no longer blocks this Project's retirement.
  Before the fix it blocked retirement for every Project.
- **TD-IF-5 (IF-02):** No production change for an unreproduced display-only
  Nit. Reordering `CompleteAttempt` would touch the transition model, which is
  out of scope. The probe test locks in the observable invariant.

## deferred_findings[]

| ID | Source | Status | Rationale |
| --- | --- | --- | --- |
| DF-IF-02 | [review 5732338943](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5732338943) IF-02; `internal/workflow/execution_transitions.go:431-434` | DEFERRED_NOT_IMPLEMENTED | `CompleteAttempt` writes result fields before its command-result refusal. The writes reach only `Update`'s discarded working copy, and the returned and persisted aggregates are unaffected (probe test). Validating before writing would be a hygiene change to the transition model, outside this bounded correction |
| DF-IF-03 | This correction | DEFERRED_NOT_IMPLEMENTED (pre-existing design) | Observation and retirement are not one atomic transaction with Execution creation. A run started after `List` and before the index commit could reference a revision being retired. Retirement preserves the immutable definition (`retire_revision_index_preserve_content`), so no content is lost. This is unchanged from base |
| DF-R-01, DF-R-02, DF-IR-01, D-01, D-02 residual, D-04, D-05, DF-07-1..3 | Earlier Evidence | Unchanged | Not touched by this correction |

## Validation

Local runs are on linux/arm64, `ubuntu-arm-01`.

| Command | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1 -timeout=30m` | exit 0. All 32 test packages `ok` (`cmd/lingo` 166.7s, `internal/local` 88.2s, `internal/install` 67.9s, `internal/stageexecution` 18.3s) |
| `go run cmd/gofmt -l cmd internal scripts` | empty (PASS). The direct `gofmt` binary was denied by the session's permission policy, so the toolchain gofmt was run through `go run`, as in the review |
| `go run ./scripts/check-architecture.go domain` | PASS (12 files) |
| `go run ./scripts/check-architecture.go application` | PASS (46 files) |
| `git diff --check 440e44a...<final HEAD>` | empty (PASS); re-run on the final HEAD and recorded in the PR reply |
| IF-01 black-box, IF-02 probe | PASS (above) |
| CR-S1-06/CR-S1-07 focused negatives (`-v`, `cmd/lingo`, `internal/stageexecution`, `internal/local`) | PASS: `TestKilledTestBinaryLeavesNoOwnedProcess`, `TestPublicWorkflowRunSignalStopsOwnedGroupAndCommitsStop`, `TestPublicReconcileAbandonRefusesLiveOrphanWithoutWriting`, `TestEngineAbandon*` (5), `TestEngineCancelPersistedBeforeTerminalCommitIsRecorded`, `TestEngineInterrupt*` (2), `TestFreezeCommandsResolvesSymlinkedExecutableBeforeExecution`, `TestRuntimeChildMarkersRefuseHumanAuthority`, `TestRuntimeLauncherCommandCarriesAttemptCorrelation`, `TestCutover*` (15). Those tests assert their own stub PID/PGID identities; none reported an orphan. A host-level `ps` inspection was denied by the session's permission policy (NOT_RUN) |
| Pre-commit `./scripts/check-sensitive-files.sh --staged .` / `gitleaks` | NOT_RUN: the script was denied by the session's permission policy, and `gitleaks` is not installed. Staged paths and content were inspected manually: code, tests with synthetic identifiers, and this document, with no credentials or provider data. The CI checks on the final SHA are authoritative |
| Shell validators, `staticcheck`, `-race` | NOT_RUN: shell script execution needs approval that this non-interactive session cannot obtain |
| Vendor Codex/Claude CLI, vendor E2E, real cutover, Darwin | NOT_RUN by design |

CI on the final SHA is recorded in the PR reply and the AXM-42 event. It is not
claimed here.

## Security and trust boundaries

- No state is created, repaired or removed by the observation. Every refusal is
  proven effect-free by fingerprint.
- Symlinks, unreadable directories, pending protocol state, newer formats and
  unknown entries all still refuse.
- Cutover and upgrade inventory semantics are byte-for-byte unchanged.
- Tests use private temporary state roots only. No real customer or host state
  was touched, and no provider CLI was invoked.

## Remaining limitations and human boundary

- Hunk-level coverage of the 318-path PR is still incomplete, and model
  diversity has not been achieved. Next gate: `DIVERSE_MODEL_HUNK_REVIEW`.
- This correction is not an approval. Merge, release, cutover and acceptance
  stay separate human decisions.
