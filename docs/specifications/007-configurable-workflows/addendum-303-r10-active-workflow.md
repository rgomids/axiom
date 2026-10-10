# Specification 007 amendment #303 — Addendum R10: Project `activeWorkflow` read field

## Status and authority

| Item | Value |
|---|---|
| Identifier | Spec 007 / amendment #303 / **Addendum R10** |
| Decision | **R10 ACCEPTED — Option B** (formal, bounded contract exception), recorded by the maintainer on [Issue #303](https://github.com/rgomids/axiom/issues/303#issuecomment-6102574634), 2026-10-10, after review of the [Plan/Tasks](plan-tasks-303-workflow-skill.md) at PR #304 head `e0df3e366b214f3e8e8222fdc79ecee0ce8b346c` |
| This document | **Accepted, 2026-10-10** by [explicit human acceptance](https://github.com/rgomids/axiom/issues/303#issuecomment-6102714056), bound to PR #304 head `a548d44ea0e0f432eca80ebde80c420616771334`, original Addendum blob `14300b70226c807d2648e8e6b30181f23be3cae4`; this is contract approval, not implementation/merge authority |
| Amends | [Specification 007 amendment #303](amendment-303-workflow-skill.md) acceptance criteria **AC-015.3** and **AC-015.8** only, as a bounded exception |
| Preserved unchanged | The accepted amendment text and blob `9d1895b6…`, [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md) blob `13a52a62…`, HD-005–HD-008, PD-1 to PD-9 |
| Origin | Risk R10 in the [Plan/Tasks](plan-tasks-303-workflow-skill.md), raised by the refined [PD-1 decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101636404) |

**Authority boundary.** The decision authorizes only this contract delta and
its documentation reconciliation. It authorizes no product code, Provider
change, release, merge, T13 or #303 completion, or technical or final
acceptance of AXM-7. Implementation stays gated by acceptance of the
Plan/Tasks as a whole and by its T01 baseline gate. R-2/R-3 stay
`deferred_to_278` ([#278](https://github.com/rgomids/axiom/issues/278) /
AXM-12).

No ADR is added: the delta is an additive read-only field on two existing
results, and it changes no architectural decision of ADR-0022 or HD-005–HD-008.

## Why an exception is needed

PD-1 requires a Project-only query that shows the active workflow's ID, name
and revision. Today neither `project show` nor `project list` exposes the
selection (`internal/projectapp/list.go:37-42`; `internal/cli/cli.go:263-277`),
and the selection ref has no name. AC-015.3 ("Existing canonical CLI
operations and domain semantics remain unchanged") and AC-015.8 ("Canonical
CLI, result, digest, preview/apply, security and authority semantics are
unchanged") would otherwise forbid any result change. This addendum records
the single permitted exception; everything else in those criteria stays in
force.

## Exception (closed scope)

**EX-R10.1 — Operations.** Only two existing read-only operations change:
- `axiom project show --selector <slug-or-id>` gains `project.activeWorkflow`;
- `axiom project list` gains `activeWorkflow` on **each** listed Project.

Human output shows the same information, and no existing information is
removed. No other command, operation or result changes.

**EX-R10.2 — Field contract.** No fields beyond these:

| Field | Presence |
|---|---|
| `status` | always: `selected` \| `none` \| `unresolvable` |
| `workflowId`, `revision`, `digest`, `source` | whenever a selection is recorded (`selected` or `unresolvable`) |
| `name` | only when the definition at the exact revision is resolved |
| `category` | only when `unresolvable`, with the applicable canonical category |

**EX-R10.3 — Safe resolution.** The read uses the Project's persisted
`workflowSelection`:
- `source: builtin`: ID, revision and digest must equal the canonical
  built-in definition;
- `source: project`: the index entry must match, the document must decode
  strictly, and identity and digest must match exactly;
- no selection: `none`;
- inconsistent, missing, inaccessible or invalid: `unresolvable`, with the
  recorded ref preserved and no presumed name or fallback.

**EX-R10.4 — No effects.** The queries never select, repair or publish. They
never change Project, Workflow or Execution state, never access credentials
and never emit local paths. In `project list`, an `unresolvable` entry never
hides other Projects or changes the listing's overall success category.

**EX-R10.5 — Compatibility.** All of these are preserved:
- existing fields, flags, commands, aliases, modes, completion categories,
  messages, authority and prior semantics;
- `workflow.select` as the only mutable selection, with preview and
  confirmation, in the Project domain;
- `axiom-workflow` as the exclusive owner of `definition.list` and the other
  configuration operations.

No route is duplicated and no new CLI command is added.

**EX-R10.6 — Tests and Evidence.** Required tests:
- selected built-in;
- selected Project revision;
- `none`;
- digest or revision divergence;
- missing or unreadable document;
- mixed listing;
- proof of zero writes.

Existing tests are preserved unmodified, as AC-015.8 requires. If an
existing test compares the full `project show`/`list` payload exactly and so
fails, the work stops for a maintainer decision; the test is not edited
under this addendum. JSON and human output samples are recorded as
compatibility Evidence. `workflow.select` preview/approval tests stay
unchanged.

## Effect on AC-015

| Criterion | Effect |
|---|---|
| AC-015.3 | Unchanged, except that `project show`/`project list` results may carry `activeWorkflow` under EX-R10.1–EX-R10.5 |
| AC-015.8 | Unchanged, except the same additive field; digest, preview/apply, security and authority semantics are untouched |
| All other AC-015 items | Unchanged |

## Traceability

| Item | Location |
|---|---|
| Decision | [Issue #303 comment 6102574634](https://github.com/rgomids/axiom/issues/303#issuecomment-6102574634) |
| Plan | [Plan/Tasks](plan-tasks-303-workflow-skill.md) §3.4, risk R10, §8 |
| Implementation task | T08a; T13 requires T08a delivered |
| Tests | EX-R10.6, mapped in T08a acceptance criteria |
