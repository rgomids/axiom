# S3.3 — Rejection, human disposition and linked rework

Attempt: `IMPLEMENT-S3-03-CODEX-01`. Scope: **S3.3 ONLY** of [Tasks](tasks.md).
Source base: `37e63e253ace7e33dfc7bc5343a880e99094be5a` on
`spec/axm-42-minimum-viable-workflow`, existing Draft [PR #366](https://github.com/rgomids/axiom/pull/366).
Authority: Rafael's 2026-11-03 08:19 BRT authorization, conveyed in the
[scoped implementation prompt](https://linear.app/rgomids/issue/AXM-42#comment-8823727a).
This does not authorize S3.4/S3.5, acceptance, closure, merge or release.

## Preflight and four pillars

- Local HEAD, `git ls-remote origin refs/heads/spec/axm-42-minimum-viable-workflow`
  and the actual GitHub PR head all matched the source base. Worktree was clean;
  ancestry was identical. Two worktrees existed on different branches. Process
  inspection identified this attempt's Codex principal and host, no competing
  implementation process in this worktree. No reset/rebase/overwrite.
- Runtime: **Codex**, running successfully (`codex exec -m gpt-6.1-sol`). No
  observed usage/model/host block, no switch, no Claude implementation or
  independent reviewer dispatched. No quota claim from historical reports.
- AXM/AC/non-goals: Tasks S3.3 and accepted Spec 015 rejection/history, no
  inherited permissions, human-only disposition, bounded explicit recovery.
- Product: one useful operational Execution path; technical process success
  remains distinct from human disposition and material-effect authority.
- Architecture/security: Constitution, Plan TD-01/02/04, ADR-0037 and its exact
  partial supersessions; existing CAS, lease, protected publication, rooted
  filesystem and artifact retention. No new owner/namespace/format marker,
  Provider authority, transport, dependency or Runtime capability.
- DESIGN: **NOT_APPLICABLE** (CLI/internal composition, no UI).
- Graph query used `GRAPHIFY_BACKEND=none` against the external card graph.
  It located Execution transitions, legacy rework and inventory consumers.
  Query was truncated (64/450 nodes); source/caller inspection supplied the
  inventory below. `rg` is unavailable; targeted `grep` was the fallback.
  Linear `get_agent_context` returned actual `Tool get_agent_context not found`;
  issue comments were readable and confirmed the scoped dispatch/authority.

## Actual inventory and scoped replacement

| Owner/boundary at base | S3.3 result |
| --- | --- |
| Canonical `Execution`, `ExecutionStore`, `Engine.Start/Run` | Same owners, store namespace and codec. Nested rejection and source link added |
| `ReviewStage`, gate approval, unknown abandonment | Existing human result review/Evidence checks reused; accepted review and terminal facts remain immutable |
| `Service.PreviewDeliveryRework`, `ReworkStore`, `reworkAdmission`, `retainedDefinition`, `ApplyDeliveryRework` | Deleted from `workflow/delivery_rework.go`; independent legacy rework composition cannot create new runs |
| Independent local `ApplyDeliveryRework` writer | Already absent at base; no deletion credit claimed |
| `ReworkInput/Preview/Link`, `Encode/DecodeReworkPreview` | Inert historical compatibility codec retained for local inventory, retained Execution loading, delivery decision attachment and artifact retention |
| `local/delivery_rework.go` legacy envelope codec/classifiers | Retained compatibility consumers: `workflow_store`, `inventory`, `delivery_decision`, `artifact_cleanup`, historical fixtures; no new operational route |
| Retired delivery/rework CLI mutation verbs | Remain retired; no preview/apply rework lifecycle restored |
| New public inputs | `workflow stage reject --execution --stage --result-digest --evidence --authorize-human --authorize-local`; ordinary `workflow run --rework-of <source-id>` plus fresh normal run inputs/authority |
| Owned artifacts | Existing retained human Evidence, process/output artifact refs, canonical Execution record; no rework file/store/queue |

The deleted legacy composition test is replaced by canonical domain/store/engine
and real public CLI fixture coverage. Historical codec/identity/retention tests
remain. Broader packet/publication/decision DTO/store/codec deletion, union
inventory and numerical deletion targets remain **S3.4**, not this task.

## Implemented behavior and limits

A human rejection binds the exact `StageResultDigest` (Execution, definition,
stage and full successful attempt/result/artifact facts), actor and retained
Evidence. The application reuses the existing observed typed-Evidence boundary:
matching Execution, stage, Workflow definition, result digest and Work Item
Repository, category/retention `evidence`, artifact publication outcome `success`.
That artifact outcome means successful Evidence creation, not acceptance of the
rejected work. The rejection content/reason remains in retained Evidence.

`rejected` is derived from this disposition; successful process facts, including
exit zero, stay successful observations. The source stops scheduling, retry,
gate/result approval and delivery. Existing human result approval cannot be
replaced. Active or unresolved unknown process/integration/delivery effects
refuse rejection, so it cannot bypass recovery. The store rejects erasure or
rewriting of any rejected history and makes a rework link immutable. Runner
claims on a rejected source refuse before changing its revision/generation.

Rework uses the ordinary fresh `Engine.Start`, invocation resolver, launcher,
store and scheduler. While holding the existing project directory lock, Create
loads the immutable rejected source, checks exact source/stage/result/rejection
provenance and Project/Work Item/Repository bindings, and requires the new
Execution to equal a freshly constructed pristine run. Source bytes never change;
only the new Execution is published, so there is no second write/transaction to
repair. A changed/missing/wrong-project/repository/attempt link refuses.

No old invocation, authority, attempt identity, gate/review, cancellation,
messages, integrations or delivery grants transfer. A selected current Workflow
and explicit artifact inputs are ordinary new-run inputs, never auto-copied from
the source. The current resolver must independently authorize the new attempt.
Git/PR grants remain bound to their own exact Execution/request; the link grants
nothing. There is no acceptance/close/merge/release operation here.

Retry/recovery retains shared controls: per-agent configured attempt limit
(default/maximum 10), maximum 24-hour attempt timeout, 8 bounded store-CAS
recomputations, 10-second detached terminal commit bound, monotonic runner lease
and exact attempt generation. Unknown effects never auto-replay; abandonment
requires human authority and observed absence of live/unobservable owned
processes. Changed invocation needs explicit `--new-selection` on a retry;
linked rework resolves a new invocation as a new run. No automatic retry/rework
loop or model fallback. Spec 015/Execution4 has no vendor token-metering or
cross-Execution token-budget capability; none is invented or certified here.

Conservative local limit: source and destination must have the same complete
Repository binding list/order. Rebinding/additional Repository context requires
resolving that mismatch, not silently retargeting provenance. Existing same-UID
and process-escape limitations of ADR-0005/0028 remain. No real cutover occurred.

## Verification and failure/repair evidence

[Machine-readable command evidence](verification-s3-03.json) records results,
source qualification and output digests. Tests use deterministic isolated
stubs/private state and temporary repositories only, with no vendor credentials
or real product remote mutation. Stub model strings are fixture data; no live
Codex/Claude test battery was launched.

Final local results: **PASS** / exit 0 for focused regressions, final full
`GOTOOLCHAIN=go1.26.0 go test ./... -count=1 -timeout=15m` (32 tested packages),
build, vet, Staticcheck v0.8.1/format/tidiness, module verification, domain and
application architecture, repository/ADR/automation governance and staged
sensitive-file scan. Installed-docs script: PASS with PARTIAL qualification
(native Runtime and source/network-denial observations NOT_RUN). Race NOT_RUN
(CGO disabled, gcc unavailable); gitleaks unavailable; live vendor/model battery,
product delivery and cutover NOT_RUN. No model fallback or live battery was
attempted. Git/CI publication is reported separately below and in the final
Linear callback, never inferred from local tests.

New contracts:

- `TestRejectedResultPreservesFactsAndStopsEveryProgression`: invalid human/agent
  actors, exit-zero distinction, no successor/approval/rejection rewrite, codec.
- `TestRejectedResultRefusesWrongEvidenceAndUncertainEffects`: stale result,
  wrong Repository/missing Evidence, unknown result refusal.
- `TestExecutionStoreRejectedHistoryAndLinkAreImmutable`: valid-shaped rejection
  erasure, actor/Evidence/result/authority rewrite and result erasure refused;
  linked pristine record, immutable link and unchanged source.
- `TestExecutionStoreRefusesForgedReworkAndInheritedState`: wrong source/stage/
  attempt/project/repository identity/Work Item, copied attempt/invocation and
  rejection/cancel state, changed timestamp and forged rejection digest refused.
- `TestLinkedReworkResolvesFreshAuthorityAndInvocationOnSharedEngine`: revoked
  current authority causes zero additional spawns/no destination; fresh own
  authority/invocation produces a new lifecycle/identity; source cannot replay.
- `TestPublicRejectedResultAndLinkedReworkUseCanonicalLifecycle`: real parser/
  application/launcher/store boundaries; missing authority, child/forged actor,
  wrong Execution/stage/result/Repository Evidence, immutable rejection, fresh
  selection/authority, source and link projections, delivery readiness denial.

Fault probes temporarily disabled one protection at a time, then restored it
before GREEN runs (these are mutation RED probes, not a claim that the new API
existed on the base). Actual observed RED/exit 1:

1. Without store rejection freeze: `erase rewrite: <nil>`.
2. Without canonical source/pristine validation: `stage forged rework accepted`.
3. Without actor ingress guard: `denial changed facts` (codec refused actor but
   the pure transition had already mutated the object).

The first CLI integration run also failed `rework mutated source`: a no-op Run
claimed a new source generation. Repair: reject source claims before ClaimRunner;
restored CLI/engine/store GREEN tests confirm unchanged source. Initial quality
run failed because inherited `GOROOT=/usr/local/go` (1.27) conflicted with the
selected 1.26 tool binary. Repair: `env -u GOROOT` with the pinned toolchain PATH;
Staticcheck v0.8.1 then passed. Neither failure is reclassified as PASS. A later test-label cleanup briefly
mismatched one local fixture case (`gate rewrite: <nil>`): the case performed no
mutation. Corrected it to `erase-results` and added explicit unhandled-case
failures; the repaired tests were rerun. This was a test fixture regression.

The first full suite also found five expected command-catalog/routing coverage
failures: the new rejection leaf/flag count, exact discovery inventory, routing
fixture, semantic-policy wording and skill table order. Reconciled these
consumers and the resulting shared-skill manifest pin; no checks were weakened.

Existing S1/S2/S3.1/S3.2 regression tests cover lost/stale/foreign fences,
unknown replay, bounded retries/recovery, explicit new selection, forged
messages/human gates, Repository/artifact provenance, exact push/PR denial,
create-only push, pre-effect refusal and actual lost-response unknowns.
No changes to the S3.2 effect/inspection semantics.

## Technical decisions and deferred findings

All decisions below are `TECHNICAL_LOCAL`, `scope_impact:none`:

- TD-S3.3-01: one nested result rejection and provenance-only source link.
  Reuse exact stage review digest/Evidence; an independent rework/decision
  object would reintroduce the owner being removed. Evidence: new domain/CLI
  tests and unchanged terminal attempt facts.
- TD-S3.3-02: validate the source and pristine destination inside existing
  Create/project lock. Immutable source needs no second mutation or new
  transaction protocol. Evidence: store mislink/inheritance and engine tests.
- TD-S3.3-03: retain inert legacy codecs only for existing compatibility,
  inventory and retention consumers; delete their independent composer now.
  Deleting the readers would break historical-data recognition; broader union
  removal belongs to S3.4. Evidence: caller inventory and full regression checks.

Deferred unchanged, separately tracked for pre-merge visibility:

- **D1**: delivered `HeadCommit/BaseCommit` provenance (prior S3.2 review).
- **D2**: `repositoryIdentity` discoverability (prior S3.2 review).
- **D3**: Inspect grant equality including `Actor` (prior S3.2 review).
- **N-REV-01**: deferred to S3.4; **N-REV-02**: deferred to S3.4/S3.5,
  unchanged from [S3.1 Evidence](implementation-s3-01.md).

No independent review dispatched. Implementation verification is not Axiom
Workflow ChatGPT CR, GitHub approval or product acceptance. Final nonforced Git
publication, actual commit/remote SHA and exact-SHA CI are reported in the same
AXM-42 completion callback after read-back. Next gate:
`AXIOM_WORKFLOW_CHATGPT_CR_S3_3`. S3.4/S3.5 remain unimplemented here.
