# S2.1 — Parallel ready tasks on the canonical engine

Attempt: `RECONCILE-S1-THREADS-TO-S2-01`. This Evidence records a bounded S2.1
checkpoint on the existing AXM-42 / PR #366 branch. It does not claim completion
of S2.2–S2.4, S3, human acceptance, merge, release or actual installed-state cutover.

## Authority and S1 gate

The current maintainer request expressly authorizes S2.1–S2.4 from the accepted
[Tasks](tasks.md), conditionally after S1 technical reconciliation. It extends
the earlier S1-only implementation authority without changing product ACs,
non-goals or [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md).

Preflight: clean worktree on `spec/axm-42-minimum-viable-workflow`; local, fetched
origin and PR HEAD all `1bf0b6c4edda5fb820949166741db035dddeda60`;
base/merge-base `440e44a68638e08e901fd3f87f5b15cd395a6365`. Process inspection
showed this executor and its own tool host, with no other overlapping executor.

[S1 reconciliation receipt](https://linear.app/rgomids/issue/AXM-42#comment-8bf68cab)
was published and read back exactly, including its event JSON and both markers.
All eight original DMH threads received one source/test/fix/review reply and
were resolved; a new API read verified eight resolved threads, two comments
each. DMH-06 Option A authority was read directly in the
[decision record](https://linear.app/rgomids/issue/AXM-42#comment-f5447be3).

Observed at that SHA:

- [CI success](https://github.com/rgomids/axiom/actions/runs/41454586146) and
  [Delivery metadata success](https://github.com/rgomids/axiom/actions/runs/41454586140).
- Targeted Go DMH tests across `cmd/lingo`, `local`, `stageexecution`,
  `runtimeapplication` and `workflowdefinition`: exit 0. Exact command is in the
  reconciliation receipt. No vendor CLI was invoked.
- Existing [independent targeted review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5737621372)
  inspected every correction hunk and reported analytical conformance. It is
  native COMMENT, not GitHub approval or human acceptance.

The receipt consolidates prior deferred findings as inherited evidence, not
tests re-executed here: D-01/02/04/05, DF-IR-01, DF-07-1/2/3, DF-IF-02/03,
DF-R-01/02, O-1, N-1/N-2, DF-DMH06-INDEX-TEXT and validator PATH risk. D-03
and fixed DMH findings remain closed. Vendor/Darwin qualification and real
cutover remain unperformed. Optional improvements do not become S2 requirements.

## Implementation and verification

The single `Engine.Run` now prepares eligible attempts serially, then runs their
existing observed lifecycle concurrently. Worker state mutations return through
a private queue to the principal loop; workers do not call the durable writer.
The loop reloads canonical state before recomputing eligibility and drains all
in-flight workers on refusal or cancellation so terminal observations are retained.
It neither creates child Executions nor changes the format-4 codec or namespace.

`Ready` now checks failed/cancelled peers explicitly: another active peer can keep
the aggregate's derived status `running`, but cannot hide the failed task and
permit further dispatch. Already-running independent tasks retain their results.
The production Runtime launcher already scopes output paths by attempt; the
Launcher interface now documents concurrent Build/Collect calls for distinct
attempts. Mutable inputs are detached through the existing canonical codec.

### Local technical decisions

| ID | Choice and reason | Alternative / trade-off |
| --- | --- | --- |
| TD-S21-01 | Principal-owned mutation queue, serial prepare and concurrent observed lifecycle reuse the S1 engine and existing store CAS/fence. | Direct concurrent store writers retain CAS but violate the accepted principal ownership; a second scheduler/store is unnecessary. The private queue adds coordination but no persisted owner. |
| TD-S21-02 | Recompute readiness after each accepted fact rather than waiting for an entire batch. | Batch barriers are smaller but delay eligible work behind an unrelated slow task. The barrier fixture proves C starts after A while B remains active. |
| TD-S21-03 | Drain already-running attempts after a dispatch refusal; stop scheduling new work without manufacturing operator cancellation. | Returning immediately would strand callbacks/process facts. The trade-off is waiting for each already-authorized attempt's bounded timeout. |
| TD-S21-04 | Keep failure-aware eligibility separate from the run's observable `running` status. | Reporting the whole run terminal while a peer remains active would conceal real running work. |

All are reversible engineering choices within Plan TD-01/TD-03. No material
product/architecture decision or new dependency was introduced.

### Deterministic proof

- `TestEngineParallelReadyTasksStreamAndCommitWithoutLoss`: a barrier requires
  A and B active together; C starts after A before B finishes; the dependent
  stage follows. All four attempts retain process and terminal facts, the
  observed durable-writer concurrency is exactly one, and returned state matches
  the canonical final store revision.
- `TestEngineParallelStopsDependentWork`: human gate, persisted cancel, operator
  interruption, unknown process result, missing required output and stale-fence
  cases refuse successors while preserving observed independent terminal facts.
- `TestEngineParallelDependencyDefinitionsRefuseBeforeSpawn`: the canonical
  constructor rejects missing/cyclic agent and stage dependencies. This proves
  definition refusal, not a counter on an unconnected fake runner.
- `TestExecutionParallelFailureStopsReadyWhilePeerStillRunning`: a failed peer
  blocks readiness and successor preparation while another peer remains active;
  that active peer's later success is retained.

Implementer observations (ordinary Go tests, no vendor CLI):

```sh
go test ./internal/stageexecution -run TestEngineParallel -count=10 -timeout=90s
go test ./internal/workflow -run TestExecutionParallelFailureStopsReadyWhilePeerStillRunning -count=1
```

Both exit 0; the ten repetitions completed in 55.981s. Main integration checks
and final independent-review outcome are recorded below after completion.

### Validation environment

The first Go1.26 build/test/architecture attempts exited 1 before source checks:
ambient `GOROOT` still referenced Go1.27.3 (`compile: version "go1.27.3" does not
match go tool version "go1.26.0"`). The correction is per-command environment,
not repository/toolchain policy:

```sh
export PATH=/home/grokbot/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-arm64/bin:/usr/bin:/bin
export GOROOT=/home/grokbot/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-arm64
export GOTOOLCHAIN=local
```

This PATH excludes real Codex/Claude CLIs and their user-local directories;
offline evaluator tests may create only their own temporary stubs. No actual
test-battery or live Runtime invocation is authorized or claimed. The maintainer's
current explicit battery pins take precedence over older repository prose.

### Independent engineering review

A fresh-context, read-only delegated reviewer, separate from the implementer,
inspected all S2.1 production changes, new tests, documentation, accepted
Plan/Tasks and ADR-0037. It inherited this Orchestrator's model/effort; no model
diversity is claimed. Verdict: **analytical conformance, no blocking finding or
required correction**, subject to the broader integration checks. This is not
human acceptance or native GitHub approval.

The reviewer independently ran, with the Go1.26 environment above:

```sh
go test ./internal/stageexecution ./internal/workflow \
  -run 'TestEngineParallel|TestExecutionParallelFailure' -count=1
```

Exit 0: stageexecution 15.375s, workflow 0.719s. The stale-fence fixture uses a
deterministic rejecting store wrapper, not an actual takeover; missing/cyclic
dependency cases exercise canonical construction, not malicious persisted bytes.
Existing strict decoding and lease tests retain those lower boundaries. Ordinary
tests and source inspection do not prove race freedom.

### Integration checks

Checks use the explicit Go1.26 environment above. This is the pre-commit snapshot
on 2026-11-02, observing this worktree delta from `1bf0b6c`; the final PR/Linear
receipt binds the committed SHA, completed checks and exact-SHA remote CI. No
success is inferred for a pending command.

| Command / check | Observed result |
| --- | --- |
| `go build ./...` | PASS, exit 0 |
| `go vet ./...` | PASS, exit 0 |
| `./scripts/check-go-quality.sh all` with `STATICCHECK=/home/grokbot/go/bin/staticcheck` | PASS, exit 0; format, module tidiness and pinned Staticcheck v0.8.1 |
| `go run scripts/check-architecture.go domain` / `application` | PASS, exit 0; 12 / 46 source/test files |
| `python3 scripts/check-adr-governance.py .` | PASS, exit 0; 38 ADRs |
| `python3 scripts/check-automation-registry.py .` | PASS, exit 0; 98 governed surfaces |
| `./scripts/check-claude-bootstrap.sh --skill-adapters .` | PASS, exit 0 |
| `./scripts/check-sensitive-files.sh .` / `git diff --check` | PASS, exit 0 |
| `go test ./... -count=1 -timeout=15m` | Pending; no result claimed |
| `./scripts/validate-repository.sh .` | PASS, exit 0; synthetic evaluators only, real vendors excluded from PATH |
| `./scripts/test-docs-offline.sh` | Pending; no result claimed |
| `go test -race ./internal/stageexecution ./internal/workflow -run TestEngineParallel -count=1 -timeout=2m` | NOT_RUN: exit 2, `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` |
| Same race command with `CGO_ENABLED=1` | NOT_RUN, build exit 1: `cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in $PATH`; no race test result |
| `gitleaks` | NOT_RUN: `command -v gitleaks` exit 1; unavailable |

Retained unsuccessful observation: the implementer's final
`go test ./internal/stageexecution ./internal/workflow -count=1 -timeout=5m`
exited 1. Stageexecution passed in 35.678s; workflow timed out at 300.208s in
historical `TestDeliveryReviewFindingsRequireExactHumanWaiver` (delivery fixture /
portable configuration validation, no scheduler stack). Concurrent broad checks
were active; the cause is not proven. This timeout is not reported as PASS and
is not by itself evidence of a scheduler regression. The parent full-suite result
is tracked separately above.

## Remaining work and next gate

This bounded checkpoint implements S2.1 only. S2.2 cross-Runtime routing,
S2.3 isolated worktree integration and S2.4 coordination/gate retirement remain
authorized but unimplemented. This change supplies neither public child
publication nor an installed workspace isolation/integration operation. All
parallel process validation here uses deterministic stubs; no live vendor, real
repository cutover, remote test PR or Provider mutation ran.

The next development gate is S2.2 on the same branch/PR, followed by S2.3/S2.4
and independent review. No ceremonial transition approval is needed under the
current request. There is no durable continuation callback in this runtime;
publication of the final completion receipt does not claim future background
execution. Human pre-merge review, merge, release, Product Accepted and AXM-12
unblocking remain separate restricted gates.
