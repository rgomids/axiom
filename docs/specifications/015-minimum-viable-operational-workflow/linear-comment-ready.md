# Linear comment artifacts — AXM-42 Specify

## Historical snapshot (attempt-01, pre-PR / pre-linkback)

**Archival only.** Captured when research existed only in the local worktree, **before** [PR #366](https://github.com/rgomids/axiom/pull/366) and before the published Linear `stage.completed` event. It is **not** an unperformed “ready-to-post” instruction for the current branch state.

- Snapshot investigation baseline SHA: `440e44a68638e08e901fd3f87f5b15cd395a6365` (code inspected; research file **not** on that commit — `blob/440e44a…/docs/research/axm-42-…` → 404).
- First commit containing these docs / reviewed HEAD: `74b668460ef8f53de7323fce54f2f0adc61d646d`.
- Published Specify completion (do **not** overwrite): [AXM-42 comment 4ee399c9](https://linear.app/rgomids/issue/AXM-42#comment-4ee399c9).
- Valid research permalink at review SHA: https://github.com/rgomids/axiom/blob/74b668460ef8f53de7323fce54f2f0adc61d646d/docs/research/axm-42-minimum-viable-workflow-investigation.md

### Preserved attempt-01 body (historical wording)

<details>
<summary>Expand historical ready-to-post text (attempt-01)</summary>

## AXM-42 Specify — investigation + Proposed Spec 015 (attempt-01)

**Outcome:** COMPLETED (research + Proposed Specification delivered locally).  
**HEAD (investigation baseline):** `440e44a68638e08e901fd3f87f5b15cd395a6365`  
**Artifacts (paths; GitHub blob at 440e44a was invalid — use 74b6684+ after PR):**  
- `docs/research/axm-42-minimum-viable-workflow-investigation.md`  
- `docs/specifications/015-minimum-viable-operational-workflow/spec.md`

First real dispatch blocked primarily by Axiom product ceremony (Plan + validator digest without public publisher; SDD ritual; manual advance), not by ADR-0032 certification. Live Codex/Claude E2E **NOT_RUN**. Linear publish from that session was **REPORT_PENDING** at snapshot time; later published as comment `4ee399c9`.

**event_id (historical, already published):** `AXM-42:specify:attempt-01:stage.completed:440e44a68638e08e901fd3f87f5b15cd395a6365`

</details>

---

## Historical correction handoff (correction-01)

**New** Linear report payload for Spec 015 F1–F4 correction on the **same** PR #366. Do not re-post attempt-01. Do not claim merge/acceptance.

| Field | Value |
|-------|-------|
| PR | https://github.com/rgomids/axiom/pull/366 |
| Head SHA | `d07cf5fe76f9852b545b7502740117af46cc8aeb` (spec body correction: `b50ccca27037ee868c984055bd397631322a437a`) |
| Prior review | https://github.com/rgomids/axiom/pull/366#pullrequestreview-5707420506 |
| Spec @ tip | https://github.com/rgomids/axiom/blob/d07cf5fe76f9852b545b7502740117af46cc8aeb/docs/specifications/015-minimum-viable-operational-workflow/spec.md |
| Research @ tip | https://github.com/rgomids/axiom/blob/d07cf5fe76f9852b545b7502740117af46cc8aeb/docs/research/axm-42-minimum-viable-workflow-investigation.md |
| Receipt | `completion-receipt-specify-correction-01.json` |
| Linear publish (this session) | `false` / REPORT_PENDING (MCP needsAuth) |

See session handoff / operator paste for the human-readable report + AWF-EVENT `correction-01`.


## Historical executor snapshot — correction-02 structural replacement proposal

Correction-01 above is archival; its F1–F4 facts and event identities remain
unchanged. The human direction changed in Linear comment `7a717e9f` and review
[AR-01](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5708094263).
Correction-02 starts from verified PR HEAD
`c3a53af4f08c1ead6328d05a5fc675e1cfcb8bcc`; main remains
`440e44a68638e08e901fd3f87f5b15cd395a6365`.

The revised [Spec](spec.md) and
[research](../../research/axm-42-minimum-viable-workflow-investigation.md)
propose structural removal/consolidation, one canonical Execution, bounded
retirement and safety outcomes. These are proposals, not implemented changes or
accepted architecture. The completion comment for **correction-02 only** carries
the exact final PR HEAD, validation results and event ID; do not replay either
older attempt. No live inference, merge, release, delivery acceptance or AXM-12
unblock is authorized or claimed.

Technical decisions: bounded source-inspection delegation, Spec integration and
validation owned by the main executor; keep accepted records and previous
receipts unchanged; count reduction baselines in Plan rather than invent them.
The prior builtin Workflow fixture drift finding remains deferred and does not
expand this correction's requirements.

Next gate: `re_review_revised_Spec015_and_human_material_acceptance`.
No real callback/continuation integration was invoked; handoff is to review,
not automatic Plan or Implementation.

### Local validation evidence (correction-02)

- PASS: `git diff --check`; Spec relative links and structural contract checklist.
- PASS: `python3 scripts/check-adr-governance.py .` (37 ADRs), its 56-case
  regression suite; automation registry (99 surfaces), its 33-case suite;
  issue-label tests (30 cases); sensitive-file checker and its regression suite.
- PASS: `GOCACHE=/tmp/axm42-c02-go-cache go build -o /tmp/axm42-c02-lingo ./cmd/lingo`
  as a synthetic validator prerequisite, not a product qualification claim.
- FAIL: `GOCACHE=/tmp/axm42-c02-go-cache ./scripts/validate-repository.sh .`
  stops in `scripts/test_model_advisor_eval.py` (one failure, five errors).
  `workflow_needs()` at `scripts/model-advisor-eval.py:135` raises
  `KeyError: 'revisions'`. A synthetic-only reproduction observed configure
  returning `Portable Project publication failed`, then workflow list
  returning `project_not_found`. Root cause remains unverified; this
  correction does not alter that code. The first run also lacked writable
  default Go cache; the writable-cache rerun above is the authoritative result.
- NOT_RUN: real Codex/Claude inference, R2/R3 two-repository E2E, process/fault
  qualification of the proposed replacement, migration and reduction counts.
- Secret scanner `gitleaks` unavailable; repository sensitive-file checks ran.

Deferred validation finding: diagnose the existing synthetic Project publication
failure outside this docs-only correction, with the commands and source locations
above as evidence. This is not an added MVP requirement or an assertion that
remote CI failed. Observe remote checks on the exact published HEAD separately.


### Publication limits observed in the correction-02 executor session (historical)

`git add` failed creating `/home/grokbot/axiom/.git/worktrees/axm-42/index.lock`
with `Read-only file system`; no commit/push occurred. The staged sensitive-file
check saw an empty index and is not evidence that these edits were staged.
Worktree sensitive-file checks passed. The GitHub metadata write tool also
refused because it requires approval while this session's approval policy is
`never`; no alternative write path was used. PR #366 still has HEAD
`c3a53af4f08c1ead6328d05a5fc675e1cfcb8bcc`. Local documentation and the generated
patch are the correction output; publication remains pending. The outdated PR
summary requires editorial application of the prepared replacement.

The Linear completion-comment write was also refused by the same approval-policy
restriction. No correction-02 comment was published and no raw remote read-back
is claimed. The generated local receipt carries `linear_published:false`,
`REPORT_PENDING` and `stage.partial`; it includes the actual unchanged PR SHA.


## Canonical publication read-back — correction-03 / F6

The correction-02 section above preserves what its executor observed **before Maestro publication**; its c3a53af HEAD and failed writes are historical, not current publication status. Remote AXM-42 [correction-02 receipt 4f4b9cf8-16c2-4d43-b460-eae29314a8e3](https://linear.app/rgomids/issue/AXM-42#comment-4f4b9cf8-16c2-4d43-b460-eae29314a8e3), read back through the authenticated connector during correction-03, records Maestro commit/push at `3558bba51b0af11091f3d41305af264ba060d7bc` and the published PARTIAL event. GitHub PR read-back independently confirmed that SHA as correction-03 input. Earlier receipts remain unchanged.

Current publication truth is [PR #366](https://github.com/rgomids/axiom/pull/366) for live HEAD/checks and the deduplicated `AXM-42/specify/correction-03/HEAD` completion event on [AXM-42](https://linear.app/rgomids/issue/AXM-42) for this correction's exact SHA, checks and publication outcome. This document does not predeclare a commit, CI pass or callback. F5 now proposes explicit per-attempt snapshot ownership/precedence and partial ADR-0024/Spec 007 #306 supersession; acceptance remains pending. F6 is reconciled by this appended remote pointer, not by rewriting historical observations.

Next gate: `re-review-PR366-then-human-material-acceptance`. No implementation, merge, release, closure or AXM-12 unblock authority. If no real continuation integration exists, the completion event is the checkpoint for orchestrator consumption, not a claim of dispatch.
