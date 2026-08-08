# Axiom Constitution

## Status

- Version: 1.0.0
- Ratified: 2026-08-08
- Last amended: 2026-08-08
- Authority: normative project governance

`MUST`, `MUST NOT`, `SHOULD` and `MAY` are normative terms. This constitution governs both Axiom development and harnesses produced by Axiom. More specific specifications and decisions MUST comply with it.

## I. Intent before implementation

- Meaningful work MUST start from an explicit problem, outcome, constraints, non-goals and acceptance evidence.
- Specifications MUST describe desired behavior before implementation choices.
- Plans and code MUST NOT silently redefine approved intent.
- Hypotheses, assumptions, requirements, principles and decisions MUST remain distinguishable.

## II. Spec-Driven Development

- Work MUST follow the smallest applicable form of `intake -> specify -> clarify -> plan -> tasks -> implement -> review -> reconcile`.
- Each phase MUST consume durable outputs from prior phases when they exist.
- A phase MAY be lightweight, but it MUST NOT be skipped when doing so would hide material ambiguity or risk.
- Human approval MUST gate choices with material product, architecture, security, data or release consequences.

## III. Durable knowledge and traceability

- Documentation is product state and MUST evolve with the behavior or decision it describes.
- Important knowledge MUST survive chat history in versioned or otherwise durable artifacts.
- Work MUST be traceable, where applicable, through `intent -> specification -> decision -> implementation -> validation -> release`.
- Every completed activity MUST leave enough context, evidence and unresolved questions for a future session to continue without reconstructing intent from chat.

## IV. Explicit decisions and honest evidence

- Important decisions MUST state evidence, alternatives, trade-offs, status and conditions for reconsideration.
- Architectural decisions with durable, cross-cutting or expensive-to-reverse impact MUST use an ADR.
- Hypotheses MUST NOT become decisions without explicit evaluation and accountable human acceptance.
- Humans and agents MUST NOT invent repository state, provider state, validation results or other evidence.

## V. Determinism before model judgment

- Invariants, schemas, relationships and validations SHOULD be deterministic and inspectable whenever practical.
- Model judgment MAY interpret intent or generate artifacts, but MUST NOT be the sole representation of authoritative state.
- Validation claims MUST name reproducible evidence; unexecuted checks MUST be reported as unverified.

## VI. Safety, observability and accountability by design

- Least privilege, explicit approval boundaries, safe defaults and preservation of existing data are mandatory.
- Security and observability MUST be considered while specifying and planning, not appended after implementation.
- Executions with material effect MUST identify the responsible human or agent, relevant inputs, outcome, evidence and approval or waiver.
- A human remains accountable for material product, architecture, security and release decisions.

## VII. Small coherent changes

- Work MUST prefer the smallest coherent change that satisfies validated intent.
- Speculative abstractions, providers, infrastructure and dependencies MUST NOT be introduced without a current requirement and evidence.
- External products SHOULD be isolated behind capability boundaries only when a real boundary or plausible replacement need exists.

## VIII. Project and repository boundaries

- `Project != Repository`.
- A Project is the logical boundary that connects intent, work, decisions, executions, evidence and releases.
- A Project MAY span multiple independent repositories. Repository proximity, common provider and Git submodules MUST NOT be assumed.
- Workspace, persistence, provider contracts and synchronization models remain open until explicitly decided.

## Governance

- Amendments require an explicit human decision, rationale, impact analysis and reconciliation of affected specifications, templates, policies and ADRs.
- Versioning follows semantic intent: MAJOR changes governance incompatibly, MINOR adds or materially expands a principle, PATCH clarifies without changing obligation.
- Every specification and plan MUST include a proportionate constitution check.
- Reviews MUST report constitutional conflicts as blocking unless an explicit, documented waiver exists.
- The repository changelog MUST record amendments.
