---
name: axiom-project
description: Operate on Axiom Projects through one domain-oriented Runtime surface backed by Lingo.
---

# Axiom Project

To inspect supported operations and accepted arguments before execution, run
`axiom --json skill inspect axiom-project` and report its `skill` payload.
Inspection stops there: do not collect inputs or execute an operation. The binary
owns argument names, requirements, accepted forms, authority metadata, and
executable command metadata; do not maintain a second argument registry here.

Supported domain operations are `configure`, `list`, and `show`. Any other
operation, such as `update`, `remove`, or `delete`, is unsupported: report that
it is not available and do not run a Lingo command for it.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `configure` | create | `axiom --json project configure` | local mutation | preview first; publish only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous create intent |
| `configure` | edit | `axiom --json project configure --project <uuid-or-slug>` | preview only | no publication; never pass `--project-id`, `--preview-digest`, or `--authorize-local` | only for unambiguous change intent |
| `list` | - | `axiom --json project list` | read-only | none | allowed |
| `show` | - | `axiom --json project show --selector <slug-or-id>` | read-only | none | allowed |

When the user supplies an explicit supported operation, use it exactly and route
directly to its Lingo command. Do not classify or reinterpret an explicit
operation semantically.

When no operation is explicit, resolve intent only among `configure`, `list`,
and `show`:

- choose `configure` only when the user intends to create a Project or change an
  existing Project's configuration or repository/provider associations;
- choose `list` when the user wants the configured Project collection;
- choose `show` when the user wants one configured Project or its associations.

If the intent is unknown or materially ambiguous, ask one bounded clarification
instead of guessing. Never turn ambiguous intent into a mutating `configure`
operation. Semantic resolution selects only an operation; it never grants
authority, supplies `--authorize-local`, invents selectors, or replaces Lingo
validation.

## configure

The mode is decided only by an explicit existing-Project selector: with
`--project <uuid-or-slug>` Lingo edits that Project; without it Lingo creates
one. Never infer the mode or the Project from CWD, Git remotes, or chat history.

### Create a Project

Collect only missing Project identity/name, repository associations, and explicit
Work Item Provider declaration. Run `axiom --json project configure --slug
<slug> --name <name> --repository <key>=<absolute-working-copy-path>
--work-item-provider <provider>` with each repository repeated, and report the
read-only normalized `setup` preview. Publish only after the user approves that
exact preview, repeating the same inputs plus the returned `--project-id`,
`--preview-digest`, and `--authorize-local`. Changed facts require a fresh
preview.

### Edit an existing Project

Run `axiom --json project configure --project <uuid-or-slug>` with only the
requested changes (`--name`, `--repository <key>=<absolute-path>`,
`--remove-repository <key>`, `--work-item-provider <provider>`, or
`--remove-work-item-provider`) and report the returned `edit` preview. Project
edit is preview-only: Lingo publishes nothing in this mode and rejects
`--project-id`, `--preview-digest`, and `--authorize-local` with
`unsupported_edit_authority`. Do not repeat an edit with those inputs, do not
ask for authority to apply it, and never report the change as applied.

Use guided `axiom project configure` only when the user wants prompts. Never
write Project files directly or reproduce validation/authority rules. Preserve
supplied values as data. Forward unknown, duplicate, or conflicting inputs to
Lingo unchanged so deterministic validation owns the result.

## list

Run `axiom --json project list` and report the returned `projects` collection.
Never enumerate filesystem state, infer Project identity from the Runtime current
working directory, or resolve Projects independently.

## show

Collect a Project slug or ID only when absent. Run
`axiom --json project show --selector <slug-or-id>` and report its structured
Project and repository associations. Preserve the supplied selector exactly.
Never infer a repository from the Runtime current working directory or resolve
identity independently.

## Result contract

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `setup`, `edit`, `projects`, `project`

Copy canonical completion fields only from Lingo's top-level JSON object. Omit
absent canonical fields. Never derive, synthesize, or reinterpret a canonical
field from an operation-specific payload. Preserve each operation-specific
payload in its original Lingo semantics and JSON position. Skill text grants no
local or Provider authority.
