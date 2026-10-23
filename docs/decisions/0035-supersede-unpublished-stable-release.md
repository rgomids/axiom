# ADR-0035 — Explicit supersession of an unpublished stable release

## Status

Proposed for human review on 2026-10-23. Merging this ADR with its
implementation accepts it; Release PR review, publication authority and
`release` environment approval remain separately gated.

## Context

ADR-0011 binds a stable version to the merged Release PR commit that records
it in `.release-please-manifest.json`, and `release-plan.sh` refuses any new
release while `main` records a version without its tag. Preparation builds
and tests that exact commit; ADR-0010 corrections replace only control
scripts, never the source under test.

v0.21.0 (release commit `3c99cf1`) cannot pass preparation: its own source
fails the native suite on hosted runners (macOS 15 refuses the documentation
generation rename; two Linux test defects). The fixes (#339) are on `main`
after the release commit. The flow therefore had no exit: v0.21.0 can never be
prepared, and no later version can start before it is published. Publishing it
anyway would ship an installer known to fail on macOS 15.

## Decision

A stable release whose release commit is on `main` but that is never published
can be explicitly superseded by the next version.

- `.github/release-supersessions.txt` is the reviewed, append-only record:
  `<MAJOR.MINOR.PATCH> <full release commit SHA> <reason>`, one row per
  version. It changes only through a normal reviewed PR.
- `release-please-config.json` pins `last-release-sha` to that release commit,
  so Release Please computes the next version from the recorded manifest
  version and the commits after it (it finds no GitHub Release for the
  superseded tag and would otherwise fall back to `bootstrap-sha`).
- `release-plan.sh` accepts an untagged `v<manifest>` only when the record names
  it at the commit that recorded it and the pin equals that commit. The range
  and bump start at the superseded release commit, matching Release Please;
  `previous_tag` becomes the published release before it, whose GitHub Release
  must be consistent. A superseded range that delivers Issues is refused: it is
  never released, so it cannot close them. A `last-release-sha` pin without a
  matching supersession, or a supersession row for a published version, is
  refused, so the pin must be removed by a normal PR once the next release is
  recorded.
- `release.sh status` no longer treats a merged Release PR of a superseded
  release as awaiting publication (`superseded_release_prs`), and
  `status --tag` of a superseded version is `state=superseded`,
  `next_action=blocked`: it is never prepared or published. Release Please
  aborts while any merged Release PR is still `autorelease: pending`, so
  `release.sh start` withdraws that label from each superseded Release PR
  before dispatching it (`effect=release_pr_unlabeled ... reason=superseded`);
  the reviewed record, not the label, stays the provenance.
- `release-notes.sh` appends, after the version's own CHANGELOG section, the
  sections of each directly preceding superseded release under
  `### Includes unpublished vX.Y.Z (superseded)`, so the published notes cover
  every change the release ships.

## Alternatives considered

- **Publish the superseded release from its commit with the native suite
  waived.** Ships a known installation failure and weakens the preparation
  gate for every release. Rejected.
- **Remove the `autorelease: pending` label by hand.** Changes a release record
  outside the flow and does not unblock the plan, which reads the manifest.
  Rejected; the withdrawal happens only as a recorded `start` effect, for a
  Release PR the reviewed record supersedes.
- **Revert the release commit on `main`.** Moves the manifest boundary away from
  the published v0.20.0 tag, which the plan refuses, and rewrites intent with
  a synthetic commit. Rejected.
- **`Release-As` footer.** Forces a version but Release Please still collects
  every commit since `bootstrap-sha`, producing an incorrect changelog.
  Rejected.

## Consequences

- The superseded tag never exists. Its CHANGELOG section stays as recorded; the
  next release's compare link starts at a tag that does not exist (cosmetic).
- After the next release is recorded, a normal PR must remove
  `last-release-sha` before any later release can start; the plan enforces it.
- Only one superseded release directly before the planned one is modelled;
  chains and superseded ranges that deliver Issues fail closed.
- What becomes harder to change: the record is part of the release contract;
  rows are never edited or removed, and widening the mechanism needs a new ADR.
