# Axiom Agent Harness

## Why this repository starts with an agent harness

Before Axiom automates development governance, the project should exercise the workflow manually with a maintainer runtime (Codex or Claude Code).

This creates a bootstrap loop:

```text
Axiom definitions
→ maintainer agent follows them manually
→ real project work exposes gaps
→ definitions improve
→ Axiom CLI automates proven workflows
```

The harness is therefore not disposable scaffolding. It is the current manual executable product hypothesis.

The accepted future direction centralizes executable workflow behavior in Lingo under Axiom contracts, while runtime skills tend toward thin entrypoints. Existing skills do not change until a Specification defines migration, compatibility, and acceptance evidence.

## Maintainer runtimes and canonical skills

```text
AGENTS.md                         canonical maintainer policy (router)
CLAUDE.md                         "@AGENTS.md" only: bootstrap for Claude Code
.agents/skills/<skill>/SKILL.md   canonical, runtime-neutral maintainer skills
  Codex: native discovery          $<skill>
.claude/skills/<skill>            symlink -> ../../.agents/skills/<skill>
  Claude Code: native discovery    /<skill>
```

[ADR-0014](decisions/0014-canonical-maintainer-skills-runtime-discovery.md)
records this contract. `.agents/skills/` is not Codex-owned: Codex happens to
scan it natively, and Claude Code reaches the same directories through thin
symlink adapters. Each skill has one implementation, so `$axiom-release` and
`/axiom-release` load the same file. `scripts/check-claude-bootstrap.sh
--skill-adapters` enforces a closed allowlist (exact relative targets, one
adapter per canonical skill, no copies, no other `.claude/` content).

Remaining asymmetries:

- the runtimes expose different delegation, model and reasoning controls;
  [`axiom-orchestrate`](../.agents/skills/axiom-orchestrate/SKILL.md) uses
  them only when present and degrades to the main session otherwise;
- adapters are symlinks; checkouts without symlink support (for example Git
  for Windows with `core.symlinks=false`) lose Claude native discovery, while
  `AGENTS.md` routing still reaches the canonical skills.

Maintainer runtime support is not Agent Factory renderer support: the only
implemented renderer is Codex. A Claude renderer needs its own Specification
and validation.

### Cross-runtime evaluation

`scripts/test-maintainer-agent.sh` (run by `scripts/validate-repository.sh`)
checks deterministically that every canonical skill is discoverable by both
runtimes and that both entry paths resolve to the same `SKILL.md` and the same
link targets. With `--runtime codex --runtime claude` it also runs read-only
behavioral scenarios (small local change, architecture decision, review,
multi-front task, destructive/external action without authority, capability
mismatch, native discovery, canonical equivalence) and asserts observable
invariants of a structured answer and an unchanged working tree, not prose.
Canonical equivalence stamps a nonce into a throwaway copy of the canonical
release skill and requires each runtime to read it back through its own
invocation; no release step runs. A missing or failing runtime is reported as
UNVERIFIED, never as a pass; `--require-runtimes` makes that a failure.

## Boundaries

This version deliberately does **not** decide:

- final CLI command model;
- persistence engine;
- cloud synchronization;
- internal task database;
- provider implementation;
- production deployment;
- final agent schema;
- adoption of token-saving/context tools;
- Lingo implementation or migration from current skills;
- product runtime adapters, Agent Planner, orchestration, execution graphs, or model discovery (the maintainer `axiom-orchestrate` skill is a manual procedure, not this product capability).

Those should emerge from specifications and validated experiments.

## Dogfood objective

The `axiom-agent-factory` skill should be used when new project agents are needed.

Each generation exercise should capture:

- what information was missing;
- which parts were repetitive;
- which steps could be deterministic;
- which context was unnecessary;
- which validations prevented mistakes;
- which concepts belong in the future Axiom core.

Those observations are direct product input.

The first documented exercise is [Dogfooding 001 — Go Pull Request Review Agent](product/dogfooding/001-go-pr-review-agent.md). It exposed the specification for [Codex Agent Harness Generation](specifications/001-codex-agent-harness-generation/spec.md) without generating application code or claiming an executable renderer exists.
