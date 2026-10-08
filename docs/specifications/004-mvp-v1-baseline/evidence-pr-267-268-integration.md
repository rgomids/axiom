# PR #267 / #268 integration evidence — 2026-10-08

## Inputs and authority

Read the published Code Review Guardian reviews and inline findings for
[267](https://github.com/rgomids/axiom/pull/267),
[268](https://github.com/rgomids/axiom/pull/268), and
[269](https://github.com/rgomids/axiom/pull/269).
Fetched `main`: `b69fa837cacaa92bee865628a09edf8797777110`.
Verified PR heads: #267 `6113d53934e525154de672a51eae095aad16f4ee`,
#268 `aad2508a65f198d4662bbfd2a1cf10bbb6330492`,
#269 `dac8efad8aa114d61b5618cd9ce3502343530346`.
Both feature branches descend from the same main baseline; #269 is exactly
one correction commit above #268. No remote mutation is authorized here.

## Local integration

`agent/268-consolidated-local` cherry-picks #269 as `63466b3`; its tree equals
#269 exactly. `agent/267-268-integration-local` merges #267 into that tree.
Seven textual conflicts were resolved without discarding either feature.
The integration preserves target selection, target/source/policy digest,
post-Check target revalidation and v1 Execution persistence. Central
`AdmitExecutionStart` replaces the old readiness-only call in the moved handler
and is checked again after policy observation, before effects.

`TestWorkflowStartRechecksAdmissionAfterPolicyCheck` archives the selected
Project during policy Check. Without the fresh admission check it reproduces
`execution_started`; with the check it refuses with `project_archived` and
preserves the exact post-archive filesystem snapshot. Existing disabled
Integration tests retain the approved resource-specific policy: Provider use
is blocked, while local Execution operations do not acquire Provider authority.

Complete skill revisions from both feature heads are retained in shared history,
including Codex/Claude receipt ownership. Canonical skill payloads and tests
retain targeting and lifecycle outputs. Specification/index conflicts preserve
both approved amendments; accepted ADRs and historical Evidence are unchanged.
Independent bounded integration/security review identified no blocking finding.

## Local validation

PASS on macOS arm64:

- `go test ./cmd/lingo ./internal/workflow ./internal/cli ./internal/codexruntime ./internal/workitem ./internal/local -count=1`
- Focused archived/disabled admission and post-policy archive regression tests.
- `go vet ./...`, `go build ./...`, `go mod verify`.
- `./scripts/validate-repository.sh .`, `./scripts/dogfood-poc.sh`.
- `git diff --check`, `gitleaks dir . --no-banner --redact`.

`81208fc99148ea921b57dcdd5f127efef8133276` is the integrated implementation
commit. Full `go test -race ./...` passed on its implementation tree. The final
Evidence update changes only this document; executable sources are identical.
No real Provider or Runtime process was invoked.

Additional PASS:

- Native macOS `./scripts/build-release-archives.sh --version 9999.0.0-acceptance.integration --output /tmp/axiom-integration-validation/candidate`, built from clean `81208fc`.
- `./scripts/test-upgrade-journeys.sh` using that candidate, published v0.9.0
  and v0.1.1 host archives, and a binary built from `v0.1.0-poc.1`:
  all steps passed, `failures=0`, including preserved state/history,
  Codex/Claude receipts, RecognizedPOC archive and idempotent rerun.
- `./scripts/check-sensitive-files.sh --staged .` and
  `gitleaks git --pre-commit --staged --redact --no-banner .` before the
  integration commit; worktree Gitleaks also passed.
- Regression red/green proof: removing only the post-policy admission check
  makes the new archive-drift test fail with `execution_started`; restoring
  it passes. The temporary removal was never committed.
- Exact integrated tree in a synthetic merge `fd7d58080a7a1067d0d5bc2436250fa1a14ff8ed`
  (first parent main, second parent `81208fc`): `./scripts/test-delivery.sh`
  passed, including repository-history parsing. Squash-shaped candidate
  `3d7ea1e7b776cbc87fbf85ddde34df97ceb93de8` (sole parent main):
  `delivery-issues.sh commits` passed. `git diff --exit-code` proves both
  candidate trees equal `81208fc`; both declare Related-Issues #137/#230,
  Completes-Issues none. These are local validation objects, not published
  merges or GitHub CI results.

## Release contract and remote gates

The delivery scripts, corrections file and CI workflow are unchanged.
`release-contract` on stacked #269 traverses #268 through first-parent history:
`b85f3b4495cb78eb3136aa091043b2e7a83db23e` and 27 other implementation commits
have Related-Issues without Completes-Issues. On consolidated `63466b3`,
`test-delivery.sh` passes 109 fixture checks before rejecting that ancestor.
This is correct strict parsing, not a defect in the three correction changes.
A main-first-parent synthetic PR merge or final squash excludes those intermediate
commits from first-parent delivery history. Validate the exact integrated tree
under that history shape; do not add 28 corrections or rewrite shared history.
Squash delivery metadata must describe actual scope, with no invented completion.

Historical CI: #267 has seven required checks passing at `6113d53`
([CI](https://github.com/rgomids/axiom/actions/runs/37728855819),
[metadata](https://github.com/rgomids/axiom/actions/runs/37728855832)).
#268 has three verify failures at `aad2508`
([CI](https://github.com/rgomids/axiom/actions/runs/37734515624)).
#269 has six required contexts passing and release-contract failing at `dac8efa`
([CI](https://github.com/rgomids/axiom/actions/runs/37772891707)).
These are not CI results for the integrated head. Native Linux/Windows and all
seven required GitHub contexts on the final integration head remain pending.

## Human next actions

1. Review and authorize any push after refreshing all remote heads. The local
   consolidation can fast-forward #268 without rewriting its existing history.
2. Obtain required human approval and explicit merge authority for #267, then
   squash it into main; retain its reviewed delivery metadata.
3. Fetch new main and update the canonical #268 branch with the consolidated
   corrections and integrated resolutions. Validate that its tree matches this
   candidate plus any newly authorized main changes; authorize its push.
4. Require all seven GitHub contexts on the exact final #268 head, independent
   human review and explicit merge authority before its squash merge.
5. Confirm #269 redundant only after read-back proves its complete correction
   delta in #268. Close it only under separate explicit authority.

No push, PR/Issue closure, self-approval, label change, protected-environment
approval, main update or release publication was performed. Human acceptance
remains distinct from technical validation.

## Final canonical #268 update after #267 squash — 2026-10-08

The maintainer explicitly authorized integrating latest main, resolving conflicts,
committing and pushing the canonical #268 branch, then verifying its seven checks.
This supersedes the earlier pending-push boundary for this update only. Merge of
#268, closure of #269 and release publication remain prohibited.

#267 was squash-merged as `ca7a3392a549c57793db0916a3aadc8b0d9c0e17`.
Its tree equals reviewed feature head `6113d53`. #268 starts from published
`63466b3f5b0f6f9daa7e2faad8e8572c3780fa5a`; a normal merge of main preserves
both shared histories without rebase or force push. The resolved executable tree
is identical to the previously validated integration candidate `7d7f77c`.

Conflicts resolved:

| File | Preserved behavior |
| --- | --- |
| `cmd/lingo/main.go` | Targeted WorkflowStart lives in execution_target.go; other lifecycle gates retained |
| `docs/specifications/004-mvp-v1-baseline/spec.md` | Both approved amendments and authority boundaries |
| `docs/specifications/README.md` | Both features and their separate Evidence |
| `internal/codexruntime/manifest.go` | Complete historical skill revisions from both branches |
| `internal/codexruntime/runtime_test.go` | Union of targeting and lifecycle output contracts |
| `internal/codexruntime/shared_history_test.go` | Both prior manifests plus integrated manifest |
| `internal/codexruntime/skills/axiom-work-item/SKILL.md` | Target-bound run and full resource lifecycle routing |

The moved handler additionally applies `AdmitExecutionStart` before policy
preview and again after policy Check. The archive-drift regression verifies no
Execution creation after archive. Exact target digest, Runtime/Profile policy,
effective context and reviewed target revalidation remain intact. Disabled
Integration rules keep the resource-specific approved distinction: reject its
Provider use, without inventing a Provider dependency for local transitions.

Existing review findings reconciled with code:

| Finding | Code/test evidence | Assessment |
| --- | --- | --- |
| CR-001 Windows permissions | `TestOperationalRoundTripIsStrictAndCanonical` uses `testfs.PrivateMode`; `TestOperationalRejectsSharedRecordPermissions` rejects inspection/commit | Correction retained; native final CI required |
| CR-002 dogfood JSON | `assert_project_repository` structurally checks available/unavailable/restored with successful project show | Correction retained; fresh local smoke passed |
| CR-003 stale link | `setState` includes local-only reconciliation when Provider already matches; digest/authority/revision/CAS tests retained | Correction retained; no redundant Provider PATCH |

The published PR body passes `delivery-issues.sh check-pr` and declares
`Related-Issues: #230` / `Completes-Issues: none`. Delivery scripts and corrections
are unchanged. Validate release history on the exact GitHub main-first-parent
merge shape; inherited intermediate commits are not the final squash history.

Fresh local runs: vet, build, module verification, repository validator, dogfood,
staged sensitive-file check, Gitleaks and diff checks passed. Full race and exact
final history results are recorded below before delivery. Final remote checks
must be tied to the new pushed head, not `63466b3` or earlier candidates.

At pre-push read-back both existing review threads were unresolved (permissions
thread outdated, stale-link thread current). This report reconciles their code
without dismissing reviews or automatically resolving reviewer threads. Human
review/thread resolution and any protected-branch approval remain separate.

`go test -race ./...` passed on the final resolved executable tree (unchanged
from `7d7f77c`; some unchanged packages reused Go's test cache). All six requested
local commands completed successfully. Exact main-first-parent release-history
validation and seven GitHub contexts are mandatory post-commit delivery gates;
the remote workflow results provide durable Evidence at the pushed head.
