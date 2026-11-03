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
