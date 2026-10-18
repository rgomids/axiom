# AXM-32 — R1 coverage and ADR clarification

## Authority and inspection boundary

Documentary clarify, 2026-10-18. Source revision:
`18d29a6c5446357defb2563483cfac180dcc4043` (all code references below).
The [human Spec acceptance](https://github.com/rgomids/axiom/pull/328#issuecomment-6246060142)
explicitly accepts [Specification 008](spec.md) at that SHA, including R1–R4 and
AC1–AC9. Its historical Proposed wording is preserved byte-for-byte, not treated
as a reason to request Spec acceptance again. The matching Linear acceptance and
[clarify gate](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-6eaf4304)
were read. Neither accepts ADR-0030 or advances a canonical Execution.

**Conclusion: NEEDS_HUMAN_DECISION; not READY_FOR_plan.** The inventory below
classifies the required facts, but does not establish universal R1 coverage.
G1 proves loss of a material typed category/uncertainty boundary; G2 proves
pre-serialization reduction of transfer effects. Existing JSON cannot simply be
assumed lossless. No envelope, field, enum, version or production change is
approved here. D1–D3 and the accepted Spec remain unchanged.

`covered` means the named fact survives the named source-to-wire path, established
by code/DTO inspection, not that every command or native host satisfies R1.
`gap_confirmed` means an identified producer fact is lost or reduced before wire
emission. `unverified` means evidence is insufficient for the stated scenario.
Tests cited below were **read, not run**. Markdown fixtures are not JSON proof.
No real personal Project JSON was available or reproduced.

## Producer and emission inventory

All paths in this table are under `internal/cli/` unless explicitly qualified.
Most completion events embed `completionEvent`: `status: completion.Status`,
`result: string`, optional `references: []string`, `next/details: string`, and
`provenance: provenanceEvent`. These are not a universal operation envelope.
`cli.Result` is an application-to-CLI carrier, not itself the wire DTO.

| Family / invocation context | Concrete producer → serializer / typed payload | Concrete consumers and compatibility limit |
|---|---|---|
| Common completions / parser failures / version | [completion.go](../../../internal/cli/completion.go) `renderCompletionJSON`, `emitDiagnosticCompletion`; [cli.go](../../../internal/cli/cli.go) `emitParserFailure`, `emitResponse`; [completion.Result](../../../internal/completion/completion.go) | `WriteCompletion`, CLI JSON callers and `scripts/smoke-prepared-artifact.py::run`; no root operation/category in common DTO. Invocation must be retained independently. |
| Project show/list, bootstrap and skill status/install | `cmd/lingo/{main,project_lifecycle}.go` → [completion.go](../../../internal/cli/completion.go) `emitProjectCompletion`, `emitProjectListCompletion`, `emitBootstrapCompletion`, `emitRuntimeCompletion`; `ProjectView`, `ProjectListView`, `BootstrapView`, `RuntimeView` in `cli.go` | `axiom-project`, `axiom-runtime`; runtime tests and shared installation history. IDs, Repository keys, availability, skill hashes/conflicts/receipt survive; unavailable payload is not an empty-success observation. |
| Project validation, preflight and admission | [cmd/lingo/project_lifecycle.go](../../../cmd/lingo/project_lifecycle.go), `cmd/lingo/main.go::Validate` → [readiness.go](../../../internal/cli/readiness.go) `readinessCompletionEvent`: `ReadinessReport`, `OperationReadiness`, `AdmissionDecision`, `ProjectStateView` | `axiom-project` and read-only Work Item prerequisites; `cmd/lingo/bootstrap_readiness_test.go::runEvent` decodes JSON. Operation blockers are scoped, separate from completion status. |
| Configure create/edit | `cmd/lingo/main.go::Configure/projectEdit` → `completion.go::emitSetupCompletion/emitEditCompletion`; [SetupPreview](../../../internal/projectapp/setup.go), [EditPreview](../../../internal/projectapp/edit.go) | `axiom-project`, prepared artifact smoke (setup digest). Edit diagnostics use `presentDiagnosticEvent`; preview effects are proposed effects, not confirmation of apply. |
| Operational Project/Integration lifecycle | `cmd/lingo/project_lifecycle.go`, application [operational.go](../../../internal/projectapp/operational.go) → [cli/operational.go](../../../internal/cli/operational.go) (`category: string`, `operational: *OperationalPreview`); [integration.go](../../../internal/cli/integration.go) (`category`, `*IntegrationReport`) | `axiom-project`, lifecycle tests. Preview, no-op and admitted mutation retain distinct application semantics. Provider and Transport remain distinct. |
| Work Item create/select/show and lifecycle | `cmd/lingo/main.go::workItemResult` and `cmd/lingo/work_item_lifecycle.go` → `completion.go::emitWorkItemCompletion` (`DraftPreview`, `SelectionPreview`, `WorkItemView`, `[]Question`) or [work_item_lifecycle.go](../../../internal/cli/work_item_lifecycle.go) (`category`, `ChangePreview`, `*[]WorkItemView`) | `axiom-work-item`; `internal/workitem/{service,lifecycle}_test.go`, CLI lifecycle tests. Routing depends on payload presence, not only verb; G1. Empty list is distinguished from omitted list. |
| Execution lifecycle/list/projection | `cmd/lingo/main.go::workflowResult` → `completion.go::emitWorkflowCompletion` (`WorkflowView`, `ProjectionPreview`); [execution_lifecycle.go](../../../internal/cli/execution_lifecycle.go) (`[]ExecutionSummaryView`) | `axiom-work-item`; `cmd/lingo/main_test.go::TestWorkflowPartialRendersCommittedExternalState`. `workflowGateCommand` returns argv including explicit JSON; caller supplies exact references, not prose-derived IDs. Root category is not included by `workflowCompletionEvent`. |
| Project workflow definitions | `cmd/lingo/workflow_authoring.go` → [workflow_authoring.go](../../../internal/cli/workflow_authoring.go): `category`, `*projectapp.WorkflowReport`, optional diagnostic | `axiom-workflow` authoring, `axiom-project` selection; `cmd/lingo/workflow_authoring_test.go`. Definition refs/revisions/digests remain immutable. |
| Stage plan / run / coordination / delivery | `cmd/lingo/{stage_service,stage_single,delivery}.go` → [workflow_stage_plan.go](../../../internal/cli/workflow_stage_plan.go) typed `stagePlanEvent`; [workflow_stage.go](../../../internal/cli/workflow_stage.go): `category`, `stage: any`, optional diagnostic | `axiom-work-item`, `cmd/lingo/stage_journey_blackbox_test.go`. Concrete dispatch DTO [DispatchPreview](../../../internal/stageexecution/preview.go), persisted graph/outcome payloads; `any` is not proof that arbitrary producer facts are present. Exhaustive delivery variant equivalence is unverified. |
| Runtime authoring/resolution/auth/capability | `cmd/lingo/runtime_*.go` → [runtime_authoring.go](../../../internal/cli/runtime_authoring.go) (`category`, `*runtimeapplication.AuthoringReport`), [runtime_resolution_completion.go](../../../internal/cli/runtime_resolution_completion.go) (`runtimeResolution`, `executionTarget`, `previewDigest`, or `runtimeAuth`), [runtime_capability.go](../../../internal/cli/runtime_capability.go) (`category`, `capability: any`) | `axiom-runtime`, Work Item execution preflight, runtime tests. `AuthoringReport` has typed revision/expectedRevision/previewDigest/candidateDigest/effects/knownImpacts; capability show is not execution authority. Full concrete capability variant audit remains unverified. |
| Effective Project context | `cmd/lingo/project_context.go` → [project_context.go](../../../internal/cli/project_context.go) `emitContextCompletion`: `projectapp.EffectiveContext` | Project/Work Item skill prerequisite readers and context tests; source refs and observations, not host instructions or permission grants. |
| Compatibility/recovery/cleanup/upgrade | [cmd/lingo/maintenance.go](../../../cmd/lingo/maintenance.go) → [cli/maintenance.go](../../../internal/cli/maintenance.go): `maintenance: any`; concrete `transferView`, `cleanupView`, `recoveryView`, `retirementView`, `upgradeView` | Direct CLI, `axiom-runtime` recovery instructions and `cmd/lingo/maintenance_blackbox_test.go`. Underlying [TransferPreview/TransferResult](../../../internal/compatibility/transfer.go) are typed; `boundedTransfer` reduces them (G2). |
| Legacy / unavailable service / help | `cli.go::event/eventFrom/emit`, `Status` with legacy values; command-specific unavailable-service paths | Old POC consumers cannot be silently recoded to MVP completion statuses. `emit` ignores encoder error; write-failure guarantees cannot be generalized to legacy. Help is deterministic text even with output selection; not a JSON operation document. |

## Required-fact coverage matrix

P/W/I/R below mean the installed `axiom-project`, `axiom-workflow`,
`axiom-work-item`, `axiom-runtime` sources respectively. These are concrete
instruction consumers today, not implemented deterministic JSON parsers. All four
require coordinated D2 migration; native JSON conversational behavior is unverified.

| R1 fact / scenario | Concrete field/type or invocation context and evidence | Consumers | Omission / unknown handling and verdict |
|---|---|---|---|
| Operation / target | `cli.go::action`, `requestInput`, closed `WorkflowStageInput`; `ProjectView.id`, `WorkItemView.{projectId,repositoryKey,provider,resource,externalId}`, `WorkflowView.executionId`; diagnostic `{operation,target}` | P/W/I/R; CLI routing tests | `covered` for invocation identity plus populated typed target. Common failure DTO lacks resolved target; retain actual invocation and report unresolved selection. No invented `operation` root is needed. |
| Completion and exit | `completion.go::completionExitCode`: success 0, interrupted 2, remaining five completion statuses 1; `completion.Result::Classify`; `TestCompletionGoldenMatrix` asserts literal JSON and codes | All CLI/skill consumers; smoke `run` checks both exit and status | `covered` for MVP; no blocked/no-op/recovery enum. Write failure returns 1 even after confirmed effects; exit alone cannot prove rollback. Legacy excluded. |
| Readiness / blockers and scope | `projectapp.ReadinessReport.{structure,effective,operations,blockers,warnings}`; each `OperationReadiness` has `operation/status/[]Finding`, `Finding.code/subject/detail`; readiness emitter preserves DTO | P/I; `bootstrap_readiness_test.go::TestBootstrapFromExplicitRepositoriesResolvesAmbiguityAndSeparatesLocalState` | `covered`: source test asserts success/0, effective partial, Work Item ready and Execution blocker `runtime_policy_unavailable`. That fixture uses Project schemaVersion 3, not diagnostic version 1. Not proof of the personal incident. |
| Warnings / qualifications | Readiness `warnings: []Finding`; Integration `findings: []IntegrationFinding` with severity/integration/capability and `truncated: bool`; draft `providerDocument.notices: []string`; runtime conflicts and impacts | P/W/I/R and typed JSON test decoders | `covered` for these payloads; no universal warnings array. Missing payload is unknown, never proof of zero warnings. Completeness across every `any` variant is `unverified`. |
| Confirmed effects / partial | `completion.Partial` requires references and next; `workItemResult` keeps confirmed Link; `WorkflowView.transitions` retains revision/refs; edit `diagnostic.confirmedEffects`; stage-plan `confirmedEffects: []string{}` | I/P; `TestWorkflowPartialRendersCommittedExternalState`, `TestOperatorPartialReferencesComeOnlyFromTypedResults` | `covered` for named typed references/diagnostics. Preview `effects` is not committed effects. Full transfer `written` is reduced: `gap_confirmed` G2. |
| Uncertain effects / safe recovery | Create `reconcileAttempt` returns `provider_create_ambiguous`; `workItemResult` retains category only in carrier; `emitResponse` falls through to common completion without it | I, future internal JSON consumer | `gap_confirmed` G1. `retryable_failure` cannot distinguish reconcile-only from proven no-effect retry. Retain uncertainty; no mutation replay based on status/prose. |
| Comment uncertainty (counterexample) | `workitem.Service.Comment` → `applyChange` always attaches `ChangePreview`; `emitWorkItemLifecycleCompletion` includes category `provider_comment_ambiguous` | I; `lifecycle_test.go::TestTransientProviderFailureIsRetryableWithoutLocalWrite` | `covered`: uncertain comment has failure/category/change; not retry-safe. This path is not evidence for G1. |
| Errors / categories | `cli.Result.Category: string`; work-item lifecycle/authoring/stage/operational/integration serializers include it; common, Work Item create/show and workflow completion DTOs do not | P/W/I/R; existing JSON test decoders | `gap_confirmed` for G1 and known discarded categories, e.g. `workflowResult` sets Category but workflow emitter omits it. Status survives; exact cause must not be recovered by prose parsing. |
| Authority / consent | Invocation flags `AuthorizeLocal/External`, preview digest and expected revision; `ReadinessReport.authority.grantsAuthority`; `DispatchPreview.children` rights/controls; diagnostic authority/options requiredAuthority | P/W/I/R; `workitem/lifecycle_test.go::TestCommentAndCompleteRequireTheReviewedDigest`, CLI routing authority tests | `covered` as typed payload plus actual invocation, not grant inferred from success. Absence of consent fields grants nothing; read-back and separate authority remain mandatory. |
| Preview / revisions / digest | `DraftPreview/SelectionPreview/ChangePreview` target/effects/expectedRevisions/digest; `EditPreview` portable/local revisions/digest; `WorkflowReport.projectRevision/reference/previewDigest`; `AuthoringReport.expectedRevision/previewDigest`; `DispatchPreview.digest/policyDigest` | P/W/I/R; setup smoke reads typed digest; authoring/recovery test sources | `covered` for emitted previews; never infer a current revision from provenance or old invocation. Missing preview/digest stops dependent mutation. Effects beyond transfer bound: G2. |
| Provenance / authorship | `completionEvent.provenance` product/version/revision/sourceState uses `provenance.SourceState`; `DraftSection.authorship` uses `provenance.Authorship`; `renderCompletionJSON` serializes directly | All; `TestCompletionGoldenMatrix`, smoke version/revision check | `covered` for MVP DTO. Dirty/unknown stays a qualification. Legacy has no equivalent guarantee. No model attribution invented. |
| References / Evidence / detail | `references: []string`, `details: string`; completion validates max 16 refs, 256 bytes each; partial requires reference; Workflow transition refs and stage payload Evidence/outcome refs | I/P; `TestCompletionPreservesStableDetailReferenceAcrossRenderers`; `cmd/lingo/stage_service.go` outcome refs | `covered` where populated; refs are handles, not observed contents or human acceptance. No global guarantee that `details` is populated; empty means no supplied detail handle. G2 partial transfer has no completed manifest to recover full written list. |
| Explicit unknown / recovery options | `Diagnostic` schemaVersion 1 has observed/notObserved/hypotheses/confirmedEffects/uncertainEffects/options/verification; options have availability/prerequisites/requiredAuthority/arguments/missingArguments | P/W/I; operator diagnostic tests and installed instructions | `covered` when emitted. `boundedDiagnostic` rejects invalid/over-16KiB enrichment; `presentDiagnosticEvent` omits when combined budget is exceeded. Material-fact retention under that omission is `unverified`, not guaranteed by the prose fallback. |
| Serialization / size / output failure | `presentEvent` writes one prebuilt wire, rejects oversize, short/error writes → 1; caps common 16KiB, Work Item/edit 128KiB, list 2MiB, bound Workflow 8MiB, stage 1MiB, runtime authoring/capability 256KiB | JSON callers; `TestRendererFailurePreservesConfirmedEffect`, `TestCompletionOutputIsBounded`; smoke independently limits 64KiB | `covered` at this transport boundary; pre-serialization reductions (G2) are not caught. Generic legacy writer ignores Encode errors. Truncated/unsupported JSON requires stop/handoff, not re-execution. |
| Real host / malformed / contradictory observations | Current four SKILL.md sources still require literal relay, JSON only on explicit user request; no evidence of future D2 parser/semantic behavior | P/W/I/R in Codex and Claude | `unverified`: native-host semantic equivalence, malformed JSON handling, redaction canaries, upgrade delivery and real incident. Source instructions are not execution evidence. |

## Confirmed gaps and bounded proposals — no schema approval

### G1 — Lost category makes create uncertainty prose-only

[service.go](../../../internal/workitem/service.go) `Create → executeCreate →
reconcileAttempt` returns `result(RetryableFailure, "provider_create_ambiguous")`
when acknowledgement is unresolved and reconciliation finds no item. This result
has no Draft/Selection/Change/Link payload; `verifyCreated` returns non-success
unchanged. `cmd/lingo/main.go::workItemResult` copies Category into `cli.Result`,
but `cli.go::emitResponse` selects `emitDiagnosticCompletion` with nil diagnostic.
`completionEvent` has no category; only `result`/`next` explain reconcile-only.
`service_test.go::TestCreateAmbiguousWithoutMatchReturnsSafeRetryBoundary` asserts
one create call across repeated attempts and that category. This is versioned
synthetic source evidence, not a remotely reproduced mutation.

Effect/consumer: I must preserve possible external creation, prohibit blind replay
and distinguish reconciliation from a proven no-effect retry. Invocation preserves
requested target but cannot establish a newly created external ID or acknowledgement.
The same category-loss mechanism affects create/select failures and Workflow
completions; it is not safe to assert universal error-category coverage.

**Recommended separate proposal:** evaluate mapping the already existing optional
`Diagnostic` schemaVersion 1 on the affected common-completion paths, carrying the
existing category, target observations and explicit uncertain-effects/reconciliation
boundary. This names only existing DTO members; no new field or envelope is
specified here. The proposal must prove bounded material-fact retention rather
than rely on optional diagnostic omission. Preserve all old fields/status/exit,
invocation identity and recovery budgets. This would still add observable data
on previously unenriched paths and therefore needs its own human review/acceptance.
Alternative: expose an existing category member on those wire shapes under a
separately approved additive contract; it is not authorized by this inventory.
Parsing `result`/`next` is incompatible with accepted R1.

Affected consumers/tests: I in both hosts, `emitResponse` JSON callers,
`internal/cli/completion_test.go`, `internal/workitem/service_test.go`,
`cmd/lingo/main_test.go`, operator diagnostic/budget tests. Future proposal must
cover ambiguous create, confirmed-no-effect retry, partial local linkage,
category retention, no duplicate create and old consumer tolerance, including
malformed/absent diagnostic and output limits. No such tests were added or run.

### G2 — Bounded maintenance view is not full transfer effects

`cmd/lingo/maintenance.go::boundedTransfer` slices `Preview.Effects`,
`Preview.Omitted` and `TransferResult.Written` to `maxViewItems = 64` before the
JSON serializer, while `effectsTotal` counts only preview effects. `transfer`
uses that view for preview, successful apply and partial apply.
The typed [TransferResult](../../../internal/compatibility/transfer.go) contains
`status/written/manifestWritten`; `TransferManifest` is written last. Thus a partial
transfer with more than 64 written objects can lose confirmed effect identities
and have no completed manifest. A `transfer:<digest>` reference alone does not
serialize those identities. No real filesystem transfer was run here. Exact
reachability with a >64 corpus remains a future deterministic verification case;
the slice and loss conditional on that input are explicit code, not speculation.

Effect/consumer: direct maintenance JSON clients and Runtime recovery guidance
cannot equate a bounded view with complete effects or exact consent scope.
Recommended separate proposal: establish the full typed transfer result as the
canonical serialization within an explicitly reviewed bound, or fail visibly
before effects when a complete result cannot be represented. Preserve existing
fields/types/status/digests; do not silently relax limits. A durable full artifact
with an existing detail handle is another candidate only if it covers partial
failure and is separately approved; no artifact/field is invented here.
Compatibility review must include legacy POC preservation/transfer consumers,
`maintenance_blackbox_test.go`, `internal/compatibility` transfer tests, 64/65+
effects, partial writes, missing manifest, redaction, consent digest and output
failure without replay. No implementation choice is made by this document.

## Installed skills and consumer compatibility

The four source consumers are
[Project](../../../internal/codexruntime/skills/axiom-project/SKILL.md),
[Workflow](../../../internal/codexruntime/skills/axiom-workflow/SKILL.md),
[Work Item](../../../internal/codexruntime/skills/axiom-work-item/SKILL.md) and
[Runtime](../../../internal/codexruntime/skills/axiom-runtime/SKILL.md).
`internal/codexruntime/runtime.go` embeds `skills/*/SKILL.md`;
[integration.go](../../../internal/codexruntime/integration.go) `NewClaude` uses
the same Service/content with a distinct integration/receipt. This proves the
installed-package production path for both hosts, not actual personal installations.
`shared_history_test.go`, `published_receipts_test.go`, `runtime_test.go`,
`workflow_skill_content_test.go` and CLI `domain_skill_routing_test.go` pin
ownership, receipts/digests, routing and current relay/Auxiliary obligations.
Published pins/state fixtures must remain historical; future upgrades must append
shared history and preserve modified user content. No installed bytes changed.

[scripts/test-codex-skills.sh](../../../scripts/test-codex-skills.sh) invokes the
optional skill validator and Go contract packages; it is not itself a JSON parser.
[scripts/smoke-prepared-artifact.py](../../../scripts/smoke-prepared-artifact.py)
requests explicit JSON, parses with `json.loads`, checks success/provenance and
setup preview/digest in an isolated corpus. Existing Go tests decode DTOs/maps;
Markdown goldens verify only presentation. External scripts and actual installed
versions outside the repository are unobservable; compatibility is not inferred.

D3 comparison: retain names, types, enum meanings, omission rules and the actual
invocation. Diagnostic version 1 is not a global result schema version; readiness
Project schemaVersion 1/2/3 and persisted format versions are independent.
Do not normalize legacy statuses or add a common operation/envelope version.
Wire-compatible byte/semantic regression plus coordinated binary/four-skill
upgrade/rollback and native host evidence remain future validation requirements.

## ADR-0030 — exact remaining human decision

The object is [ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md)
at source commit `18d29a6c5446357defb2563483cfac180dcc4043`, Git blob
`295084e6b6afa054b2f07dcaf5e73563ed7e7644`. Its bytes are unchanged by clarify.
It remains **Proposed**, independently of the approved Specification.

| Accepted direction / historical contract | ADR comparison and outcome |
|---|---|
| D1 | Decision explicitly keeps native human default in TTY and pipes, `--human`, explicit `--json`, non-TTY plain output/NO_COLOR. Fully represented; not reopened. |
| D2 | Decision explicitly requires binary plus four installed skills in Codex/Claude in AXM-32, replacing #232 and AXM-31 mandatory literal relay/Auxiliary prospectively. Fully represented; no deferred cleanup. |
| D3 | Decision preserves typed names/types/enums/status/invocation; forbids speculative fields/version; proven gaps require separate proposal/acceptance. Fully represented; G1/G2 activate its explicit prerequisite, not permission to alter it. |
| #232 / AXM-31 | Historical default Markdown/literal relay obligations conflict with the future presentation boundary as already recorded and decided in D2. Only these presentation obligations change prospectively; bounded diagnostics, authority, observed/not-observed, budgets and no automatic retries survive. No historical acceptance is reopened. |
| ADR-0003 / 0006 / 0022 / 0025 | Local control plane, detail artifacts, Project/Workflow/Work Item ownership and dispatch authority survive. No accepted ADR fragment requiring supersession identified in this review. |
| ADR-0018 | Only explicit human acceptance changes ADR lifecycle; no merge/CI/Spec acceptance implies it. No old ADR gets supersession annotations while 0030 is Proposed. |

**Pending human question:** accept or revise ADR-0030 at the exact revision/blob
above, with G1/G2 explicitly remaining separate contract decisions?
Recommendation: accept the recorded ownership/presentation boundary; retain the
D3 hold on any wire evolution. Alternative: request a bounded revision of the ADR;
name the affected fragment without reopening D1–D3. Host synthesis requires
stronger semantic/native checks; rollback becomes harder after consumers rely on
the boundary and requires a compatible binary/skill pair. This document neither
accepts the ADR nor records an answer on the maintainer's behalf.

## Handoff and verification

Current documentary result: matrix classified, G1/G2 exposed, ADR decision object
identified. Gate/Execution ID/revision are not observable canonically; no state is
reconstructed, duplicated or advanced. Linear In Progress is operational metadata.
No Plan/Tasks/implementation, merge, release, remote Runtime or Provider mutation
is authorized by this result. Git/PR documentary publication is the sole effect.

Before a future Plan: obtain independent explicit ADR acceptance (or revised
artifact acceptance), decide the bounded G1/G2 contract proposals and close any
material `unverified` producer variant relevant to that scope with deterministic
evidence, then run the subsequent gate. Do not treat inventory classification as
proof of complete JSON or permission to bypass a gap.

[Evidence](evidence.md#clarify-investigation--2026-10-18) records actual documentary
checks and remote-state observations. AC1–AC9 behavior, Go tests, integration/native
smoke, malformed JSON journeys, external consumers and actual installed host sets
are **NOT_RUN / unverified** in this documentary stage. No independent review is
claimed. No spawn: strict model inferiority was not verifiable; source searches,
integration and decisions stayed with the Orchestrator.
