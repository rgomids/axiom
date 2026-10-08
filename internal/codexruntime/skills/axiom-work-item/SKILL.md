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

Supported domain operations are `create`, `run`, `status`, `list`, `show`, `update`, `comment`, `close`, and `reopen`.
Any other operation, such as `delete` or Execution `cancel`, is unsupported:
report that it is not available and do not run a Lingo command for it. Work
Items are closed or reopened, never deleted; sequential Executions have no
cancellation contract, and Execution history and Evidence are never edited.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `create` | new | `axiom --json work-item create` | external mutation | preview first; create only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous create intent |
| `create` | existing | `axiom --json work-item select` | local mutation | preview first; link only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous select intent |
| `run` | transition | `axiom --json workflow start`, `axiom --json workflow advance`, `axiom --json workflow resume` | local mutation | `workflow start` previews the Runtime/Profile resolution first and starts only with the exact `--runtime-preview`; Lingo enforces the exact Execution revision and gate rules | only for unambiguous run intent |
| `run` | fact | `axiom --json workflow fact` | local mutation | only with `--authorize-local` and the exact Execution revision | only for unambiguous run intent |
| `run` | reconcile | `axiom --json workflow reconcile` | external mutation | preview first; publish only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous run intent |
| `status` | - | `axiom --json workflow status`, `axiom --json workflow evidence` | read-only | none | allowed |
| `status` | list | `axiom --json workflow list --project <uuid-or-slug>` | read-only | none | allowed |
| `list` | - | `axiom --json work-item list --project <uuid-or-slug>` | read-only | none | allowed |
| `show` | - | `axiom --json work-item show --project <uuid-or-slug>` | read-only | none | allowed |
| `update` | - | `axiom --json work-item update --project <uuid-or-slug>` | external mutation | preview first; update only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous update intent |
| `comment` | - | `axiom --json work-item comment --project <uuid-or-slug>` | external mutation | preview first; comment only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous comment intent |
| `close` | - | `axiom --json work-item close --project <uuid-or-slug>` | external mutation | preview first; close only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous close intent |
| `reopen` | - | `axiom --json work-item reopen --project <uuid-or-slug>` | external mutation | preview first; reopen only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous reopen intent |

When the user supplies an explicit supported operation, use it exactly and route
directly to its Lingo commands. Do not perform semantic classification for an
explicit operation.

When no operation is explicit, resolve intent only among `create`, `run`, and
`status`:

- choose `create` when the user wants to draft/create a new Work Item or select
  an existing provider Work Item into Axiom;
- choose `run` when the user wants to start, resume, advance, reconcile, or
  record an allowed workflow fact for an Execution;
- choose `status` when the user only wants workflow status or Evidence.

`list` (linked Work Items), `show`, and `status` mode `list` (Executions of a
Project, when no Execution identity is known) may also be chosen when the user
plainly asks to discover or inspect them. `update`, `comment`, `close`, and
`reopen` run only when the user names that operation explicitly; ambiguous intent
never selects them.

If the intent is unknown or materially ambiguous, ask one bounded clarification
or fail safely. Never silently resolve ambiguous intent to `create`, `run`, or
another mutating path. Semantic resolution selects only the domain operation; it
never grants external/local authority, supplies `--authorize-external` or
`--authorize-local`, invents selectors, or changes workflow state.

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

## run

Require an explicit exact Work Item on every invocation. Collect only missing
Project-scoped Repository and applicable Execution selectors. Project is optional
when Lingo resolves the effective Project through explicit > session > default
selection; preserve any supplied `--session <id>`. Never infer Project, Repository,
Work Item, or Execution from CWD, Git, conversation, or recent activity. Lingo is
the authoritative resolver and validator; stop on missing or conflicting inputs.
Use `--work-item github:<owner>/<repository>#<number>`, with
`--project <uuid-or-slug>` and `--repository <key>` when supplied or required.

Starting is a reviewed two-step protocol; the first call only previews:

```text
axiom --json workflow start <selectors> --role <role> --complexity <complexity> --capabilities <capabilities> --runtime <runtime>
axiom --json workflow start <selectors> --role <role> --complexity <complexity> --capabilities <capabilities> --runtime <runtime> --runtime-preview <previewDigest>
axiom --json workflow status <selectors> --execution <executionId>
```

