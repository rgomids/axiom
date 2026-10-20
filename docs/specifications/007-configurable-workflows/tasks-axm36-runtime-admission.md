# AXM-36 — Tasks for runtime admission without certification

**Proposed — SDD tasks, revision 1, 2026-10-19.** Three future Tasks with
ordered, bounded checkpoints; none is implemented or accepted here. Human
acceptance of these Tasks and explicit implementation authority are still
required. This artifact changes no production behavior.

## Input, authority and continuity

- [AXM-36](https://linear.app/rgomids/issue/AXM-36/simplificar-runtime-admission-e-tratamento-de-falhas-operacionais),
  [accepted amendment RA-1–RA-6](amendment-axm36-runtime-admission.md) and
  [Accepted ADR-0032](../../decisions/0032-runtime-admission-without-certification.md)
  define the behavior. Preserve the accepted partial supersession of ADR-0025;
  policy, authority, identity, supervision and immutable history remain binding.
- Input is the [approved Plan at exact revision](https://github.com/rgomids/axiom/blob/35f6a28b290df73c4de6e89d2a51d8c37469e86b/docs/specifications/007-configurable-workflows/plan-axm36-runtime-admission.md),
  SHA `35f6a28b290df73c4de6e89d2a51d8c37469e86b`, blob
  `f886fcd73fc517d4b0fd979234314187d487c4fc`.
  [PR approval record](https://github.com/rgomids/axiom/pull/334#issuecomment-6278245748)
  and [Linear approval record](https://linear.app/rgomids/issue/AXM-36/simplificar-runtime-admission-e-tratamento-de-falhas-operacionais#comment-555c8fdf)
  approve plan-to-tasks at that SHA. The Plan's historical Proposed/pending
  wording is preserved: this Tasks-only authority does not edit other artifacts.
- Authenticated remote inspection confirmed local HEAD, tracking branch and
  [Draft/open PR #334](https://github.com/rgomids/axiom/pull/334) at the input
  SHA, clean worktree and branch `docs/axm-36-runtime-admission-specify`.
  Fetch found no change to that branch. GitHub returned exactly one open PR
  for it; the Linear attachment points to #334. Comments include the existing
  Tasks prompt, Plan review without blocking findings and exact approval;
  no competing Tasks artifact/checkpoint was found. Do not duplicate them.
- Linear is `In Review`, with no blockedBy and existing blocks AXM-12/AXM-39;
  these facts are projection, not workflow authority. Neither `axiom` nor
  `lingo` is on PATH. Source inspection of `stateRoot`/`NativeStateRoot` and
  read-only inspection found the effective local state root present, not a
  symlink, empty, with zero JSON records. Canonical Execution on other hosts
  remains **unavailable**, not proven absent. No Execution was created,
  reconstructed, advanced or accepted. Current/target canonical gates and
  derived stages are unknown; reconcile before any implementation transition.
- This phase authorizes only this documentary Tasks artifact, validation,
  commit/push and updating the existing branch/PR. No code, scripts, binaries,
  probes, inference, credentials/infra changes, second PR, merge, release,
  issue closure, status/label/relationship changes or execution acceptance.
  Provider content is used only for the authorized public planning references;
  no operational payload or private configuration is imported.
- No delegation: strictly lower model capability cannot be established from
  runtime controls. Main-session deterministic inspection and review suffice;
  no model/effort override was claimed. Checkpoints below are runtime-neutral.

## Common execution contract and dependency cuts

`Tasks acceptance + bounded implementation authority + reconciled checkpoint
→ AXM36-T01 → AXM36-T02 → AXM36-T03 → technical review → human decision`.

Each Task has one deliverable and its own validation boundary. Its numbered
checkpoints are ordered work within that Task, not independently enabled
product features. Re-read remote head, direct owners, active work and available
canonical Execution before each Task. Record authorized scope, exact input SHA,
commands/results, safe Evidence references and remaining work; resume at the
last verified checkpoint. Never infer an Execution gate from PR/CI/Linear.

T01 adds inactive foundations while current dispatch remains closed to the new
contracts. T02 switches the entire production composition atomically; there is
no registry-only bypass, mixed-generation fallback or enable flag. T02 and T03
must land together before any delivery/release. The existing single PR policy
remains; these Tasks create no task issues or separate implementation PRs.

Preserve Runtime/Project/Workflow/Execution policy, explicit Runtime and binding,
enabled/allowed Profile selection, Profile-owned model/effort, ambiguity refusal,
inspection consent and separate local/external/inference authority. Keep authority
durable before spawn, exact executable path/bytes and configuration revalidation,
trusted ancestors/ownership/no-symlink, bounded parsers, credential filtering,
canonical stores/inbox lineage, leases, locks, limits, cancellation and supervisor
terminal-save ownership. No hidden Runtime/model/auth/API fallback, permission
expansion, automatic repair, substitute certification, new registry/service/store,
new dependency or speculative migration. Required capabilities remain work/effect
policy declarations, never vendor support assertions.

Stop the affected Task for material Plan/SHA divergence, competing implementation,
unavailable required authority or canonical transition facts, schema collision,
or any proposal to weaken safety/history. Resolve mechanical details with Evidence;
material contract changes require reconciliation/human decision before dependent
code. Do not use a universal compatibility claim to conceal an unresolved source.

Rollback before new-state adoption is a coordinated code revert. After adoption,
preserve records, stop/observe and reconcile live/partial/unknown work, then
explicitly retire/reconfigure for the chosen binary with its authority. Never
mutate/downgrade history or redispatch an attempt to implement rollback.

## AXM36-T01 — Inactive contract and observation foundation

- **Objective/deliverable:** reviewed proof-free successor codecs and direct
  operational observations, with deterministic fixtures; new dispatch is still
  unreachable. Implements Plan D1–D4 and the D5 storage contract foundation.
- **Dependencies:** accepted Tasks, explicit T01 authority, reconciled input and
  field/version ownership. Review the two inventories below before active wiring.
- **Confirmed owners:** `internal/runtimeprofile/{runtimeprofile,v2,v3,v3_consumer}.go`;
  `internal/runtimeadapter/{adapters,authpreflight,stage_invocation,frozen_stage,effort_sources,effort_inspector,effort_applicability,stage_effort}.go`;
  `internal/workflow/configured.go`, `internal/workflowcompiler/{compiler,stage_plan}.go`;
  `internal/executiongraph/{successor,single_stage,dispatch_inputs}.go`;
  `internal/local/{runtime_profile_store,runtime_profile_authoring,graph_store,successor_graph_store,single_stage_store}.go`.
  New files/fixtures belong beside these existing owners, without a new layer.

Ordered checkpoints:

1. Freeze the closed-schema inventory from current struct tags, strict decoders,
   canonical projections and writer/read callers. Record complete retained fields,
   required/optional semantics, bounds, digest membership, historical versions and
   fixture names in T01 Evidence before code. Use the version/delta table below;
   recheck next-unused numbers and explicitly reconcile collisions. No empty/fake
   proof, declared precedence or synthetic version values in successor records.
2. Add strict successor encode/decode/digest paths and historical read dispatch.
   Existing fixtures/bytes/digests remain unchanged. Current authoring remains
   on the old path until T02; inactive successor writers are tested directly.
   Preserve monotonic revision/tombstone digests, locks and interrupted-publication
   recovery. Old active configuration requires explicit retirement/reconfiguration,
   never implicit conversion. Isolate historical proof references as opaque data;
   reading status/history must not need an old proof store or gain dispatch rights.
3. Add direct executable/configuration/effort observations within existing adapter
   owners, with the source inventory below. Keep old active consumers intact until
   T02. Canonical explicit encoding is Profile-derived; runtime-default adds no
   selector or claimed vendor-effective effort. Reject unsafe tokens and recognized
   competing sources even with equal values. Unknown versions/unrelated settings
   alone pass. Inspect actual read-only inbox CWD and compare repository/child
   project sources for repository-write, including bytes/presence and safe anchors.
4. Separate optional version metadata from actual auth observation and invocation
   choice. Unknown/malformed/nonzero version continues to real auth status; deadline,
   cancellation, identity drift/disappearance, unsafe config and unobservable auth
   still fail at their boundary. One reviewed invocation per Runtime/operation,
   no version-based flags or fallback. Keep bounded separately consented status
   runner, identity checks around subprocesses and credential/API override filters.
   Login remains login-only; inference/billing is unconfirmed.

### T01 format/fixture inventory

Successors below consume one generation each, as approved in D2. Keep Workflow
definition schema 2, Project policy, coordination and dispatch-authority receipt
bodies unchanged when their body/meaning is unchanged. Receipts bind the exact
new preview digest. Do not silently extend a historical closed schema.

| Owner/current → successor | Closed delta and required fixtures |
| --- | --- |
| Configuration 3 → 4 | Top-level `formatVersion`, `revision`, `runtimes`, `modelProfiles`, optional `preferences`; Runtime retains `id`, `adapter`, `enabled`, `allowlistedProfileIds`, removes `effortDeclarations`. ModelProfile/preference field semantics remain. Round-trip active/retired, absent vs tombstone, revision conflict, interrupted removal/publication, historical 1/2/3 explicit refusal for active use |
| Configured Execution 3 → 4 | Proof-free resolved tuple: `profileId`, `runtimeId`, `model`, `effort`, `profileDigest`, `configurationRevision`, `configurationDigest`, `executableDigest`, `invocationMode`, `observationRevision`, `observationDigest`, `effortSourceDigest`; no `capabilityProofReference` or required `runtimeVersion`. Optional version belongs only to diagnostic metadata, outside eligibility/digest equality. Preserve remaining owner fields and frozen binding. Old/new digest vectors and stale-preview fixtures |
| Successor Graph 3 → 4; single run 1 → 2 | Embed that tuple, preserve scheduler/authority/result semantics; closed optional sanitized attempt diagnostic summary per D5. Absent-summary and each effect-certainty fixture, atomic save/crash, old graph/single inspection without store, no historical retry/continuation |
| Compiler result 2 → 3; StagePlan 2 → 3 | Same proof-free tuple/rights, remove active `capabilityEvidenceRef`/proof-derived resolution, retain required effects, workflow snapshot, plan references and authority ceiling. Snapshot/plan/digest deterministic fixtures; historical envelope refusal at new apply |
| DispatchInputs 1 → 2 | Retain `formatVersion`, `projectId`, `parentId`, `previewDigest`, `preview`; new payload semantics use fresh proof-free preview. Old digest cannot apply under successor decoder; byte-stable historical read vectors |
| Effort-source canonical projection | Adapter-rule identity, selected intent, executable/mode identity and sorted sanitized source facts (presence, safe key/category, location binding); no Runtime version, proof or precedence declaration. New digest vectors; old projection vectors unchanged; selector/presence/child drift invalidates preview |

For every successor add fixtures for unknown/future version, missing required
field, unknown field, duplicate key/case alias, corrupt/truncated/oversize input,
invalid digest and cross-generation misuse. Put new fixtures under the owning
package's `testdata` (proposed `axm36` subdirectory); extend existing
`internal/executiongraph/testdata/successor/historical-digests.json` assertions
without rewriting historical vectors. Existing workflowcompiler issue306 and
stable compatibility corpus stay unchanged. Extend local state inventory and
compatibility coverage per FR-026 and the quality policy; no release corpus
freeze or release action is authorized by these Tasks.

### T01 bounded effort-source inventory and primary evidence

Inspected production code currently hardcodes Codex `model_reasoning_effort`;
Claude JSON selector names come from declarations and tests use
`fixtureEffortKey`. That fixture is **synthetic**, never a production key.
The following fixed adapter rules replace declarations, not a new user catalog:

| Runtime | Canonical selector; recognized conflicting sources within existing boundary |
| --- | --- |
| Codex | `--config model_reasoning_effort=<Profile token>`; competing `--config`/`-c` assignments and parsed `model_reasoning_effort` settings in user/project `.codex/config.toml`, managed `/etc/codex/config.toml` and `/etc/codex/managed_config.toml`. Preserve nested-table/inline-selector detection in existing TOML parser; test present/absent/malformed sources and inherited selectors recognized by current argv/environment validation |
| Claude | Exact child env `CLAUDE_CODE_EFFORT_LEVEL=<Profile token>`; conflicting inherited env or settings `env.CLAUDE_CODE_EFFORT_LEVEL`, argv `--effort`, JSON `effortLevel` in user/project `.claude/settings.json`, project `.claude/settings.local.json`, managed `managed-settings.json` and bounded `managed-settings.d/*.json`. Managed anchor stays `/etc/claude-code` on Linux or `/Library/Application Support/ClaudeCode` on macOS, per existing adapter |

Primary references inspected on 2026-10-19:
[Codex configuration reference](https://developers.openai.com/codex/config-reference/),
[Codex configuration locations](https://developers.openai.com/codex/config-basic/),
[Claude effort configuration](https://code.claude.com/docs/en/model-config) and
[Claude settings sources](https://code.claude.com/docs/en/settings).
They corroborate keys/locations, not availability, exhaustive source detection
or vendor precedence enforcement. Codex documentation now describes additional
profile/cloud layers; Claude documentation describes per-model effort settings
and caps. They are not permission to discover arbitrary roots, execute help/probes
or add support matrices. T01 must record whether any such concrete selector is
reachable through the existing invocation/source boundary; recognize it there or
return a bounded configuration conflict, without making an unfamiliar version
or unrelated unknown setting a blocker. If the bounded rule cannot preserve
one-owner intent without expanding trust, stop for a material decision.

Retain explicit refusal of `CODEX_HOME`/`CLAUDE_CONFIG_DIR` relocation, trusted
ancestors, ownership/no-symlink, independent roots, parser size/count bounds and
no auth/credential-file reads. Do not claim universal coverage of future keys.
No per-install qualification, receipt or required vendor inspection is added.

- **Done:** inventories and fixtures reviewed; strict new formats cannot be
  mistaken for old authority; historical status/stop/reconciliation remain safe;
  direct auth/effort/identity functions are proof-free and tested for both adapters;
  no public command can dispatch with partly converted contracts.
- **Tests/Evidence:** targeted runtimeprofile/runtimeadapter, workflowcompiler,
  workflow, executiongraph and local suites. Capture old-byte/digest comparisons,
  version/auth independence, explicit/default effort, equal-value conflicts,
  inbox/child drift, relocation/symlink/bounds refusals and counters showing zero
  attempt spawn for refused pre-spawn checks. Status subprocesses are counted
  separately from inference children. Publish field/version and dependency
  inventories, commands/outcomes and unresolved source limitations. RA-AC2/3/5/9
  foundations and RA-AC10 schema checks are obligations, not acceptance here.

## AXM36-T02 — Coordinated admission, diagnostics and recovery cutover

- **Objective/deliverable:** one usable, supervised proof-free production path
  across resolution → workflow preview/start → stage plan/preview/apply/dispatch
  → retry/supervisor, preserving every independent control and D5 diagnostics.
- **Dependencies:** T01 complete and reviewed; explicit T02 authority and fresh
  checkpoint. T03 is a mandatory delivery dependency, not an optional cleanup.
- **Confirmed owners:** `internal/runtimeapplication/policy.go`;
  `cmd/lingo/{runtime_policy,execution_target,workflow_stage_plan,stage_dispatch,capability_composition,stage_service,stage_supervisor,stage_single}.go`;
  compiler/configured Workflow/adapter/profile owners in T01;
  `internal/stageexecution/{preview,run}.go`, `internal/executiongraph`,
  `internal/local` graph/single/workflow persistence;
  `internal/completion`, `internal/cli/{completion,presentation}.go` and existing
  AXM-31 diagnostic transport. Reuse these boundaries, no parallel error framework.

Ordered checkpoints:

1. Switch authoring and shared policy resolver to configuration 4 and direct
   observations; wire compiler, immutable configured Execution and stage-plan
   successor contracts together. Remove CapabilityProven/ProfileProof loops,
   version allowlist/invocation branches, proof-enriched observation and frozen
   proof copying. Required effects still constrain intent and authority. Do not
   swallow generic blocked errors. Historical read dispatch cannot flow into
   new active start/run/retry; explicit retirement/reconfiguration is required.
2. Switch graph and single preview/apply, detached supervisor and each retry to
   the same new tuple/auth/effort semantics. Reobserve each attempt, retain exact
   preview and child-view binding, durable authority before child start, ceilings,
   leases/locks/budgets/cancellation/final inbox drain and atomic terminal save.
   Remove proof-conditioned confinement admission while preserving actual
   filesystem/process/inbox controls and mandatory permission/sandbox arguments.
   Disclose advisory/unknown confinement honestly, never label it os-enforced
   from a synthetic process. No product route may reach partial conversion.
3. Attach D5 structured diagnostics at pre-spawn and terminal-save boundaries.
   Use the Plan's codes: `executable_unavailable`, `spawn_failed`,
   `invocation_incompatible`, `authentication_unavailable`, `configuration_unsafe`,
   `configuration_invalid`, `permission_denied`, `effort_override_conflict`,
   `model_unsupported`, `effort_unsupported`, `attempt_timeout`, `attempt_cancelled`,
   `subprocess_failed`, `result_missing`, `result_invalid`, `effects_uncertain`.
   Preserve existing authority/stale-preview/policy/format/recovery codes and
   completion status/exit mapping. Diagnostics carry operation/attempt correlation,
   boundary code, confirmed cause or labelled hypothesis, certainty
   `none|known|partial|unknown`, sanitized Evidence refs, next action and required
   authority. Unknown stderr/exit alone yields generic failure, not guessed cause.
   Bound recognized patterns; discard raw stderr/transcripts/config/env, secrets,
   account identifiers and host paths before storage or JSON/text presentation.
4. Persist summary with the attempt's existing atomic terminal publication.
   Supervisor classifies once; confirmed auth/permission failure after spawn does
   not imply no effects. CLI dispatch acknowledgement is not terminal success;
   history/status read persisted outcome. Exit 0 without valid qualifying result
   remains non-success. Unknown/partial/live/orphan work blocks repetition;
   reconcile appends facts without rewriting outcomes. Retry is a new budgeted
   attempt after fresh policy/auth/identity/effort and separate authority, never
   automatic redispatch, credential repair or Runtime/model/API substitution.

- **Done:** both adapters and graph/single paths share one active contract;
  historical attempts remain inspectable/reconcilable without new dispatch rights;
  all pre-spawn negative controls yield zero attempt spawn; valid authorized
  synthetic attempts succeed without registry/store/receipt. Failures preserve
  process result, contract result and effect certainty independently; operator/AI
  receive matching safe facts and recovery actions.
- **Tests/Evidence:** extend existing runtime policy/compiler/binding/stage
  preview/apply tests and installed single/graph journeys with arbitrary versions,
  new intent before preview, identity/Profile/source drift afterward, disabled or
  denied/mismatched/ambiguous selection and each missing authority independently.
  Inject option/model/effort rejection, permission/auth failure before and after
  spawn, unrecognized stderr/exit/signal, missing/invalid result despite exit 0,
  timeout/unobserved stop, supervisor crash, partial writes and lost result.
  Assert counters, authority ordering, unchanged canonical state on rejected work,
  live-process retry refusal, append-only reconciliation and fresh retry budgets.
  Seed adversarial secret/config/path/transcript payloads; assert absence from
  persisted summary/Evidence and both output modes. Record RA-AC1–10 results per
  adapter/path, plus deliberate NOT_RUN/SKIP reasons. No real vendor execution.

## AXM36-T03 — Structural retirement and installed journey closure

- **Objective/deliverable:** remove the disconnected certification subsystem and
  obsolete interfaces; complete the documented/operator/packaged-skill journey,
  with full installed synthetic and regression Evidence.
- **Dependencies:** T02 complete, explicit T03 authority and fresh checkpoint;
  T02–T03 jointly required before delivery. Preserve T01 historical readers.
- **Confirmed owners:** `internal/capabilityproof`,
  `internal/runtimeadapter/{version_registry,binding_source}.go`,
  `internal/runtimeadapter/runtime-versions.json`,
  `internal/local/{capability_store,capability_selection}.go`,
  `cmd/lingo/*capability*`, `internal/cli/runtime_capability.go`,
  `internal/cli/runtime_authoring.go`, `cmd/lingo/runtime_profile_v3.go` and
  declaration/migrate handlers; CLI help/completion/tests, `docs/commands.md`,
  `internal/codexruntime/skills/{axiom-runtime,axiom-work-item}/SKILL.md`,
  existing runtime skill packaging/tests, `cmd/lingo/*blackbox_test.go`.
  Inspect direct docs/contracts/skills consumers, including canonical maintainer
  skills if affected; update only prospective guidance. Preserve dated history.

Ordered checkpoints:

1. Delete active certification service/observer/selector/probe/canary/help
   classifiers/envelopes, embedded registry, proof stores and composition wiring
   with no remaining valid owner. Isolate only actually needed historical
   read-only codecs; enumerate each remaining reference and why it cannot admit
   or dispatch. No active imports of certification service/registry. No startup
   deletion, GC, invalidation, rewrite or automatic import of old Evidence/store.
2. Remove all `runtime <codex|claude> capability` routes (`show`, `invalidate`,
   `prepare`, `probe`, `probe-extended`), service methods, parser/help/completion
   and flags. Retire `--effort-declaration`, file ingestion/declaration handlers
   and migration targets emitting obsolete formats. Ordinary invalid-command/
   option guidance has zero subprocesses and zero store writes, even with hostile
   obsolete arguments. Preserve binding/Profile CRUD authority and revision checks.
3. Replace mandatory qualification guidance with inspect → authorize → execute
   → observe → diagnose/recover in CLI/commands and packaged skills for both
   runtimes. Thin skills repeat persisted facts, hypotheses, history refs and
   next required authority; they cannot repair, mutate canonical stores, infer
   acceptance or turn login/synthetic success into vendor support. Reconcile
   direct active proof wording while retaining historical Specs/ADR annotations.
4. Finish installed synthetic journeys using the packaged binary and actual
   packaged skills from an unrelated temporary CWD outside checkout. Extend
   `TestInstalledSyntheticSingleStageJourney`,
   `TestInstalledSyntheticGraphStageJourney`, retry/reset, supervisor crash and
   reconcile tests. Resolve through production composition/adapter: no injected
   Proven inventory or mock resolver bypass. Fresh isolated state/home, no
   registry/proof store/probe receipt/declaration, synthetic status/attempt
   executables only; use spawn/effect counters and bounded outcomes. Verify
   installed help and both runtime packages omit mandatory qualification. Re-run
   positive/negative matrix and history/security/result regressions after deletion.

- **Done:** structural search contains only enumerated historical/read-only
  certification references; all obsolete CLI paths are effect-free refusals;
  active guidance matches installed behavior; RA-AC1–10 matrix has deterministic
  Evidence for both adapters and each applicable graph/single path. No claim of
  real vendor compatibility/confinement or AXM-12 acceptance.
- **Tests/Evidence:** installed suites in `cmd/lingo/stage_journey_blackbox_test.go`,
  `stage_adversarial_blackbox_test.go`, `stage_dispatch_test.go`,
  `stage_fake_runtime_test.go`; `internal/cli/output_separation_test.go`,
  completion/presentation and packaged-skill tests; full checks below. Preserve
  security/result tests when deleting proof-gate tests; obsolete inputs remain
  historical/refusal fixtures only. Publish removed-producer/consumer inventory,
  categorized residual search, old-record no-write checks, packaged binary/skill
  provenance, unrelated CWD proof, per-adapter results and unverified limitations.

## Acceptance-to-task/test/Evidence map

Every applicable case runs through **Codex and Claude**, and **single and graph**
where those paths exist. A non-applicable combination needs an explicit reason;
blank rows are not success. All behavioral rows below are **NOT_RUN in Tasks**.

| AC | Tasks/checkpoints | Required validation and inspectable Evidence |
| --- | --- | --- |
| RA-AC1 | T01 codecs; T02.1–2; T03.4 | Installed workflow preview/start, stage plan/run outside checkout; no registry/proof/declaration/receipt; real production resolver/adapter, synthetic valid result and counters |
| RA-AC2 | T01.2–4; T02.1–2 | Arbitrary/unknown version and new model/effort before preview pass; path/bytes/Profile/config/source drift after preview refuses with zero attempt spawn; version string alone cannot stale equal identity |
| RA-AC3 | T01.1–3; T02.1–2 | runtimeprofile consumer, runtimeapplication, compiler profile-chain, binding/policy tests; disabled/denied/mismatch/ambiguity each concrete, deterministic and no fallback |
| RA-AC4 | T02.2; T03.4 | Stage preview/apply/dispatch and installed tests omit local/external/inference authority separately despite login; zero unauthorized child/effect and durable authority precedes positive spawn |
| RA-AC5 | T01.3–4; T02.3; T03.4 | Auth/effort/path suites and installed negatives: missing/expired/unobservable auth, absent executable, unsafe roots/symlinks, malformed/unreadable config, permission/override; actual boundary, no repair |
| RA-AC6 | T01.4; T02.3–4; T03.4 | Invocation/fake child/installed rejection and exit/signal cases; confirmed vs unsupported cause, preserved process result, no alternate dispatch |
| RA-AC7 | T02.4; T03.4 | Graph/single classification/inbox, exit 0 missing/invalid result; non-success contract and validator reference, no success from login/prose |
| RA-AC8 | T01.2; T02.2/4; T03.4 | Supervisor/recovery/store and installed timeout/crash/partial/lost-result cases; persisted certainty, no redispatch, live-process refusal, append-only reconciliation and budgeted authorized new retry |
| RA-AC9 | T01.3; T02.2; T03.4 | Explicit/default effort, equal-value conflicts, read-only inbox and repository-write child drift; rejected canonical/out-of-scope writes leave state unchanged; actual control/unknown confinement disclosed |
| RA-AC10 | T01 diagnostic schema; T02.3–4; T03.3–4 | CLI separation/completion, attempt summary and packaged-skill tests: seeded payload absent in JSON/text/persisted Evidence, bounded correlation and labelled hypotheses, same next action/authority |

## Validation commands and evidence protocol

Future implementation checks (not run in this documentary phase):

```bash
go test ./internal/runtimeprofile ./internal/runtimeadapter ./internal/runtimeapplication ./internal/workflowcompiler ./internal/workflow ./internal/stageexecution ./internal/executiongraph ./internal/local ./internal/cli ./internal/codexruntime ./cmd/lingo
go test ./cmd/lingo -run 'TestInstalledSynthetic' -count=1
go test ./...
go vet ./...
go build ./...
go mod verify
./scripts/validate-repository.sh .
```

The installed harness must actually place binary/skills and change CWD outside
checkout; invoking `go test` alone does not prove that property. Confirm test
selection lists each intended installed success/adversarial/recovery case; use
its exact names if outside the prefix above. No test may execute an installed
real Codex/Claude or touch operator state/credentials. Synthetic external effects
are isolated fixtures, never provider mutations. Keep deterministic secret/path
assertions and bounded subprocess cleanup.

For each Task record exact input/output commit, changed paths, checkpoint,
command/exit/result, fixture/test names, matrix cells, sanitized Evidence refs,
security review and limitations. Distinguish attempted command, CLI completion,
persisted attempt outcome and human acceptance. Run dependency searches for
`capabilityproof`, `CapabilityProven`, `ProfileProof`, `CapabilityEvidenceRef`,
`CapabilityProofReference`, `runtime-versions`, `effortDeclarations`,
`effort-declaration`, `ApplicableEffortDeclaration`, `ObserveEffortApplicability`,
`precedenceEstablished`, `ProofReference` and retired CLI routes. Enumerate
historical/negative-fixture hits instead of asserting a repository-wide zero.

Main-session engineering/security review checks Plan D1–D5/RA-1–RA-6, strict
history, independent authority, result/effect classification and scope. Any
blocking finding prevents the Task from being reported complete. Documentation
checks/CI prove structure only; real Codex/Claude E2E, inference, external effects
and product acceptance remain [AXM-12](https://linear.app/rgomids/issue/AXM-12)
/ [#278](https://github.com/rgomids/axiom/issues/278) under separate authorization.

## Tasks-phase Evidence and next gate

Only this file is changed by this phase. Plan/Spec/ADR/code remain untouched.
Executed documentary checks: `./scripts/validate-repository.sh .` PASS
(including 33 ADRs, 56 ADR-governance fixtures and automation/sensitive-file
checks; Runtime behavioral scenarios explicitly SKIP/UNVERIFIED);
`python3 scripts/check-adr-governance.py .` PASS. Ephemeral deterministic checks
passed for three complete Task contracts, ten unique RA-AC rows, local links
and the unchanged approved Plan blob. Main-session semantic/security review
against D1–D5 and RA-1–RA-6 found no blocking finding in this documentary scope;
it is neither independent review nor human acceptance.
Staged checks and publication results are recorded in PR #334 at the published
SHA; behavioral RA-AC1–10 tests remain NOT_RUN. No canonical Execution
transition, inference, probe or credential inspection occurred. Dedicated
`gitleaks` is unavailable; repository sensitive-file checks are not a replacement.

Next action: technical review and human acceptance of Tasks at the published
revision. Implementation additionally needs explicit bounded authority and a
reconciled current checkpoint; neither this proposal nor green CI supplies it.
D4 future vendor-source completeness is an explicit bounded T01 prerequisite;
canonical Execution availability is an implementation-transition prerequisite.
There is no blocking divergence from the approved Plan in the inspected input.
