# Changelog

## [2026-09-20]

- specification: draft Specification 004 for human review, consolidating the MVP
  clean-environment journey, Project/Work Item UX, workflow visibility, Runtime
  selectors, completion output, detailed artifacts, provenance, persistence and
  compatibility, onboarding, acceptance Evidence, explicit non-goals, and four
  human decisions; clean v1 is the recommended baseline, bounded-threat-model
  approval requires Specification 002/security reconciliation before Plan, and no
  Plan, Tasks, implementation, ADR, or release authority is granted.

- acceptance: record explicit human acceptance of the merged E2E Codex POC; close #21/#30/#40, preserve #19/#20 as known proof gaps, and capture MVP follow-ups #54–#57 for guided Project setup, Intent-driven Work Item creation, provider-visible workflow progress and argument-driven Runtime skills.
- reconciliation: close the historical POC validation card #53 and POC-scoped #19/#20 after acceptance without claiming their residual proof gaps are solved; carry persistence/recovery hardening into future MVP specification instead of leaving stale POC work open.

- fix: make source installation report deterministic PATH setup guidance without shell-profile mutation, mark dirty-checkout builds in installer/version metadata, and preserve confirmed GitHub Issue identity/state through post-provider Work Item or workflow persistence failures.

## [2026-09-19]

- specification: approve the complete E2E Codex POC journey, implementation plan and #31–#40 task mapping; preserve existing Axiom/Lingo, Project/Repository and portable/local boundaries. Record the supported standalone Codex `$axiom-<skill>` mapping because literal colon names fail the current hyphen-case skill contract; final human acceptance remains separate.

- feature: add a source-checkout Axiom installer that publishes Lingo to an explicit user PATH directory, records checksum ownership, supports safe idempotent reruns, refuses modified/unowned destinations, and exposes build source/revision through `lingo version`.

- feature: add the Codex Runtime bootstrap with five validated user-global `$axiom-*` skills, exact status inspection, idempotent install, conflict/symlink protection and attempt-local rollback. Skills remain thin Lingo entrypoints; no Project or workflow rules are duplicated.

- feature: resolve installed Projects by UUID or slug from protected local state independently of caller CWD; preserve multiple repository bindings and fail explicitly for ambiguous selectors, missing sources, moved repositories and invalid local records.

- feature: configure a complete Project through guided or argument-driven Lingo, publish repository keys in portable intent and absolute working-copy paths only in strict local state, then resolve immediately from any CWD. Preserve separate portable/local commit truth on post-commit failure.

- feature: add bounded GitHub Work Item create/select/show/comment/complete capabilities behind provider-neutral ports, explicit external mutation authority, exact provider-reference validation, protected local linkage and bounded `git`/`gh` transports.

- feature: add the persistent sequential Axiom delivery workflow with fixed Specification-through-Reconciliation gates, repository-anchored artifact digests, inspectable interruption/resume, and explicitly authorized Work Item completion.

- feature: complete equivalent Lingo/Codex entrypoints with a real `project show` command, default human output, explicit typed `--json` payloads, command/skill help, exact thin-skill command templates and controlled upgrades of known prior Axiom skill content.

- test: expand `dogfood-poc.sh` into the complete isolated install, global-skill, unrelated-CWD Project, GitHub Work Item, interruption/resume, Evidence, reconciliation and completion journey; record a real user-global Codex discovery run for #39.

- docs: reconcile README, architecture, setup, roadmap, Specifications and final POC report to the delivered Codex/GitHub/sequential-workflow scope; preserve #19/#20 proof gaps and explicit human acceptance for #21/#30/#40.

## [2026-09-18]

- docs: reconcile Specification 002 and Plan scope for the POC-only Linux/macOS verification workflow; classify its runs as acceptance Evidence, not product CI or release automation. Preserve #19–#21, #14 and SEC-003 human gates.

- fix: check complete private-file writes, staged-file bytes/link count, opened directory identity and platform ACLs before one-file publication; add controlled storage-fault, process-conflict, writer-conflict, ACL and replacement-race regressions. Run POC tests and dogfooding on macOS/Linux in GitHub Actions; document manual recovery and remaining fault/race proof gaps without declaring acceptance.

- fix: anchor minimal POC filesystem operations to private directory handles, reject symlink/hard-link targets, publish create/install without replacement, keep validate read-only, and report interrupted attempts through durable markers as recovery-required. Add process crash-boundary and executable black-box tests plus reproducible dogfooding Evidence; Linux fault proof and human acceptance remain open.

