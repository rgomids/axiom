# Tasks — Specification 002: Lingo Project Initialization

## T04 implementation authorization and review gate — 2026-09-16

**Specification: Approved · Plan: Approved · Tasks: Approved · Implementation: Authorized (T04 only).**

Human authorization covers only **T04 — Strict local installation record codec**.
[PR #8](https://github.com/rgomids/axiom/pull/8) merged T03 into `main` on
2026-09-16 at 15:00:21 UTC. Verified baseline:
`1de3b02d97818138c32f6b2a07555cfe1ffd4a9b`.
[T04 Evidence](evidence-t04.md) records implementation SHA, closed local format,
codec/static-boundary checks and unproven filesystem properties.

**T01: Accepted / merged. T02: Accepted / merged. T03: Accepted / merged.
T04: Ready for human implementation review. T05–T21: Not started.**
T04 merge grants no T05 authority. No self-approval or merge is authorized.
Approved contract bodies, Task definitions and ADRs remain unchanged. Earlier
entries below preserve history and are superseded only as lifecycle status.

## Historical T03 implementation authorization and review gate — 2026-09-15

**Specification: Approved · Plan: Approved · Tasks: Approved · Implementation: Authorized (T03 only).**

[PR #7](https://github.com/rgomids/axiom/pull/7) merged into `main` on
2026-09-15 at 03:37:48 UTC. Baseline:
`378338cc0af79eaaec8b17e3cada44629515c043`. Explicit human authorization covers
only **T03 — Strict manifest decoding and canonical encoding**.
[T03 Evidence](evidence-t03.md) records the exact implementation SHA, parser
experiment, tests, resource limits and exclusions.

**T01: Accepted / merged. T02: Accepted / merged.
T03: Ready for human implementation review. T04–T21: Not started.**
T03 merge grants no T04 authority. No self-approval or merge is authorized.
Approved H1–H11, schemas, Task definitions and ADRs remain unchanged; prior entries
below are historical and superseded only as lifecycle status.

## Historical T02 implementation authorization and review gate — 2026-09-14

**Specification: Approved · Plan: Approved · Tasks: Approved · Implementation: Authorized (T02 only).**

GitHub confirms [PR #6](https://github.com/rgomids/axiom/pull/6) merged into `main`
on 2026-09-14 at 19:58:54 UTC. Updated `main` baseline:
`ffc9af9b7158de62f94671472c9ddedf36439802`. T01 is incorporated at that revision.
The subsequent explicit human request authorizes only **T02 — Consumer-owned ports,
authority and outcomes**. [T02 Evidence](evidence-t02.md) records implementation,
exact delivery SHA, contract tests and limitations.

**T01: Accepted / merged. T02: Ready for human implementation review.
T03–T21: Not started.** No merge or next-Task authority follows from this delivery.
H1–H11, approved task definitions, Plan contracts and ADRs remain unchanged.
Earlier lifecycle entries below are historical; this entry supersedes status only.

## Historical T01 implementation authorization — 2026-09-14

**Specification: Approved · Plan: Approved · Tasks: Approved · Implementation: Authorized (T01 only).**

Human confirmed [PR #5](https://github.com/rgomids/axiom/pull/5) reviewed and merged,
with `main` at `f70c7e5e762c408db3093753e6dabbc74d65f035` as the implementation baseline.
Explicit authorization covers only **T01 — Project values and complete-state invariants**.
H1–H11, approved task definitions, Plan contracts and Accepted ADRs remain unchanged.
[T01 Evidence](evidence-t01.md) records delivered domain behavior and reproducible checks.
**T01: Ready for human implementation review. T02–T21: Not started.**
No advance to another delivery unit, self-approval or merge is authorized.
Earlier lifecycle entries below are dated historical evidence, superseded only as status.

## Historical status and authority before PR #5 approval

- **Plan: Approved** — PR #4 approved and merged on 2026-09-14; subsequent explicit human request authorizes Tasks only.
- **Tasks: In review** — Ready for human Tasks review.
- **Implementation: Not authorized** — all 21 Tasks below are unstarted future work.
- Baseline: current `main` / `origin/main`, `b85657ff23dcfc40fbd9ea05b7a3876959b3554e`, merge of [PR #4](https://github.com/rgomids/axiom/pull/4).

Authority chain: [Intent and journeys](spec.md#intake) → [Specification](spec.md)
→ [Clarifications H1–H11](clarifications.md) → Accepted
[ADR-0001](../../decisions/0001-project-is-not-repository.md),
[ADR-0002](../../decisions/0002-axiom-speckit-relationship.md),
[ADR-0003](../../decisions/0003-lingo-as-axiom-local-control-plane.md),
[ADR-0004](../../decisions/0004-portable-project-manifest.md) →
[Approved Plan](plan.md) → these Tasks → future authorized Implementation → Evidence.
The Plan at the baseline supplies technical authority; this document decomposes it,
without reopening decisions. H8's historical Plan gate is satisfied by the later
approval; H1–H11 and ADR text remain historical authority, not rewritten approvals.

Only this artifact and minimal lifecycle documentation are delivered now. No
production code, Go module, dependencies, local adapters, executable CLI, tests of
Lingo, new CI or partial Implementation are introduced. Future paths below are
Plan §1 expectations, not files to create during Tasks review. Human Tasks approval
and explicit Implementation authorization are required before starting any Task.

## Shared contract and completion rules

Every Task inherits the Specification and Plan, including these constraints:

- Project != Repository; immutable canonical UUID v4; mutable installation-unique
  slug and nonunique name. Portable `projects/<slug>/` and local
  `state/projects/<id>/` stay separate. Minimal init is valid.
- Init = create/no-op/conflict. Update = explicit mutation. Mutation flow:
  partial intent → load current Project → domain materializes complete proposed
  Project → validate complete state → applicable authority/version/concurrency
  checks → safe diff/confirmation → recheck exact revision/write set → persist
  complete valid result. Init has no prior Project when creating; read-only flows
  and local-only mutations do not gain portable-write authority from this flow.
- CLI/presentation → application/use cases → domain/contracts → consumer-owned
  ports → local adapters describes dispatch. Source imports point inward: domain
  imports no filesystem, YAML, CLI, port or adapter; application owns its ports;
  adapters implement them. No generic CRUD, Workspace/global persistence layer,
  DI framework, Provider/Runtime execution or generic command runner.
- H10 specifies logical Project atomicity: no mixed manifest/document state is
  accepted as valid; old complete state is authoritative before commit, new complete
  state after commit. Post-commit failures never imply false rollback. Adapter
  chooses/proves commit point, reader checks, durability and recovery. No staging
  layout, lock protocol, syscall, transaction engine or cross-root transaction is
  mandated by these Tasks. Unsafe/unsupported guarantees fail closed.
- H11 separates Project mutation, optional local Git commit and explicitly
  authorized remote sync/push. Current slice has no Git execution. No backing/sync
  engine, branch strategy, merge/rebase, remote creation, push, autoPush, hooks or
  credential helpers. Backing creates no Repository association; unsupported
  Git-backed mutation fails safely and preserves metadata pending future design.
- No parser library is approved here. T03 selects one only during authorized
  Implementation after contract fixtures pass. Durable architecture or authority
  expansion requires human decision; a material approved Plan/ADR contradiction
  stops affected work and becomes a finding.

Each Task includes its tests as part of delivery: failing behavior tests first where
practical, then implementation, passing results and proportional review. IDs in
**Traceability** refer to exact definitions in spec.md and clarifications.md;
**Plan / boundary** uses numbered Plan sections. Required Evidence records Task →
FR/SEC/AC → component/test case → command, exit status, revision and platform →
result/artifact, including skips and limitations. Public artifacts use synthetic
inputs and sanitized paths/results. Unit tests do not prove filesystem guarantees;
repository checks do not prove Lingo acceptance. No Task is complete on test names
or intended results alone.

## Dependency graph

The table is the authoritative DAG. Dependencies are completion prerequisites;
all Tasks additionally require the future Implementation gate. Numbers are stable
IDs, not a requirement to implement in numeric order.

| Task | Delivery unit | Depends on |
|---|---|---|
| T01 | Project values, complete-state invariants and equivalence | None |
| T02 | Application ports, authority and outcome contracts; test seams | T01 |
| T03 | Strict portable manifest codec | T01 |
| T04 | Local installation record codec | T01 |
| T05 | Confined access and create publication proof | T02 |
| T06 | Complete update/move persistence and recovery proof | T03, T05 |
| T07 | Native roots and ID-addressed local persistence | T04, T05 |
| T08 | Read-only complete validation | T02, T03, T07 |
| T09 | Init create/no-op/conflict use case | T08 |
| T10 | Byte-preserving reopen use case | T07, T08 |
| T11 | Complete-state explicit update use case | T06, T09, T10 |
| T12 | Slug rename and local continuity | T11 |
| T13 | Read-only checkout observations and matching | T02 |
| T14 | Install with unresolved local gaps | T09, T10, T13 |
| T15 | Explicit local binding/source replacement | T14 |
| T16 | Presence-only Runtime observations | T14 |
| T17 | Guided CLI and truthful outcome presentation | T12, T15, T16 |
| T18 | Security regression suite | T03, T06, T07, T13, T16 |
| T19 | Cross-platform fault/concurrency evidence | T12, T15, T16 |
| T20 | CLI acceptance and complete coverage audit | T17, T18, T19 |
| T21 | Post-Implementation documentation and Evidence reconciliation | T20 |

After T01, T02/T03/T04 can advance independently. After T02, T05 and T13 can
advance independently; T02 also supplies security/fault fixtures and denial spies
for subsequent Tasks. T06/T07/T08 can proceed when their own prerequisites pass.
T09 and T10 need not wait for one another. After local installation, T15/T16 and
update/rename work can proceed independently where prerequisites allow. T17, T18
and T19 converge at T20; T21 is last. T05 proves protected behavior before creation
is wired, following Plan §10; T06 proves complete artifact mutation before update.

## Task definitions

### T01 — Project values and complete-state invariants

- **Objective:** Represent minimal valid Project and deterministic proposed-state validation.
- **Scope:** UUID v4 version/variant and immutable identity; full ASCII slug grammar before path use; mutable nonunique name; Repository keys and conservative locator normalization; all field-specific declaration states, configured shapes and reference integrity. Materialize complete proposed state from current state plus partial intent, including retained fields; compare normalized intent without collapsing absence/unconfigured/empty or nested presence.
- **Dependencies:** None
- **Traceability:** FR-001, FR-002, FR-004, FR-005, FR-006, FR-007, FR-009, FR-012, FR-017, FR-018; AC-01, AC-02, AC-04, AC-05, AC-11, AC-13, AC-15; H2–H4, H9; ADR-0001/0004
- **Plan / boundary:** §1, §2, §4, §5; pure domain
- **Expected paths:** internal/project/
- **Completion criterion:** Minimal and configured Projects validate; invalid identity/slug/duplicate/dangling states fail; omitted patch values retain intent; changing declaration state changes equivalence.
- **Required Evidence:** Unit required/optional and nested declaration-state matrices, valid/invalid UUID/slug, host-only normalization and case-sensitive path cases, name-only intent retention and removal leaving an invalid retained reference. No I/O in domain tests.
- **Explicit exclusions:** YAML DTOs, filesystem resolution, identity allocation before target inspection, vendor catalogs, global ownership, generic Repository CRUD.

### T02 — Consumer-owned ports, authority and outcomes

- **Objective:** Make all application I/O and approval boundaries substitutable and narrow.
- **Scope:** Contracts for complete artifact snapshots, expected portable/local revisions, authorized destinations, identity allocation, codec, stores, observations and optional text inspection. Define preview-bound authority, safe ordered issues, distinct portable/local/readiness outcomes and actual commit status. Add synthetic fixtures, fake entropy/clock, denied-write/process/network/secret-read spies and fault/barrier seams for downstream tests.
- **Dependencies:** T01
- **Traceability:** FR-008, FR-010, FR-011, FR-013, FR-016, FR-018, FR-019, FR-021; SEC-001, SEC-002, SEC-004; AC-07, AC-08, AC-10, AC-12, AC-15, AC-18; H5, H7, H9–H11; ADR-0003/0004
- **Plan / boundary:** §1, §4, §6, §7, §9; application owns ports
- **Expected paths:** internal/projectapp/; colocated tests and testdata/
- **Completion criterion:** Ports cannot accept arbitrary path writes or manifest patches; authority identifies exact revision/write set; result contracts distinguish pre-commit failure, committed local failure and recovery-required uncertainty.
- **Required Evidence:** Unit contract tests for revoked/missing/stale authority, deterministic issue order, classification matrix and denied side effects; static dependency inspection for inward imports.
- **Explicit exclusions:** Concrete storage protocol, Git port/engine, generic command runner, global persistence abstraction, executable AI or Runtime integration.

### T03 — Strict manifest decoding and canonical encoding

- **Objective:** Round-trip approved portable intent through exactly one strict axiom.yaml.
- **Scope:** One YAML document; string keys; all-depth duplicate/unknown key rejection; no anchors, aliases, merge keys or custom tags. Exact integer schemaVersion 1; closed mappings and scalar typing; explicit byte/depth/node limits. DTO/domain mapping, declaration-state preservation, normalization, stable ordering, UTF-8/final newline and canonical equivalence. Structural secret/URL rejection and sanitized parser diagnostics. Select parser only after proving subset behavior.
- **Dependencies:** T01
- **Traceability:** FR-008, FR-009, FR-012, FR-015, FR-016; SEC-001; AC-01, AC-04, AC-06, AC-08, AC-11, AC-12, AC-15; H3, H8, H9; ADR-0004
- **Plan / boundary:** §2, §5, §7–9; manifest adapter consumes domain
- **Expected paths:** internal/manifest/; internal/project/; colocated testdata/
- **Completion criterion:** Every approved shape/form round-trips semantically; equal validated input including ID encodes identically; malformed or unsupported input yields safe issues without coercion or writes.
- **Required Evidence:** Unit golden valid/invalid schema and all-depth abuse fixtures; pairwise declaration/nested-presence equivalence and round-trip tests; bounded parser results; diagnostic sentinel exclusion; parser-selection rationale and actual contract test results.
- **Explicit exclusions:** Library adoption in Tasks phase, schema expansion/migration, format preservation on writes, rewriting source during reopen/install, split manifests.

### T04 — Strict local installation record codec

- **Objective:** Validate internal local format before reusing bindings.
- **Scope:** Plan §3 record fields only: integer formatVersion 1, UUID, observed slug/source, manifest/document digests, local revision, bindings/reference metadata, observations/attempt metadata. Keep schemaVersion portable-only; no secret values or document bodies. Missing record differs from existing malformed/unversioned record.
- **Dependencies:** T01
- **Traceability:** FR-008, FR-014, FR-016; SEC-001, SEC-005; AC-03, AC-06, AC-07, AC-08, AC-12; H1, H8; ADR-0004
- **Plan / boundary:** §3, §7, §9; local record adapter
- **Expected paths:** internal/local/; colocated testdata/
- **Completion criterion:** Missing/malformed/duplicate version, null/bool/string/non-integer/unknown integer and schemaVersion-only records fail before reuse; valid record preserves local reference metadata and ID.
- **Required Evidence:** Unit JSON/version matrix and round trips; no binding reuse on failure; portable DTO cannot encode local fields; no credential/body serialization.
- **Explicit exclusions:** Portable schema changes, migration, fallback overwrite, actual record writes, database/catalog.

### T05 — Confined access and create publication

- **Objective:** Prove safe read/write boundaries and no-replace creation before wiring init.
- **Scope:** Explicit authorized directory identity and allowed relative names; safe manifest/document reads; portable artifact allowlist and containment; create publication, target inspection, installation-wide slug coordination seam and expected revisions. Protect ancestor/leaf replacement, symlinks, hard links and cleanup. Define chosen commit point, reader validity, durability and recovery with synthetic complete artifacts; fail closed if unsupported.
- **Dependencies:** T02
- **Traceability:** FR-012, FR-013, FR-017, FR-020; SEC-003, SEC-004, SEC-005; AC-07, AC-09, AC-13, AC-17; H1, H2, H6, H10; ADR-0004
- **Plan / boundary:** §1, §3, §6, §8–10; portable local adapter
- **Expected paths:** internal/local/; filesystem integration tests/testdata/
- **Completion criterion:** Conflicting creates cannot both succeed; invalid slugs never reach paths; no authorized operation escapes or replaces unknown content; incomplete output is never read as valid. Supported filesystem assumptions and protocol documented with Evidence.
- **Required Evidence:** Linux/macOS integration with two processes and barriers, traversal/leaf/ancestor/hard-link races, short writes/disk-full/permission failure, unauthorized-tree hashes, pre/post-commit interruption and owned-only cleanup; unit port contracts.
- **Explicit exclusions:** Mandatory staging/lock/syscall/transaction choice, global storage engine, complete update/rename wiring, Git metadata mutation.

### T06 — Complete artifact update, move and recovery persistence

- **Objective:** Preserve H10 atomicity across manifest/documents and slug movement.
- **Scope:** Expected-revision complete artifact replacement and old/new destination authority; namespace collision protection across create/install/update/rename; coherent reader validation; explicit recovery and confined owned-only cleanup. Preserve old complete state before commit, new complete state after commit; classify uncertain outcomes safely. Unsupported trees/Git metadata preserved. Local reconciliation remains separate.
- **Dependencies:** T03, T05
- **Traceability:** FR-013, FR-017, FR-018; SEC-003, SEC-004; AC-07, AC-09, AC-14, AC-15, AC-16; H2, H4, H9, H10; ADR-0004
- **Plan / boundary:** §4, §6, §8–10; portable store adapter
- **Expected paths:** internal/local/; manifest/document integration fixtures
- **Completion criterion:** No mixed Project accepted by concurrent readers; stale revisions and occupied rename targets conflict; actual commit status drives recovery. Chosen protocol proves required behavior without imposing a cross-root transaction.
- **Required Evidence:** Linux/macOS adapter tests for document addition/removal, update/update and rename/update races, old/new content hashes, pre/post-commit crashes, cancellation after publication, active/ended/unknown attempt ownership and unsupported/cross-filesystem safe failure.
- **Explicit exclusions:** Patch persistence, implicit rollback after commit, prescribed filesystem mechanism, local bindings redefinition, Git-backed update engine.

### T07 — Native roots and ID-addressed local store

- **Objective:** Persist protected machine state separately from portable intent.
- **Scope:** Plan Linux/macOS native roots and LINGO_STATE_ROOT/LINGO_PROJECTS_ROOT precedence; empty/relative/unsafe override rejection; disjoint canonical roots; no portable working copy in known associated checkout. ID-addressed installation.json and expected local revision. Resolve slug across managed root and valid local source records; ambiguity fails safely. Restrictive ownership/modes/effective ACLs, digest tracking and local no-op.
- **Dependencies:** T04, T05
- **Traceability:** FR-012, FR-013, FR-014, FR-017; SEC-003, SEC-004, SEC-005; AC-03, AC-04, AC-07, AC-09, AC-13, AC-14, AC-17; H1, H2, H8, H10; ADR-0004
- **Plan / boundary:** §3, §6, §8–9; local store/discovery adapter
- **Expected paths:** internal/local/; native/override integration fixtures
- **Completion criterion:** Only validated ID addresses records; invalid records are preserved without binding reuse; local CAS prevents lost updates; equivalent record causes no timestamp/revision rewrite. Root override changes discovery, never authority.
- **Required Evidence:** Linux/macOS native roots in isolated accounts and override tests; unsafe ACL/root overlap/alias tests; local CAS races, record-write failures, unknown formats preserved and portable hashes unchanged.
- **Explicit exclusions:** Global registry/database, Workspace persistence, Windows implementation, silent permission broadening, secrets or machine state in portable backing.

### T08 — Read-only validate use case

- **Objective:** Report full portable validity separately from observations and warnings.
- **Scope:** Explicit source/validated slug resolution; strict decode then complete domain validation; referenced document containment/existence/readability and artifact allowlist. Optional text inspection produces safe finding/unavailable/failure warnings; never executes text or resolves credentials. Stable issue order and version diagnostics; no durable writes.
- **Dependencies:** T02, T03, T07
- **Traceability:** FR-007, FR-008, FR-009, FR-015, FR-016, FR-019; SEC-001, SEC-002, SEC-003; AC-06, AC-08, AC-10, AC-11, AC-12, AC-17, AC-18; H5, H6, H9, H11; ADR-0003/0004
- **Plan / boundary:** §2, §4, §7–9; application validation
- **Expected paths:** internal/projectapp/; application and adapter integration tests
- **Completion criterion:** All applicable full-state rules run; unsupported schema/unsafe refs fail with zero writes; scanner failure alone does not invalidate otherwise valid intent; text never grants authority.
- **Required Evidence:** Unit fake-port ordering/no-write tests and integration missing/unreadable document checks; scanner four-state matrix; deterministic diagnostics and no network/process/secret-source reads.
- **Explicit exclusions:** Parent discovery, document fetching, mandatory scanner service, authority from AI/text, command execution, source rewrite.

### T09 — Init create/no-op/conflict use case

- **Objective:** Create minimal Project without turning retry into update.
- **Scope:** Inspect target before entropy; reuse recognized UUID or allocate once for required creation. Retain draft values on correction; validate complete draft, preview exact portable write set, confirm/recheck, publish. No Repository/context document required. Equivalent intent no-op; changed ID/intent, unknown files or ambiguity conflict and direct to explicit update. Local installation composes later through T14.
- **Dependencies:** T08
- **Traceability:** FR-001, FR-010, FR-011, FR-012, FR-013, FR-017; SEC-002, SEC-004; AC-01, AC-04, AC-07, AC-11, AC-12, AC-13, AC-15; H2–H4, H10; ADR-0004
- **Plan / boundary:** §2, §4, §6, §9–10; create application
- **Expected paths:** internal/projectapp/; create integration fixtures
- **Completion criterion:** Minimal create is valid; equivalent retry writes/allocates nothing; every changed declaration state conflicts under init; cancellation before commit leaves no durable Project and after commit reports actual state.
- **Required Evidence:** Unit entropy/write counts, retained draft/correction and consent tests; integration minimal create/validate, each no-op form, pairwise changed-intent conflicts, permissions and cancellation outcomes.
- **Explicit exclusions:** Implicit update/force overwrite, local installation before portable validation, CLI business rules, Git effects.

### T10 — Byte-preserving reopen use case

- **Objective:** Reopen same Project and report stale local state without rewriting it.
- **Scope:** Load source by validated slug or explicit location; validate portable content; correlate local record by immutable ID and content/document revision. Report absent/stale bindings, changed source/ID, ambiguous records or recovery state. Reobservation seams report current facts once T13/T16 are available; no implicit binding replacement.
- **Dependencies:** T07, T08
- **Traceability:** FR-001, FR-012, FR-014, FR-016, FR-017; SEC-004, SEC-005; AC-01, AC-03, AC-04, AC-06, AC-07, AC-12, AC-14; H1, H2, H10; ADR-0004
- **Plan / boundary:** §3, §4, §6–7; reopen application
- **Expected paths:** internal/projectapp/; reopen integration fixtures
- **Completion criterion:** Reopen preserves manifest/documents and local bytes; externally edited supported intent revalidates, old observations cannot prove readiness; malformed local format fails locally without invalidating portable intent.
- **Required Evidence:** Unit ID/revision classification; integration before/after hashes including comments/formatting, changed documents, copied source, unsupported local format and interrupted-state recovery result.
- **Explicit exclusions:** Automatic serialization, registration/rebinding, trusting cached availability, new identity allocation.

### T11 — Explicit complete-state update use case

- **Objective:** Evolve existing Project through validated, authorized complete state.
- **Scope:** Load snapshot/revision, apply partial intent through domain, validate complete proposed artifact set, preview safe diff, confirm exact revision/write set and recheck authority/version/concurrency. Support all Plan fields and supplied draft documents; resolve retained dangling references explicitly. Persist complete result; invalidate affected local observations after commit using local revision, preserve unrelated bindings.
- **Dependencies:** T06, T09, T10
- **Traceability:** FR-001, FR-005, FR-007, FR-009, FR-012, FR-013, FR-018, FR-019; SEC-001, SEC-002, SEC-004; AC-04, AC-05, AC-07, AC-11, AC-15, AC-16; H3–H5, H9, H10; ADR-0004
- **Plan / boundary:** §2, §4, §6, §9–10; update application
- **Expected paths:** internal/projectapp/; update integration fixtures
- **Completion criterion:** Name-only intent retains UUID/other fields/forms; all supported add/remove/change operations obey complete validation; stale/revoked consent gives no commit; post-commit local failure reports committed Project with incomplete reconciliation.
- **Required Evidence:** Unit complete-state/no-direct-patch port assertions and invalid retained-reference cases; integration minimal init then Repository/document addition, declaration changes, stale preview/version, conflicting revisions and local failure after commit.
- **Explicit exclusions:** ID reassignment, direct YAML patching, implicit AI authority, Git commit/sync, mechanism selection; dedicated rename continuity belongs to T12.

### T12 — Slug rename with ID continuity

- **Objective:** Expose explicit rename as a protected complete Project update.
- **Scope:** Validate old/new slug, resolve installation-wide collisions, preview both locations, commit complete moved manifest/document state under expected revision. Reconcile observed slug/source locally by same ID, preserve bindings, invalidate only affected facts. Reject silent stale old-slug aliases; recover local continuity after successful move and local-record failure.
- **Dependencies:** T11
- **Traceability:** FR-001, FR-013, FR-017, FR-018; SEC-003, SEC-004, SEC-005; AC-13, AC-14, AC-15, AC-16; H1, H2, H4, H9, H10; ADR-0004
- **Plan / boundary:** §3, §4, §6, §9; rename application
- **Expected paths:** internal/projectapp/; rename integration fixtures
- **Completion criterion:** Success preserves UUID/documents/bindings at new slug; pre-commit collision/failure preserves prior location; post-commit record failure truthfully identifies committed location and recovery by ID.
- **Required Evidence:** Unit invalid/colliding slug and authority tests; Linux/macOS integration old/new lookup, same local record key, document hashes, TOCTOU, concurrent rename/update and move/local failure cases.
- **Explicit exclusions:** Global slug uniqueness, old-slug alias service, identity fork, forced overwrite, Git rename/sync strategy.

### T13 — Read-only checkout observations

- **Objective:** Bind explicit local paths using conservative evidence without executing Git.
- **Scope:** Check existence/directory and supported version-control metadata read-only; safe symlink resolution and canonical checkout identity. Compare remote locators with T01 rules; duplicate checkout and mismatch block binding; missing/multiple/unsupported metadata yields unresolved; suspected aliases require human resolution. Separate worktrees are not automatically one checkout.
- **Dependencies:** T02
- **Traceability:** FR-002, FR-003, FR-004; SEC-002, SEC-003; AC-02, AC-04, AC-09, AC-10; H1; ADR-0001/0003
- **Plan / boundary:** §4, §5, §8–9; local observation adapter
- **Expected paths:** internal/local/; checkout metadata fixtures
- **Completion criterion:** Facts distinguish mismatch/unresolved/not checked; remote reachability always not checked; no external process, hooks, filters, helpers, network or repository writes.
- **Required Evidence:** Unit normalization-consumer cases; integration local-only/two-provider, symlink aliases, worktrees, multiple remotes, missing/stale paths and unsupported layouts; deny-process/network ledger and unchanged checkout snapshots.
- **Explicit exclusions:** Git CLI, recursive executable config interpretation, semantic Provider identity, repository cloning or repair.

### T14 — Install with explicit unresolved gaps

- **Objective:** Install validated portable intent using machine-local bindings only.
- **Scope:** Read/validate portable snapshot before any local write; combine T13 facts, credential reference metadata and local record version. Wire current read-only checkout reobservation into reopen; collect missing/stale bindings for install, preview/confirm local destination, recheck source revision and publish record. Repeated equivalent install no-op. Same ID from another source requires relocation confirmation. Compose optional post-init install without undoing committed portable output on failure.
- **Dependencies:** T09, T10, T13
- **Traceability:** FR-003, FR-004, FR-008, FR-010, FR-012, FR-013, FR-014, FR-017; SEC-001, SEC-002, SEC-005; AC-02, AC-03, AC-04, AC-07, AC-08, AC-12, AC-13; H1–H3, H10; ADR-0004
- **Plan / boundary:** §3–6, §9–10; install application
- **Expected paths:** internal/projectapp/; two-machine integration fixtures
- **Completion criterion:** Absent checkouts/unsupported credential sources can yield installed-with-gaps; conflicting binding must be corrected or explicitly left unresolved. Source changing before publication aborts local commit. Local write failure is not successful installation.
- **Required Evidence:** Unit ordered-port/consent/reference-only tests; two-machine/native/override integration with identical portable hashes, unrelated paths, no-op write counts, source change, slug collision, invalid local version and portable-success/local-failure retry.
- **Explicit exclusions:** Credential reads/prompts/store setup, portable rewrite, clone/fetch, Provider/Runtime execution, readiness claims.

### T15 — Explicit local binding and source replacement

- **Objective:** Relocate local association/source only after confirmation and revalidation.
- **Scope:** Load ID/revision; preserve unrelated draft values; check replacement checkout/remote/duplicate identity via T13; require explicit replacement or same-ID source relocation confirmation. Revalidate complete proposed local snapshot against portable revision, then local CAS. Preserve portable bytes and immutable ID.
- **Dependencies:** T14
- **Traceability:** FR-003, FR-004, FR-012, FR-014; SEC-002, SEC-003, SEC-004; AC-02, AC-03, AC-04, AC-07, AC-09; H1, H2, H10; ADR-0001/0004
- **Plan / boundary:** §3–6, §9; local binding application
- **Expected paths:** internal/projectapp/; relocation integration fixtures
- **Completion criterion:** Unconfirmed/mismatched/duplicate replacement never commits; explicit unresolved choice is visible; successful replacement changes only authorized local data and conserves other bindings.
- **Required Evidence:** Unit correction/cancel/authority tests; integration stale path, symlink duplicate, same-ID source move, remote mismatch, concurrent local-record change and unchanged portable hashes.
- **Explicit exclusions:** Portable Repository CRUD API, remote rewriting, automatic rebinding, secret resolution, checkout writes.

### T16 — Presence-only Runtime observations

- **Objective:** Report three-state Runtime facts with an honest check basis.
- **Scope:** Explicit selected Runtime/path safe metadata checks; available for accessible regular executable, unavailable for observed missing/non-executable, unverified for absent selection/path, unsupported check or uncertain permissions. Reobserve stale facts through install/reopen; persist only through authorized local mutation. Profile mismatch remains unresolved/author-corrected, never model remapping.
- **Dependencies:** T14
- **Traceability:** FR-005, FR-006, FR-007, FR-014, FR-016; SEC-002, SEC-005; AC-03, AC-05, AC-10, AC-12; H1; ADR-0003/0004
- **Plan / boundary:** §4, §7, §9–10; local observer plus application wiring
- **Expected paths:** internal/local/; internal/projectapp/; presence fixtures
- **Completion criterion:** All states have safe basis; uncertainty is not absence, presence is not readiness, and unavailable/unverified does not invalidate Project/install. Reopen performs no record writes.
- **Required Evidence:** Unit three-state/basis matrix; controlled executable/missing/non-executable/permission-uncertain fixtures on Linux/macOS; zero launches/model/network checks and stale-observation tests.
- **Explicit exclusions:** Runtime/Provider adapters, discovery commands, model verification, executable file contents, command runner.

### T17 — Guided CLI and truthful outcomes

- **Objective:** Present tested use cases through explicit input, preview and confirmation.
- **Scope:** Composition root and thin guided init/update/install/reopen/validate/rename/binding actions. Preserve draft fields during correction; collect optional forms explicitly without account defaults; safe previews include authorized destinations and warnings. Handle cancellation and closed/noninteractive missing input promptly. Render deterministic actual outcomes, separate gaps/local failure/readiness, structured safe events and non-success for failure/cancellation.
- **Dependencies:** T12, T15, T16
- **Traceability:** FR-010, FR-011, FR-013, FR-016, FR-018, FR-019, FR-021; SEC-001, SEC-002; AC-01, AC-03, AC-04, AC-05, AC-07, AC-10, AC-11, AC-12, AC-14, AC-15, AC-16, AC-18; H3–H5, H9–H11; ADR-0003/0004
- **Plan / boundary:** §1, §4, §7, §9–10; CLI/presentation and composition only
- **Expected paths:** internal/cli/; cmd/lingo/; black-box testdata/
- **Completion criterion:** CLI delegates all validation/authority/persistence to application; numeric codes/command surface documented when selected in Implementation. No hanging missing-input flow, raw sensitive diagnostics, false rollback or implicit readiness.
- **Required Evidence:** Presentation unit tests and black-box controlled stdin/output fixtures for correction, previews, stale consent, cancel before/after commit, missing-input bounded completion and outcome/event classification.
- **Explicit exclusions:** Domain rules in CLI, approved framework/dependency now, force overwrite, log writes without explicit destination authority, Git/AI execution.

### T18 — Security boundary regression suite

- **Objective:** Provide explicit adversarial SEC evidence across codecs and local adapters.
- **Scope:** Synthetic fixtures cover credential-value structures, URL user-info/versioned sensitive-parameter policy, sentinel leaks, scanner finding/unavailable/failure, parser resource abuse, traversal, absolute refs, executable/environment interpolation, symlinks/hard links, ancestor races, overwrite, unknown cleanup, unsafe ACLs and portable/local overlap. Attack every write/cleanup/recovery phase. Deny process/network/secret reads and assert whole portable artifact exclusion.
- **Dependencies:** T03, T06, T07, T13, T16
- **Traceability:** FR-008, FR-009, FR-013, FR-015, FR-019, FR-020, FR-021; SEC-001, SEC-002, SEC-003, SEC-004, SEC-005; AC-06, AC-08, AC-09, AC-10, AC-16, AC-17, AC-18; H1, H5–H7, H10, H11; ADR-0001/0003/0004
- **Plan / boundary:** §2, §3, §5–9; security fixtures across existing boundaries
- **Expected paths:** Colocated internal/manifest/, internal/local/, internal/projectapp/ tests and testdata/
- **Completion criterion:** Invalid structures fail before writes; unauthorized trees unchanged; sentinels absent from files/temporary diagnostics/output/logs/Evidence. Shell metacharacters never execute. Unsupported guarantees fail closed; scanner warnings never claim complete secret absence.
- **Required Evidence:** Unit rejection/redaction/parser-limit matrix; Linux/macOS adversarial adapter integration, denial ledgers, permissions/ACL results and public-safe security review. T20 adds CLI end-to-end security assertions.
- **Explicit exclusions:** Real secrets, external scanner service, new authority, executing attack payloads outside controlled test targets, Git implementation or prescribed protection mechanism.

### T19 — Fault, recovery and concurrency evidence

- **Objective:** Prove complete-state outcomes on both supported operating systems.
- **Scope:** Exercise chosen adapter commit boundaries through application: create/update/install/rename/binding, manifest/document changes and local CAS. Independent writer processes and concurrent readers use barriers, not sleeps alone. Inject disk-full, short-write, permission/read-only, interruption/crash, cancellation pre/post commit, local failure after portable commit, active/stale/unknown recovery ownership and rename versus update.
- **Dependencies:** T12, T15, T16
- **Traceability:** FR-012, FR-013, FR-014, FR-017, FR-018; SEC-003, SEC-004, SEC-005; AC-03, AC-04, AC-07, AC-09, AC-13, AC-14, AC-15, AC-16; H1, H2, H4, H9, H10; ADR-0004
- **Plan / boundary:** §6, §8–10; real filesystem/application integration
- **Expected paths:** internal/local/ and internal/projectapp/ integration tests; sanitized Evidence
- **Completion criterion:** Conflicting writers cannot both succeed; readers never accept mixed state; pre-commit hashes survive, successful commit remains authoritative, uncertain outcome reports recovery-required. Retry no-op allocates/writes nothing; rename preserves UUID/bindings.
- **Required Evidence:** Separate Linux and macOS results with OS/filesystem/privilege assumptions, fault stage/commit status/outcome table, portable/local digests and process exit status. Permission cases use controlled unprivileged accounts; unavailable platform or bypassed case stays unverified and blocks completion.
- **Explicit exclusions:** Mocked-only filesystem proof, weakening tests for elevated privileges, auto-deleting unknown leftovers, false rollback, new CI or required transaction technique.

### T20 — Black-box acceptance and traceability audit

- **Objective:** Close current-slice AC coverage with observable CLI evidence.
- **Scope:** Run J1–J6 through eventual executable without internal imports, native/override roots on Linux/macOS, controlled stdin and two-machine fixtures. Cover all AC rows below with unit/integration prerequisites, actual CLI outcomes and static dependency checks. Deny Git/network/process/secret effects; preserve external Git/Runtime/MCP config snapshots. A preexisting .git never infers an association; unsupported backing writes preserve it and fail safely.
- **Dependencies:** T17, T18, T19
- **Traceability:** FR-001–FR-021; SEC-001–SEC-005; AC-01–AC-18; H1–H11; ADR-0001–0004
- **Plan / boundary:** §8–10; CLI/black-box acceptance and engineering/security review
- **Expected paths:** Black-box tests/testdata/; sanitized component-to-AC Evidence
- **Completion criterion:** Every applicable AC has reproducible passing Evidence on required platforms and reviewed limitations; current AC-18 proves denied effects/boundaries only. Inspect inward imports/no domain I/O and no scope expansion. Unmet required evidence blocks completion rather than being marked passed.
- **Required Evidence:** CLI commands/exit codes, safe stdout/stderr snapshots, hashes, no-op counts, schema/declaration matrices, security side-effect ledger, platform fault reports and reviewer findings. Static repository checks are reported separately; manual inspection only for usability/public-safe review or platform observations unavailable to automation, never substitute for required tests.
- **Explicit exclusions:** Actual Git commit/push failure tests or autoPush delivery (future Specification), production services, CI introduction, claiming current harness as Lingo acceptance.

### T21 — Post-Implementation documentation and Evidence reconciliation

- **Objective:** Reconcile delivered behavior and retained evidence after authorized Implementation.
- **Scope:** Update README run/test/stack/structure, docs/commands.md, applicable architecture/schema/operations docs and CHANGELOG from actual delivered behavior. Link Task/FR/SEC/AC to tests/commands/results and platform support, security findings, recovery/rollback guidance and limitations. Record subsequent human gates with dates; preserve approved decisions and original Evidence.
- **Dependencies:** T20
- **Traceability:** FR-015, FR-016; SEC-001, SEC-005; AC-01–AC-18 Evidence; H8, H10, H11; ADR-0001–0004; Constitution III–VI
- **Plan / boundary:** §9–12; documentation/Evidence reconciliation
- **Expected paths:** README.md; docs/commands.md; CHANGELOG.md; applicable docs/architecture/ and Specification 002 Evidence references
- **Completion criterion:** Docs describe only verified delivered behavior; full AC/security/platform matrix and reproducible commands are linked; future Git evidence stays explicitly deferred. Human release/review gate recorded, never inferred from agent tests.
- **Required Evidence:** Static/repository validation, all relevant executable test reports inherited from T20, relative-link/status checks and full diff/public-data review; list unavailable tools or remaining findings explicitly.
- **Explicit exclusions:** Retroactive decision rewriting, self-approved release/merge, new features, asserting deferred Git or untested platform delivery.

## Requirement ownership and explicit deferrals

Owners below implement or verify the indicated requirement; T20 audits coverage
and T21 reconciles results. A reference alone is not completion Evidence.

| Requirement | Responsible Tasks | Slice responsibility / limitation |
|---|---|---|
| FR-001 | T01, T09, T10, T11, T12 | UUID allocation/preservation; copy/reopen keep identity; fork and actual export/sync deferred |
| FR-002 | T01, T13, T14 | Stable association keys and optional locators; no ownership inference |
| FR-003 | T13, T14, T15 | Read-only checkout resolution, unresolved gaps and mismatch |
| FR-004 | T01, T13, T14, T15 | Conservative duplicates/aliases, checkout conflict and relocation |
| FR-005 | T01, T11, T16 | Opaque Runtime selection, presence basis, incompatible references |
| FR-006 | T01, T16 | Profile structure/reference validation; no discovery/model execution |
| FR-007 | T01, T08, T11, T16 | Independent optional declarations and missing/unverified bindings |
| FR-008 | T02, T03, T04, T08, T14, T18 | Reference metadata only, structural rejection and no secret reads |
| FR-009 | T01, T03, T08, T11 | Optional context/documents as untrusted data; AI draft assistance deferred |
| FR-010 | T02, T09, T14, T17 | Explicit collection, correction, preview and confirmation |
| FR-011 | T02, T09, T17 | Missing required input fails promptly; no weaker validation path; full flag API not required |
| FR-012 | T01, T03, T07, T09, T10, T14, T15, T19 | Target-first ID; per-form no-op, conflicts, explicit relocation |
| FR-013 | T02, T05, T06, T07, T09, T11, T14, T19 | Confinement, logical commit, recovery and truthful local failure |
| FR-014 | T04, T07, T10, T14, T15, T16 | Byte-preserving reopen/install and revalidation |
| FR-015 | T03, T08, T21 | Strict version 1 compatibility, schema documentation, no migration |
| FR-016 | T02, T03, T04, T08, T10, T16, T17, T21 | Stable issues, separate outcomes, safe events/Evidence |
| FR-017 | T01, T05, T06, T07, T09, T10, T12, T14, T19 | Slug before path use, namespace collisions and ID continuity |
| FR-018 | T01, T02, T06, T11, T12, T17, T19 | Complete proposed-state update and exact authority/revision |
| FR-019 | T02, T08, T11, T17, T18 | Caller-independent validation/authority; actual AI drafting is future scope |
| FR-020 | T05, T08, T18, T20 | Portable allowlist and backing/association independence only; export, Git init/origin/commit/publication and backing metadata require future Specification |
| FR-021 | T02, T17, T18, T20 | No implicit Git/network effects or authority; mutation outcome independent of Git. Actual commit/push failure handling, sync/automation and authority representation require future authorized design/Implementation |
| SEC-001 | T02, T03, T04, T08, T18, T20 | Structural secret rejection, scanner warnings, zero reads and full diagnostic/file sentinel coverage |
| SEC-002 | T02, T08, T13, T14, T16, T17, T18, T20 | Deny external config/Git/hooks/network/commands and AI authority |
| SEC-003 | T05, T06, T07, T12, T13, T15, T18, T19, T20 | Traversal, symlinks/hard links/ancestor races, confined access and recovery |
| SEC-004 | T05, T06, T07, T09, T11, T12, T15, T18, T19 | No overwrite/mixed validity; collision/CAS/recovery ownership |
| SEC-005 | T04, T07, T12, T14, T16, T18, T19, T20 | Native/override permissions and portable exclusion by construction |

FR-019–021 and AC-18 are deliberately split between current boundary proof and
future execution. No Task may satisfy their deferred portion by adding Git, AI,
network, Provider or Runtime execution. Future Git tests must cover independent
commit/push authority/outcomes, failure preserving confirmed local success,
portable-only backing and separately approved automation before Git delivery.

## AC-to-test and Evidence matrix

U = unit/domain/application with fake ports; I = real adapter/filesystem integration;
B = CLI black-box with controlled input and external state inspection. Every row
also receives T20 coverage audit and T21 documentary reconciliation. Linux/macOS
results are separate obligations wherever OS behavior is involved.

| AC | Responsible Tasks | Required future tests and retained Evidence |
|---|---|---|
| AC-01 | T01, T03, T09, T10, T17 | U required/optional matrix; I minimal create/reopen; B J1 UUID/schema/zero Repository/document round trip |
| AC-02 | T01, T13, T14, T15 | U conservative normalization/aliases; I independent providers/local-only/arbitrary paths on two machines; B binding outcomes and zero network ledger |
| AC-03 | T04, T07, T10, T14, T16, T17, T19 | I native and override roots on both OSes; B J3 portable hashes/UUID unchanged, equivalent intent with different paths and unresolved credentials/checkouts |
| AC-04 | T01, T03, T07, T09, T10, T11, T13, T14, T15, T17, T19 | U pairwise declaration/nested-presence equivalence; I/B no-op write/entropy/timestamp counts, changed init conflicts, duplicate keys/locators/checkouts, copy/edit/Runtime change preserving ID |
| AC-05 | T01, T11, T16, T17 | U optional forms/reference/profile matrix; I presence basis/uncertainty; B J2 missing/unconfigured Runtime/Providers/Integrations, no execution/readiness claim |
| AC-06 | T03, T04, T08, T10, T18 | U strict YAML/version/closed mappings/duplicate/anchors/alias/merge/tag and local format matrix; I/B invalid schemas produce safe errors and zero writes; semantic round-trip goldens |
| AC-07 | T05, T06, T07, T09, T10, T11, T14, T15, T17, T19 | I faults/crashes before/after actual commit, local failure and CAS races; B J5 cancellation/retry/recovery with preserved hashes and truthful commit state |
| AC-08 | T02, T03, T04, T08, T14, T18 | U structural URL/reference/value rejection and scanner four-state matrix; I/B no reads and sentinel absent from files, stdout/stderr, logs and Evidence |
| AC-09 | T05, T06, T07, T12, T13, T15, T18, T19 | I Linux/macOS traversal/symlink/hard-link/ancestor/ACL/TOCTOU attacks through write/cleanup/recovery; B unchanged outside-target snapshots and read-only binding resolution |
| AC-10 | T02, T08, T13, T16, T17, T18 | I denied effects; B init/update/install/validate with unchanged external Git/Runtime/MCP configuration, zero hooks/process/network/AI calls |
| AC-11 | T01, T03, T08, T09, T11, T17 | U context absence/unconfigured/untrusted text; B retained independent fields on correction, missing-input timeout and no interpolation/execution |
| AC-12 | T02, T03, T04, T08, T09, T10, T14, T16, T17 | U randomized order/stable issues and classification matrix; B repeated deterministic output excluding UUID allocation/attempt metadata; distinguish local-format failure, installed-with-gaps and failed installation |
| AC-13 | T01, T05, T07, T09, T12, T14, T19 | U full slug grammar and zero path/entropy use on invalid input; I two-process init/install/rename namespace collision; B conflict without occupied-target change |
| AC-14 | T06, T07, T10, T12, T17, T19 | U nonunique name/immutable ID; I move/TOCTOU/collision/local failure; B J6 old/new slug lookup, document hashes, same ID-addressed bindings |
| AC-15 | T01, T02, T03, T06, T09, T11, T12, T17, T19 | U partial intent to complete state, invalid retained reference, no patch persistence; I stale/revoked authority/version/revision zero writes; B changed init conflict versus explicit document/Repository/declaration update |
| AC-16 | T06, T11, T12, T17, T18, T19 | I both OSes concurrent readers/writers and crashes around actual commit, rename versus update, manifest/documents coherent; B pre/post-commit truthful reports, recovery and local continuity |
| AC-17 | T05, T07, T08, T18 | U artifact allowlist; I root overlap and forbidden full-tree content; B portable snapshot excludes all local state/credentials regardless of home location; no export engine |
| AC-18 | T02, T08, T17, T18, T20 | U boundary/authority denial; B zero Git/network effects for init/update/install/validate, .git never infers association, unsupported backing mutation preserves metadata. Actual Git/automation tests explicitly deferred above |

## Review gate and documentation Evidence

Self-review must inspect both these matrices and each Task's scope; counts alone
cannot establish semantic coverage. Every Task has Specification/Clarification,
Plan and ADR origins. Constitution I–IV keep Tasks subject to human approval;
V–VI require deterministic, sanitized Evidence; VII–VIII preserve small slice-local
boundaries and Project != Repository. No new architecture decision or waiver.

Current change validates documents/repository only. Planned U/I/B tests, Linux/macOS
logical atomicity, fault/concurrency/security and CLI acceptance remain unexecuted
Implementation obligations. No production behavior is claimed.

### Tasks documentation validation — 2026-09-14

- `./scripts/validate-repository.sh .` passed: harness/package validation, both
  Bash regression suites, worktree sensitive-file scan and whitespace validation.
  Individual `bash -n` checks passed for all five root scripts; `.env`/`.env.local`
  ignore and `.env.example` eligibility checks passed.
- Temporary documentation checks passed: 21 unique complete Task definitions,
  acyclic/table-consistent DAG, 44 FR/SEC/AC ownership rows, eight Markdown-only
  changed paths, 86 relative links/anchors and balanced fences/table columns.
  Specification contract body, Plan sections 1–11, H1–H11 and original clarification
  history remain byte-for-byte unchanged. ADRs and frozen experiments are untouched.
- Semantic self-review covers all Task origins, requirement owners, deferred
  FR-019–021/AC-18 scope, inward architecture, H10 mechanism neutrality and H11
  denied effects. No material blocker or Plan/ADR contradiction identified.
  README/index/roadmap/current gates match Plan Approved / Tasks In review /
  Implementation Not authorized. Historical pending gates remain explicitly dated.
- `gitleaks`, `markdownlint`, `markdownlint-cli2`, `lychee` and `shellcheck` are
  unavailable; dedicated scanner/linter/external-link coverage remains unverified.
  No dependency installed. Existing frozen experiment behavior is unaffected and
  was not rerun; Lingo tests and Linux/macOS acceptance remain future obligations.
- Full diff and staged paths/content reviewed; `git diff --check`,
  `git diff --cached --check` and
  `./scripts/check-sensitive-files.sh --staged .` passed.
- Commands/stack instructions need no operational change because no executable
  behavior is delivered. Full change is Tasks plus minimal lifecycle updates.

**Ready for human Tasks review. Implementation: Not authorized.**
