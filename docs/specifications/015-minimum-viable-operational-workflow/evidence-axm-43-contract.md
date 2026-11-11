# AXM-43 — Contract analysis Evidence

2026-11-11; base `e5b9d10880cfc210356b8c49d639b1718b494e01`.
Scope: [pending amendment / Plan / Tasks](amendment-axm-43-same-repository.md).
**DECISION_PENDING (Rafael)**. Documentation-only authorized fallback; no P-AC3
implementation, Product acceptance, merge, cutover or external Project mutation.

## Inspection and review

Clean initial worktree on the requested Linear branch. Read canonical repository
instructions, implementation/SDD/review/documentation skills, Ponytail ultra and
review discipline, security/quality/contribution/documentation policies, Spec 015,
Plan TD-03, ADR-0037, S2.3 report and #369 amendment pattern before editing.
Queried the existing repository graph with `GRAPHIFY_BACKEND=none`; its broad
query was truncated, so inspected concrete source directly. `rg` is absent;
used targeted `grep`/file reads. No graph was generated in the repository.

Confirmed the missing production call, launcher repository-level CWD selection,
succeeded-only integration admission, runner lease composition boundary and
result-only dependency/stage eligibility. These are source observations, not
failed runtime demonstrations. The need for explicit amendment approval is an
engineering interpretation, presented for Rafael's review rather than accepted
architecture. Runtime behavior remains unchanged. No code, exports, fixtures,
configuration, public help, installed docs metadata or canonical skills changed;
those surfaces must change together when the proposed behavior is implemented.

Attempt 03 (Claude, runtime failover after the Codex usage limit) re-verified the
cited source observations at `runtime_launcher.go:105`, `execution_integration.go:209`,
`integration.go:64` and `scheduler.go:61`, and kept the analysis unchanged.

Proportional review: no unrelated changes or accepted-history edits; no new
state owner, dependency, automation or public surface; links and claims checked
against source. Existing helper tests do not establish public run integration.
No independent agent review was requested or performed.

## Executed validation

Final results are recorded after each command completes. Test runs use existing
isolated fixtures and fake/stub boundaries, without vendor inference. No Axiom
binary was manually invoked, and no real runtime authentication was probed.

| Check | Result |
| --- | --- |
| `git ls-files '*.go' \| xargs gofmt -l` | PASS (empty) |
| `go vet ./...` | PASS |
| `go test ./internal/stageexecution ./internal/gitworkspace ./internal/workflow ./internal/local ./cmd/lingo -run 'TestIntegrat\|TestObserveIntegration\|TestExecution' -count=1` | PASS |
| `go test ./... -count=1` | FAIL then PASS: every package `ok` except `cmd/lingo`, which hit the default 10m package timeout (blocked in `fsync` on the shared 2-CPU host). Isolated rerun `go test ./cmd/lingo -count=1 -timeout 30m`: PASS (`ok` 667.9s), above the default limit. No Go code changed |
| `./scripts/validate-repository.sh .` | FAIL (local host, unrelated to this docs-only diff): `test_model_advisor_eval.py` `test_unavailable_runtime_exits_nonzero` expects no `claude` on `PATH` after removing its fake; the maintainer host running this attempt has a real `claude` on `PATH` (returncode 0 ≠ 1). All preceding validator steps printed PASS. CI on the PR is the clean-environment result |
| `./scripts/check-sensitive-files.sh .` | PASS (worktree) |
| `git diff --check` | PASS |
| `./scripts/check-sensitive-files.sh --staged .` | PASS |
| `gitleaks` | NOT_RUN: not installed |
| New public `workflow run` composition fault matrix | NOT_RUN: implementation awaits contract decision; existing helper tests are not this matrix |
| Real vendor / real state / Product acceptance | NOT_RUN: forbidden/outside scope |

## Deferred findings and next action

DF-S23-01 remains unimplemented pending the proposed binding/eligibility decision.
DF-S23-02..07 remain as recorded in [S2.3](implementation-s2-03.md#deferred_findings-deferred_not_implemented):
same-UID ref mutation, unpersisted check identity after runner loss, accumulated
worktrees, Git version requirement, Git process containment and missing public
integration resolution. CLI Git outcome observation already exists at this base;
do not interpret historical DF-S23-07 as absence of all outcome projection.
No new public ResolveIntegration command is proposed in this scope.

Next action: Rafael reviews the bounded amendment, selects A or an explicit
alternative, then implementation follows its five tasks. P-AC3/P-AC4 and #278
remain open and unaccepted. PR CI/remote delivery evidence belongs in the PR;
local checks do not imply CI or product acceptance.
