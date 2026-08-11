# Specification — Documented Command Deprecation Process

## Status

Clarified for a narrow documentation-only implementation. Human answers from
`.experiment/operator-answers.md` are incorporated below. Residual unknowns are
explicit and excluded from this slice rather than silently decided.

## Intent

Define observable behavior for a small, repository-local process governing the
future deprecation of documented commands while preserving all current commands
and behavior.

## Scope

The slice covers documentation that:

- states when a record is required for the frozen scenario;
- supplies a Markdown record template and index;
- defines lifecycle, content, notice, exception, and review expectations;
- remains usable from a local checkout without a provider.

For this slice, a record is required before a documented command is intentionally
renamed, marked deprecated, or removed. Whether other behavior, argument,
output, or documentation-only changes require a record remains unresolved and
outside scope because the operator batch did not answer Q-001 directly.

## Actors

- **Contributor:** creates a `Proposed` record.
- **Repository maintainer:** approves transition from `Proposed` to `Approved`.
- **Command owner:** executes later transitions to `Deprecated` and `Removed`.
- **Reviewer:** checks applicability and completeness; this may be a maintainer
  acting in the reviewer role.

No separation-of-duties rule or restriction on one person holding multiple
roles was provided.

## Scenarios

### S-001 — Covered change

Given a contributor intends to rename, deprecate, or remove a command documented
in `docs/commands.md`, when work is proposed, then a record is created in
`docs/deprecations/` with status `Proposed` before approval.

### S-002 — Proposed record changes no command status

Given a `Proposed` record exists, when current command documentation is read,
then the command remains supported; the record alone does not change command
status before a command owner performs a later lifecycle transition.

### S-003 — Complete record review

Given a required record exists, when a reviewer evaluates it, then the reviewer
can determine status, affected command, replacement or explicit absence,
rationale, affected users, migration steps, notice or release target, rollback
plan, owner, approval evidence, and relevant links without chat or a provider.

### S-004 — Incomplete migration or rollback treatment

Given a required record lacks migration steps or a rollback plan, when reviewed,
then approval is blocked. The operator answers provide no special allowance for
omitting either required field.

### S-005 — Normal removal

Given a record reaches `Deprecated`, when removal is proposed, then at least one
published release containing the deprecation notice precedes transition to
`Removed`.

### S-006 — Urgent security removal

Given an urgent security condition, when the ordinary notice minimum cannot be
met, then the exception is permitted only if the record states rationale,
impact, owner, repository-maintainer approval, and compensating migration
guidance.

### S-007 — Local, provider-neutral use

Given only a local repository checkout, when a contributor creates or reviews a
record, then the process works without credentials, network access, or a named
work-tracking or source-control provider.

### S-008 — Process introduced without a deprecation

Given this process is added, when current command documentation is inspected,
then `example validate` and `example package` remain supported, unchanged, and
not marked deprecated.

## Functional requirements

- **FR-001 — Trigger:** A record MUST be created before a documented command is
  intentionally renamed, marked deprecated, or removed.
- **FR-002 — Creation:** A contributor MUST create the record from
  `docs/deprecations/template.md`.
- **FR-003 — Convention:** Records MUST live in `docs/deprecations/` and use
  `NNNN-short-name.md`; `docs/deprecations/README.md` MUST index records.
- **FR-004 — Required content:** Every record MUST include status, affected
  command, replacement or explicit absence, rationale, affected users,
  migration steps, notice or release target, rollback plan, owner, approval
  evidence, and relevant links.
- **FR-005 — Lifecycle:** Records MUST use
  `Proposed -> Approved -> Deprecated -> Removed`.
- **FR-006 — Authority:** A repository maintainer MUST approve transition to
  `Approved`; command owners MUST execute later transitions.
- **FR-007 — Proposed semantics:** A `Proposed` record MUST NOT change command
  status.
- **FR-008 — Review gate:** Missing required content, including migration steps
  or rollback plan, MUST block approval.
- **FR-009 — Normal notice:** Removal MUST normally follow at least one
  published release containing the deprecation notice.
- **FR-010 — Security exception:** An urgent security removal MAY bypass the
  normal notice minimum only when the record states rationale, impact, owner,
  repository-maintainer approval, and compensating migration guidance.
