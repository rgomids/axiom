# Issue #238 — CI/CD Slice 4 release-candidate acceptance on prepared bytes

Scope: Research #154, #227 / #228, #234 / #235, #236 / #237, Specification
004 FR-068–FR-075 / AC-50–AC-56 and ADR-0019. No release was prepared or
published, and the release authority (ADR-0011, ADR-0012) is unchanged.

## Release boundary

```text
release-artifacts.yml (stable tag, one run)
  prepare        build once -> verify -> notes -> artifact axiom-release-<tag>
                 outputs tag, full revision, SHA256SUMS SHA-256, channel
  accept-linux-amd64 / accept-macos-arm64   (native hosts, blocking)
                 download that artifact -> test-upgrade-journeys.sh --prepared-set
                 --candidate-acceptance with the pinned identity -> inventory unchanged
                 -> artifact axiom-acceptance-<tag>-<row> (gate-evidence.json)
  acceptance-evidence
                 verify-release-acceptance.py for this run id
release.sh status / publish-release.yml
  verify-prepared-release.sh (run succeeded, set re-verified at the revision)
  publish-release.sh: verify-release-acceptance.py -> envelope binds
  acceptance.<row>=<Evidence SHA-256> -> envelope == human-authorized digest -> effects
```

- **Placement:** ADR-0019 Alternative D. Acceptance runs inside the
  preparation run, so the existing "prepared run succeeded" gate already
  covers it, and the envelope additionally binds each Evidence digest.
- **Harness:** the workflow's own `main` revision (`github.sha`) supplies
  the harness and checks; the subject is only the retained set.
- **Channel:** blocking for stable tags (FR-069, AC-51). A release candidate
  runs no acceptance, and its envelope states `acceptance=not_required_rc`.
- **Published state:** a release that already reads back published has no
  bytes left to publish. Its envelope states `acceptance=already_published`.
- **Retry:** an Evidence artifact name is used once per run. A rerun cannot
  replace recorded Evidence; retry policy belongs to Slice 11.

`verify-release-acceptance.py` reads each document once, from exactly
`<row>/gate-evidence.json` for every blocking row, without following links. It
refuses anything other than valid `axiom-gate-evidence/v1` with:

- gate `candidate-acceptance` and suite `upgrade-journeys`;
- `pass`, `completed`, exit 0;
- the row, on a native host of that row's OS family and architecture;
- a `prepared` subject with exactly the tag, version, full revision,
  SHA256SUMS digest and archive digests of the closed prepared set;
- `run.ci` of this repository's `release-artifacts.yml` on `main`, job
  `accept`, run id equal to the prepared run.

## Local executable Evidence (macOS arm64, 2026-10-06)

A clean clone at `9ee2609d6d241c33141fa0bb59b8e05cee7a5781` built one set
with the test tag `v0.6.0`. This is a synthetic candidate, not a release.

| Fact | Value |
|---|---|
| SHA256SUMS SHA-256 | `1b9828717ae28ada7b8af972a1f450303bdf6b3fbff164a02565eeed2b7caa12` |
| Executed row archive (`macos-27-arm64`) SHA-256 | `6b51b24624fd25f7afeca80ae6b194ed5af940fd52b258e3212773cac0a25db5` |
| Upgrade sources | published v0.5.0 and v0.1.1 (public assets), RecognizedPOC build |
| Acceptance run (as the `accept` job runs it) | exit 0, 3 journeys, 48/48 steps, gate `candidate-acceptance`, subject `prepared` |
| Evidence SHA-256 (local, `run.ci = null`) | `742c39818c483931e4ad14bec1df5f2b55552974958ea9e1902b89a369d4ae71` |
| Prepared directory inventory before/after | byte-identical |
| Verifier on the local Evidence | refused: not produced by the `accept` job of the prepared run |
| Same run with a simulated GitHub Actions identity (`run_id=1`, job `accept`) | every per-row check passes; Evidence SHA-256 `65460bf519cc0849c7b46d2288b4228b63cd76a0f5a8c1119023d98fc6504f00` (local simulation only, never retained as release Evidence) |

Linux amd64 cannot run natively on this host. Its first prepared-byte
acceptance Evidence comes from the first stable `release-artifacts.yml` run
after merge, together with CI-produced macOS Evidence.

## Deterministic negative coverage

- **`scripts/test-verify-release-acceptance.py`** (25 tests) covers:
  - the happy path and determinism;
  - missing, empty and absent Evidence;
  - failing, malformed, schema-invalid and oversized Evidence;
  - another row, tag/version, revision, run, repository, workflow, job or
    branch;
  - a non-native host;
  - divergent SHA256SUMS or artifact digests;
  - substituted and incomplete prepared sets;
  - rebuilt and `merge-regression` Evidence;
  - rows sharing one attempt;
  - links;
  - strict identity arguments;
  - input immutability.
- **`scripts/test-release-flow.sh`** proves `publish-release.sh` refusals in
  both envelope and authorized-publication mode, with zero publication
  effects:
  - missing row Evidence, an absent Evidence directory, a missing argument;
  - failing, malformed or rebuilt Evidence;
  - Evidence of another row, tag/version, revision or prepared run;
  - divergent SHA256SUMS or artifact digests;
  - prepared bytes substituted after acceptance;
  - an incomplete candidate.

  It also proves that:
  - Evidence bytes other than the authorized ones change the digest and stale
    authority is refused;
  - the accepted set publishes exactly its bytes;
  - inputs stay byte-identical;
  - release candidates need no acceptance;
  - `release.sh` refuses a stable prepared run lacking Evidence or carrying
    another run's Evidence;
  - the workflow downloads only the prepared run's Evidence before the
    publication step and never reruns acceptance.
- **`scripts/test-release-pipeline.sh`** pins the workflow contract:
  - stable-only blocking rows equal to the verifier's;
  - one prepared-mode harness call with the full pin;
  - no rebuild, other run or overwrite;
  - the inventory check;
  - Evidence retained apart from logs;
  - the binding job for its own run id.
- **`scripts/test-gate-evidence.py` and `scripts/test-prepared-upgrade-candidate.py`**
  check that the `candidate-acceptance` gate exists only for a verified
  prepared subject and matches the recorded command.
