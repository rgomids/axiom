# Measured Metrics

Token values come from Codex CLI `turn.completed.usage` events. Time comes from
`/usr/bin/time -p`. Each model run used `gpt-5.6-sol`, high reasoning effort,
one prompt interaction, and the same frozen scenario.

| Metric | Current Axiom baseline | C — Axiom-native | B — Spec-Kit encapsulated |
|---|---:|---:|---:|
| Capability wall time | 107.35s | 89.82s | 177.83s |
| Adapter preparation | N/A | N/A | 1.30s |
| Input tokens | 94,595 | 77,514 | 193,169 |
| Cached input tokens | 69,632 | 52,480 | 149,248 |
| Output tokens | 5,024 | 4,255 | 8,833 |
| Reasoning output tokens | 3,155 | 2,888 | 6,075 |
| Prompt interactions | 1 | 1 | 1 |
| Human follow-ups | 0 | 0 | 0 |
| Reported consulted artifacts | 10 | 10 | 9 |

B consumed 2.49 times C's input tokens and 2.08 times its output tokens. B's
capability run took 1.98 times C's wall time; including preparation, 1.99 times.
This is one controlled run, not a stable performance benchmark.

Prototype size, excluding shared neutral validation/scoring utilities:

- C: 50-line experimental skill plus 19-line prompt;
- B: 226 lines of adapter/failure-test shell plus 37-line prompt;
- B disposable generated state: 31 files, about 292 KiB across `.specify/` and
  installed Spec-Kit skills.

Raw events, results, timing, preparation output, translation manifest, and the
convergence diff remain under `evidence/runs/`.
