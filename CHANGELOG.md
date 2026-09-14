# Changelog

## [2026-09-14]

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
