# T04 — Implementation Evidence

## Invalid UTF-8 producer review response — 2026-09-16

**T04: Ready for human re-review, not Accepted. T05–T21: Not started. No merge.**

Authority: explicit instruction to resolve only the latest human PR #9 finding,
submitted by `rgomids` at `2026-09-16T18:30:43Z` against
`97eda90b5d23a5c963eaea9b0eb0bdbf9b12a344`: **Needs changes — 1 Major**.
Preflight ran `git fetch --all --prune`, `git status`, `git branch --show-current`,
`git rev-parse HEAD` and `git rev-parse origin/main`. The worktree was clean on
`codex/t04-local-installation-record-codec`; local, remote and PR HEAD matched the
reviewed SHA, with no later commit. Main remained
`1de3b02d97818138c32f6b2a07555cfe1ffd4a9b`. PR reviews and comments confirmed
the invalid-UTF-8 finding was the latest human review.

**New implementation HEAD: `612637d6f940dac6485e5a2fcc9d900f18d506d2`.**
This subsequent documentation commit records that tested implementation SHA;
the final review HEAD is obtained with `git rev-parse HEAD` after this commit.
All historical Evidence below is preserved verbatim. This section supersedes
the earlier compatibility conclusion where it omitted malformed producer strings.

### Root cause and minimal correction

`ValidDocumentName` checked lexical syntax but not UTF-8 validity. A caller could
pass `Document.Name` containing byte `0xff` alongside a valid manifest/Project;
`ReadSnapshot` accepted it and `Digests()` emitted the exact invalid Go string.
T04 correctly refused that metadata as `invalid_artifact`.

The producer now requires `utf8.ValidString(name)` before accepting any document.
Invalid bytes are rejected, never normalized or replaced. Byte `0xff` is invalid
UTF-8; the real U+FFFD rune (`\ufffd`, UTF-8 bytes `ef bf bd`) is valid text and
remains accepted. No U+FFFD blacklist was added to document/artifact names.

Removing T04's `utf8.ValidString(a.Name)` would be incorrect: `encoding/json`
can replace invalid bytes with U+FFFD, silently changing the metadata. That guard,
strict JSON input checks and surrogate-escape validation remain unchanged. Manual
invalid `RecordState` construction still fails, retaining defense in depth.

Reviewed `internal/project/validation.go`: domain references have their existing
portable lexical rules; the demonstrated escape is the supplied document boundary,
including unreferenced documents. T02 checks every supplied name even with an
injected codec, so no domain modification is needed to enforce this invariant.
No filesystem rule, normalization or architectural contract was introduced.

The application boundary checker initially rejected the new standard-library
import. Its allowlist now permits only `unicode/utf8.ValidString`, a pure predicate
with no I/O. Added fixtures accept that operation and reject an unreviewed symbol
from the same package; existing dependency/effect restrictions remain intact.

### Regression and cross-boundary review

- `TestReadSnapshotRejectsInvalidUTF8DocumentName` directly tests T02 with its
  existing pure codec/valid Project helper and the requested `context/` + `0xff`
  + `.md` bytes. Requires issues, nil `Digests()` and predicate rejection.
- `TestReadSnapshotRejectsInvalidUTF8BeforeLocalRecord` uses the real manifest
  codec with a valid minimal manifest and the raw invalid document name. Requires
  failure and nil metadata before any local-record construction; no serialization
  can repair the test input first.
- `TestReadSnapshotPreservesValidReplacementRune` explicitly accepts
  `context/api\ufffdlegacy.md` and checks exact digest-name preservation in T02.
- Existing `TestPortableArtifactDigestRecordRoundTrip` remains green for `#`, `?`,
  valid U+FFFD, composed/decomposed Unicode and other valid names through
  `manifest.Decode` → valid Project → `ReadSnapshot` → `Digests` → `NewRecord` →
  `EncodeRecord` → `DecodeRecord`, checking exact metadata and stable encoding.
- Existing `TestArtifactNameUnicodeWirePreservation` still rejects manual invalid
  UTF-8 `RecordState` and malformed wire Unicode, while preserving genuine U+FFFD.

Explicit review of the six requested boundaries confirms: `ValidDocumentName`
requires valid UTF-8; `ReadSnapshot` validates every name before sealing a snapshot;
`Digests` emits only the fixed UTF-8 manifest name and copied validated names, or
nil for an invalid snapshot. `NewRecord` retains its UTF-8 guard and shared lexical
predicate. `EncodeRecord` serializes a sealed record; `DecodeRecord` checks Unicode
before decoding and revalidates metadata. Accepted names survive as exact Go strings.

**Every ArtifactDigest.Name emitted by an ArtifactSnapshot validated by T02 is
valid UTF-8 and representable losslessly by the T04 codec.** This is a name
representation invariant; existing whole-record size/depth/node limits and other
metadata validation remain in force.

### Validation results

Platform: macOS 26.6.2 (25G83), arm64; Go 1.26.1 darwin/arm64.
All Go commands used `export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`.
No dependencies or tools installed.

| Command / check | Exit | Result |
|---|---|---|
| Targeted new producer and real-codec regressions before fix | 1 | Expected red: invalid names accepted and reusable digests emitted; U+FFFD/#/? positives passed |
| `go test ./internal/projectapp` | 0 | Passed after producer fix |
| `go test ./internal/local` | 0 | Passed, including composition and manual-invalid-state defense |
| `go test ./...` | 0 | All four packages passed |
| `go test -cover ./...` | 0 | local 95.5%; manifest 96.0%; project 99.6%; projectapp 100.0% |
| `go test -race ./...` | 0 | All four packages passed |
| `go vet ./...` | 0 | Passed |
| `go build ./...` | 0 | Passed |
| `go mod verify` | 0 | All modules verified |
| `go run ./scripts/check-project-domain.go` | 0 | Seven domain source/test files passed |
| `bash scripts/test-check-project-domain.sh` | 0 | Pure fixture accepted; seven forbidden fixtures rejected |
| `go run ./scripts/check-projectapp.go` | 0 | Seven application source/test files passed after narrow allowlist update |
| `bash scripts/test-check-projectapp.sh` | 0 | Two positive fixtures and twelve negative fixtures passed; new positive fixture was red before allowlist update |
| `go test -fuzz=FuzzRecordRoundTrip -fuzztime=20s -parallel=2 ./internal/local` | 0 | 467,168 executions; no failure |
| Projectapp fuzz inventory | — | No existing fuzz target in `internal/projectapp`; none skipped |
| `./scripts/validate-repository.sh .` | 0 | Repository/harness and Bash regression checks passed |
| `./scripts/check-sensitive-files.sh .` | 0 | Worktree passed |
| `git diff --check` | 0 | Passed |
| `bash -n scripts/test-check-projectapp.sh` | 0 | Changed shell script syntax passed |
| Staged paths/content review, `git diff --cached --check`, `./scripts/check-sensitive-files.sh --staged .` | 0 | Scoped implementation commit reviewed and passed |

