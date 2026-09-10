# Tasks — Documented Command Deprecation Process

## Ordered breakdown

### T-001 — Incorporate human clarification

- **Status:** Completed
- **Objective:** Convert frozen answers into clarified behavior without changing
  the answer source or erasing original questions.
- **Scope:** `experiment-output/specification.md` and
  `experiment-output/clarifications.md`.
- **Dependencies:** Phase-1 artifacts and `.experiment/operator-answers.md`.
- **Acceptance/evidence:** Answer mapping and residual gaps are explicit;
  operator-answer hash remains stable.
- **Repository impact:** Experiment evidence only.

### T-002 — Decide ADR treatment

- **Status:** Completed
- **Objective:** Apply the architecture threshold and record whether an ADR is
  justified.
- **Scope:** `experiment-output/decisions.md`.
- **Dependencies:** T-001.
- **Acceptance/evidence:** Alternatives, trade-offs, decision, and revisit
  conditions recorded.
- **Repository impact:** Experiment evidence only; no ADR added.

### T-003 — Add deprecation process and template

- **Status:** Completed
- **Objective:** Create the smallest provider-neutral process satisfying frozen
  lifecycle, location, content, notice, exception, and enforcement answers.
- **Scope:** `docs/deprecations/README.md` and
  `docs/deprecations/template.md`.
- **Dependencies:** T-001 and T-002.
- **Acceptance/evidence:** FR-001 through FR-010 and FR-012 through FR-014;
  required-heading checks.
- **Repository impact:** Two new Markdown files; no current deprecation record.

### T-004 — Reconcile discoverability and changelog

- **Status:** Completed
- **Objective:** Link the process from existing contributor entrypoints and
  record the relevant change.
- **Scope:** `README.md`, `docs/contributing.md`, and `CHANGELOG.md`.
- **Dependencies:** T-003.
- **Acceptance/evidence:** Links resolve; changelog contains dated entry.
- **Repository impact:** Three minimal Markdown edits.

### T-005 — Validate deterministically

- **Status:** Completed
- **Objective:** Test acceptance, links, headings, whitespace, scope, command
  preservation, executable bits, and sensitive content.
- **Scope:** Whole sample workspace with focus on changed paths.
- **Dependencies:** T-003 and T-004.
- **Acceptance/evidence:** Exact commands and exit outcomes in
  `experiment-output/validation.md`.
- **Repository impact:** Evidence only; no generated validator.

### T-006 — Review and correct

- **Status:** Completed
- **Objective:** Review specification compliance, architecture, verification,
  security, and completion criteria; fix any in-scope finding.
- **Scope:** Final diff and all experiment evidence.
- **Dependencies:** T-005.
- **Acceptance/evidence:** Severity-based result in
  `experiment-output/review.md`; no unresolved blocking finding for completed
  scope.
- **Repository impact:** Corrections only if findings require them.

### T-007 — Reconcile and close execution

- **Status:** Completed
- **Objective:** Confirm affected durable docs, compare all shared completion
  criteria, and record execution metrics and limitations.
- **Scope:** Documentation set and `experiment-output/execution.md`.
- **Dependencies:** T-006.
- **Acceptance/evidence:** Completion matrix, human interaction count,
  measurable elapsed time, difficulties, ambiguities, and unverified claims.
- **Repository impact:** Evidence finalization only.
