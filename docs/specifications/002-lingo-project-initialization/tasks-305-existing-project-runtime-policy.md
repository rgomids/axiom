# AXM-34 / G-1 — Existing Project Runtime/Profile policy Tasks

Status: **Proposed — Awaiting Human Approval — Tasks**, revision 1, 2026-10-18.
All Tasks **Not started**; every product validation below is **NOT RUN**.
Technical owner: [#305](https://github.com/rgomids/axiom/issues/305), parent
[#303](https://github.com/rgomids/axiom/issues/303); operational record:
[AXM-34](https://linear.app/rgomids/issue/AXM-34). Reuse
[Draft PR #327](https://github.com/rgomids/axiom/pull/327) and branch
`docs/axm-34-project-runtime-policy-specification`; no new Work Items or PR.

## Authority, fixed inputs and discovery

Input: [G-1 amendment](amendment-305-existing-project-runtime-policy.md),
G1-AC01–15, and [technical Plan](plan-305-existing-project-runtime-policy.md)
at exact SHA `c2118649803a086cab67588a6d7547ac721e7a71`.
The maintainer explicitly approved that Plan for Tasks in the
[GitHub receipt](https://github.com/rgomids/axiom/pull/327#issuecomment-6246039562)
and [Linear receipt](https://linear.app/rgomids/issue/AXM-34/g-1-editar-politica-de-runtime-e-model-profiles-de-project-existente#comment-7eae4598).
The Plan's pending-approval text is historical at those fixed bytes; this artifact
records the subsequent approval without rewriting the approved Plan/Specification.
Technical [review round 2](https://github.com/rgomids/axiom/pull/327#pullrequestreview-5561555695)
and CI are separate from human approval. Specification approval remains fixed to
`6a5b00e43e7a575bfde4e25d2a3110602d9a414c` and its existing receipt.

Pre-write discovery on 2026-10-18:

- Clean local HEAD, remote branch and PR head equal the approved Plan SHA;
  no subsequent material Plan change. Read #305, #327 comments/reviews/diff,
  Linear issue/relations/comments and branch history. Only #327 belongs to
  AXM-34 among the three open PRs (#317/#328 are unrelated).
- Repository artifact/history search found no G-1 Tasks to resume. Existing
  Spec 002 Tasks and #306 local Profile Tasks have distinct owners and remain
  unchanged. No implementation or completed G-1 Execution was demonstrated.
- CI run `39099896455`, `PR Checks`: **IN_PROGRESS** at discovery;
  delivery-metadata checks: **SUCCESS**. Its artifact API returned zero artifacts
  at inspection; absence during an active run proves no product behavior.
- Canonical Execution ID/revision/current gate and current/expected derived
  lifecycle stage: **unknown**. No canonical endpoint is exposed and no `.axiom`
  ledger exists in this checkout. Provider history contains prompts/receipts,
  not canonical Execution truth. No installation search or product invocation
  was used to reconstruct it. Before any future transition, recover exact local
  truth and competing Execution state; do not infer it from this Tasks proposal.
- Public repository visibility verified. Technical content derives from public
  GitHub/versioned contracts; only the user-authorized approval reference from
  Linear is linked, without importing private incident narratives/raw chat.

Governance: [AGENTS](../../../AGENTS.md), [constitution](../../product/constitution.md),
[tracking policy](../../development/linear-github-issue-tracking.md),
[security policy](../../../.agents/policies/security.md),
[documentation governance](../../documentation.md). Linear owns operational
status/assignee/blocking; repository artifacts own technical truth; canonical
Execution owns workflow truth. None is mutated by producing these Tasks.

Preserve [policy v2](runtime-policy-v2.md), [bootstrap v3](project-bootstrap-v3.md),
[Spec 002](spec.md), [Spec 007](../007-configurable-workflows/spec.md),
[HD-006](../007-configurable-workflows/amendment-303-workflow-skill.md),
[lifecycle matrix](../004-mvp-v1-baseline/issue-230-resource-lifecycle-matrix.md),
ADRs [0005](../../decisions/0005-bounded-local-filesystem-threat-model.md),
[0007](../../decisions/0007-local-publication-and-recovery-protocol.md),
[0020](../../decisions/0020-workflow-definition-revision-binding.md),
[0022](../../decisions/0022-dedicated-workflow-conversational-surface.md) and
[0024](../../decisions/0024-model-profile-effort-ownership.md), including its
accepted format-3 amendment. No accepted contract/ADR conflict identified.
If implementation cannot preserve the Plan's guards, wire compatibility or
ownership, stop the affected chain for clarify/human decision; never weaken
the Specification or infer a new API/version/permission.

## Execution boundaries and order

These are future bounded implementation units, not authorization to execute them.
Human review and explicit approval of the exact Tasks revision, followed by
implementation authority for the selected scope, are required before code.
No acceptance, merge, release, AXM/#305 closure, Execution transition, Provider
status change or automatic successor authorization follows from any check.

Global non-goals for every Task: no new registry, Doctor, resolver, command
family, workflow engine, migration, dispatch, live Runtime/auth/network probe,
credential access, local Profile authoring, Workflow authoring/selection,
Execution/Stage Plan rewrite, Work Item mutation or Provider effects. Axiom !=
Lingo; Project != Repository; Role != Model; Runtime != Model; Evidence != raw chat.
Future production edits are limited to the existing G-1 extension seams below;
local configuration/Workflow writer coordination changes may only enforce shared
locking, never change their configuration or authoring semantics.

| Task | Approved Plan unit | Depends on | Bounded result |
| --- | --- | --- | --- |
| G1-T01 | A | Approved Tasks + implementation authority | Strict draft and complete pure candidate |
| G1-T02 | B | T01 | Exact portable Workflow compatibility |
| G1-T03 | B | T01, T02 | Read-only readiness and safe deterministic preview |
| G1-T04 | C | T01, T02, T03 | Commit dependency/coordination guard |
| G1-T05 | C | T04 | Exact replay, publication and recovery classification |
| G1-T06 | C | T05 | Observed read-back and truthful completion |
| G1-T07 | D | T01–T06 | Existing CLI EDIT transport and guided parity |
| G1-T08 | D | T07 | Thin conversational route and inspection |
| G1-T09 | E | T01–T08 | Integrated regression/security/native Evidence |

Default order is the table order. No artificial independent delivery fronts:
T02/T03 share application contracts; T04/T05 share local publication; T07/T08
share adapter inspection. After T07 passes, T08 routing and T09's read-only
regression/native collection may overlap on disjoint artifacts; T09's final
consolidation waits for T08. Read-only review/test-case design can overlap with
upstream work, but no downstream merge or overlapping-file mutation precedes
passed prerequisites. Stop dependent work on upstream failure. All Tasks include
their own tests and security review; T09 consolidates rather than defers testing.
No subagents used here: strictly lower model hierarchy could not be established.
Future delegation remains capability-based, minimum sufficient effort, bounded
context/output and verifiable controls, with integration owned by the Orchestrator.

## Task contracts

Existing symbols below are extension points inspected at the approved SHA.
Draft transport, policy projection and dependency-bundle extensions are future
work, not interfaces claimed to exist. Test labels V01–V09 are required oracles,
not invented test functions or recorded results.

### G1-T01 — Strict partial policy draft and complete candidate (A)

- **Objective/scope:** extend `EditIntent` with the Plan §3 policy-only intent;
  strict bounded JSON decoding, pure partial merge, whole-candidate validation,
  canonical semantic diff and no-op. No raw manifest patch or new serializer.
- **Components/contracts:** `internal/projectapp/{edit,snapshot,ports}.go`,
  `internal/project/{project,runtime_policy}.go`,
  `internal/manifest/{codec,schema_v2,mapping,output,security}.go`;
  `Project.Propose`, `Equivalent`, `ReadSnapshot`. Review/reuse an existing strict
  JSON primitive; transport spelling is recorded with CLI metadata in T07.
- **Done:** omitted groups preserve state; explicit `absent`, `unconfigured`,
  `replace(values)` and `members(add/replace/remove)` obey closed mutual-exclusion
  rules. Exact Runtime ID/Profile key/preference role+complexity targeting;
  add requires absence, replace/remove existence, repeated/conflicting targets
  refuse, rename is explicit remove+add with dependent edits. Adding to absent/
  unconfigured starts empty; last-member removal yields configured `[]`.
  Sort exact bytes without trimming/ranking/defaults. Encode/decode the complete
  candidate, preserving UUID, slug/name, schema, context, selection, unrelated
  declaration forms and companion documents/digests. A semantic no-op retains
  original manifest/local bytes/revisions despite formatting differences.
- **V01 tests/Evidence:** table/property tests for all collection forms and
  one-member changes in each group, multiple named Profiles for one Runtime/model,
  explicit dependency repair and same-schema v2/v3 round trips. Accept final
  8/32/32; reject 9/33/33, duplicate/unknown JSON keys, singular `runtime`, malformed
  shapes, unsafe tokens/models, dangling or unconfigured references, missing/
  conflicting targets, Runtime removal with retained Profiles and Profile removal
  with retained preferences. Refuse mixed unrelated EDIT intent. Bound input to
  256 KiB/depth 32/16,384 nodes and 16/64/64 member operations; test exact bounds
  and overflow, no truncation/raw rejected-value echo. G-1 refuses v1/v4/newer
  schemas without changing shared schema-4 helpers. Prove zero I/O and unchanged
  unrelated fields/companion bytes, not merely a matching edited member.
- **Non-goals/dependencies:** global boundaries; no CLI wiring, publication,
  local-candidate seeding or local readiness save gate. First Task; subsequent
  tasks consume its complete immutable candidate and unchanged-source snapshot.

### G1-T02 — Exact Workflow compatibility and immutable impact (B)

- **Objective/scope:** pure before/after compatibility against the actual active
  definition, with complete exact-reference impact; no new Runtime selector.
- **Components/contracts:** `internal/projectapp/{active_workflow,workflow_authoring,workflow_profiles_v3}.go`,
  `internal/local/workflow_definitions.go`, `cmd/lingo/active_workflow.go`;
  `ResolveActiveWorkflow`, existing strict catalog and definition digest rules.
  Inspect `internal/workflowcompiler/compiler.go` request/`profileID` mapping for
  regression parity. Extract only the reusable portable helper; preserve existing
  authoring gates. `ValidateWorkflowProfilesV3` alone is not a portable save gate.
- **Done:** bind exact Workflow ID/revision/digest/source/schema/index or explicit
  none. Analyze every Stage/Agent, retained and changed references, Runtime
  constraints, role/complexity/capabilities with Plan §5 precedence: explicit
  Profile cannot be overridden by preferences; historical `policy-default`
  uses exact preference then admissible portable set, never first-match fallback.
  Local capability/model/effort evidence remains readiness. Preserve Workflow v1
  semantics and schema-2 Profile-owned effort. Both new and unresolved pre-existing
  incompatibility block apply. Sort full safe diagnostics by logical references;
  reuse input bounds and bounded output, refusing rather than dropping rows.
- **V02 tests/Evidence:** built-in immutable and Project revisions, none, missing/
  unreadable/tampered/unresolvable selection, stale digest/index, constraints,
  preference changes, retained references, explicit/default precedence and multiple
  admissible keys. Assert exact Workflow/Stage/Agent/Profile/Runtime before/after
  reasons and overflow refusal. Snapshot fixtures demonstrate four impact scopes:
  fresh future admission; immutable existing Execution/Stage Plan bindings whose
  dispatch/resume still checks permission/freshness; zero Work Item effects; other
  Projects/local registry unchanged. Inventory is unknown/uninspected, with no
  fabricated counts; no Workflow or Execution write.
- **Non-goals/dependencies:** T01; no inventory engine, Workflow migration,
  selection repair/default replacement, compiler/dispatch behavior change or
  capability inference from model IDs. Output feeds T03 and T04.

### G1-T03 — Proposed-snapshot readiness and sealed safe preview (B)

- **Objective/scope:** compose the candidate with retained bindings and safe
  existing local observations through existing readiness, then seal an exact
  deterministic preview; dry-run remains read-only preview.
- **Components/contracts:** `internal/projectapp/{readiness,edit,admission,project_state}.go`,
  `cmd/lingo/{readiness,runtime_policy}.go`,
  `internal/runtimeapplication/{policy,policy_v3}.go`, `internal/runtimeprofile`;
  `ProjectReadiness`, `EvaluateReadiness`, `EvaluatePolicyV3`, `AdmitProjectEdit`.
  Reuse/factor read-only evaluation; do not invoke side-effecting observers.
- **Done:** evaluate the proposed snapshot instead of accidentally reloading old
  policy by UUID. Distinguish portable validity, exact Workflow compatibility,
  administrative admission, publication and operation-scoped readiness. Valid
  compatible policy may be saved despite local execution blockers. Corrupt or
  unreadable required snapshots block exact preview; observed absence is explicit.
  Seal operation/scope, UUID/source identity, actual manifest/artifact snapshot,
  installation wire/revision, operational state, complete candidate/diff, exact
  Workflow/none, local configuration/observation revision+digest/absence tokens,
  diagnostic/impact snapshot and allowed write set. Exclude fresh timestamps/
  correlation IDs from identity; never invent revisions for read-only facts.
  Expose a safe G-1 projection, not legacy full-manifest/path `EditPreview`.
- **V03 tests/Evidence:** identical intent/snapshots yield identical digest and
  dry-run/`project validate` semantics. Absent/retired/disabled/wrong model or
  Runtime, unavailable executable/auth/capability proof and ambiguous candidates
  retain portable declarations with exact blockers and no fallback. Work Item
  readiness excludes irrelevant execution blockers. Spies reject all network,
  Runtime launch, Provider, secret-value reads and preview writes/lock creation.
  Secret/path/error sentinels in input, legacy preview fields, unrelated credential
  references and local bindings must not reach output/logs/Evidence. Keep stable
  `runtime_policy_unavailable`/`runtime_resolution_blocked` and inner reasons;
  new fixed bounded reasons must retain diagnostic scope. Preserve local format-3
  effort and historical formats without migration or executable proof acquisition.
- **Non-goals/dependencies:** T01+T02; no Doctor, auth probe, secret resolution,
  new resolver, installation refresh or gate requiring execution readiness.

### G1-T04 — Guard every consulted dependency through commit (C)

- **Objective/scope:** extend `EditPublicationRequest` and existing local
  coordination with the sealed dependency bundle; prove its writer coverage.
- **Components/contracts:** `internal/projectapp/edit_publish.go`,
  `internal/local/{edit_adapters,edit_publication,publication,safe_fs}.go`,
  `operational_store.go`, `runtime_profile_store.go`,
  `runtime_profile_authoring.go`, `runtime_authorization_epoch.go`,
  `workflow_definitions.go`; inspect actual observation adapters used by T03.
  No universal transaction abstraction. Writer factoring is limited to shared
  coordination/lock-held reads; no semantic change to local authoring.
- **Done:** produce the dependency→reader→writer→lock/check→commit-boundary→test
  coverage table below with actual symbols. Acquire state-root → projects
  directory → installation before portable source/namespace locks, matching
  Runtime/operational/Workflow writers. Hold exclusion/guarded CAS through portable
  commit; avoid nested self-locking `Inspect` and creation of missing stores.
  Recheck exact bytes/content digests as well as counters, absence and object
  identity at protected boundaries. An unguardable source refuses publication.
- **V04 tests/Evidence:** independently drift every table row, including same
  counter/different bytes, absent→present, compatible Workflow reselection, source
  relocation, slug/UUID change, companion change, Profile retirement and observation
  change. Hook-controlled interleavings immediately before publication must refuse
  without rebasing old authority. Multiprocess tests prove one conflicting writer
  commits, bounded completion/no deadlock, broad-to-narrow ordering and no unsupported
  confinement/link-count/ancestor/leaf replacement. Name native primitives and
  unsupported source/filesystem refusals; do not claim arbitrary hostile same-UID
  or power-loss protection.
- **Non-goals/dependencies:** T01–T03; no public operation exposure before T05/T06,
  no new store, reconfiguration, observation refresh or lock-order exception.

Required coverage rows (to be resolved to concrete writers/tests during T04):

| Consulted dependency | Existing owner to trace | Required guard oracle |
| --- | --- | --- |
| Manifest/source/companions and complete generation | edit adapters, publication, safe filesystem | Exact bytes/artifact digest + root/ancestor/leaf identity; relocation/link/replacement/companion drift |
| Installation wire/revision and UUID/slug | Project installation/edit publication | Shared root/installation protection + local CAS; same revision/different wire |
| Operational admission, including absent state | operational store | Writer coordination + exact content/absence token; disabled/archive race |
| Workflow selection, index, selected exact definition or none | Project edit and workflow definitions | Selection in candidate source plus guarded catalog/definition digest; none→selected and reselection |
| Runtime/Profile configuration, including absence | runtime Profile store/authoring | Shared root coordination + token/content digest; retirement and absent→present |
| Consulted proof/observation and authorization freshness | actual T03 read-only sources, authorization epoch | Enumerate every real writer; guarded revision+digest through commit, otherwise refuse |

### G1-T05 — Exact replay, CAS publication and recovery (C)

- **Objective/scope:** publish only the reviewed full candidate using the existing
  portable-then-local commit protocol and truthful effect classification.
- **Components/contracts:** `internal/projectapp/{edit,edit_publish}.go`,
  `internal/local/{edit_publication,edit_recovery,publication}.go`;
  `ApplyEdit`, `EditAuthority`, `ValidateEditAuthority`, `EditPublisher`,
  existing `EditPublicationStatus`/`EditOutcome` and owned `.axiom-edit-*` recovery.
- **Done:** replay the same intent from fresh coherent observations, require exact
  UUID/digest and sealed expected revisions, recheck admission/compatibility/all
  T04 guards through commit. Publish policy manifest plus required installation
  bookkeeping only; retain local bindings, operational bytes and companion bytes.
  No-op validates bindings but writes nothing and creates no recovery state.
  Portable commit is the authoritative point; later bookkeeping failure is partial/
  recovery, never claimed rollback. Recovery recognizes only owned prior/new complete
  generations and must not overwrite a later writer. Recovery/rollback needs its
  separate existing authority; policy reversal is a fresh inverse intent/preview.
- **V05 tests/Evidence:** all missing/partial/wrong replay tuples (UUID, digest,
  local authorization, expected revisions), draft approval booleans, readiness
  success and precommit cancellation produce zero authoritative writes. Two writers
  cannot both use one preview. Inject private staging/recovery-record/prepublication/
  portable commit/local CAS/ack/sync/cleanup faults, permission/quota/short-write/
  interruption and cancellation after commit. Observe prior/new/recovery, exact
  confirmed effects and inspect guidance, never mixed accepted state or blind
  retry. Validate recovery metadata bounds/ownership/private permissions and no-op
  formatting preservation. Test absent local configuration without creating it.
- **Non-goals/dependencies:** T04 passed with complete writer coverage; no cross-
  filesystem extension, automatic recovery, dispatch, refresh or silent retry.

### G1-T06 — Observed read-back and completion (C)

- **Objective/scope:** reopen the canonical explicitly selected Project after
  apply and report observed policy/readiness and actual publication outcome.
- **Components/contracts:** existing edit source/readers in
  `internal/local/edit_adapters.go`, `internal/projectapp/{edit_publish,readiness,snapshot}.go`,
  `cmd/lingo/{main,readiness}.go`; reuse T03's read-only evaluator and existing
  completion vocabulary without inventing successful verification.
- **Done:** compare observed UUID/schema/full policy/portable+local revisions and
  artifact digests against committed candidate; evaluate the same `project validate`
  semantics. Distinguish no-op, refusal, confirmed+verified, committed-but-unverified,
  recovery and later-writer/read-back drift. Failure to read back cannot assert
  prior policy remains effective. Output uses T03 safe projection only.
- **V06 tests/Evidence:** successful v2/v3 read-back, no-op, reader failure after
  confirmed commit, local partial/recovery, same UUID with later revision, later
  writer before read-back, stale selector/source, changed local readiness snapshots.
  Verify observed facts vs original commit separately; no reapply or rollback.
  Compare a second synthetic Project and shared local registry/bindings before and
  after removal from A. Capture all four immutable-impact scopes without product
  Execution mutation. Dry-run/read-back agree for identical snapshots; differing
  snapshots explicitly explain why agreement cannot be claimed.
- **Non-goals/dependencies:** T05; no retry engine, source adoption, rebind,
  credential probing, canonical workflow transition or acceptance inference.

### G1-T07 — Existing configure EDIT input, guided and noninteractive parity (D)

- **Objective/scope:** expose T01–T06 through the existing CLI/Lingo route with
  strict draft transport, preview/confirmation and read-back.
- **Components/contracts:** `internal/cli/{cli,skill_inspect}.go`,
  `cmd/lingo/main.go`; `configureEdit`, `projectEdit`, `editSource.SelectForEdit`.
  Existing `--project-id`, `--preview-digest`, `--authorize-local` form the replay
  tuple. Use explicit installed UUID/slug and recorded source, never effective CWD.
- **Done:** select and record the actual draft option/help spelling and inspection
  metadata together, using T01's closed shape/bounds. This Task does not predeclare
  an unimplemented flag/API. Guided and JSON/noninteractive adapters feed the same
  application intent and safe result. Read/draft/dry-run cannot publish, refresh,
  create missing stores or collect authority. Human cancellation refuses before
  commit; stale confirmation demands a fresh preview. Inspection/help/command
  reference describe the actual implemented policy-only write set and readiness.
- **V07 tests/Evidence:** CLI black-box read→partial draft→dry-run→preview→exact
  authorization→apply→read-back/validate for v2/v3 and both input modes; invalid
  selectors (missing/ambiguous/stale/unadopted/unsafe/recovery), same slug in
  differing contexts and CWD mismatch refuse or select only the explicit identity.
  Exercise malformed/oversized/unknown draft, partial authority, cancellation,
  secret/path-safe errors and G-1 v1/v4/newer refusal. Existing CREATE, unrelated
  EDIT and unrelated schema-4 operations retain behavior. Compare real write set
  to preview, include installation bookkeeping only when required. No generic
  legacy preview leaking full manifest/host destinations.
- **Non-goals/dependencies:** T01–T06; no new command family, output architecture
  redesign or AXM-32 proposal adoption; no operator-authored raw YAML workaround.

### G1-T08 — Thin axiom-project conversation and inspection (D)

- **Objective/scope:** expose the same G-1 journey conversationally without
  giving the host authority or requiring hand-authored policy JSON.
- **Components/contracts:** embedded
  `internal/codexruntime/skills/axiom-project/SKILL.md`,
  `internal/cli/skill_inspect.go`, existing installed-skill inventory/parity tests
  and receipts/history rules. Inspect existing shared distribution to preserve
  Codex/Claude maintainer-host parity, without creating runtime-specific policy.
- **Done:** route existing Project policy read/edit to the canonical operation;
  collect explicit member intent, relay application validation/impact, obtain
  exact local confirmation and consume read-back. A thin skill cannot patch files,
  choose a Profile/model/Runtime, synthesize approval, hide blockers or retry an
  uncertain mutation. Route local Profile needs to G-2 and Workflow authoring to
  its owner with separate authority, without invoking those actions automatically.
- **V08 tests/Evidence:** synthetic routing/installed-skill/inspect tests compare
  canonical payload, UUID/digest/authority/write set and sanitized result across
  hosts. Cover ambiguous identity, unavailable local Profile, Workflow incompatibility,
  incomplete consent, drift, cancellation, malicious draft instructions and read-back
  failure; all retain application refusal. Separately collect bounded native
  conversation evidence only with required live-test authority, named host/version
  and exact application receipts. Parser/fixture success is not native R-2/R-3 or
  product acceptance; unavailable native sessions remain NOT RUN/BLOCKED.
- **Non-goals/dependencies:** T07; no new Workflow selection route, host-owned
  resolver, G-2 authoring, live dispatch or change to Agent Factory renderer claims.

### G1-T09 — Integrated regression, security and Evidence closure (E)

- **Objective/scope:** close the matrix below using T01–T08 evidence; reconcile
  only actual G-1 command/readiness/recovery documentation and implementation
  Evidence, without starting new features or changing accepted requirements.
- **Components/contracts:** existing Project/projectapp/manifest/local/runtime
  application/CLI/Lingo/skill and compatibility suites;
  `internal/compatibility/testdata/stable-v1` remains append-only/unchanged history.
  Read immutable consumer fixtures in workflowcompiler/graphapplication without
  adding dispatch behavior. Follow the persisted-state quality guards if any
  actual encoding changes arise; a new format requires separately reviewed contract.
- **Done:** every G1-AC has named passing tests or explicit unresolved blockers,
  no skipped requirement relabeled PASS. Compare full unrelated bytes/digests for
  A/B Projects, shared local registry, companions, operational state, local bindings,
  immutable Workflow/Execution/Stage Plan/effort/proof snapshots and Work Items.
  Freeze no fabricated corpus; preserve v1 reads and existing CREATE/unrelated
  EDIT/schema-4 semantics. Rerun proportional integrated/security checks once
  dependencies pass; findings return to their owning Task, not a new work front.
- **V09 tests/Evidence:** scoped unit/integration/CLI suites then `go test ./...`,
  relevant `go test -race` suites, `go vet ./...`, repository validation, diff and
  sensitive-file checks. Use deterministic hooks/multiprocess tests for T04/T05
  faults, complete output/log/file/Evidence sentinel scans, traversal/links/controlled
  replacements, exact-authority refusal and A/B isolation. Record native OS/arch/
  filesystem/primitives for Linux/macOS per the
  [existing persistence support matrix](../004-mvp-v1-baseline/plan.md#10-persistence-filesystem-concurrency-and-recovery);
  generic CI/synthetic tests do not prove each native combination. Unavailable
  environments remain unverified. No paid inference/authenticated Provider test
  is implicitly authorized by this task list.
- **Non-goals/dependencies:** final closure waits T01–T08; no release, merge,
  AXM-12 E2E, acceptance or automatic Issue/status change. Human review decides
  whether the actual evidence satisfies the approved contract.

## Acceptance coverage matrix

All entries are **future validation — NOT RUN**. Task ownership is additive;
V01–V09 refer to the exact oracles above, not an assertion of existing tests.

| Criterion | Tasks | Required validations / observable Evidence |
| --- | --- | --- |
| G1-AC01 | T01, T03, T05, T06, T07, T08 | V01/V03/V05–V08: full v2/v3 read/edit/preview/exact authorization/apply/read-back journey; same UUID/schema |
| G1-AC02 | T01, T06 | V01/V06: omission/four forms, each partial member edit, preserved unrelated state/companions; semantic no-op retains original bytes/revisions |
| G1-AC03 | T01, T07 | V01/V07: valid 8/32/32 and negative 9/33/33, closed bounded draft, duplicates/shapes/unsafe/dangling references; zero effects |
| G1-AC04 | T01, T02 | V01/V02: retained dependent removals and conflicting/nonexistent targets refuse; explicit compatible reconciliation succeeds |
| G1-AC05 | T05, T06, T09 | V05/V06/V09: remove Runtime/dependents from A only; read back A+B and compare full B/shared local bytes/digests |
| G1-AC06 | T03, T05, T07 | V03/V05/V07: valid compatible policy saved under exact authority despite local absence/retirement/disable/mismatch/proof/ambiguity; precise execution blockers, no fallback/delete/dispatch |
| G1-AC07 | T02, T03, T04 | V02–V04: exact Workflow/Stage/Agent/Profile/Runtime, constraints/default precedence, none vs unresolvable/tampered, retained/new incompatibility, full bounded report and dependency drift |
| G1-AC08 | T02, T06, T09 | V02/V06/V09: future admission semantics vs frozen bindings/effort/proofs, current permission/freshness preserved, no Work Item/Execution mutation or fabricated inventory |
| G1-AC09 | T03, T05, T07, T08 | V03/V05/V07/V08: exact full replay tuple, missing/wrong revision/UUID/digest/authority, cancellation/host or draft approval cannot authorize; zero authoritative writes |
| G1-AC10 | T03, T04, T05 | V03–V05: every consulted snapshot/absence/identity/content token, same counter/new bytes, relocation/reselection, all commit boundaries; one competing writer wins |
| G1-AC11 | T05, T06, T09 | V05/V06/V09: staging→portable→local→ack/cleanup faults, permission/quota/short-write/interruption, recovery ownership, failed/later-writer read-back; truthful old/new/partial/unverified, no blind retry |
| G1-AC12 | T01, T07, T09 | V01/V07/V09: v1 read corpus preserved; G-1 v1/v4/newer refusal; unchanged v2/v3, CREATE/unrelated EDIT/schema-4; no migration or historical fixture rewrite |
| G1-AC13 | T03, T06, T07 | V03/V06/V07: proposed dry-run and persisted `project validate` agree on equal snapshots; structural/operation readiness separate; side-effect/secret-reader spies |
| G1-AC14 | T01, T03, T04, T05, T06, T07, T08, T09 | V01/V03–V09: adversarial secret/path/error/instruction inputs, bounded safe output/log/files/Evidence, confinement/links/replacements/native guards, preserved local bindings |
| G1-AC15 | T07, T08, T09 | V07–V09: canonical/conversational payload and authority/read-back parity, inspection/distribution, negative bypass refusal; separate native evidence, no hidden G-2/Workflow effects |

## Future Evidence and approval handoff

Each future Task receipt must name: Task/AC/V IDs; source and tested commit SHA;
command and exit/result actually observed; test/fixture references; synthetic
Project UUIDs/schema; before/proposed/observed revisions and artifact/content
digests; exact preview/authority outcome; expected and actual write set; sanitized
diagnostics/read-back; platform/architecture/filesystem where relevant; unsupported
or unexecuted cases. Link bounded receipts from the owning Task/PR and final
G-1 matrix; do not create empty Evidence files or copy raw chat/logs/secrets.
Use explicit PASS/FAIL/NOT_RUN/BLOCKED/justified NOT_APPLICABLE per check.
Portable validity, local readiness, synthetic validation, native conversation,
human approval and MVP acceptance must remain separately classified.

Documentary validation executed in this Tasks phase:

- `./scripts/validate-repository.sh .` — PASS, including harness/adapters,
  automation registry, issue-label, ADR governance and sensitive-file checks.
  Optional runtime behavioral scenarios were skipped and remain UNVERIFIED.
- Ephemeral Python assertions — PASS: nine unique ordered Tasks with required
  fields, 15 ordered AC rows with Task/validation references, 122 existing local
  Markdown targets/anchors across Tasks/index, balanced fences and byte-identical
  approved amendment/Plan. The one-off checker is not versioned automation.
- `git diff --check` — PASS; final staged checks and commit/push read-back are
  recorded in PR #327. `gitleaks` unavailable; not installed.
- Recheck of approved base SHA `c2118649803a086cab67588a6d7547ac721e7a71`:
  `PR Checks` and delivery-metadata completed SUCCESS after initial discovery.
  That result does not cover the subsequent Tasks commit; consult its own CI.

These checks establish document integrity only. No V01–V09 or product/Go tests
were run in this Tasks phase. Scope is this artifact and the necessary Specifications
index reconciliation. No accepted Specification, Plan, ADR, production/test code,
Runtime, local Profile, Workflow, credential or Execution artifact changes.

Constitution/security review: intent and accepted Plan precede executable work;
minimal units reuse real owners, deterministic checks and exact local authority;
portable permission stays distinct from machine readiness; immutable facts and
public-data hygiene remain intact. The Orchestrator performed discovery,
decomposition and a separate matrix/authority/security cross-check in the main
session. This is not independent-agent review or human acceptance.

Next action: human technical review and explicit approval of the exact Tasks
revision before implementation. On continuation, revalidate local/remote HEAD,
Plan approval/material drift, this artifact, only-open-PR condition, CI and actual
canonical Execution/gates/revision/competing work. Resolve canonical truth before
any workflow transition; preserve recovery-required conditions rather than invent
progress. All Tasks remain Not started. The guard coverage table, native support
Evidence and real conversation remain future passage criteria, not delivered
capabilities. No new architectural decision is silently accepted.
