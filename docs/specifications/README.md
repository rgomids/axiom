# Specifications

Specifications define desired outcomes and observable behavior before implementation
plans. Approval gates remain separate: an approved Specification permits planning;
implementation requires an approved Plan/Tasks and explicit authorization for the
bounded delivery unit. Technical merge does not imply human acceptance or next-Task authority.

## 001 — Codex agent harness generation

[Specification](001-codex-agent-harness-generation/spec.md): **Proposed**.

## 002 — Lingo Project Initialization

First active vertical slice, in incremental implementation. Current lifecycle,
reconciled on 2026-09-17 against versioned artifacts and merged PRs:

| Artifact / delivery | Status |
|---|---|
| [Specification](002-lingo-project-initialization/spec.md) | **Approved** |
| [Plan](002-lingo-project-initialization/plan.md) | **Approved** |
| [Tasks](002-lingo-project-initialization/tasks.md) | **Approved** |
| T01–T03 | **Accepted / merged** — PRs [#6](https://github.com/rgomids/axiom/pull/6), [#7](https://github.com/rgomids/axiom/pull/7), [#8](https://github.com/rgomids/axiom/pull/8) |
| T04 | **Merged; pending human acceptance/re-review** — [PR #9](https://github.com/rgomids/axiom/pull/9); not marked Accepted |
| T05–T21 | **Not started / not authorized** |

PR #9 received a final human re-review on 2026-09-17 reporting no findings and
“Ready to merge”, then merged. That review is recorded; it is not converted here
into an explicit Task Accepted decision. T04 acceptance remains a human lifecycle
gate. Neither its merge nor this documentation reconciliation authorizes T05.

Specification approval originated on 2026-09-11. Subsequent
[Clarifications](002-lingo-project-initialization/clarifications.md) preserve Q1–Q6
and H1–H12 history. Plan and Tasks approval followed in PRs
[#4](https://github.com/rgomids/axiom/pull/4) and [#5](https://github.com/rgomids/axiom/pull/5).
Dated pre-merge gates in those artifacts and Evidence describe their review-time
state; use this index for the current consolidated lifecycle.

Implementation Evidence:
[T01](002-lingo-project-initialization/evidence-t01.md),
[T02](002-lingo-project-initialization/evidence-t02.md),
[T03](002-lingo-project-initialization/evidence-t03.md),
[T04](002-lingo-project-initialization/evidence-t04.md).
These deliveries establish Project domain rules, application contracts and
portable/local codecs. They do not deliver a CLI, filesystem persistence,
operational installation, runtime adapters or orchestration. Concrete persistence
and Git mechanisms remain outside this documentation change.
