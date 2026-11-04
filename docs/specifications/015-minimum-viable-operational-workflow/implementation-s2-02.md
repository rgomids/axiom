# S2.2 — Cross-Runtime messages through the canonical attempt ingress

Attempt: `IMPLEMENT-S2-02-CLAUDE`. This Evidence records the bounded S2.2
delivery on the existing AXM-42 / PR #366 branch. It does not claim S2.3,
S2.4, S3, human acceptance, merge, release or actual installed-state cutover.

## Authority, baseline and preflight

Task S2.2 of the accepted [Tasks](tasks.md): "Route cross-Runtime
question/answer/blocker/result through S1 ingress and common Codex/Claude
adapter contracts; remove independent coordination stream/lifecycle and split
publish resolvers." Acceptance: deterministic Codex/Claude adapter stubs
exchange bidirectional correlated messages and unblock an eligible dependency
only on a validated result; wrong attempt, forged sender or stale fence cannot
unblock; duplicate replies are idempotent. Dependencies S2.1
([Evidence](implementation-s2-01.md)) and S1.6 are complete. Baseline:
[Plan](plan.md) TD-03 and Research L, [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md)
§6 (messages are bounded attempt-owned facts; content never grants authority).
Design UX: NOT_APPLICABLE. No new requirement, ADR, CLI flag, store, format
or dependency.

Preflight: clean worktree on `spec/axm-42-minimum-viable-workflow`, origin
`https://github.com/rgomids/axiom.git`; local, fetched origin and PR #366 HEAD
all `d8434e4a4d2f10da8f3ae50b481ef4cb0d2b47b9` (draft, open); merge-base with
`main` `440e44a68638e08e901fd3f87f5b15cd395a6365`. No other writer was
observed; process-table inspection was not permitted by the runner, so this
is not a machine-wide exclusion claim.

## Implementation

Source commits (both on top of the pinned baseline):

- `180c14dfcbf2c43f3e4602ab5efcef799ab196a1` — routing.
- `3e83d5fa794273f0bab9bb8005ca7ed865e1d522` — retirement.

**Common adapter contract.** `RuntimeLauncher` — the one launcher both the
Codex and the Claude Runtime use — creates two private areas inside the
launcher-issued attempt inbox (`@send`, `@recv`; `@` is not a token character,
so they cannot collide with declared outputs) and advertises them in the same
stdin packet for both Runtimes. It implements the optional `Exchange`
interface: `Poll` reads only bounded (5 KiB), regular, non-symlink,
strictly-decoded `<id>.json` candidates whose name matches their ID, refusing
the whole area on unsafe permissions or more than 256 entries; `Deliver`
writes an accepted message addressed to the attempt's agent by exclusive create
and rename, idempotent for identical bytes and refusing conflicting or
misaddressed deliveries. Moving bytes is all it does.

**One ingress, principal-owned.** After the attempt's process identity is
durable, the engine runs a message pump per running attempt. It binds every
candidate to the engine-issued `IngressContext` derived from the persisted
attempt (Execution, attempt, agent, frozen Runtime, runner generation) — the
child never names its sender — and commits it with `Execution.AcceptMessage`
through the S2.1 principal writer under the runner fence. One final drain runs
after a clean exit and before the terminal commit; cancelled, timed-out,
uncertain or spawn-failed attempts get none. Accepted messages addressed to the
attempt's agent are delivered into its `@recv`. Per-candidate refusals and
exact duplicates change nothing and do not stop scheduling (`scheduler.go`);
a lost fence or store failure is recorded by the principal, and a messaging
failure left at the final drain fails the attempt (correction S22-CR-01 below).

**Ingress rules added.** `IngressContext.Runtime` must equal the attempt's
frozen Runtime (a Codex attempt cannot speak as a Claude attempt and vice
versa). An answer whose asking attempt is no longer running is refused as late;
an exact redelivery of an already accepted answer remains idempotent. Existing
rules are unchanged: sender attempt running, matching agent/generation, no
cancel, known non-self recipient, bounded content, four kinds, one answer per
question from its addressee, duplicate ID conflict refused, 256-message cap.

