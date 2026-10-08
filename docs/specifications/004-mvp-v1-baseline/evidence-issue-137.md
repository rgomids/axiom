# Issue #137 — Safe execution-targeting Evidence

Date: 2026-10-08 (America/Sao_Paulo). Baseline:
`b69fa837cacaa92bee865628a09edf8797777110` (`main`, v0.9.0); fetched `origin/main`
matched that commit. Working tree was clean before implementation. The initial implementation phase
performed no commit, push, merge, release, Issue closure, label edit or real
Provider mutation. The maintainer subsequently authorized creation of a Pull
Request, including the required scoped commit and branch push.

Scope: T137-01–T137-06, as approved in the maintainer's implementation request.
[Specification/Plan/Tasks](issue-137-execution-targeting.md) retain requirements
separately from implementation details and this actual Evidence.

Implementation status: T137-01–T137-06 completed locally, with AC-001–AC-010
verified in controlled fixtures. One required architecture check remains FAIL on
both baseline and implementation; no scoped Major/Blocker was found. This is not
human acceptance or an all-green validation claim.

## Baseline contract gaps (T137-01)

Before production edits, the following command reproduced scoped failures:

```sh
go test ./internal/workflow ./cmd/lingo -run 'TestExecutionTargetRejects|TestWorkflowStartValidatesTarget' -count=1
```

| Baseline behavior | Requirement | Reproduced regression |
|---|---|---|
| Direct lifecycle calls with Number 0/-1 or an unlinked target returned ready Runtime preview | FR-001/003/005; AC-001/008 | `TestWorkflowStartValidatesTargetBeforeRuntimePreview` failed for all three cases |
| Direct workflow accepted generic text/zero/leading-zero selectors and a loader returning a different item/provider/resource | FR-001/003/005; AC-001/008 | `TestExecutionTargetRejectsMalformedAndMismatchedSelectors` failed for malformed nonempty selectors and all mismatched-linkage cases; missing selector already refused |
| The above previews for Numbers 0, -1 and 7 returned the same digest | FR-004/008; AC-007/008 | The same failing lifecycle test observed equal policy-only digests; `runtimeapplication.Preview` contains no Work Item or Repository |
| Canonical/alias skills required Project collection despite #233 | FR-007; AC-003/004/010 | Baseline skill inspection; now guarded by shared-run parity and executable protocol/discovery tests |

No confirmed gap was invented for #233's existing precedence, persisted Execution
v1 or CLI required-selector parsing. Their existing tests were retained.

## Acceptance traceability

Results below refer to local controlled application/CLI/store fixtures and
isolated Runtime/Provider doubles. They do not establish live external execution
or human acceptance.

