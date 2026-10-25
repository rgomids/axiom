# Attempt-13 — Corrective guidance; native acceptance remains PARTIAL

Continuation of Specification 012 r1 / Plan r1 / Tasks r1, same open Draft
[PR #343](https://github.com/rgomids/axiom/pull/343), branch
`docs/axm-39-conversational-presentation`. Input local, remote and PR HEAD all
`bdf0f743ebec502961b2a5b62ed5c97df6768c29`; working tree initially clean.
The commit containing this artifact identifies the correction revision; the
single completion event records its exact SHA and final-head CI read-back.

## Sources, authority and bounded delta

[Attempt-12 failure ledger](evidence-native-attempt12.md),
[receipt](https://linear.app/rgomids/issue/AXM-39#comment-96d8b239),
[attempt-13 prompt](https://linear.app/rgomids/issue/AXM-39#comment-63d8818d),
[Spec](spec.md), [examples C01–C16](examples.md), [Plan](plan.md) and
[Tasks](tasks.md) remain immutable inputs. Linear comment inventory read back:
`hasNextPage: false`, no attempt-13 completion at entry. Shared field-note lookup
returned `McpServerError: Tool get_agent_context not found`; canonical comments
and repository remained available. No connector/configuration repair attempted.

Single executor, serialized changes to packaged Project, Workflow and Runtime
guidance. Explicit routine human labels and complete collision IDs; inert hostile
labels plus partial-inventory uncertainty; exact Agent declaration and independent
Plan/auth/execution/acceptance/compatibility/billing provenance; no freshness from
a revision label. Natural/explicit semantic equivalence retains UI observation gaps.
No domain/CLI/schema/flag/Registry/admission/Execution semantic change, ADR or
durable automation. Work Item skill bytes unchanged. No historical fixture, test
assertion or Evidence replaced; new tests identify this attempt separately.

Full replaced four-skill revision appended to `sharedSkillHistory`, including
unchanged Work Item digest, with compatibility and skill-set version still 3.
Previous manifest `d039a843c6289c3f91af73fab96e9e62c3cc1efafe6f08f07e163c925f5cc4b7`
remains pinned. New embedded manifest:
`df2d8dcb0f0ab4808240012918d806aa8e7a1472553101bd1ecd54aea25009ce`.
No installed skill or receipt changed.

## Regression method and actual local validation

New `TestConversationalAttempt13Guidance` ran before skill edits: **FAIL**, all
17 corrective required-string checks absent. Its compound shell also ran the new semantic
test, which passed and made the compound exit 0; the red command's individual
exit was not captured. After edits, focused guidance test **PASS**, exit 0.
String checks prove packaged instructions only, not behavioral compliance.

New `TestPresentationAttempt13Corrections`: independently authored synthetic
answers for C01/C02/C04/C06/C09/C10/C13/C16; required claims, omission and invention
mutants, exact-ID substitution for each collision candidate. Reproduces routine
UUID leaks, omitted exact `policy-default`, false certainty and suffix-only IDs
as rejected candidates. Existing historical C16 specimen still includes a UUID;
it is not relabeled or weakened. New C16 context forbids that leak and explicitly
keeps narrow GUI/independent harness NOT_RUN. No generated native transcript.

| Command / check | Result / exit |
| --- | --- |
| `go test ./internal/cli -run '^TestPresentation(012IdentityRuntimeAndReadiness\|Attempt13Corrections)$' -count=1 -v` | PASS / 0 |
| `go test ./internal/codexruntime -run 'Test(ConversationalAttempt13\|EveryPublishedSharedRevision\|OperatorConversational012Upgrade\|OperatorConversational012Stage\|OperatorConversational012C15\|CodexAndClaudeConverge\|ClaudeRevisionNConverges)' -count=1` | PASS / 0 |
| `go test ./internal/codexruntime ./internal/runtimebootstrap ./internal/cli ./cmd/lingo` | Pending completion before commit; final result in completion event |
| `./scripts/check-go-quality.sh format` | PASS / 0 |
| `go vet ./internal/codexruntime ./internal/cli ./internal/runtimebootstrap ./cmd/lingo` | PASS / 0 |
| `go build ./...` | PASS / 0 |
| `./scripts/validate-repository.sh .` | PASS / 0, including this new Evidence |
| Relative file links across Spec 012 and packaged skills | PASS / 0; 142 targets, 18 Markdown files |
| `git diff --check` | PASS / 0 |
| `./scripts/check-sensitive-files.sh --staged .` | PASS / 0 |
| `gitleaks git --pre-commit --staged --redact --no-banner --no-color .` | PASS / 0, no leaks |
| CI at the new HEAD | Not yet observed when this file was authored; actual final-head status belongs to completion event |

Packaging tests use disposable roots, not user-global installation. Includes
four-skill historical ownership, both Runtime roots, upgrade conflicts,
interruption, idempotence and restoration invariants. No live rollback exercised.
A worktree scan was first invoked with unsupported `--worktree`: exit 1,
`FAIL: target directory does not exist: --worktree`. Correct invocation
`./scripts/check-sensitive-files.sh .` then PASS / 0. No files changed from
the rejected invocation.

No full local `go test ./...`, race suite or optional external skill validator;
CI results must not be inferred from this narrower local scope.

## Native model, installation and host boundary

Read-only `claude --version`: **2.1.310** (exit 0); `codex --version`:
**0.163.0** (exit 0). No battery process, retry, resume, subagent or paid inference
launched. Required future pins remain `claude --model claude-haiku-5-5` and
`codex -m gpt-6.1-sol`; effective models **NOT_OBSERVED** this attempt. There is
no new model PASS, metadata proof or session ID. Attempt-12 model proof remains
scoped to its original sessions. No auth/login command or credential operation.

Current grant reports authorization for correction prompt publication; it does
not grant new paid inference or host mutation. Both native batteries:
**NOT_RUN / AUTHORIZATION_REQUIRED**. No model-unavailable failure occurred;
the battery was never started, so no vendor stderr/exit is invented.

Read-only installed `axiom --json version`, `docs inspect`, `runtime codex status`,
`runtime claude status`, `skill inspect axiom-project` and `project list` each
exit 0. Active build `0.21.3-axm39.8`, clean source
`e045cac1e3fdcd1bd8bfba925ddcfd135ce1e8f6`; binary SHA-256
`d6582d9851e7a138b8b9a91085bfede35fe20f068ee84e5fbc1b78b346dc8683`.
Docs verified/ok, installation compatible, both receipts current and files
equivalent to that installed revision. Inspector reports loaded **unknown**.
Installed docs expected manifest
`3471519797b11ddef021d1dea536538137f8747d9faaf2f96c110cf7d86fab50`
is its own inventory identity, not the legacy embedded manifest above.
This installed pair does **not** contain attempt-13 corrections. No supported
upgrade attempted; installation authority must be obtained separately.

Actual listing: `projects: []`, `No configured Projects`. No real Project
show/validate/Workflow/Stage journey available. No Project configured to fill
Evidence. No UI/harness observation or native loading proof on corrected bytes.
Text specimens and repeating a candidate under route names do not close C16.

## Case ledger and T08 / AC reconciliation

Each deterministic PASS below is bounded synthetic regression coverage, not a
native retest. The second column preserves attempt-12's actual Claude verdict;
all current-head native cases are **NOT_RUN**. Historical Codex C01–C15
PASS_SCOPED_REPLAY and C16 PARTIAL remain at their original revisions.

| Case | Attempt-12 Claude retained | Attempt-13 deterministic |
| --- | --- | --- |
| C01 | FAIL | PASS_SCOPED — label, no UUID leak |
| C02 | FAIL | PASS_SCOPED — separate name/slug observations |
| C03 | PASS_SCOPED_REPLAY | PASS_SCOPED — unnamed fallback |
| C04 | FAIL | PASS_SCOPED — both exact IDs, unresolved name choice |
| C05 | PASS_SCOPED_REPLAY | PASS_SCOPED — exact selector/digest/authority |
| C06 | PARTIAL | PASS_SCOPED — escaped inert label, uncertain uniqueness, no target |
| C07 | PASS_SCOPED_REPLAY | PASS_SCOPED — Stage summaries/detail |
| C08 | PASS_SCOPED_REPLAY | PASS_SCOPED — risk promotion |
| C09 | PARTIAL | PASS_SCOPED — exact Agent reference, separate Project/Plan |
| C10 | PARTIAL | PASS_SCOPED — independent provenance and NOT_OBSERVED |
| C11 | PASS_SCOPED_REPLAY | PASS_SCOPED — success vs blocked |
| C12 | PASS_SCOPED_REPLAY | PASS_SCOPED — unknown effects, blockers, reconcile |
| C13 | PARTIAL | PASS_SCOPED — no invented current/full revision |
| C14 | PASS_SCOPED_REPLAY | PASS_SCOPED — no new admission probe |
| C15 | PASS_SCOPED_REPLAY | PASS_SCOPED — ordered/DAG text fallback |
| C16 | PARTIAL | PASS_SCOPED text oracle; GUI/harness NOT_RUN, aggregate PARTIAL |

T01–T06 retain historical scoped PASS; affected correction/package checks are
separate new results. T07 remains **PARTIAL**, new E07-Codex/E07-Claude **NOT_RUN**.
T08/E08 **PARTIAL**: ledger and handoff reconciled, acceptance remains open.
AC1–AC5 **PARTIAL**: deterministic corrections cannot establish native fluency,
new loaded revision or real journey; AC6 **PASS_SCOPED_REVIEW** for examined diff.
Four pillars preserved: human identity/exact authority, progressive Stage detail,
Runtime/Profile provenance and honest blockers/readiness. Self-review only;
independent review readiness unproven, no CR dispatch authorized.

## Technical decisions and next gate

- **IMPL13-TD-01** — extend owner guidance plus separate attempt-13 test oracles;
  preserve historical specimens instead of rewriting failed outputs. Evidence:
  attempt-12 ledger and new tests. `TECHNICAL_LOCAL`, owner `executor-ai`,
  `scope_impact:none`.
- **IMPL13-TD-02** — append the full replaced skill revision and new manifest pin;
  avoid per-Runtime history or receipt edits. Evidence: manifest and ownership /
  upgrade tests. `TECHNICAL_LOCAL`, owner `executor-ai`, `scope_impact:none`.
- **IMPL13-TD-03** — retain native/UI gaps; do not install, configure Projects or
  spend inference from prompt-publication authority. Evidence: current grant,
  installed inspectors and empty listing. `TECHNICAL_LOCAL`, owner `executor-ai`,
  `scope_impact:none`.

**DF-01 DEFERRED_NOT_IMPLEMENTED** retained for human pre-merge; separate
[issue #346](https://github.com/rgomids/axiom/issues/346) unchanged. Neither is a
new requirement. Demonstrated failures stay mandatory unqualified native work.

Next gate: **RECONCILE_IMPLEMENT / AWAITING_EXECUTION_REPORT**. Authorized work:
bounded commit/push, exact-head CI read-back and one idempotent Linear receipt.
Separate **AUTHORIZATION_REQUIRED**: supported corrected installed pair, both
pinned native sessions/paid inference, any Project configuration needed for a
real journey, and actual C16 surface qualification. Keep failed/partial native
verdicts until new observations. No successor dispatch, merge, release, human
acceptance, Done transition or approval asserted.
