# CR-401-F1 — refuse impossible after-gated plans before persistence

Scope: AXM-12, `pr401-correct-01`, [PR #401](https://github.com/rgomids/axiom/pull/401),
[Minor finding](https://github.com/rgomids/axiom/pull/401#pullrequestreview-5905726698).
Baseline: `27061e3f9f69096d655355dcc7c608df7bb07986`. Design: NOT_APPLICABLE (CLI).
This is deterministic correction Evidence, not a new real Runtime E2E or MVP acceptance.
The [earlier E2E observations](evidence-axm-12-e2e-v0241.md) remain historical.

## Reproduction

On the baseline, instrumented
`TestPublicWorkflowRunSameRepositoryDependentWriterAndHumanGate/gate` read the
persisted Execution through public `workflow stage status`. Observed JSON fields:

```json
{"status":"success","category":"execution_ready","stage":{"status":"ready","nextAction":"This Execution can never start: its after human gate would wait on a same-Stage dependent of an integrated writer. Move the review/acceptance gate to a separate later Stage and start a new Execution; nothing was spawned or written to Git"}}
```

This is a field excerpt, not a complete response. The new public regression then
failed on that baseline: `impossible plan advertised or persisted an Execution`.

## Local technical decision

Reuse `AfterGateIntegrationDependency` in `Engine.Start` before `store.Create`.
Return the existing `ErrAfterGateIntegrationDependency` and an empty Execution,
so the CLI reports `failure` / `integration_refused` without a `stage` payload,
Execution identity or advertised `ready` status. The existing next action tells
the operator to move review/acceptance to a separate later Stage.

Alternative: introduce a persisted refusal state or derive a different status.
Rejected as unnecessary: refusing the impossible configuration before persistence
uses the accepted validation boundary, preserves derived-state invariants and
requires no enum, aggregate format or architecture change. The existing run-time
preflight and guidance remain for previously persisted records. This correction
does not migrate or rewrite those historical records.

## Regression and review

- Engine: repeated start with a fixed identity returns the exact refusal, no
  Execution/status, read-back not found, empty store and zero spawns.
- Public CLI: repeated refusal has structured failure/category, no stage/status
  or retained Execution, actionable next step, unchanged Git refs and worktrees.
- Valid dependent writers still complete with both integrations, accepted base
  propagation and public `execution_completed` / `stage.status: completed`.
- Reviewed the Start caller, store publication boundary, existing run preflight,
  status projection and ADR-0037. No new trust boundary, authority or Git effect;
  no blocking issue identified in this scoped engineering/security review.

## Validation

Runtime fixtures are local stubs; no vendor inference, paid API, real test
repository mutation or human gate approval.

| Check | Result |
|---|---|
| Targeted `go test ./internal/stageexecution ./cmd/lingo -run 'TestStart\|TestPublicWorkflowRunSameRepositoryDependentWriterAndHumanGate' -count=1` | PASS |
| `go test ./...` (Go 1.27.3, linux/arm64; includes CLI/status tests) | PASS |
| `go vet ./...` (Go 1.27.3) | PASS |
| `./scripts/check-go-quality.sh all` (Go 1.26.0, pinned Staticcheck 0.8.1) | PASS |
| `./scripts/test-docs-offline.sh` | PASS; qualification remains PARTIAL as reported by the script |
| `./scripts/validate-repository.sh .` | PASS |
| `git diff --check` | PASS |
| Real AI E2E / native Runtime qualification | NOT_RUN (excluded from this correction) |
| Gitleaks | NOT_RUN (not installed); repository sensitive-file checks used |

The first quality run failed under Go 1.27.3 because pinned Staticcheck supports
export data version 4, while that compiler emits version 5. A PATH-only switch
to cached Go 1.26.0 also failed because inherited GOROOT still named Go 1.27.
The successful run set both PATH and GOROOT to the existing cached toolchain:
`/home/grokbot/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.linux-arm64`,
with `STATICCHECK=/home/grokbot/go/bin/staticcheck`. No dependency or tool was
installed or upgraded. Offline docs checks do not prove network/source denial
or native Runtime behavior; the pinned v1 installer was unavailable.

F-17, shell/sandbox/roots/grants, accepted Specifications/ADRs, merge, release
and MVP acceptance remain outside this correction. Re-review of the pushed SHA
is the next gate; no approval is implied by these checks.