| AC | Implementation | Reproducible test | Result / Evidence |
|---|---|---|---|
| AC-001 explicit Work Item | `workflowStartInput`, shared workflow resolver | `TestWorkflowStartValidatesTargetBeforeRuntimePreview`, `TestExecutionTargetRejectsMalformedAndMismatchedSelectors`; existing CLI parser tests | PASS; missing/malformed/conflicting inputs refuse, snapshots unchanged |
| AC-002 valid explicit target | effective UUID resolution, `ValidateTarget`, reviewed `Start` | `TestWorkflowStartReviewBindsCanonicalTargetAndContext`, `TestAuthoredPolicyPreviewThenStartBothRuntimes` | PASS; canonical persisted item/Runtime and equivalent start |
| AC-003 session winner | #233 resolver plus retained CLI selector-origin marker | `TestWorkflowStartEffectiveProjectSourcesAndDisclosure`, `TestWorkflowStartStaleContextReviewAndPersistedIdentity` | PASS; session disclosed and reviewed |
| AC-004 local default | same resolver | `TestWorkflowStartEffectiveProjectSourcesAndDisclosure`, `TestWorkflowStartStaleContextReviewAndPersistedIdentity` | PASS; default disclosed and successful confirmed start |
| AC-005 explicit precedence | always resolve original explicit selector through #233 | `TestWorkflowStartEffectiveProjectSourcesAndDisclosure`, `TestEffectiveProjectPrecedenceMatrix` | PASS; explicit bypasses stale lower preference |
| AC-006 fail closed winner | #233 resolver, canonical UUID lookup | `TestWorkflowStartEffectiveProjectSourcesAndDisclosure`, `TestWorkflowStartAmbiguousLinkAndProjectFailClosed`, `TestProjectContextFailsOnStalePortableRevision` | PASS; unresolved/stale/ambiguous winner cannot fall back |
| AC-007 visible target | `ExecutionTargetView`, Runtime completion renderer | `TestWorkflowStartEffectiveProjectSourcesAndDisclosure`, `TestWorkflowStartReviewBindsCanonicalTargetAndContext` | PASS; JSON/human disclose exact UUID/source/item; no local path disclosure |
| AC-008 no invalid execution effects | full-target digest, fresh policy/context/link checks before allocation/Create | `TestReviewedExecutionTargetRevalidatedBeforeAllocation`, `TestWorkflowStartRejectsTargetDriftDuringPolicyCheck`, `TestWorkflowStartNeverSearchesAnotherProjectForWorkItem`, `TestWorkflowStartRequiresReviewedFreshPolicy`, `TestExecutionStartCancellationAndRecoveryPreserveState`, `TestS4CancelledOperationHasNoEffect` | PASS; no Create/Save/allocator/projection effects for checked failures; local snapshots preserved |
| AC-009 immutable Execution | unchanged State v1/store and equivalence rules | `TestExecutionIdentityConflictsPreservePersistedState`, `TestWorkflowStartStaleContextReviewAndPersistedIdentity`, `TestExecutionStartCapturesEffectiveProjectBeforeContextChanges`; local-store/frozen corpus tests | PASS; original target/Runtime/history/provenance retained, explicit original Project still addresses it |
| AC-010 skill equivalence | identical canonical/alias run block and shared embedded integration | `TestWorkItemRunExecutionTargetParity`, `TestExecutionTargetPreviousSkillSetOwnedByBothRuntimes`; existing skill discovery/parser/protocol black-box tests | PASS; Codex/Claude ownership and exact reviewed selector routing |

Additional preview/replay coverage rejects a different linked Work Item, policy-only
old token, changed winning source with the same UUID, changed Repository binding,
changed URL/state, changed Runtime executable/configuration, and conflicting
Execution selectors. A callback during policy `Check` changes linkage or context;
replay requires `stale_preview` and leaves the post-drift state untouched. A duplicate
observed slug refuses; a UUID disambiguates. Same-number linkage in two resources
refuses the numeric form while the qualified selector succeeds.

## Validation commands and actual results

| Command | Result | Actual observation |
|---|---|---|
| `go test ./internal/cli ./internal/workflow ./cmd/lingo -count=1` | PASS | Integrated targeting, parser, context, both Runtime fixtures and black-box protocol tests |
| `go test -race ./...` | PASS | All package results passed, including durable-state inventory/frozen corpus and shared Runtime ownership |
| `go vet ./...` | PASS | No diagnostics |
| `go build ./...` | PASS | No build errors |
| `go mod verify` | PASS | `all modules verified` |
| `go run ./scripts/check-architecture.go domain` | PASS | 11 domain source/test files within the pure allowlist |
| `go run ./scripts/check-architecture.go application` | FAIL (baseline) | `non-allowlisted application symbol: context.WithValue` at unchanged `internal/projectapp/effective_context.go:43` |
| `./scripts/validate-repository.sh .` | PASS | Bootstrap, agent/ADR/label-policy and sensitive-file checks passed |
| `./scripts/dogfood-poc.sh` | PASS after fixture reconciliation | Controlled workflow completed, revision 17, eight global skills, selected Codex; synthetic Provider projections only |
| `gitleaks dir . --no-banner --redact` | PASS | No leaks found |
| `go test -race ./internal/workflow -count=1` | PASS | Final cancellation/recovery refusal assertions passed |
| `go test -race ./cmd/lingo -run 'TestWorkflowStartEffectiveProjectSourcesAndDisclosure\|TestWorkflowStartRejectsTargetDriftDuringPolicyCheck' -count=1` | PASS | Final CLI confirmation succeeded for default/session/explicit; drift refusals passed |
| `go test ./internal/local ./internal/compatibility ./internal/codexruntime -run 'TestEveryV1WriterIsRecognizedByInventory\|TestEverySupportedRecordLoadsThroughItsCanonicalStore\|TestStableCorpus\|TestEveryPublishedSharedRevisionStaysOwned\|TestWorkItemRunExecutionTargetParity\|TestExecutionTargetPreviousSkillSetOwnedByBothRuntimes' -count=1` | PASS | Explicit compatibility, inventory, frozen corpus and shared receipt checks |
| `git diff --check` | PASS | No whitespace errors |

