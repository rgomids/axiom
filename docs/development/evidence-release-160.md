# Release PR #160 classification repair

Recorded: 2026-10-02. Scope: release metadata correction and PR title validation.
No merge, publication, tag mutation, repository-setting change or main-history
rewrite is authorized by this repair.

## Cause and version decision

Main revision: `90cfcdd6d3df9f1bdc6049c3f825c2f67b7e3025`.
The API comparison `v0.2.1...main` contains exactly these four commits:

| PR | Squash SHA | Classification | Changelog |
|---|---|---|---|
| #158 | `30deb513dfaaa8ec93e224efee76981a33a05bad` | `feat(site)` after correction | Features |
| #159 | `142618fb385691172d48de18d508b3e55a1f9c54` | `fix(ci)` | Bug Fixes |
| #157 | `684b5aca88b1176d317fa86aa223b6bc654e4519` | `docs` | Hidden by configuration |
| #161 | `90cfcdd6d3df9f1bdc6049c3f825c2f67b7e3025` | `fix(install)` | Bug Fixes |

[Original Release Please run](https://github.com/rgomids/axiom/actions/runs/37014116904)
reported `commit could not be parsed: 30deb513dfaaa8ec93e224efee76981a33a05bad Docs/86 added landing page (#158)`
and `unexpected token` at `1:8`. It omitted the feature, yielding `0.2.2`.
The #158 description and changed files establish a new public landing page,
not merely documentation. The squash commit has one parent; its subject is the
nonconventional PR title. Repository settings use `PR_TITLE` and `PR_BODY`.

`CONTRIBUTING.md` and `release-please-config.json` specify a minor bump for a
feature during `0.x`; the greatest bump since `0.2.1` is therefore `0.3.0`.
No forced `Release-As`, versioning-config change or invented feature is needed.

## Correction applied

Preserved the complete existing body of merged PR #158 and appended:

```text
BEGIN_COMMIT_OVERRIDE
feat(site): add bootstrap landing page (#158)
END_COMMIT_OVERRIDE
```

This is the [official Release Please correction mechanism](https://github.com/googleapis/release-please#how-can-i-fix-release-notes)
for squash-merged PRs. It changes parser metadata, not the commit or its files.
Dispatched only `.github/workflows/release-please.yml` on `main`; its pinned
Action and configuration disable GitHub Release creation.

[Regeneration run](https://github.com/rgomids/axiom/actions/runs/37020489412)
succeeded and updated existing [PR #160](https://github.com/rgomids/axiom/pull/160)
to `chore(main): release 0.3.0`, head
`468954dc62651330289e2f50077e6cae0bddbc50`.
The diff contains only `.release-please-manifest.json` (`0.3.0`) and
`CHANGELOG.md` (one Features entry, both Bug Fixes entries). Documentation is
correctly hidden. Release Please generated the branch; no manual branch edit.

[Release CI run](https://github.com/rgomids/axiom/actions/runs/37020527115)
passed `verify (linux)`, `verify (macos)` and `release-contract` on that exact
head. [Delivery run](https://github.com/rgomids/axiom/actions/runs/37020532567)
passed `delivery-metadata`. Superseded automatic PR-event runs requested action;
the repository's intended dispatch runs supplied all four successful contexts.

## Recurrence protection

Before this repair, `delivery-issues.sh check-pr` validated Issue metadata and
closing keywords but accepted malformed titles. Added Conventional Commit
title validation there, retaining accepted types from `CONTRIBUTING.md` and
existing safe file/environment input. Existing `delivery-metadata` PR events
include `edited`; validation uses code from the base revision.
No extra workflow, dependency, permission or required-check name is introduced.

Regression tests: seven malformed titles, including the exact #158 title,
incorrect case/type, empty description/scope, missing colon and multiline
input, all incorrectly passed before the implementation and all are now
rejected. Six valid scoped/unscoped/breaking/release/revert titles pass.
The complete delivery suite also passes.

**Activation pending:** this protection requires human review and merge of its
separate PR. It cannot protect its own PR through the base-revision path.
The live rules API currently requires `verify (linux)`, `verify (macos)` and
`release-contract`; it omits `delivery-metadata`, despite the desired state in
`.github/rulesets/main.json`. A maintainer must reconcile the required-check
settings. No settings were changed. Semantic classification still needs review;
syntax validation does not prove that a chosen type describes the change.

## Executed local validation

- `./scripts/test-delivery.sh`: PASS; regression test red phase showed seven failures.
- `python3 scripts/test-release-pr-checks.py`: PASS, 10 tests.
- `bash -n scripts/delivery-issues.sh scripts/test-delivery.sh`: PASS.
- `./scripts/validate-repository.sh .`: PASS.
- `git diff --check`: PASS.
- `gitleaks detect --source . --no-git --redact`: PASS.

Go production code is unchanged; full product verification runs in CI.
Raw bounded investigation outputs are retained locally under
`/tmp/axiom-release-160-evidence`; public Evidence contains only repository facts.

## Publication verification

API snapshots before/after correction match on all tag refs and SHAs,
release identities, publication state and asset identities/digests.
The regeneration run has zero retained workflow artifacts. No preparation or
publish workflow was dispatched. Neither PR was merged; main remains unchanged.
