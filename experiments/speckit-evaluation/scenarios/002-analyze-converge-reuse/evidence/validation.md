# Validation Evidence

Executed from the Axiom repository on 2026-08-11.

Passed:

- Scenario 002 frozen input checksums;
- neutral protocol and prototype checksums;
- `bash -n` for every experimental shell script and failure fixture;
- deterministic validator/scorer tests;
- adapter unavailable, incompatible-output, stale-output, and capability-error tests;
- neutral-result validation for current baseline, C experimental, and B;
- JSON and JSONL parsing for all Scenario 002 data/events;
- repository bootstrap validation;
- agent-package validation and its behavior tests;
- sensitive-file checker and its behavior tests against the worktree;
- repository-wide local Markdown link resolution;
- `git diff --check`;
- absence of root `.specify/`, root `speckit-*` skills, and dependency-manifest changes;
- no tracked modifications to Scenario 001's existing `scenario/`, `axiom/`,
  `speckit/`, `comparison/`, or `evidence/` trees.

Limitations:

- `gitleaks` unavailable; not installed automatically;
- `shellcheck` unavailable; `bash -n` and executable behavior tests passed;
- hosted CI has no configured checks for this PR at validation time;
- staged-index scan is performed after final staging and recorded in the commit
  handoff.