Readiness is unchanged: dependencies require succeeded attempts with validated
outputs. A `result` message is data and cannot complete a task, release a gate,
restore admission or substitute for the fence.

### Retirement inventory

Removed only after the replacement above had producers and consumers and its
negative tests passed (commit order).

| Removed | Role | Former producers → consumers | Replacement |
| --- | --- | --- | --- |
| `internal/coordination/inbox.go` (102) | Split publish resolver: `InboxPublisher`, `ChildResolver`, `ChildContext`, `AttemptInbox` port | `local.AttemptInbox`, `coordination.Ingestor`; `cmd/lingo` used only `ErrChildContext` | Engine-issued `IngressContext`; launcher `@send` area |
| `internal/coordination/ingest.go` (188) | Independent drain/classify lifecycle, epoch fence | `local` inbox consumer; tests only | Engine message pump + final drain under runner fence |
| `internal/coordination/read.go` (104) | Stream `Reader` list/show/wait | `cmd/lingo` used only two error values; tests | Canonical `Execution.Messages` inspection (already in `stage_service.go`) |
| `internal/local/attempt_inbox.go` (240) | Private inbox journal (create/append/seal) | tests only | `RuntimeLauncher` `@send`/`@recv` + `Exchange` |
| `internal/local/coordination_ingest.go` (212) | Inbox consume, ingestion state, `CoordinationFence` | tests only | Principal-writer commit under runner fence |
| `read_test.go`, `attempt_inbox_test.go`, `coordination_ingest_test.go` | Retired-only tests | — | Safety cases ported to `exchange_test.go` (cross-Runtime Q/A/blocker, final-drain race, late/forged/oversize/symlink/flood) |

Removed exported API: 14 types, 13 functions/methods, 5 error values
(`ErrChildContext`, `ErrAttemptSealed`, `ErrEpochFenced`, `ErrRecordNotFound`,
`ErrInvalidWait`). `cmd/lingo` keeps its `child_context_required` and
`record_not_found` categories with two local error values; the CLI publish
leaf still always refuses (it has no attempt context). No CLI leaf, flag or
JSON contract changed.

Deliberately **retained**: the format-1 `coordination` Record/Stream codec,
`Service`, `AcceptanceVerifier` and `local.CoordinationStore`. The S1.8 cutover
inventory reads legacy streams (`inventory.go`: `DecodeStream`,
`coordinationName`), and its isolated cutover/compatibility tests use the store
to write legacy fixtures. Retiring their remaining duplicate representations
belongs to S2.4/S3.4. `runtimeadapter.TaskPacket` (no production consumer) is
likewise left for S2.4 — see deferred findings. No generic security guard,
process-management or S3 delivery owner was touched.

Delta against the pinned baseline (`cmd`, `internal` Go files): production
+355 / −858 lines, net **−503**, files −5 / +2; exported types −14 / +1
(`Exchange`), functions −13 / +2 (`Poll`, `Deliver`). Tests +558 / −858 lines,
files −3 / +1. Physical lines include comments/blanks.

## Deterministic proof

All tests use isolated temporary state roots, the real canonical local
`ExecutionStore`, the real `RuntimeLauncher` Build/Poll/Deliver/Collect with
non-executed stub executables named `codex`/`claude`, and an in-process fake
process runner. No OS process, vendor binary, credential, network or CLI is
used; `assertNoChildProcesses` checks `/proc/self/task/*/children` is empty
after each case (zero test-owned descendants, so no PID/start/PGID to track).
This is not evidence of vendor parity.

