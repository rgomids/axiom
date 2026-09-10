# Dogfooding 001 — Go Pull Request Review Agent

## Status

Completed process simulation on 2026-08-08 using the current `axiom-agent-factory` skill, blueprint template, Codex renderer guidance and repository validators. No agent package was generated or added to the repository. Renderer and packaging results are therefore assessed, not claimed as executed.

This is point-in-time evidence. The Spec-Kit research item recorded below was
later resolved by [ADR-0002](../../decisions/0002-axiom-speckit-relationship.md):
Axiom now owns an independent harness direction and treats Spec-Kit as a
strategic upstream reference.

## Input

> Preciso de um agente responsável por revisar Pull Requests de aplicações Go, considerando arquitetura, segurança, testes e qualidade.

## 1. Intake

### Classified input

- **Requirement:** review pull requests for Go applications.
- **Requirement:** cover architecture, security, tests and general engineering quality.
- **Fact:** target runtime requested by the current slice is Codex.
- **Assumption:** review is read-only and produces findings; it does not edit, approve, merge or publish comments.
- **Assumption:** package scope is minimal and repository-specific commands are supplied by the target project.
- **Open question:** single repository or multi-repository Project?
- **Open question:** which architecture decisions, threat model and quality gates are authoritative?
- **Open question:** should output be local, a GitHub review, or both?
- **Open question:** which actions require human approval?

### Material questions before a production package

1. Which repository or Project context is in scope?
2. Is the agent advisory-only, or may it publish reviews and request changes?
3. Which commands are authoritative for unit, integration, functional, race, lint and security checks?
4. Which architecture documents and risk policies govern findings?
5. What severity blocks approval, and who may waive it?

Safe simulation continued with advisory-only, no-provider and no-write assumptions. Real rendering should block until questions 1–4 are answered because otherwise files and authority would materially differ.

## 2. Normalized Agent Blueprint

```yaml
schema_version: "0.1"
agent:
  name: "go-pr-reviewer"
  objective: "Review Go pull-request changes for architecture, security, tests, and engineering quality."
  target: "codex"
project:
  name: "unbound-go-project"
  description: "Target Project must be selected before rendering."
  repository_mode: "unknown"
  stack: ["Go"]
  architecture: "Use target Project decisions; unresolved until binding."
  constraints:
    - "Advisory and read-only"
    - "Never invent checks or results"
    - "Report findings by severity with evidence"
sources:
  product: []
  architecture: []
  decisions: []
  task_tracking: []
  external_docs: []
workflows:
  - "inspect change and governing context"
  - "review specification, architecture, verification, and security lanes"
  - "run approved deterministic checks when available"
  - "deduplicate and report evidence-backed findings"
skills:
  - "go-pull-request-review"
specialist_roles: []
providers:
  source_control:
    required_capabilities: ["read pull-request diff and metadata"]
  tracking:
    required_capabilities: []
  documentation:
    required_capabilities: []
validation:
  commands: []
  definition_of_done:
    - "Every finding has severity, evidence, location, impact, and recommendation"
    - "Unexecuted checks are named"
    - "No mutation or provider publication occurred"
risk:
  level: "medium"
  human_approval_points:
    - "Publish provider review or request changes"
    - "Waive blocking or critical finding"
  prohibited_actions:
    - "Merge or approve pull requests"
    - "Modify target code"
    - "Expose secrets"
package:
  scope: "minimal"
  output_name: "go-pr-reviewer-codex-agent.zip"
assumptions:
  - "Simulation is advisory-only"
open_questions:
  - "Target Project and repositories"
  - "Authoritative commands and architecture"
  - "Provider publication policy"
```

The blueprint is intentionally not render-ready: unresolved target sources and commands materially affect correct output.

## 3. Artifact Plan

