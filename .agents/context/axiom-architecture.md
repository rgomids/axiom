# Axiom Architecture Context

## Current conceptual model

Canonical definitions, boundaries and open questions live in `docs/architecture/conceptual-model.md`. This file is a routing summary.

Axiom should be decomposed around stable domain concepts rather than vendor products.

Candidate concepts:

```text
Workspace
Project
Repository
Work Item
Specification
Plan
Task
Decision
Execution
Finding
Evidence
Release
Agent Blueprint
Renderer
Provider
Capability
Policy
Workflow
Runtime
Transport
Agent Profile
Model Profile
Agent Planner
Orchestrator
Business Context
```

Current classification:

- accepted distinction: Project is not Repository; aggregate and ownership boundaries remain open;
- candidate operational context: Workspace;
- accepted concepts with open representations: Specification, Plan, Decision, Evidence;
- candidate lifecycle concepts: Work Item, Execution, Release, Integration and Business Context;
- supporting concepts: Artifact, Agent, Provider.
- boundary concepts: Provider, Capability, Runtime and Transport;
- configuration concepts: Integration, Agent Profile and Model Profile;
- candidate executable capabilities: Agent Planner and Orchestrator.

## Architectural direction

Prefer:

```text
Core/domain
    ↓
application workflows
    ↓
ports/capabilities
    ↓
adapters/providers/renderers
```

External products such as GitHub, GitLab, Linear, Jira, Notion, Confluence, Codex, or future runtimes must not leak unnecessarily into core domain contracts.

The accepted ADR-0003 direction separates:

```text
Axiom product/domain/policies/contracts
    ↓
Lingo local control plane
    ↓
workflows / Project state / runtime resolution
    ↓
thin runtime adapters and skills
```

This direction is documented in `docs/architecture/conceptual-model.md` and
`docs/decisions/0003-lingo-as-axiom-local-control-plane.md`, Accepted on 2026-09-10.
Acceptance does not authorize implementation; Specification and Plan approval
and explicit authorization to advance remain required.

Use `docs/architecture/provider-boundaries.md` for the abstraction threshold. Do not create a universal provider interface before a specified workflow requires one.

Preserve `Role != Model`, `Provider != Transport`, `Integration != MCP`,
`Execution != Agent`, and `Skill != workflow source of truth`. Secrets never
belong in portable Project configuration.

Accepted Specification 004 decisions add two bounded architectural contracts:

- ADR-0005: exact authorized-target confinement, supported traversal/link/
  replacement protection, process concurrency, deterministic faults, complete
  canonical state and guided recovery remain required; malicious same-UID
  arbitrary interleavings and physical power-loss/media durability are unsupported.
- ADR-0006: detail artifacts use one Axiom-owned machine-local boundary with
  stable identity, Execution-first correlation, non-domain local correlation
  before Execution, Evidence references, purpose-based retention and explicit
  reference-aware cleanup. They never enter portable Project intent.

Concrete filesystem mechanisms, artifact paths/layout, retention durations and
size limits remain future Plan/implementation work.

## Multi-repository principle

A project may reference several repositories without Git submodules.

Repository metadata should eventually be explicit enough for the system to know:

- local path;
- role;
- source-control identity when available;
- commands;
- validation capabilities;
- documentation responsibilities.

## Context construction

Do not solve context management by blindly injecting all files.

Future Axiom context construction should distinguish:

- mandatory context;
- related context;
- policies;
- relevant decisions;
- affected files/repositories;
- recent evidence;
- irrelevant context to exclude.

## Deterministic core

State, relationships, validation rules, schemas, and workflow invariants should remain inspectable and testable without relying exclusively on model judgment.

LLMs should assist with interpretation and generation, not become the only representation of system state.

## C4

When architecture diagrams are created or updated, use the C4 Model by default:

1. System Context;
2. Container;
3. Component only when it adds decision value;
4. Code-level diagrams only when justified.

Prefer diagrams that explain boundaries and interactions rather than implementation trivia.