- `TestExchangeCodexClaudeBidirectionalUnblocksOnlyOnValidatedResult`: a Codex
  child and a Claude child, concurrently in one stage, exchange q1→a1
  (Codex→Claude) and q2→a2 (Claude→Codex), a blocker before an answer, and
  results to a dependent reviewer. Each stored message is attributed to its own
  sender attempt; each child ran its own Runtime executable. A genuine replay of
  the accepted a2 via canonical ingress returns `message_duplicate` with an
  unchanged revision; a conflicting replay of a1 leaves `v4` stored. Forged
  `sender` member, misnamed file, unmatched reply, wrong-correlation answer,
  unknown and self recipient, unknown kind (`approval`), 4 KiB+1 content and an
  over-bound file are never stored. The reviewer spawns exactly once, after both
  build attempts succeeded with validated output digests, and receives b1/r1/r2.
  After the run, late (terminal attempt), forged agent, crossed Runtime and
  stale-generation ingress are refused with no write.
- `TestExchangeMessagesCannotUnblockDependency`: with result/blocker messages
  claiming success or gate approval, the reviewer never spawns when Claude's
  output is missing (run failed), the human gate is unapproved (awaiting gate,
  gate nil), admission is revoked (reviewer attempt failed
  `admission_revoked_before_spawn`, no process), or the runner fence is lost
  (`stale_runner`; only the pre-loss question is stored).
- `TestRuntimeExchangeAreaSafety` (Codex and Claude): packet advertises both
  areas; symlinked and partial candidates are ignored; identical redelivery is
  idempotent, conflicting or misaddressed delivery refused; flooded send area
  and non-private receive area refused.
- `TestLateOrUnmatchedAnswerIsRejected` and the extended
  `TestTwoStubQuestionAnswerIngress` (crossed Runtime) at the aggregate.

Implementer validation, worktree at the source commits, ambient
`go1.27.3 linux/arm64` (`GOROOT=/usr/local/go`; `go.mod` declares 1.26.0):

| Command | Result |
| --- | --- |
| `go test ./internal/stageexecution -run 'TestExchange\|TestRuntimeExchange' -count=20 -timeout=300s` | PASS, exit 0, 44.8s (after one fix, below) |
| `go test ./internal/stageexecution ./internal/workflow -count=1 -timeout=600s` | PASS, exit 0 (includes S1.4/S1.6 and S2.1 `TestEngineParallel*` regressions) |
| `go test ./internal/workflow -run 'TestLateOrUnmatched\|TestTwoStub\|Message' -count=1` | PASS, exit 0 |
| `go test ./internal/local -run TestExecutionStore -count=1` | PASS, exit 0 |
| `go test ./internal/coordination ./internal/local ./internal/executiongraph ./internal/runtimeadapter -count=1 -timeout=900s` | PASS, exit 0 (after retirement; includes cutover inventory tests) |
| `go test ./cmd/lingo -run 'TestOperatorStageDiagnostic\|Publish\|Coordination\|Message\|Stage' -count=1` | PASS, exit 0 |
| `go build ./...` / `go vet ./...` | PASS, exit 0 |
| `go run ./scripts/check-architecture.go domain` / `application` | PASS, exit 0; 12 / 46 files |
| `git diff --check 440e44a…HEAD` and pending patch | PASS, exit 0 |
| `gofmt -l cmd internal scripts` | AUTHORIZATION_REQUIRED (runner). Substitute: temporary untracked `go/format` comparison program over the same roots, 0 files differ; program removed, never committed |
| `go test ./... -count=1 -timeout=30m` | PASS, exit 0, at `3e83d5f`: cmd/lingo 163.6s, local 78.1s, workflow 68.6s, install 55.0s, stageexecution 22.4s; every package ok |

