# ADR-0022 — Dedicated workflow configuration conversational surface

## Status

**Proposed on 2026-10-10** for human decision under
[Issue #303](https://github.com/rgomids/axiom/issues/303). The maintainer
refined the direction in the [PR #304 discussion](https://github.com/rgomids/axiom/pull/304):
the new `axiom-workflow` skill configures workflow definitions; it does **not**
start, control or plan a free-standing workflow Execution. This direction
is recorded for versioned reconciliation, not acceptance of the exact revised
text. The ADR remains `Proposed` until the maintainer accepts it together
with the [Specification 007 amendment](../specifications/007-configurable-workflows/amendment-303-workflow-skill.md).
Acceptance does not by itself authorize implementation, merge, release,
Runtime inference or Provider effects; each retains its own gate.

## Context

[ADR-0020](0020-workflow-definition-revision-binding.md)
and accepted Specification 007 HD-002 establish Project-owned workflow
definitions and active selection, and Work Item-bound Executions with immutable
workflow revision bindings. The existing CLI deliberately separates
`axiom project workflow *` (definition authoring/selection) from
`axiom workflow *` (Work Item-bound Execution operations).

Specification 007 also said not to introduce a dedicated `axiom-workflow`
skill. Delivery of #273–#275 showed the configuration UX cost: workflow
definitions, stages and agents are authored through the broad
`axiom-project` skill, and custom definitions require hand-written JSON.

A proposed dedicated skill initially grouped configuration with Execution
run/status/plan/acceptance operations. The maintainer clarified that this is
the wrong conversational boundary: a Workflow defines **how** work proceeds;
a Work Item is the specific **what**; an Execution represents its concrete
run. Executing a Workflow without a specific Work Item is not a supported
user journey.

This ADR changes only the conversational skill responsibility. Domain
ownership, persisted state, CLI contracts and authority remain unchanged.

## Decision

Introduce `axiom-workflow` as a dedicated **workflow configuration**
conversational skill in Codex and Claude, thinly routing to existing Project
workflow definition operations. The responsibilities are:

1. **Project chooses.** The Project owns definitions and selects the active
   workflow revision. `axiom-project` is the primary conversational entrypoint
   for active selection, Project policy and Project lifecycle.
2. **Workflow defines.** `axiom-workflow` lists, inspects, creates, edits,
   validates, removes and recovers definitions. It configures stages, agents,
   dependencies, validators, gates and Runtime/Model Profile references.
   It may perform read-only configuration readiness diagnostics but does
   not mutate Project policy or select the active revision.
3. **Work Item requests and controls.** `axiom-work-item` owns the
   conversational `run`, `status`, stage Plan and Execution lifecycle of a
   **specific** Work Item. The #275 conversational stage-plan acceptance
   procedure stays a #303 delivery requirement, but is initiated by the Work
   Item surface; it invokes `axiom-workflow` only for the configuration
   portion and `axiom-project` for active revision selection.
4. **Execution runs.** Running the specific Work Item resolves its Project's
   valid active workflow revision through the canonical application operation,
   freezes an immutable `WorkflowBinding` in the new Execution and never
   silently changes that binding when the Project selects a new revision.
   Missing or invalid active selection fails closed; there is no automatic
   fallback or free-standing Workflow run. This is existing canonical start
   behavior (verified against v0.15.0 in the amendment); skills add no
   selection logic.

Existing `axiom project workflow *` and `axiom workflow *` CLI operations,
their arguments, results and authority are preserved. In particular, this ADR
does not mandate adding or renaming CLI commands to `axiom work-item run`;
that is a **conversational intent**, routed to the existing Work Item-owned
canonical operations.

**No legacy skill routes (HD-007, option B).** Axiom is pre-MVP with no
external users, so there is no compatibility window. In the same #303
implementation, `axiom-project` loses its workflow definition
`workflow.list/show/create/edit/validate/remove/recover` skill routes, and
`axiom-workflow` becomes the only conversational skill for definition, stage
and agent configuration; no alias, forwarding, duplicate catalog action or
later cleanup milestone remains. `axiom-project` keeps active selection
(`workflow.select`); `axiom-work-item` keeps `run`, `status`, `plan` and its
Execution operations. Only skill routes change: the canonical CLI commands are
neither removed nor renamed. Historical skill-set receipts and history (for
example v0.15.0) stay immutable provenance, not live aliases.

Lingo and application services remain the sole owners of validation,
digests, preview/apply, authorization and state. Skills add no workflow
registry, scheduler, approval fact, result protocol or parallel domain logic.
Runtime-prepared draft files are working inputs, not Evidence or approval.
Each mutation uses its own canonical preview and explicit confirmation.

## Configuration scope and readiness (HD-006 direction)

A Project's portable policy is the set of **allowed** Runtimes and Model
Profiles, not an assertion that a Runtime is installed, authenticated or
operational on the machine. [Spec 002 policy v2](../specifications/002-lingo-project-initialization/runtime-policy-v2.md)
already permits multiple Runtime declarations, Project Model Profiles and
role/complexity preferences. The Project owns edits to that policy
([G-1 / #305](https://github.com/rgomids/axiom/issues/305)); local Runtime
configuration separately owns actual installed Runtime/Profile bindings
([G-2 / #306](https://github.com/rgomids/axiom/issues/306)).
Removing a Runtime from one Project disallows it only there; removing one
locally affects readiness of all dependent Projects without rewriting their
portable configuration or existing immutable Executions.

A Runtime may have **multiple named local Profiles**; three common example
names (`economy`, `balanced`, `advanced`) are optional presets, not a
hard-coded tier model. A Workflow's Stage/Agent records the requested
Runtime constraints, logical Profile references, complexity and effort; it
does not configure the local machine or imply that effort is comparable
between vendors. The effective candidate set is constrained by the Project
policy, local matching enabled Profiles, the Stage/Agent requirements and
authoritative capability observations. The v2 matching contract, including
portable Project model fields and current schema limits, is preserved until
an independently reviewed migration. Missing, inconsistent or ambiguous
configuration fails closed; no silent Runtime/Model fallback.

The [#231](https://github.com/rgomids/axiom/issues/231) Project readiness
contract provides the existing diagnostic owner for Repository bindings,
Provider availability and Project/runtime compatibility. A future aggregate
`axiom doctor` may reuse its validators but is **not** created by this ADR,
this PR or the two follow-ups. Active vendor inference requires separate
operator authorization; readiness never grants execution authority. Local
credential references and vendor login state are never portable Project or
Workflow data or published Evidence. G-2 includes an independent security
review before implementation.

HD-006's *separate follow-ups* are linked under [#303](https://github.com/rgomids/axiom/issues/303)
for technical traceability without reopening the frozen MVP epic. The
configuration-only skill in this ADR does not own G-1, G-2, stage dispatch,
Work Item execution or Doctor mutation. The proposed exact revised Spec/ADR
still requires human acceptance; the clarified direction does not authorize
implementation or AXM-7 acceptance.

## Alternatives considered

| Option | Coupling / complexity | User experience | Reversibility |
|---|---|---|---|
| A. Keep HD-002 (Project + Work Item skills only) | No new artifact | Workflow configuration remains buried in Project lifecycle | Trivial |
| **B. Dedicated `axiom-workflow` for configuration only (selected direction)** | One new installed skill/catalog/receipt/history pin; preserves domain and CLI | Clear boundaries: Project selects, Workflow configures, Work Item runs, Execution performs | Duplicate Project authoring skill routes removed in the same delivery (pre-MVP, HD-007); CLI unchanged; reversible by a later skill release |
| C. Skill plus new top-level workflow CLI tree | Changes accepted public commands/selection | Limited benefit for conversational configuration | Needs aliases/deprecation |
| D. New standalone Workflow domain/service | Duplicates Project ownership or migrates persisted state | Does not solve any necessary configuration UX problem beyond B | Costly and conflicts with ADR-0020 |

The earlier broad version of B (configuration **plus** Execution orchestration)
is rejected for this revision: it obscures the Work Item's lifecycle owner.

## Consequences

### Positive

- A distinct conversational skill to create and manage workflow configuration
  in both Codex and Claude without requiring operator-authored JSON.
- Project selection and Work Item execution remain understandable, separate
  user intents. A Work Item run uses its Project's active revision, with an
  immutable binding per Execution.
- #275 conversational acceptance still has an explicit home in
  `axiom-work-item`, using configuration handoff rather than converting the
  configuration skill into an execution orchestrator.
- Canonical CLI, validation, authority and revision semantics remain
  unchanged; each workflow intent has exactly one owning skill route.

### Negative / trade-offs

- A third skill adds inventory, install/upgrade/retirement, metadata, receipts,
  history pinning and parity test obligations for Codex and Claude.
- Upgrading changes the installed skill inventory immediately: the Project
  authoring skill routes disappear. After fresh install, upgrade from v0.15.0
  and reinstall in both Codex and Claude, `skill inspect`, help, the active
  inventory and routing must contain no obsolete Project authoring route,
  while Project selection and Work Item execution routes remain; retirement
  and skill-set history tests must prove it.
- Multi-surface journeys (configure → select → run) need clear handoff
  documentation and separate previews/approvals; no silent Project selection.
- Active-selection resolution on Work Item start is existing canonical
  behavior (`axiom workflow start` admission, verified against v0.15.0 in the
  amendment); the skills add no selection logic and no gap is opened.

## Relationship to accepted decisions

This ADR supersedes no ADR. It changes only the skill interaction surface
originally specified by Specification 007 HD-002, and it explicitly preserves
[ADR-0020](0020-workflow-definition-revision-binding.md)'s Project-owned
authoring/selection, Work Item-bound execution and immutable
`WorkflowBinding`. ADR-0003, ADR-0008 and ADR-0009 remain unchanged.
Superseded fragments of the accepted Specification are recorded in the
proposed amendment and are annotated only after formal acceptance.

## Revisit when

A Workflow becomes independently owned outside a Project; Work Items gain a
versioned, explicitly authorized workflow-override contract; CLI compatibility
routes are retired; or evidence shows that separate conversational ownership
prevents a required user journey.
