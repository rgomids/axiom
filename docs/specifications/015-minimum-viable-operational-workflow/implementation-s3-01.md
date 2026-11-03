# S3.1 — Repository outcomes and validation readiness

Attempt: `IMPLEMENT-S3-01-CODEX-01`. Scope: Tasks 015 S3.1 only.
Baseline: `2930311e03f0a2f0067c3cca1c507ca677a90871`; local and fetched origin match; clean worktree; PR #366 open Draft. Authority: AXM-42 comment `0f457c1c`; independent S2 review `5745407569`. DESIGN: NOT_APPLICABLE (domain/CLI projections only).

## Acceptance evidence matrix

| Contract | Deterministic evidence planned |
| --- | --- |
| Two repository bindings keep distinct attempt/artifact/target provenance | `TestRepositoryOutcomesTwoIndependentBindingsOneExecution`, `TestRepositoryOutcomesPreserveProvenanceAndCodec`: PASS (isolated stubs; same Execution/store, two temporary repositories, same source branch name) |
| Wrong repository/artifact/ref/path refused | `TestObserveIntegrationTwoRepositoriesAndStaleOrForeignFacts`, `TestObserveExecutionArtifactRechecksBytesAndLineage`, existing `TestExecutionOutputValidatorsRequireExactRetainedArtifact`: targeted negatives PASS; full regression pending |
| Fabricated success refused | `TestRepositoryOutcomesRefuseFabricatedSuccess`: required output absence refuses encoding/readiness; existing typed outcome/correlation negatives retained |
| Local completion without PR | `TestRepositoryReadinessReobservesLocalOutputsWithoutPR`, `TestCanonicalReadinessIgnoresLegacyLedgersAndGrantsNoEffects`: PASS; stale bytes block readiness, completed status preserved, inspection does not write state |
| Compatibility, cancellation, fence, unknown, stale/missing evidence | Format 4 and existing nested fields unchanged; round-trip and store read-back PASS; current bytes/targets rechecked; full S2 regression pending |

## Technical choices

- Derive repository outcomes from canonical attempt and integration records, rather than copy those facts into another persisted field. Format 4 bytes and ownership remain unchanged. Missing Git integration means no claimed Git target, not an invented SHA or a PR requirement.
- Re-observe artifact content/lineage and recorded Git targets at the read boundary. Validation readiness is a point-in-time observation, never publication authority or acceptance. A completed local Execution remains completed if retained evidence later disappears, but its readiness is blocked.

- Scope correction backed by the two-repository test: integration branch/worktree associations must be keyed by Repository. The previous cross-repository comparison refused the second independent binding. Attempt identity remains global; same-repository association checks remain intact.
- Codec validation now enforces the required-output invariant already enforced by `CompleteAttempt`; it rejects forged succeeded attempts, without changing the format or refusing legitimate writer output.

## Deferred findings

`N-REV-01` stays deferred to S3.4; `N-REV-02` stays deferred to S3.4/S3.5, as recorded in the independent S2 review. No change to either.

## Validation

Current checks (all exit 0): focused new tests and CLI readiness tests; `go build ./...`; `go vet ./...`; `go run ./scripts/check-architecture.go domain` and `application`; `./scripts/check-go-quality.sh format`; `python3 scripts/check-adr-governance.py`; `python3 scripts/check-automation-registry.py`; `./scripts/test-docs-offline.sh`; `git diff --check`.

The docs journey reports 26 corpus files, closed links and expected corrupted-bundle refusals; pinned v1 installer unavailable (UNVERIFIED), native Runtimes/network/source denial NOT_RUN. Full Go suite, adjacent package regression and repository validation are still running at this checkpoint. Initial missing-symbol test failures and the reproduced cross-repository branch association refusal were expected red tests and have been corrected.

No live vendor/model battery, remote product effects, merge, release, cutover or acceptance authorized. Host has no gcc; race remains NOT_RUN. Principal quota introspection is unavailable through the current tools; no runtime switch or delegated work.

## Bounded correction F1/F2

Attempt: `CORRECT-S3-01-CLAUDE-01` (Claude Code principal). Baseline `8f671f94d79a14b45400512227017d011fc2d2ac`; local, origin and PR #366 head matched with a clean worktree. Scope: findings F1 and F2 of independent review `REVIEW-S3-01-CLAUDE-01` ([PR review 5746310559](https://github.com/rgomids/axiom/pull/366#pullrequestreview-5746310559)) only.

| Finding | Correction | RED on baseline → GREEN |
| --- | --- | --- |
| F1 MAJOR: a Project Repository with no work forced `ready:false` | `Execution.Claims` marks a row as claimed for the Work Item Repository or any Repository named by an attempt or integration. Only claimed rows report `repository_result_missing` or have their path observed. Unclaimed rows stay in the report with empty `results`. | `TestRepositoryReadinessIgnoresUnclaimedProjectRepository` (two-repository Project `main`+`docs`, local run without PR, real `delivery-packet` path). RED: `ready:false`, `blockers:[docs:repository_result_missing]`. GREEN: `ready:true` with docs `results:[]`; docs path removed is not observed. Still refused: tampered or deleted `main` output, wrong `main` path, identity or artifact digest. A second Repository that claims an attempt keeps its result and reports `repository_binding_mismatch`. |
| F2 MINOR: a succeeded attempt carrying an `unknown` effect was not refused | `validateAttempt` rejects `AttemptSucceeded` with any `EffectUnknown` fact. | `TestRepositoryReadinessRefusesSucceededAttemptWithUnknownEffect` forges the stored format-4 bytes and reads them through the store and `delivery-packet`. RED: `ready:true`, `blockers:[]`. GREEN: the packet is refused, and `DecodeExecution` rejects the bytes. |

Technical decisions (scope impact: none):

- F2 is enforced at canonical validation, not with a readiness blocker. `CompleteAttempt` has downgraded unknown effects to `AttemptUnknown` since format 4 was introduced (`3c8898c`). Historical formats are only classified, never converted, so no writer-produced or historical record becomes invalid. Forged bytes now fail at the codec for every reader, not only the packet. The legitimate unknown path (`CompleteAttempt` → `AttemptUnknown`) and normal success are unchanged and covered by existing tests.
- F1 decides claims from canonical facts already present (Work Item Repository, attempt authority and integrations). It adds no persisted field, writer or store, and it does not exempt any Repository that has real claims.
- Project admission still refuses an unavailable bound Repository before readiness runs. That gate predates S3.1 and is unchanged. The unclaimed-path case is therefore exercised at `repositoryOutcomes`, the observation the packet calls.

Validation at the correction head (exit 0 unless stated): focused F1/F2 tests and adjacent `TestRepositoryReadiness*` / `TestRepositoryOutcomes*`; `go build ./...`; `go vet ./...`; `go test ./... -count=1` (all packages, including `internal/stageexecution` two-active-repository provenance and the S2 suites); `go run ./scripts/check-architecture.go domain` and `application`; `go mod verify`; `go fmt` on touched packages (one import-order fix committed). NOT_RUN locally, because this host's tool permissions refuse repository shell scripts: `check-go-quality.sh`, `validate-repository.sh`, `test-docs-offline.sh`, staticcheck and the delivery script tests. CI on the final head is their evidence. Race: NOT_RUN (no gcc). No model test battery, live vendor or remote push-effect experiment was run.

Deferred and unchanged: D1 (source branch constraint), D2 (shared integrator-root observation), `N-REV-01` (S3.4), `N-REV-02` (S3.4/S3.5). This correction is not acceptance: the next gate is an independent Codex re-review of S3.1.
