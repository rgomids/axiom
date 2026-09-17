# Documentation Governance v1

## Purpose

Define how Axiom documentation is classified, maintained, reviewed, and reconciled. Apply this governance before editing documentation; review each change against its canonical source, approval history, affected links, and verifiable evidence.

## Core principles

- Maintain one canonical source for each kind of information.
- Keep stable documentation separate from operational dashboards.
- Make secondary sources summarize and link to the canonical source.
- Never silently rewrite approved decisions; record explicit amendments or supersession and preserve approval history.
- Preserve history without presenting it as current state.
- Make documentation changes proportional to their impact.
- Do not manually duplicate volatile content.

## Source-of-truth matrix

| Information | Canonical source |
|---|---|
| Product vision, problem, audience, discovery, and user flows | Notion; entry point in [Product](product/README.md#documentation-authority) |
| Open product hypotheses and questions | Notion |
| Feature behavior and contracts | GitHub — [Specifications](specifications/README.md) |
| Implemented or intended architecture | GitHub — [Architecture](architecture/README.md), using C4 |
| Durable technical decisions | GitHub — [ADRs](decisions/README.md) |
| Technical guides, references, and explanations | GitHub — versioned Docs-as-Code |
| Operational state, backlog, and work in progress | GitHub Issues/Projects, when adopted |
| Implementation evidence | Evidence, tests, commits, and PRs |
| Delivery history | Git, PRs, Releases, and [Changelog](../CHANGELOG.md), as available |
| Consolidated Specification status | [docs/specifications/README.md](specifications/README.md) |

Notion does not mirror GitHub. It links to canonical technical artifacts.

Technical documents may link to discovery in Notion but must not copy volatile discovery content. Classify external content and verify publication authority before importing it, following the [security policy](../.agents/policies/security.md). This matrix assigns documentation ownership; it does not imply provider integrations, adoption of Issues/Projects, or an existing release process. It does not override the constitution, approved contracts, or executable evidence under the [repository source hierarchy](../AGENTS.md#source-hierarchy).

## Documentation categories

| Category | Purpose |
|---|---|
| Product | Summarize stable product context and link to canonical Notion discovery and approved technical contracts. |
| Architecture | Describe system structure, boundaries, and relationships; distinguish implemented behavior from intended architecture. Use C4 as the preferred visualization standard. |
| ADRs | Preserve durable decisions, context, alternatives, consequences, and trade-offs. |
| Specifications | Define feature behavior, contracts, constraints, and acceptance criteria. |
| Guides | Teach a learning sequence (Tutorial) or explain how to complete a concrete task (How-to). |
| Reference | Provide precise facts about commands, configuration, interfaces, and formats. |
| Explanation | Explain rationale, concepts, and relationships. |
| Research | Record investigations, hypotheses, sources, and findings without implying adoption. |
| Security | Define trust boundaries, security rules, reporting, and verification practices. |
| Development | Describe local setup, contribution, testing, and build workflows. |
| Evidence | Preserve reproducible observations supporting implementation and acceptance claims, with scope and limitations. |
| Agent context and policies | Summarize canonical context and define agent operating rules without replacing product or technical authority. |

Use Diátaxis only to classify Tutorial, How-to, Reference, and Explanation content. This classification requires no directory reorganization. C4 describes architecture; ADRs record decisions and trade-offs; Specifications remain the source of behavioral contracts.

## Lifecycle and status rules

- Roadmaps express direction and dependencies, not execution tracking.
- Do not manually track Tasks and PRs across multiple documents. Link to their authoritative records; use Issues/Projects for operational tracking when adopted. Until adoption, retain existing Task and PR records without inventing another tracker.
- A technical merge does not establish human acceptance. Acceptance does not automatically authorize the next Task.
- Current consolidated Specification status belongs in [the Specifications index](specifications/README.md). Detailed approval and acceptance records remain with the relevant artifacts; secondary documents link to them rather than copy current status.
- ADRs preserve decision context and history; they are not operational dashboards.
- Historical snapshots must carry a date and an explicit statement that they do not represent current state.

## Update triggers

| Change | Expected documentation |
|---|---|
| Rule or behavior | Specification and Evidence |
| Durable architectural decision | ADR and architecture |
| New boundary or integration | Architecture and possibly an ADR |
| New command or configuration | Reference/guide |
| Product change | Notion and, when a contract is affected, Specification |
| Execution state | Issue/Project when adopted; otherwise the existing Task/PR record |
| Completed delivery | Evidence, PR, and Changelog when relevant |

Update only affected sources, then reconcile their summaries and links. Review correctness, authority, and scope before delivery; record verification and any unresolved limitations in the PR.

## Anti-drift rules

- Do not copy Task/PR status into Product Foundation or Roadmap.
- Do not replicate Specification contracts in vision pages; summarize and link.
- Do not treat agent context as a higher authority than canonical artifacts.
- Do not record a hypothesis as an accepted decision.
- Do not add a status, badge, or claim that cannot be verified. Prefer source-backed dynamic displays for volatile facts instead of manually maintained values.
- When sources conflict, reconcile the secondary source and preserve the canonical source. If executable evidence conflicts with an approved contract, expose the discrepancy for explicit resolution rather than silently changing the contract or approval history.

## Localization

Translated documentation may be introduced in the future. This change creates no translations. Until a specific localization policy exists, the current [README](../README.md) remains canonical. Future translations must declare their source and how drift is prevented. Translation tools and processes remain undecided.
