# Product Foundation

## Purpose

This document classifies the current product foundation. It does not convert discovery material into approval. Normative principles live in the [Axiom Constitution](constitution.md), domain language in the [conceptual model](../architecture/conceptual-model.md), and durable architecture choices in [ADRs](../decisions/README.md).

## Classification vocabulary

| Class | Meaning |
|---|---|
| Fact | Directly observable current state or reproducible evidence. |
| Requirement | Behavior or constraint the product must satisfy. |
| Principle | Durable rule governing choices and work. |
| Hypothesis | Testable possibility not yet approved. |
| Decision | Explicitly accepted choice with accountable status. |
| Open question | Material unknown requiring evidence or human choice. |

Assumptions are temporary inputs used to continue safely. They must be visible, reversible and must not be reported as facts.

## What Axiom is and why it exists

**Requirement:** Axiom is a development control plane intended to preserve coherence across product intent, specifications, architecture, work, repositories, execution, validation, documentation, releases and evidence when humans and AI agents collaborate.

**Problem evidence from discovery:** agent-assisted development fragments context and durable state, especially when work spans repositories and external systems.

**Current product outcome:** prove a small, useful, traceable workflow through the repository harness before automating a broader platform.

## Current classification

### Facts

- The repository contains a Codex-first harness, policies, skills, templates and deterministic shell validation.
- No Axiom CLI, application runtime, `go.mod`, database, infrastructure or provider adapter exists.
- The current agent factory describes `intake -> normalized blueprint -> artifact plan -> Codex renderer -> validation -> package`.
- Notion contains discovery material and explicitly marks strategic choices as open.

### Requirements

- A Project must be able to relate multiple repositories without requiring submodules.
- Durable state must not depend exclusively on chat history.
- The product must preserve traceability between intent and delivery evidence.
- Useful deterministic behavior must remain possible without an available model.
- No source-control, tracking or documentation provider may be mandatory at the domain level.
- Relevant human approval boundaries must remain explicit.

### Principles

- Intent precedes implementation.
- Documentation is part of product state.
- Deterministic evidence outranks unsupported model judgment.
- Security and observability are design concerns.
- Prefer the smallest coherent change; avoid speculative abstraction.
- Hypotheses are not decisions.

### Decisions

- The project now uses the [Axiom Constitution](constitution.md) as normative governance.
- `Project != Repository` is accepted in [ADR-0001](../decisions/0001-project-is-not-repository.md).
- Codex is the only currently supported native renderer in the existing harness. This is a bounded current-state decision, not a commitment that Axiom will remain single-runtime.
- The first specified dogfood slice is [Codex agent harness generation](../specifications/001-codex-agent-harness-generation/spec.md).

### Hypotheses

- The product may start as a personal local-first CLI written in Go.
- Axiom may persist its own Workspace, Work Item, Execution and Evidence identities.
- Axiom may orchestrate Spec-Kit, depend on it, implement compatible concepts, or use it only as reference.
- RTK, Caveman, Ponytail, Graphify and `tech-leads-club/agent-skills` may offer useful techniques. None is adopted.

### Open questions

- Is Workspace a persisted aggregate, a local operational view, or both?
- What state is authoritative in Axiom versus repositories and external providers?
- Does a Work Item have Axiom identity, provider identity, or a hybrid mapping?
- Which execution events and evidence must be retained, for how long, and with what privacy controls?
- Is Axiom initially a preparer, orchestrator, executor, or deterministic workflow controller?
- What is the autonomy and approval model by action and risk?
- Which Spec-Kit relationship, if any, should be adopted?

## Source register and authority

| Source | Role | Classification rule |
|---|---|---|
| Repository state and executable checks | Current evidence | Facts only when observed or reproduced. |
| [Constitution](constitution.md) | Governance | Normative principles. |
| Approved specifications and ADRs | Product/architecture authority | Requirements and accepted decisions. |
| `.agents/context/` and current docs | Durable working context | Requirements, principles or hypotheses according to explicit labels. |
| [Notion discovery](https://app.notion.com/p/3b4e01f22626810791b4f9d016ab5979) | Discovery source | Mixed input; never automatic approval or publication authority. |
| Historical harness proposal | Historical evidence | Requirements source only after current validation. |
