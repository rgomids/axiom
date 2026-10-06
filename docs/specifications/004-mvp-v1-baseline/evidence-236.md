# Issue #236 — CI/CD Slice 3 prepared upgrade journeys

Scope: Research #154, #227 / #228, #234 / #235, Specification 004
FR-075 / AC-56 and ADR-0019. Acceptance wiring and envelope binding remain
Slice 4. No release publication or authority change.

## Byte identity and execution

Local native macOS arm64 runs consumed one complete clean prepared set built
once at `9aa63e4316d96303cfbdcc28466043b3b2d15e43`, with test tag
`v9999.0.0-prepared.1`. This is a synthetic regression candidate, not a release.
The harness checkout and candidate source revision are separate facts in Evidence.
The caller pinned `SHA256SUMS` SHA-256
`0b9fa39066f9918ba7162bccc8cbb17a665dda2919a67cb7db1a302a6750c48c`.

The original set is read through anchored descriptors with no symlink traversal.
Its closed five-file snapshot is verified with the existing release verifier
against a clean private local checkout of the exact source revision; no rebuild
occurs. All four archives, manifests, metadata, source-owned installer/skills,
executable format and embedded full VCS provenance are checked before executing
any candidate bytes. Structural-only verification reports `structural_pass`,
which cannot satisfy publication's literal `result=pass` requirement. The
selected private materialization then reports exact version/revision provenance
inside an isolated home and runs the shared upgrade journeys.

Before Evidence emission, every snapshot digest and every extracted file,
including the executable and installer, is compared to the pinned archives.
Prepared `subject.artifacts` binds the complete four-row set; provider artifact
id/digest remain `null`. Prepared historical-source extraction also snapshots
and validates checksums and archive paths before materialization, so traversal
cannot overwrite the prepared candidate. The rebuilt path stays supported.

## Local executable Evidence

| Check | Exit / result |
|---|---|
| `python3 scripts/test-gate-evidence.py` | 0; 103 schema/emitter/capture/harness tests |
| `python3 scripts/test-prepared-upgrade-candidate.py` | 0; 42 identity/filesystem/substitution tests |
| `./scripts/validate-repository.sh .` | 0; includes both suites and registry validation |
| `./scripts/test-release-archives.sh` | 0 |
| `./scripts/test-release-pipeline.sh` | 0 |
| `./scripts/test-release-flow.sh` | 0 |
| `go test ./...` | 0 |
| `go test -race ./...` | 0 |
| `go vet ./...` | 0 |
| `bash -n scripts/test-upgrade-journeys.sh scripts/verify-release-artifacts.sh` | 0 |
| `git diff --check`, sensitive-file checks, Gitleaks | 0; no leaks |

Real journeys used published v0.5.0 and v0.1.1 plus a POC binary built from
`v0.1.0-poc.1`. The prepared pass completed 3 journeys / 48 passing steps,
exit 0; the rebuilt regression completed the same journeys, exit 0. Passing
the candidate itself as N produced exit 1 and valid prepared failure Evidence,
with unknown category retained as `null`, no implicit retry and no masking.
The prepared set, both previous release directories and frozen fixture tree
were hashed before and after both runs: byte-identical.

Retained bounded outputs:

- [Prepared pass](../../../scripts/testdata/gate-evidence/pass-prepared.json)
- [Prepared input-defect failure](../../../scripts/testdata/gate-evidence/fail-prepared-input-defect.json)

These are observed local runs, not release acceptance. Linux/macOS CI runs the
rebuilt and prepared regression on the same freshly built set; final-head remote
results are recorded in the dedicated PR. Linux arm64 native and Windows proxy
journeys are not claimed by this Slice.

## Independent refutation

Independent review tested snapshot substitution, forged binding, archive links,
traversal, duplicate paths and file/ancestor collisions. A rerun subprocess that
lost the prepared binding was fixed to stay in-process, with a regression proving
a cached earlier installer cannot replace the verified installer. Version probe
HOME isolation and prepared historical-source extraction were tightened. No
Blocker/Major remained after independent follow-up review.

ADR-0019 and Research #154 decisions are preserved; no new ADR is needed.
FR-075, contributor instructions and automation registry are reconciled.
Issue/PR review and merge remain human gates; this Evidence authorizes no release.
