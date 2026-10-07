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

Collect only missing Project, Project-scoped Repository, exact Work Item, and
applicable Execution selectors. Use `--project <uuid-or-slug> --repository <key>
--work-item github:<owner>/<repository>#<number>`.

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
or proven capability. Preserve and report the first call's `runtimeResolution`
and `previewDigest`. A blocker stops the workflow. Otherwise show the selected
Runtime, Model Profile, model, and revisions and wait for an explicit decision;
only then repeat the identical inputs with `--runtime-preview <previewDigest>`.
Never edit a digest or reuse one after a blocker: `stale_preview` requires a
fresh preview. The Execution's Runtime is the selected
`runtimeResolution.choice.runtimeId`.
`workflow start` omits `--execution`. Every resume, advance, status, evidence, or
reconcile call forwards the exact `--execution <id>` returned by Lingo and never
passes `--runtime`, policy inputs, or `--runtime-preview`: the Execution keeps
the Runtime recorded at start. Follow `currentGate` returned by Lingo. Advance
through `axiom --json workflow advance` with its required revision, outcome, and
repository-relative artifact reference.

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
Operation-specific payloads preserved separately: `workflow`, `projection`, `runtimeResolution`, `previewDigest`

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
