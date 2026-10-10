# Issue #275 — Stage-to-graph compiler

Status: bounded local implementation for review; human acceptance remains
pending. The implementation consumes accepted Specification 007 and ADR-0020.
It does not dispatch Runtime processes, write Execution state, or perform
Provider effects.

## Delivered boundary

`internal/workflowcompiler` accepts an exact workflow revision, selected stage,
approved bounded work plan, digest-bound logical input references, parent
authority ceiling, and a Runtime/Profile preview port. It returns a structured
inspectable result and canonical digest.

- A one-agent stage returns `executionKind: single` with no graph proposal. The
  existing graph planner validates its bounded node semantics without creating
  graph identity or a second process.
- Multi-agent stages reuse `internal/executiongraph.Planner`; workflow
  dependencies, integration ownership, workspace scopes, effect ceilings,
  attempts, timeouts and stage concurrency are checked before a proposal is
  returned.
- Runtime choice is independent of the initiating Runtime. Explicit Profile
  references resolve exactly; `policy-default` uses the existing Project
  resolver. No-match, ambiguity, stale capability evidence and unsupported
  effort fail closed.
- Child inputs carry the pinned workflow/stage identity, instructions, logical
  references and digests, assigned outputs, criteria, validators, gates,
  failure policy and declared predecessor outputs. Dependency output references
  require validated digests before dependent execution.
- Codex explicit effort maps to `--config model_reasoning_effort=<value>`;
  Claude maps to `CLAUDE_CODE_EFFORT_LEVEL=<value>`. Runtime-default emits no
  override. The stage compiler requires capability proof for explicit effort.

## Verification map

- Single-agent result without graph: `TestCompileSingleAgentUsesNoGraphAndBindsStageInputs`.
- Mixed Codex/Claude DAG, integration dependencies, concurrency and stable
  digests: `TestCompileMultiAgentGraphPreservesRuntimesAndOrdering`.
- Unsupported effort, authority expansion, out-of-scope writes, forged revision
  content and missing references: `TestCompileFailsClosedForEffortAndUnboundedAuthority`,
  `TestCompileRejectsForgedDefinitionAndMissingReferences`.
- Exact requested Model Profile selection: runtimeprofile and runtimeapplication
  tests; exact adapter arguments and environment: runtimeadapter tests.

Focused checks passed locally:

```text
go test ./internal/workflowcompiler ./internal/executiongraph ./internal/runtimeprofile ./internal/runtimeapplication ./internal/runtimeadapter
```

## Boundaries and remaining work

- Public `axiom workflow stage plan` wiring remains dependent on #274's pinned
  Execution binding/status contract. Adding an alternate command or inventing an
  Execution identity here would conflict with that ownership boundary.
- Dispatch, attempts, cancellation, recovery and output-digest admission remain
  #276 scope. This proposal cannot invoke agents or grant effects.
- The production executable observer currently proves Runtime availability and
  Axiom skill integration, not model-specific reasoning-effort support. Explicit
  effort therefore remains blocked unless an authoritative observation supplies
  the exact capability; no support is inferred from model names or CLI presence.
- No live Codex/Claude invocation, human acceptance, merge, release or publication
  is claimed.
