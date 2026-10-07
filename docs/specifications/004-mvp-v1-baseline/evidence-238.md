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
                 verify-release-acceptance.py --automated-only for this run id
manual transition (maintainer, on the downloaded prepared set)
  linux-arm64 (native) / windows-amd64 (FR-072 bounded proxy)
                 -> <dir>/<row>/gate-evidence.json
release.sh status --manual-acceptance <dir>
  pack <dir> once -> unpack next to the run's automated Evidence
  verify-prepared-release.sh (run succeeded, set re-verified at the revision)
  publish-release.sh: verify-release-acceptance.py (all four rows) -> envelope binds
  acceptance.<row>=<Evidence SHA-256> for every row -> preview_digest
release.sh publish --manual-acceptance <dir> -> dispatch manual_acceptance=<packed line>
publish-release.yml
  run's automated Evidence + unpacked manual_acceptance -> publish-release.sh
  verifies all four rows again -> envelope == human-authorized digest -> effects
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
`<row>/gate-evidence.json` for every release row, without following links. It
refuses anything other than valid `axiom-gate-evidence/v1` with:

- gate `candidate-acceptance`;
- `pass`, `completed`, exit 0;
- the row;
- a `prepared` subject with exactly the tag, version, full revision,
  SHA256SUMS digest and archive digests of the closed prepared set;
- distinct attempts across rows.

The rest depends on the row:

| Row | Producer | Environment | Further checks |
|---|---|---|---|
| `linux-amd64`, `macos-27-arm64` | `accept` job of the prepared run | `native`, the row's OS family and architecture | suite `upgrade-journeys` and its upgrade matrix; `run.ci` of this repository's `release-artifacts.yml` on `main`, job `accept`, run id equal to the prepared run |
| `linux-arm64` | maintainer (manual transition) | `native`, Linux arm64 | suite `upgrade-journeys` and its upgrade matrix |
| `windows-amd64` | maintainer (manual transition) | `bounded_proxy`, Windows amd64 (FR-072) | suite `windows-bounded-proxy`; one journey whose only steps are the passing `artifact-identity`, `provenance`, `direct-cli`, `installer-server-refusal` and the FR-072 `not_applicable` `fresh-install`, `owned-upgrade`, `reinstall`; no installer outcome |

A manual row's `run.ci` is null or a run of this repository other than the
prepared run's `accept` job.

**Residual:** no tool emits the Windows proxy document yet. That emitter
belongs to the Windows proxy Slice. Until it exists, a maintainer records the
document from their own hosted Windows Server proxy run, and no stable release
can be published without it. Linux arm64 Evidence comes from the existing
harness (`--row linux-arm64`) on a native host.

### Manual Evidence transport (CR-001)

The workflow dispatch is the only channel into `publish-release.yml`. A
maintainer workstation cannot add a workflow artifact to the completed
prepared run. The manual documents therefore travel as the
`manual_acceptance` dispatch input:

- `pack` writes one line: `axiom-manual-acceptance-v1:` followed by the base64
  of a deterministic gzip of length-framed rows. It is at most 60,000
  characters, under the 65,535-character dispatch payload.
- `release.sh` packs the given directory once. The envelope is computed from
  the bytes that line carries, and `publish` dispatches that same line.
- In the workflow, the input reaches the shell only through an environment
  variable. The revision's verifier unpacks it next to the run's automated
  Evidence:
  - strict prefix, base64 and framing;
  - bounded inflation;
  - exactly the two manual rows, in order;
  - exclusive creation, without following links.
- The line is never trusted. The envelope binds each document's SHA-256, and
  publication re-verifies all four rows before any effect. A missing, changed
  or foreign manual document changes or refuses the envelope.

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

- **`scripts/test-verify-release-acceptance.py`** (48 tests) covers:
  - the happy path and determinism;
  - missing, empty and absent Evidence;
  - failing, malformed, schema-invalid and oversized Evidence;
  - another row, tag/version, revision, run, repository, workflow, job or
    branch;
  - a non-native host;
  - divergent SHA256SUMS or artifact digests;
  - substituted and incomplete prepared sets;
  - rebuilt and `merge-regression` Evidence;
  - a reduced upgrade matrix and rows sharing one attempt;
  - links;
  - strict identity arguments;
  - input immutability;
  - for each manual row (CR-001), Evidence that is:
    - missing (one or both rows);
    - failing or malformed;
    - for another row, tag/version, revision, SHA256SUMS, artifact digest
      or prepared candidate;
    - rebuilt-candidate Evidence;
    - from the wrong environment: Windows not `bounded_proxy`, Linux arm64
      not `native`;
    - a Windows proxy reporting an installer outcome, or outside its closed
      FR-072 step set (the reviewed attack: relabelled native
      upgrade-journeys Evidence), or without its passing observations;
    - a manual row claiming the prepared run's `accept` job or another
      repository;
    - Linux arm64 with a reduced upgrade matrix;
    - reusing another row's attempt;
    - reached through links;
  - automated failures are reported before pending manual Evidence;
  - `--automated-only` never prints an envelope binding;
  - the transport:
    - byte-exact, deterministic round trip;
    - the dispatch size bound;
    - malformed, reordered, foreign, truncated, oversized and
      inflation-bomb bundles;
    - no reuse or following of existing entries;
    - a forged line still refused by `verify`.
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

  For the manual rows (CR-001), envelope and authorized publication refuse
  each of the following, with zero publication effects:
  - for each of Linux arm64 and the Windows proxy, Evidence that is missing,
    failing or malformed, or that is for another tag/version, revision,
    SHA256SUMS, artifact digest or prepared candidate;
  - Linux arm64 Evidence in the Windows row;
  - Windows Evidence that is not the bounded proxy, or that claims a client
    install;
  - Linux arm64 Evidence that is not native;
  - automated Evidence alone, even with `acceptance_manual_transition`.

  The envelope binds all four rows. At the `release.sh` and workflow
  boundary:
  - a stable `prepare` stops at `accept_manually`, with no envelope;
  - `publish` without the manual Evidence, or with replaced, removed or
    other-candidate manual Evidence, is refused before dispatch;
  - the dispatch carries exactly the packed line of the envelope;
  - a discovered run whose given manual Evidence is refused is `blocked`,
    never re-prepared;
  - the publish job refuses a missing, other or other-candidate bundle with
    zero effects;
  - all four valid documents publish under the authorized digest;
  - a release-candidate dispatch carries no manual Evidence.
- **`scripts/test-release-pipeline.sh`** pins the workflow contract:
  - stable-only blocking rows equal to the verifier's;
  - one prepared-mode harness call with the full pin;
  - no rebuild, other run or overwrite;
  - the inventory check;
  - Evidence retained apart from logs;
  - the binding job for its own run id, with `--automated-only`;
  - no publication path that accepts automated Evidence alone.
- **`scripts/test-gate-evidence.py` and `scripts/test-prepared-upgrade-candidate.py`**
  check that the `candidate-acceptance` gate exists only for a verified
  prepared subject and matches the recorded command.
