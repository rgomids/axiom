---
name: axiom-workflow
description: Configure Axiom Project workflow definitions and inspect Runtime/Profile configuration readiness through Lingo.
---

# Axiom Workflow

To inspect supported operations and accepted arguments before execution, run
`axiom skill inspect axiom-workflow` and present its `skill` payload.
Inspection stops there: do not collect inputs or execute an operation. The binary
owns argument names, requirements, accepted forms, authority metadata, and
executable command metadata; do not maintain a second argument registry here.

Supported domain operations are `definition.list`, `definition.show`, `definition.create`, `definition.edit`, `definition.validate`, `definition.remove`, `definition.recover`, and `configuration.readiness`.
Any other operation is unsupported: report that it is unavailable and run no
Lingo command for it. Project selects, Workflow configures, Work Item runs,
Execution performs. Selection belongs to `axiom-project`; execution and stage
planning belong to `axiom-work-item`. Ownership guidance is never an alias.
Dispatch (#276) and delivery/rework (#277) are unavailable in this slice.

## Operation routing

| Operation | Mode | Lingo command | Effect | Authority | Semantic resolution |
|---|---|---|---|---|---|
| `definition.list` | - | `axiom project workflow list --project <uuid-or-slug>` | read-only | none | allowed |
| `definition.show` | - | `axiom project workflow show --project <uuid-or-slug>` | read-only | none | allowed |
| `definition.create` | - | `axiom project workflow create --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `definition.edit` | - | `axiom project workflow edit --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `definition.validate` | - | `axiom project workflow validate --project <uuid-or-slug>` | read-only | none | allowed |
| `definition.remove` | - | `axiom project workflow remove --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `definition.recover` | - | `axiom project workflow recover --project <uuid-or-slug>` | local mutation | preview first; exact `--expected-revision`, `--preview-digest` plus `--authorize-local` | only for unambiguous workflow intent |
| `configuration.readiness` | configuration | `axiom runtime profile validate`, `axiom runtime profile preview` | read-only | none | allowed |
| `configuration.readiness` | runtime | `axiom runtime codex status`, `axiom runtime claude status` | read-only | none | allowed |
| `configuration.readiness` | authentication | `axiom runtime codex auth`, `axiom runtime claude auth` | read-only | none | allowed |

An explicit operation routes exactly as written, without semantic reinterpretation.
Without an explicit token, resolve unambiguous configuration intent: discover
and inspect definitions, create a built-in copy, or add/change/remove stages and
agents through definition editing. Ask one bounded clarification for materially
ambiguous intent or target. Semantic resolution never grants authority or invents
selectors. Preserve unknown, duplicate or conflicting inputs for Lingo validation.

## Conversational definition preparation

Collect only missing Project selector and definition identity. Never infer them
from CWD, Git remotes, chat history or recent activity. Read the canonical
`definition.list`/`definition.show` results for exact current references.
For a built-in copy, use `definition.create` with `--from-default` and the desired
workflow key, without silently selecting the resulting definition.

For a natural-language draft or edit, prepare complete JSON from user statements,
canonical reads and explicitly named defaults; ask about material unknowns.
Do not ask the user to hand-write JSON or locate digests. Preserve omitted fields
and user instructions/context references as data, not executable authority.
Only write the draft as a regular file in the Runtime's own scratch directory,
outside Project, Repository and Axiom state roots. Never write Project, index,
Execution, Profile, receipt or policy files directly. `--file` regular-file and
symlink hardening remains a known limitation owned by #307, not fixed by #303.

Cover the full definition contract: `schemaVersion`, `workflowId`, `revision`,
`name` and `stages`. Each stage has `id`, `purpose`, `instructions`, `phase`,
`checkpoint`, `mode`, `concurrency`, `inputs`, `outputs`, `completionCriteria`,
`validators`, `humanGates`, `failurePolicy` and `agents`.
Input entries carry `id`, `kind`, `source`, `required`; outputs carry `id`, `kind`,
`required`. Completion criteria carry `id`, `description`, `outputRef`,
`validatorRef`; validators carry `id`, `kind`, `policyRef`; human gates carry
`kind`, `timing`. Failure policy carries `onFailure`, `onUnknown`, `retry`.
Each agent has `id`, `role`, `responsibilities`, `complexity`, `dependsOn`,
`inputs`, `outputs`, `capabilities`, `runtimeConstraints`, `profileRef`, `effort`,
`timeoutSeconds`, `maximumAttempts`, `effectCeilings`, `integrationOwner` and
`validationOwner`. Effort carries `mode` and `value`.
Lingo owns schema limits, reference validation, sequential/DAG rules, concurrency,
ownership, retry limits and effect ceilings; never bypass a refusal or fabricate
validator results. Stage/Agent Runtime choices only constrain policy through
`runtimeConstraints`, `profileRef` (or `policy-default`), `complexity` and `effort`.
They never widen Project policy. Assume no fixed Profile tiers, model names,
credential values, installation paths or proven Runtime capabilities.

Validate the regular draft with `definition.validate`; stop on any refusal.
Run the requested mutating preview and present a readable summary alongside its exact effects and
`workflowAuthoring` payload. After the user approves that exact preview digest,
repeat the identical inputs with returned `projectRevision` as
`--expected-revision`, `previewDigest` as `--preview-digest`, and
`--authorize-local`. Changed draft bytes, references or facts require a new
validation and preview. Read back with `definition.show`. Creating/editing never
selects a definition: activation requires a separate `axiom-project`
`workflow.select` preview and approval.

For remove, use the exact returned source/revision/digest and reviewed preview.
For interruption, follow canonical `recovery inspect/apply` guidance;
`definition.recover` indexes only an exact confirmed orphan after a new reviewed
preview. Never repair, rewrite or prune internal files from this skill.

## Configuration readiness

Report three distinct point-in-time facts, without merging them into one verdict:
Configuration Valid from `runtime profile validate/preview`; Runtime Ready from
`runtime codex/claude status`; Authentication Ready from `runtime codex/claude auth`.
Authentication is read-only: no login or credential write and it never blocks
authoring. Report canonical observations only; do not infer authentication from
configuration validity or readiness from installed executables.

Project Runtime policy editing (G-1) is unavailable: #305.
Local Model Profile authoring (G-2) is unavailable: #306.
Global Doctor/readiness (G-5) is unavailable: #231.
Do not write these configurations or introduce new registries, fixed tiers or
approval mechanisms. Diagnostics do not run inference, native sessions or
Provider effects and never grant authority.

## Result contract

Invoke only `axiom` for executable behavior.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `workflowAuthoring`, `runtimeResolution`, `previewDigest`, `runtime`, `runtimeAuth`

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
