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
```

Current classification:

- accepted distinction: Project is not Repository; aggregate and ownership boundaries remain open;
- candidate operational context: Workspace;
- accepted concepts with open representations: Specification, Plan, Decision, Evidence;
- candidate lifecycle concepts: Work Item, Execution, Release, Integration;
- supporting concepts: Artifact, Agent, Provider.

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

Use `docs/architecture/provider-boundaries.md` for the abstraction threshold. Do not create a universal provider interface before a specified workflow requires one.

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
