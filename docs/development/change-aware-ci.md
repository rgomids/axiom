# Change-aware PR CI

Issue #243 routes CI from the exact PR base/head Git diff, including both
rename sides and deleted paths. Manual dispatch always runs the full gate.
The classifier uses a closed path allowlist; sensitive, empty, unknown or mixed categories
run every suite. No labels or commit messages influence routing.

| Single category | Applicable suites |
| --- | --- |
| Documentation (Markdown under docs, root README/CHANGELOG) | Repository |
| Site and landing-page validation | Repository plus existing path-filtered landing-page workflow |
| Nonsensitive repository skills/adapters | Repository |
| Policies, rulesets, workflows, CODEOWNERS, security guide and maintainer entrypoints | All suites |
| Go source/modules | Go quality, multi-OS verify, upgrade journeys |
| Installer or compatibility | All suites |
| Release scripts/contracts | Release contracts |
| Unknown, mixed, manual dispatch | All suites |

Repository validation runs unconditionally. Workflow changes conservatively
run all suites except the existing landing-page workflow. Required contexts
retain their names from the main ruleset. Each always-running conclusion job
requires successful classification and repository validation, and either a
successful applicable suite or an explicitly unselected skipped suite.
Failure, cancellation or unexpected skip cannot produce a passing conclusion.
Matrix conclusions require both the complete suite result and a successful
completion record for their own platform. That record must match the native
runner OS, suite, tested SHA, repository, event, run ID and run attempt. A
removed platform cannot be hidden by other successful matrix rows. Missing,
malformed or mismatched platform records fail closed.

The classification checkout is shallow (depth 1). PR classification fetches
only the exact base/head SHAs with depth 64 and without tags, then verifies a
merge-base. A missing ref, failed fetch or insufficient history selects FULL
with `unverifiable_revision_relation`; it never permits partial routing.
No unbounded history fetch is needed by classification. Product/release jobs
retain their existing history requirements.

## Evidence contract (schema version 1)

Each run retains `ci-plan-<attempt>` and `ci-outcome-<suite>-<platform>-<attempt>`
artifacts for 30 days (non-matrix outcomes omit platform). Completed physical
matrix rows also retain `ci-row-<suite>-<platform>-<attempt>`. Evidence is scoped
to the current run and attempt; required conclusions download only that run's
exact artifact names. Upload/download failures fail the corresponding job.

The plan contains `schema_version`, `identity` (event, repository, tested SHA,
run ID/attempt, base and head), `changes` (status, path and matched category),
`paths`, `categories`, `full`, `reason`, the four routing flags and `suites`.
Each suite has a boolean `selected` and an explicit inclusion/N/A `reason`.
Reasons distinguish single-category routing, sensitive/unknown/mixed/empty
FULL selection, manual dispatch and an unverifiable revision relation.
The tested SHA identifies the checked-out policy and test tree; PR base/head
identify the immutable change set. Dispatch records its tested SHA as head,
has no PR base and always selects FULL.

Conclusions validate the complete v1 plan by recomputing its shape and routing,
including N/A reasons, and matching the current run identity and selected flag.
Outcome records include identity, suite/platform, selection, real upstream
classification/repository/suite results, reason, platform Evidence and final
conclusion. N/A is recorded as `selected=false`, `result=skipped`, with an
explicit exclusion reason; it is never reported as a physical test pass.
Absent or malformed plans produce a failing outcome with
`classification_unavailable`. Cancellation can prevent artifact retention;
it cannot produce a passing required check.

The classifier prints the plan and writes it to the Actions summary; required
conclusions summarize actual outcomes. Run local routing fixtures with:

```sh
python3 scripts/test-ci-classification.py
```

No main push CI or release-preparation behavior changes. The existing strict
up-to-date PR policy and release tree-equivalence lookup keep the same required
contexts; a merged tree reuses the checks for that exact PR tree.

## Implementation validation (2026-10-09)

- Routing fixtures pass, including real Git rename/deletion paths, manual
  FULL selection, sensitive precedence and complete plan validation. Conclusion
  contracts cover success, expected skip, missing output, failure, cancellation,
  spoofed N/A and absent/mismatched per-platform records. CLI fixtures check
  actual outcome artifacts, missing refs and insufficient shallow ancestry.
- Go-quality workflow contracts (2 tests), code-health contracts (11 tests),
  automation registry coverage and actionlint v1.7.12 pass.
- Repository-wide validation remains unverified locally: this Windows checkout
  materialized Claude symlinks as regular files. Registry unit fixtures also
  require POSIX executable bits/symlink privilege; release dispatch tests execute
  shell scripts directly and fail with WinError 193 here.
- GitHub Actions execution and required check conclusions remain to be observed
  on the implementation PR. No remote check or CI-minute reduction is claimed.
