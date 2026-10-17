---
name: axiom-work-item
description: Create, run, or inspect Axiom Work Items through one domain-oriented Runtime surface backed by Lingo.
---

# Axiom Work Item

To inspect supported operations and accepted arguments before execution, run
`axiom skill inspect axiom-work-item` and present its `skill` payload.
Inspection stops there: do not collect inputs or execute an operation. The binary
owns argument names, requirements, accepted forms, and executable command
metadata; do not maintain a second argument registry in this skill.

Supported domain operations are `create`, `run`, `status`, `plan`, `list`, `show`, `update`, `comment`, `close`, `reopen`, `delivery.packet`, `stage.run`, `stage.status`, `stage.cancel`, `stage.retry`, `stage.reconcile`, `stage.integrate`, `coordination.list`, and `coordination.show`.
Any other operation, such as `delete` or Execution `cancel`, is unsupported:
report that it is not available and do not run a Lingo command for it. Work
Items are closed or reopened, never deleted; sequential Executions have no
cancellation contract (only a dispatched stage has `stage.cancel`), and
Execution history and Evidence are never edited. Runtime capability `show`,
`probe` and `invalidate` are not routed here: name the CLI command
`axiom runtime <codex|claude> capability ...` as the next action.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `create` | new | `axiom work-item create` | external mutation | preview first; create only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous create intent |
| `create` | existing | `axiom work-item select` | local mutation | preview first; link only with the exact `--preview-digest` plus `--authorize-local` | only for unambiguous select intent |
| `run` | transition | `axiom workflow start`, `axiom workflow advance`, `axiom workflow resume` | local mutation | `workflow start` previews the Runtime/Profile resolution first and starts only with the exact `--runtime-preview`; Lingo enforces the exact Execution revision and gate rules | only for unambiguous run intent |
| `run` | fact | `axiom workflow fact` | local mutation | only with `--authorize-local` and the exact Execution revision | only for unambiguous run intent |
| `run` | reconcile | `axiom workflow reconcile` | external mutation | preview first; publish only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous run intent |
| `status` | - | `axiom workflow status`, `axiom workflow evidence` | read-only | none | allowed |
| `status` | list | `axiom workflow list --project <uuid-or-slug>` | read-only | none | allowed |
| `plan` | - | `axiom workflow stage plan` | read-only | none; the proposal never dispatches, grants effects or satisfies human gates | allowed |
| `list` | - | `axiom work-item list --project <uuid-or-slug>` | read-only | none | allowed |
| `show` | - | `axiom work-item show --project <uuid-or-slug>` | read-only | none | allowed |
| `update` | - | `axiom work-item update --project <uuid-or-slug>` | external mutation | preview first; update only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous update intent |
| `comment` | - | `axiom work-item comment --project <uuid-or-slug>` | external mutation | preview first; comment only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous comment intent |
| `close` | - | `axiom work-item close --project <uuid-or-slug>` | external mutation | preview first; close only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous close intent |
| `reopen` | - | `axiom work-item reopen --project <uuid-or-slug>` | external mutation | preview first; reopen only with the exact `--preview-digest` plus `--authorize-external` | only for unambiguous reopen intent |
| `delivery.packet` | - | `axiom workflow delivery packet` | local mutation | preview first; apply only with the exact `--preview-digest` and `--authorize-local` | only for unambiguous delivery packet intent |
| `stage.run` | - | `axiom workflow stage run` | external mutation | preview first; dispatch only with the exact `--preview-digest`, `--authorize-local` and `--authorize-external` (vendor inference) | only for unambiguous stage run intent |
| `stage.status` | - | `axiom workflow stage status` | read-only | none; persisted state only, never a stop or success claim | allowed |
| `stage.cancel` | - | `axiom workflow stage cancel` | local mutation | preview first; apply only with the exact `--preview-digest` and `--authorize-local` | only for unambiguous stage cancel intent |
| `stage.retry` | - | `axiom workflow stage retry` | external mutation | preview first; dispatch only with the exact `--preview-digest`, `--authorize-local` and `--authorize-external` (vendor inference) | only for unambiguous stage retry intent |
| `stage.reconcile` | - | `axiom workflow stage reconcile` | local mutation | preview first; apply only with the exact `--preview-digest` and `--authorize-local` | only for unambiguous stage reconcile intent |
| `stage.integrate` | - | `axiom workflow stage integrate` | local mutation | preview first; apply only with the exact `--preview-digest` and `--authorize-local` | only for unambiguous stage integrate intent |
| `coordination.list` | - | `axiom workflow coordination list` | read-only | none; persisted state only, never a stop or success claim | allowed |
| `coordination.show` | - | `axiom workflow coordination show` | read-only | none; persisted state only, never a stop or success claim | allowed |

