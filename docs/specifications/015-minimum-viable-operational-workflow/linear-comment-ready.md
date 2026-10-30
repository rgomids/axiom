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

## Current correction handoff (correction-01)

Use this section for the **new** Linear report after Spec 015 F1–F4 correction on the **same** PR #366. Do not re-post attempt-01. Do not claim merge/acceptance.

**PR:** https://github.com/rgomids/axiom/pull/366  
**Prior review:** https://github.com/rgomids/axiom/pull/366#pullrequestreview-5707420506 (F1–F4 analytical REQUEST_CHANGES / GitHub COMMENT)  
**Spec:** https://github.com/rgomids/axiom/blob/spec/axm-42-minimum-viable-workflow/docs/specifications/015-minimum-viable-operational-workflow/spec.md  
**Research:** https://github.com/rgomids/axiom/blob/spec/axm-42-minimum-viable-workflow/docs/research/axm-42-minimum-viable-workflow-investigation.md  

Fill `pr_head_sha` / `event_id` with the **actual** pushed correction commit after push. If Linear MCP is unavailable, keep `linear_published:false` and leave payload for operator paste — see `completion-receipt-specify-correction-01.json`.
