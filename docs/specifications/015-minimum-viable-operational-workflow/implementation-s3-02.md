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

Pending final exact-source test/status collection. No remote implementation
publication or CI success is claimed. The user excluded real remote pushes;
this attempt prepares a local commit and handoff on the existing branch only.
Next gate after evidence reconciliation: independent `REVIEW_S3_2`.