When the user supplies an explicit supported operation, use it exactly and route
directly to its Lingo commands. Do not perform semantic classification for an
explicit operation.

When no operation token is explicit, interpret clear domain intent across all
supported operations and modes. "Crie uma issue" selects `create` / `new`;
"Atualize o título dessa issue", "Comente na issue", "Feche a issue", and
"Reabra a issue" select `update`, `comment`, `close`, and `reopen`. The user
need not name an English CLI token or another skill.

Distinguish Work Item discovery (`list`) from Execution discovery (`status` /
`list`), and Work Item inspection (`show`) from Execution status or Evidence
(`status` / `default`). "Continue a execução" selects `run`; read its exact
status first and follow Lingo's applicable `resume` or gate action. Never guess
an advance, fact, or reconcile action. "Finish this" does not identify which
resource or effect: clarify first.

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

Run `axiom work-item create` without authority first. Present the
complete returned draft, authorship/assumptions, target, effects, expected
revision, and preview digest. After explicit authority for that exact preview,
repeat the same facts with `--preview-digest <digest> --authorize-external`.
Changed facts require a fresh preview.

For an existing provider Work Item, use an exact selector such as
`github:<owner>/<repository>#<number>` with
`axiom work-item select`; after preview review, repeat with its digest and
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
axiom workflow start <selectors> --role <role> --complexity <complexity> --capabilities <capabilities> --runtime <runtime>
axiom workflow start <selectors> --role <role> --complexity <complexity> --capabilities <capabilities> --runtime <runtime> --runtime-preview <previewDigest>
axiom workflow status <selectors> --execution <executionId>
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

Reconcile through `axiom workflow reconcile` with the exact Execution
revision; present the returned preview and repeat with its
`--preview-digest <digest> --authorize-external` only after explicit authority
for that exact preview.

After the reviewed start, follow Lingo's `workflow.gateAction` and
`workflow.gateCommand` argument array using the exact returned selectors and
revision. Do not ask the user to name an internal gate or say a magic phrase.

New configured Executions require an explicitly selected Project workflow; report
`workflow_selection_required` through the Project workflow surface without
silently selecting a default. Preserve `workflow.binding`, including definition
revision/digest, and inspect `stageContract`, `stageInputs`, `stageLedger` and
`blockers`. Resume always uses the retained snapshot; never substitute a newer
Project selection or repair a corrupt binding. Follow the returned ordered stage
IDs rather than imposing the legacy gate list.

Configured advance requires `--stage-result <json-file>` with exact declared input
and output references from authorized work. Preserve prior output/context digests
from `stageInputs`; the canonical Work Item needs no caller-supplied input.
Registered validators run in Lingo. Never invent an artifact, validator result,
human actor or authority. For a required human review, record `stage-review` only
after explicit authority for the exact result file and validated reference,
including `--actor <human-identity>`, `--active true` and `--authorize-local`.
Other configured human facts also require the actor. A technical pass cannot
replace a human fact or final acceptance.