### Scope and review status

Production change is confined to `internal/projectapp/snapshot.go`; other changes
are regressions, the narrow checker allowance/fixtures, CHANGELOG and this Evidence.
README and `docs/commands.md` already describe the unchanged stack and commands.
Specification, Plan, Tasks, Clarifications, ADRs, H12, local codec production files,
domain, portable codec, revision calculations and dependencies are unchanged.
No T05–T21, persistence, discovery, physical path checks, symlink/confinement/TOCTOU,
CAS, atomic writes, migration, CLI or Runtime/Provider/network execution added.

No blocking finding identified in scoped self-review. Ready for human re-review
on PR #9, without self-approval or merge. Linux/filesystem behavior is unverified.
Tool lookup confirmed `gitleaks`, `markdownlint`, `markdownlint-cli2`, `lychee` and
`shellcheck` unavailable; their dedicated coverage is not claimed. All mandatory
checks passed. Finite fuzzing and coverage do not prove universal correctness.

## Artifact digest compatibility review response — 2026-09-16

**T04: Ready for human re-review, not Accepted. T05–T21: Not started. No merge.**

Authority: explicit instruction to resolve the latest PR #9 human re-review
(2026-09-16 18:03:06 UTC, reviewed commit
`82247b086b0c6e6d6dd382bee2dd4f4a20cf165e`): **Needs changes — 1 Major**,
T04 rejecting document names accepted by the portable snapshot producer.
`git fetch --all --prune` completed before changes; branch
`codex/t04-local-installation-record-codec` was clean and matched remote/PR HEAD.
Current main baseline is `1de3b02d97818138c32f6b2a07555cfe1ffd4a9b`, already
an ancestor of that HEAD. All prior Evidence below is preserved verbatim.

### Finding, cause and correction

`project.relativeDocument` validates domain document references; T03 decodes a
strict manifest to that Project. T02 `ReadSnapshot` additionally validates supplied
document names, rejects ambiguous/duplicate names and requires referenced documents.
`ArtifactSnapshot.Digests()` emits `axiom.yaml` plus those exact document names.
T04 instead used its own `artifactName`, including a `?#` ban and `cleanText`.
Thus a valid producer snapshot could fail `local.NewRecord` as `invalid_artifact`.

- Reuse the existing T02 predicate as `projectapp.ValidDocumentName`; its body and
  `ReadSnapshot` behavior are unchanged. T01/domain and T03/manifest are unchanged.
  This exposes existing validation for reuse, without changing accepted portable
  values, snapshot completeness, digest computation, ports or revision semantics.
- Delete T04's duplicate lexical grammar and its `path` dependency/allowlist entry.
  Keep the manifest digest requirement, duplicate detection, UTF-8 representability
  and existing wire resource limits. Metadata grants no filesystem authority.
- The same comparison and red regression exposed another manifestation of this
  finding: portable document names containing escaped tab/control characters or
  literal U+FFFD are valid upstream, yet local `cleanText` rejected them. This was
  reported before production changes. Artifact names now follow the existing T02
  rules; local paths and logical references retain their existing text protections.
- To preserve valid U+FFFD without accepting lossy Unicode repair, check JSON Unicode
  escape code units before token decoding: reject isolated/reversed/mismatched
  surrogates; accept valid pairs and genuine literal/escaped U+FFFD. Malformed UTF-8
  remains rejected before JSON encoding/decoding. No replacement or normalization.
  This bounded scan uses the existing input-byte limit and standard library only.

Traversal, absolute/rooted references, backslashes, colon, interpolation markers,
NUL/CR/LF, dot/empty components, duplicate names and reserved manifest-document
collisions retain their existing responsible domain/snapshot checks. Local decoding
reuses snapshot lexical validation; it does not establish physical containment.
No approved contract or architectural decision required amendment.

### Cross-contract comparison

| Value | Producer / consumer comparison and outcome |
|---|---|
| `ArtifactDigest` | Only validated `ArtifactSnapshot.Digests()` is a sealed producer here. Exact names now share the T02 check; all 32 digest bytes survive hex serialization. Manifest digest and name uniqueness remain enforced. UTF-8/wire resource checks remain encoding constraints; arbitrary Go strings from unchecked proposals are not proof of valid portable text. |
| `RepositoryBinding` | T02 declares metadata and copies proposals; no implemented validated checkout-binding producer exists. T04 retains key uniqueness, logical-reference and path-text checks; arbitrary local path spelling remains preserved. No additional validated-producer incompatibility found. |
| `CredentialBinding` | T02 explicitly calls these untrusted strings requiring T04/T14 validation. Source/item/reference validation remains T04-owned; unsupported source kinds and unresolved source/item gaps still round-trip. No secret resolution. |
| `RuntimeBinding` | T02 declares an observer port and metadata; no runtime execution or validated producer is implemented. T04 retains optional zero binding, required ID for configured binding, literal path metadata and enum/time checks. |
| `Observation` | All three availability values and six basis values map both ways, including zero observation. Existing regression covers their Cartesian product. RFC3339Nano representability is a local wire check; T02 does not validate arbitrary `time.Time` proposals or observation truth. |
| `AttemptMetadata` | T02 carries caller-supplied correlation/time, without an implemented validated attempt producer. T04 retains absent zero attempt, reference checks and representable timestamp validation. |
| `PortableRevision` | Valid snapshot revisions expose the same 32 bytes required by T04; `RecordedPortableRevision` restores them exactly. Invalid/missing revisions remain invalid for an existing record, consistent with `NewLocalSnapshot`. |
| Project ID / observed slug | `Project` uses `ValidateIdentity`; T04 invokes the same rules and remaps diagnostics only. Existing UUID/slug regressions remain green; the cross-package test derives both values from the decoded Project. |

`NewLocalSnapshot` seals identity, source capability and present portable revision;
it does **not** certify all proposal metadata. Its contract explicitly assigns safe
metadata/wire validation to T04. No further incompatibility involving a validated
producer was established. No binding/reference/time contract was changed.

### Regression Evidence

- `TestPortableArtifactDigestRecordRoundTrip`: real `manifest.Decode` → valid
  Project → `ReadSnapshot(manifest.Codec{}, ...)` → `Digests()` → `NewRecord` →
  `EncodeRecord` → `DecodeRecord`. Separate cases include
  `context/api#legacy.md`, `context/api?legacy.md`, percent spelling, spaces,
  composed/decomposed Unicode, escaped tab/control and literal U+FFFD. A policy
  document also crosses the composition. Checks exact names, independently
  calculated SHA-256 content digests, Project equivalence, complete local metadata,
  portable revision and stable repeated encoding.