Retained unsuccessful observation: the first 20× repetition of the dependency
test failed in `invalid result` (Codex's `r1` missing in some runs). Cause: a
poll interrupted by the pump's own shutdown marked the area failed, skipping
the final drain. Fixed in `exchange.go` (context cancellation no longer
fail-closes the area) before commit; the 20× run above is after the fix. No
assertion was weakened.

Retained CI failure: [CI run 41500519022](https://github.com/rgomids/axiom/actions/runs/41500519022)
at `84c3159ce7e7da97e293a70948e7bc2cfa08912e` failed in pinned Staticcheck
v0.8.1: `internal/local/s8_stores_test.go:99:6: func opaqueID is unused (U1000)`.
The retirement commit deleted that test helper's only caller. Later CI steps
did not run in that job. The fix removes the helper only; `go vet` and
`go test ./internal/local` pass afterwards (exit 0, 51.4s). Delivery metadata at that
SHA: success. The final-SHA CI result is reported in the PR comment and
completion receipt, never inferred here.

Not run locally, with reason:

- `./scripts/check-go-quality.sh`, `bash scripts/validate-repository.sh .`,
  `./scripts/check-sensitive-files.sh .` / `--staged .`,
  `python3 scripts/check-adr-governance.py .`,
  `python3 scripts/check-automation-registry.py .`,
  `./scripts/check-claude-bootstrap.sh --skill-adapters .` and pinned
  `staticcheck`: **AUTHORIZATION_REQUIRED** — the runner's permission policy
  denies them in this session. Narrow request: `AXM_EXTRA_TOOLS` entries for
  exactly these scripts and `gofmt`. Substitute for the staged sensitive scan:
  `git grep --cached` for token/key/private-key/home-path shapes over the
  changed source files, no match (not equivalent to the script). Format,
  Staticcheck v0.8.1, `validate-repository.sh` and docs-offline run in PR CI;
  their result is the exact-SHA CI read-back, never inferred here.
- Isolated `PATH` without vendor CLIs: per-command `env PATH=… GOROOT=…` was
  denied. The ambient `PATH` contains user-local `codex`/`claude`; no test here
  executes a vendor path (fake runner only; launcher stubs are never executed).
- `-race`: NOT_RUN; S2.1 established this host lacks `gcc` (cgo). `gitleaks`:
  NOT_RUN, unavailable on S2.1's identical host; not re-probed.
- Live Codex/Claude, paid E2E, real cutover: outside authority.

## Correction S22-CR-01 (attempt CORRECT-S22-CR-01-01)

Finding ([review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5741505243),
Major): the message pump silently disabled or ignored messaging failures, and
a child that then exited zero with valid outputs was still recorded
`succeeded`. **Reproduced** at `defbac43ac0efcf779c2964439e4841a632fb06a` by
`TestExchangeFailureIsSurfacedOnTheAttempt` before any production edit:

| Case | Observed at `defbac4` (red) |
| --- | --- |
| poisoned send area (Claude child floods `@send`, >256 entries) | Claude `succeeded`, diagnostic empty; reviewer spawned |
| conflicting delivery (Claude child plants a different `@recv/q1.json`) | Claude `succeeded`; reviewer spawned |
| persistent ingress failure (every message commit refused by the store) | Codex `succeeded`; `r1` lost; only the run returned the store error |
| transient ingress failure (first message commit refused once) | `r1` never retried, lost for good |

Causes in `exchange.go`: a `Poll` refusal set `stopped` with no result; a
failed commit had already marked the candidate submitted, so it was never
retried, and then stopped routing; `load` and `Deliver` failures returned
silently; `startMessages` returned nothing to `runAttempt`.

**Fix (smallest in-scope).** `messagePump.step` returns a fixed engine code
for work it left undone; `startMessages`' stop function returns the code of the
final drain; `runAttempt` passes it to `CompleteAttempt` as the new in-memory
`AttemptResult.Fault`, which marks the attempt `failed` with that code as the
diagnostic whatever the exit code (unknown effects still override to
`unknown`). The observed exit code, artifacts and effects stay recorded. No
coordinator, queue, store, CLI, schema or codec change; `Fault` is not
persisted on its own (only the existing `diagnostic`).

Classification (only the final drain decides the attempt; earlier faults are
retried by the next tick):

| Code | Source | Class | Handling |
| --- | --- | --- | --- |
| `message_send_area_refused` | `Poll` refusal (unsafe area, entry flood) with a live context | persistent, child-induced | sticky: routing stops for the attempt, nothing in the area is accepted |
| `message_ingress_failed` | non-refusal commit error (store failure, lost fence) | transient (retried) | candidate not marked submitted, retried next tick; the principal still records the store error and stops new scheduling (unchanged `scheduler.go`) |
| `message_delivery_unconfirmed` | `load` of canonical messages failed | transient (retried) | retried next tick |
| `message_delivery_failed` | `Deliver` error with a live context (conflicting file, unsafe `@recv`) | transient (retried); a conflict persists until the drain | other messages still delivered; retried next tick |

Not faults: a `Poll`/`Deliver` interrupted by the pump's own stop (the drain
retries with a detached context), and per-candidate refusals/duplicates.
Cancelled, timed-out, unknown or spawn-failed attempts still get no drain and
keep their exact cancel/stop/unknown outcome; the fault only reaches the
clean-exit `CompleteAttempt` branch, after `output_collection_failed` and
before `unauthorized_effect_reported`, so it cannot mask either.

Provenance: the four codes are engine constants; no candidate or delivery byte
reaches them, the gate, readiness or authority. A child can only fail its own
attempt (which it can already do by exiting non-zero); a fault never succeeds
or unblocks anything. The faulty attempt's independent peer still succeeds;
the dependent stays pending and is never spawned.

Regression (green after the fix):

- `TestExchangeFailureIsSurfacedOnTheAttempt` (real `RuntimeLauncher`, real
  local store wrapped by `messageFaultStore`, fake runner): the three faulty
  cases record the faulty attempt `failed` with the exact code and exit 0, the
  peer `succeeded`, run `failed`, reviewer spawned 0 times, dependent stage
  unchanged; ingress cases return the injected store error from `Run`. The
  transient case stores the retried `r1`, Codex `succeeded`, reviewer not
  spawned in the stopped run; a second `Run` completes and the reviewer
  receives `r1` (spawned once). Zero test-owned child processes after each case.
- `TestMessagePumpFaults` (fake `Exchange`, no filesystem/process): sticky send
  area refusal; cancelled poll/delivery not a fault; failed commit, read and
  delivery reported then retried (exact commit/delivery counts, misaddressed
  message never delivered); persistent ones still reported at the drain.
- `TestMissingOutputHumanGateAndFailureStopSuccessors`: exit 0 with every
  output plus a fault → `failed`, run failed, nothing ready.

Correction validation (same host and toolchain as above):

| Command | Result |
| --- | --- |
| `go test ./internal/stageexecution -run TestExchangeFailureIsSurfacedOnTheAttempt -count=1` on `defbac4` production code | FAIL as expected (red, table above) |
| `go test ./internal/stageexecution -run TestExchange -count=1 -v` after the fix | PASS, exit 0 |
| `go test ./internal/stageexecution -run 'TestExchange\|TestRuntimeExchange\|TestMessagePump' -count=40 -timeout=900s` | PASS, exit 0, 142.7s (after the test-ordering fix below) |
| `go test ./internal/stageexecution ./internal/workflow -count=1` | PASS, exit 0 (includes S2.1 scheduler/fence/cancel and S1 regressions) |
| `go test ./... -count=1 -timeout=30m` | PASS, exit 0, 32 packages ok (cmd/lingo 157.3s, local 78.5s, workflow 69.4s, install 54.1s, stageexecution 24.4s); run before the test-only ordering fix, production code identical |
| `go build ./...` / `go vet ./...` / `git diff --check` | PASS, exit 0 |
| `go run ./scripts/check-architecture.go domain` / `application` | PASS, exit 0; 12 / 46 files |
| `go fmt ./internal/stageexecution ./internal/workflow` | applied (whitespace in the new test only) |
| `gofmt -l`, `./scripts/check-go-quality.sh format\|staticcheck`, `validate-repository.sh`, sensitive-file/ADR/automation/bootstrap scripts | AUTHORIZATION_REQUIRED (runner permission policy, as before); evidenced only by exact-SHA PR CI |
| `-race`, `gitleaks` | NOT_RUN (no `gcc`; unavailable), as before |

Retained unsuccessful observation: the first 20× repetition failed 1/20 in
`conflicting delivery` (Claude `succeeded`). Cause: test ordering, not the
fix. When Codex's `q1` was accepted before the Claude stub planted its file,
the pump delivered `q1` successfully and the stub then overwrote its own
received copy, so there really was no delivery failure. The test now makes
Codex send only after the plant; the 40× run above is after that change. No
assertion was weakened and production code did not change.

No live Codex/Claude inference ran; no vendor CLI was invoked.

## technical_decisions[]

| ID | Question → choice | Alternatives / trade-off | Risk / rollback |
| --- | --- | --- | --- |
| TD-S22-01 | Child transport → files in launcher-issued per-attempt areas, polled by the engine | Re-enable the CLI publish leaf with env identity (child can forge env); socket/FIFO transport (new abstraction) | Polling latency ≤ `CancelPoll`; revert the two commits |
| TD-S22-02 | Message commits → through the S2.1 principal queue under the runner fence | Lease-free `Engine.Ingest` from workers (second writer path) | Principal handles more commits; `Engine.Ingest` stays for out-of-run S1.6 use |
| TD-S22-03 | Refused/duplicate messages → never set the run error | Treat as worker error (one bad child message would halt independent work) | None beyond a child's wasted message |
| TD-S22-04 | Add `Runtime` to in-memory `IngressContext` | Rely on attempt/agent only | Not persisted; no codec change |
| TD-S22-05 | Refuse answers to ended askers | Store undeliverable answers | Stricter than S1.6; exact replays stay idempotent |
| TD-S22-06 | Delivery is pull-per-recipient from canonical state | Principal-side routing table across workers | Each pump reads the store each tick (lease-free read, like the cancel watcher) |
| TD-S22-07 | Retire inbox/ingest/reader now, keep format-1 codec/store | Delete the whole `coordination` package (breaks S1.8 cutover inventory and `executiongraph` acceptance verifier) | Remaining duplicate representations go to S2.4/S3.4 |

| TD-S22-08 | Exchange fault → fail the attempt (`AttemptResult.Fault`) | `MarkUnknown` (effects are observed, so unknown would be dishonest and block retry); fake non-zero exit (falsifies the observed code); run-level error only (canonical attempt stays `succeeded`) | One in-memory field; revert the correction commit |
| TD-S22-09 | Transient vs persistent → retry every fault each tick, decide at the final drain; only send-area refusal is sticky | Stop on first fault (loses retriable work); retry the drain N times (latency, no evidence it is needed) | A drain-time transient fault fails the attempt; explicit retry recovers |
| TD-S22-10 | Keep the principal's stop-scheduling on any store error | Exempt message commits (hides a correlated store failure) | A transient store blip stops the run; a re-run resumes with nothing lost |

All are reversible local choices inside Plan TD-03 and ADR-0037 §6.

## deferred_findings[] (DEFERRED_NOT_IMPLEMENTED)

- DF-S22-01: `runtimeadapter/packet.go` `TaskPacket`/`PacketContext` (old
  inbox protocol text) has no production consumer; candidate for S2.4.
- DF-S22-02: Spec 007 historical implementation tables link to deleted files
  (as they already do for earlier-retired files); historical text preserved.
  The docs-offline/link checks are evidenced by CI only.
- DF-S22-03: the pump re-reads the whole send area each tick (≤256 entries);
  acceptable at current bounds, measure before raising them.
- DF-S22-04: same-UID children can still write into a peer's area if they learn
  its path (inherent filesystem limit already noted for the old inbox). Sender
  attribution follows the area, so a forged write is attributed to the peer
  attempt, never elevated; OS confinement remains an ADR-0028 limit.
- Inherited deferred findings from S2.1/S1 remain as recorded there.

## Remaining work and next gate

S2.3 (isolated worktree integration) and S2.4 (L/E duplicate representation,
CLI/message docs reconciliation) remain unimplemented. Next gate:
**INDEPENDENT_RE_REVIEW_S2_2** of the S22-CR-01 correction (fresh, not
self-reported), then S2.3 only once that checkpoint is confirmed.
Human pre-merge review, merge, release and Product Accepted remain separate
human gates. No background continuation exists; checkpoint
`AWAITING_EXECUTION_REPORT`.
