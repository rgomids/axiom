# Intake — Documented Command Deprecation

## Status

- Phase: intake complete
- Next gate: batched human clarification
- Planning and later phases: not started

## Source boundary

This intake treats `.experiment/brief.md` as neutral scenario input. Current
repository state, the Axiom constitution, Axiom context, policies, architecture,
accepted decisions, research hypotheses, and current specifications provide the
governing context. Spec-Kit material was not consulted.

## Problem

The repository documents user-facing commands but has no durable, reviewable
command-deprecation process. A contributor could remove or rename a documented
command without recording user impact, migration guidance, timing, ownership,
or rollback considerations.

## Desired outcome

Contributors can determine when a command-deprecation record is required,
create one from a repository-local Markdown template, and have reviewers assess
it against explicit migration and rollback expectations. The process remains
understandable from a local checkout without chat history or a specific work
tracking provider.

## Actors

- Contributor proposing a change to a documented command.
- Reviewer checking whether a record is required and complete.
- Approval authority for lifecycle transitions or exceptions: unknown.

## Facts

### Observed repository facts

- `README.md` links to `docs/commands.md` and `docs/contributing.md`.
- `docs/commands.md` documents `example validate` and `example package` as
  supported commands.
- No documented command is marked deprecated.
- `docs/contributing.md` requires small changes, preserved command behavior,
  and a `CHANGELOG.md` update for relevant changes.
- `CHANGELOG.md` contains the initial documentation entry dated 2026-08-01.
- The repository contains no application runtime or dependency manifest.
- No `experiment-output/` artifact existed before this phase.

### Provided scenario facts

- The target is a small public sample repository.
- No durable, reviewable command-deprecation process currently exists.

## Requirements

- **R-001:** Add a lightweight, repository-local process for documented command
  deprecation.
- **R-002:** State when a deprecation record is required.
- **R-003:** Provide a repository-local Markdown template for creating a record.
- **R-004:** Define review expectations for migration and rollback information.
- **R-005:** Preserve all existing documented commands and behavior.
- **R-006:** Keep the eventual change documentation-only and minimal.
- **R-007:** Keep the process provider-neutral and usable from a local checkout.
- **R-008:** Persist assumptions and unresolved decisions in repository
  artifacts.
- **R-009:** Use available deterministic checks for structure, links,
  consistency, and repository scope where practical.
- **R-010:** Keep the result understandable without chat history.
- **R-011:** Record the relevant documentation change in `CHANGELOG.md` during
  the later implementation phase.
- **R-012:** Do not add application code, executables, dependencies, CI,
  providers, credentials, or network requirements.

## Assumptions

- **A-001:** “Documented command” refers at least to commands in the repository's
  user-facing command reference. Exact trigger coverage remains unresolved.
- **A-002:** “Approval” describes a repository review decision, not a required
  GitHub, issue-tracker, or other provider operation.
- **A-003:** This first slice uses human review; the frozen no-executable and
  no-CI constraints exclude automated policy enforcement.

These assumptions are bounded inputs, not accepted product or architecture
decisions.

## Unknowns

- **U-001:** Which command changes trigger a required deprecation record.
- **U-002:** The lifecycle, transition rules, approval roles, and exception
  authority.
- **U-003:** Record location, identifier, and naming convention.
- **U-004:** Exact mandatory fields and how unavailable migration or rollback
  information must be represented.
- **U-005:** Minimum notice or release interval before removal and the permitted
  exception path.

These unknowns materially change policy behavior, artifact structure, review
criteria, or approval scope. They cannot receive invented defaults.

## Non-goals

- Deprecate, rename, remove, or implement a command.
- Change either documented command's current behavior.
- Build an Axiom CLI, Lingo feature, application, service, or database.
- Add an executable validator, dependency, CI workflow, provider integration,
  credential, or network requirement.
- Add GitHub Issues, pull-request automation, or another provider workflow.
- Design a general release-management system.
- Adopt Axiom or Spec-Kit as a sample-repository dependency.
- Plan, decompose tasks, implement, review, reconcile documentation, or modify
  existing sample repository documentation during this phase.
