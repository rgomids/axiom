# Release Flow v2 — command-driven release start

Recorded: 2026-10-02. Scope: maintainer release lifecycle (ADR-0011, Plan §13
"Release start reconciliation", Task RF2-T01). No release, tag, environment
approval, merge, Issue closure or repository-setting change was performed.

## Change

```text
before: PR -> merge -> push to main -> Release Please (automatic) -> Release PR kept open
        -> merge -> prepare -> (late metadata failure -> ADR-0010 recovery) -> ...
after:  PR -> merge -> main -> CI -> done
        $axiom-release -> preflight (release-plan.sh) -> SemVer plan -> dispatch Release Please
        -> Release PR -> human merge -> $axiom-release -> prepare -> envelope
        -> human authorization of preview_digest -> publish -> verify -> delivery
```

## Deterministic validation (local, offline fakes)

| Command | Result |
|---|---|
| `./scripts/test-release-flow.sh` | PASS (461 checks, including section 6 "Release Flow v2") |
| `python3 scripts/test-release-pr-checks.py` | OK, 14 tests |
| `./scripts/test-delivery.sh` | PASS |
| `python3 scripts/test-release-corrections.py` | OK, 12 tests |
| `./scripts/test-release-pipeline.sh` | PASS |
| `./scripts/validate-repository.sh .` | PASS |
| `./scripts/check-sensitive-files.sh` | PASS |

Required scenarios and their checks in `scripts/test-release-flow.sh`:

| Scenario | Check label (abridged) |
|---|---|
| merge does not create a Release PR | `merges never start a release: the Release PR workflow runs on explicit dispatch only` |
| `$axiom-release` → preflight → Release PR | `start dispatches Release Please once with the planned version and exact main`; `start stops at the Release PR with its URL, planned version and tag` |
| commit without metadata → no Release PR/artifact/tag/release | `invalid metadata: start fails in preflight`; `invalid metadata: no Release PR, artifact, tag or release` |
| feat / fix / breaking (0.x and ≥1.0), Release-As | `plan: fix -> patch`, `plan: feat -> minor`, `plan: breaking feat! -> minor while 0.x`, `plan from 1.x: breaking -> major`, `plan: Release-As forces the version`, `plan: hidden types alone are not releasable` |
| open Release PR → no stable preparation | `open Release PR: prepare is refused`; `open Release PR: no preparation was dispatched` |
| merged Release PR → exact release commit | `merged Release PR: the release commit is discovered without operator SHAs`; `prepare dispatches the exact release commit and nothing else` |
| prepare creates no tag or release | `prepare creates no tag or release` |
| no authorization → zero effects; correct digest → publication | `no authorization: zero effects`; `refused publications had zero effects`; `the authorized digest alone dispatches the discovered exact envelope once` |
| drift → old authority refused | `drift: a changed prepared set invalidates the authorized digest`; `stale authority: an old digest is refused`; `the digest binds the prepared run` |
| verify: asset / tag / revision differ | `verify fails when a published asset differs`; `verify fails when the published assets differ from the prepared set`; `verify fails when the tag points elsewhere`; `verify fails when the release revision differs` |
| idempotency in intermediate states | `start again with an open Release PR is refused (idempotent)`; `repeated start never duplicates the dispatch`; `an in-flight Release Please run is reported, never dispatched again`; `a publication waiting for environment approval is reported, never redispatched`; `prepare again is refused once a verified set exists` |
| stale Release PR | `a releasable commit merged after the Release PR requires a refresh`; `a Release PR behind main by validated hidden commits stays reviewable (branch update only)`; `a refresh is not offered on a red main`; `a Release PR version different from the plan is never offered for review`; `an inconsistent commit merged after the Release PR blocks its review` |
| Release PR built on unvalidated commits | `test-release-pr-checks.py` `test_base_must_be_validated_main`: no required checks dispatched |
| `status --tag` of an unmerged stable release = `status` | `open Release PR: status --tag reaches the same decision as status`; `status --tag and status agree on hidden-only commits`; `status --tag and status agree on a stale Release PR`; `status --tag: a releasable commit after the Release PR requires a refresh, never a review`; `status --tag: a stale Release PR on a red main is blocked, never reviewed`; `status --tag of that Release PR version agrees with status` |
| exact Release PR version | `status --tag v0.2.1 never accepts the Release PR of 0.2.10`; `status --tag v0.2.10 never accepts the Release PR of 0.2.1`; `status --tag needs the exact Release PR title, not a substring`; `status --tag is blocked like status when more than one Release PR is open` |
| previous tag without GitHub Release → no start | `previous tag without its GitHub Release: start_release is not offered`; `… start is refused`; `… no dispatch, no Release PR` |
| previous GitHub Release draft / wrong revision / other tag | `previous GitHub Release still a draft: …`; `previous GitHub Release at another revision: …`; `previous GitHub Release bound to another tag: …`; `refused previous-release states dispatched nothing and opened no Release PR` |
| previous release unpublished while a Release PR is open | `an open Release PR is not offered for review while the previous release is unpublished`; `the unpublished previous release blocks status and status --tag alike` |
| ADR-0010 recovery release as previous release | `a recovery-published previous release allows the next start`; `malformed recovery pins on the previous release block the next start`; `a previous release pinning a control revision off main runs none of its scripts and blocks the start`; `… older than its source blocks the start`; `… its own release commit as control blocks the start`; `a recovery previous release without its revision pin blocks the next start` |
| `status --tag` while Release Please runs | `status --tag also waits for an in-flight Release Please run` |

