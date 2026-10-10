# Issue #275 — Stage-to-graph compiler

Status: partial implementation of #275, with the internal compilation contract
available for review. The public operator planning surface is not delivered;
human acceptance remains pending. The implementation consumes accepted
Specification 007 and ADR-0020.
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
- Node keys are `<stage-id>:<agent-id>`. Producer output tokens match dependent
  node input tokens. Each node references canonical stage-input bytes by their
  SHA-256 digest; the compiler returns those bytes' structured data and reference,
  without persisting an artifact or allocating Execution identities.
- Runtime choice is independent of the initiating Runtime. Explicit Profile
  references resolve exactly; `policy-default` uses the existing Project
  resolver. No-match, ambiguity, stale capability evidence and unsupported
  effort fail closed.
- Each stage input retains the exact Runtime preview, including Project,
  configuration and observation digests. Both the stage result and graph proposal
  digest change when the referenced input bytes change. Compilation refuses
  mixed Project/configuration snapshots and observable drift for the same Runtime.
- Required effects are reported separately from the parent authority ceiling.
- Child inputs carry the pinned workflow/stage identity, instructions, logical
  references and digests, assigned outputs, criteria, validators, gates,
  failure policy and declared predecessor outputs. Dependency output references
  require validated digests before dependent execution.
- Codex explicit effort maps to `--config model_reasoning_effort=<value>`;
  Claude maps to `CLAUDE_CODE_EFFORT_LEVEL=<value>`. Runtime-default emits no
  override. The stage compiler requires capability proof for explicit effort.
  The resolver additionally requires
  `Observation.nonInteractiveModelCapabilities[exactLocalModel][capability]`
  to be proven for the observed Runtime version/executable. Runtime-wide or
  declared-only effort capability is insufficient. The adapter rejects effort
  without its envelope capability and conflicting Claude effort environment keys.

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
- Existing subscription dispatch composition rejects arbitrary Codex `--config`
  arguments. #276 must explicitly admit the reviewed effort mapping while keeping
  model, credential and executable selection guards; adapter argv tests alone
  do not prove that composition dispatches explicit Codex effort.
- The production executable observer currently proves Runtime availability and
  Axiom skill integration, not model-specific reasoning-effort support. Explicit
  effort therefore remains blocked unless an authoritative observation supplies
  the exact capability; no support is inferred from model names or CLI presence.
- No live Codex/Claude invocation, human acceptance, merge, release or publication
  is claimed.

## Delivery audit against the implementation request

This audit reviews the internal contract, not whole-Issue or MVP acceptance.

| Requested behavior | Evidence / result |
| --- | --- |
| A — deterministic stage compilation and exact revision | Local compiler tests bind workflow content, Plan, stage input bytes and Runtime snapshots; forged/stale references and snapshot drift are rejected. |
| B — one agent, mixed Runtime DAG, sequential and parallel constraints | Local tests cover no fabricated single-agent graph, independent Codex/Claude nodes, integration, sequential predecessor chain, concurrency serialization and unsafe parallel overlap. |
| C — Project policy, exact Profile, capability and effort | Integration test uses `runtimeapplication.Service` and `runtimeprofile.Resolver`, checks exact model/mode proof, revalidates every returned preview and rejects missing proof. Exact Profile requests do not consult an unused default preference. |
| D — bounded inputs, references, controls and lineage | Stage-input references survive in-memory publication through existing `GraphService` envelopes; parent/child identity, node key and controls are preserved. Credential references are absent from serialized compilation output. Input bytes are not persisted here; actual referenced content admission is caller-owned. |
| E — fail-closed safety | Tests cover unapproved/unbounded Plans, zero/excess attempts and timeout, scope/Project/Repository mismatch, authority expansion, missing inputs, unsafe text, capability failure, dangling dependency and cycle. |
| F — operator inspection through existing application/presentation | **Partial:** structured internal results exist, but no public stage-plan command or canonical completion/presentation wrapper is wired. Execution binding comes from #274; composition/admission is still required. |

Therefore this PR advances #275 but does not claim to complete it. Passing the
internal tests does not satisfy the missing public inspection behavior. Live
Runtime dispatch is explicitly outside this implementation slice.

### Re-review regressions

`TestCompileBindsRuntimeSnapshotsAndStageKeys` and
`TestCompileRejectsPolicyDriftAcrossAgents` first failed on the original delivery,
then passed with the corrected binding. A credential-bearing authorized effect
target was also accepted by the original compiler; the shared portable secret
policy now refuses it before any input is returned. Additional tests cover producer/consumer
token matching, required effects vs authority ceiling, the original 32-node Plan
bound, exact model-specific effort proof and incompatible default preferences.

The original PR's `delivery-metadata` failure came from duplicate metadata keys in
the PR description. The corrected body must contain exactly one `Related-Issues`
line and one `Completes-Issues` line, checked with the base revision's
`scripts/delivery-issues.sh check-pr`; `Completes-Issues` remains `none`.

Final local re-review checks passed: focused tests; full `go test ./...` with
ambient `CODEX_API_KEY` and `OPENAI_API_KEY` unset; `go vet ./...`;
`go build ./...`; `go mod verify`; `scripts/check-go-quality.sh all` with pinned
staticcheck v0.8.1; `scripts/validate-repository.sh .`; the Specification 007
example validator; PR metadata validation; diff and security checks. The
repository validator's unrequested native maintainer behavioral scenarios remain
SKIPPED, not passed. Remote CI must be evaluated for the pushed final head.
