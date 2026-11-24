# Issue #404 — DAG concurrency mapper Specification

SPECIFY-404-01, 2026-11-24. Bounded bug specification and source Evidence;
not an implementation, Plan, Tasks, contract amendment or product PASS.

## Authority and problem

[Issue #404](https://github.com/rgomids/axiom/issues/404),
[accepted Intake](https://github.com/rgomids/axiom/issues/404#issuecomment-7132349983),
[human baseline acceptance](https://github.com/rgomids/axiom/issues/404#issuecomment-7132672246)
and [readiness](https://github.com/rgomids/axiom/issues/404#issuecomment-7132699949)
are the authority. GitHub #404 is the sole operational tracker.
Fetched `origin/main` and starting HEAD both equal
`5cd783bd2ac8189a1b9c85a790472b5335139c78` (v0.24.2 reproduction baseline).
The pre-existing suggested branch was clean at this SHA; it is preserved.
This attempt uses a new worktree and `issue/404-specify-dag-concurrency`.

Reported observation, not rerun here: immutable `ubuntu-t12` r1, digest
`80b733ae0281f99c4b5f6714db078952580242dbd7fd6b05e623adb86f550df8`,
Stage `build`, DAG concurrency 2, agents library/cli/reviewer, was refused
before Execution creation; battery 02 was BLOCKED/NOT_RUN, zero processes
([receipt reference in the public Issue](https://github.com/rgomids/axiom/issues/404)).
No private lab content was imported or independently verified.

Expected: a structurally valid DAG Stage must not be refused solely for
concurrency greater than one, while all existing limits and controls remain.
Observed source: `mapWorkflowDefinition` refuses `s.Concurrency != 1`.

## Scope and acceptance (ACCEPTED; unchanged)

Only mapper and tests for structurally valid DAG concurrency >1.

- **AC1:** mapper does not return `workflow_unmappable` solely because a valid
  DAG Stage has concurrency 2.
- **AC2:** future automated tests demonstrate mapping and run start for the
  valid case; they do not establish Execution completion or inference success.
- **AC3:** invalid cases remain refused; sequential/concurrency 1 has no regression.
- **AC4:** PR/Issue record real results and Evidence, including NOT_RUN/UNVERIFIED.

Non-goals: no immutable Workflow/digest edit, public flag, architecture expansion,
public contract change, runtime/effect/authorization/validation/security workaround,
real inference, T12 battery 03 or product external effects. No merge, release,
deploy, ready-for-review conversion, Issue closure or human acceptance.
Design is NOT_APPLICABLE: mapper/CLI without new interface.

## Contracts and verified source path

All source observations below refer to the fetched baseline SHA above.

| Source | Verified fact and consequence |
| --- | --- |
| [Spec 007 WF-005/006](spec.md#stage-and-agent-semantics-wf-005006) | Sequential requires concurrency 1 and predecessor chain; DAG preserves explicit dependencies. Overlap requires concurrency permission, non-conflicting effects and isolated writer workspaces. |
| [Definition validation](../../../internal/workflowdefinition/validation.go) around line 101 and Workflow schemas | Concurrency cannot exceed agent count; sequential cannot use a value other than 1. Schema bounds also apply. Structural validity alone does not implement every configured control. |
| [Mapper](../../../cmd/lingo/execution_run.go) `selectedWorkflow` / `mapWorkflowDefinition`, lines 683–883 | Definition Encode validates first; guard at 730 refuses >1. Stage order becomes Stage dependencies; agent dependencies are copied. Validators, gates, inputs, outputs, effects and final `NewExecution` validation have independent refusal paths. |
| [Canonical StageSpec](../../../internal/workflow/execution.go) lines 193–212 | No concurrency member exists. WorkflowSnapshot freezes StageSpec values and their digest; no declared limit is transferred through this representation. |
| [Canonical eligibility](../../../internal/workflow/execution_transitions.go) `Ready`, lines 247–304 | Applies dependencies, gates, failure/unknown/integration conditions; returns all eligible unattempted tasks without a declared Stage concurrency budget. |
| [Scheduler](../../../internal/stageexecution/scheduler.go) `runReady`, lines 61–153 | Iterates ready refs, prepares each under the principal writer, records inflight refs and starts workers. No declared Stage concurrency check was found on this path. |
| [Historical graph](../../../internal/executiongraph/successor.go) | StageConcurrency and maximum 64 exist, but are not evidence of limit propagation into canonical runs. |
| [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md) | Accepted 2026-10-30; new runs use one Execution and stageexecution engine. Successor Graph is no longer the operational owner for new runs. Spec 015's older Proposed paragraphs must be read with this accepted reconciliation. |
| [Spec 015](../015-minimum-viable-operational-workflow/spec.md), [ADR-0039](../../decisions/0039-unrestricted-runtime-stage-processes.md) | ADR-0039 accepted 2026-11-18 retains Axiom-mediated authority and unrestricted direct Runtime launch policy; neither is changed here. |
| [Spec 011 VP-04/E3](../011-rendered-workflow-presentation/spec.md) | Independent-join concurrency-2 fixture is presentation evidence only, never proof of dispatch or limit enforcement. |

Graphify was queried with `GRAPHIFY_BACKEND=none` against the external card graph;
its bounded result located mapper/StageSpec/NewExecution. Exact source inspection,
not graph freshness or inferred relationships, supports the findings above.

## Invariants, failures and risks

Retain declared dependency edges and join prerequisites without adding false
ordering; Stage ordering and one-active-Stage semantics remain unchanged.
Retain gates, validators, inputs, output obligations, effects, writer isolation,
integration ownership, frozen invocation, fencing/CAS, cancellation and truthful
unknown-effect handling. Messages or Runtime permission cannot expand authority.
No fallback Runtime/model or bypass of unsupported validators is permitted.

Invalid bounds, cycles, dangling references, invalid sequential declarations,
unmappable identifiers/controls and missing required context retain their
existing refusal semantics and must not create an Execution on pre-start refusal.
A valid concurrency value does not excuse a separate unsupported control.

**Material impediment — REQUIRED_SCOPE_CHANGE:** there is no representation or
application of the declared Stage limit on the inspected canonical path.
A structurally valid DAG with three independent roots, a declared integration/validation
owner joining all three, and concurrency 2 distinguishes
this from a two-root join fixture: removing the guard would allow all three to
be scheduled. This is a source-derived counterexample, not a performed run.
A mapper-only transfer cannot target a nonexistent StageSpec member. Restoring
the limit across snapshot and scheduling requires work outside the approved
mapper/test-only envelope, with persisted-contract implications to qualify.
No new encoding, ADR, contract change or solution is approved by this note.

Local alternatives assessed, not an implementation plan:

- Remove the guard only: smallest text change, loses the declared limit.
- Add synthetic dependencies or reduce concurrency: changes accepted DAG semantics
  or immutable intent; rejected.
- Reuse historical StageConcurrency for new runs: conflicts with ADR-0037 ownership;
  rejected.
- Preserve refusal and report the missing canonical limit: safe current disposition;
  dependent solution waits for material scope qualification/human decision.

The exact minimum representation/enforcement change remains unselected. Do not
conflate the existing accepted concurrency requirement with authority to expand
this attempt's file/contract scope. Reconsider the impediment if a reviewer supplies
an existing canonical limit path missed by this inspection.

## Future verification criteria (means for AC1–AC4, no new AC)

All rows below are **NOT_RUN**. Use local synthetic definitions, stub selectors,
stores and launchers; no vendor models or product external repositories.

| Scenario | Observable oracle / retained Evidence |
| --- | --- |
| Valid DAG 2 with independent A/B and join C | Mapping succeeds and preserves edges, controls and effective bound; start creates canonical Execution and eligible stub attempts. C cannot start before both required validated predecessors/integration. Start is not completion. |
| DAG 2 with three independent roots plus dependent join | Instrument prepared/running stub attempts with barriers; maximum active count never exceeds 2; remaining root becomes eligible as capacity returns. This detects loss of the limit which a two-root fixture cannot detect. |
| Invalid zero/out-of-bound concurrency, value above agent count, sequential 2, cycle/dangling edge | Existing validation/refusal categories remain; compare Execution store and stub spawn count before/after pre-start refusal: no new Execution, no spawn. |
| Sequential 1 and DAG 1 | Preserve predecessor chain where declared, independent edges where absent, gates/validators/output checks and bounded scheduling; no accidental concurrency increase. |
| Valid DAG 2 with separately unsupported validator/control or missing required context | Keep the specific existing refusal path; no Execution created merely because concurrency is valid. |
| Writer/effect/integration and unknown/gate cases | Reuse existing deterministic fixtures where applicable; verify isolation and existing refusal/fencing semantics remain, with no launch-policy change. |

Future PR/Issue Evidence records command, exit code, fixture, commit, observations
and PASS/FAIL/SKIP/NOT_RUN/UNVERIFIED separately. No product PASS is claimed here.
Implementation tests, real inference and T12 battery 03 are NOT_RUN by scope.

## Specify verification Evidence

`./scripts/validate-repository.sh .` exited 0 (repository/bootstrap/document
governance checks only). Relative Markdown link targets were checked locally;
the Spec 007 heading anchor was inspected. `git diff --cached --check` and
`./scripts/check-sensitive-files.sh --staged .` exited 0. Staged content was
reviewed. Gitleaks is unavailable: dedicated secret scanner SKIP; this does not
replace its coverage. Runtime behavioral scenarios were explicitly skipped by
the harness validator (UNVERIFIED). No product tests or real models ran.

## Governance and handoff

This companion note follows documentation governance's internal-bug task/Evidence
route; it does not edit the accepted Spec 007, create a numbered Specification,
or change consolidated lifecycle status. Constitution check: explicit intent,
small scope, source-backed Evidence and human material authority are preserved.
No production/test code changed; security/runtime boundaries stay unchanged.

Next gate: qualification of this Specification and its material impediment;
CLARIFY only if needed. PLAN is conditional on reconciling the mapper-only scope
with the missing canonical concurrency representation/enforcement. No PLAN,
TASKS, IMPLEMENT or Cursor dispatch is performed. Receipt publication belongs
in Issue #404; no authenticated continuation callback is assumed.
