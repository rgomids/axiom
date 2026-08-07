# Axiom Agent Harness

## Why this repository starts with an agent harness

Before Axiom automates development governance, the project should exercise the workflow manually with Codex.

This creates a bootstrap loop:

```text
Axiom definitions
→ Codex agent follows them manually
→ real project work exposes gaps
→ definitions improve
→ Axiom CLI automates proven workflows
```

The harness is therefore not disposable scaffolding. It is an executable product hypothesis.

## Boundaries

This version deliberately does **not** decide:

- final CLI command model;
- persistence engine;
- cloud synchronization;
- internal task database;
- provider implementation;
- production deployment;
- final agent schema;
- adoption of token-saving/context tools.

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
