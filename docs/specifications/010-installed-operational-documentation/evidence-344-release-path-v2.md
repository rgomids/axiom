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
