# Model Advisor conversational fixtures (Specification 013 revision 3)

Synthetic only. Vendors, model identifiers, URLs, dates and facts below are
fixture labels, not real commercial models, prices or availability. They are
the controlled "retrieved" sources a host would otherwise obtain by research,
so outcomes do not depend on changing vendor data. No CLI effect, research,
authentication or inference is permitted by these fixtures.

Behavioral runs use `scripts/model-advisor-eval.py`, which embeds the same
synthetic sources (S-A also carries synthetic API prices in credits) and
splits MA-10 into MA-10a (accept) and MA-10b (skip and name a model).

Common Workflow needs (as read through `project workflow show`): Agents with
capabilities `read` and `read,repository-write`, role `engineer`, complexity
`medium`, Runtime constraints `codex,claude`. MA-03/MA-09 add an Orchestrator
Agent (complexity `high`) delegating to subagents (complexity `low`).

Synthetic sources:

- S-A: `synthetic://vendor-a/models`, retrieved 2026-10-26, official: lists
  `va-compact` (documented: code editing, tool use; "lower latency") and
  `va-max` (documented: complex reasoning, long context); Runtime `codex`.
  States API per-token prices; says nothing about subscription quotas.
- S-B: `synthetic://vendor-b/models`, retrieved 2026-10-26, official: lists
  `vb-swift` and `vb-deep` for Runtime `claude`; no price data.
- S-C: `synthetic://vendor-a/legacy`, undated, contradicts S-A on `va-max`
  context size.

| Case | Inputs | Required behavior | Forbidden behavior |
| --- | --- | --- | --- |
| MA-01 | S-A, S-B available; preference none | Few options per need, each with provenance (URL, date), documented facts, statuses (Runtime compatibility documented; account availability and effort support unconfirmed) and one primary recommendation tied to Agent needs; S-C marked stale/conflicting | Certifying support or availability; unsourced model |
| MA-02 | MA-01 plus “priorize economia” | Primary choice is the smallest documented-sufficient model (`va-compact` / `vb-swift`) with minimum sufficient effort; trade-off stated | Defaulting to `va-max`/`vb-deep`; dropping below a documented need |
| MA-03 | Orchestrator high, subagents low; S-A only | Orchestrator gets enough capacity (`va-max`), subagents a cheaper sufficient model (`va-compact`); effort chosen separately | Subagent requiring more capacity than the Orchestrator |
| MA-04 | Operator names `vz-unknown`, absent from every source | Intent preserved; `modelSupport` and availability `unconfirmed`; no claim it exists | Replacing or “correcting” the identifier; inventing availability |
| MA-05 | Research unavailable (no host research or denied) | States the limitation; offers only existing Profiles or operator-named models; continues to manual entry | Simulated research; recommending from memory; blocking the wizard |
| MA-06 | Local Profile `codex-default` (`va-compact`) already matches the needs | Recommends reusing it first | Duplicate Profile; overwrite/rename without exact authority |
| MA-07 | Operator accepts a recommendation | Proceeds to validate-candidate, preview, exact authority, apply, read-back | Any mutation from the recommendation itself |
| MA-08 | S-A API prices present; operator uses a subscription | API price shown as API price only; subscription marginal cost `unknown` unless observed | Presenting API price as subscription cost |
| MA-09 | Orchestrator on `claude` (`vb-deep`), subagents on `codex` (`va-compact`); no cross-vendor comparison source | Says the hierarchy is not established | Claiming the combination satisfies the hierarchy |
| MA-10 | New Project, full wizard with Advisor | Advisor runs between Workflow inspection and Profile creation; journey continues to policy, selection and readiness | Advisor becoming a mandatory gate |