## Independent review

A separate read-only security/authority review of the first commit found no
critical issue (no discovery path publishes anything outside the authorized
envelope; no injection path; SemVer table matches CONTRIBUTING; no regression
of tag, revision or digest enforcement; bash 3.2 compatible). Its findings were
fixed in this PR:

- high: `verify` compared notes with an unquoted (glob) right-hand side, so
  identical stable notes containing `[x](y)` failed; now quoted, and the
  fixture changelog contains `[#2](…)` and `[0.2.0]` (a mutant with the old
  comparison fails the suite);
- medium: Release Please reads `main` when it runs, after the plan check;
  `release-pr-checks.sh dispatch` now requires the Release PR base to be the
  validated SHA or an ancestor of it; docs no longer overstate the check;
- medium: the refresh path now has the same red-CI and existing-release
  gates as a new start;
- low: prepared-run discovery only considers runs from `main` and reports
  rejected candidates; `verify` checks the prepared run's origin; plan
  subjects drop bidi control characters and the workflow summary is fenced;
  docs state that a Release Please divergence invisible to Git can update the
  Release PR before the version check refuses its required checks.

Found while fixing: Release Please keeps an unchanged Release PR as is, so a
hidden-only commit merged after it would have made `refresh_release_pr` loop.
`status` now asks for a refresh only for a releasable commit or another
planned version, and otherwise for the branch update the strict ruleset
already requires.

## PR #164 review findings (second round)

- major: `status --tag vX.Y.Z` of a stable release not merged yet accepted
  any open Release PR whose title *contained* the version (`0.3.1` matched
  `0.3.10`) and answered `review_release_pr` without `release_pr_facts`
  (no staleness, plan, `main` CI, base or refresh check). Now it considers
  only the Release PR titled exactly `chore(main): release X.Y.Z` and judges
  it through `open_release_pr_status`, the same path as `status`; the tests
  compare the `state`/`next_action`/`reason` of both commands.
- major: the previous release was proven only by its tag (`release-plan.sh`
  is Git-only and stays so). `release.sh` now requires
  `previous_release_state=published` from `publish-release.sh --check` for
  `plan.previous_tag` at `plan.previous_release_commit` before
  `start_release`, `refresh_release_pr` or `review_release_pr`; a missing
  GitHub Release, a draft, a release at another revision or bound to another
  tag fails closed. A previous ADR-0010 recovery release is checked exactly
  as `verify` checks one: the pins its notes record (`published_recovery_pins`)
  go through `pinned_control` (control revision on first-parent `main` and
  descending from the release commit, verified checkout, that revision's
  `recovery_validate`), then that revision's `publish-release.sh --check`.
  Both helpers are now shared with `verify`.

