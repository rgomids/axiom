---
name: axiom-project
description: Operate on Axiom Projects through one domain-oriented Runtime surface backed by Lingo.
---

# Axiom Project

To inspect supported operations and accepted arguments before execution, run
`axiom --json skill inspect axiom-project` and report its `skill` payload.
Inspection stops there: do not collect inputs or execute an operation. The binary
owns argument names, requirements, accepted forms, and executable command
metadata; do not maintain a second argument registry in this skill.

Supported domain operations are `configure`, `list`, and `show`.

## Operation resolution

When the user supplies an explicit supported operation, use it exactly and route
directly to the corresponding Lingo command. Do not classify or reinterpret an
explicit operation semantically.

When no operation is explicit, resolve intent only among `configure`, `list`,
and `show`:

- choose `configure` only when the user intends to create or change Project
  configuration or repository/provider associations;
- choose `list` when the user wants the configured Project collection;
- choose `show` when the user wants one configured Project or its associations.

If the intent is unknown or materially ambiguous, ask one bounded clarification
instead of guessing. Never turn ambiguous intent into a mutating `configure`
operation. Semantic resolution selects only an operation; it never grants
authority, invents selectors, or replaces Lingo validation.

## configure

Collect only missing Project identity/name, repository associations, and explicit
Work Item Provider declaration. Run `axiom --json project configure` with the
supplied facts and report the read-only normalized preview. Publish only after
the user approves that exact preview, repeating the same inputs plus the returned
`--project-id`, `--preview-digest`, and `--authorize-local`.

Use guided `axiom project configure` only when the user wants prompts. Never
infer CWD or Git remotes, write Project files directly, or reproduce
validation/authority rules. Preserve supplied values as data. Forward unknown,
duplicate, or conflicting inputs to Lingo unchanged so deterministic validation
owns the result.

Operation-specific payload: `setup`.

## list

Run `axiom --json project list` and report the returned `projects` collection.
Never enumerate filesystem state, infer Project identity from the Runtime current
working directory, or resolve Projects independently.

Operation-specific payload: `projects`.

## show

Collect a Project slug or ID only when absent. Run
`axiom --json project show --selector <slug-or-id>` and report its structured
Project and repository associations. Preserve the supplied selector exactly.
Never infer a repository from the Runtime current working directory or resolve
identity independently.

Operation-specific payload: `project`.

## Result contract

Canonical completion fields are `status`, `result`, `references`, `next`,
`details`, and `provenance`. Copy them only from Lingo's top-level JSON
object. Omit absent canonical fields. Never derive, synthesize, or reinterpret a
canonical field from an operation-specific payload.

Preserve `setup`, `projects`, and `project` payloads in their original Lingo
semantics and JSON position. Skill text grants no local or Provider authority.
