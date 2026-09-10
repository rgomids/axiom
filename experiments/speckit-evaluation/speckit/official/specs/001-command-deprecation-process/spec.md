# Feature Specification: Documented Command Deprecation Process

**Feature Branch**: `main`

**Created**: 2026-08-11

**Status**: Draft — clarified with one residual trigger-boundary ambiguity

**Input**: User description: "Add a lightweight, repository-local process for deprecating documented
commands. Contributors need to know when a deprecation record is required, how to create it, and how
reviewers confirm migration and rollback information before approval. Keep existing commands
unchanged."

## Clarifications

### Session 2026-08-11

- Q: What lifecycle and approval authority govern a command deprecation? → A: Use `Proposed ->
  Approved -> Deprecated -> Removed`. A proposed record changes no command status. A repository
  maintainer must approve transition to `Approved`; command owners execute later transitions.
- Q: Where are records stored and how are they named? → A: Keep records in
  `docs/deprecations/` using `NNNN-short-name.md`; maintain an index in
  `docs/deprecations/README.md` and a reusable `docs/deprecations/template.md`.
- Q: What content is mandatory in every deprecation record? → A: Status, affected command,
  replacement or explicit absence, rationale, affected users, migration steps, notice/release
  target, rollback plan, owner, approval evidence, and relevant links.
- Q: What notice and exception rule applies before removal? → A: Removal normally follows at least
  one published release containing the deprecation notice. An urgent security removal may bypass
  that minimum only when the record states rationale, impact, owner, approval, and compensating
  migration guidance.
- Q: What enforcement belongs in this slice? → A: No executable validator or CI. Provide a reviewer
  checklist and use deterministic repository checks for file presence, required headings, links,
  whitespace, and scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recognize Required Records (Priority: P1)

A contributor proposing a documented command lifecycle change can determine whether a deprecation
record is required before changing documentation or behavior.

**Why this priority**: A reliable trigger prevents command changes from bypassing impact review.

**Independent Test**: A reviewer can apply the documented trigger to representative command-change
proposals and reach the expected record-required or record-not-required result.

**Acceptance Scenarios**:

1. **Given** a proposed change that matches a documented trigger, **When** the contributor checks the
   process, **Then** the process states that a deprecation record is required before approval.
2. **Given** a proposed change outside the documented trigger, **When** the contributor checks the
   process, **Then** the process states whether no record is required and explains the boundary.

---

### User Story 2 - Create a Durable Record (Priority: P2)

A contributor can create a repository-local deprecation record from a Markdown template and supply
the information required for review.

**Why this priority**: A durable record makes impact, migration, timing, ownership, and rollback
context reviewable without chat history or an external provider.

**Independent Test**: Starting from the template, a contributor can create a sample record whose
required sections are identifiable and whose repository path follows the documented convention.

**Acceptance Scenarios**:

1. **Given** a change requiring a record, **When** the contributor follows the process, **Then** they
   can locate the template and determine the record's required location and name.
2. **Given** a completed record, **When** a reviewer reads it from a local checkout, **Then** every
   mandatory field is present and understandable without provider or chat context.

---

### User Story 3 - Review Migration and Rollback (Priority: P3)

A reviewer can confirm that a proposed command deprecation contains adequate migration and rollback
information and the required approval before accepting it.

**Why this priority**: Review criteria protect command users from undocumented disruption while
keeping the approval decision human-controlled.

**Independent Test**: Reviewers can use the documented criteria on complete and incomplete sample
records and consistently identify whether each is eligible for approval.

**Acceptance Scenarios**:

1. **Given** a record missing required migration or rollback information, **When** it is reviewed,
   **Then** the process prevents approval and identifies the missing information.
2. **Given** a complete record, **When** it is reviewed, **Then** the process identifies the human
   authority whose approval is required before any lifecycle change takes effect.
3. **Given** a proposed shortened or exceptional lifecycle, **When** it is reviewed, **Then** the
   documented notice and exception rules determine whether it can proceed.

### Edge Cases

- A documentation edit changes command wording but not the command name, behavior, or lifecycle.
- An urgent security removal is proposed before any published release contains the deprecation
  notice; the record must contain every required exception statement before it may bypass the
  normal minimum.
- Migration guidance exists but no viable rollback path exists.
- The named owner or approval authority is unavailable or has changed.
- A record references an undocumented replacement command.
- A contributor creates a record with a duplicate or conflicting identity.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The process MUST define the command changes that require a deprecation record and the
  changes that do not.
- **FR-002**: The process MUST provide `docs/deprecations/template.md` as the reusable,
  repository-local Markdown template and `docs/deprecations/README.md` as the record index.