| Path | Disposition | Responsibility | Why needed | Blueprint source |
|---|---|---|---|---|
| `AGENTS.md` | Planned after intake resolves | Mission, durable constraints, source hierarchy, skill routing and Definition of Done. | Codex entrypoint. | objective, constraints, sources, risk, validation |
| `.agents/skills/go-pull-request-review/SKILL.md` | Planned after intake resolves | Review procedure and evidence-backed finding schema. | Multi-step contextual workflow. | workflows, skill, Definition of Done |
| `.agents/policies/review.md` | Conditional, not yet planned | Project-approved severities and publication/waiver rules. | Add only if target policy cannot be referenced without duplication. | risk, human approval points |
| `agent-manifest.yaml` | Planned after intake resolves | Planned files, assumptions, risks and validation status. | Package traceability. | package, assumptions, open questions, validation |

No specialist subagent, provider configuration, hook or duplicated Go style guide is justified by current intake.

## 4. Codex Renderer assessment

Current guidance correctly separates vendor-neutral blueprint from Codex paths and keeps `AGENTS.md` as router. It can guide a model to draft the package. It cannot currently render the package deterministically, verify blueprint coverage or bind Project-specific sources and commands.

## 5. Validation assessment

Existing scripts can deterministically reject symlinks, sensitive filenames/content, Claude artifacts, vague placeholders and missing `SKILL.md`, and their fixtures pass in this repository.

They do not validate:

- blueprint or manifest schema;
- artifact-plan coverage;
- source provenance and classification;
- command existence;
- duplicated or contradictory rules;
- permission claims;
- ZIP contents and root extraction layout;
- semantic completeness of review lanes.

## 6. Packaging assessment

Packaging is documented but manual. No package was created. Filename, manifest contents, file allowlist, reproducibility and archive inspection are conventions rather than executable gates.

## Findings

### What worked

- Intake categories and conservative defaults exposed missing authority.
- Blueprint template captured purpose, workflows, providers, validation and risk without embedding Codex paths.
- Artifact planning removed unjustified specialists and integrations.
- Renderer guidance promoted progressive disclosure.
- Existing security validator covered meaningful package hazards.

### What failed

- No executable renderer or packager exists.
- No deterministic link proves `intake -> blueprint -> artifact plan -> files -> manifest` coverage.
- Current validation can report structural safety while missing semantic incompleteness.

### What is ambiguous

- Required intake fields versus optional fields.
- Blocking-question threshold.
- Project binding and multi-repository source selection.
- Execution/evidence identity and retention.
- Whether provider publishing belongs in generation or later operation.

### What should become deterministic

- Blueprint and manifest schema validation.
- Required-field and state-transition checks.
- Artifact allowlist and plan-to-package coverage.
- Package archive inspection and reproducible naming.
- Command resolution against selected repositories.
- Traceability IDs and validation result recording.

### What still requires human decision

- Agent authority and publication permissions.
- Target Project sources and repositories.
- Risk/severity gates and waiver owners.
- Provider and credential boundaries.
- Acceptance of the package for installation.

## Gaps

| Priority | Gap | Impact |
|---|---|---|
| P0 | No executable renderer/packager or deterministic stage handoff. | A repeatable end-to-end product flow cannot yet run. |
| P0 | Material intake unknowns lack a formal render-blocking gate. | Model may render unsafe or misleading output. |
| P1 | Blueprint and manifest have templates but no schemas, IDs or compatibility policy. | Invalid or untraceable artifacts can pass structural validation. |
| P1 | Semantic and plan-to-package validation remains model-dependent. | Coverage, contradictions and authority errors may survive. |
| P1 | Execution and Evidence records are undefined. | Results and blockers may be lost between sessions. |
| P1 | Project binding and repository command discovery are unspecified. | Generated validation cannot be trusted. |
| P2 | Multi-runtime renderers and provider publication. | Useful after Codex local generation is proven. |
| P2 | Reproducible package signing/provenance. | Valuable after package format stabilizes. |
| Research | Spec-Kit dependency/orchestration/compatibility choice. | Needs comparative experiment. |
| Research | RTK, Caveman, Ponytail, Graphify and agent-skills. | No adoption evidence yet. |

## First implementation candidate

After this specification is approved, plan only the deterministic contract spine: blueprint schema, artifact-plan/manifest relationship, blocking intake states and validation fixtures. Renderer behavior should follow that contract, not precede it.
