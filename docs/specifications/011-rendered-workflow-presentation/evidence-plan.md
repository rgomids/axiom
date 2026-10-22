# AXM-38 — Plan Evidence, revision 1

Date: 2026-10-22. **Local documentary checkpoint / PARTIAL publication.**
No implementation, native session, new CI or product acceptance is claimed.

## Verified authority and repository state

- Initial worktree clean; branch `axm-38-specify-rendered-workflow`.
- Local HEAD and authenticated GitHub PR HEAD: `c2298a9ae60d32ecbfea96c19a051a7514fe78b6`.
  PR [#336](https://github.com/rgomids/axiom/pull/336) OPEN/Draft, base `main`,
  base SHA `2e53228bfd27cff206c5c955f363bfff18fe745a`. No reconciliation delta.
- [Human approval](https://github.com/rgomids/axiom/pull/336#issuecomment-6333507483)
  and [technical review / CR-001](https://github.com/rgomids/axiom/pull/336#pullrequestreview-5609613349)
  read through authenticated GitHub integration. Acceptance binds r1 at the SHA
  above, blob `21705f315bbaa2365ac1078ad1bc868f74725555`, SHA-256
  `9fde9a000ae1f5d96f64fcc7356575157b1b82f087b0f60fc0b20efafb5ae00a`.
- Authenticated Linear AXM-38 and complete comments read (`hasNextPage: false`):
  In Progress, blockedBy empty, AXM-39/AXM-12 downstream. Approval and current
  Plan prompt verified. No operational metadata/status/relation mutation.
- `get_agent_context` failed: server returned tool-not-found. Work Item and
  repository sources remained available; no blocker inferred. No memory-derived project
  fact used; lightweight registry search returned no relevant AXM-38 entry.
- Current prompt authorizes Plan/editorial patch and one Linear callback; it
  conditions commit/push on actual authority. No new explicit commit/push grant
  found. Earlier Specify publication grant was for its own checkpoint. No Git
  write/publication, competing PR or PR body update performed.

## CR-001 retrospective reconciliation

Original checkpoint: source `2e53228bfd27cff206c5c955f363bfff18fe745a`, before
publication. [Original receipt](completion-receipt.json) remains byte-for-byte
unchanged, SHA-256 `06d11259fffb6876ebc748e2639d58148e4967ee3dbb190b275df3a5a436e422`; `pr_url:null`, uncommitted fields and old
next gate remain valid historical observations. Original Specify Evidence remains
unchanged. Later publication and human acceptance are explicitly separated in
[Spec editorial e1](spec.md), with PR/approval links and current next gate.

New Spec: **r1 editorial e1, PROPOSED**, Git blob
`ba83987e96bc31bd31778ef80ab9e4a2890b1706`,
SHA-256 `921389e131fa203366a013102dfde429df8a160f228029f995cadebf48208be1`. Delta classification: **EDITORIAL_PROVENANCE_ONLY**.
Exact comparison against accepted SHA below; requirements, P-01–P-05, AC/E1–E10,
compatibility inventory, limits and failure/security section unchanged. Ephemeral
Python asserted exact bytes from `## Problem, objectives and non-goals` through
before `## Decisions, ambiguities and next gate`; final section changes only
approval/provenance narration and next gate. Manual full diff review agrees.
Prior approval is not transferred to e1. Exact re-review/human reconfirmation needed.

**CR-001: locally patched; PARTIAL / pending publication, review and closure on
PR #336.** Not reported as remotely resolved. Reviewable patch is this Spec diff
and these local artifacts; no Task/renderer/CLI/skill/schema file changed.

## Source, safe local investigation and tests

- Inspected current WorkflowReport, Ref/Definition/Stage/Agent types, both schemas,
  byte bounds, semantic validation and current shared Workflow skill guidance.
  Packaging history is in `internal/codexruntime/manifest.go`; shared integration
  tests include Codex/Claude constructors. Source inspection is not installed proof.
- Ephemeral Python source checks: both schemas have 1–32 Stages/Agents and maximum
  concurrency 32. Go `MaxBytes`, byte-string checks and validation establish remaining
  bounds and sequential/dependency semantics. No schema/code changed.
- Four repository examples parsed locally: default r1/schema1 and default r2/schema2,
  custom r1/r2 both schema1; each has 10 Stages. Defaults have 0 Agent dependency
  edges; customs have 2. Custom `implementation` Stage has mode dag/concurrency 2,
  implementer/reviewer independent, integrator depends on both. These are definitions,
  not captured canonical completion events or observed `teste-sdd` installations.
- Initial ephemeral fixture check failed with `StopIteration` after assuming Stage
  ID `implement`. Corrected lookup uses declared Agent identity; re-run PASS, actual
  Stage ID `implementation`. No product defect or fixture edit inferred.
- Local schema/fixture inspection, normative-section/receipt byte equality: **PASS**.
- `git diff --check`: **PASS** for tracked editorial change; new-document whitespace
  and local Markdown/JSON references validated separately at final checkpoint.
- `./scripts/check-sensitive-files.sh .`: **PASS** (worktree).
- `gitleaks dir docs/specifications/011-rendered-workflow-presentation --redact --no-banner`:
  **PASS**, no leaks. Final documentary checks recorded in plan receipt.
- Transformation/renderer behavioral tests, CLI execution, installed-state mutation,
  real Codex/Claude sessions/screenshots, build and new CI: **NOT_RUN**, outside Plan.
  Source/fixture inspection does not count as deterministic implementation tests.

## Official host documentation — not empirical Evidence

Opened 2026-10-22, no host launch or rendering test:

- [OpenAI app features](https://developers.openai.com/codex/app/features) redirected
  to general features, not exact Codex build qualification.
- [OpenAI Visualizations](https://learn.chatgpt.com/docs/visualizations): named
  ChatGPT surfaces and differing availability; excludes Codex CLI/IDE from that
  feature. Does not establish exact Codex Desktop Workflow rendering.
- [Claude Code Desktop](https://code.claude.com/docs/en/desktop#preview-your-app):
  Browser preview and local HTML/PDF/video opening documented, not tested inline
  Workflow diagrams. No server/web app or new transport adopted.
- [Claude artifacts](https://support.claude.com/en/articles/17153992-what-are-artifacts-and-how-do-i-use-them):
  distinct artifact surface, Code session publication and capability requirements.
  Not a grant to publish/upload or evidence of a locally rendered diagram.

Documentation is a candidate inventory only: all empirical support **UNVERIFIED**.
The Plan defines later qualification and the VP-01 at-least-one-supported-host
witness; syntax choice does not introduce a human gate. New platform/effects or a
changed product requirement require concrete evidence and their own authority.

## CI observed for existing committed HEAD only

Authenticated `fetch_commit_workflow_runs` at `c2298a9ae60d32ecbfea96c19a051a7514fe78b6` returned first page:

- [CI 39726025605](https://github.com/rgomids/axiom/actions/runs/39726025605): completed/success.
- [Delivery metadata 39726025505](https://github.com/rgomids/axiom/actions/runs/39726025505): completed/success.
- [Delivery metadata 39726058026](https://github.com/rgomids/axiom/actions/runs/39726058026): completed/success.

These runs do not cover local Plan/e1 bytes. No CI dispatched or native acceptance
inferred. `technical_decisions[]`: [Plan §8](plan.md#8-technical_decisions-and-gates)
and [plan receipt](completion-receipt-plan.json) record PLAN-TD-01–03, owner
`executor-ai`, verification source/documentary only, `scope_impact:none`.
Plan r1 SHA-256 `2418a6814b03bbc3e7472a19dd6c3052de2eb0e66bb0cd4100ec65fbaed0dd24`; review this exact local checkpoint.

## Exact editorial Spec delta against accepted revision

```diff
diff --git a/docs/specifications/011-rendered-workflow-presentation/spec.md b/docs/specifications/011-rendered-workflow-presentation/spec.md
index 21705f3..ba83987 100644
--- a/docs/specifications/011-rendered-workflow-presentation/spec.md
+++ b/docs/specifications/011-rendered-workflow-presentation/spec.md
@@ -5 +5,3 @@
-**Proposed — revision 1, specify only, 2026-10-22.** Product owner:
+**Revision 1, editorial checkpoint e1 — PROPOSED reconciliation, 2026-10-22.**
+Behavioral baseline: accepted r1 at `c2298a9ae60d32ecbfea96c19a051a7514fe78b6`;
+this edited revision requires exact re-review and human reconfirmation. Product owner:
@@ -9 +11 @@ repository documentation; no unrelated provider content is imported.
-The verified operational gate is `READY_FOR_specify`, comment UUID
+The original Specify operational gate was `READY_FOR_specify`, comment UUID
@@ -16,4 +18,13 @@ before authoring. [Evidence](evidence-specify.md) records source checks and limi
-This local proposal has no commit or PR of its own. Exact content checkpoint is
-recorded separately in the completion receipt. Human acceptance of that exact
-revision remains required; documentary completion does not authorize Plan,
-implementation, external host sessions, merge, release or product acceptance.
+The original pre-publication checkpoint is recorded in the immutable historical
+[completion receipt](completion-receipt.json) and [Specify Evidence](evidence-specify.md).
+Its `pr_url: null`, uncommitted state and then-next gate describe that original
+observation, not present publication. Revision 1 was subsequently published in
+[PR #336](https://github.com/rgomids/axiom/pull/336) at
+`c2298a9ae60d32ecbfea96c19a051a7514fe78b6`, blob
+`21705f315bbaa2365ac1078ad1bc868f74725555`, and received
+[explicit human approval](https://github.com/rgomids/axiom/pull/336#issuecomment-6333507483)
+for VP-01–VP-15 and P-01–P-05 within that exact text's conditions.
+That approval authorizes preparing the Plan and editorial CR-001 correction;
+it does not transfer acceptance to these edited bytes or establish native rendering,
+implementation, merge, release or product acceptance. See [Plan Evidence](evidence-plan.md)
+for exact hashes, editorial diff and publication/authority limits.
@@ -29 +40,3 @@ Requirement origins are explicit:
-This whole Specification remains Proposed. No origin label accepts this revision.
+Origin and PROPOSED labels below preserve the r1 drafting vocabulary. Subsequent
+human approval applies only to the exact r1 baseline linked above; editorial e1
+remains pending reconfirmation. No requirement or refinement text is changed.
@@ -257,2 +270,3 @@ without editing acceptance history by implication.
-Material proposals P-01–P-05 await review of this exact Specification, not automatic
-promotion to MUST. Recommended: conversational detail/segmentation with textual
+At the original Specify checkpoint, P-01–P-05 awaited review. The subsequent
+approval above accepts them within the exact baseline conditions, without proving
+compatibility or adopting a renderer. Original recommendation: conversational detail/segmentation with textual
@@ -290,4 +304,6 @@ proven, and bounded research did not justify delegation overhead.
-**Next gate: `review_specification`.** Review exact local content checkpoint,
-origins and proposed refinements; authorize repository publication if desired.
-Then obtain explicit human acceptance of an observed revision. Do not start Plan
-or implementation here. Product ACs and native host acceptance remain unfulfilled.
+**Current next gate: `review_plan` plus exact editorial reconfirmation.** The
+original `review_specification` gate was followed by publication and the linked
+human approval of r1. [Plan](plan.md) is drafted against that accepted semantic
+baseline. Review the exact Plan and editorial e1 checkpoint before human approval;
+publication remains separately authorized. Do not start Tasks or implementation
+in this Plan attempt. Product ACs and native host acceptance remain unfulfilled.
```

## Publication continuation — 2026-10-22

After the original local Plan checkpoint/callback above, the user explicitly
authorized committing these four documentary artifacts and a normal push to the
existing branch/PR #336. No merge, Tasks or implementation is authorized. Prior
no-authority/uncommitted wording in Plan and receipt describes the original
checkpoint; it is retained as history, not a current absence-of-authority claim.
The original Linear event payload remains unchanged as a historical report.

Pre-commit staged whitespace check initially detected context-only spaces in
this embedded Spec diff. Replaced only its transcription with exact zero-context
`git diff --cached --unified=0` against the accepted SHA; Spec/Plan bytes unchanged.
Current Evidence hash is recorded in the plan receipt separately from the earlier
callback payload. Publication SHA and remote CI must be read back after push;
preparation alone does not prove publication or CI success.
