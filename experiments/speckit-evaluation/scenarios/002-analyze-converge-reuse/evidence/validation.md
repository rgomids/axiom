# Validation Evidence

Original experiment validation executed on 2026-08-11. Methodology-fix
validation repeated from the Axiom repository on 2026-08-12 without model
re-execution.

Passed:

- Scenario 002 frozen input checksums;
- neutral protocol and prototype checksums;
- `bash -n` for every experimental shell script and failure fixture;
- deterministic validator/scorer tests;
- exact stable-reference scoring, including substring false-match rejection;
- manual unexpected-finding review, machine-readable review validation, and
  exact regeneration of all three derived score files;
- adapter unavailable, incompatible-output, stale-output, and capability-error tests;
- neutral-result validation for current baseline, C experimental, and B;
- JSON and JSONL parsing for all Scenario 002 data/events, plus syntactic JSON
  validation of both schema files;
- repository bootstrap validation;
- agent-package validation and its behavior tests;
- sensitive-file checker and its behavior tests against the worktree;
- repository-wide local Markdown link resolution;
- `git diff --check`;
- absence of root `.specify/`, root `speckit-*` skills, and dependency-manifest changes;
- no tracked modifications to Scenario 001's existing `scenario/`, `axiom/`,
  `speckit/`, `comparison/`, or `evidence/` trees.
- no changes to original Scenario 002 model results, event streams, raw result,
  timings, or frozen scenario/protocol inputs;
- no permanent dependency or root Spec-Kit state.

Limitations:

- `gitleaks` unavailable; not installed automatically;
- `shellcheck` unavailable; `bash -n` and executable behavior tests passed;
- a standalone JSON Schema engine was unavailable; the repository's
  deterministic result/review validators and JSON schema syntax checks passed;
- hosted CI has no configured checks for this PR at validation time;
- staged-index scan is performed after final staging and recorded in the commit
  handoff.