The application architecture failure was reproduced against clean baseline
source extracted with `git archive HEAD scripts/check-architecture.go
internal/projectapp internal/project` into a fresh temporary directory, then
`go run ./scripts/check-architecture.go application`. It returned the exact same
failure. #137 changes no `internal/projectapp` source or architecture checker;
the existing allowlist was not weakened. This check remains a disclosed validation
failure requiring separate baseline reconciliation, not a new scoped finding.

The first dogfood invocation failed at its old absent-policy fixture because its
Work Item had never been linked. The reconciled fixture retains all original
Runtime blockers and reviewed local-selection authority; the complete rerun passed.
Routine command logs were kept outside the repository during validation; the
commands, observed outcomes and named assertions here are the durable Evidence.

## Review and compatibility

Delegation had bounded value: one independent read-only contract/security review,
and one isolated skill/history implementation unit. Both inherited the
Orchestrator's model and effort explicitly; no model names entered product policy.
The Orchestrator integrated and reran the checks.

Independent review inspected production diff and new target tests, ran
`go test ./internal/cli ./internal/workflow ./cmd/lingo ./internal/codexruntime -count=1`
and `git diff --check`: PASS. No scoped Major/Blocker identified. Its suggested
additional drift-during-Check coverage was implemented and passed afterward.

- Execution persistence remains v1; no store, encoding, migration or state-root
  inventory change. Existing OPEN/CLOSED/URL equivalence semantics remain intact.
- Previous eight-skill revision and complete receipt remain recognized by both
  Runtime integrations. Current payload contract additionally preserves
  `executionTarget`; CLI argument discovery/parser requirements remain unchanged.
- Older policy-only start tokens require fresh workflow-start review. The
  independent Runtime/Profile preview API and digest are unchanged.
- Reviewed target snapshots are ephemeral. They grant no execution, local,
  Provider or human-acceptance authority. Existing gate and policy checks remain.
- Dogfood's absent-policy fixture now first links its explicit synthetic Work
  Item in that Project via the existing reviewed selection/local-authority protocol;
  otherwise the new correct target preflight stops before reaching Runtime policy.
- Accepted architecture and ADRs are unchanged. Bounded filesystem guarantees
  remain those of ADR-0005/0007; this work makes no stronger arbitrary same-UID
  interleaving or physical-durability claim.

## Limits and next action

Native validation is on macOS arm64 with Go 1.26.1. Live Codex/Claude processes and real Provider
execution, native Linux/Windows validation and remote CI are UNVERIFIED here.
Runtime stubs are observed by Lingo but are never launched by start; Provider
fixtures are isolated synthetic doubles. No acceptance or publication is inferred.

The maintainer authorized PR creation after the local implementation report.
The scoped changes are being committed and pushed for review under that authority.
Review this amendment, source/tests/skills and these results; human acceptance,
merge, release and Issue closure remain separately authorized gates.
