---
name: axiom-analyze-converge-experimental
description: Experimentally analyze coherence across neutral Axiom artifacts without changing product behavior.
---

# Experimental Axiom Analyze/Converge

This capability exists only for Scenario 002. It is not an Axiom product skill.

## Inputs

Read the neutral capability contract, then all supplied specification, plan,
task, implementation, test, and decision artifacts. Never read the scoring
oracle, previous runs, or comparison.

## Procedure

1. Inventory stable references: requirements, acceptance criteria, plan items,
   tasks, decisions, implementation behaviors, and verifications.
2. Build a forward matrix from every requirement and acceptance criterion to
   plan, tasks, implementation, and verification.
3. Build a reverse matrix from every plan item, task, and implementation
   behavior to a requirement or explicit decision.
4. Compare semantic predicates across artifacts: actor, input ownership,
   generated values, invariants, side effects, error behavior, and lifecycle.
5. Check whether material technology, persistence, or operational choices have
   decision evidence. Do not demand an ADR for a reversible implementation
   detail without long-term trade-offs.
6. Check acceptance text for a metric, threshold, conditions, and observable
   evidence when the criterion claims quality or performance.
7. Separate missing work from deliberate fixture boundaries. A file marked
   non-executable is still semantic implementation evidence; absence of a full
   runtime, HTTP adapter, dependency manifest, or concrete third-party adapter
   is not itself a finding unless a neutral requirement depends on it.
8. Deduplicate by root cause. Do not emit downstream consequences as separate
   findings when one contradiction explains them.
9. Return normalized findings only; never modify artifacts or append tasks.

## Severity

Use the neutral contract. Material requirement, contradiction, unsupported
scope, decision, or unverifiable-acceptance gaps are normally `major`. Report
uncertainty through confidence and human-decision fields, not inflated counts.

## Boundaries

- No Spec-Kit files, templates, code, or runtime.
- No product implementation.
- No corrections to scenario artifacts.
- No assumption that Repository is the same as Project.
