# S3.4 — Final legacy cleanup and contract reconciliation

Attempts: `IMPLEMENT-S3-04-CODEX-01`, continued by `IMPLEMENT-S3-04-CODEX-02`. **S3.4 ONLY**, [Tasks](tasks.md).
Starting local/remote/PR HEAD: `2696dcaa088b53011a197ffe278a45d1b3cbf54b`,
canonical `spec/axm-42-minimum-viable-workflow`, existing Draft
[PR #366](https://github.com/rgomids/axiom/pull/366).
Authority: [scoped S3.4 dispatch](https://linear.app/rgomids/issue/AXM-42#comment-c93a842e)
after [S3.3 review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5751780180).
No S3.5/final review, product acceptance, AXM-42 Done, merge, release or real cutover.

## Preflight, scope and four pillars

`git fetch origin spec/axm-42-minimum-viable-workflow`, `git ls-remote`, local
HEAD and actual GitHub PR metadata all agreed with the starting SHA. Worktree
was clean at the original attempt preflight. Merge base with fetched main was `440e44a68638e08e901fd3f87f5b15cd395a6365`.
Process/worktree inspection found this principal and host, with another worktree
on a different branch; no conflicting executor in this worktree. No reset,
rebase, force push, second PR or main work occurred. `rg` was unavailable;
targeted `grep` followed the external Graphify query. The graph search was
truncated (63/4764 nodes); it was navigation, never a complete inventory.
Linear comments and the S3.3 GitHub review were read; Linear shared context
returned `Tool get_agent_context not found`, not an inferred absence of history.

The accepted AXM/AC/non-goals and product objective remain one useful workflow
with optional validation and authorized delivery. Architecture follows the
Constitution, Plan TD-01–04 and ADR-0037's exact partial supersessions.
Project != Repository, Runtime != Model, Execution != Agent, Evidence !=
conversation, Provider != Transport and Integration != MCP remain intact.
DESIGN: **NOT_APPLICABLE**, internal/CLI cleanup without a new UX decision.
No material product/architecture decision or new dependency/store/codec family.

Runtime: observed available Codex principal (`codex exec -m gpt-6.1-sol`). Two
bounded implementation/research units inherited principal settings: dead Graph
producer closure and dead adapter/Git/sequential-service closure. Their independent
file ownership and caller research made delegation useful; final integration and
repository validation remained here. No claim of a cheaper model's availability,
quota hierarchy or reduced-model execution. No live model battery or fallback,
no independent reviewer dispatched. This maintainer delegation is not Axiom
product runtime functionality. Final code review belongs exclusively to ChatGPT
Axiom Workflow; no S3.5 work is claimed.

## Final union inventory and dependency order

[Machine-readable inventory](deletion-s3-04.json) includes baseline paths and
**every changed Go path anywhere** since its source SHA, including replacement
files, command consumers, tests and maintainer Go scripts. Each row records
before/after physical lines, declaration names, SHA-256 and imports; reverse
consumers are package-level imports, including tests. The report's role/format
tables distinguish operational production from inert recognition. Imports alone
are not proof a particular symbol is called: targeted symbol searches established
the dead producer closure before editing, and the full build/static/tests check
its reverse dependencies. No historical user state was read, converted or deleted.

Deletion order: callers/independent public compositions → test fixtures using
those writers → redundant orchestration/DTOs → independent persistence mutators →
unused private helpers → strict retained codecs and canonical regression checks.
Legacy tests became byte fixtures or were removed only when they exercised a
retired engine; shared safety assertions were retained or ported to actual
canonical boundaries. No old engine was moved into tests or another package.

| Boundary | Removed producer/consumer closure | Surviving responsibility and concrete removal failure |
| --- | --- | --- |
| Graph planning/Single/Stage Plan | Proposal/ProposedNode, acceptance RunEnvelope/Evidence composer, Graph integration service, consumer scheduling/workspace projection, dispatch/retry/reconciliation mutations and derived StageOutcome builder | `executiongraph` strict Graph/successor/Single/authority/input/receipt/outcome validation feeds `local/inventory`, identity checks and artifact retention. Removing it loses recognition or weakens forged-history refusal. `planner.go` now contains envelope validation only; `integration.go` only persisted OptionalWaiver fields |
| Shared process boundary | Old Graph execution/classification orchestration | `scheduler.go`, `process*.go`, invocation/environment and diagnostic validators serve active Engine/RuntimeLauncher and CLI; deletion breaks spawn/cancel/unknown identity safeguards. Reuse avoids a replacement process implementation |
| Validator Policy | Dead `Commands` executor projection and authority/digest errors | Policy format-1 strict validation/encoding/digest remains for inventory's byte identity; deleting it loses policy recognition. No Policy store/publisher/dispatch requirement survives |
| Runtime adapters | InvocationResolver/CredentialResolver and frozen legacy stage builder | `BuildStageInvocation`, selector-source inspection and frozen canonical invocation remain active. AuthPreflight still has a runtime-bootstrap consumer; normal dispatch does not require it. Removing active inspection would lose ambiguity/redaction checks |
| Git adapter | Graph/Single Manager constructors, preparation/binding/integration/reset/ownerRecord writer, CombinedValidator runner, packet DeliveryCommitPreview/apply | Existing Manager utilities serve active Integrator and DeliveryGit. Root confinement, executable filters, Git environment, exact target/ref/CAS and no-force checks remain. Removing the utilities breaks canonical integration/delivery; no replacement Git subsystem |
| Sequential workflow service | Start/Transition/RecordLifecycleFact, Store interface and persistence compositions, WithDefinitions/ValidateTarget | `ResolveLinked` remains the active existing Work Item/Project/Repository resolver. Historical State/binding/fact/ledger/result/projection validators remain for strict decoding and retention. Removing them accepts corrupt lineage or loses historical identification; no operational historical writer |
| Delivery/rework | ComposeDeliveryPacket, DeliveryPublisher and store/authority DTOs, decision prepare/apply/store interface, packet/publication/decision local mutation methods | Packet/publication/decision/rework/projection codecs and read-only attachment/identity loaders remain for `inventory`, `artifact_cleanup`, historical provenance. Removing them risks deleting referenced artifacts or misclassifying history. Canonical Execution owns requests/grants/observations/rejection/rework link |
| CLI and installed skills | Legacy mutation verbs stay retired; no alternate route restored | `workflow run`, canonical stage controls, delivery readiness/advance and private message ingress. Catalog/route tests cover exact exposed leaves and flags. Codex/Claude share one embedded skill corpus and ownership-history pin; no installed private state modified |

These retained implementations meet the three-part survival proof: distinct
recognition/retention or active safety invariant, concrete failure above, and
simpler alternative comparison. Existing bounded codecs/helpers avoid a new
opaque compatibility codec, unsafe deletion or a dual executor. Encode functions
used to verify historical shape/digest are serialization, **not state writers**.
No retained historical reader can start, resume, retry, integrate or publish work.

## Persisted formats and writers

| Format / namespace | Final writer / reader contract |
| --- | --- |
| Canonical `kind:execution`, format 4, `executions/v2` | `ExecutionStore.Create/Save/UpdateFenced`, shared engine/CLI; sole new-run truth |
| Runner lease format 1, `run-leases/v1` | Existing lease acquire/epoch persistence remains active for canonical fencing; historical leases also inventoried. Root name does not make this active safety mechanism redundant |
| Historical Workflow State 1/2/3/5, `executions/v1`; format-4 `kind:execution_link` envelope | **No production writer**; strict snapshot/link decoders, provenance attachment, inventory and retention. This old envelope is never parsed as canonical Execution4 |
| Graph 1/2 and successor Graph 3/4, `graphs/v1` | **No production writer**; strict graph/document decoders and retained-record identity validation |
| Single run 1/2 and Single authority 2, `single-stage-runs/v1` | **No production writer**; strict historical recognition |
| Dispatch authority 1; retry authority 1; dispatch inputs 1/2 | **No production writer**; bounded historical digest/identity validation |
| Integration receipt/authority 1/2; StageOutcome 1 | **No production writer**; local classifiers/readers and artifact references |
| Packet/publication/decision/rework/projection 1, `executions/v1` | **No production writer**; read-only historical loaders/classifiers/attachment/retention. Independent composers and mutation methods removed |
| Validator policy 1, `validator-policies/v1` | **No production writer**; canonical-byte inventory recognition only |
| Coordination stream/record 1, `coordination/v1` | **No production writer/verifier**; strict decoder and lineage validation for inventory. New four-kind messages are nested Execution facts |
| Stage Plan/PlanRef and capability/probe envelopes | No restored producer/store/codec. Existing cutover catalog preserves raw legacy families, blocks incomplete/active/unknown state and never converts bytes |
| Artifacts/cleanup/retirement, Projects/Work Items, Workflow definitions/indexes, Runtime/Profile and Project context | Existing unrelated current formats/writers remain at their owners; `StateRootDirectories`, `InspectStateInventory` and existing corpus enumerate them. They are not redundant legacy Execution truth. No format/writer was added or migrated here |

Existing recovery/publication markers and historical retirement remain safety
mechanisms, not legacy execution. Backup writes only a separate authorized
private destination; actual private inventory/backup/stop/cutover is NOT_RUN.
Root/protocol validation and raw preservation are retained, with no compatibility
executor or silent old-byte reinterpretation.

## Measurement

Accepted baseline source: `0a65d9a90e907fe1904241a82878e0196418e351`, from
[deletion-baseline.json](deletion-baseline.json). Its original 172-path envelope
(88 production files / 21,872 lines) is **not** the denominator for a final
replacement-union claim. The final union extends it to all changed/added/deleted
Go paths; independent full-repository counts must yield identical net deltas.

Physical lines use `splitlines()`, including blanks/comments. Production is
all `.go` except `_test.go`, consistently including build-tag/platform/maintainer
Go sources; tests are reported separately. Declaration regexes match the approved
Plan (`^type ([A-Z]\w*)`, `^func (?:\([^\n]*?\) )?([A-Z]\w*)\(`), plus
lowercase declarations for total function/type counts. Counts are lexical,
not semantic API reachability; same-named methods count separately. Imports are
package-level consumers. The JSON embeds the exact reproduction Python and its
SHA-256: extract `reproduction_python` to a temporary `.py` file and run from
this worktree. It reads Git blobs without checkout and repository files only.
It asserts equality of union and full-repository deltas; removed files, replacements
and modified pre-existing files outside the envelope all count. No reclassification,
relocation, generated duplicate, test deletion reward or file-count-only deletion.

| Metric | Baseline union | Final union | Net removed |
| --- | ---: | ---: | ---: |
| Production files | 140 | 115 | 25 |
| Production physical lines | 37,981 | 27,327 | 10,654 |
| Production exported types | 451 | 314 | 137 |
| Production exported functions/methods | 560 | 352 | 208 |
| All production functions/methods | 1,339 | 942 | 397 |
| All production type declarations | 554 | 383 | 171 |
| Test files | 172 | 135 | 37 |
| Test physical lines | 40,415 | 29,397 | 11,018 |

All three cumulative budgets pass. S3.4 alone removes 5 production files, 4,449 lines and 48 exported types; the budgets apply cumulatively to the accepted baseline. The final union has 386 paths. Whole-repository Go deltas are identical. Separate test changes above receive no deletion credit.

## Technical coverage of P-AC (manual acceptance remains later)

| Obligation | Retained deterministic coverage / boundary |
| --- | --- |
| P-AC1 one action/default/optional Profile/invocation; no legacy objects | `TestPublicWorkflowRunMinimalDefaultCompletesWithoutPlanPolicyOrProfile`, machine-local/default/direct-override tests; `TestLegacyLifecycleHasNoProductionMutators` and exact catalog tests |
| P-AC2 truthful state/output/error/recovery | `TestPublicWorkflowRunFailureMissingOutputRetryAndNewSelection`, runtime launcher selector/redaction/output tests, process correlation and observed-stop tests |
| P-AC3 one/many scheduler/messages/integration | `TestEngineParallelReadyTasksStreamAndCommitWithoutLoss`, `TestExchangeCodexClaudeBidirectionalUnblocksOnlyOnValidatedResult`, dependency/forgery/late-message negatives and `TestIntegrateParallelWorktreesSequentiallyWithCheckEvidence` plus dirty/conflict/drift/failed-check/unknown tests |
| P-AC4 distinct Repository outcomes/artifacts | `TestRepositoryOutcomesTwoIndependentBindingsOneExecution`, public readiness reobservation, unsafe/rebound path, unknown-effect and unclaimed Repository tests |
| P-AC5 fences/ownership/unknown/cancel/authorization | Crash windows, stale lease/history/CAS, cancel during spawn/check/validation, bounded retry/timeout, unknown-no-replay/abandon-live-process tests; no live Provider inference |
| P-AC6 same delivery/rejection/rework path | Delivery denied-zero-calls, create-only push, PR mismatch, exact grant/child denial, lost response/inspect-only, rejected-history/link immutability, `TestPublicRejectedResultAndLinkedReworkUseCanonicalLifecycle`; fresh invocation/authority |
| P-AC7 actual removal/retention | Whole-repo crosschecked union JSON; AST retired-surface guard; strict history/corpus/identity/artifact-retention/cutover tests; no private-state deletion |

## Verification, decisions and limitations

[Command evidence](verification-s3-04.json) records exact commands, exit codes,
source qualification and output digests. Tests are unit/isolated temporary state,
process and local repository fixtures; stub Runtime labels are not live vendor
qualification. New historical header/lineage tests had an observed mutation RED
when the parent-lineage equality guard was temporarily removed (`invalid lineage
accepted: <nil>`, exit 1), then GREEN with the guard restored. No mutation remained.
The initial focused compile found an unused import (repaired). An overlapping
blackbox build observed intermediate delegated edits; that FAIL is recorded,
not final coverage. The installed-docs check also exposed an Evidence link outside
the installed corpus; the operational guide now names the source report without
an unresolved package link. Final integrated checks determine the result.

Technical decisions (all `TECHNICAL_LOCAL`, `scope_impact:none`):

- TD-S34-01: delete independent producers/writers after proving reverse-call
  closure; preserve strict codecs for actual inventory/provenance/retention
  consumers. Evidence: AST guard, historical identity/retention and full build.
- TD-S34-02: reuse shared process/Git/validation boundaries; port executable-filter,
  root confinement and shared-checkout tests onto active Integrator. Evidence:
  canonical process/integration/delivery regressions, no new safety subsystem.
- TD-S34-03: frozen historical byte fixtures replace old engine/writer setup;
  no retired engine copied into test helpers. Evidence: frozen corpus, unchanged
  byte identity, semantic/lineage refusal and current linked-target resolution tests.
- TD-S34-04: count full replacement union plus independent whole-repo delta;
  retain serialization only where real readers need it. Evidence: reproduction
  assertion, per-path before/after symbols and source hashes.

Deferred findings: **D1** delivered HeadCommit/BaseCommit provenance, **D2**
repositoryIdentity discoverability, **D3** Inspect grant equality including Actor
remain NOT_IMPLEMENTED, unchanged from S3.2/S3.3. **N-REV-01** is addressed by
removing the obsolete builder/verifier comments and annotating dated
`evidence-s8.md` without rewriting its history. **N-REV-02** is addressed by direct
negative tests for invalid kind/ID, stream header/lineage, trailing JSON and
oversize refusal. Their original review remains
[5745407569](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5745407569).
These are implementation claims, pending independent confirmation, to be assessed at the authorized S3.4 review,
not invented reviewer approval or resolution of D1–D3.

Same-UID/process-escape limitations of ADR-0005/0028 remain. Race is NOT_RUN
when CGO/gcc is absent; gitleaks NOT_RUN if unavailable. Live vendor/model E2E,
network/source-denial/native Runtime docs qualification, actual cutover and
manual product acceptance remain NOT_RUN. Documentation script PASS can still
have PARTIAL native qualification. No real delivery remote push/PR, merge,
release or acceptance was tested or authorized. Development Git publication to
this PR branch is separate and is recorded with exact commit/remote/CI facts
in the final AXM-42 callback. S3.5 and later manual acceptance remain pending.

### Retired test closure and surviving safety cases

| Retired test owner | Why retired / retained or ported evidence |
| --- | --- |
| Graph acceptance/planner/consumer/integration/reconciliation tests | Exercise deleted independent producers. Strict Graph/Single/version/receipt negative vectors remain; canonical parallel/dependency/integration/unknown/cancel/fence tests cover their accepted operational obligations |
| Graph classifier assertions inside process tests | Deleted classifier has no new-run consumer. Actual runner descendant/escape/stop/output assertions remain; Engine crash/unknown and validated-result tests determine current status |
| Old workspace reset/CombinedValidator tests | No reset/combined legacy API remains. Canonical writable retry requires explicit preservation; actual Integrator tests retain command-check failure, safe root/filter/shared-checkout, drift/conflict and CAS behavior |
| Legacy InvocationResolver/frozen builder tests | Canonical RuntimeLauncher/StageInvocation tests retain exact selectors, config conflicts, environment isolation and redaction; no live vendor behavior inferred |
| Sequential Service/configured/dogfood tests | Retired engine cannot be exercised. ResolveLinked target/cancel/identity tests remain; historical binding tests use frozen bytes; canonical public default/gates/retry/review tests cover new operation |
| Packet/publication/decision composition tests | Those objects no longer own operations. Canonical domain/engine/public delivery and rejection/rework tests retain authority, exact targets, no replay and immutable history. Historical readback/tamper/identity/artifact references remain; fixtures write codec bytes in temporary state only |

No test count is an acceptance target. Shared process production code and
canonical lifecycle tests were not removed. The full-suite result, not the
removed-file count, qualifies the integrated change.

## Continuation checkpoint — IMPLEMENT-S3-04-CODEX-02

The continuation preserved `cfa39d363fc6781c0ab3413653fd08781f02fab9` and all
uncommitted documentation/evidence. Fetch, ls-remote, ancestry and PR metadata
confirmed remote/PR HEAD `2696dcaa088b53011a197ffe278a45d1b3cbf54b`; no
conflicting executor was observed. Contrary to the handoff's pending-fix note,
that local commit already includes the superseded shared skill revision
`639bdbfdf235fd05cdc5b0d3cd0b5a4c18cdbe78912460fb7fcd3bbc5106ebff`.
The four skill digests were independently recovered from `2696dca`, yielding
that exact manifest digest, and `TestEveryPublishedSharedRevisionStaysOwned`
passed on the unchanged committed source. No duplicate history entry or Go edit
was necessary. The final union JSON therefore still measures the exact code
commit; its test-line total includes the history pin. Final verification records
which commands were rerun by this continuation and which earlier RED receipts
were recovered from retained logs. No subagent or model battery was started by
the continuation. Publication and exact-SHA CI are recorded in the single
Linear completion receipt.
