# AXM-37 — three v2 gzip framing failures, Python r1

## Scope and authority

Same Implement, sole OPEN/Draft PR #335. Entry branch
`docs/axm-37-offline-operational-documentation`, local/remote/PR HEAD
`b8acfe50ec9658f7cde55667b066ae841d8e9b45`, clean workspace.
AGENTS.md, security policy, current Tasks and historical framing Evidence were
read before editing. Recent Linear events through the 2026-10-21 continuation
were read (50-comment page covers this attempt and its predecessor; older
pagination was not needed for this bounded delta). Rafael explicitly authorized
Python 3 with stdlib zlib, the three refusals, requirements/tests/Evidence/Tasks,
commit and push. No newer contradictory event was observed.
Bounded process/cwd/worktree inspection found this executor on AXM-37 and a
separate AXM-36 executor; no conflicting local AXM-37 execution was identified.
This is not a distributed lease or proof about other hosts. No new Execution,
PR, work item, delegation or tool installation occurred.

## Control and compatibility

The bootstrap keeps the existing checksum and private staging, bounded system
gzip drain, physical tar EOF walk, tar validation, path/link/mode checks and
v1/v2 dispatch. Only the detected v2 path additionally runs `python3 -I -S`,
using stdlib `zlib.decompressobj(31)` against the original verified staged bytes.
It stops at the first gzip EOF and refuses either unused input in the current
chunk or any further byte in the source. This detects split tar members, empty
suffix members and empty prefix members without magic-byte guessing. CRC and
length remain checked by gzip and zlib. Input/output chunks are at most 64 KiB;
expansion above the existing v2 301989888-byte stream budget is refused.
No decompressed Python output or persistent installation state is written.
Existing binary/docs/skills identity, symlink/path/lock and tar EOF controls
remain intact. Original compressed bytes/checksum remain delegation inputs.

Python >= 3.6 with zlib is a conditional bootstrap prerequisite for v2 only;
`docs/installation.md` documents it and the `install_error` diagnostic.
Missing command, rejected interpreter flags, import failure or decoder error
fails closed before tar extraction, candidate execution or persistent writes.
`-I -S` excludes environment/current-directory and site-package imports.
No pip/external package is used. Default remains v1; its historical multistream
split/empty-suffix/empty-prefix acceptance is explicitly tested; separate
prerequisite controls prove v1 verification works without Python.
The approved prerequisite decision supersedes the previous pending boundary
in [round 2](evidence-framing-seven-fails-r2.md), not its historical results.

## Reproduction and matrix

Before the control change, the three newly enabled v2 regressions individually
FAILed: checksum-valid shell verify-only accepted while Go refused. After the
change, the same tests PASS. Each mutation recalculates SHA256SUMS; expected Go
acceptance is asserted. Normal entrypoint negatives must leave no sentinel
marker, bin directory or receipt directory. Valid controls reach a harmless
candidate sentinel that writes a temporary marker and exits 91; verify-only
never executes it or writes persistent destinations.

| Case | Entry shell / Go | Current shell / Go | Result |
| --- | --- | --- | --- |
| v2 tar split at byte 17 between members | accept / refuse | refuse / refuse | PASS |
| v2 additional empty member | accept / refuse | refuse / refuse | PASS |
| v2 prefixed empty member | accept / refuse | refuse / refuse | PASS |
| v1/v2 complete tar followed by another compressed complete tar (2 cases) | refuse / refuse | refuse / refuse | PASS |
| v1/v2 nonzero tail after tar EOF (2 cases) | refuse / refuse | refuse / refuse | PASS |
| v1 split/empty suffix/empty prefix (3 controls) | accept / accept | accept / accept | PASS |

The unchanged inert `framing-preflight-r1.patch` was extracted through a Go
overlay and rerun: PASS, 33 cases; its historical v2 symlink case stays SKIP.
The active integrity suite now has 34 cases. Four prerequisite cases prove
missing/incompatible Python refuses v2 in verify-only and normal entrypoints,
while v1 verification does not require Python. Host: development macOS arm64;
this is not native Linux qualification or Q6 proof.

## Validation receipts

Code/requirements/test commit:
`59beacdc9b534643ad6d1ff8a8cbb0b51646e79a`.

- PASS: all seven repaired cases; all 34 active integrity subcases (`-count=1`),
  four missing/incompatible-Python subcases, sentinel and destination assertions.
- PASS: original overlay experiment, 33 subcases; one historical SKIP retained.
- PASS: `GOPROXY=off GOTOOLCHAIN=local go test ./internal/install -count=1`.
- PASS: `GOPROXY=off GOTOOLCHAIN=local go vet ./...` and `go build ./...`.
- PASS: `bash -n scripts/install-release.sh`; Go quality format check.
- PASS: `scripts/test-release-pipeline.sh` from committed repair HEAD above,
  default-v1 and explicit-v2 rows, exact-byte smoke and no-publication controls.
- PASS: `scripts/test-install-posix-facade.sh` (14 cases),
  `python3 scripts/test-prepared-artifact-smoke.py` (13 tests).
- PASS: `scripts/test-docs-offline.sh`, synthetic development journey only;
  its own native/network/source-denial qualification remains PARTIAL/NOT_RUN.
- PASS: repository validator including ADR governance, gitleaks (no leaks),
  staged sensitive-file check before the repair commit, scoped diff review/check.
- FAIL: isolated `GOPROXY=off GOTOOLCHAIN=local go test
  ./internal/executiongraph -run
  '^TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation$' -count=3
  -timeout=30s`; two of three iterations failed, separately from install.
- PASS: `scripts/test-install-bootstrap.sh`, 78 cases, fake network, fixtures
  from committed repair HEAD above; no live release installation.
- PASS: `GOPROXY=off GOTOOLCHAIN=local go test ./...` on the repair tree
  (cache where applicable). This does not erase the isolated executiongraph FAIL.
- UNVERIFIED: final-head remote CI until push and exact-head read-back. The
  same-attempt Linear completion records the exact SHA/run URLs/observed status;
  no previous-head CI is reused.
- NOT_RUN: native Linux/host matrix, real Runtime sessions and Q6.

## Remaining gates

Entry-head CI was FAILURE: Linux system gzip accepted compressed zero padding
in both v1/v2, while Go refused (`zero_trailing`). This is distinct from the
three authorized cases. The v2 framing check also refuses any compressed tail;
v1 behavior is deliberately unchanged. Final-head CI is recorded separately,
without claiming entry-head CI as current PASS. A remaining v1/Linux failure
requires separate scope/authority; it is not silently fixed or waived here.

The isolated executiongraph regression remains separate: `-count=3` produced
two failures losing the process result (ExitCode -1, empty output,
Stopped/OutcomeUnknown). No executiongraph file changed; cause unestablished.
A green full suite cannot erase this isolated FAIL.

T13–T15/Q6 stay deferred to AXM-12; T16 remains partial. No default activation
or preparation, host/session provisioning, real Runtime inference, credentials,
costs, merge, release, acceptance, Done, status or blocker mutation.
Next gate: incremental review of the exact pushed HEAD; no self-acceptance.
