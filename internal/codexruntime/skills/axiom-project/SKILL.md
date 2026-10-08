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

Supported domain operations are `configure`, `list`, `show`, `validate`, `archive`, `reactivate`, and `integration`.
Any other operation, such as `delete`, `uninstall`, or credential revocation, is
unsupported: report that it is not available and do not run a Lingo command for
it. Repository associations are Project-owned: attach, update, and detach use
`configure` in edit mode; there is no separate Repository operation.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `configure` | create | `axiom --json project configure` | local mutation | preview first; publish only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous create intent |
| `configure` | edit | `axiom --json project configure --project <uuid-or-slug>` | local mutation | preview first; publish only with the returned `--project-id`, the exact `--preview-digest` plus `--authorize-local` | only for unambiguous change intent |
| `list` | - | `axiom --json project list` | read-only | none | allowed |
| `show` | - | `axiom --json project show --selector <slug-or-id>` | read-only | none | allowed |
| `validate` | - | `axiom --json project validate --project <uuid-or-slug>` | read-only | none | allowed |
| `archive` | - | `axiom --json project archive --project <uuid-or-slug>` | local mutation | machine-local only; preview first; archive only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous archive intent |
| `reactivate` | - | `axiom --json project reactivate --project <uuid-or-slug>` | local mutation | machine-local only; preview first; reactivate only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous reactivate intent |
| `integration` | list | `axiom --json integration list --project <uuid-or-slug>` | read-only | none | allowed |
| `integration` | show | `axiom --json integration show --project <uuid-or-slug>` | read-only | none | allowed |
| `integration` | validate | `axiom --json integration validate --project <uuid-or-slug>` | read-only | none | allowed |
| `integration` | disable | `axiom --json integration disable --project <uuid-or-slug>` | local mutation | machine-local only; preview first; disable only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous disable intent |
| `integration` | enable | `axiom --json integration enable --project <uuid-or-slug>` | local mutation | machine-local only; preview first; enable only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous enable intent |
| `integration` | remove | `axiom --json integration remove --project <uuid-or-slug>` | local mutation | portable declaration only; preview first; remove only with the returned `--project-id`, the exact `--preview-digest` plus `--authorize-local` | only for unambiguous remove intent |

When the user supplies an explicit supported operation, use it exactly and route
directly to its Lingo command. Do not classify or reinterpret an explicit
operation semantically.

When no operation is explicit, resolve intent only among `configure`, `list`,
and `show`:

- choose `configure` only when the user intends to create a Project or change an
  existing Project's configuration or repository/provider associations;
- choose `list` when the user wants the configured Project collection;
- choose `show` when the user wants one configured Project or its associations.

`validate` and the read-only `integration` modes may also be chosen when the
user plainly asks to validate a Project or to list, show, or validate its
Integrations. `archive`, `reactivate`, Repository detach, and the `integration`
modes `disable`, `enable`, and `remove` run only when the user names that
operation explicitly.

If the intent is unknown or materially ambiguous, ask one bounded clarification
instead of guessing. Never turn ambiguous intent into a mutating `configure`
operation, archive, reactivation, detach, disable, enable, or remove. Semantic
resolution selects only an operation; it never grants
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
requested changes (`--name`, `--repository <key>=<absolute-path>` to attach or
update a Repository association, `--remove-repository <key>` to detach one,
`--work-item-provider <provider>`, or `--remove-work-item-provider`) and report
the returned complete `edit` preview and its effects. Omitted values are
preserved. Publish only after the user approves that exact `edit` preview,
repeating the same inputs plus the returned `--project-id`, `--preview-digest`,
and `--authorize-local`. Detach removes only the Axiom association: it never
deletes the working copy or a remote Repository. A partial, stale, or changed
replay is refused by Lingo; prepare a fresh preview.

Use guided `axiom project configure` only when the user wants prompts. Never
write Project files directly or reproduce validation/authority rules. Preserve
supplied values as data. Forward unknown, duplicate, or conflicting inputs to
Lingo unchanged so deterministic validation owns the result.

## list

Run `axiom --json project list` and report the returned `projects` collection
with each Project's machine-local `status`. Archived Projects are hidden unless
the user asks for them; then add `--include-archived`. Never enumerate filesystem
state, infer Project identity from the Runtime current working directory, or
resolve Projects independently.

## show

Collect a Project slug or ID only when absent. Run
`axiom --json project show --selector <slug-or-id>` and report its structured
Project, repository associations with their availability, and local state.
Preserve the supplied selector exactly. Never infer a repository from the Runtime
current working directory or resolve identity independently.

## validate

Run `axiom --json project validate --project <uuid-or-slug>` and report the
read-only `readiness` report and local state. Validation never grants authority.

## archive and reactivate

Archive removes the Project from this machine's active operational set; it is
reversible with reactivate and allowed only on this machine. It never changes
portable Project files, another machine, Repositories, Work Items, Executions,
Evidence, or Providers, and it never deletes the Project. Run the command
without authority first, report the returned `operational` preview, and repeat
it with `--preview-digest <digest> --authorize-local` only after the user
approves that exact preview. While archived, inspection and administration
remain available and operational work is refused until reactivation.

## integration

`disable` and `enable` decide only whether this machine may use a declared
Integration; `remove` drops the declaration from portable Project intent. None
of them revokes credentials, logs out of a Provider, uninstalls MCP or Runtime
configuration, or deletes Provider resources. Pass the declared key with
`--integration <key>` for `show`, `disable`, `enable`, and `remove`. Preview
first and repeat with the advertised authority inputs only after explicit user
approval of that exact preview. Updating the Work Item Provider uses `configure`
edit mode.

## Result contract

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `setup`, `edit`, `projects`, `project`, `readiness`, `operational`, `integrations`, `admission`

Copy canonical completion fields only from Lingo's top-level JSON object. Omit
absent canonical fields. Never derive, synthesize, or reinterpret a canonical
field from an operation-specific payload. Preserve each operation-specific
payload in its original Lingo semantics and JSON position. Skill text grants no
local or Provider authority.