- **FR-003**: Each record MUST be stored in `docs/deprecations/` and named
  `NNNN-short-name.md`, where `NNNN` is a four-digit identifier and `short-name` is a concise
  hyphenated label. The index MUST link every record. Allocation and collision rules remain
  unresolved.
- **FR-004**: The template MUST require status, affected command, replacement or explicit absence,
  rationale, affected users, migration steps, notice/release target, rollback plan, owner, approval
  evidence, and relevant links.
- **FR-005**: The lifecycle MUST be `Proposed -> Approved -> Deprecated -> Removed`. Creating a
  `Proposed` record MUST NOT change command status. A repository maintainer MUST approve transition
  to `Approved`; command owners MUST execute transitions to `Deprecated` and `Removed`.
- **FR-006**: Removal MUST normally follow at least one published release containing the deprecation
  notice. An urgent security removal MAY bypass that minimum only when the record states rationale,
  impact, owner, approval, and compensating migration guidance.
- **FR-007**: Review criteria MUST prevent approval while required migration or rollback information
  is absent or while required human approval is unresolved.
- **FR-008**: Process and record artifacts MUST remain understandable from a local checkout without
  chat history, provider context, credentials, or network access.
- **FR-009**: The change MUST preserve the documented names and behavior of `example validate` and
  `example package`; it MUST NOT deprecate, rename, remove, or implement either command.
- **FR-010**: The process MUST provide a reviewer checklist covering record completeness, lifecycle
  authority, migration, notice, rollback, exception handling, file presence, required headings,
  links, whitespace, and documentation-only scope.
- **FR-011**: The first slice MUST NOT add application code, executable validation, dependencies, CI
  workflows, provider integration, credentials, or network requirements.
- **FR-012**: Relevant completed documentation changes MUST be recorded in `CHANGELOG.md`.
- **FR-013**: Validation guidance MUST use deterministic local commands for file presence, required
  headings, repository-local links, whitespace, and repository scope without adding an executable
  validator.

### Key Entities

- **Documented Command**: A contributor-facing command with a stable name and described behavior;
  currently `example validate` or `example package`.
- **Deprecation Record**: A durable Markdown artifact describing one proposed command lifecycle
  change, its impact, migration, timing, ownership, approval, and rollback information, stored as a
  `docs/deprecations/NNNN-short-name.md` file and listed in the index. Its required
  content is status, affected command, replacement or explicit absence, rationale, affected users,
  migration steps, notice/release target, rollback plan, owner, approval evidence, and relevant
  links.
- **Lifecycle Stage**: One of `Proposed`, `Approved`, `Deprecated`, or `Removed`, in that order. A
  proposed record does not change command status.
- **Approval Decision**: Repository-maintainer authorization required for the transition to
  `Approved`; command owners execute later transitions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every representative proposal used in acceptance review, contributors can
  determine whether a deprecation record is required using only repository documentation.
- **SC-002**: A contributor can locate the template and create a structurally complete sample record
  in 10 minutes or less without network access or provider-specific knowledge.
- **SC-003**: Two reviewers applying the documented criteria to the same complete and incomplete
  sample records reach the same approval-readiness result in every acceptance case.
- **SC-004**: Every mandatory field, lifecycle stage, approval requirement, notice rule, exception
  rule, migration expectation, and rollback expectation has one unambiguous documented definition,
  including the one-published-release minimum and urgent-security exception content.
- **SC-005**: Deterministic local validation reports no broken repository-local links, no unresolved
  template placeholders in completed artifacts, no changes outside documentation and Spec-Kit
  feature artifacts, and no changes to existing command descriptions or behavior.

## Assumptions

- Contributors and reviewers work from a local repository checkout and can read and edit Markdown.
- The process governs documented contributor-facing commands, not a general release-management
  system.
- No command is currently deprecated; any example record used for review is illustrative only.
- One batched human response supplied lifecycle, authority, record convention, mandatory content,
  notice, exception, and enforcement decisions.
- Automated enforcement is outside the first slice because executable validators, dependencies,
  CI, and provider workflows are prohibited.

## Non-Goals

- Deprecate, rename, remove, replace, or implement any command.
- Build a CLI, Lingo feature, application, service, database, or executable validator.
- Add dependencies, CI, GitHub Issues, pull-request automation, provider integration, credentials,
  or network requirements.
- Design a general release-management system.
- Adopt Axiom or Spec-Kit as a dependency of the sample repository.

## Unresolved Decisions

- A record is required for a proposal explicitly described as command deprecation. Whether a rename,
  replacement, or incompatible behavior change independently triggers the same process remains a
  residual human decision.
- The human answer fixes the `NNNN-short-name.md` shape but not number allocation or collision
  handling.
- The human answer requires one published release of notice but does not define release-publication
  evidence; records must preserve relevant evidence without assuming a provider.
