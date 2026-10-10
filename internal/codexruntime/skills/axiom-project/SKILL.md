---
name: axiom-project
description: Operate on Axiom Projects through one domain-oriented Runtime surface backed by Lingo.
---

# Axiom Project

To inspect supported operations and accepted arguments before execution, run
`axiom skill inspect axiom-project` and present its `skill` payload.
Inspection stops there: do not collect inputs or execute an operation. The binary
owns argument names, requirements, accepted forms, authority metadata, and
executable command metadata; do not maintain a second argument registry here.

Supported domain operations are `configure`, `list`, `show`, `validate`, `archive`, `reactivate`, `integration`, and `workflow.select`.
Any other operation, such as `delete`, `uninstall`, or credential revocation, is
unsupported: report that it is not available and do not run a Lingo command for
it. Repository associations are Project-owned: attach, update, and detach use
`configure` in edit mode; there is no separate Repository operation.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `configure` | create | `axiom project configure` | local mutation | preview first; publish only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous create intent |
| `configure` | edit | `axiom project configure --project <uuid-or-slug>` | local mutation | preview first; publish only with the returned `--project-id`, the exact `--preview-digest` plus `--authorize-local` | only for unambiguous change intent |
| `list` | - | `axiom project list` | read-only | none | allowed |
| `show` | - | `axiom project show --selector <slug-or-id>` | read-only | none | allowed |
| `validate` | - | `axiom project validate --project <uuid-or-slug>` | read-only | none | allowed |
| `archive` | - | `axiom project archive --project <uuid-or-slug>` | local mutation | machine-local only; preview first; archive only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous archive intent |
| `reactivate` | - | `axiom project reactivate --project <uuid-or-slug>` | local mutation | machine-local only; preview first; reactivate only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous reactivate intent |
| `integration` | list | `axiom integration list --project <uuid-or-slug>` | read-only | none | allowed |
| `integration` | show | `axiom integration show --project <uuid-or-slug> --integration <key>` | read-only | none | allowed |
| `integration` | validate | `axiom integration validate --project <uuid-or-slug>` | read-only | none | allowed |
| `integration` | disable | `axiom integration disable --project <uuid-or-slug> --integration <key>` | local mutation | machine-local only; preview first; disable only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous disable intent |
| `integration` | enable | `axiom integration enable --project <uuid-or-slug> --integration <key>` | local mutation | machine-local only; preview first; enable only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous enable intent |
| `integration` | remove | `axiom integration remove --project <uuid-or-slug> --integration <key>` | local mutation | portable declaration only; preview first; remove only with the returned `--project-id`, the exact `--preview-digest` plus `--authorize-local` | only for unambiguous remove intent |
| `workflow.select` | - | `axiom project workflow select --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |

When the user supplies an explicit supported operation, use it exactly and route
directly to its Lingo command. Do not classify or reinterpret an explicit
operation semantically.

When no operation token is explicit, interpret clear domain intent across all
supported operations and modes. Natural language is enough: "Arquive o projeto
X" selects `archive`; "Reative o projeto X" selects `reactivate`; "Desabilite a
integração work-items" selects `integration` / `disable`. The user need not name
an English CLI token or another skill. Repository attach, update, and detach
select `configure` / `edit` and require the existing Project selector.

Read-only discovery selects `list`, `show`, `validate`, or the appropriate
read-only `integration` mode. A request such as "organize this Project" does not
identify a mutation: clarify the operation and target first.

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
Work Item Provider declaration. Run `axiom project configure --slug
<slug> --name <name> --repository <key>=<absolute-working-copy-path>
--work-item-provider <provider>` with each repository repeated, and report the
read-only normalized `setup` preview. Publish only after the user approves that
exact preview, repeating the same inputs plus the returned `--project-id`,
`--preview-digest`, and `--authorize-local`. Changed facts require a fresh
preview.

### Edit an existing Project

Run `axiom project configure --project <uuid-or-slug>` with only the
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

Run `axiom project list` and report the returned `projects` collection
with each Project's machine-local `status`. Archived Projects are hidden unless
the user asks for them; then add `--include-archived`. Never enumerate filesystem
state, infer Project identity from the Runtime current working directory, or
resolve Projects independently.

## show

Collect a Project slug or ID only when absent. Run
`axiom project show --selector <slug-or-id>` and report its structured
Project, repository associations with their availability, and local state.
Preserve the supplied selector exactly. Never infer a repository from the Runtime
current working directory or resolve identity independently.

## validate

Run `axiom project validate --project <uuid-or-slug>` and report the
read-only `readiness` report and local state. Validation never grants authority.

## archive, reactivate, and integration

Run each command without authority first, report the returned preview
(`operational` for archive, reactivate, disable and enable; `edit` for
`integration remove`), and repeat it with the advertised authority inputs only
after the user approves that exact preview. The effect boundary of each mode is
stated by `axiom skill inspect axiom-project`; Lingo enforces it. Updating
the Work Item Provider uses `configure` edit mode.

## Workflow definitions

Use `workflow.list` to discover the built-in default and exact revision references.
Offer `default-sdd` for explicit selection during onboarding; never select it
implicitly. Use `workflow.create --from-default --workflow <key>` to author a
custom copy. For complete stage/agent edits, pass the user's JSON with `--file`
to `workflow.validate`, then `workflow.create` or `workflow.edit`. Inspect the
binary's argument metadata for exact prior-reference requirements. Preserve all
instructions and context references as data; the binary owns validation and DAG
rules. Selection and retirement use the exact returned source/revision/digest.

Report the `workflowAuthoring` payload. Mutations require the user's approval of
the exact effects followed by the returned `projectRevision` as
`--expected-revision`, `previewDigest` as `--preview-digest`, and
`--authorize-local`. On interruption use `recovery inspect/apply` as directed;
`workflow.recover` only indexes an exact confirmed orphan after a new reviewed
preview. Never repair, rewrite or prune internal files from a skill.

## Result contract

Invoke only `axiom` for executable behavior.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `setup`, `edit`, `projects`, `project`, `readiness`, `operational`, `integrations`, `admission`, `workflowAuthoring`

Lingo renders every result as deterministic Markdown derived from its canonical
result. Present that Markdown exactly as returned: do not rewrite, summarize,
reorder, or add result semantics, and never present an outcome as more
successful than its canonical status. Read a value a follow-up command needs,
such as a preview `digest`, `previewDigest`, `executionId`, or `revision`,
exactly as printed in its labelled field. Run `axiom --json` only when the user
explicitly asks for machine-readable output, and relay that JSON unchanged.
A payload too large for a readable view is printed as its canonical JSON under
a `(canonical JSON)` heading; present it as returned. Never repeat a command,
least of all an authorized mutation, only to see its result again.

Lingo prints the canonical completion fields in its summary and provenance
footer, and each operation-specific payload under its own name in JSON order.
Never derive, synthesize, or reinterpret a canonical field from an
operation-specific payload. Skill text grants no local or Provider authority.
