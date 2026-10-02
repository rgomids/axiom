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
| `./scripts/test-release-flow.sh` | PASS (all checks, including section 6 "Release Flow v2") |
| `python3 scripts/test-release-pr-checks.py` | OK, 14 tests |
| `./scripts/test-delivery.sh` | PASS |
| `python3 scripts/test-release-corrections.py` | OK |
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

## Not verified here

- The dispatch-only `release-please.yml`, the Release Please action and
  `release-pr-checks.sh dispatch` on GitHub: covered by static contracts and
  fakes only, until the first real `$axiom-release` start after merge.
- The plan reproduces the documented Release Please bump table; a PR-body
  `BEGIN_COMMIT_OVERRIDE` added after merge is invisible to Git. Such a
  divergence is refused by the planned-version checks, not predicted.
