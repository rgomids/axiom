---
name: axiom-work-item-run
description: Start or resume the bounded Axiom delivery workflow for a configured Work Item through Lingo.
---

# Run Axiom Work Item

To inspect accepted arguments before starting this workflow, run
`axiom --json skill inspect axiom-work-item-run` and report its
`skill` payload. Inspection requests stop there: do not collect inputs or execute
any command below. The binary owns argument names, descriptions, required and
conditional inputs, and accepted forms; do not maintain a separate argument list.
For workflow invocation, preserve the guided behavior below.

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

Invoke only `axiom --json`. Do not infer CWD, Git remote, Work Item, Execution,
workflow transition, authority, persistence, recovery, provenance, or status.
Unknown, duplicate, and conflicting inputs go to Lingo validation.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `workflow`, `projection`, `executionTarget`, `runtimeResolution`, `previewDigest`

Copy canonical completion fields only from Lingo's top-level JSON object. Omit
canonical fields absent from that object. Never derive, synthesize, or reinterpret
a canonical field from `workflow`, `projection`, `runtimeResolution`, or another
operation-specific payload. Preserve and report returned workflow, projection, or
Runtime resolution payloads separately according to their original operational
semantics, including Execution identity, current gate, revision, transitions,
selected Runtime/Profile, blocker, and applicable Evidence or projection data.
Keep `lifecycleStage` and auxiliary conditions as derived workflow output; never
reinterpret them as an independently writable lifecycle.
Never reinterpret an operation-specific payload as `details` or another canonical
field. Report a canonical `details` reference without copying or interpreting its
artifact. Skill text grants no Provider authority.
