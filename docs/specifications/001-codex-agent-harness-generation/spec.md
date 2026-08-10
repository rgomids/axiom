# Specification 001 — Codex Agent Harness Generation

## Status

Proposed for human review. This specification authorizes no CLI, renderer implementation, provider integration or dependency adoption.

## Intent

### Problem

Creating a repository agent directly from an ambiguous prompt loses assumptions, duplicates rules and makes package quality depend on model judgment.

### Desired outcome

Given a need for a software-development agent, Axiom can produce a validated Codex harness package whose intent, inputs, assumptions, files and validation evidence are traceable.

### Primary actor

A human responsible for the target project and for material scope, permission and risk decisions.

### Non-goals

- Implement the future Axiom CLI or Lingo.
- Implement a renderer, provider adapter, database or service.
- Support native targets other than Codex.
- Install generated output into a target repository automatically.
- Grant permissions, credentials, hooks or network access.

## Flow

```text
User intent
-> Intake
-> Normalized Agent Blueprint
-> Artifact Plan
-> Codex Renderer
-> Validation
-> Package
```

Each transition must preserve source references, assumptions, decisions, open questions and validation status.

## Scenarios

### S1 — Complete request

Given sufficient project context and constraints, the flow produces a blueprint, minimal artifact plan, Codex harness, deterministic validation evidence and root-extractable package.

### S2 — Material ambiguity

Given an unknown that changes behavior, permissions, risk or package scope, the flow stops before rendering and records the required human decision.

### S3 — Safe reversible unknown

Given a low-risk reversible unknown, the flow may continue with a clearly labeled conservative assumption that remains traceable in blueprint and package manifest.

### S4 — Validation failure

Given an unsafe, inconsistent or incomplete rendered package, validation fails, packaging does not report success, and findings identify evidence and affected artifacts.

## Functional requirements

- **FR-001 Intake:** capture objective, target project, repository mode, relevant stack/architecture, workflows, constraints, providers/capabilities, validation, risk, approval points, package scope and sources.
- **FR-002 Classification:** distinguish provided facts, requirements, assumptions and open questions. It must not infer provider availability, commands, credentials or approval.
- **FR-003 Clarification gate:** ask only questions whose answers materially change behavior, security, architecture, validation or package scope; block rendering on unresolved high-impact questions.
- **FR-004 Blueprint:** normalize accepted intake into a versioned, vendor-neutral blueprint. Target-specific paths must not leak into vendor-neutral sections.
- **FR-005 Artifact plan:** list every intended file with path, responsibility, necessity and source blueprint concepts. Unjustified files must be removed.
- **FR-006 Codex rendering:** render Codex-specific structures only at this phase. `AGENTS.md` remains a concise router; reusable procedures belong in focused skills.
- **FR-007 Conservative defaults:** rendering must not silently add secrets, credentials, permission expansion, destructive actions, external MCP dependencies, hooks or unrelated vendor artifacts.
- **FR-008 Deterministic validation:** validate structure, target isolation, file safety, symlinks, sensitive content, placeholder absence and required skill files without relying solely on a model.
- **FR-009 Semantic validation:** assess blueprint coverage, contradictions, duplicate responsibilities, real versus unresolved commands, visible assumptions and Definition of Done coverage. Findings must identify evidence.
- **FR-010 Package:** produce a package intended for extraction at repository root, containing only planned and validated files plus a manifest.
- **FR-011 Traceability:** the manifest must identify blueprint version, planned files, validation checks/status, assumptions, risks and unresolved decisions. Generated claims must reference their source or classification.
- **FR-012 Failure honesty:** unexecuted checks and unresolved package status must be labeled unverified or blocked, never successful.

## Invariants

- A package cannot be `valid` when a required deterministic check failed.
- A package cannot contain a file absent from the artifact plan, except the generated manifest when the plan explicitly requires manifest generation.
- Provider-neutral blueprint fields cannot require Codex-specific file paths.
- No secret or private provider content may be copied into output.
- Human approval is required before relaxing security or granting material operational authority.
- Chat history is not an authoritative source artifact.

## Acceptance criteria

1. A representative agent need can be traced through every flow stage without direct prompt-to-render shortcuts.
2. Material unknowns are visible and stop rendering when required.
3. The blueprint can be understood without Codex file-layout knowledge.
4. Every rendered file maps to the artifact plan and blueprint.
5. Existing deterministic package and sensitive-file checks pass on valid output and reject their covered unsafe cases.
6. Semantic gaps not covered by scripts remain explicitly reported, not implied as validated.
7. The package tree is minimal, Codex-only and extractable at repository root.
8. A later session can identify intent, assumptions, validation evidence and unresolved decisions without chat history.

## Constitution check

- Intent is recorded before rendering.
- Hypotheses and assumptions remain labeled.
- Minimal output and conservative permissions are required.
- Deterministic checks gate claims where available.
- Human approval protects material choices.
- Traceability persists across every stage.

## Open questions before implementation planning

- Canonical blueprint and manifest schemas, identifiers and versioning policy.
- Exact threshold separating blocking question from safe assumption.
- Semantic validation rules that can become deterministic.
- Whether packaging itself belongs to the renderer or a separate packager.
- Evidence storage and integrity requirements for a local-only first implementation.
- Whether generated packages contain the blueprint, only a manifest, or references to both.
