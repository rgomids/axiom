# Protocol Log

- `2026-08-11T21:05:29Z` — frozen scenario, neutral capability contract,
  expected-findings oracle, current-Axiom baseline prompt, and decision criteria
  checksummed before any Scenario 002 model execution.
- Base repository commit: `5a77fae04a23c8b8320eb7be48f5bb2af8028def`.
- Executor fixed for comparable model runs: bundled Codex CLI
  `0.147.0-alpha.6.5`, model `gpt-5.6-sol`, reasoning effort `high`.
- Latest stable Spec-Kit release observed before execution: `v0.16.2`; pinned
  evaluated commit `4871b485f97c7fa452ec58eba325d87536c55c34`.

No experimental Axiom capability or Spec-Kit adapter existed at freeze time.

- `2026-08-11T21:06:22Z` — baseline attempt 1 stopped before model execution.
  Codex rejected `oneOf` in the output schema with HTTP 400
  `invalid_json_schema`. No finding or token usage was produced. The schema was
  amended only to express the same nullable field as `type: ["string", "null"]`;
  scenario, oracle, prompt, and decision criteria remained unchanged. Original
  schema hash: `bce162e140af051cd756a9794a22cb3adc0ae8ac1d0031fbbce55f672727c66b`.

- `2026-08-11T21:07:08Z` — baseline attempt 2 also stopped before model
  execution: Codex rejected `uniqueItems` in the output schema. The shell
  wrapper then encountered zsh's reserved `status` variable while reporting the
  already-failed child. The schema retained required fields/types/enums but
  moved pattern, bounds, uniqueness, and non-empty checks to the deterministic
  validator. Previous schema hash:
  `3c3aba6a35604129f848f4d4c5f72f67d2b5f69bad49c6658e846837ebddff20`.

- `2026-08-11T21:14Z` — an inspection-only pinned Spec-Kit initialization was
  accidentally targeted at the Axiom root because the command omitted the
  temporary-directory `cd`. Git inspection showed only untracked `.specify/`
  and ten `.agents/skills/speckit-*` directories; no tracked file changed. The
  eleven exact paths were moved to recoverable quarantine
  `/tmp/axiom-accidental-specify.zzz5gh`, then root status was rechecked. The
  adapter was corrected to change directory inside a subshell before init.

- `2026-08-11T21:22Z` — current-Axiom baseline complete before either prototype
  existed: 107.35 seconds, 94,595 input tokens, 5,024 output tokens. After
  baseline scoring, experimental C capability, B adapter, validator, scorer,
  and failure tests were created, tested, and frozen in
  `protocol/prototype-freeze.sha256` before prototype model runs.

- `2026-08-11T21:26Z` — after the C run, scoring normalization was amended to
  recognize an expected stable reference contained in a compound
  `subject_reference` such as `FR-001 / AC-001`. Category still must match
  exactly. This prevents formatting-only false negatives and changes no run
  output or decision criterion. Previous scorer hash:
  `d0f3a5a26d20e136a88c8101d878014cd9855eccad67d541d4682a8d2ca3ca0b`.

- `2026-08-11T21:35Z` — post-run review found that normalization failure could
  leave a pre-existing destination file looking current. The adapter now exits
  73 when the destination exists, and the failure test proves its bytes remain
  unchanged. Measured runs and normalized results were not changed. Previous
  normalizer hash:
  `9e724d301c7ee0e76b7edbf058c966f41135cb2f1fd0e53d3d667170dd904e32`;
  previous failure-test hash:
  `b1bf02f67a85a7f043acffc4aac6f4881fbb4417e73079879b0281e6ab2a8d1e`.

- `2026-08-12T03:29:59Z` — post-run methodology amendment separated unexpected
  findings from validated false positives. All eight unexpected findings were
  individually reviewed against the frozen non-executable fixture boundary and
  classified out of scope. The scorer now accepts the review artifact, reports
  seeded recall and classification counts, rejects substring-only reference
  matches, and records the matching actual finding ID. Original scorer hash:
  `e9fd5cbbdb3f9d18cd8d65b13b3f02602786a2a4805a97ed23702367556c82f7`;
  original test hash:
  `77ef616c8ae23674f6d645c049293fb07f60d224c04f64f8acbf61962576540b`.
  The integrity manifest was amended for the methodology tooling; frozen
  scenario/protocol inputs and every original model result, event stream, and
  timing file remain unchanged. No model was re-executed.
