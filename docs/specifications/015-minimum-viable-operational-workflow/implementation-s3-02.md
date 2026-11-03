# S3.2 — Execution-owned delivery

Attempt: `IMPLEMENT-S3-02-CURSOR-01`. Scope: S3.2 only. Source baseline:
`d4e35c433e45208c89998118cf52da76c21cca9d`; base main:
`440e44a68638e08e901fd3f87f5b15cd395a6365`.

## Baseline and authority

Local branch and clean worktree matched the expected remote branch. Live GitHub
PR #366 was open/Draft, with the expected head/base. Both remote refs were
read back again after implementation. Ancestry checks exited 0. Latest AXM-42
comments contained this attempt's prompt/dispatch and the separately recorded
S3.1 correction/re-review, with no second S3.2 attempt observed. This is not a
claim that the original S3.1 partial report completed end-to-end delivery.

References: [accepted Tasks](tasks.md), [Plan TD-04](plan.md#td-04--delivery-evidence-and-recovery),
[ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md),
[F1/F2 review](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5746702906),
[S3.2 dispatch](https://linear.app/rgomids/issue/AXM-42#comment-286eadbc).

No real delivery push/PR, vendor inference, credential retrieval, cutover,
merge, release or closure was executed. No subordinate model was invoked:
strictly lower verified availability was not established. Linear's advertised
`get_agent_context` returned `Tool get_agent_context not found`; comments were
read successfully. The graph query was budget-truncated and used only to locate
source files; implementation conclusions came from code and tests.

## Implementation and technical decisions

- TD-S302-01: nested `Execution.deliveries` uses the existing format-4 codec,
  ExecutionStore, runner lease and CAS. There is no second state writer or
  delivery namespace. The store preserves intent, terminal result and prior
  observations; delivery writes require a live runner fence. Old format-4
  records with no delivery field retain their previous behavior.
- TD-S302-02: `workflow delivery advance` handles one current-phase push OR PR.
  A local read-only request review produces a digest, not a stored packet.
  The explicit human/local/effect grant binds Project, Execution/revision,
  checkout key/identity, target repository/remote, refs, commits, tree/diff,
  content/metadata and expected remote head. Source checkout identity may be
  `fs:`; it is distinct from the explicitly authorized GitHub target. Existing
  Git/Provider adapters are composed without a new service or dependency.
- TD-S302-03: intent is durable before the effect. Only exact observed facts
  confirm it. A lost response, drift after intent or failed final observation
  remains unknown/recovery-required. A subsequent runner marks abandoned
  prepared intents unknown and never repeats them. `advance --inspect` uses
  the original exact grant and appends observations only; even a matching
  remote result does not rewrite the unknown outcome. Human disposition and
  linked rework remain S3.3. Local completion needs none of this ceremony.
- TD-S302-04: S3.1 production readiness remains intact. Its two-repository test
  fixture now renames/restores an unclaimed directory instead of deleting and
  recreating it, preserving its bound filesystem identity. Before correction,
  the focused Go 1.26 run failed `repository_unavailable` after recreation;
  the same test could pass on Go 1.27 because inode reuse is incidental. The
  corrected focused regression passes without changing production admission.

The trusted application rechecks current Project admission, canonical repository
identity/path and observed S3.1 artifact readiness; the engine checks the exact
revision, grant, runner and remote observation before intent and effect. Runtime
child markers refuse human authorization at the CLI before Provider access;
private message ingress has no delivery-authority consumer. Existing OS/process
isolation limits remain; these checks do not claim a new sandbox.

## Deterministic acceptance evidence

Tests use only isolated temporary Execution stores, in-process fake Provider and
fake remote observations. No fake test invokes real remote delivery or vendors.

| Test | Safety exercised |
| --- | --- |
| `TestExecutionDeliveryDeniedEffectsHaveZeroCalls` | Push/PR absent, stale, wrong-effect, agent/message claims, changed target/SHA/binding/revision, revoked authority and cancellation: zero reads/effects |
| `TestExecutionDeliveryIntentCorrelationAndNoReplay` | Exact durable intent exists during effect; push and PR confirmations correlate; fresh revision/grant cannot replay |
| `TestExecutionDeliveryPushThenPRUsesSameExecution` | Sequential authorized push and PR phases keep one Execution and immutable confirmations |
| `TestExecutionDeliveryLostResponseInspectOnlyExactTarget` | Fake effect then lost response; unknown retained; wrong grant cannot inspect; exact inspection never repeats or fabricates success |
| `TestExecutionDeliveryDriftAndRevocationBeforeEffect` | Base/head/tree/remote-head drift, binding/authority revocation and persisted cancel after intent: zero effects |
| `TestExecutionDeliveryConcurrentLeaseStaleFenceAndHistory` | Concurrent current-revision operation loses the held lease; closed fence, lease-free history deletion and valid terminal intent rewrites are refused |
| `TestExecutionDeliveryCrashAfterIntentNeverReeffects` | Prepared intent survives a lost runner, becomes unknown and is never replayed |
| `TestExecutionDeliveryWrongObservedPRNeverConfirms` | Wrong repository/base/head/SHA/body/metadata, duplicate candidates or inconclusive observation cannot confirm |
| `TestExecutionDeliveryCodecRefusesFabricatedConfirmation` | Forged grant/request/observation, absent confirmation and future generation are refused by decoding |
| `TestPublicExecutionDeliveryExactGrantAndChildDenial` | Public CLI review has no effects; partial/stale/wrong authority and child context reach no Provider; status exposes exact confirmed intent without private executable/state paths |

Focused tests have passed, including the corrected S3.1 readiness regression.
Build, vet, module verification and architecture domain/application checks exited
0. `check-go-quality.sh all` exited 0 with the existing pinned Staticcheck v0.8.1
and cached Go 1.26.0. Earlier Go 1.27 static analysis failed on unsupported export
data; invoking the cached toolchain by PATH alone also failed because GOROOT
still selected 1.27. Explicit matching GOROOT/PATH fixed that environment issue.

Full-suite, installed-docs and final publication statuses are recorded in the
completion checkpoint below; running checks are not PASS. Initial broad checks
started during construction failed on transient compile/catalog drift and
concurrent embedded-skill/source changes; they do not establish final validity.
One earlier loaded run also failed the existing integration cancellation timing
assertion. Final stable-source results must supersede those attempts explicitly.

The full check of local implementation commit `a1e35ffe3ff30ea6f0faf738bcb85521e40e0bef`
found `TestEveryPublishedSharedRevisionStaysOwned` failing: changing the installed
skill also requires preserving its prior manifest in shared Runtime history.
The bounded correction appends the baseline's four actual skill digests and pins
the new embedded manifest in the existing ownership test. No ownership history
is removed or Runtime-only history changed. This is a required installed-skill
caller correction, not an S3.3 feature. Final checks cover the correction SHA.

## Actual ownership delta and retained scope

| Metric in this S3.2 delta | Before | After |
| --- | --- | --- |
| Active canonical Execution writers | 1 | 1 |
| Independent packet/publication/decision owners on the new-run operational route | 0 | 0 |
| Historical packet/publication/decision format/store families retained in source | 3 | 3 |
| New independent durable delivery owners/namespaces | 0 | 0 |
| Current-phase optional delivery operation | 0 | 1 |

No deletion reduction is claimed against this baseline: S1 already removed the
legacy delivery mutation route. S3.2 replaces its missing functionality using
nested Execution facts. Legacy DTOs/codecs/store methods and rework adapters
remain for S3.3/S3.4; S3.4 owns physical deletion and final union measurements.
S3.3 disposition/rejection/linked-rework and S3.4/S3.5 are not implemented.
D1/D2 and N-REV-01/02 remain deferred unchanged. DESIGN: NOT_APPLICABLE for this
CLI/internal scope. This is a self-review, not independent review or acceptance.

## Completion checkpoint

Implementation code SHA: `8dbc799bff58caaed2c954ae1c8a3b67e1daa262` (local,
unpublished), following implementation commit `a1e35ffe3ff30ea6f0faf738bcb85521e40e0bef`.
[Machine-readable verification](verification-s3-02.json) records commands, actual
source SHAs, output digests, exit codes, test names and limitations.

- PASS / exit 0 on the corrected SHA: focused delivery/readiness/ownership tests;
  full `GOTOOLCHAIN=go1.26.0 go test ./... -timeout=15m` (32 packages with tests,
  plus packages without tests); build; vet; pinned Staticcheck, format/tidiness;
  module verification. The prior exact-SHA full run at `a1e35ff` failed only
  `TestEveryPublishedSharedRevisionStaysOwned`; the final full run supersedes it.
- PASS / exit 0 on `a1e35ff`: repository/governance validation; ten quality-gate
  contracts after correcting the tool PATH; delivery and correction contracts
  with local repositories/fake gh; installed-docs journey. The latter retains
  PARTIAL qualification: real native Runtimes and source/network denial NOT_RUN.
  The subsequent correction changes only shared skill ownership history and its
  pin; full corrected-SHA Runtime/install tests pass. Documentation corpus and
  these governance scripts are unchanged by that correction.
- NOT_RUN: race (CGO disabled and gcc unavailable); gitleaks (not installed);
  remote CI on S3.2; real vendor E2E/battery, real delivery, cutover, merge,
  release and acceptance. No model availability is inferred.

Self-review identified and corrected the required fixture/skill-ownership
issues described above. No remaining blocking finding identified in that
self-review; this is not independent approval. S3.3–S3.5 and the recorded
deferred findings remain untouched.

Handoff is **stage.partial** solely for unpublished integration/remote Evidence:
the bounded implementation is locally validated, but the same PR still points
to `d4e35c433e45208c89998118cf52da76c21cca9d`. No push was performed under this
attempt's explicit exclusion of real remote pushes. CI is NOT_RUN for local
source; no GitHub S3.2 artifact/approval is fabricated. The same branch contains
local commits for publication only under separate authority. Next gate:
independent `REVIEW_S3_2`, after the authorized publication handoff. Checkpoint:
`AWAITING_EXECUTION_REPORT`. Available Cursor tools dispatch published prompts;
no authorized completion callback for this unpublished source was invoked.

## F1/F2 correction — CORRECT-S3-02-CODEX-01

Baseline: `0cb612d0953685cec4ab30dca8adad3ea6622efc`, verified against
origin and open Draft PR #366 before editing; worktree was clean. Scope is
S3.2 F1/F2 only, under Spec 015 safety obligations (line 84), Tasks S3.2,
Plan TD-04 and accepted ADR-0037. This section supersedes the earlier delivery
outcome description and unpublished checkpoint for this correction only.

- F1: a prepared intent rejected before mutation is durably terminal `refused`,
  with an end time and immutable history. It does not cause recovery_required
  or block a fresh exact authorized request. Cancellation and readiness still
  apply; there is no automatic retry. Refused results cannot be inspected into
  confirmation. Terminal writes survive caller cancellation, remain fenced,
  and a failed terminal write leaves the original intent for safe recovery.
- F2: push is create-only. `ExpectedRemoteHead` must be empty, including when an
  existing branch points at `HeadCommit`. Canonical validation rejects both
  same-SHA and different-SHA values before intent, Git or Provider construction.
  PR retains its existing expected-head semantics. No update/upsert transport.
- Prepared intents abandoned by an older runner generation remain unknown.
  An invoked mutation with an ambiguous/lost response remains unknown, blocks
  replay and can only receive append-only observations through Inspect.

### Technical local decisions

- TD-CORRECT-S302-01: add `refused` to the existing nested Execution delivery
  result, using the same codec/store/CAS/lease and history enforcement. Keep
  `unknown` for uncertainty rather than requiring human unknown disposition for
  known zero-mutation refusals. No new owner, store, command or ADR.
- TD-CORRECT-S302-02: use an explicit `ErrDeliveryRefused` adapter error only
  where the adapter knows no remote mutation was invoked (local Git guard,
  credential preparation, PR payload validation). Keep original errors joined
  for existing callers. Never infer refusal from a generic effect-call error.
- TD-CORRECT-S302-03: recheck the post-intent Execution revision before mutation,
  in addition to cancellation, generation, target observations and live checks.
  A fenced writer cannot finalize another generation's intent as refused.

### Deterministic verification

Red/green results and the publication checkpoint follow below. Tests use fake
effect counters and temporary repositories;
no real product delivery, credentials or vendor inference is exercised.

D1 (Execution result SHA provenance), D2 (repository identity discoverability)
and D3 (Inspect actor equality) remain **DEFERRED_NOT_IMPLEMENTED**. S3.3–S3.5,
merge, release and Product acceptance are outside this correction. Next gate:
`AXIOM_WORKFLOW_CHATGPT_CR_S3_2`. No independent reviewer is dispatched.

### Correction check ledger (2026-11-03)

Final local checks use Go 1.26.0 linux/arm64, with the cached toolchain first on
PATH, inherited GOROOT unset and GOTOOLCHAIN=local. Staticcheck is the repository
pin, 2026.2.1 (0.8.1). These are isolated technical checks, not Product acceptance.

| Check / command | Result | Exit |
| --- | --- | --- |
| Baseline archive at `0cb612d`: `go test ./internal/stageexecution ./cmd/lingo -run 'TestExecutionDelivery(DriftAndRevocationBeforeEffect\|PushMustCreate)$\|TestPublicExecutionDeliveryExactGrantAndChildDenial' -count=1` | RED, expected: seven pre-effect cases became unknown; both nonempty push heads passed canonical validation and reached a fake effect; CLI reached the adapter factory | 1 |
| Final Engine: `go test ./internal/stageexecution -run TestExecutionDelivery -count=1` | GREEN, including final-check context/persisted cancellation, fresh grant after refusal, codec/history, lost response and takeover | 0 |
| `go version` | PASS | 0 |
| `go test ./internal/stageexecution ./internal/workflow ./internal/gitworkspace ./internal/githubissues ./internal/local ./internal/cli ./cmd/lingo -run Delivery -count=1` | PASS | 0 |
| `go test ./... -timeout=15m` | PASS | 0 |
| `go build ./...` | PASS | 0 |
| `go vet ./...` | PASS | 0 |
| `./scripts/check-go-quality.sh` | PASS | 0 |
| `go mod verify` | PASS | 0 |
| `go run ./scripts/check-architecture.go` | Invocation error: required domain/application argument omitted; corrected below | 1 |
| `./scripts/validate-repository.sh .` | PASS | 0 |
| `git diff --check` | PASS | 0 |
| `go run ./scripts/check-architecture.go domain` | PASS | 0 |
| `go run ./scripts/check-architecture.go application` | PASS | 0 |

Exploratory runs are not omitted: initial baseline negatives exited 1 (the first
F2 fake also rejected an out-of-date fixture request; fixed before the isolated
baseline rerun above). Go 1.27.3 focused delivery tests across stageexecution,
workflow, gitworkspace, githubissues and cmd/lingo passed (exit 0); one redundant
focused run was interrupted with SIGQUIT (exit 1), and a broader duplicate
validation driver was stopped with SIGTERM (exit 143). No product failure is
inferred from those interruptions.

An initial Go 1.26 PATH-only attempt inherited Go 1.27 GOROOT: focused tests,
full tests, build, vet, quality and architecture invocation each exited 1 with
compiler/tool version mismatch. Module verification exited 0; the subsequent
repository-check driver was interrupted (143). The clean-environment final
checks above supersede these tooling failures. The no-argument architecture
invocation exited 1 (underlying usage exit 2); both explicit profiles passed.
`rg` was unavailable during discovery (127); targeted grep/find were used.

NOT_RUN: race (gcc/C compiler unavailable), gitleaks (not installed), native
vendor/model battery, real product remote push/PR, cutover and Product E2E.
Runtime usage quota is not exposed by the available tools; no paid agent/model
battery or independent review was invoked. Repository sensitive-file checks
are run against staged content before each commit.

Output SHA-256 (local logs; no raw provider/private data is imported):

- baseline red: `9ffa5c9a22076e404345fa001e33574345b3106e75ed42554da23827aef33abd`.

- final Engine: `790bad052eeb36d414f37259db79f6237283136ba066bb1d04ee0284d84d1433`.

- full suite: `e38a1c9d39706d33940add4b480e663d727e3ed1e43b4981722e3f276cef9558`.

Publication: Git transport is authorized only to the existing canonical branch
`spec/axm-42-minimum-viable-workflow` on origin, PR #366. Exact commit/remote/PR
SHAs and observed CI are recorded in the publication follow-up and the AXM-42
completion event; this does not authorize product delivery or approve review.
