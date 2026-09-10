---

description: "Task list for documented command deprecation process"
---

# Tasks: Documented Command Deprecation Process

**Input**: Design documents from `/specs/001-command-deprecation-process/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: No executable test artifact is permitted. Each story uses deterministic local document
checks from `quickstart.md`, with observed outcomes recorded there.

**Organization**: Tasks are grouped by user story and preserve documentation-only scope.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it changes a different file after core artifacts are stable.
- **[Story]**: Maps work to the corresponding specification user story.

## Phase 1: Setup (Evidence Baseline)

**Purpose**: Capture current invariants before documentation changes.

- [X] T001 Record current `docs/commands.md` command headings and descriptions, zero-deprecation state, and initial scope in `specs/001-command-deprecation-process/quickstart.md`

---

## Phase 2: Foundational (Documentation Skeleton)

**Purpose**: Create the two frozen repository-local Markdown artifacts required by all stories.

- [X] T002 Create `docs/deprecations/README.md` with process title, explicit empty state, template link, and record-index section
- [X] T003 Create `docs/deprecations/template.md` with reusable record title and all eleven required headings from FR-004

**Checkpoint**: Required directory, index, and template exist without changing command documentation.

---

## Phase 3: User Story 1 - Recognize Required Records (Priority: P1) 🎯 MVP

**Goal**: Contributors can identify the core deprecation trigger, lifecycle, authority, notice rule,
urgent exception, and declared residual boundaries.

**Independent Test**: Read only `docs/deprecations/README.md`; confirm it distinguishes explicit
deprecation from unresolved extra trigger classes and states every lifecycle and approval rule.

- [X] T004 [US1] Document the explicit-deprecation trigger and unresolved rename, replacement, and incompatible-change boundary in `docs/deprecations/README.md` per FR-001
- [X] T005 [US1] Document `Proposed -> Approved -> Deprecated -> Removed`, no status change at `Proposed`, maintainer approval, and command-owner transitions in `docs/deprecations/README.md` per FR-005
- [X] T006 [US1] Document one-published-release notice and urgent-security exception content in `docs/deprecations/README.md` per FR-006
- [X] T007 [US1] Apply record-trigger guidance to explicit deprecation, removal, editorial correction, and standalone incompatible-change cases and record resolved or residual results in `specs/001-command-deprecation-process/quickstart.md` per SC-001

**Checkpoint**: User Story 1 is independently reviewable from repository documentation.

---

## Phase 4: User Story 2 - Create a Durable Record (Priority: P2)

**Goal**: Contributors can locate, name, and complete a durable Markdown record.

**Independent Test**: Starting from `docs/deprecations/template.md`, confirm all required content is
prompted and `docs/deprecations/README.md` provides the frozen path, filename shape, and index.

- [X] T008 [US2] Add concise instructions and explicit-absence guidance beneath every required heading in `docs/deprecations/template.md` per FR-004
- [X] T009 [US2] Document `docs/deprecations/NNNN-short-name.md`, index maintenance, and unresolved number allocation in `docs/deprecations/README.md` per FR-002–FR-003
- [X] T010 [US2] Run required-heading, file-presence, known-link, and time-boxed template-completion checks and record results in `specs/001-command-deprecation-process/quickstart.md` per SC-002

**Checkpoint**: User Story 2 is independently usable without provider or chat context.

---

## Phase 5: User Story 3 - Review Migration and Rollback (Priority: P3)

**Goal**: Reviewers can determine approval readiness and run deterministic local checks.

**Independent Test**: Apply the reviewer checklist to the template as an incomplete record and to a
fully populated hypothetical record; required omissions block readiness and no provider is needed.

- [X] T011 [US3] Add reviewer checklist for record completeness, lifecycle authority, migration, notice, rollback, urgent exception, and approval evidence in `docs/deprecations/README.md` per FR-007 and FR-010
- [X] T012 [US3] Add one-off local commands for presence, headings, links, whitespace, scope, and command invariants in `docs/deprecations/README.md` per FR-013
- [X] T013 [US3] Record complete-versus-incomplete review reasoning and command outcomes in `specs/001-command-deprecation-process/quickstart.md` per SC-003–SC-005

**Checkpoint**: All three user stories are independently documented and reviewable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Connect entry points, record lifecycle change, and close validation evidence.

- [X] T014 [P] Link the deprecation process from `README.md` without changing command descriptions
- [X] T015 [P] Link the deprecation process from `docs/contributing.md` while preserving existing guidance
- [X] T016 [P] Record the documentation feature in `CHANGELOG.md`
- [X] T017 Run all `specs/001-command-deprecation-process/quickstart.md` validation commands and record dated pass/fail outcomes, including unavailable scanners, in that file
- [X] T018 Record analysis result, human interaction count, residual ambiguities, architecture decisions, ADR absence rationale, and final scope review in `specs/001-command-deprecation-process/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 has no dependencies.
- Phase 2 depends on Phase 1 and blocks story edits.
- Phases 3–5 depend on Phase 2. They are logically independent but should run sequentially because
  all update `docs/deprecations/README.md` or the shared validation record.
- Phase 6 depends on all three stories.

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Phase 2; no dependency on another story.
- **User Story 2 (P2)**: Starts after Phase 2; independent template and naming outcome.
- **User Story 3 (P3)**: Starts after Phase 2; consumes frozen requirements, not story implementation.

### Parallel Opportunities

- T014, T015, and T016 can run in parallel after core deprecation documentation is stable.
- No other tasks are marked parallel because they share files or validation evidence.

## Parallel Example: Cross-Cutting Documentation

```text
Task: "Link the deprecation process from README.md"
Task: "Link the deprecation process from docs/contributing.md"
Task: "Record the documentation feature in CHANGELOG.md"
```

## Implementation Strategy

### MVP First

1. Complete Phases 1 and 2.
2. Complete Phase 3 for User Story 1.
3. Run and record its independent checks before continuing.

### Incremental Delivery

1. Add trigger and lifecycle guidance.
2. Add record-creation template and naming guidance.
3. Add reviewer checklist and deterministic commands.
4. Reconcile entry points, changelog, and validation evidence.

## Notes

- Every task has a concrete repository path.
- No task creates code, executable validators, dependencies, CI, provider state, credentials, or
  network requirements.
- Completed tasks must be marked `[X]` by `$speckit-implement`.
- Human clarification count remains one batched response.
