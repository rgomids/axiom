# S1 full technical review and in-scope correction

Attempt `S1-FULL-REVIEW-CORRECTION-CLAUDE` on [AXM-42](https://linear.app/rgomids/issue/AXM-42) /
[PR #366](https://github.com/rgomids/axiom/pull/366). Executor: Claude Opus 5.5
(`claude-opus-5-5`, medium). The executor reviewed and corrected its own scope.
This is **not** an independent approval: `INDEPENDENT_REVIEW_PENDING`.

Outcome: **REVIEW_INCOMPLETE, corrections applied.** All production Go in the
diff was reviewed; five defects were proven with failing tests and fixed. Test,
documentation and data files were not all read line by line (see Coverage).
No merge, release, deploy, installed-state cutover, vendor call or acceptance.

## Identity

| Item | Value |
| --- | --- |
| Base (merge-base = PR base = `origin/main`) | `440e44a68638e08e901fd3f87f5b15cd395a6365` |
| Reviewed HEAD (PR head at start, local = origin, worktree clean) | `b945ecc1f3e912b3510ee65ad7036b1c05e3ca18` |
| Correction commits | `334aebc` (F-01, F-02), `9cdfef0` (F-03, F-04), `045a97e` (F-05) |
| Changed files base→reviewed HEAD | **308** (65 A / 142 M / 101 D; `git diff --name-status --no-renames`), 25,382+/30,300− |
| Ledger | [`review-s1-full-coverage.json`](review-s1-full-coverage.json), one row per path |

The 308 count from the earlier report is now verified. Preflight found only the
ignored `tmp/` and `scripts/__pycache__/`, no other writer than this session's
Maestro wrapper, and no drift of base, head or PR identity.

## Coverage

| Depth | Files | Meaning |
| --- | ---: | --- |
| `line_by_line_full_file` | 22 | All new canonical Execution, store, cutover, engine, launcher, run wiring, invocation and selector code, plus Spec/Plan/Tasks and `docs/operational/workflow.md` |
| `all_added_lines_reviewed` | 45 | Every other modified or added production Go file: each added line was read |
| `removal_verified_not_content_reaudited` | 101 | Deleted files. Build/vet/full tests pass and no production Graph/Single/StagePlan/ValidatorPolicy reader or writer remains. Deleted content was not re-audited, and the safety-test port map was not independently rechecked |
| `partial_relevant_hunks` | 3 | Engine, launcher and store tests read where relevant to the six domains |
| `category_level_only` | 137 | The rest of the test, documentation, ADR, evidence JSON and script files: not read individually |

Resume from the ledger rows with `category_level_only`, `partial_relevant_hunks`
or `removal_verified_not_content_reaudited`. Priority: ported safety tests
(`internal/stageexecution/engine_ported_test.go`, `cmd/lingo/*_test.go`), ADR-0037 and
the amended ADRs, installed skills (`internal/codexruntime/skills/*`), then
evidence JSON.

## Six-domain trace

Traced through code and tests at `b945ecc`:

- **Lifecycle:** `Engine.Start` → `NewExecution` (validates the DAG and bounds) →
  `Resolve` for every agent → `Create`. Then `Run` → `Claim` (records the new
  generation; orphaned prepared/running attempts become unknown) →
  `PrepareAttempt` committed before `Build` → `RunObserved` with `MarkStarted`
  before stdin release → terminal commit. Status is always derived from facts,
  and `Validate` recomputes the workflow digest on decode.
- **Command authority:** `--authorize-commands` is required at freeze and at
  resolve. Argv[0] is an absolute non-symlink path and the rest are positional
  arguments behind a fixed `/bin/sh` stdin gate (no expansion). Each command
  records prepare → start → finish under the fence. **F-01** fixed the cancel
  classification.
- **Artifacts and Evidence:** outputs are read only from a private per-attempt
  inbox (Lstat, `O_NOFOLLOW|O_NONBLOCK`, `SameFile`, 1 MiB bound). Validated
  outputs bind execution, attempt, stage, definition, output and repository.
  Review is bound to `StageResultDigest`. Store history is immutable via
  `preservesExecutionHistory` (**F-03** added messages).
- **Concurrency:** one exclusive `run-leases/v1` epoch, with lock order lease →
  aggregate. Revision CAS plus a lease revalidated at the commit point. A nil
  fence cannot change the generation. A stale runner gets
  `ErrStaleRunner`/`ErrConflict`.
- **Cancel/stop:** cancel intent is persisted without a lease and polled during
  Build, the run and Collect. The engine signals only the owned group via
  `OSProcessRunner`. Reconcile observes and never signals (PID reuse is
  reported). **F-01** closed the gap where a command's unconfirmed stop was
  masked.
- **Failure/recovery:** crash windows become unknown on the next claim and are
  never respawned. An interrupted publication reads as `recovery_required`.
  **F-04** made it recoverable by the existing reviewed maintenance recovery.
  Cutover helpers are inert and BLOCK on uncertainty; there is no public
  command (documented).
- **Schema/bounds:** the aggregate is ≤256 KiB with 32 stages, 32 agents, 10
  attempts, 64 outputs, 64 inputs, 256 messages of 4 KiB each, and 16 KiB
  instructions. Discriminator `execution`/4 is strict; old formats are refused.
  Review is tied to the exact result digest. Retired A–H/K routes return
  bounded `command_retired`/`workflow_legacy_operation_retired`; there is no
  fallback executor.

## Findings and fixes

| ID | Sev | Path:line @`b945ecc` | Reproducible scenario | Contract | Fix / proof |
| --- | --- | --- | --- | --- | --- |
| F-01 | Major | `internal/stageexecution/engine.go:297-306`, `runtime_launcher.go:317-319` | Cancel is persisted while a configured validation command runs and the command's group stop is unconfirmed. The engine classified from the agent's result (`Stopped=false`, not unknown) and recorded `cancelled` / `exited_after_cancel_result_refused`, with the command left `running`. | Spec 015 safety: stop reports uncertainty honestly; unknown effects block replay | The launcher returns `ErrCommandTerminated` (end confirmed) or `ErrCommandOutcomeUnknown`. The engine marks the attempt unknown (`cancel_command_stop_unconfirmed`) for any started command lacking a confirmed end. `TestEngineCancelDuringValidationCommandNeverHidesUnconfirmedStop` failed 2/3 before the fix and passes after |
| F-02 | Major | `internal/stageexecution/runtime_launcher.go:85-90` | `Build` omitted `AttemptCorrelation`, so children had no `AXIOM_ATTEMPT_ID`/`AXIOM_EXECUTION_ID`. The guard at `cmd/lingo/execution_run.go:51` never fired, so a Claude child allowed `Bash(<axiom>:*)` could run `workflow run --execution X --approve-gate S --authorize-human`. The retired supervisors set this correlation. | Spec 015: agent output never satisfies human gates; authority not expanded by agent output | `Build` sets the attempt's own correlation. The Build assertion failed without the fix (env lacked markers); the CLI refusal is covered by the existing `execution_run_test.go:741` |
| F-03 | Minor | `internal/local/execution_store.go:257-263` | With a valid record and the expected revision, `Save` accepted dropping or rewriting an accepted message | Plan TD-01/TD-03: message facts are immutable | Prior messages must be an unchanged prefix. `TestExecutionStoreRejectsMessageHistoryRewrite` failed for drop/rewrite before the fix and passes after |
| F-04 | Major | `internal/local/recovery.go:196` (new namespace `execution_store.go:59`) | An F6 interruption of a canonical save left `executions/v2/<project>` as `recovery_required`. `InspectRecovery` returned 0 plans, so there was no supported repair and the Execution could not be read, cancelled or reconciled | Plan TD-01: failure windows report recovery and the smallest safe action | Recovery enumerates `executions/v2`. `TestExecutionStoreInterruptedPublicationIsRecoverable` failed before the fix and passes after: inspect → authorize → apply → readable |
| F-05 | Minor | `cmd/lingo/workflow_authoring.go:128`, `internal/projectapp/workflow_authoring.go:270` | The canonical reference producer sets no `Source` (and a stage-0 digest that may be empty), so the exact-struct comparison never matched. A revision in use by a canonical Execution could be retired | ADR-0020: binding provenance; retirement guard | Match on Workflow ID + revision. The new case failed before the fix (`previewed`) and passes after. The frozen Execution snapshot was never affected |

No Blocker found. No finding required an architecture, product or design change.

## technical_decisions[]

- **TD-R1 (F-01):** two launcher sentinels, and the engine fails closed unless a
  started command's end is confirmed. Alternatives: an engine-only rule that any
  running command means unknown (over-reports confirmed stops), or exporting
  command process results through `Outcome` (wider API). Reversible and local.
- **TD-R2 (F-02):** the child correlation uses `ChildID = ExecutionID`,
  mirroring the retired single-stage correlation. The guard still covers only
  `--authorize-human`; see D-01.
- **TD-R3 (F-03):** messages are compared element by element with
  `time.Equal`, because `reflect.DeepEqual` treats nil and empty slices
  differently and refused the first append.
- **TD-R4 (F-04):** `executions/v2` was added to the existing reviewed recovery
  enumeration instead of building a new recovery path.
- **TD-R5 (F-05):** a conservative ID+revision match in the consumer, instead of
  adding definition digest/source to the format-4 snapshot (a format change).

## deferred_findings[] (DEFERRED_NOT_IMPLEMENTED for pre-merge reconsideration)

- **D-01 (Minor, pre-existing trust model, S2 transport):** a Runtime child can
  still invoke the Axiom CLI with `--authorize-external` (start a new vendor
  Execution) or `--authorize-local` (cancel). A Codex child with a shell could
  strip the env markers. At base, children also had `Bash(<axiom>:*)` and no
  guard. Evidence: `cmd/lingo/execution_run.go:51`,
  `internal/runtimeadapter/stage_invocation.go` (Claude `Bash(` tools).
- **D-02 (Minor, S1.8 follow-up):** the cutover helpers have no operator
  command. `cutoverKnownRoots` omits `capability-evidence`,
  `capability-probes` and `project-context`, which are listed in
  `local.StateRootDirectories`, so a real inventory would BLOCK (fail-closed).
  Evidence: `internal/local/cutover.go:31`, `internal/local/inventory.go:213`.
- **D-03 (Minor):** `freezeCommands` resolves the executable via `LookPath`,
  but `Collect` refuses any symlinked path component (for example merged-`/usr`
  layouts). The validator never runs, yet the attempt becomes unknown
  (`output_collection_failed`). That overstates uncertainty and is not unsafe.
  Evidence: `cmd/lingo/execution_run.go:1046`, `runtime_launcher.go:284`.
- **D-04 (Nit):** `Build` maps a `process` effect that `attemptEffectKinds`
  can never authorize. The `network` effect is authorizable but not mapped to
  any adapter control. Evidence: `runtime_launcher.go:89`,
  `internal/workflow/execution.go:914`.
- **D-05 (Nit):** canonical runner leases share `run-leases/v1` with legacy
  leases, so post-cutover inventories classify new leases as a legacy family.
  Evidence: `internal/local/execution_store.go:33`, `cutover.go:24`.

## Validation (final HEAD `045a97e` unless stated)

| Command | Result |
| --- | --- |
| `go build ./...` | PASS (exit 0) at `b945ecc` and `045a97e` |
| `go vet ./internal/... ./cmd/...`; `go vet ./...` | PASS at `b945ecc`; `go vet ./...` PASS at the corrected tree |
| `go run cmd/gofmt -l cmd internal scripts` | PASS (no files) |
| `go run ./scripts/check-architecture.go domain` / `application` | PASS (12 / 46 files) |
| Focused negatives: the F-01–F-05 tests above, the `stageexecution`, `local`, `workflow` and `projectapp` packages | PASS after the fixes; each new negative FAILED with its fix disabled |
| `go test ./... -count=1 -timeout=30m` | PASS at `045a97e` (exit 0; 32 packages `ok`, 0 `FAIL`). It also passed on the intermediate tree with F-01–F-03 applied |
| `git diff --check` | PASS |
| Curated staticcheck 0.8.1 | NOT_RUN locally: the tool cannot read go1.27.3 export data ("export data version 5"); CI is authoritative |
| `scripts/check-go-quality.sh`, `check-sensitive-files.sh`, `validate-repository.sh`, `test-docs-offline.sh` | NOT_RUN locally: shell approval unavailable in this session; CI PR checks on the pushed SHA are authoritative |
| Race detector | NOT_RUN |
| Vendor/E2E, installed cutover | NOT_RUN, not authorized |

## Readiness

Not ready for approval claims. Next gates: CI on the pushed SHA, then
`INDEPENDENT_REVIEW_PENDING`. That review should use a genuinely separate
reviewer, cover the ledger's unread rows, and re-review F-01–F-05. After that,
human pre-merge review. Product Accepted remains the later manual journey.
