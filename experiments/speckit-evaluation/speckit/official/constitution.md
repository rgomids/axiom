<!--
Sync Impact Report
- Version change: template scaffold -> 1.0.0
- Modified principles: placeholder principles -> seven initial project principles
- Added sections: Documentation Constraints; Review and Validation Workflow
- Removed sections: none
- Follow-up TODOs: none
-->
# Example Command Repository Constitution

## Core Principles

### I. Documentation-Only Scope
Changes governed by this constitution MUST consist only of documentation and supporting
repository-local Markdown artifacts. They MUST NOT introduce application behavior. This keeps the
sample focused on durable contributor guidance rather than an implementation.

### II. Existing Command Behavior Is Invariant
Existing documented commands, names, interfaces, and behavior MUST remain unchanged unless a
separate command-lifecycle change receives explicit human approval. A deprecation process may
describe future change but MUST NOT itself deprecate, rename, remove, or implement a command.

### III. Durable Repository-Local Markdown
Process, records, templates, decisions, migration guidance, rollback guidance, and validation
evidence MUST be stored as reviewable Markdown in the repository. Artifacts MUST remain
understandable without chat history or external work-tracking context.

### IV. Provider-Neutral Offline Use
Contributors MUST be able to create, review, and validate required documentation from a local
checkout without a specific hosting, issue-tracking, or automation provider. The process MUST NOT
require credentials or network access.

### V. Human Approval Governs Command Lifecycles
Any decision to deprecate, rename, remove, replace, or otherwise change the lifecycle of a command
MUST identify an accountable human approval authority and MUST receive that approval before the
change takes effect. Unresolved authority or lifecycle choices MUST remain explicit, not inferred.

### VI. Deterministic Validation Where Practical
Reviews MUST use deterministic, repeatable commands available in the repository environment where
practical to validate Markdown structure, links, consistency, and repository scope. When a check
cannot be made deterministic, the review record MUST state the manual check and its result.

### VII. No New Operational Requirements
Documentation changes MUST NOT add application code, executable validators, dependencies, CI
workflows, provider integrations, credentials, or network requirements. This prevents a lightweight
documentation process from creating an operational system or hidden maintenance burden.

## Documentation Constraints

- Changes MUST be the smallest coherent set that satisfies the documented need.
- Assumptions, unresolved human decisions, non-goals, migration expectations, and rollback
  expectations MUST be explicit.
- Existing command documentation and repository history MUST remain internally consistent.
- Documentation lifecycle treatment MUST include the repository changelog when a relevant change
  is completed.

## Review and Validation Workflow

1. Define the documentation problem, outcome, constraints, assumptions, and non-goals.
2. Resolve material human decisions before planning a command-lifecycle change.
3. Review proposed records for user impact, migration guidance, timing, ownership, approval, and
   rollback considerations.
4. Run deterministic offline checks that are practical for the repository and record outcomes.
5. Review the final diff to confirm documentation-only scope and unchanged command behavior.

## Governance

This constitution governs specification, planning, implementation, and review for this repository.
Amendments MUST be documented, state their rationale and migration impact, receive explicit human
approval, and update the version and amendment date. Versions follow semantic versioning: MAJOR for
incompatible governance changes, MINOR for added or materially expanded principles, and PATCH for
non-semantic clarification. Every review MUST verify constitution compliance; any exception MUST be
explicitly documented and approved by a human before work proceeds.

**Version**: 1.0.0 | **Ratified**: 2026-08-11 | **Last Amended**: 2026-08-11
