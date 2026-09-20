# Specifications

Specifications define desired outcomes and observable behavior before implementation
plans. Approval gates remain separate: an approved Specification permits planning;
implementation requires an approved Plan/Tasks and explicit authorization for the
bounded delivery unit. Technical merge does not imply human acceptance or next-Task authority.

## 001 — Codex agent harness generation

[Specification](001-codex-agent-harness-generation/spec.md): **Proposed**.

## 002 — Lingo Project Initialization

First active vertical slice, in incremental implementation. Current lifecycle,
reconciled on 2026-09-18 against versioned artifacts, merged PRs and POC issues:

| Artifact / delivery | Status |
|---|---|
| [Specification](002-lingo-project-initialization/spec.md) | **Approved** |
| [Plan](002-lingo-project-initialization/plan.md) | **Approved** |
| [Tasks](002-lingo-project-initialization/tasks.md) | **Approved** |
| T01–T03 | **Accepted / merged** — PRs [#6](https://github.com/rgomids/axiom/pull/6), [#7](https://github.com/rgomids/axiom/pull/7), [#8](https://github.com/rgomids/axiom/pull/8) |
| T04 | **Merged; pending human acceptance/re-review** — [PR #9](https://github.com/rgomids/axiom/pull/9); not marked Accepted |
| POC #16–#18 | **Merged in [PR #27](https://github.com/rgomids/axiom/pull/27)** — executable CLI, minimal init/validate/reopen/name update and separate local install |
| POC #19 | **Closed / not planned for further POC work** — bounded hardening delivered; residual physical-power-loss/fault-matrix/hostile-race proof becomes MVP hardening input |
| POC #20 | **Closed / not planned for further POC work** — deterministic/E2E Evidence was sufficient for POC acceptance; broader Evidence requirements move to the future MVP scope |
| POC #21 | **Accepted / closed** — E2E dogfooding and limitations reconciled on 2026-09-20 |
| Specification 002 / POC #14 | **Partial implementation; not Accepted/Done** |

PR #27 merged the initial POC baseline; PR #28 merged bounded hardening. Current verification and dogfooding
Evidence, including the residual limitations accepted for POC closure, are recorded in
[POC Evidence](002-lingo-project-initialization/evidence-poc.md). The POC issue
deliveries do not establish completion of the full T05–T21 Task DAG.

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
T01–T04 established Project domain rules, application contracts and portable/local
codecs. POC #16–#18 delivered the initial CLI, filesystem persistence and local
installation. Specification 003 adds bounded Codex Runtime, GitHub Work Item and
sequential workflow adapters without completing the remaining Specification 002
Task DAG. #19/#20 were closed after POC acceptance without claiming their stronger
proof obligations were satisfied; those residual concerns are inputs to MVP
hardening. #21 was accepted/closed during the 2026-09-20 E2E POC acceptance.
This does not mark Specification 002 or parent #14 complete.

## 003 — E2E Codex POC

Implementation-authorized on 2026-09-19 and tracked by #30–#40.

| Artifact | Status |
|---|---|
| [Specification](003-e2e-codex-poc/spec.md) | **Approved** |
| [Clarifications](003-e2e-codex-poc/clarifications.md) | **Resolved** |
| [Plan](003-e2e-codex-poc/plan.md) | **Approved** |
| [Tasks](003-e2e-codex-poc/tasks.md) | **Approved / authorized** |
| T31–T40 Evidence / reconciliation | **Delivered / merged to `main` in PR #52** |
| [Final report](003-e2e-codex-poc/final-report.md) | **ACCEPTED — 2026-09-20** |

The requested standalone skill spelling `axiom:<skill>` is incompatible with the
validated Codex hyphen-case skill-name contract. The authorized reversible POC
mapping is `$axiom-<skill>`. #30/#40 were explicitly accepted and closed on
2026-09-20 after PR #52 merged to `main`.

Acceptance testing created MVP follow-ups
[#54](https://github.com/rgomids/axiom/issues/54),
[#55](https://github.com/rgomids/axiom/issues/55),
[#56](https://github.com/rgomids/axiom/issues/56) and
[#57](https://github.com/rgomids/axiom/issues/57). They record guided setup,
Intent-driven Work Item creation, provider-visible workflow progress and
argument-driven Runtime skill UX; they do not retroactively expand POC scope.

## 004 — Usable MVP v1 Baseline

[Specification](004-mvp-v1-baseline/spec.md): **Draft — ready for human review;
Human decision required**.

Issue [#62](https://github.com/rgomids/axiom/issues/62) authorizes Specification
and reconciliation only. The draft consolidates #54–#57 and #63–#68 into the
clean-environment journey from installation through explicit human acceptance.
It defines Project/Work Item UX, Provider workflow projection, Runtime selectors,
completion statuses, detailed Markdown artifacts, provenance, persistence,
migration, onboarding, acceptance criteria, and required Evidence.

Four material choices remain explicit: release installation/platforms, detail-
artifact ownership/retention, filesystem threat model, and supported migration
from the accepted POC. Approval permits Plan work only. No MVP implementation,
ADR acceptance, Tasks, release, or final acceptance is authorized.
