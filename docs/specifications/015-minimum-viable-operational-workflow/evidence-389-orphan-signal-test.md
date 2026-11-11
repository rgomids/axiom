# Evidence — #389 orphan signal observer

Date: 2026-11-11. Base `e5b9d10`, Linux arm64, 2 CPUs (shared host).
Scope/acceptance: [amendment and plan](amendment-389-orphan-signal-test.md).

## Root cause

The victim test binary inherits the observer's signal dispositions. A
`SIG_IGN` survives `exec`, and the Go runtime deliberately keeps an inherited
`SIG_IGN` for SIGHUP and SIGINT (only those two). When the suite runs with
SIGHUP ignored (for example under `nohup` or a detached agent/job runner), the
victim ignores the SIGHUP under test, keeps sleeping in `TestLeakVictimHelper`
for 60 seconds, and the observer's fixed 45-second wait fails with
“victim did not exit”. That is the incident's exact signature: SIGHUP subtest
45.76 s, while SIGINT/SIGTERM passed in the same run. Isolated reruns from an
interactive shell have SIGHUP at its default and pass. Load is not the cause;
the full-suite run and the reruns differed in inherited disposition. The
incident's original runner disposition was not captured, so the match is by
mechanism and signature, not by a recorded process status.

No product leak: in every run the owned runner/stub were stopped and the
no-survivor check passed whenever the victim actually exited.

Note (not changed): the product runner's `signal.NotifyContext` handles SIGHUP
even when it was inherited as ignored; that is existing behaviour, out of
scope here.

## Fix (test harness only)

- The victim is launched as `env --default-signal=HUP,INT <test binary> …`
  (GNU coreutils ≥ 8.31; CI runs Ubuntu 24.04 with coreutils 9.4). `env` resets
  SIGHUP/SIGINT to `SIG_DFL` in the child, then execs. The PID, `Setpgid` and
  `Pdeathsig` stay the same, and the observer process never changes a signal
  disposition.
- CR-393-F1 (review of PR #393 at `94b3e43`): the first fix installed a
  handler with `signal.Notify` around `Start` and then called `signal.Reset`.
  `Reset` is process-wide. It removes any other `Notify`/`NotifyContext`
  registration for that signal, for example one left active by an in-process
  CLI test, so it was replaced by the launcher above.
  `TestLeakVictimLaunchKeepsObserverSignals` proves the observer is unchanged:
  with its own SIGHUP/SIGINT handler registered, the kernel `SigIgn`/`SigCgt`
  masks in `/proc/self/status` are identical before and after a failed `Start`
  (missing working directory) and a successful launch. The handler still
  receives SIGHUP and SIGINT afterwards. The test also shows that a child under
  an inherited ignore (`trap "" HUP INT`, as `nohup` does for SIGHUP) keeps the
  HUP/INT ignore bits with a plain exec and has them cleared through the
  launcher.
- The exit wait is bounded by `min(2m, t.Deadline() - 10s)` instead of a fixed
  45 s, leaving room for the independent no-survivor check, and its failure
  message reports whether the victim is still alive and which owned processes
  remain.
- Identity proofs, expected exit codes and the no-survivor assertion are
  unchanged.

## Reproduction (deterministic)

`nohup go test -count=1 -run '^TestKilledTestBinaryLeavesNoOwnedProcess$/^SIGHUP$' -v ./cmd/lingo/`

| Variant | Without nohup | Under nohup (SIGHUP ignored) |
| --- | --- | --- |
| Before fix (wait bound only) | PASS 1/1 (0.96 s) | **FAIL 1/1**: victim exit 1, `victim was not stopped` after 60.75 s (with the original 45 s bound this is “victim did not exit” at ~45 s) |
| After fix, SIGHUP+SIGINT `-count=3` | — | PASS 6/6 (≤ 2.63 s) |
| After fix, SIGHUP `-count=20` | — | **PASS 20/20** (1.70–3.81 s; no owned process left) |
| CR-393-F1 launcher, SIGHUP `-count=20` | — | **PASS 20/20** (0.61–0.84 s; victim exit −1; no owned process left after 182–277 ms) |
| CR-393-F1 launcher, SIGHUP+SIGINT `-count=3` | — | PASS 6/6 |
| CR-393-F1 launcher, all 7 cases + proof test | PASS 8/8 (exits −1×4, 0, 1, 2) | PASS 8/8 (same exits) |

After the CR-393-F1 runs, an independent `pgrep -af 'TestLeakVictimHelper|lingo.test'`
matched only its own shell.

Mutation checks on `TestLeakVictimLaunchKeepsObserverSignals`:

- Dropping `--default-signal` fails the test: the child's `SigIgn` keeps the HUP/INT bits.
- Re-adding the old `Notify` + `Reset` in the launcher kills the test binary
  with `signal: hangup`, because the observer's handler is gone.

An earlier attempt (previous runtime) ran an uncontrolled shared-host load
loop: 9 PASS / 11 FAIL, all 11 failures before signal delivery
(`execution_revision_conflict` in the `startHangingRunner` status poll, the
#371 family), 0 “victim did not exit”. It did not reproduce #389 and is out of
scope here.

## Validation

| Check | Result | Evidence |
| --- | --- | --- |
| `gofmt -l cmd internal scripts` | PASS | empty output |
| `go vet ./...` | PASS | no output |
| `git diff --check` | PASS | no output |
| `./scripts/check-sensitive-files.sh .` | PASS | `sensitive-file checks passed (worktree)` |
| Targeted SIGHUP/SIGINT under nohup | PASS | table above |
| `go test ./... -count=1` | FAIL (unrelated) | all packages `ok` except `cmd/lingo`, which hit Go's default 10 min package timeout on the loaded host (stack in `TestExecutableDeclarationAuthoringWithoutMigration`, waiting on a child `go build`) |
| `go test ./cmd/lingo/ -count=1 -timeout 30m` | PASS | `ok … 182.283s` |
| `./scripts/validate-repository.sh .` | FAIL (pre-existing, host) | `test_model_advisor_eval.py` `test_unavailable_runtime_exits_nonzero`: `0 != 1`. Its “no claude” PATH keeps `dirname(go)`; on this host `go` and the real `claude` share `~/.local/bin`, so the real binary is found. Not touched by this change; not rerun. |

CR-393-F1 rerun on the launcher change:

| Check | Result | Evidence |
| --- | --- | --- |
| `gofmt -l cmd internal scripts`, `go vet ./...`, `git diff --check` | PASS | no output |
| `./scripts/check-sensitive-files.sh .` | PASS | `sensitive-file checks passed (worktree)` |
| `go test ./... -count=1 -timeout 30m` | PASS | every package `ok`; `cmd/lingo` 346.959 s |
| `env PATH=/usr/local/go/bin:/usr/bin:/bin ./scripts/validate-repository.sh .` | PASS | `repository bootstrap validation passed`; vendor CLIs kept off PATH, so the host issue above is avoided, not fixed |

Real model inference is not authorized for this card; the validator failure
above means the real `claude` binary may have been reached by that test. It is
reported as a separate finding, not fixed here.
