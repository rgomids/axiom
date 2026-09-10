# B vs C Scorecard

No numeric score is used. Every comparison is tied to Scenario 002 evidence.

| Criterion | B — Spec-Kit encapsulated | C — Axiom-native experimental |
|---|---|---|
| Seeded finding recall | 3/5 expected findings (60%). Missed unsupported Redis and missing PostgreSQL decision. | 4/5 (80%). Missed unsupported Redis. |
| Unexpected findings | 3, all manually classified out of scope for the non-executable fixture. | 1, manually classified out of scope for the non-executable fixture. |
| Validated false positives | 0. Unexpected does not automatically mean false positive. | 0. Unexpected does not automatically mean false positive. |
| False negatives | 2/5. One is directly caused by decisions not entering official capability input. | 1/5. Reverse trace did not separate unsupported Redis from architecture-decision treatment. |
| Traceability | Strong spec/plan/task and appended-task trace; loses neutral decision trace. | Direct neutral references across all five artifact classes; no path translation. |
| Evidence quality | Exact lines and convergence diff; raw events preserved. Decision evidence absent by contract. | Exact cross-artifact evidence and explicit fixture limitation; raw events preserved. |
| Integration complexity | 226 shell lines + 37 prompt lines, pinned init, generated workspace, normalization, error/cleanup paths. | 50-line skill + 19-line prompt; shared neutral validator/scorer only. |
| Translation loss | Material: decisions preserved as a file but ignored by official analyze/converge; paths, categories, severity, and state translated. | None inside neutral inputs; capability reads Axiom-shaped artifacts directly. |
| Token usage | 193,169 input; 8,833 output. | 77,514 input; 4,255 output. |
| Runtime | 177.83s capability + 1.30s preparation. | 89.82s capability. |
| Context size | 2.49x C's measured input tokens despite one fewer reported artifact. | Smaller measured context in this run. |
| Failure handling | Fail-closed tests pass; upstream init failure remains raw exit 1 and partial temp cleanup. Converge mutates disposable tasks. | No external runtime failure class. Model/schema failures still need normal execution handling; baseline exposed schema compatibility errors transparently. |
| Upgrade risk | Real: pinned CLI, skills, scripts, output language, and scaffold. No newer stable existed, so migration remains untested. | No executable upstream upgrade; conceptual drift must still be monitored and mappings versioned. |
| Vendor coupling | High at adapter implementation/runtime boundary, though state is disposable. | Low runtime coupling; concepts only. |
| Multi-repo compatibility | Requires synthetic aggregate project, three projections, or per-repo runs plus Axiom aggregation. | Natural Project-level contract is possible, but not implemented or proven. |
| Axiom domain fit | Partial. Feature flow fits; Decision and Project/Repository boundaries do not. | Stronger direct fit with Specification, Decision, Finding, Evidence, and Project concepts. |
| Deterministic potential | Shell prerequisites and manifests deterministic; semantic analyze/converge remain model-judged and output normalization is fragile. | Matrix/schema/scoring can become deterministic; current semantic analysis still model-judged. |
| Maintenance cost | Adapter compatibility, pinning, failures, cleanup, migrations, raw/normalized schemas, and upstream regression fixtures. | Axiom owns capability quality and tests; less integration surface, more native implementation responsibility. |
| Replaceability | Disposable adapter protects authoritative Axiom state, but consumers of normalized upstream semantics may still couple. | No external runtime to replace; experimental skill can be discarded with this directory. |

## Decision-rule application

The pre-run rule makes major seeded-finding detection primary, then prefers
lower translation, coupling, upgrade risk, and multi-repository impedance when
quality is equivalent. C detected one more expected major finding, avoided B's
Decision-input translation loss, produced two fewer unexpected findings, and
used materially less time/context. Both paths produced zero validated false
positives. B therefore does not meet the predeclared conditions needed to
displace C in this scenario.
