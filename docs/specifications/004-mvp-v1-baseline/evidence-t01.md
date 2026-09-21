# T01 — Canonical completion and provenance Evidence

## Authority and status — 2026-09-20

- Specification 004: **Approved**. ADR-0001–ADR-0008: **Accepted**.
- Plan: **Approved**. Tasks: **Approved**.
- Human authorization: **T01 implementation only**, with the kickoff analysis
  accepted as execution baseline.
- Baseline: `main` at `ae4c6133bf25a71d7192af65bf0e8c4d9b45f0fa`, merge
  of [PR #73](https://github.com/rgomids/axiom/pull/73).
- Delivery branch: `impl/spec-004-t01-completion-provenance`. Delivery revision is
  the future commit containing this Evidence and the accompanying implementation.
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
| MVP-NFR-02 | Bounded authored text/references and 16 KiB final-output guard | `TestCompletionOutputIsBounded`, unsafe metadata tests, observed byte counts below |
| MVP-NFR-06 | Stable result references plus canonical process provenance without secrets | result/provenance tests and read-only command ledger |
| ADR-0003 | Axiom contracts remain central; Lingo CLI only projects them | AST dependency/renderer checks and package inspection |
| HD-1 T01 portion | Development builds say `development`; release identity requires clean revision-bound SemVer | provenance matrix and synthetic build-flag black box |
| User/Axiom authorship boundary | Completion summary/next accept only typed Axiom-authored text | `TestAuthoredTextRejectsTransportedUserContent`, `TestResultRejectsTransportedUserSummaryAndUnsafeMetadata` |

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

## Provenance cases and exact build inputs

| Case | Inputs | Expected canonical value |
|---|---|---|
| release | `release=true version=1.2.3 revision=abc123def456 source=clean` | `Axiom 1.2.3 abc123def456 clean` |
| development clean | `development/unavailable/unknown` plus Go VCS `revision=abc123def4567890 modified=false` | `Axiom development abc123def456 clean` |
| development dirty | same plus `modified=true` | `Axiom development abc123def456 dirty` |
| unavailable | `release=false version=development revision=unavailable source=unknown` | `Axiom development unavailable unknown` |
| invalid release | non-SemVer, unavailable revision, dirty/unknown source, or numeric prerelease with leading zero | rejected; never a release identity |

Measured black-box binary build:

```text
-trimpath
-X main.buildVersion=development
-X main.buildRevision=ae4c6133bf25
-X main.buildSourceState=clean
-X main.buildRelease=false
sha256=b72f5e9adff14dad8f095b802e79f5d4224e84d6204af58855b6e17956ad326f
```

Hash identifies this isolated Evidence build, not a release or golden artifact.

## Output bounds and mutation ledger

Observed bytes, including trailing newline, from the isolated build:

| Invocation | Exit | Bytes |
|---|---:|---:|
| `lingo version` | 0 | 113 |
| `lingo --json version` | 0 | 161 |
| `lingo project validate --slug sample` | 0 | 106 |
| `lingo --json project validate --slug sample` | 0 | 154 |
| `lingo project show --selector sample` | 0 | 189 |
| `lingo --json project show --selector sample` | 0 | 234 |

All remain below the 16 KiB renderer ceiling. Unit guards also bound authored text,
reference count/length, detail reference length, UTF-8 validity, whitespace, and
control characters.

After isolated fixture setup, all six read-only invocations above ran against the
same portable, local-state, and Repository roots:

```text
before=0db031bac7f89b511045c43dcdfcc62b6dfb88397fa1758cb37e76bc7d6258a7
after=0db031bac7f89b511045c43dcdfcc62b6dfb88397fa1758cb37e76bc7d6258a7
unchanged=true
```

The ledger hashes sorted file identities/content. It supports zero-write behavior;
it is not a filesystem-concurrency or recovery claim. Unit/black-box tests also
verify missing validation roots stay absent. No Git, network, Provider, Runtime
installation, shell-profile, release, or external mutation occurred.

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
- Worktree sensitive-file checker passed. Gitleaks scanned approximately 1.85 MB
  and reported no leaks. Scanners complement structural controls; they do not prove
  absence of every secret.

## Reproduction and executed results

Environment: macOS 27.0 build 26A428, Darwin arm64,
`go version go1.26.1 darwin/arm64`.

Initial TDD construction:

```text
go test ./internal/provenance ./internal/completion
exit 1 — expected: packages had tests but no implementation files/types
```

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
gitleaks dir . --no-banner --redact
for script in scripts/*.sh; do bash -n "$script" || exit; done
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
| Gitleaks | Exit 0; no leaks found |
| Shell syntax | Exit 0 for every `scripts/*.sh` file |
| Diff whitespace check | Exit 0 |

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
- Build hashes and mutation-ledger hashes are per-run observations, not stable
  goldens.
- Staged-only sensitive scanning was not run because no index/commit operation was
  required. Worktree scanning and Gitleaks covered all current tracked/untracked
  files.

**Return to human T01 implementation review. Do not start T02 or any successor
without a new explicit human authorization.**
