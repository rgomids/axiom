# AXM-36 implementation Evidence

## r2 input and authority

Input: `91b04156a48005d9a7498372ae7990ce4fbb515d`, clean branch
`docs/axm-36-runtime-admission-specify`, existing [PR #334](https://github.com/rgomids/axiom/pull/334).
Tasks authority: [human approval](https://github.com/rgomids/axiom/pull/334#issuecomment-6279165229).
The [r2 authorization](https://github.com/rgomids/axiom/pull/334#issuecomment-6279268645)
explicitly permits a new state root dedicated to this box. It resolves the local
prerequisite reported in [r1](https://github.com/rgomids/axiom/pull/334#issuecomment-6279208460),
without reconstructing an Execution on another host.

Read-back on 2026-10-19: remote branch and PR head match input; PR open, documentary
CI `PR Checks` and `delivery-metadata` SUCCESS. Linear AXM-36 is In Review with
no blockedBy; AXM-12/39 remain successors. Approval/gate/r2 comments were read;
no competing implementation was found in the inspected repository/provider state.
This observation cannot establish inactivity on another host.

`cmd/lingo/main.go:205` and `internal/local/state_root.go` resolve the Linux
canonical default to `/home/box/.local/state/lingo` on this box. Neither state-root
override is set. Before creation the path was absent. An exclusive directory
creation after checking all ancestors for symlinks succeeded with mode `0700`;
zero JSON records. This is local root creation, not a workflow start, canonical
gate advancement or acceptance. No external-host Execution is imported. No real
Runtime, auth command, probe, inference or external E2E is run.

No delegation: controls expose model overrides, but no evidence proves the
strict model hierarchy requested by the maintainer. All work stays in the main
session. Existing stdlib JSON, SHA-256 and repository strict JSON decoder are
sufficient for the inactive codecs; no new dependency or infrastructure is needed.

## T01 schema checkpoint before edits

Current owners and next unused generations match the approved Tasks at input.
Do not activate successors until the coordinated T02 cutover. All fields in the
following configuration and tuple projections participate in their canonical
identity. Historical codecs and digest fixtures retain their original meaning.

| Owner | Closed fields and semantics |
| --- | --- |
| Configuration 4 | Required `formatVersion`, positive `revision`, nonempty bounded `runtimes`, `modelProfiles`; optional `preferences`. Runtime: required `id`, `adapter`, `enabled`, `allowlistedProfileIds`; no declarations/credential reference. Profile: required `id`, `runtimeId`, `model`, `capabilities`, `effort`; optional `complexities`. Effort: required `mode`, `value`. Preference: required `role`, `complexity`, `modelProfileId`. Retain existing Runtime/Profile ownership, allowlist, safe-token, count and 256 KiB rules. Canonical sorting and final newline enter DigestV4. Retirement remains separate tombstone v1 |
| Resolved tuple for configured Execution 4 | Required `profileId`, `runtimeId`, `model`, `effort`, `profileDigest`, `configurationRevision`, `configurationDigest`, `executableDigest`, `invocationMode`, `observationRevision`, `observationDigest`, `effortSourceDigest`. Positive revisions, safe identifiers, noninteractive mode, lowercase 64-hex digests; no proof reference/version. Version is diagnostic metadata outside tuple identity |
| Historical configuration 1/2/3 | Existing Decode/DecodeVersioned, Encode/EncodeV3 and Digest/DigestV3 fixtures remain. V3 active consumer/authoring remains unchanged until T02. No conversion or fake declaration |
| Historical Execution/graph/single | Execution 1/2/3, graph 1/2/3 and single run 1 retain original readers/bytes. Further successor inventory must include binding/envelope/attempt fields and historical digest vectors before those owners are edited |
| Pending successor owners | Compiler result 3, StagePlan 3, graph 4, single run 2, DispatchInputs 2 and optional attempt diagnostic schema are still T01 obligations. Configuration successor alone grants no new dispatch authority |

Actual coupling at input: runtimeprofile v2/v3_consumer applies proof loops;
runtimeadapter binding_source/version_registry enriches identity; effort_inspector
and stage_effort require declarations/version; authpreflight returns early on bad
version; stage_invocation branches on exact vendor versions. Policy composition,
workflowcompiler, stageexecution and supervisor consume these facts. T02 must
switch all of them before T03 retires producers, stores and CLI routes. The
approved Plan's dependency table identifies these owners; no registry bypass is
introduced by inactive foundations.

## T01 inactive foundation checkpoint

Implemented owners:

- `internal/runtimeprofile/v4.go`, `resolved_v4.go`, `v4_consumer.go`: strict
  Configuration 4 and proof-free tuple codecs/digests, versioned read dispatch,
  exact enabled/allowlisted Profile consumption and independent operational
  observation. Collection time is excluded from identity. Observation includes
  Runtime/adapter, installed/available, executable/mode, revision, observed
  subscription and effort-source digest; no capability/model-support assertion.
  Missing auth/source facts cannot resolve. There is no new local store writer,
  implicit migration or active policy/compiler use of these contracts.
- `internal/runtimeadapter/observation.go`: fresh operational identity method
  does not invoke integration/proof inspection, auth or a Runtime. Auth and
  effort remain separate observations; identity alone cannot admit.
- `authpreflight.go`: inactive `CheckOperational` shares the existing bounded,
  separately consented status runner, config/environment filters and auth
  classifiers. Missing/unrecognized/nonzero version metadata continues to real
  status; timeout/cancellation, executable disappearance/drift and unsafe auth
  configuration still refuse. Recheck identity between version/status and after
  status. `Check` remains the active legacy entry point.
- `stage_invocation.go` / `frozen_stage.go`: inactive operational/frozen-v4
  builders use the same reviewed permission/sandbox options and closed environment
  as historical builders, without version branches. Frozen builder checks exact
  executable identity, Runtime/Profile/model and Profile-derived effort.
- `effort_v4.go`: fixed adapter source rules, direct Configuration 4 consumption,
  real child HOME/CWD root resolution and version/proof-free digest. Source
  projection binds adapter rule, intent, executable/mode and sorted sanitized
  presence/location/content facts. It reads no auth files. Preserve trusted
  anchors, no relocation/symlinks, source size/count bounds and double-read checks.
  Read-only inspection uses the actual inbox CWD; repository-write comparison
  checks presence/bytes and project location-independent facts in both views.
- `effort_sources.go`: extract the existing bounded facts inspection for reuse;
  legacy projection retains its proof field and historical ordering/bytes. The
  new projection contains neither version nor proof/precedence fields. Legacy
  TOML parsing retains its old profile-table refusal; the new path inspects
  nested profile selectors and accepts unrelated profile settings.

No new interface, dependency, store, registry, qualification receipt or probe is
added. Two new concise comments document public API semantics (`CheckOperational`
and `BuildOperationalStageInvocation`), the permitted public-API exception to
this task's no-narrative-comments rule. Historical comments remain intact.

Primary source check, 2026-10-19: the [official Codex configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
confirms the effort selector and nested local settings. The bounded existing
TOML inspection recognizes that key in nested/dotted tables; unfamiliar versions
and unrelated settings do not select adapter behavior. No cloud-source traversal
or universal coverage is asserted.

The [Claude model configuration reference](https://code.claude.com/docs/en/model-config)
identifies the environment/argv selectors, settings effort, per-model settings
and effort caps. Existing JSON roots now reject `effortLevel`,
`env.CLAUDE_CODE_EFFORT_LEVEL`, `maxEffortLevel` and `modelSettings` as competing
sources, even for equal values. `modelSettings` presence is a bounded conflict;
no speculative per-model schema/precedence is implemented. Detailed settings
reference retrieval failed (web size limit; Markdown unsupported; direct fetch
403); no alternative source or trust-root expansion was substituted. Persistent
`ultracode` alone is not treated as an effort selector. No fixture-only
`fixtureEffortKey` is used in the new rules. These are adapter rules, never
per-install qualification or a claim of exhaustive future selector support.

Tests added: `v4_test.go`, `v4_consumer_test.go`, `auth_operational_test.go`,
`observation_operational_test.go`, `effort_v4_test.go`; golden configuration,
resolved tuple and digest fixtures in `internal/runtimeprofile/testdata/axm36`.
The sole historical test edit moves the future-version assertion from 4 to 5.
Configuration 1/2/3 bytes/digests and all existing fixture files are preserved.

## Task × acceptance checkpoint matrix

`PARTIAL` below means a passing inactive foundation test, **not** satisfaction
of the complete criterion or completion of T01. `NOT_RUN` refers to successor
production behavior. Existing legacy regression results do not qualify the new
proof-free installed journey. Both-adapter unit tests are synthetic and launch
no inference children; status calls use fake runners, separately counted.

| AC | T01 foundation Evidence / remaining obligation | T02 | T03 |
| --- | --- | --- | --- |
| RA-AC1 | PARTIAL: Configuration/tuple round trips; installed production admission NOT_RUN | NOT_RUN | NOT_RUN |
| RA-AC2 | PARTIAL: optional version/auth independence, identical version-independent argv, fresh executable bytes, source/tuple digest drift; apply refusal still pending | NOT_RUN | NOT_RUN |
| RA-AC3 | PARTIAL: exact disabled/absent/mismatched/unlisted Profile consumer tests, defensive copies; shared Project policy/binding cutover pending | NOT_RUN | NOT_RUN |
| RA-AC4 | NOT_RUN: active authority and zero-spawn enforcement unchanged; successor installed ordering pending | NOT_RUN | NOT_RUN |
| RA-AC5 | PARTIAL: auth missing/unknown/API override, timeout/cancellation, executable drift/disappearance; effort malformed/oversize/relocation/symlink/unsafe roots; production preflight pending | NOT_RUN | NOT_RUN |
| RA-AC6 | PARTIAL: one version-independent invocation and frozen identity/Profile checks; real synthetic option/model/effort rejection classification pending | NOT_RUN | NOT_RUN |
| RA-AC7 | NOT_RUN: qualifying-result semantics unchanged; successor diagnostics/result regression pending | NOT_RUN | NOT_RUN |
| RA-AC8 | NOT_RUN: historical readers/supervisor unchanged; successor atomic persistence/recovery codecs pending | NOT_RUN | NOT_RUN |
| RA-AC9 | PARTIAL: explicit/default effort, equal-value conflicts, nested sources, actual read-only inbox, repository-write child bytes/presence; production confinement/dispatch negatives pending | NOT_RUN | NOT_RUN |
| RA-AC10 | PARTIAL: auth subprocess payload and effort host/config data absent from projections; persisted attempt summary and presentation schema pending | NOT_RUN | NOT_RUN |

T01 remains incomplete: finish the exact retained-field inventory and strict
successor/historical codecs for configured Execution 4, compiler result 3,
StagePlan 3, graph 4, single run 2 and DispatchInputs 2, plus optional sanitized
attempt diagnostic summary; extend local inventory/publication/compatibility
fixtures, capture remaining digest vectors and pre-spawn negative checks.
Only after that validated checkpoint may T02 begin. T02 and T03 remain a joint
delivery requirement. The active admission gate is deliberately still intact.

## Downstream inventory for the next T01 checkpoint

The following **unedited** owners were inspected after the foundation. This is
an implementation map, not activation or a claim that their successor codecs
exist. Recheck their validators and writer/read dispatch before editing.

| Owner | Retained wire and nested-contract facts |
| --- | --- |
| `internal/local/workflow_store.go:executionDTO` | `binding` and `stageOrdinal` optional; required `formatVersion`, `executionId`, `workflowVersion`, `projectId`, `repositoryKey`, `workItem`, `runtimeId`, `stage`, `revision`, `status`, `transitions`, `createdAt`, `updatedAt`, `provenance`, `projections`; optional `terminal`. Do not derive the wire from `workflow.State`, which also has storage/delivery fields outside this DTO. Existing strict read dispatch explicitly handles 2/3 and needs a separate 4 path |
| `workflow.WorkflowBinding` | Optional historical `resolvedProfile`; required `projectRevision`, `localRevision`, `contexts`, `projectId`, `definition`, `schemaVersion`, `snapshotRef`, `snapshot`, `observationDigest`, `runtimePreview`, `boundAt`. Schema-2 snapshot stays immutable; new configured-4 tuple must be required in its generation, historical tuples opaque |
| `executiongraph.SuccessorGraph` | `formatVersion`, `parent`, `children`, `classificationPolicy`, `dispatchAuthorityRef`, `stageConcurrency`; optional `cancelRequest`; storage revision excluded. Parent: execution/revision, proposal/authority digests, authority effects, integration child and creation time |
| `SuccessorChild` / `ChildEnvelope` | Child execution/parent/revision/node/envelope/attempts. Envelope retains capability intent, inputs/outputs, scope/workspace/dependencies, allowed effects/authority ref, controls, validation/integration/optional flags and resolution. Resolution contains Runtime/Profile IDs, configuration/observation revisions and the generation-specific frozen tuple |
| `SuccessorAttempt` | `attemptId`, `number`, `status`, `ambiguousEffect`, `exitCode`, `leaseEpoch`, `packetDigest`, `reconciliationFacts`; optional start/finish, result/output digests, cancellation flag, classification reason, qualifying ref, process correlation. Diagnostic summary must be optional only in successor graph/single, never added to historical closed schemas. Existing lease/authority/classification/reconciliation validators remain binding |
| `SingleStageRun` / binding | Run: kind/version, scope digest/scope, binding, created time, epoch, authority refs/attempts and optional cancellation. Binding: execution generation, workflow revision, preview/proposal/input/Axiom digests, workspace/inbox, effects, controls and resolution. Keep receipt body/version 2 unchanged; its opaque digests bind new preview bytes |
| `workflowcompiler.Result` / `StagePlan` | Result retains workflow/stage, execution kind/mode/concurrency/order, required authority/ceiling, validation, optional proposal, agents, plan revision/digest and digest. StagePlan retains execution/project/repository/workflow/stage/kind, plan/document/input refs, optional graph proposal ref, resolutions, validator/gate refs, blockers, compilation and digest. Remove only proof-derived resolution fields and use the new frozen tuple throughout nested execution inputs |
| `executiongraph.DispatchInputs` | All five members required: `formatVersion`, `projectId`, `parentId`, `previewDigest`, `preview`; 1 MiB bound. Version 2 must bind exact successor preview semantics; raw payload is not dispatch authority |

Additional nested coupling confirmed in `internal/executiongraph/planner.go`:
`WorkUnit` and `ProposedNode` both carry historical `ResolvedProfile`, and
`Proposal` is explicitly format 2 when that tuple exists. Its validator checks
`ValidProfileControls` and its digest covers the tuple. The compiler sets it in
both single validation and graph proposal creation. Reconcile this nested
version ownership with D2 before implementing the compiler/graph successors;
never put a proof-free tuple into historical Proposal 2 by blanking old fields.
The operator-supplied PlanDocument 1 and compiler-owned derived tuple have
separate authority: inspect that boundary rather than migrating raw Plans.

Direct active certification imports at the input still exist in:
`cmd/lingo/{runtime_policy,runtime_capability_probe,capability_probe,stage_service,runtime_capability,probe_canary,capability_composition,runtime_capability_extended,stage_dispatch}.go`,
`internal/runtimeadapter/{interface_observation,version_registry,binding_source,stage_effort}.go`,
`internal/stageexecution/preview.go`, and
`internal/local/{extended_probe,capability_store,inventory,extended_probe_authority,probe_envelope,probe_canary,probe_session}.go`.
Profile proof loops and compiler/workflow/graph frozen-tuple consumers add
transitive dependencies. None of these imports has been removed or bypassed;
T03's residual search must categorize them after the coordinated cutover.

## Validation and engineering/security review

PASS:

- Baseline at input: `go test ./internal/runtimeprofile ./internal/runtimeadapter
  ./internal/workflowcompiler ./internal/workflow ./internal/executiongraph
  ./internal/local` (all six packages).
- Foundation regressions: `go test ./internal/runtimeprofile
  ./internal/runtimeadapter -count=1`; new both-adapter auth/identity/effort,
  closed-schema, frozen-tuple and defensive-copy tests pass. The last review fix
  refuses inherited effort and relocation **before** closed-environment filtering;
  runtimeadapter's entire suite was rerun after that fix.
- Broader targeted suites: runtimeprofile, runtimeadapter, runtimeapplication,
  workflowcompiler, workflow, stageexecution, executiongraph, local, cli and
  codexruntime all pass.
- `AXIOM_TEST_PROBE_STATE_ROOT=/home/box/axiom-axm36-synthetic.UsdrLH go test ./...`:
  PASS, including historical fake probe tests. This dedicated mode-0700 synthetic
  test root is separate from the canonical Lingo root; it was created using
  `mktemp -d` under HOME, matching the CI trust boundary. The full sweep compiled
  before the final inactive inherited-environment guard; runtimeadapter was
  reverified afterward, and remote exact-head CI remains a separate check.
- `go test ./cmd/lingo -run 'TestInstalledSynthetic' -count=1 -v`: PASS, eight
  top-level tests: SingleLifecycleAdversarial, SingleSupervisorCrashAndReconcile,
  SingleLedgerCorruptionFailsClosed, SingleLineageCapacityAndAdmission,
  SingleStageJourney, SingleStageRetryReset, GraphStageJourney and
  GraphConflictRetryReset (all prefixed `TestInstalledSynthetic`). These use
  temporary installed binaries/skills, synthetic Runtime images and unrelated
  CWD. Current legacy journeys still seed proof/registry fixtures and use a
  test overlay: this is regression Evidence, **not** the required proof-free
  production-composition installed acceptance. No harness shortcut was added.
- `go vet ./...`, `go build ./...`, `go mod verify` (all modules verified).
- `PATH=<existing-Go-1.26-bin>:<existing-staticcheck-bin>:$PATH
  ./scripts/check-go-quality.sh`: PASS format, module tidiness and pinned
  Staticcheck 2026.2.1, including tests. No tool/dependency installed.
- `./scripts/validate-repository.sh .`: PASS repository/bootstrap,
  33 ADRs / 56 governance fixtures, automation and sensitive-file checks.
  Its maintainer Runtime behavioral scenarios remain SKIP/UNVERIFIED.
- `git diff --check`; approved Tasks/Plan/amendment/ADR unchanged against input;
  no module/dependency/historical-fixture change; added-comment review finds only
  the two permitted concise public-API comments.

Resolved local validation failures are retained here rather than hidden:

1. The quality script initially selected system Go 1.24.4 with toolchain switching
   disabled. Re-running with the **already available** Go 1.26.0 binary and pinned
   Staticcheck passed; no gate or dependency pin changed.
2. A Staticcheck invocation overlapped an in-progress source write and observed
   `OperationalObservation` before its declaration file existed. Stable-source
   reruns passed; no suppression/exclusion was introduced.
3. The first complete test run used a private synthetic probe root under `/tmp`.
   Existing envelope guards require a non-temporary private root and refused
   preparation in cmd/lingo and local tests. Reproducing with a dedicated HOME
   root passed the narrow failures and the full suite. No guard was bypassed,
   removed or relaxed.

SKIP / NOT_RUN: dedicated gitleaks (not installed); new proof-free installed
matrix and T02/T03; real vendor inference/auth/E2E and external effects (outside
this authority); native macOS behavioral suite. No result implies vendor
compatibility, confinement enforcement, execution acceptance or AXM-12 acceptance.

Main-session `axiom-review` / `axiom-document`: no blocking finding identified
within this **inactive foundation checkpoint** after fixing inherited effort
filtering. Reviewed generation separation, historical bytes, actual executable
and source drift, auth/API filters, bounded source trust, safe errors/projections,
absence of active successor calls, dead abstractions/comments and scope.
This is not independent review or completion of the approved Tasks. Runtime
admission still requires the old certification path until the complete T02/T03
cutover is implemented and verified.

## Runtime-neutral continuation

```yaml
objective: complete AXM36-T01 then T02 and T03 in existing PR 334
state: in_progress
input_sha: 91b04156a48005d9a7498372ae7990ce4fbb515d
branch: docs/axm-36-runtime-admission-specify
completed_units:
  - r2 authority and repository/provider continuity reconciled
  - new box-local canonical state root created; no external Execution reconstructed
  - inactive configuration and tuple codecs, exact Profile consumer
  - independent identity/auth/effort observations and invocation builders
  - foundation tests, legacy regressions and installed synthetic regression suite
remaining_units:
  - finish T01 downstream field/version ownership and strict successor codecs
  - finish T01 local inventory/publication/history/diagnostic fixtures and review
  - T02 coordinated policy/compiler/preview/start/dispatch/retry/supervisor cutover
  - T03 structural retirement, CLI/skills/docs and proof-free installed matrix
decisions_and_assumptions:
  - no strict lower-model guarantee; no delegation
  - no successor production dispatch before T01 completion and coordinated cutover
  - per-model Claude settings refused within existing JSON roots; no schema guessed
  - external-host canonical Execution/concurrency remains unknown, not absent
authority_granted_and_pending:
  - implementation, tests, commit, normal push and PR 334 update remain authorized
  - no merge, release, closure, Execution acceptance, real inference or external E2E
validation_status:
  - inactive foundations and current-path regression checks PASS as detailed above
  - T01 incomplete; T02 and T03 NOT_RUN; full RA-AC1-10 acceptance pending
next_action: recheck PR head and local state; resume remaining T01 codecs from this inventory
```

The commit containing this Evidence and its exact remote head/CI are recorded
in the PR continuation comment. Continue this PR; do not duplicate Tasks,
Execution or implementation. No further approval is requested for work already
authorized within the approved scope.
