# AXM-38 Implementation Evidence — attempt-01

## Authority, sources and completion boundary

Implementation baseline: `4bf046f79d7d5006fea25ff2b5dc4dafc8a48fdc`, existing
[PR #336](https://github.com/rgomids/axiom/pull/336), branch
`axm-38-specify-rendered-workflow`. Initial local and remote HEAD matched; worktree
was clean. [Human Plan/Tasks and Implement authority](https://github.com/rgomids/axiom/pull/336#issuecomment-6348014628)
was read back with [Linear receipt](https://linear.app/rgomids/issue/AXM-38#comment-54eb0043).
The current user additionally authorizes scoped commit/push to this branch/PR and
one Linear implementation callback. No host-session, installed-root, credential,
cost, merge, release or human-acceptance grant was supplied. All Linear comments
were read (one complete page, hasNextPage=false); no implementation completion
event existed. Dispatch acceptance was not treated as Execution completion.

Approved [Spec e1](spec.md), [Plan r1](plan.md), [Tasks r1](tasks.md) and historical
receipts remain byte-for-byte unchanged. Their Proposed/NOT_STARTED/NOT_RUN text
belongs to historical checkpoints. This evidence supersedes only current progress
claims. It does not change requirements or mark product acceptance.
Final publication SHA, CI and Linear event/read-back belong to the completion
callback, which identifies this file at that exact GitHub revision. This file
makes no self-referential final commit/CI claim.

**Overall: PARTIAL.** T01 complete locally; T02 local guidance/offline evaluation
complete, end-to-end fidelity UNVERIFIED; T03 NOT_RUN / AUTHORIZATION_REQUIRED;
T04 local packaging prepared/verified, Task PARTIAL until T03. AC1 unmet.
No host is declared supported for graphics, and no screenshot/manual witness is
fabricated. No actual `teste-sdd` definition or installed inventory was observed.

## T01 — synthetic inputs, independent oracle and boundaries

[Fixture inventory](../../../internal/codexruntime/testdata/rendered-workflow/README.md)
and [hash/reference manifest](../../../internal/codexruntime/testdata/rendered-workflow/manifest.json)
identify four SYNTHETIC canonical-shaped show envelopes. Original Spec 007 r1/r2
definitions are wrapped using existing Go validation/JCS and original digests.
Actual producer has status/result/references/next/provenance/category and
workflowAuthoring; operation identity comes from the show invocation, not an
invented operation field. Project UUID/revision are synthetic. The new CLI test
compares wrappers with real show serialization and native terminal rendering.

[Offline tests](../../../internal/codexruntime/workflow_presentation_test.go) pin
hand-checked Stage order, scoped nodes, exact gates, independent roots/integrator
join, Runtime eligibility and Profile intent. Full typed declarations and full
fallback leaf comparison cover I/O, criteria, failure policy and configuration.
The expected graph literals do not call the projection. Missing/extra/reversed
edge, missing node, wrong order/reference/gate/constraint and omitted fallback fact
are negative controls. Input deltas check source-order changes, added declared
edge, eligibility/Profile/purpose/revision changes and stale selected references.
Equivalent object keys/whitespace preserve semantics. Selection compares all four
fields, including source, digest and absent selection. Schema-1 effort stays
historical; schema 2 remains without Workflow effort.

[Boundary tests](../../../internal/workflowdefinition/presentation_bounds_test.go)
construct valid sequential extra-predecessor, independent join, DAG concurrency 1,
single-Agent DAG and 32-Agent/496-edge DAG. They preserve transitive edges;
removing a redundant direct edge changes the digest while remaining domain-valid.
32 Stages, exact 1 MiB input and canonical snapshot, 16384-byte Unicode prose,
64-element I/O/criteria/validator lists, attempts 1/10 and timeout 1/86400 are tested
with over-bound refusals. Aggregate size is tested separately from individual maxima.
Failure/partial/unknown, absent/truncated payload, unknown schema, stale identity,
cycle/dangling/self/duplicate dependency and control prose refuse successful graph
without repairing input. Existing domain validator remains authoritative.

## T02 — guidance and synthetic specimens

Only active product skill changed:
[axiom-workflow](../../../internal/codexruntime/skills/axiom-workflow/SKILL.md).
Project skill was inspected; selection handoff is already consistent and unchanged.
No application/domain/CLI production code, schema, flag, exit code, terminal
renderer, persisted graph or authority was changed. Mapping is host-owned and
transient. No engine, dependency, service, upload or formatting-only model call.

Guidance requires rendered primary presentation on a qualified surface, three
levels, exact full identity and selection, ordered Stage progression distinct from
Agent dependencies and I/O provenance, declared gates/configuration, complete
independent text and bounded segmentation/continuation. All declared predecessors
remain reachable, including roots and cross-segment edges. Definition inspection
never implies Execution state, resolved Runtime/model/Profile, capability, consent,
validation success or human acceptance. Failed/uncertain results remain so.

Four static `*.fallback.md` specimens retain all canonical-shaped facts through
labelled source paths; independent Go traversal compares every leaf and empty
collection. This notation is test-only, not a product format or required JSON dump.
`custom-dag.candidate.mmd` is an UNRENDERED relationship candidate: manual oracle
checks Stage and Agent nodes/edges; full gate/configuration facts remain in text.
It does not adopt Mermaid or claim a complete rendered visual presentation.
Escape candidates check quotes/brackets/backticks/newlines, XML/HTML, directives,
Unicode combining/bidi/control and instruction-like labels as inert data.
Test-only escapes are not an implemented renderer. Actual host grammar safety and
AI response fidelity remain UNVERIFIED, never inferred from these tests or keywords.

## T03 — ready qualification protocol; NOT_RUN

No real host/runtime session, live CLI inspection, installed-root mutation,
external renderer or image generation was performed. Compatibility for Codex
Desktop, Codex CLI/IDE, Claude Code terminal/Desktop and any comparator remains
**UNVERIFIED** for AXM-38; functional text is the safe route for unknown capability.
No format is selected from documentation or exposed tools alone.

Before any session, require a bounded grant naming existing supported surface,
actions, synthetic/public inputs and cost/credential limits. Then record:

- Actual app/build/OS/settings, source/binary version and installed or candidate
  skill SHA-256 (checkout never implies installed skill).
- Each input hash/full reference; format/tool, actual response and visible capture
  hash, named manual witness/date, viewport/normal zoom and accessibility method.
- Workflow → Stage → Agent, default/custom, sequential/DAG and gates; compare
  actual output to independent oracle, preserving failures and zero missing facts.
- 32/32/dense/Unicode segmentation, cross-segment predecessors and independent
  text/keyboard reading; any inaccessible or clipped scope remains unqualified.
- Forced unknown/no renderer, literal fence, link-only, clipped/empty graphic and
  renderer error. Retain fallback and limitations, never call these graphical PASS.
- Recheck final skill/input/build/settings tuple. Drift invalidates qualification.

E1/E7 and all empirical AC portions are NOT_RUN. A future visible result plus
manual witness on at least one existing supported host is necessary for AC1.
An authorization gap does not prove the product requirement impossible and does
not require a material architecture/product/design decision now.

## T04 — shared source, isolated installation and rollback

[Shared manifest/history](../../../internal/codexruntime/manifest.go) appends the
exact replaced four-skill revision. Historical text is preserved in
[pre-rendered-workflow](../../../internal/codexruntime/testdata/pre-rendered-workflow/PROVENANCE.md).
Current four skills retain skill-set version 3 / binary compatibility 3; only
Workflow text differs. Codex and Claude consume the same embedded bytes and
history through existing Runtime installers. No Runtime-only history or manual
installed receipt edits were introduced; copied history is test-only.

[Exact baseline upgrade tests](../../../internal/codexruntime/rendered_workflow_upgrade_test.go)
cover both Runtime roots: owned upgrade, repeat no-op, user modification/unowned
refusal, interrupted upgrade, mixed inventory rejection and resumed convergence.
Existing clean-install, conflict/retirement tests remain active. Existing
`TestGuidedConfigurationSyntheticPackageRollbackBothHosts` verifies compatible
package restoration and stale rollback refusal through supported upgrade sessions;
`TestOutputSeparationCoordinatedUpgradeAndPairRestoration` checks mixed pairs,
interruption and compatible-pair restoration. These are temporary isolated roots,
not real installations or native graphical tests. No operator data is deleted.
CLI serialization/native text tests and existing
`TestActiveWorkflowShowListSelectedNoneAndNoWrites` cover read-only separation.

Rollout remains: offline review → separately authorized exact-host qualification →
review final bytes → separately authorized live install through supported mechanism.
Rollback uses supported known compatible binary/skill pair; refuse stale/unsupported
replacement rather than overwrite operator edits. Complete text mitigates graphical
failure but does not satisfy graphical acceptance. T04 remains PARTIAL pending T03.

## Acceptance and Evidence matrix

All PASS below mean deterministic local evidence only. Every real AI/host response,
rendering, manual visual/accessibility portion remains NOT_RUN / UNVERIFIED.

| Task / AC | Evidence | Local result and remaining boundary |
| --- | --- | --- |
| T01 / AC2,3,5,6 | E2/E3/E4/E6/E8/E9/E10 | PASS fixture/oracle/negative/bounds tests; no host claim |
| T02 / AC1 contribution | E1 | PASS guidance only; actual rendered primary NOT_RUN |
| T02 / AC2 | E2/E3/E9 | PASS declared graph sets/bounds/specimens; real legibility NOT_RUN |
| T02 / AC3 | E4 | PASS full identity/selection, effort and declared eligibility; real response NOT_RUN |
| T02 / AC4 | E5/E6 | PASS synthetic complete text and failure vocabulary; actual forced host fallback NOT_RUN |
| T02 / AC5 | E6/E8 | PASS deterministic semantic tests and source preservation; E7 real output NOT_RUN |
| T02 / AC6 | E8/E10 | PASS no production state change, malformed/unsafe/refusal tests; live host behavior NOT_RUN |
| T03 / AC1–6 empirical portions | E1/E7 plus matching E2–E10 cases | NOT_RUN / AUTHORIZATION_REQUIRED; AC1 unmet |
| T04 / AC2–6 local integration | E8/E10 | Isolated packaging/CLI checks; no installed/native qualification |
| T04 / AC1 final acceptance | E1/E7 | NOT_RUN; depends on completed T03 |

## Technical decisions and review

| ID / category | Choice / alternative / rationale / verification |
| --- | --- |
| IMPL-TD-01 / TECHNICAL_LOCAL | Update existing shared Workflow skill; no Go product mapper or persisted view. ADR-0030 assigns conversation to host; offline mapper/specimens remain test-only. Verified production diff restricted to skill/history. |
| IMPL-TD-02 / TECHNICAL_LOCAL | Existing Go decoder supplies fixture JCS/digests; manual graph literals and independently authored specimens avoid expected=actual projection. Alternative duplicate canonicalizer rejected. Fixture/CLI/boundary tests verify. |
| IMPL-TD-03 / TECHNICAL_LOCAL | Retain shared compatibility 3; append replaced four-skill history. Prose adds no binary/wire incompatibility. Alternative version bump unnecessary; both installer consumers tested. |
| IMPL-TD-04 / TECHNICAL_LOCAL | Format-neutral guidance and full text until exact-host qualification. Candidate Mermaid stays synthetic/source-only. Alternative assume fence renders rejected; no support expansion. |
| IMPL-TD-05 / TECHNICAL_LOCAL | Execute directly: strict lower-capability hierarchy not demonstrated. Deterministic checks provide bounded verification; no delegation/model claim. |

Scope impact: none; decision maker executor-ai, stage implement, attempt-01.
Rollback: reviewed source reversal plus supported compatible-pair restoration;
no real installation was changed. ADR-0030, ADR-0024 and Spec 007 remain coherent;
no material architecture/product/design change or new ADR is required.
Self-review covers source trust, identity drift, graph independence, inert labels,
no remote effects, secret scan, preserved history and isolated installer ownership.
Independent review of this PR remains pending; no self-approval is inferred.
Deferred finding IMPL-DF-01 (TECH_DEBT, detected during implement): existing
`docs/commands.md:374` links `../.github/workflows/poc-verification.yml`, absent
already at baseline `4bf046f`. New owning-document links pass; the pre-existing
historical POC reference is unrelated to Workflow presentation and was not changed.
Evidence: [baseline reference](https://github.com/rgomids/axiom/blob/4bf046f79d7d5006fea25ff2b5dc4dafc8a48fdc/docs/commands.md#L374).
Status DEFERRED_NOT_IMPLEMENTED; low-impact documentation follow-up for human
pre-merge consideration, not an added requirement or a new work item.

## Executed checks

Validation results are appended below after execution; CI is reported separately
for the final published SHA in the Linear callback. Initial focused tests found
and corrected test-fixture assumptions (custom per-Agent Runtime eligibility,
removal of a non-redundant dense edge, empty-array specimen escaping) and legacy
wording assertions; these were test/evaluation defects, not product behavior fixes.

- `go test ./internal/workflowdefinition ./internal/codexruntime ./internal/cli -run 'Test(RenderedWorkflow|Presentation|WorkflowShowSynthetic)' -count=1`: PASS.
- `go test ./internal/codexruntime -run 'Test(RenderedWorkflow|OperatorConversationalPresentationContracts|EveryPublishedSharedRevision)' -count=1`: PASS.
- `go vet ./...`, `go build ./...`, `./scripts/check-go-quality.sh format`: PASS.
- `./scripts/validate-repository.sh .`: PASS (including ADR/bootstrap/security checks).
- `bash scripts/test-docs-offline.sh`: local macos-arm64 artifact/install/refusal checks PASS; script qualification explicitly PARTIAL, network/source denial and native Runtime observation NOT_RUN. Temporary development archive, not release/publication or real install.
- `./scripts/check-sensitive-files.sh --staged .` and `gitleaks protect --staged --redact --no-banner`: PASS.
- Bundled Codex Python validator dependency unavailable; script warns and retains Go validation. Standalone Staticcheck unavailable locally; no local PASS claim.

Source Workflow skill SHA-256: `8b47aa65611d28395eab3695ce4ca90773e541800f72d4189838565268a7f235`.
Current shared manifest SHA-256: `508f3f1b1a31a403d5deba227cffef981bc67061812394420948290a78c03216`.
Both temporary-installed Runtime consumers are tested against embedded skill bytes
and digest. This is not an installed production inventory or native host evidence.

- New local Markdown links and byte equality of approved Spec/Plan/Tasks plus all historical receipts: PASS. Baseline-wide link check exposed only the unrelated POC reference recorded above; no claim that all historical links resolve.

- Initial `go test ./...`: FAIL only in unchanged `internal/executiongraph` test `TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation`, whose callback sleeps 100 ms. CLI black-box package passed (487.415 s). Retained failure: actual process result was stopped/unknown instead of exit 7 with rejection text. No executiongraph source/test was changed. Focused reproduction and final full rerun are reported below; timing sensitivity is not silently waived.
- Fixture provenance was corrected from synthetic `dev` to producer-valid `development`; CLI test now compares the entire envelope, including exact provenance, against the real serializer. Final focused CLI/semantic tests PASS.

- `go test ./internal/executiongraph -run TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation -count=10`: FAIL (4/10) in candidate; same command against an isolated `git archive` of unchanged baseline `4bf046f`: FAIL (2/10). Temporary baseline copy removed after reproduction; no branch or production state changed.
- Final `go test ./...`: PASS; unchanged cmd/lingo black-box results reused from the completed successful package run, remaining packages passed/cached. Earlier failure remains documented, not retroactively converted to PASS.
- Final `bash scripts/test-codex-skills.sh`: PASS; same Python-validator warning, Go contract checks passed for codexruntime/runtimebootstrap/cmd/lingo/cli.
- `go test ./internal/workflowdefinition ./internal/codexruntime`: PASS.

Deferred finding IMPL-DF-02 (POTENTIAL_BUG, detected during implement): reproducible
baseline timing-sensitive process-result test above. Source
[baseline process_test.go:114–127](https://github.com/rgomids/axiom/blob/4bf046f79d7d5006fea25ff2b5dc4dafc8a48fdc/internal/executiongraph/process_test.go#L114)
and its 100 ms callback window predate this PR. Impact: intermittent broad-suite
failure under this macOS environment; exact cause not established. Status
DEFERRED_NOT_IMPLEMENTED; recommend separate investigation, no executiongraph change
or added AXM-38 requirement. Final suite success does not erase this limitation.
