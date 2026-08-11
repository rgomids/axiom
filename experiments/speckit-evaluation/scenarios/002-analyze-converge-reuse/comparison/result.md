# Scenario 002 Result

## Evidence

- Current Axiom baseline: 4/5 expected, 1 missed, 4 unexpected.
- Experimental C: 4/5 expected, 1 missed, 1 unexpected.
- Encapsulated B: 3/5 expected, 2 missed, 3 unexpected.
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

The experimental Axiom-native method did not improve expected-finding recall
over the current baseline, but its explicit fixture boundary sharply reduced
noise. Both Axiom runs still failed to identify Redis as unsupported behavior,
so C is not complete and should not become official behavior from this result.

## Recommendation

Keep **C — conceptual compatibility** as the leading hypothesis. B remains a
viable optional adapter strategy for a future bounded workflow, not the default
architecture. Ranking remains C, B, D, A.

This result does not accept ADR-0002, authorize implementation, or establish an
interoperability contract.

## Limitations

- One small, synthetic, single-repository semantic fixture.
- One model run per approach; no variance estimate.
- Non-executable source/test fixtures made unexpected-finding adjudication
  dependent on the frozen oracle.
- C prototype remains model-driven; deterministic analyzer behavior is only a
  design possibility.
- No newer stable Spec-Kit release existed for upgrade testing.
- Multi-repository behavior is a thought experiment because Axiom's minimum
  Project/repository execution contract remains undefined.

## Remaining evidence

No Scenario 003 was executed. A real multi-repository coordination scenario
could change B vs C only after Axiom defines the minimum Project/repository
input and evidence contracts; running it now would compare invented harnesses.
A real upgrade scenario could change B's maintenance assessment when a newer
stable Spec-Kit release exists. Agent-harness generation would not test the
observed analyze/converge decision gap, so it is not justified here.
