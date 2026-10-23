# GitHub #344 — stable release path prepared for generation 2

**PARTIAL.** Draft change on branch `fix/344-stable-docs-v2-release-path`
(base `921507c27c5b`). Baseline B:
[decision](https://github.com/rgomids/axiom/issues/344#issuecomment-6368343359).
Cause confirmed: `build-release-archives.sh` defaults to `format=1` and
`release-artifacts.yml` never passed `--format 2`, so v0.21.1 shipped v1
(no descriptor/docs). v0.21.1 is immutable and untouched.

## Change (explicit selection; script defaults stay historical v1)

- `release-artifacts.yml`: build, verify and both exact-byte smokes use
  `--format 2`; macOS row is `macos-arm64`; smoke artifact renamed accordingly.
- `native-suite.yml`, `native-acceptance-qualification.yml`,
  `native-artifact-acceptance.yml`, `accept-native-artifact.py` (`--format`):
  candidate archives are format 2; previous releases stay v1.
- `test-upgrade-journeys.sh`: `archive_in` accepts exactly `macos-27-arm64`
  (historical N) or `macos-arm64` (v2 candidate); no URL fallback.
- `verify-prepared-release.sh`, `release.sh verify --download`: pass
  `--format 2` only when the checksum inventory names `macos-arm64`;
  correction-source (older) verifiers still receive no new flag.
- `docs/installation.md`: states v1 releases carry no verified docs.
- Tests updated: release pipeline workflow assertions, native-acceptance fixture.

## Evidence (host: macOS arm64, development; HEAD 921507c + working tree)

PASS: `test-release-pipeline.sh` (v1 and v2 sets, 13 smoke tests),
`test-native-acceptance.py` (4), `test-prepared-artifact-smoke.py` (13),
`test-release-flow.sh`, `test-install-bootstrap.sh`, `test-docs-offline.sh`
(synthetic development only), repository validator, ADR governance, automation
registry. v2 linux-arm64 archive: 49 entries (acceptance bound 256).
NOT_RUN: `test-upgrade-journeys.sh` (needs published releases/network;
syntax-checked only), Linux/macOS hosted CI until the PR runs, native T13-T16,
Q6, AC-08, v0.1.0 floor, real Runtime inference. Nothing here is native
qualification, merge, release or default promotion in scripts.

## Review correction (CR-345-01..03, base 559033d9d138)

- CR-345-01: `accept-native-artifact.py --format 2` adds scenario
  `installed-docs-verified`: fresh installed binary, cwd outside the checkout,
  `docs inspect` must report `verified/ok`, binding to the artifact binary and
  revision, existing non-symlink document paths under the installation, and
  `man.state=available`. Report gains `installedDocs` (network/Q6 `NOT_RUN`).
  Refusal scenarios and bounds unchanged; format 1 unchanged.
- CR-345-02: `test-upgrade-journeys.sh archive_in` requires exactly one host-row
  archive per directory, listed exactly once in its `SHA256SUMS`; zero, mixed
  v1/v2, duplicates or unlisted fail closed.
- CR-345-03: `release-archive-format.sh` (sourced, registered) names the
  generation for `verify-prepared-release.sh` and `release.sh verify
  --download`; unreadable, mixed, duplicated or row-less inventories are
  refused, and the strict verifier is never relaxed.
- PASS (host macOS arm64): test-release-pipeline.sh (new format/archive_in
  cases), test-native-acceptance.py (5), test-release-flow.sh,
  test-prepared-artifact-smoke.py, test-docs-offline.sh, validate-repository.sh
  (registry), ADR governance. NOT_RUN: executing the new native scenario on
  Linux, test-upgrade-journeys.sh end to end, hosted CI, native T13-T16, Q6.

## Review correction (CR-345-04, base 26c3e80d5a72)

- CR-345-04: `installed_docs()` now requires `docs.paths.manRoot` and
  `docs.man.page` to pass `confined()` (new narrow helper in
  `accept-native-artifact.py`): absolute, no `..`, strictly inside the
  installed documentation root (page also inside the man root), no symlink on
  any component from the root down, and the expected kind (dir/file). The
  inspector's own strict checks are unchanged and remain independent.
- Tests: `test-native-acceptance.py::test_confined_man_paths` executes the real
  helper on a temp fixture: valid root/page pass; external man root/page,
  relative, traversal, wrong kind, root itself, symlinked man root (ancestor of
  the page) and symlinked page fail. Static scenario assertion also requires
  `confined(`.
- PASS (host macOS arm64): `python3 scripts/test-native-acceptance.py` (6 tests).
  Other suites: see the Issue #344 receipt for exact results.
- NOT_RUN: executing the native scenario on Linux, hosted CI for the new HEAD
  until it runs, test-upgrade-journeys.sh end to end, native T13-T16, Q6, AC-08,
  actual release. No native qualification is claimed.

## Native qualification 01 (HEAD 2ef9b7ae4103, macOS arm64 host only)

Host: macOS 27.0.1 arm64, go1.26.1, 2026-10-23 America/Sao_Paulo. Artifacts are
local development builds (`build-release-archives.sh --only macos-arm64`,
versions `0.21.2-qual.1` and `0.21.3-qual.1`), not published assets. Roots are
isolated temp dirs with `env -i`, cwd with spaces; no network used for docs.

| case | host/runtime | artifact SHA | method/command | observed result | status | evidence link | limitation |
|---|---|---|---|---|---|---|---|
| T14 v2 install + `docs inspect` | macos-arm64 / none | archive `ad46bdd0e0cf…`, binary `7df4648c4307…` | `install-release.sh` into root A; `axiom --json docs inspect` from other cwd | `docs.state=verified`, 29 files, generation `13de7501a060…`, manRoot/page confined, `man axiom` renders matching version/commit | PASS (development) | this file | no network/source denial observed |
| T14 offline suite | macos-arm64 | same | `test-docs-offline.sh --archive … --checksums …` | corpus 26 files links closed; 7 refusals (extra-output, swapped, stale metadata, tamper, modified binary, v1 installer rejects v2) | PASS (development) | this file | script states network/source denial NOT_RUN |
| A/B independence | macos-arm64 | A `13de…`, B `8ad2…` | two roots, install B after A | A generation unchanged after B operations | PASS | this file | one host |
| v1 install | macos-arm64 | binary `0d84b20f7b83…` | install v1 archive into root B | installed; `docs.state=unknown` (no docs) | PASS | this file | |
| v1→v2 same version | macos-arm64 | v2 0.21.2 over v1 0.21.2 | `install-release.sh` | refused atomically `divergent_equivalent_version`, binary unchanged | PASS (refusal) | this file | |
| v1→v2 upgrade | macos-arm64 | v2 0.21.3 `937bbe73b952…` | install over v1 | `upgraded`; receipt version/sha updated; docs verified, new generation `8ad2322ed1f4…` | PASS | this file | fresh helper, not published assets |
| reinstall / downgrade | macos-arm64 | same / 0.21.2 | repeat install | `unchanged`; downgrade `downgrade_refused`, state intact | PASS | this file | |
| tampered installed doc | macos-arm64 | — | append byte to installed man page | `docs.state=invalid` | PASS | this file | inspector not relaxed |
| `test-native-acceptance.py` | macos-arm64 | — | run | see Issue receipt | PASS if exit 0 | this file | |
| T13 Linux amd64/arm64 | none available | — | — | no host | NOT_RUN | — | no authorized hosts |
| `test-upgrade-journeys.sh` E2E, v0.1.0 floor | — | — | — | needs published releases/network | NOT_RUN | — | authority required |
| T15/E15 Q6, AC-08, V9 | — | — | — | real Codex/Claude inference + OS observation not authorized | NOT_RUN | — | AUTHORIZATION_REQUIRED |
| symlink/foreign/crossed-docs negatives, PATH/man switch | macos-arm64 | — | — | not exercised beyond tamper | NOT_RUN | — | |
| T16 aggregate | — | — | — | mandatory rows missing | NOT_RUN | — | no overall PASS |

## Native qualification 02 (HEAD 08012db6d126, macOS arm64 host only)

Host: macOS 27.0.1 arm64, 2026-10-23 America/Sao_Paulo. Authority: Issue #344
[grant](https://github.com/rgomids/axiom/issues/344#issuecomment-6375188966).
All roots are isolated temp dirs (`env -i`, cwd with spaces); no source-checkout
or maintainer-home path was an allowed documentary source. Linux was not
touched; T13 Linux amd64/arm64 stay **NOT_RUN**. Nothing here is merge, release
or default promotion, and full release readiness is **not established**.

### A. Published-release upgrade journeys (`test-upgrade-journeys.sh`, end to end)

Published inputs (macOS rows; downloaded with `gh release download`, then checked
against `SHA256SUMS` and the GitHub API asset `digest`; both matched; no
attestation exists for any of them, so provenance is checksum + API digest only):

| release | tag commit | archive sha256 | SHA256SUMS sha256 |
|---|---|---|---|
| v0.21.1 (N) | `921507c27c5b` | `4ea306d19591…` | `6d707c19d483…` (+ verifier helper `21300f58fc83…`, API digest matched) |
| v0.15.0 | `37662730c64e` | `3ca48fb132a9…` | `66e3a69aad3c…` |
| v0.1.0 (floor) | `084cc384d4c9` | `f817d8550b58…` | `5b3b2379a595…` |

Candidate: **local** release-mode build, not a published asset
(`build-release-archives.sh --version 0.21.2-qual.2 --format 2 --only macos-arm64`
from clean HEAD `08012db6d126`; archive `bcbf1fe55e14…`, binary `e3b99dad9957…`).

Result: `exit=0`, 37 `result=pass` steps, `failures=0`, journeys
`v1-state-v0.21.1` (v1→v2), `skills-v0.15.0` (incl. PD-3 legacy refusal),
`skills-v0.1.0` (floor), including upgrade, version, state preservation, Runtime
readiness and idempotent rerun. Status: **PASS (macOS arm64, local v2 candidate)**.
Not covered: RecognizedPOC journey (`--poc-binary`; `v0.1.0-poc.1` publishes no
assets, a local build was not substituted) = **NOT_RUN**; Linux rows = NOT_RUN.
Two earlier attempts were input errors, not product defects: v0.21.1 needs its
published verifier helper asset; a `--development` candidate reports version
`development` and fails the version assertion by design.

### B. Q6 / AC-08 / V9 — Codex native sessions (Claude NOT_RUN)

Runtime: codex-cli 0.163.0 (binary `8813367285…`), fresh `exec --ephemeral`
sessions with an isolated `CODEX_HOME` (copied `auth.json` only, web search
disabled), HOME/TMPDIR/cwd under the run root, installed A or B bundle skills.
Control: `sandbox-exec` profile denying `file-read-data` on the maintainer home,
`/private/tmp`, `/Users/Shared`, `/Volumes` except the installation roots and
session dirs; all outbound network denied except a local logging CONNECT proxy
allowing only `chatgpt.com` (inference/auth transport). Documentary retrieval was
not requested or permitted: the proxy denied `ab.chatgpt.com` and
`*.oaiusercontent.com` in every session.

Observers actually available: Unified Log kernel `Sandbox: … deny` events
(unprivileged, per operation/pid/path), the proxy log, a 0.3 s `ps` sampler.
**Not available:** `fs_usage`/`eslogger` (root required, `sudo` needs a password),
`sandbox-exec (trace)` (produced no output). Consequently **successful reads are
not independently observed**; only denials are. Short-lived children fall between
samples, so PID-tree attribution is partial (denials are also listed unattributed
by time window). No air-gap or zero server-side retrieval claim is made.

| case | runtime | identity | method | observed | status | limitation |
|---|---|---|---|---|---|---|
| Q6-1 identities/versions | codex | A `e63f8fb4d0dc`/`e3b99dad9957`, B `55593c409303`/`ece2efc9bcb9`, same version 0.21.2-qual.2 | install.sh into roots A/B | recorded | PASS | B is a scratch-clone same-version build (extra beacon line), never pushed |
| Q6-2 AC-01, no Runtime, network denied | none | A | `docs inspect` + index link closure, cwd with spaces, `curl` denied | verified/ok, 10 links closed, curl exit 6 | PASS (native macOS) | file reads not observer-confirmed |
| Q6-3/4 effective denial controls | codex+descendants | — | `neg` session: read checkout AGENTS.md, canary, `~/.claude`; curl via proxy to example.com; `curl --noproxy`; python socket | exits 1,1,1,56,6,1; kernel denies for cat, cat, ls, curl, Python; proxy denied example.com | PASS for denial controls | does not prove absence of other reads |
| Q6-4 positive-read observation of installed docs | codex | — | none available | tool transcript shows reads; no OS-level record | **UNVERIFIED** | no root observer |
| Q6-6 A positive | codex | A | `a-pos` | state verified, gen `e63f8fb4d0dc`, beacon NONE, docFile `text/cli.md`, `man -w` in A | PASS (functional only) | Q6-4 gap |
| Q6-6 B via PATH switch | codex | B | `b-switch` | gen `55593c409303`, quotes beacon QUAL-B-4471 from B docs | PASS (functional only) | same |
| Q6-6 man selection differs | codex | A bin, B MANPATH | `man-mismatch` | docs gen A, beacon NONE, `man -w` reports B path | PASS (functional only) | session did not flag mismatch unprompted |
| Q6-6 explicit A/B + broken | codex | six explicit paths | `explicit` | A, B verified; crossed `invalid/inventory_mismatch`, absent `absent/binding_absent`, tampered `invalid/hash_mismatch` all REFUSED without fallback | PASS (functional only) | same |
| Q6-6 symlinked docs root | codex | A with `generations/<g>/docs` → `docs.real` | `explicit` | `docs inspect` returned verified/ok and the session read docs through the symlink | **FAIL** | see deferred finding DF-1 |
| Q6-6 crossed skills | — | — | — | skills carry no docs binding at this HEAD | NOT_RUN | |
| Q6 any row, Claude | claude | — | — | only credential is an expired OAuth token in the maintainer keychain; use forces refresh-token rotation that can invalidate the maintainer login; new enrolment not authorized | **NOT_RUN** | AUTHORIZATION_REQUIRED |
| AC-08 / V9 aggregate | codex+claude | — | — | Codex functional rows pass; observation incomplete; Claude absent | **UNVERIFIED** | |

Inference was bounded to 5 Codex sessions plus 2 smoke runs; real `auth.json`
digest was unchanged afterwards (no refresh). Raw logs, proxy logs and process
samples stay in the local scratch area, not in Git.

### C. T13/T14 macOS-only, repo tests, T16

Local suites on this host, all exit 0: `test-native-acceptance.py`, `test-release-pipeline.sh`, `test-release-flow.sh`, `test-install-bootstrap.sh`, `test-prepared-artifact-smoke.py`, `test-docs-offline.sh` (candidate archive; script still reports PARTIAL for source/network denial), `validate-repository.sh`, and `go test` for operationaldocs, cli, install and axiom-archive-verify. T14 macOS
installed-docs refusals now also cover crossed, absent, tampered (PASS) and
symlinked docs root (FAIL, DF-1). **T16 aggregate: NOT_RUN/blocked** — Linux
T13, Claude Q6, positive-read observation and DF-1 are open.

Deferred finding DF-1: `axiom docs inspect` accepts a generation whose `docs`
directory is a symlink (the check Lstat-guards only the final component of
sub-paths, and `docs` is walked as a non-directory entry). Impact is limited to a
same-user writable tree, but spec symlink refusals and the native helper's
`confined()` both expect refusal. Not fixed here: it changes reviewed product code
and needs a scoped decision plus new review.

## Maintainer decision and handoff (2026-10-23, America/Sao_Paulo)

Decision (maintainer, human): reduce the qualification scope to conclude the
main correction. The final E2E is run by the maintainer, not the agent. No new
Claude credentials, root/`fs_usage`, Linux hosts or further Q6 sessions are
requested for this round. This does not turn any unproven contract criterion
into PASS.

| item | status |
|---|---|
| Baseline B (release path selects format 2 explicitly) | implemented in PR #345 |
| Hosted CI on reviewed code SHA `2ef9b7ae4103` | PASS |
| macOS arm64 local checks and 37-step upgrade E2E (local v2 candidate, not a published release) | PASS (development) |
| **DF-1** `generations/<g>/docs` symlink accepted by `docs inspect` | **OPEN / ACCEPTED_RISK / DEFERRED_TO [#346](https://github.com/rgomids/axiom/issues/346)** |
| Final E2E (manual checklist below) | **OWNER: HUMAN / PENDING** |
| Claude Q6, independent observation of positive reads (Q6-4), AC-08/V9 | NOT_RUN / UNVERIFIED |
| Linux T13 amd64/arm64, RecognizedPOC journey, crossed skills | NOT_RUN |
| T16 aggregate | NOT_RUN (mandatory rows missing) |

DF-1 scope and severity: product code `axiom docs inspect` (Lstat guards only
the final component of sub-paths); impact limited to a same-user writable
installation tree (no privilege boundary crossed). It contradicts the spec's
symlink refusal and the native helper's `confined()` expectation, and the
Q6 symlink row stays **FAIL**. It is not fixed, hidden, suppressed or relaxed
in this PR; the risk acceptance is a human decision by the maintainer, and full
conformity is **not** declared.

### Manual E2E checklist (maintainer)

Use a clean or isolated root (`env -i`, fresh `HOME`/install prefix), cwd outside the checkout.

1. Build or download the v2 candidate (`build-release-archives.sh --format 2 --only <row>`) and install it with `install-release.sh`.
2. `axiom version` shows the expected version/commit.
3. `axiom --json docs inspect` -> `docs.state=verified`, `man.state=available`.
4. `man axiom` renders the matching version.
5. Upgrade over a previous v1 install (higher version) -> `upgraded`, docs verified; same-version v1->v2 is refused atomically; downgrade refused.
6. Functional regression: existing project/Work Item commands still behave.
7. Record result and links in a comment on Issue #344 (PASS/FAIL per step, host, SHAs).

### Pending gates before a v2 promotion

- Manual E2E above (human).
- Qualification rows NOT_RUN above remain open: promotion as "release_qualified" is not supported by current Evidence; the maintainer must either accept that scope explicitly in the release decision or run the rows. Smallest path: record an explicit scope-reduction acceptance on #344 and run the checklist.
- No merge, release publication or Issue closure is performed by the agent.