- `TestInvalidPortableDocumentNamesNeverProduceDigests`: invalid references fail
  domain construction, real manifest decoding and snapshot construction; no digests
  escape. This proves rejection remains at the portable boundary.
- `TestArtifactNamesKeepSnapshotBoundaryProtections`: rejects invalid extra
  documents in T02 and matching invalid/duplicate local digests through constructor
  and decoder, including dot/empty path components and reserved `axiom.yaml`.
- `TestArtifactNameUnicodeWirePreservation`: valid literal/escaped U+FFFD and
  surrogate pairs preserve metadata; malformed pairs/escapes and invalid UTF-8 fail;
  literal backslash-u text in local paths remains literal data.
- Existing path/identity, payload rejection, duplicate/closed-schema, observation,
  missing/corrupt, H12 exact-byte revision and obsolete-field regressions retained.

### Commands and results

Platform: macOS 26.6.2 arm64; Go 1.26.1 darwin/arm64.
All Go commands used `export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`.
No dependency or tool installation.

| Command / check | Exit | Result |
|---|---|---|
| `git fetch --all --prune`; branch/status/HEAD/main and PR/reviews inspection | 0 | Correct clean branch, current main ancestry and latest human review verified |
| `go test ./internal/local -run 'TestPortableArtifactDigestRecordRoundTrip\|TestInvalidPortableDocumentNamesNeverProduceDigests\|TestArtifactNamesKeepSnapshotBoundaryProtections'` before production edit | 1 | Expected red for `#`, `?`, tab and U+FFFD at `invalid_artifact`. Initial extra raw DEL fixture failed YAML decoding and was removed from positive cases; it was not evidence of a valid producer value. |
| `go test ./internal/local` after edits | 0 | All local tests passed |
| `go test ./...` | 0 | All four packages passed |
| `go test -cover ./...` | 0 | local 95.5%; manifest 96.0%; project 99.6%; projectapp 100.0% |
| `go test -race ./...` | 0 | All four packages passed |
| `go vet ./...` | 0 | Passed |
| `go build ./...` | 0 | Package build passed; no CLI |
| `go mod verify` | 0 | All modules verified |
| `go run ./scripts/check-project-domain.go` | 0 | Seven domain source/test files passed |
| `bash scripts/test-check-project-domain.sh` | 0 | Pure fixture accepted; seven forbidden fixtures rejected |
| `go run ./scripts/check-projectapp.go` | 0 | Seven application source/test files passed |
| `bash scripts/test-check-projectapp.sh` | 0 | Inward fixture accepted; eleven forbidden fixtures rejected |
| `go test -fuzz=FuzzRecordRoundTrip -fuzztime=20s -parallel=2 ./internal/local` | 0 | 505,849 executions; no failure |
| `./scripts/validate-repository.sh .` | 0 | Harness/package and Bash suites, sensitive-file and whitespace checks passed |
| `./scripts/check-sensitive-files.sh .` | 0 | Worktree passed |
| `git diff --check` | 0 | Passed |
| Individual `bash -n` for `scripts/*.sh` | 0 | Passed |
| `git diff --cached --name-status`; `git diff --cached`; `git diff --cached --check`; `./scripts/check-sensitive-files.sh --staged .` | 0 | Seven scoped paths/content reviewed; staged whitespace and sensitive-file checks passed |
| Temporary Python history/contract/document check | 0 | Prior Evidence preserved verbatim; T02 predicate body identical; domain, portable codec, other application contracts, approved artifacts, ADRs and dependencies unchanged; seven local documentation links and fences checked |

### Scope, limits and review recommendation

Reviewed producer/consumer code, new public-boundary regressions and incremental
diff against the previous PR HEAD and current main. README and `docs/commands.md`
already document unchanged run/test/stack commands. Specification, Clarifications,
Plan, Tasks, ADRs and dependencies need no update for this compatibility fix.

No remaining blocking finding identified in scoped self-review. **Ready for human
re-review** on PR #9; no self-approval or merge. H12 remains intact. T05–T21,
filesystem persistence, root discovery, symlink/confinement/TOCTOU, physical CAS,
atomic writes, CLI, migration and Runtime/Provider/network execution are excluded.

Linux and physical filesystem behavior remain unverified. Finite fuzzing/coverage
cannot prove correctness or absence of secrets. Tool lookup (`shutil.which`)
confirmed `gitleaks`, `markdownlint`, `markdownlint-cli2`, `lychee` and `shellcheck`
unavailable; dedicated scanner/linter coverage is not claimed. All mandatory
commands ran; no required validation was skipped.

## H12 / Option B implementation — 2026-09-16

**Human revision decision resolved. T04: Ready for human re-review, not Accepted.
T05–T21: Not started. No merge.**

