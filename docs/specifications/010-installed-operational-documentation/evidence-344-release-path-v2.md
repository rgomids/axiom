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
