# Change-aware PR CI

Issue #243 routes CI from the exact PR base/head Git diff, including both
rename sides and deleted paths. Manual dispatch always runs the full gate.
The classifier uses a closed path allowlist; empty, unknown or mixed categories
run every suite. No labels or commit messages influence routing.

| Single category | Applicable suites |
| --- | --- |
| Documentation (Markdown under docs, root README/CHANGELOG) | Repository |
| Site and landing-page validation | Repository plus existing path-filtered landing-page workflow |
| Repository policies, skills and rulesets | Repository |
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
Matrix conclusion checks conservatively share the complete suite result.

The classifier prints changed paths/categories as JSON and writes routing to
the Actions summary. Conclusion jobs explain whether the suite passed or was
not applicable. Run local routing fixtures with:

```sh
python3 scripts/test-ci-classification.py
```

No main push CI or release-preparation behavior changes. The existing strict
up-to-date PR policy and release tree-equivalence lookup keep the same required
contexts; a merged tree reuses the checks for that exact PR tree.

## Implementation validation (2026-10-09)

- Routing fixtures pass, including real Git rename/deletion paths and manual
  full selection. Each required conclusion's actual Bash body is exercised for
  success, expected skip, missing output, failure and cancellation (40 cases).
- Go-quality workflow contracts (2 tests), code-health contracts (11 tests),
  automation registry coverage and actionlint v1.7.12 pass.
- Repository-wide validation remains unverified locally: this Windows checkout
  materialized Claude symlinks as regular files. Registry unit fixtures also
  require POSIX executable bits/symlink privilege; release dispatch tests execute
  shell scripts directly and fail with WinError 193 here.
- GitHub Actions execution and required check conclusions remain to be observed
  on the implementation PR. No remote check or CI-minute reduction is claimed.
