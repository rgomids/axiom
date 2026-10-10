# Issue #275 — Stage-to-graph compiler

Status: implementation of #275 complete for review. PR #297 delivered the
internal compilation contract; the
[public stage planning completion](#public-stage-planning-completion) wires the
read-only `axiom workflow stage plan` operation over the #274 Execution binding.
Technical review and human acceptance remain pending. The implementation
consumes accepted Specification 007 and ADR-0020.
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

- Public `axiom workflow stage plan` wiring was deferred here until #274's pinned
  Execution binding existed; it is now delivered by the
  [public stage planning completion](#public-stage-planning-completion) below,
  without an alternate command or invented Execution identity.
- Dispatch, attempts, cancellation, recovery and output-digest admission remain
  #276 scope. This proposal cannot invoke agents or grant effects.
- Subscription dispatch admits only the adapter's trailing Codex effort override
  matching the capability-proven child envelope. Arbitrary/configured `--config`
  arguments remain rejected; Runtime/Profile, model, credential, executable and
  authentication guards remain authoritative. This boundary fix does not deliver
  #276 orchestration or live inference proof.
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

### PR #297 — subscription effort review remediation

The two P1 findings are corrected at the existing adapter/dispatch boundaries:

- `TestSubscriptionDispatchWithValidatedEffort` crosses the real policy resolver,
  adapter and subscription guard for Codex and Claude. The Codex regression failed
  with `ErrInvalidComposition` before the guard correction.
  `TestSubscriptionCodexEffortArgumentsFailClosed` refuses changed effort,
  arbitrary/duplicate configuration, model override and missing capability proof.
- Claude effort retains a private machine-local inherited environment snapshot
  separate from bounded explicit overrides. The preliminary auxiliary commit
  `e1444f28565a947322ca56bcc0f88dc6c28428c5` was inspected; copying inheritance
  directly into `Invocation.Env` was insufficient because scheduler limits would
  reject ordinary large/lowercase inherited environments. Adapter regressions
  first failed for lost inheritance and inherited effort conflicts; they now
  check snapshot stability, explicit/credential isolation and duplicate refusal.
- `TestClaudeSubscriptionEffortUsesInheritedEnvironmentInRealPreflight` runs the
  real preflight with controlled vendor status, proves HOME/USERPROFILE resolution
  and environment equivalence, and blocks inherited API credentials without
  exposing values. `TestClaudeEffortDispatchPreservesLargeInheritedEnvironment`
  crosses adapter, subscription guard and scheduler with a fake runner and more
  than 64 inherited entries plus a lowercase key. Preflight and runner receive
  identical effective environments. `TestInheritedSnapshotKeepsExplicitEnvironmentValidation`
  preserves explicit count/key/conflict checks; `TestOSProcessRunnerUsesCapturedEnvironment`
  checks snapshot/override delivery through a real local helper process.

These tests use synthetic profiles and controlled login observations. They do
not run vendor inference, establish subscription usability, or record human
acceptance. The public planning and production effort-observation gaps above
remain unchanged. Reviewer threads remain their authors' responsibility.

Local remediation validation passed: focused runtimeadapter/executiongraph/
graphapplication/workflowcompiler/runtimeprofile tests; full `go test ./...`;
focused `go test -race` for the four affected execution packages;
`go build ./...`, `go vet ./...`, `go mod verify`;
`scripts/check-go-quality.sh all` (Staticcheck v0.8.1),
`python3 scripts/test-go-quality.py`, and `scripts/validate-repository.sh .`.
Test processes unset ambient `CODEX_API_KEY` and `OPENAI_API_KEY`; toolchain
checks use consistent Go 1.26.0 PATH/GOROOT. Initial host toolchain mismatch
was corrected before successful quality checks. Native maintainer behavioral
scenarios are not requested by repository validation; no native/live vendor
acceptance is inferred. Remote CI must be read against the pushed final head.

## Public stage planning completion

This section completes #275 on top of v0.14.0 (`874b24b`). It reuses PR #297's
compiler and PR #299's (#274) Execution binding; it adds no scheduler, store,
Runtime invocation or Provider effect, and does not change Specification 007's
approved text or ADR-0020.

### Delivered boundary

- `axiom workflow stage plan --repository --work-item --execution
  --expected-revision --stage --plan [--project]` is registered in the command
  tree, hierarchical help, `skill inspect axiom-work-item` (operation `plan`,
  read-only) and the canonical routing catalog. It has no authority argument.
- `workflow.Service.StagePlanningContext` loads the exact Execution through the
  existing selector/scope checks, requires the exact expected revision, reads the
  retained `WorkflowBinding` snapshot (never the current Project selection) and
  binds stage inputs only from retained facts: pinned Project context digests,
  validated stage-ledger outputs and the canonical Work Item identity digest.
  Missing required inputs return `stage_prerequisite_missing` with
  `missing_input:<id>` conditions; historical format-1 Executions return
  `configured_binding_required`; completed or earlier stages return
  `stage_not_plannable`. It never saves.
- `workflowcompiler.Compiler.PlanStage` is the canonical application use case:
  it invokes the existing `Compile` and wraps its result in the Specification 007
  `StagePlan` DTO (`executionRef`, `workflowRef`, `stageId`, `stageInputRef`,
  `graphProposalRef` only for graph stages, `executionKind`, `resolutions` with
  requested/effective effort and capability-evidence reference, `validatorRefs`,
  `gateRefs`, `blockers`), the complete `compilation`, the Plan reference and
  Plan document digest, and a digest over all of them. Pending `before` human
  gates are blockers; planning never records or satisfies a fact.
- The approved Plan is a strict, bounded (`256 KiB`) `PlanDocument`: the
  existing `ApprovedPlan` plus an explicit `authorityCeiling`. Its digest is over
  the canonical encoding, so formatting alone does not change it.
- `ClassifyFailure` maps compiler/graph errors to the Specification 007
  categories: `unsupported_effort`, `runtime_unresolvable`, `authority_denied`
  (`denied_authority`; a new `ErrAuthorityExceeded` sentinel still matches
  `ErrInvalidRequest`), `invalid_stage_topology` and `invalid_stage_plan`.
- `cmd/lingo` composes admission (`execution.status` inspection class), the
  binding context, the Plan file, the existing Project Runtime policy service and
  the compiler, then re-reads the Execution; a concurrent revision change returns
  `stale_execution_revision` instead of a stale success. The capability check
  probes each supported Runtime with an explicit constraint and selects nothing.
- Presentation reuses the canonical completion plus `category`,
  `confirmedEffects` (always empty), `executionRef`, `workflowRef`, `stageId`,
  `conditions` and `plan`. Human output is the existing deterministic Markdown
  projection of that JSON; no LLM call formats it.
- The `axiom-work-item` skill routes `plan` to the same command and states that
  the proposal grants no authority; skill-set history pins the replaced v0.14.0
  digest.

### Issue #275 acceptance criteria

| Criterion | Test | Evidence |
| --- | --- | --- |
| Same definition/plan/policy snapshot → same proposal/digest | `TestPlanStageSingleAgentHasNoGraphAndIsDeterministic`; `TestWorkflowStagePlanPublicJourneySingleAndMixedRuntimeGraph` (repeated CLI plan); `TestDecodePlanDocumentIsStrictAndFormattingIndependent` | Equal digests for identical inputs; Execution revision, Plan document, Runtime configuration and observation changes each change the digest |
| One stage previews independent Claude/Codex children and integration dependency; another uses one agent | `TestPlanStageGraphExposesMixedRuntimesDependenciesAndGates`; public journey (custom R1 `implementation` vs built-in `intake`) | Graph: implementer→Codex/`medium`, reviewer→Claude/`high`, integrator→Codex owning both dependencies, ordering within concurrency 2; single: `executionKind: single`, no graph, one resolution |
| No-match, unavailable model/profile, unsupported effort, dangling dependency, cycle, unsafe overlap or excess authority fails before dispatch | `TestPlanStageFailsClosedWithSpecificationCategories`; public journey; `TestWorkflowStagePlanExecutableFailsClosedAndNeverMutates`; existing #297 compiler tests for cycle/dangling | `runtime_unresolvable` (no match, unavailable Claude without fallback, production observer), `unsupported_effort` (no model-specific proof), `authority_denied`, `invalid_stage_topology`; tree snapshots unchanged |
| Every child envelope preserves stage/workflow, lineage, scope, context/artifact refs and validation obligations | Public journey child-input assertions; existing `TestCompiledInputsSurviveExistingEnvelopePublication` | Each agent input carries the pinned workflow ref, stage ID, two digest-bound inputs (prior output + business context), validators and the human gate as read-only criteria |
| Configuration changes need a newly reviewed proposal; no implicit fallback or authority expansion | Determinism and drift assertions above; `--expected-revision` stale case | Observation drift changes the digest; stale revision is `denied_authority`; unavailable Claude is refused, not replaced |

### WF-007/WF-008 and AC-007/AC-008

| ID | Evidence |
| --- | --- |
| WF-007 / AC-007 | Deterministic compilation of single and DAG stages through the public operation; independent nodes allowed within concurrency; overlap/limit/topology refusals; no LLM involved in planning or rendering |
| WF-008 / AC-008 | Context and prior-output digests bound from the retained Execution; per-agent Project policy/Profile/effort resolution with exact model proof; no-match/unavailable/unsupported effort refused; child envelopes preserve controls; no implicit fallback |

### Verification

Local checks on the final working tree (Go 1.26.0; `CODEX_API_KEY` and
`OPENAI_API_KEY` unset):

- `go build ./...`, `go vet ./...`, `go mod verify`: pass.
- `scripts/check-go-quality.sh all` (Staticcheck v0.8.1): pass.
- `scripts/validate-repository.sh .`: pass; native maintainer behavioral
  scenarios SKIPPED (not requested), not passed.
- `python3 docs/specifications/007-configurable-workflows/validate_examples.py`: pass.
- `go test ./...`: all new and affected tests pass. Eight pre-existing tests fail
  identically on unmodified `874b24b` in the authoring container, which runs as
  root (symlink/foreign-owner refusals cannot trigger) and exports the hosting
  Claude Code session environment (authentication preflight reports
  `environment_ambiguous`): `TestRuntimeAuthObservesTheLookedUpExecutable`,
  `TestSkillRootRefusesSymlinkedRoot`,
  `TestSubscriptionDispatchRejectsInheritedEnvironment`,
  `TestClaudeSubscriptionEffortUsesInheritedEnvironmentInRealPreflight`,
  `TestUpgradeSkillConflictsHaveZeroEffects`,
  `TestInspectRecordedSourceRejectsUnsafeOrMissingSources`,
  `TestPortableStoreRejectsUserSymlinkAncestor` and
  `TestPublicationDirectoryRefusesSymlinkForeignOwnerAndMissing`. Remote CI on
  the pushed head is the authoritative run for them.
- `go test -race` for the workflow, compiler, CLI, graph, policy and profile
  packages and the `cmd/lingo` stage-plan, configured-execution, skill and help
  tests: pass.

### Independent review remediation

A bounded independent review found no Blocker. Accepted: the `--plan` path
must be a regular, non-symlink file checked before opening, so a FIFO cannot
block planning (`TestReadPlanDocumentRefusesNonRegularFiles`); gate satisfaction
is scoped to the current stage as a defensive guard (definitions already reject
a repeated gate kind with `duplicate_gate`;
`TestStagePlanningReportsRecordedCurrentGate`). Not changed, with reason:
`priorOutput` remains the same binding `workflow advance` validates; an empty
observation digest cannot reach a resolution because `previewMatches` requires
valid digests; an effect outside the ceiling remains an authority refusal.

### PR review remediation (CR-001)

Confirmed Major: a bare `..` Plan scope path, with a matching `repository-write`
target and authority ceiling, was accepted because the shared graph
`validRelativePath` and the compiler's `withinScope` refused `../...` but not
the bare parent token. The graph validator now refuses `..`, and Plan admission
requires every work scope path and write/integration target to be a normalized
Repository-confined slash path (no root, parent, absolute or backslash form),
refused as `authority_denied`. Regressions
`TestValidRelativePathRefusesParentAndRoot`,
`TestPlanStageFailsClosedWithSpecificationCategories` (parent scope, parent
write target within a matching ceiling, backslash scope) and the public journey's
`..` Plan each failed before the fix and pass after it, with no state change.

### Limitations

- The production executable observer proves only Axiom skill integration. On a
  real machine, stages needing other capabilities (the built-in `read`) or
  explicit effort are therefore refused as `runtime_unresolvable` or
  `unsupported_effort`. This is the intended fail-closed outcome, demonstrated by
  the executable test; authoritative capability/effort observation remains #272
  scope. Positive journeys use controlled observations over the real installed
  Project policy, configuration and Execution stores; they are synthetic tests,
  not operational Evidence of vendor inference.
- Plan approval is asserted by the operator-supplied Plan document (`approved`,
  `planDigest`); planning records no approval fact. #276 must bind the reviewed
  `plan.digest` before any dispatch.
- Artifact-kind stage inputs have no authoritative pre-execution source and stay
  unbound; a required one blocks planning with `stage_prerequisite_missing`.
- `controls.timeout` uses the existing graph DTO encoding (nanoseconds).
- No dispatch, retry, cancellation, delivery, merge, release or human acceptance
  is claimed.