- fix: honor cancellation observed immediately before update/install publication; clear the owned attempt marker and preserve the prior manifest or absent local record. Add deterministic pre-commit cancellation regressions.

- fix: align Lingo's default machine-local root with native macOS Application Support and Linux XDG state directories; reject invalid explicit root overrides. Reconcile Specification 002's POC delivery status while #19–#21 and human acceptance remain pending.

- feature: add the initial local Lingo POC lifecycle: `project init`, `validate`, `reopen`, and explicit name `update` for a minimal one-file portable Project. The adapter uses validated slugs, private staging for create, atomic manifest replacement for update, per-slug coordination and sanitized JSON outcomes. Optional documents, installation/local state, bindings, rename, runtime/provider integration, Git and network remain unsupported.

- feature: add explicit local installation for the minimal Lingo POC. `project install` reads a strict portable manifest and writes an ID-addressed `installation.json` outside the portable root without rewriting portable bytes. Credential resolution, bindings and automatic local-state reconciliation remain unavailable.

- docs: record consolidated POC lifecycle Evidence, reproducible checks and explicit filesystem/security limitations for final human review.

## [2026-09-17]

- docs: adopt Apache-2.0 and establish contribution, conduct, and support policies with structured GitHub Issue and Pull Request templates. No implementation or new Task authorization.

- docs: add the approved Axiom logo and centered README header with verifiable Go 1.26+, main last-commit, and repository-stars badges.

- docs: establish documentation governance with canonical sources, classification, lifecycle and anti-drift rules; align the agent documentation policy and README navigation.

- docs: reconcile product and agent context with the implemented Go foundation; separate roadmap/ADR direction from Specification lifecycle, preserve pending T04 acceptance, and make onboarding portable. Keep README and approved contracts unchanged.

- docs: rewrite the README as a project landing page with problem and audience, verified maturity, conceptual workflow, runnable exploration steps, and links to deeper documentation. Preserve detailed implementation history in existing technical documents; record the absence of a project license file.

## [2026-09-16]

- fix: reject invalid UTF-8 document names at the T02 snapshot producer for T04 lossless JSON metadata; preserve valid U+FFFD, # and ?, retain local defense in depth, and add producer/composition regressions. No T05+ work.

- fix: resolve PR #9 ArtifactDigest.Name compatibility finding: T04 reuses the unchanged T02 document-name predicate, preserves valid names including #, ?, controls and literal U+FFFD without normalization, and rejects malformed Unicode without repair. Add real manifest/Project/snapshot/digest/local-record round-trip and negative boundary regressions; preserve portable protections, H12 and all prior Evidence. No T05+ work.

- fix: apply human-approved H12 / Option B within T04: remove persisted localRevision from local format v1, retain unchanged T02 exact-byte LocalRevision and independent portableRevision metadata. Update closed DTO, mapping, fixtures and regressions; reject obsolete field without migration or reuse.
- docs: reconcile Specification/Clarifications/Plan/Tasks and T04 Evidence with explicit human authority; preserve earlier decisions and Evidence. T04 Ready for human re-review, not Accepted; T05–T21 remain unstarted.
- fix: address PR #9 human review within T04: preserve arbitrary local path text without filesystem policy; map domain identity diagnostics to local schema fields. Add public codec regressions; retain invalid-text rejection and shared domain rules.
- docs: investigate ambiguous persisted localRevision versus T02 exact-byte CAS; record both options and recommend single revision pending explicit human decision. T04 blocked on that decision; no revision-model change or T05+ implementation.
- feature: deliver authorized Specification 002 T04 strict local installation JSON codec with exact formatVersion 1, closed metadata DTOs, safe diagnostics, no partial records, explicit missing/corrupt distinction and consumer-owned T02 metadata reuse. Share existing domain identity validation without changing its rules. Add version/structure/security/round-trip tests and static boundary Evidence; no filesystem persistence or T05+ work.
- docs: record T01–T03 Accepted / merged, verified baseline `1de3b02d97818138c32f6b2a07555cfe1ffd4a9b`, T04 Ready for human implementation review and T05–T21 Not started. T04 merge grants no T05 authority; preserve prior lifecycle history and approved contracts.

## [2026-09-15]

