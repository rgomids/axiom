# ADR-0036 — Native qualification at main gates the release start

## Status

Proposed for human review on 2026-10-25. The maintainer chose this option in
the 2026-10-25 release pipeline analysis ("Gate no start"). Merging this ADR
with its implementation accepts it; Release PR review, publication authority
and `release` environment approval remain separately gated.

## Context

[ADR-0029](0029-minimal-pr-gate-native-qualification.md) keeps the PR gate to
`PR Checks` and `delivery-metadata` and runs the native suite (Linux/macOS
`go test -race`, release contracts, upgrade journeys) nightly, on manual
dispatch and in release preparation, where a failure blocks publication. It
accepted that "release preparation takes longer and can fail late" and set a
revisit trigger: "Nightly failures repeatedly reach release preparation".

That trigger fired. Between 2026-10-19 and 2026-10-25 the nightly was red on
most days while releases kept starting:

- v0.20.0: preparation failed in `upgrade-journeys` (#332, #333);
- v0.21.0: the nightly had been red since #334 (2026-10-20); preparation failed
  twice in the native suite, and the version was superseded (ADR-0035, #338,
  #339, #340, #341, #347);
- v0.22.0: the 2026-10-25 nightly failed with the same 404 in
  `upgrade-journeys (macos)` six hours before preparation failed with it (#352).

Each case started a release from a `main` commit that the native suite had
not passed, so the failure appeared only after the Release PR was merged,
when the release commit is fixed and only a hotfix, another Release PR or a
supersession can move on.

## Decision

1. `release.sh status` offers `start_release` and `refresh_release_pr` only
   when the newest `nightly.yml` run on `main` at exactly the current `main`
   SHA (scheduled or dispatched) succeeded. It reports `main_native` and
   `main_native_run`.
2. When no such run exists, `next_action=qualify_main`. `release.sh qualify`
   dispatches `nightly.yml` on `main`, waits and prints `status` again. It
   publishes nothing and accepts no option.
3. While that run is in flight, `next_action=await_run`. When it failed,
   `next_action=blocked` with the run URL: fix `main`, or a human re-runs the
   run when the failure is not reproducible.
4. The PR gate, the ruleset and the preparation gate of ADR-0029 are unchanged.
   Preparation still runs the native suite at the release commit. This ADR
   adds a gate and supersedes no decision of ADR-0029.

## Alternatives considered

| Alternative | Trade-offs |
| --- | --- |
| A — gate the start on a green native suite at `main` (chosen) | A red or stale native signal stops the release before the Release PR; PRs stay fast; starting a release may wait for one native run (about 30 minutes). |
| B — A plus a required native PR job (ADR-0029 alternative C) | Catches regressions per PR; every PR waits for macOS and race suites and the remote ruleset changes. |
| C — A plus a non-required native run on PRs touching release paths | Earlier signal for release changes; reintroduces path filters that ADR-0029 removed and can be ignored. |
| D — keep ADR-0029 as is | No waiting at start; failures keep reaching preparation after the Release PR merge. |

## Consequences

### Positive

The native suite of the commit a release starts from has passed before any
Release PR exists, so preparation fails only on what differs between that
commit and the release commit (the Release Please changelog commit) or on
hosted-runner drift.

### Negative / trade-offs

Starting a release can wait for a native run. A `main` that moves after
qualification must be qualified again before a start or a refresh. A
regression still merges and is detected nightly or at the next start, as
ADR-0029 accepted.

## Revisit when

Release preparations keep failing after a qualified start, or the native run
time makes the start wait materially longer than its value.
