# Measured Metrics

Token values below come from Codex CLI `turn.completed.usage` events. They are
real reported values, not estimates. `input_tokens` includes the portion
reported separately as cached input.

| Metric | Axiom | Spec-Kit |
|---|---:|---:|
| Controlled execution wall time | 15m55s | 25m23s |
| Spec-Kit initialization | N/A | 0.63s |
| Input tokens | 4,271,455 | 7,919,796 |
| Cached input tokens | 4,068,608 | 7,670,272 |
| Output tokens | 60,008 | 71,325 |
| Reasoning output tokens | 24,497 | 28,724 |
| Human clarification batches | 1 | 1 |
| Live follow-up batches after frozen answer | 0 | 0 |
| Workflow evidence files retained | 9 | 9 |
| Workflow evidence lines retained | 1,186 | 938 |
| Changed implementation files | 5 | 5 |
| Changed implementation lines | 170 | 196 |
| Generated workflow scaffold | Existing Axiom harness | 31 files, about 296 KiB |

## Timing boundary

- Axiom: first nested Codex start at 03:04:10Z through final closure at
  03:20:05Z.
- Spec-Kit: first nested Codex start at 03:24:06Z through final output at
  03:49:29Z.

Wall time includes model and local tool latency. It is directly observable for
this run but not a general performance benchmark.

## Interpretation limit

Spec-Kit consumed about 1.85 times Axiom's input tokens in this experiment, but
the run also executed more named quality gates and produced richer official
design artifacts. Axiom retained more evidence lines because its custom
validation and review logs were more verbose. One scenario cannot establish a
stable token-cost ratio.
