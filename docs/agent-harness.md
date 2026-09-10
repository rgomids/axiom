# Agent Harness

## Purpose

The current harness makes Axiom's product hypotheses executable through repository-native agent instructions, skills, policies, templates, and deterministic validators before the product CLI exists.

```text
Axiom definitions
→ manual agent workflows prove behavior
→ validated behavior becomes product contracts
→ Axiom CLI automates proven workflows
```

The harness is therefore not disposable scaffolding. It is the current manual
executable product hypothesis.

The accepted future direction centralizes executable workflow behavior in Lingo
under Axiom contracts, while runtime skills tend toward thin entrypoints.
Existing skills do not change until a Specification defines migration,
compatibility, and acceptance evidence.

## Boundaries

This version deliberately does **not** decide:

- production deployment;
- final agent schema;
- adoption of token-saving/context tools;
- Lingo implementation or migration from current skills;
- runtime adapters, Agent Planner, orchestration, execution graphs, or model discovery.

Those should emerge from specifications and validated experiments.