- fix: address PR #8 human review within T03: reject relative/local URI paths in logical references and known credential payloads in identifier fields; bound canonical output during serialization. Add encode/decode, escaped-syntax, writer-bound and positive regressions; retain prior implementation Evidence. T01/T02 contracts unchanged; T04–T21 Not started; human re-review required.

- implementation: deliver Specification 002 T03 strict portable manifest codec, closed DTOs, canonical round trips, declaration-state preservation, resource limits, structural security rejection and sanitized diagnostics. Pin experimentally validated YAML parser v3.0.5; add golden, schema, abuse, bounds, pairwise, port integration, fuzz and dependency-boundary tests. T01/T02 Accepted / merged; T03 Ready for human implementation review; T04–T21 Not started. T03 merge does not authorize T04.

## [2026-09-14]

- implementation: deliver authorized Specification 002 T02 consumer-owned application ports, complete snapshots and revisions, exact preview-bound authority, safe ordered issues and truthful portable/local commit outcomes. Add contract/fault/barrier tests and deterministic dependency checks. T01 Accepted / merged (PR #6); T02 Ready for human implementation review; T03–T21 Not started. No concrete codec, persistence, CLI or Git execution.

- implementation: deliver authorized Specification 002 T01 pure Go domain: immutable Project identity, declaration/reference invariants, conservative locators, complete proposed-state materialization and presence-preserving equivalence. Add behavior tests and domain I/O boundary verification; retain reproducible Evidence. Specification/Plan/Tasks Approved; T01 ready for human implementation review; T02–T21 Not started.

Earlier Tasks review on this date (historical):

- planning: record approved/merged PR #4 and explicit Tasks-only authorization; add 21 Specification 002 Tasks with dependency DAG, FR/SEC/AC ownership, H1–H11/ADR/Plan traceability and future platform/security Evidence obligations. Plan: Approved; Tasks: In review; Implementation: Not authorized. Preserve prior decisions and validation history.

- specification: record PR #4 Git Authority approval as H11; preserve H1–H10 and separate Project mutation, optional local Git commit and explicitly authorized remote sync/push, with independent authority/outcomes and no false rollback or local-state export.
- planning: remove resolved Git Authority review gates; defer concrete Git mechanisms and automation design with future Evidence. Reconcile status references and PR description to H1–H11/current scope. Ready for final human Plan review; no Tasks or Implementation.

Earlier H10 reconciliation on this date (historical):

- planning: reconcile PR #4 atomicity decision as H10 across Plan, Specification, Clarifications and ADR-0004. Preserve logical Project atomicity and pre/post-commit outcomes; leave multi-file transaction, staging, lock and syscall choices to implementation with Linux/macOS Evidence. Project model and init/update contract approved; remaining Git Authority review explicit. No Tasks or Implementation.

Earlier reconciliation on this date (historical):

- specification: reconcile PR #4's latest human decision as H9: partial update intent is allowed; domain materializes and validates complete proposed Project state before application-authorized complete persistence. Preserve H1–H8 and original approval history.
- planning: align FR-018, AC-15, use cases and planned verification; forbid direct partial manifest mutation or validation bypass; identify remaining init/update and Git/authority review concerns. Plan ready for new human review; no Tasks or Implementation.
- architecture: refine Accepted ADR-0004 and conceptual model with the same update contract; preserve confirmed identity, portability, minimal init and separate Git authority boundaries. Update H9 index references and correct stale architectural/context references to ADR-0003's existing 2026-09-10 acceptance, without changing that ADR.

## [2026-09-12]

- specification: reopen/reconcile Specification 002 and Clarifications after subsequent human Plan review; preserve original 2026-09-11 approval and record H1–H8 replacing/refining location, minimum and FR-012/update contracts.
- architecture: extend Accepted ADR-0004 with immutable ID versus mutable slug/name, portable working copy versus local state, incremental mutation and optional dedicated Git backing with explicit remote authority; no new ADR or sync engine.
- planning: reconcile Plan for minimal init, explicit update, safe slug move, rollback/concurrency and ID-addressed state continuity; extend future AC-13–AC-18 and security traceability. Ready for human re-review; Tasks and Implementation remain blocked.

Earlier review in this same date (retained historical entries):

- architecture: record Accepted ADR-0004 — Portable Project Manifest from human Plan review; distinguish the durable versioned contract, current axiom.yaml representation and initial schema details.
- planning: reconcile Specification 002 Plan with distinct absent/unconfigured/empty intent and internal installation.json formatVersion; extend future round-trip, no-op and local-version failure coverage.
- docs: reconcile ADR index, conceptual boundary and lifecycle references; preserve Specification/clarification history and ADR-0001–0003. Plan awaits human re-review and final approval; no Tasks or Implementation.

## [2026-09-11]

- planning: add Specification 002 Plan for human review after PR #3 merged, covering slice architecture, strict manifest, local state, safe persistence, security and AC-to-test Evidence; Tasks and Implementation remain gated by explicit Plan approval.

- specification: approve Specification 002 — Lingo Project Initialization after final human review on 2026-09-11; Q1–Q6 resolved and planning authorized after PR #3 merges, in a new change, with implementation still gated by an approved Plan.

- specification: reconcile Specification 002 with human-approved Q1–Q6 decisions; simplify required configuration, specify UUID v4 and strict axiom.yaml, native Linux/macOS local state, presence-only Runtime observations and conservative Repository matching.
- security: distinguish deterministic secret rejection from optional best-effort scanning; express write-target protection as a behavioral invariant covering traversal, symlink redirection and TOCTOU.
- docs: align journeys, FR/SEC/AC coverage, clarifications, index, roadmap and README; record final Approved status and resolved clarifications, without creating Plan, Tasks or implementation.

## [2026-09-10]

- specification: propose Lingo Project Initialization with portable/local boundaries, deterministic creation and installation behavior, security criteria, acceptance evidence, and open clarifications; stop before planning or implementation.
- docs: link Specification 002 from the index, roadmap and README; reconcile stale README references with Accepted ADR-0003.

- architecture: accept ADR-0002 with an independent Axiom SDD harness, domain, and lifecycle informed by Spec-Kit as a strategic upstream reference.
- architecture: accept ADR-0003 with Lingo as Axiom's local executable control plane while preserving Axiom as product, domain, policies, and contracts; implementation remains gated by an approved Specification.
- architecture: refine Project, Execution, Agent, Provider, Integration, Capability, Runtime, Transport, Agent Profile, Model Profile, Agent Planner, Orchestrator, and Business Context boundaries.
- security: separate portable Project configuration from local state and require credential references instead of versioned secrets.
- docs: record no runtime, architectural, behavioral, or file-format compatibility commitment and preserve deliberate divergence.
- docs: add control-plane direction and roadmap for Project configuration, runtime/model portability, capability negotiation, multi-agent orchestration, thin runtime skills, and a future Project Wizard.
- research: preserve Scenario 001 and Scenario 002 as historical evidence while recording the later human decision.
- planning: document an approximately weekly Spec-Kit Upstream Watch as future work without selecting or implementing its mechanism.

## [2026-08-12]

- research: correct Scenario 002 methodology by separating unexpected findings from manually validated false positives and reporting seeded recall.
- test: require exact stable-reference matches, validate unexpected-finding classifications, and preserve original model execution evidence.
- architecture: keep ADR-0002 Proposed with C as the evidence-supported leading hypothesis; no Scenario 003 or implementation authorized.

## [2026-08-11]

- research: run the controlled Axiom and GitHub Spec-Kit `v0.16.2` comparison and preserve temporary reproducibility evidence.
- architecture: propose ADR-0002 recommending conceptual compatibility without a mandatory Spec-Kit dependency.
- docs: publish durable findings, overlap analysis, strategy ranking, measured metrics, and reconsideration conditions.
- research: add Scenario 002 comparing current Axiom, experimental Axiom-native analyze/converge, and a confined Spec-Kit adapter with accuracy, failure, upgrade, and multi-repository evidence.
- architecture: keep ADR-0002 Proposed and retain C as the leading hypothesis after capability-level B vs C evidence.

## [2026-08-08]

- product: add the normative Axiom Constitution and classified product foundation.
- architecture: define the initial conceptual model, provider boundaries, and accept Project as distinct from Repository in ADR-0001.
- specification: propose the Codex agent harness generation vertical slice.
- research: compare four possible relationships with GitHub Spec-Kit without adopting it.
- dogfooding: simulate a Go pull-request review agent and prioritize P0, P1, P2, and Research gaps.

## [2026-08-07]

- feature: bootstrap the public Axiom repository with its Codex agent harness.
- feature: add product, architecture, decision, research, security, and development documentation.
- security: add repository ignore rules, sensitive-file validation, and public-repository policies.
- test: cover local sensitive-file validation, including staged index content.
