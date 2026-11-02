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
a lost fence or store failure stops routing and is recorded by the principal.

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
**INDEPENDENT_REVIEW_S2_2**, then S2.3 only once that checkpoint is confirmed.
Human pre-merge review, merge, release and Product Accepted remain separate
human gates. No background continuation exists; checkpoint
`AWAITING_EXECUTION_REPORT`.
