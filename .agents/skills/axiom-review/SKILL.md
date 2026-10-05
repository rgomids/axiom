---
name: axiom-review
description: Review Axiom changes against specification, architecture, verification evidence, and security with severity-based findings.
---

# Axiom Review

## Lanes

Review independently where useful:

1. specification compliance;
2. architecture;
3. verification;
4. security.

Do not create artificial findings to fill every lane.

## Finding schema

Each finding should include:

```yaml
severity: blocker|critical|major|minor|note
category:
evidence:
location:
rule_or_requirement:
impact:
recommendation:
confidence:
blocking:
```

## Severity

- **blocker** — cannot safely proceed;
- **critical** — severe correctness/security issue;
- **major** — must fix or receive explicit waiver;
- **minor** — improvement/non-blocking issue;
- **note** — observation.

## Procedure

1. Read spec/acceptance criteria.
2. Inspect diff and affected behavior.
3. Run deterministic checks where possible.
4. Verify tests test the intended behavior.
5. Check architecture/dependencies.
6. Check security boundaries and dangerous operations.
7. Identify the accepted ADRs the change affects (`docs/decisions/README.md`) and check for material contradiction.
8. Remove duplicate findings.
9. Report highest-severity issues first.
10. State explicitly when no blocking finding was identified.

An unreconciled material contradiction with an accepted ADR is a finding of severity **major** (blocking) or higher until the implementation complies with the accepted ADR or an ADR evolution under `docs/decisions/0018-adr-evolution-and-supersession-governance.md` resolves it. A proposed ADR needs explicit human acceptance; never infer acceptance from passing tests, CI, or merge readiness. Passing `scripts/check-adr-governance.py` proves structure only, not the absence of semantic conflict.

Do not equate passing tests with proof of correctness.
