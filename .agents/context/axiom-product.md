# Axiom Product Context

## Product thesis

Axiom aims to be a control plane for AI-assisted software development.

Canonical product classification is in `docs/product/foundation.md`. Normative governance is in `docs/product/constitution.md`; do not duplicate those documents here.

The problem is not merely code generation. The product should help preserve coherence across:

- product intent;
- specifications;
- architecture;
- ADRs;
- work items;
- repositories;
- implementation;
- validation;
- documentation;
- release and operational evidence.

## Primary user scenarios

A user may be:

- an individual software engineer;
- a developer working within a team;
- a person building a product from zero.

A project may span multiple independent repositories on the same machine.

The agent must understand that:

```text
Project != Repository
```

A project can own or reference multiple repositories.

This boundary is accepted by `docs/decisions/0001-project-is-not-repository.md`. Workspace persistence and ownership remain open.

## Desired lifecycle

```text
idea/discovery
→ product definition
→ specification
→ architecture
→ planning
→ tasks
→ implementation
→ validation
→ documentation
→ release
→ observation
→ learning
→ new change
```

## Product direction

Axiom should eventually provide a CLI.

Planned CLI implementation language: **Go**.

Do not infer that the CLI must be built immediately. The current harness exists to validate workflows before automating them.

## Important product characteristics

- local-first is a strong candidate, not yet an immutable decision;
- external systems should be adapters/providers rather than core dependencies;
- Codex is the first operational agent runtime for this repository;
- durable context should live in artifacts rather than only conversation history;
- structured events and decisions are preferred over raw chat logs;
- token/context efficiency is a first-class concern;
- the product should still expose useful deterministic behavior when an LLM is unavailable.

## Current strategic questions

Still open unless an ADR/spec says otherwise:

- exact MVP boundary;
- local-first vs hybrid synchronization model;
- authoritative sources for technical and operational state;
- internal work-item model;
- execution vs preparation/orchestration boundaries;
- autonomy/approval model;
- repository discovery and workspace model;
- remote service requirements;
- first persistence model;
- packaging and distribution strategy.

## First vertical slice

`docs/specifications/001-codex-agent-harness-generation/spec.md` proposes the first flow:

```text
intent -> intake -> normalized blueprint -> artifact plan -> Codex renderer -> validation -> package
```

It is proposed, not implementation approval. Dogfood evidence and prioritized gaps live in `docs/product/dogfooding/001-go-pr-review-agent.md`.