Authority: explicit human instruction to adopt **Option B — single revision model**,
recorded as [H12](clarifications.md#single-local-revision-decision--2026-09-16).
It authorizes removal of persisted `localRevision` from local v1 and reconciliation
of Specification/Clarifications/Plan/Tasks/Evidence, while preserving T02 and
prohibiting T05. This entry supersedes the pending decision and dual-revision
interpretation in the dated sections below; their complete text is retained as
historical Evidence, including the earlier alternatives and test results.

Baseline: clean branch `codex/t04-local-installation-record-codec`, local/remote/PR
head `05622fb022d27233ab4a7fd11927d164ab917d23`, verified after
`git fetch --all --prune`. Main comparison baseline:
`1de3b02d97818138c32f6b2a07555cfe1ffd4a9b`. Only T04 code and affected
contract/status/Evidence documents change in this response.

**H12 implementation SHA: `a8a8ea28fdc373819d42ad1f0a2fbbec42a98573`.**
This subsequent documentation commit records final hygiene results and that SHA.

### Current revision contract and delivered changes

- Removed `RecordState.LocalRevision`, DTO `localRevision`, both mapping directions,
  its validation and both fixture members. Local v1 accepts records without it and
  rejects the removed member as `unknown_field` through the existing closed schema.
  No replacement label, migration, compatibility fallback or overwrite was added.
- `DecodeObservedRecord` still returns T02's `projectapp.LocalRevision` from
  `ObserveLocalRevision(input)`: SHA-256 of exact observed bytes, including JSON
  whitespace. It hashes no reconstructed/canonical form, excludes no field and
  stores no hash inside its own input. Production implementation of that function
  and all `internal/projectapp` source/tests remain unchanged.
- `portableRevision` remains required independent metadata for the validated
  portable snapshot. It survives local binding changes and round trips; it is not
  the revision of `installation.json`.
- Missing record still returns explicit `MissingLocalRevision`; invalid existing
  records, including those with the removed member, return zero unusable Record
  and invalid zero revision. Input bytes are preserved; no reusable binding escapes.
- Existing path/identity corrections remain: arbitrary valid local path text is
  metadata only; domain UUID/slug rules remain shared and codec issues use local
  schema paths. Portable codec/domain, dependencies and ADR-0004 are unchanged.
- Specification local-state contract, Plan §1/§3/§6 and Tasks T04/T07 now document
  H12. T07 text clarifies future obligations only; no T07 implementation or new
  authority. A future store must compare externally observed exact bytes under
  protected commit semantics and preserve bytes on no-op, without allocating a
  persisted revision. ADR-0004 already delegates internal record details to Plan,
  so no architectural amendment is necessary.

### Regression Evidence

| Test / change | Observed behavior / traceability |
|---|---|
| `TestLocalRevisionDerivedOnlyFromObservedBytes` | Valid v1 without localRevision decodes; whitespace-only differences preserve all metadata but change local revision; encode omits the field; encoded bytes yield their own exact observation; changed checkout path changes local revision while preserving portableRevision. H12, T04, FR-014/016, AC-03/12 |
| `TestPersistedLocalRevisionRejectedWithoutReuse` | Old field fails as unknown with safe stable diagnostics, no usable Record/revision/output and unchanged input. H12, T04, AC-06/07/12 |
| `TestLocalReaderFailureNeverReusesBinding` | Controlled T02 LocalReader integration now also rejects old-field record before binding reuse. No actual filesystem reader/store claimed |
| `TestRecordDTOClosedMetadataInventory` | Closed v1 inventory has no localRevision member; independent portableRevision retained |
| Updated fixtures/security/resource tests | Minimal/configured fixtures omit removed field; Unicode and oversized-output tests now exercise sourceLocation; reference tests retain all remaining reference slots; missing/corrupt and previous path/identity regressions still pass |

### Commands and results for H12

macOS 26.6.2 (25G83), Darwin 25.6.0 arm64; Go 1.26.1 darwin/arm64.
All Go commands used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`.
No dependency or tool installation.

| Command / check | Exit | Result |
|---|---|---|
| `go test ./internal/local -run 'TestLocalRevisionDerivedOnlyFromObservedBytes\|TestPersistedLocalRevisionRejectedWithoutReuse\|TestRecordDTOClosedMetadataInventory'` before production edit | 1 | Expected red: old DTO inventory, missing-field requirement and reusable obsolete-field record |
| `go test ./internal/local` after production edit | 0 | Passed |
| `go test ./...` | 0 | All four packages passed |
| `go test -cover ./...` | 0 | local 95.2%; manifest 96.0%; project 99.6%; projectapp 100.0% |
| `go test -race ./...` | 0 | All four packages passed |
| `go vet ./...` | 0 | Passed |
| `go build ./...` | 0 | Package build passed; no CLI |
| `go mod verify` | 0 | All modules verified |
| `go test -fuzz=FuzzRecordRoundTrip -fuzztime=20s -parallel=2 ./internal/local` | 0 | 441,381 executions, no failure |
| `go run ./scripts/check-project-domain.go` | 0 | Seven domain source/test files passed |
| `bash scripts/test-check-project-domain.sh` | 0 | Pure fixture accepted; seven forbidden fixtures rejected |
| `go run ./scripts/check-projectapp.go` | 0 | Seven application source/test files passed |
| `bash scripts/test-check-projectapp.sh` | 0 | Inward fixture accepted; eleven forbidden fixtures rejected |
| `./scripts/validate-repository.sh .` | 0 | Harness/package validation, both Bash regression suites, worktree sensitive-file and whitespace checks passed |
| `./scripts/check-sensitive-files.sh .` | 0 | Worktree passed |
| `./scripts/check-sensitive-files.sh --staged .` | 0 | Staged content passed |
| `git diff --check`; `git diff --cached --check` | 0 | Passed |
| Individual `bash -n` for `scripts/*.sh` | 0 | Passed |
| Temporary Python document/boundary check | 0 | 107 local link targets, H12 anchors and fences checked; all prior T04 Evidence and clarification history preserved; T01/T02/T03, ADRs and dependency files unchanged from baseline |

Reviewed staged paths/content and scoped diff against the previous PR head and
`main`. `docs/commands.md` already contains the required commands; run/test/stack
instructions need no operational change. README records the resolved H12 gate.

### Limits and review recommendation

H12 removes the revision decision blocker; no remaining blocking finding identified
in this scoped self-review. Recommend **Ready for human re-review** on PR #9,
not Accepted or merge. Approval of Option B does not approve the implementation.
The proposed local v1 shape deliberately rejects earlier PR records with
`localRevision`; no deployed compatibility or migration is claimed.

Filesystem persistence, native roots, canonical resolution, symlink/confinement,
TOCTOU, ownership/ACLs, atomicity, CAS enforcement and Linux execution remain
unimplemented/unverified. Existing T02 content-revision semantics (including
byte-equal ABA limits) are unchanged. Finite fuzzing and coverage prove neither
correctness nor secret absence. Tool lookup (`shutil.which`) found `gitleaks`,
`markdownlint`, `markdownlint-cli2`, `lychee` and `shellcheck` unavailable;
dedicated scanning/lint coverage remains unverified. No T05–T21 responsibilities,
new ports, frameworks or dependencies were introduced.

## Human review response — 2026-09-16

**T04: Blocked on human decision. Not Accepted. T05–T21: Not started.**

Authority: [latest human review on PR #9](https://github.com/rgomids/axiom/pull/9#pullrequestreview-5225789699),
submitted by `rgomids` at `2026-09-16T17:03:41Z` against
`2e8b61c5ae22ca79156fec1cde4b8797a29dab54`, plus the explicit request to address
its independent path/diagnostic findings and investigate revision semantics.
`git fetch --all --prune` completed before editing; clean current branch and PR
head both matched that SHA on `codex/t04-local-installation-record-codec`.
Comparison baseline remains `origin/main` at
`1de3b02d97818138c32f6b2a07555cfe1ffd4a9b`.

This response supersedes the initial path policy and initial “no remaining blocking
finding” conclusion below. All sections starting at “Authority, baseline and
delivery” preserve the original delivery record, including its unapproved dual
revision interpretation; those statements are historical, not new authority.
No approved Specification, Clarifications, Plan, Tasks, ADR, T02 contract or
revision wire field was changed in this response.

### Findings and implemented corrections

- **Major — local path validation:** replace `absoluteLocation` with metadata-only
  `localLocation`: nonempty valid UTF-8 text without control characters or U+FFFD.
  Required `sourceLocation` remains nonempty; empty Repository/Runtime paths still
  represent unresolved bindings. Preserve `#`, `$`, `?`, backticks, backslashes,
  quotes, Unicode, spaces, relative spelling, repeated separators and dot components
  literally. No trim, interpolation, `path.Clean`, absolute-path requirement or
  physical lookup applies to these three local slots. Plan §3's canonical source
  remains a future adapter observation obligation; successful codec validation
  cannot establish canonicality or qualify metadata for filesystem use.
  Portable artifact-name and credential-reference policies are unchanged.
- **Minor — identity diagnostics:** continue calling `project.ValidateIdentity`
  as the only UUID/slug rule implementation. Translate its first issue's field:
  `project.id` → `installation.projectId`, `project.slug` →
  `installation.observedSlug`. Preserve domain code, error severity, first-error
  ordering and local-state category/remedy. No domain rule or T02 API changed.
- **Major — revision ambiguity:** investigated below; neither option implemented.
  Keeping the existing field while awaiting review is not approval of its semantics.

New public-boundary regressions in `internal/local/review_test.go`:

| Test | Evidence / traceability |
|---|---|
| `TestLocalPathsPreserveArbitraryMetadata` | NewRecord → encode → observed decode preserves all three path slots, complete metadata and exact observed input revision; T04, FR-014, AC-03/12, Plan §3 |
| `TestLocalPathTextRejectedAtEachBoundary` | Each slot rejects controls (including C1), invalid UTF-8, replacement characters and unpaired surrogates; no reusable Record/output; source required versus unresolved optional paths retained; T04, FR-016, AC-06/12 |
| `TestIdentityDiagnosticsUseLocalSchemaWithDomainRules` | Construction and wire decode compare acceptance and first error directly with domain validation for valid UUIDs/slugs, UUID case/version/variant, missing values, invalid slug forms and simultaneous failures; only schema paths change; T04, FR-016, AC-12 |

Existing negative tests in `record_test.go` and `structure_test.go` now use empty
required source and NUL Runtime path instead of rejecting relative spelling.
Tests exercise codec behavior and domain integration in memory; no CLI or real
filesystem flow exists or is claimed.

### HUMAN DECISION REQUIRED

Investigation found insufficient authority to establish two deliberate revision
concepts, or to remove a Plan-listed field without human approval:

- Specification “Portable configuration and local state” requires recording the
  validated **portable** revision/content. FR-013/018 and SEC-004 require concurrency
  protection; they do not define a second persisted local label.
- Plan §1 supplies an expected-record-revision store contract. Plan §3 lists
  “local revision” among stored metadata but defines no independent label semantics,
  producer, lifecycle or relation to CAS. Plan §6 requires expected local revision
  for reconciliation. The original Plan commit `061846a` already contained this
  same ambiguity; it did not establish two revision models.
- T02 (`5e18a48`, merged via PR #7) concretely defines `ObserveLocalRevision(record)`
  as SHA-256 of exact observed record bytes in `internal/projectapp/snapshot.go`.
  `LocalReader`, `PreviewLocal` and `ExpectedRevisions.Local` propagate this separate
  observation. `LocalState` contains no second revision label. T02 Evidence's
  “Exact revisions” section confirms this contract.
- Tasks T02 owns expected revisions; T04 repeats Plan §3 “local revision”; T07 owns
  future local CAS and forbids revision/timestamp rewrites on equivalent no-op.
  None establishes independent semantics for another token.
- Clarifications Q3/P3/H1/H8/H9/H10 and ADR-0004 establish versioned machine-local
  state, ID continuity and authority/concurrency, not a second revision label.
  ADR-0004 leaves internal layout/version details to Plan. Later lifecycle approvals
  do not resolve the ambiguity. T04's implementation/Evidence interpretation is
  evidence of what was written, not a human decision authorizing that interpretation.

Putting the exact-byte revision inside the same record would require the embedded
value to equal the hash of bytes that include that value. Writing the computed hash
changes those bytes again; ordinary hashing/encoding cannot provide that fixed-point
contract. Excluding the field or storing a predecessor hash would be a different
contract, not T02's exact-byte observation. The existing opaque string avoids that
cycle only by introducing a second concept whose purpose is unspecified.

#### Option A — dual revision model

Keep exact-byte CAS `projectapp.LocalRevision`; rename the persisted value to
**`reconciliationLabel`**. Proposed semantics, subject to human approval: an opaque
identifier naming the authorized local reconciliation that produced the record,
with diagnostic/correlation value only. It is neither content digest, monotonic
counter, freshness/authenticity proof nor CAS precondition. A future store/use-case
would supply it in the complete proposal before preview/approval, retain it on
no-op, and preserve/report the actually committed value after failure. Its overlap
with existing `Attempt.Correlation` needs a demonstrated independent consumer.
No allocation or persistence protocol is selected here.

Exact reconciliation set if approved:

1. `spec.md`: local metadata meaning and relation to concurrency; dated authority.
2. `clarifications.md`: new dated human decision preserving Q1–Q6/H1–H11 history.
3. `plan.md` §1/§3/§6: two distinct responsibilities, wire name, producer, proposal
   authority, no-op and post-commit semantics; exact-byte CAS remains external.
4. `tasks.md` T02/T04/T07: metadata versus CAS ownership and required Evidence.
5. `docs/decisions/0004-portable-project-manifest.md`: dated boundary clarification,
   keeping detailed internal wire layout in Plan and preserving accepted history.
6. `internal/projectapp/ports.go` (`LocalState`) and contract/authority tests: carry
   the label in the complete approved proposal so a writer cannot add an unreviewed
   value. `LocalRevision`, `ExpectedRevisions` and hashing semantics stay unchanged.
7. `internal/local/record.go`, `dto.go`, `mapping.go`, `validation.go`, both JSON
   fixtures, affected `record_test.go`, `structure_test.go`, `boundary_test.go` and
   `integration_test.go`: rename, validate, map, round-trip and protect the new field.
8. `evidence-t02.md` (dated addition), `evidence-t04.md`, `CHANGELOG.md`: record the
   approved change, validation and compatibility implications. README/index/roadmap
   lifecycle entries change only when the actual review gate changes.

Cost: second lifecycle and authority-bearing proposal field, possible duplication
of attempt metadata, and another v1 compatibility commitment. Benefit exists only
if an independent reconciliation consumer needs that durable label.

#### Option B — single revision model

Remove persisted `localRevision`; retain `projectapp.LocalRevision` as the sole
**local-record** revision contract. `portableRevision` remains distinct content
metadata describing the portable snapshot; it is unaffected.

Consequences and exact reconciliation set if approved:

1. `clarifications.md`: dated human resolution; `spec.md` local-state section can
   clarify external observed local revision without changing portable-revision
   requirements. Preserve earlier approvals.
2. `plan.md` §3: remove local revision from stored fields and explicitly describe
   deriving it from exact read bytes. §1/§6 retain expected-revision semantics,
   cross-referencing that clarification.
3. `tasks.md` T04: distinguish codec metadata from returned observed revision;
   T07: derive/recheck exact bytes, with no stored label to allocate or increment.
4. `internal/local/record.go` (`RecordState`), `dto.go`, `mapping.go`, `validation.go`:
   remove the persisted member and its validation/mapping. Keep
   `DecodeObservedRecord`/missing/invalid behavior unchanged.
5. Both JSON fixtures and affected `record_test.go`, `structure_test.go`,
   `boundary_test.go`, `integration_test.go`: update closed inventory and assumptions;
   add explicit unknown-field rejection for obsolete `localRevision`. Retain exact
   observed-byte revision assertions. Fuzz seeds follow updated fixtures.
6. `evidence-t04.md` and `CHANGELOG.md`: record approved reconciliation and rerun
   Evidence; lifecycle references change only when the gate changes. T02 contracts
   and historical Evidence remain valid. ADR-0004 needs no semantic change: it does
   not specify a stored revision field; verify its Plan references remain coherent.

Future store reads exact bytes, validates metadata, passes the external observation
into preview/CAS, and rechecks under its eventual protected commit protocol. It
hashes newly encoded bytes to observe the resulting revision; no self-reference or
label allocation. Equivalent no-op preserves bytes. Byte-equal ABA remains a limit
of the existing T02 content-observation contract; a new history-sensitive guarantee
would need separate authority. No filesystem CAS is implemented here.

Compatibility: this changes the proposed closed v1 format, so existing PR fixtures
with the removed field would fail, not migrate or be overwritten. Human approval
must settle that before freezing v1. No deployed store or migration is claimed.

**Recommendation: Option B.** It matches the already accepted T02 consumer
contract, removes an unused second concept and avoids self-reference. Plan/Tasks
still require explicit reconciliation; technical preference is not authority.
PR remains **Blocked on human decision**, open for review, without merge or T05.

### Review-response commands and results

Platform: macOS 26.6.2 (25G83), Darwin 25.6.0 arm64; Go 1.26.1 darwin/arm64.
All Go checks used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; no dependencies
or tools installed. These results concern this response, not inferred old Evidence.

| Command / check | Exit | Result |
|---|---|---|
| `go test ./internal/local -run 'TestLocalPathsPreserveArbitraryMetadata\|TestLocalPathTextRejectedAtEachBoundary\|TestIdentityDiagnosticsUseLocalSchemaWithDomainRules'` before production edit | 1 | Expected red: valid paths rejected and identity fields used portable namespace |
| `go test ./internal/local` after correction | 0 | Passed |
| `go test ./...` | 0 | Four packages passed |
| `go test -cover ./...` | 0 | local 95.3%; manifest 96.0%; project 99.6%; projectapp 100.0% |
| `go test -race ./...` | 0 | Four packages passed |
| `go vet ./...` | 0 | Passed |
| `go build ./...` | 0 | Package build passed; no CLI |
| `go mod verify` | 0 | All modules verified |
| `go test -fuzz=FuzzRecordRoundTrip -fuzztime=20s -parallel=2 ./internal/local` | 0 | 127,379 executions; no failure |
| `go run ./scripts/check-project-domain.go` | 0 | Seven domain source/test files passed |
| `bash scripts/test-check-project-domain.sh` | 0 | Pure fixture accepted, seven forbidden fixtures rejected |
| `go run ./scripts/check-projectapp.go` | 0 | Seven application source/test files passed |
| `bash scripts/test-check-projectapp.sh` | 0 | Inward fixture accepted, eleven forbidden fixtures rejected |
| `./scripts/validate-repository.sh .` | 0 | Harness/package validation, both Bash regression suites, sensitive-file and whitespace checks passed |
| `./scripts/check-sensitive-files.sh .` | 0 | Worktree passed |
| `./scripts/check-sensitive-files.sh --staged .` | 0 | Staged content passed |
| `git diff --check`; `git diff --cached --check` | 0 | Passed |
| Individual `bash -n` for `scripts/*.sh` | 0 | All root shell scripts passed syntax checks |
| Temporary Python documentation/boundary check | 0 | 27 local link targets and fences checked; original Evidence suffix preserved byte-for-byte; Specification/Clarifications/Plan/Tasks/ADR-0004, project/projectapp/manifest and dependency files unchanged from review head |

Diff/security review: seven changed files, production changes confined to
`internal/local/validation.go`; regressions plus Evidence, README gate and changelog.
No filesystem API, new port, dependency or T05+ responsibility introduced.
`docs/commands.md` already documents the executed validation commands; stack,
run/test instructions and directory layout remain current. Optional scanners were
looked up with Python `shutil.which`; all five named below were absent.

Remaining limits: no proof of filesystem existence, canonicality, confinement,
symlinks/TOCTOU, ownership, permissions/ACLs, atomic writes or CAS enforcement.
Linux execution remains unverified; tests run on macOS. Finite fuzzing/coverage
cannot prove correctness or absence of secrets. `gitleaks`, `markdownlint`,
`markdownlint-cli2`, `lychee` and `shellcheck` are unavailable; their dedicated
coverage is unverified. Existing reference validation, closed schema and no-effects
boundary remain intact; arbitrary valid path text is not a secret-scanning guarantee.

## Authority, baseline and delivery — 2026-09-16

- Explicit human authorization covers **T04 only**. Source authority:
  [Specification](spec.md), [Clarifications](clarifications.md), [Plan](plan.md),
  [Tasks](tasks.md), [ADR-0004](../../decisions/0004-portable-project-manifest.md)
  and its referenced ADRs, Constitution and repository policies.
- Initial working tree was clean on `main`, locally at
  `378338cc0af79eaaec8b17e3cada44629515c043`. `git fetch origin` found only the
  expected advancement. `git merge --ff-only origin/main` established exact
  `main` / `origin/main` / HEAD baseline
  **`1de3b02d97818138c32f6b2a07555cfe1ffd4a9b`**; tree remained clean.
- GitHub confirms [PR #8](https://github.com/rgomids/axiom/pull/8), T03, merged
  at that SHA on `2026-09-16T15:00:21Z`. T01/T02/T03 production contracts and
  tests were inspected before adding local types.
- Branch: `codex/t04-local-installation-record-codec`.
- **Implementation SHA: `95f1a552b1659d898686977918caeb003ad2effd`.**
  The subsequent documentation commit records this SHA without a circular hash.
- **T01: Accepted / merged. T02: Accepted / merged. T03: Accepted / merged.
  T04: Ready for human implementation review. T05–T21: Not started.**
  No self-approval or merge. T04 merge grants no T05 authority.

## Implemented contract and decisions

[`internal/local/`](../../../internal/local/) implements an in-memory codec:

```text
existing JSON bytes -> bounded tokens -> exact version -> closed DTO shape
-> T02 metadata mapping -> domain identity and local metadata validation
-> sealed Record -> closed DTO -> JSON -> checked decode
```

- Standard-library `encoding/json`; no new dependency. Token inspection precedes
  struct decoding because ordinary JSON struct unmarshalling accepts duplicate
  and case-insensitive fields and null scalar values. Fixed DTO reflection checks
  exact field names, requiredness and types; it is private codec machinery, not
  a persistence abstraction or extensible schema framework.
- Exactly integer token `1` is supported. Missing/null/bool/string/fractional,
  decimal `1.0`, exponent `1e0` and malformed versions are invalid local state;
  other integer tokens, including integers larger than the machine integer range,
  are unsupported local format. `schemaVersion` cannot substitute for the version.
- Unknown and duplicate fields fail at every object depth. Escaped key aliases
  count as duplicates. Reject malformed/trailing JSON, nulls, invalid types and
  invalid UTF-8. Replacement character U+FFFD is rejected in metadata strings,
  including replacement generated by JSON decoding of unpaired surrogates;
  no silent Unicode repair is accepted. All errors return the zero Record.
- `RecordState` reuses `projectapp.ArtifactDigest`, `PortableRevision`,
  `RepositoryBinding`, `CredentialBinding`, `RuntimeBinding`, `Observation` and
  `AttemptMetadata`. `Record.State()` detaches slices. No new application port
  or domain model for local persistence was introduced.
- `project.ValidateIdentity` exposes the existing UUID v4/variant/canonical-case
  and slug rules. Portable construction delegates to the same function; tests
  compare results. Rules stay in the domain rather than being copied into JSON.
- A record stores a source location as metadata. It never serializes or constructs
  a `Destination` capability. Source/path syntax checks do not establish physical
  canonicality, containment, permissions, existence or authority.
- The Plan's local revision is represented as a nonempty opaque reference token
  supplied by a future store. No allocation, increment or persistence protocol is
  chosen. It is distinct from T02's existing **exact-byte CAS revision**:
  `DecodeObservedRecord` returns `ObserveLocalRevision(input)` for valid existing
  bytes. The stored label cannot replace that CAS observation or confer authority.
- `portableRevision` restores T02's recorded aggregate content digest. Individual
  digests use T02 `ArtifactDigest`; `axiom.yaml` is required and names are unique.
  Hex encodes the existing 32-byte digests. The codec neither recomputes T02's
  length-framed content revision nor validates digest correspondence without bytes.
  T02 continues to compute complete revisions in stable document path order.
- Encoder emits explicit DTO fields with a final newline; preserves collection
  order and all metadata values semantically. No extra keyed sorting, revision
  allocation or manifest canonicalization policy was added. Checked decode must
  succeed before any bytes are returned. Oversized output returns no bytes.

### Closed local wire inventory

Only Plan §3 metadata is represented. Exact JSON spellings are local adapter
implementation details, not additions to the portable schema.

| Fields | Wire and consumer contract |
|---|---|
| `formatVersion` | Required exact integer `1` |
| `projectId`, `observedSlug` | Required domain-valid UUID and observed slug; no portable friendly name |
| `sourceLocation` | Required lexical absolute clean POSIX path for the Linux/macOS slice; never a capability |
| `portableRevision`, `localRevision` | Recorded 32-byte portable digest in hex; opaque local revision label |
| `artifactDigests` | Required array of closed `{name, digest}` metadata; manifest plus referenced-document digests, no bytes/text bodies |
| `repositories` | Required array of T02 repository-key, explicit-path, canonical-identity and observation metadata |
| `credentials` | Required array of `{referenceKey, sourceKind, itemReference}`; empty source/item represents unresolved binding, never a secret value |
| `runtime` | Optional closed `{runtimeId, explicitPath, observation}` matching T02 metadata |
| `attempt` | Optional closed `{correlation, at}` matching T02 metadata |
| nested `observation` | Closed `{availability, basis, observedAt}`; exactly T02 enum names and RFC3339Nano time metadata |

Required arrays can be empty. Local metadata has no portable absent/unconfigured
union; zero Runtime/Attempt metadata maps to omission. Portable declaration-state
semantics remain in T01/T03 unchanged. Empty paths/reference metadata for unresolved
bindings are retained, as are unknown source kinds: naming a source does not select,
resolve or authenticate it. Observation codes and times are validated as metadata;
checking their factual basis and current applicability remains T13/T16/use-case work.

Reference fields reject whitespace/control characters, payload delimiters,
interpolation, rooted/traversal references, URL forms, and known sensitive
`name:value` families (token/password/API-key/signature and related families already
used by T03). Nested percent escapes are inspected under a finite budget without
rewriting identifiers. This is structural validation, not heuristic secret scanning.
Local path fields permit machine paths as required by the Plan; logical reference
fields do not become arbitrary key/value configuration bags.

### Missing versus invalid and diagnostics

`DecodeRecord` always means an existing record, even for nil or empty bytes.
`DecodeObservedRecord(bytes, exists)` is an explicit in-memory read seam:

| Observation | Record / revision / issues |
|---|---|
| Absent, no bytes | Zero Record, `MissingLocalRevision`, no issues |
| Absent with bytes | Invalid observation; zero Record/revision, error |
| Existing valid bytes | Valid Record, exact T02 observed revision, no issues |
| Existing malformed/unversioned/unsupported bytes | Zero Record, invalid zero revision (not missing), safe error |

The test-only `LocalReader` fake bridges valid metadata to T02 `LocalSnapshot`.
Failure never reaches its reuse branch; input bytes remain unchanged. This proves
codec/consumer integration, not operating-system read-error classification or an
implemented reopen/install use case. A future reader must classify not-found
separately from permission/I/O errors before using the seam.

Issues carry fixed codes, safe schema paths/numeric indices, category, severity
and remedy. First-error ordering follows token order, then fixed DTO order and
metadata validation order; unknown keys are reported at their known parent,
never echoed. Identical input produces identical issues. Unknown integer versions
use `unsupported_local_format`; other failures use `invalid_local_state`.
Remedy is to preserve and inspect local state, never recreate or overwrite it.

### Resource limits

Input limit: **262,144 bytes**, checked before parser allocation. Token-tree limit:
**16 levels**, root at 1; **16,384 nodes**, counting keys and values but not closing
delimiters. Checks run during traversal before descending further. Tests exercise
exact accepted boundaries and rejected excesses. These are tunable adapter limits.
Encoder checks resulting bytes before return; it still allocates DTO/output for
caller-supplied in-memory proposals. No claim of allocation-free or bounded-memory
encoding of arbitrarily large proposals is made.

## Traceability and observed behavior

All rows concern T04's contribution, not completion of the full acceptance criteria.

| Task → requirements / authority | Component / executable Evidence |
|---|---|
| T04 → FR-014, AC-06, H8, Plan §3 | `TestVersionMatrix`, `TestStrictStructures`, `TestEveryNestedShape`, `TestTokenLimitsAtBoundary`: exact version, strict JSON/closed nested structures, domain identity reuse |
| T04 → FR-008, SEC-001, AC-08, Plan §3 | `TestSecurityClosedFieldsAndReferenceValues`, `TestAllReferenceSlotsRejectPayload`, `TestNestedSecretAndBodyFields`, `TestEncodeValidatesProposals`: explicit forbidden fields/payloads rejected, no partial Record or encoded bytes |
| T04 → FR-014, AC-03, H1, ADR-0004 | `TestRecordRoundTrips`, `TestMetadataValidationAndSourceKinds`, `TestObservationAndGapRoundTrips`: identity, revisions, digests, bindings, reference source kinds, observations, attempts and unresolved metadata preserved |
| T04 → FR-014, AC-07, AC-12, Plan §3/§7 | `TestMissingDiffersFromInvalid`, `TestLocalReaderFailureNeverReusesBinding`: absent distinct from corruption; fake consumer reuses no binding on error; exact byte revision retained |
| T04 → FR-016, AC-12, Plan §7 | `rejected` helper repeats decode and compares safe diagnostics for every rejection; sentinel assertions check error content; category/remedy vocabulary fixed |
| T04 → SEC-001, SEC-005, AC-03, AC-08, H1/H8, ADR-0004 | `TestRecordDTOClosedMetadataInventory`, `TestLocalProductionDependencyBoundary`, `TestPortableCodecRejectsLocalFields`: closed metadata types, restricted imports/effects, portable codec rejects local fields and emits none |
| T04 → Plan §9 | Behavior tests, controlled T02 port integration, static source/DTO checks and `FuzzRecordRoundTrip`; existing T01–T03 regressions retained |
| T04 → domain boundary | `TestIdentityValidationSharedByPortableAndLocalMetadata`, domain/application AST checkers and their negative fixture suites |

## Commands and results

Platform: **macOS 26.6.2 (25G83), Darwin 25.6.0, arm64**.
Toolchain: **go1.26.1 darwin/arm64**. No tool or module installed.
All Go commands below used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`.

| Command / check | Exit | Result |
|---|---|---|
| Initial `go test ./internal/local` before production files | 1 | Expected red baseline: no non-test Go files; behavioral tests already written |
| `go test ./...` | 0 | All four packages passed |
| `go test -cover ./...` | 0 | local 94.5%; manifest 96.0%; project 99.6%; projectapp 100.0% statement coverage |
| `go test -race ./...` | 0 | All four packages passed |
| `go vet ./...` | 0 | Passed |
| `go build ./...` | 0 | Passed; package build, no CLI binary |
| `go mod verify` | 0 | All modules verified |
| `go test -fuzz=FuzzRecordRoundTrip -fuzztime=20s -parallel=2 ./internal/local` | 0 | 827,215 executions, no failure; production code matches implementation SHA |
| `go run ./scripts/check-project-domain.go` | 0 | Seven domain source/test files passed pure-symbol boundary |
| `bash scripts/test-check-project-domain.sh` | 0 | Pure fixture accepted; seven forbidden fixtures rejected |
| `go run ./scripts/check-projectapp.go` | 0 | Seven application source/test files passed inward boundary |
| `bash scripts/test-check-projectapp.sh` | 0 | Inward fixture accepted; eleven forbidden fixtures rejected |
| `./scripts/validate-repository.sh .` | 0 | Harness, both Bash suites, sensitive-file scan and whitespace passed |
| Temporary documentation check (Python, repository root) | 0 | 121 relative links/anchors, balanced fences; four normative bodies/history preserved; ADRs, portable codec, application contracts and dependencies unchanged |
| `git diff --check`; `git diff --cached --check` | 0 | Passed |
| `./scripts/check-sensitive-files.sh --staged .` | 0 | Passed before each commit |

Repository checks are hygiene Evidence only. Coverage and finite fuzzing do not
prove correctness or confidentiality. `gitleaks`, `markdownlint`,
`markdownlint-cli2`, `lychee` and `shellcheck` were unavailable (lookup exit 1);
dedicated scanner/linter/external-link coverage remains unverified.

## Self-review and security limits

Reviewed complete change against `main`, Plan §3 field inventory, existing T01–T03
contracts, architecture direction and T04 exclusions. No remaining blocking
finding identified. Only domain identity validation was extracted for reuse;
application contracts, portable codec/schema, dependencies and ADRs are unchanged.
Normative artifact updates add lifecycle/Evidence entries only, retaining history.
No new durable architecture decision, persistence mechanism or migration policy.

Security Evidence is deliberately separated:

- **Codec/unit:** prohibited explicit credential/body fields, unsafe reference
  payload forms, malformed metadata and partial-result reuse are rejected.
  Encoder cannot accept arbitrary payload maps or document bodies through its DTO.
- **Static boundary:** exact DTO field/type inventory, production import/effect
  inspection, unchanged portable production codec and inward dependency checks.
  No filesystem, environment-value, secret-store, process, network or logging API
  is reachable through the inspected local production code. Test runners and
  source-inspection tests naturally perform build/read/report I/O.
- **Not proved:** arbitrary secrets disguised as otherwise valid identifier/path
  strings cannot be recognized universally. No heuristic scanner was implemented.
  Closed shapes and rejection tests do not certify complete absence of secrets.
- **Future filesystem integration:** record writes, owner-only modes/ACLs,
  ownership, native roots/overrides, physical canonical source resolution,
  symlink/TOCTOU confinement, collision/concurrency/CAS enforcement, crash safety,
  logical commit, preservation under real I/O failure and Linux behavior remain
  unimplemented and unproved. Unit/fake integration proves none of these.
- **Future use cases:** comparing recorded UUID/slug/digests and stale bindings
  against a freshly validated portable snapshot; referenced-document completeness;
  factual Runtime/checkout observations; duplicate checkout resolution; no-op,
  relocation authority and install/reopen outcomes remain later tasks.
- **Explicit exclusions:** T05–T21, filesystem writes/persistence, native state
  discovery, migrations, recovery/fallback overwrite, CLI, portable schema changes,
  database/catalog/Workspace, generic repository/persistence framework, DI,
  Runtime/Provider execution, Git/network execution and secret resolution.

No full AC-03/06/07/08/12 or SEC-005 platform acceptance is claimed by this codec
slice. Human implementation review is the next gate; merge is not authorization
to start T05.
