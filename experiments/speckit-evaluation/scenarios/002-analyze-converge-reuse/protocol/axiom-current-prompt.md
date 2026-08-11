# Current Axiom Baseline Prompt

Use only the current Axiom harness copied into this isolated workspace. Do not
use Spec-Kit or any experimental analyze/converge capability.

Read every file under `scenario/` and the neutral contract at
`protocol/capability-contract.md`. Analyze consistency across specification,
plan, tasks, implementation, tests, and decisions. Do not change any file.

Return only JSON conforming to `protocol/findings.schema.json`.

Controls:

- do not read an oracle, expected result, prior run, or comparison;
- record exact relative paths in `artifacts_consulted`;
- use one interaction; do not ask follow-up questions;
- report human decisions needed in the result instead of inventing answers;
- do not infer missing product requirements;
- set `approach` to `C-current-axiom-baseline`;
- assign finding IDs `AXC-001`, `AXC-002`, and so on.
