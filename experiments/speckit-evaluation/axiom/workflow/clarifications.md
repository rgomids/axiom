# Clarifications — Documented Command Deprecation Process

## Status

Human batch consumed from `.experiment/operator-answers.md` without modifying
that source. Original questions remain below. Answers and residual gaps follow
the batch.

## HC-001 — Policy and artifact contract

1. **Q-001 — Trigger scope:** Which changes to a documented command require a
   deprecation record: removal, rename, replacement, behavior change, argument
   or output change, and/or removal from documentation? State any category that
   is explicitly exempt.

2. **Q-002 — Lifecycle and authority:** What lifecycle statuses and transitions
   must a record use, and which provider-neutral roles may propose, review,
   approve, reject, or grant an exception? State whether one person may hold
   multiple roles in this sample repository.

3. **Q-003 — Record convention:** Where must records live, and what identifier
   and filename convention must they use?

4. **Q-004 — Mandatory fields:** Beyond affected command, user impact,
   migration guidance, timing, owner, and rollback treatment, which fields are
   mandatory? How must a contributor represent “no replacement,” “migration not
   applicable,” or “rollback not technically possible” without leaving the
   review gate ambiguous?

5. **Q-005 — Notice and exceptions:** What minimum notice period or release
   interval is required before removal? Which exceptional conditions may shorten
   or bypass it, who approves an exception, and what rationale or evidence must
   the record retain?

## Gate rationale

These answers change behavior, approval authority, durable artifact structure,
mandatory review evidence, and removal timing. Defaults would create policy
rather than safely fill reversible detail. Automated enforcement is not a
question for this slice because the frozen constraints prohibit executables and
CI.

## Recorded answers

- **Q-001 — Partial:** No operator answer directly classifies every trigger
  category. The frozen scenario explicitly covers rename and removal, and the
  approved lifecycle covers marking a command deprecated. This implementation
  uses only those triggers. Behavior, argument, output, and documentation-only
  changes remain unresolved and outside scope.
- **Q-002 — Answered in part:** Lifecycle is
  `Proposed -> Approved -> Deprecated -> Removed`. A `Proposed` record changes
  no command status. A repository maintainer approves transition to `Approved`;
  command owners execute later transitions. Role overlap, proposal authority,
  and rejection mechanics were not specified and are not invented.
- **Q-003 — Answered:** Records live in `docs/deprecations/`, use
  `NNNN-short-name.md`, appear in `docs/deprecations/README.md`, and start from
  `docs/deprecations/template.md`.
- **Q-004 — Answered in part:** Required content is status, affected command,
  replacement or explicit absence, rationale, affected users, migration steps,
  notice or release target, rollback plan, owner, approval evidence, and
  relevant links. Explicit absence is authorized for replacement. No special
  omission rule was supplied for migration or rollback, so both remain required
  and their absence blocks approval.
- **Q-005 — Answered:** Removal normally follows at least one published release
  containing the deprecation notice. Urgent security removal may bypass the
  minimum only when the record states rationale, impact, owner, approval, and
  compensating migration guidance. Within the supplied lifecycle, maintainer
  approval to `Approved` must explicitly cover the exception.

## Supplemental enforcement answer

No executable validator or CI belongs in this slice. Review uses a checklist
and deterministic repository checks for file presence, required headings,
links, whitespace, and scope.

## Clarification outcome

Gate released for the narrow frozen-scenario implementation. Residual gaps are
bounded in `specification.md`; they remain open rather than becoming policy.
