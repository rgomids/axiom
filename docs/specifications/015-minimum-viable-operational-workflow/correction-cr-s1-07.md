# CR-S1-07 correction: independent full S1 review findings IR-01..IR-09

Attempt `CR-S1-07-CLAUDE` on [AXM-42](https://linear.app/rgomids/issue/AXM-42) /
[PR #366](https://github.com/rgomids/axiom/pull/366), from
[review 5730553478](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5730553478)
(analytical verdict REQUEST_CHANGES, native COMMENT). Executor: Claude Opus 5.5
(`claude-opus-5-5`, effort medium), direct work with no delegation. This is a scoped
correction, not a review: `INDEPENDENT_RE_REVIEW` is the next gate. The executor's
model is the same as the implementer's and the reviewer's, so model diversity is
still not achieved. [`review-s1-full.md`](review-s1-full.md) and
[`correction-cr-s1-06.md`](correction-cr-s1-06.md) stay historical, except for
the IR-07 note on their `git diff --check` rows.

## Identity

| Item | Value |
| --- | --- |
| Base / merge-base | `440e44a68638e08e901fd3f87f5b15cd395a6365` |
| HEAD before correction (local = origin = PR head; worktree clean apart from ignored `tmp/`) | `78d99f8f2bd8bf58a9337d493cc428aeb3fd6d5e` |
| Correction commits / final HEAD | recorded in the PR reply and the AXM-42 event |
| CR-S1-07-FINAL start / fixture commit | `8877b1613208d18b8aa6e06bc78f88cb8e16c879` / `912f54fe31f094aa9469734177f93d89b9df7a7e` (pushed); this Evidence follows as a docs commit |

## Findings and resolution

| ID | Severity | Status | Change | Regression scenario | Proof |
| --- | --- | --- | --- | --- | --- |
| IR-01 | Major | FIXED, TESTED | `Engine.Abandon` (`internal/stageexecution/engine.go`) holds the runner lease without claiming it. It re-observes every prepared, running or unknown attempt and every unfinished command through `observeRecorded`. It refuses with `ErrOrphanLiveness` / category `stop_orphan_then_abandon` on `process_alive_effects_unknown` or `observation_failed`, and writes nothing. It allows and reports `process_gone_effects_unknown`, `pid_reused_not_signalled` and `no_process_identity_recorded`, never signals, and commits under a revision CAS. CLI response, `recovery_required` next action (`cmd/lingo/execution_run.go`), diagnostic routing (`operator_diagnostic.go`), the `ResolveUnknown` contract note (`internal/workflow/execution_transitions.go`) and the docs are reconciled | A runner dies with its agent alive; abandon plus retry would run a second writer on the same workspace | `TestEngineAbandonRefusesLiveOrphanThenAllowsOneRetry` (live orphan: refused, revision unchanged, `Resolution` nil, not signalled, retry refused with 0 spawns; after the group is killed, abandon succeeds and the retry spawns exactly once), `TestEngineAbandonRefusesWhileRunnerHeld`, `TestEngineAbandonRefusesFailedObservation` (injected failure), `TestEngineAbandonReportsReusedPIDWithoutSignalling`, `TestEngineAbandonRefusesLiveUnfinishedCommand`, and CLI black-box `TestPublicReconcileAbandonRefusesLiveOrphanWithoutWriting` (category, bounded next action, Execution bytes unchanged, retry refused, abandon then exactly one retry after the stop) |
| IR-02 | Major | FIXED, TESTED with a local stub; vendor behavior NOT_RUN | `interruptibleRun` wraps `execution-run` (start and continuation) and `retry` in `signal.NotifyContext(SIGINT, SIGTERM, SIGHUP)`. A cancelled run context makes `OSProcessRunner` stop and confirm its own verified group: `sameLeader` refuses a reused PID and the stop is bounded by `processStopBound`. Every fenced attempt commit uses a bounded `context.WithoutCancel` (`terminalCommitBound` = 10s), which keeps context values and the runner fence. An interrupt records the runner's own cancel intent (`runner:interrupted`) in the same commit as the observed stop: `cancelled` (`group_stop_confirmed`, or `cancelled_before_spawn` before any start) or `unknown` (`cancel_stop_unconfirmed`) when the stop is unconfirmed | A Ctrl-C, SIGTERM or SSH hang-up kills only the runner; the agent's own group survives and stays `running` with no Axiom stop path | Black-box `TestPublicWorkflowRunSignalStopsOwnedGroupAndCommitsStop`: the built `lingo` binary runs a `/bin/sh` stub agent and receives SIGINT, SIGTERM and SIGHUP. It exits 2 (`workflow_cancelled`), `kill(-pgid, 0)` returns ESRCH, and the attempt is `cancelled`/`group_stop_confirmed` with Execution `cancelled`. Unit: `TestEngineInterruptedRunnerCommitsObservedStop` (confirmed → cancelled; unconfirmed → unknown; the commit succeeds with a cancelled caller context) and `TestEngineInterruptBeforeSpawnRecordsCancelledWithoutSpawning` |
| IR-03 | Minor | FIXED, TESTED | The terminal closure takes the cancel branch on `cancelRequested \|\| x.Cancel != nil`, using the reloaded aggregate. CAS and idempotency are unchanged, and the unconfirmed-command check still applies first | A cancel persisted after the last poll made `CompleteAttempt` return `ErrCancelled`: nothing was saved and the attempt stayed `running` | `TestEngineCancelPersistedBeforeTerminalCommitIsRecorded`: a store wrapper persists the cancel immediately before the terminal `UpdateFenced`. The attempt ends `cancelled`/`exited_after_cancel_result_refused` and a later runner finds nothing orphaned |
| IR-04 | Minor | FIXED, TESTED | The validation-command environment (`internal/stageexecution/runtime_launcher.go`) appends the attempt's own closed correlation set. That set is `AttemptCorrelation.Environment()`, exported from `internal/runtimeadapter/stage_invocation.go`, and is the same set the agent receives. `StageEnvironment` still drops inherited correlation and secret-bearing variables, and `CODEX_HOME`/`CLAUDE_CONFIG_DIR` are still removed. Advisory only: D-01 stays deferred | Workspace code edited by an agent and run as a command could call `axiom … --authorize-human` without tripping the child guard | `TestRuntimeLauncherCommandCarriesAttemptCorrelation`: the stub command sees the exact attempt and Execution IDs (not an inherited forged value), and no `PRIVATE_TOKEN` or `CLAUDE_CONFIG_DIR`. `TestRuntimeChildMarkersRefuseHumanAuthority`: either marker refuses `--authorize-human` before any write. The existing review test covers `AXIOM_ATTEMPT_ID` on `workflow run` |
| IR-05 | Minor | FIXED, TESTED | `freezeCommands` resolves `argv[0]` with LookPath, Abs and `filepath.EvalSymlinks`. It then requires `stageexecution.CleanRuntimePath` (the launcher's own rule, newly exported) before any Execution exists, and freezes the symlink-free target. An unmappable or dangling link is refused as `workflow_unmappable` with zero writes. Collect's containment is unchanged | A `python3`-style link was refused after the agent ran, and the attempt became `unknown`/`output_collection_failed` for a command that never started | `TestFreezeCommandsResolvesSymlinkedExecutableBeforeExecution`: a symlinked executable freezes to its target, passes the unchanged `Collect` check and runs, and a dangling link is refused |
| IR-06 | Minor (governance) | JUSTIFIED (no code change) | Evidence only, recorded as TD-07-6 below. The allowlist is not changed | — | Base `440e44a` fails `check-architecture.go domain` (`encoding/hex`) and `application` (`strconv`) with its own allowlist, re-run locally in a temporary detached worktree. HEAD passes both |
| IR-07 | Nit | FIXED | `linear-comment-ready.md:19-22`: the trailing-space hard breaks become CommonMark backslash breaks, and the breaks before or inside the list are dropped. Rendering is unchanged. The PASS rows in `review-s1-full.md:149` and `correction-cr-s1-06.md:70` are annotated with their real scope, the unstaged worktree only. The whitespace came from `74b6684`/`b50ccca` | — | `git diff --check 440e44a...<final HEAD>` result below |
| IR-08 | Nit | FIXED | The attempt-output root is created through `os.Root.MkdirAll` on the state root instead of a path-based `os.MkdirAll`, so no component can redirect creation outside the state root. Build's symlink refusal is unchanged. The cutover namespace classification is unchanged and consistent: `executions/v2` itself is already a post-cutover namespace that BLOCKs a pre-switch inventory, and the output root exists only after new runs | Stray creation through a redirected component | Existing launcher/run tests (`cmd/lingo`, `internal/stageexecution`) |
| IR-09 | Nit | FIXED | stdlib and module imports are grouped in `internal/local/workflow_store.go` and `historical_execution_formats.go` | — | gofmt and build |

Prior corrections re-verified by the full suite at the final tree: F-01
(`TestEngineCancelDuringValidationCommandNeverHidesUnconfirmedStop`), F-02
(agent correlation, unchanged and now shared with commands), F-03, F-04, F-05
and CR-S1-06 (`TestCutover*`).

## technical_decisions[]

- **TD-07-1 (IR-01):** Abandon holds the lease but does not claim it, so a
  refusal writes no generation or revision. Commit uses a lease-free CAS pinned
  to the observed revision. Alternative: Claim, then observe. That bumps the
  revision on refusal, which the acceptance tests forbid. Rollback: revert the
  IR-01 commit.
- **TD-07-2 (IR-01):** Observe every prepared, running or unknown attempt of
  the Execution, not only the target. A sibling's live process also forbids a
  disposition that unblocks scheduling. Alternative: target only. That is
  weaker, and only the derived status blocks siblings.
- **TD-07-3 (IR-02):** An operator signal is recorded as the runner's own
  cancel intent (`runner:interrupted`), which makes the Execution cancelled.
  This reuses the only existing truthful `cancelled` transition and is
  fail-closed: nothing is scheduled after an interrupt. Alternative: a
  retryable `failed` attempt, like `attempt_timeout`. Rejected because it reads
  as an agent failure and the review asked for `cancelled`. Trade-off: an
  accidental hang-up ends the Execution, and a new `workflow run` is needed.
  Rollback: revert the IR-02 commit.
- **TD-07-4 (IR-02):** All fenced commits of an attempt, not only the terminal
  one, use the bounded detached context. Otherwise an interrupt racing
  `MarkStarted` or a command record would lose an observed fact and leave the
  attempt `prepared`/`running`. Reads and the Claim still honor cancellation.
- **TD-07-5 (IR-05):** Freeze the resolved symlink-free target, not refuse
  every link. Refusing would make S1.5 commands unusable on merged-`/usr`
  hosts. Containment is not relaxed: the frozen path itself must pass the
  unchanged launcher rule. Limitation: multi-call executables that dispatch on
  `argv[0]` (busybox-style) receive the target name. Configure the target
  directly or a wrapper script.
- **TD-07-6 (IR-06):** Keep `05c5787`'s allowlist as is and record its
  rationale. The S1-02, S1-05 and S1-06 claims ("no allowlist widening";
  "required check remains FAIL") were accurate when written (`f719465` is
  immediately before `05c5787`). `05c5787` then resolved that deferred finding
  without recording a decision. The added entries transform supplied values
  only:
  - stdlib: `Sprint`, `Sprintf`, `Itoa`, `bytes.Contains`/`Equal`, slices
    helpers, `json.Unmarshal`, `hex.DecodeString`, `context.WithoutCancel`/`WithValue`;
  - the Workflow-definition, Runtime-Profile, `strictjson` and `portableconfig`
    codecs and selectors, which resolve no executable and read no file;
  - `runtimeprofiletest`, admitted as a test-only import.

  The allowlist stays symbol-level. `fmt.Print*`, `time.Now`, the filesystem,
  env and process remain denied. The base failure was pre-existing on `main`
  (verified above). No widening in this correction.
- **TD-07-7 (IR-08):** Fixed via `os.Root` because the change is one call and
  matches the rooted policy. The cutover catalog is not touched. Its existing
  classification already blocks post-cutover namespaces by design.
- **TD-07-8 (FINAL, leak):** Bound the fixtures themselves (a parent-bound,
  time-capped, non-spinning hang), not the runtime. The leak is in test
  ownership. The runtime's group stop on caught signals is proven, and runner
  SIGKILL survival is the accepted DF-07-2 model. Alternatives rejected:
  - `Pdeathsig` on the production agent child. It changes S1 orphan semantics,
    reaches only the direct child (not the group), and needs an ADR-0037
    reconsideration;
  - a cgroup or supervisor. That is a new OS dependency or daemon;
  - relying on `t.Cleanup` or `NotifyContext`, which cannot run on SIGKILL.

  Rollback: revert the fixture commit.
- **TD-07-9 (FINAL, leak):** `Pdeathsig: SIGKILL` on the test's runner, not
  SIGTERM. The hard-kill chain must not depend on runner cooperation; the
  graceful path is already proven by the signal test. Caveat: Linux delivers
  `Pdeathsig` when the forking OS thread exits. Go retires a thread only when a
  locked goroutine exits, which these tests never do, and `boundedHang`'s 120s
  cap backstops it.
- **TD-07-10 (FINAL, leak):** The proof runs outside the killed binary. An
  observer re-executes the test binary as the victim, and any emergency stop
  counts as a FAIL, so a cleanup cannot mask a leak. `boundedHang` is duplicated
  as a const in the two test packages (`cmd/lingo`, `internal/stageexecution`)
  rather than adding a shared test package for five lines.

## deferred_findings[]

| ID | Stage | Status | Evidence / reason |
| --- | --- | --- | --- |
| D-01 | S2 transport | DEFERRED_NOT_IMPLEMENTED | Covers a Codex child stripping the markers and child use of `--authorize-external`/`--authorize-local`. The env guard stays advisory. Only the command-env piece (IR-04) was in scope |
| D-02 residual | operator cutover | DEFERRED_NOT_IMPLEMENTED (by design) | No public cutover command. A real cutover stays NOT_RUN and separately authorized |
| D-04 | S2 | DEFERRED_NOT_IMPLEMENTED | `process` mapping dead, `network` not mapped; no guarantee claimed |
| D-05 | later | DEFERRED_NOT_IMPLEMENTED | Unchanged. Its IR-08 extension is fixed |
| DF-IR-01 | pre-merge decision | DEFERRED_NOT_IMPLEMENTED | Frozen executable recorded by path, not content identity. ADR-0037 requires an "executable reference"; flagged for the human pre-merge review |
| DF-07-1 | skills | DEFERRED_NOT_IMPLEMENTED | `internal/codexruntime/skills/axiom-work-item/SKILL.md` does not mention the abandon refusal. Its digest is pinned in the manifest, so editing it is skill-release churn. The CLI response carries the bounded next action |
| DF-07-2 | process boundary | DEFERRED_NOT_IMPLEMENTED | A SIGKILL/OOM of the runner still leaves the agent group alive by construction. IR-01 now makes that safe: abandon is refused until the operator stops it. An automatic supervised stop would need ADR-0037 reconsideration. CR-S1-07-FINAL bounds only the test stubs, not vendor agents |
| DF-07-3 | test portability | DEFERRED_NOT_IMPLEMENTED | `boundedHang`'s Darwin fallback (`kill -0 $PPID`) and the stub fixtures on macOS were not run on this host. The observer harness is Linux-only |

## Evidence

Host `ubuntu-arm-01`, Go 1.27.3 linux/arm64. Results are recorded against the
final tree; the PR reply names the SHA.

| Check | Result |
| --- | --- |
| Mutation checks (one fix line reverted at a time, focused tests re-run, fix restored) | IR-01: abandon refusal removed → 3 Abandon tests FAIL. IR-02: `WithoutCancel` removed → the interrupted tests FAIL with the attempt left `running`/`prepared`. IR-02 CLI: `NotifyContext` removed → the black-box test FAILs with `signal: interrupt/terminated/hangup`, and the stub agents demonstrably **survived their runner**, reproducing the review's scenario. IR-03: reloaded-cancel guard removed → FAIL. IR-04: markers removed → FAIL (`"\|\|unset\|unset"`). IR-05: `EvalSymlinks` removed → FAIL |
| Focused negatives IR-01..IR-05 (tests above) | PASS |
| `go build ./...`, `go vet ./...` | PASS |
| `go run cmd/gofmt -l cmd internal scripts` | PASS (no files) |
| `go run ./scripts/check-architecture.go domain` / `application` | PASS (12 / 46 files) |
| `go test ./... -count=1 -timeout=30m` | PASS (exit 0; 32 packages `ok`, 0 `FAIL`) on the code tree of `eeba7d9` (`cmd/lingo` 387s, `internal/local` 197s, `internal/stageexecution` 41s, slowed by host load). Intermediate commits `722721d` and `896a6fb` were each built and vetted in a detached worktree |
| `git diff --check 440e44a...<final HEAD>` | PASS (exit 0, no output); the exact final SHA is in the PR reply |
| `check-sensitive-files.sh`, `validate-repository.sh`, `check-go-quality.sh`, `test-docs-offline.sh`, staticcheck, race detector | NOT_RUN locally (shell-script execution needs approval in this session); CI on the pushed SHA is authoritative |
| Vendor (Codex/Claude) child behavior on EPIPE/SIGPIPE, paid E2E, installed-state cutover | NOT_RUN, not authorized. All process evidence uses `/bin/sh` stubs |
| FINAL baseline: `go test ./... -count=1 -timeout=30m` at `8877b16` (temporary detached worktree, since removed) | PASS (exit 0; 32 `ok`, 0 `FAIL`). A normal run leaves no stub; the leak needs an abnormal death |
| FINAL before-fix observer: `go test ./cmd/lingo -run '^TestKilledTestBinaryLeavesNoOwnedProcess$' -count=1` on the `8877b16` fixtures | FAIL as expected. All five death cases leaked the runner and the stub; see the case matrix |
| FINAL after-fix observer: the same run with `-count=3` on `912f54fe31f094aa9469734177f93d89b9df7a7e` | PASS (exit 0, 54.9s; 7 cases × 3, no emergency stop) |
| FINAL mutation checks (`Pdeathsig` removed; parent check removed) | FAIL as expected, fix restored; see above |
| FINAL `go test ./... -count=1 -timeout=30m` at `912f54f` | PASS (exit 0; 32 `ok`, 0 `FAIL`; `cmd/lingo` 224s, `internal/local` 137s, `internal/stageexecution` 23s). This includes IR-01..IR-05, F-01..F-05, CR-S1-06, cancel/abandon/liveness, signal group, command env markers and symlink freeze |
| FINAL `go run cmd/gofmt -l cmd internal scripts`, `go build ./...`, `go vet ./...`, `check-architecture.go domain`/`application` at `912f54f` | PASS (no files; PASS 12 / 46 files) |
| FINAL owned-process inspection after each full suite (`pgrep -f 'bin/codex\|\.test \|lingo --json'`) | No owned process left, apart from PID 1809586: the before-fix reproduction stub described above, handed to the operator |
| FINAL shell validators (`check-go-quality.sh`, `validate-repository.sh`, `check-sensitive-files.sh`, `test-docs-offline.sh`), staticcheck | NOT_RUN locally (shell execution needs approval in this session); CI on the pushed SHA is authoritative |

## CR-S1-07-FINAL: stub agents surviving a dead test binary

Attempt `CR-S1-07-FINAL-CLAUDE`, Claude Opus 5.5 (`claude-opus-5-5`, effort
medium), direct work with no delegation, from PR head
`8877b1613208d18b8aa6e06bc78f88cb8e16c879`. It addresses the operator-reported
incident: three fake `…/bin/codex` stubs from
`TestPublicWorkflowRunSignalStopsOwnedGroupAndCommitsStop/{interrupt,terminate,hangup}`
were found alive about 4h after their test run, at ~32% CPU each. The scope is
test fixtures only. No product code, public flag, ADR or S1 safety boundary
changes.

**Root cause (reproduced).** Every hanging fixture was an endless shell loop
(`while :; do :; done`, which spins at full CPU) placed in its own process group.
That is the ADR-0037 containment the runtime under test must use. Only two
things could stop it: an in-binary `t.Cleanup`, or the runner's own signal
handler. Neither runs when the test binary dies by SIGKILL, OOM or a
`-test.timeout` panic. Neither runs when a `t.Fatal` fires before the stub
cleanup is registered, because then only the runner is killed. In the black-box
signal test, the built `lingo` runner also outlived a dead test binary: nothing
tied it to its parent, so it kept waiting on its never-ending stub. The same
unbounded pattern existed in the IR-01 orphan stand-ins: `read -t 1` on
`/dev/null` is rejected by dash, so that loop spins too. It also existed in
`internal/stageexecution` (`engine_ported_test.go` orphans and the
command-cancel stub). The exact trigger of the reported incident was not
re-observed. It is consistent with the IR-02 CLI mutation run recorded in
Evidence above: `NotifyContext` was removed, all three runners died from their
signal, and the three subtest names and the ~4h timing match. The runtime itself
did not leak on caught signals: the existing signal test and the `complete` case
below both confirm the group stop.

**Fix (test-only).**

1. `boundedHang` replaces every unbounded fixture loop: `stubRuntimeScript`
   `mode:hang`, the IR-01 CLI orphan, the two `stageexecution` orphans and the
   command-cancel stub. It sleeps via `/bin/sleep 0.2` (no spinning) only while
   the shell keeps the parent it started with, and for at most ~120s (600
   iterations, then `exit 9`). On Linux the current parent comes from
   `/proc/$$/stat` field 4, which is immune to PID reuse. Elsewhere it uses
   `kill -0 $PPID`. When the parent dies the kernel re-parents the shell, so the
   loop ends even if no cleanup ever runs.
2. The black-box runner (`startHangingRunner`, shared by the signal test and the
   harness) starts with `Pdeathsig: SIGKILL`. Go re-checks the parent after
   `PR_SET_PDEATHSIG`, closing the fork/exit race. A dead test binary therefore
   kills the runner, and the stub then leaves `boundedHang`.

**Independent-observer harness.** `TestKilledTestBinaryLeavesNoOwnedProcess`
(`cmd/lingo/execution_orphan_leak_test.go`, Linux) builds `lingo` and
re-executes its own test binary as a victim (`TestLeakVictimHelper`, skipped
unless invoked by the observer). The victim runs a real foreground runner on a
hanging stub agent, in its own group with `TMPDIR` under the observer's temp
directory. Before acting, the observer reads `/proc` and requires this exact
chain: victim → runner (`ppid` = victim), runner → stub (`ppid` = runner,
`pgid` = stub). It records the kernel start identity of the runner, the stub
and every stub-group member alive at that moment. The victim proceeds only
after the observer has recorded the family. The observer then kills, fails or
times out the victim and waits for its exit. It requires, within 10s, that no
recorded identity is still alive and that no live process remains in the stub's
group. A survivor triggers an emergency stop: exact re-observed identities only,
and the stub group only through its still-identical leader. That stop is a
**FAIL**, never a pass. The observer signals only the victim PID, its own
unreaped child.

| Case | Victim death | Before fix (`8877b16` fixtures) | After fix |
| --- | --- | --- | --- |
| SIGINT to test binary | killed by signal (Go default action, no cleanup) | FAIL: runner and stub alive after 10s, emergency stop | PASS ×3, nothing left after 151–211 ms |
| SIGTERM | killed by signal | FAIL: same | PASS ×3, 153–214 ms |
| SIGHUP | killed by signal | FAIL: same | PASS ×3, 196–202 ms |
| SIGKILL (hard kill, no handler or cleanup can run) | killed | FAIL: same | PASS ×3, 150–191 ms |
| `-test.timeout=12s` panic | exit 2, no cleanup | FAIL: same | PASS ×3, 68–198 ms |
| forced `t.Fatal` before the stub cleanup | exit 1, only the runner cleanup runs | Stub left with `ppid=1`, state `R` (spinning): the incident signature | PASS ×3, 124–153 ms |
| normal completion (runner SIGTERM, group stop committed) | exit 0 | Not observed (harness race, fixed by the handshake) | PASS ×3, 8–18 ms |

Mutation checks against the fixed tree (`SIGKILL` and `forced_failure` cases,
fix restored afterwards):

- `Pdeathsig` removed → SIGKILL FAILs with the runner and stub alive.
  `forced_failure` passes, because the cleanup kills the runner.
- The parent check removed from `boundedHang` → both FAIL, with the stub group
  alive.

Each mechanism is therefore necessary. The before-fix `forced_failure` stub (PID
1809586) aborted the observer's ownership proof before the family was recorded,
so it was not stopped. This session cannot signal arbitrary PIDs, so it is
handed to the operator, identified by its unique run path.

**Guarantee scope.** Proven for the controlled stub family on Linux: the test
binary, its `lingo` runner, the stub agent and the stub's own group, under
SIGINT, SIGTERM, SIGHUP, SIGKILL, timeout panic, forced failure and normal
completion. **NOT_PROVEN**, and not provided:

- a vendor agent child's survival when a production runner is SIGKILLed. That is
  by construction (DF-07-2) and handled by IR-01's abandon refusal;
- descendants that leave the stub group;
- Darwin. The `kill -0` fallback was not run there, and `Pdeathsig` is
  Linux-only (the black-box signal test file is Linux-only).

## Security and trust boundaries

- No new public CLI flag, service, ADR, migration or Provider write. The new
  refusal category is bounded and payload-safe (reconciliation facts only).
- Abandonment can no longer authorize a concurrent writer. The env-marker guard
  is still advisory (D-01).
- Signal handling stops only the verified own group (`sameLeader`). It never
  signals a reused PID, and confirmation is never inferred from signal delivery.
- The frozen command path is symlink-free at freeze time and re-checked at use.

## Remaining limitations

- Vendor CLIs were not exercised. Whether a real Codex/Claude child would have
  died on its own after a runner loss stays unverified (EVIDENCE_GAP). It no
  longer matters for an interrupt that the runner catches, and IR-01 covers
  SIGKILL.
- The no-orphan guarantee after a dead test binary covers only the controlled
  stub family on Linux (CR-S1-07-FINAL). A vendor agent outliving a SIGKILLed
  production runner is NOT_PROVEN and is not prevented (DF-07-2). Darwin is
  NOT_RUN (DF-07-3).
- An interrupt during Claim, before any attempt is prepared, exits with
  `workflow_cancelled` and records nothing, which is correct because nothing ran.
- Independent re-review of CR-S1-07, then the human pre-merge review. No merge,
  release, cutover or acceptance is claimed.
