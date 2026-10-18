# AXM-31 — Implementation Evidence

Classification: **synthetic**, unless explicitly stated otherwise. This record
is not independent capability proof, native conversation Evidence, human Review
or AXM/MVP acceptance. Contract: [accepted Tasks](tasks-axm31-operator-autonomy.md),
input revision `10cb320df7ebbf727590acee9f091c51b54a2853`.

## Authority and preflight

The human IMPLEMENT T01–T08 request authorizes scoped implementation, tests,
Evidence, commits and updating the existing branch/PR. Tasks acceptance was
revalidated through the [Linear decision](https://linear.app/rgomids/issue/AXM-31/revisar-autonomia-escopo-e-responsabilidade-da-ia-operadora-no-axiom#comment-56d8406a)
and [PR comment](https://github.com/rgomids/axiom/pull/325#issuecomment-6243413855),
not a native GitHub review. The accepted revision equals local HEAD, remote
branch and [Draft PR #325](https://github.com/rgomids/axiom/pull/325)'s head at
preflight. No Task drift; working tree initially clean. Only other open PR was
#317, unrelated. No new PR, status, relation or acceptance change authorized.

Linear read returned In Progress, no blockedBy, blocks AXM-12 and related
AXM-8/AXM-9. Comments had no remaining page. No accessible record identified a
concurrent implementation. `axiom` is absent from PATH; the effective state root,
resolved from `stateRoot` and `local.NativeStateRoot`, is absent on this host.
No store was created and no product command used to reconstruct state.
Canonical Execution/current gate and concurrency on other hosts remain unknown;
Linear/PR/filesystem do not establish their absence. No Execution transition.

No delegation: a lower model profile could not be established from available
runtime evidence. The Orchestrator performs bounded sequential work and review.
No actual Provider/Runtime inference, privileged push, merge, release or human
acceptance is covered. Synthetic child/probe fixtures remain synthetic.

## Task matrix

| Task | AO | Scenarios | Tests / Evidence | Result |
|---|---|---|---|---|
| T01 | 03,04,05,08 | P3/N3,P5/N5,N7 | `TestOperatorStageDiagnostic*`, `TestOperatorDiagnostic*`, `TestOperatorPartialReferencesComeOnlyFromTypedResults`; JSON/Markdown golden and installed single/Graph regressions | PASS (synthetic) |
| T02 | 02,03,05,12 | P2/N2,P8/N10,N7 | `TestOperatorProjectEditDiagnosticUsesRealReadOnlyFailure`, `TestOperatorWorkflowDiagnosticPreservesPartialAndStale`, existing edit publication/recovery and authoring suites | PASS (synthetic) |
| T03 | 01,02,05,10,11,12 | P1/N1,P2/N2,P7/N9,P8/N10,N8 | `TestOperatorContinuityInstructionContracts`, canonical routing/parser/catalog suites | PASS (instruction contract) |
| T04 | 02,06,07,08,09,10 | P2/N2,P4/N4,P5/N5,P6/N6,N7,N8 | `TestOperatorBudgetAndHandoffInstructionContracts`; programmed budgets/handoff tests in T07 | PASS (instruction contract) |
| T05 | 03,05,10,13 | P2/N2,P3/N3,P8/N10,N8 | `TestOperatorAuxiliaryPresentationContracts`, `TestOperatorAuxiliaryFallbackMatchesCanonicalOrderAndGates`, `TestOperatorStageDiagnosticUsesCanonicalEmitter` | PASS (synthetic) |
| T06 | 01,02,03,05,10,11,12,13 | P1/N1,P2/N2,P7/N9,P8/N10,N8 | `TestOperatorSyntheticDualEntryProjectWorkflowJourney`, `TestOperatorSyntheticDivergentReadBackStopsWithoutReapply`; existing scratch/authoring/edit suites | PASS (synthetic) |
| T07 | 03,04,05,06,07,08,09,10 | P3/N3,P4/N4,P5/N5,P6/N6,N7,N8 | `TestOperatorSyntheticBudgetsFakeClock`, `TestOperatorSyntheticHandoffTwoSessionsAndDrift`, `TestOperatorSyntheticNoReplayOnTimeoutPartialAndUntrustedData`; installed single/Graph and recovery/proof suites | PASS (synthetic) |
| T08 | 01–13 | P1–P8/N1–N10 | `TestOperatorAutonomyUpgradeBothRuntimeRoots`, install/receipt/history suites; integrated checks below | PASS (synthetic/local; CI publication record in PR #325) |

## S1 observations

- Optional output-only `diagnostic.schemaVersion=1` is appended after `stage`;
  existing payload topology, category and completion fields are preserved.
  No persistence codec, status classifier or domain service is changed.
- Enrichment takes typed errors and existing results only. It has no I/O ports.
  Observations, hypotheses, missing prerequisites and uncertain effects remain
  separate; unknown capability/authority/confinement never becomes proof.
  Missing identity/revision is explicit. At most three ordered options; an
  unavailable probe has no executable operation. Unknown producer support is
  not reported as proven unavailable or as a guaranteed probe resolution.
- Output enrichment is bounded to 16 KiB within the existing event limit;
  invalid/oversize enrichment is omitted, preserving the canonical refusal.
- Regression exposed existing partial completion construction with no confirmed
  reference. Output composition now consumes only existing typed receipt or
  authority references, validated as digests. No status or persisted fact is
  rewritten; no reference is extracted from arbitrary error text/maps.
- `TestOperator*` focused command: PASS after correcting test invocation to the
  existing Run(JSON)/RunInteractive(Markdown) contract. Initial test FAIL exposed
  the partial-reference defect above; it is not reported as a final PASS.

## External verification

Native Codex/Claude conversations, host graphical support, LLM budget compliance,
independent behavioral capability proof, live delivery/rework and MVP acceptance:
**EXTERNAL / NOT_RUN**. Owners remain AXM-8/#276 (dispatch/proof), AXM-9/#277
(delivery/rework) and AXM-12/#278 (native/live journey and MVP acceptance).

## Integration scenario matrix

All tests below use programmed local inputs, synthetic consent and isolated
stores. A fixture's decision to ask, stop or hand off is explicit test code;
there is no semantic resolver, live LLM evaluation or new product enforcement.
The fixture never labels a fake child, fake behavioral probe or recorded
`proven` sample as independent operational proof.

| Scenario | Executable verification | Observed boundary / result |
|---|---|---|
| P1/N1 | `TestOperatorSyntheticDualEntryProjectWorkflowJourney`, `TestEditSelectorFailuresNeverFallBackToCreate`, `TestCanonicalReadOnlyCatalogDispatch` | Two entry scripts share operation traces; reads preserve tree bytes; multiple Projects require scripted exact identity; ambiguous/unknown selectors do not create a Project. PASS |
| P2/N2 | `TestWorkflowScratchDraftAuthorityJourney303`, `TestEditReplayNoOpStaleAndDriftFailWithZeroWrites`, `TestOperatorSyntheticDivergentReadBackStopsWithoutReapply` | Previous draft/revision retained; changed draft, selector or reviewed digest refuses effects; divergent canonical association read-back stops without reapply. PASS |
| P3/N3 | `TestOperatorStageDiagnosticTypedRefusals`, `TestCheckBindingDrift`, `TestCheckExpiredAndWrongVariant`, `TestRequiredNonProvenBlocks`, installed single/Graph journeys | Missing/expired/drift/non-proven fixtures never grant dispatch. A coarse error is explicitly marked insufficient to distinguish expiry/absence/producer support; no additional probe/read is performed by enrichment. PASS |
| P4/N4 | `TestOperatorSyntheticBudgetsFakeClock` | Initial-call baseline, two repetitions, forbidden third, 60-second inclusive boundary, expired deadline and lower limit; two progressive remediations and immediate stop after first no-progress; joint action consumes both counters. PASS |
| P5/N5 | `TestReconcileStaleAndUnknownAppendsFactsOnly`, `TestReconcileRefusals`, `TestSingleRetryRefusesUnreconciledUnknownAndStalePreview`, installed single crash/adversarial and Graph retry journeys | Original attempt/status/process identity remains; fact append only; live/uncertain/foreign-lease/drift refusals; own retry authority, disposition and attempt limits remain required. PASS |
| P6/N6 | `TestOperatorSyntheticHandoffTwoSessionsAndDrift` | JSON reference-only checkpoint carries original deadline and both counters into second session; revision alone does not reset; missing/stale checkpoint, target/binding/attempt drift and absent authority cause zero scripted effects. PASS |
| N7 | `TestOperatorStageDiagnosticUnknownAndPartialKeepPayload`, `TestOperatorPartialReferencesComeOnlyFromTypedResults`, `TestIntegrateUnknownEffectBlocksRerun`, `TestIntegrateProjectionFailureIsPartialAndRepairable`, existing edit partial recovery | Payload bytes/confirmed references retained; prior confirmed effects are not erased; timeout/ack-lost/partial/unknown never triggers blind replay. Live delivery/rework stays AXM-9/AXM-12. PASS (synthetic); operational delivery EXTERNAL / NOT_RUN |
| N8 | `TestOperatorStageDiagnosticTypedRefusals`, `TestOperatorSyntheticNoReplayOnTimeoutPartialAndUntrustedData`, skill instruction contracts and existing adversarial suites | Joined error canary never enters diagnostic; programmed error/diagnostic/definition/child/Provider canaries do not enter closed checkpoint or invoke arbitrary host commands, proof, grants or acceptance; previous effects retained. PASS (synthetic protocol), no native LLM claim |
| P7/N9 | Dual-entry journey; `TestExplicitOperationRoutingIsDeterministic`, `TestUnsupportedDomainOperationsFailSafely`, inspect/catalog/parser suites | Exact explicit operation is preserved; scripted next intent continues; inspect is read-only and ends; absent skill/operation/argument does not fabricate an executable operation. PASS |
| P8/N10 | Dual-entry journey, scratch/activeWorkflow/authoring suites, auxiliary fallback fixture | Canonical re-read, Project association preview/apply/read-back, built-in byte preservation, human-chosen editable copy, separate selection; static fallback independently checks all stage IDs/order/input dependencies/human gates. PASS (synthetic); native graphics EXTERNAL / NOT_RUN |

## Compatibility, review and security

Output-only DTO lives in `internal/cli`; application mappers in
`cmd/lingo/operator_diagnostic.go` have no I/O, probe, dispatch or persistence
ports. `completion.Result`, its classifier, existing domain reports, stores,
Execution formats and status/attempt/reconciliation semantics are unchanged.
No additional dependencies, CLI/skill routes, renderer or durable automation.
Installed inventory remains four skills; three instruction files changed.

Known parser/consumer inspection found JSON unmarshalling and the generic ordered
Markdown renderer at this output boundary, without closed completion decoders.
Persisted strict codecs were not modified. Unknown external consumers are not
claimed tested. `diagnostic` is optional; invalid/oversize enrichment falls back
to the same refusal. New JSON/Markdown golden records only synthetic identifiers.

Review corrections before final validation:

- Restored validation of `completion.Result` in the additive generic emitter.
- Preserved pre-existing canonical treatment of capability stale errors rather
  than introducing a new status/category mapping.
- Bounded operation identity to real stage/coordination operations; excluded
  delivery/unsupported operations from enrichment even though they share error
  helpers. Delivery business logic remains untouched.
- Partial references come only from existing typed Graph/single publication,
  retry and integration results; arbitrary map/text references are refused.
- Built-in edit via the real CLI refuses `prior_revision_required`; exact copy
  replay may be `unchanged`. Collision handling is a programmed intent decision
  before mutation, not a fabricated domain error.

The Orchestrator reviewed specification/architecture/verification/security lanes;
no unresolved blocking finding identified within this bounded scope. This is
self-review, not independent Review or human acceptance. Final integrated
validation remains the gate below. Accepted Spec/Plan/ADR bytes are preserved;
the Tasks artifact only adds the factual acceptance annotation.

## Checks and artifact identities

| Check actually executed | Result |
|---|---|
| `go test ./internal/cli ./cmd/lingo -count=1` (T01) | PASS |
| `go test ./cmd/lingo ./internal/cli ./internal/projectapp -count=1` (S1) | PASS |
| `go test ./internal/cli ./internal/codexruntime -count=1` (T03/T04/T05) | PASS; T05 output-contract expectation was updated for the additive payload after an initial FAIL |
| T06/T07 focused commands recorded in task/scenario rows | PASS |
| `go test ./internal/codexruntime ./internal/install ./internal/cli -count=1` (T08 focused) | PASS |
| `go test ./... -count=1` (first integrated run) | PASS, including installed synthetic stage journeys; final review adjustments require the final run below |
| `go test ./... -count=1` (final source) | PASS; final run exit 0, including installed single/Graph journeys |
| `./scripts/validate-repository.sh .` | PASS; validator's own Runtime scenarios remain SKIP/UNVERIFIED |
| `go vet ./...`, `go build ./...`, `go mod tidy -diff`, `go mod verify` | PASS |
| `git diff --check` | PASS |
| `./scripts/check-sensitive-files.sh --staged .`, staged diff review | PASS; 31 scoped files reviewed |
| `gitleaks`, local `staticcheck` | SKIP: not installed; no automatic installation |
| GitHub CI of final pushed SHA | Exact SHA, run URLs and result are recorded in the publication/validation block of [PR #325](https://github.com/rgomids/axiom/pull/325); no prior-head CI is attributed to this implementation |

Development binary built with `go build -o <temporary artifact> ./cmd/lingo`,
without executing product operations: SHA-256
`3b4dd428715bb0a3bc775620af84b65c2a2b76f895394a288f4f56cb61470db0`.
This is a development build, not a release artifact or installed capability.

Current shared skill manifest:
`238503af1bf9bdbfb3cc6c054f7641f3992f4aac0cace989126ac6c7670bf3b7`.

| Skill | SHA-256 |
|---|---|
| axiom-project | `d676740525a254105521e1bf426284cc0a457830183da6ed6f8057dc7a8dfc26` |
| axiom-workflow | `c7c5dec9abcf5dd133036ef97c69203b52ca7d6aee1aa50dbf9d3616680f4773` |
| axiom-work-item | `965df8b3435c4afa3fd674ceae86ce98ecae405440a00854478610c74ef18c39` |

The replaced revision was appended to shared history; historical receipt fixtures
and manifest pins stay immutable. Both roots upgrade from the exact old bytes
(hash checked), converge to current bytes/receipts, reinstall without effects,
and refuse modified/foreign content. Existing installer interruption/resume and
published-release upgrade suites remain in the integrated regression.

## Local completion and publication handoff

- objective: finish T01–T08 under the accepted contracts on PR #325.
- state: local implementation and validation complete; publication/CI record
  belongs to the canonical PR and is verified after the branch update.
- completed units: T01–T08 local/synthetic verification and repository checks.
- publication handoff: scoped commit and ordinary authorized branch update, then
  record exact-head CI observation in PR #325. If CI fails, fix only the bounded
  regression, repeat affected checks and reobserve the new SHA before Review.
- authority: implementation/tests/Evidence/commit/same-branch PR update granted;
  real inference, product Provider effects, privileged push, merge, release,
  product acceptance and blocker removal remain outside scope.
- next gate after all checks: Review; never infer AXM-31 or MVP acceptance.

The final local source regression completed with exit 0. The publication record
in PR #325 supplies commit/head and CI URLs without a self-referential commit
hash inside this versioned file. No operational counters, grants or Execution
state are created by this handoff.
