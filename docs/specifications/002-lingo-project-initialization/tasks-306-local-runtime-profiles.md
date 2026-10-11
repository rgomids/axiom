# Tasks — Issue #306: Local Runtime/Profile authoring and onboarding

## Status and exact authority

**Proposed — Awaiting Human Approval — Tasks, 2026-10-11.** Every Task is **Not started**. [The maintainer accepted six planning artifacts](https://github.com/rgomids/axiom/pull/310#issuecomment-6109183466) at `823bce3141bae5e90951c5239181c46d88cfbfe9`; [acceptance/Tasks Evidence](evidence-306-acceptance-tasks.md) records their exact blobs and this documentary reconciliation. That acceptance and the current request authorize Tasks preparation, not their approval or implementation. Same branch and Draft PR #310; no production/schema/fixture/configuration/provider changes, inference, merge, release or Issue state changes.

Canonical scope is issue-specific `tasks-306-local-runtime-profiles.md`, consistent with existing `I132-Txx` issue-scoped units in Spec 004 and the #303 artifact. It does not replace original Spec 002 `tasks.md`. IDs remain stable across revisions; X IDs are the separately gated cross-cutting bundle, not a new Issue or implicit #306 ownership.

## Sources and reference legend

- **LA-xx**: [accepted Spec 002 local-authoring amendment](amendment-306-local-runtime-authoring.md).
- **PE-xx / ON-xx / AC-PE / AC-ON / AC-SEC / §3**: [accepted Spec 007 effort/onboarding amendment](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md).
- **Plan**: [accepted Plan #306](plan-306-local-runtime-profiles.md); **Security**: [accepted design baseline](security-review-306-local-runtime-profiles.md).
- **ADR-0023/0024**: [local configuration surface](../../decisions/0023-local-runtime-configuration-surface.md) / [sole effort owner](../../decisions/0024-model-profile-effort-ownership.md); exact file paths distinguish ADR identities. [ADR-0018](../../decisions/0018-adr-evolution-and-supersession-governance.md) governs historical annotations.
- Other directly affected foundations: [Spec 002](spec.md), [portable Runtime policy v2](runtime-policy-v2.md), [Spec 007](../007-configurable-workflows/spec.md), [#303 workflow skill contract](../007-configurable-workflows/amendment-303-workflow-skill.md), [#231 readiness](evidence-issue-231.md), [ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md), [ADR-0007](../../decisions/0007-local-publication-and-recovery-protocol.md), [ADR-0020](../../decisions/0020-workflow-definition-revision-binding.md).
- Task structure follows repository `axiom-sdd` and issue-scoped conventions. The requested “00 - Axiom Core”, “01 - Protocolo Operacional”, “02 - Playbook SDD” were not located by bounded repository/connected-source lookup; precise sources have been requested. Their verification is pending and must be completed before declaring Playbook conformance; no unseen requirement is invented.

## Ownership, external dependencies and human gates

| Boundary | Owner / precise dependency | This artifact's action |
| --- | --- | --- |
| Local domain/store/authoring/CLI/skill/onboarding | #306 / G-2 | T01–T16: confirmed local scope, Tasks not approved |
| Workflow schema/migration/compiler/Execution/graph/default | **PD-OWN — Pending Human Decision** | X01–X07: complete bounded dependency decomposition; recommendation is a separately authorized technical Work Item/bundle; no new Issue created and no implementation assignment inferred |
| Portable Project policy authoring | #305 / G-1 | Consume existing policy/readiness; needed policy edits handed off with separate confirmation; never implement #305 in T/X units |
| Positive Runtime/model/effort proof | G-3 | Consume exact evidence contract; deterministic injected inputs test refusal/consumption only. Real acquisition needs authority; proposed #276 PR #311 includes this responsibility but is not accepted by this artifact |
| Runtime dispatch / attempts / coordination | #276 | X05 owns the accepted D3-compatible snapshot/guard seam only if PD-OWN assigns it; public dispatch composition and positive proof delivery remain #276; no second scheduler |
| Delivery/rework/new continuation lifecycle | #277 | Consume existing lifecycle only; no old Execution rewrite or new automatic successor mechanism |
| Actual R-2/R-3/end-to-end product acceptance | #278 | Separate native/vendor authority and acceptance; cannot be satisfied by local/synthetic CI |
| Consented Provider linkage | G-4 | Independent; neither setup nor preview grants Provider authority |

**PD-OWN recommendation:** keep #306 local scope, approve a separate bounded migration bundle X01–X07 with explicit integration owner and dependency relation before dependent implementation. Alternative: maintainer expressly assigns that same bundle to #306. Either choice needs a recorded human decision; this artifact does not choose it. Existing accepted format/window/no-override choices are not reopened. X tasks are blocked by PD-OWN; T16's complete preparation integration additionally depends on X07. T01–T15 can establish local behavior without fabricating Workflow readiness. All T/X work still needs exact Tasks approval and explicit bounded implementation authority.

**External readiness gates:** G-3 exact model/executable/version/mode/token/no-override proof; #305 policy edits where required; #276 supported dispatch seam; #277 continuation if needed; #278 native acceptance. Unsupported adapter effort-source fields/precedence must block; their authoritative proof is a prerequisite for declaring support, not permission to invent vendor behavior. No authenticated test is run in this phase.

**Concurrent PR check:** #311 head `4c857ead8f4c7ec8d027df2d8eef746464b55d17` is a Proposed #276 planning change. It references accepted #306 effort ownership but also introduces a different ADR file numbered 0024 and touches the same indexes. ADR identity is path-based; preserve the accepted #306 path and reconcile indexes on eventual integration. No renumbering, edits to #311 or acceptance of its proposals here.

## Phases and scheduling

| Phase | Units | Count | Completion output |
| --- | --- | --- | --- |
| 1 Interfaces/domain/resolution | T01–T03 | 3 | Local versioned contracts and sole resolver |
| 2 Persistence/recovery/migration | T04–T06 | 3 | Protected monotonic lifecycle and explicit migration |
| 3 Safety/authoring/CLI/readiness | T07–T10 | 4 | Exact confirmed local authoring and diagnostic seams |
| 4 Transversal migration (PD-OWN) | X01–X07 | 7 | Compatible Workflow/compiler/Execution/graph/default chain |
| 5 Skill/passive onboarding/templates | T11–T13 | 3 | Both Runtime integrations and guided setup |
| 6 Integration/native local verification | T14–T15 | 2 | Classified local/synthetic/native receipts |
| 7 Evidence and dependency handoff | T16 | 1 | G-2 traceability and honest remaining blockers |
| Total | 16 local + 7 transversal | **23** | No unit started or accepted |

Phase order is a review grouping, not an extra dependency: follow each Task's exact DAG. Recommended spine: T01→T02→T04→T05→T06, with T07 safety also required before T06; T03/T07 feed T08→T09→T11→T12→T13; T10 integrates readiness; T14/T15 produce local Evidence. After PD-OWN, X01 can run alongside local foundations, X02 after local migration, X03 after resolution, X04→X05 after frozen contracts; X06 joins onboarding, X07 then T16. Do not hold all local work for externally acquired positive proof: save intent truthfully as unproven, keep use blocked.

Parallelizable after their prerequisites: T03 and T04 and T07 after T02; T09 and T10 after T08; X01 alongside T03/T04 once interfaces stable and PD-OWN resolved; X02 and X03 after their distinct inputs; T14 and T15 after onboarding/native prerequisites. Shared files require disjoint ownership or serial integration; parallel arrows are permission to schedule, never permission for concurrent writes to the same file. No live/native vendor acceptance is delegated automatically.

## Shared completion and Evidence contract

Each unit below has its own objective, scope, components, dependencies, acceptance, negatives and Evidence. **Done condition for every unit:** all listed acceptance assertions and relevant refusal cases verified at the final implementation HEAD; expected Evidence committed/classified; applicable independent security/native/compatibility review recorded; no unrelated effects; human review/acceptance remains separate. An unavailable required check leaves that unit pending, never silently Done. No Task completion checkbox is checked in this artifact.

Every implementation unit includes its proportional tests; T14/X07 integrate cross-boundary journeys rather than repeat unit suites. Evidence names exact commit/blob, command, platform/mode, expected and observed result, safe digests, PASS/FAIL/BLOCKED/SKIPPED/UNAVAILABLE/NOT_APPLICABLE and synthetic/local/native classification. Synthetic skill/observations are not real conversation/model/auth support. Native filesystem/integration is not native vendor inference. Boundaries preserve Project != Repository; Runtime != Model; Role != Model; Execution != Agent; Provider != Capability; portable intent != local state; Skill/Evidence never grants authority.

## Phase 1 Tasks

### I306-T01 — Freeze canonical authoring and compatibility interfaces

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Specify typed snapshot/state/conditional-store and result ports in the existing Runtime application. Keep stored validation/resolution preview distinct from mutation preview. Define format dispatch and safe diagnostic categories used by subsequent units; this is implementation of accepted interfaces, not a new registry.
- **Contract references:** LA-01/04/10; PE-04; Plan §3–4; ADR-0007/0024.
- **Likely components/files:** `internal/runtimeapplication; internal/runtimeprofile; internal/local/runtime_profile_store.go; internal/cli; cmd/lingo`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** None within the Task DAG; common approval/implementation gates apply.
- **Acceptance:** Every write port requires exact prior state/candidate/effects and local authority; read/prepare/preview ports promise zero filesystem/process/network writes; fresh/active-v1/active-v2/retired/unsafe/pending are distinguishable; existing CLI contracts have unchanged tests.
- **Negative scenarios:** Numeric revision alone, unrecognized state, mutation envelope reused for selection, raw config/path in public DTO.
- **Expected Evidence:** Consumer/port contract tests, negative API tests and compatibility signatures reviewed with Security.
- **Flags / gates:** security-review, persisted-contract.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T02 — Add local v2 effort domain and closed codecs

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Extend the existing domain with required explicit-token or runtime-default effort; retain v1 inspection semantics and introduce tombstone wire recognition. Preserve canonical store location and independent generation. Keep Profile eligibility as Runtime Enabled + allowlist membership.
- **Contract references:** LA-02/03/06/07/10; PE-01/02; AC-PE1/2.
- **Likely components/files:** `internal/runtimeprofile/runtimeprofile.go; codec/domain tests; internal/local compatibility inventory`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T01.
- **Acceptance:** Eight Runtime/32 Profile limits and >3 Profiles per Runtime hold; same Runtime/model with distinct effort IDs has distinct digest; complexity remains independent; active v2 rejects credentials and alternate effort-capability owners; strict tombstone format1 has only kind/version/revision/prior digest; deterministic encode/digest works across ordering.
- **Negative scenarios:** Missing/null effort; duplicate IDs/JSON keys; unknown/mixed version fields; zero/fractional/overflow revision; invalid digest; dangling/empty allowlists; credentialReference nested or top-level; unsafe tokens.
- **Expected Evidence:** Table-driven codec/domain/canonical digest tests plus unchanged historical v1 fixtures and append-only new v2/tombstone corpus.
- **Flags / gates:** security-review, persisted-contract, migration.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T03 — Resolve the exact Profile before effort proof

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Extend the sole resolver and Project projection for local2. Resolve explicit ref/exact Project preference/unique permitted candidate first, derive effort only then. Introduce the safe versioned resolvedProfile contract consumed by compiler; do not edit portable policy or acquire proof.
- **Contract references:** LA-07/09; PE-03/04/06; AC-PE1/3; ADR-0024.
- **Likely components/files:** `internal/runtimeprofile resolver; internal/runtimeapplication/policy.go; internal/projectapp readiness`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T02.
- **Acceptance:** Exact Project Runtime/Profile/model intersection maintained; tuple includes all PE-04 identities/digests/mode/proof refs; effort failure blocks chosen Profile; two otherwise eligible candidates remain ambiguous even if only one effort works; saved unproven intent is structurally valid but not operationally ready.
- **Negative scenarios:** Runtime-wide/declared/auth-only proof; stale exact-model evidence; initiating Runtime preference; effort used as tiebreak; fallback; local preference replacing Project choice; v1 fabricated default.
- **Expected Evidence:** Injected exact/mismatched/unknown observations and Project intersection regressions; safe snapshot equality and drift tests. No native proof acquisition.
- **Flags / gates:** security-review, persisted-contract, external:G-3.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

## Phase 2 Tasks

### I306-T04 — Publish conditional lifecycle with canonical tombstones

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Implement conditional create/update/retire/recreate on the existing protected store with ancestor/file/directory identity fences, ordered locks, atomic publication and read-back. Valid initial compound config uses revision1; tombstone preserves final-removal history.
- **Contract references:** LA-03/04/10; AC-SEC; Plan §4.
- **Likely components/files:** `internal/local/runtime_profile_store.go; safe_fs.go; publication.go; local tests`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T01, I306-T02.
- **Acceptance:** Create valid nonempty config; each committed change increments once, no-op unchanged; last removal publishes tombstone atomically and recreation uses prior+1; exact identity and reviewed bytes checked inside lock and after read-back; concurrent winner has complete state and loser refuses.
- **Negative scenarios:** CREATE collision; empty existing version directory; same revision changed bytes; same-byte replacement; ancestor/root replacement; ABA; stale tombstone/replay; overflow; cancellation around commit; arbitrary externally erased history cannot be claimed detectable.
- **Expected Evidence:** Fault-point and multi-process tests with retained prior/new bytes, exact revisions and path identities; race checks where supported.
- **Flags / gates:** security-review, persisted-contract, migration, native-filesystem.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T05 — Extend recovery and recognition for all local generations

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Register canonical runtime-profiles/v1 active1/active2/tombstone1 with recovery discovery/inventory/every-writer recognition. Reuse marker protocol, scoped preview and exact recovery authority; no automatic repair.
- **Contract references:** LA-10; Spec 007 §3; AC-SEC; FR-026; ADR-0007.
- **Likely components/files:** `internal/local recoveryDirectories/publication/inventory; compatibility corpus; recovery CLI seam`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T04.
- **Acceptance:** Crash/restart yields complete prior/new state or actionable pending/unknown state; exact uncommitted restore or provable finalize only; committed rollback is new authorized generation; every reader/writer/recovery recognizes family; unsafe and unknown objects preserved.
- **Negative scenarios:** Lost fsync/read-back; malformed/conflicting marker; interrupted retirement/recreation; external root substitution; unrecognized temp object; rollback resetting revision; blind retry/deletion; unsupported downgrade.
- **Expected Evidence:** Restart/fault matrices, inventory correspondence, refusal-before-effects spies and supported native filesystem receipts.
- **Flags / gates:** security-review, persisted-contract, migration, native-filesystem.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T06 — Migrate local v1 with explicit per-Profile effort

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Build reviewed v1→v2 candidate on canonical application/store ports. Each effort chosen by operator; any legacy reference removal is a visible approved local effect. Migration does not touch vendor login state or infer from Workflow.
- **Contract references:** LA-06/07/10; PE-01; Spec 007 §3; AC-PE2.
- **Likely components/files:** `internal/runtimeapplication migration; runtimeprofile codecs; local store; CLI migration DTO`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T02, I306-T04, I306-T05, I306-T07.
- **Acceptance:** Source digest/revision/identities fixed; unchanged historical v1 remains inspectable; every Profile receives explicit selected effort; legacy credentialReference explicitly cleared or migration refuses; confirmed migration increments once and restart read-back matches; unrelated entries preserved.
- **Negative scenarios:** Missing mapping; effort inferred from legacy capability/Agent/name; stale source; unsafe legacy value echoed; implicit credential clearing; auto-migration on read; down-conversion; partial multi-Profile conversion.
- **Expected Evidence:** Golden before/candidate/after and authority/stale/fault tests, canary-safe reports and original v1 byte preservation.
- **Flags / gates:** security-review, persisted-contract, migration.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

## Phase 3 Tasks

### I306-T07 — Enforce safe local authoring and vendor-auth boundaries

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Apply pure bounded safety/normalization before draft/output and all canonical authoring routes. Produce safe list/show/validation/error projections for legacy unsafe fields. Reuse subscription guards; create no credential backend.
- **Contract references:** LA-05/06/08; Security threat table; AC-SEC; D2.
- **Likely components/files:** `internal/runtimeapplication authoring; internal/portableconfig safety reuse; runtimeadapter auth boundary; CLI renderers`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T02.
- **Acceptance:** Reject secret/path/command/env/provider payloads without echo; no login/session/credential read or resolver during authoring; auth/model/effort facts distinct; ordinary reads/prepare/preview/apply cause zero vendor/network effects; unchanged historical unsafe fields only safe inspection.
- **Negative scenarios:** URL/assignment/Unicode/escaped multiline canaries; unsafe model/effort/reference; raw adapter errors; unknown nested keys; credential helper; malformed input causing diagnostic leakage.
- **Expected Evidence:** Canary assertions across stdout/stderr/log/completion/preview/Evidence, zero process/network spies and boundary regression tests.
- **Flags / gates:** security-review.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T08 — Implement exact preview/apply authoring use cases

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Implement complete typed candidates for read/list/show/validate/create/edit/enable-disable/remove and scoped recovery. Compute exact effects, snapshot token and preview digest; apply only reviewed candidate via conditional store. Disclose bounded impact inventory.
- **Contract references:** LA-01/03/04/09; Plan §3–4; D1/D5.
- **Likely components/files:** `internal/runtimeapplication authoring use cases; store port; canonical result/diagnostic DTO`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T03, I306-T04, I306-T05, I306-T06, I306-T07.
- **Acceptance:** Preview/validate zero filesystem mutations, including lock-file creation, and zero processes/network; nonmutating shared directory locks remain permitted; envelope binds existence, format, generation, content and object identities, candidate and effects; explicit confirmation plus snapshot/digest/local authority required; valid compound removal or precise constraint; read-back reports confirmed/uncertain truth; replay is stale.
- **Negative scenarios:** Wrong target/digest/authority; modified candidate/effects; same revision content drift; partial removal creates dangling allowlist/preference; unknown/pending mistaken for absence; undisclosed impacts; successful replay treated idempotently.
- **Expected Evidence:** Application mutation/effect matrix, snapshot/digest goldens, conflict/replay/partial-fault tests and safe bounded dependency report.
- **Flags / gates:** security-review, persisted-contract.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T09 — Expose canonical CLI operations and metadata

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Wire thin CLI transport to application ports. Add discoverable candidate/mutation/recovery operations and canonical rendering/help/skill inspection metadata; preserve current stored validate no-argument and resolution preview contracts. Names follow accepted Plan surface, with no direct Save escape hatch.
- **Contract references:** LA-01/11; Plan §3; ADR-0023.
- **Likely components/files:** `cmd/lingo runtime command routes; internal/cli command/result metadata and completion`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T08.
- **Acceptance:** Each route delegates same application authority; missing authorization returns preview only; explicit args carry snapshot/digest; machine-readable result and natural rendering agree; existing flags/results remain compatible; errors/next actions deterministic and redacted.
- **Negative scenarios:** CLI bypassing application; mutation via read alias; old resolution digest authorizes edit; malformed/duplicate flags; raw JSON/token error echo; commands unavailable but advertised.
- **Expected Evidence:** Black-box command/help/inspect/completion fixtures, exit/status/result contract tests, unchanged old command tests.
- **Flags / gates:** security-review.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T10 — Integrate bounded impacts with existing readiness

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Reuse #231 readiness and Project inventory to report local dependency/removal effects and configuration/authentication/operational categories. Verify #305 portable policy intersection read-only; diagnostics point to rightful owner.
- **Contract references:** LA-09; ON-03/04; AC-ON4; Plan §8; #231.
- **Likely components/files:** `internal/projectapp readiness; runtimeapplication availability/projections; Project catalog interfaces`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T03, I306-T08.
- **Acceptance:** Two Projects sharing a Profile remain byte-identical after local removal; future resolution blocks both; known dependencies plus incomplete/inaccessible inventory limits disclosed; no arbitrary repo scan; missing/mismatch/disabled/ambiguous/auth/model/effort/capability distinct; repaired local config restores future candidates only.
- **Negative scenarios:** Claim complete inventory with missing Projects; auto-patch policy/Workflow/Execution; unavailable #305 implies local edit permission; auth failure prevents structural intent save; new aggregate Doctor; Provider effects.
- **Expected Evidence:** Bounded inventory fixtures and two-Project byte/digest before-after test; structured readiness diagnostics against actual policy codecs.
- **Flags / gates:** security-review, external:#305.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

## Phase 4 Tasks

### I306-X01 — Introduce Workflow schema 2 with historical codecs

- **Status:** Not started; blocked by PD-OWN and common authorization gates.
- **Objective / implementation scope:** Add closed schema2/model/validator family without Agent effort or alternate reserved effort requirements. Retain schema1 parse/encode/digest meanings and original schema/fixtures addressably.
- **Contract references:** PE-02; Spec 007 §3; AC-PE1/2; ADR-0024.
- **Likely components/files:** `internal/workflowdefinition/definition.go, validation.go, schema/catalog; documentary versioned schema/examples`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T01, I306-T02; PD-OWN.
- **Acceptance:** Strict version dispatch; schema2 role/complexity/Profile refs unchanged; explicit/null/empty Agent effort and reserved effort selectors rejected; old fixtures produce identical bytes/digests; mixed schemas fail.
- **Negative scenarios:** Replacing v1 schema under same identity; dropping historical effort; unknown/duplicate fields; capability/WorkUnit/CLI alternate effort owner.
- **Expected Evidence:** Schema/codec compatibility corpus and negative-validation tests linked to AC-PE1/2.
- **Flags / gates:** security-review, persisted-contract, migration, ownership:PD-OWN.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-X02 — Publish explicitly mapped Workflow migrations

- **Status:** Not started; blocked by PD-OWN and common authorization gates.
- **Objective / implementation scope:** Use existing Project-owned authorship to prepare schema2 from exact schema1 revision with operator per-Agent mapping. Compare historical requested effort with selected Profile effort; publish a new immutable revision of same ID, under separate authority.
- **Contract references:** Spec 007 §3; PE-02/03; AC-PE2; ADR-0020/0024.
- **Likely components/files:** `internal/projectapp Workflow authoring; workflowdefinition; existing CLI/axiom-workflow configuration route`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-X01, I306-T06, I306-T08; PD-OWN.
- **Acceptance:** Preview shows source ref/digest and each changed request/Profile/effort; source bytes preserved; exact confirmation publishes next revision/read-back; no active selection/start/local config/policy mutation; incompatible replacement visibly changed behavior; disallowed Profile returns project_policy_update_required.
- **Negative scenarios:** Automatic Profile derivation; partial mapping; changed source/local/policy after preview; old digest approves new revision; silent Project preference write; binding existing Execution to new revision.
- **Expected Evidence:** Migration comparison golden, immutable source digest and separate authority/selection spies; recovery/restart cases.
- **Flags / gates:** security-review, persisted-contract, migration, ownership:PD-OWN, external:#305.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-X03 — Compile Profile-owned controls and Stage Plan v2

- **Status:** Not started; blocked by PD-OWN and common authorization gates.
- **Objective / implementation scope:** Consume sole-resolver resolvedProfile and produce compiler/result/Stage Plan2 with profileEffort, source provenance and safe exact tuple. Frozen child controls derive only from tuple; preserve v1 RequestedEffort DTO.
- **Contract references:** PE-03/04/05; AC-PE1/3; ADR-0024.
- **Likely components/files:** `internal/workflowcompiler/compiler.go, stage_plan.go; workflow stage planning; CLI result version dispatch`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-X01, I306-T03; PD-OWN.
- **Acceptance:** Profile selected before proof; controls/tokens match exact effort; single and multi-agent paths carry profile/config/exe/version/mode/obs/proof/source digests; old approvals invalid for v2; unknown facts block rather than fabricate dispatchable placeholders.
- **Negative scenarios:** Effort as tiebreak; substitute Profile on unsupported effort; WorkUnit/CLI override; mismatched tuple; mutable config re-read to invent old requestedEffort; v1 digest replay.
- **Expected Evidence:** Compiler/Stage Plan deterministic goldens, domain-to-compiler negative boundary tests and single/mixed graph snapshots.
- **Flags / gates:** security-review, persisted-contract, ownership:PD-OWN, external:G-3.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-X04 — Freeze configured Execution format 3 bindings

- **Status:** Not started; blocked by PD-OWN and common authorization gates.
- **Objective / implementation scope:** Extend canonical Execution codec/admission/inventory for format3 configured-sequential-v2/schema2. Start freezes Workflow and initial Profile; later authorized stage admission freezes actual child tuples with protected publication. Keep legacy inspection/completion.
- **Contract references:** PE-04/05; Spec 007 §3; AC-PE2/3; ADR-0020/0024.
- **Likely components/files:** `internal/workflow/configured.go, service.go; internal/local/workflow_store.go; compatibility inventory`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-X01, I306-X03; PD-OWN.
- **Acceptance:** New records bind exact immutable Workflow/Profile tuple; later lifecycle appends cannot rewrite prior bindings; format1/2 byte meaning retained; legacy new dispatch/retry lacking tuple returns legacy_execution_contract; Project selection/local edit never rebinds existing Execution; old Workflow starts require workflow_migration_required.
- **Negative scenarios:** Recompile old Execution after migration; synthesize legacy snapshot from current config; implicit successor/continuation identity; partial publication loses tuple; stale admission approval.
- **Expected Evidence:** Versioned codec/admission/fault/restart tests and historical binding digests before-after.
- **Flags / gates:** security-review, persisted-contract, migration, ownership:PD-OWN.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-X05 — Bind graph v2 and effort-source attempt checks

- **Status:** Not started; blocked by PD-OWN and common authorization gates.
- **Objective / implementation scope:** Extend existing proposal/graph2 per-node snapshots and same single/graph attempt checks. Implement bounded, separately consented effort-source inspection with adapter exact-version/mode field/precedence provenance; reuse auth/argument/env guards. Unknown semantics refuse. Dispatch composition and positive proof acquisition remain with their owners.
- **Contract references:** PE-05/05a/06; AC-PE3; Security sole-owner review.
- **Likely components/files:** `internal/executiongraph codecs/planner; graphapplication/runtime_policy.go; runtimeadapter/adapters/authpreflight; existing scheduler guard seam`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-X03, I306-X04, I306-T07; PD-OWN.
- **Acceptance:** Tuple/controls/proof/source digest verified before effects each attempt; only Profile-derived explicit override accepted; runtime-default has no external selector; selector-free native default permitted without asserted concrete level; safe source classes/key-presence only; sources rebound/rechecked; original graph1 unchanged.
- **Negative scenarios:** External Codex config/Claude effort env even same token; unknown version/precedence/selected section; unreadable/malformed/oversized/drop-in/reparse/symlink source; duplicate normalized environment keys/multiple Claude effort selectors; selector insertion after preview or changed effortSourceDigest despite unchanged model/configuration revision; changed exe/model/mode/proof/config; hashing unknown source; fixture/auth-only/Runtime-wide proof accepted; settings repair.
- **Expected Evidence:** Injected adapter/preflight-to-policy-to-scheduler tests with zero-effect assertions; authoritative source references required per supported mapping. Native proof/support remains external G-3/#276.
- **Flags / gates:** security-review, persisted-contract, migration, ownership:PD-OWN, external:G-3/#276, native-runtime-gate.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-X06 — Distribute default-sdd r2 and confirmed derived setup

- **Status:** Not started; blocked by PD-OWN and common authorization gates.
- **Objective / implementation scope:** Add r2/schema2 template with generic supported Runtime constraints/policy-default and no concrete model/effort. Keep builtin r1 identity/digest addressable. Guide separately confirmed Project-owned copy, references or portable preference through existing owners.
- **Contract references:** ON-04/ON-05; AC-ON3; PE-02/03; ADR-0023/0024.
- **Likely components/files:** `workflowdefinition builtin catalog/new default-sdd-r2; examples; axiom-workflow guidance; Project selection routes`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-X01, I306-X02, I306-X03, I306-T10, I306-T11, I306-T13; PD-OWN.
- **Acceptance:** Codex-only/Claude-only unique permitted candidate disclosed/confirmed; both explicit assignment supports mixed Runtime stage; r1 bytes unchanged; templates not auto-selected; local configuration/policy/revision publication/selection/start separate previews/authorities.
- **Negative scenarios:** Unresolved multiplicity; allowlist/model mismatch; initiating vendor priority; local preference invented as portable intent; r1 replacement; new template auto-selects Project or changes frozen Execution.
- **Expected Evidence:** Default catalog digest tests and synthetic full setup-to-ready scenarios for four installed states and existing selections.
- **Flags / gates:** security-review, persisted-contract, migration, ownership:PD-OWN, external:#305.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-X07 — Verify the cross-cutting compatibility chain

- **Status:** Not started; blocked by PD-OWN and common authorization gates.
- **Objective / implementation scope:** Consolidate versioned local/Workflow/StagePlan/Execution/graph corpus, every-reader/writer recognition and integration faults. Verify new admission and immutable old inspection without duplicating unit checks.
- **Contract references:** AC-PE1/2/3; AC-ON3; AC-SEC; FR-026.
- **Likely components/files:** `Affected Workflow/compiler/Execution/graph integration tests; append-only compatibility corpus and Evidence`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-X01, I306-X02, I306-X03, I306-X04, I306-X05, I306-X06; PD-OWN.
- **Acceptance:** End-to-end synthetic migration publishes only reviewed owner effects; restart preserves all historical hashes; version matrix covers read/new admission/retry/forward/unsupported downgrade; faults block safely with exact remediation; proof gaps classified.
- **Negative scenarios:** Old approval/new family reuse; legacy continuation silently works; corpus replacement; partial migration selects active revision; CI/fake results claimed live or #278 acceptance.
- **Expected Evidence:** Exact-head integration compatibility matrix, original/new hashes, command/results and classified missing native/proof gates.
- **Flags / gates:** security-review, persisted-contract, migration, ownership:PD-OWN, external:G-3/#276/#278.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

## Phase 5 Tasks

### I306-T11 — Install the thin axiom-runtime skill in both Runtimes

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Add one canonical embedded product skill/catalog and native Codex/Claude discovery using existing installers/upgrade ownership. Guide safe intents and canonical CLI only. Preserve historical receipts, recognized migration sets and retired wrappers.
- **Contract references:** LA-11; ADR-0023; D4; Spec004 inventory annotation.
- **Likely components/files:** `internal/codexruntime/skills and catalog; Claude integration renderer/discovery; runtime installation/upgrade inventories`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T09.
- **Acceptance:** Same management semantics both integrations; help/inspect resolves executable operations; create/edit/remove/recovery requests exact confirmation; no duplicated validation/store/selectors; fresh/upgrade/reinstall sees exactly four active skills while historical receipts retained.
- **Negative scenarios:** Manual JSON/credential edit guidance; implicit apply/auth probe; retired alias restored; stale/conflicting inventory blindly replaced; new runtime-specific business rules; raw secret prompt reflection.
- **Expected Evidence:** Installer/catalog/owned-retirement tests, canonical CLI spies and synthetic conversational intent→preview→confirm parity cases.
- **Flags / gates:** security-review, migration, native-integration.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T12 — Detect passive first-use readiness without forced interaction

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Reuse deterministic passive detection separately from runtimebootstrap.Run. Classify installed/integrated/auth/model/effort facts; route first configuration-dependent missing state to skill. Preserve valid state and noninteractive CLI installation.
- **Contract references:** LA-12; ON-01/03; AC-ON1/2/4.
- **Likely components/files:** `internal/runtimebootstrap detection; CLI first-use/readiness; runtimeapplication inventory; skill next-action guidance`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T09, I306-T10, I306-T11.
- **Acceptance:** None/Codex-only/Claude-only/both use absolute observed executables deterministically; no absent binding; installed subset user-chosen; valid existing bytes unchanged; legacy/pending/unsafe produces explicit migration/recovery; unobserved auth does not block save or trigger probe, but blocks operational readiness.
- **Negative scenarios:** Calling bootstrap Run during read; vendor process/credential read; interactive installer forced; template overwrite; shell/PATH ambiguity wrongly ready; detected Runtime silently selected.
- **Expected Evidence:** Injected filesystem/PATH/process spies, fresh/pre-existing installation cases and next-action/diagnostic goldens.
- **Flags / gates:** security-review, native-integration.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T13 — Guide neutral templates through confirmed local setup

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Distribute descriptive fast/balanced/deep guidance and conversational candidate preparation. Present only provenance-backed options, accept explicit unproven intent honestly and hand off observations separately. Local setup does not author Project policy or Workflow.
- **Contract references:** LA-02/12; ON-02/03/04; AC-ON1/2/4; ADR-0023.
- **Likely components/files:** `Embedded axiom-runtime guidance/templates; runtimeapplication safe-option projections; CLI inspect/result`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T08, I306-T11, I306-T12.
- **Acceptance:** One Profile per selected installed Runtime sufficient structurally; three templates optional and >3 operator names supported; no hardcoded model/token/universal tier; exact candidate confirmation/apply/restart read-back; unknown model/effort/auth distinct; unsupported required capability blocks use; existing config preserved.
- **Negative scenarios:** Invented recommendation from chat model; effort translated by tier/complexity; inferred default; absent Runtime configured; probe consent implies apply/start; forced three Profiles; current auth proof treated model support.
- **Expected Evidence:** Synthetic user journeys with option provenance and confirmation records, one/three/four Profiles per Runtime and zero unauthorized owner effects.
- **Flags / gates:** security-review, external:G-3.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

## Phase 6 Tasks

### I306-T14 — Verify local authoring and CLI/skill integration

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Exercise production authoring in isolated clean/pre-existing state without fixture-writing configured state. Use injected observations only for synthetic capability scenarios; integrate CLI/application/skill/readiness and migration/remove/recreate faults.
- **Contract references:** Issue acceptance IA-01–07; LA-01–12; AC-ON1/2/4; AC-SEC.
- **Likely components/files:** `Runtime application/CLI black-box suites; Project readiness integration; synthetic skill harness`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T06, I306-T07, I306-T09, I306-T10, I306-T11, I306-T12, I306-T13.
- **Acceptance:** Both Runtimes/four Profiles each via supported authoring; two Projects share/remove/repair safely; absence/unknown/disabled/mismatch/ambiguous/stale auth/effort diagnostics; exact confirmation/restart parity; snapshots unchanged except reviewed local records; no Doctor.
- **Negative scenarios:** Fixture config counted authoring proof; synthetic observation counted native; process/network during preview; credential leakage; partial effects hidden; draft changes after confirm accepted.
- **Expected Evidence:** Exact-head black-box journal, zero-effect/secret canaries, safe hashes and focused tests; wider vet/build/race only where affected/risk warrants.
- **Flags / gates:** security-review, migration.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

### I306-T15 — Prove supported native filesystem and installation boundaries

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Validate actual supported POSIX/Windows private roots, permissions/ACLs/reparse paths, anchored identity, locking, fault/recovery and Runtime integration inventory in isolated state. These native tests run Axiom only; real vendor model/auth tests are not granted here.
- **Contract references:** LA-10/11/12; AC-SEC; ADR-0005/0010/0019; Security residual risks.
- **Likely components/files:** `Native local store/CLI/runtime installation tests; platform Evidence`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T04, I306-T05, I306-T11, I306-T12.
- **Acceptance:** Supported native environments show refusal for unsafe ownership/ACL/symlink/hardlink/reparse paths; contention and replacement protect canonical state; fresh/upgrade/reinstall inventory truthful; recovery/refusal retains objects; unsupported/unavailable platforms classified and required gaps remain open.
- **Negative scenarios:** Stub filesystem PASS called native; vendor login/model probe smuggled into install test; broad cleanup; altered stable user state; unavailable platform claimed accepted; existing supported threat exclusions widened.
- **Expected Evidence:** Per-platform exact-head native receipts PASS/FAIL/SKIPPED/UNAVAILABLE and independent security review. No authentication/inference receipt.
- **Flags / gates:** security-review, native-filesystem, native-integration.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

## Phase 7 Tasks

### I306-T16 — Consolidate G-2 Evidence and external readiness handoff

- **Status:** Not started; awaiting Tasks approval and bounded implementation authority.
- **Objective / implementation scope:** Reconcile durable contracts/help/docs and Evidence for local authoring delivered; integrate separate cross-cutting receipts when available, with ownership trace. Replay G-2 preparation using actual authoring, not fixture config; report G-3/G-4/dispatch/delivery/real acceptance independently.
- **Contract references:** Plan §8; Issue IA-01–07; AC-PE1–3/ON1–4/SEC; #303 Lane L / #275 C–G.
- **Likely components/files:** `Issue-scoped implementation Evidence; command docs; specs/ADR indexes as behavior changes; acceptance matrix`. These are investigation targets, not permission for unrelated refactoring.
- **Dependencies:** I306-T14, I306-T15, I306-X07.
- **Acceptance:** Each accepted criterion maps exact commit/blob/command/result/category; historical hashes/receipts preserved; G-2 removed blockers distinguished from unresolved cross-cutting/G-3/G-4; security review attached; no automatic Issue closure/human acceptance; full preparation blocked if dependency absent.
- **Negative scenarios:** CI accepted as MVP; external owner inferred green; omitted unavailable platform/proof; Tasks checked complete without authority; operation journal leaks raw vendor output.
- **Expected Evidence:** Integrated classified Evidence and independent human review. Native R-2/R-3 remains #278 with separately authorized G-3/#276/#277 work.
- **Flags / gates:** security-review, external:G-3/#276/#277/#278.
- **Objective completion condition:** Shared Done contract above plus every assertion and expected receipt of this unit; unsupported/unavailable required capability leaves its use blocked and accurately reported.

## Complete acceptance traceability

Issue acceptance aliases below identify the seven original Issue #306 checkboxes in order; they do not edit or mark the Issue accepted. Accepted D2/D3 amendments govern newer required effort/no-credential semantics where the older Issue body still says optional effort/credential references.

| Source criterion | Owning Tasks and verification |
| --- | --- |
| IA-01 clean state, both Runtimes, multiple named Profiles/conversation | T02, T08–T09, T11–T14; four per Runtime via authoring; native integration T15; real conversation #278 |
| IA-02 exact Project intersection/fail-closed auth/model | T03, T07, T10, T14; operational proof G-3, no credential backend |
| IA-03 two Projects/removal/immutable records/future repair | T04–T05, T08, T10, T14; X04/X07 historical bindings |
| IA-04 truthful capability/effort/auth/stale/ambiguity/redaction | T03, T07, T10, T13–T14; X03/X05; positive proof G-3 |
| IA-05 exact mutation authority/no credential disclosure | T01, T04–T09, T14–T15; X02/X05/X07 |
| IA-06 independent contract/security acceptance/no Doctor | Receipt reconciled as Accepted design; T07/T10/T14–T16 future code review; Tasks/implementation gates still separate |
| IA-07 G-2-specific Lane L C–G blockers; no automatic R-3 | T14–T16 plus X07 integration; G-3/G-4/#276/#277/#278 remain separate |
| D1; LA-03/10; AC-SEC | T02, T04–T06, T08, T14–T15; X07 compatibility |
| D2; LA-06/08 | T06–T07, T10, T12–T16; X05 safety boundary; G-3/#276 native authority |
| D3; LA-07; PE-01–06 including PE-05a | T02–T03/T06/T07; X01–X05/X07; sole Profile owner, exact tuple and bounded source check |
| D4; LA-11; ADR-0023 | T09, T11–T15; X06 separate Workflow route |
| D5; LA-04/09 | T01, T04–T05, T08/T10/T14; X02/X05/X07 stale/impact checks |
| LA-01/02/05 | T01/T02/T07–T09/T13/T14; one store, bounds, safe projection |
| LA-12; ON-01/02/03/04/05 | T11–T15; X06/X07; first-use and explicit multi-owner journey |
| AC-PE1 sole owner | T02–T03; X01/X03/X05/X07 |
| AC-PE2 preservation/migration | T02/T05/T06; X01/X02/X04/X07 |
| AC-PE3 snapshot/proof/source drift | T03; X03–X05/X07; real evidence G-3/#276 |
| AC-ON1 none/one/both, one/three/>three Profiles | T02/T12–T15; X06 installed-state default matrix |
| AC-ON2 preservation/consent/zero effects/read-back | T04–T09/T11–T15; X02/X06/X07 |
| AC-ON3 default-sdd and ambiguous both | X01–X03/X06/X07; T10/T13 readiness/setup |
| AC-ON4 unavailable facts/no fallback | T03/T07/T10/T12–T14; X03/X05/X06 |
| Plan §8 all seven Evidence rows and additional acceptance | IA rows and AC rows above; final classified T16 receipt |
| Security threat table + residual risks | T02 strict wire; T04/T05 lifecycle/identity/faults; T07 redaction/auth; T08 authority; T15 native; X05 source/dispatch; final independent code review |
| FR-026 all version families/readers/writers | T02/T05/T06; X01–X07; append-only corpus/new-format recognition |

## Stop and review

Tasks are complete as a proposed decomposition, not execution results. Review exact Tasks artifact and PD-OWN recommendation; complete the outstanding source-convention verification. Then request bounded implementation authority only after exact Tasks approval. **Awaiting Human Approval — Tasks. STOP.** Do not start implementation, run vendor probes, mutate configuration/Providers, mark units completed, mark PR Ready, merge/release or change Issue operational state.
