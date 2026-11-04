# S2.4 — Canonical messages and coordination/gate reconciliation

Attempt: `IMPLEMENT-S2-04-CODEX-01`; scope: accepted Spec 015 Task S2.4 only.
Source: `4de87d18a098db7518bd52d69efd159a68c246c5`; local HEAD, remote branch
and open Draft PR #366 matched before edits; clean worktree, no ancestry drift.
[AXM-42 dispatch](https://linear.app/rgomids/issue/AXM-42#comment-c3030858)
fixes the implementation authority. [S2.2 review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5742024731)
and [S2.3 re-review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5744033110)
are analytical approvals, not native approval or acceptance. No later blocking
drift was present. Constitution, accepted Task/Plan and ADR-0037 supersession
map were checked before mutation; the complete Spec acceptance/non-goals were
rechecked during review. DESIGN: NOT_APPLICABLE (internal/CLI only).

## Reconciliation and deletion evidence

The [original inventory](deletion-baseline.json), Plan A–L E/L and S2.2's
TD-S22-07 identify the obsolete stream/store and split protocol. The
[before/after report](deletion-s2-04.json) records every changed/deleted Go path,
its original-inventory or expanded consumer-closure origin, physical lines,
lexical declarations, retired writers/formats and survival proofs. It counts
all Go files under cmd/internal so replacements outside the original envelope
cannot hide growth; comments/blanks count. No final S3 reduction claim is made.

Removed: `CoordinationStore` (including Publish/Latest/ReadStream),
`coordination.Service`/Input/Store/publisher, exported encoders,
AcceptanceVerifier and ClassifierRecord. The obsolete runtime task packet and
its unused fake-runtime hook/tests are removed with their consumers. The old
acceptance journey and test-only exports tested the deleted publisher, not the
canonical engine; they were retired, not rerun as product evidence. Graphify
outputs are excluded. No installed state was read, rewritten, converted or deleted.

One operational record remains: `workflow.Message` nested in Execution4;
`MessageCandidate` is its untrusted payload, not another persisted schema.
`Engine.Ingest` and the pump both commit through `Execution.AcceptMessage` and
the same principal writer/fence. The Codex/Claude common launcher shares this
record. No adapter-owned status, store or codec was introduced.

**Historical decoder survival proof:** `internal/coordination/coordination.go`
now exposes only DecodeStream/ValidRecord/ValidStream. The sole production
import is `internal/local/inventory.go`, which binds historical records to their
owning graph for upgrade classification. Released frozen corpus compatibility
requires this recognition; removing it would break the retained upgrade contract.
The decoder is not an operational message codec, cannot publish, and has no
path into canonical message ingestion or task eligibility. Frozen valid and
tampered records, relocated records and missing owners remain tested. Raw bytes
stay inert. The test fixture now reads already-frozen bytes instead of restoring
a retired writer. No equivalent codec/store was moved elsewhere.

**Gate ownership:** canonical before/after gates, exact-result human review,
S2.1 dependencies and S2.3 integration stay Execution-owned. Retained
`workflow/gates.go`/configured/service lifecycle types have historical delivery
consumers awaiting S3; deleting those here would reopen S3. No Graph gate/coordination verifier is wired into new-run operations.
The legacy `executiongraph/acceptance.go` verifier port, projections, constants
and BuildEvidence validation path survive without a production implementation
or caller; category G retirement is deferred to S3.4 in the survival report.
Test-supplied verifiers would not prove live Runtime evidence. A message cannot approve a human gate,
complete its own/another attempt, dispose unknown effects or establish integration.
The new four-kind test compares every non-message byte of the aggregate before
and after spoofed approval/integration/cancel claims.

CLI help/man generation, installed Workflow guide, commands, architecture and
bundled Work Item skill now describe the same record and private ingress.
The replaced shared skill manifest is registered in existing sharedSkillHistory,
and the new embedded manifest is pinned, preserving installed upgrade ownership.
List/show remain read-only; `--child` means sender agent. Legacy nonzero cursors
refuse. `coordination publish` is retained solely as an always-refused historical
spelling (`child_context_required`), never a duplicate ingress. S2.3 public
composition remains deferred. Behaviour step 4 in S2.3 Evidence is reconciled
with the prior review Minor: unobservable check → unknown/post_merge_check_unknown,
recovery_required until human disposition; no new behavior.

## Validation

All tests use isolated stubs/local temporary repositories, never live vendor inference.

| Evidence | Result |
| --- | --- |
| `TestHistoryExportsAreReadOnly` against baseline source in an isolated temp directory | RED, expected exit 1: detects New/Publish/VerifyCoordination/Encode/EncodeStream/ClassifierRecord/capacity APIs |
| Same test against changed source, `TestFrozenHistoryRejectsTampering` | PASS |
| `TestEveryV1WriterIsRecognizedByInventory`, `TestEverySupportedRecordLoadsThroughItsCanonicalStore`, retired stream without owning graph | PASS; historical compatibility and untouched bytes |
| `TestFourMessageKindsPreserveAllOperationalTruth` | PASS; question/answer/blocker/result grant no gate, attempt, effect or integration authority |
| `TestTwoStubQuestionAnswerIngress`, `TestLateOrUnmatchedAnswerIsRejected`, `TestIntegrationMessagesGrantNothing` | PASS; spoof, stale fence, duplicate/conflicting ID/reply, retired acceptance/dependency-resolution kinds, post-cancel and late-answer negatives |
| `go test -count=1 ./...` initial working snapshot | FAIL in installed skill ownership (below); final exact-commit suite status belongs to the CI receipt |
| `./scripts/test-docs-offline.sh` | PASS exit 0, installed linux-arm64 corpus/links and tamper/refusal journey; qualification PARTIAL (native Runtime observation and network/source denial NOT_RUN) |
| `./scripts/validate-repository.sh .` | PASS exit 0, including ADR/automation/sensitive-file checks; optional Runtime behavioral scenarios NOT_RUN |
| `./scripts/check-go-quality.sh format` and `tidy` | PASS; final changed Go files gofmt clean |
| Initial full-suite `TestEveryPublishedSharedRevisionStaysOwned` | FAIL: changed installed skill lacked replaced revision registration; corrected using existing sharedSkillHistory and manifest pin; `go test ./internal/codexruntime -count=1` PASS (18.091s) |
| Local Staticcheck | Initially unavailable; installed v0.8.1 into a task-local temporary directory after the CI finding; corrected run reported below |
| `go vet ./...`, `go build ./...` | PASS |
| `go run ./scripts/check-architecture.go domain` and `application` | PASS |
| gofmt on changed Go files; `git diff --check` | PASS |
| `go test -race ./internal/coordination` | NOT_RUN: tool refuses, CGO_ENABLED=0; no gcc |
| gitleaks | NOT_RUN: unavailable |
| Live Codex/Claude E2E, model CLI battery, remote-effect tests, cutover, merge/release/acceptance | NOT_RUN by scope |

The initial exploratory expectation that a historical stream should classify
`unknown` failed: it correctly classified `malformed` without its owner. This
was an investigation failure, not evidence of a product defect. Compatibility
source evidence required preserving recognition; the final negative checks
refusal and unchanged bytes without weakening classifications.

Current Codex usage was read through the local app-server account/rateLimits/read:
ordinary use allowed, primary 10% used/300 min, secondary 45%/10080 min. No old
reset time was trusted, no Claude fallback or subordinate model was used. No
vendor CLI test battery ran; any later invocation must use the user-pinned models.

## technical_decisions[]

All have `scope_impact: none`, local reversible implementation choices under
[Plan TD-03](plan.md#td-03--scheduling-messages-and-integration) and
[ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md).

- TD-S24-01: delete dormant writers and runtime packet rather than preserve an
  alternate ingress. Call/import closure proves no production packet consumer.
  This addresses DF-S22-01 within the explicit S2.4 duplicate-protocol scope.
- TD-S24-02: retain only the inert historical decoder in place. Alternative:
  remove it or copy a codec into inventory; both respectively break compatibility
  or duplicate code. Frozen corpus recognition is the survival invariant.
- TD-S24-03: retain historical publish refusal and existing list/show spellings;
  clarify help instead of adding/renaming a message command. Preserve bounded
  refusal and no cross-boundary writes; generated installed contracts follow help.
- TD-S24-04: retain shared historical lifecycle gates until S3 while verifying
  canonical gates against message spoofing; no new owner/implicit override.

## deferred_findings[] and limits

- DF-S22-01: addressed only as obsolete packet removal required for S2.4;
  historical finding remains recorded in S2.2 Evidence.
- DF-S22-02: historical Spec 007 links to deleted implementation paths remain
  deferred; do not rewrite history as current implementation.
- DF-S22-03: bounded whole-send-area reread each tick, measure before changing bounds.
- DF-S22-04: same-UID child filesystem access is not OS isolation.
- DF-S23-01: public composition of integration/worktree preparation remains deferred.
- DF-S23-02: same-UID ref writes are detected, not prevented.
- DF-S23-03: check-process identity not persisted across runner loss.
- DF-S23-04: worktree disposal remains explicit operator work.
- DF-S23-05: merge-tree requires Git >=2.38.
- DF-S23-06: pre-existing Git subprocess containment limits.
- DF-S23-07: integration disposition/status has no public CLI composition.
- DF-RR-01: existing store guard does not distinguish leased/lease-less
  prepared→terminal saves; no production lease-less caller does that. Deferred
  per [S2.3 re-review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5744033110).
- Earlier S2.1/S1 deferred findings remain linked through their Evidence; no
  optional work is automatically implemented or treated as a new requirement.

Self-review found no blocking finding in the bounded S2.4 change. It follows
Axiom/Ponytail engineering/security discipline; independent
S2.4 review remains the next gate, not claimed here. S2 needs that review and
integrated reconciliation. S3.1–S3.5, public integration composition decisions,
human pre-merge review, actual cutover and manual product acceptance remain.
No callback/continuation integration was established; handoff is
`AWAITING_EXECUTION_REPORT`, next gate `INDEPENDENT_REVIEW_S2_4`.

## CI correction on the same branch

Initial pushed SHA `4fbede0dc310890de24e5196c54b2370fac0aa45`:
[CI 41536438481](https://github.com/rgomids/axiom/actions/runs/41536438481)
FAIL in curated Staticcheck U1000. Four local test helpers (`localGraphFixture`,
`fixtureGraph`, `sortedEffects`, `effectsDigest`) had no remaining consumer after
coordination store tests were removed. Their complete caller inventory was
confined to `s8_stores_test.go`; the helpers and newly unused imports are deleted.
No production behavior, authority or test assertion changes. Delivery metadata
[41536438478](https://github.com/rgomids/axiom/actions/runs/41536438478) PASS at that SHA.
This is a bounded correction inside TD-S24-01, not a deferred optional finding.
The initial local full suite completed with exactly the installed skill-history
failure described above; all other packages passed. The corrected skill package
passed independently. Final exact-SHA full-suite/installed-docs/Staticcheck CI
is recorded in the completion receipt; the initial failure is not relabeled PASS.

Correction validation: `go test ./internal/local -count=1` PASS (66.441s),
and pinned `./scripts/check-go-quality.sh staticcheck` PASS with Go 1.26.0
binary and GOROOT matching CI. Three earlier local analysis attempts FAIL from
environment mismatch: host Go 1.27 export data was unsupported (including a
GOTOOLCHAIN-only attempt overridden by the script), then a Go 1.26 binary with
host Go 1.27 GOROOT. No project/tool version or source policy was changed to
hide those failures. The final version is Staticcheck 2026.2.1 (v0.8.1), checks
SA1*/SA2*/SA3*/SA4*/SA5*/SA9*/U1000 excluding SA1019, over `./...` including tests.