- **FR-011 — Existing behavior:** The process MUST NOT deprecate, rename,
  remove, reimplement, or otherwise alter `example validate` or
  `example package` in this slice.
- **FR-012 — Local durability:** Process, template, index, and review
  expectations MUST be understandable from committed Markdown without chat.
- **FR-013 — Provider neutrality:** No step MUST require GitHub, an issue
  tracker, a documentation provider, credentials, or network access.
- **FR-014 — Manual enforcement:** The slice MUST NOT add executable policy
  enforcement or CI. Review uses a checklist plus deterministic repository
  checks for file presence, required headings, links, whitespace, and scope.
- **FR-015 — Change record:** The implementation MUST add a relevant
  `CHANGELOG.md` entry.

## Invariants

- Both existing commands remain supported and retain current descriptions.
- A `Proposed` record alone cannot change command status.
- A required record cannot be approved with missing mandatory migration or
  rollback treatment.
- Absence of a provider reference cannot make a record invalid.
- Assumptions and residual unknowns cannot be presented as approved policy.
- The slice adds no application code, executable, dependency, CI, provider,
  credential, or network requirement.

## Residual unknowns and bounded treatment

- **U-001:** Behavior, argument, output, and documentation-only changes that do
  not rename, deprecate, or remove a command have no approved trigger rule.
  They are outside this slice.
- **U-002:** Whether one person may hold multiple lifecycle roles is unanswered.
  Documentation names responsibilities without creating a separation-of-duties
  rule.
- **U-003:** No approved representation exists for omitted migration steps or
  an impossible rollback. Both fields remain mandatory; review blocks rather
  than inventing an exception.

## Non-functional constraints

- Documentation-only and smallest coherent repository change.
- Markdown committed with the repository.
- Provider-neutral, offline-capable, and free of credentials.
- Deterministically checkable file presence, headings, links, whitespace, and
  repository scope.
- No general release-management design.

## Acceptance criteria

- **AC-001:** Documentation states that renaming, marking deprecated, or
  removing a documented command requires a record and does not claim an answer
  for other change categories.
- **AC-002:** A contributor can locate the index and template, then create a
  record using `docs/deprecations/NNNN-short-name.md`.
- **AC-003:** Template contains every required content heading from FR-004.
- **AC-004:** Reviewer checklist blocks approval when required migration or
  rollback content is missing.
- **AC-005:** Lifecycle and authority match
  `Proposed -> Approved -> Deprecated -> Removed`, maintainer approval, and
  command-owner execution of later transitions.
- **AC-006:** Normal removal requires one published release with notice; urgent
  security exception content matches FR-010.
- **AC-007:** `example validate` and `example package` retain their original
  names and descriptions and remain supported.
- **AC-008:** Process, index, and template are usable from a local checkout
  without chat, provider access, credentials, or network access.
- **AC-009:** All new internal Markdown links resolve; required headings exist;
  Markdown has no trailing whitespace; `CHANGELOG.md` records the change.
- **AC-010:** Repository scope contains no new application code, executable,
  dependency manifest, CI workflow, provider integration, or credential.

## Explicit non-goals

- Perform an actual command deprecation.
- Change current command names, descriptions, or behavior.
- Decide trigger policy for unrelated behavior, argument, output, or
  documentation-only changes.
- Automate enforcement.
- Choose or access a provider workflow.
- Design release management beyond command deprecation.
- Change Axiom's workflow, architecture, product documentation, or runtime.

## Constitution check

- **Intent before implementation:** Existing intake, refined behavior,
  constraints, non-goals, scenarios, and acceptance evidence precede planning.
- **Smallest applicable SDD flow:** Durable intake and clarification feed this
  clarified specification; later phases consume it in order.
- **Classification:** Human answers, derived narrow scope, and residual unknowns
  remain distinct.
- **Human approval:** Operator answers control lifecycle, authority, artifact
  convention, required content, notice, exception, and enforcement.
- **Durability and traceability:** Repository artifacts retain source,
  clarification mapping, decisions, implementation, and evidence.
- **Determinism:** Acceptance includes reproducible checks for objective
  documentation invariants; semantic review remains explicit.
- **Safety and scope:** Existing command behavior and repository boundaries are
  preserved; no permission, provider, dependency, execution, or network scope
  expands.
- **Architecture:** No Axiom architecture boundary changes. ADR threshold is
  evaluated separately before implementation.
