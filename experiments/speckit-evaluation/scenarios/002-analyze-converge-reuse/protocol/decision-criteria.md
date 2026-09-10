# Pre-run Decision Criteria

Frozen before any Scenario 002 execution. Criteria may only change through an
explicit dated amendment that preserves this original text and explains why.

## Evidence favoring B — Spec-Kit encapsulation

B is favored if it:

- detects materially more expected findings or materially fewer false positives;
- preserves exact evidence and traceability through a small isolated adapter;
- incurs low translation loss and invents no material Axiom semantics;
- fails closed with visible upstream errors;
- tolerates pinned-version execution and controlled upgrades;
- has acceptable runtime, token, context, and operational cost;
- remains replaceable without migrating authoritative Axiom state.

## Evidence favoring C — Axiom-native

C is favored if it:

- reaches equivalent expected-finding detection and evidence quality;
- materially reduces translation, coupling, context, or execution overhead;
- represents Axiom Project, Repository, Decision, Evidence, and multi-repository
  concepts without tool-specific leakage;
- exposes failures deterministically and works without a Spec-Kit runtime;
- has lower long-term maintenance and upgrade risk;
- remains conceptually compatible without copying upstream templates or code.

## Decision rule

Detection of major expected findings is the primary gate. A strategy that misses
a major contradiction or requirement gap cannot win on cost alone. If detection
is equivalent, prefer lower translation loss, coupling, upgrade risk, and
multi-repository impedance. One scenario cannot authorize production adoption;
an inconclusive result remains valid.
