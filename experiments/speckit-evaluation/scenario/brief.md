# Frozen Scenario — Documented Command Deprecation

## Problem

The sample repository documents user-facing commands but has no durable,
reviewable process for deprecating one. A contributor could remove or rename a
command without recording user impact, migration guidance, timing, ownership,
or rollback considerations.

## Desired outcome

Contributors can identify when a command-deprecation record is required, create
one from a repository-local template, and review it against explicit migration
and rollback expectations. The result remains understandable without chat
history or a specific work-tracking provider.

## Initial user input

> Add a lightweight, repository-local process for deprecating documented
> commands. Contributors need to know when a deprecation record is required,
> how to create it, and how reviewers confirm migration and rollback
> information before approval. Keep existing commands unchanged.

## Constraints

- Preserve all existing documented commands and behavior.
- Make the smallest coherent documentation-only change.
- Add no application code, executable validator, dependency, CI workflow,
  provider integration, credential, or network requirement.
- Keep the process provider-neutral and usable from a local checkout.
- Use Markdown artifacts committed with the repository.
- Leave assumptions and unresolved human decisions explicit.
- Validate structure, links, consistency, and repository scope with available
  deterministic commands where practical.

## Known context

- This is a small public sample repository.
- `README.md` points contributors to `docs/commands.md`.
- `docs/commands.md` documents two supported commands: `example validate` and
  `example package`.
- `CHANGELOG.md` records relevant repository changes.
- No command is currently deprecated.
- The repository contains no application runtime or dependency manifest.

## Unknowns

- Exact deprecation lifecycle and approval authority.
- Record location and naming convention.
- Mandatory record fields.
- Minimum notice before removal and any exception path.
- Whether the first slice needs automated enforcement.

## Non-goals

- Deprecate, rename, remove, or implement any command.
- Build a CLI, Lingo feature, application, service, or database.
- Add GitHub Issues, pull-request automation, CI, or another provider workflow.
- Design a general release-management system.
- Adopt Axiom or Spec-Kit as a dependency of the sample repository.