Configured technical completion stops at `reviewed` with
`delivery_packet_required`. Exact packet-bound acceptance belongs to the delivery
operation (#277), which is not available in this slice. Never submit configured
`human-acceptance` with an Evidence file or stage output as a substitute packet.
Legacy format-1 acceptance retains its historical protocol.

- An action with `automatic: true` runs `workflow advance --automatic` during
  the authorized run without another conversational confirmation. Intake is
  the only legacy gate Lingo can evaluate automatically; configured stages require
  their complete result contract. Do not combine automatic mode with gate,
  outcome, reference, stage-result or next inputs.
- Other advance actions require actual authorized work and its observed `pass`
  or `fail` result with applicable validated Evidence. Fill command placeholders
  from observations; a file digest alone does not prove correctness.
- Fact actions identify explicit authority or a condition to resolve. Present
  the missing decision/input and record it only with validated references and
  explicit local authority. A returned `--authorize-local` grants no authority.
  Never clear conditions merely to make the workflow progress.
- Resume uses the exact committed revision. Re-read status after each result;
  stop and report denied/failed operations without inferring progression.

Gate policies remain in Lingo; the canonical domain skill routes directly
to the supported operations. The reviewed Runtime/Profile start protocol above remains
mandatory before these actions.

Record planning authority, implementation authority, review start, human
acceptance, or auxiliary conditions only through
`axiom workflow fact` with the exact Execution revision, one validated
reference, explicit `--active` value, and `--authorize-local`. Never infer a
fact from GitHub, CI, merge, review, Issue state, or conversation. Human
acceptance additionally requires terminal canonical completion and an explicit
human decision.

## status

Collect only missing Project, Project-scoped Repository, exact Work Item, and
Execution selectors. Run `axiom workflow status` and, when requested,
`axiom workflow evidence` with the exact selectors.

This operation is read-only. Never infer selectors from CWD, Git, Provider,
Runtime chat, or global discovery, and never classify workflow state
independently.

For configured Executions report the retained definition/revision/digest,
Project/Repository/Work Item identity, current stage, criteria, ledger, human facts,
blockers and returned next action. Do not expose retained context bodies or infer
acceptance from Provider projection.

To discover Executions without an opaque identity, run
`axiom workflow list --project <uuid-or-slug>` (optionally
`--repository <key>`) and report the returned `executions`.

## plan

Collect only missing Project, Project-scoped Repository, exact Work Item,
Execution, expected revision, stage ID and approved Plan file inputs, then run
`axiom workflow stage plan` with the persisted selectors returned by Lingo.
Never invent the stage, Plan file, scope, effects or authority ceiling: they
come from the user or the approved Plan. Lingo reads the Execution's retained
workflow revision, never the current Project selection.

This operation is read-only. Report `plan.executionKind`, `resolutions`,
`compilation`, `blockers`, `gateRefs` and the `plan.digest` exactly as
returned. A proposal is not authority: it never dispatches agents, records a
human fact or grants effects, and a changed Execution, Plan, Project policy or
Runtime observation requires a fresh plan. A refusal such as
`runtime_unresolvable`, `unsupported_effort`, `authority_denied` or
`stage_prerequisite_missing` stops the operation; never retry with a weaker
constraint, another Runtime or Profile, or a widened authority ceiling.

## acceptance (stage-plan, R-1)

For #275 stage-plan acceptance, first read the exact Execution status and its
retained `workflowRef`. Prepare the Plan document from the user's intent in
Runtime scratch as a regular file, outside Project, Repository and Axiom state
roots. Ask only for material unknowns. A provisional `approved: true` is needed
for canonical planning, but a Runtime-written value is never human approval.
Include `planRevision` and `planDigest` as opaque references without authority.
`planDigest` is the SHA-256 of exact bytes of the reviewed Plan/Tasks artifact
the user names, or a bounded scope artifact prepared in scratch and shown to
the user. Drafts are working files, not Evidence.

Run the read-only `plan` operation with the exact Execution revision, stage and
Plan file. Present the canonical proposal, blockers, scope, effects and bindings.
Obtain an explicit human confirmation bound to `planDocumentDigest`, Execution
ID and expected revision, `stageId`, `workflowRef` and `plan.digest`. The document
digest comes from Lingo canonical encoding; never define a Runtime digest or
claim byte identity that Lingo does not enforce. Record who confirmed which
digest only in conversation and the Evidence report (G-5), with no Axiom fact;
never infer it from the provisional value or a passing technical result. Any
change to these inputs or policy/Runtime observations requires a fresh proposal
and confirmation. Never create a new CLI approval operation or record a human
fact merely because planning succeeded.

Synthetic R-1 acceptance uses the deterministic maintainer runner
`scripts/acceptance/stage-plan-r1.py` against the exact clean source revision;
its sanitized Evidence is technical evidence, not human acceptance. R-2/R-3
remain deferred to #278. Publication/decision/rework (#277) remain unavailable. A proposal
never performs Execution work; dispatch is the separately authorized
`stage.run`. Definition configuration
belongs to `axiom-workflow`, selection belongs to `axiom-project`, and Work Item
`run`/`status`/`plan` retain the canonical execution boundary.

## delivery packet

Use `axiom workflow delivery packet` with the exact persisted Project, Repository,
Work Item and Execution selectors. Report `packetDigest`, readiness, blockers and
next action unchanged. Retain only after exact local authority by repeating with
`--preview-digest <packetDigest> --authorize-local`. `--packet <digest>` inspects
a retained immutable packet.
Packet readiness grants no publication, acceptance or rework authority.

## stage dispatch and coordination

Stage operations act on one exact Execution, using the persisted selectors
Lingo returned. Collect only missing selectors, expected revision, stage ID,
approved Plan file and reviewed validator policy digest; never invent them.

Every mutation is a two-step protocol. Run the command without authority
first and present the returned preview unchanged: per-child rights,
capability states, confinement (`os-enforced`, `advisory` or `unknown`),
`not_tested` roots, effects and the digest. Only after an explicit human
decision for that exact preview, repeat the identical inputs with
`--preview-digest <digest>` and exactly the authority flags in the routing
table. `stage.run` and `stage.retry` start vendor inference and need both
`--authorize-local` and `--authorize-external`; `stage.cancel`,
`stage.reconcile` and `stage.integrate` need only `--authorize-local`.
Routing, a preview or a passing status never grants authority, and a skill
never supplies an authority flag on its own.

Resume after a restart is a `stage.run` re-invocation over the already
committed authority; there is no separate resume command. Stale or orphan
attempts need `stage.reconcile` first; `stage.retry` needs an explicit
`--disposition` after possible workspace effects. `cancellation_requested`
reports an observed request, never an observed stop. `unknown`,
`recovery_required` and `partial` / `confirmed_effect_reconciliation_required`
are reported exactly; never retry an ambiguous effect blindly. Stage
integration is a local repository effect: it never pushes, opens a PR,
merges, or accepts the Work Item.

`coordination.list` and `coordination.show` read canonical records only.
`axiom workflow coordination publish` is used by a running stage child from its
task packet, never by this skill or from a human shell. Records grant no
authority.

## list and show

Run `axiom work-item list --project <uuid-or-slug>` (optionally
`--repository <key>`) to discover Work Items linked to the Project; unlinked
Provider Issues are not listed. Run `axiom work-item show` with the exact
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

Invoke only `axiom` for executable behavior. Unknown, duplicate, and
conflicting inputs go to Lingo validation. Runtime skill text is a presentation
and routing surface, not the workflow or domain source of truth.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `draft`, `selection`, `workItem`, `workflow`, `projection`, `executionTarget`, `runtimeResolution`, `previewDigest`, `executions`, `workItems`, `change`, `admission`, `stage`

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
operation-specific payload, and do not extract, duplicate, rename, or relocate
a payload.

Resolving an operation never grants Provider or local mutation authority.
