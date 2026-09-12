# Changelog

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
