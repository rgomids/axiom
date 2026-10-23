# AXM-39 — Specify Evidence, attempt-01

Date: 2026-10-23. Base/source HEAD:
`921507c27c5b1ec5468508a6d48ce0e080e90a42`, initially clean detached checkout,
equal to remote main by `git ls-remote`. Scope: Specification and documentary
verification only; no installed Runtime, inference, code or skill mutation.

## Source checks and provenance

- Authenticated Linear: AXM-39 card/relations and all six comments read;
  `hasNextPage: false`. Approved baseline revision
  `card-description-2026-10-19T05:08:35.105Z`; decision comment
  `70d285fc-718a-4ad8-8ca2-899e1a629974`, intake
  `a9e0c096-b459-4471-8d64-6ce03d1ea79a`, readiness
  `054cdea2-3a21-4856-aa36-ba04807e51f8`. Public behavioral intent import
  explicitly authorized by the task; no private transcript imported.
- Current AXM-31/32/33/36/37/38 cards read: operationally Done. AXM-12 read:
  In Progress, real native/E2E and final acceptance remain separately owned.
  No operational status, relation, assignee or acceptance changed.
- `gh pr list --state all --search AXM-39` found incidental mentions only in
  #335/#336/#318; open-PR inventory found only unrelated #317. Remote branch
  query `*axm-39*` returned none. No canonical AXM-39 Spec/PR found before work.
- Repository 007, accepted runtime-admission amendment, ADR-0030/0032,
  Stage/Agent semantics and resolution boundaries inspected at source HEAD;
  008/009/010/011 index and relevant presentation precedents checked. Original
  Proposed headers in accepted historical artifacts do not revoke later exact
  approval; acceptance provenance remains in their index/annotations.
- Runtime discrepancy: Intake observed AXM-36 Awaiting Release. Current Linear
  says Done; [PR #334](https://github.com/rgomids/axiom/pull/334) is merged at
  `0f1073ac6a3af615d48b651340d8fe895827080a`. [Versioned implementation
  Evidence](../007-configurable-workflows/evidence-axm36-implementation.md)
  reports completed synthetic coverage, not vendor execution. Latest published
  release observed: [v0.21.1](https://github.com/rgomids/axiom/releases/tag/v0.21.1),
  targeting source HEAD. No particular host installation was inspected. An
  attempted lookup of v0.21.0 returned “release not found”; no claim about that
  release is used. Dependency delivery/readiness must be rechecked before
  implementation/native qualification, not frozen to the old Intake snapshot.
  GitHub compare confirms source HEAD is eight commits ahead of the AXM-36
  merge, establishing inclusion in the observed release target, not native
  operation or installed-host qualification.
- Specifications index contains stale progress summaries for predecessors;
  later versioned Evidence/approval records and current provider facts are
  distinguished rather than rewriting accepted history in this narrow change.

## technical_decisions[]

| ID / category | Choice and alternative | Rationale / GitHub Evidence | Scope impact |
| --- | --- | --- | --- |
| TD-01 / TECHNICAL_LOCAL | New 012 presentation Specification; instead of changing 007's domain contract | Separate presentation precedents [008](../008-output-separation/spec.md), [009](../009-guided-project-runtime-configuration/spec.md), [011](../011-rendered-workflow-presentation/spec.md); 007 remains owner | none |
| TD-02 / TECHNICAL_LOCAL | Full ID only for collision/absence/operational need; no auxiliary abbreviation algorithm | Approved fallback; [P1–P2](spec.md#project-labels-and-exact-identity) need no new prefix registry and preserve exact consent targets | none |
| TD-03 / TECHNICAL_LOCAL | Semantic content cases and expected turns in Markdown; no durable checker/schema | [Examples](examples.md) specify test oracles within documentation authority; implementation chooses existing harness later | none |
| TD-04 / TECHNICAL_LOCAL | Execute directly, no subagents | Bounded documentary task; model capability hierarchy not verified; no delegation needed | none |

All choices are reversible document organization/test-detail choices inside the
approved baseline. No new product/architecture/design decision or ADR accepted.

## Documentary verification and limitations

Product C01–C16, native
Codex/Claude conversations, installed compatibility, real inference, Provider,
AXM-12 E2E and human acceptance: **NOT_RUN**. No canonical Execution identity
or gate reconstructed. No CI result is claimed from local checks.

Actual local results:

- `./scripts/validate-repository.sh .` — PASS, including documentation/bootstrap,
  ADR governance, packaging/automation and sensitive-file checks it invokes.
- Python standard-library document assertions — PASS: 169 relative references
  across new documents/index resolve to existing paths; six AC rows and sixteen
  documented cases present; required thematic sections present. This checks
  document structure, not presentation behavior or external URL availability.
- `git diff --check` — PASS; source/diff review restricted to three new Markdown
  files plus the Specifications index. No flags, canonical contract or runtime
  files changed.
- `gitleaks detect --source docs/specifications/012-conversational-project-stage-presentation --no-git --redact`
  — PASS, no leaks.
- `./scripts/check-sensitive-files.sh --staged .` — PASS before commit.

External references: Linear card/comment existence verified through current
issue/comment reads; GitHub PR/release/compare read through authenticated CLI.
Local canonical links are checked as paths, not exhaustive heading/HTTP crawl.
No Go build/tests, behavioral regression runner, native host sessions or CI
were run for this documentation-only change.

Manual source/AC review: six AC mapped in Spec; every material rule has CARD or
ACCEPTED provenance; no optional proposal promoted to a new MUST/AC. Summary
cannot hide blockers; allowed/declared/resolved/observed and definition/Execution
remain separate; exact identity/digest/authority and RA-1–4 safety preserved.

Deferred finding DF-01 (TECH_DEBT, DEFERRED_NOT_IMPLEMENTED): predecessor status
summaries in [Specifications index](../README.md) still describe incomplete
AXM-36/37/38 work despite later Evidence/current operational delivery. Optional
historical-status reconciliation is outside this Specification scope; surface
at pre-merge with owning Evidence, without changing any accepted historical
bytes or inferring native acceptance.

Next gate: readiness evaluation for Clarify if a material gap emerges, otherwise
Plan. This executor stops at Specify. No continuation callback tool is proven
available; publish same-AXM completion event and preserve
`AWAITING_EXECUTION_REPORT` for workflow reconciliation, without promising a
background watcher or requesting ceremonial continue.
