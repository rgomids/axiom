# ADR-0011 — Command-driven release start

## Status

Proposed for human review on 2026-10-02. Merging this ADR with its
implementation accepts it; publication, environment approval and repository
settings remain separately gated.

## Context

Release Please ran on every push to `main` and kept one Release PR open. A
merge therefore also started versioning: it created or updated the Release PR,
dispatched its checks and moved the next release forward without any
maintainer intent. The delivery metadata of the commits in that range was only
resolved at preparation, after the Release PR was merged. v0.3.0 showed the
cost: an undeclared commit (#157) was found after the release commit existed,
and the release could only be prepared through the pinned recovery of
ADR-0010.

## Decision

Merges integrate code. `$axiom-release` starts releases.

- `release-please.yml` has no `push` trigger. It runs only on
  `workflow_dispatch` from `main`, with the planned version and the exact
  `main` SHA as inputs. It re-runs the release plan on that SHA and refuses
  before Release Please runs if the SHA, the plan or the planned version
  differ. After Release Please, it dispatches the required checks only for a
  Release PR whose title records the planned version.
- `scripts/release-plan.sh` is the deterministic, Git-only preflight of a new
  release. For the first-parent commits since the last release commit on
  `main` it requires a Conventional Commit subject and declared (or reviewed,
  committed) delivery metadata; the previous release must be published at its
  release commit; the planned tag must not exist and must be newer than every
  stable tag. It computes the SemVer bump with the documented Release Please
  semantics. One inconsistent commit stops the release before any Release PR,
  artifact, tag or release exists.
- `scripts/release.sh start` runs that preflight plus the remote facts
  (required CI on `main`, no in-flight run, no GitHub Release for the planned
  tag), then dispatches `release-please.yml`, waits, and reports the Release PR
  whose version must equal the plan. It never approves or merges it.
- `scripts/release.sh status` is a state machine
  (`state=` + `next_action=`) that discovers the release in progress from
  GitHub and Git: no release, Release PR open (stale when `main` moved or the
  plan changed), Release PR merged, preparation or publication in flight,
  awaiting publication authority, published. It discovers the newest verified
  prepared run of the release, so the maintainer never supplies SHAs, run ids
  or intermediate digests; `publish` only needs the `preview_digest` the human
  authorized.
- Release Please stays the versioning, `CHANGELOG.md`, manifest and Release PR
  mechanism. The prepare, publication envelope, authority, publish, verify and
  delivery phases are unchanged.

## Alternatives

- Keep the push trigger and only add a preflight before preparation: still
  couples merges to versioning and still discovers bad history after the
  Release PR.
- Replace Release Please with an in-repository versioner: removes a
  dependency but rewrites changelog and manifest handling with no evidence
  of a defect in Release Please.
- Pass the planned version to Release Please (`release-as`): the pinned action
  has no such input, and a config edit per release would be a write to `main`.
  Release Please computes the version; the plan must agree, or the start
  fails closed.

## Consequences

- A Release PR exists only between a maintainer's start and its merge. Its
  version can go stale when `main` moves; `status` reports
  `refresh_release_pr` and `start` re-validates and refreshes it.
- The release plan duplicates the documented Release Please bump table. A
  divergence (for example a commit-override footer) is caught by the version
  check in the workflow and in `start`; nothing is merged or published on a
  mismatch.
- Delivery metadata gaps are fixed before the release by a normal reviewed PR
  that appends a correction to `.github/delivery-corrections.txt`; the release
  commit then contains it. ADR-0010 recovery remains for historical or
  partially published releases (v0.3.0) and is not part of the normal path.
- Delivery tracking is unchanged: merge moves Issues to Awaiting Release, the
  Release PR merge records Target Release, stable publication releases and
  closes them. A release is not human acceptance.

Related: [ADR-0010](0010-pinned-release-corrections.md),
[Plan §13](../specifications/004-mvp-v1-baseline/plan.md),
[CONTRIBUTING.md — Release flow](../../CONTRIBUTING.md#release-flow).
