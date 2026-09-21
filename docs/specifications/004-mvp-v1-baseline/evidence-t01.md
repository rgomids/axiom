# T01 — Canonical completion and provenance Evidence

## Authority and status — 2026-09-20

- Specification 004: **Approved**. ADR-0001–ADR-0008: **Accepted**.
- Plan: **Approved**. Tasks: **Approved**.
- Human authorization: **T01 implementation only**, with the kickoff analysis
  accepted as execution baseline.
- Baseline: `main` at `ae4c6133bf25a71d7192af65bf0e8c4d9b45f0fa`, merge
  of [PR #73](https://github.com/rgomids/axiom/pull/73).
- Delivery branch: `impl/spec-004-t01-completion-provenance`. Technical review
  started from PR head `494b12f97c1f`; the reproducible delivery state is the
  final PR head containing this Evidence and its remediation.
- **T01 Implementation: Completed. T01 Evidence: Produced. T01 Human Acceptance:
  Pending. T02–T25: Not authorized.**

Checks, review, commit, merge, or this Evidence do not provide human acceptance or
authorize a successor Task.

## Delivered vertical slice

`internal/completion` owns exactly six semantic result fields: `status`, `result`,
`references`, `next`, `details`, and `provenance`. Its status set is closed to:

```text
success
failure
validation_failure
denied_authority
partial
interrupted
retryable_failure
```

Classification consumes application facts before presentation. Ambiguous facts,
false success, `partial` without a confirmed reference/next action, and retryable
failure without a next action are rejected. Result values own detached reference
copies, require Axiom-authored bounded summary/next text, and carry one valid
canonical provenance value.

`internal/provenance` owns product identity `Axiom`, release semantic version or
`development`, short embedded/observed revision or `unavailable`, and source state
`clean`, `dirty`, or `unknown`. A release identity requires valid SemVer, clean
source, and an available revision. Invalid release metadata fails closed to the
composition root's explicit development/unknown value; it never invents a release.
Go VCS build settings are observed only when explicit metadata is unavailable.

`internal/cli` renders the same immutable completion result as bounded human or
JSON output. Renderers read fields only; they do not classify outcomes or hardcode
product/version values. Rendering failure returns a presentation failure exit while
leaving the original status and confirmed references unchanged.

Initial read-only surfaces:

- `lingo version` and `lingo --json version`;
- `lingo [--human|--json] project validate ...`;
- `lingo [--human|--json] project show ...`.

Existing domain effects remain unchanged. `project show` returns stable Project and
Repository identifiers as references, not machine-local paths. Remaining POC
commands retain historical output until their separately authorized MVP Tasks.

Review remediation keeps classification out of renderers and command handlers:

- parser/required-input failures for `project validate` and `project show` create
  `validation_failure` directly from central completion/provenance contracts and
  never invoke an application service with invented empty input;
- `project show` maps selector/not-found, invalid/recovery state, unavailable
  Project/Repository bindings, interruption, and operational failure to distinct
  truthful canonical outcomes without transporting raw OS errors or local paths;
- `scripts/dogfood-poc.sh` consumes canonical `project show` fields and rejects the
  removed legacy category/path shape.

The source installer now injects `development`, short revision, and clean/dirty
source state into the binary. It retains existing owned-destination, receipt,
idempotency, conflict, and no-shell-profile behavior. No release/distribution path
was added.

## Requirement traceability

| Requirement / decision | Implementation | Executable Evidence |
|---|---|---|
| FR-016 truthful failure | Closed status classification; ambiguous/confirmed effects cannot become success | `TestStatusEffectClassificationMatrix`, `TestClassificationRejectsAmbiguousOrFalseSuccess`, `TestRendererFailurePreservesConfirmedEffect` |
| FR-023 confirmed-effect truth | Immutable references; partial requires confirmed reference and next action; renderer cannot mutate result | `TestResultPreservesSixCanonicalFieldsAndCopiesReferences`, `TestResultRequiresPartialAndRetryContext`, `TestRendererFailurePreservesConfirmedEffect` |
| AC-12 T01 portion | Seven statuses and equivalent human/JSON projections | `TestCanonicalStatusesAreClosed`, `TestCompletionGoldenMatrix` |
| AC-15 T01 portion | Central released/development/dirty/unavailable provenance, consumed by initial surfaces | `TestBuildProvenanceMatrix`, `TestReleaseProvenanceFailsClosed`, `TestExecutableVersionHumanJSONAndBuildProvenance` |
| MVP-NFR-01 | Closed classification, stable field order, one process value, deterministic renderers | classification/provenance tables and golden output |
| MVP-NFR-02 | Bounded authored text/references and 16 KiB final-output guard | `TestCompletionOutputIsBounded` and unsafe metadata tests |
| MVP-NFR-06 | Stable result references plus canonical process provenance without secrets | result/provenance tests and `TestCanonicalReadOnlySurfacesDoNotMutateState` |
| ADR-0003 | Axiom contracts remain central; Lingo CLI only projects them | AST dependency/renderer checks and package inspection |
| HD-1 T01 portion | Development builds say `development`; release identity requires clean revision-bound SemVer | provenance matrix and synthetic build-flag black box |
| User/Axiom authorship boundary | Completion summary/next accept only typed Axiom-authored text | `TestAuthoredTextRejectsTransportedUserContent`, `TestResultRejectsTransportedUserSummaryAndUnsafeMetadata` |
| Parser failure boundary | Rejected flags/missing slug or selector never call `Validate`/`Show`; rejected values are not echoed | `TestCanonicalProjectParserFailuresDoNotCallApplicationServices` |
| Project inspection truth | Resolution causes preserve distinct status/result/next semantics | `TestProjectShowClassifiesResolutionCauses` and dogfood success/not-found/unavailable cases |

AC-12 Runtime convergence remains T15/T24 work. AC-15 Provider, Runtime, Markdown,
Evidence, and generated source/text propagation remains with its approved successor
owners. T01 supplies their central contract only.

## Golden status and exit matrix

`TestCompletionGoldenMatrix` renders every row through both renderers from the same
`completion.Result`:

| Status | Human meaning equals JSON | Current CLI exit |
|---|---|---:|
| `success` | yes | 0 |
| `failure` | yes | 1 |
| `validation_failure` | yes | 1 |
| `denied_authority` | yes | 1 |
| `partial` | yes; confirmed reference and next action retained | 1 |
| `interrupted` | yes; persisted reference and next action retained | 2 |
| `retryable_failure` | yes; safe retry action retained | 1 |

Exit mapping preserves the existing `0`/`1`/`2` process convention. It is not a
new architecture decision and does not redefine status meaning.

## Provenance cases

| Case | Inputs | Expected canonical value |
|---|---|---|
| release | `release=true version=1.2.3 revision=abc123def456 source=clean` | `Axiom 1.2.3 abc123def456 clean` |
| development clean | `development/unavailable/unknown` plus Go VCS `revision=abc123def4567890 modified=false` | `Axiom development abc123def456 clean` |
| development dirty | same plus `modified=true` | `Axiom development abc123def456 dirty` |
| unavailable | `release=false version=development revision=unavailable source=unknown` | `Axiom development unavailable unknown` |
| invalid release | non-SemVer, unavailable revision, dirty/unknown source, or numeric prerelease with leading zero | rejected; never a release identity |

## Output bounds and read-only behavior

`TestCompletionOutputIsBounded` enforces the 16 KiB renderer ceiling. Constructor
tests bound authored text, reference count/length, detail reference length, UTF-8,
whitespace, and control characters. `TestCanonicalReadOnlySurfacesDoNotMutateState`
snapshots portable, local-state, and Repository trees around human and JSON
`project validate`/`project show` invocations. Missing-root tests prove validation
does not create roots or lock files. These tests establish the T01 read-only
boundary; they do not claim filesystem concurrency or recovery behavior.

## Static architecture and security Evidence

- `internal/completion/boundary_test.go` allowlists only pure standard-library
  helpers plus canonical provenance; no filesystem, process, network, Provider,
  Runtime, or presentation import.
- `internal/provenance/boundary_test.go` permits validation/build-input helpers
  only; build observation stays in the composition root.
- `internal/cli/completion_structure_test.go` inspects renderer AST bodies and
  rejects completion-status classification or hardcoded `Axiom`/`development`.
- Authorship sentinels prove user text cannot enter the Axiom-authored result/next
  fields through the canonical constructors.
- Metadata tests reject invalid UTF-8 boundaries, control/newline injection,
  oversized values, unknown statuses/states, and unsafe release claims.
- Worktree and staged sensitive-file checks passed. Gitleaks 8.30.1 scanned the
  worktree with redaction and reported no leaks. Scanners complement structural
  controls; they do not prove absence of every secret.

## Reproduction and executed results

Environment: macOS 27.0 build 26A428, Darwin arm64,
`go version go1.26.0 darwin/arm64`.

Final commands from repository root:

```bash
go test -race -shuffle=on -count=3 \
  ./internal/completion ./internal/provenance ./internal/cli ./cmd/lingo
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
./scripts/test-install-axiom.sh
./scripts/dogfood-poc.sh
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
./scripts/check-sensitive-files.sh --staged .
gitleaks dir . --no-banner --redact
for script in scripts/*.sh; do
  bash -n "$script" || exit 1
done
git diff --check
```

| Check | Result |
|---|---|
| Targeted race/shuffle, three repetitions | Exit 0; four packages passed |
| Full Go suite | Exit 0; eleven packages passed |
| `go vet`, build, module verification | Exit 0; `all modules verified` |
| Installer behavior regression | Exit 0 |
| Existing isolated POC dogfood regression | Exit 0; PATH notice only, all state/provider fakes remained under temporary roots |
| Repository validator | Exit 0; package structure, validator regressions, sensitive scan, bootstrap checks passed |
| Worktree sensitive-file scan | Exit 0; passed |
| Staged sensitive-file scan | Exit 0; passed against the complete proposed index |
| Gitleaks 8.30.1 worktree scan | Exit 0; approximately 1.86 MB scanned; no leaks found |
| Shell syntax | Exit 0 for every `scripts/*.sh` file |
| Diff whitespace check | Exit 0 |

At reviewed head `494b12f97c1f`, POC verification run `35553059841` failed on
both macOS and Ubuntu because dogfood still asserted the legacy `project show`
event. The remediated dogfood passes locally and now asserts canonical success,
not-found, and repository-unavailable results. GitHub Actions status is live
provider state and must be checked at the final PR head; it is not inferred from
local validation or embedded as self-acceptance in this document.

## Review findings and limitations

Review lanes covered Specification compliance, architecture, verification, and
security. No blocker, critical, or major finding remains.

- Only `version`, `project validate`, and `project show` use the canonical result.
  This is the approved T01 initial surface, not a claim of full CLI/Runtime
  convergence. Historical POC event shapes remain visible elsewhere.
- `details` is represented but never materialized. T02 remains unauthorized.
- Runtime, Provider comments, workflow/Execution persistence, filesystem
  publication/recovery, release/distribution, and POC migration were not changed or
  tested as T01 behavior.
- Release provenance is proven with synthetic build flags only. No release, tag,
  archive, checksum publication, signing, or authenticity claim occurred.
- This run provides macOS arm64 implementation Evidence. T01 has no native target
  matrix ownership; T22/T24 remain responsible for approved exact-target Evidence.
- No per-run binary or mutation-ledger hash is promoted as a stable golden.

**Return to human T01 implementation review. Do not start T02 or any successor
without a new explicit human authorization.**
