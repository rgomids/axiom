---
name: axiom-work-item-run
description: Start, advance, authorize, resume or reconcile a configured Work Item workflow through Lingo.
---

# Run Axiom Work Item

To inspect accepted arguments before starting this workflow, run
`axiom --json skill inspect axiom-work-item-run` and report its
`skill` payload. Inspection requests stop there: do not collect inputs or execute
any command below. The binary owns argument names, descriptions, required and
conditional inputs, and accepted forms; do not maintain a separate argument list.
For workflow invocation, preserve the guided behavior below.

Collect only missing Project, Project-scoped Repository, exact Work Item, and
applicable Execution selectors. Use `--project <uuid-or-slug> --repository <key>
--work-item github:<owner>/<repository>#<number>`. `workflow start` omits
`--execution` and passes `--runtime codex` or `--runtime claude`, naming the
Runtime executing this skill; never guess it from installed executables. Every
resume, advance, status, evidence, or reconcile call forwards the exact
`--execution <id>` returned by Lingo and never passes `--runtime`: the Execution
keeps the Runtime recorded at start.

## Explicit operations and gate progression

Supported operation inputs are `start`, `advance`, `fact`, `resume`, and
`reconcile`; forward an explicit operation deterministically to the same
`axiom --json workflow <operation>` command. When the user requests delivery,
start/resume the exact workflow and follow Lingo's returned `workflow.gateAction`
and `workflow.gateCommand` argument array. Do not ask the user to say an internal
gate name or a magic phrase to continue.

- An action with `automatic: true` runs `workflow advance --automatic` using the
  exact returned selectors and revision. Intake currently is the only gate whose
  prerequisites Lingo can evaluate automatically. Do this during an authorized
  workflow run without a conversational confirmation.
- Other `advance` actions require actual work and its observed `pass` or `fail`
  result, plus applicable validated Evidence. Fill command placeholders with
  observed inputs; never treat a file digest as proof that its contents passed.
- A `fact` action requires the exact decision/condition and reference. Present
  that explicit action when missing human authority or input is required. Set
  `--authorize-local` only for an explicitly authorized decision, never because
  the returned command includes it. Clearing a condition requires Evidence that
  it was resolved; never clear it merely to make the workflow progress.
- A `resume` action uses the exact committed revision. Re-read status after each
  result; stop on a denied/failed operation, report it, and preserve local truth.

Examples (supply exact selectors on every command):

```text
axiom --json workflow advance <selectors> --expected-revision <revision> --automatic
axiom --json workflow advance <selectors> --expected-revision <revision> --gate <currentGate> --outcome <pass-or-fail> --reference <kind>:<path>:<sha256>
axiom --json workflow fact <selectors> --expected-revision <revision> --fact planning-authority --active --reference specification:<path>:<sha256> --authorize-local
```

`--automatic` cannot be combined with gate, outcome, reference or next inputs.
Do not synthesize gate policies in this skill. The existing run entrypoint stays
compatible until the domain-oriented Work Item surface (#229) adopts these same
application operations; no per-gate skills are introduced.

Record planning authority, implementation authority, review start, human
acceptance, or an auxiliary condition only through `axiom --json workflow fact`
with the exact Execution revision, one validated reference, explicit `--active`
value, and `--authorize-local`. Never infer a fact from GitHub, CI, merge, review,
Issue state, or conversation. Human acceptance additionally requires terminal
canonical completion and an explicit human decision.

Invoke only `axiom --json`. Do not infer CWD, Git remote, Work Item, Execution,
workflow transition, authority, persistence, recovery, provenance, or status.
Unknown, duplicate, and conflicting inputs go to Lingo validation.

Canonical completion fields: `status`, `result`, `references`, `next`, `details`, `provenance`
Operation-specific payloads preserved separately: `workflow`, `projection`

Copy canonical completion fields only from Lingo's top-level JSON object. Omit
canonical fields absent from that object. Never derive, synthesize, or reinterpret
a canonical field from `workflow`, `projection`, or another operation-specific
payload. Preserve and report returned workflow or projection payloads separately
according to their original operational semantics, including Execution identity,
current gate, revision, transitions, and applicable Evidence or projection data.
Keep `lifecycleStage` and auxiliary conditions as derived workflow output; never
reinterpret them as an independently writable lifecycle.
Never reinterpret an operation-specific payload as `details` or another canonical
field. Report a canonical `details` reference without copying or interpreting its
artifact. Skill text grants no Provider authority.