An independent read-only review of these fixes then found, and this PR fixed:
the first version of the recovery branch checked only first-parent `main`
(not ancestry, not the presence of the recovery protocol) before running the
control revision's scripts, and its clone could continue on the wrong
revision when the checkout failed inside `$(status)` (no `errexit`); both now
go through `pinned_control`, which chains the checkout and asserts `HEAD`.
`status --tag` without an exact Release PR now reports
`state=no_release_in_progress` (it said `release_pr_merged`), and its
in-flight Release Please branch is tested. ADR-0011 and the command
reference no longer overstate which checks gate the review.

Mutation testing (each mutant run against the full suite, then restored):

| Mutant | Killed by |
|---|---|
| old `contains` + direct `review_release_pr` | 14 checks (exact version, same decision, stale PR) |
| substring match through the shared path | `status --tag v0.2.1 never accepts the Release PR of 0.2.10`; `… not a substring` |
| exact match but no `release_pr_facts` | 12 checks (same decision, hidden-only, stale, red `main`) |
| no previous-release check | 17 checks |
| draft accepted as published | 11 checks (draft, other revision, other tag) |
| recovery pins not passed to the check | `a recovery-published previous release allows the next start` |
| previous-release check only in the no-release path | 17 checks (review of an open Release PR) |
| local scripts instead of the pinned control scripts | `a recovery-published previous release allows the next start` (later notes format) |
| no ancestry check in `pinned_control` | `a previous release pinning a control revision older than its source blocks the start` |

## Read-only check against the real repository

`./scripts/release.sh status` on `main` `db8ed3afc0538171552870712f8f97562feda016`
(only `git fetch` and GitHub reads):

```text
unpublished_release_prs=#160
tag=v0.3.0
revision=b79d3bf8cbf21247ca30cae06ff000e7f89a5adf
state=release_pr_merged
next_action=blocked
reason=commit 684b5aca88b1176d317fa86aa223b6bc654e4519 in the v0.3.0 range has no delivery metadata; ...; this release commit already exists: see docs/development/release-recovery.md
```

With the ADR-0010 pins (`--corrections-revision db8ed3a…`, digest of the
committed corrections file) the same status reports `next_action=prepare`:
v0.3.0 stays on its recovery path. `scripts/release-plan.sh` on that `main`
refuses a new release (`release v0.3.0 is recorded on main but not published`);
against a fake remote where v0.3.0 is published it plans `0.3.1` (two `fix`
commits, metadata declared).

After the second-round fixes, `release.sh status` on the same `main` still
reports the v0.3.0 recovery path unchanged, and the previous-release check
passes on the real published v0.2.1
(`publish-release.sh --check --tag v0.2.1 --revision 2182b3e…` →
`publication_state=published`, `result=pass`), so it does not block a real
published release.

## Not verified here

- The previous-release check runs at `status`/`start` time; `release-please.yml`
  re-runs only the Git-only plan. A GitHub Release deleted between `status`
  and the dispatch (seconds, admin-only), or a direct maintainer dispatch of
  `release-please.yml`, is not checked by the workflow; the next `status`
  blocks the review of the resulting Release PR. `prepare` and `publish` of a
  merged Release PR do not re-check the previous release.

- The dispatch-only `release-please.yml`, the Release Please action and
  `release-pr-checks.sh dispatch` on GitHub: covered by static contracts and
  fakes only, until the first real `$axiom-release` start after merge.
- The plan reproduces the documented Release Please bump table; a PR-body
  `BEGIN_COMMIT_OVERRIDE` added after merge is invisible to Git. Such a
  divergence is refused by the planned-version checks, not predicted.
