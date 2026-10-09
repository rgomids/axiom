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
| Public product vision, problem, audience, discovery, and user flows | [GitHub Wiki](https://github.com/rgomids/axiom/wiki); technical requirements remain in approved repository artifacts |
| Public product hypotheses and questions | GitHub Wiki, explicitly classified and reconciled against approved contracts before publication |
| Feature behavior and contracts | GitHub — [Specifications](specifications/README.md) |
| Implemented or intended architecture | GitHub — [Architecture](architecture/README.md), using C4 |
| Durable technical decisions | GitHub — [ADRs](decisions/README.md) |
| Technical guides and references that must evolve with code | GitHub Repository — versioned Docs-as-Code |
| Public tutorials, user guides, concepts, and editorial explanations | GitHub Wiki; summarize and link to code-coupled canonical references |
| Product roadmap, product epics/stories, priorities, milestones and delivery dates | Linear, AXM team; see [Linear and GitHub tracking](development/linear-github-issue-tracking.md) |
| Technical Work Items, bugs, implementation tasks and engineering progress | GitHub Issues and PRs, with technical contracts in versioned repository artifacts |
| Implementation evidence | Evidence, tests, commits, and PRs |
| Delivery history | Git, PRs, Releases, and [Changelog](../CHANGELOG.md), as available |
| Consolidated Specification status | [docs/specifications/README.md](specifications/README.md) |
| Stable repository landing page | Canonical English [README.md](../README.md) and synchronized official [README.pt-BR.md](README.pt-BR.md) translation |

GitHub Wiki and Repository have distinct ownership. Wiki explanations link to technical artifacts rather than duplicate their contracts. Notion is a migration source, not the target operational source.

Classify external content and verify publication authority before importing it, following the [security policy](../.agents/policies/security.md). Historical source references may remain for provenance; no current workflow should require Notion. Migration completion requires source inventory reconciliation, validated destinations and links, and explicit publication authority. Local migration drafts are temporary review artifacts, not a third source of truth. This matrix assigns documentation ownership; it does not imply that GitHub/Linear integrations or issue synchronization have been configured. Product delivery is tracked in Linear; technical Work Item and publication authority remain in their owning GitHub and Execution artifacts. It does not override the constitution, approved contracts, or executable evidence under the [repository source hierarchy](../AGENTS.md#source-hierarchy).

## Documentation categories

| Category | Purpose |
|---|---|
| Product | Public discovery and explanations live in Wiki; repository product summaries link to Wiki and approved technical contracts. Normative constitution and Evidence remain versioned. |
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
- Do not manually track Tasks and PRs across multiple documents. Link to their authoritative records: Linear for product delivery planning and GitHub Issues/PRs for technical work. Link mapped records without duplicating volatile state or treating Linear as a canonical Execution ledger.
- A technical merge does not establish human acceptance. Acceptance does not automatically authorize the next Task.
- Current consolidated Specification status belongs in [the Specifications index](specifications/README.md). Detailed approval and acceptance records remain with the relevant artifacts; secondary documents link to them rather than copy current status.
- ADRs preserve decision context and history; they are not operational dashboards.
- ADR lifecycle statuses are `Proposed`, `Accepted`, `Superseded`, and `Rejected`. Partial supersession is not a status: the earlier ADR keeps `Accepted`, annotates the superseded fragment in place, and adds a `Partially superseded by` note to the [ADR index](decisions/README.md#index). Only full replacement uses `Superseded`. The superseding ADR links back in a `## Supersedes` section.
- [ADR-0018](decisions/0018-adr-evolution-and-supersession-governance.md) defines this convention. `scripts/check-adr-governance.py` enforces its structure; whether a change semantically contradicts an accepted ADR remains a review responsibility.
- Historical snapshots must carry a date and an explicit statement that they do not represent current state.

## Update triggers

| Change | Expected documentation |
|---|---|
| Rule or behavior | Specification and Evidence |
| Durable architectural decision | ADR and architecture |
| New boundary or integration | Architecture and possibly an ADR |
| New command or configuration | Reference/guide |
| Public product change | Wiki and, when a contract is affected, Specification |
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
- README is a stable project landing page, not an operational status dashboard.
  Keep current Task, Slice, PR, daily implementation, acceptance-date, and
  temporary follow-up state in their owning Specifications, Evidence, Issues,
  PRs, Changelog, or other canonical records; link to those records instead of
  copying their state into README.

## Localization

[README.md](../README.md) is the canonical English README.
[README.pt-BR.md](README.pt-BR.md) is its official complete Brazilian
Portuguese translation. Both files provide reciprocal language navigation.

Any material change to the canonical README must reconcile the translation in
the same work and Pull Request. `README.md` must remain English except for
unavoidable language-navigation labels. Other translations remain future work
and require an explicit source and drift-prevention rule before introduction.
