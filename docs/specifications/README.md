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
| POC #19 | **In progress** — PR #28 merged; [draft PR #29](https://github.com/rgomids/axiom/pull/29) adds macOS/Linux, ACL and sync-fault proof; full fault and hostile-race proof pending |
| POC #20 | **In progress** — black-box and cross-platform Evidence recorded; full POC failure matrix pending |
| POC #21 | **Dogfooding recorded; human acceptance pending** |
| Specification 002 / POC #14 | **Partial implementation; not Accepted/Done** |

PR #27 merged the initial POC baseline; PR #28 merged bounded hardening. Current verification and dogfooding
Evidence, including open blockers, are recorded in
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
codecs. POC #16–#18 additionally delivered the limited CLI, filesystem
persistence and local installation noted above. Runtime adapters and orchestration
remain unavailable; #19–#21 proof and acceptance work remains open.

## 003 — E2E Codex POC

Implementation-authorized on 2026-09-19 and tracked by #30–#40.

| Artifact | Status |
|---|---|
| [Specification](003-e2e-codex-poc/spec.md) | **Approved** |
| [Clarifications](003-e2e-codex-poc/clarifications.md) | **Resolved** |
| [Plan](003-e2e-codex-poc/plan.md) | **Approved** |
| [Tasks](003-e2e-codex-poc/tasks.md) | **Approved / authorized** |
| [T31 Evidence](003-e2e-codex-poc/evidence-t31.md) | **Ready for review** |

The requested standalone skill spelling `axiom:<skill>` is incompatible with the
validated Codex hyphen-case skill-name contract. The authorized reversible POC
mapping is `$axiom-<skill>`. Technical completion will leave #30/#40 awaiting
explicit human acceptance.
