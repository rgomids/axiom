# Specification 010 — Plan acceptance reconciliation

## Exact authority — 2026-10-20

The [human decision](https://github.com/rgomids/axiom/pull/335#issuecomment-6296942470)
by `rgomids`, revalidated during Tasks preflight, accepts the strategy and P1–P6
of [Plan 010 rev.1](https://github.com/rgomids/axiom/blob/a966510849d9dda0ab12f01eb4b396d51d7f4390/docs/specifications/010-installed-operational-documentation/plan.md)
at `a966510849d9dda0ab12f01eb4b396d51d7f4390`, subject to its D1–D4 dispositions.
[Linear trace](https://linear.app/rgomids/issue/AXM-37/distribuir-documentacao-operacional-versionada-junto-a-instalacao-do#comment-57738019)
confirms the same revision and authority: prepare Tasks only.

This is an editorial addendum, not Plan rev.2 or a new architecture decision.
The complete accepted Plan bytes, including Proposed/HUMAN_DECISION_REQUIRED and
old next-gate language, remain historical. This later exact decision controls
readiness for Tasks. [Plan Evidence](evidence-plan.md) and the earlier technical
[COMMENT review](https://github.com/rgomids/axiom/pull/335#pullrequestreview-5588642593)
retain their historical scope; neither substitutes for human acceptance.

## Dispositions incorporated into Tasks

| Decision | Exact accepted direction / remaining qualification | Tasks gate |
| --- | --- | --- |
| D1 — Accepted | Explicit v2 evolution, preserve v1 reading/history; archive, metadata, receipts and discovery independently versioned. Validate closed schemas, quotas and producer/consumer matrix before dependent implementation; proposed budgets need coverage verification | T01/T04; shell/Go pre-execution validation in T07 |
| D2 — Accepted with qualification | Private physical-executable namespace and immutable generations. Concrete mechanism depends on native macOS/Linux executing-image proof, especially A surviving executable replacement. Failure blocks adoption; current pathname hash alone cannot prove old image | T02/T04, all three host rows; dependent production blocked by G1 |
| D3 — Accepted | `axiom docs inspect`, `--json`/`--human`, public versioned schema/stable fields and states, defined exits, session-local man. Strictly read-only; executable help owns grammar | T01/T04 → T10–T12 |
| D4 — Accepted | Complete immutable objects/binding before binary rename; confirmed binary effect is logical commit. Installation identity coordinates locks despite differing receipt roots. Explicit recovery, truthful partial/uncertain effects; no automatic rollback/cleanup | T03/T04 → T08/T09 |

No concrete unresolved mechanism/limit is represented as proven. P1 concretizes
and validates these contracts; material infeasibility returns to a distinct human
decision before dependent work. No accepted Spec/Q1–Q6/ADR text is changed, and no
older ADR is superseded implicitly. Source remains unqualified, Windows withdrawn,
independent compatible A is the return route, in-place downgrade refused, no GC.

## Authority boundary

[Tasks rev.1](tasks.md) is Proposed, delivered for review. This acceptance neither
accepts Tasks nor authorizes product implementation, native experiments, merge,
release, functional acceptance or AXM-39/AXM-12 unblocking. AC-01–AC-08 remain
**NOT_RUN**. No canonical Execution or next dispatch is inferred.
