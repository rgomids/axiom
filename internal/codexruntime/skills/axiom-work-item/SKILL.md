---
name: axiom-work-item
description: Create, run, or inspect Axiom Work Items through one domain-oriented Runtime surface backed by Lingo.
---

# Axiom Work Item

To inspect supported operations and accepted arguments before execution, run
`axiom --json skill inspect axiom-work-item` and report its `skill` payload.
Inspection stops there: do not collect inputs or execute an operation. The binary
owns argument names, requirements, accepted forms, and executable command
metadata; do not maintain a second argument registry in this skill.

Supported domain operations are `create`, `run`, and `status`.

## Operation resolution

When the user supplies an explicit supported operation, use it exactly and route
directly to the corresponding Lingo/application commands. Do not perform semantic
classification for an explicit operation.

When no operation is explicit, resolve intent only among `create`, `run`, and
`status`:

- choose `create` when the user wants to draft/create a new Work Item or select
  an existing provider Work Item into Axiom;
- choose `run` when the user wants to start, resume, advance, reconcile, or
  record an allowed workflow fact for an Execution;
- choose `status` when the user only wants workflow status or Evidence.

If the intent is unknown or materially ambiguous, ask one bounded clarification
or fail safely. Never silently resolve ambiguous intent to `create` or another
mutating path. Semantic resolution selects only the domain operation; it never
grants external/local authority, invents selectors, or changes workflow state.

## create

Start from the user's supplied intent. Do not ask the user to fill a schema or
repeat information already present. Build the canonical Work Item sections from
known facts and ask only for material gaps: problem, desired outcome, context,
scope, constraints, non-goals, and acceptance expectations.

Preserve user wording as user-authored. Mark synthesis or inferred content as
Axiom-authored elaboration. Do not invent environments, actors, deadlines,
technical commitments, or acceptance Evidence.

Collect only missing target facts: Project selector, Project-scoped repository
key, and explicit GitHub `owner/repository`. Do not infer the target from CWD or
Git remotes. Use `story`, `bug`, or `task` according to delivery intent; a
story requires beneficiary and concrete value. Preserve explicitly requested
provider classifications and let Lingo validate them.

Run `axiom --json work-item create` without authority first. Present the
complete returned draft, authorship/assumptions, target, effects, expected
revision, and preview digest. After explicit authority for that exact preview,
repeat the same facts with `--preview-digest <digest> --authorize-external`.
Changed facts require a fresh preview.

For an existing provider Work Item, use an exact selector such as
`github:<owner>/<repository>#<number>` with
`axiom --json work-item select`; after preview review, repeat with its digest and
`--authorize-local`.

Never call GitHub directly, retry an ambiguous create blindly, or treat linkage
as human acceptance.

Operation-specific payloads: `draft`, `selection`, `workItem`.

## run

Collect only missing Project, Project-scoped Repository, exact Work Item, and
applicable Execution selectors. Use
`--project <uuid-or-slug> --repository <key> --work-item github:<owner>/<repository>#<number>`.

Start through `axiom --json workflow start`, passing `--runtime codex` or
`--runtime claude` for the Runtime executing this skill. Resume/advance/status/
evidence/reconcile calls forward the exact `--execution <id>` returned by Lingo
and never pass `--runtime` again.

Follow `currentGate` returned by Lingo. Advance only through
`axiom --json workflow advance` with the required revision, outcome, and
repository-relative artifact reference.

Record planning authority, implementation authority, review start, human
acceptance, or auxiliary conditions only through
`axiom --json workflow fact` with the exact Execution revision, one validated
reference, explicit `--active` value, and `--authorize-local`. Never infer a
fact from GitHub, CI, merge, review, Issue state, or conversation. Human
acceptance additionally requires terminal canonical completion and an explicit
human decision.

Operation-specific payloads: `workflow`, `projection`.

## status

Collect only missing Project, Project-scoped Repository, exact Work Item, and
Execution selectors. Run `axiom --json workflow status` and, when requested,
`axiom --json workflow evidence` with the exact selectors.

This operation is read-only. Never infer selectors from CWD, Git, Provider,
Runtime chat, or global discovery, and never classify workflow state
independently.

Operation-specific payloads: `workflow`, `projection`.

## Shared invariants

Invoke only `axiom --json` for executable behavior. Unknown, duplicate, and
conflicting inputs go to Lingo validation. Runtime skill text is a presentation
and routing surface, not the workflow or domain source of truth.

Canonical completion fields are `status`, `result`, `references`, `next`,
`details`, and `provenance`. Copy them only from Lingo's top-level JSON
object. Omit absent canonical fields and never derive them from an
operation-specific payload. Preserve operation-specific payloads in their
original semantics and JSON position.

Resolving an operation never grants Provider or local mutation authority.
