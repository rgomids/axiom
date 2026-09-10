# Scenario 002 Result

## Evidence

- Current Axiom baseline: 4/5 expected, 1 missed, 80% seeded recall,
  4 unexpected, 0 validated false positives.
- Experimental C: 4/5 expected, 1 missed, 80% seeded recall, 1 unexpected,
  0 validated false positives.
- Encapsulated B: 3/5 expected, 2 missed, 60% seeded recall, 3 unexpected,
  0 validated false positives.
- Manual review classified every unexpected finding as an out-of-scope
  production-completeness observation for the frozen non-executable fixture.
- C used 77,514 input tokens and 89.82 seconds.
- B used 193,169 input tokens and 179.13 seconds including preparation.
- B's translation boundary preserved the decisions file but official
  analyze/converge did not consume it; PostgreSQL without an ADR was missed.
- B converge appended six tasks to the disposable task file.
- Adapter failure tests fail closed; a missing pinned ref leaves partial temp
  state and exposes a raw upstream exit.
- Latest stable Spec-Kit equaled the pin, so real upgrade migration was not testable.

## Interpretation

Spec-Kit's capability remains mature and useful inside its native feature flow.
Encapsulation successfully produced findings and actionable convergence tasks,
but did not preserve all Axiom semantics through translation. The adapter added
operational state, mutation, failure, version, and cleanup responsibilities
without improving detection on this fixture.

The experimental Axiom-native method did not improve seeded-finding recall over
the current baseline, but its explicit fixture boundary produced fewer
unexpected, out-of-scope observations. This is not evidence that B produced
more false positives: both B and C produced zero validated false positives.
Every path still failed to identify Redis as unsupported behavior, so C is not
complete and should not become official behavior from this result.

## Recommendation

Keep **C — conceptual compatibility** as the leading hypothesis. B remains a
viable optional adapter strategy for a future bounded workflow, not the default
architecture. Ranking remains C, B, D, A. The basis is C's higher seeded recall,
B's Decision-input translation loss, and C's lower runtime/context, runtime
coupling, failure surface, and multi-repository impedance—not an invalid
false-positive comparison.

This result does not accept ADR-0002, authorize implementation, or establish an
interoperability contract.

## Limitations

- One small, synthetic, single-repository semantic fixture.
- One model run per approach; no variance estimate.
- Unexpected-finding classification requires review judgment because the oracle
  intentionally covers only seeded faults. The explicit review is reproducible,
  but it is not complete ground truth for all possible findings.
- C prototype remains model-driven; deterministic analyzer behavior is only a
  design possibility.
- No newer stable Spec-Kit release existed for upgrade testing.
- Multi-repository behavior is a thought experiment because Axiom's minimum
  Project/repository execution contract remains undefined.

## Redis gap

All approaches missed Redis introduced without a supporting requirement. This
remains a candidate for a future Axiom capability that detects introduced
behavior or implementation scope without requirement, decision, or approved
plan rationale. No capability implementation is authorized by this result.

## Remaining evidence that could materially change the B vs C decision

No additional scenario is currently required before human review of ADR-0002.

Future evidence could reopen the comparison, but prerequisites do not exist now:

| Question | Why it matters | Result favoring B | Result favoring C |
|---|---|---|---|
| Can a bounded B adapter preserve Axiom Decision input and match C's seeded recall without adding Axiom-native semantic analysis? | Decision translation loss caused B's lower detection in Scenario 002. | B reaches or exceeds C's seeded recall with small, isolated translation. | Decision loss persists or matching C requires duplicating native analysis. |
| How do B and C operate after Axiom defines minimum Project/repository input and evidence contracts? | Multi-repository delivery is central to Axiom and unproven for both paths. | B aggregates repositories with lower total complexity and equivalent traceability. | B needs synthetic projections or loses provenance while C consumes the Project contract directly. |
| What is B's real migration cost when a newer stable Spec-Kit release exists? | Current upgrade risk is reasoned because the evaluated release was latest. | Upgrade is controlled, compatible, and cheaper than native maintenance. | Adapter/schema/failure changes create material migration or regression cost. |

Agent-harness generation does not test the observed analyze/converge Decision
gap and is not justified as Scenario 003.