Collect role, complexity, and comma-separated capabilities from the user or the
approved Plan; never invent them. `<runtime>` is `codex` or `claude`, naming the
Runtime executing this skill, which narrows the Project policy and never widens
it. Never default to Codex or infer a Runtime from installed executables. Lingo
observes Runtimes itself: never supply, invent, or claim an observation, version,
or proven capability. Preserve and report the first call's `executionTarget`,
`runtimeResolution`, and `previewDigest`. `executionTarget` contains `projectId`, `projectSource`,
`repositoryKey`, `provider`, `resource`, `externalId`, and `workItem`. The digest
binds this full target, including Project source, and the Runtime/Profile policy.
A blocker stops the workflow. Otherwise show the exact target, selected Runtime,
Model Profile, model, and revisions and wait for an explicit decision; only then
repeat the identical selectors, `--session`, and policy inputs with
`--runtime-preview <previewDigest>`. Changing a selector or Project source,
including replacing an omitted Project with an explicit one, needs a fresh
preview even when the resolved Project UUID is the same.
Never edit a digest or reuse one after a blocker: `stale_preview` requires a
fresh preview. The Execution's Runtime is the selected
`runtimeResolution.choice.runtimeId`. `workflow start` omits `--execution`.
After start, use the persisted exact Project UUID, Repository key, Work Item,
and `--execution <id>` returned by Lingo for resume, advance, status, evidence,
fact, and reconcile calls. Never resolve a different effective Project for an
existing Execution, and never pass `--runtime`, policy inputs, or
`--runtime-preview` again. In the status line above, `<selectors>` means these
persisted selectors, not the original optional Project context.

Reconcile through `axiom --json workflow reconcile` with the exact Execution
revision; present the returned preview and repeat with its
`--preview-digest <digest> --authorize-external` only after explicit authority
for that exact preview.

After the reviewed start, follow Lingo's `workflow.gateAction` and
`workflow.gateCommand` argument array using the exact returned selectors and
revision. Do not ask the user to name an internal gate or say a magic phrase.

- An action with `automatic: true` runs `workflow advance --automatic` during
  the authorized run without another conversational confirmation. Intake is
  currently the only gate Lingo can evaluate automatically. Do not combine
  automatic mode with gate, outcome, reference or next inputs.
- Other advance actions require actual authorized work and its observed `pass`
  or `fail` result with applicable validated Evidence. Fill command placeholders
  from observations; a file digest alone does not prove correctness.
- Fact actions identify explicit authority or a condition to resolve. Present
  the missing decision/input and record it only with validated references and
  explicit local authority. A returned `--authorize-local` grants no authority.
  Never clear conditions merely to make the workflow progress.
- Resume uses the exact committed revision. Re-read status after each result;
  stop and report denied/failed operations without inferring progression.

Gate policies remain in Lingo; the domain skill and compatible run alias route
to the same operations. The reviewed Runtime/Profile start protocol above remains
mandatory before these actions.

Record planning authority, implementation authority, review start, human
acceptance, or auxiliary conditions only through
`axiom --json workflow fact` with the exact Execution revision, one validated
reference, explicit `--active` value, and `--authorize-local`. Never infer a
fact from GitHub, CI, merge, review, Issue state, or conversation. Human
acceptance additionally requires terminal canonical completion and an explicit
human decision.

## status

Collect only missing Project, Project-scoped Repository, exact Work Item, and
Execution selectors. Run `axiom --json workflow status` and, when requested,
`axiom --json workflow evidence` with the exact selectors.

This operation is read-only. Never infer selectors from CWD, Git, Provider,
Runtime chat, or global discovery, and never classify workflow state
independently.

To discover Executions without an opaque identity, run
`axiom --json workflow list --project <uuid-or-slug>` (optionally
`--repository <key>`) and report the returned `executions`.

## list and show

Run `axiom --json work-item list --project <uuid-or-slug>` (optionally
`--repository <key>`) to discover Work Items linked to the Project; unlinked
Provider Issues are not listed. Run `axiom --json work-item show` with the exact
Project, Repository, and Work Item selectors for one link. Both are local and
read-only.

## update, comment, close, and reopen

Collect only missing Project, Project-scoped Repository, and exact Work Item
selectors. `update` changes only `--title` and the Axiom-authored sections;
classification and type stay create-time. Run the command without authority
first and present the returned preview, target, and effects. Repeat the same
inputs with `--preview-digest <digest> --authorize-external` only after explicit
authority for that exact preview. A Work Item already in the requested state is a
no-op. A comment is not idempotent: a repeated authorized comment posts again,
so inspect the Work Item before repeating one. Report partial Provider effects
exactly as Lingo returns them; never retry an ambiguous mutation blindly. `work-item complete` remains a CLI
compatibility spelling of close and is not routed here.

## Archived Projects and disabled Integrations

Lingo refuses operational evolution on an archived Project (`project_archived`)
and any use of a locally disabled Integration (`integration_disabled`) before any
effect. Report the refusal and its next action; never work around it.

## Shared invariants

Invoke only `axiom --json` for executable behavior. Unknown, duplicate, and
conflicting inputs go to Lingo validation. Runtime skill text is a presentation
and routing surface, not the workflow or domain source of truth.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `draft`, `selection`, `workItem`, `workflow`, `projection`, `executionTarget`, `runtimeResolution`, `previewDigest`, `executions`, `workItems`, `change`, `admission`

Copy canonical completion fields only from Lingo's top-level JSON object. Omit
absent canonical fields. Never derive, synthesize, or reinterpret a canonical
field from an operation-specific payload. Preserve each operation-specific
payload in its original Lingo semantics and JSON position; do not extract,
duplicate, rename, or relocate it.

Resolving an operation never grants Provider or local mutation authority.
