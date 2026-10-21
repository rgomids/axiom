# AXM-37 — v2 default promotion preflight r1

**PARTIAL. Default remains v1. Activation is not ready for authorization.**
Same Implement, branch and Draft [PR #335](https://github.com/rgomids/axiom/pull/335).
Entry local/remote SHA: `6042aa1b7ec5be19166fc24bbedad30a06787190`, clean tree.
The local AXM-37 process belongs to this executor; the other identified active
worktree executor was AXM-36. Linear event log was read before editing, including
the latest `v2-default-promotion-preflight-r1` dispatch. No competing AXM-37
executor was identified in that bounded local/provider inspection; this is not
a global lease claim. No new Execution or delegation occurred.

Authority: current bounded preflight prompt and
[explicit patch/commit/push authority](https://linear.app/rgomids/issue/AXM-37#comment-e16ecc09)
cover compatibility fixes while preserving default v1. No later default
activation authority was found in the event log. Entry CI and review 9 remain
historical. No raw provider content is imported.

## Demonstrated difference and repair

Source of tested repair: `e44d3cb4c6160439492bbaee57d929bb67436558`.
Host: development macOS arm64, Apple gzip 487.0.1, bsdtar 3.5.3 /
libarchive 3.7.4. No Linux equivalence is inferred from this host.

The system tar stops at the first tar EOF without necessarily draining gzip.
At entry, checksum-valid CRC corruption, compressed trailing garbage, duplicate
file/directory entries and an over-limit LICENSE reached the shell's candidate
version probe despite Go refusing them. Safe shell candidates wrote only a
temporary marker and exited 91; no real candidate install was delegated.
`--verify-only` accepted the same invalid packaging without executing it.

The shell now drains gzip into a bounded private tar before listing/extraction,
checks gzip's exit status, captures tar listing failure, rejects normalized
duplicate entries and enforces compressed/checksum/stream/entry/file bounds.
The existing system `gzip` executable is now an explicit bootstrap prerequisite;
no tool was installed, no library or generic parser was added. Installed and
source prerequisites were reconciled. Legacy gzip multistream remains accepted.
Staged original compressed bytes and their checksum still reach the Go installer;
they are not rebuilt or substituted with the decompressed staging file.

`TestShellArchiveIntegrityBeforeExecution` covers 20 v1/v2 cases: valid controls,
CRC, truncation, compressed garbage, duplicate directory/file, path, manifest,
metadata and per-file bound. Verify-only never executes or publishes; normal
entrypoint refuses negative cases before the marker/destinations. Valid controls
prove the marker is executable. Fixtures select native metadata on Linux too;
only macOS local execution was performed here.

## Remaining technical gate — actual FAIL

The broader adversarial experiment remains **FAIL**, not exhaustive equivalence:

| Checksum-valid mutation | Go | Shell after repair |
| --- | --- | --- |
| v1 single tar across gzip members, including empty members | accept | accept |
| v2 single-member positive control | accept | accept |
| compressed garbage or zero padding after gzip; CRC; truncation | refuse | refuse on tested host |
| duplicate entries and over-limit LICENSE | refuse | refuse |
| complete tar followed by another tar, v1 and v2 | refuse | accepts; normal probe reached |
| nonzero bytes after tar EOF inside gzip, v1 and v2 | refuse | accepts; normal probe reached |
| v2 tar split across members; extra empty member; empty prefix member | refuse | accepts; normal probe reached |

The full 34-case experiment has seven failing subcases and one explicitly
skipped v2 link mutation (v1 link and existing Go/shell suites cover link refusal).
These are newly demonstrated pre-existing validation gaps; no Go loader change
or relaxed single-member contract was made. `gzip -l`/verbose output is not a
sound member-count oracle: an empty prefixed member leaves the final size/CRC
unchanged. A byte-magic heuristic or canonical recompression was not adopted.

[Pending regression patch](framing-preflight-r1.patch) preserves the experiment
without adding an intentionally failing test to required CI. It is inert
Evidence, not an activation patch or a green test. Reproduce without editing the
checkout: extract added lines after its `@@` header into a temporary Go test;
create a Go overlay mapping the absolute
`internal/install/archive_shell_preflight_test.go` path to that temporary file;
run `go test -overlay /absolute/overlay.json ./internal/install -run '^TestPreflightShellGoFraming$' -v -count=1`.
All mutations recompute SHA256SUMS; the failure is not an outer checksum bypass.

## Release-consumer audit and activation scope

No default/CI activation files were edited. Causal step 1 still fails, so an
activation-ready patch and its approval request are **NOT_PREPARED**. The audit
identified these coordinated requirements for the next bounded patch:

- `build-release-archives.sh`, `verify-release-artifacts.sh`,
  `smoke-prepared-artifact.py`: explicit historical format 1, default prepared
  format 2, three closed rows and matching metadata/token checks.
- `release-artifacts.yml`: PREPARE, reverify and both exact-byte smokes must
  agree on format, including `macos-arm64`; retained bytes must remain the
  publication inputs. No rebuild after approval.
- `verify-prepared-release.sh` and `release.sh verify --download`: currently
  invoke a verifier without format selection. They need coherent v2 selection
  and explicit historical-v1 behavior, including correction-source verification.
- `native-suite.yml`, `native-acceptance-qualification.yml` and
  `test-upgrade-journeys.sh`: separate historical release token selection from
  v2 candidate token selection; do not rename v1 historical assets or add URL
  fallback. Existing Linux tokens remain unchanged.
- `test-release-pipeline.sh`, `test-install-bootstrap.sh`,
  `test-prepared-artifact-smoke.py`, `test-s7-native.sh` and historical archive
  fixtures need explicit v1 where history is under test, plus default-v2
  prepared-path assertions. The older `test-release-archives.sh` also contains
  historical inventory assumptions; none were silently converted here.

`install.sh` already selects a closed checksum inventory for either generation;
`install-release.sh`/LoadCandidate dispatch by verified metadata.
`publish-release.sh` compares exact SHA256SUMS, evidence digests, inventory and
asset bytes; no format-specific rebuild was found there. This bounded audit
does not prove the complete prepared/publication v2 path.

## Validation and limits

- PASS: repaired 20-case shell/Go regression; affected Go packages;
  `test-install-posix-facade.sh`; 13 `test-prepared-artifact-smoke.py` tests;
  `test-docs-offline.sh` (26 outputs and refusal controls, development only).
- PASS: `test-release-pipeline.sh` at clean committed source `e44d3cb`, full
  default-v1 and explicit-v2 three-row sets, checksums and no-publication guards.
- Initial bootstrap PASS at entry SHA (78 cases); clean repaired-source rerun,
  final full-suite status and exact-head CI are reported in the same-attempt
  GitHub/Linear receipt. No earlier SHA's PASS proves a later head.
- PASS: Go formatting, repository validator, ADR governance (35 ADRs),
  `git diff --check`, staged sensitive-file check and gitleaks before repair
  commit. Later Evidence security/validation checks are reported with delivery.
- FAIL: broad framing experiment above. It is not masked by passing narrow
  regression, bootstrap, release pipeline or CI.
- FAIL: isolated `GOPROXY=off GOTOOLCHAIN=local go test ./internal/executiongraph -run '^TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation$' -count=3 -timeout=30s`.
  Lost process result (`ExitCode=-1`, empty output, Stopped/OutcomeUnknown)
  reproduced. This module is unchanged; cause remains unproven.

No descriptor/binding/skills identity, lock policy, migration floor, recovery,
downgrade, GC, governance, ADR, status or dependency relationship changed.
Native T13–T15/Q6 remain deferred to AXM-12 by the recorded human decision;
NOT_RUN/PARTIAL/UNVERIFIED are retained, T16 is partial. No native sessions,
observers, hosts, tool installation, merge, tag, release or acceptance occurred.

Next: incremental review of this actual repair head, then finish the demonstrated
framing refusals within the accepted host contract (or obtain a specific human
decision if that contract must change). Only after that technical gate and a
tested coordinated activation diff should one grouped default-activation
authorization be requested. A pending authority gate does not waive these failures.
