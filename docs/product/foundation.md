# Product Foundation

Short versioned summary of the minimum product context repository agents need.
The full product and system narrative (vision, problem, audience, discovery,
flows, open questions) lives in [Notion — Axiom](https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979), per
[Documentation Governance](../documentation.md). This summary does not convert
discovery material into approval and must not copy Task/PR status.

Normative principles: [Axiom Constitution](constitution.md). Domain language:
[conceptual model](../architecture/conceptual-model.md). Durable decisions:
[ADRs](../decisions/README.md). Contracts and current lifecycle:
[Specification index](../specifications/README.md).

## Classification vocabulary

Fact (observed or reproduced), Requirement (must be satisfied), Principle
(durable rule), Hypothesis (testable, not approved), Decision (explicitly
accepted), Open question (needs evidence or human choice). Assumptions are
temporary, visible and reversible; never report them as facts.

## What Axiom is

Axiom is the product, domain, policies and contracts that preserve coherence
across product intent, specifications, architecture, work, repositories,
execution, validation, documentation, releases and evidence when humans and AI
agents collaborate. Lingo is the accepted local executable control plane
([ADR-0003](../decisions/0003-lingo-as-axiom-local-control-plane.md)); the names
are not interchangeable.

## Requirements agents must preserve

- A Project may relate multiple repositories without submodules
  ([ADR-0001](../decisions/0001-project-is-not-repository.md)).
- Durable state must not depend exclusively on chat history; traceability from
  intent to delivery evidence is preserved.
- Useful deterministic behavior remains possible without a model.
- No source-control, tracking, documentation or runtime provider is mandatory at
  the domain level.
- Human approval boundaries remain explicit.
- Portable Project configuration excludes secrets and machine-specific state
  ([ADR-0004](../decisions/0004-portable-project-manifest.md)).
- Roles are not coupled to concrete models; `Runtime != Model`,
  `Role != Model`, `Execution != Agent`, `Provider != Transport`,
  `Integration != MCP`.

## Principles

Intent precedes implementation. Documentation is part of product state.
Deterministic evidence outranks unsupported model judgment. Security and
observability are design concerns. Prefer the smallest coherent change.
Hypotheses are not decisions.

## Authority

Approved Specifications and ADRs are product/architecture authority; repository
state and executable checks are evidence; Notion is the product narrative and
discovery home and never automatic approval or publication authority. Neither
roadmap direction, code availability nor technical merge substitutes for human
acceptance.
